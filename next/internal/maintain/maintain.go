// Package maintain is the unattended maintenance loop
// (plans/os-8a5f14bb.md; SEED-NEXT.md conformance III.J): reap
// expired or wedged claims, reconcile divergence, rebuild
// projections, checkpoint, and lint — runnable with no scheduler and
// no wake channel, and audited as an ordinary actor.
//
// The decision logic lives here with its effects injected, so every
// rule below is drillable without a ledger. What the lane may
// actually DO is not decided here at all: every act it takes is
// signed with the maintenance key and crosses the same admission
// boundary as anyone else's, which is what "audited as an ordinary
// actor" has to mean if it is to mean anything.
package maintain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/shaunlmason/open-seed/next/internal/artifact"
	"github.com/shaunlmason/open-seed/next/internal/checkpoint"
	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/obligation"
	"github.com/shaunlmason/open-seed/next/internal/obs"
	"github.com/shaunlmason/open-seed/next/internal/packet"
	"github.com/shaunlmason/open-seed/next/internal/reconcile"
	"github.com/shaunlmason/open-seed/next/internal/transition"
	"github.com/shaunlmason/open-seed/next/internal/verdict"
)

// Corroboration is the LEDGER-side half of a reap's evidence: facts at
// known chain positions saying the holder was asked to stop.
//
// It exists as its own type because the reap rule turns on the two
// halves being independent, and a type you have to fill in separately
// from the classification makes that structural rather than hoped for.
type Corroboration struct {
	// Interrupted: an admitted run.interrupted stands on the active
	// fence with no deliberate exit after it. The "no exit after it"
	// half needs no scan: every deliberate exit leaves in_progress,
	// and a reap is admissible only FROM in_progress on that same
	// fence, so a window still standing on the interrupted fence has
	// had no exit by construction.
	Interrupted bool
	// Wedged: an admitted wedge.declared stands on the active fence.
	Wedged bool
}

// Stands reports whether any corroborating fact was found.
func (c Corroboration) Stands() bool { return c.Interrupted || c.Wedged }

// Reapable is the whole reap rule, and it is deliberately pure.
//
// A reap answers an UNANSWERED REQUEST, never a timeout. The
// observation channel is declared ephemeral and lossy
// (next/spec/observations.md), so a dropped stream and dead work look
// identical from outside and silence alone can never reap. The
// corroboration that makes a reap honest is a ledger fact that the
// holder was asked to stop and did not — which is exactly the force
// path next/spec/executors.md names this loop as the consumer of:
// "a worker that ignores its interrupt is killed... B-style automatic
// timeout reaping is the Phase 9 maintenance loop's job; it
// presupposes exactly these semantics."
//
// So a reap means "someone asked, and nothing happened", not "long
// enough has passed". That is the only corroboration a channel
// declared lossy can support, and it is why there is no threshold
// here to tune.
//
// no_data carries NO reap path whatever, however old the claim: a
// stream that holds nothing at all is the absence of evidence, and
// this is where the instinct to reap is strongest and the evidence
// weakest.
func Reapable(c obs.Classification, corr Corroboration) (bool, string) {
	switch c.State {
	case obs.NoData:
		return false, "the stream holds nothing at all, so there is no evidence either way — absence of data is stated, never read as death (next/spec/observations.md)"
	case obs.Live:
		return false, fmt.Sprintf("the stream is live (last observation %s)", c.LastObservation)
	case obs.Expired, obs.Wedged:
	default:
		return false, fmt.Sprintf("unclassified stream state %q", c.State)
	}
	if !corr.Stands() {
		return false, fmt.Sprintf("the stream is %s, but nothing in the chain says the holder was asked to stop: silence is not a request nobody answered, and the channel is lossy by design", c.State)
	}
	return true, ""
}

// Reap is one decided reap, carrying the evidence that decided it.
type Reap struct {
	Subject string `json:"subject"`
	Fence   int    `json:"fence"`
	Holder  string `json:"holder"`
	State   string `json:"state"`
	Because string `json:"because"`
}

