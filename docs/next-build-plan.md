# next-build-plan.md — Implementing Seed (SEED-NEXT)

> **Scope and authority.** This plan governs sequencing and per-phase acceptance for
> implementing **Seed**, the system chartered in [`SEED-NEXT.md`](../SEED-NEXT.md). For everything under `next/**`, the charter is the design authority and this
> plan is the build order — the same relationship `docs/design-options.md` and
> `docs/build-plan.md` have for the v1 template. The charter's Part II is normative;
> its Part III is the conformance checklist every phase's exit criteria cite. Where this
> plan sets a default the charter leaves open, the default is binding until a recorded
> decision changes it.
>
> **Autonomy contract.** This plan exists so agents can implement Seed **without human
> intervention**. Every foreseeable decision has a default below. When you hit one that
> doesn't: (1) prefer the charter's normative text; (2) prefer the smallest reversible
> choice consistent with it; (3) record the choice in `decisions/` inside your task PR
> and continue. Escalate to a human (card → `blocked(needs-you)`) **only** for:
> amendments to the charter itself, changes to the protected surface outside `next/**`,
> renaming Seed, publishing/spin-out, or granting new credentials. Everything else you
> decide.

## 0. Ground rules

- **Where work happens.** All Seed implementation lives under `next/` in this
  repository. Nothing under `next/**` may modify v1 surfaces (`.seed/**`,
  `scripts/seed*`, the root template files) except the explicitly listed integration
  points: the `Makefile` (`check-next` target, Phase 0) and this docs tree. The v1
  template keeps working, untouched, throughout.
- **How work is coordinated.** Dogfood the existing loop: file cards via
  `scripts/seed task` (one card per work item below, `next:` title prefix), claim
  before working, plan-first above L1 (`plans/<task-id>.md` via plan PR), implement in
  a worktree on `seed/<task-id>`, attach evidence, `make check` green, move to review.
  The rules in AGENTS.md apply unchanged.
- **Language and layout.** Go (matches the existing engine and its toolchain), one
  module at `next/`: `next/cmd/seed/` (CLI), `next/internal/...` (packages),
  `next/spec/` (protocol schemas + versioned canonical-form spec), `next/fixtures/`
  (drill fixtures), `next/docs/` (implementation notes). Binary name: `seed` — the
  successor claims the name. During incubation it is built into `next/bin/` and invoked
  explicitly (`go run ./next/cmd/seed`, or `next/bin/seed`), never installed on PATH:
  `scripts/seed` (v1) remains the only coordination entry point for doing the work
  until spin-out.
- **Quality bar.** Table-driven unit tests per package; every charter drill lands as an
  automated test the phase it becomes possible; ≥90% statement coverage for
  `next/internal/...` from Phase 1 on; `make check` stays green on main at every merge.
  Conformance mapping: each test that satisfies a Part III criterion names it in a
  comment (`// conformance: III.A — ordering is admitted ancestry`).
- **Commit hygiene.** Small scoped commits; no model identifiers in any committed
  artifact; PR bodies state which plan item and which conformance criteria the change
  advances.

## 1. Fixed defaults (binding until a recorded decision changes them)

| Decision | Default | Rationale |
|---|---|---|
| Signature scheme | Ed25519; accept OpenSSH `ed25519` keys so operators reuse existing keys | Ubiquitous, small, fast |
| Hash | SHA-256 everywhere (chain, commitments, receipts) | Boring, universal |
| Canonical event form | JSON per RFC 8785 (JCS); `sig` over the JCS bytes of the event including `prev` | Deterministic without a custom codec |
| Protocol version | `seed/0` in genesis; bump discipline recorded in `next/spec/` | Versioned from day one |
| Ledger ref | `refs/seed/ledger`; artifact store at `refs/seed/artifacts` (git-addressed) with filesystem fallback | Charter Reference deployment |
| Segments | JSONL, one file per UTC day under `ledger/segments/`, `HEAD` record with tip hash | Simple, streamable |
| Sealed-check encryption | `filippo.io/age` (X25519 recipients = verifier keyring) | Audited, minimal |
| Observation channel | v0: per-executor file under `next/var/obs/` (gitignored) + optional `refs/seed/obs/<actor>` push; supervisor tails both | Lossy-by-declaration is the contract, so simplest channel first |
| SQLite cache | `modernc.org/sqlite` (no cgo) | Portable builds |
| Admission postures | Cooperative validation library first (same rule set), then enforced self-hosted (`pre-receive` hook binary) in Phase 2; forge-hosted service deferred to Phase 12 | Shared-rule-set requirement drives the order |
| Racing mode | Deferred entirely to post-plan backlog | Opt-in feature, not core |
| Envelope exit codes | Reuse v1 conventions where semantics match (2 contention, 6 fence, 10 version); new codes documented in `next/spec/envelope.md` | Continuity for tooling |

