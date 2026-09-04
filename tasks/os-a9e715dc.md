---
id: os-a9e715dc
title: 'next: the refusal journal cannot tell a blind retry from a corrected one (III.R row 5''s blind-retry clause)'
state: blocked
priority: P2
squad: core
blocked_on:
    - plan:332
created_at: "2026-09-04T22:15:17Z"
updated_at: "2026-09-04T23:31:38Z"
---

plans/os-16e55c11.md D5 words the five-bar audit's guardrail bar as 'no refusal followed by a blind retry; every claim within its ceiling'. The ceiling clause is os-b5051f2e's. The blind-retry clause is not the chain's to show, since a refused append never lands, and review on #322 (chatgpt-codex-connector) found it is not the refusal journal's either as the journal stands: an entry keeps actor, verb, subject, outcome, code and position but no digest of the attempted payload, so an unchanged retry and a corrected retry write indistinguishable entries, and the report's refusal rate aggregates them further into counts. No surface can measure the clause today, and os-b5051f2e records it as explicitly unmet rather than routed. This card owns the measurement: the journal entry gains the attempted record's canonical-form digest (next/spec/refusals.md), the report's refusal-rate section counts a refusal followed by an attempt with the same digest by the same actor on the same subject as a blind retry, and the five-bar audit's description names that count as the clause's evidence. Bounds: the journal stays client-side and lossy by declaration; no chain content changes. Tier: standard. Plan-first.

## Evidence ev-d8975524 (pr, seed-next-implementer, 2026-09-04T23:22:42Z)

332

## Evidence ev-948bda6d (pr, seed-next-implementer, 2026-09-04T23:31:38Z)

333
