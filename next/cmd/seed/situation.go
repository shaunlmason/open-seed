package main

// The situation read (plans/os-52d5da3f.md D4; build plan Phase 9
// item 5): what is true for me now, in one position-stamped envelope
// — my standing obligations with what discharges each, the windows I
// hold, and the budget block the envelope already renders. With
// --since it is a COMPLETE CHANGE REPORT rather than a filtered list:
// arisen-or-changed rows plus an explicit discharged list keyed by
// the stable identity (subject, kind), so applying the response to a
// prior snapshot reproduces the standing set exactly. A delta of
// standing rows alone would leave a resuming lane holding a
// discharged obligation forever.
//
// Read-only and idempotent: it opens the ledger read-only, mutates
// nothing, and journals no attempt, because a read is not an
// admission-boundary attempt.

import (
	"crypto/ed25519"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"

	"github.com/shaunlmason/open-seed/next/internal/admit"
	"github.com/shaunlmason/open-seed/next/internal/envelope"
	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/keyring"
	"github.com/shaunlmason/open-seed/next/internal/obligation"
	"github.com/shaunlmason/open-seed/next/internal/project"
	"github.com/shaunlmason/open-seed/next/internal/transition"
)

// owedToMe reports whether a row is the caller's: their fingerprint
// outright, or a lane the caller holds the capability for. Lane-owed
// kinds exist because independence forbids naming an individual (a
// verdict is owed by the verifier lane, not by the claimant).
func owedToMe(row obligation.Row, fp string, lanes map[string]bool) bool {
	if row.OwedBy == fp {
		return true
	}
	return lanes[row.OwedBy]
}

