# Deep Dive: Workflow-as-Committed-Config Schemas

> Implementation-grade deep dive for the open-seed design study, researched 2026-08-22.
> Covers tutti, agent-runbook, crewplane, Fusion, and Archon. All read at source level
> (config structs, validators, executors), not just READMEs.

## 1. nutthouse/tutti — `tutti.toml` (Rust)

The complete schema comes from `src/config/mod.rs` (serde structs) plus the repo's own dogfood `tutti.toml` (399 lines) and `docs/examples/tutti-codex-sdlc.toml`.

**Top level:** `[workspace]` (name, description, `[workspace.env]` with git_name/git_email + flattened extra env, `[workspace.auth].default_profile`), `[defaults]` (`worktree` bool default true, `runtime`), `[roles]`, `[launch]` (`mode = safe|auto|unattended`, `policy = constrained|bypass`), `[[agent]]`, `[[tool_pack]]`, `[[workflow]]`, `[[hook]]` (events `agent_stop|workflow_complete`), `[handoff]`, `[observe]`, `[budget]`, `[[webhook]]`.

**Roles mapping** — a plain table; agents reference it via `role`:

```toml
[roles]
planner = "claude-code"
implementer = "claude-code"
reviewer = "codex"

[[agent]]
name = "backend"
role = "backend"     # runtime resolved via [roles]
```

**`[[agent]]` fields** (verbatim from `AgentConfig`): `name` (required), `runtime` (validated against `["claude-code", "codex", "aider", "openclaw"]`), `scope` (a single glob string, e.g. `scope = "src/**"` — not a list), `prompt`, `depends_on` (agent names), `worktree`, `fresh_worktree`, `branch`, `persistent` (bool), `memory` (path to persistent memory file), `env` (map), `role`.

**Workflow step types** — a serde enum tagged by `type`, six variants:

```rust
#[serde(tag = "type", rename_all = "snake_case")]
pub enum WorkflowStepConfig {
  Prompt { id, depends_on: Vec<usize>, agent, text, inject_files: Vec<String>,
           output_json, wait_for_idle, wait_timeout_secs, startup_grace_secs,
           artifact_glob, artifact_name, direct, provider, model, policy },
  Command { id, depends_on, run, cwd: workspace|agent_worktree, subdir, agent,
            timeout_secs, fail_mode: open|closed, output_json },
  EnsureRunning { depends_on, agent, fail_mode },
  Workflow { depends_on, workflow, agent, strict, fail_mode },   // nested workflow
  Land { depends_on, agent, pr: Option<bool>, force, fail_mode },
  Review { depends_on, agent, reviewer, fail_mode },
}
```

Note `prompt` steps carry per-step `provider` and `model` overrides — exactly open-seed's per-step harness/model premise.

**depends_on parallelism**: on steps it is `Vec<usize>` — **1-based step numbers, not ids** (README: "independent `ensure_running`/`review`/`land` steps run in parallel waves"). The executor computes ready waves and runs each wave on threads, joining before the next wave; a `fail_mode = "closed"` failure hard-stops the DAG.

**Artifacts**: a prompt step with `artifact_glob` + `artifact_name` switches from idle-detection to **artifact-polling** — tutti polls the glob until a new file appears. Verbatim from the dogfood config:

```toml
[[workflow.step]]
id = "design"
type = "prompt"
agent = "planner"
text = "/office-hours"
artifact_glob = "~/.gstack/projects/{slug}/*-design-*.md"
artifact_name = "design_doc"
# No wait_for_idle — artifact-polling mode

[[workflow.step]]
id = "eng_review"
inject_files = ["{{output.design_doc.path}}"]
```

Injection tokens are `{{output.<step_id_or_artifact_name>.path}}` and `{{output.<name>.json}}`; rendering fails if any `{{output.` token remains unresolved. `output_json = "path.json"` on a step registers that file as the step's typed output. Glob patterns interpolate `~`, `{slug}`, `{workspace}`, `{agent}`.

