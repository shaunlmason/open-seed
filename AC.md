# AC.md — Vision and Acceptance Criteria

> **Authority note.** This file is vision and acceptance criteria, not design authority:
> [`docs/design-options.md`](docs/design-options.md) remains binding, and where this file
> and the design doc disagree, the design doc wins. ◇ items are proposals — implementing
> one that changes settled design requires a `docs/design-options.md` PR first.

## 1. What open-seed is trying to accomplish

Open-seed exists so that **a software organization can run most of its engineering through
fleets of AI agents without giving up determinism, reviewability, or ownership of its own
coordination state.**

The bet has five parts:

1. **Git is the control plane.** Every piece of coordination state — task cards, plans,
   claims, evidence, receipts, decisions, memory — lives in the repository (or a git ref
   beside it), versioned, diffable, and reviewable with the same tools as the code. There is
   no proprietary server whose database becomes the real source of truth. Adopting open-seed
   is cloning a template; leaving it is deleting a directory.

2. **Agents are interchangeable workers; the process is the product.** Any harness (Claude
   Code, Codex, Cursor, OpenCode, whatever comes next) can pick up a card, because the
   contract is files and a small CLI, not a vendor SDK. The value accrues in the checked-in
   process — plans, guardrails, workflows, learnings — which outlives any individual model
   or tool.

3. **Autonomy is earned through gates, not granted by vibes.** Work above a trivial tier
   requires an approved plan; done requires evidence and an independent review; dangerous
   actions require operator identity; budgets and protected paths bound the blast radius.
   Humans move from routing work to holding the few gates that matter.

4. **Everything an agent does is observable and attributable.** Receipts bind PRs to plans,
   the run log binds state changes to actors and verbs, anchors make history tamper-evident,
   and audits distinguish governed actions from side channels. Trust comes from being able
   to check, not from being asked to.

5. **The system compounds.** Learnings and dead-ends persist across sessions; handoff
   packets make agents resumable; recurring work distills from one-off agent sessions into
   deterministic workflows that agents merely maintain. The org gets faster because the
   repo gets smarter, not because the model got bigger.

The end state: a repository where a human can say "make it so," watch a dispatcher route
cards, planners open plan PRs, implementers ship reviewed diffs with receipts, a reviewer
lane hold the line, and maintenance keep the queue honest — and at any moment `git log`
explains exactly what happened, why, and on whose authority. All of it open source, MIT,
forkable, with no phone-home and no seat to rent.

## 2. Acceptance criteria — the complete checklist

A criterion is met only when it is implemented, tested, documented, and enforceable (by
lint, CI, or protocol — not by convention alone). Items marked ◇ are aspirational beyond
the current build; unmarked items exist at least partially today and must be kept true.

### A. Coordination substrate (state, truth, safety)

- [ ] Exactly one configured store is authoritative per repo. On the default filecards
      backend all coordination state lives in git (the `seed-state` ref); fastcards is
      machine-local by declaration; an external backend (Jira, Linear, Paperclip, …) is
      itself the authority when selected. Portability is machine-readable
      (`state_portability`), and a lossless export can be produced on demand — a
      migration path out of any store always exists (export/import is a migration
      capability, not a live replica).
- [ ] ◇ Loss protection matches the declared portability: on `machine`- and
      `server`-portability backends, a scheduled `state export` snapshot (retained like
      anchors) bounds what a lost laptop or a vanished external service can take with it.
- [ ] Every state mutation goes through the port: one verb, one atomic transaction (a
      single commit on git-backed stores), one appended run-log event with actor, verb,
      timestamp, and task id.
- [ ] The port is specified as data (states, transitions, verb classes, effects,
      preconditions) and the spec validates itself; an inconsistent spec refuses to load.
- [ ] Protocol version is pinned in the repo and enforced with a distinct exit code on
      mismatch; engine and template negotiate versions explicitly.
- [ ] Claims are synchronous and exclusive: atomic checkout, fencing tokens on every
      subsequent verb, contention reported with a structured envelope and distinct exit code.
