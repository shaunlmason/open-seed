# Plan: next Phase 8.3 — refusal-rate metric in the report (os-edf73d66)

Implements `docs/next-build-plan.md` Phase 8 item 3: "Refusal-rate
metric in the report." Design authority: SEED-NEXT.md §II.10
("Refusal rates are tracked — a rising rate signals an affordance
gap, not agent error") and charter III.I row 4 ("Refusal rates are
tracked as an affordance-gap metric"). The constraint that shapes
everything: refusals never reach the chain — admission refuses
before anything is written — so the metric needs a recorded source,
and the report is a byte-identical, position-identified projection
that must not read wall-clock state. The tree already holds the
pattern for exactly this: the observation section, where locally
gathered facts enter the build as **declared inputs** (digest-
covered, echoed in the output, section null on input-free builds;
`next/spec/observations.md`).

## Design decisions (binding for this task)

- **D1 — refusals are journaled at the refusing seams, locally,
  outside the chain.** New `internal/refusals`: an append-only
  JSONL journal, one line per admission-boundary refusal, shape
  `{"ts", "position", "actor", "verb", "subject", "code"}` — ts the
  refusing process's RFC3339 UTC instant, position the tip-ordinal
  stamp the refusal envelope carried, actor the refused signer's
  fingerprint, code the envelope's machine code (`fenced`,
  `chain_invalid`, …). `refusals.Note(dir, entry)` appends
  best-effort: journaling must never fail or slow the verb, so
  every error is swallowed, exactly the stamping posture. The
  journal lives beside the ledger as `refusals.jsonl` in the ledger
  directory — operator-local telemetry, never synced by gitref,
  never part of verification. The seams are the commands that
  already hold a ledger directory and render refusal envelopes with
  affordances (the 8.1 stamp list): ledger append's preview and
  chain-invalid refusals, offer publish, verdict render, seal
  create, and budget reserve/settle/release's admission refusals.
  The remote server-side boundary is named an extension point in
  the spec, not wired: the report builds locally, and the local
  journal is the local truth.
- **D2 — the journal enters the report as a declared input, the
  observations pattern verbatim.** `project.Inputs` grows the
  parsed journal; the declared-inputs `Digest` covers it (the doc
  comment's own rule: the digest spans EVERY declared input).
  `seed project build`/`rebuild` gain `--refusals <file>`, usable
  with or without `--obs`. The report gains a nullable `refusals`
  section: null on builds that declare no journal (absence of data
  stated, never fabricated), else `{inputs: {digest, entries},
  refused, admitted, by_code, by_verb, span: {from, to}, rate}` —
  `refused` the journal lines inside the span, `admitted` the chain
  records with positions in `[from, to]`, `span` the journal's
  min/max positions, `by_code`/`by_verb` refusal counts, and `rate`
  refused/(refused+admitted) rendered as a fixed four-decimal
  string (deterministic bytes; consumers needing precision divide
  the integers themselves). Malformed journal lines fail the build
  with a usage-shaped refusal, mirroring how a bad observation
  snapshot refuses: inputs are declared, so garbage is the
  declarer's error, never silently skipped telemetry.
- **D3 — versioning per the projection discipline.** Report
  version "9" → "10" (new nullable section; version, not content,
  republishes input-free prefixes), cache generation bumped with a
  `refusals` key in the report KV table written only when the
  section is non-null (the reconciliation/observation precedent).
  Spec: a short `next/spec/refusals.md` (journal wire shape, seam
  list, the input declaration, the section shape, the rate
  definition, the remote extension point), referenced from
  `projections.md`'s report bullet and from `envelope.md`'s
  refusal-code table note. No envelope schema change: the metric
  reads codes the envelope already carries.
- **D4 — drills.** (a) Journal unit tests: Note appends and
  swallows errors (unwritable dir journals nothing and returns
  nothing). (b) A CLI seam drill: a refused append journals one
  line whose code and position match the rendered envelope; a
  successful append journals nothing. (c) A report drill: build
  with a fixture journal asserts the section's counts, span, and
  rate against hand-counted chain records, byte-identical across
  two builds; an input-free build keeps the section null and the
  version bump alone republishes. (d) The digest drill: two builds
  differing only in the journal digest differ, per the Digest rule.
- **D5 — scope guard.** No aggregation service, no rotation, no
  thresholds, no trend windows: v0 is counts, span, and one rate,
  locally journaled and declared. A rising-rate alarm, promotion
  into packets, and remote-boundary journaling are later phases'
  work (§II.10's flywheel reading arrives with the curator). If a
  seam turns out not to render admission refusals at all (nothing
  to journal), it is left unwired and the spec's seam list names
  what ships, recorded in decisions.md.

## Steps

1. `internal/refusals`: journal writer + reader + tests.
2. Wire `refusals.Note` at the D1 seams in `cmd/seed`, beside the
   existing refusal-envelope construction.
3. `project.Inputs` + `Digest` + report section + version bumps +
   cache key; `--refusals` flag on build/rebuild.
4. `next/spec/refusals.md`; touch `projections.md` and
   `envelope.md` references.
5. Drills per D4; `make check`; docs (progress frontier, decisions
   entry); receipt; evidence; review.

## File Scope

- `next/internal/refusals/` (new)
- `next/cmd/seed/` (seam wiring, project flags, drills)
- `next/internal/project/` (Inputs, Digest, report, cache, drills)
- `next/spec/refusals.md` (new), `next/spec/projections.md`,
  `next/spec/envelope.md` (reference sentences)
- `next/docs/progress.md`, `next/docs/decisions.md`

## Acceptance Criteria

- A refused verb at every wired seam journals one well-formed line;
  success paths journal nothing; journaling failures never surface.
- `seed project rebuild --refusals` emits the section with correct
  counts, span, and rate; input-free builds keep it null; the
  declared-inputs digest covers the journal; builds are
  byte-identical for identical inputs.
- Report "10", cache generation bumped, specs updated.
- `make check` green, coverage gate ≥90% held.

## Validation Commands

```sh
cd next && gofmt -l . && go vet ./... && go build ./...
cd next && go test ./internal/refusals/ ./internal/project/ ./cmd/seed/ -count=1
make check
```

## Expected diff shape

One new package, one new spec file, seam-local edits in cmd/seed,
the report/cache/version block in internal/project, and the docs
pair. Roughly +700/-40 lines, all under `next/**`.
