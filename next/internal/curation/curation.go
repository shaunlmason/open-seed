// Package curation is the staged learning pipeline's ledger half
// (plans/os-f30ee0d3.md; SEED-NEXT.md §II.12; build plan Phase 11 item
// 1): the three curation facts, their strict payload shapes, the
// hypothesis id derivation, and the fold that renders the stages.
//
// Trajectories are untrusted inputs, so the stages have distinct
// storage and distinct gates. Observations live on the contract (the
// packet's findings, and deadend.recorded inside the holder's window);
// hypotheses live on their own subject, derived from the claim, and
// only the curator's proposal grant reaches them; validated lessons are
// files merged by PR, and the ledger carries the observation of that
// merge, citing the hypothesis it promotes. No stage skips: a promotion
// that cites no admitted hypothesis folds unbound, never as a lesson.
package curation

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/keyring"
	"github.com/shaunlmason/open-seed/next/internal/packet"
	"github.com/shaunlmason/open-seed/next/internal/transition"
)

// The three verbs, under the catalog's curation namespace
// (next/spec/protocol.md). Additive growth: no version bump.
const (
	DeadEndVerb    = "curation.deadend.recorded"
	HypothesisVerb = "curation.hypothesis.proposed"
	LessonVerb     = "curation.lesson.promoted"
)

// LessonsDir is the validated-lessons store: files merged by PR, one
// per lesson, whose frontmatter Lint checks (next/knowledge/lessons).
const LessonsDir = "next/knowledge/lessons"

// SupportMinimum is the structural floor a hypothesis must cite: at
// least this many admitted observations, on at least this many
// distinct non-failed contracts. Item 2's promotion gate builds on it;
// a single run can never be promoted from, by construction.
const SupportMinimum = 2

// hypothesisIDRE is the subject shape a hypothesis lives on.
var hypothesisIDRE = regexp.MustCompile(`^h-[0-9a-f]{12}$`)

// anchorRE is the classification lint's commit-anchor form: a path in
// a filename alphabet anchored to a commit.
var anchorRE = regexp.MustCompile(`^[A-Za-z0-9._/-]{1,200} @ [0-9a-f]{7,64}$`)

// HypothesisID derives the subject a claim is proposed on: h- and the
// first twelve hex digits of the SHA-256 of the canonical claim text
// (whitespace-trimmed, single-spaced). Two proposals of one claim
// derive one subject, so a re-proposal refuses as a duplicate at the
// boundary rather than accumulating.
func HypothesisID(claim string) string {
	sum := sha256.Sum256([]byte(canonicalClaim(claim)))
	return "h-" + hex.EncodeToString(sum[:])[:12]
}

func canonicalClaim(claim string) string {
	return strings.Join(strings.Fields(claim), " ")
}

// IsHypothesisSubject reports whether the subject has a hypothesis id's
// shape.
func IsHypothesisSubject(subject string) bool { return hypothesisIDRE.MatchString(subject) }

// DeadEnd is deadend.recorded's payload: the packet finding's shape
// plus the charter's failure condition and environment, inside the
// holder's window. A candidate and nothing more: it has no field a
// conclusion could live in.
type DeadEnd struct {
	Fence       string `json:"fence"`
	Tried       string `json:"tried"`
	Outcome     string `json:"outcome"`
	Condition   string `json:"condition"`
	Environment string `json:"environment"`
	Pointer     string `json:"pointer,omitempty"`
}

// Hypothesis is hypothesis.proposed's payload.
type Hypothesis struct {
	Claim       string   `json:"claim"`
	AppliesWhen string   `json:"applies_when"`
	Support     []string `json:"support"`
	Exceptions  []string `json:"exceptions"`
	Provenance  []string `json:"provenance"`
}

// Lesson is lesson.promoted's payload: the observation that a lesson
// file landed by PR, citing the hypothesis it promotes.
type Lesson struct {
	Lesson     string `json:"lesson"`
	Hypothesis string `json:"hypothesis"`
	PR         string `json:"pr"`
}

