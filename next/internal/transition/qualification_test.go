package transition_test

// The tuple folds (plans/os-8e53ffd9.md D3, D6): a run.started carries
// its declared tuple into the RunStartFact and an offer its scoped
// tuples into the OfferFact, each read tolerantly: the fold is a
// record of what was appended, and admission is where a malformed
// declaration is refused.

import (
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/transition"
)

const foldTuple = `{"principal": "acme", "harness": "local-worktree/v0", "model": "fable/5.1", "tool_policy": "default", "environment": "detached-git-worktree"}`

func TestRunStartAndOfferFoldsReadTuples(t *testing.T) {
	tab, err := transition.Default()
	if err != nil {
		t.Fatal(err)
	}
	recs := []*event.Record{
		payloadEvent("seed/1", "intent.filed", "c-1", `{"tier": "trivial", "budget": "small"}`),                                                  // 0
		lifecycleEvent("contract.specified", "c-1"),                                                                                              // 1
		payloadEvent("seed/2", "offer.published", "c-1", `{"eligibility": {"tuples": [`+foldTuple+`]}, "expires": "2027-01-01T00:00:00Z"}`),      // 2
		payloadEvent("seed/2", "offer.published", "c-1", `{"eligibility": {"tuples": [{"principal": "x"}]}, "expires": "2027-01-01T00:00:00Z"}`), // 3: malformed member, dropped
		lifecycleEvent("claim.taken", "c-1"),                                                                            // 4
		budgetEvent("budget.reserve", "c-1", `{"amount": "10"}`),                                                        // 5
		payloadEvent("seed/2", "run.started", "c-1", `{"fence": "4", "reservation": "5", "tuple": `+foldTuple+`}`),      // 6
		payloadEvent("seed/2", "run.started", "c-1", `{"fence": "4", "reservation": "5", "tuple": {"principal": "x"}}`), // 7: malformed, folds without one
	}
	s, ok := tab.FoldRecords(recs).State("c-1")
	if !ok {
		t.Fatal("c-1 folds")
	}
	if len(s.Offers) != 2 {
		t.Fatalf("both offers fold: %+v", s.Offers)
	}
	if len(s.Offers[0].Tuples) != 1 || s.Offers[0].Tuples[0].Model != "fable/5.1" {
		t.Fatalf("the offer's scoped tuples fold: %+v", s.Offers[0].Tuples)
	}
	if len(s.Offers[1].Tuples) != 0 {
		t.Fatalf("a malformed member is dropped, never folded as a partial tuple: %+v", s.Offers[1].Tuples)
	}
	if len(s.RunStarts) != 2 {
		t.Fatalf("both starts fold: %+v", s.RunStarts)
	}
	if s.RunStarts[0].Tuple == nil || s.RunStarts[0].Tuple.Principal != "acme" {
		t.Fatalf("the declared tuple folds onto the start: %+v", s.RunStarts[0])
	}
	if s.RunStarts[1].Tuple != nil {
		t.Fatalf("a malformed declaration folds as no declaration: %+v", s.RunStarts[1])
	}
}
