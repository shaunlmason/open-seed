// Package gitref rides the ledger on a git ref (charter Part II section 1
// Reference deployment; plans/os-62e2aa1d.md): a small exec seam around the
// git CLI (the engine's gitx pattern, no module dependencies) plus the
// optimistic append loop that makes the ref safely multi-writer. The loop
// re-links and re-signs on every retry (prev is inside the signed form, so
// a signing closure is the only way to move a draft onto a fresh tip), and
// the client persists its newest verified remote head, refusing head
// regression (charter III.A freshness; the monotonic-head rule).
package gitref

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Typed refusals: each is a distinct condition callers branch on.
var (
	ErrNonFastForward = errors.New("push rejected: non-fast-forward (another writer won the race)")
	ErrRemoteRejected = errors.New("push rejected by the remote (a policy refusal, not a race)")
	ErrUnavailable    = errors.New("remote unreachable")
	ErrHeadRegression = errors.New("fetched tip regresses the persisted verified head — refusing (monotonic-head rule)")
	ErrRetriesSpent   = errors.New("append retries exhausted without winning the race")
)

// Client is one actor's view of a remote ledger ref: a private git dir for
// fetches and commits, a head cache dir for the persisted verified heads
// (client state, never ledger state), and the remote+ref coordinates.
type Client struct {
	Remote   string
	Ref      string
	gitDir   string
	cacheDir string
}

// NewClient prepares the client's git dir under stateDir and uses
// stateDir/heads as the persisted-head cache.
func NewClient(stateDir, remote, ref string) (*Client, error) {
	gitDir := filepath.Join(stateDir, "gitdir")
	cacheDir := filepath.Join(stateDir, "heads")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(gitDir, "HEAD")); errors.Is(err, os.ErrNotExist) {
		if _, err := runGit("", "init", "-q", "--bare", gitDir); err != nil {
			return nil, err
		}
	}
	if err := hardenGitDir(gitDir); err != nil {
		return nil, err
	}
	return &Client{Remote: remote, Ref: ref, gitDir: gitDir, cacheDir: cacheDir}, nil
}

// noAutoGC is the repository-local configuration every git dir the
// engine creates for itself carries (plans/os-711b3028.md D1, D2):
// the client's private transport dir and the verifier's per-run
// clone are ephemeral, engine-owned state, and a collector git
// detaches after a fetch, a receive or a checkout would mutate them
// after the process that armed it has exited. A caller removing its
// state dir (a CI job cleaning a workspace) then hits the same
// directory-not-empty failure the drills hit under t.TempDir.
var noAutoGC = [][2]string{
	{"gc.auto", "0"},
	{"gc.autoDetach", "false"},
	{"receive.autoGC", "false"},
}