// ShapeError is a payload shape refusal naming the part.
type ShapeError struct {
	Verb, Subject, Reason string
}

func (e *ShapeError) Error() string {
	return fmt.Sprintf("%s on %s: %s", e.Verb, e.Subject, e.Reason)
}

func strict(raw []byte, into any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&into); err != nil {
		return err
	}
	if dec.More() {
		return errors.New("trailing data after the payload object")
	}
	return nil
}

// ParseDeadEnd decodes and shape-checks a dead end: every field
// non-empty, the fence a decimal position, the pointer (if any) an
// anchored path.
func ParseDeadEnd(subject string, raw []byte) (*DeadEnd, error) {
	var d DeadEnd
	if err := strict(raw, &d); err != nil {
		return nil, &ShapeError{DeadEndVerb, subject, "the payload is the strict object {fence, tried, outcome, condition, environment, pointer?}: " + err.Error()}
	}
	var missing []string
	for _, f := range []struct{ name, v string }{{"fence", d.Fence}, {"tried", d.Tried}, {"outcome", d.Outcome}, {"condition", d.Condition}, {"environment", d.Environment}} {
		if strings.TrimSpace(f.v) == "" {
			missing = append(missing, f.name)
		}
	}
	if len(missing) > 0 {
		return nil, &transition.IncompleteError{Verb: DeadEndVerb, Subject: subject, Missing: missing}
	}
	if _, err := strconv.Atoi(d.Fence); err != nil {
		return nil, &ShapeError{DeadEndVerb, subject, fmt.Sprintf("fence %q is not a position", d.Fence)}
	}
	if d.Pointer != "" && !anchorRE.MatchString(d.Pointer) {
		return nil, &ShapeError{DeadEndVerb, subject, fmt.Sprintf("pointer %q is not an anchored path (\"<path> @ <commit>\"): a bare path assumes a shared filesystem disposable executors do not have", d.Pointer)}
	}
	return &d, nil
}

// Citation is one support entry, "<contract>@<position>".
type Citation struct {
	Contract string
	Position int
}

// ParseCitation splits "<contract>@<position>".
func ParseCitation(s string) (Citation, bool) {
	i := strings.LastIndex(s, "@")
	if i <= 0 || i == len(s)-1 {
		return Citation{}, false
	}
	pos, err := strconv.Atoi(s[i+1:])
	if err != nil || pos < 0 {
		return Citation{}, false
	}
	return Citation{Contract: s[:i], Position: pos}, true
}

// ParseHypothesis decodes and shape-checks a proposal: the claim and
// applies-when non-empty, the subject derived from the claim, support
// entries well-formed citations, provenance entries anchored paths.
// Lists may be empty here; the support floor is the boundary's rule,
// judged against the record.
func ParseHypothesis(subject string, raw []byte) (*Hypothesis, error) {
	var h Hypothesis
	if err := strict(raw, &h); err != nil {
		return nil, &ShapeError{HypothesisVerb, subject, "the payload is the strict object {claim, applies_when, support, exceptions, provenance}: " + err.Error()}
	}
	var missing []string
	if strings.TrimSpace(h.Claim) == "" {
		missing = append(missing, "claim")
	}
	if strings.TrimSpace(h.AppliesWhen) == "" {
		missing = append(missing, "applies_when")
	}
	if h.Support == nil {
		missing = append(missing, "support")
	}
	if h.Exceptions == nil {
		missing = append(missing, "exceptions")
	}
	if h.Provenance == nil {
		missing = append(missing, "provenance")
	}
	if len(missing) > 0 {
		return nil, &transition.IncompleteError{Verb: HypothesisVerb, Subject: subject, Missing: missing}
	}
	if want := HypothesisID(h.Claim); subject != want {
		return nil, &ShapeError{HypothesisVerb, subject, fmt.Sprintf("the subject is not derived from the claim: the claim derives %s", want)}
	}
	for _, s := range h.Support {
		if _, ok := ParseCitation(s); !ok {
			return nil, &ShapeError{HypothesisVerb, subject, fmt.Sprintf("support entry %q is not \"<contract>@<position>\"", s)}
		}
	}
	for _, p := range h.Provenance {
		if !anchorRE.MatchString(p) {
			return nil, &ShapeError{HypothesisVerb, subject, fmt.Sprintf("provenance entry %q is not an anchored path (\"<path> @ <commit>\")", p)}
		}
	}
	return &h, nil
}

