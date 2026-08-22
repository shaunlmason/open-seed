# Deep Dive: Governance & Convention Formats

> Implementation-grade deep dive for the open-seed design study, researched 2026-08-22.
> Covers loop-engineering, orc, antfarm, loki-mode, and kodo. Quoted from actual source.
>
> **Erratum (design review, 2026-08-22):** the proposed `guardrails.yaml` skeleton in this
> file's synthesis is superseded by `docs/design-options.md` D4.1 on two points: (1) the
> `auto_merge_allowlist` example (`"**/*.md"`) is unsafe in open-seed's design, where most of
> the instruction surface is markdown — the orchestration control surface is never
> auto-mergeable regardless of allowlist entries; (2) the self-protection entry
> `"guardrails.yaml"` must be `.seed/guardrails.yaml` (and the rest of `.seed/**`) to match
> the repository layout. Receipts are additionally written only by the gate, never by agents,
> and verified by `seed receipt verify` in CI.

## 1. cobusgreyling/loop-engineering (priority — the pattern book)

The repo is exactly what we hoped: a mostly-docs pattern book that dogfoods its own loops. Root files: `LOOP.md`, `STATE.md`, `gate.yaml`, `loop-budget.md`, `loop-constraints.md`, `loop-run-log.md`, plus `patterns/`, `templates/` (24 templates), `docs/`, `starters/`, and 11 CLI tools under `tools/` (`loop-gate`, `loop-audit`, `loop-sync`, `loop-init`, `loop-cost`, etc.).

**Correction to the earlier survey:** there is no `.budget.json` or `run-log.json`. Budget and run-log are **markdown files** (`loop-budget.md`, `loop-run-log.md`) — the run log embeds a JSON entry schema per run. The only pure-machine files are `gate.yaml` and `patterns/registry.yaml`.

### gate.yaml (verbatim, repo root)

```yaml
# Machine-readable twin of docs/safety.md's Path Denylist and Auto-Merge
# Policy sections. Used by tools/loop-gate. Keep in sync with docs/safety.md
# when either changes.
version: 1

denylist:
  - ".env"
  - ".env.*"
  - "**/secrets/**"
  - "**/credentials/**"
  - "**/*_key*"
  - "**/*_secret*"
  - ".terraform/**"
  - "k8s/production/**"
  - "**/migrations/**"
  - "auth/**"
  - "payments/**"
  - "billing/**"

maxFiles: 10

autoMergeAllowlist:
  - "docs/**"
  - "**/*.md"
  - "**/*.test.mjs"
```

Template comments: "`Escalate instead of auto-merging when a change touches more than this many files`" (maxFiles) and "`--action auto-merge only proceeds if every changed path matches one of these`". Enforcement: "`loop-gate check` mechanically enforces the denylist + auto-merge allowlist above from `gate.yaml`" (LOOP.md).

### STATE.md.template (verbatim)

```markdown
# Loop State — {{PROJECT_NAME}}

Last run: (set by loop on each run)

## High Priority (loop is acting or waiting on human)
<!-- Format:
- [ ] ID — one-line description
  Loop action: what the loop did last
  Human decision: (if any)
-->

## Watch List
<!-- Items to monitor but not act on yet -->

## Recent Noise (ignored this run)
<!-- Brief list — helps tune triage skill -->

---
Run log: (timestamp) | findings | actions | escalations
---
```

The live root `STATE.md` follows this exactly, adding a machine-updated `Last run:` line and quantified goals ("Maintain loop readiness score ≥ 58 (current: **100**, level **L3**)").

### Run-log entry schema (verbatim from the template)

"Append one entry per run. Prune entries older than 30 days."

```json
{
  "run_id": "2026-06-09T08:15:00Z",
  "pattern": "daily-triage",
  "duration_s": 45,
  "items_found": 4,
  "actions_taken": 1,
  "escalations": 0,
  "tokens_estimate": 52000,
  "outcome": "report-only | fix-proposed | escalated | no-op"
}
```

### Budget file (`templates/loop-budget.md.template`)

