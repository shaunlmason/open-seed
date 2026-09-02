# Plan: next — Phase 10 item 2, eval contracts through the production machinery; grants cite passing tuples; scheduled spot-checks; suspension on failure (os-03e47abb)

The build plan's Phase 10 item 2: *"Eval contracts against fixture repos
through the production machinery; grants cite passing tuples; scheduled
spot-checks; suspension on failure."* Charter §II.16 is the definition:
*"Eval contracts: synthetic work with known verdicts, fixture repos, run
through the production machinery; passing gates grants for the tuple
that ran them; scheduled spot-checks re-test active tuples."* §II.5 is
the consequence: *"Spot-check evals re-test active tuples; drifted or
failing tuples get grants suspended by the supervisor — an attributable
event, no operator ceremony."* Conformance III.E row 7 (spot-checks
suspend attributably) and III.O row 1 (eval contracts gate tuple
qualification; spot-checks run scheduled) are the targets.

This plan is written against item 1's surfaces as they stand in #216
(`plans/os-8e53ffd9.md`, `next/spec/qualification.md`) and depends on
that PR merging first; the card says so.

## What the tree actually shows

Measured, not assumed:

- **Item 1 gave the tuple somewhere to live and the rule that reads
  it.** `actor.granted` cites an optional `tuple` at `seed/2`; the
  keyring keeps the SET per capability (`GrantTuples`); `run.started`
  declares a tuple and the run rule refuses drift `out_of_grant` against
  the CLAIM HOLDER's set, the empty set being the bridge; `Provision`
  holds the adapter to the declaration; offers scope by `tuples`.
- **Nothing mints a grant from a verdict.** `actor.qualified` is
  cataloged ("cites eval results and the runtime tuple", `protocol.md`)
  and the keyring's `default:` arm refuses it by name, pointing at this
  item. The only way a tuple enters a grant today is an operator's hand.
- **Nothing suspends a grant.** `actor.suspended {reason}` suspends the
  KEY's standing, and every `actor.*` verb is accepted from `operator`
  alone (`AcceptedCapabilities`: `IsActorVerb → [operator]`). There is
  no grant-level or tuple-level suspension, and the supervisor role
  (`supervise`) can sign no actor event at all.
- **The verdict pipeline is the production machinery an eval must run
  through**, and it is whole: `contract.specified` with a commit-anchored
  executable acceptance and gate evidence; `seed verdict render
  --repo` executes the acceptance body's validation commands in a
  workspace at the submission head, derives the permissible verdict
  from the transcripts (`checks_red` forbids pass over red), stores the
  receipt content-addressed, and `verdict check` recomputes it. The
  lockout's `authenticFail(c, subject, s)` is the tree's one
  "boundary-validated verdict" derivation; there is no `authenticPass`.
- **The maintenance pass is the only unattended loop, and it is
  wakeless.** `seed maintain run` runs reap, lint, file, rebuild,
  checkpoint against a DECLARED `--as-of` instant, signs every act with
  its own key through `admit.Check`, files findings as `intent.filed`
  under stable ids so the boundary's duplicate refusal makes filing
  idempotent, and holds `maintenance` + `operator`. Its `Deps` inject
  every effect, so its rules are drillable without a ledger.
- **Fixture repositories exist only as a test helper** (`verdictRepo`):
  nothing under `next/` ships an eval, and `next/fixtures/` (named by
  the build plan for drill fixtures) does not exist.
- **`intent.filed` is presence-checked, not strictly decoded**
  (`completeness`: `intent`, `tier`, `budget`, `routing`); the fold
  keeps `Tier` and nothing else from it. A new field is admission
  policy, not chain validity.
- **The set rule refuses exactly the run an eval needs.** A worker
  qualified for A that runs an eval under B is drifting by item 1's
  rule; a worker whose only tuple is disqualified could never re-test
  it. Evaluation is where a configuration proves itself, so the rule
  that qualification feeds cannot gate the act that mints it.

## Design decisions (binding for this task)

- **D1 — an eval contract is an ordinary contract, marked at filing,
  and everything downstream is the production machinery unchanged.**
  `intent.filed` gains an OPTIONAL `eval` object: `{"fixture":
  "<name>", "tuple": {…} | absent}`. `fixture` names a shipped eval
  definition (D3); `tuple` is present on a spot-check and names the
  configuration under re-test, advisory. The fold keeps it as
  `SubjectState.Eval *EvalInfo`. From there the contract is specified,
  offered, claimed, reserved, started, worked, submitted and judged by
  exactly the verbs and rules every other contract crosses: the verdict
  is rendered by a verifier whose key is L1-disjoint from the
  implementer, the receipt is content-addressed and recomputable, and
  the lockout applies. "Through the production machinery" means there
  is no eval runner; there is a marker and a consequence.

  Refused: a separate eval pipeline or a `seed eval run` that executes
  the acceptance itself. An eval that does not cross the verifier
  boundary proves the machinery nothing and can be gamed by whoever
  runs it.

