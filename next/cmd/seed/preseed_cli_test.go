package main

// The preseed at the terminal (plans/os-0d4f2af3.md D1, D2, D3, D5):
// one file bootstraps a deployment idempotently, drift refuses by name,
// the file is checked with no ledger, the plan lint holds a file scope
// to the path floors, and the doctor reports the blocks.

import (
	"crypto/ed25519"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/event"
)

func preseedFile(t *testing.T, root, protocol string, extra string) string {
	t.Helper()
	body := `{"posture": "cooperative", "protocol": "` + protocol + `", "governance": {"root": "` + root + `", "owners": ["@root"], "change_process": "pr+owner-review"}, "protected": ["next/spec", "next/internal/admit", "next/internal/transition", "next/internal/keyring", "next/internal/verdict", "next/internal/seal", "next/internal/eval", "next/evals", "next/internal/curation", "next/knowledge/lessons", "next/lanes", "next/cmd/seed-admit", "next/cmd/covergate", "Makefile", ".github/workflows", "scripts"], "guardrails": {"squads": {"core": {"default": "standard", "max_agent": "standard"}}, "paths": [{"prefix": "next/internal/admit", "min": "critical"}]}, "teams": {"squads": [{"name": "core", "lanes": ["implementer", "verifier"]}]}` + extra + `}`
	return writeDeclaration(t, body)
}

// conformance: III.P row 3 — one declarative file bootstraps a
// deployment idempotently and CI-verifiably: the first run writes
// genesis and the declared activations, the second appends nothing,
// and a declaration the chain contradicts refuses `preseed_drift`
// naming the field and both values with the chain untouched.
func TestInitPreseedIsIdempotentAndDriftRefuses(t *testing.T) {
	_, priv, _ := writeKeys(t)
	privBytes, _ := os.ReadFile(priv)
	signer, err := event.ParsePrivateKey(privBytes)
	if err != nil {
		t.Fatal(err)
	}
	fp, _ := event.Fingerprint(signer.Public().(ed25519.PublicKey))
	cfg := preseedFile(t, fp, "seed/2", "")
	ledgerDir := filepath.Join(t.TempDir(), "ledger")

	e, code := runEnv(t, "init", "--ledger", ledgerDir, "--key", priv, "--preseed", cfg, "--lanes", "../../lanes")
	if code != 0 || !e.OK || e.Result["unchanged"] != false || *e.Position != "2" {
		msg := ""
		if e.Error != nil {
			msg = e.Error.Message
		}
		t.Fatalf("the first run writes genesis and two activations: %d %+v %s", code, e, msg)
	}
	appended, _ := e.Result["appended"].([]any)
	if len(appended) != 3 || appended[0] != "system.genesis" || !strings.HasSuffix(appended[2].(string), "seed/2") {
		t.Fatalf("genesis, then seed/1 and seed/2 in order: %v", appended)
	}
	e, code = runEnv(t, "init", "--ledger", ledgerDir, "--key", priv, "--preseed", cfg, "--lanes", "../../lanes")
	if code != 0 || e.Result["unchanged"] != true || *e.Position != "2" {
		t.Fatalf("the second run appends nothing: %d %+v", code, e)
	}
	e, code = runEnv(t, "preseed", "check", "--config", cfg, "--ledger", ledgerDir, "--lanes", "../../lanes")
	if code != 0 || !e.OK || *e.Position != "2" {
		t.Fatalf("check agrees with init: %d %+v", code, e)
	}

	// A higher protocol declared later: check names the pending
	// activation as drift, init appends exactly it.
	higher := preseedFile(t, fp, "seed/3", "")
	e, code = runEnv(t, "preseed", "check", "--config", higher, "--ledger", ledgerDir, "--lanes", "../../lanes")
	if code != 28 || e.Error.Code != "preseed_drift" || !strings.Contains(e.Error.Message, "seed/3") {
		t.Fatalf("a pending activation is drift the check names: %d %+v", code, e)
	}
	e, code = runEnv(t, "init", "--ledger", ledgerDir, "--key", priv, "--preseed", higher, "--lanes", "../../lanes")
	if code != 0 || len(e.Result["appended"].([]any)) != 1 || *e.Position != "3" {
		t.Fatalf("init appends the one missing activation: %d %+v", code, e)
	}

	// A lower protocol than the chain's, and a different root: drift,
	// the chain untouched.
	for name, decl := range map[string]string{
		"lower protocol": preseedFile(t, fp, "seed/1", ""),
		"other root":     preseedFile(t, strings.Repeat("ab", 32), "seed/3", ""),
	} {
		e, code := runEnv(t, "init", "--ledger", ledgerDir, "--key", priv, "--preseed", decl, "--lanes", "../../lanes")
		if code != 28 || e.Error == nil || e.Error.Code != "preseed_drift" || !strings.Contains(e.Error.Message, "the declaration says") {
			t.Fatalf("%s must refuse as drift naming both values: %d %+v", name, code, e)
		}
		if e2, _ := runEnv(t, "ledger", "verify", "--ledger", ledgerDir); e2.Result["count"].(float64) != 4 {
			t.Fatalf("%s: the chain is untouched by a refused preseed: %+v", name, e2)
		}
	}
	// A fresh ledger under a root the key is not: refused before genesis.
	other := preseedFile(t, strings.Repeat("cd", 32), "seed/1", "")
	empty := filepath.Join(t.TempDir(), "ledger")
	if e, code := runEnv(t, "init", "--ledger", empty, "--key", priv, "--preseed", other, "--lanes", "../../lanes"); code != 28 || e.Error.Code != "preseed_drift" {
		t.Fatalf("a root that is not the initializing key refuses: %d %+v", code, e)
	}
	if _, err := os.Stat(filepath.Join(empty, "HEAD")); err == nil {
		if e, _ := runEnv(t, "ledger", "verify", "--ledger", empty); e.OK && e.Result["count"].(float64) != 0 {
			t.Fatal("nothing is written under a refused root")
		}
	}
}

