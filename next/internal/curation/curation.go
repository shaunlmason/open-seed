// Package curation is the staged learning pipeline's ledger half
// (plans/os-f30ee0d3.md, plans/os-96850e5a.md; SEED-NEXT.md §II.12;
// build plan Phase 11 items 1 and 2): the curation facts, their strict
// payload shapes, the hypothesis id derivation, the applies-when
// predicate, the promotion gate's two halves, the surfacing set, and
// the fold that renders the stages.
//
// Trajectories are untrusted inputs, so the stages have distinct
// storage and distinct gates. Observations live on the contract (the
// packet's findings, and deadend.recorded inside the holder's window);
// hypotheses live on their own subject, derived from the claim and its
// exceptions, and only the curator's proposal grant reaches them; a
// contest is the curator's attributable judgment over held-out
// evidence; validated lessons are files merged by PR, and the ledger
// carries the observation of that merge, citing the hypothesis it
// promotes and the adversarial evaluation it survived. No stage
// skips: a promotion that cites no admitted hypothesis folds unbound,
// never as a lesson.
package curation

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/keyring"
	"github.com/shaunlmason/open-seed/next/internal/packet"
	"github.com/shaunlmason/open-seed/next/internal/transition"
)

// The verbs, under the catalog's curation namespace
// (next/spec/protocol.md). Additive growth: no version bump.
const (
	DeadEndVerb    = "curation.deadend.recorded"
	HypothesisVerb = "curation.hypothesis.proposed"
	ContestVerb    = "curation.hypothesis.contested"
	LessonVerb     = "curation.lesson.promoted"
)

// LessonsDir is the validated-lessons store: files merged by PR, one
// per lesson, whose frontmatter Lint checks (next/knowledge/lessons).
const LessonsDir = "next/knowledge/lessons"

// SupportMinimum is the structural floor a hypothesis must cite: at
// least this many admitted observations, on at least this many
// distinct non-failed contracts, and, where the family allows it, from
// at least this many distinct holders. A single run can never be
// promoted from, by construction.
const SupportMinimum = 2

// Carriers is where a promoted lesson lands (plans/os-96850e5a.md D4).
// Every carrier is behavior-changing, because promotion is what puts
// the lesson in front of a worker at claim time, so none is exempt
// from the adversarial evaluation.
var Carriers = []string{"knowledge", "role", "skill", "workflow", "harness"}

// IsCarrier reports whether the name is a carrier.
func IsCarrier(c string) bool {
	for _, k := range Carriers {
		if k == c {
			return true
		}
	}
	return false
}

// hypothesisIDRE is the subject shape a hypothesis lives on.
var hypothesisIDRE = regexp.MustCompile(`^h-[0-9a-f]{12}$`)

// anchorRE is the classification lint's commit-anchor form: a path in
// a filename alphabet anchored to a commit.
var anchorRE = regexp.MustCompile(`^[A-Za-z0-9._/-]{1,200} @ [0-9a-f]{7,64}$`)

// digestRE is a lowercase-hex sha256.
var digestRE = regexp.MustCompile(`^[0-9a-f]{64}$`)

// The gate registry (plans/os-96850e5a.md D4): every refusal of the
// proposal, contest and promotion rules and of the lint's file half is
// a GateError naming a gate registered here, and the constructor
// refuses an unregistered name. The registry is the one authority the
// poisoning drill enumerates to derive its coverage, so a gate added
// here without a poison there is a red test rather than a silent gap.
var gates = map[string]string{}

func register(name, what string) string {
	gates[name] = what
	return name
}

// The gates, by name: <rule>.<part>.
var (
	GateDeadEndShape        = register("deadend.shape", "the dead end's payload shape and fields")
	GateDeadEndHolder       = register("deadend.holder", "the dead end is the window holder's own")
	GateProposalShape       = register("proposal.shape", "the proposal's payload shape and fields")
	GateProposalSubject     = register("proposal.subject", "the subject is derived from the claim and its exceptions")
	GateAppliesWhen         = register("applies_when.shape", "the predicate names at least one record-derivable field, each well-formed")
	GateSupportDuplicate    = register("support.duplicate", "one claim is proposed once")
	GateSupportObservation  = register("support.observation", "every citation is an admitted observation")
	GateSupportFailed       = register("support.failed", "no cited contract stands failed")
	GateSupportFloor        = register("support.floor", "at least two observations on two distinct contracts")
	GateSupportActors       = register("support.actors", "two distinct holders where the family allows it")
	GateContestShape        = register("contest.shape", "the contest's payload shape and fields")
	GateContestHypothesis   = register("contest.hypothesis", "the contest cites an admitted hypothesis on its own subject")
	GateContestEvidence     = register("contest.evidence", "every evidence citation is an admitted observation")
	GateContestSelected     = register("contest.selected", "evidence lies on a contract the applies-when selects")
	GateContestHeldOut      = register("contest.held_out", "evidence lies outside the support set")
	GatePromotionShape      = register("promotion.shape", "the promotion's payload shape and fields")
	GatePromotionHypothesis = register("promotion.hypothesis", "the promotion cites an admitted hypothesis on its own subject")
	GatePromotionContested  = register("promotion.contested", "a contested hypothesis is not promotable")
	GatePromotionCarrier    = register("promotion.carrier", "the carrier is a member")
	GatePromotionStamps     = register("promotion.stamps", "last_validated and expires are RFC3339 and ordered")
	GatePromotionDigest     = register("promotion.digest", "the digest is the lesson file's sha256")
	GatePromotionAdversary  = register("promotion.adversarial", "the adversarial evaluation is an authenticated pass bound to this hypothesis and this lesson anchor, filed after the hypothesis")
	GatePromotionSupport    = register("promotion.support", "the hypothesis's support still satisfies the arms at promotion")
	GateLintFrontmatter     = register("lint.frontmatter", "the lesson file opens with the frontmatter block and its keys")
	GateLintHypothesis      = register("lint.hypothesis", "the frontmatter cites the fact's hypothesis")
	GateLintAppliesWhen     = register("lint.applies_when", "the frontmatter's applies-when parses and equals the hypothesis's")
	GateLintSupport         = register("lint.support", "the frontmatter's support equals the hypothesis's")
	GateLintProvenance      = register("lint.provenance", "every provenance anchor resolves in the repository at its commit")
	GateLintDigest          = register("lint.digest", "the file's bytes at the anchor hash to the fact's digest")
	GateLintAncestry        = register("lint.ancestry", "the anchor commit is an ancestor of the repository's head")
	GateLintStamps          = register("lint.stamps", "last-validated is not after the declared instant, expires is after it, and both equal the fact's")
	GateLintCarrier         = register("lint.carrier", "the frontmatter's carrier equals the fact's")
)

// Gates lists every registered gate, sorted.
func Gates() []string {
	out := make([]string, 0, len(gates))
	for g := range gates {
		out = append(out, g)
	}
	sort.Strings(out)
	return out
}

// GateDescription is what a gate checks.
func GateDescription(gate string) (string, bool) {
	d, ok := gates[gate]
	return d, ok
}

// GateError is a curation refusal naming the gate it failed.
type GateError struct {
	Gate, Verb, Subject, Reason string
}

