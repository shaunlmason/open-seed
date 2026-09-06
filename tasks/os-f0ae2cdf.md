---
id: os-f0ae2cdf
title: 'next: III.F row 12 (dependency cascades, hold suppression, initiative rollups, goal-ancestry warnings) is routed to a Phase 13 catch-all that names no item'
state: review
priority: P2
squad: core
author: seed-next-implementer
created_at: "2026-09-05T19:47:44Z"
updated_at: "2026-09-06T07:37:17Z"
---

Found by a read-only review of the tree at bfa01638 (2026-09-05). The conformance report (next/spec/conformance.json, pillar F row 12) is routed with the note "holds with suppression, initiative rollups and goal-ancestry warnings to Phase 13's catch-all, which names no item". The Phase 13 list (docs/next-build-plan.md:397-414) contains no such item and the backlog holds no matching card. Advisory wakes landed with #145 (Phase 7), but the cascade, hold-suppression, initiative-rollup and goal-ancestry half is unowned, so the row can never flip and Part III complete stays unreachable. This card: own the row - name the work (dependency-cascade derivation with advisory wakes, hold cascade with suppression, initiative rollups rendered, goal-ancestry warnings) with a plan and drills. Tier: standard. Plan-first.

## Evidence ev-1f8287e8 (, seed-next-implementer, 2026-09-06T07:36:39Z)

branch seed/os-f0ae2cdf: receipts/os-f0ae2cdf.json generated with --run --write, all five plan validation commands exit 0 including make check; III.F row 12 met

## Comment cm-120ef255 (seed-next-implementer, 2026-09-06T07:37:17Z)

task PR: https://github.com/shaunlmason/open-seed/pull/352
