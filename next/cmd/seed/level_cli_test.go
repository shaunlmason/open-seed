package main

// The independence levels at the CLI (plans/os-99829835.md AC5, AC6;
// next/spec/verdicts.md "Independence levels"): `verdict render`
// computes the achieved level from the window's declaration, its own
// declared tuple and the folded acceptance, refuses at usage without
// the flags on a tier the record cannot satisfy, refuses level_short
// with a same-family declaration, and writes the level; `verdict
// check` renders it; `reconcile` classifies what raw pushes claim.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/version"
)

func levelTuple(model string) string {
	return fmt.Sprintf(`{"principal": "acme", "harness": "local-worktree/v0", "model": %q, "tool_policy": "default", "environment": "detached-git-worktree"}`, model)
}

// levelLedger is offerLedger upgraded to seed/4, with a driver that
// files a contract at the given tier with a prose or an executable
// gated spec, claims it, reserves, declares a start under the base
// model, and submits: the window every level is computed against.
func levelLedger(t *testing.T) (ld, src, specCommit, rng, priv string, keys map[string]string, drive func(subject, tier string, executable bool) int) {
	t.Helper()
	ld, src, base, specCommit, head, priv, rootKey, keys, _ := offerLedger(t)
	// A sealer, so the tiers that require sealed checks can be
	// rendered: the level is computed after the unsealed refusal, and
	// every tier but trivial requires the seal.
	sealKey, sealPub, sealFP := writeWorkerKey(t, 25)
	keys["sealer"] = sealKey
	for _, step := range [][]string{
		{"actor.enrolled", sealFP, fmt.Sprintf(`{"key": %q, "kind": "agent", "name": "sealer"}`, sealPub)},
		{"actor.granted", sealFP, `{"capability": "sealer"}`},
	} {
		if e, code := runEnv(t, "ledger", "append", "--ledger", ld, "--key", priv,
			"--verb", step[0], "--subject", step[1], "--payload", step[2]); code != 0 {
			t.Fatalf("%s: %d %+v", step[0], code, e)
		}
	}
	checks := writeChecks(t, "# sealed", "true")
	for _, to := range []string{version.Seed2, version.Seed3, version.Seed4} {
		if e, code := runEnv(t, "ledger", "append", "--ledger", ld, "--key", priv,
			"--verb", "system.protocol.upgraded", "--subject", "system", "--payload", `{"to": "`+to+`"}`); code != 0 {
			t.Fatalf("upgrade to %s: %d %+v", to, code, e)
		}
	}
	rng = base + ".." + head
	drive = func(subject, tier string, executable bool) int {
		t.Helper()
		spec := fmt.Sprintf(`{"acceptance": {"ref": "accept.md @ %s", "executable": false}}`, specCommit)
		if executable {
			spec = fmt.Sprintf(`{"acceptance": {"ref": "accept.md @ %s", "executable": true, "gate": "pr/6 @ %s"}}`, specCommit, specCommit)
		}
		rawAppendAt(t, ld, rootKey, version.Seed4, "intent.filed", subject, fmt.Sprintf(`{"intent": "drill", "tier": %q, "budget": "small", "routing": "core"}`, tier))
		rawAppendAt(t, ld, rootKey, version.Seed4, "contract.specified", subject, spec)
		if tier != "trivial" {
			if e, code := runEnv(t, "seal", "create", "--ledger", ld, "--subject", subject, "--repo", src,
				"--checks", checks, "--key", keys["sealer"]); code != 0 {
				t.Fatalf("seal create on %s: %d %+v", subject, code, e)
			}
		}
		fence := rawAppendAt(t, ld, rootKey, version.Seed4, "claim.taken", subject, `{}`)
		res := rawAppendAt(t, ld, rootKey, version.Seed4, "budget.reserve", subject, fmt.Sprintf(`{"amount": "10", "fence": "%d"}`, fence))
		rawAppendAt(t, ld, workerRawKey(21), version.Seed4, "run.started", subject, fmt.Sprintf(`{"fence": "%d", "reservation": "%d", "tuple": %s}`, fence, res, levelTuple("fable/5.1")))
		return rawAppendAt(t, ld, rootKey, version.Seed4, "submission.made", subject, fmt.Sprintf(
			`{"fence": "%d", "packet": {"acceptance": ["%s ok"], "decisions": [], "base": %q, "refs": [], "findings": []}}`, fence, subject, rng))
	}
	return
}

