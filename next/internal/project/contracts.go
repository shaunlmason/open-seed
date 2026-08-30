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
	"strings"

	"github.com/shaunlmason/open-seed/next/internal/event"
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

// ContractEntry is one work subject's stream, in chain order.
type ContractEntry struct {
	Subject       string          `json:"subject"`
	FirstPosition int             `json:"first_position"`
	LastPosition  int             `json:"last_position"`
	Events        []ContractEvent `json:"events"`
}

// Contracts returns the contract-detail projection.
func Contracts() Projection {
	return Projection{Name: "contracts", Build: buildContracts}
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

func buildContracts(records []*event.Record) (map[string][]byte, error) {
	entries := contractEntries(records)
	out := make([]ContractEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, *e)
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, err
	}
	return map[string][]byte{ContractsFile: append(b, '\n')}, nil
}
