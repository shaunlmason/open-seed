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
