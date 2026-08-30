# SEED-NEXT.md — The Next Substrate

> **Authority note.** This document is the target architecture for the next major version
> of open-seed, designed from first principles with everything the research program has
> taught us (docs/research/01–12, the binding §7.1–7.5 decisions, the proposed §7.6–7.7
> sections, and the adjacent-tool field). It is **vision and acceptance criteria, not design authority**:
> [`docs/design-options.md`](docs/design-options.md) remains binding for the current
> system, and any element here that contradicts settled design becomes real only through
> a `docs/design-options.md` PR. This document will eventually replace
> [`AC.md`](AC.md); until then AC.md remains the checklist for the current build, and
> criteria marked ● below exist in the current system and carry forward.

---

## Part I — Vision

### The problem, stated from zero

A software organization wants to run most of its engineering through fleets of AI agents.
The agents are capable but unreliable in specific, well-documented ways: they claim
completion without proof, they drift from instructions, they collide with each other, they
forget everything between sessions, and they can be manipulated by the content they read.
The organization cannot accept losing determinism (what happened must be knowable),
reviewability (a human must be able to check any of it), or ownership (the coordination
state must belong to the org, forkable and portable, hostage to no vendor's database).

Every system in this space chooses where truth lives and who is allowed to say "done."
Get those two choices wrong and no amount of agent capability saves you; get them right
and even mediocre agents compound. SEED-NEXT is the design that follows from getting them
right on purpose, from the first commit — rather than arriving at them by repair, as the
current system partially did.

### Ten first principles

Everything in Part II derives from these. Each one earned its place: it was either
independently converged upon by every serious source we studied, or learned the hard way
in our own design fights.

1. **History is the database.** The authoritative object is an append-only, hash-chained,
   signed log of events. Cards, queues, claims, messages, verdicts, budgets — all of it is
   *derived* from the log, never stored beside it. Tamper-evidence, atomicity, attribution,
   and audit are then properties of the structure, not features to build and police.

2. **One authority; everything else is a projection.** Exactly one log is authoritative
   per repository. Every other surface — a queue view, a dashboard, a Jira mirror, a local
   cache — is a one-way, rebuildable, disposable projection. Inbound edits from any
   projection are *requests*, mediated by a governed actor. Bidirectional synchronization
   is forbidden, permanently, because a second writable copy is a second authority in
   waiting.

3. **Verification is the product.** The bottleneck of every agent loop is the verifier,
   not the model. A unit of work is therefore born as a *contract* — intent plus
   machine-checkable acceptance — and "done" is a verdict rendered by an independent
   verifier against checks the implementer can neither read nor write. The model may
   propose "done"; it can never approve it.

4. **Identity is cryptographic.** Every actor — human, agent, service — is a keypair.
   Every event is signed. The roster is a keyring. "Reviewer distinct from implementer"
   is then signature verification, not protocol convention; a forged history is a broken
   chain, not a subtle audit finding.

5. **Compute is disposable; the record is durable.** Any executor — a laptop worktree, a
   container, a cloud session, an ephemeral VM — must be destroyable at any moment after
   its last confirmed push with zero loss. Nothing that matters ever exists only in a
   process, a context window, or a machine. The handoff packet, the ledger, and pushed
   branches carry everything.

6. **Affordances over refusals.** The system tells every actor what it may legally do
   next, computed from the spec, the state, and the actor's own standing — before the
   actor acts. Refusals still exist and are structured, but an actor that reads its
   envelope should almost never meet one. Invalid actions should be *unrepresentable*
   before they are *refused*.

7. **Learning is a governed loop, not a hope.** Preserving experience is not learning
   from it. Evidence is collected online, always; lessons are distilled offline, by
   comparison across trajectories, with provenance; and they are promoted — or retired —
   through the same gates as code. Recurring work distills into deterministic programs
   that agents merely maintain. This is the compounding mechanism: the repo gets smarter,
   independent of the models.

8. **Data is never instructions.** Content that flows through the system — card bodies,
   messages, mirrored issue text, tool output, fetched pages — is evidence to be judged,
   never instructions to be obeyed. This boundary is stated, taught, and mechanically
   fenced wherever content is interpolated into a prompt or a command.

9. **Nothing modifies its own gate.** An agent may propose changes to anything, including
   the system itself — but the validators, verifiers, thresholds, keyrings, and audit
   structures that approve its work are outside its write capability, always. Every
   self-improvement is a candidate that an independent gate admits or rejects.

10. **Humans hold gates, not queues.** The human role is to file intent, approve
    high-consequence plans, review the protected surface, and resolve escalations — each
    escalation arriving as a bounded packet with one decision attached, never a
    transcript. Everything between those gates runs without asking.

### The end state

A repository where a human states an intent in one sentence; a dispatcher turns it into
contracts with acceptance checks; a supervisor employs a roster of enrolled, qualified,
cost-metered agents across whatever compute is cheapest that hour; planners open
falsifiable plans; implementers ship diffs from disposable machines; verifiers render
signed verdicts against sealed checks; a curator distills the week's trajectories into
lessons and deterministic workflows; and at any moment, one command over the ledger
reconstructs exactly what happened, in what order, by whose key, at what cost, and on
whose authority. All of it open source, MIT, offline-capable, forkable in an afternoon,
with no server that knows anything the repository doesn't.

---

## Part II — Design

### 1. The Ledger

The ledger is the single authoritative object: an append-only sequence of events stored
on a dedicated git ref (`refs/seed/ledger`), replicated wherever the repository is
replicated, and usable offline like anything else in git.

**Event structure.** Every event carries:

- `seq` — monotonic sequence number.
- `ts` — timestamp (recorded, never trusted for ordering; `seq` orders).
- `actor` — the acting identity's key fingerprint.
- `verb` — what happened (`intent.filed`, `claim.taken`, `verdict.rendered`, …).
- `subject` — what it happened to (a contract id, an actor id, the system itself).
- `payload` — verb-specific structured data, schema-validated.
- `prev` — hash of the preceding event's canonical form.
- `sig` — the actor's signature over the canonical form (which includes `prev`).

The chain of `prev` hashes makes history tamper-evident end to end; the signatures make
every entry attributable and non-repudiable. There is no separate anchor mechanism, no
separate run log, no separate audit trail: the ledger *is* all three.

**Storage and append.** Events append as records in date/size-partitioned segments
committed to the ledger ref; one push appends one batch from one actor, in order.
Concurrency control is git's own: a losing push fetches, re-validates its events against
the new tip (a claim that lost its race fails re-validation and is not re-appended — the
actor sees the winner instead), recomputes `prev`, and re-pushes. This is the current
system's push-race discipline, promoted from implementation detail to the definition of
the store.

**Checkpoints.** Periodic `checkpoint` events embed the hash of the canonical projection
state at that `seq`, so a reader can verify a checkpoint against replayed history once and
then start from it. Checkpoints bound replay cost without ever truncating history; full
history is retained, and pruning is a policy decision recorded (like everything) as an
event.

**Genesis and halt.** A ledger begins with a signed `genesis` event naming the initial
operator keys and the protocol version. A `halt.declared` event stops all appends by any
writer except `halt.lifted` from an operator key. Halt is therefore not a marker file that
tooling must remember to check — it is part of the same history every writer must be on
the tip of in order to append at all.

**Why git.** Offline-native, replicated by the same channels as the code, forge-agnostic,
inspectable with universal tooling, and hosting-free. The ledger inherits the repo's
backup story, access control, and disaster recovery for free. A team that can host a git
remote can host the entire coordination system.

### 2. Projections

A projection is a deterministic function from a ledger prefix to a view. All projections
share five properties:

1. **Derived** — computed from events, never written directly.
2. **Stamped** — every projection carries the `seq` it was built at.
3. **Rebuildable** — deleting any projection loses nothing; rebuild is one command.
4. **Read-only outward** — external systems receive projections; they never write them.
5. **Non-authoritative** — no decision anywhere may treat a projection as truth when the
   ledger disagrees; staleness is visible (the stamp) and never silent.

**Standard projections.** The ready queue (filtered by actor eligibility), the contract
detail view, the actor view (what am I holding, what's addressed to me), the report
(queue health, stalled work, unread messages, budget burn), a local SQLite cache for
high-frequency reads on one machine, the dashboard, and external mirrors (GitHub Issues,
Jira, Linear, or anything else someone writes an exporter for).

**Inbound flow.** A projection surface may accept human input — someone edits a mirrored
issue, clicks a button on the dashboard — but that input enters the system exclusively as
a *request event* appended by a governed actor (the dispatcher, or the dashboard's own
enrolled service identity), validated against strict shapes, and honored or refused under
the same guardrails as any other actor's action. There is no path by which an external
system's database writes coordination state.

**What this deletes.** The entire authoritative-backend apparatus of the current system:
backend plugins as truth stores, capability negotiation (`atomic_claim`,
`state_portability`), the emulated-claim variance matrix, and the conformance burden of
keeping a second brain honest. We built that machinery, then argued through §7.7 (a
proposed design-doc section) that its
central feature — an external system as the authority — is one we must never use. In
SEED-NEXT it does not exist. Integration effort goes into exporters and request adapters,
which are simple, and stays out of consensus, which is not. The current `fastcards`
backend dissolves into "the SQLite projection": same speed, no second authority.

### 3. Identity and actors

**Enrollment.** An actor joins by an `actor.enrolled` event carrying its public key, its
kind (`human`, `agent`, `service`), and its display name. The roster is the keyring
derived from enrollment and revocation events. Enrollment of an agent requires two
things: an identity, and a git credential scoped to what that agent may push. That is the
entire "device registration" story — no daemon handshake, no heartbeat server.

**Capabilities.** Roles are capability grants recorded as events: this key may claim
contracts in these squads at these tiers; this key may render verdicts; this key is an
operator; this key may append curation candidates. Grants are checked at append time by
every writer and at read time by the affordance computation (§7). Operator-only verbs
refuse non-operator keys structurally.

**Qualification.** An agent's grants are earned, not assumed. A role grant above the
trivial tier requires passing *eval contracts* — synthetic work items with known verdicts,
run through the same machinery as real work (§14). Qualification results are events;
grants cite them; a degraded agent (failing spot-check evals) can have grants suspended by
the supervisor without ceremony.

**Humans are seats.** The roster distinguishes human keys from machine keys everywhere it
matters: guardrails that bind only agents, metrics that report human-vs-agent activity,
and the operator gates that only human keys can hold.

### 4. Work contracts

The unit of work is a **contract** — the successor to the card, renamed because its
defining feature is no longer its state but its acceptance.

**Birth.** A contract is created by `intent.filed` (from a human, a mirror request, or a
lane) and becomes claimable only when it carries:

- **Intent** — what and why, in prose (data, not instructions — see §11).
- **Acceptance spec** — the machine-checkable definition of done: commands that must
  succeed, state assertions, artifacts that must exist, review rubric items for the
  qualitative residue. A contract without an acceptance spec cannot leave `draft`.
- **Sealed checks** (above the trivial tier) — additional acceptance checks encrypted to
  the verifier keyring. The implementer can prove they existed before implementation (the
  sealed hash is in the ledger) but cannot read them. This is the structural answer to
  reward-seeking: you cannot aim at a target you cannot see, so you must aim at the
  intent.
- **Tier, budget, routing** — autonomy tier, spend budget, squad/priority.

**Lifecycle.** The familiar states — `backlog`, `ready`, `in_progress`, `review`, `done`,
`blocked`, `cancelled` — survive as the *projection vocabulary* of the event stream, with
the same discipline as today: claim is the `ready → in_progress` transition, not a state;
every exit from `in_progress` is deliberate (submit, release, park, reap); nothing is
silently abandoned. The transition rules are data (the spec validates itself; an
inconsistent spec refuses to load) and every writer enforces them at append time. ●

**Claims and liveness.** `claim.taken` carries a fencing sequence; every subsequent event
on that contract from that claimant must cite it, and a stale fence is refused. Liveness
is one mechanism, not two: the claimant appends periodic `heartbeat` events whose payload
*is* its progress note (items completed, current focus, blockers). Lease renewal and
progress reporting are the same event. A claimant whose heartbeats stop is expired; a
claimant whose heartbeats continue but whose progress payload has not changed in N
periods is *wedged* — a distinct, visible condition that reaping heuristics and the
supervisor act on. The current system's separate lease-renew and progress-file ideas
collapse into this.

**Plans as falsifiable change contracts.** Above the trivial tier, claiming an unplanned
contract authorizes planning only. ● A plan is a file, merged through its own single-file
PR, and it must be *falsifiable*: it names the boundary set (what is broken and will be
shown fixed), the retention set (what works and will be shown unharmed), the validation
commands for both, and the expected shape of the diff. A plan without a retention check
does not lint. Plan PRs and implementation PRs remain structurally disjoint. ●

**Handoff packets.** The packet is the *only* interface between executors, and it has
exactly three parts: the acceptance criteria (from the contract), the settled decisions
and constraints (so a successor never relitigates them), and artifact references by path
— branch, plan, receipts, dirty-file inventory — never contents, never trajectories.
Packets are bounded, mechanical, written on every deliberate exit and every reap, and
their sufficiency is tested (§14: packet-resume drills). ●

### 5. The verdict pipeline

**Done is a verdict.** When an implementer believes a contract is complete, it appends
`submission.made` citing the branch and the evidence it gathered. It cannot append
anything stronger. A **verifier** — an identity holding the verdict grant, provably
distinct from the implementer's key — then:

1. Checks out the submission in a clean, isolated workspace (per-run-id, parallel-safe).
2. Runs the visible acceptance spec.
3. Unseals and runs the sealed checks.
4. Recomputes the receipt: plan hash at merge-base, diff hash, changed-file inventory,
   command transcripts.
5. Renders `verdict.rendered` — pass or fail, with the receipt as payload, signed.

A passing verdict is what moves the contract to `done` (together with the merge, which is
gated on the verdict: a red required check is unmergeable ●). A failing verdict moves it
back with the failure evidence attached and the implementer locked out of self-approval
paths, as today. ●

**Independence is structural.** The verifier's verdict is grounded exclusively in what it
executed and read itself; it has no input channel for the implementer's claims. Verifier
keys cannot hold implementer grants on the same contract; the sealed-check keyring is
disjoint from every implementer capability; and the verifier's own code, rubrics, and
thresholds live on the protected surface (§11) that no agent — verifier included — can
modify for itself.

**Qualitative residue.** Acceptance that cannot be a command (tone, judgment, taste) is a
rubric the verifier scores item-by-item with cited evidence and explicit uncertainty —
never a single holistic score — and rubric calibration is itself maintained against a
human-scored gold set (§14).

### 6. The supervisor

The employer question is answered natively: SEED-NEXT ships a **supervisor** — a runtime
loop, not a service — that turns the roster and the queue into scheduled work.

**Pull, never push.** Following the proposed §7.6: workers pull work; nothing requires inbound
connectivity to any executor. The supervisor is itself just a privileged actor running a
loop: read the queue projection, decide assignments, wake executors, meter costs, reap
the dead. It can run as a daemon, a cron, a CI schedule, or a human typing commands — the
ledger cannot tell the difference, which is the point.

**Scheduling.** Assignment is a function of: contract priority and tier; the actor's
grants and qualification; the actor's and org's *remaining* budget (real, metered spend —
not step counts); concurrency caps per actor and per squad; and executor cost class
(mechanical work routes to cheap models/substrates, hard work to strong ones). Because
budgets are remaining-balance-visible, workers can be budget-aware — adapting strategy to
what's left rather than discovering exhaustion at the cap.

**Executor adapters.** An assignment dispatches through an adapter: local worktree,
container, cloud agent session, ephemeral VM, enrolled remote worker. Adapters implement
exactly three things: *provision* (give the executor a workspace and the packet), *wake*
(the channel — process spawn, session message, webhook, or nothing but the executor's own
poll; wake channels are accelerators, never correctness dependencies ●), and *meter*
(emit `run.metered` events — tokens, cost, wall-clock — attributed to contract, actor,
and model class). Disposal is the adapter's business and begins only after the last
confirmed push. ●

**Cost is a first-class stream.** Every run emits metering events; budget enforcement,
the report's burn view, and the scheduler's routing all read the same stream. Soft alert
at threshold, hard stop at cap where enforcement is possible, advisory where it is not —
but always from measured spend, never guesses.

**Preemption and holds.** A hold on a contract or a subtree is an event; the supervisor
suppresses wakeups beneath it. Urgent work can preempt: the supervisor messages the
executor, which exits deliberately at a safe point (a park with a packet), never by kill
if grace is available.

### 7. Affordances

Every response any actor receives from the system is an **affordance envelope**: the
result of the verb, plus the set of verbs currently legal *for this actor on this
subject*, computed from the transition spec, the contract's event history, the actor's
grants and fences, the tier, and the halt state.

The envelope is versioned, schema-stable JSON with structured errors and meaningful exit
codes ● — but the exit codes for illegality (contention, fencing, tier gates) become rare
events rather than the primary feedback channel, because the legal set was in the
previous envelope. An agent's loop becomes: read envelope → pick a legal verb → act. This
is the cheapest reliability improvement in the design: it removes an entire class of
wasted turns and makes the model's job "choose among legal moves" instead of "guess and
be corrected."

The transition spec that drives affordances is the same data that drives append-time
validation and the same data that generates the documentation (§13) — one artifact, three
consumers, zero drift.

### 8. Lanes

Work flows through six lanes. Each is a role definition (grants + conventions), not a
privileged binary; any qualified actor can staff a lane, and every lane's inputs and
outputs are ordinary events.

1. **Dispatcher** — turns intents and mirror requests into routed, prioritized contracts
   with draft acceptance specs; triages; never implements. The only lane that converts
   outside-world input into system state, which is why its input handling is the most
   heavily fenced (§11).
2. **Planner** — turns an unplanned contract into a falsifiable plan PR. Planning
   receives the strongest models and the most context; a wrong decomposition poisons
   everything downstream, so plan quality is measured (unedited-approval rate) and
   invested in first.
3. **Implementer** — takes an approved plan to a submission: worktree, minimal diffs,
   evidence gathering, heartbeats with real progress payloads, deliberate exits.
4. **Verifier** — renders verdicts (§5). Staffed by the reviewer lane's app identity,
   qualified human keys, or both; never by the implementer's key.
5. **Curator** — the offline learning loop (§9). Runs on idle or on schedule, never in
   the path of live work.
6. **Maintenance** — reaps expired and wedged claims (always leaving packets ●), prunes,
   rebuilds projections, runs lints, files defect contracts for anomalies. Safe to run
   unattended on a schedule, and itself fully audited — it is just another actor. ●

**Escalation.** Any lane can raise `blocked(needs-you)`: an event addressed to a human
gate carrying the packet, the specific question, and the minimal decision needed. The
supervisor routes it, the report surfaces it, and nothing else about the contract moves
until it is answered. A human's unit of interruption is one decision, never a transcript.

### 9. The curator and the flywheel

The curator is the system's answer to "preserving experience is not learning."

**Evidence first, always.** Online lanes append evidence — trajectories summarized in
heartbeats and submissions, metering, verdicts, failures — and never write conclusions.
The raw record is immutable by construction (it's the ledger).

**Distillation offline.** On idle or on schedule, the curator: groups completed contracts
by family; compares trajectories *across* the family (what did successes share, what did
failures lack, under which environment versions); proposes **lessons** with full
provenance — applies-when conditions, supporting contract ids (a recommendation requires
support from more than one non-failed trajectory; a single accidental success is never
promotable), exceptions, and a last-validated stamp; and proposes **retirements** when new
evidence contradicts old lessons — evidence kept, conclusion revoked, never silently
deleted.

**Promotion through gates.** A lesson candidate becomes durable knowledge only through a
PR to the knowledge area, reviewed like any change; a lesson that should change *behavior*
routes to the right carrier — a knowledge doc if it's a fact, a role-file or skill patch
if it's a stated procedure, a workflow or harness change if it's mechanizable, and each
through its own gate. The curator can propose anything and approve nothing (principle 9).

**Dead ends symmetrically.** Failed approaches are recorded with the failure condition
and environment, so they can be *un-retired* when the environment changes — a dead end is
a fact about a context, not a permanent verdict.

**The workflow flywheel.** The curator's second job: when the ledger shows the same shape
of trajectory recurring, it drafts a deterministic workflow from the recorded runs (noise
filtered, variables extracted), validates it in mock, and proposes it as a PR. Once
merged, the second execution of that chore is a program; agents handle only exceptions,
and a failing step falls back to a bounded repair role that proposes the workflow patch —
as a PR, never as silent self-modification. Every chore an agent does twice becomes
infrastructure. This is the compounding loop, and it is a founding component, not an
add-on: the ledger is its recorder, the workflow engine its output format, the gates its
safety.

**Retrieval.** Promoted knowledge is indexed and surfaced *into the packet and the
envelope* at the moments it applies (a lesson's applies-when conditions are matchable
against the contract's family and files), because knowledge nobody is shown at the right
moment is knowledge that doesn't exist.

### 10. Workflows and determinism

The workflow engine carries forward substantially as built ●: DAGs of steps (AI, `run:`,
gate-only, loop groups) with typed inputs, schema-enforced produces, dependency waves,
retries, `on_fail` policy, wall-clock budgets, and checkpoint/resume. Gates cover the
trust boundary (approval, review with verdict loops, CI checks). Mock mode is total —
zero credentials, zero side effects, gates auto-passed — so every workflow is testable in
CI. ● Validation is exhaustive and refuses invalid graphs. ● Secrets are vault-indirect,
resolved at run time, never frozen into YAML or echoed to logs.

Two deliberate boundaries: workflow definitions live in the repo and change by PR (the
flywheel proposes, gates dispose); and the engine remains DAG + gates — statechart
semantics (hierarchical/history states) are rejected as expressiveness the validation
suite would have to chase, revisited only if a real workflow cannot be expressed.

### 11. Guardrails and governance

**Tiers.** Autonomy tiers gate what an actor may do without a plan or an operator,
declared per-squad/per-path in checked-in config. ●

**The protected surface.** The orchestration contract — transition spec, guardrails,
verifier code and rubrics, sealed-check keyring, curator promotion gates, role
definitions, the supervisor's policy — requires PR + owner review, and **no agent key
holds write capability to the gates that approve its own work**. ● The set of
non-self-modifiable things is enumerated in config, not implied.

**Data/instruction fencing.** Intent prose, message bodies, mirrored text, and tool
output are data. ● Wherever such content is interpolated into a prompt, it is wrapped in
unforgeable delimiters; wherever it is interpolated into a command, it is validated
against strict shapes. The dispatcher — the lane that touches the most outside text —
runs with the least standing capability.

**Budgets.** Real-spend denominated (§6), org/actor/contract granularity, soft/hard
enforcement by declared capability, remaining-balance visible to workers.

**Per-verb policy.** The MCP/API surface carries allow / deny / require-approval per
actor and risk class, with approvals routed to an operator inbox as request events and
resolved attributably.

**Server-side mirrors of the contract.** Branch protection with required checks, ledger
ref no-force-push/no-delete, release tags immutable — declared in a checked-in
desired-state file that a reconciler diffs, applies, and verifies against the forge, so
protections are versioned intent, not clicked-through hope.

### 12. Interoperability

**Harness-agnostic.** Any coding harness can staff any lane, because the contract is
files, a CLI, and envelopes — never a vendor SDK. ● Per-harness config fans out from
single sources and drifts are CI failures. ● Model and harness names are registry data.

**Forge-agnostic.** GitHub is an integration: the core loop — ledger, contracts, claims,
verdicts — runs on any git remote with zero forge features. Forge adapters supply the
extras (PR gates, protections reconciler, mirrors) per forge.

**Spec-derived docs.** The lifecycle documentation in AGENTS.md and the role files is
*generated* from the transition spec through the same fan-out, so prose about the state
machine structurally cannot drift from the machine. Design docs carry
`last-verified: <date> @ <commit>` stamps with a CI lint against stale stamps on
behavior-changing PRs.

**Trust boundaries.** Within an organization, projections and request events suffice.
Across organizations, collaboration speaks a public protocol (A2A-shaped): capability
cards, a task lifecycle state machine, and opaque collaboration — tasks and artifacts
cross the boundary; prompts, reasoning, and internals never do.

**Federation.** One ledger per repository, always. An organization-level view is a
read-only projection *across* ledgers (remotes addressed uniformly, e.g.
`seed --remote infra queue`); cross-repo work is an intent filed into the target repo's
ledger as a request. There is no super-ledger, because there is no super-authority.

### 13. Observability and operations

Everything observable derives from the ledger, because everything that happened is in it.

- **The report** — queue health, stalled verdicts, long-parked plans, unread messages,
  expired and *wedged* claims, escalations waiting, budget burn, human-vs-agent
  breakdown — is a projection, runnable offline.
- **The dashboard** — Tier 0: a static HTML report artifact per maintenance tick. Tier 1:
  a live, read-mostly server (ledger watch → snapshots) whose only write path is port
  verbs as an enrolled service identity. Tier 2: multi-repo federation views, live event
  feed, packet viewer. At every tier the dashboard holds zero state beyond view
  preferences — the moment it caches writes or grows annotations it has rebuilt someone
  else's Postgres by accident.
- **Side-channel audit.** Harness hooks append tool-use events (never-throw writers; a
  broken hook must never hurt a session), and the audit view separates port-mediated
  actions from raw git/file side channels — giving "use the port" evidence and teeth, and
  feeding the flywheel's recorder.
- **Doctor.** One preflight command checks everything — ledger reachable and chain-valid,
  keyring sane, projections fresh, hooks installed, workflow registry valid, protections
  reconciled, version pins consistent — with per-integration health and fix-it hints.
- **Metrics from git alone.** Cycle time, verdict latency, rework rate, packet-resume
  success, cost per contract, flywheel conversion (chores → workflows) — all derivable by
  anyone with a clone, because the ledger is the metrics store.

### 14. Evaluation as infrastructure

The evaluation environment is part of the template, not tooling around it — because both
qualification (§3) and calibration (§5) depend on it.

- **Eval contracts**: synthetic work items with known verdicts, run through the real
  machinery (claim, plan where applicable, implement, verify) against fixture repos.
  Passing them earns and retains role grants.
- **Verifier calibration**: rubric-scored verdicts are periodically sampled against a
  human-scored gold set; drift beyond tolerance suspends the affected rubric's authority
  (falls back to human verdict) and files a defect contract.
- **Drills, in CI**: disaster recovery (rebuild all projections from genesis; restore
  from checkpoint), packet-resume (kill an executor mid-contract; a fresh executor
  completes from the packet alone), race storms (concurrent claim contention converges
  with no lost updates), and halt (appends refuse until lifted).
- **Boundary + retention for the system itself**: any change to a role file, guardrail,
  rubric, or workflow names the failing case it fixes and demonstrates the
  previously-working cases still pass. A fix validated only against its trigger is not
  accepted.

### 15. Distribution, supply chain, migration

- **Template + pinned engine.** ● Adopting is clone-and-init; the engine is a pinned
  release fetched by a bootstrap shim, SHA-256-verified from a checked-in lock, cached
  outside the repo, never committed; vendored paths serve air-gapped use. Upgrades move
  pins against tagged releases with checksum verification, protocol preflight, and
  atomic lock rewrite. ●
- **Everything hash-pinned.** Engine, skills, workflow dependencies; nothing executes
  from an unpinned source. ●
- **Preseed.** One declarative file bootstraps a new adoption — config, guardrails,
  teams, protections desired-state — idempotently and CI-verifiably.
- **Migration from the current system.** The current store's lossless export is the
  migration tool: export → transform to genesis ledger (each historical run-log entry
  becomes a ledger event; anchors verify the source history before conversion) → import
  refuses non-empty ledgers. The migration is a documented, drilled, two-command path,
  and it is how we prove the export requirement was real.
- **Boring install.** One command, no telemetry, no account, no phone-home. ●

### 16. Deliberately absent

Absence is design. Each of these is omitted with a reason, so future proposals to add
them must argue against the reason, not against forgetfulness:

| Absent | Why |
|---|---|
| Authoritative external backends | A second writable authority is the root of every sync pathology; §7.7's logic, made permanent. Exporters + request adapters cover every real need at a fraction of the complexity. |
| A separate mail subsystem | A message is an event; threading, unread counts, and acks are projections. |
| A separate anchors mechanism | The signed hash chain *is* the tamper-evidence; checkpoints *are* the recovery points. |
| A separate run log / audit log | The ledger is both. |
| A machine-local card backend | The SQLite projection provides the throughput without a second authority. |
| Terminal-multiplexer coupling | Wake channels are adapter details; no coordination feature may assume any of them. |
| Statechart workflow semantics | DAG + gates covers observed use; expressiveness the validators must chase is a cost, not a feature. |
| Model-vendor coupling anywhere | Models are scheduling inputs (cost class, qualification), never architecture. |
| A hosted control plane | The repository is the control plane. Anything a server adds must be a projection anyone can re-derive. |

---

## Part III — Acceptance Criteria

A criterion is met only when it is implemented, tested, documented, and **enforceable**
(by lint, CI, drill, or protocol — not by convention alone). Criteria marked ● exist in
the current system at least partially and must be preserved through the transition; all
others are targets. This section will replace AC.md when SEED-NEXT becomes the build.

### A. The Ledger

- [ ] Every coordination fact is an event on a single per-repo ledger ref; there is no
      second writable store of coordination state anywhere in the system.
- [ ] Events carry seq, timestamp, actor fingerprint, verb, subject, schema-valid
      payload, previous-event hash, and a signature over the canonical form; a verifier
      can validate the entire chain from genesis with one command.
- [ ] Any mutation of history — reorder, rewrite, deletion, forgery — is detected by
      chain verification and refused by every reader; the drill proves it with
      deliberately corrupted fixtures.
- [ ] Appends are atomic per push; a losing push re-validates its events against the new
      tip before re-append, and events invalidated by the race (a lost claim) are
      reported to their actor, never silently re-appended. ●
- [ ] Ordering authority is the sequence number, never the timestamp; clock skew between
      writers cannot reorder history.
- [ ] Checkpoints embed a projection-state hash; replay from a verified checkpoint equals
      replay from genesis (proven in CI on every release); checkpoints never truncate
      retained history, and any pruning policy is itself recorded as an event.
- [ ] A genesis event names the initial operator keys and protocol version; protocol
      version is enforced with a distinct exit code on mismatch. ●
- [ ] `halt.declared` stops all appends by every writer except an operator's
      `halt.lifted`; the halt drill proves no writer path bypasses it. ●
- [ ] The ledger is offline-native: all local operations work with zero network; appends
      queue and reconcile on reconnect. ●
- [ ] Ledger performance is bounded and measured: append latency, replay-from-checkpoint
      time, and projection rebuild time have budgets tracked in CI against a
      representative history size.

### B. Projections

- [ ] Every read surface (queue, contract view, actor view, report, cache, dashboard,
      mirrors) is a deterministic function of a ledger prefix, carries the seq it was
      built at, and can be deleted and rebuilt with one command, byte-identically.
- [ ] No code path anywhere writes a projection directly or treats one as authoritative;
      a lint over the engine enforces the write boundary.
- [ ] Staleness is always visible: any surface displaying projected state displays its
      seq stamp; consumers can demand a minimum seq.
- [ ] The local cache projection (SQLite) delivers single-machine read throughput
      equivalent to the current machine-local backend, with zero authority: deleting the
      cache file mid-operation loses nothing.
- [ ] External mirrors (issues, Jira, Linear, …) are one-way exporters; a mirror-side
      edit arrives only as a request event appended by a governed identity, validated
      against strict shapes, and honored or refused under guardrails — proven by a
      conformance suite every exporter/adapter passes.
- [ ] Bidirectional synchronization with any external store is structurally impossible:
      no component holds both an export path and a direct write path.
- [ ] A destroyed-and-rebuilt-from-genesis drill (all projections) runs green in CI.

### C. Identity and attribution

- [ ] Every actor is a keypair; enrollment, grants, suspension, and revocation are
      events; the roster/keyring is a projection of them.
- [ ] Every event's signature is verified on append and on read; an event signed by an
      unenrolled, suspended, or revoked key is refused with a structured error.
- [ ] Agent enrollment requires exactly an identity plus a scoped git credential; no
      inbound connectivity, daemon, or registration server is ever required. ●
- [ ] Role grants are capability data checked at append time; an actor cannot perform a
      verb its grants do not cover, and the refusal is structural, not advisory.
- [ ] Operator-only verbs refuse non-operator keys; agents can never approve their own
      work into done — enforced by key disjointness, not workflow convention. ●
- [ ] Humans and machines are distinguished in the roster; guardrails that bind only
      agents, and metrics that split human/agent activity, read that distinction. ●
- [ ] Grants above the trivial tier cite passing eval-contract results; the citation is
      checkable; the supervisor can suspend grants on failed spot-check evals without
      operator ceremony (and the suspension is an attributable event).
- [ ] Key rotation and revocation are drilled: a compromised key can be revoked, its
      unexpired claims reaped, and its historical events remain attributed to it.

### D. Work contracts and lifecycle

- [ ] A contract cannot leave draft without an acceptance spec: machine-checkable
      commands/assertions/artifacts, plus rubric items for qualitative residue.
- [ ] Above the trivial tier, contracts carry sealed checks encrypted to the verifier
      keyring; the sealed hash in the ledger proves the checks predate implementation;
      no implementer capability can decrypt them — proven by a capability audit in CI.
- [ ] The lifecycle vocabulary (backlog, ready, in_progress, review, done, blocked,
      cancelled; claim as the ready→in_progress transition, not a state) is exactly the
      transition spec's, which is data, validates itself, and refuses to load when
      inconsistent. ●
- [ ] Claims are synchronous and exclusive with fencing: a stale fence on any subsequent
      verb is refused with a distinct code; contention returns a structured envelope. ●
- [ ] Heartbeats carry progress payloads; expiry (no heartbeat) and wedging (heartbeats
      without progress change for N periods) are distinct, visible conditions with
      distinct reap heuristics — and the wedged case is covered by a drill.
- [ ] Every exit from in_progress is deliberate (submit, release, park, reap) and every
      involuntary exit leaves a three-part handoff packet; silent abandonment is
      impossible by construction. ●
- [ ] Packets contain exactly: acceptance criteria, settled decisions/constraints, and
      artifact references by path — bounded in size, mechanical in content, with a CI
      lint on shape. ●
- [ ] Packet sufficiency is drilled: an executor killed mid-contract is replaced by a
      fresh executor that completes the contract from the packet alone, in CI, on a
      fixture contract.
- [ ] Plans are falsifiable change contracts: boundary set, retention set, validation
      commands for both, and expected diff shape are required sections; a plan missing a
      retention check does not lint. ●
- [ ] Plan PRs and implementation PRs remain structurally disjoint, classified and
      enforced by CI. ●
- [ ] Dependencies cascade with wakeups: closing a contract unblocks dependents, plan
      merges unpark plan-blocked contracts, and every unblock wakes the affected party
      through its adapter — no contract waits on a poll.
- [ ] Holds cascade down subtrees and suppress wakeups; initiative trees roll up child
      states into progress views.
- [ ] Goal ancestry: open work traces to a stated mission, and the report warns when it
      does not. ●

### E. Verdicts and evidence

- [ ] "Done" is reachable only through a signed `verdict.rendered` event from a key
      holding the verdict grant and provably distinct from every key that implemented the
      contract; there is no other path, including for operators (an operator override is
      its own attributable verb, not a disguised verdict).
- [ ] The verifier executes in a clean, per-run-id isolated workspace; parallel verdicts
      never collide; cleanup fires on pass and fail.
- [ ] The verdict's receipt binds: contract id, plan hash at merge-base, diff hash,
      changed-file inventory, visible-check transcripts, sealed-check results, and
      environment fingerprint; receipt verification recomputes everything from the
      submission head and fails on any mismatch. ●
- [ ] The verifier has no input channel for implementer claims: its verdict inputs are
      enumerable and are exclusively things it executed or read itself (the
      information-gain criterion, structurally enforced).
- [ ] A red verdict is unmergeable (required check) and locks the implementer out of
      self-approval paths until a new submission. ●
- [ ] Rubric verdicts score item-by-item with cited evidence and explicit uncertainty;
      low-confidence rubric items route to human verdict rather than resolving silently.
- [ ] Rubric calibration runs on a schedule against a human-scored gold set; drift
      beyond tolerance suspends that rubric's authority automatically and files a defect
      contract.
- [ ] Verifier code, rubrics, thresholds, and the sealed-check keyring live on the
      protected surface; no agent key (verifier keys included) can modify the gates that
      judge its own lane.
- [ ] Evidence, receipts, and verdicts are queryable as a data layer (by contract, actor,
      time range, outcome) — not just files that happen to exist.

### F. Supervisor and execution

- [ ] A supervisor loop ships with the template and is the recorded answer to "who plays
      employer": it schedules from the queue projection, respecting priority, tier,
      grants, qualification, remaining budget, and concurrency caps — and every
      assignment is an attributable event.
- [ ] Workers pull; no coordination feature requires inbound connectivity to any
      executor; wake channels are declared per-adapter and are accelerators only — the
      system is fully correct with every wake channel disabled (proven by a
      polling-only CI run). ●
- [ ] Executor adapters implement provision/wake/meter for at least: local worktree,
      container, cloud agent session, and enrolled remote worker; the adapter interface
      is public and documented so new substrates are additive.
- [ ] Disposability is enforced: an executor's workspace can be destroyed at any moment
      after its last confirmed push with zero loss — drilled in CI by killing executors
      at randomized points and completing their contracts elsewhere. ●
- [ ] Every run emits metering events (tokens, cost, wall-clock) attributed to contract,
      actor, and model class; budget enforcement, reports, and scheduling read the same
      stream; there is no unmetered execution path.
- [ ] Budgets are denominated in real spend at org/actor/contract granularity; soft
      alert at threshold, hard stop at cap where the substrate can enforce, advisory
      where it cannot — and which applies is declared, not discovered.
- [ ] Remaining budget is visible to the worker in its envelope, enabling budget-aware
      strategy; exhaustion mid-contract produces a deliberate park with packet, never a
      vanished worker.
- [ ] Preemption is graceful-first: the supervisor's interrupt reaches the executor at a
      safe point and produces a park; force-kill is the fallback and still yields a reap
      packet.
- [ ] Model/harness routing is data: cost classes and qualification map work to
      substrates without code changes.

### G. Affordances and the actor interface

- [ ] Every verb response is a versioned, schema-stable envelope with structured errors
      and meaningful exit codes ● — and includes the set of verbs currently legal for
      this actor on this subject, computed from the transition spec, event history,
      grants, fences, tier, and halt state.
- [ ] The affordance computation and the append-time validation are the same code over
      the same spec — an action listed as legal cannot be refused for legality reasons
      one seq later (absent a concurrent event, which the envelope's seq stamp makes
      detectable).
- [ ] The CLI is the complete interface: everything an agent ever needs is a verb; the
      MCP surface exposes the same verbs with identical semantics. ●
- [ ] Refusals remain structured and machine-branchable ●, and refusal rates are a
      tracked metric — a rising refusal rate signals an affordance gap, not agent error.
- [ ] Windows is first-class; shell parity is documented and tested. ●
- [ ] Relevant promoted knowledge (lessons whose applies-when matches the contract)
      surfaces in the packet and envelope at claim time — knowledge is delivered at the
      moment of use, not merely stored.

### H. Lanes

- [ ] Role definitions exist for all six lanes — dispatcher, planner, implementer,
      verifier, curator, maintenance — as grants + conventions composable from ordered
      fragments (base + role + squad), resolved and checked by validation. ●
- [ ] The dispatcher is the only lane converting outside input into system state; it
      runs with least standing capability, and its input handling passes the injection
      conformance suite (hostile fixture corpus: instructions embedded in intents,
      mirrored issues, and tool output must be quoted as data, never obeyed). ●
- [ ] Dispatcher output quality is measured: routed contracts carry draft acceptance
      specs, and the rate of human re-triage is tracked.
- [ ] Planner quality is measured: plan PRs' unedited-approval rate is tracked, and the
      planner lane receives the strongest available model class by scheduling policy.
- [ ] Implementers follow the worker contract: claim before work, heartbeat with real
      progress, minimal diffs, deliberate exits, evidence gathered before submission. ●
- [ ] Maintenance runs green unattended on a schedule — reap (expired and wedged),
      projection rebuilds, lints, mirror export, checkpoint — and is itself fully
      audited as an ordinary actor. ●
- [ ] Escalations are bounded: a `blocked(needs-you)` event carries the packet, the
      question, and the minimal decision; the report surfaces waiting escalations with
      age; and escalation resolution latency is a tracked metric.
- [ ] Any qualified actor can staff any lane it holds grants for; lanes are roles, not
      binaries — proven by running the full loop with a single actor holding all grants
      (small-team mode) and with disjoint actors per lane (fleet mode) in CI.

### I. Curation, memory, and the flywheel

- [ ] Online lanes append evidence only; no online path writes conclusions, lessons, or
      long-term knowledge — the write boundary is enforced by grants.
- [ ] The curator runs offline (idle or scheduled), groups completed contracts by
      family, and compares across trajectories; its proposals are events pointing at
      candidate PRs, never direct writes to knowledge.
- [ ] A promoted lesson carries: applies-when conditions, ≥2 supporting non-failed
      contract ids, exceptions, provenance links, and a last-validated stamp; a single
      accidental success is structurally non-promotable (lint on the promotion PR).
- [ ] Lessons route to the correct carrier — knowledge doc, role/skill patch, workflow,
      or harness change — each through its own gate; the curator can propose to any
      carrier and approve none.
- [ ] Retirement is precise: contradicting evidence produces a retirement proposal that
      revokes the conclusion and keeps the evidence; retired lessons remain queryable
      with their retirement reason.
- [ ] Dead ends record failure condition and environment, and are un-retirable when the
      environment changes; the curator checks dead-end applicability, not just lesson
      applicability.
- [ ] The flywheel closes: recurring trajectory shapes are detected from the ledger,
      drafted as deterministic workflows, validated in mock, and proposed as PRs; the
      conversion rate (recurring chores → merged workflows) is a tracked metric.
- [ ] Workflow self-healing is gated: a failing step's repair role completes the run
      within bounds and proposes the workflow patch as a PR — silent self-modification
      is impossible (the workflow registry is on the protected surface).
- [ ] Knowledge bloat is managed: consolidation merges duplicates (provenance retained),
      long-unused lessons are flagged for revalidation, and the knowledge area has a
      size/structure lint — a handbook, not "99 ironclad rules."

### J. Guardrails and governance

- [ ] Autonomy tiers gate un-planned and un-operator'd action, declared per-squad and
      per-path in checked-in config. ●
- [ ] The protected surface is enumerated in config — transition spec, guardrails,
      verifier code/rubrics, sealed-check keyring, curator gates, role definitions,
      supervisor policy — requires PR + owner review, and is write-denied to every agent
      key whose work it gates; the capability audit proving disjointness runs in CI. ●
- [ ] Data/instruction fencing is mechanical: interpolation of ledger-carried prose into
      prompts uses unforgeable delimiters; interpolation into commands validates against
      strict shapes; the hostile-corpus suite passes on every release. ●
- [ ] Per-verb policy (allow/deny/require-approval by actor and risk class) governs the
      MCP/API surface; approvals are request events resolved attributably in an operator
      inbox.
- [ ] Forge-side protections are declared desired-state in the repo and reconciled by
      command: branch protection with required checks, ledger ref no-force-push/
      no-delete, immutable release tags — diff, apply, verify. ●
- [ ] Scheduled and CI identities are least-privilege; no scheduled job can push to the
      default branch; ledger pushes from automation use a dedicated machine identity. ●
- [ ] Process changes pass boundary + retention: any change to a role file, guardrail,
      rubric, or workflow names the failing case it fixes and demonstrates prior cases
      still pass — enforced as a PR convention with a lint for the declared sets.

### K. Workflows

- [ ] The engine runs DAGs with typed inputs, schema-enforced produces, dependency
      waves, triggers, when-expressions, retries, on_fail policy, wall-clock budgets,
      and checkpoint/resume that refuses mixed-graph resumes. ●
- [ ] Gates cover the trust boundary: approval (pause → response → resume), review
      (verdict loop with remediation and max revisions), checks (CI green + zero
      unresolved threads via forge adapters). ●
- [ ] Mock mode is total — zero credentials, zero side effects, schema-valid stubs,
      recorded-not-executed commands, auto-passed gates — so every workflow is testable
      in CI. ●
- [ ] Validation is exhaustive (schema, ids, acyclicity, action XOR, artifact/role/
      registry closure, token lint, loop rules) and refuses invalid graphs. ●
- [ ] Secrets are vault-indirect: resolved at run time from env/keychain/extension,
      never frozen into definitions, never echoed to logs.
- [ ] Workflow definitions change only by PR; the registry is on the protected surface.

### L. Interoperability and federation

- [ ] Any harness staffs any lane: per-harness config fans out from single sources,
      drift fails CI, fan-outs are never hand-edited. ●
- [ ] Lifecycle documentation (AGENTS.md state-machine prose, role lifecycle tables) is
      generated from the transition spec via the same fan-out — spec and docs cannot
      drift.
- [ ] Design docs carry `last-verified` stamps; a CI lint flags behavior-changing PRs
      whose subsystem doc kept a stale stamp.
- [ ] The core loop runs on any git remote with zero forge features; forge extras (PR
      gates, protections, mirrors) are adapters with documented equivalents for at least
      one non-GitHub forge.
- [ ] Cross-organization collaboration is opaque: a public protocol layer (capability
      card, task lifecycle, artifacts-only exchange) crosses trust boundaries; internals
      never do.
- [ ] Federation is projection-only: remotes are addressed uniformly for read
      (`--remote`), cross-repo work enters as request events into the target ledger, and
      no super-authority exists — proven by the absence of any cross-ledger write path.
- [ ] Nothing in the worker contract names a model vendor; models are registry data and
      scheduling inputs. ●

### M. Observability and operations

- [ ] The report is a projection covering: queue health, stalled verdicts, long-parked
      plans, unread messages, expired and wedged claims, waiting escalations with age,
      budget burn, and human/agent breakdown — runnable offline from a clone. ●
- [ ] Harness hooks append tool-use audit events with never-throw writers; the audit
      view separates port-mediated actions from side channels; doctor verifies hook
      health.
- [ ] `doctor` is one preflight for everything — ledger chain, keyring, projection
      freshness, hooks, workflow registry, protections, version pins — with
      per-integration health and fix-it hints.
- [ ] The dashboard is projection-only at every tier (static artifact → live
      read-mostly server → federated), writes solely through port verbs as an enrolled
      identity, and holds zero state beyond view preferences — audited by inspection of
      its storage.
- [ ] Core metrics derive from the ledger alone: cycle time, verdict latency, rework
      rate, packet-resume success, cost per contract, refusal rate, flywheel conversion,
      escalation latency — computable by anyone with a clone.
- [ ] Every refusal, everywhere, is a structured envelope a machine can branch on and a
      human can read. ●

### N. Evaluation infrastructure

- [ ] Eval contracts (synthetic work with known verdicts, fixture repos) run through the
      production machinery; passing them gates role grants; spot-check evals run on a
      schedule against active agents.
- [ ] The verifier-calibration loop runs on a schedule against a human gold set, with
      automatic authority suspension on drift.
- [ ] The four drills run in CI: full projection rebuild from genesis; packet-resume
      with randomized executor kills; concurrent-claim race storms with convergence
      proof; halt enforcement across all writer paths.
- [ ] Trajectory-prefix regression exists for lane behavior: recorded decision points
      (e.g., "about to declare done without running checks") are replayable against
      lane configurations to catch behavioral regressions in role/prompt changes.
- [ ] Simulation mode runs the whole system against synthetic intents end-to-end (mock
      executors, fixture repos) with zero credentials — the system's own mock-total
      test.

### O. Distribution, supply chain, migration

- [ ] Adoption is clone-and-init from a tagged release; template upgrades merge
      three-way against a lock with --check and explicit rollback. ●
- [ ] The engine is a pinned, SHA-256-verified release fetched by a bootstrap shim,
      cached outside the repo, never committed; air-gapped paths exist. ●
- [ ] Engine upgrades verify checksums, preflight protocol compatibility, refuse
      semver downgrades, and rewrite the lock atomically. ●
- [ ] Everything executable is hash-pinned: engine, skills, workflow deps; nothing runs
      from an unpinned source. ●
- [ ] Preseed: one declarative file bootstraps config, guardrails, teams, and
      protections desired-state, idempotently, verified in CI.
- [ ] Migration from the predecessor system is a documented, drilled, two-command path:
      lossless export → genesis import (which refuses non-empty ledgers), with source
      history verified before conversion; the drill runs against a real predecessor
      fixture in CI.
- [ ] Install is boring: one command, no telemetry, no account, no network beyond the
      pinned artifact fetch. ●

### P. Quality, docs, community

- [ ] One fast backpressure command stays green on main. ●
- [ ] Engine coverage stays above 90% (union statement coverage); template scripts and
      hooks carry smoke tests; the exporter/adapter conformance suite runs per
      integration. ●
- [ ] Docs are governed: operator handbook, generated worker docs, design docs gated by
      the mechanism they enable, stamped and linted. ●
- [ ] The research corpus is maintained: sources surveyed, adopted ideas traced,
      rejections recorded with reasons — and §16's absences are treated as standing
      rejections that proposals must argue against explicitly. ●
- [ ] Decisions are recorded and binding; contributor docs state the authority order. ●
- [ ] Governance is fork-friendly: MIT, no CLA, no open-core split, stated explicitly. ●
- [ ] Dogfooding is total: the system's own development runs through the system, and
      every feature ships only after coordinating its own delivery. ●

### Q. The autonomy end-state

- [ ] A human files a one-sentence intent; the dispatcher produces a routed contract
      with a draft acceptance spec that survives human review unedited in the majority
      of cases.
- [ ] Planner plan-PRs pass human review >80% unedited; implementers take approved
      plans to green, verdict-passed submissions without intervention on the happy path.
- [ ] The verifier lane holds quality alone on low tiers; humans review only high-tier
      plans, the protected surface, and escalations.
- [ ] Every escalation reaching a human is one packet + one question + one decision;
      transcript-dumping on a human is a defect.
- [ ] The system runs unattended for a week on a real backlog with: zero ledger-chain
      violations, zero lost updates, zero silent abandonments, zero guardrail breaches,
      zero unmetered spend — and the ledger alone reconstructs and justifies everything.
- [ ] The flywheel demonstrably compounds over a quarter: recurring-chore workflow
      conversion, packet-resume success, and cost-per-contract all trend favorably, from
      ledger metrics alone.
- [ ] A team that has never spoken to the authors adopts from the README in under an
      hour, on their forge, with their harness, and reaches their first agent-shipped,
      verifier-passed, human-reviewed PR the same day. ●
