package admit

// The affordance computation (plans/os-f5551001.md; SEED-NEXT.md
// §II.10; charter III.I rows 1 and 2): the verbs currently legal for
// this actor on this subject, computed by drafting one SIGNED probe
// record per catalog verb and running the SAME Check pipeline
// admission enforces — one rule set, two consumers, zero exceptions,
// zero drift by construction. The catalog's payload synthesizers are
// synthesis data, never legality logic: they fill each verb's strict
// wire shape from the live context (the active fence, an open
// reservation, the bound submission, the standing verdict), and
// where an anchor is absent they fill a placeholder position, which
// the rules then refuse exactly as they would refuse the real
// append. Probes are drafted and checked, never appended, so
// computing affordances mutates nothing.

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/shaunlmason/open-seed/next/internal/checkpoint"
	"github.com/shaunlmason/open-seed/next/internal/curation"
	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/keyring"
	"github.com/shaunlmason/open-seed/next/internal/transition"
	"github.com/shaunlmason/open-seed/next/internal/tuple"
)

// probeView is the context snapshot the synthesizers fill templates
// from. Absent anchors hold "0": a well-formed citation the rules
// refuse, so illegality is judged by the rule set, never by the
// synthesizer.
type probeView struct {
	// support and hypothesis are the curation probes' citations
	// (curationProbes).
	support, hypothesis string
	// version is the chain's active protocol version: a probe must
	// synthesize the payload shape THAT version admits, or the
	// affordance list would say a verb is unavailable because the
	// probe spoke the wrong dialect (plans/os-8e53ffd9.md).
	version     string
	now         string
	expires     string
	fence       string
	active      bool
	reservation string
	submission  string
	verdict     string
	packet      string
	escalation  string
	// standing is whether a question stands right now: several
	// synthesizers must carry its citation only then.
	standing bool
	// choice is an id the STANDING question actually offers. A guess
	// would make a legal answer look unavailable: the probe is
	// judged by the same rule admission enforces, which refuses a
	// choice outside the set (review finding on #200).
	choice string
	// position is this view's own count, so a checkpoint probe cites
	// the position it would actually materialize rather than a
	// constant the rule would have to be loosened to accept.
	position string
	// qualify and disqualify are the qualification verbs' payloads
	// for the queried subject read as an ACTOR (plans/os-03e47abb.md):
	// the eval pass this actor's window earned and the configuration
	// it ran, or the eval fail whose configuration this actor still
	// holds. A citation the rules would refuse when none stands, so
	// the list says "you may mint this now" only when it is true.
	qualify    string
	disqualify string
}

// qualificationProbes derives the qualification verbs' payloads for
// the queried subject read as an actor: the first eval subject whose
// authenticated pass sits on a window this actor held, with the
// configuration that window's admitted start declared, and the first
// eval fail whose declared configuration this actor still holds. Where
// none stands the probe cites a contract and position the rules
// refuse, so illegality is judged by the rule set, never here.
// probeQualify and probeDisqualify are the citations that stand when
// no eval does: well-formed, and refused by the qualification rule.
const (
	probeQualify    = `{"capability": "claim", "tuple": ` + probeTuple + `, "contract": "probe", "verdict": "0"}`
	probeDisqualify = `{"capability": "claim", "tuple": ` + probeTuple + `, "contract": "probe", "verdict": "0", "reason": "probe"}`
)

func qualificationProbes(ctx *Context, actor string) (qualify, disqualify string) {
	qualify, disqualify = probeQualify, probeDisqualify
	if ctx == nil || ctx.Lifecycle == nil || ctx.Keyring == nil {
		return qualify, disqualify
	}
	foundQ, foundD := false, false
	for _, subject := range ctx.Lifecycle.Subjects() {
		s, ok := ctx.Lifecycle.State(subject)
		if !ok || s.Eval == nil {
			continue
		}
		if !foundQ {
			if fact := authenticPass(ctx, subject, s); fact != nil {
				if fence, holder, ok := submissionWindow(ctx.Records, subject, fact.Submission); ok && holder == actor {
					if declared := windowDeclaration(ctx, subject, s, fence); declared != nil {
						b, _ := json.Marshal(declared)
						qualify = fmt.Sprintf(`{"capability": "claim", "tuple": %s, "contract": %q, "verdict": "%d"}`, b, subject, fact.Pos)
						foundQ = true
					}
				}
			}
		}
		if !foundD {
			if fact := authenticFail(ctx, subject, s); fact != nil {
				if fence, _, ok := submissionWindow(ctx.Records, subject, fact.Submission); ok {
					if declared := windowDeclaration(ctx, subject, s, fence); declared != nil {
						for _, held := range ctx.Keyring.GrantTuples(actor, keyring.CapClaim) {
							if held.Equal(*declared) {
								b, _ := json.Marshal(declared)
								disqualify = fmt.Sprintf(`{"capability": "claim", "tuple": %s, "contract": %q, "verdict": "%d", "reason": "the eval failed"}`, b, subject, fact.Pos)
								foundD = true
								break
							}
						}
					}
				}
			}
		}
	}
	return qualify, disqualify
}

