---
id: os-40ed0ca0
title: 'next: Phase 13 item 5 — the A2A-shaped cross-organization boundary (III.N)'
state: review
priority: P2
squad: core
author: seed-next-implementer
created_at: "2026-09-03T02:39:57Z"
updated_at: "2026-09-03T12:00:43Z"
---

Build plan Phase 13 item 5 (docs/next-build-plan.md): the A2A-shaped cross-organization boundary.

Charter: §II.15 (cross-org collaboration is opaque: capability cards, a task lifecycle, artifacts only), III.N row 3 ("Cross-org collaboration is opaque (capability cards, task lifecycle, artifacts-only)"). Deps: Phase 12; builds on item 4's request ingress and read remotes.

What the tree has: the actor keyring with kinds and capabilities (actors.md), offers scoped by eligibility (offers.md), artifacts by digest with sealed variants (artifact store, sealed-checks.md), the classification lint refusing content in payloads, and after item 4 a request ingress. No notion of another organization exists: every actor is enrolled in this ledger, and every artifact is this deployment's.

Expected shape, for the plan to settle: a capability card as the published, signed description of what a deployment offers (the A2A agent-card shape as reference, never a dependency), a task lifecycle across the boundary that maps onto the existing contract lifecycle through request ingress and artifact exchange (artifacts by digest, never bodies in payloads, never a foreign key in the keyring), opacity as the invariant (the other side sees offers, states and artifacts and nothing of the ledger), drilled against a second deployment in CI (two ledgers in one test, no shared write path), and the honest statement of what is not federated. Phase 13 opens when the Phase 12 exit record merges; plan now, implement as a draft until then (decisions/0003).

## Comment cm-cf05a331 (seed-next-implementer, 2026-09-03T06:27:24Z)



## Evidence ev-6581e9ca (receipt, seed-next-implementer, 2026-09-03T08:52:16Z)

receipts/os-40ed0ca0.json

## Evidence ev-c2a65141 (receipt, seed-next-implementer, 2026-09-03T09:52:14Z)

receipts/os-40ed0ca0.json

## Comment cm-3b4c9ade (seed-next-implementer, 2026-09-03T11:32:31Z)



## Comment cm-6ddca4fd (seed-next-implementer, 2026-09-03T12:00:43Z)


