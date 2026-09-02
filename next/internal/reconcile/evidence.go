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
	"github.com/shaunlmason/open-seed/next/internal/transition"
)

// Evidence runs the checks a projection build cannot: the cited
// receipt must be retrievable intact, the observed merge must relate
// to the attested head by clean ancestry, and the merged commit must
// still sit under the target tip.
func Evidence(id string, s transition.SubjectState, store *artifact.Store, repo string) []Finding {
	if s.Merged == nil || s.Merged.SHA == "" || s.Verdict == nil {
		// Record-derivable classes already cover chains this
		// incomplete; there is no evidence to grade.
		return nil
	}
	var out []Finding
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
