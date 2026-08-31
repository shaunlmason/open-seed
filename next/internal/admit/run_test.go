package admit

// The run admission drills (plans/os-1dad487d.md;
// next/spec/executors.md): run.started is the spending gate's first
// real customer — refused with no open valid reservation, admitted
// against one on the active fence, once per fence — and run.settled
// aggregates once per started fence, prior fences included.

import (
	"crypto/ed25519"
	"fmt"
	"strings"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/keyring"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
	"github.com/shaunlmason/open-seed/next/internal/version"
)

type runKeys struct {
	signer, holder, supervisor ed25519.PrivateKey
}

// runFixture enrolls a claim worker and a supervisor, files and
// specifies c-1 (budget small = 100), and has the holder take the
// claim.
func runFixture(t *testing.T) (*Context, runKeys, func(priv ed25519.PrivateKey, v, verb, subject, payload string) *Context) {
	t.Helper()
	store, resolve, signer := seededStore(t)
	k := runKeys{signer: signer, holder: fixtureKey(t, 2), supervisor: fixtureKey(t, 11)}
	all := []ed25519.PrivateKey{k.signer, k.holder, k.supervisor}
	loose := func(fp string) (ed25519.PublicKey, bool) {
		for _, p := range all {
			if fpOf(t, p) == fp {
				return p.Public().(ed25519.PublicKey), true
			}
		}
		return resolve(fp)
	}
	appendSigned(t, store, loose, signer, ledger.UpgradeVerb, "system", `{"to": "`+version.Seed1+`"}`)
	for _, e := range []struct {
		key  ed25519.PrivateKey
		name string
		cap  string
	}{{k.holder, "holder", keyring.CapClaim}, {k.supervisor, "supervisor", keyring.CapSupervise}} {
		appendSignedV(t, store, loose, signer, version.Seed1, keyring.VerbEnrolled, fpOf(t, e.key), enrollBody(t, e.key, "agent", e.name))
		appendSignedV(t, store, loose, signer, version.Seed1, keyring.VerbGranted, fpOf(t, e.key), `{"capability": "`+e.cap+`"}`)
	}
	step := func(priv ed25519.PrivateKey, v, verb, subject, payload string) *Context {
		t.Helper()
		appendSignedV(t, store, loose, priv, v, verb, subject, payload)
		c, err := ContextAt(store)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	ctx := step(signer, version.Seed1, "intent.filed", "c-1", filedBody)
	ctx = step(signer, version.Seed1, "contract.specified", "c-1", specBody)
	ctx = step(k.holder, version.Seed1, "claim.taken", "c-1", `{}`)
	return ctx, k, step
}

// conformance: III.H — execution is fenced to the reservation:
// run.started is the spending-verb table's first entry, and no start
// admits without an open valid reservation.
func TestRunAdmissionMatrix(t *testing.T) {
	ctx, k, step := runFixture(t)
	fence := fenceOf(t, ctx, "c-1")
	startBody := func(res string) string {
		return fmt.Sprintf(`{"fence": %q, "reservation": %q}`, fence, res)
	}

	// The spending gate fires first: no reservation exists at all.
	if err := Check(ctx, draftV(t, k.supervisor, version.Seed1, "run.started", "c-1", startBody("3"), ctx.Tip)); err == nil || !strings.Contains(err.Error(), "spending verbs require an admitted budget.reserve") {
		t.Fatalf("a start with no reservation on the subject refuses at the spending gate: %v", err)
	}

	// Two reservations: one to close (specific-citation drills), one
	// to stay open (so the gate passes and the run rule speaks).
	ctx = step(k.holder, version.Seed1, "budget.reserve", "c-1", reserveBody("10", fence))
	closedRes := fmt.Sprintf("%d", ctx.Count-1)
	ctx = step(k.holder, version.Seed1, "budget.reserve", "c-1", reserveBody("10", fence))
	openRes := fmt.Sprintf("%d", ctx.Count-1)
	ctx = step(k.holder, version.Seed1, "budget.settle", "c-1", `{"reservation": "`+closedRes+`", "actuals": "10", "fence": "`+fence+`"}`)

	// Lanes and shape.
	if err := Check(ctx, draftV(t, k.holder, version.Seed1, "run.started", "c-1", startBody(openRes), ctx.Tip)); err == nil || !strings.Contains(err.Error(), "not granted any of") {
		t.Fatalf("the claim lane is out of the run lanes: %v", err)
	}
	for name, body := range map[string]string{
		"extra key":            `{"fence": "` + fence + `", "reservation": "` + openRes + `", "note": "x"}`,
		"dangling res":         startBody("2"),
		"closed res":           startBody(closedRes),
		"non-position res":     startBody("soon"),
		"not the active fence": `{"fence": "1", "reservation": "` + openRes + `"}`,
	} {
		if err := Check(ctx, draftV(t, k.supervisor, version.Seed1, "run.started", "c-1", body, ctx.Tip)); err == nil {
			t.Fatalf("%s must refuse", name)
		}
	}

	// The laundering shape (review finding on the task PR): a
	// raw-pushed start by a boundary-invalid signer neither blocks
	// the legitimate supervisor's start nor satisfies a settle.
	ctx = step(k.holder, version.Seed1, "run.started", "c-1", startBody(openRes))
	if err := Check(ctx, draftV(t, k.supervisor, version.Seed1, "run.settled", "c-1", `{"fence": "`+fence+`", "units": "1", "lines": "1"}`, ctx.Tip)); err == nil || !strings.Contains(err.Error(), "no admitted run.started") {
		t.Fatalf("a raw invalid start satisfies no settle: %v", err)
	}

	// The happy path admits once per fence; the supervisor and the
	// operator both stand in the lane, and the raw invalid start
	// blocked neither.
	if err := Check(ctx, draftV(t, k.signer, version.Seed1, "run.started", "c-1", startBody(openRes), ctx.Tip)); err != nil {
		t.Fatalf("the operator's start admits past the raw invalid one: %v", err)
	}
	ctx = step(k.supervisor, version.Seed1, "run.started", "c-1", startBody(openRes))
	if err := Check(ctx, draftV(t, k.supervisor, version.Seed1, "run.started", "c-1", startBody(openRes), ctx.Tip)); err == nil || !strings.Contains(err.Error(), "one run per claim window") {
		t.Fatalf("a second start on the fence refuses: %v", err)
	}

	// Settles: refuse on a start-less fence, admit on the started
	// one (even after the window closes), once.
	settle := func(f, units string) string {
		return fmt.Sprintf(`{"fence": %q, "units": %q, "lines": "2"}`, f, units)
	}
	if err := Check(ctx, draftV(t, k.supervisor, version.Seed1, "run.settled", "c-1", settle("9", "5"), ctx.Tip)); err == nil || !strings.Contains(err.Error(), "not an applied claim position") {
		t.Fatalf("a settle citing a dangling fence refuses: %v", err)
	}
	ctx = step(k.holder, version.Seed1, "claim.released", "c-1", fmt.Sprintf(`{"fence": %q, "packet": %s}`, fence, minPacket))
	ctx = step(k.holder, version.Seed1, "claim.taken", "c-1", `{}`)
	fence2 := fenceOf(t, ctx, "c-1")
	if err := Check(ctx, draftV(t, k.supervisor, version.Seed1, "run.settled", "c-1", settle(fence2, "5"), ctx.Tip)); err == nil || !strings.Contains(err.Error(), "no admitted run.started") {
		t.Fatalf("a settle on a start-less fence refuses: %v", err)
	}

	// The temporal laundering shape (review finding on the follow-up
	// PR): validity is judged against the prefix the start appended
	// onto. A raw capability-bearing start citing a reservation that
	// lands only later validates nothing.
	futureRes := fmt.Sprintf("%d", ctx.Count+1)
	start2 := func(res string) string {
		return fmt.Sprintf(`{"fence": %q, "reservation": %q}`, fence2, res)
	}
	ctx = step(k.supervisor, version.Seed1, "run.started", "c-1", start2(futureRes))
	ctx = step(k.holder, version.Seed1, "budget.reserve", "c-1", reserveBody("10", fence2))
	if got := fmt.Sprintf("%d", ctx.Count-1); got != futureRes {
		t.Fatalf("the drill's reservation must land at the cited position: %s != %s", got, futureRes)
	}
	if err := Check(ctx, draftV(t, k.supervisor, version.Seed1, "run.settled", "c-1", settle(fence2, "5"), ctx.Tip)); err == nil || !strings.Contains(err.Error(), "no admitted run.started") {
		t.Fatalf("a start citing a later-appended reservation satisfies no settle: %v", err)
	}
	// Nor does one citing a reservation already closed at the start's
	// own position.
	ctx = step(k.holder, version.Seed1, "budget.release", "c-1", `{"reservation": "`+futureRes+`", "fence": "`+fence2+`"}`)
	ctx = step(k.supervisor, version.Seed1, "run.started", "c-1", start2(futureRes))
	if err := Check(ctx, draftV(t, k.supervisor, version.Seed1, "run.settled", "c-1", settle(fence2, "5"), ctx.Tip)); err == nil || !strings.Contains(err.Error(), "no admitted run.started") {
		t.Fatalf("a start citing an already-closed reservation satisfies no settle: %v", err)
	}
	// Neither temporal raw start blocks the legitimate one.
	ctx = step(k.holder, version.Seed1, "budget.reserve", "c-1", reserveBody("10", fence2))
	if err := Check(ctx, draftV(t, k.supervisor, version.Seed1, "run.started", "c-1", start2(fmt.Sprintf("%d", ctx.Count-1)), ctx.Tip)); err != nil {
		t.Fatalf("the legitimate start admits past the temporal raw ones: %v", err)
	}
	if err := Check(ctx, draftV(t, k.supervisor, version.Seed1, "run.settled", "c-1", settle(fence, "-3"), ctx.Tip)); err == nil {
		t.Fatal("negative units refuse")
	}
	if err := Check(ctx, draftV(t, k.supervisor, version.Seed1, "run.settled", "c-1", settle(fence, "5"), ctx.Tip)); err != nil {
		t.Fatalf("the prior started fence settles after its window closed: %v", err)
	}
	ctx = step(k.supervisor, version.Seed1, "run.settled", "c-1", settle(fence, "5"))
	if err := Check(ctx, draftV(t, k.supervisor, version.Seed1, "run.settled", "c-1", settle(fence, "5"), ctx.Tip)); err == nil || !strings.Contains(err.Error(), "one run, one aggregate") {
		t.Fatalf("a second settle on the fence refuses: %v", err)
	}
}
