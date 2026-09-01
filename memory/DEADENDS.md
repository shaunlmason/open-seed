# Dead ends

Append-only, via task PRs (D5). Approaches tried and abandoned, with the
reason, so the next agent doesn't burn a session rediscovering why.
- 2026-08-24 (os-501a29c2): **template variant repos** as the flavor mechanism.
  One repo per flavor makes each variant a fork that must track upstream
  itself, and `.seed/template.lock` carries exactly one `repo` line: a flavored
  instantiation could follow the variant (never seeing core releases) or the
  core (never seeing flavor updates), never both. Also multiplies the
  maintainer release checklist by the number of flavors. Rejected in ADR 0002.
- 2026-08-24 (os-501a29c2): an **engine-side `seed init --flavor`**. The engine
  is a separate Go repo on its own release train, pinned by
  `.seed/engine.lock`; flavor content there would version against the engine
  rather than the template it was installed into, so `seed template upgrade`
  could never reason about it. Rejected in ADR 0002.
- 2026-08-24 (os-501a29c2): the framing that **install alone is the update
  path** — that ordinary `seed template upgrade` merging keeps installed
  flavors current. It does not; the merge is by path and never reaches the
  root copies install made. This claim survived a full plan draft and was
  caught by both PR reviewers independently. If a mechanism copies files, ask
  where the *next* version of those files lands before claiming it tracks
  releases.
- 2026-08-24 (os-501a29c2): proving core purity with
  `git diff --quiet <merge-base> -- Makefile .seed ...` as a permanent test.
  Vacuous on `main` (merge-base is HEAD) and spurious on every later branch
  that legitimately touches those paths. The durable form is a behavioural
  invariant — the gate's output is unchanged when the flavor tree is removed —
  plus install confinement, neither of which depends on branch position.
- 2026-08-24 (os-2c0c474c): stopping paperclip's post-checkout
  `in_progress -> blocked` sweep by any route other than pausing the agent.
  Per-agent `runtimeConfig.heartbeat.enabled` is already `false` at create and
  the transition fires anyway (the mover is the recovery sweep, not the
  heartbeat); `pauseReason`/`pausedAt` are accepted and ignored (the field that
  matters is `status`); there is no heartbeat switch in `config.json` (only
  database/logging/server/telemetry/auth/storage/secrets); and every entry in
  `/api/adapters` is a real executor, so there is no inert adapter type to
  assign. What works is `PATCH /api/agents/<id> {"status":"paused"}` after
  create.
- 2026-08-24 (os-2c0c474c): trusting an experiment whose fixtures silently
  failed to exist. The first pause test passed `role: "eng"`, which is not in
  the role enum, so both agents 400'd, both checkouts 400'd, and both issues sat
  untouched at `todo`: a green-looking result that measured nothing. Assert that
  fixtures were actually created before drawing a conclusion from what happened
  to them.
- 2026-09-01 (os-ef715d17): refusals-only telemetry cannot produce a
  rate. The first 8.3 design journaled refusals and took the
  denominator from the chain over the journal's position span, which
  miscounts by construction (one refusal at tip 10 followed by a
  hundred clean appends reads 0.5000, not ~0.0099) and mixes an
  operator-local numerator with an all-actors denominator, so the
  metric would have measured unrelated ledger traffic rather than an
  affordance gap. Journal ATTEMPTS, both outcomes, at the same seams:
  numerator and denominator must come from one population.
- 2026-09-01 (os-768361cc): promoting at "core conformance" while
  skipping Phases 10 and 11. Contradicted by those phases' own exits
  (III.E, III.G, III.O; III.K), by the criterion demanding every
  pillar's mechanisms, and by Phase 12's `deps: all` — and it would
  have handed real coordination to a system with uncalibrated
  verdicts and unqualified grants. If an earlier supervised step is
  ever wanted it is a distinct milestone that names what it trades
  away, not a reinterpretation of the existing phasing.
- 2026-09-01 (os-7e197768): deriving a fence or reservation from a
  locally cached view before pushing to a remote. It reads correctly
  in every uncontended test and is wrong in exactly the case that
  makes claiming online-only: under contention the local copy is
  stale, so the derived citation names a window someone else closed.
  The fix was structural, not defensive — extract the remote session
  so the derivation and the admission pre-flight read one
  materialized remote tip, rather than reconstructing a view per
  verb and hoping they agree.
- 2026-09-01 (os-d6963652): auto-closing a reservation when its claim
  window ends, as a derivation rather than an act. It would free the
  stranded capacity with no new verb and no admission change, and it
  is exactly the fabrication the budget rule already refuses for
  unknown classes: it records ZERO spend for work that may have spent
  plenty, and destroys the distinction between "we spent nothing" and
  "nobody said". Capacity is returned by someone stating what was
  spent, or not at all.
- 2026-09-01 (os-d6963652): fixing the admission gate alone and
  expecting the affordance sweep to go green. The three budget probe
  synthesizers interpolated `"fence": "<v.fence>"` unconditionally,
  so outside a window they cite fence "0" and the FENCE rule refuses
  them before the budget rule is ever consulted — the sweep stayed red
  at precisely the prefixes the fix was for. A probe that invents a
  citation does not test the rule you think it tests; it tests the
  citation.
