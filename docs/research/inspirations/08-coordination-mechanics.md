# Deep Dive: Coordination Protocol Mechanics

> Implementation-grade deep dive for the open-seed design study, researched 2026-08-22.
> Covers swarm-protocol, wit, hcom, multi-agent-shogun, claude_codex_bridge, and claudexor.
> All six repos cloned and read at source level.

## 1. phuryn/swarm-protocol

**The checked-in half** is a single drop-in snippet, `claude-md/COORDINATION.md`, quoted in full:

```markdown
## Team Coordination (Swarm Protocol)
This project uses Swarm Protocol for team coordination.
MCP server: mcp://localhost:3333/swarm-protocol

### Before starting any work:
1. Call `get_team_status` for your team to see what's in flight
2. If picking up an intent, call `claim_work` with the files you expect to touch
3. Call `check_conflicts` with your file list to verify no collisions

### While working:
- Call `heartbeat` every 10-15 minutes with updated file list
- If blocked, call `send_signal` with type "blocked" and explain what you need

### After completing work:
1. Call `complete_claim` with a summary of what was done
2. If your work unblocks other intents, include them in `unblocks`

### If creating new work:
- Call `create_intent` to draft new work items
- Refine description, constraints, and acceptance criteria
- Call `publish_intent` when ready for someone to pick it up
- Use `decompose_intent` to break large intents into sub-intents
```

**The 19 MCP tools** (the SPEC lists 18 — `get_board` is the undocumented 19th): create_team, list_teams, get_team_status, get_overview, get_board | create_intent, publish_intent (`draft`→`open`; validates non-empty acceptance_criteria), list_intents, get_intent, update_intent, decompose_intent (children get `parent_id`; parent stays open until children done) | claim_work (`intent_id, claimed_by, files_touching?[], branch?` — fails unless intent is `open`, returns conflict warnings, sets intent `claimed`), heartbeat, release_claim (claim→`abandoned`, intent→`open`), complete_claim (`message?, unblocks?[]` — auto-creates a completion signal; dependents whose deps are all met flip to `open`) | check_conflicts | send_signal, get_signals | get_context.

**Schemas** (abridged to load-bearing fields):

```json
// Intent
{ "id": "intent_abc123", "title": "...", "status": "draft|open|claimed|blocked|done|cancelled",
  "priority": "critical|high|medium|low", "parent_id": null, "depends_on": ["intent_xyz"],
  "context": "...", "constraints": [...], "acceptance_criteria": [...],
  "files_likely_touched": ["src/middleware/", "src/api/"] }
// Claim
{ "id": "claim_def456", "intent_id": "...", "claimed_by": "pawel", "agent_session": "cc_sess_789",
  "files_touching": ["src/middleware/rateLimit.ts"], "branch": "feat/rate-limiting",
  "status": "active|paused|completed|abandoned", "last_heartbeat": "..." }
// Signal
{ "type": "completion|blocked|conflict|info|request", "from": "...", "intent_id": "...",
  "claim_id": "...", "message": "...", "unblocks": ["intent_xyz"] }
```

The **Context Package** (`get_context`) is assembled per call: intent + parent + dependency intents/statuses + active claims on overlapping files + last 10 signals + team conventions (a free-form `conventions` TEXT column on `teams`).

**Heartbeat mechanics**: tool description verbatim — "Call every 10-15 minutes. Claims with no heartbeat for 30 min get flagged as stale." Staleness is *only* a query. Nothing expires or releases the claim; heartbeat may also update `files_touching`.

**check_conflicts semantics**: active claims whose `files_touching` JSONB array has **exact string equality** with any input path — no prefix/glob matching, so `src/api/` in a claim will not conflict with `src/api/router.ts`. Purely advisory.

## 2. amaar-mc/wit

Bun/TypeScript daemon, Unix socket `.wit/daemon.sock`, **JSON-RPC 2.0 + `"witVersion": "1"`** over HTTP POST `/rpc` (docs/PROTOCOL.md is a full spec; docs/openrpc.json mirrors it). 12 methods: `ping, register, lock.acquire, lock.release, lock.query, intent.declare, intent.update, intent.query, contract.propose, contract.respond, contract.query, check-contracts`.

