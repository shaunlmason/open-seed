# The open-seed handbook

How to run a project on open-seed: setup, the task lifecycle, the loop,
guardrails, and how the system degrades and scales. This is the user-facing
companion to the [design](design-options.md) (which explains *why*), the
[build plan](build-plan.md) (which tracked *how it was built*), and the
[architecture map](architecture.md) (which spans both repos: the layering,
the port, the evidence chain, and where each gate grounds).

## 1. Getting started

1. **Instantiate the template** (GitHub "Use this template", or clone and
   re-init). Everything you need is checked in; the only binary: the seed
   engine: is downloaded on first use by `scripts/seed`, pinned and
   SHA-256-verified against `.seed/engine.lock`.
2. **Create the coordination state ref:**

   ```sh
   scripts/seed init          # creates the seed-state branch (never check it out)
   ```

3. **Apply server-side protections**: these are what make the gates real
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
   and wire your real lint/test into `make check` (keep it fast: it is the
   backpressure everything multiplies through).
5. Windows users run `scripts/seed.ps1`; air-gapped machines set a
   `vendor <path>` line in `engine.lock` or `SEED_ENGINE=<path>`.

## 2. The task lifecycle

States: `backlog → ready → in_progress → review → done`, plus `blocked` and
`cancelled`. The full edge table lives in `.seed/port-schema/transitions.json`
and is enforced by the engine: invalid transitions are refused, not coerced.

```sh
scripts/seed task create --title "Fix login" --actor alice   # → backlog
scripts/seed task promote os-1a2b --actor alice              # → ready (operator)
scripts/seed task ready --actor agent-1                      # discover work
scripts/seed task claim os-1a2b --actor agent-1 --lease 60m  # → in_progress
```

- **Claim before working.** Claims are synchronous and exclusive: exit 2
  means someone else has it: move on. Keep the returned `claim_token`;
  every later verb on the card needs it (a reaped claim's stale token gets
  exit 6). Renew with `seed task lease-renew` at half-lease cadence: the
  loop does this for you.
