# decisions.md — Seed implementation decision log

One line per build-plan default exercised or overridden, linking the PR that
did it (docs/next-build-plan.md Phase 0 item 3). Decisions that needed more
than a line get an ADR under the repository's `decisions/` and a pointer
here. Newest last.

- 2026-08-30 — Phase 0 filed as **one card** (os-116ca9ac): items 1–3 are one
  coherent cluster sharing one exit criterion; later phases file one card per
  separable item. (PR #72)
- 2026-08-30 — Module path `github.com/shaunlmason/open-seed/next`, pinned
  `go 1.25.0` for engine-toolchain parity (build-plan §0 "matches the
  existing engine"); no dependencies. (PR #72)
- 2026-08-30 — Exit codes: v1-inherited table (0/2/3/4/5/6/10/64) adopted
  verbatim; new-code allocation rule documented in `next/spec/envelope.md`
  (fixed default: reuse v1 conventions where semantics match). (PR #72)
- 2026-08-30 — Envelope schema identifier `seed-envelope/0`; spec split:
  `protocol.md` (canonical form, events, verbs) + `envelope.md` (envelope,
  exit codes), satisfying both Phase 0 item 2 and the fixed-defaults table
  row naming `next/spec/envelope.md`. (PR #72)
- 2026-08-30 — Genesis `prev` = SHA-256 of zero bytes (`e3b0c442…`); actor
  fingerprint = lowercase hex SHA-256 of the raw 32-byte Ed25519 public key;
  OpenSSH ed25519 keys accepted at key load only. (PR #72)
- 2026-08-30 — Wire encodings fixed for `seed/0` (review finding on #71):
  everything is lowercase hex (chain hash 64 chars, signature 128 chars,
  fingerprint 64 chars); the ledger record is the wrapper
  `{"event": …, "sig": …}` with canonicalization applying to the inner
  event only; OpenSSH `SHA256:<base64>` display form excluded from the
  wire. (PR #72)
- 2026-08-30 — Coverage gate (≥90% on `next/internal/...`) enforced from
  Phase 0, not Phase 1: enabling it early is strictly harder, never weaker;
  implemented via `-coverpkg=./internal/...` so CLI-level tests count toward
  internal coverage. (PR #72)
- 2026-08-30 — Go comments in `next/**` follow the engine's no-em-dash rule
  (spin-out parity: the code migrates to the engine's conventions);
  markdown under `next/` keeps the template repo's prose style. (PR #72)
- 2026-08-30 — `flavors/core-Makefile` refreshed alongside the `Makefile`
  edit: the validator enforces byte-identity (its refusal names the fix), so
  the mirror is a mechanical fan-out of the named integration point, not a
  second v1 surface. (PR #72)
- 2026-08-30 — Version semantics refined (review finding on #75): every
  event carries the version *active at its position*; `system.protocol.upgraded`
  is the last event of the old version; verification is against a declared
  supported-versions set, so valid older prefixes replay after upgrades;
  exit 10 covers discipline violations and unsupported versions only. (PR #72)
- 2026-08-30 — `check-next` output made byte-stable (CI finding on #72):
  tool output is captured and shown only on failure, because the
  flavor-test core-gate-independence check diffs `make check` output across
  runs and go's test timings and toolchain-download notices vary. (PR #72)
- 2026-08-30 — OpenSSH key fixtures are generated at test runtime
  (deterministic seed, `t.TempDir()`), never committed: forge push
  protection refuses private-key-shaped bytes even when synthetic, and the
  loaders only care about the wire format, which `ssh.MarshalPrivateKey`
  produces. Standing rule for every later fixture (verifier keyrings,
  sealed-check keys). (PR for os-aa146827)
- 2026-08-30 — Strict wire parsing (review findings on #76): records refuse
  unknown fields, duplicate keys at any level, and trailing data; hex
  encodings refuse uppercase rather than normalizing. Spec amended in the
  same PR (a normative addition beyond the plan's file scope, made under
  the review-fix allowance and noted here). (PR #76)
- 2026-08-30 — First module dependencies for 1.1: `github.com/gowebpki/jcs`
  v1.0.1 (RFC 8785 canonicalization; writing correct JCS by hand is
  subtle-risk with no upside) and `golang.org/x/crypto` (OpenSSH ed25519 key
  parsing); both boring, both pinned by go.sum. Ed25519/SHA-256 themselves
  stay stdlib. (PR for os-aa146827)
- 2026-08-30 — Ledger signature checks take a `Resolver` seam
  (fingerprint to public key): the keyring is Phase 3's projection, genesis
  bootstrap and tests inject fixture resolvers; verification stays pure and
  the keyring lands as a parameter, not a rework. (PR #79)
- 2026-08-30 — HEAD reconciliation reads the whole segment stream in v0
  (correctness first); the segment-seek optimization the plan sketches is
  deferred until the Phase 12 performance budgets exist to justify it.
  (PR #79)
- 2026-08-30 — Classification v0 bounds (plan os-d6f81ec6): canonical
  payload 4096 bytes, string 512 bytes, aggregate non-reference text 1024
  bytes, depth 8, array 64, base64 run 256; the anchor exemption requires
  a filename-alphabet path (no whitespace, 200 bytes max); pointers are
  RFC 6901-escaped; lint canonicalizes (JCS) before walking. (PR #80)
- 2026-08-30 — Genesis payload carries raw root keys hex-encoded and is
  the trust bootstrap (Resolver refuses fingerprint/key mismatches and
  signers outside the root); init refusal is exit 3 with machine code
  `ledger_not_empty`; smallest-home decision placed genesis in its own
  `internal/genesis`. (PR #83)
- 2026-08-30 — Halt state is a pure chain projection (State{Halted, By,
  Reason}; no flag file); malformed halt events are skipped in replay
  (admission refuses them at the boundary; replay must not wedge on
  history it did not admit); exit 7 allocated. (PR #84)
- 2026-08-30 — Exits 8 (`chain_invalid`) and 9 (`classification_refused`)
  allocated; error body stays {code, message} with "position N: reason:
  detail" rendered in, structured error data deferred to a versioned
  envelope bump (per #82 review). Reading records needed a public
  iterator: `ledger.Records` added as the smallest read surface (a plan
  gap, noted for review; genesis Bootstrap and `show` consume it).
  (PR #85)
- 2026-08-30 — gitref rides the git CLI (the engine's gitx pattern, no
  module deps); the loop verifies the fetched stream (full replay) before
  the head cache advances, correctness over speed until Phase 12 budgets;
  lost races surface as two rejection shapes (stale-parent non-fast-forward
  and mid-push ref-lock contention), both mapped to the same retry path.
  (PR #86)
- 2026-08-30 — Admission refusals wrap in `admit.Refusal` (rule name +
  `Unwrap` to the underlying typed error) so the exit map (7/8/9/10)
  survives composition; context is built by one verified replay
  (`ledger.WithObserver` feeds the halt projection); the version rule
  stamps refusals at the tip position. (PR #90)
- 2026-08-30 — Exits 11 `remote_rejected` and 12 `head_regression`
  allocated (lowest unused, one meaning per code): the remote's own
  admission refusal carries its reason verbatim, and a rollback is a
  freshness refusal, never misread as corrupt ledger data; loop
  exhaustion maps to exit 2 `contention`. The pre-flight's verified tip
  persists via the new `gitref.RecordVerifiedHead` before the loop runs,
  closing the fresh-client rollback window between the two fetches. The
  update-phase rejection shape ("failed to update ref") is a race marker
  again: the 2.2 CLI race drill showed a rival landing between
  advertisement and update produces it server-side, while hook declines
  keep surfacing as `ErrRemoteRejected`. `AppendLoop` grew a trailing
  variadic `ledger.VerifyOption` so per-attempt re-verification honors
  the caller's supported set. (PR #93)
- 2026-08-30 — seed-admit's division of labor: one full
  VerifyFromGenesis proves parse, linkage, signatures, actor resolution,
  version discipline, and upgrade schemas for every pushed record; the
  per-new-record admission pass then applies only what verification
  tolerates in history (halted, halt shapes, classification), and a
  record-level prefix check pins append-only-ness, since commit
  fast-forward alone would admit a descendant commit whose tree rewrites
  admitted records. Actor and version rules are not re-run per record:
  the completed replay already enforced them. (PR #94)
- 2026-08-30 — Drill fixtures select postures through the production
  declaration (`internal/posture.Load`), never a test-only switch; the
  cooperative half of the adversary drill asserts the landed raw push
  (the named consequence observable) beside the client's local refusal;
  kill-and-replace compares the hook's refusal lines byte-for-byte
  across a host rebuilt from a fresh mirror clone (replicated Git data
  only, so no host-local state can leak into the comparison). (PR #99)
- 2026-08-30 — Posture declaration is client/deployment state in a
  strict JSON file (never ledger content; the guarded ref's tree stays
  HEAD + segments); the charter's cooperative consequence lives as one
  exported constant that the doctor quotes verbatim (machine field and
  operator-facing stderr both); undeclared refuses at exit 4, invalid
  declarations at the newly allocated exit 13 `posture_invalid` (spec
  table row + envelope constant land in the same change, per the
  lowest-unused rule; exit 3 stays reserved for illegal transitions);
  the forge-hosted posture names its Phase 12 gap honestly. (PR #98)
- 2026-08-30 — v1-loop delegation for `next:` cards: operator queue verbs
  (`promote`; `plan-unblock` once the gate PR is genuinely merged) run under
  the session principal (`shaunlmason`), work verbs under
  `seed-next-implementer`; implementation may be prepared ahead of the plan
  merge as a draft PR when the owner is merging in batches (CI's
  plan-at-merge-base gate still orders the merges). ADR:
  `decisions/0003-next-loop-delegation.md`. (PR #72)
- 2026-08-30 — The keyring semantics activate behind a **`seed/1`
  boundary** (review finding on #97): they are a validation-rule change a
  conformant `seed/0` validator judges differently, so per protocol.md's
  own bump discipline `actor.*` records at `seed/0` positions stay inert
  and grandfathered, an `actor.*` draft at a `seed/0` tip refuses as an
  illegal verb, and `version.Supported()` ({seed/0, seed/1}) seeds every
  default supported set. `version.Protocol` (the genesis default) stays
  `seed/0` until a recorded decision moves it. (PR #100)
- 2026-08-30 — One transition function: `keyring.Advance` owns actor
  payload shapes, standing legality (re-enrollment reinstates a
  suspended actor; revocation is terminal; root liveness never leaves
  zero active roots — review finding on #97), and effects, consumed by
  verification replay (chain validity, `bad_actor_event` at the failing
  position) and admission preview alike. The interim root-only
  authorization for `actor.*` verbs is **admission policy** like halt
  and classification — verification tolerates it in history (the
  cooperative posture's named consequence) — so 3.2 swaps it for grant
  checks without another protocol bump; exit 14 `out_of_grant`
  allocated for the refusal the charter names. (PR #100)
- 2026-08-30 — Capability vocabulary v0 (plans/os-3979d48b.md): rows are
  **sets of accepted capabilities** per verb (any one admits), mirrored
  from the normative table in next/spec/actors.md and pinned by test;
  halt declare/lift, protocol upgrade, and actor.* accept `operator`
  only, `system.checkpoint` accepts `maintenance` or `operator` (the
  charter's maintenance loop must not hold operator authority);
  governance roots hold `operator` implicitly; grant-level withdrawal
  short of ending standing is deferred until the catalog grows a verb.
  The no-protocol-bump stance for admission-policy evolution (recorded
  at 3.1, argued in the plan's versioning section) is applied: the
  grant rule replaces root-only authorization at `seed/1` with no new
  version. (PR #102)
- 2026-08-30 — The revocation drill's shape (plans/os-d1f35a8c.md): the
  working era runs through the legitimate cooperative client against
  the enforced boundary, hostile eras push raw; standing-before-grants
  is proven by granting the doomed key a capability before revocation
  and asserting it confers nothing after; terminality and root liveness
  are asserted at the boundary, not only in the library. The Phase 3
  exit record claims only the III.E subset and enumerates every unmet
  criterion with its landing phase (per #103 review): honest conformance
  bookkeeping outranks a tidy exit line. (PR #104)
- 2026-08-30 — Projection publication scheme (plans/os-4d5cacff.md): a
  directory rename cannot atomically replace a non-empty directory and
  delete-then-rename opens the window atomicity forbids (review finding
  on #105), so publication is **immutable builds plus a pointer**:
  `<out>/<name>/builds/<position>-<tip12>/` trees named by the stamp
  (identical prefixes reproduce identical ids) with an atomically
  renamed `CURRENT` file; the pointer swaps only after the tree is
  complete; superseded builds and stray partials prune after the swap;
  a killed build leaves at worst an orphan. Stamp conventions:
  `projection.json` carries the verified record **count**, the CLI
  envelope stamps the tip's zero-based **index** (count-1), both stated
  in spec/projections.md so consumers never conflate them. (PR #109)
- 2026-08-30 — Roster candidates derive from the chain itself
  (plans/os-4d5cacff.md): the genesis payload's governance roots plus
  every enrollment subject, resolved through `keyring.StateAt` — every
  keyring entry appears, roots included (`root: true`, empty kind and
  name, per #105 review), and the projection stays a pure function of
  the records without adding a keyring iterator the plan's file scope
  did not name. (PR #109)
- 2026-08-30 — Build identity carries the derivation version; the
  superseded build survives one swap (review findings on #109). The id
  is `<position>-<tip12>-v<version>`: a projection whose build logic
  changes bumps its version and republishes at an unchanged tip, so
  the same-id discard can never preserve obsolete semantics; and the
  build CURRENT named before a swap is retained through the prune so a
  reader that resolved just before the swap still holds a complete
  tree (older builds and stray partials prune; losing two consecutive
  swaps means re-resolving). The stamp gains the version field, giving
  consumers the derivation identity beside the position. (PR #109)
- 2026-08-30 — Standard-projection semantics (plans/os-fecfb3f7.md): the
  v0 work classifier is the prefix rule (everything outside `system.*`
  and `actor.*` is work vocabulary) until Phase 5's transition table
  replaces it with explicit vocabulary; the queue publishes with
  `derivation: "none"` so an underived queue is machine-distinguishable
  from "nothing ready" (per #106 review); contract payloads are
  content-preserved but re-indented by the view (canonical bytes live
  only in the ledger, and the spec says so); exit 15 `stale` is
  allocated for the minimum-position demand, spec row and constant in
  one change per the allocation rule. Two position conventions coexist
  deliberately: stamps carry the verified count, tip-stamped envelopes
  carry the zero-based index; `seed project current` reports the count
  verbatim and the spec states both. (PR #111)
- 2026-08-30 — Consumer-verb refusal discipline (review findings on
  #111): `seed project current` resolves only registered projections
  (a name outside the registry is not_found whatever directories
  exist, which also keeps traversal components out of the path), and
  splits absence from damage: nothing published refuses 4 not_found,
  while a layout that exists but cannot resolve (unreadable or empty
  CURRENT, unreadable or unparseable stamp) refuses 5 unavailable, so
  automation branching on exit codes never mistakes a damaged
  publication for an unpublished one. The verb and the rebuild rows
  also surface the stamp's derivation version. (PR #111)
- 2026-08-30 — Refusal discipline round two (review findings on
  #117): absence means the projection's own directory is absent — a
  layout that exists without its CURRENT pointer is damage (5), since
  publication swaps CURRENT atomically and a published pointer never
  vanishes on its own; a stamp that parses but is incomplete or
  inconsistent (wrong name, empty version, tip contradicting
  position) is the same damage as an unparseable one, so a zero-value
  stamp cannot satisfy even --min-position 0; and the stale refusal
  stamps its envelope position with the observed count, per the
  envelope contract that every response computed at a verified
  position carries it structurally. (PR #117)
- 2026-08-30 — Write-boundary enforcement shape (plans/os-8d5e9c45.md,
  amended per #107 review): three layers — the vocabulary lint, the
  seam/write-separation lint (a non-test file importing the engine
  makes no os write-family calls; per-file because cmd/seed's importer
  and writers are legitimately different files of one package), and
  locked publication (0444 files, 0555 directories including the
  projection root, engine-only 0755 window around the swap). Both
  lints are one go/parser test in the engine suite, self-checked on
  planted fixtures, which is what "wired into check-next" means: the
  suite is the gate. Deletion becomes mode-walk-then-remove and the
  sanctioned recovery stays `seed project rebuild`, which runs the
  walk itself. Residual risks named in the spec instead of papered
  over: cross-file splits (closed by modes), direct syscall, and
  chmod-capable owners (uid 0 bypasses modes entirely, so the
  OS-refusal drills gate on an unprivileged runner; CI provides one).
  (PR #112)
- 2026-08-30 — Boundary hardening round two (review findings on #112):
  the output root itself locks 0555 between rebuilds because rename
  permission lives in the parent (a writable parent let a whole
  projection root be renamed away, evading both per-file lints); the
  window opens only after verification so refuse-before-write holds,
  and every return path relocks via defer, failed publications
  included (only a killed process leaves a window open); every
  published mode is set by explicit chmod so the process umask cannot
  weaken the protocol (WriteFile's mode argument is umask-masked).
  Renaming the output root from ITS parent stays outside the engine's
  ownership, equivalent to repointing --out, and is named residual in
  the spec. (PR #112)
- 2026-08-30 — Boundary hardening round three (review findings on
  #118): a partial window open rolls itself back — openDirs relocks
  the directories it already opened before surfacing the error, so a
  refusing directory (builds/ occupied by a regular file) never
  strands a projection root writable until some later successful
  rebuild; and the lint vocabulary derives from the engine's own
  declarations (CurrentFile, StampFile) instead of re-typed literals,
  with the unexported builds directory pinned behaviorally against a
  freshly published layout — unexported deliberately, since exporting
  it would hand non-engine code the very path piece the lint denies.
  (PR #118)
- 2026-08-30 — The cache is registered projection number six
  (plans/os-acc1ac78.md as amended by #110): modernc.org/sqlite
  v1.57.0 pinned; byte-determinism engineered by closing the variance
  sources (one connection, one ordered transaction, rollback journal,
  fixed page_size, no auto_vacuum/ANALYZE/AUTOINCREMENT) and enforced
  by the registry's existing byte-identical drill; the stamp table
  derives the same (name, position, tip, version) the engine stamps
  in projection.json (position = len(records), tip = last event hash,
  both provably equal the verification report); PRAGMA user_version
  carries the table-set generation. A same-id republish deliberately
  keeps the existing tree for readers that hold it, so tamper
  recovery is the documented deletion walk plus one rebuild; the
  locks are the anti-tamper layer, and the drill proves the poison is
  never an input. (PR #119)
- 2026-08-30 — The lifecycle contract is data (plans/os-d69a6c91.md):
  transitions.json is the normative copy with a byte-identical
  embedded twin (the classify.json precedent) and refuses to load
  unless self-validation passes — one initial state, one birth verb
  landing on it, no terminal outgoing rows, no duplicates, full
  reachability, no wedge, and the in_progress exits pinned to the
  four deliberate ones. Legality is admission policy at seed/1
  (seed/0 inert, the keyring precedent): the lifecycle rule folds
  the subject's history and refuses illegal (state, verb) pairs as
  exit 3 naming subject, state, and verb; completeness presence for
  intent.filed/contract.specified refuses at shape; capability lanes
  dispatch/claim gate every lifecycle verb with cancellation and
  merge.observed operator-only in v0. Raw-pushed illegal history
  verifies and surfaces as per-contract anomaly counts; the fold
  feeds contracts v2 (state + anomalies), queue v2 (transitions/1
  ready set, oldest first), and cache generation 2 — the derivation
  bump exercising Phase 4's version-in-identity machinery in anger.
  The hook's per-record context now carries the fold, so the shared
  rule set enforces at the boundary too. (PR #122)
- 2026-08-30 — Claims are fenced, structured, and online-only
  (plans/os-5dc16a7c.md): the fence is the admitted claim.taken
  position — derived, never asserted — cited as {"fence": "<n>"}; on
  a held subject the four deliberate exits must cite it, so must
  free events from the holder or any prior claimant (a reaped worker
  cannot demote itself to observer, the #114 review binding), any
  citation present must match the active fence whoever signs, and
  outside a claim window citing a fence refuses because fences die
  with their windows. The fence rule slots between grant and
  lifecycle per the charter's check order, so out_of_grant beats
  fenced_out beats invalid_transition, and citations are required
  only for the verbs the table allows out of the held state — an
  illegal jump stays exit 3, never a misleading 6. Contention is the
  lifecycle rule's structured exit-2 refusal naming holder and fence
  (a holder double-claiming loses too); the offline exclusive-verb
  refusal reuses exit 2's own phrase (exclusivity not granted) at
  the dev-tool seam, with the boundary enforcing regardless. The
  contracts view carries {holder, fence} while in_progress
  (Version 3), the cache mirrors it (generation 3), and the race
  storm drives six concurrent claimants through the full rule set
  against a real remote: one winner, structured losers, a verified
  chain. (PR #123)
- 2026-08-31 — Packets are the only executor interface, enforced
  (plans/os-b07b0f59.md): all four deliberate exits carry an inline
  four-part packet under the payload's packet key — acceptance
  non-empty, decisions structurally marked verified/asserted, the
  mandatory bare base range (zero-length for no work) with combined
  anchored refs, findings possibly empty but never absent — strict
  keys, 3072 canonical bytes fitting the 4096 payload cap, and
  packet_ref reserved for the Phase 6 artifact store. The packet
  rule runs between fence and lifecycle so a shape-invalid packet
  refuses before the transition applies; the classifier's reference
  exemption gained the bare commit-range form (exported predicates
  keep packet and classifier grammars from drifting); and the
  tolerant fold now counts raw-pushed fence violations and
  packetless exits as visible anomalies while still applying the
  exit, because skipping it would wedge the subject on a dead
  holder. Sufficiency is the A/B resume drill: B, a function of the
  packet alone in a fresh clone, completes the acceptance list,
  reproduces the artifacts from the anchors, and never re-tries the
  recorded dead end. (PR #124)
- 2026-08-31 — The spec gate is structural and universal
  (plans/os-73c00a50.md): contract.specified carries the structured
  acceptance object — commit-anchored ref naming one commit, the
  executable flag, gate evidence present iff executable and bound to
  the ref's exact revision by string equality (an unrelated merged
  PR vouches for nothing) — with no tier exemption: the
  provenance-gated trivial-tier relaxation waits for the tier
  system. request.* payloads structurally cannot carry executable or
  gate keys at any depth (outside text proposes, never arms), the
  fold records {ref, executable, gated} per subject with raw-pushed
  ungated content visibly anomalous and never marked gated, and the
  contracts view (Version 4) plus cache (generation 4) publish it so
  Phase 6's verifier reads "may this spec run?" from a projection.
  The same bump is the republish trigger for 5.3's fold change
  (raw-pushed fence violations and packetless exits now counted as
  anomalies): that change shipped in #124 without a derivation-version
  bump, so a prefix published under Version 3 from a ledger holding
  such raw records would rebuild to different bytes under the same
  build id and be discarded as a duplicate; Version 4 / generation 4
  re-key every such rebuild (#124 review). Gate-before-specified
  enforced here; gate-before-run named as Phase 6's half. (PR #125)
- 2026-08-31 — Plan-gating is ordering plus citation at admission,
  ancestry at the verdict (plans/os-16c1d142.md): the fold tracks
  the filed tier and plan.approved facts (plan.* verbs are facts,
  not transitions — the pinned four exits stand), and above the
  literal "trivial" tier a submission refuses exit 16 plan_required
  without an admitted approval AND the cited plan anchor; Phase 6's
  receipt (plan hash at merge-base) closes the
  implementation-before-approval window, named not silent. The
  falsifiable-plan lint is a pure reader of the repo's own plan
  shape with missing-retention the charter-quoted distinct finding;
  seed plan classify is the CI-invocable disjointness check refusing
  mixed change sets at exit 9, with forge-required wiring assigned
  to Phase 12's protections reconciler. plan.proposed rides the
  claim lane under the existing fence matrix; plan.approved is
  operator-attested v0 like merge.observed. (PR #126)
- 2026-08-31 — Coverage collection in check-next serializes package
  test binaries (-p 1), both Makefiles in lockstep: under the
  subprocess-heavy drills, concurrent test binaries can collide
  coverage counter files (same pid and second after heavy pid
  recycling), silently dropping one package from the merged profile
  and misreading totals far below truth (79-89% readings against a
  stable 91.1%). The meter now measures the same number every run;
  the gate itself is unchanged. (PR #125)
- 2026-08-31 — 5.3's post-merge review round hardened in place
  (os-b07b0f59 follow-up): packet array parts must be literal arrays
  (a null decodes into the same nil slice an absent key does and
  admitted past presence checks); the resume drill now drills what it
  claims — every ref resolves from its OWN anchor (the fixture makes
  the base and head disagree about the bytes, and the range form is
  exercised at the range head), A dies between acceptance items so B
  performs and durably lands genuinely unfinished work, and B's
  repository coordinate comes from the instantiation's durable
  config, never A's environment: the packet carries anchors into the
  instantiation, not the instantiation's address. The round's
  projection-version finding rides #125 (Version 4 / generation 4,
  noted there).
- 2026-08-31 — 5.2's post-merge review round (os-5dc16a7c follow-up):
  the fence rule's exclusive-verb bypass returned before citation
  validation, so a claim.taken on an unheld subject could smuggle an
  asserted or retired fence citation past the "outside in_progress,
  citing one refuses" rule the spec already states. The bypass now
  applies only on a held subject (where a rival claim is contention,
  the lifecycle rule's refusal); claimless citations on the claiming
  verb refuse exit 6 like any other. The round's fold finding was
  already closed by #124's anomaly counting, with apply-and-count
  over skip the recorded decision (a skipped exit wedges the subject
  on a dead holder).
- 2026-08-31 — 5.1's post-merge review round (os-d69a6c91 follow-up):
  the lifecycle fold now honors the seed/1 activation boundary the
  spec and admission already stated (records under seed/0 are
  grandfathered inert, the keyring.Applies posture). The fold
  filtered by verb name alone, so an upgraded ledger's pre-activation
  events would occupy states and make the real seed/1 filing refuse
  as a second birth; version discipline pins e.V to the version
  active at each position, so the filter is one record-level check.
  Every fold-consuming derivation bumps with the corrected fold
  (contracts 5, queue 3, cache 5): a published projection rebuilds to
  different bytes for such a ledger, and only a new version re-keys
  the build id so the correction republishes at an unchanged tip
  (#129 review, the #124-round principle).
- 2026-08-31 — Observations are declared inputs, never ambient
  (plans/os-2ff8dbf1.md): the v0 channel is per-executor JSONL
  streams keyed <actor>/<fence> under next/var/obs/ (the existing
  var/ ignore rule already covers the directory), unsigned and lossy
  by declaration; classification (live/expired/wedged/no_data) is a
  pure function of the active claim's own stream, a declared as_of,
  and the spec'd thresholds (900s/1800s), with no wall clock in any
  build. The engine grew one seam: Builders take Inputs, only the
  report declares consumption (Version "2"), and an input-bearing
  build keys its stamp and build id with the snapshot's RFC 8785
  digest (-i<digest12>) so changed inputs republish at an unchanged
  tip while input-free projections stay byte-identical by
  construction. progress.milestone admits strictly advancing counts
  at a 25-position minimum spacing (position-derived, never ts,
  which is metadata with no ordering authority); wedge.declared is
  an operator fact with presence-checked evidence; both are facts,
  never transitions. The ungoverned-verb test specimen moved from
  progress.milestone to message.sent, since milestones now carry a
  capability row.
- 2026-08-31 — The verdict pipeline's first half binds at admission
  and runs in declared isolation (plans/os-f6d2c267.md, task PR for
  os-f6d2c267). verdict.rendered is piped: a fact admitted only on
  review subjects, strict payload {verdict, receipt, submission,
  independence: "L1"}, bound to the fold-recorded submission
  {position, signer}, and refused exit 17 not_independent when the
  signer is any claimant or the bound submission's signer — the one
  capability row without an operator fallback (III.G: override is
  6.4's own verb; roots that judge hold explicit verdict grants).
  The verifier workspace is an origin-stripped local clone, never a
  git worktree (a worktree's .git link shares the parent's refs and
  objects, handing hostile spec commands update-ref reach back into
  the host; review finding on the plan), and the charter's "sandbox
  with declared, minimal capability" lands as a runner profile named
  in every receipt: v0 exec scrubs the environment, bounds each
  command with a process-group-killing timeout, and declares
  network: unrestricted honestly rather than pretending a boundary
  portable no-root Go cannot enforce; namespaced profiles slot in at
  the Phase 7 adapter seam. Receipts are JCS-canonical
  {contract, merge_base, head (full immutable SHAs, head must
  descend from merge-base), plan-at-merge-base | null, diff_sha256,
  files, transcripts (output digests, never inline bytes),
  environment}, stored content-addressed under next/var/artifacts
  (the git-addressed refs/seed/artifacts push stays deferred, the
  observation-channel precedent). Render derives the permissible
  verdict from the transcripts it just executed — pass over any
  nonzero exit refuses the new exit 20 checks_red naming the
  command, fail stays renderable, prose-only pass stays explicit
  judgment — because the verifier is the only party holding what it
  just ran; a raw-pushed pass-over-red goes red under seed verdict
  check, whose recompute-and-mismatch refusal took the new exit 21
  receipt_mismatch (both allocated per envelope.md's lowest-unused
  rule alongside the plan's 17/18/19; the transcript-gate and
  mismatch conditions surfaced in review and at implementation).
  Gate-before-run consumes the projection's gated flag: ungated
  executable content refuses 18 with nothing executed, and
  declared-executable content yielding no parseable commands
  refuses 19 — the command grammar is the plan grammar via the
  exported plan.Commands, so the verifier executes exactly what the
  lint reads. The contracts view is unchanged; surfacing verdicts in
  projections rides 6.2's divergence work.
- 2026-08-31 — The reconciliation chain is fully piped and divergence
  is a detected, surfaced state (plans/os-6cdc15be.md, task PR for
  os-6cdc15be). merge.requested admits only in review citing the
  admitted pass verdict by position (the chain-legality half of "a
  red verdict is unmergeable"; the implementer lockout half is
  6.4's), from the claim lane; merge.observed stays the table's one
  transition to done and deepens to the observer's forge fact
  {merged: full sha, pr}, admitted only behind the full chain (pass
  verdict, then a request citing it), from the new observer
  capability lane. The fold records the chain facts (latest verdict
  pass-or-fail for 6.4, latest request with its citation, and THE
  admitted observation, singular by construction since done is
  terminal; a raw-pushed skipped chain applies tolerantly and counts
  an anomaly like a packetless exit). Divergence detection is split
  by what each surface may read: internal/reconcile and the report's
  new reconciliation section carry the record-derivable classes
  (merge_without_verdict, chain_skipped, unreconciled — the last
  reported neutrally: no build carries a wall clock, so
  pending-versus-failed is Phase 9 maintenance's age judgment), and
  seed reconcile alone reads the artifact store and the repository
  for the evidence grades: attested-head reconciliation with honest
  ancestry cases (fast-forward and true merge commits clean;
  anything else, rebase and squash flows included by design in v0,
  surfaces attested_divergence as a state, not a fabrication
  verdict, with patch-equivalence reconciliation against the
  receipt's diff hash as the named successor) and target-rewrite
  detection by observing the target ref (v0: the checked-out default
  branch tip; a forge force-push writes no ledger event, so the
  reachability of the observed merge under the tip is the signal —
  review finding that replaced an unreachable two-observation
  design). Detection is a report at exit 0, never a refusal;
  projections surface the pipeline at contracts v6, report v3, and
  cache generation 5.
- 2026-08-31 — Sealed checks are a commitment with a pre-claim window,
  encrypted custody, and a verifier-boundary gate (os-3128535a,
  plans/os-3128535a.md; charter II §7, §8; III.F sealed rows;
  next/spec/sealed-checks.md). check.sealed is a fact admitted only
  in ready with NO prior claim.taken — the release path lands
  subjects back in ready, so state alone would launder
  post-implementation seals (review finding on plan #138) — one
  commitment per subject, and the fold records the fact only from
  that legal window (raw seals outside it are anomalies, never
  facts: they must not retroactively claim a pre-existence the
  ordering disproves). The commitment is SHA-256 over the JCS sealed
  envelope {salt, checks}, the 32-byte salt living only inside the
  ciphertext (publishing it would invite dictionary attacks on
  low-entropy check bodies), so commitment verification is a
  decrypting party's act, performed at every unseal; empty seals
  refuse at creation AND at unseal (a zero-check envelope would pass
  vacuously). Encryption rides the binding default filippo.io/age
  v1.2.1 with recipients derived from the verdict-granted ed25519
  keys as ssh-ed25519 age recipients (agessh) — "recipients = the
  verifier keyring" with no new key material; the cross-protocol use
  of one ed25519 key for verdict signatures and seal unwrapping is
  the documented v0 trade, dedicated X25519 enrollment the named
  successor. Ciphertext is mutable custody at
  next/var/artifacts/sealed/<commitment>.age (the ledger references
  it by the immutable commitment; deletion is the charter's erasure
  path, surfaced by the audit); rotation decrypts with a still-able
  identity and re-encrypts open subjects to the current keyring,
  writing no ledger events, terminal subjects skipped (exposure
  bounded to the compromise window). Authoring isolation binds three
  times: the sealer capability row has no operator fallback (the
  verdict-lane posture: operator stands in the claim lanes, so a
  fallback would put authoring and implementation on one capability
  and the audit could prove nothing — review finding), actor.granted
  refuses sealer/claim and sealer/operator co-holding both
  directions (a root's implicit operator standing included), and
  claim.taken refuses the seal author. Receipts gain commitment +
  sealed_transcripts via omitempty so every pre-6.3 receipt's
  canonical bytes and digest are unchanged (the compatibility twin
  of the views' explicit-null convention); verdict check on a sealed
  subject requires an unsealing identity and reruns the sealed
  commands inside recompute-and-mismatch (review finding: presence
  checking would let invented sealed transcripts check green).
  Render gates above-trivial unsealed subjects at exit 24; exits 22
  seal_broken and 23 not_recipient name the broken-seal and
  rotation-lag refusals. seed seal audit reads only the age header's
  recipient stanza tags (agessh four-byte fingerprints — an
  identification hint; the decrypt drills carry the cryptographic
  claim) and reports stale/foreign/missing at exit 0; reconcile
  gains the neutral unsealed class. Projections: contracts v7
  (sealed explicit-null), report v4, cache generation 6.
- 2026-08-31 — The red-verdict lockout binds authenticated fails to
  submissions, the return path resolves the review exit, and the
  override is a gated escape hatch (os-d2497eb7,
  plans/os-d2497eb7.md; charter II §8, III.G lockout and override
  rows). contract.returned (review to ready, a new table row) admits
  only citing a standing fail verdict on the current submission that
  passes the 6.2 verifier boundary; prior facts and the 6.3 seal
  survive the return (the commitment predates the FIRST claim, which
  is the proof that matters). The lockout refuses
  verdict.rendered(pass) at admission and seed verdict render (exit
  25 red_locked) while any authenticated fail cites the bound
  submission; the fold keeps the whole window's fails
  (SubmissionFails, cleared on each submission.made) precisely so a
  raw-pushed later verdict can never bury an authentic fail and
  unlock pass (review finding on plan #140: unauthenticated fails
  lock nothing and authorize nothing — a raw fail would otherwise be
  a denial-of-service lever). Fail restatements stay renderable.
  merge.overridden is the third no-fallback capability row (operator
  only): strict {reason, verdict} citing an admitted,
  boundary-validated fail on the current submission (review finding:
  without the citation gate the hatch was a general bypass of
  independent verification), one per submission window, folded as
  OverrideFact never a verdict. merge.requested cites exactly one of
  verdict or override; merge.observed accepts the override-backed
  path with both chain steps validating the override signer against
  the operator boundary; reconcile adds VerifyOverrides with
  override_unverified (position-accurate) and the neutral overridden
  class, and an override-backed done is never merge_without_verdict
  (the override is its sanctioned cover; a raw override still
  surfaces override_unverified beside it). The
  no-verifier-available emergency (overriding with no verdict at
  all) is deferred to the halt and governance paths. Projections:
  contracts v8 (override explicit-null), report v5, cache
  generation 7.
- 2026-08-31 — Offers are supervisor facts inviting claims, consumed
  by the claims they invite (os-c61c3392, plans/os-c61c3392.md;
  charter II §9, III.H rows 1–2). offer.published admits only on
  ready subjects from the new supervise capability or its operator
  fallback (no disjointness: an offer grants nothing — the claim it
  invites settles at admission like any claim); strict
  {eligibility{capabilities, tiers}, expires} payload with a
  deterministic born-dead refusal against the event's own ts, since
  admission never reads a wall clock. Liveness is fully derived,
  never stored: ready AND unexpired AND no applied claim.taken after
  the offer's position (review finding on plan #144: without the
  consumption term a taken offer resurrects when the subject
  re-readies inside its expiry window, rescheduling from stale
  supervisor intent — "claimed or expire" makes re-offering a fresh
  publication). No offer.expired/withdrawn verbs; no position
  throttle (publishing is lane-gated, unlike open-lane milestones).
  The tolerant fold records raw pushes as OfferFacts and seed offer
  list applies the laundering shape at the consuming surface
  (position-accurate supervise boundary via keyring replay), so
  foreign offers are inert: never listed, never authority. The
  wakeless poll-only drill runs the whole loop to done with no wake
  channel in existence, and the race drill's loser gets the
  structured contention refusal at admission. Verb-catalog growth is
  additive (no protocol bump). Projections: contracts v9 (offers +
  last_claim, omitempty so offer-free chains keep byte-identical
  views), report v6 (republish only), cache generation 8 (offers
  table + last_claim column). Review round on #145: operator
  standing satisfies every eligibility scope at the list surface
  (admission already lets the operator take any offered work, so it
  must be able to discover it), and the view serializes last_claim
  only beside offer facts, so ever-claimed offer-free subjects keep
  byte-identical v8 bodies; the cache column stays full-fidelity
  under its new generation.
- 2026-08-31 — Budgets are per-contract reservations checked and
  decremented at admission, with every derived surface applying the
  laundering shape (os-cecac5de, plans/os-cecac5de.md as amended by
  #148; charter II §9 "Budgets are reservations, not observations",
  III.H rows 3–5). Capacity comes from the filed budget class
  through the spec-pinned table (small 100, medium 1000, large
  10000; unknown class = zero capacity, never fudged); org/actor
  granularity and the per-adapter risk-limit surface stay named
  extension points for 7.3+. budget.reserve/settle/release are
  {claim, operator} facts admitted only in_progress; a reserve
  additionally requires the drafting signer to be the ACTIVE claim
  holder or operator (review finding on #147: prior claimants can
  cite the active fence, and treating them as boundary-valid would
  let a released worker consume the next holder's budget). The
  fold records reserves and close attempts as independent lists and
  NEVER mutates a reservation (second #147 finding: a fold-level
  close on any well-shaped raw fact would let a foreign release
  free capacity for over-spend and a foreign settle lock the owner
  out); effective closure is derived at every consuming surface as
  the first close by the reservation's own signer or operator,
  position-accurately. Remaining = capacity − open valid
  reservations − settled actuals; settle records TRUE actuals,
  overruns included. The race drill passes two 8-unit drafts
  through one pre-admission view of a 10-unit class and admits
  exactly one — the §II.9 argument executed. The spending-verb gate
  ships as an empty data table with an injection drill; the
  envelope budget block is Phase 8's and park-on-exhaustion 7.4's.
  Projections: contracts v10 (budget + reservations, omitempty so
  budget-inactive chains keep byte-identical views), report v7
  (republish only), cache generation 9 (reservations table + budget
  columns).
- 2026-08-31 — The executor adapter is a public package and execution
  is fenced to the reservation end to end (os-1dad487d,
  plans/os-1dad487d.md as amended in the #150 review round; charter
  II §9 executor prose, III.H rows 6–8). next/executor is the
  module's one non-internal package: Adapter
  (Provision/Wake/Tuple) and Run (Workspace/Meter/Dispose), with the
  local worktree adapter first (detached git worktree, fixed argv,
  packet at .seed-run/packet.json, Wake the documented no-op, Tuple
  the honest local-worktree/v0 stub until Phase 10). run.started is
  the spending-verb table's first entry — strict {fence,
  reservation}, in_progress only, the ACTIVE fence, an open valid
  reservation revalidated position-accurately, once per fence,
  {supervise, operator} — and Provision refuses without the admitted
  start it cites, so the bracket is reserve, start, provision,
  meter, settle, budget.settle. run.settled aggregates once per
  started fence (prior fences included: a run settles after its
  window closed); metering rides the observation channel via the
  additive units line field. One review-visible deviation from the
  plan's mechanism note: the plan said the run facts' fence field
  "doubles as the fence rule's citation", but the fence rule's
  active-fence semantics would refuse the prior-fence settles the
  plan REQUIRES, so run verbs are fence-rule-exempt and the run rule
  validates the reference against the applied claim positions
  instead — behavior preserved, mechanism corrected. The
  disposability drill SIGKILLs a real worker subprocess (test-binary
  re-exec) at a seeded randomized site after an admitted
  submission.made: the chain verifies, the contract completes
  elsewhere from the surviving ledger alone, the loss is exactly the
  post-sync observation lines, and disposing the dead workspace
  loses nothing admitted. Projections: contracts v11 (run facts,
  omitempty), report v8 (republish only), cache generation 10 (runs
  table).

## 2026-08-31 — 7.4 graceful preemption (os-0f718b4e, plan #153)

- The interrupt request is a ledger fact: run.interrupted, strict
  {fence}, admitted only in_progress on the ACTIVE claim fence, from
  {supervise, operator}, once per fence (raw duplicates fold as
  anomalies, the run-fact posture; a raw invalid interrupt neither
  blocks the legitimate supervisor's nor is one). The chain is the
  canonical channel — workers already poll it for liveness, so no
  marker files or adapter side-channels exist in v0. Deliberately
  NOT a spending verb: preemption is supervisory control and must
  not be budget-gated. Validity is position-accurate from birth
  (InterruptValid at records[:pos]; InterruptRequested the one
  shared derivation), applying the RunStartValid review lesson
  proactively and closing the raw-push denial-of-service shape:
  conforming workers park only for boundary-valid interrupts.
- Safe-point semantics are the worker contract, specified in
  executors.md's Preemption section: check at bounded intervals (at
  least once per metering/poll cycle), finish the current step,
  write the four-part packet, exit deliberately via claim.parked —
  park, not release, so blocked hands routing back to the dispatch
  lane. The next/executor public surface is untouched: the check is
  the worker's, exactly where the charter locates it.
- The force path yields an honest reap packet from what is known:
  SIGKILL the interrupt-ignoring worker, claim.reaped by
  dispatch/operator (acceptance from the specified criteria, base
  the zero-length range when no pushed work is known, findings
  recording the ignored interrupt and the kill), subject back to
  ready and immediately re-claimable; run.settled records the dead
  run afterward. B-style automatic timeout reaping stays Phase 9's
  maintenance loop, which presupposes exactly these semantics. Both
  paths drilled end-to-end with real subprocess workers completing
  elsewhere from the surviving ledger alone. Projections: contracts
  v12 (interrupts, omitempty), report v9 (republish only), cache
  generation 11 (interrupts table).

## 2026-08-31 — 8.1 affordance envelope (os-f5551001, plan #158)

- Affordances are computed by probing the real rule set: one signed
  draft per catalog verb through the same admit.Check admission
  enforces — one rule set, two consumers, zero exceptions (the
  signature rule included: probes are signed with the caller's key,
  so the API takes the private key, and a fingerprint-only surface
  cannot compute). The probe-payload table is synthesis data, never
  legality logic; templates fill from the live context (active
  fence, open reservation, bound submission, standing verdict) with
  placeholder positions where anchors are absent, so illegality is
  always the rules' judgment. Fence-optional verbs cite the active
  fence exactly when one stands, matching the fence matrix for
  holder and non-holder signers alike.
- No under-listing: every catalog verb has a synthesizer, pinned by
  a completeness test against the spec verb table, and the
  lifecycle walk asserts each verb listed somewhere legal. The one
  named carve-out is actor.enrolled, whose valid payload requires
  the queried subject's public key: fingerprints are hashes, no
  prober can derive it, and the enrollment surface knows its key
  out of band (spec sentence records it).
- Every append-path response stamps affordances and the budget
  block, success and refusal alike, via one shared helper; stamping
  degrades to the empty list rather than ever failing the verb.
  budget status gains --key (not --actor: probes must be signed).
  The budget block derives from the shared BudgetViewAt: reserved =
  open valid reservation sum, remaining = derived remaining.
- The walk corrected two of the plan's legality guesses against the
  actual rules (halt.lifted is admissible with no halt standing;
  contract.blocked's one source state is ready): the affordance
  computation reports the rules, so the drills assert the rules.

## 2026-08-31 — ledger writeHead race fix (os-c6fb95ee, plan #159)

- The store's HEAD rewrite used one shared HEAD.tmp path from two
  processes by design: the appender, and any poll-only reader whose
  Open landed mid-append and ran healHead's repair — the reader's
  rename consumed the appender's temp and the append failed with
  ENOENT (the TestGracefulPreemptionDrill CI flake, and in
  hindsight the #156 verify make-check red). writeHead now
  allocates per-writer unique temps via os.CreateTemp with
  best-effort removal on every failure path; renames stay atomic
  over their own files, both racers write forward-consistent heads,
  and a momentarily stale HEAD self-heals on the next Open, which
  is the repair path's job. No lockfile, no layout change.
- The regression test attacks the store directly (one goroutine
  appending 400 records, one Opening in a loop): the drill-level
  window is too narrow on fast disks, the store-level one is not —
  pre-fix it reproduces the exact ENOENT at the first contended
  append, post-fix every append succeeds and the chain verifies.

## 2026-08-31 — 8.2 affordance regression class (os-148d3ba1, plan #162)

- The III.I bug class is executable: TestAffordanceRegressionClass
  sweeps every prefix of the shared walk scenario (the walk's
  history extracted into walkScript, so the walk's stations and the
  sweep replay one script) and, for all seven enrolled-lane pairs,
  asserts sortedness, determinism, and that every listed verb
  re-admits through the enforcing Check when independently
  re-drafted at that position; failures name the class (position,
  lane, subject, verb, refusal). The sweep's probe-view derivation
  is a deliberate independent copy of the production one:
  divergence between the two fails the class, which is the drift
  signal the test exists to raise.
- The position stamp is pinned end to end at the CLI: success and
  refusal envelopes alike stamp the tip ordinal of the context
  their list was computed at (ctx.Count - 1; the plan review caught
  the off-by-one), and the stamped list equals an independent
  admit.Affordances recomputation at that context for the signing
  key on both paths. The drill's refusal half is the keyring
  preview's malformed-actor-event refusal, because that is the
  local append path's own refusal: the path runs the preview, not
  the full admission pipeline, so lane-invalid non-actor payloads
  append raw there by design and the fold-side validity families
  keep them inert.

## 2026-09-01 — 8.3 refusal-rate metric (os-edf73d66, plan #164)

- The metric's source is the attempts journal, attempts.jsonl beside
  the ledger: one strict JSONL line per admission-boundary attempt,
  BOTH outcomes, journaled best-effort (never failing the verb, the
  stamping posture) at every CLI seam that renders a
  position-stamped, signed envelope — ledger append's preview and
  chain-invalid refusals and its success (the lifecycle and budget
  verbs ride this seam), offer publish, verdict render, seal create,
  remote-refusal renders included. Responses without a stamped
  position are not boundary attempts; read surfaces never journal;
  the remote server-side boundary stays a named spec extension
  point. One population for the rate's numerator and denominator
  (the plan review's finding): the chain is never the denominator,
  and the section's span is positional context only.
- The journal enters the report as a declared input on the
  observations pattern: project rebuild --refusals, composing
  freely with --obs; Inputs.Digest now spans every declared family,
  each contributing its keys only when declared, so obs-only
  digests are unchanged; malformed journal lines refuse the build
  naming the line. Report v10 adds the nullable refusals section
  {inputs, refused, admitted, by_code, by_verb, span, rate}, rate
  fixed to four decimals ("0.0000" on an empty journal); the
  section echoes the journal's own digest while the stamp carries
  the full declared-inputs digest.
- The journal's durability posture (review finding on the task PR):
  a short write restores the previous length when the fragment is
  provably the file's tail (an ambiguous size under concurrent
  O_APPEND writers is left alone rather than risking a rival's
  line), and Load treats the terminating newline as the commit
  marker, ignoring a final unterminated fragment: a best-effort
  writer must never be able to poison the strict reader forever.
  Terminated malformed lines still refuse.
- Plan correction, recorded per D5: the plan's D3 guessed a cache
  generation bump with a conditional refusals key, but the cache
  mirrors the INPUT-FREE report (it builds from reportView, exactly
  why it never carries the observation section), so the refusals
  section never reaches it and no cache change ships. The
  observation precedent, not the reconciliation one, is the
  operative analogy: reconciliation is record-derived,
  refusals input-derived.

## 2026-09-01 — promotion (spin-out) defined (os-768361cc, plan #167)

- Promotion is two steps, self-hosting then distribution, and
  NEITHER cutover is autonomously decidable: the ground rules make
  v1 the entry point "until spin-out", so spin-out is the
  entry-point switch itself and the self-hosting cutover is the
  reserved escalation. The plan defines criteria and mechanics for
  both and authorizes neither.
- Promotion requires Phases 0 through 12; Phase 13 alone follows.
  The first draft of this amendment claimed Phases 10 and 11 were
  not blockers by reading "Phases 0-12 deliver core conformance" as
  permission to skip two of them. That is a misreading, recorded
  here rather than quietly corrected: the phrase describes what
  those phases deliver collectively, Phase 10's exit owns III.E and
  III.G and the III.O eval items, Phase 11's owns III.K, the
  core-conformance criterion demands every pillar's mechanisms, and
  Phase 12 declares deps: all. Promoting without them would hand
  real coordination to a system whose verdicts carry no calibration
  and whose grants carry no runtime qualification.
- The compromised-actor drill gates the SELF-HOSTING cutover, not
  distribution. The cutover is when real authority moves, so the
  drill proving the ceiling against a valid stolen key must precede
  it; nothing is at risk while Seed coordinates nothing. The first
  draft had this backwards, reasoning from visibility rather than
  from authority.
- Phase 9 gains item 5, the lane-facing surface: an obligations
  projection (what is owed, by whom, since when, discharged by which
  verbs, derived from the same tables admission enforces), a
  situation read with --since so a resuming lane pays for the delta
  rather than the world, and loop verbs that derive every derivable
  argument and refuse before signing what the tables would refuse
  after. The gap it closes is that Seed represents permission
  (affordances) but not obligation, so nothing answers what a lane
  owes; every fact needed is already folded, so this is a projection
  and never a new authority.

## 2026-09-01 — obligations projection and the situation read (os-52d5da3f, plan #170)

- Obligation is a projection over the fold, never a new authority:
  state-shaped kinds read discharged_by from the transition tables,
  and the closed fact-shaped set maps each remaining kind to the spec
  pairing it with its fact (run.settled, budget settle/release,
  verdict.rendered, merge.requested).
- The dischargeability sweep earned its place on first run by
  catching a real modeling error: verdict.unmerged advertised
  merge.observed while admission still refused it for want of a
  merge.requested. The merge chain is two events, so the kind now has
  two shapes — until a request cites the verdict the debt is the
  operator's and merge.requested pays it; after that the forge fact
  is the observer's. Same class the affordance sweep catches one
  level down, which is the argument for building the class with the
  surface rather than after it.
- The sweep asserts AT LEAST ONE discharging verb admits for the owed
  actor, not all of them: discharged_by names the acts that end the
  obligation, while who may perform each is the affordance layer's
  business (a claim is discharged by release, park, reap or
  submission, of which the holder may take three and the supervisor
  the fourth). Requiring all would force capability policy into a
  derivation that must stay a projection.
- Under a declared halt every obligation still stands and none is
  dischargeable; the sweep checks only well-formedness at halted
  positions, because that is the halt working rather than an
  obligation defect.
- --since cites a tip ORDINAL, so the prefix a resuming lane last saw
  is records[:position+1]. The first implementation used
  records[:position] and the round-trip drill caught it — the same
  off-by-one the envelope's position stamp turns on.
- Part (c), the loop verbs, is split to os-7e197768 for
  reviewability; item 5 stays one obligation with two landings.

## 2026-09-01 — loop verbs (os-7e197768, plan #172)

- The verbs are not sugar over `seed ledger append`, and the
  distinction is structural rather than stylistic: the raw seam
  consults the admission boundary NOT AT ALL, so a lane acting
  through it learns its act was illegal from a chain-level refusal
  after signing. Every loop verb runs the same `admit.Check`
  admission enforces before anything is signed into the chain, and
  renders the boundary's own error beside the caller's affordances.
  That is Phase 8's one-rule-set principle carried from legality to
  construction.
- An argument the system can compute is never asked for, because a
  value the boundary would refuse is not a choice being offered. The
  fence comes from the active window, the reservation from the shared
  budget view, the plan anchor from the approval (an approval admits
  ONE exact revision, so no other citation could be legal), and the
  resume range from the repository. What stays caller-supplied is
  what is a judgment: `--amount`, `--actuals`, and the packet's prose.
- Three failure shapes, three owners, and the split is the design:
  a MISSING fact refuses in the CLI naming what would establish it; an
  AMBIGUITY refuses in the CLI naming the candidates, because two open
  reservations are a spend decision the lane owns and a silent choice
  would make it for them; an ABSENT WINDOW is not a derivation failure
  at all — the key is omitted and the boundary refuses with its own
  account of the state, which is better than anything a derivation
  could say.
- On the remote path the derivation and the pre-flight read one
  materialized remote tip. A fence or reservation read from a stale
  local copy would be wrong under exactly the contention that makes
  claiming online-only, so `openRemoteSession` was extracted from
  `runLedgerAppendRemote` to share that single view rather than
  reconstruct it per verb.
- `claim take` refuses `--ledger` with the raw seam's own account of
  online-only claiming, extracted to one function: a lane must never
  meet two explanations of one rule.
- Derivation refusals are journaled like admission refusals. A lane
  that could not act is exactly the affordance gap the metric
  measures, and `by_code` keeps `usage` and `not_found`
  distinguishable from the boundary's codes.
- Remote successes carry no affordances and journal nothing: there is
  no local ledger to reopen at the landed tip or journal beside. That
  is the standing shape of every remote append, recorded as a
  deliberate absence rather than left implicit; giving the remote
  posture its own journal home is a client-state decision no card has
  made yet.

## 2026-09-01 — agent-ergonomics obligations for Phase 9 lanes (os-68ea0b2d, plan #174)

- A surface with no obligated consumer is a surface that may never be
  used. Item 5 landed the lane-facing surface; item 1 never said the
  lanes must USE it, so a worker written against the raw append seam,
  waking on a trusted event stream, emitting remembered heartbeats and
  retrying blind would have satisfied every word of item 1 and failed
  promotion criterion 1 on inspection rather than on a check. Each
  obligation is stated so a fixture or a validation check can fail on
  it, and each is a new consumer of an existing table rather than a
  new authority.
- Seed has no lease, so the card's "lease renewal rides every
  holder-signed verb" does not transfer as written. The transferable
  obligation is sharper: the observations liveness is classified from
  are emitted by the loop's own steps, so a working lane is a live
  lane by construction and an expired classification is true
  information rather than forgotten bookkeeping.
- That obligation is enforced by VOCABULARY, not by detection. The
  first draft asked for a lint flagging claim windows whose only
  observations are non-advancing; observations.md already classifies
  that stream as live then wedged, and a legitimate long-running step
  emits exactly it, so with no heartbeat discriminator the lint would
  either flag valid work or never fire. Withdrawn: the loop's verb set
  contains nothing whose only purpose is liveness, which a role
  fragment and a fixture can both be checked against.
- Item 5(b) is recorded PARTIALLY MET rather than the single-read
  contract being relaxed. The read is specified to carry unread
  messages and does not, so a lane told to orient from one read
  cannot learn it has mail without either a second read or trusting
  the pushed message, and the one-inbox doctrine forbids the latter.
  The remainder is named in item 5's own text, on the #157/#169
  precedent that a routing binds only where the phasing authority
  says it.
- "Unread" needs no stored read-state and gets no `message.read`
  verb: the position a lane carries forward IS its read cursor, so
  the messages new to it are those arisen after the cited `--since`
  position, which is the identity `--since` already has for
  obligations. The surface gains a section, not a concept.

- Two over-claims corrected on review, both about treating a signal as
  stronger than it is. (1) "Absence of observation is absence of work"
  is exactly the inference the lossy declaration forbids: the channel
  is ephemeral and lossy by charter §II.3, a dropped stream and dead
  work are indistinguishable from outside it, and `no_data` has no
  reap path at all. What loop-emitted observations buy is the removal
  of forgotten bookkeeping as a CAUSE of silence, which makes the
  classification better evidence without making it proof; the reap
  stays a judgment needing corroboration. (2) One-retry convergence as
  "admit or escalate" rejects correct behavior: in fleet mode a claim
  race means the loser re-orients and takes different work, and a
  fence invalidated by a concurrent reap says the same. The third arm
  is a refreshed read showing the act is no longer owed, and it is the
  common case rather than a loophole. What stays forbidden is the
  fourth outcome: a blind retry or a silent loop.
