// The cooperative posture (docs/next-build-plan.md Phase 2 item 2;
// plans/os-895bf828.md): the remote mode of seed ledger append validates
// every draft through the shared admission rule set before any bytes
// reach the remote, then rides gitref's optimistic append loop. The
// consequence of cooperation is honest: nothing here stops a client that
// skips validation from pushing; the enforced posture (2.3) closes that
// hole at the remote.
package main

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/shaunlmason/open-seed/next/internal/admit"
	"github.com/shaunlmason/open-seed/next/internal/envelope"
	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/genesis"
	"github.com/shaunlmason/open-seed/next/internal/gitref"
	"github.com/shaunlmason/open-seed/next/internal/halt"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
	"github.com/shaunlmason/open-seed/next/internal/version"
)

// remoteMaxAttempts bounds the optimistic loop's races per invocation.
const remoteMaxAttempts = 5

func runLedgerAppendRemote(remote, refName, stateDir, verb, subject, payload, supported string, signer ed25519.PrivateKey, stdout, stderr io.Writer) int {
	if stateDir == "" {
		cache, err := os.UserCacheDir()
		if err != nil {
			return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("no --state and no user cache dir: %v", err)), stdout, stderr)
		}
		stateDir = filepath.Join(cache, "seed", "gitref")
	}
	// One invocation owns the state dir at a time: the git dir, tracking
	// ref, and persisted-head cache inside it are not safe under
	// concurrent writers, and serializing here keeps a slow writer from
	// regressing a newer verified head (#93 review). Racing appenders
	// use distinct --state dirs; the remote race is theirs to lose.
	unlock, err := lockStateDir(stateDir)
	if err != nil {
		return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", fmt.Sprintf("cannot lock client state: %v", err)), stdout, stderr)
	}
	defer unlock()
	client, err := gitref.NewClient(stateDir, remote, refName)
	if err != nil {
		return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", fmt.Sprintf("cannot prepare client state: %v", err)), stdout, stderr)
	}

	// Pre-flight: fetch and verify the remote stream once to learn the
	// resolver and the active protocol version the draft must carry (the
	// #85 review discipline, applied to the remote tip).
	tip, err := client.Fetch()
	if err != nil {
		return render(remoteFailureEnvelope(err), stdout, stderr)
	}
	if tip == "" {
		return render(envelope.Fail(envelope.ExitChainInvalid, "chain_invalid",
			"remote ledger ref is empty: no genesis to append onto"), stdout, stderr)
	}
	workDir, err := os.MkdirTemp("", "seed-remote-append-*")
	if err != nil {
		return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()), stdout, stderr)
	}
	defer os.RemoveAll(workDir)
	if err := client.Materialize(tip, workDir); err != nil {
		return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", fmt.Sprintf("cannot materialize remote tip: %v", err)), stdout, stderr)
	}
	store, err := ledger.Open(workDir)
	if err != nil {
		return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()), stdout, stderr)
	}
	resolve, _, err := genesis.Bootstrap(store)
	if err != nil {
		return render(envelope.Fail(envelope.ExitChainInvalid, "chain_invalid", err.Error()), stdout, stderr)
	}
	supportedSet := map[string]bool{}
	for _, v := range version.Supported() {
		supportedSet[v] = true
	}
	var vopts []ledger.VerifyOption
	var aopts []admit.Option
	if supported != "" {
		vs := strings.Split(supported, ",")
		vopts = append(vopts, ledger.WithSupportedVersions(vs...))
		aopts = append(aopts, admit.WithSupportedVersions(vs...))
		supportedSet = map[string]bool{}
		for _, v := range vs {
			supportedSet[v] = true
		}
	}
	rep, err := store.VerifyFromGenesis(resolve, vopts...)
	if err != nil {
		var fail *ledger.Failure
		if errors.As(err, &fail) {
			return render(failureEnvelope(fail), stdout, stderr)
		}
		return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()), stdout, stderr)
	}
	// Persist the verified tip before the loop: without it, a rollback
	// landing between this pre-flight and the loop's own fetch would be
	// accepted by a fresh client (#91 review).
	if err := client.RecordVerifiedHead(tip); err != nil {
		return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()), stdout, stderr)
	}
	if !supportedSet[rep.ActiveVersion] {
		env := envelope.Fail(envelope.ExitVersionMismatch, ledger.ReasonVersionUnsupported,
			fmt.Sprintf("the version active at the remote tip is %q; this build appends only %s", rep.ActiveVersion, supportedList(supportedSet)))
		return render(stampTip(env, rep.Count), stdout, stderr)
	}
	fp, err := event.Fingerprint(signer.Public().(ed25519.PublicKey))
	if err != nil {
		return render(envelope.Fail(envelope.ExitUsage, "usage", err.Error()), stdout, stderr)
	}

	// The loop re-links prev and re-signs per attempt; admission runs
	// against the refreshed tip before every push. The signing closure
	// keeps the last record so the envelope can report the landed hash,
	// and the validate wrapper records the tip position each refusal was
	// computed at so the envelope stamps it (#93 review).
	inner := admit.Validate(aopts...)
	validate := func(store *ledger.Store, rec *event.Record) error {
		verr := inner(store, rec)
		if verr == nil {
			return nil
		}
		if _, n, terr := store.Tip(); terr == nil && n > 0 {
			return &refusalAt{count: n, err: verr}
		}
		return verr
	}
	var lastRec *event.Record
	res, err := client.AppendLoop(gitref.Draft{
		V:       rep.ActiveVersion,
		TS:      time.Now().UTC().Format(time.RFC3339),
		Actor:   fp,
		Verb:    verb,
		Subject: subject,
		Payload: []byte(payload),
	}, func(e event.Event) (*event.Record, error) {
		rec, serr := event.Sign(e, signer)
		lastRec = rec
		return rec, serr
	}, resolve, validate, remoteMaxAttempts, vopts...)
	if err != nil {
		return render(remoteFailureEnvelope(err), stdout, stderr)
	}
	hash, err := lastRec.Event.Hash()
	if err != nil {
		return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()), stdout, stderr)
	}
	env := envelope.OK(map[string]any{
		"appended": hash,
		"verb":     verb,
		"commit":   res.Commit,
		"attempts": res.Attempts,
	})
	return render(stampTip(env, res.Position+1), stdout, stderr)
}

