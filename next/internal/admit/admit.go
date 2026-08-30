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
	"fmt"
	"strings"

	"github.com/shaunlmason/open-seed/next/internal/classify"
	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/genesis"
	"github.com/shaunlmason/open-seed/next/internal/halt"
	"github.com/shaunlmason/open-seed/next/internal/keyring"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
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
			if s, ok := c.Lifecycle.State(rec.Event.Subject); ok {
				current = s.State
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
