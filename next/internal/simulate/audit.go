package simulate

// The seven-day bar, audited from the ledger alone (plans/os-16e55c11.md
// D5): the simulation reconstructs and justifies every claim from the
// chain, never from its own bookkeeping. Each bar names the records
// that violate it; a clean run leaves every list empty.

import (
	"sort"

	"github.com/shaunlmason/open-seed/next/internal/admit"
	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/transition"
)

// AuditResult is the five bars III.R holds a run to, each an evidence
// list rather than a boolean, so a violation names itself.
type AuditResult struct {
	ChainViolations    []string `json:"chain_violations"`
	LostUpdates        []string `json:"lost_updates"`
	SilentAbandonments []string `json:"silent_abandonments"`
	GuardrailBreaches  []string `json:"guardrail_breaches"`
	UnreservedSpend    []string `json:"unreserved_spend"`
	Clean              bool     `json:"clean"`
}

// ClaimTakenVerb is the one verb the bars read that the protocol
// publishes no constant for (plans/os-b86dab4c.md D1).
const ClaimTakenVerb = "claim.taken"

// AuditedVerbs are the verbs the bars switch on. Each must be one the
// boundary drafts: a bar that counts a verb the protocol never emits
// is a bar that never fires, which is how the unreserved-spend bar
// came to count "budget.reserved" against a protocol that emits
// budget.reserve. The drill holds this list to admit.CatalogVerbs
// (D3), not to the transition table, whose Verbs() is the lifecycle
// subset and carries neither budget.reserve nor run.started. The
// deliberate exits are absent because the audit no longer names them:
// transition.IsExit is their one authority (D2).
var AuditedVerbs = []string{
	transition.OfferPublishedVerb,
	ClaimTakenVerb,
	transition.BudgetReserveVerb,
	transition.RunStartedVerb,
}

// Audit reconstructs the five bars from the records alone.
func Audit(records []*event.Record) AuditResult {
	res := AuditResult{
		ChainViolations:    []string{},
		LostUpdates:        []string{},
		SilentAbandonments: []string{},
		GuardrailBreaches:  []string{},
		UnreservedSpend:    []string{},
	}

	// Chain violations: the admitted chain must fold without an illegal
	// transition. StateAt replays every record; a raw-pushed illegal
	// transition surfaces as a fold error.
	tbl, err := transition.Default()
	if err != nil {
		res.ChainViolations = append(res.ChainViolations, "no transition table: "+err.Error())
	} else if _, foldErr := foldAll(tbl, records); foldErr != nil {
		res.ChainViolations = append(res.ChainViolations, foldErr.Error())
	}

	// Lost updates: the materialized chain is the tip, so every record
	// read is present; the bar is that the chain is non-empty and its
	// prev-links are contiguous (a gap is a lost update). event.Record
	// verification on materialize already rejects a broken link, so a
	// non-empty chain that materialized is contiguous by construction;
	// an empty chain is itself a lost-everything.
	if len(records) == 0 {
		res.LostUpdates = append(res.LostUpdates, "the chain is empty")
	}

	// Per-subject: silent abandonment (a window opened by claim.taken
	// and never closed by a deliberate exit) and unreserved spend.
	type window struct {
		open    bool
		offered bool
		starts  []int
	}
	subj := map[string]*window{}
	get := func(s string) *window {
		w := subj[s]
		if w == nil {
			w = &window{}
			subj[s] = w
		}
		return w
	}
	for pos, rec := range records {
		s := rec.Event.Subject
		switch rec.Event.Verb {
		case transition.OfferPublishedVerb:
			get(s).offered = true
		case ClaimTakenVerb:
			w := get(s)
			// A claim must ride a published offer: claiming work the
			// supervisor never offered is a guardrail breach.
			if !w.offered {
				res.GuardrailBreaches = append(res.GuardrailBreaches, s)
			}
			w.open = true
		case transition.BudgetReserveVerb:
			// Read below, against the protocol's own rule rather than a
			// boolean kept here (plans/os-88df7ab2.md D1).
			get(s)
		case transition.RunStartedVerb:
			w := get(s)
			w.starts = append(w.starts, pos)
		}
		// The deliberate exits are the protocol's, not a copy of them
		// (D2): transition.IsExit names the four verbs that legally end
		// an in_progress window.
		if transition.IsExit(rec.Event.Verb) {
			get(s).open = false
		}
	}
	// A subject still open (claim taken, no deliberate exit) is a silent
	// abandonment; a done contract is closed by construction.
	for s, w := range subj {
		if w.open {
			res.SilentAbandonments = append(res.SilentAbandonments, s)
		}
	}
	starts := make(map[string][]int, len(subj))
	for s, w := range subj {
		starts[s] = w.starts
	}
	res.UnreservedSpend = append(res.UnreservedSpend, unreservedSpend(tbl, records, starts)...)

	res.Clean = len(res.ChainViolations) == 0 && len(res.LostUpdates) == 0 &&
		len(res.SilentAbandonments) == 0 && len(res.GuardrailBreaches) == 0 &&
		len(res.UnreservedSpend) == 0
	return res
}

