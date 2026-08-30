// Package transition is the lifecycle transition table as data
// (docs/next-build-plan.md Phase 5 item 1; plans/os-d69a6c91.md;
// SEED-NEXT.md Part II §6 and conformance III.F "the lifecycle
// vocabulary and transition rules are self-validating data enforced at
// admission; claim is a transition, not a state"). The contract is
// data, not branching: legality comes from the parsed table, and
// hand-written conditionals that re-derive what the table says are a
// design violation. next/spec/transitions.json is the normative
// reviewable copy; the embedded table.json is byte-identical, pinned
// by test, exactly the classify.json precedent.
package transition

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/shaunlmason/open-seed/next/internal/event"
)

//go:embed table.json
var tableJSON []byte

// TableJSON exposes the embedded bytes for the byte-identity drill
// against next/spec/transitions.json.
func TableJSON() []byte { return append([]byte(nil), tableJSON...) }

// stateDecl is one declared state in the table file.
type stateDecl struct {
	Name     string `json:"name"`
	Initial  bool   `json:"initial"`
	Terminal bool   `json:"terminal"`
}

// row is one transition row: from is nil for the single birth verb.
type row struct {
	Verb string    `json:"verb"`
	From *[]string `json:"from"`
	To   string    `json:"to"`
}

type tableFile struct {
	SchemaVersion string      `json:"schema_version"`
	States        []stateDecl `json:"states"`
	Transitions   []row       `json:"transitions"`
}

// Table is the parsed, self-validated transition table.
type Table struct {
	SchemaVersion string
	initial       string
	terminal      map[string]bool
	states        map[string]bool
	birth         string
	birthTo       string
	// legal maps verb -> (from-state -> to-state); the birth verb has
	// no entry here.
	legal map[string]map[string]string
	verbs []string
}

// InvalidTransitionError is the typed lifecycle refusal (exit 3
// invalid_transition, the v1-continuity allocation): the verb is not
// legal for the subject's current state. From is empty when the
// subject does not exist yet.
type InvalidTransitionError struct {
	Subject string
	From    string
	Verb    string
}

func (e *InvalidTransitionError) Error() string {
	if e.From == "" {
		return fmt.Sprintf("verb %s is illegal for subject %s: the subject does not exist (only the birth verb creates one)", e.Verb, e.Subject)
	}
	return fmt.Sprintf("verb %s is illegal for subject %s in state %s", e.Verb, e.Subject, e.From)
}

// IncompleteError is the contract-completeness shape refusal
// (plans/os-d69a6c91.md): the charter's birth rule enforced as
// payload-field presence at the shape level, content schemas landing
// with their own items.
type IncompleteError struct {
	Verb    string
	Subject string
	Missing []string
}

func (e *IncompleteError) Error() string {
	return fmt.Sprintf("%s on %s is incomplete: missing non-empty %s (the charter's completeness rule, presence-checked at admission)", e.Verb, e.Subject, strings.Join(e.Missing, ", "))
}

