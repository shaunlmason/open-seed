package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/posture"
)

func writePosture(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "seed.json")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func runDoctorEnv(t *testing.T, args ...string) (ledgerEnv, int, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := run(append([]string{"doctor"}, args...), &out, &errOut)
	var e ledgerEnv
	if err := json.Unmarshal(out.Bytes(), &e); err != nil {
		t.Fatalf("not an envelope: %v\n%s", err, out.String())
	}
	return e, code, errOut.String()
}

// conformance: III posture item — the preflight tool states the posture
// and, for cooperative, the named consequence verbatim and in plain
// words in front of the operator.
func TestDoctorStatesPostures(t *testing.T) {
	e, code, errOut := runDoctorEnv(t, "--config", writePosture(t, `{"posture": "cooperative"}`))
	if code != 0 || !e.OK || e.Result["posture"] != "cooperative" || e.Result["enforced"] != false {
		t.Fatalf("cooperative doctor failed: %d %+v", code, e)
	}
	if e.Result["consequence"] != posture.Consequence {
		t.Fatalf("the consequence must ride verbatim, got %q", e.Result["consequence"])
	}
	for _, phrase := range []string{"does not hold", "advisory against a hostile credential"} {
		if !strings.Contains(errOut, phrase) {
			t.Fatalf("the operator-facing output must say %q, got %q", phrase, errOut)
		}
	}

	e, code, errOut = runDoctorEnv(t, "--config", writePosture(t, `{"posture": "enforced-self-hosted"}`))
	if code != 0 || e.Result["posture"] != "enforced-self-hosted" || e.Result["enforced"] != true {
		t.Fatalf("enforced doctor failed: %d %+v", code, e)
	}
	if _, ok := e.Result["consequence"]; ok || errOut != "" {
		t.Fatalf("enforced posture carries no cooperative consequence, got %+v %q", e.Result, errOut)
	}

	// The third posture reports its deployment (plans/os-5c8a312c.md
	// D7): endpoint, identity and the ledger branch, and no gap sentence.
	e, code, errOut = runDoctorEnv(t, "--config", writePosture(t, `{"posture": "enforced-forge-hosted", "admission": {"endpoint": "http://127.0.0.1:1", "identity": "seed-admission[bot]", "checks": ["check"], "reviews": 1, "owners": ["@root"]}}`))
	if code != 0 || e.Result["enforced"] != true {
		t.Fatalf("forge-hosted doctor failed: %d %+v", code, e)
	}
	adm, _ := e.Result["admission"].(map[string]any)
	if adm["endpoint"] != "http://127.0.0.1:1" || adm["identity"] != "seed-admission[bot]" || adm["ledger_ref"] != posture.DefaultLedgerRef {
		t.Fatalf("the doctor must report the admission block, got %+v", e.Result)
	}
	if _, ok := e.Result["gap"]; ok {
		t.Fatalf("the forge-hosted gap sentence retired with the gap, got %+v", e.Result)
	}
	for _, phrase := range []string{"proposals go to http://127.0.0.1:1", posture.DefaultLedgerRef, "seed-admission[bot] alone"} {
		if !strings.Contains(errOut, phrase) {
			t.Fatalf("the operator-facing output must say %q, got %q", phrase, errOut)
		}
	}
	// Without its block the posture is a declaration that names a
	// service nothing can reach: refused at 13 and named.
	e, code, _ = runDoctorEnv(t, "--config", writePosture(t, `{"posture": "enforced-forge-hosted"}`))
	if code != 13 || e.Error == nil || e.Error.Code != "posture_invalid" || !strings.Contains(e.Error.Message, "admission block") {
		t.Fatalf("a forge-hosted declaration without its block must exit 13 naming the block, got %d %+v", code, e)
	}
}

func TestDoctorRefusals(t *testing.T) {
	e, code, _ := runDoctorEnv(t, "--config", filepath.Join(t.TempDir(), "absent.json"))
	if code != 4 || e.Error == nil || e.Error.Code != "posture_undeclared" || !strings.Contains(e.Error.Message, "MUST declare") {
		t.Fatalf("undeclared deployment must exit 4 with the charter's wording, got %d %+v", code, e)
	}
	for name, content := range map[string]string{
		"unknown posture": `{"posture": "anarchy"}`,
		"unknown field":   `{"posture": "cooperative", "mode": "yolo"}`,
	} {
		e, code, _ := runDoctorEnv(t, "--config", writePosture(t, content))
		if code != 13 || e.Error == nil || e.Error.Code != "posture_invalid" {
			t.Fatalf("%s must exit 13 posture_invalid, got %d %+v", name, code, e)
		}
		if !strings.Contains(e.Error.Message, "valid postures") {
			t.Fatalf("%s must name the valid postures, got %+v", name, e.Error)
		}
	}
	// A config that exists but cannot be read (here: a directory) is an
	// operational failure at exit 66, never a content judgment.
	e, code, _ = runDoctorEnv(t, "--config", t.TempDir())
	if code != 66 || e.Error == nil || e.Error.Code != "posture_unreadable" {
		t.Fatalf("an unreadable config must exit 66 posture_unreadable, got %d %+v", code, e)
	}
	if _, code, _ := runDoctorEnv(t, "extra-arg"); code != 64 {
		t.Fatal("stray arguments are a usage error")
	}
}
