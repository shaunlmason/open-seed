package main

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/genesis"
	"github.com/shaunlmason/open-seed/next/internal/gitref"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
)

const remoteRef = "refs/seed/ledger"

// fixturePriv is the same key writeKeys marshals for the CLI, so the
// CLI's --key is the genesis governance root.
func fixturePriv(t testing.TB) ed25519.PrivateKey {
	t.Helper()
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i + 1)
	}
	return ed25519.NewKeyFromSeed(seed)
}

func bareRemote(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "remote.git")
	if out, err := exec.Command("git", "init", "-q", "--bare", dir).CombinedOutput(); err != nil {
		t.Fatalf("bare init: %v %s", err, out)
	}
	return dir
}

// seedRemoteGenesis lands the genesis on the remote through the loop and
// returns the resolver for library-side appends.
func seedRemoteGenesis(t *testing.T, remote string) ledger.Resolver {
	t.Helper()
	priv := fixturePriv(t)
	c, err := gitref.NewClient(t.TempDir(), remote, remoteRef)
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
		t.Fatalf("genesis append: %+v %v", res, err)
	}
	return resolve
}

// libAppend lands one signed event on the remote through the loop.
func libAppend(t *testing.T, remote string, resolve ledger.Resolver, v, verb, subject, payload string, vopts ...ledger.VerifyOption) {
	t.Helper()
	priv := fixturePriv(t)
	fp, err := event.Fingerprint(priv.Public().(ed25519.PublicKey))
	if err != nil {
		t.Fatal(err)
	}
	c, err := gitref.NewClient(t.TempDir(), remote, remoteRef)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.AppendLoop(gitref.Draft{
		V: v, TS: "2026-09-01T01:00:00Z", Actor: fp,
		Verb: verb, Subject: subject, Payload: json.RawMessage(payload),
	}, func(e event.Event) (*event.Record, error) { return event.Sign(e, priv) }, resolve, nil, 5, vopts...); err != nil {
		t.Fatal(err)
	}
}