// Skip is one claim the pass looked at and did NOT reap, with the
// reason. Skips are reported rather than dropped: a maintenance loop
// that silently declines is indistinguishable from one that never
// ran, and an operator reading the report is owed the difference.
type Skip struct {
	Subject string `json:"subject"`
	State   string `json:"state"`
	Because string `json:"because"`
}

// Filing is one defect contract the pass filed for a lint finding.
type Filing struct {
	Subject string `json:"subject"`
	Class   string `json:"class"`
	Filed   string `json:"filed"`
}

// Refusal is an act the boundary declined. The loop holds no private
// powers, so its acts can be refused like anyone else's, and a refused
// act is reported rather than retried or worked around.
type Refusal struct {
	Verb    string `json:"verb"`
	Subject string `json:"subject"`
	Reason  string `json:"reason"`
}

// Report is what one pass did.
type Report struct {
	Reaped     []Reap                 `json:"reaped"`
	Skipped    []Skip                 `json:"skipped"`
	Findings   []reconcile.Finding    `json:"findings"`
	Filed      []Filing               `json:"filed"`
	Rebuilt    []string               `json:"rebuilt"`
	Checkpoint *checkpoint.Checkpoint `json:"checkpoint,omitempty"`
	Refusals   []Refusal              `json:"refusals"`
}

// Deps is everything the pass reads and everything it does. The
// effects are injected so the rules above are drillable without a
// ledger, which is the shape internal/covergate established
// (plans/os-cafba959.md): a rule living in a CLI verb is a
// correctness claim nothing can check.
type Deps struct {
	// Now is the DECLARED as-of instant the classification is judged
	// at. It is a parameter rather than a clock read for the same
	// reason obs.Classify takes one: a pass must be reproducible, and
	// a maintenance loop that consulted a wall clock could not be
	// replayed.
	Now        time.Time
	Records    []*event.Record
	Table      *transition.Table
	Fold       *transition.Fold
	Obs        *obs.Snapshot
	Thresholds obs.Thresholds
	Store      *artifact.Store
	Repo       string
	// Unseal opens a sealed subject's checks under the maintenance
	// actor's identity, for the L3 reproduction the evidence grade
	// runs (plans/os-99829835.md D5); a subject it cannot open is
	// reported skipped with the reason, never passed over silently.
	Unseal func(s transition.SubjectState) (*verdict.SealedInput, error)
	// Obligations are the rows the obligation projection derived. The
	// unsettled-run lint CONSUMES these; re-deriving the anchoring
	// here would put it in two places (D2).
	Obligations []obligation.Row

	// Corroborate answers the ledger half of the reap rule for one
	// subject's active fence. Injected because the derivation belongs
	// to internal/admit, which owns "did this fact pass the boundary
	// at its own position" for every other fact too.
	Corroborate func(subject string, fence int) Corroboration
	// Append signs and appends one act. A refusal comes back as an
	// error and is reported, never retried.
	Append func(verb, subject string, payload []byte) error
	// File files a defect contract for a finding and returns the id
	// it filed under.
	File func(f reconcile.Finding) (string, error)
	// Rebuild rebuilds the projections and names what it rebuilt.
	Rebuild func() ([]string, error)
	// Materialize returns the canonical snapshot bytes and the
	// position they materialize.
	Materialize func() (body []byte, position int, err error)
}

// Run executes one pass in the fixed order: reap, lint, file, rebuild,
// checkpoint. The order matters in one place only — the checkpoint is
// last, because it attests to the state the rest of the pass produced.
func Run(d Deps) (Report, error) {
	rep := Report{
		Reaped: []Reap{}, Skipped: []Skip{}, Findings: []reconcile.Finding{},
		Filed: []Filing{}, Rebuilt: []string{}, Refusals: []Refusal{},
	}
	d.reap(&rep)
	rep.Findings = append(rep.Findings, d.lint(&rep)...)
	d.file(&rep)
	if err := d.rebuild(&rep); err != nil {
		return rep, err
	}
	return rep, d.checkpoint(&rep)
}

