// Package retirement reads the v1 retirement plan
// (docs/v1-retirement.md; decisions/0006-v1-retirement.md) and holds
// its deletion inventory to the tree. The plan's stage 4 removes the
// v1 template in one pull request and reads its list of paths from
// that document, so a path that has since moved or been renamed would
// be discovered at the deletion rather than before it. That is what
// stage 2 exists to prevent: the inventory is pinned here, under make
// check, while v1 is still running.
//
// The parser is deliberately narrow. Only the "Deleted at stage 4"
// table is load-bearing, because only it is read as instructions. The
// "Edited, not deleted" and "Kept, permanently" sections carry prose
// and non-path entries (a git ref, a range of decision records), so
// holding them to the filesystem would be a category error.
package retirement

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// PlanPath is the repo-relative path of the retirement plan.
const PlanPath = "docs/v1-retirement.md"

// DeletionHeading opens the load-bearing table. The parser reads rows
// until the next bold heading, so the section's boundary is the
// document's own structure rather than a line count.
const DeletionHeading = "**Deleted at stage 4.**"

// MinEntries guards against a parser that silently matches nothing:
// the inventory is large, and a run that finds one row has found a
// broken document, not a shrunken v1.
const MinEntries = 10

// Entry is one row of the deletion inventory: the paths the row names
// and the description that justifies removing them.
type Entry struct {
	Paths []string
	What  string
	Line  int
}

var (
	rowRE  = regexp.MustCompile(`^\|(.+)\|(.+)\|$`)
	codeRE = regexp.MustCompile("`([^`]+)`")
)

// Parse reads the deletion inventory out of the plan document. It
// returns the rows in document order.
func Parse(doc string) ([]Entry, error) {
	lines := strings.Split(doc, "\n")
	start := -1
	for i, ln := range lines {
		if strings.TrimSpace(ln) == DeletionHeading {
			start = i
			break
		}
	}
	if start < 0 {
		return nil, fmt.Errorf("the plan has no %q section", DeletionHeading)
	}

	var out []Entry
	for i := start + 1; i < len(lines); i++ {
		ln := strings.TrimSpace(lines[i])
		// The next bold heading ends the section.
		if strings.HasPrefix(ln, "**") && ln != DeletionHeading {
			break
		}
		m := rowRE.FindStringSubmatch(ln)
		if m == nil {
			continue
		}
		cell, what := strings.TrimSpace(m[1]), strings.TrimSpace(m[2])
		// Skip the header row and its separator.
		if cell == "path" || strings.HasPrefix(cell, "---") {
			continue
		}
		var paths []string
		for _, c := range codeRE.FindAllStringSubmatch(cell, -1) {
			paths = append(paths, strings.TrimSpace(c[1]))
		}
		if len(paths) == 0 {
			return nil, fmt.Errorf("line %d: an inventory row names no path in backticks: %q", i+1, ln)
		}
		if what == "" {
			return nil, fmt.Errorf("line %d: the row for %v says nothing about what it is", i+1, paths)
		}
		out = append(out, Entry{Paths: paths, What: what, Line: i + 1})
	}
	if len(out) < MinEntries {
		return nil, fmt.Errorf("the deletion inventory parsed %d rows, fewer than the %d a real inventory carries", len(out), MinEntries)
	}
	return out, nil
}

// Load reads and parses the plan under root (the repository root).
func Load(root string) ([]Entry, error) {
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(PlanPath)))
	if err != nil {
		return nil, err
	}
	return Parse(string(b))
}

// Finding is one inventory path that does not resolve against the tree.
type Finding struct {
	Path   string
	Line   int
	Reason string
}

// Check holds every inventory path to the tree under root. A path that
// does not exist is a stale instruction, and a path that escapes the
// repository is refused outright rather than resolved: the inventory is
// read as a deletion list, so an entry reaching outside the tree is the
// one mistake that must never be followed.
func Check(root string, entries []Entry) ([]Finding, error) {
	var out []Finding
	for _, e := range entries {
		for _, p := range e.Paths {
			clean := filepath.ToSlash(filepath.Clean(p))
			if strings.HasPrefix(clean, "/") || clean == ".." || strings.HasPrefix(clean, "../") {
				out = append(out, Finding{Path: p, Line: e.Line, Reason: "the path escapes the repository"})
				continue
			}
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(clean))); err != nil {
				if os.IsNotExist(err) {
					out = append(out, Finding{Path: p, Line: e.Line, Reason: "the path does not exist"})
					continue
				}
				return nil, err
			}
		}
	}
	return out, nil
}
