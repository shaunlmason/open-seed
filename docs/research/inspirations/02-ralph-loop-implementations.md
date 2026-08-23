# Deep Dive: Ralph Loop Implementations

> Implementation-grade deep dive for the open-seed design study, researched 2026-08-22.
> Covers ralphex, ralph-claude-code, dex, wreckit, and martin-loop. All findings quoted from
> actual source (shallow clones / raw file downloads), not READMEs.

# 1. umputun/ralphex (Go)

**Plan file format (real example, `docs/plans/long-lines-plan.md`; 57 more real plans in `docs/plans/completed/`):**

```markdown
# Fix Windows Command-Line Length Limits

## Overview
Windows has an 8191-character command-line limit ...

## Context
- `ClaudeExecutor` (Go): already fixed: passes prompt via `cmd.Stdin`
...

## Success Criteria
- CodexExecutor passes prompt via stdin, not as a CLI argument
- `make lint` and `make test` pass cleanly

### Task 1: Fix CodexExecutor to pass prompt via stdin
- [x] Add `stdin io.Reader` field to `execCodexRunner` struct
- [ ] Run `make test` and `make lint`
```

**Parsing rules** (`pkg/plan/parse.go`):
```go
taskHeaderPattern = regexp.MustCompile(`^###\s+(?:Task|Iteration)\s+([^:]+?):\s*(.*)$`)
checkboxPattern   = regexp.MustCompile(`^\s*-\s+\[([ xX])\]\s*(.*)$`)
titlePattern      = regexp.MustCompile(`^#\s+(.*)$`)
```
Key details: checkboxes may be indented; lines inside CommonMark code fences are skipped so example checkboxes aren't parsed; an `##` (h2) heading closes the current task so checkboxes under `## Success Criteria` aren't attributed to a task; `###`/`####` are subsections and do NOT close a task. Task statuses: `pending|active|done|failed`.

**Crucial finding: "first unchecked task" is located by the *agent*, not the Go code.** The task prompt (`pkg/config/defaults/prompts/task.txt`) instructs: *"Read the plan file at {{PLAN_FILE}}. Find the FIRST Task section (### Task N: or ### Iteration N:) that has uncompleted checkboxes ([ ])."* The Go parser is used for progress display and completion counting. Same for `## Validation Commands`: it is a **documented README convention only**, no Go code parses or executes it. The task prompt says: *"Run the test and lint commands specified in the plan (e.g., 'cargo test', 'go test ./...')"*. Validation is delegated to the agent. This corrects survey-level assumptions that ralphex executes validation commands itself.

**Non-automatable escape hatch** (verbatim from task.txt): *"If a Task section has [ ] checkboxes you cannot complete (manual testing, deployment verification, external checks): mark them [x] with a note like '[x] manual test (skipped - not automatable)' and proceed."* And for leftover non-task checkboxes: *"either satisfy those items and mark them [x] if actionable, or output <<<RALPHEX:ALL_TASKS_DONE>>> if they are verification-only."*

