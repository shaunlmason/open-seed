# Research: Autonomous Task Runners (issue/CI-driven agents)

> Category survey of [awesome-agent-orchestrators](https://github.com/andyrewlee/awesome-agent-orchestrators),
> researched 2026-08-21 for the open-seed design study. Highly relevant because CI workflows
> ARE checked-in repo content. All 17 repos resolved; none 404'd.

## Project Profiles

### 1. github/gh-aw: GitHub Agentic Workflows (deep dive)

**What/maturity:** GitHub Next project (~5k stars, actively evolving). CLI is a `gh` extension: commands `gh aw compile | add | run | logs | audit | mcp`.

**Core model:** A workflow is a Markdown file in `.github/workflows/*.md`: YAML frontmatter for configuration, Markdown body as the agent's natural-language instructions. `gh aw compile` validates it and emits a deterministic `.lock.yml` GitHub Actions workflow that is also committed. Both source and compiled artifact are repo content: the purest "everything in the repo" design in the category.

**Frontmatter surface (selected):** `on:` (standard Actions triggers plus extensions: command/mention triggers, `reaction:`, `stop-after:`, `manual-approval:`, `roles:`/`skip-bots:`, `forks:`); `engine:` (copilot | claude | codex | gemini | pi); `permissions:` (read scopes for the agent job); `tools:` (`github:` toolsets, `edit:`, `bash:` allowlists like `bash: ["gh issue comment"]`, `web-fetch:`, `playwright:`, `mcp:` servers); `network:`; `safe-outputs:`; `imports: [shared/common-tools.md]` for shared fragments; budget keys `max-turns` (default 500), `max-ai-credits`, `max-daily-ai-credits`, `user-rate-limit`.

**Safety model (the crown jewel):** Compiled workflows decompose into staged jobs: pre-activation (role check) → activation (sanitization, lock-file validation) → **agent job with read-only permissions** → detection job (a separate security-focused AI scans buffered artifacts for secret leakage/malicious patterns) → **safe-output jobs** that perform scoped writes only after detection passes. "Agent execution never has direct write access to external state."

- **Safe-outputs:** the agent emits structured JSON requests; separate permission-scoped jobs apply them. ~40 types: `create-issue`, `add-comment`, `create-pull-request`, `push-to-pull-request-branch`, `create-code-scanning-alert` (SARIF), `dispatch-workflow`, `assign-to-agent`, plus system types. Each has a conservative `max:` (usually 1), plus options like `title-prefix: "[ai] "`, forced `labels:`, `deduplicate-by-title`, `expires: 7d`, cross-repo targeting with dedicated tokens, `staged` preview mode, and hidden workflow-id markers for searchable provenance.
- **Sanitization:** untrusted input is rewritten *before* the agent sees it: `@user` → backticked (no notifications), `fixes #123` → backticked (no auto-linking), HTML/XML defanged, non-HTTPS URIs redacted, size caps. Outbound: `max-bot-mentions`, `allowed-domains`, `max-patch-size`.
- **Network:** Agent Workflow Firewall containerizes the agent and forces HTTP(S) through a proxy; frontmatter `network: { firewall: true, allowed: [defaults, python, node, api.example.com] }`: ecosystem bundles plus explicit domains.
- **Secrets:** temp dirs scanned and redacted before artifact upload; audit via `gh aw audit`.

**Steal:** the compile step (human-writable markdown → verified YAML), read-only agent + safe-output job split, sanitization pipeline, `imports:` for shared tool fragments, budget keys, ecosystem-bundle egress allowlists.

### 2. anthropics/claude-code-action (deep dive)

**What/maturity:** Anthropic's official Action (~8.7k stars, v1.0, Marketplace-listed). Runs Claude Code on *your* runner; API via Anthropic, Bedrock, Vertex, or Foundry.

**Triggers/modes:** auto-detects context: interactive mode on `@claude` mentions/issue assignment/PR comments; automation mode when an explicit `prompt:` input is set (schedules, `workflow_dispatch`, `pull_request` events). Inputs: `prompt`, `override_prompt`, `custom_instructions`, `claude_args` (raw CLI passthrough: `--allowedTools`, `--max-turns`, `--model`, `--mcp-config`), `settings` (JSON incl. permissions allow/deny), `use_sticky_comment`, `track_progress` (live checkbox progress).

**Guardrails:** Trigger gating: only users with **write access** can trigger; bots blocked by default (`allowed_bots` opt-in); `allowed_non_write_users` flagged "extreme caution." Tool defaults: file ops, comments, basic GitHub ops only; no arbitrary Bash by default; grant granularly: `--allowedTools "Bash(npm install),Bash(npm run test)"`. Output: PRs are *not* auto-created by default; Claude pushes a branch and posts a PR-creation link ("human oversight before any code is proposed"). Tokens: short-lived GitHub App token scoped to the single repo. Commit signing optional. Injection defense: strips HTML comments, invisible characters, image alt text; for `pull_request_target`, check out base ref at root and untrusted PR files in a subdirectory. Env-var scrubbing plus bubblewrap isolation on Linux.

**Steal:** write-access trigger gating, per-command Bash allowlisting syntax, branch-not-PR default, sticky progress comment, `CLAUDE.md` as checked-in agent context.

### 3. aeonfun/aeon

Autonomous agent framework running unattended **on GitHub Actions** (~677 stars, MIT, very active). Everything is checked in: `aeon.yml` (model, schedules, chains, per-skill `var`), `.github/workflows/scheduler.yml` (cron tick every 5 min; skills dispatch only on match), `skills/*/SKILL.md` (frontmatter: `name, category, requires: [API_KEY,...]`, `mode: read-only|write`, `mcp:`), `STRATEGY.md` (north-star imported into every run), `soul/SOUL.md` (voice), `memory/` (state committed to git: `memory/cron-state.json`, or an append-only closed GitHub Issue as state backend), `.mcp.json`. Six harnesses behind one `run-harness` contract. Guardrails: `mode: read-only` is sandbox-enforced (strips Write/Edit/git); declared `capabilities:` taxonomy; per-skill circuit breaker (3 failures → half-open probes every 6h); dry-run gate with synthetic secrets before self-authored skills PR themselves in; optional ALLOW/BLOCK endpoint that **fails closed**; append-only JSONL audit artifact (action names + secret *names*, never values); token usage logged to `memory/token-usage.csv`. Distinctive: self-healing `heartbeat → skill-health → skill-repair → self-improve` loop; skill chaining with scored conditional routing.

### 4. ColeMurray/background-agents

Open-source clone of Ramp's Inspect (~2.7k stars). Hosted control plane (Cloudflare Durable Objects) + isolated sandboxes. Triggers: web UI, Slack, Linear, GitHub bot, authenticated webhooks with JSONPath condition filters, cron. Repo content: `.openinspect/setup.sh` and `.openinspect/start.sh` with timeout env vars. Explicitly single-tenant/trusted-users-only. Distinctive: child sessions, multi-repo sessions, per-prompt git identity for commit attribution, sandbox pre-warming.

### 5. paradigmxyz/centaur

Paradigm's self-hosted platform (~1.2k stars). Slack-thread-native plus REST; Postgres-durable, replayable sessions; per-conversation **Kubernetes sandboxes** with default-deny NetworkPolicy. Standout guardrail: **iron-proxy credential boundary**: agents see placeholder strings; real secrets injected only on approved outbound routes; tools are CLI shims so raw API keys never enter agent memory; full outbound audit trail. BYO harness.

### 6. openai/codex-action

Official Codex Action (~1.2k stars, stable v1). Inputs: `prompt`/`prompt-file` (checked-in prompt content), `output-schema` (structured output validation), `model`, `effort`. Safety: `safety-strategy:`: `drop-sudo` (default: irreversibly revokes sudo, removes Docker socket, capability drop), `unprivileged-user`, `read-only`, `unsafe`. Beta `permission-profile:` input: filesystem and network access as policy-as-code. Approval gating via `allow-users`/`allow-bots`; default requires write access. Steal: step-level privilege dropping, `prompt-file`, output schemas.

### 7. junhoyeo/contrabass

Go implementation of OpenAI's Symphony spec (~215 stars; terminal-first TUI). Polls Linear, GitHub Issues, **or a built-in Internal Board on the local filesystem** (`.contrabass/board/`: tasks as files in repo, no external service). Workflows are checked-in **`WORKFLOW.md` files with YAML frontmatter**. Execution in git worktrees; guardrails: `BlockedBy` dependency gating, branch-advance verification (did the agent actually commit?), stall detection, deterministic retry backoff. Five-stage run lifecycle (Exploration → Editing → Testing → Reviewing → Wrapping).

### 8. cyrusagents/cyrus

Background agent that watches **issues assigned to it / mentions** across Linear, GitHub, GitLab, Slack (~779 stars). Runs Claude Code / Codex / Cursor / Gemini sessions in per-issue git worktrees, streams results back to the tracker with interactive UI. Repo carries `CLAUDE.md`/`AGENTS.md`. Thin on checked-in guardrails.

### 9. owainlewis/factory

Go control plane + workers for "repeatable software-engineering Runs" (~193 stars, developer preview). Operators save prompts as versioned **Tasks** with schedules; workers poll outbound-only. Nothing checked into target repos: a pattern to *avoid* for open-seed, but the "prompt as versioned, schedulable Task" concept maps to committed markdown task specs.

### 10. tim-smart/lalph

CLI orchestrator (~130 stars, very active) that "pulls work from an issue source": GitHub Issues, Linear, pluggable. Plan → spec → "PRD tasks" pipeline; git worktrees for concurrency. Agent presets pick the CLI harness per task, with **label-based agent routing**. Guardrails: plan-mode confirmation by default, per-issue `autoMerge` frontmatter for the PR flow.

### 11. multica-ai/multica

"Assign work to AI agents the way you'd assign it to a teammate" (very large, fast-moving). Web/desktop/iOS workspace; local daemon spawns agent CLIs next to your code. Triggers: issue assignment, mention, chat, cron "autopilots," CLI/API. **23 harnesses**. Guardrails: review gates (no direct merges to main), roles, per-agent permission scopes, execution logs as audit trail. Repo content: skills as `.agents/skills/...` playbooks with a `skills-lock.json` pinning pattern. Distinctive: "squads" mixing humans and agents with a leader that routes work.

### 12. langchain-ai/open-swe

**Pivoted** from the 2025 planner/programmer/reviewer label-driven app to an ~10.6k-star **framework for building your org's internal coding agent** (LangGraph + Deep Agents). Triggers are webhook endpoints: Slack mention, Linear comments, GitHub mentions, dashboard. Per-thread persistent cloud sandboxes (langsmith | daytona | runloop | e2b | modal | local). Subagent fan-out; **message-queue middleware** injects mid-run human comments before the next model call. Repo-side contract: **`AGENTS.md` at repo root injected into the system prompt**. Validation is prompt-driven, no hard review gate.

### 13. OpenHands/OpenHands

Now "the self-hosted developer control center" (~84.7k stars). The classic resolver workflow (`fix-me` label) moved to OpenHands Cloud. The durable repo-side surface is the **`.openhands/` directory**: `setup.sh` (runs at session start), microagents/skills, and **`hooks.json`**: six lifecycle hooks (`PreToolUse`, `PostToolUse`, `UserPromptSubmit`, `Stop`, `SessionStart`, `SessionEnd`), blocking via exit code 2 or JSON deny decisions. **Stop hooks as quality gates** ("block completion until formatting, linting, tests pass") are directly transferable. Also: ACP for harness interop, org budget limits, confirmation policy + security analyzer in the SDK.

### 14. aws-samples/remote-swe-agents

AWS sample (~241 stars): serverless control plane (Lambda, DynamoDB) that provisions a dedicated EC2 instance per session; Bedrock models; triggers via Slack, web UI, REST API, and a published GitHub Action. Guardrails are IAM-shaped: minimal default worker policy, explicit additional-policy opt-in. Steal: "guardrails as cloud IAM policy" framing and cost-per-session accounting.

### 15. google-github-actions/run-gemini-cli

Google's official Action (~2.1k stars). Ships **four ready-made workflows in `examples/workflows`** with `gemini-dispatch.yml` as a central router that fans `@gemini-cli /review`, `/triage`, and free-form requests out to issue-triage, PR-review, and assistant workflows: a checked-in dispatcher pattern. Repo content: workflows + `GEMINI.md` context file. WIF for keyless GCP auth; `/setup-github` self-install command.

### 16. sortie-ai/sortie

Single Go binary orchestrator (~130 stars but 1,719 commits, Apache-2.0, no telemetry). "Turn tracker tickets into autonomous agent sessions": polls GitHub/GitLab/Gitea Issues, Linear, Jira for issues matching workflow criteria (e.g. `label:agent-ready`). The contract is a checked-in **`WORKFLOW.md`** (YAML frontmatter: tracker, query filters, agent selection, `max_concurrent_agents`, prompt template) that sortie **hot-reloads without restart**. Isolated workspace per issue; stall detection, timeouts, retry/backoff, cost tracking; "orchestrator is the single authority for all scheduling decisions": the orchestrator, not agents, writes tracker state.

### 17. openai/symphony

"Low-key engineering preview." The repo is primarily a **SPEC.md**: you either prompt an agent to build Symphony from the spec or run the Elixir reference implementation. Monitors a Linear board, spawns isolated autonomous runs, and reframes the job as *managing work, not supervising agents*: each run must return **proof-of-work evidence**: CI status, PR review feedback, complexity analysis, walkthrough videos. Builds on OpenAI's "harness engineering" idea: the codebase must first be made agent-legible (fast deterministic checks, good docs) before autonomy scales. Spec-as-product and evidence-bundles are the exports here.

---

## Synthesis

### Recurring patterns

**Triggers converge on five verbs:** (1) *label* (`agent-ready`, `openhands`, historical `fix-me`); (2) *mention/command* (`@claude`, `@openswe`, `@gemini-cli /review`); (3) *assignment* (cyrus, multica: "assign like a teammate" is the emerging UX); (4) *schedule* (aeon's 5-min tick with match-gating, cron autopilots); (5) *webhook/event*. Mature systems support all five behind one dispatcher (run-gemini-cli's `gemini-dispatch.yml` router is the cleanest checked-in version).

**Where tasks live splits three ways:** external tracker (Linear dominant; sortie is the most tracker-agnostic), GitHub Issues (the only option that is *also* repo-adjacent and API-native to Actions), and **files in the repo** (contrabass's `.contrabass/board/`, aeon's `memory/`, lalph's spec directories). Several projects deliberately make tracker choice pluggable behind one interface: the task-source abstraction, not the tracker, is the design decision.

**Execution splits cleanly into "Actions-native" vs "external control plane."** Actions-native (gh-aw, claude-code-action, codex-action, run-gemini-cli, aeon) = zero infrastructure, everything reviewable in-repo, bounded by runner limits. External (OpenHands, open-swe, centaur, background-agents, remote-swe, factory, multica) = persistent sessions, better isolation options, but the orchestrator itself is not versioned with the code. Nearly everyone uses **git worktrees or per-task workspaces** for isolation regardless of tier.

**Permissions & safety: the field has converged on layered patterns:**
1. *Who may trigger:* write-access checks, role gates, bot allowlists, never trust the event payload's author.
2. *Read-only agent, mediated writes:* gh-aw's safe-outputs is the strongest form; claude-code-action's branch-not-PR default and codex-action's `read-only` strategy are weaker cousins. Centaur's iron-proxy applies the same idea to credentials.
3. *Input sanitization:* gh-aw and claude-code-action both treat issue/comment text as hostile.
4. *Tool allowlisting:* per-command Bash grants, MCP tool filters.
5. *Egress control:* gh-aw firewall with ecosystem bundles; centaur default-deny NetworkPolicy.
6. *Budgets/limits:* gh-aw `max-turns`/`max-ai-credits`/`user-rate-limit`; aeon circuit breakers + token CSV; sortie cost tracking.
7. *Completion gates:* OpenHands Stop hooks; symphony evidence bundles; contrabass branch-advance verification.

**Agent-context files are now a de facto standard:** `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, aeon's `STRATEGY.md`/`SOUL.md`. Same for environment bootstrap scripts: `.openhands/setup.sh`, `.openinspect/setup.sh`/`start.sh`: a two-script setup/start convention with timeouts appears independently twice.

### What a template repo should ship

1. **Workflow files**: a small set of `.github/workflows/`: an agent dispatcher (mention/label router), a scheduled-maintenance agent workflow, and a PR-review workflow; ideally authored as gh-aw markdown with committed `.lock.yml`, or as plain claude-code-action YAML for fewer moving parts.
2. **Labels + issue templates as the task API.** A checked-in label taxonomy (`agent-ready`, `agent:in-progress`, `agent:blocked`, `needs-human`) and issue forms whose fields (acceptance criteria, blocked-by, autoMerge) are machine-parseable.
3. **Agent instruction files:** `AGENTS.md` (harness-neutral) with `CLAUDE.md`/`GEMINI.md` referencing it; optionally a `STRATEGY.md`-style tie-breaker doc.
4. **Bootstrap + gates:** `setup.sh` (env provisioning, timeout-bounded) and hook config for completion gating (Stop hook runs lint/test before the agent may finish): mirrored in Claude Code's own hooks in `.claude/settings.json`.
5. **Guardrail config as data:** permissions blocks pinned minimal in every workflow; tool allowlists; an egress allowlist file; budget knobs; a `.mcp.json`; secret *names* documented, never values.
6. **Provenance conventions:** title prefixes (`[ai]`), forced labels on bot-created artifacts, hidden workflow-id markers in bodies, and sticky progress comments.

### Five most transferable ideas

1. **Read-only agent + safe-output split (gh-aw).** Even without gh-aw, a template can implement it with two plain Actions jobs: agent job with `permissions: contents: read` writing a JSON artifact; a second job validating it against a checked-in schema and performing writes. This single pattern neutralizes most prompt-injection blast radius.
2. **Markdown-with-frontmatter as the universal automation unit.** gh-aw workflows, aeon SKILL.md, sortie/contrabass WORKFLOW.md, and Claude skills independently converged on it: human-readable prompt body + machine-readable YAML header, optionally *compiled* to executable form with the compiled artifact committed and drift-checked in CI.
3. **Repo-root agent contract files** (`AGENTS.md` + `setup.sh` + blocking lifecycle hooks). Cheapest, most portable guardrail: every harness in this survey reads some form of them, and Stop-hook quality gates convert convention into enforcement.
4. **Labels-as-state-machine with the orchestrator as single writer** (sortie/OpenHands/lalph). Tickets carry trigger (`agent-ready`), state, and routing (label→agent preset); the orchestrator, never the agent, reconciles tracker state: making runs idempotent, auditable, and resumable.
5. **Declared budgets, provenance, and evidence.** Max-turns/credit caps and circuit breakers checked in as config; every bot artifact self-identifying (prefix, label, hidden marker); and symphony's proof-of-work bundle (CI status + diff stats + walkthrough) as the required body of every agent PR: shifting review from "watch the agent" to "audit the evidence."

**Bottom line for open-seed:** the Actions-native tier proves everything you need can be checked in: triggers (workflow YAML), tasks (issues + labels + templates), instructions (markdown contract files), guardrails (permissions blocks, tool/egress allowlists, hooks, budgets), and even the orchestration language itself (compiled markdown). Treat gh-aw as the safety-architecture reference, claude-code-action as the pragmatic baseline runner, and aeon/sortie/OpenHands as the sources for repo-file conventions.
