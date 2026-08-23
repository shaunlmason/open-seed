# Plan: workflow engine + mock harness: checked-in DAGs (os-52b9aed0)

The v2 workflow engine (§7.3): goals become **checked-in step DAGs**:
YAML files whose steps declare `depends_on` edges, artifact contracts,
and gates: validated in CI and executed in parallel waves. The format
and validate rules are already decided at implementation grade in
[inspirations/04](../docs/research/inspirations/04-workflow-as-config.md)
(SYNTHESIS section) and §7.3 of the design doc; this plan implements
that decision, it does not reopen it. The card-level dependency DAG
(blocks/blocked-by + ready-gating + close cascade) is untouched: a
workflow is the *intra-run* DAG a single driver executes; cards remain
the inter-agent coordination layer, and **every task-state mutation a
workflow step makes still goes through `scripts/seed task <verb>`**
(the §7.1 port rule: the engine adds no side channel).

## Steps

1. **Schema** (template, control surface):
   `.seed/workflow-schema/workflow.schema.json`: JSON Schema (2020-12)
   encoding the decided step schema: `schema_version: "1"`, `name`
   (must match filename), `description`, `inputs`, `defaults`
   (harness/model), `budgets` (`max_wall_clock_minutes`,
   `max_step_retries`), and `steps[]` with kebab-case unique ids,
   `depends_on` **by id** (never index), XOR of
   `prompt`/`prompt_file`/`run`, `role`/`harness`/`model` overrides,
   `tools: readonly|coding`, `output_format` (inline JSON Schema
   constraining an AI step's structured output: validated here,
   enforced by the executor on the step's declared `produces` file),
   `consumes`/`produces` artifact contracts, `when`, `trigger_rule`,
   `on_fail: block|continue`, loop groups (`until` + `steps` +
   `max_iterations`), and `gate` (`approval|review|checks`). Unknown
   fields fail validation. Workflows live at
   `.seed/workflows/<name>.yaml`. `tools` maps onto the existing
   harness contract's `SEED_PERMISSION`: `readonly` → `read-only`,
   `coding` → `safe-edit`; `yolo` is **not reachable from a workflow
   file**, no `tools` value maps to it (the mapping is declared in the
   schema description and asserted by an executor test).
2. **Validator** (engine, `internal/workflow`):
   `seed workflow validate [<file>|--all]` implementing the thirteen
   decided rules: schema conformance, `schema_version` match, unique
   ids, DAG acyclicity + `depends_on` existence, prompt XOR,
   referenced prompt/schema files exist, **artifact closure** (every
   `consumes` produced by a step reachable through `depends_on`),
   role closure against `.seed/roles/*`, harness AND model values
   against the pinned registry (a `[workflows]` table in
   `.seed/config.toml` listing known harnesses: seeded from
   `scripts/harness/`, and permitted model identifiers; a misspelled
   or unlisted model fails preflight, not the first live invocation),
   budget sanity, loop requirements, template-token lint
   (`{{inputs.*}}`, `{{output.*}}` must resolve), and optional
   `--with-harnesses` PATH checks. Envelope lists per-rule findings;
   any error exits 3.
3. **Executor**: `seed workflow run <name> --input k=v [--mock]
   [--resume <run-id>]`: topological **parallel waves** over
   `depends_on`; `when` skips, `trigger_rule` arbitrates fan-in,
   `on_fail: block` stops the DAG while `continue` marks-and-proceeds;
   loop groups re-run their body until `until` or `max_iterations`.
   AI steps go through the existing harness adapter contract
   (`scripts/seed-harness`: prompt on stdin, JSON envelope out,
   exits 0/1/3/124/127) with role/harness/model resolved per step and
   `SEED_PERMISSION` set from the step's `tools` mapping (default
   `coding` → `safe-edit`); `run:` steps execute directly: except
   under `--mock`, below. Artifacts land under the run directory and
   `{{output.<name>.path}}` tokens resolve exactly as validated;
   a step declaring `output_format` has its produced file checked
   against that schema before dependents run.
4. **Run state is local, never committed** (the inspirations/04
   erratum): checkpoints and artifacts live under
   `<git-common-dir>/seed-runs/<run-id>/` (the fastcards placement
   precedent: shared across linked worktrees, invisible to commits
   and CI). Checkpoint records completed steps by id, step results,
   output files, **and the sha256 of the fully resolved workflow
   definition + inputs**; `--resume` refuses a succeeded run, refuses
   a checkpoint whose definition/input hash no longer matches (no
   mixed-graph results: a changed workflow starts a fresh run), and
   re-runs only failed/incomplete steps with prior `{{output.*}}`
   restored.
