---
id: os-f30ee0d3
title: 'next: Phase 11 item 1 — staged curation stores (observations → hypotheses → validated lessons → policy) with grant-gated boundaries; workers append candidates only'
state: in_progress
priority: P1
squad: core
claim:
    actor: seed-next-implementer
    token: c-2d579e1807f9022e
    claimed_at: "2026-09-02T13:57:37Z"
    lease_expires: "2026-09-02T15:57:37Z"
created_at: "2026-09-02T11:38:44Z"
updated_at: "2026-09-02T14:25:22Z"
---

Build plan Phase 11 item 1 (docs/next-build-plan.md): the poisoning-resistant pipeline's stages with distinct storage and distinct gates between them (charter SEED-NEXT.md §12 'A poisoning-resistant pipeline'; conformance III.K rows 1 and 2): online lanes append evidence only, conclusion-writing is grant-gated to the curator's proposal path, workers append candidate observations and never promoted lessons; no stage skips. The catalog's curation.* verbs (hypothesis.proposed, lesson.promoted, lesson.retired, deadend.recorded) are named in protocol.md and implemented nowhere; the curator lane holds no grant. Items 2 through 5 (the promotion gate, the poisoning drill, expiry and rollback, the workflow flywheel) build on these stores. Plan-first (core squad, L2).

## Evidence ev-4f57b75d (pr, seed-next-implementer, 2026-09-02T14:25:22Z)

https://github.com/shaunlmason/open-seed/pull/234
