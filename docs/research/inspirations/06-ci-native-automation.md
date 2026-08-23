# Deep Dive: CI-Native Agent Automation: Exact Schemas and Verbatim Formats

> Implementation-grade deep dive for the open-seed design study, researched 2026-08-22.
> Sources: full shallow clones of gh-aw, aeon, sortie, contrabass, and claude-code-action.
>
> **Erratum (design review, 2026-08-22):** superseded by `docs/design-options.md` D1/D7 on two
> points: (1) the state label is `seed:in_progress` (not `seed:working`), with the full
> state↔label mapping defined in D1 (backlog = no state label; cancelled = closed as
> not-planned); (2) the dispatcher draft's direct `gh issue edit` label swaps should be read
> as the *mirror-export* step rendering card state: actual task-state mutations in workflows
> go through `scripts/seed task <verb>` per the port rule (§7.1), with labels as a one-way
> rendered view (cards authoritative).

## 1. github/gh-aw (GitHub Agentic Workflows)

**Survey correction up front:** the repo has moved from `githubnext/gh-aw` to **`github/gh-aw`** (first-party GitHub org); docs live in-repo at `docs/src/content/docs/`. The trigger formerly called `command:` is now **`slash_command:`**, and there is a separate **`label_command:`** trigger.

### 1.1 Complete example agentic workflow (verbatim, abridged; frontmatter complete)

From `.github/workflows/auto-triage-issues.md` (the repo dogfoods ~100 of these; each has a committed `.lock.yml` sibling):

```yaml
---
emoji: "🔧"
name: Auto-Triage Issues
description: Automatically labels new and existing unlabeled issues to improve discoverability and triage efficiency
on:
  issues:
    types: [opened, edited]
  schedule:
    - cron: "11 */6 * * *" # Explicit offset to avoid the shared 22:29 UTC batch
  workflow_dispatch:
max-daily-ai-credits: 10000
user-rate-limit:
  max-runs-per-window: 5
  window: 60
permissions:
  contents: read
  issues: read
  copilot-requests: write
model: copilot/gpt-5.4
engine:
  id: pi
strict: true
network:
  allowed:
    - defaults
    - github
imports:
  - shared/mcp-pagination.md
  - shared/github-guard-policy.md
  - shared/reporting.md
  - shared/otlp.md
tools:
  cli-proxy: true
  github:
    mode: gh-proxy
    toolsets:
      - issues
    min-integrity: approved
  bash:
    - "*"
steps:
  - name: Fetch unlabeled issues
    env:
      GH_TOKEN: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
    run: |
      mkdir -p /tmp/gh-aw/agent
      gh api "repos/github/gh-aw/issues?state=open&labels=&per_page=30" \
        --jq '[.[] | select(.labels | length == 0) | {number: .number, title: .title, body: .body}]' \
        > /tmp/gh-aw/agent/unlabeled-issues.json
safe-outputs:
  add-labels:
    max: 10
  create-discussion:
    expires: 1d
    title-prefix: "[Auto-Triage] "
    category: "audits"
    close-older-discussions: true
    max: 1
  noop:
timeout-minutes: 15
sandbox:
  agent:
    runtime: gvisor
evals:
  - id: labels-applied
    question: Did the agent apply at least one label to an unlabeled issue, or correctly call noop when no unlabeled issues were found?
---

# Auto-Triage Issues Agent 🏷️
[natural-language body: objective, per-trigger branching, explicit noop instructions, label taxonomy]
```

### 1.2 Frontmatter schema (from `docs/reference/frontmatter.md`; `frontmatter-full.md` is a 22,535-line generated field reference)

