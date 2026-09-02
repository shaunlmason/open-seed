# protocol.md — Seed wire protocol, version `seed/0`

> Status: v0, normative for `next/**`. Authority: [`SEED-NEXT.md`](../../SEED-NEXT.md)
> Part II §1 (the ledger) and Appendix B (wire-level sketch);
> [`docs/next-build-plan.md`](../../docs/next-build-plan.md) §1 fixed defaults.
> This file is the versioned canonical-form spec the charter's Appendix B
> defers to; the genesis event names the protocol version that binds it.
> Envelope shape and exit codes live in [`envelope.md`](envelope.md).

## Protocol version

- The protocol version is the string `seed/0`.
- Genesis names it; **every event carries the version active at its
  position** in its `v` field.
- **Active version**: the version a chain runs at, starting as the version
  genesis names and changing only at a `system.protocol.upgraded` event.
  The upgrade event is the last event of the old version: it carries the
  old version in `v` and names the new version in its payload; every later
  event carries the new version.
- **Verification across history**: an implementation declares the set of
  versions it supports. Replay accepts a chain when every event's `v`
  equals the version active at that event's position and every active
  version is in the supported set. The version-mismatch refusal (exit 10,
  see `envelope.md`) is reserved for an event whose `v` differs from the
  active-at-position version, or an active version the implementation does
  not support; a valid older prefix under a supported older version always
  verifies. It never guesses.
- **Admission**: proposals must carry the current active version; anything
  else refuses with the same distinct code.
- **Bump discipline**: `seed/N` increments on any change to the canonical
  form, the hash or signature algorithms, verb semantics, or validation rules
  that a conformant `seed/N-1` validator would judge differently. A bump
  lands as a PR editing this file plus the `system.protocol.upgraded` event.
  Additive verb-catalog growth that older validators safely refuse as
  unknown does not bump the version.
- **Version register**:
  - `seed/0` — the genesis default: chain form, halt, upgrade schemas,
    payload classification.
  - `seed/1` — activates the actor keyring semantics
    ([`actors.md`](actors.md)): `actor.*` payload schemas, standing
    transitions (root liveness included), and standing-aware signature
    resolution. A `seed/0` validator imposes no actor schema, so it
    judges these records differently — hence the bump; `actor.*` records
    at `seed/0` positions stay inert and grandfathered, and a new
    `actor.*` proposal at a `seed/0` tip refuses until the deployment
    upgrades.
  - `seed/2` — activates the runtime-tuple semantics
    ([`qualification.md`](qualification.md)): `actor.granted` may cite
    a `tuple`, `run.started` must declare one, an offer may scope by
    `tuples`, and drift refuses as `out_of_grant`. A `seed/1` validator
    strictly decodes `actor.granted` as `{capability}` and so fails a
    tuple-bearing grant at its position, which is exactly a validation
    rule the two would judge differently — hence the bump. At `seed/1`
    positions the field stays unknown-and-refused under a `seed/2`
    validator too, so every existing chain verifies byte for byte; a
    `seed/1`-only validator refuses an upgraded chain at the first
    `seed/2` record by version, never by misjudging the grant.
  - `seed/3` — activates the eval semantics ([`evals.md`](evals.md)):
    `intent.filed` may carry the `eval` marker, and `actor.qualified`
    and `actor.disqualified` are defined. Actor payload shapes are
    chain validity ([`actors.md`](actors.md)), so a `seed/2` validator's
    unknown-verb arm fails a chain carrying either verb as
    `bad_actor_event` at its position while this validator accepts it:
    the two judge the same record differently, hence the bump, which
    makes a `seed/2`-only validator refuse an upgraded chain at the
    first `seed/3` record by version rather than as corruption. At
    `seed/2` positions both verbs stay unknown-and-refused under a
    `seed/3` validator too, and the marker is neither read by the fold
    nor admitted at the boundary, so every existing chain verifies byte
    for byte.
  - `seed/4` — activates the independence levels
    ([`verdicts.md`](verdicts.md), "Independence levels"):
    `verdict.rendered`'s `independence` widens from the literal `L1`
    to the ordered vocabulary `L1`, `L2`, `L3`, the verdict may declare
    the verifier's `tuple`, and the tier table's `independence` column
    is enforced at the verdict boundary and reapplied along the merge
    chain. A `seed/3` validator's strict verdict decode refuses the
    `tuple` field and any literal but `L1`, so the two judge a `seed/4`
    verdict differently, hence the bump, which makes a `seed/3`-only
    validator refuse an upgraded chain at the first `seed/4` record by
    version rather than by misjudging a level. At `seed/3` positions
    the literal `L1` alone admits under a `seed/4` validator too, and
    the eval semantics of `seed/3` hold at `seed/4` alike (the gate is
    a named list), so every existing chain verifies byte for byte.
    Two later additions ride the same entry (plans/os-6bd9ffff.md D4,
    D5, D7; [`trajectories.md`](trajectories.md)): `contract.specified`
    gains the `ready` origin, the dispatcher's re-specification of an
    unclaimed contract ([`lifecycle.md`](lifecycle.md)), which a
    `seed/3` validator's table refuses as an illegal transition where
    this one applies it; and `plan.proposed` and `plan.approved`
    carry the plan's content `digest` ([`plans.md`](plans.md)), which
    a `seed/3` validator's shape has no field for. Each refuses at a
    `seed/3` position by version, naming `seed/4`, and the fold reads
    each at `seed/4` positions only, so every existing chain verifies
    byte for byte.