func (e *GateError) Error() string {
	return fmt.Sprintf("%s on %s refused at gate %s: %s", e.Verb, e.Subject, e.Gate, e.Reason)
}

// NewGateError constructs a refusal at a registered gate; an
// unregistered name is refused as a plain error, never a GateError,
// so the poisoning drill's coverage cannot be widened by a typo.
func NewGateError(gate, verb, subject, reason string) error {
	if _, ok := gates[gate]; !ok {
		return fmt.Errorf("%s on %s: refusal names gate %q, which is not registered: register the gate beside the rule that raises it", verb, subject, gate)
	}
	return &GateError{Gate: gate, Verb: verb, Subject: subject, Reason: reason}
}

// HypothesisID derives the subject a claim is proposed on: h- and the
// first twelve hex digits of the SHA-256 of the canonical claim text
// (whitespace-trimmed, single-spaced) and the canonical exceptions,
// sorted. One claim with one exception set derives one subject, so a
// re-proposal that changes nothing refuses as a duplicate at the
// boundary, and one that adds an exception (the road out of a
// contest) is a new subject.
func HypothesisID(claim string, exceptions []string) string {
	parts := []string{canonical(claim)}
	ex := make([]string, 0, len(exceptions))
	for _, e := range exceptions {
		ex = append(ex, canonical(e))
	}
	sort.Strings(ex)
	parts = append(parts, ex...)
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return "h-" + hex.EncodeToString(sum[:])[:12]
}

