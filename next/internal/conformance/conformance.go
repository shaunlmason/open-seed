// Package conformance is the Part III table (plans/os-83bc3d84.md;
// build plan Phase 13's exit line): the charter's conformance rows,
// each with the status the exit records gave it, held row for row to
// SEED-NEXT.md by a parser of the charter itself. It is a document
// and a doctor section, never an admission rule: nothing here refuses
// a record, and the table's authority is the exit records it
// transcribes.
package conformance

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// The status vocabulary (D1): met with evidence, partial or routed
// with a note naming where the rest lives, or open.
const (
	StatusMet     = "met"
	StatusPartial = "partial"
	StatusRouted  = "routed"
	StatusOpen    = "open"
)

// Statuses is the vocabulary in the order the report counts it.
var Statuses = []string{StatusMet, StatusPartial, StatusRouted, StatusOpen}

// The postures a row is judged at: any deployment, or the enforced
// postures alone (the charter's (*enforced-only*) marker).
const (
	PostureAny          = "any"
	PostureEnforcedOnly = "enforced-only"
)

// TablePath is the table's location under the repository root, and
// CharterPath the charter's.
const (
	TablePath   = "next/spec/conformance.json"
	CharterPath = "SEED-NEXT.md"
)

// Row is one charter row and its judgment.
type Row struct {
	Row      int    `json:"row"`
	Text     string `json:"text"`
	Posture  string `json:"posture"`
	Status   string `json:"status"`
	Phase    string `json:"phase"`
	Evidence string `json:"evidence"`
	Note     string `json:"note"`
}

// Pillar is one lettered section of Part III.
type Pillar struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Rows  []Row  `json:"rows"`
}

// Table is the whole of Part III as judged.
type Table struct {
	Pillars []Pillar `json:"pillars"`
}

// CharterRow is one row as the charter states it.
type CharterRow struct {
	Row     int
	Text    string
	Posture string
}

// CharterPillar is one lettered section as the charter states it.
type CharterPillar struct {
	ID    string
	Title string
	Rows  []CharterRow
}

var (
	pillarHeading = regexp.MustCompile(`^### ([A-R])\. (.+)$`)
	rowStart      = regexp.MustCompile(`^- \[ \] (.*)$`)
	spaces        = regexp.MustCompile(`\s+`)
)

// ParseCharter reads Part III out of the charter: the lettered
// headings and every checkbox row with its continuation lines
// (indented by six spaces), whitespace-normalized, the posture read
// off the (*enforced-only*) marker. The section runs from the "Part
// III" heading to the next second-level heading.
func ParseCharter(b []byte) ([]CharterPillar, error) {
	lines := strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
	start := -1
	for i, l := range lines {
		if strings.HasPrefix(l, "## Part III") {
			start = i
			break
		}
	}
	if start < 0 {
		return nil, errors.New("the charter has no \"## Part III\" heading")
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			end = i
			break
		}
	}
	var pillars []CharterPillar
	var cur *CharterPillar
	var row *CharterRow
	for _, l := range lines[start:end] {
		if m := pillarHeading.FindStringSubmatch(l); m != nil {
			pillars = append(pillars, CharterPillar{ID: m[1], Title: strings.TrimSpace(m[2])})
			cur = &pillars[len(pillars)-1]
			row = nil
			continue
		}
		if cur == nil {
			continue
		}
		if m := rowStart.FindStringSubmatch(l); m != nil {
			cur.Rows = append(cur.Rows, CharterRow{Row: len(cur.Rows) + 1, Text: strings.TrimSpace(m[1])})
			row = &cur.Rows[len(cur.Rows)-1]
			continue
		}
		if row != nil && strings.HasPrefix(l, "      ") && strings.TrimSpace(l) != "" {
			row.Text += " " + strings.TrimSpace(l)
			continue
		}
		if strings.TrimSpace(l) == "" {
			row = nil
		}
	}
	if len(pillars) == 0 {
		return nil, errors.New("the charter's Part III has no lettered pillars")
	}
	for i := range pillars {
		for j := range pillars[i].Rows {
			r := &pillars[i].Rows[j]
			r.Text = strings.TrimSpace(spaces.ReplaceAllString(r.Text, " "))
			r.Posture = PostureAny
			if strings.Contains(r.Text, "(*enforced-only*)") {
				r.Posture = PostureEnforcedOnly
			}
		}
	}
	return pillars, nil
}

// Parse decodes the table strictly.
func Parse(b []byte) (Table, error) {
	var t Table
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&t); err != nil {
		return Table{}, fmt.Errorf("the conformance table is the strict object {pillars: [{id, title, rows: [{row, text, posture, status, phase, evidence, note}]}]}: %v", err)
	}
	if dec.More() {
		return Table{}, errors.New("the conformance table is one object, and bytes follow it")
	}
	return t, nil
}

// Load reads and validates the table and the charter under root.
func Load(root string) (Table, error) {
	tb, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(TablePath)))
	if err != nil {
		return Table{}, err
	}
	t, err := Parse(tb)
	if err != nil {
		return Table{}, err
	}
	cb, err := os.ReadFile(filepath.Join(root, CharterPath))
	if err != nil {
		return Table{}, err
	}
	charter, err := ParseCharter(cb)
	if err != nil {
		return Table{}, err
	}
	if err := Validate(t, charter); err != nil {
		return Table{}, err
	}
	return t, nil
}

