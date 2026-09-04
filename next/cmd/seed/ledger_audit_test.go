package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/history"
	"github.com/shaunlmason/open-seed/next/internal/version"
)

// auditLedger initialises a ledger and raw-appends the verbs for one
// subject with the governance root's key through the library, past
// admission on purpose (the dev tool self-validates, as the
// cooperative posture must): the audit reads what the chain holds,
// and a bar is a claim about the chain, not about what the boundary
// would have admitted. The root signs so the chain still verifies.
func auditLedger(t *testing.T, verbs ...string) (string, string) {
	t.Helper()
	dir, priv, _ := writeKeys(t)
	ld := filepath.Join(dir, "ledger")
	if e, code := runEnv(t, "init", "--ledger", ld, "--key", priv); code != 0 || !e.OK {
		t.Fatalf("init: %d %+v", code, e)
	}
	// The chain speaks seed/1 before any lifecycle record, as every
	// deployment does; the upgrade goes through the dev tool.
	if e, code := runEnv(t, "ledger", "append", "--ledger", ld, "--key", priv, "--verb", "system.protocol.upgraded", "--subject", "system", "--payload", `{"to": "`+version.Seed1+`"}`); code != 0 || !e.OK {
		t.Fatalf("upgrade: %d %+v", code, e)
	}
	for _, v := range verbs {
		rawAppend(t, ld, fixturePriv(t), v, "c-1", `{}`)
	}
	return ld, priv
}

// happyPath drives one subject to a deliberate exit. Every record is
// raw-appended with a {} payload, so its run.started cites no
// reservation the fold can read and the unreserved-spend bar names it
// (plans/os-88df7ab2.md D7): a start nothing fenced is spend nothing
// fenced, whatever verbs precede it. The covered arm is
// TestAdmittedChainAuditsCleanThroughTheVerb, which audits a chain
// admission actually took.
var happyPath = []string{"intent.filed", "contract.specified", "offer.published", "claim.taken", "budget.reserve", "run.started", "submission.made"}

// conformance: III.R row 5 — the five-bar audit over any ledger
// (plans/os-7599c27d.md AC1, AC2). A subject driven cleanly to a
// deliberate exit audits clean, every list empty, stamped at the tip;
// each plantable violation refuses 28 audit_violated naming its bar
// and the record.
func TestLedgerAuditCleanAndEachBar(t *testing.T) {
	ld, _ := auditLedger(t, happyPath...)
	e, code := runEnv(t, "ledger", "audit", "--ledger", ld)
	// The shape-reading bars stay quiet on this chain; the
	// unreserved-spend bar names its unfenced start (D7), so the verb
	// refuses and the refusal names that bar alone.
	if code != 28 || e.Error == nil || e.Error.Code != "audit_violated" {
		t.Fatalf("a raw start that cites no reservation is refused: %d %+v", code, e)
	}
	if !strings.Contains(e.Error.Message, "unreserved_spend [c-1]") {
		t.Fatalf("the refusal names the unfenced start: %q", e.Error.Message)
	}
	for _, bar := range []string{"chain_violations", "lost_updates", "silent_abandonments", "guardrail_breaches"} {
		if strings.Contains(e.Error.Message, bar) {
			t.Errorf("%s must stay quiet on this chain: %q", bar, e.Error.Message)
		}
	}
	if e.Position == nil || *e.Position != "8" {
		t.Fatalf("the audit is stamped at the tip it audited: %+v", e)
	}

	cases := []struct {
		name  string
		verbs []string
		bar   string
	}{
		{"an unclosed window", []string{"intent.filed", "contract.specified", "offer.published", "claim.taken", "budget.reserve"}, "silent_abandonments"},
		{"an unreserved run", []string{"intent.filed", "contract.specified", "offer.published", "claim.taken", "run.started", "submission.made"}, "unreserved_spend"},
		// The guardrail bar's case is the sealed author claiming the
		// subject it sealed, which admission refuses; an unoffered
		// claim is not a breach, because admission takes one
		// (plans/os-aaec6a3c.md D1). The raw appends here carry no
		// readable seal, so that arm is drilled at the library in
		// internal/simulate, and this table keeps the bars a raw chain
		// can show.
		{"an illegal transition", []string{"claim.taken", "intent.filed"}, "chain_violations"},
	}
	for _, c := range cases {
		ld, _ := auditLedger(t, c.verbs...)
		e, code := runEnv(t, "ledger", "audit", "--ledger", ld)
		if code != 28 || e.Error == nil || e.Error.Code != "audit_violated" {
			t.Fatalf("%s: must refuse 28 audit_violated, got %d %+v", c.name, code, e)
		}
		if !strings.Contains(e.Error.Message, c.bar) {
			t.Errorf("%s: the message names the bar %s: %q", c.name, c.bar, e.Error.Message)
		}
		if c.bar != "chain_violations" && !strings.Contains(e.Error.Message, "c-1") {
			t.Errorf("%s: the message names the record: %q", c.name, e.Error.Message)
		}
		if e.Position == nil {
			t.Errorf("%s: the refusal is stamped at the tip audited", c.name)
		}
	}
}

