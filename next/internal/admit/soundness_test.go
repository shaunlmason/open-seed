package admit

// The III.I regression class made executable (plans/os-148d3ba1.md;
// docs/next-build-plan.md Phase 8 item 2): an affordance-listed
// verb refused at admission for legality at the same position is a
// bug, and this sweep is that class as a test — at every prefix of
// the shared walk scenario, for every enrolled lane, the listed set
// must be sorted and deterministic, and every listed verb must
// admit when independently re-drafted and run through the enforcing
// Check at that position. Today listed-implies-admits holds by
// construction, because the computation is the probe run; the sweep
// pins the construction so a later split — caching, memoization, a
// parallel legality derivation, a rule consulted by one consumer
// and not the other — fails here as a named class instead of
// drifting silently.

import (
	"crypto/ed25519"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/shaunlmason/open-seed/next/internal/event"
)

// probeViewAt is the sweep's independent copy of the affordance
// computation's view derivation (the anchors a probe payload cites:
// the active fence, an open reservation, the bound submission, the
// standing verdict, placeholders where absent). It is deliberately
// not shared with the production derivation in Affordances: if the
// two ever disagree on an anchor, the re-drafted record diverges
// from the probe and the class fails, which is exactly the drift
// signal this test exists to raise.
func probeViewAt(ctx *Context, subject string) *probeView {
	now := time.Now().UTC()
	v := &probeView{
		now:         now.Format(time.RFC3339),
		expires:     now.Add(time.Hour).Format(time.RFC3339),
		fence:       "0",
		reservation: "0",
		submission:  "0",
		verdict:     "0",
		packet:      probePacket,
		position:    fmt.Sprintf("%d", ctx.Count),
		request:     "0",
		erasable:    strings.Repeat("0", 64),
	}
	if ctx.Lifecycle != nil {
		// The request probes' citations, as the production view
		// carries them (next/spec/requests.md): the queried contract
		// as the filing's `about`, the first unanswered request on
		// the queried subject as the answer's citation.
		for _, r := range ctx.Lifecycle.Requests() {
			if r.Subject == subject && r.Answered == nil {
				v.request = fmt.Sprintf("%d", r.Pos)
				break
			}
		}
		if s, ok := ctx.Lifecycle.State(subject); ok {
			v.requestAbout = subject
			// The erasure probe's digest, as the production view
			// carries it (plans/os-db5cd353.md D6): the sealed
			// commitment, else the latest receipt.
			// The first reference not yet erased, so the verb is
			// drafted exactly while something remains erasable and
			// never for a tombstoned digest (plans/os-db5cd353.md D6).
			candidates := []string{}
			if s.Sealed != nil {
				candidates = append(candidates, s.Sealed.Commitment)
			}
			for _, vf := range s.Verdicts {
				if vf.Receipt != "" {
					candidates = append(candidates, vf.Receipt)
				}
			}
			for _, d := range candidates {
				if _, erased := Erasure(ctx.Records, ctx.Lifecycle, d); !erased {
					v.erasable = d
					break
				}
			}
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
	// The chain's active version, as the production view carries it:
	// a probe at seed/4 or later must carry the plan digest the shape
	// rule requires, or the regression class fires on its own helper.
	v.version = ctx.Active
	return v
}

// conformance: III.I row 2 — the same-rule-set property test: the
// affordance computation and admission enforcement consume the same
// rule set, so a listed verb refused for legality at the listed
// position (no concurrent event; the position is held fixed by
// construction) is a bug class, and this harness is its regression
// test. The matrix is every enrolled lane against the subjects
// carrying the scenario's legality state (plan D3); the per-position
// verb set is never trimmed. The sweep stays inside the fast gate:
// roughly thirty prefixes by seven pairs, two listings and at most
// one re-check per listed verb each.
func TestAffordanceRegressionClass(t *testing.T) {
	store, resolve, signer := seededStore(t)
	lanes := walkLanes(t)
	keys := map[string]ed25519.PrivateKey{"root": signer}
	for name, key := range lanes {
		keys[name] = key
	}
	loose := walkResolver(t, resolve, lanes)
	pairs := []struct{ lane, subject string }{
		{"root", "system"},
		{"root", "c-1"},
		{"holder", "c-1"},
		{"supervisor", "c-1"},
		{"verifier", "c-1"},
		{"sealer", "c-1"},
		{"observer", "c-1"},
	}
	synth := map[string]func(*probeView) string{}
	for _, p := range affordanceCatalog {
		synth[p.verb] = p.synth
	}
	sweep := func(pos int) {
		ctx, err := ContextAt(store)
		if err != nil {
			t.Fatalf("position %d: %v", pos, err)
		}
		for _, pair := range pairs {
			key := keys[pair.lane]
			listed := Affordances(ctx, key, pair.subject)
			if again := Affordances(ctx, key, pair.subject); !slices.Equal(listed, again) {
				t.Errorf("III.I class: nondeterministic affordances for %s on %s at position %d: %v vs %v",
					pair.lane, pair.subject, pos, listed, again)
			}
			// Strictly ascending: sorted AND deduplicated in one
			// assertion, because IsSorted alone accepts adjacent
			// duplicates (review finding on this PR) and the
			// contract's list is a set.
			for i := 1; i < len(listed); i++ {
				if listed[i-1] >= listed[i] {
					t.Errorf("III.I class: affordances not strictly ascending (unsorted or duplicated) for %s on %s at position %d: %v",
						pair.lane, pair.subject, pos, listed)
					break
				}
			}
			fp := fpOf(t, key)
			v := probeViewAt(ctx, pair.subject)
			for _, verb := range listed {
				fill, ok := synth[verb]
				if !ok {
					t.Fatalf("III.I class: %s listed but absent from the catalog at position %d", verb, pos)
				}
				rec, err := event.Sign(event.Event{
					V: ctx.Active, TS: v.now, Actor: fp, Verb: verb,
					Subject: pair.subject, Payload: []byte(fill(v)), Prev: ctx.Tip,
				}, key)
				if err != nil {
					t.Fatalf("drafting %s at position %d: %v", verb, pos, err)
				}
				if err := Check(ctx, rec); err != nil {
					t.Errorf("III.I regression class: %s listed for %s on %s at position %d but refused at admission: %v",
						verb, pair.lane, pair.subject, pos, err)
				}
			}
		}
	}
	sweep(0)
	for i, s := range walkScript(t, lanes) {
		runWalkStep(t, store, loose, keys, s)
		sweep(i + 1)
	}
}
