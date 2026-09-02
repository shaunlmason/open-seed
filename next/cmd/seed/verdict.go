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
	"crypto/ed25519"
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

	"github.com/shaunlmason/open-seed/next/executor"
	"github.com/shaunlmason/open-seed/next/internal/admit"
	"github.com/shaunlmason/open-seed/next/internal/artifact"
	"github.com/shaunlmason/open-seed/next/internal/curation"
	"github.com/shaunlmason/open-seed/next/internal/envelope"
	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/genesis"
	"github.com/shaunlmason/open-seed/next/internal/keyring"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
	"github.com/shaunlmason/open-seed/next/internal/plan"
	"github.com/shaunlmason/open-seed/next/internal/reconcile"
	"github.com/shaunlmason/open-seed/next/internal/transition"
	"github.com/shaunlmason/open-seed/next/internal/tuple"
	"github.com/shaunlmason/open-seed/next/internal/verdict"
	"github.com/shaunlmason/open-seed/next/internal/version"
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
	case "defer":
		return runVerdictDefer(args[1:], stdout, stderr)
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
	// The verifier's declared tuple (plans/os-99829835.md D2): the
	// three fields the workspace cannot know, the run start posture;
	// harness and environment come from the adapter the verifier
	// works in.
	principal := fs.String("principal", "", "the principal the verifier ran as: a judgment the workspace cannot make")
	model := fs.String("model", "", "the verifier's model as <family>/<version> or <provider>/<family>/<version>")
	toolPolicy := fs.String("tool-policy", "", "the verifier's tool policy profile")
	scorecardPath := fs.String("scorecard", "", "the verifier's item-by-item scoring of the spec's rubric (required when the spec carries one)")
	parseErr := fs.Parse(args)
	missing := ""
	if *repo == "" || (*verdictFlag != "pass" && *verdictFlag != "fail") {
		missing = "and --repo <dir> --verdict pass|fail [--artifacts <dir>] [--timeout <dur>] [--principal <p> --model <m> --tool-policy <t>] [--scorecard <file>]"
	}
	declaring := *principal != "" || *model != "" || *toolPolicy != ""
	if declaring && (*principal == "" || *model == "" || *toolPolicy == "") {
		missing = "and --principal --model --tool-policy together: the verifier's declaration is the whole tuple or none"
	}
	if env := f.usage("verdict render", parseErr, fs.NArg(), missing); env != nil {
		return render(env, stdout, stderr)
	}
	var declared *tuple.Tuple
	if declaring {
		declared = &tuple.Tuple{Principal: *principal, Harness: executor.LocalHarness, Model: *model,
			ToolPolicy: *toolPolicy, Environment: executor.LocalEnvironment}
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
	// The achieved level, computed from the same facts the boundary
	// reads before anything is drafted (plans/os-99829835.md D3): a
	// tier the record cannot satisfy refuses here, naming the flags
	// when no declaration was made, and the client never drafts a
	// doomed verdict. Before seed/4 the level is the literal L1 and a
	// declaration has nowhere to go.
	if declared != nil && !version.LevelsApply(ls.ctx.Active) {
		return render(stampTip(envelope.Fail(envelope.ExitUsage, "usage",
			fmt.Sprintf("the chain is at %s: the verifier's declaration (--principal --model --tool-policy) activates at %s (next/spec/protocol.md)", ls.ctx.Active, version.Seed4)), st.count), stdout, stderr)
	}
	if version.LevelsApply(ls.ctx.Active) {
		required := transition.TierGates(s.Tier).Independence
		achieved := admit.LevelAchieved(ls.ctx.Records, ls.ctx.Table, *subject, s, declared)
		if !achieved.Satisfies(required) {
			if declared == nil {
				return render(stampTip(envelope.Fail(envelope.ExitUsage, "usage",
					fmt.Sprintf("contract %s (tier %q) requires independence %s and the record supports %s with no declaration: declare the verifier's tuple with --principal --model --tool-policy, a different model family, provider or harness from the window's (next/spec/verdicts.md)", *subject, s.Tier, required, achieved)), st.count), stdout, stderr)
			}
			return render(stampTip(envelope.Fail(envelope.ExitNotIndependent, "level_short",
				(&admit.LevelShortError{Subject: *subject, Tier: s.Tier, Required: required, Achieved: achieved}).Error()), st.count), stdout, stderr)
		}
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
	// The rubric (plans/os-2e34f66a.md D1): the spec's residue, read
	// at its anchor exactly as the commands are; a spec with one
	// renders only over a scorecard.
	rubric, failEnv := rubricAt(*repo, s, st.count)
	if failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	if len(rubric) > 0 && *scorecardPath == "" {
		return render(stampTip(envelope.Fail(envelope.ExitUsage, "usage",
			fmt.Sprintf("the acceptance spec carries a rubric of %d items — render needs --scorecard <file>, the verifier's item-by-item scoring with cited evidence and explicit uncertainty (next/spec/verdicts.md)", len(rubric))), st.count), stdout, stderr)
	}
	if len(rubric) == 0 && *scorecardPath != "" {
		return render(stampTip(envelope.Fail(envelope.ExitUsage, "usage", "the acceptance spec carries no rubric, so there is nothing a scorecard scores"), st.count), stdout, stderr)
	}
	// The human verdict (D4): on a human-review tier, or after a
	// deferral on this window, the render is a human's; a key with
	// no operator standing refuses here rather than drafting a
	// verdict admission would refuse.
	fp, _ := event.Fingerprint(signer.Public().(ed25519.PublicKey))
	if version.LevelsApply(ls.ctx.Active) {
		if human, why := admit.HumanVerdictRequired(s, ls.ctx.Count); human && !admit.OperatorStanding(ls.ctx.Keyring, fp) {
			return render(stampTip(envelope.Fail(envelope.ExitChecksRed, transition.CodeHumanVerdict,
				fmt.Sprintf("%s — the render is a human's, a key with operator standing beside its verdict grant, and %s holds no operator standing (next/spec/verdicts.md)", why, fp)), st.count), stdout, stderr)
		}
	}
	// The human renders over the deferral's receipt (D4): a key with
	// operator standing is never a sealed-checks recipient, so the
	// receipt is the one the deferring verifier computed and cited,
	// retrieved intact from the store rather than recomputed; the
	// machine verifier computes its own.
	var r *verdict.Receipt
	digest := ""
	if s.Deferred != nil && version.LevelsApply(ls.ctx.Active) && admit.OperatorStanding(ls.ctx.Keyring, fp) {
		body, err := artifact.Open(artifactsDir(*artifacts, *repo)).Get(s.Deferred.Receipt)
		if err != nil {
			return render(stampTip(envelope.Fail(envelope.ExitReceiptMismatch, "receipt_mismatch",
				fmt.Sprintf("the deferral's receipt %s is not retrievable intact from the artifact store: %v — the human renders over the receipt the verifier computed, and it must survive verbatim", s.Deferred.Receipt, err)), st.count), stdout, stderr)
		}
		var loaded verdict.Receipt
		if err := json.Unmarshal(body, &loaded); err != nil {
			return render(stampTip(envelope.Fail(envelope.ExitReceiptMismatch, "receipt_mismatch",
				fmt.Sprintf("the deferral's receipt %s does not parse: %v", s.Deferred.Receipt, err)), st.count), stdout, stderr)
		}
		r, digest = &loaded, s.Deferred.Receipt
	} else {
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
		computed, err := verdict.Compute(in)
		if err != nil {
			return render(stampTip(verdictFailEnvelope(err), st.count), stdout, stderr)
		}
		r = computed
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
	// The scorecard (D2, D3): validated against the rubric and the
	// receipt, stored, and the verdict derived from its items exactly
	// as from the transcripts.
	var scoreRef *transition.ScorecardRef
	if *scorecardPath != "" {
		sc, failEnv := loadScorecard(*scorecardPath, *subject, s, rubric, r, *repo, st.count)
		if failEnv != nil {
			return render(failEnv, stdout, stderr)
		}
		ref, err := sc.Ref()
		if err != nil {
			return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()), stdout, stderr)
		}
		derived, code, item := transition.DeriveScores(ref.Items)
		switch {
		case code == transition.CodeHumanVerdict:
			return render(stampTip(envelope.Fail(envelope.ExitChecksRed, transition.CodeHumanVerdict,
				fmt.Sprintf("rendering refused: item %q is scored at high uncertainty — low confidence routes to a human verdict (seed verdict defer), and neither pass nor fail is renderable over it", item)), st.count), stdout, stderr)
		case *verdictFlag == "pass" && derived == "fail":
			return render(stampTip(envelope.Fail(envelope.ExitChecksRed, transition.CodeRubricRed,
				fmt.Sprintf("rendering pass refused: item %q scores fail — the verdict derives from the scorecard, and a failing item forbids pass (fail stays renderable)", item)), st.count), stdout, stderr)
		}
		if _, err := storeScorecard(sc, artifactsDir(*artifacts, *repo)); err != nil {
			return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()), stdout, stderr)
		}
		scoreRef = &ref
	}
	if digest == "" {
		stored, err := storeReceipt(r, artifactsDir(*artifacts, *repo))
		if err != nil {
			return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()), stdout, stderr)
		}
		digest = stored
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
		return verdictPayload(ctx, *subject, *verdictFlag, digest, declared, scoreRef)
	}
	payload, env := derive(ls.ctx)
	if env != nil {
		return render(stampTip(env, st.count), stdout, stderr)
	}
	summary := receiptSummary(*subject, r, digest)
	level := verdictLevel(ls.ctx, *subject, declared)
	return ls.commit(f, loopAct{verb: transition.VerdictRenderedVerb, payload: payload, derive: derive,
		resultAt: func(int) map[string]any {
			out := map[string]any{}
			for k, v := range summary {
				out[k] = v
			}
			out["verdict"] = *verdictFlag
			out["submission"] = strconv.Itoa(s.Submission.Pos)
			out["independence"] = level
			if scoreRef != nil {
				out["scorecard"] = scoreRef.Digest
				out["items"] = scoreRef.Items
			}
			return out
		}}, signer, stdout, stderr)
}

