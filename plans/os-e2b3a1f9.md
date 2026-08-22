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
   `PAPERCLIP_API_KEY`, `PAPERCLIP_COMPANY_ID`. Verb → API mapping:
   - `create` → POST issues (state backlog); priority P0–P3 →
     critical/high/medium/low
   - `ready` → GET issues filtered state=todo & unassigned
   - `get`/`list` → GET issue(s), states mapped todo→ready,
     in_review→review (others 1:1)
   - `claim` → the checkout endpoint (DB-atomic `checkoutRunId`);
     contention → exit 2 with holder; the **fence rides the per-agent
     bearer key** — the server itself refuses non-owner transitions, so
     exit 6 maps from the server's ownership refusal (stronger than any
     token we could mint)
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
   `none` — server is truth, budget **native**; optional: ancestry,
   budget), `backends.lock.json` in-template entry.
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
  2 with holder; a non-owner transition maps the server refusal to exit 6.
- `budget` is declared `native` in the manifest; the README documents the
  hard-stop semantics honestly (enforced by the server, not the repo).
- No engine changes required (the v0.7.0 dispatch seam is sufficient);
  filecards and beads behavior unchanged (validate.sh fully green).
- Every variance is declared in README + manifest, none discovered.

## Validation Commands

- `test -x .seed/backends/paperclip/bin/seed-backend`
- `sh .seed/backends/paperclip/test.sh`
- `sh scripts/validate.sh`