**Permissions** live in global config (`~/.config/tutti/config.toml`), not tutti.toml: `[permissions] allow = [...]` where entries are shell command prefixes (`"git status"`, `"cargo test"`) and/or Claude tool names (`Read`, `Edit`). Policy evaluation prefix-matches normalized commands; on deny it returns a `suggested_rule`. With launch `mode="auto"`, `policy="constrained"`, non-interactive runs *require* configured allow rules or `tt up` fails. Decisions append to `.tutti/state/policy-decisions.jsonl`.

**Land gate** (`src/cli/land.rs`): enabled by `TT_ENFORCE_MERGE_GATE`; requires `gh`; queries `gh pr list` for an open PR on the branch, then `gh pr checks <n> --required` (required checks must be green) and the GraphQL review-thread payload (zero unresolved review threads). Failure messages are explicit: `"merge gate blocked: required checks are not green for PR #N"`, `"...has N unresolved review thread(s)"`.

**Checkpoint/resume**: `.tutti/state/workflow-checkpoints/<run_id>.json` serializes `{run_id, workflow_name, strict, origin, agent_scope, started_at, finished_at, success, failed_steps, step_results, output_files}`; step outputs land in `.tutti/state/workflow-outputs/<run_id>/<step_id>.json`. `tt run --resume <run_id>` refuses if `success == true`, drops failed step results, rebuilds completed steps from surviving indices, reloads `output_files` so `{{output.*}}` still resolves, and re-executes only non-completed steps. Validation checks: duplicate agent names, agent depends_on existence + self-dep + cycles (topological sort), role table closure, known runtimes, and step depends_on bounds.

## 2. KnoxOps/agent-runbook — `runbook.yaml` → SKILL.md compiler (Python)

Crucial framing: **this is a compiler, not a runtime**. `python3 -m agent_runbook generate runbook.yaml --output dir` compiles to a `SKILL.md` that the agent itself executes; checkpoints/loops are instructions in the compiled prose plus generated helper scripts.

**Schema** (`agent_runbook/schema.py`, Pydantic): top level `Runbook { name, description, input_params: [{name, type: string|number|boolean, required, description}], steps: [Step], error_handling: [{scenario, handling}] }`. The Step model:

```python
class Step(BaseModel):
    id: str
    type: StepType = INLINE      # enum: inline | agent | script | loop
    description: str = ""
    prompt: Optional[str]        # XOR prompt_file (inline/agent)
    prompt_file: Optional[str]
    command: Optional[str]       # required for script
    input: list[StepInputRef]    # {schema, from_step, file}
    output: list[StepOutputDef]  # {schema, file}
    depends_on: list[str]        # REQUIRED, by step id
    parallel: Optional[ParallelConfig]   # {enabled, max_instances, item_key}
    checkpoint: Optional[str]    # path to checkpoint script
    condition: Optional[str]     # branch
    quality_check: Optional[QualityCheckConfig]  # {blocking, review_prompt, rules[], output_file}
    goal: Optional[str]          # required for loop
    max_iterations: int = 10
    body: Optional[list["Step"]] # loop sub-steps
```

The README markets "eight step types," but **the enum has only four**; `parallel`, `branch` (`condition`), `checkpoint`, and `quality_check` are *fields/decorators on steps*, each with its own strategy module. Worth remembering when copying the marketing table.

**Contract declaration**: every step's `output:` pairs a file with a JSON Schema path; schemas are ordinary draft-07 files (e.g. a `oneOf` over the real payload vs `{"done": true}` — their sentinel convention for "loop finished").

**Contract-closure validation** (`validator.py`, 13 checks): duplicate step IDs; `prompt_file` existence; **schema file existence**; DAG cycle detection; `from_step` must exist AND must appear in that step's `depends_on` (this is the closure rule — you cannot consume an artifact from a step you don't depend on); condition validity; parallel config sanity; type/field mismatches; output-file conflict warnings; orphan-step warnings; quality_check config; loop requirements (`goal` + non-empty `body`).

**Compiled SKILL.md** (verbatim structure): YAML frontmatter (`name`, `description`, `user-invocable: true`); an Input Parameters table; then an "Execution Flow" that begins with a **`task_context.json` state file the agent must maintain**:

