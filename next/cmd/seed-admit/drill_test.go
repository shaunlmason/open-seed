package main

// The Phase 2 admission drills (plans/os-028dda91.md; conformance
// III.B): the raw-git adversary run against both declared postures, and
// kill-and-replace proving the hook host holds no state worth keeping.

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/admit"
	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/gitref"
	"github.com/shaunlmason/open-seed/next/internal/halt"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
	"github.com/shaunlmason/open-seed/next/internal/posture"
)

// deployment is a bare remote standing at a declared posture.
type deployment struct {
	remote  string
	posture posture.Posture
}

// newDeployment declares the posture in the production format and reads
// it back through internal/posture to decide enforcement, so fixtures
// and production share one declaration (the "both postures selectable
// in fixtures" exit criterion).
func newDeployment(t *testing.T, p posture.Posture) deployment {
	t.Helper()
	host := t.TempDir()
	remote := filepath.Join(host, "remote.git")
	if out, err := exec.Command("git", "init", "-q", "--bare", remote).CombinedOutput(); err != nil {
		t.Fatalf("bare init: %v %s", err, out)
	}
	cfgPath := filepath.Join(host, "seed.json")
	if err := os.WriteFile(cfgPath, []byte(`{"posture": "`+string(p)+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := posture.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Posture == posture.EnforcedSelfHosted {
		installHook(t, remote)
	}
	return deployment{remote: remote, posture: cfg.Posture}
}

func installHook(t *testing.T, remote string) {
	t.Helper()
	shim := "#!/bin/sh\nexec " + hookBin + "\n"
	if err := os.WriteFile(filepath.Join(remote, "hooks", "pre-receive"), []byte(shim), 0o755); err != nil {
		t.Fatal(err)
	}
}

func advKey(t testing.TB, first byte) ed25519.PrivateKey {
	t.Helper()
	seed := make([]byte, ed25519.SeedSize)
	seed[0] = first
	return ed25519.NewKeyFromSeed(seed)
}

// advResolver resolves every listed key: adversary chain construction
// only, never the system under test.
func advResolver(t testing.TB, privs ...ed25519.PrivateKey) ledger.Resolver {
	t.Helper()
	ring := map[string]ed25519.PublicKey{}
	for _, p := range privs {
		fp, err := event.Fingerprint(p.Public().(ed25519.PublicKey))
		if err != nil {
			t.Fatal(err)
		}
		ring[fp] = p.Public().(ed25519.PublicKey)
	}
	return func(fp string) (ed25519.PublicKey, bool) {
		pub, ok := ring[fp]
		return pub, ok
	}
}

func advSignedBy(t *testing.T, priv ed25519.PrivateKey, v, verb, subject, payload, prev string) *event.Record {
	t.Helper()
	fp, err := event.Fingerprint(priv.Public().(ed25519.PublicKey))
	if err != nil {
		t.Fatal(err)
	}
	rec, err := event.Sign(event.Event{
		V: v, TS: "2026-09-01T02:00:00Z", Actor: fp,
		Verb: verb, Subject: subject, Payload: json.RawMessage(payload), Prev: prev,
	}, priv)
	if err != nil {
		t.Fatal(err)
	}
	return rec
}

// adversaryCases is the one hostile-event table both posture fixtures
// run (plans/os-028dda91.md step 2: "the same adversary suite against
// both fixtures"): each case crafts its raw event(s) and names the
// refusal the cooperative side must still produce for the same damage.
func adversaryCases(t *testing.T) map[string]struct {
	raw     func(store *ledger.Store)
	payload string
	refuse  func(t *testing.T, err error)
} {
	hostile := `{"transcript": "` + strings.Repeat("all work and no play ", 40) + `"}`
	root, rogue := fixtureKey(t), advKey(t, 11)
	loose := advResolver(t, root, rogue)
	contains := func(parts ...string) func(t *testing.T, err error) {
		return func(t *testing.T, err error) {
			t.Helper()
			for _, p := range parts {
				if err == nil || !strings.Contains(err.Error(), p) {
					t.Fatalf("cooperative refusal must carry %q, got %v", p, err)
				}
			}
		}
	}
	return map[string]struct {
		raw     func(store *ledger.Store)
		payload string
		refuse  func(t *testing.T, err error)
	}{
		"classification": {
			raw: func(store *ledger.Store) {
				appendRaw(t, store, loose, signed(t, "progress.milestone", "c-0001", hostile, tipOf(t, store)))
			},
			payload: hostile,
			refuse: func(t *testing.T, err error) {
				t.Helper()
				var cls *admit.ClassificationError
				if !errors.As(err, &cls) {
					t.Fatalf("the client must self-refuse the classified payload, got %v", err)
				}
			},
		},
		"version": {
			raw: func(store *ledger.Store) {
				appendRaw(t, store, loose, signedV(t, "seed/9", "progress.milestone", "c-0001", `{"n": 9}`, tipOf(t, store)))
			},
			payload: `{"n": 1}`,
			// The landed violation breaks the shared chain for every
			// reader: the client's pre-flight replay refuses it.
			refuse: contains("failed verification", "version"),
		},
		"halted": {
			// The halt declaration itself is a legal event; the
			// violation is the append under it.
			raw: func(store *ledger.Store) {
				appendRaw(t, store, loose, signed(t, "system.halt.declared", "system", `{"reason": "drill"}`, tipOf(t, store)))
				appendRaw(t, store, loose, signed(t, "progress.milestone", "c-0001", `{"n": 9}`, tipOf(t, store)))
			},
			payload: `{"n": 1}`,
			refuse: func(t *testing.T, err error) {
				t.Helper()
				var h *halt.HaltedError
				if !errors.As(err, &h) {
					t.Fatalf("the client must refuse to append under halt, got %v", err)
				}
			},
		},
		"malformed upgrade": {
			raw: func(store *ledger.Store) {
				appendRaw(t, store, loose, signed(t, ledger.UpgradeVerb, "system", `{"nope": true}`, tipOf(t, store)))
			},
			payload: `{"n": 1}`,
			refuse:  contains("failed verification", "bad_payload"),
		},
		"forged signer": {
			raw: func(store *ledger.Store) {
				appendRaw(t, store, loose, advSignedBy(t, rogue, "seed/0", "progress.milestone", "c-0001", `{"n": 9}`, tipOf(t, store)))
			},
			payload: `{"n": 1}`,
			refuse:  contains("failed verification", "unknown_actor"),
		},
	}
}

// conformance: III.B — the raw-git adversary drill, per posture, over
// the shared case table. Under enforced, every invalid raw push refuses
// and the ref never moves. Under cooperative the same pushes land — the
// charter's named consequence (posture.Consequence) made observable —
// and the cooperative side still refuses the same damage locally: as a
// draft refusal where the chain stays valid (classification, halt), and
// as a pre-flight verification refusal where the landed violation broke
// the shared chain for every reader (version, schema, forged signer).
func TestDrillRawAdversaryPerPosture(t *testing.T) {
	for _, p := range []posture.Posture{posture.EnforcedSelfHosted, posture.Cooperative} {
		for name, c := range adversaryCases(t) {
			t.Run(string(p)+"/"+name, func(t *testing.T) {
				d := newDeployment(t, p)
				resolve := seedGenesis(t, d.remote)
				before := remoteTip(t, d.remote)

				err := craftPush(t, d.remote, resolve, func(dir string, store *ledger.Store) { c.raw(store) })
				after := remoteTip(t, d.remote)
				if d.posture.Enforced() {
					if !errors.Is(err, gitref.ErrRemoteRejected) || after != before {
						t.Fatalf("enforced posture must refuse the raw adversary with the ref unmoved, got %v (tip %s -> %s)", err, before, after)
					}
					return
				}
				if err != nil || after == before {
					t.Fatalf("under cooperative the raw push lands — %q — got %v (tip unchanged)", posture.Consequence, err)
				}

				// The cooperating client still refuses the same damage
				// locally: the self-validation half of the posture.
				priv := fixtureKey(t)
				fp, ferr := event.Fingerprint(priv.Public().(ed25519.PublicKey))
				if ferr != nil {
					t.Fatal(ferr)
				}
				cl, cerr := gitref.NewClient(t.TempDir(), d.remote, guardedRef)
				if cerr != nil {
					t.Fatal(cerr)
				}
				_, err = cl.AppendLoop(gitref.Draft{
					V: "seed/0", TS: "2026-09-01T03:00:00Z", Actor: fp,
					Verb: "progress.milestone", Subject: "c-0002", Payload: json.RawMessage(c.payload),
				}, func(e event.Event) (*event.Record, error) { return event.Sign(e, priv) }, resolve, admit.Validate(), 3)
				c.refuse(t, err)
			})
		}
	}
}

// conformance: III.B statelessness — kill-and-replace: destroy the hook
// host, rebuild it from a fresh bare clone of the guarded repository,
// install a freshly built hook, and every decision comes out the same:
// the same refusals, an admitted valid append, and a green from-genesis
// verification.
func TestDrillKillAndReplace(t *testing.T) {
	d := newDeployment(t, posture.EnforcedSelfHosted)
	resolve := seedGenesis(t, d.remote)
	priv := fixtureKey(t)
	fp, _ := event.Fingerprint(priv.Public().(ed25519.PublicKey))

	appendValid := func(remote, subject string) error {
		c, err := gitref.NewClient(t.TempDir(), remote, guardedRef)
		if err != nil {
			t.Fatal(err)
		}
		_, err = c.AppendLoop(gitref.Draft{
			V: "seed/0", TS: "2026-09-01T03:00:00Z", Actor: fp,
			Verb: "progress.milestone", Subject: subject, Payload: json.RawMessage(`{"n": 1}`),
		}, func(e event.Event) (*event.Record, error) { return event.Sign(e, priv) }, resolve, admit.Validate(), 3)
		return err
	}
	if err := appendValid(d.remote, "c-0001"); err != nil {
		t.Fatalf("original host must admit a valid append: %v", err)
	}

	refusals := func(remote string) map[string]string {
		hostile := `{"transcript": "` + strings.Repeat("all work and no play ", 40) + `"}`
		got := map[string]string{}
		for name, mutate := range map[string]func(dir string, store *ledger.Store){
			"classification": func(dir string, store *ledger.Store) {
				appendRaw(t, store, resolve, signed(t, "progress.milestone", "c-0009", hostile, tipOf(t, store)))
			},
			"halted": func(dir string, store *ledger.Store) {
				appendRaw(t, store, resolve, signed(t, "system.halt.declared", "system", `{"reason": "drill"}`, tipOf(t, store)))
				appendRaw(t, store, resolve, signed(t, "progress.milestone", "c-0009", `{"n": 9}`, tipOf(t, store)))
			},
			"verify": func(dir string, store *ledger.Store) {
				appendRaw(t, store, resolve, signedV(t, "seed/9", "progress.milestone", "c-0009", `{"n": 9}`, tipOf(t, store)))
			},
		} {
			before := remoteTip(t, remote)
			err := craftPush(t, remote, resolve, mutate)
			if !errors.Is(err, gitref.ErrRemoteRejected) || remoteTip(t, remote) != before {
				t.Fatalf("%s must refuse on %s, got %v", name, remote, err)
			}
			got[name] = ruleLine(err.Error())
		}
		return got
	}
	originalDecisions := refusals(d.remote)

	// Kill the host: the replacement is a fresh mirror clone, so only
	// replicated Git data (refs and objects) crosses over — no repo-local
	// config, hooks, or cached validator state. A mirror clone is what
	// replicates the guarded ref: a plain bare clone maps refs/heads/*
	// only and would miss it. The original is then deleted outright, so
	// nothing the replacement decides can read the old path, and the
	// replacement's hook is an independently installed copy of the built
	// validator at a fresh path inside the new host — no filesystem state
	// shared with the original installation (review finding on #99).
	replacement := filepath.Join(t.TempDir(), "replacement.git")
	if out, err := exec.Command("git", "clone", "-q", "--mirror", d.remote, replacement).CombinedOutput(); err != nil {
		t.Fatalf("replace host: %v %s", err, out)
	}
	if err := os.RemoveAll(d.remote); err != nil {
		t.Fatalf("kill original host: %v", err)
	}
	hookBytes, err := os.ReadFile(hookBin)
	if err != nil {
		t.Fatal(err)
	}
	freshBin := filepath.Join(replacement, "hooks", "seed-admit")
	if err := os.WriteFile(freshBin, hookBytes, 0o755); err != nil {
		t.Fatal(err)
	}
	shim := "#!/bin/sh\nexec " + freshBin + "\n"
	if err := os.WriteFile(filepath.Join(replacement, "hooks", "pre-receive"), []byte(shim), 0o755); err != nil {
		t.Fatal(err)
	}

	replacementDecisions := refusals(replacement)
	for name, want := range originalDecisions {
		if replacementDecisions[name] != want {
			t.Fatalf("%s decision drifted across kill-and-replace: %q vs %q", name, want, replacementDecisions[name])
		}
	}
	if err := appendValid(replacement, "c-0002"); err != nil {
		t.Fatalf("replacement host must admit a valid append: %v", err)
	}
	c, err := gitref.NewClient(t.TempDir(), replacement, guardedRef)
	if err != nil {
		t.Fatal(err)
	}
	tip, err := c.Fetch()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := c.Materialize(tip, dir); err != nil {
		t.Fatal(err)
	}
	store, err := ledger.OpenReadOnly(dir)
	if err != nil {
		t.Fatal(err)
	}
	rep, err := store.VerifyFromGenesis(resolve)
	if err != nil || rep.Count != 3 {
		t.Fatalf("replacement chain must verify from genesis: %+v %v", rep, err)
	}
}

// ruleLine extracts the hook's refusal line ("seed-admit: ...") from the
// porcelain-wrapped push error, the part that must not drift across a
// host replacement.
func ruleLine(msg string) string {
	for _, line := range strings.Split(msg, "\n") {
		if i := strings.Index(line, "seed-admit:"); i >= 0 {
			return strings.TrimSpace(line[i:])
		}
	}
	return msg
}
