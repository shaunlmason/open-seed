---
id: os-e2f1ad23
title: 'next: Phase 11 item 3 — poisoning drill: constructed trajectories fail to achieve promotion'
state: in_progress
priority: P1
squad: core
claim:
    actor: seed-next-implementer
    token: c-77557bfeff1d87d2
    claimed_at: "2026-09-02T15:24:58Z"
    lease_expires: "2026-09-02T18:13:57Z"
created_at: "2026-09-02T12:15:19Z"
updated_at: "2026-09-02T16:13:57Z"
---

Build plan Phase 11 item 3 (docs/next-build-plan.md): the poisoning drill. Trajectories are untrusted inputs (charter §12, III.K row 4): a corpus of deliberately constructed trajectories (single accidental success, one actor replaying itself, forged support, fabricated provenance, a lesson smuggled without its adversarial pass, a contested claim re-proposed unchanged, a worker promoting its own run) must fail to achieve promotion at every gate item 2 built (os-96850e5a), in CI, with each refusal named. Depends on os-f30ee0d3 (stores) and os-96850e5a (the gate). Plan-first (core squad, L2).
