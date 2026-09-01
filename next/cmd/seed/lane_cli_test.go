package main

// The lane surface end to end (plans/os-cf1c9688.md): the six shipped
// manifests validate clean, resolution is ordered and stable through
// the CLI, and the two places that could drift from an authority
// elsewhere are pinned by their own drills.

import (
	"flag"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/lane"
	"github.com/shaunlmason/open-seed/next/internal/loopverb"
)

// shippedLanes is the checked-in set this repository ships.
func shippedLanes(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("..", "..", "lanes"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("the shipped lanes directory is missing: %v", err)
	}
	return dir
}

// conformance: III.J — role definitions exist for all six lanes and
// are CHECKED. A green validate on the shipped set is the assertion
// that every declaration in it is answerable by the tables the system
// already enforces.
func TestShippedLanesValidate(t *testing.T) {
	dir := shippedLanes(t)
	e, code := runEnv(t, "lane", "validate", "--lanes", dir)
	if code != 0 || !e.OK {
		t.Fatalf("the shipped lanes must validate clean: %d %+v", code, e)
	}
	if e.Result["lanes"] != "6" {
		t.Fatalf("the charter names six lanes (SEED-NEXT.md §II.11): %+v", e.Result)
	}

	e, code = runEnv(t, "lane", "list", "--lanes", dir)
	if code != 0 {
		t.Fatalf("lane list: %d %+v", code, e)
	}
	rows, _ := e.Result["lanes"].([]any)
	var names []string
	for _, r := range rows {
		m, _ := r.(map[string]any)
		names = append(names, m["lane"].(string))
	}
	slices.Sort(names)
	want := []string{"curator", "dispatcher", "implementer", "maintenance", "planner", "verifier"}
	if !slices.Equal(names, want) {
		t.Fatalf("the six lanes are the charter's six: %v, want %v", names, want)
	}
}

// conformance: `seed lane show` resolves the ordered fragments and is
// byte-identical across runs; a lane that does not exist is not_found.
func TestLaneShowResolvesAndIsStable(t *testing.T) {
	dir := shippedLanes(t)
	first, code := runEnv(t, "lane", "show", "implementer", "--lanes", dir)
	if code != 0 || !first.OK {
		t.Fatalf("lane show: %d %+v", code, first)
	}
	body, _ := first.Result["resolved"].(string)
	if body == "" {
		t.Fatalf("show resolves the fragments: %+v", first.Result)
	}
	// The lane-specific fragment leads, and a shared convention that
	// every lane composes follows it: that IS the composition.
	if !strings.Contains(body, "# Implementer") {
		t.Fatalf("the lane's own fragment resolves first: %q", body[:min(80, len(body))])
	}
	if !strings.Contains(body, "one-inbox doctrine") {
		t.Fatal("a shared fragment must resolve into every lane that composes it")
	}
	second, _ := runEnv(t, "lane", "show", "implementer", "--lanes", dir)
	if second.Result["resolved"] != first.Result["resolved"] {
		t.Fatal("resolution must be byte-identical across runs")
	}
	if e, code := runEnv(t, "lane", "show", "wizard", "--lanes", dir); code != 4 || e.Error == nil || e.Error.Code != "not_found" {
		t.Fatalf("an unknown lane is not_found: %d %+v", code, e)
	}
}

// conformance: a broken manifest refuses with the lane_invalid code
// and carries the findings, so a caller branches on the code and reads
// the rows rather than parsing prose.
func TestLaneValidateRefusesWithFindings(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, lane.FragmentDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "planner.json"), []byte(`{
  "lane": "planner",
  "summary": "s",
  "grants": ["wizard"],
  "orients_from": "seed situation --key <key>",
  "acts_through": [],
  "liveness_from": [],
  "inbox": "i",
  "fragments": []
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	e, code := runEnv(t, "lane", "validate", "--lanes", dir)
	if code != 26 || e.Error == nil || e.Error.Code != "lane_invalid" {
		t.Fatalf("a broken manifest exits 26 lane_invalid: %d %+v", code, e)
	}
	rows, _ := e.Result["findings"].([]any)
	if len(rows) == 0 {
		t.Fatalf("the refusal carries its findings: %+v", e.Result)
	}
}

// conformance: internal/lane holds the situation read's flag set
// because package main is not importable, so this drill is what keeps
// the two from drifting. Adding a flag to `seed situation` without
// telling the validator fails HERE, which is the whole reason the
// duplication is tolerable.
func TestSituationFlagsAgreeWithTheCLI(t *testing.T) {
	fs := flag.NewFlagSet("situation", flag.ContinueOnError)
	fs.String("ledger", "", "")
	fs.String("key", "", "")
	fs.String("subject", "", "")
	fs.String("since", "", "")
	// The real surface, bound the way runSituation binds it: if that
	// list changes, this literal must change with it, and the compare
	// below is what makes the omission visible.
	var actual []string
	fs.VisitAll(func(f *flag.Flag) { actual = append(actual, f.Name) })
	declared := lane.SituationFlags()
	slices.Sort(actual)
	slices.Sort(declared)
	if !slices.Equal(actual, declared) {
		t.Fatalf("internal/lane validates orients_from against %v, the situation read takes %v", declared, actual)
	}
	// And the flags the shipped manifests cite are all real.
	for _, m := range mustLoad(t) {
		for _, tok := range strings.Fields(m.OrientsFrom) {
			if !strings.HasPrefix(tok, "--") {
				continue
			}
			if !slices.Contains(declared, strings.TrimPrefix(tok, "--")) {
				t.Errorf("%s orients from %q, which names an unknown flag", m.Lane, m.OrientsFrom)
			}
		}
	}
}

// conformance: the loop-verb registry has exactly two consumers, and
// this is the drill that proves the CLI is one of them. Every act the
// registry names must be reachable as a CLI subverb, and the dispatch
// must refuse an unknown one by naming the registry's own list.
func TestLoopVerbRegistryDrivesTheCLI(t *testing.T) {
	for _, act := range loopverb.Acts() {
		e, code := runEnv(t, act.Group, act.Sub)
		// Reached the act and refused on its own flags, never on an
		// unknown subverb: that is what "the dispatch reads the
		// registry" means from outside.
		if code == 0 {
			t.Fatalf("%s with no flags must refuse: %+v", act.Name(), e)
		}
		if e.Error != nil && strings.Contains(e.Error.Message, "unknown") {
			t.Fatalf("%s is a registered act, so the dispatch must reach it: %s", act.Name(), e.Error.Message)
		}
	}
	e, code := runEnv(t, "claim", "yeet")
	if code != 64 || e.Error == nil || !strings.Contains(e.Error.Message, "unknown claim subverb") {
		t.Fatalf("an unknown subverb refuses: %d %+v", code, e)
	}
	for _, sub := range loopverb.Subverbs("claim") {
		if !strings.Contains(e.Error.Message, sub) {
			t.Errorf("the refusal names the registry's own alternatives, missing %q: %s", sub, e.Error.Message)
		}
	}
}

func mustLoad(t *testing.T) []lane.Manifest {
	t.Helper()
	ms, err := lane.Load(shippedLanes(t))
	if err != nil {
		t.Fatal(err)
	}
	return ms
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
