package main

// Small-team and fleet mode end to end (plans/os-6a08b166.md; Phase 9
// item 4; charter III.J's closing row).
//
// Both modes are ONE builder with two identity plans, because two
// hand-written drills that mean to be the same run drift and a
// difference that drifts in is invisible. The builder never carries
// its own capability table: it reads each lane's `grants` from the
// SHIPPED manifest in next/lanes/, so a manifest that stops describing
// its lane fails here, when the key it provisions stops being able to
// act.
//
// Neither mode wires a wake channel of any kind. That is not a setting
// to turn off — internal/loop has no wake seam at all — so the
// wakeless posture is pinned by surface rather than asserted by
// absence (TestLoopSurfaceIsWakeless).

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/lane"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
	"github.com/shaunlmason/open-seed/next/internal/version"
)

// identity is one actor the mode provisions: which lane it plays, and
// the name it is enrolled under. The lane decides its grants.
type identity struct {
	lane  string
	actor string
	seed  byte
}

// smallTeam is the charter's floor: ONE principal operating a minimal
// set of identities — "at least an implementing actor and a distinct
// verifying actor, so verdict key disjointness holds even when one
// person runs everything".
//
// The observer is here because the merge chain's terminal step is the
// observer lane's act, and "the full loop" ends at done. It is still
// one principal: three key files on one machine.
var smallTeam = []identity{
	{lane: "implementer", actor: "impl", seed: 51},
	{lane: "verifier", actor: "verify", seed: 52},
	{lane: "curator", actor: "observe", seed: 53},
}

// fleet is one identity per SHIPPED lane, built from the manifest set
// rather than listed here, so a lane added to next/lanes/ joins the
// fleet automatically.
func fleetPlan(t *testing.T) []identity {
	t.Helper()
	var out []identity
	seed := byte(60)
	for _, m := range mustLoad(t) {
		out = append(out, identity{lane: m.Lane, actor: m.Lane, seed: seed})
		seed++
	}
	if len(out) < 5 {
		t.Fatalf("the fleet is one identity per shipped lane, got %d", len(out))
	}
	return out
}

// modeStand is a provisioned mode: the ledger in one posture, and the
// keys the plan named.
type modeStand struct {
	remote, state, ld string
	src, base, head   string
	priv              string
	keys, fps         map[string]string
	grants            map[string][]string
}

// posture returns the args a verb takes to reach this stand's ledger.
func (m *modeStand) posture() []string {
	if m.remote != "" {
		return []string{"--remote", m.remote, "--state", m.state}
	}
	return []string{"--ledger", m.ld}
}

// buildMode provisions one mode. The grants come from the shipped lane
// manifests: this is the whole of D2, and it is why the builder takes
// a lane name rather than a capability list.
func buildMode(t *testing.T, plan []identity, remotePosture bool) *modeStand {
	t.Helper()
	m := &modeStand{keys: map[string]string{}, fps: map[string]string{}, grants: map[string][]string{}}
	byLane := map[string]lane.Manifest{}
	for _, man := range mustLoad(t) {
		byLane[man.Lane] = man
	}

	var appendRaw func(verb, subject, payload string)
	if remotePosture {
		dir, root, _ := writeKeys(t)
		m.priv = root
		m.remote = bareRemote(t)
		m.state = filepath.Join(dir, "state")
		resolve := seedRemoteGenesis(t, m.remote)
		libAppend(t, m.remote, resolve, "seed/0", ledger.UpgradeVerb, "system", `{"to": "`+version.Seed1+`"}`)
		appendRaw = func(verb, subject, payload string) {
			t.Helper()
			if e, code := runEnv(t, "ledger", "append", "--remote", m.remote, "--state", m.state,
				"--key", root, "--verb", verb, "--subject", subject, "--payload", payload); code != 0 {
				t.Fatalf("%s %s: %d %+v", verb, subject, code, e)
			}
		}
	} else {
		var keys, fps map[string]string
		m.ld, m.src, m.base, _, m.head, m.priv, _, keys, fps = offerLedger(t)
		_, _ = keys, fps
		appendRaw = func(verb, subject, payload string) {
			t.Helper()
			if e, code := runEnv(t, "ledger", "append", "--ledger", m.ld, "--key", m.priv,
				"--verb", verb, "--subject", subject, "--payload", payload); code != 0 {
				t.Fatalf("%s %s: %d %+v", verb, subject, code, e)
			}
		}
	}

	for _, id := range plan {
		man, ok := byLane[id.lane]
		if !ok {
			t.Fatalf("the plan names lane %q, which next/lanes/ does not ship", id.lane)
		}
		path, pub, fp := writeWorkerKey(t, id.seed)
		m.keys[id.actor], m.fps[id.actor] = path, fp
		m.grants[id.actor] = man.Grants
		appendRaw("actor.enrolled", fp, fmt.Sprintf(`{"key": %q, "kind": "agent", "name": %q}`, pub, id.actor))
		// The grants are the MANIFEST's, never a list this fixture
		// keeps. A lane holding none (the curator) is provisioned with
		// none: the case most likely to be wrong is the one a fixture
		// is most tempted to skip.
		for _, g := range man.Grants {
			appendRaw("actor.granted", fp, `{"capability": "`+g+`"}`)
		}
	}
	return m
}

