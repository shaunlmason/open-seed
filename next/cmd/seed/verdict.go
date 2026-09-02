// The verdict verbs (plans/os-f6d2c267.md; next/spec/verdicts.md):
// receipt computes and stores the receipt for a review subject's bound
// submission; render recomputes it fresh, derives the permissible
// verdict from the transcripts it just executed, and appends the
// signed verdict.rendered through local admission; check recomputes
// from the submission head and fails on mismatch against the latest
// rendered verdict — the recompute-and-mismatch surface. All three
// are cooperative dev tools over a local ledger, the obs/ledger-append
// posture; the boundary enforces the same rules regardless.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/shaunlmason/open-seed/next/internal/admit"
	"github.com/shaunlmason/open-seed/next/internal/artifact"
	"github.com/shaunlmason/open-seed/next/internal/curation"
	"github.com/shaunlmason/open-seed/next/internal/envelope"
	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/genesis"
	"github.com/shaunlmason/open-seed/next/internal/keyring"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
	"github.com/shaunlmason/open-seed/next/internal/transition"
	"github.com/shaunlmason/open-seed/next/internal/verdict"
)

func runVerdict(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "verdict requires a subverb: receipt, render, or check"), stdout, stderr)
	}
	switch args[0] {
	case "receipt":
		return runVerdictReceipt(args[1:], stdout, stderr)
	case "render":
		return runVerdictRender(args[1:], stdout, stderr)
	case "check":
		return runVerdictCheck(args[1:], stdout, stderr)
	}
	return render(envelope.Fail(envelope.ExitUsage, "usage", "verdict requires a subverb: receipt, render, or check"), stdout, stderr)
}

// verdictState is the read side every verdict verb shares: the
// verified records, the lifecycle fold, and the chain report.
type verdictState struct {
	records []*event.Record
	fold    *transition.Fold
	table   *transition.Table
	tip     string
	active  string
	count   int
}

func loadVerdictState(dir string) (*verdictState, *envelope.Envelope) {
	store, failEnv := openStoreReadOnly(dir)
	if failEnv != nil {
		return nil, failEnv
	}
	resolve, _, err := genesis.Bootstrap(store)
	if err != nil {
		return nil, envelope.Fail(envelope.ExitChainInvalid, "chain_invalid", err.Error())
	}
	return verdictStateAt(store, resolve)
}

// verdictStateAt builds the read model from an ALREADY-OPEN store, so
// the local and remote postures share one derivation rather than
// growing a second copy that can disagree with this one. The remote
// caller passes the session's own resolver and verify options: the
// view a lane orients from must be the view its acts are judged
// against, which is the whole reason the read learned the posture
// (plans/os-abb206c8.md D3).
func verdictStateAt(store *ledger.Store, resolve ledger.Resolver, vopts ...ledger.VerifyOption) (*verdictState, *envelope.Envelope) {
	var records []*event.Record
	opts := append([]ledger.VerifyOption{}, vopts...)
	opts = append(opts, ledger.WithObserver(func(pos int, r *event.Record) {
		records = append(records, r)
	}))
	rep, err := store.VerifyFromGenesis(resolve, opts...)
	if err != nil {
		var fail *ledger.Failure
		if errors.As(err, &fail) {
			return nil, failureEnvelope(fail)
		}
		return nil, envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error())
	}
	table, err := transition.Default()
	if err != nil {
		return nil, envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error())
	}
	return &verdictState{records: records, fold: table.FoldRecords(records), table: table,
		tip: rep.Tip, active: rep.ActiveVersion, count: rep.Count}, nil
}

