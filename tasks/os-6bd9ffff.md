---
id: os-6bd9ffff
title: 'next: Phase 10 item 5 — trajectory-prefix regression harness; dispatcher re-triage rate and planner unedited-approval rate'
state: review
priority: P1
squad: core
author: seed-next-implementer
created_at: "2026-09-02T11:35:44Z"
updated_at: "2026-09-02T20:33:48Z"
---

Build plan Phase 10 item 5 (docs/next-build-plan.md): a trajectory-prefix regression harness for lane decision points, and the two lane-quality metrics the Phase 9 exit record routed here (III.J row 3): the dispatcher's re-triage rate and the planner's unedited-approval rate, both meaningless without the harness. Depends on items 2 through 4 (os-03e47abb, os-99829835, os-2e34f66a) for the eval machinery, the levels and the scorecards a decision point is replayed against. Plan-first (core squad, L2).

## Evidence ev-1743e9c3 (pr, seed-next-implementer, 2026-09-02T20:21:29Z)

https://github.com/shaunlmason/open-seed/pull/239

## Evidence ev-4b2a8b37 (receipt, seed-next-implementer, 2026-09-02T20:33:44Z)

receipts/os-6bd9ffff.json
