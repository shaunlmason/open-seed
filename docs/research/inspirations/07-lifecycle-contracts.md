# Deep Dive: Checked-in Lifecycle/Worktree Contracts

> Implementation-grade deep dive for the open-seed design study, researched 2026-08-22.
> Every finding read from a fresh shallow clone of each repo's default branch, citing actual
> source files — not READMEs alone.

## 1. superset-sh/superset — `.superset/config.json`

Schema (from `packages/host-service/src/runtime/setup/config.ts`, verbatim types):

```ts
export interface SetupConfig {
  setup?: string[];
  teardown?: string[];
  run?: string[];
  cwd?: string;
}
```

README example:

```json
{
  "setup": ["./.superset/setup.sh"],
  "teardown": ["./.superset/teardown.sh"],
  "run": ["./.superset/run.sh"]
}
```

Key mechanics:

- **Four config layers, per-key "later wins"**: (1) `<repo>/.superset/config.json`, (2) `<worktree>/.superset/config.json` (branch override), (3) `~/.superset/projects/<repoPath>/config.json` (per-machine user override), then (4) a **local overlay** `config.local.json` (worktree's first, else repo's) whose keys are either a replacement array or `{ "before": [...], "after": [...] }` merge — `[...before, ...base, ...after]`.
- **Fallback when no commands configured**: `.superset/<key>.sh` (`setup.sh`/`teardown.sh`/`run.sh`), worktree first then main repo, run as `bash '<path>'`.
- **Invocation semantics**: `setup` commands are joined with `" && "` **so failures short-circuit**; the whole string is an `initialCommand` typed into a real PTY terminal session (login-shell env parity). A chained command (e.g. agent launch) is appended `&& <chainCommand>` so it only runs after setup succeeds. `run` resolves per-project (main repo only). `cwd` from config rides along.
- **Teardown failure handling**: transient hidden PTY, `TEARDOWN_TIMEOUT_MS = 60_000`, 2s SIGKILL grace, result is `ok | skipped | failed {exitCode, signal, timedOut, outputTail}` with a 4096-byte output tail surfaced only on failure.
- **Env**: `SUPERSET_TERMINAL_ID`, `SUPERSET_WORKSPACE_ID`, `SUPERSET_WORKSPACE_PATH`, `SUPERSET_ROOT_PATH` (main repo), `SUPERSET_ENV`, `SUPERSET_AGENT_HOOK_PORT`, `SUPERSET_AGENT_HOOK_VERSION`, `SUPERSET_ORGANIZATION_ID`, `SUPERSET_HOME_DIR`, plus `SUPERSET_WORKSPACE_NAME`.

## 2. ShreyPaharia/octomux — hooks + loop budget

**Correction to survey-level knowledge**: octomux's `.d` hooks are **task-lifecycle events, not worktree-lifecycle events**. The full event list (`server/hook-types.ts`, verbatim):

```ts
export type HookEventName =
  | 'workflow_status_changed' | 'summary_updated' | 'note_added'
  | 'ref_added' | 'ref_removed' | 'task_created' | 'runtime_state_changed';
```

Discovery and execution (`server/hook-dispatcher.ts`):

- Dirs, in order: `~/.octomux/hooks/<event>.d/` (global) then `<repo>/.octomux/hooks/<event>.d/` (repo-local). Within each dir: **executable regular files only, sorted alphabetically**; scripts run **sequentially** (awaited one at a time).
- **Contract per script**: spawned directly (no shell), `cwd` = task worktree if set, else repo path, else `$HOME`. Full `HookEnvelope` JSON (`{event, task, data}`) is written to **stdin**. Env adds only `OCTOMUX_EVENT`, `OCTOMUX_TASK_ID`, `OCTOMUX_HOOK_DIR`.
- **Exit codes are advisory**: non-zero → warn only; `fireHook` "never throws" and is fire-and-forget. Timeout 30s default (`OCTOMUX_HOOK_TIMEOUT_MS`), SIGTERM on timeout. Output logged to `~/.octomux/logs/hooks/` with duration/exit-code footers; 50 logs retained per event+script. Hooks can be individually disabled via a registry (`scope` = `global` or `repo:<path>`).
- `octomux.yml` is **not** repo config — it is the **plugin manifest** at `~/.octomux/octomux.yml`.
- **Loop budget** (`server/types.ts`): `LoopSpec { prompt, verify, maxIterations, budget?: { tokens?, timeMs? }, noProgress?: { afterIters } }`. Termination reasons include `max_iterations | budget | no_progress`. Per-iteration state is checkpointed **into the worktree itself** at `.octomux/loop-status.json`:

