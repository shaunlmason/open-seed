// The plan verbs (plans/os-16c1d142.md): the falsifiable-plan lint and
// the plan/implementation classifier as CI-invocable entrypoints. The
// lint reads content structure only; the classifier takes the shape a
// forge's changed-files list provides (args or newline-separated
// stdin) and refuses mixed PRs — making the check forge-required for
// self-hosted deployments is the Phase 12 protections reconciler's
// item.
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/shaunlmason/open-seed/next/internal/envelope"
	"github.com/shaunlmason/open-seed/next/internal/plan"
)

func runPlan(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "plan requires a subverb: lint, classify"), stdout, stderr)
	}
	switch args[0] {
	case "lint":
		return runPlanLint(args[1:], stdout, stderr)
	case "classify":
		return runPlanClassify(args[1:], stdin, stdout, stderr)
	default:
		return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("unknown plan subverb %q", args[0])), stdout, stderr)
	}
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