- [ ] Leases expire; renewal is cheap; reap is safe, attributable, and always leaves a
      handoff stub.
- [ ] Concurrent writers converge: push races retry with rebuild; no lost updates; no
      partial multi-file commits (atomicity lint proves it on replay).
- [ ] A HALT marker stops all mutation until an operator resumes; halts are attributable.
- [ ] State lint verifies full-history conformance (legal transitions, run-log atomicity,
      done-consistency: evidence present, review attribution verifiable — no-PR closes
      resolving to the human-operator roster — plan resolvable or exempt).
- [ ] Anchors make history tamper-evident; sync verifies anchor ancestry and refuses
      forged or rewritten state.
- [ ] ◇ Anchor retention policy (keep N / expire after D) with maintenance pruning, and a
      first-class `state restore <anchor>` disaster-recovery verb, drilled in CI.
- [ ] State export/import round-trips losslessly between backends; import refuses
      non-empty stores; migration between any two backends is a documented two-command path.
- [ ] The store degrades gracefully offline: local backends work with zero network; git
      backends queue and reconcile.
- [ ] ◇ Disposable-compute completability: a card can be taken to done by a worker on a
      machine that is destroyed afterward, with nothing lost — packet, receipts, and state
      ref carry everything; no coordination feature assumes a persistent host.

### B. Work lifecycle (cards, plans, review, done)

- [ ] A card is the unit of work: id, title, state, priority, squad, author, labels,
      links, blocked_on, claim, review block, body-as-work-order.
- [ ] The state machine covers the full life — backlog, ready, in_progress, review, done,
      blocked, cancelled, exactly the design doc's state set (claim is the ready →
      in_progress transition, not a state) — with block/unblock, park, release, reject,
      cancel, reinstate, and nothing else.
- [ ] Above the trivial tier, claiming an unplanned card authorizes planning only; the
      plan is a file, merged via its own single-file PR before implementation starts.
- [ ] Plans are lintable: required sections, validation commands, hashable content.
- [ ] Task PRs and plan PRs are structurally disjoint (task PRs never touch `plans/**`);
      CI classifies branches and enforces purity.
- [ ] Done requires: evidence attached, reviewer distinct from implementer (D4.5 — the
      reviewer lane's server-attributed app identity qualifies), merged PR (or an
      explicit, server-attributed no-PR exemption, whose closing actor must resolve to
      the human-operator roster).
- [ ] Dependencies cascade: closing a card unblocks dependents; plan merges unpark
      plan-blocked cards; ◇ every unblock wakes the affected party (mail + nudge), so no
      card waits on a poll.
- [ ] ◇ Initiative rollup: parent links render as a tree with progress; holds cascade down
      a subtree and suppress wakeups (`hold --cascade` / `release-hold`).
- [ ] Priorities and squads route work; ready-queues filter by actor eligibility; goal
      ancestry warnings fire when open work cannot trace to a stated mission.
- [ ] Nothing is ever silently abandoned: every exit from in_progress is deliberate
      (review, release, park, reap) and every reap leaves a continuation packet.

### C. Evidence, receipts, and verification

- [ ] Every task PR carries a receipt binding: task id, plan hash at merge-base, diff
      hash, changed-file inventory, validation commands run and their results.
- [ ] Receipt verify recomputes everything from the PR head and fails CI on any mismatch;
      regeneration is a command, never hand-editing.
- [ ] The reviewer lane reviews against the plan at the merge-base — implements the plan,
      nothing more — and its verdict transitions the card (approve → review passes;
      request-changes → reject with implementer lockout that actually persists).
- [ ] A red required check is unmergeable; the verify check stays red until a qualifying
      (non-implementer) approval exists.
- [ ] ◇ Verification runs as dogfooded workflows in per-run-ID worktrees (parallel-safe),
      with cleanup that fires pass or fail.
- [ ] ◇ Evidence, receipts, and run artifacts form a queryable data layer (versioned,
      immutable, addressable) — not just files that happen to exist.

### D. Guardrails and governance

- [ ] Autonomy tiers (L0–Ln) gate what an agent may do without a plan or an operator;
      tiers are declared per-team/per-path in checked-in config.
