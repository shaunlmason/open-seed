---
id: os-f781f0da
title: 'next: Phase 9 item 2 — escalation with packet, question and decision'
state: blocked
priority: P1
squad: core
blocked_on:
    - plan:197
created_at: "2026-09-01T17:25:06Z"
updated_at: "2026-09-01T17:33:59Z"
---

Phase 9 item 2 of docs/next-build-plan.md, the next frontier item after
item 1 closed (1a #188, 1c #191, 1b #192, follow-ups #194 and #196).

Scope, quoted from the plan: escalation (blocked(needs-you)) with packet
+ question + decision; report surfaces age. An escalation raised in
answer to a refusal carries that refusal's code and message in its
packet, so the question a human is asked is the boundary's own account
rather than a lane's paraphrase of it.

Above L1: plan first at plans/<id>.md via its own PR.

What is already in the tree and must be consumed rather than
reinvented:

- Four-part packets and their validation (Phase 5.3, next/spec/packets.md).
  An escalation packet is a packet, not a new shape.
- The verbatim-refusal rule, already implemented for claim park in
  next/internal/loop: findings carry the boundary's code and message
  unchanged. Escalation must reuse that discipline, and the two should
  not grow separate copies of it.
- next/spec/loop-verbs.md's account of what a lane owes on a refusal it
  cannot act on: "a refusal it cannot act on is escalation's business
  (item 2), not a reason to reach past the boundary that gave it."
  That sentence is a promise this card pays.
- The envelope's journaled refusals and by_code breakdown (Phase 8),
  which is where age and volume become reportable.

Open questions this card must answer, per the plan's decision rubric,
and record in next/docs/decisions.md:

1. Is escalation a LEDGER VERB or a task-state transition? The
   transition table is the design authority for the former; check
   whether a needs-you state already exists there before adding one.
2. Where does the QUESTION live: the packet's decisions part, a new
   part, or a field on the escalation record? Adding a fifth part to a
   four-part packet is a charter-level change and almost certainly the
   wrong answer.
3. What renders the DECISION back to the lane, and how does the lane
   read it? The one-inbox doctrine says a lane acts on its
   position-stamped read, never on a wake, so the answer cannot be a
   notification.
4. What exactly does "report surfaces age" mean: age of the escalation
   since its position, and in which surface (situation, a projection,
   or a report verb)?

Acceptance: the escalation path is exercised end to end by a drill in
which a real refusal from the admission boundary becomes an escalation
whose packet carries that refusal's code and message VERBATIM, and the
drill fails if the paraphrase is substituted. make check green, receipt
generated at the content tip.
