---
id: os-116ca9ac
title: 'next: Phase 0 — workspace and spec skeleton'
state: in_progress
priority: P1
squad: core
claim:
    actor: seed-next-implementer
    token: c-68675a13ea182940
    claimed_at: "2026-08-30T03:21:02Z"
    lease_expires: "2026-08-30T04:21:02Z"
created_at: "2026-08-30T03:15:44Z"
updated_at: "2026-08-30T03:21:02Z"
---

Build-plan item: docs/next-build-plan.md Phase 0 (items 1-3, one coherent cluster). Scaffold the next/ Go module with CLI skeleton (seed version), wire make check-next (build+vet+test+coverage gate) into make check (the named v1 integration point); write next/spec/protocol.md v0 (JCS canonical form, event fields, hash/signature algorithms, verb catalog per charter Appendix B, envelope shape, exit codes); create next/docs/decisions.md (decision log) and next/docs/progress.md (frontier tracker per plan section 4). Exit criteria: make check green with check-next included; spec doc exists and matches charter Appendix B; decision log exists. Advances conformance groundwork for SEED-NEXT.md Part III.A/I; design authority SEED-NEXT.md Part II; sequencing docs/next-build-plan.md.