// Validate holds the table to the charter row for row (D2) and to the
// vocabulary (D1): the same pillars in the same order with the same
// titles, the same rows with the same text and posture, every status
// in the vocabulary, met rows carrying evidence, partial and routed
// rows carrying a note.
func Validate(t Table, charter []CharterPillar) error {
	if len(t.Pillars) != len(charter) {
		return fmt.Errorf("the table has %d pillars, the charter %d", len(t.Pillars), len(charter))
	}
	for i, p := range t.Pillars {
		c := charter[i]
		if p.ID != c.ID || p.Title != c.Title {
			return fmt.Errorf("pillar %d: the table says %s. %q, the charter %s. %q", i+1, p.ID, p.Title, c.ID, c.Title)
		}
		if len(p.Rows) != len(c.Rows) {
			return fmt.Errorf("pillar %s: the table has %d rows, the charter %d", p.ID, len(p.Rows), len(c.Rows))
		}
		for j, r := range p.Rows {
			cr := c.Rows[j]
			id := fmt.Sprintf("%s.%d", p.ID, j+1)
			if r.Row != cr.Row {
				return fmt.Errorf("%s: the row is numbered %d", id, r.Row)
			}
			if r.Text != cr.Text {
				return fmt.Errorf("%s: the table's text is not the charter's:\n table: %s\n charter: %s", id, r.Text, cr.Text)
			}
			if r.Posture != cr.Posture {
				return fmt.Errorf("%s: the posture is %q, the charter says %q", id, r.Posture, cr.Posture)
			}
			if err := validateStatus(r); err != nil {
				return fmt.Errorf("%s: %v", id, err)
			}
		}
	}
	return nil
}

func validateStatus(r Row) error {
	switch r.Status {
	case StatusMet:
		if strings.TrimSpace(r.Evidence) == "" {
			return errors.New("a met row names its evidence")
		}
	case StatusPartial, StatusRouted:
		if strings.TrimSpace(r.Note) == "" {
			return fmt.Errorf("a %s row's note names where the rest lives", r.Status)
		}
	case StatusOpen:
	default:
		return fmt.Errorf("status %q is outside the vocabulary %v", r.Status, Statuses)
	}
	return nil
}

// Counts is the table's rows by status.
type Counts struct {
	Met     int `json:"met"`
	Partial int `json:"partial"`
	Routed  int `json:"routed"`
	Open    int `json:"open"`
}

// Count tallies a set of rows.
func Count(rows []Row) Counts {
	var c Counts
	for _, r := range rows {
		switch r.Status {
		case StatusMet:
			c.Met++
		case StatusPartial:
			c.Partial++
		case StatusRouted:
			c.Routed++
		default:
			c.Open++
		}
	}
	return c
}

// OpenRow names one row still open: its id, phase and posture.
type OpenRow struct {
	ID      string `json:"id"`
	Phase   string `json:"phase,omitempty"`
	Posture string `json:"posture"`
	Note    string `json:"note,omitempty"`
}

// Report is what the doctor says about the table at a posture (D4):
// the counts over the rows judged there, the rows that remain open,
// the enforced-only rows a cooperative deployment cannot judge, and
// whether Part III is complete here: no open row, every enforced-only
// row judged.
type Report struct {
	Counts        Counts    `json:"counts"`
	OpenRows      []OpenRow `json:"open_rows"`
	NotApplicable []string  `json:"not_applicable_here,omitempty"`
	Complete      bool      `json:"complete"`
	Because       string    `json:"because"`
}

// Assess judges the table at a posture: enforced reads every row;
// cooperative sets the enforced-only rows aside as not applicable,
// which by itself keeps Part III from being complete there.
func Assess(t Table, enforced bool) Report {
	var rep Report
	var judged []Row
	for _, p := range t.Pillars {
		for _, r := range p.Rows {
			id := fmt.Sprintf("%s.%d", p.ID, r.Row)
			if !enforced && r.Posture == PostureEnforcedOnly {
				rep.NotApplicable = append(rep.NotApplicable, id)
				continue
			}
			judged = append(judged, r)
			if r.Status == StatusOpen {
				rep.OpenRows = append(rep.OpenRows, OpenRow{ID: id, Phase: r.Phase, Posture: r.Posture, Note: r.Note})
			}
		}
	}
	rep.Counts = Count(judged)
	sort.Strings(rep.NotApplicable)
	switch {
	case len(rep.NotApplicable) > 0:
		rep.Complete = false
		rep.Because = fmt.Sprintf("%d enforced-only rows cannot be judged at the cooperative posture; Part III is complete only at an enforced posture", len(rep.NotApplicable))
	case len(rep.OpenRows) > 0:
		rep.Complete = false
		rep.Because = fmt.Sprintf("%d rows are open", len(rep.OpenRows))
	default:
		rep.Complete = true
		rep.Because = "no row is open at the enforced posture; partial and routed rows carry their notes"
	}
	if rep.OpenRows == nil {
		rep.OpenRows = []OpenRow{}
	}
	return rep
}
