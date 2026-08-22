# Plan: Paperclip backend adapter (os-e2b3a1f9)

Wrap the Paperclip control plane (paperclipai/paperclip; research/10 Part 1)
behind the port. The deepest capability match in the field: its seven issue
states are ours almost verbatim, checkout is DB-atomic with server-enforced
owner/board release (our worker/operator split), blocker resolution fires
native wakeups, goal ancestry is mandatory, and budgets have **hard stops** —
the first backend where the `budget` capability is `native` (R6's missing
enforcement, supplied by a platform through the seam built for it).

## Steps

1. **Adapter** — `.seed/backends/paperclip/bin/seed-backend` (POSIX sh +
   curl + jq), REST against `$PAPERCLIP_API_URL` with the agent-scoped
   bearer key. Env via manifest `requires_env`: `PAPERCLIP_API_URL`,
   `PAPERCLIP_API_KEY`, `PAPERCLIP_COMPANY_ID`, `PAPERCLIP_DEFAULT_GOAL_ID`. Verb → API mapping:
   - `create` → POST issues (state backlog); priority P0–P3 →
     critical/high/medium/low. **Goal ancestry is mandatory in Paperclip**,
     but the port allows parentless creates: the adapter resolves a
     configured default (`PAPERCLIP_DEFAULT_GOAL_ID`, or a project id) when
     no `--parent` is given, and refuses with exit 5 + remediation when
     neither exists. The contract test covers parentless creation both ways.
   - `ready` → **claimable work, not merely unassigned work**: state todo,
     **no checkout lock**, and (unassigned OR assigned to the calling
     actor) — Paperclip assignment is routing, checkout is the claim, and
     conflating them would hide assigned-but-unclaimed work while showing
     issues the caller cannot take. Rejected-author filtering applies as
     below, and the predicate also requires **no open blockers of either
     kind**: no seed `blockedOn` entries *and* no nonterminal native
     Paperclip dependency (`blockerIds` — deps created in the Paperclip
     UI gate claimability too). Contract case: a todo issue with an
     unresolved native blocker is absent from `ready` and appears when
     the blocker reaches a terminal state.
   - `get`/`list` → GET issue(s), states mapped todo→ready,
     in_review→review (others 1:1)
   - `claim` → the checkout endpoint (DB-atomic `checkoutRunId`);
     contention → exit 2 with holder. **The fence is a minted per-claim
     token, not the bearer key alone** (D1 binding): the adapter generates
     a fresh token at claim, persists it with the issue (metadata field,
     or a marker comment where metadata is unavailable), and every fenced
     verb validates the presented token against the stored one *in
     addition to* the server's ownership check — the bearer key cannot
     distinguish a reaped predecessor from the same actor's new claim;
     the rotating token can. Stale or missing tokens exit 6. Checkout +
     mint form **one exclusive claim**: a held checkout is contention
     (exit 2) for every caller, the same actor included, so no second
     process can silently rotate a live token. **Declared variance
     (never silent): the fence is check-then-act, not atomic** —
     Paperclip has no conditional metadata update, so validate-then-
     mutate spans two requests and a worker reaped and superseded inside
     that one-round-trip window can still land one mutation past the
     platform's own same-actor ownership check. The backend declares it
     cannot provide an atomic fence (filecards remains the backend that
     does); the README carries this variance.
   - `transition`/`release`/operator verbs → state PATCH; the server's
     `assertTransition()` refusal maps to exit 3
   - `close` → done/cancelled transition; blocker cascade is native
     (`issue_blockers_resolved` wakeups)
   - `comment` → comments; `attach-evidence` → comment with evidence
     prefix (document attachments are a later refinement)
   - `lease-renew` → no-op success with a note: staleness is
     `issueWatchdogs`' job (declared variance)
   - `event-append` → comment on the task; Paperclip's immutable activity
     log is the audit substrate (declared, as with beads)
   - Exact route shapes are confirmed against the Mintlify docs at
     implementation time; the manifest pins the tested Paperclip version
     and the README declares API-drift risk (very high upstream velocity,
     no shipped SDK).
2. **Manifest + lock**: `backend.toml` (atomic_claim native, offline
   `none` — server is truth, budget **native**; optional: lease-renew,
   ancestry, budget), `backends.lock.json` in-template entry. **Every
   advertised optional capability ships its handler**: `ancestry <id>`
   walks the parent-issue chain up to the goal, and `budget <id>` reports
   the issue's goal budget as the platform enforces it (ok / alert at
   80% / paused at 100%) — both covered by the contract test, so
   capability negotiation never selects an unimplemented verb.
3. **Contract test** — `test.sh` + `testdata/fake-paperclip` (a small
   Python3 `http.server` implementing the API subset with in-memory state,
   started on a random localhost port). Exercises every required verb's
   envelope and exit-code mapping, including checkout contention and a
   non-owner transition refusal. Wired into `scripts/validate.sh` behind a
   `command -v python3` guard. **Stretch, not gating**: a live-server test
   via `npx paperclipai onboard --yes` (embedded Postgres) documented in
   the README and tracked as a follow-up card if it can't run in CI.
4. **README**: install/connect (self-host, agent API key creation),
   state/priority mapping tables, declared variances (server-is-truth — no
   offline, no fork portability; leases are watchdogs; state-lint replay
   N/A; roster enforcement rides Paperclip's board/agent permissions), and
   what the adapter deliberately does NOT use (heartbeat-driven execution,
   MCP gateway — those are platform integrations beyond the port).
5. **Handbook §6**: add the control-plane rung — when budgets must be
   enforced rather than advised, point at this backend.

## File Scope

- `.seed/backends/paperclip/**`, `.seed/backends.lock.json`,
  `scripts/validate.sh`, `docs/handbook.md`

## Acceptance Criteria

- All nine required verbs round-trip against the fake server with valid
  envelopes and mapped exit codes (0/2/3/4/5/6); checkout contention exits
  2 with holder; a non-owner transition maps the server refusal to exit 6,
  and a **stale minted token from a reaped-then-reclaimed same-actor
  session also exits 6** (token rotation covered by the contract test).
- `ready` returns assigned-but-unclaimed work for the caller and never
  returns checked-out or other-actor-assigned issues.
- Parentless `create` succeeds via the configured default goal and fails
  with remediation without one.
- `budget` is declared `native` in the manifest; the README documents the
  hard-stop semantics honestly (enforced by the server, not the repo).
- No engine changes required (the v0.7.0 dispatch seam is sufficient);
  filecards and beads behavior unchanged (validate.sh fully green).
- Every variance is declared in README + manifest, none discovered.

## Validation Commands

- `test -x .seed/backends/paperclip/bin/seed-backend`
- `sh .seed/backends/paperclip/test.sh`
- `sh scripts/validate.sh`