func canonical(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

// IsHypothesisSubject reports whether the subject has a hypothesis id's
// shape.
func IsHypothesisSubject(subject string) bool { return hypothesisIDRE.MatchString(subject) }

func strict(raw []byte, into any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(into); err != nil {
		return err
	}
	if dec.More() {
		return errors.New("trailing data after the payload object")
	}
	return nil
}

// AppliesWhen is the predicate a hypothesis holds under
// (plans/os-96850e5a.md D1): at least one of routing (equal to the
// subject's intent routing), tier (equal to the subject's tier) and
// paths (one entry prefixes the subject's acceptance ref path);
// present fields conjoin. Record-derivable, so a claim-time match is
// evaluated from the fold alone.
type AppliesWhen struct {
	Routing string   `json:"routing,omitempty"`
	Tier    string   `json:"tier,omitempty"`
	Paths   []string `json:"paths,omitempty"`
}

// ParseAppliesWhen decodes the predicate strictly: an empty object, an
// unknown field, an empty paths list, an empty string or a non-string
// value refuses naming the part.
func ParseAppliesWhen(raw json.RawMessage) (AppliesWhen, error) {
	var a struct {
		Routing *string   `json:"routing"`
		Tier    *string   `json:"tier"`
		Paths   *[]string `json:"paths"`
	}
	if len(raw) == 0 {
		return AppliesWhen{}, errors.New("applies_when is absent")
	}
	if err := strict(raw, &a); err != nil {
		return AppliesWhen{}, fmt.Errorf("applies_when is the strict object {routing?, tier?, paths?}: %v", err)
	}
	var out AppliesWhen
	if a.Routing != nil {
		if strings.TrimSpace(*a.Routing) == "" {
			return AppliesWhen{}, errors.New("applies_when.routing is empty")
		}
		out.Routing = *a.Routing
	}
	if a.Tier != nil {
		if strings.TrimSpace(*a.Tier) == "" {
			return AppliesWhen{}, errors.New("applies_when.tier is empty")
		}
		out.Tier = *a.Tier
	}
	if a.Paths != nil {
		if len(*a.Paths) == 0 {
			return AppliesWhen{}, errors.New("applies_when.paths is empty: a present list names at least one path prefix")
		}
		for _, p := range *a.Paths {
			if strings.TrimSpace(p) == "" {
				return AppliesWhen{}, errors.New("applies_when.paths holds an empty entry")
			}
		}
		out.Paths = *a.Paths
	}
	if out.Routing == "" && out.Tier == "" && out.Paths == nil {
		return AppliesWhen{}, errors.New("applies_when names no field: at least one of routing, tier and paths")
	}
	return out, nil
}

// Selects reports whether the predicate holds for the subject: every
// present field matches, and an absent field matches nothing on its
// own.
func (a AppliesWhen) Selects(s transition.SubjectState) bool {
	if a.Routing == "" && a.Tier == "" && a.Paths == nil {
		return false
	}
	if a.Routing != "" && s.Routing != a.Routing {
		return false
	}
	if a.Tier != "" && s.Tier != a.Tier {
		return false
	}
	if a.Paths != nil {
		if s.Acceptance == nil {
			return false
		}
		path := AcceptancePath(s.Acceptance.Ref)
		hit := false
		for _, p := range a.Paths {
			if strings.HasPrefix(path, p) {
				hit = true
				break
			}
		}
		if !hit {
			return false
		}
	}
	return true
}

// Equal compares two predicates field for field.
func (a AppliesWhen) Equal(b AppliesWhen) bool {
	return a.Routing == b.Routing && a.Tier == b.Tier && strings.Join(a.Paths, "\x00") == strings.Join(b.Paths, "\x00")
}

// AcceptancePath is the path half of an anchored acceptance ref.
func AcceptancePath(ref string) string {
	path, _, _ := strings.Cut(ref, " @ ")
	return strings.TrimSpace(path)
}

// AnchorParts splits "<path> @ <commit>".
func AnchorParts(anchor string) (path, commit string, ok bool) {
	path, commit, ok = strings.Cut(anchor, " @ ")
	return strings.TrimSpace(path), strings.TrimSpace(commit), ok && anchorRE.MatchString(anchor)
}

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

// ParseDeadEnd decodes and shape-checks a dead end: every field
// non-empty, the fence a decimal position, the pointer (if any) an
// anchored path.
func ParseDeadEnd(subject string, raw []byte) (*DeadEnd, error) {
	var d DeadEnd
	if err := strict(raw, &d); err != nil {
		return nil, NewGateError(GateDeadEndShape, DeadEndVerb, subject, "the payload is the strict object {fence, tried, outcome, condition, environment, pointer?}: "+err.Error())
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
		return nil, NewGateError(GateDeadEndShape, DeadEndVerb, subject, fmt.Sprintf("fence %q is not a position", d.Fence))
	}
	if d.Pointer != "" && !anchorRE.MatchString(d.Pointer) {
		return nil, NewGateError(GateDeadEndShape, DeadEndVerb, subject, fmt.Sprintf("pointer %q is not an anchored path (\"<path> @ <commit>\"): a bare path assumes a shared filesystem disposable executors do not have", d.Pointer))
	}
	return &d, nil
}

// Citation is one support or evidence entry, "<contract>@<position>".
type Citation struct {
	Contract string
	Position int
}

func (c Citation) String() string { return fmt.Sprintf("%s@%d", c.Contract, c.Position) }

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

// Hypothesis is hypothesis.proposed's payload.
type Hypothesis struct {
	Claim       string          `json:"claim"`
	AppliesWhen json.RawMessage `json:"applies_when"`
	Support     []string        `json:"support"`
	Exceptions  []string        `json:"exceptions"`
	Provenance  []string        `json:"provenance"`
	// Applies is the parsed predicate.
	Applies AppliesWhen `json:"-"`
}

// ParseHypothesis decodes and shape-checks a proposal: the claim
// non-empty, the predicate well-formed, the subject derived from the
// claim and its exceptions, support entries well-formed citations,
// provenance entries anchored paths. Lists may be empty here; the
// support floor is the boundary's rule, judged against the record.
func ParseHypothesis(subject string, raw []byte) (*Hypothesis, error) {
	var h Hypothesis
	if err := strict(raw, &h); err != nil {
		return nil, NewGateError(GateProposalShape, HypothesisVerb, subject, "the payload is the strict object {claim, applies_when, support, exceptions, provenance}: "+err.Error())
	}
	var missing []string
	if strings.TrimSpace(h.Claim) == "" {
		missing = append(missing, "claim")
	}
	if len(h.AppliesWhen) == 0 {
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
	applies, err := ParseAppliesWhen(h.AppliesWhen)
	if err != nil {
		return nil, NewGateError(GateAppliesWhen, HypothesisVerb, subject, err.Error())
	}
	h.Applies = applies
	if want := HypothesisID(h.Claim, h.Exceptions); subject != want {
		return nil, NewGateError(GateProposalSubject, HypothesisVerb, subject, fmt.Sprintf("the subject is not derived from the claim and its exceptions: they derive %s", want))
	}
	for _, s := range h.Support {
		if _, ok := ParseCitation(s); !ok {
			return nil, NewGateError(GateProposalShape, HypothesisVerb, subject, fmt.Sprintf("support entry %q is not \"<contract>@<position>\"", s))
		}
	}
	for _, p := range h.Provenance {
		if !anchorRE.MatchString(p) {
			return nil, NewGateError(GateProposalShape, HypothesisVerb, subject, fmt.Sprintf("provenance entry %q is not an anchored path (\"<path> @ <commit>\")", p))
		}
	}
	return &h, nil
}

// Contest is hypothesis.contested's payload (plans/os-96850e5a.md D3):
// held-out evidence against an admitted hypothesis, the curator's
// attributable judgment, never an average.
type Contest struct {
	Hypothesis string   `json:"hypothesis"`
	Evidence   []string `json:"evidence"`
	Reason     string   `json:"reason"`
}

// ParseContest decodes and shape-checks a contest: the cited
// hypothesis this subject, at least one well-formed evidence citation,
// a reason.
func ParseContest(subject string, raw []byte) (*Contest, error) {
	var c Contest
	if err := strict(raw, &c); err != nil {
		return nil, NewGateError(GateContestShape, ContestVerb, subject, "the payload is the strict object {hypothesis, evidence, reason}: "+err.Error())
	}
	var missing []string
	if strings.TrimSpace(c.Hypothesis) == "" {
		missing = append(missing, "hypothesis")
	}
	if len(c.Evidence) == 0 {
		missing = append(missing, "evidence")
	}
	if strings.TrimSpace(c.Reason) == "" {
		missing = append(missing, "reason")
	}
	if len(missing) > 0 {
		return nil, &transition.IncompleteError{Verb: ContestVerb, Subject: subject, Missing: missing}
	}
	cit, ok := ParseCitation(c.Hypothesis)
	if !ok || !IsHypothesisSubject(cit.Contract) {
		return nil, NewGateError(GateContestShape, ContestVerb, subject, fmt.Sprintf("hypothesis %q is not \"<h-id>@<position>\"", c.Hypothesis))
	}
	if cit.Contract != subject {
		return nil, NewGateError(GateContestShape, ContestVerb, subject, fmt.Sprintf("the cited hypothesis %s is not this subject: a contest is a fact on the hypothesis it contests", cit.Contract))
	}
	for _, e := range c.Evidence {
		if _, ok := ParseCitation(e); !ok {
			return nil, NewGateError(GateContestShape, ContestVerb, subject, fmt.Sprintf("evidence entry %q is not \"<contract>@<position>\"", e))
		}
	}
	return &c, nil
}

// Adversarial is the evaluation a promotion cites: the eval's name and
// the position of its authenticated pass.
type Adversarial struct {
	Eval    string `json:"eval"`
	Verdict string `json:"verdict"`
}

// Lesson is lesson.promoted's payload: the observation that a lesson
// file landed by PR, citing the hypothesis it promotes and the
// evaluation it survived, carrying what the frontmatter says so that
// expiry is derivable from the fold without the file.
type Lesson struct {
	Lesson        string       `json:"lesson"`
	Hypothesis    string       `json:"hypothesis"`
	PR            string       `json:"pr"`
	Carrier       string       `json:"carrier"`
	Adversarial   *Adversarial `json:"adversarial"`
	LastValidated string       `json:"last_validated"`
	Expires       string       `json:"expires"`
	Digest        string       `json:"digest"`
}

// ParseLesson decodes and shape-checks a promotion: anchored lesson
// path under the store, a hypothesis citation whose subject is the
// event's, an anchored PR, a member carrier, the adversarial citation,
// RFC3339 stamps in order, a sha256 digest.
func ParseLesson(subject string, raw []byte) (*Lesson, error) {
	var l Lesson
	if err := strict(raw, &l); err != nil {
		return nil, NewGateError(GatePromotionShape, LessonVerb, subject, "the payload is the strict object {lesson, hypothesis, pr, carrier, adversarial{eval, verdict}, last_validated, expires, digest}: "+err.Error())
	}
	var missing []string
	for _, f := range []struct{ name, v string }{{"lesson", l.Lesson}, {"hypothesis", l.Hypothesis}, {"pr", l.PR}, {"carrier", l.Carrier}, {"last_validated", l.LastValidated}, {"expires", l.Expires}, {"digest", l.Digest}} {
		if strings.TrimSpace(f.v) == "" {
			missing = append(missing, f.name)
		}
	}
	if l.Adversarial == nil {
		missing = append(missing, "adversarial")
	}
	if len(missing) > 0 {
		return nil, &transition.IncompleteError{Verb: LessonVerb, Subject: subject, Missing: missing}
	}
	if !anchorRE.MatchString(l.Lesson) {
		return nil, NewGateError(GatePromotionShape, LessonVerb, subject, fmt.Sprintf("lesson %q is not an anchored path (\"<path> @ <commit>\")", l.Lesson))
	}
	if !UnderLessonsDir(l.Lesson) {
		return nil, NewGateError(GatePromotionShape, LessonVerb, subject, fmt.Sprintf("lesson %q is not under %s/: the validated store is one place, and a path that climbs out of it names a file the store's lint never sees", l.Lesson, LessonsDir))
	}
	if !anchorRE.MatchString(l.PR) {
		return nil, NewGateError(GatePromotionShape, LessonVerb, subject, fmt.Sprintf("pr %q is not \"<pr> @ <merged-commit>\"", l.PR))
	}
	c, ok := ParseCitation(l.Hypothesis)
	if !ok || !IsHypothesisSubject(c.Contract) {
		return nil, NewGateError(GatePromotionShape, LessonVerb, subject, fmt.Sprintf("hypothesis %q is not \"<h-id>@<position>\"", l.Hypothesis))
	}
	if c.Contract != subject {
		return nil, NewGateError(GatePromotionHypothesis, LessonVerb, subject, fmt.Sprintf("the cited hypothesis %s is not this subject: a promotion is a fact on the hypothesis it promotes", c.Contract))
	}
	if !IsCarrier(l.Carrier) {
		return nil, NewGateError(GatePromotionCarrier, LessonVerb, subject, fmt.Sprintf("carrier %q is not one of %s", l.Carrier, strings.Join(Carriers, ", ")))
	}
	if strings.TrimSpace(l.Adversarial.Eval) == "" || strings.TrimSpace(l.Adversarial.Verdict) == "" {
		return nil, NewGateError(GatePromotionAdversary, LessonVerb, subject, "adversarial names the eval and the position of its pass: every carrier is behavior-changing, so none is exempt")
	}
	if _, err := strconv.Atoi(l.Adversarial.Verdict); err != nil {
		return nil, NewGateError(GatePromotionAdversary, LessonVerb, subject, fmt.Sprintf("adversarial.verdict %q is not a position", l.Adversarial.Verdict))
	}
	if _, _, err := StampsOrdered(l.LastValidated, l.Expires); err != nil {
		return nil, NewGateError(GatePromotionStamps, LessonVerb, subject, err.Error())
	}
	if !digestRE.MatchString(l.Digest) {
		return nil, NewGateError(GatePromotionDigest, LessonVerb, subject, fmt.Sprintf("digest %q is not a lowercase-hex sha256 of the lesson file's bytes", l.Digest))
	}
	return &l, nil
}

// UnderLessonsDir reports whether the anchor's path lies inside the
// lessons store: the path is clean (no `.` or `..` segment, no doubled
// or trailing separator, so a lexical prefix cannot be climbed out
// of), relative, and prefixed by the store. A path such as
// `next/knowledge/lessons/../x` matches the anchor grammar and
// resolves outside the store, which is what the promotion must never
// name (review finding on the item 1 PR).
func UnderLessonsDir(anchor string) bool {
	p, _, _ := strings.Cut(anchor, " @ ")
	p = strings.TrimSpace(p)
	if p == "" || path.Clean(p) != p || strings.HasPrefix(p, "/") {
		return false
	}
	return strings.HasPrefix(p, LessonsDir+"/")
}

// StampsOrdered parses the two RFC3339 stamps and requires expires
// after last_validated.
func StampsOrdered(lastValidated, expires string) (time.Time, time.Time, error) {
	lv, err := time.Parse(time.RFC3339, lastValidated)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("last_validated %q is not RFC3339", lastValidated)
	}
	ex, err := time.Parse(time.RFC3339, expires)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("expires %q is not RFC3339", expires)
	}
	if !ex.After(lv) {
		return time.Time{}, time.Time{}, fmt.Errorf("expires %s is not after last_validated %s", expires, lastValidated)
	}
	return lv, ex, nil
}

