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
