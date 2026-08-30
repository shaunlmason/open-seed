# Plan: next Phase 4.3 — the SQLite cache projection (os-acc1ac78)

Implements [`docs/next-build-plan.md`](../docs/next-build-plan.md) Phase 4
item 3: "SQLite cache projection; mid-operation deletion drill", with the
plan's binding default `modernc.org/sqlite` (no cgo, portable builds).
Design authority: [`SEED-NEXT.md`](../SEED-NEXT.md) §II.2 (every cache is
a one-way, rebuildable, disposable projection), §II.4 ("a local database
cache for single-machine read throughput", "a local cache projection
delivers machine-local speed without a second authority"), the
what-Seed-does-not-have row ("A machine-local authoritative store — the
cache projection provides the throughput without a second authority"),
and conformance III.D's cache row: "The cache projection delivers
single-machine read throughput with zero authority (mid-operation
deletion loses nothing)." Deps: the 4.1 engine and 4.2 views
(plans #105/#106); implementation stacks on their branches until they
merge.

## Design decisions (binding for this task)

- **The cache is not a registered projection and not byte-identical.**
  III.D row 1 (byte-identical, one command) governs the file-tree read
  surfaces; the cache carries its own conformance row with its own
  rebuild property, which would be redundant if row 1 already bound it.
  SQLite files are not byte-stable across library versions, so claiming
  byte-identity would weld the drill to `modernc.org/sqlite` internals.
  The cache's contract is **semantic identity**: two builds from the
  same verified prefix hold the same logical content (schema, rows,
  stamp), compared by a Go-side ordered dump-and-hash, never by file
  bytes. The registry (`project.Default()`) stays view-only; the
  byte-identical drill over the publication trees is untouched.
- **Placement: outside the locked view trees.** The database publishes
  to `<out>/cache.db`, sibling of the `<name>/` projection roots, never
  inside them — so 4.4's locked directories (plan #107) stay fully
  read-only, the whole-tree byte comparison never sees the DB, and the
  engine atomically replaces the file by build-to-temp plus rename in
  the writable `<out>/` root. One database for all views, not one per
  view: single-machine throughput is the point and cross-view joins are
  the payoff. When 4.4 lands, its vocabulary lint (Lint A) grows the
  `"cache.db"` literal; whichever branch merges second carries the
  union (recorded in both PRs).
- **Built from the records, not from the view files.** The cache
  builder consumes the same verified record prefix as every Builder,
  reusing the view derivations' intermediate state (roster entries,
  contract streams, keyring state) rather than parsing published JSON
  back in — one derivation, two serializations. `seed project rebuild`
  builds it by default alongside the views (III.D's one-command posture
  and the rebuild-everything drill cover both); `--no-cache` skips it.
- **The consumer surface is SQL itself.** SQLite is the API: any
  driver or the `sqlite3` shell reads the file directly, which is the
  entire point of choosing the format. The engine's obligations are
  the stamp table (staleness visible; a consumer demands a minimum
  position with one documented query) and read-only-outward openness
  (consumers open read-only; the spec says a writable open of the
  cache is a programming error and why it cannot become authority).
  No new query verbs; `seed project rebuild`'s envelope reports the
  cache alongside the views.
- **Throughput is the purpose, correctness is the test.** The drills
  assert indexed per-subject and per-actor lookups return exactly the
  view-equivalent facts; CI runs no benchmarks (timing assertions
  flake). The spec states this division honestly.

## Schema (v1, stated in the spec)

- `stamp(schema_version TEXT, position INTEGER, tip TEXT)` — one row,
  written last inside the build transaction; equals the verification
  report the views were stamped with.
- `roster(fingerprint TEXT PRIMARY KEY, kind TEXT, name TEXT,
  standing TEXT, root INTEGER, grants TEXT)` — grants as a JSON array,
  matching `roster.json`.
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

Deterministic build: one connection, fixed pragmas, tables created and
rows inserted in spec order; determinism is asserted at the semantic
layer (above), so pragma drift cannot silently weaken the drill.

## Steps

1. **The cache builder** (`next/internal/project/cache.go`): build to
   `cache.db.partial` in `<out>/`, insert everything in one
   transaction from the shared derivations, write the stamp row last,
   close, rename over `cache.db`; discard the partial on any error.
   Wire into `Rebuild` after view publication (`--no-cache` in the CLI
   maps to an engine option), reporting `{name: "cache", position,
   tip}` in the results. Add `modernc.org/sqlite` to `next/go.mod`
   pinned at the current stable release, recorded in `go.sum`.
2. **Semantic dump** (`next/internal/project/cachedump.go`): open
   read-only, walk tables in schema order with `ORDER BY` primary
   key/rowid, hash the typed rows — the equality surface for every
   drill below and the documented way an operator diffs two caches.
3. **Drills** (library + CLI):
   - *Semantic rebuild*: build twice from one prefix — equal dumps;
     grow the chain — dumps differ and the stamp advances with the
     report.
   - *View equivalence*: every row set equals the published JSON views
     it mirrors, fixture by fixture (roster, contracts, queue with
     `derivation` `"none"`, actor streams, report facts).
   - *Mid-operation deletion loses nothing*: delete `cache.db` while a
     read connection holds it open (POSIX keeps the inode; reads
     complete), after it, and mid-rebuild (kill the build before the
     rename; only the partial is lost) — in every case the ledger tree
     is byte-unchanged and one rebuild restores an equal dump. The
     drill that gives the card its name.
   - *Zero authority*: rebuild with a poisoned cache (extra row,
     mutated standing) — the rebuilt cache matches the ledger, the
     views never read it, and nothing downstream of the engine treated
     it as input; plus the staleness demand: the documented
     minimum-position query refuses (returns below-minimum) on a stale
     cache and passes after rebuild.
   - *Refusals*: an unverifiable ledger (stale `HEAD`) refuses before
     the partial is created; `--no-cache` leaves no `cache.db`; the
     overlap refusals are unchanged.
4. **Spec.** Extend `next/spec/projections.md` with a "Cache" section:
   the schema above, semantic-vs-byte identity and why (with the
   library-version argument), placement outside the locked trees, the
   read-only consumer contract, the minimum-position query, and the
   conformance mapping marking III.D's cache row drilled. State the
   Phase 5 queue-derivation handoff applies to both serializations.

## File Scope

- `next/internal/project/**` (cache builder, dump, drills)
- `next/cmd/seed/**` (`--no-cache` flag, envelope rows, CLI drills)
- `next/go.mod`, `next/go.sum` (the pinned driver)
- `next/spec/projections.md` (extend)
- `next/docs/decisions.md`, `next/docs/progress.md`,
  `memory/LEARNINGS.md` (if lessons emerge)

## Acceptance Criteria

**Boundary set (new, shown working):**

- One `seed project rebuild` publishes the views and `cache.db`; the
  envelope reports the cache with the same position/tip as the views;
  the stamp table equals the verification report.
- Two builds from one prefix produce equal semantic dumps; every
  mirrored row set equals its published JSON view across the Phase 3/4
  fixtures.
- Deleting the database before, during (open handle), or mid-rebuild
  (pre-rename kill) loses nothing: the ledger tree is byte-unchanged
  and one rebuild restores an equal dump.
- A poisoned cache never influences a rebuild or a view; the
  documented minimum-position query exposes staleness and clears after
  rebuild.
- The view trees' byte-identical drill passes unchanged with the cache
  present (the DB lives outside every `<name>/` tree).

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
