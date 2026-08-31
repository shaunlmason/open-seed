// Package keyring is the actor standing projection over a chain prefix
// (docs/next-build-plan.md Phase 3 item 1; plans/os-52a2d688.md;
// SEED-NEXT.md Part II "Enrollment" and "Capabilities"). Enrollment,
// grants, suspension, and revocation are ledger events; the keyring every
// verifier and admission point consults is a pure projection of them —
// seeded from the genesis governance root and advanced by one transition
// function (Advance) shared by verification replay and admission preview,
// never stored anywhere. The semantics activate at protocol version
// seed/1 per the bump discipline in next/spec/protocol.md: actor events
// at seed/0 positions are grandfathered as inert, so chains that verified
// before Phase 3 still verify.
package keyring

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/version"
)

// The actor lifecycle verbs from the charter's catalog
// (next/spec/protocol.md; payload schemas in next/spec/actors.md).
const (
	VerbEnrolled  = "actor.enrolled"
	VerbGranted   = "actor.granted"
	VerbSuspended = "actor.suspended"
	VerbRevoked   = "actor.revoked"
)

// These mirror ledger.UpgradeVerb, genesis.Verb, halt.DeclareVerb, and
// halt.LiftVerb; keyring cannot import those packages (ledger imports
// keyring), so the parity is pinned by tests.
const (
	upgradeVerb      = "system.protocol.upgraded"
	genesisVerb      = "system.genesis"
	haltDeclaredVerb = "system.halt.declared"
	haltLiftedVerb   = "system.halt.lifted"
	checkpointVerb   = "system.checkpoint"
)

// The capability vocabulary (plans/os-3979d48b.md; SEED-NEXT.md Part II
// "Capabilities"): grants are events, checked at admission on every
// verb.
const (
	CapOperator    = "operator"
	CapMaintenance = "maintenance"
	// CapDispatch is queue management: filing, specifying, blocking,
	// unblocking, and reaping contracts (plans/os-d69a6c91.md).
	CapDispatch = "dispatch"
	// CapObserver is the observer lane (plans/os-6cdc15be.md): a
	// governed observer records forge fact (merge.observed) behind
	// the full chain rule; the charter names merge.observed an
	// observation by a governed observer, and Phase 6 adds the lane.
	CapObserver = "observer"
	// CapVerdict is the verifier lane: rendering verdicts
	// (plans/os-f6d2c267.md). Deliberately the one row without the
	// operator fallback: III.G names operator override its own
	// attributable verb, never a disguised verdict — that verb lands
	// with 6.4, and a governance root that judges holds an explicit
	// verdict grant, with L1 independence applying to every signer.
	CapVerdict = "verdict"
	// CapClaim is the worker set: taking, releasing, and parking
	// claims, and submitting work.
	CapClaim = "claim"
	// CapSealer authors sealed checks (plans/os-3128535a.md). Like the
	// verdict lane it has no operator fallback: operator already
	// stands in the claim and submission lanes, so an operator row
	// here would put authoring and implementation authority on one
	// capability and the charter's capability audit could prove
	// nothing. Grant-level disjointness with claim and operator is
	// enforced at actor.granted admission.
	CapSealer = "sealer"
	// CapSupervise is the supervisor lane (plans/os-c61c3392.md;
	// SEED-NEXT.md §II.9): publishing eligibility-scoped offers. No
	// disjointness constraints attach — an offer grants nothing, the
	// claim it invites settles at admission like any claim — so the
	// row keeps the standard operator fallback.
	CapSupervise = "supervise"
)