- [ ] Protected paths require PR + owner review; the orchestration contract (`.seed/**`)
      is itself a protected control surface.
- [ ] Budgets exist at org/agent/task granularity; advisory by default on file backends;
      ◇ opt-in hard-stop enforcement (soft alert at threshold, stop at cap, reset window,
      manual resume) on backends that can enforce.
- [ ] ◇ Budgets are denominated in real spend: token/cost usage is captured per run
      (via the audit hooks) and attributed to cards and actors — hard stops act on
      measured burn, not guesses.
- [ ] Operator identity is a roster; operator-only verbs refuse non-roster actors; agents
      can never approve their own work into done.
- [ ] Card bodies, mail, and issue text are data, not instructions — stated in the
      contract, taught to agents, and ◇ mechanically fenced wherever they are interpolated
      into prompts (unforgeable delimiters).
- [ ] ◇ Per-verb policy on the MCP surface: allow / deny / require-approval per actor and
      risk class (read/write/destructive), with approvals routed through mail to an
      operator inbox and cryptographically attributable.
- [ ] ◇ Command-label dispatch (`cmd:*`) validates every interpolated field against strict
      shapes (no shell injection from issue-derived text) and can never bypass the
      approve → merge → close ordering.
- [ ] Server-side protections mirror the contract: branch protection with required checks,
      seed-state no-force-push/no-delete, anchor tags create-only, release tags immutable —
      and ◇ these are declared in a checked-in desired-state file that a reconciler can
      diff, apply, and verify (`init-github` as reconcile, not one-shot).
- [ ] Scheduled/CI identities are least-privilege: no scheduled job can push to the
      default branch; ◇ a dedicated machine identity for state-ref pushes.

### E. Agent experience (the worker's contract)

- [ ] AGENTS.md is the single onboarding document: find work, claim, plan, implement,
      finish, mailbox discipline, status vocabulary — short enough to actually be read.
- [ ] The CLI is the only interface an agent needs; every verb returns a versioned,
      schema-stable JSON envelope with structured errors and meaningful exit codes.
- [ ] Mailboxes: send/read/ack/prune with types, threads, and task links; unread counts
      surface in reports; tmux nudges reach live sessions and no-op gracefully without.
- [ ] Handoff packets: bounded (≤8KB), mechanical-first (card + git anchors, prior claim,
      blocked_on, evidence trail, dirty-file inventory), written on release/park/reap and
      on demand.
- [ ] ◇ Packet enrichment, still mechanical-only: plan validation commands, last failing
      check output, diffstat vs merge-base.
- [ ] Memory compounds: LEARNINGS.md and DEADENDS.md are append-paths in every task PR;
      ◇ retrieval guidance so agents actually consult them before repeating history.
- [ ] Skills are packaged, locked (manifest + lockfile), and installed reproducibly;
      frozen installs refuse drift.
- [ ] ◇ Activity-scoped convention docs (planning / implementation / verification /
      review / triage) that role definitions reference, instead of one ever-growing rule file.
- [ ] Role definitions exist for the whole loop — dispatcher, planner, implementer,
      reviewer — and ◇ compose from ordered profile fragments (base + role + squad overlay)
      resolved and checked by validate, instead of copy-paste flat files.

### F. Harness and forge agnosticism

- [ ] Fan-outs generate every harness's native config (`.claude/`, `.agents/`, AGENTS.md
      managed block) from single sources; `sync --check` fails CI on drift; fan-outs are
      never hand-edited (R1).
- [ ] MCP transport exposes the full port to any MCP-speaking tool with identical
      semantics to the CLI.
- [ ] Windows is first-class (`seed.ps1` parity documented and tested).
- [ ] Nothing in the worker contract assumes a specific model vendor; harness and model
      names are registry data, not code.
- [ ] GitHub is an integration, not a dependency: ◇ the CI lanes, dispatch, mirror, and
      protections reconciler have forge adapters (Gitea/Forgejo, GitLab) or documented
      equivalents, and the core loop runs on any git remote with zero forge features.
