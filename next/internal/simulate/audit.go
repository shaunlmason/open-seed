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
		open     bool
		reserved bool
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
		case ClaimTakenVerb:
			// A claim rides a published offer in the scheduling model
			// (SEED-NEXT.md II.9), but admission does not require one:
			// its claim arms are authoring isolation and the lifecycle
			// transition, and nothing there reads the subject's offers.
			// So an unoffered claim is not a guardrail breach; the bar
			// reports what the boundary refuses, and the scheduling
			// concern is internal/eval's ready-with-no-live-offers read
			// (plans/os-aaec6a3c.md D1, D3).
			w := get(s)
			w.open = true
			w.reserved = false
		case transition.BudgetReserveVerb:
			get(s).reserved = true
		case transition.RunStartedVerb:
			if w := get(s); !w.reserved {
				res.UnreservedSpend = append(res.UnreservedSpend, s)
			}
		}
		// The deliberate exits are the protocol's, not a copy of them
		// (D2): transition.IsExit names the four verbs that legally end
		// an in_progress window.
		if transition.IsExit(rec.Event.Verb) {
			get(s).open = false
		}
	}
	res.GuardrailBreaches = append(res.GuardrailBreaches, sealedAuthorClaims(tbl, records)...)

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

// sealedAuthorClaims names every subject claimed by the key that
// sealed its checks (plans/os-aaec6a3c.md D1). This is the guardrail
// admission actually enforces on the claim path — "the key that sealed
// the subject's checks never implements against them" — and it is
// visible in the chain alone, because the fold carries the sealing
// position and signer. A raw push is the case that matters: the
// boundary would have refused the claim, so a chain that holds one is
// a chain where the guardrail was bypassed.
func sealedAuthorClaims(tbl *transition.Table, records []*event.Record) []string {
	if tbl == nil {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	for _, rec := range records {
		if rec.Event.Verb != ClaimTakenVerb {
			continue
		}
		s := rec.Event.Subject
		if seen[s] {
			continue
		}
		state, ok := tbl.StateAt(records, s)
		if !ok || state.Sealed == nil {
			continue
		}
		if state.Sealed.Signer == rec.Event.Actor {
			seen[s] = true
			out = append(out, s)
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
