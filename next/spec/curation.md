# Curation: the staged learning pipeline's ledger half

> Design authority: SEED-NEXT.md §II.12 ("A poisoning-resistant
> pipeline": trajectories are untrusted inputs; the pipeline is staged,
> with distinct storage and distinct gates between stages; workers
> append candidate observations only, and promotion authority is not
> grantable to implementing lanes; dead ends record failure condition
> and environment; hypotheses carry applies-when conditions, support
> from more than one non-failed trajectory and more than one actor
> where the family allows it, exceptions and provenance; validation
> runs an adversarial evaluation for behavior-changing lessons;
> conflicting evidence is a first-class contested state, never
> averaged, and contested lessons do not surface) and conformance
> III.K rows 1, 2, 3 and 5 and III.I row 5. Build plan Phase 11 items
> 1 and 2; plans `plans/os-f30ee0d3.md` and `plans/os-96850e5a.md`.
> Implemented by `internal/curation` (the shapes, the id derivation,
> the predicate, the gate registry, the fold, the surfacing set, the
> lint), the keyring's `curate` capability and its rows, the admission
> rule `curation`, the `knowledge` projection, `seed knowledge`, the
> bound eval marker and the claim-time delivery.

Before this spec the catalog named four `curation.*` verbs and the tree
implemented none; the curator held no grant; and stage one already
existed unnamed, as the packet's `findings`. This spec names the
stages, gives three of them a ledger fact with its own gate, and puts
the proposal behind a grant nothing implementing can hold.

## The four stages

| stage | storage | fact | who appends |
|---|---|---|---|
| observations | the ledger, on the contract | the packet's `findings` ([`packets.md`](packets.md)) and **`curation.deadend.recorded`** | the window's holder (`claim`), `operator` |
| hypotheses | the ledger, on a hypothesis subject | **`curation.hypothesis.proposed`**, **`curation.hypothesis.contested`** | **`curate`** alone, no operator fallback |
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
duplicate rule, the contest and the promotion read admission, never
presence); and a raw-pushed pass clears no authenticated fail.

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
derived from the claim AND its exceptions (the first twelve hex digits
of the SHA-256 of the whitespace-normalized claim text joined with the
sorted, normalized exceptions, `curation.HypothesisID`), carries
`{"claim", "applies_when": {…}, "support": ["<contract>@<position>",
…], "exceptions": [...], "provenance": ["<path @ commit>", …]}`. A
subject not derived from the claim and exceptions refuses; one claim
with one exception set derives one subject, so a re-proposal that
changes nothing refuses as a duplicate at the boundary rather than
accumulating (the earliest proposal that passed the grant and the
support is the admitted one; a raw-pushed proposal that passed no
boundary reserves nothing, `curation.AdmittedProposalBefore`), and one
that adds an exception (the road out of a contest, below) is a new
subject.

