# Research: Inspiration Catalog — Swamp, workflow-use, Incus, and the Agent-Platform Field

> Idea harvest researched 2026-08-29 for the open-seed backlog. Unlike the category surveys
> (01–10), this doc is an adoption list: every concept from the surveyed tools judged worth
> incorporating, each mapped to the open-seed mechanism it would extend, with priority and
> effort. Items already adopted by v1/v2 (goal ancestry, atomic checkout, worktree isolation,
> merge gates) are excluded.
>
> Sources: git.swamp-club.com/swamp-club/swamp (source + design/ docs, read from a local
> clone at version 20260206.200442.0), github.com/browser-use/workflow-use,
> github.com/lxc/incus, ampcode.com (what-are-orbs, orbs-explained, event-driven-orbs),
> github.com/MiaAI-Lab/sparkDash, github.com/paperclipai/paperclip (re-read Aug 29), plus
> the Paperclip/Factory/Augment findings in
> [10-org-control-planes.md](./10-org-control-planes.md) and the web sources cited there.
> Facts are from these reads (Aug 29, 2026) unless flagged uncertain.

Priority: **Now** (high value, fits current architecture) / **Next** (valuable, needs design) /
**Later** (good idea, not urgent). Effort: S / M / L.

## Part 1 — From Swamp (AGPL; take concepts, not code)

| # | Idea | What open-seed does with it | Priority | Effort |
|---|------|-----------------------------|----------|--------|
| S1 | **Audit surface: governed vs. direct timeline.** Swamp hooks every coding harness's post-tool-use event (five tools, one normalizer each), appends one JSONL row per event to date-partitioned files under `.swamp/audit/`, and renders a timeline separating tool-mediated from direct commands. Writer never throws — a broken hook must never hurt a session. | `seed audit`: harness hooks append to `audit/commands-YYYY-MM-DD.jsonl`; the renderer separates port verbs from raw git/file actions. Gives R1 ("use the port, not raw git") evidence and teeth. Feeds W1. | Now | S–M |
| S2 | **`doctor` as one front door.** Swamp ships per-integration preflight diagnostics (`doctor audit`, `doctor vaults`, `doctor secrets`). | `seed doctor`: one command running the whole preflight — state ref reachable, backend lock verified, sync clean, hooks installed, workflow registry sane, protections live (when a token is present). Wraps the scattered `validate`/`spec lint`/`state lint`/`backend verify`/`sync --check`. | Now | S |
| S3 | **Primitive-gated design docs + `last-verified` stamps.** A design doc may exist only if it *is* a primitive or names the primitive it enables; every doc carries `last-verified: <date> @ <commit>`; a PR changing a subsystem bumps its doc. | Adopt the rule for `docs/`; add a CI lint that flags behavior-changing PRs whose touched subsystem doc kept a stale stamp. Mechanical doc-rot resistance. | Now | S |
| S4 | **Dogfooded verification workflows, worktrees keyed by run ID.** Swamp verifies its own builds and agent reviews as swamp workflows, each in a fresh `git worktree` at the verified commit keyed by **run ID, not commit SHA**, so parallel runs never collide; cleanup fires pass or fail. | Express parts of `make check`/CI as `.seed/workflows/verify-*.yaml` run by the engine's own workflow runner. Steal the run-ID worktree keying for `.seed/hooks/` regardless. | Next | M |
| S5 | **Vault indirection.** Secrets resolved at run time, never frozen into YAML. | `{{vault.<name>}}` tokens in workflow defs, resolved at execution from env/keychain/extension; keeps credentials structurally out of checked-in workflows. | Next | M |
| S6 | **Data as a primitive.** Method runs produce versioned, immutable, *queryable* artifacts. | Upgrade receipts + workflow artifacts from "files that exist" to a queryable record (`seed data ls/get` over receipts/, workflow runs, evidence). | Later | M–L |
| S7 | **Activity-scoped convention docs.** Swamp's `agent-constraints/` holds per-dimension conventions (planning, implementation, verification, triage, adversarial review) loaded for the matching activity. | Split AGENTS.md's growing rule set into per-activity fragments the role files reference; sync fan-out already handles distribution. | Later | S |
| S8 | **Humans-are-seats accounting.** "Agents and CI never count" — the roster distinguishes human from machine actors. | `seed maintain report` breaks down activity by human vs. agent actors; useful for adoption metrics and for guardrails that only bind agents. | Later | S |

## Part 2 — From workflow-use (browser-use)