- **Plan first (above L1).** An unplanned card authorizes *planning only*:
  author `plans/<task-id>.md` (sections: Steps, File Scope, Acceptance
  Criteria, Validation Commands), open a PR from branch `seed/<id>-plan`
  touching only that file, then park the card
  (`transition --to blocked --blocked-on plan:<pr>`). Maintenance unparks it
  when the plan PR merges. The **approved plan is the blob at your task PR's
  merge-base**: amending a plan means a new plan PR, then rebasing your
  task branch (CI's stale-plan check forces exactly this).
- **Implement on `seed/<task-id>`** in a worktree. Task PRs never touch
  `plans/**` (CI rejects them). Generate your receipt
  (`seed receipt generate <id> --base origin/main --run --write`), commit
  it, push, open the PR.
- **Finish:** `transition --to review` with your token, attach evidence.
  A human (or the reviewer lane, once activated) reviews; the PR merges
  through the server gates; maintenance closes the card and unblocks
  dependents. For work that never lands as a PR, an operator runs the
  **seed-close-no-pr** workflow: the dispatch actor is server-attributed
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
map per-harness and the mapping is **declared, never silent**: see the role
files' `permission:` frontmatter.

**Checked-in workflows** (v2, §7.3): multi-step jobs live as step DAGs at
`.seed/workflows/<name>.yaml`: steps with `depends_on` edges (by id),
`consumes`/`produces` artifact contracts, `when`/`trigger_rule` branching,
loop groups, and `approval|review|checks` gates; the format is
`.seed/workflow-schema/workflow.schema.json`. Two commands run the story:

```sh
scripts/seed workflow validate --all      # thirteen preflight rules; runs in CI
scripts/seed workflow run smoke --mock    # end-to-end, zero credentials, zero side effects
scripts/seed workflow run fix-issue --input issue=42 --input repo=o/r --input pr=7 \
  --input head_sha=$(gh pr view 7 --json headRefOid -q .headRefOid)
```

Independent steps run in parallel waves; AI steps ride the same
`scripts/seed-harness` adapters as the loop, with the step's
`tools: readonly|coding` mapped onto `SEED_PERMISSION` (`read-only` /
`safe-edit`, nothing in a workflow file reaches `yolo`), and harness/model
values validated against the `[workflows]` registry in `.seed/config.toml`.
Run state (checkpoints, artifacts, gate records) lives under
`<git-common-dir>/seed-runs/<run-id>/`: local, shared across linked
worktrees, never committed. An `approval` gate pauses the run until you
write its response file and `--resume <run-id>`; resuming re-executes only
incomplete steps and refuses a run whose definition or inputs changed.
Under `--mock` every AI step goes to `scripts/harness/mock` and every
`run:` command is recorded, never executed: a mock run can prove any
workflow, `fix-issue` included, without touching anything. Steps that
mutate task state do it through `scripts/seed task <verb>` like every
other caller: workflows are the intra-run DAG; **cards stay the
inter-agent coordination layer** (dep edges + `ready`-gating + the close
cascade already schedule work across agents topologically).
`.claude/workflows/` remains the home for Claude-native dynamic
workflows, never under `.seed/`.

**The MCP transport** (research/10 §5.4: v2): `seed mcp serve` is an
MCP stdio server exposing one tool per port verb: the worker surface
(create/ready/get/list/claim/lease-renew/release/transition/comment/
attach-evidence) and one tool per operator verb (close, promote,
deprioritize, reject, cancel, reinstate, block, unblock, plan-unblock).
`.mcp.json` ships the registration (strict JSON: the format admits no
comments, so the entry is live but inert until a harness loads the
file):

```json
{ "mcpServers": { "seed": { "command": "sh", "args": ["-c", "exec scripts/seed mcp serve"] } } }
```

The shipped registration spawns `sh` (the POSIX bootstrap). On a native
Windows checkout swap the entry for the PowerShell bootstrap: same
server, same tools:

```json
{ "mcpServers": { "seed": { "command": "powershell", "args": ["-NoProfile", "-File", "scripts/seed.ps1", "mcp", "serve"] } } }
```

MCP is an ADDITIONAL transport, never a replacement: `tools/call`
dispatches through the identical service path the CLI uses: same
fencing, same transition table, same run-log events, same envelopes.
Port failures come back as tool results with `isError: true` carrying
the refusal envelope and exit class (contention, invalid transition,
fenced out, halted): JSON-RPC errors stay reserved for transport
faults. The wrapper adds no authority: `--actor` remains an asserted
tool argument, operator tools still check the `[operators]` roster, and
a HALT marker refuses mutating tools exactly as it refuses CLI verbs.

Prefer the MCP surface for tool-native harnesses (schema'd calls beat
shell strings) and for MCP-gateway governance in the Paperclip style;
CI, cron, and bare shells stay on the CLI, which remains the source of
truth, no verb exists MCP-only.

**Sharing skills between repos** (D8): `seed.yaml` at the template
root names upstream skill sources; `seed.lock` pins them (commit SHA +
content sha256, full source coordinates); both are control surface
(D4.1: CODEOWNERS-reviewed, never auto-merged). The flow:

```sh
$EDITOR seed.yaml                      # name sources; optionally compose
scripts/seed skills lock               # resolve refs → commits, hash trees
scripts/seed skills install            # materialize under skills/managed/
git add seed.yaml seed.lock skills/managed && git commit
```

CI runs `seed skills install --frozen` (the D8 supply-chain rule): an
unlocked manifest edit, a hash mismatch, or on-disk drift fails the
build: with an empty manifest the step is a no-op, so fresh
instantiations stay green with zero configuration. Managed skills flow
through `seed sync` to `.claude/skills/` and `.agents/skills/` exactly
like local ones (a local skill with the same name wins); install prunes
only `skills/managed/`: local skills are never touched.

`compose:` entries generate a NEW skill from an ordered `use:` list
(bodies concatenated with headings demoted, supporting files carried
over; unknown inputs, self-use, and cycles are refused at parse time).
Composed skills are not locked: they are deterministic functions of
locked inputs, regenerated at install.

**Injection-review posture**: skill updates arrive as ordinary PRs whose
diff shows the new skill content: review happens in the review pane on
that diff, never at install time. Treat upstream skill text like any
other third-party code contribution: the lock pins what you reviewed,
and `--frozen` guarantees CI runs only that.

**Multi-squad routing** (§6: v2 activated): v1 ships one `core` squad
whose bare-`**` scope satisfies every rule trivially; a second
`.seed/teams/<name>.yaml` (start from `platform.yaml.example`) activates
the full semantics. `seed validate` enforces: non-overlapping specific
scopes (core's `**` fallback is exempt: it is the "matches what nothing
else claims" floor; two bare-`**` squads are refused; a specific overlap
passes only under a `shared_scope` entry naming one owning squad),
unique priority ints, a human lead per squad, and tier ≤ the guardrails
ceiling. Cards route **explicit `squad:` → lowest-priority backlog label
match → core**, no card can be invisible, and `get`/`list`/`ready`
surface the resolved squad. One loop per squad is the scaling unit:

```sh
scripts/loop.sh --actor web-loop --squad web
scripts/seed task ready --actor you --squad web
```

Cross-squad merges ride the owning squad's gate: CODEOWNERS + that
squad's tier govern merges into its scope (once >1 squad exists, a
codeowners-reviewing squad whose lead is missing from CODEOWNERS gets a
validation warning). Goal-ancestry checking activates on the literal
`>1 squad || any mission`: open cards with no resolvable parent chain
to a mission card warn (report, never refusal): core-only, missionless
repos see nothing, which is why the shipped core.yaml keeps its
`mission:` commented out until you set a real one.

