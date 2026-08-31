// Package verdict is the verdict pipeline's first half
// (plans/os-f6d2c267.md; SEED-NEXT.md Part II §8; conformance III.G
// rows 3-5; next/spec/verdicts.md is normative): the clean per-run
// verifier workspace, the profiled runner, and receipt computation.
// The verifier's inputs are enumerable and exclusively self-executed
// or self-read: the submission packet's anchors only name the range,
// and every hash, diff, inventory, and transcript is recomputed from
// the verifier's own checkout and the ledger. Admission-side checks
// (capability, L1 independence, submission binding) live in the admit
// rule set; this package is the run.
package verdict

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"

	"github.com/gowebpki/jcs"

	"github.com/shaunlmason/open-seed/next/internal/artifact"
	"github.com/shaunlmason/open-seed/next/internal/plan"
	"github.com/shaunlmason/open-seed/next/internal/transition"
)

// PlanRef is the approved plan blob hashed at the merge-base; nil in a
// receipt for a planless trivial-tier contract.
type PlanRef struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// Environment is the receipt's environment fingerprint, the runner
// capability profile included: a verdict says what boundary it ran
// under (next/spec/verdicts.md).
type Environment struct {
	OS     string `json:"os"`
	Arch   string `json:"arch"`
	Go     string `json:"go"`
	Runner string `json:"runner"`
}

// Receipt binds contract id, plan hash at merge-base, diff hash,
// changed-file inventory, transcripts, and environment fingerprint
// (III.G row 5). Its digest is the SHA-256 of the JCS bytes, and the
// verdict cites the digest.
type Receipt struct {
	Contract    string       `json:"contract"`
	MergeBase   string       `json:"merge_base"`
	Head        string       `json:"head"`
	Plan        *PlanRef     `json:"plan"`
	DiffSHA256  string       `json:"diff_sha256"`
	Files       []string     `json:"files"`
	Transcripts []Transcript `json:"transcripts"`
	Environment Environment  `json:"environment"`
}

// Canonical returns the receipt's RFC 8785 (JCS) bytes.
func (r *Receipt) Canonical() ([]byte, error) {
	b, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	return jcs.Transform(b)
}

// Digest returns the SHA-256 hex of the canonical bytes: the value a
// verdict.rendered payload cites.
func (r *Receipt) Digest() (string, error) {
	b, err := r.Canonical()
	if err != nil {
		return "", err
	}
	return artifact.Digest(b), nil
}

// RangeError refuses a submission range the verifier cannot attest:
// malformed, unresolvable, or a head that does not descend from its
// merge-base. It rides the shape-refusal mapping.
type RangeError struct {
	Contract string
	Base     string
	Reason   string
}

func (e *RangeError) Error() string {
	return fmt.Sprintf("submission range %q on %s: %s (next/spec/verdicts.md)", e.Base, e.Contract, e.Reason)
}

// UngatedError is the gate-before-run refusal (exit 18 ungated):
// declared-executable acceptance content without gate evidence never
// runs, and no verdict is rendered.
type UngatedError struct {
	Contract string
	Ref      string
}

func (e *UngatedError) Error() string {
	return fmt.Sprintf("acceptance spec %s on %s declares executable content without gate evidence — gate-before-run: nothing runs anywhere until a review gate vouches for the exact revision (next/spec/verdicts.md)", e.Ref, e.Contract)
}

// SpecUnrunnableError is the declared-armed-but-empty refusal (exit 19
// spec_unrunnable): the declaration promised runnable content and the
// body yields no parseable commands, so a vacuous pass must not exist.
type SpecUnrunnableError struct {
	Contract string
	Ref      string
}

func (e *SpecUnrunnableError) Error() string {
	return fmt.Sprintf("acceptance spec %s on %s declares executable content but its validation-commands section yields no parseable commands — silence must never decide, so the run refuses rather than passing vacuously (next/spec/verdicts.md)", e.Ref, e.Contract)
}

// Input names everything one receipt computation consumes. Base is the
// submission packet's mandatory range; PlanAnchor is the approved plan
// anchor ("path @ commit", empty for a planless trivial-tier
// contract); Acceptance is the fold's view of the contract's spec.
type Input struct {
	RepoDir    string
	Contract   string
	Base       string
	PlanAnchor string
	Acceptance *transition.AcceptanceInfo
	Runner     Runner
}

// anchorParts splits a combined anchor "path @ commit".
func anchorParts(anchor string) (path, commit string, ok bool) {
	path, commit, ok = strings.Cut(anchor, " @ ")
	if !ok || path == "" || commit == "" || strings.Contains(commit, "..") {
		return "", "", false
	}
	return path, commit, true
}

