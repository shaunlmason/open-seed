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
	"flag"
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
	"github.com/shaunlmason/open-seed/next/internal/transition"
	"github.com/shaunlmason/open-seed/next/internal/version"
)

// remoteMaxAttempts bounds the optimistic loop's races per invocation.
const remoteMaxAttempts = 5

// remoteSession is one prepared view of the remote ledger: the client,
// the materialized and verified tip, and the records the loop verbs
// derive their arguments from. Callers must call close when done.
type remoteSession struct {
	client  *gitref.Client
	tip     string
	store   *ledger.Store
	resolve ledger.Resolver
	rep     *ledger.Report
	vopts   []ledger.VerifyOption
	aopts   []admit.Option
	close   func()
}

// openRemoteSession fetches, materializes and verifies the remote tip,
// persisting the verified head before any caller runs a loop against
// it. Sharing this prologue is what lets an argument be derived from
// the SAME view admission will judge the draft against: a fence read
// from a stale local copy would be wrong under exactly the contention
// that makes claiming online-only (plans/os-7e197768.md D3).
func openRemoteSession(remote, refName, stateDir, supported string) (*remoteSession, *envelope.Envelope) {
	if stateDir == "" {
		cache, err := os.UserCacheDir()
		if err != nil {
			return nil, envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("no --state and no user cache dir: %v", err))
		}
		stateDir = filepath.Join(cache, "seed", "gitref")
	}
	unlock, err := lockStateDir(stateDir)
	if err != nil {
		return nil, envelope.Fail(envelope.ExitUnavailable, "unavailable", fmt.Sprintf("cannot lock client state: %v", err))
	}
	done := []func(){unlock}
	fail := func(env *envelope.Envelope) (*remoteSession, *envelope.Envelope) {
		for i := len(done) - 1; i >= 0; i-- {
			done[i]()
		}
		return nil, env
	}
	client, err := gitref.NewClient(stateDir, remote, refName)
	if err != nil {
		return fail(envelope.Fail(envelope.ExitUnavailable, "unavailable", fmt.Sprintf("cannot prepare client state: %v", err)))
	}
	tip, err := client.Fetch()
	if err != nil {
		return fail(remoteFailureEnvelope(err))
	}
	if tip == "" {
		return fail(envelope.Fail(envelope.ExitChainInvalid, "chain_invalid", "remote ledger ref is empty: no genesis to append onto"))
	}
	workDir, err := os.MkdirTemp("", "seed-remote-append-*")
	if err != nil {
		return fail(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()))
	}
	done = append(done, func() { os.RemoveAll(workDir) })
	if err := client.Materialize(tip, workDir); err != nil {
		return fail(envelope.Fail(envelope.ExitUnavailable, "unavailable", fmt.Sprintf("cannot materialize remote tip: %v", err)))
	}
	store, err := ledger.Open(workDir)
	if err != nil {
		return fail(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()))
	}
	resolve, _, err := genesis.Bootstrap(store)
	if err != nil {
		return fail(envelope.Fail(envelope.ExitChainInvalid, "chain_invalid", err.Error()))
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
		var f *ledger.Failure
		if errors.As(err, &f) {
			return fail(failureEnvelope(f))
		}
		return fail(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()))
	}
	if err := client.RecordVerifiedHead(tip); err != nil {
		return fail(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()))
	}
	if !supportedSet[rep.ActiveVersion] {
		env := envelope.Fail(envelope.ExitVersionMismatch, ledger.ReasonVersionUnsupported,
			fmt.Sprintf("the version active at the remote tip is %q; this build appends only %s", rep.ActiveVersion, supportedList(supportedSet)))
		return fail(stampTip(env, rep.Count))
	}
	return &remoteSession{client: client, tip: tip, store: store,
		resolve: resolve, rep: rep, vopts: vopts, aopts: aopts,
		close: func() {
			for i := len(done) - 1; i >= 0; i-- {
				done[i]()
			}
		}}, nil
}

