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

// DeclarationPath is where the declaration lives: the repository
// root, read by the doctor and the remote verbs from the working tree
// and by the hook from the default branch's tip. It is on the
// protected surface by construction.
const DeclarationPath = "seed.json"

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

// Governance names the governance root and how the protected surface
// changes (charter §II.14; plans/os-0d4f2af3.md D1, D5). Root is the
// root key's fingerprint; Owners are the forge identities that review
// the protected surface; ChangeProcess is the one process the charter
// names.
type Governance struct {
	Root          string   `json:"root"`
	Owners        []string `json:"owners,omitempty"`
	ChangeProcess string   `json:"change_process"`
}

// ChangeProcessPROwnerReview is the only change process the charter
// names for the protected surface.
const ChangeProcessPROwnerReview = "pr+owner-review"

// SquadGuardrail is a squad's tier posture: the tier work files at by
// default and the highest tier an agent-kind key may claim.
type SquadGuardrail struct {
	Default  string `json:"default"`
	MaxAgent string `json:"max_agent"`
}

// PathFloor is the minimum tier a contract touching a prefix files at.
type PathFloor struct {
	Prefix string `json:"prefix"`
	Min    string `json:"min"`
}

// Guardrails is the declaration's guardrails block: tiers per squad and
// per path (charter III.L row 1), the agent ceiling reading the
// roster's kind (III.E row 9).
type Guardrails struct {
	Squads map[string]SquadGuardrail `json:"squads,omitempty"`
	Paths  []PathFloor               `json:"paths,omitempty"`
}

// Squad is one declared squad and the lane manifests it runs.
type Squad struct {
	Name  string   `json:"name"`
	Lanes []string `json:"lanes"`
}

// Teams is the declaration's teams block: the squads `routing` names.
type Teams struct {
	Squads []Squad `json:"squads"`
}

// Config is the deployment declaration. Protected is the protected
// surface as path prefixes (charter §II.14): what only the governance
// root may change, which the hook's code-ref half refuses for every
// other key and the reconciler renders into CODEOWNERS.
type Config struct {
	Posture     Posture      `json:"posture"`
	Admission   *Admission   `json:"admission,omitempty"`
	Protected   []string     `json:"protected,omitempty"`
	Checkpoints *Checkpoints `json:"checkpoints,omitempty"`
	// The preseed's blocks (plans/os-0d4f2af3.md D1): the protocol
	// version the deployment activates through, the governance root
	// and change process, the guardrails and the teams. Each is
	// undeclared when absent, never defaulted.
	Protocol   string      `json:"protocol,omitempty"`
	Governance *Governance `json:"governance,omitempty"`
	Guardrails *Guardrails `json:"guardrails,omitempty"`
	Teams      *Teams      `json:"teams,omitempty"`
}

// SquadNames lists the declared squads, sorted; nil when teams are
// undeclared.
func (c *Config) SquadNames() []string {
	if c == nil || c.Teams == nil {
		return nil
	}
	out := make([]string, 0, len(c.Teams.Squads))
	for _, s := range c.Teams.Squads {
		out = append(out, s.Name)
	}
	sort.Strings(out)
	return out
}

// AgentCeiling is the highest tier an agent-kind key may claim in the
// squad; ok is false when the guardrails or the squad are undeclared.
func (c *Config) AgentCeiling(squad string) (string, bool) {
	if c == nil || c.Guardrails == nil {
		return "", false
	}
	g, ok := c.Guardrails.Squads[squad]
	if !ok || g.MaxAgent == "" {
		return "", false
	}
	return g.MaxAgent, true
}

// Floor is the minimum tier a contract touching path files at: the
// strictest floor among the declared prefixes the path is under; ok is
// false when none applies.
func (c *Config) Floor(path string, above func(a, b string) bool) (string, bool) {
	if c == nil || c.Guardrails == nil {
		return "", false
	}
	path = strings.TrimSuffix(path, "/")
	floor, found := "", false
	for _, f := range c.Guardrails.Paths {
		prefix := strings.TrimSuffix(f.Prefix, "/")
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			if !found || above(f.Min, floor) {
				floor, found = f.Min, true
			}
		}
	}
	return floor, found
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
	if c.Checkpoints != nil && c.Checkpoints.Trust != TrustReplay && c.Checkpoints.Trust != TrustSigners {
		return nil, fmt.Errorf("checkpoints.trust is %q or %q (a fresh reader's verification obligation is declared, not defaulted), got %q", TrustReplay, TrustSigners, c.Checkpoints.Trust)
	}
	for _, p := range c.Protected {
		if !validProtectedEntry(p) {
			return nil, fmt.Errorf("protected entries are clean repository-relative path prefixes, got %q", p)
		}
	}
	if err := c.validatePreseed(); err != nil {
		return nil, err
	}
	return &c, nil
}

