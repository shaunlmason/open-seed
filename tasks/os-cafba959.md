---
id: os-cafba959
title: 'next: the coverage gate fails nondeterministically, reporting far below truth'
state: in_progress
priority: P2
squad: core
claim:
    actor: seed-next-implementer
    token: c-96cf48e2644fb59e
    claimed_at: "2026-09-01T19:03:54Z"
    lease_expires: "2026-09-01T20:03:54Z"
created_at: "2026-09-01T10:34:01Z"
updated_at: "2026-09-01T19:37:51Z"
---

The coverage gate in `make check` (check-next) fails nondeterministically, reporting a total far below truth on a tree that is fine. Observed repeatedly on 2026-09-01 while merging main into several branches:

    check-next: coverage 73.1% is below the 90% gate
    check-next: coverage 61.5% is below the 90% gate
    check-next: coverage 62.2% is below the 90% gate

...on trees that give a stable 90.4% on the next run, cold cache, with no change at all. Every test PASSES in the bad runs; only the merged profile is short.

This is the failure mode the Makefile's own comment already names:

    # -p 1 serializes package test binaries: concurrent binaries under the
    # subprocess-heavy drills can collide coverage counter files (same pid
    # and second after heavy pid recycling), silently dropping one package
    # from the merged profile and misreading coverage far below truth.

So `-p 1` reduced the collision rate but did not eliminate it. Diagnosis on one bad profile: the deficit is concentrated in internal/admit/admit.go, internal/admit/affordances.go, internal/obligation/obligation.go and internal/project/obligations.go, which is consistent with one or more test binaries' contributions being dropped rather than with any code change.

WHY THIS MATTERS MORE THAN A NORMAL FLAKE. The failure presents as 'your change dropped coverage 28 points', which is exactly what a real regression looks like. An unattended agent seeing it will go hunting for a coverage regression it did not cause, and the correct response (re-run once, then treat a second failure as real) is a rule it has to apply against its own instinct. That is the same argument os-c4e8b57a made for the auto-gc cleanup race, and the same reasoning applies: removing the race is cheaper than paying the diagnosis cost on every future red.

It also makes CI's verdict on any PR probabilistic, since the same gate runs there.

LIKELY FIX. Go 1.20+ supports GOCOVERDIR-based coverage collection (go test -cover with -args, or `go build -cover`), which writes per-process counter files into a directory instead of merging via the legacy -coverprofile path, and does not have the pid-collision failure mode. Alternatives: give each package's profile a distinct output path and merge explicitly with `go tool covdata`, or set GOCOVERDIR for the subprocess-spawning drills. Worth measuring the collision rate before and after so the fix is evidenced rather than assumed.

Scope: the Makefile's check-next target, possibly a small helper script, and whatever the drills need so subprocess re-execs do not collide. No production code.

## Evidence ev-f523fd94 (pr, seed-next-implementer, 2026-09-01T19:37:51Z)

https://github.com/shaunlmason/open-seed/pull/201
