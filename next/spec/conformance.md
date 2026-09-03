# Conformance: Part III as a table

Status: normative for `next/**` (plans/os-83bc3d84.md; build plan Phase
13's exit line: "the conformance report shows Part III complete at the
enforced self-hosted posture"; charter III.Q row 3). Last verified:
Phase 13.

The charter's Part III is the list of what conformance means: eighteen
lettered pillars, A through R, of checkbox rows. The phase exit records
in `next/docs/progress.md` walk those rows, phase by phase, and say for
each whether the tree meets it and by what drill. This document names
the table that carries those verdicts in machine form, the rendering
that publishes it, and the doctor's sentence about it. The table is a
transcription of the records, never a judgment of its own: a row's
status changes when an exit record says so, and the record is the
evidence the row cites.

## The table

`next/spec/conformance.json` is the strict object `{"pillars":
[{"id", "title", "rows": [{"row", "text", "posture", "status",
"phase", "evidence", "note"}]}]}`:

- `id` and `title` are the charter's lettered heading; `row` counts
  from 1 within the pillar; `text` is the charter's row, verbatim
  after whitespace normalization (continuation lines joined by one
  space).
- `posture` is `any` or `enforced-only`, read off the row's
  `(*enforced-only*)` marker: an enforced-only row is judged at the
  enforced postures alone.
- `status` is exactly one of `met`, `partial`, `routed`, `open`. A
  `met` row names its `evidence` (pull requests and test names, as the
  record cites them); a `partial` or `routed` row's `note` names where
  the rest lives (a phase, an item, the backlog, promotion); an `open`
  row is one no record has met, and its note may say which item lands
  it.
- `phase` is the phase whose exit record judged the row, as its
  number; empty where none has.

`internal/conformance` parses Part III out of `SEED-NEXT.md` itself
(the `### <letter>. <title>` headings, the `- [ ]` rows and their
six-space continuation lines) and holds the table to it in both
directions: the same pillars in the same order with the same titles,
the same rows with the same text and posture. A row the charter has
that the table lacks, a row the table has that the charter lacks, a
reworded row, a posture that is not the charter's, a status outside
the vocabulary, a `met` without evidence or a `partial` or `routed`
without a note each refuse by name, and the generator refuses to
render a table that does not hold. III.R, the autonomy end-state, is
`open` throughout: its rows are outcomes promotion measures (build
plan §5), not mechanisms a phase lands, and each row's note says so.

## The rendering

`seed docs generate` writes `next/docs/generated/conformance.md` from
the table: a summary of every pillar's rows by status, then each
pillar's rows with status, phase, the charter's text, the evidence and
the note. `seed docs check` regenerates and diffs it like the other
governed documents ([`simulation.md`](simulation.md), "docs"), failing
`docs_drift` under `make check-next` when the committed file no longer
matches the table. The rendering carries no date and no clock: the
table is the source and the exit records are the evidence.

## The doctor

`seed doctor` reports `conformance` whenever the source tree it reads
(`--repo`, default the working directory) carries the table:

```json
"conformance": {"counts": {"met": 0, "partial": 0, "routed": 0, "open": 0},
                "open_rows": [{"id": "N.2", "phase": "", "posture": "any", "note": "Phase 13 item 3"}],
                "not_applicable_here": ["B.1", "B.4", "B.5"],
                "complete": false,
                "because": "..."}
```

The counts are over the rows judged at the declared posture; at the
cooperative posture the enforced-only rows are set aside as
`not_applicable_here` and Part III is not complete there, whatever the
rest says; at an enforced posture `complete` is true exactly when no
row is `open`, `partial` and `routed` rows carrying their notes. The
`open_rows` list is what the build plan's Phase 13 preamble asks for:
which criteria remain open, by pillar and row, with the phase that
last judged them. A table that does not hold against the charter is an
operational failure (`unavailable`, exit 5), never a silently absent
section; a tree without the table has no section.

## Conformance mapping

- III.Q row 3 "docs governed: operator handbook, generated worker docs,
  stamped design docs": the table rendered by the same generator and
  gated by the same drift check as the lifecycle, capability, exit-code
  and lane documents.
- Build plan Phase 13, exit line and preamble: `seed doctor`'s
  `complete` and `open_rows` at the declared posture, and the Phase 13
  exit record flipping the rows it meets with itself as the evidence.
