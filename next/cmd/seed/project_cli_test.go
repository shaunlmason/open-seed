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
	t.Cleanup(func() {
		// Published trees are locked (0555 directories); unlock before
		// testing's own TempDir cleanup so RemoveAll succeeds on an
		// unprivileged runner.
		_ = filepath.WalkDir(out, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				_ = os.Chmod(p, 0o755)
			}
			return nil
		})
	})
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
	if !ok || len(list) != 5 {
		t.Fatalf("the result must list all five registered projections, got %+v", e.Result)
	}
	row := list[0].(map[string]any)
	if row["name"] != "roster" || row["position"] != "3" {
		t.Fatalf("roster row wrong: %+v", row)
	}
	names := map[string]bool{}
	for _, r := range list {
		names[r.(map[string]any)["name"].(string)] = true
	}
	for _, want := range []string{"roster", "contracts", "queue", "actors", "report"} {
		if !names[want] {
			t.Fatalf("registry must include %s: %+v", want, names)
		}
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

	// The consumer verb (plans/os-fecfb3f7.md step 5): resolves the
	// published build and stamps the envelope with the stamp's count
	// verbatim (the rebuild envelope above stamped the tip index "2";
	// the same build's stamp reads 3 — both conventions on one
	// fixture, per next/spec/projections.md).
	cur2, code := runEnv(t, "project", "current", "--out", out, "--name", "roster")
	if code != 0 || !cur2.OK || cur2.Position == nil || *cur2.Position != "3" {
		t.Fatalf("current must report the stamp count verbatim: %d %+v", code, cur2)
	}
	if cur2.Result["tip"] == "" || cur2.Result["path"] == "" {
		t.Fatalf("current must report tip and path: %+v", cur2.Result)
	}
	if _, err := os.Stat(filepath.Join(cur2.Result["path"].(string), "roster.json")); err != nil {
		t.Fatalf("the reported path must hold the view: %v", err)
	}
	if _, code := runEnv(t, "project", "current", "--out", out, "--name", "roster", "--min-position", "3"); code != 0 {
		t.Fatal("a satisfied minimum position must pass")
	}
	if e, code := runEnv(t, "project", "current", "--out", out, "--name", "roster", "--min-position", "4"); code != 15 ||
		e.Error == nil || !strings.Contains(e.Error.Message, "3") || !strings.Contains(e.Error.Message, "4") {
		t.Fatalf("a stale stamp must refuse with exit 15 naming both positions, got %d %+v", code, e)
	}
	if cur2.Result["version"] != "1" {
		t.Fatalf("current must report the derivation version: %+v", cur2.Result)
	}
	if _, code := runEnv(t, "project", "current", "--out", out, "--name", "nonesuch"); code != 4 {
		t.Fatal("an unknown projection must refuse not_found")
	}
	if _, code := runEnv(t, "project", "current", "--out", out, "--name", "../roster"); code != 4 {
		t.Fatal("a traversal name is outside the registry and must refuse not_found")
	}
	// Exit-code discipline for damaged layouts (review finding on
	// #111): a registered name with nothing published is not_found; a
	// layout that exists but cannot resolve is an operational failure,
	// unavailable.
	if _, code := runEnv(t, "project", "current", "--out", t.TempDir(), "--name", "roster"); code != 4 {
		t.Fatal("a registered projection with nothing published must refuse not_found")
	}
	broken := t.TempDir()
	if err := os.MkdirAll(filepath.Join(broken, "roster"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(broken, "roster", "CURRENT"), []byte("\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if e2, code := runEnv(t, "project", "current", "--out", broken, "--name", "roster"); code != 5 || e2.Error == nil {
		t.Fatalf("an empty CURRENT pointer is a damaged layout and must refuse unavailable, got %d %+v", code, e2)
	}
	if _, code := runEnv(t, "project", "current", "--out", out); code != 64 {
		t.Fatal("a missing --name is a usage error")
	}
	if b, err := os.ReadFile(filepath.Join(out, "roster", "CURRENT")); err != nil || strings.TrimSpace(string(b)) != strings.TrimSpace(string(cur)) {
		t.Fatal("the consumer verb's refusals must leave the published layout untouched")
	}
}
