// Package eval is the eval-contract machinery (plans/os-03e47abb.md;
// next/spec/evals.md; SEED-NEXT.md §II.16): shipped definitions under
// next/evals/<name>/, anchored in this repository at their last
// reviewed commit; the filing that turns one into an ordinary contract
// marked as an eval; the check that proves its known verdict (fixture
// red, solution green, through the verifier's own runner); and the
// derivation of what the chain owes at a declared instant: mints for
// passes whose receipts recompute green, disqualifications for fails,
// offers for waiting evals, and spot-checks for qualifications older
// than the policy interval. The derivation performs nothing: seed eval
// act signs what its key's grants admit.
package eval

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/shaunlmason/open-seed/next/internal/admit"
	"github.com/shaunlmason/open-seed/next/internal/artifact"
	"github.com/shaunlmason/open-seed/next/internal/keyring"
	"github.com/shaunlmason/open-seed/next/internal/plan"
	"github.com/shaunlmason/open-seed/next/internal/transition"
	"github.com/shaunlmason/open-seed/next/internal/tuple"
	"github.com/shaunlmason/open-seed/next/internal/verdict"
	"github.com/shaunlmason/open-seed/next/internal/version"
)

// Root is where definitions live, relative to the repository root.
const Root = "next/evals"

// Applies reports whether the qualification verbs and the eval marker
// are defined at a record carrying version v: seed/3 exactly (D8).
func Applies(v string) bool { return version.EvalApplies(v) }

// Definition is one shipped eval: eval.json beside a fixture/ (the
// files the eval is worked in, the acceptance spec among them) and a
// solution/ (the files the reference solution changes).
type Definition struct {
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Tier    string `json:"tier"`
	// Acceptance is the acceptance spec's repository-relative path,
	// under this definition's fixture/.
	Acceptance string `json:"acceptance"`
}

// Dir is the definition's repository-relative directory.
func (d Definition) Dir() string { return Root + "/" + d.Name }

// Load reads every definition under <repo>/next/evals, sorted by name,
// and refuses a malformed one by name: a definition is armed content,
// and one that does not say what it is files nothing.
func Load(repo string) ([]Definition, error) {
	entries, err := os.ReadDir(filepath.Join(repo, Root))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Definition
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(repo, Root, e.Name(), "eval.json"))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		var d Definition
		if err := json.Unmarshal(raw, &d); err != nil {
			return nil, fmt.Errorf("eval %s: eval.json: %v", e.Name(), err)
		}
		if d.Name != e.Name() {
			return nil, fmt.Errorf("eval %s: eval.json names %q; a definition is named by its directory", e.Name(), d.Name)
		}
		if strings.TrimSpace(d.Tier) == "" || strings.TrimSpace(d.Summary) == "" {
			return nil, fmt.Errorf("eval %s: eval.json needs a tier and a summary", d.Name)
		}
		if !strings.HasPrefix(d.Acceptance, d.Dir()+"/fixture/") {
			return nil, fmt.Errorf("eval %s: the acceptance spec must live under %s/fixture/, got %q", d.Name, d.Dir(), d.Acceptance)
		}
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(d.Acceptance))); err != nil {
			return nil, fmt.Errorf("eval %s: the acceptance spec %s is not in the tree: %v", d.Name, d.Acceptance, err)
		}
		if sol, err := os.ReadDir(filepath.Join(repo, Root, d.Name, "solution")); err != nil || len(sol) == 0 {
			return nil, fmt.Errorf("eval %s: solution/ must carry the reference solution's files", d.Name)
		}
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Find resolves a definition by name.
func Find(defs []Definition, name string) (Definition, bool) {
	for _, d := range defs {
		if d.Name == name {
			return d, true
		}
	}
	return Definition{}, false
}

// Anchor is the reviewed revision an eval is filed at (D3): the last
// commit touching the definition and the PR whose squash merge landed
// it, both read from the repository. Ref and Gate name the SAME commit,
// which is what acceptance.md's equality rule demands, and both are
// facts about a review that happened.
type Anchor struct {
	Commit string
	PR     string
}