// verdictInput assembles the receipt computation's input: the bound
// submission's packet names the range, the fold carries the plan
// anchor and acceptance, and everything else is recomputed
// (enumerable inputs). requireReview gates receipt and render to
// review subjects; check passes false, because reconciliation runs
// exactly after merge.observed has moved the contract on and the
// fold retains the bound submission (review finding on the task PR).
func (st *verdictState) verdictInput(subject, repo string, timeout time.Duration, requireReview bool) (verdict.Input, transition.SubjectState, *envelope.Envelope) {
	s, ok := st.fold.State(subject)
	if !ok {
		if !requireReview {
			return verdict.Input{}, s, envelope.Fail(envelope.ExitNotFound, "not_found",
				fmt.Sprintf("no contract %s in the fold", subject))
		}
		return verdict.Input{}, s, envelope.Fail(envelope.ExitInvalidTransition, "invalid_transition",
			(&transition.InvalidTransitionError{Subject: subject, Verb: transition.VerdictRenderedVerb}).Error())
	}
	if requireReview && s.State != "review" {
		return verdict.Input{}, s, envelope.Fail(envelope.ExitInvalidTransition, "invalid_transition",
			(&transition.InvalidTransitionError{Subject: subject, From: s.State, Verb: transition.VerdictRenderedVerb}).Error())
	}
	// The construction is the verifier's own (verdict.InputFor), shared
	// with the qualification derivation that recomputes receipts.
	in, err := verdict.InputFor(st.records, st.fold, s, subject, repo, timeout)
	if errors.Is(err, verdict.ErrNoSubmission) {
		return verdict.Input{}, s, envelope.Fail(envelope.ExitNotFound, "not_found",
			fmt.Sprintf("no bound submission recorded for %s", subject))
	}
	if err != nil {
		return verdict.Input{}, s, envelope.Fail(envelope.ExitChainInvalid, "chain_invalid", err.Error())
	}
	return in, s, nil
}

func verdictFailEnvelope(err error) *envelope.Envelope {
	var ug *verdict.UngatedError
	if errors.As(err, &ug) {
		return envelope.Fail(envelope.ExitUngated, "ungated", err.Error())
	}
	var su *verdict.SpecUnrunnableError
	if errors.As(err, &su) {
		return envelope.Fail(envelope.ExitSpecUnrunnable, "spec_unrunnable", err.Error())
	}
	var re *verdict.RangeError
	if errors.As(err, &re) {
		// The submission's own data fails the verifier structurally:
		// the established shape-refusal mapping.
		return envelope.Fail(envelope.ExitChainInvalid, "chain_invalid", err.Error())
	}
	return envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error())
}

// storeReceipt writes the canonical receipt into the content-addressed
// artifact store and returns its digest.
func storeReceipt(r *verdict.Receipt, artifacts string) (string, error) {
	canonical, err := r.Canonical()
	if err != nil {
		return "", err
	}
	return artifact.Open(artifacts).Put(canonical)
}

func redTranscript(r *verdict.Receipt) (verdict.Transcript, bool) {
	for _, tr := range r.Transcripts {
		if tr.Exit != 0 {
			return tr, true
		}
	}
	// A red sealed check forbids pass exactly like a visible one
	// (plans/os-3128535a.md): the verdict derives from every
	// transcript the run executed.
	for _, tr := range r.SealedTranscripts {
		if tr.Exit != 0 {
			return tr, true
		}
	}
	return verdict.Transcript{}, false
}

func receiptSummary(subject string, r *verdict.Receipt, digest string) map[string]any {
	red := 0
	for _, tr := range r.Transcripts {
		if tr.Exit != 0 {
			red++
		}
	}
	for _, tr := range r.SealedTranscripts {
		if tr.Exit != 0 {
			red++
		}
	}
	out := map[string]any{
		"subject":     subject,
		"receipt":     digest,
		"merge_base":  r.MergeBase,
		"head":        r.Head,
		"files":       fmt.Sprintf("%d", len(r.Files)),
		"transcripts": fmt.Sprintf("%d", len(r.Transcripts)),
		"red":         fmt.Sprintf("%d", red),
		"runner":      r.Environment.Runner,
	}
	if r.Commitment != "" {
		out["commitment"] = r.Commitment
		out["sealed_transcripts"] = fmt.Sprintf("%d", len(r.SealedTranscripts))
	}
	return out
}