// conformance: III.L rows 1 and 2 — the declaration is checked with no
// ledger: an unknown tier, a lane that is no manifest, a guardrail for
// an undeclared squad, and a protected surface missing a required
// member each refuse by name; the fixture deployment's declaration
// passes, which is what the CI line asserts.
func TestPreseedCheckLintsTheFile(t *testing.T) {
	fp := strings.Repeat("ef", 32)
	if e, code := runEnv(t, "preseed", "check", "--config", preseedFile(t, fp, "seed/4", ""), "--lanes", "../../lanes"); code != 0 || !e.OK || e.Result["ledger"] != nil {
		t.Fatalf("a complete declaration lints clean with no ledger: %d %+v", code, e)
	}
	if e, code := runEnv(t, "preseed", "check", "--config", "../../fixtures/deployment/seed.json", "--lanes", "../../lanes"); code != 0 || !e.OK {
		t.Fatalf("the fixture deployment lints clean: %d %+v", code, e)
	}
	for name, body := range map[string]string{
		"unknown tier":       `{"posture": "cooperative", "guardrails": {"squads": {"core": {"default": "huge", "max_agent": "standard"}}}, "teams": {"squads": [{"name": "core", "lanes": ["implementer"]}]}}`,
		"no such manifest":   `{"posture": "cooperative", "teams": {"squads": [{"name": "core", "lanes": ["wizard"]}]}}`,
		"undeclared squad":   `{"posture": "cooperative", "guardrails": {"squads": {"ops": {"default": "trivial", "max_agent": "trivial"}}}, "teams": {"squads": [{"name": "core", "lanes": ["implementer"]}]}}`,
		"incomplete surface": `{"posture": "cooperative", "protected": ["Makefile"]}`,
		"unknown protocol":   `{"posture": "cooperative", "protocol": "seed/9"}`,
	} {
		e, code := runEnv(t, "preseed", "check", "--config", writeDeclaration(t, body), "--lanes", "../../lanes")
		if code != 13 || e.Error == nil || e.Error.Code != "preseed_incomplete" {
			t.Errorf("%s must refuse preseed_incomplete, got %d %+v", name, code, e)
		}
	}
	for name, body := range map[string]string{
		"bad change process":  `{"posture": "cooperative", "governance": {"root": "x", "change_process": "vibes"}}`,
		"duplicate squad":     `{"posture": "cooperative", "teams": {"squads": [{"name": "core", "lanes": ["implementer"]}, {"name": "core", "lanes": ["verifier"]}]}}`,
		"squad without lanes": `{"posture": "cooperative", "teams": {"squads": [{"name": "core", "lanes": []}]}}`,
	} {
		e, code := runEnv(t, "preseed", "check", "--config", writeDeclaration(t, body), "--lanes", "../../lanes")
		if code != 13 || e.Error == nil || e.Error.Code != "posture_invalid" {
			t.Errorf("%s must refuse at parse, got %d %+v", name, code, e)
		}
	}
	if e, code := runEnv(t, "preseed", "audit"); code != 64 || e.Error.Code != "usage" {
		t.Fatalf("unknown subverb: %d %+v", code, e)
	}
}

