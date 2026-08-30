# Plan: next Phase 1.4 — push-race append loop against a git remote (os-62e2aa1d)

Implements [`docs/next-build-plan.md`](../docs/next-build-plan.md) Phase 1
item 4: the optimistic append loop that makes `refs/seed/ledger` a
multi-writer ref. Design authority: [`SEED-NEXT.md`](../SEED-NEXT.md) Part
II §1 (ordering is admitted ancestry; the reference deployment rides a git
ref) and §II.3; builds on `internal/ledger` (os-ead12024) and
`internal/event`. Conformance: III.A — the race drill: two concurrent
appenders, no lost updates, one linear chain.

## Steps

1. **Git seam.** `next/internal/gitref`: a small exec wrapper around the
   `git` CLI (the engine's `internal/gitx` pattern; no new module
   dependencies) with exactly the verbs the loop needs: clone/fetch a ref,
   materialize the ledger tree into a store directory, commit the store
   directory as a tree on the ref, push, and detect non-fast-forward
   rejection distinctly from other push failures.
2. **Monotonic head (charter III.A "freshness")**: the client persists the
   newest verified remote head per (remote, ref) in a local head cache
   outside the ledger tree (`next/var/heads/`, gitignored; client state,
   never ledger state). Fetch/materialization refuses **head regression**:
   a fetched tip that is not a descendant of the persisted head (equal or
   extending it) is a typed `head_regression` refusal naming both heads,
   never silently accepted; the cache updates only after verification of
   the fetched stream. The rollback drill: roll the bare remote back to an
   earlier but internally valid tip; the loop refuses to append and the
   refusal surfaces the divergence.
3. **The loop.** `AppendLoop(remote, ref, draft, sign, validate)`:
   fetch → materialize → derive tip → **re-link** the draft (`prev` = the
   fresh tip) → **re-sign** (prev is inside the signed form, so only the
   key holder can re-link; the loop therefore takes a signing closure,
   never a pre-signed record) → run the `validate` callback against the
   refreshed store (the seam where Phase 2's admission rule set composes;
   v0 wires chain-level checks) → append locally → commit and push. On
   non-fast-forward: refetch and retry, bounded attempts. **A draft that
   fails re-validation after a refetch is reported with the losing reason,
   never silently re-appended** (the plan's normative sentence); retry
   exhaustion is a distinct typed error.
4. **Race drill** (conformance III.A): a local bare repository as the
   remote; two writers append interleaved batches concurrently (goroutines
   with real subprocess pushes); the drill asserts every event landed,
   exactly once, on one linear verifying chain, and that at least one
   non-fast-forward retry actually occurred (the race was real). A
   validate-refusal case shows the losing draft surfacing its reason.
5. **Tests** beyond the drill: materialize round-trip (ref → store →
   commit → ref, byte-stable); non-fast-forward detection; retry bound;
   offline remote surfaces a typed unavailable error (the offline-boundary
   groundwork: exclusivity-taking verbs are online-only, charter §II.6).

## File Scope

- `next/internal/gitref/**` (new package + drill + the persisted
  head cache under `next/var/heads/`, gitignored)
- `next/docs/decisions.md`, `next/docs/progress.md`

## Acceptance Criteria

**Boundary set (new, shown working):**

- Two concurrent appenders against one bare remote converge to a single
  linear chain containing every event exactly once; `VerifyFromGenesis`
  accepts the result; the drill proves a real retry happened.
- Re-linking always re-signs: no code path mutates `prev` on a signed
  record (the loop API makes it unrepresentable).
- A draft failing re-validation after refetch surfaces its reason to the
  caller; nothing silently re-appends; retry exhaustion is typed.
- Non-fast-forward, other push failures, and unreachable remotes are
  distinct typed errors.
- The monotonic-head rule holds: the persisted head only advances after
  verification, a rolled-back remote refuses with `head_regression`
  (drilled against the bare-remote fixture), and a fresh client with no
  persisted head verifies integrity but is documented as bounding
  freshness only by out-of-band anchors (the charter's honest residual).

**Retention set (existing, shown unharmed):**

- Every `next/internal` package present on main at implementation time,
  and the Phase 0/1 CLI, pass unchanged
  (`cd next && go test ./internal/... ./cmd/... -count=1`).
- The repo-wide gate stays green with the ≥90% coverage gate
  (`make check`).

## Validation Commands

- `make check-next`
- `make check`
- `cd next && go test ./internal/gitref/... -count=1`
- `cd next && go test ./internal/... ./cmd/... -count=1`
