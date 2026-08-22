# open-seed — Design Options

> Synthesis of a survey of all 180 projects in
> [awesome-agent-orchestrators](https://github.com/andyrewlee/awesome-agent-orchestrators), the
> August 2026 SOTA (harness-native primitives, task tracking, guardrails, methodology), and
> implementation-grade deep dives of the key inspiration projects.
> Category evidence lives in [`docs/research/`](./research/); exact file formats, schemas, and
> algorithms live in [`docs/research/inspirations/`](./research/inspirations/).
> Researched 2026-08-21/22. Precedence: on **facts about third-party projects**, the deep dives
> (source-level evidence) win and this document is kept in sync; on **open-seed's own proposed
> schemas**, this document supersedes the deep dives' synthesis sections where they differ
> (supersessions are called out inline).
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
3. **Portable capability layer = Agent Skills** (`<dir>/SKILL.md` per agentskills.io) — adopted by Claude Code, Codex, Gemini CLI, Cursor, Copilot, OpenCode, and ~30 more. The cross-client checked-in location converging in the wild is **`.agents/skills/`**. Markdown-with-YAML-frontmatter is the universal file format for all agent config.
4. **Tools layer = MCP**, configured in a checked-in `.mcp.json` with env-var expansion for secrets.
5. **Fresh context per task, state on disk.** Conversation memory is a liability; files in the repo are the memory (Ralph doctrine, confirmed by every loop runner and by Anthropic's own context-engineering guidance).
6. **CI is the real guardrail ("backpressure").** Agents merge whatever passes, so a fast, deterministic `make check` + branch protection + required review is the quality system; everything else layers on top.
7. **Agent status vocabulary**: working / blocked(needs-you) / idle / done — the de-facto four-state enum every monitoring tool understands (tmux-ide's `@agent_state` grammar is the concrete wire format).

## 3. Design dimensions and options

### D1. Task-tracking substrate

The central architectural choice — resolved structurally by the plugin decision in §7 (all substrates sit behind one port), leaving only the *default*. Four options:

| Option | Evidence | Pros | Cons |
|---|---|---|---|
| **A. Task-card files in-repo** (one markdown+frontmatter file per task; schema in [inspirations/01](./research/inspirations/01-git-native-task-substrates.md)) | The dominant loop-runner convention (ralphex, Automaker, wreckit, gnap, tick-md) | Zero deps; diffable; PR-reviewable; works offline; travels with forks; works with any harness | Atomic claim is only *emulated* (push-wins); fine at ≤~10 writers per remote, breaks beyond |
| **B. Beads** (steveyegge/beads, 26.5k★) — git-native graph tracker: hash IDs, typed dependency edges, `bd ready`, atomic `--claim`, native leases+heartbeats, AGENTS.md integration | What gastown, orc, and ralph-tui all bet on | Purpose-built for parallel agents; ready-work dispatch; crash-safe leases; memory across sessions | Binary dependency (Go + Dolt); Dolt-backed (embedded single-writer by default, server mode for concurrent writers) with an evolving schema — pin a version |
| **C. GitHub Issues + label state machine** (sortie's query-filter + label swaps, lalph, OpenHands) | The CI-automation category standard | Human visibility for free; API-native to Actions; server-atomic assignment; zero new infra | Rate-limited; slow for agent churn; no offline; state lives outside the clone |
| **D. Harness-native task lists** (Claude Code agent teams' shared task list) | Free intra-session coordination | Zero setup | Ephemeral, machine-local; never a system of record |

**Recommendation: A as the shipped default; B as the documented upgrade; C not as a backend but as an optional one-way *mirror*** (see below). One **canonical state machine** used everywhere (task cards, the port's `transition` verb, and the mirror's rendering): `backlog → ready → in_progress → review → done`, plus `blocked` and `cancelled`, with `review → ready` as the rejection edge. The two-stage done that Automaker/claude-command-center converged on is preserved as `review` (agent-finished) vs `done` (human/verifier-accepted). On rejection, the rejecting reviewer is recorded and the card gains a `rejected_authors` entry; the `claim` verb refuses a claim by anyone in that list (squad's reviewer-lockout, made mechanical — filecards can enforce this in the shim because it reads the card before claiming; backends that can't declare the capability absent).

**The GitHub mirror is a component, not a backend.** Exactly one backend is active (§7). When the mirror is enabled, a one-way exporter (a step in the maintenance/dispatcher workflows) renders card state to issue labels; **cards are authoritative and the export direction always wins** — label edits by humans are read back only as *requests* (the dispatcher turns a human's label change into a `seed task transition` call, which may refuse). Label mapping is explicit: `ready → seed:ready`, `in_progress → seed:in_progress`, `review → seed:review`, `done → seed:done`, `blocked → seed:blocked`; `backlog` = no state label; `cancelled` = issue closed as not-planned. (This supersedes the `seed:working` name in [inspirations/06](./research/inspirations/06-ci-native-automation.md).)

Key conventions regardless of substrate: **task ↔ branch ↔ worktree 1:1:1 mapping**; hash-based IDs, never sequential counters (two branches consuming a `next_id` is a guaranteed merge conflict — tick-md's design flaw); every closed task must reference evidence (commit SHA, check run) via its receipt (D4.5).

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

**Recommendation: thin, mandatory, gated.** The single highest-leverage quality convention found across all categories is **plan-as-gated-artifact**: every non-trivial task produces a committed plan file (steps, file scope, acceptance criteria, `## Validation Commands`) that is reviewed — by human or reviewer agent — *before* execution, and implementation is reviewed *against that same file*. The concrete grammar (ralphex's parser rules + frankbria's Optional-section semantics + the dex skip convention) is specified in [inspirations/02](./research/inspirations/02-ralph-loop-implementations.md). One deliberate departure from precedent: in ralphex/dex, `## Validation Commands` is advisory — the *agent* is told to run it; open-seed's loop runner and pre-merge gate also execute it **mechanically** (martin-loop's fresh-evidence rule), so completion is evidence, not assertion.

Plan lifecycle mechanics (owner assigned, links stable): plans live at `plans/<task-id>.md`; the task card's `plan:` field stores the *task id*, not a path — lookups check `plans/<id>.md` then `plans/completed/<id>.md`, so links never dangle. Archival to `plans/completed/` is a **shim-side effect of `seed task close`** (the backend's `close` verb handles card state; the shim wrapping it moves the plan). The validator lints that every closed card's plan resolves in `completed/`. Full SDD frameworks (spec-kit/BMAD) are contested and heavy; they remain optional adapters, not the core.

### D4. Guardrails stack

Defense in depth; every layer is checked-in-able. The `guardrails.yaml` skeleton in [inspirations/03](./research/inspirations/03-governance-and-gates.md) is the starting point, **superseded here on two points** (its allowlist example and its self-protection path — see erratum note there).

1. **Policy file** (`.seed/guardrails.yaml`): protected-path globs (restriction-only, claudexor-style — matching changes require a human decision), auto-merge allowlist vs denylist (loop-engineering's `gate.yaml` split), `max_files_per_change`, budgets (with the honest caveat in §8-R6), and **named autonomy tiers** (L1 report-only / L2 assisted-in-worktree / L3 unattended-with-gates). **Hard rule: the orchestration control surface is never auto-mergeable and always protected, regardless of any allowlist entry** — `AGENTS.md`, `CLAUDE.md`, `skills/**`, `rules/**`, `.seed/**`, `.claude/**`, `.agents/**`, `scripts/**`, `Makefile`, `.github/**`, `.mcp.json`, `seed.yaml`, `seed.lock`, `.worktreeinclude`, `.gitattributes`, `CODEOWNERS`. A naive `**/*.md` allowlist would otherwise let an L3 agent rewrite every future agent's instructions and auto-merge it — in this design, most of the instruction surface *is* markdown. The validator rejects any guardrails file whose allowlist intersects the control surface. Loop-engineering comes closest to this vocabulary today; nobody yet ships enforced tiers as one machine-readable, hook-enforced file — that combination is the open-seed differentiator.
2. **Harness enforcement**: `.claude/settings.json` permissions (allow/ask/deny) + hooks (PreToolUse denials of destructive commands and protected paths; Stop/TaskCompleted hooks that block completion until lint/tests pass), mirrored minimally for Codex/OpenCode. Best-effort per harness; the layers below are the backstop.
3. **Git/server enforcement — the real gate for L3.** Client-side hooks and CI jobs that run *inside a PR* can be edited by that same PR (GitHub runs the PR's own workflow files for `pull_request` events), so they cannot be the last line of defense. L3 auto-merge is sound **only** with server-side settings the template documents and scripts (`seed init-github` guidance): branch protection with required status checks configured in repo settings, required review via a shipped **CODEOWNERS** file covering the control surface (also the mechanism for §6's chapter leads), and no agent credentials that can bypass them. Client-side: pre-commit hooks (gitleaks, contract checks) and the **blocking `pre-merge.d/` gate** — notably, no surveyed *worktree tool's lifecycle hook* blocks a merge (dmux's `pre_merge` is detached/non-blocking), so a genuinely blocking local gate is novel in that tool class (platform-side merge gates do exist: tutti's land gate, Fusion's pre-merge review phase). Agents never push the default branch; coordination writes go to the state ref (§7.2).
4. **CI enforcement**: one fast `make check`; CI also validates the orchestration files themselves — task-card/plan lint (including closed-card plan archival), workflow validate, skills-lock `--frozen` check, guardrails schema check (including the control-surface/allowlist intersection rule), **receipt verification** (D4.5), run-log append-only check (defined as *set-inclusion against the merge base*, not prefix equality — union merges reorder lines), and fan-out drift check (§8-R1).
5. **Verification separation and receipts.** No agent verifies its own work (antfarm, kodo, loki-mode; squad's reviewer-lockout, enforced per D1). Ship a read-only **reviewer role** distinct from the implementer. **Receipts are written only by the seed CLI/gate — never by agents** (`.seed/receipts/` is on the protected control surface): the gate re-runs the verify commands fresh, records exit codes + diff SHA + protected-path scan into the receipt (facts), attaches the reviewer verdict (assessments), and lists every skipped check (honesty) — headline computed from facts alone; silence never reads as pass; refuse "done" on empty diffs. `seed receipt verify` (loki-style re-derivation: re-hash the diff, optionally re-run commands) runs at the pre-merge gate and in CI, so a hand-edited receipt is caught. Receipts are evidence of a change, so they travel *with the task's PR* on the default branch.
6. **Secrets**: never in agent-readable config; env/`settings.local.json`/`.mcp.json` expansion; gitleaks in pre-commit + CI; secrets *names* documented, values never; "secrets never touch the model" posture documented.

### D5. Memory conventions

Converged pattern worth shipping: a committed **`LEARNINGS.md`** (build/test commands, discoveries — agent-appendable; the Ralph-family "AGENT.md", renamed to avoid confusion with `AGENTS.md`), **`DEADENDS.md`** (failed approaches so fresh sessions don't retry them — dex; derivable from run-log entries with failure status, agent-annotated), append-only **`run-log.jsonl`** (checked per D4.4), and optional `decisions/` (ADR-style; squad's `decisions.md` with `merge=union` is the merge-safe precedent — union is safe here because decisions files are genuinely append-only). Rules about *who may mutate what* (agent may append, never rewrite; humans only edit guardrails; only the gate writes receipts) matter more than the filenames. Personal-assistant-style `MEMORY.md`/daily-notes rollups are overkill for a project template — skip.

### D6. Lifecycle/workspace contract

The full contract (hook names, env vars, exit-code semantics, per-tool shims) is specified in [inspirations/07](./research/inspirations/07-lifecycle-contracts.md). Summary: `.seed/hooks/` with **`setup`, `run`, `teardown`** single-file hooks plus `post-create.d/` and `pre-merge.d/` run-parts dirs; executable-bit-as-opt-in, spawned without a shell, context via `SEED_*` env vars only; setup/teardown advisory (never strand a worktree), **pre-merge gating** (the one blocking set, with the D4.3 caveat that server-side protection is the backstop); a root **`.worktreeinclude`** adopting agent-deck's format verbatim. Ship 2–5-line shims for superset/amux/vibe-tree/dmux/agent-deck/octomux so any of those tools drives the same scripts.

### D7. CI/automation layer

Actions-native proves out fully (gh-aw, claude-code-action, aeon). Ship: a mention/label **dispatcher workflow**, a scheduled maintenance workflow (stale-claim/lease reaping, mirror export, expiry cleanup, blocked-item reporting), and a PR-review workflow — built on `anthropics/claude-code-action@v1` as the pragmatic default (write-access trigger gating, per-command Bash allowlists, branch-not-PR default), with **gh-aw as the documented upgrade** for its read-only-agent + safe-outputs + sanitization + egress-firewall architecture (and for real budget enforcement — §8-R6).

**Workflows obey the port rule (§7.1):** all task-state mutations inside workflows go through `scripts/seed task <verb>`; label changes are performed only by the mirror-export step rendering card state (this corrects the inspirations/06 draft, where the dispatcher swapped labels directly — that draft's label commands should be read as the mirror step, not as state mutation). One-shot command labels `cmd:*` (auto-removed on activation — gh-aw's `label_command` semantics) and the forced `by:agent` provenance label are unchanged. Provenance conventions everywhere: `[ai]` title prefixes, hidden `<!-- seed-workflow-id -->` body markers, sticky progress comments. Scheduled crons use scattered minutes (never `:00`) and assume missed ticks (aeon's debt-model catch-up). The dispatcher's routing instructions live at `.seed/agents/dispatcher.md` (referenced from the workflow prompt).

### D8. Skills/agents packaging & harness portability

- Portable core: `AGENTS.md` + a source-of-truth skills tree + `.mcp.json`; per-harness **fan-out by copy, not symlink** (symlinks break on Windows checkouts, zip downloads, and some CI sandboxes) into `.claude/skills/` and `.agents/skills/`, with drift policed by an offline check (§8-R1). Source-of-truth paths are root **`skills/`** and **`rules/`** (this supersedes inspirations/05's `seed/skills/` path — content dirs stay at root, `.seed/` is reserved for the orchestration contract); rules additionally sync into a marker-fenced managed block in `AGENTS.md` (skillfold's byte-exact round-trip, so the check is offline-verifiable).
- Agent role definitions as markdown-with-frontmatter in `.seed/agents/*.md` (dual-format: Claude Code subagent fields + sub-agents-skills' `run-agent`/`permission: read-only|safe-edit|yolo`/`## Done When`; schema in [inspirations/05](./research/inspirations/05-skills-packaging.md)), fanned out unchanged to `.claude/agents/`.
- Shared skills across cloned repos: **manifest + lockfile** (`seed.yaml` + `seed.lock`, skillfold semantics — SHA+sha256 pins, `install --frozen` in CI, managed-directory-only pruning, timestamp-free alphabetized entries); skill updates are PRs with injection review.
- Workflows-as-files: **YAML with a markdown-prompt escape hatch** (`prompt: |` inline or `prompt_file:` — the Archon middle path; full step schema and validate rules in [inspirations/04](./research/inspirations/04-workflow-as-config.md)), with a `seed workflow validate` preflight in CI and a **mock harness**; `.claude/workflows/` additionally for Claude-native dynamic workflows. (v2 — see §7.3.)

## 4. Repository layout

Reconciled with all decisions (§6 teams, §7 plugins/state-ref/scope-cut) and the deep-dive contracts. Committed config lives under `.seed/`; **mutable coordination state lives on the `seed/state` ref** (§7.2); runtime scratch (worktrees, checkpoints) is excluded via `.git/info/exclude` (orc's trick). Items marked *(v2)* per the §7.3 scope cut.

```
open-seed/
├── AGENTS.md                  # source of truth for agent instructions (+ managed rules block)
├── CLAUDE.md                  # "@AGENTS.md" import + Claude-specific notes
├── README.md
├── Makefile                   # make check — the one fast backpressure command
├── .mcp.json
├── .worktreeinclude           # gitignored-file propagation into new worktrees (agent-deck format)
├── .gitattributes             # merge=union on decisions/ only (genuinely append-only files)
├── CODEOWNERS                 # control-surface review + §6 chapter leads
├── seed.yaml / seed.lock      # skills manifest + lockfile (v2; local skills need no manifest)
├── .claude/
│   ├── settings.json          # permissions, hooks (Stop gate, PreToolUse denials), plugin decls
│   ├── ci-settings.json       # settings profile used by CI workflows
│   ├── agents/                # fan-out copies of .seed/agents/ (do not edit here)
│   ├── skills/                # fan-out copies of skills/ (do not edit here)
│   └── workflows/             # optional Claude dynamic workflows
├── .agents/skills/            # cross-harness fan-out (agentskills.io convention)
├── skills/                    # Agent Skills source of truth, harness-neutral
├── rules/                     # rule fragments synced into AGENTS.md managed block
├── plans/                     # gated plan files, plans/<task-id>.md (grammar per inspirations/02)
│   └── completed/
├── .seed/                     # the orchestration contract (committed, control surface)
│   ├── config.toml            # active backend, defaults, roles→runtime map
│   ├── guardrails.yaml        # autonomy tiers, budgets, protected paths, allowlists
│   ├── version                # protocol version int (enforced by the shim, exit 10)
│   ├── agents/                # role definitions incl. dispatcher.md (fanned to .claude/agents/)
│   ├── teams/                 # squad definitions: mission, members, scope, rituals, tier
│   ├── receipts/              # evidence receipts — written only by the gate, travel with PRs
│   ├── memory/                # LEARNINGS.md, DEADENDS.md, decisions/
│   ├── handoff/               # continuation packets (inspirations/08)
│   ├── backends/              # coordination-backend plugins (filecards ships in-template)
│   ├── backends.lock.json     # SHA+hash pins for installed backend plugins
│   ├── workflows/             # workflow YAML (v2; repo overrides bundled defaults by name)
│   ├── port-schema/           # JSON Schemas for the task-port contract (conformance tests)
│   └── hooks/                 # setup / run / teardown / post-create.d/ / pre-merge.d/
├── scripts/
│   ├── seed                   # the CLI shim: task/receipt/sync/backend/hooks subcommands
│   ├── loop.sh                # Ralph loop: dual-gate exit, circuit breaker, budgets
│   └── validate.sh            # lint all orchestration files (also run in CI)
├── .github/
│   ├── workflows/             # check, validate, seed-dispatch, seed-maintenance, pr-review
│   └── ISSUE_TEMPLATE/        # machine-parseable forms + label taxonomy
└── docs/                      # this study + conventions handbook

# On the seed/state ref (not the default branch):
#   tasks/                     # task cards (filecards backend)
#   run-log.jsonl              # append-only event log
#   mail/<agent>/<msg-id>.yaml # one file per MESSAGE (v2; never rewritten, no union needed)
```

## 5. What open-seed would be that nothing else is

Closest prior art, per the research: **tutti** (committed org-code TOML — with real caveats: index-based step deps, role→runtime-only mapping; see [inspirations/04](./research/inspirations/04-workflow-as-config.md)) + **beads** (committed task graph) + **loki-mode's gates** (deterministic verification with evidence receipts) + **loop-engineering** (checked-in loop conventions with autonomy tiers) + **bradygaster/squad** (repo-resident team charters). *No project combines these in template form.* The unclaimed gaps open-seed can own:

1. A **checked-in autonomy-tier + guardrails vocabulary** enforced by hooks, server-side protection, and CI (apps keep this in settings; loop-engineering documents it; nobody versions-and-enforces it as one file).
2. The **task↔plan↔evidence chain**: task card → gated plan → implementation → gate-written, re-verifiable receipt, all as diffable files.
3. A **blocking local pre-merge gate** in the worktree lifecycle contract (no surveyed worktree tool's lifecycle hooks block; platform merge gates like tutti's exist but require their runtime).
4. **Runner-agnostic degradation**: the same repo works with a lone human, one Claude Code session, Claude agent teams, a Ralph loop in CI, or any of the 60+ external orchestrators surveyed — because the contract is files, with documented shims to each tool's convention.

## 6. Team layer: organizing work the squad-model way

**Direction (2026-08-22):** open-seed's orchestration layer organizes work and executes in *teams*, modeled on the Spotify Squad Model (squads / tribes / chapters / guilds, "aligned autonomy"). The research maps onto it cleanly — several surveyed projects independently reinvented pieces of it (bradygaster/squad's `.squad/` charters+routing, tutti's roles+scope globs, kodo's team.json, gastown's crews, corellis/opengoat's org charts, Claude Code's native agent teams):

| Squad-model concept | open-seed realization | Precedent |
|---|---|---|
| **Squad** — small, autonomous, cross-functional team owning a mission | Checked-in team definition `.seed/teams/<squad>.yaml`: `mission`, `members` (refs to agent-role defs, mixing humans and agents — gnap's `type: ai\|human`), `scope` (code-ownership globs), `backlog` (task-card filter), `rituals` (workflow refs), `tier` (≤ the repo ceiling) | squad's `team.md` roster + charters; tutti `[[agent]] scope`; kodo team.json |
| **Tribe** — collection of squads in one area | The repo (or org overlay for multi-repo); repo-level `guardrails.yaml` is the floor | qm's org-overlay repos; corellis fleet governance |
| **Chapter** — competency line across squads with a lead maintaining standards | The agent-role definitions (`.seed/agents/reviewer.md` etc.): one canonical definition per role shared by every squad; the chapter lead is the human CODEOWNER of that file | sub-agents-skills role files; opengoat; antfarm shared role files |
| **Guild** — voluntary interest community sharing practice | The shared skills library (`skills/` + manifest/lockfile), publishable across repos/orgs | skillfold; Claude Code plugin marketplaces |
| **Mission/OKR alignment** | Goal ancestry on task cards (`parent`/goal links tracing to the squad mission) | Paperclip's mandatory goal ancestry; beads epics |
| **Autonomy within alignment** | Squads own *how*; tribe owns *what* (missions, guardrails, quality bar via CI + receipts) | kodo's "Tell them WHAT, never HOW"; Fusion automation levels |

**Routing semantics (normative, so the layer is buildable):**

- **Tier precedence:** `guardrails.yaml` sets `autonomy.default_tier` and `autonomy.max_tier` (the ceiling). A squad file's `tier` must satisfy `tier ≤ max_tier` — a simple comparison the validator checks (avoiding the undecidable "squads may only tighten arbitrary globs/budgets" formulation). Squads cannot override protected paths, allowlists, or budgets — those are repo-level only. Per-squad *harness* enforcement happens at spawn time (the spawn script generates per-worktree settings from the squad's tier); the gate layer (which reads the squad tier directly) is the backstop.
- **Scope:** squads' `scope` globs may not overlap — overlapping scopes are a validation error, resolved by splitting or by an explicit shared-scope entry naming one owning squad for review purposes. Files matching no squad's scope belong to the default **`core`** squad.
- **Backlog:** a card is routed to exactly one squad: its explicit `squad:` field if set, else the first squad (in a validator-checked priority order) whose `backlog` filter matches, else `core`. No card can be invisible. `seed task ready --squad <name>` is an optional port capability (backends without it get shim-side filtering of `ready` output).
- **Goal-ancestry validation** activates only when the repo defines more than one squad or any mission — a solo clone pays nothing (cards' `parent` stays optional).

The squad model's documented failure modes (per the [ideaplan case study](https://www.ideaplan.io/case-studies/spotify-squad-model): "the model as described in the whitepaper never fully existed in practice") map to mitigations that are *mechanical* in an agent org where they were only aspirational in a human one:

- **Fragmentation**: tribe-level guardrails, lint/test config, and chapter role definitions are enforced by CI and hooks — agents cannot drift on standards, because the standards are executable.
- **Chapter-lead dysfunction**: the chapter lead is a CODEOWNERS entry on a role-definition file — zero people-management burden.
- **Tribal silos**: cross-squad dependencies are typed `blocks`/`waits-for` edges visible in `ready` queries; Spotify's "internal open source" maps directly — any squad's agents may PR into another squad's scope, subject to that squad's review gate.
- **Guild decline**: the guild is a versioned skills library — it doesn't depend on volunteer energy.
- **Alignment**: squad missions connect to repo objectives via goal ancestry, validated per the activation rule above.

The article's real lesson — adopt the principles, not the org chart — is the template's posture: team files are conventions a project tunes, not a mandated hierarchy. In the minimal clone exactly one squad exists (`core`, containing the human and a default agent trio); the squad layer costs nothing until a second squad is added.

## 7. Decisions made

**7.1 Coordination backends are plugins (decided 2026-08-21).** Whatever coordination backend is used — task-card files, beads, GitHub Issues, Paperclip, Gas Town, or anything future — it is a **plugin behind a stable port interface**; nothing else in the template (scripts, skills, hooks, CI, agent instructions) may talk to a backend directly (workflows included — see D7). The seed exposes a plugin system: packaging (manifest + implementation), checked-in declaration of the active backend, capability negotiation, version pinning with a lockfile, and a trust model for third-party plugins (pinned SHAs, review-before-install, plugin output treated as untrusted input to agents). The concrete spec lives in [`docs/research/10-org-control-planes.md`](./research/10-org-control-planes.md) Part 5: a **JSON-over-CLI port** (`seed task <verb> --json` → `.seed/backends/<name>/bin/seed-backend`); nine required verbs (create, ready, get/list, claim, release, transition, close-with-blocker-cascade, comment/attach-evidence, event-append) plus optional capabilities (lease-renew, ancestry, deps, event-emit, wake, budget, watch, squad-filtered ready per §6); exit 2 reserved for claim contention, exit 10 for schema/version mismatch (the shim is the enforcement point for `.seed/version` — out-of-tree tools SHOULD check it but only the shim refuses); `backend.toml` capability manifest; `.seed/backends.lock.json` pins; MCP as the v2 transport over the same CLI.

**Claim protocol (binding on all backends):** `claim` is synchronous and completes *before any work begins* — for filecards that means the claim commit is pushed inside the verb, and a push rejection is re-fetched and re-checked: if the task is now claimed by another, the verb exits 2 (the contention window collapses into the verb; a "loser" never has a half-built worktree). Leases: filecards emulates leases as card fields; expiry is reaped by the scheduled maintenance workflow, so **reap latency = maintenance cadence** (documented; teams needing tighter leases use the beads backend). Crash recovery: a reaped claim releases the card and posts a `handoff` note referencing any pushed branch so the next claimant can salvage or delete it.

**7.2 Coordination state lives on a dedicated ref (decided in review, 2026-08-22).** Mutable coordination state — task cards, `run-log.jsonl`, mail — lives on a dedicated, *unprotected* branch **`seed/state`**, written only by the port shim (fetch → commit → push, retry on rejection; reads fetch first, so staleness is bounded and explicit). This resolves the otherwise-fatal conflict between "agents never push the default branch" (D4.3) and "claiming = pushing a commit": the default branch carries conventions, code, plans (which are *reviewed* artifacts and arrive via PRs), and receipts (which travel with the task's PR); the state ref carries the high-frequency machine-written files and needs no checks per write. Precedents: beads syncing through `refs/dolt/data` on the ordinary remote; squad's git-notes state protocol; gh-pages-style out-of-band branches. `ready`/`get` are answered from the fetched state ref, eliminating the claimed-on-branch/ready-on-main split-brain. The shim owns all plumbing; agents and humans never check out `seed/state` directly.

**7.3 v1/v2 scope cut (decided in review, 2026-08-22).** The research warns explicitly against scope creep ("the trap is scope-creep toward daemons and dashboards"), and every shipped convention costs a shipped validator (R9). **v1**: the port + `filecards` backend + state ref; task cards, plan grammar, receipts + `seed receipt verify`; `.seed/guardrails.yaml` + CODEOWNERS + validators; `.seed/hooks/` lifecycle contract + `.worktreeinclude`; role definitions + fan-out sync; `loop.sh`; memory files; two CI workflows (check+validate, seed-maintenance) with the dispatcher shipped inert until secrets are configured; one `core` squad. **v2**: `beads` and `github-issues` backends + the mirror exporter; the workflow engine + mock harness; skills manifest/lockfile + compose; mailboxes + handoff automation; multi-squad routing; per-tool lifecycle shims beyond the top three; `paperclip` adapter; MCP transport.

**7.4 Teams are the organizing unit (direction, 2026-08-22).** Work is organized and executed in squads per §6, with team definitions as checked-in files. The one-member `core` squad is the degenerate default so the layer costs nothing for solo use.

## 8. Risks, gotchas, and mitigations

Known sharp edges the design must handle (each mitigation is part of the template, not an aspiration):

- **R1 — Fan-out drift.** Skills/agents exist in a source tree *and* per-harness copies; they will diverge silently. Mitigation: fan-out is generated only by `scripts/seed sync`; `seed sync --check` (offline, hash-based) runs in CI and fails on drift; fanned directories carry a "do not edit here" marker file.
- **R2 — Windows portability.** The hooks contract relies on executable bits (lost on some Windows checkouts) and the layout forbids symlinks. Mitigation: fan-out by copy everywhere; `scripts/seed hooks run <name>` as an interpreter-invoking fallback runner; document `git config core.fileMode` caveats; no path components differing only by case.
- **R3 — Untrusted content is everywhere.** Task-card bodies, issue text, PR comments, mail, and backend-plugin output are all injected into agent context — every one is a prompt-injection vector, and per D4.1 much of the *instruction surface itself* is markdown. Mitigation: the port shim schema-validates and sanitizes backend output; AGENTS.md instructs agents to treat task/mail content as data; CI workflows follow claude-code-action's write-access trigger gating (gh-aw's sanitization + safe-outputs as the upgrade); backend plugins are SHA-pinned with review-before-update; and the control surface is excluded from auto-merge with server-side review required (D4.1/D4.3).
- **R4 — Emulated claims have a ceiling.** Push-wins claiming on filecards is honest only up to ~10 concurrent writers per remote. The claim-inside-the-verb protocol (§7.1) means contention costs a retry, never lost work — but throughput still degrades. Mitigation: the capability matrix says `atomic_claim: emulated`; the maintenance workflow reports contention rates; the documented upgrade is the beads backend. Never pretend the file backend is atomic.
- **R5 — Committed coordination state churns git history.** Mitigation: high-frequency machine writes go to the `seed/state` ref (§7.2), keeping the default branch's history human-meaningful; one file per task (and per mail message — never rewritten files under `merge=union`; union applies only to genuinely append-only `decisions/`); hash IDs so parallel creates never collide; the append-only check is set-inclusion vs. the merge base; runtime scratch in `.git/info/exclude`.
- **R6 — Budgets are advisory without a server.** A repo cannot hard-stop spend across machines; only run-local caps (max-iterations, wall-clock, per-run token counts from harness output) are enforceable. Mitigation: state this honestly in guardrails.yaml docs; enforce what is enforceable locally (loop budgets, attempt caps, circuit breaker); post-hoc accounting in the run log; teams needing hard org-wide stops use gh-aw (per-run/daily credit caps) or a control plane (Paperclip) via the backend/event seams.
- **R7 — Harness churn.** Agent teams are experimental, hook vocabularies change, vendors absorb orchestration features fast. Mitigation: the harness-neutral core (AGENTS.md, skills, port CLI, hooks contract) carries the semantics; harness-specific files are generated shims; CI never depends on experimental harness features.
- **R8 — Template drift after clone.** GitHub template repos cannot push updates to clones. Mitigation: keep the update surface small and versioned (`.seed/version`, lockfiles); provide `seed upgrade` guidance (diff against tagged template releases); distribute the *evolving* parts (skills, role defs, workflows) additionally as a Claude Code plugin/marketplace (ties to §9 Q4).
- **R9 — Convention without enforcement rots.** Every convention lacking a validator becomes stale documentation (loop-engineering's "State Rot"). Mitigation: `scripts/validate.sh` + the validate CI workflow check *every* checked-in orchestration artifact (cards, plans incl. archival, receipts, workflows, guardrails incl. the control-surface rule, team files incl. tier/scope/backlog rules, lockfiles, run-log inclusion check, fan-out drift); "shipped convention" and "shipped validator" are one deliverable — which is precisely why the v1 scope cut (§7.3) is small.

## 9. Open questions to resolve next

1. **Harness posture:** Claude-Code-first with portable shims (recommended — richest primitive set: hooks, subagents, sandbox, workflows), or strictly harness-neutral from day one (more work, lower ceiling)?
2. **How much automation ships enabled?** §7.3 proposes: check+validate and maintenance workflows live, dispatcher in-tree but inert until secrets are configured, `claude-code-action` default with gh-aw documented. Needs your confirmation.
3. **Language/stack coupling:** is open-seed language-agnostic (Makefile contract only), or does it ship opinionated stacks (e.g. a TS variant with lint/test wired)?
4. **Distribution model:** GitHub template repo (clone-and-drift), a Claude Code plugin/marketplace (updatable), or both — with the plugin carrying skills/agents/hooks and the template carrying structure? (Interacts with R8.)
5. **State-ref hosting constraints:** `seed/state` assumes the remote allows unprotected branch pushes by agent credentials — confirm this fits your GitHub permissions model (fine-grained PATs / deploy keys scoped to that ref), or whether the v1 default should further restrict which principals may push it.
