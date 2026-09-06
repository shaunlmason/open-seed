package ranking_test

// The resumption drills (plans/os-29e2fef2.md AC1, AC5): over one
// real chain, a return by observation yields the returned window's
// declared tuple, holder and consumed offer, and every missing link
// refuses by name: a verdict return, no return, no admitted start, a
// start that declared no tuple, a suspended holder, and an active
// holder whose tuple a disqualification removed; a raw-pushed offer
// by an ungranted key is never the consumed offer.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/keyring"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
	"github.com/shaunlmason/open-seed/next/internal/ranking"
	"github.com/shaunlmason/open-seed/next/internal/transition"
	"github.com/shaunlmason/open-seed/next/internal/version"
)

const resumeHead = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

// resumeStand is the ranking stand carried to seed/8 with one
// contract on it: filed, specified, offered by the root, and claimed
// by worker a under a claim grant citing its tuple.
type resumeStand struct {
	*stand
	tuple string
	offer int
	fence int
}

func (s *stand) at8(who, verb, subject, payload string) int {
	s.t.Helper()
	s.clock++
	return s.addAt(who, version.Seed8, fmt.Sprintf("2026-09-02T%02d:00:00Z", s.clock%24), verb, subject, payload)
}

func newResumeStand(t *testing.T) *resumeStand {
	t.Helper()
	s := newStand(t)
	for _, v := range []string{version.Seed5, version.Seed6, version.Seed7, version.Seed8} {
		s.next("root", ledger.UpgradeVerb, "system", `{"to": "`+v+`"}`)
	}
	r := &resumeStand{stand: s, tuple: tupleJSON("lineage/1", "detached-git-worktree")}
	s.at8("root", keyring.VerbGranted, s.fps["a"], `{"capability": "claim", "tuple": `+r.tuple+`}`)
	s.at8("root", "intent.filed", "c-1", `{"intent": "drill", "tier": "trivial", "budget": "small", "routing": "core"}`)
	s.at8("root", "contract.specified", "c-1", `{"acceptance": {"ref": "spec.md @ 0123456789abcdef", "executable": false}}`)
	r.offer = s.at8("root", "offer.published", "c-1", `{"eligibility": {"capabilities": ["claim"], "tiers": ["trivial"]}, "expires": "2027-01-01T00:00:00Z"}`)
	return r
}

// claim opens worker a's window and, unless told otherwise, reserves
// and starts a run declaring the tuple.
func (r *resumeStand) claim(t *testing.T, start string) {
	t.Helper()
	r.fence = r.at8("a", "claim.taken", "c-1", `{}`)
	reservation := r.at8("a", "budget.reserve", "c-1", fmt.Sprintf(`{"amount": "10", "fence": "%d"}`, r.fence))
	switch start {
	case "declared":
		r.at8("a", "run.started", "c-1", fmt.Sprintf(`{"fence": "%d", "reservation": "%d", "tuple": %s}`, r.fence, reservation, r.tuple))
	case "undeclared":
		r.at8("a", "run.started", "c-1", fmt.Sprintf(`{"fence": "%d", "reservation": "%d"}`, r.fence, reservation))
	case "none":
	default:
		t.Fatalf("unknown start %q", start)
	}
}

func (r *resumeStand) submit(t *testing.T) int {
	t.Helper()
	packet := `{"acceptance": ["c-1"], "decisions": [], "base": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa..` + resumeHead + `", "refs": [], "findings": []}`
	return r.at8("a", "submission.made", "c-1", fmt.Sprintf(`{"fence": "%d", "packet": %s, "pr": "pr/1"}`, r.fence, packet))
}

// returnRed observes red and returns citing the observation.
func (r *resumeStand) returnRed(t *testing.T) (observation, ret int) {
	t.Helper()
	observation = r.at8("root", transition.CheckObservedVerb, "c-1", `{"pr": "pr/1", "head": "`+resumeHead+`", "checks": "red", "review": "none"}`)
	ret = r.at8("root", transition.ContractReturnedVerb, "c-1", fmt.Sprintf(`{"observation": "%d"}`, observation))
	return
}

