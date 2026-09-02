// The knowledge verbs (plans/os-f30ee0d3.md D4; next/spec/curation.md):
// the three curation facts driven against a real ledger with the fence
// and the hypothesis id derived, and the fold rendered. No loop act:
// the worker loop's exits already carry findings, and a standalone dead
// end is the holder's deliberate extra, like plan propose.

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
	"strings"
	"time"

	"github.com/shaunlmason/open-seed/next/internal/admit"
	"github.com/shaunlmason/open-seed/next/internal/curation"
	"github.com/shaunlmason/open-seed/next/internal/envelope"
	"github.com/shaunlmason/open-seed/next/internal/project"
)

func runKnowledge(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "knowledge requires a subverb: deadend | propose | validate | contest | promote | lint | show"), stdout, stderr)
	}
	switch args[0] {
	case "deadend":
		return runKnowledgeDeadEnd(args[1:], stdout, stderr)
	case "propose":
		return runKnowledgePropose(args[1:], stdout, stderr)
	case "validate":
		return runKnowledgeValidate(args[1:], stdout, stderr)
	case "contest":
		return runKnowledgeContest(args[1:], stdout, stderr)
	case "promote":
		return runKnowledgePromote(args[1:], stdout, stderr)
	case "lint":
		return runKnowledgeLint(args[1:], stdout, stderr)
	case "show":
		return runKnowledgeShow(args[1:], stdout, stderr)
	}
	return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("unknown knowledge subverb %q: deadend | propose | validate | contest | promote | lint | show", args[0])), stdout, stderr)
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
	applies := fs.String("applies-when", "", "the predicate the claim holds under, the strict JSON object {routing?, tier?, paths?}")
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
	subject := curation.HypothesisID(*claim, exceptions)
	h := curation.Hypothesis{Claim: *claim, AppliesWhen: json.RawMessage(*applies), Support: support, Exceptions: exceptions, Provenance: provenance}
	if !json.Valid(h.AppliesWhen) {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "--applies-when is the strict JSON object {routing?, tier?, paths?}"), stdout, stderr)
	}
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
	repo := fs.String("repo", "", "the repository the lesson merged in: the digest is read from the file at its anchor")
	carrier := fs.String("carrier", "", "where the lesson lands: "+strings.Join(curation.Carriers, " | "))
	adversarial := fs.String("adversarial", "", "the counter-trajectory it survived, <eval>@<verdict position>")
	lastValidated := fs.String("last-validated", "", "RFC3339: restates the reviewed file's last-validated (read from the frontmatter at the anchor; refuses when it disagrees)")
	expires := fs.String("expires", "", "RFC3339: restates the reviewed file's expires (read from the frontmatter at the anchor; refuses when it disagrees)")
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
	case strings.TrimSpace(*repo) == "":
		extra = "--repo <dir>"
	case strings.TrimSpace(*carrier) == "":
		extra = "--carrier <" + strings.Join(curation.Carriers, "|") + ">"
	case strings.TrimSpace(*adversarial) == "":
		extra = "--adversarial <eval>@<verdict position>"
	}
	if failEnv := knowledgeUsage("knowledge promote", f, err, fs.NArg(), extra); failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	cit, ok := curation.ParseCitation(*hypothesis)
	if !ok || !curation.IsHypothesisSubject(cit.Contract) {
		return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("--hypothesis %q is not <h-id>@<position>", *hypothesis)), stdout, stderr)
	}
	evalName, verdictPos, ok := strings.Cut(*adversarial, "@")
	if !ok || evalName == "" || verdictPos == "" {
		return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("--adversarial %q is not <eval>@<verdict position>", *adversarial)), stdout, stderr)
	}
	path, commit, ok := curation.AnchorParts(*lesson)
	if !ok {
		return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("--lesson %q is not \"<path> @ <commit>\"", *lesson)), stdout, stderr)
	}
	at, gerr := exec.Command("git", "-C", *repo, "show", commit+":"+path).Output()
	if gerr != nil {
		return render(envelope.Fail(envelope.ExitNotFound, "not_found", fmt.Sprintf("the lesson %s does not exist at its anchor in %s: the digest is read from the merged file, never typed", *lesson, *repo)), stdout, stderr)
	}
	// The lifecycle stamps are the reviewed file's (review finding on
	// the item 3 PR): read from the frontmatter at the anchor, never
	// typed; a flag may restate them and refuses when it disagrees, so
	// the fact never carries dates nobody reviewed.
	fm, ferr := curation.Frontmatter(at)
	if ferr != nil {
		return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("the lesson at %s has no readable frontmatter: %v", *lesson, ferr)), stdout, stderr)
	}
	for _, st := range []struct{ flag, key, given string }{{"--last-validated", "last-validated", *lastValidated}, {"--expires", "expires", *expires}} {
		if strings.TrimSpace(st.given) != "" && strings.TrimSpace(st.given) != fm[st.key] {
			return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("%s %s disagrees with the reviewed file's %s %s at %s: the stamps are the merged frontmatter's, and a promotion recording others would carry dates nobody reviewed", st.flag, st.given, st.key, fm[st.key], *lesson)), stdout, stderr)
		}
	}
	subject := cit.Contract
	l := curation.Lesson{Lesson: *lesson, Hypothesis: *hypothesis, PR: *pr, Carrier: *carrier,
		Adversarial:   &curation.Adversarial{Eval: evalName, Verdict: verdictPos},
		LastValidated: fm["last-validated"], Expires: fm["expires"], Digest: curation.Digest(at)}
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

