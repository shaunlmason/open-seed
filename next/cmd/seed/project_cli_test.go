package main

// The rebuild verb end-to-end (plans/os-4d5cacff.md step 3;
// conformance III.D): one command, stamped result, published layout,
// and the refusal exits.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/version"
)

func TestProjectRebuildCLI(t *testing.T) {
	dir, priv, _ := writeKeys(t)
	ld := filepath.Join(dir, "ledger")
	if _, code := runEnv(t, "init", "--ledger", ld, "--key", priv); code != 0 {
		t.Fatal("init failed")
	}
	_, wpub, wfp := writeWorkerKey(t, 7)
	if e, code := runEnv(t, "ledger", "append", "--ledger", ld, "--key", priv,
		"--verb", "system.protocol.upgraded", "--subject", "system", "--payload", `{"to": "`+version.Seed1+`"}`); code != 0 {
		t.Fatalf("upgrade failed: %d %+v", code, e)
	}
	if e, code := runEnv(t, "ledger", "append", "--ledger", ld, "--key", priv,
		"--verb", "actor.enrolled", "--subject", wfp, "--payload", enrollArg(wpub)); code != 0 {
		t.Fatalf("enrollment failed: %d %+v", code, e)
	}

	out := filepath.Join(t.TempDir(), "proj")
	e, code := runEnv(t, "project", "rebuild", "--ledger", ld, "--out", out)
	if code != 0 || !e.OK {
		t.Fatalf("rebuild failed: %d %+v", code, e)
	}
	// stampTip stamps the tip's zero-based position (count-1), the
	// CLI-wide convention; the projection stamp itself carries the
	// count.
	if e.Position == nil || *e.Position != "2" {
		t.Fatalf("the envelope must stamp the tip position, got %+v", e.Position)
	}
	list, ok := e.Result["projections"].([]any)
	if !ok || len(list) != 1 {
		t.Fatalf("the result must list the rebuilt projections, got %+v", e.Result)
	}
	row := list[0].(map[string]any)
	if row["name"] != "roster" || row["position"] != "3" {
		t.Fatalf("roster row wrong: %+v", row)
	}
	cur, err := os.ReadFile(filepath.Join(out, "roster", "CURRENT"))
	if err != nil {
		t.Fatal(err)
	}
	build := filepath.Join(out, "roster", "builds", strings.TrimSpace(string(cur)))
	for _, f := range []string{"roster.json", "projection.json"} {
		if _, err := os.Stat(filepath.Join(build, f)); err != nil {
			t.Fatalf("the published build must carry %s: %v", f, err)
		}
	}

	// Refusals: usage, overlap (both directions), unreadable ledger.
	if _, code := runEnv(t, "project", "rebuild"); code != 64 {
		t.Fatal("a missing --ledger is a usage error")
	}
	if e, code := runEnv(t, "project", "rebuild", "--ledger", ld, "--out", filepath.Join(ld, "p")); code != 64 || e.Error == nil || !strings.Contains(e.Error.Message, "overlap") {
		t.Fatalf("an output inside the ledger must refuse as usage, got %d %+v", code, e)
	}
	if _, code := runEnv(t, "project", "rebuild", "--ledger", ld, "--out", dir); code != 64 {
		t.Fatal("a ledger inside the output must refuse as usage")
	}
	if _, err := os.Stat(filepath.Join(ld, "p")); !os.IsNotExist(err) {
		t.Fatal("the overlap refusal must create nothing")
	}
	if e, code := runEnv(t, "project", "rebuild", "--ledger", t.TempDir(), "--out", filepath.Join(t.TempDir(), "x")); code != 5 || e.Error == nil {
		t.Fatalf("a directory that is not a ledger must refuse unavailable, got %d %+v", code, e)
	}
	if _, code := runEnv(t, "project", "nope"); code != 64 {
		t.Fatal("an unknown subverb is a usage error")
	}
}