func (r *resumeStand) resume(t *testing.T) (ranking.Resumption, bool) {
	t.Helper()
	records := r.records()
	table, err := transition.Default()
	if err != nil {
		t.Fatal(err)
	}
	return ranking.Resume(records, table.FoldRecords(records), "c-1")
}

// conformance: AC1 — the resumption derives from the chain: the
// returned window's declared tuple, its holder, the submission, the
// return and the consumed offer.
func TestResumeDerivesThePriorSubmittersTuple(t *testing.T) {
	r := newResumeStand(t)
	r.claim(t, "declared")
	submission := r.submit(t)
	observation, ret := r.returnRed(t)
	got, ok := r.resume(t)
	if !ok {
		t.Fatalf("the resumption derives after a return by observation, refused: %s", got.Because)
	}
	if got.Tuple == nil || !got.Tuple.Equal(parse(t, r.tuple)) || got.Holder != r.fps["a"] {
		t.Fatalf("the tuple and holder are the window's: %+v", got)
	}
	if got.Submission != submission || got.Return != ret || got.Observation != observation || got.Fence != r.fence {
		t.Fatalf("the positions are the chain's: %+v (submission %d, return %d, observation %d, fence %d)", got, submission, ret, observation, r.fence)
	}
	if got.Offer == nil || got.Offer.Pos != r.offer || len(got.Offer.Tiers) != 1 || got.Offer.Tiers[0] != "trivial" {
		t.Fatalf("the consumed offer is the root's: %+v", got.Offer)
	}
}

// conformance: AC1, AC5 — every missing link refuses by name, and a
// mutation that derived after a verdict return, preferred a suspended
// holder's tuple, or preferred a disqualified tuple fails here.
func TestResumeRefusesByName(t *testing.T) {
	for name, tc := range map[string]struct {
		arrange func(t *testing.T, r *resumeStand)
		says    string
		window  bool
	}{
		"no return": {func(t *testing.T, r *resumeStand) {
			r.claim(t, "declared")
			r.submit(t)
		}, "carries no return", false},
		"a verdict return routes by the ranking": {func(t *testing.T, r *resumeStand) {
			r.claim(t, "declared")
			sub := r.submit(t)
			v := r.at8("v", "verdict.rendered", "c-1", fmt.Sprintf(`{"verdict": "fail", "receipt": "%s", "submission": "%d", "independence": "L1"}`, strings.Repeat("0", 64), sub))
			r.at8("root", transition.ContractReturnedVerb, "c-1", fmt.Sprintf(`{"verdict": "%d"}`, v))
		}, "cited a verdict, not an observation", false},
		"no admitted run.started": {func(t *testing.T, r *resumeStand) {
			r.claim(t, "none")
			r.submit(t)
			r.returnRed(t)
		}, "carries no admitted run.started", true},
		"a start that declared no tuple": {func(t *testing.T, r *resumeStand) {
			r.claim(t, "undeclared")
			r.submit(t)
			r.returnRed(t)
		}, "declared no tuple", true},
		"a suspended holder": {func(t *testing.T, r *resumeStand) {
			r.claim(t, "declared")
			r.submit(t)
			r.returnRed(t)
			r.at8("root", keyring.VerbSuspended, r.fps["a"], `{"reason": "drill"}`)
		}, "is suspended", true},
		"a revoked holder": {func(t *testing.T, r *resumeStand) {
			r.claim(t, "declared")
			r.submit(t)
			r.returnRed(t)
			r.at8("root", keyring.VerbRevoked, r.fps["a"], `{"reason": "drill"}`)
		}, "is revoked", true},
		"an active holder whose tuple was disqualified": {func(t *testing.T, r *resumeStand) {
			r.claim(t, "declared")
			r.submit(t)
			r.returnRed(t)
			r.at8("root", keyring.VerbDisqualified, r.fps["a"], `{"capability": "claim", "tuple": `+r.tuple+`, "contract": "e-1", "verdict": "3", "reason": "the eval failed"}`)
		}, "admissible claim grant does not cite it", true},
	} {
		t.Run(name, func(t *testing.T) {
			r := newResumeStand(t)
			tc.arrange(t, r)
			got, ok := r.resume(t)
			if ok || got.Tuple != nil {
				t.Fatalf("the resumption must refuse, derived %+v", got)
			}
			if !strings.Contains(got.Because, tc.says) {
				t.Fatalf("the refusal names the link: want %q in %q", tc.says, got.Because)
			}
			if tc.window && (got.Holder != r.fps["a"] || got.Fence != r.fence || got.Offer == nil) {
				t.Fatalf("the window still derives for the re-offer's scope (D3): %+v", got)
			}
			if !tc.window && got.Holder != "" {
				t.Fatalf("no window derives without a return by observation: %+v", got)
			}
		})
	}
}