// AcceptedCapabilities returns the set of capabilities any one of which
// admits the verb, mirroring the normative table in
// next/spec/actors.md "Capabilities" (pinned by test). A nil result
// means the verb needs active standing only. The table is data:
// later phases append rows (claim rights by squad and tier, verdict
// rights, curation-proposal rights) when their verbs land.
func AcceptedCapabilities(verb string) []string {
	if IsActorVerb(verb) {
		return []string{CapOperator}
	}
	switch verb {
	case haltDeclaredVerb, haltLiftedVerb, upgradeVerb:
		return []string{CapOperator}
	case checkpointVerb:
		// The charter names checkpoints as signed by the maintenance
		// actor or an operator; folding maintenance into operator would
		// hand the Phase 9 loop halt and actor-management authority it
		// must not hold (review finding on #101).
		return []string{CapMaintenance, CapOperator}
	// The contract-lifecycle rows (plans/os-d69a6c91.md, review
	// finding on #113: a verb without a row needs active standing
	// only, which would let any enrolled actor cancel or specify
	// anything). Dispatch manages the queue; claim is the worker set;
	// cancellation and the done observation stay operator-gated in v0
	// (Phase 6 adds the observer lane for merge.observed).
	case "intent.filed", "contract.specified", "contract.blocked",
		"contract.unblocked", "claim.reaped":
		return []string{CapDispatch, CapOperator}
	case "claim.taken", "claim.released", "claim.parked", "submission.made":
		return []string{CapClaim, CapOperator}
	case "contract.cancelled":
		return []string{CapOperator}
	// The reconciliation chain (plans/os-6cdc15be.md): asking for the
	// merge is the work lane's act; observing the forge fact is the
	// observer lane's.
	case "merge.requested":
		return []string{CapClaim, CapOperator}
	case "merge.observed":
		return []string{CapObserver, CapOperator}
	// The sealed-checks commitment (plans/os-3128535a.md): sealer
	// only, no operator fallback, mirroring the verdict lane's
	// posture — authoring isolation is the row's whole point.
	case "check.sealed":
		return []string{CapSealer}
	// The red-verdict companions (plans/os-d2497eb7.md): returning a
	// fail-verdicted contract to the queue is queue management; the
	// override is the third no-fallback row — its own attributable
	// verb, operator judgment and nothing else.
	case "contract.returned":
		return []string{CapDispatch, CapOperator}
	case "merge.overridden":
		return []string{CapOperator}
	// The supervisor lane (plans/os-c61c3392.md): offers invite
	// claims and grant nothing, so the standard operator fallback
	// stands.
	case "offer.published":
		return []string{CapSupervise, CapOperator}
	// The budget-reservation facts (plans/os-cecac5de.md): the claim
	// lane reserves and settles inside its window; the budget rule
	// further pins reserves to the ACTIVE holder and closes to the
	// reservation's owner, which capability rows alone cannot say.
	case "budget.reserve", "budget.settle", "budget.release":
		return []string{CapClaim, CapOperator}
	// The execution-run facts (plans/os-1dad487d.md; the safe-point
	// interrupt per plans/os-0f718b4e.md): adapter-side initiation,
	// summarization, and preemption are the supervisor lane's acts,
	// like offers.
	case "run.started", "run.settled", "run.interrupted":
		return []string{CapSupervise, CapOperator}
	// The plan verbs (plans/os-16c1d142.md): the claim holder plans
	// (the fence matrix applies on a claimed subject); approval is an
	// external-fact observation, operator-attested in v0 like
	// merge.observed.
	case "plan.proposed":
		return []string{CapClaim, CapOperator}
	case "plan.approved":
		return []string{CapOperator}
	// The observation summarization verbs (plans/os-2ff8dbf1.md): a
	// milestone is the claim lane's coarse fact (the fence matrix
	// applies on the claimed subject); declaring a wedge is operator
	// judgment in v0, the merge.observed posture.
	case "progress.milestone":
		return []string{CapClaim, CapOperator}
	case "wedge.declared":
		return []string{CapOperator}
	// The verdict lane (plans/os-f6d2c267.md): verdict-granted keys
	// only, no operator fallback — the one such row, see CapVerdict.
	case "verdict.rendered":
		return []string{CapVerdict}
	}
	return nil
}

// Applies reports whether the keyring semantics are active under the
// given protocol version: seed/1 introduced them (next/spec/actors.md),
// and records at earlier positions are grandfathered as inert.
func Applies(active string) bool { return active == version.Seed1 }

// IsActorVerb reports whether the verb is in the actor.* namespace.
func IsActorVerb(verb string) bool { return strings.HasPrefix(verb, "actor.") }

// Standing is an actor's current standing in the projection.
type Standing string

