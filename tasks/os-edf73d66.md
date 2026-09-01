---
id: os-edf73d66
title: 'next: Phase 8.3 — refusal-rate metric in the report'
state: done
priority: P2
squad: core
author: seed-next-implementer
review:
    reviewer: shaunlmason
    reviewed_at: "2026-09-01T03:03:35Z"
    outcome: accepted
    evidence: https://github.com/shaunlmason/open-seed/pull/165
created_at: "2026-08-31T23:10:23Z"
updated_at: "2026-09-01T03:03:35Z"
---

Build plan Phase 8 item 3; charter III.I row 4 (refusal rates tracked as an affordance-gap metric) and section II.10. Refusals never reach the chain, so the metric needs a recorded source: journal admission refusals locally at the refusing CLI seams (best-effort, never failing the verb), declare the journal to the report build as an input following the observations pattern (digest-covered, section null on input-free builds), and emit counts, by-code/by-verb breakdowns, and a position-span rate. Plan-first (L2).

## Evidence ev-6f294c30 (pr, seed-next-implementer, 2026-09-01T02:49:20Z)

https://github.com/shaunlmason/open-seed/pull/165
