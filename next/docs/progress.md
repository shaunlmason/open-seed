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
  PR #71, task PR #72 — **done** (merged; card closed)
- 0.2 spec skeleton (`next/spec/protocol.md`, `next/spec/envelope.md`) —
  os-116ca9ac — task PR #72 — **done** (merged; encodings and upgrade-safe
  version semantics per #71/#75 review)
- 0.3 decision log (`next/docs/decisions.md`; plus this frontier file) —
  os-116ca9ac — task PR #72 — **done** (merged)
- 1.1 event model + JCS + Ed25519 — os-aa146827 — plan PR #73, task PR
  #76 — **done** (merged; strict wire parsing + lowercase-only hex per
  review; card closed)
- 1.2 chain/segments/HEAD — os-ead12024 — plan PR #74 (merged, amended),
  task PR #79 — review
- 1.3 genesis via `seed init` — os-d636299d — plan PR #75 (merged,
  amended) — blocked on dep:os-ead12024 (frees when 1.2 closes)
- 1.4 push-race append loop — os-62e2aa1d — backlog (deps 1.2, 1.3; plan
  when claimed)
- 1.5 halt semantics in the rule set — os-bce3fb98 — plan PR #78 open
  (amended: exit code 7, reason-carrying state) — parked on plan:78
  (+dep:os-ead12024)
- 1.6 payload classification lint + hostile corpus — os-d6f81ec6 — plan
  PR #77 open (amended: aggregate free-text budget, embedded rules) —
  parked on plan:77
- 1.7 CLI `seed ledger verify/append/show` — os-89412090 — backlog (dep 1.3)

## Frontier

Phase 0 and 1.1 are merged and closed; 1.2 (`next/internal/ledger`) is
implemented and in review on task PR #79. **Next action: when #79 merges,
close os-ead12024 (the cascade frees os-d636299d and os-bce3fb98's dep
entries) and implement 1.3 (`seed init`, plan merged) on
`seed/os-d636299d`; plans #77 (1.6) and #78 (1.5) unblock their cards on
merge. 1.4 (os-62e2aa1d) and 1.7 (os-89412090) get planned when their deps
close.** If #79 is red or carries review feedback, drive it green first —
nothing merges out of order (CI's plan-at-merge-base rule).