// Observation is one admitted stage-one fact: a packet finding's
// deliberate exit, or a dead end, on a contract, at a position, by an
// actor, in the window of a holder.
type Observation struct {
	Contract string
	Position int
	Verb     string
	Actor    string
	Holder   string
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
	return &Observation{Contract: e.Subject, Position: c.Position, Verb: e.Verb, Actor: e.Actor, Holder: s.Claim.Holder}, true
}

// WindowAdmitted re-judges the claim window the fold reports: the
// claim.taken at the fence is on the subject, signed by the holder,
// and the holder held a capability the claim accepts at that prefix.
// The lifecycle fold applies a legal transition whoever signed it, so
// a raw-pushed claim by a grantless key opens an apparent window;
// what looks like a holder's observation inside it passed no boundary
// (review finding on the item 1 PR).
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
// (review finding on the item 1 PR). A failed trajectory supports
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

// Family is the set of contracts the applies-when selects in the
// record, with every holder they had over the chain (the current
// holder and every prior claimant, both record facts). The actor arm
// applies where the family has two or more holders: a family with one
// holder cannot supply two, and says so.
type Family struct {
	Contracts []string
	Holders   map[string]bool
}

// FamilyOf computes the family from the selection, never from the
// support set.
func FamilyOf(fold *transition.Fold, applies AppliesWhen) Family {
	fam := Family{Holders: map[string]bool{}}
	for _, id := range fold.Subjects() {
		s, ok := fold.State(id)
		if !ok || !applies.Selects(s) {
			continue
		}
		fam.Contracts = append(fam.Contracts, id)
		if s.Claim != nil {
			fam.Holders[s.Claim.Holder] = true
		}
		for who := range s.PriorClaimants {
			fam.Holders[who] = true
		}
	}
	return fam
}

// Support is what CheckSupport establishes: the observations, their
// distinct holders, and whether the actor arm was waived because the
// family has one holder.
type Support struct {
	Observations      []Observation
	Holders           []string
	SingleActorFamily bool
}