- **D2 — the consequence is `actor.qualified`, minted from an
  authenticated pass verdict, citing what it read.** Subject: the
  actor's fingerprint. Payload, strict: `{"capability": "claim",
  "tuple": {…}, "contract": "<eval subject>", "verdict":
  "<position>"}`. Chain validity (keyring, the `actors.md` posture):
  shape, subject enrolled and not revoked, tuple strict; effect: the
  tuple joins the actor's admissible set for the capability, the
  capability joins `Grants` if absent (a qualification IS a grant with
  evidence), and the entry records the qualification's provenance
  (`{Tuple, Contract, Verdict, Pos}`) for D5's derivation. Admission
  (policy): the signer holds `supervise` or `operator`; the cited
  contract folds with `Eval != nil`; the cited position is that
  subject's **authenticated pass** verdict (a new `authenticPass`
  beside `authenticFail`: verdict grant plus L1 disjointness, replayed
  at the verdict's own position); the `run.started` admitted in that
  verdict's submission window (`RunStartValid`) declared exactly
  `tuple`; that window's holder is the subject; and no earlier
  `actor.qualified` cites the same verdict (the boundary's duplicate
  refusal is what makes minting idempotent). Each is a drilled
  refusal.

  Refused: minting from `verdict.rendered` directly (a verdict is the
  verifier's fact about a submission; a grant is a standing fact about
  an actor, signed by whoever owns qualification). Refused: a
  qualification for the SIGNER of the run (the supervisor declares; the
  holder executed; item 1's rule reads the holder and so does this).

- **D3 — evals ship as reviewed files, and the fixture repository is
  materialized deterministically from them.** `next/evals/<name>/`
  holds `eval.json` (`{"name", "summary", "tier": "trivial",
  "acceptance": {"ref": "<path> @ <sha>", "gate": "eval/<name> @
  <sha>"}, "fixture_commit": "<sha>"}`), `fixture/` (the repository's
  files, the acceptance spec among them), and `solution/` (the files
  the reference solution changes). `internal/eval.Materialize` runs
  `git init`, adds `fixture/`, and commits with a fixed author and
  fixed dates, so the commit is a function of the files; the pinned
  `fixture_commit` is checked against what materialization produced
  and a drifted fixture refuses naming both. The acceptance `ref` cites
  that commit; the `gate` cites the eval definition's own review under
  the established `<pr> @ <commit>` shape, with `eval/<name>` as the
  gate name: the review that armed this executable content is the PR
  that merged the definition, and the commit it binds to is the one the
  ref names, which is what `acceptance.md`'s equality rule checks.

  **Known verdicts are proven, not asserted.** `seed eval check --eval
  <name>` materializes the fixture, runs the acceptance commands there
  through the verifier's own runner, and requires RED; applies
  `solution/` in a second commit, runs them again, and requires GREEN.
  An eval whose unsolved fixture already passes tests nothing and
  refuses `eval_vacuous`, a refinement inside exit 19's family
  (`spec_unrunnable`: the declaration promised something decidable and
  the body decides nothing); a solution that stays red refuses
  `checks_red` (20). One shipped eval, `fix-the-check`: a fixture whose
  `check.sh` fails until a one-line defect is fixed, chosen so the
  verifier's transcript grammar and the worker's edit are both real
  and neither needs a toolchain beyond `sh`.

  Refused: git bundles (binary, unreviewable). Refused: fixtures
  generated at run time from templates (the acceptance ref must name
  one commit, and a commit nobody can recompute is a ref nobody can
  check).

- **D4 — failure suspends the GRANT for that tuple, tuple-wide, by a
  verb of its own: `actor.disqualified`.** Payload, strict:
  `{"capability": "claim", "tuple": {…}, "contract": "<eval>",
  "verdict": "<position>", "reason": "<non-empty>"}`. Chain validity:
  shape, subject enrolled, tuple currently admissible for the capability
  (nothing to disqualify otherwise); effect: the tuple leaves the
  admissible set and the entry remembers it was cited. Admission: signer
  `supervise` or `operator`; the cited verdict is an **authenticated
  fail** on an eval subject whose window's `run.started` declared
  exactly `tuple`. The holder of that window need not be the subject:
  the charter qualifies configurations, not keys, and "failing tuples
  get grants suspended" reads as every grant citing the tuple. D5's
  derivation therefore disqualifies EVERY actor whose admissible set
  holds the failed tuple, one attributable event each, and the drill
  provisions two.

  **The bridge does not reopen.** Item 1's set rule reads the
  admissible set; this card refines it in one place: an actor that has
  EVER been cited for the capability and whose admissible set is now
  empty refuses every run `out_of_grant` ("every cited configuration is
  disqualified"), never bridges. `GrantTuples` returns admissible tuples
  only; a new `EverCited(actor, capability)` distinguishes the two empty
  sets. Re-qualification is a later `actor.qualified` for the same
  tuple citing a verdict after the disqualification; the keyring keeps
  the latest event per (capability, tuple).

  Refused: extending `actor.suspended` with a tuple. Capability would
  then depend on payload (a supervisor may suspend a grant, never a
  key), and `actor.suspended` keeps its one meaning. Refused: a
  disqualification that only touches the failing holder. The
  configuration failed; a second key running the same configuration is
  the same configuration.

- **D5 — "scheduled" means DERIVED FROM THE CHAIN AND PERFORMED BY THE
  PASS, because the pass is the only unattended loop and it is
  wakeless.** `internal/eval.Due(records, table, ring, now, after)`
  returns the acts owed at `now`: a **mint** for every eval subject
  carrying an authenticated pass whose run declared a tuple the holder
  is not yet qualified for by that verdict; a **disqualification** for
  every admissible (actor, tuple) whose tuple an authenticated fail on
  an eval subject names; and a **spot-check** for every admissible
  (actor, tuple) whose latest qualification is older than `after` with
  no open eval subject naming that tuple. A spot-check is a filing, a
  specification and an offer from the shared construction site
  (`eval.Filing`), under a stable id derived from the fixture, the
  tuple and the count of prior evals for that pair, so a second pass in
  the same window refuses the duplicate at the boundary and a satisfied
  window advances the count. `seed maintain run` gains the three acts
  after `file` and before `rebuild`, a `--spot-check-after <duration>`
  (default `168h`; `0` disables) and `--evals <dir>` (default
  `next/evals`), all reported like every other act and refused like
  every other act (a key holding only `maintenance` refuses each
  `out_of_grant` at the door, which the no-private-powers drill
  extends to cover). `seed eval act --key <supervisor>` performs the
  SAME `Due` by hand: one derivation, two signers.

  Refused: a timer, a cron surface, or a wake channel. Refused: due-ness
  stored anywhere; it is recomputed from the qualification's own
  position and the declared `now`, so a pass replays.

- **D6 — eval offers are never tuple-scoped, and the run rule exempts
  eval subjects from the set rule.** An eval for B exists to qualify
  workers NOT yet qualified for B, and a re-eval of a disqualified T is
  taken by a worker that cannot currently see a T-scoped offer; so
  `eval.Filing` publishes the offer scoped by capability and tier only,
  and `run.started` on a subject with `Eval != nil` admits any declared
  tuple, disqualified ones included. The mint reads what the run
  DECLARED, never what the intent's advisory `tuple` said; a spot-check
  taken under a different configuration qualifies that configuration
  and leaves the checked one due.

- **D7 — the signer is the pass, and the supervisor by hand; both are
  the charter's "supervisor".** `actor.qualified` and
  `actor.disqualified` are accepted from `[supervise, operator]`: the
  first `actor.*` rows that are not operator-only. The maintenance lane
  holds `operator` and is audited as an ordinary actor; the supervisor
  role holds `supervise`. "No operator ceremony" is satisfied by
  either: each event is signed by its own key and cites the verdict it
  acted on.

- **D8 — `seed/3`, for the same reason item 1 needed `seed/2`.** A
  `seed/2` validator refuses `actor.qualified` and `actor.disqualified`
  by name at chain validity; this validator accepts them. `version.Seed3`
  joins `Supported()` and `Activated`; `tuple.Applies` becomes true for
  `seed/2` and later (a named list); a new `eval.Applies(active)` gates
  the two verbs and the `eval` field at `seed/3` exactly. The
  mixed-version drill repeats: a `seed/2`-only build refuses an upgraded
  chain at the first `seed/3` record by version, never by misjudging a
  qualification; existing chains verify byte for byte.

- **D9 — scope guard.** No ranking of tuples (offers' `tuples` scope
  stays the policy input); no calibration, gold set or authority
  suspension for VERIFIERS (item 4); no levels (item 3); no re-triage or
  approval rates (item 5); no container or cloud adapter. One exit-code
  refinement (`eval_vacuous` under 19), no new exit. No transition row
  moves: an eval contract's lifecycle is the lifecycle.

## Steps

0. `next/internal/version/` — `Seed3`, `Supported()`, `Activated`;
   `next/internal/tuple/` — `Applies` for `seed/2` and later;
   `next/spec/protocol.md` — the `seed/3` register entry and the catalog
   rows for `actor.qualified` and `actor.disqualified` (D8).
1. `next/internal/eval/` (new) — `Definition` and `Load(dir)`;
   `Materialize(def, into)` with fixed author and dates, checked
   against `fixture_commit`; `Check(def)` (fixture red, solution green,
   through the verifier's runner); `Filing(def, tuple)` (the three
   payloads and the stable id); `Due(...)` (D5); `Applies`.
2. `next/evals/fix-the-check/` (new) — `eval.json`, `fixture/`,
   `solution/`; pinned commit; `seed eval check` green on it.
3. `next/internal/transition/` — `intent.filed`'s optional `eval`
   folded to `SubjectState.Eval`.
4. `next/internal/keyring/` — `actor.qualified` and `actor.disqualified`
   at `seed/3` positions; `Entry` gains the qualification record;
   `GrantTuples` returns the admissible set; `EverCited`; `Grants`
   unchanged in shape.
5. `next/internal/admit/` — `AcceptedCapabilities` rows for the two
   verbs; `authenticPass`; the two admission rules (D2, D4); the set
   rule's eval exemption and disqualified-empty refusal (D4, D6);
   affordance probes for the two verbs.
6. `next/internal/maintain/` — the three acts in `Deps` and `Run` (D5),
   reported and refused like the rest.
7. `next/cmd/seed/eval.go` (new) — `seed eval list | check | file |
   status | act`; `next/cmd/seed/maintain.go` — the flags and wiring;
   `main.go` — the dispatch case; `next/lanes/maintenance.json` — the
   summary names the acts.
8. Drills: `internal/eval` (materialization determinism, drift, check,
   filing id, `Due` matrix against planted chains); keyring (the verbs'
   shapes at `seed/2` and `seed/3`, admissible set, `EverCited`, latest
   wins); admit (every refusal in D2 and D4, the exemption, the
   disqualified-empty refusal, holder versus signer); maintain (acts
   reported and refused); `cmd/seed` (`eval check` on the shipped eval
   and on planted vacuous and drifted ones; `eval act` by the
   supervisor and by an unentitled key; the mixed-version replay); the
   modes fixture's end-to-end run (AC2, AC4, AC5).
9. Specs: new `next/spec/evals.md`; `qualification.md` (what item 2
   adds, replacing that section); `actors.md` (the two verbs, the first
   non-operator rows); `maintenance.md` (the acts; the lint set stays
   closed, these are not lints); `envelope.md` (the `eval_vacuous`
   refinement row); `lifecycle.md` (the `eval` field).
10. `next/docs/progress.md`, `next/docs/decisions.md`,
    `memory/LEARNINGS.md`; receipt; evidence; review.

## File Scope

- `next/internal/version/**`, `next/internal/tuple/**`,
  `next/internal/eval/**` (new), `next/internal/keyring/**`,
  `next/internal/transition/**`, `next/internal/admit/**`,
  `next/internal/maintain/**`, `next/cmd/seed/eval.go` (new),
  `next/cmd/seed/eval_cli_test.go` (new), `next/cmd/seed/maintain.go`,
  `next/cmd/seed/maintain_cli_test.go`, `next/cmd/seed/main.go`,
  `next/cmd/seed/modes_e2e_test.go`, `next/cmd/seed/run_cli_test.go`
  (the set-rule refinement's CLI rows), `next/evals/**` (new),
  `next/lanes/maintenance.json`
- `next/spec/evals.md` (new), `next/spec/qualification.md`,
  `next/spec/actors.md`, `next/spec/maintenance.md`,
  `next/spec/protocol.md`, `next/spec/envelope.md` (one refinement
  row), `next/spec/lifecycle.md`
- `next/docs/progress.md`, `next/docs/decisions.md`, `memory/*`
- `receipts/os-03e47abb.json`

Nothing outside `next/**` except the work-product files above. NOT
`next/spec/transitions.json` (no row moves), NOT `next/internal/envelope/**`
(no exit allocated; the refinement is a spec row and an emitted code),
NOT `next/internal/verdict/**` beyond calling its runner, NOT
`next/executor/**`.

## Acceptance Criteria

1. `seed eval check --eval fix-the-check` reports the unsolved fixture
   RED and the solution GREEN through the verifier's own runner, and the
   materialized commit equals the pinned `fixture_commit`; an eval whose
   fixture is edited refuses naming both commits; a planted eval whose
   fixture already passes refuses `eval_vacuous` (exit 19); one whose
   solution stays red refuses `checks_red` (exit 20).
2. In small-team mode, an eval contract filed by `seed eval file` runs
   through the production machinery end to end (offer, claim, reserve,
   `run start` declaring the worker's tuple, work applying the solution,
   submit, an L1-disjoint verifier's `verdict render` pass with a
   recomputable receipt), and `seed maintain run` then mints
   `actor.qualified` for the HOLDER and the DECLARED tuple citing that
   verdict; afterwards a run on an ordinary contract under that tuple
   admits and under a tuple differing in one field refuses `out_of_grant`
   (item 1's rule, now fed by a mint rather than an operator's hand).
3. `actor.qualified` refuses, each its own row with nothing appended: a
   non-eval subject; a verdict that is not boundary-authenticated (raw
   pushed by a key without `verdict`, or by the implementer); a fail
   verdict; a tuple differing from the window's declaration in any one
   field (five rows); a subject that is not the window's holder (the
   supervisor who signed the start, drilled explicitly); a signer holding
   neither `supervise` nor `operator`; a second mint citing the same
   verdict. At a `seed/2` position the verb fails verification as
   `bad_actor_event`.
4. On an authenticated fail on an eval whose run declared T,
   `seed maintain run` appends one `actor.disqualified` for EVERY actor
   whose admissible set holds T (two provisioned), each citing the
   verdict; afterwards a run under T on an ordinary contract refuses
   `out_of_grant` for each, an actor whose only cited tuple was T admits
   NOTHING (the bridge does not reopen), an actor also qualified for U
   still admits U, and a later passing eval under T re-qualifies.
   `actor.disqualified` refuses a tuple not currently admissible, a pass
   verdict, a non-eval subject, and an unentitled signer.
5. With `--spot-check-after 24h` and `--as-of` 25 hours past a
   qualification, the pass files, specifies and offers exactly one
   spot-check for that (fixture, tuple), unscoped by tuple, and reports
   it; a second pass at the same instant appends nothing (the duplicate
   refuses at the boundary and is reported as a refusal, not an error);
   at 23 hours nothing is due; with an open eval naming the tuple nothing
   is due; with `0` nothing is ever due. A spot-check taken and failed
   drives AC4; taken and passed advances the qualification so the next
   window is measured from it.
6. `run.started` on an eval subject admits any declared tuple, a
   disqualified one included; on a non-eval subject the set rule stands
   (a mutation applying the exemption everywhere goes red).
7. `seed eval act --key <supervisor>` performs exactly the acts `Due`
   returns and `seed maintain run` would perform, signed by the
   supervisor; a key holding neither capability refuses each
   `out_of_grant` with nothing appended; the maintenance key holding
   only `maintenance` refuses the same way, reported, never retried.
8. A `seed/2`-only build refuses an upgraded chain at the first `seed/3`
   record by version, never by judging a qualification; this build
   verifies it; every pre-existing fixture chain verifies byte for byte.
9. **Mutation evidence.** Each must fail a drill: `authenticPass`
   accepting an unverified verdict; the mint reading the signer rather
   than the holder; the mint comparing the tuple on four fields; the
   duplicate-verdict refusal removed; `GrantTuples` returning disqualified
   tuples; `EverCited` returning false after a disqualification (the
   bridge reopening); the disqualification touching only the failing
   holder; the exemption applied to non-eval subjects; `Due` filing a
   spot-check with an open eval standing; `Due` measuring from the
   qualification's `ts` rather than its position (planted out-of-order
   timestamps); `Materialize` with a wall-clock date; `eval check`
   skipping the red half; the `eval` field read at `seed/2`.
10. `make check` green with coverage measured **cold**, at least three
    readings above the gate, and the suites pass **unprivileged** under
    `setpriv --reuid=65534`.

## Validation Commands

```sh
cd next && gofmt -l . && go vet ./... && go build ./...
cd next && go test ./internal/version/ ./internal/tuple/ ./internal/eval/ ./internal/keyring/ ./internal/transition/ ./internal/admit/ ./internal/maintain/ ./cmd/seed/ -count=1
cd next && go test ./... -count=1
make check
```
