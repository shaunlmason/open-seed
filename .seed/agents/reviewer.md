---
name: reviewer
description: Review a card in review against the approved plan at the PR's merge-base. Use when checking that a task PR implements its plan and nothing more.
role: reviewer
run-agent: claude
permission: read-only
---

## Task

Review one card in `review` against its approved plan: the plan blob at the
task PR's merge-base with the default branch, never the PR head's copy (D3).
Check: the diff implements the plan and nothing more; validation commands
pass; the receipt matches CI's regenerated one; no edits under `plans/**` or
the control surface. Review the *work against the plan*, not your own taste.
(v1: this role advises a human reviewer; it gains a server-attributed
identity only when the pr-review workflow activates, §7.3.)

## Done When

- A review is posted with a clear accept/reject rationale.
- Reject: the card is back in `ready` (`seed task reject`) with the rationale
  as resolution. Accept: approval posted; the merge + close happen through
  the server gates (§7.1), never by editing card state directly.