// CheckSupport judges a proposal's citations against the record: each
// an admitted observation, no duplicate, at least SupportMinimum on at
// least SupportMinimum distinct contracts, none failed, and, where the
// family the predicate selects has SupportMinimum or more holders,
// from SupportMinimum or more distinct holders.
func CheckSupport(records []*event.Record, table *transition.Table, fold *transition.Fold, subject string, h *Hypothesis) (*Support, error) {
	seen := map[string]bool{}
	contracts := map[string]bool{}
	holders := map[string]bool{}
	sup := &Support{}
	for _, raw := range h.Support {
		c, _ := ParseCitation(raw)
		if seen[raw] {
			return nil, NewGateError(GateSupportObservation, HypothesisVerb, subject, fmt.Sprintf("support cites %s twice", raw))
		}
		seen[raw] = true
		o, ok := ObservationAt(records, table, c)
		if !ok {
			return nil, NewGateError(GateSupportObservation, HypothesisVerb, subject, fmt.Sprintf("support entry %s is not an admitted observation (a deliberate exit with findings or a dead end, inside the holder's window)", raw))
		}
		if FailedAt(records, table, c.Contract) {
			return nil, NewGateError(GateSupportFailed, HypothesisVerb, subject, fmt.Sprintf("support entry %s cites a failed contract: a failed trajectory supports nothing", raw))
		}
		contracts[c.Contract] = true
		holders[o.Holder] = true
		sup.Observations = append(sup.Observations, *o)
	}
	if len(sup.Observations) < SupportMinimum || len(contracts) < SupportMinimum {
		return nil, NewGateError(GateSupportFloor, HypothesisVerb, subject, fmt.Sprintf("support cites %d observation(s) on %d contract(s); a hypothesis needs at least %d admitted observations on %d distinct non-failed contracts, so no single run is promotable by construction", len(sup.Observations), len(contracts), SupportMinimum, SupportMinimum))
	}
	for who := range holders {
		sup.Holders = append(sup.Holders, who)
	}
	sort.Strings(sup.Holders)
	fam := FamilyOf(fold, h.Applies)
	if len(fam.Holders) >= SupportMinimum {
		if len(sup.Holders) < SupportMinimum {
			return nil, NewGateError(GateSupportActors, HypothesisVerb, subject, fmt.Sprintf("the family the applies-when selects has %d holders and the support comes from %d: where the family allows it, support comes from at least %d distinct holders", len(fam.Holders), len(sup.Holders), SupportMinimum))
		}
	} else {
		sup.SingleActorFamily = true
	}
	return sup, nil
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
// the item 1 PR). The earliest proposal that passes the grant and the
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

// CheckContest judges a contest against the record as the boundary
// does: the cited hypothesis admitted on this subject, every evidence
// citation an admitted observation on a contract the applies-when
// selects, outside the support set. The fold re-judges every contest
// through this same function at its own position (review finding on
// the task PR), so a raw-pushed contest by a key holding no curate, or
// one citing fabricated or support-set evidence, moves nothing to
// contested.
func CheckContest(records []*event.Record, table *transition.Table, fold *transition.Fold, subject string, ct *Contest) (*Hypothesis, error) {
	cit, _ := ParseCitation(ct.Hypothesis)
	h, ok := HypothesisValid(records, table, cit)
	if !ok {
		return nil, NewGateError(GateContestHypothesis, ContestVerb, subject, fmt.Sprintf("%s is not an admitted hypothesis: a contest judges a proposal that passed the boundary", ct.Hypothesis))
	}
	inSupport := map[string]bool{}
	for _, s := range h.Support {
		inSupport[s] = true
	}
	for _, raw := range ct.Evidence {
		ec, _ := ParseCitation(raw)
		if _, ok := ObservationAt(records, table, ec); !ok {
			return nil, NewGateError(GateContestEvidence, ContestVerb, subject, fmt.Sprintf("evidence %s is not an admitted observation", raw))
		}
		st, ok := fold.State(ec.Contract)
		if !ok || !h.Applies.Selects(st) {
			return nil, NewGateError(GateContestSelected, ContestVerb, subject, fmt.Sprintf("evidence %s lies on a contract the applies-when does not select: counter-evidence is evidence the claim was supposed to hold over", raw))
		}
		if inSupport[raw] {
			return nil, NewGateError(GateContestHeldOut, ContestVerb, subject, fmt.Sprintf("evidence %s is in the support set: a contest cites held-out evidence, never what the proposal already rested on", raw))
		}
	}
	return h, nil
}

// ContestedBefore reports whether an admitted contest on the subject
// stands before the position: one position-accurate scan, never a
// refold. CheckPromotion asks this from inside the fold's own
// promotion replay, and a refold from there would re-enter every
// earlier promotion's replay in turn, exponential in the promotions a
// chain carries (review finding on the item 3 PR).
func ContestedBefore(records []*event.Record, table *transition.Table, subject string, before int) bool {
	if before > len(records) {
		before = len(records)
	}
	for pos := 0; pos < before; pos++ {
		e := &records[pos].Event
		if e.Verb != ContestVerb || e.Subject != subject {
			continue
		}
		if _, ok := ContestValid(records, table, pos); ok {
			return true
		}
	}
	return false
}

// ContestValid re-judges the record at the position as an admitted
// contest: a contest on the subject whose signer held curate at that
// prefix and whose citations pass CheckContest there.
func ContestValid(records []*event.Record, table *transition.Table, pos int) (*Contest, bool) {
	if table == nil || pos < 0 || pos >= len(records) {
		return nil, false
	}
	e := &records[pos].Event
	if e.Verb != ContestVerb || !keyring.Applies(e.V) {
		return nil, false
	}
	ct, err := ParseContest(e.Subject, e.Payload)
	if err != nil {
		return nil, false
	}
	prefix := records[:pos]
	ring, _, err := keyring.StateAt(prefix)
	if err != nil || ring == nil || !ring.HasAnyCapability(e.Actor, keyring.AcceptedCapabilities(ContestVerb)) {
		return nil, false
	}
	if _, err := CheckContest(prefix, table, table.FoldRecords(prefix), e.Subject, ct); err != nil {
		return nil, false
	}
	return ct, true
}

// CheckPromotion judges a promotion's ledger half against the record
// as the boundary does (plans/os-96850e5a.md D4, D5): an admitted,
// uncontested hypothesis on this subject whose support still satisfies
// the arms, and an adversarial evaluation that is an authenticated
// pass, replayed at its own position, on an eval filed after the
// hypothesis and bound to it and to this lesson anchor. The fold
// re-judges every promotion through this same function at its own
// position (review finding on the task PR), so a promotion raw-pushed
// past the boundary binds nothing and surfaces nowhere.
func CheckPromotion(records []*event.Record, table *transition.Table, fold *transition.Fold, subject string, l *Lesson) error {
	cit, _ := ParseCitation(l.Hypothesis)
	h, ok := HypothesisValid(records, table, cit)
	if !ok {
		return NewGateError(GatePromotionHypothesis, LessonVerb, subject, fmt.Sprintf("%s is not an admitted hypothesis: a lesson promotes a proposal that passed the boundary, and no stage skips", l.Hypothesis))
	}
	if ContestedBefore(records, table, subject, len(records)) {
		return NewGateError(GatePromotionContested, LessonVerb, subject, "the hypothesis stands contested: a contested hypothesis is never promoted or averaged back; a new proposal citing the counter-evidence as an exception is the road out")
	}
	if _, err := CheckSupport(records, table, fold, subject, h); err != nil {
		return NewGateError(GatePromotionSupport, LessonVerb, subject, "the hypothesis's support no longer satisfies the arms at the tip: "+err.Error())
	}
	return adversarialSurvived(records, table, fold, subject, cit.Position, l)
}

// adversarialSurvived is the promotion's adversarial arm: the cited
// verdict is the authenticated pass of an eval whose marker binds this
// hypothesis and this lesson anchor, filed after the hypothesis.
func adversarialSurvived(records []*event.Record, table *transition.Table, fold *transition.Fold, subject string, hypothesisPos int, l *Lesson) error {
	refuse := func(reason string) error {
		return NewGateError(GatePromotionAdversary, LessonVerb, subject, reason)
	}
	pos, _ := strconv.Atoi(l.Adversarial.Verdict)
	if pos < 0 || pos >= len(records) {
		return refuse(fmt.Sprintf("the cited verdict position %d is not on the chain", pos))
	}
	evalSubject := records[pos].Event.Subject
	es, ok := fold.State(evalSubject)
	if !ok || es.Eval == nil {
		return refuse(fmt.Sprintf("position %d is not a verdict on an eval: survival is proven by a constructed counter-trajectory, filed with the eval marker", pos))
	}
	if es.Eval.Name != l.Adversarial.Eval {
		return refuse(fmt.Sprintf("the verdict at position %d judges eval %q, the promotion names %q", pos, es.Eval.Name, l.Adversarial.Eval))
	}
	if !EvalBound(es) {
		return refuse(fmt.Sprintf("the contract names eval %q but its acceptance spec is not that definition's fixture at a gated revision", es.Eval.Name))
	}
	if es.Eval.Lesson != l.Hypothesis || es.Eval.Carrier != l.Lesson {
		return refuse(fmt.Sprintf("the eval's marker binds lesson %q and carrier %q, the promotion is for %q at %q: a pass on an eval filed for another hypothesis, another revision, or nothing in particular is not survival", es.Eval.Lesson, es.Eval.Carrier, l.Hypothesis, l.Lesson))
	}
	filed := -1
	for i, rec := range records {
		if rec.Event.Verb == "intent.filed" && rec.Event.Subject == evalSubject {
			filed = i
			break
		}
	}
	if filed <= hypothesisPos {
		return refuse(fmt.Sprintf("the eval was filed at position %d, before the hypothesis at %d: a counter-trajectory is constructed against the candidate, never before it existed", filed, hypothesisPos))
	}
	if fact := AuthenticPass(records, table, evalSubject, es); fact == nil || fact.Pos != pos {
		return refuse(fmt.Sprintf("position %d is not the eval's authenticated pass verdict: survival is a pass rendered by a verdict-granted key disjoint from the implementer, replayed at its own position", pos))
	}
	return nil
}

// EvalBound reports whether the subject is an eval contract bound to
// its definition: the marker names an eval and the acceptance is that
// definition's fixture, executable, at a gated revision.
func EvalBound(s transition.SubjectState) bool {
	return s.Eval != nil && s.Acceptance != nil && s.Acceptance.Executable && s.Acceptance.Gated &&
		strings.HasPrefix(s.Acceptance.Ref, transition.EvalFixturePrefix(s.Eval.Name))
}

// PassLevelCheck is the level half of a pass's authentication (the
// verifier boundary's rule from seed/4, plans/os-99829835.md D3): the
// verdict's recorded independence equals the level the record
// supports and satisfies the tier. The rule lives in internal/admit
// beside LevelAchieved, the one implementation the verdict rule, the
// merge chain, render and reconcile share, and admit installs it here
// at init so the fold's promotion replay applies it too; nil means no
// level rule is installed, which no seed binary runs with (the drill
// in internal/admit pins the installation).
var PassLevelCheck func(records []*event.Record, table *transition.Table, subject string, s transition.SubjectState, fact transition.VerdictFact) bool

// AuthenticPass is the subject's latest verdict when it is a pass
// rendered by a key that held a verdict grant at the verdict's own
// position, disjoint from the implementer (the verifier boundary's L1
// rule, replayed to the position as FailedAt replays it), and, where
// the level vocabulary applies, at the level the record supports; nil
// otherwise.
func AuthenticPass(records []*event.Record, table *transition.Table, subject string, s transition.SubjectState) *transition.VerdictFact {
	if s.Verdict == nil || s.Verdict.Verdict != "pass" || s.Verdict.Pos < 0 || s.Verdict.Pos > len(records) {
		return nil
	}
	ring, _, err := keyring.StateAt(records[:s.Verdict.Pos])
	if err != nil || ring == nil || !ring.HasAnyCapability(s.Verdict.Signer, keyring.AcceptedCapabilities(transition.VerdictRenderedVerb)) {
		return nil
	}
	if s.PriorClaimants[s.Verdict.Signer] || (s.Submission != nil && s.Verdict.Signer == s.Submission.Signer) {
		return nil
	}
	if PassLevelCheck != nil && !PassLevelCheck(records, table, subject, s, *s.Verdict) {
		return nil
	}
	return s.Verdict
}

// PromotionValid re-judges the record at the position as an admitted
// promotion: a promotion on the subject whose signer held a capability
// the promotion accepts at that prefix and whose citations pass
// CheckPromotion there.
func PromotionValid(records []*event.Record, table *transition.Table, pos int) (*Lesson, bool) {
	if table == nil || pos < 0 || pos >= len(records) {
		return nil, false
	}
	e := &records[pos].Event
	if e.Verb != LessonVerb || !keyring.Applies(e.V) {
		return nil, false
	}
	l, err := ParseLesson(e.Subject, e.Payload)
	if err != nil {
		return nil, false
	}
	prefix := records[:pos]
	ring, _, err := keyring.StateAt(prefix)
	if err != nil || ring == nil || !ring.HasAnyCapability(e.Actor, keyring.AcceptedCapabilities(LessonVerb)) {
		return nil, false
	}
	if err := CheckPromotion(prefix, table, table.FoldRecords(prefix), e.Subject, l); err != nil {
		return nil, false
	}
	return l, true
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
	ID          string      `json:"id"`
	Pos         int         `json:"position"`
	Proposer    string      `json:"proposer"`
	Claim       string      `json:"claim"`
	AppliesWhen AppliesWhen `json:"applies_when"`
	Support     []string    `json:"support"`
	Exceptions  []string    `json:"exceptions"`
	Provenance  []string    `json:"provenance"`
	// Stage is proposed, promoted once a lesson cites it, or contested
	// once a contest lands, from either.
	Stage string `json:"stage"`
	// Lesson is the first promotion's position, when promoted.
	Lesson *int `json:"lesson,omitempty"`
	// Contest is the contest's position, when contested.
	Contest *int `json:"contest,omitempty"`
	// SingleActorFamily says the actor arm was waived because the
	// family the predicate selects had one holder at proposal.
	SingleActorFamily bool `json:"single_actor_family,omitempty"`
}

// ContestFact is a folded contest.
type ContestFact struct {
	Pos      int      `json:"position"`
	Actor    string   `json:"actor"`
	Evidence []string `json:"evidence"`
	Reason   string   `json:"reason"`
}

// LessonFact is a folded promotion.
type LessonFact struct {
	Pos           int         `json:"position"`
	Actor         string      `json:"actor"`
	Lesson        string      `json:"lesson"`
	Hypothesis    string      `json:"hypothesis"`
	PR            string      `json:"pr"`
	Carrier       string      `json:"carrier"`
	Adversarial   Adversarial `json:"adversarial"`
	LastValidated string      `json:"last_validated"`
	Expires       string      `json:"expires"`
	Digest        string      `json:"digest"`
}

// The stages a hypothesis renders under.
const (
	StageProposed  = "proposed"
	StagePromoted  = "promoted"
	StageContested = "contested"
)

// State is the curation fold: dead ends by contract, hypotheses by id,
// contests by hypothesis, lessons by path, and the promotions that
// cite no admitted hypothesis, which are unbound rather than lessons.
type State struct {
	DeadEnds   map[string][]DeadEndFact
	Hypotheses map[string]*HypothesisFact
	Contests   map[string][]ContestFact
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

// LessonsOf returns the promotions citing the hypothesis, in position
// order.
func (s *State) LessonsOf(id string) []LessonFact {
	var out []LessonFact
	for _, l := range s.Lessons {
		if c, _ := ParseCitation(l.Hypothesis); c.Contract == id {
			out = append(out, l)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Pos < out[j].Pos })
	return out
}

// Fold renders the curation facts of a verified prefix. Tolerant like
// every fold: a malformed raw fact counts an anomaly and folds nothing;
// facts before the keyring boundary are inert. Whether a fact PASSED
// the boundary (the window, the grant, the support, the gate) is
// admission's question; the fold records what the chain carries, and
// the boundary re-judges citations from the records.
func Fold(records []*event.Record) *State {
	s := &State{DeadEnds: map[string][]DeadEndFact{}, Hypotheses: map[string]*HypothesisFact{},
		Contests: map[string][]ContestFact{}, Lessons: map[string]LessonFact{}}
	// A proposal folds only when it passed the boundary at its own
	// position: the fold is what the duplicate rule, the contest and
	// the promotion's citation read, so a raw-pushed proposal must be
	// an anomaly here, never a hypothesis (review finding on the item
	// 1 PR). The table is the shipped one; without it nothing can be
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
				AppliesWhen: h.Applies, Support: h.Support, Exceptions: h.Exceptions, Provenance: h.Provenance, Stage: StageProposed}
			s.order = append(s.order, e.Subject)
		case ContestVerb:
			c, err := ParseContest(e.Subject, e.Payload)
			if err != nil {
				s.Anomalies++
				continue
			}
			cit, _ := ParseCitation(c.Hypothesis)
			h, ok := s.Hypotheses[cit.Contract]
			if !ok || h.Pos != cit.Position || terr != nil {
				s.Anomalies++
				continue
			}
			// A contest moves the stage only when it passed the
			// boundary at its own position: a raw-pushed contest by a
			// key holding no curate, or citing fabricated or
			// support-set evidence, could otherwise disable a
			// legitimate lesson on every delivery surface (review
			// finding on the task PR).
			if _, ok := ContestValid(records, table, pos); !ok {
				s.Anomalies++
				continue
			}
			s.Contests[e.Subject] = append(s.Contests[e.Subject], ContestFact{Pos: pos, Actor: e.Actor, Evidence: c.Evidence, Reason: c.Reason})
			if h.Stage != StageContested {
				h.Stage = StageContested
				p := pos
				h.Contest = &p
			}
		case LessonVerb:
			l, err := ParseLesson(e.Subject, e.Payload)
			if err != nil {
				s.Anomalies++
				continue
			}
			fact := LessonFact{Pos: pos, Actor: e.Actor, Lesson: l.Lesson, Hypothesis: l.Hypothesis, PR: l.PR, Carrier: l.Carrier,
				Adversarial: *l.Adversarial, LastValidated: l.LastValidated, Expires: l.Expires, Digest: l.Digest}
			c, _ := ParseCitation(l.Hypothesis)
			h, ok := s.Hypotheses[c.Contract]
			if !ok || h.Pos != c.Position {
				s.Unbound = append(s.Unbound, fact)
				continue
			}
			// A promotion binds only when it passed the boundary at
			// its own position: the signer's grant, the uncontested
			// state, the support and the bound adversarial pass are
			// all re-judged there, so a promotion raw-pushed past the
			// boundary folds unbound and surfaces nowhere (review
			// finding on the task PR).
			if terr != nil {
				s.Anomalies++
				continue
			}
			if _, ok := PromotionValid(records, table, pos); !ok {
				s.Unbound = append(s.Unbound, fact)
				continue
			}
			s.Lessons[l.Lesson] = fact
			if h.Stage == StageProposed {
				h.Stage = StagePromoted
				p := pos
				h.Lesson = &p
			}
		}
	}
	return s
}

