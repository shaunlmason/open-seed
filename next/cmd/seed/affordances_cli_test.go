package main

// The envelope-stamping drills (plans/os-f5551001.md; charter
// III.I row 1): append-path responses carry the affordances for
// their signing actor and subject at the stamped position, and the
// budget block on budget-active subjects — on ledger append and on
// a specialized append path (verdict render) alike — while
// responses without a ledger+key+subject context keep the empty
// list and the null block.

import (
	"fmt"
	"slices"
	"testing"
)

// conformance: III.I — every verb response includes the verbs
// currently legal for this actor on this subject, computed from the
// same rule set admission enforces, at the position it carries.
func TestAffordanceStamping(t *testing.T) {
	ld, src, base, specCommit, head, priv, _, keys, _ := offerLedger(t)
	rng := base + ".." + head
	offerFile(t, ld, priv, specCommit, "c-1")

	// A plain append stamps the signer's affordances at the response
	// tip; no budget facts stand, so the block stays null.
	e, code := runEnv(t, "ledger", "append", "--ledger", ld, "--key", priv,
		"--verb", "message.sent", "--subject", "c-1", "--payload", `{"n": 1}`)
	if code != 0 {
		t.Fatalf("append: %d %+v", code, e)
	}
	if !slices.Contains(e.Affordances, "contract.cancelled") || !slices.Contains(e.Affordances, "message.sent") {
		t.Fatalf("the operator's response lists the ready-state verbs: %v", e.Affordances)
	}
	if !slices.IsSorted(e.Affordances) {
		t.Fatalf("stamped affordances are sorted: %v", e.Affordances)
	}
	if e.Budget != nil {
		t.Fatalf("no budget facts stand, so the block stays null: %+v", e.Budget)
	}

	// After the claim, the holder's reserve response carries both
	// fields: the window verbs and the derived budget block.
	fencePos, err := admitAppend(t, ld, workerRawKey(22), "claim.taken", "c-1", `{}`)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	fence := fmt.Sprintf("%d", fencePos)
	e, code = runEnv(t, "ledger", "append", "--ledger", ld, "--key", keys["workerA"],
		"--verb", "budget.reserve", "--subject", "c-1", "--payload", `{"amount": "10", "fence": "`+fence+`"}`)
	if code != 0 {
		t.Fatalf("reserve: %d %+v", code, e)
	}
	for _, verb := range []string{"budget.settle", "claim.released", "submission.made"} {
		if !slices.Contains(e.Affordances, verb) {
			t.Fatalf("the holder's response lists %s inside the window: %v", verb, e.Affordances)
		}
	}
	if e.Budget == nil || e.Budget.Reserved != "10" || e.Budget.Remaining != "90" {
		t.Fatalf("the budget block derives from the open reservation against class small: %+v", e.Budget)
	}

	// The specialized append path stamps too: the verifier's verdict
	// response lists their next legal moves on the judged subject.
	if _, err := admitAppend(t, ld, workerRawKey(22), "submission.made", "c-1", fmt.Sprintf(
		`{"fence": "%d", "packet": {"acceptance": ["c-1 ok"], "decisions": [], "base": %q, "refs": [], "findings": []}}`,
		fencePos, rng)); err != nil {
		t.Fatalf("submission: %v", err)
	}
	e, code = runEnv(t, "verdict", "render", "--ledger", ld, "--subject", "c-1", "--repo", src,
		"--key", keys["verifier"], "--verdict", "pass")
	if code != 0 {
		t.Fatalf("verdict: %d %+v", code, e)
	}
	if !slices.Contains(e.Affordances, "message.sent") {
		t.Fatalf("the verifier's verdict response stamps affordances: %v", e.Affordances)
	}
	if e.Budget == nil || e.Budget.Reserved != "10" {
		t.Fatalf("the budget block rides the specialized path too: %+v", e.Budget)
	}

	// A read surface guesses no identity: without --key the list is
	// empty and the envelope block null; with the key it stamps.
	e, code = runEnv(t, "budget", "status", "--ledger", ld, "--subject", "c-1")
	if code != 0 {
		t.Fatalf("budget status: %d %+v", code, e)
	}
	if len(e.Affordances) != 0 || e.Budget != nil {
		t.Fatalf("keyless status keeps the advisory fields empty: %v %+v", e.Affordances, e.Budget)
	}
	e, code = runEnv(t, "budget", "status", "--ledger", ld, "--subject", "c-1", "--key", keys["workerA"])
	if code != 0 {
		t.Fatalf("budget status --key: %d %+v", code, e)
	}
	if len(e.Affordances) == 0 || e.Budget == nil {
		t.Fatalf("keyed status stamps both advisory fields: %v %+v", e.Affordances, e.Budget)
	}
}