| # | Idea | What open-seed does with it | Priority | Effort |
|---|------|-----------------------------|----------|--------|
| W1 | **Record → parameterize → deterministic replay.** An agent performs the task once; the recording is distilled (noise filtered, variables extracted) into a deterministic workflow; replays are cheap; the agent handles only exceptions. | `seed workflow distill`: draft a `run:`-step workflow from a successful session's audit trail (S1 is the recorder). The second execution of any recurring chore becomes deterministic. | Next | M |
| W2 | **Self-healing steps with agent fallback.** A failing deterministic step falls back to the AI agent, which repairs the run and can propose a workflow update. | `on_fail: agent` in the workflow engine: bounded repair role finishes the step, then emits a *proposed patch to the workflow file* as a PR — self-healing inside the plan/PR gates, never silent self-modification. | Next | M |
| W3 | **Generation mode.** Workflows drafted from a natural-language description, no recording needed. | `seed workflow draft "<description>"`: harness-backed authoring that must pass `workflow validate` + a `--mock` run before the file is written. | Later | S–M |

## Part 3 — From Incus

| # | Idea | What open-seed does with it | Priority | Effort |
|---|------|-----------------------------|----------|--------|
| I1 | **Profiles: ordered, composable config inheritance.** An instance = `default` + `gpu` + `dev`, applied in order. | Compose role/agent/team definitions from profile fragments (base-worker + reviewer + squad overlay) instead of copy-paste flat files; `validate` resolves and checks the composition. | Next | M |
| I2 | **Remotes: one CLI, many servers.** `incus launch prod:img` — uniform verbs over named endpoints. | `seed --remote <name> task ready`: named state stores (several repos/queues) over the existing backend abstraction; the multi-repo org story. | Next | M |
| I3 | **Preseed + desired-state reconcile.** `incus admin init --preseed < yaml` bootstraps a whole server declaratively. | `seed init --preseed seed-init.yaml`; and turn `init-github` into a reconciler (desired protections/rulesets in a checked-in file → diff → apply → verify) instead of a one-shot script. The os-18135882 work becomes re-runnable. | Now | M |
| I4 | **Snapshot schedule + expiry.** Snapshots carry creation schedules and retention (`snapshots.expiry`). | Anchor retention policy in `guardrails.yaml` (keep N, expire after D — maintenance prunes) plus a first-class `seed state restore <anchor>` verb; completes the disaster-recovery loop. | Now | S–M |
| I5 | **Projects (multi-tenancy).** Instances/images/profiles isolated per project on shared infra. | Project-scoped queue views layered on squads (`seed task ready --project X`); mostly a listing/filter feature until a real multi-tenant deployment demands isolation. | Later | S |
| I6 | **Fork-friendly governance.** Apache-2.0, no CLA, community-run — the fork survived vendor capture. | Affirmation, not adoption: state the no-CLA, MIT, external-PRs-welcome commitment explicitly in CONTRIBUTING (the deliberate opposite of Swamp's employees-only policy). | Now | S |

## Part 4 — From Paperclip / Factory Droid / Augment Code (not yet adopted)

| # | Idea | What open-seed does with it | Priority | Effort |
|---|------|-----------------------------|----------|--------|
| P1 | **Wakeups on blocker resolution.** Resolving a blocker fires `issue_blockers_resolved`; the assignee's agent wakes. | Wire `plan-unblock` (and dependency-cascade unblocks) to auto-send mail + nudge to the card's author/last claimant — today an unblocked card waits for a poll. | Now | S |
| P2 | **Subtree pause-holds.** A hold on a parent suppresses wakeups down the whole subtree. | `seed task hold <id> --cascade` / `release-hold`: park an initiative without touching each child card. | Later | M |
| P3 | **Hard budget stops.** 80% soft alert, 100% hard stop with reset window and manual resume. | Opt-in `enforcement: hard` for budgets in `guardrails.yaml` on backends that can enforce (advisory stays the file-backend default). | Later | M |
| P4 | **Per-tool policy on the MCP surface.** Allow / deny / require-approval per tool, per agent, by risk class (read/write/destructive); approval-gated calls go to a human inbox. | Guardrails-driven verb policy for `seed mcp serve`: which actors may call which port verbs, with `require_approval` routing an operator-ack request through seed mail. | Next | M |
| F1 | **Initiative rollup.** Factory's pitch: plan a complex initiative, delegate to parallel agents, watch progress. | Cards already carry `parent`; add an initiative view (`seed task tree <id>` / report section) rolling up child states into progress. | Next | S |
| A1 | **Context quality beats agent cleverness.** Augment's context engine thesis. | Enrich the handoff packet — still mechanical-only: the plan's validation commands, last failing check output, diffstat vs. merge base. The packet is the seam that decides whether a cold resume succeeds. | Now | S |

## Part 5 — From Amp Orbs (ampcode.com)

Orbs are Amp's execution substrate: one thread = one ephemeral remote VM (repo, plugins,
tools pre-cloned), pause-at-idle at zero cost, minute billing, event-driven wakeups via a
per-orb external URL, agents spawning agents in fresh orbs. Not a competitor — a substrate
open-seed workers could run on. Two abstractions worth taking:

