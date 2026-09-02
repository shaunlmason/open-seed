# Qualification

The runtime tuple and what binds to it (SEED-NEXT.md §II.5 and glossary;
charter III.E row 6; docs/next-build-plan.md Phase 10 item 1;
plans/os-8e53ffd9.md). "What passes an eval is not an agent, it is a
configuration tuple: principal, harness and version, model family and
version, tool policy, environment profile." Grants cite one, adapters
report one, and an actor invoking a materially different configuration
than its grant cites is **out of grant**.

This spec is item 1's half of Phase 10: the tuple, where it lives, and
the drift rule. Item 2 (eval contracts) is what mints a qualified grant
from a passing eval and defines `actor.qualified`; until then grants
are qualified by the operator's hand, exactly as they are granted.

## The tuple

One strict JSON object, spelled once in `internal/tuple`:

```json
{
  "principal": "acme",
  "harness": "local-worktree/v0",
  "model": "fable/5.1",
  "tool_policy": "default",
  "environment": "detached-git-worktree"
}
```

Every field is a non-empty string; unknown fields refuse, so a
misspelling cannot pass as an absent field; `null` is not a tuple.
`harness` and `model` carry their versions as `<name>/<version>` and
`<family>/<version>` by convention, which the parser does not police:
what it polices is presence, because drift is a per-field comparison
and a missing field is a field that cannot drift.

Refused: one string with five things in it. Drift must say WHICH field
moved, and a packed string cannot.

## Activation: the `seed/2` boundary

