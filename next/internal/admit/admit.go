// Package admit is the admission rule set as a pure library
// (docs/next-build-plan.md Phase 2 item 1; plans/os-3898f232.md): one
// ordered set of rules importable by the cooperative client (2.2), the
// seed-admit pre-receive hook (2.3), and later the forge service, so
// postures differ in where the rules run, never in which rules run.
// Phase 2 carries the rules that exist so far: halt, shape, actor
// signature against the genesis governance root (the Phase 3 keyring
// projection replaces the resolver, not the rule), protocol version
// discipline, and payload classification. Capability rules land in
// Phase 3, fences in Phase 5, budget reservations in Phase 7, each as an
// appended rule; nothing here special-cases them in advance.
package admit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shaunlmason/open-seed/next/internal/classify"
	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/genesis"
	"github.com/shaunlmason/open-seed/next/internal/halt"
	"github.com/shaunlmason/open-seed/next/internal/keyring"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
	"github.com/shaunlmason/open-seed/next/internal/packet"
	"github.com/shaunlmason/open-seed/next/internal/transition"
	"github.com/shaunlmason/open-seed/next/internal/version"
)

// Context is everything a rule may consult, derived by one verified
// replay of the chain. Rules are pure functions of (context, record).
type Context struct {
	Count     int
	Tip       string
	Active    string
	Halt      halt.State
	Resolve   ledger.Resolver
	Keyring   *keyring.State
	Table     *transition.Table
	Lifecycle *transition.Fold
	Supported map[string]bool
	// Records is the verified prefix the context was computed from:
	// the budget rule's position-accurate validity replays need it
	// (plans/os-cecac5de.md D4).
	Records []*event.Record
}

// Option configures context construction.
type Option func(*options)

type options struct{ supported []string }

// WithSupportedVersions declares the protocol versions this admission
// point accepts, mirroring ledger.WithSupportedVersions. The default is
// the build's own protocol version.
func WithSupportedVersions(vs ...string) Option {
	return func(o *options) { o.supported = vs }
}

// ContextAt builds the admission context with a single VerifyFromGenesis
// replay (records observed via ledger.WithObserver feed the halt
// projection). A chain that does not verify yields no context: admission
// never grows an invalid chain.
func ContextAt(store *ledger.Store, opts ...Option) (*Context, error) {
	var o options
	for _, f := range opts {
		f(&o)
	}
	resolve, _, err := genesis.Bootstrap(store)
	if err != nil {
		return nil, fmt.Errorf("admission context: %w", err)
	}
	supported := map[string]bool{}
	for _, v := range version.Supported() {
		supported[v] = true
	}
	var vopts []ledger.VerifyOption
	if len(o.supported) > 0 {
		vopts = append(vopts, ledger.WithSupportedVersions(o.supported...))
		supported = map[string]bool{}
		for _, v := range o.supported {
			supported[v] = true
		}
	}
	var records []*event.Record
	vopts = append(vopts, ledger.WithObserver(func(pos int, rec *event.Record) {
		records = append(records, rec)
	}))
	rep, err := store.VerifyFromGenesis(resolve, vopts...)
	if err != nil {
		return nil, err
	}
	ring, _, err := keyring.StateAt(records)
	if err != nil {
		// A verified chain cannot fail the keyring projection: the
		// replay above already applied the same transitions.
		return nil, err
	}
	if keyring.Applies(rep.ActiveVersion) && ring.Seeded() {
		// From seed/1 the keyring is the resolver: standing decides who
		// signs at the tip (the Phase 3 projection replacing the genesis
		// resolver, exactly as the package doc promised).
		resolve = ring.Resolver()
	}
	table, err := transition.Default()
	if err != nil {
		// The embedded table failing self-validation is a build
		// defect, not a chain condition; admission refuses outright.
		return nil, fmt.Errorf("admission context: %w", err)
	}
	return &Context{
		Count:     rep.Count,
		Tip:       rep.Tip,
		Active:    rep.ActiveVersion,
		Halt:      halt.StateAt(records),
		Resolve:   resolve,
		Keyring:   ring,
		Table:     table,
		Lifecycle: table.FoldRecords(records),
		Supported: supported,
		Records:   records,
	}, nil
}

// Refusal is an admission refusal naming the rule that refused. It
// unwraps to the rule's own typed error, so the envelope layer keeps the
// established exit mapping (halted 7, chain 8, classification 9,
// version 10).
type Refusal struct {
	Rule string
	Err  error
}

func (r *Refusal) Error() string {
	return fmt.Sprintf("admission refused by rule %s: %v", r.Rule, r.Err)
}

func (r *Refusal) Unwrap() error { return r.Err }

// ClassificationError carries the lint violations for a refused payload;
// its message matches the CLI's exit-9 rendering.
type ClassificationError struct{ Violations []classify.Violation }

func (e *ClassificationError) Error() string {
	parts := make([]string, 0, len(e.Violations))
	for _, v := range e.Violations {
		parts = append(parts, fmt.Sprintf("%s: %s", v.Pointer, v.Rule))
	}
	return "payload fails data classification: " + strings.Join(parts, "; ")
}

// OutOfGrantError is the capability refusal the charter names
// (SEED-NEXT.md Part II "Capabilities"): the actor holds none of the
// capabilities the verb accepts. Grants are events (actor.granted),
// checked at admission on every verb against the vocabulary in
// next/spec/actors.md; governance roots hold operator implicitly. It
// maps to exit 14 out_of_grant (next/spec/envelope.md).
type OutOfGrantError struct {
	Actor    string
	Verb     string
	Accepted []string
}

func (e *OutOfGrantError) Error() string {
	return fmt.Sprintf("actor %s is not granted any of [%s], which %s accepts — grants are capability data checked at admission (plans/os-3979d48b.md)", e.Actor, strings.Join(e.Accepted, ", "), e.Verb)
}

// FenceError is the stale-or-missing fence refusal (exit 6
// fenced_out, the v1-continuity row): on a subject with an active
// claim, the event either had to cite the fence and did not, cited a
// fence that is not the active one, or cited a fence when no claim is
// active. Cited is empty for a missing citation; Active is -1 when no
// claim is active (plans/os-5dc16a7c.md).
type FenceError struct {
	Subject string
	Cited   string
	Active  int
	Holder  string
}

func (e *FenceError) Error() string {
	if e.Active < 0 {
		return fmt.Sprintf("event on %s cites fence %s but no claim is active — a fence dies with its claim window", e.Subject, e.Cited)
	}
	if e.Cited == "" {
		return fmt.Sprintf("event on %s must cite the active fence %d held by %s — claim-scoped events carry {\"fence\": \"<position>\"}", e.Subject, e.Active, e.Holder)
	}
	return fmt.Sprintf("event on %s cites stale fence %s; the active fence is %d, held by %s", e.Subject, e.Cited, e.Active, e.Holder)
}

// ContentionError is the exclusivity refusal (exit 2 contention): the
// subject is already held. The fence is the position the active claim
// was taken at, so the loser learns who holds and since when.
type ContentionError struct {
	Subject string
	Holder  string
	Fence   int
}

func (e *ContentionError) Error() string {
	return fmt.Sprintf("contract %s is already claimed by %s (fence %d, held since position %d) — exclusivity is granted at admission, one claim at a time", e.Subject, e.Holder, e.Fence, e.Fence)
}

// NotIndependentError is the L1 independence refusal (exit 17
// not_independent, next/spec/verdicts.md): the verdict signer is an
// implementing key on this contract — a claimant, past or present, or
// the bound submission's signer. Distinct from OutOfGrantError:
// capability is global, independence is per-contract, and a perfectly
// good verdict grant does not cure being disqualified here.
type NotIndependentError struct {
	Subject string
	Actor   string
	Role    string
}

