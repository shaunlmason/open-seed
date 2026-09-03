package version

import (
	"regexp"
	"testing"
)

func TestIdentity(t *testing.T) {
	if Name != "seed" {
		t.Fatalf("Name = %q, want %q (the successor claims the name)", Name, "seed")
	}
	if Version == "" {
		t.Fatal("Version must not be empty")
	}
	// conformance: III.A groundwork; a genesis event names the governance
	// root and protocol version. The protocol identifier must stay in the
	// seed/N form the spec defines so version-mismatch refusal can compare.
	if !regexp.MustCompile(`^seed/[0-9]+$`).MatchString(Protocol) {
		t.Fatalf("Protocol = %q, want seed/<n> per next/spec/protocol.md", Protocol)
	}
}

// conformance: plans/os-99829835.md AC2, D4 — seed/4 is registered and
// every gate is a named list: eval semantics hold at seed/3 and seed/4
// alike, the levels at seed/4 exactly, and an unregistered version
// activates nothing however it would sort.
func TestSeed4GatesAreNamedLists(t *testing.T) {
	supported := map[string]bool{}
	for _, v := range Supported() {
		supported[v] = true
	}
	if !supported[Seed3] || !supported[Seed4] || !supported[Seed5] {
		t.Fatalf("Supported must carry seed/3, seed/4 and seed/5: %v", Supported())
	}
	for _, v := range []string{Seed1, Seed2, Seed3, Seed4, Seed5} {
		if !Activated(v) {
			t.Fatalf("Activated(%s) must hold", v)
		}
	}
	if Activated(Protocol) || Activated("seed/9") {
		t.Fatal("Activated is a named list: the genesis default and an unregistered version activate nothing")
	}
	if EvalApplies(Seed2) || !EvalApplies(Seed3) || !EvalApplies(Seed4) || !EvalApplies(Seed5) || EvalApplies("seed/9") {
		t.Fatal("EvalApplies is the named list {seed/3, seed/4, seed/5}: the equality that was right while seed/3 was newest closes on every later version")
	}
	if LevelsApply(Seed3) || !LevelsApply(Seed4) || !LevelsApply(Seed5) || LevelsApply("seed/9") {
		t.Fatal("LevelsApply is the named list {seed/4, seed/5}")
	}
	if ImportApplies(Seed4) || !ImportApplies(Seed5) || ImportApplies("seed/9") {
		t.Fatal("ImportApplies is seed/5 exactly, a named list of one")
	}
}