**Symbol locks**: `symbolPath` = `<relative-file-path>:<symbol-name>` (`.ts/.tsx/.js/.jsx/.py`). `lock.acquire` is exclusive with TTL (default 30 min; same-session acquire refreshes; expired locks are taken over). On acquire, the daemon **Tree-sitter-parses the file and builds a caller graph**; the result carries `warnings: CallerWarning[]` — informational, never blocking. Conflict is JSON-RPC error `-32000 LOCK_CONFLICT` with `data: { heldBy, expiresAt }`. Note: no lock heartbeat — TTL only.

**Intent**: `intent.declare(sessionId, description, files[], symbols?[])` always succeeds and returns a `ConflictReport` with three item kinds: `INTENT_OVERLAP` (same file + overlapping **byte range**), `LOCK_INTERSECTION`, `DEP_CHAIN` (a *callee* of your declared symbol is locked). Lifecycle forward-only: `declared → active → resolved|abandoned`.

**Contracts**: `contract.propose(sessionId, symbolPath)` — daemon reads the signature from disk via tree-sitter, e.g. `"signature": "(token: string): Promise<User | null>"`. A *different* session must `contract.respond(accept)` (`SELF_ACCEPT_NOT_ALLOWED`). **Pre-commit enforcement** — hook written verbatim by `wit hook install`:

```sh
#!/bin/sh
# Managed by wit. Do not edit -- run `wit hook install` to regenerate.
STAGED_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep -E '\.(ts|tsx|py)$')
if [ -z "$STAGED_FILES" ]; then exit 0; fi
REPO_ROOT=$(git rev-parse --show-toplevel)
echo "$STAGED_FILES" | xargs bun run --cwd "$REPO_ROOT" wit check-contracts
exit $?
```

`wit check-contracts` reads **staged** content via `git show`, POSTs `{files: [{path, content}]}`; violations report `{contractId, symbolPath, expected, actual}`. Per PROTOCOL.md: "The hook uses a 2-second timeout and **fails open** — if the daemon is unreachable, the commit proceeds."

**Git trailer**: a `prepare-commit-msg` hook calls `wit _active-intents "$SESSION_ID"` (500ms timeout, silent exit if no daemon) and appends one trailer per active intent via `git interpret-trailers`. Exact trailer name: **`Wit-Intent`**.

**Auto-generated CLAUDE.md** (`wit init`): "Multiple AI agents may be working on this codebase simultaneously. **You MUST follow this protocol**" — session identity via `export WIT_SESSION="agent-$(openssl rand -hex 8)"`; then `wit status` → `wit declare` → `wit lock --symbol "file:fn"` before editing; "**ALWAYS re-read the file immediately before editing it**"; on `LOCK_CONFLICT`: "Do NOT edit it — work on other unlocked symbols first, then retry"; "Lock at the symbol level, not the file level"; "Do NOT end your task with unfinished work due to lock conflicts. Keep working."

## 3. aannoo/hcom

Now a Rust workspace (correction to earlier survey: it was billed as a single-file Python tool; `pyproject.toml` remains only as an installer shim). Integrates 11 harnesses. Data plane: "agent → hooks → db → hooks → other agent" — one SQLite DB under `~/.hcom` (or `$HCOM_DIR` for per-project isolation).

**Hook installation, Claude Code** (src/hooks/claude.rs, verbatim):

```rust
const CLAUDE_HOOK_CONFIGS: &[(&str, &str, &str, Option<u64>)] = &[
    ("SessionStart", "", "sessionstart", None),
    ("UserPromptSubmit", "", "userpromptsubmit", None),
    ("PreToolUse", "Bash|PowerShell|Agent|Task|Write|Edit", "pre", None),
    ("PostToolUse", "", "post", Some(86400)),
    ("PostToolUseFailure", "", "post-failure", None),
    ("Stop", "", "poll", Some(86400)),
    ("StopFailure", "", "stop-failure", None),
    ("PermissionRequest", "", "permission-request", None),
    ("PermissionDenied", "", "permission-denied", None),
    ("SubagentStart", "", "subagent-start", None),
    ("SubagentStop", "", "subagent-stop", Some(86400)),
    ("Notification", "", "notify", None),
    ("SessionEnd", "", "sessionend", None),
];
```