func (e *NotIndependentError) Error() string {
	return fmt.Sprintf("verdict on %s refused: signer %s is an implementing key on this contract (%s) — L1 independence separates failure domains, and a verdict grant does not cure disqualification on the contract being judged (next/spec/verdicts.md)", e.Subject, e.Actor, e.Role)
}

// VerdictError is the verdict.rendered shape-and-binding refusal: a
// malformed payload, an illegal literal, or a citation of anything but
// the bound submission. It rides the established shape-refusal exit
// mapping, like the milestone rule's.
type VerdictError struct {
	Subject string
	Reason  string
}

func (e *VerdictError) Error() string {
	return fmt.Sprintf("verdict.rendered on %s refused: %s (next/spec/verdicts.md)", e.Subject, e.Reason)
}

// OfferError is the offer.published shape refusal: a malformed
// payload, a malformed expiry, or a born-dead expiry at or before the
// event's own ts. It rides the established shape-refusal exit
// mapping, like the verdict rule's.
type OfferError struct {
	Subject string
	Reason  string
}

func (e *OfferError) Error() string {
	return fmt.Sprintf("offer.published on %s refused: %s (next/spec/offers.md)", e.Subject, e.Reason)
}

// BudgetError is the budget rule's refusal: a malformed payload, a
// reserve outside the holder/operator boundary or beyond remaining,
// a close of a missing, invalid, or already-closed reservation, or a
// spending verb with no open reservation. It rides the established
// shape-refusal exit mapping.
type BudgetError struct {
	Subject string
	Reason  string
}

func (e *BudgetError) Error() string {
	return fmt.Sprintf("budget on %s refused: %s (next/spec/budgets.md)", e.Subject, e.Reason)
}

// RunError is the run rule's refusal: a malformed payload, a start
// outside its claim window or citation boundary, or a settle on a
// start-less or already-settled fence. It rides the established
// shape-refusal exit mapping.
type RunError struct {
	Subject string
	Reason  string
}

func (e *RunError) Error() string {
	return fmt.Sprintf("run on %s refused: %s (next/spec/executors.md)", e.Subject, e.Reason)
}

// VerbInactiveError refuses a verb whose semantics are not active under
// the chain's protocol version: an actor.* draft on a seed/0 tip is a
// verb illegal in this state (exit 3) until the deployment upgrades.
type VerbInactiveError struct {
	Verb   string
	Active string
	Needs  string
}

func (e *VerbInactiveError) Error() string {
	return fmt.Sprintf("verb %s is not active at protocol %s: it activates at %s (append system.protocol.upgraded first)", e.Verb, e.Active, e.Needs)
}

// Rule is one named admission rule.
type Rule struct {
	Name  string
	Check func(*Context, *event.Record) error
}