// ParseLesson decodes and shape-checks a promotion: an anchored lesson
// path under the store, a hypothesis citation whose subject is the
// event's, and an anchored PR.
func ParseLesson(subject string, raw []byte) (*Lesson, error) {
	var l Lesson
	if err := strict(raw, &l); err != nil {
		return nil, &ShapeError{LessonVerb, subject, "the payload is the strict object {lesson, hypothesis, pr}: " + err.Error()}
	}
	var missing []string
	for _, f := range []struct{ name, v string }{{"lesson", l.Lesson}, {"hypothesis", l.Hypothesis}, {"pr", l.PR}} {
		if strings.TrimSpace(f.v) == "" {
			missing = append(missing, f.name)
		}
	}
	if len(missing) > 0 {
		return nil, &transition.IncompleteError{Verb: LessonVerb, Subject: subject, Missing: missing}
	}
	if !anchorRE.MatchString(l.Lesson) {
		return nil, &ShapeError{LessonVerb, subject, fmt.Sprintf("lesson %q is not an anchored path (\"<path> @ <commit>\")", l.Lesson)}
	}
	if !UnderLessonsDir(l.Lesson) {
		return nil, &ShapeError{LessonVerb, subject, fmt.Sprintf("lesson %q is not under %s/: the validated store is one place, and a path that climbs out of it names a file the store's lint never sees", l.Lesson, LessonsDir)}
	}
	if !anchorRE.MatchString(l.PR) {
		return nil, &ShapeError{LessonVerb, subject, fmt.Sprintf("pr %q is not \"<pr> @ <merged-commit>\"", l.PR)}
	}
	c, ok := ParseCitation(l.Hypothesis)
	if !ok || !IsHypothesisSubject(c.Contract) {
		return nil, &ShapeError{LessonVerb, subject, fmt.Sprintf("hypothesis %q is not \"<h-id>@<position>\"", l.Hypothesis)}
	}
	if c.Contract != subject {
		return nil, &ShapeError{LessonVerb, subject, fmt.Sprintf("the cited hypothesis %s is not this subject: a promotion is a fact on the hypothesis it promotes", c.Contract)}
	}
	return &l, nil
}

// UnderLessonsDir reports whether the anchor's path lies inside the
// lessons store: the path is clean (no `.` or `..` segment, no doubled
// or trailing separator, so a lexical prefix cannot be climbed out
// of), relative, and prefixed by the store. A path such as
// `next/knowledge/lessons/../x` matches the anchor grammar and
// resolves outside the store, which is what the promotion must never
// name (review finding on the task PR).
func UnderLessonsDir(anchor string) bool {
	p, _, _ := strings.Cut(anchor, " @ ")
	p = strings.TrimSpace(p)
	if p == "" || path.Clean(p) != p || strings.HasPrefix(p, "/") {
		return false
	}
	return strings.HasPrefix(p, LessonsDir+"/")
}

// Observation is one admitted stage-one fact: a packet finding's
// deliberate exit, or a dead end, on a contract, at a position, by an
// actor.
type Observation struct {
	Contract string
	Position int
	Verb     string
	Actor    string
}

