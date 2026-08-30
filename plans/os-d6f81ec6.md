# Plan: next Phase 1.6 — payload data classification lint + hostile corpus (os-d6f81ec6)

Implements [`docs/next-build-plan.md`](../docs/next-build-plan.md) Phase 1
item 6: the references-not-bodies lint over event payloads, with a hostile
fixture corpus. Design authority: [`SEED-NEXT.md`](../SEED-NEXT.md) Part II
§1 ("Data classification"): payloads carry coordination facts and
references (hashes, paths, short structured fields), never content bodies;
bulk or expirable content belongs in the artifact store by hash.
Conformance: III.A — "Payload data classification is enforced at admission:
coordination facts and references only; … the hostile-payload lint corpus
passes." Phase 1 delivers the lint as a pure library plus corpus; the
admission rule set (`internal/admit`, Phase 2) imports it unchanged, the
same pattern the build plan sets for halt (1.5).

## Steps

1. **Rules as data, embedded.** `next/internal/classify`: `Lint(payload)
   []Violation` with typed violations, driven by a rule table embedded in
   the package via `go:embed` (`next/internal/classify/rules.json`) so the
   compiled admission binary needs no runtime file IO or working-directory
   assumptions; `next/spec/classify.json` stays the normative, reviewable
   copy and a byte-identity sync test fails `check-next` on drift (the
   repo's fan-out pattern). The v0 bounds: maximum canonical payload size
   (4096 bytes), maximum string field length (512 bytes) with an exemption
   for reference shapes (a 64-lowercase-hex hash, or a `path @ commit`/
   ref-range anchor per the packet grammar), an **aggregate free-text
   budget** (the byte sum of all non-reference string values in one
   payload, 1024 bytes) so a body chunked into many small strings cannot
   pass item-level caps, maximum nesting depth (8), maximum array length
   (64), and a base64-blob heuristic (a contiguous base64 run over 256
   bytes fails as an embedded body). Bounds are data so a later recorded
   decision tunes the table, not the code.
2. **Violations name their remedy.** Each violation cites the offending
   JSON pointer, the rule, and the artifact-store alternative ("store the
   body, reference its hash"), so refusals teach the affordance instead of
   just refusing (charter §I.3 principle 7).
3. **Hostile corpus.** `next/internal/classify/testdata/hostile/*.json`:
   payloads that MUST refuse — embedded long prose (a transcript-shaped
   field), a **chunked body** (prose split across many strings that all
   pass the per-item caps but bust the aggregate free-text budget), an
   oversized base64 blob, a nesting bomb, a huge array, an oversized
   total payload, a long string smuggled inside a nested object —
   and `testdata/benign/*.json`: payloads that MUST pass — typical
   coordination facts (short titles, 64-hex hashes, commit-anchored refs,
   small structured fields). A corpus-driven test asserts every hostile
   fixture refuses with the expected rule and every benign fixture passes.
4. **Wiring seam.** The lint exposes exactly the signature the Phase 2
   admission rule set will call (pure function over the payload bytes, no
   IO); a doc comment in the package names that contract, and a test proves
   determinism (same input, same violations, order-stable).

Intra-phase ordering note: the build plan's "within a phase, items are
ordered" is read here as dependency order; 1.6's technical dependency is
1.1 (event payload shapes) and the card's `dep:` edge records it. Work
merged from intervening items (1.2's ledger and onward) is protected by
the retention set below, which covers every `next/internal` package
present on main at implementation time.

## File Scope

- `next/internal/classify/**` (new package, embedded `rules.json`,
  `testdata/` corpus)
- `next/spec/classify.json` (the normative v0 rule table; byte-identical
  to the embedded copy, sync-tested)
- `next/docs/decisions.md`, `next/docs/progress.md` (bounds decision;
  frontier move)

## Acceptance Criteria

**Boundary set (new, shown working):**

- `Lint` refuses every hostile-corpus fixture with the expected rule and
  JSON pointer — the chunked-body fixture included — and benign fixtures
  pass with zero violations.
- Bounds live in the embedded rule table, byte-identical to
  `next/spec/classify.json` (sync test); the package contains no
  hand-written bound constants (schema test + a test that edits a bound in
  a temp copy and observes the changed verdict).
- Violations are deterministic and order-stable across runs.

**Retention set (existing, shown unharmed):**

- Every `next/internal` package present on main at implementation time,
  and the Phase 0 CLI, pass unchanged
  (`cd next && go test ./internal/... ./cmd/... -count=1`).
- The repo-wide gate stays green with the coverage gate
  (`make check`, ≥90% on `next/internal/...`).

## Validation Commands

- `make check-next`
- `make check`
- `cd next && go test ./internal/classify/... -count=1`
- `cd next && go test ./internal/... ./cmd/... -count=1`