// validatePreseed holds the preseed blocks to their shapes: names
// non-empty and unique, tiers non-empty (the vocabulary is the tier
// table's and is checked where the table is, at admission and at
// `seed preseed check`), prefixes clean, the change process the one
// the charter names.
func (c *Config) validatePreseed() error {
	if c.Governance != nil {
		if strings.TrimSpace(c.Governance.Root) == "" {
			return errors.New("governance.root names the governance root's key fingerprint")
		}
		if c.Governance.ChangeProcess != ChangeProcessPROwnerReview {
			return fmt.Errorf("governance.change_process is %q, the one process the charter names for the protected surface, got %q", ChangeProcessPROwnerReview, c.Governance.ChangeProcess)
		}
		for _, o := range c.Governance.Owners {
			if strings.TrimSpace(o) == "" {
				return errors.New("governance.owners must not carry an empty identity")
			}
		}
	}
	if c.Guardrails != nil {
		for name, g := range c.Guardrails.Squads {
			if strings.TrimSpace(name) == "" {
				return errors.New("guardrails.squads must not carry an empty squad name")
			}
			if strings.TrimSpace(g.Default) == "" || strings.TrimSpace(g.MaxAgent) == "" {
				return fmt.Errorf("guardrails.squads.%s declares both default and max_agent tiers", name)
			}
		}
		for _, f := range c.Guardrails.Paths {
			if !validProtectedEntry(f.Prefix) || strings.TrimSpace(f.Min) == "" {
				return fmt.Errorf("guardrails.paths entries are clean repository-relative prefixes with a min tier, got %+v", f)
			}
		}
	}
	if c.Teams != nil {
		seen := map[string]bool{}
		for _, s := range c.Teams.Squads {
			if strings.TrimSpace(s.Name) == "" {
				return errors.New("teams.squads must not carry an empty squad name")
			}
			if seen[s.Name] {
				return fmt.Errorf("teams.squads names %q twice", s.Name)
			}
			seen[s.Name] = true
			if len(s.Lanes) == 0 {
				return fmt.Errorf("teams.squads.%s runs at least one lane manifest", s.Name)
			}
			for _, l := range s.Lanes {
				if strings.TrimSpace(l) == "" {
					return fmt.Errorf("teams.squads.%s names an empty lane", s.Name)
				}
			}
		}
	}
	return nil
}

// validProtectedEntry admits a clean, relative, forward-slash path: no
// empty or whitespace entry, no leading slash, no backslash, and
// nothing path.Clean would rewrite (".", "..", "../x", doubled
// separators), so the surface is compared by string with no surprises.
func validProtectedEntry(p string) bool {
	if strings.TrimSpace(p) == "" || strings.HasPrefix(p, "/") || strings.Contains(p, "\\") {
		return false
	}
	trimmed := strings.TrimSuffix(p, "/")
	if trimmed == "" || trimmed == "." || trimmed == ".." || strings.HasPrefix(trimmed, "../") {
		return false
	}
	return path.Clean(trimmed) == trimmed
}

// ProtectedSurface is the declared entries plus the declaration itself,
// deduplicated and sorted: the surface the hook refuses for every key
// without root standing and the reconciler renders into CODEOWNERS.
func (c *Config) ProtectedSurface() []string {
	seen := map[string]bool{DeclarationPath: true}
	out := []string{DeclarationPath}
	for _, p := range c.Protected {
		p = strings.TrimSuffix(p, "/")
		if p != "" && !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

// Protects reports whether p is on the protected surface: exactly an
// entry, or under one as a directory.
func (c *Config) Protects(p string) bool {
	p = strings.TrimSuffix(p, "/")
	for _, entry := range c.ProtectedSurface() {
		if p == entry || strings.HasPrefix(p, entry+"/") {
			return true
		}
	}
	return false
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
