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

	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/keyring"
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

// ClassVerdictUnverified is record-derivable like the fold classes but
// needs the raw records: a folded verdict whose signer never satisfied
// the verifier boundary (no verdict grant at the verdict's position,
// or an implementing key). Admission refuses laundering such a verdict
// through the chain's later steps; this class surfaces the ones
// raw-pushed history already carries (review finding on the 6.2 task
// PR).
const ClassVerdictUnverified = "verdict_unverified"

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

// VerifyVerdicts replays the keyring to each folded verdict's own
// position and checks the verifier boundary retroactively: the signer
// held the verdict capability there and was no implementing key.
// Record-derivable and deterministic, so the report may carry it.
func VerifyVerdicts(records []*event.Record, f *transition.Fold) []Finding {
	var out []Finding
	for _, id := range f.Subjects() {
		s, ok := f.State(id)
		if !ok || s.Verdict == nil {
			continue
		}
		pos, signer := s.Verdict.Pos, s.Verdict.Signer
		if pos < 0 || pos >= len(records) {
			continue
		}
		ring, _, err := keyring.StateAt(records[:pos])
		if err != nil {
			continue
		}
		if !ring.HasAnyCapability(signer, keyring.AcceptedCapabilities(transition.VerdictRenderedVerb)) {
			out = append(out, Finding{Subject: id, Class: ClassVerdictUnverified,
				Detail: fmt.Sprintf("the verdict at position %d was signed by %s, which held no verdict grant there — it never passed the verifier boundary", pos, signer)})
			continue
		}
		if s.PriorClaimants[signer] || (s.Submission != nil && signer == s.Submission.Signer) {
			out = append(out, Finding{Subject: id, Class: ClassVerdictUnverified,
				Detail: fmt.Sprintf("the verdict at position %d was signed by implementing key %s — L1 independence never held", pos, signer)})
		}
	}
	return out
}

// Classify walks every folded subject in first-appearance order and
// appends the retroactive verdict verification.
func Classify(records []*event.Record, f *transition.Fold) []Finding {
	var out []Finding
	for _, id := range f.Subjects() {
		if s, ok := f.State(id); ok {
			out = append(out, Subject(id, s)...)
		}
	}
	return append(out, VerifyVerdicts(records, f)...)
}