// SingleActorFamily reports whether the actor arm was waived for the
// hypothesis at its own position: the family the predicate selected
// there had one holder.
func (s *State) SingleActorFamily(records []*event.Record, table *transition.Table, id string) bool {
	h, ok := s.Hypotheses[id]
	if !ok || table == nil {
		return false
	}
	fam := FamilyOf(table.FoldRecords(records[:h.Pos]), h.AppliesWhen)
	return len(fam.Holders) < SupportMinimum
}

// Any reports whether the prefix carries any curation fact at all, the
// condition under which the report emits its knowledge section.
func (s *State) Any() bool {
	return len(s.DeadEnds) > 0 || len(s.Hypotheses) > 0 || len(s.Contests) > 0 || len(s.Lessons) > 0 || len(s.Unbound) > 0 || s.Anomalies > 0
}

// Contested reports whether the hypothesis stands contested.
func (s *State) Contested(id string) bool {
	h, ok := s.Hypotheses[id]
	return ok && h.Stage == StageContested
}

// HeldOut lists the admitted observations on contracts the
// hypothesis's predicate selects that are outside its support set:
// what a contest may cite, and what `seed knowledge validate` lists.
func HeldOut(records []*event.Record, table *transition.Table, fold *transition.Fold, h *HypothesisFact) []Observation {
	inSupport := map[string]bool{}
	for _, c := range h.Support {
		inSupport[c] = true
	}
	var out []Observation
	for pos, rec := range records {
		e := &rec.Event
		st, ok := fold.State(e.Subject)
		if !ok || !h.AppliesWhen.Selects(st) {
			continue
		}
		cit := Citation{Contract: e.Subject, Position: pos}
		if inSupport[cit.String()] {
			continue
		}
		if o, ok := ObservationAt(records, table, cit); ok {
			out = append(out, *o)
		}
	}
	return out
}

