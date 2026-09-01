package main

// The seam drill (plans/os-edf73d66.md D4b): admission-boundary
// responses journal their attempts beside the ledger, both
// outcomes — an admitted append journals one admitted line and a
// preview-refused append one refused line, each matching the
// rendered envelope's stamp and code — the specialized verdict
// path journals too, and read surfaces journal nothing.

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/refusals"
)

func TestAttemptJournaling(t *testing.T) {
	ld, src, base, specCommit, head, priv, rootKey, keys, _ := offerLedger(t)
	rng := base + ".." + head
	journal := func() []refusals.Entry {
		j, err := refusals.Load(filepath.Join(ld, refusals.File))
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			t.Fatal(err)
		}
		return j.Entries
	}
	rootFP, err := event.Fingerprint(rootKey.Public().(ed25519.PublicKey))
	if err != nil {
		t.Fatal(err)
	}

	// An admitted append journals one admitted line matching the
	// envelope's stamp, signer, verb, and subject.
	before := len(journal())
	e, code := runEnv(t, "ledger", "append", "--ledger", ld, "--key", priv,
		"--verb", "message.sent", "--subject", "c-1", "--payload", `{"n": 1}`)
	if code != 0 || e.Position == nil {
		t.Fatalf("append: %d %+v", code, e)
	}
	entries := journal()
	if len(entries) != before+1 {
		t.Fatalf("one admitted attempt journals one line: %d -> %d", before, len(entries))
	}
	last := entries[len(entries)-1]
	if last.Outcome != refusals.OutcomeAdmitted || last.Code != "" || last.Verb != "message.sent" ||
		last.Subject != "c-1" || last.Actor != rootFP || last.Position != *e.Position {
		t.Fatalf("the admitted line mirrors the envelope: %+v vs position %s", last, *e.Position)
	}

	// A preview-refused append journals one refused line carrying
	// the envelope's machine code and stamped position.
	before = len(journal())
	e, code = runEnv(t, "ledger", "append", "--ledger", ld, "--key", priv,
		"--verb", "actor.enrolled", "--subject", "c-9", "--payload", `{"garbage": true}`)
	if code == 0 || e.Error == nil || e.Position == nil {
		t.Fatalf("the malformed enroll refuses stamped: %d %+v", code, e)
	}
	entries = journal()
	if len(entries) != before+1 {
		t.Fatalf("one refused attempt journals one line: %d -> %d", before, len(entries))
	}
	last = entries[len(entries)-1]
	if last.Outcome != refusals.OutcomeRefused || last.Code != e.Error.Code ||
		last.Verb != "actor.enrolled" || last.Position != *e.Position {
		t.Fatalf("the refused line mirrors the envelope: %+v vs %+v", last, e.Error)
	}

	// A read surface is no attempt: budget status journals nothing,
	// whatever it answers (here not_found: c-1 carries no budget
	// facts), and neither does a verify.
	before = len(journal())
	runEnv(t, "budget", "status", "--ledger", ld, "--subject", "c-1", "--key", keys["workerA"])
	if e, code := runEnv(t, "ledger", "verify", "--ledger", ld); code != 0 {
		t.Fatalf("verify: %d %+v", code, e)
	}
	if got := len(journal()); got != before {
		t.Fatalf("read surfaces journal nothing: %d -> %d", before, got)
	}

	// The specialized verdict path journals its attempt too; the
	// library-level appends the fixture makes (admitAppend bypasses
	// the CLI boundary) journal nothing.
	offerFile(t, ld, priv, specCommit, "c-2")
	afterOffer := len(journal())
	fencePos, err := admitAppend(t, ld, workerRawKey(22), "claim.taken", "c-2", `{}`)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if _, err := admitAppend(t, ld, workerRawKey(22), "submission.made", "c-2", fmt.Sprintf(
		`{"fence": "%d", "packet": {"acceptance": ["c-2 ok"], "decisions": [], "base": %q, "refs": [], "findings": []}}`,
		fencePos, rng)); err != nil {
		t.Fatalf("submission: %v", err)
	}
	if got := len(journal()); got != afterOffer {
		t.Fatalf("library appends are not CLI attempts: %d -> %d", afterOffer, got)
	}
	e, code = runEnv(t, "verdict", "render", "--ledger", ld, "--subject", "c-2", "--repo", src,
		"--key", keys["verifier"], "--verdict", "pass")
	if code != 0 {
		t.Fatalf("verdict: %d %+v", code, e)
	}
	entries = journal()
	if len(entries) != afterOffer+1 {
		t.Fatalf("the verdict render journals one attempt: %d -> %d", afterOffer, len(entries))
	}
	last = entries[len(entries)-1]
	if last.Outcome != refusals.OutcomeAdmitted || last.Verb != "verdict.rendered" || last.Subject != "c-2" {
		t.Fatalf("the specialized path's line names its verb: %+v", last)
	}
}
