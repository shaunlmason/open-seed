package refusals

// The journal drills (plans/os-edf73d66.md D4a): Note appends
// well-formed lines and swallows every failure, Load refuses
// malformed declared input line by line, and the digest is keyed by
// content.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func entry(outcome, code string) Entry {
	return Entry{
		TS: "2026-09-01T02:30:00Z", Position: "7", Actor: "aa11",
		Verb: "message.sent", Subject: "c-1", Outcome: outcome, Code: code,
	}
}

func TestNoteAppendsAndLoadReads(t *testing.T) {
	dir := t.TempDir()
	Note(dir, entry(OutcomeAdmitted, ""))
	Note(dir, entry(OutcomeRefused, "chain_invalid"))
	j, err := Load(filepath.Join(dir, File))
	if err != nil {
		t.Fatal(err)
	}
	if len(j.Entries) != 2 || j.Entries[0].Outcome != OutcomeAdmitted || j.Entries[1].Code != "chain_invalid" {
		t.Fatalf("the journal replays what was noted: %+v", j.Entries)
	}
}

// conformance: journaling must never fail or slow the verb it rides:
// an unwritable target journals nothing and raises nothing.
func TestNoteSwallowsFailures(t *testing.T) {
	Note("", entry(OutcomeAdmitted, ""))
	file := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// The "directory" is a regular file, so the open fails; Note
	// returns anyway.
	Note(filepath.Join(file, "sub"), entry(OutcomeAdmitted, ""))
}

// conformance: inputs are declared, so garbage is the declarer's
// error, never silently skipped telemetry; refusals name the line.
func TestLoadRefusesMalformedLines(t *testing.T) {
	for name, line := range map[string]string{
		"not json":         `{`,
		"unknown field":    `{"ts": "t", "position": "1", "actor": "a", "verb": "v", "subject": "s", "outcome": "admitted", "extra": 1}`,
		"trailing data":    `{"ts": "t", "position": "1", "actor": "a", "verb": "v", "subject": "s", "outcome": "admitted"} {}`,
		"unknown outcome":  `{"ts": "t", "position": "1", "actor": "a", "verb": "v", "subject": "s", "outcome": "maybe"}`,
		"refused no code":  `{"ts": "t", "position": "1", "actor": "a", "verb": "v", "subject": "s", "outcome": "refused"}`,
		"admitted w/ code": `{"ts": "t", "position": "1", "actor": "a", "verb": "v", "subject": "s", "outcome": "admitted", "code": "fenced"}`,
		"bad position":     `{"ts": "t", "position": "tip", "actor": "a", "verb": "v", "subject": "s", "outcome": "admitted"}`,
		"missing verb":     `{"ts": "t", "position": "1", "actor": "a", "verb": "", "subject": "s", "outcome": "admitted"}`,
	} {
		path := filepath.Join(t.TempDir(), File)
		if err := os.WriteFile(path, []byte(line+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "line 1") {
			t.Fatalf("%s must refuse naming the line, got %v", name, err)
		}
	}
}

func TestDigestKeyedByContent(t *testing.T) {
	a := &Journal{Entries: []Entry{entry(OutcomeAdmitted, "")}}
	b := &Journal{Entries: []Entry{entry(OutcomeRefused, "fenced")}}
	da, err := a.Digest()
	if err != nil {
		t.Fatal(err)
	}
	db, err := b.Digest()
	if err != nil {
		t.Fatal(err)
	}
	if da == db {
		t.Fatal("different journals must digest differently")
	}
	da2, err := (&Journal{Entries: []Entry{entry(OutcomeAdmitted, "")}}).Digest()
	if err != nil {
		t.Fatal(err)
	}
	if da != da2 {
		t.Fatal("equal journals must digest equally")
	}
}