func remoteTip(t *testing.T, remote string) string {
	t.Helper()
	out, err := exec.Command("git", "--git-dir", remote, "rev-parse", "--quiet", "--verify", remoteRef).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// conformance: III.B cooperative posture — a clean remote append lands
// through self-validation and the landed chain verifies from genesis.
func TestRemoteAppendRoundTrip(t *testing.T) {
	dir, priv, _ := writeKeys(t)
	remote := bareRemote(t)
	seedRemoteGenesis(t, remote)

	e, code := runEnv(t, "ledger", "append", "--remote", remote, "--state", filepath.Join(dir, "state"),
		"--key", priv, "--verb", "progress.milestone", "--subject", "c-0001", "--payload", `{"n": 1}`)
	if code != 0 || !e.OK || e.Position == nil || *e.Position != "1" {
		t.Fatalf("remote append failed: %d %+v", code, e)
	}
	if e.Result["commit"] == "" || e.Result["appended"] == "" || e.Result["attempts"].(float64) < 1 {
		t.Fatalf("result must carry commit, hash, attempts: %+v", e.Result)
	}

	// The landed chain verifies from genesis on a fresh materialization.
	c, err := gitref.NewClient(t.TempDir(), remote, remoteRef)
	if err != nil {
		t.Fatal(err)
	}
	tip, err := c.Fetch()
	if err != nil {
		t.Fatal(err)
	}
	ld := filepath.Join(dir, "materialized")
	if err := c.Materialize(tip, ld); err != nil {
		t.Fatal(err)
	}
	if e, code := runEnv(t, "ledger", "verify", "--ledger", ld); code != 0 || e.Result["count"].(float64) != 2 {
		t.Fatalf("landed chain must verify: %d %+v", code, e)
	}
}

// conformance: III.B cooperative posture — invalid drafts refuse locally
// with the remote tip unchanged.
func TestRemoteAppendCooperativeRefusals(t *testing.T) {
	_, priv, _ := writeKeys(t)

	t.Run("halted chain refuses at 7", func(t *testing.T) {
		remote := bareRemote(t)
		resolve := seedRemoteGenesis(t, remote)
		libAppend(t, remote, resolve, "seed/0", "system.halt.declared", "system", `{"reason": "drill"}`)
		before := remoteTip(t, remote)
		e, code := runEnv(t, "ledger", "append", "--remote", remote, "--state", t.TempDir(),
			"--key", priv, "--verb", "progress.milestone", "--subject", "c-0001", "--payload", `{"n": 1}`)
		if code != 7 || e.Error == nil || e.Error.Code != "halted" || !strings.Contains(e.Error.Message, "drill") {
			t.Fatalf("halted remote must refuse at 7 with the reason, got %d %+v", code, e)
		}
		if e.Position == nil || *e.Position != "1" {
			t.Fatalf("admission refusals are stamped at the tip they were computed at, got %+v", e.Position)
		}
		if after := remoteTip(t, remote); after != before {
			t.Fatalf("refused draft must push nothing: %s vs %s", before, after)
		}
	})

	t.Run("hostile payload refuses at 9", func(t *testing.T) {
		remote := bareRemote(t)
		seedRemoteGenesis(t, remote)
		before := remoteTip(t, remote)
		hostile := `{"transcript": "` + strings.Repeat("all work and no play ", 40) + `"}`
		e, code := runEnv(t, "ledger", "append", "--remote", remote, "--state", t.TempDir(),
			"--key", priv, "--verb", "progress.milestone", "--subject", "c-0001", "--payload", hostile)
		if code != 9 || e.Error == nil || e.Error.Code != "classification_refused" {
			t.Fatalf("hostile payload must refuse at 9, got %d %+v", code, e)
		}
		if after := remoteTip(t, remote); after != before {
			t.Fatal("refused draft must push nothing")
		}
	})

	t.Run("upgraded remote refuses stale build at 10", func(t *testing.T) {
		remote := bareRemote(t)
		resolve := seedRemoteGenesis(t, remote)
		libAppend(t, remote, resolve, "seed/0", ledger.UpgradeVerb, "system", `{"to": "seed/9"}`)
		before := remoteTip(t, remote)
		e, code := runEnv(t, "ledger", "append", "--remote", remote, "--state", t.TempDir(),
			"--key", priv, "--verb", "progress.milestone", "--subject", "c-0001", "--payload", `{"n": 1}`)
		if code != 10 || e.Error == nil || e.Error.Code != "version_unsupported" {
			t.Fatalf("upgraded remote must refuse the stale build at 10, got %d %+v", code, e)
		}
		if after := remoteTip(t, remote); after != before {
			t.Fatal("refused draft must push nothing")
		}
	})
}

// installRivalHook arms the remote so each push attempt loses to the
// next prepared rival (a valid chain extension) via ref-lock contention,
// until the rivals run out.
func installRivalHook(t *testing.T, remote string, rivals []string) {
	t.Helper()
	rf := filepath.Join(remote, "rivals")
	if err := os.WriteFile(rf, []byte(strings.Join(rivals, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Pre-receive hooks run in git's quarantine, which forbids ref
	// updates; the rival commits already live in the main object store
	// (pushed, then rewound), so lifting the quarantine env for the
	// update is safe and gives a deterministic mid-push race.
	hook := fmt.Sprintf(`#!/bin/sh
r=$(head -n1 %[1]s 2>/dev/null)
if [ -n "$r" ]; then
  unset GIT_QUARANTINE_PATH
  git update-ref %[2]s "$r"
  tail -n +2 %[1]s > %[1]s.n && mv %[1]s.n %[1]s
fi
exit 0
`, rf, remoteRef)
	if err := os.WriteFile(filepath.Join(remote, "hooks", "pre-receive"), []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}
}

// buildRivals prepares n successive valid appends and rewinds the ref,
// returning the commit at each step for the hook to replay.
func buildRivals(t *testing.T, remote string, resolve ledger.Resolver, n int) []string {
	t.Helper()
	base := remoteTip(t, remote)
	var rivals []string
	for i := 0; i < n; i++ {
		libAppend(t, remote, resolve, "seed/0", "progress.milestone", "c-rival", fmt.Sprintf(`{"n": %d}`, i))
		rivals = append(rivals, remoteTip(t, remote))
	}
	if out, err := exec.Command("git", "--git-dir", remote, "update-ref", remoteRef, base).CombinedOutput(); err != nil {
		t.Fatalf("rewind: %v %s", err, out)
	}
	return rivals
}

func TestRemoteAppendRaceRetriesAndLands(t *testing.T) {
	dir, priv, _ := writeKeys(t)
	remote := bareRemote(t)
	resolve := seedRemoteGenesis(t, remote)
	installRivalHook(t, remote, buildRivals(t, remote, resolve, 1))

	e, code := runEnv(t, "ledger", "append", "--remote", remote, "--state", filepath.Join(dir, "state"),
		"--key", priv, "--verb", "progress.milestone", "--subject", "c-0001", "--payload", `{"n": 9}`)
	if code != 0 || !e.OK {
		t.Fatalf("raced append must still land: %d %+v", code, e)
	}
	if e.Result["attempts"].(float64) < 2 {
		t.Fatalf("the race was not real: %+v", e.Result)
	}
	if e.Position == nil || *e.Position != "2" {
		t.Fatalf("landed position must sit after the rival, got %+v", e.Position)
	}
}

func TestRemoteAppendExhaustsAtContention(t *testing.T) {
	dir, priv, _ := writeKeys(t)
	remote := bareRemote(t)
	resolve := seedRemoteGenesis(t, remote)
	installRivalHook(t, remote, buildRivals(t, remote, resolve, remoteMaxAttempts+1))

	e, code := runEnv(t, "ledger", "append", "--remote", remote, "--state", filepath.Join(dir, "state"),
		"--key", priv, "--verb", "progress.milestone", "--subject", "c-0001", "--payload", `{"n": 9}`)
	if code != 2 || e.Error == nil || e.Error.Code != "contention" {
		t.Fatalf("perpetual rivals must exhaust at 2 contention, got %d %+v", code, e)
	}
}

func TestRemoteAppendHookRejectionAt11(t *testing.T) {
	dir, priv, _ := writeKeys(t)
	remote := bareRemote(t)
	seedRemoteGenesis(t, remote)
	hook := "#!/bin/sh\necho 'seed policy says no' >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(remote, "hooks", "pre-receive"), []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	before := remoteTip(t, remote)
	e, code := runEnv(t, "ledger", "append", "--remote", remote, "--state", filepath.Join(dir, "state"),
		"--key", priv, "--verb", "progress.milestone", "--subject", "c-0001", "--payload", `{"n": 1}`)
	if code != 11 || e.Error == nil || e.Error.Code != "remote_rejected" {
		t.Fatalf("hook decline must exit 11 remote_rejected, got %d %+v", code, e)
	}
	if !strings.Contains(e.Error.Message, "seed policy says no") {
		t.Fatalf("the remote's reason must ride the message, got %q", e.Error.Message)
	}
	if after := remoteTip(t, remote); after != before {
		t.Fatal("rejected push must land nothing")
	}
}

func TestRemoteAppendVanishedRefRegression(t *testing.T) {
	dir, priv, _ := writeKeys(t)
	remote := bareRemote(t)
	seedRemoteGenesis(t, remote)
	state := filepath.Join(dir, "state")

	if e, code := runEnv(t, "ledger", "append", "--remote", remote, "--state", state,
		"--key", priv, "--verb", "progress.milestone", "--subject", "c-0001", "--payload", `{"n": 1}`); code != 0 {
		t.Fatalf("first append failed: %d %+v", code, e)
	}
	if out, err := exec.Command("git", "--git-dir", remote, "update-ref", "-d", remoteRef).CombinedOutput(); err != nil {
		t.Fatalf("delete ref: %v %s", err, out)
	}
	e, code := runEnv(t, "ledger", "append", "--remote", remote, "--state", state,
		"--key", priv, "--verb", "progress.milestone", "--subject", "c-0002", "--payload", `{"n": 2}`)
	if code != 12 || e.Error == nil || e.Error.Code != "head_regression" {
		t.Fatalf("vanished ref after a verified head must exit 12 head_regression, got %d %+v", code, e)
	}
}

// conformance: III.A freshness — a rollback landing mid-invocation
// (after the pre-flight verify, before the loop's push wins) refuses at
// exit 12 via the recorded verified head; nothing appends over the
// regression (#91 review).
func TestRemoteAppendMidInvocationRollbackRefuses(t *testing.T) {
	dir, priv, _ := writeKeys(t)
	remote := bareRemote(t)
	resolve := seedRemoteGenesis(t, remote)
	genesisTip := remoteTip(t, remote)
	libAppend(t, remote, resolve, "seed/0", "progress.milestone", "c-0001", `{"n": 1}`)

	rf := filepath.Join(remote, "rollback")
	if err := os.WriteFile(rf, []byte(genesisTip+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hook := fmt.Sprintf(`#!/bin/sh
r=$(head -n1 %[1]s 2>/dev/null)
if [ -n "$r" ]; then
  unset GIT_QUARANTINE_PATH
  git update-ref %[2]s "$r"
  : > %[1]s
fi
exit 0
`, rf, remoteRef)
	if err := os.WriteFile(filepath.Join(remote, "hooks", "pre-receive"), []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}

	e, code := runEnv(t, "ledger", "append", "--remote", remote, "--state", filepath.Join(dir, "state"),
		"--key", priv, "--verb", "progress.milestone", "--subject", "c-0002", "--payload", `{"n": 2}`)
	if code != 12 || e.Error == nil || e.Error.Code != "head_regression" {
		t.Fatalf("mid-invocation rollback must refuse at exit 12, got %d %+v", code, e)
	}
	if tip := remoteTip(t, remote); tip != genesisTip {
		t.Fatalf("nothing may append over the regression, ref at %s", tip)
	}
}

// Two appenders sharing one state dir (the no --state default shape)
// serialize on the state lock instead of corrupting the shared git
// index, tracking ref, or persisted-head cache (#93 review).
func TestRemoteAppendSharedStateSerializes(t *testing.T) {
	dir, priv, _ := writeKeys(t)
	remote := bareRemote(t)
	seedRemoteGenesis(t, remote)
	state := filepath.Join(dir, "shared-state")

	type outcome struct {
		code int
		out  string
	}
	results := make(chan outcome, 2)
	for i := 0; i < 2; i++ {
		go func(i int) {
			var out, errOut strings.Builder
			code := run([]string{"ledger", "append", "--remote", remote, "--state", state,
				"--key", priv, "--verb", "progress.milestone", "--subject", fmt.Sprintf("c-%04d", i),
				"--payload", fmt.Sprintf(`{"n": %d}`, i)}, &out, &errOut)
			results <- outcome{code, out.String()}
		}(i)
	}
	for i := 0; i < 2; i++ {
		r := <-results
		if r.code != 0 {
			t.Fatalf("shared-state appenders must serialize and land, got %d %s", r.code, r.out)
		}
	}
	c, err := gitref.NewClient(t.TempDir(), remote, remoteRef)
	if err != nil {
		t.Fatal(err)
	}
	tip, err := c.Fetch()
	if err != nil {
		t.Fatal(err)
	}
	ld := t.TempDir()
	if err := c.Materialize(tip, ld); err != nil {
		t.Fatal(err)
	}
	if e, code := runEnv(t, "ledger", "verify", "--ledger", ld); code != 0 || e.Result["count"].(float64) != 3 {
		t.Fatalf("both appends must land: %d %+v", code, e)
	}
}

// conformance: III.F — the lifecycle happy path admits through the CLI
// (the cooperative client runs the shared rule set), done is reached
// only through merge.observed, and illegal jumps refuse exit 3 naming
// subject, state, and verb (plans/os-d69a6c91.md step 7).
func TestRemoteAppendLifecycle(t *testing.T) {
	dir, priv, _ := writeKeys(t)
	remote := bareRemote(t)
	resolve := seedRemoteGenesis(t, remote)
	libAppend(t, remote, resolve, "seed/0", ledger.UpgradeVerb, "system", `{"to": "seed/1"}`)
	state := filepath.Join(dir, "state")

	steps := []struct{ verb, subject, payload string }{
		{"intent.filed", "c-1", `{"intent": "fix", "tier": "standard", "budget": "s", "routing": "core"}`},
		{"contract.specified", "c-1", `{"acceptance": "specs/c1.md @ abc"}`},
		{"claim.taken", "c-1", `{}`},
		// The claim admitted at position 4 (genesis, upgrade, filed,
		// specified, taken): the submission cites that fence and, like
		// every deliberate exit, carries its handoff packet.
		{"submission.made", "c-1", `{"branch": "seed/c-1", "fence": "4", "packet": {"acceptance": ["c-1 resumes"], "decisions": [], "base": "1234567..1234567", "refs": [], "findings": []}}`},
		{"merge.observed", "c-1", `{"pr": "9"}`},
	}
	for _, s := range steps {
		e, code := runEnv(t, "ledger", "append", "--remote", remote, "--state", state,
			"--key", priv, "--verb", s.verb, "--subject", s.subject, "--payload", s.payload)
		if code != 0 || !e.OK {
			t.Fatalf("%s must land through the CLI: %d %+v", s.verb, code, e)
		}
	}

	before := remoteTip(t, remote)
	e, code := runEnv(t, "ledger", "append", "--remote", remote, "--state", state,
		"--key", priv, "--verb", "claim.taken", "--subject", "c-1", "--payload", `{}`)
	if code != 3 || e.Error == nil || e.Error.Code != "invalid_transition" ||
		!strings.Contains(e.Error.Message, "c-1") || !strings.Contains(e.Error.Message, "done") ||
		!strings.Contains(e.Error.Message, "claim.taken") {
		t.Fatalf("an illegal transition must refuse exit 3 naming subject, state, and verb: %d %+v", code, e)
	}
	if after := remoteTip(t, remote); after != before {
		t.Fatal("refused draft must push nothing")
	}

	// Completeness is a shape refusal: an incomplete filing never
	// leaves the client.
	e, code = runEnv(t, "ledger", "append", "--remote", remote, "--state", state,
		"--key", priv, "--verb", "intent.filed", "--subject", "c-2", "--payload", `{"intent": "x"}`)
	if code != 8 || e.Error == nil || !strings.Contains(e.Error.Message, "incomplete") {
		t.Fatalf("an incomplete filing must refuse as a shape refusal: %d %+v", code, e)
	}
	if after := remoteTip(t, remote); after != before {
		t.Fatal("refused draft must push nothing")
	}
}

// conformance: III.F — claims are exclusive with fencing at the CLI:
// contention returns the structured envelope, a fence-free holder
// event refuses 6, and exclusive verbs refuse offline (the cooperative
// client's online-only seam; plans/os-5dc16a7c.md).
func TestRemoteClaimFencingCLI(t *testing.T) {
	dir, priv, _ := writeKeys(t)
	remote := bareRemote(t)
	resolve := seedRemoteGenesis(t, remote)
	libAppend(t, remote, resolve, "seed/0", ledger.UpgradeVerb, "system", `{"to": "seed/1"}`)
	state := filepath.Join(dir, "state")
	appendCLI := func(verb, subject, payload string) (ledgerEnv, int) {
		return runEnv(t, "ledger", "append", "--remote", remote, "--state", state,
			"--key", priv, "--verb", verb, "--subject", subject, "--payload", payload)
	}
	if _, code := appendCLI("intent.filed", "c-1", `{"intent": "fix", "tier": "standard", "budget": "s", "routing": "core"}`); code != 0 {
		t.Fatal("filing failed")
	}
	if _, code := appendCLI("contract.specified", "c-1", `{"acceptance": "specs/c1.md @ abc"}`); code != 0 {
		t.Fatal("specification failed")
	}
	if _, code := appendCLI("claim.taken", "c-1", `{}`); code != 0 {
		t.Fatal("claim failed")
	}

	// Contention: the same key re-claiming its held contract loses
	// with the structured envelope (one claim at a time, whoever asks).
	e, code := appendCLI("claim.taken", "c-1", `{}`)
	if code != 2 || e.Error == nil || e.Error.Code != "contention" ||
		!strings.Contains(e.Error.Message, "fence 4") {
		t.Fatalf("a rival claim must refuse structured contention at 2, got %d %+v", code, e)
	}

	// The holder's fence-free milestone refuses 6 naming the fence.
	e, code = appendCLI("progress.milestone", "c-1", `{"n": 1}`)
	if code != 6 || e.Error == nil || e.Error.Code != "fenced_out" ||
		!strings.Contains(e.Error.Message, "4") {
		t.Fatalf("a fence-free holder event must refuse 6, got %d %+v", code, e)
	}
	if _, code := appendCLI("progress.milestone", "c-1", `{"n": 1, "fence": "4"}`); code != 0 {
		t.Fatal("the holder citing the active fence must admit")
	}

	// A packetless exit is a shape refusal: the packet obligation
	// travels with the four deliberate exits (plans/os-b07b0f59.md).
	e, code = appendCLI("claim.released", "c-1", `{"fence": "4"}`)
	if code != 8 || e.Error == nil || !strings.Contains(e.Error.Message, "packet") {
		t.Fatalf("a packetless exit must refuse as a shape refusal naming the packet, got %d %+v", code, e)
	}

	// Offline: an exclusive verb refuses through the local dev tool
	// with the charter's posture in the message; reading stays local.
	ld := filepath.Join(t.TempDir(), "ledger")
	if _, code := runEnv(t, "init", "--ledger", ld, "--key", priv); code != 0 {
		t.Fatal("init failed")
	}
	e, code = runEnv(t, "ledger", "append", "--ledger", ld, "--key", priv,
		"--verb", "claim.taken", "--subject", "c-9", "--payload", `{}`)
	if code != 2 || e.Error == nil || e.Error.Code != "contention" ||
		!strings.Contains(e.Error.Message, "online-only") {
		t.Fatalf("an offline exclusive verb must refuse with the online-only posture, got %d %+v", code, e)
	}
	if _, code := runEnv(t, "ledger", "verify", "--ledger", ld); code != 0 {
		t.Fatal("offline reading is unaffected")
	}
}
