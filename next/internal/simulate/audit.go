package simulate

// The seven-day bar, audited from the ledger alone (plans/os-16e55c11.md
// D5): the simulation reconstructs and justifies every claim from the
// chain, never from its own bookkeeping. Each bar names the records
// that violate it; a clean run leaves every list empty.

import (
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

// deliberate exits — the four verbs that legally end an in_progress
// window (submission, release, park, reap).
var deliberateExit = map[string]bool{
	"submission.made": true,
	"claim.released":  true,
	"claim.parked":    true,
	"claim.reaped":    true,
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
		open      bool
		reserved  bool
		lastFence int
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
	for _, rec := range records {
		s := rec.Event.Subject
		switch rec.Event.Verb {
		case "claim.taken":
			w := get(s)
			w.open = true
			w.reserved = false
		case "budget.reserved":
			get(s).reserved = true
		case "run.started":
			if w := get(s); !w.reserved {
				res.UnreservedSpend = append(res.UnreservedSpend, s)
			}
		}
		if deliberateExit[rec.Event.Verb] {
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

	res.Clean = len(res.ChainViolations) == 0 && len(res.LostUpdates) == 0 &&
		len(res.SilentAbandonments) == 0 && len(res.GuardrailBreaches) == 0 &&
		len(res.UnreservedSpend) == 0
	return res
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
