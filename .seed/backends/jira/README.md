# jira backend

Wraps Jira Cloud REST v3 behind the seed port — the enterprise-reach
adapter. Jira's twist: statuses are workflow-specific and moves happen via
the transitions API, so the adapter resolves every transition **by target
status name** at call time rather than assuming ids; Jira's workflow is the
arbiter, honestly surfaced (a move it refuses is exit 3).

## Connect

1. Create an API token (id.atlassian.com → Security → API tokens); note
   the account email and site URL.
2. Export the environment the manifest declares: `JIRA_BASE_URL`
   (`https://<site>.atlassian.net`), `JIRA_EMAIL`, `JIRA_API_TOKEN`,
   `JIRA_PROJECT_KEY`.
3. **Satisfy the status convention** (table below) in the project's
   workflow — company-managed workflows can add statuses once for all
   projects. `Backlog` and `Blocked` are the two most commonly missing.
   A missing status refuses with exit 5 + remediation at first use,
   distinguished from an illegal move (exit 3) via the project statuses
   API.
4. **Map your actors**: edit `.seed/backends/jira/actors.json` —
   `{"<seed actor>": "<Jira accountId>"}` (accountIds are in profile URLs
   or the user search API). Jira Cloud assignment requires accountIds, so
   an unmapped actor is refused with exit 5; envelope `holder` fields use
   the reverse mapping.
5. Set `.seed/config.toml`: `[coordination] backend = "jira"`, then
   `scripts/seed backend verify jira`.

## Status & priority mapping (by status name)

| port | Jira |
|---|---|
| backlog | `Backlog` |
| ready | `To Do` |
| in_progress | `In Progress` |
| review | `In Review` |
| blocked | `Blocked` |
| done | `Done` |
| cancelled | `Done` + resolution `Won't Do` |

P0–P3 → Highest / High / Medium / Low.

## What the adapter enforces

- **Create lands in `backlog`, whatever the workflow's initial status**:
  the port contract says create returns backlog, and a workflow whose
  initial status is `To Do` (the common default) would otherwise mint
  ready cards — after the POST the adapter reads the created issue's
  status and transitions it to `Backlog` (contract-tested against a fake
  whose initial status is To Do).
- **Claim** assigns the mapped accountId and moves to In Progress; a held
  issue is contention (exit 2 + holder). **The fence is a minted per-claim
  token** riding a `seed:tok:<hex>` label, validated with the
  assignee-accountId match on every fenced verb (D1 binding — the token
  distinguishes a reaped predecessor from the same actor's new claim);
  rotation on reclaim is contract-tested.
- **close/accept is review → done only** (the port table's accept+cascade
  edge; the workflow refuses the rest anyway — from Done the fake offers
  only reopen). Cancellation is the `cancel` verb → Done + `Won't Do`.
- **Both close and cancel run the emulated cascade**: dep edges ride
  `seed:bo:dep:<key>` labels; the terminal transition removes its entry
  everywhere and moves issues left blocked on nothing back to To Do,
  returning released keys in `cascaded`.
- Plan parking (`seed:bo:plan:<pr>`), entry-by-entry release, the reviewer
  lockout (`seed:rejected:*` at claim and in `ready`), priority-ordered
  `ready`, ADF comments for evidence/events (taskless events → a
  `seed: audit log` issue; Jira's changelog is the audit substrate).

## Declared variances (never silent)

- **Emulated claim**: read-check then assign, last-write-wins — no server
  arbitration (weaker than filecards' push-wins). The adapter closes the
  practical gap with **post-claim verification**: after its writes it
  re-reads the issue and succeeds only if the substrate holds *its*
  assignee and token — a lost interleave reports contention with the
  real holder instead of returning a dead token. Declared.
- **No leases**: `lease-renew` validates the fence and succeeds; staleness
  is `seed maintain reap` policy.
- **Server is truth**: no offline operation, no fork portability
  (`state_portability = "server"`).
- **Cascade is emulated via `seed:bo:*` labels** — implemented and
  contract-tested, never described as absent; its limitation is that label
  bookkeeping is only as trustworthy as label write access. Label writes
  use Jira's **atomic add/remove update operations** (never whole-array
  replacement), so concurrent label edits elsewhere on an issue are not
  clobbered. The release transition runs before an entry label is
  removed, and a per-dependent refusal never aborts the terminal verb or
  the rest of the cascade: the dependency stays recorded, the dependent
  is reported in the envelope's `cascade_skipped`, and recovery is an
  operator `unblock` once the workflow allows the move — never a Blocked
  issue with no blocker on record.
- **Statuses outside the convention are not seed cards**: `list` skips
  them, and `get` on one refuses with remediation (exit 5) rather than
  emitting an invalid port state.
- **Rate limits**: Jira Cloud throttles per-account; `ready`, `list`, and
  the cascade are search-backed — fine for a squad's queue, not bulk.
- **Jira Server/DC (v2 API) is out of scope** — Cloud REST v3 only.

## Testing

`sh test.sh` runs the offline contract test against `testdata/fake-jira`
(Python3 stdlib HTTP server: transitions offered from the current status
only, To-Do-initial workflow, accountId assignment, labels, JQL search).
It covers every required verb plus the backlog-landing create, the
accountId mapping refusal, token rotation, both cascades, close outside
review, and the missing-Blocked-status refusal; it runs in `make check`
via validate.sh.
