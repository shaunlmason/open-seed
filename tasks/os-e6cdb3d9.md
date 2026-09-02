---
id: os-e6cdb3d9
title: 'next: Phase 9 exit record'
state: in_progress
priority: P1
squad: core
claim:
    actor: seed-next-implementer
    token: c-ed1449c68d11a754
    claimed_at: "2026-09-02T06:53:33Z"
    lease_expires: "2026-09-02T07:53:33Z"
created_at: "2026-09-02T06:53:07Z"
updated_at: "2026-09-02T06:53:33Z"
---

Phase 9 (lanes, escalation, maintenance) is complete once #212 merges,
and this card is its exit record: the update to next/docs/progress.md
that the Phase 5, 6, 7 and 8 exit records made, confirming charter III.J
row by row against what shipped.

Every numbered item has an implementation on main:

- 1 (lane fragments, dispatcher posture, injection corpus, worker loop
  with exhaustion parking): #188, #191, #192, with review fixes in
  os-378e44f3.
- 2 (escalation with packet, question, decision): #200.
- 3 (unattended maintenance loop): #205.
- 4 (small-team and fleet end-to-end fixtures): #207.
- 5(a) obligations projection and 5(c) loop verbs: #171, #173; 5(b)
  the situation read carrying the caller's mail: #211.
- The role-grant gap #207 found (no manifest granted supervise,
  observer or sealer): #212, pending merge at filing.

What the exit record must do, and not do:

- Confirm III.J's rows against the tree, by citation, and say plainly
  where a row is met with a recorded residual (the mirror arm of row 2
  is two-thirds met because request.* has no transition rows; the
  tier residual is owned by Phase 10). A row is not met because a
  paragraph says so.
- Correct the frontier line, which this phase has already had to
  correct once: an earlier revision claimed the phase complete when
  5(b) had no implementation. The exit record is the moment to
  re-derive the item list from the build plan rather than read the
  summary, and to say the next action is Phase 10 item 1.
- Carry forward what the phase learned about its own drills, since
  three cards in a row found hand-listed counts wrong: the exit-code
  table (os-d03bde01), the refusal-site matrix (same card), the
  capability-coverage gap (os-d6a52784).
- Change no code and no spec. If confirming a row turns up a gap, that
  is a card, not a line in this one.

Plan-first: docs-only, but the exit record is the document the next
phase orients from, and the last frontier claim was wrong. Block on
#212 until it merges.
