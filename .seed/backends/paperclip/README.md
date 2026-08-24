# paperclip backend

Wraps the [Paperclip](https://github.com/paperclipai/paperclip) control
plane's REST API behind the seed port. The deepest capability match in the
field: Paperclip's seven issue states are the port's nearly verbatim,
checkout is DB-atomic, transitions are server-validated
(`assertTransition()`), the blocker cascade fires native wakeups, and it is
the **first backend with `budget = "native"`**: hard stops (80% alert, 100%
pause) enforced by the platform, closing the one gap (R6) a repo can never
close alone.

## Connect

1. Self-host Paperclip (`npx paperclipai onboard --yes`; Node 20+, embedded
   Postgres works) or point at an existing instance.
2. Create an agent-scoped API key and set the environment the manifest
   declares (a quickstart instance runs `local_trusted` and serves the
   API unauthenticated over loopback, where any placeholder key works;
   an `authenticated` deployment needs a real one):
   `PAPERCLIP_API_URL`, `PAPERCLIP_API_KEY`,
   `PAPERCLIP_COMPANY_ID`, and `PAPERCLIP_DEFAULT_GOAL_ID` (goal ancestry
   is **mandatory** in Paperclip: parentless `create` uses this default
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
: Paperclip assignment is routing; checkout is the claim. Results are
  normalized to the port vocabulary (P0–P3) and **sorted by priority**
  (the loop claims the first entry). `ready` also excludes issues with a
  **nonterminal native blocker** (`blockedBy`: deps created in the
  Paperclip UI count, not just seed's own `blockedOn` entries);
  `create --blocked-by` mirrors each edge into both, writing the native
  side as `blockedByIssueIds`. The issue *list* projection omits
  `blockedBy`, so `ready` re-reads each candidate rather than trusting
  the list: one extra round trip per candidate, paid for correctness.
- **A held checkout is contention for everyone: the same agent included**
  (exit 2): checkout + token mint form one exclusive claim, so a repeated
  same-owner checkout may never silently succeed (two processes of one
  actor would both "win" and the second would invalidate the first's
  fence). The adapter pre-checks the holder; the atomic arbiter is the
  server, whose checkout endpoint must 409 on *any* held lock: the
  contract the fake encodes; reconcile against the live API on upgrades.
- **`--parent` is a task id, never a goal id**: the child inherits the
  parent issue's goal for the mandatory ancestry and keeps `parentId`;
  the optional `ancestry` verb walks child → parents → goal, and the
  optional `budget` verb reports the platform-enforced budget
  (`ok` / `alert` at 80% / `paused` at 100%: the native R6 stop).
  **Scope correction (os-2c0c474c):** that budget is the **company's**,
  not the goal's. Paperclip enforces budgets at company and agent scope;
  a goal object carries no budget fields at all. The hard stop is real
  and still closes R6, but the port asks about a task and the honest
  answer is the enclosing company's envelope, reported as
  `budget.scope = "company"`.
- **The fence is a minted per-claim token** stored in the issue's `seed`
  document (`seedToken`) and validated on every fenced verb *in addition
  to* the server's ownership check: the bearer key cannot distinguish a
  reaped predecessor from the same actor's new claim; the rotating token
  can. The write is compare-and-swap against the revision validation
  read, which is what makes it atomic (see variances).
- **`--actor` is resolved to a Paperclip agent**, because checkout
  requires an agent row's UUID and the port hands over a free-form name.
  Resolution order: `PAPERCLIP_AGENT_MAP="actor=<uuid>,..."` if set, then
  the company's agent roster matched on `name` or `urlKey`. Both failure
  modes refuse rather than guess: **no match** exits 5 naming the roster
  and the override, **ambiguous match** exits 5 rather than picking a
  winner. Identity is checked before contention, so an unregistered actor
  is told so even when the card is held by someone else.
- Port bookkeeping rides the per-issue **document store** at key `seed`
  (issues have no `metadata` field: a PATCH carrying one is accepted and
  silently dropped): `blockedOn` entries (plan parking,
  dep edges: released entry-by-entry, cascaded on **close and cancel**
  alike, per the transition table), `seedAuthor` (implementer of record),
  `rejected` (reviewer lockout, enforced at claim and in `ready`).
  Server-arbitrated state PATCHes are checked **before** bookkeeping
  mutates: a refused transition never burns the claim token or reports
  success.

## Declared variances (never silent)

- **Server is truth**: no offline operation, no fork portability: the
  exact inverse of filecards, and the reason both exist behind one port.
- **The fence IS atomic** (corrected against a live server, os-2c0c474c):
  the earlier release declared it check-then-act because issue `metadata`
  had no conditional update. Issues turn out to have no `metadata` at all;
  the bookkeeping moved to the per-issue **document store**, and that
  store enforces optimistic concurrency: an update without
  `baseRevisionId` is refused `409`, and a stale base is refused too. The
  adapter validates the stored `seedToken`, remembers the revision it
  validated, and writes against exactly that revision, so a worker reaped
  and superseded between validation and write has its write **refused by
  the server** rather than landing behind the winner. That path exits 6
  (`fenced_out`), not a false success. Verified by the shared corpus
  against both the fake and a real instance.
- **Leases are Paperclip's watchdogs** (`issueWatchdogs`): `lease-renew`
  validates the fence and succeeds; staleness detection is the platform's.
- **Audit rides Paperclip's immutable activity log** (+ issue comments);
  the seed run-log stays authoritative only for filecards, and the state
  lint's transition replay does not apply here. `event-append` names its
  sink in the envelope: a task-scoped event becomes an issue comment
  (`sink: "issue-comment"`), and a **taskless** event (the port allows
  them: `task` is optional in `verbs.json`) is written to the company
  activity log (`sink: "company-activity"`, with the event id). There is
  no generic `/events` route to fall back on, so the taskless path is a
  real substrate, not a silent success.
- **Entering `blocked` requires a reason**: Paperclip refuses the status
  with 422 unless the issue has unresolved blockers, a pending
  interaction, or an `unblockDescriptor`. The port's `blocked_on` entry is
  exactly that statement, so the adapter carries it across as the
  descriptor (owner `board`, action naming the entry) rather than
  inventing one.
- **Checkout dispatches work, it does not merely lock** (live-only, and
  the reason `live-test.sh` has a quiescence gate): assigning an issue
  wakes the agent, and if that agent has no working runtime its run fails
  and `recovery.reconcile_stranded_assigned_issue` moves the issue
  `in_progress -> blocked` within ~10s. Seed drives its own work, so the
  agents a seed deployment owns should be **paused** in Paperclip
  (`PATCH /api/agents/<id> {"status":"paused"}`; the field is ignored on
  create). Pausing does not weaken the claim: checkout, contention, and
  the fence all behave identically. An unpaused roster will look like
  random contract failures.
- **API drift risk**: very high upstream velocity, no shipped SDK.
  `backend.toml` now carries a real `paperclip_version` pin (the release
  `live-test.sh` was last run against), and the fake server is a
  reconciled fixture rather than a documentation transcription. The first
  live run (os-2c0c474c, against 2026.817.0) found the adapter's whole
  transport layer wrong: single-issue routes are **not** company-scoped
  (`/api/issues/<id>`, not `/api/companies/<co>/issues/<id>`), the state
  field is **`status`** and a PATCH naming an unknown field is accepted
  and ignored (so a wrong field name reports success while nothing
  moves), issues carry no `metadata`, checkout takes
  `{agentId, expectedStatuses[]}` against a real agent row, dependency
  edges are written as `blockedByIssueIds` and read back as `blockedBy`,
  and there is no generic `/events` route. Re-run `live-test.sh` on every
  upgrade; the corpus re-reads after every transition precisely because
  the server will not tell you when you are wrong.
- The MCP gateway and heartbeat-driven execution are platform integrations
  **beyond the port**: this adapter deliberately uses only the issue
  surface.

## Testing

Both suites run the **same corpus** (`corpus.sh`), so they cannot drift
apart silently:

- `sh test.sh`: offline, against `testdata/fake-paperclip` (a Python3
  stdlib server reconciled against the real API, not transcribed from
  docs). Runs in `make check` via `validate.sh`.
- `sh live-test.sh`: against a real instance. Point it at one with
  `PAPERCLIP_API_URL=<url>`, or let it boot its own with
  `sh live-test.sh --onboard` (needs Node >= 20 and network; runs
  `npx paperclipai@<pin> onboard --yes` with embedded Postgres on
  loopback). It **self-skips with exit 0** when neither is available, so
  `make check` stays green and offline, and it is wired into
  `validate.sh` on those terms.

The corpus covers every required verb, the atomic document fence
including rotation on reclaim, checkout contention, checkout-aware
`ready`, mandatory ancestry, the reviewer lockout, cascades on close and
cancel, `comment_id`/`evidence_id`, and `event-append` in **both** the
task-scoped and taskless shapes. Every state assertion re-reads the
issue, because this server returns 200 for writes it ignores.

**Where the live test lives (the os-2c0c474c decision).** It is *not* on
the default PR path: a run costs a large npm install, an embedded
Postgres and 210 migrations (~90s cold), which no ordinary PR should pay.
It runs three ways instead: on demand by a developer
(`sh live-test.sh --onboard`), automatically via the opt-in
`.github/workflows/paperclip-live.yml` (`workflow_dispatch` + weekly
schedule, **non-required**), and implicitly in `make check` as a skip
whenever no instance is configured. Re-run it and bump
`paperclip_version` whenever the upstream release moves.
