package conformance_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/conformance"
)

const root = "../../.."

func charter(t *testing.T) []conformance.CharterPillar {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, conformance.CharterPath))
	if err != nil {
		t.Fatal(err)
	}
	c, err := conformance.ParseCharter(b)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func table(t *testing.T) conformance.Table {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(conformance.TablePath)))
	if err != nil {
		t.Fatal(err)
	}
	tb, err := conformance.Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	return tb
}

// conformance: plans/os-83bc3d84.md D2, AC1 — the charter parses into
// its eighteen pillars and the committed table is the charter row for
// row: the same ids, titles, texts and postures.
func TestTableIsTheCharterRowForRow(t *testing.T) {
	c := charter(t)
	if len(c) != 18 || c[0].ID != "A" || c[17].ID != "R" {
		t.Fatalf("Part III is eighteen lettered pillars A through R: %d", len(c))
	}
	rows := 0
	for _, p := range c {
		rows += len(p.Rows)
		for _, r := range p.Rows {
			if strings.Contains(r.Text, "  ") || strings.HasPrefix(r.Text, "- [ ]") {
				t.Fatalf("%s.%d: the text is whitespace-normalized: %q", p.ID, r.Row, r.Text)
			}
		}
	}
	if rows == 0 {
		t.Fatal("no rows parsed")
	}
	if err := conformance.Validate(table(t), c); err != nil {
		t.Fatalf("the committed table is not the charter: %v", err)
	}
	if _, err := conformance.Load(root); err != nil {
		t.Fatalf("Load: %v", err)
	}
	// The enforced-only marker is read off the text: opening the row
	// it marks the whole row, inside it one clause (mixed).
	whole, mixed := 0, 0
	for _, p := range c {
		for _, r := range p.Rows {
			switch {
			case strings.HasPrefix(r.Text, conformance.Marker):
				if r.Posture != conformance.PostureEnforcedOnly {
					t.Fatalf("%s.%d opens with the marker and is not enforced-only", p.ID, r.Row)
				}
				whole++
			case strings.Contains(r.Text, conformance.Marker):
				if r.Posture != conformance.PostureMixed {
					t.Fatalf("%s.%d carries the marker inside a clause and is not mixed", p.ID, r.Row)
				}
				mixed++
			case r.Posture != conformance.PostureAny:
				t.Fatalf("%s.%d carries no marker and is not any", p.ID, r.Row)
			}
		}
	}
	if whole == 0 || mixed == 0 {
		t.Fatalf("the charter marks whole rows (%d) and clauses (%d) enforced-only", whole, mixed)
	}
}

// conformance: D2 — a row added to, removed from or reworded on either
// side fails validation by name.
func TestTableDriftFromTheCharterIsRefused(t *testing.T) {
	c := charter(t)
	tb := table(t)
	dropped := tb
	dropped.Pillars = append([]conformance.Pillar(nil), tb.Pillars...)
	p := dropped.Pillars[1]
	p.Rows = p.Rows[:len(p.Rows)-1]
	dropped.Pillars[1] = p
	if err := conformance.Validate(dropped, c); err == nil || !strings.Contains(err.Error(), "pillar B") {
		t.Fatalf("a charter row missing from the table is refused by pillar: %v", err)
	}
	reworded := table(t)
	reworded.Pillars[0].Rows[0].Text += " and more"
	if err := conformance.Validate(reworded, c); err == nil || !strings.Contains(err.Error(), "A.1") {
		t.Fatalf("a reworded row is refused by id: %v", err)
	}
	grown := c
	grown = append([]conformance.CharterPillar(nil), c...)
	grown[2].Rows = append(grown[2].Rows, conformance.CharterRow{Row: len(grown[2].Rows) + 1, Text: "a new criterion", Posture: conformance.PostureAny})
	if err := conformance.Validate(table(t), grown); err == nil || !strings.Contains(err.Error(), "pillar C") {
		t.Fatalf("a charter row the table lacks is refused: %v", err)
	}
	posture := table(t)
	posture.Pillars[1].Rows[0].Posture = conformance.PostureAny
	if err := conformance.Validate(posture, c); err == nil || !strings.Contains(err.Error(), "posture") {
		t.Fatalf("a posture that is not the charter's is refused: %v", err)
	}
}