written into `~/.claude/settings.json` with a fail-silent command wrapper, plus auto-approved permission rules like `Bash(hcom send:*)`. Other harnesses: Gemini/Antigravity use their hook events; Codex uses its native hooks; OpenCode/Kilo/Pi/OMP get a bundled TS plugin; anything else joins via `hcom start` + `hcom listen`.

**Message delivery mid-turn**: pending messages are delivered from the **PostToolUse** hook's JSON output — model-facing text rides `hookSpecificOutput.additionalContext`, human-facing rides `systemMessage`. So a message lands *between two tool calls* of an active turn. When idle, the **Stop hook blocks**: it polls the DB (up to `wait_timeout`), and delivers via exit 0 + `decision: "block"` with the messages as the block reason (code comment: "Claude ignores stdout JSON on exit 2 for Stop, so a delivered message must go out as exit 0 + decision:block"). Delivery is acked only after stdout flush (at-least-once).

**Collision detection**: "if two agents edit the same file within 30 seconds, both get notified." Mechanics: every Write/Edit hook logs a `status` event with the file path; on each such event, subscribers with the `collision` filter run a SQL check for another instance's event on the same file within 30s; a match sends a system message from `[hcom-events]` addressed `@<agent>` and wakes only that agent. Opt-in via `auto_subscribe` presets: `collision`, `created`, `stopped`, `blocked`.

**SQLite events schema**: `events(id INTEGER PRIMARY KEY AUTOINCREMENT, timestamp TEXT, type TEXT, instance TEXT, data TEXT)` — append-only; `instances` carries per-agent cursor `last_event_id`, status fields, transcript path, parent linkage. **CLI**: `hcom [N] claude|codex|gemini|...` (launch N), `hcom send -b @name -- msg`, `hcom list`, `hcom events --wait`, `hcom kill`, TUI via bare `hcom`. Agents learn commands from a ~700-token primer injected at launch.

## 4. yohey-w/multi-agent-shogun

tmux + files + watchers. Roles: Shogun (Opus, strategy) → Karo (Sonnet, decomposition/dispatch) → Gunshi (Opus, L4–L6 analysis + QC + dashboard) → Ashigaru 1–7 (implementation). Three layers: tmux panes, `lib/` scripts, YAML queue (`queue/shogun_to_karo.yaml`, `queue/tasks/`, `queue/reports/`, `queue/inbox/`).

**Task file format** — one file per agent, `queue/tasks/ashigaru{N}.yaml` (verbatim):

```yaml
task:
  task_id: subtask_001
  parent_cmd: cmd_001
  bloom_level: L3        # L1-L3=Ashigaru, L4-L6=Gunshi
  description: "Create hello1.md with content 'おはよう1'"
  target_path: "hello1.md"
  echo_message: "🔥 足軽1号、先陣を切って参る!"
  status: assigned
  timestamp: "2026-01-25T12:00:00"
# Dependent task:
task:
  task_id: subtask_003
  blocked_by: [subtask_001, subtask_002]
  status: blocked         # Initial status when blocked_by exists
```

Shogun→Karo cmds carry `id, timestamp, north_star, purpose, acceptance_criteria[], command: |, project, priority, status: pending`. Karo checks every acceptance criterion before marking a cmd done; when a task completes it scans all task YAMLs for `blocked_by` containing the finished id and unblocks.

**Inbox + inotifywait flow**: messages are appended by `scripts/inbox_write.sh <target> <content> <type> <from>` into `queue/inbox/<agent>.yaml` under a `mkdir`-lock + `flock` combo, atomic tmp+rename, schema `{id: msg_YYYYMMDD_HHMMSS_<4hex>, from, timestamp, type, content, read: false}`, capped at 50 (all unread + newest 30 read). One `inbox_watcher.sh` per agent runs `inotifywait` (fswatch on macOS; 30s timeout fallback for WSL2).

**The nudge**: message content never crosses tmux — only a wake signal. Exact mechanics: `local nudge="inbox${unread_count}"`, then

```bash
tmux send-keys -t "$PANE_TARGET" C-u     # clear line
tmux send-keys -t "$PANE_TARGET" "$nudge"
tmux send-keys -t "$PANE_TARGET" Enter    # text and Enter sent separately (Codex TUI workaround)
```

