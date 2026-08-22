# linear backend

Wraps Linear's GraphQL API behind the seed port. The cleanest external
state-model fit surveyed: Linear's default team workflow plus one required
custom state maps onto the port states **by name** — no label tricks for
states at all.

## Connect

1. Create a personal or OAuth API key with read/write issue scope; find the
   team id (Settings → API, or the `team` query).
2. Export the environment the manifest declares: `LINEAR_API_KEY`,
   `LINEAR_TEAM_ID` (`LINEAR_API_URL` overrides the endpoint for testing).
3. **Add the required `Blocked` workflow state** to the team (one click in
   team settings → Workflow). Without it, above-L1 tasks could never park
   on their plan PRs — so it is part of the convention, not optional: the
   adapter refuses with exit 5 + remediation when any required state is
   missing.
4. Set `.seed/config.toml`: `[coordination] backend = "linear"`, then
   `scripts/seed backend verify linear`.

## State & priority mapping (by workflow-state name)

| port | Linear |
|---|---|
| backlog | `Backlog` |
| ready | `Todo` |
| in_progress | `In Progress` |
| review | `In Review` |
| blocked | `Blocked` (required custom state) |
| done | `Done` |
| cancelled | `Canceled` |

P0–P3 → urgent / high / medium / low.

## What the adapter enforces

- **Claim** assigns the caller and moves to In Progress; a held issue is
  contention (exit 2 + holder). **The fence is a minted per-claim token**
  riding a `seed:tok:<hex>` label, validated on every fenced verb *in
  addition to* the assignee match — the API key alone cannot tell a reaped
  predecessor from the same actor's new claim; the rotating token can
  (D1 binding, exceeding the plan's assignee-match floor to match the
  beads/paperclip posture).
- **close/accept is review → done only** (the port table's accept+cascade
  edge); anything else exits 3. Cancellation is the separate `cancel` verb.
- **Both close and cancel run the emulated cascade**: dep edges ride
  `seed:bo:dep:<id>` labels; the terminal transition removes its entry
  everywhere and moves issues left blocked on nothing back to Todo,
  returning the released ids in `cascaded`.
- Port bookkeeping rides labels: `seed:bo:<entry>` (plan parking and dep
  edges, released entry-by-entry), `seed:author:<actor>` (implementer of
  record), `seed:rejected:<actor>` (reviewer lockout — enforced at claim
  and in `ready`).

## Declared variances (never silent)

- **Emulated claim**: Linear updates are last-write-wins, so the
  read-check-assign window is real — weaker than filecards' push-wins
  (no server arbitration). Declared, never hidden.
- **No leases**: `lease-renew` validates the fence and succeeds;
  staleness is handled by `seed maintain reap` policy, not the platform.
- **Server is truth**: no offline operation, no fork portability
  (`state_portability = "server"`).
- **Cascade is emulated via `seed:bo:*` labels** — implemented and
  contract-tested; its limitation is that label bookkeeping is only as
  trustworthy as label write access.
- **Actors are Linear user handles**: assignment needs real Linear users,
  so seed actor names must be resolvable Linear user identifiers (the
  contract test's fake accepts any string; live Linear will not).
- **Rate limits**: Linear's API is rate-limited (~1500 requests/hour for
  personal keys); the cascade and `ready` are O(issues) queries — fine for
  a squad's queue, not for bulk imports.
- Taskless `event-append` lands on a low-priority `seed: audit log` issue;
  Linear's own issue history is the audit substrate.

## Testing

`sh test.sh` runs the offline contract test against `testdata/fake-linear`
(Python3 stdlib HTTP server modeling the GraphQL subset: workflow states by
name, assignee, labels-on-demand, last-write-wins updates). It covers every
required verb, token rotation on reclaim, contention, plan parking, the
lockout, the cascade on close AND cancel, close refused outside review, and
the missing-Blocked-state refusal; it runs in `make check` via validate.sh.