// conformance: AC3, AC5 — the consumed offer is derived with the
// listing's two predicates: a well-shaped offer raw-pushed by a key
// holding no supervise standing between the legitimate offer and the
// claim is not the consumed offer, and neither is a legitimate offer
// the claimant was not eligible for; a mutation copying the latest
// folded fact fails here.
func TestResumeConsumedOfferIsAuthorizedAndMet(t *testing.T) {
	r := newResumeStand(t)
	// c holds claim, never supervise: its offer folds and is inert.
	r.at8("c", "offer.published", "c-1", `{"eligibility": {"tiers": ["huge"]}, "expires": "2027-01-01T00:00:00Z"}`)
	// The root's second offer scopes a tier a's contract is not in, so
	// a could not have taken it.
	r.at8("root", "offer.published", "c-1", `{"eligibility": {"capabilities": ["claim"], "tiers": ["standard"]}, "expires": "2027-01-01T00:00:00Z"}`)
	r.claim(t, "declared")
	r.submit(t)
	r.returnRed(t)
	got, ok := r.resume(t)
	if !ok || got.Offer == nil || got.Offer.Pos != r.offer {
		t.Fatalf("the consumed offer is the latest authorized offer the claimant met, the root's first: %+v (%s)", got.Offer, got.Because)
	}
}

// conformance: AC5 — with no offer before the claim the window derives
// with no consumed offer, and the tuple still does.
func TestResumeWithoutAnOffer(t *testing.T) {
	s := newStand(t)
	for _, v := range []string{version.Seed5, version.Seed6, version.Seed7, version.Seed8} {
		s.next("root", ledger.UpgradeVerb, "system", `{"to": "`+v+`"}`)
	}
	r := &resumeStand{stand: s, tuple: tupleJSON("lineage/1", "detached-git-worktree")}
	s.at8("root", keyring.VerbGranted, s.fps["a"], `{"capability": "claim", "tuple": `+r.tuple+`}`)
	s.at8("root", "intent.filed", "c-1", `{"intent": "drill", "tier": "trivial", "budget": "small", "routing": "core"}`)
	s.at8("root", "contract.specified", "c-1", `{"acceptance": {"ref": "spec.md @ 0123456789abcdef", "executable": false}}`)
	r.claim(t, "declared")
	r.submit(t)
	r.returnRed(t)
	got, ok := r.resume(t)
	if !ok || got.Offer != nil || got.Tuple == nil {
		t.Fatalf("no offer stood, the tuple derives: %+v (%s)", got, got.Because)
	}
}

// A subject that never existed, and a nil fold, refuse rather than
// panic.
func TestResumeOnNothing(t *testing.T) {
	r := newResumeStand(t)
	records := r.records()
	table, _ := transition.Default()
	if _, ok := ranking.Resume(records, table.FoldRecords(records), "c-9"); ok {
		t.Fatal("an unknown subject has nothing to resume")
	}
	if _, ok := ranking.Resume(records, nil, "c-1"); ok {
		t.Fatal("a nil fold has nothing to resume")
	}
}
