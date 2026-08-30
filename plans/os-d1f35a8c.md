# Plan: next Phase 3.3 — key rotation and revocation drill (os-d1f35a8c)

Implements [`docs/next-build-plan.md`](../docs/next-build-plan.md) Phase 3
item 3 and closes the Phase 3 exit: the charter's revocation drill as an
automated test — revoke ends the key's standing, its history stays
attributed. Design authority: [`SEED-NEXT.md`](../SEED-NEXT.md) Part II
"Enrollment" ("Key rotation and revocation are events; a revoked key's
history remains attributed to it; revocation is drilled") and the
compromised-actor consequence ("revocation — detection ends it").
Conformance: the III.E revocation item's Phase-3 half (standing ends,
attribution preserved, rotation works); claim reaping lands with claims
(Phase 5), sealed-check keyring rotation with verifier keys (Phase 6),
and the exit record names each. 3.1 landed the lifecycle mechanics and
verification tests (#100); this task turns them into the charter's
operational drill across every surface, the shape Phase 12's
compromised-actor drill extends.

## Steps

1. **The rotation drill.** On an enforced deployment at `seed/1`: enroll
   key A, let it work (admitted events through the boundary), then
   rotate — enroll replacement B, revoke A (reason recorded), both by
   an operator. Assert the full consequence set: A's pre-revocation
   history verifies from genesis and stays attributed to A's
   fingerprint; every A-signed proposal after revocation refuses on
   every surface (the admission rule set, the `seed-admit` boundary,
   and the CLI remote path, each with its established refusal shape);
   B works across the same surfaces; and a from-genesis verification of
   the final chain is green with the standing changes in place.
2. **The compromised-key cut, per posture.** The drill the charter's
   threat model implies: a hostile holder of the revoked key pushes raw.
   Under enforced, the push refuses with the ref unmoved. Under
   cooperative, the raw push lands — and the landed record breaks the
   shared chain for every reader at exactly the revoked signer's
   position (the named consequence made observable, matching the
   adversary-table pattern from #99): the cooperative client's
   pre-flight replay refuses with the standing named in the detail.
3. **Rotation is complete replacement.** After the rotation, revoking B
   requires another active operator path (root liveness holds); A
   cannot be re-enrolled (revocation is terminal, drilled at the
   boundary rather than only in the library); and a grant issued to A
   before revocation confers nothing after it (capability checks read
   standing first).
4. **Phase 3 exit bookkeeping.** `next/docs/progress.md` records the
   Phase 3 exit as exactly what docs/next-build-plan.md's exit line
   scopes it to — **the Phase 3 subset of III.E**, never "III.E met":
   events + projection + standing-aware signature verification (#100),
   grant checks with structural operator-only refusals and kind as a
   drilled assertion (#102), and revocation drilled (this task). The
   record enumerates every still-unmet III.E criterion with its landing
   phase: implementer-disjoint self-approval (verdict verbs, Phase 6);
   sealed-check keyring rotation on verifier-key revocation (Phase 6);
   claim reaping on revocation (claims, Phase 5); qualification tuples
   and grants citing them (Phase 10); scheduled spot-check suspension
   (Phase 10); and the roster distinction consumed by agent-only
   guardrails and human/agent metrics (with those surfaces, Phases 8
   and 11). The frontier points at Phase 4 (projections).

## File Scope

- `next/cmd/seed-admit/**` (the rotation and compromised-key drills)
- `next/cmd/seed/**` (CLI rotation path assertions)
- `next/internal/admit/**`, `next/internal/keyring/**` (test-only
  additions if the drills need fixtures; no behavior changes planned)
- `next/docs/decisions.md`, `next/docs/progress.md`,
  `memory/LEARNINGS.md` (if lessons emerge)

## Acceptance Criteria

**Boundary set (new, shown working):**

- The rotation drill passes end-to-end on the enforced deployment:
  A works, A is revoked, A's history verifies attributed, every
  post-revocation A proposal refuses at the rule set, the boundary,
  and the CLI, and B works on all three.
- The compromised-key cut shows the posture split: refusal with the
  ref unmoved under enforced; the landed raw push breaking the shared
  chain at the revoked signer's position under cooperative, named in
  the client's refusal detail.
- Revocation is terminal and grants die with standing, both drilled at
  the boundary; root liveness holds through the rotation.
- The Phase 3 exit paragraph in `next/docs/progress.md` claims only
  the Phase 3 subset of III.E, cites the passing drills, and
  enumerates every still-unmet III.E criterion with its landing phase
  (self-approval disjointness and sealed-check rotation in 6, claim
  reaping in 5, tuples and spot-check suspension in 10, roster-reading
  guardrails and metrics with their surfaces).

**Retention set (existing, shown unharmed):**

- All existing verb suites pass unchanged
  (`cd next && go test ./internal/... ./cmd/... -count=1`).
- The repo-wide gate stays green with the ≥90% coverage gate
  (`make check`).

## Validation Commands

- `make check-next`
- `make check`
- `cd next && go test ./cmd/seed-admit/... -count=1`
- `cd next && go test ./internal/... ./cmd/... -count=1`