// rubricAt reads the rubric of the subject's acceptance spec at its
// anchor from the repository (plans/os-2e34f66a.md D1), the commands'
// own read; a rubric the parser refuses is a spec that cannot decide.
func rubricAt(repo string, s transition.SubjectState, count int) ([]plan.Item, *envelope.Envelope) {
	if s.Acceptance == nil {
		return nil, nil
	}
	path, commit, ok := curation.AnchorParts(s.Acceptance.Ref)
	if !ok {
		return nil, nil
	}
	body, err := exec.Command("git", "-C", repo, "show", commit+":"+path).Output()
	if err != nil {
		return nil, stampTip(envelope.Fail(envelope.ExitChainInvalid, "chain_invalid",
			fmt.Sprintf("acceptance spec %s does not resolve at its anchored commit in %s", s.Acceptance.Ref, repo)), count)
	}
	items, err := plan.Rubric(body)
	if err != nil {
		return nil, stampTip(envelope.Fail(envelope.ExitSpecUnrunnable, "spec_unrunnable",
			fmt.Sprintf("acceptance spec %s: %v — a rubric that cannot be scored item by item cannot decide", s.Acceptance.Ref, err)), count)
	}
	return items, nil
}

// loadScorecard reads and validates the verifier's scorecard against
// the rubric, the receipt and the repository, naming the part that
// refuses.
func loadScorecard(path, subject string, s transition.SubjectState, rubric []plan.Item, r *verdict.Receipt, repo string, count int) (*verdict.Scorecard, *envelope.Envelope) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("cannot read --scorecard: %v", err))
	}
	sc, err := verdict.ParseScorecard(raw)
	if err != nil {
		return nil, stampTip(envelope.Fail(envelope.ExitUsage, "usage", err.Error()), count)
	}
	if s.Submission == nil {
		return nil, stampTip(envelope.Fail(envelope.ExitNotFound, "not_found", fmt.Sprintf("no submission stands on %s", subject)), count)
	}
	if err := verdict.Validate(sc, subject, s.Submission.Pos, rubric, r, repo); err != nil {
		return nil, stampTip(envelope.Fail(envelope.ExitUsage, "usage", err.Error()), count)
	}
	return sc, nil
}

