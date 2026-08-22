# Plan: workflow engine + mock harness — checked-in DAGs (os-52b9aed0)

The v2 workflow engine (§7.3): goals become **checked-in step DAGs** —
YAML files whose steps declare `depends_on` edges, artifact contracts,
and gates — validated in CI and executed in parallel waves. The format
and validate rules are already decided at implementation grade in
[inspirations/04](../docs/research/inspirations/04-workflow-as-config.md)
(SYNTHESIS section) and §148 of the design doc; this plan implements
that decision, it does not reopen it. The card-level dependency DAG
(blocks/blocked-by + ready-gating + close cascade) is untouched — a
workflow is the *intra-run* DAG a single driver executes; cards remain
the inter-agent coordination layer, and **every task-state mutation a
workflow step makes still goes through `scripts/seed task <verb>`**
(the §7.1 port rule — the engine adds no side channel).

## Steps

1. **Schema** (template, control surface):
   `.seed/workflow-schema/workflow.schema.json` — JSON Schema (2020-12)
   encoding the decided step schema: `schema_version "1"`, `name`
   (must match filename), `description`, `inputs`, `defaults`
   (harness/model), `budgets` (`max_wall_clock_minutes`,
   `max_step_retries`), and `steps[]` with kebab-case unique ids,
   `depends_on` **by id** (never index), XOR of
   `prompt`/`prompt_file`/`run`, `role`/`harness`/`model` overrides,
   `tools: readonly|coding`, `consumes`/`produces` artifact contracts,
   `when`, `trigger_rule`, `on_fail: block|continue`, loop groups
   (`until` + `steps` + `max_iterations`), and `gate`
   (`approval|review|checks`). Unknown fields fail validation.
   Workflows live at `.seed/workflows/<name>.yaml`.
2. **Validator** (engine, `internal/workflow`):
   `seed workflow validate [<file>|--all]` implementing the thirteen
   decided rules — schema conformance, `schema_version` match, unique
   ids, DAG acyclicity + `depends_on` existence, prompt XOR,
   referenced prompt/schema files exist, **artifact closure** (every
   `consumes` produced by a step reachable through `depends_on`),
   role closure against `.seed/roles/*`, harness values against the
   known-harness list (the `scripts/harness/` contract), budget sanity,
   loop requirements, template-token lint (`{{inputs.*}}`,
   `{{output.*}}` must resolve), and optional `--with-harnesses`
   PATH checks. Envelope lists per-rule findings; any error exits 3.
3. **Executor**: `seed workflow run <name> --input k=v [--mock]
   [--resume <run-id>]` — topological **parallel waves** over
   `depends_on`; `when` skips, `trigger_rule` arbitrates fan-in,
   `on_fail: block` stops the DAG while `continue` marks-and-proceeds;
   loop groups re-run their body until `until` or `max_iterations`.
   AI steps go through the existing harness adapter contract
   (`scripts/seed-harness`: prompt on stdin, JSON envelope out,
   exits 0/1/3/124/127) with role/harness/model resolved per step;
   `run:` steps execute directly. Artifacts land under the run
   directory and `{{output.<name>.path}}` tokens resolve exactly as
   validated.
4. **Run state is local, never committed** (the inspirations/04
   erratum): checkpoints and artifacts live under
   `<git-common-dir>/seed-runs/<run-id>/` (the fastcards placement
   precedent — shared across linked worktrees, invisible to commits
   and CI). Checkpoint records completed steps by id, step results,
   and output files; `--resume` refuses a succeeded run and re-runs
   only failed/incomplete steps with prior `{{output.*}}` restored.
5. **Gates**: `approval` pauses the run — the checkpoint records the
   pending gate and message, the envelope says how to resume, and the
   captured response file joins the artifacts; `review` runs the
   reviewer-role step and loops on `REVISE` up to `max_revisions`;
   `checks` polls the GitHub commit-status/checks API for the named
   ref using `GITHUB_TOKEN` when present and refuses with remediation
   when absent. Under `--mock`, all gates auto-pass and say so in the
   run report.
6. **Mock harness** (template): `scripts/harness/mock` honoring the
   harness contract with zero credentials — deterministic canned
   envelopes, `produces` files materialized from the step's declared
   contract (schema-valid stubs) — so `seed workflow run --mock`
   proves a workflow end-to-end in CI. This is the crewplane feature
   that makes the validate story credible.
7. **Shipped workflows + docs**: `.seed/workflows/fix-issue.yaml` (the
   decided example: fetch → plan → approval gate → implement →
   review loop → land; validate-only in CI since it needs `gh`) and
   `.seed/workflows/smoke.yaml` (fully offline; mock-run in CI).
   Handbook gains a workflows section (§3 area): authoring, validate,
   mock runs, run-state location, the port rule for steps, and the
   `.claude/workflows/` note (Claude-native dynamic workflows live
   there, never under `.seed/`). `scripts/validate.sh` gains
   `seed workflow validate --all`.
8. **Engine tests** (offline): fixture workflows violating each
   validate rule one at a time; executor tests driven by the mock
   harness — wave ordering asserted, `when`/`trigger_rule`/`on_fail`
   semantics, loop termination both ways, artifact-token resolution,
   approval-gate pause/resume round-trip, resume-refusal on success,
   failed-step-only re-execution. Engine release + `seed upgrade`-
   performed pin bump carried in this card's task PR.

## File Scope

- Engine repo: `internal/workflow/` (new), `cmd/seed/`
- Template: `.seed/workflow-schema/workflow.schema.json` (new),
  `.seed/workflows/` (new: `fix-issue.yaml`, `smoke.yaml`,
  `prompts/`, `schemas/`), `scripts/harness/mock` (new),
  `scripts/validate.sh`, `docs/handbook.md`, `.seed/engine.lock`
  (the new engine release pin, moved by `seed upgrade`)

## Acceptance Criteria

- `seed workflow validate` refuses each of the thirteen rule
  violations with a named finding, and passes both shipped workflows.
- `seed workflow run smoke --mock` completes offline with zero
  credentials: parallel waves ordered by `depends_on`, artifacts
  materialized and tokens resolved, gates auto-passed and reported,
  run state under `<git-common-dir>/seed-runs/` with nothing to
  commit.
- Interrupting a run and resuming re-executes only incomplete steps;
  resuming a succeeded run is refused.
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
