# open-seed — Design Options

> Synthesis of a survey of all 180 projects in
> [awesome-agent-orchestrators](https://github.com/andyrewlee/awesome-agent-orchestrators) plus the
> August 2026 SOTA (harness-native primitives, task tracking, guardrails, methodology).
> Per-category evidence lives in [`docs/research/`](./research/). Researched 2026-08-21.
>
> **Goal restated:** open-seed is a *template repository* teams clone to give new projects
> standardized tooling for multi-agent orchestration, task tracking, and guardrails.

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

These have converged hard enough across 180 projects + vendor docs to treat as decided:

1. **Isolation = git worktree per task/agent**, one branch per unit of work, with a naming convention (`<repo>-worktrees/<slug>`, branch `seed/<task-id>` or similar). Containers/devcontainers are an optional second ring for untrusted/full-auto runs, never the default. (~40 of the surveyed projects use exactly this.)
2. **Portable instruction layer = AGENTS.md** (Linux Foundation-stewarded, read natively by every harness except Claude Code) **+ a one-line `CLAUDE.md` that imports it**, plus `GEMINI.md`/`.cursor/rules` shims only if needed.
3. **Portable capability layer = Agent Skills** (`skills/<name>/SKILL.md` per agentskills.io) — now adopted by Claude Code, Codex, Gemini CLI, Cursor, Copilot, OpenCode, and ~30 more. Markdown-with-YAML-frontmatter is the universal file format for all agent config.
4. **Tools layer = MCP**, configured in a checked-in `.mcp.json` with env-var expansion for secrets.
5. **Fresh context per task, state on disk.** Conversation memory is a liability; files in the repo are the memory (Ralph doctrine, confirmed by every loop runner and by Anthropic's own context-engineering guidance).
6. **CI is the real guardrail ("backpressure").** Agents merge whatever passes, so a fast, deterministic `make check` + branch protection + required review is the quality system; everything else layers on top.
7. **Agent status vocabulary**: working / blocked(needs-you) / idle / done — the de-facto four-state enum every monitoring tool understands.

## 3. Design dimensions and options

### D1. Task-tracking substrate

The central architectural choice. Four options, not mutually exclusive:

| Option | Evidence | Pros | Cons |
|---|---|---|---|
| **A. Markdown task/plan files in-repo** (ralphex plan format, Automaker's `.automaker/features/`, wreckit per-item dirs, gnap JSON entities) | The dominant loop-runner convention | Zero deps; diffable; PR-reviewable; progress = checkbox diffs; works with any harness | Degrades under parallelism: no locking, no dependency queries, merge conflicts |
| **B. Beads** (steveyegge/beads, 26.5k★) — git-native graph tracker: hash IDs, dependency edges, `bd ready`, atomic `--claim`, AGENTS.md integration | What gastown, orc, and ralph-tui all bet on | Purpose-built for parallel agents; ready-work dispatch; memory across sessions | Binary dependency; storage layer still churning (JSONL→Dolt migration); pin a version |
| **C. GitHub Issues + label state machine** (sortie's `label:agent-ready`, lalph, OpenHands) | The CI-automation category standard | Human visibility for free; API-native to Actions; zero new infra | Rate-limited; slow for agent churn; state lives outside the clone |
| **D. Harness-native task lists** (Claude Code agent teams' shared task list) | Free intra-session coordination | Zero setup | Ephemeral, machine-local; never a system of record |

**Recommendation: layered A + C, with B as an opt-in module.** Ship markdown task files with a defined state machine (`inbox → planned → in-progress → in-review → done`, plus the two-stage done that Automaker/claude-command-center converged on: *agent-finished ≠ human-verified*) as the zero-dependency floor; a label taxonomy + issue templates syncing GitHub Issues as the human mirror; and a documented `bd` bootstrap for teams that hit the parallelism wall. Key convention regardless of substrate: **task ↔ branch ↔ worktree 1:1:1 mapping**, and every closed task must reference evidence (commit SHA, check run).

### D2. Orchestration topology

| Option | Evidence | When it wins |
|---|---|---|
| **A. Single-agent Ralph loop** — checked-in `loop.sh` + plan file + fresh session per task | ralphex, ralph-claude-code, dex; 100% repo-native | Default. Solo dev or one workstream; cheapest, most auditable |
| **B. Parallel flat worktrees** — N independent agents, human merges | claude-squad archetype, superset, emdash, all TUIs | Independent tasks; needs only worktree conventions + merge gate |
| **C. Coordinator–worker** — planner decomposes, workers execute, coordinator never codes | orc, kodo (cheap coordinator/expensive workers), gastown Mayor, Claude Code agent teams | Larger efforts; requires the task graph (D1) to be real |
| **D. Ticket-claim blackboard** — no assignment; agents wake and atomically claim ready work | paperclip heartbeats, beads `--claim`, gnap | Most robust to agent death; degrades gracefully to 1 agent; needs atomic claim |

**Recommendation: ship A as the paved road, with conventions that make B–D emerge naturally.** The worktree contract + task substrate + claim convention *are* the topology; the template shouldn't hardcode a hierarchy. Document the escalation path: one agent → parallel worktrees → coordinator role prompt (a subagent/skill, not a daemon) → external orchestrator (gastown, etc.) as optional endgame. Claude Code's native agent teams + `.claude/workflows/` cover C natively for Claude shops — open-seed should ride those primitives, not reimplement them.

### D3. Plan/spec discipline

Options range from none → Ralph-thin (PROMPT.md + fix_plan.md) → plan-as-gated-artifact (Fusion, Ivy-Tendril, dex) → full SDD (spec-kit, BMAD).

**Recommendation: thin, mandatory, gated.** The single highest-leverage quality convention found across all categories is **plan-as-gated-artifact**: every non-trivial task produces a committed plan file (steps, file scope, acceptance criteria, `## Validation Commands`) that is reviewed — by human or reviewer agent — *before* execution, and implementation is reviewed *against that same file*. Adopt the ralphex plan format (checkbox tasks + validation commands in the file + `plans/completed/` archive + Optional-section semantics). Full SDD frameworks (spec-kit/BMAD) are contested and heavy; make them optional adapters, not the core.

### D4. Guardrails stack

Defense in depth; every layer is checked-in-able:

1. **Policy file** (`guardrails.yaml` or similar): protected-path globs / `never_touch` (ralphy, claudexor), allowed-tools, auto-merge allowlist vs denylist (loop-engineering's `gate.yaml` split), budgets (max iterations/turns/USD — fractal's run/iteration/step three-level caps), and **named autonomy tiers** (L1 report-only / L2 assisted / L3 unattended-with-gates — loop-engineering; or supervised/auto/full — t3code, zeroclaw, repomon). Nobody in the ecosystem checks this vocabulary into the repo today — an open-seed differentiator.
2. **Harness enforcement**: `.claude/settings.json` permissions (allow/ask/deny) + hooks (PreToolUse deny destructive commands, Stop/TaskCompleted hooks that block completion until lint/tests pass — the OpenHands `hooks.json` / Claude Code Stop-hook pattern), mirrored minimally for Codex/OpenCode.
3. **Git enforcement**: pre-commit/pre-merge hook scripts (dmux's `.dmux-hooks` slot generalized); branch protection + required checks + merge queue documented; agents never push default branches.
4. **CI enforcement**: one fast `make check`; CI validates orchestration files themselves (plan format lint, workflow validate, skills-lock frozen check, append-only run-log check).
5. **Verification separation**: no agent verifies its own work (antfarm, kodo, loki-mode). Ship a read-only **reviewer subagent** definition distinct from the implementer, and require **evidence receipts**: done = fresh test output + diff SHA recorded in the task file (loki's facts-vs-opinions receipts, martin-loop's fresh-evidence rule, symphony's proof-of-work bundles). Refuse "done" on empty diffs.
6. **Secrets**: never in agent-readable config; env/`settings.local.json`/`.mcp.json` expansion; gitleaks in pre-commit + CI; document the "secrets never touch the model" posture.

### D5. Memory conventions

Converged pattern worth shipping wholesale: committed **`AGENT.md`-style learnings file** (build/test commands, discoveries — agent-appendable), **`DEADENDS.md`** (failed approaches so fresh sessions don't retry them — dex), append-only **`run-log.jsonl`** (append verified by CI), and optional `decisions/` (ADR-style). Rules about *who may mutate what* (agent may append, never rewrite) matter more than the filenames. Personal-assistant-style `MEMORY.md`/daily-notes rollups are overkill for a project template — skip.

### D6. Lifecycle/workspace contract

Merge the three independent inventions (superset `.superset/config.json`, octomux `hooks/<event>.d/`, amux setup/run/archive): a dot-directory (e.g. `.seed/`) with **`setup`, `run`, `teardown`** scripts plus optional `pre-merge.d/`/`post-create.d/` hook dirs, a `.worktreeinclude` (propagate gitignored env files into new worktrees — agent-deck), and documented env vars carrying task context (`SEED_TASK_ID`, `SEED_TASK_DESC`, `SEED_PORT` — ouijit/amux patterns). Any human, agent, TUI, or CI job can execute these; that's the interop surface with the entire external-tool ecosystem.

### D7. CI/automation layer

Actions-native proves out fully (gh-aw, claude-code-action, aeon). Ship: a mention/label **dispatcher workflow** (run-gemini-cli's `gemini-dispatch.yml` pattern), a scheduled maintenance workflow, and a PR-review workflow — built on `anthropics/claude-code-action@v1` as the pragmatic baseline (write-access trigger gating, per-command Bash allowlists, branch-not-PR default), with **gh-aw as the documented upgrade** for its read-only-agent + safe-outputs + sanitization + egress-firewall architecture. Provenance conventions everywhere: `[ai]` title prefixes, forced labels, sticky progress comments.

### D8. Skills/agents packaging & harness portability

- Portable core: `AGENTS.md` + `skills/` (Agent Skills standard) + `.mcp.json`; per-harness shims (`CLAUDE.md` import; skills fanned out or symlinked into `.claude/skills/`, `.agents/skills/`, `.gemini/skills/`).
- Agent role definitions as markdown-with-frontmatter (`agents/*.md` with `model`, `permission: read-only|safe-edit|yolo`, `## Done When` — sub-agents-skills pattern), doubling as Claude Code subagents in `.claude/agents/`.
- If open-seed curates shared skills across many cloned repos: **manifest + lockfile** (skillfold pattern — SHA+sha256 pins, `--frozen` CI check) rather than copy-paste; treat skill updates as PRs with injection review.
- Workflows-as-files: keep any multi-step pipeline definitions as markdown/YAML with a `validate` preflight runnable in CI (crewplane/agent-runbook/Archon compile-and-validate pattern); use `.claude/workflows/` for Claude-native dynamic workflows.

## 4. Strawman repository layout

```
open-seed/
├── AGENTS.md                  # source of truth for agent instructions
├── CLAUDE.md                  # "@AGENTS.md" import + Claude-specific notes
├── README.md
├── Makefile                   # make check — the one fast backpressure command
├── .mcp.json
├── .claude/
│   ├── settings.json          # permissions, hooks (Stop gate, PreToolUse denials), plugin decls
│   ├── agents/                # implementer / reviewer / planner subagent defs
│   ├── skills/                # → or fan-out from skills/
│   └── workflows/             # optional Claude dynamic workflows
├── skills/                    # Agent Skills standard, harness-neutral
├── seed/                      # the orchestration contract
│   ├── guardrails.yaml        # autonomy tier, budgets, protected paths, allowlists
│   ├── tasks/                 # task cards (markdown+frontmatter state machine)
│   │   └── done/
│   ├── plans/                 # gated plan files (ralphex format)
│   │   └── completed/
│   ├── memory/                # AGENT-learnings.md, DEADENDS.md, run-log.jsonl
│   └── hooks/                 # setup / run / teardown / pre-merge.d/ + .worktreeinclude
├── scripts/
│   ├── task.sh                # create/claim/status/close over seed/tasks (JSON out)
│   ├── worktree.sh            # new/list/cleanup with naming convention
│   ├── loop.sh                # Ralph loop: dual-gate exit, circuit breaker, max-iter
│   └── validate.sh            # lint orchestration files (also run in CI)
├── .github/
│   ├── workflows/             # check, agent-dispatch, scheduled-maintenance, pr-review
│   └── ISSUE_TEMPLATE/        # machine-parseable forms + label taxonomy
└── docs/                      # this study + conventions handbook
```

## 5. What open-seed would be that nothing else is

Closest prior art, per the research: **tutti** (committed org-code declaring roles/scopes/workflows/gates) + **beads** (committed task graph) + **loki-mode's gates** (deterministic verification with evidence receipts) + **loop-engineering** (checked-in loop conventions with autonomy tiers). *No project combines these in template form.* The unclaimed gaps open-seed can own:

1. A **checked-in autonomy-tier + guardrails vocabulary** enforced by hooks and CI (everyone has this in app settings; nobody versions it in the repo).
2. The **task↔plan↔evidence chain**: task card → gated plan → implementation → fresh-evidence receipt, all as diffable files.
3. **Runner-agnostic degradation**: the same repo works with a lone human, one Claude Code session, Claude agent teams, a Ralph loop in CI, or any of the 60+ external orchestrators surveyed — because the contract is files.

## 6. Team layer: organizing work the squad-model way

**Direction (2026-08-22):** open-seed's orchestration layer should organize work and execute in *teams*, modeled on the Spotify Squad Model (squads / tribes / chapters / guilds, "aligned autonomy"). The research maps onto it cleanly — several surveyed projects independently reinvented pieces of it (bradygaster/squad's `.squad/` charters+routing, tutti's roles+scope globs, kodo's team.json, gastown's crews, corellis/opengoat's org charts, Claude Code's native agent teams):

| Squad-model concept | open-seed realization | Precedent |
|---|---|---|
| **Squad** — small, autonomous, cross-functional team owning a mission | Checked-in team definition `.seed/teams/<squad>.yaml`: `mission`, `members` (refs to agent-role defs, mixing humans and agents — gnap's `type: ai\|human`), `scope` (code-ownership globs), `backlog` (task-card filter), `rituals` (workflow refs: e.g. scheduled triage), `autonomy_tier` (L1/L2/L3 per squad) | squad's `team.md` roster + charters; tutti `[[agent]] scope`; kodo team.json |
| **Tribe** — collection of squads in one area | The repo (or org overlay for multi-repo); tribe-level `guardrails.yaml` cascades down, squads may only tighten, never loosen | qm's org-overlay repos; corellis fleet governance |
| **Chapter** — competency line across squads (all reviewers, all testers) with a lead maintaining standards | The agent-role definitions themselves (`.seed/agents/reviewer.md` etc.) are chapter artifacts: one canonical definition per role shared by every squad; the chapter lead is the human CODEOWNER of that file, so role changes are reviewed by the person who owns the craft | sub-agents-skills role files; opengoat role-differentiated skills; antfarm shared role files |
| **Guild** — voluntary interest community sharing practice | The shared skills library (`skills/` + manifest/lockfile), publishable across repos/orgs | skillfold; Claude Code plugin marketplaces |
| **Mission/OKR alignment** | Goal ancestry on every task card (`parent`/goal links tracing to the squad mission) — agents always see the "why" | Paperclip's mandatory goal ancestry; beads epics |
| **Autonomy within alignment** | Squads own *how* (agents choose implementation; squad picks its workflows); tribe owns *what* (missions, guardrails, quality bar via CI + evidence receipts) | kodo's "Tell them WHAT, never HOW"; Fusion automation levels |

The squad model's documented failure modes also have direct mitigations in the substrate: **cross-squad dependencies** (its classic weakness) become typed `blocks`/`waits-for` edges between task cards plus mailbox messages, visible in `ready` queries rather than discovered in standups; **alignment drift** is countered by goal ancestry + tribe-level guardrails validated in CI; **chapter erosion** is countered by making role definitions reviewable files with owners rather than tribal knowledge.

This also resolves the topology question (D2) more concretely: the paved road is *squad-shaped* — a small named team with a mission and a lead role — rather than a single anonymous loop, with the loop remaining the degenerate one-member squad.

## 7. Decisions made

1. **Coordination backends are plugins (decided 2026-08-21).** Whatever coordination server/backend is used — markdown task files, beads, GitHub Issues, Paperclip, Gas Town, or anything future — it is implemented as a **plugin behind a stable port interface**, and nothing else in the template (scripts, skills, hooks, CI, agent instructions) may talk to a backend directly. This goes beyond a fixed adapter set: the seed exposes a plugin system — packaging (manifest + implementation), checked-in declaration of the active backend, capability negotiation (plugins declare which port operations they support so the seed degrades gracefully — e.g. a markdown backend without true atomic claims), version pinning with a lockfile, and a trust model for third-party plugins (pinned SHAs, review-before-install, plugin output treated as untrusted input to agents). Precedents from the research: sortie/lalph/ralph-tui's pluggable task sources ("the task-source abstraction, not the tracker, is the design decision"), ORCH's shell adapter, ouijit's JSON task CLI, Claude Code plugins/marketplaces, skillfold's manifest+lockfile, thurbox's capability-gated plugins. The concrete spec lives in [`docs/research/10-org-control-planes.md`](./research/10-org-control-planes.md) Part 5. Summary of the v1 proposal: a **JSON-over-CLI port** — every script, skill, hook, and agent uses only `seed task <verb> --json`, which shells out to the active backend plugin at `.seed/backends/<name>/bin/seed-backend`; nine required verbs (create, ready, get/list, claim, release, transition, close-with-blocker-cascade, comment/attach-evidence, event-append) plus optional capabilities (lease-renew, ancestry, deps, event-emit, wake, budget, watch); exit code 2 reserved for claim contention; manifest (`backend.toml`) declares capabilities (`atomic_claim: native|emulated|none`, offline, budget) for graceful degradation; `.seed/backends.lock.json` pins installed plugins by SHA+hash; MCP is the v2 transport via one generic wrapper over the same CLI. Ship order: `filecards` (zero-dep reference impl) → `beads` (atomicity + Gas Town/Gas City adjacency) → `github-issues` (the bridge to multica/Fusion/humans), with a `paperclip` REST adapter as the deferred fourth.

## 8. Open questions to resolve next

1. **Harness posture:** Claude-Code-first with portable shims (recommended — richest primitive set: hooks, subagents, sandbox, workflows), or strictly harness-neutral from day one (more work, lower ceiling)?
2. **Task substrate default (behind the adapter):** markdown-first with beads opt-in (recommended), or beads-first (heavier, stronger under parallelism)?
3. **How much automation ships enabled?** Conventions + scripts only, vs. CI workflows live from clone (dispatcher, scheduled maintenance) — the latter needs secrets setup and a decision on claude-code-action vs gh-aw.
4. **Scope of the loop runner:** ship `loop.sh` (crosses from "conventions" into "runtime"), or document the pattern and point at ralphex/dex?
5. **Language/stack coupling:** is open-seed language-agnostic (Makefile contract only), or does it ship opinionated stacks (e.g. a TS variant with lint/test wired)?
6. **Distribution model:** GitHub template repo (clone-and-drift), a Claude Code plugin/marketplace (updatable), or both — with the plugin carrying skills/agents/hooks and the template carrying structure?
