// Package project is the projection engine (docs/next-build-plan.md
// Phase 4 item 1; plans/os-4d5cacff.md; SEED-NEXT.md Part II
// "Projections"): every projection is a deterministic function from the
// verified record prefix to a view that is derived, stamped,
// rebuildable, read-only outward, and non-authoritative. The engine
// opens the ledger read-only (it is structurally incapable of the
// healing writes the ordinary open performs), verifies from genesis
// before writing anything, and publishes each projection as immutable
// build directories plus an atomically replaced CURRENT pointer, since
// a directory rename cannot replace a non-empty target and deleting
// first would open the very window atomicity forbids (review finding
// on #105).
package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
)

// Builder is one projection: a pure function from the verified record
// prefix to named output files (slash-separated relative paths).
type Builder func(records []*event.Record) (map[string][]byte, error)

// Projection pairs a name with its builder. Version identifies the
// builder's derivation semantics, not its input: bump it when Build's
// logic changes so one ledger prefix republishes under a new build id
// rather than being discarded as a same-id duplicate. Empty means "1".
type Projection struct {
	Name    string
	Version string
	Build   Builder
}

// Stamp is the staleness surface every build carries
// (next/spec/projections.md): the exact verified position the view was
// built at, and the derivation version that built it. Consumers read
// it; the engine never hides it.
type Stamp struct {
	Name     string `json:"name"`
	Position int    `json:"position"`
	Tip      string `json:"tip"`
	Version  string `json:"version"`
}

// The published layout under <out>/<name>/: immutable build trees and
// the pointer naming the active one.
const (
	StampFile   = "projection.json"
	CurrentFile = "CURRENT"
	buildsDir   = "builds"
)

// Default returns the registered projections. Later phases append
// (standard projections 4.2, the cache 4.3); registration is data.
func Default() []Projection {
	return []Projection{Roster(), Contracts(), Queue(), Actors(), Report()}
}

// ErrOverlap refuses a ledger/output path overlap in either direction:
// a projection target must never coincide with authoritative state
// (review finding on #105).
var ErrOverlap = errors.New("the projection output must not overlap the ledger")

// canonical returns the cleaned absolute path with symlinks resolved
// over the longest existing ancestor, so a link cannot smuggle one path
// inside the other.
func canonical(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	probe := abs
	for {
		if r, err := filepath.EvalSymlinks(probe); err == nil {
			rest := strings.TrimPrefix(abs, probe)
			return filepath.Clean(r + rest), nil
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			return abs, nil
		}
		probe = parent
	}
}

