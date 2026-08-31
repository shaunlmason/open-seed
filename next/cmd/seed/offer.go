// The supervisor's offer surface (plans/os-c61c3392.md;
// next/spec/offers.md; SEED-NEXT.md §II.9): publish appends the
// eligibility-scoped, expiring invitation; list is the worker's poll —
// the pull half of offers-not-assignments, whose total wake failure
// costs only latency. Liveness is derived here, never stored: ready
// subject, unexpired, no later claim, and a signer who held the
// supervise boundary at the offer's own position.

package main

import (
	"crypto/ed25519"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/shaunlmason/open-seed/next/internal/admit"
	"github.com/shaunlmason/open-seed/next/internal/envelope"
	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/keyring"
	"github.com/shaunlmason/open-seed/next/internal/transition"
)

func runOffer(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "offer requires a subverb: publish or list"), stdout, stderr)
	}
	switch args[0] {
	case "publish":
		return runOfferPublish(args[1:], stdout, stderr)
	case "list":
		return runOfferList(args[1:], stdout, stderr)
	}
	return render(envelope.Fail(envelope.ExitUsage, "usage", "offer requires a subverb: publish or list"), stdout, stderr)
}

// offerEligibility is the published eligibility scope: empty arrays
// mean unscoped (any active worker, any tier), and omitempty keeps the
// canonical payload minimal.
type offerEligibility struct {
	Capabilities []string `json:"capabilities,omitempty"`
	Tiers        []string `json:"tiers,omitempty"`
}

func runOfferPublish(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("offer publish", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("ledger", "", "ledger directory")
	subject := fs.String("subject", "", "contract in ready to invite claims on")
	keyPath := fs.String("key", "", "OpenSSH ed25519 private key of the supervisor")
	expires := fs.String("expires", "", "RFC3339 expiry, strictly after now")
	var capabilities, tiers repeatedFlag
	fs.Var(&capabilities, "capability", "capability the taking worker must hold (repeatable; none = any active worker)")
	fs.Var(&tiers, "tier", "contract tier the offer covers (repeatable; none = any tier)")
	if err := fs.Parse(args); err != nil || *dir == "" || *subject == "" || *keyPath == "" || *expires == "" || fs.NArg() != 0 {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "offer publish requires --ledger <dir> --subject <id> --key <path> --expires <RFC3339> [--capability c]... [--tier t]..."), stdout, stderr)
	}
	keyBytes, err := os.ReadFile(*keyPath)
	if err != nil {
		return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("cannot read --key: %v", err)), stdout, stderr)
	}
	signer, err := event.ParsePrivateKey(keyBytes)
	if err != nil {
		return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("--key: %v", err)), stdout, stderr)
	}
	store, failEnv := openStore(*dir)
	if failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	ctx, err := admit.ContextAt(store)
	if err != nil {
		return render(envelope.Fail(envelope.ExitChainInvalid, "chain_invalid", err.Error()), stdout, stderr)
	}
	payload, err := json.Marshal(struct {
		Eligibility offerEligibility `json:"eligibility"`
		Expires     string           `json:"expires"`
	}{offerEligibility{Capabilities: capabilities, Tiers: tiers}, *expires})
	if err != nil {
		return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()), stdout, stderr)
	}
	fp, err := event.Fingerprint(signer.Public().(ed25519.PublicKey))
	if err != nil {
		return render(envelope.Fail(envelope.ExitUsage, "usage", err.Error()), stdout, stderr)
	}
	rec, err := event.Sign(event.Event{
		V:       ctx.Active,
		TS:      time.Now().UTC().Format(time.RFC3339),
		Actor:   fp,
		Verb:    transition.OfferPublishedVerb,
		Subject: *subject,
		Payload: json.RawMessage(payload),
		Prev:    ctx.Tip,
	}, signer)
	if err != nil {
		return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("cannot sign the offer: %v", err)), stdout, stderr)
	}
	if err := admit.Check(ctx, rec); err != nil {
		return render(stampTip(stampAffordances(remoteFailureEnvelope(err), *dir, signer, *subject), ctx.Count), stdout, stderr)
	}
	pos, err := store.Append(rec, ctx.Resolve)
	if err != nil {
		return render(stampAffordances(envelope.Fail(envelope.ExitChainInvalid, "chain_invalid", err.Error()), *dir, signer, *subject), stdout, stderr)
	}
	return render(stampTip(stampAffordances(envelope.OK(map[string]any{
		"subject": *subject,
		"expires": *expires,
	}), *dir, signer, *subject), pos+1), stdout, stderr)
}

