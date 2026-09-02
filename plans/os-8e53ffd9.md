# Plan: next — Phase 10 item 1, runtime tuples in grants, reported by adapters, drift as out-of-grant (os-8e53ffd9)

The build plan's Phase 10 item 1: *"Runtime tuples in enrollment/grants;
adapters report provisioned tuple; drift = out-of-grant."* Charter III.E
row 6 is the target, and §II.5 is the normative sentence: *"Grants cite
the qualified tuple; an actor invoking a materially different
configuration than its grant cites is out of grant, and configuration
is part of what executor adapters report (§9)."* The Phase 9 exit
record (#214) routes the first half of III.J row 3 here too: *"the
planner lane receives the strongest tuples by policy."*

## What the tree actually shows

Measured, not assumed:

- **The charter fixes the tuple's five fields** (§II.5, glossary):
  principal, harness+version, model family+version, tool policy,
  environment profile. Nothing in `next/**` carries any of them.
- **`executor.Tuple` is `{Runtime string}`**, the honest v0 stub
  `local-worktree/v0`, and `Adapter.Tuple()` is **called by nothing
  outside the executor package's own tests**. So "adapters report the
  provisioned tuple" is true of the interface and false of the chain.
- **No `cmd/seed` verb emits `run.started` or `run.settled`.** Both
  reach the chain only through `ledger append`, the raw seam that runs
  no rules; the fixtures append them by hand. There is no `seed run`
  verb at all.
- **`run.started` is strictly `{fence, reservation}`** and
  `RunStartFact` is `{Pos, Signer, Fence, Reservation}`: no
  configuration anywhere on a run.
- **`actor.granted` is strictly `{capability}`** and the keyring's
  `Entry` is `{Key, Kind, Name, Standing, Root, Grants []string}`: a
  grant cannot cite anything.
- **`actor.qualified` is cataloged and refused**: `protocol.md` says it
  "cites eval results and the runtime tuple", `actors.md` says
  "undefined until qualification lands (Phase 10)", and the keyring's
  `default:` arm refuses it by name.
- **`out_of_grant` is capability absence only** (`OutOfGrantError{Actor,
  Verb, Accepted}`), and the tree already records that capability
  absence is a different property from disjointness (os-6a08b166).

## Design decisions (binding for this task)

- **D1 — the tuple is the charter's five fields, as one strict object,
  spelled once.** `{"principal": …, "harness": "<name>/<version>",
  "model": "<family>/<version>", "tool_policy": …, "environment": …}`,
  every field a non-empty string, unknown fields refused. It lives in a
  new package `next/internal/tuple` (`Tuple`, `Parse`, `Equal`) and
  `executor.Tuple` becomes that type. The local worktree adapter
  reports what it can honestly know: `harness: "local-worktree/v0"`,
  `environment: "<the workspace's provenance>"`, and the three fields
  it cannot know — principal, model, tool policy — come from the run's
  declaration (D3), never invented by the adapter.

  Refused: keeping `Runtime string` and packing five things into it.
  Drift is a per-field comparison (D4), and a string with five things
  in it cannot say which one moved.

- **D2 — grants cite the tuple; enrollment does not.** `actor.granted`
  gains an OPTIONAL `tuple` (D1's object). The keyring's `Entry` stores
  grants as `{Capability, Tuple *Tuple}` instead of `[]string`, with
  `Grants()` keeping the old string view for every existing caller.

  Optional is the bridge, and it is stated rather than smuggled: a grant
  with no tuple is an **unqualified** grant and admits exactly as today,
  so no existing chain, fixture or deployment changes meaning. A grant
  with a tuple is a **qualified** grant, and D4 applies. Phase 10 item
  2 is what makes evals mint qualified grants; this card gives the
  grant somewhere to put one.

  Enrollment stays `{key, kind, name}`. The build plan says
  "enrollment/grants"; the charter says "grants cite the qualified
  tuple" and never mentions enrollment. Qualification binds to the
  runtime, not the key (§II.5's own heading), and an enrollment is the
  key. A tuple on the enrollment would say a key IS a configuration,
  which is the conflation the charter is at pains to refuse.

- **D3 — the adapter's report reaches the chain on `run.started`,
  through a verb.** `run.started`'s payload gains a REQUIRED `tuple`.
  And there is a verb to carry it: **`seed run start --subject <id>
  --key <path> [--adapter local-worktree] [--principal …] [--model …]
  [--tool-policy …]`**, the supervisor's act, deriving the fence from
  the active window and the reservation from the shared budget view
  exactly as the loop verbs do, filling `harness` and `environment`
  from `Adapter.Tuple()`, and pre-flighting through `admit.Check` so a
  refusal carries the boundary's own error.

  Required, not optional, because a run with no declared configuration
  is a run nothing can qualify, and a spend that qualification cannot
  see is the thing III.E row 6 forbids. Existing fixtures that append
  `run.started` raw gain the field; there are eleven such appends and
  the receipt counts them.

  Refused: reporting the tuple on `run.settled`. Settlement is telemetry
  after the fact ("never authority", `executors.md`); drift must refuse
  BEFORE the spend, and the spending verb is `run.started`.

  Refused: a new `run.provisioned` verb. The adapter provisions only
  against an admitted `run.started` for its fence (`executors.md` step
  3), so the start IS the moment the configuration is committed to.

- **D4 — drift is a per-field inequality between the run's tuple and
  the CLAIM HOLDER's qualified grant, refused as `out_of_grant`.** At
  `run.started` admission: take the subject's active claim holder; take
  that holder's `claim` grant; if it cites a tuple and any of the five
  fields differs from the run's declared tuple, refuse
  `OutOfGrantError` with a new `Drift` detail naming the holder, the
  field, the cited value and the provisioned value.

  The holder, not the signer. The supervisor signs `run.started`; the
  work executes under the holder's key, and the charter's sentence is
  about "an actor invoking a materially different configuration than
  its grant cites" — the invoking actor is the one whose window the
  run spends. A drill plants the mismatch on the signer's grant and the
  match on the holder's, and must admit; and the reverse must refuse.

  "Materially different" is every field, in v0. The charter leaves
  "materially" to policy; the honest v0 has no policy and says so:
  any difference is drift, and a later card that wants tolerance (a
  patch-version bump) owns the decision. Drilled per field, five rows.

  The code stays `out_of_grant` (exit 14): the charter names this
  refusal "out of grant" in those words, and `envelope.md`'s rule is
  that a refinement inside a family keeps the family's exit.
  `OutOfGrantError` gains the `Drift` field so the message can say
  which; the wire code does not change.

- **D5 — `actor.qualified` stays undefined here.** It "cites eval
  results" (`protocol.md`), and eval results are Phase 10 item 2. This
  card gives grants a tuple field; item 2 is what mints a grant from a
  passing eval. Defining the verb now would define a fact with nothing
  to cite.

- **D6 — III.J row 3's "strongest tuples by policy" lands as a
  scheduling INPUT, not a scheduler.** `offer.published`'s eligibility
  already scopes offers by capability and tier; it gains an optional
  `tuples` list (D1 objects), and `offer list` / the loop's poll filter
  a qualified worker by it: a worker whose `claim` grant cites a tuple
  sees an offer only if its tuple is in the offer's list, or the offer
  names none. That is the whole of "by policy" the tree can honestly
  hold: the supervisor writes the policy into the offer. A policy
  engine deciding which tuples are "strongest" is item 2's eval results
  turned into offers, and is not built here.

- **D7 — the tier card (os-be12ac16) stays separate.** It is the tier
  vocabulary at `intent.filed`, which Phase 10 item 3 (levels declared
  per tier) owns; nothing in item 1 reads a tier.

- **D8 — scope guard.** No verb is added to the ledger vocabulary
  (`run.started` and `actor.granted` grow a field each; `seed run start`
  is a CLI verb over an existing ledger verb, like `claim take`). No
  transition row moves. No exit code is allocated. `actor.enrolled` is
  untouched.

## Steps

1. `next/internal/tuple/` — `Tuple`, `Parse` (strict), `Equal`,
   `Diff` (the first differing field); unit-drilled including every
   malformed shape.
2. `next/executor/executor.go` — `Tuple` becomes `tuple.Tuple`; the
   local adapter reports `harness` and `environment` and leaves the
   other three empty for the caller to fill; its test asserts it
   invents nothing.
3. `next/internal/keyring/keyring.go` — `actor.granted` accepts optional
   `tuple`; `Entry` stores `[]Grant{Capability, Tuple}`; `Grants()` keeps
   the string view; a new `GrantTuple(actor, capability) *Tuple`.
4. `next/internal/transition/transition.go` — `RunStartFact` gains
   `Tuple`; the fold reads it.
5. `next/internal/admit/admit.go` — `run.started` requires `tuple`; the
   drift check (D4) in the run rule, refusing `OutOfGrantError{…, Drift}`.
6. `next/cmd/seed/run.go` (new) — `seed run start`, deriving fence and
   reservation, filling the tuple from the adapter and flags,
   pre-flighting through `admit.Check`; `run_cli_test.go` gains its
   drills.
7. `next/internal/admit/affordances.go` — the `run.started` probe
   carries a tuple so the affordance sweep stays green.
8. Offers: `offer.published` accepts optional `tuples`; `offer list` and
   the loop's poll filter on it (D6); the modes fixture provisions its
   workers with qualified grants and proves a mismatched tuple is
   refused at `run.started` and unseen at `offer list`.
9. Every fixture that appends `run.started` raw gains the field.
10. Specs: new `next/spec/qualification.md` (D1–D6, the bridge, the
    per-field drift rule, what item 2 adds); `actors.md` (the grant
    payload, `actor.qualified` still undefined and why);
    `executors.md` (the tuple's five fields, the start carries it, the
    stub retired); `offers.md` (the `tuples` eligibility); `lanes.md`
    (the residual table's tier row unchanged — it is item 3's).
11. `next/docs/progress.md` (Phase 10 opened, item 1 recorded),
    `next/docs/decisions.md`, `memory/LEARNINGS.md`; receipt; evidence;
    review.

## File Scope

- `next/internal/tuple/**` (new), `next/executor/**`,
  `next/internal/keyring/**`, `next/internal/transition/**`,
  `next/internal/admit/**`, `next/cmd/seed/run.go` (new),
  `next/cmd/seed/run_cli_test.go`, `next/cmd/seed/offer.go`,
  `next/cmd/seed/offer_cli_test.go`, `next/cmd/seed/modes_e2e_test.go`,
  `next/internal/loop/**` (the poll filter only), and every `_test.go`
  that appends `run.started` raw
- `next/spec/qualification.md` (new), `next/spec/actors.md`,
  `next/spec/executors.md`, `next/spec/offers.md`
- `next/docs/progress.md`, `next/docs/decisions.md`, `memory/*`
- `receipts/os-8e53ffd9.json`

Nothing outside `next/**` except the work-product files above. In
particular NOT `next/spec/transitions.json` and NOT
`next/internal/envelope/**`: D8 says no row moves and no code is
allocated, and those are where a violation would land.

## Acceptance Criteria

1. A `run.started` whose tuple differs from the claim holder's
   qualified grant in any ONE of the five fields refuses `out_of_grant`
   (exit 14) naming the holder, the field, and both values; drilled per
   field, five rows, through `seed run start` against a real ledger.
2. A matching tuple admits; a holder whose `claim` grant cites no tuple
   admits any run (the bridge); a mismatch planted on the SIGNER's grant
   with a match on the holder's admits, and the reverse refuses — so the
   check is shown to read the holder.
3. `seed run start` fills `harness` and `environment` from
   `Adapter.Tuple()`, never invents principal, model or tool policy,
   refuses (usage) when a required field is neither supplied nor
   derivable, and pre-flights so the envelope carries the boundary's
   own refusal beside the caller's affordances.
4. `actor.granted` with a malformed tuple (a missing field, an empty
   string, an unknown field, a non-string) fails verification at its
   position as `bad_actor_event`, the chain-validity posture
   `actors.md` gives payload shapes; a grant with no tuple folds as
   today.
5. `offer list` hides an offer naming tuples from a qualified worker
   whose tuple is not among them, shows it to a worker whose tuple is,
   and shows an offer naming none to every eligible worker; the loop's
   poll agrees with the listing.
6. `actor.qualified` is still refused by name, and the refusal message
   still points at item 2.
7. **Mutation evidence.** Each must fail a drill: drift comparing
   against the signer instead of the holder; the comparison skipping
   any one field; `tuple` made optional on `run.started`; the adapter
   inventing a model; `Grants()` losing a qualified grant from the
   string view; the offer filter passing a tuple-naming offer to a
   non-listed worker; and the drift refusal emitted under a code other
   than `out_of_grant`.
8. `make check` green with coverage measured **cold**, at least three
   readings above the gate, and the suites pass **unprivileged** under
   `setpriv --reuid=65534`.

## Validation Commands

```sh
cd next && gofmt -l . && go vet ./... && go build ./...
cd next && go test ./internal/tuple/ ./internal/keyring/ ./internal/admit/ ./executor/ ./cmd/seed/ -count=1
cd next && go test ./... -count=1
make check
```