// pushDraft rides the optimistic append loop: prev is re-linked and the
// record re-signed per attempt, with admission re-run against the
// refreshed tip before every push.
//
// recheck, when given, runs FIRST against that same refreshed view. It
// is how a derived argument is re-examined after the tip moves: the
// loop verbs derive the fence and the reservation from the view the
// session opened at, and a rival appending mid-flight can make that
// value stale without making it inadmissible, which is the one case
// admission alone cannot catch (plans/os-9b3f3ef3.md D1).
func (s *remoteSession) pushDraft(verb, subject, payload string, signer ed25519.PrivateKey, fp string,
	recheck func(*admit.Context) error) (*event.Record, *gitref.Result, error) {
	// One context per attempt, shared by the recheck and the admission
	// check: the same view judging both, and one replay rather than two.
	validate := func(store *ledger.Store, rec *event.Record) error {
		ctx, cerr := admit.ContextAt(store, s.aopts...)
		if cerr != nil {
			return cerr
		}
		var verr error
		if recheck != nil {
			verr = recheck(ctx)
		}
		if verr == nil {
			verr = admit.Check(ctx, rec)
		}
		if verr == nil {
			return nil
		}
		return &refusalAt{count: ctx.Count, ctx: ctx, err: verr}
	}
	var lastRec *event.Record
	res, err := s.client.AppendLoop(gitref.Draft{
		V:       s.rep.ActiveVersion,
		TS:      time.Now().UTC().Format(time.RFC3339),
		Actor:   fp,
		Verb:    verb,
		Subject: subject,
		Payload: []byte(payload),
	}, func(e event.Event) (*event.Record, error) {
		rec, serr := event.Sign(e, signer)
		lastRec = rec
		return rec, serr
	}, s.resolve, validate, remoteMaxAttempts, s.vopts...)
	return lastRec, res, err
}