with retries, verification, a 60s cooldown (300s for Codex), busy-suppression via idle flag files (maintained by a Stop hook), and escalation: 0–2 min normal nudge → 2–4 min Escape×2+Ctrl-C for some harnesses → 4 min+ send `/clear` (max once per 5 min) followed by a full recovery prompt ("Session Start — do ALL of this in one turn... Read queue/tasks/${AGENT_ID}.yaml. Read queue/inbox/${AGENT_ID}.yaml, mark read:true..."). The README's rationale table: "Message content stays in files — only a short 'you have mail' nudge is sent through tmux"; "flock … serializes writes"; "inotifywait is event-driven (zero CPU while idle)".

**Role prompts**: ashigaru.md is YAML front matter with numbered forbidden actions (`F001 direct_shogun_report`, `F004 polling — "Wastes API credits"`, `F005 skip_context_reading`) and an 11-step workflow: read own task YAML only → `status: in_progress` → execute → write report YAML → `status: done` → `inbox_write` **to gunshi** (mandatory) → check own inbox before idling; per-file `race_condition: {rule: "No concurrent writes to same file by multiple ashigaru", action_if_conflict: blocked}`. Karo's key patterns: "Don't: Forward shogun's instruction verbatim. Doing so is Karo's failure of duty," and "**Wake = Full Scan**: Claude Code cannot 'wait'... Scan ALL report files (not just the reporting one)"; "After dispatching all subtasks: STOP. Do not launch background monitors or sleep loops." **Results flow up**: ashigaru report YAML → gunshi QC → gunshi report + dashboard aggregation → inbox to karo (final judgment, unblocking) → shogun → user.

## 5. SeemSeam/claude_codex_bridge (CCB)

**`.ccb/ccb_memory.md`**: created only if missing (atomic create-if-missing; seed hash recorded so a user-edited file is never overwritten; template upgrades apply only when the hash still matches). Default template verbatim:

```md
# CCB Project Memory
This project uses CCB for visible multi-agent collaboration.
## Collaboration
- You are one agent in a CCB-managed project team.
- Use CCB `ask` for project-level collaboration with configured agents.
- Delegate with the goal, scope/files, assumptions, expected output, and verification needs.
- Reply concisely with findings, changes, verification, blockers, and risks when relevant.
## Ask Communication
Preferred form:  /ask <agent> <message>
- Submit once, then stop. Do not wait, poll, or run `pend`/`watch`/`ping` unless diagnostics were requested.
- During an active CCB ask task, use `ask --chain` when a child result is needed to finish the
  current task; use `ask --silence` only for independent no-result-needed work.
- Plain nested `ask` from an active task is rejected by CCB.
```

Each agent gets a generated "memory bundle" — one markdown doc with explicit source sections (CCB runtime rules → shared project memory → provider-native memory → agent-private memory).

**Collaboration graph**: no static graph config. Graphs (`A -> B -> C`, `A,B -> C`) are *emergent* from chained asks; `ask --chain` (result feeds parent) vs `ask --silence` (fire-and-forget), plain nested ask rejected (loop prevention). Topology is declared only as tmux layout in `.ccb/ccb.config`: `[windows] work = "worker1:codex(worktree), worker2:claude(worktree)"`.

**Circuit breaker** (pane recovery): a respawned pane sits in a **90-second probation window** with queued work held until a healthy observation; unstable recovery backs off **30s/60s/120s/5m/10m/30m, then opens a circuit after six attempts**, surfacing `health=recovery-circuit-open`; only explicit restart/remount clears the counter.

## 6. razzant/claudexor

**Continuation packet** (`packages/orchestrator/src/continuity.ts` — pure builder): a *lane* is `(thread, harness, profile)`; when a lane's checkpoint lags the thread head (lane switch A→B→A, or a gap), the engine writes `context/THREAD.md` and discloses a typed `session.continuity` event; a plain in-lane turn resumes natively with **no packet**. Budgets verbatim:

```ts
export const PER_TURN_BUDGET_BYTES = 8 * 1024;   // per delta turn (prompt + primary output)
export const TOTAL_BUDGET_BYTES = 24 * 1024;     // whole packet; older turns collapse past this
export const COLLAPSE_PREFIX_CHARS = 200;        // one-line collapsed entry keeps this many chars
```

