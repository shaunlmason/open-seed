# Learnings

Append-only, via task PRs (D5). One dated bullet per durable insight about
*this codebase*: build quirks, API gotchas, decisions that keep resurfacing.
Fresh sessions read this file instead of rediscovering.

- 2026-08-22 (os-94ac6371): database/sql's `Begin` issues a *deferred*
  BEGIN: the fastcards store forces the write lock up front with a no-op
  `UPDATE meta` so the whole verb serializes (the plan's BEGIN IMMEDIATE
  semantics); rely on `busy_timeout` + a bounded jittered retry around it,
  never on raw SQLITE_BUSY reaching the caller.
- 2026-08-22 (os-94ac6371): `Mutation.Events` appends onto the run log read
  at the *pre-mutation* head: a migration that imports a run log must fold
  its own event into the imported content instead, or the append clobbers
  the migrated history (caught in the export/import round-trip test).
- 2026-08-22 (os-69bd5a64): a fake substrate must apply a mutation
  atomically: validate the whole input before writing any of it. The
  first refused-assign fixture applied `stateId` then errored on
  `assigneeId`, leaving a partial write no real GraphQL mutation could
  produce, and the adapter's (correct) compensation looked broken.
- 2026-08-22 (os-69bd5a64): when a port spec splits I/O declarations
  (`verbs.json`) from the wire schema (`envelope.schema.json`), fields
  land in one and not the other silently: the shim drops what the
  schema omits, so verb outputs existed on paper while callers never
  saw them. Cross-check the pair whenever either grows a field.
- 2026-08-22 (os-23494e11): `git fetch <url> <ref>` with no destination
  refspec stores no local ref: resolve what arrived through
  `FETCH_HEAD^{commit}` (which also peels annotated tags); pairing the
  fetch with `--no-write-fetch-head` leaves the objects unnameable.
- 2026-08-22 (os-23494e11): a lockfile can never record the SHA of the
  commit that contains it (writing the SHA changes the tree, which
  changes the SHA). Record the tag at release time and let the consumer
  stamp the resolved commit after the fact: provenance splits into
  "what the release declares" and "what the consumer verified".
- 2026-08-22 (os-52b9aed0): a gate is only trustworthy if it runs
  BEFORE the action it guards, and mock mode is only trustworthy if it
  stubs side-effecting `run:` steps too: "zero credentials" without
  "zero side effects" lets a mock run of a landing workflow reach a
  real merge.
- 2026-08-22 (os-52b9aed0): a review loop needs a remediation step
  wired into the gate itself; re-invoking a read-only reviewer on an
  unchanged implementation converges on nothing and burns
  max_revisions.
- 2026-08-22: bd v1.2.2 live validation (os-435d7b61): real `bd show --json`
  returns an ARRAY (fake had returned a bare object): every object-shaped
  jq read silently fell back, degrading fence-token minting; normalize
  shapes at ONE adapter seam (`show()`), not per call site. Real `bd list`
  hides closed issues (use `--all` when the port must surface terminal
  cards); comments are listed via `bd comments`, not `show`. Shared corpus
  file (`corpus.sh`) sourced by both offline and live suites is what keeps
  a fake and its real counterpart from drifting apart silently.
