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

Candidate actors derive from the chain itself (the genesis payload's
governance roots plus every enrollment subject, first-appearance
order); every actor-bearing view keys off that one derivation. The v0
**work classifier** is a prefix rule: verbs outside the `system.*` and
`actor.*` governance namespaces are work vocabulary, keyed by subject;
Phase 5's transition table replaces the rule with explicit vocabulary.

- **`roster`** (`roster.json`): every keyring entry — genesis
  governance roots included, appearing with `root: true` and empty
  kind/name (they are seeded from the genesis payload, not enrolled) —
  plus every enrolled actor with kind, name, standing, root flag, and
  accumulated grants, in first-appearance chain order.
- **`contracts`** (`contracts.json`): every work subject in
  first-appearance order — `{subject, first_position, last_position,
  events: [{position, verb, actor, payload}]}` with the signer
  fingerprint as `actor` and the payload's JSON content unchanged
  (the view re-indents for readability; canonical bytes live only in
  the ledger). No
  `state` field exists yet: state derivation arrives with Phase 5's
  transition table rather than a field that would always read null.
  One file, not per-subject files (subjects are opaque strings; the
  cache is the lookup-throughput surface). An empty chain yields an
  empty array, not a missing file.
- **`queue`** (`queue.json`): the claimable-work surface, schema fixed
  now — `{schema_version: "1", derivation, ready: […]}`, entries
  carrying at least `{subject, since_position}` (the field set is
  Phase 5's to extend). The v0 `derivation` is `"none"`: the Phase 4
  vocabulary defines no claimable states, so `ready` is empty **by
  definition** and the marker says so machine-readably — a consumer
  MUST NOT treat an underived queue as meaning "nothing to do".
  Phase 5 item 1 replaces the derivation and its marker; the
  eligibility filter follows later, per the build plan.
- **`actors`** (`actors.json`): the per-actor drill-down — the roster
  fields plus `standing_history` (each `actor.*` event on the subject:
  position, verb, acting signer) and `signed` (position, verb, subject
  of every record the fingerprint signed). The attribution surface: a
  revoked key keeps its full history here while the roster shows the
  ended standing.
- **`report`** (`report.json`): the operational summary — `chain`
  (position, tip, active version), `actors` (counts by standing,
  roots, total), `halt` (halted flag; declaring position and actor
  while halted), `checkpoints` (count; last position when any exist),
  `contracts` (subject and event counts). Sections needing Phase 5+
  facts (claims, offers, budgets, expiry-vs-wedge, divergence) are
  extension points named here, not emitted empty.

The SQLite cache lands in 4.3; the write-boundary lint in 4.4.
Registration is data: later phases append.

## The consumer verb and staleness

`seed project current --name <projection> [--out <dir>]
[--min-position <n>]` resolves `CURRENT`, reads the build's stamp,
and reports `{name, position, tip, version, path}`. Two position
conventions coexist and are both normative: the **stamp** (and this
verb's envelope) carries the verified record **count**; the **rebuild
envelope** stamps the tip's zero-based **index** (count-1), the
CLI-wide tip convention. Consumers demanding freshness pass
`--min-position`: a stamp below the demand refuses with exit 15
`stale`, naming the stamped and demanded positions — charter III.D's
"consumers can demand a minimum position", made scriptable. The stale
refusal is computed at a verified stamp, so its envelope carries that
stamp's position like any post-ledger response (`spec/envelope.md`):
machine consumers detecting staleness read the observed position
structurally, not out of the message text. Only **registered**
projections resolve: a name outside the registry refuses exit 4
`not_found` whatever directories exist under the output root (which
also keeps traversal components out of the path), and a registered
name with nothing published refuses 4 the same way — absence meaning
the projection's own directory does not exist; a published layout
that exists but cannot be resolved — a missing, unreadable, or empty
`CURRENT` (publication swaps `CURRENT` atomically, so a layout
without its pointer is a damaged publication, not an unpublished
one), an unreadable, unparseable, or incomplete stamp (wrong name,
empty version, a tip inconsistent with its position) — refuses
exit 5 `unavailable`, an operational failure, never mistaken for an
unpublished projection. The verb takes no `--ledger` flag: it is
structurally a consumer and cannot touch authoritative state; no
refusal creates or modifies anything.

## Write boundary

III.D's "no code path writes a projection directly" is enforced in
three layers; the boundary is **code-path discipline**, which is what
that row claims — not tamper-proofing against a root-privileged actor.

1. **The vocabulary lint (Lint A).** No non-test Go file outside
   `next/internal/project` may contain the publication vocabulary
   literals (exactly `"CURRENT"`, `"projection.json"`, `"builds"`):
   nobody constructs projection paths by hand. Test files are exempt
   (drills read the layout to assert it; reading is not a violation).
2. **Seam/write separation (Lint B).** A non-test file outside the
   engine that imports the engine must contain no `os` write-family
   calls (`WriteFile`, `Create`, `OpenFile`, `Rename`, `Remove`,
   `RemoveAll`, `Mkdir`, `MkdirAll`, `Chmod`, `Truncate`, `Link`,
   `Symlink`): the file that can obtain a published path (the engine
   returns real paths by design; views exist to be found and read)
   is a file that cannot write one. Both lints live in one
   `go/parser` test in the engine's own suite — a test in the suite
   is wired into `check-next` by construction — and both are
   self-checked against planted fixtures, so a detector that fails to
   fire is itself a test failure.
3. **Locked publication.** Published trees carry `0444` files and
   `0555` directories — the projection root **and the output root
   itself** included, since rename permission lives in the parent and
   a writable parent would let a whole projection root be renamed
   away — so rename-over, unlink-plus-recreate, in-tree creation,
   `CURRENT` repointing, and root renames all fail at the operating
   system for every non-engine code path, however the path was
   obtained. The engine opens a write window (`0755`) on exactly the
   output root, the projection root, and `builds/` for its own swap
   (after verification, keeping refuse-before-write intact) and every
   return path relocks, failed publications included; every published
   mode is set by explicit `chmod`, so the process umask cannot
   weaken the protocol. Only a killed process leaves an open window —
   at worst writable directories and an orphan partial, never a
   broken view — and the next rebuild relocks everything.

**Deletion.** With directories read-only, a bare `rm -rf` needs the
mode walk first (`chmod 0755` every directory under the output root,
then remove); `seed project rebuild` runs the same walk itself, so
the sanctioned one-command recovery is unchanged: rebuild.

**Residual risk, named.** The lints bind single files: deliberately
splitting seam access and writes across files, or writing through
`syscall` directly, evades them — the locked trees stop the former,
root renames included. File modes stop no process that may `chmod`
(uid 0 bypasses permission checks entirely), so mode-refusal drills
require an unprivileged runner, which CI provides; and the output
root's own parent belongs to the invoker's filesystem, so renaming
the output root itself is outside the engine's ownership — it is
equivalent to repointing `--out`, and consumers that name the root
by configuration are unaffected by what modes cannot reach.
Authority safety does not rest on any of this: projections stay
non-authoritative, stamped, and rebuildable whatever happens to the
files.

## Conformance mapping

- III.D "every read surface is a deterministic function of a ledger
  prefix … stamped with its build position, and rebuildable
  byte-identically with one command" — the engine + `seed project
  rebuild`, drilled in `internal/project`.
- III.D "no code path writes a projection directly …; the
  write-boundary lint enforces it" — the three layers above ("Write
  boundary"), implemented as code-path discipline: both lints
  self-checked and green over the tree, locked publication drilled
  with refused replacement operations and bytes-unchanged assertions.
- III.D "Staleness is visible everywhere projected state is shown;
  consumers can demand a minimum position" — every view is stamped and
  `seed project current --min-position` refuses stale builds with exit
  15, drilled end-to-end.
- III.D cache row — 4.3; mirrors, observation inputs, and the CI
  rebuild-everything drill — with their phases.
