package main

// The seal verbs end-to-end (plans/os-3128535a.md): create commits and
// stores ciphertext (refusing an empty checks file); render on a
// sealed subject runs the sealed checks into the receipt behind the
// above-trivial gate; check holds the full recompute-and-mismatch
// guarantee over sealed transcripts and refuses non-recipients — the
// capability audit's CLI face; rotate re-encrypts to the current
// verifier keyring without touching the ledger; audit surfaces
// missing evidence, stale recipients, and foreign recipients at
// exit 0.

import (
	"crypto/ed25519"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/version"
)

func writeChecks(t *testing.T, lines ...string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "sealed-checks.txt")
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// driveToReview files a standard-tier contract, optionally seals it,
// then claims and submits through the library (admitted history is
// what the drills need). Returns the commitment ("" when unsealed).
func driveToReview(t *testing.T, ld, src, sealerKey string, rootKey ed25519.PrivateKey, subject, specCommit, rng, checksFile string) string {
	t.Helper()
	for _, step := range [][]string{
		{"intent.filed", `{"intent": "sealed drill", "tier": "standard", "budget": "small", "routing": "core"}`},
		{"contract.specified", fmt.Sprintf(`{"acceptance": {"ref": "accept.md @ %s", "executable": false}}`, specCommit)},
	} {
		if e, code := runEnv(t, "ledger", "append", "--ledger", ld, "--key", ld+"/../id_ed25519",
			"--verb", step[0], "--subject", subject, "--payload", step[1]); code != 0 {
			t.Fatalf("%s %s: %d %+v", subject, step[0], code, e)
		}
	}
	commitment := ""
	if checksFile != "" {
		e, code := runEnv(t, "seal", "create", "--ledger", ld, "--subject", subject, "--repo", src,
			"--checks", checksFile, "--key", sealerKey)
		if code != 0 || !e.OK {
			t.Fatalf("seal create on %s: %d %+v", subject, code, e)
		}
		commitment, _ = e.Result["commitment"].(string)
		if len(commitment) != 64 {
			t.Fatalf("create returns the commitment, got %q", commitment)
		}
	}
	fencePos := verdictLibAppend(t, ld, rootKey, "claim.taken", subject, `{}`)
	verdictLibAppend(t, ld, rootKey, "submission.made", subject, fmt.Sprintf(
		`{"fence": "%d", "packet": {"acceptance": ["%s ok"], "decisions": [], "base": %q, "refs": [], "findings": []}}`,
		fencePos, subject, rng))
	return commitment
}

