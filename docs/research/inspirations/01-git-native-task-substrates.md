# Deep Dive: Git-Native Task Substrates

> Implementation-grade deep dive for the open-seed design study, researched 2026-08-21/22.
> Covers beads, gnap, ORCH, squad, and tick-md. All findings are from cloned source, not READMEs alone.
> Note: Squad is **`bradygaster/squad`** (MIT), not a github/githubnext org project — correcting the earlier survey.

## 1. steveyegge/beads (`bd`) — distributed graph issue tracker, Dolt-backed

**Storage architecture (correction to earlier findings):** The JSONL→Dolt migration is *done*. Docs state: "Dolt is the only storage backend." Issues live in a local Dolt DB; sync uses `refs/dolt/data` on your git remote; **`.beads/issues.jsonl` is a passive export** for viewers/interchange — explicitly *not* the sync protocol ("Do not use routine `bd import .beads/issues.jsonl` as a replacement for sync"). Modes: **Embedded** (default, `bd init`) → data in `.beads/embeddeddolt/`, one file-locked writer; **Server** (`bd init --server` or `BEADS_DOLT_SERVER_MODE=1`) → `.beads/dolt/`, many concurrent writers via `dolt sql-server`.

**`.beads/` layout:** `embeddeddolt/` or `dolt/` (DB), `issues.jsonl` (export, refreshed by pre-commit hook when `export.auto=true`), `config.yaml` + `config.local.yaml` (tool config; `sync.remote` lives here), `metadata.json` (records `dolt_database` name, mode), `.local_version`, `recipes.toml`, optional `redirect` (points at a shared .beads dir), `formulas/`. Config precedence: flags > `BD_*` env > `$BEADS_DIR/config.yaml` > repo `.beads/config.yaml` > `~/.config/bd/config.yaml` > `~/.beads/config.yaml`.

**Issue field schema** (from `internal/types/types.go`, JSON names): core: `id`, `title`, `description`, `design`, `acceptance_criteria`, `notes`, `spec_id`; workflow: `status` (`open|in_progress|blocked|deferred|closed|pinned|hooked`), `priority` (int 0–4, 0 valid = P0), `issue_type` (`bug|feature|task|epic|chore` + `decision`, `merge-request`, `molecule`), `is_blocked` (journal-only projection); assignment: `assignee`, `owner` (git author email, "CV attribution"), `estimated_minutes`; timestamps: `created_at`, `created_by`, `updated_at`, `started_at`, `closed_at`, `close_reason`, `closed_by_session`; **leasing** (claim TTL + heartbeat): `lease_expires_at`, `heartbeat_at`, `lease_granted_node` (a lease is only enforceable on the replica that granted it — foreign leases refuse reclaim without override); concurrency: `RowVersion` (`row_lock` cell, random per write, equality-only optimistic token, surfaced as `revision` in the HTTP detail DTO); scheduling: `due_at`, `defer_until` (hidden from `bd ready` until then); integration: `external_ref` (`"gh-9"`, `"jira-ABC"`), `source_system`; `metadata` (free JSON); compaction fields; relational: `labels[]`, `dependencies[]`, `comments[]`; messaging/wisps: `sender`, `ephemeral`, `no_history`, `wisp_type`, `storage_class`; markers: `pinned`, `is_template`; gates: `await_type` (`gh:run|gh:pr|timer|human|mail`), `await_id`, `timeout`, `waiters[]`; molecules/swarm: `mol_type` (`swarm|patrol|work`), `work_type` (`mutex|open_competition`), `bonded_from[]`; events: `event_kind`, `actor`, `target`, `payload`. A deterministic `ContentHash` (SHA-256 over substantive fields, excluding ID/timestamps) detects identical content across clones.

