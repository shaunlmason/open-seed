---
id: os-2ff8dbf1
title: 'next: observation streams v0, monotonic progress, expiry-vs-wedge in report (build plan 5.6)'
state: blocked
priority: P2
blocked_on:
    - plan:121
created_at: "2026-08-30T22:01:17Z"
updated_at: "2026-08-30T22:02:59Z"
---

Phase 5 item 6 (docs/next-build-plan.md): observation streams v0 + monotonic progress counts; expiry vs wedge detection in the report. Charter authority: Part II section 5 traffic classification (observations are ephemeral, lossy-by-declaration streams; the ledger records material transitions only), conformance III.F rows on traffic classification, monotonic progress counts, expiry-vs-wedge as distinct visible conditions, and lossy-by-declaration. Catalog verbs: progress.milestone (coarse, bounded frequency), wedge.declared. Defaults table binds the v0 channel: per-executor file under next/var/obs/ (gitignored). Plan-first: this card authorizes planning only until plans/<id>.md merges.
