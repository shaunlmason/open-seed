---
id: os-d76b7d6c
title: 'next: a herdr wake adapter for executors (backlog, adapter-only per §II.18)'
state: backlog
priority: P2
squad: core
labels:
    - next
created_at: "2026-09-05T18:48:15Z"
---

docs/next-build-plan.md §3 "Borrowed from surveyed tools", card 3. An executor adapter (charter §II.9: provision, wake, meter) whose advisory wake is `herdr agent prompt <name>` and whose blocked-state observation (`herdr agent wait <name> --until blocked`) is reported onto the observation stream so blocked(needs-you) reaches the operator. Adapter-only by charter rule: §II.18 forbids any coordination feature assuming a multiplexer; a worker without herdr loses latency and nothing else. Mock-total like every adapter (§II.13). Source: github.com/herdrdev/herdr. Not conformance-blocking. Above L1: plan first.
