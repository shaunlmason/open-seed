// Package posture holds the deployment's declared admission posture
// (SEED-NEXT.md Part II "Postures"; plans/os-3c72f93f.md): every
// deployment MUST declare which of the three named deployments it runs,
// and cooperative is a declared mode with a named consequence the
// preflight tool states in plain words. The declaration is
// client/deployment state, never ledger content: the guarded ref's tree
// stays HEAD and segments only.
package posture

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
)

// Posture is one of the charter's three named admission deployments.
type Posture string

const (
	EnforcedSelfHosted  Posture = "enforced-self-hosted"
	EnforcedForgeHosted Posture = "enforced-forge-hosted"
	Cooperative         Posture = "cooperative"
)

// Consequence is the charter's named cooperative consequence, in the
// charter's own words. One copy: the doctor and any future README
// generator quote it, never re-derive it.
const Consequence = "cooperative posture: no server-side enforcement — every writer self-validates; the security invariant (SEED-NEXT.md §I.2) does not hold, and protocol rules are advisory against a hostile credential"

// ForgeHostedGap names the honest state of the forge-hosted posture
// until its admission service lands (docs/next-build-plan.md Phase 12).
const ForgeHostedGap = "enforced-forge-hosted is declared, but its admission service lands in Phase 12 (docs/next-build-plan.md); until then this deployment must document its interim enforcement"

// ErrUndeclared is the refusal for a deployment with no declaration.
var ErrUndeclared = errors.New("no admission posture is declared: every deployment MUST declare which posture it runs (SEED-NEXT.md Part II, \"Postures\")")

// ErrUnreadable marks a declaration that exists but cannot be read (a
// directory, denied permissions, an I/O failure): an operational
// failure, never a judgment on the declaration's content.
var ErrUnreadable = errors.New("posture declaration cannot be read")

func validList() string {
	return fmt.Sprintf("%s, %s, %s", EnforcedSelfHosted, EnforcedForgeHosted, Cooperative)
}

// Valid reports whether p is one of the three charter postures.
func (p Posture) Valid() bool {
	switch p {
	case EnforcedSelfHosted, EnforcedForgeHosted, Cooperative:
		return true
	}
	return false
}

// Enforced reports whether the posture carries server-side enforcement.
func (p Posture) Enforced() bool {
	return p == EnforcedSelfHosted || p == EnforcedForgeHosted
}

// DeclarationPath is where the reference deployment keeps the
// declaration, relative to the repository root: the file `seed doctor`
// reads by default, and the one the enforced hook reads from the
// default branch's tip to learn the protected surface
// (plans/os-465e356e.md D1). It is protected by construction: a list
// that could be unprotected by editing the file that carries it would
// protect nothing.
const DeclarationPath = "seed.json"

// Config is the deployment declaration.
type Config struct {
	Posture Posture `json:"posture"`
	// Protected enumerates the protected surface (SEED-NEXT.md §II.14)
	// as repository-relative path prefixes: a path equal to an entry,
	// or under an entry as a directory, is write-denied to every agent
	// credential at the enforced hook. Optional; the declaration's own
	// path is always protected whether or not it is listed.
	Protected []string `json:"protected,omitempty"`
}

// Load reads a strict declaration {"posture": "<name>", "protected"?}:
// unknown fields, trailing data, unknown postures and malformed
// protected entries refuse; a missing file is ErrUndeclared.
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrUndeclared
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnreadable, err)
	}
	return Parse(b)
}

// Parse decodes the declaration's bytes with Load's strictness, for a
// reader that already holds them (the hook reads the file out of a
// commit rather than off a filesystem).
func Parse(b []byte) (*Config, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var c Config
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("posture declaration does not parse: %v (valid postures: %s)", err, validList())
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("posture declaration carries trailing data (valid postures: %s)", validList())
	}
	if !c.Posture.Valid() {
		return nil, fmt.Errorf("%q is not a Seed posture (valid postures: %s)", c.Posture, validList())
	}
	for i, p := range c.Protected {
		if err := validProtectedEntry(p); err != nil {
			return nil, fmt.Errorf("protected[%d] %q: %v (valid postures: %s)", i, p, err, validList())
		}
	}
	return &c, nil
}

// validProtectedEntry admits a clean repository-relative path prefix:
// non-empty, not absolute, not escaping the root, nothing a git
// pathname would not spell the same way.
func validProtectedEntry(p string) error {
	trimmed := strings.TrimSuffix(p, "/")
	switch {
	case strings.TrimSpace(trimmed) == "":
		return errors.New("an empty entry protects nothing")
	case strings.HasPrefix(p, "/"):
		return errors.New("entries are repository-relative, never absolute")
	case strings.Contains(p, "\\"):
		return errors.New("entries use forward slashes")
	case path.Clean(trimmed) != trimmed || trimmed == "." || trimmed == ".." || strings.HasPrefix(trimmed, "../"):
		return errors.New("entries are clean relative paths (no '.', '..' or doubled separators)")
	}
	return nil
}

// ProtectedSurface is the declared list plus the declaration's own
// path, deduplicated and sorted: the enumeration the hook and the
// doctor both report.
func (c *Config) ProtectedSurface() []string {
	seen := map[string]bool{DeclarationPath: true}
	out := []string{DeclarationPath}
	for _, p := range c.Protected {
		e := strings.TrimSuffix(p, "/")
		if !seen[e] {
			seen[e] = true
			out = append(out, e)
		}
	}
	sort.Strings(out)
	return out
}

// Protects reports whether a repository path falls on the protected
// surface: equal to an entry, or under it as a directory.
func (c *Config) Protects(p string) bool {
	for _, e := range c.ProtectedSurface() {
		if p == e || strings.HasPrefix(p, e+"/") {
			return true
		}
	}
	return false
}