// Default returns the ordered Phase 2 rule set. The halted refusal
// dominates (a malformed forbidden draft under halt refuses as halted,
// the reviewed halt.Check ordering); later phases append rules to this
// set rather than editing it.
func Default() []Rule {
	return []Rule{
		{Name: "halted", Check: func(c *Context, rec *event.Record) error {
			if c.Halt.Halted && rec.Event.Verb != halt.LiftVerb {
				return &halt.HaltedError{By: c.Halt.By, Reason: c.Halt.Reason}
			}
			return nil
		}},
		{Name: "shape", Check: func(c *Context, rec *event.Record) error {
			if _, err := rec.Event.Hash(); err != nil {
				return err
			}
			if err := halt.ValidateShape(&rec.Event); err != nil {
				return err
			}
			// Admission applies the upgrade schema unconditionally: a
			// signed but schema-broken upgrade admitted to the chain
			// would wedge every later verification at bad_payload.
			return ledger.ValidateUpgradeShape(&rec.Event)
		}},
		{Name: "standing", Check: func(c *Context, rec *event.Record) error {
			// The activation check precedes signer resolution: an actor
			// verb before the seed/1 boundary is illegal for every
			// signer, so the cooperative client must refuse it exactly
			// as the hook does (exit 3; review finding on #100), never
			// as an unresolvable-signer chain complaint.
			if keyring.IsActorVerb(rec.Event.Verb) && !keyring.Applies(c.Active) {
				return &VerbInactiveError{Verb: rec.Event.Verb, Active: c.Active, Needs: version.Seed1}
			}
			return nil
		}},
		{Name: "actor", Check: func(c *Context, rec *event.Record) error {
			pub, ok := c.Resolve(rec.Event.Actor)
			if !ok {
				return fmt.Errorf("%w: %s", ledger.ErrUnknownActor, rec.Event.Actor)
			}
			return rec.Verify(pub)
		}},
		{Name: "version", Check: func(c *Context, rec *event.Record) error {
			if !c.Supported[c.Active] {
				return &ledger.Failure{Position: c.Count, Reason: ledger.ReasonVersionUnsupported,
					Detail: fmt.Sprintf("active version %q is not in this implementation's supported set", c.Active)}
			}
			if rec.Event.V != c.Active {
				return &ledger.Failure{Position: c.Count, Reason: ledger.ReasonVersionMismatch,
					Detail: fmt.Sprintf("event carries %q, the version active at the tip is %q", rec.Event.V, c.Active)}
			}
			return nil
		}},
		{Name: "classification", Check: func(c *Context, rec *event.Record) error {
			if vs := classify.Lint(rec.Event.Payload); len(vs) > 0 {
				return &ClassificationError{Violations: vs}
			}
			return nil
		}},
		{Name: "grant", Check: func(c *Context, rec *event.Record) error {
			if !keyring.Applies(c.Active) || c.Keyring == nil {
				return nil
			}
			if accepted := keyring.AcceptedCapabilities(rec.Event.Verb); len(accepted) > 0 &&
				!c.Keyring.HasAnyCapability(rec.Event.Actor, accepted) {
				return &OutOfGrantError{Actor: rec.Event.Actor, Verb: rec.Event.Verb, Accepted: accepted}
			}
			if keyring.IsActorVerb(rec.Event.Verb) {
				// The shared transition function is the shape and
				// legality authority; admission previews it so a draft
				// that history would refuse never leaves the client.
				return c.Keyring.Preview(rec)
			}
			return nil
		}},
		{Name: "fence", Check: func(c *Context, rec *event.Record) error {
			// The fence rule (plans/os-5dc16a7c.md), between grant and
			// lifecycle per the charter's check order. The fence is the
			// admitted claim.taken position, derived never asserted; on
			// a held subject the four deliberate exits must cite it, so
			// must free events from the holder or any prior claimant of
			// the subject (a reaped worker cannot demote itself to
			// observer), and any citation present must match the active
			// fence whoever signs. Outside a claim window no fence
			// exists: none is required, and citing one refuses.
			if !keyring.Applies(c.Active) || c.Table == nil || c.Lifecycle == nil {
				return nil
			}
			verb := rec.Event.Verb
			if verb == transition.RunStartedVerb || verb == transition.RunSettledVerb {
				// The run facts' "fence" field is the run's window
				// reference, not a fence-rule citation: a settle
				// legitimately cites a PRIOR fence after its window
				// closed (plans/os-1dad487d.md), which the
				// active-fence citation semantics here would refuse.
				// The run rule validates the reference against the
				// applied claim positions instead.
				return nil
			}
			if c.Table.Exclusive(verb) {
				if s, ok := c.Lifecycle.State(rec.Event.Subject); ok && s.Claim != nil {
					// A rival claim is contention, the lifecycle
					// rule's structured refusal, never a fence
					// complaint.
					return nil
				}
				// With no active claim no fence exists, and the
				// claiming verb asserts none: fences are derived from
				// the admitted position, so a citation here refuses
				// like any other claimless citation.
				if cited, hasCited := fenceCitation(rec.Event.Payload); hasCited {
					return &FenceError{Subject: rec.Event.Subject, Cited: cited, Active: -1}
				}
				return nil
			}
			cited, hasCited := fenceCitation(rec.Event.Payload)
			s, ok := c.Lifecycle.State(rec.Event.Subject)
			if !ok || s.Claim == nil {
				if hasCited {
					return &FenceError{Subject: rec.Event.Subject, Cited: cited, Active: -1}
				}
				return nil
			}
			required := false
			if c.Table.IsLifecycleVerb(verb) {
				// The deliberate exits are exactly the lifecycle verbs
				// the table allows out of the held state.
				required = c.Table.Allows(s.State, verb)
			} else if rec.Event.Actor == s.Claim.Holder || s.PriorClaimants[rec.Event.Actor] {
				required = true
			}
			if !hasCited {
				if required {
					return &FenceError{Subject: rec.Event.Subject, Active: s.Claim.Fence, Holder: s.Claim.Holder}
				}
				return nil
			}
			if cited != fmt.Sprintf("%d", s.Claim.Fence) {
				return &FenceError{Subject: rec.Event.Subject, Cited: cited, Active: s.Claim.Fence, Holder: s.Claim.Holder}
			}
			return nil
		}},
		{Name: "packet", Check: func(c *Context, rec *event.Record) error {
			// Every deliberate exit carries a four-part handoff packet
			// (plans/os-b07b0f59.md): the shape refusal lands before the
			// transition applies, after the fence. seed/0 stays inert.
			if !keyring.Applies(c.Active) {
				return nil
			}
			if !packet.Required(rec.Event.Verb) {
				return nil
			}
			_, err := packet.FromPayload(rec.Event.Subject, rec.Event.Payload)
			return err
		}},
		{Name: "plan", Check: func(c *Context, rec *event.Record) error {
			// The plan gate (plans/os-16c1d142.md): plan.* payloads
			// carry their anchors, and a submission above the trivial
			// tier requires an admitted plan.approved plus the cited
			// plan anchor (exit 16 plan_required). The ancestry
			// binding is Phase 6's receipt computation.
			if !keyring.Applies(c.Active) || c.Lifecycle == nil {
				return nil
			}
			verb := rec.Event.Verb
			if err := transition.CheckPlanEventShape(verb, rec.Event.Subject, rec.Event.Payload); err != nil {
				return err
			}
			if verb != "submission.made" {
				return nil
			}
			tier := ""
			if s, ok := c.Lifecycle.State(rec.Event.Subject); ok {
				tier = s.Tier
			}
			return c.Lifecycle.CheckPlanGate(rec.Event.Subject, tier, rec.Event.Payload)
		}},
		{Name: "observation", Check: func(c *Context, rec *event.Record) error {
			// The summarization boundary (plans/os-2ff8dbf1.md):
			// milestones are coarse, monotonic, position-throttled
			// facts, and a declared wedge carries its evidence. Both
			// are free verbs; capability rows and the fence matrix
			// apply through their own rules.
			if !keyring.Applies(c.Active) || c.Lifecycle == nil {
				return nil
			}
			switch rec.Event.Verb {
			case transition.MilestoneVerb:
				return c.Lifecycle.CheckMilestone(rec.Event.Subject, c.Count, rec.Event.Payload)
			case transition.WedgeDeclaredVerb:
				return transition.CheckWedgeShape(rec.Event.Subject, rec.Event.Payload)
			}
			return nil
		}},
		{Name: "proposal", Check: func(c *Context, rec *event.Record) error {
			// Outside text can propose, never arm (III.F row 2,
			// plans/os-73c00a50.md): request.* payloads structurally
			// cannot carry executable or gate keys at any depth.
			if !keyring.Applies(c.Active) {
				return nil
			}
			if !strings.HasPrefix(rec.Event.Verb, transition.ProposalVerbPrefix) {
				return nil
			}
			return transition.CheckProposalShape(rec.Event.Subject, rec.Event.Payload)
		}},
		{Name: "verdict", Check: func(c *Context, rec *event.Record) error {
			// The verdict pipeline's admission half
			// (plans/os-f6d2c267.md; next/spec/verdicts.md): a fact
			// admitted only on review subjects, bound to the
			// submission it judges, signed by a key disjoint from
			// every implementing key (L1). Capability rides the grant
			// rule; the fence rule already refuses citations outside
			// a claim window.
			if !keyring.Applies(c.Active) || c.Lifecycle == nil {
				return nil
			}
			if rec.Event.Verb != transition.VerdictRenderedVerb {
				return nil
			}
			subject := rec.Event.Subject
			s, ok := c.Lifecycle.State(subject)
			if !ok {
				return &transition.InvalidTransitionError{Subject: subject, Verb: rec.Event.Verb}
			}
			if s.State != "review" {
				return &transition.InvalidTransitionError{Subject: subject, From: s.State, Verb: rec.Event.Verb}
			}
			var p struct {
				Verdict      string `json:"verdict"`
				Receipt      string `json:"receipt"`
				Submission   string `json:"submission"`
				Independence string `json:"independence"`
			}
			dec := json.NewDecoder(bytes.NewReader(rec.Event.Payload))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&p); err != nil {
				return &VerdictError{Subject: subject, Reason: fmt.Sprintf("the payload is the strict object {verdict, receipt, submission, independence}: %v", err)}
			}
			var missing []string
			for _, f := range []struct{ name, v string }{{"verdict", p.Verdict}, {"receipt", p.Receipt}, {"submission", p.Submission}, {"independence", p.Independence}} {
				if strings.TrimSpace(f.v) == "" {
					missing = append(missing, f.name)
				}
			}
			if len(missing) > 0 {
				return &transition.IncompleteError{Verb: rec.Event.Verb, Subject: subject, Missing: missing}
			}
			if p.Verdict != "pass" && p.Verdict != "fail" {
				return &VerdictError{Subject: subject, Reason: fmt.Sprintf("verdict %q is neither literal pass nor fail", p.Verdict)}
			}
			if !receiptDigestRE.MatchString(p.Receipt) {
				return &VerdictError{Subject: subject, Reason: fmt.Sprintf("receipt %q is not a lowercase-hex sha256 digest of the receipt's JCS bytes", p.Receipt)}
			}
			if p.Independence != "L1" {
				return &VerdictError{Subject: subject, Reason: fmt.Sprintf("independence %q is not the v0 literal \"L1\" — the level vocabulary widens when Phase 10 declares levels per tier", p.Independence)}
			}
			if s.Submission == nil {
				return &VerdictError{Subject: subject, Reason: "the fold records no submission on this review subject, so there is nothing to bind a verdict to"}
			}
			if p.Submission != fmt.Sprintf("%d", s.Submission.Pos) {
				return &VerdictError{Subject: subject, Reason: fmt.Sprintf("submission %q is not the bound submission at position %d — a verdict judges exactly the submission that produced the review state", p.Submission, s.Submission.Pos)}
			}
			// The red-verdict lockout (plans/os-d2497eb7.md): pass
			// over a submission an authenticated fail already judged
			// refuses until a NEW submission; fail restatements stay
			// admissible. Only boundary-validated fails lock, and the
			// whole window is scanned, so a raw-pushed later verdict
			// can never bury an authentic fail.
			if p.Verdict == "pass" {
				if locked := authenticFail(c, subject, s); locked != nil {
					return &VerdictError{Subject: subject, Reason: fmt.Sprintf("a fail verdict at position %d already judged the bound submission — a red verdict locks pass out until a new submission (contract.returned, re-claim, resubmit)", locked.Pos)}
				}
			}
			if s.PriorClaimants[rec.Event.Actor] {
				return &NotIndependentError{Subject: subject, Actor: rec.Event.Actor, Role: "a claimant, past or present"}
			}
			if rec.Event.Actor == s.Submission.Signer {
				return &NotIndependentError{Subject: subject, Actor: rec.Event.Actor, Role: "the bound submission's signer"}
			}
			return nil
		}},
		{Name: "offer", Check: func(c *Context, rec *event.Record) error {
			// The supervisor's invitation (plans/os-c61c3392.md;
			// next/spec/offers.md; SEED-NEXT.md §II.9): a fact admitted
			// only while the subject folds to ready, strictly shaped,
			// whose expiry lies strictly after the event's own ts —
			// admission never reads a wall clock, so a born-dead offer
			// refuses deterministically. Capability rides the grant
			// rule; liveness (claimed-or-expire) is derived at the
			// consuming surface, never stored.
			if !keyring.Applies(c.Active) || c.Lifecycle == nil {
				return nil
			}
			if rec.Event.Verb != transition.OfferPublishedVerb {
				return nil
			}
			subject := rec.Event.Subject
			s, ok := c.Lifecycle.State(subject)
			if !ok {
				return &transition.InvalidTransitionError{Subject: subject, Verb: rec.Event.Verb}
			}
			if s.State != "ready" {
				return &transition.InvalidTransitionError{Subject: subject, From: s.State, Verb: rec.Event.Verb}
			}
			var p struct {
				Eligibility *struct {
					Capabilities []string `json:"capabilities"`
					Tiers        []string `json:"tiers"`
				} `json:"eligibility"`
				Expires string `json:"expires"`
			}
			dec := json.NewDecoder(bytes.NewReader(rec.Event.Payload))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&p); err != nil {
				return &OfferError{Subject: subject, Reason: fmt.Sprintf("the payload is the strict object {eligibility{capabilities, tiers}, expires}: %v", err)}
			}
			var missing []string
			if p.Eligibility == nil {
				missing = append(missing, "eligibility")
			}
			if strings.TrimSpace(p.Expires) == "" {
				missing = append(missing, "expires")
			}
			if len(missing) > 0 {
				return &transition.IncompleteError{Verb: rec.Event.Verb, Subject: subject, Missing: missing}
			}
			exp, err := time.Parse(time.RFC3339, strings.TrimSpace(p.Expires))
			if err != nil {
				return &OfferError{Subject: subject, Reason: fmt.Sprintf("expires %q is not an RFC3339 timestamp", p.Expires)}
			}
			ts, err := time.Parse(time.RFC3339, rec.Event.TS)
			if err != nil {
				return &OfferError{Subject: subject, Reason: fmt.Sprintf("the event ts %q is not an RFC3339 timestamp to anchor expiry against", rec.Event.TS)}
			}
			if !exp.After(ts) {
				return &OfferError{Subject: subject, Reason: fmt.Sprintf("expires %s is not after the event's own ts %s — a born-dead offer invites nothing (claimed or expire, SEED-NEXT.md §II.9)", exp.Format(time.RFC3339), ts.Format(time.RFC3339))}
			}
			return nil
		}},
		{Name: "budget", Check: func(c *Context, rec *event.Record) error {
			// The reservation machinery (plans/os-cecac5de.md;
			// next/spec/budgets.md; SEED-NEXT.md §II.9): admission is
			// the one place with a serialized view, so the reserve is
			// checked and decremented HERE — the second of two racing
			// drafts admits against the tip that already carries the
			// first. Capability rides the grant rule; the fence rule
			// forces the holder's citation; validity and effective
			// closure are position-accurate derivations, never stored
			// state.
			if !keyring.Applies(c.Active) || c.Lifecycle == nil {
				return nil
			}
			verb := rec.Event.Verb
			isBudget := verb == transition.BudgetReserveVerb || verb == transition.BudgetSettleVerb || verb == transition.BudgetReleaseVerb
			if !isBudget && !transition.IsSpendingVerb(verb) {
				return nil
			}
			subject := rec.Event.Subject
			s, ok := c.Lifecycle.State(subject)
			if !ok {
				return &transition.InvalidTransitionError{Subject: subject, Verb: verb}
			}
			view := BudgetViewAt(c.Records, c.Table, subject, s)
			if !isBudget {
				// The spending gate (D5): a listed verb needs an open
				// valid reservation; the table ships empty and 7.3's
				// metering fills it.
				if len(view.Open) == 0 {
					return &BudgetError{Subject: subject, Reason: fmt.Sprintf("%s spends, and no open valid reservation stands — spending verbs require an admitted budget.reserve (SEED-NEXT.md §II.9)", verb)}
				}
				return nil
			}
			if s.State != "in_progress" {
				return &transition.InvalidTransitionError{Subject: subject, From: s.State, Verb: verb}
			}
			operatorNow := c.Keyring != nil && c.Keyring.HasAnyCapability(rec.Event.Actor, []string{keyring.CapOperator})
			switch verb {
			case transition.BudgetReserveVerb:
				var p struct {
					Amount string `json:"amount"`
					Fence  string `json:"fence"`
				}
				dec := json.NewDecoder(bytes.NewReader(rec.Event.Payload))
				dec.DisallowUnknownFields()
				if err := dec.Decode(&p); err != nil {
					return &BudgetError{Subject: subject, Reason: fmt.Sprintf("the reserve payload is the strict object {amount, fence}: %v", err)}
				}
				amount, err := strconv.Atoi(strings.TrimSpace(p.Amount))
				if err != nil || amount <= 0 {
					return &BudgetError{Subject: subject, Reason: fmt.Sprintf("amount %q is not a positive integer of class units", p.Amount)}
				}
				if !operatorNow && (s.Claim == nil || s.Claim.Holder != rec.Event.Actor) {
					return &BudgetError{Subject: subject, Reason: "only the active claim holder or the operator lane reserves — a prior claimant's reserve would consume a budget it no longer works under"}
				}
				if !view.Known {
					return &BudgetError{Subject: subject, Reason: fmt.Sprintf("budget class %q has no capacity in the class table — absent knowledge is never fudged into spendable units", s.Budget)}
				}
				if amount > view.Remaining {
					return &BudgetError{Subject: subject, Reason: fmt.Sprintf("amount %d exceeds remaining %d of capacity %d — reservations are checked and decremented at admission, the serialized view", amount, view.Remaining, view.Capacity)}
				}
				return nil
			default:
				var p struct {
					Reservation string `json:"reservation"`
					Actuals     string `json:"actuals"`
					Fence       string `json:"fence"`
				}
				dec := json.NewDecoder(bytes.NewReader(rec.Event.Payload))
				dec.DisallowUnknownFields()
				if err := dec.Decode(&p); err != nil {
					return &BudgetError{Subject: subject, Reason: fmt.Sprintf("the close payload is the strict object {reservation%s, fence}: %v", map[bool]string{true: ", actuals"}[verb == transition.BudgetSettleVerb], err)}
				}
				if verb == transition.BudgetReleaseVerb && p.Actuals != "" {
					return &BudgetError{Subject: subject, Reason: "release frees a reservation with zero actuals — settle is the verb that records spend"}
				}
				cited, err := strconv.Atoi(strings.TrimSpace(p.Reservation))
				if err != nil {
					return &BudgetError{Subject: subject, Reason: fmt.Sprintf("reservation %q is not a chain position", p.Reservation)}
				}
				var res *transition.ReservationFact
				for i := range s.Reservations {
					if s.Reservations[i].Pos == cited {
						res = &s.Reservations[i]
						break
					}
				}
				if res == nil {
					return &BudgetError{Subject: subject, Reason: fmt.Sprintf("position %d is no reservation on this subject", cited)}
				}
				if !ReservationValid(c.Records, c.Table, subject, *res) {
					return &BudgetError{Subject: subject, Reason: fmt.Sprintf("the reservation at position %d never passed the authoring boundary — closing it would launder it into spend history", cited)}
				}
				if closed, ok := view.ClosedBy[cited]; ok {
					return &BudgetError{Subject: subject, Reason: fmt.Sprintf("the reservation at position %d is already effectively closed at position %d", cited, closed.Pos)}
				}
				if verb == transition.BudgetSettleVerb {
					n, err := strconv.Atoi(strings.TrimSpace(p.Actuals))
					if err != nil || n < 0 {
						return &BudgetError{Subject: subject, Reason: fmt.Sprintf("actuals %q is not a non-negative integer of class units", p.Actuals)}
					}
				}
				if rec.Event.Actor != res.Signer && !operatorNow {
					return &BudgetError{Subject: subject, Reason: fmt.Sprintf("only the reservation's own reserving signer or the operator lane closes it — the reservation at %d belongs to %s", cited, res.Signer)}
				}
				return nil
			}
		}},
		{Name: "run", Check: func(c *Context, rec *event.Record) error {
			// The execution-run facts (plans/os-1dad487d.md;
			// next/spec/executors.md): run.started is the spending
			// gate's first customer (the budget rule already required
			// an open valid reservation on the subject; this rule
			// revalidates the SPECIFIC citation, the laundering
			// shape), and run.settled is the once-per-fence aggregate
			// on a fence that carries an admitted start. Capability
			// rides the grant rule.
			if !keyring.Applies(c.Active) || c.Lifecycle == nil {
				return nil
			}
			verb := rec.Event.Verb
			if verb != transition.RunStartedVerb && verb != transition.RunSettledVerb {
				return nil
			}
			subject := rec.Event.Subject
			s, ok := c.Lifecycle.State(subject)
			if !ok {
				return &transition.InvalidTransitionError{Subject: subject, Verb: verb}
			}
			startValid := func(st transition.RunStartFact) bool {
				return RunStartValid(c.Records, c.Table, subject, s, st)
			}
			if verb == transition.RunStartedVerb {
				var p struct {
					Fence       string `json:"fence"`
					Reservation string `json:"reservation"`
				}
				dec := json.NewDecoder(bytes.NewReader(rec.Event.Payload))
				dec.DisallowUnknownFields()
				if err := dec.Decode(&p); err != nil {
					return &RunError{Subject: subject, Reason: fmt.Sprintf("the start payload is the strict object {fence, reservation}: %v", err)}
				}
				if s.State != "in_progress" {
					return &transition.InvalidTransitionError{Subject: subject, From: s.State, Verb: verb}
				}
				fence, err := strconv.Atoi(strings.TrimSpace(p.Fence))
				if err != nil || s.Claim == nil || fence != s.Claim.Fence {
					return &RunError{Subject: subject, Reason: fmt.Sprintf("fence %q is not the active claim fence — a run starts inside the claim window it spends under", p.Fence)}
				}
				for _, st := range s.RunStarts {
					if st.Fence == fence && startValid(st) {
						return &RunError{Subject: subject, Reason: fmt.Sprintf("fence %d already carries a run.started at position %d — one run per claim window", fence, st.Pos)}
					}
				}
				cited, err := strconv.Atoi(strings.TrimSpace(p.Reservation))
				if err != nil {
					return &RunError{Subject: subject, Reason: fmt.Sprintf("reservation %q is not a chain position", p.Reservation)}
				}
				var res *transition.ReservationFact
				for i := range s.Reservations {
					if s.Reservations[i].Pos == cited {
						res = &s.Reservations[i]
						break
					}
				}
				if res == nil {
					return &RunError{Subject: subject, Reason: fmt.Sprintf("position %d is no reservation on this subject", cited)}
				}
				view := BudgetViewAt(c.Records, c.Table, subject, s)
				if !ReservationValid(c.Records, c.Table, subject, *res) {
					return &RunError{Subject: subject, Reason: fmt.Sprintf("the reservation at position %d never passed the authoring boundary — a run cannot be fenced to laundered spend", cited)}
				}
				if _, closed := view.ClosedBy[cited]; closed {
					return &RunError{Subject: subject, Reason: fmt.Sprintf("the reservation at position %d is already effectively closed — a run needs an open reservation", cited)}
				}
				return nil
			}
			var p struct {
				Fence string `json:"fence"`
				Units string `json:"units"`
				Lines string `json:"lines"`
			}
			dec := json.NewDecoder(bytes.NewReader(rec.Event.Payload))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&p); err != nil {
				return &RunError{Subject: subject, Reason: fmt.Sprintf("the settle payload is the strict object {fence, units, lines}: %v", err)}
			}
			fence, err := strconv.Atoi(strings.TrimSpace(p.Fence))
			if err != nil || !s.ClaimFences[fence] {
				return &RunError{Subject: subject, Reason: fmt.Sprintf("fence %q is not an applied claim position on this subject", p.Fence)}
			}
			started := false
			for _, st := range s.RunStarts {
				if st.Fence == fence && startValid(st) {
					started = true
					break
				}
			}
			if !started {
				return &RunError{Subject: subject, Reason: fmt.Sprintf("fence %d carries no admitted run.started — a run settles only after its gated initiation", fence)}
			}
			for _, r := range s.Runs {
				if r.Fence == fence {
					return &RunError{Subject: subject, Reason: fmt.Sprintf("fence %d already carries a run.settled at position %d — one run, one aggregate", fence, r.Pos)}
				}
			}
			for name, v := range map[string]string{"units": p.Units, "lines": p.Lines} {
				n, err := strconv.Atoi(strings.TrimSpace(v))
				if err != nil || n < 0 {
					return &RunError{Subject: subject, Reason: fmt.Sprintf("%s %q is not a non-negative integer", name, v)}
				}
			}
			return nil
		}},
		{Name: "chain", Check: func(c *Context, rec *event.Record) error {
			// The reconciliation chain (plans/os-6cdc15be.md;
			// next/spec/reconciliation.md): done is reachable only
			// through verdict.rendered(pass), merge.requested, and
			// merge.observed, in order. merge.requested is a fact
			// admitted only in review citing the pass verdict;
			// merge.observed's state legality stays the table's (the
			// lifecycle rule), and this rule holds its chain links and
			// forge-fact shape.
			if !keyring.Applies(c.Active) || c.Lifecycle == nil {
				return nil
			}
			verb := rec.Event.Verb
			switch verb {
			case transition.MergeRequestedVerb:
				subject := rec.Event.Subject
				s, ok := c.Lifecycle.State(subject)
				if !ok {
					return &transition.InvalidTransitionError{Subject: subject, Verb: verb}
				}
				if s.State != "review" {
					return &transition.InvalidTransitionError{Subject: subject, From: s.State, Verb: verb}
				}
				var p struct {
					Verdict  string `json:"verdict"`
					Override string `json:"override"`
				}
				dec := json.NewDecoder(bytes.NewReader(rec.Event.Payload))
				dec.DisallowUnknownFields()
				if err := dec.Decode(&p); err != nil {
					return &transition.ChainError{Subject: subject, Verb: verb, Reason: fmt.Sprintf("the payload is the strict object {verdict} or {override}: %v", err)}
				}
				hasV, hasO := strings.TrimSpace(p.Verdict) != "", strings.TrimSpace(p.Override) != ""
				if hasV == hasO {
					return &transition.ChainError{Subject: subject, Verb: verb, Reason: "the request cites exactly one of verdict or override — the two chain paths never blur (plans/os-d2497eb7.md)"}
				}
				if hasO {
					cited, err := strconv.Atoi(strings.TrimSpace(p.Override))
					if err != nil {
						return &transition.ChainError{Subject: subject, Verb: verb, Reason: fmt.Sprintf("override %q is not a chain position", p.Override)}
					}
					if s.Override == nil {
						return &transition.ChainError{Subject: subject, Verb: verb, Reason: "no override stands on this subject — merge.overridden is the operator's attributable act, and citing one that does not exist substitutes for nothing"}
					}
					if cited != s.Override.Pos {
						return &transition.ChainError{Subject: subject, Verb: verb, Reason: fmt.Sprintf("cites position %d; the admitted override on this subject is at position %d", cited, s.Override.Pos)}
					}
					if ce := overrideBacking(c, subject, verb, s); ce != nil {
						return ce
					}
					return nil
				}
				cited, err := strconv.Atoi(strings.TrimSpace(p.Verdict))
				if err != nil {
					return &transition.ChainError{Subject: subject, Verb: verb, Reason: fmt.Sprintf("verdict %q is not a chain position", p.Verdict)}
				}
				if s.Verdict == nil {
					return &transition.ChainError{Subject: subject, Verb: verb, Reason: "no verdict has been rendered on this subject — the chain starts at verdict.rendered(pass)"}
				}
				if cited != s.Verdict.Pos {
					return &transition.ChainError{Subject: subject, Verb: verb, Reason: fmt.Sprintf("cites position %d; the admitted verdict on this subject is at position %d", cited, s.Verdict.Pos)}
				}
				if s.Verdict.Verdict != "pass" {
					return &transition.ChainError{Subject: subject, Verb: verb, Reason: fmt.Sprintf("the verdict at position %d is %q — a red verdict is unmergeable", s.Verdict.Pos, s.Verdict.Verdict)}
				}
				if ce := launderedVerdict(c, subject, verb, s); ce != nil {
					return ce
				}
				return nil
			case transition.MergeObservedVerb:
				subject := rec.Event.Subject
				s, ok := c.Lifecycle.State(subject)
				if !ok || s.State != "review" {
					// State legality is the table's; the lifecycle rule
					// refuses with the proper positioned message.
					return nil
				}
				var p struct {
					Merged string `json:"merged"`
					PR     string `json:"pr"`
				}
				dec := json.NewDecoder(bytes.NewReader(rec.Event.Payload))
				dec.DisallowUnknownFields()
				if err := dec.Decode(&p); err != nil {
					return &transition.ChainError{Subject: subject, Verb: verb, Reason: fmt.Sprintf("the payload is the strict object {merged, pr}: %v", err)}
				}
				var missing []string
				if strings.TrimSpace(p.Merged) == "" {
					missing = append(missing, "merged")
				}
				if strings.TrimSpace(p.PR) == "" {
					missing = append(missing, "pr")
				}
				if len(missing) > 0 {
					return &transition.IncompleteError{Verb: verb, Subject: subject, Missing: missing}
				}
				if !mergedSHARE.MatchString(strings.TrimSpace(p.Merged)) {
					return &transition.ChainError{Subject: subject, Verb: verb, Reason: fmt.Sprintf("merged %q is not a full lowercase-hex commit — the observer records which commit the forge merged", p.Merged)}
				}
				// The override-backed path (plans/os-d2497eb7.md): an
				// admitted override plus a request citing it stands in
				// for the pass verdict plus its citation — each step
				// still its own event.
				if s.Override != nil && s.Requested != nil && s.Requested.CitedOverride == s.Override.Pos {
					if ce := overrideBacking(c, subject, verb, s); ce != nil {
						return ce
					}
					return nil
				}
				if s.Verdict == nil || s.Verdict.Verdict != "pass" {
					return &transition.ChainError{Subject: subject, Verb: verb, Reason: "no pass verdict on this subject — done is reachable only through the full chain"}
				}
				if s.Requested == nil || s.Requested.CitedVerdict != s.Verdict.Pos {
					return &transition.ChainError{Subject: subject, Verb: verb, Reason: fmt.Sprintf("no merge.requested cites the pass verdict at position %d — each chain step is its own event", s.Verdict.Pos)}
				}
				if ce := launderedVerdict(c, subject, verb, s); ce != nil {
					return ce
				}
				return nil
			case transition.MergeOverriddenVerb:
				// The operator's attributable substitute for a pass
				// verdict (plans/os-d2497eb7.md): admitted only over a
				// standing, boundary-validated fail on the current
				// submission — an escape hatch, never a bypass.
				subject := rec.Event.Subject
				s, ok := c.Lifecycle.State(subject)
				if !ok {
					return &transition.InvalidTransitionError{Subject: subject, Verb: verb}
				}
				if s.State != "review" {
					return &transition.InvalidTransitionError{Subject: subject, From: s.State, Verb: verb}
				}
				var p struct {
					Reason  string `json:"reason"`
					Verdict string `json:"verdict"`
				}
				dec := json.NewDecoder(bytes.NewReader(rec.Event.Payload))
				dec.DisallowUnknownFields()
				if err := dec.Decode(&p); err != nil {
					return &transition.ChainError{Subject: subject, Verb: verb, Reason: fmt.Sprintf("the payload is the strict object {reason, verdict}: %v", err)}
				}
				var missing []string
				if strings.TrimSpace(p.Reason) == "" {
					missing = append(missing, "reason")
				}
				if strings.TrimSpace(p.Verdict) == "" {
					missing = append(missing, "verdict")
				}
				if len(missing) > 0 {
					return &transition.IncompleteError{Verb: verb, Subject: subject, Missing: missing}
				}
				if s.Override != nil {
					return &transition.ChainError{Subject: subject, Verb: verb, Reason: fmt.Sprintf("an override already stands at position %d — one override per submission window; a return and a new submission open the next", s.Override.Pos)}
				}
				cited, err := strconv.Atoi(strings.TrimSpace(p.Verdict))
				if err != nil {
					return &transition.ChainError{Subject: subject, Verb: verb, Reason: fmt.Sprintf("verdict %q is not a chain position", p.Verdict)}
				}
				fail := windowFail(s, cited)
				if fail == nil {
					return &transition.ChainError{Subject: subject, Verb: verb, Reason: fmt.Sprintf("position %d is not a fail verdict on the current submission — the override overrules a standing red verdict, it never routes around independent verification", cited)}
				}
				if ce := verdictBoundary(c, subject, verb, s, *fail); ce != nil {
					return ce
				}
				return nil
			case transition.ContractReturnedVerb:
				// State legality is the table's (the lifecycle rule);
				// this case holds the citation: the return is
				// authorized by a standing, boundary-validated fail
				// (plans/os-d2497eb7.md).
				subject := rec.Event.Subject
				s, ok := c.Lifecycle.State(subject)
				if !ok || s.State != "review" {
					return nil
				}
				var p struct {
					Verdict string `json:"verdict"`
				}
				dec := json.NewDecoder(bytes.NewReader(rec.Event.Payload))
				dec.DisallowUnknownFields()
				if err := dec.Decode(&p); err != nil {
					return &transition.ChainError{Subject: subject, Verb: verb, Reason: fmt.Sprintf("the payload is the strict object {verdict}: %v", err)}
				}
				if strings.TrimSpace(p.Verdict) == "" {
					return &transition.IncompleteError{Verb: verb, Subject: subject, Missing: []string{"verdict"}}
				}
				cited, err := strconv.Atoi(strings.TrimSpace(p.Verdict))
				if err != nil {
					return &transition.ChainError{Subject: subject, Verb: verb, Reason: fmt.Sprintf("verdict %q is not a chain position", p.Verdict)}
				}
				fail := windowFail(s, cited)
				if fail == nil {
					return &transition.ChainError{Subject: subject, Verb: verb, Reason: fmt.Sprintf("position %d is not a fail verdict on the current submission — nobody yanks an in-review contract whose verdict is pass or pending", cited)}
				}
				if ce := verdictBoundary(c, subject, verb, s, *fail); ce != nil {
					return ce
				}
				return nil
			}
			return nil
		}},
		{Name: "seal", Check: func(c *Context, rec *event.Record) error {
			// The sealed-checks commitment and its authoring isolation
			// (plans/os-3128535a.md; next/spec/sealed-checks.md).
			// Capability rides the grant rule (sealer only); grant
			// disjointness rides the keyring preview.
			if !keyring.Applies(c.Active) || c.Lifecycle == nil {
				return nil
			}
			verb := rec.Event.Verb
			switch verb {
			case transition.CheckSealedVerb:
				subject := rec.Event.Subject
				s, ok := c.Lifecycle.State(subject)
				if !ok {
					return &transition.InvalidTransitionError{Subject: subject, Verb: verb}
				}
				if s.State != "ready" {
					return &transition.InvalidTransitionError{Subject: subject, From: s.State, Verb: verb}
				}
				if len(s.PriorClaimants) > 0 {
					// Ready again after claim.released or claim.reaped:
					// implementation already began, and a commitment
					// appended now would not prove pre-existence
					// (review finding on the 6.3 plan).
					return &transition.ChainError{Subject: subject, Verb: verb, Reason: "the subject has already been claimed — a commitment after implementation began proves nothing; cancel and re-file to change sealed checks"}
				}
				if s.Sealed != nil {
					return &transition.ChainError{Subject: subject, Verb: verb, Reason: fmt.Sprintf("a commitment already stands at position %d — one seal per subject; rotation re-encrypts, never re-commits", s.Sealed.Pos)}
				}
				var p struct {
					Commitment string `json:"commitment"`
				}
				dec := json.NewDecoder(bytes.NewReader(rec.Event.Payload))
				dec.DisallowUnknownFields()
				if err := dec.Decode(&p); err != nil {
					return &transition.ChainError{Subject: subject, Verb: verb, Reason: fmt.Sprintf("the payload is the strict object {commitment}: %v", err)}
				}
				if strings.TrimSpace(p.Commitment) == "" {
					return &transition.IncompleteError{Verb: verb, Subject: subject, Missing: []string{"commitment"}}
				}
				if !receiptDigestRE.MatchString(strings.TrimSpace(p.Commitment)) {
					return &transition.ChainError{Subject: subject, Verb: verb, Reason: fmt.Sprintf("commitment %q is not a lowercase-hex sha256 — the ledger references the sealed body by its salted hash", p.Commitment)}
				}
				return nil
			case "claim.taken":
				// The per-subject half of authoring isolation: the key
				// that sealed the checks never implements against them.
				s, ok := c.Lifecycle.State(rec.Event.Subject)
				if ok && s.Sealed != nil && s.Sealed.Signer == rec.Event.Actor {
					return &transition.ChainError{Subject: rec.Event.Subject, Verb: verb, Reason: fmt.Sprintf("the claiming key authored this subject's sealed checks at position %d — sealed checks are authored under a grant disjoint from implementation", s.Sealed.Pos)}
				}
				return nil
			}
			return nil
		}},
		{Name: "lifecycle", Check: func(c *Context, rec *event.Record) error {
			// Lifecycle legality is admission policy at seed/1, the
			// halt/classification/grant precedent (plans/os-d69a6c91.md):
			// seed/0 records are grandfathered inert, verification
			// tolerates illegal history, and the projection fold skips
			// it visibly. The table is the only legality authority.
			if !keyring.Applies(c.Active) || c.Table == nil || c.Lifecycle == nil {
				return nil
			}
			verb := rec.Event.Verb
			if !c.Table.IsLifecycleVerb(verb) {
				return nil
			}
			if err := transition.CheckCompleteness(verb, rec.Event.Subject, rec.Event.Payload); err != nil {
				return err
			}
			current := ""
			var claim *transition.Claim
			if s, ok := c.Lifecycle.State(rec.Event.Subject); ok {
				current = s.State
				claim = s.Claim
			}
			if c.Table.Exclusive(verb) && claim != nil {
				// Exclusivity not granted: the subject is held. The
				// loser learns who holds and since when (exit 2).
				return &ContentionError{Subject: rec.Event.Subject, Holder: claim.Holder, Fence: claim.Fence}
			}
			_, err := c.Table.Check(rec.Event.Subject, current, verb)
			return err
		}},
	}
}

