package main

// The docs surface through the CLI (plans/os-16e55c11.md D1): check
// passes clean against the committed tree, a subverb is required, and a
// generate into a fresh root then check comes back clean. The renderer
// drills live in internal/docs; this pins the CLI wiring and the exit
// codes it returns.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/envelope"
)

func TestDocsCheckCleanOnCommitted(t *testing.T) {
	// The committed generated tree matches what the generator renders.
	e, code := runEnv(t, "docs", "check", "--root", "../../..")
	if code != envelope.ExitOK || !e.OK {
		t.Fatalf("docs check must pass on the committed tree, got exit %d code %v", code, e.Error)
	}
}

func TestDocsRequiresSubverb(t *testing.T) {
	e, code := runEnv(t, "docs")
	if code != envelope.ExitUsage || e.Error == nil {
		t.Fatalf("docs with no subverb must be a usage error, got %d", code)
	}
	if _, code := runEnv(t, "docs", "bogus"); code != envelope.ExitUsage {
		t.Fatalf("an unknown docs subverb must be a usage error, got %d", code)
	}
}

func TestDocsGenerateThenCheck(t *testing.T) {
	// Mirror the sources into a temp root, generate, and check clean.
	real, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "next"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{"lanes", "internal"} {
		if err := os.Symlink(filepath.Join(real, "next", d), filepath.Join(tmp, "next", d)); err != nil {
			t.Fatal(err)
		}
	}
	if _, code := runEnv(t, "docs", "generate", "--root", tmp); code != envelope.ExitOK {
		t.Fatalf("docs generate must succeed, got %d", code)
	}
	e, code := runEnv(t, "docs", "check", "--root", tmp)
	if code != envelope.ExitOK || !e.OK {
		t.Fatalf("docs check must pass after generate, got exit %d code %v", code, e.Error)
	}
	// A hand edit is caught as docs_drift (exit 28).
	f := filepath.Join(tmp, "next", "docs", "generated", "lifecycle.md")
	b, _ := os.ReadFile(f)
	os.WriteFile(f, append(b, []byte("x\n")...), 0o644)
	e, code = runEnv(t, "docs", "check", "--root", tmp)
	if code != envelope.ExitDrift || e.Error == nil || e.Error.Code != envelope.CodeDocsDrift {
		t.Fatalf("a hand edit must fail docs_drift (exit 28), got exit %d code %v", code, e.Error)
	}
}