A per-loop table — `| Loop | Max runs/day | Max tokens/day | Max sub-agent spawns/run |` — e.g. `Daily Triage | 2 | 100k | 0 (L1) / 2 (L2)`, `PR Babysitter | 288 | 2M | 3`, `CI Sweeper | 96 | 1M | 3`. Then **On budget exceed**: "1. Pause all schedulers … 2. Append event to `loop-run-log.md` 3. Notify human (Slack / issue / STATE.md High Priority)". **Kill switch**: "Command or issue label: `loop-pause-all`. Resume only after human clears the flag in STATE.md." Safety doc adds: "humans only may edit `loop-budget.md`".

### patterns/registry.yaml — per-pattern schema

Each of the seven patterns is an entry with keys: `id, name, file, goal, cadence, risk, tools, skills, state, phases, human_gates, starter, week_one_mode, token_cost`, and a `cost:` block `{tokens_noop, tokens_report, tokens_action, stable_fraction, suggested_daily_cap, early_exit_required}`. The seven loop designs:

| Pattern | Cadence/Risk | Phases | Human gates | Week-one |
|---|---|---|---|---|
| **Daily Triage** | 1d–2h, low | report → act-small-wins → escalate | design-decisions, multi-file-refactors | L1 |
| **PR Babysitter** | 5m–15m, medium | discover → triage → fix → verify → notify | security, payments, auth, max-fix-attempts | L1 |
| **CI Sweeper** | 5m–15m, medium | detect → classify → fix → verify → escalate | infra-failures, max-attempts, security-tests | L2 |
| **Dependency Sweeper** | 6h–1d, medium | scan → triage-risk → patch-safe → verify-worktree → escalate-risky | major-bumps, high-sev-cve, denylisted-packages, max-attempts | L2 |
| **Changelog Drafter** | 1d, low | scan-merges → categorize → draft → review → publish | breaking-changes, security, major-features, marketing-sensitive | L1 |
| **Post-Merge Cleanup** | 1d–6h, low | scan-merges → prioritize → fix-small → ticket-large | architectural-debt, feature-flags, large-diffs | L1 |
| **Issue Triage** | 2h–1d, low | discover → dedupe → score → propose-labels → human-review | security, p0-p1, ambiguous-duplicates, stale-closures | L1 |

Note `early_exit_required: true` on the three action-heavy loops — the cheap-triage-before-expensive-work rule made mechanical.

### Autonomy tiers

Defined operationally rather than in one canonical block: **L1 = report-only, no auto-fix**; **L2 = assisted/cautious fixes** ("Worktrees for suggested fixes; verifier required; no auto-merge by default"); **L3 = unattended**. The rollout rule: "Phased rollout: **L1 report → L2 assisted fixes → L3 unattended**", and anti-pattern #4 mandates "L1 report-only week one. Measure triage accuracy before enabling L2."

### Drift detection (`tools/loop-sync`)

Checks four areas: required files present (STATE.md, LOOP.md, AGENTS.md); cross-file consistency between STATE.md and LOOP.md; `.claude/skills/` existence + version data in SKILL.md files; missing references/orphaned files. Outputs a 0–100 health score (Healthy 90–100 / Warning 70–89 / Critical 0–69), with `--auto-fix`, `--dry-run`, `--json` for CI.

### CI validation

`.github/workflows/validate-patterns.yml` — checkout, Node 22, `bash scripts/ci-validate-gates.sh`. The script enforces: **registry↔files bijection** (every `patterns/*.md` registered; every registry entry has a file); **required sections** in each pattern (`## Scheduling`, `## Required Skills`, verifier-strategy language present); **template existence**; then schema validation via `ajv` plus tool builds/tests.

### Failure-mode catalog (verbatim names, S1 annoying / S2 harmful / S3 critical)

