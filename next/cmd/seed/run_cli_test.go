package main

// The executor drills end-to-end (plans/os-1dad487d.md;
// next/spec/executors.md): the spend bracket (reserve, start,
// provision, meter, settle), the local worktree adapter's surfaces,
// and the Phase 7 exit's remaining named drill — disposability under
// a real SIGKILL after an admitted synchronization, with the loss
// window exactly the observation lines after the last sync.

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/shaunlmason/open-seed/next/executor"
	"github.com/shaunlmason/open-seed/next/internal/admit"
	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
	"github.com/shaunlmason/open-seed/next/internal/obs"
)

// admitAppendErr is admitAppend without the testing.T: the helper process
// runs the same online admission sequence (context at the tip, full
// Check, append with the context resolver) and reports errors as
// values.
func admitAppendErr(ld string, key ed25519.PrivateKey, verb, subject, payload string) (int, error) {
	store, err := ledger.Open(ld)
	if err != nil {
		return -1, err
	}
	ctx, err := admit.ContextAt(store)
	if err != nil {
		return -1, err
	}
	fp, err := event.Fingerprint(key.Public().(ed25519.PublicKey))
	if err != nil {
		return -1, err
	}
	rec, err := event.Sign(event.Event{
		V: ctx.Active, TS: time.Now().UTC().Format(time.RFC3339), Actor: fp,
		Verb: verb, Subject: subject, Payload: json.RawMessage(payload), Prev: ctx.Tip,
	}, key)
	if err != nil {
		return -1, err
	}
	if err := admit.Check(ctx, rec); err != nil {
		return -1, err
	}
	return store.Append(rec, ctx.Resolve)
}

// TestHelperWorker is the killable worker process (the test-binary
// re-exec pattern): inside the provisioned workspace it meters, lands
// the admitted synchronization, and keeps metering until killed. It
// never cleans up — SIGKILL is the point.
func TestHelperWorker(t *testing.T) {
	if os.Getenv("SEED_RUN_HELPER") != "1" {
		t.Skip("helper process, spawned by the disposability drill")
	}
	ld := os.Getenv("SEED_RUN_LEDGER")
	subject := os.Getenv("SEED_RUN_SUBJECT")
	obsDir := os.Getenv("SEED_RUN_OBS")
	actor := os.Getenv("SEED_RUN_ACTOR")
	fence, _ := strconv.Atoi(os.Getenv("SEED_RUN_FENCE"))
	seedByte, _ := strconv.Atoi(os.Getenv("SEED_RUN_KEY"))
	rng := os.Getenv("SEED_RUN_RNG")
	key := workerRawKey(byte(seedByte))
	meter := func(units int, step string) {
		_ = obs.Append(obsDir, actor, fmt.Sprintf("%d", fence), obs.Line{
			TS: time.Now().UTC().Format(time.RFC3339), Subject: subject, Step: step, Units: units,
		})
	}
	meter(3, "warmup")
	if _, err := admitAppendErr(ld, key, "submission.made", subject, fmt.Sprintf(
		`{"fence": "%d", "packet": {"acceptance": ["%s ok"], "decisions": [], "base": %q, "refs": [], "findings": []}}`,
		fence, subject, rng)); err != nil {
		os.Exit(7)
	}
	for {
		meter(1, "post-sync")
		time.Sleep(20 * time.Millisecond)
	}
}

