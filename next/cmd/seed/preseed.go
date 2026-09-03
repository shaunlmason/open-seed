package main

// The preseed (plans/os-0d4f2af3.md D1, D2, D5; charter §II.17,
// Appendix D.1): `seed init --preseed seed.json` bootstraps a
// deployment from the one declaration, idempotently — a second run
// appends nothing — and `seed preseed check` is the same comparison
// with no writes, which CI runs. Drift between the file and the chain
// refuses by name; a required member missing from the protected
// surface refuses as an incomplete declaration; init never edits
// history to match a file.

import (
	"crypto/ed25519"
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/shaunlmason/open-seed/next/internal/envelope"
	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/genesis"
	"github.com/shaunlmason/open-seed/next/internal/keyring"
	"github.com/shaunlmason/open-seed/next/internal/lane"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
	"github.com/shaunlmason/open-seed/next/internal/posture"
	"github.com/shaunlmason/open-seed/next/internal/transition"
	"github.com/shaunlmason/open-seed/next/internal/version"
)

// RequiredProtected is the protected surface the charter requires a
// declaration to enumerate (§II.14; next/spec/postures.md "The
// preseed"), as prefixes compared by string: the transition spec and
// every normative table, the admission rules, the standing and
// capability rules, verifier code and rubrics and the sealed-check
// machinery, the curator's gates and the policy stage, the role
// definitions, the check pipeline's own definitions, and the
// declaration itself (always on the surface by construction).
var RequiredProtected = []string{
	"next/spec",
	"next/internal/admit",
	"next/internal/transition",
	"next/internal/keyring",
	"next/internal/verdict",
	"next/internal/seal",
	"next/internal/eval",
	"next/evals",
	"next/internal/curation",
	"next/knowledge/lessons",
	"next/lanes",
	"next/cmd/seed-admit",
	"next/cmd/covergate",
	"Makefile",
	".github/workflows",
	"scripts",
}

// preseedDrift is a declaration the chain contradicts: the field and
// both values.
type preseedDrift struct {
	Field, Declared, Observed string
}

func (d *preseedDrift) Error() string {
	return fmt.Sprintf("preseed drift on %s: the declaration says %q, the chain has %q", d.Field, d.Declared, d.Observed)
}

// preseedIncomplete is a declaration missing something the charter
// requires it to say.
type preseedIncomplete struct{ Detail string }

func (e *preseedIncomplete) Error() string { return "preseed incomplete: " + e.Detail }

// lintPreseed holds the declaration's content to the tables: tiers to
// the vocabulary, lane manifests to the shipped set, the protected
// surface to the required members, the protocol to the register. It
// reads no ledger.
func lintPreseed(cfg *posture.Config, lanesDir string) error {
	if cfg.Protocol != "" {
		known := false
		for _, v := range version.Supported() {
			if v == cfg.Protocol {
				known = true
			}
		}
		if !known {
			return &preseedIncomplete{Detail: fmt.Sprintf("protocol %q is not in this build's register (%s)", cfg.Protocol, strings.Join(version.Supported(), ", "))}
		}
	}
	if cfg.Guardrails != nil {
		for name, g := range cfg.Guardrails.Squads {
			for _, tier := range []string{g.Default, g.MaxAgent} {
				if _, ok := transition.Tier(tier); !ok {
					return &preseedIncomplete{Detail: fmt.Sprintf("guardrails.squads.%s names tier %q, not in the vocabulary (%s)", name, tier, strings.Join(transition.TierOrder(), ", "))}
				}
			}
		}
		for _, f := range cfg.Guardrails.Paths {
			if _, ok := transition.Tier(f.Min); !ok {
				return &preseedIncomplete{Detail: fmt.Sprintf("guardrails.paths %s names tier %q, not in the vocabulary", f.Prefix, f.Min)}
			}
		}
		if cfg.Teams != nil {
			for name := range cfg.Guardrails.Squads {
				found := false
				for _, s := range cfg.Teams.Squads {
					if s.Name == name {
						found = true
					}
				}
				if !found {
					return &preseedIncomplete{Detail: fmt.Sprintf("guardrails.squads.%s is not a declared team", name)}
				}
			}
		}
	}
	if cfg.Teams != nil && lanesDir != "" {
		manifests, err := lane.Load(lanesDir)
		if err != nil {
			return fmt.Errorf("reading lane manifests: %w", err)
		}
		known := map[string]bool{}
		for _, m := range manifests {
			known[m.Lane] = true
		}
		for _, s := range cfg.Teams.Squads {
			for _, l := range s.Lanes {
				if !known[l] {
					return &preseedIncomplete{Detail: fmt.Sprintf("teams.squads.%s runs %q, which is not a manifest under %s", s.Name, l, lanesDir)}
				}
			}
		}
	}
	if cfg.Governance != nil || len(cfg.Protected) > 0 {
		for _, req := range RequiredProtected {
			if !cfg.Protects(req) {
				return &preseedIncomplete{Detail: fmt.Sprintf("protected omits %s, a member the charter requires on the surface (next/spec/postures.md)", req)}
			}
		}
	}
	return nil
}