```json
{ "loopRunId": "...", "groupId": null, "taskId": "...", "status": "...",
  "iteration": 3, "maxIterations": 10, "terminationReason": null, "updatedAt": "..." }
```

## 3. sahithvibudhi/vibe-tree — `.vibetree/hooks/`

Exactly **two** hooks (`packages/core/src/utils/worktree-hooks.ts`): `post-create` and `pre-remove`. Contract:

- Path: `<mainRepo>/.vibetree/hooks/<name>` — a single executable file, no `.d` dirs, no args. Git-hooks trust model: the executable bit is the opt-in; a hooks dir symlinked outside the project is refused.
- Spawned directly (never through a shell), `cwd` = **worktree**, detached process group; env adds `VIBETREE_HOOK` (hook name), `VIBETREE_PROJECT_PATH`, `VIBETREE_WORKTREE_PATH`, `VIBETREE_BRANCH`. Timeout 120s → SIGKILL of the whole group; combined output capped at 64KiB.
- **Exit codes never block**: post-create failure is "reported, not fatal — the worktree already exists"; pre-remove failure "warns but never blocks removal — the user asked to delete and a broken script should not hold that hostage."

## 4. andyrewlee/amux — `.amux/workspaces.json`, trust, tmux orchestration

Full schema (`internal/process/scripts.go`):

```go
type WorkspaceConfig struct {
    SetupWorkspace []string `json:"setup-workspace"`
    RunScript      string   `json:"run"`
    ArchiveScript  string   `json:"archive"`
}
```

README example:

```json
{
  "setup-workspace": ["npm install", "cp $ROOT_WORKSPACE_PATH/.env.local .env.local"],
  "run": "npm start",
  "archive": "tar -czf archive.tar.gz ."
}
```

- `setup-workspace` runs once at workspace creation; `run` is a toggleable dev-server command; `archive` runs just before worktree deletion (≤2 min, **best-effort** — failure or untrusted repo warns but deletion proceeds; must tolerate running twice).
- **Env** (all three): `AMUX_WORKSPACE_NAME`, `AMUX_WORKSPACE_ROOT`, `AMUX_WORKSPACE_BRANCH`, `ROOT_WORKSPACE_PATH` (main repo), `AMUX_PORT` (allocated per-workspace port), `AMUX_PORT_RANGE` (`start-end`).
- **Trust** (`script_trust.go`): `~/.amux/trusted-scripts.json` is a flat JSON map `{ "<normalized-repo-path>": "<hex sha256 of workspaces.json content>" }`. Fail-closed; **any byte change to the file invalidates approval** and re-gates; a TOCTOU guard binds approval to the reviewed hash. UI-entered run/archive commands are never gated.
- **`docs/ORCHESTRATION.md`** — amux deliberately has **no CLI**; the public seam for external orchestrators is tmux itself, pinned by a contract test: session names `amux-<workspaceID>-<tabPart>` (workspace ID = `hex(sha256(repo+root)[:8])`); discover (never construct) sessions via `tmux -L "${AMUX_TMUX_SERVER:-amux}" list-sessions -F '#{session_name}|#{@amux_workspace}|#{@amux_tab}|#{@amux_type}'`; send input with `send-keys -t '=<exact-name>' -l 'text'` then a **separate** `-H 0D` carriage return after a pause (raw-mode agents drop fused CRs); read state from `@amux_*` session options, notably `@amux_agent_state` ∈ `idle|working|done`.

## 5. standardagents/dmux — `.dmux-hooks/`

Complete hook list (checked-in `.dmux-hooks/README.md` — the repo dogfoods its own contract): `before_pane_create`, `pane_created`, `worktree_created`, `before_pane_close`, `pane_closed`, `before_worktree_remove`, `worktree_removed`, `pre_merge`, `post_merge`, `run_test`, `run_dev`.