| # | Idea | What open-seed does with it | Priority | Effort |
|---|------|-----------------------------|----------|--------|
| O1 | **"Only diffs leave the orb."** The machine is cattle; git output is the record. Orbs prove disposable compute composes with a git-only source of truth. | An explicit AC criterion: a card must be completable by a worker on a machine destroyed afterward, with nothing lost — packet, receipts, and state ref carry everything. Exposes the tmux mail-nudge as the one substrate-coupled assumption. | Now | S (criterion) |
| O2 | **Executor adapters + wake-by-webhook.** Orbs wake on GitHub webhooks, CI failures, issue trackers, monitors. | A substrate-adapter seam: dispatch a claimed card to a local worktree, a cloud session, an Orb, an enrolled worker — with "nudge" generalized to "wake the executor" (webhook URL, session message, or tmux, per adapter). Generalizes P1 beyond local tmux. | Next | M–L |

## Part 6 — Recommendation: visibility plane — build, don't adopt

> This file is research evidence, not authority. Adopting this recommendation is a design
> decision: it becomes binding only via a PR to `docs/design-options.md` (authority order,
> CONTRIBUTING-AGENTS). Until then, nothing here puts a dashboard in scope.

**Question.** Use Paperclip as the visibility/control plane, or build one? (Prompted by
sparkDash, a fleet dashboard for DGX Spark hosts: central server + WebSocket snapshots,
"one model, N instances" hot config, per-unit tab pages, sparklines, one-click batch ops.)

**Recommendation: build.** Paperclip is not a visibility layer you attach; it is a control plane
you move into — its truth lives in its Postgres, and its repo-side surface is a `cwd` plus
REST (doc 10). Adopting it for visibility means dual sources of truth and a permanent sync,
violating the first pillar (no out-of-band database is ever authoritative). Open-seed
already emits everything a dashboard needs as JSON (cards, claims/leases, mail, report,
run-log events, receipts, workflow runs, anchors); a dashboard is a **projection**, and its
control actions are existing port verbs invoked as an authenticated port client — same
guardrails, roster, and audit.

| # | Idea | Shape | Priority | Effort |
|---|------|-------|----------|--------|
| D1 | **Projection-only dashboard, tiered.** Tier 0: `seed report --html` — static dashboard artifact per maintenance tick, zero server. Tier 1: live read-mostly server (state-ref watch → WebSocket snapshots → React) with verb buttons, shipped as an optional component like the MCP server. Tier 2: multi-repo via remotes (I2 — the dashboard is what makes remotes worth building), live event feed, packet viewer. sparkDash donates the shape: hot config for adding repos, per-card tab pages (card/plan/receipt/PR/events), sparklines (queue depth, review latency, lease countdowns, budget burn), batch ops (reap all expired, nudge all idle). **Hard rule from day one: the dashboard holds zero state beyond view preferences** — the moment it caches writes or grows annotations, it has rebuilt Paperclip's Postgres by accident. | Build | Now (T0) / Next (T1) | S / M |

## Part 7 — Gap analysis: what Swamp and Paperclip offer that open-seed (+ dashboard) still lacks

Gaps already on this catalog: S1 audit, S2 doctor, S5 vault, S6 data layer, P3 hard
budgets, P4 tool policy, O2 executor adapters, D1 dashboard. Beyond those:

**Structural gaps — Paperclip:**

| # | Gap | Notes | Disposition |
|---|-----|-------|-------------|
| R1 | **The employer runtime.** Paperclip *starts* agents: heartbeat scheduler with intervals/event wakeups, concurrency caps, budget preflight, workspace resolution, secret injection, adapter invocation (DB-backed wakeup queue with coalescing). Open-seed coordinates work but assumes agents show up — humans, cron, CI lanes play employer today. | The one gap that changes what open-seed *is*. Either a native supervisor loop (`seed run <role>`) or an explicit decision to delegate employment to executor adapters (O2) + schedulers/Routines/CI. Needs a decision record either way. | Decide, then Next |
| R2 | **Cost/token capture.** Paperclip records token usage and cost per run, attributed to issues/agents. Open-seed budgets are denominated in nothing — the engine never sees model spend; P3 hard stops are meaningless without this feed. | Harness hooks (the S1 audit pipeline) are the natural capture point. | Next |
| R3 | **Agent evals / qualification.** Paperclip's skill_test work mode runs an agent against test issues before trusting it. Open-seed has no notion of qualifying an agent for a tier or role. | Pairs with autonomy tiers: qualification as the gate to higher tiers. | Later |
| — | Delegation economy (agents formally requesting hires under a budget gate); per-issue watchdogs; work-product docs with edit locks; attachments; multi-org hosting; self-healing run retries. | Smaller; revisit after R1. | Later |

