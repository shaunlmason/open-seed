---
id: os-0d537fbd
title: 'next: Phase 11 item 4 — expiry for revalidation, retirement (evidence kept), rollback-by-revert; dead ends un-retired on environment change; staleness flags'
state: done
priority: P1
squad: core
author: seed-next-implementer
review:
    reviewer: shaunlmason
    reviewed_at: "2026-09-02T19:43:07Z"
    outcome: accepted
    evidence: https://github.com/shaunlmason/open-seed/pull/237
created_at: "2026-09-02T12:21:26Z"
updated_at: "2026-09-02T19:43:07Z"
---

Build plan Phase 11 item 4 (docs/next-build-plan.md): every promoted lesson carries a last-validated stamp and an expiry-for-revalidation; retirement revokes conclusions and keeps evidence; a promoted lesson implicated in a regression rolls back by reverting its PR, one command because it was a PR (charter §12, III.K row 6). Dead ends carry failure condition and environment and can be un-retired when the environment changes; the curator checks dead-end applicability, not just lesson applicability (III.K row 7). Knowledge bloat: staleness flags, dedup with provenance, structure lint (III.K row 9, routed here since no Phase 11 item names it). Builds on os-f30ee0d3 (stores) and os-96850e5a (the gate and the surfacing set). Plan-first (core squad, L2).

## Evidence ev-41cd3bba (pr, seed-next-implementer, 2026-09-02T17:55:27Z)

https://github.com/shaunlmason/open-seed/pull/237

## Evidence ev-1627b847 (receipt, seed-next-implementer, 2026-09-02T19:16:23Z)

receipts/os-0d537fbd.json
