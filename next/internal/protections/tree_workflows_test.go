package protections

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// conformance: III.L row 5 — least-privilege CI identities. The tree's
// own scheduled workflows are held to the scheduled-writer lint
// (plans/os-a00d3f34.md D4): the only scheduled workflow allowed a
// write is v1's maintenance lane, whose contents: write is the state
// ref's anchor and the operator identity's by design (.seed/config.toml
// roster); every other scheduled workflow, the scale benchmark first,
// is read-only, and a copy of it given contents: write is a finding.
func TestTreeWorkflowsHaveNoScheduledWriters(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	findings, err := LintWorkflows(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		if filepath.Base(f.File) != "seed-maintenance.yml" {
			t.Errorf("a scheduled writer outside the v1 maintenance lane: %s: %s", f.File, f.Detail)
		}
	}
	scale, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "perf-scale.yml"))
	if err != nil {
		t.Fatalf("the scale workflow is committed: %v", err)
	}
	if !strings.Contains(string(scale), "schedule:") || !strings.Contains(string(scale), "contents: read") || strings.Contains(string(scale), "contents: write") {
		t.Fatal("perf-scale.yml is scheduled and read-only")
	}
	// Mutation: the same workflow given write permission is a finding.
	dir := t.TempDir()
	wf := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wf, 0o755); err != nil {
		t.Fatal(err)
	}
	mutated := strings.Replace(string(scale), "contents: read", "contents: write", 1)
	if err := os.WriteFile(filepath.Join(wf, "perf-scale.yml"), []byte(mutated), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LintWorkflows(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !strings.Contains(got[0].Detail, "scheduled workflow") {
		t.Fatalf("the writable copy must be one finding, got %v", got)
	}
}
