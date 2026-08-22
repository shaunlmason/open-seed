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

1. **Isolation = git worktree per task/agent**, one branch per unit of work, branch `seed/<task-id>` (see §7.2 for why the coordination ref lives *outside* this namespace). Containers/devcontainers are an optional second ring for untrusted/full-auto runs, never the default. (~40 of the surveyed projects use exactly this.)
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
| **A. Task-card files in-repo** (one markdown+frontmatter file per task; schema in [inspirations/01](./research/inspirations/01-git-native-task-substrates.md)) | The dominant loop-runner convention (ralphex, Automaker, wreckit, gnap, tick-md) | Zero deps; diffable; auditable via ref history; works offline; works with any harness | Atomic claim only *emulated* (push-wins, see §7.1); all writes serialize through one ref (§8-R4); cards live on the state ref, so they are **not** PR-reviewed — the plan gate is the review point (see below); forks copy the default branch only, so coordination state does not travel with forks automatically |
| **B. Beads** (steveyegge/beads, 26.5k★) — git-native graph tracker: hash IDs, typed dependency edges, `bd ready`, atomic `--claim`, native leases+heartbeats, AGENTS.md integration | What gastown, orc, and ralph-tui all bet on | Purpose-built for parallel agents; ready-work dispatch; crash-safe leases; memory across sessions | Binary dependency (Go + Dolt); Dolt-backed (embedded single-writer by default, server mode for concurrent writers) with an evolving schema — pin a version |
| **C. GitHub Issues + label state machine** (sortie's query-filter + label swaps, lalph, OpenHands) | The CI-automation category standard | Human visibility for free; API-native to Actions; server-atomic assignment; zero new infra | Rate-limited; slow for agent churn; no offline; state lives outside the clone |
| **D. Harness-native task lists** (Claude Code agent teams' shared task list) | Free intra-session coordination | Zero setup | Ephemeral, machine-local; never a system of record |

**Recommendation: A as the shipped default; B as the documented upgrade; C not as a backend but as an optional one-way *mirror* (v2).**

**Canonical state machine** — states `backlog, ready, in_progress, review, done, blocked, cancelled`; full edge table (backends and validators implement exactly this; anything else is an invalid-transition error):

| From | Allowed to |
|---|---|
| `backlog` | `ready`, `cancelled` |
| `ready` | `in_progress` (claim), `backlog` (deprioritize), `blocked`, `cancelled` |
| `in_progress` | `review`, `ready` (release), `blocked`, `cancelled` |
| `blocked` | `ready`, `cancelled` |
| `review` | `done` (accept), `ready` (reject), `cancelled` |
| `done` | — terminal; reopening = a new card linked `relates-to` |
| `cancelled` | `backlog` (reinstate) |

The two-stage done (Automaker/claude-command-center) is preserved as `review` (agent-finished) vs `done` (human/verifier-accepted). **Rejection bookkeeping:** on `review → ready`, the *rejected implementer* is appended to the card's `rejected_authors` list and the rejecting reviewer is recorded in the card's review block; the `claim` verb refuses claimants in `rejected_authors` (squad's reviewer-lockout, made mechanical — backends that cannot enforce it declare the capability absent). On any re-claim (after rejection, release, or lease reaping) the new claimant **resets the task branch** (delete + recreate from base) unless the handoff note (§7.1) marks the prior work salvageable.

**Cards are instructions, so their review point must be explicit.** Cards are machine-written to the state ref without PR review; a card body is untrusted input *and* a work order. The compensating control is normative: **above L1, implementation requires an approved plan** — and plans land on the default branch via reviewed PRs (D3), so the plan gate is where a human (or, at L2, a reviewer agent per guardrails) vets what the work order actually authorizes. Claiming an *unplanned* card is allowed at any tier but scopes the claimant to **planning only** (read-only exploration + authoring the plan PR); implementation may begin only once the plan is approved (merge-base rule, D3). This also serializes plan authorship — the claim is the mutex, so two agents can't burn spend writing rival plan PRs for the same card.

**The GitHub mirror is a component, not a backend (v2).** When enabled, a one-way exporter renders card state to issue labels; **cards are authoritative and the export direction always wins** — human label edits are read back only as *requests* (the dispatcher turns them into `seed task transition` calls, which may refuse). Label mapping: `ready → seed:ready`, `in_progress → seed:in_progress`, `review → seed:review`, `done → seed:done`, `blocked → seed:blocked`; `backlog` = no state label; `cancelled` = issue closed as not-planned. (Supersedes `seed:working` in [inspirations/06](./research/inspirations/06-ci-native-automation.md).)

Key conventions regardless of substrate: **task ↔ branch ↔ worktree 1:1:1 mapping**; hash-based IDs, never sequential counters (tick-md's `next_id` is a guaranteed merge conflict); every closed task must reference evidence via its receipt (D4.5).

### D2. Orchestration topology

| Option | Evidence | When it wins |
|---|---|---|
| **A. Single-agent Ralph loop** — checked-in loop runner + plan file + fresh session per task | ralphex, ralph-claude-code, dex; 100% repo-native | Solo dev or one workstream; cheapest, most auditable |
| **B. Parallel flat worktrees** — N independent agents, human merges | claude-squad archetype, superset, emdash, all TUIs | Independent tasks; needs only worktree conventions + merge gate |
| **C. Coordinator–worker** — planner decomposes, workers execute, coordinator never codes | orc, kodo, gastown Mayor, Claude Code agent teams | Larger efforts; requires the task graph (D1) to be real |
| **D. Ticket-claim blackboard** — no assignment; agents wake and atomically claim ready work | paperclip heartbeats, beads `--claim`, gnap | Most robust to agent death; degrades gracefully to 1 agent; needs atomic claim |

**Recommendation: the paved road is *squad-shaped* (§6) — a small named team with a mission — with A as its degenerate one-member case.** The worktree contract + task substrate + claim convention *are* the topology; the template doesn't hardcode a hierarchy beyond the team files. Escalation path: one agent → parallel worktrees → coordinator role prompt (a subagent/skill, not a daemon) → external orchestrator (gastown, Paperclip, etc.) as optional endgame. Claude Code's native agent teams + `.claude/workflows/` cover C natively for Claude shops — open-seed rides those primitives rather than reimplementing them, while keeping the durable task ledger in the repo (agent-team state is machine-local and ephemeral).

### D3. Plan/spec discipline

Options range from none → Ralph-thin (PROMPT.md + fix_plan.md) → plan-as-gated-artifact (Fusion, Ivy-Tendril, dex) → full SDD (spec-kit, BMAD).

**Recommendation: thin, mandatory, gated, pinned.** The single highest-leverage quality convention found across all categories is **plan-as-gated-artifact**: every non-trivial task produces a committed plan file (steps, file scope, acceptance criteria, `## Validation Commands`) that is reviewed *before* execution, and implementation is reviewed *against that same file*. Grammar (ralphex parser rules + frankbria Optional-section semantics + dex skip convention) in [inspirations/02](./research/inspirations/02-ralph-loop-implementations.md). Departure from precedent: in ralphex/dex, `## Validation Commands` is advisory; open-seed's loop runner and pre-merge gate also execute it **mechanically** (martin-loop's fresh-evidence rule).

Plan lifecycle mechanics:

- **One stable location.** Plans live at `plans/<task-id>.md`, forever — no `completed/` move (archival was cosmetic and had no legal writer; the card's state already says a task is closed). The plan path is derived from the card id (no `plan:` field needed); the validator lints that every card in `review`/`done` has a resolvable plan.
- **Approval = the merge-base, not a card field.** The trust root is the default branch, never the state ref (which is push-access-deep, §8-R10). Normatively: *the approved plan for a task PR is the blob of `plans/<task-id>.md` at the PR's merge-base with the default branch.* The gate and CI parse `## Validation Commands` **from the merge-base blob, never from the PR head's copy** (an implementer-controlled checkout must not supply its own acceptance criteria — those commands run in CI with the workflow's credentials). The receipt records the merge-base plan's sha256 (and optionally the plan PR's merge commit); any `plan_hash` on the card is a cache for agents' convenience, never an authority. This is loki's spec-lock adapted to per-task plans with the pin *derived* rather than stored.
- **PR purity rule (the discriminator that makes the gates checkable).** Two disjoint PR classes, distinguishable server-side by head branch: **plan PRs** (branch `seed/<task-id>-plan`) touch *only* `plans/<task-id>.md` — one plan file, nothing else; **task PRs** (branch `seed/<task-id>`) may not touch `plans/**` *at all* (not just their own plan — a task PR burying an edit to another task's plan inside a large diff would launder plan tampering through an unrelated review). CI fails any PR violating its class. Implementation starts from a default-branch head that already contains the merged plan (the task branch is created or reset after plan approval), so the plan-PR and task-PR lifecycles never share a branch.
- **Plan-PR review is enforced, not assumed.** Who reviews a plan PR, by tier: v1 (and any tier before automation secrets exist) — a human; after pr-review activation, L2/L3 — the reviewer workflow's server-attributed app identity, which must differ from the plan author (same identity rule as D4.5). `plans/**` is deliberately not CODEOWNERS control surface, but it is **never auto-mergeable either**: the guardrails validator rejects any auto-merge allowlist entry intersecting `plans/**` exactly as it does for the control surface (otherwise an L3 agent could approve its own work order — author a plan whose validation commands are attacker shell, auto-merge it unreviewed, and have CI execute it).
- **No stale-plan replay.** A superseded plan must be revocable: `seed init-github` enables strict required checks ("require branches to be up to date"), and independently `seed receipt verify` fails when the merge-base plan blob differs from the plan blob at the current default-branch head ("plan changed since branch base — rebase and re-verify"). Without this, an implementer could keep executing a revoked plan v1 forever by never updating their branch.

Full SDD frameworks (spec-kit/BMAD) are contested and heavy; they remain optional adapters, not the core.

### D4. Guardrails stack

Defense in depth; every layer is checked-in-able. The `guardrails.yaml` skeleton in [inspirations/03](./research/inspirations/03-governance-and-gates.md) is the starting point, superseded here per its erratum.

1. **Policy file** (`.seed/guardrails.yaml`): protected-path globs (restriction-only — matching changes require a human decision), auto-merge allowlist vs denylist, `max_files_per_change`, budgets (caveat §8-R6), and **named autonomy tiers** (L1 report-only / L2 assisted-in-worktree / L3 unattended-with-gates). **Hard rule: the orchestration *control surface* is never auto-mergeable and always protected, regardless of any allowlist entry.** The control surface is exactly: `AGENTS.md`, `CLAUDE.md`, `skills/**`, `rules/**`, `.seed/**`, `.claude/**`, `.agents/**`, `scripts/**`, `Makefile`, `.github/**`, `.mcp.json`, `seed.yaml`, `seed.lock`, `.worktreeinclude`, `.gitattributes`, `CODEOWNERS`. Deliberately **not** control surface (they are work products with their own gates): `plans/` (integrity via pinning, D3), `receipts/` (integrity via the CI verify check, D4.5), `memory/`, `decisions/`, `docs/`, and source code (governed by the ordinary allowlist/denylist + review tiers). After the §7.2/§7.3 split, everything under `.seed/` is genuinely human-owned contract, so the blanket glob is safe. The validator rejects any guardrails file whose auto-merge allowlist intersects the control surface **or `plans/**`** (plans are the trust root for what CI executes — D3). A naive `**/*.md` allowlist would hand agents their own instruction surface — most of which is markdown — so the validator enforces this structurally, not by convention.
2. **Harness enforcement**: `.claude/settings.json` permissions + hooks (PreToolUse denials of destructive commands and protected paths; Stop/TaskCompleted completion gates), mirrored minimally for Codex/OpenCode. Best-effort per harness; the layers below are the backstop.
3. **Git/server enforcement — the real gate for L3.** Client-side hooks and CI jobs that run *inside a PR* can be edited by that same PR (GitHub runs the PR's own workflow files for `pull_request` events), so they cannot be the last line of defense. L3 auto-merge is sound **only** with server-side settings the template documents and scripts (`seed init-github`): branch protection with required status checks configured in repo settings, required review via the shipped **CODEOWNERS** covering the control surface, and no *agent* credentials that bypass them. **Solo mode (explicit, because GitHub forbids approving your own PR):** when the repo has one human, `seed init-github` configures enforce-admins *off* so the human owner can merge their own control-surface PRs; required CODEOWNERS review activates when a second human joins. Without this, a solo user could never customize `.seed/guardrails.yaml` after cloning — deadlocking the template's primary degradation case (§1). **Prerequisite it prints when flipping enforce-admins off:** "admin bypass is for humans only" has no mechanism if agents run under the owner's own credential (the default solo setup) — so solo operation above L1/L2 **requires agents to run under a separate non-admin machine identity** (fine-grained PAT or app with no bypass); until one exists, the effective tier is capped and `seed init-github` says so. Client-side: pre-commit hooks (gitleaks, contract checks) and the **blocking `pre-merge.d/` gate** — no surveyed *worktree tool's lifecycle hook* blocks a merge (dmux's `pre_merge` is detached), so a genuinely blocking local gate is novel in that tool class (platform-side merge gates exist: tutti's land gate, Fusion's pre-merge phase). Agents never push the default branch (all default-branch changes arrive via PR); coordination writes go to the state ref (§7.2).
4. **CI enforcement**: one fast `make check`; CI also validates the orchestration files — task-card/plan lint (fetching the state ref explicitly — see §7.2 CI visibility), plan-hash pinning, **receipt verification** (D4.5), guardrails schema check (including the control-surface/allowlist intersection rule), fan-out drift check (§8-R1), team-file rules (§6), and — v2, once the manifest exists — skills-lock `--frozen`. The run-log integrity check runs in the maintenance job against the state ref's own history (commit-over-commit inclusion; there is no merge base to compare against — the shim linearizes writes), with the honest caveat that it is only as strong as the ref's integrity (§8-R10).
5. **Verification separation and receipts.** No agent verifies its own work (antfarm, kodo, loki-mode; reviewer-lockout per D1). Ship a read-only **reviewer role** distinct from the implementer. Receipts live at root **`receipts/<task-id>.json`** (work product, not control surface) and travel with the task's PR. Authorship is tiered honestly:
   - The local gate (`pre-merge.d/` + `seed` CLI) writes a receipt as a *pre-check* — but it runs under the implementer's credentials, so **locally-written receipts are advisory** (an agent with Bash could fabricate one; loki says this out loud: "unsigned receipts are defense-in-depth, not non-forgeability").
   - **For anything above L1, CI is the author of record**: the required `seed receipt verify` status check regenerates the facts — the diff is the **merge-base diff excluding `receipts/**`** (excluding receipts makes the hash stable when the receipt itself is committed; merge-base makes it stable as the default branch advances), verify commands are re-run in the CI environment **from the merge-base plan blob (D3), never the PR head's copy**, protected paths re-scanned, plan blob hash re-derived (including the stale-plan check, D3) — and fails on any mismatch with the committed receipt, so a forged local receipt buys nothing.
   - **The check cannot be evaded by omission.** `seed receipt verify` runs on every PR and classifies by head branch (the D3 purity rule): a `seed/<task-id>` PR **fails** — not skips — if it lacks its receipt, a resolvable approved plan, or a qualifying review; a `seed/<task-id>-plan` PR is checked against the plan-purity and plan-review rules; any other PR (docs, human feature work) passes the receipt check vacuously and is governed by the repo's ordinary review baseline — which the template sets as branch-protection required review for everything, CODEOWNERS adding the *mandatory* reviewers only on the control surface.
   - **Reviewer identity is bound to server-attributed facts, not commit metadata** (git author strings are arbitrary client-set text): the accepting verdict must be a GitHub PR review (server-authenticated) or a commit with a verified signature from the review identity; the implementer's identity is the PR author login; the verify step checks the two differ. In v1 the reviewer is a human; a reviewer-agent workflow joins when dispatcher secrets are configured (§7.3), using its own app identity.
   - Facts/assessments/honesty split per loki: headline computed from facts alone; every skipped check listed as a gap; refuse "done" on empty diffs.
6. **Secrets**: never in agent-readable config; env/`settings.local.json`/`.mcp.json` expansion; gitleaks in pre-commit + CI; secrets *names* documented, values never.

### D5. Memory conventions

Converged pattern worth shipping: **`memory/LEARNINGS.md`** (build/test commands, discoveries — the Ralph-family "AGENT.md", renamed to avoid confusion with `AGENTS.md`), **`memory/DEADENDS.md`** (failed approaches — dex), and **`decisions/`** (ADR-style; `merge=union` in `.gitattributes` applies to `decisions/**` — safe because ADR files are genuinely append-only). All three live at the **repo root on the default branch, outside the control surface**, and their legal write path is the task PR itself: an agent appends learnings/dead-ends in its task branch and they land with the work, reviewed at the same tier as the code (this resolves "agent-appendable" with "agents never push the default branch"). The high-frequency machine log (`run-log.jsonl`) is not a memory file — it lives on the state ref (§7.2). Mutation rules matter more than filenames: agents append, never rewrite; humans only edit guardrails; only the gate/CI writes receipts. Personal-assistant-style `MEMORY.md`/daily-notes rollups are overkill — skip.

### D6. Lifecycle/workspace contract

Full contract (hook names, env vars, exit-code semantics, per-tool shims) in [inspirations/07](./research/inspirations/07-lifecycle-contracts.md). Summary: `.seed/hooks/` with **`setup`, `run`, `teardown`** single-file hooks plus `post-create.d/` and `pre-merge.d/` run-parts dirs; executable-bit-as-opt-in, spawned without a shell, context via `SEED_*` env vars only; setup/teardown advisory (never strand a worktree), **pre-merge gating** (the one blocking set, with the D4.3 caveat that server-side protection is the backstop); a root **`.worktreeinclude`** adopting agent-deck's format verbatim. Ship shims for the top three external tools in v1 (superset, agent-deck, vibe-tree), the rest v2.

### D7. CI/automation layer

Actions-native proves out fully (gh-aw, claude-code-action, aeon). v1 ships two live workflows and one inert one (§7.3):

- **check+validate** (live): `make check` + `scripts/validate.sh` (all D4.4 checks), fetching the state ref for card lint.
- **seed-maintenance** (live): **deterministic scripting only — requires no model secrets**: lease reaping (§7.1), orphaned-claim recovery (contrabass pattern: `in_progress` card with no live branch → release), **closing merged tasks** (merged `seed/<id>` PR + green verify ⇒ `seed task close <id>`, §7.1), the **done-consistency lint** (every `done` card must correspond to a merged `seed/<id>` PR and a `receipts/<id>.json` on the default branch — a mismatch means a forged accept and writes the HALT marker, §7.2), state-ref checkpoint anchoring (§7.2), contention-rate reporting, expiry cleanup of `by:agent` artifacts. Scheduled with scattered minutes (never `:00`), assuming missed ticks (aeon's debt-model catch-up). Mirror export joins this workflow in v2.
- **seed-dispatch** and **pr-review** (both in-tree, inert until secrets configured): built on `anthropics/claude-code-action@v1` (write-access trigger gating, per-command Bash allowlists, branch-not-PR default), with **gh-aw as the documented upgrade** (read-only-agent + safe-outputs + sanitization + egress firewall + real budget enforcement — §8-R6). Routing instructions live at `.seed/agents/dispatcher.md`. Until activation, **v1 review is human-only** (D4.5's identity check accepts a human PR review); pr-review's activation adds the reviewer-agent lane under its own app identity.

**Workflows obey the port rule (§7.1):** all task-state mutations inside workflows go through `scripts/seed task <verb>`; label changes are performed only by the v2 mirror-export step rendering card state (the inspirations/06 draft's direct label swaps should be read as that step — see its erratum). One-shot command labels `cmd:*` (auto-removed on activation) and the forced `by:agent` provenance label ship with the dispatcher; the full state-label taxonomy activates with the v2 mirror. Provenance conventions everywhere: `[ai]` title prefixes, hidden `<!-- seed-workflow-id -->` body markers, sticky progress comments.

### D8. Skills/agents packaging & harness portability

- Portable core: `AGENTS.md` + a source-of-truth skills tree + `.mcp.json`; per-harness **fan-out by copy, not symlink** (symlinks break on Windows checkouts, zip downloads, some CI sandboxes) into `.claude/skills/` and `.agents/skills/`, drift-policed offline (§8-R1). Source-of-truth paths are root **`skills/`** and **`rules/`** (supersedes inspirations/05's `seed/skills/` path — content dirs stay at root; `.seed/` is reserved for the orchestration contract); rules additionally sync into a marker-fenced managed block in `AGENTS.md` (skillfold's byte-exact round-trip, offline-verifiable).
- Agent role definitions as markdown-with-frontmatter in `.seed/agents/*.md` (dual-format: Claude Code subagent fields + sub-agents-skills' `run-agent`/`permission`/`## Done When`; schema in [inspirations/05](./research/inspirations/05-skills-packaging.md)), fanned out unchanged to `.claude/agents/`.
- Shared skills across cloned repos (v2): **manifest + lockfile** (`seed.yaml` + `seed.lock`, skillfold semantics — SHA+sha256 pins, `install --frozen` in CI, managed-directory-only pruning); skill updates are PRs with injection review. v1 ships local skills only — no manifest needed.
- Workflows-as-files (v2): **YAML with a markdown-prompt escape hatch** (Archon middle path; step schema + validate rules in [inspirations/04](./research/inspirations/04-workflow-as-config.md)), `seed workflow validate` preflight + mock harness; `.claude/workflows/` additionally for Claude-native dynamic workflows.

## 4. Repository layout

Committed contract lives under `.seed/` (all control surface); **work products** (plans, receipts, memory, decisions) live at root on the default branch with their own gates; **machine-written coordination state lives on the `seed-state` ref** (§7.2); runtime scratch is excluded via `.git/info/exclude`. Items marked *(v2)* per §7.3.

```
open-seed/                     # default branch
├── AGENTS.md                  # agent instructions (+ managed rules block)
├── CLAUDE.md                  # "@AGENTS.md" import + Claude-specific notes
├── README.md
├── Makefile                   # make check — the one fast backpressure command
├── .mcp.json
├── .worktreeinclude           # gitignored-file propagation into worktrees (agent-deck format)
├── .gitattributes             # merge=union on decisions/** only
├── CODEOWNERS                 # control-surface review + §6 chapter leads
├── seed.yaml / seed.lock      # skills manifest + lockfile (v2)
├── .claude/                   # control surface (generated shims + settings)
│   ├── settings.json
│   ├── ci-settings.json       # settings profile used by CI workflows
│   ├── agents/                # fan-out of .seed/agents/ (do not edit here)
│   ├── skills/                # fan-out of skills/ (do not edit here)
│   └── workflows/             # Claude dynamic workflows (v2)
├── .agents/skills/            # cross-harness fan-out (agentskills.io convention)
├── skills/                    # Agent Skills source of truth
├── rules/                     # rule fragments synced into AGENTS.md managed block
├── plans/                     # plans/<task-id>.md — stable path, pinned by hash (D3)
├── receipts/                  # <task-id>.json — gated by CI verify (D4.5); the pr-review
│                              #   workflow adds <task-id>.review.md once activated (v1: the
│                              #   GitHub PR review itself is the review record)
├── memory/                    # LEARNINGS.md, DEADENDS.md — appended via task PRs (D5)
├── decisions/                 # ADRs, append-only, merge=union
├── .seed/                     # the orchestration contract (control surface, human-owned)
│   ├── config.toml            # active backend, defaults, roles→runtime map, squad priority
│   ├── guardrails.yaml        # autonomy tiers, budgets, protected paths, allowlists
│   ├── version                # protocol version int (enforced by the shim, exit 10)
│   ├── agents/                # role definitions incl. dispatcher.md
│   ├── teams/                 # squad definitions (§6)
│   ├── backends/              # backend plugins (filecards ships in-template)
│   ├── backends.lock.json     # SHA+hash pins for installed backend plugins
│   ├── workflows/             # workflow YAML (v2)
│   ├── port-schema/           # JSON Schemas for the task-port contract
│   └── hooks/                 # setup / run / teardown / post-create.d/ / pre-merge.d/
├── scripts/
│   ├── seed                   # CLI shim: task/receipt/sync/backend/hooks/init subcommands
│   ├── loop.sh                # Ralph loop: dual-gate exit, circuit breaker, budgets, lease renewal
│   └── validate.sh            # lint all orchestration artifacts (also run in CI)
├── .github/
│   ├── workflows/             # check+validate, seed-maintenance, seed-dispatch (inert),
│   │                          #   pr-review (inert — activates with dispatcher secrets)
│   └── ISSUE_TEMPLATE/        # machine-parseable forms — inert scaffolding until dispatcher/
│                              #   mirror activation; v1 card creation is `seed task create` only
└── docs/                      # this study + conventions handbook

# On the seed-state ref (created by `seed init`; never checked out directly):
#   tasks/                     # task cards (filecards backend)
#   run-log.jsonl              # append-only event log (one commit per verb)
#   handoff/<task-id>.md       # machine-generated handoff/reap notes (§7.1)
#   mail/<agent>/<msg-id>.yaml # one file per MESSAGE, never rewritten (v2)
```

## 5. What open-seed would be that nothing else is

Closest prior art, per the research: **tutti** (committed org-code TOML — with real caveats; see [inspirations/04](./research/inspirations/04-workflow-as-config.md)) + **beads** (committed task graph) + **loki-mode's gates** (deterministic verification with evidence receipts) + **loop-engineering** (checked-in loop conventions with autonomy tiers) + **bradygaster/squad** (repo-resident team charters). *No project combines these in template form.* The unclaimed gaps open-seed can own:

1. A **checked-in autonomy-tier + guardrails vocabulary** enforced by hooks, server-side protection, and CI.
2. The **task↔plan↔evidence chain**: card → hash-pinned gated plan → implementation → CI-verified receipt, all as diffable files.
3. A **blocking local pre-merge gate** in the worktree lifecycle contract (no surveyed worktree tool's lifecycle hooks block; platform merge gates require their runtime).
4. **Runner-agnostic degradation**: the same repo works with a lone human, one Claude Code session, Claude agent teams, a Ralph loop in CI, or any of the 60+ external orchestrators surveyed — because the contract is files, with documented shims.

## 6. Team layer: organizing work the squad-model way

**Direction (2026-08-22):** open-seed organizes work and executes in *teams*, modeled on the Spotify Squad Model (squads / tribes / chapters / guilds, "aligned autonomy"). Several surveyed projects independently reinvented pieces of it (bradygaster/squad, tutti's roles+scopes, kodo's team.json, gastown's crews, corellis/opengoat, Claude Code agent teams):

| Squad-model concept | open-seed realization | Precedent |
|---|---|---|
| **Squad** | `.seed/teams/<squad>.yaml`: `mission`, `lead` (a human), `members` (role refs, mixing humans and agents), `scope` (globs), `backlog` (card filter), `priority` (unique int), `rituals`, `tier` (≤ ceiling), `review: codeowners\|agent` (default `agent`) | squad's team.md; tutti scope; kodo team.json |
| **Tribe** | The repo (or org overlay); repo-level `guardrails.yaml` is the floor | qm overlays; corellis |
| **Chapter** | Role definitions (`.seed/agents/*.md`): one canonical definition per role; chapter lead = human CODEOWNER of that file | sub-agents-skills; opengoat; antfarm |
| **Guild** | Shared skills library (+ v2 manifest/lockfile) | skillfold; plugin marketplaces |
| **Mission/OKR alignment** | Goal ancestry on cards (`parent` links to the squad mission) | Paperclip; beads epics |
| **Autonomy within alignment** | Squads own *how*; tribe owns *what* (missions, guardrails, quality bar) | kodo; Fusion levels |

**Routing semantics** (normative for the v2 multi-squad activation; in v1 exactly one squad exists — `core`, holding the human `lead` and the default agent trio — and every rule below is trivially satisfied):

- **Tier precedence:** `guardrails.yaml` sets `autonomy.default_tier` and `autonomy.max_tier`; a squad's `tier ≤ max_tier` (a comparison the validator checks — avoiding the undecidable "tighten arbitrary globs/budgets" formulation). Squads cannot override protected paths, allowlists, or budgets. Per-squad *harness* enforcement happens at spawn time (generated per-worktree settings from the squad tier); the gate layer, which reads the squad tier directly, is the backstop.
- **Scope:** squads' `scope` globs may not overlap (validation error), except via an explicit shared-scope entry naming one owning squad. Files matching no scope belong to `core`.
- **Cross-squad work:** any squad's agents may PR into another squad's scope; **the scope owner's gate governs the merge** — concretely, the scope-owning squad's `tier` sets the merge autonomy for that PR, and its human `lead` is the CODEOWNERS entry for its scope paths when the squad sets `review: codeowners` (otherwise the owner squad's reviewer agent at the owner's tier). CODEOWNERS names humans, so any squad wanting human-gated scope review must have a human lead — every squad has one by schema.
- **Backlog:** a card routes to exactly one squad: explicit `squad:` field, else the matching squad with the lowest `priority` int (uniqueness validated), else `core`. No card can be invisible. `seed task ready --squad <name>` is an optional port capability (shim-side filtering otherwise).
- **Goal-ancestry validation** activates only when >1 squad or any mission is defined — a solo clone pays nothing.

The squad model's documented failure modes (per the [ideaplan case study](https://www.ideaplan.io/case-studies/spotify-squad-model): "the model as described in the whitepaper never fully existed in practice") map to mitigations that are *mechanical* here where they were aspirational for humans: **fragmentation** → standards are executable (CI, hooks, shared role files); **chapter-lead dysfunction** → the lead is a CODEOWNERS entry, zero people-management; **tribal silos** → cross-squad dependencies are typed `blocks`/`waits-for` edges visible in `ready` queries, and internal-open-source maps to cross-scope PRs under the owner's gate; **guild decline** → the guild is versioned artifacts, not volunteer energy; **alignment** → goal ancestry, validated per the activation rule. The article's real lesson — adopt the principles, not the org chart — is the template's posture: team files are conventions a project tunes.

## 7. Decisions made

**7.1 Coordination backends are plugins (decided 2026-08-21).** Whatever backend is used — task-card files, beads, GitHub Issues, Paperclip, Gas Town, or anything future — it is a **plugin behind a stable port interface**; nothing else in the template (scripts, skills, hooks, CI workflows, agent instructions) may talk to a backend directly. Plugin system: packaging (manifest + implementation), checked-in active-backend declaration, capability negotiation, lockfile pinning, and a trust model (pinned SHAs, review-before-install, plugin output treated as untrusted input). Spec: [`research/10-org-control-planes.md`](./research/10-org-control-planes.md) Part 5 — a **JSON-over-CLI port** (`seed task <verb> --json` → `.seed/backends/<name>/bin/seed-backend`); nine required verbs plus optional capabilities (lease-renew, ancestry, deps, event-emit, wake, budget, watch, squad-filtered ready); exit 2 = claim contention, exit 6 = fenced out (stale claim token), exit 10 = schema/version mismatch (the shim is the enforcement point for `.seed/version`; out-of-tree tools SHOULD check it but only the shim refuses); `backend.toml` capability manifest; `.seed/backends.lock.json` pins; MCP as the v2 transport.

**Claim protocol (binding):**
- `claim` is synchronous and completes *before any work begins*: for filecards the claim commit is pushed inside the verb; on push rejection the shim re-fetches and re-checks — task now claimed by another → exit 2. A "loser" never has a half-built worktree. A claim on an unplanned card authorizes planning only (D1).
- **Leases are mandatory on filecards** (default 60m, configurable); `lease-renew` is a required capability of the filecards backend, and the loop runner renews at half-lease cadence. **Leases apply only to `in_progress` cards.** A claim whose lease expires is reaped by the maintenance workflow (reap latency = maintenance cadence, documented; teams needing tighter leases use beads); reaping performs `in_progress → ready` (a release, *not* a rejection — `rejected_authors` is untouched) and writes a machine-generated **handoff stub** to `handoff/<task-id>.md` *on the state ref* (v1 — a few generated lines: last known branch, HEAD, lease timestamps; independent of the v2 handoff-packet automation) so the next claimant can salvage or reset.
- **Claim end-of-life:** the claim (and its lease) ends with the implementer's exit from `in_progress` — any of `in_progress → review|ready|blocked` is a claim-ending fenced act (`→ blocked` releases the claim exactly like a release, writing the handoff stub, so a crashed agent's blocked card is never stuck holding a dead claim and an unblocked card re-enters `ready` claim-free). Cards in `review`/`blocked` hold no claim and are never reaped; a stalled review is the maintenance workflow's *reporting* concern, not a lease event.
- **Verbs are classed, and the fence applies only where a claim exists.** *Worker verbs* — `transition ready→in_progress` (via claim), `in_progress → review|ready|blocked`, `comment`/`attach-evidence` and lease renewal while holding the claim — must present the current claim token; a reaped predecessor's late operations fail with **exit 6** (fenced out). *Operator verbs* — accept (`review → done`), reject (`review → ready`, appending the implementer to `rejected_authors`), `cancel`, reinstate (`cancelled → backlog`), promotion/deprioritization (`backlog → ready`, `ready → backlog`), manual blocking/unblocking of unclaimed cards (`ready → blocked`, `blocked → ready`) — require an **operator-class credential** (the squad's human lead, or the reviewer workflow's app identity once activated), not a claim token; the shim enforces the class by identity, and which principals hold operator credentials is part of Q5. (`blocked → ready` also fires automatically as the blocker-cascade — that path is the shim acting on `close`/`cancel`, not a third credential class.) Every legal transition in the D1 table is thus assigned to exactly one class. Card bookkeeping (`rejected_authors`, review block, claim block) is written by the shim as defined side effects of these verbs — there is no free-form field-set verb.
- **`close` is not a second path to done, and it happens after merge:** `close` = accept (`transition review → done`) plus the blocker-cascade, valid only from `review`; any other state is an invalid transition (exit 3). The normative sequence is **server PR approval → merge → close** — the cascade fires only once the default branch actually contains the work, so dependents never build on unmerged changes. The v1 caller is deterministic: a seed-maintenance step detects a merged `seed/<task-id>` PR with a green verify check and invokes `seed task close <task-id>` under the workflow's operator credential (a human lead may also close manually). `cancel` is terminal from any non-terminal state and also cascades. The D1 transition table is the single authority; [research/10 Part 5](./research/10-org-control-planes.md) is amended by erratum accordingly (exit 6, `--token` on worker verbs, `close` restriction, `ready --squad` capability).

**7.2 Coordination state lives on a dedicated ref (decided in review, 2026-08-22).** Mutable machine-written state — task cards, `run-log.jsonl`, handoff notes, v2 mail — lives on the branch **`seed-state`** (named *outside* the `seed/<task-id>` task-branch namespace so branch cleanup can never delete it), written only by the port shim. The default branch carries the human-owned contract, code, and reviewed/verified work products (plans, receipts, memory, decisions — see §4). Full lifecycle:

- **Write path:** fetch → commit → push, retrying with jittered backoff on rejection; **one commit per verb**, containing the card mutation *and* its run-log event line atomically. Reads (`ready`/`get`) fetch first — staleness is bounded and explicit.
- **Bootstrap:** GitHub template instantiation copies the default branch only, so `seed init` creates the orphan `seed-state` ref (and `seed init-github` verifies it plus the branch-protection settings). Creation races resolve trivially: if the create-push is rejected because the ref exists, fetch and proceed.
- **Integrity (honest limits, §8-R10):** any principal that can push `seed-state` can bypass the shim — integrity is *push-access-deep*, not cryptographic. Mitigations: branch protection on `seed-state` allowing pushes but **blocking force-pushes and deletion**; the shim treats an observed non-fast-forward rewrite as an incident (halt + escalate, never silently adopt). **Checkpoint anchors are protected tags, not default-branch commits** (a scheduled job pushing the default branch would be exactly the bypass credential D4.3 forbids): the maintenance workflow tags the current `seed-state` head as `seed-anchor/<timestamp>` under a tag-protection rule (create-only for the maintenance credential, no deletion); **on fetch, the shim verifies the newest anchor is an ancestor of the fetched head** — this gives fresh clones (which have no prior baseline) rewrite detection too; failure is the same halt+escalate incident. Semantic (fast-forward) tampering that survives the shim is caught by the maintenance conformance lint (transition-table conformance, lease/rejection consistency, run-log inclusion); **a conformance failure writes a `HALT` marker at the state-ref root — the shim refuses mutating verbs while it is present (reads warn) — cleared only by a human via `seed state resume` (operator credential)**. Reviewer-lockout, `rejected_authors`, and the run-log are exactly as trustworthy as this ref — stated plainly rather than implied otherwise; the *load-bearing* gates (plan approval, receipt verification, merge) deliberately ground on the default branch and server-attributed identities instead (D3, D4.5).
- **CI visibility:** pushes to `seed-state` trigger no workflows (its tree has no workflow files); all state validation (card lint, run-log commit-over-commit inclusion, transition-table conformance) runs in the check+validate and maintenance workflows, which **explicitly fetch the ref**; state-validation latency = their cadence.
- **Growth:** every verb is a commit; history grows unboundedly. Discipline: shims use shallow/filtered fetches; when truncation is warranted it is a *human maintenance operation* (documented: temporarily lift protection, checkpoint, re-root, bump `.seed/version` so stale shims refuse) — never automatic, because it conflicts with the no-force-push rule.
- **Forks/mirrors:** forks copy the default branch only; a fork wanting live coordination runs `seed init` for its own state ref. The state ref is single-remote by design in v1 (multi-remote sync is what beads' Dolt layer is for — that's the upgrade path).

**7.3 v1/v2 scope cut (decided in review, 2026-08-22).** **v1**: the port + `filecards` backend + `seed-state` ref (init, claim/lease/fence protocol, handoff stubs); task cards + transition table; plan grammar + hash pinning; receipts + `seed receipt verify` (CI author-of-record); `.seed/guardrails.yaml` + CODEOWNERS + validators; `.seed/hooks/` + `.worktreeinclude` + three tool shims; role definitions + fan-out sync; `loop.sh` (with lease renewal); memory/decisions conventions; CI = check+validate and seed-maintenance live (deterministic, no model secrets), seed-dispatch and pr-review in-tree but inert (v1 review is human-only until activation); one `core` squad. **v2**: `beads` and `github-issues` backends + the mirror exporter + state-label taxonomy; the workflow engine + mock harness; skills manifest/lockfile + compose; mailboxes + handoff-packet automation; multi-squad routing activation; remaining lifecycle shims; `paperclip` adapter; MCP transport.

**7.4 Teams are the organizing unit (direction, 2026-08-22).** Work is organized and executed in squads per §6; the one-member `core` squad is the degenerate default so the layer costs nothing for solo use.

## 8. Risks, gotchas, and mitigations

- **R1 — Fan-out drift.** Source tree vs per-harness copies diverge silently. Mitigation: fan-out generated only by `seed sync`; `seed sync --check` (offline, hash-based) in CI; "do not edit here" markers.
- **R2 — Windows portability.** Executable bits and symlinks are unreliable. Mitigation: copy-based fan-out; `seed hooks run <name>` fallback runner; `core.fileMode` documented; no case-colliding paths.
- **R3 — Untrusted content is everywhere.** Card bodies, issue text, PR comments, mail, and backend output are injected into agent context; cards are also *work orders*. Mitigation: shim schema-validates/sanitizes backend output; AGENTS.md treats task/mail content as data; the plan gate is the human/reviewer checkpoint before an unreviewed card becomes action above L1 (D1); CI trigger gating per claude-code-action, gh-aw as upgrade; backend plugins SHA-pinned; control surface excluded from auto-merge with server-side review (D4.1/D4.3).
- **R4 — One ref serializes all writes.** Beyond claim contention, *every* mutating verb (transitions, comments, event appends) contends on `seed-state` — the real ceiling is writes/minute on the ref, and retry storms start well below 10 agents if they are chatty. Mitigation: one commit per verb; jittered backoff; agents batch comments; the maintenance workflow reports contention; the documented upgrade for higher throughput is the beads backend. Never pretend the file backend is atomic or high-throughput.
- **R5 — Committed coordination state churns history.** Mitigation: machine writes on `seed-state` keep the default branch human-meaningful; one file per task and per mail message (never rewritten files under `merge=union`; union only on `decisions/**`); hash IDs; runtime scratch in `.git/info/exclude`; state-ref growth handled per §7.2.
- **R6 — Budgets are advisory without a server.** Only run-local caps are enforceable in-repo. Mitigation: say so in guardrails docs; enforce what's local (loop budgets, attempt caps, circuit breaker); post-hoc accounting in the run log; hard org-wide stops via gh-aw credits or a control plane (Paperclip) through the backend/event seams.
- **R7 — Harness churn.** Mitigation: the harness-neutral core (AGENTS.md, skills, port CLI, hooks) carries the semantics; harness files are generated shims; CI never depends on experimental harness features.
- **R8 — Template drift after clone.** Mitigation: small versioned update surface (`.seed/version`, lockfiles); `seed upgrade` guidance against tagged releases; evolving parts (skills, roles, workflows) also distributed as a plugin/marketplace (Q4).
- **R9 — Convention without enforcement rots.** Mitigation: `validate.sh` + CI check every orchestration artifact (cards, plans + pinning, receipts, guardrails incl. the intersection rule, team files, lockfiles, run-log inclusion, fan-out drift); "shipped convention" and "shipped validator" are one deliverable — which is why the v1 cut (§7.3) is small.
- **R10 — State-ref integrity is push-access-deep.** Anyone who can push `seed-state` can bypass the shim: clear a rejection list, forge lease fields, rewrite the log, or **forge an operator verb** (a spoofed `done` cannot reach a merged PR — merging grounds on required checks and server-attributed review, and no merge input reads card state — but it would fire the blocker-cascade and set a fleet building on unmerged work). Mitigation: no-force-push/no-delete protection on the ref; shim halts on observed non-FF rewrite; anchor-tag ancestry checks (§7.2); the maintenance done-consistency lint (D7) catches forged accepts and HALTs; the cascade's v1 trigger is the maintenance close step, which fires only on a verified merged PR; scope which principals hold state-ref push access (Q5); and honesty — audit claims about the run-log are conditional on this, everywhere they're made.
- **R11 — The gate runs on the implementer's credentials.** Local receipts and local pre-merge results are forgeable by the agent that produced the work. Mitigation: above L1, the CI verify check is the author of record for receipts and required checks are configured server-side (D4.3/D4.5); the local gate is a fast pre-check, not the authority.

## 9. Open questions to resolve next

1. **Harness posture:** Claude-Code-first with portable shims (recommended — richest primitive set), or strictly harness-neutral from day one (more work, lower ceiling)?
2. **Automation-on-clone:** §7.3 proposes check+validate and maintenance live, dispatcher inert until secrets, `claude-code-action` default with gh-aw documented. Needs your confirmation.
3. **Language/stack coupling:** language-agnostic (Makefile contract only), or opinionated stack variants (e.g. TS with lint/test wired)?
4. **Distribution model:** GitHub template repo, a Claude Code plugin/marketplace, or both — plugin carrying skills/agents/hooks, template carrying structure? (Interacts with R8.)
5. **State-ref principals:** which credentials may push `seed-state` (fine-grained PATs / deploy keys scoped to that ref vs. all contributors), given R10 — and whether v1 should default to a narrower set.