**Completion sentinels** (`pkg/status/status.go`):
```go
Completed  = "<<<RALPHEX:ALL_TASKS_DONE>>>"
Failed     = "<<<RALPHEX:TASK_FAILED>>>"
ReviewDone = "<<<RALPHEX:REVIEW_DONE>>>"
CodexDone  = "<<<RALPHEX:CODEX_REVIEW_DONE>>>"
Question   = "<<<RALPHEX:QUESTION>>>"
PlanReady  = "<<<RALPHEX:PLAN_READY>>>"
PlanDraft  = "<<<RALPHEX:PLAN_DRAFT>>>"
```
The review_first prompt has notable sentinel discipline language: *"A signal marker is irreversible... If you emit a marker it MUST be the final non-empty line of your output... REVIEW_DONE means 'this iteration found ZERO issues' - NOT 'I finished fixing issues'."* Three paths: A (no issues → REVIEW_DONE), B (fixed issues → plain summary, **no marker**, loop runs another verification round), C (can't fix → TASK_FAILED).

**Review pipeline**: Phase 1 tasks → Phase 2 `review_first.txt` launches 5 parallel agents via `{{agent:quality}} {{agent:implementation}} {{agent:testing}} {{agent:simplification}} {{agent:documentation}}` (agent bodies embedded in `pkg/config/defaults/agents/*.txt`; overridable in `~/.config/ralphex/agents/`) → Phase 3 external review (codex, `read-only` sandbox by default) → Phase 4 `review_second.txt` with only quality+implementation, critical/major only. The orchestrating agent itself dedupes, verifies each finding against code ("Check full context (20-30 lines around)"), classifies CONFIRMED vs FALSE POSITIVE, fixes, commits `fix: address code review findings`.

**Config** (`.ralphex/config`, same format as `~/.config/ralphex/config`; precedence CLI > local > global > embedded): plain `key = value`, full-line `#` comments only. Notables with defaults: `iteration_delay_ms = 2000`; `task_retry_count = 1`; `max_iterations` default 50; `max_external_iterations = 0` (auto = `max(3, max_iterations/5)`); `review_patience = 0` (disabled); `plans_dir = docs/plans`; `move_plan_on_completion = true`; per-phase `plan_model`/`task_model`/`review_model` with `model[:effort]` syntax; `session_timeout`/`idle_timeout` (Go durations); `commit_trailer`.

**Stalemate/review-patience** (`pkg/processor/phase/git_state.go`): before/after each external-review round it snapshots `head` (and working-tree `diff`); `unchanged = after.head == before.head && after.diff == before.diff`; counter increments on unchanged, resets to 0 on change; at `unchangedRounds >= ReviewPatience` it logs *"stalemate detected after %d unchanged rounds, external review terminated early"*: this is "Claude wins the dispute" against the external reviewer.

**Worktree mode**: `git worktree add .ralphex/worktrees/<branch> -b <branch>` (branch derived from plan filename); ralphex writes `.ralphex/.gitignore` containing `.gitignore\nprogress/\nworktrees/\n` so state stays untracked.

# 2. frankbria/ralph-claude-code (Bash, 154KB `ralph_loop.sh` + `lib/`)

**PROMPT.md template** (ships in `templates/PROMPT.md`, copied to `.ralph/PROMPT.md`): full verbatim status block schema:

```
---RALPH_STATUS---
STATUS: IN_PROGRESS | COMPLETE | BLOCKED
TASKS_COMPLETED_THIS_LOOP: <number>
FILES_MODIFIED: <number>
TESTS_STATUS: PASSING | FAILING | NOT_RUN
WORK_TYPE: IMPLEMENTATION | TESTING | DOCUMENTATION | REFACTORING
EXIT_SIGNAL: false | true
RECOMMENDATION: <one line summary of what to do next>
---END_RALPH_STATUS---
```

EXIT_SIGNAL=true requires ALL of: all fix_plan items `[x]`, tests passing, no errors/warnings, all specs implemented, nothing meaningful left. The template also embeds six **"Exit Scenarios (Specification by Example)"**: Given/When/Then examples teaching the model exactly which block to emit for completion, test-only loops, recurring errors, no-work-remaining, progress, and external blockers. Other verbatim principles: "ONE task per loop", "LIMIT testing to ~20% of your total effort per loop", protected-files list (`.ralph/`, `.ralphrc`).

**Dual-gate exit** (`ralph_loop.sh` `should_exit_gracefully()`, priority order, thresholds hardcoded: `MAX_CONSECUTIVE_TEST_LOOPS=3`, `MAX_CONSECUTIVE_DONE_SIGNALS=2`):
0. permission denials → halt `permission_denied` (unless every denial is a compound-command CLI-matching artifact → warn and continue);
1. `test_only_loops >= 3` → `test_saturation`;
2. `done_signals >= 2` → `completion_signals`;
3. safety breaker: `completion_indicators >= 5` → forced `safety_circuit_breaker` ("catches cases where Claude signals completion 5+ times but the normal exit path didn't trigger");
4. **the dual gate**: `completion_indicators >= 2 && claude_exit_signal == "true"` → `project_complete`;
5. plan gate: `total_items > 0 && completed == total` counting only **blocking** unchecked items → `plan_complete`.

Signal accumulator file `.ralph/.exit_signals`: `{"test_only_loops": [], "done_signals": [], "completion_indicators": []}`. `completion_indicators` only accumulates when Claude explicitly sets EXIT_SIGNAL=true. The analyzer (`lib/response_analyzer.sh`) extracts the block from Claude CLI JSON `.result` text, anchored `^[[:space:]]*(---RALPH_STATUS---|RALPH_STATUS:)` to avoid false hits in commit messages, and respects an explicit `EXIT_SIGNAL: false` even when STATUS is COMPLETE.

**Optional sections** (`fix_plan.md` template + `_count_blocking_unchecked` awk): default `OPTIONAL_SECTIONS="Optional,Future,Future Enhancements,Nice to Have"`; a heading whose trimmed lowercase title matches an entry marks its section optional; optional context persists into deeper subsections and closes at the next same-or-higher heading level; unchecked items inside don't block exit. Checkbox regexes deliberately strict: `^[[:space:]]*- \[ \]` / `- \[[xX]\]` (so `[2026-01-29]` dates don't count).

