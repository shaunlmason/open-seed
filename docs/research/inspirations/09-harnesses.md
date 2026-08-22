# Deep Dive: Harness CLIs — Non-Interactive Invocation Surfaces

> Implementation-grade deep dive for the open-seed design study, researched 2026-08-22.
> The prior research covered orchestrators; this covers the HARNESSES' own programmatic
> surfaces — the missing piece for the spawn runner, loop runner, and per-squad harness
> variance (design §6). Sources: live official docs (code.claude.com, learn.chatgpt.com,
> docs.x.ai, docs.github.com, opencode.ai, cursor.com, qwenlm.github.io) plus source-level
> verification from sparse clones of openai/codex, google-gemini/gemini-cli, and sst/opencode.
> Verbatim-from-source claims are marked; unconfirmed items are flagged UNVERIFIED.

## Tier 1

### 1. Claude Code (`claude`)

**Headless invocation.** `claude -p "prompt"` (also `--print`); prompt via argv, or piped stdin (capped at **10 MB** — exceeding it exits non-zero), or both (stdin becomes context). `--bare` is the recommended scripting mode: skips auto-discovery of hooks, skills, commands, subagents, plugins, MCP, auto memory, and CLAUDE.md; never reads OAuth/keychain (requires `ANTHROPIC_API_KEY` or `apiKeyHelper` via `--settings`); docs state it "will become the default for `-p` in a future release". Working dir: process cwd plus `--add-dir <dirs>` (grants file access; most `.claude/` config is *not* discovered from added dirs, but `.claude/skills/` there is). Session naming: `--session-id <uuid>`, `-n/--name <display-name>` (resumable by name), `--fork-session`, `--no-session-persistence` (print mode only).

**Output formats.** `--output-format text|json|stream-json`; `--input-format text|stream-json`. The `json` envelope carries the final text in `result`, plus `session_id`, `total_cost_usd` (with a per-model cost breakdown; both documented as "client-side estimates"), `usage` (`input_tokens`, `output_tokens`, `cache_read_input_tokens`, `cache_creation_input_tokens`), and — with `--json-schema '<schema>'` — validated output in `structured_output`. (Additional fields `num_turns`, `is_error`, `subtype`, `duration_ms` per the Agent SDK message schema.) `stream-json` requires `--verbose`; last line is the `result` message; `--include-partial-messages` adds token deltas; `--forward-subagent-text` (v2.1.211+) emits subagent transcripts with `parent_tool_use_id` threading; `--include-hook-events` adds hook lifecycle events. First stream event is `system/init` (model, tools, `mcp_servers`/`mcp_server_errors`, `plugins`/`plugin_errors`, `capabilities` array for feature detection). Exit codes: 0 success, non-zero on failure; **143 on SIGTERM** (turn left unfinished, resumable; SessionEnd hooks still run); invalid flags error to stderr pre-run; in-run failures print as the result on stdout.

**Resume.** `-c/--continue` (most recent in cwd; skips `-p`/SDK sessions unless the caller is itself `-p`), `-r/--resume <id|name>`. External processes can resume any session by ID — **since v2.1.223 the ID is found in any project on the machine**, not just the launch directory. `--fork-session` resumes into a new ID.

**Permissions.** `--permission-mode default|acceptEdits|plan|auto|dontAsk|bypassPermissions|manual` (`manual` = alias for `default`, v2.1.200+; `auto` = classifier-reviewed; `dontAsk` = deny anything not allow-listed or in the read-only command set — the locked-down CI mode). `--allowedTools`/`--disallowedTools` use permission-rule syntax (`Bash(git diff *)` — trailing space-star is prefix matching; bare tool name in disallowed removes the tool from context entirely; `mcp__*` removes all MCP tools). `--tools` restricts availability (`""` none, `"default"` all). `--dangerously-skip-permissions` = bypassPermissions. `-p` on every plan starts in Manual, so headless callers must pass a mode. Docs warn: without `--bare`, a `-p` session runs project `.claude/settings.json` hooks and connects `.mcp.json` servers **even in never-trusted folders** (no trust dialog in print mode) — a real security consideration for a spawn runner. Budget flags: `--max-turns N` and `--max-budget-usd X` (print mode only; subagent spend counts).

