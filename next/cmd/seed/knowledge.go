// The knowledge verbs (plans/os-f30ee0d3.md D4; next/spec/curation.md):
// the three curation facts driven against a real ledger with the fence
// and the hypothesis id derived, and the fold rendered. No loop act:
// the worker loop's exits already carry findings, and a standalone dead
// end is the holder's deliberate extra, like plan propose.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/shaunlmason/open-seed/next/internal/admit"
	"github.com/shaunlmason/open-seed/next/internal/curation"
	"github.com/shaunlmason/open-seed/next/internal/envelope"
	"github.com/shaunlmason/open-seed/next/internal/project"
)

func runKnowledge(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "knowledge requires a subverb: deadend | propose | promote | show"), stdout, stderr)
	}
	switch args[0] {
	case "deadend":
		return runKnowledgeDeadEnd(args[1:], stdout, stderr)
	case "propose":
		return runKnowledgePropose(args[1:], stdout, stderr)
	case "promote":
		return runKnowledgePromote(args[1:], stdout, stderr)
	case "show":
		return runKnowledgeShow(args[1:], stdout, stderr)
	}
	return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("unknown knowledge subverb %q: deadend | propose | promote | show", args[0])), stdout, stderr)
}

// listFlag collects a repeatable string flag.
type listFlag []string

func (l *listFlag) String() string     { return strings.Join(*l, ",") }
func (l *listFlag) Set(v string) error { *l = append(*l, v); return nil }

// knowledgeUsage is the transport half's refusal for the verbs whose
// subject is derived rather than given.
func knowledgeUsage(name string, f *loopFlags, parseErr error, narg int, extra string) *envelope.Envelope {
	if parseErr == nil && (*f.dir == "") != (*f.remote == "") && *f.keyPath != "" && *f.subject == "" && narg == 0 && extra == "" {
		return nil
	}
	msg := fmt.Sprintf("%s requires --ledger or --remote (not both) and --key <path>; the subject is derived, never given", name)
	if extra != "" {
		msg += ", " + extra
	}
	return envelope.Fail(envelope.ExitUsage, "usage", msg)
}

func runKnowledgeDeadEnd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("knowledge deadend", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	f := bindLoopFlags(fs)
	tried := fs.String("tried", "", "the approach that failed")
	outcome := fs.String("outcome", "", "why it failed")
	condition := fs.String("condition", "", "the failure condition: what was true when it failed")
	environment := fs.String("environment", "", "the environment it failed in")
	pointer := fs.String("pointer", "", "an anchored artifact (\"<path> @ <commit>\"), optional")
	err := fs.Parse(args)
	extra := ""
	for _, req := range []struct{ name, v string }{{"--tried", *tried}, {"--outcome", *outcome}, {"--condition", *condition}, {"--environment", *environment}} {
		if err == nil && strings.TrimSpace(req.v) == "" {
			extra = req.name + " <text>"
			break
		}
	}
	if failEnv := f.usage("knowledge deadend", err, fs.NArg(), extra); failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	signer, failEnv := loopSigner(*f.keyPath, *f.as)
	if failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	ls, failEnv := openLoopSession(f)
	if failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	defer ls.done()
	subject := *f.subject
	derive := func(ctx *admit.Context) ([]byte, *envelope.Envelope) {
		fence, ok := activeFence(ctx, subject)
		if !ok {
			return nil, envelope.Fail(envelope.ExitInvalidTransition, "invalid_transition",
				fmt.Sprintf("no claim window is open on %s — a dead end is recorded by the holder inside its window (next/spec/curation.md)", subject))
		}
		d := curation.DeadEnd{Fence: fence, Tried: *tried, Outcome: *outcome, Condition: *condition, Environment: *environment, Pointer: *pointer}
		if _, perr := curation.ParseDeadEnd(subject, mustJSON(d)); perr != nil {
			return nil, envelope.Fail(envelope.ExitUsage, "usage", perr.Error())
		}
		return mustJSON(d), nil
	}
	payload, failEnv := derive(ls.ctx)
	if failEnv != nil {
		return render(stampTip(failEnv, ls.ctx.Count), stdout, stderr)
	}
	return ls.commit(f, loopAct{verb: curation.DeadEndVerb, payload: payload, derive: derive, resultAt: terse(subject)}, signer, stdout, stderr)
}