Top-level keys: `description`, `emoji`, `labels`, `metadata`, `on`, `if`, `imports`, `import-schema` (typed inputs for shared workflows), `pre-steps`/`steps`/`pre-agent-steps`/`post-steps`/`jobs`, `cache`, `checkout` (incl. `checkout: false`), `ambient-folders`, `permissions`, `github-app`, `engine` (`id`, `harness.watchdog-timeout`, `driver`, `extensions`), `network`, `tools`, `mcp-servers`, `mcp-scripts`, `skills` (Copilot skills, SHA-pinned at compile), `plugins`, `safe-outputs`, `threat-detection-suppress`, `run-name`/`runs-on`/`timeout-minutes` (default **20 min**), `concurrency`, `env`, `excluded-env`, `secrets`, `environment`, `container`, `services`, `observability.otlp`, `resources`, `runtimes`, `source`, `redirect`, `tracker-id`, `private`, `check-for-updates`, `features`, `strict` (default **true**; `strict: false` workflows refuse to run on public repos).

**Budget keys with defaults (verbatim semantics):**

| Key | Default | Meaning |
|---|---|---|
| `max-turns` | `500` | chat-iteration cap enforced by the AWF proxy |
| `max-turn-cache-misses` | `5` | consecutive proxy cache misses before requests are blocked |
| `max-ai-credits` | `1000` | per-run AI-credits budget; steering warnings at 80/90/95/99%; `-1` disables |
| `max-daily-ai-credits` | disabled | 24h per-workflow cap aggregated across runs; on breach: warn, create issue, skip agent job |
| `user-rate-limit` | n/a | `max-runs-per-window` (1–10, required), `window` (default 60 min, max 180), `events`, `ignored-roles` (default `[admin, maintain, write]`) |
| `timeout-minutes` | `20` | agent job wall clock |

**Triggers (`on:`)**: standard Actions triggers plus: `reaction:`, `status-comment:`, `stop-after:` (absolute date or `+7d` relative to compile time; recompiling resets), `manual-approval: <environment-name>`, `forks:`, `roles:` (default `[admin, maintainer, write]`), `skip-roles:`, `bots:`/`skip-bots:`, `skip-author-associations:`, `skip-if-match:`/`skip-if-no-match:` (GitHub search query with `max:` threshold), `on.steps:`, `on.needs:`. Schedules support fuzzy syntax: `schedule: daily`, `daily around 14:00`, `weekly on friday around 5pm`, `every 10 minutes` (5-min floor): the compiler scatters a deterministic time per file path.

**Command triggers:**

```yaml
on:
  slash_command:
    name: investigate
    events: [issues, issue_comment]
```

**Label command (labels-as-one-shot-commands: directly relevant to open-seed):**

```yaml
on:
  label_command:
    name: deploy
    events: [pull_request]
    remove_label: false   # default true: label auto-removed after activation so it can re-trigger
```

The compiler generates `types: [labeled]` events, a `workflow_dispatch` with `item_number` for testing, and a label-removal step; matched label exposed as `needs.activation.outputs.label_command`. With `remove_label: false` the label is persistent state.

**Tools block (verbatim syntax):**

```yaml
tools:
  edit:
  bash:                              # Default safe commands (echo, printf, ls, pwd, cat, head, tail, grep, wc, sort, uniq, date, yq)
  bash: []                           # Disable all commands
  bash: ["echo", "ls", "git status"] # Specific commands only
  bash: [":*"]                       # All commands; "git:*" = command family
  github:
    toolsets: [repos, issues]        # modes: local | remote | gh-proxy
  web-fetch:
  web-search:
  playwright: { version: "1.56.1" }
  cache-memory:
  repo-memory:
  cli-proxy: true                    # mount MCP servers as CLI commands on PATH
  timeout: 120                       # per-tool-call seconds
  startup-timeout: 60                # MCP init
mcp-servers:
  slack:
    command: "npx"
    args: ["-y", "@slack/mcp-server"]
    env: { SLACK_BOT_TOKEN: "${{ secrets.SLACK_BOT_TOKEN }}" }
    allowed: ["send_message", "get_channel_history"]
    required: false                  # best-effort startup
```

**Network:** `network: { allowed: [defaults, python, "api.example.com"] }`: ecosystem identifiers + domain allowlist, enforced by a Squid/api-proxy firewall container stack (the lock manifest pins the firewall images by digest).

