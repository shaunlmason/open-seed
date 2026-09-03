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
	for _, p := range []Posture{EnforcedSelfHosted, Cooperative} {
		cfg, err := Load(write(t, `{"posture": "`+string(p)+`"}`))
		if err != nil || cfg.Posture != p {
			t.Errorf("%s must round-trip, got %+v %v", p, cfg, err)
		}
	}
	// The third posture round-trips with the block it requires
	// (TestAdmissionBlockIsPostureGated holds it to that).
	if cfg, err := Load(write(t, forgeHosted)); err != nil || cfg.Posture != EnforcedForgeHosted {
		t.Errorf("%s must round-trip with its block, got %+v %v", EnforcedForgeHosted, cfg, err)
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
}

const forgeHosted = `{"posture": "enforced-forge-hosted", "admission": {"endpoint": "http://127.0.0.1:8437", "identity": "seed-admission[bot]", "ledger_ref": "refs/heads/seed-ledger", "checks": ["check", "verify"], "reviews": 1, "owners": ["@governance-root"]}}`

// conformance: III.B — the third posture is declared with the block
// the service and the reconciler read (plans/os-5c8a312c.md D3, D5):
// required under enforced-forge-hosted, refused under the others, and
// strict about the endpoint, the identity and the branch namespace.
func TestAdmissionBlockIsPostureGated(t *testing.T) {
	cfg, err := Load(write(t, forgeHosted))
	if err != nil {
		t.Fatalf("a complete forge-hosted declaration must load: %v", err)
	}
	if cfg.Admission == nil || cfg.Admission.Endpoint != "http://127.0.0.1:8437" || cfg.Admission.Identity != "seed-admission[bot]" || cfg.LedgerRef() != "refs/heads/seed-ledger" || cfg.Admission.Reviews != 1 || len(cfg.Admission.Checks) != 2 || len(cfg.Admission.Owners) != 1 {
		t.Fatalf("the block must round-trip, got %+v", cfg.Admission)
	}
	cfg, err = Load(write(t, `{"posture": "enforced-forge-hosted", "admission": {"endpoint": "https://admit.example", "identity": "bot"}}`))
	if err != nil || cfg.LedgerRef() != DefaultLedgerRef {
		t.Fatalf("an unnamed ledger ref defaults to the branch %s, got %q %v", DefaultLedgerRef, cfg.LedgerRef(), err)
	}
	for _, p := range []Posture{EnforcedSelfHosted, Cooperative} {
		cfg, err := Load(write(t, `{"posture": "`+string(p)+`"}`))
		if err != nil || cfg.LedgerRef() != "" {
			t.Fatalf("%s carries no ledger ref of its own, got %q %v", p, cfg.LedgerRef(), err)
		}
	}
	for name, content := range map[string]string{
		"missing block":     `{"posture": "enforced-forge-hosted"}`,
		"block elsewhere":   `{"posture": "cooperative", "admission": {"endpoint": "http://x", "identity": "bot"}}`,
		"no endpoint":       `{"posture": "enforced-forge-hosted", "admission": {"identity": "bot"}}`,
		"bad scheme":        `{"posture": "enforced-forge-hosted", "admission": {"endpoint": "ftp://x", "identity": "bot"}}`,
		"no identity":       `{"posture": "enforced-forge-hosted", "admission": {"endpoint": "http://x", "identity": " "}}`,
		"custom namespace":  `{"posture": "enforced-forge-hosted", "admission": {"endpoint": "http://x", "identity": "bot", "ledger_ref": "refs/seed/ledger"}}`,
		"negative reviews":  `{"posture": "enforced-forge-hosted", "admission": {"endpoint": "http://x", "identity": "bot", "reviews": -1}}`,
		"empty check":       `{"posture": "enforced-forge-hosted", "admission": {"endpoint": "http://x", "identity": "bot", "checks": [""]}}`,
		"empty owner":       `{"posture": "enforced-forge-hosted", "admission": {"endpoint": "http://x", "identity": "bot", "owners": [" "]}}`,
		"unknown sub-field": `{"posture": "enforced-forge-hosted", "admission": {"endpoint": "http://x", "identity": "bot", "token": "secret"}}`,
		"protected slash":   `{"posture": "cooperative", "protected": ["/Makefile"]}`,
		"protected dotdot":  `{"posture": "cooperative", "protected": ["../x"]}`,
		"protected unclean": `{"posture": "cooperative", "protected": ["next//spec"]}`,
		"protected empty":   `{"posture": "cooperative", "protected": [" "]}`,
	} {
		_, err := Load(write(t, content))
		if err == nil || errors.Is(err, ErrUndeclared) || errors.Is(err, ErrUnreadable) {
			t.Errorf("%s must refuse with a validity error, got %v", name, err)
		}
	}
	if _, err := Load(write(t, `{"posture": "enforced-forge-hosted"}`)); err == nil || !strings.Contains(err.Error(), "admission block") {
		t.Errorf("the missing block must be named, got %v", err)
	}
}
