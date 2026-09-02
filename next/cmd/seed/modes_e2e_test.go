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
	"os"
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
// machine. The supervisor and observer the loop also needs are the
// charter's non-loop ROLES, provisioned from their own manifests in
// every mode (see buildMode).
var smallTeam = []identity{
	{lane: "implementer", actor: "impl", seed: 51},
	{lane: "verifier", actor: "verify", seed: 52},
}

// fleet is one identity per SHIPPED LANE, built from the manifest set
// rather than listed here. Filtered on kind (plans/os-d6a52784.md D7):
// a manifest of kind role is the charter's supervisor or observer,
// which invite and attest to work rather than take it, and a fleet
// that provisioned them as workers would have a key both inviting work
// and competing for it. The first draft ranged over every manifest, so
// adding the role files would have grown the fleet to eight lanes
// while the plan declared them non-lanes (review finding on #210).
func fleetPlan(t *testing.T) []identity {
	t.Helper()
	var out []identity
	seed := byte(60)
	for _, m := range mustLoad(t) {
		if m.Kind != lane.KindLane {
			continue
		}
		out = append(out, identity{lane: m.Lane, actor: m.Lane, seed: seed})
		seed++
	}
	if len(out) != len(lane.CharterLanes()) {
		t.Fatalf("the fleet is one identity per charter lane (%d), got %d", len(lane.CharterLanes()), len(out))
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

	// The charter's non-loop ROLES, provisioned from their own
	// manifests in every mode (plans/os-d6a52784.md D7). offer.published
	// accepts supervise or operator and merge.observed accepts observer
	// or operator; the supervisor (§II.9) and the governed observer
	// (§8) are the parts the charter defines to hold them, outside the
	// six lanes. An earlier draft staged both as identities this
	// fixture invented, because no manifest granted either capability:
	// honest then, and once the manifests exist, continuing to invent
	// them would leave this fixture asserting exactly what it asserted
	// before the gap closed. Their grants are the MANIFESTS', like every
	// lane's, so the grants drill covers them too.
	roleSeeds := map[string]byte{"supervisor": 59, "observer": 58}
	for _, man := range mustLoad(t) {
		if man.Kind != lane.KindRole {
			continue
		}
		seed, ok := roleSeeds[man.Lane]
		if !ok {
			t.Fatalf("role %q shipped with no fixture seed: add one here rather than letting it be skipped", man.Lane)
		}
		path, pub, fp := writeWorkerKey(t, seed)
		m.keys[man.Lane], m.fps[man.Lane] = path, fp
		m.grants[man.Lane] = man.Grants
		appendRaw("actor.enrolled", fp, fmt.Sprintf(`{"key": %q, "kind": "agent", "name": %q}`, pub, man.Lane))
		for _, g := range man.Grants {
			appendRaw("actor.granted", fp, `{"capability": "`+g+`"}`)
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
	// The roles came from MANIFESTS, not from this file (D7): their
	// grants equal what next/lanes/ ships, and they are not in the
	// fleet's identity plan.
	roles := 0
	for _, man := range mustLoad(t) {
		if man.Kind != lane.KindRole {
			continue
		}
		roles++
		if got := strings.Join(m.grants[man.Lane], ","); got != strings.Join(man.Grants, ",") {
			t.Errorf("role %s: provisioned %q, its manifest says %v", man.Lane, got, man.Grants)
		}
		for _, id := range fleetPlan(t) {
			if id.lane == man.Lane {
				t.Errorf("role %s is in the fleet's identity plan: a role invites or attests, it does not take work", man.Lane)
			}
		}
	}
	if roles != 2 {
		t.Errorf("the charter defines two non-loop roles (supervisor §II.9, observer §8), found %d", roles)
	}
	if got := len(fleetPlan(t)); got != 6 {
		t.Errorf("the fleet is exactly the charter's six lanes after the role files are added, got %d", got)
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

// attempt is one act the lane put through the seam, with the position
// the envelope was stamped at and the refusal code if any.
type attempt struct {
	iter     int
	act      string
	position string
	code     string
}

// transcript wraps the REAL verbs and only observes. Instrumenting a
// loop by replacing the boundary it is being tested against proves
// nothing about that boundary (#202), so this records and forwards.
type transcript struct {
	inner loop.Verbs
	iter  int
	log   []attempt
}

func (tr *transcript) Run(args ...string) loop.Result {
	res := tr.inner.Run(args...)
	act := args[0]
	if len(args) > 1 && !strings.HasPrefix(args[1], "-") {
		act += " " + args[1]
	}
	tr.log = append(tr.log, attempt{iter: tr.iter, act: act, position: res.Position, code: res.Code})
	return res
}

// refusals returns the attempts the boundary declined.
func (tr *transcript) refusals() []attempt {
	var out []attempt
	for _, a := range tr.log {
		if a.code != "" {
			out = append(out, a)
		}
	}
	return out
}

// blindRetry reports the first spin: the SAME act refusing with the
// SAME code on consecutive iterations from a position that did not
// advance. Same act, same knowledge, same answer is the fourth
// outcome the build plan forbids — a lane that learned nothing and
// tried again.
func (tr *transcript) blindRetry() (attempt, bool) {
	prev := map[string]attempt{}
	for _, a := range tr.refusals() {
		key := a.act + "\x00" + a.code
		if p, seen := prev[key]; seen && a.iter == p.iter+1 && a.position == p.position {
			return a, true
		}
		prev[key] = a
	}
	return attempt{}, false
}

// racing lands a RIVAL claim inside the window, just before the lane's
// own `claim take` reaches the boundary.
//
// Two workers stepped one after another do not race: the second polls
// after the first has claimed, finds nothing offered, and goes idle at
// the poll with no refusal at all. The middle arm answers a REFUSAL,
// so a drill built that way would have had no refusal to answer and
// would have counted an empty poll as convergence.
//
// Planting the rival inside the window puts the lane in the race BY
// CONSTRUCTION rather than by timing, which is the same reason #202's
// interleaving drill rotates a key from inside the seam: a race
// reproduced by sleeping passes green on a slower runner.
type racing struct {
	*transcript
	t     *testing.T
	plant func()
	fired bool
}

func (r *racing) Run(args ...string) loop.Result {
	if !r.fired && len(args) > 1 && args[0] == "claim" && args[1] == "take" {
		r.fired = true
		r.plant()
	}
	return r.transcript.Run(args...)
}

// conformance: acceptance criteria 3, 4 and 5 — fleet mode, one

// identity per shipped lane, two workers racing `claim take` against
// one remote, reaching `done`; every convergence arm exercised; and
// the blind-retry detector run over the whole transcript.
func TestFleetModeConvergesAndReachesDone(t *testing.T) {
	m := buildMode(t, fleetPlan(t))
	const subject = "c-1"
	m.contract(t, subject, "supervisor")

	// Two workers, both eligible, both polling the same offer. Only
	// one can hold the window: the loser's `claim take` is REFUSED for
	// contention, which is the refusal the middle arm answers.
	//
	// The planner lane is the second worker rather than a second
	// implementer, because it is the other shipped lane granted
	// `claim` — the fleet races the lanes it actually has.
	tr := &transcript{inner: loopVerbs{}}
	r := &racing{transcript: tr, t: t, plant: func() {
		// The rival: the OTHER shipped lane granted `claim`, taking
		// the window through the same admitted verb.
		if e, code := runEnv(t, append(append([]string{"claim", "take"}, m.posture()...),
			"--subject", subject, "--key", m.keys["planner"])...); code != 0 {
			t.Fatalf("the rival claim must land, or there is no race: %d %+v", code, e)
		}
	}}

	lost, err := m.stepWith(t, r, "implementer", subject)
	if err != nil {
		t.Fatalf("losing a race is not an error: %v", err)
	}
	if lost.Outcome != loop.Idle {
		t.Fatalf("the loser re-orients rather than parking what it never held: %s", lost.Outcome)
	}
	if lost.Step != "claim take" || lost.Cause.Code == "" {
		t.Fatalf("the losing iteration must END on a REFUSED claim, carrying it: step=%q cause=%+v",
			lost.Step, lost.Cause)
	}
	if lost.Cause.Code != "contention" {
		t.Errorf("a lost race is ordinary contention, not %q: %s", lost.Cause.Code, lost.Cause.Message)
	}

	// THE MIDDLE ARM: that refusal answered on the next iteration by a
	// refreshed position-stamped read showing the act is no longer
	// owed. The lane takes no different work here because the fixture
	// offers only one contract, which is the arm in its purest form.
	tr.iter++
	next, err := m.stepWith(t, tr, "implementer", subject)
	if err != nil {
		t.Fatalf("the next iteration must not error: %v", err)
	}
	if next.Outcome != loop.Idle {
		t.Fatalf("the refreshed read shows the work is no longer owed: %s (%+v)", next.Outcome, next.Cause)
	}
	if spun, found := tr.blindRetry(); found {
		t.Fatalf("a blind retry: %s refused %q twice from position %s with nothing learned",
			spun.act, spun.code, spun.position)
	}

	// ANTI-VACUITY: the arm must have been EXERCISED. Every assertion
	// above is true of a lane that met no refusal at all, which is
	// exactly how this whole fixture could ship proving nothing.
	armed := false
	for _, a := range tr.refusals() {
		if a.act == "claim take" && a.code == "contention" {
			armed = true
		}
	}
	if !armed {
		t.Fatal("the middle arm is UNEXERCISED: no claim was refused, so the refreshed read answered nothing")
	}

	// And the rival drives the contract the rest of the way to done,
	// which is the loop completing elsewhere from the shared ledger.
	m.finish(t, subject, "planner")
	if got := m.stateOf(t, subject); got != "done" {
		t.Fatalf("fleet mode ends at done, got %q", got)
	}
}

// finish drives a held contract through submission, verdict and the
// merge chain, each step its own admitted act.
func (m *modeStand) finish(t *testing.T, subject, holder string) {
	t.Helper()
	for _, act := range [][]string{
		{"budget", "reserve", "--amount", "3"},
		{"budget", "settle", "--actuals", "1"},
		{"submission", "make", "--packet", m.packet(t, subject), "--base", m.base + ".." + m.head},
	} {
		call := append(append([]string{act[0], act[1]}, m.posture()...),
			"--subject", subject, "--key", m.keys[holder])
		call = append(call, act[2:]...)
		if e, code := runEnv(t, call...); code != 0 {
			t.Fatalf("%v: %d %+v", act[:2], code, e)
		}
	}
	if e, code := runEnv(t, append(append([]string{"verdict", "render"}, m.posture()...),
		"--subject", subject, "--repo", m.src, "--key", m.keys["verifier"], "--verdict", "pass")...); code != 0 {
		t.Fatalf("verdict render: %d %+v", code, e)
	}
	if e, code := runEnv(t, append(append([]string{"merge", "request"}, m.posture()...),
		"--subject", subject, "--key", m.keys[holder])...); code != 0 {
		t.Fatalf("merge request: %d %+v", code, e)
	}
	if e, code := runEnv(t, append(append([]string{"merge", "observe"}, m.posture()...),
		"--subject", subject, "--key", m.keys["observer"], "--merged", m.head, "--pr", "pr/2")...); code != 0 {
		t.Fatalf("merge observe: %d %+v", code, e)
	}
}

// step runs one loop iteration for a lane's actor through the given
// transcript.
func (m *modeStand) stepWith(t *testing.T, verbs loop.Verbs, laneName, subject string) (loop.StepResult, error) {
	t.Helper()
	var man lane.Manifest
	for _, l := range mustLoad(t) {
		if l.Lane == laneName {
			man = l
		}
	}
	d, err := loop.New(man, verbs, m.posture(), m.keys[laneName],
		loop.WorkFunc(func(string, loop.Situation) (int, error) { return 1, nil }),
		loop.WithBase(m.base+".."+m.head))
	if err != nil {
		t.Fatal(err)
	}
	return d.Step(3)
}

// packet writes the four-part handoff a deliberate exit carries. The
// submission verb takes a file, so the fixture supplies one rather
// than reaching for a flag that does not exist.
func (m *modeStand) packet(t *testing.T, subject string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "packet.json")
	body := fmt.Sprintf(`{"acceptance": [%q], "decisions": [], "base": %q, "refs": [], "findings": []}`,
		"accept.md @ "+m.spec, m.base+".."+m.head)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// conformance: acceptance criterion 5 — the blind-retry detector
// detects. A detector that has only ever seen converging runs has not
// been shown to catch anything, so it is fed a known spin and a known
// non-spin directly.
func TestBlindRetryDetector(t *testing.T) {
	for _, tc := range []struct {
		name string
		log  []attempt
		spin bool
	}{
		{"the same act, same code, same position on consecutive iterations", []attempt{
			{iter: 1, act: "claim take", position: "9", code: "contention"},
			{iter: 2, act: "claim take", position: "9", code: "contention"},
		}, true},
		{"the position advanced, so the lane learned something", []attempt{
			{iter: 1, act: "claim take", position: "9", code: "contention"},
			{iter: 2, act: "claim take", position: "11", code: "contention"},
		}, false},
		{"a different code is a different refusal", []attempt{
			{iter: 1, act: "claim take", position: "9", code: "contention"},
			{iter: 2, act: "claim take", position: "9", code: "fenced_out"},
		}, false},
		{"not consecutive: something else happened between", []attempt{
			{iter: 1, act: "claim take", position: "9", code: "contention"},
			{iter: 3, act: "claim take", position: "9", code: "contention"},
		}, false},
		{"successes are not retries", []attempt{
			{iter: 1, act: "claim take", position: "9"},
			{iter: 2, act: "claim take", position: "9"},
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tr := &transcript{log: tc.log}
			if _, got := tr.blindRetry(); got != tc.spin {
				t.Errorf("blindRetry = %v, want %v", got, tc.spin)
			}
		})
	}
}
