---
id: os-9fc2c72f
title: 'next: the racing view''s settled-out claims vanish when the settled-out racers exit or are reaped'
state: backlog
priority: P2
squad: core
created_at: "2026-09-05T19:47:24Z"
---

Found by a read-only review of the tree at bfa01638 (2026-09-05). The contracts projection builds SettledOut from the fold's live claims (next/internal/project/contracts.go:311-320), and a claim-scoped exit drops the claim (next/internal/transition/transition.go:2548-2560, called at :1606-1618). So after a race settles, once the settled-out racers take their deliberate exits (or the maintenance pass reaps them) s.Claims is empty and the racing object's settled_out becomes empty, losing the identities the spec promises: "after settlement the settling position and the claims it settled out" (next/spec/lifecycle.md:202-208; plans/os-56bee171.md:100-106). Repro: settle a two-racer contract, read the contracts view (settled_out names the loser), reap or exit the loser, read again (settled_out is empty). This card: capture the settled-out claims into the fold at settlement (RaceSettled) rather than deriving them from live claims. Tier: standard. Plan-first.
