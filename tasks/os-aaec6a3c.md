---
id: os-aaec6a3c
title: 'next: the guardrail bar requires an offer the boundary does not, so an admitted chain trips it'
state: in_progress
priority: P2
squad: core
claim:
    actor: seed-next-implementer
    token: c-81c3fd5a748c778a
    claimed_at: "2026-09-04T13:14:07Z"
    lease_expires: "2026-09-04T14:14:07Z"
created_at: "2026-09-04T13:12:49Z"
updated_at: "2026-09-04T13:14:07Z"
---

simulate.Audit's guardrail-breach bar names any subject whose claim.taken did not follow an offer.published (next/internal/simulate/audit.go). Admission does not hold that rule: internal/history.Generate writes an admission-grade chain that stages intent.filed, contract.specified and claim.taken with no offer at all, and that chain verifies and passes the seed-admit hook. So auditing a chain the boundary took reports guardrail breaches for every subject.

Found while implementing os-88df7ab2 (#311): its covered-arm drills audit a generated chain, and the guardrail bar fired on all of it. Those drills assert only the unreserved-spend bar and say why, so nothing is hidden, but the disagreement stands.

One of the two is wrong and the card decides which. Either the offer is a real precondition, in which case admission should refuse a claim that rides none and the history generator is writing chains the boundary should not take; or it is not, in which case the bar is restating a rule the protocol does not hold and III.R row 5's audit reports breaches that are not breaches. The same shape as os-b86dab4c (a bar counting a verb the protocol does not emit) and os-88df7ab2 (a bar deriving a rule admission already owns): the bar and the boundary must not disagree about what the chain means.

Expected shape: read transition's offer rules and admit's claim path, decide which authority holds, then either move the bar to ask the boundary's predicate or fix the generator and the rule it breaks. Tier: standard (it changes what a conformance bar counts, or what admission accepts).
