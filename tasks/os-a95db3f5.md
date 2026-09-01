---
id: os-a95db3f5
title: 'next: force-preemption drill polls for growth instead of a fixed window'
state: done
priority: P3
squad: core
author: seed-next-implementer
review:
    reviewer: shaunlmason
    reviewed_at: "2026-09-01T12:35:20Z"
    outcome: accepted
    evidence: https://github.com/shaunlmason/open-seed/pull/186
created_at: "2026-08-31T21:20:29Z"
updated_at: "2026-09-01T12:35:20Z"
---

TestForcePreemptionDrill asserts the deaf worker kept metering by comparing obs line counts across a fixed 300ms sleep; on a loaded runner under -p 1 atomic-coverage instrumentation the helper subprocess can boot slower than the window, flaking the assertion. Replace the fixed sleep with poll-until-growth (bounded deadline), the same robustness posture as the drills' other waits. Suspect in the open-seed#156 verify flake.

## Evidence ev-39d51b7f (, seed-next-implementer, 2026-09-01T08:42:57Z)