// ObservationAt re-judges the record at the cited position as an
// admitted observation on the cited contract: a deliberate exit whose
// packet carries at least one finding, or a dead end, each signed by
// the window's holder (or a prior claimant, for exits) and citing the
// fence active at its own prefix. Fold presence is never proof of
// admission (the RunStartValid posture), so a raw-pushed dead end
// outside a window supports nothing.
func ObservationAt(records []*event.Record, table *transition.Table, c Citation) (*Observation, bool) {
	if table == nil || c.Position < 0 || c.Position >= len(records) {
		return nil, false
	}
	rec := records[c.Position]
	e := &rec.Event
	if e.Subject != c.Contract {
		return nil, false
	}
	prefix := table.FoldRecords(records[:c.Position])
	s, ok := prefix.State(e.Subject)
	if !ok || s.Claim == nil || !WindowAdmitted(records, s) {
		return nil, false
	}
	switch {
	case e.Verb == DeadEndVerb:
		d, err := ParseDeadEnd(e.Subject, e.Payload)
		if err != nil || d.Fence != strconv.Itoa(s.Claim.Fence) || e.Actor != s.Claim.Holder {
			return nil, false
		}
	case isExit(e.Verb):
		if !table.Allows(s.State, e.Verb) {
			return nil, false
		}
		if e.Actor != s.Claim.Holder && !s.PriorClaimants[e.Actor] {
			return nil, false
		}
		p, err := packet.FromPayload(e.Subject, e.Payload)
		if err != nil || len(p.Findings) == 0 {
			return nil, false
		}
		var f struct {
			Fence string `json:"fence"`
		}
		if json.Unmarshal(e.Payload, &f) != nil || f.Fence != strconv.Itoa(s.Claim.Fence) {
			return nil, false
		}
	default:
		return nil, false
	}
	return &Observation{Contract: e.Subject, Position: c.Position, Verb: e.Verb, Actor: e.Actor}, true
}

// isExit reports whether the verb is one of the four deliberate exits:
// the packets that carry findings as observations. A raise carries a
// packet too and is not an exit; what it asks is a question, not a
// finding a hypothesis may rest on.
func isExit(verb string) bool {
	for _, v := range packet.ExitVerbs {
		if v == verb {
			return true
		}
	}
	return false
}

// WindowAdmitted re-judges the claim window the fold reports: the
// claim.taken at the fence is on the subject, signed by the holder,
// and the holder held a capability the claim accepts at that prefix.
// The lifecycle fold applies a legal transition whoever signed it, so
// a raw-pushed claim by a grantless key opens an apparent window;
// what looks like a holder's observation inside it passed no boundary
// (review finding on the task PR).
func WindowAdmitted(records []*event.Record, s transition.SubjectState) bool {
	if s.Claim == nil || s.Claim.Fence < 0 || s.Claim.Fence >= len(records) {
		return false
	}
	at := &records[s.Claim.Fence].Event
	if at.Verb != "claim.taken" || at.Actor != s.Claim.Holder {
		return false
	}
	ring, _, err := keyring.StateAt(records[:s.Claim.Fence])
	return err == nil && ring != nil && ring.HasAnyCapability(s.Claim.Holder, keyring.AcceptedCapabilities("claim.taken"))
}

// FailedAt reports whether the contract stands failed: its latest
// AUTHENTICATED verdict is a fail. A verdict authenticates at its own
// position when its signer held a verdict grant there and was no
// claimant, past or present, nor the bound submission's signer at that
// prefix (the verifier boundary's L1 rule); a raw-pushed pass by a
// grantless or implementing key after an authentic fail clears nothing
// (review finding on the task PR). A failed trajectory supports
// nothing (the charter's support rule).
func FailedAt(records []*event.Record, table *transition.Table, contract string) bool {
	failed := false
	for pos, rec := range records {
		e := &rec.Event
		if e.Verb != transition.VerdictRenderedVerb || e.Subject != contract || !keyring.Applies(e.V) {
			continue
		}
		var p struct {
			Verdict string `json:"verdict"`
		}
		if json.Unmarshal(e.Payload, &p) != nil || (p.Verdict != "pass" && p.Verdict != "fail") {
			continue
		}
		prefix := table.FoldRecords(records[:pos])
		s, ok := prefix.State(contract)
		if !ok || s.State != "review" || s.Submission == nil {
			continue
		}
		if e.Actor == s.Submission.Signer || s.PriorClaimants[e.Actor] || (s.Claim != nil && e.Actor == s.Claim.Holder) {
			continue
		}
		ring, _, err := keyring.StateAt(records[:pos])
		if err != nil || ring == nil || !ring.HasAnyCapability(e.Actor, keyring.AcceptedCapabilities(transition.VerdictRenderedVerb)) {
			continue
		}
		failed = p.Verdict == "fail"
	}
	return failed
}

