# Research: Personal Assistants + Inactive Projects (pattern scan)

> Category survey of [awesome-agent-orchestrators](https://github.com/andyrewlee/awesome-agent-orchestrators),
> researched 2026-08-21 for the open-seed design study. Light pattern scan focused on transferable
> ideas: memory conventions, persistence, scheduling/heartbeats, guardrails.

## Personal Assistants

**openclaw/openclaw**: the reference architecture for this whole category. Gateway-centric: a local "control plane for sessions, tools, events, and channel connections," with CLI/TUI/Control-UI clients, channel adapters, and companion nodes for device-local actions. **Memory conventions (the most-copied pattern in the ecosystem):** a workspace holding four markdown files: `USER.md` (stable preferences/relationships), `MEMORY.md` (long-term durable facts, loaded at session start under a separate token budget), `memory/YYYY-MM-DD.md` daily notes (today + yesterday auto-loaded), and optional `DREAMS.md` (consolidation summaries for human review). Two mechanisms matter for a template: a **pre-compaction "memory flush"**: a silent turn reminding the agent to write important context to files before the conversation is summarized, and a background **"dreaming" consolidation** that scores short-term recall candidates and promotes only qualified items into long-term memory, leaving a human-reviewable audit trail in `DREAMS.md`. Skills live in a `skills/` directory, plugins in a Plugin SDK with community sharing, plus slash commands. Guardrails: "treat inbound messages as untrusted input," DM pairing approval, and opt-in sandboxing.

**NousResearch/hermes-agent**: multi-platform self-improving agent. Memory in `MEMORY.md` + `USER.md` (same convention as OpenClaw), plus FTS5 session search with LLM summarization for cross-session recall. The transferable idea is its **closed learning loop**: after completing a complex task the agent autonomously writes a skill file, and skills "self-improve during use"; "periodic nudges" prompt memory curation. Skills follow the agentskills.io open standard. Guardrails: command approval, container isolation, allowlists. Built-in cron scheduler.

**mikeyobrien/rho**: the cleanest heartbeat exemplar. Always-on daemon with a **"proactive heartbeat": autonomous check-ins every 30 minutes by default**, adjustable and manually triggerable. Memory is a two-tier split worth stealing: **append-only structured log (`brain.jsonl`) + human-readable markdown knowledge vault**, inspectable/editable. Skills are "portable markdown runbooks" in `skills/`. Guardrails: local-first state, allowlists, mention gating, outbound policy limits.

**MarlBurroW/hivekeep**: persistent agent team in one Docker container + SQLite. Agents have durable identities (name, role, expertise, model) and message each other via request/reply with async sub-agents. Memory: each agent is "one continuous session, never resets," backed by hybrid retrieval (vector KNN + FTS5 fused, re-ranked by importance/temporal decay) with **progressive compaction that summarizes old context but "never deletes the originals."** Notable guardrails: an AES-256-GCM **secrets vault never exposed to the LLM** (secrets route UI→vault, bypassing the model), token-transparency Context Viewer, per-agent cost tracking, and "Scout" delegation of read-only chores to cheaper models.

**razzant/ouroboros**: durable-identity agent that can rewrite its own implementation. Runtime data (identity, state, history, skills) in a **git-backed repository: "identity, memory, dialogue, knowledge, reflections, and version history form one ongoing biography."** Scheduling via CLI cron plus a "background consciousness" for reflection. Governance is the transferable bit: a checked-in constitution (`BIBLE.md`, 13 principles), protected surfaces, restart checks making self-modification traceable, and a marketplace where extensions are security-audited before enablement.

**sentientwave/automata**: Matrix-native workspace coordinating humans + agents. Differentiator: **Temporal durable workflows**: work saves progress step-by-step so it survives restarts, with retries and clear execution records. Guardrails via role-based permissions, tool whitelisting, "company laws" applied during reasoning, governance (laws/proposals/voting), admin activity logs.

**cloudflare/cloudflare-os**: "company OS" on Workers, each workspace a Durable Object. Transferable idea: **"Gatekeepers"**: capability-based sandboxed wrappers around external services (GitHub, Google, Slack) that add authorization, logging, and human-in-the-loop approval, and can *simulate outcomes before approving*. "Blueprints" are shareable app templates.

**kcosr/assistant**: panel-based multi-agent workspace (runs Claude Code/Codex/Pi as subprocesses). Sessions as append-only JSONL event logs; **git auto-versioning of plugin data at configurable intervals**; plugins export CLIs as reusable skills for external agents. Per-agent tool allowlists.

**accomplish-ai/coworker**: discontinued; README-only. Skip.

**DenchHQ/DenchClaw**: OpenClaw distribution for CRM work; profile-isolated config, daemon, device-pairing approval. Nothing new beyond upstream.

**b1rdmania/ghostclaw**: deliberately tiny (~4K LOC) OpenClaw alternative. Per-group `CLAUDE.md` personality files, markdown skills security-scanned before install, and a good cost guardrail: **daily budget cap env var (`GHOSTCLAW_DAILY_BUDGET_USD`) with automatic fallback to cheaper models when exceeded.**

**nearai/ironclaw**: Rust "agent OS." Path-based workspace filesystem for notes/logs/context + identity files; Routines Engine (cron, event triggers, webhooks) and a heartbeat system. Strongest sandbox story in the set: **WASM tools with capability-based permissions and endpoint allowlists, credential injection at the host boundary with leak detection**, prompt-injection pattern detection. Orchestrator/worker multi-agent with per-job auth tokens.

**smixs/iva**: Telegram assistant writing into an Obsidian-compatible vault. Best-articulated **hierarchical memory rollup ("Memory Tree")**: verbatim `daily/YYYY-MM-DD.md` leaves → `summaries/` daily/weekly/monthly/yearly branches → a trunk `CORE.md` capped at ~1,200 chars injected into every prompt, plus typed cards (contacts/projects/decisions). A nightly job distills upward, *rewriting* facts rather than accumulating. Guardrails: prompt-injection sanitizer on all ingested web/vision content, secret-redaction gate on outbound messages, fail-closed allowlist.

**z80dev/lemon**: Elixir/OTP platform; OTP-supervised per-run processes for crash recovery, SQLite FTS memory, and notably **compiler/AST-enforced architectural boundaries plus a contract-test compliance kit for extension points**: guardrails as CI, not prompts. Also deterministic offline simulation arenas for benchmarking agents.

**leon-ai/leon**: v2 preview. Layered memory (durable prefs / day-to-day / recent) with a "compact self-model"; skills hierarchy `Skills → Actions → Tools → Functions` split into `native/` (controlled) and `agent/` (backed by `SKILL.md` workflows); a "bounded proactive pulse."

**netease-youdao/LobsterAI**: Electron desktop shell over the OpenClaw runtime. Same markdown memory convention (`MEMORY.md`, `USER.md`, `SOUL.md`, daily notes) plus SQLite; "Expert Kits" bundle capability selections; permission gates on sensitive tool actions.

**lucinate-ai/lucinate**: terminal chat client for OpenClaw/Hermes/Ollama. Routines as markdown step files (`routines/<name>/STEPS.md`); thin client; little else transferable.

**aiming-lab/MetaClaw**: proxy layer that makes any agent learn from live traffic: auto-summarizes conversations into markdown skill files, retrieves top-6 per turn. Clever scheduling idea: **defers weight updates/heavy work to inactive windows** (sleep hours, keyboard idle, calendar-detected meetings).

**HKUDS/nanobot**: lightweight Python framework; cron tool, background gateway, inline subagents, "Dream" long-term memory. Generic.

**nanocoai/nanoclaw**: container-per-agent-group with per-group `CLAUDE.md` + memory and per-session SQLite (single-writer). Two good ideas: **60-second sweep cycle** (detects stale sessions and due scheduled messages; containers wake on message or schedule), and **branch-based module distribution**: channel adapters live on a `channels` branch, installed per-fork, so trunk ships only registry + infrastructure. Guardrails are OS-level (container isolation, explicit mounts, credentials via vault never entering the container).

**nullclaw/nullclaw**: 678KB Zig binary. Skills from **TOML/JSON manifests or YAML-frontmatter markdown**; cron with JSON persistence plus per-agent configurable heartbeat intervals; deny-all channel allowlists by default; Landlock/Firejail/Bubblewrap/Docker sandboxes.

**sipeed/picoclaw**: Go, runs on $10 hardware. JSONL memory store; `SKILL.md` modules + registry; split of `config.json` (non-sensitive) vs `.security.yml` (secrets); command-job gates requiring approval for cron-executed commands.

**agentscope-ai/QwenPaw**: AgentScope-based assistant. Three-layer memory (live context / verbatim history / self-evolving knowledge base) where conversations convert into "readable, editable, searchable, linked Markdown memory." Strong guardrail stack: kernel sandboxes (Seatbelt/Bubblewrap/AppContainer), **Tool Guard: a YAML rule engine detecting injection/path traversal/obfuscation**, File Guard for sensitive directories, and pre-activation skill scanning.

**rowboatlabs/rowboat**: desktop coworker; indexes email/meetings/Slack into an Obsidian-style backlinked markdown knowledge graph, all local plain files. Background agents trigger on events or schedules.

**tnm/zclaw**: ESP32 firmware assistant. Charming, not transferable.

**zeroclaw-labs/zeroclaw**: Rust runtime. Event-triggered SOPs (MQTT/webhook/cron), resumable runs with approval gates, and two transferable guardrails: a **default `supervised` autonomy mode (medium-risk ops need approval, high-risk blocked; explicit "YOLO mode" opt-out)** and **cryptographic "tool receipts"** as an audit trail of actions.

## Inactive / resting

- **21st-dev/1code**: archived 2026-07. Orchestration UI for Claude Code/Codex; git-worktree isolation and plan-mode-before-execution were its guardrails.
- **ariana-dot-dev/ariana**: 404, repo gone.
- **yoheinakajima/babyagi3**: active-ish. Three-layer memory (immutable event log → LLM-extracted knowledge graph → hierarchical summaries refreshed on staleness) plus a persistent `user_preferences` summary learned from feedback; background loops; budget caps + spend approval thresholds.
- **moltlaunch/cashclaw**: autonomous freelance-work agent; BM25 knowledge search with 30-day temporal decay, rotating "self-study" sessions; guardrails are hard numeric caps (3 concurrent tasks, 10 LLM turns/task).
- **getclawe/clawe**: "Trello for OpenClaw agents." Four role agents (lead/editor/designer/SEO), each with an isolated workspace of `SOUL.md`, `MEMORY.md`, and **`HEARTBEAT.md` defining the wake-up routine**; cron heartbeats **staggered 15 minutes apart to avoid rate limits**, with 1-hour trigger tolerance for crash resilience; shared backend for task state.
- **Dimillian/CodexMonitor**: active Tauri app orchestrating Codex agents; worktree/clone agents for parallel isolated work. UI tool, light on transferable conventions.
- **letta-ai/lettabot**: archived 2026-05 into Letta Code. One unified agent across all channels with persistent memory; heartbeats triggering periodic check-ins in "Silent Mode" (only speaks when a cron has an explicit delivery target); **read-only tools by default**; outbound-connections-only architecture.
- **Michaelliv/mercury**: archived 2026-04. SQLite memory with "ambient context" captured from non-triggering group messages; RBAC blocking denied CLIs at the bash level.
- **marian2js/opengoat**: hierarchical agent org ("create 'CTO' --manager --reports-to goat"); role-differentiated board skills; named persistent sessions.

## Synthesis for open-seed

**Memory-file conventions worth adopting.** The ecosystem has converged hard on *plain markdown files with reserved names in a checked-in workspace*: exactly what a template repo can standardize. The de-facto standard (OpenClaw, Hermes, LobsterAI, clawe, ghostclaw) is: `MEMORY.md` (durable facts, loaded every session under a token budget), `USER.md`/`SOUL.md` (stable preferences / identity), and `memory/YYYY-MM-DD.md` daily notes with only today+yesterday auto-loaded. Two refinements worth layering on: **iva's hierarchical rollup** (daily → weekly → monthly summaries feeding a hard-capped `CORE.md` trunk, with facts *rewritten* not appended: a scheduled distillation job is template-able) and **rho's two-tier split** of append-only machine log (JSONL) + curated human-readable vault. Ouroboros adds the cheapest durable-memory trick of all: keep the memory directory **git-backed**, so history, auditability, and rollback come free: natural for a template repo where memory files are already in-repo.

**Scheduling/heartbeat patterns.** Three composable patterns recur: (1) **interval heartbeat with a checked-in routine**: clawe's `HEARTBEAT.md` is the best template idea here: each agent's wake-up behavior is a versioned file, not code, and heartbeats are *staggered* per agent to avoid rate-limit collisions, with a tolerance window for crash resilience; (2) **cron + one-shot tasks persisted to disk** so schedules survive restarts; (3) **sweep loops** (nanoclaw's 60s stale-session/due-message sweep). Two enrichments: OpenClaw's **pre-compaction memory flush** (a mandatory "write it down before you forget" turn: trivially expressible as a hook/convention in a template) and MetaClaw's **idle-window deferral** of expensive maintenance.

**Skills packaging.** Overwhelming convergence on **`SKILL.md`-style markdown-with-frontmatter modules in a `skills/` directory** (picoclaw, nullclaw, leon, rho, ghostclaw, MetaClaw, hermes/agentskills.io), often with a registry and *pre-activation security scanning* (QwenPaw, ghostclaw). A template repo should ship the directory convention + a manifest schema + a scan step in CI. Leon's `native/` vs `agent/` split (deterministic code vs SKILL.md workflow) is a useful taxonomy.

**Top 5 transferable ideas for open-seed:**
1. **Standard memory workspace as checked-in convention**: `MEMORY.md` + `USER.md` + `memory/YYYY-MM-DD.md` + a size-capped always-injected core file, git-versioned, with a documented pre-compaction flush rule and a scheduled rollup/consolidation job that writes an auditable summary (the `DREAMS.md` pattern).
2. **`HEARTBEAT.md` per agent** (clawe/nullclaw/rho): wake cadence and check-in routine as a versioned file; stagger offsets across agents; persist schedules to disk so they survive restarts.
3. **Tiered autonomy as default config** (zeroclaw): `supervised` mode out of the box: low-risk auto, medium-risk approval-gated, high-risk blocked: with explicit opt-out; pair with hard numeric caps (daily budget env var à la ghostclaw, max concurrent tasks/turns à la cashclaw).
4. **Secrets never touch the model**: split `config.json` vs `.security.yml` (picoclaw), vault-routed credentials injected at the host boundary (hivekeep, ironclaw, nanoclaw), plus an outbound secret-redaction gate (iva). Easily encoded as template file layout + lint rule.
5. **Guardrails as checked-in policy files enforced by machinery, not prompts**: QwenPaw's YAML Tool Guard rules, ouroboros's constitution file with protected surfaces, lemon's contract-tested extension points, zeroclaw's tool receipts for audit. For a template repo, that means a `policies/` directory with a rule schema + CI checks, and an append-only action log for auditability.

Notable gaps: coworker is dead, ariana is 404, lettabot/mercury/1code are archived (lettabot's "silent heartbeat + read-only-by-default tools" still worth borrowing); opengoat and clawe are the only real team-topology references: role hierarchy with role-differentiated board skills, and shared task backend + isolated per-agent workspaces respectively, both of which map directly onto open-seed's orchestration + task-tracking goals.
