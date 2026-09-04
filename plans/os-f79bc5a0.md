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
   - Flip criterion 4's status from `reserved` to `met` in the
     criteria-at-a-glance table and the section 4 heading.
   - Replace section 4's `Missing:` and `Question:` lines with a
     plain statement of the operator's protocol amendment: the
     accelerated simulation is the run in place of the live seven-day
     shadow run; the deviation is named (synthetic backlog through
     the real boundary, zero credentials, mock executor; no live
     dual-run beside v1, no card of this repository filed). The
     five-bar audit over the simulated chain is clean (zero chain
     violations, zero lost updates, zero silent abandonments, zero
     guardrail breaches, zero unreserved spend; all 24 intents
     reached `done` in the accelerated seven-day window). Cite the
     simulation drills (`TestSimulateReachesDoneEnforced`,
     `TestSimulateAcceleratedBacklog`,
     `TestAuditCatchesSilentAbandonment`).
   - Update the gate sentence after the criteria table to reflect
     that every criterion is now met, with criterion 4 met by the
     operator's protocol amendment.
   - In the III.R measurement ledger intro paragraph, state that the
     operator's amendment supplies the measurement for R.4 and R.5,
     and that R.1–R.3, R.6, R.7 remain not measured (human review /
     real elapsed time / external adoption).
   - In the III.R ledger table, set R.4 and R.5 status to `measured`
     (the closed vocabulary word only; the explanation lives in the
     intro paragraph, not the table cells). Leave R.1–R.3, R.6, R.7
     as `not measured`.
2. Update `next/internal/promotion/promotion_test.go`:
   - In `TestPacketWritesTheCutoverDown`, replace the assertion that
     every ledger row is `not measured` with an assertion that R.4
     and R.5 are `measured` and the rest are `not measured`
     (reflecting the operator's amendment).

## File Scope

- `next/docs/promotion.md`
- `next/internal/promotion/promotion_test.go`

No other files. The conformance table
(`next/spec/conformance.json`) is not touched — the R.4/R.5 flip is
a follow-up spec card (protected path).

## Acceptance Criteria

- Criterion 4's status is `met` in both the criteria-at-a-glance
  table and section 4, with the deviation named plainly.
- The III.R ledger records R.4 and R.5 as `measured` and R.1–R.3,
  R.6, R.7 as `not measured`, with the reasons in the intro
  paragraph.
- The promotion drills (`TestPacketCitesRealDrills`,
  `TestPacketWritesTheCutoverDown`, and the rest of
  `internal/promotion`) pass.
- `make check` is green.
- The packet does not claim R.1–R.3, R.6, or R.7 are met; the
  doctor will still report Part III not complete, naming exactly
  those rows.
- Neither cutover is performed; both remain reserved escalations.

## Validation Commands

- `cd next && go test ./internal/promotion/ -count=1`
- `make check`

## Decisions

- The deviation is recorded in the packet itself (section 4), not
  only in the PR description, so the gate document is self-contained.
- The status column stays in the closed vocabulary; the explanation
  lives in the intro paragraph. This keeps the parser's strict
  shape check intact.
- The conformance-table flip is deferred to a follow-up spec card
  because `next/spec` is a protected path and the flip is a separate
  concern from the packet amendment.