// comparePreseed holds the chain to the declaration: the governance
// root, the protocol. It returns the versions still to activate (empty
// when the chain is at or past the declared protocol), or drift.
func comparePreseed(cfg *posture.Config, records []*event.Record) ([]string, error) {
	if len(records) == 0 {
		return nil, nil
	}
	payload, err := genesis.Parse(records[0])
	if err != nil {
		return nil, fmt.Errorf("the chain's genesis does not parse: %w", err)
	}
	if cfg.Governance != nil {
		found := false
		for _, rk := range payload.GovernanceRoot {
			if rk.Fingerprint == cfg.Governance.Root {
				found = true
			}
		}
		if !found {
			roots := make([]string, 0, len(payload.GovernanceRoot))
			for _, rk := range payload.GovernanceRoot {
				roots = append(roots, rk.Fingerprint)
			}
			sort.Strings(roots)
			return nil, &preseedDrift{Field: "governance.root", Declared: cfg.Governance.Root, Observed: strings.Join(roots, ",")}
		}
	}
	_, active, err := keyring.StateAt(records)
	if err != nil {
		return nil, err
	}
	if cfg.Protocol == "" {
		return nil, nil
	}
	supported := version.Supported()
	rank := func(v string) int {
		for i, s := range supported {
			if s == v {
				return i
			}
		}
		return -1
	}
	want, have := rank(cfg.Protocol), rank(active)
	if want < 0 {
		return nil, &preseedIncomplete{Detail: fmt.Sprintf("protocol %q is not in this build's register", cfg.Protocol)}
	}
	if have < 0 {
		return nil, &preseedDrift{Field: "protocol", Declared: cfg.Protocol, Observed: active}
	}
	if have > want {
		return nil, &preseedDrift{Field: "protocol", Declared: cfg.Protocol, Observed: active}
	}
	return supported[have+1 : want+1], nil
}