// lockStateDir takes the exclusive advisory lock guarding one client
// state dir, blocking until it is free, and returns the release.
func lockStateDir(stateDir string) (func(), error) {
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(stateDir, ".lock"), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, err
	}
	return func() {
		syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
	}, nil
}

// refusalAt carries the materialized tip position an admission refusal
// was computed at, so the envelope can stamp it.
type refusalAt struct {
	count int
	err   error
}

func (r *refusalAt) Error() string { return r.err.Error() }
func (r *refusalAt) Unwrap() error { return r.err }

// remoteFailureEnvelope maps the cooperative client's refusals
// (plans/os-895bf828.md step 2): admission refusals keep their typed
// exits (7/8/9/10), the loop's own outcomes land on contention (2),
// the remote's policy rejection (11, reason verbatim), head regression
// (12, a freshness refusal distinct from chain trouble), or
// unavailable (5).
func remoteFailureEnvelope(err error) *envelope.Envelope {
	var at *refusalAt
	if errors.As(err, &at) {
		return stampTip(remoteFailureEnvelope(at.err), at.count)
	}
	var herr *halt.HaltedError
	if errors.As(err, &herr) {
		return envelope.Fail(envelope.ExitHalted, "halted", err.Error())
	}
	var cerr *admit.ClassificationError
	if errors.As(err, &cerr) {
		return envelope.Fail(envelope.ExitClassificationRef, "classification_refused", err.Error())
	}
	var oog *admit.OutOfGrantError
	if errors.As(err, &oog) {
		return envelope.Fail(envelope.ExitOutOfGrant, "out_of_grant", err.Error())
	}
	var vin *admit.VerbInactiveError
	if errors.As(err, &vin) {
		return envelope.Fail(envelope.ExitInvalidTransition, "invalid_transition", err.Error())
	}
	var fail *ledger.Failure
	if errors.As(err, &fail) {
		return failureEnvelope(fail)
	}
	switch {
	case errors.Is(err, ledger.ErrUnknownActor):
		return envelope.Fail(envelope.ExitChainInvalid, "chain_invalid", err.Error())
	case errors.Is(err, gitref.ErrRetriesSpent):
		return envelope.Fail(envelope.ExitContention, "contention", err.Error())
	case errors.Is(err, gitref.ErrRemoteRejected):
		return envelope.Fail(envelope.ExitRemoteRejected, "remote_rejected", err.Error())
	case errors.Is(err, gitref.ErrHeadRegression):
		return envelope.Fail(envelope.ExitHeadRegression, "head_regression", err.Error())
	case errors.Is(err, gitref.ErrUnavailable):
		return envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error())
	}
	var ref *admit.Refusal
	if errors.As(err, &ref) {
		return envelope.Fail(envelope.ExitChainInvalid, "chain_invalid", err.Error())
	}
	return envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error())
}
