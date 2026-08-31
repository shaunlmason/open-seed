# reconciliation.md — done is a verdict, and a reconciliation

> Status: v0, normative for `next/**`. Authority: [`SEED-NEXT.md`](../../SEED-NEXT.md)
> Part II §8 (the verdict and the merge live in two systems that share
> no transaction, so "done" is an explicit reconciliation, never a
> pretended atomic step) and conformance III.G rows 1–2;
> [`docs/next-build-plan.md`](../../docs/next-build-plan.md) Phase 6
> item 2; plan `plans/os-6cdc15be.md`. Implemented by the chain checks
> in `internal/transition`/`internal/admit`, the classifier in
> `internal/reconcile`, and `seed reconcile`. Red-verdict lockout is
> 6.4; sealed checks 6.3; the unattended loop that runs these
> detections on a schedule is Phase 9's maintenance.

## The chain

```
verdict.rendered(pass) → merge.requested → merge.observed → done
```

Each arrow is its own event and no code path collapses them
(III.G row 1). `verdict.rendered` was piped in 6.1
([`verdicts.md`](verdicts.md)); this stage pipes the rest.

**`merge.requested`** is a fact, not a transition: strict payload

```json
{"verdict": "<position>"}
```

citing the chain position of an admitted **pass** `verdict.rendered`
on the same subject, admitted only while the subject is in `review`.
Citing a fail verdict, a non-verdict position, or another subject's
verdict refuses by name — the chain-legality half of "a red verdict is
unmergeable"; the implementer-lockout half lands with 6.4. Capability:
`claim` or `operator` (asking for the merge is the work lane's act);
review carries no fence, so a citation refuses under the established
fence rule.

**`merge.observed`** stays the one transition to `done`
(`review` → `done`, the table row) and deepens to the observer's
forge fact: strict payload

```json
{"merged": "<full sha>", "pr": "<ref>"}
```

both presence-checked, `merged` 40–64 lowercase hex — the fact worth
recording is *which commit* the forge merged, the comparison anchor
for every divergence check below. The chain rule at admission: the
subject carries an admitted pass verdict AND an admitted
`merge.requested` citing it, in that order. Capability: the observer
lane — `observer` or `operator` (`actors.md`; the fold records the
single admitted observation `{position, sha}`, singular by
construction: the first valid observation lands on terminal `done`,
so a second can never admit, and raw-pushed extras stay anomalies).

Raw-pushed chain violations verify and fold tolerated with anomalies
counted, the established posture — which is exactly what makes
divergence *detection* necessary rather than assumed impossible.

## Divergence: detected, surfaced, reconciled

Detection is split across two surfaces by what each may read.

**Record-derivable classes** — pure functions of the fold, computed by
`internal/reconcile` and surfaced in the report's reconciliation
section (projection builds read records and 5.6's declared obs inputs,
never the artifact store or the repository):

| class | condition |
|---|---|
| `merge_without_verdict` | done, or an observed merge, with no admitted pass verdict |
| `chain_skipped` | an observed merge with no admitted `merge.requested` citing the verdict |
| `unreconciled` | a pass verdict with no observed merge yet |

`unreconciled` is reported **neutrally**, never as an accusation: no
build carries a wall clock, so "failed" versus "pending" is an age
judgment that belongs to Phase 9's maintenance thresholds.

**Evidence-grade checks** — `seed reconcile` only, which may read the
artifact store and the repository:

- **Attested-head reconciliation.** The cited receipt is loaded from
  the artifact store exactly as `verdict check` loads it
  (`evidence_missing` when the store cannot produce it intact — lost
  evidence is a finding, never silence). The clean cases are observed
  sha == attested head (fast-forward) and attested head an ancestor of
  the observed sha (a true merge commit). Anything else surfaces
  `attested_divergence` naming both shas and the receipt digest — a
  *surfaced state, not a fabrication verdict*: rebase and squash flows
  land here by design in v0 because the forge mints a new sha, and
  patch-equivalence reconciliation (comparing the merged change
  against the receipt's diff hash) is the named successor, recorded in
  the decision log.
- **Target-rewrite detection** (the charter's force-push case). A
  forge force-push writes no ledger event, so detection observes the
  target ref: reconcile resolves the target as the repository's
  checked-out default branch tip (v0; a declared target ref can ride a
  later phase) and reports `target_rewritten` when the observed merged
  sha no longer resolves or is no longer an ancestor of that tip.

`seed reconcile --ledger <dir> --repo <dir> [--subject <id>]
[--artifacts <dir>]` walks one subject or every folded subject, merges
both kinds, and returns the divergence set as envelope data at exit 0:
detection is a report, not a refusal — refusals stay operational
(usage, unreadable store, chain trouble). Phase 9's loop consumes the
same output.

## Visibility

Contracts `Version: "6"`: each entry carries
`verdict: {position, verdict, receipt} | null`,
`requested: <position> | null`, and `merged: {position, sha} | null`.
Report `Version: "3"`: the `reconciliation` section lists subjects
with record-derivable divergences by class, with counts, and names
`seed reconcile` for the evidence-grade rest. The cache mirrors the
new columns at schema generation 5. Every derivation change
republishes under a new version-bearing build id at an unchanged tip.

## Conformance mapping

- III.G row 1 (done only through the chain; each step its own event;
  no collapsing) — the chain rule at admission, the pinned
  review→done table row, and the full-chain drill.
- III.G row 2 (verdict/merge divergence is a detected, surfaced,
  reconciled state, drilled by inducing each) — the divergence
  taxonomy above with its induced drills: merge-without-verdict and
  chain-skipped from raw-pushed histories, unreconciled from a
  stalled chain, attested divergence from an observation outside the
  clean ancestry cases, target rewrite from a force-moved target ref,
  and evidence-missing from a lost receipt.
- Part II §8 ("maintenance surfaces and reconciles") — surfacing
  lands here as the report section and the reconcile verb; the
  scheduled, unattended runner is Phase 9 item 3.