- [ ] ◇ Execution substrates are adapters: a claimed card can be dispatched to a local
      worktree, a cloud agent session, an ephemeral VM (Orb-style), or an enrolled remote
      worker — with "nudge/wake" generalized per adapter (tmux, session message, webhook
      URL) so unblock wakeups reach any executor.
- [ ] ◇ Who plays employer is decided and recorded: either a native supervisor loop that
      schedules/wakes the agent roster (concurrency caps, budget preflight), or an explicit
      delegation of employment to executor adapters plus external schedulers — never left
      implicit.

### G. Backends and portability

- [ ] The two builtin backends cover the spectrum: filecards (the default — cards ride
      the anchored `seed-state` git ref, offline-native) and fastcards (machine-local
      SQLite for single-machine throughput, same layout).
- [ ] External backends are plugins: manifest with declared capabilities, lockfile with
      source + content hash, hash verified before every invocation, minimal environment
      (PATH/HOME + declared requires_env), envelope-validated output, exit codes passed
      through; schema-invalid output is discarded with a distinct code.
- [ ] Capability negotiation is machine-readable (atomic_claim, offline, budget,
      state_portability) so the engine can warn about variance instead of guessing.
- [ ] Adapters exist (and stay conformant via a shared port-conformance suite) for the
      ecosystems that matter: beads, GitHub Issues, Jira, Linear, Paperclip — ◇ each
      revalidated against live instances on a schedule.
- [ ] ◇ Remotes: named state stores addressed uniformly (`seed --remote infra task ready`)
      so one operator spans many repos/queues with one CLI.
- [ ] ◇ Projects/tenancy: project-scoped queue views layered on squads.

### H. Workflows and deterministic automation

- [ ] The workflow engine runs DAGs of steps (AI, run:, gate-only, loop groups) with
      typed inputs, produces with JSON-schema enforcement, dependency waves, trigger
      rules, when-expressions, retries, on_fail policy, wall-clock budgets, and
      checkpoint/resume that refuses mixed-graph resumes.
- [ ] Gates cover the trust boundary: approval (pause → response file → resume), review
      (reviewer role, verdict loop with remediation and max-revisions), checks (CI green +
      zero unresolved threads via forge APIs).
- [ ] Mock mode is total: zero credentials, zero side effects, AI steps stubbed
      schema-validly, commands recorded not executed, gates auto-passed — so every
      workflow is testable in CI.
- [ ] Validation is exhaustive (schema, ids, DAG acyclicity, action XOR, artifact closure,
      role closure, registry closure, token lint, loop rules) and refuses to run an
      invalid workflow.
- [ ] ◇ `workflow distill`: draft a deterministic workflow from a recorded session's audit
      trail (noise filtered, variables extracted) — the second run of any chore is
      deterministic.
- [ ] ◇ `on_fail: agent` self-healing: a bounded repair role finishes the failed step and
      proposes the workflow patch as a PR — never silent self-modification.
- [ ] ◇ `workflow draft "<description>"`: NL-to-workflow authoring gated by validate + a
      green mock run.
- [ ] ◇ Vault indirection: `{{vault.*}}` resolved at run time from env/keychain/extension;
      secrets never frozen into YAML, never echoed into logs.

### I. Observability, audit, and operations

- [ ] The run log is a complete, append-only account of every state change; replay lint
      proves it.
- [ ] ◇ `seed audit`: harness hooks append every agent tool-use to date-partitioned JSONL
      (never-throw writer); the timeline separates port-mediated actions from direct
      git/file side channels; doctor verifies hook health.
- [ ] `maintain report` surfaces queue health: stalled reviews, long-parked plans, unread
      mail, expired leases, ancestry gaps; ◇ human-vs-agent actor breakdown.
- [ ] `maintain reap` is safe to run on a schedule; the maintenance lane (reap, lint,
      close-on-merge, mirror, prune, anchor) runs green unattended and is itself audited.
- [ ] ◇ `seed doctor`: one preflight command that checks everything (state ref, backend
      lock, sync drift, hooks, workflow registry, protections, version pins) and reports
      per-integration health with fix-it hints.
