# SEED-NEXT.md — The Next Substrate

> **Authority note.** This document is the target architecture for the next major version
> of open-seed, designed from first principles with everything the research program has
> taught us (docs/research/01–12, the binding §7.1–7.5 decisions, the proposed §7.6–7.7
> sections, and the adjacent-tool field). It is **vision and acceptance criteria, not
> design authority**: [`docs/design-options.md`](docs/design-options.md) remains binding
> for the current system, and any element here that contradicts settled design becomes
> real only through a `docs/design-options.md` PR. This document will eventually replace
> [`AC.md`](AC.md); until then AC.md remains the checklist for the current build, and
> criteria marked ● below exist in the current system and carry forward.
>
> **Revision note.** Hardened 2026-08-30 after an adversarial architecture review. The
> major corrections: an explicit **admission boundary** replaces the earlier implicit
> assumption of cooperative writers; the security invariant in §I.2 is now the organizing
> test; offline semantics, event-stream separation, budget reservation, verifier failure
> domains, sealed-check mechanics, ledger/forge atomicity, and the learning loop's
> poisoning defenses are specified rather than asserted.

---

## Part I — Vision

### 1. The problem, stated from zero

A software organization wants to run most of its engineering through fleets of AI agents.
The agents are capable but unreliable in specific, well-documented ways: they claim
completion without proof, they drift from instructions, they collide with each other, they
forget everything between sessions, and they can be manipulated by the content they read.
Worse: any individual agent must be assumed *compromisable* — by prompt injection, by a
stolen credential, by a poisoned dependency. The organization cannot accept losing
determinism (what happened must be knowable), reviewability (a human must be able to
check any of it), or ownership (the coordination state must belong to the org, forkable
and portable, hostage to no vendor's database).

Every system in this space chooses where truth lives, who is allowed to say "done," and —
the question this revision adds — **what a fully compromised participant can make the
organization believe**. Get those wrong and no amount of agent capability saves you; get
them right and even mediocre agents compound.

### 2. The security invariant

SEED-NEXT is built around one test, and the architecture is not settled until it passes:

> An enrolled implementer is fully compromised. It holds its legitimate private key and
> its legitimate git credential. It can invoke arbitrary git commands, forge its own
> client, modify anything its OS account reaches, and emit arbitrary syntactically valid
> events. It cannot compromise another actor's key or the trusted git infrastructure.
>
> **Then:** it can spend only its reserved budget; modify only its authorized code
> surface; produce signed lies attributed to itself; and submit work for independent
> verification. It **cannot** claim unauthorized work, approve itself, rewrite history,
> alter gates, impersonate another actor, exceed its lease, or cause an invalid state
> transition to enter authoritative history.

Three properties this invariant forces us to keep distinct, because they are different
things: **tamper evidence** (you can detect that history was altered) is not
**authorization** (invalid history cannot enter); **serialization** (git orders the
pushes) is not **consensus** (the winning push was valid); and **attribution** (this key
signed it) is not **trust** (the principal behind the key meant it, or is who the roster
says). The design below never substitutes one for another.

### 3. Ten first principles

Each is a bounded property with an explicit edge, not an absolute — the edges are where
the earlier draft over-claimed, and they are stated here on purpose.

1. **History is the authoritative coordination record.** An append-only, hash-chained,
   signed log of events is the single source of truth *for coordination state*: what work
   exists, who holds it, what was decided, what was verified, what was spent. Source
   code, build artifacts, secrets, external-world facts, and provider billing live in
   their own systems; the ledger records signed **observations** of them, and never
   pretends observation is control.

2. **One authority for coordination state; everything else is a projection.** Exactly
   one ledger is authoritative per repository. Every other coordination surface — queue
   views, dashboards, mirrors, caches — is a one-way, rebuildable, disposable projection.
   Inbound edits from any projection are requests. Bidirectional synchronization is
   forbidden. External systems remain authoritative for *their own* facts (a merge, a CI
   result, an invoice); the ledger holds observations of those facts, reconciled
   explicitly (§II.8).

3. **Admission is the boundary.** Validity is enforced where events enter authoritative
   history, by a trusted, minimal, stateless **admission boundary** — not by every writer
   promising to behave. Actors propose; admission validates and orders; the ledger
   records; projections interpret; executors execute; verifiers judge. Without this,
   "the ledger is the authority" quietly means "the git server is the authority," and
   every protocol rule is advisory to anyone with a push credential.

4. **Verification is the product.** The bottleneck of every agent loop is the verifier,
   not the model. Work is born as a contract — intent plus machine-checkable acceptance —
   and "done" is a verdict rendered by an independent verifier. The model may propose
   "done"; it can never approve it. Independence means separated *failure domains*, not
   merely separated keys (§II.8).

5. **Identity is attributable, never assumed trustworthy.** Every actor is a keypair and
   every event is signed — which proves possession of a key, and nothing more. The design
   distinguishes identity (the key), credential (what the key can reach), principal (who
   or what is behind it), and runtime (the model, harness, and environment actually
   executing) — and it qualifies and constrains the *runtime configuration*, not the bare
   key (§II.5). A signature attributes; the threat model decides what attribution is
   worth.

6. **Compute is disposable after durable synchronization points.** Any executor must be
   destroyable *after its last confirmed, admitted synchronization* with zero loss of
   coordination state. Work in flight since that point can be lost; the design bounds
   that loss window (frequent cheap sync, bounded findings in packets) instead of
   pretending it is zero.

7. **Affordances over refusals.** The system tells every actor what it may legally do
   next, computed from the spec, the state, and the actor's standing, before the actor
   acts. Refusals remain — admission refuses invalid events regardless of what any
   envelope suggested — but an actor that reads its envelope should rarely meet one.

8. **Learning is a governed, adversarially-hardened loop.** Evidence is collected
   online; lessons are distilled offline with provenance; and every promotion passes
   gates that assume the underlying trajectories may have been *constructed by an
   attacker*. A compounding mechanism that can be poisoned compounds poison; the curator
   pipeline is therefore staged, minimum-supported, expiring, and rollback-able (§II.12).

9. **Data is never instructions — as a defense-in-depth posture, not a solved problem.**
   Content flowing through the system is evidence to be judged, never instructions to be
   obeyed. No fencing makes an LLM immune to adversarial text in its context; therefore
   the *primary* control is capability restriction (a reader of untrusted content holds
   the least standing capability), and provenance typing, delimiter fencing, strict-shape
   interpolation, sandboxing, and network policy are layered on top (§II.14).

10. **Every gate has a governance root; nothing modifies its own gate.** No agent key
    holds write capability to the validators, verifiers, thresholds, keyrings, or
    admission rules that judge its own work. Self-modification is not eliminated — it is
    *routed*: someone can change the protected surface, and the design names exactly who
    (the governance root: operator keys plus owner review), through exactly which process.
    Humans hold gates, not queues: intent, high-consequence plans, the protected surface,
    and escalations — each escalation one packet, one question, one decision.

### 4. The end state

A repository where a human states an intent in one sentence; a dispatcher turns it into
contracts with acceptance checks; a supervisor publishes offers to a roster of enrolled,
qualified, cost-metered agents across whatever compute is cheapest that hour; planners
open falsifiable plans; implementers ship diffs from disposable machines; verifiers
render signed verdicts in separated failure domains; a curator distills the week's
trajectories into lessons and deterministic workflows through poisoning-resistant gates;
an admission boundary guarantees that none of it — not even from a compromised
participant — enters history invalidly; and at any moment, one command over the ledger
reconstructs exactly what happened, in what order, by whose key, at what cost, and on
whose authority. All of it open source, MIT, forkable, with no server holding any state
the repository cannot regenerate.

---

## Part II — Design

### 1. The Ledger

The ledger is the authoritative coordination record: an append-only sequence of events on
a dedicated git ref (`refs/seed/ledger`), replicated wherever the repository is
replicated.

**Event structure.** Every event carries:

- `ts` — timestamp (recorded for humans; never an ordering authority).
- `actor` — the acting identity's key fingerprint.
- `verb` — what happened (`intent.filed`, `claim.taken`, `verdict.rendered`, …).
- `subject` — what it happened to.
- `payload` — verb-specific structured data, schema-validated (see data classification).
- `prev` — hash of the preceding event's canonical form.
- `sig` — the actor's signature over the canonical form (which includes `prev`).

**Ordering.** Order is defined by admitted ancestry — the chain of `prev` hashes on the
authoritative ref. There is no writer-assigned global sequence number: in a multi-writer
optimistic design, a client-computed `seq` is an extra invariant every concurrent writer
must get right and gains nothing over ancestry. Where an ordinal is useful (projections,
citations), it is *derived* — assigned at admission or computed during projection — never
asserted by the proposer.

**Data classification.** The ledger is immutable and replicated, so what enters it is a
permanent commitment. Payloads therefore carry **coordination facts and references, not
content bodies**: hashes and paths for artifacts, receipts, transcripts, and any
user-authored or model-derived prose beyond short structured fields. Bulk and expirable
content — logs, transcripts, attachments, anything that could carry personal data,
secrets, or material someone may later be obliged to erase — lives in an addressed
artifact store with an erasure path, referenced by hash. A payload lint enforces the
classification at admission; "pruning is an event" applies to policy, and actual erasure
happens in the referenced store, by design rather than by apology.

**Checkpoints.** Periodic `checkpoint` events embed the hash of the canonical projection
state at that point, **signed by the maintenance or operator identity**. What a
checkpoint buys is explicit: a reader who has *once* verified a checkpoint against full
replay (or who explicitly trusts the checkpoint signer set — a declared trust choice, not
a default) may start from it. The checkpointed state's materialization format is
specified and versioned; a fresh clone's verification obligations are documented, not
implied. Checkpoints never truncate retained history.

**Genesis and halt.** A ledger begins with a signed `genesis` event naming the initial
governance root (operator keys) and the protocol version. `halt.declared` stops
admission of all events except an operator's `halt.lifted` — enforced *at the admission
boundary* (§2), which is what makes it real: a writer on a pre-halt tip, or one that
bypasses the client entirely, is stopped where events enter history, not by its own good
manners.

**Why git.** Offline-capable reads and drafts, replication by the same channels as the
code, forge-agnostic storage, inspectable with universal tooling. The ledger *data* is
portable git data forever. What git alone does not provide is enforcement — which is the
next section, and the one honest amendment to "a team that can host a git remote can host
the entire system": the remote must also run (or front) the admission validator.

### 2. The Admission Boundary

The admission boundary is the trusted component that makes every protocol rule real. It
is deliberately tiny: a **stateless validator** with no unique durable state — everything
it knows is derived from the ledger and the repo, and a replacement instance rebuilds
from a clone.

**What it does.** For every proposed event (or batch): verify the signature against the
enrolled keyring; check the actor's capability grants cover the verb; check the fence
against the current claim; check the transition against the spec; check schema and data
classification; check protocol version; check halt state; check budget reservation where
the verb spends (§9); then admit atomically onto the authoritative ref — or refuse with a
structured, attributable rejection. Admission is where ordering happens: the admitted
chain *is* the order.

**Implementations, by trust posture.** The boundary is a role, with three named
deployments:

- **Enforced, self-hosted**: a git server-side receive hook on the ledger ref — the
  reference implementation. The remote refuses any push whose events fail validation;
  force-updates and deletions of the ref are refused outright.
- **Enforced, forge-hosted**: where hooks aren't available (e.g. GitHub), the ledger ref
  is written only by a dedicated admission identity (a small service or serverless
  endpoint that actors propose to), and forge protections make that identity the ref's
  sole writer. The service holds no durable state.
- **Cooperative**: no server-side enforcement; every writer self-validates. This is the
  current system's posture and remains supported for small high-trust teams — but it is
  a **declared mode with a named consequence**: the security invariant (§I.2) does not
  hold, protocol rules are advisory against a hostile credential, and the doctor and
  README say so in plain words. Cooperative mode is a stance you choose, never a default
  you fall into.

**What this changes elsewhere.** Halt, fences, capability checks, and transition legality
stop being claims about client behavior and become properties of admitted history. The
git credential story also sharpens: an actor's credential allows *proposing* to the
ledger and pushing to its own authorized code branches — it never allows writing the
ledger ref directly in enforced modes.

### 3. Streams: the hot path and the ledger

A single ref appended by every actor for every event is a global mutex, and the earlier
draft would have melted it: with hundreds of workers, heartbeats and per-run metering
would force every unrelated verdict to race the same tip. The correction is a
classification of traffic, not a new database:

- **Coordination facts** — contract lifecycle, claims, parks, reaps, verdicts,
  enrollment, grants, budget reservations and settlements, promotions — are **admitted
  ledger events**. Low frequency, high value, globally ordered.
- **Observations** — liveness, progress, fine-grained metering — are **ephemeral
  streams**: written to a non-authoritative channel (a per-executor observation ref, a
  local socket to the supervisor, or the adapter's own telemetry), read by the
  supervisor and projections, and *summarized* into ledger facts at material transitions:
  `claim.taken`, `progress.milestone` (coarse, bounded frequency), `wedge.declared`,
  `run.settled` (aggregate metering at run end), `park`/`reap`.

Liveness is thus a signal, not a ledger write: a claimant's heartbeat-with-progress flows
on the observation channel every N seconds at near-zero cost; the *ledger* learns about
liveness only when something durable happens (a milestone, a wedge declaration, an
expiry). Lease semantics are unchanged — expiry (no observations) and wedging
(observations without progress change) remain distinct, visible conditions — but the
40-heartbeats-a-minute traffic never touches the authoritative tip.

Where a single admitted ref still contends at scale, admission may shard (per-squad or
per-class proposal queues merged by the admission point) — an implementation freedom the
design permits because ordering authority already lives at admission, not in writers.

### 4. Projections

A projection is a deterministic function from a ledger prefix (plus, where declared,
observation streams) to a view. All projections share five properties: derived (never
written directly), stamped (with the ledger position they were built at), rebuildable
(deletion loses nothing), read-only outward, and non-authoritative (no decision may
prefer a projection over the ledger; staleness is visible, never silent).

**Standard projections**: the ready queue (filtered by actor eligibility), contract
detail, actor view, report, local SQLite cache, dashboard, and external mirrors (GitHub
Issues, Jira, Linear, and anything else someone writes an exporter for).

**Inbound flow.** Input arriving at a projection surface — an edited mirror issue, a
dashboard button — enters the system exclusively as a *request event* proposed by a
governed identity, validated at admission like everything else. There is no path by
which an external system's database writes coordination state.

**External facts.** Mirrors of external authorities (forge merge state, CI results)
enter the ledger as **observations** (`merge.observed`, `check.observed`) recorded by
governed observers — the ledger never pretends to control what it can only watch (§8).

**What this deletes.** The authoritative-backend apparatus of the current system —
backend plugins as truth stores, capability negotiation, the emulated-claim variance
matrix — does not exist here. Integration effort goes into exporters and request
adapters, which are simple, and stays out of consensus, which is not. `fastcards`
dissolves into the SQLite projection: same speed, no second authority.

### 5. Identity, threat model, and qualification

**Four things, kept distinct.** An **identity** is a keypair. A **credential** is what
that key can reach (git scopes, API tokens). A **principal** is who or what is behind it
(a person, a team's bot, a vendor's runtime). A **runtime** is the actual executing
configuration — model and version, harness and version, tool policy, environment image.
A signature proves the identity signed; it proves nothing about the principal's intent or
the runtime's integrity. Every claim the system makes is scoped to what signatures
actually prove; everything beyond that is threat-modeled, not assumed.

**Enrollment.** `actor.enrolled` carries the public key, the *asserted* kind (`human`,
`agent`, `service` — an assertion by the enrolling operator, not a cryptographic fact,
and documented as such), and display name. Agent enrollment requires an identity plus a
scoped git credential; no inbound connectivity or registration server. ● Key rotation and
revocation are events; a revoked key's history remains attributed to it; revocation is
drilled (§16).

**Capabilities.** Grants are events: claim rights by squad and tier, verdict rights,
operator rights, curation-proposal rights. Admission checks grants on every verb.
Operator-only verbs refuse non-operator keys structurally. ●

**Qualification binds to the runtime, not the key.** What passes an eval is not "an
agent" — it is a **configuration tuple**: principal, harness and version, model family
and version, tool policy, environment profile. Grants cite the qualified tuple; an actor
invoking a materially different configuration than its grant cites is out of grant, and
configuration is part of what executor adapters report (§9). Spot-check evals re-test
active tuples; drifted or failing tuples get grants suspended by the supervisor — an
attributable event, no operator ceremony.

**The compromised-actor consequence.** Under the §I.2 invariant, a stolen key + credential
is assumed. What bounds the damage is: admission (no invalid transitions, no
impersonation — other keys are uncompromised), capability scoping (only its authorized
surface), budget reservation (only its reserved spend), verification (its lies are
submitted, not believed), and revocation (detection ends it). The design accepts that
the compromised actor can produce signed garbage attributed to itself — and makes that
the *ceiling* of what it can do.

### 6. Work contracts

The unit of work is a **contract** — the successor to the card, renamed because its
defining feature is its acceptance, not its state.

**Birth.** A contract is created by `intent.filed` and becomes claimable only when it
carries: intent prose (data, never instructions); an **acceptance spec** (below); tier,
budget, and routing.

**Acceptance specs are privileged code.** An acceptance spec ultimately causes command
execution on a verifier. It is therefore a protected artifact: spec authorship above the
trivial tier passes the same review gate as code; a mirror-derived intent can *propose*
acceptance content but the dispatcher's draft becomes executable only through that gate;
and the verifier executes specs in a sandbox with declared, minimal capability. No text
that arrived from outside the trust boundary becomes an executed command without a
gate between.

**Sealed checks** (above the trivial tier) — designed as a real subsystem, not a phrase:

- **Commitment**: the ledger records a commitment (hash of the canonical plaintext plus
  salt) proving the checks predate implementation.
- **Confidentiality**: the check body is encrypted to the *current verifier keyring* and
  stored in the artifact store (mutable ciphertext, immutable commitment) — so keyring
  rotation re-encrypts without touching history, and a compromised verifier key triggers
  rotation plus re-encryption, with exposure bounded to contracts open during the
  compromise window.
- **Authoring isolation**: sealed checks are authored under a grant disjoint from
  implementation grants, and the authoring channel is fenced from implementer-visible
  surfaces.
- **Honest scope**: sealed checks are **defense-in-depth against specification gaming,
  not a structural solution to it**. An implementer can still infer likely checks,
  overfit the visible spec, or exploit verifier weaknesses. The load-bearing defenses
  remain independent evidence, invariant and property-based tests, adversarial cases,
  and human review where the spec itself is incomplete; sealed checks raise the cost of
  aiming at the proxy.

**Lifecycle.** The familiar vocabulary — `backlog`, `ready`, `in_progress`, `review`,
`done`, `blocked`, `cancelled`; claim as the ready→in_progress transition, not a state —
is the projection vocabulary of the event stream, with transition rules as
self-validating data enforced at admission. ●

**Claims and the offline boundary.** `claim.taken` carries a fence; every subsequent
event on that contract from that claimant cites it; stale fences are refused at
admission. ● Claiming is **online-only**: exclusivity is a property granted at admission,
and two offline actors "claiming" the same contract have not claimed anything — they have
drafted proposals, one of which will lose *after* both did the work. The offline boundary
is therefore explicit: reading, planning, drafting, and continuing work on an
already-admitted claim are fully offline-capable ●; verbs that take exclusivity or spend
reservation are not, unless a squad explicitly opts into **racing mode** (duplicate
execution tolerated, first verified success settles — a deliberate compute-for-latency
trade, declared per squad, never an accident of connectivity).

**Liveness.** Observation-stream heartbeats with progress payloads (§3); expiry and
wedging as distinct conditions; every reap leaves a packet. ●

**Plans as falsifiable change contracts.** Above the trivial tier, claiming an unplanned
contract authorizes planning only ●; a plan names its boundary set, retention set,
validation commands for both, and expected diff shape; a plan without a retention check
does not lint. Plan PRs and implementation PRs remain structurally disjoint. ●

**Handoff packets.** The packet is the only interface between executors. Four bounded,
mechanical parts: (1) acceptance criteria; (2) settled decisions and constraints; (3)
artifact references by path — branch, plan, receipts, dirty-file inventory; (4)
**investigation findings** — the negative knowledge a successor must not rediscover:
approaches tried and why they failed, hypotheses ruled out, with pointers into durable
artifacts where they exist. Part 4 exists because "acceptance + decisions + refs" is too
lossy for debugging-heavy work: dead ends are the most expensive thing to lose. Packets
are written on every deliberate exit and every reap ●, size-bounded, shape-linted, and
their sufficiency is drilled (§16).

### 7. Dependencies, routing, and escalation

Dependencies cascade with wakeups: closing a contract unblocks dependents; plan merges
unpark plan-blocked contracts; every unblock notifies the affected party through its
adapter. Holds cascade down subtrees and suppress wakeups; initiative trees roll up
progress. Priorities and squads route; ready-queues filter by eligibility; goal-ancestry
warnings fire when open work cannot trace to a mission. ● Escalation is
`blocked(needs-you)`: an event addressed to a human gate carrying the packet, the
question, and the minimal decision — never a transcript.

### 8. The verdict pipeline

**Done is a verdict — and a reconciliation.** An implementer's strongest act is
`submission.made`. A verifier — verdict grant, key-disjoint from every implementing key
on this contract — checks out the submission in a clean per-run workspace, runs the
visible spec, unseals and runs the sealed checks, recomputes the receipt (plan hash at
merge-base, diff hash, changed-file inventory, transcripts, environment fingerprint),
and renders a signed `verdict.rendered`. ●

Because the verdict and the merge live in **two systems that share no transaction** (the
ledger and the forge), "done" is an explicit reconciliation, not a pretended atomic step:

```
verdict.rendered(pass) → merge.requested → merge.observed → done
```

Each arrow is its own event; `merge.observed` is an observation of forge fact by a
governed observer; divergence (verdict passed but merge failed; merge happened without a
verdict; force-push detected on the target) is a first-class detected state that
maintenance surfaces and reconciles — never a silent assumption. A red verdict is
unmergeable via required checks ● and locks the implementer out of self-approval paths. ●

**Independence is failure-domain separation.** Distinct keys are necessary and
insufficient: implementer and verifier can share a model, a poisoned dependency, a
prompt template, a runner image. The design defines **independence levels**, declared
per tier: L1 — distinct keys and workspaces (minimum, always); L2 — distinct runtime
tuples (different model family or provider, different harness image); L3 — deterministic
verification first (tests, assertions, diffs) with model judgment only in the residue,
plus distinct evidence-acquisition paths. High-consequence contracts require the level
their tier declares, and the verdict records which level was actually achieved.

**Qualitative residue.** Rubric items are scored item-by-item with cited evidence and
explicit uncertainty; low-confidence items route to human verdict; rubric calibration
runs against a human gold set with automatic authority suspension on drift (§16).

**Protected, with a governance root.** Verifier code, rubrics, thresholds, the sealed
keyring, and the admission rules live on the protected surface. No agent key can modify
the gates that judge its own lane — and the surface *can* be changed, by the governance
root (operator keys + owner review), through PRs, visibly. Self-modification is routed,
not denied.

### 9. The supervisor

**Offers, not assignments; pull with advisory wake.** The earlier draft said "pull,
never push" and then had the supervisor "decide assignments and wake executors" — two
different scheduling models. The precise model: the supervisor **publishes offers**
(eligibility-scoped: this contract, these tiers, these tuples, this budget reservation
available), workers **pull and claim**, and the claim settles at admission — first valid
claim wins, exactly like any claim. "Wake" is advisory transport ("offers exist for
you"), never the grant itself; a worker that never receives a wake and simply polls
loses nothing but latency. Failover and duplicate-scheduling semantics follow from this:
there is no assignment to orphan, only offers that get claimed or expire.

**Scheduling inputs**: priority, tier, grants and qualification tuples, remaining budget,
concurrency caps, and executor cost class (mechanical work to cheap tuples, hard work to
strong ones).

**Budgets are reservations, not observations.** After-the-fact metering cannot enforce a
cap: two workers can each observe $10 remaining and each spend $8. Spending verbs
therefore require an admitted `budget.reserve` (checked and decremented at admission —
the one place with a serialized view), execution runs fenced against the reservation,
and `budget.settle` (or release) records actuals from adapter metering. Where a provider
cannot be stopped synchronously, the budget is honestly a **risk limit, not a
guarantee** — declared per adapter, surfaced in the report, never fudged. Remaining
reservation is visible to the worker in its envelope, enabling budget-aware strategy;
exhaustion produces a deliberate park with packet.

**Executor adapters** implement provision (workspace + packet), wake (advisory channel),
and meter (observation-stream usage, settled to the ledger at run end), and report the
runtime tuple actually provisioned (qualification depends on it, §5). Substrates: local
worktree, container, cloud agent session, ephemeral VM, enrolled remote worker.
Disposal begins only after the last confirmed, admitted synchronization. ●

**Preemption** is graceful-first: interrupt at a safe point → park with packet; force-kill
is the fallback and still yields a reap packet.

### 10. Affordances

Every verb response is an **affordance envelope**: the result, plus the verbs currently
legal for this actor on this subject, computed from the same spec and rules admission
enforces — one rule set, two consumers (advisory computation in the envelope,
authoritative enforcement at admission), zero drift by construction. Envelopes are
versioned, schema-stable JSON with structured errors and meaningful exit codes ●;
envelopes carry the ledger position they were computed at, so a concurrent change is
detectable rather than mysterious. The CLI is the complete interface; MCP exposes the
same verbs with identical semantics ●; refusal rates are tracked (a rising rate is an
affordance gap, not agent error). Relevant promoted knowledge (lessons whose
applies-when matches) surfaces in the packet and envelope at claim time.

### 11. Lanes

Six lanes; each a role (grants + conventions), not a binary; any qualified tuple can
staff any lane it holds grants for; every lane's inputs and outputs are events. ●

1. **Dispatcher** — converts intents and mirror requests into routed contracts with
   *draft* acceptance specs (executable only after the spec gate, §6). Touches the most
   untrusted text, so it runs with least standing capability and its input handling
   passes the injection conformance suite.
2. **Planner** — falsifiable plan PRs; strongest tuples by policy; unedited-approval
   rate tracked.
3. **Implementer** — claim, heartbeat with real progress, minimal diffs, deliberate
   exits, evidence before submission. ●
4. **Verifier** — verdicts at the declared independence level; never an implementing key
   on the same contract.
5. **Curator** — the offline learning loop (§12); proposes everything, approves nothing.
6. **Maintenance** — reaps (expired and wedged), reconciles verdict/merge divergence,
   rebuilds projections, runs lints and checkpoints, files defect contracts; unattended,
   scheduled, and itself fully audited as an ordinary actor. ●

### 12. The curator and the flywheel

**Evidence first.** Online lanes append evidence and never conclusions; the write
boundary is a grant.

**A poisoning-resistant pipeline.** Trajectories are *untrusted inputs* — an attacker
who can influence what agents experience can construct trajectories designed to teach
the system something false. The pipeline is therefore staged, with distinct storage and
distinct gates between stages:

```
observations → hypotheses → validated lessons → policy
```

- **Hypotheses** are the curator's cross-trajectory comparisons: candidate lessons with
  applies-when conditions, supporting contract ids, exceptions, provenance. Minimum
  support: more than one non-failed trajectory, from more than one actor where the
  family allows it; a single accidental success is structurally non-promotable.
- **Validation** runs the hypothesis against held-out evidence and, for
  behavior-changing lessons, an adversarial evaluation: does the lesson survive
  deliberately constructed counter-trajectories? Conflicting evidence is a first-class
  state (the lesson is contested, not silently averaged).
- **Promotion** to a lesson or policy is a PR through the normal gates, routed to the
  correct carrier (knowledge doc / role or skill patch / workflow / harness change),
  each on the protected surface where behavior-changing. The curator proposes; gates
  dispose.
- **Expiry and rollback**: every promoted lesson carries a last-validated stamp and an
  expiry-for-revalidation; retirement revokes conclusions and keeps evidence; a promoted
  lesson implicated in a regression is rolled back by reverting its PR — one command,
  because it was a PR.

**Dead ends** record failure condition and environment, un-retirable when the
environment changes.

**The workflow flywheel.** Recurring trajectory shapes are detected from the ledger,
drafted as deterministic workflows, validated in mock, proposed as PRs; a failing step's
repair role completes within bounds and proposes the workflow patch as a PR — the
registry is protected, silent self-modification impossible. Conversion rate (recurring
chores → merged workflows) is a tracked metric. Every chore an agent does twice becomes
infrastructure — through gates.

### 13. Workflows and determinism

Carried forward substantially as built ●: DAGs with typed inputs, schema-enforced
produces, dependency waves, triggers, retries, on_fail policy, wall-clock budgets,
checkpoint/resume refusing mixed-graph resumes; gates for approval, review (verdict loop
with max revisions), and checks; mock mode total (zero credentials, zero side effects,
schema-valid stubs, auto-passed gates) ●; exhaustive validation refusing invalid
graphs ●; vault-indirect secrets, never frozen, never echoed. Definitions change only by
PR; the registry is protected. Statechart semantics remain rejected (expressiveness the
validators must chase), revisited only if a real workflow cannot be expressed.

### 14. Guardrails and governance

- **Tiers** gate un-planned and un-operator'd action, per-squad/per-path, checked-in. ●
- **The protected surface** is enumerated in config — transition spec, admission rules,
  guardrails, verifier code/rubrics, sealed keyring, curator gates, role definitions,
  supervisor policy — changed only by the governance root through PR + owner review, and
  write-denied to every agent key whose work it gates; a capability audit proving
  disjointness runs in CI. ●
- **Data/instruction defense-in-depth**, concretely: capability restriction first (the
  reader of untrusted content is the least-privileged actor in the system); provenance
  typing on prompt channels (trusted instructions vs. quoted content, structurally
  distinguished in prompt assembly); unforgeable delimiters on interpolated prose;
  strict-shape validation on anything interpolated into commands; sandboxing and network
  policy on executors; secret isolation (credentials never in any context window). The
  README says what this does *not* solve — a model can still be persuaded by adversarial
  text it reads — which is why capability bounds, not fencing, carry the invariant. ●
- **Budgets**: reservation-based (§9), real-spend settled, org/actor/contract
  granularity, risk-limit honesty where enforcement is impossible.
- **Per-verb policy** on the MCP/API surface: allow/deny/require-approval by actor and
  risk class; approvals as request events resolved attributably.
- **Forge-side protections as declared desired-state**, reconciled by command: branch
  protection with required checks, ledger ref writable only by admission (enforced
  modes), immutable release tags. ● Scheduled/CI identities least-privilege; ledger
  admission runs under a dedicated identity. ●
- **Boundary + retention for process changes**: any change to a role file, guardrail,
  rubric, or workflow names the failing case it fixes and demonstrates prior cases still
  pass.

### 15. Interoperability and federation

Any harness staffs any lane — the contract is files, a CLI, and envelopes ●; per-harness
config fans out from single sources, drift fails CI ●; lifecycle documentation is
generated from the transition spec (spec and docs cannot drift); design docs carry
`last-verified` stamps with a staleness lint. The core loop runs on any git remote that
can host the admission posture the team declares; forge extras (PR gates, protections,
mirrors) are adapters, with at least one non-GitHub forge supported. Cross-organization
collaboration is opaque (capability card, task lifecycle, artifacts-only exchange).
Federation is projection-only: one ledger per repo, org-level views read across ledgers,
cross-repo work enters as requests into the target ledger; there is no super-authority.
Models are registry data and scheduling inputs, never architecture. ●

### 16. Evaluation, drills, and the red team

Evaluation infrastructure is part of the template, because qualification, calibration,
and the security invariant all depend on it.

- **Eval contracts**: synthetic work with known verdicts, fixture repos, run through the
  production machinery; passing gates grants for the *tuple* that ran them; scheduled
  spot-checks re-test active tuples.
- **Verifier calibration**: scheduled sampling against a human gold set; automatic
  authority suspension on drift; defect contract filed.
- **The compromised-actor drill**: a red-team harness plays the §I.2 adversary — valid
  key, valid credential, arbitrary git and event forgery — against a fixture deployment
  in enforced mode, and asserts the invariant's ceiling: no unauthorized claim, no
  self-approval, no history rewrite, no gate modification, no impersonation, no
  over-lease, no invalid transition admitted, no spend beyond reservation. This drill is
  the architecture's definition of done, and it runs in CI.
- **Standing drills**: full projection rebuild from genesis; checkpoint verification;
  packet-resume with randomized executor kills; claim race storms converging with no
  lost updates; halt enforcement across all writer paths *including a client-bypassing
  raw-git writer* (enforced mode); key revocation with claim reaping; verdict/merge
  divergence reconciliation; ledger data-classification lint against a hostile payload
  corpus.
- **Trajectory-prefix regression** for lane behavior; **simulation mode** running the
  whole system end-to-end against synthetic intents with zero credentials.

### 17. Distribution, supply chain, migration

Template + pinned engine ●: clone-and-init adoption; engine as a SHA-256-verified pinned
release fetched by a bootstrap shim, never committed; air-gapped paths ●; upgrades with
checksum verification, protocol preflight, semver-downgrade refusal, atomic lock
rewrite ●; everything executable hash-pinned ●. The admission validator ships as part of
the engine (hook mode) and as a deployable artifact (service mode), both stateless.
Preseed: one declarative file bootstraps config, guardrails, teams, protections, and the
declared admission posture, idempotently. Migration from the current system: lossless
export → transform to genesis ledger (run-log entries become events; anchors verify
source history before conversion) → import refuses non-empty ledgers; documented,
two-command, drilled in CI against a real predecessor fixture. Install stays boring: one
command, no telemetry, no account. ●

### 18. Deliberately absent

Absence is design; proposals to add these must argue against the reason, not against
forgetfulness:

| Absent | Why |
|---|---|
| Authoritative external backends | A second writable authority is the root of every sync pathology (per the proposed §7.7's argument, made permanent here). Exporters + request adapters cover the real needs. |
| A separate mail subsystem | A message is an event; threading, unread counts, acks are projections. |
| A separate anchors mechanism | The signed hash chain is the tamper evidence; signed checkpoints are the recovery points. |
| A separate run log / audit log | The ledger is both. |
| A machine-local card backend | The SQLite projection provides the throughput without a second authority. |
| Client-assigned global sequence numbers | Ordering authority is admitted ancestry; ordinals are derived, not asserted. |
| High-frequency ephemera on the authoritative ref | Liveness and fine metering are observation streams, summarized into ledger facts at material transitions. |
| Terminal-multiplexer coupling | Wake channels are adapter details; no coordination feature assumes any of them. |
| Statechart workflow semantics | DAG + gates covers observed use; chased expressiveness is a cost. |
| Model-vendor coupling anywhere | Models are scheduling inputs and qualification-tuple fields, never architecture. |
| A hosted control plane with unique state | The repository is the control plane. The admission validator is trusted but stateless — everything it knows rebuilds from a clone. |

---

## Part III — Acceptance Criteria

A criterion is met only when it is implemented, tested, documented, and **enforceable**
(by admission, lint, CI, drill, or protocol — not by convention alone). Criteria marked ●
exist in the current system at least partially and must be preserved; all others are
targets. This section replaces AC.md when SEED-NEXT becomes the build.

### A. The Ledger

- [ ] Every coordination fact is an event on a single per-repo ledger ref; no second
      writable store of coordination state exists anywhere in the system.
- [ ] Events carry timestamp, actor fingerprint, verb, subject, schema-valid payload,
      previous-event hash, and a signature over the canonical form; the entire chain
      verifies from genesis with one command.
- [ ] Ordering authority is admitted ancestry; no writer asserts a global sequence
      number; ordinals used by projections and citations are derived at admission or
      projection time.
- [ ] Any mutation of admitted history — reorder, rewrite, deletion, forgery — is
      detected by chain verification and, in enforced modes, refused at the remote;
      the drill proves both with corrupted fixtures and a raw-git adversary.
- [ ] Payload data classification is enforced at admission: coordination facts and
      references only; content bodies (transcripts, attachments, user prose beyond
      short structured fields) live in an addressed artifact store with a documented
      erasure path, referenced by hash; the hostile-payload lint corpus passes.
- [ ] Erasure obligations are honorable: erasing a referenced artifact never breaks
      chain verification, and the erasure is itself an attributable event.
- [ ] Checkpoints are signed by maintenance/operator identities, embed the projection
      state hash, and specify their materialization format; a fresh clone's verification
      obligations (full replay once, or explicit trust in the signer set) are documented
      and the choice is declared, not defaulted; replay-from-checkpoint equals
      replay-from-genesis in CI.
- [ ] A genesis event names the governance root and protocol version; version mismatch
      refuses with a distinct exit code. ●
- [ ] `halt.declared` stops admission of everything except an operator's `halt.lifted`,
      enforced at the admission boundary; the halt drill includes a client-bypassing
      writer in enforced mode. ●
- [ ] Reads, drafts, planning, and work on admitted claims are offline-capable ●;
      exclusivity-taking and reservation-spending verbs are online-only and documented
      as such.
- [ ] Ledger performance is budgeted and tracked in CI: admission latency, replay time,
      projection rebuild time, against a representative history.

### B. The Admission Boundary

- [ ] A stateless admission validator exists and is the only writer of the ledger ref in
      enforced modes; it validates signature, capability, fence, transition, schema,
      data classification, protocol version, halt state, and budget reservation before
      atomic admission, and refuses with structured, attributable rejections.
- [ ] The validator holds no unique durable state: a replacement instance rebuilds
      entirely from a clone, proven by a kill-and-replace drill.
- [ ] Three admission postures are implemented and declared per deployment: enforced
      self-hosted (receive hook), enforced forge-hosted (sole-writer admission identity
      behind forge protections), cooperative (self-validation only).
- [ ] Cooperative mode is a declared stance with named consequences: doctor and README
      state plainly that the §I.2 invariant does not hold there; no deployment lands in
      cooperative mode by default or by accident.
- [ ] In enforced modes, actor git credentials cannot write the ledger ref directly —
      verified by an attempted direct push in the drill.
- [ ] The compromised-actor drill passes in CI: a valid-key, valid-credential adversary
      with raw git access cannot claim unauthorized work, approve itself, rewrite
      history, alter gates, impersonate another actor, exceed its lease, exceed its
      reservation, or admit an invalid transition.
- [ ] Admission may shard proposal intake without changing semantics; ordering remains
      solely the admitted chain.

### C. Streams and the hot path

- [ ] Traffic is classified: coordination facts are admitted events; liveness, progress,
      and fine-grained metering ride non-authoritative observation channels.
- [ ] Heartbeats-with-progress flow on observation channels at configured frequency with
      near-zero coordination cost; the ledger records only material transitions (claim,
      coarse milestones, wedge declaration, run settlement, park/reap).
- [ ] Expiry (no observations) and wedging (observations without progress change) are
      distinct, visible conditions with distinct reap heuristics, drilled.
- [ ] A contention benchmark demonstrates the target scale (hundreds of concurrent
      actors) without unrelated writes racing each other's admissions pathologically;
      the benchmark is tracked in CI.
- [ ] Observation channels are lossy by declaration: nothing durable depends on them;
      losing every observation stream loses no coordination state.

### D. Projections

- [ ] Every read surface is a deterministic function of a ledger prefix (plus declared
      observation inputs), stamped with its build position, and rebuildable
      byte-identically with one command.
- [ ] No code path writes a projection directly or treats one as authoritative; the
      write-boundary lint enforces it.
- [ ] Staleness is visible everywhere projected state is shown; consumers can demand a
      minimum position.
- [ ] The SQLite cache projection matches the current machine-local backend's read
      throughput with zero authority (mid-operation deletion loses nothing).
- [ ] External mirrors are one-way exporters; mirror-side edits arrive only as request
      events from governed identities, validated at admission; an
      exporter/request-adapter conformance suite passes per integration.
- [ ] Bidirectional sync is structurally impossible: no component holds both an export
      path and a coordination write path.
- [ ] External facts (merges, CI results) enter only as observations by governed
      observers; nothing in the system treats an observation as control.
- [ ] The rebuild-everything-from-genesis drill runs green in CI.

### E. Identity, threat model, qualification

- [ ] Identity, credential, principal, and runtime are distinct concepts in the design
      docs, the schemas, and the threat model; no claim in user-facing docs exceeds what
      signatures prove.
- [ ] Every actor is a keypair; enrollment, grants, suspension, revocation are events;
      the keyring is a projection; admission verifies signatures against it on every
      proposal.
- [ ] Enrolled kind (human/agent/service) is documented as an operator assertion;
      nothing security-relevant assumes kind is cryptographically proven.
- [ ] Agent enrollment requires exactly an identity plus a scoped credential; no inbound
      connectivity or registration server. ●
- [ ] Grants are capability data checked at admission; out-of-grant verbs are refused
      structurally; operators-only verbs refuse non-operator keys; no agent key can
      approve its own work into done — by key disjointness. ●
- [ ] Qualification binds to the runtime tuple (principal, harness+version, model
      family+version, tool policy, environment profile); grants cite tuples; adapters
      report the provisioned tuple; materially drifted tuples are out of grant.
- [ ] Scheduled spot-check evals re-test active tuples; failures suspend grants
      attributably without operator ceremony.
- [ ] Key rotation and revocation are drilled: revocation ends the key's standing, reaps
      its claims, preserves its history's attribution, and (for verifier keys) triggers
      sealed-check keyring rotation.
- [ ] Humans and machines are distinguished in the roster; agent-only guardrails and
      human/agent metrics read the distinction. ●

### F. Work contracts and lifecycle

- [ ] A contract cannot leave draft without an acceptance spec; the spec's executable
      content passes a review gate before it can run anywhere; specs are protected
      artifacts.
- [ ] Text originating outside the trust boundary (mirrors, intents) can propose but
      never directly become executable acceptance content; the gate between is
      structural and tested against a hostile corpus.
- [ ] Above the trivial tier, contracts carry sealed checks: ledger commitment (salted
      hash) proving pre-existence; ciphertext encrypted to the current verifier keyring
      in the artifact store; keyring rotation re-encrypts without touching history;
      authoring grants disjoint from implementation grants; a capability audit proves
      no implementer path can decrypt.
- [ ] Sealed-check documentation states their honest scope: defense-in-depth against
      specification gaming, not a structural solution; the load-bearing defenses
      (independent evidence, invariants, property/adversarial tests, human review)
      are named where sealed checks are described.
- [ ] The lifecycle vocabulary and transition rules are self-validating data enforced at
      admission; claim is a transition, not a state. ●
- [ ] Claims are exclusive with fencing, granted only at admission (online); stale
      fences refuse with distinct codes; contention returns structured envelopes. ●
- [ ] Racing mode (duplicate execution, first verified success settles) exists only as
      an explicit per-squad opt-in with its compute cost stated; offline "claiming" is
      impossible by construction, not by convention.
- [ ] Every exit from in_progress is deliberate; every involuntary exit leaves a packet;
      silent abandonment is impossible. ●
- [ ] Packets contain exactly four bounded parts — acceptance criteria; settled
      decisions; artifact references by path; investigation findings (approaches tried,
      failures and why, hypotheses ruled out) — shape-linted and size-bounded. ●
- [ ] Packet sufficiency is drilled: a fresh executor completes a killed executor's
      contract from the packet alone, including *not* re-trying recorded dead ends
      (asserted by the drill).
- [ ] Plans are falsifiable: boundary set, retention set, validation commands for both,
      expected diff shape; missing retention fails lint; plan and implementation PRs
      remain structurally disjoint. ●
- [ ] Dependencies cascade with wakeups; holds cascade with suppression; initiative
      rollups render; goal ancestry warns. ●

### G. Verdicts and evidence

- [ ] "Done" is reachable only through the reconciliation chain
      `verdict.rendered(pass) → merge.requested → merge.observed → done`; each step is
      its own event; there is no code path that collapses them.
- [ ] Verdict/merge divergence (verdict without merge, merge without verdict, target
      force-push) is a detected, surfaced, reconciled state — drilled by inducing each
      divergence in CI.
- [ ] Verdicts are signed by verdict-granted keys provably disjoint from every
      implementing key on the contract; operator override is its own attributable verb,
      never a disguised verdict.
- [ ] The verifier executes in clean per-run isolation; parallel verdicts never collide;
      cleanup fires pass or fail; the verifier's verdict inputs are enumerable and
      exclusively self-executed or self-read (no implementer-claims channel).
- [ ] Receipts bind contract id, plan hash at merge-base, diff hash, changed-file
      inventory, visible and sealed check transcripts, and environment fingerprint;
      verification recomputes everything from the submission head and fails on
      mismatch. ●
- [ ] Independence levels L1–L3 are defined, declared per tier, enforced at verdict
      time, and recorded in the verdict; high-consequence tiers require runtime-tuple
      separation (L2) or deterministic-first verification (L3).
- [ ] A red verdict is unmergeable and locks the implementer out of self-approval until
      a new submission. ●
- [ ] Rubric verdicts score per-item with cited evidence and explicit uncertainty;
      low-confidence items route to human verdict; calibration runs against a human
      gold set with automatic authority suspension on drift.
- [ ] Verifier code, rubrics, thresholds, sealed keyring, and admission rules are on the
      protected surface; the governance root and its change process are named in config;
      the capability audit proves agent-key disjointness in CI.
- [ ] Evidence, receipts, and verdicts are queryable by contract, actor, time, and
      outcome.

### H. Supervisor and execution

- [ ] The scheduling model is offers-and-claims: the supervisor publishes
      eligibility-scoped offers; workers pull and claim; claims settle at admission;
      wake is advisory transport whose total failure costs only latency — proven by a
      wakeless (poll-only) CI run. ●
- [ ] Offers expire; there are no assignments to orphan; duplicate scheduling is
      impossible because exclusivity settles at admission.
- [ ] Spending verbs require an admitted `budget.reserve`; execution is fenced to the
      reservation; `budget.settle`/release records actuals; the two-workers-see-$10
      over-spend scenario is structurally impossible for reservable resources — drilled.
- [ ] Where a substrate cannot stop spend synchronously, the budget is documented and
      surfaced as a risk limit, not a guarantee, per adapter.
- [ ] Remaining reservation is visible in the worker's envelope; exhaustion produces a
      deliberate park with packet, never a vanished worker.
- [ ] Executor adapters implement provision/wake/meter, report the provisioned runtime
      tuple, and exist for at least: local worktree, container, cloud session, enrolled
      remote worker; the adapter interface is public.
- [ ] Metering flows on observation channels and settles to the ledger at run end;
      no execution path is unmetered.
- [ ] Disposability after last admitted synchronization is drilled with randomized
      kills; the loss window (work since last sync) is bounded by policy (sync
      frequency) and stated honestly in docs.
- [ ] Preemption is graceful-first (safe-point park with packet); force-kill still
      yields a reap packet.
- [ ] Scheduling inputs include cost class and qualification tuples; routing is data.

### I. Affordances and the actor interface

- [ ] Every verb response is a versioned, schema-stable envelope with structured errors
      and meaningful exit codes ●, includes the verbs currently legal for this actor on
      this subject, and carries the ledger position it was computed at.
- [ ] The affordance computation and admission enforcement consume the same rule set;
      an affordance-listed verb refused at admission for legality (absent a concurrent
      event, detectable via the position stamp) is a bug class with a regression test.
- [ ] The CLI is the complete interface; MCP exposes identical semantics ●; Windows
      parity is documented and tested. ●
- [ ] Refusal rates are tracked as an affordance-gap metric.
- [ ] Matching promoted lessons surface in packets and envelopes at claim time.

### J. Lanes

- [ ] Role definitions exist for all six lanes as grants + conventions, composable from
      ordered fragments, resolved and checked by validation. ●
- [ ] The dispatcher runs with least standing capability and passes the injection
      conformance suite (hostile corpus: embedded instructions in intents, mirrors, and
      tool output are quoted as data, never obeyed). ●
- [ ] Dispatcher re-triage rate and planner unedited-approval rate are tracked; the
      planner lane receives the strongest tuples by policy.
- [ ] Maintenance runs green unattended — reaps, divergence reconciliation, projection
      rebuilds, lints, checkpoints — and is audited as an ordinary actor. ●
- [ ] Escalations carry packet + question + minimal decision; waiting escalations
      surface with age; resolution latency is tracked.
- [ ] Small-team mode (one actor, all grants) and fleet mode (disjoint actors per lane)
      both run the full loop in CI.

### K. Curation, memory, and the flywheel

- [ ] Online lanes append evidence only; conclusion-writing is grant-gated to the
      curator's proposal path.
- [ ] The pipeline is staged with distinct storage and gates: observations → hypotheses
      → validated lessons → policy; no stage skips.
- [ ] Promotion requires: applies-when conditions; support from >1 non-failed
      trajectory (and >1 actor where the family allows); provenance links;
      last-validated stamp; and — for behavior-changing lessons — survival of an
      adversarial evaluation against constructed counter-trajectories.
- [ ] Trajectories are treated as untrusted inputs; the poisoning drill (attacker
      constructs trajectories to teach a false lesson) fails to achieve promotion in CI.
- [ ] Conflicting evidence is a first-class contested state, never silently averaged;
      contested lessons do not surface in envelopes.
- [ ] Lessons expire for revalidation; retirement revokes conclusions and keeps
      evidence; a lesson implicated in a regression rolls back by reverting its PR.
- [ ] Dead ends carry failure condition and environment and are un-retirable on
      environment change; the curator checks dead-end applicability.
- [ ] The flywheel closes through gates: recurring shapes → drafted workflows → mock
      validation → PR; repair roles propose patches as PRs; conversion rate is tracked.
- [ ] Knowledge bloat is managed: dedup with provenance, staleness flags, structure
      lint.

### L. Guardrails and governance

- [ ] Tiers gate un-planned/un-operator'd action per-squad and per-path. ●
- [ ] The protected surface is enumerated in config, includes the admission rules, is
      changed only by the named governance root via PR + owner review, and is
      write-denied to every agent key it gates — capability audit in CI. ●
- [ ] Data/instruction defense is layered and documented with its limits: least
      capability for untrusted-content readers (primary), provenance-typed prompt
      channels, unforgeable delimiters, strict-shape command interpolation, sandboxing,
      network policy, secret isolation; the hostile corpus passes on every release. ●
- [ ] Per-verb policy (allow/deny/require-approval by actor and risk class) governs the
      MCP/API surface with attributable approvals.
- [ ] Forge protections are declared desired-state and reconciled: required checks,
      admission-only ledger writes (enforced modes), immutable tags. ● Scheduled/CI
      identities are least-privilege. ●
- [ ] Process changes pass boundary + retention, lint-checked for declared sets.

### M. Workflows

- [ ] DAG engine with typed inputs, schema-enforced produces, waves, triggers, retries,
      on_fail, budgets, and safe resume ●; approval/review/check gates ●; total mock
      mode ●; exhaustive validation ●; vault-indirect secrets; PR-only definition
      changes on a protected registry.

### N. Interoperability and federation

- [ ] Harness fan-outs from single sources with drift-failing CI ●; lifecycle docs
      generated from the transition spec; `last-verified` stamps with staleness lint.
- [ ] The core loop runs on any git remote supporting the declared admission posture;
      at least one non-GitHub forge is supported by adapters.
- [ ] Cross-org collaboration is opaque (capability cards, task lifecycle,
      artifacts-only).
- [ ] Federation is projection-only: uniform read remotes, request-event ingress,
      no cross-ledger write path — proven by its absence.
- [ ] No worker-contract dependency on any model vendor. ●

### O. Evaluation infrastructure

- [ ] Eval contracts run through production machinery against fixture repos and gate
      tuple qualification; spot-checks run scheduled.
- [ ] Verifier calibration runs scheduled with automatic authority suspension.
- [ ] The compromised-actor drill (§16) passes in CI on every release — this is the
      architecture's own definition of done.
- [ ] Standing drills run in CI: projection rebuild, checkpoint verification,
      packet-resume with dead-end assertions, claim race storms, halt (including raw-git
      bypass), key revocation with keyring rotation, verdict/merge divergence,
      data-classification hostile corpus, budget reservation races, curator poisoning.
- [ ] Trajectory-prefix regression covers lane decision points; simulation mode runs
      the whole system credential-free end-to-end.

### P. Distribution, supply chain, migration

- [ ] Clone-and-init adoption from tagged releases; three-way template upgrades with
      rollback ●; pinned SHA-256-verified engine, never committed, air-gap paths ●;
      checksum/protocol/downgrade-safe engine upgrades ●; everything executable
      hash-pinned ●.
- [ ] The admission validator ships in hook and service form, both stateless, both
      rebuildable from a clone.
- [ ] Preseed bootstraps config, guardrails, teams, protections, and the declared
      admission posture in one idempotent, CI-verified file.
- [ ] Migration from the predecessor is a drilled two-command path: lossless export →
      genesis import (refusing non-empty ledgers), source history verified before
      conversion, against a real predecessor fixture in CI.
- [ ] Install is boring: one command, no telemetry, no account. ●

### Q. Quality, docs, community

- [ ] One fast backpressure command stays green on main. ●
- [ ] Engine coverage above 90%; template scripts and hooks smoke-tested; conformance
      suites per exporter/adapter. ●
- [ ] Docs governed: operator handbook, generated worker docs, stamped design docs. ●
- [ ] Research corpus maintained; §18's absences are standing rejections proposals must
      argue against. ●
- [ ] Decisions recorded and binding; authority order stated. ●
- [ ] MIT, no CLA, no open-core split, stated explicitly. ●
- [ ] Dogfooding total: the system coordinates its own development. ●

### R. The autonomy end-state

- [ ] One-sentence intents become routed contracts whose draft acceptance specs survive
      human review unedited in the majority of cases.
- [ ] Planner plan-PRs pass human review >80% unedited; implementers reach
      verdict-passed submissions unassisted on the happy path.
- [ ] The verifier lane holds quality alone on low tiers; humans review only high-tier
      plans, the protected surface, and escalations.
- [ ] Every escalation is one packet + one question + one decision; transcript-dumping
      is a defect.
- [ ] The system runs unattended for a week on a real backlog with zero chain
      violations, zero lost updates, zero silent abandonments, zero guardrail breaches,
      zero unreserved spend — and the ledger alone reconstructs and justifies
      everything.
- [ ] The flywheel demonstrably compounds over a quarter (chore→workflow conversion,
      packet-resume success, cost per contract), from ledger metrics alone.
- [ ] A team that has never spoken to the authors adopts from the README in under an
      hour, on their forge, with their declared admission posture, and reaches their
      first agent-shipped, verifier-passed, human-reviewed PR the same day. ●
