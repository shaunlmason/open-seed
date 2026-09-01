package project_test

// The refusal-rate report drills (plans/os-edf73d66.md D4c-d;
// charter III.I row 4): the section derives from the declared
// attempts journal alone — the plan review's worked example, one
// refusal beside a hundred admissions, must read 0.0099 and never
// the chain-denominator 0.5000 — builds are byte-identical for
// identical inputs, an input-free build states refusals: null, and
// the declared-inputs digest covers the journal, alone or beside
// the observation family.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shaunlmason/open-seed/next/internal/obs"
	"github.com/shaunlmason/open-seed/next/internal/project"
	"github.com/shaunlmason/open-seed/next/internal/refusals"
)

func attemptsFixture(t *testing.T) *refusals.Journal {
	t.Helper()
	dir := t.TempDir()
	refusals.Note(dir, refusals.Entry{
		TS: "2026-09-01T02:00:00Z", Position: "10", Actor: "aa11",
		Verb: "claim.taken", Subject: "c-1", Outcome: refusals.OutcomeRefused, Code: "fenced",
	})
	for i := 0; i < 100; i++ {
		refusals.Note(dir, refusals.Entry{
			TS: "2026-09-01T02:01:00Z", Position: "11", Actor: "aa11",
			Verb: "message.sent", Subject: "c-1", Outcome: refusals.OutcomeAdmitted,
		})
	}
	j, err := refusals.Load(filepath.Join(dir, refusals.File))
	if err != nil {
		t.Fatal(err)
	}
	return j
}

func TestReportRefusalsSection(t *testing.T) {
	root := pKey(t, 1)
	dir, resolve, _ := fixtureChain(t, root)
	journal := attemptsFixture(t)
	in := project.Inputs{Refusals: journal}

	out := lockedTempOut(t, "views")
	if _, err := project.RebuildWith(dir, out, project.Default(), resolve, in); err != nil {
		t.Fatal(err)
	}
	var rep project.ReportView
	readView(t, out, "report", project.ReportFile, &rep)
	if rep.Refusals == nil {
		t.Fatal("a declared journal must produce the refusals section")
	}
	if rep.Observation != nil {
		t.Fatalf("no observation inputs were declared: %+v", rep.Observation)
	}
	s := rep.Refusals
	if s.Refused != 1 || s.Admitted != 100 || s.Inputs.Entries != 101 {
		t.Fatalf("counts come from the journal's outcomes: %+v", s)
	}
	// The plan review's worked example: the rate is one population,
	// never refusals over a chain span (which would read 0.5000).
	if s.Rate != "0.0099" {
		t.Fatalf("1 refusal beside 100 admissions reads 0.0099, got %q", s.Rate)
	}
	if s.Span == nil || s.Span.From != 10 || s.Span.To != 11 {
		t.Fatalf("the span is the journal's stamped positions: %+v", s.Span)
	}
	if s.ByCode["fenced"] != 1 || s.ByVerb["claim.taken"] != 1 || len(s.ByCode) != 1 || len(s.ByVerb) != 1 {
		t.Fatalf("breakdowns count refusals only: %+v %+v", s.ByCode, s.ByVerb)
	}
	jd, err := journal.Digest()
	if err != nil {
		t.Fatal(err)
	}
	if s.Inputs.Digest != jd {
		t.Fatalf("the section echoes the journal's own digest: %s vs %s", s.Inputs.Digest, jd)
	}

	// The stamp and build id carry the declared-inputs digest, the
	// refusals family declaring alone.
	full, err := in.Digest()
	if err != nil {
		t.Fatal(err)
	}
	var stamp project.Stamp
	readView(t, out, "report", project.StampFile, &stamp)
	if stamp.Inputs != full {
		t.Fatalf("the stamp carries the inputs digest: %+v", stamp)
	}
	cur, err := os.ReadFile(filepath.Join(out, "report", "CURRENT"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(strings.TrimSpace(string(cur)), "-i"+full[:12]) {
		t.Fatalf("the input-bearing id ends in the digest segment: %s", cur)
	}

	// Byte-identical for identical inputs.
	out2 := lockedTempOut(t, "views")
	if _, err := project.RebuildWith(dir, out2, project.Default(), resolve, in); err != nil {
		t.Fatal(err)
	}
	one := readRaw(t, out, "report", project.ReportFile)
	two := readRaw(t, out2, "report", project.ReportFile)
	if !bytes.Equal(one, two) {
		t.Fatal("identical inputs must build identical report bytes")
	}

	// Input-free: the section is null, stated not fabricated, and
	// the version bump alone republishes existing prefixes.
	out3 := lockedTempOut(t, "views")
	if _, err := project.Rebuild(dir, out3, project.Default(), resolve); err != nil {
		t.Fatal(err)
	}
	readView(t, out3, "report", project.ReportFile, &rep)
	if rep.Refusals != nil {
		t.Fatalf("an input-free report must state refusals: null, got %+v", rep.Refusals)
	}
	if v := project.Report().Version; v != "10" {
		t.Fatalf("the refusals section is report version 10, got %s", v)
	}
}

// conformance: the declared-inputs digest spans EVERY declared
// input, families contributing only when declared, so an obs-only
// digest is unchanged and a journal rekeys any build it joins.
func TestInputsDigestCoversRefusals(t *testing.T) {
	journal := attemptsFixture(t)
	other := &refusals.Journal{Entries: journal.Entries[:1]}
	obsIn := project.Inputs{Obs: &obs.Snapshot{}, AsOf: time.Date(2026, 9, 1, 1, 0, 0, 0, time.UTC), Thresholds: obs.DefaultThresholds()}
	both := obsIn
	both.Refusals = journal
	dObs, err := obsIn.Digest()
	if err != nil {
		t.Fatal(err)
	}
	dBoth, err := both.Digest()
	if err != nil {
		t.Fatal(err)
	}
	dJournal, err := project.Inputs{Refusals: journal}.Digest()
	if err != nil {
		t.Fatal(err)
	}
	dOther, err := project.Inputs{Refusals: other}.Digest()
	if err != nil {
		t.Fatal(err)
	}
	if dObs == dBoth || dJournal == dOther || dJournal == dBoth || dObs == dJournal {
		t.Fatalf("every declared combination digests distinctly: obs %s both %s journal %s other %s", dObs, dBoth, dJournal, dOther)
	}
}

func readRaw(t *testing.T, out, name, file string) []byte {
	t.Helper()
	cur, err := os.ReadFile(filepath.Join(out, name, "CURRENT"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(out, name, "builds", strings.TrimSpace(string(cur)), file))
	if err != nil {
		t.Fatal(err)
	}
	return b
}