// conformance: plans/os-7599c27d.md AC3 — verification comes first: a
// corrupted segment and a ledger with no genesis both refuse
// chain_invalid with no audit fields, and a usage slip is a usage
// refusal.
func TestLedgerAuditVerifiesBeforeAnyBar(t *testing.T) {
	ld, _ := auditLedger(t, "intent.filed")
	segs, err := filepath.Glob(filepath.Join(ld, "segments", "*.jsonl"))
	if err != nil || len(segs) == 0 {
		t.Fatalf("segments: %v %v", segs, err)
	}
	b, err := os.ReadFile(segs[0])
	if err != nil {
		t.Fatal(err)
	}
	corrupted := strings.Replace(string(b), `"subject":"c-1"`, `"subject":"c-9"`, 1)
	if corrupted == string(b) {
		t.Fatal("corruption did not apply")
	}
	if err := os.WriteFile(segs[0], []byte(corrupted), 0o644); err != nil {
		t.Fatal(err)
	}
	e, code := runEnv(t, "ledger", "audit", "--ledger", ld)
	if code != 8 || e.Error == nil || e.Error.Code != "chain_invalid" || e.Result != nil {
		t.Fatalf("a corrupted chain refuses chain_invalid before any bar: %d %+v", code, e)
	}
	empty := filepath.Join(t.TempDir(), "empty")
	e, code = runEnv(t, "ledger", "audit", "--ledger", empty)
	if code != 8 || e.Error == nil || e.Error.Code != "chain_invalid" {
		t.Fatalf("a ledger with no genesis refuses chain_invalid: %d %+v", code, e)
	}
	if _, code = runEnv(t, "ledger", "audit"); code != 64 {
		t.Fatalf("audit without --ledger is a usage refusal, got %d", code)
	}
	if e, code := runEnv(t, "ledger", "audit", "--ledger", ld, "--supported", "seed/9"); code != 10 || e.Error == nil {
		t.Fatalf("the supported-version discipline is verify's: %d %+v", code, e)
	}
}

// conformance: plans/os-7599c27d.md D6 — a refusal naming several
// records names them in one order on every run. The audit fills the
// silent-abandonment list from map iteration; three windows left
// open, appended in reverse, come back sorted five times over.
func TestLedgerAuditRefusalOrderIsDeterministic(t *testing.T) {
	dir, priv, _ := writeKeys(t)
	ld := filepath.Join(dir, "ledger")
	if e, code := runEnv(t, "init", "--ledger", ld, "--key", priv); code != 0 || !e.OK {
		t.Fatalf("init: %d %+v", code, e)
	}
	if e, code := runEnv(t, "ledger", "append", "--ledger", ld, "--key", priv, "--verb", "system.protocol.upgraded", "--subject", "system", "--payload", `{"to": "`+version.Seed1+`"}`); code != 0 || !e.OK {
		t.Fatalf("upgrade: %d %+v", code, e)
	}
	for _, subject := range []string{"c-3", "c-2", "c-1"} {
		for _, v := range []string{"intent.filed", "contract.specified", "offer.published", "claim.taken"} {
			rawAppend(t, ld, fixturePriv(t), v, subject, `{}`)
		}
	}
	for i := 0; i < 5; i++ {
		e, code := runEnv(t, "ledger", "audit", "--ledger", ld)
		if code != 28 || e.Error == nil || e.Error.Code != "audit_violated" {
			t.Fatalf("run %d: must refuse 28 audit_violated, got %d %+v", i, code, e)
		}
		msg := e.Error.Message
		if !strings.Contains(msg, "silent_abandonments") || !(strings.Index(msg, "c-1") < strings.Index(msg, "c-2") && strings.Index(msg, "c-2") < strings.Index(msg, "c-3")) {
			t.Fatalf("run %d: the records are named in one order: %q", i, msg)
		}
	}
}

// conformance: plans/os-88df7ab2.md D1, D7 — the covered arm through
// the verb. A chain admission actually took carries a fence and a
// cited reservation on every start, so the unreserved-spend bar names
// nobody. The guardrail bar does name subjects here, because the
// history stages a claim without publishing an offer and that bar
// requires one while admission does not; the disagreement is carded
// on its own and this drill asserts only the bar this card moves.
func TestAdmittedChainAuditsCleanThroughTheVerb(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ledger")
	if _, err := history.Generate(history.Spec{Seed: 11, Contracts: 2, Dir: dir}); err != nil {
		t.Fatalf("generating an admitted chain: %v", err)
	}
	e, code := runEnv(t, "ledger", "audit", "--ledger", dir)
	if code == 0 {
		if list, ok := e.Result["unreserved_spend"].([]any); !ok || len(list) != 0 {
			t.Fatalf("an admitted chain's runs are fenced: %v", e.Result["unreserved_spend"])
		}
		return
	}
	if e.Error == nil || e.Error.Code != "audit_violated" {
		t.Fatalf("an admitted chain audits or refuses by a bar, not by an error: %d %+v", code, e)
	}
	if strings.Contains(e.Error.Message, "unreserved_spend") {
		t.Fatalf("an admitted chain's runs cite open reservations: %q", e.Error.Message)
	}
}
