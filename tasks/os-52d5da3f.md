---
id: os-52d5da3f
title: 'next: obligations projection, situation read, and loop verbs (Phase 9 item 5)'
state: blocked
priority: P1
squad: core
blocked_on:
    - plan:170
created_at: "2026-09-01T03:18:27Z"
updated_at: "2026-09-01T04:05:35Z"
---

Implements the lane-facing surface named by the promotion amendment (plan #167, build plan Phase 9 item 5). THE IDEA: Seed represents permission (affordances, 8.1/8.2) but not OBLIGATION — nothing answers what is owed, by whom, since when, under what clock, and which verbs discharge it, although every fact is already folded (an active claim with its lease, a submission awaiting a verdict, an admitted run.started with no run.settled, a contract blocked awaiting a human, an open reservation, a red gate). THREE DELIVERABLES, one story. (1) OBLIGATIONS PROJECTION: a projection like every other — deterministic, byte-identical, position-stamped, rebuildable, NON-AUTHORITATIVE, derived from the same fold admission enforces — emitting per (subject, actor) rows of {kind, owed_by, since_position, due_at or clockless, discharged_by: [verbs]}. It invents no legality: discharged_by comes from the transition tables, so an obligation whose discharging verb is refused at the same position is the III.I row-2 bug class one level up, and gets the same regression-class treatment as 8.2's sweep. (2) SITUATION READ: seed situation --key <k> [--subject s] [--since <position>] returning one position-stamped envelope: my standing obligations with clocks, my active windows and fences, unread messages, budget headroom, and — with --since — ONLY what changed since that position, so a resuming lane pays for the delta rather than reconstructing the world (the append-only history makes this a tail read; today every cold start re-derives everything). Read-only, idempotent, no state change. (3) LOOP VERBS: the acts a lane takes, with every derivable argument derived (fence from the active window, reservation id from the budget view, base range from the repository, submission position from the fold) and every refusal raised BEFORE signing, naming the act that would succeed — the same one-rule-set principle Phase 8 landed for legality, applied to argument construction, so the CLI makes the legal act cheap and the illegal act hard to express. Scope guard: no new authority, no new source of truth, no orchestration layer, success envelopes stay terse (teaching text lives in refusals only). Plan-first (L2); the plan decides the obligation-kind taxonomy, the exact envelope shape, and which verbs the surface covers first.
