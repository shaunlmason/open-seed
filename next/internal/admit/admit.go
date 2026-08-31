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
			if s.PriorClaimants[rec.Event.Actor] {
				return &NotIndependentError{Subject: subject, Actor: rec.Event.Actor, Role: "a claimant, past or present"}
			}
			if rec.Event.Actor == s.Submission.Signer {
				return &NotIndependentError{Subject: subject, Actor: rec.Event.Actor, Role: "the bound submission's signer"}
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
	signer := s.Verdict.Signer
	if !c.Keyring.HasAnyCapability(signer, keyring.AcceptedCapabilities(transition.VerdictRenderedVerb)) {
		return &transition.ChainError{Subject: subject, Verb: verb,
			Reason: fmt.Sprintf("the cited verdict at position %d was signed by %s, which holds no verdict grant — a raw-pushed verdict cannot be laundered through the admitted chain", s.Verdict.Pos, signer)}
	}
	if s.PriorClaimants[signer] || (s.Submission != nil && signer == s.Submission.Signer) {
		return &transition.ChainError{Subject: subject, Verb: verb,
			Reason: fmt.Sprintf("the cited verdict at position %d was signed by implementing key %s — L1 independence is not launderable through the admitted chain", s.Verdict.Pos, signer)}
	}
	return nil
}

// fenceCitation extracts the payload's fence field: the string form
// of the cited claim position, absent when the payload carries none.
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