- 2026-08-22: multi-squad activation (os-10c10aae): a fallback scope
  (core's bare `**`) must be EXEMPT from pairwise overlap validation or
  no second squad can ever validate: "matches what nothing else claims"
  necessarily intersects everything; only two catch-alls or two specific
  scopes overlapping are real conflicts. Activation checks that read
  shipped placeholder config (core's example mission) fire on every
  fresh clone: gate them on a deliberate act (>1 squad || a real
  mission) and ship the placeholder commented out.
- 2026-08-23 (os-488323ec): the D7 done-consistency exemption is the
  `no-pr:` evidence PREFIX, minted only by a `--no-pr` close (engine
  task.go `record_review` effect): a plain CLI `task close` on a
  cross-repo card (no template plan, no `seed/<id>` PR) passes its own
  transition but leaves the state ref conformance-failing, and `done`
  is terminal with no re-record verb: the next maintenance tick writes
  HALT and blocks every mutating port verb until a human resumes. The
  recovery is one replay-legal ref commit (evidence edit, state
  unchanged, run-log line, one commit) plus `state resume`; better,
  close no-PR work through the `seed-close-no-pr` workflow_dispatch so
  the marker mints itself.
- 2026-08-23 (state-ref repair): the D7 no-PR exemption lives in the
  evidence string, not the run log, a done card whose work never landed
  as a `seed/<id>` PR must have evidence starting `no-pr:` (the
  `accept`/`close --no-pr` flag mints it), else the done-consistency lint
  HALTs the whole state ref: "done without a resolvable plan". The
  marker alone is not the proof: D7 requires a server-attributed artifact
  behind it (the seed-close-no-pr workflow's run URL, or a comment/issue
  by the closing human): a bare `accept` mints no such artifact, so any
  repair must create one and cite it in the evidence (this incident:
  issue #51, cited 2026-08-23). Repair mechanics: one commit per card
  (front-matter evidence edit + a run-log comment line); the replay lint
  allows same-state card edits, and `state resume` is exempt from the
  HALT (checkHalt=false) so it can always run.
- 2026-08-24 (os-501a29c2): `seed template upgrade` merges by **path**, and
  that single fact governs anything that copies template content into a
  consumer's repo. A flavor that installs `flavors/typescript/package.json` to
  `/package.json` never receives updates: upstream has never owned the root
  path, so the merge cannot reach it. Two rules follow, now binding in
  `flavors/README.md`: keep every destination outside `flavors/` **thin and
  stable** with the churn in the flavor directory (the root `tsconfig.json` is
  two keys that `extends` a base file, so compiler-option churn arrives with
  no reapply at all), and record a real merge **base** — not just a hash — for
  the residue that cannot be thinned, or reconciliation is a clobber rather
  than a merge. `.seed/flavor.lock` records what was written;
  `.seed/flavor-base/` stores the payload as installed, which is what
  `git merge-file` needs as its base.
- 2026-08-24 (os-501a29c2): a test that instantiates the template and runs
  `make check` inside it **cannot** live in `scripts/validate.sh`, because a
  flavored `check` runs `validate`, which runs `validate.sh` again. `make
  smoke` has always been a separate target for exactly this reason. The trap
  is easy to re-enter through a side door: the core-gate-independence check
  (two `make check` runs, compared) looked like a cheap filesystem assertion
  and was not — it tripled the gate's runtime and only terminated because of
  the `SEED_FLAVOR_TEST` guard. Split such suites explicitly: a `--offline`
  half that never shells into `make`, and an integration half behind its own
  target.
- 2026-08-24 (os-501a29c2): TypeScript 7 does not auto-include `@types`; a
  `node:` import fails to typecheck until `"types": ["node"]` is set. When
  splitting a `tsconfig` across `extends`, remember that **path-relative**
  options (`typeRoots`) resolve against the file that *declares* them, so they
  belong in the root config next to `node_modules`, while by-name options
  (`types`) resolve fine from an extended base. Getting this backwards is a
  silent typecheck failure, not a config error.
- 2026-08-24 (os-2c0c474c): a backend fake written from the same reading of
  the docs as the adapter proves only that you were **consistent**, not
  correct. paperclip's contract suite passed every verb for two releases; the
  first run against a real instance falsified the whole transport layer
  (single-issue routes are not company-scoped, the field is `status` not
  `state`, issues carry no `metadata`, checkout needs an agent-row UUID, deps
  are written `blockedByIssueIds` and read `blockedBy`, there is no `/events`
  route). Ship the live suite *with* the adapter, share one corpus between fake
  and live so they cannot drift, and treat any fixture assertion you cannot
  point at a live response for as an untested assumption.
- 2026-08-24 (os-2c0c474c): Paperclip returns **200 for a PATCH naming a field
  it does not have**. That is the drift class to fear: not an error, a silent
  success, so every transition reported OK while nothing moved. Never assert a
  state change from a write's response on this backend: re-read the entity. The
  re-read is the only reason the `state`/`status` bug cannot come back.
- 2026-08-24 (os-2c0c474c): Paperclip's checkout **dispatches work**, it is not
  a passive lock, so it is not the port's `claim`. Assignment wakes the agent,
  a runtime-less agent's run fails, and
  `recovery.reconcile_stranded_assigned_issue` moves the issue
  `in_progress -> blocked` within ~10s. Agents a seed deployment owns must be
  paused (`PATCH /api/agents/<id> {"status":"paused"}`; ignored on create).
  Diagnosis came from the platform's own activity log, which names the mover in
  `details.source`: read the substrate's audit trail before theorising about a
  flake.
- 2026-08-24 (os-2c0c474c): the per-issue document store enforces optimistic
  concurrency (an update without `baseRevisionId` is refused 409), which is a
  real compare-and-swap, but it protects **only the document**. No other
  Paperclip write takes a revision precondition, so a CAS there does not make a
  fence atomic: the issue mutation is always a separate unconditional request.
  Use it to assert the claim *before* touching the substrate (a superseded
  worker is then refused instead of half-applying), and declare the residual
  window rather than upgrading the guarantee. Claiming atomicity from a
  partial CAS was a review catch, not a self-catch.
- 2026-08-24 (plugin channel, os-221f5929): Claude Code **silently
  skips** a subagent file missing `name` or `description`: the parse
  error goes to the debug log, never the session, so the four roles in
  `.claude/agents/` had been loading as nothing at all since the fan-out
  landed. Nothing failed, which is why it survived. D8 already required
  the sources to be dual-format; the drift was in the files, not the
  design. Lesson generalized: when a fan-out targets a harness, the
  harness's own validator (`claude plugin validate <dir>`) is worth
  running in the gate, because "no error" from a harness usually means
  "not loaded", not "loaded fine". Watch the YAML too: an unquoted colon
  in a `description:` makes the whole frontmatter invalid.
- 2026-08-24 (generated trees): a fan-out that only ever *writes* its
  desired files cannot notice deletions. Deleting a source left the
  generated copy on disk with `sync --check` still clean, so a removed
  capability kept shipping through the published plugin. The fix that
  made this tractable was making the owned roots **100% generated**,
  marker README included: once no hand-written file lives under a root,
  "delete anything the render did not produce" needs no exception list.
  The same gap still exists for `.claude/agents/`, `.claude/skills/` and
  `.agents/skills/`, whose roots hold hand-written READMEs (follow-up
  card os-fa598c76).

- Seed (next/**) loop mechanics: `backlog→ready` promote and plan-unblock are
  operator-class with no agent path when dispatch is inert; the Seed workstream
  runs them under the session principal per decisions/0003-next-loop-delegation.md.
  Symptom if you hit it: `{"error":"operator_required"}` exit 3 on promote.

- Never commit private-key-shaped bytes, even synthetic test fixtures: GitHub
  push protection blocks the push (GH013) regardless of the key being a
  deterministic dummy. Generate key fixtures at test runtime into t.TempDir()
  (x/crypto ssh.MarshalPrivateKey emits the same wire format the loaders parse).

- 2026-08-30 (os-beac85e1): Parallel `next:` task PRs conflict on exactly
  two kinds of line: tail appends (`next/docs/decisions.md`, adjacent
  exit-code table rows) and the shared frontier. The pattern that kept
  five simultaneous PRs merge-order-independent: resolve tail appends by
  keeping both sides in order, and make every in-flight PR carry a
  **byte-identical** `next/docs/progress.md` (identical changes never
  conflict; the first merge lands the file, the rest rebase to a zero
  delta). Receipts must be regenerated after every rebase since they bind
  to the merge-base diff.

- 2026-08-30 (os-895bf828): git pre-receive hooks run in a quarantine
  that forbids ref updates ("ref updates forbidden inside quarantine
  environment"), so a hook cannot move the guarded ref mid-push. For
  deterministic race drills the escape is unsetting GIT_QUARANTINE_PATH
  inside the test hook (safe when the rival commits already live in the
  main object store); the resulting client-visible rejection is
  "[remote rejected] ... (failed to update ref)", which is genuine
  update-phase contention, distinct from "(pre-receive hook declined)".
- When a protocol version joins the build's supported set, every test
  that used it as the "unsupported future version" flips silently; use a
  far sentinel (seed/9) for refusal fixtures and pin narrow sets with
  WithSupportedVersions explicitly instead of leaning on the default.
- The next/ coverage gate's cross-package -coverpkg aggregation can
  mis-merge (warm mixed-flag caches routinely; even a cold run
  occasionally), reading 85-89% on a 92%+ tree. The tell is an
  impossible per-function number (a fully-tested function at 36%): that
  marks a bad profile merge, not missing tests. Truth procedure:
  cd next && go clean -testcache && go test ./... \
  -coverprofile=coverage.out -covermode=atomic -coverpkg=./internal/...
  repeated until two runs agree, sanity-checking per-function output,
  then make check against the warmed cache.
- Conformance checklists are conjunctive: a specific row (the cache's
  throughput/zero-authority/deletion row) never narrows a universal one
  ("every read surface rebuilds byte-identically"); reading redundancy
  into overlap authorized a design the charter forbade, and the fix
  (engineered SQLite byte-determinism: one ordered transaction,
  rollback journal, fixed page_size, no auto_vacuum/ANALYZE) was
  cheaper than the exemption it replaced.
- An owner can merge a plan PR while its review amendment is still in
  flight: the merged file, not the branch, is what authorizes
  implementation. Before implementing any plan, diff the merged copy
  against the amended branch; a stranded amendment re-opens as a fresh
  plan PR and the card re-parks on it.