**Mail and handoff packets** (§7.2, inspirations/08 as amended by its
erratum): inter-agent messages are one never-rewritten file per message
at `mail/<recipient>/<msg-id>.yaml` on the seed-state ref: trust =
push access, like every coordination artifact; no daemon. Verbs:

```sh
scripts/seed mail send --actor you --to agent-2 --type request --text "..." [--task id]
scripts/seed mail read --actor you --unread
scripts/seed mail ack  --actor you --id msg-...
scripts/seed mail nudge agent-2     # tmux-only, content-free "you have mail"
```

Direct ack is a file MOVE into `mail/<you>/acked/`; a `_all` broadcast
is COPIED there instead (the shared file stays for other readers) and
maintenance prunes acked history to the newest 30 per recipient.
Mailboxes are read at natural checkpoints, not watched: the loop
injects unread mail into the harness prompt **fenced as untrusted
data** and acks it only after the iteration succeeds; AGENTS.md carries
the same checkpoint rule for interactive agents. `seed maintain
report` surfaces unread counts per actor.

`seed handoff generate <task> [--write]` renders the bounded (≤8KB)
mechanical continuation packet: card goal/criteria, claim block,
evidence trail, branch/HEAD/dirty-file anchors from git: at
`handoff/<task-id>.md` on the state ref. Worker release/park writes one
automatically with real workspace anchors; a maintenance **reap** runs
in its own checkout, so reap-written packets mark the anchors
unavailable instead of recording the reaper's git state.

**Worktree tool fidelity** (D6 "the rest v2"): `.seed/hooks/` is the
runner-agnostic lifecycle contract, and `.seed/hooks/shims/<tool>/` ships
checked-in fragments for the surveyed external tools. Support is declared,
never silent: the per-tool matrix (README in each shim dir has the full
table and install steps):