// Run applies the rules in order; the first refusing rule wraps its
// error in a Refusal.
func Run(ctx *Context, rec *event.Record, rules []Rule) error {
	for _, r := range rules {
		if err := r.Check(ctx, rec); err != nil {
			return &Refusal{Rule: r.Name, Err: err}
		}
	}
	return nil
}

// Check runs the default rule set.
func Check(ctx *Context, rec *event.Record) error {
	return Run(ctx, rec, Default())
}

// Validate adapts the rule set to the gitref.Validate seam: the closure
// the cooperative client hands to AppendLoop (2.2) and the hook embeds
// (2.3). The store the loop passes is the refreshed materialized tip, so
// every retry re-runs admission against current state.
func Validate(opts ...Option) func(*ledger.Store, *event.Record) error {
	return func(store *ledger.Store, rec *event.Record) error {
		ctx, err := ContextAt(store, opts...)
		if err != nil {
			return err
		}
		return Check(ctx, rec)
	}
}

// receiptDigestRE is the verdict payload's receipt-digest wire form.
var receiptDigestRE = regexp.MustCompile(`^[0-9a-f]{64}$`)

// mergedSHARE is merge.observed's forge-fact wire form: a full
// lowercase-hex commit (40-64 hex, the classify anchor convention).
var mergedSHARE = regexp.MustCompile(`^[0-9a-f]{40,64}$`)

