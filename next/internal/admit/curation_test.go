package admit

// The staged curation stores at the boundary (plans/os-f30ee0d3.md
// AC1 through AC5): the dead end inside the holder's window, the
// proposal's support floor and its curate-only row, the grant-level
// disjointness, the promotion's re-judged citation, the curator's
// raise, and the curator's reachable set derived from the boundary.

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/curation"
	"github.com/shaunlmason/open-seed/next/internal/keyring"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
	"github.com/shaunlmason/open-seed/next/internal/transition"
	"github.com/shaunlmason/open-seed/next/internal/version"
)

const trivialFiling = `{"intent": "drill", "tier": "trivial", "budget": "small", "routing": "core"}`

// findingPacket is a packet carrying one finding: a deliberate exit
// that is a stage-one observation.
const findingPacket = `{"acceptance": ["resume"], "decisions": [], "base": "1234567..1234567", "refs": [], "findings": [{"tried": "x", "outcome": "y"}]}`

type curationStand struct {
	ctx                                          *Context
	root, worker, worker2, curator, observer     ed25519.PrivateKey
	verifier, dispatcher                         ed25519.PrivateKey
	step                                         func(priv ed25519.PrivateKey, v, verb, subject, payload string) *Context
	deadEnd1, deadEnd1b, park2, submission3, hyp int
	raise5                                       int
	claim, id                                    string
}

func deadEndBody(fence string) string {
	return fmt.Sprintf(`{"fence": %q, "tried": "x", "outcome": "y", "condition": "z", "environment": "w"}`, fence)
}

