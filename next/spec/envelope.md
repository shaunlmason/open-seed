# envelope.md — the affordance envelope and exit codes

> Status: v0, normative for `next/**`. Authority: [`SEED-NEXT.md`](../../SEED-NEXT.md)
> Part II §10 and Appendix B; [`docs/next-build-plan.md`](../../docs/next-build-plan.md)
> §1 fixed defaults (exit-code continuity with v1). Envelope schema version:
> **`seed-envelope/0`**, carried in every response's `v` field and bumped by
> the same discipline as the protocol version (a field addition, removal, or
> rename is a schema change).

## Shape

Every verb response is exactly one JSON line:

```json
{
  "v": "seed-envelope/0",
  "ok": true,
  "result": { "…": "…" },
  "error": null,
  "position": "<ledger position computed at, or null>",
  "affordances": ["verb", "…"],
  "budget": { "reserved": "…", "remaining": "…" },
  "exit": 0
}
```

- `result` and `error` are mutually exclusive: a refusal replaces `result`
  with `error` (`{code, message}`: a stable machine-branchable code and a
  human message) and a distinct exit code.
- `position` is the ledger position the response was computed at, so a
  concurrent change is detectable rather than mysterious (charter §II.10).
  It is `null` until the ledger lands (Phase 1); from then on every response
  stamps it.
- `affordances` lists the verbs currently legal **for this actor on this
  subject**, computed from the same rule set admission enforces (one rule
  set, two consumers; the property test lands in Phase 8). Subjectless verbs
  (e.g. `version`) carry an empty list.
- `budget` is the reservation block (`null` until budgets land, Phase 7).
- `exit` duplicates the process exit code so machine callers can branch on
  the envelope alone.

## Exit codes

The build-plan fixed default: **reuse v1 conventions where semantics match**,
for tooling continuity. Inherited allocations:

| code | name | meaning |
|---|---|---|
| 0 | `ok` | verb succeeded |
| 2 | `contention` | claim contention or author lockout: exclusivity not granted — a rival `claim.taken` on a held contract returns the holding fingerprint and the active fence position (the loser learns who holds and since when), and an exclusive verb drafted offline refuses here too: claiming is online-only (`lifecycle.md`) |
| 3 | `invalid_transition` | transition absent from the spec tables, or verb illegal in this state — the contract-lifecycle refusals included (`lifecycle.md`): an illegal (state, verb) pair, a birth verb on an existing subject, a non-birth verb on an unknown one, each naming subject, current state, and verb |
| 4 | `not_found` | subject does not resolve |
| 5 | `unavailable` | authoritative remote or backend unreachable |
| 6 | `fenced_out` | stale or missing fence (claim token): on a held contract the deliberate exits and every event from the holder or a prior claimant must cite the active fence as `{"fence": "<position>"}`; the refusal names the cited fence, the active fence, and the holder (`lifecycle.md`) |
| 7 | `halted` | admission is halted (`system.halt.declared`); only an operator's `system.halt.lifted` may append |
| 8 | `chain_invalid` | ledger verification failed (parse, linkage, signature, actor, or HEAD trouble); the error message carries `position N: <reason>: <detail>` |
| 9 | `classification_refused` | payload failed the data-classification lint; the error message joins the violations' pointers and rules |
| 10 | `version_mismatch` | protocol or envelope version mismatch |
| 11 | `remote_rejected` | the remote's own admission refused the push (e.g. a `pre-receive` policy hook declined); the error message carries the remote's reason verbatim |
| 12 | `head_regression` | the remote serves a chain that regresses this client's persisted verified head (rollback or vanished ref): a freshness refusal, distinct from failed chain verification |
| 13 | `posture_invalid` | the deployment's posture declaration is malformed or names an unknown posture; every deployment MUST declare exactly one of the three charter postures (SEED-NEXT.md Part II "Postures"), and the message names the valid ones — distinct from `not_found`, which covers a deployment with no declaration at all |
| 14 | `out_of_grant` | the actor holds none of the capabilities the verb accepts (charter Part II "Capabilities"): a structural authorization refusal, distinct from a signature that fails to verify (`chain_invalid`). The accepted-capability sets per verb are the normative table in `actors.md`; governance roots hold `operator` implicitly, and the message names the accepted set |
| 15 | `stale` | a projection consumer demanded a minimum build position (`--min-position`) and the resolved build's stamp sits below it (charter III.D: staleness is visible and demandable); the message names the stamped and demanded positions. A freshness judgment on a successfully resolved view — distinct from `not_found` (no published build at all) and from `head_regression` (the authoritative chain itself went backwards) |
| 64 | `usage` | CLI usage error (EX_USAGE); never a verb result |
| 66 | `unreadable` | an input the invocation names exists but cannot be opened or read (EX_NOINPUT: a directory, denied permissions, an I/O failure): an operational failure in the usage class, distinct from a judgment on the content (`posture_invalid`) and from a missing declaration (`not_found`) |

**Allocation rule for new codes**: a new condition takes the lowest unused
integer in 7–63, lands as a PR editing this table before any code emits it,
and is mirrored by a constant in `next/internal/envelope`. Codes 64 and
above stay reserved for CLI-usage-class errors. Distinct refusals the
charter names (halt, out-of-grant, classification refusal, …) get their own
codes when the refusing rule lands; nothing shares a code with a different
meaning.

## Deferred: structured error data

Error stays exactly `{code, message}` in `seed-envelope/0`. Rich failure
data (reason and position fields, per-violation pointer lists) is rendered
deterministically into `message`; promoting it to structured fields is a
schema change that lands only with a versioned envelope bump, when a
machine consumer needs it.

## Conformance mapping

- III.I: "Every verb response is a versioned, schema-stable envelope with
  structured errors and meaningful exit codes" — enforced for every verb the
  CLI grows by `next/internal/envelope` (schema-stability test pins the
  serialized field set) and the CLI tests in `next/cmd/seed`.
- III.I "affordance computation and admission enforcement consume the same
  rule set" — Phase 8, against `internal/admit`.