var prSubject = regexp.MustCompile(`\(#(\d+)\)\s*$`)

// ErrNotReviewed refuses a definition whose last commit is not a merged
// pull request, or which is dirty in the working tree: content that has
// not been through the gate arms nothing.
var ErrNotReviewed = errors.New("the eval definition is not at a reviewed revision")

// AnchorOf reads the definition's anchor from the repository.
func AnchorOf(repo string, def Definition) (Anchor, error) {
	dirty, err := git(repo, "status", "--porcelain", "--", def.Dir())
	if err != nil {
		return Anchor{}, err
	}
	if strings.TrimSpace(dirty) != "" {
		return Anchor{}, fmt.Errorf("%w: %s has uncommitted changes", ErrNotReviewed, def.Dir())
	}
	out, err := git(repo, "log", "-1", "--format=%H%x1f%s", "--", def.Dir())
	if err != nil {
		return Anchor{}, err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return Anchor{}, fmt.Errorf("%w: no commit touches %s", ErrNotReviewed, def.Dir())
	}
	commit, subject, _ := strings.Cut(out, "\x1f")
	m := prSubject.FindStringSubmatch(subject)
	if m == nil {
		return Anchor{}, fmt.Errorf("%w: the last commit touching %s (%.12s, %q) carries no merged pull request number in its subject", ErrNotReviewed, def.Dir(), commit, subject)
	}
	return Anchor{Commit: commit, PR: "pr/" + m[1]}, nil
}

// Ref is the acceptance anchor the filing cites.
func (a Anchor) Ref(def Definition) string { return def.Acceptance + " @ " + a.Commit }

// Gate is the gate evidence the filing cites: the merged review, at
// the ref's own commit.
func (a Anchor) Gate() string { return a.PR + " @ " + a.Commit }