func runVerdictReceipt(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("verdict receipt", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("ledger", "", "ledger directory")
	subject := fs.String("subject", "", "contract in review")
	repo := fs.String("repo", "", "source repository the submission range names")
	artifacts := fs.String("artifacts", "", "artifact store root (default <repo>/next/var/artifacts)")
	timeout := fs.Duration("timeout", 0, "per-command wall-clock bound (default 10m)")
	if err := fs.Parse(args); err != nil || *dir == "" || *subject == "" || *repo == "" || fs.NArg() != 0 {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "verdict receipt requires --ledger <dir> --subject <id> --repo <dir> [--artifacts <dir>] [--timeout <dur>]"), stdout, stderr)
	}
	st, failEnv := loadVerdictState(*dir)
	if failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	in, _, failEnv := st.verdictInput(*subject, *repo, *timeout, true)
	if failEnv != nil {
		return render(stampTip(failEnv, st.count), stdout, stderr)
	}
	r, err := verdict.Compute(in)
	if err != nil {
		return render(stampTip(verdictFailEnvelope(err), st.count), stdout, stderr)
	}
	digest, err := storeReceipt(r, artifactsDir(*artifacts, *repo))
	if err != nil {
		return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()), stdout, stderr)
	}
	return render(stampTip(envelope.OK(receiptSummary(*subject, r, digest)), st.count), stdout, stderr)
}

func artifactsDir(flagValue, repo string) string {
	if flagValue != "" {
		return flagValue
	}
	return filepath.Join(repo, "next", "var", "artifacts")
}

