package posture

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "seed.json")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// conformance: III posture item — every deployment declares one of the
// three charter postures; everything else refuses.
func TestLoadRoundTripsAndRefuses(t *testing.T) {
	for _, p := range []Posture{EnforcedSelfHosted, EnforcedForgeHosted, Cooperative} {
		cfg, err := Load(write(t, `{"posture": "`+string(p)+`"}`))
		if err != nil || cfg.Posture != p {
			t.Errorf("%s must round-trip, got %+v %v", p, cfg, err)
		}
	}

	if _, err := Load(filepath.Join(t.TempDir(), "absent.json")); !errors.Is(err, ErrUndeclared) {
		t.Errorf("missing declaration must be ErrUndeclared, got %v", err)
	}
	// A declaration that exists but cannot be read is an operational
	// failure, never a content judgment.
	if _, err := Load(t.TempDir()); !errors.Is(err, ErrUnreadable) {
		t.Errorf("an unreadable declaration must be ErrUnreadable, got %v", err)
	}
	for name, content := range map[string]string{
		"unknown posture": `{"posture": "anarchy"}`,
		"unknown field":   `{"posture": "cooperative", "mode": "yolo"}`,
		"trailing data":   `{"posture": "cooperative"} {}`,
		"not json":        `posture=cooperative`,
		"empty object":    `{}`,
	} {
		_, err := Load(write(t, content))
		if err == nil || errors.Is(err, ErrUndeclared) || errors.Is(err, ErrUnreadable) {
			t.Errorf("%s must refuse with a parse/validity error, got %v", name, err)
			continue
		}
		if !strings.Contains(err.Error(), "valid postures") {
			t.Errorf("%s must name the valid postures, got %v", name, err)
		}
	}
}

func TestPostureProperties(t *testing.T) {
	if !EnforcedSelfHosted.Enforced() || !EnforcedForgeHosted.Enforced() || Cooperative.Enforced() {
		t.Fatal("enforcement classification is wrong")
	}
	if Posture("anarchy").Valid() {
		t.Fatal("unknown postures must not validate")
	}
	for _, phrase := range []string{"does not hold", "advisory against a hostile credential", "§I.2"} {
		if !strings.Contains(Consequence, phrase) {
			t.Errorf("the named consequence must keep the charter phrase %q", phrase)
		}
	}
	if !strings.Contains(ForgeHostedGap, "Phase 12") {
		t.Error("the forge-hosted gap must name Phase 12")
	}
}

// conformance: III.L — the protected surface is enumerated in config
// (plans/os-465e356e.md D1): the declaration carries it as clean
// repository-relative prefixes, its own path is protected by
// construction, and membership is exact-or-under.
func TestProtectedSurfaceIsDeclaredAndSelfIncluding(t *testing.T) {
	cfg, err := Load(write(t, `{"posture": "enforced-self-hosted", "protected": ["Makefile", ".github/workflows/", "next/spec/transitions.json", "Makefile"]}`))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{".github/workflows", "Makefile", "next/spec/transitions.json", "seed.json"}
	if got := cfg.ProtectedSurface(); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("surface %v, want %v (deduplicated, sorted, self-including)", got, want)
	}
	for p, want := range map[string]bool{
		"Makefile":                        true,
		"seed.json":                       true,
		".github/workflows/check.yml":     true,
		".github/workflows":               true,
		"next/spec/transitions.json":      true,
		"next/spec/transitions.json.bak":  false,
		"Makefile.old":                    false,
		"docs/seed.json":                  false,
		"next/spec/lanes.md":              false,
		".github/workflowsX/a.yml":        false,
		"next/internal/redteam/x_test.go": false,
	} {
		if got := cfg.Protects(p); got != want {
			t.Errorf("Protects(%q) = %v, want %v", p, got, want)
		}
	}
	// A declaration without the field protects exactly its own path.
	bare, err := Load(write(t, `{"posture": "enforced-self-hosted"}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := bare.ProtectedSurface(); len(got) != 1 || got[0] != DeclarationPath || !bare.Protects(DeclarationPath) || bare.Protects("Makefile") {
		t.Fatalf("an unlisted surface is the declaration alone, got %v", got)
	}
	for name, content := range map[string]string{
		"empty entry":     `{"posture": "cooperative", "protected": [""]}`,
		"absolute":        `{"posture": "cooperative", "protected": ["/etc"]}`,
		"escaping":        `{"posture": "cooperative", "protected": ["../x"]}`,
		"dot":             `{"posture": "cooperative", "protected": ["."]}`,
		"unclean":         `{"posture": "cooperative", "protected": ["a//b"]}`,
		"backslash":       `{"posture": "cooperative", "protected": ["a\\b"]}`,
		"not a list":      `{"posture": "cooperative", "protected": "Makefile"}`,
		"not strings":     `{"posture": "cooperative", "protected": [1]}`,
		"unknown sibling": `{"posture": "cooperative", "protected": [], "protect": []}`,
	} {
		if _, err := Load(write(t, content)); err == nil || !strings.Contains(err.Error(), "valid postures") {
			t.Errorf("%s must refuse naming the valid postures, got %v", name, err)
		}
	}
	// Parse is Load without the file: the same strictness on bytes.
	if _, err := Parse([]byte(`{"posture": "anarchy"}`)); err == nil {
		t.Fatal("Parse must apply Load's validity check")
	}
}