const (
	StandingActive    Standing = "active"
	StandingSuspended Standing = "suspended"
	StandingRevoked   Standing = "revoked"
)

// Entry is one actor's projected state. Kind is the enrolling operator's
// assertion, never a cryptographic fact (SEED-NEXT.md Part II
// "Enrollment"); Grants accumulate as capability data for the Phase 3.2
// admission checks.
type Entry struct {
	Key      ed25519.PublicKey
	Kind     string
	Name     string
	Standing Standing
	Root     bool
	Grants   []string
}

// State is the keyring at one chain position.
type State struct {
	entries map[string]*Entry
	seeded  bool
}

// New returns an empty, unseeded keyring.
func New() *State { return &State{entries: map[string]*Entry{}} }

// Seeded reports whether a governance root has been loaded. An unseeded
// keyring refuses every actor event: standing has no anchor without one.
func (s *State) Seeded() bool { return s.seeded }

// SeedGenesis loads the governance root out of a system.genesis record's
// payload (the same schema internal/genesis owns; entries are taken
// verbatim, since genesis.Bootstrap already enforces the
// fingerprint-to-key binding on every CLI path). It never fails a chain:
// an unparseable payload just leaves the keyring unseeded, and the
// refusals then surface where standing is actually consulted.
func (s *State) SeedGenesis(rec *event.Record) {
	var p struct {
		GovernanceRoot []struct {
			Fingerprint string `json:"fingerprint"`
			PublicKey   string `json:"public_key"`
		} `json:"governance_root"`
	}
	if err := json.Unmarshal(rec.Event.Payload, &p); err != nil {
		return
	}
	for _, rk := range p.GovernanceRoot {
		raw, err := hex.DecodeString(rk.PublicKey)
		if err != nil || len(raw) != ed25519.PublicKeySize {
			continue
		}
		s.entries[rk.Fingerprint] = &Entry{Key: raw, Standing: StandingActive, Root: true}
		s.seeded = true
	}
}

// Resolve maps a fingerprint to its public key iff the actor's standing
// is active: the standing-aware half of signature resolution.
func (s *State) Resolve(fp string) (ed25519.PublicKey, bool) {
	e := s.entries[fp]
	if e == nil || e.Standing != StandingActive {
		return nil, false
	}
	return e.Key, true
}

// Resolver adapts Resolve to the ledger.Resolver shape.
func (s *State) Resolver() func(string) (ed25519.PublicKey, bool) {
	return s.Resolve
}

// Get returns a copy of an actor's entry.
func (s *State) Get(fp string) (Entry, bool) {
	e := s.entries[fp]
	if e == nil {
		return Entry{}, false
	}
	cp := *e
	cp.Grants = append([]string(nil), e.Grants...)
	return cp, true
}

// IsActiveRoot reports whether the fingerprint is a governance root in
// active standing.
func (s *State) IsActiveRoot(fp string) bool {
	e := s.entries[fp]
	return e != nil && e.Root && e.Standing == StandingActive
}

// sealerDisjoint refuses a grant that would co-hold sealed-check
// authoring and implementation authority on one key: sealer cannot
// join claim or operator (a governance root's implicit operator
// standing included), and neither can join sealer.
func sealerDisjoint(cur *Entry, granting string) error {
	implLane := map[string]bool{CapClaim: true, CapOperator: true}
	has := func(c string) bool {
		if c == CapOperator && cur.Root {
			return true
		}
		for _, g := range cur.Grants {
			if g == c {
				return true
			}
		}
		return false
	}
	if granting == CapSealer && (has(CapClaim) || has(CapOperator)) {
		return errors.New("sealer cannot be granted to a key holding claim or operator — sealed checks are authored under a grant disjoint from implementation grants (plans/os-3128535a.md)")
	}
	if implLane[granting] && has(CapSealer) {
		return fmt.Errorf("%s cannot be granted to a key holding sealer — sealed checks are authored under a grant disjoint from implementation grants (plans/os-3128535a.md)", granting)
	}
	return nil
}

