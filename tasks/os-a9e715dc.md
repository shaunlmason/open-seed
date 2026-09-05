---
id: os-a9e715dc
title: 'next: the refusal journal cannot tell a blind retry from a corrected one (III.R row 5''s blind-retry clause)'
state: in_progress
priority: P2
squad: core
claim:
    actor: seed-next-implementer
    token: c-66ce4d0a01957e61
    claimed_at: "2026-09-05T00:48:39Z"
    lease_expires: "2026-09-05T01:48:39Z"
created_at: "2026-09-04T22:15:17Z"
updated_at: "2026-09-05T00:48:39Z"
---

plans/os-16e55c11.md D5 words the five-bar audit's guardrail bar as 'no refusal followed by a blind retry; every claim within its ceiling'. The ceiling clause is os-b5051f2e's. The blind-retry clause is not the chain's to show, since a refused append never lands, and review on #322 (chatgpt-codex-connector) found it is not the refusal journal's either as the journal stands: an entry keeps actor, verb, subject, outcome, code and position but no digest of the attempted payload, so an unchanged retry and a corrected retry write indistinguishable entries, and the report's refusal rate aggregates them further into counts. No surface can measure the clause today, and os-b5051f2e records it as explicitly unmet rather than routed. This card owns the measurement: the journal entry gains the attempted record's canonical-form digest (next/spec/refusals.md), the report's refusal-rate section counts a refusal followed by an attempt with the same digest by the same actor on the same subject as a blind retry, and the five-bar audit's description names that count as the clause's evidence. Bounds: the journal stays client-side and lossy by declaration; no chain content changes. Tier: standard. Plan-first.

## Evidence ev-d8975524 (pr, seed-next-implementer, 2026-09-04T23:22:42Z)

332

## Evidence ev-948bda6d (pr, seed-next-implementer, 2026-09-04T23:31:38Z)

333

## Comment cm-e0c9d8a8 (seed-next-implementer, 2026-09-04T23:33:41Z)

Plan PR #332 (amended for its two review findings: D3 reads modes.md's blind-retry definition, the same act refused with the same code from an unadvanced position, an admitted next attempt being convergence; the report version moves 17 to 18). Task PR #333 (draft until the plan merges; verify refuses a task PR whose plan is not at the branch point). make check was green on the first cut (coverage 91.6%); re-running on the amended tree.

## Comment cm-a1fe22d9 (seed-next-implementer, 2026-09-04T23:52:44Z)

make check green on seed/os-a9e715dc at b4272532 (validate ok; check-next gofmt/vet/build/test ok, coverage 91.6% over the 90% gate; perf under ceilings). The first CI run of #333 failed only in the new terminal drill's cleanup: the projection tree it rebuilt is locked read-only on publish and the temp dir could not remove it on the runner's tmpfs; the drill now unlocks the tree in its cleanup, the project drills' posture. #333 CI re-running; verify stays red until #332 merges (D3).
