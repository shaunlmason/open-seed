# plans.md — plans as falsifiable change contracts

> Status: v0, normative for `next/**`. Authority: [`SEED-NEXT.md`](../../SEED-NEXT.md)
> Part II §6 "Plans as falsifiable change contracts";
> [`docs/next-build-plan.md`](../../docs/next-build-plan.md) Phase 5 item 5;
> plan `plans/os-16c1d142.md`. Implemented by `next/internal/plan`,
> the `plan` admission rule, and the `seed plan lint` / `seed plan
> classify` verbs.

## The falsifiable-plan lint

Above the trivial tier, claiming an unplanned contract authorizes
planning only; a plan is a file merged through its own single-file PR
at `plans/<subject>.md`, and it MUST be falsifiable: four required,
non-empty parts —

1. **Boundary set** — what is broken and will be shown fixed.
2. **Retention set** — what works and will be shown unharmed. **A
   plan without a retention check does not lint** (the charter's own
   clause, a distinct named finding).
3. **Validation commands** — visibly covering the boundary AND the
   retention set (a `Boundary:` and a `Retention:` command line).
4. **Expected diff shape**.

A section marker is a markdown heading or a bold line beginning with
the part's phrase (the repository's own plan shape, the
v1-continuity default); `seed plan lint <file>` renders findings at
exit 9 and the lint never executes commands. The lint's grammar and
the loop's practice are pinned together: the drill lints this task's
own plan clean.

## The plan verbs and the submission gate

The catalog's only plan verbs: `plan.proposed` (payload
`{"plan": "<path @ commit>"}`, the claim lane — the fence matrix
applies on a claimed subject) and `plan.approved` (the observation of
the plan PR merge: `{"plan": "<path @ commit>", "pr": "<pr @
merged-commit>"}`, operator-attested in v0, the `merge.observed`
posture). Both are **facts, not transitions**: no lifecycle state
changes, and the 5.1 pinned invariant stands.

**The gate bites at `submission.made`** — the charter allows claiming
an unplanned contract, and planning needs the claim's exclusivity, so
the earliest ledger point where implementation-not-planning manifests
is the submission. Above the trivial tier (`tier` ≠ `"trivial"`, the
one distinguished value; content semantics wait for the tier system),
a submission refuses **exit 16 `plan_required`** unless the subject
carries an admitted `plan.approved` AND the submission cites the plan
anchor it implements (`{"plan": "<path @ commit>"}`), anchor for
anchor: the citation must equal the approved anchor exactly, because
an approval admits one revision and any other anchor leaves the
receipt verifier holding a value nothing vouched for. Two layers:
this ordering-plus-citation check at admission, and the **ancestry
binding** — the implementation actually built on the approved plan —
by Phase 6's receipt computation ("plan hash at merge-base"), the
named closing item for the v0 window. Mis-tiering to dodge planning
is ledger-visible and a named residual until tier provenance lands.

## Structural disjointness

Plan PRs and implementation PRs are structurally disjoint: a change
set may touch exactly one `plans/` file and nothing else (a plan PR),
or no `plans/` file at all (an implementation PR); anything else is
mixed and refuses. `seed plan classify [paths… | -]` is the
CI-invocable check (args or newline-separated stdin, the shape a
forge's changed-files list provides), exit 9 on mixed. Making the
check **forge-required** for self-hosted deployments is the Phase 12
protections desired-state reconciler's item; the SEED-NEXT
development loop in this repository is enforced today by v1's own
classifier in the verify workflow.

## Conformance mapping

- III.F "Plans are falsifiable: boundary set, retention set,
  validation commands for both, expected diff shape; missing
  retention fails lint; plan and implementation PRs are structurally
  disjoint" — the lint (missing-retention distinct), the classifier
  and its refusing entrypoint, the plan verbs, and the submission
  gate; the ancestry half explicitly Phase 6's receipt.
