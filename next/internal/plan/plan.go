// Package plan is the falsifiable-plan lint and the plan/implementation
// PR classifier (plans/os-16c1d142.md; SEED-NEXT.md Part II §6 "Plans
// as falsifiable change contracts"; conformance III.F "Plans are
// falsifiable: boundary set, retention set, validation commands for
// both, expected diff shape; missing retention fails lint; plan and
// implementation PRs are structurally disjoint"). The lint reads
// content structure only and never executes commands.
package plan

import (
	"fmt"
	"strings"
)

// Finding kinds: one per named violation, missing retention distinct
// because the charter singles it out ("a plan without a retention
// check does not lint").
const (
	KindMissingBoundary          = "missing_boundary"
	KindMissingRetention         = "missing_retention"
	KindMissingCommands          = "missing_commands"
	KindMissingDiffShape         = "missing_diff_shape"
	KindEmptySection             = "empty_section"
	KindCommandsMissingBoundary  = "commands_missing_boundary"
	KindCommandsMissingRetention = "commands_missing_retention"
)

// Finding is one lint violation.
type Finding struct {
	Kind   string
	Detail string
}

func (f Finding) String() string { return fmt.Sprintf("%s: %s", f.Kind, f.Detail) }

// The four required parts. A section marker is a markdown heading or a
// bold line whose text begins with the part's phrase, case-insensitive
// (the repository's own plan shape, the v1-continuity default).
var parts = []struct {
	phrase, missing string
}{
	{"boundary set", KindMissingBoundary},
	{"retention set", KindMissingRetention},
	{"validation commands", KindMissingCommands},
	{"expected diff shape", KindMissingDiffShape},
}

// markerText returns the marker phrase of a line, or "" for content.
func markerText(line string) string {
	t := strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(t, "#"):
		return strings.ToLower(strings.TrimSpace(strings.TrimLeft(t, "#")))
	case strings.HasPrefix(t, "**"):
		return strings.ToLower(strings.Trim(t, "*: "))
	}
	return ""
}

// Lint checks a plan document for the four falsifiable parts: each
// present and non-empty, with the commands section visibly covering
// the boundary AND the retention set.
func Lint(doc []byte) []Finding {
	lines := strings.Split(string(doc), "\n")
	// sections maps part phrase -> content lines until the next marker.
	sections := map[string][]string{}
	current := ""
	for _, line := range lines {
		if m := markerText(line); m != "" {
			current = ""
			for _, p := range parts {
				if strings.HasPrefix(m, p.phrase) {
					current = p.phrase
					if _, seen := sections[current]; !seen {
						sections[current] = []string{}
					}
					break
				}
			}
			continue
		}
		if current != "" && strings.TrimSpace(line) != "" {
			sections[current] = append(sections[current], line)
		}
	}
	var findings []Finding
	for _, p := range parts {
		content, ok := sections[p.phrase]
		if !ok {
			detail := fmt.Sprintf("the plan names no %q section", p.phrase)
			if p.missing == KindMissingRetention {
				detail = "a plan without a retention check does not lint: name what works and will be shown unharmed"
			}
			findings = append(findings, Finding{Kind: p.missing, Detail: detail})
			continue
		}
		if len(content) == 0 {
			findings = append(findings, Finding{Kind: KindEmptySection, Detail: fmt.Sprintf("the %q section is empty", p.phrase)})
		}
	}
	if cmds, ok := sections["validation commands"]; ok {
		if !hasLabeledCommand(cmds, "boundary:") {
			findings = append(findings, Finding{Kind: KindCommandsMissingBoundary, Detail: "the validation commands need a non-empty \"Boundary:\" command line covering the boundary set"})
		}
		if !hasLabeledCommand(cmds, "retention:") {
			findings = append(findings, Finding{Kind: KindCommandsMissingRetention, Detail: "the validation commands need a non-empty \"Retention:\" command line covering the retention set"})
		}
	}
	return findings
}

// hasLabeledCommand reports whether some line carries the label (after
// any list marker or emphasis) with a non-empty command behind it:
// prose that merely mentions the word cannot satisfy the
// falsifiability contract (next/spec/plans.md).
func hasLabeledCommand(lines []string, label string) bool {
	for _, l := range lines {
		t := strings.TrimLeft(strings.TrimSpace(l), "-*+ \t")
		if len(t) >= len(label) && strings.EqualFold(t[:len(label)], label) && strings.TrimSpace(t[len(label):]) != "" {
			return true
		}
	}
	return false
}

// Class is a changed-path set's classification.
type Class string

const (
	ClassPlan           Class = "plan"
	ClassImplementation Class = "implementation"
	ClassMixed          Class = "mixed"
)

// PlansDir is the plan documents' directory in an instantiation.
const PlansDir = "plans/"

// Classify maps a changed-path set to plan | implementation | mixed:
// exactly one path, under plans/, is a plan PR; zero plans/** paths is
// an implementation PR; anything else — a mixture, or two plan files —
// is mixed, and the CI entrypoint refuses it. Structural disjointness
// is the III.F contract; making the check forge-required for
// self-hosted deployments is the Phase 12 protections reconciler's
// item.
func Classify(paths []string) Class {
	plans := 0
	for _, p := range paths {
		if strings.HasPrefix(p, PlansDir) {
			plans++
		}
	}
	switch {
	case plans == 0:
		return ClassImplementation
	case plans == len(paths) && plans == 1:
		return ClassPlan
	default:
		return ClassMixed
	}
}