## 2. Phases

Each phase is one or more cards. **Exit criteria are the phase's acceptance test** —
cite them as evidence when moving cards to review. A phase may start when its
dependencies' exit criteria are merged, not before. Within a phase, items are ordered.

### Phase 0 — Workspace and spec skeleton  *(deps: none)*

1. Scaffold `next/` module, CLI skeleton (`seed version`), CI wiring: `make check-next`
   (build + vet + test + coverage gate) and hook it into `make check`.
2. Write `next/spec/protocol.md` v0: canonical form (JCS), event fields, hash/signature
   algorithms, verb namespace catalog copied from charter Appendix B, envelope shape,
   exit codes.
3. Create `next/docs/decisions.md` (implementation decision log — one line per default
   exercised or overridden, linking the PR).

*Exit:* `make check` green with `check-next` included; spec doc exists and matches
charter Appendix B; a decision log exists.

### Phase 1 — Ledger core  *(deps: 0)*

1. Event model + JCS canonicalization + Ed25519 sign/verify (`internal/event`).
2. Chain: append, verify-from-genesis, `prev` linkage; segment storage; `HEAD`.
3. Genesis (`seed init` writes signed genesis naming governance-root keys + `seed/0`).
4. Push-race append loop against a git remote: fetch → re-validate → re-link → push;
   losing events that fail re-validation are reported, never silently re-appended.
5. Halt semantics in the validation rule set (`halt.declared`/`halt.lifted`).
6. Payload data classification lint (references-not-bodies) with a hostile fixture
   corpus.
7. CLI: `seed ledger verify`, `seed ledger append` (dev tool), `seed ledger show`.

*Exit:* charter III.A items — chain verifies from genesis in one command; corrupted
fixtures (reordered, rewritten, forged-sig, bad-prev) all detected; classification
corpus passes; race drill (two concurrent appenders, no lost updates) green.

### Phase 2 — Admission  *(deps: 1)*

1. Validation rule set as a pure library (`internal/admit`): signature vs. keyring,
   capability, fence, transition, schema, classification, protocol version, halt,
   reservation — one rule set, importable by client, hook, and (later) service.
2. Cooperative posture: client self-validates via the library before pushing.
3. Enforced self-hosted posture: `seed-admit` `pre-receive` hook binary refusing
   invalid pushes, force-updates, and deletions of the ledger ref; stateless
   (rebuilds context from the repo it guards).
4. Posture declaration in config; `seed doctor` states the posture and, for
   cooperative, prints the named consequence verbatim.
5. Drills: raw-git adversary (direct push of invalid events) refused under enforced;
   kill-and-replace the hook host, revalidate.

*Exit:* the III.B subset implementable at this point — validator exists and is the
ledger ref's sole writer under enforced posture; statelessness (kill-and-replace
drill); posture declaration with cooperative consequences printed; direct-push refusal
drill; both postures selectable in fixtures — with the rule set covering the checks
that exist so far (signature, schema, classification, protocol version, halt).
Capability rules slot in at Phase 3, fences at Phase 5, budget reservations at
Phase 7, and the full III.B including the compromised-actor drill closes at Phase 12;
the rule-set library is built so those land as added rules, not rework.

### Phase 3 — Identity and grants  *(deps: 2)*

1. `actor.enrolled` / `granted` / `suspended` / `revoked`; keyring projection.
2. Admission checks grants per verb; operator-only verbs; kind documented as
   assertion.
3. Key rotation/revocation drill: revoke → standing ends → history stays attributed.

*Exit:* charter III.E items except qualification tuples (Phase 10) — signature +
grant checks live in admission; revocation drill green.

### Phase 4 — Projections  *(deps: 1; grants integration after 3)*

1. Projection engine: deterministic build from prefix, position stamp, one-command
   rebuild (`seed project rebuild`).
2. Standard projections: contract detail, ready queue (eligibility-filtered after
   Phase 5), actor view, report skeleton.
3. SQLite cache projection; mid-operation deletion drill.
4. Write-boundary lint: nothing outside the projection engine writes projection dirs.

*Exit:* charter III.D core — byte-identical rebuild drill; stamps everywhere; lint
wired into `check-next`.

### Phase 5 — Contracts, claims, packets  *(deps: 3, 4)*

1. Transition table as data (`next/spec/transitions.json`), self-validating, loaded by
   the admission rule set; lifecycle verbs.
