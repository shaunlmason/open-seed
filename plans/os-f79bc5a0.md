# Plan: promotion packet — criterion 4 met-by-accelerated-simulation; III.R ledger R.4/R.5 measured

## Card

os-f79bc5a0 (P1, core).

## Context

The operator has accepted the credential-free accelerated simulation
(`seed simulate`, `--days 7`, `--intents 24`, `--posture
enforced-self-hosted`) as the criterion-4 evidence, in explicit
deviation from the build plan's live seven-day shadow run. The
promotion packet (`next/docs/promotion.md`) currently records
criterion 4 as `reserved` and every III.R ledger row as `not
measured`. This plan amends the packet to record the operator's
amendment honestly.

## Steps

1. Amend `next/docs/promotion.md`:
   - Flip criterion 4's status from `reserved` to `partial` in the
     criteria-at-a-glance table and the section 4 heading, with a
     `Missing:` line naming what the simulation does not satisfy.
   - Replace section 4's `Question:` line with a plain statement of
     the operator's substitution: the accelerated simulation is a
     substitution for the live seven-day shadow run; the deviation is
     named (synthetic backlog through the real boundary, zero
     credentials, mock executor; no live dual-run beside v1, no
     divergence reconciliation, no real backlog, no real week, no
     escalations). The five-bar audit over the simulated chain is
     clean (zero chain violations, zero lost updates, zero silent
     abandonments, zero guardrail breaches, zero unreserved spend;
     all 24 intents reached `done` in the accelerated seven-day
     window). Cite the simulation drills
     (`TestSimulateReachesDoneEnforced`,
     `TestSimulateAcceleratedBacklog`,
     `TestAuditCatchesSilentAbandonment`).
   - Update the gate sentence after the criteria table to reflect
     that the gate is not open (criterion 4 is `partial`).
   - In the III.R measurement ledger intro paragraph, state that the
     operator's substitution does not supply the measurement for any
     III.R row: the simulation does not run unattended for a week on
     a real backlog (R.5), it does not generate escalations (R.4),
     it has no human reviewer (R.1–R.3), it does not substitute for
     a quarter of real elapsed time (R.6), and it is internal and
     synthetic, not an external adoption (R.7). Every row remains
     `not measured`.
   - In the III.R ledger table, leave every row as `not measured`
     (the closed vocabulary word only; the explanation lives in the
     intro paragraph, not the table cells).
2. The promotion drill (`TestPacketWritesTheCutoverDown` in
   `next/internal/promotion/promotion_test.go`) is unchanged: it
   still asserts every ledger row is `not measured`, which is now
   correct (the simulation does not measure any III.R row).

## File Scope

- `next/docs/promotion.md`

No other files. The conformance table
(`next/spec/conformance.json`) is not touched — no III.R row is
flipped because the simulation does not measure any of them. The
promotion drill is unchanged (it still asserts every row is not
measured, which is now correct).

## Acceptance Criteria

- Criterion 4's status is `partial` in both the criteria-at-a-glance
  table and section 4, with the `Missing:` line naming what the
  simulation does not satisfy and the deviation named plainly.
- The III.R ledger records every row as `not measured`, with the
  reasons in the intro paragraph.
- The promotion drills (`TestPacketCitesRealDrills`,
  `TestPacketWritesTheCutoverDown`, and the rest of
  `internal/promotion`) pass.
- `make check` is green.
- The packet does not claim any III.R row is met; the doctor will
  report Part III not complete, naming every III.R row as
  outstanding.
- The gate is not open; neither cutover is performed; both remain
  reserved escalations.

## Validation Commands

- `cd next && go test ./internal/promotion/ -count=1`
- `make check`

## Decisions

- The deviation is recorded in the packet itself (section 4), not
  only in the PR description, so the gate document is self-contained.
- The status column stays in the closed vocabulary; the explanation
  lives in the intro paragraph. This keeps the parser's strict
  shape check intact.
- The conformance table is not flipped because the simulation does
  not measure any III.R row. The doctor will report Part III not
  complete, naming every III.R row as outstanding.
- The criterion is `partial`, not `met`, because the simulation
  provides some evidence (the five-bar audit is clean, the lanes
  reach done) but does not satisfy the criterion as written.
