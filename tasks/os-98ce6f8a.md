---
id: os-98ce6f8a
title: 'next: the promotion evidence packet — build-plan §5''s seven self-hosting criteria mapped to evidence on main, missing items named (feeds the Phase 13 exit record)'
state: blocked
priority: P2
squad: core
blocked_on:
    - plan:291
created_at: "2026-09-03T17:59:39Z"
updated_at: "2026-09-03T22:58:35Z"
---

Build plan section 5 lists seven criteria for the self-hosting cutover — a HUMAN decision the plan says agents drive up to and STOP at (never autonomously decidable). This card produces the evidence document the operator reads at that gate: one place mapping each of the seven criteria to the evidence already on main, and naming what is still missing.

The seven criteria (docs/next-build-plan.md §5): (1) loop-completeness (Phase 9 item 5), (2) lanes operable (Phase 9), (3) migration proven — seed import --from-open-seed drilled against a REAL export of this repo's v1 state, (4) shadow run (the declared-slice run beside v1 — the item that is still MISSING and itself human-gated), (5) cutover and rollback written down, (6) core conformance (Phases 0-12), (7) the compromised-actor drill green in CI (Phase 12 item 1) before cutover.

Deliverable: a document (next/docs/) that, per criterion, cites the concrete evidence on main by name — the drills, the report's sections, the simulate audit, the migration drill — and states each criterion's status (met / partially met / not started), explicitly naming the shadow run as the remaining gap and marking the cutover itself as a reserved human escalation. It presents evidence; it does not decide. This feeds the Phase 13 exit record (os-d63c7441): the record re-derives the Frontier toward promotion, and this packet is what that re-derivation reads.

Tier: trivial in code (a document + any read-only survey helper), plan-first because it is what the promotion gate reads. Depends on the Phase 13 items being on main (their drills are part of the evidence).
