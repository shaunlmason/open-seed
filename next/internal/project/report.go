// The report projection (plans/os-fecfb3f7.md step 4): the operational
// summary whose sections later phases extend. Sections that need
// Phase 5+ facts (claims, offers, budgets, expiry-vs-wedge,
// divergence) are named in the spec as extension points, not emitted
// empty.

package project

import (
	"encoding/json"

	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/halt"
	"github.com/shaunlmason/open-seed/next/internal/keyring"
)

// ReportFile is the report projection's one view file.
const ReportFile = "report.json"

// checkpointVerb mirrors keyring's private constant (keyring cannot be
// the exporter: it mirrors the same literal itself); the report drill
// pins parity by counting a real checkpoint record.
const checkpointVerb = "system.checkpoint"

// ReportChain is the chain section: the verified prefix's identity.
type ReportChain struct {
	Position      int    `json:"position"`
	Tip           string `json:"tip"`
	ActiveVersion string `json:"active_version"`
}

// ReportActors is the actor section: counts by standing plus roots.
type ReportActors struct {
	ByStanding map[string]int `json:"by_standing"`
	Roots      int            `json:"roots"`
	Total      int            `json:"total"`
}

// ReportHalt is the halt section; DeclaredPosition is present only
// while halted.
type ReportHalt struct {
	Halted           bool   `json:"halted"`
	DeclaredPosition *int   `json:"declared_position,omitempty"`
	By               string `json:"by,omitempty"`
}

// ReportCheckpoints is the checkpoint section; LastPosition is present
// only when a checkpoint exists.
type ReportCheckpoints struct {
	Count        int  `json:"count"`
	LastPosition *int `json:"last_position,omitempty"`
}

// ReportContracts is the work section: subject and event counts.
type ReportContracts struct {
	Subjects int `json:"subjects"`
	Events   int `json:"events"`
}

// ReportView is the report.json shape.
type ReportView struct {
	Chain       ReportChain       `json:"chain"`
	Actors      ReportActors      `json:"actors"`
	Halt        ReportHalt        `json:"halt"`
	Checkpoints ReportCheckpoints `json:"checkpoints"`
	Contracts   ReportContracts   `json:"contracts"`
}

// Report returns the report projection.
func Report() Projection {
	return Projection{Name: "report", Build: buildReport}
}

func buildReport(records []*event.Record) (map[string][]byte, error) {
	state, active, err := keyring.StateAt(records)
	if err != nil {
		return nil, err
	}
	view := ReportView{
		Chain:  ReportChain{Position: len(records), ActiveVersion: active},
		Actors: ReportActors{ByStanding: map[string]int{}},
	}
	if n := len(records); n > 0 {
		tip, err := records[n-1].Event.Hash()
		if err != nil {
			return nil, err
		}
		view.Chain.Tip = tip
	}
	for _, fp := range candidateFingerprints(records) {
		e, ok := state.Get(fp)
		if !ok {
			continue
		}
		view.Actors.ByStanding[string(e.Standing)]++
		view.Actors.Total++
		if e.Root {
			view.Actors.Roots++
		}
	}
	hs := halt.StateAt(records)
	view.Halt.Halted = hs.Halted
	view.Halt.By = hs.By
	if hs.Halted {
		for pos, rec := range records {
			if rec.Event.Verb == halt.DeclareVerb {
				p := pos
				view.Halt.DeclaredPosition = &p
			}
		}
	}
	for pos, rec := range records {
		if rec.Event.Verb == checkpointVerb {
			view.Checkpoints.Count++
			p := pos
			view.Checkpoints.LastPosition = &p
		}
	}
	entries := contractEntries(records)
	view.Contracts.Subjects = len(entries)
	for _, e := range entries {
		view.Contracts.Events += len(e.Events)
	}
	b, err := json.MarshalIndent(view, "", "  ")
	if err != nil {
		return nil, err
	}
	return map[string][]byte{ReportFile: append(b, '\n')}, nil
}
