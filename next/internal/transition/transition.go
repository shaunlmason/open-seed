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
	"github.com/shaunlmason/open-seed/next/internal/packet"
	"github.com/shaunlmason/open-seed/next/internal/version"
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
// Exclusive marks a verb whose admission grants exclusivity (claims):
// online-only by construction, since only the admission boundary can
// order two rivals (plans/os-5dc16a7c.md).
type row struct {
	Verb      string    `json:"verb"`
	From      *[]string `json:"from"`
	To        string    `json:"to"`
	Exclusive bool      `json:"exclusive"`
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
	legal     map[string]map[string]string
	exclusive map[string]bool
	verbs     []string
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
		exclusive:     map[string]bool{},
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
		if r.Exclusive {
			// Exclusivity is meaningful only for a verb that enters a
			// held state from somewhere: a birth verb cannot be a claim.
			if r.From == nil {
				return nil, fmt.Errorf("invalid transition table: the birth verb %q cannot be exclusive", r.Verb)
			}
			t.exclusive[r.Verb] = true
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

// Exclusive reports whether the verb's admission grants exclusivity.
func (t *Table) Exclusive(verb string) bool { return t.exclusive[verb] }

// Allows reports whether the verb legally leaves the given state.
func (t *Table) Allows(from, verb string) bool {
	_, ok := t.legal[verb][from]
	return ok
}

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

// AcceptanceInfo is the fold's view of a contract's acceptance spec:
// Gated reports that gate evidence is present or not required, so
// "may this spec run?" is a projection read (plans/os-73c00a50.md).
type AcceptanceInfo struct {
	Ref        string
	Executable bool
	Gated      bool
}

// Claim is the active claim on an in_progress subject: the fence is
// the chain position of the admitted claim.taken record — derived,
// never asserted (plans/os-5dc16a7c.md) — and the holder its signer.
type Claim struct {
	Holder string
	Fence  int
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
	// Claim is the active claim while the subject is in_progress,
	// cleared by every deliberate exit; nil otherwise.
	Claim *Claim
	// Tier is the contract's filed tier (presence-only data whose one
	// distinguished value, "trivial", exempts the plan gate;
	// plans/os-16c1d142.md).
	Tier string
	// Acceptance is the folded acceptance spec from the last admitted
	// contract.specified: the artifact anchor, the executable flag,
	// and whether gate evidence bound to the revision is present (or
	// not required). Nil until specified; a raw-pushed specification
	// whose acceptance is invalid counts an anomaly and leaves what
	// is honestly derivable.
	Acceptance *AcceptanceInfo
	// PriorClaimants is every fingerprint that has ever held a claim
	// on this subject: the fence rule's who-must-cite input — a
	// reaped or released worker cannot demote itself to observer
	// (plans/os-5dc16a7c.md, review finding on #114).
	PriorClaimants map[string]bool
}

// milestoneFact is a subject's milestone high-water mark: the highest
// admitted count and the chain position of the latest milestone.
type milestoneFact struct {
	Count int
	Pos   int
}

// Fold is the folded lifecycle state of every subject in a record
// prefix, in first-appearance order.
type Fold struct {
	states map[string]*SubjectState
	order  []string
	// planned maps subject -> the last admitted plan.approved's plan
	// anchor. plan.* events are facts, not transitions
	// (plans/os-16c1d142.md): they change no lifecycle state and the
	// submission gate consults them.
	planned map[string]string
	// milestones maps subject -> its milestone high-water mark
	// (plans/os-2ff8dbf1.md): progress.milestone is a fact too, and
	// the summarization boundary consults the mark. Raw-pushed
	// regressions keep the maximum count, so tolerated history never
	// lowers the monotonic bar.
	milestones map[string]milestoneFact
}

// PlanApproved reports the subject's approved-plan anchor, if any.
func (f *Fold) PlanApproved(subject string) (string, bool) {
	ref, ok := f.planned[subject]
	return ref, ok
}

// FoldRecords folds every subject's lifecycle events, skipping illegal
// history without wedging, the halt.StateAt posture.
func (t *Table) FoldRecords(records []*event.Record) *Fold {
	f := &Fold{states: map[string]*SubjectState{}, planned: map[string]string{}, milestones: map[string]milestoneFact{}}
	for pos, rec := range records {
		e := &rec.Event
		if e.V != version.Seed1 {
			// Lifecycle semantics activate at seed/1 (the
			// keyring.Applies posture): grandfathered earlier records
			// stay inert even where their verb names later became
			// lifecycle verbs, so an upgraded ledger's history cannot
			// occupy states or make a real filing look like a second
			// birth. Version discipline pins e.V to the version
			// active at the record's position.
			continue
		}
		if e.Verb == PlanApprovedVerb {
			if ref, _ := planAnchor(e.Payload); ref != "" {
				f.planned[e.Subject] = ref
			}
			continue
		}
		if e.Verb == MilestoneVerb {
			// Milestone semantics activate at seed/1 like the rest of
			// the summarization boundary: a grandfathered pre-upgrade
			// record whose payload happens to carry a count must not
			// become the high-water mark and wedge legitimate
			// post-upgrade progress. Version discipline pins e.V to
			// the version active at the record's position.
			var m struct {
				Count *int `json:"count"`
			}
			if e.V == version.Seed1 && json.Unmarshal(e.Payload, &m) == nil && m.Count != nil {
				fact, seen := f.milestones[e.Subject]
				if !seen || *m.Count > fact.Count {
					fact.Count = *m.Count
				}
				fact.Pos = pos
				f.milestones[e.Subject] = fact
			}
			continue
		}
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
		// Raw-pushed history can carry a legal transition with a
		// stale or missing fence citation, or a deliberate exit
		// without its handoff packet. Admission refuses both; the
		// tolerant fold applies the transition (the exit happened —
		// skipping it would wedge the subject on a dead holder) and
		// counts the violation visibly, never silently
		// (plans/os-5dc16a7c.md, plans/os-b07b0f59.md).
		if s.Claim != nil && t.Allows(current, e.Verb) {
			if cited, _ := fenceCited(e.Payload); cited != fmt.Sprintf("%d", s.Claim.Fence) {
				s.Anomalies++
			}
		}
		if packet.Required(e.Verb) {
			if _, perr := packet.FromPayload(e.Subject, e.Payload); perr != nil {
				s.Anomalies++
			}
		}
		if e.Verb == t.birth {
			var filed struct {
				Tier string `json:"tier"`
			}
			if json.Unmarshal(e.Payload, &filed) == nil {
				s.Tier = filed.Tier
			}
		}
		if e.Verb == "contract.specified" {
			if a, aerr := ParseAcceptance(e.Subject, e.Payload); aerr == nil {
				s.Acceptance = &AcceptanceInfo{Ref: a.Ref, Executable: a.Executable, Gated: !a.Executable || a.Gate != ""}
			} else {
				// Raw-pushed ungated or malformed acceptance:
				// tolerated in history, visibly counted, with what is
				// honestly derivable kept for the view.
				s.Anomalies++
				var raw struct {
					Acceptance Acceptance `json:"acceptance"`
				}
				if json.Unmarshal(e.Payload, &raw) == nil && raw.Acceptance.Ref != "" {
					s.Acceptance = &AcceptanceInfo{Ref: raw.Acceptance.Ref, Executable: raw.Acceptance.Executable, Gated: false}
				}
			}
		}
		s.State, s.Since = to, pos
		if t.exclusive[e.Verb] {
			s.Claim = &Claim{Holder: e.Actor, Fence: pos}
			if s.PriorClaimants == nil {
				s.PriorClaimants = map[string]bool{}
			}
			s.PriorClaimants[e.Actor] = true
		} else if s.Claim != nil && s.State != "in_progress" {
			// Every deliberate exit ends the claim window; the fence
			// dies with it.
			s.Claim = nil
		}
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

// completeness maps intent.filed to the payload fields that must be
// present and non-empty (plans/os-d69a6c91.md); contract.specified
// deepened from presence to the structured acceptance rule
// (plans/os-73c00a50.md).
var completeness = map[string][]string{
	"intent.filed": {"intent", "tier", "budget", "routing"},
}

// CheckCompleteness enforces the completeness rules for the verb's
// payload: field presence for filings, the structured acceptance
// field (gate included) for specifications.
func CheckCompleteness(verb, subject string, payload []byte) error {
	if verb == "contract.specified" {
		_, err := ParseAcceptance(subject, payload)
		return err
	}
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

// fenceCited extracts a payload's fence citation for the tolerant
// fold; admission's strict twin lives in the fence rule.
func fenceCited(payload []byte) (string, bool) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(payload, &m); err != nil {
		return "", false
	}
	raw, ok := m["fence"]
	if !ok {
		return "", false
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return strings.TrimSpace(string(raw)), true
	}
	return s, true
}

// The plan.* vocabulary (the charter catalog's only two plan verbs;
// plans/os-16c1d142.md): proposals and merge observations are facts
// the fold and the submission gate consult.
const (
	PlanProposedVerb = "plan.proposed"
	PlanApprovedVerb = "plan.approved"
	// TrivialTier is the one distinguished tier value: the charter's
	// own term for the tier whose contracts submit without a plan.
	TrivialTier = "trivial"
)

// The observation summarization vocabulary (plans/os-2ff8dbf1.md;
// SEED-NEXT.md Part II §5): the ephemeral channel is summarized into
// ledger facts at material transitions, and these are the facts. Both
// are free verbs, never transitions: the pinned four in_progress
// exits stand.
const (
	MilestoneVerb     = "progress.milestone"
	WedgeDeclaredVerb = "wedge.declared"
	// MinMilestoneSpacing is the v0 bounded-frequency default: the
	// minimum chain positions since the subject's last admitted
	// milestone. Spacing is measured in positions, never timestamps:
	// ts is human-readable metadata with no ordering authority, so a
	// time rule would be signer-gameable and skew-prone, while
	// position spacing is admission-derived, replay-deterministic,
	// and bounds the protected quantity, the subject's share of
	// ledger volume.
	MinMilestoneSpacing = 25
)

// MilestoneError refuses a progress.milestone that violates the
// monotonic or bounded-frequency rule at the summarization boundary.
// It rides the established shape-refusal exit mapping: the plan
// allocates no new code.
type MilestoneError struct {
	Subject string
	Reason  string
}

func (e *MilestoneError) Error() string {
	return fmt.Sprintf("progress.milestone on %s refused: %s (next/spec/observations.md)", e.Subject, e.Reason)
}

// CheckMilestone enforces the summarization boundary for a milestone
// landing at position tip: payload presence ({count, step}), a
// strictly advancing count against the fold's high-water mark, and
// the minimum position spacing since the subject's latest milestone.
func (f *Fold) CheckMilestone(subject string, tip int, payload []byte) error {
	var m struct {
		Count *int   `json:"count"`
		Step  string `json:"step"`
	}
	var missing []string
	if json.Unmarshal(payload, &m) != nil || m.Count == nil {
		missing = append(missing, "count")
	}
	if strings.TrimSpace(m.Step) == "" {
		missing = append(missing, "step")
	}
	if len(missing) > 0 {
		return &IncompleteError{Verb: MilestoneVerb, Subject: subject, Missing: missing}
	}
	last, ok := f.milestones[subject]
	if !ok {
		return nil
	}
	if *m.Count <= last.Count {
		return &MilestoneError{Subject: subject, Reason: fmt.Sprintf("count %d does not advance the last admitted milestone count %d — progress counts are monotonic, never clock-derived", *m.Count, last.Count)}
	}
	if tip-last.Pos < MinMilestoneSpacing {
		return &MilestoneError{Subject: subject, Reason: fmt.Sprintf("position spacing %d since the milestone at position %d is under the minimum %d — milestones are coarse, bounded-frequency summaries", tip-last.Pos, last.Pos, MinMilestoneSpacing)}
	}
	return nil
}

// CheckWedgeShape enforces wedge.declared's presence payload
// ({observed, count, since}): the visible wedge condition recorded
// durably, changing no state.
func CheckWedgeShape(subject string, payload []byte) error {
	var m struct {
		Observed string `json:"observed"`
		Count    *int   `json:"count"`
		Since    string `json:"since"`
	}
	if json.Unmarshal(payload, &m) != nil {
		return &IncompleteError{Verb: WedgeDeclaredVerb, Subject: subject, Missing: []string{"observed", "count", "since"}}
	}
	var missing []string
	if strings.TrimSpace(m.Observed) == "" {
		missing = append(missing, "observed")
	}
	if m.Count == nil {
		missing = append(missing, "count")
	}
	if strings.TrimSpace(m.Since) == "" {
		missing = append(missing, "since")
	}
	if len(missing) > 0 {
		return &IncompleteError{Verb: WedgeDeclaredVerb, Subject: subject, Missing: missing}
	}
	return nil
}

// planAnchor extracts a payload's plan anchor.
func planAnchor(payload []byte) (string, bool) {
	var m struct {
		Plan string `json:"plan"`
	}
	if err := json.Unmarshal(payload, &m); err != nil {
		return "", false
	}
	return m.Plan, m.Plan != ""
}

// PlanRequiredError is the plan-gate refusal (exit 16 plan_required):
// claiming an unplanned contract authorizes planning only, so a
// submission above the trivial tier needs an approved plan and must
// cite the plan anchor it implements. Missing names which half.
type PlanRequiredError struct {
	Subject string
	Tier    string
	Missing string
}

func (e *PlanRequiredError) Error() string {
	return fmt.Sprintf("submission on %s (tier %q) refused: %s — above the trivial tier, claiming an unplanned contract authorizes planning only; merge the plan PR (observed as plan.approved) and cite its anchor (next/spec/plans.md)", e.Subject, e.Tier, e.Missing)
}

// CheckPlanGate enforces the submission gate: above the trivial tier,
// the subject carries an admitted plan.approved and the submission
// payload cites the plan anchor it implements. The ancestry binding
// (the implementation actually built on the approved plan) is
// Phase 6's receipt computation, the named closing item.
func (f *Fold) CheckPlanGate(subject, tier string, payload []byte) error {
	if tier == TrivialTier {
		return nil
	}
	approved, ok := f.PlanApproved(subject)
	if !ok {
		return &PlanRequiredError{Subject: subject, Tier: tier, Missing: "no plan.approved on the subject"}
	}
	cited, ok := planAnchor(payload)
	if !ok {
		return &PlanRequiredError{Subject: subject, Tier: tier, Missing: "the submission must cite the plan anchor it implements ({\"plan\": \"<path @ commit>\"})"}
	}
	// Citation means THE approved plan, anchor for anchor: an approval
	// admits one exact revision, and citing any other leaves the
	// receipt verifier holding an anchor nothing vouched for.
	if cited != approved {
		return &PlanRequiredError{Subject: subject, Tier: tier, Missing: fmt.Sprintf("the cited plan %q is not the approved plan %q", cited, approved)}
	}
	return nil
}

// CheckPlanEventShape enforces payload presence for the plan.* verbs:
// a proposal names the plan artifact anchor; an approval names the
// plan anchor and the merged PR (both combined anchors, the
// external-fact observation posture).
func CheckPlanEventShape(verb, subject string, payload []byte) error {
	switch verb {
	case PlanProposedVerb:
		if _, ok := planAnchor(payload); !ok {
			return &IncompleteError{Verb: verb, Subject: subject, Missing: []string{"plan"}}
		}
	case PlanApprovedVerb:
		var m struct {
			Plan string `json:"plan"`
			PR   string `json:"pr"`
		}
		if err := json.Unmarshal(payload, &m); err != nil || m.Plan == "" || m.PR == "" {
			var missing []string
			if m.Plan == "" {
				missing = append(missing, "plan")
			}
			if m.PR == "" {
				missing = append(missing, "pr")
			}
			return &IncompleteError{Verb: verb, Subject: subject, Missing: missing}
		}
	}
	return nil
}
