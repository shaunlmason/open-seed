// The SQLite cache projection (plans/os-acc1ac78.md, amended per the
// #110 review): a registered standard projection, byte-identical like
// every view, publishing one queryable database for single-machine
// read throughput with zero authority. Byte-determinism is engineered
// by closing the variance sources — one connection, one ordered
// transaction, rollback journal (never WAL, whose headers embed
// salts), fixed page_size, no auto_vacuum, no ANALYZE, no
// AUTOINCREMENT — and the registry's byte-identical drill enforces it
// against the go.sum-pinned driver.

package project

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/transition"

	_ "modernc.org/sqlite"
)

// CacheFile is the cache projection's one view file: the database is
// the API, opened read-only by any SQL client.
const CacheFile = "cache.db"

// cacheSchemaVersion is the database's own schema generation, stamped
// via PRAGMA user_version; it bumps with the table set, not with the
// chain position. Generation 2 added contract_state and the derived
// queue rows (plans/os-d69a6c91.md); generation 3 the claim columns
// (plans/os-5dc16a7c.md); generation 4 the acceptance columns
// (plans/os-73c00a50.md); generation 5 the reconciliation-chain
// columns (plans/os-6cdc15be.md).
const cacheSchemaVersion = 5

// cacheVersion is the projection's derivation version, carried in the
// stamp table and the build id alike.
const cacheVersion = "6"

// Cache returns the cache projection.
func Cache() Projection {
	return Projection{Name: "cache", Version: cacheVersion, Build: buildCache}
}

// cacheDDL creates the tables in spec order (next/spec/projections.md
// "Cache"). The stamp row is written last in the build transaction and
// carries exactly the tree stamp's fields.
var cacheDDL = []string{
	`CREATE TABLE stamp (name TEXT NOT NULL, position INTEGER NOT NULL, tip TEXT NOT NULL, version TEXT NOT NULL)`,
	`CREATE TABLE roster (fingerprint TEXT PRIMARY KEY, kind TEXT NOT NULL, name TEXT NOT NULL, standing TEXT NOT NULL, root INTEGER NOT NULL, grants TEXT NOT NULL) WITHOUT ROWID`,
	`CREATE TABLE contracts (subject TEXT NOT NULL, position INTEGER NOT NULL, verb TEXT NOT NULL, actor TEXT NOT NULL, payload TEXT NOT NULL)`,
	`CREATE INDEX contracts_subject ON contracts(subject)`,
	`CREATE TABLE queue (subject TEXT NOT NULL, since_position INTEGER NOT NULL)`,
	`CREATE TABLE contract_state (subject TEXT PRIMARY KEY, state TEXT, anomalies INTEGER NOT NULL, holder TEXT, fence TEXT, acc_ref TEXT, acc_executable INTEGER, acc_gated INTEGER, verdict_position INTEGER, verdict TEXT, verdict_receipt TEXT, requested_position INTEGER, merged_position INTEGER, merged_sha TEXT) WITHOUT ROWID`,
	`CREATE TABLE queue_meta (schema_version TEXT NOT NULL, derivation TEXT NOT NULL)`,
	`CREATE TABLE actor_history (fingerprint TEXT NOT NULL, position INTEGER NOT NULL, verb TEXT NOT NULL, acting TEXT NOT NULL)`,
	`CREATE INDEX actor_history_fp ON actor_history(fingerprint)`,
	`CREATE TABLE actor_signed (fingerprint TEXT NOT NULL, position INTEGER NOT NULL, verb TEXT NOT NULL, subject TEXT NOT NULL)`,
	`CREATE INDEX actor_signed_fp ON actor_signed(fingerprint)`,
	`CREATE TABLE report (key TEXT PRIMARY KEY, value TEXT NOT NULL) WITHOUT ROWID`,
}