// Compute builds the receipt for one submission: resolve and check the
// range, check out the head in a clean per-run workspace, gate-check
// the acceptance spec, run its commands under the profile, and bind
// the result. Cleanup fires pass or fail.
func Compute(in Input) (*Receipt, error) {
	mbRef, headRef, ok := strings.Cut(in.Base, "..")
	if !ok || mbRef == "" || headRef == "" {
		return nil, &RangeError{Contract: in.Contract, Base: in.Base, Reason: "not a <merge-base>..<head> range"}
	}
	ws, err := NewWorkspace(in.RepoDir, headRef)
	if err != nil {
		return nil, err
	}
	defer ws.Cleanup()
	return computeIn(ws, in, mbRef, headRef)
}

func computeIn(ws *Workspace, in Input, mbRef, headRef string) (*Receipt, error) {
	// Full immutable SHAs: a verdict attests exactly this triple, never
	// a ref name (review finding on the plan).
	mb, err := ws.git("rev-parse", "--verify", mbRef+"^{commit}")
	if err != nil {
		return nil, &RangeError{Contract: in.Contract, Base: in.Base, Reason: fmt.Sprintf("merge-base does not resolve: %v", err)}
	}
	head, err := ws.git("rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return nil, &RangeError{Contract: in.Contract, Base: in.Base, Reason: fmt.Sprintf("head does not resolve: %v", err)}
	}
	mb, head = strings.TrimSpace(mb), strings.TrimSpace(head)
	if _, err := ws.git("merge-base", "--is-ancestor", mb, head); err != nil {
		return nil, &RangeError{Contract: in.Contract, Base: in.Base, Reason: fmt.Sprintf("head %.12s does not descend from merge-base %.12s — the range must produce the diff the verdict attests", head, mb)}
	}
	r := &Receipt{
		Contract:    in.Contract,
		MergeBase:   mb,
		Head:        head,
		Files:       []string{},
		Transcripts: []Transcript{},
		Environment: Environment{OS: runtime.GOOS, Arch: runtime.GOARCH, Go: runtime.Version(), Runner: ExecProfile},
	}
	if in.PlanAnchor != "" {
		path, _, ok := anchorParts(in.PlanAnchor)
		if !ok {
			return nil, &RangeError{Contract: in.Contract, Base: in.Base, Reason: fmt.Sprintf("approved plan anchor %q is not \"path @ commit\"", in.PlanAnchor)}
		}
		// The plan hash binds at the merge-base: what the submission
		// was built against, not whatever revision the anchor names
		// (III.G row 5, D3).
		blob, err := ws.git("show", mb+":"+path)
		if err != nil {
			return nil, &RangeError{Contract: in.Contract, Base: in.Base, Reason: fmt.Sprintf("approved plan %s does not exist at the merge-base: %v", path, err)}
		}
		r.Plan = &PlanRef{Path: path, SHA256: artifact.Digest([]byte(blob))}
	}
	diff, err := ws.git("diff", mb, head, "--")
	if err != nil {
		return nil, err
	}
	r.DiffSHA256 = artifact.Digest([]byte(diff))
	names, err := ws.git("diff", "--name-only", mb, head, "--")
	if err != nil {
		return nil, err
	}
	for _, f := range strings.Split(strings.TrimSpace(names), "\n") {
		if f != "" {
			r.Files = append(r.Files, f)
		}
	}
	if in.Acceptance != nil && in.Acceptance.Executable {
		if !in.Acceptance.Gated {
			return nil, &UngatedError{Contract: in.Contract, Ref: in.Acceptance.Ref}
		}
		specPath, specCommit, ok := anchorParts(in.Acceptance.Ref)
		if !ok {
			return nil, &RangeError{Contract: in.Contract, Base: in.Base, Reason: fmt.Sprintf("acceptance ref %q is not \"path @ commit\"", in.Acceptance.Ref)}
		}
		body, err := ws.git("show", specCommit+":"+specPath)
		if err != nil {
			return nil, &RangeError{Contract: in.Contract, Base: in.Base, Reason: fmt.Sprintf("acceptance spec %s does not resolve at its anchored commit: %v", in.Acceptance.Ref, err)}
		}
		cmds := plan.Commands([]byte(body))
		if len(cmds) == 0 {
			return nil, &SpecUnrunnableError{Contract: in.Contract, Ref: in.Acceptance.Ref}
		}
		for _, c := range cmds {
			r.Transcripts = append(r.Transcripts, in.Runner.Run(ws, c))
		}
	}
	return r, nil
}