**The predicate.** `applies_when` is an object over record-derivable
fields, at least one of `routing` (equal to the subject's intent
routing, which the fold now reads), `tier` (equal to the subject's
tier) and `paths` (a non-empty list; one entry must prefix the
subject's acceptance ref path); present fields conjoin; an empty
object, an unknown field, an empty `paths` or a non-string value
refuses naming the part. One implementation (`curation.AppliesWhen`,
`Selects`) serves the boundary, the lint and the delivery, so a
claim-time match is evaluated from the fold alone.

**The support floor**: at least two citations, each an admitted
observation (a deliberate exit whose packet carries at least one
finding, or a dead end, signed by the window's holder and citing the
fence active at its own prefix), on at least two distinct contracts,
none of which stands failed: its latest AUTHENTICATED verdict a fail
(`curation.FailedAt`: a verdict counts when its signer held `verdict` at
the verdict's own position and was neither the bound submission's
signer nor a claimant, past or present, so a raw-pushed pass by an
implementing or grantless key clears nothing). That is the
structural minimum the charter makes non-promotable to skip: a worker
promoting its own single run is refused by construction.

**The actor arm.** The holders of the cited observations must be two
or more distinct keys **where the family allows it**: the family is
the set of contracts the applies-when selects in the record, computed
from the selection and never from the support set, and it allows
when those contracts have two or more distinct holders over the chain
(the current holder and every prior claimant, both record facts). A
family with one holder admits support from that one holder, and the
fold records `single_actor_family` on the hypothesis so the projection
says why the arm was not applied; a family with two holders refuses a
two-contract support from one of them naming the arm.

**`curation.hypothesis.contested`**, on the hypothesis subject,
carries `{"hypothesis": "<h-id>@<position>", "evidence":
["<contract>@<position>", …], "reason"}`: the curator's attributable
judgment that held-out evidence contradicts the claim, never an
average. The boundary requires the cited hypothesis to be an admitted
proposal on this subject, and each evidence to be an admitted
observation on a contract the applies-when selects that is NOT in the
support set: held-out evidence, by construction. The fold moves the
hypothesis to `contested` from `proposed` or `promoted`; a contested
hypothesis refuses `lesson.promoted`, and a promoted lesson whose
hypothesis is contested stops surfacing while its file and every fact
remain, which is "evidence kept". A contested hypothesis is never
edited or averaged back: the road out is a new proposal whose claim or
exceptions differ, citing the counter-evidence as an exception and
judged on its own support. `seed knowledge validate --hypothesis
<h@position>` is the held-out listing: the observations on selected
contracts outside the support set, for the curator to judge, then
contest or promote. **The residual, stated:** no machine decides that
an outcome contradicts a claim; a contest is a curator's attributable
judgment over evidence the record already holds.

**`curation.lesson.promoted`**, on the hypothesis subject, carries
`{"lesson": "next/knowledge/lessons/<id>.md @ <commit>", "hypothesis":
"<h-id>@<position>", "pr": "<pr> @ <merged-commit>", "carrier",
"adversarial": {"eval": "<name>", "verdict": "<position>"},
"last_validated", "expires", "digest"}`: anchored paths, the lesson
under the one store (a clean relative path prefixed by
`next/knowledge/lessons/`, `curation.UnderLessonsDir`: a path that
climbs out through `..` matches the anchor grammar and refuses at
`promotion.shape`), the cited hypothesis this subject, `carrier` one
of `knowledge`, `role`, `skill`, `workflow`, `harness` (where the
lesson lands), the two stamps RFC3339 with `expires` after
`last_validated` (so expiry is derivable from the fold without the
file), and `digest` the sha256 of the lesson file's bytes at the
anchor, which every reader verifies before surfacing.

**The promotion gate has a ledger half and a file half.** The ledger
half, at admission: the cited hypothesis is an admitted proposal on
this subject (re-judged from the record: the signer held `curate`
there, the support passed there, the claim was not already proposed)
and not contested at the tip; its support still satisfies the arms;
`carrier` is a member; the stamps are present and ordered; and
`adversarial` is REQUIRED for every carrier: its verdict is an
authenticated pass, replayed at its own position, on a subject whose
`eval` marker names that eval AND binds it to this hypothesis and to
this lesson anchor ([`evals.md`](evals.md)), filed after the
hypothesis's position. Every promoted lesson is behavior-changing,
because promotion is what puts it in front of a worker at claim time;
a `knowledge` carrier says where the lesson lands, never that it is
harmless, so no carrier is exempt. The file half, `seed knowledge
lint <file> --ledger <dir> --repo <dir> [--now <RFC3339>]`: the
frontmatter names its hypothesis, applies-when, support, provenance,
`last-validated`, `expires` and `carrier`
(`next/knowledge/lessons/README.md`) and agrees with the fact and the
hypothesis; the file's bytes at the anchor hash to the fact's
`digest` and the anchor commit is an ancestor of the repository's
head (the promotion PR merged); every provenance anchor resolves in
the repository at its commit; `last-validated` is not after the
declared instant and `expires` is after it. A drill applies the
presence lint to every file in the store, so `make check` refuses a
shipped lesson that fails it: the PR gate the charter routes promotion
through, on the surface the tree already gates that way.

**The gate registry.** Every refusal of the dead end's shape and
holder checks, of the proposal, contest and promotion rules and of the
lint's file half is a `curation.GateError` naming a gate registered in
`curation.Gates()` at init (`<rule>.<part>`), and the constructor
refuses an unregistered name. The registry is the one authority the
poisoning drill (item 3) enumerates to derive its coverage, pinned to
this table in both directions, so a gate added here without a poison
there is a red test rather than a silent gap.

| gate | what it checks |
|---|---|
| `applies_when.shape` | the predicate names at least one record-derivable field, each well-formed |
| `contest.evidence` | every evidence citation is an admitted observation |
| `contest.held_out` | evidence lies outside the support set |
| `contest.hypothesis` | the contest cites an admitted hypothesis on its own subject |
| `contest.selected` | evidence lies on a contract the applies-when selects |
| `contest.shape` | the contest's payload shape and fields |
| `deadend.holder` | the dead end is the window holder's own |
| `deadend.shape` | the dead end's payload shape and fields |
| `lint.ancestry` | the anchor commit is an ancestor of the repository's head |
| `lint.applies_when` | the frontmatter's applies-when parses and equals the hypothesis's |
| `lint.carrier` | the frontmatter's carrier equals the fact's |
| `lint.digest` | the file's bytes at the anchor hash to the fact's digest |
| `lint.frontmatter` | the lesson file opens with the frontmatter block and its keys |
| `lint.hypothesis` | the frontmatter cites the fact's hypothesis |
| `lint.provenance` | every provenance anchor resolves in the repository at its commit |
| `lint.stamps` | last-validated is not after the declared instant and expires is after it |
| `lint.support` | the frontmatter's support equals the hypothesis's |
| `promotion.adversarial` | the adversarial evaluation is an authenticated pass bound to this hypothesis and this lesson anchor, filed after the hypothesis |
| `promotion.carrier` | the carrier is a member |
| `promotion.contested` | a contested hypothesis is not promotable |
| `promotion.digest` | the digest is the lesson file's sha256 |
| `promotion.hypothesis` | the promotion cites an admitted hypothesis on its own subject |
| `promotion.shape` | the promotion's payload shape and fields |
| `promotion.stamps` | last_validated and expires are RFC3339 and ordered |
| `promotion.support` | the hypothesis's support still satisfies the arms at promotion |
| `proposal.shape` | the proposal's payload shape and fields |
| `proposal.subject` | the subject is derived from the claim and its exceptions |
| `support.actors` | two distinct holders where the family allows it |
| `support.duplicate` | one claim is proposed once |
| `support.failed` | no cited contract stands failed |
| `support.floor` | at least two observations on two distinct contracts |
| `support.observation` | every citation is an admitted observation |


## Delivery

The surfacing set (`curation.Surfacing`) is every promoted lesson
whose hypothesis is not contested, whose applies-when selects the
subject, AND whose fact resolves in the repository the reader holds:
the anchor commit is an ancestor of the repository's head (the
promotion PR merged) and the file's bytes at the anchor hash to the
fact's `digest`. A fact that does not resolve is reported as
`lessons_unresolved` (anchor and reason) and never surfaces, so an
attested promotion naming a missing, unmerged or altered file reaches
no worker; `seed reconcile` classifies the same facts
`lesson_unverified` at evidence grade
([`reconciliation.md`](reconciliation.md)). Three surfaces carry the
set from that one derivation: the envelope of `claim take --repo
<dir>` gains `lessons` (present on every claim, empty when nothing
matches; without `--repo` nothing surfaces and `lessons_unverified`
counts what a repository would have verified,
[`loop-verbs.md`](loop-verbs.md)); the provisioned handoff, where
`Provision` writes the set to `.seed-run/lessons.json` beside
`packet.json` ([`executors.md`](executors.md)); and the orienting
read, `seed situation --repo <dir>`, which lists the same rows per
held subject and without `--repo` reports only the count as
`lessons_unverified`, naming the flag. Contested lessons appear on
none of them.

