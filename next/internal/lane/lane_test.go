package lane

// The lane validation drills (plans/os-cf1c9688.md): each check fails
// with its own finding, and the shipped set validates clean. The point
// of every case below is that the finding comes from an authority
// elsewhere in the tree rather than from a list this package keeps.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/keyring"
	"github.com/shaunlmason/open-seed/next/internal/loopverb"
)

// good is a manifest that passes every check, for mutation by each
// case: a drill that builds its own broken manifest from scratch can
// pass for the wrong reason when a second field is broken too.
func good() Manifest {
	return Manifest{
		Lane:         "implementer",
		Summary:      "takes a card to a PR",
		Grants:       []string{keyring.CapClaim},
		OrientsFrom:  "seed situation --ledger <dir> --key <key> --since <position>",
		ActsThrough:  []string{"claim take", "budget settle"},
		LivenessFrom: []string{"budget settle"},
		Inbox:        "wakes on push, convinces on the read",
		Fragments:    []string{"a.md"},
	}
}

// fixture writes a lanes directory holding one manifest and the
// fragments it names.
func fixture(t *testing.T, m Manifest, fragments map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, FragmentDir), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range fragments {
		p := filepath.Join(dir, FragmentDir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, m.Lane+".json"), b, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func findingFields(fs []Finding) []string {
	var out []string
	for _, f := range fs {
		out = append(out, f.Field)
	}
	return out
}

// conformance: III.J — the declarations are checked, and each failure
// names the field it failed on.
func TestEachCheckHasItsOwnFinding(t *testing.T) {
	for name, tc := range map[string]struct {
		mutate func(*Manifest)
		field  string
		says   string
	}{
		"a grant outside the keyring vocabulary": {
			mutate: func(m *Manifest) { m.Grants = []string{"wizard"} },
			field:  "grants",
			says:   "not a capability",
		},
		"an act that is not a loop act": {
			mutate: func(m *Manifest) { m.ActsThrough = []string{"claim yeet"}; m.LivenessFrom = []string{"claim yeet"} },
			field:  "acts_through",
			says:   "not a loop act",
		},
		"grants that do not intersect what the act accepts": {
			mutate: func(m *Manifest) { m.Grants = []string{keyring.CapObserver} },
			field:  "acts_through",
			says:   "admits for any of",
		},
		"an empty liveness declaration": {
			mutate: func(m *Manifest) { m.LivenessFrom = nil },
			field:  "liveness_from",
			says:   "declares no liveness source",
		},
		"a liveness source that is not a work step": {
			mutate: func(m *Manifest) { m.LivenessFrom = []string{"claim park"} },
			field:  "liveness_from",
			says:   "heartbeat by another name",
		},
		"no orienting read": {
			mutate: func(m *Manifest) { m.OrientsFrom = "" },
			field:  "orients_from",
			says:   "names no orienting read",
		},
		"an orienting read that is not the situation read": {
			mutate: func(m *Manifest) { m.OrientsFrom = "seed project rebuild" },
			field:  "orients_from",
			says:   "is not the situation read",
		},
		"a flag the situation read does not take": {
			mutate: func(m *Manifest) { m.OrientsFrom = "seed situation --ledger <dir> --everything" },
			field:  "orients_from",
			says:   "is not a flag",
		},
		"an orienting read omitting a flag the surface requires": {
			// Naming only real flags is not enough: this command
			// exits 64 without reaching the ledger, so the lane never
			// orients at all.
			mutate: func(m *Manifest) { m.OrientsFrom = "seed situation --key <key> --since <position>" },
			field:  "orients_from",
			says:   "omits --ledger",
		},
		"no inbox declaration": {
			mutate: func(m *Manifest) { m.Inbox = "" },
			field:  "inbox",
			says:   "declares no inbox",
		},
		"a missing fragment": {
			mutate: func(m *Manifest) { m.Fragments = []string{"absent.md"} },
			field:  "fragments",
			says:   "absent.md",
		},
		"a lane granted claim that declares no acts": {
			mutate: func(m *Manifest) { m.ActsThrough = nil; m.LivenessFrom = nil },
			field:  "acts_through",
			says:   "claiming is a loop act",
		},
	} {
		m := good()
		tc.mutate(&m)
		dir := fixture(t, m, map[string]string{"a.md": "prose\n"})
		got := Validate(dir, []Manifest{m})
		found := false
		for _, f := range got {
			if f.Field == tc.field && strings.Contains(f.Message, tc.says) {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: want a %s finding saying %q, got %v (%v)", name, tc.field, tc.says, findingFields(got), got)
		}
	}
}

// conformance: the dispatcher's posture is an ALLOWLIST, so a
// capability nobody thought to exclude fails by default. operator is
// the case that motivated it: it is not an authoring, verdict or
// sealing grant, so the first draft's blocklist would have admitted
// the strongest capability in the keyring on the lane that reads the
// most hostile input.
func TestDispatcherPostureIsAnAllowlist(t *testing.T) {
	base := func() Manifest {
		m := good()
		m.Lane = "dispatcher"
		m.Grants = []string{keyring.CapDispatch}
		m.ActsThrough = nil
		m.LivenessFrom = nil
		return m
	}
	clean := base()
	dir := fixture(t, clean, map[string]string{"a.md": "prose\n"})
	if got := Validate(dir, []Manifest{clean}); len(got) != 0 {
		t.Fatalf("the allowlisted dispatcher validates clean: %v", got)
	}

	for _, extra := range []string{keyring.CapOperator, keyring.CapClaim, keyring.CapVerdict, keyring.CapMaintenance} {
		m := base()
		m.Grants = append(m.Grants, extra)
		dir := fixture(t, m, map[string]string{"a.md": "prose\n"})
		found := false
		for _, f := range Validate(dir, []Manifest{m}) {
			if f.Field == "grants" && strings.Contains(f.Message, extra) && strings.Contains(f.Message, "allowlist") {
				found = true
			}
		}
		if !found {
			t.Errorf("%q on the dispatcher must fail by name against the allowlist", extra)
		}
	}

	// And the allowlist is required, not merely permitted.
	m := base()
	m.Grants = nil
	dir = fixture(t, m, map[string]string{"a.md": "prose\n"})
	found := false
	for _, f := range Validate(dir, []Manifest{m}) {
		if f.Field == "grants" && strings.Contains(f.Message, "does not grant") {
			found = true
		}
	}
	if !found {
		t.Error("a dispatcher granting nothing must fail: the allowlist names what it needs")
	}
}

// conformance: the accepted-capability set is an OR-set, so holding
// ONE alternative passes. A containment check here would force
// operator onto every lane that claims or spends, which is the posture
// the charter forbids.
func TestAcceptedCapabilitiesAreAlternatives(t *testing.T) {
	act, ok := loopverb.ByName("budget settle")
	if !ok {
		t.Fatal("budget settle is a loop act")
	}
	accepted := keyring.AcceptedCapabilities(act.Verb)
	if len(accepted) < 2 {
		t.Fatalf("this drill needs a verb with alternatives, %s accepts %v", act.Verb, accepted)
	}
	for _, one := range accepted {
		m := good()
		m.Grants = []string{one}
		m.ActsThrough = []string{"budget settle"}
		m.LivenessFrom = []string{"budget settle"}
		dir := fixture(t, m, map[string]string{"a.md": "prose\n"})
		for _, f := range Validate(dir, []Manifest{m}) {
			if f.Field == "acts_through" {
				t.Errorf("holding %q alone must satisfy %s, which accepts %v: %s", one, act.Verb, accepted, f.Message)
			}
		}
	}
}

// conformance: resolution is ordered and byte-stable, and the order is
// the manifest's rather than the directory's.
func TestResolveIsOrderedAndStable(t *testing.T) {
	m := good()
	m.Fragments = []string{"b.md", "a.md"}
	dir := fixture(t, m, map[string]string{"a.md": "alpha\n", "b.md": "beta\n"})
	first, err := Resolve(dir, m)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(first, "beta") {
		t.Fatalf("fragments resolve in DECLARED order, not directory order: %q", first)
	}
	second, err := Resolve(dir, m)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("resolution must be byte-stable across runs")
	}
}

// conformance: an unreferenced fragment is prose nobody reads, and the
// ordered lists are the only thing that makes a fragment part of a
// role.
func TestOrphanFragmentIsAFinding(t *testing.T) {
	m := good()
	dir := fixture(t, m, map[string]string{"a.md": "prose\n", "stray.md": "unreferenced\n"})
	found := false
	for _, f := range Validate(dir, []Manifest{m}) {
		if f.Field == "fragments" && strings.Contains(f.Message, "stray.md") {
			found = true
		}
	}
	if !found {
		t.Error("an orphaned fragment must be named")
	}
}

// conformance: the prose sweep is the belt to the property check's
// braces — a fragment could tell an agent to heartbeat without
// declaring it.
func TestFragmentInstructingABareHeartbeatIsAFinding(t *testing.T) {
	m := good()
	dir := fixture(t, m, map[string]string{"a.md": "Every 30 seconds run `seed obs emit --count 1`.\n"})
	found := false
	for _, f := range Validate(dir, []Manifest{m}) {
		if f.Field == "fragments" && strings.Contains(f.Message, "bare liveness emit") {
			found = true
		}
	}
	if !found {
		t.Error("a fragment instructing a bare emit must be named")
	}
}

// conformance: the manifest is strict data — an unknown field is a
// typo that would otherwise be silently ignored, and the filename
// carries the lane's identity.
func TestManifestsAreStrict(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, FragmentDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "implementer.json"),
		[]byte(`{"lane": "implementer", "grants": [], "acts_thru": []}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Fatal("an unknown field must refuse rather than be ignored")
	}
	if err := os.WriteFile(filepath.Join(dir, "implementer.json"),
		[]byte(`{"lane": "planner"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), "must live in") {
		t.Fatalf("the filename carries the lane's identity: %v", err)
	}
}