Everything below activates for records at positions under protocol
version `seed/2`, reached via `system.protocol.upgraded` from `seed/1`
([`protocol.md`](protocol.md)'s register). The bump is forced by
[`actors.md`](actors.md)'s own posture: actor payload shapes are chain
validity, strictly decoded, so a `seed/1` validator fails a grant
carrying `tuple` at its position (`bad_actor_event`) while this
validator accepts it, which is exactly a validation rule the two would
judge differently.

At `seed/1` positions nothing changes: a grant's `tuple` field stays
unknown-and-refused under this build too, `run.started` carries the
strict `{fence, reservation}` and refuses a `tuple`, an offer's
`tuples` scope refuses, and every existing chain verifies byte for
byte. A build supporting only `seed/1` refuses a chain that upgraded at
the first `seed/2` record, by version (`version_unsupported`, in the
`version_mismatch` exit family), never by misjudging a grant.

## Where the tuple lives

- **Grants cite it.** `actor.granted` gains an OPTIONAL `tuple`
  ([`actors.md`](actors.md)). Enrollment stays `{key, kind, name}`:
  qualification binds to the runtime, not the key, and an enrollment
  is the key.
- **Starts declare it.** `run.started` gains a REQUIRED `tuple` at
  `seed/2` ([`executors.md`](executors.md)): a run with no declared
  configuration is a run nothing can qualify, and a spend that
  qualification cannot see is what III.E row 6 forbids. `seed run
  start` is the supervisor's verb for it.
- **Adapters report it, twice.** `Adapter.Tuple()` is the static,
  partial report before any provision: the fields the adapter
  controls. `Run.Tuple()` is what the adapter RESOLVED, and `Provision`
  compares it to the admitted declaration before any execution is
  released ([`executors.md`](executors.md)).
- **Offers may name it.** `offer.published`'s eligibility gains an
  optional `tuples` list ([`offers.md`](offers.md)): the supervisor
  writes the configurations it wants into the offer, which is the whole
  of "strongest tuples by policy" the tree can honestly hold until item
  2's eval results exist to rank them.

## The set rule, and the bridge

Grants only accumulate: the tree has no grant-level withdrawal, and
suspension and re-enrollment preserve grants. So the qualification rule
reads the SET. At `run.started`, take every tuple cited by the CLAIM
HOLDER's `claim` grants:

- **empty**: the holder is **unqualified**. This is the bridge, stated
  rather than smuggled: an actor none of whose grants cite a tuple
  admits exactly as it did before `seed/2`, so no existing chain,
  fixture or deployment changes meaning.
- **non-empty**: the declared tuple must EQUAL one member, per field.
  Otherwise the holder is invoking a configuration its grant does not
  cite: drift.

Three consequences, each drilled. The FIRST qualified grant makes
qualification authoritative for that actor, with no migration verb and
no withdrawal: the bridge grant is not removed, it simply stops being
the only thing consulted. A second qualified grant ADDS an admissible
configuration rather than replacing the first. A tuple-less grant
appended after a qualified one changes nothing, because the set is
what is read.

Refused: retiring the bridge by a global switch. A deployment
qualifies actors one at a time as evals pass, and a switch that flips
everyone at once would leave every not-yet-evaluated actor unable to
run.

## Drift is per field, against the holder, and out of grant

"Materially different" is every field, in v0. The charter leaves
"materially" to policy; the honest v0 has no policy and says so: any
difference on any of the five fields is drift, and a later card that
wants tolerance (a patch-version bump, say) owns that decision.

**The holder, not the signer.** The supervisor signs `run.started`; the
work executes under the holder's key, and the charter's sentence is
about the actor INVOKING the configuration, which is the one whose
window the run spends. A mismatch on the signer's own grants is
irrelevant; a mismatch on the holder's refuses whoever signs.

**The code is `out_of_grant`, exit 14.** The charter names this refusal
in those words, and [`envelope.md`](envelope.md)'s rule is that a
refinement inside a family keeps the family's exit. The message names
the holder, the field that moved, the declared value and the cited
set; no wire code is allocated.

## Declaration versus report

`run.started` carries the caller's DECLARATION, admitted before
provisioning; on its own that is a promise. The adapter's report is a
second, post-provision value: `Provision`, which already replays the
chain to find the admitted start for the spec's fence, compares the
tuple it RESOLVED against the admitted one field by field and refuses
`ErrTupleMismatch` with full rollback on any difference, so no
workspace is ever handed to a caller for a configuration the ledger
did not admit. The check is the adapter's, inside `Provision`, because
that is the one place the resolved value exists and execution has not
yet started.

The v0 local adapter is honest about its limit: it provisions a
worktree, and a worktree cannot see which model a lane process will
call. It resolves `harness` (`local-worktree/v0`) and `environment`
(`detached-git-worktree`) from what it built and takes the other three
from the declaration. A container or cloud adapter (Phase 13) resolves
all five and inherits the check unchanged.

Refused: reporting the tuple on `run.settled`. Settlement is telemetry
after the fact; drift must refuse BEFORE the spend, and the spending
verb is `run.started`. Refused: a new `run.provisioned` verb. The
adapter provisions only against an admitted start for its fence, so
the start IS the moment the configuration is committed to.

## Surfaces

- `seed run start (--ledger <dir> | --remote <repo> [--ref <ref>]
  [--state <dir>]) --key <path> --subject <id> [--adapter
  local-worktree] [--principal <p>] [--model <family>/<version>]
  [--tool-policy <t>] [--as <fingerprint>]` — the supervisor's start:
  derives the fence from the active window and the reservation from
  the shared budget view exactly as the loop verbs do
  ([`loop-verbs.md`](loop-verbs.md)), fills `harness` and
  `environment` from the adapter's static report, and never invents
  the three fields an adapter cannot know: at `seed/2` a missing one
  refuses as usage naming its flag; on a `seed/1` chain the three
  flags refuse as usage naming the version, and a bare start admits as
  before. Pre-flights through the same `admit.Check` admission
  enforces, so drift reaches the caller as the boundary's own
  `out_of_grant` beside its affordances, with nothing signed. The
  success names the reservation the run spends under and the tuple it
  declared.
- `seed offer publish ... [--tuple <json>]...` and `seed offer list`:
  [`offers.md`](offers.md).

## What item 2 adds

`actor.qualified` stays cataloged and undefined here: it "cites eval
results" ([`protocol.md`](protocol.md)), and eval results are item 2.
Item 2 mints a qualified grant from a passing eval against a fixture
repository through the production machinery, schedules spot-checks,
and suspends on failure. This spec gives the grant somewhere to put the
tuple, the start somewhere to declare it, and the boundary the rule
that compares them.

## Conformance mapping

- III.E row 6 (an actor invoking a materially different configuration
  than its grant cites is out of grant): the set rule and the per-field
  drift refusal, drilled per field through `seed run start` against a
  real ledger and at the admission boundary; the resolved-versus-
  admitted check in `Provision`.
- III.J row 3's "strongest tuples by policy": the offer's `tuples`
  scope as the scheduling input; the metrics half of the row is Phase
  10 item 5's.
