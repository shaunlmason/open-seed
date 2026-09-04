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
	"archive/tar"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/shaunlmason/open-seed/next/internal/event"
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
	proposer Proposer
}

// Proposer lands signed records on the remote ref through a third
// party instead of the client's own push: the admission service under
// the enforced-forge-hosted posture (plans/os-5c8a312c.md D3). The
// records arrive already linked and signed; a proposer relays, never
// re-signs.
type Proposer interface {
	Propose(ref string, recs []*event.Record) (*Result, error)
}

// Refusal is a proposal the service refused with the boundary's own
// envelope. Exit, Code and Message are the service's verbatim, so a
// proposer renders exactly the code the hook posture would have
// produced for the same record; it unwraps to ErrRemoteRejected for
// callers that only ask whether the remote said no.
type Refusal struct {
	Exit     int
	Code     string
	Message  string
	Position *string
}

func (r *Refusal) Error() string { return fmt.Sprintf("%s: %s", r.Code, r.Message) }
func (r *Refusal) Unwrap() error { return ErrRemoteRejected }

// WithProposer makes the append loop propose instead of push. Every
// step before the last — fetch, materialize, verify, re-link, re-sign,
// validate — is unchanged: the cooperative half is what every posture
// keeps, and only the write moves to the service.
func (c *Client) WithProposer(p Proposer) *Client {
	c.proposer = p
	return c
}

// GitDir is the client's private git dir: the objects it fetched and
// the commits it built, which the admission service judges in place.
func (c *Client) GitDir() string { return c.gitDir }

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

// hardenGitDir ensures noAutoGC is set in the git dir's own config on
// every construction, not only at init: a state dir an older build
// created is hardened the first time a new build opens it. The write
// is in-process, in git's own file format, and touches only the
// repository's config file: three `git config --local` processes per
// construction were the largest single source of spawns in the CLI
// suite, and a spawn is dear on Windows (next/spec/platform.md). A key
// already carrying its value is left alone, so the file never grows
// with repeated opens; a file that cannot be read or written is the
// client's error, named by key. GIT_CONFIG, which selects the file a
// `git config` invocation reads and writes (review finding on #232),
// no longer enters into it: no git process runs here.
func hardenGitDir(gitDir string) error {
	return ensureConfig(gitDir, noAutoGC)
}

// ensureConfig appends every key of kvs whose value the git dir's
// config does not already carry, grouped under section headers in the
// format git itself writes. Keys are `section.key`; a subsection is
// not a shape this needs. For a single-valued key the last occurrence
// wins, so a differing value is superseded by appending, never by
// rewriting what an operator or git wrote.
func ensureConfig(gitDir string, kvs [][2]string) error {
	path := filepath.Join(gitDir, "config")
	existing, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("harden: read %s: %w", path, err)
	}
	have := configValues(existing)
	var buf strings.Builder
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		buf.WriteByte('\n')
	}
	lastSection := ""
	for _, kv := range kvs {
		section, key, ok := strings.Cut(kv[0], ".")
		if !ok || strings.Contains(key, ".") {
			return fmt.Errorf("harden %s: a key is section.key", kv[0])
		}
		if have[strings.ToLower(section)+"."+strings.ToLower(key)] == kv[1] {
			continue
		}
		if section != lastSection {
			fmt.Fprintf(&buf, "[%s]\n", section)
			lastSection = section
		}
		fmt.Fprintf(&buf, "\t%s = %s\n", key, kv[1])
	}
	if lastSection == "" {
		return nil
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return fmt.Errorf("harden: open %s: %w", path, err)
	}
	if _, err := f.WriteString(buf.String()); err != nil {
		f.Close()
		return fmt.Errorf("harden: write %s: %w", path, err)
	}
	return f.Close()
}

// configValues reads a git config file far enough to answer which
// `section.key` (or `section.subsection.key`) holds which value:
// section and key names fold to lower case, a subsection is kept
// verbatim, and the last occurrence of a key wins, as in git. Includes
// are not followed; a value that only an include supplies reads as
// absent, and the explicit append that follows still wins in git's own
// resolution, since it comes later in the file.
func configValues(b []byte) map[string]string {
	vals := map[string]string{}
	section := ""
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' || line[0] == ';' {
			continue
		}
		if line[0] == '[' {
			end := strings.IndexByte(line, ']')
			if end < 0 {
				section = ""
				continue
			}
			name, sub, hasSub := strings.Cut(line[1:end], " ")
			section = strings.ToLower(strings.TrimSpace(name))
			if hasSub {
				section += "." + strings.Trim(strings.TrimSpace(sub), "\"")
			}
			continue
		}
		if section == "" {
			continue
		}
		key, value, _ := strings.Cut(line, "=")
		key = strings.ToLower(strings.TrimSpace(key))
		if key == "" {
			continue
		}
		// A trailing comment ends the value; quotes around it are git's
		// own way of keeping leading or trailing blanks.
		value = strings.TrimSpace(value)
		if strings.HasPrefix(value, "\"") {
			if end := strings.IndexByte(value[1:], '"'); end >= 0 {
				value = value[1 : 1+end]
			}
		} else if i := strings.IndexAny(value, "#;"); i >= 0 {
			value = strings.TrimSpace(value[:i])
		}
		vals[section+"."+key] = value
	}
	return vals
}