func runVerdictRender(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("verdict render", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	// BOTH postures (plans/os-6a08b166.md D6.5). Rendering was
	// local-only, so a fleet's verifier lane could not act against the
	// shared ledger its workers claim on: the terminal half of the
	// contract lifecycle had no reachable surface in the deployment
	// the charter's fleet mode describes.
	f := bindLoopFlags(fs)
	repo := fs.String("repo", "", "source repository the submission range names")
	verdictFlag := fs.String("verdict", "", "pass or fail")
	artifacts := fs.String("artifacts", "", "artifact store root (default <repo>/next/var/artifacts)")
	timeout := fs.Duration("timeout", 0, "per-command wall-clock bound (default 10m)")
	parseErr := fs.Parse(args)
	missing := ""
	if *repo == "" || (*verdictFlag != "pass" && *verdictFlag != "fail") {
		missing = "and --repo <dir> --verdict pass|fail [--artifacts <dir>] [--timeout <dur>]"
	}
	if env := f.usage("verdict render", parseErr, fs.NArg(), missing); env != nil {
		return render(env, stdout, stderr)
	}
	subject := f.subject
	signer, env := loopSigner(*f.keyPath, *f.as)
	if env != nil {
		return render(env, stdout, stderr)
	}
	ls, env := openLoopSession(f)
	if env != nil {
		return render(env, stdout, stderr)
	}
	defer ls.done()
	st, failEnv := ls.verdictState()
	if failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	in, s, failEnv := st.verdictInput(*subject, *repo, *timeout, true)
	if failEnv != nil {
		return render(stampTip(failEnv, st.count), stdout, stderr)
	}
	// The red-verdict lockout at the render surface
	// (plans/os-d2497eb7.md): pass over a submission an authenticated
	// fail already judged refuses until a new submission; fail stays
	// renderable. The admission rule enforces the same bound.
	if *verdictFlag == "pass" {
		if locked := renderLocked(st, s); locked != nil {
			return render(stampTip(envelope.Fail(envelope.ExitRedLocked, "red_locked",
				fmt.Sprintf("a fail verdict at position %d already judged the bound submission — a red verdict locks pass out until a new submission (contract.returned, re-claim, resubmit; next/spec/verdicts.md)", locked.Pos)), st.count), stdout, stderr)
		}
	}
	// The "contracts carry sealed checks" gate, enforced at the
	// verifier boundary (plans/os-3128535a.md): above the trivial
	// tier a subject with no commitment does not render; the trivial
	// tier is exempt.
	if s.Sealed == nil && transition.TierGates(s.Tier).SealedChecksRequired {
		return render(stampTip(envelope.Fail(envelope.ExitUnsealed, "unsealed",
			fmt.Sprintf("contract %s (tier %q) carries no sealed-checks commitment — above the trivial tier contracts carry sealed checks, sealed before the first claim (next/spec/sealed-checks.md)", *subject, s.Tier)), st.count), stdout, stderr)
	}
	sealedIn, sealFail := unsealChecks(st.records, s, signer, artifact.Open(artifactsDir(*artifacts, *repo)))
	if sealFail != nil {
		return render(stampTip(sealFail, st.count), stdout, stderr)
	}
	in.Sealed = sealedIn
	// The bound eval's carrier (plans/os-96850e5a.md D5): a
	// counter-trajectory judged without the candidate applied proves
	// nothing about it, so a subject whose marker names a carrier
	// renders only when the carrier commit is an ancestor of the
	// submission head.
	if s.Eval != nil && s.Eval.Carrier != "" {
		_, carrierCommit, _ := curation.AnchorParts(s.Eval.Carrier)
		_, head, _ := strings.Cut(in.Base, "..")
		if !gitIsAncestor(*repo, carrierCommit, head) {
			return render(stampTip(envelope.Fail(envelope.ExitChecksRed, "carrier_absent",
				fmt.Sprintf("the eval is bound to carrier %s and the submission head %s does not descend from it — the lesson was never applied, so its survival cannot be judged (next/spec/evals.md)", s.Eval.Carrier, head)), st.count), stdout, stderr)
		}
	}
	r, err := verdict.Compute(in)
	if err != nil {
		return render(stampTip(verdictFailEnvelope(err), st.count), stdout, stderr)
	}
	// Render derives the permissible verdict from the transcripts it
	// just executed: pass over any red check refuses, naming the
	// command; fail is always renderable; prose-only pass stays the
	// verifier's explicit judgment (next/spec/verdicts.md).
	if *verdictFlag == "pass" {
		if tr, red := redTranscript(r); red {
			return render(stampTip(envelope.Fail(envelope.ExitChecksRed, "checks_red",
				fmt.Sprintf("rendering pass refused: %q exited %d — the verdict derives from the transcripts, and a red check forbids pass (fail stays renderable)", tr.Cmd, tr.Exit)), st.count), stdout, stderr)
		}
	}
	digest, err := storeReceipt(r, artifactsDir(*artifacts, *repo))
	if err != nil {
		return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()), stdout, stderr)
	}
	// The submission citation is DERIVED, and on the remote path the
	// tip can move between drafting and landing. `recheckDerivation`
	// re-derives against each refreshed view and REFUSES on a change
	// rather than re-pointing: a verdict bound to whatever submission
	// happens to be current is the laundering shape, and binding is
	// the whole point of the field (plans/os-9b3f3ef3.md D1).
	//
	// The receipt is NOT re-derived. It is content-derived from the
	// repository, not from the ledger view, so a moving tip cannot
	// change it and re-running acceptance per attempt would buy
	// nothing.
	derive := func(ctx *admit.Context) ([]byte, *envelope.Envelope) {
		return verdictPayload(ctx, *subject, *verdictFlag, digest)
	}
	payload, env := derive(ls.ctx)
	if env != nil {
		return render(stampTip(env, st.count), stdout, stderr)
	}
	summary := receiptSummary(*subject, r, digest)
	return ls.commit(f, loopAct{verb: transition.VerdictRenderedVerb, payload: payload, derive: derive,
		resultAt: func(int) map[string]any {
			out := map[string]any{}
			for k, v := range summary {
				out[k] = v
			}
			out["verdict"] = *verdictFlag
			out["submission"] = strconv.Itoa(s.Submission.Pos)
			return out
		}}, signer, stdout, stderr)
}