5. **Gates**: `approval` pauses the run: the checkpoint records the
   pending gate and message, the envelope says how to resume, and the
   captured response file joins the artifacts. `review` is a
   review-and-fix loop, not a bare re-poll: the gate names a
   `remediation` step (a coding-role step, validated to exist) that
   runs on every `REVISE` verdict before the reviewer-role step is
   re-run, up to `max_revisions`: re-reviewing an unchanged
   implementation is exactly what the loop must never do. `checks`
   requires BOTH the decided conditions: commit statuses/check runs
   green (REST) **and** `unresolved_threads: 0` via the review-thread
   GraphQL query, using `GITHUB_TOKEN` when present and refusing with
   remediation when absent; a test proves an unresolved thread keeps
   the gate closed.
6. **Mock mode + mock harness** (template): under `--mock` the run has
   **zero side effects and zero credentials**: AI steps go to
   `scripts/harness/mock` (honoring the harness contract:
   deterministic canned envelopes, `produces` files materialized as
   schema-valid stubs), **`run:` steps are stubbed too** (command
   recorded in the run report, never executed, `produces` stubbed the
   same way), and gates auto-pass with that fact in the report, so a
   mock run of ANY workflow, `fix-issue` included, can never reach a
   real `gh` call or a land step. Proving a workflow end-to-end in CI
   this way is the crewplane feature that makes the validate story
   credible; executing real commands stays exclusive to non-mock
   runs.
7. **Shipped workflows + docs**: `.seed/workflows/fix-issue.yaml` (the
   decided example: fetch → plan → approval gate → implement →
   review loop → land; validate-only in CI since it needs `gh`) and
   `.seed/workflows/smoke.yaml` (fully offline; mock-run in CI).
   Handbook gains a workflows section (§3 area): authoring, validate,
   mock runs, run-state location, the port rule for steps, and the
   `.claude/workflows/` note (Claude-native dynamic workflows live
   there, never under `.seed/`). `scripts/validate.sh` gains a
   `sh scripts/seed workflow validate --all` step (the script's
   existing engine-invocation style).
8. **Engine tests** (offline): fixture workflows violating each
   validate rule one at a time; executor tests driven by the mock
   harness: wave ordering asserted, `when`/`trigger_rule`/`on_fail`
   semantics, loop termination both ways, artifact-token resolution,
   `output_format` enforcement, the `tools`→`SEED_PERMISSION` mapping
   (a `readonly` step's harness sees `read-only`; no mapping yields
   `yolo`), mock-mode purity (a `run:` step's command is recorded,
   never executed), the checks gate held closed by an unresolved
   review thread (fake API), the review gate running its remediation
   step between verdicts, approval-gate pause/resume round-trip,
   resume-refusal on success and on a changed definition hash,
   failed-step-only re-execution, and an invalid-model fixture refused
   by the registry rule. Engine release + `seed upgrade`-
   performed pin bump carried in this card's task PR.

## File Scope

- Engine repo: `internal/workflow/` (new), `cmd/seed/`
- Template: `.seed/workflow-schema/workflow.schema.json` (new),
  `.seed/workflows/` (new: `fix-issue.yaml`, `smoke.yaml`,
  `prompts/`, `schemas/`), `scripts/harness/mock` (new),
  `scripts/validate.sh`, `.seed/config.toml` (the `[workflows]`
  harness/model registry), `docs/handbook.md`, `.seed/engine.lock`
  (the new engine release pin, moved by `seed upgrade`)

## Acceptance Criteria

- `seed workflow validate` refuses each of the thirteen rule
  violations with a named finding, and passes both shipped workflows.
- `seed workflow run smoke --mock` completes offline with zero
  credentials AND zero side effects: parallel waves ordered by
  `depends_on`, artifacts materialized and tokens resolved, `run:`
  commands recorded but never executed, gates auto-passed and
  reported, run state under `<git-common-dir>/seed-runs/` with
  nothing to commit.
- A `tools: readonly` step's harness receives `SEED_PERMISSION=read-only`;
  no workflow value maps to `yolo`.
- Interrupting a run and resuming re-executes only incomplete steps;
  resuming a succeeded run, or a checkpoint whose definition/input
  hash changed: is refused.
- No workflow step mutates task state except through
  `scripts/seed task <verb>`; the executor adds no direct backend
  access.
- The handbook documents authoring, validation, mock runs, and the
  run-state location; `validate.sh` fails on an invalid checked-in
  workflow.

## Validation Commands

- `test -f .seed/workflow-schema/workflow.schema.json`
- `scripts/seed workflow validate --all`
- `scripts/seed workflow run smoke --mock`
- `grep -q "seed workflow validate" docs/handbook.md`
- `sh scripts/validate.sh`
