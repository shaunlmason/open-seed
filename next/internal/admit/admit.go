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
	"github.com/shaunlmason/open-seed/next/internal/ledger"
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
	supported := map[string]bool{version.Protocol: true}
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
	return &Context{
		Count:     rep.Count,
		Tip:       rep.Tip,
		Active:    rep.ActiveVersion,
		Halt:      halt.StateAt(records),
		Resolve:   resolve,
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
