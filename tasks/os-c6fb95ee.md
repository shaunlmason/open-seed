---
id: os-c6fb95ee
title: 'next: ledger writeHead race — unique temp file per writer'
state: in_progress
priority: P2
squad: core
claim:
    actor: seed-next-implementer
    token: c-57ffd8a23938aa59
    claimed_at: "2026-08-31T21:35:21Z"
    lease_expires: "2026-08-31T22:35:21Z"
created_at: "2026-08-31T21:34:48Z"
updated_at: "2026-08-31T21:35:21Z"
---

next/internal/ledger uses one shared HEAD.tmp path for the atomic HEAD rewrite, in both the append path and Open's HEAD-repair path. When a poll-only reader Opens mid-append (segment written, HEAD not yet renamed — the designed multi-process posture: workers poll while supervisors append), the repair path's rename consumes the appender's tmp and the appender fails with 'rename HEAD.tmp HEAD: no such file or directory'. Seen on main in open-seed#158's check job (TestGracefulPreemptionDrill, preempt_cli_test.go:220) and almost certainly the open-seed#156 verify make-check flake. Fix: per-writer unique temp names (os.CreateTemp in the store dir) so concurrent writeHeads never share a tmp; regression test hammering Append concurrently with Open. Plan-first (plans/<id>.md).
