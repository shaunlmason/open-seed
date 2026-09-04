---
id: os-9ef9ab34
title: 'next: a charter MAY sits open in the table, so seed doctor can never report complete (III.B row 6 blocks promotion)'
state: blocked
priority: P1
squad: core
blocked_on:
    - plan:307
created_at: "2026-09-04T11:41:16Z"
updated_at: "2026-09-04T11:43:29Z"
---

`conformance.Assess` reports `complete: true` only when every applicable row is `met`: partial and routed are outstanding, and there is no exemption for a permission (next/internal/conformance/conformance.go, the Assess doc comment and the Outstanding loop). III.B row 6 is a permission, not an obligation: 'Admission MAY shard proposal intake without changing semantics; ordering remains solely the admitted chain' (SEED-NEXT.md line 833). It sits `open` in next/spec/conformance.json with the note 'not claimed: MAY; the backlog names sharded intake as a true extra'.

Those two facts are inconsistent, and the inconsistency is on the promotion critical path:

- plans/os-d63c7441.md D3 closes Phase 13 only when the doctor reports `complete: true`, which means no row open, partial or routed.
- next/docs/promotion.md criterion 6 cites the doctor's conformance section, and build plan section 5 reads the packet.
- So while B.6 stays `open`, the doctor can never report complete, the Phase 13 exit record can never close, and promotion cannot finish — no matter what the shadow run and the cutovers produce.

Two ways out, and the plan should pick one and say why:

1. A permission is satisfied by abstention. A system that does not shard intake conforms to a MAY trivially; the row's real content is conditional (IF admission shards, THEN semantics do not change and ordering stays the admitted chain). Then B.6 is `met` today, with the evidence being that intake is single-path and ordering is the admitted chain, and the table gains a way to say so — either the status with a note, or a posture/kind marker for permissions that Assess sets aside like the enforced-only rows.
2. The permission is treated as an obligation. Then sharded intake (os-7953612b) is not a 'true extra' but a promotion blocker, and the backlog line and the frontier both say the opposite today.

Reading 1 looks right (the charter's other MAY-shaped rows should be checked for the same shape while planning). Whichever is chosen, the fix touches the conformance table and possibly Assess, so it is plan-first: this card is filed, not started. Related: III.C row 4 stays partial until the weekly scale run is green, and its 'without unrelated writes racing each other's admissions pathologically' clause is also sharded intake's, so reading 2 would make os-7953612b block two rows.

Found while checking whether any agent-side conformance work remained before the promotion gate (2026-09-04). Tier: standard.