// launderedVerdict refuses an admitted chain step built on a folded
// verdict that never passed the verifier boundary (review finding on
// the 6.2 task PR): in raw-pushed history any active signer can plant
// a verdict.rendered, and without this check an ADMITTED
// merge.requested or merge.observed would launder it into a clean
// chain. The signer must hold the verdict capability (checked against
// the tip keyring, the v0 approximation of standing at the verdict's
// own position; seed reconcile replays position-accurately) and be
// disjoint from the implementing keys, exactly what the verdict rule
// enforces on the front door.
func launderedVerdict(c *Context, subject, verb string, s transition.SubjectState) *transition.ChainError {
	if c.Keyring == nil || s.Verdict == nil {
		return nil
	}
	return verdictBoundary(c, subject, verb, s, *s.Verdict)
}

// verdictBoundary checks one folded verdict fact against the verifier
// boundary: the signer holds the verdict grant (tip keyring, the v0
// approximation) and is no implementing key on the contract.
func verdictBoundary(c *Context, subject, verb string, s transition.SubjectState, fact transition.VerdictFact) *transition.ChainError {
	if c.Keyring == nil {
		return nil
	}
	signer := fact.Signer
	if !c.Keyring.HasAnyCapability(signer, keyring.AcceptedCapabilities(transition.VerdictRenderedVerb)) {
		return &transition.ChainError{Subject: subject, Verb: verb,
			Reason: fmt.Sprintf("the cited verdict at position %d was signed by %s, which holds no verdict grant — a raw-pushed verdict cannot be laundered through the admitted chain", fact.Pos, signer)}
	}
	if s.PriorClaimants[signer] || (s.Submission != nil && signer == s.Submission.Signer) {
		return &transition.ChainError{Subject: subject, Verb: verb,
			Reason: fmt.Sprintf("the cited verdict at position %d was signed by implementing key %s — L1 independence is not launderable through the admitted chain", fact.Pos, signer)}
	}
	return nil
}