**Verbatim JSONL export line** (from the repo's own `.beads/issues.jsonl`):

```json
{"id":"bd-main-idj","title":"Pattern-collapse pass: mechanical cruft inventory and reduction","description":"Quantify near-duplicate functions...","status":"in_progress","priority":2,"issue_type":"chore","owner":"maphew@gmail.com","created_at":"2026-04-18T16:19:12Z","created_by":"matt wilkie","updated_at":"2026-04-18T16:30:16Z","started_at":"2026-04-18T16:30:16Z","dependency_count":0,"dependent_count":0,"comment_count":0}
```

Dependency records serialize as `{id?, issue_id, depends_on_id, type, created_at, created_by?, metadata?, thread_id?}`.

**ID scheme** (`internal/idgen/hash.go`): `GenerateHashID(prefix, title, description, creator, timestamp, length, nonce)` → SHA-256 of `"title|description|creator|unixnano|nonce"`, truncated to 2–5 bytes, **base36-encoded** at length 3–8, emitted as `prefix-hash` (e.g. `bd-a1b2`). Nonce handles collisions; content+time hashing gives zero-conflict IDs across concurrent clones. Real IDs also show rig-scoped prefixes (`bd-main-idj`).

**Dependency types:** blocking: `blocks`, `parent-child`, `conditional-blocks` (B runs only if A fails), `waits-for` (fanout gate); non-blocking: `related`, `discovered-from`; graph links: `replies-to`, `relates-to`, `duplicates`, `supersedes`; provenance: `authored-by`, `assigned-to`, `approved-by`, `attests`; plus `tracks`, `until`.

**CLI surface** (~70 top verbs; docs/CLI_REFERENCE.md). Working set: `create` (`-t type -p 0-4 --description --deps discovered-from:bd-123 --acceptance --design --validate --json`; batch from markdown/graph JSON), `q` (quick capture → ID only), `update` (`--claim` = "Atomically claim the issue (sets assignee to you, status to in_progress; idempotent if already claimed by you)"), `close <id> --reason`, `reopen`, `assign`, `defer`/`undefer`, `supersede`, `duplicate`, `show`, `list`, `search`, `query`, `count`, `note`, `comment`, `label`, `link`, `dep add|remove|list|tree|cycles|relate`, `epic status|close-eligible`, `children`, `graph`. **`bd ready`**: "Show ready work (open issues with no active blockers). Excludes in_progress, blocked, deferred, and hooked" — flags include `--claim` (atomically claim first match), `--claim-next`, `--explain`, `--assignee`, `--label/--label-any`, `--parent`, `--sort priority|hybrid|oldest`, `--unassigned`, `--gated`, `--mol`. Memory: `prime`, `remember`/`recall`/`memories`/`forget`, `kv get|set`. Coordination: `gate create|check|resolve|add-waiter|discover`, `merge-slot acquire|release|check|create` (serialized conflict resolution), `swarm`, `mol`, `formula`, `human list|respond|dismiss`. Sync/maint: `export`/`import` (JSONL), `dolt push|pull|start|stop|remote`, `vc commit|merge|status`, `branch`, `backup`, `compact`, `flatten`, `gc`, `prune`, `doctor`, `migrate schema` (idempotent), `upgrade status|ack|review`, `worktree create|list|remove|info`, `init`, `onboard`, `setup`, `sql`, `batch`.

**What `bd init` writes into AGENTS.md** — a fenced managed block, verbatim markers: `<!-- BEGIN BEADS INTEGRATION v:1 profile:full hash:bacef91e -->` … `<!-- END BEADS INTEGRATION -->`. Content (abridged, exact phrasing): "**IMPORTANT**: This project uses **bd (beads)** for ALL issue tracking. Do NOT use markdown TODOs…"; Quick Start (`bd ready --json`; `bd create "Issue title" --description="…" -t bug|feature|task -p 0-4 --json`; `bd update <id> --claim --json`; `bd close bd-42 --reason "Completed" --json`); issue types and priority meanings (0=critical … 4=backlog); the 5-step agent workflow (check ready → claim atomically → work → create `discovered-from`-linked issues → close); "Architecture in one line: issues live in a local Dolt DB; sync uses `refs/dolt/data` on your git remote; `.beads/issues.jsonl` is a passive export"; rules (✅ always `--json`, ✅ link discovered work, ❌ no markdown TODO lists); and **Agent Context Profiles** (`conservative` default / `minimal` / `team-maintainer`) governing whether agents may commit/push at session close. The version+hash in the marker is the migration mechanism for the managed block.

**MCP server** (`integrations/beads-mcp`, Python): tools `beads_ready_work`, `beads_list_issues`, `beads_show_issue`, `beads_create_issue`, `beads_update_issue`, `beads_claim_issue`, `beads_close_issue`, `beads_reopen_issue`, `beads_add_dependency`, `beads_add_comment`, `beads_list_comments`, `beads_add_note`, `beads_quickstart`, `beads_stats`, `beads_blocked`, `beads_inspect_migration`, `beads_get_schema_info`, `beads_repair_deps`, `beads_detect_pollution`, `beads_validate`, `beads_init`.

**Versioning/migrations:** `bd migrate schema` applies pending migrations idempotently (leases arrived in migrations 0054/0055); `bd upgrade status/ack/review` tracks binary-version changes; docs have an explicit era-detection table keyed on `.beads/metadata.json` + `.local_version` deciding embedded-vs-server upgrade paths. Execution metadata for orchestrators rides `metadata` (`execution_agent_type`, `execution_suggested_model`, `execution_reasoning_effort`, `execution_mode`, `execution_parallel_group`).

## 2. farol-team/gnap — Git-Native Agent Protocol

A ~400-line *protocol spec*, not a tool. Four entities, MIT. `.gnap/version` is "the protocol version as a plain integer (e.g. `4`)"; agents "SHOULD check this file on startup and refuse to operate if the version is higher than they support." Directory: `.gnap/{version, agents.json, tasks/*.json, runs/*.json, messages/*.json}`.

**Agent** (`agents.json`, array under `"agents"`). Required: `id`, `name`, `role`, `type` (`ai|human`), `status` (`active|paused|terminated`). Optional: `runtime`, `reports_to` (creates org tree), `heartbeat_sec` (default 300), `contact`, `capabilities[]`. ID `*` reserved for broadcast. Verbatim example entry:

```json
{ "id": "carl", "name": "Carl", "role": "CRO", "type": "ai", "runtime": "openclaw",
  "reports_to": "ori", "capabilities": ["sales", "outreach", "analytics"],
  "heartbeat_sec": 600, "status": "active" }
```

**Task** (`.gnap/tasks/{id}.json`). Required: `id` (matches filename), `title`, `assigned_to` (array), `state`, `created_by`, `created_at`. Optional: `parent`, `desc`, `priority` (0 = highest), `due`, `blocked` (bool) + `blocked_reason`, `reviewer`, `updated_at`, `tags[]`, `comments[]` of `{by, at, text}`. Verbatim example:

```json
{ "id": "FA-1", "title": "Build Q2 lead pipeline — 20 qualified leads",
  "desc": "Research and compile 20 qualified leads for Sebastian",
  "created_by": "ori", "assigned_to": ["carl"], "reviewer": "mayak",
  "state": "in_progress", "priority": 1, "due": "2026-03-19", "tags": ["Sales"],
  "created_at": "2026-03-12T09:00:00Z", "updated_at": "2026-03-12T10:30:00Z",
  "comments": [{ "by": "carl", "at": "2026-03-12T10:30:00Z", "text": "Found 8 leads so far..." }] }
```

**State machine:** `backlog → ready → in_progress → review → done`; reverse: `review → in_progress` (reviewer rejects), `blocked → ready` (unblocked); `blocked → cancelled`; terminals `done`, `cancelled`.

**Run** (`.gnap/runs/{task-id}-{attempt}.json`). Required: `id`, `task`, `agent`, `state` (`running|completed|failed|cancelled`), `started_at`. Optional: `attempt` (1-based), `finished_at`, `tokens` `{input, output}`, `cost_usd`, `result`, `error`, `commits[]` (git SHAs), `artifacts[]` (paths). "A failed run doesn't fail the task." Runs give "**cost tracking** (budget = sum of runs), **retry history**, **audit**, **performance**."

**Message** (`.gnap/messages/{id}.json`). Required: `id`, `from`, `to` (array; `["*"]` = broadcast), `at`, `text`. Optional: `type` (`directive|status|request|info|alert`), `channel`, `thread`, `read_by[]`.

**Heartbeat loop (verbatim):** `1. git pull → 2. Read agents.json → am I active? → 3. Read tasks/ → anything assigned to me? → 4. Read messages/ → anything new for me? → 5. Do the work → commit → git push → 6. Sleep until next heartbeat`.

**Commit grammar:** `<agent-id>: <action> [details]` — e.g. `carl: done FA-1 — Stripe test mode live`, `ori: create FA-3 onboarding-v2`, `leo: assign FA-1 to carl`. **Consistency:** "Eventual consistency, bounded by max heartbeat interval. Conflicts: Standard git merge. If conflict, pull + rebase + retry push. Ordering: timestamps." Note: GNAP has **no claim primitive** — work routes by `assigned_to`; last-writer-wins under rebase is accepted.

## 3. oxgeneral/ORCH — file-based agent runtime (TypeScript, MIT)

**`.orchestry/` layout file-by-file** (all IDs sanitized against `^[A-Za-z0-9._-]+$`): `config.yml`; `state.json`; `orchestry.lock`; `tasks/{id}.yml`; `agents/{id}.yml`; `runs/{id}.json` (metadata) + `runs/{id}.jsonl` (append-only events); `goals/{id}.yml`; `teams/{id}.yml`; `templates/default.md` (LiquidJS prompt templates); `logs/`; `context/{key}.json`; `messages/{id}.json`; `attachments/{taskId}/`; `.gitignore`; `workspace-exclude`. Root found by walking up for `.orchestry/`. Writes go through `atomicWrite()` (temp file + rename); JSONL reads via tail helper.

**Task record**: `id, title, description, status, priority, assignee?, labels[], depends_on[], created_at, updated_at, attempts, max_attempts, workspace_mode? ('shared'|'worktree'|'isolated'), workspace?, proof? {branch?, pr_url?, files_changed[], test_results?, agent_summary?}, review_criteria? ('test_pass'|'typecheck'|'lint'), review_results?, scope?[], feedback?, goalId?, attachments?[]`. **States and transitions** (richer than the README's 4-state diagram):

```ts
todo: ['in_progress', 'cancelled']
in_progress: ['review', 'retrying', 'failed', 'cancelled']
retrying: ['in_progress', 'failed', 'cancelled']
review: ['done', 'todo', 'cancelled']   // 'todo' = rejection with feedback
done: []          failed: ['todo', 'retrying']       cancelled: ['todo']
```

Terminal: `done|failed|cancelled`; dispatchable: `todo|retrying`; a task is blocked if any `depends_on` is non-terminal (deleted deps treated as resolved).

**Agent config** (`agents/{id}.yml`): `id, name, adapter, role?, status ('idle'|'running'|'error'|'disabled'), current_task?, autonomous?, stats {tasks_completed, tasks_failed, total_runs, total_runtime_ms, tokens_used?}, last_error?`, and `config: {command?, model?, effort? ('low'|'medium'|'high'), approval_policy? ('suggest'|'auto'|'manual'), max_turns?, timeout_ms?, stall_timeout_ms?, env?, system_prompt?, workspace_mode?, skills?[]}`.

**Run + JSONL events:** Run = `{id, task_id, agent_id, attempt, status ('preparing'|'running'|'succeeded'|'failed'|'timed_out'|'cancelled'), started_at, finished_at?, workspace_path, prompt, pid?, error?, tokens {input, output, reasoning, total, cache_read, cache_write}}`. Event lines: `{timestamp, type, data}` with types `agent_output|file_changed|command_run|tool_call|error|done`. Serve-mode log lines, verbatim from README:

```json
{"ts":"2026-03-17T03:00:10.000Z","level":"info","event":"agent:started","agentId":"agt_abc","taskId":"tsk_123","runId":"run_xyz"}
{"ts":"2026-03-17T03:12:45.000Z","level":"info","event":"task:status_changed","taskId":"tsk_123","from":"in_progress","to":"review"}
```

**IDs:** `tsk_${nanoid(7)}`, `agt_${nanoid(7)}`, `run_${nanoid(7)}` — random, typed prefixes, no content hashing.

**Lock mechanics**: PID lock file, only the watch daemon acquires it (one-shot commands don't). Acquire serialized by an in-process promise mutex; stale detection = dead PID **or** mtime older than `LOCK_STALE_MS = 60_000` (guards PID recycling — the holder must touch the lock each tick); creation is atomic `O_CREAT|O_EXCL`; a second `orch serve` exits with `LockConflictError`.

**Adapters:** not config files — TypeScript classes implementing `IAgentAdapter { kind; test(); execute(params): {pid, events: AsyncGenerator<AgentEvent>}; stop(pid) }`, registered in a registry, resolved by the agent's `adapter` field (claude, opencode, codex, cursor, pi, grok, antigravity, shell). The shell adapter takes the agent's `config.command`.

**CLI:** `orch init|doctor|update`; `agent add <name> --adapter X --role "..."|list|disable|enable`; `org list|deploy <template> [--goal]|export`; `task add "Title" -p 1|list|assign|cancel`; `team create --lead|join|add-task|disband`; `goal add|list|status <id> achieved`; `msg send|broadcast --team`; `context set <key> <value>`; `run --all --watch | run <task-id>`; `serve [--once] [--tick-interval ms] [--log-file] [--log-format json|text]`; `status`; `logs <run-id>`; `tui`; `config edit`. **`serve --once` semantics:** processes all existing todo tasks, skips autonomous task seeding, exits when everything is terminal; exit 0 = all done, exit 1 = has failures. Orchestrator tick = Reconcile (PIDs/stalls/zombies) → Dispatch (claim idle agents) → Collect.

## 4. bradygaster/squad — AI agent teams on GitHub Copilot (MIT)

`squad init` scaffolds `.github/agents/squad.agent.md` (coordinator) plus `.squad/`:

```
.squad/
├── team.md         # roster: Coordinator + Members tables (Name|Role|Charter|Status)
├── routing.md      # who handles what
├── decisions.md    # shared brain
├── ceremonies.md   # sprint ceremonies config
├── casting/        # policy.json, registry.json, history.json (model casting)
├── agents/{name}/  # charter.md + history.md per agent (+ scribe/)
├── skills/         # SKILL.md packs (test-discipline, secret-handling, …)
├── identity/       # now.md, wisdom.md
└── log/            # orchestration logs (Scribe-written)
```

"Commit this folder. Your team persists."

**Charter template** (verbatim skeleton): `# {Name} — {Role}` with sections **Identity** (Name/Role/Expertise/Style), **What I Own**, **How I Work**, **Boundaries** ("**I handle:** … **I don't handle:** … **When I'm unsure:** I say so and suggest who might know."), **Model** (Preferred: auto), **Collaboration** — which encodes the memory protocol: "Before starting work, read `.squad/decisions.md` for team decisions that affect me. After making a decision others should know, write it to `.squad/decisions/inbox/{my-name}-{brief-slug}.md` — the Scribe will merge it." — and **Voice** ("This agent has OPINIONS."). **history.md** template: `# Project Context` (Owner/Project/Stack/Created) + `## Learnings` append-only.

**decisions.md** scaffold: `# Squad Decisions` / `## Active Decisions` / `## Governance` (– All meaningful changes require team consensus – Document architectural decisions here – Keep history focused on work, decisions focused on direction). Merge conflicts are pre-solved: init writes `.gitattributes` with **`.squad/decisions.md merge=union`**; agents write to a per-agent inbox and a Scribe merges — two mechanisms open-seed should steal. There is also a git-notes state backend ("Squad Notes Protocol v1.0", per-agent namespaces, promoted to permanent state post-merge).

**Coordinator (`squad.agent.md`):** frontmatter `name: Squad`, `tools: ["*"]`; hard rules — "You are a DISPATCHER, not a DOER"; may not generate domain artifacts, bypass reviewer approval, or work inline except Direct Mode. Routing.md is a markdown table (`Work Type | Route To | Examples`) plus **issue routing** via labels: `squad` label = inbox → Lead triages → applies `squad:{member}` → member picks up next session. Rules: "Eager by default", "Scribe always runs… as background", "'Team, …' → fan-out". **Reviewer protocol — Strict Lockout:** on rejection the original author is locked out of that artifact ("No exceptions"); a *different* agent must revise (reassign or escalate/spawn); coordinator MUST refuse if the reviewer names the original author; repeated rejection locks out each successive author; full-lockout deadlock escalates to the human.

## 5. Purple-Horizons/tick-md — single-file TICK.md substrate

**Format** (verbatim from the project's own dogfooded `TICK.md`): YAML frontmatter + `## Agents` markdown table + `## Tasks` where each task is a `###` heading with an embedded YAML code fence:

```markdown
---
project: tick-md
title: Tick.md - Multi-Agent Coordination Protocol
schema_version: "1.0"
created: ...
updated: 2026-02-18T03:45:11.565Z
default_workflow: [backlog, todo, in_progress, review, done]
id_prefix: TASK
next_id: 52
---

## Agents
| Agent | Type | Role | Status | Working On | Last Active | Trust Level |
|-------|------|------|--------|------------|-------------|-------------|
| @claude-code | bot | engineer | idle | - | 2026-02-18T03:45:11.565Z | trusted |

## Tasks
### TASK-020 · Build tick validate command
    id: TASK-020
    status: done
    priority: high
    assigned_to: null
    claimed_by: null
    created_by: "@gianni-dalerta"
    created_at: 2026-02-08T01:14:22.398Z
    updated_at: 2026-02-08T01:14:28.374Z
    tags: [cli, validation]
    history:
      - {ts: ..., who: "@gianni-dalerta", action: created}
      - {ts: ..., who: "@claude-code", action: completed, from: backlog, to: done}
```

Full Task type adds: `due_date?`, `depends_on[]`, `blocks[]`, `estimated_hours?/actual_hours?`, `detail_file?` (overflow to a side file), `deliverables[]` `{name, type: file|url|artifact|other, path?, completed?, notes?}`. Statuses: `backlog|todo|in_progress|review|done|blocked|reopened`; priority is symbolic (`urgent|high|medium|low`); IDs are sequential `TASK-NNN` via frontmatter `next_id`. Agents carry `trust level` (`owner|trusted|restricted|read-only`).

**Stale-write mechanism** (`packages/tick-core/src/io/tick-file.ts`): read captures `{mtimeMs, size}`; atomic write calls an assertion — `if (current.mtimeMs !== expected.mtimeMs || current.size !== expected.size) throw new Error("Concurrent modification detected. Re-read TICK.md and retry.")` — then writes a `TICK.md.{pid}.{ts}.tmp` and `rename()`s over. Also refuses to write through a symlinked TICK.md unless `TICK_ALLOW_SYMLINK_WRITE=1`. So: optimistic concurrency on stat identity + atomic rename, cross-machine conflicts left to git merge.

## SYNTHESIS

### Unified task-record field superset

| Field | beads | gnap | ORCH | squad | tick-md |
|---|---|---|---|---|---|
| id | hash `bd-xxx` | `FA-1` seq | `tsk_nanoid7` | GH issue # | `TASK-NNN` seq |
| title / description | title, description, **design, acceptance_criteria, notes** | title, desc | title, description | issue body | title, description, `detail_file` |
| state | 7 statuses incl. `deferred`, `hooked`, `pinned` | 7 incl. `backlog`, `cancelled` | 7 incl. `retrying`, `failed` | issue open/closed + labels | 7 incl. `reopened` |
| priority | int 0–4 | int, 0 highest | int 1–4 | — | enum urgent…low |
| assignee vs claim | `assignee` + atomic `--claim` | `assigned_to[]` only | `assignee`, orchestrator-dispatched | `squad:{name}` label | `assigned_to` **and** `claimed_by` (distinct) |
| lease/TTL | ✅ `lease_expires_at`, `heartbeat_at`, `lease_granted_node` | ✗ (heartbeat_sec is agent-level) | stall/zombie timeouts on runs | ✗ | ✗ |
| deps | typed edges (blocks, parent-child, discovered-from, waits-for…) | `parent` only + `blocked` bool | `depends_on[]` untyped | GH issue links | `depends_on[]` + `blocks[]` |
| parent/goal | `parent-child` edge, epics, molecules | `parent` | `goalId` | — | — |
| evidence/proof | comments, `metadata`, gates | run `result/commits/artifacts` | ✅ `proof {branch, pr_url, files_changed, test_results}` + `review_criteria/results` | PR + decisions.md | `deliverables[]`, `history[]` |
| timestamps | created/updated/started/closed + defer/due | created/updated/due | created/updated + run times | GH-native | created/updated/due + per-event history |
| cost/tokens | — (metadata) | ✅ run `tokens`, `cost_usd` | ✅ run TokenUsage incl. cache | — | `estimated/actual_hours` |
| audit trail | Dolt history + `bd history` | git log + runs/ | runs/*.jsonl | git + log/ + Scribe | inline `history[]` |

### ID schemes
Content-hash base36 with prefix (beads — merge-proof, opaque); sequential human-readable (gnap `FA-1`, tick `TASK-NNN` via `next_id` — readable but a **guaranteed merge conflict** on concurrent create: two branches both consume `next_id`); random nanoid with type prefix (ORCH — collision-safe, unreadable). For a git-native substrate with parallel writers, beads' scheme is the only one that is both collision-free and diff-stable.

### Claim mechanics
beads: DB-atomic `bd update <id> --claim` / `bd ready --claim` (idempotent per-claimer) + node-scoped **leases with heartbeat** for crash recovery + `work_type: open_competition` opt-out. tick-md: `claimed_by` field guarded by single-file stat-check-then-rename (atomic per host, not across clones). GNAP: no claim at all — assignment by a manager agent, git rebase-retry on conflict. ORCH: no self-claim — a single orchestrator process (PID lock) dispatches; mutual exclusion comes from the daemon, not the data. Squad: label-claim on GitHub Issues (server-atomic). These are four distinct points on a spectrum open-seed's `atomic_claim: native|emulated|none` capability flag already anticipates — filecards is honest as `emulated`.

### Recommended open-seed filecards frontmatter

One file per task, `.seed/tasks/{id}.md`, YAML frontmatter + markdown body (body = description; `## Acceptance`, `## Design`, `## Log` sections). Blend: beads' hash IDs/typed deps/lease fields, GNAP's run-cost separation, ORCH's proof/review evidence, tick's claim-vs-assign split:

```yaml
---
id: sd-a1b2c            # {prefix}-{base36(sha256(title|creator|ts|nonce))[:5]} — beads scheme; no next_id counter
title: Implement seed task claim verb
type: task              # task|bug|feature|epic|decision|chore
state: ready            # backlog|ready|in_progress|review|done|blocked|cancelled  (gnap machine + ORCH's review→ready rejection)
priority: 2             # 0-4, 0 critical (beads convention)
created_by: "human:shaun"
created_at: 2026-08-22T00:00:00Z
updated_at: 2026-08-22T00:00:00Z
assigned_to: null       # routing hint (gnap/tick)
claimed_by: null        # actual executor — distinct from assignment (tick)
claim:                  # lease block (beads) — emulated atomicity: write claim, commit, push; retry on reject
  at: null
  lease_expires_at: null   # claim is void past expiry; any agent may reclaim
  heartbeat_at: null
deps:                   # typed edges, beads-style; blocks/parent-child/discovered-from/waits-for/related
  - {on: sd-9zzt4, type: blocks}
parent: sd-epic1        # or goal ref
labels: [backend]
due_at: null
defer_until: null       # hidden from `seed task ready` until then (beads)
review:                 # ORCH evidence gate
  criteria: [test_pass, lint]
  reviewer: null        # squad rule: reviewer ≠ claimed_by; on reject, next claimant ≠ previous author
proof:                  # filled at review time (ORCH TaskProof)
  branch: null
  pr_url: null
  commits: []
history:                # append-only inline event log (tick) — union-merge friendly
  - {ts: ..., who: "human:shaun", action: created}
---
```

Companion conventions: `runs/` NDJSON per attempt carrying `{tokens, cost_usd, commits, artifacts}` (GNAP/ORCH — cost lives on runs, not tasks); `.gitattributes` `merge=union` on history-bearing files plus per-agent inbox dirs merged by a scribe step (squad); commit grammar `<actor>: <verb> <id> [detail]` (GNAP); `.seed/version` plain integer with refuse-if-newer (GNAP); writer discipline = stat-check + tmp-rename (tick) with git push as the cross-clone arbiter, exit 2 on claim contention as already specced.

### Corrections/updates to earlier findings
(1) design-options.md says beads' "storage layer still churning (JSONL→Dolt migration)" — that migration is **complete**; Dolt (embedded by default) is the only backend, JSONL is a passive export, and treating `issues.jsonl` as the sync protocol is now a documented anti-pattern. Pinning a version remains wise (`bd migrate schema`/`bd upgrade` exist because schema still moves — leases landed at migration 0054/0055). (2) Beads now has **native leases + heartbeats and merge-slot/gate primitives**, strengthening the "ticket-claim blackboard" option beyond what the survey recorded. (3) "GitHub's Squad" is community-authored `bradygaster/squad` (MIT) — GitHub-blog-covered but not an official github/ org project; its coordination substrate is GitHub Issues + `.squad/` markdown, with no task-card file format of its own. (4) ORCH's real state machine has `retrying`/`failed` states and `review → todo` rejection — richer than its advertised 4-state diagram — and its "adapters" are compiled-in classes, not user-definable config files, so it's precedent for open-seed's *port interface*, not for checked-in plugin packaging.