## The adversarial evaluation

A counter-trajectory is an eval definition under `next/evals/<name>/`
whose fixture is the situation the lesson claims to improve,
constructed so that the known verdict fails if the lesson is wrong.
It is filed with `seed eval file --for-lesson <h@position> --carrier
<path @ commit>`, and the filing's marker grows to `{"name",
"tuple"?, "lesson", "carrier"}` ([`evals.md`](evals.md)): the
hypothesis and the exact candidate revision the eval is for, on the
record at filing. It is worked in a workspace that includes the
carrier revision and judged by the verifier, and `seed verdict render`
on a subject whose marker names a carrier refuses `carrier_absent`
(under `checks_red`) unless the carrier commit is an ancestor of the
submission head: the lesson was actually applied. The promotion
boundary then requires the cited verdict's subject to carry a marker
whose `lesson` equals the promoted hypothesis and whose `carrier`
equals the promoted `lesson` anchor, so a pass on an eval filed for
another hypothesis, another revision, or nothing in particular is not
survival. Nothing else runs: the machinery is Phase 10 item 2's, and
no definition is shipped for this; the drills build theirs.

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

The curator's reachable set at the boundary is the proposal, the
contest, the raise, and `message.sent`, which any enrolled active key
appends (the relay the injection suite names for the dispatcher); the
injection suite derives the set from `admit.Affordances` and pins it.

## Versioning

The four verbs are catalog growth under an existing namespace, the
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
  <json> --support <contract>@<position>… [--provenance <anchor>]…
  [--exception <text>]…` — the curator's proposal, the subject derived
  from the claim and the exceptions (never given); fewer than two
  citations or a predicate that is no object refuse at usage naming
  the part.