Rendered format: `# Thread continuation packet` + preamble "You are continuing an existing conversation on a new lane. The earlier turns below did not run on this lane's native session — read them before answering the new prompt."; then either `## Earlier conversation (summary)` (cached LLM summary; stale-boundary summaries rejected) + `## Recent turns`, or `## Earlier turns` with mechanical one-liners as the always-available fallback ("a missing/failed summary never loses information"); verbatim turns render as `### Turn N / **User asked:** ... / **The assistant answered:** ...`; then `## Active plan` and `## Workspace anchor` (`HEAD <sha> · N file(s) with uncommitted changes at the start of this turn.`). Continuity kind is disclosed on the turn: `native_resume | packet | fresh`.

**`.claudexor/config.yaml`** ("safe, versioned settings only"; sensitive settings live in `~/.claudexor/v3/trust/<repo-hash>.yaml`): `constraints.protected_paths` is an array of canonical repo-relative globs (no absolute roots, no `..`) — restriction-only: "matching changes require a human decision before apply". **Test gate exact-argv**: `tests.commands` = `{program (executable, no implicit shell), args: [exact argv], cwd?, envAllowlist: []}`; execution additionally requires a user-level `TestCommandGrant` pinning digests of the project config, the command, and *the resolved executable bytes* — a checked-in config can never smuggle a new binary past the operator.

**Best-of-N race + reviewer**: `--n N` gives each candidate its own isolated worktree; pipeline = reserve budget → run harness → capture diff → deterministic gates → review/revalidate findings → optionally synthesize a merged candidate → arbitrate → auto-apply winner. Protected-path hits and an all-refused race are terminal, not auto-applied.

---

# SYNTHESIS for open-seed

## (a) Inter-agent messaging in v1: yes, but minimal and mailbox-shaped

**Ship it, as files, without a daemon.** Every system that works without central infrastructure converges on one-file-per-recipient + optional nudge (shogun's inbox YAML, gnap's `messages/*.json`); the systems that inject mid-turn (hcom) need per-harness hook surgery that a checked-in template cannot assume. What v1 must *not* include: delivery daemons, blocking Stop-hook polls, or tmux escalation ladders. The mailbox is read at natural checkpoints (task start, pre-commit, post-task); hcom is the upgrade path for live injection.

Proposed format — one file per recipient, append-only, merging shogun's schema (lock + read-flag + cap) with gnap's fields (broadcast, thread, typed):

```yaml
# .seed/mail/<agent-id>.yaml  — one file per recipient; writers append under flock
messages:
  - id: msg_20260822_141502_a3f9
    from: agent-redwood            # sender agent id (self-send rejected)
    at: "2026-08-22T14:15:02Z"
    type: info                     # directive | status | request | info | alert | handoff
    task: T-014                    # optional: task-card id this concerns
    thread: msg_20260822_1401_...  # optional: reply-to
    read: false
    text: |
      Renamed validateToken -> verifyToken in src/auth.ts; update your imports before merging.
```

Conventions to adopt verbatim: writer-side `flock` + tmp-file + rename (shogun); overflow cap "keep all unread + newest 30 read"; broadcast via a `_all.yaml` file or gnap's `to: ["*"]`; **nudge-optional** — a `seed mail nudge <agent>` port that, when tmux is present, does exactly shogun's send-keys sequence and is a no-op otherwise. The CLI port gets `seed mail send/read/ack`, and AGENTS.md gets a swarm-protocol-style checklist ("check your mailbox before starting and after finishing a task").

## (b) Advisory conflict detection: claim-time file scope + pre-edit check, wit-style contracts as the escalation

Three tiers, matching cost to risk:

**Tier 1 — declare at claim time, in the task card.** Follow swarm-protocol's Claim shape but keep it in the card:

```yaml
# task card fragment
claim:
  by: agent-redwood
  branch: task/T-014-rate-limit
  worktree: .worktrees/T-014
  files:                      # glob-capable, unlike swarm-protocol's exact-match
    - src/middleware/**
    - src/api/v1/router.ts
  claimed_at: "2026-08-22T14:00:00Z"
  renewed_at: "2026-08-22T14:12:00Z"   # heartbeat: renew every 10-15 min; stale after 30
```

Adopt swarm-protocol's numbers verbatim (renew 10–15 min, flag stale at 30) but make staleness a *report*, never an auto-release. Fix their bug: match by glob/prefix, not string equality.