// Surfaced is one lesson delivered at claim time: what a worker is
// shown, verified against the repository before anything surfaces.
type Surfaced struct {
	Lesson      string      `json:"lesson"`
	Hypothesis  string      `json:"hypothesis"`
	AppliesWhen AppliesWhen `json:"applies_when"`
	Carrier     string      `json:"carrier"`
	Digest      string      `json:"digest"`
}

// Unresolved is a promotion whose fact does not resolve in the
// repository the reader holds: reported, never surfaced.
type Unresolved struct {
	Lesson     string `json:"lesson"`
	Hypothesis string `json:"hypothesis"`
	Reason     string `json:"reason"`
}

// Candidates lists the promotions that would surface on the subject
// before any repository check: promoted, not contested, the predicate
// selecting the subject. With subject empty every promoted,
// uncontested lesson is a candidate (reconcile's view).
func Candidates(st *State, fold *transition.Fold, subject string) []LessonFact {
	var out []LessonFact
	for _, id := range st.HypothesisIDs() {
		h, _ := st.Hypothesis(id)
		if h.Stage != StagePromoted {
			continue
		}
		if subject != "" {
			s, ok := fold.State(subject)
			if !ok || !h.AppliesWhen.Selects(s) {
				continue
			}
		}
		out = append(out, st.LessonsOf(id)...)
	}
	return out
}

// Verify checks a promotion against the repository: the anchor commit
// is an ancestor of the head (the promotion PR merged) and the file's
// bytes at the anchor hash to the fact's digest.
func Verify(repo string, l LessonFact) error {
	path, commit, ok := AnchorParts(l.Lesson)
	if !ok {
		return errors.New("the lesson anchor is malformed")
	}
	if !gitAncestor(repo, commit, "HEAD") {
		return fmt.Errorf("the anchor commit %s is not an ancestor of the repository's head: the promotion PR has not merged here", commit)
	}
	b, err := gitShow(repo, commit, path)
	if err != nil {
		return fmt.Errorf("the file does not exist at the anchor: %v", err)
	}
	if got := Digest(b); got != l.Digest {
		return fmt.Errorf("the file's bytes at the anchor hash to %s, the fact says %s", got, l.Digest)
	}
	return nil
}

