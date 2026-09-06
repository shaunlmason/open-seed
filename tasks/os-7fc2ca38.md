---
id: os-7fc2ca38
title: 'next: trace-shaped test evidence in the receipt (backlog, §II.8 receipts, III.G evidence paths)'
state: in_progress
priority: P2
squad: core
labels:
    - next
claim:
    actor: seed-next-implementer
    token: c-f0e3d7241e2caad6
    claimed_at: "2026-09-06T06:51:45Z"
    lease_expires: "2026-09-06T08:11:49Z"
created_at: "2026-09-06T04:32:18Z"
updated_at: "2026-09-06T07:17:40Z"
---

docs/next-build-plan.md §3 "Borrowed from practice", card 1. Extend the receipt (next/spec/verdicts.md) with an optional `traces` entry per transcript: the trace id, the artifact-store digest of the spans the harness exported to a declared path, and the digest of the trace normalized shape (span tree by name, kind, status, declared attributes; ids, timestamps, durations stripped). The shape digest is what `verdict check` recomputes and what the L3 reproduces predicate compares; raw spans are bulk content in the artifact store by hash with the erasure path (charter §II.1 data classification). The verifier own run produces the trace (III.G, no implementer-claims channel): an implementer-attached trace is a claim, never a receipt input. Rubric verdicts cite a span as `trace:<id>/span:<id>`; the III.G evidence query joins verdict outcome to span status and outcome attributes. Not an observability subsystem: §II.18 forbids a second run log, so exporters and collectors are adapter details on the observation channel; a harness emitting no trace loses only the richer evidence. Source: the HerdrTestServer harness in herdr-sdk-development (2026-09-05). Not conformance-blocking. Above L1: plan first.

## Evidence ev-97dcfd4a (pr, seed-next-implementer, 2026-09-06T07:17:40Z)

https://github.com/shaunlmason/open-seed/pull/350