```json
{ "task_id": "<task_id from input>", "current_step": 0, "current_step_id": null,
  "status": "running", "steps": { "inspect": "pending", "fix_loop": "pending" },
  "updated_at": "<ISO timestamp>" }
```

Each step compiles to a `### Step N: <id>` section with **Type**, **Description**, an execution block, Output Files, and a Progress Tracking block. Loops compile a "Goal Evaluation" epilogue: goal met → proceed; not met + iterations remain → reset body steps; max reached → complete with `max_iterations_reached`. Human approval is not a first-class type — it's an `inline` step whose prompt says "WAIT for the human… copy pending_action.json to approved_action.json / write skip_action.json", with downstream steps branching on which file exists. **All state passing is files in the working directory.**

## 3. crewplaneai/crewplane — Markdown workflows with YAML frontmatter (Python)

Workflow files are `*.task.md` under `.crewplane/workflows/`. Verbatim minimum:

```yaml
---
schema_version: "1.0"
name: "Example"
description: "Optional description"
nodes:
  - id: inspect
    mode: parallel
    providers: ["claude"]
---

## inspect
Inspect the project.
```

**Frontmatter**: `schema_version` (must match the tool's), `name`, `description`, `inputs` (workflow input name → local input-node id), `imports` (`{path, as, with, inputs}` — alias-namespaced, cycle-checked, project-root-bounded, unused `with` params fail), `worktrees` (experimental), `nodes`.

**Node fields**: `id` (`[a-z0-9._-]+`, reserved names rejected), `mode: parallel|sequential|input`, `providers` (shorthand strings = executors, or objects `{provider, model, reasoning, role: executor|reviewer}`), `needs`, `continue_on_failure`, `findings` (bool — write a findings artifact), `source` (input mode only), `depth` (sequential rounds / remediation depth), `audit_rounds`, `review_starts_with`, `failure_threshold` (parallel), `token_budget: {warn_threshold_chars, fail_threshold_chars}`, `worktree`.

**Body mapping**: every non-input node requires exactly one `## <node-id>` section; unmarked markdown is shared prompt, and role-scoped segments use standalone HTML comments `<!-- crewplane:executor -->…<!-- /crewplane:executor -->` / `crewplane:reviewer`. Review loops (sequential multi-provider: contiguous executors then contiguous reviewers) require all reviewers to approve; loop order is `for audit_round: for local_round in 1..depth+1: candidate → review → stop on approval`. Template tokens: `{{file:path}}`, `{{env:KEY}}`, `{{var:KEY}}`, `{{node.output}}`, `{{node.findings}}`, plus `_path`/`_size`/`_sha256` metadata variants. Artifact references are **only valid for upstream `needs` dependencies**, and `findings` refs require the producer to declare `findings: true` — validation enforces artifact-closure like agent-runbook does. The bundled `refactoring-example.task.md` shows the full audit→plan→execute→parallel-review→fix→handoff pipeline mixing gemini/claude/codex per node.

**config.yml wiring**: `.crewplane/config.yml` defines `agents.<name>` — `cli_cmd` (argv), `provider_kind: claude|codex|copilot|gemini|kilo|generic`, `default_model`, `model_arg`, `prompt_transport: stdin|argv`, `extra_args`, and an unusually rich retry/quota block (`retry_on_exit_codes`, `retry_on_stderr_contains`, `quota_reached_on_contains`, `quota_retry_max_wait_seconds`, `invocation_idle_timeout_seconds` default 1800, per-million-token `pricing`). Settings include `max_audit_rounds` (default 5), `max_concurrent_nodes`, global `token_budget`.

**Mock provider**: `crewplane init` generates a config with one `mock` agent and `invoker.implementation: "mock"` with options `{output_mode: "lorem", seed: 42, delay_seconds: 0.25}` — deterministic scaffolding output so the first `validate` and `run` are provider-free.

**Validate**: `crewplane validate [TASKS_FILE] -c .crewplane/config.yml` — no providers invoked, no artifacts written; checks workflow + config validity and (for the cli invoker) provider CLI availability on PATH.

**Artifact layout**: run key = `<workflow-id>-<run-id>`; `.crewplane/execution-stages/<run-key>/` (logs/events.ndjson, preflight plans/manifests, per-node logs) and `.crewplane/execution-results/<run-key>/` (`<node-id>-result.md`, `<node-id>-findings.md`, generated-files).

## 4. Runfusion/Fusion (TypeScript)

**Correction to survey framing**: Fusion is *not* really workflow-as-committed-config. Workflows are IR graphs stored in PostgreSQL and edited in a dashboard Workflow Editor; the *committed* artifacts are per-task blobs under `.fusion/tasks/{TASK_ID}/` (`PROMPT.md`, `agent.log`, `attachments/*`) plus `.fusion/project.json` and `.fusion/memory/MEMORY.md`.

**PROMPT.md structure**: `# Task: <ID>` preamble, then `## Mission`, `## Dependencies`, `## File Scope`, `## Steps` containing `### Step N: <title>` headings (0-based; `### Step 0: Preflight`, then Implement / Testing / Docs), `## Completion Criteria`, `## Git Commit Convention` ("FN-prefixed conventional commits"). Additional generated sections: `## Review Advisory Notes` (non-blocking findings) and, for multi-repo workspaces, a mandatory `## Repository Scope`. File Scope is enforced (`strictScopeEnforcement`; scope-overlap blocks parallel tasks; scope additions require approval; validation requires referenced files to be in File Scope, "(new)"-marked, or created by a prior step). Planning output goes through *deterministic validation* before Plan Review; failures retry with backoff and finally park the task.

**Automation levels** (`plannerOversightLevel`, per-task nullable override; resolution: task override → workflow value → `autonomous` default): `off` — oversight disabled; `observe` — watches/logs only; `steer` — injects guidance or suggests revisions; `autonomous` — bounded retry + targeted-fix recovery (stall detection default 2h) — **but merge/PR progression and destructive/external side effects always require an explicit recorded human confirmation, even at `autonomous`**.

**Gates**: quality gates are `optional-group` graph nodes with config `{ name?, defaultOn?, maxRevisions?: number | "unbounded", phase?: "pre-merge" | "post-merge", template: {nodes, edges} }`. `phase: "pre-merge"` (default) blocks merge on failure with a fix→re-review loop budgeted by `maxRevisions`; `phase: "post-merge"` runs after a successful merge and failures are non-blocking. Gate nodes also carry `gateMode: "gate" | "advisory"` (**advisory by default**), `toolMode: "readonly" | "coding"` (readonly is a hard allowlist; violations fail closed as `READONLY_VIOLATION`), and verdicts use a structured final-line JSON envelope `{"verdict":"APPROVE|APPROVE_WITH_NOTES|REVISE","notes":"..."}`.

**Branch/worktree**: canonical branch `fusion/<taskid-lowercase>`; per-step foreach instances `fusion/<task>-step-<i>` (deterministic for crash-resume). Worktree paths are leased in a path-keyed registry for mutual exclusion + liveness.

**Plugin runtime interface** (verbatim from the Hermes plugin):

```typescript
export interface AgentRuntime {
  id: string;
  name: string;
  createSession(options: AgentRuntimeOptions): Promise<AgentSessionResult>;
  promptWithFallback(session, prompt, options?): Promise<void>;
  describeModel(session): string;
  dispose?(session): Promise<void>;
}
// AgentRuntimeOptions: cwd, systemPrompt, tools?: "coding"|"readonly",
//   onText/onThinking/onToolStart/onToolEnd callbacks,
//   defaultProvider/defaultModelId, fallbackProvider/fallbackModelId,
//   defaultThinkingLevel, skills?: string[], runtimeContext
```

## 5. coleam00/Archon — brief

Workflows are YAML files; the repo dogfoods 21 under its own `.archon/workflows/defaults/`. Verbatim node-schema reference: top level `name`, multi-line `description` with **`Use when:` / `Triggers:` / `NOT for:` router conventions**, optional `provider: claude|codex`, `model: small|medium|large` (tier names), `interactive: true`. Node types — exactly one of `prompt`, `bash`, `command` (references `.archon/commands/<name>.md`), `script` (`runtime: bun|uv`), `loop` (`{prompt, until: COMPLETION_SIGNAL, max_iterations, fresh_context}`), `loop_group` (sealed sub-DAG re-run until signal), `approval`, `cancel: "reason"`. Common options: `depends_on: [ids]`, `when: "$<node>.output == 'value'"`, `trigger_rule: all_success | one_success | all_done`, `timeout` (ms), and for AI nodes `model`, `allowed_tools`/`denied_tools`, `context: fresh`, `output_format` (inline JSON Schema, consumed as `$classify.output.issue_type` in `when:` clauses). Variables: `$ARGUMENTS`, `$ARTIFACTS_DIR`, `$<nodeId>.output`. Deterministic/AI mixing is idiomatic: `parse-request` (AI, small) → `fetch-issue` (bash + gh) → `classify` (AI, structured output, `allowed_tools: []`) → conditional branches → a bash guard with `trigger_rule: one_success` that *fails* if neither artifact exists ("an AI node that declines the task still exits 0").

**Approval gate** verbatim:

```yaml
- id: foundation-gate
  approval:
    message: "Answer the foundation questions above. Your answers will guide the research phase."
    capture_response: true      # reviewer's reply becomes $foundation-gate.output
  depends_on: [initiate]
```

**Override resolution** (README, verbatim): "Keep a workflow copyable by placing its YAML, commands, and scripts together under `.archon/workflows/<pack>/<workflow>/`… **Same-named workflow files in your repo override bundled defaults.**" So: repo `.archon/workflows/` > user `~/.archon/workflows/` > bundled defaults, matched by workflow name.

---

# SYNTHESIS — open-seed's workflow format

**Choose YAML files with a markdown-prompt escape hatch, not markdown-with-frontmatter.** The evidence: the two systems closest to open-seed's premise where *structure dominates* (tutti, Archon) use pure config; crewplane's markdown-body format is elegant but pays for it with a bespoke parser that CI validation must reimplement. Archon shows the winning middle: YAML for the DAG, `prompt: |` blocks inline for short prompts, and `prompt_file:` references into ordinary markdown files for long ones. That keeps `seed workflow validate` a plain schema check (JSON Schema over YAML), keeps prompts diffable as markdown, and matches the already-decided "validate preflight" direction. One workflow per file under `.seed/workflows/<name>.yaml`, with Archon's override rule verbatim: repo `.seed/workflows/` overrides template-bundled defaults by name.

**Proposed step schema** (borrowing: tutti's role/provider/model split + fail_mode + land gate; agent-runbook/crewplane's artifact closure; Archon's `when`/`trigger_rule`/structured output; Fusion's phase+advisory gate semantics):

```yaml
# .seed/workflows/<name>.yaml
schema_version: "1"
name: string                # must match filename
description: |              # include Use when: / NOT for: router hints (Archon convention)
inputs:
  - {name: issue, type: string, required: true, description: ...}
defaults: {harness: claude-code, model: sonnet}
budgets:
  max_wall_clock_minutes: 90
  max_step_retries: 2
steps:
  - id: kebab-case          # unique; depends_on BY ID (never index — tutti's usize indices
    role: implementer       #   are its worst design decision; ids survive reordering)
    harness: claude-code    # per-step override; validated against a known-harness list
    model: opus             # tier or exact id
    prompt: |               # exactly one of prompt | prompt_file | run  (XOR, validated)
    prompt_file: prompts/x.md
    run: "make check"       # deterministic command step
    tools: readonly|coding  # Fusion's readonly allowlist semantics
    output_format: {...}    # inline JSON Schema for AI steps (Archon)
    consumes: [plan]        # artifact names; must be produced by a step in depends_on (closure)
    produces:
      - {name: plan, file: artifacts/plan.md, schema: schemas/plan.schema.json}
    depends_on: [ids]       # DAG; parallel waves like tutti
    when: "steps.classify.output.type == 'bug'"
    trigger_rule: all_success|one_success|all_done
    on_fail: block|continue # tutti fail_mode closed/open, renamed honestly
    max_iterations: 5       # for loop groups: {until: ..., steps: [...]}
    gate:                   # gates expressed on the step they guard
      type: approval|review|checks
      # approval: {message, capture_response}          (Archon)
      # review:   {reviewer_role, verdict: APPROVE|APPROVE_WITH_NOTES|REVISE,
      #            max_revisions: 2|unbounded, blocking_severity: high}   (Fusion)
      # checks:   {required_ci: true, unresolved_threads: 0}              (tutti land gate)
```

**CI validate rules** (`seed workflow validate`, union of all five validators): (1) JSON Schema over the file — unknown fields fail; (2) `schema_version` match; (3) unique step ids; (4) DAG acyclicity + depends_on existence; (5) XOR of prompt/prompt_file/run; (6) referenced files/schemas exist; (7) **artifact closure**: every `consumes` name is `produces`-declared by a step reachable through `depends_on`; (8) role closure against the roles mapping; (9) harness/model values against the pinned registry; (10) budget sanity (fail ≥ warn); (11) loop steps require `until` + body + max_iterations; (12) template-token lint (no unresolvable references); (13) optionally `--with-harnesses`: check CLIs on PATH. Ship a crewplane-style **mock harness** so `seed workflow run --mock` works in CI with zero credentials — that single feature is why crewplane's validate story is credible.

**Runtime conventions to adopt**: checkpoint file per run at `.seed/state/runs/<run-id>.json` recording `{completed_steps (by id), step_results, output_files}` with tutti's resume semantics (refuse resuming a succeeded run; re-run failed steps only); artifacts under `.seed/state/runs/<run-id>/artifacts/`.

**Example workflow** (plan → implement → review-loop → land):

```yaml
schema_version: "1"
name: fix-issue
description: |
  Use when: a triaged issue needs a fix landed.
  NOT for: exploratory research or multi-issue batches.
inputs:
  - {name: issue, type: string, required: true, description: "Issue number or URL"}
defaults: {harness: claude-code, model: sonnet}
budgets: {max_wall_clock_minutes: 120, max_step_retries: 2}

steps:
  - id: fetch-issue
    run: "gh issue view {{inputs.issue}} --json title,body,labels,comments,url"
    produces: [{name: issue_json, file: artifacts/issue.json}]

  - id: plan
    role: planner
    model: opus
    depends_on: [fetch-issue]
    consumes: [issue_json]
    tools: readonly
    prompt_file: prompts/plan.md
    produces:
      - {name: plan, file: artifacts/plan.md, schema: schemas/plan.schema.json}

  - id: plan-gate
    depends_on: [plan]
    gate: {type: approval, message: "Review artifacts/plan.md. Approve to implement.", capture_response: true}

  - id: implement
    role: implementer
    depends_on: [plan-gate]
    consumes: [plan]
    tools: coding
    prompt: |
      Implement exactly the plan in {{artifacts.plan.path}}. Commit and push
      to the task branch. Report changed files and the pushed SHA.
    produces: [{name: change_summary, file: artifacts/change-summary.md}]

  - id: verify
    run: "make check"
    depends_on: [implement]
    on_fail: block

  - id: review-loop
    role: implementer
    depends_on: [verify]
    consumes: [plan, change_summary]
    gate:
      type: review
      reviewer_role: reviewer         # different harness via roles, e.g. codex
      max_revisions: 2
      blocking_severity: high
    prompt: |
      Address reviewer findings, rerun `make check`, push fixes.

  - id: land
    depends_on: [review-loop]
    gate: {type: checks, required_ci: true, unresolved_threads: 0}
    run: "seed land --pr"
```

**Corrections to earlier findings**: (1) design-options groups Fusion with checked-in workflow systems — its workflow graphs are DB-resident and dashboard-edited; only per-task `PROMPT.md` and memory files are repo content, so Fusion is prior art for *gate semantics and plan-artifact structure*, not for committed workflow files. (2) agent-runbook's "eight step types" is marketing; the schema has four types plus four decorator fields. (3) tutti — cited as "closest prior art" — resolves step `depends_on` by 1-based index, has no per-step budget, and its `[roles]` table maps role→runtime only (not role→model); open-seed's step schema is a superset, not a copy. (4) Archon's README says "19 default workflows"; the repo ships 21. (5) Every system that validates artifact handoffs validates *closure against the dependency graph*, not just schema existence — that stricter rule should be the standard.
