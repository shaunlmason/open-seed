// The evidence-grade half of divergence detection
// (plans/os-8a5f14bb.md D2.5). These checks read the artifact store
// and the repository, which projection builds never do, and they are
// the ones that see divergence with no record to derive it from: a
// receipt that no longer retrieves, an observed merge that does not
// descend from the head a verdict attested, a target ref rewritten
// after the observation.
//
// They lived unexported in cmd/seed until the maintenance loop needed
// them too. A second copy would have meant two divergence surfaces
// that drift, and a maintenance pass built on the record-derived half
// alone would report clean over exactly the divergence the charter
// asks it to reconcile — green, and wrong (review finding on #203).
// One implementation, two callers.
package reconcile

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/shaunlmason/open-seed/next/internal/artifact"
	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/transition"
	"github.com/shaunlmason/open-seed/next/internal/verdict"
)

// Evidence runs the checks a projection build cannot: the cited
// receipt must be retrievable intact, the observed merge must relate
// to the attested head by clean ancestry, and the merged commit must
// still sit under the target tip.
func Evidence(id string, s transition.SubjectState, store *artifact.Store, repo string) []Finding {
	return EvidenceAt(id, s, store, repo, nil, nil)
}

// EvidenceAt is Evidence with the chain and its fold in hand, which the
// L3 reproduction needs (plans/os-99829835.md D5): `seed reconcile`
// passes them; a caller without them grades everything but that.
func EvidenceAt(id string, s transition.SubjectState, store *artifact.Store, repo string, records []*event.Record, fold *transition.Fold) []Finding {
	var out []Finding
	// The L3 reproduction (plans/os-99829835.md D1, D5): a verdict that
	// recorded deterministic-first verification cites a receipt that
	// recomputes from the verifier's own checkout to the cited digest;
	// one that does not is an L3 the evidence does not support. Sealed
	// subjects need the sealer's key to recompute and are `verdict
	// check --key`'s; the record half above still judges them.
	if s.Verdict != nil && s.Verdict.Levels && s.Verdict.Independence == string(transition.L3) && s.Sealed == nil && fold != nil {
		digest, err := reproduce(records, fold, s, id, repo)
		if err != nil || digest != s.Verdict.Receipt {
			why := "the recomputed receipt digest differs from the cited " + s.Verdict.Receipt
			if err != nil {
				why = err.Error()
			}
			out = append(out, Finding{Subject: id, Class: ClassIndependenceUnverified,
				Detail: fmt.Sprintf("the verdict at position %d records L3 and its receipt does not reproduce from the verifier's own checkout: %s — deterministic-first verification is what the reproduction proves", s.Verdict.Pos, why)})
		}
	}
	if s.Merged == nil || s.Merged.SHA == "" || s.Verdict == nil {
		// Record-derivable classes already cover chains this
		// incomplete; there is no evidence to grade.
		return out
	}
	merged := s.Merged.SHA
	// The attested comparison needs the receipt; the target check
	// below does not, so lost evidence never hides a rewritten target
	// when multiple failures coexist (review finding on #137).
	attestedHead := ""
	if body, err := store.Get(s.Verdict.Receipt); err != nil {
		out = append(out, Finding{Subject: id, Class: ClassEvidenceMissing,
			Detail: fmt.Sprintf("the receipt %s cited by the verdict at position %d is not retrievable intact: %v — the evidence a verdict points at must survive verbatim", s.Verdict.Receipt, s.Verdict.Pos, err)})
	} else {
		var rcpt struct {
			Head string `json:"head"`
		}
		if jerr := json.Unmarshal(body, &rcpt); jerr != nil || rcpt.Head == "" {
			out = append(out, Finding{Subject: id, Class: ClassEvidenceMissing,
				Detail: fmt.Sprintf("the stored receipt %s carries no attested head", s.Verdict.Receipt)})
		} else {
			attestedHead = rcpt.Head
		}
	}
	// Clean ancestry: fast-forward (observed == attested) or a true
	// merge commit (attested head an ancestor of the observed sha).
	// Anything else is a surfaced state, not a fabrication verdict:
	// rebase and squash flows land here by design in v0.
	if attestedHead != "" && merged != attestedHead && !gitAncestor(repo, attestedHead, merged) {
		out = append(out, Finding{Subject: id, Class: ClassAttestedDivergence,
			Detail: fmt.Sprintf("the observed merge %s is neither the attested head %s nor its descendant (receipt %s) — rebase and squash flows surface here until patch-equivalence reconciliation lands", merged, attestedHead, s.Verdict.Receipt)})
	}
	// Target rewrite: the merged commit must resolve and still sit
	// under the target tip (v0: the repository's checked-out default
	// branch). A force-push that dropped it is the charter's
	// detected-divergence case.
	if !gitResolves(repo, merged) || !gitAncestor(repo, merged, "HEAD") {
		out = append(out, Finding{Subject: id, Class: ClassTargetRewritten,
			Detail: fmt.Sprintf("the observed merge %s no longer resolves under the target tip — the target ref was rewritten after the observation at position %d", merged, s.Merged.Pos)})
	}
	return out
}

func gitResolves(repo, rev string) bool {
	return exec.Command("git", "-C", repo, "rev-parse", "--verify", "--quiet", rev+"^{commit}").Run() == nil
}

func gitAncestor(repo, ancestor, descendant string) bool {
	return exec.Command("git", "-C", repo, "merge-base", "--is-ancestor", ancestor, descendant).Run() == nil
}

// reproduce recomputes the receipt for the subject's bound submission
// exactly as `seed verdict check` does, through the verifier's own
// input seam, and returns its digest.
func reproduce(records []*event.Record, fold *transition.Fold, s transition.SubjectState, subject, repo string) (string, error) {
	in, err := verdict.InputFor(records, fold, s, subject, repo, 0)
	if err != nil {
		return "", err
	}
	r, err := verdict.Compute(in)
	if err != nil {
		return "", err
	}
	return r.Digest()
}
