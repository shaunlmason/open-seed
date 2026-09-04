---
id: os-0dba8c6a
title: 'plan PR: os-2e34f66a''s dangling obligations.md citation'
state: in_progress
priority: P3
squad: core
claim:
    actor: seed-next-implementer
    token: c-362eb5368f3ac20a
    claimed_at: "2026-09-04T20:43:29Z"
    lease_expires: "2026-09-04T21:43:29Z"
created_at: "2026-09-04T13:15:10Z"
updated_at: "2026-09-04T20:43:29Z"
---

plans/os-2e34f66a.md line 130 writes ([`obligations.md`](obligations.md)) inside a plan file, so the target resolves against plans/, where no such file is. The document it means is next/spec/obligations.md, and the correction is one link target: `](obligations.md)` becomes `](../next/spec/obligations.md)`. The plan's substance does not change.

Why it is a card and not a line in someone's task PR: AGENTS.md binds a plan file to change only through its own plan PR touching that one file, so the correction needs a `seed/os-2e34f66a-plan` branch carrying nothing else. The citation stage of `seed docs check` (os-5fe43832, PR #305) therefore does not read `plans/` at all: a whole-tree gate that refused this citation would be unsatisfiable, since no branch allowed to carry the fix may also carry the gate. That exemption is why this rot needs a card to stay visible rather than a gate to catch it.

Found by the sweep that filed os-5fe43832: 7 of 570 relative markdown links in the tree did not resolve. Five are fixed on PR #305, one was a false positive (a regex in a code span), and this is the seventh.

Bounds: one file, one line, one target. Tier: trivial. Not conformance-blocking.
