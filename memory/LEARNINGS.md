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
- File-mode enforcement drills need an unprivileged runner: uid 0
  bypasses permission checks (CAP_DAC_OVERRIDE), so refusal
  assertions gate on os.Geteuid() != 0 and the dev-container root run
  skips them while CI exercises them. Emulate CI before pushing with
  a compiled test binary under setpriv --reuid=nobody; that run, not
  the root run, is the one that catches t.TempDir cleanup failing on
  locked trees (testing's own RemoveAll cannot descend 0555 dirs, so
  every test publishing locked output must register an unlock
  t.Cleanup, which LIFO-runs before the framework's).
- 2026-08-30 — The unprivileged emulation must cover every package
  that publishes locked output, cmd/seed included, not only the
  engine package: the #118/#119 restack went red in CI on a
  cmd-level refusal drill whose locked proj2/proj3 layouts lacked
  unlock cleanups, while the local root run and the engine-only
  nobody-run both passed. Before pushing mode-touching tests, build
  and run each affected package's test binary under
  setpriv --reuid=nobody; a green root run proves nothing about
  cleanup under 0555.

- Read the charter Appendix verb catalog before binding any event
  vocabulary: the 5.1 plan invented promote/accept/submit names that
  the catalog already assigns (contract.specified, merge.observed,
  submission.made), and every later phase is scheduled around the
  canonical strings.
- A stacked PR "merged" into its base branch has not reached main:
  the #111/#112 stack collapse folded both diffs into side branches
  after the stack root had merged, so the content needed re-landing
  PRs rebuilt from main. Verify the files exist on main before
  closing cards or claiming a phase exit.
- 2026-08-31 — A same-key race fixture can converge rivals onto
  byte-identical records: identical drafts (same signer, fixed ts,
  same prev) build the same commit, and git treats pushing an
  already-landed commit as an idempotent success, so every "rival"
  reports victory while the chain correctly holds one record — the
  race was never real. Give each simulated rival a distinct draft
  (its own ts, or its own key) and assert the exact count of the
  contested record on the converged chain, not just total length.
  The storm only surfaced under the slower unprivileged run: timing
  shifts change which fixture flaws fire, one more reason the
  nobody-run is part of the gate.

- The coverage mis-merge root cause: `go test ./...` runs package
  test binaries concurrently, and under subprocess-heavy drills
  (hundreds of short-lived git/CLI children recycling pids) two
  binaries can write coverage counter files with colliding names
  (same pid, same second), so the merged profile silently loses one
  package: that package's functions then read as covered only where
  OTHER binaries exercised them via -coverpkg. Fingerprint: exactly
  one function craters between runs while its isolated package run
  is fine. Fix: `-p 1` on the coverage invocation (three cold runs
  agree to the decimal); receipts embed their own run, so READ the
  recorded exits before claiming a receipt green.
- The mis-merge poisons the testcache too: a cached "ok" replays the
  coverage data recorded when it ran, so one bad parallel run keeps
  resurfacing in every warm run after it (87-89% readings with -p 1
  already in place). After adopting -p 1, flush once
  (go clean -testcache); fresh entries then stay consistent.
- Never chain `git rebase` with `reset --hard`, receipt generation,
  or push in one command line. A conflicted rebase parks HEAD on the
  new base; the chained reset then amputates whatever the base's
  last commit is, and receipt+push will happily validate and publish
  the crippled branch, because a receipt proves the tree it saw runs
  green, not that the tree holds the content you meant. This
  destroyed two branch tips in one night (one was pushed: a
  receipt-only PR branch with its implementation commit gone). Run
  the rebase ALONE, read its exit and log, and only then reset,
  receipt, and push.
- Sweep a PR's review threads immediately before and after merge: a
  round posted between the last sweep and the owner's merge click
  otherwise lands on merged history silently. Findings on merged
  content are still owed fixes, restarted from main as a follow-up
  change, with replies carrying real shas and threads resolved.
- The coverage mis-merge has a per-function fingerprint: the merged
  `./...` profile nondeterministically drops one test binary's
  counters, so exactly one function craters (Parse 89.7% isolated vs
  46.4% merged) while the rest hold. Diff `go tool cover -func`
  between runs to spot it, disprove with the isolated package run,
  and only trust two agreeing cold uniform-flag runs.
- The coverage mis-merge's real root cause: the shared Go BUILD
  cache, not the testcache. Many worktrees of the same module leave
  stale instrumented objects for identical import paths; a run in a
  freshly switched worktree can link them and mis-attribute counter
  rows, so totals misread anywhere from 89% down to 80% while the
  suite passes and CI (always cold) stays green. go clean
  -testcache never purges it; `go clean -cache -testcache` before a
  receipt in a switched worktree restores the true reading
  immediately. Keep reading recorded receipt exits with a checker
  that sys.exit(1)s on any nonzero, and never let a retry loop's
  echo swallow that status into a chained push.
- The receipt gate makes every task PR plan-first in practice,
  whatever the card's tier: `seed receipt generate`/`verify` refuse
  without a merged plan at the merge-base (D3), and CI's verify job
  demands the receipt on every `seed/<id>` branch — so even an L1
  administrative card routes through a plan PR. Budget the human
  review round-trip up front instead of discovering it at receipt
  time (os-6e37b10e).
- Normative spec tables that are parsed by tests treat every
  backticked token in a data cell as data: keep prose asides in
  those cells unticked, or the parser reads the aside as vocabulary
  (the actors.md capability table's parser caught exactly this when
  the verdict row's aside quoted the capability name).
- Prior art worth consulting when sealed checks (6.3) and Phase 7
  metering get planned: 27-GROUP/kveritas-go (Apache-2.0, reviewed
  2026-08-31 at ab0b1f8) independently converged on much of the
  verdict pipeline's shape — canonical-JSON receipts under signature,
  a hash-chained agent-session log with tamper localization, salted
  commitments with disclosure levels for private artifacts, and
  selective-disclosure Merkle proofs that reveal one leaf against a
  signed snapshot. Its two ideas Seed lacks: metric-blind
  execution-coherence scoring over run telemetry, and physical-bounds
  compute attestation — the attestation answer to work recomputation
  cannot verify, a natural fit over the 5.6 observation streams.
  Design input to verify against the charter, never authority; the
  6.3 card carries the detailed pointer (comment cm-126a17b7).

- agessh makes "recipients = the verifier keyring" literal: filippo.io/age's
  agessh package derives X25519 age recipients and identities from ordinary
  ed25519 keys (ssh wire form), so an encrypted-to-the-keyring design needs
  no parallel key enrollment; the header's `-> ssh-ed25519 <tag>` stanzas
  carry base64(SHA-256(wire pubkey)[:4]) tags, auditable against the keyring
  without any secret. Two caveats worth repeating: it is cross-protocol use
  of a signing key (fine as a documented v0 trade, name the dedicated-keys
  successor), and the tags are four bytes — identification hints, not
  exclusivity proofs, so keep a real decrypt drill beside any tag audit.
- Adding fields to a hash-anchored artifact (the verdict receipt) wants
  `omitempty`, not the explicit-null convention used for versioned views:
  omitted-when-absent keeps every previously stored artifact's canonical
  bytes and digest stable, so old receipts still recompute-and-match, while
  views republish under a version bump anyway and can afford explicit null.
  One compatibility rule per surface kind, chosen by what anchors identity.

- When an authorization consults "the latest X" from a tolerant fold,
  ask what a raw-pushed later X does to it: latest-wins facts let an
  attacker bury an authentic fact under a garbage one (a raw fail
  hiding a real fail would have unlocked the 6.4 red-verdict
  lockout). The fix shape: keep the whole window's facts in the fold
  (cleared at the window boundary) and authorize against any
  authenticated member, never only the latest.
- cmd/seed test infra: genesis.Bootstrap's resolver knows genesis
  keys only, so verdictLibAppend can raw-append solely as the root;
  raw-pushing as any other enrolled key needs a loose resolver that
  short-circuits that key's fingerprint (rawAppend in the lockout
  drills). Symptom otherwise: "actor fingerprint not in the keyring"
  on a key that is genuinely enrolled.

- The laundering countermeasure now has a fixed shape worth applying
  on sight: any admitted step that trusts a folded fact must both
  (a) validate the fact's signer against its authoring boundary and
  (b) revalidate the fact's own citations, because the tolerant fold
  records any well-shaped raw push. 6.2 learned it for verdicts, 6.3
  for seals, 6.4 for fails and overrides (each time as a review
  finding or a preemption of one); Phase 7's offers and budget
  reservations will meet the same pattern, so design it in from the
  plan, not the review round.

- The next CLI holds exclusive verbs (claim.taken) to the online
  path: `seed ledger append` refuses them offline by design, so CLI
  drills that need a claim run the same admission sequence through
  the library (ContextAt, Check, Append with the context resolver) —
  which is also exactly the shape that lets a race drill assert the
  loser's structured contention refusal instead of a process exit.

- `seed task plan-unblock` takes the PR as `--pr N` (it builds the
  `plan:N` entry itself); passing the entry via `--blocked-on` or a
  URL yields `invalid_transition`, the same error as a wrong state.
- 2026-09-01 (os-ef715d17): a best-effort writer paired with a strict
  reader needs a commit marker, or one torn write poisons every later
  read forever. The attempts journal writes best-effort (journaling
  must never fail the verb it rides) but is read strictly (a declared
  input's garbage is the declarer's error), so a short write under
  disk pressure would have refused every future report build; the
  terminating newline became the commit marker and an unterminated
  final fragment is ignored, while terminated lines stay strict.
- 2026-09-01 (os-ef715d17): a cache several processes rewrite needs a
  per-writer temp name. The ledger's HEAD used one shared `HEAD.tmp`
  while both the append path and `Open`'s repair path wrote it, so a
  reader's rename could consume the appender's temp and fail its
  rename with ENOENT — a CI flake that only reproduced under
  contention. Also preserve the target's mode: `os.CreateTemp` opens
  at 0600 and a rename carries that over whatever the file had.
- 2026-09-01 (os-ef715d17): advertisement/enforcement drift is
  testable as a class, not just at curated stations. Sweep every
  prefix position of one shared scenario, and at each position
  independently re-draft every advertised option and run it through
  the enforcing path: today the property holds by construction, and
  the sweep is what keeps a later split (caching, memoization, a
  parallel derivation) from drifting silently. Curated
  presence/absence assertions are not equivalent coverage — they
  never re-draft what they list.
- 2026-09-01 (os-ef715d17): a position stamp is only useful if the
  reader knows which ordinal it carries. `stampTip` writes the
  zero-based tip position while a context's `Count` is the record
  count, so any check comparing them must use `Count - 1`; the
  distinction is easy to get wrong in exactly the drill meant to
  prove the stamp trustworthy.
- 2026-09-01 (os-768361cc): when a document you are writing reaches
  a conclusion that serves the goal you were just handed, re-derive
  it from the authority text adversarially before shipping it. Three
  independent review findings on one plan turned out to be a single
  failure — motivated reading — and each would have been caught by
  quoting the source instead of paraphrasing it. Two specific
  shapes: reading a collective claim distributively ("Phases 0-12
  deliver core conformance" so any subset suffices), and splitting a
  defined term so an existing escalation lands only on the step you
  were not proposing.
- 2026-09-01 (os-768361cc): place security gates by WHEN AUTHORITY
  MOVES, not by when visibility increases. Publishing feels like the
  bigger exposure than an internal cutover, and is not: the cutover
  is when a stolen key starts reaching real state, so the drill that
  proves the ceiling belongs before it.
- 2026-09-01 (os-768361cc): machine-checkable declarations outrank
  narrative shortcuts. A phase's `deps: all` and an exit line's
  named criteria are facts about sequencing; check them before
  amending order, because prose elsewhere can read as permission
  they do not grant.
- 2026-09-01 (os-52d5da3f): build the drift class WITH the surface,
  not after it. The obligation sweep failed on its first run and the
  failure was a real modeling error (a pass verdict advertised
  merge.observed while admission wanted merge.requested first: the
  merge chain is two events). A class that re-derives the advertised
  act through the enforcing path finds this in seconds; review and
  reasoning had both missed it.
- 2026-09-01 (os-52d5da3f): when advertising what discharges an
  obligation, separate WHAT ends it from WHO may do it. Asserting
  that every discharging verb admits for the owed party forces
  capability policy into a derivation; asserting that at least one
  does keeps the derivation a projection and still catches the
  undischargeable case.
- 2026-09-01 (os-52d5da3f): a position stamp is a tip ORDINAL, so the
  prefix a client that stopped there last saw is records[:pos+1].
  Both times this codebase has reasoned about a stamped position
  across a boundary, the off-by-one appeared; write the +1 with a
  comment naming the ordinal.
- 2026-09-01 (os-7e197768): a CLI that wraps a protocol seam earns
  its keep only if it consults the boundary the seam ignores.
  `ledger append` signs and appends without running admission at all,
  so wrapping it in nicer flags would have shipped the same
  after-the-fact refusal in better clothes. The value is the
  pre-flight: run the SAME check admission runs, refuse before
  signing, and render what IS legal beside what was not.
- 2026-09-01 (os-7e197768): when an argument has exactly one legal
  value, asking for it is an invitation to be wrong. The fence, the
  sole open reservation and the approved plan anchor are each pinned
  by a rule that refuses every other value, so a flag could only ever
  carry a mistake. Derive those; keep flags for judgments (amounts,
  actuals, prose).
- 2026-09-01 (os-7e197768): three shapes of underivability want three
  different owners. A missing fact and an ambiguity refuse in the
  tool (naming what would establish one, or the candidates it will
  not choose between); an absent precondition should be left to the
  boundary, whose refusal names the state and beats anything the
  derivation could say. Deciding this per shape rather than per verb
  kept the error surface small.
- 2026-09-01 (os-68ea0b2d): shipping a surface and obliging its use are
  two different pieces of work, and the second is the one that gets
  forgotten. Phase 9 item 5 built the lane-facing surface; nothing said
  the lanes had to call it, so a conforming-on-paper worker could still
  hand-assemble protocol arguments. When a phase delivers a capability
  another phase is supposed to consume, name the obligation in the
  CONSUMING item's own text, or the capability has no customer.
- 2026-09-01 (os-68ea0b2d): before transferring an obligation from an
  older system, check the mechanism still exists. "Lease renewal rides
  every holder-signed verb" was v1 vocabulary; Seed has no lease at
  all, and liveness is classified from an observation stream. The
  transferable obligation was sharper than the original once restated
  in the target's own terms, and would have been nonsense if carried
  over literally.
- 2026-09-01 (os-68ea0b2d): prefer enforcement by vocabulary over
  enforcement by detection when the signal has no discriminator. A lint
  for "heartbeat-shaped" observation streams cannot work, because a
  legitimate long-running step emits the same shape; but a loop whose
  verb set contains nothing whose only purpose is liveness cannot emit
  a heartbeat at all. Ask what the system can be UNABLE to do before
  asking what it can detect.
- 2026-09-01 (os-9b3f3ef3): when a retry loop re-runs validation, ask
  what else it should be re-running. gitref's AppendLoop re-fetched,
  re-signed and re-judged per attempt, which looks complete until you
  notice the PAYLOAD was computed once, outside it. Anything derived
  from a view must be recomputed wherever that view is refreshed, or
  the refresh is a half-measure that reads like a whole one.
- 2026-09-01 (os-9b3f3ef3): on divergence, refuse rather than
  re-derive-and-proceed. Substituting a freshly derived value looks
  like the helpful fix and is the dangerous one: the caller's act was
  authorized against a view that no longer exists, so a different
  value is a different decision, not a better argument. Refuse and
  name what changed; let the lane re-orient.
- 2026-09-01 (os-9b3f3ef3): a position stamp that is merely PRESENT is
  not correct. `remoteFailureEnvelope` computed the right refreshed
  position and the caller's helper then overwrote it with a stale one,
  which is worse than never stamping: it inverts the concurrency
  signal the field exists for. When two layers can both stamp, make
  the outer one defer to a stamp that already exists.
- 2026-09-01 (os-9b3f3ef3): a malformed-input drill is only as wide as
  the shapes it tries. Every bad packet the drills used was an object,
  so the JSON value `null` — which unmarshals into a nil map with no
  error — walked past them into a panic. When a decoder is tolerant of
  a whole VALUE class, test the class, not the variations within one
  member of it.
- 2026-09-01 (os-c4e8b57a): a fixture sweep that lists the fixtures
  misses the repositories production code creates. The repository that
  actually lost this race in CI was `<stateDir>/gitdir`, which
  `gitref.NewClient` inits under a `t.TempDir` the test hands it: no
  fixture line exists to harden, so per-repo writes alone would have
  shipped a fix that did not fix the observed failure. `GIT_CONFIG_GLOBAL`
  in TestMain covers every git process the binary spawns, whoever spawns
  it, and needs no production change. When hardening a test environment,
  ask who ELSE creates the resource, not only which fixture does.
- 2026-09-01 (os-c4e8b57a): a cleanup flake fails AFTER the assertions
  pass, which is the worst shape for an unattended loop: the signal says
  "your change is broken" when the change is fine, and the correct
  response — re-run once, then treat a second failure as real — is a
  rule an agent has to apply against its instinct to hunt a bug it did
  not write. Removing the race is cheaper than paying that cost on
  every future red.
- 2026-09-01 (os-c4e8b57a): give a source-walking guard a floor on what
  it found. A regex over a tree fails silently in the direction that
  looks like success: match nothing, pass everything. Asserting a
  minimum number of detected sites turns "the pattern rotted" from a
  green build into a red one.
- 2026-09-01 (os-d6963652): when a gate is written against a verb
  FAMILY, check whether the family shares the reason. Admission gated
  reserve, settle and release on `in_progress` in one line, but only
  the reserve has a claim on the window: closing a reservation
  honestly is wrong in no state. The derivation half had said so all
  along — `BudgetCloseValid` validates a close by identity alone —
  which is the tell for an over-broad admission gate generally: when
  the derivation asks less than admission does, one of them is guessing.
- 2026-09-01 (os-d6963652): "detect it and let maintenance reap it" is
  not an alternative to fixing an admission rule when the rule is why
  nothing can reap it. The maintenance lane is audited as an ordinary
  actor precisely so it has no private powers, so with the gate in
  place NO act freed the capacity, for anyone. Detection without an
  admissible remedy is a report nobody can act on.
- 2026-09-01 (os-d6963652): an obligation is owed by whoever can
  discharge it, which is only usually whoever opened it. The
  `budget.open` row was attributed to the claim holder, and admission
  closes a reservation for its own signer or the operator lane and
  nobody else — so on any reservation the holder had not signed, the
  row named a party admission refuses. Standing widens it further:
  `HasAnyCapability` is standing-aware, so a suspended signer can no
  longer pay a debt keyed to their fingerprint, and a keyed read then
  hides it from the operator who can.
- 2026-09-01 (os-d6963652): a conformance walk of only ACTIVE actors
  cannot reach the positions where standing decides anything. Ending
  the shared scenario by suspending and then revoking a lane that
  still holds an open reservation cost two script lines and turned
  ownership attribution from an assertion into a swept class — it
  fails on mutation, which the hand-picked drill alone would not.
- 2026-09-01 (os-d6963652): changing WHO owes something is a change a
  position-keyed delta cannot see. `seed situation --since` reported
  rows whose `Since` advanced, and an ownership transfer deliberately
  keeps the position it arose at — so the transferred row was
  "unchanged" for the new owner, while the removals, derived from the
  prior set filtered to the caller, never held it either. The one
  party able to act heard nothing, in the mode documented as a
  complete change report. When a projection gains a field that can
  move, check every consumer that decides freshness from a different
  field.
- 2026-09-01 (os-fa69345e): when a read surface refuses, ask what the
  read already established. `show --position` scanned the entire chain
  and then discarded the count on the not_found path, which is the one
  refusal where a concurrent append is most likely: a caller asking
  for position N on a chain of length N cannot otherwise tell "not
  yet" from "never". The distinction that matters for a position stamp
  is not success-versus-refusal but reached-the-data-versus-did-not.
- 2026-09-01 (os-fa69345e): a partial scan is not a short chain.
  Stamping the count an aborted iteration reached would assert a tip
  the failure disproves, so the error branch stays unstamped even
  though a count sits right there in scope. Symmetry between sibling
  branches is not a reason on its own; each has to earn its stamp.
- 2026-09-01 (os-a95db3f5): when a drill sleeps, ask what the sleep is
  standing in for. Both windows in the preemption file stood in for
  "the worker is up", and each failed differently: the positive one
  flaked when a subprocess booted slower than 300ms, and the negative
  one PASSED VACUOUSLY when it did — nothing parked because nothing was
  running. The vacuous pass is worse in kind, because an assertion that
  succeeds for the wrong reason never fails and so never gets filed.
- 2026-09-01 (os-a95db3f5): raising a timeout makes a flake rarer
  without fixing it, because the assertion still says "within N
  milliseconds" where it means "at all". Poll a condition against a
  deadline instead, and treat the deadline as a failure bound rather
  than a pacing device: on a fast runner it returns in one interval,
  and a slow one only delays the proof.
- 2026-09-01 (os-a95db3f5): a negative assertion needs a handshake, not
  a window, and initial liveness is not enough — adding "and it kept
  growing across the window" would have traded a vacuous pass for a new
  instance of the same flake, since a descheduled worker fails a growth
  assertion while behaving exactly as it should. The worker's own loop
  supplied the handshake: its cycle emits a line THEN checks, so two
  lines after the event landed prove it evaluated the event and
  declined. Look for an ordering the system already guarantees before
  reaching for a duration.
- 2026-09-01 (os-cf1c9688): when a plan says "check X against the
  existing table", verify the table exists before believing it. The
  first draft pointed a validator at "cmd/seed's loop-verb table";
  there was none, and the acts were case arms in package main, which
  nothing can import. The plan would have failed on contact, in the
  one way it had explicitly forbidden: writing the seven names down a
  second time. A cited authority is a claim about the tree, and claims
  about the tree are checkable before they are binding.
- 2026-09-01 (os-cf1c9688): an OR-set read as an AND-set inverts the
  check. `AcceptedCapabilities` returns alternatives — any one admits
  the verb — so requiring a lane to hold all of them would have forced
  `operator` onto every lane that claims or spends, to make a
  capability-separation validator pass. When a check's green state
  requires the posture the system forbids, the check is backwards.
- 2026-09-01 (os-cf1c9688): state a least-privilege rule as an
  allowlist. The draft's "holds no authoring, verdict or sealing
  grant" admitted `operator`, the strongest capability there is,
  because a blocklist admits by default everything nobody thought to
  name. On the surface that reads hostile input, "we remembered the
  dangerous ones" is not a posture.
- 2026-09-01 (os-cf1c9688): when a validator cannot establish the whole
  obligation, split it at the seam where the EVIDENCE changes and
  record who inherits the rest. A manifest can say which work steps a
  lane calls its liveness sources; only a running loop can show they
  emit. Writing that down turned a silent overclaim into a named
  obligation on the next card.
- 2026-09-01 (os-cf1c9688): a flake that repeats is still a flake. The
  coverage gate reported the same failing number three times running,
  which is what a real regression looks like and what sent me hunting
  for the code that caused it — but `go test` caches a package's
  result INCLUDING its coverage contribution, so a warm re-run replays
  the same lost counters at the same number forever. Cold-cache runs
  gave 86.7 / 90.7 / 90.7. When a re-run is the discriminator, make
  sure the re-run actually re-runs.

## A capability table is not a reachability oracle (os-b779b4c7, os-abb206c8)

`keyring.AcceptedCapabilities` answers "which capabilities admit this
verb", and its switch falls through to `nil` for the standing-only
class — the verbs `internal/admit`'s grant rule describes as needing
"active standing only", `message.sent` among them. Deriving "what can
this key do" by filtering that table for a capability therefore misses
every verb in that class, silently.

The general shape, which cost three review findings in one round: a
lookup that *looks* like consulting an authority is only as complete as
the authority's own domain. When the question is "what is reachable",
ask the boundary — attempt the act and observe the outcome — rather
than asking a table built to answer a narrower question.

## A test double that guesses the boundary's shape teaches a false one

The loop's unit drills answered a refused reserve with
`{Exit: 9, Code: "budget"}`, invented from what a budget refusal ought
to look like. The real boundary returns `{Exit: 8, Code:
"chain_invalid"}` with the account in the message. The drills passed
either way, so the double would have quietly taught this package a
boundary that does not exist, and the packet-carries-the-refusal claim
would have been verified against fiction.

Doubles get their shapes from a run against the real thing, and the
end-to-end drill that produced the shape is what keeps them honest.

## Warm coverage readings hide the counter-loss flake (os-cafba959)

`make check`'s gate read 90.0%, 90.1% and 90.3% on warm caches while
cold runs of the same tree gave 89.3%, 90.7% and 90.7%. `go test`
caches a package's result including its coverage contribution, so warm
re-runs replay whatever counters the cached run lost, at the same
number, which reads as stability.

Consequence for any card landing near the gate: measure cold, several
times, and leave headroom for the full observed swing (1.4 points here)
rather than for the best reading.

## The coverage a package's own tests report is not what the gate reads (os-f262585a)

`go test ./internal/... -cover` ranks packages by what their OWN tests
reach, and the gate's profile is `./...` with `-coverpkg=./internal/...`,
where `cmd/seed`'s tests alone cover 72% of the internal tree. The two
rankings disagree wildly: `internal/reconcile` reads 31% own-test and
92% in the gate's profile, because the CLI drills drive it. Picking
targets off the own-test ranking spends the effort where it moves
nothing.

Rank off the merged profile instead, by uncovered statements rather than
by percentage: `go tool cover -func` on the gate's own profile, or a
per-block max rollup of it (the raw profile carries one block set per
test binary, so summing it without taking the max per block reports
nonsense). The packages that actually moved the number here were the
ones no CLI path drives: `internal/protections`'s forge adapters,
`internal/artifact`, `internal/posture`'s accessors and preseed
validator, and `internal/imported`, which had no `_test.go` at all
while `-coverpkg` counted its statements: check for that case directly,
because a package with no test binary reports nothing and reads like a
package nobody needed to test.

Two mechanical notes from the same measurement. This module's toolchain
has no `covdata`, so a `-coverpkg` run over a package with no test files
(`internal/imported`) prints `go: no such tool "covdata"` and `go test`
exits 1 under `-p 1` while still writing a mergeable profile: the
reading is valid, the exit code is the toolchain gap, not a suite
failure. And covergate cannot supply a cold acceptance reading at a
passing number, because its re-collection engages only below the gate.

## Running the thing finds what reading it cannot

Two gaps in os-abb206c8 were invisible to inspection and immediate once
a loop actually ran: the situation read withheld the acceptance anchor,
so a lane could not write the packet its own deliberate exit requires;
and the manifests declared a liveness source that emitted nothing,
because no loop verb ever touched `internal/obs`. Both had been read
past repeatedly, including while writing the plan that named liveness
as the inherited obligation.

## Characterization drills correct the prose that motivated them (os-b779b4c7)

The residual table asserted that `claim.reaped` "carries no precondition
beyond the capability check". The boundary refused the drill twice
before it passed: the reap must cite the active fence, and must carry a
four-part packet. Both survive as residuals — they are freshness and
attribution checks, not authorization, and a persuaded lane satisfies
them by reading and writing — but the claim as first written was simply
false, twice over.

The lesson is not "write more careful prose". It is that a residual
recorded only in prose is unverified prose, and the drill that pins it
is also the thing that finds out what it actually says.

## Sweep for a marker across serialized output, not across struct fields

The containment drill searches the JSON-serialized envelope of every
agent-facing read for a marker string, rather than inspecting the types
that build it. A field added later is covered automatically; a drill
that read `packet.Packet`'s definition would have gone on passing while
a new field carried the text.

The same sweep, applied to the projections, is what found that they
carry every payload verbatim — which no reading of the worker-facing
types would ever have surfaced.

## Green locally meant green as root (os-b779b4c7)

Three CI failures on one PR traced to validating against the wrong
thing. `TestProjectionsCarryPayloadsVerbatimIncludingHostileText` passed
here and failed in CI because published projection trees are locked
0555/0444 and `t.TempDir()` cleanup cannot remove them as a non-root
user — this container runs as root, the runner does not, and root
ignores the bits that break the runner.

`project_cli_test.go` had already hit this and carried a comment saying
so. The lesson is not "read more comments": it is that a test touching
anything the tree locks, or anything permission-dependent at all, is
worth running under `setpriv --reuid=65534` before pushing. That is a
ten-second check that would have saved two red builds.

## A receipt is stale the moment another commit lands (os-b779b4c7)

`verify` failed with "receipt mismatch — the committed copy is forged or
stale". It was stale: generated at the first content tip, then a merge
of main and a CI fix landed on top, leaving the receipt describing a
tree two commits behind.

The rule is receipt-after-content with head == content tip, and it means
the LAST content commit, not the first. Any later push — a merge, a
review fix, a lint correction — invalidates it and it must be
regenerated before the branch is green.

## A drill named for a failure path must enter it (os-378e44f3)

The packet-unlink drill was added because a review pointed out the
behavior was unasserted. It called `writePacket` successfully, checked
the file existed, called the success-path cleanup, and passed — while
both error-branch unlinks could be deleted with no effect. It asserted
the wrong thing, under the right name, in direct response to being told
that thing was unasserted.

The check that would have caught it is the same one that catches every
member of this family, and it costs one command: delete the behavior,
re-run the drill. If it still passes, the drill is about something else.
Doing this to the fix is not optional politeness toward the reviewer; it
is the only evidence that the test tests anything.

## A drill that asserts "it refuses" cannot tell you WHERE it refused

Writing the escalation channel (os-f781f0da), the drill for
"a malformed question is validated at the door, before a session opens"
asserted that the CLI refused and that the chain did not grow. Deleting
the door check left it green: the admission boundary refuses the same
payload, and appends nothing either, so both readings satisfied every
assertion the drill made.

The mutation is what surfaced it, and the fix was to assert the
observable that actually differs — the refusal **code**. A door refusal
is `usage`; a boundary refusal carries the rule's own code and, on the
remote path, costs a round-trip the caller never needed to spend.

The general form is worth keeping: when two implementations differ only
in *where* work happens, a drill written in terms of the *outcome*
cannot distinguish them, however carefully it is named. Find the
observable that changes — an exit code, a call count, an artifact that
does or does not appear — or accept that the drill covers less than its
name claims.

## Diff the two things you actually mean to compare

Proving the coverage gate's output was unchanged (os-cafba959) took
three attempts, and the first two compared main against itself. Running
`make check` in one directory, then `cd`-ing and running it again, does
not switch trees when the second `cd` silently fails or when the shell's
working directory has already moved — both runs then measure the same
tree and the diff is empty for the wrong reason.

An empty diff is the result you WANT when proving byte-identity, which
is exactly why it deserves suspicion: the happy answer and the broken
experiment look identical. The fix was `make -C <tree>` for each side,
naming both trees explicitly so the comparison cannot quietly collapse
into one, plus normalizing the paths `make` prints so the real
difference stands out from the directory noise.

The general form: when a passing result and a botched setup produce the
same output, add an assertion that fails when the setup is wrong. Here
that was checking the two runs reported different tree paths before
trusting that their contents matched.
## The guard fired; the drill was looking somewhere else

Closing the key-file TOCTOU (os-9a89245c), the interleaving drill failed
with what looked like the fix not working: the error was the loop's own
`ErrKeyRotated`, not the signing site's refusal. The obvious reading was
that `--as` had not taken effect.

It had. The rotated act refused at the seam exactly as designed, the
loop then ACTED on that refusal, and its next `checkIdentity` produced
the error the drill happened to inspect. The guard's refusal was real
and simply not the last thing that happened.

The fix was to capture the result of the very call the rotation landed
inside — the wrapper already had it in hand — rather than asserting on
the iteration's final error. What made this worth recording is that the
failure mode is inverted from the usual one: not a drill passing for the
wrong reason, but a drill FAILING while the code was right, which is the
kind that tempts you to "fix" working code. The tell was that the error
message described a real thing correctly; nothing in it was false, it
just answered a different question than the one being asked.

## A catalog sweep is only as wide as the thing it enumerates

Criterion 5 of os-9a89245c asked for a sweep "over the act catalog
rather than a hand-listed subset, so an act added later without `--as`
fails". Two versions shipped that wording while not providing it:

1. The first swept the acts one `Step` happened to reach — four of
   seven — and `continue`d past the rest.
2. The second swept the acts the TEST FIXTURE's manifest declared. That
   looked like a catalog sweep and read like one, but the fixture
   declares five acts while the shipped manifest declares seven.

Both passed. Both would have let a new act ship without `--as`.

The tell was that the mutation did nothing: deleting `--as` for
`budget release` left the drill green. A mutation that changes nothing
means the drill never reached the code, and that is worth more than the
drill's own green.

What the property actually belongs to decides what to enumerate: `--as`
is attached in `actGated`, not by any manifest, so the sweep is over
`loopverb.Names()` with a manifest constructed to declare all of them —
never over whatever subset a fixture happens to carry.

## A drill that reads the report is not a drill that reads the chain (os-8a5f14bb)

The maintenance loop's first reap never landed. `claim.reaped` is a
claim-scoped event, so the fence rule refuses one whose payload does
not cite the active window — and the first `ReapPacket` returned the
bare four-part packet with no `{"fence": "<position>"}` beside it.

What caught it was the drill that **folded the chain back** and looked
for the `claim.reaped` record. A drill asserting the report's `reaped`
list would have been just as green either way, because the pass
genuinely decided to reap; it was the append that the boundary
refused, and the refusal landed in a `refusals` list nobody was
reading.

The general form: **when a verb's whole job is to change the ledger,
the assertion belongs in the ledger.** A report is the actor's account
of what it meant to do.

## A mutation the boundary refuses is not a mutation (os-8a5f14bb)

Testing "a lint finding must never raise an escalation", the mutation
was to make the filing path raise one instead. The drill stayed green —
which looked like a weak assertion. It was not: the fixture's subjects
were `in_progress` and `done`, and `escalation.raised` admits only from
`ready` or `review`, so the mutated code path refused at the boundary
and fell through to the original filing. **The mutation never
executed.**

The tell is the one already recorded on #202: a mutation that changes
nothing means the drill never reached the code. Here it meant something
narrower and worth separating out — the mutation never reached the code
*either*, so the experiment was void rather than the drill weak.

What settled it was proving the assertion LIVE directly: plant an
`escalation.raised` signed by the maintenance key into the fixture and
watch the drill fail. That also improved the assertion, which had been
scanning for the bare verb and would have fired on another lane's
escalation; it now names the actor.

**When a mutation cannot be made to execute, prove the assertion fires
by planting the condition it forbids.**

## A snapshot is one record behind its own checkpoint, by construction (os-8a5f14bb)

The checkpoint round-trip drill compared the fetched snapshot against a
rebuild from the chain's tip, and two projections differed. Nothing was
wrong: the snapshot materializes position N, the checkpoint event is
appended *after* it at N, so the tip is N+1 — and the report projection
counts that very checkpoint.

This is the failure mode that tempts you to "fix" working code, the
inverse of the usual one (#202 recorded its sibling: a drill failing
because the fix worked). The fix was to compare against the prefix the
snapshot NAMES. **A materialization is evidence about a position, and
the position it names is the only one it can be checked against.**

## Running as root masks the permission failures CI will find (os-8a5f14bb)

`make check` was green locally and red in CI on a `TempDir RemoveAll
cleanup: permission denied`. `project.Rebuild` locks its published
build trees (0555 directories) by design, so Go's `t.TempDir()` cleanup
cannot remove them — unless the process is root, which ignores the
permission bits entirely.

The card's own acceptance criterion said "the suites pass
unprivileged". Coverage was measured cold, as asked; the unprivileged
run was skipped because everything was already green, which is exactly
when it is worth the least and costs the most to skip.

Two things follow. **The unprivileged run is not a formality when the
container is root** — it is the only run that sees this class of
failure at all. And **the fix already existed in the tree**:
`cmd/seed/project_cli_test.go` carries a `t.Cleanup` that walks the
output and chmods directories writable, with a comment naming "an
unprivileged runner". A new drill against a locked surface needs the
same cleanup, registered AFTER the `t.TempDir()` it unlocks so LIFO
ordering runs it first.

## A mutation you cannot revert is a mutation you will ship (os-8a5f14bb)

A mutation test left a deliberate defect in the shipped tree, and review
caught it rather than any drill. The revert was:

```sh
git checkout cmd/seed/maintain.go 2>/dev/null || cp /tmp/m2.bak internal/maintain/maintain.go
```

`cmd/seed/maintain.go` was a NEW file, so `git checkout` had nothing to
restore, failed into `/dev/null`, and the fallback restored **a
different file**. The tree kept the mutation. Everything afterwards was
green, and green is exactly what the mutation predicted: it made the
filing path try an escalation first, which that fixture always refused,
so the loop fell through to the correct behavior.

Three things follow.

**Revert by construction, not by command.** Snapshot the file you are
about to mutate and restore that same path — `cp X X.bak` then `cp
X.bak X` — never a `git` verb whose behavior depends on whether the
file is tracked, and never with the failure silenced.

**Verify the revert, not the tests.** A passing suite cannot tell "the
mutation is gone" from "the mutation is inert here". The check that
works is a diff or a grep for the mutation's own text.

**An inert mutation is a void experiment, not a passing one.** This
mutation could never land: `escalation.raised` requires a packet, so
the payload was refused on shape before any rule about escalations ran.
Re-running it with a WELL-FORMED payload against a subject in `review`
made it land, and the guard fired immediately. The rule already
recorded on #202 — a mutation that changes nothing means the drill
never reached the code — has a second half: it may equally mean the
mutation never reached the code, and the two are told apart only by
making the mutation land.
## Two workers stepped in sequence do not race (os-6a08b166)

The fleet fixture's middle arm needed a genuinely refused `claim take`.
The obvious construction — step worker A, then step worker B — produces
no refusal at all: B polls after A has claimed, finds nothing offered,
and goes idle at the POLL with an empty `Cause`. The drill would have
counted an empty poll as refusal convergence.

The fix is the shape #202 established for a different race: plant the
rival **from inside the seam**, on the wrapper's first sight of `claim
take`, so the lane is in the window by construction. A race reproduced
by sleeping passes green on a slower runner; a race reproduced by
ordering the calls is not a race at all.

The general form: **when a drill needs contention, ask what the second
actor SEES, not what it does.** Sequential actors observe sequential
worlds and never contend.

## Capability absence is not disjointness (os-6a08b166)

The small-team drill asserted that the implementing key cannot render
the verdict, and it passed — with `out_of_grant`, because that key held
`claim` and not `verdict`. It was proving that a key without the grant
cannot render, which is true of every key and says nothing about
independence.

The charter's claim is that disjointness holds **when one person runs
everything**, and a principal running everything can grant themselves
everything. So the drill now grants the implementing actor `verdict`
first, and the refusal becomes `not_independent`.

`admit.go` says this in its own words — "distinct from `out_of_grant`,
which is capability absence" — and the drill still walked into it. When
a refusal has a near neighbour, assert the CODE, and pick the fixture
that makes the neighbour impossible.

## A matrix that counts its rows is not a matrix that covers its cases

The narrowness drill for `budget_exhausted` had to prove that thirteen
other refusals in one admission rule still returned the old code. The
first version had twelve rows, every row passed, and it reached **eight**
of the thirteen. The five it never touched were the ones needing a chain
shaped a particular way rather than just a bad payload: a subject whose
budget class the table does not know, a reservation staged raw so it
never passed the authoring boundary, one already closed, one closed by a
stranger, and a spend with nothing reserved. Twelve rows all landed on
the eight easy sites, several of them twice.

Then three of the twelve turned out to be testing the wrong site
outright. Two "malformed payload" rows omitted the `reservation` field
to make the payload bad; a missing field decodes cleanly to the empty
string, so both refused at the chain-position site, not the strict-decode
one. A "non-numeric actuals" row cited position 0, which is refused as
no-such-reservation long before any actuals field is read.

Every one of those rows was green. Green meant only "some refusal in
this rule happened", which is the assertion a row is least useful for.

**Two cheap habits fix the whole class.** First, each row states the
site it must reach and the drill asserts the refusal's own message
against it — the row stops being able to pass by landing somewhere
convenient. Second, the drill reads the rule's refusal sites **out of
the source** and fails when one has no row, so the count in its name is
derived rather than typed, and a fourteenth site added later cannot ship
with nothing asserting the code it returns.

That second habit is the same one this card applied to
`next/spec/envelope.md`: the exit-code drill used to hold a hand-copied
map, so adding a code took three edits with nothing forcing the last
two. Parsing both sides and comparing them in both directions found a
real drift on its first run — five constants with no rows, and one
constant whose wire name did not match the row it had. **A drill that
must be updated by hand to catch a regression cannot catch that
regression**, and it reads exactly like one that can.

## A frontier line is a claim, and it can disagree with its own document

`next/docs/progress.md` said Phase 9's next action was the exit record,
"every numbered item in the phase has an implementation". Two other
paragraphs **in the same file** said item 5(b) was PARTIALLY MET, and
the build plan said it in its own item text. So the frontier line and
the body of the document disagreed, and the frontier is the line an
agent reads to decide what to do next.

The disagreement survived because nothing forced the two to move
together. A phase's item list lives in the build plan, its per-item
status lives in prose scattered through progress.md, and the "next
action" line is a third copy written by whoever last touched it. Three
copies of one claim, none of them derived.

It was caught by trying to *write* the exit record, which meant
enumerating the items — the first thing in weeks that read the list as
a list rather than as background. One item too late to be free: two
cards had already been planned and merged against a frontier that was
wrong about what remained.

**The habit that would have caught it earlier:** when a document states
a status per item AND a summary over those items, treat the summary as
derived and re-derive it, rather than reading it. The cheap version is
to enumerate before believing — five seconds of listing 1, 2, 3, 4, 5(a),
5(b), 5(c) against their write-ups would have shown the gap any week.

This is the same shape as the two drill lessons already recorded here,
one level up from code: a hand-maintained summary cannot notice what it
was never told, and it reads exactly like one that can.
## The count you were given is the count someone found, not the count there is

The card for the lane-grants gap said two capabilities were ungranted.
Review found a third. The drill, once it read the capability table's
own source instead of any of those lists, found **six ungranted verbs**
across those three capabilities: the three `run.*` verbs need
`supervise` too, and nobody — the card, the reviewer, the plan, me —
had listed them.

Each list was honest. Each was what one careful reader had noticed.
None was derived, so none could be complete, and the plan's acceptance
criterion inherited the reviewer's three the way the first draft had
inherited the card's two.

The pattern is now recorded here three times in three weeks, at three
altitudes: a hand-copied exit-code map (os-d03bde01), a hand-typed
"all thirteen" refusal count (same card), and now a hand-listed
capability gap. **When a criterion says "all N", N is a claim the
drill must derive, never a number the drill is told.** The cheap
version, every time, was the same: find the source the list is a copy
of, and read it.

## A version gate written as equality is a gate that closes on the next version

The lifecycle fold activated "at seed/1" as `e.V != version.Seed1 →
skip`. It read as an activation boundary and behaved as one for two
phases, because there was only one version past genesis. The moment
`seed/2` existed, the same line silently dropped every claim, budget
and offer record at `seed/2` positions, and the first drill to run
under the new version refused two rules earlier than the one under
test, with a message ("no open valid reservation stands") that named
the symptom and not the cause.

The fix is small (one named list in `internal/version`, consulted by
both the fold and the keyring), and the lesson is the pattern: **an
"activates at N" rule is a rule about N and every later version, and
`== N` says the opposite**. Grep for `== version.` and `!= version.`
before adding a version, not after the first drill fails. The
exception is deliberate and rare: a gate that must NOT carry forward
(a grandfathering boundary for one version's junk) is written as
equality on purpose and says so in its comment.

The companion rule from the same card: the list is a list, never an
ordering. `Applies("seed/9")` must stay false on a build that has not
registered `seed/9`, or a chain upgraded past what the build
understands would fold under rules it never implemented.

## `go build ./...` writes nothing; `go build ./cmd/seed` writes a landmine

Two build spellings that read alike behave oppositely. With a
multi-package pattern (`./...`) `go build` compiles and DISCARDS every
executable, which is why `make check` never dirties the tree; with a
single main package (`./cmd/seed`) it writes `seed` into the current
directory, which for a run from `next/` is `next/seed`, a 12 MB file
that was committed twice by builds nobody meant to commit and then had
to be reverted by hand before every commit. The repository-level fix is
one ignore rule (os-a487b3b5); the habit is to build into `next/bin/`
(`go build -o bin/seed ./cmd/seed`), the path the build plan already
names, or to use `go run ./cmd/seed`.

## A plan merged over open review threads still owes them an answer

Plan #219 merged with three reviewer findings unresolved (a protocol
bump argument, a fixture-scope gap, a residual the plan claimed closed
that it only narrowed). A merged plan is the authority for its task,
but merging did not make the findings wrong: one of them (the residual)
changed what the replaced drill had to assert, and one (the fixtures)
would have turned the suite red on the first run. Read the plan PR's
threads before implementing, answer each in the task PR's decisions,
and let the implementation carry the corrections the plan's text
lacks. Derive counts from the tree while you are at it: the card said
two authority sites and the tree had three.

## Run the guard suites before the unit suites feel done

The tree carries guards that fire only under `go test ./...`: the
envelope table guard refuses a wire code the spec's tables do not
list, and the gitref fixture guard refuses a test that inits a git
repository without hardening it against detached auto-gc. Both went
red on item 2's first full run after every package under change was
green in isolation, and both were one-line fixes once seen (the
refinement row; `hardenGitRepo` after `git init`, with the helper
copied into the package the way `internal/verdict` does). The lesson
is cheap: **run the whole suite in the background as soon as the new
packages compile**, not after the specs are written, so a guard's
finding lands while the code it judges is still open.

## A fixture that drives subjects inside closures must hand back a stand

A fixture returning a context value plus a `step` closure works while
the test reassigns the context from every call. It stops working the
moment the fixture drives several subjects from inside its own
closures, because the test's copy is whatever the fixture returned
last, and the failure ("the cited contract is not in the fold") names
the symptom two subjects late. Return a struct the closures write and
the assertions read, and there is no copy to go stale.

- **An effective-value assertion can pass for the wrong reason.**
  `git config --get` reads every scope, so a drill asserting a
  repository-local property against it is satisfied by a process-wide
  global the test harness installed for other reasons (os-711b3028:
  the client's git dir drill passed before the client wrote anything).
  Assert the scope you mean (`--local`), and for a write that must
  happen on every open, stage the state an older build would leave and
  prove the second open changes nothing.
- The receipt binds the approved plan's bytes at the MERGE-BASE, not at
  the commit the anchor names. A drill on a planned tier must cite a
  file the fixture repository actually holds at the base of the
  submission range; citing `plans/<subject>.md` against a fixture with
  no such tree fails `chain_invalid` at render, two steps after the
  approval was staged.
- A fixture on the remote posture cannot reach a ledger-only verb.
  Stage the fact through the library under the fixture's background
  rule (signed by a key the authorization check accepts) and record
  the gap; widening the verb in a task PR is scope the plan did not
  grant.
- Above the trivial tier every render pays for the seal, so a drill
  that wants reconcile to reproduce a receipt without a key uses the
  trivial tier: the one tier whose subjects are unsealed and whose
  executable gated spec still reaches L3.
- A test fixture's resolver must know every key that will sign in the
  drill: the store's resolver answers genesis keys only, and a lane key
  enrolled on the chain still fails "actor fingerprint not in the
  keyring" at append. Build the loose resolver over the drill's own
  key list rather than reusing a fixture that closes over another
  drill's.
- When a probe must be signed on a subject the caller did not name (a
  fact on a derived subject), map the override by verb beside the
  catalog rather than adding a field to the catalog's anonymous
  struct: every positional literal in the table breaks otherwise.
- Adding a registered projection moves every count pin of the registry
  (`len(results)`, `len(list)`); grep for the old number across the
  packages before running the suite.

- A tolerant lifecycle fold reports windows, verdicts and proposals
  whoever signed them. Every curation check that reads a folded fact
  (a window's holder, the latest verdict, an earlier proposal) must
  re-judge it at its own prefix (`WindowAdmitted`, `FailedAt`,
  `AdmittedProposalBefore`) or raw history launders past the boundary.
  Drill each with an enrolled, grantless key: the store appends
  anything the resolver can verify, and the fold applies the legal
  transition regardless of the signer.
- An "admitted before" check that recurses through full validity
  (which itself asks "admitted before") is exponential in the number
  of duplicates on one subject. The earliest proposal that passes the
  grant and the support is the admitted one, so one forward scan
  settles the duplicate question in linear time.
- A lexical prefix test on an anchored path is not containment:
  require `path.Clean(p) == p` and refuse an absolute path before the
  prefix test, or `<store>/../x` passes the anchor grammar and the
  prefix both.
- When a stacked card widens a function that an earlier card's drills
  call (`HypothesisID(claim)` gaining exceptions), grep every test
  file in the tree for the old arity before running the suite; `go
  vet` reports the first file per package and hides the rest.
- A projection build is input-free: anything that needs a repository
  (ancestry, a file's digest) belongs to the readers that hold one,
  and the projection carries only the record half with a note saying
  so. Trying to verify in the build would have made `project rebuild`
  depend on a checkout it is not given.
- Every stage a fold binds from a raw fact is a stage an attacker with
  raw ledger access can plant. The test for each fold arm is the same:
  raw-push the fact the boundary would refuse and assert the fold
  changed nothing. Sharing one check function between the admission
  rule and the fold (proposals, contests, promotions all do now) is
  what keeps the two from drifting.
- A verifier that reads one thing from a caller and hashes another
  from a store judges two revisions. Read the stored bytes first,
  require the caller's copy to equal them, then judge the stored bytes
  alone.
- A result derived from a view is stale the moment the optimistic
  loop retries against a refreshed one; carry the derivation in the
  act so every attempt recomputes it, even when the payload itself
  holds nothing derived.
- A coverage claim is only as good as its derivation: enumerate the
  registry the rules construct refusals from (`curation.Gates()`) and
  fail on any gate no drill attacks, rather than hand-listing the gates
  a drill covers. The registry and the spec table pinned both ways
  keep the three from drifting apart.
- When a script fixes positions ahead of time (an eval marker citing
  where the hypothesis WILL land), compute the cited position from the
  count before the records land and assert equality with where the
  fact actually landed, rather than subtracting a constant from the
  last record's position: the constant drifts the first time the
  fixture adds a record.
- A validity check that recurses into the validity of every earlier
  fact of the same kind is exponential in the count of those facts,
  even when each level is a linear scan. Split it into the arms (never
  recursive) and the ordering rule, and find "the latest admitted one
  before here" in one forward pass that judges each candidate through
  the arms and the ordering against the latest it has admitted so far.
  The refold drill from the previous item caught the retirement
  ordering's first cut within a second.
- Key a fold's per-path maps by the path, not the anchor, when the
  readers' questions are about the path; the first drill that asks
  `fold.Lessons[path]` will say so.
- Two parallel shell calls share one working directory: a `cd` in one
  moves the other. Use absolute paths in every call that runs beside
  another, or a merge intended for one worktree lands in another.
- Before designing a human's act, check what the human's key can and
  cannot hold. The sealed-checks recipient set excludes operator
  standing, so a "human = verdict + operator" key can never compute a
  receipt on a sealed subject; the deferral had to carry the receipt.
  The drills that first went red were the existing critical-tier ones,
  which is what they are for.
- A shadowed `rep :=` inside an `if` is invisible until an act cites
  the wrong payload two steps later; when a derivation "re-owes"
  something already performed, check which payload was appended
  before suspecting the derivation.
- A recorded corpus that replays against the boundary is only as
  deterministic as its inputs: derive every key from a fixed seed, keep
  every instant out of the recorded frame, and let the scenario fix
  positions and verbs, or two recordings on two machines differ in a
  fingerprint and the byte-for-byte drill is unwritable.
- A class that judges a recorded point against the current
  configuration must ask whether the point was ever inside it: a
  refused attempt at an undeclared act is a decision point too, and a
  class that holds it to the manifest fails the corpus on the day it
  is recorded. Hold admitted points to the configuration; hold refused
  ones to their frame.
- Never run a mutation script that restores with a directory-wide
  `git checkout -- <dir>` while the tree carries uncommitted edits
  elsewhere in that directory: the restore reverts them too. Restore
  the mutated files by name, and commit the docs before the mutations
  run.
- When a stacked or parallel card bumps the same projection version,
  resolve the merge by taking the later number and saying so in the
  merge commit; two "12"s with different meanings is a silent
  republish under a wrong id.

- Judge a slice bound on the value the input supplies, never on a sum
  derived from it: `position+1 > len` overflows at the largest value
  the parser admits and slips past the guard. Compare the position
  and add afterwards.
- A field documented as an anchor is validated as one at the shape
  rule, not only where a flag's help text says so: the CLI is one
  path to the boundary, and the raw seam is another.
- A boundary that guards one ref by content but exempts every other ref
  by identity has only half a ceiling. The Phase 2 hook refused invalid
  ledger pushes but passed every code ref untouched, so a compromised
  credential could push the default branch, another actor's branch, a
  tag, or the check pipeline. A release-gate drill that exempted
  non-ledger refs would have reported three §I.2 clauses green by never
  asking. The code-ref half derives its authorization from the same
  ledger the hook already guards (standing, claim holders) plus the
  transport's identity assertion — one derivation, two callers.
- "The protected surface is changed only by the governance root" is not
  "by an operator". The maintenance lane holds `operator` and is an
  agent key, so gating the surface on operator standing lets a
  compromised maintenance key rewrite the gates that judge it. Gate the
  protected surface on `keyring.IsActiveRoot`; operator standing moves
  the branch, root standing moves the surface (Copilot review, #247).
- A revoked key still "holds" its claim in the tolerant lifecycle fold
  until the claim is reaped, so a code-ref rule that authorizes by claim
  holder alone lets a revoked key keep pushing its branch. Gate the
  contract-branch push on active standing (`HasAnyCapability(claim|
  operator)`) too: the branch closes with the standing, before the
  reap.
- A rewrite drill must push a DESCENDANT of the admitted tip, not an
  orphan. An orphan force-push bounces at the commit-graph
  fast-forward check before the hook's content rules run, so it proves
  nothing about append-only-ness; committing on the current tip and
  rewriting the last record's payload (re-signed, same version and
  prev) verifies from genesis yet diverges, which is exactly what the
  record-level prefix check catches and a fast-forward check misses.
- A fold that re-judges a fact "through the same checks" must include
  the grant: re-running the record gates alone binds a well-formed
  fact pushed raw under the wrong key. Read the keyring at the
  position, as the curation fold does, and let a raw-push drill under
  a claim key say so.
- An affordance probe must carry every citation the rule demands at
  that position; a probe that omits one the record has just made
  mandatory (a passed repair) hides the verb at the moment it is owed.
  Derive the probe's citations from the same helper the rule reads.
- Invoking the v1 engine from a fixture needs the whole instantiation
  surface it reads (`.seed/`, `scripts/seed`, `scripts/seed-harness`,
  `scripts/harness/`), and a scrubbed runner needs the engine vendored
  into the fixture's own lock, because the shim's cache lookup goes
  through HOME.
- `workflow run --mock` refuses when a declared required input is not
  supplied; bind placeholders per declared input wherever the mock is
  the check.
- Check the boundary before a side effect the boundary might refuse:
  a branch written ahead of an out-of-grant append is a branch left
  behind a refusal.
- `loop.go`'s `gitOut` treats empty output as an error; git verbs that
  succeed silently (`add`, `commit --quiet`, `worktree remove`) need a
  runner that reads the exit status alone.

- In one test process "who is pushing" is an environment fact. A
  fixture that models a forge's sole-writer rule with an environment
  variable must set it around the service's pushes and clear it around
  the actor's, or the actor's raw push and the service's proposal look
  like the same writer and the drill proves nothing about either.
- A hook that wraps refusals with `%v` prints the same text as one
  that wraps with `%w`, and only the second lets a service built on the
  same judgment answer with the boundary's typed code. Wrap with `%w`
  where a second caller might ever need `errors.As`.
- Forges protect branches and tags, not custom ref namespaces: a
  ledger that a forge must guard lives under `refs/heads/`. The ref
  name is one parameter of every seam, so this is a declaration, not a
  code path — but only if nobody hard-codes the default a second time.
- A client that lands a write through a third party must not persist
  a commit it does not hold: the monotonic-head rule compares by
  ancestry in the client's own git dir, and an object that is not
  there reads as a regression on the next fetch.
- Copying a mapping into a second binary is how two postures come to
  disagree on a code. When the second caller appears, move the mapping
  to a package both can import before writing the second caller.
- A proof that two paths produce identical output must not stamp the
  output with which path produced it. Put the provenance beside the
  output (a basis file in the root), or the proof is unwritable.
- A chain hash that excludes the signature is what makes "trust the
  signer set" a real trade-off: the drill that shows a corrupted prefix
  signature slipping past the trusting reader is the drill worth
  having, not the one that shows the two readers agreeing.
- A generated representative history is a better adversary than any
  hand-written fixture: pushing one through the hook found a context
  the hook built by hand and the CLI built by constructor, disagreeing
  on the one field the budget rule reads.
- Under one optimistic ref, `n` racing writers cost about `n/2`
  attempts each. Budget the ratio to that shape and say why; a flat
  ceiling would fail on the day someone raises the writer count.
- Before asserting "no manifest grants X" in an audit, derive the set
  and print it: the shipped maintenance lane holds operator by design,
  and an audit written from the charter's sentence rather than the
  tree fails on its first run. Hold the derived set to a named list.
- Go's flag package stops at the first positional: a verb taking a
  file then flags must parse twice, or every flag after the file lands
  in NArg as a usage error the drill reads as "refused".
- A test helper's third return is not always what its name suggests:
  `writeKeys`'s `pub` is a second operator's key, not the signer's.
  Derive a fingerprint from the key you signed with.
- Replaying a thousand records through a from-scratch admission
  context is quadratic in JSON decoding, not in signatures: profile
  before optimizing, and cut passes (derive grants from the source
  before replay) rather than the boundary. A store whose append
  rescans every segment is quadratic too; a one-pass batch append that
  checks exactly what the single append checks is the fix, not a raw
  segment write.
- A verb that overrules a standing fail verdict (`merge.overridden`)
  cannot stand in for a missing pass: when the predecessor recorded no
  verdict, the honest record is a pass over an artifact that says no
  receipt was recorded, with the disposition noted, never an override
  of a failure nobody rendered.
- Card evidence blocks and run-log entries share one clock in v1, so
  matching by kind and instant within seconds works; blocks the
  predecessor later pruned simply have no match, and the entry itself
  is then the artifact. Do not loosen the match to make the count look
  better.
- A table's self-validation is a design tool: when a new behavior
  needs a state's exits to grow, ask whether it is an exit at all. A
  racing claim is not, so it became a claim-scoped fact and the table
  stayed the contract it was.
- 2026-09-03 (no card; from Can Bölük, "The Harness Playbook", Stencil,
  2026-09-02): the machine-protocol surface (build plan Phase 13 item
  6) should stay tiny. Every permanent tool taxes every turn and a
  changing tool roster invalidates provider caches, so expose one
  passthrough over the CLI plus at most a situation read, with the
  affordance envelope as the schema the model inspects, never one
  protocol tool per verb. The CLI is already the complete interface
  (charter II.15); a second surface adds no second semantics.
- 2026-09-03 (no card; same source): when planning the container, cloud
  session and remote worker executor adapters (Phase 13 item 2), use the
  playbook's design questions as an adapter checklist: a kill boundary
  the adapter can terminate rather than cooperative cancellation (the
  force-kill fallback in charter II.9 assumes one exists), every stream
  leaving the executor bounded before it reaches host memory or the
  ledger, and workspace isolation as "child gets a view, parent receives
  a diff", which is how packets already anchor artifacts. The playbook
  trusts the harness host and distrusts only the sandbox; Seed trusts
  neither, so its host-side policy is defense in depth the ledger never
  relies on. Keep that distinction out of the handbook.
- 2026-09-03 (no card; same source): nothing under `next/spec` or the
  build plan carries a formal model of admission. Ordering, fences, the
  monotonic-head rule and halt are proven by drills (claim race storms,
  rollback, raw-git writers), which are examples rather than proofs. A
  TLA+ model of admission over concurrent proposers is a backlog item
  (build plan §3, non-conformance-blocking, filed after Phase 13), not
  a charter change.
- 2026-09-03 (no card; from github.com/exoharness/exo, read at its
  2026-09 head): the strongest external example of the failure mode
  the admission boundary exists for. Exo's docs call its event log
  immutable and "the only part the agent cannot modify"; the tree has
  one JSON file per event under `.exo/`, no hash chain, no signatures,
  no append-only enforcement, and the startup script mounts `.exo/`
  read-write into the agent's own sandbox. Cost is self-reported with
  no caps, the self-rebuild gate is a compile with no tests, the
  scheduler's claim is documented as racy behind a PID lockfile, and
  the README's cross-clone lineage does not exist in code. Its
  substrate-versus-executor split is Seed's ledger-versus-executor
  split one layer down, so the divergence is exactly principle 3: a
  ledger the writer can reach makes the filesystem the authority.
  Cite it when a design review asks why admission is not a convention.
- 2026-09-03 (no card; same source): three mechanisms for the Phase 13
  item 2 adapters (container, cloud session, remote worker). (1)
  Snapshots as opaque payloads tagged with an open format string, with
  each backend declaring the formats it can consume and the harness
  refusing a restore before dispatch when the target does not; restore
  on a different provider falls out of that for free, and it fits the
  artifact store. (2) One backend-agnostic contract test suite that
  every substrate implementation runs, behind a feature, rather than
  per-adapter tests. (3) The kill boundary as Exo's firecracker guest
  implements it: every command runs as a process-group leader and is
  killed as a group on timeout, the in-VM supervisor refuses any vsock
  peer that is not the host, and wire frames are size-capped and
  fuzzed. Exo's own agent-facing
  shell tool has no timeout and no output cap, so the checklist is
  easy to skip. Two smaller notes for later specs: recurring schedules
  should carry a missed-fire policy (skip, once, all with a cap) and
  fire on the grid rather than drift, and any adapter settling actuals
  into `budget.settle` needs the additive-versus-inclusive cached-token
  distinction and token-boundary model lookup, or cache-heavy runs
  misprice by up to ten times.

- claim.reaped admission never gated on reap corroboration — that
  discipline (InterruptValid/WedgeDeclared) is the maintenance loop's
  (the Corroborate closure + Reapable), not an admission rule. A card
  that assumed an "admission reap gate" would wire its change to the
  wrong place; the reap is already admissible from the reaping lane, and
  what a new corroboration changes is whether the LOOP chooses to reap.
- A revocation is the one reap corroboration the ledger itself supplies:
  a revoked holder provably cannot exit its window, so its claim reaps
  in every classification state, no_data included — the single exception
  to "no_data is never reaped", and only because the chain, not the
  lossy observation channel, corroborates it. Judge the revocation at
  its own position (the InterruptValid posture) so a raw or unprivileged
  one, or a suspension whose standing can return, corroborates nothing.
- Replaying a thousand records through a from-scratch admission
  context is quadratic in JSON decoding, not in signatures: profile
  before optimizing, and cut passes (derive grants from the source
  before replay) rather than the boundary. A store whose append
  rescans every segment is quadratic too; a one-pass batch append that
  checks exactly what the single append checks is the fix, not a raw
  segment write.
- A verb that overrules a standing fail verdict (`merge.overridden`)
  cannot stand in for a missing pass: when the predecessor recorded no
  verdict, the honest record is a pass over an artifact that says no
  receipt was recorded, with the disposition noted, never an override
  of a failure nobody rendered.
- Card evidence blocks and run-log entries share one clock in v1, so
  matching by kind and instant within seconds works; blocks the
  predecessor later pruned simply have no match, and the entry itself
  is then the artifact. Do not loosen the match to make the count look
  better.
- CI wall clock for check-validate was dominated by four full runs of the
  next/ suite per pull-request run (the check job, the plan's `make check`
  under receipt verify, and the two core-gate instantiations in
  flavor-test), each with a cold Go cache and packages serialized under
  `-p 1`. Measure a job's steps before changing anything: the fix was
  caches restored from main, package-level parallelism on the first
  coverage reading, the two core-gate checks run concurrently, the four
  backend fakes run concurrently, a tmpfs TMPDIR (the drills fsync and
  spawn git, so sys time halves), and one live run per pull request. No
  gate, ceiling, or command changed.

- When two surfaces must expose the same verbs, draw both from one
  table and hold the table to the dispatchers' own usage text in a
  drill; a hand-kept second list drifts the day someone adds a verb.
- bufio.ScanLines strips a trailing carriage return and TrimSpace
  strips another: a "refuse CRLF" rule needs a split function and a
  trim that keep the CR, or the parser never sees what it must refuse.
## Docs generation and simulation mode (os-16e55c11, Phase 12 item 6)

- **Go constants are not runtime-enumerable.** To generate a doc from a
  package's constants (the exit-code table) without a hand-kept duplicate,
  parse the source with `go/ast` at generation time. `docs check` runs in
  the repo, so `<root>/next/internal/envelope/envelope.go` is available;
  a planted constant change then fails the drift check.
- **The loop's per-iteration act is not a function of its trajectory
  frame.** Identical frames recorded different acts (implementer pos44 vs
  pos53), because the loop resolves the choice from internal state the
  frame does not carry. A "decider" over the frame must be PARTIAL — decide
  where the frame determines the act, abstain otherwise — or it false-
  positives on legitimately-identical frames. This is the same reason
  #239 recorded frames rather than deciders.
- **`claim take` is remote-only** (an exclusive, online-only verb), so any
  end-to-end run of the loop — including a "local/cooperative" simulation —
  needs a bare git remote, not just a local ledger. Admission stamps each
  event's ts with the real wall clock, so a simulated/accelerated clock can
  only feed the reporting surfaces (`--now`/`--as-of`), never the event ts;
  an offer's expiry must sit past the real now.
- **The credential-free in-process seam is `loopVerbs` → `run(args,...)`**
  in `cmd/seed` (package main is not importable), so a non-test package
  that drives the CLI (`internal/simulate`) takes an injected `loop.Verbs`
  and the `cmd/seed` verb supplies `loopVerbs{}`.
- An ingress verb is safest as a fact with a derived subject: hold the
  record's subject to what the payload cites (a contract on the chain
  or `system`) at admission, and every downstream notice can carry
  the subject without re-checking it.
- A projection section that only chains from a new version can carry
  needs no projection version bump: `omitempty` keeps every older
  build byte-identical, and the spec can say so in place of a
  republish. Bump when an unchanged tip would render differently.
- A federation read should never reuse the loop's session opener:
  the opener applies the local declaration (ledger ref, proposer) to
  whatever remote it opens, which is wrong for a foreign ledger. Open
  with the gitref client and the remote's own genesis instead.

## The executor adapters and their budget postures (os-083112ac, Phase 13 item 2)

- **An optional interface keeps the public contract stable.** Adding
  `Described` as an OPTIONAL interface (not a method on `Adapter`) lets
  adapters state their budget posture without breaking external
  implementers; a type assertion + a safe default (`risk-limit`) handles
  the ones that do not implement it.
- **The four adapters share `verifyStarted` + the `Diff` mismatch check
  + `buildWorktree`.** The substrate-specific part is just the
  provisioning (container start, session open, packet handoff); the
  admitted-start gate and the tuple-mismatch refusal are the same for
  all, so factor them.
- **A credential in a declaration is an env-var NAME.** Lint it to an
  identifier shape (`^[A-Za-z_][A-Za-z0-9_]*$`), which a token cannot
  satisfy — the secret stays in the environment, never the tree.
## The GitHub↔Gitea protection impedance (os-ad610334, Phase 13 item 3)

- **Gitea/Forgejo has no unified "ruleset" model.** Protection splits
  across `branch_protections` (keyed by `rule_name`, which IS the branch
  glob — no separate name) and `tag_protections` (keyed by id). A branch
  protection's existence carries deletion + non-fast-forward; the push
  whitelist carries "only the identity may update"; so an adapter maps
  the four Desired rulesets onto those two resources and remembers each
  protection's key (like the GitHub adapter's `ids`) for update/delete.
- **`Unexpressible` is rule-TYPE level, and `differences()` is
  param-exact.** A forge that expresses a rule partially (Forgejo:
  approvals yes, thread-resolution/code-owner no) cannot both apply the
  expressible part and avoid drift on the rest. The honest resolution is
  to mark the WHOLE rule unexpressible (manual) rather than half-apply it
  and read back as falsely compliant — the mutation drill forbids the
  silent drop.
- **A forge field defaulting keeps old declarations valid.**
  `admission.forge` absent = github; the CLI `--forge` default must stay
  the credential-free `snapshot` arm the existing drills rely on.
- **A refinement code emitted as a string literal needs a row in
  `next/spec/envelope.md`.** `merge observe`'s `not_merged` on exit 3
  (invalid_transition) is a refinement; `TestEmittedCodesAppearInTheTable`
  parses the refinements table and rejects any emitted code that is
  neither the exit's canonical code nor a listed refinement. Only codes
  promoted to `Code*` constants render into generated `exit-codes.md`;
  literals live solely in the table. The strict test arrived via the
  item-6 merge, so a pre-merge draft receipt passed and the
  merge-forward surfaced it — regenerate after merging main.
- A published statement across a trust boundary should be a strict
  object with a pinned field list on both sides: the writer refuses
  to add a field without moving the pin, and the reader refuses a
  field it does not know, so opacity survives either side's drift.
- Sign over canonical bytes computed from the struct itself, and
  verify by recomputing; never store the canonical bytes beside the
  signature, or the two can disagree without anyone noticing.


## Tuple ranking as supervisor policy (os-c7554f18, Phase 13 item 7)

- A policy table belongs in one place with a mirror the tests hold to
  it: `ranking.Rules` states the rows in the spec's own words and a
  drill parses `ranking.md` and compares, so a change to the policy in
  either place fails until both move. The behavior drills pin the code
  to the rows; the table drill pins the spec to the code.
- Scheduling policy and admission are different authorities: the
  ranking is read by the offer writer and never by a rule, so the
  bridge for never-qualified workers (admission never judges an
  offer's scope) survives a policy that sends work to the strongest,
  and a first eval's offer stays the one unscoped door.
- A cross-platform CI leg that is an order of magnitude slower is
  measured, not guessed at: the per-package `go test` lines in the job
  log name the package, and a counting shim ahead of `git` on PATH
  names the spawns. The Windows `platform` leg was one serial package
  (`cmd/seed`) spawning seventeen thousand git processes, and the
  largest bucket was three `git config` writes per client
  construction, not the fetch or the push. That bucket stays: the
  hardening is bound to git's own writer on every construction
  (plans/os-711b3028.md D1), and the in-process parser that would have
  skipped the writes was refused in review on #298 for drifting from
  git's resolution (a concatenated value like auto = "0"1 reads as 01
  in git, not 0). Cut spawns where the counts are, shard the serial
  package, and only then reach for a longer timeout.

## The conformance report (os-83bc3d84)

- A conformance claim is only as good as its provenance: keep the
  status beside the criterion's verbatim text and the record that
  judged it, hold the criteria to the charter by parsing the charter,
  and let the doctor say what is still open rather than what is done.

## The seventh race shape (os-5063e8ba)

- A retry classifier that names its shapes is honest about what it
  does not retry, and a refusal that lands beyond the position a
  client appended is the client's tree disagreeing with the remote,
  which a fresh fetch settles either way; keep the refused tree
  before retrying, because a lost temp dir is a lost diagnosis.
- Measure the count of a mitigation (`relinked`) beside the budgets
  it protects, never as a budget: a clean run reads zero, and a
  non-zero reading points at the kept evidence.

## seed ledger audit (os-7599c27d)

- A verb that reads an invariant over a chain verifies the chain
  first and only then reads: a bar over an unverified chain is a bar
  over nothing, and the ordering makes the "empty chain" arm of the
  library's audit unreachable through the verb, which is correct
  rather than a gap.
- `ledger append` is the cooperative posture's self-validating client,
  not a raw seam; a drill that needs a chain admission would refuse
  signs with the root and appends through the library.

## The contention benchmark at target scale (os-a00d3f34)

- When a gate is parameterized by size, scale is a data change: a
  second budget file and a schedule, never a second measurer. The
  drill that pins the profile's size is what keeps the data change
  honest.
- The optimistic append loop costs writers/2 attempts per landed
  record, so its wall time is quadratic in writers; hundreds of
  writers belong on a schedule, and the per-PR gate's 24 is a budget
  on wall clock, not a claim about scale.
- A receipt's diff hash covers git's default diff text, whose index
  lines abbreviate blob ids to a length that grows with the clone's
  object count (seven hex digits below 16384 objects, eight above);
  CI's full clone crossed that count on 2026-09-03, so a receipt
  generated in a partial clone mismatches on verify although plan,
  files and commands agree. Until the engine pin carries the
  --full-index fix (open-seed-engine, receipt: hash the diff with
  full blob ids), fetch every ref and repack before generating a
  receipt, so the local clone abbreviates as CI's does.
- A benchmark's claim is only as strong as its identities: a storm that
  signs every writer with one key measures one actor's contention, not
  N actors'. Enroll a key per writer and have the measurer count the
  distinct actors in the landed chain, so the number in the row is the
  number the chain proves.
- Quote a charter row verbatim in the plan and in the spec that
  claims it. A paraphrase ("without lost updates") stood in for the
  row's clause ("without unrelated writes racing each other's
  admissions pathologically") and would have flipped a conformance row
  on a run that demonstrates the pathology bounded, not absent; the
  task PR's review caught it, and the fix was to state the claim
  against the row's own words and leave the row partial.

## The promotion evidence packet (os-98ce6f8a)

- Evidence at a gate rots quietly: a document that names drills is
  only as good as the tree it names, so cite in one machine-readable
  shape and hold every citation to the tree under the gate that
  already runs. The parser that reads the shape is the same one that
  refuses a stale claim.
- A packet presents; the sentence that selects an option, starts a
  run or flips an entry point is the decision the build plan reserves,
  and the drill checks the reserved criteria carry a question rather
  than a choice.
- A gate over a document holds only what its parser models: the two
  cutover questions lived in prose the parser skipped, so deleting one
  or answering it in place passed the gate that claimed to keep them
  questions. Model every claim the gate is said to protect, one
  physical line each, so an absence is a parse error and not prose
  read past; and confine every cited path before joining it, since
  `filepath.Join` cleans `..` into a read outside the tree the gate
  promises.

## The flywheel drill's skip path (os-222189a3)

- A guard over fixture helpers is only as wide as the helper names it
  knows: os-c4e8b57a's regex held `git(`, `run(` and `gitOut(` to the
  hardening and never saw the flywheel package's `gitIn(`, so one
  repository in the tree stayed unhardened while the guard passed.
  When a package names its helper differently, widen the alternation
  in the same PR; the property is the tree's, not the three packages'.
- Decide a skip before building what it abandons. A precondition that
  needs nothing from the fixture (here, whether the pinned engine is
  in the cache, answered by the source tree the fixture copies) goes
  first: a `t.Skip` after twelve git processes is twelve chances to
  race the harness's cleanup, and on macOS it lost, reporting a
  skipped drill as a failed one on a PR that touched neither.

## A bar that counted a verb the protocol does not emit (os-b86dab4c)

- A fixture that repeats the code's mistake proves nothing, and reads
  like proof. The unreserved-spend bar counted `budget.reserved` and
  its own drill filed `budget.reserved`, so the bar passed for as long
  as it was only ever read through that fixture. When a drill and the
  code it checks can share a typo, hold the code to an authority
  outside both: the protocol's constants, and a drill over the
  boundary's verb catalog.
- Ask which list is the authority before holding anything to it. The
  transition table's `Verbs()` looks like "every protocol verb" and is
  the lifecycle subset; `admit.CatalogVerbs()` is every verb the
  boundary drafts. Writing the guard against the wrong one failed on
  `budget.reserve` and `run.started`, verbs the protocol does define.

## The citation stage of `docs check` (os-5fe43832)

- A document is a projection of the tree, and nothing was holding it to
  the tree. `docs check` proved the generated documents match their
  tables and stopped there, so every hand-written citation was
  unchecked: seven relative links did not resolve, four of them in
  `docs/CONTRIBUTING-AGENTS.md`, which III.Q row 5 names as the
  authority order's own evidence. A gate over generated output says
  nothing about the prose beside it.
- The failure that produced four of the seven is worth naming, because
  it is invisible in review: a link written root-relative
  (`](docs/build-plan.md)`) from inside `docs/` reads correctly to a
  human and resolves nowhere. Targets resolve against the containing
  file's directory, never the repository root, and no reviewer catches
  the difference by eye.
- Mask before you match. A regex in prose
  (`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`) is link-shaped, so a sweep that
  does not blank code spans and fenced blocks reports it and trains the
  reader to ignore the gate. Blank the masked regions with spaces
  rather than deleting them: the offsets keep naming the lines they
  came from, and a finding that cites the wrong line is a finding
  nobody acts on.
- A gate that checks nothing passes. The citation stage returns the
  count it held alongside the findings, and the drill asserts the count
  stays above a floor, because a walk that stopped finding documents
  reports zero findings and reads exactly like a clean tree. This is
  the same shape as the boundary check comparing nothing to nothing
  (os-1c284ba8); assert on the work done, not only on the verdict.
- Reuse the exit, split the code. `broken_citation` shares exit 28 with
  `docs_drift` because both are the declared-versus-observed comparison
  the base code generalizes, but the fix differs: `seed docs generate`
  repairs drift and cannot repair a citation, so a caller branching on
  the code must be able to tell them apart.
- Scope a whole-tree gate to what its own trippers may repair. The
  first cut swept `plans/` too, and the only broken citation left in
  the tree was in a merged plan: a plan file changes only through its
  own single-file plan PR, so no branch carrying the gate could carry
  the fix and no branch carrying the fix could carry the gate. A gate
  whose only remedy breaks another rule is mis-scoped, not strict, and
  the tell is that satisfying it took a rule violation in the same
  change. Ask what a person who trips the gate is allowed to do about
  it before choosing the walk.
- A blind spot in a gate is worse than the gate's absence, because it
  is read as a guarantee. The destination pattern excluded parentheses,
  which markdown permits when balanced, and the effect was not a skipped
  link but a dropped citation: unread, uncounted, and green. Scan a
  delimited region by counting depth rather than excluding the
  delimiter, and be suspicious of a test that asserts the gate
  deliberately does not look at something.