// windowDeclaration is the tuple the admitted run.started in the
// window at fence declared, nil when no valid start declared one.
func windowDeclaration(ctx *Context, subject string, s transition.SubjectState, fence int) *tuple.Tuple {
	for i := range s.RunStarts {
		st := s.RunStarts[i]
		if st.Fence == fence && RunStartValid(ctx.Records, ctx.Table, subject, st) {
			return st.Tuple
		}
	}
	return nil
}

// fenceKV is the optional fence citation: on a held subject the
// fence rule requires holder-signed events to cite the active fence
// and lets anyone else cite it, while outside a window any citation
// refuses — so fence-optional verbs cite it exactly when one is
// active.
func (v *probeView) fenceKV() string {
	if !v.active {
		return ""
	}
	return `"fence": "` + v.fence + `", `
}

// probePacket is the minimal shape-valid four-part packet: non-empty
// acceptance, marked decisions (none), the zero-length base range,
// and honestly empty refs and findings (next/spec/packets.md).
const probePacket = `{"acceptance": ["probe"], "decisions": [], "base": "0000000000000000000000000000000000000000..0000000000000000000000000000000000000000", "refs": [], "findings": []}`

// probeClaim is the claim the hypothesis probe proposes; its subject is
// the id the claim derives, so the probe is signed where the rule
// looks. The two defaults are what the probes cite when the record
// holds nothing citable: shape-valid, and refused.
const (
	probeClaim      = "probe"
	probeSupport    = `["probe@0", "probe@0"]`
	probeHypothesis = "h-000000000000@0"
)

func (v *probeView) supportOr() string {
	if v.support == "" {
		return probeSupport
	}
	return v.support
}

func (v *probeView) hypothesisOr() string {
	if v.hypothesis == "" {
		return probeHypothesis
	}
	return v.hypothesis
}

// curationProbes derives the citations the curation probes make: two
// admitted observations on two distinct non-failed contracts (the
// support a proposal needs) and the latest admitted hypothesis (the
// citation a promotion needs). Where the record holds none the probe
// cites what the rules refuse, so the affordance is invisible exactly
// when the act is not yet legal.
func curationProbes(ctx *Context) (support, hypothesis string) {
	support, hypothesis = probeSupport, probeHypothesis
	if ctx == nil || ctx.Table == nil || ctx.Lifecycle == nil {
		return
	}
	byContract := map[string]string{}
	var cited []string
	for pos, rec := range ctx.Records {
		e := &rec.Event
		if _, ok := byContract[e.Subject]; ok || len(cited) >= curation.SupportMinimum {
			continue
		}
		cit := curation.Citation{Contract: e.Subject, Position: pos}
		if _, ok := curation.ObservationAt(ctx.Records, ctx.Table, cit); !ok || curation.FailedAt(ctx.Records, ctx.Table, e.Subject) {
			continue
		}
		byContract[e.Subject] = fmt.Sprintf("%s@%d", e.Subject, pos)
		cited = append(cited, fmt.Sprintf("%q", byContract[e.Subject]))
	}
	if len(cited) >= curation.SupportMinimum {
		support = "[" + strings.Join(cited, ", ") + "]"
	}
	fold := curation.Fold(ctx.Records)
	for _, id := range fold.HypothesisIDs() {
		h, _ := fold.Hypothesis(id)
		if _, ok := curation.HypothesisValid(ctx.Records, ctx.Table, curation.Citation{Contract: id, Position: h.Pos}); ok {
			hypothesis = fmt.Sprintf("%s@%d", id, h.Pos)
		}
	}
	return
}

// probeEscalation is the minimal shape-valid question: one sentence
// and the two-option floor a minimal decision needs
// (next/spec/escalation.md).
const probeEscalation = `{"question": "probe?", "options": [{"id": "a", "choice": "probe a"}, {"id": "b", "choice": "probe b"}]}`

