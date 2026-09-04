---
id: os-88df7ab2
title: 'next: the unreserved-spend bar counts a reservation''s occurrence, not an open valid one (III.R row 5)'
state: review
priority: P2
squad: core
author: seed-next-implementer
created_at: "2026-09-04T09:52:36Z"
updated_at: "2026-09-04T16:34:25Z"
---

simulate.Audit's unreserved-spend bar sets one boolean on any `budget.reserve` record and clears it only on `claim.taken` (next/internal/simulate/audit.go). Two consequences, both on the chains `seed ledger audit` is pointed at:

1. A reservation closed by `budget.settle` or `budget.release` keeps covering later spend: reserve, settle, then a second `run.started` in the same claim window reads as covered, and the bar stays empty.
2. A `budget.reserve` admission would reject counts the same as an admitted one, because the bar reads the verb's occurrence and not its validity. The audit runs after chain verification, which establishes signatures and transition legality, never admission; a raw-pushed reservation is exactly the case the verb exists to judge.

The protocol already folds this correctly: transition.ReservationFact carries the position, the reserving signer and the amount (plans/os-cecac5de.md), so the bar can ask the fold rather than re-derive a boolean. That is the same shape as os-b86dab4c's fix, which stopped the audit restating the protocol's verb names; this card stops it restating the protocol's budget model.

Found by review on #306 (chatgpt-codex-connector, 2026-09-04). Out of that card's bounds by its plan D5 (no bar changes meaning), so it is filed rather than folded in. Expected shape: derive the covering reservation from the fold, position-accurately, with drills for the settle-then-spend and release-then-spend arms and for a reservation admission would reject. Tier: standard (it changes what a conformance bar counts).

## Comment cm-86311f52 (seed-next-implementer, 2026-09-04T11:55:27Z)

Dependency, since the queue would not take a second blocked_on entry (the engine refuses blocked -> blocked, so the card carries plan:309 alone): this card also waits on os-b86dab4c. Its fix (#306) renames the verb this bar counts to the protocol's budget.reserve and adds the exported verb set the switch reads, so implementing here before that is on main would rewrite the same switch and conflict. The plan states it (amendment on #309, 'The base is #306, not main as it stands'). Order: #306 merges, then #309, then this card's implementation.

## Evidence ev-64fd7b40 (pr, seed-next-implementer, 2026-09-04T16:33:52Z)

https://github.com/shaunlmason/open-seed/pull/311

## Evidence ev-2bcd1361 (receipt, seed-next-implementer, 2026-09-04T16:33:56Z)

receipts/os-88df7ab2.json
