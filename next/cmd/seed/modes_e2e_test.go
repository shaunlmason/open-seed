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
	"time"

	"github.com/shaunlmason/open-seed/next/internal/gitref"
	"github.com/shaunlmason/open-seed/next/internal/lane"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
	"github.com/shaunlmason/open-seed/next/internal/loop"
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
// It is exactly the charter's floor and no more: two key files on one
// machine. The supervisor and observer the loop also needs are staged
// as BACKGROUND identities, because no shipped lane grants what they
// require (see buildMode).
var smallTeam = []identity{
	{lane: "implementer", actor: "impl", seed: 51},
	{lane: "verifier", actor: "verify", seed: 52},
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
	remote, state, ld     string
	src, base, spec, head string
	priv                  string
	keys, fps             map[string]string
	grants                map[string][]string
	// appendRaw stages BACKGROUND facts on the raw seam. Setup may
	// use it (D3); nothing a drill asserts may come from it, because
	// `ledger append` runs no rules and a fact staged there proves
	// nothing about the boundary.
	appendRaw func(verb, subject, payload string)
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
//
// BOTH modes run on the REMOTE posture, and the plan's D6 was wrong to
// split them by transport. It reasoned that fleet needs remote for
// contention (true) and small-team needs local to finish, because
// `verdict render` was local-only. This card gave that verb
// `--remote`, so the second half evaporated — and the deeper reason is
// that small-team could never have run locally at all: `claim take` is
// refused off the remote ("an exclusive verb and claiming is
// online-only — two offline actors claiming the same contract have not
// claimed anything"), and a claim is the loop's FIRST act.
//
// So the mode is purely the identity plan, which is what the charter
// says it is: "one principal operating a minimal set of actor
// identities" versus "disjoint actors per lane". Neither clause
// mentions transport.
func buildMode(t *testing.T, plan []identity) *modeStand {
	t.Helper()
	m := &modeStand{keys: map[string]string{}, fps: map[string]string{}, grants: map[string][]string{}}
	byLane := map[string]lane.Manifest{}
	for _, man := range mustLoad(t) {
		byLane[man.Lane] = man
	}

	dir, root, _ := writeKeys(t)
	m.priv = root
	m.remote = bareRemote(t)
	m.state = filepath.Join(dir, "state")
	m.src, m.base, m.spec, m.head = verdictRepo(t)
	resolve := seedRemoteGenesis(t, m.remote)
	libAppend(t, m.remote, resolve, "seed/0", ledger.UpgradeVerb, "system", `{"to": "`+version.Seed1+`"}`)
	appendRaw := func(verb, subject, payload string) {
		t.Helper()
		if e, code := runEnv(t, "ledger", "append", "--remote", m.remote, "--state", m.state,
			"--key", root, "--verb", verb, "--subject", subject, "--payload", payload); code != 0 {
			t.Fatalf("%s %s: %d %+v", verb, subject, code, e)
		}
	}
	m.appendRaw = appendRaw

	// TWO background identities, deliberately outside the identity
	// plan, because the SHIPPED LANE SET CANNOT SUPPLY THEM.
	//
	// `offer.published` accepts only `supervise` or `operator`, and
	// `merge.observed` only `observer` or `operator`. No lane manifest
	// in next/lanes/ grants `supervise` or `observer` — the six lanes
	// grant claim, dispatch, verdict, maintenance+operator, and (the
	// curator) nothing. So a mode built purely from lanes could
	// neither publish the offer its own workers poll for nor record
	// the merge that ends the loop, and only the maintenance lane
	// could do either, through `operator`, which is not its job.
	//
	// That is a gap in the LANE SET, not in these fixtures, and lanes
	// are read-only in this card — so it is carded rather than papered
	// over. Both are kept out of the identity plan so the grants
	// assertion still measures only lane-derived identities.
	for _, bg := range []struct {
		name string
		cap  string
		seed byte
	}{{"supervisor", "supervise", 59}, {"observer", "observer", 58}} {
		path, pub, fp := writeWorkerKey(t, bg.seed)
		m.keys[bg.name], m.fps[bg.name] = path, fp
		appendRaw("actor.enrolled", fp, fmt.Sprintf(`{"key": %q, "kind": "agent", "name": %q}`, pub, bg.name))
		appendRaw("actor.granted", fp, `{"capability": "`+bg.cap+`"}`)
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
	m := buildMode(t, fleetPlan(t))
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
	m := buildMode(t, fleetPlan(t))
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

// contract stages one specified, offered contract. Setup rides the raw
// seam (D3): these are BACKGROUND facts, and nothing this file asserts
// is derived from them — every asserted property is produced by an
// admitted act.
func (m *modeStand) contract(t *testing.T, subject, supervisor string) {
	t.Helper()
	m.appendRaw("intent.filed", subject, `{"intent": "drill", "tier": "trivial", "budget": "small", "routing": "core"}`)
	m.appendRaw("contract.specified", subject,
		fmt.Sprintf(`{"acceptance": {"ref": "accept.md @ %s", "executable": false}}`, m.spec))
	offer := fmt.Sprintf(`{"eligibility": {"capabilities": ["claim"], "tiers": ["trivial"]}, "expires": %q}`,
		time.Now().UTC().Add(time.Hour).Format(time.RFC3339))
	if e, code := runEnv(t, "ledger", "append", "--remote", m.remote, "--state", m.state,
		"--key", m.keys[supervisor], "--verb", "offer.published", "--subject", subject,
		"--payload", offer); code != 0 {
		t.Fatalf("offer: %d %+v", code, e)
	}
}

// conformance: acceptance criteria 1, 2 and 7 — small-team mode runs
// the FULL loop on one principal's minimal identity set and reaches
// `done`, and verdict key disjointness is proven by a REFUSAL rather
// than by the fixture having written two key files.
func TestSmallTeamModeReachesDone(t *testing.T) {
	m := buildMode(t, smallTeam)
	const subject = "c-1"
	m.contract(t, subject, "supervisor")

	// The worker half runs through internal/loop — the library exists
	// for exactly this — with NO wake channel of any kind.
	worked := 0
	d, err := loop.New(implementerManifest(t), loopVerbs{}, m.posture(), m.keys["impl"],
		loop.WorkFunc(func(s string, sit loop.Situation) (int, error) {
			worked++
			if !sit.Holds(s) {
				return 0, fmt.Errorf("the work step runs inside a held window")
			}
			return 3, nil
		}), loop.WithBase(m.base+".."+m.head))
	if err != nil {
		t.Fatal(err)
	}
	step, err := d.Step(5)
	if err != nil {
		t.Fatalf("the loop must reach a deliberate exit: %v", err)
	}
	if step.Outcome != loop.Submitted || step.Subject != subject {
		t.Fatalf("the loop must submit: %s %s (%+v)", step.Outcome, step.Subject, step.Cause)
	}
	if worked != 1 {
		t.Fatalf("the work step runs once per iteration, ran %d", worked)
	}

	// Disjointness, PROVEN — and proven against the case that actually
	// threatens it.
	//
	// The implementing actor is granted `verdict` HERE, deliberately.
	// A principal running everything can grant themselves everything,
	// and the charter's claim is that disjointness holds anyway. Left
	// ungranted, this key would be refused `out_of_grant` — capability
	// absence, which `admit.go` is careful to call a different thing —
	// and the drill would have proven only that a key without the
	// grant cannot render. That is not the row.
	m.appendRaw("actor.granted", m.fps["impl"], `{"capability": "verdict"}`)

	e, code := runEnv(t, append(append([]string{"verdict", "render"}, m.posture()...),
		"--subject", subject, "--repo", m.src, "--key", m.keys["impl"], "--verdict", "pass")...)
	if code != 17 || e.Error == nil || e.Error.Code != "not_independent" {
		t.Fatalf("the implementing key must be refused as NOT INDEPENDENT, got %d %+v", code, e.Error)
	}
	if !strings.Contains(e.Error.Message, "claimant") && !strings.Contains(e.Error.Message, "submission") {
		t.Errorf("the refusal must name why: %q", e.Error.Message)
	}
	if e, code := runEnv(t, append(append([]string{"verdict", "render"}, m.posture()...),
		"--subject", subject, "--repo", m.src, "--key", m.keys["verify"], "--verdict", "pass")...); code != 0 {
		t.Fatalf("the distinct verifying key must be admitted: %d %+v", code, e)
	}

	// The terminal chain, each step its own admitted event.
	if e, code := runEnv(t, append(append([]string{"merge", "request"}, m.posture()...),
		"--subject", subject, "--key", m.keys["impl"])...); code != 0 {
		t.Fatalf("merge request: %d %+v", code, e)
	}
	if e, code := runEnv(t, append(append([]string{"merge", "observe"}, m.posture()...),
		"--subject", subject, "--key", m.keys["observer"], "--merged", m.head, "--pr", "pr/1")...); code != 0 {
		t.Fatalf("merge observe: %d %+v", code, e)
	}
	if got := m.stateOf(t, subject); got != "done" {
		t.Fatalf("the full loop ends at done, got %q", got)
	}
}

// stateOf folds the remote's own chain and reports the subject's
// state. It materializes the ref rather than reading a verb's report:
// when a chain of admitted acts is the thing under test, the chain is
// where the answer lives.
func (m *modeStand) stateOf(t *testing.T, subject string) string {
	t.Helper()
	c, err := gitref.NewClient(t.TempDir(), m.remote, "refs/seed/ledger")
	if err != nil {
		t.Fatal(err)
	}
	tip, err := c.Fetch()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := c.Materialize(tip, dir); err != nil {
		t.Fatal(err)
	}
	st, failEnv := loadVerdictState(dir)
	if failEnv != nil {
		t.Fatalf("the remote chain must verify: %+v", failEnv)
	}
	s, ok := st.fold.State(subject)
	if !ok {
		t.Fatalf("no %s in the remote fold", subject)
	}
	return s.State
}