// probeTuple is the configuration the run.started probe declares at
// seed/2. A probe asks "could a start be admitted here"; a holder whose
// qualified grants cite something else would refuse it as drift, which
// is the honest answer to that question for that holder.
const probeTuple = `{"principal": "probe", "harness": "probe/0", "model": "probe/0", "tool_policy": "probe", "environment": "probe"}`

// affordanceCatalog is every verb the envelope can list, each with
// its payload synthesizer. Completeness is pinned by test: a catalog
// verb without a synthesizer, or a spec-table verb missing from the
// catalog, is a test failure — absence from an envelope always means
// the probe was REFUSED at this position, never that synthesis was
// skipped. The one named carve-out is actor.enrolled, whose valid
// payload requires the queried subject's public key: fingerprints
// are hashes, so no prober can derive it, and the synthesizer's
// ephemeral key is refused by the keyring's subject binding on any
// subject whose key the caller does not hold (recorded decision;
// the enrollment surface knows its key out of band).
// probeSubjects names, for the verbs that live on a derived subject,
// the subject the probe is signed on instead of the caller's: the
// curation facts (a hypothesis on the id its claim derives, a
// promotion on the hypothesis it cites) are legal on no contract.
var probeSubjects = map[string]func(v *probeView) string{
	"curation.hypothesis.proposed": func(v *probeView) string { return curation.HypothesisID(probeClaim) },
	"curation.lesson.promoted": func(v *probeView) string {
		h, _ := curation.ParseCitation(v.hypothesisOr())
		return h.Contract
	},
}

