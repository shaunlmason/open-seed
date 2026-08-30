# protocol.md — Seed wire protocol, version `seed/0`

> Status: v0, normative for `next/**`. Authority: [`SEED-NEXT.md`](../../SEED-NEXT.md)
> Part II §1 (the ledger) and Appendix B (wire-level sketch);
> [`docs/next-build-plan.md`](../../docs/next-build-plan.md) §1 fixed defaults.
> This file is the versioned canonical-form spec the charter's Appendix B
> defers to; the genesis event names the protocol version that binds it.
> Envelope shape and exit codes live in [`envelope.md`](envelope.md).

## Protocol version

- The protocol version is the string `seed/0`.
- Genesis names it; **every event carries it** in its `v` field.
- A validator or client meeting an event (or a ledger) whose version differs
  from the one it implements refuses with the distinct exit code for version
  mismatch (see `envelope.md`); it never guesses.
- **Bump discipline**: `seed/N` increments on any change to the canonical
  form, the hash or signature algorithms, verb semantics, or validation rules
  that a conformant `seed/N-1` validator would judge differently. A bump
  lands as a PR editing this file plus a `system.protocol.upgraded` event;
  admission refuses mixed-version appends. Additive verb-catalog growth that
  older validators safely refuse as unknown does not bump the version.

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

- **Signature**: `sig` is the Ed25519 signature over the canonical (JCS)
  bytes of the event **including `prev`**. `sig` is carried alongside the
  canonical form (it is not part of the signed bytes).
- **Chain hash**: the SHA-256 of the same canonical bytes, lowercase hex.
  The next event's `prev` cites it.
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
  public key. Display surfaces may shorten it; events always carry the full
  fingerprint.

## Ordering

Order is **admitted ancestry**: the chain of `prev` hashes on the
authoritative ledger ref. No writer asserts a global sequence number; where
an ordinal is useful (projections, citations) it is derived at admission or
during projection, never proposed. (Charter §II.1; conformance III.A.)

## Storage reference (non-normative)

The reference deployment stores the ledger on the git ref `refs/seed/ledger`
as JSONL segments, one file per UTC day under `ledger/segments/`, with a
`HEAD` record carrying the tip hash; the artifact store rides
`refs/seed/artifacts` (git-addressed) with a filesystem fallback. These are
build-plan fixed defaults for the reference implementation, not protocol
requirements; any storage satisfying the canonical form, ordering, and
admission rules conforms.

## Verb namespace catalog

Copied from charter Appendix B; payload schemas land with the phases that
implement them (schema files will sit beside this spec and be linted at
admission).

- `system.*` — `genesis`, `halt.declared`, `halt.lifted`, `checkpoint`,
  `protocol.upgraded`.
- `actor.*` — `enrolled`, `granted`, `suspended`, `revoked`, `qualified`
  (cites eval results and the runtime tuple).
- `intent.*` / `contract.*` — `intent.filed`, `contract.specified`
  (acceptance spec gate passed; sealed commitment), `contract.blocked`,
  `contract.cancelled` ….
- `claim.*` — `taken` (carries fence), `released`, `parked`, `reaped`
  (packet ref), `wedge.declared`.
- `plan.*` — `proposed`, `approved` (observation of the plan PR merge).
- `progress.*` — `milestone` (coarse; bounded frequency).
- `submission.*` — `made` (branch, evidence refs).
- `verdict.*` — `rendered` (pass/fail, receipt, independence level achieved).
- `merge.*` / `check.*` — `requested`, `observed` (external-fact
  observations).
- `budget.*` — `reserve`, `settle`, `release`.
- `run.*` — `settled` (aggregate metering).
- `message.*` — `sent`, `acked`.
- `request.*` — inbound proposals from projection surfaces (mirror edits,
  dashboard actions).
- `curation.*` — `hypothesis.proposed`, `lesson.promoted` (PR observation),
  `lesson.retired`, `deadend.recorded`.

## Data classification (summary)

Payloads carry **coordination facts and references, not content bodies**:
hashes and paths for artifacts, receipts, transcripts, and any prose beyond
short structured fields. Bulk or expirable content lives in the artifact
store with an erasure path, referenced by hash; erasing a referenced
artifact never breaks chain verification. The payload lint that enforces
this at admission lands in Phase 1 with a hostile fixture corpus
(conformance III.A).
