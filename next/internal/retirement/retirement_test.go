package retirement

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot is the repository root from this package's directory.
func repoRoot() string { return filepath.Join("..", "..", "..") }

// conformance: docs/v1-retirement.md stage 2 (the inventory is pinned)
// — every path stage 4 is instructed to delete resolves against the
// tree today, so a rename or a move is found here rather than at the
// deletion.
func TestInventoryResolvesAgainstTheTree(t *testing.T) {
	entries, err := Load(repoRoot())
	if err != nil {
		t.Fatal(err)
	}
	findings, err := Check(repoRoot(), entries)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		t.Errorf("line %d: %s: %s", f.Line, f.Path, f.Reason)
	}
	var paths int
	for _, e := range entries {
		paths += len(e.Paths)
	}
	if paths < len(entries) {
		t.Fatalf("every row names at least one path: %d paths over %d rows", paths, len(entries))
	}
}

// A planted stale path must be found, and a path reaching outside the
// tree must be refused by name rather than resolved: the inventory is
// read as a deletion list.
func TestCheckFindsStaleAndEscapingPaths(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "real"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	entries := []Entry{{Paths: []string{"real", "gone", "../outside", "/etc/passwd"}, What: "x", Line: 7}}
	findings, err := Check(dir, entries)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 3 {
		t.Fatalf("findings: %+v", findings)
	}
	if findings[0].Path != "gone" || !strings.Contains(findings[0].Reason, "does not exist") {
		t.Errorf("the missing path is reported: %+v", findings[0])
	}
	for _, f := range findings[1:] {
		if !strings.Contains(f.Reason, "escapes the repository") {
			t.Errorf("an escaping path is refused: %+v", f)
		}
	}
	if findings[0].Line != 7 {
		t.Errorf("a finding carries its row's line: %+v", findings[0])
	}
}

// A stat error that is not "not exist" surfaces rather than being
// counted as a missing path: Makefile/x is ENOTDIR, not ENOENT.
func TestCheckSurfacesUnexpectedStatErrors(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "file"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Check(dir, []Entry{{Paths: []string{"file/below"}, What: "x"}}); err == nil {
		t.Fatal("a path below a regular file must surface the read error")
	}
}

func TestLoadFailsWithoutThePlan(t *testing.T) {
	if _, err := Load(t.TempDir()); err == nil {
		t.Fatal("Load must fail where no plan exists")
	}
}

func TestParseIsStrict(t *testing.T) {
	rows := func(n int) string {
		var b strings.Builder
		b.WriteString(DeletionHeading + "\n\n| path | what it is |\n|---|---|\n")
		for i := 0; i < n; i++ {
			b.WriteString("| `p" + string(rune('a'+i)) + "` | a thing |\n")
		}
		return b.String()
	}
	for _, tc := range []struct {
		name string
		doc  string
		want string
	}{
		{"no section", "# nothing here\n", "has no"},
		{"a row naming no path", rows(MinEntries) + "| bare cell | a thing |\n", "names no path"},
		{"a row saying nothing", rows(MinEntries) + "| `x` |  |\n", "says nothing"},
		{"too few rows", rows(MinEntries - 1), "fewer than"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Parse(tc.doc); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q, got %v", tc.want, err)
			}
		})
	}

	// The happy path: the header and separator are skipped, several
	// paths in one cell are all taken, and the next bold heading ends
	// the section so a later table is not read as inventory.
	doc := DeletionHeading + "\n\n| path | what it is |\n|---|---|\n" +
		"| `a`, `b` | two things |\n" + rows(MinEntries)[strings.Index(rows(MinEntries), "|---|---|\n")+len("|---|---|\n"):] +
		"\n**Kept, permanently.**\n\n| path | what |\n|---|---|\n| `never-parsed` | no |\n"
	got, err := Parse(doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != MinEntries+1 {
		t.Fatalf("rows: %d", len(got))
	}
	if len(got[0].Paths) != 2 || got[0].Paths[0] != "a" || got[0].Paths[1] != "b" || got[0].What != "two things" {
		t.Fatalf("a cell's paths are all taken: %+v", got[0])
	}
	for _, e := range got {
		for _, p := range e.Paths {
			if p == "never-parsed" {
				t.Fatal("the section ends at the next bold heading")
			}
		}
	}
}
