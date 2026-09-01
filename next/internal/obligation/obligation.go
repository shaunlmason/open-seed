// Package obligation derives what is OWED on each subject
// (plans/os-52d5da3f.md; docs/next-build-plan.md Phase 9 item 5).
// Seed represents permission already: admit.Affordances answers "what
// may I do", computed from the rule set admission enforces. Nothing
// answered "what is owed, by whom, since when, and which verbs
// discharge it", although every fact needed is folded already: the
// active claim with its fence, the bound submission, the standing
// verdict, run starts against run settles, open reservations, and the
// state itself with the position that set it.
//
// This package is a projection over that fold and never a new
// authority. It invents no legality: a state-shaped obligation reads
// its discharging verbs from the transition table, and the closed set
// of fact-shaped obligations (whose closing verb changes no lifecycle
// state and so appears in no table row) maps each to the spec that
// pairs it with its fact.
package obligation

import (
	"sort"

	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/transition"
)

// The obligation kinds. The list is closed on purpose: an open-ended
// taxonomy would make this projection a policy surface rather than a
// derivation (plans/os-52d5da3f.md D3).
const (
	// KindClaimHeld is an active claim window: the holder owes a
	// deliberate exit.
	KindClaimHeld = "claim.held"
	// KindSubmissionPending is a submission awaiting judgment: the
	// verifier lane owes a verdict.
	KindSubmissionPending = "submission.pending"
	// KindVerdictUnmerged is a pass verdict whose merge is not yet
	// observed.
	KindVerdictUnmerged = "verdict.unmerged"
	// KindRunUnsettled is an admitted run.started whose fence carries
	// no run.settled, flagged only once the window can no longer
	// settle it (position-anchored; the Phase 7 exit's
	// metering-detection obligation).
	KindRunUnsettled = "run.unsettled"
	// KindBudgetOpen is an open valid reservation inside the live
	// claim window, where its closing verbs admit.
	KindBudgetOpen = "budget.open"
	// KindContractBlocked is a blocked subject awaiting whoever the
	// block named.
	KindContractBlocked = "contract.blocked"
)

// Lane names used where an obligation is owed by a role rather than
// by one fingerprint: independence forbids naming the claimant as the
// verifier, and a merge is observed by whoever holds the standing.
const (
	LaneVerifier   = "lane:verdict"
	LaneObserver   = "lane:observer"
	LaneSupervisor = "lane:supervise"
	LaneOperator   = "lane:operator"
)

// factDischargers is the closed set of fact-shaped obligations: their
// closing verb changes no lifecycle state, so it appears in no
// transition-table row and must be mapped from the spec that pairs it
// with its fact (review finding on the plan PR: a table-only
// derivation advertises no discharger at all for these).
var factDischargers = map[string][]string{
	// next/spec/executors.md: metering settles at run end.
	KindRunUnsettled: {"run.settled"},
	// next/spec/budgets.md: a reservation closes by settle or release.
	KindBudgetOpen: {"budget.settle", "budget.release"},
	// next/spec/verdicts.md: the verdict is a fact, not a transition.
	KindSubmissionPending: {"verdict.rendered"},
}

// mergeRequestVerb discharges a standing pass verdict that no merge
// request cites yet. The merge chain is two events, not one: the
// drift sweep caught the first draft advertising merge.observed while
// admission still refused it for want of a request, which is exactly
// the class the sweep exists to raise
// (next/spec/reconciliation.md: each chain step is its own event).
const mergeRequestVerb = "merge.requested"

// Row is one obligation. Identity is (Subject, Kind): the situation
// read's delta names removals by that pair, so it is normative rather
// than incidental (plans/os-52d5da3f.md D4).
type Row struct {
	Subject string `json:"subject"`
	Kind    string `json:"kind"`
	// OwedBy is a fingerprint, or a "lane:<capability>" name where the
	// obligation belongs to a role rather than one actor.
	OwedBy string `json:"owed_by"`
	// Since is the chain position the obligation arose at.
	Since int `json:"since"`
	// DischargedBy is every verb that discharges it, sorted. Never
	// empty: an obligation nobody can discharge is an anomaly, not an
	// obligation, so a kind with no reachable discharger is not
	// emitted at all.
	DischargedBy []string `json:"discharged_by"`
}

