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
	"net/url"
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

// DefaultLedgerRef is the ledger ref a forge-hosted deployment rides
// when the admission block does not name one: a branch, because forges
// protect branches and tags and nothing under refs/seed/* (#244).
const DefaultLedgerRef = "refs/heads/seed-ledger"

// Admission is the forge-hosted deployment's admission block. Endpoint
// is where `seed-admit serve` answers proposals; Identity is the forge
// login or app slug the service's git credential belongs to (the
// ledger branch's sole writer); LedgerRef is the branch the ledger
// rides (DefaultLedgerRef when empty); Checks and Reviews are the
// default branch's required status checks and review count; Owners
// are the forge identities that review the protected surface, rendered
// into CODEOWNERS by the reconciler (#244).
type Admission struct {
	Endpoint  string   `json:"endpoint"`
	Identity  string   `json:"identity"`
	LedgerRef string   `json:"ledger_ref"`
	Checks    []string `json:"checks"`
	Reviews   int      `json:"reviews"`
	Owners    []string `json:"owners"`
}

// Trust choices a fresh reader may declare for checkpoints
// (next/spec/checkpoints.md; plans/os-7508ab9e.md D1).
const (
	TrustReplay  = "replay"  // every fresh reader verifies from genesis once
	TrustSigners = "signers" // a fresh reader may start from a capable signer's checkpoint
)

// Checkpoints is the declaration's checkpoint-trust block: which of a
// fresh clone's two verification obligations this deployment chose.
// An absent block is undeclared — never a default — because the
// charter says the choice is declared, not defaulted.
type Checkpoints struct {
	Trust string `json:"trust"`
}

// Config is the deployment declaration.
type Config struct {
	Posture   Posture    `json:"posture"`
	Admission *Admission `json:"admission,omitempty"`
	// Protected enumerates the protected surface (SEED-NEXT.md §II.14)
	// as repository-relative path prefixes: a path equal to an entry,
	// or under an entry as a directory, is write-denied to every agent
	// credential at the enforced hook. Optional; the declaration's own
	// path is always protected whether or not it is listed.
	Protected   []string     `json:"protected,omitempty"`
	Checkpoints *Checkpoints `json:"checkpoints,omitempty"`
}

// CheckpointTrust returns the declared trust choice, or "" when the
// block is absent (undeclared).
func (c *Config) CheckpointTrust() string {
	if c == nil || c.Checkpoints == nil {
		return ""
	}
	return c.Checkpoints.Trust
}

// LedgerRef is the ref the ledger rides under this declaration: the
// admission block's branch under enforced-forge-hosted, or "" for the
// other postures, whose callers keep their own default (#244).
func (c *Config) LedgerRef() string {
	if c.Posture != EnforcedForgeHosted || c.Admission == nil {
		return ""
	}
	if c.Admission.LedgerRef == "" {
		return DefaultLedgerRef
	}
	return c.Admission.LedgerRef
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
	if err := c.validateAdmission(); err != nil {
		return nil, fmt.Errorf("%v (valid postures: %s)", err, validList())
	}
	if c.Checkpoints != nil && c.Checkpoints.Trust != TrustReplay && c.Checkpoints.Trust != TrustSigners {
		return nil, fmt.Errorf("checkpoints.trust is %q or %q (a fresh reader's verification obligation is declared, not defaulted), got %q", TrustReplay, TrustSigners, c.Checkpoints.Trust)
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

// validateAdmission holds the admission block to its posture: required
// and well-formed under enforced-forge-hosted (#244), refused elsewhere.
func (c *Config) validateAdmission() error {
	if c.Posture != EnforcedForgeHosted {
		if c.Admission != nil {
			return fmt.Errorf("the admission block is valid under %s only (this declaration says %q)", EnforcedForgeHosted, c.Posture)
		}
		return nil
	}
	a := c.Admission
	if a == nil {
		return fmt.Errorf("%s requires the admission block {endpoint, identity, ledger_ref, checks, reviews, owners}: the service actors propose to and the identity the forge lets write the ledger branch", EnforcedForgeHosted)
	}
	u, err := url.Parse(a.Endpoint)
	if a.Endpoint == "" || err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("admission.endpoint must be an http(s) URL the admission service answers on, got %q", a.Endpoint)
	}
	if strings.TrimSpace(a.Identity) == "" {
		return errors.New("admission.identity must name the forge identity the admission service pushes as (the ledger branch's sole writer)")
	}
	if a.LedgerRef != "" && !strings.HasPrefix(a.LedgerRef, "refs/heads/") {
		return fmt.Errorf("admission.ledger_ref must be a branch (refs/heads/...), because forges protect branches and tags and nothing under refs/seed/*: got %q", a.LedgerRef)
	}
	if a.Reviews < 0 {
		return fmt.Errorf("admission.reviews must be zero or more, got %d", a.Reviews)
	}
	for _, chk := range a.Checks {
		if strings.TrimSpace(chk) == "" {
			return errors.New("admission.checks must not carry an empty check name")
		}
	}
	for _, o := range a.Owners {
		if strings.TrimSpace(o) == "" {
			return errors.New("admission.owners must not carry an empty identity")
		}
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
