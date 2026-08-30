---
id: os-62e2aa1d
title: 'next: Phase 1.4 — push-race append loop against a git remote'
state: blocked
priority: P1
squad: core
blocked_on:
    - dep:os-d636299d
    - plan:81
created_at: "2026-08-30T03:35:38Z"
updated_at: "2026-08-30T05:21:02Z"
---

Build-plan item: Phase 1 item 4. fetch, re-validate, re-link, push loop on refs/seed/ledger; losing events failing re-validation are reported, never silently re-appended; race drill: two concurrent appenders, no lost updates. Conformance: III.A (race drill). Intra-phase deps: 1.2 (os-ead12024), 1.3 (os-d636299d).
