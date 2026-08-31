# actors.md — actor events and the keyring projection

> Status: v0, normative for `next/**` from protocol version **`seed/1`**.
> Authority: [`SEED-NEXT.md`](../../SEED-NEXT.md) Part II "Enrollment" and
> "Capabilities"; [`docs/next-build-plan.md`](../../docs/next-build-plan.md)
> Phase 3 item 1; plan `plans/os-52a2d688.md`. The verb catalog lives in
> [`protocol.md`](protocol.md); this file owns the payload schemas and
> standing semantics, implemented in one transition function
> (`next/internal/keyring`) consumed by both verification and admission.

## Activation: the `seed/1` boundary

Everything below activates for records whose position is under protocol
version `seed/1` (reached via `system.protocol.upgraded`). Records at
`seed/0` positions are **grandfathered**: `actor.*` events there are
inert — no keyring effect, no payload judgment — so every chain that
verified before `seed/1` existed still verifies (the bump discipline in
`protocol.md`: these semantics are exactly a validation-rule change a
`seed/0` validator would judge differently). New `actor.*` proposals at a
`seed/0` tip refuse as a verb illegal in this state (exit 3) until the
deployment upgrades.

## The keyring

The keyring is a **projection**: seeded from the genesis payload's
`governance_root` (those keys are roots, standing active) and advanced by
the events below, never stored. From `seed/1`, signature resolution is
standing-aware at every position: a key resolves only while its standing
is active, so a key signs only between its enrollment (or genesis) and
its suspension or revocation — and records signed before a standing
change keep verifying, which is what keeps a revoked key's history
attributed to it.

## Payload schemas (strict: unknown fields refuse)

- **`actor.enrolled`** — subject: the enrolled key's fingerprint.
  Payload `{"key": "<64-char hex of the raw 32-byte Ed25519 public
  key>", "kind": "human" | "agent" | "service", "name": "<non-empty
  display name>"}`. The subject MUST equal the fingerprint of `key`.
  `kind` is **an assertion by the enrolling operator, not a
  cryptographic fact** (SEED-NEXT.md Part II), and nothing
  security-relevant may assume otherwise.
- **`actor.granted`** — subject: an enrolled fingerprint. Payload
  `{"capability": "<non-empty>"}`. Grants accumulate as capability data;
  admission checks them per verb from Phase 3.2.
- **`actor.suspended`** / **`actor.revoked`** — subject: an enrolled
  fingerprint. Payload `{"reason": "<non-empty>"}`.
- **`actor.qualified`** — cataloged, undefined until qualification lands
  (build plan Phase 10); a `seed/1` chain refuses it.

## Standing transitions

| event | precondition | effect |
|---|---|---|
| `enrolled` (new key) | fingerprint unknown | standing active |
| `enrolled` (suspended key) | standing suspended | reinstated: standing active, kind/name updated |
| `enrolled` (active key) | — | **refuses** (already enrolled) |
| `enrolled`/`suspended`/`granted` (revoked key) | — | **refuses**: revocation is terminal |
| `granted` | subject enrolled, not revoked | capability appended |
| `suspended` | subject active | standing suspended |
| `revoked` | subject not already revoked | standing revoked |

**Root liveness.** Suspending or revoking a governance root refuses when
it would leave zero active roots: no admitted transition may leave the
deployment without a key admission accepts `actor.*` from. Root rotation
beyond that guard is genesis-level governance, outside these events.

## Capabilities

Grants are events (`actor.granted`) checked at admission on every verb
(SEED-NEXT.md Part II "Capabilities"). The normative vocabulary maps
each governed verb to the **set of capabilities any one of which
admits** (mirrored by `internal/keyring.AcceptedCapabilities`, pinned by
test); a verb with no row needs active standing only. Governance roots
hold `operator` implicitly — the genesis trust anchor a deployment's
first grants must come from. Only active standing counts: a suspended
or revoked actor holds nothing, and grant-level withdrawal short of
ending standing is deferred until the catalog grows a verb for it.

| verb | accepted capabilities |
|---|---|
| `system.halt.declared` | `operator` |
| `system.halt.lifted` | `operator` (the charter: only an operator's lift may append) |
| `system.protocol.upgraded` | `operator` |
| `system.checkpoint` | `maintenance`, `operator` (the charter names checkpoints as signed by the maintenance actor or an operator) |
| `actor.*` (every lifecycle verb) | `operator` |
| `intent.filed` | `dispatch`, `operator` |
| `contract.specified` | `dispatch`, `operator` |
| `contract.blocked` | `dispatch`, `operator` |
| `contract.unblocked` | `dispatch`, `operator` |
| `claim.reaped` | `dispatch`, `operator` (reaping is queue management, not worker self-service) |
| `claim.taken` | `claim`, `operator` |
| `claim.released` | `claim`, `operator` |
| `claim.parked` | `claim`, `operator` |
| `submission.made` | `claim`, `operator` |
| `contract.cancelled` | `operator` (until a real need appears) |
| `plan.proposed` | `claim`, `operator` (the claim holder plans; the fence matrix applies) |
| `plan.approved` | `operator` (an external-fact observation, the merge.observed posture) |
| `progress.milestone` | `claim`, `operator` (the claim lane's coarse summarization fact; the fence matrix applies) |
| `wedge.declared` | `operator` (operator judgment in v0; the maintenance lane inherits it later) |
| `merge.observed` | `operator` (Phase 6 adds the observer lane) |
| `verdict.rendered` | `verdict` (deliberately no operator fallback: III.G names operator override its own attributable verb, never a disguised verdict — that verb is 6.4's; a governance root that judges holds an explicit verdict grant, and L1 independence applies to every signer, verdicts.md) |

A signer holding none of a verb's accepted capabilities refuses at exit
14 `out_of_grant` (`envelope.md`), the message naming the accepted set.
Later phases append rows (claim rights by squad and tier, verdict
rights, curation-proposal rights) when their verbs land.

**Authorization is admission policy, not chain validity**: like the
halt gate and payload classification, verification tolerates it in
history — one of the cooperative posture's named consequences — so the
vocabulary evolves without protocol bumps (the versioning stance
recorded in plans/os-3979d48b.md). Transition legality and payload
shapes above, by contrast, are chain validity: an event violating them
fails verification at its position (`bad_actor_event`), whichever
posture admitted it.

## Conformance mapping

- III.E "every actor is a keypair; enrollment, grants, suspension,
  revocation are events; the keyring is a projection; admission verifies
  signatures on every proposal" — `internal/keyring` + the verification
  and admission wiring, drilled in `internal/ledger`, `internal/admit`,
  `cmd/seed-admit`, and `cmd/seed` tests.
- III.E "enrolled kind is documented as an operator assertion" — this
  file plus the package doc.
- III.E revocation drill (standing ends; history stays attributed) —
  Phase 3.3 turns the verification tests here into the charter drill.