Infinite Fix Loop (S2), State Rot (S1→S2), Verifier Theater (S2), Notification Fatigue (S1→S2), Token Burn (S1), Over-Reach / Wrong Scope (S2→S3), Comprehension Debt Spiral (S2 long-term), Cognitive Surrender (S2 cultural), Parallel Collision (S2), Escalation Failure (S2). Signature mitigations: "Hard cap on attempts (e.g. 3) → escalate to human", "Different instructions: 'find reasons to reject'", "`isolation: worktree` for all code-editing sub-agents", "Alert if item in [High Priority] section >24h".

### Anti-patterns (verbatim names)

1. Same agent implements and verifies ("Verifier default stance: REJECT"); 2. No attempt cap; 3. Vague triage output; 4. L3 before L1 quality; 5. Shared state without schema; 6. MCP with write-everything scope; 7. No kill switch; 8. Fixing flakes with code; 9. Auto-merge without allowlist; 10. No run log.

---

## 2. spencermarx/orc

### Complete `.orc/config.toml` schema (resolution: `{project}/.orc/config.toml` > `config.local.toml` > global)

```toml
[defaults]
agent_cmd = "auto"
agent_flags = ""
agent_template = ""
yolo_flags = ""
max_workers = 3

[planning.goal]
plan_creation_instructions = ""
bead_creation_instructions = ""
when_to_involve_user_in_plan = ""

[dispatch.goal]
assignment_instructions = ""

[approval]
ask_before_dispatching = "ask"
ask_before_reviewing = "auto"
ask_before_merging = "ask"

[review.dev]
review_instructions = ""
how_to_determine_if_review_passed = ""
max_rounds = 3

[review.goal]
review_instructions = ""
how_to_determine_if_review_passed = ""
how_to_address_review_feedback = ""
max_rounds = 3

[branching]
strategy = ""
[worktree]
setup_instructions = ""
[delivery.goal]
on_completion_instructions = ""
when_to_involve_user_in_delivery = ""
[tickets]
strategy = ""
[notifications]
system = false
sound = false
```

Every phase-instruction key "accepts natural language or slash commands; empty strings trigger defaults" — governance-as-prompt-fragments, with `max_rounds = 3` as the only hard number in the review loop.

### Roles (docs/personas.md + concepts.md)

Three-tier orchestration: **Root** ("routes requests across multiple projects", never codes), **Project** ("decomposes your request into goals, dispatches goal orchestrators, monitors progress"), **Goal** ("owns one goal end-to-end: delegates planning, creates beads, dispatches engineers, runs the review loop, merges, delivers"), plus **Engineer** ("isolated implementation within a single bead") and **Reviewer** ("code review verdicts"). Personas are markdown system prompts; project overrides at `{project}/.orc/{role}.md` layer additively over defaults.

### Slash commands

Shipped at `.claude/commands/orc/`: `plan, dispatch, done, blocked, check, complete-goal, feedback, index, leave, status, view`. `/orc:plan` is a five-phase script: Investigate/Scout → Decompose (goals get `feat|fix|task` type + kebab-case name; beads get title/description/acceptance-criteria/files-touched/dependencies) → Propose as dependency table + ASCII graph → **"Do NOT create branches until the user approves"** → Create (`git branch feat/<goal-name>` or `bd create --title "<title>" --desc "<description>"`), then check `echo $ORC_YOLO` — if `ORC_YOLO=1`, self-invoke `/orc:dispatch` with "No questions, no delays."

`/orc:done` (engineer, verbatim): run tests ("Do NOT signal for review with failing tests") → self-review the diff → lint/format → conventional commit → signal via a **file-based protocol**: `echo "review" > .worker-status` (optionally with `found: <out-of-scope discovery>`) → "**STOP here.** … The orchestrator will read your status, launch a review, and either approve your work or send feedback via `.worker-feedback`." `/orc:dispatch` uses `bd ready` for dependency-ready beads, spawns via `orc spawn` into tmux, then polls `/orc:check` every 60–90s.

### Beads + git hygiene

`.beads/` is "the single source of truth for work item status and dependencies". Runtime dirs `.worktrees/`, `.goals/`, `.worker-status`, `.beads/` are added to **`.git/info/exclude`** "automatically to remain invisible to git" — deliberately not `.gitignore`, so orchestration leaves no trace in the repo's tracked files.