Script contract (from the auto-generated `.dmux-hooks/AGENTS.md` + `src/utils/hooks.ts`):

- Resolution priority: `.dmux-hooks/<name>` (version-controlled, team) → `.dmux/hooks/<name>` → `~/.dmux/hooks/<name>`. Single executable file per hook, no args.
- **Non-blocking by design**: spawned `detached: true` + `unref()`; "hook errors are logged but don't stop dmux". Exit codes are therefore not gating — even `pre_merge` cannot veto a merge.
- Env: always `DMUX_ROOT`, `DMUX_SERVER_PORT`; pane context `DMUX_PANE_ID`, `DMUX_SLUG`, `DMUX_PROMPT`, `DMUX_AGENT`, `DMUX_TMUX_PANE_ID`; worktree context `DMUX_WORKTREE_PATH`, `DMUX_BRANCH`; merge context `DMUX_TARGET_BRANCH`.
- **Feedback channel is HTTP, not exit codes**: `run_test`/`run_dev` PUT to `http://localhost:$DMUX_SERVER_PORT/api/panes/$DMUX_PANE_ID/test` (`{"status": "running"|"passed"|"failed", "output": ...}`) and `/dev` (`{"status": "running"|"stopped", "url": ...}`).
- **Smart merge**: `pre_merge` fires, auto-commit + merge executes, `post_merge` fires on success; on conflict, dmux spawns a **new pane running an AI agent** dedicated to resolving the conflict, rather than failing the merge.

## 6. asheshgoplani/agent-deck

- **`.agent-deck/worktree-setup.sh`**: runs automatically after worktree creation **and after** `.worktreeinclude` processing. Env: `AGENT_DECK_REPO_ROOT`, `AGENT_DECK_WORKTREE_PATH`. Runs via `sh -e`, 60s timeout; failure → warning, session proceeds. Symmetric **`.agent-deck/worktree-destruction.sh`** runs just before removal, same contract, failure never blocks removal.
- **`.worktreeinclude`** (repo root), verbatim from README:

```gitignore
# .worktreeinclude — gitignore-syntax patterns
.env
.env.local
.mcp.json
secrets/
```

Semantics: gitignore syntax; a file is copied only if it is **both pattern-matched AND gitignored** (tracked files never duplicated); directories copied recursively and merged; existing files never overwritten; explicitly "matches Claude Code Desktop semantics". Sparse-checkout inheritance runs before it.
- **`.agent-deck/skills.toml`**: project-local skill-attachment state, TOML `[[skills]]` with fields `id`, `name`, `source`, `source_path`, `entry_name`, `target_path` (e.g. under `.claude/skills/`), `mode`, `attached_at`.

## 7. wavyrai/tmux-ide

**`.tmux-ide/workspace.yml`**: top-level `version: 1` (required), `name` (tmux session name), `before` (pre-launch shell hook), `terminal`, `app`, `harnesses`, `agents`, `missions`. `terminal.rows[]`: `size` (percent height) + `panes[]`. Pane fields: `title`, `command`, `size` (width %), `dir`, `focus` (bool), `env` (map), `type` (widget: `explorer|changes|preview|setup|config|sidebar`), `target`. Verbatim minimal example:

```yaml
version: 1
name: my-app
terminal:
  rows:
    - size: 70%
      panes:
        - title: Claude
          command: claude
          focus: true
        - title: Shell
    - panes:
        - title: Dev Server
          command: pnpm dev
```

**Agent-status protocol**: any agent self-reports by stamping its own tmux pane option:

```bash
tmux set-option -p @agent_state "working:$(date +%s)"
```

Value grammar: `<state>:<unix epoch>` with state ∈ `working | blocked | done | idle`. A `working`/`blocked` report **older than 10 minutes is stale** and the detector falls back to process-tree + screen classification; a fresh `@agent_state` always wins. Also `@agent_session_id` for Claude resume.

**`wait`**: `tmux-ide wait output <pane|session> --match <regex> [--timeout <ms>]` and `tmux-ide wait agent-status <session> --status <s> [--timeout <ms>]`; **exit 0 = matched, 1 = timeout**. `tmux-ide send <target>` types into another agent's prompt (>~150 chars auto-spills to a dispatch file).