**Safe outputs: full type list**: `create-issue` (max 1), `update-issue`, `close-issue`, `link-sub-issue`, `create-discussion`, `update-discussion`, `close-discussion`, `create-pull-request`, `update-pull-request`, `close-pull-request` (max 10), `approve-workflow-run`, `merge-pull-request` (experimental), `create-pull-request-review-comment` (max 10), `reply-to-pull-request-review-comment`, `resolve-pull-request-review-thread`, `submit-pull-request-review`, `add-reviewer` (max 3), `push-to-pull-request-branch`, `add-comment`, `hide-comment` (max 5), `add-labels` (max 3), `remove-labels` (max 3), `assign-milestone`, `assign-to-agent`, `assign-to-user`, `unassign-from-user`, `set-issue-type` (max 5), `set-issue-field` (max 5), `create-project`, `update-project` (max 10), `create-project-status-update`, `update-release`, `upload-artifact`, `upload-asset`, `dispatch-workflow` (max 3), `call-workflow`, `dispatch-repository`, `create-code-scanning-alert`, `autofix-code-scanning-alert`, `create-check-run`, `create-agent-session`, plus auto-enabled system types `noop`, `missing-tool`, `missing-data`. With no `safe-outputs:` section, `create-issue` is auto-injected (`max: 1`). Global fields: `staged:`, `report-failed-jobs`, `report-failure-as-issue`, `group-reports`, `github-token`, `github-app`, `environment`, `allowed-domains`, `allowed-github-references`, `max-bot-mentions` (default 10), `data:` (schema-validated JSON side channel).

Per-handler option set (create-issue, verbatim):

```yaml
safe-outputs:
  create-issue:
    title-prefix: "[ai] "
    labels: [automation, agentic]
    assignees: [user1, copilot]
    max: 5                           # default 1
    expires: 7                       # auto-close after 7 days ("7d","2w","1m","2h"; false disables)
    group: true                      # sub-issues under a parent (max 64)
    close-older-issues: true
    group-by-day: true
    deduplicate-by-title: 1          # true=exact; int=Levenshtein distance
    target-repo: "owner/repo"
    allowed-repos: ["org/repo1"]
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }}
```

`expires` auto-generates an `agentics-maintenance.yml` cleanup workflow (frequency derived from the shortest expiry). Every created item carries hidden `<!-- gh-aw-workflow-id: NAME -->` and optional tracker-id markers, searchable via `repo:owner/repo "gh-aw-workflow-id: daily-team-status" in:body`.

### 1.3 Compiled `.lock.yml` job graph

The lock for auto-triage is **2,162 lines** for a 375-line md. Header: `# gh-aw-metadata:` (schema v4, frontmatter/body hashes, engine versions) and `# gh-aw-manifest:` (every secret name, every action SHA-pinned, every container image digest-pinned, MCP servers + tools). Job graph (exact `needs`):

```
pre_activation ──▶ activation (if activated=='true') ──▶ agent (if daily credits not exceeded)
agent ──▶ detection (always() && agent!=skipped)                 # threat detection
{activation, agent, detection} ──▶ safe_outputs (!cancelled() && detection==success)
{activation, agent, detection} ──▶ evals ──▶ push_evals_state
{...all...} ──▶ conclusion
```

### 1.4 Sanitization rules