// authenticFail returns the first fail verdict in the current
// submission window whose signer passes the verifier boundary: the
// red-verdict lockout consults only authenticated fails, and scanning
// the window means a later raw verdict can never bury one
// (plans/os-d2497eb7.md).
func authenticFail(c *Context, subject string, s transition.SubjectState) *transition.VerdictFact {
	for i := range s.SubmissionFails {
		if verdictBoundary(c, subject, "", s, s.SubmissionFails[i]) == nil {
			return &s.SubmissionFails[i]
		}
	}
	return nil
}

// windowFail finds the fail verdict at the cited position in the
// current submission window, nil when no such fail exists there.
func windowFail(s transition.SubjectState, pos int) *transition.VerdictFact {
	for i := range s.SubmissionFails {
		if s.SubmissionFails[i].Pos == pos {
			return &s.SubmissionFails[i]
		}
	}
	return nil
}

// launderedOverride checks the folded override's signer against the
// operator boundary (tip keyring): a raw-pushed override by an
// ungranted key substitutes for nothing (plans/os-d2497eb7.md).
func launderedOverride(c *Context, subject, verb string, o *transition.OverrideFact) *transition.ChainError {
	if c.Keyring == nil || o == nil {
		return nil
	}
	if !c.Keyring.HasAnyCapability(o.Signer, keyring.AcceptedCapabilities(transition.MergeOverriddenVerb)) {
		return &transition.ChainError{Subject: subject, Verb: verb,
			Reason: fmt.Sprintf("the cited override at position %d was signed by %s, which holds no operator standing — a raw-pushed override substitutes for nothing", o.Pos, o.Signer)}
	}
	return nil
}

