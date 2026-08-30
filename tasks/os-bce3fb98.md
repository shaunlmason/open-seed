---
id: os-bce3fb98
title: 'next: Phase 1.5 — halt semantics in the validation rule set'
state: in_progress
priority: P1
squad: core
blocked_on:
    - dep:os-ead12024
claim:
    actor: seed-next-implementer
    token: c-bf97f42798158e1e
    claimed_at: "2026-08-30T04:34:26Z"
    lease_expires: "2026-08-30T05:34:26Z"
created_at: "2026-08-30T03:35:40Z"
updated_at: "2026-08-30T04:34:26Z"
---

Build-plan item: Phase 1 item 5. halt.declared stops admission of everything except operator halt.lifted, as data-driven validation rules (the nascent rule set internal/admit consumes in Phase 2). Conformance: III.A halt item (boundary enforcement completes Phase 2). Intra-phase dep: 1.2 (os-ead12024).
