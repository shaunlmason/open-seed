# v1-retirement.md — retiring open-seed v1

> **Scope and authority.** This document sequences the removal of the v1
> template from this repository. Its authority is
> [`decisions/0006-v1-retirement.md`](../decisions/0006-v1-retirement.md), the
> operator's recorded decision to retire v1, which adds this step after the
> self-hosting cutover that [`docs/next-build-plan.md`](next-build-plan.md) §5
> reserves. The cutover itself is not this document's: §5 defines it and
> [`next/docs/promotion.md`](../next/docs/promotion.md) presents its evidence.
> This document begins where the cutover ends.
>
> **What retirement is.** The cutover moves authority to Seed and freezes v1
> read-only. Retirement removes v1's code, contract and workflows from the tree
> and archives its engine. The frozen history is never removed.

## 1. What retirement requires, and what it does not

Retirement requires the **self-hosting** cutover, because v1 is this
repository's coordination entry point until that cutover merges. It does not
require the **distribution** cutover, which is about what new users clone and
carries its own preconditions (a released binary, an adoptable README). A
repository can retire v1 and never distribute Seed; the two are independent
after the flip.

Retirement is not a rollback path. Once stage 5 merges, the packet's "The path
back" no longer applies: unfreezing the `seed-state` ref would restore the
history but not the shim that reads it. **The rollback window closes when stage
5 merges, and stage 5 is gated on the day-7 audit for exactly that reason.**

## 2. The posture this repository retires at

[`decisions/0005-cooperative-posture.md`](../decisions/0005-cooperative-posture.md)
records the operator's choice of the `cooperative` posture for the self-hosting
cutover, in place of §5's `enforced-self-hosted` precondition. That choice
removes the deployment's infrastructure (no server, no `pre-receive` hook, no
admission service) and is what makes the cutover reachable without new
credentials. Its cost is stated in full in that record and printed at runtime by
`seed doctor`: there is no server-side enforcement, and against a hostile
credential the protocol rules are advisory.

Retirement does not depend on the posture. A later decision may move the
deployment to an enforced posture without revisiting this document, because
`internal/admit` is one rule set and the posture only changes where it runs.

## 3. The order

Five stages. Each names what moves, what it depends on, and how it is verified.
Stages 1 and 2 are ordinary task cards and can land before the cutover. Stages 3
through 5 follow the flip.

### Stage 1 — decouple CI from v1 where Seed is ready  *(before the cutover)*

The pull request gate in `check-validate.yml` runs six v1 steps. Measured
against this tree, they do not all have replacements, and the difference decides
what moves now:

| v1 step | Seed replacement | can move before the cutover |
|---|---|---|
| `seed pr classify <branch> --files` (purity) | `seed plan classify` | **yes**, verified equivalent |
| `seed skills install --frozen` | none needed | **yes**: `seed.yaml` declares no skills, so the step is a no-op and is dropped, not replaced |
| `seed pr classify <branch>` (derive class and task id) | no Seed verb | no: branch naming is a repository convention, and the cutover pull request rewrites those conventions |
| `seed plan lint <file>` | `seed plan lint` | **no**: see below |
| `seed receipt verify` | the verdict pipeline's receipts | no: needs the ledger |
| `scripts/seed-review-identity` | the verdict pipeline's independence rule | no: needs the ledger |

`seed plan classify` is a true equivalent for the purity half: it refuses a
change set that mixes `plans/` with anything else, and one that touches two plan
files, both with `classification_refused` (exit 9), and classifies a
plans-free set as `implementation`.

**`seed plan lint` is not a drop-in replacement, and porting it now would break
the gate.** Seed's plan grammar (`next/spec/plans.md`) requires a boundary set, a
retention check, validation commands carrying `Boundary:` and `Retention:` lines,
and an expected diff shape. The v1 grammar requires none of these. Measured on
this tree: **100 of 166 plans in `plans/` fail `seed plan lint`, and seven of the
twelve most recently added plans fail it.** The repository has not adopted Seed's
plan grammar, so swapping the verb would fail roughly half of new plan pull
requests on their grammar rather than their content.

