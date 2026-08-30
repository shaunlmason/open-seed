# Plan: next Phase 4.3 — the SQLite cache projection (os-acc1ac78)

Implements [`docs/next-build-plan.md`](../docs/next-build-plan.md) Phase 4
item 3: "SQLite cache projection; mid-operation deletion drill", with the
plan's binding default `modernc.org/sqlite` (no cgo, portable builds).
Design authority: [`SEED-NEXT.md`](../SEED-NEXT.md) §II.2 (every cache is
a one-way, rebuildable, disposable projection), §II.4 ("a local database
cache for single-machine read throughput" among the **standard
projections**; "a local cache projection delivers machine-local speed
without a second authority"), the what-Seed-does-not-have row ("A
machine-local authoritative store — the cache projection provides the
throughput without a second authority"), and conformance III.D: row 1
binds **every** read surface — this cache included — to deterministic,
stamped, **byte-identical** rebuild, and the cache row adds "delivers
single-machine read throughput with zero authority (mid-operation
deletion loses nothing)" on top. Deps: the 4.1 engine and 4.2 views
(plans #105/#106); implementation stacks on their branches until they
merge.

## Design decisions (binding for this task)

- **The cache is a registered standard projection, byte-identical like
  every other.** §II.4 lists it among the standard projections and
  III.D row 1 binds every read surface byte-identically; the cache
  checklist row adds requirements, it does not narrow that one (review
  finding on #108 — an earlier draft read it otherwise). So the cache
  registers in `project.Default()` as projection `cache` with one view
  file, `cache.db`, and the 4.1 engine's machinery applies unchanged:
  immutable `builds/<position>-<tip12>/` trees, `CURRENT`, the
  `projection.json` stamp, pruning, 4.4's locked directories, the
  whole-output byte-identical drill, and `seed project current`
  resolution — no separate publication path, no exemptions.
- **Byte-determinism is engineered, then enforced by the existing
  drill.** SQLite output is a pure function of the operation sequence
  for a pinned library version when the variance sources are closed:
  the builder uses one connection, one transaction, rollback journal
  (never WAL — WAL headers embed salts), fixed `page_size`, no
  `auto_vacuum`, no `ANALYZE`, no `AUTOINCREMENT`, tables created and
  rows inserted in spec order. The registry's byte-identical drill
  then proves it on every fixture: two builds from one prefix, same
  bytes. A driver upgrade may change the encoding, which changes both
  builds together and keeps the drill green; the drill compares builds
  of one binary, and `go.sum` pins the driver. If real
  nondeterminism ever surfaces, the drill fails loudly and the answer
  is a builder fix or an escalated charter question — never a weaker
  drill.
- **Built from the records, not from the view files.** The cache
  builder is a `Builder` like the others: it consumes the verified
  record prefix, reuses the view derivations' intermediate state
  (roster entries, contract streams, keyring state) rather than
  parsing published JSON back in, builds the database in a private
  temp file, and returns its bytes. Coordination-scale ledgers make
  the in-memory hand-off cheap; the size posture is stated in the
  spec and a streaming publication seam is deliberately deferred
  until a real ledger outgrows it.
- **The consumer surface is SQL itself.** SQLite is the API: any
  driver or the `sqlite3` shell opens the published file read-only
  (the spec states the read-only contract and why a writable open is
  a programming error; 4.4's locked trees refuse in-place writes
  mechanically). Staleness stays visible twice over: the build tree's
  `projection.json` as everywhere, plus an in-database `stamp` table
  equal to it, so a pure-SQL consumer demands a minimum position with
  one documented query. No new verbs: `seed project rebuild` builds
  it, `seed project current --name cache` resolves it.
- **Throughput is the purpose, correctness is the test.** The drills
  assert indexed per-subject and per-actor lookups return exactly the
  view-equivalent facts; CI runs no benchmarks (timing assertions
  flake). The spec states this division honestly.

## Schema (v1, stated in the spec)

- `stamp(name TEXT, position INTEGER, tip TEXT, version TEXT)` — one
  row, written last in the build transaction, carrying **exactly the
  tree stamp's fields** so the equality drill and a pure-SQL consumer
  compare it to `projection.json` field-for-field (review finding on
  #110). The database's own schema generation is versioned separately
  via `PRAGMA user_version = 1`, part of the deterministic recipe, and
  bumps with the table set rather than sharing the stamp's row.
- `roster(fingerprint TEXT PRIMARY KEY, kind TEXT, name TEXT,
  standing TEXT, root INTEGER, grants TEXT)` — grants as a JSON
  array, matching `roster.json`.
- `contracts(subject TEXT, position INTEGER, verb TEXT, actor TEXT,
  payload TEXT)` with an index on `subject` — the per-subject lookup
  the 4.2 plan deferred here as "the lookup-throughput answer".
- `queue(subject TEXT, since_position INTEGER)` plus
  `queue_meta(schema_version TEXT, derivation TEXT)` — mirrors 4.2's
  queue including `derivation: "none"` at Phase 4; Phase 5 swaps the
  derivation in both serializations at once.
- `actor_history(fingerprint TEXT, position INTEGER, verb TEXT,
  acting TEXT)` and `actor_signed(fingerprint TEXT, position INTEGER,
  verb TEXT, subject TEXT)`, indexed on `fingerprint` — the actor
  view's two streams.
- `report(key TEXT PRIMARY KEY, value TEXT)` — the report skeleton's
  facts as JSON values.

## Steps

1. **The cache builder** (`next/internal/project/cache.go`): the
   `Builder` above — deterministic pragmas, one ordered transaction
   from the shared derivations, stamp row last, close, read the temp
   file's bytes, return `{"cache.db": bytes}`; the temp file lives in
   an engine-owned scratch directory and is removed on every path.
   Register `cache` in `Default()`. Add `modernc.org/sqlite` to
   `next/go.mod` pinned at the current stable release, recorded in
   `go.sum`.
2. **Drills** (library + CLI):
   - *Byte-identical, via the registry*: the engine's existing
     repeat-rebuild and delete-then-rebuild drills now cover the
     cache automatically; assert they run with `cache` present and
     that two independent builds of the same prefix produce identical
     `cache.db` bytes on every fixture (populated, root-only, halted,
     grown chain — growth changes bytes and advances the stamp).
   - *View equivalence*: every row set equals the published JSON view
     it mirrors, fixture by fixture (roster, contracts, queue with
     `derivation` `"none"`, actor streams, report facts); the `stamp`
     table equals `projection.json`.
   - *Mid-operation deletion loses nothing*: delete the cache build
     tree (mode walk, per 4.4) while a read connection holds
     `cache.db` open (POSIX keeps the inode; in-flight reads
     complete), and after close; kill a rebuild before publication
     (only the engine's partial is lost) — in every case the ledger
     tree is byte-unchanged and one rebuild republishes the identical
     build. The drill that gives the card its name.
   - *Zero authority*: corrupt an unlocked copy of the cache and show
     a rebuild reproduces the ledger's truth without ever reading the
     poisoned file (builders consume records only); the documented
     minimum-position query refuses (returns below-minimum) against a
     stale cache and passes after rebuild.
   - *Consumer contract*: a read-only SQL open of the published
     `cache.db` works under 4.4-style locked modes; the per-subject
     and per-actor indexed lookups return the fixture facts.
   - *Refusals*: an unverifiable ledger (stale `HEAD`) refuses before
     anything is written — unchanged engine behavior, asserted with
     `cache` registered.
3. **Spec.** Extend `next/spec/projections.md`: register `cache` in
   the projections list; a "Cache" section with the schema above, the
   determinism recipe (the closed variance sources) and its
   enforcement by the byte-identical drill, the read-only consumer
   contract and the minimum-position query, the in-memory size
   posture, and the conformance mapping: III.D row 1 extended over
   the cache, the cache row (throughput, zero authority,
   deletion-loses-nothing) drilled. State that Phase 5's
   queue-derivation swap applies to both serializations.

## File Scope

- `next/internal/project/**` (cache builder, drills)
- `next/cmd/seed/**` (CLI drills; envelope already reports every
  registered projection)
- `next/go.mod`, `next/go.sum` (the pinned driver)
- `next/spec/projections.md` (extend)
- `next/docs/decisions.md`, `next/docs/progress.md`,
  `memory/LEARNINGS.md` (if lessons emerge)

## Acceptance Criteria

**Boundary set (new, shown working):**

- One `seed project rebuild` publishes `cache` through the standard
  layout (`CURRENT`, `builds/<position>-<tip12>/cache.db`,
  `projection.json`); the envelope reports it beside the views with
  the same position and tip; the `stamp` table equals the tree stamp.
- Two builds from one prefix produce byte-identical `cache.db` on
  every fixture; the registry-wide byte-identical and
  delete-then-rebuild drills pass with `cache` registered.
- Every mirrored row set equals its published JSON view across the
  Phase 3/4 fixtures; the indexed lookups return exactly the fixture
  facts through a read-only SQL open.
- Deleting the cache build tree before or during an open read handle,
  or killing a rebuild pre-publication, loses nothing: the ledger
  tree is byte-unchanged and one rebuild republishes the identical
  build.
- A poisoned copy never influences a rebuild; the documented
  minimum-position query exposes staleness and clears after rebuild.

**Retention set (existing, shown unharmed):**

- All existing verb suites pass unchanged
  (`cd next && go test ./internal/... ./cmd/... -count=1`).
- The repo-wide gate stays green with the ≥90% coverage gate
  (`make check`).

## Validation Commands

- `make check-next`
- `make check`
- `cd next && go test ./internal/project/... -count=1`
- `cd next && go test ./internal/... ./cmd/... -count=1`