// storeScorecard writes the canonical scorecard into the artifact
// store and returns its digest, the citation the verdict carries.
func storeScorecard(sc *verdict.Scorecard, artifacts string) (string, error) {
	canonical, err := sc.Canonical()
	if err != nil {
		return "", err
	}
	return artifact.Open(artifacts).Put(canonical)
}

// runVerdictDefer is the human-verdict deferral (plans/os-2e34f66a.md
// D4): the verifier's scorecard, validated and stored as at render,
// with at least one item at high uncertainty, and verdict.deferred
// appended naming those items. The subject stays in review for the
// human's render.
func runVerdictDefer(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("verdict defer", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	f := bindLoopFlags(fs)
	repo := fs.String("repo", "", "source repository the submission range names")
	artifacts := fs.String("artifacts", "", "artifact store root (default <repo>/next/var/artifacts)")
	timeout := fs.Duration("timeout", 0, "per-command wall-clock bound (default 10m)")
	scorecardPath := fs.String("scorecard", "", "the verifier's scoring, with the items it could not judge at high uncertainty (required when the spec carries a rubric)")
	parseErr := fs.Parse(args)
	missing := ""
	if *repo == "" {
		missing = "and --repo <dir> [--scorecard <file>] [--artifacts <dir>] [--timeout <dur>]"
	}
	if env := f.usage("verdict defer", parseErr, fs.NArg(), missing); env != nil {
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
	if !version.LevelsApply(ls.ctx.Active) {
		return render(stampTip(envelope.Fail(envelope.ExitUsage, "usage",
			fmt.Sprintf("the chain is at %s: the human-verdict deferral activates at %s (next/spec/verdicts.md)", ls.ctx.Active, version.Seed4)), ls.ctx.Count), stdout, stderr)
	}
	st, failEnv := ls.verdictState()
	if failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	in, s, failEnv := st.verdictInput(*subject, *repo, *timeout, true)
	if failEnv != nil {
		return render(stampTip(failEnv, st.count), stdout, stderr)
	}
	rubric, failEnv := rubricAt(*repo, s, st.count)
	if failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	humanTier := transition.TierGates(s.Tier).HumanReview
	if len(rubric) > 0 && *scorecardPath == "" {
		return render(stampTip(envelope.Fail(envelope.ExitUsage, "usage",
			fmt.Sprintf("the acceptance spec carries a rubric of %d items — defer needs --scorecard <file>, the items it could not judge scored at high uncertainty", len(rubric))), st.count), stdout, stderr)
	}
	if len(rubric) == 0 && *scorecardPath != "" {
		return render(stampTip(envelope.Fail(envelope.ExitUsage, "usage", "the acceptance spec carries no rubric, so there is nothing a scorecard scores"), st.count), stdout, stderr)
	}
	if len(rubric) == 0 && !humanTier {
		return render(stampTip(envelope.Fail(envelope.ExitUsage, "usage",
			fmt.Sprintf("nothing to defer: the acceptance spec carries no rubric and tier %q does not route every verdict to a human, so render instead", s.Tier)), st.count), stdout, stderr)
	}
	sealedIn, sealFail := unsealChecks(st.records, s, signer, artifact.Open(artifactsDir(*artifacts, *repo)))
	if sealFail != nil {
		return render(stampTip(sealFail, st.count), stdout, stderr)
	}
	in.Sealed = sealedIn
	r, err := verdict.Compute(in)
	if err != nil {
		return render(stampTip(verdictFailEnvelope(err), st.count), stdout, stderr)
	}
	receipt, err := storeReceipt(r, artifactsDir(*artifacts, *repo))
	if err != nil {
		return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()), stdout, stderr)
	}
	scorecard := ""
	var deferred []string
	if *scorecardPath != "" {
		sc, failEnv := loadScorecard(*scorecardPath, *subject, s, rubric, r, *repo, st.count)
		if failEnv != nil {
			return render(failEnv, stdout, stderr)
		}
		for _, it := range sc.Items {
			if it.Uncertainty == "high" {
				deferred = append(deferred, it.ID)
			}
		}
		if len(deferred) == 0 && !humanTier {
			return render(stampTip(envelope.Fail(envelope.ExitUsage, "usage", "every item is scored at low uncertainty: nothing to defer, render instead"), st.count), stdout, stderr)
		}
		if scorecard, err = storeScorecard(sc, artifactsDir(*artifacts, *repo)); err != nil {
			return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()), stdout, stderr)
		}
	}
	derive := func(ctx *admit.Context) ([]byte, *envelope.Envelope) {
		if ctx == nil || ctx.Lifecycle == nil {
			return nil, envelope.Fail(envelope.ExitUnavailable, "unavailable", "no lifecycle view to bind the deferral to")
		}
		cur, ok := ctx.Lifecycle.State(*subject)
		if !ok || cur.Submission == nil {
			return nil, envelope.Fail(envelope.ExitNotFound, "not_found", fmt.Sprintf("no submission stands on %s", *subject))
		}
		fields := map[string]any{"receipt": receipt, "submission": strconv.Itoa(cur.Submission.Pos)}
		if scorecard != "" {
			fields["scorecard"] = scorecard
		}
		if len(deferred) > 0 {
			fields["items"] = deferred
		}
		b, err := json.Marshal(fields)
		if err != nil {
			return nil, envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error())
		}
		return b, nil
	}
	payload, env := derive(ls.ctx)
	if env != nil {
		return render(stampTip(env, st.count), stdout, stderr)
	}
	summary := receiptSummary(*subject, r, receipt)
	return ls.commit(f, loopAct{verb: transition.VerdictDeferredVerb, payload: payload, derive: derive,
		resultAt: func(int) map[string]any {
			out := map[string]any{}
			for k, v := range summary {
				out[k] = v
			}
			out["submission"] = strconv.Itoa(s.Submission.Pos)
			out["owed_by"] = "lane:operator"
			if scorecard != "" {
				out["scorecard"] = scorecard
			}
			if deferred != nil {
				out["items"] = deferred
			}
			return out
		}}, signer, stdout, stderr)
}