// applyPreseed brings an empty or matching ledger to the declaration:
// genesis under the signer when the ledger is empty, then every
// protocol version up to the declared one, in order. It returns what it
// appended. A ledger that disagrees with the file is drift, and nothing
// is written.
func applyPreseed(store *ledger.Store, cfg *posture.Config, signer ed25519.PrivateKey, extras []ed25519.PublicKey, now time.Time) ([]string, error) {
	var appended []string
	var records []*event.Record
	if err := store.Records(func(pos int, r *event.Record) error { records = append(records, r); return nil }); err != nil {
		return nil, err
	}
	fp, err := event.Fingerprint(signer.Public().(ed25519.PublicKey))
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		if cfg.Governance != nil && cfg.Governance.Root != fp {
			return nil, &preseedDrift{Field: "governance.root", Declared: cfg.Governance.Root, Observed: fp + " (the initializing key)"}
		}
		rec, err := genesis.Init(store, signer, extras, now)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
		appended = append(appended, "system.genesis")
	}
	todo, err := comparePreseed(cfg, records)
	if err != nil {
		return nil, err
	}
	if len(todo) == 0 {
		return appended, nil
	}
	resolve, _, err := genesis.Bootstrap(store)
	if err != nil {
		return nil, err
	}
	_, active, err := keyring.StateAt(records)
	if err != nil {
		return nil, err
	}
	tip, _, err := store.Tip()
	if err != nil {
		return nil, err
	}
	for i, next := range todo {
		ring, _, err := keyring.StateAt(records)
		if err != nil {
			return nil, err
		}
		res := resolve
		if keyring.Applies(active) && ring.Seeded() {
			res = ring.Resolver()
		}
		rec, err := event.Sign(event.Event{
			V: active, TS: now.Add(time.Duration(i+1) * time.Second).UTC().Format(time.RFC3339), Actor: fp,
			Verb: ledger.UpgradeVerb, Subject: "system", Payload: []byte(`{"to": "` + next + `"}`), Prev: tip,
		}, signer)
		if err != nil {
			return nil, err
		}
		if _, err := store.Append(rec, res); err != nil {
			return nil, fmt.Errorf("activating %s: %w", next, err)
		}
		records = append(records, rec)
		tip, err = rec.Event.Hash()
		if err != nil {
			return nil, err
		}
		active = next
		appended = append(appended, ledger.UpgradeVerb+" "+next)
	}
	return appended, nil
}

func preseedFailEnvelope(err error) *envelope.Envelope {
	var drift *preseedDrift
	if errors.As(err, &drift) {
		return envelope.Fail(envelope.ExitDrift, "preseed_drift", err.Error())
	}
	var inc *preseedIncomplete
	if errors.As(err, &inc) {
		return envelope.Fail(envelope.ExitPostureInvalid, "preseed_incomplete", err.Error())
	}
	if errors.Is(err, ledger.ErrNotEmpty) {
		return envelope.Fail(envelope.ExitInvalidTransition, "ledger_not_empty", err.Error())
	}
	return envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error())
}

func runPreseed(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "check" {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "preseed takes the subverb check"), stdout, stderr)
	}
	fs := flag.NewFlagSet("preseed check", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	config := fs.String("config", posture.DeclarationPath, "deployment declaration")
	dir := fs.String("ledger", "", "ledger directory to compare the declaration against (omit to lint the file alone)")
	lanesDir := fs.String("lanes", "next/lanes", "lane manifests the teams block is checked against (empty to skip)")
	if err := fs.Parse(args[1:]); err != nil || fs.NArg() != 0 {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "preseed check [--config <file>] [--ledger <dir>] [--lanes <dir>]"), stdout, stderr)
	}
	cfg, failEnv := loadDeclarationFor(*config)
	if failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	if err := lintPreseed(cfg, *lanesDir); err != nil {
		return render(preseedFailEnvelope(err), stdout, stderr)
	}
	result := map[string]any{
		"config":     *config,
		"protocol":   cfg.Protocol,
		"governance": cfg.Governance != nil,
		"guardrails": cfg.Guardrails != nil,
		"teams":      cfg.Teams != nil,
		"protected":  cfg.ProtectedSurface(),
	}
	if *dir == "" {
		result["ledger"] = nil
		return render(envelope.OK(result), stdout, stderr)
	}
	store, failEnv := openStoreReadOnly(*dir)
	if failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	var records []*event.Record
	if err := store.Records(func(pos int, r *event.Record) error { records = append(records, r); return nil }); err != nil {
		return render(envelope.Fail(envelope.ExitChainInvalid, "chain_invalid", err.Error()), stdout, stderr)
	}
	todo, err := comparePreseed(cfg, records)
	if err != nil {
		return render(stampTip(preseedFailEnvelope(err), len(records)), stdout, stderr)
	}
	result["ledger"] = *dir
	result["pending"] = todo
	if len(todo) > 0 {
		env := envelope.Fail(envelope.ExitDrift, "preseed_drift", fmt.Sprintf("the chain has not activated %s the declaration names; `seed init --preseed` appends the missing activations", strings.Join(todo, ", ")))
		return render(stampTip(env, len(records)), stdout, stderr)
	}
	return render(stampTip(envelope.OK(result), len(records)), stdout, stderr)
}
