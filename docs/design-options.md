# open-seed: Design Options

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
> standardized tooling for multi-agent orchestration, task tracking, and guardrails:
> an orchestration layer that organizes work and executes in teams.

---

## 1. The one finding that settles the positioning

Across every category, the same signal repeats: **orchestration that lives as reviewable repo content outlasts orchestration that lives in an app.**

- vibe-kanban hit 27.9k stars and is sunsetting; humanlayer hit 11.3k and deprecated its code, but humanlayer's *checked-in* research→plan→implement command convention was copied everywhere and survives its product.
- Archon (23.2k★) thrives specifically because its workflows are checked into target repos.
- The Ralph loop: the most influential methodology of the period, is literally one shell line plus markdown conventions in the repo.
- Every mature external app (agent-deck, superset, dmux, octomux, vibe-tree, amux) was *forced* to push a checked-in contract into user repos anyway: setup/teardown scripts, worktree hooks, workspace configs.
- gh-aw and aeon prove a full autonomous runner can be 100% repo content (markdown workflows compiled to committed Actions YAML).

So open-seed's bet: conventions, config, scripts, hooks, and CI checked into a template, is not just viable; it is the *durable* half of the entire ecosystem. The apps churn; the file conventions converge and persist. Corollary: open-seed should be **runner-agnostic**: everything must work with plain Claude Code (or Codex/Gemini) sessions, degrade gracefully to a single human + single agent, and get *better* (not different) when someone layers a TUI/desktop orchestrator or CI runner on top.

## 2. Settled ground (adopt, don't debate)

These have converged hard enough across 180 projects + vendor docs (as of Aug 2026) to treat as decided:

