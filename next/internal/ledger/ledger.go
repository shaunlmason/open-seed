// Package ledger implements the append-only chain store over
// next/internal/event (charter Part II section 1; plans/os-ead12024.md).
// Layout is the build-plan fixed default: JSONL segments, one file per UTC
// day under segments/, and a HEAD record carrying the tip hash. The
// segment stream is authoritative and HEAD is a derived cache of it:
// Append reconciles the true tip from the stream before checking prev, so
// a crash between the durable segment line and the HEAD rename heals on
// the next use instead of forking the chain.
package ledger

import (
	"bufio"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/shaunlmason/open-seed/next/internal/event"
)

// Resolver maps an actor fingerprint to its public key. The keyring
// projection supplies this from Phase 3 on; genesis bootstrap and tests
// supply fixture resolvers.
type Resolver func(fingerprint string) (ed25519.PublicKey, bool)

// Typed refusals, mapped to envelope codes by later phases.
var (
	ErrUnknownActor = errors.New("actor fingerprint not in the keyring")
	ErrWrongPrev    = errors.New("prev does not cite the current tip")
	ErrEmptyRecord  = errors.New("nil record")
)

const (
	segmentsDir = "segments"
	headFile    = "HEAD"
)

// Head is the derived cache of the segment stream's end.
type Head struct {
	Tip     string `json:"tip"`
	Count   int    `json:"count"`
	Segment string `json:"segment"`
}

// Store is a ledger checkout rooted at a directory.
type Store struct {
	dir string
	now func() time.Time
}

// Option configures a Store.
type Option func(*Store)

// WithClock injects the append clock (tests; the default is time.Now).
func WithClock(now func() time.Time) Option {
	return func(s *Store) { s.now = now }
}

// Open prepares a store rooted at dir, creating the layout if absent.
func Open(dir string, opts ...Option) (*Store, error) {
	s := &Store{dir: dir, now: time.Now}
	for _, o := range opts {
		o(s)
	}
	if err := os.MkdirAll(filepath.Join(dir, segmentsDir), 0o755); err != nil {
		return nil, err
	}
	return s, nil
}

// segmentNames returns the segment file names in read order.
func (s *Store) segmentNames() ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(s.dir, segmentsDir))
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".jsonl") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// scan streams every record in order, calling fn with the zero-based
// position, the segment name, and the raw line. fn returning an error
// stops the scan.
func (s *Store) scan(fn func(pos int, segment string, line []byte) error) error {
	names, err := s.segmentNames()
	if err != nil {
		return err
	}
	pos := 0
	for _, name := range names {
		f, err := os.Open(filepath.Join(s.dir, segmentsDir, name))
		if err != nil {
			return err
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" {
				continue
			}
			if err := fn(pos, name, []byte(line)); err != nil {
				f.Close()
				return err
			}
			pos++
		}
		if err := sc.Err(); err != nil {
			f.Close()
			return err
		}
		f.Close()
	}
	return nil
}

// reconcile derives the true tip from the segment stream: the hash of the
// last record, the total count, and the last segment name. An empty stream
// reports the empty hash at count zero. HEAD is never consulted for the
// answer, only healed from it.
func (s *Store) reconcile() (tip string, count int, lastSegment string, err error) {
	tip = event.EmptyHash
	err = s.scan(func(pos int, segment string, line []byte) error {
		rec, perr := event.ParseRecord(line)
		if perr != nil {
			return fmt.Errorf("position %d (%s): %w", pos, segment, perr)
		}
		h, herr := rec.Event.Hash()
		if herr != nil {
			return fmt.Errorf("position %d (%s): %w", pos, segment, herr)
		}
		tip, count, lastSegment = h, pos+1, segment
		return nil
	})
	return tip, count, lastSegment, err
}

// Tip reports the reconciled tip hash and count.
func (s *Store) Tip() (string, int, error) {
	tip, count, _, err := s.reconcile()
	return tip, count, err
}

// segmentForAppend picks the file a new record lands in: the append day's
// name, unless an existing segment sorts later (a clock regression), in
// which case the newest existing segment keeps growing so segment names
// never regress and read order stays linear (plans/os-ead12024.md step 1).
func (s *Store) segmentForAppend(newest string) string {
	today := s.now().UTC().Format("2006-01-02") + ".jsonl"
	if newest != "" && newest > today {
		return newest
	}
	return today
}

// Append verifies the record against the resolver, checks prev against the
// reconciled tip, writes the line durably, then rewrites HEAD atomically.
// It returns the record's zero-based position.
func (s *Store) Append(rec *event.Record, resolve Resolver) (int, error) {
	if rec == nil {
		return 0, ErrEmptyRecord
	}
	pub, ok := resolve(rec.Event.Actor)
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrUnknownActor, rec.Event.Actor)
	}
	if err := rec.Verify(pub); err != nil {
		return 0, err
	}
	tip, count, lastSegment, err := s.reconcile()
	if err != nil {
		return 0, err
	}
	if rec.Event.Prev != tip {
		return 0, fmt.Errorf("%w: prev %.12s, tip %.12s", ErrWrongPrev, rec.Event.Prev, tip)
	}
	line, err := rec.Marshal()
	if err != nil {
		return 0, err
	}
	segment := s.segmentForAppend(lastSegment)
	f, err := os.OpenFile(filepath.Join(s.dir, segmentsDir, segment), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return 0, err
	}
	if _, err := f.Write(line); err != nil {
		f.Close()
		return 0, err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return 0, err
	}
	if err := f.Close(); err != nil {
		return 0, err
	}
	newTip, err := rec.Event.Hash()
	if err != nil {
		return 0, err
	}
	if err := s.writeHead(Head{Tip: newTip, Count: count + 1, Segment: segment}); err != nil {
		return 0, err
	}
	return count, nil
}

// writeHead rewrites HEAD atomically (temp + rename).
func (s *Store) writeHead(h Head) error {
	b, err := json.Marshal(h)
	if err != nil {
		return err
	}
	tmp := filepath.Join(s.dir, headFile+".tmp")
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(s.dir, headFile))
}

// ReadHead returns the cached HEAD record; ok is false when none exists.
func (s *Store) ReadHead() (Head, bool, error) {
	b, err := os.ReadFile(filepath.Join(s.dir, headFile))
	if errors.Is(err, os.ErrNotExist) {
		return Head{}, false, nil
	}
	if err != nil {
		return Head{}, false, err
	}
	var h Head
	if err := json.Unmarshal(b, &h); err != nil {
		return Head{}, false, fmt.Errorf("HEAD does not parse: %w", err)
	}
	return h, true, nil
}