// withoutGitConfigSelection drops GIT_CONFIG from an environment. The
// variable selects the file `git config` reads and writes: an
// unqualified write under it lands in whatever file the operator
// named, and `--local` under it refuses ("only one config file at a
// time") rather than overriding it (review finding on #232). The
// hardening no longer runs git at all (ensureConfig writes the
// repository's own file directly), so this filter now serves the
// drill that reads the hardened values back with --local under a
// planted GIT_CONFIG. internal/verdict carries the same filter for
// its workspace clone.
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
	// The ledger is LF-only (next/spec/platform.md): git is told so on
	// every call, so a checkout or archive on a platform whose
	// default converts line endings never hands the verifier bytes
	// the signatures do not cover.
	full := append([]string{"-c", "core.autocrlf=false", "-c", "core.eol=lf"}, args...)
	if gitDir != "" {
		full = append([]string{"--git-dir", gitDir, "-c", "core.autocrlf=false", "-c", "core.eol=lf"}, args...)
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
//
// The common path is one git process: the fetch itself, with the tip
// read back from the tracking ref it wrote. The ls-remote that used to
// precede every fetch is spent only when the fetch refuses, to tell an
// absent ref (a fresh ledger) from an unreachable remote. Each spawn is
// cheap on Linux and dear on Windows, and every append pays this path
// at least once (next/spec/platform.md, the cmd/seed residual).
func (c *Client) Fetch() (commit string, err error) {
	if _, fetchErr := runGit(c.gitDir, "fetch", "-q", "--force", c.Remote, c.Ref+":"+localTracking); fetchErr != nil {
		out, lsErr := runGit(c.gitDir, "ls-remote", c.Remote, c.Ref)
		if lsErr != nil {
			return "", fmt.Errorf("%w: %v", ErrUnavailable, lsErr)
		}
		if len(strings.Fields(out)) != 0 {
			return "", fmt.Errorf("%w: %v", ErrUnavailable, fetchErr)
		}
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
	tip, err := c.trackingTip()
	if err != nil {
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

// trackingTip reads the commit the tracking ref names. A fetch writes
// the ref loose, so the common path is one file read and no process;
// a ref stored any other way (a packed or reftable store) is resolved
// by git itself.
func (c *Client) trackingTip() (string, error) {
	if b, err := os.ReadFile(filepath.Join(c.gitDir, filepath.FromSlash(localTracking))); err == nil {
		if id := strings.TrimSpace(string(b)); isObjectID(id) {
			return id, nil
		}
	}
	out, err := runGit(c.gitDir, "rev-parse", "--verify", localTracking)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// isObjectID reports whether s is a full hex object id (SHA-1 or SHA-256).
func isObjectID(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	for _, r := range s {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

// Materialize extracts the fetched ledger tree into dir (which must be
// empty or absent). An empty commit id materializes an empty dir. The
// archive git writes is read by this process: the tar that used to
// unpack it was a second spawn per materialization, and every append
// materializes at least once.
func (c *Client) Materialize(commit, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if commit == "" {
		return nil
	}
	archive := exec.Command("git", "-c", "core.autocrlf=false", "-c", "core.eol=lf", "--git-dir", c.gitDir, "archive", "--format=tar", commit)
	var stderr strings.Builder
	archive.Stderr = &stderr
	pipe, err := archive.StdoutPipe()
	if err != nil {
		return err
	}
	if err := archive.Start(); err != nil {
		return err
	}
	extractErr := untar(pipe, dir)
	// Drain what is left so git can exit whatever the extraction did.
	io.Copy(io.Discard, pipe)
	if err := archive.Wait(); err != nil {
		return fmt.Errorf("git archive %.12s: %w: %s", commit, err, strings.TrimSpace(stderr.String()))
	}
	if extractErr != nil {
		return fmt.Errorf("git archive %.12s: %w", commit, extractErr)
	}
	return nil
}

// untar unpacks a tar stream under dir: directories, regular files
// with their permission bits, and symbolic links, which is what a git
// tree holds. An entry that would land outside dir is refused by name.
func untar(r io.Reader, dir string) error {
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag == tar.TypeXGlobalHeader {
			continue
		}
		target := filepath.Join(dir, filepath.FromSlash(hdr.Name))
		if rel, err := filepath.Rel(dir, target); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fmt.Errorf("entry %q escapes the target directory", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, hdr.FileInfo().Mode().Perm())
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return err
			}
		default:
			return fmt.Errorf("entry %q has unsupported type %q", hdr.Name, hdr.Typeflag)
		}
	}
}

// CommitAndPush commits dir's tree on top of parent and pushes it to the
// remote ref. A non-fast-forward rejection returns ErrNonFastForward;
// other push failures return ErrUnavailable.
func (c *Client) CommitAndPush(dir, parent, message string) (string, error) {
	commit, err := c.Commit(dir, parent, message)
	if err != nil {
		return "", err
	}
	if err := c.Push(commit); err != nil {
		return "", err
	}
	return commit, nil
}

// Commit writes dir's tree as a commit on top of parent in the client's
// git dir and returns its id, pushing nothing: the admission service
// judges a candidate this way before deciding whether it is pushed at
// all (plans/os-5c8a312c.md D1).
func (c *Client) Commit(dir, parent, message string) (string, error) {
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
	cmd := exec.Command("git", append([]string{"--git-dir", c.gitDir, "-c", "core.autocrlf=false", "-c", "core.eol=lf",
		"-c", "user.name=seed-ledger", "-c", "user.email=ledger@seed"}, commitArgs...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("commit-tree: %w: %s", err, out)
	}
	return strings.TrimSpace(string(out)), nil
}

// Push updates the remote ref to commit, fast-forward only. A lost race
// returns ErrNonFastForward, a policy rejection ErrRemoteRejected, and
// anything else ErrUnavailable.
func (c *Client) Push(commit string) error {
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
			return ErrNonFastForward
		}
		if strings.Contains(combined, "[remote rejected]") || strings.Contains(combined, "[rejected]") {
			return fmt.Errorf("%w: %s", ErrRemoteRejected, strings.TrimSpace(pushOut))
		}
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return nil
}
