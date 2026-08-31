// Package artifact is the minimal content-addressed artifact store
// (plans/os-f6d2c267.md; SEED-NEXT.md Part II §8): bytes keyed by the
// lowercase-hex SHA-256 of their content, rooted on the filesystem.
// The build plan's git-addressed refs/seed/artifacts push is deferred
// (the observation-channel precedent: simplest channel first); the
// deferral is recorded in next/docs/decisions.md. Receipts are the
// first tenant; sealed-check ciphertexts join in 6.3.
package artifact

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// Store is a filesystem-rooted content-addressed store.
type Store struct{ root string }

var digestRE = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Open returns the store rooted at dir (created on first put).
func Open(dir string) *Store { return &Store{root: dir} }

// Digest returns the store's key for b: lowercase-hex SHA-256.
func Digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func (s *Store) path(digest string) string {
	return filepath.Join(s.root, "sha256", digest)
}

// Put stores b and returns its digest. A concurrent put of the same
// content is idempotent: the write lands in a unique temp file and
// renames into place, so rivals cannot interleave partial bytes.
func (s *Store) Put(b []byte) (string, error) {
	digest := Digest(b)
	dst := s.path(digest)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", fmt.Errorf("artifact store: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(dst), "put-*")
	if err != nil {
		return "", fmt.Errorf("artifact store: %w", err)
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", fmt.Errorf("artifact store: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return "", fmt.Errorf("artifact store: %w", err)
	}
	if err := os.Rename(tmp.Name(), dst); err != nil {
		os.Remove(tmp.Name())
		return "", fmt.Errorf("artifact store: %w", err)
	}
	return digest, nil
}

// Get returns the bytes stored under digest, recomputing and checking
// the digest on the way out: a store whose disk content no longer
// hashes to its name is corrupt, and corrupt content must never be
// returned as if addressed.
func (s *Store) Get(digest string) ([]byte, error) {
	if !digestRE.MatchString(digest) {
		return nil, fmt.Errorf("artifact store: %q is not a lowercase-hex sha256 digest", digest)
	}
	b, err := os.ReadFile(s.path(digest))
	if err != nil {
		return nil, fmt.Errorf("artifact store: %w", err)
	}
	if got := Digest(b); got != digest {
		return nil, fmt.Errorf("artifact store: content under %s hashes to %s — the store is corrupt at that address", digest, got)
	}
	return b, nil
}