**Structural gaps — Swamp:**

| # | Gap | Notes | Disposition |
|---|-----|-------|-------------|
| — | **Models + drivers for external systems.** Swamp's core: typed models of infrastructure (EC2, DNS, servers) whose method runs produce versioned queryable data — deterministic *ops* automation. A whole adjacent domain. | **Recommended non-goal** (see below; binding only via a design-doc decision). S6 covers only the receipts/artifacts sliver. | Non-goal (proposed) |
| — | **Enrolled remote workers** (`swamp worker`, enrollment tokens) — dispatch-to-registered-device, which Paperclip also lacks. | Subsumed by O2 executor adapters. | Next (via O2) |
| — | **Daemon mode** (`swamp serve`, WebSocket API). Open-seed is invocation-only (CLI + stdio MCP). | The Tier-1 dashboard (D1) is the first daemon; generalize only if demand appears. | Later |
| — | **Real expression language** (CEL) vs. the deliberately minimal `when:` grammar. | Adopt only when workflow conditions demonstrably outgrow the grammar. | Later |
| — | **General extension ecosystem** (packaged drivers/vaults/datastores with publishing). | Skills + backend plugins cover current need; a registry is a hosted service — concept ports, service doesn't. | Later |

**Recommended non-goals (the inverse of the vision — do not "fix"; binding only once
recorded in the design doc):** Paperclip's Postgres-as-truth, web-first control, and
org-chart-as-database; Swamp's telemetry, hosted auth/registry/Lab, employees-only
contributions, and ops-domain modeling. Losing these is the point of open-seed.

**Backends: one authority + projections, never bidirectional sync (recommendation —
proposed to the design authority as §7.7; binding only if that design-doc PR merges).**
A recurring question this survey sharpens: should the card backends be *reflections* of
`seed-state`, kept in sync both ways? The recommendation is no. The binding design decides
this today only for the GitHub Issues mirror (D1: cards authoritative, export wins,
inbound label edits are requests); everything beyond that sentence is this file's
proposal, not settled ground. The proposed generalization: per backend selection, exactly
one store is authoritative — filecards' truth *is* the seed-state ref; fastcards is a
machine-local authority by declaration; an external backend (Jira, Linear, Paperclip) is
the authority when chosen, and seed-state is then simply not in play. Where two systems
must both see the cards, **the authority projects outward one-way (export always wins),
and inbound edits are read back only as *requests*** — commands routed through the
dispatcher into port verbs, which may refuse.
True bidirectional sync would make two stores authoritative at once: atomic claims cannot
span two masters, replay lint becomes unverifiable, and every conflict-resolution policy
(last-write-wins, CRDT merge) silently discards someone's claim or transition — exactly
the failure classes the port exists to prevent. The one useful variant is a **reverse
mirror**: when an external backend is authoritative, a strictly read-only projection of
its cards *into* the seed-state ref would restore git-native visibility (dashboards,
lint, receipts cross-checks) without a second writer. That is a projection with a
direction bit, not sync — sanctioned in the proposed §7.7 and worth a card only if
external-backend adoption materializes.

## Part 8 — Loose ends already identified elsewhere

- **Engine-side release resolution fallback** (from os-0f76f772's evidence): `seed template
  upgrade` follows the unauthenticated `/releases/latest` HTML redirect; add an
  `api.github.com` fallback with optional token for private forks. Engine change; file as an
  engine card. (Now / S)

## Suggested first wave

Three cards, in order: **S1 audit surface**, **S2 `seed doctor`**, **A1 handoff-packet
enrichment** — all small, all immediately visible. Second wave: **I3 protections reconciler**,
**I4 anchor retention + restore**, **P1 unblock wakeups**, **I6 governance statement**,
**D1 Tier-0 dashboard**. The differentiating bet after that is **W1+W2** (distill +
self-heal), which turns the audit log and workflow engine into a compounding loop: every
chore an agent does twice becomes a deterministic workflow that agents only maintain. The
open *decision* (not card) is **R1**: who plays employer — a native supervisor loop or
executor adapters plus external schedulers.
