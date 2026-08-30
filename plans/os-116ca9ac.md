# Plan: next Phase 0 — workspace and spec skeleton (os-116ca9ac)

Implements [`docs/next-build-plan.md`](../docs/next-build-plan.md) Phase 0
(items 1–3, one coherent cluster: the module skeleton, the spec documents, and
the tracking docs are interlocked and share one exit criterion). Design
authority: [`SEED-NEXT.md`](../SEED-NEXT.md) Part II §1 (event structure),
§10 (envelopes), Appendix B (wire-level sketch). Conformance groundwork for
Part III.A (canonical form, ordering, protocol version) and III.I (versioned
envelopes, exit codes) — the enforcing tests land in Phases 1+; this phase
lays the spec text and CI plumbing they cite. Every fixed default exercised
here is logged in `next/docs/decisions.md` (created by this task).

## Steps

1. **Module scaffold.** `next/go.mod` (module `github.com/shaunlmason/open-seed/next`,
   `go 1.25.0` to match the engine toolchain); `next/cmd/seed/main.go` with a
   testable `run()` seam; `seed version` emits a v0 affordance envelope
   (JSON: `v`, `ok`, `result{name, version, protocol}`, `error`, `position`,
   `affordances`, `budget`, `exit`) and exits 0; unknown or missing verbs exit
   64 (usage) with a structured error envelope. `next/internal/version/`
   holds the Name/Version/Protocol constants. CLI-level tests drive `run()`
   (version happy path, unknown verb, envelope keys). `next/.gitignore`
   ignores `bin/`, `var/`, and coverage artifacts. Binary is built to
   `next/bin/` only, never installed on PATH (incubation rule).
2. **CI wiring (the named v1 integration point).** `Makefile` gains
   `check-next`: inside `next/` run gofmt (must list no files), `go vet ./...`,
   `go build ./...`, `go test ./...` with a coverage profile over
   `./internal/...`, and fail if total statement coverage < 90%. `check`
   depends on `validate` + `check-next`. `check-next` self-skips with an
   explicit message when `next/` is absent (template instantiations and
   flavor tests must keep working). No other v1 surface is touched.
3. **Spec documents.** `next/spec/protocol.md` v0: canonical event form (JCS,
   RFC 8785), event fields (`v`, `ts`, `actor`, `verb`, `subject`, `payload`,
   `prev`; `sig` carried alongside, computed over the JCS bytes including
   `prev`), hash and signature algorithms (SHA-256, Ed25519 accepting OpenSSH
   ed25519 keys), the empty-hash genesis rule, protocol version `seed/0` and
   its bump discipline, and the verb namespace catalog copied from charter
   Appendix B. `next/spec/envelope.md`: the envelope shape and the exit-code
   table (v1-inherited codes where semantics match: 0 ok, 2 contention,
   3 invalid transition, 4 not found, 5 backend unavailable, 6 fenced,
   10 version mismatch, 64 usage; new-code allocation rule documented).
4. **Tracking docs.** `next/docs/decisions.md`: the implementation decision
   log, one line per default exercised or overridden, linking the PR; seeded
   with the decisions this phase exercises. `next/docs/progress.md`: the
   frontier file per build-plan §4 (`phase.item — card id — PR — state`, one
   line per plan item), initialized with the Phase 0 rows and the actor
   roster in use, written so a fresh agent can resume from it alone.

## File Scope

- `next/**` — all new files: `go.mod`, `.gitignore`, `cmd/seed/`,
  `internal/version/`, `spec/`, `docs/`.
- `Makefile` — add `check-next` and wire it into `check`; this is the
  integration point `docs/next-build-plan.md` §0 names. Nothing else outside
  `next/**`.

## Acceptance Criteria

- `make check` is green with `check-next` included; `check-next` enforces
  gofmt, vet, build, tests, and the ≥90% statement-coverage gate on
  `next/internal/...`.
- `cd next && go run ./cmd/seed version` prints a v0 envelope with
  `ok=true`, the binary name `seed`, a dev version, and protocol `seed/0`;
  an unknown verb exits 64 with an error envelope.
- `next/spec/protocol.md` matches charter Appendix B: every verb namespace
  (`system.*`, `actor.*`, `intent.*`/`contract.*`, `claim.*`, `plan.*`,
  `progress.*`, `submission.*`, `verdict.*`, `merge.*`/`check.*`,
  `budget.*`, `run.*`, `message.*`, `request.*`, `curation.*`) appears, and
  the canonical form and algorithms match the build plan's fixed defaults
  (JCS, Ed25519, SHA-256, `seed/0`).
- `next/spec/envelope.md` documents every envelope field from Appendix B and
  the exit-code inheritance from v1 (2 contention, 6 fence, 10 version).
- `next/docs/decisions.md` exists with one entry per default exercised;
  `next/docs/progress.md` exists and states the frontier truthfully
  (on merge: Phase 0 done, Phase 1 item 1 next).
- No v1 surface outside the named `Makefile` integration point is modified.

## Validation Commands

- `make check-next`
- `cd next && go run ./cmd/seed version`
- `test -f next/spec/protocol.md`
- `test -f next/spec/envelope.md`
- `test -f next/docs/decisions.md`
- `test -f next/docs/progress.md`