// stateDischargers reads the verbs that leave a state from the
// transition table, so legality is never restated here.
func stateDischargers(table *transition.Table, state string) []string {
	var out []string
	for _, verb := range table.Verbs() {
		if table.Allows(state, verb) {
			out = append(out, verb)
		}
	}
	sort.Strings(out)
	return out
}

// Derive folds the records and returns every standing obligation, in
// a stable order (subject, then kind). budgetOpen supplies the open
// valid reservations for a subject: the caller passes the one shared
// budget derivation rather than this package re-deriving validity.
func Derive(records []*event.Record, table *transition.Table, budgetOpen func(string, transition.SubjectState) []transition.ReservationFact) []Row {
	fold := table.FoldRecords(records)
	var rows []Row
	for _, subject := range fold.Subjects() {
		s, ok := fold.State(subject)
		if !ok {
			continue
		}
		rows = append(rows, subjectRows(subject, s, table, budgetOpen)...)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Subject != rows[j].Subject {
			return rows[i].Subject < rows[j].Subject
		}
		return rows[i].Kind < rows[j].Kind
	})
	if rows == nil {
		rows = []Row{}
	}
	return rows
}

func subjectRows(subject string, s transition.SubjectState, table *transition.Table, budgetOpen func(string, transition.SubjectState) []transition.ReservationFact) []Row {
	var rows []Row
	add := func(kind, owedBy string, since int, dischargers []string) {
		// Never advertise an empty discharging set: the sweep asserts
		// this, and emitting one would make the drift class pass
		// vacuously (review finding on the plan PR).
		if len(dischargers) == 0 {
			return
		}
		rows = append(rows, Row{Subject: subject, Kind: kind, OwedBy: owedBy, Since: since, DischargedBy: dischargers})
	}

	if s.Claim != nil {
		add(KindClaimHeld, s.Claim.Holder, s.Since, stateDischargers(table, s.State))
	}
	if s.State == "blocked" {
		add(KindContractBlocked, LaneOperator, s.Since, stateDischargers(table, s.State))
	}
	if s.Submission != nil && (s.Verdict == nil || s.Verdict.Submission != s.Submission.Pos) {
		add(KindSubmissionPending, LaneVerifier, s.Submission.Pos, factDischargers[KindSubmissionPending])
	}
	if s.Verdict != nil && s.Verdict.Verdict == "pass" && s.Merged == nil {
		// One kind, two shapes, because the merge chain is two
		// events: until a request cites the verdict the debt is the
		// operator's and merge.requested pays it; after that the
		// forge fact is the observer's to record.
		if s.Requested == nil {
			add(KindVerdictUnmerged, LaneOperator, s.Verdict.Pos, []string{mergeRequestVerb})
		} else {
			add(KindVerdictUnmerged, LaneObserver, s.Requested.Pos, []string{"merge.observed"})
		}
	}
	// The budget window restriction is a finding, not a preference:
	// admission gates every budget verb on in_progress, so outside the
	// live window both closing verbs refuse and the reservation is a
	// maintenance concern rather than an obligation (card
	// os-d6963652).
	if s.State == "in_progress" && budgetOpen != nil {
		for _, r := range budgetOpen(subject, s) {
			owner := r.Signer
			if s.Claim != nil {
				owner = s.Claim.Holder
			}
			add(KindBudgetOpen, owner, r.Pos, factDischargers[KindBudgetOpen])
			break
		}
	}
	for _, start := range s.RunStarts {
		if settled(s, start.Fence) || !runFlaggable(s, start.Fence) {
			continue
		}
		add(KindRunUnsettled, LaneSupervisor, start.Pos, factDischargers[KindRunUnsettled])
		break
	}
	return rows
}

func settled(s transition.SubjectState, fence int) bool {
	for _, r := range s.Runs {
		if r.Fence == fence {
			return true
		}
	}
	return false
}

// runFlaggable is the position anchor the Phase 7 exit named: an
// unsettled run is flagged only once the subject has taken a
// SUBSEQUENT claim window or reached a terminal state, because
// post-close settlement is a valid intermediate state and a
// closed-without-settle predicate would file spurious findings
// mid park or reap flow.
func runFlaggable(s transition.SubjectState, fence int) bool {
	if s.State == "done" || s.State == "cancelled" {
		return true
	}
	for f := range s.ClaimFences {
		if f > fence {
			return true
		}
	}
	return false
}