var affordanceCatalog = []struct {
	verb  string
	synth func(v *probeView) string
}{
	{"system.halt.declared", func(v *probeView) string { return `{"reason": "probe"}` }},
	{"system.halt.lifted", func(v *probeView) string { return `{}` }},
	{"system.protocol.upgraded", func(v *probeView) string { return `{"to": "seed/1"}` }},
	{"system.checkpoint", func(v *probeView) string {
		// A shape-valid snapshot citation (next/spec/maintenance.md):
		// the versioned format, a well-formed digest, a fetchable
		// location, and this view's own position. The old `{"n": 1}`
		// stopped being admissible when the checkpoint rule landed,
		// and a synthesizer that no longer matches the rule makes a
		// LEGAL act invisible in the orientation read — the #200
		// failure, which is why the catalog is swept by test.
		return `{"format": "` + checkpoint.Format + `", "snapshot": "` +
			strings.Repeat("0", 64) + `", "location": "` + checkpoint.Location +
			`", "position": "` + v.position + `"}`
	}},
	{"actor.enrolled", func(v *probeView) string {
		pub, _, err := ed25519.GenerateKey(nil)
		if err != nil {
			return `{}`
		}
		return fmt.Sprintf(`{"key": "%x", "kind": "agent", "name": "probe"}`, []byte(pub))
	}},
	{"actor.granted", func(v *probeView) string { return `{"capability": "claim"}` }},
	{"actor.qualified", func(v *probeView) string {
		if v.qualify == "" {
			return probeQualify
		}
		return v.qualify
	}},
	{"actor.disqualified", func(v *probeView) string {
		if v.disqualify == "" {
			return probeDisqualify
		}
		return v.disqualify
	}},
	{"actor.suspended", func(v *probeView) string { return `{"reason": "probe"}` }},
	{"actor.revoked", func(v *probeView) string { return `{"reason": "probe"}` }},
	{"intent.filed", func(v *probeView) string {
		return `{"intent": "probe", "tier": "trivial", "budget": "small", "routing": "core"}`
	}},
	{"contract.specified", func(v *probeView) string {
		return `{"acceptance": {"ref": "accept.md @ 0000000000000000000000000000000000000000", "executable": true, "gate": "probe @ 0000000000000000000000000000000000000000"}}`
	}},
	{"contract.blocked", func(v *probeView) string { return `{}` }},
	{"contract.unblocked", func(v *probeView) string { return `{}` }},
	{"contract.cancelled", func(v *probeView) string {
		// Once a question stands, cancelling is legal only WITH the
		// citation, and it is one of the two documented ways to answer
		// the gate — so a bare {} would hide it precisely when it
		// matters (review finding on #200).
		if v.standing {
			return `{"escalation": "` + v.escalation + `"}`
		}
		return `{}`
	}},
	{"escalation.raised", func(v *probeView) string {
		return `{"packet": ` + v.packet + `, "escalation": ` + probeEscalation + `}`
	}},
	{"decision.recorded", func(v *probeView) string {
		return `{"escalation": "` + v.escalation + `", "choice": "` + v.choice + `"}`
	}},
	{"contract.returned", func(v *probeView) string { return `{"verdict": "` + v.verdict + `"}` }},
	{"claim.taken", func(v *probeView) string { return `{}` }},
	{"claim.released", func(v *probeView) string {
		return `{"fence": "` + v.fence + `", "packet": ` + v.packet + `}`
	}},
	{"claim.parked", func(v *probeView) string {
		return `{"fence": "` + v.fence + `", "packet": ` + v.packet + `}`
	}},
	{"claim.reaped", func(v *probeView) string {
		return `{"fence": "` + v.fence + `", "packet": ` + v.packet + `}`
	}},
	{"submission.made", func(v *probeView) string {
		return `{"fence": "` + v.fence + `", "packet": ` + v.packet + `}`
	}},
	{"curation.deadend.recorded", func(v *probeView) string {
		return `{"fence": "` + v.fence + `", "tried": "probe", "outcome": "probe", "condition": "probe", "environment": "probe"}`
	}},
	{"curation.hypothesis.proposed", func(v *probeView) string {
		return `{"claim": "` + probeClaim + `", "applies_when": "probe", "support": ` + v.supportOr() + `, "exceptions": [], "provenance": []}`
	}},
	{"curation.lesson.promoted", func(v *probeView) string {
		return `{"lesson": "` + curation.LessonsDir + `/probe.md @ 0000000000000000000000000000000000000000", "hypothesis": "` + v.hypothesisOr() + `", "pr": "pr/0 @ 0000000000000000000000000000000000000000"}`
	}},
	{"plan.proposed", func(v *probeView) string {
		return `{` + v.fenceKV() + `"plan": "probe.md @ 0000000000000000000000000000000000000000"}`
	}},
	{"plan.approved", func(v *probeView) string {
		return `{` + v.fenceKV() + `"plan": "probe.md @ 0000000000000000000000000000000000000000", "pr": "probe"}`
	}},
	{"progress.milestone", func(v *probeView) string {
		return `{` + v.fenceKV() + `"count": 1, "step": "probe"}`
	}},
	{"wedge.declared", func(v *probeView) string {
		return `{` + v.fenceKV() + `"observed": "` + v.now + `", "count": 0, "since": "` + v.now + `"}`
	}},
	{"message.sent", func(v *probeView) string { return `{` + v.fenceKV() + `"n": 1}` }},
	{"offer.published", func(v *probeView) string {
		return `{"eligibility": {"capabilities": ["claim"], "tiers": []}, "expires": "` + v.expires + `"}`
	}},
	// The budget facts cite the fence CONDITIONALLY
	// (plans/os-d6963652.md D5): a reservation outlives its claim
	// window and closes wherever it stands, so an unconditional
	// citation would have the fence rule refuse the probe outside a
	// window and hide a legal close. Reserve joins them although its
	// own rule still refuses it there: refused by the right rule, so
	// the envelope answers "why not" honestly.
	{"budget.reserve", func(v *probeView) string {
		return `{` + v.fenceKV() + `"amount": "1"}`
	}},
	{"budget.settle", func(v *probeView) string {
		return `{` + v.fenceKV() + `"reservation": "` + v.reservation + `", "actuals": "1"}`
	}},
	{"budget.release", func(v *probeView) string {
		return `{` + v.fenceKV() + `"reservation": "` + v.reservation + `"}`
	}},
	{"run.started", func(v *probeView) string {
		if tuple.Applies(v.version) {
			return `{"fence": "` + v.fence + `", "reservation": "` + v.reservation + `", "tuple": ` + probeTuple + `}`
		}
		return `{"fence": "` + v.fence + `", "reservation": "` + v.reservation + `"}`
	}},
	{"run.settled", func(v *probeView) string {
		return `{"fence": "` + v.fence + `", "units": "0", "lines": "0"}`
	}},
	{"run.interrupted", func(v *probeView) string { return `{"fence": "` + v.fence + `"}` }},
	{"verdict.rendered", func(v *probeView) string {
		return `{"verdict": "pass", "receipt": "0000000000000000000000000000000000000000000000000000000000000000", "submission": "` + v.submission + `", "independence": "L1"}`
	}},
	{"check.sealed", func(v *probeView) string {
		return `{"commitment": "0000000000000000000000000000000000000000000000000000000000000000"}`
	}},
	{"merge.requested", func(v *probeView) string { return `{"verdict": "` + v.verdict + `"}` }},
	{"merge.observed", func(v *probeView) string {
		return `{"merged": "0000000000000000000000000000000000000000", "pr": "probe"}`
	}},
	{"merge.overridden", func(v *probeView) string {
		return `{"reason": "probe", "verdict": "` + v.verdict + `"}`
	}},
}

