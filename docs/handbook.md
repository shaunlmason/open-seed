# The open-seed handbook

How to run a project on open-seed: setup, the task lifecycle, the loop,
guardrails, and how the system degrades and scales. This is the user-facing
companion to the [design](design-options.md) (which explains *why*) and the
[build plan](build-plan.md) (which tracked *how it was built*).

## 1. Getting started

1. **Instantiate the template** (GitHub "Use this template", or clone and
   re-init). Everything you need is checked in; the only binary — the seed
   engine — is downloaded on first use by `scripts/seed`, pinned and
   SHA-256-verified against `.seed/engine.lock`.
2. **Create the coordination state ref:**

   ```sh
   scripts/seed init          # creates the seed-state branch (never check it out)
   ```

3. **Apply server-side protections** — these are what make the gates real
   (`scripts/seed init-github` prints this checklist):
   - Branch protection on `main`: require the `check-validate` checks and a
     review; **require conversation resolution before merging** (otherwise
     review-thread fixes pushed while you merge are silently stranded);
     no force pushes.
   - Branch rule for `seed-state`: allow pushes, **block force-pushes and
     deletion**.
   - Tag rule for `seed-anchor/*`: create-only, no deletion.
4. **Edit the identity files:** put your leads in `CODEOWNERS`, your operator
   roster in `.seed/config.toml`, your mission in `.seed/teams/core.yaml`,
   and wire your real lint/test into `make check` (keep it fast — it is the
   backpressure everything multiplies through).
5. Windows users run `scripts/seed.ps1`; air-gapped machines set a
   `vendor <path>` line in `engine.lock` or `SEED_ENGINE=<path>`.

## 2. The task lifecycle

States: `backlog → ready → in_progress → review → done`, plus `blocked` and
`cancelled`. The full edge table lives in `.seed/port-schema/transitions.json`
and is enforced by the engine — invalid transitions are refused, not coerced.

```sh
scripts/seed task create --title "Fix login" --actor alice   # → backlog
scripts/seed task promote os-1a2b --actor alice              # → ready (operator)
scripts/seed task ready --actor agent-1                      # discover work
scripts/seed task claim os-1a2b --actor agent-1 --lease 60m  # → in_progress
```

- **Claim before working.** Claims are synchronous and exclusive: exit 2
  means someone else has it — move on. Keep the returned `claim_token`;
  every later verb on the card needs it (a reaped claim's stale token gets
  exit 6). Renew with `seed task lease-renew` at half-lease cadence — the
  loop does this for you.