That makes the plan lint port a **plan-grammar migration**, not a verb swap. It
belongs with the cutover pull request, which rewrites the plan and task
conventions anyway, or as its own card ahead of it. It is called out here because
discovering it at the flip would have made the cutover pull request itself
unmergeable.

*Exit:* `check-validate.yml` calls `seed plan classify` for the purity check and
drops the skills step; the classify-branch, plan lint, receipt and reviewer
identity steps are unchanged on v1; a plan pull request and a task pull request
both pass.

### Stage 2 — the retirement inventory is pinned  *(before the cutover)*

Everything stage 4 deletes is listed in §4 below, and a test holds the list to
the tree so a path that moves is caught here rather than at the deletion. The
list is data; the deletion reads it.

*Exit:* the inventory in §4 resolves against the tree under `make check`.

### Stage 3 — the cutover  *(build plan §5; the operator's escalation)*

Not this document's. It is the packet's, in the packet's order, at the
cooperative posture decision 0005 records. What matters here is what it leaves:
authority on the ledger, `seed-state` frozen at its final anchor, and
`scripts/seed task` already retired from every role file and workflow by the
cutover pull request itself.

After it merges, resolve the four gate steps stage 1 left on v1. Two are ports,
one is a migration, and **one is a deletion rather than a port**, which matters
because reimplementing it in CI would duplicate a rule admission already
enforces:

- **`receipt verify` becomes `seed verdict check`.** A strict superset on the
  recompute side (`next/spec/verdicts.md`, "The receipt"): `merge_base` and
  `head` resolve to full SHAs with a descent check before checkout, the approved
  plan is hashed **at the merge-base** exactly as D3 requires, `diff_sha256`,
  `files` and the command transcripts recompute from the submission head, and a
  mismatch refuses **exit 21 `receipt_mismatch`**. It runs in every
  post-submission state, not only `review`.
- **The reviewer-identity check is deleted, not ported.** v1 reads GitHub reviews
  in CI to prove the reviewer differs from the implementer. Seed enforces the
  same property *at admission*: a `verdict.rendered` whose signer is in the
  contract's implementing-key set (every fingerprint that ever signed a
  `claim.taken` on it, plus the bound submission's signer) refuses **exit 17
  `not_independent`**, per contract and independent of any forge fact. Adding a
  CI step for it would re-derive on the forge what the ledger already refuses.
- **`pr classify <branch>` follows the cutover's conventions.** It derives a
  class and a task id from a branch name, which is a convention rather than a
  verb, and the cutover pull request rewrites those conventions.
- **`plan lint` is the grammar migration stage 1 measured and deferred.** Every
  plan the gate lints from here must carry a boundary set, a retention check,
  `Boundary:` and `Retention:` command lines and an expected diff shape. Existing
  plans are not rewritten: the gate lints the plan of the pull request at hand,
  so the corpus stays as merged evidence.

Drop `state lint`: the ref it lints is v1's.

**One v1 rule has no Seed counterpart by design, and stage 3 must not
reimplement it.** v1's stale-plan check fails a task PR whose merge-base plan
blob differs from the plan blob at the current default-branch head, forcing a
rebase so a revoked plan cannot be replayed forever. Seed reaches the same
property by a different route: a submission above the trivial tier refuses **exit
16 `plan_required`** unless it cites the approved plan anchor (`<path @ commit>`)
*exactly*, because an approval admits one revision, and the ancestry binding is
the receipt's plan hash at the merge-base (`next/spec/plans.md`, "The gate bites
at `submission.made`"). An amended plan is a new approved anchor, so the old
submission's citation stops matching and refuses. Porting v1's comparison on top
of this would add a second, weaker check of a property admission already holds.

*Exit:* no workflow invokes `scripts/seed`; `make fixture-import` has been run
once at the final anchor, so the import fixture records the whole v1 history.

### Stage 4 — the deletion  *(after the day-7 audit reads clean)*