- `seed knowledge validate (--ledger <dir> | --remote <repo>)
  --hypothesis <h-id>@<position>` — the held-out listing.
- `seed knowledge contest … --key <path> --hypothesis
  <h-id>@<position> --evidence <contract>@<position>… --reason <text>`
  — the curator's contest.
- `seed knowledge promote … --key <path> --lesson <path @ commit>
  --hypothesis <h-id>@<position> --pr <pr @ commit> --repo <dir>
  --carrier <c> --adversarial <eval>@<verdict position>
  --last-validated <RFC3339> --expires <RFC3339>` — the observer's
  promotion, the subject the cited hypothesis's, the digest read from
  the file at its anchor in the repository (never typed).
- `seed knowledge lint <file> (--ledger <dir> | --remote <repo>) --repo
  <dir> [--now <RFC3339>]` — the gate's file half; a refusal is
  `lint_refused` naming the gate.
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
- III.K row 3 (promotion requires applies-when conditions; support
  from more than one non-failed trajectory and more than one actor
  where the family allows; provenance links; a last-validated stamp;
  and, for behavior-changing lessons, survival of an adversarial
  evaluation against constructed counter-trajectories): the predicate,
  the actor arm, the promotion gate's two halves and the bound eval,
  drilled per refusal at the boundary and through `seed knowledge`.
- III.K row 5 (conflicting evidence is a first-class contested state,
  never silently averaged; contested lessons do not surface): the
  contest, the fold's stage and the surfacing set's exclusion,
  drilled at the boundary, in the projection and on every delivery
  surface.
- III.I row 5 (knowledge shown at the right moment): the surfacing set
  on `claim take`, in the provisioned handoff and in the orienting
  read, verified against the repository before anything surfaces.
