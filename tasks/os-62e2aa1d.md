---
id: os-62e2aa1d
title: 'next: Phase 1.4 — push-race append loop against a git remote'
state: done
priority: P1
squad: core
author: seed-next-implementer
review:
    reviewer: shaunlmason
    reviewed_at: "2026-08-30T06:52:59Z"
    outcome: accepted
    evidence: https://github.com/shaunlmason/open-seed/pull/86
created_at: "2026-08-30T03:35:38Z"
updated_at: "2026-08-30T06:52:59Z"
---

Build-plan item: Phase 1 item 4. fetch, re-validate, re-link, push loop on refs/seed/ledger; losing events failing re-validation are reported, never silently re-appended; race drill: two concurrent appenders, no lost updates. Conformance: III.A (race drill). Intra-phase deps: 1.2 (os-ead12024), 1.3 (os-d636299d).

## Evidence ev-f4f23154 (, seed-next-implementer, 2026-08-30T06:24:50Z)


