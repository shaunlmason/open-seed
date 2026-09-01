package loop

// The loop drills (plans/os-abb206c8.md). The point of every case here
// is that it fails when the defense is removed rather than when the
// test is edited: the act gate is derived from a manifest this package
// does not author, and the liveness property is checked against what
// the loop actually invoked rather than against a spelling.

import (
	"errors"
	"strings"
	"testing"

	"os"
	"path/filepath"
	"slices"

	"github.com/shaunlmason/open-seed/next/internal/keyring"
	"github.com/shaunlmason/open-seed/next/internal/lane"
)

// recorder is the verb seam as a double: it records every invocation
// and answers from a scripted table, so a drill can assert on what the
// loop DID rather than on what it meant to do.
type recorder struct {
	calls  [][]string
	answer func(args []string) Result
}

func (r *recorder) Run(args ...string) Result {
	r.calls = append(r.calls, args)
	if r.answer != nil {
		return r.answer(args)
	}
	return Result{Exit: 0, OK: true}
}

// verb is the "<group> <sub>" of one recorded call, or the read's name.
func verb(args []string) string {
	if len(args) >= 2 && !strings.HasPrefix(args[1], "-") {
		return args[0] + " " + args[1]
	}
	if len(args) >= 1 {
		return args[0]
	}
	return ""
}

func (r *recorder) verbs() []string {
	var out []string
	for _, c := range r.calls {
		out = append(out, verb(c))
	}
	return out
}

// implementer is the shipped lane's shape: the acts a worker performs
// and the liveness its work steps emit.
func implementer() lane.Manifest {
	return lane.Manifest{
		Lane:         "implementer",
		Summary:      "takes a card to a PR",
		Grants:       []string{keyring.CapClaim},
		OrientsFrom:  "seed situation --remote <repo> --key <key> --since <position>",
		ActsThrough:  []string{"claim take", "budget reserve", "budget settle", "submission make", "claim park"},
		LivenessFrom: []string{"budget settle"},
		Inbox:        "wakes on push, convinces on the read",
		Fragments:    []string{"a.md"},
	}
}