// SupportError is the boundary's refusal of a proposal's support.
type SupportError struct {
	Subject, Reason string
}

func (e *SupportError) Error() string {
	return fmt.Sprintf("%s on %s: %s", HypothesisVerb, e.Subject, e.Reason)
}

// CheckSupport judges a proposal's citations against the record: each
// an admitted observation, no duplicate, at least SupportMinimum on at
// least SupportMinimum distinct contracts, none failed. It returns the
// observations, in citation order.
func CheckSupport(records []*event.Record, table *transition.Table, fold *transition.Fold, subject string, h *Hypothesis) ([]Observation, error) {
	seen := map[string]bool{}
	contracts := map[string]bool{}
	var out []Observation
	for _, raw := range h.Support {
		c, _ := ParseCitation(raw)
		if seen[raw] {
			return nil, &SupportError{subject, fmt.Sprintf("support cites %s twice", raw)}
		}
		seen[raw] = true
		o, ok := ObservationAt(records, table, c)
		if !ok {
			return nil, &SupportError{subject, fmt.Sprintf("support entry %s is not an admitted observation (a deliberate exit with findings or a dead end, inside the holder's window)", raw)}
		}
		if FailedAt(records, table, c.Contract) {
			return nil, &SupportError{subject, fmt.Sprintf("support entry %s cites a failed contract: a failed trajectory supports nothing", raw)}
		}
		contracts[c.Contract] = true
		out = append(out, *o)
	}
	if len(out) < SupportMinimum || len(contracts) < SupportMinimum {
		return nil, &SupportError{subject, fmt.Sprintf("support cites %d observation(s) on %d contract(s); a hypothesis needs at least %d admitted observations on %d distinct non-failed contracts, so no single run is promotable by construction", len(out), len(contracts), SupportMinimum, SupportMinimum)}
	}
	return out, nil
}

// HypothesisValid re-judges the cited record as an admitted proposal:
// a hypothesis on the cited subject at the cited position, whose
// signer held curate at that prefix, whose support passed the
// boundary's rule there, and that no earlier admitted proposal of the
// same subject makes a duplicate. Fold presence is never proof of
// admission, so a raw-pushed proposal promotes nothing: the stage it
// would skip is the one the citation re-judges.
func HypothesisValid(records []*event.Record, table *transition.Table, c Citation) (*Hypothesis, bool) {
	h, ok := proposalPasses(records, table, c)
	if !ok || AdmittedProposalBefore(records, table, c.Contract, c.Position) {
		return nil, false
	}
	return h, true
}

// proposalPasses is the position-accurate half of admission without
// the duplicate rule: the shape, the grant at the prefix, and the
// support judged there.
func proposalPasses(records []*event.Record, table *transition.Table, c Citation) (*Hypothesis, bool) {
	if table == nil || c.Position < 0 || c.Position >= len(records) {
		return nil, false
	}
	e := &records[c.Position].Event
	if e.Verb != HypothesisVerb || e.Subject != c.Contract || !keyring.Applies(e.V) {
		return nil, false
	}
	h, err := ParseHypothesis(e.Subject, e.Payload)
	if err != nil {
		return nil, false
	}
	prefix := records[:c.Position]
	ring, _, err := keyring.StateAt(prefix)
	if err != nil || ring == nil || !ring.HasAnyCapability(e.Actor, []string{keyring.CapCurate}) {
		return nil, false
	}
	if _, err := CheckSupport(prefix, table, table.FoldRecords(prefix), e.Subject, h); err != nil {
		return nil, false
	}
	return h, true
}