func buildCache(records []*event.Record, _ Inputs) (files map[string][]byte, err error) {
	tmpDir, err := os.MkdirTemp("", "seed-cache-build-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)
	path := filepath.Join(tmpDir, CacheFile)

	// One connection; the pragmas close the variance sources before
	// any page is allocated.
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	closed := false
	defer func() {
		if !closed {
			_ = db.Close()
		}
	}()
	for _, pragma := range []string{
		"PRAGMA page_size = 4096",
		"PRAGMA journal_mode = OFF",
		"PRAGMA auto_vacuum = NONE",
		fmt.Sprintf("PRAGMA user_version = %d", cacheSchemaVersion),
	} {
		if _, err := db.Exec(pragma); err != nil {
			return nil, fmt.Errorf("cache pragma: %v", err)
		}
	}

	roster, err := rosterEntries(records)
	if err != nil {
		return nil, err
	}
	view, err := reportView(records)
	if err != nil {
		return nil, err
	}
	// The stamp row is written last and must equal the tree stamp the
	// engine writes beside this database (drilled).
	position := len(records)
	tip := ""
	if position > 0 {
		if tip, err = records[position-1].Event.Hash(); err != nil {
			return nil, err
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	// One error captor for the whole ordered write sequence.
	w := &txWriter{tx: tx}
	for _, ddl := range cacheDDL {
		w.exec(ddl)
	}
	for _, r := range roster {
		root := 0
		if r.Root {
			root = 1
		}
		w.exec(`INSERT INTO roster VALUES (?, ?, ?, ?, ?, ?)`,
			r.Fingerprint, r.Kind, r.Name, r.Standing, root, w.jsonOf(r.Grants))
	}
	table, err := transition.Default()
	if err != nil {
		return nil, err
	}
	fold := table.FoldRecords(records)
	for _, c := range contractEntries(records) {
		for _, e := range c.Events {
			w.exec(`INSERT INTO contracts VALUES (?, ?, ?, ?, ?)`,
				c.Subject, e.Position, e.Verb, e.Actor, string(e.Payload))
		}
		var state, holder, fence, accRef, accExec, accGated any
		var verdictPos, verdictVal, verdictReceipt, requestedPos, mergedPos, mergedSHA any
		anomalies := 0
		if s, ok := fold.State(c.Subject); ok {
			anomalies = s.Anomalies
			if s.State != "" {
				state = s.State
			}
			if s.Claim != nil {
				holder, fence = s.Claim.Holder, fmt.Sprintf("%d", s.Claim.Fence)
			}
			if s.Acceptance != nil {
				accRef = s.Acceptance.Ref
				accExec, accGated = boolInt(s.Acceptance.Executable), boolInt(s.Acceptance.Gated)
			}
			if s.Verdict != nil {
				verdictPos, verdictVal, verdictReceipt = s.Verdict.Pos, s.Verdict.Verdict, s.Verdict.Receipt
			}
			if s.Requested != nil {
				requestedPos = s.Requested.Pos
			}
			if s.Merged != nil {
				mergedPos, mergedSHA = s.Merged.Pos, s.Merged.SHA
			}
		}
		w.exec(`INSERT INTO contract_state VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, c.Subject, state, anomalies, holder, fence, accRef, accExec, accGated, verdictPos, verdictVal, verdictReceipt, requestedPos, mergedPos, mergedSHA)
	}
	// The queue mirrors the JSON view's derivation exactly: the
	// transition table's ready set, oldest first.
	ready, err := readyEntries(records)
	if err != nil {
		return nil, err
	}
	w.exec(`INSERT INTO queue_meta VALUES (?, ?)`, QueueSchemaVersion, QueueDerivationTransitions)
	for _, q := range ready {
		w.exec(`INSERT INTO queue VALUES (?, ?)`, q.Subject, q.SincePosition)
	}
	for _, fp := range candidateFingerprints(records) {
		for pos, rec := range records {
			ev := &rec.Event
			if isActorVerbOn(ev, fp) {
				w.exec(`INSERT INTO actor_history VALUES (?, ?, ?, ?)`, fp, pos, ev.Verb, ev.Actor)
			}
			if ev.Actor == fp {
				w.exec(`INSERT INTO actor_signed VALUES (?, ?, ?, ?)`, fp, pos, ev.Verb, ev.Subject)
			}
		}
	}
	w.exec(`INSERT INTO report VALUES ('chain', ?)`, w.jsonOf(view.Chain))
	w.exec(`INSERT INTO report VALUES ('actors', ?)`, w.jsonOf(view.Actors))
	w.exec(`INSERT INTO report VALUES ('halt', ?)`, w.jsonOf(view.Halt))
	w.exec(`INSERT INTO report VALUES ('checkpoints', ?)`, w.jsonOf(view.Checkpoints))
	w.exec(`INSERT INTO report VALUES ('contracts', ?)`, w.jsonOf(view.Contracts))
	w.exec(`INSERT INTO stamp VALUES (?, ?, ?, ?)`, "cache", position, tip, cacheVersion)
	if err = w.err; err != nil {
		return nil, fmt.Errorf("cache write: %v", err)
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	closed = true
	if err = db.Close(); err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return map[string][]byte{CacheFile: b}, nil
}

// txWriter runs an ordered write sequence capturing the first error,
// so the deterministic insert order reads as data, not error plumbing.
type txWriter struct {
	tx  *sql.Tx
	err error
}

func (w *txWriter) exec(q string, args ...any) {
	if w.err != nil {
		return
	}
	_, w.err = w.tx.Exec(q, args...)
}

// jsonOf marshals for a column value, capturing the first error.
func (w *txWriter) jsonOf(v any) string {
	if w.err != nil {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		w.err = err
		return ""
	}
	return string(b)
}

// boolInt renders a flag as a SQLite integer.
func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// isActorVerbOn reports an actor.* event applied to the fingerprint,
// mirroring the actor view's standing-history derivation.
func isActorVerbOn(ev *event.Event, fp string) bool {
	return ev.Subject == fp && len(ev.Verb) > 6 && ev.Verb[:6] == "actor."
}