---

## 3. snarktank/antfarm

Architecture: "YAML + SQLite + cron. That's it." TypeScript CLI, zero deps, Node ≥ 22; agents run as "isolated cron jobs in OpenClaw", each polls SQLite for a claimable step. "Each agent runs in a fresh session with clean context. Memory persists through git history and progress files." Three workflows ship: feature-dev, security-audit, bug-fix.

### workflow.yml (feature-dev, key structure)

Top: `id`, `name`, `version: 5`, `description`, `polling: {model: default, timeoutSeconds: 120}`. Then `agents:` — six entries (`planner, setup, developer, verifier, tester, reviewer`), each with `role` (analysis/coding/verification/testing), `workspace.baseDir`, per-agent identity `files:` (`AGENTS.md`, `SOUL.md`, `IDENTITY.md` — shared roles pull from `agents/shared/`), and optional `skills: [agent-browser]`.

Then `steps:` — each step is `{id, agent, input: |, expects: "STATUS: done", max_retries, on_fail}`. The load-bearing constructs, verbatim:

```yaml
  - id: implement
    agent: developer
    type: loop
    loop:
      over: stories
      completion: all_done
      fresh_session: true
      verify_each: true
      verify_step: verify
```

and the retry/escalation syntax:

```yaml
    expects: "STATUS: done"
    on_fail:
      retry_step: implement
      max_retries: 2
      on_exhausted:
        escalate_to: human
```

Steps pass context via `{{template}}` interpolation of prior-step output keys (`{{repo}}`, `{{branch}}`, `{{build_cmd}}`, `{{verify_feedback}}`, `{{current_story}}`, `{{progress}}`); agents must reply in parseable `KEY: value` form (`STATUS/REPO/BRANCH/STORIES_JSON`, `STATUS/CHANGES/TESTS`, `STATUS: retry` + `ISSUES:` list). Acceptance-criteria rules from the planner prompt: "Each story must fit in one developer session (one context window)… Every acceptance criterion must be mechanically verifiable… Always include 'Typecheck passes' as the last criterion in every story… Every story MUST include test criteria."

### Roles

