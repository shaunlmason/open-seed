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
	"github.com/shaunlmason/open-seed/next/internal/keyring"
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
	// KindBudgetOpen is an open valid reservation: it stands from the
	// reserve until a settle or a release closes it, inside the window
	// that opened it and after (plans/os-d6963652.md D3).
	KindBudgetOpen = "budget.open"
	// KindContractBlocked is a blocked subject awaiting whoever the
	// block named.
	KindContractBlocked = "contract.blocked"
	// KindEscalationPending is a standing blocked(needs-you): a
	// question addressed to a human gate that nothing else about the
	// contract moves past (plans/os-f781f0da.md). It is the narrower
	// sibling of KindContractBlocked, and both are emitted on an
	// escalated subject: the first says a human owes a decision, the
	// second that the contract is stopped.
	KindEscalationPending = "escalation.pending"
	// KindVerdictHuman is a human-verdict deferral nobody has rendered
	// over (plans/os-2e34f66a.md D4): owed by the operator lane, since
	// a human is a key with operator standing, and discharged by the
	// verdict.rendered such a key makes on the same submission.
	KindVerdictHuman = "verdict.human"
	// KindReapOwed is an in_progress claim whose holder has been revoked:
	// it can end no other way than a reap, because a revoked holder
	// cannot submit, release or park (plans/os-32d06c65.md D4). Distinct
	// from KindClaimHeld, which any deliberate exit discharges.
	KindReapOwed = "claim.reap-owed"
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
	// next/spec/escalation.md: both answers close the question, and
	// cancelling counts because it must cite the escalation it
	// closes — an answer of "this work should not happen".
	KindEscalationPending: {"contract.cancelled", "decision.recorded"},
	// next/spec/verdicts.md: the human's render answers the deferral.
	KindVerdictHuman: {"verdict.rendered"},
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
	// TS is the raising event's own timestamp, present only where the
	// obligation's age is meaningful in elapsed time. Positions order
	// without measuring: an escalation untouched for hours has the
	// same position difference as one answered instantly after a burst
	// of unrelated traffic, so latency derived from Since would be
	// event count wearing a clock's clothes. The reading surface
	// computes now minus TS at its own instant, never at admission
	// (next/spec/offers.md's live-read posture).
	TS string `json:"ts,omitempty"`
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

// Deps are the derivations this projection READS rather than
// recomputing, so it stays a projection over one fold and never a
// second opinion.
type Deps struct {
	// BudgetOpen supplies the open valid reservations for a subject:
	// the caller passes the one shared budget derivation rather than
	// this package re-deriving validity.
	BudgetOpen func(subject string, s transition.SubjectState) []transition.ReservationFact
	// CanDischarge reports whether an actor may still perform ANY of
	// the named verbs. Standing is the keyring's authority, not this
	// package's, and ownership of a fact-shaped obligation follows who
	// can pay it (plans/os-d6963652.md D4). A nil predicate is "no
	// standing projection was supplied", which cannot establish that
	// anyone is unable, so the usual owner stands.
	CanDischarge func(actor string, verbs []string) bool
}

// able is the standing question with the nil case named once.
func (d Deps) able(actor string, verbs []string) bool {
	return d.CanDischarge == nil || d.CanDischarge(actor, verbs)
}