// Parse parses and self-validates a table. Every invariant violation
// refuses by name: an invalid table never loads.
func Parse(b []byte) (*Table, error) {
	var f tableFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("transition table does not parse: %v", err)
	}
	t := &Table{
		SchemaVersion: f.SchemaVersion,
		terminal:      map[string]bool{},
		states:        map[string]bool{},
		legal:         map[string]map[string]string{},
	}
	for _, s := range f.States {
		if t.states[s.Name] {
			return nil, fmt.Errorf("invalid transition table: duplicate state %q", s.Name)
		}
		t.states[s.Name] = true
		if s.Terminal {
			t.terminal[s.Name] = true
		}
		if s.Initial {
			if t.initial != "" {
				return nil, fmt.Errorf("invalid transition table: two initial states (%q, %q)", t.initial, s.Name)
			}
			t.initial = s.Name
		}
	}
	if t.initial == "" {
		return nil, fmt.Errorf("invalid transition table: no initial state")
	}
	outgoing := map[string]map[string]bool{}
	for _, r := range f.Transitions {
		if _, dup := t.legal[r.Verb]; dup || r.Verb == t.birth {
			return nil, fmt.Errorf("invalid transition table: duplicate verb %q", r.Verb)
		}
		if !t.states[r.To] {
			return nil, fmt.Errorf("invalid transition table: verb %q targets unknown state %q", r.Verb, r.To)
		}
		if r.From == nil {
			if t.birth != "" {
				return nil, fmt.Errorf("invalid transition table: two birth verbs (%q, %q)", t.birth, r.Verb)
			}
			t.birth, t.birthTo = r.Verb, r.To
			t.verbs = append(t.verbs, r.Verb)
			continue
		}
		m := map[string]string{}
		for _, from := range *r.From {
			if !t.states[from] {
				return nil, fmt.Errorf("invalid transition table: verb %q leaves unknown state %q", r.Verb, from)
			}
			if t.terminal[from] {
				return nil, fmt.Errorf("invalid transition table: terminal state %q has an outgoing row (%q)", from, r.Verb)
			}
			if _, dup := m[from]; dup {
				return nil, fmt.Errorf("invalid transition table: verb %q lists state %q twice", r.Verb, from)
			}
			m[from] = r.To
			if outgoing[from] == nil {
				outgoing[from] = map[string]bool{}
			}
			outgoing[from][r.Verb] = true
		}
		t.legal[r.Verb] = m
		t.verbs = append(t.verbs, r.Verb)
	}
	if t.birth == "" {
		return nil, fmt.Errorf("invalid transition table: no birth verb (a row with from: null)")
	}
	if t.birthTo != t.initial {
		return nil, fmt.Errorf("invalid transition table: the birth verb lands on %q, not the initial state %q", t.birthTo, t.initial)
	}
	// Reachability from the initial state; then every non-terminal
	// state must reach a terminal one (no wedge).
	next := map[string]map[string]bool{}
	for _, m := range t.legal {
		for from, to := range m {
			if next[from] == nil {
				next[from] = map[string]bool{}
			}
			next[from][to] = true
		}
	}
	reached := map[string]bool{t.initial: true}
	queue := []string{t.initial}
	for len(queue) > 0 {
		s := queue[0]
		queue = queue[1:]
		for to := range next[s] {
			if !reached[to] {
				reached[to] = true
				queue = append(queue, to)
			}
		}
	}
	for s := range t.states {
		if !reached[s] {
			return nil, fmt.Errorf("invalid transition table: state %q is unreachable from the initial state", s)
		}
	}
	for s := range t.states {
		if t.terminal[s] {
			continue
		}
		if !reachesTerminal(s, next, t.terminal, map[string]bool{}) {
			return nil, fmt.Errorf("invalid transition table: state %q cannot reach a terminal state (wedge)", s)
		}
	}
	// The pinned invariant (plans/os-d69a6c91.md, review finding on
	// #113): leaving in_progress happens only through the four
	// deliberate exits, so silent abandonment is structural.
	deliberate := []string{"claim.parked", "claim.released", "claim.reaped", "submission.made"}
	sort.Strings(deliberate)
	var got []string
	for v := range outgoing["in_progress"] {
		got = append(got, v)
	}
	sort.Strings(got)
	if strings.Join(got, " ") != strings.Join(deliberate, " ") {
		return nil, fmt.Errorf("invalid transition table: the in_progress exits must be exactly the four deliberate exits [%s], got [%s]", strings.Join(deliberate, " "), strings.Join(got, " "))
	}
	return t, nil
}

func reachesTerminal(s string, next map[string]map[string]bool, terminal, seen map[string]bool) bool {
	if terminal[s] {
		return true
	}
	if seen[s] {
		return false
	}
	seen[s] = true
	for to := range next[s] {
		if reachesTerminal(to, next, terminal, seen) {
			return true
		}
	}
	return false
}

var (
	defaultOnce  sync.Once
	defaultTable *Table
	defaultErr   error
)

// Default returns the embedded table, parsed and validated once. The
// embedded copy passing self-validation is itself drilled, so a build
// carrying an invalid table cannot pass its own suite.
func Default() (*Table, error) {
	defaultOnce.Do(func() { defaultTable, defaultErr = Parse(tableJSON) })
	return defaultTable, defaultErr
}

// Initial returns the initial state.
func (t *Table) Initial() string { return t.initial }

// BirthVerb returns the single verb that creates a subject.
func (t *Table) BirthVerb() string { return t.birth }

// Terminal reports whether the state is terminal.
func (t *Table) Terminal(state string) bool { return t.terminal[state] }

// Verbs returns the lifecycle vocabulary in table order.
func (t *Table) Verbs() []string { return append([]string(nil), t.verbs...) }

