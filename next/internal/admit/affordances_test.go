package admit

// The affordance drills (plans/os-f5551001.md; charter III.I rows 1
// and 2): the catalog is complete against the spec verb table, the
// lifecycle walk lists every catalog verb somewhere legal (the
// completeness half) and never lists the curated illegal set (the
// soundness half), and the output is sorted and deterministic. The
// listed-iff-admits property is the computation itself — every probe
// runs the enforcing Check — so these drills pin the corpus item 2's
// regression-class harness generalizes.

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/keyring"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
	"github.com/shaunlmason/open-seed/next/internal/version"
)

// specCatalogVerbs mirrors the actors.md spec table plus the
// ungoverned active-standing verb, the same literal-list pinning the
// keyring completeness test uses: drift between this list, the
// catalog, and the spec table is a test failure.
var specCatalogVerbs = []string{
	"system.halt.declared", "system.halt.lifted", "system.protocol.upgraded",
	"system.checkpoint",
	"actor.enrolled", "actor.granted", "actor.suspended", "actor.revoked",
	"intent.filed", "contract.specified", "contract.blocked",
	"contract.unblocked", "contract.cancelled", "contract.returned",
	"claim.taken", "claim.released", "claim.parked", "claim.reaped",
	"submission.made", "plan.proposed", "plan.approved",
	"progress.milestone", "wedge.declared",
	"offer.published", "budget.reserve", "budget.settle", "budget.release",
	"run.started", "run.settled", "run.interrupted",
	"verdict.rendered", "check.sealed",
	"merge.requested", "merge.observed", "merge.overridden",
	"message.sent",
}

func TestAffordanceCatalogCompleteness(t *testing.T) {
	inCatalog := map[string]bool{}
	for _, p := range affordanceCatalog {
		if p.synth == nil {
			t.Errorf("%s has no payload synthesizer", p.verb)
			continue
		}
		v := &probeView{now: "2026-09-01T00:00:00Z", expires: "2026-09-01T01:00:00Z",
			fence: "0", reservation: "0", submission: "0", verdict: "0", packet: probePacket}
		if body := p.synth(v); !json.Valid([]byte(body)) {
			t.Errorf("%s synthesizes invalid JSON: %s", p.verb, body)
		}
		inCatalog[p.verb] = true
	}
	for _, verb := range specCatalogVerbs {
		if !inCatalog[verb] {
			t.Errorf("%s is in the spec table but missing from the catalog", verb)
		}
	}
	if len(inCatalog) != len(specCatalogVerbs) {
		t.Errorf("catalog holds %d verbs, the spec list %d — an extra catalog verb needs a spec row", len(inCatalog), len(specCatalogVerbs))
	}
}

