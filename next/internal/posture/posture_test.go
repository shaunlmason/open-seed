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
