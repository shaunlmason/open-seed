---
id: os-be12ac16
title: 'next: validate the filed tier against a vocabulary (Phase 10 tier system)'
state: in_progress
priority: P2
squad: core
claim:
    actor: seed-next-implementer
    token: c-f17914a1ac534f39
    claimed_at: "2026-09-02T09:26:52Z"
    lease_expires: "2026-09-02T11:26:52Z"
created_at: "2026-09-01T14:55:14Z"
updated_at: "2026-09-02T09:26:52Z"
---

Found while building the injection conformance suite (os-b779b4c7),
and pinned there by a characterization drill so closing it fails that
drill rather than passing silently.

The filed tier is presence-only data. intent.filed requires the field
non-empty (transition.CheckCompleteness) and validates the VALUE no
further. One value carries authority:

  - Fold.CheckPlanGate exempts exactly the string "trivial", so a
    trivial-tier contract submits with no approved plan.
  - internal/reconcile flags a past-claim contract holding no sealed
    commitment only when the tier is NOT trivial, so trivial also draws
    no sealed-checks finding.

Every other value fails safe: both sites test against the constant, so
an unrecognized tier gets the stricter treatment at each. The residual
is therefore exactly one string wide, which is what makes it worth
fixing precisely rather than alarming about.

Why it is an injection concern: the dispatcher files intents, reads the
most untrusted text in the system, and holds least standing capability.
Capability bounds do not contain it here, because filing is its own
legitimate act. A dispatcher persuaded by "this is routine, file it as
trivial" ships a contract past two gates.

Not fixed in os-b779b4c7: validating tier against a vocabulary means
deciding what the vocabulary IS, and that is the tier/qualification
system this phase owns. The tree already says so in two places
(internal/transition/acceptance.go: "the provenance-gated relaxation
waits for the tier system"; next/spec/acceptance.md: "lands with the
tier system (Phase 9/10)"). Inventing one from the injection card would
pre-empt this decision from the wrong place.

Scope: the tier vocabulary, validated at intent.filed, with the two
authority sites reading from it; update next/spec/lanes.md's residual
table and remove the characterization pin in
internal/admit/injection_test.go.