// AdmittedProposalBefore reports whether an admitted proposal on the
// subject stands at a position before the given one: the duplicate a
// re-proposal is refused against. Only a proposal that passed the
// boundary counts, so a raw-pushed, well-shaped proposal by a key
// holding no curate cannot reserve a hypothesis id (review finding on
// the task PR). The earliest proposal that passes the grant and the
// support is the admitted one, every later one its duplicate, so one
// forward scan settles it.
func AdmittedProposalBefore(records []*event.Record, table *transition.Table, subject string, before int) bool {
	if before > len(records) {
		before = len(records)
	}
	for pos := 0; pos < before; pos++ {
		e := &records[pos].Event
		if e.Verb != HypothesisVerb || e.Subject != subject {
			continue
		}
		if _, ok := proposalPasses(records, table, Citation{Contract: subject, Position: pos}); ok {
			return true
		}
	}
	return false
}

// DeadEndFact is a folded dead end.
type DeadEndFact struct {
	Pos         int    `json:"position"`
	Actor       string `json:"actor"`
	Fence       string `json:"fence"`
	Tried       string `json:"tried"`
	Outcome     string `json:"outcome"`
	Condition   string `json:"condition"`
	Environment string `json:"environment"`
	Pointer     string `json:"pointer,omitempty"`
}

// HypothesisFact is a folded hypothesis with its stage.
type HypothesisFact struct {
	ID          string   `json:"id"`
	Pos         int      `json:"position"`
	Proposer    string   `json:"proposer"`
	Claim       string   `json:"claim"`
	AppliesWhen string   `json:"applies_when"`
	Support     []string `json:"support"`
	Exceptions  []string `json:"exceptions"`
	Provenance  []string `json:"provenance"`
	// Stage is proposed, or promoted once a lesson cites it.
	Stage string `json:"stage"`
	// Lesson is the promotion's position, when promoted.
	Lesson *int `json:"lesson,omitempty"`
}

// LessonFact is a folded promotion.
type LessonFact struct {
	Pos        int    `json:"position"`
	Actor      string `json:"actor"`
	Lesson     string `json:"lesson"`
	Hypothesis string `json:"hypothesis"`
	PR         string `json:"pr"`
}

// The stages a hypothesis renders under.
const (
	StageProposed = "proposed"
	StagePromoted = "promoted"
)

// State is the curation fold: dead ends by contract, hypotheses by id,
// lessons by path, and the promotions that cite no admitted
// hypothesis, which are unbound rather than lessons.
type State struct {
	DeadEnds   map[string][]DeadEndFact
	Hypotheses map[string]*HypothesisFact
	Lessons    map[string]LessonFact
	Unbound    []LessonFact
	// Anomalies counts malformed raw facts: counted, never folded.
	Anomalies int
	order     []string
}

// HypothesisIDs returns the folded hypothesis ids in proposal order.
func (s *State) HypothesisIDs() []string { return append([]string(nil), s.order...) }

// Hypothesis returns the folded hypothesis, if any.
func (s *State) Hypothesis(id string) (*HypothesisFact, bool) {
	h, ok := s.Hypotheses[id]
	return h, ok
}