func (d Deps) reap(rep *Report) {
	if d.Fold == nil || d.Obs == nil {
		return
	}
	for _, id := range d.Fold.Subjects() {
		s, ok := d.Fold.State(id)
		if !ok || s.Claim == nil || s.State != "in_progress" {
			continue
		}
		stream, _ := d.Obs.StreamFor(s.Claim.Holder, obs.FormatFence(s.Claim.Fence))
		class := obs.Classify(stream, d.Now, d.Thresholds)
		var corr Corroboration
		if d.Corroborate != nil {
			corr = d.Corroborate(id, s.Claim.Fence)
		}
		ok2, because := Reapable(class, corr)
		if !ok2 {
			rep.Skipped = append(rep.Skipped, Skip{Subject: id, State: string(class.State), Because: because})
			continue
		}
		payload, err := ReapPacket(s, s.Claim.Fence, class, corr)
		if err != nil {
			rep.Refusals = append(rep.Refusals, Refusal{Verb: "claim.reaped", Subject: id, Reason: err.Error()})
			continue
		}
		if d.Append == nil {
			continue
		}
		if err := d.Append("claim.reaped", id, payload); err != nil {
			rep.Refusals = append(rep.Refusals, Refusal{Verb: "claim.reaped", Subject: id, Reason: err.Error()})
			continue
		}
		rep.Reaped = append(rep.Reaped, Reap{
			Subject: id, Fence: s.Claim.Fence, Holder: s.Claim.Holder,
			State: string(class.State), Because: reapBecause(corr),
		})
	}
}

func reapBecause(corr Corroboration) string {
	if corr.Interrupted {
		return "an admitted run.interrupted on the active fence went unanswered"
	}
	return "an admitted wedge.declared stands on the active fence"
}

// ReapPacket composes the whole claim.reaped payload: the fence the
// reap closes, and the honest four-part packet a forced reap leaves
// behind "from what is known" (next/spec/executors.md) — acceptance
// from the contract's specified criteria, base as the zero-length
// range because no pushed work is known, and findings recording the
// ignored request and the reap. Nothing here is invented: where the
// loop does not know, it says so.
//
// The FENCE citation is not decoration. claim.reaped is a
// claim-scoped event, and the fence rule refuses one that does not
// cite the active window — which is how a reap aimed at a window that
// already closed is refused rather than landing on whatever claim
// stands now. The first draft of this function returned the bare
// packet and every reap refused at the boundary; the drill that
// caught it is the one that read the chain back instead of the
// report.
func ReapPacket(s transition.SubjectState, fence int, c obs.Classification, corr Corroboration) ([]byte, error) {
	acceptance := "the contract's acceptance spec, which the fold does not carry"
	if s.Acceptance != nil && s.Acceptance.Ref != "" {
		acceptance = s.Acceptance.Ref
	}
	tried := "the holder was asked to stop by an admitted run.interrupted on this fence"
	if !corr.Interrupted {
		tried = "the holder's run was declared wedged on this fence"
	}
	outcome := fmt.Sprintf("no deliberate exit followed and the stream classified %s (last observation %q, last advance %q, count %d) — reaped by the maintenance lane, and the run's actuals settle afterward",
		c.State, c.LastObservation, c.LastAdvance, c.Count)
	body, err := packet.Render(packet.Packet{
		Acceptance: []string{acceptance},
		Decisions:  []packet.Decision{},
		Base:       packet.ZeroRange,
		Refs:       []string{},
		Findings:   []packet.Finding{{Tried: tried, Outcome: outcome}},
	})
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{
		"fence":    strconv.Itoa(fence),
		packet.Key: json.RawMessage(body),
	})
}