// verdictPayload binds a rendered verdict to the submission that
// produced the review state, read from the view being judged against.
func verdictPayload(ctx *admit.Context, subject, v, receipt string) ([]byte, *envelope.Envelope) {
	if ctx == nil || ctx.Lifecycle == nil {
		return nil, envelope.Fail(envelope.ExitUnavailable, "unavailable", "no lifecycle view to bind the verdict to")
	}
	s, ok := ctx.Lifecycle.State(subject)
	if !ok || s.Submission == nil {
		return nil, envelope.Fail(envelope.ExitNotFound, "not_found",
			fmt.Sprintf("no submission stands on %s — a verdict judges exactly the submission that produced the review state", subject))
	}
	b, err := json.Marshal(map[string]string{
		"verdict":      v,
		"receipt":      receipt,
		"submission":   strconv.Itoa(s.Submission.Pos),
		"independence": "L1",
	})
	if err != nil {
		return nil, envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error())
	}
	return b, nil
}

// verdictState builds the read model from whichever posture the
// session opened, so the view a verdict is computed against is the
// same one its act is judged against.
func (ls *loopSession) verdictState() (*verdictState, *envelope.Envelope) {
	if ls.remote != nil {
		return verdictStateAt(ls.remote.store, ls.remote.resolve, ls.remote.vopts...)
	}
	resolve, _, err := genesis.Bootstrap(ls.store)
	if err != nil {
		return nil, envelope.Fail(envelope.ExitChainInvalid, "chain_invalid", err.Error())
	}
	return verdictStateAt(ls.store, resolve)
}

