package main

// The attempts-journal ride-along (plans/os-edf73d66.md D1; charter
// III.I row 4): every admission-boundary response that stamps a
// position also journals its attempt beside the ledger — outcome
// from the envelope's own verdict, code from its error, position
// from its stamp — best-effort, both outcomes, so the report's
// refusal rate draws numerator and denominator from one population.
// Responses without a stamped position (usage errors, store-level
// failures before a tip was read) are not boundary attempts and
// journal nothing; read surfaces never call this.

import (
	"crypto/ed25519"
	"time"

	"github.com/shaunlmason/open-seed/next/internal/envelope"
	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/refusals"
)

// journalAttempt notes the attempt the envelope describes into the
// ledger directory's journal and hands the envelope back unchanged.
// Degrading, never failing: any missing piece journals nothing.
func journalAttempt(env *envelope.Envelope, dir string, key ed25519.PrivateKey, verb, subject string) *envelope.Envelope {
	if env == nil || env.Position == nil || dir == "" || verb == "" || subject == "" || len(key) != ed25519.PrivateKeySize {
		return env
	}
	fp, err := event.Fingerprint(key.Public().(ed25519.PublicKey))
	if err != nil {
		return env
	}
	e := refusals.Entry{
		TS:       time.Now().UTC().Format(time.RFC3339),
		Position: *env.Position,
		Actor:    fp,
		Verb:     verb,
		Subject:  subject,
		Outcome:  refusals.OutcomeAdmitted,
	}
	if env.Error != nil {
		e.Outcome = refusals.OutcomeRefused
		e.Code = env.Error.Code
	}
	refusals.Note(dir, e)
	return env
}