// curationFixture stages the record the drills judge against: three
// contracts worked by the worker (one with a dead end, one parked with
// a finding, one submitted and failed), and the curation lanes
// enrolled with their grants.
func curationFixture(t *testing.T) *curationStand {
	t.Helper()
	store, resolve, root := seededStore(t)
	st := &curationStand{root: root, worker: fixtureKey(t, 2), claim: "retry the fetch once"}
	st.id = curation.HypothesisID(st.claim)
	st.worker2, st.curator, st.observer, st.verifier, st.dispatcher = fixtureKey(t, 11), fixtureKey(t, 12), fixtureKey(t, 13), fixtureKey(t, 14), fixtureKey(t, 15)
	keys := []ed25519.PrivateKey{root, st.worker, st.worker2, st.curator, st.observer, st.verifier, st.dispatcher}
	loose := func(fp string) (ed25519.PublicKey, bool) {
		for _, p := range keys {
			if fpOf(t, p) == fp {
				return p.Public().(ed25519.PublicKey), true
			}
		}
		return resolve(fp)
	}
	appendSigned(t, store, loose, root, ledger.UpgradeVerb, "system", `{"to": "`+version.Seed1+`"}`)
	step := func(priv ed25519.PrivateKey, v, verb, subject, payload string) *Context {
		t.Helper()
		appendSignedV(t, store, loose, priv, v, verb, subject, payload)
		c, err := ContextAt(store)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	st.step = step
	worker := st.worker
	ctx := step(root, version.Seed1, keyring.VerbEnrolled, fpOf(t, worker), enrollBody(t, worker, "agent", "worker"))
	ctx = step(root, version.Seed1, keyring.VerbGranted, fpOf(t, worker), `{"capability": "`+keyring.CapClaim+`"}`)
	for _, e := range []struct {
		key  ed25519.PrivateKey
		name string
		cap  string
	}{
		{st.worker2, "worker2", keyring.CapClaim}, {st.curator, "curator", keyring.CapCurate},
		{st.observer, "observer", keyring.CapObserver}, {st.verifier, "verifier", keyring.CapVerdict},
		{st.dispatcher, "dispatcher", keyring.CapDispatch},
	} {
		ctx = step(root, version.Seed1, keyring.VerbEnrolled, fpOf(t, e.key), enrollBody(t, e.key, "agent", e.name))
		ctx = step(root, version.Seed1, keyring.VerbGranted, fpOf(t, e.key), `{"capability": "`+e.cap+`"}`)
	}
	open := func(subject string) *Context {
		ctx = step(root, version.Seed1, "intent.filed", subject, trivialFiling)
		ctx = step(root, version.Seed1, "contract.specified", subject, specBody)
		return step(worker, version.Seed1, "claim.taken", subject, `{}`)
	}
	// c-4: specified and never claimed, the ready contract.
	ctx = step(root, version.Seed1, "intent.filed", "c-4", trivialFiling)
	ctx = step(root, version.Seed1, "contract.specified", "c-4", specBody)
	// c-1: two dead ends inside the window, both admitted.
	ctx = open("c-1")
	for _, at := range []*int{&st.deadEnd1, &st.deadEnd1b} {
		body := deadEndBody(activeFence(t, ctx, "c-1"))
		if err := Check(ctx, draftV(t, worker, version.Seed1, curation.DeadEndVerb, "c-1", body, ctx.Tip)); err != nil {
			t.Fatalf("the holder's dead end inside its window admits: %v", err)
		}
		*at = ctx.Count
		ctx = step(worker, version.Seed1, curation.DeadEndVerb, "c-1", body)
	}
	// c-2: parked with a finding.
	ctx = open("c-2")
	st.park2 = ctx.Count
	ctx = step(worker, version.Seed1, "claim.parked", "c-2", `{"fence": "`+activeFence(t, ctx, "c-2")+`", "packet": `+findingPacket+`}`)
	// c-5: held, then a raise carrying a finding: a question, not an
	// exit, so no observation.
	ctx = open("c-5")
	st.raise5 = ctx.Count
	ctx = step(worker, version.Seed1, "escalation.raised", "c-5", `{"fence": "`+activeFence(t, ctx, "c-5")+`", "packet": `+findingPacket+`, "escalation": {"question": "which?", "options": [{"id": "a", "choice": "this"}, {"id": "b", "choice": "that"}]}}`)
	// c-3: submitted with a finding, then failed.
	ctx = open("c-3")
	st.submission3 = ctx.Count
	ctx = step(worker, version.Seed1, "submission.made", "c-3", `{"fence": "`+activeFence(t, ctx, "c-3")+`", "packet": `+findingPacket+`}`)
	ctx = step(st.verifier, version.Seed1, "verdict.rendered", "c-3",
		`{"verdict": "fail", "receipt": "`+strings.Repeat("0", 64)+`", "submission": "`+fmt.Sprint(st.submission3)+`", "independence": "L1"}`)
	st.ctx = ctx
	return st
}

func (st *curationStand) proposal(support ...string) string {
	quoted := make([]string, len(support))
	for i, s := range support {
		quoted[i] = fmt.Sprintf("%q", s)
	}
	return fmt.Sprintf(`{"claim": %q, "applies_when": "the fetch is flaky", "support": [%s], "exceptions": [], "provenance": ["plans/x.md @ 0123456"]}`,
		st.claim, strings.Join(quoted, ", "))
}

func cite(contract string, pos int) string { return fmt.Sprintf("%s@%d", contract, pos) }

// conformance: AC1 — the dead end admits from the window's holder
// citing the active fence and refuses everything else naming the part.
func TestDeadEndIsTheHoldersInsideItsWindow(t *testing.T) {
	st := curationFixture(t)
	ctx := st.ctx
	fence := activeFence(t, ctx, "c-1")
	// Outside a window the fence rule speaks first: no claim is
	// active, so the citation refuses as a fence refusal naming the
	// window's absence.
	var fe *FenceError
	if err := Check(ctx, draftV(t, st.worker, version.Seed1, curation.DeadEndVerb, "c-4", deadEndBody("0"), ctx.Tip)); !errors.As(err, &fe) || !strings.Contains(err.Error(), "no claim is active") {
		t.Fatalf("outside a window refuses naming the window: %v", err)
	}
	var ce *CurationError
	if err := Check(ctx, draftV(t, st.worker2, version.Seed1, curation.DeadEndVerb, "c-1", deadEndBody(fence), ctx.Tip)); !errors.As(err, &ce) || !strings.Contains(err.Error(), "holder") {
		t.Fatalf("a non-holder refuses naming the holder: %v", err)
	}
	if err := Check(ctx, draftV(t, st.worker, version.Seed1, curation.DeadEndVerb, "c-1", deadEndBody("0"), ctx.Tip)); !errors.As(err, &fe) {
		t.Fatalf("a stale fence refuses as a fence refusal: %v", err)
	}
	var inc *transition.IncompleteError
	if err := Check(ctx, draftV(t, st.worker, version.Seed1, curation.DeadEndVerb, "c-1",
		`{"fence": "`+fence+`", "tried": "x", "outcome": "y", "condition": "", "environment": "w"}`, ctx.Tip)); !errors.As(err, &inc) || fmt.Sprint(inc.Missing) != "[condition]" {
		t.Fatalf("a missing field refuses naming it: %v", err)
	}
	var se *curation.ShapeError
	if err := Check(ctx, draftV(t, st.worker, version.Seed1, curation.DeadEndVerb, "c-1",
		`{"fence": "`+fence+`", "tried": "x", "outcome": "y", "condition": "z", "environment": "w", "pointer": "notes.md"}`, ctx.Tip)); !errors.As(err, &se) || !strings.Contains(err.Error(), "anchored") {
		t.Fatalf("a bare-path pointer refuses: %v", err)
	}
	var oog *OutOfGrantError
	if err := Check(ctx, draftV(t, st.dispatcher, version.Seed1, curation.DeadEndVerb, "c-1", deadEndBody(fence), ctx.Tip)); !errors.As(err, &oog) {
		t.Fatalf("a dispatch-only key cannot reach the dead end: %v", err)
	}
	if err := Check(ctx, draftV(t, st.worker, version.Seed1, curation.DeadEndVerb, "c-1",
		`{"fence": "`+fence+`", "tried": "x", "outcome": "y", "condition": "z", "environment": "w", "pointer": "notes.md @ 0123456"}`, ctx.Tip)); err != nil {
		t.Fatalf("an anchored pointer admits: %v", err)
	}
}

// conformance: AC2 — the proposal admits from curate on the derived
// subject citing two admitted observations on two distinct non-failed
// contracts, and refuses each other shape, the root included.
func TestHypothesisNeedsTwoObservationsOnTwoContractsFromCurate(t *testing.T) {
	st := curationFixture(t)
	ctx := st.ctx
	good := st.proposal(cite("c-1", st.deadEnd1), cite("c-2", st.park2))
	if err := Check(ctx, draftV(t, st.curator, version.Seed1, curation.HypothesisVerb, st.id, good, ctx.Tip)); err != nil {
		t.Fatalf("two observations on two contracts from curate admit: %v", err)
	}
	var oog *OutOfGrantError
	if err := Check(ctx, draftV(t, st.worker, version.Seed1, curation.HypothesisVerb, st.id, good, ctx.Tip)); !errors.As(err, &oog) {
		t.Fatalf("a claim key refuses out of grant: %v", err)
	}
	if err := Check(ctx, draftV(t, st.root, version.Seed1, curation.HypothesisVerb, st.id, good, ctx.Tip)); !errors.As(err, &oog) || fmt.Sprint(oog.Accepted) != "[curate]" {
		t.Fatalf("a root's implicit operator standing does not reach the proposal; the accepted set is [curate] alone: %v", err)
	}
	var se *curation.ShapeError
	if err := Check(ctx, draftV(t, st.curator, version.Seed1, curation.HypothesisVerb, "h-000000000000", good, ctx.Tip)); !errors.As(err, &se) || !strings.Contains(err.Error(), st.id) {
		t.Fatalf("a subject not derived from the claim refuses naming the derived one: %v", err)
	}
	var sup *curation.SupportError
	for name, body := range map[string]string{
		"one contract":     st.proposal(cite("c-1", st.deadEnd1)),
		"two on one":       st.proposal(cite("c-1", st.deadEnd1), cite("c-1", st.deadEnd1b)),
		"not observation":  st.proposal(cite("c-1", st.deadEnd1-3), cite("c-2", st.park2)),
		"failed contract":  st.proposal(cite("c-1", st.deadEnd1), cite("c-3", st.submission3)),
		"wrong contract":   st.proposal(cite("c-2", st.deadEnd1), cite("c-2", st.park2)),
		"cited twice":      st.proposal(cite("c-1", st.deadEnd1), cite("c-1", st.deadEnd1), cite("c-2", st.park2)),
		"out of the chain": st.proposal(cite("c-1", st.deadEnd1), cite("c-2", 99999)),
		"a raise":          st.proposal(cite("c-1", st.deadEnd1), cite("c-5", st.raise5)),
	} {
		if err := Check(ctx, draftV(t, st.curator, version.Seed1, curation.HypothesisVerb, st.id, body, ctx.Tip)); !errors.As(err, &sup) {
			t.Errorf("%s: refuses as a support refusal, got %v", name, err)
		}
	}
	if err := Check(ctx, draftV(t, st.curator, version.Seed1, curation.HypothesisVerb, st.id, st.proposal(cite("c-1", st.deadEnd1)), ctx.Tip)); err == nil || !strings.Contains(err.Error(), "2 distinct") {
		t.Fatalf("the refusal names the floor: %v", err)
	}
	// The duplicate: one claim derives one subject.
	ctx = st.step(st.curator, version.Seed1, curation.HypothesisVerb, st.id, good)
	var ce *CurationError
	if err := Check(ctx, draftV(t, st.curator, version.Seed1, curation.HypothesisVerb, st.id, good, ctx.Tip)); !errors.As(err, &ce) || !strings.Contains(err.Error(), "proposed at position") {
		t.Fatalf("a re-proposal of an admitted claim refuses as a duplicate: %v", err)
	}
}

// conformance: AC3 — curate is disjoint from claim and operator at
// the grant, both directions, as chain validity; a root included.
func TestCurateIsDisjointAtTheGrant(t *testing.T) {
	st := curationFixture(t)
	ctx := st.ctx
	for name, draft := range map[string][2]string{
		"curate onto claim":    {fpOf(t, st.worker), keyring.CapCurate},
		"claim onto curate":    {fpOf(t, st.curator), keyring.CapClaim},
		"operator onto curate": {fpOf(t, st.curator), keyring.CapOperator},
		"curate onto a root":   {fpOf(t, st.root), keyring.CapCurate},
	} {
		err := Check(ctx, draftV(t, st.root, version.Seed1, keyring.VerbGranted, draft[0], `{"capability": "`+draft[1]+`"}`, ctx.Tip))
		if err == nil || !strings.Contains(err.Error(), "disjoint") {
			t.Errorf("%s: refuses at the grant naming the disjointness: %v", name, err)
		}
	}
}

// conformance: AC4 — the promotion admits from observer citing the
// admitted hypothesis on its own subject, refuses every other
// citation, and a raw-pushed promotion citing nothing folds unbound.
func TestPromotionCitesAnAdmittedHypothesis(t *testing.T) {
	st := curationFixture(t)
	ctx := st.ctx
	st.hyp = ctx.Count
	ctx = st.step(st.curator, version.Seed1, curation.HypothesisVerb, st.id, st.proposal(cite("c-1", st.deadEnd1), cite("c-2", st.park2)))
	lesson := func(path, hyp, pr string) string {
		return fmt.Sprintf(`{"lesson": %q, "hypothesis": %q, "pr": %q}`, path, hyp, pr)
	}
	path := curation.LessonsDir + "/retry.md @ 0123456"
	good := lesson(path, cite(st.id, st.hyp), "pr/9 @ 0123456")
	if err := Check(ctx, draftV(t, st.observer, version.Seed1, curation.LessonVerb, st.id, good, ctx.Tip)); err != nil {
		t.Fatalf("the observer promotes the admitted hypothesis: %v", err)
	}
	var ce *CurationError
	if err := Check(ctx, draftV(t, st.observer, version.Seed1, curation.LessonVerb, st.id, lesson(path, cite(st.id, st.hyp+1), "pr/9 @ 0123456"), ctx.Tip)); !errors.As(err, &ce) {
		t.Fatalf("citing a position that is no hypothesis refuses: %v", err)
	}
	var se *curation.ShapeError
	if err := Check(ctx, draftV(t, st.observer, version.Seed1, curation.LessonVerb, "h-000000000000", good, ctx.Tip)); !errors.As(err, &se) || !strings.Contains(err.Error(), "not this subject") {
		t.Fatalf("a mismatched subject refuses: %v", err)
	}
	if err := Check(ctx, draftV(t, st.observer, version.Seed1, curation.LessonVerb, st.id, lesson(curation.LessonsDir+"/retry.md", cite(st.id, st.hyp), "pr/9 @ 0123456"), ctx.Tip)); !errors.As(err, &se) {
		t.Fatalf("a bare path refuses: %v", err)
	}
	var oog *OutOfGrantError
	if err := Check(ctx, draftV(t, st.curator, version.Seed1, curation.LessonVerb, st.id, good, ctx.Tip)); !errors.As(err, &oog) {
		t.Fatalf("a curate key cannot promote: %v", err)
	}
	// A raw-pushed proposal never passed the boundary (a claim key
	// signed it), so a promotion citing it refuses: no stage skips.
	other := "a different claim"
	otherID := curation.HypothesisID(other)
	rawPos := ctx.Count
	ctx = st.step(st.worker, version.Seed1, curation.HypothesisVerb, otherID,
		strings.Replace(st.proposal(cite("c-1", st.deadEnd1), cite("c-2", st.park2)), st.claim, other, 1))
	if err := Check(ctx, draftV(t, st.observer, version.Seed1, curation.LessonVerb, otherID, lesson(path, cite(otherID, rawPos), "pr/9 @ 0123456"), ctx.Tip)); !errors.As(err, &ce) {
		t.Fatalf("a promotion citing a proposal that never passed the boundary refuses: %v", err)
	}
	ctx = st.step(st.observer, version.Seed1, curation.LessonVerb, "h-000000000000", lesson(path, "h-000000000000@0", "pr/9 @ 0123456"))
	if fold := curation.Fold(ctx.Records); len(fold.Unbound) != 1 || len(fold.Lessons) != 0 {
		t.Fatalf("a raw-pushed promotion citing no hypothesis folds unbound, never as a lesson: %+v", fold.Unbound)
	}
}

// conformance: AC5 — the curator raises exactly as any raiser, and its
// reachable set, derived from the boundary, is the proposal, the raise
// and the standing-only relay, each a named residual.
func TestCuratorRaisesAndItsReachableSetIsNamed(t *testing.T) {
	st := curationFixture(t)
	ctx := st.ctx
	raise := `{"packet": ` + minPacket + `, "escalation": {"question": "which?", "options": [{"id": "a", "choice": "this"}, {"id": "b", "choice": "that"}]}}`
	// c-4 is ready, c-2 blocked by its park, c-1 held: the curator's
	// answer on each is the dispatcher's.
	for _, subject := range []string{"c-4", "c-2", "c-1"} {
		curatorErr := Check(ctx, draftV(t, st.curator, version.Seed1, "escalation.raised", subject, raise, ctx.Tip))
		dispatchErr := Check(ctx, draftV(t, st.dispatcher, version.Seed1, "escalation.raised", subject, raise, ctx.Tip))
		if (curatorErr == nil) != (dispatchErr == nil) {
			t.Fatalf("on %s the curator raises exactly as the dispatcher does: %v vs %v", subject, curatorErr, dispatchErr)
		}
	}
	if err := Check(ctx, draftV(t, st.curator, version.Seed1, "escalation.raised", "c-4", raise, ctx.Tip)); err != nil {
		t.Fatalf("the curator raises on a ready contract: %v", err)
	}
	if err := Check(ctx, draftV(t, st.curator, version.Seed1, "escalation.raised", "c-2", raise, ctx.Tip)); err == nil {
		t.Fatal("a blocked contract refuses the raise from the curator as from any raiser")
	}

	named := map[string]bool{}
	for _, r := range loadResidualsFrom(t, "curator-residuals.json") {
		named[r.Verb] = true
	}
	seen := map[string]bool{}
	for _, subject := range []string{"c-1", "c-2", "c-3", "c-4"} {
		for _, verb := range Affordances(ctx, st.curator, subject) {
			seen[verb] = true
		}
	}
	for verb := range seen {
		if !named[verb] {
			t.Errorf("the curator can reach %s and the residual table does not name it", verb)
		}
	}
	for verb := range named {
		if !seen[verb] {
			t.Errorf("the residual table names %s and the curator cannot reach it", verb)
		}
	}
	// Before two observations stand, the proposal is not an affordance:
	// a fresh stand holds the curator and no observation yet.
	early := curationEarly(t)
	for _, verb := range Affordances(early.ctx, early.curator, "c-1") {
		if verb == curation.HypothesisVerb {
			t.Fatal("with nothing citable the proposal is invisible")
		}
	}
}

// curationEarly is the stand before any contract: the lanes enrolled
// and granted, nothing to cite.
func curationEarly(t *testing.T) *curationStand {
	t.Helper()
	store, resolve, root := seededStore(t)
	st := &curationStand{root: root, curator: fixtureKey(t, 12)}
	loose := func(fp string) (ed25519.PublicKey, bool) {
		for _, p := range []ed25519.PrivateKey{root, st.curator} {
			if fpOf(t, p) == fp {
				return p.Public().(ed25519.PublicKey), true
			}
		}
		return resolve(fp)
	}
	appendSigned(t, store, loose, root, ledger.UpgradeVerb, "system", `{"to": "`+version.Seed1+`"}`)
	appendSignedV(t, store, loose, root, version.Seed1, keyring.VerbEnrolled, fpOf(t, st.curator), enrollBody(t, st.curator, "agent", "curator"))
	appendSignedV(t, store, loose, root, version.Seed1, keyring.VerbGranted, fpOf(t, st.curator), `{"capability": "`+keyring.CapCurate+`"}`)
	ctx, err := ContextAt(store)
	if err != nil {
		t.Fatal(err)
	}
	st.ctx = ctx
	return st
}

func loadResidualsFrom(t *testing.T, name string) []residual {
	t.Helper()
	b, err := readInjectionFile(t, name)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Residuals []residual `json:"residuals"`
	}
	if err := jsonUnmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Residuals) == 0 {
		t.Fatalf("%s is empty: the reachable set would pass vacuously", name)
	}
	return doc.Residuals
}