- **Plan first (above L1).** An unplanned card authorizes *planning only*:
  author `plans/<task-id>.md` (sections: Steps, File Scope, Acceptance
  Criteria, Validation Commands), open a PR from branch `seed/<id>-plan`
  touching only that file, then park the card
  (`transition --to blocked --blocked-on plan:<pr>`). Maintenance unparks it
  when the plan PR merges. The **approved plan is the blob at your task PR's
  merge-base** — amending a plan means a new plan PR, then rebasing your
  task branch (CI's stale-plan check forces exactly this).
- **Implement on `seed/<task-id>`** in a worktree. Task PRs never touch
  `plans/**` (CI rejects them). Generate your receipt
  (`seed receipt generate <id> --base origin/main --run --write`), commit
  it, push, open the PR.
- **Finish:** `transition --to review` with your token, attach evidence.
  A human (or the reviewer lane, once activated) reviews; the PR merges
  through the server gates; maintenance closes the card and unblocks
  dependents. For work that never lands as a PR, an operator runs the
  **seed-close-no-pr** workflow — the dispatch actor is server-attributed
  and becomes the card's evidence.

Two habits that compound: append durable insights to `memory/LEARNINGS.md`
and failed approaches to `memory/DEADENDS.md` in your task PR; record
irreversible decisions as ADRs in `decisions/` (append-only, union-merged).

## 3. Running the loop

```sh
scripts/loop.sh --actor loop-1 --harness claude        # until queue empty
scripts/loop.sh --once                                 # one card
make smoke                                             # deterministic end-to-end proof
```

Per iteration the loop claims the highest-priority planned card, creates a
fresh worktree (`.seed/hooks/post-create.d/` runs, propagating
`.worktreeinclude` files), invokes the harness through the adapter contract,
then gates mechanically: the blocking `pre-merge.d/` hooks must pass *and*
the merge-base plan's Validation Commands must run green during receipt
generation. Green → evidence + hand-off to review; red → release with a
handoff stub and count toward the circuit breaker
(`max_attempts_per_task` consecutive failures stops the loop;
`loop_max_iterations` bounds the run). Harnesses: `scripts/harness/claude`
and `scripts/harness/codex` ship; add your own by dropping an executable in
`scripts/harness/` honoring the contract in `scripts/seed-harness`
(prompt on stdin, JSON envelope out, exits 0/1/3/124/127). Permission tiers
map per-harness and the mapping is **declared, never silent** — see the role
files' `permission:` frontmatter.

## 4. Guardrails, honestly

`.seed/guardrails.yaml` is the vocabulary; enforcement is layered — hooks
locally, CODEOWNERS + branch protection server-side, validators in CI:

- **Autonomy tiers**: L1 report-only, L2 assisted-in-worktree (agent
  implements against an approved plan, human merges — the default ceiling),
  L3 unattended-with-gates (activates with the pr-review lane).
- **Budgets are advisory** on the file backend: the loop enforces its own
  iteration/attempt caps, but nothing in a repo can hard-stop an external
  process. Post-hoc accounting lives in the run log. Hard org-wide stops
  need a platform (that's the backend/event seam, not the template).
- **Control surface** (`.seed/**`, workflows, CODEOWNERS, the shim…): never
  auto-mergeable, always owner-reviewed. The auto-merge allowlist may not
  intersect it — nor `plans/**`, so no agent can approve its own work order.
- **Trust, precisely:** everything on the `seed-state` ref is exactly as
  trustworthy as push access to it (leases, rejections, the run log). The
  load-bearing gates deliberately ground elsewhere: merged plans, CI-
  regenerated receipts, server-attributed reviews and dispatch actors. The
  maintenance conformance lint replays the ref against the transition table
  and HALTs coordination on tampering (`seed state resume` after a human
  look).

## 5. The degradation ladder

The same repo works at every rung; each rung only adds convention, never
requirements:

1. **Solo human, no engine** — cards are readable markdown on the state ref
   (`git fetch origin seed-state && git cat-file -p FETCH_HEAD:tasks/…`);
   CODEOWNERS and CI still gate PRs; `validate.sh` degrades to a warning.
2. **Human + engine** — the port verbs, receipts, validators.
3. **One agent session** — AGENTS.md teaches any harness the loop manually.
4. **The loop** — unattended L2 as in §3.
5. **Squads** — add team files as scopes grow (`core` is the degenerate
   default; multi-squad routing activates in v2).
6. **External orchestrators** — anything that can run a CLI can drive the
   port; TUIs and platforms layer on top without changing the contract.

## 6. Scaling and upgrading

- **Write ceiling (file backend):** every mutating verb is one commit+push
  on one ref — contention starts well below ten chatty agents. The engine
  retries with backoff and reports. Two throughput upgrades, split by shape
  (R4): **one machine** hammering the loop → the **fastcards builtin**
  (`.seed/backends/fastcards/`): a SQLite store inside the engine — native
  atomic claims, no network, linked worktrees share one DB. Machine-local
  by declaration (`state_portability = "machine"`): state does not travel
  with clones or CI, so **the close lane is local** — you close review
  cards yourself (`seed task close <id> --no-pr …`), the CI auto-close
  never sees them. **Multiple writers** → the
  **beads backend** (ships in `.seed/backends/beads/`): install `bd` + `jq`,
  `bd init`, native atomic claim and close-cascade, replicated state. Either
  switch is a reviewed config line **plus the state move**:
  `scripts/seed state export > cards.json`, flip `backend =` in
  `.seed/config.toml`, `scripts/seed init`, then
  `scripts/seed state import cards.json` — ids, states, dep edges,
  rejections, and the run log all travel; import refuses a non-empty
  target. Then `scripts/seed backend verify <name>`. Read each README for
  the declared variances. For teams already living in a tracker, the
  **linear backend** (`.seed/backends/linear/`) puts cards on a Linear
  team's workflow (one required custom `Blocked` state, states mapped by
  name; emulated claim and cascade, declared), and the **jira backend**
  (`.seed/backends/jira/`) does the same for Jira Cloud (status-name
  convention incl. `Backlog` + `Blocked`, transitions arbitrated by the
  workflow, actors mapped to accountIds in `actors.json`).
- **Human visibility — the issues mirror:** set `[mirror] enabled = true` in
  `.seed/config.toml` and the maintenance workflow renders every card as a
  labeled GitHub issue (`seed:ready`, `seed:in_progress`, `seed:review`,
  `seed:blocked`, `seed:done`; backlog unlabeled; done/cancelled close as
  completed/not-planned). Strictly **one-way**: cards stay authoritative,
  and editing an issue's labels changes nothing — treat issues as a read-only
  dashboard (label edits become *requests* only once the dispatcher lane is
  active).
- **When budgets must be enforced, not advised** — the control-plane rung:
  the **paperclip backend** (`.seed/backends/paperclip/`) puts cards on a
  Paperclip server with DB-atomic checkouts, server-validated transitions,
  and **native hard budget stops** (the platform pauses over-budget agents
  — the enforcement R6 says a repo alone can never provide). Server is
  truth: no offline, no fork portability; read its README for setup and
  the declared variances.
- **Merge throughput:** don't enable repo-wide "require branches up to
  date" — stale-plan safety is already per-PR in verify. At scale, enable a
  GitHub **merge queue**; the verify workflow already handles `merge_group`
  by deriving the PR and classifying by its real head branch.
- **Upgrading the engine** is two commands: `scripts/seed upgrade`
  (resolve the latest release — or `--to vX.Y.Z`, rollback included —
  verify its checksums, preflight protocol compatibility, and rewrite
  `.seed/engine.lock` atomically; `--check` only reports), then a
  **reviewed PR** with the diff — the command never touches git, because
  the lockfile is control surface, and its output walks you through the
  review steps and the release notes. An incompatible release is refused
  before anything is written (the alternative is a pin that exits 10 on
  every invocation).
- **Upgrading the template** is the same two-command story:
  `scripts/seed template upgrade` reads your provenance from
  `.seed/template.lock` (repo, recorded version, and — after the first
  upgrade — the upstream commit it merged from, stamped by the command),
  fetches the target release, and three-way merges what changed upstream
  against what you changed locally onto a new local branch
  `template-upgrade/<tag>`: conflicts staged as standard markers, your
  work products (`plans/`, `receipts/`, `memory/`, `decisions/`) never
  merged, your working tree untouched. Then the **reviewed PR** — the
  command never pushes and never opens one: resolve any conflicts on the
  branch, run `make check`, push, merge through the ordinary gates.
  A `.seed/version` change in the target is called out in the envelope;
  run `scripts/seed upgrade` next as its own reviewed step. Pull-based
  by design (§7.1): upstream never pushes into your repo. `--check`
  reports current vs latest without creating anything.
- **Cutting a template release (maintainers):** bump `version` in
  `.seed/template.lock`, commit, tag that commit — version-then-tag. Do
  not write a `commit` line: the lockfile cannot record the SHA of the
  commit that contains it; consumers resolve your immutable release tag
  instead — which is why the seed-anchor-style tag protections in §1
  extend to release tags.

## 7. Where everything lives

| Path | What |
|---|---|
| `.seed/` | The contract: config, guardrails, port spec, roles, teams, hooks, engine pin |
| `plans/` `receipts/` `memory/` `decisions/` | Work products, each with its own gate |
| `scripts/seed` | The only coordination entry point |
| `scripts/loop.sh` · `scripts/seed-harness` | The loop and the harness adapters |
| `seed-state` ref | Machine-written coordination state — never checked out, never hand-edited |
| `.github/workflows/` | check-validate + maintenance (live); dispatch + pr-review (inert until secrets) |
