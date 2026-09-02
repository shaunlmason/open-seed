---
id: os-d03bde01
title: 'next: budget exhaustion refuses as chain_invalid; no budget exit code exists'
state: in_progress
priority: P2
squad: core
claim:
    actor: seed-next-implementer
    token: c-91d0d80ffe1234af
    claimed_at: "2026-09-02T03:50:36Z"
    lease_expires: "2026-09-02T04:50:36Z"
created_at: "2026-09-01T14:19:51Z"
updated_at: "2026-09-02T03:50:36Z"
---

Found while implementing the worker loop (os-abb206c8): budget
exhaustion at `budget.reserve` refuses with the generic `chain_invalid`
(exit 8), the same code as a malformed payload or a broken chain.
next/spec/envelope.md allocates no budget-specific code.

Observed verbatim:

  code="chain_invalid" exit=8
  msg="admission refused by rule budget: budget on c-2 refused: amount
  101 exceeds remaining 100 of capacity 100 - reservations are checked
  and decremented at admission, the serialized view"

Why it matters rather than being cosmetic: exhaustion is a first-class,
expected, RECOVERABLE condition in the reservation model, and the loop's
exhaustion-parking exit carries the refusal's code and message into the
packet so the next worker gets the boundary's own account. A successor
reading "chain_invalid" would conclude the ledger is broken rather than
the budget spent. The message carries the whole account; the code
actively misleads.

It is also the affordance-gap class the charter cares about: a caller
cannot branch on exhaustion without string-matching the message, which
is exactly what structured codes exist to prevent.

Not fixed in os-abb206c8: an exit code is protocol surface governed by
next/spec/envelope.md, and that card's scope guard forbids widening it.
The current behavior is pinned by a characterization assertion in
cmd/seed/loop_e2e_test.go, so closing this fails that drill and forces
the spec to be updated with it.

Scope: allocate a budget exit code, map the budget rule's refusals onto
it, update next/spec/envelope.md and next/spec/budgets.md, and remove
the pin.
