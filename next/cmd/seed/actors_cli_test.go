package main

// End-to-end actor lifecycle through the CLI (plans/os-52a2d688.md step
// 5; conformance III.E): the root upgrades and enrolls over the wire and
// locally, the enrolled key appends, standing refusals surface at their
// envelope exits (14 out_of_grant, 3 invalid_transition before the
// upgrade), and a revoked key stops resolving.

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/version"
)

func writeWorkerKey(t *testing.T, first byte) (path string, pubHex, fp string) {
	t.Helper()
	seed := make([]byte, ed25519.SeedSize)
	seed[0] = first
	k := ed25519.NewKeyFromSeed(seed)
	block, err := ssh.MarshalPrivateKey(k, "worker-fixture")
	if err != nil {
		t.Fatal(err)
	}
	path = filepath.Join(t.TempDir(), "worker_ed25519")
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatal(err)
	}
	pub := k.Public().(ed25519.PublicKey)
	f, err := event.Fingerprint(pub)
	if err != nil {
		t.Fatal(err)
	}
	return path, hex.EncodeToString(pub), f
}

func enrollArg(pubHex string) string {
	return fmt.Sprintf(`{"key": %q, "kind": "agent", "name": "worker"}`, pubHex)
}

func TestActorLifecycleLocalCLI(t *testing.T) {
	dir, priv, _ := writeKeys(t)
	ld := filepath.Join(dir, "ledger")
	if _, code := runEnv(t, "init", "--ledger", ld, "--key", priv); code != 0 {
		t.Fatal("init failed")
	}
	wkey, wpub, wfp := writeWorkerKey(t, 7)

	if e, code := runEnv(t, "ledger", "append", "--ledger", ld, "--key", priv,
		"--verb", "system.protocol.upgraded", "--subject", "system", "--payload", `{"to": "`+version.Seed1+`"}`); code != 0 {
		t.Fatalf("upgrade failed: %d %+v", code, e)
	}
	if e, code := runEnv(t, "ledger", "append", "--ledger", ld, "--key", priv,
		"--verb", "actor.enrolled", "--subject", wfp, "--payload", enrollArg(wpub)); code != 0 {
		t.Fatalf("enrollment failed: %d %+v", code, e)
	}
	// The enrolled key appends: the keyring, not the genesis root set,
	// resolves the signer from seed/1.
	if e, code := runEnv(t, "ledger", "append", "--ledger", ld, "--key", wkey,
		"--verb", "progress.milestone", "--subject", "c-0001", "--payload", `{"n": 1}`); code != 0 {
		t.Fatalf("enrolled key must append: %d %+v", code, e)
	}
	if e, code := runEnv(t, "ledger", "verify", "--ledger", ld); code != 0 || e.Result["count"].(float64) != 4 {
		t.Fatalf("upgraded chain with an enrolled signer must verify: %d %+v", code, e)
	}

	// A malformed actor event refuses before anything is written: the
	// next replay would reject it, so the append must not create it.
	e, code := runEnv(t, "ledger", "append", "--ledger", ld, "--key", priv,
		"--verb", "actor.enrolled", "--subject", "c-0002", "--payload", `{"garbage": true}`)
	if code != 8 || e.Error == nil || !strings.Contains(e.Error.Message, "would fail verification") {
		t.Fatalf("a malformed actor event must refuse pre-write, got %d %+v", code, e)
	}
	if e, code := runEnv(t, "ledger", "verify", "--ledger", ld); code != 0 || e.Result["count"].(float64) != 4 {
		t.Fatalf("the refused event must leave the chain untouched: %d %+v", code, e)
	}

	// Revocation ends the worker's standing: its next local append fails
	// to resolve.
	if _, code := runEnv(t, "ledger", "append", "--ledger", ld, "--key", priv,
		"--verb", "actor.revoked", "--subject", wfp, "--payload", `{"reason": "compromise"}`); code != 0 {
		t.Fatal("revocation append failed")
	}
	e, code = runEnv(t, "ledger", "append", "--ledger", ld, "--key", wkey,
		"--verb", "progress.milestone", "--subject", "c-0003", "--payload", `{"n": 2}`)
	if code != 8 || e.Error == nil || !strings.Contains(e.Error.Message, "not resolvable") {
		t.Fatalf("a revoked key must not append, got %d %+v", code, e)
	}
}

func TestActorRemoteRefusalExits(t *testing.T) {
	_, priv, _ := writeKeys(t)
	wkey, wpub, wfp := writeWorkerKey(t, 7)
	_, tpub, tfp := writeWorkerKey(t, 8)

	remote := bareRemote(t)
	resolve := seedRemoteGenesis(t, remote)

	// Before the upgrade, actor verbs are not active: exit 3.
	e, code := runEnv(t, "ledger", "append", "--remote", remote, "--state", t.TempDir(),
		"--key", priv, "--verb", "actor.enrolled", "--subject", wfp, "--payload", enrollArg(wpub))
	if code != 3 || e.Error == nil || e.Error.Code != "invalid_transition" {
		t.Fatalf("an actor verb at a seed/0 tip must exit 3 invalid_transition, got %d %+v", code, e)
	}

	libAppend(t, remote, resolve, "seed/0", "system.protocol.upgraded", "system", `{"to": "`+version.Seed1+`"}`)
	if e, code := runEnv(t, "ledger", "append", "--remote", remote, "--state", t.TempDir(),
		"--key", priv, "--verb", "actor.enrolled", "--subject", wfp, "--payload", enrollArg(wpub)); code != 0 {
		t.Fatalf("root enrollment over the wire failed: %d %+v", code, e)
	}

	// The enrolled worker appends ordinary verbs but is refused actor
	// verbs at the newly allocated exit 14 (spec table row lands with
	// this change).
	if e, code := runEnv(t, "ledger", "append", "--remote", remote, "--state", t.TempDir(),
		"--key", wkey, "--verb", "progress.milestone", "--subject", "c-0001", "--payload", `{"n": 1}`); code != 0 {
		t.Fatalf("the enrolled key must append over the wire: %d %+v", code, e)
	}
	e, code = runEnv(t, "ledger", "append", "--remote", remote, "--state", t.TempDir(),
		"--key", wkey, "--verb", "actor.enrolled", "--subject", tfp, "--payload", enrollArg(tpub))
	if code != 14 || e.Error == nil || e.Error.Code != "out_of_grant" {
		t.Fatalf("a non-root actor verb must exit 14 out_of_grant, got %d %+v", code, e)
	}
	if !strings.Contains(e.Error.Message, "governance root") {
		t.Fatalf("the refusal must name the interim policy, got %+v", e.Error)
	}
}