1. **Isolation = git worktree per task/agent**, one branch per unit of work, branch `seed/<task-id>` (see §7.2 for why the coordination ref lives *outside* this namespace). Containers/devcontainers are an optional second ring for untrusted/full-auto runs, never the default. (~40 of the surveyed projects use exactly this.)
2. **Portable instruction layer = AGENTS.md** (Linux Foundation-stewarded; read natively by every major harness except Claude Code, which imports it via a one-line `CLAUDE.md`), plus `GEMINI.md`/`.cursor/rules` shims only if needed.
3. **Portable capability layer = Agent Skills** (`<dir>/SKILL.md` per agentskills.io): adopted by Claude Code, Codex, Gemini CLI, Cursor, Copilot, OpenCode, and ~30 more. The cross-client checked-in location converging in the wild is **`.agents/skills/`**. Markdown-with-YAML-frontmatter is the universal file format for all agent config.
4. **Tools layer = MCP**, configured in a checked-in `.mcp.json` with env-var expansion for secrets.
5. **Fresh context per task, state on disk.** Conversation memory is a liability; files in the repo are the memory (Ralph doctrine, confirmed by every loop runner and by Anthropic's own context-engineering guidance).
6. **CI is the real guardrail ("backpressure").** Agents merge whatever passes, so a fast, deterministic `make check` + branch protection + required review is the quality system; everything else layers on top.
7. **Agent status vocabulary**: working / blocked(needs-you) / idle / done: the de-facto four-state enum every monitoring tool understands (tmux-ide's `@agent_state` grammar is the concrete wire format).

## 3. Design dimensions and options

### D1. Task-tracking substrate

The central architectural choice: resolved structurally by the plugin decision in §7 (all substrates sit behind one port), leaving only the *default*. Four options:

| Option | Evidence | Pros | Cons |
|---|---|---|---|
| **A. Task-card files in-repo** (one markdown+frontmatter file per task; schema in [inspirations/01](./research/inspirations/01-git-native-task-substrates.md)) | The dominant loop-runner convention (ralphex, Automaker, wreckit, gnap, tick-md) | Zero deps; diffable; auditable via ref history; works offline; works with any harness | Atomic claim only *emulated* (push-wins, see §7.1); all writes serialize through one ref (§8-R4); cards live on the state ref, so they are **not** PR-reviewed: the plan gate is the review point (see below); forks copy the default branch only, so coordination state does not travel with forks automatically |
| **B. Beads** (steveyegge/beads, 26.5k★): git-native graph tracker: hash IDs, typed dependency edges, `bd ready`, atomic `--claim`, native leases+heartbeats, AGENTS.md integration | What gastown, orc, and ralph-tui all bet on | Purpose-built for parallel agents; ready-work dispatch; crash-safe leases; memory across sessions | Binary dependency (Go + Dolt); Dolt-backed (embedded single-writer by default, server mode for concurrent writers) with an evolving schema: pin a version |
| **C. GitHub Issues + label state machine** (sortie's query-filter + label swaps, lalph, OpenHands) | The CI-automation category standard | Human visibility for free; API-native to Actions; server-atomic assignment; zero new infra | Rate-limited; slow for agent churn; no offline; state lives outside the clone |
| **D. Harness-native task lists** (Claude Code agent teams' shared task list) | Free intra-session coordination | Zero setup | Ephemeral, machine-local; never a system of record |

**Recommendation: A as the shipped default; B as the documented upgrade; C not as a backend but as an optional one-way *mirror* (v2).**

**Canonical state machine**: states `backlog, ready, in_progress, review, done, blocked, cancelled`; full edge table (backends and validators implement exactly this; anything else is an invalid-transition error):

| From | Allowed to |
|---|---|
| `backlog` | `ready`, `cancelled` |
| `ready` | `in_progress` (claim), `backlog` (deprioritize), `blocked`, `cancelled` |
| `in_progress` | `review`, `ready` (release), `blocked`, `cancelled` |
| `blocked` | `ready`, `cancelled` |
| `review` | `done` (accept), `ready` (reject), `cancelled` |
| `done` |: terminal; reopening = a new card linked `relates-to` |
| `cancelled` | `backlog` (reinstate) |

The two-stage done (Automaker/claude-command-center) is preserved as `review` (agent-finished) vs `done` (human/verifier-accepted). **Rejection bookkeeping:** on `review → ready`, the *rejected implementer* is appended to the card's `rejected_authors` list and the rejecting reviewer is recorded in the card's review block; the `claim` verb refuses claimants in `rejected_authors` (squad's reviewer-lockout, made mechanical: backends that cannot enforce it declare the capability absent). On any re-claim (after rejection, release, or lease reaping) the new claimant **resets the task branch, and any orphaned `seed/<task-id>-plan` branch** (delete + recreate from base) unless the handoff note (§7.1) marks the prior work salvageable.

**Cards are instructions, so their review point must be explicit.** Cards are machine-written to the state ref without PR review; a card body is untrusted input *and* a work order. The compensating control is normative: **above L1, implementation requires an approved plan**, and plans land on the default branch via reviewed PRs (D3), so the plan gate is where a human (or, at L2, a reviewer agent per guardrails) vets what the work order actually authorizes. Claiming an *unplanned* card is allowed at any tier but scopes the claimant to **planning only** (read-only exploration + authoring the plan PR); implementation may begin only once the plan is approved (merge-base rule, D3).

**The planning phase and its mutex, precisely** (leases only cover live agents, and plan review routinely outlasts a lease): the planner claims the card (`ready → in_progress`, leased) and authors the plan; **after opening the plan PR: strictly PR-first, then park, the shim parks the card `in_progress → blocked`**, a claim-ending act like any other exit from `in_progress`, and `blocked` cards are claim-free, unreaped, and *unclaimable*, so no rival planner can start while review is pending. **`blocked_on` is a normative, shim-written, multi-valued card field**: entries `plan:<pr-number>`, `dep:<task-id>`, `manual:<operator>`, and every unblock path removes *only its own entry*, transitioning to `ready` only when the set empties (without this, the blocker-cascade resolving a `dep:` edge on a plan-parked card would wrongly make it claimable mid-review). **Parking applies to every plan PR, amendments included** (D3): the parking handoff stub records `plan PR #n (amendment of <sha>)` and marks prior implementation work *salvageable*, so the post-amendment re-claim **rebases the existing task branch onto the new default head instead of resetting it**: an amendment must never destroy in-flight work. The PR-first ordering makes crash recovery uniform: a planner that crashes before opening the PR still holds a leased `in_progress` card and is reaped normally (the desired outcome); a crash after opening it leaves a parked card whose unblock is state-shaped, below. Maintenance's unblock condition is **state-shaped, not event-shaped**: `blocked_on` contains a `plan:` entry ∧ that PR is merged or closed ⇒ remove the entry (merged → the card now carries an approved plan at the default head; closed unmerged → re-planning).

**The GitHub mirror is a component, not a backend (v2).** When enabled, a one-way exporter renders card state to issue labels; **cards are authoritative and the export direction always wins**: human label edits are read back only as *requests* (the dispatcher turns them into `seed task transition` calls, which may refuse). Label mapping: `ready → seed:ready`, `in_progress → seed:in_progress`, `review → seed:review`, `done → seed:done`, `blocked → seed:blocked`; `backlog` = no state label; `cancelled` = issue closed as not-planned. (Supersedes `seed:working` in [inspirations/06](./research/inspirations/06-ci-native-automation.md).)

Key conventions regardless of substrate: **task ↔ branch ↔ worktree 1:1:1 mapping**; hash-based IDs, never sequential counters (tick-md's `next_id` is a guaranteed merge conflict); every closed task must reference evidence via its receipt (D4.5): except cards closed under the no-PR exemption (D7), whose evidence record is the server-attributed close artifact.

### D2. Orchestration topology

| Option | Evidence | When it wins |
|---|---|---|
| **A. Single-agent Ralph loop**: checked-in loop runner + plan file + fresh session per task | ralphex, ralph-claude-code, dex; 100% repo-native | Solo dev or one workstream; cheapest, most auditable |
| **B. Parallel flat worktrees**: N independent agents, human merges | claude-squad archetype, superset, emdash, all TUIs | Independent tasks; needs only worktree conventions + merge gate |
| **C. Coordinator–worker**: planner decomposes, workers execute, coordinator never codes | orc, kodo, gastown Mayor, Claude Code agent teams | Larger efforts; requires the task graph (D1) to be real |
| **D. Ticket-claim blackboard**: no assignment; agents wake and atomically claim ready work | paperclip heartbeats, beads `--claim`, gnap | Most robust to agent death; degrades gracefully to 1 agent; needs atomic claim |

**Recommendation: the paved road is *squad-shaped* (§6): a small named team with a mission, with A as its degenerate one-member case.** The worktree contract + task substrate + claim convention *are* the topology; the template doesn't hardcode a hierarchy beyond the team files. Escalation path: one agent → parallel worktrees → coordinator role prompt (a subagent/skill, not a daemon) → external orchestrator (gastown, Paperclip, etc.) as optional endgame. Claude Code's native agent teams + `.claude/workflows/` cover C natively for Claude shops: open-seed rides those primitives rather than reimplementing them, while keeping the durable task ledger in the repo (agent-team state is machine-local and ephemeral).

### D3. Plan/spec discipline

Options range from none → Ralph-thin (PROMPT.md + fix_plan.md) → plan-as-gated-artifact (Fusion, Ivy-Tendril, dex) → full SDD (spec-kit, BMAD).

**Recommendation: thin, mandatory, gated, pinned.** The single highest-leverage quality convention found across all categories is **plan-as-gated-artifact**: every non-trivial task produces a committed plan file (steps, file scope, acceptance criteria, `## Validation Commands`) that is reviewed *before* execution, and implementation is reviewed *against that same file*. Grammar (ralphex parser rules + frankbria Optional-section semantics + dex skip convention) in [inspirations/02](./research/inspirations/02-ralph-loop-implementations.md). Departure from precedent: in ralphex/dex, `## Validation Commands` is advisory; open-seed's loop runner and pre-merge gate also execute it **mechanically** (martin-loop's fresh-evidence rule).

Plan lifecycle mechanics:

- **One stable location.** Plans live at `plans/<task-id>.md`, forever, no `completed/` move (archival was cosmetic and had no legal writer; the card's state already says a task is closed). The plan path is derived from the card id (no `plan:` field needed); the validator lints that every card in `review`/`done` has a resolvable plan (`done` cards closed under the no-PR exemption (D7) are excluded, by the same server-artifact test the done-consistency lint uses).
- **Approval = the merge-base, not a card field.** The trust root is the default branch, never the state ref (which is push-access-deep, §8-R10). Normatively: *the approved plan for a task PR is the blob of `plans/<task-id>.md` at the PR's merge-base with the default branch.* The gate and CI parse `## Validation Commands` **from the merge-base blob, never from the PR head's copy** (an implementer-controlled checkout must not supply its own acceptance criteria: those commands run in CI with the workflow's credentials). The receipt records the merge-base plan's sha256 (and optionally the plan PR's merge commit); any `plan_hash` on the card is a cache for agents' convenience, never an authority. This is loki's spec-lock adapted to per-task plans with the pin *derived* rather than stored.
- **PR purity rule (the discriminator that makes the gates checkable).** Two disjoint PR classes, distinguishable server-side by head branch: **plan PRs** (branch `seed/<task-id>-plan`) touch *only* `plans/<task-id>.md`: one plan file, nothing else; **task PRs** (branch `seed/<task-id>`) may not touch `plans/**` *at all* (not just their own plan: a task PR burying an edit to another task's plan inside a large diff would launder plan tampering through an unrelated review). CI fails any PR violating its class. Implementation starts from a default-branch head that already contains the merged plan (the task branch is created or reset after plan approval), so the plan-PR and task-PR lifecycles never share a branch.
- **Plan-PR review is enforced, not assumed.** Who reviews a plan PR, by tier: v1 (and any tier before automation secrets exist): a human; after pr-review activation, L2/L3: the reviewer workflow's server-attributed app identity, which must differ from the plan author (same identity rule as D4.5). `plans/**` is deliberately not CODEOWNERS control surface, but it is **never auto-mergeable either**: the guardrails validator rejects any auto-merge allowlist entry intersecting `plans/**` exactly as it does for the control surface (otherwise an L3 agent could approve its own work order: author a plan whose validation commands are attacker shell, auto-merge it unreviewed, and have CI execute it).
- **No stale-plan replay.** A superseded plan must be revocable: `seed receipt verify` fails when the merge-base plan blob differs from the plan blob at the current default-branch head ("plan changed since branc base — rebase and re-verify"). Without this, an implementer could keep executing a revoked plan v1 forever by never updating their branch. This is deliberately enforced **per-PR in the verify check, not via repo-wide "require branches to be up to date"**: the strict setting applies to every PR against the protected branch and would serialize all merges (each landing invalidates every other open PR's checks and forces full re-runs, including verify's command execution), directly fighting the N-way parallel posture (§8-R12). Teams wanting semantic-merge safety at scale should enable a **merge queue: which requires verify-check adaptation, not "the same checks" unchanged**: queue checks report against temporary `merge_group` refs where the PR-event trigger and head-branch classification both vanish (a `pull_request`-only verify stalls every queued PR; a naive `merge_group` trigger classifies the queue ref as a non-task PR and passes vacuously, losing stale-plan detection at exactly the landing moment). The specified adaptation: verify also subscribes to `merge_group`, derives the PR number from the queue ref name, classifies by *that PR's* head branch via the API, and re-runs the merge-base/stale-plan logic against the merge-group base.
- **Amending a plan mid-flight** (the purity rule bars editing `plans/**` in the task PR, so the recourse must be explicit): open a new `seed/<task-id>-plan` PR with the amended plan → it is reviewed and merges → rebase `seed/<task-id>` onto the new default head, which updates the merge-base blob (and thus the executed `## Validation Commands`); the stale-plan check forces exactly this rebase, so an un-rebased task PR cannot silently keep running the old plan.

Full SDD frameworks (spec-kit/BMAD) are contested and heavy; they remain optional adapters, not the core.

### D4. Guardrails stack

Defense in depth; every layer is checked-in-able. The `guardrails.yaml` skeleton in [inspirations/03](./research/inspirations/03-governance-and-gates.md) is the starting point, superseded here per its erratum.

1. **Policy file** (`.seed/guardrails.yaml`): protected-path globs (restriction-only: matching changes require a human decision), auto-merge allowlist vs denylist, `max_files_per_change`, budgets (caveat §8-R6), and **named autonomy tiers** (L1 report-only / L2 assisted-in-worktree / L3 unattended-with-gates). **Hard rule: the orchestration *control surface* is never auto-mergeable and always protected, regardless of any allowlist entry.** The control surface is exactly: `AGENTS.md`, `CLAUDE.md`, `skills/**`, `rules/**`, `.seed/**`, `.claude/**`, `.agents/**`, `scripts/**`, `Makefile`, `.github/**`, `.mcp.json`, `seed.yaml`, `seed.lock`, `.worktreeinclude`, `.gitattributes`, `CODEOWNERS`. Deliberately **not** control surface (they are work products with their own gates): `plans/` (integrity via pinning, D3), `receipts/` (integrity via the CI verify check, D4.5), `memory/`, `decisions/`, `docs/`, and source code (governed by the ordinary allowlist/denylist + review tiers). After the §7.2/§7.3 split, everything under `.seed/` is genuinely human-owned contract, so the blanket glob is safe. The validator rejects any guardrails file whose auto-merge allowlist intersects the control surface **or `plans/**`** (plans are the trust root for what CI executes: D3). A naive `**/*.md` allowlist would hand agents their own instruction surface: most of which is markdown, so the validator enforces this structurally, not by convention.
2. **Harness enforcement**: `.claude/settings.json` permissions + hooks (PreToolUse denials of destructive commands and protected paths; Stop/TaskCompleted completion gates), mirrored minimally for Codex/OpenCode. Best-effort per harness; the layers below are the backstop.
3. **Git/server enforcement: the real gate for L3.** Client-side hooks and CI jobs that run *inside a PR* can be edited by that same PR (GitHub runs the PR's own workflow files for `pull_request` events), so they cannot be the last line of defense. L3 auto-merge is sound **only** with server-side settings the template documents and scripts (`seed init-github`): branch protection with required status checks configured in repo settings, required review via the shipped **CODEOWNERS** covering the control surface, and no *agent* credentials that bypass them. **Solo mode (explicit, because GitHub forbids approving your own PR):** when the repo has one human, `seed init-github` configures enforce-admins *off* so the human owner can merge their own control-surface PRs; required CODEOWNERS review activates when a second human joins. Without this, a solo user could never customize `.seed/guardrails.yaml` after cloning: deadlocking the template's primary degradation case (§1). **Prerequisite it prints when flipping enforce-admins off:** "admin bypass is for humans only" has no mechanism if agents run under the owner's own credential (the default solo setup), so solo operation above L1/L2 **requires agents to run under a separate non-admin machine identity** (fine-grained PAT or app with no bypass); until one exists, the effective tier is capped and `seed init-github` says so. Client-side: pre-commit hooks (gitleaks, contract checks) and the **blocking `pre-merge.d/` gate**, no surveyed *worktree tool's lifecycle hook* blocks a merge (dmux's `pre_merge` is detached), so a genuinely blocking local gate is novel in that tool class (platform-side merge gates exist: tutti's land gate, Fusion's pre-merge phase). Agents never push the default branch (all default-branch changes arrive via PR); coordination writes go to the state ref (§7.2).
4. **CI enforcement**: one fast `make check`; CI also validates the orchestration files: task-card/plan lint (fetching the state ref explicitly: see §7.2 CI visibility), plan-hash pinning, **receipt verification** (D4.5), guardrails schema check (including the control-surface/allowlist intersection rule), fan-out drift check (§8-R1), team-file rules (§6), and, v2, once the manifest exists: skills-lock `--frozen`. The run-log integrity check runs in the maintenance job against the state ref's own history (commit-over-commit inclusion; there is no merge base to compare against: the shim linearizes writes), with the honest caveat that it is only as strong as the ref's integrity (§8-R10).
5. **Verification separation and receipts.** No agent verifies its own work (antfarm, kodo, loki-mode; reviewer-lockout per D1). Ship a read-only **reviewer role** distinct from the implementer. Receipts live at root **`receipts/<task-id>.json`** (work product, not control surface) and travel with the task's PR. Authorship is tiered honestly:
   - The local gate (`pre-merge.d/` + `seed` CLI) writes a receipt as a *pre-check*, but it runs under the implementer's credentials, so **locally-written receipts are advisory** (an agent with Bash could fabricate one; loki says this out loud: "unsigned receipts are defense-in-depth, not non-forgeability").
   - **For anything above L1, CI is the author of record**: the required `seed receipt verify` status check regenerates the facts: the diff is the **merge-base diff excluding `receipts/**`** (excluding receipts makes the hash stable when the receipt itself is committed; merge-base makes it stable as the default branch advances), verify commands are re-run in the CI environment **from the merge-base plan blob (D3), never the PR head's copy**, protected paths re-scanned, plan blob hash re-derived (including the stale-plan check, D3), and fails on any mismatch with the committed receipt, so a forged local receipt buys nothing.
   - **The check cannot be evaded by omission.** `seed receipt verify` runs on every PR and classifies by head branch (the D3 purity rule): a `seed/<task-id>` PR **fails**, not skips, if it lacks its receipt, a resolvable approved plan, or a qualifying review; a `seed/<task-id>-plan` PR is checked against the plan-purity and plan-review rules; any other PR (docs, human feature work) passes the receipt check vacuously and is governed by the repo's ordinary review baseline: which the template sets as branch-protection required review for everything, CODEOWNERS adding the *mandatory* reviewers only on the control surface.
   - **Reviewer identity is bound to server-attributed facts, not commit metadata** (git author strings are arbitrary client-set text): the accepting verdict must be a GitHub PR review (server-authenticated) or a commit with a verified signature from the review identity; the implementer's identity is the PR author login; the verify step checks the two differ. In v1 the reviewer is a human; a reviewer-agent workflow joins when dispatcher secrets are configured (§7.3), using its own app identity.
   - Facts/assessments/honesty split per loki: headline computed from facts alone; every skipped check listed as a gap; refuse "done" on empty diffs.
6. **Secrets**: never in agent-readable config; env/`settings.local.json`/`.mcp.json` expansion; gitleaks in pre-commit + CI; secrets *names* documented, values never.

### D5. Memory conventions

Converged pattern worth shipping: **`memory/LEARNINGS.md`** (build/test commands, discoveries: the Ralph-family "AGENT.md", renamed to avoid confusion with `AGENTS.md`), **`memory/DEADENDS.md`** (failed approaches: dex), and **`decisions/`** (ADR-style; `merge=union` in `.gitattributes` applies to `decisions/**`: safe because ADR files are genuinely append-only). All three live at the **repo root on the default branch, outside the control surface**, and their legal write path is the task PR itself: an agent appends learnings/dead-ends in its task branch and they land with the work, reviewed at the same tier as the code (this resolves "agent-appendable" with "agents never push the default branch"). The high-frequency machine log (`run-log.jsonl`) is not a memory file: it lives on the state ref (§7.2). Mutation rules matter more than filenames: agents append, never rewrite; humans only edit guardrails; only the gate/CI writes receipts. Personal-assistant-style `MEMORY.md`/daily-notes rollups are overkill: skip.

### D6. Lifecycle/workspace contract

Full contract (hook names, env vars, exit-code semantics, per-tool shims) in [inspirations/07](./research/inspirations/07-lifecycle-contracts.md). Summary: `.seed/hooks/` with **`setup`, `run`, `teardown`** single-file hooks plus `post-create.d/` and `pre-merge.d/` run-parts dirs; executable-bit-as-opt-in, spawned without a shell, context via `SEED_*` env vars only; setup/teardown advisory (never strand a worktree), **pre-merge gating** (the one blocking set, with the D4.3 caveat that server-side protection is the backstop); a root **`.worktreeinclude`** adopting agent-deck's format verbatim. Ship shims for the top three external tools in v1 (superset, agent-deck, vibe-tree), the rest v2.

### D7. CI/automation layer

Actions-native proves out fully (gh-aw, claude-code-action, aeon). v1 ships two live workflows and one inert one (§7.3):

- **check+validate** (live): `make check` + `scripts/validate.sh` (all D4.4 checks), fetching the state ref for card lint.
- **seed-maintenance** (live): **deterministic scripting only: requires no model secrets**: lease reaping (§7.1), orphaned-claim recovery (contrabass pattern: `in_progress` card with no live branch → release), **closing merged tasks** (merged `seed/<id>` PR + green verify ⇒ `seed task close <id>`, §7.1), **unblocking cards whose plan PR merged or closed** (§D1 planning phase), the **done-consistency lint** (every `done` card must correspond to a merged `seed/<id>` PR and a `receipts/<id>.json` on the default branch, **or** to a **server-attributed** human no-PR close: the `--no-pr` path executes via `workflow_dispatch`, whose triggering actor GitHub records server-side, or must cite a GitHub comment authored by the closing human; **the lint validates against that server artifact, never against card text, and the actor must resolve to a member of the human-operator roster** (the `lead` humans in `.seed/teams/*.yaml` / CODEOWNERS): a machine identity as actor is a forged accept and HALTs; "not a `[bot]` suffix" is *not* a humanness check, since agent PATs ride ordinary-looking machine user accounts: card-resident annotations are forgeable at exactly the trust level this lint polices (R10), and D4.5's "server-attributed facts, not commit metadata" rule extends to this verb; anything else is a forged accept and writes the HALT marker, §7.2. **This clause is the normative no-PR exemption, defined once and referenced by every validator:** a `done` card whose evidence is the server-attributed no-PR artifact is exempt from the plan-resolvability lint (D3) and the receipt-reference rule (D1/D4.5): its evidence record *is* the dispatch-run or cited-comment URL, which the shim records in the card's review block; any validator touching `done` cards must apply this exemption by the same server-artifact test the lint uses, never by card text, so no binding validator can accept a state another rejects), state-ref checkpoint anchoring (§7.2), contention-rate reporting, expiry cleanup of `by:agent` artifacts. Scheduled with scattered minutes (never `:00`), assuming missed ticks (aeon's debt-model catch-up). Mirror export joins this workflow in v2.
- **seed-dispatch** and **pr-review** (both in-tree, inert until secrets configured): built on `anthropics/claude-code-action@v1` (write-access trigger gating, per-command Bash allowlists, branch-not-PR default), with **gh-aw as the documented upgrade** (read-only-agent + safe-outputs + sanitization + egress firewall + real budget enforcement: §8-R6). Routing instructions live at `.seed/agents/dispatcher.md`. Until activation, **v1 review is human-only** (D4.5's identity check accepts a human PR review); pr-review's activation adds the reviewer-agent lane under its own app identity.

**Workflows obey the port rule (§7.1):** all task-state mutations inside workflows go through `scripts/seed task <verb>`; label changes are performed only by the v2 mirror-export step rendering card state (the inspirations/06 draft's direct label swaps should be read as that step: see its erratum). One-shot command labels `cmd:*` (auto-removed on activation) and the forced `by:agent` provenance label ship with the dispatcher; the full state-label taxonomy activates with the v2 mirror. Provenance conventions everywhere: `[ai]` title prefixes, hidden `<!-- seed-workflow-id -->` body markers, sticky progress comments.

### D8. Skills/agents packaging & harness portability

- Portable core: `AGENTS.md` + a source-of-truth skills tree + `.mcp.json`; per-harness **fan-out by copy, not symlink** (symlinks break on Windows checkouts, zip downloads, some CI sandboxes) into `.claude/skills/` and `.agents/skills/`, drift-policed offline (§8-R1). Source-of-truth paths are root **`skills/`** and **`rules/`** (supersedes inspirations/05's `seed/skills/` path: content dirs stay at root; `.seed/` is reserved for the orchestration contract); rules additionally sync into a marker-fenced managed block in `AGENTS.md` (skillfold's byte-exact round-trip, offline-verifiable).
- Agent role definitions as markdown-with-frontmatter in `.seed/agents/*.md` (dual-format: Claude Code subagent fields + sub-agents-skills' `run-agent`/`permission`/`## Done When`; schema in [inspirations/05](./research/inspirations/05-skills-packaging.md)), fanned out unchanged to `.claude/agents/`.
- Shared skills across cloned repos (v2): **manifest + lockfile** (`seed.yaml` + `seed.lock`, skillfold semantics: SHA+sha256 pins, `install --frozen` in CI, managed-directory-only pruning); skill updates are PRs with injection review. v1 ships local skills only, no manifest needed.
- Workflows-as-files (v2): **YAML with a markdown-prompt escape hatch** (Archon middle path; step schema + validate rules in [inspirations/04](./research/inspirations/04-workflow-as-config.md)), `seed workflow validate` preflight + mock harness; `.claude/workflows/` additionally for Claude-native dynamic workflows.

## 4. Repository layout

Committed contract lives under `.seed/` (all control surface); **work products** (plans, receipts, memory, decisions) live at root on the default branch with their own gates; **machine-written coordination state lives on the `seed-state` ref** (§7.2); runtime scratch is excluded via `.git/info/exclude`. Items marked *(v2)* per §7.3.

```
open-seed/                     # default branch
├── AGENTS.md                  # agent instructions (+ managed rules block)
├── CLAUDE.md                  # "@AGENTS.md" import + Claude-specific notes
├── README.md
├── Makefile                   # make check: the one fast backpressure command
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
├── plans/                     # plans/<task-id>.md: stable path, pinned by hash (D3)
├── receipts/                  # <task-id>.json: gated by CI verify (D4.5); the pr-review
│                              #   workflow adds <task-id>.review.md once activated (v1: the
│                              #   GitHub PR review itself is the review record)
├── memory/                    # LEARNINGS.md, DEADENDS.md: appended via task PRs (D5)
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
│   ├── seed                   # bootstrap shim: verifies pin + hash, execs the engine binary (§7.5)
│   ├── seed.ps1               # Windows twin of the bootstrap shim
│   ├── loop.sh                # Ralph loop: dual-gate exit, circuit breaker, budgets, lease renewal
│   └── validate.sh            # lint all orchestration artifacts (also run in CI)
├── .github/
│   ├── workflows/             # check+validate, seed-maintenance, seed-dispatch (inert),
│   │                          #   pr-review (inert: activates with dispatcher secrets)
│   └── ISSUE_TEMPLATE/        # machine-parseable forms: inert scaffolding until dispatcher/
│                              #   mirror activation; v1 card creation is `seed task create` only
└── docs/                      # this study + conventions handbook

# On the seed-state ref (created by `seed init`; never checked out directly):
#   tasks/                     # task cards (filecards backend)
#   run-log.jsonl              # append-only event log (one commit per verb)
#   handoff/<task-id>.md       # machine-generated handoff/reap notes (§7.1)
#   mail/<agent>/<msg-id>.yaml # one file per MESSAGE, never rewritten (v2)
```

## 5. What open-seed would be that nothing else is

Closest prior art, per the research: **tutti** (committed org-code TOML: with real caveats; see [inspirations/04](./research/inspirations/04-workflow-as-config.md)) + **beads** (committed task graph) + **loki-mode's gates** (deterministic verification with evidence receipts) + **loop-engineering** (checked-in loop conventions with autonomy tiers) + **bradygaster/squad** (repo-resident team charters). *No project combines these in template form.* The unclaimed gaps open-seed can own:

1. A **checked-in autonomy-tier + guardrails vocabulary** enforced by hooks, server-side protection, and CI.
2. The **task↔plan↔evidence chain**: card → hash-pinned gated plan → implementation → CI-verified receipt, all as diffable files.
3. A **blocking local pre-merge gate** in the worktree lifecycle contract (no surveyed worktree tool's lifecycle hooks block; platform merge gates require their runtime).
4. **Runner-agnostic degradation**: the same repo works with a lone human, one Claude Code session, Claude agent teams, a Ralph loop in CI, or any of the 60+ external orchestrators surveyed, because the contract is files, with documented shims.

## 6. Team layer: organizing work the squad-model way

**Direction (2026-08-22):** open-seed organizes work and executes in *teams*, modeled on the Spotify Squad Model (squads / tribes / chapters / guilds, "aligned autonomy"). Several surveyed projects independently reinvented pieces of it (bradygaster/squad, tutti's roles+scopes, kodo's team.json, gastown's crews, corellis/opengoat, Claude Code agent teams):

| Squad-model concept | open-seed realization | Precedent |
|---|---|---|
| **Squad** | `.seed/teams/<squad>.yaml`: `mission`, `lead` (a human), `members` (role refs, mixing humans and agents), `scope` (globs), `backlog` (card filter), `priority` (unique int), `rituals`, `tier` (≤ ceiling), `review: codeowners\|agent` (default `agent`) | squad's team.md; tutti scope; kodo team.json |
| **Tribe** | The repo (or org overlay); repo-level `guardrails.yaml` is the floor | qm overlays; corellis |
| **Chapter** | Role definitions (`.seed/agents/*.md`): one canonical definition per role; chapter lead = human CODEOWNER of that file. **Per-squad harness variance is allowed in *binding*, never in *craft***: a role may have variants (`reviewer.codex.md`, `reviewer.gemini.md`) that differ **only in frontmatter** (`run-agent`, `model`, `effort`): the validator enforces body-identity across a role's variants by hash, so the chapter's standard (`## Task`/`## Done When`) stays uniform while squads pick engines; the chapter lead's CODEOWNERS entry covers all variants. Caveat the validator surfaces: permission-tier semantics differ per harness (the same `safe-edit` maps to materially different sandbox flags per CLI: inspirations/05), so a variant is also a *guardrails* variance and the tier mapping table in [inspirations/09](./research/inspirations/09-harnesses.md) is the reference for what each harness can faithfully enforce (notably: Gemini's safe-edit and Pi's tiers cannot be implemented faithfully: declared variance, not silent difference) | sub-agents-skills; opengoat; antfarm |
| **Guild** | Shared skills library (+ v2 manifest/lockfile) | skillfold; plugin marketplaces |
| **Mission/OKR alignment** | Goal ancestry on cards (`parent` links to the squad mission) | Paperclip; beads epics |
| **Autonomy within alignment** | Squads own *how*; tribe owns *what* (missions, guardrails, quality bar) | kodo; Fusion levels |

**Routing semantics** (normative for the v2 multi-squad activation; in v1 exactly one squad exists: `core`, holding the human `lead` and the default agent trio, and every rule below is trivially satisfied):

- **Tier precedence:** `guardrails.yaml` sets `autonomy.default_tier` and `autonomy.max_tier`; a squad's `tier ≤ max_tier` (a comparison the validator checks: avoiding the undecidable "tighten arbitrary globs/budgets" formulation). Squads cannot override protected paths, allowlists, or budgets. Per-squad *harness* enforcement happens at spawn time (generated per-worktree settings from the squad tier); the gate layer, which reads the squad tier directly, is the backstop.
- **Scope:** squads' `scope` globs may not overlap (validation error), except via an explicit shared-scope entry naming one owning squad. Files matching no scope belong to `core`.
- **Cross-squad work:** any squad's agents may PR into another squad's scope; **the scope owner's gate governs the merge**: concretely, the scope-owning squad's `tier` sets the merge autonomy for that PR, and its human `lead` is the CODEOWNERS entry for its scope paths when the squad sets `review: codeowners` (otherwise the owner squad's reviewer agent at the owner's tier). CODEOWNERS names humans, so any squad wanting human-gated scope review must have a human lead: every squad has one by schema.
- **Backlog:** a card routes to exactly one squad: explicit `squad:` field, else the matching squad with the lowest `priority` int (uniqueness validated), else `core`. No card can be invisible. `seed task ready --squad <name>` is an optional port capability (shim-side filtering otherwise).
- **Goal-ancestry validation** activates only when >1 squad or any mission is defined: a solo clone pays nothing.

The squad model's documented failure modes (per the [ideaplan case study](https://www.ideaplan.io/case-studies/spotify-squad-model): "the model as described in the whitepaper never fully existed in practice") map to mitigations that are *mechanical* here where they were aspirational for humans: **fragmentation** → standards are executable (CI, hooks, shared role files); **chapter-lead dysfunction** → the lead is a CODEOWNERS entry, zero people-management; **tribal silos** → cross-squad dependencies are typed `blocks`/`waits-for` edges visible in `ready` queries, and internal-open-source maps to cross-scope PRs under the owner's gate; **guild decline** → the guild is versioned artifacts, not volunteer energy; **alignment** → goal ancestry, validated per the activation rule. The article's real lesson: adopt the principles, not the org chart: is the template's posture: team files are conventions a project tunes.

## 7. Decisions made

**7.1 Coordination backends are plugins (decided 2026-08-21).** Whatever backend is used: task-card files, beads, GitHub Issues, Paperclip, Gas Town, or anything future, it is a **plugin behind a stable port interface**; nothing else in the template (scripts, skills, hooks, CI workflows, agent instructions) may talk to a backend directly. Plugin system: packaging (manifest + implementation), checked-in active-backend declaration, capability negotiation, lockfile pinning, and a trust model (pinned SHAs, review-before-install, plugin output treated as untrusted input). Spec: [`research/10-org-control-planes.md`](./research/10-org-control-planes.md) Part 5: a **JSON-over-CLI port** (`seed task <verb> --json` → `.seed/backends/<name>/bin/seed-backend`); nine required verbs plus optional capabilities (lease-renew, ancestry, deps, event-emit, wake, budget, watch, squad-filtered ready); exit 2 = claim contention, exit 6 = fenced out (stale claim token), exit 10 = schema/version mismatch (the shim is the enforcement point for `.seed/version`; out-of-tree tools SHOULD check it but only the shim refuses); `backend.toml` capability manifest; `.seed/backends.lock.json` pins; MCP as the v2 transport. **Amended 2026-08-22 (builtin stores):** the plugin seam is the boundary for *external substrates*; **builtin stores live inside the engine** behind the same port, verbs, and capability manifest, selected when the manifest declares `entry = "builtin"`: `filecards` has been exactly this since v1, and `fastcards` (§7.3) joins it. The trust model differs deliberately and is declared, not silent: a builtin rides the engine's own pin (`.seed/engine.lock` hash + release attestation), not a `backends.lock.json` directory hash, so "review the plugin" collapses into "review the engine upgrade": the same reviewed-PR gate. Nothing outside the engine may distinguish builtin from plugin: callers see one port.

**Claim protocol (binding):**
- `claim` is synchronous and completes *before any work begins*: for filecards the claim commit is pushed inside the verb; on push rejection the shim re-fetches and re-checks: task now claimed by another → exit 2. A "loser" never has a half-built worktree. A claim on an unplanned card authorizes planning only (D1).
- **Leases are mandatory on filecards** (default 60m, configurable); `lease-renew` is a required capability of the filecards backend, and the loop runner renews at half-lease cadence. **Leases apply only to `in_progress` cards.** A claim whose lease expires is reaped by the maintenance workflow (reap latency = maintenance cadence, documented; teams needing tighter leases use beads); reaping performs `in_progress → ready` (a release, *not* a rejection: `rejected_authors` is untouched) and writes a machine-generated **handoff stub** to `handoff/<task-id>.md` *on the state ref* (v1: a few generated lines: last known branch, HEAD, lease timestamps; independent of the v2 handoff-packet automation) so the next claimant can salvage or reset.
- **Claim end-of-life:** the claim (and its lease) ends with *any* exit from `in_progress`: the implementer's fenced exits `in_progress → review|ready|blocked` (`→ blocked` releases the claim exactly like a release, writing the handoff stub, so a crashed agent's blocked card is never stuck holding a dead claim and an unblocked card re-enters `ready` claim-free), **and the operator's `cancel` (`in_progress → cancelled`), which clears the claim and lease as a defined shim side effect and writes the handoff stub** (so a later reinstate finds the salvage note; the operator presents no claim token: cancel is operator-class, below). No state other than `in_progress` ever carries a claim. Cards in `review`/`blocked` hold no claim and are never reaped; a stalled review is the maintenance workflow's *reporting* concern, not a lease event.
- **Verbs are classed, and the fence applies only where a claim exists.** *Worker verbs*: `transition ready→in_progress` (via claim), `in_progress → review|ready|blocked`, `comment`/`attach-evidence` and lease renewal while holding the claim, are fenced, **with `claim` as the token-issuing bootstrap exception**: a `ready` card carries no claim, so `claim` presents no token: it *mints* a fresh one, written into the card's claim block by the claim commit itself, with contention resolved push-wins (exit 2, first bullet). Every subsequent worker verb must present the current claim token; a reaped predecessor's late operations fail with **exit 6** (fenced out). *Operator verbs*: accept (`review → done`), reject (`review → ready`, appending the implementer to `rejected_authors`), `cancel`, reinstate (`cancelled → backlog`), promotion/deprioritization (`backlog → ready`, `ready → backlog`), manual blocking/unblocking of unclaimed cards (`ready → blocked`, `blocked → ready`): require an **operator-class credential** (the squad's human lead, or the reviewer workflow's app identity once activated), not a claim token; the shim enforces the class by identity, and which principals hold operator credentials is part of Q5. (`blocked → ready` also fires automatically in two shim-mediated paths: the blocker-cascade on `close`/`cancel` removing a `dep:` entry, and the maintenance workflow's plan-unblock (§D1/D7) removing a `plan:` entry under its operator credential; each removes only its own `blocked_on` entry, and the transition happens only when the set empties.) Every legal transition in the D1 table is thus assigned to exactly one class. Card bookkeeping (`rejected_authors`, review block, claim block) is written by the shim as defined side effects of these verbs: there is no free-form field-set verb.
- **`close` is not a second path to done, and it happens after merge:** `close` = accept (`transition review → done`) plus the blocker-cascade, valid only from `review`; any other state is an invalid transition (exit 3). The normative sequence is **server PR approval → merge → close**: the cascade fires only once the default branch actually contains the work, so dependents never build on unmerged changes. The v1 caller is deterministic: a seed-maintenance step detects a merged `seed/<task-id>` PR with a green verify check and invokes `seed task close <task-id>` under the workflow's operator credential (a human lead may also close manually: for cards whose work never lands as a `seed/<id>` PR, via the `--no-pr` `workflow_dispatch` path so the act is server-attributed to the human and the done-consistency lint reads it as legitimate rather than forged: D7). `cancel` is terminal from any non-terminal state and also cascades. The D1 transition table is the single authority; [research/10 Part 5](./research/10-org-control-planes.md) is amended by erratum accordingly (exit 6, `--token` on worker verbs, `close` restriction, `ready --squad` capability).

**7.2 Coordination state lives on a dedicated ref (decided in review, 2026-08-22).** Mutable machine-written state: task cards, `run-log.jsonl`, handoff notes, v2 mail, lives on the branch **`seed-state`** (named *outside* the `seed/<task-id>` task-branch namespace so branch cleanup can never delete it), written only by the port shim. The default branch carries the human-owned contract, code, and reviewed/verified work products (plans, receipts, memory, decisions: see §4). Full lifecycle:

- **Write path:** fetch → commit → push, retrying with jittered backoff on rejection; **one commit per verb**, containing the card mutation *and* its run-log event line atomically. Reads (`ready`/`get`) fetch first: staleness is bounded and explicit.
- **Bootstrap:** GitHub template instantiation copies the default branch only, so `seed init` creates the orphan `seed-state` ref (and `seed init-github` verifies it plus the branch-protection settings). Creation races resolve trivially: if the create-push is rejected because the ref exists, fetch and proceed.
- **Integrity (honest limits, §8-R10):** any principal that can push `seed-state` can bypass the shim: integrity is *push-access-deep*, not cryptographic. Mitigations: branch protection on `seed-state` allowing pushes but **blocking force-pushes and deletion**; the shim treats an observed non-fast-forward rewrite as an incident (halt + escalate, never silently adopt). **Checkpoint anchors are protected tags, not default-branch commits** (a scheduled job pushing the default branch would be exactly the bypass credential D4.3 forbids): the maintenance workflow tags the current `seed-state` head as `seed-anchor/<timestamp>` under a tag-protection rule (create-only for the maintenance credential, no deletion); **on fetch, the shim verifies the newest anchor is an ancestor of the fetched head**: this gives fresh clones (which have no prior baseline) rewrite detection too; failure is the same halt+escalate incident. Semantic (fast-forward) tampering that survives the shim is caught by the maintenance conformance lint (transition-table conformance, lease/rejection consistency, run-log inclusion); **a conformance failure writes a `HALT` marker at the state-ref root: the shim refuses mutating verbs while it is present (reads warn): cleared only by a human via `seed state resume` (operator credential)**. Reviewer-lockout, `rejected_authors`, and the run-log are exactly as trustworthy as this ref: stated plainly rather than implied otherwise; the *load-bearing* gates (plan approval, receipt verification, merge) deliberately ground on the default branch and server-attributed identities instead (D3, D4.5).
- **CI visibility:** pushes to `seed-state` trigger no workflows (its tree has no workflow files); all state validation (card lint, run-log commit-over-commit inclusion, transition-table conformance) runs in the check+validate and maintenance workflows, which **explicitly fetch the ref**; state-validation latency = their cadence.
- **Growth:** every verb is a commit; history grows unboundedly. Discipline: shims use shallow/filtered fetches; when truncation is warranted it is a *human maintenance operation* (documented: temporarily lift protection, checkpoint, re-root, bump `.seed/version` so stale shims refuse), never automatic, because it conflicts with the no-force-push rule.
- **Forks/mirrors:** forks copy the default branch only; a fork wanting live coordination runs `seed init` for its own state ref. The state ref is single-remote by design in v1 (multi-remote sync is what beads' Dolt layer is for: that's the upgrade path).

**7.3 v1/v2 scope cut (decided in review, 2026-08-22).** **v1**: the port + `filecards` backend + `seed-state` ref (init, claim/lease/fence protocol, handoff stubs); task cards + transition table; plan grammar + hash pinning; receipts + `seed receipt verify` (CI author-of-record); `.seed/guardrails.yaml` + CODEOWNERS + validators; `.seed/hooks/` + `.worktreeinclude` + three tool shims; role definitions + fan-out sync; `loop.sh` (with lease renewal); memory/decisions conventions; CI = check+validate and seed-maintenance live (deterministic, no model secrets), seed-dispatch and pr-review in-tree but inert (v1 review is human-only until activation); one `core` squad. **v2**: `beads` and `github-issues` backends + the mirror exporter + state-label taxonomy; the `fastcards` builtin backend (amended 2026-08-22: a SQLite store inside the engine: same verbs, same card format, transactional claims, per the §7.1 builtin-store amendment: as the *single-machine* throughput rung; state is machine-local and does not travel with clones or CI, the exact variance its manifest declares, **which also moves the close lane local**: the CI merged-PR auto-close and no-PR dispatch cannot see a local DB, so on this rung the solo human operator closes review cards through the port on their machine: `seed task close` with their operator identity, evidence recorded on the card; the done-consistency lint's server-attribution clause applies only to state CI can fetch, and the fastcards README must say all of this); the workflow engine + mock harness; skills manifest/lockfile + compose; mailboxes + handoff-packet automation; multi-squad routing activation; remaining lifecycle shims; `paperclip` adapter; MCP transport.

**7.4 Teams are the organizing unit (direction, 2026-08-22).** Work is organized and executed in squads per §6; the one-member `core` squad is the degenerate default so the layer costs nothing for solo use.

**7.5 The seed engine ships as a pinned Go binary; the contract stays files (decided 2026-08-22).** The protocol-critical core: port shim, claim/lease/fence protocol, anchor-ancestry verification, schema validation, `receipt verify`, fan-out sync hashing, validators, `init`, is one static, cross-compiled **Go binary** (the *engine*). The "files, not an app" bet (§1) is untouched: the normative contract (port schemas, transition table, guardrails, card/plan/receipt formats) remains checked-in files, and the engine is a replaceable implementation of that spec. Shape:

- **What stays scripts:** the user-tunable layer: `loop.sh`, `.seed/hooks/*`, backend plugins, where hackability beats robustness. What must be binary-grade is exactly the code where a quoting bug is a data-corruption bug and where local/CI results must be bit-identical (`receipt verify`: R11).
- **Never commit the binary.** `scripts/seed` becomes a ~50-line **bootstrap shim** (POSIX sh + a `.ps1` twin): read the pinned engine version + SHA-256 from the lockfile, download the release asset to a cache *outside the repo*, verify the hash, exec. Same trust model as backend plugins (pin, verify, review upgrades). Air-gapped escape hatch: a config key pointing at a vendored binary path. CI installs the identical pinned version, so shim and CI cannot skew; `.seed/version` + exit 10 already covers protocol-version mismatch.
- **Two repos.** The engine lives in its own repo publishing releases; the template repo pins a version + hash in its lockfile. Building the engine inside each instantiated template makes no sense; this also gives R8's `seed upgrade` a concrete artifact to move the pin against.
- **Distribution (GitHub-native).** No Go registry exists in GitHub Packages and none is needed: (1) source via Go's own module path: semver tags fetched through `proxy.golang.org`/`sum.golang.org`; (2) **GitHub Releases as the canonical binary channel**: goreleaser matrix build + `checksums.txt` + GitHub artifact attestations (build provenance the shim can verify in a stricter mode); (3) optionally a GHCR image for container-first CI, never the primary channel (it would put a container runtime in the desktop path R2 exists to avoid).
- **Why Go (over Rust and compiled-TS runtimes):** the engine deliberately drives the **system git** for all `seed-state` remote operations: reusing the user's existing auth (credential helpers, SSH agents) instead of reimplementing it: which removes embedded-git (Rust/gitoxide's strongest card) from the decision. What remains is Go's home turf: trivial CGO-free `GOOS/GOARCH` cross-compilation (kills R2), unanimous ecosystem precedent (beads/`bd`, gh-aw), the module checksum-DB supply chain, millisecond cold start (each verb is one short-lived process; git/network dominate), fast builds, and high contributor/agent accessibility. Rust's stricter typing of the state machine is real but mostly theoretical here; the mitigation is structural in either language: **the transition table and verb classes are data, generated from `.seed/port-schema/`, with an exhaustive conformance test**: the spec file, not hand-written code, is the authority.
- **Degradation guarantee (differentiator #4 holds):** a clone with no engine installed is still a working repo: cards are readable files, PRs still gate on CODEOWNERS + server-side checks, a human can work bare-handed. The binary adds the protocol conveniences; it is not a hard dependency for participation.
- **Accepted costs:** a release pipeline to maintain; one network fetch before first use (vendor path mitigates); a new "shim can't fetch the engine" failure mode `seed init` must report clearly; slightly higher friction for engine contributions than editing a script in place.

**7.6 Employment is pull-based: supervisors, not a scheduler (proposed 2026-08-29).**
The question this decides: *who starts agents* when the goal is around-the-clock operation
across many projects and devices. Control planes in the surveyed field (Paperclip's
heartbeat engine; research/10, research/11 R1) answer it with **push-based central
employment**: a server owns the roster and fires heartbeats at adapters — which is exactly
the always-on, database-authoritative infrastructure §7.2 exists to avoid. Open-seed
adopts the inverse, and it is the only shape consistent with the substrate: **pull-based
distributed employment**. An *employer* is any number of independent **supervisor loops**,
each bound to an actor identity, that: sync state → read the mailbox → claim one eligible
`ready` card (atomic, §7.1) → spawn the configured harness in a worktree → renew the lease
at half-cadence → on any exit, leave through a deliberate verb — with a handoff packet on
every *continuation* exit (release, park, reap, cancel: exactly the transition table's
`write_handoff` effects), while the success exit `in_progress → review` needs no packet
because its continuation record is the pushed task branch, PR, and receipt. v1's
`loop.sh` *is* this supervisor in single-card form and remains the hackable script layer
(§7.5); an engine-grade `seed run <role>` may later harden the loop (crash-safe
re-entry, concurrency cap, budget preflight once R2 cost capture exists) without changing
the model. Mechanics and consequences:

- **Enrollment is an identity, not a connection.** A device joins the fleet with an actor
  name, a git credential for the state remote, and whatever roster/guardrails bindings its
  role needs — nothing else. No inbound connectivity, no control-plane session, no
  heartbeat registration: any machine that can `git fetch` (laptop behind NAT, closet
  Mac mini, CI runner, cloud session, ephemeral VM) is a worker. The per-device credential
  is the security surface, and the Q5/§10.5 hardening (dedicated machine identities,
  ref-scoped push rules) is the mitigation — enrollment ergonomics must never bypass it.
- **Fleet correctness needs no new primitives — with the declared claim variance intact.**
  Two devices racing for one card, a device dying mid-task, a stale worker resuming: on
  backends declaring `atomic_claim: native` (filecards push-wins, fastcards transactions)
  these are already solved by atomic claims, fencing tokens, leases, reap, and packets
  (§7.1). On **emulated-claim backends** (Linear, Jira: last-write-wins with post-claim
  verification, per their declared variances) the §7.1 variance stands, not this
  paragraph: fencing still prevents duplicate *state* — the loser's next fenced verb
  exits 6 — but it cannot prevent duplicate *effort* in the window between verification
  and that verb; a multi-supervisor fleet on such a backend must re-verify ownership
  before expensive phases and accept (or avoid, by choosing an atomic rung) the wasted
  work. A central scheduler would re-solve all of this worse, in a second store.
- **Wake is an accelerator, never a correctness dependency.** Polling at the supervisor's
  cadence is the floor; wake *adapters* — a forge webhook, an external scheduler tick, a
  tmux nudge, a hosted session's event trigger — only shrink latency. The system must be
  fully correct with polling alone (offline-native, §7.2), so wake adapters can be added
  per substrate without design changes.
- **Execution substrates are adapters below the supervisor.** Where the harness process
  runs — local worktree, cloud agent session, Orb-class ephemeral VM, an enrolled remote
  box — is invisible to the port. The portable contract is the supervisor loop plus the
  disposable-compute rule: a worker's machine may be destroyed after any card with nothing
  lost, because packet, receipts, and the state ref carry everything. **Disposability
  begins only once every durable artifact has confirmably left the machine**: the
  supervisor must treat a failed task-branch push as blocking — retry, then park the card
  with the failure in the packet, never advance to `review` — because a review card
  pointing at commits that exist only on a dead worker is exactly the loss this rule
  forbids. (v1's `loop.sh` currently warns and proceeds on push failure; adopting this
  decision makes that a defect to card and fix.)
- **Consequence for the field:** external control planes (Paperclip and kin) are
  *integration targets* (a backend selection per §7.1, or a projection per §7.7), never
  required infrastructure; any visibility plane open-seed adopts observes and issues port
  verbs but never employs. Accepted costs: pull latency as the wake floor; N devices mean
  N credentials to manage; per-actor budget enforcement stays advisory until cost capture
  lands.

**7.7 One authority, one-way projections: the mirror rule generalized (proposed
2026-08-29).** §D1 already settles this for the GitHub Issues mirror ("cards are
authoritative and the export direction always wins; human label edits are read back only
as requests"). This decision promotes that rule from a property of one component to the
**normative law for every pairing of stores**: per §7.1 backend selection, **exactly one
store is authoritative per repo** — filecards' truth is the `seed-state` ref; fastcards is
a machine-local authority by declaration; a selected external backend (Jira, Linear,
Paperclip, …) is itself the authority, and the seed-state ref is then simply not in play.
Where a second system must *see* the cards, the authority **projects outward one-way**
(export always wins, deletions included), and edits made on the projection side re-enter
only as **requests**: the dispatcher translates them into port verbs, which may refuse.
**Bidirectional synchronization between stores is forbidden**, not discouraged: two
authorities cannot both honor an atomic claim (each side's claimant "wins" locally),
replay lint becomes unverifiable against a history partly written elsewhere, and every
conflict-resolution policy — last-write-wins, CRDT merge — resolves by silently discarding
someone's claim or transition, which are precisely the failure classes the §7.1 fence
exists to prevent. One sanctioned reflection exists: the **reverse mirror** — when an
external backend is authoritative, a strictly read-only projection of its cards *into* the
seed-state ref may restore git-native visibility (dashboards, lint cross-checks, receipt
references); it is a projection with the direction bit flipped, carries a machine-written
"projection, not authority" marker at the ref root, accepts no writes, and is v2+ work
gated on a card. Validators, AC criteria, and adopter docs must describe backends in these
terms: substrate-portable guarantees (port semantics, atomic claims, lossless export on
demand) versus authority-specific guarantees (git history, anchors, replay lint — the
filecards/state-ref rung only).

## 8. Risks, gotchas, and mitigations

- **R1: Fan-out drift.** Source tree vs per-harness copies diverge silently. Mitigation: fan-out generated only by `seed sync`; `seed sync --check` (offline, hash-based) in CI; "do not edit here" markers.
- **R2: Windows portability.** Executable bits and symlinks are unreliable. Mitigation: the engine is a static cross-compiled binary with a `.ps1` bootstrap twin (§7.5); copy-based fan-out; `seed hooks run <name>` fallback runner; `core.fileMode` documented; no case-colliding paths.
- **R3: Untrusted content is everywhere.** Card bodies, issue text, PR comments, mail, and backend output are injected into agent context; cards are also *work orders*. Mitigation: shim schema-validates/sanitizes backend output; AGENTS.md treats task/mail content as data; the plan gate is the human/reviewer checkpoint before an unreviewed card becomes action above L1 (D1); CI trigger gating per claude-code-action, gh-aw as upgrade; backend plugins SHA-pinned; control surface excluded from auto-merge with server-side review (D4.1/D4.3).
- **R4: One ref serializes all writes.** Beyond claim contention, *every* mutating verb (transitions, comments, event appends) contends on `seed-state`: the real ceiling is writes/minute on the ref, and retry storms start well below 10 agents if they are chatty. Mitigation: one commit per verb; jittered backoff; agents batch comments; the maintenance workflow reports contention. The documented upgrades (amended 2026-08-22): **fastcards** for a *single machine* hammering the loop (builtin SQLite store, transactional claims, no network: trading state portability for speed) and **beads** for *multi-writer* replication across machines. Never pretend the file backend is atomic or high-throughput.
- **R5: Committed coordination state churns history.** Mitigation: machine writes on `seed-state` keep the default branch human-meaningful; one file per task and per mail message (never rewritten files under `merge=union`; union only on `decisions/**`); hash IDs; runtime scratch in `.git/info/exclude`; state-ref growth handled per §7.2.
- **R6: Budgets are advisory without a server.** Only run-local caps are enforceable in-repo. Mitigation: say so in guardrails docs; enforce what's local (loop budgets, attempt caps, circuit breaker); post-hoc accounting in the run log; hard org-wide stops via gh-aw credits or a control plane (Paperclip) through the backend/event seams.
- **R7: Harness churn.** Mitigation: the harness-neutral core (AGENTS.md, skills, port CLI, hooks) carries the semantics; harness files are generated shims; CI never depends on experimental harness features.
- **R8: Template drift after clone.** Mitigation: small versioned update surface (`.seed/version`, lockfiles); `seed upgrade` guidance against tagged releases; evolving parts (skills and roles: see the Q4 amendment on workflows) also distributed as a plugin/marketplace (Q4). Residual limits, documented in the handbook rather than papered over: a git-based marketplace source pins by `ref`, not by commit SHA, so the plugin channel is tag-trust, weaker than `seed.lock`'s commit+content pinning; and the channel's ref advances by hand (`seed plugin enable --ref`) until os-6eb32b94 adds a resolve-and-re-pin command plus a release floor. The cross-channel check reports `aligned`/`ahead`/`floating` and fails only a stale `behind` pin: a gate inside `make check` must not forbid the capability-only update the channel exists to allow.
- **R9: Convention without enforcement rots.** Mitigation: `validate.sh` + CI check every orchestration artifact (cards, plans + pinning, receipts, guardrails incl. the intersection rule, team files, lockfiles, run-log inclusion, fan-out drift); "shipped convention" and "shipped validator" are one deliverable: which is why the v1 cut (§7.3) is small.
- **R10: State-ref integrity is push-access-deep.** Anyone who can push `seed-state` can bypass the shim: clear a rejection list, forge lease fields, rewrite the log, or **forge an operator verb** (a spoofed `done` cannot reach a merged PR: merging grounds on required checks and server-attributed review, and no merge input reads card state, but it would fire the blocker-cascade and set a fleet building on unmerged work). Mitigation: no-force-push/no-delete protection on the ref; shim halts on observed non-FF rewrite; anchor-tag ancestry checks (§7.2); the maintenance done-consistency lint (D7) catches forged accepts and HALTs; the cascade's v1 trigger is the maintenance close step, which fires only on a verified merged PR; scope which principals hold state-ref push access (Q5); and honesty: audit claims about the run-log are conditional on this, everywhere they're made.
- **R11: The gate runs on the implementer's credentials.** Local receipts and local pre-merge results are forgeable by the agent that produced the work. Mitigation: above L1, the CI verify check is the author of record for receipts and required checks are configured server-side (D4.3/D4.5); the local gate is a fast pre-check, not the authority.
- **R12: Default-branch merge throughput is its own ceiling.** N parallel task PRs against one protected branch contend at landing: every merge advances the base, and required checks (including verify's command execution) re-run on update. Mitigation: no repo-wide "require branches up to date" (stale-plan safety is per-PR in verify: D3); document GitHub merge queue as the scale option (batched rebasing behind the same checks: the platform version of gastown's Refinery); keep `make check` fast because it is the term that multiplies.

## 9. Glossary (normative)

Terms below have exactly one meaning in this design and its research corpus. Where a quoted third-party project uses a term differently (notably sub-agents-skills, which calls harnesses "backends"), the quote keeps its original wording and this glossary governs open-seed's own usage.

| Term | Meaning, and only this |
|---|---|
| **Harness** | An agent CLI/runtime that executes model turns: claude-code, codex, gemini-cli, opencode, cursor-agent, … The role frontmatter field `run-agent:` names a harness. |
| **Backend** | A coordination store behind the port: an external plugin (`.seed/backends/<name>/bin/seed-backend`: beads, github-issues, paperclip) or an engine builtin (`entry = "builtin"`: filecards, fastcards). Never a harness. |
| **Port / shim / verb** | The port is the JSON-over-CLI task contract (nine required verbs); the shim is `scripts/seed`: the bootstrap that verifies and execs the engine (§7.5), whose port implementation is the only code that invokes a backend; a verb is one port operation (`claim`, `transition`, …). |
| **Engine** | The pinned Go binary implementing the spec'd protocol core (port, claim/lease/fence, validators, `receipt verify`, sync, init): §7.5. Distributed via GitHub Releases; never committed. Not a backend, not a harness. |
| **Role** | A definition file in `.seed/agents/*.md` (frontmatter: harness binding + permission tier; body: craft: `## Task`, `## Done When`). A chapter artifact. A role **variant** shares the body, differs only in frontmatter (§6). |
| **Agent (instance)** | A live harness session bound to a role. **Sub-agent**: a session spawned by another agent, inheriting the parent's claim and credentials. |
| **Member / squad / lead** | Member: an entry in a team file (human or a role ref). Squad: a team file in `.seed/teams/`. Lead: the squad's named human. |
| **Runner** | Avoided as a bare term. Qualified uses only: the **loop runner** (`scripts/loop.sh`), the **spawn runner** (the cross-harness sub-agent script, D8), an **external runner tool** (superset, dmux, …). |
| **Claim / lease / fence** | Claim: exclusive right to work a card (`ready → in_progress`). Lease: the claim's expiry, renewed while the agent lives. Fence: the claim token worker verbs must present (exit 6 when stale). |
| **Worker vs operator verb** | Worker: fenced verbs the claimant performs. Operator: credentialed verbs a lead or workflow identity performs (accept/reject/cancel/unblock/…). |
| **Gate** | Always qualified: the **plan gate** (plan-PR review before implementation), the **pre-merge gate** (`.seed/hooks/pre-merge.d/`, blocking, local), the **verify check** (`seed receipt verify`, required CI status), the **merge gate** (server-side branch protection + review). |
| **Control surface** | The path set in D4.1 that is never auto-mergeable and always CODEOWNERS-protected. **Work product**: paths with their own gates instead (plans, receipts, memory, decisions, code). |
| **State ref** | The `seed-state` branch holding machine-written coordination state (cards, run-log, handoff, mail). Push-access-deep trust (R10). |
| **Card / plan / receipt** | Card: a task record on the state ref (untrusted work order). Plan: the reviewed work authorization at `plans/<id>.md` (trust root = merge-base blob). Receipt: the evidence record at `receipts/<id>.json` (CI is author of record above L1). |
| **Tier (L1/L2/L3)** | Autonomy levels in guardrails.yaml: report-only / assisted-in-worktree / unattended-with-gates. Distinct from a role's **permission tier** (`read-only|safe-edit|yolo`), which is a harness sandbox setting. |
| **Mirror** | The one-way exporter rendering card state to GitHub issue labels (v2). Never a source of truth. |

## 10. Formerly open questions: resolved as provisional defaults (2026-08-22)

Each answer below is the adopted v1 default. "Provisional" means a reviewed edit to this section may still change one before GA; implementation proceeds on these as written, and diverging without editing this section is a design violation.

1. **Harness posture → Claude-Code-first with portable shims.** The richest primitive set (agent teams, hooks, skills, workflows) is the paved road. The harness-neutral core (AGENTS.md, Agent Skills, the port CLI, hooks) carries the semantics, so other harnesses stay first-class through generated shims and the `seed-harness` adapters (D8, [inspirations/09](./research/inspirations/09-harnesses.md)); CI never depends on experimental harness features (R7).
2. **Automation-on-clone → as proposed in §7.3, confirmed.** check+validate and seed-maintenance live on clone (deterministic, no model secrets); seed-dispatch and pr-review in-tree but inert until secrets are added; `claude-code-action` is the default activation path with gh-aw documented as the upgrade.
3. **Language/stack coupling → language-agnostic core.** The template's only contract with the project's stack is `make check` (the fast backpressure command) plus the hooks. Opinionated stack variants (e.g. a TypeScript flavor with lint/test/typecheck pre-wired into `make check`) are v2 template flavors, never the v1 core.
4. **Distribution → template repo in v1; plugin channel added in v2.** GitHub template instantiation is the v1 distribution. In v2 the evolving parts additionally ship as a Claude Code plugin/marketplace package to soften R8: plugin carrying capabilities, template carrying structure, with `.seed/version` + lockfiles as the update surface on both paths. **Amended (os-221f5929):** the channel carries **skills and roles**, not workflows. A Claude Code plugin's `workflows/` directory holds Claude-native dynamic-workflow scripts, a different artifact from the YAML DAGs `seed workflow run` executes (D8 already distinguishes the two), and resolving seed DAGs out of a plugin cache would couple the engine to a harness-owned path. Seed workflow DAGs are structure and stay on the template channel. Implementation: the plugin package and its marketplace manifest are **generated fan-outs** rendered by `seed sync` from the same sources as the other fan-outs, so `sync --check` polices both channels offline with one mechanism; the release coordinate for both is the `version` line in `.seed/template.lock`.
5. **State-ref principals → contributors by default, documented hardening, enumerated operators.** v1 default: any write-access principal may push `seed-state` (under the §7.2 protections: no force-push, no delete): stated honestly per R10 rather than implying more. Documented hardening for teams that need it: a push ruleset restricting `seed-state` to a dedicated machine identity (fine-grained PAT or deploy key) plus squad leads. **Operator-class credentials** (§7.1) are held by the squads' human leads and, once activated, the maintenance/pr-review workflows' app identities: enumerated in `.seed/config.toml` so the shim can enforce the verb class by identity; `seed init-github` prints the hardening option.
