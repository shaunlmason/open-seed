# projections.md — the projection engine

> Status: v0, normative for `next/**`. Authority: [`SEED-NEXT.md`](../../SEED-NEXT.md)
> Part II "Projections"; [`docs/next-build-plan.md`](../../docs/next-build-plan.md)
> Phase 4; plan `plans/os-4d5cacff.md`. Implemented by
> `next/internal/project` and the `seed project rebuild` verb.

## The five properties (normative)

Every projection is **derived** (never written directly), **stamped**
(with the ledger position it was built at), **rebuildable** (deletion
loses nothing; the rebuilt tree is byte-identical), **read-only
outward** (a build never writes the ledger), and **non-authoritative**
(no decision may prefer a projection over the ledger; staleness is
visible, never silent).

## The engine contract

- **Input**: the verified record prefix. The engine opens the ledger
  with the read-only open (it is structurally incapable of the healing
  writes the ordinary open performs) and verifies from genesis; a
  verification failure — a stale `HEAD` included — refuses the build
  before anything is written.
- **Determinism**: a projection is a pure function of the records;
  outputs are written in sorted order with no timestamps, so one prefix
  always produces one byte-identical tree.
- **Path safety**: the ledger directory and the output root are
  canonicalized, and any overlap in either direction refuses as a
  usage error before anything is created — a projection target can
  never coincide with authoritative state.

## Publication: immutable builds plus a pointer

A directory rename cannot atomically replace a non-empty directory, and
delete-then-rename opens the very window atomicity forbids, so
publication is versioned:

```
<out>/<name>/CURRENT                                 the active build id (atomically renamed into place)
<out>/<name>/builds/<position>-<tip12>-v<version>/   immutable, complete build trees
```

The build id derives from the stamp — the verified position and tip
**and the projection's derivation version** — so one prefix under one
derivation reproduces one identical id and tree, `CURRENT` included,
which keeps the byte-identical rebuild drill meaningful; and a
projection whose build logic changes (Phase 5 replacing the queue's
derivation, say) bumps its version and republishes under a new id
instead of being discarded as a same-id duplicate. The pointer is
swapped only once the tree is complete. After the swap, the build the
pointer named immediately before is **retained** — a reader that
resolved `CURRENT` just before the swap must find a complete tree at
the path it holds — while everything older, and stray partials,
prune; a reader that loses the race to two consecutive swaps
re-resolves `CURRENT`. A killed build leaves at worst an orphan
directory.

## The stamp

Every build tree carries `projection.json`:

```json
{"name": "<projection>", "position": <verified record count>, "tip": "<chain tip hash>", "version": "<derivation version>"}
```

The stamp is the staleness surface: consumers read it, may demand a
minimum position, and never treat the view as authoritative. The
`seed project rebuild` envelope reports the same values per projection.

## Registered projections

- **`roster`** (`roster.json`): every keyring entry — genesis
  governance roots included, appearing with `root: true` and empty
  kind/name (they are seeded from the genesis payload, not enrolled) —
  plus every enrolled actor with kind, name, standing, root flag, and
  accumulated grants, in first-appearance chain order.

The standard work projections (contract detail, ready queue, actor
view, report) land in Phase 4.2; the SQLite cache in 4.3; the
write-boundary lint in 4.4. Registration is data: later phases append.

## Conformance mapping

- III.D "every read surface is a deterministic function of a ledger
  prefix … stamped with its build position, and rebuildable
  byte-identically with one command" — the engine + `seed project
  rebuild`, drilled in `internal/project`.
- III.D "no code path writes a projection directly …; the
  write-boundary lint enforces it" — the lint lands in 4.4; the engine
  is the only writer this spec admits.
- III.D cache row — 4.3; mirrors, observation inputs, and the CI
  rebuild-everything drill — with their phases.
