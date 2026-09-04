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

// conformance: III.P row 1 (clone-and-init adoption from tagged
// releases; everything executable hash-pinned) and build plan section
// 5's distribution precondition (plans/os-2e46aa2f.md D5) — the Seed
// release workflow is dispatch-only, holds exactly the three
// permissions a release needs, mints its tag in the seed/v namespace
// at HEAD, publishes sha256 checksums and attests build provenance;
// the scheduled-writer lint reports nothing for it, and the same file
// given a schedule is a finding.
func TestSeedReleaseWorkflowIsDispatchOnly(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "seed-release.yml"))
	if err != nil {
		t.Fatalf("the release workflow is committed: %v", err)
	}
	wf := string(b)
	// The trigger block is workflow_dispatch and nothing else.
	on := wf[strings.Index(wf, "\non:"):strings.Index(wf, "\npermissions:")]
	for _, forbidden := range []string{"schedule:", "push:", "pull_request", "tags:", "release:"} {
		if strings.Contains(on, forbidden) {
			t.Errorf("a release is the operator's act: the trigger block carries %q", forbidden)
		}
	}
	if !strings.Contains(on, "workflow_dispatch:") || !strings.Contains(on, "version:") {
		t.Fatal("the workflow is dispatched with a version input")
	}
	perms := wf[strings.Index(wf, "\npermissions:"):strings.Index(wf, "\nconcurrency:")]
	for _, want := range []string{"contents: write", "id-token: write", "attestations: write"} {
		if !strings.Contains(perms, want) {
			t.Errorf("the release needs %s", want)
		}
	}
	if n := strings.Count(perms, ": write") + strings.Count(perms, ": read"); n != 3 {
		t.Errorf("exactly three permissions, got %d in %q", n, perms)
	}
	for _, want := range []string{
		`tag="seed/v$VERSION"`,
		`git tag "$tag"`,
		`-X github.com/shaunlmason/open-seed/next/internal/version.Version=$VERSION`,
		"sha256sum",
		"gh release create",
		"actions/attest-build-provenance@",
		"subject-checksums: dist/checksums.txt",
	} {
		if !strings.Contains(wf, want) {
			t.Errorf("the workflow lacks %q", want)
		}
	}
	for _, target := range []string{"linux/amd64", "linux/arm64", "darwin/amd64", "darwin/arm64", "windows/amd64", "windows/arm64"} {
		if !strings.Contains(wf, target) {
			t.Errorf("the matrix lacks %s", target)
		}
	}
	findings, err := LintWorkflows(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		if filepath.Base(f.File) == "seed-release.yml" {
			t.Errorf("a dispatch-only writer is not a scheduled writer: %s", f.Detail)
		}
	}
	// Mutation: the same workflow on a schedule is a finding.
	dir := t.TempDir()
	wfDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mutated := strings.Replace(wf, "on:\n  workflow_dispatch:", "on:\n  schedule:\n    - cron: \"0 0 * * *\"\n  workflow_dispatch:", 1)
	if mutated == wf {
		t.Fatal("the mutation did not apply")
	}
	if err := os.WriteFile(filepath.Join(wfDir, "seed-release.yml"), []byte(mutated), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err = LintWorkflows(dir)
	if err != nil || len(findings) != 1 {
		t.Fatalf("a scheduled release writer is one finding: %v %v", findings, err)
	}
}
