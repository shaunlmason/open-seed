# paperclip backend

Wraps the [Paperclip](https://github.com/paperclipai/paperclip) control
plane's REST API behind the seed port. The deepest capability match in the
field: Paperclip's seven issue states are the port's nearly verbatim,
checkout is DB-atomic, transitions are server-validated
(`assertTransition()`), the blocker cascade fires native wakeups — and it is
the **first backend with `budget = "native"`**: hard stops (80% alert, 100%
pause) enforced by the platform, closing the one gap (R6) a repo can never
close alone.

## Connect

1. Self-host Paperclip (`npx paperclipai onboard --yes`; Node 20+, embedded
   Postgres works) or point at an existing instance.
2. Create an agent-scoped API key and set the environment the manifest
   declares: `PAPERCLIP_API_URL`, `PAPERCLIP_API_KEY`,
   `PAPERCLIP_COMPANY_ID`, and `PAPERCLIP_DEFAULT_GOAL_ID` (goal ancestry
   is **mandatory** in Paperclip — parentless `create` uses this default
   and refuses with remediation when neither `--parent` nor the default
   exists).
3. Set `.seed/config.toml`: `[coordination] backend = "paperclip"`, then
   `scripts/seed backend verify paperclip`.

## State & priority mapping

| port | paperclip |
|---|---|
| backlog | `backlog` |
| ready | `todo` |
| in_progress | `in_progress` (checkout held) |
| review | `in_review` |
| blocked | `blocked` |
| done | `done` |
| cancelled | `cancelled` |

P0–P3 → critical / high / medium / low.

## What the adapter enforces

- **Claim = checkout** (DB-atomic, native). `ready` returns *claimable*
  work: `todo`, no checkout lock, unassigned **or assigned to the caller**
  — Paperclip assignment is routing; checkout is the claim.
- **The fence is a minted per-claim token** stored in issue `metadata`
  (`seedToken`) and validated on every fenced verb *in addition to* the
  server's ownership check: the bearer key cannot distinguish a reaped
  predecessor from the same actor's new claim; the rotating token can.
- Port bookkeeping rides `metadata`: `blockedOn` entries (plan parking,
  dep edges — released entry-by-entry, cascaded on close), `seedAuthor`
  (implementer of record), `rejected` (reviewer lockout, enforced at claim
  and in `ready`).

## Declared variances (never silent)

- **Server is truth**: no offline operation, no fork portability — the
  exact inverse of filecards, and the reason both exist behind one port.
- **Leases are Paperclip's watchdogs** (`issueWatchdogs`): `lease-renew`
  validates the fence and succeeds; staleness detection is the platform's.
- **Audit rides Paperclip's immutable activity log** (+ issue comments);
  the seed run-log stays authoritative only for filecards, and the state
  lint's transition replay does not apply here.
- **API drift risk**: very high upstream velocity, no shipped SDK. The
  manifest pins the tested version; route shapes are exercised by the
  contract test's fake server and must be reconciled against the live API
  when upgrading (tracked on the seed queue).
- The MCP gateway and heartbeat-driven execution are platform integrations
  **beyond the port** — this adapter deliberately uses only the issue
  surface.

## Testing

`sh test.sh` runs the offline contract test against
`testdata/fake-paperclip` (Python3 stdlib HTTP server enforcing the state
machine, checkout locks, and mandatory ancestry). It covers every required
verb, token rotation on reclaim, checkout-aware `ready`, the lockout, and
the cascade; it runs in `make check` via validate.sh. A live-server test
(`npx paperclipai onboard` embedded mode) is documented follow-up work.