// IsLifecycleVerb reports whether the verb has a transition row.
func (t *Table) IsLifecycleVerb(verb string) bool {
	if verb == t.birth {
		return true
	}
	_, ok := t.legal[verb]
	return ok
}

// Check returns the state the verb moves the subject to. current is ""
// for a subject that does not exist yet; only the birth verb is legal
// then, and the birth verb is illegal for any existing subject.
func (t *Table) Check(subject, current, verb string) (string, error) {
	if verb == t.birth {
		if current != "" {
			return "", &InvalidTransitionError{Subject: subject, From: current, Verb: verb}
		}
		return t.birthTo, nil
	}
	if current == "" {
		return "", &InvalidTransitionError{Subject: subject, Verb: verb}
	}
	if to, ok := t.legal[verb][current]; ok {
		return to, nil
	}
	return "", &InvalidTransitionError{Subject: subject, From: current, Verb: verb}
}

// SubjectState is one subject's folded lifecycle state.
type SubjectState struct {
	State string
	// Since is the chain position of the event that put the subject
	// in its current state.
	Since int
	// Anomalies counts lifecycle events on the subject that the table
	// refuses: verification tolerates them (admission policy, not
	// chain validity), the fold skips them, and the projections
	// surface the count, never silence (plans/os-d69a6c91.md).
	Anomalies int
}

// Fold is the folded lifecycle state of every subject in a record
// prefix, in first-appearance order.
type Fold struct {
	states map[string]*SubjectState
	order  []string
}

// FoldRecords folds every subject's lifecycle events, skipping illegal
// history without wedging, the halt.StateAt posture.
func (t *Table) FoldRecords(records []*event.Record) *Fold {
	f := &Fold{states: map[string]*SubjectState{}}
	for pos, rec := range records {
		e := &rec.Event
		if !t.IsLifecycleVerb(e.Verb) {
			continue
		}
		s, ok := f.states[e.Subject]
		current := ""
		if ok {
			current = s.State
		}
		to, err := t.Check(e.Subject, current, e.Verb)
		if err != nil {
			if !ok {
				s = &SubjectState{}
				f.states[e.Subject] = s
				f.order = append(f.order, e.Subject)
			}
			s.Anomalies++
			continue
		}
		if !ok {
			s = &SubjectState{}
			f.states[e.Subject] = s
			f.order = append(f.order, e.Subject)
		}
		s.State, s.Since = to, pos
	}
	return f
}

// State returns a subject's folded state; ok is false when no
// lifecycle event ever named the subject. A subject whose every
// lifecycle event was anomalous has ok true and an empty State.
func (f *Fold) State(subject string) (SubjectState, bool) {
	s, ok := f.states[subject]
	if !ok {
		return SubjectState{}, false
	}
	return *s, true
}

// Subjects returns every folded subject in first-appearance order.
func (f *Fold) Subjects() []string { return append([]string(nil), f.order...) }

// StateAt folds one subject's lifecycle state out of a record prefix,
// the plan's named convenience over FoldRecords.
func (t *Table) StateAt(records []*event.Record, subject string) (SubjectState, bool) {
	return t.FoldRecords(records).State(subject)
}

// completeness maps the two completeness-gated verbs to the payload
// fields that must be present and non-empty (plans/os-d69a6c91.md:
// presence now, content schemas and gates with their own items).
var completeness = map[string][]string{
	"intent.filed":       {"intent", "tier", "budget", "routing"},
	"contract.specified": {"acceptance"},
}

// CheckCompleteness enforces the presence rule for the verb's payload.
func CheckCompleteness(verb, subject string, payload []byte) error {
	fields := completeness[verb]
	if len(fields) == 0 {
		return nil
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(payload, &m); err != nil {
		return &IncompleteError{Verb: verb, Subject: subject, Missing: fields}
	}
	var missing []string
	for _, f := range fields {
		raw, ok := m[f]
		if !ok || emptyJSON(raw) {
			missing = append(missing, f)
		}
	}
	if len(missing) > 0 {
		return &IncompleteError{Verb: verb, Subject: subject, Missing: missing}
	}
	return nil
}

// emptyJSON reports a value with no content: JSON null, an empty
// string, an empty object, or an empty array.
func emptyJSON(raw json.RawMessage) bool {
	s := strings.TrimSpace(string(raw))
	return s == "" || s == "null" || s == `""` || s == "{}" || s == "[]"
}