// Affordances lists the verbs currently legal for the signing key's
// actor on the subject, at the context's position. Each catalog verb
// is drafted with its synthesized payload, signed with the caller's
// key (the pipeline's actor rule verifies record signatures, so a
// fingerprint alone cannot probe), and run through the full Check;
// the verb is listed iff its probe admits. The result is sorted and
// deduplicated for schema stability. Errors never escape: a context
// that cannot be probed yields the empty list, because affordances
// are advisory and must not break the verb that carries them.
func Affordances(ctx *Context, key ed25519.PrivateKey, subject string) []string {
	if ctx == nil || len(key) != ed25519.PrivateKeySize || subject == "" {
		return []string{}
	}
	fp, err := event.Fingerprint(key.Public().(ed25519.PublicKey))
	if err != nil {
		return []string{}
	}
	now := time.Now().UTC()
	v := &probeView{
		now:         now.Format(time.RFC3339),
		expires:     now.Add(time.Hour).Format(time.RFC3339),
		fence:       "0",
		reservation: "0",
		submission:  "0",
		verdict:     "0",
		packet:      probePacket,
		escalation:  "0",
		position:    fmt.Sprintf("%d", ctx.Count),
	}
	v.version = ctx.Active
	v.qualify, v.disqualify = qualificationProbes(ctx, subject)
	v.support, v.hypothesis = curationProbes(ctx)
	if ctx.Lifecycle != nil {
		if s, ok := ctx.Lifecycle.State(subject); ok {
			if s.Claim != nil {
				v.fence = fmt.Sprintf("%d", s.Claim.Fence)
				v.active = true
			}
			if s.Submission != nil {
				v.submission = fmt.Sprintf("%d", s.Submission.Pos)
			}
			if s.Verdict != nil {
				v.verdict = fmt.Sprintf("%d", s.Verdict.Pos)
			}
			if s.Escalation != nil {
				v.escalation = fmt.Sprintf("%d", s.Escalation.Pos)
				v.standing = true
				if len(s.Escalation.Options) > 0 {
					v.choice = s.Escalation.Options[0].ID
				}
			}
			view := BudgetViewAt(ctx.Records, ctx.Table, subject, s)
			if len(view.Open) > 0 {
				v.reservation = fmt.Sprintf("%d", view.Open[0].Pos)
			}
		}
	}
	var out []string
	seen := map[string]bool{}
	for _, p := range affordanceCatalog {
		on := subject
		if derive, ok := probeSubjects[p.verb]; ok {
			if derived := derive(v); derived != "" {
				on = derived
			}
		}
		rec, err := event.Sign(event.Event{
			V: ctx.Active, TS: v.now, Actor: fp, Verb: p.verb, Subject: on,
			Payload: []byte(p.synth(v)), Prev: ctx.Tip,
		}, key)
		if err != nil {
			continue
		}
		if Check(ctx, rec) == nil && !seen[p.verb] {
			seen[p.verb] = true
			out = append(out, p.verb)
		}
	}
	sort.Strings(out)
	if out == nil {
		out = []string{}
	}
	return out
}

// BudgetBlock derives the envelope's {reserved, remaining} strings
// for the subject at the context's position, from the one shared
// budget derivation. ok is false on subjects carrying no budget
// facts, where the envelope keeps its null block.
func BudgetBlock(ctx *Context, subject string) (reserved, remaining string, ok bool) {
	if ctx == nil || ctx.Lifecycle == nil {
		return "", "", false
	}
	s, found := ctx.Lifecycle.State(subject)
	if !found || (len(s.Reservations) == 0 && len(s.BudgetCloses) == 0) {
		return "", "", false
	}
	view := BudgetViewAt(ctx.Records, ctx.Table, subject, s)
	open := 0
	for _, r := range view.Open {
		// Saturating, mirroring the budget arithmetic: presentation
		// must never wrap where the derivation would not.
		if open > int(^uint(0)>>1)-r.Amount {
			open = int(^uint(0) >> 1)
		} else {
			open += r.Amount
		}
	}
	return fmt.Sprintf("%d", open), fmt.Sprintf("%d", view.Remaining), true
}