// Surfacing is the delivery set for a subject (plans/os-96850e5a.md
// D6): every candidate whose fact resolves in the repository, and the
// candidates that do not, with the reason. With repo empty nothing
// surfaces and every candidate is unresolved as unverified.
func Surfacing(records []*event.Record, fold *transition.Fold, repo, subject string) ([]Surfaced, []Unresolved) {
	st := Fold(records)
	surfaced, unresolved := []Surfaced{}, []Unresolved{}
	for _, l := range Candidates(st, fold, subject) {
		if repo == "" {
			unresolved = append(unresolved, Unresolved{Lesson: l.Lesson, Hypothesis: l.Hypothesis, Reason: "no repository to verify against"})
			continue
		}
		if err := Verify(repo, l); err != nil {
			unresolved = append(unresolved, Unresolved{Lesson: l.Lesson, Hypothesis: l.Hypothesis, Reason: err.Error()})
			continue
		}
		c, _ := ParseCitation(l.Hypothesis)
		h, _ := st.Hypothesis(c.Contract)
		surfaced = append(surfaced, Surfaced{Lesson: l.Lesson, Hypothesis: l.Hypothesis, AppliesWhen: h.AppliesWhen, Carrier: l.Carrier, Digest: l.Digest})
	}
	return surfaced, unresolved
}

// Digest is the lowercase-hex sha256 of the bytes.
func Digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func gitAncestor(repo, ancestor, descendant string) bool {
	return exec.Command("git", "-C", repo, "merge-base", "--is-ancestor", ancestor, descendant).Run() == nil
}

func gitShow(repo, commit, path string) ([]byte, error) {
	out, err := exec.Command("git", "-C", repo, "show", commit+":"+path).Output()
	if err != nil {
		return nil, err
	}
	return out, nil
}

func gitResolves(repo, commit, path string) bool {
	return exec.Command("git", "-C", repo, "cat-file", "-e", commit+":"+path).Run() == nil
}

// FrontmatterKeys are the fields a lesson file's frontmatter names
// (next/knowledge/lessons/README.md): item 1 lints their presence,
// item 2 their content.
var FrontmatterKeys = []string{"hypothesis", "applies-when", "support", "provenance", "last-validated", "expires", "carrier"}

// Frontmatter parses the leading block into key/value pairs.
func Frontmatter(b []byte) (map[string]string, error) {
	text := strings.ReplaceAll(string(b), "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return nil, errors.New("a lesson opens with a --- frontmatter block naming its hypothesis, applies-when, support, provenance, last-validated, expires and carrier")
	}
	rest := text[4:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return nil, errors.New("the frontmatter block is not closed by ---")
	}
	have := map[string]string{}
	for _, line := range strings.Split(rest[:end], "\n") {
		k, v, ok := strings.Cut(line, ":")
		if ok && !strings.HasPrefix(line, " ") {
			have[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return have, nil
}

// Lint checks a lesson file's frontmatter presence: the block, and
// every FrontmatterKeys entry with a non-empty value.
func Lint(b []byte) error {
	have, err := Frontmatter(b)
	if err != nil {
		return err
	}
	var missing []string
	for _, k := range FrontmatterKeys {
		if have[k] == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("the frontmatter is missing %s", strings.Join(missing, ", "))
	}
	return nil
}

// LintFile is the promotion gate's file half (plans/os-96850e5a.md D4):
// the frontmatter agrees with the fact and the hypothesis, the file's
// bytes at the anchor hash to the fact's digest, the anchor commit is
// an ancestor of the head, every provenance anchor resolves, and the
// stamps hold at the declared instant. Every refusal is a GateError.
func LintFile(repo string, body []byte, fact LessonFact, h *HypothesisFact, now time.Time) error {
	subject := fact.Hypothesis
	// Both halves judge one revision: the bytes at the promoted anchor.
	// The working file is required to BE those bytes, so a valid
	// frontmatter in an uncommitted or later revision cannot stand in
	// for an invalid one the promotion recorded (review finding on the
	// task PR).
	path, commit, ok := AnchorParts(fact.Lesson)
	if !ok {
		return NewGateError(GateLintDigest, LessonVerb, subject, "the fact's lesson anchor is malformed")
	}
	at, err := gitShow(repo, commit, path)
	if err != nil || Digest(at) != fact.Digest {
		return NewGateError(GateLintDigest, LessonVerb, subject, fmt.Sprintf("the file's bytes at %s do not hash to the fact's digest %s", fact.Lesson, fact.Digest))
	}
	if !bytes.Equal(body, at) {
		return NewGateError(GateLintDigest, LessonVerb, subject, fmt.Sprintf("the file differs from the promoted bytes at %s: the lint judges the revision the promotion recorded, and a later edit is a later promotion", fact.Lesson))
	}
	body = at
	if err := Lint(body); err != nil {
		return NewGateError(GateLintFrontmatter, LessonVerb, subject, err.Error())
	}
	fm, _ := Frontmatter(body)
	if fm["hypothesis"] != fact.Hypothesis {
		return NewGateError(GateLintHypothesis, LessonVerb, subject, fmt.Sprintf("the frontmatter cites %q, the fact %q", fm["hypothesis"], fact.Hypothesis))
	}
	applies, err := ParseAppliesWhen(json.RawMessage(fm["applies-when"]))
	if err != nil {
		return NewGateError(GateLintAppliesWhen, LessonVerb, subject, err.Error())
	}
	if h == nil || !applies.Equal(h.AppliesWhen) {
		return NewGateError(GateLintAppliesWhen, LessonVerb, subject, "the frontmatter's applies-when differs from the hypothesis's")
	}
	support := splitList(fm["support"])
	if strings.Join(support, "\x00") != strings.Join(h.Support, "\x00") {
		return NewGateError(GateLintSupport, LessonVerb, subject, fmt.Sprintf("the frontmatter's support %v differs from the hypothesis's %v", support, h.Support))
	}
	for _, anchor := range splitList(fm["provenance"]) {
		path, commit, ok := AnchorParts(anchor)
		if !ok || !gitResolves(repo, commit, path) {
			return NewGateError(GateLintProvenance, LessonVerb, subject, fmt.Sprintf("provenance anchor %q does not resolve in the repository", anchor))
		}
	}
	if !gitAncestor(repo, commit, "HEAD") {
		return NewGateError(GateLintAncestry, LessonVerb, subject, fmt.Sprintf("the anchor commit %s is not an ancestor of the repository's head: the promotion PR has not merged", commit))
	}
	lv, ex, err := StampsOrdered(fm["last-validated"], fm["expires"])
	if err != nil {
		return NewGateError(GateLintStamps, LessonVerb, subject, err.Error())
	}
	if lv.After(now) {
		return NewGateError(GateLintStamps, LessonVerb, subject, fmt.Sprintf("last-validated %s is after the declared instant %s", fm["last-validated"], now.Format(time.RFC3339)))
	}
	if !ex.After(now) {
		return NewGateError(GateLintStamps, LessonVerb, subject, fmt.Sprintf("expires %s is not after the declared instant %s", fm["expires"], now.Format(time.RFC3339)))
	}
	if fm["last-validated"] != fact.LastValidated || fm["expires"] != fact.Expires {
		return NewGateError(GateLintStamps, LessonVerb, subject, fmt.Sprintf("the frontmatter's stamps (%s, %s) differ from the fact's (%s, %s): the reviewed file's dates are the lifecycle dates, and a promotion that recorded others carries dates nobody reviewed", fm["last-validated"], fm["expires"], fact.LastValidated, fact.Expires))
	}
	if fm["carrier"] != fact.Carrier {
		return NewGateError(GateLintCarrier, LessonVerb, subject, fmt.Sprintf("the frontmatter's carrier %q differs from the fact's %q", fm["carrier"], fact.Carrier))
	}
	return nil
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
