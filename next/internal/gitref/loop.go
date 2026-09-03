// The optimistic append loop: fetch, re-link, re-sign, validate, push,
// retry on races (plans/os-62e2aa1d.md step 3). Losing a race never
// silently re-appends: every retry re-derives the tip, re-signs, and
// re-runs validation against the refreshed store, and a draft that fails
// re-validation surfaces its reason to the caller.
package gitref

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/keyring"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
)

// Draft is an unsigned event body: the loop owns prev (re-linked each
// attempt) and the caller owns everything else.
type Draft struct {
	V       string
	TS      string
	Actor   string
	Verb    string
	Subject string
	Payload []byte
}

// Signer signs one re-linked event; only the key holder can, which is what
// makes mutate-prev-without-resign unrepresentable.
type Signer func(e event.Event) (*event.Record, error)

// Validate inspects the refreshed store and the candidate record before
// the push; refusing here is how admission rules compose into the loop
// (Phase 2 wires the full rule set through this seam).
type Validate func(store *ledger.Store, rec *event.Record) error

// Result reports a won race.
type Result struct {
	Position int
	Commit   string
	Attempts int
}

// AppendLoop lands one draft on the remote ref, retrying non-fast-forward
// losses up to maxAttempts. The resolver verifies the record's signature
// at append (the genesis bootstrap or, from Phase 3, the keyring
// projection supplies it). The persisted verified head advances after
// each successful validate, and again after a winning push. Trailing
// verify options (e.g. ledger.WithSupportedVersions) apply to the
// per-attempt stream re-verification, so a client appending past a
// protocol upgrade verifies with the same set it admits with
// (plans/os-895bf828.md step 1).
func (c *Client) AppendLoop(draft Draft, sign Signer, resolve ledger.Resolver, validate Validate, maxAttempts int, vopts ...ledger.VerifyOption) (*Result, error) {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		tipCommit, err := c.Fetch()
		if err != nil {
			return nil, err
		}
		workDir, err := os.MkdirTemp("", "seed-gitref-*")
		if err != nil {
			return nil, err
		}
		res, err := c.attempt(draft, sign, resolve, validate, tipCommit, workDir, vopts)
		os.RemoveAll(workDir)
		if err == nil {
			res.Attempts = attempt
			return res, nil
		}
		if errors.Is(err, ErrNonFastForward) {
			lastErr = err
			continue
		}
		return nil, err
	}
	return nil, fmt.Errorf("%w after %d attempts: %v", ErrRetriesSpent, maxAttempts, lastErr)
}

func (c *Client) attempt(draft Draft, sign Signer, resolve ledger.Resolver, validate Validate, tipCommit, workDir string, vopts []ledger.VerifyOption) (*Result, error) {
	storeDir := filepath.Join(workDir, "ledger")
	if err := c.Materialize(tipCommit, storeDir); err != nil {
		return nil, err
	}
	store, err := ledger.Open(storeDir)
	if err != nil {
		return nil, err
	}
	if tipCommit != "" {
		if _, err := store.VerifyFromGenesis(resolve, vopts...); err != nil {
			return nil, fmt.Errorf("fetched stream failed verification: %w", err)
		}
	}
	tip, _, err := store.Tip()
	if err != nil {
		return nil, err
	}
	rec, err := sign(event.Event{
		V: draft.V, TS: draft.TS, Actor: draft.Actor,
		Verb: draft.Verb, Subject: draft.Subject,
		Payload: draft.Payload, Prev: tip,
	})
	if err != nil {
		return nil, err
	}
	if validate != nil {
		if err := validate(store, rec); err != nil {
			return nil, fmt.Errorf("draft failed re-validation against the refreshed tip: %w", err)
		}
	}
	if tipCommit != "" {
		if err := c.persistHead(tipCommit); err != nil {
			return nil, err
		}
	}
	// From seed/1 the keyring projection is the append-time resolver:
	// standing decides who signs at this tip, so an enrolled key appends
	// and a suspended or revoked one refuses, while seed/0 chains keep
	// the caller's root-seam resolver untouched.
	appendResolve := resolve
	if tipCommit != "" {
		var records []*event.Record
		if err := store.Records(func(pos int, r *event.Record) error {
			records = append(records, r)
			return nil
		}); err != nil {
			return nil, err
		}
		ring, ringActive, err := keyring.StateAt(records)
		if err != nil {
			return nil, err
		}
		if keyring.Applies(ringActive) && ring.Seeded() {
			appendResolve = ring.Resolver()
		}
	}
	pos, err := store.Append(rec, appendResolve)
	if err != nil {
		return nil, err
	}
	if c.proposer != nil {
		// The forge-hosted posture: the service, not this client, is
		// the ref's writer. The record proposed is the one just
		// validated, linked to the tip this attempt fetched; a moved
		// tip comes back as the race it is and the loop re-links. The
		// persisted head is left at the fetched tip: the commit the
		// service made is not in this git dir until the next fetch
		// verifies it, and persisting an object we do not hold would
		// turn the monotonic-head rule into a self-inflicted regression.
		res, err := c.proposer.Propose(c.Ref, []*event.Record{rec})
		if err != nil {
			return nil, err
		}
		if res.Position != pos {
			return nil, fmt.Errorf("%w: the service admitted at position %d where this proposer expected %d", ErrNonFastForward, res.Position, pos)
		}
		return res, nil
	}
	commit, err := c.CommitAndPush(storeDir, tipCommit, "ledger: "+draft.Verb)
	if err != nil {
		return nil, err
	}
	if err := c.persistHead(commit); err != nil {
		return nil, err
	}
	return &Result{Position: pos, Commit: commit}, nil
}
