---
id: os-ebddb380
title: 'seed: a blocked card cannot gain a second blocked_on entry, though the model and the unblock path both expect several'
state: backlog
priority: P3
squad: core
created_at: "2026-09-04T11:57:56Z"
---

The card model treats `blocked_on` as a list and the unblock path drains it one entry at a time: `PlanUnblock` removes the `plan:<pr>` entry and only moves the card to `ready` when the list is empty (internal/task/maintain.go). So a card blocked on two things is a state the engine expects to exist.

Nothing can create it. The transition table (.seed/port-schema/transitions.json) carries exactly two edges into `blocked`, both from another state: `ready -> blocked` (verb block) and `in_progress -> blocked` (verb transition), each with the `add_blocked_on` effect. There is no `blocked -> blocked` edge, so a card that is already blocked refuses a second entry with `invalid_transition`.

Hit today on os-88df7ab2: the card was parked on `plan:309`, and its real second blocker (dep:os-b86dab4c, whose #306 renames the verb the card's switch reads) could not be recorded. The dependency had to live in the plan and in a card comment instead of in the field the ready queue actually reads, which means `seed task ready` would offer the card the moment its plan merged, while the dependency still held.

Expected shape: one `blocked -> blocked` edge with the `add_blocked_on` effect, in the operator class beside `block`, so the entry can be added without the card leaving `blocked` or losing the entries it has. The contract is data, so the change is the table plus the engine's mirrored copy under internal/spec/testdata and a round-trip drill that a card blocked twice needs both entries removed before it is ready. Note the engine also derives `blocked_on` legality from `blockedOnPattern` (`plan:<n>`, `dep:os-<hex>`, `manual:<text>`), which already anticipates the three kinds coexisting.

Touches .seed/port-schema (a protected path, owner review at merge) and open-seed-engine, so it is plan-first. Tier: standard. Filed 2026-09-04.