// overrideBacking validates everything a chain step must re-check
// before trusting the folded override: the signer's operator standing
// AND the override's own citation — a standing, boundary-validated
// fail on the current submission. The override admission rule checks
// the citation for cooperative appends, but a raw-pushed well-shaped
// override by an operator-capable key folds without it, and trusting
// it here would turn the escape hatch into a wholesale bypass (review
// finding on the task PR).
func overrideBacking(c *Context, subject, verb string, s transition.SubjectState) *transition.ChainError {
	if ce := launderedOverride(c, subject, verb, s.Override); ce != nil {
		return ce
	}
	if s.Override == nil {
		return nil
	}
	fail := windowFail(s, s.Override.CitedVerdict)
	if fail == nil {
		return &transition.ChainError{Subject: subject, Verb: verb,
			Reason: fmt.Sprintf("the override at position %d cites position %d, which is not a fail verdict on the current submission — an override overrules a standing red verdict, it never routes around independent verification", s.Override.Pos, s.Override.CitedVerdict)}
	}
	if ce := verdictBoundary(c, subject, verb, s, *fail); ce != nil {
		return ce
	}
	return nil
}

// fenceCitation extracts the payload's fence field: the string form
// of the cited claim position, absent when the payload carries none.
// ReservationValid replays the keyring and the fold to the
// reservation's own position (plans/os-cecac5de.md D4, the
// VerifySeals pattern): valid iff the signer was the operator lane
// there, or held the claim capability AND was the subject's ACTIVE
// claim holder there. Prior claimants are excluded — a released
// worker reserving under the next holder's window would consume a
// budget it no longer works under (review finding on plan #147).
func ReservationValid(records []*event.Record, table *transition.Table, subject string, r transition.ReservationFact) bool {
	if r.Pos < 0 || r.Pos >= len(records) {
		return false
	}
	ring, _, err := keyring.StateAt(records[:r.Pos])
	if err != nil || ring == nil {
		return false
	}
	if ring.HasAnyCapability(r.Signer, []string{keyring.CapOperator}) {
		return true
	}
	if !ring.HasAnyCapability(r.Signer, []string{keyring.CapClaim}) {
		return false
	}
	s, ok := table.StateAt(records[:r.Pos], subject)
	return ok && s.Claim != nil && s.Claim.Holder == r.Signer
}

