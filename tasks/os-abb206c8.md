---
id: os-abb206c8
title: 'next: Phase 9 item 1c — the worker loop made executable, with exhaustion parking'
state: done
priority: P2
squad: core
author: seed-next-implementer
review:
    reviewer: shaunlmason
    reviewed_at: "2026-09-01T15:09:21Z"
    outcome: accepted
    evidence: https://github.com/shaunlmason/open-seed/pull/191
created_at: "2026-09-01T13:28:53Z"
updated_at: "2026-09-01T15:09:21Z"
---

Phase 9 item 1, third of three cards (1a landed as #188; 1b is the dispatcher's injection conformance suite, still uncarded).

This is the one promotion criterion 1 needs: a lane that cannot run the loop cannot run unattended, whatever the conformance report says.

SCOPE. The worker lane's loop, made EXECUTABLE, with exhaustion parking: a budget refusal at a spending gate triggers the claim.parked exit with its packet (the III.H row the Phase 7 exit routes here), consuming Phase 8's envelope budget block. The loop runs poll, claim, work, meter, sync, deliberate exit, acting only through the item 5(c) loop verbs and orienting only from the single position-stamped read.

INHERITED FROM 1a, recorded in next/docs/progress.md and next/spec/lanes.md. 1a could check that a lane's declared liveness_from entries are work steps it performs; it could NOT check that running them emits anything, because nothing executes at 1a — manifests are data. 1c must close that:
- drill that running the declared liveness steps advances the observation stream keyed to the lane's actor and fence;
- drill that the loop reaches no liveness-only surface, so the vocabulary really does contain no verb whose only purpose is to report liveness.
Without those, the third obligation is asserted by a manifest and proven by nobody.

WHAT ALREADY EXISTS AND MUST BE CONSUMED RATHER THAN REBUILT:
- internal/loopverb, the registry of the seven acts (#188).
- internal/lane, the manifests and their validation (#188); the implementer manifest already declares acts_through and liveness_from, so the loop's shape is DECLARED and the executable loop must match its own manifest, which is itself a checkable property.
- The loop verbs themselves (#173, fixed by #181): they derive the fence, the reservation, the plan anchor and the resume range, and pre-flight through admit.Check.
- seed situation with --since (#171), the orienting read.
- The envelope budget block (Phase 8) and BudgetViewAt.
- The observation channel and its expiry/wedge classification (internal/obs).

OPEN QUESTIONS FOR THE PLAN:
- Where the loop LIVES. next/executor exists; is the loop a package there, a cmd/seed verb (`seed loop run`), or a library a harness drives? A verb is testable end to end but invites someone to treat it as the agent; a library keeps the model out of the loop, which the fixtures need.
- How the loop is drilled without a model. The preemption drills already re-exec the test binary as a helper worker (next/cmd/seed/preempt_cli_test.go), which is the established pattern and probably the right one.
- What exhaustion parking does with the packet's acceptance and findings when the refusal is the only thing known.
- Whether the loop asserts its own manifest (read internal/lane, act only through the declared acts) or merely happens to agree with it. Asserting it is stronger and is the natural close of 1a's loop.

## Evidence ev-70588c6f (pr, seed-next-implementer, 2026-09-01T14:41:54Z)

https://github.com/shaunlmason/open-seed/pull/191