**Tier 2 — pre-edit check script.** `scripts/check-conflicts <file...>`: scans all open task cards' `claim.files`, prints wit-style advisory output (card id, owner, overlapping globs), exit 0 always. Wire it two ways: as a `PreToolUse` hook on `Write|Edit` for Claude Code users, and as advice in AGENTS.md for other harnesses. Pair with wit's cheapest, most transferable rule, quoted into open-seed's AGENTS.md: "ALWAYS re-read the file immediately before editing it. Another agent may have modified it since you last read it."

**When to escalate to wit-style hard contracts**: only when (1) two claims *knowingly* overlap on a shared interface, and (2) the boundary is a signature, not file contents. Then record it in the card (`contracts: [{symbol: "src/auth.ts:verifyToken", signature: "(token: string): Promise<User|null>", agreed_by: [T-014, T-015]}]`) and enforce at pre-commit with a grep/tree-sitter check of the staged signature. Copy wit's enforcement posture exactly: check **staged content** (`git show :file`), **2-second timeout, fail open**, and stamp provenance with a trailer (`Seed-Task: T-014`, mirroring `Wit-Intent`). Do not adopt symbol *locks*: they require a daemon and TTL bookkeeping — worktree-per-task already removes the write-write race that locks solve.

## (c) Continuation/handoff packet

Claudexor proves the packet should be **mechanical-first, bounded, and disclosed**; guild-style Briefs and CCB's memory template supply the forward-looking half (goal/scope/verification) that claudexor's backward-looking transcript delta lacks. Proposed `handoff/<task-id>.md`, written by the outgoing agent (or generated by the CLI port from card + git):

```markdown
# Continuation packet — T-014
> From: agent-redwood (claude) · To: (any) · 2026-08-22T15:04Z
> Read this before acting; your session has no native memory of prior turns.

## Task
Card: .seed/tasks/T-014.yaml — goal, acceptance criteria, and claimed file scope live there. Re-read it.

## State of work
- DONE: rate-limit middleware in src/middleware/rateLimit.ts; unit tests pass.
- IN PROGRESS: wiring into src/api/v1/router.ts — half-applied, see dirty files below.
- NOT STARTED: 429 response headers; dashboard criterion.

## Decisions & constraints (do not re-litigate)
- Redis token bucket, not in-memory (card constraint).
- verifyToken signature is contracted with T-015: (token: string): Promise<User | null>.

## Recent turns (delta since last handoff)
### Turn 1
**User asked:** ...
**The assistant answered:** ...   <!-- verbatim, ≤8KB/turn; collapse oldest to one-liners past ~24KB -->

## Workspace anchor
HEAD 4f2a91c · 3 file(s) with uncommitted changes: src/api/v1/router.ts, ...

## Next step
Finish router wiring; run the test gate: `pnpm vitest run --dir src/middleware`.
```

Adopt from claudexor verbatim: the budgets (8 KB/turn, ~24 KB total, 200-char collapse lines), the "no packet needed" rule (same agent resuming its own native session skips the packet), the workspace anchor line format, and the disclosure principle (the card records `continuity: packet|native_resume|fresh` per handoff so failures are debuggable). Skip the cached-LLM-summary layer for v1 — mechanical one-liners are claudexor's own guaranteed fallback. The `handoff` message type in the mailbox carries just the pointer.

## Corrections to earlier survey-level findings

1. **hcom is no longer a single Python file** — it is a full Rust workspace (~150 source files) with a PTY layer, relay/E2E-crypto multi-device mode, and 11 harness integrations. 2. **swarm-protocol's SPEC documents 18 tools; the code registers 19.** Its conflict matching is exact-string, weaker than "file overlap detection" implies, and heartbeat/staleness is query-only — nothing enforces the cadence. 3. **wit has no heartbeat at all** — locks are TTL-based (30 min default, takeover on expiry); claim-renewal belongs to swarm-protocol. 4. **shogun's "you have mail" nudge is literally the string `inbox<N>`** typed into the pane — not a prose message — and ashigaru now report to **Gunshi** (QC) rather than Karo. 5. **CCB's A→B→C graphs are not a config format** — they are runtime ask-chains, and its circuit breaker guards pane *recovery*, not messaging. 6. **claudexor's packet is per-lane, not per-run**: same-harness continuation uses native session resume with zero packet; the packet exists only for cross-harness/gap turns — precisely the case open-seed's handoff format needs to cover.
