---
id: os-c1dbf7bc
title: 'engine: seed situation — one-envelope resume packet for an actor'
state: cancelled
priority: P2
squad: core
created_at: "2026-09-01T03:01:21Z"
updated_at: "2026-09-01T03:12:31Z"
---

Agent-ergonomics program, item 1 (the highest-leverage item). An agent's dominant cost is resumption: after every compaction, wake, or crash it reconstructs the world from 6-10 calls (claims, lease clocks, card states, evidence, mail, frontier), and every error in that reconstruction compounds downstream — the requirements document is the hand-rolled state capsule the implementing agent maintained in its scheduler trigger prompt for exactly this purpose. Add one verb: seed situation --actor <fp> returning one position-stamped envelope with (a) the actor's claimed cards, each with state, lease expiry, blocked-on, attached evidence refs, and legal next transitions with required arguments (computed from the same transition tables that enforce them — one rule set, second consumer, zero drift); (b) unread mail count; (c) the frontier's next-action line when the repo carries a progress file. Claim tokens stay claim-time secrets, never re-shown. Success path stays terse; this is a read verb, idempotent, no state change. Metric: time-to-accurate-world-model.