// conformance: D2 — the builder reads the SHIPPED manifests, so a lane
// that stops describing itself fails here. Asserted directly: every
// provisioned actor's grants equal its manifest's, and the curator's
// emptiness is carried rather than skipped.
func TestModeGrantsComeFromTheShippedManifests(t *testing.T) {
	m := buildMode(t, fleetPlan(t), false)
	byLane := map[string][]string{}
	for _, man := range mustLoad(t) {
		byLane[man.Lane] = man.Grants
	}
	for _, id := range fleetPlan(t) {
		got, want := m.grants[id.actor], byLane[id.lane]
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("lane %s: provisioned %v, manifest says %v", id.lane, got, want)
		}
	}
	if len(m.grants["curator"]) != 0 {
		t.Errorf("the curator holds no write grant at all, and the fixture carries that: %v", m.grants["curator"])
	}
}

// conformance: D2 — the lane with NO grants can read and cannot write.
// A fixture that skipped it would be hiding the case most likely to be
// wrong, and "reads" is asserted rather than assumed.
func TestCuratorReadsAndCannotWrite(t *testing.T) {
	m := buildMode(t, fleetPlan(t), false)
	args := append([]string{"situation"}, m.posture()...)
	if e, code := runEnv(t, append(args, "--key", m.keys["curator"])...); code != 0 {
		t.Fatalf("the curator must be able to READ: %d %+v", code, e)
	}
	// The writes refuse AT THE GRANT RULE, asserted by code rather
	// than by "it failed". The first draft of this drill also tried
	// `claim take`, which refuses with `contention` because claiming
	// is online-only — a refusal that has nothing to do with grants,
	// and would have made this drill pass whatever the curator held.
	//
	// `merge request` is deliberately NOT in this list: its citation
	// is derived client-side, so on a subject with no verdict it
	// refuses `not_found` before the grant rule is ever reached. That
	// is the cooperative posture working — derive what you need, and
	// say so when you cannot — but it means the act cannot testify
	// about grants here.
	for _, act := range [][]string{
		{"merge", "observe", "--subject", "c-1", "--merged", strings.Repeat("a", 40), "--pr", "pr/1"},
	} {
		call := append(append([]string{act[0], act[1]}, m.posture()...), "--key", m.keys["curator"])
		call = append(call, act[2:]...)
		e, code := runEnv(t, call...)
		if code != 14 || e.Error == nil || e.Error.Code != "out_of_grant" {
			t.Errorf("%v must refuse OUT OF GRANT for a lane holding none, got %d %+v", act[:2], code, e.Error)
		}
	}
}