| Tool | Post-create | Teardown | Blocking pre-merge |
|---|---|---|---|
| superset | yes (`.superset/config.json`) | yes | no |
| agent-deck | yes (`worktree-setup.sh`) | best-effort | no |
| vibe-tree | yes (`.vibetree/hooks/`) | yes | no |
| octomux | yes (`task_created`) | approximate (`runtime_state_changed`) | no: fire-and-forget hooks |
| amux | yes (`setup-workspace`) | best-effort (`archive`) | no |
| dmux | yes (`worktree_created`) | yes (`before_worktree_remove`) | **no: dmux spawns `pre_merge` detached; it cannot veto** |
| tmux-ide | no | no | no |
| ouijit | via `start` hook | approximate (`done`) | no |
| parallel-code | README-only: no hook surface | n/a | n/a |

No surveyed tool can honor a blocking pre-merge, which is why the local
`pre-merge.d/` gate is a convenience pre-check and **CI verify is the
merge authority everywhere** (R11).

## Activating the agent lanes

The seed-dispatch and pr-review workflows ship in-tree and **inert**
(D7): without secrets every run is a cheap no-op. Everything
mechanizable already landed: the deterministic label router
(`scripts/seed-dispatch-route`, contract-tested in validate.sh), the
D4.5 identity check (`scripts/seed-review-identity`, wired into verify
on `pull_request_review` events), and the audited workflow conventions.
The flip itself is yours; in order:

1. **Secrets**: add `ANTHROPIC_API_KEY` (repo → Settings → Secrets →
   Actions). Both workflows activate on its presence alone.
2. **The reviewer's identity**: install/choose the GitHub App the
   pr-review lane posts through, and add that identity to
   `[operators].actors` in `.seed/config.toml`: the roster is what
   authorizes `seed task reject` and the other operator verbs the lane
   uses. Without this the lane can review but every reject is refused.
3. **Repo settings** (once you want L3): add the reviewer app to branch
   protection's allowed approvers; the D4.5 step already enforces
   reviewer ≠ implementer, and a review posted after the last push
   re-runs verify automatically.
4. **Guardrails tier**: raise `autonomy.max_tier` L2 → L3 in
   `.seed/guardrails.yaml` (its own reviewed PR: control surface).
5. **Solo-mode caveat** (§10 Q1): on a solo repo your own account is
   admin, implementer, AND operator: agents need a non-admin machine
   identity before L3 means anything. Do not skip this.

**Live checklist** (run after flipping; attach each artifact's URL as
evidence on card os-70028620):

| # | Exercise | Expected artifact |
|---|---|---|
| 1 | `cmd:promote` label on a mirrored backlog card's issue | dispatch run: promote applied, label removed, `by:agent` + sticky comment |
| 2 | `cmd:promote` from an account without write access | dispatch run log shows the refusal; no state change |
| 3 | `cmd:frobnicate` label | run ignores it; label left for a human |
| 4 | hand-edit a `state:*` mirror label | sticky comment marks it a REQUEST; no state write |
| 5 | issue form (no cmd label) | AI dispatcher files exactly one card, `[ai]` title, sticky marker |
| 6 | open a task PR touching one file | pr-review posts a REAL review under the app identity; verify re-runs on the review; `seed-review-identity` passes |

## 4. Guardrails, honestly

`.seed/guardrails.yaml` is the vocabulary; enforcement is layered: hooks
locally, CODEOWNERS + branch protection server-side, validators in CI:

- **Autonomy tiers**: L1 report-only, L2 assisted-in-worktree (agent
  implements against an approved plan, human merges: the default ceiling),
  L3 unattended-with-gates (activates with the pr-review lane).