// conformance: III.I — the walk drives one chain through birth,
// claim, spend, run, review, verdicts pass and fail, block, and
// halt, querying affordances for each lane along the way: every
// catalog verb except the actor.enrolled carve-out is listed at a
// position where it is legal, the curated illegal set never appears,
// and the output is sorted and stable.
func TestAffordancesWalk(t *testing.T) {
	store, resolve, signer := seededStore(t)
	keys := map[string]ed25519.PrivateKey{
		"holder":     fixtureKey(t, 2),
		"supervisor": fixtureKey(t, 3),
		"verifier":   fixtureKey(t, 4),
		"sealer":     fixtureKey(t, 5),
		"observer":   fixtureKey(t, 6),
	}
	loose := func(fp string) (ed25519.PublicKey, bool) {
		for _, p := range keys {
			if fpOf(t, p) == fp {
				return p.Public().(ed25519.PublicKey), true
			}
		}
		return resolve(fp)
	}
	ctxAt := func() *Context {
		c, err := ContextAt(store)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	union := map[string]bool{}
	list := func(key ed25519.PrivateKey, subject string) []string {
		out := Affordances(ctxAt(), key, subject)
		if !slices.IsSorted(out) {
			t.Fatalf("affordances must be sorted: %v", out)
		}
		for _, verb := range out {
			union[verb] = true
		}
		return out
	}
	has := func(list []string, verb string) bool { return slices.Contains(list, verb) }
	step := func(priv ed25519.PrivateKey, verb, subject, payload string) {
		t.Helper()
		appendSignedV(t, store, loose, priv, version.Seed1, verb, subject, payload)
	}

	// Pre-upgrade: the root can upgrade the protocol.
	if l := list(signer, "system"); !has(l, "system.protocol.upgraded") {
		t.Fatalf("the root lists the upgrade on the fresh chain: %v", l)
	}
	appendSigned(t, store, loose, signer, ledger.UpgradeVerb, "system", `{"to": "`+version.Seed1+`"}`)
	for name, cap := range map[string]string{
		"holder": keyring.CapClaim, "supervisor": keyring.CapSupervise,
		"verifier": keyring.CapVerdict, "sealer": keyring.CapSealer,
		"observer": keyring.CapObserver,
	} {
		step(signer, keyring.VerbEnrolled, fpOf(t, keys[name]), enrollBody(t, keys[name], "agent", name))
		step(signer, keyring.VerbGranted, fpOf(t, keys[name]), `{"capability": "`+cap+`"}`)
	}

	// Actor management and the system surface, for the operator.
	if l := list(signer, fpOf(t, keys["holder"])); !has(l, "actor.granted") || !has(l, "actor.suspended") || !has(l, "actor.revoked") {
		t.Fatalf("the operator lists actor management on an enrolled actor: %v", l)
	}
	sys := list(signer, "system")
	if !has(sys, "system.halt.declared") || !has(sys, "system.checkpoint") {
		t.Fatalf("the operator lists halt and checkpoint on system: %v", sys)
	}

	// Birth: filing is listed before the subject exists; nothing
	// claim-shaped is.
	fresh := list(signer, "c-1")
	if !has(fresh, "intent.filed") {
		t.Fatalf("filing lists on a fresh subject: %v", fresh)
	}
	if has(fresh, "claim.taken") || has(fresh, "run.started") {
		t.Fatalf("claim and run must not list before birth: %v", fresh)
	}
	step(signer, "intent.filed", "c-1", filedBody)
	if l := list(signer, "c-1"); !has(l, "contract.specified") {
		t.Fatalf("specification lists on a filed subject: %v", l)
	}
	step(signer, "contract.specified", "c-1", specBody)

	// Ready: the worker can claim, the supervisor can offer, the
	// sealer can commit, the operator can cancel; the run verbs
	// cannot fire yet.
	if l := list(keys["holder"], "c-1"); !has(l, "claim.taken") {
		t.Fatalf("the holder lists the claim at ready: %v", l)
	}
	if l := list(keys["supervisor"], "c-1"); !has(l, "offer.published") || has(l, "run.started") {
		t.Fatalf("the supervisor lists the offer and no run at ready: %v", l)
	}
	if l := list(keys["sealer"], "c-1"); !has(l, "check.sealed") {
		t.Fatalf("the sealer lists the commitment at ready: %v", l)
	}
	if l := list(signer, "c-1"); !has(l, "contract.cancelled") {
		t.Fatalf("the operator lists cancellation at ready: %v", l)
	}

	// Claimed: the holder's window opens; a second claim does not
	// list; the verifier has nothing to judge.
	step(keys["holder"], "claim.taken", "c-1", `{}`)
	held := list(keys["holder"], "c-1")
	for _, verb := range []string{"claim.released", "claim.parked", "submission.made", "plan.proposed", "progress.milestone", "budget.reserve", "message.sent"} {
		if !has(held, verb) {
			t.Fatalf("the holder lists %s inside the window: %v", verb, held)
		}
	}
	if has(held, "claim.taken") {
		t.Fatalf("a second claim must not list while held: %v", held)
	}
	if l := list(signer, "c-1"); !has(l, "claim.reaped") {
		t.Fatalf("the operator lists the reap on a held subject: %v", l)
	}
	if l := list(keys["verifier"], "c-1"); has(l, "verdict.rendered") {
		t.Fatalf("no submission stands, so the verdict must not list: %v", l)
	}

	// Spend: reserve, then the closes and the supervisor's run.
	step(keys["holder"], "budget.reserve", "c-1", reserveBody("2", fenceOf(t, ctxAt(), "c-1")))
	if l := list(keys["holder"], "c-1"); !has(l, "budget.settle") || !has(l, "budget.release") {
		t.Fatalf("the holder lists the closes over an open reservation: %v", l)
	}
	sup := list(keys["supervisor"], "c-1")
	if !has(sup, "run.started") || !has(sup, "run.interrupted") {
		t.Fatalf("the supervisor lists the run verbs over an open reservation: %v", sup)
	}
	fence := fenceOf(t, ctxAt(), "c-1")
	step(keys["supervisor"], "run.started", "c-1", `{"fence": "`+fence+`", "reservation": "`+reservationOf(t, ctxAt(), "c-1")+`"}`)
	sup = list(keys["supervisor"], "c-1")
	if !has(sup, "run.settled") {
		t.Fatalf("the supervisor lists the settle after the start: %v", sup)
	}
	if has(sup, "run.started") {
		t.Fatalf("one run per window: a second start must not list: %v", sup)
	}
	if l := list(signer, "c-1"); !has(l, "plan.approved") {
		t.Fatalf("the operator lists plan approval: %v", l)
	}

	// Review: the verifier's window opens; the pass chain follows.
	step(keys["holder"], "submission.made", "c-1", `{"fence": "`+fence+`", "packet": `+minPacket+`}`)
	if l := list(keys["verifier"], "c-1"); !has(l, "verdict.rendered") {
		t.Fatalf("the verifier lists the verdict on review: %v", l)
	}
	if l := list(keys["holder"], "c-1"); has(l, "verdict.rendered") {
		t.Fatalf("the holder is no verifier — the verdict must not list for the claim lane: %v", l)
	}
	step(keys["verifier"], "verdict.rendered", "c-1", `{"verdict": "pass", "receipt": "`+zeros64+`", "submission": "`+submissionOf(t, ctxAt(), "c-1")+`", "independence": "L1"}`)
	if l := list(signer, "c-1"); !has(l, "merge.requested") {
		t.Fatalf("the operator lists the merge request over the pass verdict: %v", l)
	}
	step(signer, "merge.requested", "c-1", `{"verdict": "`+verdictOf(t, ctxAt(), "c-1")+`"}`)
	if l := list(keys["observer"], "c-1"); !has(l, "merge.observed") {
		t.Fatalf("the observer lists the forge fact after the request: %v", l)
	}

	// The fail path on a second subject: return and override list
	// only over a standing fail.
	for _, s := range [][3]string{
		{"intent.filed", "c-2", filedBody},
		{"contract.specified", "c-2", specBody},
	} {
		step(signer, s[0], s[1], s[2])
	}
	step(keys["holder"], "claim.taken", "c-2", `{}`)
	fence2 := fenceOf(t, ctxAt(), "c-2")
	step(keys["holder"], "submission.made", "c-2", `{"fence": "`+fence2+`", "packet": `+minPacket+`}`)
	if l := list(signer, "c-2"); has(l, "contract.returned") || has(l, "merge.overridden") {
		t.Fatalf("no fail stands, so return and override must not list: %v", l)
	}
	step(keys["verifier"], "verdict.rendered", "c-2", `{"verdict": "fail", "receipt": "`+zeros64+`", "submission": "`+submissionOf(t, ctxAt(), "c-2")+`", "independence": "L1"}`)
	if l := list(signer, "c-2"); !has(l, "contract.returned") || !has(l, "merge.overridden") {
		t.Fatalf("the operator lists return and override over the fail: %v", l)
	}
	if l := list(signer, "c-2"); !has(l, "wedge.declared") {
		t.Fatalf("the operator lists the wedge: %v", l)
	}

	// The blocked path on a third subject (blocked's one source
	// state is ready).
	step(signer, "intent.filed", "c-3", filedBody)
	step(signer, "contract.specified", "c-3", specBody)
	if l := list(signer, "c-3"); !has(l, "contract.blocked") {
		t.Fatalf("the operator lists blocking at ready: %v", l)
	}
	step(signer, "contract.blocked", "c-3", `{}`)
	if l := list(signer, "c-3"); !has(l, "contract.unblocked") {
		t.Fatalf("the operator lists unblocking on a blocked subject: %v", l)
	}

	// Halt last: under the halt the work verbs stop listing, and the
	// lift remains.
	step(signer, "system.halt.declared", "system", `{"reason": "walk"}`)
	if l := list(signer, "system"); !has(l, "system.halt.lifted") {
		t.Fatalf("the operator lists the lift under halt: %v", l)
	}
	if l := Affordances(ctxAt(), keys["holder"], "c-3"); len(l) != 0 {
		t.Fatalf("under halt the worker's list empties: %v", l)
	}
	step(signer, "system.halt.lifted", "system", `{}`)

	// Determinism at the final position.
	a := Affordances(ctxAt(), keys["holder"], "c-1")
	b := Affordances(ctxAt(), keys["holder"], "c-1")
	if !slices.Equal(a, b) {
		t.Fatalf("affordances must be deterministic: %v vs %v", a, b)
	}

	// Completeness: every catalog verb except the actor.enrolled
	// carve-out (its valid payload needs the queried subject's public
	// key, which no prober holds) was listed somewhere legal.
	var missing []string
	for _, p := range affordanceCatalog {
		if p.verb == "actor.enrolled" {
			continue
		}
		if !union[p.verb] {
			missing = append(missing, p.verb)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("catalog verbs never listed anywhere legal across the walk: %v", missing)
	}
}

const zeros64 = "0000000000000000000000000000000000000000000000000000000000000000"

func reservationOf(t *testing.T, ctx *Context, subject string) string {
	t.Helper()
	s, ok := ctx.Lifecycle.State(subject)
	if !ok || len(s.Reservations) == 0 {
		t.Fatalf("no reservation on %s", subject)
	}
	view := BudgetViewAt(ctx.Records, ctx.Table, subject, s)
	if len(view.Open) == 0 {
		t.Fatalf("no open reservation on %s", subject)
	}
	return fmt.Sprintf("%d", view.Open[0].Pos)
}

func submissionOf(t *testing.T, ctx *Context, subject string) string {
	t.Helper()
	s, ok := ctx.Lifecycle.State(subject)
	if !ok || s.Submission == nil {
		t.Fatalf("no submission on %s", subject)
	}
	return fmt.Sprintf("%d", s.Submission.Pos)
}

func verdictOf(t *testing.T, ctx *Context, subject string) string {
	t.Helper()
	s, ok := ctx.Lifecycle.State(subject)
	if !ok || s.Verdict == nil {
		t.Fatalf("no verdict on %s", subject)
	}
	return fmt.Sprintf("%d", s.Verdict.Pos)
}
