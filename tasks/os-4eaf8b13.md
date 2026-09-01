---
id: os-4eaf8b13
title: 'next: the coverage flake looks deterministic because go test caches the bad profile'
state: cancelled
priority: P3
squad: core
created_at: "2026-09-01T13:12:43Z"
updated_at: "2026-09-01T20:49:05Z"
---

The nondeterministic coverage-counter loss (os-cafba959) LOOKS DETERMINISTIC when go test's result cache is warm, and that is what makes it dangerous rather than merely annoying.

Observed 2026-09-01 while implementing os-cf1c9688: `make check` reported "coverage 87.4% is below the 90% gate" three times in a row, identical to the tenth of a point. That reproducibility is what a real regression looks like, and it is why I initially concluded the drop was real and went looking for the code that caused it. Cold-cache runs of the same tree immediately afterwards gave 86.7%, 90.7%, 90.7%.

The mechanism: `go test` caches a package's result INCLUDING its coverage profile contribution. Re-running `make check` without clearing the cache replays the same merged profile, so a run that lost a package's counters keeps losing them, at the same number, indefinitely. The earlier sighting shows the same signature: 73.1%, 73.1% (both cached), then 61.5% after `go clean -testcache`.

Consequences for whoever takes os-cafba959, and for anyone diagnosing a coverage red before then:

1. "Re-run before believing it" is only sound if the re-run is COLD. `go clean -testcache && go test ...` is the discriminator; a bare re-run of `make check` proves nothing.
2. A stable coverage number across warm re-runs is NOT evidence the drop is real. Distinguishing a real regression from this flake requires several cold-cache runs, and the tell is variance (86.7 / 90.7 / 90.7), not repetition.
3. Whatever fix os-cafba959 lands should be validated cold-cache and repeatedly, or it will appear to work for the same reason the bug appears to be real.

This is a note on os-cafba959 rather than a separate defect; filed as its own card only so the diagnostic guidance is findable by someone hitting the red rather than buried in a PR thread. Close it with os-cafba959, or fold this text into that card and close this one.
