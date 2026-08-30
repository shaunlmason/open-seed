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
- 2026-08-30 — v1-loop delegation for `next:` cards: operator queue verbs
  (`promote`; `plan-unblock` once the gate PR is genuinely merged) run under
  the session principal (`shaunlmason`), work verbs under
  `seed-next-implementer`; implementation may be prepared ahead of the plan
  merge as a draft PR when the owner is merging in batches (CI's
  plan-at-merge-base gate still orders the merges). ADR:
  `decisions/0003-next-loop-delegation.md`. (PR #72)
