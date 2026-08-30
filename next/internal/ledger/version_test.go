package ledger

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/event"
)

func signedVersioned(t *testing.T, v, verb, subject, payload, prev string) *event.Record {
	t.Helper()
	priv := fixtureKey(t, 1)
	rec, err := event.Sign(event.Event{
		V: v, TS: "2026-09-01T00:00:00Z", Actor: fingerprintOf(t, priv),
		Verb: verb, Subject: subject, Payload: json.RawMessage(payload), Prev: prev,
	}, priv)
	if err != nil {
		t.Fatal(err)
	}
	return rec
}

func buildChain(t *testing.T, recs []*event.Record) *Store {
	t.Helper()
	s, err := Open(t.TempDir(), WithClock(clockAt("2026-09-01T00:00:00Z")))
	if err != nil {
		t.Fatal(err)
	}
	resolve := fixtureResolver(t, fixtureKey(t, 1))
	for _, rec := range recs {
		if _, err := s.Append(rec, resolve); err != nil {
			t.Fatal(err)
		}
	}
	return s
}

func chainWithUpgrade(t *testing.T) []*event.Record {
	t.Helper()
	a := signedVersioned(t, "seed/0", "progress.milestone", "c-0001", `{"n": 1}`, event.EmptyHash)
	ha, _ := a.Event.Hash()
	up := signedVersioned(t, "seed/0", UpgradeVerb, "system", `{"to": "seed/1"}`, ha)
	hup, _ := up.Event.Hash()
	b := signedVersioned(t, "seed/1", "progress.milestone", "c-0001", `{"n": 2}`, hup)
	return []*event.Record{a, up, b}
}

// conformance: III.A — verification runs against a declared
// supported-versions set with the version active at each chain position,
// so a valid older prefix replays after a protocol upgrade
// (next/spec/protocol.md; review finding on #75).
func TestUpgradedChainVerifiesWithSupportedSet(t *testing.T) {
	s := buildChain(t, chainWithUpgrade(t))
	resolve := fixtureResolver(t, fixtureKey(t, 1))
	rep, err := s.VerifyFromGenesis(resolve, WithSupportedVersions("seed/0", "seed/1"))
	if err != nil || rep.Count != 3 {
		t.Fatalf("upgraded chain must verify with both versions supported: %+v %v", rep, err)
	}
	if _, err := s.VerifyFromGenesis(resolve); err == nil {
		t.Fatal("default supported set (seed/0 only) must refuse the seed/1 suffix")
	}
}

func TestUnsupportedActiveVersionRefuses(t *testing.T) {
	s := buildChain(t, chainWithUpgrade(t))
	resolve := fixtureResolver(t, fixtureKey(t, 1))
	_, err := s.VerifyFromGenesis(resolve, WithSupportedVersions("seed/1"))
	var fail *Failure
	if !errors.As(err, &fail) || fail.Reason != ReasonVersionUnsupported || fail.Position != 0 {
		t.Fatalf("seed/0 prefix under seed/1-only support must refuse as %s@0, got %v", ReasonVersionUnsupported, err)
	}
}

// conformance: III.A — version mismatch refuses with a distinct reason: an
// event whose v differs from the active-at-position version breaks the
// discipline even when the version itself is supported.
func TestVersionMismatchWithoutUpgradeRefuses(t *testing.T) {
	a := signedVersioned(t, "seed/0", "progress.milestone", "c-0001", `{"n": 1}`, event.EmptyHash)
	ha, _ := a.Event.Hash()
	b := signedVersioned(t, "seed/1", "progress.milestone", "c-0001", `{"n": 2}`, ha)
	s := buildChain(t, []*event.Record{a, b})
	resolve := fixtureResolver(t, fixtureKey(t, 1))
	_, err := s.VerifyFromGenesis(resolve, WithSupportedVersions("seed/0", "seed/1"))
	var fail *Failure
	if !errors.As(err, &fail) || fail.Reason != ReasonVersionMismatch || fail.Position != 1 {
		t.Fatalf("want %s@1, got %v", ReasonVersionMismatch, err)
	}
}

func TestUpgradePayloadMustNameTarget(t *testing.T) {
	a := signedVersioned(t, "seed/0", "progress.milestone", "c-0001", `{"n": 1}`, event.EmptyHash)
	ha, _ := a.Event.Hash()
	up := signedVersioned(t, "seed/0", UpgradeVerb, "system", `{"note": "missing to"}`, ha)
	s := buildChain(t, []*event.Record{a, up})
	resolve := fixtureResolver(t, fixtureKey(t, 1))
	_, err := s.VerifyFromGenesis(resolve)
	var fail *Failure
	if !errors.As(err, &fail) || fail.Reason != ReasonBadPayload || fail.Position != 1 {
		t.Fatalf("upgrade without 'to' must refuse as %s@1, got %v", ReasonBadPayload, err)
	}
}

// conformance: III.A — the initial active version is the version genesis
// names in its payload; a genesis event whose own v disagrees refuses as a
// mismatch at position 0 instead of being accepted tautologically (#83
// review finding).
func TestGenesisPayloadBootstrapsActiveVersion(t *testing.T) {
	priv := fixtureKey(t, 1)
	lying := signedVersioned(t, "seed/1", "system.genesis", "system",
		`{"protocol": "seed/0", "governance_root": [{"fingerprint": "x", "public_key": "y"}]}`, event.EmptyHash)
	s := buildChain(t, []*event.Record{lying})
	resolve := fixtureResolver(t, priv)
	_, err := s.VerifyFromGenesis(resolve, WithSupportedVersions("seed/0", "seed/1"))
	var fail *Failure
	if !errors.As(err, &fail) || fail.Reason != ReasonVersionMismatch || fail.Position != 0 {
		t.Fatalf("genesis carrying a v that differs from its named protocol must refuse as %s@0, got %v", ReasonVersionMismatch, err)
	}
}
