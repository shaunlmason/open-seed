// The plan verbs (plans/os-16c1d142.md): the falsifiable-plan lint and
// the plan/implementation classifier as CI-invocable entrypoints. The
// lint reads content structure only; the classifier takes the shape a
// forge's changed-files list provides (args or newline-separated
// stdin) and refuses mixed PRs — making the check forge-required for
// self-hosted deployments is the Phase 12 protections reconciler's
// item. propose and approve (plans/os-6bd9ffff.md D5) append the two
// plan facts through the boundary, deriving the plan's content digest
// from the repository at the anchor so the fold can say whether the
// approval kept the planner's original decomposition.
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/shaunlmason/open-seed/next/internal/admit"
	"github.com/shaunlmason/open-seed/next/internal/curation"
	"github.com/shaunlmason/open-seed/next/internal/envelope"
	"github.com/shaunlmason/open-seed/next/internal/plan"
	"github.com/shaunlmason/open-seed/next/internal/transition"
)

func runPlan(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "plan requires a subverb: lint, classify, propose, approve"), stdout, stderr)
	}
	switch args[0] {
	case "lint":
		return runPlanLint(args[1:], stdout, stderr)
	case "classify":
		return runPlanClassify(args[1:], stdin, stdout, stderr)
	case "propose":
		return runPlanFact(args[1:], transition.PlanProposedVerb, "plan propose", stdout, stderr)
	case "approve":
		return runPlanFact(args[1:], transition.PlanApprovedVerb, "plan approve", stdout, stderr)
	default:
		return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("unknown plan subverb %q — lint, classify, propose, or approve", args[0])), stdout, stderr)
	}
}

// runPlanFact is propose and approve: one loop-seam act each, the
// anchor and (for the approval) the merged PR as the caller's own
// values, and the digest DERIVED from the repository at the anchor
// rather than asked for, so the figure the fold compares is the plan
// bytes' and never a caller's claim about them. The fence is derived
// too, cited exactly when a claim window is active (the fence matrix
// binds free events from the holder), and re-derived against every
// refreshed view on the remote path.
func runPlanFact(args []string, verb, name string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	f := bindLoopFlags(fs)
	anchor := fs.String("plan", "", "the plan artifact anchor, \"<path> @ <commit>\"")
	repo := fs.String("repo", "", "repository the anchor is read from: the digest is derived there")
	pr := fs.String("pr", "", "the merged plan PR as an anchor, \"<pr> @ <merged-commit>\" (approve only)")
	parseErr := fs.Parse(args)
	missing := ""
	switch {
	case *anchor == "" || *repo == "":
		missing = "and --plan \"<path> @ <commit>\" --repo <dir>"
		if verb == transition.PlanApprovedVerb {
			missing += " --pr \"<pr> @ <merged-commit>\""
		}
	case verb == transition.PlanApprovedVerb && strings.TrimSpace(*pr) == "":
		missing = "and --pr \"<pr> @ <merged-commit>\": an approval observes the plan PR's merge"
	case verb == transition.PlanProposedVerb && *pr != "":
		missing = "without --pr: a proposal precedes the PR, and the approval is what observes its merge"
	}
	if env := f.usage(name, parseErr, fs.NArg(), missing); env != nil {
		return render(env, stdout, stderr)
	}
	path, commit, ok := curation.AnchorParts(*anchor)
	if !ok {
		return render(envelope.Fail(envelope.ExitUsage, "usage",
			fmt.Sprintf("--plan %q is not an anchor: \"<path> @ <commit>\" (next/spec/plans.md)", *anchor)), stdout, stderr)
	}
	body, err := exec.Command("git", "-C", *repo, "show", commit+":"+path).Output()
	if err != nil {
		return render(envelope.Fail(envelope.ExitNotFound, "not_found",
			fmt.Sprintf("the anchor %q does not resolve in %s: the digest is the plan bytes' at the anchor, and an anchor the repository lacks has none to carry (next/spec/plans.md)", *anchor, *repo)), stdout, stderr)
	}
	sum := sha256.Sum256(body)
	digest := hex.EncodeToString(sum[:])
	signer, env := loopSigner(*f.keyPath, *f.as)
	if env != nil {
		return render(env, stdout, stderr)
	}
	ls, env := openLoopSession(f)
	if env != nil {
		return render(env, stdout, stderr)
	}
	defer ls.done()
	subject := *f.subject
	derive := func(ctx *admit.Context) ([]byte, *envelope.Envelope) {
		m := map[string]any{"plan": *anchor, "digest": digest}
		if verb == transition.PlanApprovedVerb {
			m["pr"] = *pr
		}
		if s, ok := ctx.Lifecycle.State(subject); ok && s.Claim != nil {
			m["fence"] = fmt.Sprintf("%d", s.Claim.Fence)
		}
		return mustJSON(m), nil
	}
	payload, _ := derive(ls.ctx)
	return ls.commit(f, loopAct{verb: verb, payload: payload, derive: derive, resultAt: func(int) map[string]any {
		return map[string]any{"subject": subject, "plan": *anchor, "digest": digest}
	}}, signer, stdout, stderr)
}

func runPlanLint(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "plan lint <file>"), stdout, stderr)
	}
	doc, err := os.ReadFile(args[0])
	if err != nil {
		return render(envelope.Fail(envelope.ExitUnreadable, "unreadable", fmt.Sprintf("cannot read plan: %v", err)), stdout, stderr)
	}
	findings := plan.Lint(doc)
	if len(findings) > 0 {
		parts := make([]string, 0, len(findings))
		for _, f := range findings {
			parts = append(parts, f.String())
		}
		return render(envelope.Fail(envelope.ExitClassificationRef, "classification_refused",
			"the plan is not falsifiable: "+strings.Join(parts, "; ")), stdout, stderr)
	}
	return render(envelope.OK(map[string]any{"plan": args[0], "falsifiable": true}), stdout, stderr)
}

func runPlanClassify(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	paths := args
	if len(args) == 1 && args[0] == "-" {
		paths = nil
		sc := bufio.NewScanner(stdin)
		for sc.Scan() {
			if line := strings.TrimSpace(sc.Text()); line != "" {
				paths = append(paths, line)
			}
		}
		if err := sc.Err(); err != nil {
			return render(envelope.Fail(envelope.ExitUnreadable, "unreadable", fmt.Sprintf("reading paths: %v", err)), stdout, stderr)
		}
	}
	if len(paths) == 0 {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "plan classify <path>... | -   (newline-separated paths on stdin)"), stdout, stderr)
	}
	class := plan.Classify(paths)
	if class == plan.ClassMixed {
		return render(envelope.Fail(envelope.ExitClassificationRef, "classification_refused",
			"plan and implementation PRs are structurally disjoint: a change set may touch exactly one plans/ file and nothing else, or no plans/ file at all (next/spec/plans.md)"), stdout, stderr)
	}
	return render(envelope.OK(map[string]any{"class": string(class), "paths": len(paths)}), stdout, stderr)
}
