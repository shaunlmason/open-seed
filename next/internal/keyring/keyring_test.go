package keyring_test

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/genesis"
	"github.com/shaunlmason/open-seed/next/internal/keyring"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
	"github.com/shaunlmason/open-seed/next/internal/version"
)

func key(t testing.TB, first byte) ed25519.PrivateKey {
	t.Helper()
	seed := make([]byte, ed25519.SeedSize)
	seed[0] = first
	return ed25519.NewKeyFromSeed(seed)
}

func fp(t testing.TB, priv ed25519.PrivateKey) string {
	t.Helper()
	f, err := event.Fingerprint(priv.Public().(ed25519.PublicKey))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func rec(t testing.TB, priv ed25519.PrivateKey, verb, subject, payload string) *event.Record {
	t.Helper()
	r, err := event.Sign(event.Event{
		V: version.Seed1, TS: "2026-09-01T00:00:00Z", Actor: fp(t, priv),
		Verb: verb, Subject: subject, Payload: json.RawMessage(payload), Prev: event.EmptyHash,
	}, priv)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func enrollPayload(t testing.TB, priv ed25519.PrivateKey, kind, name string) string {
	t.Helper()
	pub := priv.Public().(ed25519.PublicKey)
	return fmt.Sprintf(`{"key": %q, "kind": %q, "name": %q}`, hex.EncodeToString(pub), kind, name)
}

func seeded(t testing.TB, root ed25519.PrivateKey, extra ...ed25519.PublicKey) *keyring.State {
	t.Helper()
	g, err := genesis.Build(root, extra, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	s := keyring.New()
	s.SeedGenesis(g)
	if !s.Seeded() {
		t.Fatal("genesis must seed the keyring")
	}
	return s
}

// conformance: III.E — every actor is a keypair and the keyring is a
// projection: the genesis governance root seeds it and only active
// standing resolves.
func TestSeedAndStandingResolution(t *testing.T) {
	root, worker := key(t, 1), key(t, 2)
	s := seeded(t, root)
	if _, ok := s.Resolve(fp(t, root)); !ok {
		t.Fatal("the governance root must resolve")
	}
	if !s.IsActiveRoot(fp(t, root)) {
		t.Fatal("the governance root must be an active root")
	}
	if _, ok := s.Resolve(fp(t, worker)); ok {
		t.Fatal("an unenrolled key must not resolve")
	}

	if err := s.Advance(rec(t, root, "actor.enrolled", fp(t, worker), enrollPayload(t, worker, "agent", "worker"))); err != nil {
		t.Fatalf("enrollment must apply: %v", err)
	}
	if _, ok := s.Resolve(fp(t, worker)); !ok {
		t.Fatal("an enrolled key must resolve")
	}
	if s.IsActiveRoot(fp(t, worker)) {
		t.Fatal("an enrolled worker is not a governance root")
	}
	e, ok := s.Get(fp(t, worker))
	if !ok || e.Kind != "agent" || e.Name != "worker" || e.Standing != keyring.StandingActive {
		t.Fatalf("enrollment entry wrong: %+v", e)
	}
}

// The standing lifecycle: suspension pauses resolution, re-enrollment
// reinstates, revocation is terminal (recorded decisions,
// plans/os-52a2d688.md).
func TestStandingLifecycle(t *testing.T) {
	root, worker := key(t, 1), key(t, 2)
	s := seeded(t, root)
	wfp := fp(t, worker)
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	refuse := func(err error, phrase string) {
		t.Helper()
		if err == nil || !strings.Contains(err.Error(), phrase) {
			t.Fatalf("want refusal containing %q, got %v", phrase, err)
		}
	}

	must(s.Advance(rec(t, root, "actor.enrolled", wfp, enrollPayload(t, worker, "agent", "worker"))))
	refuse(s.Advance(rec(t, root, "actor.enrolled", wfp, enrollPayload(t, worker, "agent", "worker"))), "already enrolled")

	must(s.Advance(rec(t, root, "actor.suspended", wfp, `{"reason": "drift"}`)))
	if _, ok := s.Resolve(wfp); ok {
		t.Fatal("a suspended key must not resolve")
	}
	refuse(s.Advance(rec(t, root, "actor.suspended", wfp, `{"reason": "again"}`)), "already suspended")
	must(s.Advance(rec(t, root, "actor.granted", wfp, `{"capability": "claim:core"}`)))

	must(s.Advance(rec(t, root, "actor.enrolled", wfp, enrollPayload(t, worker, "agent", "worker-2"))))
	if e, _ := s.Get(wfp); e.Standing != keyring.StandingActive || e.Name != "worker-2" {
		t.Fatalf("re-enrollment must reinstate, got %+v", e)
	}
	if e, _ := s.Get(wfp); len(e.Grants) != 1 || e.Grants[0] != "claim:core" {
		t.Fatalf("grants must accumulate as data, got %+v", e.Grants)
	}

	must(s.Advance(rec(t, root, "actor.revoked", wfp, `{"reason": "compromise"}`)))
	if _, ok := s.Resolve(wfp); ok {
		t.Fatal("a revoked key must not resolve")
	}
	refuse(s.Advance(rec(t, root, "actor.enrolled", wfp, enrollPayload(t, worker, "agent", "worker"))), "terminal")
	refuse(s.Advance(rec(t, root, "actor.suspended", wfp, `{"reason": "x"}`)), "terminal")
	refuse(s.Advance(rec(t, root, "actor.granted", wfp, `{"capability": "x"}`)), "terminal")
}

// Strict payload shapes: unknown fields, trailing data, malformed keys,
// bad kinds, and subject mismatches all refuse through the one
// transition function.
func TestAdvanceShapeRefusals(t *testing.T) {
	root, worker := key(t, 1), key(t, 2)
	wpub := hex.EncodeToString(worker.Public().(ed25519.PublicKey))
	wfp := fp(t, worker)
	// A payload with trailing data cannot be signed at all (JCS
	// canonicalization refuses it), so that strict-parse branch is
	// unrepresentable in a record; every representable malformation is
	// here.
	cases := map[string]struct{ verb, subject, payload string }{
		"unknown field":    {"actor.enrolled", wfp, `{"key": "` + wpub + `", "kind": "agent", "name": "w", "mode": "x"}`},
		"bad key hex":      {"actor.enrolled", wfp, `{"key": "zz", "kind": "agent", "name": "w"}`},
		"bad kind":         {"actor.enrolled", wfp, `{"key": "` + wpub + `", "kind": "robot", "name": "w"}`},
		"empty name":       {"actor.enrolled", wfp, `{"key": "` + wpub + `", "kind": "agent", "name": ""}`},
		"subject mismatch": {"actor.enrolled", "c-0001", `{"key": "` + wpub + `", "kind": "agent", "name": "w"}`},
		"empty capability": {"actor.granted", wfp, `{"capability": ""}`},
		"empty reason":     {"actor.suspended", wfp, `{"reason": ""}`},
		"unknown verb":     {"actor.qualified", wfp, `{"tuple": "x"}`},
		"not enrolled":     {"actor.revoked", "c-9999", `{"reason": "x"}`},
	}
	for name, c := range cases {
		s := seeded(t, root)
		if err := s.Advance(rec(t, root, c.verb, c.subject, c.payload)); err == nil {
			t.Errorf("%s must refuse", name)
		}
	}
}

// Root liveness (review finding on #97): no admitted transition may
// leave the keyring without an active governance root.
func TestRootLiveness(t *testing.T) {
	root1, root2 := key(t, 1), key(t, 3)
	s := seeded(t, root1)
	if err := s.Advance(rec(t, root1, "actor.revoked", fp(t, root1), `{"reason": "x"}`)); err == nil || !strings.Contains(err.Error(), "root liveness") {
		t.Fatalf("revoking the only root must refuse with root liveness, got %v", err)
	}
	if err := s.Advance(rec(t, root1, "actor.suspended", fp(t, root1), `{"reason": "x"}`)); err == nil {
		t.Fatal("suspending the only root must refuse")
	}

	s = seeded(t, root1, root2.Public().(ed25519.PublicKey))
	if err := s.Advance(rec(t, root1, "actor.suspended", fp(t, root1), `{"reason": "rotate"}`)); err != nil {
		t.Fatalf("suspending one of two roots must apply: %v", err)
	}
	if err := s.Advance(rec(t, root2, "actor.revoked", fp(t, root2), `{"reason": "x"}`)); err == nil {
		t.Fatal("ending the last remaining active root must refuse")
	}
	if err := s.Advance(rec(t, root2, "actor.enrolled", fp(t, root1), enrollPayload(t, root1, "human", "op"))); err != nil {
		t.Fatalf("re-enrolling a suspended root must reinstate it: %v", err)
	}
	if !s.IsActiveRoot(fp(t, root1)) {
		t.Fatal("a reinstated root keeps its root standing")
	}
	if err := s.Advance(rec(t, root2, "actor.revoked", fp(t, root2), `{"reason": "rotate"}`)); err != nil {
		t.Fatalf("revoking a root with another active must apply: %v", err)
	}
}

// StateAt tracks the protocol version the way verification does: actor
// events at seed/0 positions stay inert (the grandfathering boundary),
// and the upgrade verb switches enforcement on. The upgrade and genesis
// verb literals keyring mirrors are exercised against the real
// ledger.UpgradeVerb and genesis.Verb here, so a drift fails this test.
func TestStateAtVersionBoundary(t *testing.T) {
	root, worker := key(t, 1), key(t, 2)
	g, err := genesis.Build(root, nil, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	seed0 := func(verb, subject, payload string) *event.Record {
		r, err := event.Sign(event.Event{
			V: version.Protocol, TS: "2026-09-01T00:00:00Z", Actor: fp(t, root),
			Verb: verb, Subject: subject, Payload: json.RawMessage(payload), Prev: event.EmptyHash,
		}, root)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	junk := seed0("actor.enrolled", "c-0001", `{"garbage": true}`)
	up := seed0(ledger.UpgradeVerb, "system", `{"to": "`+version.Seed1+`"}`)
	enroll := rec(t, root, "actor.enrolled", fp(t, worker), enrollPayload(t, worker, "agent", "worker"))

	s, active, err := keyring.StateAt([]*event.Record{g, junk, up, enroll})
	if err != nil {
		t.Fatalf("grandfathered junk must stay inert: %v", err)
	}
	if active != version.Seed1 {
		t.Fatalf("StateAt must track the upgrade, got %q", active)
	}
	if _, ok := s.Resolve(fp(t, worker)); !ok {
		t.Fatal("the post-upgrade enrollment must apply")
	}
	if _, ok := s.Get("c-0001"); ok {
		t.Fatal("the seed/0 junk event must have no keyring effect")
	}

	bad := rec(t, root, "actor.enrolled", "c-0002", `{"garbage": true}`)
	if _, _, err := keyring.StateAt([]*event.Record{g, up, bad}); err == nil || !strings.Contains(err.Error(), "position 2") {
		t.Fatalf("an illegal actor event at a seed/1 position must fail with its position, got %v", err)
	}
}

// Preview is admission's dry run: it refuses what Advance would refuse
// and never mutates the real state.
func TestCloneAndPreviewIsolation(t *testing.T) {
	root, worker := key(t, 1), key(t, 2)
	s := seeded(t, root)
	good := rec(t, root, "actor.enrolled", fp(t, worker), enrollPayload(t, worker, "agent", "worker"))
	if err := s.Preview(good); err != nil {
		t.Fatalf("preview of a legal event must pass: %v", err)
	}
	if _, ok := s.Resolve(fp(t, worker)); ok {
		t.Fatal("preview must not mutate the state")
	}
	if err := s.Preview(rec(t, root, "actor.revoked", fp(t, root), `{"reason": "x"}`)); err == nil {
		t.Fatal("preview must refuse what Advance refuses")
	}

	c := s.Clone()
	if err := c.Advance(good); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Resolve(fp(t, worker)); ok {
		t.Fatal("advancing a clone must not touch the original")
	}
}

func TestUnseededKeyringRefusesActorEvents(t *testing.T) {
	root := key(t, 1)
	s := keyring.New()
	if s.Seeded() {
		t.Fatal("a fresh keyring is unseeded")
	}
	err := s.Advance(rec(t, root, "actor.enrolled", fp(t, root), enrollPayload(t, root, "human", "op")))
	if err == nil || !strings.Contains(err.Error(), "governance root") {
		t.Fatalf("an unseeded keyring must refuse actor events, got %v", err)
	}
	if err := s.Advance(rec(t, root, "progress.milestone", "c-0001", `{"n": 1}`)); err != nil {
		t.Fatalf("non-actor verbs no-op even unseeded: %v", err)
	}
}

func TestAppliesOnlyAtSeed1(t *testing.T) {
	if keyring.Applies(version.Protocol) || !keyring.Applies(version.Seed1) || keyring.Applies("seed/9") {
		t.Fatal("keyring semantics activate exactly at seed/1")
	}
	if !keyring.IsActorVerb("actor.enrolled") || keyring.IsActorVerb("progress.milestone") {
		t.Fatal("actor verb detection is namespace-based")
	}
}
