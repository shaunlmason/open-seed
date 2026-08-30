package ledger

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/event"
)

func TestFailureErrorFormat(t *testing.T) {
	f := &Failure{Position: 3, Reason: ReasonBadPrev, Detail: "x"}
	if got := f.Error(); got != "position 3: bad_prev: x" {
		t.Fatalf("Failure.Error() = %q", got)
	}
}

func TestOpenRefusesFilePath(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(file); err == nil {
		t.Fatal("Open on a file path must refuse")
	}
}

func TestBlankLinesAreSkipped(t *testing.T) {
	dir := copyFixture(t)
	seg := segmentPath(t, dir, "2026-09-03.jsonl")
	f, err := os.OpenFile(seg, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("\n\n"); err != nil {
		t.Fatal(err)
	}
	f.Close()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	rep, err := s.VerifyFromGenesis(fixtureResolver(t, fixtureKey(t, 1)))
	if err != nil || rep.Count != 4 {
		t.Fatalf("blank lines must not count as records: %+v %v", rep, err)
	}
}

func TestTipReportsUnparseableStream(t *testing.T) {
	dir := copyFixture(t)
	rewriteFile(t, segmentPath(t, dir, "2026-09-02.jsonl"), func(string) string { return "garbage\n" })
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Tip(); err == nil {
		t.Fatal("Tip over an unparseable stream must error")
	}
	priv := fixtureKey(t, 1)
	rec := signedRecord(t, priv, 9, "2026-09-03T12:00:00Z", event.EmptyHash)
	if _, err := s.Append(rec, fixtureResolver(t, priv)); err == nil {
		t.Fatal("Append over an unparseable stream must refuse")
	}
}

func TestMissingSegmentsDirSurfacesErrors(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(dir, segmentsDir)); err != nil {
		t.Fatal(err)
	}
	priv := fixtureKey(t, 1)
	resolve := fixtureResolver(t, priv)
	rec := signedRecord(t, priv, 0, "2026-09-01T10:00:00Z", event.EmptyHash)
	if _, err := s.Append(rec, resolve); err == nil {
		t.Fatal("Append without the segments dir must error")
	}
	if _, err := s.VerifyFromGenesis(resolve); err == nil {
		t.Fatal("verify without the segments dir must error")
	}
	var fail *Failure
	if _, err := s.VerifyFromGenesis(resolve); errors.As(err, &fail) {
		t.Fatalf("infrastructure errors are not chain Failures, got %v", err)
	}
}

func TestWriteHeadFailureSurfaces(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := s.writeHead(Head{Tip: "abc", Count: 1, Segment: "x"}); err == nil {
		t.Fatal("writeHead into a removed root must error")
	}
}

func TestHeadAsDirectoryRefuses(t *testing.T) {
	dir := copyFixture(t)
	if err := os.Remove(filepath.Join(dir, headFile)); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, headFile), 0o755); err != nil {
		t.Fatal(err)
	}
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.ReadHead(); err == nil {
		t.Fatal("HEAD as a directory must error on read")
	}
	_, err = s.VerifyFromGenesis(fixtureResolver(t, fixtureKey(t, 1)))
	var fail *Failure
	if !errors.As(err, &fail) || fail.Reason != ReasonHeadWrong {
		t.Fatalf("unreadable HEAD must report %s, got %v", ReasonHeadWrong, err)
	}
	if !strings.Contains(fail.Detail, "HEAD") && fail.Detail == "" {
		t.Fatalf("failure detail should explain, got %q", fail.Detail)
	}
}

func TestUnopenableSegmentSurfacesError(t *testing.T) {
	dir := copyFixture(t)
	if err := os.Symlink("does-not-exist", segmentPath(t, dir, "2026-09-09.jsonl")); err != nil {
		t.Fatal(err)
	}
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Tip(); err == nil {
		t.Fatal("an unopenable segment must surface an error")
	}
	priv := fixtureKey(t, 1)
	rec := signedRecord(t, priv, 9, "2026-09-10T00:00:00Z", event.EmptyHash)
	if _, err := s.Append(rec, fixtureResolver(t, priv)); err == nil {
		t.Fatal("Append must refuse when the stream cannot be reconciled")
	}
}