// lint runs the CLOSED finding set: the record-derived classes, the
// evidence-grade classes that need the artifact store and the
// repository, and the one class this card adds. Closed means a new
// lint lands by adding a class WITH the spec that pairs it to its
// fact; an open-ended list would make this loop a policy surface,
// which is what "audited as an ordinary actor" denies.
func (d Deps) lint(rep *Report) []reconcile.Finding {
	if d.Fold == nil {
		return nil
	}
	out := reconcile.Classify(d.Records, d.Fold)
	// The evidence-grade half. A pass built on Classify alone reports
	// clean over a rewritten target or an unretrievable receipt —
	// green, and omitting exactly the divergence this loop is
	// chartered to reconcile (D2.5).
	if d.Store != nil && d.Repo != "" {
		// The chain and the fold ride along so the L3 reproduction
		// runs here as it does under `seed reconcile` (review finding
		// on the item 3 task PR: a wrapper passing nil disabled it for
		// every maintenance pass); what the actor's key cannot open is
		// a skip the report carries.
		repro := reconcile.Reproduction{Records: d.Records, Fold: d.Fold, Unseal: d.Unseal,
			NotAttempted: func(subject, why string) {
				state := ""
				if s, ok := d.Fold.State(subject); ok {
					state = s.State
				}
				rep.Skipped = append(rep.Skipped, Skip{Subject: subject, State: state,
					Because: "the L3 verdict's receipt was not reproduced: " + why})
			}}
		for _, id := range d.Fold.Subjects() {
			if s, ok := d.Fold.State(id); ok {
				out = append(out, reconcile.EvidenceAt(id, s, d.Store, d.Repo, repro)...)
			}
		}
	}
	out = append(out, reconcile.Unsettled(d.Obligations)...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Subject != out[j].Subject {
			return out[i].Subject < out[j].Subject
		}
		return out[i].Class < out[j].Class
	})
	return out
}

// file turns each finding into a FILED DEFECT CONTRACT, never an
// escalation. The distinction is real: an escalation freezes a
// contract and demands a human decision, while a lint finding is work
// somebody should do.
//
// The consequence is worth stating rather than burying: this loop can
// therefore create work, which is authority. It is bounded by being
// attributable — its own key, its own lane — and by filing nothing but
// contracts, since it cannot claim what it files.
func (d Deps) file(rep *Report) {
	if d.File == nil {
		return
	}
	for _, f := range rep.Findings {
		id, err := d.File(f)
		if err != nil {
			rep.Refusals = append(rep.Refusals, Refusal{Verb: "intent.filed", Subject: f.Subject, Reason: err.Error()})
			continue
		}
		rep.Filed = append(rep.Filed, Filing{Subject: f.Subject, Class: f.Class, Filed: id})
	}
}

func (d Deps) rebuild(rep *Report) error {
	if d.Rebuild == nil {
		return nil
	}
	names, err := d.Rebuild()
	if err != nil {
		return err
	}
	rep.Rebuilt = append(rep.Rebuilt, names...)
	return nil
}

// checkpoint materializes the canonical projection state, stores it
// retrievably, and appends the signed citation.
//
// A checkpoint that carried only a signature over a hash would let
// every other criterion in this pass pass while the checkpoint itself
// was unusable: a reader could confirm somebody attested to a state it
// has no way to obtain, and would replay anyway. So the snapshot is
// written FIRST and the event names where it is and what it hashes to.
func (d Deps) checkpoint(rep *Report) error {
	if d.Materialize == nil || d.Store == nil || d.Append == nil {
		return nil
	}
	body, position, err := d.Materialize()
	if err != nil {
		return err
	}
	digest, err := d.Store.Put(body)
	if err != nil {
		return err
	}
	payload, err := checkpoint.Payload(digest, position)
	if err != nil {
		return err
	}
	if err := d.Append(checkpoint.Verb, "seed/0", payload); err != nil {
		rep.Refusals = append(rep.Refusals, Refusal{Verb: checkpoint.Verb, Subject: "seed/0", Reason: err.Error()})
		return nil
	}
	c, err := checkpoint.Parse("seed/0", payload)
	if err != nil {
		return err
	}
	rep.Checkpoint = c
	return nil
}

// DefectID is the id a finding files under: a stable hash of the class
// and the subject, prefixed so a filed defect is recognizable as one.
//
// Deriving it rather than allocating a fresh id makes filing
// IDEMPOTENT through the ledger itself: a second pass over the same
// standing finding re-files the same subject and the boundary refuses
// the duplicate. The alternative is for the loop to remember what it
// filed, and a maintenance loop that remembers is a maintenance loop
// that can forget — which on this surface means filing the same defect
// once per pass, forever.
func DefectID(f reconcile.Finding) string {
	sum := sha256.Sum256([]byte(f.Class + "\x00" + f.Subject))
	return "d-" + hex.EncodeToString(sum[:8])
}
