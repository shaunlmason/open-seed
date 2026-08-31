---
id: os-c6fb95ee
title: 'next: ledger writeHead race — unique temp file per writer'
state: done
priority: P2
squad: core
author: seed-next-implementer
review:
    reviewer: shaunlmason
    reviewed_at: "2026-08-31T22:53:25Z"
    outcome: accepted
    evidence: https://github.com/shaunlmason/open-seed/pull/161
created_at: "2026-08-31T21:34:48Z"
updated_at: "2026-08-31T22:53:25Z"
---

next/internal/ledger uses one shared HEAD.tmp path for the atomic HEAD rewrite, in both the append path and Open's HEAD-repair path. When a poll-only reader Opens mid-append (segment written, HEAD not yet renamed — the designed multi-process posture: workers poll while supervisors append), the repair path's rename consumes the appender's tmp and the appender fails with 'rename HEAD.tmp HEAD: no such file or directory'. Seen on main in open-seed#158's check job (TestGracefulPreemptionDrill, preempt_cli_test.go:220) and almost certainly the open-seed#156 verify make-check flake. Fix: per-writer unique temp names (os.CreateTemp in the store dir) so concurrent writeHeads never share a tmp; regression test hammering Append concurrently with Open. Plan-first (plans/<id>.md).

## Evidence ev-7efa88af (pr, seed-next-implementer, 2026-08-31T22:02:00Z)

https://github.com/shaunlmason/open-seed/pull/161
