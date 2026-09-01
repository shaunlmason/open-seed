package obligation

// The derivation drills (plans/os-52d5da3f.md D3, D5): each kind
// arises from its folded fact and nothing else, run.unsettled is
// position-anchored, and no emitted row ever carries an empty
// discharging set.

import (
	"slices"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/transition"
)

func table(t *testing.T) *transition.Table {
	t.Helper()
	tbl, err := transition.Default()
	if err != nil {
		t.Fatal(err)
	}
	return tbl
}

// rowsFor derives from one hand-built state, bypassing the fold: the
// kinds are a function of the state's facts, and the fold's own
// correctness is its package's business.
func rowsFor(t *testing.T, s transition.SubjectState, open []transition.ReservationFact) []Row {
	t.Helper()
	return subjectRows("c-1", s, table(t), func(string, transition.SubjectState) []transition.ReservationFact { return open })
}

func kinds(rows []Row) []string {
	var out []string
	for _, r := range rows {
		out = append(out, r.Kind)
	}
	slices.Sort(out)
	return out
}

func TestKindsAriseFromTheirFacts(t *testing.T) {
	for name, tc := range map[string]struct {
		state transition.SubjectState
		open  []transition.ReservationFact
		want  []string
	}{
		"ready subject owes nothing": {
			state: transition.SubjectState{State: "ready", Since: 3},
		},
		"an active claim is owed by its holder": {
			state: transition.SubjectState{State: "in_progress", Since: 5, Claim: &transition.Claim{Holder: "aa", Fence: 5}},
			want:  []string{KindClaimHeld},
		},
		"a submission with no verdict is owed by the verifier lane": {
			state: transition.SubjectState{State: "review", Since: 8, Submission: &transition.SubmissionFact{Pos: 8, Signer: "aa"}},
			want:  []string{KindSubmissionPending},
		},
		"a verdict citing the submission discharges it": {
			state: transition.SubjectState{State: "review", Since: 8,
				Submission: &transition.SubmissionFact{Pos: 8, Signer: "aa"},
				Verdict:    &transition.VerdictFact{Pos: 9, Verdict: "pass", Submission: 8}},
			want: []string{KindVerdictUnmerged},
		},
		"a fail verdict owes no merge": {
			state: transition.SubjectState{State: "review", Since: 8,
				Submission: &transition.SubmissionFact{Pos: 8, Signer: "aa"},
				Verdict:    &transition.VerdictFact{Pos: 9, Verdict: "fail", Submission: 8}},
		},
		"blocked is owed by the operator lane": {
			state: transition.SubjectState{State: "blocked", Since: 4},
			want:  []string{KindContractBlocked},
		},
		"an open reservation inside the window is owed by the holder": {
			state: transition.SubjectState{State: "in_progress", Since: 5, Claim: &transition.Claim{Holder: "aa", Fence: 5}},
			open:  []transition.ReservationFact{{Pos: 6, Signer: "aa", Amount: 2}},
			want:  []string{KindBudgetOpen, KindClaimHeld},
		},
		"an open reservation outside the window is no obligation": {
			// admission gates every budget verb on in_progress, so
			// outside the window nothing can discharge it: an
			// obligation nobody can discharge is an anomaly (card
			// os-d6963652).
			state: transition.SubjectState{State: "review", Since: 8, Submission: &transition.SubmissionFact{Pos: 8, Signer: "aa"}},
			open:  []transition.ReservationFact{{Pos: 6, Signer: "aa", Amount: 2}},
			want:  []string{KindSubmissionPending},
		},
	} {
		got := kinds(rowsFor(t, tc.state, tc.open))
		if !slices.Equal(got, tc.want) {
			t.Errorf("%s: kinds %v, want %v", name, got, tc.want)
		}
	}
}

// conformance: the Phase 7 exit's metering-detection obligation is
// position-anchored — post-close settlement is a valid intermediate
// state, so an unsettled run is flagged only once the subject has
// taken a SUBSEQUENT claim window or reached a terminal state.
func TestUnsettledRunIsPositionAnchored(t *testing.T) {
	started := transition.SubjectState{State: "ready", Since: 9,
		RunStarts:   []transition.RunStartFact{{Pos: 7, Fence: 5}},
		ClaimFences: map[int]bool{5: true}}
	if k := kinds(rowsFor(t, started, nil)); slices.Contains(k, KindRunUnsettled) {
		t.Fatalf("a closed window awaiting its settle is not yet a finding: %v", k)
	}
	later := started
	later.ClaimFences = map[int]bool{5: true, 11: true}
	if k := kinds(rowsFor(t, later, nil)); !slices.Contains(k, KindRunUnsettled) {
		t.Fatalf("a subsequent claim window makes the missing settle a finding: %v", k)
	}
	terminal := started
	terminal.State = "done"
	if k := kinds(rowsFor(t, terminal, nil)); !slices.Contains(k, KindRunUnsettled) {
		t.Fatalf("a terminal subject makes the missing settle a finding: %v", k)
	}
	settled := later
	settled.Runs = []transition.RunFact{{Pos: 8, Fence: 5, Units: 1}}
	if k := kinds(rowsFor(t, settled, nil)); slices.Contains(k, KindRunUnsettled) {
		t.Fatalf("a settled fence is no finding: %v", k)
	}
}

// conformance: an obligation nobody can discharge is an anomaly, not
// an obligation — and a drift sweep over an empty discharging set
// would pass vacuously (review finding on the plan PR).
func TestEveryRowCarriesADischarger(t *testing.T) {
	states := []transition.SubjectState{
		{State: "in_progress", Since: 5, Claim: &transition.Claim{Holder: "aa", Fence: 5}},
		{State: "review", Since: 8, Submission: &transition.SubmissionFact{Pos: 8, Signer: "aa"}},
		{State: "review", Since: 8, Verdict: &transition.VerdictFact{Pos: 9, Verdict: "pass", Submission: 8}},
		{State: "blocked", Since: 4},
		{State: "done", Since: 12, RunStarts: []transition.RunStartFact{{Pos: 7, Fence: 5}}, ClaimFences: map[int]bool{5: true}},
	}
	seen := map[string]bool{}
	for _, s := range states {
		for _, row := range rowsFor(t, s, []transition.ReservationFact{{Pos: 6, Signer: "aa", Amount: 2}}) {
			if len(row.DischargedBy) == 0 {
				t.Fatalf("%s carries no discharging verb", row.Kind)
			}
			seen[row.Kind] = true
		}
	}
	for _, kind := range []string{KindClaimHeld, KindSubmissionPending, KindVerdictUnmerged, KindBudgetOpen, KindContractBlocked, KindRunUnsettled} {
		if !seen[kind] {
			t.Errorf("the fixture set never produced %s, so its dischargers are unchecked", kind)
		}
	}
}
