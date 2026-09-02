# Curation: the staged learning pipeline's ledger half

> Design authority: SEED-NEXT.md §II.12 ("A poisoning-resistant
> pipeline": trajectories are untrusted inputs; the pipeline is staged,
> with distinct storage and distinct gates between stages; workers
> append candidate observations only, and promotion authority is not
> grantable to implementing lanes; dead ends record failure condition
> and environment) and conformance III.K rows 1 and 2. Build plan Phase
> 11 item 1; plan `plans/os-f30ee0d3.md`. Implemented by
> `internal/curation` (the shapes, the id derivation, the fold), the
> keyring's `curate` capability and its rows, the admission rule
> `curation`, the `knowledge` projection, and `seed knowledge`.

Before this spec the catalog named four `curation.*` verbs and the tree
implemented none; the curator held no grant; and stage one already
existed unnamed, as the packet's `findings`. This spec names the
stages, gives three of them a ledger fact with its own gate, and puts
the proposal behind a grant nothing implementing can hold.

## The four stages

| stage | storage | fact | who appends |
|---|---|---|---|
| observations | the ledger, on the contract | the packet's `findings` ([`packets.md`](packets.md)) and **`curation.deadend.recorded`** | the window's holder (`claim`), `operator` |
| hypotheses | the ledger, on a hypothesis subject | **`curation.hypothesis.proposed`** | **`curate`** alone, no operator fallback |
| validated lessons | `next/knowledge/lessons/<id>.md`, merged by PR, and the ledger | **`curation.lesson.promoted`** (the PR observation) | `observer`, `operator` |
| policy | role, skill and workflow patches on the protected surface, by PR | none in this item | the governance root, through gates |

**No stage skips, by citation.** A hypothesis cites the observations it
rests on and the boundary re-judges each from the record; a promotion
cites the hypothesis it promotes and the boundary re-judges that the
citation passed. A lesson that cites no admitted hypothesis folds
`unbound`, never as a lesson. Fold presence is never proof of admission
(the `RunStartValid` posture), and neither is a tolerant lifecycle fold:
a raw-pushed dead end outside a window supports nothing; a dead end
inside a window that a grantless key's raw `claim.taken` opened supports
nothing either, since the window is re-judged at its fence (the claim on
the subject, signed by its holder, who held `claim` at that prefix:
`curation.WindowAdmitted`); a raw-pushed proposal neither promotes nor
reserves its subject (an unadmitted proposal folds as an anomaly, so the
duplicate rule reads admission, never presence); and a raw-pushed pass
clears no authenticated fail.

Refused: a lessons store in the ledger. A lesson is a document humans
and lanes read, it changes behavior when it lands, and the charter
routes promotion through a PR precisely so that rollback is a revert;
the ledger carries the observation of that merge, the `plan.approved`
posture. Refused here: `lesson.retired`; retirement and expiry are item
4's, and a verb with no reader is a verb nobody drills.

## The facts

**`curation.deadend.recorded`**, on a contract, carries the strict
object `{"fence", "tried", "outcome", "condition", "environment",
"pointer"?}`: the packet finding's shape plus the charter's failure
condition and environment. Every field non-empty; the fence the active
claim window's; the pointer, if any, an anchored path (`"<path> @
<commit>"`, never a bare path). Admitted only from the window's holder
inside its window: outside a window, from a non-holder, or citing a
stale fence it refuses naming the part. It is a candidate and nothing
more, since it has no field a conclusion could live in.

**`curation.hypothesis.proposed`**, on subject **`h-<12 hex>`**
derived from the claim (the first twelve hex digits of the SHA-256 of
the whitespace-normalized claim text, `curation.HypothesisID`), carries
`{"claim", "applies_when", "support": ["<contract>@<position>", …],
"exceptions": [...], "provenance": ["<path @ commit>", …]}`. A subject
not derived from the claim refuses; one claim derives one subject, so a
re-proposal of an admitted claim refuses as a duplicate at the boundary
rather than accumulating (the earliest proposal that passed the grant
and the support is the admitted one; a raw-pushed proposal that passed
no boundary reserves nothing, `curation.AdmittedProposalBefore`). **The support floor**: at least two citations, each an
admitted observation (a deliberate exit whose packet carries at least
one finding, or a dead end, signed by the window's holder and citing
the fence active at its own prefix), on at least two distinct
contracts, none of which stands failed: its latest AUTHENTICATED
verdict a fail (`curation.FailedAt`: a verdict counts when its signer
held `verdict` at the verdict's own position and was neither the bound
submission's signer nor a claimant, past or present, so a raw-pushed
pass by an implementing or grantless key clears nothing).
That is the structural minimum the charter makes non-promotable to
skip: a worker promoting its own single run is refused by
construction. The full promotion gate (more than one actor where the
family allows, provenance, last-validated, the adversarial evaluation,
the contested state) is item 2's, on top of this shape.