Two directions. **Input** (activation job's sanitized output): neutralizes @mentions and bot triggers (`fixes #123`), XML-injection protection, HTTPS-only URI filtering to trusted domains, 0.5 MB / 65k-line caps, ANSI-escape stripping. **Output** (before safe-output apply): "XML escaped, HTTPS only, domain allowlist (GitHub by default), 0.5MB/65k line limits, control char stripping"; HTML comments removed; URLs off-allowlist → `(redacted)`; GitHub refs to unlisted repos backtick-escaped; bot mentions beyond `max-bot-mentions` escaped; secret redaction. Threat-detection job rules are `CTR-###` diagnostics, suppressible via `threat-detection-suppress: [{rule, reason, expires}]`.

### 1.5 CLI verbs

`gh aw` subcommands: `init, add-wizard, add, new, secrets, doctor, fix, compile, validate, lint, trial, run, list, status, logs, audit, outcomes, health, checks, forecast, experiments, enable, disable, remove, update, deploy, upgrade, env, mcp, pr transfer, mcp-server, domains, version, completion, project, hash-frontmatter`.

## 2. aeonfun/aeon

Architecture: root `aeon.yml` config → `scheduler.yml` (5-min tick) parses it in **bash regex** (not a YAML parser!), cron-matches, and `gh workflow run aeon.yml -f skill=<name>` per match; the `aeon.yml` workflow (1,935 lines) runs one skill through a harness; state in `memory/cron-state.json` committed back.

**aeon.yml schema (verbatim skill entries):**

```yaml
skills:
  pr-review: { enabled: false, schedule: "0 9 * * *", var: "" }
  deploy-uni-hook: { enabled: false, schedule: "workflow_dispatch", harness: "claude", model: "claude-opus-4-8", var: "" }
  skill-repair: { enabled: false, schedule: "reactive", var: "" }  # never cron-matched
  heartbeat: { enabled: true, schedule: "0 8 * * *", var: "" }     # the only default-enabled skill
reactive:
  # skill-repair:
  #   trigger:
  #     - { on: "*", when: "consecutive_failures >= 3" }
chains:
  # routine:
  #   schedule: "0 7 * * *"
  #   on_error: fail-fast          # fail-fast (default) | continue
  #   max_dispatches: 10
  #   steps:
  #     - { parallel: [skill-a, skill-b] }
  #     - { skill: skill-c, consume: [skill-a, skill-b] }
  #     - { skill: polish,  consume: [skill-c], when: "score > 3" }   # Haiku-scored 1-5
model: claude-sonnet-5
harness: claude          # claude | grok | codex | pi | vibe | kimi
gateway: { provider: auto }
telegram: { enabled: true }
```

Reactive `when:` conditions: `consecutive_failures >= N`, `success_rate <op> X`, `last_status = <value>`.

**SKILL.md frontmatter (verbatim):**

```yaml
---
name: glim-mcp
description: Live-data research via the glim.sh MCP - ...
metadata:
  title: Glim MCP
  mode: read-only          # only valid values: read-only | write; unknown falls back to WRITE (aeon-doctor flags this)
  category: basics
  var: ""
  tags: [research, data, mcp]
  mcp:
    - glim
  capabilities:
    - external_api
    - sends_notifications
---
```

(`requires: [RESEND_API_KEY?]`: `?` = optional secret. `scorable: false` skips the post-run quality scorer. `depends_on:` is parsed by the scheduler for dispatch ordering.)

**scheduler.yml mechanics:** `on: schedule '*/5 * * * *'` + `workflow_dispatch` + `repository_dispatch: [cron-tick]` (an external pinger can POST cron-tick because "GitHub delivers only ~10% of */5 schedule ticks"). `concurrency: {group: aeon-scheduler, cancel-in-progress: false}` serializes ticks. Match pipeline per skill: circuit breaker gate → failed-retry (30-min cooldown) → exact-slot "debt" cron matching (compares the most recent slot within `CATCHUP_HOURS` to `last_dispatch`: can't double-fire, catches up missed slots).

**Circuit breaker:** `scripts/breaker.sh` → `closed | open | probe` (half-open lets one probe through; success resets `consecutive_failures`). Thresholds overridable via repo vars `BREAKER_THRESHOLD` / `BREAKER_COOLDOWN_MIN`; `0` disables.

**cron-state.json (verbatim):**

```json
{ "heartbeat": { "last_dispatch": "2026-07-02T21:23:42Z", "last_status": "dispatched" } }
```

**Issues-as-state backend:** `STATE_BACKEND` repo var = `file` (default, commit the JSON) | `dual` (file authoritative + mirror to a labeled Issue) | `issues` (Issue ledger authoritative; materialize folds the append-only event stream into the state file; no commit).

**token-usage.csv (exact header):** `date,skill,model,input_tokens,output_tokens,cache_read,cache_creation`

**run-harness contract** ("one Claude Code-shaped contract, six coding-agent harnesses", adapters in `adapters/<name>.sh`): `run-harness <claude|grok|codex|pi|vibe|kimi> [options] < prompt.txt`; flags `--model`, `--allowed-tools "<Claude --allowedTools grammar>"`, `--mode read-only|write` (derived from allowed-tools if omitted; default write), `--mcp-config <claude-style .mcp.json>` (translated per harness), `--max-turns` (claude/grok only), `--json-schema` (native on claude/grok/codex; prompt+validate+retry on pi/vibe/kimi), `--append-system-prompt`, `--no-sandbox`. Stdout contract: `{"result": "...", "usage": {input_tokens, output_tokens, cache_read_input_tokens, cache_creation_input_tokens}, [session_id], [total_cost_usd]}`; exit 0 ok / 3 abnormal-stop-with-no-output.

**Self-healing loop:** `heartbeat` (daily fleet check) → `skill-health` (daily; classifies per-skill health from Actions run data, files/resolves issues in `memory/issues/`) → `skill-repair` (`schedule: "reactive"`, fired when `consecutive_failures >= 3`) auto-fixes and PRs. `aeon-doctor` (weekly) is a static linter for the config-silent-failure class.

## 3. sortie-ai/sortie

A long-running Go daemon (not Actions-hosted) driven by one `WORKFLOW.md`. **Root WORKFLOW.md frontmatter (verbatim):**

```yaml
---
tracker:
  kind: github
  api_key: $GITHUB_TOKEN
  project: sortie-ai/sortie
  query_filter: "label:agent-ready"
  active_states: [todo, in-progress]
  in_progress_state: in-progress
  terminal_states: [done, rejected, duplicate]
polling:
  interval_ms: 60000
workspace:
  root: ~/workspace/sortie
hooks:
  after_create: |
    git clone --depth 1 git@github.com:sortie-ai/sortie.git .
    go mod download
  before_run: |
    git fetch origin main
    git checkout -B "sortie/${SORTIE_ISSUE_IDENTIFIER}" origin/main
  after_run: |
    rm -f CLAUDE.md
    make fmt 2>/dev/null || true
    git add -A
    git diff --cached --quiet || git commit -m "sortie(${SORTIE_ISSUE_IDENTIFIER}): automated changes"
    git push origin "sortie/${SORTIE_ISSUE_IDENTIFIER}"
  before_remove: |
    git push origin --delete "sortie/${SORTIE_ISSUE_IDENTIFIER}" 2>/dev/null || true
  timeout_ms: 120000
agent:
  kind: claude-code
  command: claude
  max_turns: 5
  max_sessions: 3
  max_concurrent_agents: 1
  turn_timeout_ms: 1800000
  read_timeout_ms: 10000
  stall_timeout_ms: 300000
  max_retry_backoff_ms: 120000
claude-code:
  permission_mode: bypassPermissions
  model: claude-sonnet-4-20250514
  max_budget_usd: 5
  max_turns: 50
---
[Go-template prompt body: {{ .issue.identifier }}, {{ .issue.title }}, {{ .issue.description }},
 {{ .issue.labels }}, {{ .run.is_continuation }}, {{ .attempt }}, {{ .issue.url }}]
```

**Query syntax per tracker**: **GitHub**: `query_filter` is appended into a GitHub *search* query: `repo:%s/%s type:issue state:open %s` against `/search/issues` (so any search qualifier works); states are repo **labels**. **Jira**: JQL fragment. **Linear**: an `IssueFilter` JSON object merged (ANDed) with the base query; states are workflow-state names. **GitLab/Gitea**: URL query fragments, allowlist-checked.

**Tracker writes:** exactly two orchestrator-initiated transitions: `in_progress_state` at dispatch time and `handoff_state` after a successful run (suppressed if already terminal). On label-based trackers the adapter swaps the state label; terminal label also closes the issue; missing labels are created on demand.

**Hot reload:** fsnotify on the WORKFLOW.md *parent directory* (catches atomic-rename saves); invalid reloads keep last-known-good; per-field reload table: `polling.interval_ms`, `max_concurrent_agents`, `max_retry_backoff_ms`, `max_sessions`, `max_tokens` are **immediate**; most others "future dispatches"; `db_path` and `server.port` need restart.

**Stall/timeout/retry (defaults):** `turn_timeout_ms` 1h, `read_timeout_ms` 5s, `stall_timeout_ms` 5m (0 disables), `max_retry_backoff_ms` 5m cap on exponential backoff, `max_sessions` 0=unlimited (per-issue session budget), `max_tokens` 0=unlimited (cumulative per-issue token ceiling), `max_concurrent_agents` 10 + a by-state map.

**Cost tracking:** opt-in `token_rates` map keyed by adapter kind (USD per Mtok) → dashboard live-cost + `sortie stats`.

## 4. junhoyeo/contrabass

Symphony-derived, local-first. **WORKFLOW.md format** (verbatim from testdata): same markdown+frontmatter shape as sortie but flatter keys and mustache-style placeholders:

```yaml
---
max_concurrency: 5
poll_interval_ms: 10000
model: anthropic/claude-sonnet
agent_timeout_ms: 600000
stall_timeout_ms: 120000
tracker:
  type: github        # or: internal (alias local)
  owner: example-org
  repo: example-repo
  labels: [bug, agent]
  assignee: bot-user
  token: $GITHUB_TOKEN
agent:
  type: opencode      # codex | opencode | omx | omc
---
# GitHub Workflow
Issue title: {{ issue.title }}
Issue description: {{ issue.description }}
```

**`.contrabass/board/` internal tracker:** `manifest.json` (`{"schema_version":"1","issue_prefix":"CB","next_issue_number":2}`), `issues/CB-1.json`, `comments/CB-1.jsonl` (append-only). Real task file (verbatim):

```json
{
  "id": "CB-1", "identifier": "CB-1",
  "title": "Add godoc comment to EventLog struct",
  "state": "retry",
  "assignee": "issue-cb-1",
  "url": "local://CB-1",
  "tracker_meta": {
    "last_task_id": "001-cb-1-plan", "last_team_event": "run_error",
    "last_worker_id": "coordinator", "team_name": "issue-cb-1",
    "team_phase": "team-plan", "team_status": "retry"
  },
  "created_at": "2026-03-08T06:16:38.652099Z", "updated_at": "2026-03-08T06:24:55.083119Z"
}
```

Board states `todo | in_progress | retry | done` map onto the **five-state run lifecycle**: `Unclaimed → Claimed → Running → RetryQueued → Released` (test-asserted). **BlockedBy gating:** dispatch skips issues whose `blocked_by` list isn't all `done`. **Branch-advance verification**: HEAD SHA recorded at claim; on completion re-check: HEAD unchanged → `"branch_unchanged"` → success is *paused* (`pauseUnverifiedSuccess`), not released; git error → fail open. Retry uses deterministic exponential backoff with hash jitter (reproducible across restarts); orphaned Claimed issues reset to Unclaimed on restart.

## 5. anthropics/claude-code-action (v1)

**(a) Mention handling: `examples/claude.yml` verbatim (core):**

```yaml
name: Claude Code
on:
  issue_comment: { types: [created] }
  pull_request_review_comment: { types: [created] }
  issues: { types: [opened, assigned] }
  pull_request_review: { types: [submitted] }
jobs:
  claude:
    if: |
      (github.event_name == 'issue_comment' && contains(github.event.comment.body, '@claude')) ||
      (github.event_name == 'pull_request_review_comment' && contains(github.event.comment.body, '@claude')) ||
      (github.event_name == 'pull_request_review' && contains(github.event.review.body, '@claude')) ||
      (github.event_name == 'issues' && (contains(github.event.issue.body, '@claude') || contains(github.event.issue.title, '@claude')))
    runs-on: ubuntu-latest
    permissions:
      contents: write
      pull-requests: write
      issues: write
      id-token: write
      actions: read # Required for Claude to read CI results on PRs
    steps:
      - uses: actions/checkout@v6
        with: { fetch-depth: 1 }
      - uses: anthropics/claude-code-action@v1
        with:
          anthropic_api_key: ${{ secrets.ANTHROPIC_API_KEY }}
```

**(b) Automation with prompt: `examples/issue-triage.yml` verbatim (core):**

```yaml
on:
  issues: { types: [opened] }
jobs:
  triage-issue:
    runs-on: ubuntu-latest
    timeout-minutes: 10
    permissions: { contents: read, issues: write }
    steps:
      - uses: actions/checkout@v6
      - uses: anthropics/claude-code-action@v1
        with:
          prompt: "/label-issue REPO: ${{ github.repository }} ISSUE_NUMBER: ${{ github.event.issue.number }}"
          anthropic_api_key: ${{ secrets.ANTHROPIC_API_KEY }}
          github_token: ${{ secrets.GITHUB_TOKEN }}
```

**Syntax:** `claude_args` = raw Claude CLI flags (permissions live here as `--allowedTools`/`--disallowedTools`); `settings` = Claude Code settings.json as JSON string or file path. Full input list: `trigger_phrase` (default `@claude`), `assignee_trigger`, `label_trigger`, `base_branch`, `branch_prefix`, `branch_name_template`, `allowed_bots`, `allowed_non_write_users`, `prompt`, `settings`, auth keys, `github_token`, `claude_args`, `additional_permissions`, `use_sticky_comment`, `use_commit_signing`, `track_progress`, `plugins`. **Mode detection:** no `prompt` → interactive tag mode with a tracking comment; `prompt` present → automation mode. `track_progress: true` forces tag mode (sticky progress comment + full GitHub context injected) but **only** for `pull_request: opened|synchronize|ready_for_review|reopened` and `issues: opened|edited|labeled|assigned`.

---

## 6. SYNTHESIS: open-seed's shipped CI set

**Recommendation: plain `claude-code-action@v1` YAML as the shipped default; gh-aw documented as the hardened upgrade path.** New facts strengthening it: (1) one gh-aw workflow compiles to a ~2,100-line lock file with digest-pinned firewall containers: in a *template* repo that clone-time users must own and re-compile, that is a heavy toolchain dependency; (2) gh-aw `strict: false` workflows refuse to run on public repos; (3) gh-aw's dogfood defaults are Copilot-oriented. What open-seed *should* copy from gh-aw regardless of runner: the **label_command one-shot semantics** (auto-remove label on activation so it re-arms), **provenance markers** (hidden `<!-- seed-workflow-id: … -->` comment + `[ai]` title prefix + forced label), **skip-if-match dedup guards**, **expiry/close-older for recurring reports**, and **budget keys** (map to `--max-turns` + `timeout-minutes` + concurrency).

**Label taxonomy (exact names + transitions).** Two disjoint families, following gh-aw's command/state split and sortie's gate label:

- *State labels (persistent, mirror the task state machine):* `seed:ready` → `seed:working` → `seed:review` → `seed:done`; escape hatch `seed:blocked`. Transitions: human (or planner agent) applies `seed:ready`; dispatcher swaps `ready→working` at dispatch (sortie's `in_progress_state` pattern); agent success swaps `working→review` (sortie's `handoff_state`: agent-finished ≠ human-verified); human merge/close swaps to `seed:done`. The dispatcher only ever *selects* on `seed:ready` and never re-enters `working` items (contrabass's claim gating).
- *Command labels (one-shot, auto-removed on activation):* `cmd:fix`, `cmd:review`, `cmd:docs`. Re-apply to re-trigger.
- *Provenance:* `by:agent` forced onto everything the automation creates.

**Scheduled maintenance workflow:** one `seed-maintenance.yml`, cron `17 5 * * *` (explicit scattered minute: both gh-aw and aeon warn against :00 herd times), `workflow_dispatch`, `concurrency: {group: seed-maintenance, cancel-in-progress: false}`, `timeout-minutes: 20`, prompt-mode claude-code-action doing: close stale `by:agent` issues past expiry, report `seed:blocked` items older than 48h, verify every `seed:working` item has a live branch (contrabass orphan-claim recovery: if not, flip back to `seed:ready`).

**Dispatcher draft (`.github/workflows/seed-dispatch.yml`):**

```yaml
name: Seed Dispatch
on:
  issue_comment: { types: [created] }
  pull_request_review_comment: { types: [created] }
  issues: { types: [opened, assigned, labeled] }
  pull_request_review: { types: [submitted] }

concurrency:
  group: seed-dispatch-${{ github.event.issue.number || github.event.pull_request.number }}
  cancel-in-progress: false

jobs:
  route:
    # Mention routing (@seed) OR command-label routing (cmd:*)
    if: |
      (github.event_name == 'issue_comment' && contains(github.event.comment.body, '@seed')) ||
      (github.event_name == 'pull_request_review_comment' && contains(github.event.comment.body, '@seed')) ||
      (github.event_name == 'pull_request_review' && contains(github.event.review.body, '@seed')) ||
      (github.event_name == 'issues' && github.event.action == 'labeled' &&
        startsWith(github.event.label.name, 'cmd:')) ||
      (github.event_name == 'issues' && github.event.action != 'labeled' &&
        (contains(github.event.issue.body, '@seed') || contains(github.event.issue.title, '@seed')))
    runs-on: ubuntu-latest
    timeout-minutes: 30
    permissions:
      contents: write
      pull-requests: write
      issues: write
      id-token: write
      actions: read
    steps:
      # gh-aw label_command semantics: one-shot: remove the cmd:* label so it re-arms
      - name: Consume command label
        if: github.event.action == 'labeled'
        env: { GH_TOKEN: ${{ secrets.GITHUB_TOKEN }} }
        run: |
          gh issue edit "${{ github.event.issue.number }}" \
            --repo "${{ github.repository }}" \
            --remove-label "${{ github.event.label.name }}" \
            --add-label "seed:working" --remove-label "seed:ready" || true

      - uses: actions/checkout@v6
        with: { fetch-depth: 1 }

      - uses: anthropics/claude-code-action@v1
        with:
          anthropic_api_key: ${{ secrets.ANTHROPIC_API_KEY }}
          github_token: ${{ secrets.GITHUB_TOKEN }}
          trigger_phrase: "@seed"
          track_progress: true
          use_sticky_comment: true
          branch_prefix: "seed/"
          settings: ".claude/ci-settings.json"   # checked-in: sandboxed env, deny-list
          claude_args: |
            --max-turns 30
            --allowedTools "Edit,Read,Grep,Glob,Bash(git:*),Bash(make check),Bash(make test),Bash(gh issue:*),Bash(gh pr:*)"
          prompt: |
            You are open-seed's dispatcher agent. Routing context:
            event=${{ github.event_name }} action=${{ github.event.action }}
            label=${{ github.event.label.name }}
            Follow .github/agents/DISPATCH.md: map cmd:fix -> implement on branch seed/<n>,
            cmd:review -> review only, mention -> answer or act as asked.
            On success: swap label seed:working -> seed:review, prefix any created
            item's title with "[ai] ", add label by:agent, and append the marker
            <!-- seed-workflow-id: seed-dispatch --> to bodies you create.
```

(Write-access gating is claude-code-action's default; the `settings`/`--allowedTools` pair is the plain-YAML stand-in for gh-aw's `tools.bash` allowlist.)

**Corrections to earlier findings:** (1) repo is `github/gh-aw`, and the trigger vocabulary is `slash_command:`/`label_command:`, not `command:`; (2) sortie's `label:agent-ready` is a `query_filter` fragment, while the actual state machine is `active_states`/`in_progress_state`/`handoff_state`/`terminal_states` label swaps; (3) gh-aw budgets are richer than surveyed: per-run credits (default 1000) *and* daily credits *and* turn cap (default 500) *and* per-user rate limits: the plain-YAML default can only emulate the turn cap and timeout, which is the strongest single argument for upgrading to gh-aw; (4) aeon's "checked-in scheduler" is more fragile than the survey implied: it parses its own YAML with bash regex and ships a linter skill for the resulting silent-failure class; open-seed should not imitate that parsing approach, but its debt-model cron catch-up (GitHub delivers ~10% of `*/5` ticks) and circuit breaker are directly reusable.
