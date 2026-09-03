package simulate

// The audit reconstructs the five bars from the ledger alone
// (plans/os-16e55c11.md D5, AC5/AC6): a clean chain passes every bar,
// and each bar's violation, planted once, is named. These craft the
// records directly — the audit reads the chain, not signatures — so a
// bar can be exercised without a full deployment.

import (
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/event"
)

func rec(verb, subject string) *event.Record {
	return &event.Record{Event: event.Event{Verb: verb, Subject: subject, Payload: []byte("{}")}}
}

// happy is a subject driven cleanly to a submission (a deliberate exit).
func happy(subject string) []*event.Record {
	return []*event.Record{
		rec("intent.filed", subject),
		rec("contract.specified", subject),
		rec("offer.published", subject),
		rec("claim.taken", subject),
		rec("budget.reserved", subject),
		rec("run.started", subject),
		rec("submission.made", subject),
	}
}

func TestAuditCleanRun(t *testing.T) {
	a := Audit(happy("c-1"))
	if !a.Clean {
		t.Fatalf("a clean chain must pass every bar: %+v", a)
	}
}

func TestAuditCatchesSilentAbandonment(t *testing.T) {
	// A claim taken and never closed by a deliberate exit.
	recs := []*event.Record{
		rec("intent.filed", "c-1"),
		rec("contract.specified", "c-1"),
		rec("offer.published", "c-1"),
		rec("claim.taken", "c-1"),
		rec("budget.reserved", "c-1"),
	}
	a := Audit(recs)
	if len(a.SilentAbandonments) != 1 || a.SilentAbandonments[0] != "c-1" {
		t.Fatalf("an unclosed window must be a silent abandonment, got %+v", a.SilentAbandonments)
	}
	if a.Clean {
		t.Fatal("a run with a silent abandonment is not clean")
	}
}

func TestAuditCatchesUnreservedSpend(t *testing.T) {
	// run.started with no covering budget.reserved.
	recs := []*event.Record{
		rec("intent.filed", "c-1"),
		rec("contract.specified", "c-1"),
		rec("offer.published", "c-1"),
		rec("claim.taken", "c-1"),
		rec("run.started", "c-1"),
		rec("submission.made", "c-1"),
	}
	a := Audit(recs)
	if len(a.UnreservedSpend) != 1 || a.UnreservedSpend[0] != "c-1" {
		t.Fatalf("a run.started with no reservation must be unreserved spend, got %+v", a.UnreservedSpend)
	}
}

func TestAuditCatchesChainViolation(t *testing.T) {
	// claim.taken with no prior contract: an illegal transition from the
	// empty state.
	recs := []*event.Record{rec("claim.taken", "c-9")}
	a := Audit(recs)
	if len(a.ChainViolations) == 0 {
		t.Fatal("an illegal transition must be a chain violation")
	}
}

func TestAuditCatchesEmptyChain(t *testing.T) {
	a := Audit(nil)
	if len(a.LostUpdates) == 0 {
		t.Fatal("an empty chain is a lost-everything")
	}
}

func TestAuditCatchesUnofferedClaim(t *testing.T) {
	// A claim with no preceding offer.published is a guardrail breach.
	recs := []*event.Record{
		rec("intent.filed", "c-1"),
		rec("contract.specified", "c-1"),
		rec("claim.taken", "c-1"),
		rec("budget.reserved", "c-1"),
		rec("submission.made", "c-1"),
	}
	a := Audit(recs)
	if len(a.GuardrailBreaches) != 1 || a.GuardrailBreaches[0] != "c-1" {
		t.Fatalf("a claim with no offer must be a guardrail breach, got %+v", a.GuardrailBreaches)
	}
}

func TestCatalogIsDeterministicInSeed(t *testing.T) {
	a := catalog(0)
	b := catalog(0)
	if len(a) != len(b) || len(a) == 0 {
		t.Fatal("catalog must be non-empty and stable")
	}
	for i := range a {
		if a[i].name != b[i].name {
			t.Fatal("catalog(seed) must be deterministic")
		}
	}
	// A different seed rotates the draw.
	if catalog(1)[0].name == catalog(0)[0].name && len(shipped) > 1 {
		t.Error("a different seed should rotate the first intent")
	}
}

func TestAuditMidChainViolationAndMultiSubject(t *testing.T) {
	// A subject that reaches review then illegally takes a claim again:
	// foldAll catches the mid-chain illegal transition. A second, clean
	// subject in the same chain is unaffected.
	recs := append(happy("c-1"),
		rec("intent.filed", "c-2"),
		rec("contract.specified", "c-2"),
		rec("claim.taken", "c-2"), // illegal: c-2 has no offer path here, and this is from ready
	)
	// Make c-2's claim illegal by placing it with no prior 'ready' via a
	// duplicate birth.
	recs = append(recs, rec("intent.filed", "c-1")) // birth on an existing subject: illegal
	a := Audit(recs)
	if len(a.ChainViolations) == 0 {
		t.Fatal("a second birth on an existing subject is a chain violation")
	}
}
