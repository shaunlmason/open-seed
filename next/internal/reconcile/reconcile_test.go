package reconcile

import (
	"strings"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/transition"
)

func find(fs []Finding, class string) *Finding {
	for i := range fs {
		if fs[i].Class == class {
			return &fs[i]
		}
	}
	return nil
}

func TestSubjectClassifiesInducedDivergences(t *testing.T) {
	pass := &transition.VerdictFact{Pos: 9, Verdict: "pass", Receipt: "r"}
	fail := &transition.VerdictFact{Pos: 9, Verdict: "fail", Receipt: "r"}
	req := &transition.RequestFact{Pos: 10, CitedVerdict: 9}
	merged := &transition.MergeFact{Pos: 11, SHA: "abc"}

	cases := map[string]struct {
		state transition.SubjectState
		want  []string
	}{
		"clean full chain": {
			transition.SubjectState{State: "done", Verdict: pass, Requested: req, Merged: merged}, nil},
		"no chain activity": {
			transition.SubjectState{State: "in_progress"}, nil},
		"merge without any verdict": {
			transition.SubjectState{State: "done", Merged: merged}, []string{ClassMergeWithoutVerdict}},
		"merge over a fail verdict": {
			transition.SubjectState{State: "done", Verdict: fail, Merged: merged}, []string{ClassMergeWithoutVerdict}},
		"chain skipped, no request": {
			transition.SubjectState{State: "done", Verdict: pass, Merged: merged}, []string{ClassChainSkipped}},
		"chain skipped, wrong citation": {
			transition.SubjectState{State: "done", Verdict: pass,
				Requested: &transition.RequestFact{Pos: 10, CitedVerdict: 3}, Merged: merged}, []string{ClassChainSkipped}},
		"unreconciled pass verdict": {
			transition.SubjectState{State: "review", Verdict: pass}, []string{ClassUnreconciled}},
		"fail verdict alone is not unreconciled": {
			transition.SubjectState{State: "review", Verdict: fail}, nil},
	}
	for name, c := range cases {
		got := Subject("c-x", c.state)
		if len(got) != len(c.want) {
			t.Fatalf("%s: got %+v, want classes %v", name, got, c.want)
		}
		for _, w := range c.want {
			if find(got, w) == nil {
				t.Fatalf("%s: missing class %s in %+v", name, w, got)
			}
		}
	}
}

func TestUnreconciledStaysNeutral(t *testing.T) {
	// The class is a surfaced state, never an accusation: with no wall
	// clock in any build, pending versus failed is maintenance's age
	// judgment. The detail prose is pinned to say so.
	fs := Subject("c-1", transition.SubjectState{State: "review",
		Verdict: &transition.VerdictFact{Pos: 4, Verdict: "pass"}})
	f := find(fs, ClassUnreconciled)
	if f == nil {
		t.Fatal("a pass verdict with no merge is unreconciled")
	}
	low := strings.ToLower(f.Detail)
	for _, banned := range []string{"failed", "stale", "violat", "accus"} {
		if strings.Contains(low, banned) {
			t.Fatalf("unreconciled must stay neutral, detail %q contains %q", f.Detail, banned)
		}
	}
}
