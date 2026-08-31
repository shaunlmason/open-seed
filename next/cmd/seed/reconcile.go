// The reconcile verb (plans/os-6cdc15be.md; next/spec/reconciliation.md):
// divergence detection as an invocable surface. The record-derivable
// classes come from internal/reconcile over the fold; the
// evidence-grade checks run only here, where the artifact store and
// the repository may be read: attested-head reconciliation against
// the cited receipt, and target-rewrite detection against the
// repository's checked-out default branch tip. Detection is a report
// at exit 0, never a refusal; Phase 9's maintenance loop consumes the
// same output on a schedule.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os/exec"

	"github.com/shaunlmason/open-seed/next/internal/artifact"
	"github.com/shaunlmason/open-seed/next/internal/envelope"
	"github.com/shaunlmason/open-seed/next/internal/reconcile"
	"github.com/shaunlmason/open-seed/next/internal/transition"
)

func runReconcile(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("reconcile", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("ledger", "", "ledger directory")
	repo := fs.String("repo", "", "repository the merges landed in")
	subject := fs.String("subject", "", "one contract (default: every folded subject)")
	artifacts := fs.String("artifacts", "", "artifact store root (default <repo>/next/var/artifacts)")
	if err := fs.Parse(args); err != nil || *dir == "" || *repo == "" || fs.NArg() != 0 {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "reconcile requires --ledger <dir> --repo <dir> [--subject <id>] [--artifacts <dir>]"), stdout, stderr)
	}
	st, failEnv := loadVerdictState(*dir)
	if failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	subjects := st.fold.Subjects()
	if *subject != "" {
		if _, ok := st.fold.State(*subject); !ok {
			return render(stampTip(envelope.Fail(envelope.ExitNotFound, "not_found",
				fmt.Sprintf("no contract %s in the fold", *subject)), st.count), stdout, stderr)
		}
		subjects = []string{*subject}
	}
	store := artifact.Open(artifactsDir(*artifacts, *repo))
	verdictFindings := map[string][]reconcile.Finding{}
	for _, f := range reconcile.VerifyVerdicts(st.records, st.fold) {
		verdictFindings[f.Subject] = append(verdictFindings[f.Subject], f)
	}
	for _, f := range reconcile.VerifySeals(st.records, st.fold) {
		verdictFindings[f.Subject] = append(verdictFindings[f.Subject], f)
	}
	var findings []reconcile.Finding
	checked := 0
	for _, id := range subjects {
		s, ok := st.fold.State(id)
		if !ok {
			continue
		}
		checked++
		findings = append(findings, reconcile.Subject(id, s)...)
		findings = append(findings, verdictFindings[id]...)
		findings = append(findings, evidenceFindings(id, s, store, *repo)...)
	}
	if findings == nil {
		findings = []reconcile.Finding{}
	}
	byClass := map[string]int{}
	for _, f := range findings {
		byClass[f.Class]++
	}
	return render(stampTip(envelope.OK(map[string]any{
		"subjects": fmt.Sprintf("%d", checked),
		"findings": findings,
		"by_class": byClass,
		"clean":    fmt.Sprintf("%t", len(findings) == 0),
	}), st.count), stdout, stderr)
}

// evidenceFindings runs the checks a projection build cannot: the
// cited receipt must be retrievable intact, the observed merge must
// relate to the attested head by clean ancestry, and the merged
// commit must still sit under the target tip.
func evidenceFindings(id string, s transition.SubjectState, store *artifact.Store, repo string) []reconcile.Finding {
	if s.Merged == nil || s.Merged.SHA == "" || s.Verdict == nil {
		// Record-derivable classes already cover chains this
		// incomplete; there is no evidence to grade.
		return nil
	}
	var out []reconcile.Finding
	merged := s.Merged.SHA
	// The attested comparison needs the receipt; the target check
	// below does not, so lost evidence never hides a rewritten target
	// when multiple failures coexist (review finding on this PR).
	attestedHead := ""
	if body, err := store.Get(s.Verdict.Receipt); err != nil {
		out = append(out, reconcile.Finding{Subject: id, Class: reconcile.ClassEvidenceMissing,
			Detail: fmt.Sprintf("the receipt %s cited by the verdict at position %d is not retrievable intact: %v — the evidence a verdict points at must survive verbatim", s.Verdict.Receipt, s.Verdict.Pos, err)})
	} else {
		var rcpt struct {
			Head string `json:"head"`
		}
		if jerr := json.Unmarshal(body, &rcpt); jerr != nil || rcpt.Head == "" {
			out = append(out, reconcile.Finding{Subject: id, Class: reconcile.ClassEvidenceMissing,
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
		out = append(out, reconcile.Finding{Subject: id, Class: reconcile.ClassAttestedDivergence,
			Detail: fmt.Sprintf("the observed merge %s is neither the attested head %s nor its descendant (receipt %s) — rebase and squash flows surface here until patch-equivalence reconciliation lands", merged, attestedHead, s.Verdict.Receipt)})
	}
	// Target rewrite: the merged commit must resolve and still sit
	// under the target tip (v0: the repository's checked-out default
	// branch). A force-push that dropped it is the charter's
	// detected-divergence case.
	if !gitResolves(repo, merged) || !gitAncestor(repo, merged, "HEAD") {
		out = append(out, reconcile.Finding{Subject: id, Class: reconcile.ClassTargetRewritten,
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
