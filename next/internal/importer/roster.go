package importer

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"sort"

	"github.com/shaunlmason/open-seed/next/internal/keyring"
)

// VerifierName is the dedicated import identity that renders the pass
// verdict when the v1 closer of a card was also one of its claimants:
// Seed's independence rule refuses a claimant's verdict, and the
// import does not manufacture independence a name never had.
const VerifierName = "import-verifier"

// actor is one import-generated identity: a v1 actor name, the key
// the importer generated for it, and the grants it needs.
type actor struct {
	name  string
	kind  string
	known bool
	s     *signer
	// verbs is every verb the identity signed in the pass, for the
	// manifest and the drills.
	verbs     map[string]bool
	grants    []string
	enrolled  int
	suspended int
	// operator marks the importing operator's own key, which signs
	// what no generated identity may (contract.cancelled, the card
	// reconciliations) and is neither enrolled nor suspended here.
	operator bool
}

// capFor is the one capability the import grants for a verb it
// synthesizes: never operator, which no generated key ever holds.
func capFor(verb string) string {
	switch verb {
	case "intent.filed", "contract.specified", "contract.blocked", "contract.unblocked", "contract.returned", "claim.reaped":
		return keyring.CapDispatch
	case "claim.taken", "claim.released", "claim.parked", "submission.made", "merge.requested":
		return keyring.CapClaim
	case "merge.observed":
		return keyring.CapObserver
	case "verdict.rendered":
		return keyring.CapVerdict
	}
	return ""
}

// grantsFor derives an identity's grants from the run-log before
// replay (plans/os-cf13fb51.md D3): the capabilities the events its
// v1 verbs become will consume, and nothing else. Operator-only verbs
// (contract.cancelled) are signed by the importing operator, so no
// generated key is ever granted operator; the verdict on a card its
// closer never claimed is the closer's, otherwise the import
// verifier's.
func grantsFor(name string, entries []Entry) []string {
	set := map[string]bool{}
	claimed := map[string]bool{}
	for _, e := range entries {
		if e.Actor == name && (e.Verb == "claim" || e.Verb == "release" || e.Verb == "transition") {
			claimed[e.Task] = true
		}
	}
	for _, e := range entries {
		if e.Actor != name || e.Task == "" {
			continue
		}
		switch e.Verb {
		case "create", "promote", "plan-unblock":
			set[keyring.CapDispatch] = true
		case "claim":
			set[keyring.CapClaim] = true
		case "release", "transition":
			// A move out of in_progress is the holder's claim act; one
			// between ready and blocked is dispatch; a release of a
			// claim the actor does not hold is claim.reaped (dispatch).
			set[keyring.CapClaim] = true
			set[keyring.CapDispatch] = true
		case "close", "accept":
			set[keyring.CapObserver] = true
			if !claimed[e.Task] {
				set[keyring.CapVerdict] = true
			}
		}
	}
	out := make([]string, 0, len(set))
	for c := range set {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

// roster is the identity plan: one actor per distinct v1 name, keys
// held in memory for the import only.
type roster struct {
	byName map[string]*actor
	order  []string
}

func newRoster(t *Table, names []string, entries []Entry) (*roster, error) {
	r := &roster{byName: map[string]*actor{}}
	set := map[string]bool{}
	for _, n := range names {
		set[n] = true
	}
	set[VerifierName] = true
	sorted := make([]string, 0, len(set))
	for n := range set {
		sorted = append(sorted, n)
	}
	sort.Strings(sorted)
	for _, n := range sorted {
		id, known := t.IdentityFor(n)
		if n == VerifierName {
			id, known = Identity{Kind: "service", Name: VerifierName}, true
		}
		_, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, err
		}
		s, err := newSigner(priv)
		if err != nil {
			return nil, err
		}
		grants := grantsFor(n, entries)
		if n == VerifierName {
			grants = []string{keyring.CapVerdict}
		}
		r.byName[n] = &actor{name: id.Name, kind: id.Kind, known: known, s: s, verbs: map[string]bool{}, grants: grants}
		r.order = append(r.order, n)
	}
	return r, nil
}

// get resolves a v1 actor name; a name the plan never saw (a card
// author absent from the run-log, say) is enrolled on the fly as the
// table resolves it, and listed.
func (r *roster) get(name string) *actor {
	if a, ok := r.byName[name]; ok {
		return a
	}
	return nil
}

func (r *roster) verifier() *actor { return r.byName[VerifierName] }

// reset clears what a pass recorded, keeping the keys and grants.
func (r *roster) reset() {
	for _, a := range r.byName {
		a.verbs = map[string]bool{}
		a.enrolled, a.suspended = 0, 0
	}
}

func (a *actor) pubHex() string { return hex.EncodeToString(a.s.pub) }