// hardenGitDir writes noAutoGC into the git dir on every construction,
// not only at init: a state dir an older build created is hardened the
// first time a new build opens it, and three idempotent config writes
// cost less than a stat-and-branch that could drift from what the
// drill asserts. A write that fails is the client's error, named by
// key: a git that cannot configure its own repository cannot be
// trusted to fetch from it either.
func hardenGitDir(gitDir string) error {
	for _, kv := range noAutoGC {
		cmd := exec.Command("git", "--git-dir", gitDir, "config", "--local", kv[0], kv[1])
		cmd.Env = withoutGitConfigSelection(os.Environ())
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("harden %s: git config: %w: %s", kv[0], err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

// withoutGitConfigSelection drops GIT_CONFIG from an environment. The
// variable selects the file `git config` reads and writes: an
// unqualified write under it lands in whatever file the operator
// named, and `--local` under it refuses ("only one config file at a
// time") rather than overriding it (review finding on #232). The
// hardening therefore names its target explicitly AND runs without
// the variable, so the repository's own config is the only file it
// touches and a file the operator selected is never mutated by Seed.
// internal/verdict carries the same filter for its workspace clone.
func withoutGitConfigSelection(env []string) []string {
	out := make([]string, 0, len(env))
	for _, kv := range env {
		if strings.HasPrefix(kv, "GIT_CONFIG=") {
			continue
		}
		out = append(out, kv)
	}
	return out
}

func runGit(gitDir string, args ...string) (string, error) {
	full := args
	if gitDir != "" {
		full = append([]string{"--git-dir", gitDir}, args...)
	}
	cmd := exec.Command("git", full...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

const localTracking = "refs/seed/fetched"

// headKey is the cache file for this remote+ref.
func (c *Client) headKey() string {
	sum := sha256.Sum256([]byte(c.Remote + "\x00" + c.Ref))
	return filepath.Join(c.cacheDir, hex.EncodeToString(sum[:16]))
}

// RecordVerifiedHead persists commit as the newest verified remote
// head: the caller's statement that it fetched and fully verified this
// tip. From then on Fetch refuses anything that regresses it (the
// monotonic-head rule), including later in the same invocation, before
// AppendLoop's own persistence kicks in (plans/os-895bf828.md step 1).
func (c *Client) RecordVerifiedHead(commit string) error {
	return c.persistHead(commit)
}

// PersistedHead returns the last verified remote commit, if any.
func (c *Client) PersistedHead() (string, bool, error) {
	b, err := os.ReadFile(c.headKey())
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return strings.TrimSpace(string(b)), true, nil
}

func (c *Client) persistHead(commit string) error {
	tmp := c.headKey() + ".tmp"
	if err := os.WriteFile(tmp, []byte(commit+"\n"), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, c.headKey())
}

// Fetch pulls the remote ref into the local tracking ref and enforces the
// monotonic-head rule: a fetched tip that does not contain the persisted
// verified head refuses with ErrHeadRegression naming both commits. An
// absent remote ref yields an empty commit id (a fresh ledger).
func (c *Client) Fetch() (commit string, err error) {
	out, lsErr := runGit(c.gitDir, "ls-remote", c.Remote, c.Ref)
	if lsErr != nil {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, lsErr)
	}
	fields := strings.Fields(out)
	if len(fields) == 0 {
		// An absent ref is a fresh ledger only for a client that never
		// verified a head. After a verified head, a vanished ref is the
		// deepest possible regression: refusing here keeps AppendLoop from
		// pushing a brand-new root over deleted history (#86 review).
		persisted, have, err := c.PersistedHead()
		if err != nil {
			return "", err
		}
		if have {
			return "", fmt.Errorf("%w: persisted %.12s, remote ref vanished", ErrHeadRegression, persisted)
		}
		return "", nil
	}
	tip := fields[0]
	if _, err := runGit(c.gitDir, "fetch", "-q", "--force", c.Remote, c.Ref+":"+localTracking); err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	persisted, have, err := c.PersistedHead()
	if err != nil {
		return "", err
	}
	if have && persisted != tip {
		if _, err := runGit(c.gitDir, "merge-base", "--is-ancestor", persisted, tip); err != nil {
			return "", fmt.Errorf("%w: persisted %.12s, fetched %.12s", ErrHeadRegression, persisted, tip)
		}
	}
	return tip, nil
}

// Materialize extracts the fetched ledger tree into dir (which must be
// empty or absent). An empty commit id materializes an empty dir.
func (c *Client) Materialize(commit, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if commit == "" {
		return nil
	}
	archive := exec.Command("git", "--git-dir", c.gitDir, "archive", commit)
	untar := exec.Command("tar", "-x", "-C", dir)
	pipe, err := archive.StdoutPipe()
	if err != nil {
		return err
	}
	untar.Stdin = pipe
	if err := untar.Start(); err != nil {
		return err
	}
	if err := archive.Run(); err != nil {
		return fmt.Errorf("git archive %.12s: %w", commit, err)
	}
	return untar.Wait()
}

// CommitAndPush commits dir's tree on top of parent and pushes it to the
// remote ref. A non-fast-forward rejection returns ErrNonFastForward;
// other push failures return ErrUnavailable.
func (c *Client) CommitAndPush(dir, parent, message string) (string, error) {
	if _, err := runGit(c.gitDir, "-c", "core.bare=false", "--work-tree", dir, "add", "-A"); err != nil {
		return "", err
	}
	tree, err := runGit(c.gitDir, "write-tree")
	if err != nil {
		return "", err
	}
	commitArgs := []string{"commit-tree", strings.TrimSpace(tree), "-m", message}
	if parent != "" {
		commitArgs = append(commitArgs, "-p", parent)
	}
	cmd := exec.Command("git", append([]string{"--git-dir", c.gitDir,
		"-c", "user.name=seed-ledger", "-c", "user.email=ledger@seed"}, commitArgs...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("commit-tree: %w: %s", err, out)
	}
	commit := strings.TrimSpace(string(out))
	pushOut, err := runGit(c.gitDir, "push", "--porcelain", c.Remote, commit+":"+c.Ref)
	if err != nil {
		combined := pushOut + " " + err.Error()
		// Only a lost race retries, and races have specific shapes: the
		// stale-parent rejection (reasons "non-fast-forward", "fetch
		// first", "stale info"), server-side receive contention ("failed
		// to lock", and its update-phase shape "failed to update ref",
		// which a rival landing between advertisement and update produces
		// on the receiving side), and mid-push ref-lock contention
		// ("cannot lock ref ... but expected") when the other writer
		// lands between our fetch and our update. Any other rejection
		// (a pre-receive policy hook declining, say) is a refusal to
		// surface, never to retry (#86 review; the update-phase shape
		// was observed in the 2.2 CLI race drill).
		race := strings.Contains(combined, "non-fast-forward") ||
			strings.Contains(combined, "fetch first") ||
			strings.Contains(combined, "stale info") ||
			strings.Contains(combined, "failed to lock") ||
			strings.Contains(combined, "failed to update ref") ||
			(strings.Contains(combined, "cannot lock ref") && strings.Contains(combined, "but expected"))
		if race {
			return "", ErrNonFastForward
		}
		if strings.Contains(combined, "[remote rejected]") || strings.Contains(combined, "[rejected]") {
			return "", fmt.Errorf("%w: %s", ErrRemoteRejected, strings.TrimSpace(pushOut))
		}
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return commit, nil
}
