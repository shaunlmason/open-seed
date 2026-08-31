// The report projection (plans/os-fecfb3f7.md step 4): the operational
// summary whose sections later phases extend. Sections that need
// Phase 5+ facts (claims, offers, budgets, expiry-vs-wedge,
// divergence) are named in the spec as extension points, not emitted
// empty.

package project

import (
	"encoding/json"
	"time"

	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/halt"
	"github.com/shaunlmason/open-seed/next/internal/keyring"
	"github.com/shaunlmason/open-seed/next/internal/obs"
	"github.com/shaunlmason/open-seed/next/internal/reconcile"
	"github.com/shaunlmason/open-seed/next/internal/transition"
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

// ReportObservationInputs echoes the declared inputs an observation
// section was computed from: reproducibility means naming them.
type ReportObservationInputs struct {
	AsOf               string `json:"as_of"`
	Digest             string `json:"digest"`
	ExpiryAfterSeconds int    `json:"expiry_after_seconds"`
	WedgeAfterSeconds  int    `json:"wedge_after_seconds"`
}

// ReportObservation is the expiry-vs-wedge section: one classification
// per active claim, read ONLY from the claim's own fence-keyed stream.
type ReportObservation struct {
	Inputs ReportObservationInputs       `json:"inputs"`
	Claims map[string]obs.Classification `json:"claims"`
}

// ReportView is the report.json shape. Observation is null when the
// rebuild declared no observation inputs: absence of data is stated,
// never fabricated (next/spec/observations.md).
type ReportView struct {
	Chain          ReportChain           `json:"chain"`
	Actors         ReportActors          `json:"actors"`
	Halt           ReportHalt            `json:"halt"`
	Checkpoints    ReportCheckpoints     `json:"checkpoints"`
	Contracts      ReportContracts       `json:"contracts"`
	Reconciliation *ReportReconciliation `json:"reconciliation"`
	Observation    *ReportObservation    `json:"observation"`
}

// ReportReconciliation is the record-derivable half of divergence
// detection (plans/os-6cdc15be.md; next/spec/reconciliation.md):
// findings by class over the lifecycle fold, with seed reconcile named
// for the evidence-grade rest — projection builds never read the
// artifact store or the repository. Null only when no work subject
// exists at all, so the section's presence is itself record-derived.
type ReportReconciliation struct {
	Findings []reconcile.Finding `json:"findings"`
	ByClass  map[string]int      `json:"by_class"`
	// EvidenceGrade names the surface carrying the checks a build
	// cannot run (attested heads, target rewrites).
	EvidenceGrade string `json:"evidence_grade"`
}

// Report returns the report projection. Version "2" added the
// observation section and the declared-inputs identity (the section is
// null on an input-free build, so version, not content, is what
// republishes existing prefixes); Version "3" the reconciliation
// section (plans/os-6cdc15be.md). Inputs marks it as the one
// input-consuming projection, everything else staying byte-identical
// with and without inputs by construction.
func Report() Projection {
	return Projection{Name: "report", Version: "3", Inputs: true, Build: buildReport}
}

// reportView is the report derivation shared by the JSON view and the
// cache tables.
func reportView(records []*event.Record) (*ReportView, error) {
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
	if len(entries) > 0 {
		table, err := transition.Default()
		if err != nil {
			return nil, err
		}
		findings := reconcile.Classify(table.FoldRecords(records))
		if findings == nil {
			findings = []reconcile.Finding{}
		}
		rec := &ReportReconciliation{Findings: findings, ByClass: map[string]int{},
			EvidenceGrade: "seed reconcile (attested heads, target rewrites; next/spec/reconciliation.md)"}
		for _, f := range findings {
			rec.ByClass[f.Class]++
		}
		view.Reconciliation = rec
	}
	return &view, nil
}

func buildReport(records []*event.Record, in Inputs) (map[string][]byte, error) {
	view, err := reportView(records)
	if err != nil {
		return nil, err
	}
	if in.Obs != nil {
		section, err := observationSection(records, in)
		if err != nil {
			return nil, err
		}
		view.Observation = section
	}
	b, err := json.MarshalIndent(view, "", "  ")
	if err != nil {
		return nil, err
	}
	return map[string][]byte{ReportFile: append(b, '\n')}, nil
}

// observationSection classifies every active claim against its own
// fence-keyed stream at the declared as_of. A claim whose stream is
// missing classifies no_data; a predecessor's stream under a dead
// fence can neither revive nor wedge the current claim, because
// StreamFor keys on (holder, fence) exactly.
func observationSection(records []*event.Record, in Inputs) (*ReportObservation, error) {
	table, err := transition.Default()
	if err != nil {
		return nil, err
	}
	digest, err := in.Digest()
	if err != nil {
		return nil, err
	}
	section := &ReportObservation{
		Inputs: ReportObservationInputs{
			AsOf:               in.AsOf.UTC().Format(time.RFC3339),
			Digest:             digest,
			ExpiryAfterSeconds: int(in.Thresholds.ExpiryAfter / time.Second),
			WedgeAfterSeconds:  int(in.Thresholds.WedgeAfter / time.Second),
		},
		Claims: map[string]obs.Classification{},
	}
	fold := table.FoldRecords(records)
	for _, subject := range fold.Subjects() {
		s, ok := fold.State(subject)
		if !ok || s.State != "in_progress" || s.Claim == nil {
			continue
		}
		stream, _ := in.Obs.StreamFor(s.Claim.Holder, obs.FormatFence(s.Claim.Fence))
		section.Claims[subject] = obs.Classify(stream, in.AsOf, in.Thresholds)
	}
	return section, nil
}