## Canonical event form

Events are JSON objects canonicalized per **RFC 8785 (JCS)**. The canonical
bytes of an event are the JCS serialization of exactly these fields:

| field | type | meaning |
|---|---|---|
| `v` | string | protocol version (`seed/0`) |
| `ts` | string | RFC 3339 UTC timestamp; recorded for humans, **never an ordering authority** |
| `actor` | string | the acting identity's key fingerprint (below) |
| `verb` | string | `namespace.verb` from the catalog below |
| `subject` | string | contract id, actor id, or `system` |
| `payload` | object | verb-specific, schema-validated; coordination facts and references only (data classification, charter §II.1) |
| `prev` | string | chain hash of the preceding event's canonical form |

- **Signature**: `sig` is the Ed25519 signature (RFC 8032) over the
  canonical (JCS) bytes of the event **including `prev`**, encoded as
  **lowercase hex** (64 bytes, 128 hex characters). `sig` is carried
  alongside the canonical form (it is not part of the signed bytes).
- **Chain hash**: the SHA-256 of the same canonical bytes, encoded as
  **lowercase hex** (32 bytes, 64 hex characters). The next event's `prev`
  cites it.
- **Ledger record**: what storage and transport carry is the wrapper
  `{"event": <event object>, "sig": "<hex>"}`. Canonicalization and hashing
  apply to the inner `event` object alone, never the wrapper, so wrapper
  field order and whitespace are irrelevant to identity. Verifiers MUST
  recompute the JCS bytes from the parsed event; they never trust stored
  byte sequences.
- **Strict parsing (one accepted wire form)**: parsers MUST refuse a record
  carrying unknown fields in the wrapper or the event object, a duplicate
  key at any level (payload included), or trailing data after the record.
  Otherwise a record could carry correctly signed core fields plus unsigned
  material that survives in storage while escaping canonicalization, schema
  validation, and the classification lint; and duplicate keys let two
  parsers disagree about the same bytes.
- **Encodings, uniformly**: every hash, fingerprint, and signature in this
  protocol is lowercase hex with no prefix; uppercase hex is refused, not
  normalized, so exactly one encoding of a value is accepted on the wire.
  Base64 and multibase forms are not used anywhere.
- **Genesis**: the first event's `prev` is the **empty hash**: the SHA-256 of
  zero bytes, `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`.
  Genesis (`system.genesis`) names the initial governance root (operator
  keys) and the protocol version.

## Algorithms

- **Hash**: SHA-256 everywhere (chain, commitments, receipts), lowercase hex
  encoding.
- **Signatures**: Ed25519. OpenSSH `ed25519` public/private keys are accepted
  at key load so operators reuse existing keys; internally a key is the raw
  32-byte public key.
- **Actor fingerprint**: the lowercase hex SHA-256 of the raw 32-byte Ed25519
  public key (64 hex characters). Display surfaces may shorten it; events
  always carry the full fingerprint. The OpenSSH display form
  (`SHA256:<base64>`) is **not** used in events or projections: OpenSSH
  acceptance is a key-loading affordance only (the loader parses the OpenSSH
  ed25519 key format down to the raw 32-byte public key, and fingerprints
  that).

## Ordering

