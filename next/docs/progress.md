# progress.md — the Seed implementation frontier

The single resume point (docs/next-build-plan.md §4): one line per plan item,
`phase.item — card id — PR — state`. A fresh agent resumes from this file
alone: read it top to bottom, verify the PR states it claims against the
forge, then take the frontier line's next action. Keep it truthful in every
PR that touches `next/**`; never start new work while it misstates the
frontier.

## Roster and conventions

- Implementer actor: `seed-next-implementer` (work verbs). Operator queue
  verbs for `next:` cards run under the session principal per
  `decisions/0003-next-loop-delegation.md`.
- Cards carry the `next:` title prefix; branches `seed/<card>` (task) and
  `seed/<card>-plan` (plan); receipts generated with `--run --write` before
  review.
- Phase gate: a phase starts when its dependencies' exit criteria are
  **merged** (docs/next-build-plan.md §2).

## Ledger

- 0.1 module scaffold + CI wiring (`make check-next`) — os-116ca9ac — plan
  PR #71 (merged), task PR #72 — review
- 0.2 spec skeleton (`next/spec/protocol.md`, `next/spec/envelope.md`) —
  os-116ca9ac — task PR #72 — review (wire encodings fixed per #71 review)
- 0.3 decision log (`next/docs/decisions.md`; plus this frontier file) —
  os-116ca9ac — task PR #72 — review
- 1.1 event model + JCS + Ed25519 — os-aa146827 — plan PR #73 — planned,
  parked on plan:73
- 1.2 chain/segments/HEAD — os-ead12024 — plan PR #74 — planned, parked on
  plan:74 (+dep:os-aa146827)
- 1.3 genesis via `seed init` — os-d636299d — plan PR #75 — planned, parked
  on plan:75 (+dep:os-ead12024)
- 1.4 push-race append loop — os-62e2aa1d — backlog (deps 1.2, 1.3; plan
  when claimed)
- 1.5 halt semantics in the rule set — os-bce3fb98 — backlog (dep 1.2)
- 1.6 payload classification lint + hostile corpus — os-d6f81ec6 — backlog
  (dep 1.1)
- 1.7 CLI `seed ledger verify/append/show` — os-89412090 — backlog (dep 1.3)

## Frontier

Phase 0 is implemented and in review on task PR #72; Phase 1 cards are
filed, and the critical-path plans (#73, #74, #75) are open per the batch
working model (`decisions/0003-next-loop-delegation.md`): plans pipeline
while merges batch, implementation waits for its plan AND its deps.
**Next action: once #72 and #73 are merged and os-aa146827 is unblocked,
claim it and implement 1.1 (`next/internal/event`) in a worktree on
`seed/os-aa146827`; then 1.2 → 1.3 in dep order, planning 1.4–1.7 as their
deps close.** If #72 is red or carries review feedback, drive it green
first — nothing merges out of order (CI's plan-at-merge-base rule).
