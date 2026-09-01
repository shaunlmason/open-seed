---
id: os-a95db3f5
title: 'next: force-preemption drill polls for growth instead of a fixed window'
state: in_progress
priority: P3
squad: core
claim:
    actor: seed-next-implementer
    token: c-62e4b3a1a71e69be
    claimed_at: "2026-09-01T08:32:24Z"
    lease_expires: "2026-09-01T09:32:24Z"
created_at: "2026-08-31T21:20:29Z"
updated_at: "2026-09-01T08:32:24Z"
---

TestForcePreemptionDrill asserts the deaf worker kept metering by comparing obs line counts across a fixed 300ms sleep; on a loaded runner under -p 1 atomic-coverage instrumentation the helper subprocess can boot slower than the window, flaking the assertion. Replace the fixed sleep with poll-until-growth (bounded deadline), the same robustness posture as the drills' other waits. Suspect in the open-seed#156 verify flake.