Order is **admitted ancestry**: the chain of `prev` hashes on the
authoritative ledger ref. No writer asserts a global sequence number; where
an ordinal is useful (projections, citations) it is derived at admission or
during projection, never proposed. (Charter §II.1; conformance III.A.)

## Storage reference (non-normative)

The reference deployment stores the ledger on the git ref `refs/seed/ledger`
as JSONL segments, one file per UTC day under `ledger/segments/`, each line
one ledger record (`{"event": …, "sig": …}`), with a `HEAD` record carrying
the tip hash; the artifact store rides `refs/seed/artifacts` (git-addressed)
with a filesystem fallback. These are
build-plan fixed defaults for the reference implementation, not protocol
requirements; any storage satisfying the canonical form, ordering, and
admission rules conforms. The client's private transport git dir under
its state directory carries auto-gc disabled, written on every open
rather than only at init, so a state dir an older build created is
hardened the first time a newer one opens it and no repository the
engine creates for itself is mutated by a collector after the process
that armed it has exited.

## Verb namespace catalog

Copied from charter Appendix B; payload schemas land with the phases that
implement them (schema files will sit beside this spec and be linted at
admission).

- `system.*` — `genesis`, `halt.declared`, `halt.lifted`, `checkpoint`,
  `protocol.upgraded`.
- `actor.*` — `enrolled`, `granted`, `suspended`, `revoked`, `qualified`
  (cites eval results and the runtime tuple) and its inverse
  `disqualified`, both from `seed/3` ([`evals.md`](evals.md)). Payload
  schemas and standing semantics: [`actors.md`](actors.md), active from
  `seed/1`.
- `intent.*` / `contract.*` — `intent.filed` (from `seed/3` optionally
  carrying the `eval` marker, [`evals.md`](evals.md)),
  `contract.specified` (acceptance spec gate passed; sealed
  commitment), `contract.blocked`, `contract.cancelled` ….
- `claim.*` — `taken` (carries fence), `released`, `parked`, `reaped`
  (packet ref), `wedge.declared`.
- `plan.*` — `proposed`, `approved` (observation of the plan PR merge).
- `progress.*` — `milestone` (coarse; bounded frequency).
- `submission.*` — `made` (branch, evidence refs).
- `verdict.*` — `rendered` (pass/fail, receipt, independence level
  achieved: `L1` alone before `seed/4`, `L1`/`L2`/`L3` with the
  verifier's declared tuple from it, [`verdicts.md`](verdicts.md)).
- `merge.*` / `check.*` — `requested`, `observed` (external-fact
  observations).
- `offer.*` — `published` (the supervisor's eligibility-scoped,
  expiring invitation to claim; a fact, never a transition —
  [`offers.md`](offers.md), active from `seed/1`. Catalog growth here
  is additive: older validators refuse the unknown verb safely, so
  the protocol version does not bump).
- `budget.*` — `reserve`, `settle`, `release`.
- `run.*` — `started` (the spending gate; declares the runtime tuple
  from `seed/2`, [`qualification.md`](qualification.md)), `settled`
  (aggregate metering), `interrupted`.
- `message.*` — `sent`, `acked`.
- `request.*` — inbound proposals from projection surfaces (mirror edits,
  dashboard actions).
- `curation.*` — `deadend.recorded` (the holder's candidate
  observation, on the contract), `hypothesis.proposed` (the curator's,
  on the subject its claim and exceptions derive),
  `hypothesis.contested` (the curator's held-out counter-evidence, on
  the hypothesis), `lesson.promoted` (the PR observation, on the
  hypothesis, citing the adversarial evaluation it survived),
  `lesson.retired` (the observation that a promotion is revoked, on the
  hypothesis, by regression, supersession or expiry; the evidence
  stays), `deadend.retired` and `deadend.unretired` (the curator's
  judgment that a dead end's environment moved, on the contract), all
  seven from [`curation.md`](curation.md) as additive catalog growth,
  active from `seed/1`.

## Data classification (summary)

Payloads carry **coordination facts and references, not content bodies**:
hashes and paths for artifacts, receipts, transcripts, and any prose beyond
short structured fields. Bulk or expirable content lives in the artifact
store with an erasure path, referenced by hash; erasing a referenced
artifact never breaks chain verification. The payload lint that enforces
this at admission lands in Phase 1 with a hostile fixture corpus
(conformance III.A).
