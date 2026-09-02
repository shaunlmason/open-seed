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
	"github.com/shaunlmason/open-seed/next/internal/obligation"
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

// ClassRunUnsettled is a metered run whose fence never settled
// (plans/os-8a5f14bb.md D2). It is the one class Phase 9's maintenance
// loop adds, and it is CONSUMED from internal/obligation rather than
// re-derived here.
//
// The reason is the whole subtlety: post-close settlement is valid, so
// the flag is position-anchored — raised only once the subject has
// taken a subsequent claim window or reached a terminal state. A
// closed-without-settle predicate written fresh here would look
// obviously right and would file spurious findings against every park
// and reap in flight. Two copies of that anchoring is exactly the
// failure the projection exists to prevent.
const ClassRunUnsettled = "run_unsettled"

// ClassVerdictUnverified is record-derivable like the fold classes but
// needs the raw records: a folded verdict whose signer never satisfied
// the verifier boundary (no verdict grant at the verdict's position,
// or an implementing key). Admission refuses laundering such a verdict
// through the chain's later steps; this class surfaces the ones
// raw-pushed history already carries (review finding on the 6.2 task
// PR).
const ClassVerdictUnverified = "verdict_unverified"

// ClassUnsealed is an above-trivial subject at or past in_progress
// with no sealed-checks commitment (plans/os-3128535a.md): reported
// neutrally — the hard gate is verdict render's exit 24, and this
// class is the record-derivable surfacing beside it.
const ClassUnsealed = "unsealed"

// ClassSealUnverified is a folded seal whose signer, replayed to the
// seal's own position, held no sealer grant — a raw-pushed seal that
// never passed the authoring boundary. Render and check refuse to
// unseal it; this class surfaces it record-side.
const ClassSealUnverified = "seal_unverified"

// ClassOverridden is a subject whose merge chain ran through an
// operator override (plans/os-d2497eb7.md): reported neutrally and
// always by name — "never a disguised verdict" means the report says
// so wherever it happened.
const ClassOverridden = "overridden"

// ClassOverrideUnverified is a folded override whose signer, replayed
// to the override's own position, held no operator standing — a
// raw-pushed override that substitutes for nothing.
const ClassOverrideUnverified = "override_unverified"

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
	// An override is the sanctioned cover only when the chain actually
	// ran through it: the request must cite it (review finding on the
	// task PR — a raw override beside a skipped chain is divergence,
	// not the sanctioned path). Authenticity stays VerifyOverrides'
	// separate finding.
	overrideBacked := s.Override != nil && s.Requested != nil && s.Requested.CitedOverride == s.Override.Pos
	if merged && !passVerdict && !overrideBacked {
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
	if overrideBacked && merged {
		out = append(out, Finding{Subject: id, Class: ClassOverridden,
			Detail: fmt.Sprintf("the merge chain ran through the operator override at position %d (reason: %s) — an attributable substitute for a pass verdict, surfaced by name, never a disguised verdict", s.Override.Pos, s.Override.Reason)})
	}
	pastClaim := s.State == "in_progress" || s.State == "review" || s.State == "done"
	if pastClaim && s.Tier != transition.TrivialTier && s.Sealed == nil {
		out = append(out, Finding{Subject: id, Class: ClassUnsealed,
			Detail: fmt.Sprintf("tier %q with implementation under way and no sealed-checks commitment — above the trivial tier contracts carry sealed checks, and render refuses a verdict without one", s.Tier)})
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

// VerifySeals replays the keyring to each folded seal's own position
// and checks the authoring boundary retroactively: the signer held the
// sealer grant there. The fold's window rule already refuses seals
// after any claim, but it cannot see grants — a raw seal planted by an
// implementation-capable key inside the window would otherwise become
// the authoritative fact (review finding on the task PR: the
// verdict-laundering pattern, replayed against seals). Render and
// check refuse such a seal at use; this is the record-derivable
// surfacing beside them.
func VerifySeals(records []*event.Record, f *transition.Fold) []Finding {
	var out []Finding
	for _, id := range f.Subjects() {
		s, ok := f.State(id)
		if !ok || s.Sealed == nil {
			continue
		}
		pos, signer := s.Sealed.Pos, s.Sealed.Signer
		if pos < 0 || pos >= len(records) {
			continue
		}
		ring, _, err := keyring.StateAt(records[:pos])
		if err != nil {
			continue
		}
		if !ring.HasAnyCapability(signer, keyring.AcceptedCapabilities(transition.CheckSealedVerb)) {
			out = append(out, Finding{Subject: id, Class: ClassSealUnverified,
				Detail: fmt.Sprintf("the seal at position %d was signed by %s, which held no sealer grant there — it never passed the authoring boundary, and render refuses to unseal it", pos, signer)})
		}
	}
	return out
}

// VerifyOverrides replays the keyring to each folded override's own
// position and checks the operator boundary retroactively — the
// laundering countermeasure, applied to overrides from the start
// (plans/os-d2497eb7.md).
func VerifyOverrides(records []*event.Record, f *transition.Fold) []Finding {
	var out []Finding
	for _, id := range f.Subjects() {
		s, ok := f.State(id)
		if !ok || s.Override == nil {
			continue
		}
		pos, signer := s.Override.Pos, s.Override.Signer
		if pos < 0 || pos >= len(records) {
			continue
		}
		ring, _, err := keyring.StateAt(records[:pos])
		if err != nil {
			continue
		}
		if !ring.HasAnyCapability(signer, keyring.AcceptedCapabilities(transition.MergeOverriddenVerb)) {
			out = append(out, Finding{Subject: id, Class: ClassOverrideUnverified,
				Detail: fmt.Sprintf("the override at position %d was signed by %s, which held no operator standing there — it substitutes for nothing", pos, signer)})
		}
	}
	return out
}

// Classify walks every folded subject in first-appearance order and
// appends the retroactive verdict, seal, and override verification.
func Classify(records []*event.Record, f *transition.Fold) []Finding {
	var out []Finding
	for _, id := range f.Subjects() {
		if s, ok := f.State(id); ok {
			out = append(out, Subject(id, s)...)
		}
	}
	out = append(out, VerifyVerdicts(records, f)...)
	out = append(out, VerifySeals(records, f)...)
	return append(out, VerifyOverrides(records, f)...)
}

// Unsettled renders the obligation projection's run.unsettled rows as
// findings. It is a translation, not a derivation: the anchoring that
// decides WHEN a run counts as unsettled stays in internal/obligation,
// and this function would report nothing at all if that projection
// stopped emitting the kind.
func Unsettled(rows []obligation.Row) []Finding {
	var out []Finding
	for _, r := range rows {
		if r.Kind != obligation.KindRunUnsettled {
			continue
		}
		out = append(out, Finding{Subject: r.Subject, Class: ClassRunUnsettled,
			Detail: fmt.Sprintf("a metered run opened at position %d has no run.settled and the window can no longer settle it (owed by %s) — telemetry, never authority, but a run nobody settled is a run nobody metered", r.Since, r.OwedBy)})
	}
	return out
}
