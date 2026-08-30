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

1. **Rules as data.** `next/internal/classify`: `Lint(payload) []Violation`
   with typed violations, driven by a checked-in rule table
   (`next/spec/classify.json`, loaded at init and validated by a schema
   test) rather than hand-written conditionals, holding the v0 bounds:
   maximum canonical payload size (4096 bytes), maximum string field length
   (512 bytes) with an exemption for reference shapes (a 64-lowercase-hex
   hash, or a `path @ commit`/ref-range anchor per the packet grammar),
   maximum nesting depth (8), maximum array length (64), and a
   base64-blob heuristic (a contiguous base64 run over 256 bytes fails as
   an embedded body). Bounds are data so a later recorded decision tunes
   the table, not the code.
2. **Violations name their remedy.** Each violation cites the offending
   JSON pointer, the rule, and the artifact-store alternative ("store the
   body, reference its hash"), so refusals teach the affordance instead of
   just refusing (charter §I.3 principle 7).
3. **Hostile corpus.** `next/internal/classify/testdata/hostile/*.json`:
   payloads that MUST refuse — embedded long prose (a transcript-shaped
   field), an oversized base64 blob, a nesting bomb, a huge array, an
   oversized total payload, a long string smuggled inside a nested object —
   and `testdata/benign/*.json`: payloads that MUST pass — typical
   coordination facts (short titles, 64-hex hashes, commit-anchored refs,
   small structured fields). A corpus-driven test asserts every hostile
   fixture refuses with the expected rule and every benign fixture passes.
4. **Wiring seam.** The lint exposes exactly the signature the Phase 2
   admission rule set will call (pure function over the payload bytes, no
   IO); a doc comment in the package names that contract, and a test proves
   determinism (same input, same violations, order-stable).

## File Scope

- `next/internal/classify/**` (new package + `testdata/` corpus)
- `next/spec/classify.json` (the v0 rule table)
- `next/docs/decisions.md`, `next/docs/progress.md` (bounds decision;
  frontier move)

## Acceptance Criteria

**Boundary set (new, shown working):**

- `Lint` refuses every hostile-corpus fixture with the expected rule and
  JSON pointer; benign fixtures pass with zero violations.
- Bounds live in `next/spec/classify.json`; the package contains no
  hand-written bound constants (schema test + a test that edits a bound in
  a temp copy and observes the changed verdict).
- Violations are deterministic and order-stable across runs.

**Retention set (existing, shown unharmed):**

- Phase 1.1 event suites and Phase 0 CLI/envelope suites pass unchanged
  (`cd next && go test ./internal/event/... ./internal/envelope/... ./cmd/... -count=1`).
- The repo-wide gate stays green with the coverage gate
  (`make check`, ≥90% on `next/internal/...`).

## Validation Commands

- `make check-next`
- `make check`
- `cd next && go test ./internal/classify/... -count=1`
- `cd next && go test ./internal/event/... ./internal/envelope/... ./cmd/... -count=1`
