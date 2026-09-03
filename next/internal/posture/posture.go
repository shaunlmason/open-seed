// Package posture holds the deployment's declared admission posture
// (SEED-NEXT.md Part II "Postures"; plans/os-3c72f93f.md): every
// deployment MUST declare which of the three named deployments it runs,
// and cooperative is a declared mode with a named consequence the
// preflight tool states in plain words. The declaration is
// client/deployment state, never ledger content: the guarded ref's tree
// stays HEAD and segments only.
//
// Under enforced-forge-hosted the declaration also carries the
// admission block (plans/os-5c8a312c.md D3, D5): where the admission
// service listens, which forge identity it pushes as, which branch the
// ledger rides, and the protections the reconciler derives the forge's
// desired state from. The block is required under that posture and
// refused under the others, so a declaration never says two things.
package posture

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
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

// DefaultLedgerRef is the ledger ref a forge-hosted deployment rides
// when its declaration names none: a branch, because forges protect
// branches and tags and nothing under refs/seed/* (plans/os-5c8a312c.md
// D4). The hook posture keeps refs/seed/ledger untouched.
const DefaultLedgerRef = "refs/heads/seed-ledger"

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

// Admission is the forge-hosted deployment's admission block. Endpoint
// is where `seed-admit serve` answers proposals; Identity is the forge
// login or app slug the service's git credential belongs to (the
// ledger branch's sole writer); LedgerRef is the branch the ledger
// rides (DefaultLedgerRef when empty); Checks and Reviews are the
// default branch's required status checks and review count; Owners
// are the forge identities that review the protected surface, rendered
// into CODEOWNERS by the reconciler.
type Admission struct {
	Endpoint  string   `json:"endpoint"`
	Identity  string   `json:"identity"`
	LedgerRef string   `json:"ledger_ref"`
	Checks    []string `json:"checks"`
	Reviews   int      `json:"reviews"`
	Owners    []string `json:"owners"`
}

// Config is the deployment declaration.
type Config struct {
	Posture   Posture    `json:"posture"`
	Admission *Admission `json:"admission,omitempty"`
}

// LedgerRef is the ref the ledger rides under this declaration: the
// admission block's branch under enforced-forge-hosted, or "" for the
// other postures, whose callers keep their own default.
func (c *Config) LedgerRef() string {
	if c.Posture != EnforcedForgeHosted || c.Admission == nil {
		return ""
	}
	if c.Admission.LedgerRef == "" {
		return DefaultLedgerRef
	}
	return c.Admission.LedgerRef
}

// Load reads a strict declaration {"posture": "<name>", ...}: unknown
// fields, trailing data, unknown postures, and an admission block that
// disagrees with the posture refuse; a missing file is ErrUndeclared.
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

// Parse decodes a declaration's bytes under Load's rules.
func Parse(b []byte) (*Config, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var c Config
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("posture declaration does not parse: %v (valid postures: %s)", err, validList())
	}
	if dec.More() {
		return nil, fmt.Errorf("posture declaration carries trailing data (valid postures: %s)", validList())
	}
	if !c.Posture.Valid() {
		return nil, fmt.Errorf("%q is not a Seed posture (valid postures: %s)", c.Posture, validList())
	}
	if err := c.validateAdmission(); err != nil {
		return nil, err
	}
	return &c, nil
}

// validateAdmission holds the block to its posture: required under
// enforced-forge-hosted with an endpoint, an identity and a branch-
// namespace ledger ref; refused under every other posture, because a
// block nothing consults is a declaration that lies about its shape.
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