func runLedgerAppendRemote(remote, refName, stateDir, verb, subject, payload, supported string, signer ed25519.PrivateKey, stdout, stderr io.Writer) int {
	session, failEnv := openRemoteSession(remote, refName, stateDir, supported)
	if failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	defer session.close()
	fp, err := event.Fingerprint(signer.Public().(ed25519.PublicKey))
	if err != nil {
		return render(envelope.Fail(envelope.ExitUsage, "usage", err.Error()), stdout, stderr)
	}
	lastRec, res, err := session.pushDraft(verb, subject, payload, signer, fp, nil)
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

// refusalAt carries the view a refusal was computed at: the position,
// so the envelope stamps the tip it was judged against rather than the
// one the session opened at, and the context, so the affordances
// answering "then what may I do?" describe that same view.
type refusalAt struct {
	count int
	ctx   *admit.Context
	err   error
}

// refusalView returns the context a refusal was computed at, falling
// back to the caller's own when the failure carries none.
func refusalView(err error, fallback *admit.Context) *admit.Context {
	var at *refusalAt
	if errors.As(err, &at) && at.ctx != nil {
		return at.ctx
	}
	return fallback
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
	var div *derivedDivergence
	if errors.As(err, &div) {
		return div.env
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
	var itr *transition.InvalidTransitionError
	if errors.As(err, &itr) {
		return envelope.Fail(envelope.ExitInvalidTransition, "invalid_transition", err.Error())
	}
	var be *admit.BudgetError
	if errors.As(err, &be) && be.Exhausted {
		// Exhaustion only. Every other budget refusal falls through to
		// the catch-all below and keeps chain_invalid: a caller that
		// answers this code by asking for less must not also be
		// answering a malformed payload (plans/os-d03bde01.md D1).
		return envelope.Fail(envelope.ExitBudgetExhausted, "budget_exhausted", err.Error())
	}
	var ce *admit.ContentionError
	if errors.As(err, &ce) {
		return envelope.Fail(envelope.ExitContention, "contention", err.Error())
	}
	var fe *admit.FenceError
	if errors.As(err, &fe) {
		return envelope.Fail(envelope.ExitFenced, "fenced_out", err.Error())
	}
	var nie *admit.NotIndependentError
	if errors.As(err, &nie) {
		return envelope.Fail(envelope.ExitNotIndependent, "not_independent", err.Error())
	}
	var pre *transition.PlanRequiredError
	if errors.As(err, &pre) {
		return envelope.Fail(envelope.ExitPlanRequired, "plan_required", err.Error())
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

// readPosture is the transport half a position-stamped READ shares:
// `--ledger` xor `--remote`, the same shape the loop verbs already
// take. A lane orients and acts in ONE posture or it orients against a
// copy nothing refreshes, and `claim take` is remote-only because only
// the push round-trip can order two rivals: a read that could not
// follow it there would leave the loop calling a stale position
// authoritative (plans/os-abb206c8.md D3).
type readPosture struct {
	dir       *string
	remote    *string
	refName   *string
	stateDir  *string
	supported *string
}

func bindReadPosture(fs *flag.FlagSet) *readPosture {
	return &readPosture{
		dir:       fs.String("ledger", "", "ledger directory"),
		remote:    fs.String("remote", "", "remote ledger repository: the posture a lane works in"),
		refName:   fs.String("ref", "refs/seed/ledger", "remote ledger ref"),
		stateDir:  fs.String("state", "", "client state dir for the persisted verified head (default: user cache)"),
		supported: fs.String("supported", "", "comma-separated supported protocol versions (default: this build's)"),
	}
}

// resolved reports whether exactly one posture was named. Neither is
// as wrong as both: a read with no ledger has nothing to derive from,
// and a read naming two has no answer to which one it stamped.
func (r *readPosture) resolved() bool { return (*r.dir == "") != (*r.remote == "") }

// open builds the read model in whichever posture was named, plus the
// admission context the affordance stamp derives from. Both come from
// the SAME store: a read that stamped affordances from one view and
// reported obligations from another would be two reads wearing one
// position.
//
// The returned closer is always safe to call and releases the remote
// session's lock and workdir on that path; on the local path it is a
// no-op, so callers defer it unconditionally.
func (r *readPosture) open() (*verdictState, *admit.Context, func(), *envelope.Envelope) {
	noop := func() {}
	var aopts []admit.Option
	if *r.supported != "" {
		aopts = append(aopts, admit.WithSupportedVersions(strings.Split(*r.supported, ",")...))
	}
	if *r.remote == "" {
		store, failEnv := openStoreReadOnly(*r.dir)
		if failEnv != nil {
			return nil, nil, noop, failEnv
		}
		resolve, _, err := genesis.Bootstrap(store)
		if err != nil {
			return nil, nil, noop, envelope.Fail(envelope.ExitChainInvalid, "chain_invalid", err.Error())
		}
		var vopts []ledger.VerifyOption
		if *r.supported != "" {
			vopts = append(vopts, ledger.WithSupportedVersions(strings.Split(*r.supported, ",")...))
		}
		st, failEnv := verdictStateAt(store, resolve, vopts...)
		if failEnv != nil {
			return nil, nil, noop, failEnv
		}
		// A context the boundary refuses to build is not fatal to a
		// READ: the obligations and windows above are still honest, and
		// the affordance block is the part that goes missing.
		ctx, err := admit.ContextAt(store, aopts...)
		if err != nil {
			ctx = nil
		}
		return st, ctx, noop, nil
	}
	rs, failEnv := openRemoteSession(*r.remote, *r.refName, *r.stateDir, *r.supported)
	if failEnv != nil {
		return nil, nil, noop, failEnv
	}
	st, failEnv := verdictStateAt(rs.store, rs.resolve, rs.vopts...)
	if failEnv != nil {
		rs.close()
		return nil, nil, noop, failEnv
	}
	ctx, err := admit.ContextAt(rs.store, rs.aopts...)
	if err != nil {
		ctx = nil
	}
	return st, ctx, rs.close, nil
}