// Granted returns the active entries holding the capability, sorted by
// fingerprint: the sealed-check recipient set is "the current verifier
// keyring", and rotation re-derives it from here.
func (s *State) Granted(capability string) []Entry {
	var out []Entry
	fps := make([]string, 0, len(s.entries))
	for fp := range s.entries {
		fps = append(fps, fp)
	}
	sort.Strings(fps)
	for _, fp := range fps {
		e := s.entries[fp]
		if e.Standing != StandingActive {
			continue
		}
		for _, g := range e.Grants {
			if g == capability {
				out = append(out, *e)
				break
			}
		}
	}
	return out
}

// HasAnyCapability reports whether the actor holds any of the listed
// capabilities: governance roots hold operator implicitly (the genesis
// trust anchor a deployment's first grants must come from), enrolled
// actors hold exactly what actor.granted accumulated, and only active
// standing counts — a suspended or revoked actor holds nothing.
func (s *State) HasAnyCapability(fp string, capabilities []string) bool {
	e := s.entries[fp]
	if e == nil || e.Standing != StandingActive {
		return false
	}
	for _, want := range capabilities {
		if want == CapOperator && e.Root {
			return true
		}
		for _, g := range e.Grants {
			if g == want {
				return true
			}
		}
	}
	return false
}

func (s *State) activeRoots() int {
	n := 0
	for _, e := range s.entries {
		if e.Root && e.Standing == StandingActive {
			n++
		}
	}
	return n
}

// Clone deep-copies the state.
func (s *State) Clone() *State {
	c := New()
	c.seeded = s.seeded
	for fp, e := range s.entries {
		cp := *e
		cp.Grants = append([]string(nil), e.Grants...)
		c.entries[fp] = &cp
	}
	return c
}

// Preview applies one record to a clone: admission's dry run of the
// shared transition function, leaving the real state untouched.
func (s *State) Preview(rec *event.Record) error {
	return s.Clone().Advance(rec)
}

func strict(payload json.RawMessage, into any) error {
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.DisallowUnknownFields()
	if err := dec.Decode(into); err != nil {
		return err
	}
	if dec.More() {
		return errors.New("trailing data")
	}
	return nil
}