func newDriver(t *testing.T, m lane.Manifest, r *recorder) *Driver {
	t.Helper()
	d, err := New(m, r, []string{"--remote", "/repo"}, "/key", "SHA256:actor",
		WorkFunc(func(string, Situation) (int, error) { return 3, nil }),
		WithBase("abc1234..abc1234"))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// offers answers a poll with one subject and a situation holding it.
func offers(subject string, override func(args []string) (Result, bool)) func([]string) Result {
	return func(args []string) Result {
		if override != nil {
			if res, ok := override(args); ok {
				return res
			}
		}
		switch verb(args) {
		case "offer list":
			return Result{OK: true, Position: "10", Result: map[string]any{
				"offers": []any{map[string]any{"subject": subject}}}}
		case "situation":
			return Result{OK: true, Position: "11", Result: map[string]any{
				"windows": []any{map[string]any{"subject": subject, "fence": "9"}}}}
		}
		return Result{OK: true, Position: "12"}
	}
}

// conformance: D2 — the loop performs no act its manifest does not
// declare, and the refusal comes BEFORE anything is signed. The gate is
// what makes "the manifest describes the loop" enforced rather than
// coincidental: editing one without the other now fails.
func TestActGateRefusesAnUndeclaredAct(t *testing.T) {
	m := implementer()
	m.ActsThrough = []string{"claim take", "budget reserve", "budget settle", "submission make", "claim park"}
	r := &recorder{}
	d := newDriver(t, m, r)

	// Declared: reaches the seam.
	if _, err := d.Act("claim take", "c-1"); err != nil {
		t.Fatalf("a declared act must reach the seam: %v", err)
	}
	if len(r.calls) != 1 {
		t.Fatalf("the declared act must have been invoked once, saw %v", r.verbs())
	}

	// Undeclared: refuses, and NOTHING reached the seam. A gate that
	// let the call through and judged it afterwards would have signed
	// the act it was meant to prevent.
	m2 := implementer()
	m2.ActsThrough = []string{"claim take"}
	r2 := &recorder{}
	d2 := newDriver(t, m2, r2)
	_, err := d2.Act("budget reserve", "c-1")
	if !errors.Is(err, ErrUndeclaredAct) {
		t.Fatalf("an undeclared act must refuse as ErrUndeclaredAct, got %v", err)
	}
	if len(r2.calls) != 0 {
		t.Fatalf("nothing may reach the seam on a refused act, saw %v", r2.verbs())
	}
	if !strings.Contains(err.Error(), "claim take") {
		t.Errorf("the refusal names what the lane DOES act through, so the caller can fix it: %v", err)
	}

	// And a name that is no loop act at all refuses the same way,
	// against the registry rather than against a list kept here.
	if _, err := d.Act("claim yeet", "c-1"); !errors.Is(err, ErrUndeclaredAct) {
		t.Errorf("a name outside the loop-verb registry must refuse: %v", err)
	}
}

// conformance: D4/AC5 — a budget refusal at the worker's spending gate
// parks the claim, and the packet's findings carry the refusal's code
// and message VERBATIM. The drill asserts the exact strings, so a lane
// that paraphrased its boundary would fail here.
func TestExhaustionParksCarryingTheRefusalVerbatim(t *testing.T) {
	// The double answers with the shape the REAL boundary produces,
	// verified against it in cmd/seed's end-to-end drill. A double that
	// invented a friendlier code would teach this package a boundary
	// that does not exist, which is the same vacuity the act gate and
	// the liveness property are written to avoid.
	const code = "chain_invalid"
	const message = "admission refused by rule budget: budget on c-1 refused: amount 100 exceeds remaining 4 of capacity 100 — reservations are checked and decremented at admission, the serialized view (next/spec/budgets.md)"
	var packetPath string
	r := &recorder{}
	r.answer = offers("c-1", func(args []string) (Result, bool) {
		switch verb(args) {
		case "budget reserve":
			return Result{Exit: 8, OK: false, Code: code, Message: message}, true
		case "claim park":
			for i, a := range args {
				if a == "--packet" && i+1 < len(args) {
					packetPath = args[i+1]
				}
			}
			return Result{OK: true, Position: "13"}, true
		}
		return Result{}, false
	})
	d := newDriver(t, implementer(), r)

	step, err := d.Step(100)
	if err != nil {
		t.Fatalf("parking is a deliberate exit, not an error: %v", err)
	}
	if step.Outcome != Parked || step.Subject != "c-1" {
		t.Fatalf("a refused reserve must park the window: %s %s", step.Outcome, step.Subject)
	}
	if packetPath == "" {
		t.Fatal("the park must carry a packet: every deliberate exit does")
	}
	body := readFile(t, packetPath)
	if !strings.Contains(body, code) {
		t.Errorf("the packet's findings must carry the refusal's code %q verbatim: %s", code, body)
	}
	if !strings.Contains(body, message) {
		t.Errorf("the packet's findings must carry the refusal's message verbatim: %s", body)
	}
	if !strings.Contains(body, "budget reserve") {
		t.Errorf("the finding names the step that was tried: %s", body)
	}

	// The window closed, and it closed through the loop verb rather
	// than by abandonment.
	if got := r.verbs(); got[len(got)-1] != "claim park" {
		t.Errorf("the last act must be the deliberate exit, saw %v", got)
	}
}

// conformance: D5 arm 2 — the loop reaches no liveness-only surface.
// Checked as a PROPERTY of what it invoked, not as a spelling: every
// call the loop made is either one of the reads it orients from or an
// act its manifest declares, so there is no path on which it emits
// without working.
func TestLoopReachesNoLivenessOnlySurface(t *testing.T) {
	r := &recorder{}
	m := implementer()
	r.answer = offers("c-1", nil)
	d := newDriver(t, m, r)
	if _, err := d.Step(5); err != nil {
		t.Fatal(err)
	}

	reads := map[string]bool{"offer list": true, "situation": true}
	declared := map[string]bool{}
	for _, a := range m.ActsThrough {
		declared[a] = true
	}
	if len(r.calls) == 0 {
		t.Fatal("this drill is vacuous unless the loop actually ran")
	}
	for _, got := range r.verbs() {
		if reads[got] || declared[got] {
			continue
		}
		t.Errorf("the loop invoked %q, which is neither a read it orients from nor an act %q declares: "+
			"the loop's vocabulary contains NO verb whose only purpose is to report liveness", got, m.Lane)
	}
	// Named explicitly, because it is the one the charter forbids.
	for _, got := range r.verbs() {
		if strings.HasPrefix(got, "obs ") {
			t.Errorf("the loop reached the observation surface directly (%q): liveness rides the work", got)
		}
	}
}

// conformance: the loop orients from ONE position-stamped read and
// carries the position forward as --since, so a resuming lane pays for
// the delta rather than reconstructing the world.
func TestOrientCarriesThePositionForward(t *testing.T) {
	r := &recorder{}
	r.answer = offers("c-1", nil)
	d := newDriver(t, implementer(), r)
	if _, err := d.Step(5); err != nil {
		t.Fatal(err)
	}
	var situations [][]string
	for _, c := range r.calls {
		if verb(c) == "situation" {
			situations = append(situations, c)
		}
	}
	if len(situations) < 2 {
		t.Fatalf("the loop orients before and after claiming, saw %d reads", len(situations))
	}
	if hasFlag(situations[0], "--since") {
		t.Error("the first read of a fresh loop has no position to resume from")
	}
	if !hasFlag(situations[1], "--since") {
		t.Errorf("the second read carries the first's position forward: %v", situations[1])
	}
	if got := flagValue(situations[1], "--since"); got != "11" {
		t.Errorf("--since must be the position the LAST read stamped, got %q", got)
	}
}

// conformance: construction refuses a lane that cannot run a loop, so
// the failure lands at build time rather than mid-window.
func TestConstructionRefusesALaneThatCannotLoop(t *testing.T) {
	r := &recorder{}
	work := WorkFunc(func(string, Situation) (int, error) { return 0, nil })
	for name, mutate := range map[string]func(*lane.Manifest){
		"a lane declaring no acts":     func(m *lane.Manifest) { m.ActsThrough = nil },
		"a lane declaring no liveness": func(m *lane.Manifest) { m.LivenessFrom = nil },
	} {
		m := implementer()
		mutate(&m)
		if _, err := New(m, r, []string{"--remote", "/r"}, "/k", "a", work, WithBase("a..a")); err == nil {
			t.Errorf("%s must refuse at construction", name)
		}
	}
	m := implementer()
	if _, err := New(m, r, []string{"--remote", "/r"}, "/k", "a", work); err == nil {
		t.Error("a loop with no resume coordinate must refuse: every deliberate exit carries a packet")
	}
	if _, err := New(m, r, nil, "/k", "a", work, WithBase("a..a")); err == nil {
		t.Error("a loop with no posture must refuse: it would have no ledger to orient against")
	}
	if _, err := New(m, r, []string{"--remote", "/r"}, "/k", "a", nil, WithBase("a..a")); err == nil {
		t.Error("a loop with no work step is a heartbeat, which the charter forbids")
	}
}

// conformance: a lost claim race is ordinary, not an error. In fleet
// mode two workers racing claim take means the loser re-orients and
// takes different work; treating that as a failure would manufacture an
// escalation storm out of contention.
func TestLostClaimIsIdleNotAnError(t *testing.T) {
	r := &recorder{}
	r.answer = offers("c-1", func(args []string) (Result, bool) {
		if verb(args) == "claim take" {
			return Result{Exit: 2, OK: false, Code: "contention", Message: "fence 4"}, true
		}
		return Result{}, false
	})
	d := newDriver(t, implementer(), r)
	step, err := d.Step(5)
	if err != nil {
		t.Fatalf("losing a race is ordinary lane behavior: %v", err)
	}
	if step.Outcome != Idle {
		t.Fatalf("a lost claim leaves no window to exit, so the iteration is idle: %s", step.Outcome)
	}
	for _, got := range r.verbs() {
		if got == "claim park" {
			t.Error("nothing was claimed, so nothing is owed a deliberate exit")
		}
	}
}

func hasFlag(args []string, name string) bool {
	for _, a := range args {
		if a == name {
			return true
		}
	}
	return false
}

func flagValue(args []string, name string) string {
	for i, a := range args {
		if a == name && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read the packet the loop wrote: %v", err)
	}
	return string(b)
}

// conformance: D5 arm 1 / AC7 — running the loop's declared liveness
// steps ADVANCES the observation stream keyed to the lane's actor and
// fence. This is the half 1a could not establish: it compared two
// labels and said so. Here the stream is sampled exactly as
// internal/obs keys it, so a stream written under the wrong actor or
// the wrong fence fails rather than passing on a technicality.
func TestLivenessRidesTheWorkAndIsKeyedToActorAndFence(t *testing.T) {
	const actor = "SHA256:actor"
	const fence = "9"
	obsDir := t.TempDir()
	r := &recorder{answer: offers("c-1", nil)}
	m := implementer()
	d, err := New(m, r, []string{"--remote", "/repo"}, "/key", actor,
		WorkFunc(func(string, Situation) (int, error) { return 3, nil }),
		WithBase("abc1234..abc1234"), WithObservations(obsDir))
	if err != nil {
		t.Fatal(err)
	}

	before := streamLines(t, obsDir, actor, fence)
	if _, err := d.Step(5); err != nil {
		t.Fatal(err)
	}
	after := streamLines(t, obsDir, actor, fence)
	if after <= before {
		t.Fatalf("running the loop must advance the stream at %s/%s/%s.jsonl: %d then %d",
			obsDir, actor, fence, before, after)
	}

	// Keyed to THIS lane's actor and fence, and nothing else. A stream
	// under another key is invisible to the classifier, so writing one
	// would look like liveness here while being silence to the reaper.
	if n := streamLines(t, obsDir, actor, "0"); n != 0 {
		t.Errorf("the stream is keyed by the window's fence, found %d lines under fence 0", n)
	}
	if n := streamLines(t, obsDir, "SHA256:someone-else", fence); n != 0 {
		t.Errorf("the stream is keyed by the lane's own actor, found %d lines under another", n)
	}

	// And exactly the DECLARED steps emit. budget settle is the
	// implementer's liveness source; claim take and submission make are
	// acts it performs without being liveness sources, so a loop
	// emitting per-act rather than per-declared-act would overcount.
	lines := streamBody(t, obsDir, actor, fence)
	for _, a := range m.ActsThrough {
		want := slices.Contains(m.LivenessFrom, a)
		if got := strings.Contains(lines, `"step":"`+a+`"`); got != want {
			t.Errorf("act %q: liveness_from says emits=%v, the stream says %v: %s", a, want, got, lines)
		}
	}
}

// conformance: a REFUSED act emits nothing. Otherwise a lane wedged at
// a boundary would look busiest exactly when it is most stuck, and the
// classification the maintenance reap depends on would be reading
// failure as progress.
func TestARefusedActEmitsNoLiveness(t *testing.T) {
	const actor = "SHA256:actor"
	obsDir := t.TempDir()
	r := &recorder{}
	r.answer = offers("c-1", func(args []string) (Result, bool) {
		if verb(args) == "budget settle" {
			return Result{Exit: 8, OK: false, Code: "chain_invalid", Message: "no open reservation"}, true
		}
		return Result{}, false
	})
	m := implementer()
	d, err := New(m, r, []string{"--remote", "/repo"}, "/key", actor,
		WorkFunc(func(string, Situation) (int, error) { return 1, nil }),
		WithBase("a..a"), WithObservations(obsDir))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Step(5); err != nil {
		t.Fatal(err)
	}
	if n := streamLines(t, obsDir, actor, "9"); n != 0 {
		t.Errorf("the only liveness source refused, so the stream must hold nothing: %d lines", n)
	}
}

func streamBody(t *testing.T, dir, actor, fence string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, actor, fence+".jsonl"))
	if err != nil {
		return ""
	}
	return string(b)
}

func streamLines(t *testing.T, dir, actor, fence string) int {
	t.Helper()
	body := strings.TrimSpace(streamBody(t, dir, actor, fence))
	if body == "" {
		return 0
	}
	return len(strings.Split(body, "\n"))
}
