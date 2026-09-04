---
id: os-b86dab4c
title: 'next: the five-bar audit counts budget.reserved, a verb the protocol does not define, so a real chain audits as unreserved spend (III.R row 5)'
state: in_progress
priority: P1
squad: core
claim:
    actor: seed-next-implementer
    token: c-3bdfe6ca40fae005
    claimed_at: "2026-09-04T06:43:41Z"
    lease_expires: "2026-09-04T07:43:41Z"
created_at: "2026-09-04T06:43:23Z"
updated_at: "2026-09-04T06:43:41Z"
---

simulate.Audit's unreserved-spend bar counts a reservation under the verb `budget.reserved` (next/internal/simulate/audit.go, the case at line 93). The protocol defines and emits `budget.reserve` (transition.BudgetReserveVerb; every cmd/seed drill that files one uses that name). So a real chain whose run.started is covered by an admitted budget.reserve is reported as unreserved spend, and `seed ledger audit` (os-7599c27d, #296) cannot measure III.R row 5 over the shadow run, which is the one thing it exists for.

The wrong name lives in exactly two places, audit.go and its own fixture (internal/simulate/audit_test.go), and the fixture repeats it, which is why the bar's drills pass; the CLI drill added with #296 (cmd/seed/ledger_audit_test.go) inherited the same happy path. Found by review on #296 (chatgpt-codex-connector, 2026-09-04); the analysis and the two decisions are written up in the plan amendment #303, which is closed unmerged because #296 landed first and a merged card's plan is not the place to authorize new work.

Expected shape: name the protocol's verb in the bar, move both fixtures to it, and add the regression the current drills could not fail: a chain with an admitted budget.reserve audits clean, and one carrying budget.reserved reads as unreserved spend. Tests plus one internal file. Tier: standard (it changes what a conformance bar counts).