2. Claims with fences; online-only claiming; stale-fence refusal; contention envelope.
3. Four-part packets (acceptance criteria; verified/asserted decisions; commit-anchored
   refs incl. diff-vs-merge-base range; investigation findings) — schema, size bound,
   lint; written on release/park/reap.
4. Acceptance-spec field on contracts; the spec-gate flag (executable content requires
   a gate event before any run) — enforcement lands with verdicts (Phase 6).
5. Plans: falsifiable-plan lint (boundary set, retention set, commands, diff shape);
   plan-gating rule above the trivial tier.
6. Observation streams v0 + monotonic progress counts; expiry vs. wedge detection in
   the report.

*Exit:* charter III.F items except sealed checks — claim race storm drill; packet
lint + a packet-resume fixture drill (fresh actor completes from packet alone,
asserting recorded dead ends are not re-tried); offline-boundary tests (exclusivity
verbs refuse without remote).

### Phase 6 — Verdict pipeline  *(deps: 5)*

1. `submission.made`; verifier workspace (clean per-run checkout); receipt computation
   (plan hash at merge-base, diff hash, inventory, transcripts, environment
   fingerprint); `verdict.rendered` signed by verdict-granted, implementer-disjoint
   key (L1 independence).
2. Reconciliation chain: `merge.requested`, `merge.observed` (observer records forge
   fact), `done`; divergence detection in maintenance; induced-divergence drills.
3. Sealed checks: commitment event (salted SHA-256), age-encrypted body in the
   artifact store, keyring rotation re-encryption, authoring-grant disjointness;
   capability audit test proving no implementer path decrypts.
4. Red-verdict lockout; operator override as its own verb.

*Exit:* charter III.G minus L2/L3 levels and rubric calibration (Phase 10/11) — the
reconciliation drills, receipt recompute-and-mismatch test, sealed-check audit.

### Phase 7 — Supervisor, offers, budgets  *(deps: 5)*

1. Offers (`offer.published`, eligibility-scoped, expiring); workers pull and claim;
   wakeless poll-only test proving wake is advisory.
2. `budget.reserve` / `settle` / `release`; admission decrements reservations;
   reservation race drill (concurrent over-spend impossible); risk-limit declaration
   per adapter.
3. Executor adapter interface (provision / wake / meter / report-tuple); **local
   worktree adapter** first; metering to observation stream, `run.settled` aggregate.
4. Graceful preemption: safe-point interrupt → park with packet; force path → reap
   packet.

*Exit:* charter III.H items for the implemented adapter — poll-only run, reservation
race drill, disposability drill (randomized kill after sync; complete elsewhere).

### Phase 8 — Affordance envelopes  *(deps: 5)*

1. Envelope with `affordances` (computed from the same `internal/admit` rule set),
   `position`, budget block, structured errors, exit codes per spec.
2. Regression test class: affordance-listed verb refused for legality at the same
   position = bug.
3. Refusal-rate metric in the report.

*Exit:* charter III.I — same-rule-set property test; envelope schema stable and
versioned.

### Phase 9 — Lanes, escalation, maintenance  *(deps: 6, 7, 8)*

1. Role definitions (grants + conventions) for the six lanes as fragments; dispatcher
   least-capability posture; injection conformance suite (hostile corpus) against
   dispatcher input handling. The worker lane's role definition includes its loop
   (poll, claim, work, meter, sync, deliberate exit), and exhaustion parking is part
   of that loop: a budget refusal at a spending gate triggers the `claim.parked` exit
   with packet (the III.H row the Phase 7 exit routes here), consuming Phase 8's
   envelope budget block.
2. Escalation (`blocked(needs-you)`) with packet + question + decision; report
   surfaces age.
3. Maintenance loop: reap expired/wedged, reconcile divergence, rebuild projections,
   checkpoint (signed), lints — runnable unattended; audited as an ordinary actor.
   Lints include unsettled-run detection: a claim window carrying an admitted
   `run.started` whose `run.settled` is still missing once the subject has taken a
   subsequent claim window or reached a terminal state is flagged (the Phase 7 exit's
   metering-detection obligation; post-close settlement is a valid intermediate
   state, so the condition is position-anchored, never mid park/reap flow).
4. Small-team mode and fleet mode end-to-end fixtures.

*Exit:* charter III.J — both modes run the full loop in CI; injection corpus green;
maintenance runs unattended in the fixture.

### Phase 10 — Qualification and evaluation  *(deps: 9)*

1. Runtime tuples in enrollment/grants; adapters report provisioned tuple; drift =
   out-of-grant.
