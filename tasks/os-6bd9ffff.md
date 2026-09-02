---
id: os-6bd9ffff
title: 'next: Phase 10 item 5 — trajectory-prefix regression harness; dispatcher re-triage rate and planner unedited-approval rate'
state: blocked
priority: P1
squad: core
blocked_on:
    - plan:227
created_at: "2026-09-02T11:35:44Z"
updated_at: "2026-09-02T12:09:37Z"
---

Build plan Phase 10 item 5 (docs/next-build-plan.md): a trajectory-prefix regression harness for lane decision points, and the two lane-quality metrics the Phase 9 exit record routed here (III.J row 3): the dispatcher's re-triage rate and the planner's unedited-approval rate, both meaningless without the harness. Depends on items 2 through 4 (os-03e47abb, os-99829835, os-2e34f66a) for the eval machinery, the levels and the scorecards a decision point is replayed against. Plan-first (core squad, L2).