**Planner**: explores codebase, ≤20 stories ordered schema→backend→frontend→integration; "If you cannot describe the change in 2-3 sentences, it is too big." **Verifier** (a quality gate, notably close to loki's evidence stance): "Inspect the actual diff… This is your source of truth, not the claimed changes from previous agents"; reject on empty diff; hard security checks first (reject if `.env`, `*.key`, `*.pem`, `*.secret`, `credentials.*` in diff, or missing `.gitignore` — "a security failure is always a rejection"); "Don't fix the code yourself — send it back." **Developer**: implement one story, write tests, commit `feat: {id} - {title}`, rewrite `progress-{{run_id}}.txt` (+ Codebase Patterns section). **Tester**: integration/E2E only. **Reviewer**: posts real `gh pr review --approve|--request-changes`, plus a visual design pass when `{{has_frontend_changes}}`.

---

## 4. asklokesh/loki-mode

### The eight blocking gates (`skills/quality-gates.md`)

All wired into `autonomy/run.sh`; "8 BLOCKING default-on gates… 3 ADVISORY default-on… 1 OPT-IN":

1. **Static Analysis** — linters + type-checker on the diff; blocks via severity ladder.
2. **Test Suite** — project runner pass/fail, "red blocks"; explicitly does NOT measure coverage.
3. **Blind Code Review** — 3 blind reviewers (agents, parallel); "Critical/High block, Medium/Low advisory."
4. **Anti-Sycophancy Devil's Advocate** — an agent re-review triggered **only on a unanimous PASS**; its Crit/High findings block.
5. **Mock Integrity Detector** — script: "tautological assertions, internal-mock ratio, tests that do not import source"; HIGH blocks.
6. **Test Mutation Detector** — script over recent commits: "assertion-value churn alongside implementation changes (test-fitting), low assertion density"; HIGH blocks.
7. **Documentation Coverage** — README presence, docs freshness within 10 commits, API docs for exports.
8. **Magic Modules Debate** — agent spec-vs-implementation debate; BLOCK-severity findings block.

So: gates 1,2,5,6,7 are script-driven; 3,4,8 are agent-driven; severity-based blocking (Crit/High block; Med/Low → TODO) is the shared policy. Advisory tier gates surface findings into the next iteration's prompt; coverage is opt-in because "it doubles test runtime."

### Devil's Advocate prompt (verbatim)

> "You are a Devil's Advocate reviewer. Independent reviewers may all approve this change. Unanimous approval is a red flag for insufficient scrutiny. Your SOLE job is to find a Critical or High severity issue they missed.
> Be adversarial and concrete. Hunt for: security holes, data loss, race conditions, broken error handling, silent failures, off-by-one and boundary bugs, resource leaks, injection, and logic that does not match intent. Do not rubber-stamp. If after genuine effort you find no Critical/High issue, say so honestly and do not invent one. …
> Output format (STRICT - follow exactly): VERDICT: PASS or FAIL / FINDINGS: - [severity] description (file:line) … Output VERDICT: FAIL only if you found a real Critical or High issue."

### Evidence Receipt (`.loki/proofs/<run_id>/proof.json`)

Top level: `schema_version, run_id, generated_at, loki_version, started_at, wall_clock_sec, spec, provider, iterations, files_changed, diffs, council, quality_gates, cost, deployment, tree_sha256, effort_estimate` plus the evidence triple:

- **`facts`** ("deterministic, re-derivable, NON-LLM"): `git: {base_sha, head_sha, diff, diff_sha256, tree_sha256, tree_manifest_version}`; `execution`; `build` and `tests` (command + exit code); `quality_gates: [{name, status, provenance}]`; `security`; functional axes (`{state: proven|gap|not_checked, reason}`); `healthcheck`; `cost`; `meta`.
- **`assessments`**: `{"_note": "AI judgment, not deterministic proof", "council": …, "completion_claim": {claimed, evidence_gate_verdict}}` — "a green council verdict is an opinion… it never contributes to the deterministic headline."
- **`honesty`**: `{headline, degraded: [{item, status, reason, post_headline?}], evidence_gate}` — degraded is "the explicit honesty ledger: a reader sees exactly what was NOT verified rather than inferring it from silence"; operator-disabled gates are appended as `status: "disabled"`.

Headline computed **only from facts**: `VERIFIED` (real test command exited 0, diff non-empty, nothing skipped) / `VERIFIED WITH GAPS` (each gap listed by name) / `NOT VERIFIED`. `loki proof verify` does a tamper check (re-hash) and a drift check (re-derive diff, compare counts + `diff_sha256`); exit 0 clean, 1 tamper/drift. Unsigned receipts are "defense-in-depth, not non-forgeability"; GPG signing closes it.

### RARV + spec-lock

RARV-C: "**Reason** (read state) — **Act** (execute, commit) — **Reflect** (update context) — **Verify** (run tests, check spec)" + Close. The verified-completion evidence gate "refuses any 'done' claim on an empty git diff against the run-start commit, blocks completion when tests run red," and caps at `MAX_ITERATIONS`. Spec: `loki spec` locks with a deterministic hash into `spec.lock`; divergence lands in `drift-report.json` and as a `SPEC_DRIFT` finding in `loki verify` (exit codes 0 VERIFIED / 1 CONCERNS / 2 BLOCKED); OpenAPI contracts get per-operation hashes.

---

## 5. ikamensh/kodo

### team.json (full schema)

Lookup: `{project}/.kodo/team.json` → `~/.kodo/teams/{name}.json` → built-ins. Shape: `{"name": str, "agents": {key: {...}}}`. Per-agent fields: **required** `backend` (`claude|cursor|codex|gemini-cli`) and `model`; **optional with defaults** `max_turns: 15`, `timeout_s: null`, `chrome: false`, `description: ""`, `system_prompt: null`, `fallback_model: null`. README example (excerpt):

```json
{ "name": "saga-with-designer",
  "agents": {
    "worker_smart": {"backend": "claude", "model": "opus",
      "description": "Deep-thinking worker for complex tasks."},
    "architect": {"backend": "claude", "model": "opus",
      "description": "Reviews architecture, validates direction.",
      "max_turns": 10, "timeout_s": 600},
    "designer": {"backend": "claude", "model": "opus",
      "system_prompt": "You are a UX/UI design advisor. ... Say 'ALL CHECKS PASS' if clean.",
      "max_turns": 10, "fallback_model": "sonnet"} } }
```

Any agent added to the dict is visible to the orchestrator as a delegation tool.

### Orchestrator protocol (verbatim system prompt)

> "You are an orchestrator. Get the user's desired outcome. Your agents have full codebase access and are expert coders. Every implementation detail you specify risks making the result worse. Tell them WHAT, never HOW. 1. Define desired outcome… 2. Delegate as small, verifiable goals. 3. Verify results match intent. Commit good work, revert bad iterations. The team shares .kodo/architecture.md — the architect updates it, workers read it. You decide: priorities, scope, what 'done' looks like, when to revert. Agents decide: code structure, libraries, patterns, file organization."

Plans are `GoalPlan{stages: [GoalStage{name, description, acceptance_criteria, browser_testing, parallel_group}]}`; same-`parallel_group` stages run concurrently in worktrees. When the orchestrator calls `done(summary, success)`, the rejection loop runs: send a verification prompt to each **tester** ("Verify this works end-to-end. Report ONLY issues found. If everything works, say 'ALL CHECKS PASS'.") and each **reviewer/architect**. Acceptance = `ALL CHECKS PASS` or a minor-signal in the report. First `done()` attempt resets verifier sessions "for a clean baseline; subsequent calls reuse the session so verifiers have persistent context." If no dedicated verifiers exist, a worker is drafted **in a fresh session** as fallback verifier. Any failing report returns: `"DONE REJECTED (attempt N) — verification found issues that must be fixed: … Fix these issues and try calling done again."` — the README run shows 9 consecutive rejection rounds before acceptance.

### Run archive

`~/.kodo/runs/<ts>/run.jsonl` (+ `goal.md`): every event is one JSON line `{"ts": <UTC ISO>, "event": <name>, ...}`. Event types: `run_init, cli_args, run_start, stage_start, stage_end, cycle_end (finished: bool), run_end, session_query_end, orchestrator_tool_call, agent_run_end, done_verification, orchestrator_done_rejected, done_verification_error`. Resume and the HTML viewer replay this file.

---

## SYNTHESIS: proposed open-seed formats

The five projects triangulate cleanly: **loop-engineering** gives the file-format-as-governance vocabulary; **orc** shows per-phase natural-language instruction keys + bounded `max_rounds`; **antfarm** shows `expects/on_fail/retry_step/escalate_to` as declarative loop wiring; **loki** shows facts-vs-assessments evidence; **kodo** shows the verifier rejection protocol capped by attempts.

### Proposed `guardrails.yaml` for open-seed

```yaml
version: 1

autonomy:
  default_tier: L1
  tiers:
    L1:   # report-only (loop-engineering L1)
      may: [read, comment, open-issue, update-memory-files]
      may_not: [edit-code, merge, spawn-agents]
      verifier: none-required
    L2:   # assisted fix (loop-engineering L2 + kodo verify_done)
      may: [edit-in-worktree, open-pr, spawn-agents]
      may_not: [merge, touch-denylist]
      requires: {worktree: true, verifier: independent, attempt_cap: 3}
    L3:   # unattended (gate-checked auto-merge only)
      may: [auto-merge]
      requires: {allowlist_match: all-paths, verifier: independent, ci_green: true}

paths:
  protected:        # gate.yaml denylist, adapted
    - ".env"
    - ".env.*"
    - "**/secrets/**"
    - "**/*_key*"
    - "**/*_secret*"
    - "**/migrations/**"
    - "auth/**"
    - "payments/**"
    - ".github/workflows/**"     # open-seed addition: loops may not edit their own CI
    - "guardrails.yaml"          # loki spec-lock analog: humans only edit governance
  auto_merge_allowlist: ["docs/**", "**/*.md"]
  max_files_per_change: 10

budgets:            # loop-budget.md, made machine-readable
  per_loop:
    default: {max_runs_per_day: 4, max_tokens_per_day: 200000, max_subagents_per_run: 2}
  on_exceed: [pause-schedulers, append-run-log, open-issue]
  kill_switch: {label: loop-pause-all, resume: human-clears-STATE}

tools:              # anti-pattern #6: least privilege per tier
  L1: {github: [read, comment, label]}
  L2: {github: [read, comment, label, push-branch, open-pr]}
  L3: {github: [merge]}

verify:             # kodo/antfarm: commands are facts, agents are assessments
  commands: [ "npm run typecheck", "npm test" ]     # per-project
  independent_reviewer: true      # never same session as implementer
  reviewer_stance: reject         # loop-engineering: "Verifier default stance: REJECT"
  devils_advocate_on_unanimous_pass: false          # loki gate 4, opt-in

escalation:
  attempt_cap: 3
  on_exhausted: {action: escalate_to_human, record: STATE.md#high-priority, alert_after_hours: 24}
```

CI validation follows loop-engineering exactly: a `validate-orchestration.yml` running a script that checks guardrails.yaml against a JSON Schema (ajv), verifies memory-file templates exist, and enforces cross-file consistency, plus a loop-sync-style drift score.

### Proposed evidence-receipt schema (task closure)

Adopt loki's facts/assessments/honesty split, simplified, one file per closed task at `.seed/receipts/<task_id>.json`:

```json
{
  "schema_version": 1,
  "task_id": "…", "run_id": "…", "tier": "L2",
  "facts": {
    "git": {"base_sha": "…", "head_sha": "…", "files_changed": 3, "diff_sha256": "…"},
    "commands": [{"cmd": "npm test", "exit_code": 0, "at": "…"}],
    "protected_paths_touched": [],
    "budget": {"tokens_estimate": 52000, "attempts": 1}
  },
  "assessments": {
    "_note": "AI judgment, not deterministic proof",
    "reviewer_verdict": "approve", "reviewer_report_sha256": "…"
  },
  "honesty": {
    "headline": "VERIFIED | VERIFIED_WITH_GAPS | NOT_VERIFIED",
    "gaps": [{"item": "e2e", "status": "not_run", "reason": "…"}]
  }
}
```

Headline computed only from `facts` (non-empty diff + all commands exit 0 + no protected paths → VERIFIED); every skipped check must appear in `gaps` — silence never reads as pass. This doubles as the run-log entry (superset of loop-engineering's run-log JSON).

### Corrections to earlier survey findings

1. loop-engineering has **no `.budget.json`/`run-log.json`** — both are markdown with an embedded JSON entry schema; only `gate.yaml` and `registry.yaml` are machine-first. Implication: open-seed's choice to make guardrails fully YAML is *ahead* of the pattern book, matching its gate.yaml trajectory.
2. L1/L2/L3 are **not defined in one verbatim block** anywhere — they're an operating convention scattered across LOOP.md/safety.md/checklist. open-seed codifying tiers in guardrails.yaml is novel relative to its closest cousin.
3. orc's runtime exclusion uses `.git/info/exclude`, not `.gitignore` — worth copying for open-seed runtime dirs.
4. antfarm's "acceptance criteria" live in planner-generated stories (data), not in the workflow YAML (only `expects:` string-match lives there) — mechanical verifiability is enforced by prompt convention plus the verifier agent, not schema.
5. kodo has no gate files at all — governance is purely protocol (rejection loop), the opposite pole from loki; open-seed's design should state explicitly that it combines file-governance (loop-engineering/loki) with protocol-governance (kodo/orc).
