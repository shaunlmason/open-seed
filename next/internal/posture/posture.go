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
	"os"
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

// Config is the deployment declaration.
type Config struct {
	Posture Posture `json:"posture"`
}

// Load reads a strict declaration {"posture": "<name>"}: unknown fields,
// trailing data, and unknown postures refuse; a missing file is
// ErrUndeclared.
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrUndeclared
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnreadable, err)
	}
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
	return &c, nil
}