// unreservedSpend names every subject whose run was not fenced to an
// open, valid reservation of its own (plans/os-88df7ab2.md D1, D7).
//
// A start is fenced only if BOTH halves hold, because the fold and
// admission answer different questions:
//
//   - The fold recorded it. transition records a RunStartFact only
//     where the payload named a fence and a reservation it could read,
//     so a run.started the fold did not record cited nothing checkable
//     and is spend with no fence at all (D7). Counting records against
//     facts is what keeps a malformed raw start visible.
//   - admit.RunStartValid accepts it. A start cites ONE reservation
//     (RunStartFact.Reservation) and that predicate judges the citation
//     at the start's own position: the strict payload, the fence
//     against the active claim, the cited reservation's validity, and
//     BudgetViewAt proving it was not already closed there. Asking
//     instead "was some reservation open" is weaker than the protocol
//     and misses the fencing the bar exists to check: a start citing a
//     closed or absent reservation passes it whenever an unrelated
//     reservation is open (review finding on plan #309).
//
// The cost is the protocol's too, and it is not linear: each
// RunStartValid replays the keyring and the table over the start's
// prefix and derives a budget view there, so a chain of n records with
// r runs and k reservations costs about O(r*k*n) per subject (D6).
//
// Where the table could not be built the bar reports nothing: the
// chain-violation arm has already named that, and a bar cannot judge a
// chain it cannot fold (D4).
func unreservedSpend(tbl *transition.Table, records []*event.Record, starts map[string][]int) []string {
	if tbl == nil {
		return nil
	}
	subjects := make([]string, 0, len(starts))
	for s := range starts {
		subjects = append(subjects, s)
	}
	// Map iteration is unordered and this list is evidence: one order
	// on every run.
	sort.Strings(subjects)
	var out []string
	for _, s := range subjects {
		if len(starts[s]) == 0 {
			continue
		}
		state, ok := tbl.StateAt(records, s)
		if !ok {
			// The fold placed no state for a subject that started a
			// run: nothing fenced any of them.
			for range starts[s] {
				out = append(out, s)
			}
			continue
		}
		// The fold's facts by the position of the record each came
		// from, so a raw start is judged against its own fact rather
		// than against a count (D1).
		fact := make(map[int]transition.RunStartFact, len(state.RunStarts))
		for _, st := range state.RunStarts {
			fact[st.Pos] = st
		}
		for _, pos := range starts[s] {
			st, folded := fact[pos]
			if !folded {
				// The fold read no fence and no reservation from this
				// record, so there is no citation to judge: spend with
				// no fence, named before any citation check (D1).
				out = append(out, s)
				continue
			}
			if !admit.RunStartValid(records, tbl, s, st) {
				out = append(out, s)
			}
		}
	}
	return out
}

// foldAll replays the lifecycle verbs through the table, tracking each
// subject's state locally and returning the first illegal transition.
// It reads the verbs and subjects, never signatures, so it judges the
// chain the audit was handed rather than re-deriving a fold.
func foldAll(tbl *transition.Table, records []*event.Record) (int, error) {
	state := map[string]string{}
	for i, rec := range records {
		v := rec.Event.Verb
		if !tbl.IsLifecycleVerb(v) {
			continue
		}
		s := rec.Event.Subject
		to, err := tbl.Check(s, state[s], v)
		if err != nil {
			return i, err
		}
		state[s] = to
	}
	return len(records), nil
}
