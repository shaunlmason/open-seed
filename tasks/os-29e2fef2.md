---
id: os-29e2fef2
title: 'next: prefer the prior submitter''s tuple when re-offering a contract returned on the forge''s word'
state: in_progress
priority: P2
squad: core
claim:
    actor: seed-next-implementer
    token: c-1cb696fd976a383b
    claimed_at: "2026-09-06T09:33:07Z"
    lease_expires: "2026-09-06T10:33:07Z"
created_at: "2026-09-06T07:24:20Z"
updated_at: "2026-09-06T09:33:07Z"
---

Follow-up named by plans/os-0cd18799.md D8 and next/docs/decisions.md (os-0cd18799): a contract returned by contract.returned citing a check.observed (the forge says the submission is not mergeable) records no lockout, so the prior submitter is the natural next claimant, but nothing prefers it. Supervisor ranking policy (Phase 13 item 7, next/spec/ranking.md): when publishing the re-offer for a subject whose latest return cited an observation, rank the tuple that made the returned submission first. Charter §II.9. Not conformance-blocking.