// runKnowledgeValidate is the held-out listing (plans/os-96850e5a.md
// D3): the admitted observations on contracts the hypothesis's
// predicate selects that are outside its support set, for the curator
// to judge, then contest or promote. No machine decides that an
// outcome contradicts a claim.
func runKnowledgeValidate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("knowledge validate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	f := bindLoopFlags(fs)
	hypothesis := fs.String("hypothesis", "", "the admitted hypothesis, <h-id>@<position>")
	if err := fs.Parse(args); err != nil || (*f.dir == "") == (*f.remote == "") || *hypothesis == "" || fs.NArg() != 0 {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "knowledge validate requires --ledger <dir> or --remote <repo> (not both) and --hypothesis <h-id>@<position>"), stdout, stderr)
	}
	cit, ok := curation.ParseCitation(*hypothesis)
	if !ok || !curation.IsHypothesisSubject(cit.Contract) {
		return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("--hypothesis %q is not <h-id>@<position>", *hypothesis)), stdout, stderr)
	}
	ls, failEnv := openLoopSession(f)
	if failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	defer ls.done()
	st := curation.Fold(ls.ctx.Records)
	h, ok := st.Hypothesis(cit.Contract)
	if !ok || h.Pos != cit.Position {
		return render(stampTip(envelope.Fail(envelope.ExitNotFound, "not_found", fmt.Sprintf("no hypothesis %s in the fold", *hypothesis)), ls.ctx.Count), stdout, stderr)
	}
	rows := []map[string]any{}
	for _, o := range curation.HeldOut(ls.ctx.Records, ls.ctx.Table, ls.ctx.Lifecycle, h) {
		rows = append(rows, map[string]any{"contract": o.Contract, "position": fmt.Sprintf("%d", o.Position), "verb": o.Verb, "holder": o.Holder})
	}
	return render(stampTip(envelope.OK(map[string]any{"hypothesis": *hypothesis, "stage": h.Stage, "support": h.Support, "held_out": rows}), ls.ctx.Count), stdout, stderr)
}

func runKnowledgeContest(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("knowledge contest", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	f := bindLoopFlags(fs)
	hypothesis := fs.String("hypothesis", "", "the admitted hypothesis, <h-id>@<position>")
	reason := fs.String("reason", "", "why the evidence contradicts the claim")
	var evidence listFlag
	fs.Var(&evidence, "evidence", "a held-out observation, <contract>@<position> (repeatable)")
	err := fs.Parse(args)
	extra := ""
	switch {
	case err != nil:
	case strings.TrimSpace(*hypothesis) == "":
		extra = "--hypothesis <h-id>@<position>"
	case len(evidence) == 0:
		extra = "at least one --evidence <contract>@<position>"
	case strings.TrimSpace(*reason) == "":
		extra = "--reason <text>"
	}
	if failEnv := knowledgeUsage("knowledge contest", f, err, fs.NArg(), extra); failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	cit, ok := curation.ParseCitation(*hypothesis)
	if !ok || !curation.IsHypothesisSubject(cit.Contract) {
		return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("--hypothesis %q is not <h-id>@<position>", *hypothesis)), stdout, stderr)
	}
	subject := cit.Contract
	payload := mustJSON(curation.Contest{Hypothesis: *hypothesis, Evidence: evidence, Reason: *reason})
	if _, perr := curation.ParseContest(subject, payload); perr != nil {
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
	return ls.commit(f, loopAct{verb: curation.ContestVerb, payload: payload,
		resultAt: func(int) map[string]any { return map[string]any{"subject": subject, "hypothesis": *hypothesis} }}, signer, stdout, stderr)
}