**Circuit breaker** (`lib/circuit_breaker.sh`, "Based on Michael Nygard's 'Release It!' pattern"): states `CLOSED`/`HALF_OPEN`/`OPEN`; thresholds `CB_NO_PROGRESS_THRESHOLD=3`, `CB_SAME_ERROR_THRESHOLD=5`, `CB_OUTPUT_DECLINE_THRESHOLD=70` (%), `CB_PERMISSION_DENIAL_THRESHOLD=2`, `CB_COOLDOWN_MINUTES=30` (OPEN→HALF_OPEN auto-recovery), `CB_AUTO_RESET=false`. State file `.ralph/.circuit_breaker_state`:
```json
{"state":"CLOSED","last_change":"...","consecutive_no_progress":0,"consecutive_same_error":0,
 "consecutive_permission_denials":0,"last_progress_loop":0,"total_opens":0,"reason":"","current_loop":0}
```
plus `.circuit_breaker_history` (JSON array of transitions).

**.ralphrc** (full template captured): PROJECT_NAME/TYPE, `MAX_CALLS_PER_HOUR=100`, `CLAUDE_TIMEOUT_MINUTES=15`, `CLAUDE_OUTPUT_FORMAT="json"`, `ALLOWED_TOOLS` (deliberately enumerates safe git subcommands: "broad Bash(git *) allows destructive commands like git clean/git rm"), `SESSION_CONTINUITY=true`, `SESSION_EXPIRY_HOURS=24`, `TASK_SOURCES="local"` (also beads/github), CB_* thresholds, sandbox (Docker/E2B) and sync-filter options. Env vars > .ralphrc > defaults.

**status.json** (rewritten each loop): `timestamp, loop_count, calls_made_this_hour, max_calls_per_hour, tokens_used_this_hour, max_tokens_per_hour, last_action, status, exit_reason, sandbox, next_reset`. **metrics.jsonl** (append-only, `logs/metrics.jsonl`): `{"timestamp":"...","loop":N,"duration":N,"success":true,"calls":N}`.

**AGENT.md auto-maintenance**: purely prompt-driven: "Keep .ralph/AGENT.md updated with build/run instructions", no script mechanics beyond seeding the template.

# 3. francescoalemanno/dex (Rust)

**Plan format** `dex plan` produces (from `prompts/plan.txt`): identical skeleton to ralphex's make_plan (Title / Overview / Context / Development Approach / Implementation Steps with `### Task N:` + `**Files:**` lists + checkboxes / `Task N: Verify acceptance criteria` / `Task N+1: Update documentation`): dex is a conscious descendant. It adds an explicit pre-write validation checklist ("Tasks are reasonably sized (aim for 3-7 items)", "Task section checkboxes are automatable by the agent", YAGNI list: "No backwards compatibility or fallbacks unless explicitly requested").

