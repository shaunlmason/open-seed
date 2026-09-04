package admit

// The erasure fact's admission (plans/os-db5cd353.md D1, D2, D3, D6;
// SEED-NEXT.md III.A row 7): artifact.erased admits under the operator
// grant on the contract whose fold references the digest or on system,
// with its strict shape, once per artifact per subject, and is drafted
// in the affordances exactly where the subject references something.

import (
	"crypto/ed25519"
	"errors"
	"strings"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/erasure"
	"github.com/shaunlmason/open-seed/next/internal/version"
)

func erasedBody(digest, reason string) string {
	return `{"artifact": "` + digest + `", "reason": "` + reason + `"}`
}

// conformance: III.A row 7 (the erasure is itself an attributable
// event) — the record admits only from an operator, only for a digest
// the contract's fold references (or on system, where the operator's
// attestation is the reference), only once, and only in its strict
// shape; every refusal names why.
func TestErasureAdmitsReferencedOnceUnderOperator(t *testing.T) {
	ctx, k, step := sealFixture(t)
	ctx = step(k.sealer, version.Seed1, "check.sealed", "c-1", sealBody(testCommitment))

	var refusal *erasure.Error
	// The operator erases the commitment the subject references.
	if err := Check(ctx, draftV(t, k.signer, version.Seed1, erasure.Verb, "c-1", erasedBody(testCommitment, "a retention obligation"), ctx.Tip)); err != nil {
		t.Fatalf("the operator erases a referenced artifact: %v", err)
	}
	// Claim and plain standing refuse before any reference logic.
	var grant *OutOfGrantError
	for name, priv := range map[string]ed25519.PrivateKey{"claim": k.worker, "sealer": k.sealer, "plain": k.plain} {
		err := Check(ctx, draftV(t, priv, version.Seed1, erasure.Verb, "c-1", erasedBody(testCommitment, "x"), ctx.Tip))
		if !errors.As(err, &grant) {
			t.Fatalf("%s standing refuses out of grant: %v", name, err)
		}
	}
	// A digest the contract does not reference refuses naming what it
	// does reference.
	other := strings.Repeat("ab", 32)
	err := Check(ctx, draftV(t, k.signer, version.Seed1, erasure.Verb, "c-1", erasedBody(other, "x"), ctx.Tip))
	if !errors.As(err, &refusal) || !strings.Contains(err.Error(), "sealed commitment "+testCommitment) {
		t.Fatalf("an unreferenced digest refuses naming the references: %v", err)
	}
	// A contract the chain does not hold refuses.
	if err := Check(ctx, draftV(t, k.signer, version.Seed1, erasure.Verb, "c-none", erasedBody(testCommitment, "x"), ctx.Tip)); !errors.As(err, &refusal) || !strings.Contains(err.Error(), "no contract") {
		t.Fatalf("an unknown contract refuses: %v", err)
	}
	// On system the operator's attestation is the reference.
	if err := Check(ctx, draftV(t, k.signer, version.Seed1, erasure.Verb, erasure.SystemSubject, erasedBody(other, "referenced from a packet"), ctx.Tip)); err != nil {
		t.Fatalf("an erasure on system admits any well-formed digest: %v", err)
	}
	// The shape is strict.
	for name, body := range map[string]string{
		"not a digest":  erasedBody("sha256:"+testCommitment, "x"),
		"empty reason":  erasedBody(testCommitment, ""),
		"unknown field": `{"artifact": "` + testCommitment + `", "reason": "x", "body": "no"}`,
	} {
		if err := Check(ctx, draftV(t, k.signer, version.Seed1, erasure.Verb, "c-1", body, ctx.Tip)); !errors.As(err, &refusal) {
			t.Errorf("%s refuses as an erasure error: %v", name, err)
		}
	}
	// Once: the landed erasure is not recorded again, the refusal
	// naming the first.
	ctx = step(k.signer, version.Seed1, erasure.Verb, "c-1", erasedBody(testCommitment, "a retention obligation"))
	fact, ok := ctx.Lifecycle.Erasure("c-1", testCommitment)
	if !ok || fact.Signer != fpOf(t, k.signer) || fact.Reason != "a retention obligation" || fact.Pos != ctx.Count-1 {
		t.Fatalf("the fold keeps the erasure with its signer, reason and position: %+v %v", fact, ok)
	}
	err = Check(ctx, draftV(t, k.signer, version.Seed1, erasure.Verb, "c-1", erasedBody(testCommitment, "again"), ctx.Tip))
	if !errors.As(err, &refusal) || !strings.Contains(err.Error(), "was erased at position") {
		t.Fatalf("a second erasure refuses naming the first: %v", err)
	}
	// The fact changed no lifecycle state.
	if s, _ := ctx.Lifecycle.State("c-1"); s.State != "ready" {
		t.Fatalf("an erasure is a fact, never a transition: %s", s.State)
	}
}

// conformance: III.A row 7; III.I (affordances are computed from the
// same rule set) — the verb is drafted for the operator exactly where
// the subject references an artifact: on the sealed subject, not on
// an unsealed one, and never for a key without the grant.
func TestErasureIsAffordedWhereSomethingIsErasable(t *testing.T) {
	ctx, k, step := sealFixture(t)
	listed := func(priv ed25519.PrivateKey, subject string) bool {
		for _, v := range Affordances(ctx, priv, subject) {
			if v == erasure.Verb {
				return true
			}
		}
		return false
	}
	if listed(k.signer, "c-1") {
		t.Fatal("an unsealed subject references nothing by digest: nothing to erase")
	}
	ctx = step(k.sealer, version.Seed1, "check.sealed", "c-1", sealBody(testCommitment))
	if !listed(k.signer, "c-1") {
		t.Fatalf("the operator may erase the sealed subject's commitment: %v", Affordances(ctx, k.signer, "c-1"))
	}
	if listed(k.worker, "c-1") || listed(k.sealer, "c-1") {
		t.Fatal("only the operator is offered the erasure")
	}
	if !listed(k.signer, erasure.SystemSubject) {
		t.Fatal("on system the operator may erase any digest it attests")
	}
	ctx = step(k.signer, version.Seed1, erasure.Verb, "c-1", erasedBody(testCommitment, "done"))
	if listed(k.signer, "c-1") {
		t.Fatal("an erased artifact is not offered for erasure again")
	}
}
