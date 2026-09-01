---
id: os-8a5f14bb
title: 'next: Phase 9 item 3 — the unattended maintenance loop'
state: ready
priority: P1
squad: core
created_at: "2026-09-01T20:22:10Z"
updated_at: "2026-09-01T20:22:48Z"
---

Phase 9 item 3 of docs/next-build-plan.md. Above L1: plan first at
plans/<id>.md via its own PR.

Scope, quoted from the plan: "Maintenance loop: reap expired/wedged,
reconcile divergence, rebuild projections, checkpoint (signed), lints
- runnable unattended; audited as an ordinary actor."

Three obligations the plan attaches to it, each already load-bearing
somewhere else in the tree:

1. UNSETTLED-RUN DETECTION, inherited from the Phase 7 exit. A claim
   window carrying an admitted run.started whose run.settled is still
   missing once the subject has taken a subsequent claim window or
   reached a terminal state. Position-anchored, never mid park/reap
   flow: post-close settlement is a valid intermediate state, so a
   closed-without-settle predicate would file spurious findings. The
   obligations projection already emits KindRunUnsettled with exactly
   this anchoring - the lint consumes it rather than re-deriving it.

2. REAPING REQUIRES CORROBORATION BEYOND SILENCE. The observation
   channel is ephemeral and lossy by declaration (charter II.3,
   next/spec/observations.md), so a dropped stream and dead work look
   identical from outside, and no_data carries no reap path whatever.
   Item 1's loop emitting liveness from its own steps makes an expired
   classification BETTER EVIDENCE - it removes forgotten bookkeeping as
   a cause of silence - but does NOT make silence proof. The reap stays
   a judgment. No heartbeat predicate is added: non-advancing
   observations are not a heartbeat signature, since a legitimate
   long-running step emits that shape and the existing expiry/wedge
   classification already distinguishes it.

3. AUDITED AS AN ORDINARY ACTOR. The maintenance lane signs with its
   own key and every act it takes is admitted like anyone's. It already
   holds CapMaintenance for system.checkpoint, and keyring.go's comment
   records why that is not folded into operator: it would hand this
   loop halt and actor-management authority it must not have.

Two callers already name this lane's reap as their recovery path, and
the plan should honour both:

- next/spec/loop-verbs.md: a window stranded by a key rotation cannot
  be parked by the rotated key (the fence rule admits holder-signed
  events only from the holder), so "recovering it is the maintenance
  lane's reap".
- next/spec/escalation.md: a contract frozen by a persuaded lane
  raising a question nobody can answer is named as what this lane's
  lints should surface.

Open questions for the plan to decide and record:

1. Is the maintenance loop a LIBRARY like internal/loop, a CLI verb, or
   both? internal/loop is deliberately a library with no `seed loop
   run` verb, because Seed does not own the work. The maintenance lane
   DOES own its work, which is an argument the other way - but "seed
   maintain run" is a surface, and surfaces are decisions.
2. Which lints ship in v0, and what makes the list closed rather than
   open-ended? An open-ended lint set turns this into a policy surface.
3. What does a lint FINDING become - a filed defect contract (the
   charter says "files defect contracts"), an escalation, or a report
   row? Each has a different authority.
4. How is "runnable unattended" drilled without a scheduler? Item 4's
   fixtures run wakeless; this loop should be drillable the same way.
