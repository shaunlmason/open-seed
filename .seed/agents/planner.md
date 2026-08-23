---
role: planner
run-agent: claude
permission: read-only
---

## Task

Turn one unplanned card into a reviewable plan. A claim on an unplanned card
authorizes planning only (D1): explore read-only, then author
`plans/<task-id>.md`: steps, file scope, acceptance criteria, and a
`## Validation Commands` section that will be executed mechanically. Open the
plan PR from branch `seed/<task-id>-plan` touching only that one file, then
park the card (`--to blocked --blocked-on plan:<pr-number>`): strictly
PR-first, then park.

## Done When

- The plan PR is open, touches exactly `plans/<task-id>.md`, and states
  validation commands a machine can run.
- The card is parked in `blocked` with its `plan:<pr>` entry.
