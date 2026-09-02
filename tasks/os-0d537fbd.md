---
id: os-0d537fbd
title: 'next: Phase 11 item 4 — expiry for revalidation, retirement (evidence kept), rollback-by-revert; dead ends un-retired on environment change; staleness flags'
state: in_progress
priority: P1
squad: core
claim:
    actor: seed-next-implementer
    token: c-37269ea16180d02d
    claimed_at: "2026-09-02T17:01:11Z"
    lease_expires: "2026-09-02T19:53:54Z"
created_at: "2026-09-02T12:21:26Z"
updated_at: "2026-09-02T17:53:54Z"
---

Build plan Phase 11 item 4 (docs/next-build-plan.md): every promoted lesson carries a last-validated stamp and an expiry-for-revalidation; retirement revokes conclusions and keeps evidence; a promoted lesson implicated in a regression rolls back by reverting its PR, one command because it was a PR (charter §12, III.K row 6). Dead ends carry failure condition and environment and can be un-retired when the environment changes; the curator checks dead-end applicability, not just lesson applicability (III.K row 7). Knowledge bloat: staleness flags, dedup with provenance, structure lint (III.K row 9, routed here since no Phase 11 item names it). Builds on os-f30ee0d3 (stores) and os-96850e5a (the gate and the surfacing set). Plan-first (core squad, L2).
