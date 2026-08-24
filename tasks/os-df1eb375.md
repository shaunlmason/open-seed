---
id: os-df1eb375
title: 'paperclip: surface the paused-agent setup step in Connect'
state: in_progress
priority: P3
squad: core
claim:
    actor: shaunlmason
    token: c-3479e493dcf01453
    claimed_at: "2026-08-24T03:33:52Z"
    lease_expires: "2026-08-24T04:33:52Z"
created_at: "2026-08-24T03:33:41Z"
updated_at: "2026-08-24T03:34:44Z"
---

Follow-up from os-2c0c474c. Paperclip's checkout dispatches work rather than merely locking: assigning an issue wakes the agent, a runtime-less agent's run fails, and recovery.reconcile_stranded_assigned_issue moves the issue in_progress -> blocked within ~10s. Agents a seed deployment owns must therefore be paused (PATCH /api/agents/<id> {"status":"paused"}; the field is ignored on create).

That requirement is currently documented only under 'Declared variances', which is where a reader goes to understand semantics, not where an operator goes when setting the backend up. An operator following Connect will miss it, and their first claim will die within ~10s looking like a random contract failure rather than a missing setup step.

Scope:
- Move (or cross-reference) the requirement into the README's Connect checklist as a numbered setup step, with a copy-pasteable snippet that pauses every seed-owned agent.
- Keep the mechanism explanation in Declared variances: Connect says what to do, variances say why.
- The requirement is already exercised by live-test.sh, which pauses each corpus agent after provisioning; the doc should point at that as the worked example rather than restating it.

Acceptance: an operator following Connect end to end configures a working deployment without reading the variances section; the snippet is accurate against the pinned paperclip_version; make check stays green.

## Comment cm-dd241d1c (shaunlmason, 2026-08-24T03:34:44Z)

Plan PR: https://github.com/shaunlmason/open-seed/pull/64 (plan lint ok, class=plan purity ok). Parked pending review.