// Advance applies one record's actor-event effect: the one transition
// function, owning payload shapes, standing legality, and effects, that
// both verification replay and admission consume (the shared-rule-set
// requirement). Non-actor verbs no-op. Callers gate it on Applies: at
// seed/0 positions actor events are grandfathered and Advance is not
// called.
func (s *State) Advance(rec *event.Record) error {
	e := &rec.Event
	if !IsActorVerb(e.Verb) {
		return nil
	}
	if !s.seeded {
		return errors.New("keyring has no governance root: the chain's genesis names none")
	}
	switch e.Verb {
	case VerbEnrolled:
		var p struct {
			Key  string `json:"key"`
			Kind string `json:"kind"`
			Name string `json:"name"`
		}
		if err := strict(e.Payload, &p); err != nil {
			return fmt.Errorf("%s payload: %v", e.Verb, err)
		}
		raw, err := hex.DecodeString(p.Key)
		if err != nil || len(raw) != ed25519.PublicKeySize {
			return fmt.Errorf("%s key must be the raw 32-byte ed25519 public key in hex", e.Verb)
		}
		if p.Kind != "human" && p.Kind != "agent" && p.Kind != "service" {
			return fmt.Errorf("%s kind %q is not one of human, agent, service", e.Verb, p.Kind)
		}
		if p.Name == "" {
			return fmt.Errorf("%s requires a display name", e.Verb)
		}
		fp, err := event.Fingerprint(ed25519.PublicKey(raw))
		if err != nil {
			return err
		}
		if e.Subject != fp {
			return fmt.Errorf("%s subject %q does not match the enrolled key's fingerprint %s", e.Verb, e.Subject, fp)
		}
		switch cur := s.entries[fp]; {
		case cur == nil:
			s.entries[fp] = &Entry{Key: raw, Kind: p.Kind, Name: p.Name, Standing: StandingActive}
		case cur.Standing == StandingRevoked:
			return fmt.Errorf("actor %s is revoked; revocation is terminal", fp)
		case cur.Standing == StandingSuspended:
			// Re-enrollment reinstates a suspended actor (recorded
			// decision, plans/os-52a2d688.md).
			cur.Standing = StandingActive
			cur.Kind = p.Kind
			cur.Name = p.Name
		default:
			return fmt.Errorf("actor %s is already enrolled and active", fp)
		}
	case VerbGranted:
		var p struct {
			Capability string `json:"capability"`
		}
		if err := strict(e.Payload, &p); err != nil {
			return fmt.Errorf("%s payload: %v", e.Verb, err)
		}
		if p.Capability == "" {
			return fmt.Errorf("%s must name a capability", e.Verb)
		}
		cur := s.entries[e.Subject]
		if cur == nil {
			return fmt.Errorf("%s subject %s is not enrolled", e.Verb, e.Subject)
		}
		if cur.Standing == StandingRevoked {
			return fmt.Errorf("actor %s is revoked; revocation is terminal", e.Subject)
		}
		// Sealed-check authoring isolation (plans/os-3128535a.md):
		// sealer and the implementation lanes (claim, and operator,
		// which stands in for claim) are disjoint at the grant, both
		// directions, so the capability audit has something to prove.
		if err := sealerDisjoint(cur, p.Capability); err != nil {
			return fmt.Errorf("%s to %s: %v", e.Verb, e.Subject, err)
		}
		cur.Grants = append(cur.Grants, p.Capability)
	case VerbSuspended, VerbRevoked:
		var p struct {
			Reason string `json:"reason"`
		}
		if err := strict(e.Payload, &p); err != nil {
			return fmt.Errorf("%s payload: %v", e.Verb, err)
		}
		if p.Reason == "" {
			return fmt.Errorf("%s must name a reason", e.Verb)
		}
		cur := s.entries[e.Subject]
		if cur == nil {
			return fmt.Errorf("%s subject %s is not enrolled", e.Verb, e.Subject)
		}
		if cur.Standing == StandingRevoked {
			return fmt.Errorf("actor %s is already revoked; revocation is terminal", e.Subject)
		}
		if e.Verb == VerbSuspended && cur.Standing == StandingSuspended {
			return fmt.Errorf("actor %s is already suspended", e.Subject)
		}
		if cur.Root && cur.Standing == StandingActive && s.activeRoots() == 1 {
			return errors.New("refusing to end the last active governance root's standing: the keyring must keep at least one active root (root liveness, plans/os-52a2d688.md)")
		}
		if e.Verb == VerbSuspended {
			cur.Standing = StandingSuspended
		} else {
			cur.Standing = StandingRevoked
		}
	default:
		return fmt.Errorf("actor verb %q is not defined at %s (actor.qualified lands with qualification, build plan Phase 10)", e.Verb, version.Seed1)
	}
	return nil
}

// StateAt projects the keyring over a prefix of verified records,
// tracking the active protocol version the same way verification does:
// genesis names the initial version, and system.protocol.upgraded
// switches it as the last event of the old version. It returns the state
// and the version active after the last record. The prefix must already
// have verified; an Advance error here means it had not.
func StateAt(records []*event.Record) (*State, string, error) {
	s := New()
	active := ""
	for pos, rec := range records {
		if active == "" {
			active = rec.Event.V
			if pos == 0 && rec.Event.Verb == genesisVerb && rec.Event.Subject == "system" {
				var g struct {
					Protocol string `json:"protocol"`
				}
				if err := json.Unmarshal(rec.Event.Payload, &g); err == nil && g.Protocol != "" {
					active = g.Protocol
				}
				s.SeedGenesis(rec)
			}
		}
		if Applies(active) {
			if err := s.Advance(rec); err != nil {
				return nil, "", fmt.Errorf("position %d: %v", pos, err)
			}
		}
		if rec.Event.Verb == upgradeVerb && rec.Event.Subject == "system" {
			var up struct {
				To string `json:"to"`
			}
			if err := json.Unmarshal(rec.Event.Payload, &up); err == nil && up.To != "" {
				active = up.To
			}
		}
	}
	if active == "" {
		active = version.Protocol
	}
	return s, active, nil
}