// runKnowledgeLint is the promotion gate's file half
// (plans/os-96850e5a.md D4): the lesson file against its fact, its
// hypothesis and the repository, at the declared instant.
func runKnowledgeLint(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("knowledge lint", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	f := bindLoopFlags(fs)
	repo := fs.String("repo", "", "the repository the lesson merged in")
	nowFlag := fs.String("now", "", "RFC3339: the instant the stamps are judged at (default: now)")
	if err := fs.Parse(args); err != nil || (*f.dir == "") == (*f.remote == "") || *repo == "" || fs.NArg() != 1 {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "knowledge lint <file> requires --ledger <dir> or --remote <repo> (not both) and --repo <dir> [--now <RFC3339>]"), stdout, stderr)
	}
	now := time.Now().UTC()
	if *nowFlag != "" {
		parsed, err := time.Parse(time.RFC3339, *nowFlag)
		if err != nil {
			return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("--now %q is not RFC3339", *nowFlag)), stdout, stderr)
		}
		now = parsed.UTC()
	}
	file := fs.Arg(0)
	body, err := os.ReadFile(file)
	if err != nil {
		return render(envelope.Fail(envelope.ExitNotFound, "not_found", fmt.Sprintf("cannot read %s: %v", file, err)), stdout, stderr)
	}
	ls, failEnv := openLoopSession(f)
	if failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	defer ls.done()
	fm, err := curation.Frontmatter(body)
	if err != nil {
		return render(stampTip(envelope.Fail(envelope.ExitChecksRed, "lint_refused", err.Error()), ls.ctx.Count), stdout, stderr)
	}
	st := curation.Fold(ls.ctx.Records)
	cit, ok := curation.ParseCitation(fm["hypothesis"])
	h, found := st.Hypothesis(cit.Contract)
	if !ok || !found || h.Pos != cit.Position {
		return render(stampTip(envelope.Fail(envelope.ExitNotFound, "not_found", fmt.Sprintf("the frontmatter cites hypothesis %q, which the fold does not hold", fm["hypothesis"])), ls.ctx.Count), stdout, stderr)
	}
	var fact *curation.LessonFact
	rel := relativeTo(*repo, file)
	for _, l := range st.LessonsOf(cit.Contract) {
		if p, _, _ := curation.AnchorParts(l.Lesson); p == rel {
			fact = &l
		}
	}
	if fact == nil {
		return render(stampTip(envelope.Fail(envelope.ExitNotFound, "not_found", fmt.Sprintf("no admitted promotion cites %s: the file half judges a lesson the ledger promoted", rel)), ls.ctx.Count), stdout, stderr)
	}
	if err := curation.LintFile(*repo, body, *fact, h, now); err != nil {
		var ge *curation.GateError
		if errors.As(err, &ge) {
			return render(stampTip(envelope.Fail(envelope.ExitChecksRed, "lint_refused", err.Error()), ls.ctx.Count), stdout, stderr)
		}
		return render(stampTip(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()), ls.ctx.Count), stdout, stderr)
	}
	return render(stampTip(envelope.OK(map[string]any{"lesson": fact.Lesson, "hypothesis": fact.Hypothesis, "carrier": fact.Carrier, "digest": fact.Digest, "lint": "ok"}), ls.ctx.Count), stdout, stderr)
}

// relativeTo is the file's repository-relative path when it lies
// under the repository, else the path as given.
func relativeTo(repo, file string) string {
	absRepo, err1 := filepath.Abs(repo)
	absFile, err2 := filepath.Abs(file)
	if err1 != nil || err2 != nil {
		return file
	}
	rel, err := filepath.Rel(absRepo, absFile)
	if err != nil || strings.HasPrefix(rel, "..") {
		return file
	}
	return filepath.ToSlash(rel)
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
