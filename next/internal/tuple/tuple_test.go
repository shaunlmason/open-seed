package tuple

import (
	"strings"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/version"
)

func full() Tuple {
	return Tuple{Principal: "p", Harness: "h/1", Model: "m/1", ToolPolicy: "tp", Environment: "env"}
}

// conformance: plans/os-8e53ffd9.md D1 — the tuple is strict, five
// fields, each non-empty; every malformed shape refuses by name.
func TestParseIsStrict(t *testing.T) {
	good := `{"principal":"p","harness":"h/1","model":"m/1","tool_policy":"tp","environment":"env"}`
	got, err := Parse([]byte(good))
	if err != nil || got != full() {
		t.Fatalf("a complete tuple parses: %v %+v", err, got)
	}
	for _, tc := range []struct{ name, raw, want string }{
		{"a missing field", `{"principal":"p","harness":"h/1","model":"m/1","tool_policy":"tp"}`, `"environment" is empty`},
		{"an empty field", `{"principal":"","harness":"h/1","model":"m/1","tool_policy":"tp","environment":"env"}`, `"principal" is empty`},
		{"whitespace only", `{"principal":"  ","harness":"h/1","model":"m/1","tool_policy":"tp","environment":"env"}`, `"principal" is empty`},
		{"an unknown field", `{"principal":"p","harness":"h/1","model":"m/1","tool_policy":"tp","environment":"env","runtime":"x"}`, "unknown field"},
		{"a non-string", `{"principal":7,"harness":"h/1","model":"m/1","tool_policy":"tp","environment":"env"}`, "strict object"},
		{"not an object", `["p"]`, "strict object"},
		{"trailing bytes", good + ` {}`, "bytes follow"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse([]byte(tc.raw))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("must refuse naming %q, got %v", tc.want, err)
			}
		})
	}
}

// conformance: D4 — equality is per field, and Diff names the first
// differing field with both values. Drilled per field, so a comparison
// that skipped one would fail here before any drill above it.
func TestDiffNamesTheFieldPerField(t *testing.T) {
	base := full()
	if !base.Equal(full()) {
		t.Fatal("a tuple equals itself")
	}
	for _, f := range Fields() {
		other := full()
		switch f {
		case "principal":
			other.Principal = "q"
		case "harness":
			other.Harness = "h/2"
		case "model":
			other.Model = "m/2"
		case "tool_policy":
			other.ToolPolicy = "tq"
		case "environment":
			other.Environment = "env2"
		}
		field, have, want, differs := base.Diff(other)
		if !differs || field != f || have == want {
			t.Errorf("field %s: Diff must name it with both values: %q %q %q %v", f, field, have, want, differs)
		}
		if base.Equal(other) {
			t.Errorf("field %s: a one-field difference is a difference", f)
		}
	}
	if len(Fields()) != 5 {
		t.Fatalf("the charter's tuple has five fields, this package names %d", len(Fields()))
	}
}

func TestAppliesAtSeed2AndLater(t *testing.T) {
	if Applies(version.Protocol) || Applies(version.Seed1) || !Applies(version.Seed2) || !Applies(version.Seed3) || Applies("seed/9") {
		t.Fatal("tuple semantics activate at seed/2 and stay on at every later registered version, never by ordering")
	}
	partial := Tuple{Harness: "h/1", Environment: "env"}
	if partial.Complete() || !full().Complete() {
		t.Fatal("Complete reports whether every field is set")
	}
}