One pull request removes the inventory in §4. It touches no `next/**` code: by
this stage nothing under `next/**` reads a v1 path except the import fixture,
which is data and stays.

*Exit:* `make check` green with the v1 targets gone; the root `README.md` and
`AGENTS.md` describe Seed alone; no tracked file outside `next/fixtures/import/`
and the archived history references `scripts/seed`.

### Stage 5 — the engine repository  *(after stage 4 merges)*

`shaunlmason/open-seed-engine` is dead once nothing execs the pinned binary.
**Archive it; do not delete it.** Its releases are the provenance the frozen
history was produced under, and `.seed/engine.lock`'s hashes are meaningless
without the artifacts they pin. Archiving keeps both readable and makes the repo
read-only.

*Exit:* the engine repository is archived with a README note naming Seed as its
successor and this document as the record.

## 4. The inventory

**Deleted at stage 4.**

| path | what it is |
|---|---|
| `.seed/` | the v1 orchestration contract: config, guardrails, roles, teams, port schema, backends, hooks, workflows |
| `scripts/seed`, `scripts/seed.ps1` | the engine bootstrap shims |
| `scripts/seed-flavor`, `scripts/seed-harness`, `scripts/seed-dispatch-route`, `scripts/seed-review-identity` | v1 helper entry points |
| `scripts/loop.sh`, `scripts/smoke-loop.sh`, `scripts/validate.sh`, `scripts/flavor-test.sh`, `scripts/check-protections.sh`, `scripts/harness/` | the v1 loop runner and its gates |
| `flavors/` | v2 stack variants of the template |
| `plugin/`, `.claude-plugin/`, `.agents/`, `skills/`, `rules/` | the v1 plugin, marketplace and role packaging |
| `seed.yaml`, `seed.lock` | the skills manifest and lockfile |
| `AC.md`, `docs/design-options.md`, `docs/build-plan.md`, `docs/handbook.md`, `docs/architecture.md`, `docs/CONTRIBUTING-AGENTS.md` | v1 vision, design authority, build order, user guide and cross-repo map |
| `.worktreeinclude` | v1 worktree hook input |
| `.github/workflows/seed-dispatch.yml`, `.github/workflows/seed-maintenance.yml`, `.github/workflows/seed-close-no-pr.yml`, `.github/workflows/seed-release.yml`, `.github/workflows/paperclip-live.yml` | the v1 dispatch, maintenance, close, template-release and backend-adapter workflows |

**Edited, not deleted.**

- `Makefile`: the `validate`, `smoke` and `flavor-test` targets go; `check`
  becomes `check-next` alone. `fixture-import` goes with stage 3's final run.
- `README.md`, `AGENTS.md`, `CLAUDE.md`: rewritten around Seed. `next/docs/handbook.md`
  becomes the handbook.
- `CODEOWNERS`: v1 paths drop; the protected surface `seed.json` declares takes over.
- `.github/workflows/check-validate.yml`, `pr-review.yml`: v1 verbs and the
  `Bash(scripts/seed:*)` allowlist replaced by their Seed equivalents.

**Kept, permanently.**

- The `seed-state` ref, frozen at its final anchor, and every `seed-anchor/*` tag.
  This is the history, and the import fixture's provenance.
- `next/fixtures/import/open-seed/`: the export the migration was drilled against.
  It stops being regenerable at stage 3 and is frozen thereafter.
- `decisions/0001`–`0004`: the v1-era records. A decision is not retired because
  its subject is.
- `plans/`, `receipts/`, `memory/`: work products, not v1 machinery.
- `docs/research/`: the research corpus the charter's Appendix C cites as Seed's
  lineage. It is evidence, not template code.

## 5. What retirement does not change

- **The charter and Part III.** Retirement removes no conformance obligation.
  III.R's rows stay measured against the live deployment, before and after.
- **The frozen history's readability.** Anchor tags are ordinary git tags and
  survive the shim that wrote them.
- **The two cutovers' status as escalations.** This document schedules deletion;
  it does not authorize the flip, which stays build plan §5's reserved decision.