## 8. ouijit/ouijit (brief)

Hooks are per-project **DB records, not files**, set via CLI: `ouijit hook set <type> --name "..." --command "..."`; types: `start, continue, run, review, done, editor`. Task env passed to hook PTYs: `OUIJIT_HOOK_TYPE`, `OUIJIT_PROJECT_PATH`, `OUIJIT_WORKTREE_PATH`, `OUIJIT_TASK_BRANCH`, `OUIJIT_TASK_NAME`, `OUIJIT_TASK_DESCRIPTION` (shell-safe transformed), and `OUIJIT_TASK_PROMPT` (a *deprecated alias*). Canonical idiom: `ouijit hook set start --command 'claude "$OUIJIT_TASK_DESCRIPTION"'`.

## 9. johannesjo/parallel-code (brief)

`.claude/steps.json` is an **agent-maintained progress file**, prompted via an injected instruction: one JSON array of entry objects. Fields: `summary` (≤60 chars, present tense), `detail` (optional single sentence), `status` ∈ `starting | investigating | implementing | testing | awaiting_review | done`, `files_touched` (written files only), `agent_id` (sub-agent label, omit for self), `timestamp` (**host-stamped**: the watcher overwrites timestamps on new entries with the host clock, distrusting the AI's clock, and excludes the file via git info/exclude). `awaiting_review` + pause is the human-gate signal.

---

# Synthesis: proposed `.seed/hooks/` contract

**Layout** (checked in):

```
.seed/
  hooks/
    setup            # single executable; runs once after worktree create + .worktreeinclude copy
    run              # dev-server / long-running command (start/stop toggled by tool)
    teardown         # just before worktree removal, while it still exists
    post-create.d/   # run-parts extras after `setup`, alphabetical
    pre-merge.d/     # gates before merging task branch → target, alphabetical
.worktreeinclude     # repo root (where agent-deck and Claude Code look)
```

**Execution contract** (the intersection every tool above can honor):

- Hooks are executable regular files; **executable bit = opt-in** (vibe-tree/octomux model); spawned directly, not through a shell (scripts choose their own shebang); no argv — context via env only, `cwd` = the worktree (repo root for `pre-merge.d` when merging from the main checkout).
- `.d` dirs: executable files sorted alphabetically (`10-lint`, `20-test`), run sequentially — octomux's semantics.
- **Exit codes**: `setup`/`post-create.d`/`teardown` are *advisory* — non-zero warns, never strands or blocks a worktree (unanimous across vibe-tree, dmux, amux archive, agent-deck). `pre-merge.d` is the one *gating* set: any non-zero exit aborts the merge (this is the niche none of the surveyed tools fills — dmux's `pre_merge` is detached/non-blocking, so open-seed adds real value here). Timeouts: 120s per hook default (vibe-tree's number), overridable via `SEED_HOOK_TIMEOUT_MS`.
- **Env-var set** (superset of every tool's mapping needs):

```
SEED_EVENT              # setup | run | teardown | post-create | pre-merge
SEED_REPO_ROOT          # main repository path
SEED_WORKTREE           # worktree path
SEED_BRANCH             # task branch
SEED_TARGET_BRANCH      # merge target (pre-merge.d only)
SEED_TASK_ID            # stable task identifier
SEED_TASK_TITLE         # short label
SEED_TASK_DESCRIPTION   # full prompt/description
SEED_AGENT              # harness name (claude|codex|opencode|...), optional
SEED_PORT               # allocated per-task port
SEED_PORT_RANGE         # "start-end"
```

- **`.worktreeinclude`**: adopt agent-deck's format verbatim — gitignore-syntax patterns, copy only files that are matched AND gitignored, recursive dir merge, never overwrite — because it is already a de-facto standard ("matches Claude Code Desktop semantics"). Repo root, not inside `.seed/`.
- Optional status file: agents may write `.seed/steps.json` (parallel-code's array schema) and orchestrators may stamp `@agent_state "<working|blocked|done|idle>:<epoch>"` on their tmux pane (tmux-ide grammar, compatible with amux's `@amux_agent_state` values).

**Per-tool shims** (each is a 2–5 line checked-in file):

- **superset** — `.superset/config.json`: `{"setup": [".seed/hooks/setup && run-parts .seed/hooks/post-create.d"], "teardown": [".seed/hooks/teardown"], "run": [".seed/hooks/run"]}`. Map env in the scripts: `SEED_REPO_ROOT=${SEED_REPO_ROOT:-$SUPERSET_ROOT_PATH}`, `SEED_WORKTREE=${SEED_WORKTREE:-$SUPERSET_WORKSPACE_PATH}`.
- **amux** — `.amux/workspaces.json`: `{"setup-workspace": [".seed/hooks/setup"], "run": ".seed/hooks/run", "archive": ".seed/hooks/teardown"}`; shim env `SEED_WORKTREE=$AMUX_WORKSPACE_ROOT`, `SEED_BRANCH=$AMUX_WORKSPACE_BRANCH`, `SEED_PORT=$AMUX_PORT`. Caveat: editing the shim re-triggers amux's trust prompt (content-hash), so keep it byte-stable and put logic in `.seed/hooks/`.
- **vibe-tree** — `.vibetree/hooks/post-create` → `exec "$VIBETREE_PROJECT_PATH/.seed/hooks/setup"`; `pre-remove` → teardown. (vibe-tree does not copy gitignored files — setup must handle `.worktreeinclude` itself, so ship a portable `seed-copy-include` helper.)
- **dmux** — `.dmux-hooks/worktree_created` → setup, `before_worktree_remove` → teardown, `pre_merge` → run `pre-merge.d` (but dmux won't honor a failing exit — document that the gate is advisory under dmux), `run_dev` → `.seed/hooks/run` + PUT status callback. Env map: `DMUX_ROOT→SEED_REPO_ROOT`, `DMUX_WORKTREE_PATH→SEED_WORKTREE`, `DMUX_BRANCH→SEED_BRANCH`, `DMUX_TARGET_BRANCH→SEED_TARGET_BRANCH`, `DMUX_PROMPT→SEED_TASK_DESCRIPTION`.
- **agent-deck** — `.agent-deck/worktree-setup.sh` → `exec "$AGENT_DECK_REPO_ROOT/.seed/hooks/setup"`; `worktree-destruction.sh` → teardown; `.worktreeinclude` needs **no shim** (identical format). Keep scripts `sh -e`-safe and under 60s or background long work.
- **octomux** — `.octomux/hooks/task_created.d/10-seed` and `runtime_state_changed.d/10-seed` shims that read the JSON envelope from stdin (`jq -r .task.worktree`) and export SEED_* before dispatching; octomux has no worktree-create event, so setup should be idempotently runnable from `task_created`.
- **tmux-ide / amux orchestration** — no hook shim needed; open-seed's docs should tell orchestrators to (a) stamp `@agent_state`, (b) use `tmux-ide wait output/agent-status` (exit 0/1) as the wait primitive, and (c) follow amux's send-keys rules (exact-match `=` target, separate literal CR).
- **ouijit** — one-time: `ouijit hook set start --command '"$OUIJIT_PROJECT_PATH/.seed/hooks/setup"'`; inside, map `SEED_TASK_DESCRIPTION=$OUIJIT_TASK_DESCRIPTION`, etc.
- **parallel-code** — no lifecycle hooks exist; compatibility is via the progress file: agents write `.claude/steps.json` in its documented array schema (or symlink `.seed/steps.json` → `.claude/steps.json`).

**Corrections to earlier survey findings**: (1) octomux's `.d` hooks fire on *task* events, not worktree create/merge, and pass context via a **stdin JSON envelope** plus only 3 env vars — a design open-seed should optionally support (`--stdin-envelope`) but not require; `octomux.yml` is a user-level plugin manifest, not repo config. (2) dmux's `pre_merge` cannot actually block a merge (detached spawn) — any survey claim of a merge gate there is wrong; open-seed's blocking `pre-merge.d` is genuinely novel among these tools. (3) superset's `.superset/config.json` has no per-hook timeout/failure keys — failure handling is positional (`&&` chaining for setup, 60s cap for teardown) — and it additionally supports `config.local.json` before/after overlays worth borrowing for per-developer customization.