- **Budgets are advisory** on the file backend: the loop enforces its own
  iteration/attempt caps, but nothing in a repo can hard-stop an external
  process. Post-hoc accounting lives in the run log. Hard org-wide stops
  need a platform (that's the backend/event seam, not the template).
- **Control surface** (`.seed/**`, workflows, CODEOWNERS, the shim…): never
  auto-mergeable, always owner-reviewed. The auto-merge allowlist may not
  intersect it: nor `plans/**`, so no agent can approve its own work order.
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

1. **Solo human, no engine**: cards are readable markdown on the state ref
   (`git fetch origin seed-state && git cat-file -p FETCH_HEAD:tasks/…`);
   CODEOWNERS and CI still gate PRs; `validate.sh` degrades to a warning.
2. **Human + engine**: the port verbs, receipts, validators.
3. **One agent session**: AGENTS.md teaches any harness the loop manually.
4. **The loop**: unattended L2 as in §3.
5. **Squads**: add team files as scopes grow (`core` is the degenerate
   default; multi-squad routing activates in v2).
6. **External orchestrators**: anything that can run a CLI can drive the
   port; TUIs and platforms layer on top without changing the contract.

## 6. Scaling and upgrading

- **Write ceiling (file backend):** every mutating verb is one commit+push
  on one ref: contention starts well below ten chatty agents. The engine
  retries with backoff and reports. Two throughput upgrades, split by shape
  (R4): **one machine** hammering the loop → the **fastcards builtin**
  (`.seed/backends/fastcards/`): a SQLite store inside the engine: native
  atomic claims, no network, linked worktrees share one DB. Machine-local
  by declaration (`state_portability = "machine"`): state does not travel
  with clones or CI, so **the close lane is local**: you close review
  cards yourself (`seed task close <id> --no-pr …`), the CI auto-close
  never sees them. **Multiple writers** → the
  **beads backend** (ships in `.seed/backends/beads/`): install `bd` + `jq`,
  `bd init`, native atomic claim and close-cascade, replicated state. Either
  switch is a reviewed config line **plus the state move**:
  `scripts/seed state export > cards.json`, flip `backend =` in
  `.seed/config.toml`, `scripts/seed init`, then
  `scripts/seed state import cards.json`: ids, states, dep edges,
  rejections, and the run log all travel; import refuses a non-empty
  target. Then `scripts/seed backend verify <name>`. Read each README for
  the declared variances. For teams already living in a tracker, the
  **linear backend** (`.seed/backends/linear/`) puts cards on a Linear
  team's workflow (one required custom `Blocked` state, states mapped by
  name; emulated claim and cascade, declared), and the **jira backend**
  (`.seed/backends/jira/`) does the same for Jira Cloud (status-name
  convention incl. `Backlog` + `Blocked`, transitions arbitrated by the
  workflow, actors mapped to accountIds in `actors.json`).
- **Human visibility: the issues mirror:** set `[mirror] enabled = true` in
  `.seed/config.toml` and the maintenance workflow renders every card as a
  labeled GitHub issue (`seed:ready`, `seed:in_progress`, `seed:review`,
  `seed:blocked`, `seed:done`; backlog unlabeled; done/cancelled close as
  completed/not-planned). Strictly **one-way**: cards stay authoritative,
  and editing an issue's labels changes nothing: treat issues as a read-only
  dashboard (label edits become *requests* only once the dispatcher lane is
  active).
- **When budgets must be enforced, not advised**: the control-plane rung:
  the **paperclip backend** (`.seed/backends/paperclip/`) puts cards on a
  Paperclip server with DB-atomic checkouts, server-validated transitions,
  and **native hard budget stops** (the platform pauses over-budget agents
: the enforcement R6 says a repo alone can never provide). Server is
  truth: no offline, no fork portability; read its README for setup and
  the declared variances.
- **Merge throughput:** don't enable repo-wide "require branches up to
  date" — stalee-plan safety is already per-PR in verify. At scale, enable a
  GitHub **merge queue**; the verify workflow already handles `merge_group`
  by deriving the PR and classifying by its real head branch.
- **Upgrading the engine** is two commands: `scripts/seed upgrade`
  (resolve the latest release, or `--to vX.Y.Z`, rollback included:
  verify its checksums, preflight protocol compatibility, and rewrite
  `.seed/engine.lock` atomically; `--check` only reports), then a
  **reviewed PR** with the diff: the command never touches git, because
  the lockfile is control surface, and its output walks you through the
  review steps and the release notes. An incompatible release is refused
  before anything is written (the alternative is a pin that exits 10 on
  every invocation).