**Hooks/config headless.** All hooks fire in `-p`; `--init`/`--maintenance` run Setup-hook matchers pre-session; `--init-only` runs Setup+SessionStart then exits. Settings honored: user/project/local (`--setting-sources user,project,local` to restrict; `--settings <file-or-json>` to overlay, 2 MiB cap). `--mcp-config <files|json>` (+ `--strict-mcp-config`); with `-p`, waits up to `MCP_TIMEOUT` (default 30 s). System prompt: `--system-prompt[-file]`, `--append-system-prompt[-file]`, `--append-subagent-system-prompt` (v2.1.205+), `--exclude-dynamic-system-prompt-sections` for cross-machine prompt-cache reuse. `--agents '<json>'` defines subagents inline.

**Model/effort.** `--model sonnet|opus|haiku|fable|<full-id>`; `--fallback-model sonnet,haiku` (comma list tried in order on overload). `--effort low|medium|high|xhigh|max|ultracode` (model-dependent).

**Cost/usage.** Best-in-class: `total_cost_usd` + per-model breakdown + full token usage in every `json`/`stream-json` result; explicitly a client-side estimate.

**Failure modes.** `system/api_retry` events with `attempt`, `max_retries`, `retry_delay_ms`, `error` category enum (`rate_limit`, `overloaded`, `billing_error`, …). Background Bash tasks killed ~5 s after result; background subagents awaited up to 10 min (`CLAUDE_CODE_PRINT_BG_WAIT_CEILING_MS`). Slow consumers: exit waits for output drain up to 30 s (v2.1.214+). SIGINT ends the turn cleanly; SIGTERM abandons it (resumable).

### 2. OpenAI Codex CLI (`codex exec`)

**Headless invocation** (verbatim from `codex-rs/exec/src/cli.rs`): usage `codex exec [OPTIONS] [PROMPT]`. Prompt as argv; "If not provided as an argument (or if `-` is used), instructions are read from stdin. If stdin is piped and a prompt is also provided, stdin is appended as a `<stdin>` block." Working dir: `-C/--cd <DIR>`; extra writable roots via `--add-dir <DIR>`. `--skip-git-repo-check` to run outside a repo. `--ephemeral` runs "without persisting session files to disk". `--ignore-user-config`, `--ignore-rules` (skip execpolicy `.rules`), `--strict-config`. Progress goes to stderr; only the final message goes to stdout (pipe-friendly).

**Output formats.** `--json` (alias `--experimental-json` — the flag has graduated) emits JSONL. Event enum verbatim from `exec_events.rs`: `thread.started` (`{thread_id}` — "Can be used to resume the thread later"), `turn.started`, `item.started/updated/completed` (items: `agent_message`, `command_execution`, `mcp_tool_call`, …), `turn.completed` with `usage: {input_tokens, cached_input_tokens, cache_write_input_tokens, output_tokens, reasoning_output_tokens}`, `turn.failed` (`{error}`), `error`. Final text = last `item.completed` with `item.type == "agent_message"`, or use `-o/--output-last-message <FILE>`. `--output-schema <FILE>` = native structured output. Exit codes: 0 success; 1 on fatal errors (source tracks `error_seen` "so we can exit with a non-zero status for automation-friendly signaling"). **No cost-USD reporting; no session_id in the final event (take it from `thread.started`).**

**Resume.** `codex exec resume [SESSION_ID|thread-name] [PROMPT]`, `--last` (most recent, cwd-filtered), `--all` (disable cwd filtering); plus **`codex exec fork <SESSION_ID> [PROMPT]`** (fork into a fresh session). External resume works: sessions are rollout files under `$CODEX_HOME` (unless `--ephemeral`), addressable by UUID from any process.

**Permission/sandbox.** `-s/--sandbox read-only|workspace-write|danger-full-access` (OS-level: Seatbelt / Landlock+seccomp). **exec defaults to `read-only` sandbox and hard-codes `approval_policy=Never` headlessly** (verified in `lib.rs`: "Default to never ask for approvals in headless mode"). The exec entry point exposes **no `-a/--ask-for-approval` flag at all**; the interactive CLI's approval enum is now only `on-request|never` (`on-failure` survives as a serde alias of on-request; a new `granular` policy exists in config with per-category booleans). New: `--approve-for-me` (alias `--not-so-yolo`) routes approvals through an automatic reviewer. `--dangerously-bypass-approvals-and-sandbox` (alias `--yolo`) unchanged. `--full-auto` is deprecated (prints a warning; use `--sandbox workspace-write`). Project-level **execpolicy `.rules` files** now gate commands (`--ignore-rules` to skip). Config overrides: `-c key=value` ("Values parse as TOML if possible").