func runSituation(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("situation", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("ledger", "", "ledger directory")
	keyPath := fs.String("key", "", "OpenSSH ed25519 private key: the actor the situation is read for")
	subject := fs.String("subject", "", "restrict to one contract")
	since := fs.String("since", "", "report only what changed at or after this position")
	if err := fs.Parse(args); err != nil || *dir == "" || fs.NArg() != 0 {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "situation requires --ledger <dir> [--key <path>] [--subject <id>] [--since <position>]"), stdout, stderr)
	}
	var sincePos int
	haveSince := *since != ""
	if haveSince {
		n, err := strconv.Atoi(*since)
		if err != nil || n < 0 {
			return render(envelope.Fail(envelope.ExitUsage, "usage",
				fmt.Sprintf("--since takes a ledger position, got %q", *since)), stdout, stderr)
		}
		sincePos = n
	}
	st, failEnv := loadVerdictState(*dir)
	if failEnv != nil {
		return render(failEnv, stdout, stderr)
	}

	// The caller's identity: a fingerprint alone cannot sign probes,
	// so affordance stamping needs the key itself, and a keyless read
	// guesses no identity — it reports the whole board unfiltered.
	fp := ""
	var signer ed25519.PrivateKey
	if *keyPath != "" {
		keyBytes, err := os.ReadFile(*keyPath)
		if err != nil {
			return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("cannot read --key: %v", err)), stdout, stderr)
		}
		signer, err = event.ParsePrivateKey(keyBytes)
		if err != nil {
			return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("cannot parse --key: %v", err)), stdout, stderr)
		}
		if fp, err = event.Fingerprint(signer.Public().(ed25519.PublicKey)); err != nil {
			return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()), stdout, stderr)
		}
	}
	lanes := map[string]bool{}
	if fp != "" {
		ring, _, err := keyring.StateAt(st.records)
		if err == nil {
			for lane, capability := range map[string]string{
				obligation.LaneVerifier:   "verdict",
				obligation.LaneObserver:   "observer",
				obligation.LaneSupervisor: "supervise",
				obligation.LaneOperator:   "operator",
			} {
				if ring.HasAnyCapability(fp, []string{capability}) {
					lanes[lane] = true
				}
			}
		}
	}

	rows, err := project.DeriveObligations(st.records)
	if err != nil {
		return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()), stdout, stderr)
	}
	mine := []obligation.Row{}
	for _, row := range rows {
		if *subject != "" && row.Subject != *subject {
			continue
		}
		if fp != "" && !owedToMe(row, fp, lanes) {
			continue
		}
		mine = append(mine, row)
	}

	// The delta: rows that arose at or after the cited position, plus
	// the removals a prior snapshot must drop. Removals are derived
	// by deriving the standing set AT the cited position and naming
	// what no longer stands, which is what makes the response
	// applicable rather than merely informative.
	obligations := mine
	discharged := []map[string]any{}
	unchanged := 0
	if haveSince {
		// --since cites a tip ORDINAL, so the prefix the lane last saw
		// is records[:pos+1], not records[:pos] (the same off-by-one
		// the envelope's position stamp turns on).
		prior, err := project.DeriveObligations(prefix(st.records, sincePos+1))
		if err != nil {
			return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()), stdout, stderr)
		}
		standing := map[string]bool{}
		for _, row := range mine {
			standing[row.Subject+"\x00"+row.Kind] = true
		}
		for _, row := range prior {
			if *subject != "" && row.Subject != *subject {
				continue
			}
			if fp != "" && !owedToMe(row, fp, lanes) {
				continue
			}
			if !standing[row.Subject+"\x00"+row.Kind] {
				discharged = append(discharged, map[string]any{
					"subject": row.Subject,
					"kind":    row.Kind,
					"at":      fmt.Sprintf("%d", st.count),
				})
			}
		}
		sort.Slice(discharged, func(i, j int) bool {
			a, b := discharged[i], discharged[j]
			if a["subject"] != b["subject"] {
				return a["subject"].(string) < b["subject"].(string)
			}
			return a["kind"].(string) < b["kind"].(string)
		})
		changed := []obligation.Row{}
		for _, row := range mine {
			if row.Since > sincePos {
				changed = append(changed, row)
			} else {
				unchanged++
			}
		}
		obligations = changed
	}

	out := []map[string]any{}
	for _, row := range obligations {
		out = append(out, map[string]any{
			"subject":       row.Subject,
			"kind":          row.Kind,
			"owed_by":       row.OwedBy,
			"since":         fmt.Sprintf("%d", row.Since),
			"discharged_by": row.DischargedBy,
		})
	}
	result := map[string]any{
		"actor":       fp,
		"obligations": out,
		"windows":     windowsHeld(st, fp, *subject),
	}
	if haveSince {
		result["since"] = *since
		result["discharged"] = discharged
		result["unchanged"] = fmt.Sprintf("%d", unchanged)
	}
	env := envelope.OK(result)
	if signer != nil && *subject != "" {
		env = stampAffordances(env, *dir, signer, *subject)
	}
	return render(stampTip(env, st.count), stdout, stderr)
}

// windowsHeld lists the subjects where the caller holds the active
// claim, with the fence every holder-signed event must cite: the
// argument a lane would otherwise read out of a projection by hand.
func windowsHeld(st *verdictState, fp, only string) []map[string]any {
	out := []map[string]any{}
	if fp == "" {
		return out
	}
	for _, subject := range st.fold.Subjects() {
		if only != "" && subject != only {
			continue
		}
		s, ok := st.fold.State(subject)
		if !ok || s.Claim == nil || s.Claim.Holder != fp {
			continue
		}
		out = append(out, map[string]any{
			"subject": subject,
			"fence":   fmt.Sprintf("%d", s.Claim.Fence),
			"state":   s.State,
		})
	}
	return out
}

// prefix returns the first n records: with n = position + 1 this is
// the prefix a lane that stopped at that stamped position last saw.
func prefix(records []*event.Record, pos int) []*event.Record {
	if pos >= len(records) {
		return records
	}
	return records[:pos]
}

// budgetOpenAt is the shared derivation, exposed for the drills.
func budgetOpenAt(records []*event.Record, table *transition.Table, subject string, s transition.SubjectState) []transition.ReservationFact {
	return admit.BudgetViewAt(records, table, subject, s).Open
}