// BudgetCloseValid reports whether a close attempt may close the
// cited reservation: its signer is the reservation's own reserving
// signer (identity, not escalation), or the operator lane at the
// attempt's position — so a raw foreign settle or release never
// closes anyone's reservation (review finding on plan #147).
func BudgetCloseValid(records []*event.Record, c transition.CloseFact, r transition.ReservationFact) bool {
	if c.Signer == r.Signer {
		return true
	}
	if c.Pos < 0 || c.Pos >= len(records) {
		return false
	}
	ring, _, err := keyring.StateAt(records[:c.Pos])
	return err == nil && ring != nil && ring.HasAnyCapability(c.Signer, []string{keyring.CapOperator})
}

// BudgetViewAt derives the subject's budget view over the verified
// prefix with the position-accurate callbacks: the one computation
// admission, seed budget status, and the projections share.
func BudgetViewAt(records []*event.Record, table *transition.Table, subject string, s transition.SubjectState) transition.BudgetView {
	return s.DeriveBudget(
		func(r transition.ReservationFact) bool {
			return ReservationValid(records, table, subject, r)
		},
		func(c transition.CloseFact, r transition.ReservationFact) bool {
			return BudgetCloseValid(records, c, r)
		},
	)
}

// RunStartValid reports whether a folded run.started passed the
// admission boundary at its own position (review findings on the
// task PR: fold presence is never proof of admission): the signer
// held the run lanes there, the cited fence was the subject's ACTIVE
// claim fence there, and the cited reservation exists and passed the
// authoring boundary. The run rule's one-per-fence and
// carries-a-start checks and the executor's Provision gate all share
// this one derivation, so a raw start neither blocks the legitimate
// supervisor, nor launders a settle through, nor provisions an
// unbudgeted workspace.
func RunStartValid(records []*event.Record, table *transition.Table, subject string, s transition.SubjectState, st transition.RunStartFact) bool {
	if st.Pos < 0 || st.Pos >= len(records) {
		return false
	}
	ring, _, err := keyring.StateAt(records[:st.Pos])
	if err != nil || ring == nil ||
		!ring.HasAnyCapability(st.Signer, keyring.AcceptedCapabilities(transition.RunStartedVerb)) {
		return false
	}
	prior, ok := table.StateAt(records[:st.Pos], subject)
	if !ok || prior.Claim == nil || prior.Claim.Fence != st.Fence {
		return false
	}
	for _, r := range s.Reservations {
		if r.Pos == st.Reservation {
			return ReservationValid(records, table, subject, r)
		}
	}
	return false
}

func fenceCitation(payload []byte) (string, bool) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(payload, &m); err != nil {
		return "", false
	}
	raw, ok := m["fence"]
	if !ok {
		return "", false
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		// A non-string fence is a citation that matches nothing.
		return strings.TrimSpace(string(raw)), true
	}
	return s, true
}
