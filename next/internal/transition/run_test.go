package transition_test

// The run-fact fold drills (plans/os-1dad487d.md;
// next/spec/executors.md): well-shaped facts fold tolerantly as
// independent lists, a fact citing a fence that is no applied claim
// position counts an anomaly, and the claim-fence domain tracks every
// applied claim.

import (
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/transition"
)

func TestRunFactFold(t *testing.T) {
	tab, err := transition.Default()
	if err != nil {
		t.Fatal(err)
	}
	records := []*event.Record{
		payloadEvent("seed/1", "intent.filed", "c-1", `{"tier": "trivial", "budget": "small"}`),     // 0
		lifecycleEvent("contract.specified", "c-1"),                                                 // 1
		lifecycleEvent("claim.taken", "c-1"),                                                        // 2
		budgetEvent("budget.reserve", "c-1", `{"amount": "10"}`),                                    // 3
		payloadEvent("seed/1", "run.started", "c-1", `{"fence": "2", "reservation": "3"}`),          // 4
		payloadEvent("seed/1", "run.started", "c-1", `{"fence": "9", "reservation": "3"}`),          // 5: dangling fence, anomaly
		payloadEvent("seed/1", "run.settled", "c-1", `{"fence": "2", "units": "7", "lines": "3"}`),  // 6
		payloadEvent("seed/1", "run.settled", "c-1", `{"fence": "2", "units": "-1", "lines": "0"}`), // 7: malformed, no fact
	}
	fold := tab.FoldRecords(records)
	s, ok := fold.State("c-1")
	if !ok || !s.ClaimFences[2] || len(s.ClaimFences) != 1 {
		t.Fatalf("the claim-fence domain tracks applied claims: %+v", s.ClaimFences)
	}
	if len(s.RunStarts) != 1 || s.RunStarts[0].Pos != 4 || s.RunStarts[0].Fence != 2 || s.RunStarts[0].Reservation != 3 {
		t.Fatalf("the well-shaped start folds: %+v", s.RunStarts)
	}
	if len(s.Runs) != 1 || s.Runs[0].Pos != 6 || s.Runs[0].Units != 7 || s.Runs[0].Lines != 3 {
		t.Fatalf("the well-shaped settle folds, the malformed one to nothing: %+v", s.Runs)
	}
	if s.Anomalies == 0 {
		t.Fatal("a run fact citing a dangling fence counts an anomaly, never a fact")
	}
}
