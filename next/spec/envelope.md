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
  human message) and a distinct exit code. These codes are also what the
  attempts journal records per refused attempt for the report's
  refusal-rate metric ([`refusals.md`](refusals.md)); the envelope
  schema itself is unchanged by that metric.
- `position` is the ledger position the response was computed at, so a
  concurrent change is detectable rather than mysterious (charter §II.10).
  Every response that reached the ledger stamps it; a refusal raised before a
  tip was ever read (a malformed invocation, an unreadable ledger directory)
  carries `null` rather than inventing a position, because a fabricated stamp
  would break exactly the detection this field exists for. The attempts journal
  turns on the same distinction: an unstamped response is not an
  admission-boundary attempt ([`refusals.md`](refusals.md)).
- `affordances` lists the verbs currently legal **for this actor on this
  subject**, computed by drafting one signed probe per catalog verb and
  running the same rule set admission enforces (one rule set, two
  consumers, zero exceptions; `admit.Affordances`, landed in Phase 8.1
  with its lifecycle-walk and catalog-completeness drills; item 2's
  regression-class harness generalizes them). Every append-path response
  stamps the list for its signing actor and subject at the stamped
  position, refusals included: the refusal plus what IS legal is the
  envelope's point, and the loop verbs
  ([`loop-verbs.md`](loop-verbs.md)) are where that point does the
  most work, pre-flighting an act through the same rule set and
  answering "not that" and "then what may I do?" in one envelope. A
  computation failure degrades to the empty
  list, never to a failed verb. Responses lacking a ledger, signing
  key, or subject (e.g. `version`, `doctor`, keyless read surfaces:
  probes must be signed, so a fingerprint alone cannot compute) carry
  the empty list. One named carve-out: `actor.enrolled` is listed only
  where the prober could supply the subject's public key, which no
  fingerprint-holder can derive — the enrollment surface knows its key
  out of band.
- `budget` is the reservation block, derived from the shared budget view
  (`admit.BudgetBlock`): `reserved` sums the open valid reservations,
  `remaining` is the derived remaining, both decimal strings; `null` on
  subjects carrying no budget facts. `seed budget status` accepts
  `--key` to stamp affordances beside the block it already reports.
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
| 16 | `plan_required` | the submission names a contract above the trivial tier with no approved plan (or no cited plan anchor): claiming an unplanned contract authorizes planning only, so go plan — a merged plan PR observed as `plan.approved`, and the submission citing the plan anchor it implements (`plans.md`) |
| 17 | `not_independent` | the verdict signer is an implementing key on this contract (a claimant, past or present, or the bound submission's signer): L1 independence is failure-domain separation checked at admission, and holding a verdict grant does not cure being disqualified on the contract being judged — distinct from `out_of_grant`, which is capability absence (`verdicts.md`) |
| 18 | `ungated` | the verifier refuses to execute declared-executable acceptance content whose gate evidence is absent: gate-before-run, the half `acceptance.md` assigns to verdicts — nothing runs, no verdict is rendered (`verdicts.md`) |
| 19 | `spec_unrunnable` | declared-executable acceptance content yields no parseable commands: the declaration promised runnable content and the body carries none, an inconsistency only run time can see — silence must never decide, so the run refuses rather than passing vacuously (`verdicts.md`) |
| 20 | `checks_red` | rendering `pass` refused: the verifier derives the permissible verdict from the transcripts it just executed, and at least one visible check exited nonzero — `fail` stays renderable, and the message names the failing command (`verdicts.md`) |
| 21 | `receipt_mismatch` | recomputation from the submission head does not reproduce a cited receipt digest: the stored receipt, the range, or the repository's content has diverged from what the verdict attested — the recompute-and-mismatch refusal, and 6.2 reconciliation input; the message names both digests (`verdicts.md`) |
| 22 | `seal_broken` | a sealed-checks envelope does not open: missing ciphertext, a commitment that does not match the body, or an envelope carrying no checks. The commitment is the contract, so a body that fails it is not a weaker seal but a different document (`sealed-checks.md`) |
| 23 | `not_recipient` | the identity opening a sealed envelope is outside its recipient set, which rotation lag produces routinely: the sealer encrypted to a key set this actor is no longer (or not yet) in, and re-sealing to the current set is the fix, never a bypass (`sealed-checks.md`) |
| 24 | `unsealed` | an above-trivial contract reaches the verifier with no sealed-checks commitment: contracts above the trivial tier carry sealed checks, sealed before the first claim, and render refuses rather than judging against criteria the implementer could read (`sealed-checks.md`) |
| 25 | `red_locked` | rendering `pass` over a submission an authenticated `fail` already judged: a red verdict locks pass out until a NEW submission (`contract.returned`, re-claim, resubmit). Fail restatements stay admissible, and only boundary-validated fails lock (`verdicts.md`) |
| 26 | `lane_invalid` | a checked-in lane manifest makes a claim the tables refuse: a grant outside the vocabulary, an act whose accepted capabilities the lane does not hold, a liveness source that is not a work step, a missing fragment. Distinct from `posture_invalid`, which judges a deployment's posture declaration rather than a role definition (`lanes.md`) |
| 27 | `budget_exhausted` | a reservation asks for more than the class has left: capacity is checked and decremented at admission, so exhaustion is a first-class, EXPECTED, recoverable condition rather than chain trouble, and the worker loop's exhaustion park carries this code into its packet for the next worker to read. Narrow by design: the budget rule's other refusals (malformed payload, wrong signer, unknown class, double close, laundering) stay `chain_invalid`, because a caller that answers this code by asking for less must not also be answering a bug (`budgets.md`) |
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
