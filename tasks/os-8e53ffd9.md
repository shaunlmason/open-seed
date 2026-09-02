---
id: os-8e53ffd9
title: 'next: Phase 10 item 1 — runtime tuples in enrollment and grants; adapters report the provisioned tuple; drift is out-of-grant'
state: backlog
priority: P1
squad: core
created_at: "2026-09-02T07:29:45Z"
---

Phase 10 (qualification and evaluation) item 1, the frontier named by
the Phase 9 exit record (os-e6cdb3d9, #214): runtime tuples in
enrollment/grants; adapters report the provisioned tuple; drift =
out-of-grant (docs/next-build-plan.md Phase 10 item 1; charter III.E).

Also carries the first half of III.J row 3, routed here by the exit
record: "the planner lane receives the strongest tuples by policy".

What the tree has today, to be measured by the plan rather than
assumed: actor.enrolled and actor.granted carry a key, a kind and a
capability; the run.* facts carry a runner; nothing names a runtime
tuple (model, version, harness, provider) anywhere, and out_of_grant
is capability absence only (next/internal/admit: "capability absence"
is explicitly a different property from disjointness, per
os-6a08b166). The plan must decide what a tuple IS as ledger data,
where it is declared (enrollment vs grant vs both), how an adapter
reports the tuple it actually provisioned, and what "drift" compares.

Plan-first: above L1, and the tuple is a new fact family that Phases
10 through 13 build on (eval contracts cite passing tuples; grants are
suspended on failure; independence levels are declared per tier).