2. Eval contracts against fixture repos through the production machinery; grants cite
   passing tuples; scheduled spot-checks; suspension on failure.
3. Independence levels L2/L3 declared per tier and recorded in verdicts.
4. Rubric verdicts (per-item, evidence-cited, uncertainty-marked); calibration harness
   against a gold set with authority suspension on drift.
5. Trajectory-prefix regression harness for lane decision points.

*Exit:* charter III.E (tuples) + III.G (levels, calibration) + III.O eval items.

### Phase 11 — Curator and flywheel  *(deps: 9)*

1. Staged pipeline stores (observations → hypotheses → validated → policy) with
   grant-gated boundaries; workers append candidates only.
2. Promotion gate: ≥2-trajectory support, applies-when, provenance, last-validated;
   adversarial evaluation for behavior-changing lessons; contested state. The
   applies-when carries its delivery moment: a promoted lesson whose applies-when
   matches the claimed subject surfaces in the claim packet and in the response
   envelope at claim time (the charter III.I row the Phase 8 exit routes here —
   knowledge nobody is shown at the right moment is knowledge that does not exist).
3. Poisoning drill: constructed trajectories fail to achieve promotion.
4. Expiry, retirement (evidence kept), rollback-by-revert.
5. Flywheel v0: recurring-shape detection from the ledger → drafted workflow → mock
   validation → proposal PR.

*Exit:* charter III.K — poisoning drill green; a real recurring chore in the fixture
converts to a workflow through the gates.

### Phase 12 — Hardening, distribution, migration  *(deps: all)*

1. **Compromised-actor drill**: red-team harness with valid key + credential + raw git
   against an enforced fixture; asserts the §I.2 ceiling item by item. This is the
   release gate — it runs in CI from here on.
2. Forge-hosted admission service (stateless, sole-writer pattern) + protections
   desired-state reconciler.
3. Checkpoint trust docs + replay-equals-genesis CI proof; performance budgets
   (admission latency, replay, rebuild) tracked.
4. Preseed file (config, guardrails, teams, protections, posture) — idempotent init.
5. **Migration**: `seed import --from-open-seed <export>` — v1 lossless export → verify
   anchors → transform (cards→contracts, run-log→events, receipts→verdict records,
   mail→messages) → genesis import refusing non-empty ledgers; drilled against a real
   v1 fixture.
6. Docs generation: lifecycle prose from `transitions.json`; operator handbook;
   simulation mode (credential-free end-to-end).

*Exit:* charter III.B (service posture), III.O (compromised-actor drill in CI),
III.P complete; the fixture organization runs a week-long simulated backlog
(accelerated clock) meeting III.R's zero-violation bar.

### Phase 13 — Conformance completion  *(deps: 12)*

The items Part III requires that Phases 0–12 deliberately defer. **Full Part III
conformance is claimable only after this phase**; Phases 0–12 deliver core conformance
(every pillar's mechanisms at the enforced self-hosted posture with the local-worktree
adapter), and the doctor reports which Phase 13 criteria remain open until then.

1. Racing mode as the per-squad opt-in with first-verified settlement (III.F).
2. Remaining executor adapters: container, cloud agent session, enrolled remote
   worker (III.H).
3. Non-primary forge adapter for the forge extras (III.N).
4. Federation read remotes and cross-repo request ingress (III.N).
5. A2A-shaped cross-organization boundary (III.N).
6. Machine-protocol surface exposing the CLI's verbs with identical semantics, and
   platform parity (including Windows) documented and tested (III.I — the row the
   Phase 8 exit routes here; the CLI stays the complete interface, so this adds a
   second surface over the same rule set, never a second semantics). The surface
   ships **with** the per-verb policy that governs it — allow / deny /
   require-approval by actor and risk class, with approvals attributable to the
   identity that gave them (III.L's machine-protocol row, scheduled nowhere else in
   this plan; a protocol surface exposed without its policy would widen reach while
   leaving a required control unbuilt).

*Exit:* the named III.F/III.H/III.I/III.N criteria plus III.L's machine-protocol
policy row green; the conformance report shows Part III complete at the enforced
self-hosted posture.

## 3. Backlog (true extras, not conformance-blocking)

Dashboard tiers beyond the report (no Part III criterion requires them) and sharded
admission intake (III.B says MAY). File cards when Phase 13 is exhausted.

## 4. Progress tracking

Maintain `next/docs/progress.md`: one line per plan item — `phase.item — card id —
PR — state`. Update it in every task PR touching `next/**`. It is the single place a
fresh agent reads to find the frontier; keep it truthful before starting new work.