// verdictLevel is the level a render writes at a view: the literal L1
// before seed/4, the achieved level (plans/os-99829835.md D1) from it.
func verdictLevel(ctx *admit.Context, subject string, declared *tuple.Tuple) string {
	if ctx == nil || ctx.Lifecycle == nil || !version.LevelsApply(ctx.Active) {
		return string(transition.L1)
	}
	s, ok := ctx.Lifecycle.State(subject)
	if !ok {
		return string(transition.L1)
	}
	return string(admit.LevelAchieved(ctx.Records, ctx.Table, subject, s, declared))
}

// verdictPayload binds a rendered verdict to the submission that
// produced the review state, read from the view being judged against,
// and records the level that view supports with the declaration it was
// computed from (plans/os-99829835.md D2, D3).
func verdictPayload(ctx *admit.Context, subject, v, receipt string, declared *tuple.Tuple, scorecard *transition.ScorecardRef) ([]byte, *envelope.Envelope) {
	if ctx == nil || ctx.Lifecycle == nil {
		return nil, envelope.Fail(envelope.ExitUnavailable, "unavailable", "no lifecycle view to bind the verdict to")
	}
	s, ok := ctx.Lifecycle.State(subject)
	if !ok || s.Submission == nil {
		return nil, envelope.Fail(envelope.ExitNotFound, "not_found",
			fmt.Sprintf("no submission stands on %s — a verdict judges exactly the submission that produced the review state", subject))
	}
	fields := map[string]any{
		"verdict":      v,
		"receipt":      receipt,
		"submission":   strconv.Itoa(s.Submission.Pos),
		"independence": verdictLevel(ctx, subject, declared),
	}
	if declared != nil && version.LevelsApply(ctx.Active) {
		fields["tuple"] = declared
	}
	if scorecard != nil && version.LevelsApply(ctx.Active) {
		fields["scorecard"] = scorecard
	}
	b, err := json.Marshal(fields)
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
		Verdict      string                   `json:"verdict"`
		Receipt      string                   `json:"receipt"`
		Submission   string                   `json:"submission"`
		Independence string                   `json:"independence"`
		Scorecard    *transition.ScorecardRef `json:"scorecard"`
	}
	found := false
	citedPos := -1
	for i := len(st.records) - 1; i >= 0; i-- {
		e := &st.records[i].Event
		if e.Verb == transition.VerdictRenderedVerb && e.Subject == *subject {
			if err := json.Unmarshal(e.Payload, &cited); err == nil {
				found = true
				citedPos = i
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
	// A rubric verdict's other artifact (plans/os-2e34f66a.md D3;
	// review finding on the task PR): the cited scorecard retrieves
	// intact and its items are the payload's, the same check
	// reconcile classifies as scorecard_unverified, so a check is not
	// green over a store that lost or altered what the verifier
	// scored.
	if cited.Scorecard != nil {
		if f := reconcile.ScorecardAt(*subject, &transition.VerdictFact{Pos: citedPos, Scorecard: cited.Scorecard}, artifact.Open(artifactsDir(*artifacts, *repo))); f != nil {
			return render(stampTip(envelope.Fail(envelope.ExitReceiptMismatch, "receipt_mismatch", f.Detail), st.count), stdout, stderr)
		}
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
	result := map[string]any{
		"subject":      *subject,
		"receipt":      digest,
		"verdict":      cited.Verdict,
		"submission":   cited.Submission,
		"independence": cited.Independence,
		"artifact":     "verified",
	}
	if cited.Scorecard != nil {
		result["scorecard"] = "verified"
	}
	return render(stampTip(envelope.OK(result), st.count), stdout, stderr)
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
