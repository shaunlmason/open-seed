// The contract-detail projection (plans/os-fecfb3f7.md step 1): every
// work subject with its full event stream. The v0 work classifier is a
// prefix rule: verbs outside the system.* and actor.* governance
// namespaces are work vocabulary; Phase 5's transition table replaces
// the rule with explicit vocabulary (the spec and decision log say so).
// One file, not per-subject files: subjects are opaque strings, and
// mapping them to paths would trade the engine's path safety for an
// encoding scheme nothing consumes yet; the 4.3 cache is the
// lookup-throughput answer.

package project

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/transition"
)

// ContractsFile is the contract-detail projection's one view file.
const ContractsFile = "contracts.json"

// ContractEvent is one work event in a contract's stream. Actor is the
// signer fingerprint; Payload is the canonical payload verbatim.
type ContractEvent struct {
	Position int             `json:"position"`
	Verb     string          `json:"verb"`
	Actor    string          `json:"actor"`
	Payload  json.RawMessage `json:"payload"`
}

// ContractEntry is one work subject's stream, in chain order. State
// is the transition table's folded lifecycle state (null for a
// subject no lifecycle event ever validly created), and Anomalies
// counts the lifecycle events the table refused: tolerated in history
// per the cooperative posture, skipped by the fold, surfaced here,
// never silent (plans/os-d69a6c91.md).
type ContractEntry struct {
	Subject       string              `json:"subject"`
	State         *string             `json:"state"`
	Anomalies     int                 `json:"anomalies"`
	Claim         *ContractClaim      `json:"claim,omitempty"`
	Acceptance    *ContractAcceptance `json:"acceptance,omitempty"`
	Verdict       *ContractVerdict    `json:"verdict"`
	Requested     *string             `json:"requested"`
	Merged        *ContractMerge      `json:"merged"`
	FirstPosition int                 `json:"first_position"`
	LastPosition  int                 `json:"last_position"`
	Events        []ContractEvent     `json:"events"`
}

// ContractAcceptance is the folded acceptance spec: the artifact
// anchor, the executable flag, and whether gate evidence bound to the
// revision is present (or not required) — "may this spec run?" is a
// projection read Phase 6's verifier consumes (plans/os-73c00a50.md).
type ContractAcceptance struct {
	Ref        string `json:"ref"`
	Executable bool   `json:"executable"`
	Gated      bool   `json:"gated"`
}

// ContractVerdict is the fold's latest rendered verdict on a subject
// (plans/os-6cdc15be.md): the chain position (string per the envelope
// position convention), the pass/fail literal, and the cited receipt
// digest, so the reconciliation chain is independently checkable
// against the published view.
type ContractVerdict struct {
	Position string `json:"position"`
	Verdict  string `json:"verdict"`
	Receipt  string `json:"receipt"`
}

// ContractMerge is the admitted merge.observed: its chain position
// and the merged commit the observer recorded.
type ContractMerge struct {
	Position string `json:"position"`
	SHA      string `json:"sha"`
}

// ContractClaim is the active claim while a subject is in_progress:
// the holder's fingerprint and the fence (the admitted claim.taken
// position, string per the envelope position convention), so
// contention answers and stale-fence refusals are independently
// checkable against the published view (plans/os-5dc16a7c.md). Absent
// outside a claim window.
type ContractClaim struct {
	Holder string `json:"holder"`
	Fence  string `json:"fence"`
}

// Contracts returns the contract-detail projection. Version "2" added
// the folded state and anomaly count; Version "3" the claim object;
// Version "4" the acceptance field; Version "5" the fold's seed/1
// activation boundary (pre-activation records inert); Version "6" the
// reconciliation-chain facts (verdict, requested, merged;
// plans/os-6cdc15be.md) — each republishing under a new build id via
// the version-in-identity machinery.
func Contracts() Projection {
	return Projection{Name: "contracts", Version: "6", Build: buildContracts}
}

// isWorkVerb is the v0 classifier: everything outside the governance
// namespaces contributes to contract detail, keyed by subject.
func isWorkVerb(verb string) bool {
	return !strings.HasPrefix(verb, "system.") && !strings.HasPrefix(verb, "actor.")
}

func contractEntries(records []*event.Record) []*ContractEntry {
	bySubject := map[string]*ContractEntry{}
	var order []*ContractEntry
	for pos, rec := range records {
		e := &rec.Event
		if !isWorkVerb(e.Verb) {
			continue
		}
		entry, ok := bySubject[e.Subject]
		if !ok {
			entry = &ContractEntry{Subject: e.Subject, FirstPosition: pos}
			bySubject[e.Subject] = entry
			order = append(order, entry)
		}
		entry.LastPosition = pos
		entry.Events = append(entry.Events, ContractEvent{
			Position: pos,
			Verb:     e.Verb,
			Actor:    e.Actor,
			Payload:  e.Payload,
		})
	}
	return order
}

func buildContracts(records []*event.Record, _ Inputs) (map[string][]byte, error) {
	table, err := transition.Default()
	if err != nil {
		return nil, err
	}
	fold := table.FoldRecords(records)
	entries := contractEntries(records)
	out := make([]ContractEntry, 0, len(entries))
	for _, e := range entries {
		if s, ok := fold.State(e.Subject); ok {
			e.Anomalies = s.Anomalies
			if s.State != "" {
				st := s.State
				e.State = &st
			}
			if s.Claim != nil {
				e.Claim = &ContractClaim{Holder: s.Claim.Holder, Fence: fmt.Sprintf("%d", s.Claim.Fence)}
			}
			if s.Acceptance != nil {
				e.Acceptance = &ContractAcceptance{Ref: s.Acceptance.Ref, Executable: s.Acceptance.Executable, Gated: s.Acceptance.Gated}
			}
			if s.Verdict != nil {
				e.Verdict = &ContractVerdict{Position: fmt.Sprintf("%d", s.Verdict.Pos), Verdict: s.Verdict.Verdict, Receipt: s.Verdict.Receipt}
			}
			if s.Requested != nil {
				pos := fmt.Sprintf("%d", s.Requested.Pos)
				e.Requested = &pos
			}
			if s.Merged != nil {
				e.Merged = &ContractMerge{Position: fmt.Sprintf("%d", s.Merged.Pos), SHA: s.Merged.SHA}
			}
		}
		out = append(out, *e)
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, err
	}
	return map[string][]byte{ContractsFile: append(b, '\n')}, nil
}