// conformance: III.H — disposability after the last admitted
// synchronization, drilled with a randomized SIGKILL of a real worker
// process: no admitted fact is lost, the contract completes elsewhere
// from the surviving ledger alone, and the loss window is exactly the
// observation lines after the last sync. Also the spend bracket:
// execution is fenced to the reservation (reserve, run.started,
// Provision) and metering settles to the ledger at run end.
func TestDisposabilityDrill(t *testing.T) {
	seed := time.Now().UnixNano()
	r := rand.New(rand.NewSource(seed))
	site := []string{"after-sync", "after-loss", "after-settled"}[r.Intn(3)]
	t.Logf("disposability drill: seed %d, kill site %s", seed, site)

	ld, src, base, specCommit, head, priv, _, keys, fps := offerLedger(t)
	rng := base + ".." + head
	offerFile(t, ld, priv, specCommit, "c-1")
	fencePos, err := admitAppend(t, ld, workerRawKey(22), "claim.taken", "c-1", `{}`)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	fence := fmt.Sprintf("%d", fencePos)
	e, code := runEnv(t, "ledger", "append", "--ledger", ld, "--key", keys["workerA"],
		"--verb", "budget.reserve", "--subject", "c-1", "--payload", `{"amount": "10", "fence": "`+fence+`"}`)
	if code != 0 {
		t.Fatalf("reserve: %d %+v", code, e)
	}
	reservation := *e.Position

	// The bracket: Provision refuses before the admitted start.
	obsDir := filepath.Join(t.TempDir(), "obs")
	var lw executor.LocalWorktree
	spec := executor.ProvisionSpec{
		Ledger: ld, Repo: src, Base: base, Subject: "c-1", Actor: fps["workerA"],
		Fence: fencePos, Started: -1, Packet: []byte(`{"drill": "packet"}`), ObsDir: obsDir,
	}
	if _, err := lw.Provision(spec); !errors.Is(err, executor.ErrNoAdmittedStart) {
		t.Fatalf("Provision refuses without the admitted run.started: %v", err)
	}

	// Fold presence is never admission (review finding on the task
	// PR): raw-pushed starts fold into the list but fail the
	// position-accurate boundary — the holder lacks the verb's
	// capability, and a supervisor start citing no valid reservation
	// fails its citation — so Provision refuses both.
	rawStart := rawAppend(t, ld, workerRawKey(22), "run.started", "c-1",
		`{"fence": "`+fence+`", "reservation": "`+reservation+`"}`)
	spec.Started = rawStart
	if _, err := lw.Provision(spec); !errors.Is(err, executor.ErrNoAdmittedStart) {
		t.Fatalf("a raw holder-signed start provisions nothing: %v", err)
	}
	rawCite := rawAppend(t, ld, workerRawKey(21), "run.started", "c-1",
		`{"fence": "`+fence+`", "reservation": "999"}`)
	spec.Started = rawCite
	if _, err := lw.Provision(spec); !errors.Is(err, executor.ErrNoAdmittedStart) {
		t.Fatalf("a start citing no valid reservation provisions nothing: %v", err)
	}
	if e, code = runEnv(t, "ledger", "append", "--ledger", ld, "--key", keys["supervisor"],
		"--verb", "run.started", "--subject", "c-1", "--payload",
		`{"fence": "`+fence+`", "reservation": "`+reservation+`"}`); code != 0 {
		t.Fatalf("run.started: %d %+v", code, e)
	}
	started, _ := strconv.Atoi(*e.Position)
	spec.Started = started

	// A refused provision leaks no checkout (review finding on the
	// task PR): fail provisioning after the worktree add — the
	// observation root is a file, so the stream directory cannot be
	// made — and the worktree registration rolls back.
	worktrees := func() int {
		out, err := exec.Command("git", "-C", src, "worktree", "list", "--porcelain").Output()
		if err != nil {
			t.Fatal(err)
		}
		return strings.Count(string(out), "worktree ")
	}
	before := worktrees()
	badObs := filepath.Join(t.TempDir(), "obs-as-file")
	if err := os.WriteFile(badObs, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	badSpec := spec
	badSpec.ObsDir = badObs
	if _, err := lw.Provision(badSpec); err == nil {
		t.Fatal("provisioning must fail when the observation root is a file")
	}
	if got := worktrees(); got != before {
		t.Fatalf("a failed provision rolls its worktree back: %d registered, want %d", got, before)
	}

	run, err := lw.Provision(spec)
	if err != nil {
		t.Fatalf("Provision against the admitted start: %v", err)
	}
	if b, err := os.ReadFile(filepath.Join(run.Workspace(), ".seed-run", "packet.json")); err != nil || string(b) != `{"drill": "packet"}` {
		t.Fatalf("the packet lands in the workspace: %v %q", err, b)
	}
	if got := lw.Tuple(); got.Runtime != "local-worktree/v0" {
		t.Fatalf("the v0 tuple stub: %+v", got)
	}
	if err := lw.Wake(fps["workerA"]); err != nil {
		t.Fatalf("wake is the advisory no-op: %v", err)
	}
	if err := run.Meter(2, "provisioned"); err != nil {
		t.Fatalf("meter: %v", err)
	}

	// The killable worker, running inside the workspace.
	cmd := exec.Command(os.Args[0], "-test.run", "TestHelperWorker$", "-test.v")
	cmd.Dir = run.Workspace()
	cmd.Env = append(os.Environ(),
		"SEED_RUN_HELPER=1", "SEED_RUN_LEDGER="+ld, "SEED_RUN_SUBJECT=c-1",
		"SEED_RUN_OBS="+obsDir, "SEED_RUN_ACTOR="+fps["workerA"],
		"SEED_RUN_FENCE="+fence, "SEED_RUN_KEY=22", "SEED_RUN_RNG="+rng)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	synced := false
	for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); time.Sleep(30 * time.Millisecond) {
		st, failEnv := loadVerdictState(ld)
		if failEnv != nil {
			continue
		}
		if s, ok := st.fold.State("c-1"); ok && s.State == "review" {
			synced = true
			break
		}
	}
	if !synced {
		_ = cmd.Process.Kill()
		t.Fatal("the worker never landed the admitted synchronization")
	}
	if site == "after-loss" {
		time.Sleep(150 * time.Millisecond) // accumulate post-sync loss lines
	}
	if site == "after-settled" {
		if e, code = runEnv(t, "ledger", "append", "--ledger", ld, "--key", keys["supervisor"],
			"--verb", "run.settled", "--subject", "c-1", "--payload",
			`{"fence": "`+fence+`", "units": "6", "lines": "4"}`); code != 0 {
			t.Fatalf("run.settled before the kill: %d %+v", code, e)
		}
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()

	// Every admitted fact survives: the chain verifies from genesis.
	st, failEnv := loadVerdictState(ld)
	if failEnv != nil {
		t.Fatalf("the chain must verify after the kill: %+v", failEnv)
	}
	if s, _ := st.fold.State("c-1"); s.State != "review" {
		t.Fatalf("the admitted synchronization survives: %s", s.State)
	}

	// The contract completes elsewhere, from the surviving ledger
	// alone: verdict, request, observed — none of it needs the dead
	// worker or its workspace.
	e, code = runEnv(t, "verdict", "render", "--ledger", ld, "--subject", "c-1", "--repo", src,
		"--key", keys["verifier"], "--verdict", "pass")
	if code != 0 {
		t.Fatalf("verdict after the kill: %d %+v", code, e)
	}
	verdictPos := *e.Position
	if e, code = runEnv(t, "ledger", "append", "--ledger", ld, "--key", priv,
		"--verb", "merge.requested", "--subject", "c-1", "--payload", `{"verdict": "`+verdictPos+`"}`); code != 0 {
		t.Fatalf("request after the kill: %d %+v", code, e)
	}
	if e, code = runEnv(t, "ledger", "append", "--ledger", ld, "--key", priv,
		"--verb", "merge.observed", "--subject", "c-1", "--payload", `{"merged": "`+head+`", "pr": "pr/7"}`); code != 0 {
		t.Fatalf("observed after the kill: %d %+v", code, e)
	}
	if site != "after-settled" {
		if e, code = runEnv(t, "ledger", "append", "--ledger", ld, "--key", keys["supervisor"],
			"--verb", "run.settled", "--subject", "c-1", "--payload",
			`{"fence": "`+fence+`", "units": "6", "lines": "4"}`); code != 0 {
			t.Fatalf("run.settled after the kill: %d %+v", code, e)
		}
	}

	// The loss window: the stream holds the worker's lines — the
	// warmup and the unsynchronized post-sync tail — and completion
	// needed none of them.
	snap, err := obs.Load(obsDir)
	if err != nil {
		t.Fatal(err)
	}
	stream, ok := snap.StreamFor(fps["workerA"], fence)
	if !ok || len(stream.Lines) < 2 {
		t.Fatalf("the stream holds the metered lines: %+v", stream)
	}
	post := 0
	for _, l := range stream.Lines {
		if l.Step == "post-sync" {
			post++
		}
	}
	if post == 0 {
		t.Fatal("the post-sync loss lines exist in the stream and nowhere else")
	}

	// Disposing the dead run's workspace loses nothing admitted.
	if err := run.Dispose(); err != nil {
		t.Fatalf("dispose after the kill: %v", err)
	}
	if _, failEnv := loadVerdictState(ld); failEnv != nil {
		t.Fatalf("the chain verifies after disposal: %+v", failEnv)
	}
}
