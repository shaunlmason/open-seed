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