// conformance: III.L row 1 — tiers per path: a plan whose file scope
// touches a floored prefix at a tier below the floor refuses
// `under_tiered`; at or above the floor it passes; with no declaration
// the lint is what it was.
func TestPlanLintHoldsTheScopeToThePathFloors(t *testing.T) {
	plan := filepath.Join(t.TempDir(), "plan.md")
	body := "# Plan: touch the boundary\n\n## Acceptance Criteria\n\n**Boundary set (new, shown working):**\n\n- the rule refuses\n\n**Retention set (existing, shown unharmed):**\n\n- every suite passes\n\n## Validation Commands\n\n- Boundary: go test ./internal/admit/...\n- Retention: make check\n\n## Expected diff shape\n\nOne rule, one drill.\n\n## File Scope\n\n- `next/internal/admit/admit.go`\n- `next/docs/progress.md`\n"
	if err := os.WriteFile(plan, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := preseedFile(t, strings.Repeat("ab", 32), "seed/4", "")
	e, code := runEnv(t, "plan", "lint", plan)
	if code != 0 || !e.OK {
		t.Fatalf("the plan lints as before with no declaration: %d %+v", code, e)
	}
	e, code = runEnv(t, "plan", "lint", plan, "--config", cfg, "--tier", "standard")
	if code != 18 || e.Error == nil || e.Error.Code != "under_tiered" || !strings.Contains(e.Error.Message, "next/internal/admit/admit.go needs critical") {
		t.Fatalf("below the floor refuses naming the path and tiers: %d %+v", code, e)
	}
	e, code = runEnv(t, "plan", "lint", plan, "--config", cfg, "--tier", "critical")
	if code != 0 || !e.OK || e.Result["tier"] != "critical" {
		t.Fatalf("at the floor passes: %d %+v", code, e)
	}
	if e, code := runEnv(t, "plan", "lint", plan, "--config", cfg); code != 64 || e.Error.Code != "usage" {
		t.Fatalf("--config needs --tier: %d %+v", code, e)
	}
}

// The doctor reports every preseed block as declared or undeclared.
func TestDoctorReportsThePreseedBlocks(t *testing.T) {
	cfg := preseedFile(t, strings.Repeat("ab", 32), "seed/4", "")
	e, code, _ := runEnvErr(t, "doctor", "--config", cfg)
	if code != 0 || !e.OK {
		t.Fatalf("doctor: %d %+v", code, e)
	}
	pre, _ := e.Result["preseed"].(map[string]any)
	if pre["protocol"] != "seed/4" || pre["governance"] != true || pre["guardrails"] != true || pre["teams"] != true {
		t.Fatalf("the doctor reports the blocks: %+v", e.Result)
	}
	var probe map[string]any
	_ = json.Unmarshal([]byte(`{}`), &probe)
}
