---
id: os-62e2aa1d
title: 'next: Phase 1.4 — push-race append loop against a git remote'
state: in_progress
priority: P1
squad: core
blocked_on:
    - dep:os-ead12024
    - dep:os-d636299d
claim:
    actor: seed-next-implementer
    token: c-54eafa55fb4ea34f
    claimed_at: "2026-08-30T05:05:34Z"
    lease_expires: "2026-08-30T06:05:34Z"
created_at: "2026-08-30T03:35:38Z"
updated_at: "2026-08-30T05:05:34Z"
---

Build-plan item: Phase 1 item 4. fetch, re-validate, re-link, push loop on refs/seed/ledger; losing events failing re-validation are reported, never silently re-appended; race drill: two concurrent appenders, no lost updates. Conformance: III.A (race drill). Intra-phase deps: 1.2 (os-ead12024), 1.3 (os-d636299d).