func runKnowledgePropose(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("knowledge propose", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	f := bindLoopFlags(fs)
	claim := fs.String("claim", "", "the hypothesis, as a claim")
	applies := fs.String("applies-when", "", "the condition the claim holds under")
	var support, provenance, exceptions listFlag
	fs.Var(&support, "support", "an admitted observation, <contract>@<position> (repeatable; at least two on two contracts)")
	fs.Var(&provenance, "provenance", "an anchored artifact, \"<path> @ <commit>\" (repeatable)")
	fs.Var(&exceptions, "exception", "a known exception to the claim (repeatable)")
	err := fs.Parse(args)
	extra := ""
	switch {
	case err != nil:
	case strings.TrimSpace(*claim) == "":
		extra = "--claim <text>"
	case strings.TrimSpace(*applies) == "":
		extra = "--applies-when <text>"
	case len(support) < curation.SupportMinimum:
		extra = fmt.Sprintf("at least %d --support citations on %d distinct contracts (a single run is never promotable)", curation.SupportMinimum, curation.SupportMinimum)
	}
	if failEnv := knowledgeUsage("knowledge propose", f, err, fs.NArg(), extra); failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	if exceptions == nil {
		exceptions = listFlag{}
	}
	if provenance == nil {
		provenance = listFlag{}
	}
	subject := curation.HypothesisID(*claim)
	h := curation.Hypothesis{Claim: *claim, AppliesWhen: *applies, Support: support, Exceptions: exceptions, Provenance: provenance}
	payload := mustJSON(h)
	if _, perr := curation.ParseHypothesis(subject, payload); perr != nil {
		return render(envelope.Fail(envelope.ExitUsage, "usage", perr.Error()), stdout, stderr)
	}
	signer, failEnv := loopSigner(*f.keyPath, *f.as)
	if failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	ls, failEnv := openLoopSession(f)
	if failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	defer ls.done()
	*f.subject = subject
	return ls.commit(f, loopAct{verb: curation.HypothesisVerb, payload: payload,
		resultAt: func(int) map[string]any { return map[string]any{"subject": subject, "hypothesis": subject} }}, signer, stdout, stderr)
}

func runKnowledgePromote(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("knowledge promote", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	f := bindLoopFlags(fs)
	lesson := fs.String("lesson", "", "the merged lesson file, \"next/knowledge/lessons/<id>.md @ <commit>\"")
	hypothesis := fs.String("hypothesis", "", "the admitted hypothesis, <h-id>@<position>")
	pr := fs.String("pr", "", "the merged PR, \"<pr> @ <merged-commit>\"")
	err := fs.Parse(args)
	extra := ""
	switch {
	case err != nil:
	case strings.TrimSpace(*lesson) == "":
		extra = "--lesson <path @ commit>"
	case strings.TrimSpace(*hypothesis) == "":
		extra = "--hypothesis <h-id>@<position>"
	case strings.TrimSpace(*pr) == "":
		extra = "--pr <pr @ commit>"
	}
	if failEnv := knowledgeUsage("knowledge promote", f, err, fs.NArg(), extra); failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	cit, ok := curation.ParseCitation(*hypothesis)
	if !ok || !curation.IsHypothesisSubject(cit.Contract) {
		return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("--hypothesis %q is not <h-id>@<position>", *hypothesis)), stdout, stderr)
	}
	subject := cit.Contract
	l := curation.Lesson{Lesson: *lesson, Hypothesis: *hypothesis, PR: *pr}
	payload := mustJSON(l)
	if _, perr := curation.ParseLesson(subject, payload); perr != nil {
		return render(envelope.Fail(envelope.ExitUsage, "usage", perr.Error()), stdout, stderr)
	}
	signer, failEnv := loopSigner(*f.keyPath, *f.as)
	if failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	ls, failEnv := openLoopSession(f)
	if failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	defer ls.done()
	*f.subject = subject
	return ls.commit(f, loopAct{verb: curation.LessonVerb, payload: payload,
		resultAt: func(int) map[string]any { return map[string]any{"subject": subject, "lesson": *lesson} }}, signer, stdout, stderr)
}

func runKnowledgeShow(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("knowledge show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	f := bindLoopFlags(fs)
	if err := fs.Parse(args); err != nil || (*f.dir == "") == (*f.remote == "") || fs.NArg() != 0 {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "knowledge show requires --ledger <dir> or --remote <repo> (not both)"), stdout, stderr)
	}
	ls, failEnv := openLoopSession(f)
	if failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	defer ls.done()
	view := project.DeriveKnowledge(ls.ctx.Records)
	var out map[string]any
	if err := json.Unmarshal(mustJSON(view), &out); err != nil {
		return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()), stdout, stderr)
	}
	return render(stampTip(envelope.OK(out), ls.ctx.Count), stdout, stderr)
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("null")
	}
	return b
}
