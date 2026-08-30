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
| 2 | `contention` | claim contention or author lockout: exclusivity not granted |
| 3 | `invalid_transition` | transition absent from the spec tables, or verb illegal in this state |
| 4 | `not_found` | subject does not resolve |
| 5 | `unavailable` | authoritative remote or backend unreachable |
| 6 | `fenced_out` | stale or missing fence (claim token) |
| 7 | `halted` | admission is halted (`system.halt.declared`); only an operator's `system.halt.lifted` may append |
| 8 | `chain_invalid` | ledger verification failed (parse, linkage, signature, actor, or HEAD trouble); the error message carries `position N: <reason>: <detail>` |
| 9 | `classification_refused` | payload failed the data-classification lint; the error message joins the violations' pointers and rules |
| 10 | `version_mismatch` | protocol or envelope version mismatch |
| 64 | `usage` | CLI usage error (EX_USAGE); never a verb result |

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