func TestSealEndToEndCLI(t *testing.T) {
	dir, priv, _ := writeKeys(t)
	ld := filepath.Join(dir, "ledger")
	if _, code := runEnv(t, "init", "--ledger", ld, "--key", priv); code != 0 {
		t.Fatal("init failed")
	}
	src, base, specCommit, head := verdictRepo(t)
	v1Key, v1Pub, v1FP := writeWorkerKey(t, 9)
	sealKey, sealPub, sealFP := writeWorkerKey(t, 10)
	implKey, implPub, implFP := writeWorkerKey(t, 11)
	v2Key, v2Pub, v2FP := writeWorkerKey(t, 12)
	for _, step := range [][]string{
		{"system.protocol.upgraded", "system", `{"to": "` + version.Seed1 + `"}`},
		{"actor.enrolled", v1FP, fmt.Sprintf(`{"key": %q, "kind": "agent", "name": "verifier"}`, v1Pub)},
		{"actor.enrolled", sealFP, fmt.Sprintf(`{"key": %q, "kind": "agent", "name": "sealer"}`, sealPub)},
		{"actor.enrolled", implFP, fmt.Sprintf(`{"key": %q, "kind": "agent", "name": "implementer"}`, implPub)},
		{"actor.enrolled", v2FP, fmt.Sprintf(`{"key": %q, "kind": "agent", "name": "verifier2"}`, v2Pub)},
		{"actor.granted", v1FP, `{"capability": "verdict"}`},
		{"actor.granted", sealFP, `{"capability": "sealer"}`},
		{"actor.granted", implFP, `{"capability": "claim"}`},
	} {
		if e, code := runEnv(t, "ledger", "append", "--ledger", ld, "--key", priv,
			"--verb", step[0], "--subject", step[1], "--payload", step[2]); code != 0 {
			t.Fatalf("%s: %d %+v", step[0], code, e)
		}
	}
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i + 1)
	}
	rootKey := ed25519.NewKeyFromSeed(seed)
	rng := base + ".." + head
	marker := filepath.Join(t.TempDir(), "sealed-marker")

	// An empty checks file never commits: the vacuous-seal refusal.
	empty := writeChecks(t, "# only a comment", "")
	if e, code := runEnv(t, "seal", "create", "--ledger", ld, "--subject", "c-1", "--repo", src,
		"--checks", empty, "--key", sealKey); code != 19 {
		t.Fatalf("an empty checks file refuses at 19 spec_unrunnable: %d %+v", code, e)
	}

	// c-1: sealed happy path. The sealed check observes the marker so
	// the forged-transcript drill can flip its outcome later.
	checks := writeChecks(t, "# sealed", "test ! -f "+marker)
	commitment := driveToReview(t, ld, src, sealKey, rootKey, "c-1", specCommit, rng, checks)
	ctPath := filepath.Join(src, "next", "var", "artifacts", "sealed", commitment+".age")
	if _, err := os.Stat(ctPath); err != nil {
		t.Fatalf("create stores the ciphertext in the sealed bucket: %v", err)
	}
	// A second commitment on the same subject refuses at admission.
	if e, code := runEnv(t, "seal", "create", "--ledger", ld, "--subject", "c-1", "--repo", src,
		"--checks", checks, "--key", sealKey); code == 0 {
		t.Fatalf("a second seal must refuse: %+v", e)
	}

	// c-3: the above-trivial gate — a standard-tier subject with no
	// commitment does not render.
	driveToReview(t, ld, src, "", rootKey, "c-3", specCommit, rng, "")
	if e, code := runEnv(t, "verdict", "render", "--ledger", ld, "--subject", "c-3", "--repo", src,
		"--key", v1Key, "--verdict", "pass"); code != 24 {
		t.Fatalf("an above-trivial unsealed subject refuses render at 24 unsealed: %d %+v", code, e)
	}

	// c-1 renders: the sealed check runs into the receipt.
	e, code := runEnv(t, "verdict", "render", "--ledger", ld, "--subject", "c-1", "--repo", src,
		"--key", v1Key, "--verdict", "pass")
	if code != 0 || !e.OK {
		t.Fatalf("render on the sealed subject: %d %+v", code, e)
	}
	if e.Result["commitment"] != commitment || e.Result["sealed_transcripts"] != "1" || e.Result["red"] != "0" {
		t.Fatalf("the receipt carries the sealed half: %+v", e.Result)
	}

	// check: no key on a sealed subject refuses; an implementer key
	// cannot decrypt (exit 23 — the capability audit's CLI face); the
	// verifier key recomputes clean.
	if e, code = runEnv(t, "verdict", "check", "--ledger", ld, "--subject", "c-1", "--repo", src); code != 64 {
		t.Fatalf("check without a key on a sealed subject is a usage refusal: %d %+v", code, e)
	}
	if e, code = runEnv(t, "verdict", "check", "--ledger", ld, "--subject", "c-1", "--repo", src, "--key", implKey); code != 23 {
		t.Fatalf("the implementing key must not decrypt: %d %+v", code, e)
	}
	if e, code = runEnv(t, "verdict", "check", "--ledger", ld, "--subject", "c-1", "--repo", src, "--key", v1Key); code != 0 {
		t.Fatalf("the verifier's check recomputes clean: %d %+v", code, e)
	}

	// The forged/stale sealed-transcript drill: flip the sealed
	// check's real outcome and the recomputed receipt no longer
	// matches the cited digest — sealed transcripts are inside the
	// recompute-and-mismatch boundary.
	if err := os.WriteFile(marker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if e, code = runEnv(t, "verdict", "check", "--ledger", ld, "--subject", "c-1", "--repo", src, "--key", v1Key); code != 21 {
		t.Fatalf("a diverged sealed outcome is receipt_mismatch: %d %+v", code, e)
	}
	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}

	// c-2: a red sealed check forbids pass and leaves fail renderable.
	redChecks := writeChecks(t, "false")
	driveToReview(t, ld, src, sealKey, rootKey, "c-2", specCommit, rng, redChecks)
	if e, code = runEnv(t, "verdict", "render", "--ledger", ld, "--subject", "c-2", "--repo", src,
		"--key", v1Key, "--verdict", "pass"); code != 20 {
		t.Fatalf("pass over a red sealed check refuses at 20 checks_red: %d %+v", code, e)
	}
	if e, code = runEnv(t, "verdict", "render", "--ledger", ld, "--subject", "c-2", "--repo", src,
		"--key", v1Key, "--verdict", "fail"); code != 0 {
		t.Fatalf("fail stays renderable over red sealed checks: %d %+v", code, e)
	}

	// audit: clean while the sole granted verifier is the recipient.
	if e, code = runEnv(t, "seal", "audit", "--ledger", ld, "--repo", src); code != 0 || e.Result["clean"] != "true" {
		t.Fatalf("audit is clean before the keyring moves: %d %+v", code, e)
	}

	// The keyring rotates: verifier2 granted, verifier1 revoked. The
	// audit now names both drifts on each open sealed subject.
	for _, step := range [][]string{
		{"actor.granted", v2FP, `{"capability": "verdict"}`},
		{"actor.revoked", v1FP, `{"reason": "compromise drill"}`},
	} {
		if e, code := runEnv(t, "ledger", "append", "--ledger", ld, "--key", priv,
			"--verb", step[0], "--subject", step[1], "--payload", step[2]); code != 0 {
			t.Fatalf("%s: %d %+v", step[0], code, e)
		}
	}
	e, _ = runEnv(t, "seal", "audit", "--ledger", ld, "--repo", src)
	if by := classesOf(t, e); by["recipients_stale"] != 2 || by["recipient_foreign"] != 2 {
		t.Fatalf("the audit names stale and foreign recipients on both sealed subjects: %+v", e.Result)
	}

	// rotate with the old (still-able) identity re-encrypts to the
	// current keyring: the ledger is untouched, the commitment stands,
	// the revoked identity is locked out, the new verifier unseals.
	posBefore := *e.Position
	e, code = runEnv(t, "seal", "rotate", "--ledger", ld, "--repo", src, "--key", v1Key)
	if code != 0 {
		t.Fatalf("rotate: %d %+v", code, e)
	}
	rotated, _ := e.Result["rotated"].([]any)
	if len(rotated) != 2 {
		t.Fatalf("both open sealed subjects rotate: %+v", e.Result)
	}
	if *e.Position != posBefore {
		t.Fatalf("rotation writes no ledger events: position %s -> %s", posBefore, *e.Position)
	}
	if e, code = runEnv(t, "seal", "audit", "--ledger", ld, "--repo", src); code != 0 || e.Result["clean"] != "true" {
		t.Fatalf("audit is clean after rotate: %d %+v", code, e)
	}
	if e, code = runEnv(t, "verdict", "check", "--ledger", ld, "--subject", "c-1", "--repo", src, "--key", v1Key); code != 23 {
		t.Fatalf("the revoked identity is locked out after rotate: %d %+v", code, e)
	}
	if e, code = runEnv(t, "verdict", "check", "--ledger", ld, "--subject", "c-1", "--repo", src, "--key", v2Key); code != 0 {
		t.Fatalf("the current verifier unseals and recomputes clean after rotate: %d %+v", code, e)
	}

	// Broken seals: tampered ciphertext, then erased ciphertext. Both
	// are surfaced states, never silence.
	if err := os.WriteFile(ctPath, []byte("age-encryption.org/v1\n-> garbage\n--- x\njunk"), 0o600); err != nil {
		t.Fatal(err)
	}
	if e, code = runEnv(t, "verdict", "check", "--ledger", ld, "--subject", "c-1", "--repo", src, "--key", v2Key); code != 22 {
		t.Fatalf("a tampered ciphertext is 22 seal_broken: %d %+v", code, e)
	}
	if err := os.Remove(ctPath); err != nil {
		t.Fatal(err)
	}
	if e, code = runEnv(t, "verdict", "check", "--ledger", ld, "--subject", "c-1", "--repo", src, "--key", v2Key); code != 22 {
		t.Fatalf("an erased ciphertext is 22 seal_broken: %d %+v", code, e)
	}
	e, _ = runEnv(t, "seal", "audit", "--ledger", ld, "--repo", src)
	if by := classesOf(t, e); by["seal_evidence_missing"] != 1 {
		t.Fatalf("the audit surfaces the erased ciphertext: %+v", e.Result)
	}
}