// offerAuthorized replays the keyring to the offer's own position and
// checks the supervise boundary retroactively: the tolerant fold
// records raw pushes, so a foreign offer folds as a fact and must be
// inert at the consuming surface — the laundering-countermeasure
// shape (validate the signer against the authoring boundary where the
// fact is trusted), replayed against offers.
func offerAuthorized(records []*event.Record, o transition.OfferFact) bool {
	if o.Pos < 0 || o.Pos >= len(records) {
		return false
	}
	ring, _, err := keyring.StateAt(records[:o.Pos])
	return err == nil && ring != nil &&
		ring.HasAnyCapability(o.Signer, keyring.AcceptedCapabilities(transition.OfferPublishedVerb))
}

// offerRow is one live, eligible offer in the list envelope.
type offerRow struct {
	Subject      string   `json:"subject"`
	Position     string   `json:"position"`
	Tier         string   `json:"tier,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	Tiers        []string `json:"tiers,omitempty"`
	Expires      string   `json:"expires"`
}

func runOfferList(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("offer list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("ledger", "", "ledger directory")
	actor := fs.String("actor", "", "polling worker's fingerprint")
	nowFlag := fs.String("now", "", "RFC3339 liveness instant (default: now)")
	if err := fs.Parse(args); err != nil || *dir == "" || *actor == "" || fs.NArg() != 0 {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "offer list requires --ledger <dir> --actor <fingerprint> [--now <RFC3339>]"), stdout, stderr)
	}
	now := time.Now().UTC()
	if *nowFlag != "" {
		parsed, err := time.Parse(time.RFC3339, *nowFlag)
		if err != nil {
			return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("--now %q is not an RFC3339 timestamp", *nowFlag)), stdout, stderr)
		}
		now = parsed
	}
	st, failEnv := loadVerdictState(*dir)
	if failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	ring, _, err := keyring.StateAt(st.records)
	if err != nil {
		return render(envelope.Fail(envelope.ExitChainInvalid, "chain_invalid", err.Error()), stdout, stderr)
	}
	rows := []offerRow{}
	if ring != nil {
		if e, ok := ring.Get(*actor); ok && e.Standing == keyring.StandingActive {
			for _, subject := range st.fold.Subjects() {
				s, ok := st.fold.State(subject)
				if !ok {
					continue
				}
				for _, o := range s.LiveOffers(now) {
					if !eligibleFor(ring, *actor, s.Tier, o) || !offerAuthorized(st.records, o) {
						continue
					}
					rows = append(rows, offerRow{
						Subject:      subject,
						Position:     fmt.Sprintf("%d", o.Pos),
						Tier:         s.Tier,
						Capabilities: o.Capabilities,
						Tiers:        o.Tiers,
						Expires:      o.Expires,
					})
				}
			}
		}
	}
	return render(stampTip(envelope.OK(map[string]any{
		"actor":  *actor,
		"now":    now.Format(time.RFC3339),
		"offers": rows,
	}), st.count), stdout, stderr)
}

// eligibleFor applies the offer's scopes to the polling worker: every
// scoped capability must be held, with operator standing satisfying
// every scope (a root's implicit operator included) — scopes describe
// the taking lane, and admission already lets the operator act
// everywhere in it, so hiding offers from operators would let them
// claim work they cannot discover. The subject's filed tier must be
// in the scoped tier set. Empty scopes match any active worker, any
// tier.
func eligibleFor(ring *keyring.State, fp, tier string, o transition.OfferFact) bool {
	for _, c := range o.Capabilities {
		if !ring.HasAnyCapability(fp, []string{c, keyring.CapOperator}) {
			return false
		}
	}
	if len(o.Tiers) > 0 {
		found := false
		for _, t := range o.Tiers {
			if t == tier {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