**But dex's parser is more permissive than ralphex's** (`src/plan.rs`): *any* heading `#{1,6}` starts a task group; a group is a "task" iff its body contains ≥1 checkbox (`^(\s*)-\s+\[([ xX])\]\s+(.*)$`); `next_open_task` = first group with `open > 0`. No `Task N:` naming requirement, no code-fence skipping (a regression vs ralphex worth noting).

**questions.md flow**: planner writes `.dex/questions.md`, one `Q: question? [Option 1 | Option 2 | Option 3]` line, then STOPs; the CLI parses it, prompts the user, appends the answer into `feedbacks.json` (JSON array of strings), deletes questions.md, and reruns the plan prompt with a feedback section.

**Stalemate + stall-note (the standout mechanism)**: `src/phases.rs`, `IMPLEMENTATION_STALEMATE_LIMIT: usize = 4`. Each iteration records git HEAD before/after and `(open, total)` checkbox counts before/after. It then generates a `StallNote` injected into the *next* prompt keyed on the 2×2 of (made_commits, made_plan_progress): verbatim, e.g. no/no: *"the previous iteration produced no git commits and ticked no checkboxes. You must either (a) finish the current task and commit, or (b) if it is genuinely blocked, mark the offending checkbox `- [x] <description> (sipped — blocked: <reason>)` so the loop can move on. Do NOT silently re-attempt the same approach without changes."*: plus dedicated messages for committed-but-unticked and ticked-but-uncommitted. If `(open,total)` unchanged for 4 consecutive iterations → hard error: `"STALEMATE: total plan steps and remaining plan steps were unchanged for N consecutive implementation iterations."` No sentinel needed for completion: **completion is detected purely by `open == 0` (plan-delta gate)**: the impl prompt just says "STOP".

