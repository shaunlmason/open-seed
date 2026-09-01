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
	"fmt"
	"sort"
	"time"

	"github.com/shaunlmason/open-seed/next/internal/event"
)

// probeView is the context snapshot the synthesizers fill templates
// from. Absent anchors hold "0": a well-formed citation the rules
// refuse, so illegality is judged by the rule set, never by the
// synthesizer.
type probeView struct {
	now         string
	expires     string
	fence       string
	active      bool
	reservation string
	submission  string
	verdict     string
	packet      string
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
var affordanceCatalog = []struct {
	verb  string
	synth func(v *probeView) string
}{
	{"system.halt.declared", func(v *probeView) string { return `{"reason": "probe"}` }},
	{"system.halt.lifted", func(v *probeView) string { return `{}` }},
	{"system.protocol.upgraded", func(v *probeView) string { return `{"to": "seed/1"}` }},
	{"system.checkpoint", func(v *probeView) string { return `{"n": 1}` }},
	{"actor.enrolled", func(v *probeView) string {
		pub, _, err := ed25519.GenerateKey(nil)
		if err != nil {
			return `{}`
		}
		return fmt.Sprintf(`{"key": "%x", "kind": "agent", "name": "probe"}`, []byte(pub))
	}},
	{"actor.granted", func(v *probeView) string { return `{"capability": "claim"}` }},
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
	{"contract.cancelled", func(v *probeView) string { return `{}` }},
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
	}
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
			view := BudgetViewAt(ctx.Records, ctx.Table, subject, s)
			if len(view.Open) > 0 {
				v.reservation = fmt.Sprintf("%d", view.Open[0].Pos)
			}
		}
	}
	var out []string
	seen := map[string]bool{}
	for _, p := range affordanceCatalog {
		rec, err := event.Sign(event.Event{
			V: ctx.Active, TS: v.now, Actor: fp, Verb: p.verb, Subject: subject,
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
