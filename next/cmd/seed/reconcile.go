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
	"flag"
	"fmt"
	"io"

	"github.com/shaunlmason/open-seed/next/internal/artifact"
	"github.com/shaunlmason/open-seed/next/internal/envelope"
	"github.com/shaunlmason/open-seed/next/internal/reconcile"
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
	for _, f := range reconcile.VerifyOverrides(st.records, st.fold) {
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
		findings = append(findings, reconcile.EvidenceAt(id, s, store, *repo, st.records, st.fold)...)
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
