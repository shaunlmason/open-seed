# open-seed — Design Options

> Synthesis of a survey of all 180 projects in
> [awesome-agent-orchestrators](https://github.com/andyrewlee/awesome-agent-orchestrators), the
> August 2026 SOTA (harness-native primitives, task tracking, guardrails, methodology), and
> implementation-grade deep dives of the key inspiration projects.
> Category evidence lives in [`docs/research/`](./research/); exact file formats, schemas, and
> algorithms live in [`docs/research/inspirations/`](./research/inspirations/).
> Researched 2026-08-21/22. Where this document and a deep dive disagree, the deep dive
> (source-level evidence) wins; this document is kept in sync.
>
> **Goal restated:** open-seed is a *template repository* teams clone to give new projects
> standardized tooling for multi-agent orchestration, task tracking, and guardrails —
> an orchestration layer that organizes work and executes in teams.

---

## 1. The one finding that settles the positioning

Across every category, the same signal repeats: **orchestration that lives as reviewable repo content outlasts orchestration that lives in an app.**

- vibe-kanban hit 27.9k stars and is sunsetting; humanlayer hit 11.3k and deprecated its code — but humanlayer's *checked-in* research→plan→implement command convention was copied everywhere and survives its product.
- Archon (23.2k★) thrives specifically because its workflows are checked into target repos.
- The Ralph loop — the most influential methodology of the period — is literally one shell line plus markdown conventions in the repo.
- Every mature external app (agent-deck, superset, dmux, octomux, vibe-tree, amux) was *forced* to push a checked-in contract into user repos anyway: setup/teardown scripts, worktree hooks, workspace configs.
- gh-aw and aeon prove a full autonomous runner can be 100% repo content (markdown workflows compiled to committed Actions YAML).

So open-seed's bet — conventions, config, scripts, hooks, and CI checked into a template — is not just viable; it is the *durable* half of the entire ecosystem. The apps churn; the file conventions converge and persist. Corollary: open-seed should be **runner-agnostic**: everything must work with plain Claude Code (or Codex/Gemini) sessions, degrade gracefully to a single human + single agent, and get *better* (not different) when someone layers a TUI/desktop orchestrator or CI runner on top.

## 2. Settled ground (adopt, don't debate)

These have converged hard enough across 180 projects + vendor docs (as of Aug 2026) to treat as decided:

1. **Isolation = git worktree per task/agent**, one branch per unit of work, with a naming convention (worktrees under a dedicated dir, branch `seed/<task-id>`). Containers/devcontainers are an optional second ring for untrusted/full-auto runs, never the default. (~40 of the surveyed projects use exactly this.)
2. **Portable instruction layer = AGENTS.md** (Linux Foundation-stewarded; read natively by every major harness except Claude Code, which imports it via a one-line `CLAUDE.md`), plus `GEMINI.md`/`.cursor/rules` shims only if needed.
3. **Portable capability layer = Agent Skills** (`<dir>/SKILL.md` per agentskills.io) — adopted by Claude Code, Codex, Gemini CLI, Cursor, Copilot, OpenCode, and ~30 more. The cross-client checked-in location converging in the wild is **`.agents/skills/`** (Gemini CLI documents it as an alias; the Vercel `skills` CLI treats it as canonical). Markdown-with-YAML-frontmatter is the universal file format for all agent config.
4. **Tools layer = MCP**, configured in a checked-in `.mcp.json` with env-var expansion for secrets.
5. **Fresh context per task, state on disk.** Conversation memory is a liability; files in the repo are the memory (Ralph doctrine, confirmed by every loop runner and by Anthropic's own context-engineering guidance).
6. **CI is the real guardrail ("backpressure").** Agents merge whatever passes, so a fast, deterministic `make check` + branch protection + required review is the quality system; everything else layers on top.
7. **Agent status vocabulary**: working / blocked(needs-you) / idle / done — the de-facto four-state enum every monitoring tool understands (tmux-ide's `@agent_state` grammar is the concrete wire format).

## 3. Design dimensions and options

### D1. Task-tracking substrate

The central architectural choice — resolved structurally by the plugin decision in §7 (all substrates sit behind one port), leaving only the *default*. Four options:

| Option | Evidence | Pros | Cons |
|---|---|---|---|
| **A. Task-card files in-repo** (one markdown+frontmatter file per task; see the proposed schema in [inspirations/01](./research/inspirations/01-git-native-task-substrates.md)) | The dominant loop-runner convention (ralphex, Automaker, wreckit, gnap, tick-md) | Zero deps; diffable; PR-reviewable; works offline; travels with forks; works with any harness | Atomic claim is only *emulated* (push-wins); fine at ≤~10 writers per remote, breaks beyond |
| **B. Beads** (steveyegge/beads, 26.5k★) — git-native graph tracker: hash IDs, typed dependency edges, `bd ready`, atomic `--claim`, native leases+heartbeats, AGENTS.md integration | What gastown, orc, and ralph-tui all bet on | Purpose-built for parallel agents; ready-work dispatch; crash-safe leases; memory across sessions | Binary dependency (Go + Dolt); Dolt-backed (embedded single-writer by default, server mode for concurrent writers) with an evolving schema — pin a version |
| **C. GitHub Issues + label state machine** (sortie's query-filter + label swaps, lalph, OpenHands) | The CI-automation category standard | Human visibility for free; API-native to Actions; server-atomic assignment; zero new infra | Rate-limited; slow for agent churn; no offline; state lives outside the clone |
| **D. Harness-native task lists** (Claude Code agent teams' shared task list) | Free intra-session coordination | Zero setup | Ephemeral, machine-local; never a system of record |

**Recommendation: A as the shipped default, C as the human mirror, B as the documented upgrade** — matching the plugin ship order (`filecards` → `beads` → `github-issues`, §7). One **canonical state machine** used everywhere (task cards, labels, port `transition` verb): `backlog → ready → in_progress → review → done`, plus `blocked` and `cancelled`, with `review → ready` as the rejection edge. The two-stage done that Automaker/claude-command-center converged on is preserved as `review` (agent-finished) vs `done` (human/verifier-accepted). Key conventions regardless of substrate: **task ↔ branch ↔ worktree 1:1:1 mapping**; hash-based IDs, never sequential counters (two branches consuming a `next_id` is a guaranteed merge conflict — tick-md's design flaw); every closed task must reference evidence (commit SHA, check run) via its receipt (D4.5).

### D2. Orchestration topology

| Option | Evidence | When it wins |
|---|---|---|
| **A. Single-agent Ralph loop** — checked-in loop runner + plan file + fresh session per task | ralphex, ralph-claude-code, dex; 100% repo-native | Solo dev or one workstream; cheapest, most auditable |
| **B. Parallel flat worktrees** — N independent agents, human merges | claude-squad archetype, superset, emdash, all TUIs | Independent tasks; needs only worktree conventions + merge gate |
| **C. Coordinator–worker** — planner decomposes, workers execute, coordinator never codes | orc, kodo (cheap coordinator/expensive workers), gastown Mayor, Claude Code agent teams | Larger efforts; requires the task graph (D1) to be real |
| **D. Ticket-claim blackboard** — no assignment; agents wake and atomically claim ready work | paperclip heartbeats, beads `--claim`, gnap | Most robust to agent death; degrades gracefully to 1 agent; needs atomic claim |

**Recommendation: the paved road is *squad-shaped* (§6) — a small named team with a mission — with A as its degenerate one-member case.** The worktree contract + task substrate + claim convention *are* the topology; the template doesn't hardcode a hierarchy beyond the team files. Escalation path: one agent → parallel worktrees → coordinator role prompt (a subagent/skill, not a daemon) → external orchestrator (gastown, Paperclip, etc.) as optional endgame. Claude Code's native agent teams + `.claude/workflows/` cover C natively for Claude shops — open-seed rides those primitives rather than reimplementing them, while keeping the durable task ledger in the repo (agent-team state is machine-local and ephemeral).

### D3. Plan/spec discipline

Options range from none → Ralph-thin (PROMPT.md + fix_plan.md) → plan-as-gated-artifact (Fusion, Ivy-Tendril, dex) → full SDD (spec-kit, BMAD).

**Recommendation: thin, mandatory, gated.** The single highest-leverage quality convention found across all categories is **plan-as-gated-artifact**: every non-trivial task produces a committed plan file (steps, file scope, acceptance criteria, `## Validation Commands`) that is reviewed — by human or reviewer agent — *before* execution, and implementation is reviewed *against that same file*. The concrete grammar (ralphex's parser rules + frankbria's Optional-section semantics + the dex skip convention) is specified in [inspirations/02](./research/inspirations/02-ralph-loop-implementations.md). One deliberate departure from precedent: in ralphex/dex, `## Validation Commands` is advisory — the *agent* is told to run it; open-seed's loop runner and pre-merge gate also execute it **mechanically** (martin-loop's fresh-evidence rule), so completion is evidence, not assertion. Each plan is linked from its task card (`plan:` field) and archived to `plans/completed/` on close. Full SDD frameworks (spec-kit/BMAD) are contested and heavy; they remain optional adapters, not the core.

### D4. Guardrails stack

Defense in depth; every layer is checked-in-able. The concrete `guardrails.yaml` schema and evidence-receipt schema are specified in [inspirations/03](./research/inspirations/03-governance-and-gates.md).

1. **Policy file** (`.seed/guardrails.yaml`): protected-path globs (restriction-only, claudexor-style — matching changes require a human decision), auto-merge allowlist vs denylist (loop-engineering's `gate.yaml` split), `max_files_per_change`, budgets (runs/day, tokens/day, attempts — with the honest caveat in §8-R6), and **named autonomy tiers** (L1 report-only / L2 assisted-in-worktree / L3 unattended-with-gates). Loop-engineering comes closest to this today (`gate.yaml` + markdown budget tables + operational L1/L2/L3 convention); nobody yet ships enforced tiers as one machine-readable, hook-enforced file — that combination is the open-seed differentiator.
2. **Harness enforcement**: `.claude/settings.json` permissions (allow/ask/deny) + hooks (PreToolUse denials of destructive commands and protected paths; Stop/TaskCompleted hooks that block completion until lint/tests pass — the OpenHands `hooks.json` / Claude Code Stop-hook pattern), mirrored minimally for Codex/OpenCode. Harness enforcement is best-effort per harness; the layers below are the backstop.
3. **Git enforcement**: pre-commit hooks (gitleaks, contract checks) and the **blocking `pre-merge.d/` gate** — notably, no surveyed tool actually blocks on pre-merge hooks (dmux's `pre_merge` is detached/non-blocking), so this is novel; branch protection + required checks + merge queue documented; agents never push default branches.
4. **CI enforcement**: one fast `make check`; CI also validates the orchestration files themselves (task-card/plan lint, workflow validate, skills-lock `--frozen` check, guardrails schema check, append-only run-log check, fan-out drift check per §8-R1).
5. **Verification separation**: no agent verifies its own work (antfarm, kodo, loki-mode; squad's reviewer-lockout goes further — on rejection the author is locked out and a different agent revises). Ship a read-only **reviewer role** distinct from the implementer, and require **evidence receipts** on close: a per-task JSON with a `facts` / `assessments` / `honesty` split (loki-mode), headline computed from facts alone (fresh command exit codes + non-empty diff + no protected paths → VERIFIED; every skipped check listed as a gap — silence never reads as pass). Refuse "done" on empty diffs.
6. **Secrets**: never in agent-readable config; env/`settings.local.json`/`.mcp.json` expansion; gitleaks in pre-commit + CI; secrets *names* documented, values never; "secrets never touch the model" posture documented.

### D5. Memory conventions

Converged pattern worth shipping: a committed **`LEARNINGS.md`** (build/test commands, discoveries — agent-appendable; the Ralph-family "AGENT.md", renamed to avoid confusion with `AGENTS.md`), **`DEADENDS.md`** (failed approaches so fresh sessions don't retry them — dex; derivable from run-log entries with failure status, agent-annotated with reasons), append-only **`run-log.jsonl`** (append-only-ness verified by CI), and optional `decisions/` (ADR-style; squad's `decisions.md` with `merge=union` in `.gitattributes` is the merge-safe precedent). Rules about *who may mutate what* (agent may append, never rewrite; humans only edit guardrails) matter more than the filenames. Personal-assistant-style `MEMORY.md`/daily-notes rollups are overkill for a project template — skip.

### D6. Lifecycle/workspace contract

The full contract (hook names, env vars, exit-code semantics, per-tool shims) is specified in [inspirations/07](./research/inspirations/07-lifecycle-contracts.md). Summary: `.seed/hooks/` with **`setup`, `run`, `teardown`** single-file hooks plus `post-create.d/` and `pre-merge.d/` run-parts dirs; executable-bit-as-opt-in, spawned without a shell, context via `SEED_*` env vars only; setup/teardown advisory (never strand a worktree), **pre-merge gating** (the one blocking set); a root **`.worktreeinclude`** adopting agent-deck's format verbatim (gitignore-syntax; copy only matched AND gitignored files; never overwrite — it is already a de-facto standard matching Claude Code Desktop semantics). Ship 2–5-line shims for superset/amux/vibe-tree/dmux/agent-deck/octomux so any of those tools drives the same scripts.

### D7. CI/automation layer

Actions-native proves out fully (gh-aw, claude-code-action, aeon). Ship: a mention/label **dispatcher workflow**, a scheduled maintenance workflow (stale-claim recovery, expiry cleanup, blocked-item reporting), and a PR-review workflow — built on `anthropics/claude-code-action@v1` as the pragmatic default (write-access trigger gating, per-command Bash allowlists, branch-not-PR default), with **gh-aw as the documented upgrade** for its read-only-agent + safe-outputs + sanitization + egress-firewall architecture (and for real budget enforcement — see §8-R6). Label taxonomy with two disjoint families (state labels `seed:ready|working|review|done|blocked` mirroring D1's state machine; one-shot command labels `cmd:*` auto-removed on activation, gh-aw's `label_command` semantics) plus a forced `by:agent` provenance label. Provenance conventions everywhere: `[ai]` title prefixes, hidden `<!-- seed-workflow-id -->` body markers, sticky progress comments. Scheduled crons use scattered minutes (never `:00`) and assume missed ticks (GitHub delivers a fraction of short-interval schedules — aeon's debt-model catch-up). Draft workflows in [inspirations/06](./research/inspirations/06-ci-native-automation.md).

### D8. Skills/agents packaging & harness portability

- Portable core: `AGENTS.md` + a source-of-truth skills tree + `.mcp.json`; per-harness **fan-out by copy, not symlink** (symlinks break on Windows checkouts, zip downloads, and some CI sandboxes) into `.claude/skills/` and `.agents/skills/`, with drift policed by an offline check (§8-R1).
- Agent role definitions as markdown-with-frontmatter in `.seed/agents/*.md` (dual-format: Claude Code subagent fields + sub-agents-skills' `run-agent`/`permission: read-only|safe-edit|yolo`/`## Done When` — one file serves both; schema in [inspirations/05](./research/inspirations/05-skills-packaging.md)), fanned out unchanged to `.claude/agents/`.
- Shared skills across cloned repos: **manifest + lockfile** (`seed.yaml` + `seed.lock`, skillfold semantics — SHA+sha256 pins, `install --frozen` in CI, managed-directory-only pruning, timestamp-free alphabetized entries for clean merges); skill updates are PRs with injection review.
- Workflows-as-files: **YAML with a markdown-prompt escape hatch** (`prompt: |` inline or `prompt_file:` for long prompts — the Archon middle path; full step schema and 13 validate rules in [inspirations/04](./research/inspirations/04-workflow-as-config.md)), with a `seed workflow validate` preflight in CI and a **mock harness** so workflows are testable without credentials; `.claude/workflows/` additionally for Claude-native dynamic workflows.

## 4. Repository layout

Reconciled with all decisions (§6 teams, §7 backend plugins) and the deep-dive contracts. Committed config lives under `.seed/`; runtime state (worktrees, run scratch, checkpoints) is excluded via `.git/info/exclude` (orc's trick — invisible to both history and `.gitignore` diffs).

```
open-seed/
├── AGENTS.md                  # source of truth for agent instructions (+ managed blocks)
├── CLAUDE.md                  # "@AGENTS.md" import + Claude-specific notes
├── README.md
├── Makefile                   # make check — the one fast backpressure command
├── .mcp.json
├── .worktreeinclude           # gitignored-file propagation into new worktrees (agent-deck format)
├── .gitattributes             # merge=union on history-bearing files (decisions/, mail/)
├── seed.yaml / seed.lock      # skills manifest + lockfile (skillfold semantics)
├── .claude/
│   ├── settings.json          # permissions, hooks (Stop gate, PreToolUse denials), plugin decls
│   ├── agents/                # fan-out copies of .seed/agents/ (do not edit here)
│   ├── skills/                # fan-out copies of skills/ (do not edit here)
│   └── workflows/             # optional Claude dynamic workflows
├── .agents/skills/            # cross-harness fan-out (agentskills.io convention)
├── skills/                    # Agent Skills source of truth, harness-neutral
├── .seed/                     # the orchestration contract (committed)
│   ├── config.toml            # active backend, defaults, roles→runtime map
│   ├── guardrails.yaml        # autonomy tiers, budgets, protected paths, allowlists
│   ├── version                # protocol version int; refuse-if-newer (gnap)
│   ├── agents/                # role definitions (planner/implementer/reviewer/tester)
│   ├── teams/                 # squad definitions: mission, members, scope, rituals, tier
│   ├── tasks/                 # task cards (filecards backend; other backends ignore)
│   ├── plans/                 # gated plan files (grammar per inspirations/02)
│   │   └── completed/
│   ├── receipts/              # per-task evidence receipts (facts/assessments/honesty)
│   ├── memory/                # LEARNINGS.md, DEADENDS.md, run-log.jsonl, decisions/
│   ├── mail/                  # optional file mailboxes (one per agent, flock+append)
│   ├── backends/              # coordination-backend plugins (filecards ships in-template)
│   ├── backends.lock.json     # SHA+hash pins for installed backend plugins
│   ├── workflows/             # workflow YAML (repo overrides bundled defaults by name)
│   ├── port-schema/           # JSON Schemas for the task-port contract (conformance tests)
│   └── hooks/                 # setup / run / teardown / post-create.d/ / pre-merge.d/
├── scripts/
│   ├── seed                   # the CLI shim: task/mail/backend/workflow/hooks subcommands
│   ├── loop.sh                # Ralph loop: dual-gate exit, circuit breaker, budgets
│   └── validate.sh            # lint all orchestration files (also run in CI)
├── .github/
│   ├── workflows/             # check, seed-dispatch, seed-maintenance, pr-review, validate
│   └── ISSUE_TEMPLATE/        # machine-parseable forms + label taxonomy
└── docs/                      # this study + conventions handbook
```

## 5. What open-seed would be that nothing else is

Closest prior art, per the research: **tutti** (committed org-code TOML — with real caveats: index-based step deps, role→runtime-only mapping; see [inspirations/04](./research/inspirations/04-workflow-as-config.md)) + **beads** (committed task graph) + **loki-mode's gates** (deterministic verification with evidence receipts) + **loop-engineering** (checked-in loop conventions with autonomy tiers) + **bradygaster/squad** (repo-resident team charters). *No project combines these in template form.* The unclaimed gaps open-seed can own:

1. A **checked-in autonomy-tier + guardrails vocabulary** enforced by hooks and CI (apps keep this in settings; loop-engineering documents it; nobody versions-and-enforces it as one file).
2. The **task↔plan↔evidence chain**: task card → gated plan → implementation → fresh-evidence receipt, all as diffable files.
3. A **blocking pre-merge gate** in the lifecycle contract (every surveyed tool's merge hooks are advisory).
4. **Runner-agnostic degradation**: the same repo works with a lone human, one Claude Code session, Claude agent teams, a Ralph loop in CI, or any of the 60+ external orchestrators surveyed — because the contract is files, with documented shims to each tool's convention.

## 6. Team layer: organizing work the squad-model way

**Direction (2026-08-22):** open-seed's orchestration layer organizes work and executes in *teams*, modeled on the Spotify Squad Model (squads / tribes / chapters / guilds, "aligned autonomy"). The research maps onto it cleanly — several surveyed projects independently reinvented pieces of it (bradygaster/squad's `.squad/` charters+routing, tutti's roles+scope globs, kodo's team.json, gastown's crews, corellis/opengoat's org charts, Claude Code's native agent teams):

| Squad-model concept | open-seed realization | Precedent |
|---|---|---|
| **Squad** — small, autonomous, cross-functional team owning a mission | Checked-in team definition `.seed/teams/<squad>.yaml`: `mission`, `members` (refs to agent-role defs, mixing humans and agents — gnap's `type: ai\|human`), `scope` (code-ownership globs), `backlog` (task-card filter), `rituals` (workflow refs: e.g. scheduled triage), `autonomy_tier` (L1/L2/L3 per squad) | squad's `team.md` roster + charters; tutti `[[agent]] scope`; kodo team.json |
| **Tribe** — collection of squads in one area | The repo (or org overlay for multi-repo); tribe-level `guardrails.yaml` cascades down, squads may only tighten, never loosen | qm's org-overlay repos; corellis fleet governance |
| **Chapter** — competency line across squads (all reviewers, all testers) with a lead maintaining standards | The agent-role definitions themselves (`.seed/agents/reviewer.md` etc.) are chapter artifacts: one canonical definition per role shared by every squad; the chapter lead is the human CODEOWNER of that file, so role changes are reviewed by the person who owns the craft | sub-agents-skills role files; opengoat role-differentiated skills; antfarm shared role files |
| **Guild** — voluntary interest community sharing practice | The shared skills library (`skills/` + manifest/lockfile), publishable across repos/orgs | skillfold; Claude Code plugin marketplaces |
| **Mission/OKR alignment** | Goal ancestry on every task card (`parent`/goal links tracing to the squad mission) — agents always see the "why" | Paperclip's mandatory goal ancestry; beads epics |
| **Autonomy within alignment** | Squads own *how* (agents choose implementation; squad picks its workflows); tribe owns *what* (missions, guardrails, quality bar via CI + evidence receipts) | kodo's "Tell them WHAT, never HOW"; Fusion automation levels |

The squad model's documented failure modes (per the [ideaplan case study](https://www.ideaplan.io/case-studies/spotify-squad-model): "the model as described in the whitepaper never fully existed in practice" — even Spotify moved away from it) map to mitigations that are *mechanical* in an agent org where they were only aspirational in a human one:

- **Fragmentation** (excessive autonomy → technical inconsistency across codebases): tribe-level guardrails, lint/test config, and chapter role definitions are enforced by CI and hooks — agents cannot drift on standards the way human squads did, because the standards are executable.
- **Chapter-lead dysfunction** (line-management duties consuming the lead's IC time): the chapter lead here is just a CODEOWNERS entry on a role-definition file — reviewing changes to the craft's canonical definition, with zero people-management burden.
- **Tribal silos / resistance to cross-tribe work**: cross-squad dependencies become typed `blocks`/`waits-for` edges between task cards plus mailbox messages, visible in `ready` queries rather than discovered in standups; Spotify's "internal open source" practice maps directly — any squad's agents may PR into another squad's scope, subject to that squad's review gate.
- **Guild decline** (voluntary participation failing at scale): the guild is a versioned skills library with a manifest/lockfile — it doesn't depend on volunteer energy to stay alive.
- **Alignment** ("each musician improvises, but they are all playing the same song in the same key"): squad missions connect to repo-level objectives via goal ancestry on every task card, validated in CI.

The article's real lesson — adopt the principles (aligned autonomy, small mission-owning teams, horizontal craft standards), not the org chart — is exactly the posture for the template: team files are conventions a project tunes, not a mandated hierarchy. Tribe sizing concerns (Dunbar's number) translate to a practical cap on squads-per-repo before splitting into an org overlay. In the minimal clone, exactly one squad exists ("core", containing the human and a default agent trio) — the squad layer must cost nothing until a second squad is added.

## 7. Decisions made

1. **Coordination backends are plugins (decided 2026-08-21).** Whatever coordination server/backend is used — markdown task files, beads, GitHub Issues, Paperclip, Gas Town, or anything future — it is implemented as a **plugin behind a stable port interface**, and nothing else in the template (scripts, skills, hooks, CI, agent instructions) may talk to a backend directly. This goes beyond a fixed adapter set: the seed exposes a plugin system — packaging (manifest + implementation), checked-in declaration of the active backend, capability negotiation (plugins declare which port operations they support so the seed degrades gracefully — e.g. a markdown backend without true atomic claims), version pinning with a lockfile, and a trust model for third-party plugins (pinned SHAs, review-before-install, plugin output treated as untrusted input to agents). Precedents: sortie/lalph/ralph-tui's pluggable task sources, ORCH's shell adapter, ouijit's JSON task CLI, Claude Code plugins/marketplaces, skillfold's manifest+lockfile, thurbox's capability-gated plugins. The concrete spec lives in [`docs/research/10-org-control-planes.md`](./research/10-org-control-planes.md) Part 5. Summary of the v1 proposal: a **JSON-over-CLI port** — every script, skill, hook, and agent uses only `seed task <verb> --json`, which shells out to the active backend plugin at `.seed/backends/<name>/bin/seed-backend`; nine required verbs (create, ready, get/list, claim, release, transition, close-with-blocker-cascade, comment/attach-evidence, event-append) plus optional capabilities (lease-renew, ancestry, deps, event-emit, wake, budget, watch); exit code 2 reserved for claim contention; manifest (`backend.toml`) declares capabilities (`atomic_claim: native|emulated|none`, offline, budget) for graceful degradation; `.seed/backends.lock.json` pins installed plugins by SHA+hash; MCP is the v2 transport via one generic wrapper over the same CLI. Ship order: `filecards` (zero-dep reference impl) → `beads` (atomicity + Gas Town/Gas City adjacency) → `github-issues` (the bridge to multica/Fusion/humans), with a `paperclip` REST adapter as the deferred fourth.
2. **Teams are the organizing unit (direction, 2026-08-22).** Work is organized and executed in squads per §6, with team definitions as checked-in files. The one-member squad is the degenerate default so the layer costs nothing for solo use.

## 8. Risks, gotchas, and mitigations

Known sharp edges the design must handle (each mitigation is part of the template, not an aspiration):

- **R1 — Fan-out drift.** Skills/agents exist in a source tree *and* per-harness copies; they will diverge silently. Mitigation: fan-out is generated only by `scripts/seed sync`; `seed sync --check` (offline, hash-based — skillfold's `check`) runs in CI and fails on drift; fanned directories carry a "do not edit here" marker file.
- **R2 — Windows portability.** The hooks contract relies on executable bits (lost on some Windows checkouts) and the layout forbids symlinks for the same reason. Mitigation: fan-out by copy everywhere; `scripts/seed hooks run <name>` as an interpreter-invoking fallback runner; document `git config core.fileMode` caveats; no path components that differ only by case.
- **R3 — Untrusted content is everywhere.** Task-card bodies, issue text, PR comments, mailbox messages, and backend-plugin output are all injected into agent context — every one is a prompt-injection vector. Mitigation: the port shim schema-validates and sanitizes backend output (strip control chars/tool-call-shaped markup); AGENTS.md instructs agents to treat task/mail content as data, not instructions; CI workflows follow claude-code-action's write-access trigger gating and, when upgraded, gh-aw's sanitization + safe-outputs; backend plugins are SHA-pinned with review-before-update (a malicious backend can fabricate claims and inject prompts — §7's trust model).
- **R4 — Emulated claims have a ceiling.** Push-wins claiming on filecards is honest only up to ~10 concurrent writers per remote; beyond that, contention wastes work. Mitigation: the capability matrix makes this explicit (`atomic_claim: emulated`); the maintenance workflow reports contention rates; the documented upgrade is the beads backend (native claims + leases). Never pretend the file backend is atomic.
- **R5 — Committed coordination state churns git history.** Task cards, run logs, and mail create commit noise and conflict risk. Mitigation: one file per task/recipient (never shared counters or single-file boards — the tick-md lesson); `merge=union` on append-only files; per-agent inbox + scribe merge for shared docs (squad's pattern); runtime state (worktrees, checkpoints, scratch) in `.git/info/exclude`, never committed; hash IDs so parallel creates never collide.
- **R6 — Budgets are advisory without a server.** A repo cannot hard-stop spend across machines; only run-local caps (max-iterations, wall-clock, per-run token counts from harness output) are enforceable. Mitigation: state this honestly in guardrails.yaml docs; enforce what is enforceable locally (loop budgets, attempt caps, circuit breaker); post-hoc accounting in the run log; point teams needing hard org-wide stops at gh-aw (per-run/daily credit caps) or a control plane (Paperclip) via the backend/event seams.
- **R7 — Harness churn.** Agent teams are experimental, hook vocabularies change, and vendors absorb orchestration features fast (Feb–Aug 2026 brought agent teams, dynamic workflows, five hook types). Mitigation: the harness-neutral core (AGENTS.md, skills, port CLI, hooks contract) carries the semantics; harness-specific files are generated shims; CI never depends on experimental harness features.
- **R8 — Template drift after clone.** GitHub template repos cannot push updates to clones. Mitigation: keep the update surface small and versioned (`.seed/version`, `seed.lock`, `backends.lock.json`); provide `seed upgrade` guidance (diff against the template's tagged releases); distribute the *evolving* parts (skills, role defs, workflows) additionally as a Claude Code plugin/marketplace so they update independently of the cloned structure (ties to §9 Q5).
- **R9 — Convention without enforcement rots.** Every convention in this doc that lacks a validator becomes stale documentation (loop-engineering's "State Rot" failure mode). Mitigation: `scripts/validate.sh` + the validate CI workflow check *every* checked-in orchestration artifact (cards, plans, workflows, guardrails, team files, lockfiles, run-log append-only-ness); the doc treats "shipped convention" and "shipped validator" as one deliverable.

## 9. Open questions to resolve next

1. **Harness posture:** Claude-Code-first with portable shims (recommended — richest primitive set: hooks, subagents, sandbox, workflows), or strictly harness-neutral from day one (more work, lower ceiling)?
2. **How much automation ships enabled?** Conventions + scripts only, vs. CI workflows live from clone (dispatcher, scheduled maintenance). Current recommendation: ship the workflows in-tree but inert until secrets are configured, `claude-code-action` as default with gh-aw documented (per D7); needs your confirmation.
3. **Scope of the loop runner:** ship `loop.sh` (crosses from "conventions" into "runtime"; the exit-detection design is fully specified in inspirations/02), or document the pattern and point at ralphex/dex? Current lean: ship it — it's the reference consumer of the port CLI.
4. **Language/stack coupling:** is open-seed language-agnostic (Makefile contract only), or does it ship opinionated stacks (e.g. a TS variant with lint/test wired)?
5. **Distribution model:** GitHub template repo (clone-and-drift), a Claude Code plugin/marketplace (updatable), or both — with the plugin carrying skills/agents/hooks and the template carrying structure? (Interacts with R8.)
6. **Mailbox in v1:** the file-mailbox format is specified (inspirations/08) and daemon-free, but it's one more convention to validate — include in v1 or defer until the first multi-squad user?