func runVerdictCheck(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("verdict check", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("ledger", "", "ledger directory")
	subject := fs.String("subject", "", "contract with a rendered verdict")
	repo := fs.String("repo", "", "source repository the submission range names")
	keyPath := fs.String("key", "", "OpenSSH ed25519 private key able to unseal (required for a sealed subject)")
	artifacts := fs.String("artifacts", "", "artifact store root (default <repo>/next/var/artifacts)")
	timeout := fs.Duration("timeout", 0, "per-command wall-clock bound (default 10m)")
	if err := fs.Parse(args); err != nil || *dir == "" || *subject == "" || *repo == "" || fs.NArg() != 0 {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "verdict check requires --ledger <dir> --subject <id> --repo <dir> [--key <path>] [--artifacts <dir>] [--timeout <dur>]"), stdout, stderr)
	}
	st, failEnv := loadVerdictState(*dir)
	if failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	// The latest rendered verdict on the subject is the one checked.
	var cited struct {
		Verdict    string `json:"verdict"`
		Receipt    string `json:"receipt"`
		Submission string `json:"submission"`
	}
	found := false
	for i := len(st.records) - 1; i >= 0; i-- {
		e := &st.records[i].Event
		if e.Verb == transition.VerdictRenderedVerb && e.Subject == *subject {
			if err := json.Unmarshal(e.Payload, &cited); err == nil {
				found = true
			}
			break
		}
	}
	if !found {
		return render(stampTip(envelope.Fail(envelope.ExitNotFound, "not_found",
			fmt.Sprintf("no rendered verdict on %s", *subject)), st.count), stdout, stderr)
	}
	// The stored receipt is retrievable evidence: a check is not green
	// over a store that lost or corrupted the cited artifact, however
	// clean the recomputation (review finding on the task PR).
	// artifact.Get digest-verifies content on the way out.
	if _, err := artifact.Open(artifactsDir(*artifacts, *repo)).Get(strings.TrimSpace(cited.Receipt)); err != nil {
		return render(stampTip(envelope.Fail(envelope.ExitReceiptMismatch, "receipt_mismatch",
			fmt.Sprintf("the cited receipt %s is not retrievable intact from the artifact store: %v — the evidence a verdict points at must survive verbatim (6.2 reconciliation input)", cited.Receipt, err)), st.count), stdout, stderr)
	}
	in, s, failEnv := st.verdictInput(*subject, *repo, *timeout, false)
	if failEnv != nil {
		return render(stampTip(failEnv, st.count), stdout, stderr)
	}
	// A sealed subject holds the full recompute-and-mismatch
	// guarantee (review finding on the 6.3 plan): the check decrypts
	// and reruns the sealed commands, so a raw-pushed receipt with
	// invented passing sealed transcripts recomputes differently and
	// fails below. There is no silent partial verification: without a
	// capable key, the check refuses.
	if s.Sealed != nil {
		if *keyPath == "" {
			return render(stampTip(envelope.Fail(envelope.ExitUsage, "usage",
				fmt.Sprintf("contract %s carries sealed checks — verdict check needs --key with an identity able to unseal, or the sealed transcripts cannot be recomputed", *subject)), st.count), stdout, stderr)
		}
		keyBytes, err := os.ReadFile(*keyPath)
		if err != nil {
			return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("cannot read --key: %v", err)), stdout, stderr)
		}
		identity, err := event.ParsePrivateKey(keyBytes)
		if err != nil {
			return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("--key: %v", err)), stdout, stderr)
		}
		sealedIn, sealFail := unsealChecks(st.records, s, identity, artifact.Open(artifactsDir(*artifacts, *repo)))
		if sealFail != nil {
			return render(stampTip(sealFail, st.count), stdout, stderr)
		}
		in.Sealed = sealedIn
	}
	r, err := verdict.Compute(in)
	if err != nil {
		return render(stampTip(verdictFailEnvelope(err), st.count), stdout, stderr)
	}
	digest, err := r.Digest()
	if err != nil {
		return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()), stdout, stderr)
	}
	if digest != strings.TrimSpace(cited.Receipt) {
		return render(stampTip(envelope.Fail(envelope.ExitReceiptMismatch, "receipt_mismatch",
			fmt.Sprintf("recomputation from the submission head yields %s, the rendered verdict cites %s — the receipt, the range, or the repository's content has diverged from what was attested (6.2 reconciliation input)", digest, cited.Receipt)), st.count), stdout, stderr)
	}
	return render(stampTip(envelope.OK(map[string]any{
		"subject":    *subject,
		"receipt":    digest,
		"verdict":    cited.Verdict,
		"submission": cited.Submission,
		"artifact":   "verified",
	}), st.count), stdout, stderr)
}

// renderLocked finds the authenticated fail that locks pass out of the
// current submission window, mirroring the admission rule's bound: the
// signer holds the verdict grant at the tip keyring and is no
// implementing key, so a raw-pushed fail locks nothing
// (plans/os-d2497eb7.md).
func renderLocked(st *verdictState, s transition.SubjectState) *transition.VerdictFact {
	ks, _, err := keyring.StateAt(st.records)
	if err != nil || ks == nil {
		return nil
	}
	for i := range s.SubmissionFails {
		f := &s.SubmissionFails[i]
		if !ks.HasAnyCapability(f.Signer, keyring.AcceptedCapabilities(transition.VerdictRenderedVerb)) {
			continue
		}
		if s.PriorClaimants[f.Signer] || (s.Submission != nil && f.Signer == s.Submission.Signer) {
			continue
		}
		return f
	}
	return nil
}

// gitIsAncestor reports whether ancestor reaches descendant in the
// repository.
func gitIsAncestor(repo, ancestor, descendant string) bool {
	if ancestor == "" || descendant == "" {
		return false
	}
	return exec.Command("git", "-C", repo, "merge-base", "--is-ancestor", ancestor, descendant).Run() == nil
}
