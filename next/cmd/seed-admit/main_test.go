package main

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shaunlmason/open-seed/next/internal/admit"
	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/genesis"
	"github.com/shaunlmason/open-seed/next/internal/gitref"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
)

const guardedRef = "refs/seed/ledger"

var hookBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "seed-admit-bin-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	hookBin = filepath.Join(dir, "seed-admit")
	if out, err := exec.Command("go", "build", "-o", hookBin, ".").CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "building hook: %v %s\n", err, out)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func fixtureKey(t testing.TB) ed25519.PrivateKey {
	t.Helper()
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i + 1)
	}
	return ed25519.NewKeyFromSeed(seed)
}

// guardedRemote is a bare repository with seed-admit installed.
func guardedRemote(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "remote.git")
	if out, err := exec.Command("git", "init", "-q", "--bare", dir).CombinedOutput(); err != nil {
		t.Fatalf("bare init: %v %s", err, out)
	}
	shim := "#!/bin/sh\nexec " + hookBin + "\n"
	if err := os.WriteFile(filepath.Join(dir, "hooks", "pre-receive"), []byte(shim), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func seedGenesis(t *testing.T, remote string) ledger.Resolver {
	t.Helper()
	priv := fixtureKey(t)
	c, err := gitref.NewClient(t.TempDir(), remote, guardedRef)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := genesis.Build(priv, nil, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	payload, err := genesis.Parse(rec)
	if err != nil {
		t.Fatal(err)
	}
	resolve, err := payload.Resolver(rec.Event.Actor)
	if err != nil {
		t.Fatal(err)
	}
	res, err := c.AppendLoop(gitref.Draft{
		V: rec.Event.V, TS: rec.Event.TS, Actor: rec.Event.Actor,
		Verb: rec.Event.Verb, Subject: rec.Event.Subject, Payload: rec.Event.Payload,
	}, func(e event.Event) (*event.Record, error) { return event.Sign(e, priv) }, resolve, nil, 3)
	if err != nil || res.Position != 0 {
		t.Fatalf("genesis through the hook: %+v %v", res, err)
	}
	return resolve
}

func remoteTip(t *testing.T, remote string) string {
	t.Helper()
	out, err := exec.Command("git", "--git-dir", remote, "rev-parse", "--quiet", "--verify", guardedRef).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func signed(t *testing.T, verb, subject, payload, prev string) *event.Record {
	t.Helper()
	return signedV(t, "seed/0", verb, subject, payload, prev)
}

func signedV(t *testing.T, v, verb, subject, payload, prev string) *event.Record {
	t.Helper()
	priv := fixtureKey(t)
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

// craftPush materializes the remote tip, mutates the tree via fn, and
// pushes the result as a raw adversary (no client-side validation).
func craftPush(t *testing.T, remote string, resolve ledger.Resolver, fn func(dir string, store *ledger.Store)) error {
	t.Helper()
	c, err := gitref.NewClient(t.TempDir(), remote, guardedRef)
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
	store, err := ledger.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	fn(dir, store)
	_, err = c.CommitAndPush(dir, tip, "adversary: crafted tree")
	return err
}

func appendRaw(t *testing.T, store *ledger.Store, resolve ledger.Resolver, rec *event.Record) {
	t.Helper()
	if _, err := store.Append(rec, resolve); err != nil {
		t.Fatal(err)
	}
}

func tipOf(t *testing.T, store *ledger.Store) string {
	t.Helper()
	tip, _, err := store.Tip()
	if err != nil {
		t.Fatal(err)
	}
	return tip
}

// conformance: III.B — with the hook installed, cooperative appends land
// and every raw invalid stream refuses with the rule named, leaving the
// ref untouched.
func TestHookAdmitsValidAndRefusesInvalid(t *testing.T) {
	remote := guardedRemote(t)
	resolve := seedGenesis(t, remote)
	priv := fixtureKey(t)

	// A valid cooperative append lands end to end.
	c, err := gitref.NewClient(t.TempDir(), remote, guardedRef)
	if err != nil {
		t.Fatal(err)
	}
	fp, _ := event.Fingerprint(priv.Public().(ed25519.PublicKey))
	if _, err := c.AppendLoop(gitref.Draft{
		V: "seed/0", TS: "2026-09-01T01:00:00Z", Actor: fp,
		Verb: "message.sent", Subject: "c-0001", Payload: json.RawMessage(`{"n": 1}`),
	}, func(e event.Event) (*event.Record, error) { return event.Sign(e, priv) }, resolve, admit.Validate(), 3); err != nil {
		t.Fatalf("valid append must land through the hook: %v", err)
	}

	hostile := `{"transcript": "` + strings.Repeat("all work and no play ", 40) + `"}`
	cases := []struct {
		name, want string
		mutate     func(dir string, store *ledger.Store)
	}{
		{"hostile payload", "classification", func(dir string, store *ledger.Store) {
			appendRaw(t, store, resolve, signed(t, "message.sent", "c-0002", hostile, tipOf(t, store)))
		}},
		{"halt violation inside push", "halted", func(dir string, store *ledger.Store) {
			appendRaw(t, store, resolve, signed(t, "system.halt.declared", "system", `{"reason": "drill"}`, tipOf(t, store)))
			appendRaw(t, store, resolve, signed(t, "message.sent", "c-0002", `{"n": 2}`, tipOf(t, store)))
		}},
		{"malformed declare", "shape", func(dir string, store *ledger.Store) {
			appendRaw(t, store, resolve, signed(t, "system.halt.declared", "system", `{}`, tipOf(t, store)))
		}},
		{"wrong version", "verify", func(dir string, store *ledger.Store) {
			appendRaw(t, store, resolve, signedV(t, "seed/9", "message.sent", "c-0002", `{"n": 2}`, tipOf(t, store)))
		}},
		{"claim contention", "already claimed", func(dir string, store *ledger.Store) {
			appendRaw(t, store, resolve, signed(t, ledger.UpgradeVerb, "system", `{"to": "seed/1"}`, tipOf(t, store)))
			appendRaw(t, store, resolve, signedV(t, "seed/1", "intent.filed", "c-0009",
				`{"intent": "fix", "tier": "standard", "budget": "s", "routing": "core"}`, tipOf(t, store)))
			appendRaw(t, store, resolve, signedV(t, "seed/1", "contract.specified", "c-0009",
				`{"acceptance": {"ref": "specs/c9.md @ abc1234", "executable": false}}`, tipOf(t, store)))
			appendRaw(t, store, resolve, signedV(t, "seed/1", "claim.taken", "c-0009", `{}`, tipOf(t, store)))
			appendRaw(t, store, resolve, signedV(t, "seed/1", "claim.taken", "c-0009", `{}`, tipOf(t, store)))
		}},
		{"illegal lifecycle transition", "lifecycle", func(dir string, store *ledger.Store) {
			appendRaw(t, store, resolve, signed(t, ledger.UpgradeVerb, "system", `{"to": "seed/1"}`, tipOf(t, store)))
			appendRaw(t, store, resolve, signedV(t, "seed/1", "claim.taken", "c-0002", `{"note": "no such subject"}`, tipOf(t, store)))
		}},
		{"schema-broken upgrade", "verify", func(dir string, store *ledger.Store) {
			appendRaw(t, store, resolve, signed(t, ledger.UpgradeVerb, "system", `{"note": "missing to"}`, tipOf(t, store)))
		}},
		{"forged signature", "verify", func(dir string, store *ledger.Store) {
			appendRaw(t, store, resolve, signed(t, "message.sent", "c-0002", `{"n": 5}`, tipOf(t, store)))
			segs, err := filepath.Glob(filepath.Join(dir, "segments", "*.jsonl"))
			if err != nil || len(segs) == 0 {
				t.Fatal("no segments")
			}
			b, err := os.ReadFile(segs[len(segs)-1])
			if err != nil {
				t.Fatal(err)
			}
			forged := strings.Replace(string(b), `"n":5`, `"n":6`, 1)
			if forged == string(b) {
				t.Fatal("forgery did not apply")
			}
			if err := os.WriteFile(segs[len(segs)-1], []byte(forged), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tc := range cases {
		before := remoteTip(t, remote)
		err := craftPush(t, remote, resolve, tc.mutate)
		if !errors.Is(err, gitref.ErrRemoteRejected) {
			t.Errorf("%s: want ErrRemoteRejected, got %v", tc.name, err)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: refusal must name %q, got %q", tc.name, tc.want, err.Error())
		}
		if after := remoteTip(t, remote); after != before {
			t.Errorf("%s: refused push moved the ref", tc.name)
		}
	}

	// A multi-record valid push validates each record and lands.
	if err := craftPush(t, remote, resolve, func(dir string, store *ledger.Store) {
		appendRaw(t, store, resolve, signed(t, "message.sent", "c-0003", `{"n": 3}`, tipOf(t, store)))
		appendRaw(t, store, resolve, signed(t, "message.sent", "c-0004", `{"n": 4}`, tipOf(t, store)))
	}); err != nil {
		t.Fatalf("multi-record valid push must land: %v", err)
	}

	// The lifecycle happy path admits through the hook: upgrade, file,
	// specify in one push, each record checked by the shared rule set
	// (plans/os-d69a6c91.md step 7).
	if err := craftPush(t, remote, resolve, func(dir string, store *ledger.Store) {
		appendRaw(t, store, resolve, signed(t, ledger.UpgradeVerb, "system", `{"to": "seed/1"}`, tipOf(t, store)))
		appendRaw(t, store, resolve, signedV(t, "seed/1", "intent.filed", "c-0005",
			`{"intent": "fix", "tier": "standard", "budget": "s", "routing": "core"}`, tipOf(t, store)))
		appendRaw(t, store, resolve, signedV(t, "seed/1", "contract.specified", "c-0005",
			`{"acceptance": {"ref": "specs/c5.md @ abc1234", "executable": false}}`, tipOf(t, store)))
	}); err != nil {
		t.Fatalf("the lifecycle happy path must land through the hook: %v", err)
	}
}

// conformance: III.B — append-only at the record level: a descendant
// commit whose tree rewrites admitted records is refused even though the
// commit graph fast-forwards.
func TestHookRefusesHistoryRewriteAndRefShapes(t *testing.T) {
	remote := guardedRemote(t)
	resolve := seedGenesis(t, remote)
	priv := fixtureKey(t)

	// Land one admitted milestone.
	c, err := gitref.NewClient(t.TempDir(), remote, guardedRef)
	if err != nil {
		t.Fatal(err)
	}
	fp, _ := event.Fingerprint(priv.Public().(ed25519.PublicKey))
	if _, err := c.AppendLoop(gitref.Draft{
		V: "seed/0", TS: "2026-09-01T01:00:00Z", Actor: fp,
		Verb: "message.sent", Subject: "c-0001", Payload: json.RawMessage(`{"n": 1}`),
	}, func(e event.Event) (*event.Record, error) { return event.Sign(e, priv) }, resolve, admit.Validate(), 3); err != nil {
		t.Fatal(err)
	}

	// Rewrite: a valid *different* chain of the same shape, committed as
	// a descendant of the admitted tip commit.
	before := remoteTip(t, remote)
	rc, err := gitref.NewClient(t.TempDir(), remote, guardedRef)
	if err != nil {
		t.Fatal(err)
	}
	tipCommit, err := rc.Fetch()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	// Materialize only the genesis: rebuild the alternative history from
	// scratch in an empty dir seeded with a fresh store.
	store, err := ledger.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	grec, err := genesis.Build(priv, nil, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Append(grec, resolve); err != nil {
		t.Fatal(err)
	}
	appendRaw(t, store, resolve, signed(t, "message.sent", "c-9999", `{"n": 9}`, tipOf(t, store)))
	_, err = rc.CommitAndPush(dir, tipCommit, "adversary: rewritten history as descendant commit")
	if !errors.Is(err, gitref.ErrRemoteRejected) || !strings.Contains(err.Error(), "rewrites admitted history") {
		t.Fatalf("record-level rewrite must refuse, got %v", err)
	}
	if remoteTip(t, remote) != before {
		t.Fatal("refused rewrite moved the ref")
	}

	// A descendant commit that truncates the ledger refuses: commit
	// ancestry does not make tree contents append-only (#92 review).
	genesisOnlyTrunc := t.TempDir()
	tstore, err := ledger.Open(genesisOnlyTrunc)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tstore.Append(grec, resolve); err != nil {
		t.Fatal(err)
	}
	tc2, err := gitref.NewClient(t.TempDir(), remote, guardedRef)
	if err != nil {
		t.Fatal(err)
	}
	tipCommit2, err := tc2.Fetch()
	if err != nil {
		t.Fatal(err)
	}
	_, err = tc2.CommitAndPush(genesisOnlyTrunc, tipCommit2, "adversary: truncated ledger as descendant commit")
	if !errors.Is(err, gitref.ErrRemoteRejected) || !strings.Contains(err.Error(), "drops admitted records") {
		t.Fatalf("record-level truncation must refuse, got %v", err)
	}
	if remoteTip(t, remote) != before {
		t.Fatal("refused truncation moved the ref")
	}

	// Deletion refuses and the ref survives.
	scratch := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", scratch).CombinedOutput(); err != nil {
		t.Fatalf("scratch init: %v %s", err, out)
	}
	out, err := exec.Command("git", "-C", scratch, "push", remote, ":"+guardedRef).CombinedOutput()
	if err == nil || !strings.Contains(string(out), "deletion of the ledger ref is refused") {
		t.Fatalf("deletion must refuse via the hook, got %v %s", err, out)
	}
	if remoteTip(t, remote) != before {
		t.Fatal("deletion moved the ref")
	}

	// A plain non-fast-forward force-update refuses at the ref shape: an
	// unrelated parentless commit forced over the admitted tip.
	genesisOnly := t.TempDir()
	gstore, err := ledger.Open(genesisOnly)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gstore.Append(grec, resolve); err != nil {
		t.Fatal(err)
	}
	rootCommit := parentlessCommit(t, genesisOnly)
	gd := filepath.Dir(rootCommit)
	sha := filepath.Base(rootCommit)
	out, err = exec.Command("git", "--git-dir", gd, "push", "--force", remote, sha+":"+guardedRef).CombinedOutput()
	if err == nil || !strings.Contains(string(out), "non-fast-forward update is refused") {
		t.Fatalf("forced non-fast-forward must refuse, got %v %s", err, out)
	}
	if remoteTip(t, remote) != before {
		t.Fatal("forced update moved the ref")
	}

	// An unrelated ref passes untouched.
	exec.Command("git", "-C", scratch, "config", "user.email", "t@t").Run()
	exec.Command("git", "-C", scratch, "config", "user.name", "t").Run()
	if out, err := exec.Command("git", "-C", scratch, "commit", "--allow-empty", "-qm", "docs").CombinedOutput(); err != nil {
		t.Fatalf("scratch commit: %v %s", err, out)
	}
	if out, err := exec.Command("git", "-C", scratch, "push", remote, "HEAD:refs/heads/docs").CombinedOutput(); err != nil {
		t.Fatalf("unrelated ref must pass: %v %s", err, out)
	}
}

// parentlessCommit commits dir's tree with no parent into a scratch git
// dir and returns "<gitdir>/<sha>" for explicit pushes.
func parentlessCommit(t *testing.T, dir string) string {
	t.Helper()
	gd := filepath.Join(t.TempDir(), "gd.git")
	if out, err := exec.Command("git", "init", "-q", "--bare", gd).CombinedOutput(); err != nil {
		t.Fatalf("init: %v %s", err, out)
	}
	if out, err := exec.Command("git", "--git-dir", gd, "-c", "core.bare=false", "--work-tree", dir, "add", "-A").CombinedOutput(); err != nil {
		t.Fatalf("add: %v %s", err, out)
	}
	treeOut, err := exec.Command("git", "--git-dir", gd, "write-tree").Output()
	if err != nil {
		t.Fatal(err)
	}
	sha, err := exec.Command("git", "--git-dir", gd,
		"-c", "user.name=t", "-c", "user.email=t@t",
		"commit-tree", strings.TrimSpace(string(treeOut)), "-m", "adversary root").Output()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(gd, strings.TrimSpace(string(sha)))
}

// The hook is driven entirely by its stdin contract; malformed lines and
// a guarded-ref override behave as documented.
func TestHookStdinContractAndRefOverride(t *testing.T) {
	var errOut strings.Builder
	if code := run(strings.NewReader("garbage line\n"), &errOut, ".", guardedRef); code != 1 || !strings.Contains(errOut.String(), "malformed") {
		t.Fatalf("malformed update line must refuse: %d %q", code, errOut.String())
	}
	// Updates to other refs pass without touching the repo.
	if code := run(strings.NewReader(fmt.Sprintf("%s %s refs/heads/main\n", zeroID, zeroID)), &strings.Builder{}, "/nonexistent", guardedRef); code != 0 {
		t.Fatal("non-guarded refs must pass without repo access")
	}
	// With an override, the default ref is no longer guarded.
	if code := run(strings.NewReader(fmt.Sprintf("%s %s %s\n", zeroID, zeroID, guardedRef)), &strings.Builder{}, "/nonexistent", "refs/other/ledger"); code != 0 {
		t.Fatal("an overridden guard must release the default ref")
	}
	// Deletion of the guarded ref refuses before any repo access.
	var del strings.Builder
	if code := run(strings.NewReader(fmt.Sprintf("%s %s %s\n", "a1b2c3", zeroID, guardedRef)), &del, "/nonexistent", guardedRef); code != 1 || !strings.Contains(del.String(), "deletion") {
		t.Fatalf("deletion must refuse: %d %q", code, del.String())
	}
}

// conformance: III.B + the references-not-content boundary — the guarded
// ref carries only the ledger layout: a fast-forward push riding extra
// files beside an unchanged (or even validly extended) record stream
// refuses (#94 review).
func TestHookRefusesNonLedgerFiles(t *testing.T) {
	remote := guardedRemote(t)
	resolve := seedGenesis(t, remote)

	before := remoteTip(t, remote)
	err := craftPush(t, remote, resolve, func(dir string, store *ledger.Store) {
		if werr := os.WriteFile(filepath.Join(dir, "transcript.txt"), []byte("smuggled content"), 0o644); werr != nil {
			t.Fatal(werr)
		}
	})
	if !errors.Is(err, gitref.ErrRemoteRejected) || !strings.Contains(err.Error(), "outside the ledger layout") {
		t.Fatalf("extra file with unchanged records must refuse, got %v", err)
	}
	if remoteTip(t, remote) != before {
		t.Fatal("refused push moved the ref")
	}

	err = craftPush(t, remote, resolve, func(dir string, store *ledger.Store) {
		appendRaw(t, store, resolve, signed(t, "message.sent", "c-0001", `{"n": 1}`, tipOf(t, store)))
		if werr := os.MkdirAll(filepath.Join(dir, "segments", "nested"), 0o755); werr != nil {
			t.Fatal(werr)
		}
		if werr := os.WriteFile(filepath.Join(dir, "segments", "nested", "blob.jsonl"), []byte("{}"), 0o644); werr != nil {
			t.Fatal(werr)
		}
	})
	if !errors.Is(err, gitref.ErrRemoteRejected) || !strings.Contains(err.Error(), "outside the ledger layout") {
		t.Fatalf("nested non-layout path must refuse even beside a valid record, got %v", err)
	}
	if remoteTip(t, remote) != before {
		t.Fatal("refused push moved the ref")
	}
}

// The boundary's rule share is selected by exclusion: only replay-owned
// checks drop, so rules future phases append to the shared set flow
// through automatically (#94 review).
func TestAdmissionRulesSelectByExclusion(t *testing.T) {
	fake := admit.Rule{Name: "capability", Check: func(*admit.Context, *event.Record) error { return nil }}
	got := admissionRules(append(admit.Default(), fake))
	names := map[string]bool{}
	for _, r := range got {
		names[r.Name] = true
	}
	for _, want := range []string{"halted", "shape", "classification", "capability"} {
		if !names[want] {
			t.Errorf("rule %q must survive selection", want)
		}
	}
	for _, dropped := range []string{"actor", "version"} {
		if names[dropped] {
			t.Errorf("replay-owned rule %q must not re-run at the boundary", dropped)
		}
	}
}