**`curation.lesson.promoted`**, on the hypothesis subject, carries
`{"lesson": "next/knowledge/lessons/<id>.md @ <commit>", "hypothesis":
"<h-id>@<position>", "pr": "<pr> @ <merged-commit>"}`: anchored paths
all three, the lesson under the one store (a clean relative path
prefixed by `next/knowledge/lessons/`, `curation.UnderLessonsDir`: a
path that climbs out through `..` matches the anchor grammar and
refuses, or the promotion would name a file the store's lint never
sees), the cited hypothesis this subject. It admits only citing an admitted proposal at that position:
the citation is re-judged from the record (the signer held `curate`
there, the support passed there, the claim was not already proposed).
The lesson file's frontmatter names its hypothesis, applies-when,
support, provenance, `last-validated` and `expires`
(`next/knowledge/lessons/README.md`); item 1 lints the keys' presence
(`curation.Lint`, applied to every file in the store by drill), item 2
their content.

## The grant

`curate` accepts `curation.hypothesis.proposed` and nothing else does:
the fifth no-fallback row, beside `verdict.rendered`, `check.sealed`,
`decision.recorded` and `merge.overridden`. Governance roots hold
`operator` implicitly, and `operator` already reaches `claim.taken`,
the deliberate exits and `curation.deadend.recorded`, so an operator
fallback on the proposal would let one key write a trajectory's
observations and then conclude from them without ever holding two
conflicting grants: the poisoning boundary is real only if the
proposal is reachable through `curate` alone.

`curate` cannot be granted to a key holding `claim` or `operator` (a
root's implicit operator standing included), nor `claim` or `operator`
to a key holding `curate`: the sealer rule one capability over, against
both lanes it names, chain validity at the position. A worker promoting
its own runs, and a root concluding from its own, are refused at the
grant, not at the proposal. The curator manifest holds `curate` and
nothing else; `escalation.raised` accepts it too, the charter's "any
lane can raise" reaching the one lane it did not
([`escalation.md`](escalation.md)).

The curator's reachable set at the boundary is the proposal, the raise,
and `message.sent`, which any enrolled active key appends (the relay
the injection suite names for the dispatcher); the injection suite
derives the set from `admit.Affordances` and pins it.

## Versioning

The three verbs are catalog growth under an existing namespace, the
`offer.published` precedent: an older validator refuses the unknown
verb safely, admission policy shapes them, the fold counts a malformed
raw fact as an anomaly, and every existing chain verifies byte for
byte. No protocol bump.

## Surfaces

- `seed knowledge deadend (--ledger <dir> | --remote <repo> [--ref
  <ref>] [--state <dir>]) --key <path> --subject <id> --tried <text>
  --outcome <text> --condition <text> --environment <text> [--pointer
  <anchor>]` — the holder's dead end, the fence derived from the active
  window like the loop verbs; no window refuses `invalid_transition`.
- `seed knowledge propose … --key <path> --claim <text> --applies-when
  <text> --support <contract>@<position>… [--provenance <anchor>]…
  [--exception <text>]…` — the curator's proposal, the subject derived
  from the claim (never given); fewer than two citations refuse at
  usage naming the floor.
- `seed knowledge promote … --key <path> --lesson <path @ commit>
  --hypothesis <h-id>@<position> --pr <pr @ commit>` — the observer's
  promotion, the subject the cited hypothesis's.
- `seed knowledge show (--ledger <dir> | --remote <repo>)` — the fold:
  the stage counts, dead ends by contract, hypotheses with their stage
  (`proposed`, `promoted`), lessons, the unbound promotions.
- The `knowledge` projection (`knowledge.json`) publishes the same view
  ([`projections.md`](projections.md)); the report carries a
  `knowledge` section counting the stages when the chain holds any
  curation fact.

## Conformance mapping

- III.K row 1 (online lanes append evidence only; conclusion-writing is
  grant-gated to the curator's proposal path; workers append candidate
  observations, never promoted lessons): the dead end's window rule and
  the proposal's `curate`-only row with its grant-level disjointness,
  drilled per refusal at the boundary and through `seed knowledge`.
- III.K row 2 (the pipeline is staged with distinct storage and gates;
  no stage skips): the four stages above, the support floor, the
  promotion's re-judged citation and the `unbound` fold, drilled at the
  boundary and in the projection.
