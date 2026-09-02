---
id: os-03e47abb
title: 'next: Phase 10 item 2 — eval contracts through the production machinery; grants cite passing tuples; scheduled spot-checks; suspension on failure'
state: in_progress
priority: P1
squad: core
claim:
    actor: seed-next-implementer
    token: c-d6f465c6a5c42c61
    claimed_at: "2026-09-02T09:31:42Z"
    lease_expires: "2026-09-02T12:07:59Z"
created_at: "2026-09-02T08:46:00Z"
updated_at: "2026-09-02T10:07:59Z"
---

Phase 10 (qualification and evaluation) item 2, the frontier after item 1
(os-8e53ffd9, #216): eval contracts against fixture repos through the
production machinery; grants cite passing tuples; scheduled spot-checks;
suspension on failure (docs/next-build-plan.md Phase 10 item 2; charter
§II.5 "spot-check evals re-test active tuples; drifted or failing tuples
get grants suspended by the supervisor — an attributable event, no
operator ceremony"; §II.16 "Eval contracts: synthetic work with known
verdicts, fixture repos, run through the production machinery; passing
gates grants for the tuple that ran them; scheduled spot-checks re-test
active tuples"; conformance III.E rows 6–7 and III.O row 1).

What the tree has after item 1, to be measured by the plan rather than
assumed: the five-field runtime tuple (`internal/tuple`), grants citing
tuples at seed/2 (`actor.granted` with `tuple`), `run.started`
declaring one and the set rule refusing drift as out_of_grant, `Provision`
holding the adapter to the declaration, offers scoping by `tuples`;
`actor.qualified` cataloged and refused by name ("cites eval results",
protocol.md; undefined until item 2, actors.md and qualification.md);
`actor.suspended` accepted from `operator` only; the verdict pipeline
(receipt, gate-before-run, render derives from transcripts, red lockout)
and the maintenance pass (reap, lint, file, rebuild, checkpoint; wakeless,
no private powers) as the production machinery an eval must run through;
fixture repositories only as test helpers (`verdictRepo`). Nothing today
mints a grant from a verdict, re-tests a tuple on a schedule, or lets a
supervisor suspend a grant.

The plan must decide, per the build plan's decision rubric and record in
the plan PR: what an eval contract IS on the chain (a contract with a known
verdict against a fixture repository: how it is marked, who files it, how
"known verdict" is expressed so the machinery cannot be gamed); what
`actor.qualified` cites (the verdict, the receipt, the tuple the run
declared) and whether it mints the grant itself or an `actor.granted`
follows it; what "scheduled spot-check" means in a wakeless system (the
maintenance pass is the only unattended loop; a spot-check is presumably a
re-filed eval contract, due by policy, not by timer); what "suspension on
failure" suspends (the grant, per tuple, rather than the key's standing:
the charter says "grants suspended", and a key with two qualified tuples
keeps the other) and who signs it (the supervisor, "no operator
ceremony", which means a capability row moves or a new grant-level verb
lands); and the protocol consequence (a new actor verb or a new grant
state is a seed/3 candidate under actors.md's chain-validity posture, to
be judged by protocol.md's bump discipline, not assumed).

Depends on #216 (item 1) merging; plan against its surfaces and say so.
Scope guard: no ranking engine ("strongest tuples" ranking is offers'
policy input, item 1); calibration (item 4) and levels (item 3) are
separate cards.