- [ ] Every refusal is a structured envelope a machine can branch on and a human can read.
- [ ] ◇ Metrics that matter are derivable from git alone: cycle time per card, review
      latency, rework rate, budget burn, packet-resume success.
- [ ] ◇ A visibility plane exists and is projection-only: dashboards (static report
      artifact and/or live server) render solely from port-queryable state, issue writes
      solely through port verbs under the same guardrails and audit, and hold zero state
      of their own beyond view preferences.
- [ ] ◇ Agents are qualified, not assumed: an eval/skill-test mode runs an agent against
      test cards and gates its admission to roles and higher autonomy tiers.

### J. Distribution, upgrades, and supply chain

- [ ] The template is a tagged, released artifact; adopting it is clone-and-init;
      `template upgrade` pulls new template versions via lock + three-way merge, with
      --check and explicit --to rollback.
- [ ] The engine is a pinned release: bootstrap shim downloads the exact version, verifies
      SHA-256 from the checked-in lock, caches outside the repo, and execs; the binary is
      never committed; vendor/SEED_ENGINE paths serve air-gapped use.
- [ ] `seed upgrade` moves engine pins against tagged releases with checksum verification,
      protocol preflight, semver-downgrade refusal, and atomic lock rewrite.
- [ ] Release resolution works for everyone: public HTML redirect path and ◇ authenticated
      API fallback for private forks.
- [ ] Everything hash-pinned: engine releases, backend plugins, skills; nothing executes
      from an unpinned source.
- [ ] ◇ `init --preseed`: declarative one-file bootstrap of a new adoption (config,
      guardrails, teams, protections desired-state), idempotent and CI-verifiable.
- [ ] Install experience is boring: one command, no telemetry, no account, no network
      beyond the pinned artifact fetch.

### K. Project quality, docs, and community

- [ ] `make check` is the single fast backpressure command and stays green on main.
- [ ] Engine unit coverage stays above 90% (union statement coverage, cmd included);
      template scripts and hooks carry smoke tests; CI runs the whole port-conformance
      suite per backend.
- [ ] Docs are governed: handbook for operators, AGENTS.md for workers, design docs gated
      by the mechanism they enable, ◇ every design doc stamped `last-verified: <date> @
      <commit>` with a CI lint that flags stale stamps on behavior-changing PRs.
- [ ] The research corpus (docs/research/) is maintained: adjacent tools surveyed, adopted
      ideas traced to their source, rejected ideas recorded with reasons.
- [ ] Decisions are recorded (decisions/) and binding; contributor instructions state the
      authority order.
- [ ] Governance is explicitly fork-friendly: MIT, no CLA, external PRs welcome, no
      trademark gymnastics, no open-core split — stated in CONTRIBUTING, not just implied.
- [ ] Dogfooding is total: open-seed's own development runs through open-seed — cards,
      plans, receipts, reviewer lane, maintenance — and every new feature ships only after
      it has coordinated its own delivery.

### L. The autonomy end-state (what "done" ultimately looks like)

- [ ] ◇ A human files an intent (issue, mail, or one sentence); the dispatcher lane turns
      it into routed, prioritized cards without human triage.
- [ ] ◇ Planner agents produce plan PRs that pass human review >80% unedited; implementer
      agents take approved plans to green, receipted PRs without intervention on the happy
      path.
- [ ] ◇ The reviewer lane holds quality alone on low tiers; humans review only high-tier
      plans, protected paths, and escalations.
- [ ] ◇ Escalation is structured: blocked(needs-you) reaches a human with the packet, the
      question, and the minimal decision needed — never a wall of transcript.
- [ ] ◇ The system runs for a week unattended on a real backlog with zero state
      corruption, zero silent abandonment, zero guardrail breaches — and the git history
      alone is sufficient to reconstruct and justify everything that happened.
- [ ] ◇ A team that has never spoken to the authors adopts open-seed from the README in
      under an hour, on their forge, with their harness, and reaches their first
      agent-shipped, human-reviewed PR the same day.