- **Upgrading the template** is the same two-command story:
  `scripts/seed template upgrade` reads your provenance from
  `.seed/template.lock` (repo, recorded version, and: after the first
  upgrade: the upstream commit it merged from, stamped by the command),
  fetches the target release, and three-way merges what changed upstream
  against what you changed locally onto a new local branch
  `template-upgrade/<tag>`: conflicts staged as standard markers, your
  work products (`plans/`, `receipts/`, `memory/`, `decisions/`) never
  merged, your working tree untouched. Then the **reviewed PR**: the
  command never pushes and never opens one: resolve any conflicts on the
  branch, run `make check`, push, merge through the ordinary gates.
  A `.seed/version` change in the target is called out in the envelope;
  run `scripts/seed upgrade` next as its own reviewed step. Pull-based
  by design (§7.1): upstream never pushes into your repo. `--check`
  reports current vs latest without creating anything.
- **Flavors (opinionated stack variants).** The core is language-agnostic on
  purpose: its only contract with your stack is `make check` plus the hooks
  (§10 Q3). A **flavor** goes beyond that without breaking it:

      scripts/seed-flavor list
      scripts/seed-flavor install typescript
      npm install            # the script runs no package manager itself
      scripts/seed-flavor status

  `install` is a **fresh-instantiation step**, not an idempotent converger: it
  refuses rather than overwrite a `Makefile` you have already wired, a
  destination that already exists, or a repo that is already flavored, and it
  names the alternative each time. It records what it wrote in
  `.seed/flavor.lock`.

  Keeping a flavored repo current is **two separate commands, in this order**:

      scripts/seed template upgrade    # brings new payload into flavors/<name>/
      scripts/seed-flavor upgrade      # applies it to the installed destinations

  Both are consumer-initiated, and the second does not happen automatically:
  `seed template upgrade` merges by *path*, so it updates
  `flavors/<name>/...` but never the copies `install` made at the repo root.
  Run the first without the second and you keep running the payload you
  installed; `seed-flavor status` shows which destinations have diverged.
  `seed-flavor upgrade` merges against the payload as installed, so a file you
  have edited yourself conflicts (standard markers) instead of being
  overwritten. See `decisions/0002-template-flavors.md`.

- **Cutting a template release (maintainers):** bump `version` in
  `.seed/template.lock`, commit, tag that commit: version-then-tag; then
  push the commit and the tag, and publish a GitHub Release for that tag,
  e.g. `git push origin main && git push origin v0.1.0` followed by
  `gh release create v0.1.0 -t v0.1.0 --verify-tag`. Push the tag before
  the publish: if the tag exists only locally, `gh release create`
  silently synthesizes one from the head of the default branch, so a
  Release published without `--verify-tag` can anchor to a different
  commit than your version bump, and release tags are immutable under
  the §1 protections, so the mistake cannot be corrected in place.
  `--verify-tag` makes the command fail instead. Publishing is the
  load-bearing step: consumers resolve the release through the
  `/releases/latest` redirect, which GitHub serves only for a published
  Release, not a bare tag, so the published Release is what `seed
  template upgrade` (and any `/releases/latest`-based resolution) follows.
  Do not write a `commit` line: the lockfile cannot record the SHA of the
  commit that contains it; consumers resolve your immutable release tag
  instead: which is why the seed-anchor-style tag protections in §1
  extend to release tags.

### Two distribution channels

open-seed reaches your repo two ways, and they carry different things.

- **The template channel** carries **structure**: `.seed/**` (config,
  guardrails, port schema, workflow DAGs, hooks), the Makefile, the CI
  definitions, the docs. It arrives once by GitHub template instantiation
  and afterwards by `scripts/seed template upgrade`, a three-way merge onto
  a branch you review. Structure is what a repo *is*, so it changes by merge.
- **The plugin channel** carries **capabilities**: agent skills and the
  role/subagent definitions. It is a Claude Code plugin published from a
  marketplace manifest in the open-seed repo, and Claude Code installs it
  for anyone who trusts the folder. Capabilities are what your agents can
  *do*, so they can change without touching your tree.