// Derive folds the records and returns every standing obligation, in
// a stable order (subject, then kind).
func Derive(records []*event.Record, table *transition.Table, deps Deps) []Row {
	fold := table.FoldRecords(records)
	// The keyring's standing is what tells the reap obligation from an
	// ordinary held claim: a revoked holder's open window owes a reap.
	ring, _, _ := keyring.StateAt(records)
	var rows []Row
	for _, subject := range fold.Subjects() {
		s, ok := fold.State(subject)
		if !ok {
			continue
		}
		rows = append(rows, subjectRows(subject, s, table, deps)...)
		if ring != nil && s.Claim != nil && s.State == "in_progress" {
			if e, ok := ring.Get(s.Claim.Holder); ok && e.Standing == keyring.StandingRevoked {
				rows = append(rows, Row{Subject: subject, Kind: KindReapOwed, OwedBy: LaneOperator, Since: s.Claim.Fence, DischargedBy: []string{"claim.reaped"}})
			}
		}
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

func subjectRows(subject string, s transition.SubjectState, table *transition.Table, deps Deps) []Row {
	var rows []Row
	// ts is empty except where an obligation's age is meaningful in
	// elapsed time; it is a parameter rather than a returned pointer
	// because a pointer into rows would dangle the moment the next
	// append reallocated.
	add := func(kind, owedBy string, since int, ts string, dischargers []string) {
		// Never advertise an empty discharging set: the sweep asserts
		// this, and emitting one would make the drift class pass
		// vacuously (review finding on the plan PR).
		if len(dischargers) == 0 {
			return
		}
		rows = append(rows, Row{Subject: subject, Kind: kind, OwedBy: owedBy, Since: since, TS: ts, DischargedBy: dischargers})
	}

	if s.Claim != nil {
		add(KindClaimHeld, s.Claim.Holder, s.Since, "", stateDischargers(table, s.State))
	}
	if s.State == "blocked" {
		add(KindContractBlocked, LaneOperator, s.Since, "", stateDischargers(table, s.State))
	}
	if s.Escalation != nil {
		// Since is the RAISE's position, not the state's: a question
		// carried by a claim.parked arrives with the exit that raised
		// it, and a later reader needs the position that asked, not
		// the one that blocked. The row also carries the raise's TS,
		// because age is elapsed time and a position measures nothing
		// (next/spec/escalation.md).
		add(KindEscalationPending, LaneOperator, s.Escalation.Pos, s.Escalation.TS, factDischargers[KindEscalationPending])
	}
	if s.Submission != nil && (s.Verdict == nil || s.Verdict.Submission != s.Submission.Pos) {
		add(KindSubmissionPending, LaneVerifier, s.Submission.Pos, "", factDischargers[KindSubmissionPending])
	}
	// A deferral on the current window with no render after it: the
	// verifier could not judge, and the debt moved to the human.
	if s.Deferred != nil && (s.Verdict == nil || s.Verdict.Pos < s.Deferred.Pos) {
		add(KindVerdictHuman, LaneOperator, s.Deferred.Pos, "", factDischargers[KindVerdictHuman])
	}
	// An eval's chain ends at its verdict (plans/os-03e47abb.md D10):
	// it is never merged, its consequence is a qualification or a
	// disqualification, and a merge owed forever would be a debt
	// nobody can pay.
	if s.Verdict != nil && s.Verdict.Verdict == "pass" && s.Merged == nil && s.Eval == nil {
		// One kind, two shapes, because the merge chain is two
		// events: until a request cites the verdict the debt is the
		// operator's and merge.requested pays it; after that the
		// forge fact is the observer's to record.
		if s.Requested == nil {
			add(KindVerdictUnmerged, LaneOperator, s.Verdict.Pos, "", []string{mergeRequestVerb})
		} else {
			add(KindVerdictUnmerged, LaneObserver, s.Requested.Pos, "", []string{"merge.observed"})
		}
	}
	// An open reservation is owed WHEREVER it stands (os-d6963652):
	// the earlier in_progress restriction existed only because
	// admission gated the closing verbs on the same state, so outside
	// the window the advertised dischargers were unreachable and the
	// row would have been an anomaly. Admission now gates only the
	// reserve, so the closes are reachable and the debt is an
	// obligation again — which matters most on the failed-verdict
	// retry, where the next claimant is a different worker and the
	// previous attempt's unclosed hold would silently tax them.
	//
	// The owner is whoever can still pay it, never whoever holds the
	// window: admission closes a reservation for its own reserving
	// signer or the operator lane and nobody else, so attributing the
	// row to the current holder named a party admission refuses on any
	// reservation the holder did not sign. The signer keeps it until
	// suspension or revocation means every close from them refuses,
	// and then the operator lane is the only party left; keying the
	// row to a fingerprint nobody can sign for would hide it from the
	// one actor able to act on it.
	if deps.BudgetOpen != nil {
		for _, r := range deps.BudgetOpen(subject, s) {
			owner := r.Signer
			if !deps.able(owner, factDischargers[KindBudgetOpen]) {
				owner = LaneOperator
			}
			add(KindBudgetOpen, owner, r.Pos, "", factDischargers[KindBudgetOpen])
			break
		}
	}
	for _, start := range s.RunStarts {
		if settled(s, start.Fence) || !runFlaggable(s, start.Fence) {
			continue
		}
		add(KindRunUnsettled, LaneSupervisor, start.Pos, "", factDischargers[KindRunUnsettled])
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
