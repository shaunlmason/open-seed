---
id: os-d76b7d6c
title: 'next: a herdr wake adapter for executors (backlog, adapter-only per §II.18)'
state: backlog
priority: P2
squad: core
labels:
    - next
created_at: "2026-09-05T18:48:15Z"
updated_at: "2026-09-06T01:20:31Z"
---

docs/next-build-plan.md §3 "Borrowed from surveyed tools", card 3. An executor adapter (charter §II.9: provision, wake, meter) whose advisory wake is `herdr agent prompt <name>` and whose blocked-state observation (`herdr agent wait <name> --until blocked`) is reported onto the observation stream so blocked(needs-you) reaches the operator. Adapter-only by charter rule: §II.18 forbids any coordination feature assuming a multiplexer; a worker without herdr loses latency and nothing else. Mock-total like every adapter (§II.13). Source: github.com/herdrdev/herdr. Not conformance-blocking. Above L1: plan first.

## Comment cm-dedb4cb8 (shaunlmason, 2026-09-06T01:20:31Z)

Scope correction (PR #336 review): herdr pane state (agent wait --until blocked) is liveness only, metered onto the observation stream as a wedge signal for the supervisor preemption path (charter §II.9). It never becomes an escalation. Escalation is a ledger event (§II.7: blocked(needs-you) carries packet, question, and minimal decision); the adapter surfaces admitted escalations to the operator pane over the wake channel and mints none from terminal state.
