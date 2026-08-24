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