**Reviewers** (`prompts/reviewers.json`, overridable via `.dex/reviewers.json`): two tiers. `broad` = 5 (quality, implementation, simplification, testing, documentation: condensed ports of ralphex's agent files, each ending "Report problem only — no positive observations"); `focused` = 2 (critical-correctness, critical-coverage). Pipeline: 1 broad round → fixer; then up to `MAX_FOCUSED_ROUNDS = 3` focused rounds, exiting early when all focused reviewers write "ZERO FINDINGS". Reviewers write files `.dex/review-<name>-r<round>.md` in a fixed plain-text format (`- [critical|major|minor] path:line / Issue / Impact / Fix`); a clean review is `- none` + `- ZERO FINDINGS` (matched case-insensitively). Review files double as resume state.

**Fixer/false-positive filter**: single `prompts/fix.txt`: the fixer both filters and fixes: dedupe by file:line, "Verify EVERY Finding... Check full context (20-30 lines around)... Classify as CONFIRMED / FALSE POSITIVE", fix pre-existing issues too, always commit `fix: address code review findings`.

**`.dex/` state files**: `plan.md`, `plan-request.txt`, `questions.md` (transient), `feedbacks.json`, `impl-commits.jsonl` (one `ImplCommit` per line with `before` SHA: the review **base ref is derived from the first line's `before` field**; a legacy `base_ref` config field is explicitly ignored: this corrects the earlier "review-base-ref.txt" survey finding), `review-<name>-r<n>.md`, `config.json` (`{cli, timeout, clis: {name: {cmd, args...}}}`: CLI-agnostic), `research.jsonl`.

**`dex research`**: agent makes ONE committed change per iteration ("Do NOT run the benchmark yorself — Dex handles benchmarking... Dex... keeps or reverts"); dex runs the benchmark command and parses **`METRIC` lines** with `(?m)^METRIC\s+([\w.µ]+)=(\S+)\s*$` (last occurrence wins per name; NaN/inf rejected; missing METRIC line → revert). Ledger `research.jsonl` entries: `{run, commit, metric, status, description, timestamp, confidence}`. **Dead-end ledger**: the most recent `MAX_DEAD_ENDS` entries with status `discard|crash|checks_failed`, rendered `- <description> (<status>)` into the prompt under *"## Dea Ends — do NOT retry these approaches"*, with commit-message bodies serving as the "what I learned / what to try next" memory.

# 4. mikehostetler/wreckit (TypeScript, zod schemas in `src/schemas.ts`)

**State machine** (`ItemStateSchema`): `"idea" → "researched" → "planned" → "implementing" → "critique" → "in_pr" → "done"`.

**`.wreckit/` layout**: `config.json`, `config.local.json`, `index.json`, `batch-progress.json`, `prompts/` (10 overridable templates: ideas, research, plan, implement, critique, pr, interview, dream, media, strategy), `items/NNN-slug/` containing `item.json`, `research.md`, `plan.md`, `prd.json`, `progress.log`. A `prompt.md` path helper exists but the current flow renders prompts **in memory** from templates + variables per phase (correction to any survey claim that prompt.md is materialized per item).

**item.json** (`ItemSchema`, key fields): `schema_version, id, title, section?, state, overview, branch|null, pr_url|null, pr_number|null, last_error|null, created_at, updated_at`, structured context (`problem_statement, motivation, success_criteria[], technical_constraints[], scope_in_scope[], scope_out_of_scope[], priority_hint(low|medium|high|critical), urgency_hint`), `rollback_sha?`, audit (`completed_at, merged_at, merge_commit_sha, checks_passed`), and `depends_on[]`/`campaign` for orchestration.

**prd.json** (`PrdSchema`): `{schema_version: 1, id, branch_name, user_stories: [{id, title, acceptance_criteria: string[], priority: number, status: "pending"|"done", notes}]}`. The implement prompt loop: "Pick the highest priority pending story... Ensure all acceptance criteria are met... Call the `update_story_status` tool with the story ID and status 'done'... Append learnings/notes to {{item_path}}/progress.log... When ALL stories have status 'done', output... {{completion_signal}}": default signal `<promise>COMPLETE</promise>` (configurable per agent adapter).

**Phase-output validation with retry**: research phase retries up to 3 times, appending *"CRITICAL: Your previous attempt failed validation with the following errors: ... You MUST fix these issues"*: a validator gates artifact quality, not just existence.

**doctor** (`src/doctor.ts`): `diagnose()` runs config validation (zod), prompt checks, per-item checks (missing/invalid item.json; artifact accessibility; state-vs-artifact consistency: e.g. state says `planned` but plan.md missing: many `fixable: true`, fixed by **state regression** with backup sessions), dependency DFS cycle detection over `depends_on`, index consistency, stale-PID detection. Doctor is also invoked mid-run as "healing" (`runAgentWithHealing`, config `doctor: {enabled, auto_repair, max_retries, timeout_ms}`).

**config.json** (defaults): `base_branch: "main"`, `branch_prefix: "wreckit/"`, `merge_mode: "pr"|"direct"`, `agent` (process | claude-sdk | amp | codex | sprite), `max_iterations: 100`, `timeout_seconds: 3600`, plus `pr_checks.commands[]`, `story_scope` (path globs with excludes like `*.lock`).

# 5. Keesan12/martin-loop (TypeScript monorepo)

Lifecycle (docs, verbatim): `DEFINE -> PREFLIGHT -> CONTROL -> VERIFY -> RECOVER -> PROVE -> ANALYZE`; canonical flow "Definition of Done -> Controlled Run -> Verified Handoff" with outcomes `VERIFIED`, `STOPPED`, `NEEDS REVIEW`. `martin run` auto-chains `doctor`, `session-start`, `preflight`; the MCP run tool "expects matching doctor, plan, and preflight receipts for the same task before it will execute."

**Task contract** (`LoopTask`, `packages/contracts/src/index.ts`): `title, objective, repoRoot?, verificationPlan: string[], verificationTimeoutMs?, verificationStack?, mutationMode? ("edit"), definitionOfDonePreSatisfied?, executionProfile? ("strict_local"|"ci_safe"|"staging_controlled"|"research_untrusted"), allowedNetworkDomains?, approvalPolicy? ({dependencyAdds?, migrations?, configChanges?, externalWrites?}), agentExecutionIntent?, providerExecutionTimeoutMs?, allowedPaths?: string[] (globs, empty = unrestricted), deniedPaths?: string[], acceptanceCriteria?: string[]` ("injected into the prompt as a checklist").

**The 13 FailureClass values** (verbatim): `logic_error, hallucination, syntax_error, type_error, test_regression, scope_creep, no_progress, repo_grounding_failure, verification_failure, environment_mismatch, budget_pressure, safety_leash_blocked, sandbox_write_blocked`. Each attempt records `failureClass`, an `intervention` from `compress_context | change_model | tighten_task | switch_adapter | run_verifier | escalate_human | stop_loop`, and a `diagnosticHint` "injected into the next attempt's prompt": the recovery analogue of dex's StallNote. `LoopLifecycleState` doubles as an exit-reason taxonomy: `created, running, verifying, completed, budget_exit, diminishing_returns, stuck_exit, human_escalation, wall_clock, error_threshold, external_event`.

**martin.config.yaml** (example, verbatim): `policyProfile: balanced; budget: {maxUsd: 8, softLimitUsd: 5, maxIterations: 3, maxTokens: 20000}; governance: {destructiveActionPolicy: approval, telemetryDestination: local-only, verifierRules: [pnpm test, pnpm lint]}`.

**run-receipt.json** (`martin.share-receipt.v1`): `schemaVersion, generatedAt, loop` (loopId, objective, status/lifecycleState, attempts, spend/budget), `receiptIntegrity` (verdicts: `verified, unsigned, tamper_detected, relocated, material_missing, selector_noncanonical`), `verification`, `receipt` (next-safe-action/risk posture), `artifacts`, `proofCard`, `warnings`. Persistent run store: `~/.martin/runs/<loopId>/loop-record.json` (full record: task, budget, cost, attempts[], events[], terminationEnvelope) + JSONL event logs.

# SYNTHESIS

## (a) Unified plan-file grammar for open-seed

Take dex/ralphex's shared skeleton, ralphex's parser rigor, and frankbria's optional-section semantics:

```markdown
# <Title>                                  ← first h1 = plan title

## Overview                                ← prose, agent context
## Context                                 ← files involved / patterns / dependencies

## Validation Commands                     ← one fenced or bulleted command per line
- `make test`
- `make lint`

## Success Criteria                        ← checkboxes allowed; NOT attached to any task
- [ ] <criterion>

## Implementation Steps

### Task 1: <Title>                        ← `### Task N:` (accept `Iteration N:`)
**Files:**                                 ← optional per-task metadata
- Modify: `path/to/file`
- [ ] step
- [ ] write tests for this task
- [ ] run validation commands - must pass before Task 2

## Optional                                ← unchecked items here never block exit
- [ ] nice-to-have
```

Parser rules to adopt: ralphex's regexes verbatim; skip code-fenced lines (ralphex has it, dex lost it: a real bug class); h2 closes the current task; frankbria's `OPTIONAL_SECTIONS` semantics (case-insensitive title match, context persists into subsections, closes at same-or-higher heading). Semantics to adopt: the tool parses only for **counting and stall detection**; task selection and validation execution are delegated to the agent via prompt (all three projects converged on this). Adopt ralphex/dex's skip convention as canon: `- [x] <desc> (skipped: blocked: <reason>)`. `## Validation Commands` stays advisory-to-agent, but open-seed's loop script *can* additionally execute it as a hard gate, none of these projects do; martin-loop is the one that runs verifier commands itself (`verifierRules`), and that's its differentiator worth stealing.

## (b) Exit-detection decision table

| # | Gate | Signal source | Threshold (as shipped) | Action |
|---|------|--------------|------------------------|--------|
| 1 | Permission denial | CLI output analysis | ≥1 real denial (frankbria) | halt, `permission_denied` |
| 2 | Plan complete | checkbox count, excluding Optional sections | blocking-unchecked == 0 && total > 0 (frankbria #5; dex's *only* completion gate) | success exit |
| 3 | Sentinel | `<<<...ALL_TASKS_DONE>>>` last non-empty line / `EXIT_SIGNAL: true` block | dual gate: explicit signal **AND** ≥2 accumulated completion indicators (frankbria) | success exit |
| 4 | Sentinel safety valve | consecutive explicit completion signals | ≥5 (frankbria) or ≥2 done_signals | forced exit |
| 5 | Stalemate by plan delta | (open,total) unchanged AND no new commits | warn+StallNote at 1, hard fail at 4 consecutive (dex); frankbria CB opens at 3 no-progress loops | error exit `STALEMATE` |
| 6 | Same-error loop | error fingerprint match | 5 consecutive (frankbria CB) | circuit OPEN |
| 7 | Test-only saturation | WORK_TYPE=TESTING + 0 files modified | 3 consecutive (frankbria) | exit `test_saturation` |
| 8 | Review stalemate | git HEAD+diff unchanged across review rounds | `review_patience` N rounds (ralphex, default disabled), focused-round cap 3 (dex), external cap max(3, max_iter/5) (ralphex) | accept current state |
| 9 | Hard budgets | iterations / wall clock / USD / tokens | max_iterations 50 (ralphex) / 100 (wreckit) / 3 (martin); timeout 15 min/loop (frankbria), 3600s (wreckit); maxUsd + maxTokens (martin only) | `budget_exit` / `wall_clock` |
| 10 | Recovery | OPEN → HALF_OPEN cooldown | 30 min (frankbria) | resume monitored |

Recommended composition for open-seed: gate 2 as ground truth, gate 3 as accelerator (never sufficient alone: that's the dual-gate lesson), gates 5–7 as the circuit breaker (dex's escalating StallNote *before* tripping is the best-designed piece here: cheap, prompt-level, self-healing), gate 9 as the outermost envelope with martin-style exit-reason taxonomy in the run record.

## (c) Minimal state files for a loop script

1. **The plan file itself**: dex proves checkbox deltas make it the primary progress store; everything else is derivable.
2. **Append-only run log**, JSONL, one line per iteration (frankbria `metrics.jsonl` + dex `impl-commits.jsonl` merged): `{"ts", "loop", "task_header", "before_sha", "after_sha", "open_before", "open_after", "duration_s", "success", "exit_signal"}`. The first line's `before_sha` doubles as the review base ref (dex's trick, no separate base-ref file needed).
3. **Circuit-breaker state**, single small JSON (frankbria's schema is the template): `{"state": "CLOSED|HALF_OPEN|OPEN", "consecutive_no_progress", "consecutive_same_error", "last_change", "reason", "total_opens"}`.
4. **status.json**, rewritten each loop for dashboards/monitors (frankbria's field set).
5. **Dead-end ledger**: dex shows it can be *derived*: failed/discarded entries from the run log rendered into the next prompt as "do NOT retry", so open-seed's DEADENDS.md can be either agent-maintained (ralph-style) or generated from run-log entries with `status ∈ {discard, crash, checks_failed}`; the dual approach (generated skeleton + agent-appended reasoning, with commit bodies as the learning record) is the strongest option.
6. Transient: `questions.md` (dex's `Q: ...? [A | B | C]` one-line format is the simplest human-gate protocol found) and `feedbacks.json`.

**Corrections to earlier survey findings**: (1) ralphex does not itself execute `## Validation Commands`: advisory convention; nor does Go code "locate the first unchecked task" for execution: the agent does, per prompt. (2) dex has no `review-base-ref.txt`: base ref comes from `impl-commits.jsonl`. (3) dex has no separate false-positive-filter prompt: filtering is folded into the fixer prompt. (4) wreckit's per-item `prompt.md` is a vestigial path; prompts are rendered in memory. (5) frankbria's dual-gate thresholds are hardcoded in `ralph_loop.sh`, not `.ralphrc`-configurable (only CB_* thresholds are).