**Hooks/config.** Codex now has a full Claude-shaped hooks system — event names verbatim from `codex-rs/protocol`: `PreToolUse, PermissionRequest, PostToolUse, PreCompact, PostCompact, SessionStart, SessionEnd, UserPromptSubmit, SubagentStart, SubagentStop, Stop`; handler types `command|mcp_tool|prompt|agent`. Hooks require persisted trust; `--dangerously-bypass-hook-trust` for vetted automation; admins can force `allow_managed_hooks_only = true` in `requirements.toml`. Config: `$CODEX_HOME/config.toml`; profiles are v2: `-p/--profile <name>` layers `$CODEX_HOME/<name>.config.toml` on the base. Instructions: AGENTS.md (nested). Skills: repo + `~/.codex` level.

**Model/effort.** `-m/--model`; effort via `-c model_reasoning_effort=<v>`; enum verbatim from source: `none|minimal|low|medium|high|xhigh|max|ultra` (+ forward-compatible custom strings), default `medium`. `--oss`/`--local-provider lmstudio|ollama` for local models.

**Cost/usage.** Tokens per `turn.completed` only (incl. cache read/write and reasoning tokens); **no USD figure — adapter must price externally** (aeon's `token-usage.csv` approach).

**Failure modes.** Exit 1 with `turn.failed`/`error` events; rate-limit/retry behavior is internal (no retry event stream equivalent to Claude's `api_retry`). Crash leaves the rollout file; `resume` continues it. Auth: `CODEX_API_KEY` (docs recommend scoping the env var to the single invocation) or ChatGPT login; `codex-action` for CI.

### 3. Gemini CLI (`gemini`)

**Headless invocation.** Triggered "in a non-TTY environment or when providing a query with `-p`". `-p/--prompt "text"` ("Appended to stdin input if provided. Forces non-interactive mode"); bare `cat file | gemini` also works; `-i/--prompt-interactive` runs then drops to REPL. Working dir: cwd + `--include-directories` (**no `--cwd` flag** — the adapter must `cd`). No session-naming flag.

**Output formats** (verbatim from `docs/cli/headless.md`): `--output-format text|json|stream-json` (`-o`). `json`: single object — `response` (string, the final answer), `stats` (token usage + API latency), `error` (optional). `stream-json`: JSONL events `init` ("Session metadata (session ID, model)"), `message`, `tool_use`, `tool_result`, `error`, `result` ("Final outcome with aggregated statistics and per-model token usage breakdowns"). **Documented exit codes: `0` success; `1` general/API error; `42` input error; `53` turn limit exceeded.** No USD cost anywhere.

**Resume.** `-r/--resume "latest"|<index>|<session-id>` with optional new prompt (`gemini -r "abc123" "Finish this PR"`); `--list-sessions`, `--delete-session <index>`. Sessions are project-scoped; external resume works within the same project directory.

**Permissions.** `--approval-mode default|auto_edit|yolo|plan` (verified current; underscore, not hyphen). `-y/--yolo` deprecated in favor of `--approval-mode=yolo`. `--allowed-tools` is now **deprecated** — "Use the Policy Engine instead": TOML policy files under `.gemini/policies/` giving allow/deny/ask rules. `-s/--sandbox` (boolean; container/seatbelt via settings). `--skip-trust` bypasses folder trust for the session. Headless caveat: in `default`/`auto_edit`, any tool call that would prompt simply fails back — there is no approver — so effective safe-edit requires the task to need only auto-approved edit tools.

**Hooks/config.** Full hooks system, 11 events (verbatim from `docs/hooks/`): `SessionStart, SessionEnd, BeforeAgent, AfterAgent, BeforeModel, AfterModel, BeforeToolSelection, BeforeTool, AfterTool, PreCompress, Notification` — synchronous, JSON over stdin/stdout, with block/rewrite/retry powers. Config: `settings.json` (user/workspace), `GEMINI.md` hierarchy, extensions, `GEMINI_SYSTEM_MD=<file>` full system-prompt override (still supported — sub-agents-skills' injection trick remains valid). Auth: `GEMINI_API_KEY` / OAuth / Vertex.

**Model/effort.** `-m/--model` aliases `auto` (default) | `pro` | `flash` | `flash-lite`; `auto`/`pro` resolve to `gemini-2.5-pro` or `gemini-3-pro-preview` when preview features are on. **No effort/thinking flag.**

**Cost/usage.** `stats` (json) / `result` event (stream-json) give tokens + latency, per-model breakdowns; no USD.

**Failure modes.** Distinct exit codes are the best of any harness for scripting (42 input vs 53 turn-cap vs 1 general). Free-tier rate limits historically fall back pro→flash; crash leaves the project-scoped session resumable by index.

### 4. OpenCode (`opencode run`)

**Headless invocation** (flags verbatim from `packages/opencode/src/cli/cmd/run.ts`): `opencode run [message..]` — message from argv (also after `--`); `--command` runs a stored command with message as args. Working dir: `--dir <path>`; also `--attach <http://host:port>` to run against a remote/persistent server (with `--username`/`--password`, `OPENCODE_SERVER_PASSWORD`) — a unique capability: the CLI is a thin client over `opencode serve`. Session naming: `--title <t>`; `-s/--session <id>` targets a specific session, `-c/--continue` last session, `--fork` branches before continuing, `--share` publishes.

**Output formats.** `--format default|json`. JSON mode emits per-event lines shaped `{"type", "timestamp", "sessionID", ...}` (verbatim from the `emit()` helper) with types `text`, `reasoning`, `tool_use`, `step_start`, `step_finish`, `error`. **No final aggregate envelope** — the result is the last `text` event; usage/cost ride on `step_finish` parts, whose schema (verbatim from `packages/schema/src/v1/session.ts`) is `cost: number` plus `tokens: {input, output, reasoning, cache: {read, write}, total?}`. Exit: `process.exitCode = 1` set on `session.error`; 0 otherwise.

**Resume.** Yes, externally: sessions live in the server's storage; `--session <id>` from any process (same project), `opencode session list --format json` to enumerate, and `--attach` makes cross-process sharing first-class.

**Permissions.** No permission flags on `run` except `--auto` — "auto-approve permissions that are not explicitly denied (dangerous!)" — with **hidden aliases `--yolo` and `--dangerously-skip-permissions`** (verified in source). Real control is config: `permission` in `opencode.json(c)` with keys `read, edit, glob, grep, bash, task, skill, lsp, question, webfetch, websearch, external_directory, doom_loop`, values `allow|ask|deny` or pattern objects (`"bash": {"*": "ask", "git *": "allow", "rm *": "deny"}` — last matching rule wins). Defaults: most `allow`; `doom_loop` and `external_directory` `ask`; `.env` denied. **`OPENCODE_PERMISSION` env var (JSON) still exists** (verified in `packages/core/src/flag/flag.ts`) — sub-agents-skills' mechanism remains valid. Per-agent permission overrides in agent frontmatter. In headless, `ask` resolves to deny-by-default unless `--auto`. No OS sandbox — policy enforcement only.

**Hooks/config.** No hook system per se; **plugins** (JS, `.opencode/plugin/`) subscribe to the event bus and can intercept `"permission.ask"`, tool execution, etc. Config merge order (verbatim from docs): remote `.well-known/opencode` → global `~/.config/opencode/opencode.json` → `OPENCODE_CONFIG` → project `opencode.json` → `.opencode` dirs → `OPENCODE_CONFIG_CONTENT` → managed → macOS managed prefs. `instructions: [globs]`, AGENTS.md native, agents in `.opencode/agents/*.md`, `{env:VAR}`/`{file:path}` substitution.

**Model/effort.** `-m/--model provider/model`; `--variant` — "model variant (provider-specific reasoning effort, e.g., high, max, minimal)"; `small_model` config for cheap tasks. Fallback: none automatic (config-level only).

**Cost/usage.** Native USD `cost` per step and per assistant message (priced from models.dev data) — the only Tier-1 harness besides Claude with USD, and it works for any provider. Reliability: good, but it is list-price arithmetic.

**Failure modes.** `session.error` event → exit 1; retries surface as `retry` parts (schema has `RetryPart {attempt, error}`); sessions persist server-side so crashes are resumable; the server/client split means a killed CLI does not necessarily kill the run when attached.

## Tier 2

**Cursor CLI (`cursor-agent`).** Headless: `cursor-agent -p "prompt"` — print mode "has access to all tools, including write and shell"; `--output-format text|json|stream-json` (+ `--stream-partial-output`); `--workspace <path>`; `create-chat` mints a session ID, `ls` lists, `--resume [chatId]`, `--continue` = "alias for `--resume=-1`". Permissions: `--mode plan|ask` (agent default), `-f/--force` (alias `--yolo`) bypasses command approval, `--trust` ("Trust the workspace without prompting (headless mode only)"), `--sandbox enabled|disabled`, `--approve-mcps`. Auth `CURSOR_API_KEY`/`--api-key`; `--model`; `--plugin-dir` repeatable. The inspirations/05 mapping (`--mode plan` / `--trust` / `-f --trust`) remains valid. Exit codes and JSON envelope schema: not documented (UNVERIFIED); no USD cost. No effort flag.

**GitHub Copilot CLI (`copilot`).** Programmatic mode: `copilot -p "prompt"` (piped stdin works but is *discarded* if `-p` is present); `-s` suppresses stats/decoration "so you get clean text". **No JSON output format at all** — the biggest gap in the field; parse plain text or use `--share <path>` (Markdown transcript) / `--share-gist`. Permissions: deny-by-default with `--allow-tool=TOOL`/`--deny-tool=TOOL` (deny always wins, even over `--allow-all-tools`/saved approvals) using grammar `shell(git:*)`, `shell(npm test)`, `write(README.md)`, plus `--allow-url`/`--allow-all-urls` (`url(https://*.github.com)`), `--add-dir`, `--no-ask-user`; env `COPILOT_ALLOW_ALL=true`. Model: `--model` (e.g. `gpt-5.2`, `claude-sonnet-4.6`); precedence custom-agent → flag → `COPILOT_MODEL` → config → default. `--agent` delegates to repo custom agents. Config in `~/.copilot` (`COPILOT_HOME`); auth `COPILOT_GITHUB_TOKEN`/`GH_TOKEN`. CLI-flag session resume is not documented on the programmatic pages (interactive `/resume` only) — treat external resume as absent (UNVERIFIED whether a `--resume` flag exists). No token/cost reporting (billing is premium-request-based, invisible per run).

**Grok CLI ("Grok Build", xAI — GA'd from May 2026 beta).** Headless via `-p, --single <PROMPT>`; `--output-format plain|json|streaming-json` (streaming now includes tool calls/results and usage events as of 0.2.116); sessions: `-s/--session-id <ID>` "Create or resume a named headless session" (the only listed harness with caller-chosen string session names), `-r/--resume <ID>`, `-c/--continue`; `--cwd <PATH>`. Permissions: default `ask`; `--permission-mode dontAsk` documented, config `permission_mode = "always-approve" | "ask"` in `~/.grok/config.toml`, `--always-approve` flag; allow-lists use Claude-style grammar (`--allow 'Bash(git *)'`); OS sandbox profiles `--sandbox workspace|read-only|strict|devbox|off` with custom deny globs. `--reasoning-effort` per inspirations/05 (not on the current headless page — UNVERIFIED). Auth `XAI_API_KEY`. 05's `--permission-mode bypassPermissions` and `--verbatim` do not appear in current docs — likely renamed/removed. JSON envelope fields undocumented publicly (UNVERIFIED).

**Qwen Code (`qwen`).** Gemini-CLI fork that has meaningfully diverged toward the Claude envelope: `-p/--prompt`, stdin; `--output-format text|json|stream-json` where **json emits Claude-style message objects** (`type: system|assistant|result`, `subtype` e.g. `session_start|success`, `session_id`, `message` with usage, final `result`) and stream-json supports `--include-partial-messages`; `--continue` and `--resume <uuid>` work headlessly (sessions under `~/.qwen/projects/<sanitized-cwd>/chats` as JSONL). Approval: `--approval-mode plan|default|auto-edit|auto|yolo` (note hyphenated `auto-edit` and an extra `auto` tier vs Gemini), `--yolo`, `--sandbox`. Unique budget surface: `--max-session-turns N`, `--max-wall-time 5m`, `--max-tool-calls 50`; exit `53` = turn cap, `55` = budget overrun; `QWEN_CODE_UNATTENDED_RETRY=1` retries 429/529 indefinitely with backoff — genuinely useful for loop runners. `--system-prompt`/`--append-system-prompt` supported (unlike Gemini). Auth: Qwen OAuth (generous free tier) or OpenAI-compatible env vars.

**Amp (`amp`).** `amp -x/--execute "prompt"` (or stdin) for one-shot; `--stream-json` emits JSONL using **"a Claude Code-compatible protocol with a `type` field discriminator"** (tool calls, token usage, thinking, permission decisions), `--stream-json-input` for streamed input; multi-turn across invocations via `amp threads continue [thread-id]` (threads are server-side, shareable — external resume is first-class). Permissions: `--dangerously-allow-all` to auto-approve; finer control lives in settings, not flags (exact settings schema UNVERIFIED). Model selection is largely managed by Amp itself; no public effort flag. Cost: usage-based billing; token usage in stream events, USD not per-run. The Claude-compatible stream makes an Amp adapter nearly free if you already parse Claude stream-json.

**Pi (`pi`).** Minimalist by design: `-p/--print` single-shot (stdin merges into the prompt); `--mode json` = NDJSON events; `--mode rpc` = full JSON-over-stdio command protocol (the richest programmatic surface of the small harnesses); sessions are JSONL trees: `--session <path|id>`, `-c/--continue`, `--fork`, `--no-session`. Models: `--model provider/id` or patterns incl. thinking level suffix (`sonnet:high`); `--thinking off|minimal|low|medium|high|xhigh|max`; `--provider`, `--api-key`. **No permission system at all**: tools (`read, bash, edit, write, grep, find, ls`) run unprompted; the only gate is project-trust (`defaultProjectTrust`, `-a/--approve` / `-na/--no-approve` for project-file loading). Scoping is via tool allowlists: `--tools <list>`, `--exclude-tools`, `--no-tools`. Extensions are TypeScript modules (`-e`). Env: `PI_SESSION_ID`, `PI_SESSION_FILE`, `PI_PROVIDER`, `PI_MODEL` exported to bash tool commands. gh-aw ships it as a first-party engine, which is why it matters despite its size.

## SYNTHESIS for open-seed

### (a) Capability matrix

Legend: ● native · ◐ partial/caveat · ○ absent. (CC=Claude Code, CX=Codex, GM=Gemini, OC=OpenCode, CU=Cursor, CP=Copilot, GK=Grok, QW=Qwen, AM=Amp, PI=Pi)

| Dimension | CC | CX | GM | OC | CU | CP | GK | QW | AM | PI |
|---|---|---|---|---|---|---|---|---|---|---|
| 1. Headless invocation (argv+stdin+cwd flag) | ● | ● (`-C`) | ◐ (no cwd flag) | ● (`--dir`) | ● | ◐ (stdin xor `-p`) | ● (`--cwd`) | ◐ | ● | ● |
| 2. JSON envelope w/ result+usage | ● | ◐ (events; no final envelope, no cost) | ● (`response`+`stats`) | ◐ (events only) | ◐ (undocumented schema) | ○ (text only) | ◐ | ● (Claude-shaped) | ● (Claude-compatible) | ◐ |
| 3. External session resume | ● (any-project by ID) | ● (rollout files) | ◐ (project-scoped) | ● (server-backed) | ● | ○ | ● (named sessions) | ● | ● (threads) | ● (files) |
| 4. Permission modes + tool grammar | ● | ● (+OS sandbox) | ◐ (policy engine; no OS sandbox default) | ◐ (config/env only) | ◐ | ● (grammar, no modes) | ● (+OS sandbox) | ◐ | ◐ | ○ |
| 5. Hooks headless / project config | ● | ● (11 events, trust-gated) | ● (11 events) | ◐ (plugins) | ◐ | ◐ | ? | ◐ | ◐ | ◐ (extensions) |
| 6. Model + effort flags | ● | ● (`-c model_reasoning_effort`) | ◐ (model only) | ● (`--variant`) | ◐ | ◐ | ◐ | ◐ | ○ | ● (`--thinking`) |
| 7. USD cost per run | ● | ○ | ○ | ● | ○ | ○ | ○ | ○ | ○ | ◐ |
| 8. Distinct exit codes / budget flags | ◐ (0/143/nonzero; `--max-turns`, `--max-budget-usd`) | ◐ (0/1) | ● (0/1/42/53) | ◐ (0/1) | ? | ? | ? | ● (53/55 + wall-time/tool-call caps) | ? | ? |

### (b) `harness adapter` contract proposal

One aeon-shaped contract, checked in at `.seed/adapters/<name>.sh`, invoked by both the spawn runner and `loop.sh`:

```
seed-harness <name> [--model M] [--effort E] [--permission read-only|safe-edit|yolo]
             [--cwd DIR] [--resume SESSION_ID] [--json-schema FILE]
             [--max-turns N] [--timeout SECS] [--append-system-prompt TEXT] < prompt.txt
```

Env in (never argv, per sub-agents-skills' credential rule): harness API keys, `SEED_TASK_ID`. **Prompt always on stdin** (every Tier-1 harness accepts it; the opencode adapter alone must read stdin into argv, quoting-safe). Stdout: exactly one JSON object:

```json
{"result": "...", "usage": {"input_tokens": 0, "output_tokens": 0,
 "cache_read_input_tokens": 0, "cache_creation_input_tokens": 0},
 "session_id": "...", "cost_usd": 1.23, "harness": "codex", "raw_exit": 0}
```

(`session_id`/`cost_usd` optional — matching aeon's stdout contract exactly, extended with `harness`/`raw_exit` for receipts.) Normalized exit codes, merging aeon + sub-agents-skills: **0** ok · **1** harness-reported failure · **3** abnormal stop with no result · **124** timeout (adapter-enforced `timeout(1)`; harness-native wall-time caps used where they exist, e.g. qwen `--max-wall-time`) · **127** harness binary missing.

Per-harness mapping (base command → result extraction → usage → session → cost):

| Harness | Base command | result | usage | session_id | cost_usd |
|---|---|---|---|---|---|
| claude | `claude --bare -p --output-format json <perm> [--max-turns N]` | `.result` | `.usage` (already contract-shaped) | `.session_id` | `.total_cost_usd` |
| codex | `codex exec --json --skip-git-repo-check -C "$cwd" <perm> -` | last `item.completed` where `item.type=="agent_message"` (or `-o` tmpfile) | sum `turn.completed.usage`; map `cached_input_tokens`→`cache_read`, `cache_write_input_tokens`→`cache_creation` | `thread.started.thread_id` | absent (optional price table) |
| gemini | `cd "$cwd" && gemini --skip-trust --output-format json <perm> -p "$(cat)"` | `.response` | map `.stats` token fields | stream-json `init` only; omit in json mode | absent |
| opencode | `opencode run --format json --dir "$cwd" <perm> "$(cat)"` | last `text` event's `part.text` | sum `step_finish.part.tokens` (`cache.read/write`→contract names) | any event's `.sessionID` | sum `step_finish.part.cost` |
| resume | `--resume "$id"` / `codex exec resume "$id" -` / `gemini -r "$id"` / `--session "$id"` | | | | |
| effort | `--effort $E` | `-c model_reasoning_effort="$E"` | unsupported → warn | `--variant $E` | |
| schema | `--json-schema "$(cat F)"` → `.structured_output` | `--output-schema F` | prompt+validate+retry (aeon fallback) | prompt+validate+retry | |

**Permission-tier mapping and fidelity flags** (any ◐/✗ below is a *guardrails variance* the squad's variance record must declare, per design §6):

| Tier | claude | codex | gemini | opencode |
|---|---|---|---|---|
| read-only | `--permission-mode plan` (or stricter: `--tools "Read,Grep,Glob" --permission-mode dontAsk`) ● | `-s read-only` ● (OS-enforced — strongest) | `--approval-mode plan` ◐ (policy-level; no OS sandbox unless `-s`) | `OPENCODE_PERMISSION='{"edit":"deny","bash":"deny",...}'` ◐ (policy-level) |
| safe-edit | `--permission-mode acceptEdits` ● (shell beyond read-set still gated → fails closed headlessly) | `-s workspace-write` ● (`approval_policy=never` is now the exec default — 05's `-c` override is redundant but harmless) | `--approval-mode auto_edit` ◐ **cannot be faithful**: auto-approves edits only; any shell step dies with no approver — document as variance | `OPENCODE_PERMISSION='{"edit":"allow","bash":"allow","external_directory":"deny",...}'` ◐ (bash allowed = broader than "workspace-write"; no fs scoping — variance) |
| yolo | `--dangerously-skip-permissions` ● | `--dangerously-bypass-approvals-and-sandbox` ● | `--approval-mode yolo` ● | `--auto` (aliases `--yolo`, `--dangerously-skip-permissions`) ● |

Tier-2 fidelity flags: **Copilot has no read-only tier** (`--deny-tool 'write' --deny-tool 'shell'` approximates it but kills git reads — variance) and no yolo-with-sandbox; **Pi cannot implement safe-edit at all** (no workspace scoping; read-only only via `--tools read,grep,find,ls`, yolo is the default — double variance); **Amp's** read-only depends on settings-file rules, not flags (variance); **Grok** maps cleanly (`--sandbox read-only|workspace|off` + `--permission-mode dontAsk`) and is the only Tier-2 with OS enforcement; **Qwen** maps like Gemini plus real budget caps; **Cursor** `--mode plan` / `--trust` / `-f --trust`, `--sandbox disabled` needed for true yolo.

### (c) v1 adapters vs documented extension points

**v1 adapters (ship): `claude`, `codex`.** Claude Code is the contract's native shape (zero translation; richest envelope, cost, resume, hooks) and matches the design's Claude-Code-first posture; Codex is the strongest *second* precisely because it is maximally different where it matters (OS sandbox instead of permission prompts, JSONL instead of envelope, no cost) — an adapter layer proven against these two generalizes. Both have deterministic headless defaults (Manual-mode `-p` / `approval_policy=Never`) that fail closed.

**v1.5 (adapter stubs in-tree, tested best-effort): `opencode`, `gemini`.** OpenCode earns it with native USD cost for any provider and server-backed sessions (useful for the loop runner's crash recovery), but its env-JSON permission channel and no-final-envelope stream need more glue. Gemini earns it on distinct exit codes and ubiquity, but its safe-edit tier is unfaithful (documented variance) and it lacks a cwd flag.

**Documented extension points only: everything else.** Qwen (easy port of the claude adapter — envelope is Claude-shaped — but niche), Amp (Claude-compatible stream, but thread/billing model is product-coupled), Grok (clean mapping, subscriber-gated access), Cursor (fine but API-key/product-coupled), Copilot (no JSON output — cannot honor the envelope contract without scraping; document as "text-only degraded mode"), Pi (permission tiers unimplementable; suited to gh-aw-style externally-sandboxed CI, not the local spawn runner).

### (d) Corrections to existing research

1. **05, Codex map:** the `-c approval_policy=never` override is now redundant: `codex exec` hard-defaults `approval_policy=Never` headlessly (verified in source). Also since 05: `--full-auto` deprecated; `-a on-failure` demoted to an alias of `on-request` and `-a` accepts only `on-request|never` (plus a config-only `granular` policy); `--yolo` is an official alias; new `--approve-for-me`/`--not-so-yolo` auto-reviewer mode; `codex exec fork` exists; `--ephemeral` for stateless CI runs; `--profile/-p` is a v2 layered config.
2. **05, Grok map:** `--permission-mode bypassPermissions` and `--verbatim` do not appear in current xAI docs; documented values are `ask`/`dontAsk` (flag) and `always-approve` (config), and the sandbox enum has grown to `workspace|read-only|strict|devbox|off`. The product is now branded "Grok Build". Treat 05's grok row as stale until re-verified against a live binary.
3. **05, Gemini:** `--approval-mode` values confirmed unchanged, but `--allowed-tools` is now deprecated in favor of the Policy Engine (`.gemini/policies/` TOML) — an orchestrator should not build on that flag. `GEMINI_SYSTEM_MD` injection still valid.
4. **05, Claude:** permission-mode enum has grown (`auto`, `dontAsk`, `manual` added); `--effort` now has six values up to `ultracode`; new `--bare` changes the recommended spawn shape, and `-p` bypasses workspace trust for hooks/MCP — a real security consideration for the spawn runner that 05's map doesn't mention.
5. **06 (aeon):** its run-harness claim "`--json-schema` native on claude/grok/codex" is confirmed for claude (`--json-schema`) and codex (`--output-schema`); its envelope remains the right template — Codex's `cache_write_input_tokens` and OpenCode's `cache.write` map cleanly onto aeon's `cache_creation_input_tokens`.
6. **09-sota:** "requirements.toml … implying Codex now has lifecycle hooks; the project-level hook layout is lower-confidence" — now confirmed at source level: 11 named events, four handler types, hook-trust persistence, `--dangerously-bypass-hook-trust`. Also 09-sota's "Codex config primarily user-level" understates the new project-level surface: execpolicy `.rules` files and skills are project-scoped. `developers.openai.com/codex/*` URLs cited in 09-sota now 308-redirect to `learn.chatgpt.com`. Minor: OpenCode agents dir is `.opencode/agents/` (plural) per current config docs.

**Confidence notes:** Claude, Codex, Gemini, OpenCode verified against primary docs and (for Codex/Gemini/OpenCode) source code cloned 2026-08-22. Copilot/Cursor/Qwen verified against official docs pages; Grok against docs.x.ai plus secondary sources (flag details beyond the headless page UNVERIFIED); Amp against ampcode.com news/manual plus SDK docs; Pi against its repo README. Exact JSON envelope internals for Cursor and Grok, and the existence of a Copilot `--resume` flag, remain the three open verification items — all Tier-2, none load-bearing for the v1 adapter cut.
