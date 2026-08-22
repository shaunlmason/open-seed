# Research: Agent Infrastructure & Primitives

> Category survey of [awesome-agent-orchestrators](https://github.com/andyrewlee/awesome-agent-orchestrators),
> researched 2026-08-21 for the open-seed design study. Focus: what can live *inside* a repo
> vs. what requires an external app/daemon. This category is highly relevant because several
> of these ARE checked-in-content systems (runbooks, skills, workflows).

All 24 repos located and researched. Deep-dive projects first, then the rest grouped by architecture; synthesis at the end.

---

## Deep-dive: orchestration-as-repo-content

### KnoxOps/agent-runbook
**(a)** A compiler that turns contract-based YAML runbooks into executable SKILL.md files for Claude Code/Codex. **(b)** Core primitive: the **contract-validated step** — each step declares inputs/outputs via JSON Schema, and the compiler verifies "contract closure" (all dependencies exist, schemas resolve, DAG is acyclic) at *build time*, before any agent runs. **(c)** Checked in: `runbook.yaml`, `schemas/*.schema.json`, `prompts/*.agent.md`, `scripts/*.py`; build emits `SKILL.md` + checkpoint scripts. Eight step types: `inline`, `agent`, `script`, `parallel` (with `max_instances`, `item_key`), `branch`, `loop` (with `goal`, `max_iterations`, `body`), `checkpoint`, `quality_check` (`blocking: true`, `rules: [...]`). **(d)** Portability from targeting the SKILL.md standard plus **file-based state passing** (steps communicate through JSON files on disk, not context windows). **(e)** Guardrails: build-time contract validation, blocking quality gates dispatched to a `@supervisor`, cycle rejection, checkpoint/resume after crashes. **(f)** Very early: 17 stars, Apache-2.0. **(g)** Steal: compile-don't-interpret (validate orchestration before an LLM ever sees it); "files over context" as the state-passing discipline.

### byronxlg/skillfold
**(a)** A declarative dependency manager for agent skills — "npm for skills." **(b)** Core primitive: the **manifest + lockfile pair**. `skillfold.yaml` declares intent; `skillfold.lock` records exact resolutions (commit SHA or version + sha256 content hash) so every machine gets byte-identical skills. **(c)** Checked in: both files. Manifest syntax: `skills: {commit-helper: ./skills/commit-helper, frontend-design: github:anthropics/skills/skills/frontend-design, planning: npm:skillfold/planning}` — three source types (local dir, GitHub subpath, npm). Install targets configured via `targets: [claude, codex]`: skills land in `.claude/skills` and `.agents/skills`; rules sync into `.claude/rules/` and `AGENTS.md`. CLI mirrors npm: `init/add/install/install --frozen/update/check/list/search`. **(d)** Claude Code + Codex today; "supporting another tool is just another install location" — skills are harness-neutral Markdown, only the install path differs. **(e)** Only lockfile-named directories are managed (hand-authored skills never touched); `check` verifies manifest/lock/installed agreement offline; `--frozen` fails on any drift (npm-ci semantics); composition cycles rejected at parse. **(f)** Early (12 stars) but well-built: MIT, CI, full docs. **(g)** Steal: the entire manifest/lockfile model, hash-verified frozen installs for CI, one manifest fanning out to multiple harness directories, and the "manage only what you own" safety rule.

### shinpr/sub-agents-skills
**(a)** A portable Agent Skill whose Python script reads Markdown agent definitions and routes each to one of 8+ CLI backends (Codex, Claude Code, Cursor CLI, GLM, Kimi, Grok, Gemini CLI, OpenCode). **(b)** Core primitive: the **backend-annotated Markdown agent** — plain text file where YAML frontmatter picks the executor. **(c)** Checked in: `.agents/*.md` files (filename = agent id). Frontmatter: `run-agent` (backend), `model`, `effort`, `permission` (`read-only` | `safe-edit` default | `yolo`); body sections `## Task` and `## Done When`. Follows the Agent Skills open standard (~30+ compatible tools). Installed as a Claude Code/Codex plugin or via curl for Cursor/Gemini. **(d)** Best-in-class routing: frontmatter declares the backend, env vars carry credentials (never argv), CLIs pointed at provider endpoints via base-URL overrides. Enables mixed fleets: "run code-reviewer and alternate-reviewer in parallel, then send agreed changes to kimi-implementer." **(e)** Three-level permission sandbox; no stdin to children (kills interactive-prompt deadlocks); fresh context per invocation; explicit warning that agent definitions are system prompts — "only use definitions you've written or trust." **(f)** 80 stars, MIT, CI + tests, active. **(g)** Steal: the `.agents/` directory of single-responsibility Markdown agents with a `permission` field; treating checked-in agent definitions as a trust/security boundary; per-agent backend selection so one repo mixes providers.

### crewplaneai/crewplane
**(a)** A provider-neutral, CLI-first control plane executing multi-stage workflows defined as Markdown DAGs. **(b)** Core primitive: the **Markdown-defined DAG** — YAML frontmatter declares nodes (`id`, `mode: sequential|parallel`, `findings`, `providers: ["mock"]`, `schema_version`), the Markdown body carries per-node prompts/context. Reviewed in PRs like any code. **(c)** Checked in: `.crewplane/workflows/*.md` and `.crewplane/config.yml` (provider wiring); run artifacts are treated as build outputs, gitignore-able. CLI: `init`, `validate` (preflight DAG check), `run`, `onboarding`. **(d)** Portability by **CLI invocation, not SDK wrapping**: each stage's `providers` list names Claude Code, Codex, Gemini, Copilot, "or any CLI"; swapping providers is a config edit. A deterministic `mock` provider allows credential-free testing. **(e)** "Runs exactly the plan you wrote" — no autonomous re-planning; preflight validation; resumption from validated stage boundaries; everything on disk for audit. **(f)** Early: 33 stars, Apache-2.0. **(g)** Steal: mock provider for CI-testable workflows; the `validate` preflight command; stage-boundary resumability; frontmatter-DAG-plus-prose-body as the workflow file shape.

### farol-team/gnap (Git-Native Agent Protocol)
**(a)** An RFC-draft protocol coordinating AI agents and humans through a shared git repo — "No servers. No databases. No vendor lock-in. Just git." **(b)** Core primitive: the **heartbeat loop over four JSON entities** in `.gnap/`: `agents.json` (roster: id, role, `type: ai|human`, status, capabilities, `heartbeat_sec`, `reports_to`), `tasks/{id}.json` (state machine: backlog → ready → in_progress → review → done, plus blocked/cancelled), `runs/{task-id}-{attempt}.json` (per-attempt tokens, `cost_usd`, commits, artifacts), `messages/{id}.json` (`to: ["*"]` broadcast, types directive/status/request/info/alert, threads). Loop: pull → check status → read tasks → read messages → work → push → sleep. **(c)** Everything is checked-in repo content plus a `.gnap/version` integer; commit convention `<agent-id>: <action> [details]` (e.g. `carl: done FA-1 — Stripe test mode live`). **(d)** Maximal portability: "if it can git push, it can participate" — humans included. **(e)** Eventual consistency bounded by heartbeat interval; git merge + rebase-and-retry on collisions; git history *is* the audit log. **(f)** 81 stars; runs a real 4-agent team over 50+ tasks at Farol Labs; still a draft. **(g)** Steal: the four-entity split (agent/task/run/message) with runs separate from tasks so retries and cost accrue transparently; humans as first-class `type: human` agents; protocol/application layering; the commit-message convention as machine-parseable activity feed.

### phuryn/swarm-protocol
**(a)** A headless MCP server for "agent-first teams. No UI. No sprints. No Jira. Just state sync" — coordinates multiple developers each running agents on one codebase. **(b)** Core primitives (19 MCP tools): **Intent** (draft → open → claimed → done), **Claim** (task reservation *declaring which files it touches*, kept alive by 10–15 min heartbeats), **Signal** (events that cascade state changes — `complete_claim` auto-unblocks dependents), **Context Package** (`get_context` returns intent + dependencies + active claims + team conventions in one call). `check_conflicts` queries active claims' file lists *before* an agent edits. **(c)** Server is external (Node + Postgres), but the checked-in half is explicit: copy `claude-md/COORDINATION.md` into the repo's `CLAUDE.md` — the README calls this pattern "as important as the server itself." **(d)** Anything speaking MCP. **(e)** Conflicts are **advisory, not enforced** — no locking. **(f)** Alpha, 53 stars. **(g)** Steal: file-set declaration at claim time + advisory pre-edit conflict check; the CLAUDE.md snippet as the *entire* client integration; the context-package idea (one call assembles everything a fresh agent needs).

### zippoxer/subtask
**(a)** A Claude Skill + Go CLI letting Claude Code delegate tasks to parallel subagents, each in its own **git worktree**. **(b)** Core primitive: task = worktree + persisted conversation; Claude can interrupt/converse with running subagents. **(c)** Commands: `subtask draft fix/auth-bug`, `list` (draft/working/replied statuses). **(d)** Supports Claude and Codex subagents. **(e)** Review-before-merge flow: subagent finishes → Claude reviews diff → user decides merge or revise. **(f)** Explicitly "early development." **(g)** Steal: worktree-per-task as the parallelism/isolation primitive; parent-reviews-child-diff-before-merge; conversation persistence per task folder.

---

## Remaining projects

### agentlas-ai/Agentlas-OS
Local-first "agent OS": specialist agents live in a hub, a **temporary orchestrator is spun up per task** and dissolved after. Core primitive: the **package contract** — an inspectable bundle of routing card, memory map, typed I/O contracts, permissions, verification manifest. Heavily checked-in: `.agentlas/routing-card.json` (triggers/anti-triggers/risk levels), `.agentlas/memory-map.json`, `contracts/intake.schema.json` + `output.schema.json`, `AGENTS.md`. Portability via thin adapters (`.claude/`, `codex/`, `.gemini/`) — "only the method document travels," credentials stay local. Guardrails: ambiguity-scored briefing interviews gating builds, verification as tri-state (verified/unverified/blocked), deterministic Policy Gate outside the LLM, curator-gated memory promotion. 1.1k stars. Steal: routing cards with **anti-triggers**; verification tri-state; policy enforced by pipeline code, not prompt text.

### agenttier/agenttier
Kubernetes-native sandbox platform: a **Sandbox CRD** with PVC-backed persistence. Requires a cluster — not template-repo material. Steal (conceptually): volume-snapshot cloning for byte-identical workspace forks.

### coleam00/Archon
Now a "harness builder": YAML-defined DAG workflows mixing **deterministic nodes** (bash, tests, git ops) with **AI nodes** (plan, generate, review). Highly relevant: workflows are checked into `.archon/workflows/` *in target repos* and **repo-local workflows override bundled defaults**. Same YAML runs from CLI, Web UI, Slack/Telegram/Discord, GitHub webhooks via adapters. Guardrails: approval gates, isolated worktree per run, deterministic validation phases, structured-output enforcement. Very mature: 23.2k stars, 19 bundled workflows. Steal: deterministic+AI node mixing; repo-overrides-defaults workflow resolution.

### razzant/claudexor
Local control plane routing one conversation thread across Claude Code/Codex/Cursor/OpenCode with **quota-aware account rotation**. Checked in: versioned `.claudexor/` (protected-path globs, deterministic test gates with exact argv, credential profile names); "files are the source of truth." **Continuation packets** (delta summaries of missed turns) bridge harness/account switches. Guardrails: `--max-usd` budget caps (unknown cost never reported as $0), protected paths pause for human approval, best-of-N races with independent reviewers. Steal: protected-path globs checked into the repo; continuation packets; "typed evidence over vibes."

### codecast-sh/codecast
Daemon that watches local agent history files and syncs to a server, giving a **live triage inbox** plus `cast ask`/`cast search` so agents can query team session memory. Integration = snippets injected into `~/.claude/CLAUDE.md`, `~/.codex/AGENTS.md`, `~/.cursor/rules/` — nothing repo-level. Guardrails: secret redaction pre-sync, per-conversation privacy tiers. Steal: graceful-degradation capability matrix; instruction-snippet-injection as universal integration.

### mathomhaus/guild
Single Go binary + embedded SQLite exposed as an **MCP server**: Quests (atomic tasks, dependencies, atomic parallel-safe claims, cascade-unblocking), Lore (typed knowledge: observation/decision/research/principle with auto-staling — research 30d, decisions 180d), Oaths (principles auto-loaded at session start), Briefs (handoff notes for the next session). `guild init` writes an `AGENTS.md` block; harness-agnostic. 313 stars, Apache-2.0. Steal: typed knowledge with expiry policies; the Brief (structured session handoff) — both implementable as pure Markdown conventions.

### dazuiba/handoff
Slash-commands delegating a task from a live session to DeepSeek/Gemini/Codex/Opus. Core primitive: the **`RESULT=<path>` protocol** — spawn isolated background CLI, stream output to disk, drop a one-line pointer into the parent session, resume later. Steal: result-pointer files as the cheapest possible cross-agent handoff contract.

### moshthepitt/lionclaw
Local control plane making agent CLIs "durable, auditable workers." Primitive: the **project instance** — a `.lionclaw/` directory (TOML configs) holding identity, runtime profiles, sessions, audit records, skills, jobs. Runtimes interchangeable. Guardrails: containerized confinement, mount allowlists, staged auth, audit trails. Steal: separating the control/audit boundary from the runtime.

### NVIDIA/NemoClaw
Alpha reference stack for running agents inside NVIDIA OpenShell sandboxes. Version-controlled YAML network policies and agent blueprints; egress rules with operator approval. Steal: checked-in, reviewable network egress policy as a guardrail artifact.

### gintasz/neuralyzer
A single no-arg tool: the agent calls `neuralyzer`, all messages are wiped and the first message is resent — the agent loops itself with clean context. Plugins for pi and OpenCode; notably Claude Code doesn't expose the hooks needed. Steal: agent-initiated context reset as a loop primitive.

### omnigent-ai/omnigent
Alpha "meta-harness": agents defined in **short YAML files** (prompt + `harness:` + tools); swap harnesses by editing one field. Three-level policies (server/agent/session): shell approval gates, tool-call limits, spend caps. Steal: `harness:` as a one-line executor field; three-scope policy layering.

### open-multi-agent/open-multi-agent
TypeScript runtime where **a coordinator plans the task DAG at runtime** from a goal (`runTeam()`), vs. explicit pipelines (`runTasks()`). Guardrails are strong: default-deny tools, per-call gating, token/cost budgets, loop detection, durable approvals, checkpoint/resume, plan repair, execution receipts, offline run viewer. Mixed runtimes on one DAG. 6.8k stars, MIT, production users. Steal: default-deny tool posture; execution receipts.

#### Addendum (2026-08-22): open-multi-agent.com site review

A direct review of the project's website, done while evaluating OMA as a
potential open-seed backend. Findings beyond the survey entry above:

- **SDK-first, not files-first.** OMA is consumed as a TypeScript library:
  teams, tools, budgets, and approval policies are declared in code at
  runtime. There are no declarative manifests checked into the repo by
  default. This is the opposite of open-seed's files-first bet, so OMA is
  **not a backend candidate** for the `.seed/backends/` plugin system — its
  state lives in a process, not in git, and there is no CLI port to wrap.
  It remains a design reference for guardrail semantics.
- **Honest budget semantics.** The docs state a token/cost budget "can
  overshoot by one model turn" — the budget is checked between turns, not
  mid-stream. This is the same honesty posture open-seed adopts for R6
  (budgets are advisory circuit breakers, not hard walls) and is worth
  citing as precedent for documenting enforcement gaps instead of implying
  hard guarantees.
- **`planOnly` mode.** A run can be executed with `planOnly` to produce and
  inspect the task DAG without executing any task — a cheap dry-run gate.
  Analogous to open-seed's plan-before-work parking (`blocked_on: plan:`),
  but enforced by the runtime rather than by review.
- **Offline Run Viewer.** Execution receipts render in a local viewer with
  no server dependency — precedent for open-seed's receipts being
  self-contained JSON that tooling can render without a service.
- **ExecutionRouter.** Per-task routing of model/runtime on one DAG —
  runtime-level precedent for open-seed's per-squad-member harness/model
  heterogeneity (design doc §6), with the same caveat that permission-tier
  fidelity varies per runtime.
- **Correction to the entry above:** the earlier "append-only plan repair"
  claim is not documented on the site. Plan repair exists (the coordinator
  can revise the DAG mid-run), but the site does not describe the repair
  log as append-only. The claim has been softened in the survey entry
  accordingly.

### RightNow-AI/openfang
Rust "Agent OS," single binary. Primitive: **Hands** — autonomous capability packages bundling `HAND.toml` manifest + system prompt + **SKILL.md expertise file** + guardrails, running on schedules. 16 security layers incl. WASM sandboxing, Merkle audit chains, Ed25519 manifest signing. Steal: manifest signing for skill provenance; SKILL.md adopted even outside the Anthropic ecosystem.

### rivet-dev/sandbox-agent
Rust server + TS SDK: one HTTP/SSE API driving six agents via internal adapters emitting a **normalized JSON event schema** — "write your code once, swap agents with a config change." Ships per-harness skill dirs (`.agents/skills`, `.codex/skills`, `.opencode/skills`). Steal: a normalized cross-agent event schema; per-harness skill directory layout.

### amaar-mc/wit
Coordination daemon that prevents conflicts *before code is written* by locking **symbols, not files**: Tree-sitter parses ASTs so two agents can safely edit different functions in one file. Primitives: Intents (announced scope), Locks (function/class/type), Conflicts (advisory warnings), **Contracts** (agreed signatures — the only hard block, enforced by git pre-commit hooks). Agents integrate via auto-generated CLAUDE.md instructions, a Claude Code plugin, or JSON-RPC. Intent-to-commit tracking via **git trailers**. Steal: advisory-by-default/enforce-only-contracts; signature contracts enforced by pre-commit hooks (a pure checked-in mechanism); git trailers linking work-in-progress to commits.

### BloopAI/vibe-kanban
The category's popularity ceiling: kanban planning + agent workspaces + diff review, 10+ agents. 27.9k stars, Apache-2.0 — **but the project is sunsetting**, and config is env-vars/external app, nothing checked in. Lesson for open-seed: external-app orchestration is fragile even at 28k stars; the plan→execute-isolated→review-diff loop is the durable idea, not the app.

### snarktank/antfarm
TypeScript CLI installing an agent team (planner, developer, verifier, tester, reviewer) into OpenClaw with one command. Core primitive: the **"Ralph loop"** — every step runs in a fresh session; state persists only via git history and progress files. Workflows are plain YAML + Markdown (bundled: feature-dev, security-audit, bug-fix) with steps, acceptance criteria, retry/escalation logic. "YAML + SQLite + cron. That's it." Guardrails: curated-repo-only installs, pre-merge review for prompt injection, and **separate verifier agents so no agent marks its own work done**. 2.5k stars, active. Steal: fresh-context-per-step with git as memory; the verifier-role separation; reviewable-before-install workflow text.

---

## SYNTHESIS

**How projects achieve harness-portability.** Four mechanisms recur, in ascending order of coupling:

1. **Markdown/config instructions in known locations** — the universal LCD. Every harness reads an instruction file (`CLAUDE.md`, `AGENTS.md`, `.cursor/rules/*.mdc`, `GEMINI.md`), so swarm-protocol ships its entire client as a CLAUDE.md snippet, guild and Agentlas-OS emit `AGENTS.md` blocks, wit auto-generates CLAUDE.md, codecast injects per-harness snippets. Zero-dependency, works everywhere, but advisory-only.
2. **CLI invocation** — the dominant execution abstraction. crewplane ("invokes provider CLIs directly instead of wrapping in a vendor SDK"), sub-agents-skills, handoff, antfarm, claudexor, lionclaw all shell out to `claude`/`codex`/`gemini` with env-var credentials. This is what a template repo can do with plain scripts.
3. **MCP** — for stateful coordination services (swarm-protocol, guild): any MCP client connects, but a server must run.
4. **Adapter daemons with normalized event schemas** (sandbox-agent, agenttier, omnigent) — richest control, heaviest footprint; out of scope for repo content.

**Emerging standards.** (i) **AGENTS.md** is the cross-vendor instruction file — Codex-native, and even Claude-centric tools now write it alongside CLAUDE.md. (ii) **SKILL.md / Agent Skills format** has escaped Anthropic: agent-runbook compiles *to* it, sub-agents-skills claims 30+ compatible tools, sandbox-agent ships skill dirs for three harnesses, even Rust-native openfang embeds SKILL.md. It's becoming the "executable capability" unit the way AGENTS.md is the "ambient instruction" unit. (iii) **MCP** is uncontested for tool/coordination surfaces. (iv) **Slash commands / plugin marketplaces** (`.claude/commands/`, `/plugin marketplace add owner/repo`) are the distribution channel — a GitHub repo *is* a plugin registry. (v) **Frontmatter-on-Markdown** is the de-facto metadata convention everywhere.

**Lockfile/versioning ideas.** skillfold is the reference design: manifest = intent, lockfile = exact commit SHA/version **plus sha256 content hash**; `install` never moves pins, only `update` does; `install --frozen` = npm-ci for CI; `check` audits drift offline; hand-authored skills are out of scope so the tool can't clobber them. Complementary: openfang's Ed25519-signed manifests (provenance), gnap's `.gnap/version` protocol-version file, antfarm's curated-source-only installs with pre-merge prompt-injection review. For open-seed: pin skills by SHA+hash, verify frozen in CI, and treat skill *content review* (injection audit) as part of the update PR.

**Five most transferable ideas for open-seed:**

1. **Manifest + lockfile for skills** (skillfold): check in `skills.yaml` + `skills.lock` with SHA+sha256 pins; a small sync script fans skills out to `.claude/skills/`, `.agents/skills/`, etc.; CI runs the frozen check. Solves team reproducibility with zero daemons.
2. **Git-as-coordination-bus** (gnap, + antfarm's Ralph loop): a checked-in task directory of JSON/Markdown task files with an explicit state machine, separate per-attempt run records carrying tokens/cost, an agent-ID commit convention, and humans as first-class agents. No server, full audit trail, works with any harness that can commit.
3. **Backend-annotated Markdown agents with permission tiers** (sub-agents-skills): `.agents/*.md` with `run-agent:` / `model:` / `permission: read-only|safe-edit|yolo` frontmatter and `## Done When` criteria, executed by one small checked-in script. Gives per-task harness routing and a legible security posture in pure text.
4. **Compile-and-validate workflows before execution** (agent-runbook, crewplane, Archon): keep orchestration in YAML/Markdown-with-frontmatter in-repo, add a `validate` step (DAG acyclicity, schema/contract closure, referenced files exist) that runs in CI *and* as a pre-run gate, pass state between steps as files, and include a mock provider so workflows are testable without credentials.
5. **Advisory coordination + hard gates only where cheap** (wit, swarm-protocol, antfarm, claudexor): declare intended file/symbol scope when claiming a task and warn on overlap (advisory); enforce only the mechanical invariants — signature contracts and protected-path globs via pre-commit hooks/CI, and never let an agent verify its own work (separate verifier role). This matches git's philosophy and needs nothing beyond hooks and conventions already living in the repo.

A cross-cutting sixth lesson from the graveyard and the winners alike: vibe-kanban (28k stars, sunsetting) versus Archon (workflows checked into target repos, thriving) is the category's clearest signal — **orchestration that lives as reviewable repo content outlasts orchestration that lives in an app**, which is precisely open-seed's bet.
