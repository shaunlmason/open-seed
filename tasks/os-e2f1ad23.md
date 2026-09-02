---
id: os-e2f1ad23
title: 'next: Phase 11 item 3 — poisoning drill: constructed trajectories fail to achieve promotion'
state: review
priority: P1
squad: core
author: seed-next-implementer
created_at: "2026-09-02T12:15:19Z"
updated_at: "2026-09-02T17:36:55Z"
---

Build plan Phase 11 item 3 (docs/next-build-plan.md): the poisoning drill. Trajectories are untrusted inputs (charter §12, III.K row 4): a corpus of deliberately constructed trajectories (single accidental success, one actor replaying itself, forged support, fabricated provenance, a lesson smuggled without its adversarial pass, a contested claim re-proposed unchanged, a worker promoting its own run) must fail to achieve promotion at every gate item 2 built (os-96850e5a), in CI, with each refusal named. Depends on os-f30ee0d3 (stores) and os-96850e5a (the gate). Plan-first (core squad, L2).

## Evidence ev-35206908 (pr, seed-next-implementer, 2026-09-02T16:18:02Z)

https://github.com/shaunlmason/open-seed/pull/236

## Evidence ev-2e6edd94 (receipt, seed-next-implementer, 2026-09-02T17:36:55Z)

receipts/os-e2f1ad23.json
