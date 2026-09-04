package simulate

// The covered arm needs a chain admission would have taken
// (plans/os-88df7ab2.md D7): the bar judges a start by the reservation
// it cited, and a synthetic record with a {} payload cites nothing the
// protocol can read, so it is unfenced spend by construction. The
// tree already builds an admissible chain in internal/history, which
// carries budget.reserve, run.started, run.settled and submission.made
// under enrolled lane keys, so these drills read its records back
// rather than crafting their own.

import (
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/history"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
	"github.com/shaunlmason/open-seed/next/internal/transition"
)

// admittedChain generates a history of n contracts and returns the
// records the verification replay observed, in position order.
func admittedChain(t *testing.T, n int) []*event.Record {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "ledger")
	res, err := history.Generate(history.Spec{Seed: 7, Contracts: n, Dir: dir})
	if err != nil {
		t.Fatalf("generating an admitted chain: %v", err)
	}
	store, err := ledger.OpenReadOnly(dir)
	if err != nil {
		t.Fatal(err)
	}
	var records []*event.Record
	if _, err := store.VerifyFromGenesis(res.Resolve,
		ledger.WithObserver(func(_ int, rec *event.Record) { records = append(records, rec) })); err != nil {
		t.Fatalf("the generated chain must verify: %v", err)
	}
	return records
}

// conformance: III.R row 5 — a run fenced to the reservation it cited
// audits clean (plans/os-88df7ab2.md D1, AC1). The chain is
// admission-grade, so every start carries a fence and a cited
// reservation that was open at its own position.
func TestAdmittedChainAuditsClean(t *testing.T) {
	a := Audit(admittedChain(t, 3))
	if len(a.UnreservedSpend) != 0 {
		t.Errorf("a chain whose runs cite open reservations is not unreserved spend: %v", a.UnreservedSpend)
	}
	if len(a.ChainViolations) != 0 {
		t.Errorf("an admitted chain folds without an illegal transition: %v", a.ChainViolations)
	}
	// The guardrail bar names every subject on this fixture because the
	// history stages intent, contract and claim without publishing an
	// offer, and that bar requires a claim to ride one; admission takes
	// the chain regardless. Bar and boundary disagree there, which is
	// not this card's to settle (D5 moves one bar) and is carded on its
	// own.
}

// conformance: plans/os-88df7ab2.md D6 — the bar's cost over a chain
// of the shadow window's shape is a number in the record, not a claim:
// each RunStartValid replays the keyring and the table over the
// start's prefix, so the audit is not linear in the chain.
func TestAuditCostOnAShadowWindowChain(t *testing.T) {
	if testing.Short() {
		t.Skip("the measurement generates a 40-contract chain")
	}
	records := admittedChain(t, 40)
	start := time.Now()
	a := Audit(records)
	elapsed := time.Since(start)
	t.Logf("audited %d records in %s (unreserved_spend %d)", len(records), elapsed.Round(time.Millisecond), len(a.UnreservedSpend))
	if len(a.UnreservedSpend) != 0 {
		t.Errorf("the shadow-window chain is admitted throughout: %v", a.UnreservedSpend)
	}
	if elapsed > 10*time.Second {
		t.Errorf("the audit took %s over %d records, past the plan's ten-second ceiling: memoize the per-position replays (D6)", elapsed, len(records))
	}
}

// conformance: plans/os-88df7ab2.md D1, D3 — the arm the review on
// #309 named. A start is fenced to the reservation it CITED, not to
// whichever reservation happens to be open: a raw start citing a
// reservation the chain already settled is unreserved spend even
// while another reservation stands open on the same subject. The
// records are appended raw, which is the threat model: the boundary
// would have refused them, and the bar is what notices afterwards.
func TestStartCitingAClosedReservationIsUnfenced(t *testing.T) {
	records := admittedChain(t, 1)
	tbl, err := transition.Default()
	if err != nil {
		t.Fatal(err)
	}
	var subject string
	for _, rec := range records {
		if rec.Event.Verb == transition.RunStartedVerb {
			subject = rec.Event.Subject
			break
		}
	}
	if subject == "" {
		t.Fatal("the generated chain must start a run")
	}
	state, ok := tbl.StateAt(records, subject)
	if !ok || len(state.RunStarts) == 0 {
		t.Fatalf("the fold must place the subject's start: %v", ok)
	}
	base := Audit(records)
	if len(base.UnreservedSpend) != 0 {
		t.Fatalf("the generated chain is fenced before the plant: %v", base.UnreservedSpend)
	}
	// The chain settled its reservation (run.settled, then the close),
	// so citing it again after the fact cites something closed.
	cited := state.RunStarts[0]
	planted := append(append([]*event.Record(nil), records...),
		&event.Record{Event: event.Event{
			V: records[len(records)-1].Event.V, Verb: transition.RunStartedVerb, Subject: subject,
			Actor:   cited.Signer,
			Payload: []byte(`{"fence": "` + itoa(cited.Fence) + `", "reservation": "` + itoa(cited.Reservation) + `"}`),
		}})
	a := Audit(planted)
	if len(a.UnreservedSpend) == 0 {
		t.Fatalf("a second start citing the settled reservation is unfenced spend: %+v", a)
	}
	for _, s := range a.UnreservedSpend {
		if s != subject {
			t.Errorf("the bar names the subject that spent: %v", a.UnreservedSpend)
		}
	}
}

func itoa(n int) string { return strconv.Itoa(n) }