// conformance: D1, AC2 — the vocabulary holds: a status outside it, a
// met row without evidence, a partial or routed row without a note.
func TestVocabularyHolds(t *testing.T) {
	c := charter(t)
	for _, tc := range []struct {
		name string
		edit func(r *conformance.Row)
		want string
	}{
		{"outside", func(r *conformance.Row) { r.Status = "done" }, "outside the vocabulary"},
		{"met without evidence", func(r *conformance.Row) { r.Status = conformance.StatusMet; r.Evidence = "" }, "names its evidence"},
		{"partial without note", func(r *conformance.Row) { r.Status = conformance.StatusPartial; r.Note = "" }, "names where the rest lives"},
		{"routed without note", func(r *conformance.Row) { r.Status = conformance.StatusRouted; r.Note = "" }, "names where the rest lives"},
	} {
		tb := table(t)
		tc.edit(&tb.Pillars[0].Rows[0])
		if err := conformance.Validate(tb, c); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: %v", tc.name, err)
		}
	}
	tb := table(t)
	tb.Pillars[0].Rows[0].Status = conformance.StatusOpen
	tb.Pillars[0].Rows[0].Evidence = ""
	if err := conformance.Validate(tb, c); err != nil {
		t.Fatalf("an open row needs nothing else: %v", err)
	}
}

// conformance: the table decodes strictly.
func TestParseIsStrict(t *testing.T) {
	if _, err := conformance.Parse([]byte(`{"pillars": [], "extra": 1}`)); err == nil {
		t.Fatal("an unknown field is refused")
	}
	if _, err := conformance.Parse([]byte(`{"pillars": []} {}`)); err == nil {
		t.Fatal("trailing bytes are refused")
	}
	if _, err := conformance.ParseCharter([]byte("# nothing here\n")); err == nil {
		t.Fatal("a charter without Part III is refused")
	}
}

// conformance: D4, AC4 — complete means every row judged at the
// enforced posture is met: open, partial and routed rows are all
// outstanding, named by pillar and row with their status; the
// cooperative posture sets the enforced-only rows aside, judges the
// mixed rows while naming them, and is never complete.
func TestAssessJudgesAtThePosture(t *testing.T) {
	tb := conformance.Table{Pillars: []conformance.Pillar{{ID: "B", Title: "The Admission Boundary", Rows: []conformance.Row{
		{Row: 1, Text: "x", Posture: conformance.PostureEnforcedOnly, Status: conformance.StatusMet, Evidence: "#1"},
		{Row: 2, Text: "y", Posture: conformance.PostureAny, Status: conformance.StatusOpen, Phase: "13", Note: "item 6"},
		{Row: 3, Text: "z", Posture: conformance.PostureAny, Status: conformance.StatusRouted, Note: "backlog"},
		{Row: 4, Text: "w", Posture: conformance.PostureMixed, Status: conformance.StatusPartial, Note: "half"},
	}}}}
	rep := conformance.Assess(tb, true)
	ids := func(rows []conformance.OutstandingRow) string {
		var out []string
		for _, r := range rows {
			out = append(out, r.ID+":"+r.Status)
		}
		return strings.Join(out, " ")
	}
	if rep.Complete || ids(rep.Outstanding) != "B.2:open B.3:routed B.4:partial" || rep.Outstanding[0].Phase != "13" || rep.Counts.Met != 1 || rep.Counts.Routed != 1 || rep.Counts.Open != 1 || rep.Counts.Partial != 1 || len(rep.NotApplicable) != 0 || len(rep.MixedHere) != 0 {
		t.Fatalf("enforced: every unmet row is outstanding, by status: %+v", rep)
	}
	for i := 1; i < 4; i++ {
		tb.Pillars[0].Rows[i].Status = conformance.StatusMet
		tb.Pillars[0].Rows[i].Evidence = "#2"
		tb.Pillars[0].Rows[i].Note = ""
	}
	rep = conformance.Assess(tb, true)
	if !rep.Complete || len(rep.Outstanding) != 0 || !strings.Contains(rep.Because, "every row is met") {
		t.Fatalf("enforced with every row met: complete: %+v", rep)
	}
	rep = conformance.Assess(tb, false)
	if rep.Complete || len(rep.NotApplicable) != 1 || rep.NotApplicable[0] != "B.1" || len(rep.MixedHere) != 1 || rep.MixedHere[0] != "B.4" || rep.Counts.Met != 3 || !strings.Contains(rep.Because, "cooperative") {
		t.Fatalf("cooperative: the enforced-only row is set aside, the mixed row judged and named, and Part III is not complete here: %+v", rep)
	}
	b, _ := json.Marshal(rep)
	if !strings.Contains(string(b), `"outstanding_rows":[]`) {
		t.Fatalf("the outstanding rows render as an empty list, never null: %s", b)
	}
}
