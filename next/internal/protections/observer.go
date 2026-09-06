package protections

// Observer reads a pull request's merge state from a forge, so the
// merge.observed fact's {merged} sha can be filled from the forge rather
// than typed by the caller (plans/os-ad610334.md D4). It never writes,
// and the ledger fact is unchanged: {merged, pr}, forge-neutral. GitHub
// and Forgejo both carry `merged` and `merge_commit_sha`, and the
// snapshot arm answers the same shape from a file for the drills.

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// Observer answers a pull request's merge state, and from
// plans/os-0cd18799.md D5 what the forge says about its head: the
// checks, the review threads, the review state (checks.go). It never
// writes.
type Observer interface {
	Merged(pr string) (sha string, merged bool, err error)
	Checks(pr string) (Observation, error)
}

// PRNumber extracts the numeric id from a pr reference ("pr/12" or
// "12"): the merge.observed ref grammar, shared with submission.made's
// pr field and check.observed.
func PRNumber(pr string) (string, error) { return prNumber(pr) }

// prNumber extracts the numeric id from a pr reference ("pr/12" or "12").
func prNumber(pr string) (string, error) {
	n := strings.TrimPrefix(pr, "pr/")
	if n == "" {
		return "", fmt.Errorf("pr %q names no pull request", pr)
	}
	if _, err := strconv.Atoi(n); err != nil {
		return "", fmt.Errorf("pr %q must be a number or pr/<n>", pr)
	}
	return n, nil
}

// prState is the merge-relevant subset both forges' pull-request objects
// carry.
type prState struct {
	Merged         bool   `json:"merged"`
	MergeCommitSHA string `json:"merge_commit_sha"`
}

// Merged reads a GitHub pull request's merge state.
func (g *GitHub) Merged(pr string) (string, bool, error) {
	n, err := prNumber(pr)
	if err != nil {
		return "", false, err
	}
	var st prState
	if _, err := g.do(http.MethodGet, g.repoPath()+"/pulls/"+n, nil, &st); err != nil {
		return "", false, err
	}
	return st.MergeCommitSHA, st.Merged, nil
}

// Merged reads a Forgejo pull request's merge state.
func (f *Forgejo) Merged(pr string) (string, bool, error) {
	n, err := prNumber(pr)
	if err != nil {
		return "", false, err
	}
	var st prState
	if _, err := f.do(http.MethodGet, f.repoPath()+"/pulls/"+n, nil, &st); err != nil {
		return "", false, err
	}
	return st.MergeCommitSHA, st.Merged, nil
}

// SnapshotObserver answers merge state from a JSON file, the
// credential-free arm the drills use:
// {"pulls": {"pr/1": {"merged": true, "merge_commit_sha": "<sha>"}}}.
type SnapshotObserver struct{ Path string }

// readSnapshot reads the snapshot file, naming it in the error.
func readSnapshot(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading the pull-request snapshot: %w", err)
	}
	return b, nil
}

// Merged reads the pull request's state from the snapshot.
func (s SnapshotObserver) Merged(pr string) (string, bool, error) {
	var doc struct {
		Pulls map[string]prState `json:"pulls"`
	}
	if err := s.load(&doc); err != nil {
		return "", false, err
	}
	st, ok := doc.Pulls[pr]
	if !ok {
		return "", false, fmt.Errorf("the snapshot names no pull request %q", pr)
	}
	return st.MergeCommitSHA, st.Merged, nil
}
