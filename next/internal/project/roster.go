// The roster projection (plans/os-4d5cacff.md step 2): the actor
// roster derived from Phase 3's keyring state — every keyring entry,
// genesis governance roots included (they are seeded from the genesis
// payload, not enrolled, so they carry root true with empty kind and
// name; review finding on #105). Candidate fingerprints derive from
// the chain itself (the genesis governance root plus every enrollment
// subject), so the projection needs no keyring iterator and stays a
// pure function of the records.

package project

import (
	"encoding/json"

	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/keyring"
)

// RosterFile is the roster projection's one view file.
const RosterFile = "roster.json"

// RosterEntry is one actor in the roster view. Kind stays the
// enrollment assertion (empty for genesis-seeded roots).
type RosterEntry struct {
	Fingerprint string   `json:"fingerprint"`
	Kind        string   `json:"kind"`
	Name        string   `json:"name"`
	Standing    string   `json:"standing"`
	Root        bool     `json:"root"`
	Grants      []string `json:"grants"`
}

// Roster returns the roster projection.
func Roster() Projection {
	return Projection{Name: "roster", Build: buildRoster}
}

func buildRoster(records []*event.Record) (map[string][]byte, error) {
	state, _, err := keyring.StateAt(records)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var order []string
	note := func(fp string) {
		if fp != "" && !seen[fp] {
			seen[fp] = true
			order = append(order, fp)
		}
	}
	for pos, rec := range records {
		if pos == 0 && rec.Event.Verb == "system.genesis" && rec.Event.Subject == "system" {
			var g struct {
				GovernanceRoot []struct {
					Fingerprint string `json:"fingerprint"`
				} `json:"governance_root"`
			}
			if err := json.Unmarshal(rec.Event.Payload, &g); err == nil {
				for _, rk := range g.GovernanceRoot {
					note(rk.Fingerprint)
				}
			}
		}
		if rec.Event.Verb == keyring.VerbEnrolled {
			note(rec.Event.Subject)
		}
	}
	entries := make([]RosterEntry, 0, len(order))
	for _, fp := range order {
		e, ok := state.Get(fp)
		if !ok {
			continue
		}
		grants := e.Grants
		if grants == nil {
			grants = []string{}
		}
		entries = append(entries, RosterEntry{
			Fingerprint: fp,
			Kind:        e.Kind,
			Name:        e.Name,
			Standing:    string(e.Standing),
			Root:        e.Root,
			Grants:      grants,
		})
	}
	// Chain order is deterministic already (first appearance); keep it,
	// so the roster reads as an enrollment history.
	b, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return nil, err
	}
	return map[string][]byte{RosterFile: append(b, '\n')}, nil
}
