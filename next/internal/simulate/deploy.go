// Package simulate runs the whole system end to end against synthetic
// intents with a mock executor and zero credentials (SEED-NEXT.md
// §II.16; plans/os-16e55c11.md D3): a throwaway deployment under the
// declared posture, one identity per shipped lane and role, a seeded
// catalog of fix-the-check intents, and every lane driven through the
// real boundary — the loop lanes through internal/loop, the rest
// through the same CLI verbs an operator runs. No forge, no model, no
// network beyond a local bare git remote.
package simulate

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/genesis"
	"github.com/shaunlmason/open-seed/next/internal/gitref"
	"github.com/shaunlmason/open-seed/next/internal/keyring"
	"github.com/shaunlmason/open-seed/next/internal/lane"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
	"github.com/shaunlmason/open-seed/next/internal/loop"
	"github.com/shaunlmason/open-seed/next/internal/version"
)

const ledgerRef = "refs/seed/ledger"

// deployment is a provisioned simulation: a bare remote seeded to
// seed/1 with one enrolled, granted identity per shipped lane and role.
type deployment struct {
	dir      string
	remote   string
	state    string
	lanesDir string
	verbs    loop.Verbs
	opPriv   ed25519.PrivateKey
	opKey    string
	keys     map[string]string // lane/role name -> private key path
	fps      map[string]string // lane/role name -> fingerprint
	grants   map[string][]string
	manifest map[string]lane.Manifest
	now      time.Time
}

// keyAt writes a deterministic ed25519 key in the SSH private-key form
// the CLI's --key reads, returning its path, public hex and fingerprint.
func keyAt(dir, name string, seed byte) (path, pubHex, fp string, err error) {
	s := make([]byte, ed25519.SeedSize)
	s[0] = seed
	k := ed25519.NewKeyFromSeed(s)
	block, err := ssh.MarshalPrivateKey(k, name)
	if err != nil {
		return "", "", "", err
	}
	path = filepath.Join(dir, name+"_ed25519")
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		return "", "", "", err
	}
	pub := k.Public().(ed25519.PublicKey)
	f, err := event.Fingerprint(pub)
	if err != nil {
		return "", "", "", err
	}
	return path, hex.EncodeToString(pub), f, nil
}

// append1 lands one signed event through the CLI seam, returning an
// error carrying the boundary's refusal when it does not admit.
func (d *deployment) append1(keyPath, verb, subject, payload string) error {
	res := d.verbs.Run("ledger", "append", "--remote", d.remote, "--state", d.state,
		"--key", keyPath, "--verb", verb, "--subject", subject, "--payload", payload)
	if res.Exit != 0 || !res.OK {
		return fmt.Errorf("append %s %s refused: exit %d code %q: %s", verb, subject, res.Exit, res.Code, res.Message)
	}
	return nil
}

// build provisions the deployment: a hardened bare remote, genesis by
// the operator key, an upgrade to seed/1, and one enrolled+granted
// identity per shipped lane and role, the grants read from the
// manifests (never a list this package keeps).
func build(cfg Config) (*deployment, error) {
	dir, err := os.MkdirTemp(cfg.workRoot(), "seed-sim-")
	if err != nil {
		return nil, err
	}
	d := &deployment{
		dir:      dir,
		state:    filepath.Join(dir, "state"),
		lanesDir: cfg.LanesDir,
		verbs:    cfg.Verbs,
		keys:     map[string]string{},
		fps:      map[string]string{},
		grants:   map[string][]string{},
		manifest: map[string]lane.Manifest{},
		now:      cfg.now(),
	}
	ms, err := lane.Load(cfg.LanesDir)
	if err != nil {
		return nil, fmt.Errorf("load manifests: %w", err)
	}
	for _, m := range ms {
		d.manifest[m.Lane] = m
	}

	// The operator key is the genesis governance root.
	keydir := filepath.Join(dir, "keys")
	if err := os.MkdirAll(keydir, 0o755); err != nil {
		return nil, err
	}
	opPath, _, _, err := keyAt(keydir, "operator", 1)
	if err != nil {
		return nil, err
	}
	d.opKey = opPath
	s := make([]byte, ed25519.SeedSize)
	s[0] = 1
	d.opPriv = ed25519.NewKeyFromSeed(s)

	// The bare remote and its genesis.
	d.remote = filepath.Join(dir, "remote.git")
	if out, err := exec.Command("git", "init", "-q", "--bare", d.remote).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("bare init: %v: %s", err, out)
	}
	for _, kv := range [][2]string{{"gc.auto", "0"}, {"gc.autoDetach", "false"}, {"receive.autoGC", "false"}} {
		if out, err := exec.Command("git", "-C", d.remote, "config", kv[0], kv[1]).CombinedOutput(); err != nil {
			return nil, fmt.Errorf("harden %s: %v: %s", kv[0], err, out)
		}
	}
	if cfg.Enforced {
		if err := installHook(d.remote, filepath.Dir(cfg.LanesDir)); err != nil {
			return nil, fmt.Errorf("install enforced hook: %w", err)
		}
	}
	if err := d.seedGenesis(); err != nil {
		return nil, err
	}

	// Upgrade to seed/1 so the lifecycle verbs admit.
	if err := d.append1(d.opKey, ledger.UpgradeVerb, "system", `{"to": "`+version.Seed1+`"}`); err != nil {
		return nil, err
	}

	// One identity per shipped lane and role; grants are the manifest's.
	seed := byte(10)
	for _, m := range ms {
		path, pub, fp, err := keyAt(keydir, m.Lane, seed)
		seed++
		if err != nil {
			return nil, err
		}
		d.keys[m.Lane], d.fps[m.Lane], d.grants[m.Lane] = path, fp, m.Grants
		if err := d.append1(d.opKey, keyring.VerbEnrolled, fp,
			fmt.Sprintf(`{"key": %q, "kind": "agent", "name": %q}`, pub, m.Lane)); err != nil {
			return nil, err
		}
		for _, g := range m.Grants {
			if err := d.append1(d.opKey, keyring.VerbGranted, fp, `{"capability": "`+g+`"}`); err != nil {
				return nil, err
			}
		}
	}
	return d, nil
}

// seedGenesis lands the operator-signed genesis on the remote through
// the gitref append loop — the same round-trip the CLI's --remote path
// runs, so the deployment is credential-free and network-free.
func (d *deployment) seedGenesis() error {
	c, err := gitref.NewClient(filepath.Join(d.dir, "genesis-work"), d.remote, ledgerRef)
	if err != nil {
		return err
	}
	rec, err := genesis.Build(d.opPriv, nil, d.now)
	if err != nil {
		return err
	}
	payload, err := genesis.Parse(rec)
	if err != nil {
		return err
	}
	resolve, err := payload.Resolver(rec.Event.Actor)
	if err != nil {
		return err
	}
	res, err := c.AppendLoop(gitref.Draft{
		V: rec.Event.V, TS: rec.Event.TS, Actor: rec.Event.Actor,
		Verb: rec.Event.Verb, Subject: rec.Event.Subject, Payload: rec.Event.Payload,
	}, func(e event.Event) (*event.Record, error) { return event.Sign(e, d.opPriv) }, resolve, nil, 3)
	if err != nil {
		return err
	}
	if res.Position != 0 {
		return fmt.Errorf("genesis landed at position %d, not 0", res.Position)
	}
	_ = ledger.Resolver(resolve)
	return nil
}
