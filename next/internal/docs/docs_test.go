package docs

// The governed-docs drills (plans/os-16e55c11.md D1, AC1): the four
// documents are generated from the tables, the committed output matches
// what the generator renders now, and `Check` catches drift in every
// direction — a hand edit, a document the generator no longer produces,
// and a stale file left behind.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot is the repository root relative to this package's test cwd
// (next/internal/docs): three directories up.
const repoRoot = "../../.."

func TestGeneratedDocsAreCommitted(t *testing.T) {
	drift, err := Check(repoRoot)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(drift) != 0 {
		t.Fatalf("committed docs are stale — run `seed docs generate`: %v", drift)
	}
}

func TestGenerateIsDeterministic(t *testing.T) {
	a, err := Generate(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Generate(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != len(b) {
		t.Fatalf("generation is not deterministic: %d vs %d docs", len(a), len(b))
	}
	for k, va := range a {
		if b[k] != va {
			t.Fatalf("generation is not deterministic for %s", k)
		}
	}
}

func TestGeneratedContentIsFromTheTables(t *testing.T) {
	docs, err := Generate(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	life := docs[filepath.Join(GenDir, "lifecycle.md")]
	// A verb only the table carries, with its exclusivity, is present.
	if !strings.Contains(life, "`claim.taken`") || !strings.Contains(life, "yes") {
		t.Error("lifecycle.md must render claim.taken and its exclusivity from the table")
	}
	caps := docs[filepath.Join(GenDir, "capabilities.md")]
	if !strings.Contains(caps, "`intent.filed`") {
		t.Error("capabilities.md must render the catalog verbs")
	}
	exits := docs[filepath.Join(GenDir, "exit-codes.md")]
	if !strings.Contains(exits, "`ExitDrift`") || !strings.Contains(exits, "`docs_drift`") {
		t.Error("exit-codes.md must render the exit constants and the refinement codes")
	}
}

// mirrorRoot builds a temp repository root whose sources (next/lanes,
// next/internal) are symlinks to the real tree, so the generator renders
// identical content, but whose next/docs/generated is a mutable copy the
// drift drills can perturb.
func mirrorRoot(t *testing.T) string {
	t.Helper()
	real, err := filepath.Abs(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	nextTmp := filepath.Join(tmp, "next")
	if err := os.MkdirAll(nextTmp, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{"lanes", "internal"} {
		if err := os.Symlink(filepath.Join(real, "next", d), filepath.Join(nextTmp, d)); err != nil {
			t.Fatal(err)
		}
	}
	// Copy the committed generated tree into the temp root.
	src := filepath.Join(real, GenDir)
	dst := filepath.Join(tmp, GenDir)
	if err := filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	}); err != nil {
		t.Fatal(err)
	}
	// Sanity: the mirror is clean before perturbation.
	drift, err := Check(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if len(drift) != 0 {
		t.Fatalf("mirror is not clean before perturbation: %v", drift)
	}
	return tmp
}

func TestCheckCatchesHandEdit(t *testing.T) {
	tmp := mirrorRoot(t)
	f := filepath.Join(tmp, GenDir, "lifecycle.md")
	b, _ := os.ReadFile(f)
	if err := os.WriteFile(f, append(b, []byte("hand edit\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	drift, err := Check(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if len(drift) != 1 || !strings.HasSuffix(drift[0], "lifecycle.md") {
		t.Fatalf("a hand edit must be caught as drift, got %v", drift)
	}
}

func TestCheckCatchesMissingAndStale(t *testing.T) {
	// A committed document the generator still produces, deleted: drift.
	tmp := mirrorRoot(t)
	if err := os.Remove(filepath.Join(tmp, GenDir, "capabilities.md")); err != nil {
		t.Fatal(err)
	}
	drift, err := Check(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if len(drift) != 1 || !strings.HasSuffix(drift[0], "capabilities.md") {
		t.Fatalf("a missing generated doc must be drift, got %v", drift)
	}

	// A stale file the generator no longer produces: drift.
	tmp2 := mirrorRoot(t)
	stale := filepath.Join(tmp2, GenDir, "lanes", "ghost.md")
	if err := os.WriteFile(stale, []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	drift, err = Check(tmp2)
	if err != nil {
		t.Fatal(err)
	}
	if len(drift) != 1 || !strings.HasSuffix(drift[0], filepath.Join("lanes", "ghost.md")) {
		t.Fatalf("a stale generated file must be drift, got %v", drift)
	}
}

func TestWriteThenCheckClean(t *testing.T) {
	tmp := mirrorRoot(t)
	// Perturb, then Write must restore a clean tree (and drop stale).
	os.WriteFile(filepath.Join(tmp, GenDir, "lanes", "ghost.md"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(tmp, GenDir, "lifecycle.md"), []byte("x"), 0o644)
	if _, err := Write(tmp); err != nil {
		t.Fatal(err)
	}
	drift, err := Check(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if len(drift) != 0 {
		t.Fatalf("Write must produce a clean tree, got drift %v", drift)
	}
	if _, err := os.Stat(filepath.Join(tmp, GenDir, "lanes", "ghost.md")); !os.IsNotExist(err) {
		t.Fatal("Write must remove a stale generated file")
	}
}
