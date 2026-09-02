# Evals: synthetic work with a known verdict

> Design authority: SEED-NEXT.md §5 (qualification binds to the runtime
> tuple; spot-check evals re-test active tuples; failing tuples get
> grants suspended by the supervisor, attributably), §16 (eval
> contracts: synthetic work with known verdicts, fixture repos, run
> through the production machinery; passing gates grants for the
> *tuple* that ran them; scheduled spot-checks); conformance III.E
> ("scheduled spot-check evals re-test active tuples; failures suspend
> grants attributably without operator ceremony") and III.O ("eval
> contracts run through production machinery against fixture repos and
> gate tuple qualification; spot-checks run scheduled"). Build plan
> Phase 10 item 2; plan `plans/os-03e47abb.md`. Implemented by
> `internal/eval`, the two verbs in `internal/keyring`, the
> qualification rule in `internal/admit`, and `seed eval`.

Item 1 ([`qualification.md`](qualification.md)) gave grants somewhere to
put the runtime tuple, the start somewhere to declare it, and the
boundary the rule that compares them. It left the set of cited tuples
to be written by hand. This spec is what writes it: an **eval** is an
ordinary contract whose verdict is known in advance, and the
consequence of that verdict is a grant or its withdrawal for the
configuration that ran it. Nothing here is a second lifecycle. An eval
is filed, offered, claimed, run, submitted and judged by the same verbs
and the same rules as any contract, and the one thing added is what the
verdict *means*.

## A definition

A definition lives at `next/evals/<name>/` in this repository:

- `eval.json` — `{"name", "summary", "tier", "acceptance"}`; `name`
  MUST equal the directory, and `acceptance` is the repository-relative
  path of the acceptance spec, which MUST live under the definition's
  `fixture/`.
- `fixture/` — the files the eval is worked in: the acceptance spec
  ([`acceptance.md`](acceptance.md)'s executable form) and whatever it
  checks.
- `solution/` — the reference solution: the files a correct submission
  changes, laid over `fixture/`. It MUST be non-empty.

A definition that does not say what it is files nothing: `eval.Load`
refuses a misnamed, tier-less, summary-less, misplaced-acceptance or
solution-less definition by name. `seed eval list --repo <dir>` renders
what loads and whether each is at a reviewed revision.

**The anchor is read from the repository, never declared.** A
definition is filed at its last reviewed revision: the last commit
touching its directory, whose subject carries the merged pull request
number a squash merge leaves (`… (#N)`). The filing's acceptance `ref`
is `<acceptance path> @ <that commit>` and its `gate` is
`pr/N @ <that commit>`: the same commit on both sides, which is what
[`acceptance.md`](acceptance.md)'s equality rule demands, and both are
facts about a review that happened rather than assertions. A definition
whose last commit carries no pull request number, or which has
uncommitted changes under its directory, is **not at a reviewed
revision** and refuses `ungated` (exit 18): armed content that has not
been through the gate runs nowhere.

**The known verdict is proven, never asserted.** `seed eval check
--repo <dir> --eval <name>` takes a verifier workspace at the anchor
([`verdicts.md`](verdicts.md)'s clean per-run checkout, removed either
way), runs the acceptance commands through the verifier's own runner
and requires them RED, overlays `solution/` onto `fixture/` and runs
them again and requires them GREEN. An eval whose unsolved fixture
already passes decides nothing and refuses **`eval_vacuous`**, a
refinement of exit 19 `spec_unrunnable` ([`envelope.md`](envelope.md)):
the family's answer, "this spec cannot decide", is already right, and
the extra word says which way. A solution that stays red cannot
reproduce the known verdict and refuses `checks_red` (exit 20). An
acceptance spec with no parseable commands refuses `spec_unrunnable` as
it would at render.

## An eval is an ordinary contract

`seed eval file (--ledger <dir> | --remote <repo> --state <dir>) --repo
<dir> --eval <name> --key <path> [--tuple <json>]` appends two records:

- `intent.filed` with the ordinary `intent`, `tier` (the definition's),
  `budget` and `routing`, plus the marker **`eval`**:
  `{"name": "<definition>"}`, or `{"name", "tuple": {…}}` when the
  filing is a spot-check of one configuration. The tuple is advisory:
  it names what the re-test is *for*, and the run declares what
  actually ran.
- `contract.specified` with `{"acceptance": {"ref", "executable":
  true, "gate"}}` at the anchor.

The subject id is `eval-<12 hex>` derived from the definition's name,
the tuple under re-test and the count of evals already filed for that
pair, so a second filing for a satisfied window gets a new id and a
duplicate in the same window refuses at the boundary.

**The marker activates at `seed/3`** ([`protocol.md`](protocol.md)). The
lifecycle fold reads it into `SubjectState.Eval` (`{Name, Tuple}`) at
`seed/3` positions only, dropping and counting an advisory tuple that
does not parse rather than folding a partial configuration; at an
earlier tip admission refuses a filing that carries it, naming the
version, so a filing a `seed/2` validator's fold would silently read as
an ordinary contract is never admitted.

From there the lifecycle is the lifecycle. **No transition row moves.**
The supervisor's offer (`offer.published`, never scoped by `tuples`: an
eval is how a configuration proves itself, so no set may gate who takes
it), the worker's claim and reservation, the supervisor's `run.started`
declaring the worker's tuple, the submission, and the verdict under L1
independence with a receipt, all as [`lifecycle.md`](lifecycle.md),
[`offers.md`](offers.md), [`executors.md`](executors.md) and
[`verdicts.md`](verdicts.md) say. Two things differ, and both are
consequences of what an eval is:

- **The set rule is exempt on eval subjects.** `run.started` on an eval
  admits any declared tuple, a disqualified one included; on a non-eval
  subject the set rule stands unchanged. The eval is what a
  configuration must pass to enter the set, so the set cannot be what
  gates the eval.
- **The verdict is the eval's terminal fact.** No merge is owed:
  `verdict.unmerged` is not projected for an eval subject
  ([`obligations.md`](obligations.md)) and `unreconciled` is not
  classified for one ([`reconciliation.md`](reconciliation.md)). The
  consequence of the verdict is the qualification below, not a merge.

## The consequences: `actor.qualified` and `actor.disqualified`

Two actor verbs, the catalog's `actor.qualified` ("cites eval results
and the runtime tuple") and its inverse, both defined at `seed/3`
([`actors.md`](actors.md)):

| verb | subject | payload (strict) |
|---|---|---|
| `actor.qualified` | an enrolled fingerprint | `{"capability", "tuple", "contract", "verdict"}` |
| `actor.disqualified` | an enrolled fingerprint | `{"capability", "tuple", "contract", "verdict", "reason"}` |

`tuple` is the strict five-field runtime tuple; `contract` is the eval
subject; `verdict` is the cited verdict's chain position as a string;
`reason` is required on a disqualification and refused on a
qualification (the cited verdict is the reason). Both accept
`supervise` or `operator`: the capability table's first actor rows the
supervisor holds, because §5 makes suspension the supervisor's
attributable act with no operator ceremony, and operator stays the
standing override.

**In the keyring** a qualification is a grant with evidence: it grants
the capability if absent, adds the tuple to the holder's admissible set
(`GrantTuples`), marks the capability as ever cited, and records
`{capability, tuple, contract, verdict, ts}` with the event's own `ts`.
The sealer disjointness a grant obeys applies. A disqualification
removes the tuple from the admissible set, refusing when the actor holds
no admissible grant citing it ("nothing to disqualify"), keeps the mark,
and records the same with its reason. At a `seed/2` position either verb
fails verification as `bad_actor_event` at its position, exactly as a
`seed/2` validator fails it.

**At the boundary** the qualification rule reads the lifecycle, and
every refusal is its own row with nothing appended:

- the cited contract is an eval;
- `actor.qualified` cites the eval's **authenticated pass**: the latest
  verdict is a pass whose signer, replayed to the verdict's own
  position, held a verdict grant and was no implementing key (the
  verifier boundary re-checked, so a raw-pushed pass qualifies nobody);
  `actor.disqualified` cites an authenticated **fail** on the eval;
- the claim window the verdict's submission cites carries an admitted
  `run.started` declaring exactly the cited tuple, all five fields: a
  qualification is for the configuration that ran, never another;
- for `actor.qualified` the subject is that window's **holder**: the
  supervisor declares, the holder executes, and the holder is who the
  eval qualifies;
- the record's `ts` is not before the cited verdict's;
- no earlier record on the actor cites the same verdict with the same
  kind: one verdict, one consequence;
- the signer holds `supervise` or `operator`, else `out_of_grant`.

**A mint needs a recomputed receipt.** Admission cannot tell an
invented digest from a real one ([`verdicts.md`](verdicts.md)), so the
derivation that owes a mint retrieves the cited receipt from the
artifact store and recomputes it from the submission head with the same
function `seed verdict check` runs, and mints only when the digest
reproduces and every transcript exited zero. A receipt that does not
retrieve, does not reproduce, or carries a red transcript mints nothing,
and the derivation says which by name (`receipt_missing`,
`receipt_mismatch`, `checks_red`; `receipt_unchecked` when it had no
store or repository to recompute against, or the subject carries sealed
checks it does not unseal).

**A disqualification is tuple-wide.** An authenticated fail under T
owes one `actor.disqualified` for EVERY actor whose admissible set holds
T, each citing the verdict: a configuration that failed is not one
configuration per holder.

**The bridge does not reopen.** [`qualification.md`](qualification.md)'s
set rule reads an empty set as the bridge. Refined here: an empty set
that was **ever cited** admits nothing, and the refusal says every cited
configuration is disqualified. An actor whose only cited configuration
failed must pass an eval again, which the exemption above lets it do.

## What the chain owes: `seed eval status` and `seed eval act`

`eval.Due(now, after)` derives, at a DECLARED instant, the acts the
chain owes and the lane each is owed by:

| act | when | lane |
|---|---|---|
| `offer.published` | an eval is `ready` with no live offer (expires a day past `now`, unscoped) | `supervise` |
| `actor.qualified` | an authenticated pass whose window declared a tuple, not yet cited, whose receipt recomputes green | `supervise` |
| `actor.disqualified` | an authenticated fail, for every holder of the declared tuple not yet cited | `supervise` |
| `intent.filed` + `contract.specified` | a spot-check is due (below) | `dispatch` |

**Spot-checks age from the record, against the declared instant.** For
each admissible `(actor, tuple)` whose latest qualification is older
than `after` (`--spot-check-after`, default 168h; `0` disables), the
derivation files a re-test of the eval that qualification cites, naming
the tuple, unless an eval naming that tuple is already open. Age is the
qualification record's own `ts` measured against `--as-of` (default
now), never a clock read inside the derivation, so a run replays; a
qualification dated **after** the declared instant is due at once and
noted as an anomaly, because a lie about time cannot postpone a
re-test. A spot-check taken and failed drives the disqualification; one
taken and passed mints a new qualification whose `ts` is what the next
window measures from. A tuple granted by hand with no qualification has
no eval to re-run.

`seed eval status (posture) --repo <dir> [--artifacts <dir>] [--as-of
<rfc3339>] [--spot-check-after <dur>] [--timeout <dur>]` renders the
derivation: `owed` and `notes`. **`seed eval act`**, the same flags plus
`--key`, performs the subset of `Due` its key's grants admit and reports
the rest as `owed` by their lane: the supervisor offers, mints and
disqualifies; the dispatcher files and specifies; the operator does all.
A key holding no eval lane at all refuses every act `out_of_grant` (exit
14) with nothing appended and never retried. An act the boundary
refuses is reported under `refused` and the invocation exits
`chain_invalid`. Each act is one derivation at one instant: what the
performed acts make due next (a filed spot-check's offer, say) surfaces
on the next. `seed maintain run` is unchanged and appends none of
these; the acts are the supervisor's and the dispatcher's, attributably.

## Exit codes

No new exit. `ungated` (18) for a definition not at a reviewed
revision; `spec_unrunnable` (19) with the `eval_vacuous` refinement;
`checks_red` (20); `not_found` (4) for an unknown definition;
`out_of_grant` (14) for a key holding no eval lane; `chain_invalid`
(8) for a qualification the boundary refuses.

## Deferred, by name

Ranking of tuples (the offer's `tuples` scope stays the policy input;
D9), verifier calibration and authority suspension for verifiers (item
4), independence levels per tier (item 3), re-triage and unedited-
approval rates (item 5), and container or cloud adapters. A receipt
carrying sealed checks is not unsealed by the derivation, so an eval
above the trivial tier mints nothing until that seam exists.

## Conformance mapping

- III.E "Scheduled spot-check evals re-test active tuples; failures
  suspend grants attributably without operator ceremony": `Due`'s
  spot-checks and tuple-wide disqualifications, performed by the
  supervisor's `seed eval act` under its own grant; drilled on a local
  ledger and in small-team mode against a remote.
- III.O row 1 "Eval contracts run through production machinery against
  fixture repos and gate tuple qualification; spot-checks run
  scheduled": the shipped `fix-the-check` definition, `seed eval check`
  proving its verdict, and the end-to-end drill through the production
  verbs with the mint gated on a recomputed receipt. "Scheduled" is
  what a Routine invoking `seed eval act` at an interval provides; the
  derivation is what makes such a run replayable.
- III.E row 6 (out of grant on drift): item 1's rule, now fed by mints
  rather than hand grants, with the closed-bridge refinement.