// Fold renders the curation facts of a verified prefix. Tolerant like
// every fold: a malformed raw fact counts an anomaly and folds nothing;
// facts before the keyring boundary are inert. Whether a fact PASSED
// the boundary (the window, the grant, the support) is admission's
// question; the fold records what the chain carries, and the
// boundary re-judges citations from the records.
func Fold(records []*event.Record) *State {
	s := &State{DeadEnds: map[string][]DeadEndFact{}, Hypotheses: map[string]*HypothesisFact{}, Lessons: map[string]LessonFact{}}
	// A proposal folds only when it passed the boundary at its own
	// position: the fold is what the duplicate rule and the
	// promotion's citation read, so a raw-pushed proposal must be an
	// anomaly here, never a hypothesis (review finding on the task
	// PR). The table is the shipped one; without it nothing can be
	// judged and every proposal counts an anomaly.
	table, terr := transition.Default()
	for pos, rec := range records {
		e := &rec.Event
		if !keyring.Applies(e.V) {
			continue
		}
		switch e.Verb {
		case DeadEndVerb:
			d, err := ParseDeadEnd(e.Subject, e.Payload)
			if err != nil {
				s.Anomalies++
				continue
			}
			s.DeadEnds[e.Subject] = append(s.DeadEnds[e.Subject], DeadEndFact{Pos: pos, Actor: e.Actor, Fence: d.Fence,
				Tried: d.Tried, Outcome: d.Outcome, Condition: d.Condition, Environment: d.Environment, Pointer: d.Pointer})
		case HypothesisVerb:
			h, err := ParseHypothesis(e.Subject, e.Payload)
			if err != nil {
				s.Anomalies++
				continue
			}
			if _, dup := s.Hypotheses[e.Subject]; dup {
				// A second proposal of one claim is the duplicate the
				// boundary refuses; in history it changes nothing.
				s.Anomalies++
				continue
			}
			if terr != nil {
				s.Anomalies++
				continue
			}
			if _, ok := HypothesisValid(records, table, Citation{Contract: e.Subject, Position: pos}); !ok {
				s.Anomalies++
				continue
			}
			s.Hypotheses[e.Subject] = &HypothesisFact{ID: e.Subject, Pos: pos, Proposer: e.Actor, Claim: h.Claim,
				AppliesWhen: h.AppliesWhen, Support: h.Support, Exceptions: h.Exceptions, Provenance: h.Provenance, Stage: StageProposed}
			s.order = append(s.order, e.Subject)
		case LessonVerb:
			l, err := ParseLesson(e.Subject, e.Payload)
			if err != nil {
				s.Anomalies++
				continue
			}
			fact := LessonFact{Pos: pos, Actor: e.Actor, Lesson: l.Lesson, Hypothesis: l.Hypothesis, PR: l.PR}
			c, _ := ParseCitation(l.Hypothesis)
			h, ok := s.Hypotheses[c.Contract]
			if !ok || h.Pos != c.Position {
				s.Unbound = append(s.Unbound, fact)
				continue
			}
			s.Lessons[l.Lesson] = fact
			if h.Stage != StagePromoted {
				h.Stage = StagePromoted
				p := pos
				h.Lesson = &p
			}
		}
	}
	return s
}

// Any reports whether the prefix carries any curation fact at all, the
// condition under which the report emits its knowledge section.
func (s *State) Any() bool {
	return len(s.DeadEnds) > 0 || len(s.Hypotheses) > 0 || len(s.Lessons) > 0 || len(s.Unbound) > 0 || s.Anomalies > 0
}

// FrontmatterKeys are the fields a lesson file's frontmatter names
// (next/knowledge/lessons/README.md): item 1 lints their presence,
// item 2 their content.
var FrontmatterKeys = []string{"hypothesis", "applies-when", "support", "provenance", "last-validated", "expires"}

// Lint checks a lesson file's frontmatter: a leading YAML block whose
// keys include every FrontmatterKeys entry with a non-empty value.
func Lint(b []byte) error {
	text := strings.ReplaceAll(string(b), "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return errors.New("a lesson opens with a --- frontmatter block naming its hypothesis, applies-when, support, provenance, last-validated and expires")
	}
	rest := text[4:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return errors.New("the frontmatter block is not closed by ---")
	}
	have := map[string]bool{}
	for _, line := range strings.Split(rest[:end], "\n") {
		k, v, ok := strings.Cut(line, ":")
		if ok && strings.TrimSpace(v) != "" && !strings.HasPrefix(line, " ") {
			have[strings.TrimSpace(k)] = true
		}
	}
	var missing []string
	for _, k := range FrontmatterKeys {
		if !have[k] {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("the frontmatter is missing %s", strings.Join(missing, ", "))
	}
	return nil
}
