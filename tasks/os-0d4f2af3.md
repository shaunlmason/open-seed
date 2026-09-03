---
id: os-0d4f2af3
title: 'next: Phase 12 item 4 — the preseed file (config, guardrails, teams, protections, posture), idempotent and CI-verified; agent-only guardrails and the protected surface in config'
state: in_progress
priority: P1
squad: core
claim:
    actor: seed-next-implementer
    token: c-e67c7c0690956738
    claimed_at: "2026-09-03T01:29:17Z"
    lease_expires: "2026-09-03T05:10:46Z"
created_at: "2026-09-03T00:10:17Z"
updated_at: "2026-09-03T04:10:46Z"
---

Build plan Phase 12 item 4 (docs/next-build-plan.md): the preseed file — config, guardrails, teams, protections desired-state and the declared admission posture — bootstrapping a new adoption idempotently and CI-verifiably; plus the clauses the Phase 10 exit record routes here: the guardrails include agent-only ones read off the roster's `kind` and the report's lane rates split by kind (III.E row 9), and the config enumerates the protected surface and names the governance root and its change process (III.G row 9, III.L row 2).

Charter: §II.17 "Preseed" and Appendix D.1 (genesis is the primary adoption path: clone at a tagged release, init with a preseed, genesis names the root, enroll actors, file the first intent); §II.14 (the protected surface enumerated in config — transition spec, admission rules, guardrails, verifier code/rubrics, sealed keyring, curator gates, role definitions, supervisor policy, the check pipeline's own definitions — changed only by the governance root through PR + owner review, write-denied to every agent key it gates, with the test-content residual stated; tiers per squad and per path); III.P row 3; III.L rows 1 and 2; III.E row 9; III.G row 9.

What the tree has: seed init writes genesis from flags (#83); the posture declaration is one JSON field read through internal/posture (#98); lane manifests under next/lanes/ are the role definitions; tiers.md is the tier table; the roster projection and cache carry `kind` that nothing reads; nothing under next/ enumerates a protected surface or names a change process, and v1's .seed/config.toml, guardrails.yaml and teams.yaml are the predecessor shapes to learn from, never to import wholesale.

Expected shape, for the plan to settle: one declarative file (format decided in the plan) that `seed init --preseed` consumes idempotently — a second run changes nothing and reports drift — and a check verb CI runs against a fresh deployment; the protections block being what item 2's reconciler applies; the guardrails block carrying tiers per squad/path and the agent-only guardrails, both enforced somewhere real (a guardrail nobody reads is prose); the roster's `kind` consumed by those guardrails and by the report's lane rates; the protected-surface enumeration and the governance root's change process consumed by the doctor and by a capability audit that proves agent-key disjointness in CI. Phase 12 opens when the Phase 11 exit record merges; plan now, implement as a draft until then (decisions/0003).

## Evidence ev-8994ea1a (pr, seed-next-implementer, 2026-09-03T02:36:58Z)

https://github.com/shaunlmason/open-seed/pull/254
