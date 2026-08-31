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