// conformance: AC5 — render computes the level and refuses what the
// tier cannot get; check and the contracts view render it.
func TestVerdictRenderComputesTheLevel(t *testing.T) {
	ld, src, _, _, _, keys, drive := levelLedger(t)
	render := func(subject string, extra ...string) (ledgerEnv, int) {
		return runEnv(t, append([]string{"verdict", "render", "--ledger", ld, "--subject", subject, "--repo", src,
			"--key", keys["verifier"], "--verdict", "pass"}, extra...)...)
	}
	declare := func(model string) []string {
		return []string{"--principal", "acme", "--model", model, "--tool-policy", "default"}
	}

	drive("c-std", "standard", false)
	e, code := render("c-std")
	if code != 0 || e.Result["independence"] != "L1" {
		t.Fatalf("a prose standard contract with no declaration renders L1: %d %+v", code, e)
	}
	if e, code := runEnv(t, "verdict", "check", "--ledger", ld, "--subject", "c-std", "--repo", src, "--key", keys["verifier"]); code != 0 || e.Result["independence"] != "L1" {
		t.Fatalf("verdict check renders the recorded level: %d %+v", code, e)
	}

	drive("c-crit", "critical", false)
	e, code = render("c-crit")
	if code != 64 || e.Error == nil || !strings.Contains(e.Error.Message, "--principal") || !strings.Contains(e.Error.Message, "L2") {
		t.Fatalf("a critical contract with no declaration refuses at usage naming the flags and the requirement: %d %+v", code, e)
	}
	e, code = render("c-crit", declare("fable/9.9")...)
	if code != 17 || e.Error == nil || e.Error.Code != "level_short" || !strings.Contains(e.Error.Message, "critical") {
		t.Fatalf("the same family on a critical contract refuses level_short: %d %+v", code, e)
	}
	e, code = render("c-crit", "--principal", "acme", "--model", "other/1")
	if code != 64 {
		t.Fatalf("a partial declaration refuses at usage: %d %+v", code, e)
	}
	e, code = render("c-crit", declare("other/1")...)
	if code != 0 || e.Result["independence"] != "L2" {
		t.Fatalf("a different model family on a critical contract renders L2: %d %+v", code, e)
	}

	drive("c-exec", "standard", true)
	e, code = render("c-exec")
	if code != 0 || e.Result["independence"] != "L3" {
		t.Fatalf("an executable gated spec renders L3 with no declaration: %d %+v", code, e)
	}

	// The contracts view carries the level beside the verdict.
	out := t.TempDir() + "/out"
	if e, code := runEnv(t, "project", "rebuild", "--ledger", ld, "--out", out); code != 0 {
		t.Fatalf("project rebuild: %d %+v", code, e)
	}
	cur, code := runEnv(t, "project", "current", "--out", out, "--name", "contracts")
	if code != 0 {
		t.Fatalf("project current: %d %+v", code, cur)
	}
	b, err := os.ReadFile(filepath.Join(cur.Result["path"].(string), "contracts.json"))
	if err != nil {
		t.Fatal(err)
	}
	view := string(b)
	if !strings.Contains(view, `"independence": "L3"`) || !strings.Contains(view, `"independence": "L2"`) {
		t.Fatalf("the contracts view shows the recorded levels: %s", view)
	}
}

// conformance: AC6 — reconcile classifies a raw-pushed level the
// records do not support, one short of its tier, and an L3 whose
// receipt does not reproduce, and never a supported claim that meets
// the tier.
func TestReconcileClassifiesUnsupportedLevels(t *testing.T) {
	ld, src, _, _, _, _, drive := levelLedger(t)
	verifier := workerRawKey(24)
	classes := func(subject string) map[string]float64 {
		e, code := runEnv(t, "reconcile", "--ledger", ld, "--repo", src, "--subject", subject)
		if code != 0 {
			t.Fatalf("reconcile %s: %d %+v", subject, code, e)
		}
		return classesOf(t, e)
	}
	verdict := func(sub int, level, receipt, tup string) string {
		body := fmt.Sprintf(`{"verdict": "pass", "receipt": %q, "submission": "%d", "independence": %q`, receipt, sub, level)
		if tup != "" {
			body += `, "tuple": ` + tup
		}
		return body + `}`
	}
	bogus := strings.Repeat("ab", 32)

	sub := drive("c-fake", "standard", false)
	rawAppendAt(t, ld, verifier, version.Seed4, "verdict.rendered", "c-fake", verdict(sub, "L2", bogus, ""))
	if classes("c-fake")["independence_unverified"] != 1 {
		t.Fatal("L2 with no declaration classifies independence_unverified")
	}
	sub = drive("c-same", "standard", false)
	rawAppendAt(t, ld, verifier, version.Seed4, "verdict.rendered", "c-same", verdict(sub, "L2", bogus, levelTuple("fable/6.0")))
	if classes("c-same")["independence_unverified"] != 1 {
		t.Fatal("L2 with a same-family declaration classifies independence_unverified")
	}
	sub = drive("c-short", "critical", false)
	rawAppendAt(t, ld, verifier, version.Seed4, "verdict.rendered", "c-short", verdict(sub, "L1", bogus, ""))
	if classes("c-short")["independence_unverified"] != 1 {
		t.Fatal("a verdict short of its tier classifies independence_unverified")
	}
	// Trivial, so the subject is unsealed: a sealed subject's receipt
	// needs the sealer's key to recompute and is `verdict check
	// --key`'s to judge, so reconcile reproduces the unsealed ones.
	sub = drive("c-l3", "trivial", true)
	rawAppendAt(t, ld, verifier, version.Seed4, "verdict.rendered", "c-l3", verdict(sub, "L3", bogus, ""))
	if classes("c-l3")["independence_unverified"] != 1 {
		t.Fatal("an L3 whose receipt does not reproduce classifies independence_unverified at evidence grade")
	}
	sub = drive("c-ok", "standard", false)
	rawAppendAt(t, ld, verifier, version.Seed4, "verdict.rendered", "c-ok", verdict(sub, "L2", bogus, levelTuple("other/1")))
	if by := classes("c-ok"); by["independence_unverified"] != 0 {
		t.Fatalf("a supported claim that meets the tier classifies nothing: %v", by)
	}
}
