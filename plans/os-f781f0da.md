# Plan: next — escalation with packet, question and decision (os-f781f0da)

Phase 9 item 2 of `docs/next-build-plan.md`: *"Escalation
(`blocked(needs-you)`) with packet + question + decision; report
surfaces age. An escalation raised in answer to a refusal carries that
refusal's `code` and `message` in its packet."*

The charter is unusually specific here, and most of the design work is
reading it rather than inventing:

- §II.7 — "Escalation is `blocked(needs-you)`: an event addressed to a
  human gate carrying the packet, the question, and the minimal
  decision — never a transcript."
- §II.11 — "Any lane can raise `blocked(needs-you)`: packet + question
  + minimal decision; the supervisor routes it, the report surfaces it
  with age, and **nothing else about the contract moves until it is
  answered**. A human's unit of interruption is one decision, never a
  transcript."
- §I.3 — "Humans hold gates, not queues: intent, high-consequence
  plans, the protected surface, and escalations."
- III — "Escalations carry packet + question + minimal decision;
  waiting escalations surface with age; resolution latency is
  tracked"; "Every escalation is one packet + one question + one
  decision; transcript-dumping is a defect."

`next/spec/packets.md` already points forward to this card: *"Escalation
packets (`blocked(needs-you)` carrying packet + question + minimal
decision) reuse this schema when the routing/escalation phase lands — a
forward pointer, not a gap."* This plan pays that pointer.

## Design decisions (binding for this task)