func within(inner, outer string) bool {
	rel, err := filepath.Rel(outer, inner)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// CheckOverlap refuses when the ledger directory and the output root
// overlap in either direction, before anything is created.
func CheckOverlap(ledgerDir, outDir string) error {
	l, err := canonical(ledgerDir)
	if err != nil {
		return err
	}
	o, err := canonical(outDir)
	if err != nil {
		return err
	}
	if within(l, o) || within(o, l) {
		return fmt.Errorf("%w (ledger %s, out %s)", ErrOverlap, l, o)
	}
	return nil
}

// Result reports one rebuilt projection.
type Result struct {
	Name     string
	Position int
	Tip      string
	Version  string
}

// Rebuild is the one-command rebuild: refuse overlapping paths, open
// the ledger read-only, verify from genesis (a failure refuses before
// anything is written), run every registered projection over the
// verified records, and publish each tree. The build id derives from
// the stamp, so identical prefixes reproduce identical trees, CURRENT
// included: deleting the output loses nothing.
func Rebuild(ledgerDir, outDir string, projections []Projection, resolve ledger.Resolver, vopts ...ledger.VerifyOption) ([]Result, error) {
	if err := CheckOverlap(ledgerDir, outDir); err != nil {
		return nil, err
	}
	store, err := ledger.OpenReadOnly(ledgerDir)
	if err != nil {
		return nil, err
	}
	var records []*event.Record
	vopts = append(vopts, ledger.WithObserver(func(pos int, rec *event.Record) {
		records = append(records, rec)
	}))
	rep, err := store.VerifyFromGenesis(resolve, vopts...)
	if err != nil {
		return nil, err
	}
	results := make([]Result, 0, len(projections))
	for _, p := range projections {
		ver := p.Version
		if ver == "" {
			ver = "1"
		}
		// The build id derives from (position, tip, version): the
		// verified prefix AND the derivation semantics. A projection
		// whose Build logic changes bumps Version, so the same ledger
		// tip republishes under a new id instead of being discarded
		// as a same-id duplicate (review finding on #109).
		buildID := fmt.Sprintf("%08d-%.12s-v%s", rep.Count, rep.Tip, ver)
		files, err := p.Build(records)
		if err != nil {
			return nil, fmt.Errorf("projection %s: %v", p.Name, err)
		}
		stamp, err := json.Marshal(Stamp{Name: p.Name, Position: rep.Count, Tip: rep.Tip, Version: ver})
		if err != nil {
			return nil, err
		}
		if files == nil {
			files = map[string][]byte{}
		}
		files[StampFile] = append(stamp, '\n')
		if err := publish(filepath.Join(outDir, p.Name), buildID, files); err != nil {
			return nil, fmt.Errorf("projection %s: %v", p.Name, err)
		}
		results = append(results, Result{Name: p.Name, Position: rep.Count, Tip: rep.Tip, Version: ver})
	}
	return results, nil
}

// publish writes one immutable, complete build tree and only then
// swaps CURRENT to it; superseded builds and stray partials prune
// after the swap. A reader always resolves CURRENT to a complete tree;
// a killed build leaves at worst an orphan directory.
func publish(root, buildID string, files map[string][]byte) error {
	buildRoot := filepath.Join(root, buildsDir, buildID)
	tmp := buildRoot + ".partial"
	if err := os.RemoveAll(tmp); err != nil {
		return err
	}
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		dst := filepath.Join(tmp, filepath.FromSlash(n))
		if !within(dst, tmp) {
			return fmt.Errorf("projection file %q escapes its build directory", n)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, files[n], 0o644); err != nil {
			return err
		}
	}
	if _, err := os.Stat(buildRoot); err == nil {
		// The same prefix rebuilt: determinism makes the existing tree
		// byte-identical to the one just built, and a reader may be on
		// it, so keep it and discard the duplicate.
		if err := os.RemoveAll(tmp); err != nil {
			return err
		}
	} else if err := os.Rename(tmp, buildRoot); err != nil {
		return err
	}
	cur := filepath.Join(root, CurrentFile)
	// The build CURRENT named before this swap is retained through the
	// prune: a reader that resolved CURRENT just before the swap must
	// find a complete tree at the path it holds (review finding on
	// #109). Everything older, and stray partials, prune; a reader
	// that loses the race to two consecutive swaps re-resolves.
	prev := ""
	if b, err := os.ReadFile(cur); err == nil {
		prev = strings.TrimSpace(string(b))
	}
	tmpCur := cur + ".tmp"
	if err := os.WriteFile(tmpCur, []byte(buildID+"\n"), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmpCur, cur); err != nil {
		return err
	}
	entries, err := os.ReadDir(filepath.Join(root, buildsDir))
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.Name() != buildID && e.Name() != prev {
			if err := os.RemoveAll(filepath.Join(root, buildsDir, e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

// Current resolves a projection's active build directory from its
// CURRENT pointer.
func Current(outDir, name string) (string, error) {
	root := filepath.Join(outDir, name)
	b, err := os.ReadFile(filepath.Join(root, CurrentFile))
	if err != nil {
		return "", err
	}
	id := strings.TrimSpace(string(b))
	if id == "" {
		return "", fmt.Errorf("projection %s has an empty CURRENT pointer", name)
	}
	return filepath.Join(root, buildsDir, id), nil
}
