// Package reconcile is the record-derivable half of divergence
// detection (plans/os-6cdc15be.md; next/spec/reconciliation.md;
// SEED-NEXT.md conformance III.G row 2): a pure classifier over the
// lifecycle fold, consumed by the report's reconciliation section and
// by seed reconcile, which adds the evidence-grade checks (attested
// heads, target rewrites) that need the artifact store and the
// repository — the two things projection builds never read. The
// scheduled, unattended runner over this surface is Phase 9's
// maintenance loop.
package reconcile

import (
	"fmt"

	"github.com/shaunlmason/open-seed/next/internal/transition"
)

// The record-derivable divergence classes. Classes are surfaced
// states, never accusations: unreconciled in particular is neutral,
// because no build carries a wall clock and "failed" versus "pending"
// is an age judgment that belongs to maintenance thresholds.
const (
	ClassMergeWithoutVerdict = "merge_without_verdict"
	ClassChainSkipped        = "chain_skipped"
	ClassUnreconciled        = "unreconciled"
)

// The evidence-grade classes seed reconcile adds over the artifact
// store and the repository; named here so both surfaces share one
// vocabulary, computed only where those reads are allowed.
const (
	ClassEvidenceMissing    = "evidence_missing"
	ClassAttestedDivergence = "attested_divergence"
	ClassTargetRewritten    = "target_rewritten"
)

// Finding is one surfaced divergence on one subject.
type Finding struct {
	Subject string `json:"subject"`
	Class   string `json:"class"`
	Detail  string `json:"detail"`
}

// Subject classifies one folded subject. A subject with a complete,
// consistent chain (or no chain activity at all) yields nothing.
func Subject(id string, s transition.SubjectState) []Finding {
	var out []Finding
	merged := s.Merged != nil || s.State == "done"
	passVerdict := s.Verdict != nil && s.Verdict.Verdict == "pass"
	if merged && !passVerdict {
		detail := "the subject reached done with no admitted pass verdict"
		if s.Verdict != nil {
			detail = fmt.Sprintf("the subject reached done and the admitted verdict at position %d is %q", s.Verdict.Pos, s.Verdict.Verdict)
		}
		out = append(out, Finding{Subject: id, Class: ClassMergeWithoutVerdict, Detail: detail})
	}
	if merged && passVerdict && (s.Requested == nil || s.Requested.CitedVerdict != s.Verdict.Pos) {
		out = append(out, Finding{Subject: id, Class: ClassChainSkipped,
			Detail: fmt.Sprintf("the merge was observed with no merge.requested citing the pass verdict at position %d — each chain step is its own event", s.Verdict.Pos)})
	}
	if !merged && passVerdict {
		out = append(out, Finding{Subject: id, Class: ClassUnreconciled,
			Detail: fmt.Sprintf("the pass verdict at position %d has no observed merge yet — pending or diverged is an age judgment for maintenance, not this classifier", s.Verdict.Pos)})
	}
	return out
}

// Classify walks every folded subject in first-appearance order.
func Classify(f *transition.Fold) []Finding {
	var out []Finding
	for _, id := range f.Subjects() {
		if s, ok := f.State(id); ok {
			out = append(out, Subject(id, s)...)
		}
	}
	return out
}