This is the R8 mitigation the template repo alone cannot provide (design
§10 Q4). A clone freezes the template at instantiation time; the plugin
channel lets capability updates arrive without a merge, while structure
keeps its reviewed merge path.

**Which one is right.** If you never enable the plugin channel, nothing
changes: the template channel already carries every capability in-tree, and
that is the supported default. Enable the plugin channel when you run
several instantiated repos and want skill and role updates to land in all
of them without N template merges. Do not enable it expecting structural
updates: `seed template upgrade` remains the only path for those.

**Seed workflows stay on the template channel.** A Claude Code plugin's
`workflows/` directory holds Claude-native dynamic-workflow scripts, which
are not the YAML DAGs `seed workflow run` executes. Reaching those from a
plugin cache would couple the engine to a harness-owned path, so
`.seed/workflows/*.yaml` travels with structure. §10 Q4 records this.

**Opting in.** `scripts/seed plugin enable` composes the declaration into
`.claude/settings.json`, reading the repo and ref from `.seed/template.lock`
so both channels name one release:

```json
{
  "extraKnownMarketplaces": {
    "open-seed": {
      "source": { "source": "github", "repo": "shaunlmason/open-seed", "ref": "v0.1.0" }
    }
  },
  "enabledPlugins": { "open-seed@open-seed": true }
}
```

The doubled `source` key is the real schema, not a typo: the marketplace
entry's `source` field is itself an object with its own `source`
discriminator. `.claude/settings.json` is control surface, so the command
composes the edit and you merge it through the ordinary gates.
`scripts/seed plugin disable` removes exactly those two entries.

**How drift is caught, on both halves.** The plugin package (`plugin/**`)
and the catalog (`.claude-plugin/marketplace.json`) are *generated
fan-outs*, rendered by `seed sync` from `skills/` and `.seed/agents/`
exactly like `.claude/agents/` and `.agents/skills/`. So `seed sync
--check` already fails offline if the published channel and the in-tree
sources disagree, and it runs in `make check` and CI. The other half is
cross-channel: `scripts/seed plugin status --check` fails when the
marketplace ref you pinned no longer names the release
`.seed/template.lock` does. A repo that has not opted in passes trivially.

**Honest limits.**

- A git-based **marketplace** source pins by `ref` (branch or tag) only, not
  by commit SHA. So the plugin channel is tag-trust, not hash-trust: weaker
  than `seed.lock`, which pins third-party skills by commit and content
  hash. Re-pointing a tag upstream changes what you install. Individual
  plugin *sources* inside a marketplace can pin a `sha`; the marketplace
  itself cannot.
- Enabling the channel means Claude Code clones the marketplace repo when
  someone trusts the folder. That is a network fetch at session start, on
  every teammate's machine.
- `skills/managed/**` is deliberately excluded from the published package.
  Those are third-party skills your own `seed.yaml`/`seed.lock` pins;
  republishing them under open-seed's manifest would misattribute
  provenance and route around the lock's hash pinning.
- Maintainers: the plugin's version is `.seed/template.lock`'s `version`
  line, so the release checklist above already moves both channels. Run
  `scripts/seed sync` after the bump so the rendered manifests follow.

## 7. Where everything lives

| Path | What |
|---|---|
| `.seed/` | The contract: config, guardrails, port spec, roles, teams, hooks, engine pin |
| `plans/` `receipts/` `memory/` `decisions/` | Work products, each with its own gate |
| `flavors/` | Opinionated stack variants (v2): applied by `scripts/seed-flavor install` |
| `plugin/` `.claude-plugin/` | The Claude Code plugin channel: generated by `seed sync`, do not edit |
| `scripts/seed` | The only coordination entry point |
| `scripts/loop.sh` · `scripts/seed-harness` | The loop and the harness adapters |
| `seed-state` ref | Machine-written coordination state, never checked out, never hand-edited |
| `.github/workflows/` | check-validate + maintenance (live); dispatch + pr-review (inert until secrets) |