func git(repo string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// ErrVacuous refuses an eval whose unsolved fixture already passes:
// it decides nothing, so a pass on it would qualify nobody
// (eval_vacuous, a refinement in spec_unrunnable's family).
var ErrVacuous = errors.New("the eval is vacuous: its unsolved fixture already passes every acceptance command, so it decides nothing")

// ErrSolutionRed refuses an eval whose reference solution stays red:
// the known verdict cannot be reproduced (checks_red).
var ErrSolutionRed = errors.New("the eval's reference solution stays red: its known verdict cannot be reproduced")

// CheckReport is what a check ran: the acceptance commands and their
// transcripts over the fixture and over the solution.
type CheckReport struct {
	Commands []string
	Fixture  []verdict.Transcript
	Solution []verdict.Transcript
}

// Check proves the known verdict: in a verifier workspace at the
// anchor it runs the acceptance commands and requires red, overlays
// the solution files and requires green. The workspace is removed
// either way.
func Check(repo string, def Definition, anchor Anchor, runner verdict.Runner) (CheckReport, error) {
	ws, err := verdict.NewWorkspace(repo, anchor.Commit)
	if err != nil {
		return CheckReport{}, err
	}
	defer ws.Cleanup()
	body, err := os.ReadFile(filepath.Join(ws.Repo, filepath.FromSlash(def.Acceptance)))
	if err != nil {
		return CheckReport{}, fmt.Errorf("eval %s: acceptance spec %s at %.12s: %v", def.Name, def.Acceptance, anchor.Commit, err)
	}
	rep := CheckReport{Commands: plan.Commands(body)}
	if len(rep.Commands) == 0 {
		return rep, &verdict.SpecUnrunnableError{Contract: def.Name, Ref: anchor.Ref(def)}
	}
	for _, c := range rep.Commands {
		rep.Fixture = append(rep.Fixture, runner.Run(ws, c))
	}
	if allGreen(rep.Fixture) {
		return rep, ErrVacuous
	}
	if err := overlay(filepath.Join(repo, Root, def.Name, "solution"), filepath.Join(ws.Repo, Root, def.Name, "fixture")); err != nil {
		return rep, err
	}
	for _, c := range rep.Commands {
		rep.Solution = append(rep.Solution, runner.Run(ws, c))
	}
	if !allGreen(rep.Solution) {
		return rep, ErrSolutionRed
	}
	return rep, nil
}

func allGreen(ts []verdict.Transcript) bool {
	for _, t := range ts {
		if t.Exit != 0 {
			return false
		}
	}
	return true
}

// overlay copies every file under from onto to, creating directories.
func overlay(from, to string) error {
	return filepath.WalkDir(from, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(from, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(to, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(dst, b, info.Mode().Perm())
	})
}

// Filing is what turns a definition into a contract: the intent (with
// the eval marker) and the specification (the anchored acceptance and
// its gate), under a stable id derived from the eval, the tuple under
// re-test and the count of prior evals for that pair, so a second run
// in the same window refuses the duplicate at the boundary.
type Filing struct {
	Subject string
	Intent  []byte
	Spec    []byte
}

// File shapes the filing for a definition at its anchor; tu is the
// configuration under re-test on a spot-check, nil on a first eval.
func File(def Definition, anchor Anchor, tu *tuple.Tuple, prior int) (Filing, error) {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%d", def.Name, tupleKey(tu), prior)))
	id := "eval-" + hex.EncodeToString(sum[:6])
	marker := map[string]any{"name": def.Name}
	if tu != nil {
		marker["tuple"] = *tu
	}
	intent, err := json.Marshal(map[string]any{
		"intent":  fmt.Sprintf("eval %s: %s", def.Name, def.Summary),
		"tier":    def.Tier,
		"budget":  "small",
		"routing": "core",
		"eval":    marker,
	})
	if err != nil {
		return Filing{}, err
	}
	spec, err := json.Marshal(map[string]any{
		"acceptance": map[string]any{"ref": anchor.Ref(def), "executable": true, "gate": anchor.Gate()},
	})
	if err != nil {
		return Filing{}, err
	}
	return Filing{Subject: id, Intent: intent, Spec: spec}, nil
}

func tupleKey(tu *tuple.Tuple) string {
	if tu == nil {
		return ""
	}
	b, _ := json.Marshal(tu)
	return string(b)
}

// The act kinds Due returns, and the lane each is owed by.
const (
	KindMint       = "mint"
	KindDisqualify = "disqualify"
	KindOffer      = "offer"
	KindSpotCheck  = "spot-check"

	LaneSupervise = keyring.CapSupervise
	LaneDispatch  = keyring.CapDispatch
)

// Act is one act the chain owes: the verb and payload to sign, on
// which subject, and the capability that owns it.
type Act struct {
	Kind    string `json:"kind"`
	Verb    string `json:"verb"`
	Subject string `json:"subject"`
	Payload string `json:"payload"`
	Lane    string `json:"lane"`
	Because string `json:"because"`
}

// Note is something Due looked at and did not turn into an act, by
// name: a receipt that did not recompute, a qualification dated in the
// future, a definition that is missing.
type Note struct {
	Kind    string `json:"kind"`
	Subject string `json:"subject"`
	Detail  string `json:"detail"`
}

// Report is what one derivation found.
type Report struct {
	Acts  []Act  `json:"acts"`
	Notes []Note `json:"notes"`
}

// Inputs is everything Due reads. Now is the DECLARED instant, never a
// clock read, so a run replays; After is the spot-check interval (zero
// disables); Repo and Store are what the receipt recomputation needs.
type Inputs struct {
	Ctx     *admit.Context
	Ring    *keyring.State
	Store   *artifact.Store
	Repo    string
	Now     time.Time
	After   time.Duration
	Evals   []Definition
	Timeout time.Duration
	// OfferTTL is how long an offer Due publishes stays live; the
	// default is a day.
	OfferTTL time.Duration
}

// Due derives the acts owed at the declared instant (D5). Mints come
// only with a recomputed green receipt (D2); disqualifications are
// tuple-wide (D4); offers are never tuple-scoped (D6); spot-checks age
// from the qualification record's own ts, a future-dated one being due
// at once (D5).
func Due(in Inputs) Report {
	rep := Report{Acts: []Act{}, Notes: []Note{}}
	if in.Ctx == nil || in.Ctx.Lifecycle == nil || in.Ring == nil {
		return rep
	}
	ttl := in.OfferTTL
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	fold := in.Ctx.Lifecycle
	for _, subject := range fold.Subjects() {
		s, ok := fold.State(subject)
		if !ok || s.Eval == nil {
			continue
		}
		// Offers: a waiting eval with no live offer.
		if s.State == "ready" && len(s.LiveOffers(in.Now)) == 0 {
			payload, _ := json.Marshal(map[string]any{
				"eligibility": map[string]any{"capabilities": []string{keyring.CapClaim}, "tiers": []string{s.Tier}},
				"expires":     in.Now.Add(ttl).UTC().Format(time.RFC3339),
			})
			rep.Acts = append(rep.Acts, Act{Kind: KindOffer, Verb: transition.OfferPublishedVerb, Subject: subject,
				Payload: string(payload), Lane: LaneSupervise, Because: "the eval is ready and no offer is live"})
		}
		// Mints: an authenticated pass whose receipt recomputes green.
		if pass := admit.AuthenticPass(in.Ctx, subject, s); pass != nil {
			fence, holder, ok := admit.SubmissionWindow(in.Ctx.Records, subject, pass.Submission)
			declared := (*tuple.Tuple)(nil)
			if ok {
				declared = admit.WindowDeclaration(in.Ctx, subject, s, fence)
			}
			switch {
			case !ok || declared == nil:
				rep.Notes = append(rep.Notes, Note{Kind: "no_declaration", Subject: subject,
					Detail: "the pass verdict's window carries no admitted run.started declaring a tuple, so nothing qualifies"})
			case alreadyCited(in.Ring, holder, subject, pass.Pos, false):
				// minted already; nothing owed
			default:
				if note := recompute(in, subject, s, pass); note != nil {
					rep.Notes = append(rep.Notes, *note)
				} else {
					payload, _ := json.Marshal(map[string]any{
						"capability": keyring.CapClaim, "tuple": *declared, "contract": subject, "verdict": fmt.Sprintf("%d", pass.Pos),
					})
					rep.Acts = append(rep.Acts, Act{Kind: KindMint, Verb: keyring.VerbQualified, Subject: holder,
						Payload: string(payload), Lane: LaneSupervise,
						Because: fmt.Sprintf("the eval's pass at position %d recomputed green for the configuration the run declared", pass.Pos)})
				}
			}
		}
		// Disqualifications: an authenticated fail, tuple-wide.
		if fail := admit.AuthenticFail(in.Ctx, subject, s); fail != nil {
			fence, _, ok := admit.SubmissionWindow(in.Ctx.Records, subject, fail.Submission)
			declared := (*tuple.Tuple)(nil)
			if ok {
				declared = admit.WindowDeclaration(in.Ctx, subject, s, fence)
			}
			if declared == nil {
				rep.Notes = append(rep.Notes, Note{Kind: "no_declaration", Subject: subject,
					Detail: "the fail verdict's window carries no admitted run.started declaring a tuple, so nothing is disqualified"})
			} else {
				for _, actor := range in.Ring.Actors() {
					if !holds(in.Ring, actor, *declared) || alreadyCited(in.Ring, actor, subject, fail.Pos, true) {
						continue
					}
					payload, _ := json.Marshal(map[string]any{
						"capability": keyring.CapClaim, "tuple": *declared, "contract": subject, "verdict": fmt.Sprintf("%d", fail.Pos),
						"reason": fmt.Sprintf("the eval %s failed at position %d under this configuration", subject, fail.Pos),
					})
					rep.Acts = append(rep.Acts, Act{Kind: KindDisqualify, Verb: keyring.VerbDisqualified, Subject: actor,
						Payload: string(payload), Lane: LaneSupervise,
						Because: fmt.Sprintf("the eval's fail at position %d names a configuration this actor holds", fail.Pos)})
				}
			}
		}
	}
	if in.After > 0 {
		rep = spotChecks(in, rep)
	}
	return rep
}

// recompute retrieves the cited receipt and recomputes it against the
// submission head with the same function seed verdict check runs; a
// nil return means it retrieved intact, reproduced the digest, and
// every transcript exited zero.
func recompute(in Inputs, subject string, s transition.SubjectState, pass *transition.VerdictFact) *Note {
	if in.Store == nil || in.Repo == "" {
		return &Note{Kind: "receipt_unchecked", Subject: subject, Detail: "no artifact store or repository to recompute the cited receipt against; nothing is minted on an unchecked receipt"}
	}
	if _, err := in.Store.Get(pass.Receipt); err != nil {
		return &Note{Kind: "receipt_missing", Subject: subject, Detail: fmt.Sprintf("the cited receipt %s is not retrievable intact: %v", pass.Receipt, err)}
	}
	if s.Sealed != nil {
		return &Note{Kind: "receipt_unchecked", Subject: subject, Detail: "the eval carries sealed checks, which this derivation does not unseal; nothing is minted on an unchecked receipt"}
	}
	input, err := verdict.InputFor(in.Ctx.Records, in.Ctx.Lifecycle, s, subject, in.Repo, in.Timeout)
	if err != nil {
		return &Note{Kind: "receipt_mismatch", Subject: subject, Detail: fmt.Sprintf("the submission cannot be recomputed: %v", err)}
	}
	r, err := verdict.Compute(input)
	if err != nil {
		return &Note{Kind: "receipt_mismatch", Subject: subject, Detail: fmt.Sprintf("recomputation failed: %v", err)}
	}
	digest, err := r.Digest()
	if err != nil {
		return &Note{Kind: "receipt_mismatch", Subject: subject, Detail: err.Error()}
	}
	if digest != pass.Receipt {
		return &Note{Kind: "receipt_mismatch", Subject: subject, Detail: fmt.Sprintf("the cited receipt %s does not reproduce from the submission head (recomputed %s)", pass.Receipt, digest)}
	}
	if !allGreen(r.Transcripts) || !allGreen(r.SealedTranscripts) {
		return &Note{Kind: "checks_red", Subject: subject, Detail: "the recomputed receipt carries a red transcript: a pass over red qualifies nobody"}
	}
	return nil
}

func holds(ring *keyring.State, actor string, t tuple.Tuple) bool {
	for _, have := range ring.GrantTuples(actor, keyring.CapClaim) {
		if have.Equal(t) {
			return true
		}
	}
	return false
}

func alreadyCited(ring *keyring.State, actor, contract string, verdictPos int, disqualified bool) bool {
	for _, q := range ring.Qualifications(actor) {
		if q.Contract == contract && q.Verdict == verdictPos && q.Disqualified == disqualified {
			return true
		}
	}
	return false
}

// spotChecks files a re-test for every admissible (actor, tuple) whose
// latest qualification is older than the interval and which no open
// eval already names.
func spotChecks(in Inputs, rep Report) Report {
	fold := in.Ctx.Lifecycle
	filed := map[string]bool{}
	for _, actor := range in.Ring.Actors() {
		for _, held := range in.Ring.GrantTuples(actor, keyring.CapClaim) {
			var latest *keyring.Qualification
			for i := range in.Ring.Qualifications(actor) {
				q := in.Ring.Qualifications(actor)[i]
				if q.Capability == keyring.CapClaim && !q.Disqualified && q.Tuple.Equal(held) {
					latest = &q
				}
			}
			if latest == nil {
				continue // granted by hand: no eval to re-run
			}
			ts, err := time.Parse(time.RFC3339, latest.TS)
			if err != nil {
				rep.Notes = append(rep.Notes, Note{Kind: "anomaly", Subject: actor, Detail: fmt.Sprintf("the qualification citing %s carries an unreadable ts %q; treated as due", latest.Contract, latest.TS)})
			} else if ts.After(in.Now) {
				rep.Notes = append(rep.Notes, Note{Kind: "anomaly", Subject: actor, Detail: fmt.Sprintf("the qualification citing %s is dated %s, after the declared instant %s; a lie about time cannot postpone a re-test, so it is due now", latest.Contract, latest.TS, in.Now.UTC().Format(time.RFC3339))})
			} else if in.Now.Sub(ts) <= in.After {
				continue
			}
			key := tupleKey(&held)
			if filed[key] {
				continue
			}
			name, open, prior := evalsNaming(fold, latest.Contract, held)
			if name == "" {
				rep.Notes = append(rep.Notes, Note{Kind: "no_definition", Subject: actor, Detail: fmt.Sprintf("the qualification cites %s, which names no eval; nothing to re-run", latest.Contract)})
				continue
			}
			if open {
				continue
			}
			def, ok := Find(in.Evals, name)
			if !ok {
				rep.Notes = append(rep.Notes, Note{Kind: "no_definition", Subject: actor, Detail: fmt.Sprintf("eval %s is not among the shipped definitions", name)})
				continue
			}
			anchor, err := AnchorOf(in.Repo, def)
			if err != nil {
				rep.Notes = append(rep.Notes, Note{Kind: "not_reviewed", Subject: actor, Detail: err.Error()})
				continue
			}
			f, err := File(def, anchor, &held, prior)
			if err != nil {
				rep.Notes = append(rep.Notes, Note{Kind: "anomaly", Subject: actor, Detail: err.Error()})
				continue
			}
			filed[key] = true
			because := fmt.Sprintf("the qualification of %s citing %s is older than %s; re-testing the configuration", actor, latest.Contract, in.After)
			rep.Acts = append(rep.Acts,
				Act{Kind: KindSpotCheck, Verb: "intent.filed", Subject: f.Subject, Payload: string(f.Intent), Lane: LaneDispatch, Because: because},
				Act{Kind: KindSpotCheck, Verb: "contract.specified", Subject: f.Subject, Payload: string(f.Spec), Lane: LaneDispatch, Because: because})
		}
	}
	return rep
}

// Prior counts the evals already filed for the definition and the
// configuration under re-test (nil on a first eval): the count File's
// stable id derives from, so a satisfied window advances it and a
// second run in the same window refuses the duplicate.
func Prior(fold *transition.Fold, name string, tu *tuple.Tuple) int {
	n := 0
	for _, subject := range fold.Subjects() {
		s, ok := fold.State(subject)
		if !ok || s.Eval == nil || s.Eval.Name != name {
			continue
		}
		switch {
		case tu == nil && s.Eval.Tuple == nil:
			n++
		case tu != nil && s.Eval.Tuple != nil && s.Eval.Tuple.Equal(*tu):
			n++
		}
	}
	return n
}

// evalsNaming reads, for the eval a qualification cites, its
// definition's name, whether an eval naming this tuple is still open,
// and how many evals have named this (name, tuple) pair.
func evalsNaming(fold *transition.Fold, contract string, t tuple.Tuple) (name string, open bool, prior int) {
	if s, ok := fold.State(contract); ok && s.Eval != nil {
		name = s.Eval.Name
	}
	if name == "" {
		return "", false, 0
	}
	for _, subject := range fold.Subjects() {
		s, ok := fold.State(subject)
		if !ok || s.Eval == nil || s.Eval.Name != name || s.Eval.Tuple == nil || !s.Eval.Tuple.Equal(t) {
			continue
		}
		prior++
		if s.Verdict == nil && s.State != "done" && s.State != "cancelled" {
			open = true
		}
	}
	return name, open, prior
}