- **D1 — escalation is a QUALIFIER on the exits that already reach
  `blocked`, never a fifth exit.** From `in_progress` it rides
  `claim.parked`; from `ready` it rides `contract.blocked`.

  This is forced, not preferred. `next/spec/lifecycle.md` pins the four
  deliberate exits from `in_progress` by self-validation, and III.F
  ("every exit from `in_progress` is deliberate … silent abandonment is
  impossible by construction") depends on that set being **closed**. A
  new `escalation.raised` row out of `in_progress` would open it to buy
  nothing: the park already carries the four-part packet, already
  requires the active fence, and already lands in `blocked`. An
  escalation is what the payload SAYS, not a new door.

  Consequence, stated rather than hidden: "any lane can raise" is
  bounded in v0 by who can already reach `blocked`. The claim lane can
  (its park) and the dispatch lane can (`contract.blocked`). A verifier
  holding a `review` subject **cannot** — there is no `review` →
  `blocked` row, and adding one is a lifecycle change with its own
  consequences for the reconciliation chain. Recorded as a known v0
  gap with its reason rather than closed by inventing a row this item
  does not ask for.

- **D2 — the ANSWER is its own operator-only verb: `decision.recorded`
  (`blocked` → `ready`).** It cannot ride `contract.unblocked`, which
  is `{dispatch, operator}` (`internal/keyring`): a machine lane
  answering a human gate is the exact inversion of §I.3, and a
  capability row is where that is said. So `decision.recorded` takes
  **operator and nothing else** — the FOURTH no-fallback row, after
  `verdict.rendered`, `check.sealed` and `merge.overridden`, and for
  the reason the third has one: the charter names the act attributable
  human judgment, and a fallback would quietly make it something
  else.

- **D3 — while an escalation stands unanswered, `contract.unblocked`
  REFUSES.** That is "nothing else about the contract moves until it is
  answered", made structural. `blocked` has exactly two machine-visible
  exits; this closes the one a machine holds. The refusal follows
  `next/spec/refusals.md` discipline: it names the standing
  escalation's position and the verb that discharges it, so the caller
  is told what IS legal.

  `contract.cancelled` stays legal, deliberately: it is already
  operator-only, and cancelling an escalated contract IS a human
  decision — refusing it would trap the contract with no operator path
  out, which is a worse failure than the one being prevented.

  The lockout is derived from the fold at admission time, following the
  red-verdict lockout precedent (`plans/os-d2497eb7.md`,
  `internal/admit/lockout_test.go`), never from a stored flag.

- **D4 — the question and the decision are SIBLING payload fields, never
  a fifth packet part.** `packets.md` binds "exactly four parts, all
  keys present" and refuses unknown keys at the top level. So the raise
  carries the packet unchanged beside a new `escalation` object, and
  the answer carries its own fields:

  ```json
  {"fence": "12", "packet": {…four parts…},
   "escalation": {"question": "<one sentence>",
                  "options": [{"id": "a", "choice": "<one sentence>"},
                              {"id": "b", "choice": "<one sentence>"}]}}
  ```
  ```json
  {"escalation": "<position>", "choice": "a", "because": "<one sentence>"}
  ```

- **D5 — `options` is REQUIRED, with at least two entries, and the
  answer must cite one of them by id.** This is the decision most worth
  arguing, because it is the one that makes "one decision, never a
  transcript" checkable rather than aspirational.

  A free-text question with no answer set is an invitation to design
  work, which is what §II.11's "minimal decision" excludes. Requiring a
  closed set forces the reducing work onto the lane, where it belongs:
  a lane that cannot state two candidate answers has not reduced its
  problem far enough to spend a human's attention on it.

  Cost, stated: a genuinely open question ("what should this be
  called?") cannot be escalated as such. The lane must either propose
  candidates or file an intent, which is the correct routing for
  open-ended work anyway. An "other — say what" escape hatch is
  **deliberately not offered**: it re-admits the open question through
  the door built to exclude it, and every escalation would grow one.

  Free text (`question`, `choice`, `because`) falls under the
  classification lint's aggregate free-text budget, exactly as packet
  prose does. Transcript-dumping is then a refusal rather than a
  style note.

- **D6 — waiting escalations surface as an OBLIGATION, and age needs no
  new surface.** `next/spec/obligations.md` rows already carry `since
  <position>` under a named clock, so a new fact-shaped kind is the
  whole of "the report surfaces it with age":

  | kind | owed by | discharged by |
  | --- | --- | --- |
  | `escalation.pending` | `lane:operator` | `decision.recorded`, `contract.cancelled` |

  An operator's `seed situation` read then lists waiting escalations
  with the position that raised each, and **resolution latency is the
  chain's own arithmetic** — the answer's position minus the raise's —
  derivable by anyone from history, stored nowhere.

  Not chosen: a `seed report` verb. Nothing in the tree has one; the
  build plan's "report surfaces age" names an outcome, not a surface;
  and rendering a projection `situation` already renders would be a
  second copy of one derivation, which is the failure mode
  `obligations.md` exists to prevent.

- **D7 — one new loop verb (`decision record`) and two new flags on
  `claim park`.** The raise is `claim park --question <s> --option
  <id>=<s>` (repeatable); the answer is `seed decision record --subject
  <c> --choice <id> [--because <s>]`, deriving the escalation citation
  from the single standing escalation in the fold and **refusing on
  ambiguity or absence** rather than picking — the derivation discipline
  `next/spec/loop-verbs.md` already binds. `contract.blocked` keeps no
  loop verb: it is a dispatch act, and `loop-verbs.md` deliberately
  leaves filing and specification verbs on the raw seam.

- **D8 — the refusal an escalation answers is carried VERBATIM, and the
  existing rule is reused rather than copied.** `internal/loop` already
  puts a refusal's `code` and `message` unchanged into the park
  packet's `findings`. The escalation adds the question beside that
  packet; it does not restate, summarize or re-render the refusal. A
  second copy of the verbatim rule would be a second thing to drift.

- **D9 — scope guard.** No supervisor routing (§7's dependency cascade
  is a later item's work). No policy in `internal/loop` about WHEN to
  escalate: the loop gains the ability, the caller keeps the judgment,
  and item 4's fixtures supply the policy. No new exit codes, no new
  projection, no change to the four-part packet.

## Steps

1. `next/spec/transitions.json` + `next/spec/lifecycle.md` — the
   `decision.recorded` row (`blocked` → `ready`) and the quotation that
   a drill pins to the table.
2. `next/internal/transition` — the embedded table moves with it.
3. `next/internal/keyring` + `next/spec/actors.md` — the operator-only
   capability row, its no-fallback reason recorded in the form the
   three existing such rows use.
4. `next/internal/admit` — the escalation shape rule (question,
   options, the answer's citation) and the standing-escalation lockout
   on `contract.unblocked`.
5. `next/internal/obligation` — the `escalation.pending` kind; and
   `next/spec/obligations.md`'s fact-shaped table gains its row.
6. `next/cmd/seed` — `claim park --question/--option`, `decision record`
   with its derivation, and the affordance catalog entry.
7. `next/spec/escalation.md` — the new spec, and the forward pointer in
   `packets.md` becomes a link.
8. `next/docs/decisions.md`, `memory/LEARNINGS.md`; receipt; evidence;
   review.

## File Scope

- `next/spec/escalation.md` (new), `next/spec/transitions.json`,
  `next/spec/lifecycle.md`, `next/spec/obligations.md`,
  `next/spec/actors.md`, `next/spec/packets.md`
- `next/internal/{transition,keyring,admit,obligation}/**`
- `next/cmd/seed/**`
- `next/docs/decisions.md`, `memory/*`
- `receipts/os-f781f0da.json`

Nothing outside `next/**` except the work-product files above.

## Acceptance Criteria

1. A `claim.parked` carrying `escalation` admits, and the folded
   subject is distinguishable from an ordinarily-blocked one: the
   obligations projection emits `escalation.pending` for the first and
   not the second. A drill asserts both directions, so the kind cannot
   pass by being emitted always.
2. `escalation` with fewer than two options, with a missing `question`,
   or with a duplicate option id **refuses at admission**, naming the
   offending part. A drill plants each.
3. While an escalation stands, `contract.unblocked` refuses and the
   refusal names both the escalation's position and
   `decision.recorded`. `contract.cancelled` still admits — asserted,
   because a lockout that traps the contract is the failure this
   criterion exists to exclude.
4. `decision.recorded` admits from an operator key and **refuses from a
   dispatch key** citing capability, which is the whole of D2. A drill
   uses a real dispatch-capability actor, not an unenrolled one, so the
   refusal is attributable to the row rather than to standing.
5. `decision.recorded` citing a position that is not the standing
   escalation, or a `choice` id the escalation does not offer, refuses.
6. `seed decision record` derives the citation from the fold; with two
   standing escalations it refuses naming both candidates rather than
   choosing, and with none it refuses naming what would establish one.
7. An operator's `seed situation` lists the waiting escalation with the
   position that raised it, and the delta form (`--since`) reports its
   removal by `(subject, kind)` once answered.
8. **Mutation evidence, per fix rather than in aggregate.** Each of
   these must fail its drill: deleting the lockout branch; relaxing the
   two-option minimum; widening `decision.recorded` to accept
   `CapDispatch`; dropping the `escalation.pending` row from the
   obligations kinds.
9. `make check` green with coverage measured **cold**, at least three
   readings above the 90% gate, and the suites pass unprivileged.

## Validation Commands

```sh
cd next && gofmt -l . && go vet ./... && go build ./...
cd next && go test ./internal/... -count=1
cd next && go test ./cmd/seed/ -count=1
make check
```

## Expected diff shape

One table row, one capability row, one admission rule, one obligations
kind, two CLI surfaces, one new spec file and four amended ones, plus
drills. Roughly +700/-40 lines, all under `next/**` and the work-product
files.

## A risk worth naming now

D5 (options required, minimum two) is the decision most likely to be
wrong, and it is wrong in a way that is cheap to discover and expensive
to undo: if real escalations turn out to be open questions, the rule
converts a usable channel into an unusable one, and by then lanes will
have been written against it.

So the drill for D5 asserts the refusal, and this plan records the
alternative it rejected (an "other" escape hatch) with the reason,
rather than leaving a later reader to guess whether the constraint was
considered or merely inherited. Relaxing it later is a one-row change
to a shape rule; tightening it later is a break for every lane that
learned to send free text.
