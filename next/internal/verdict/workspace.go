// The verifier workspace and the profiled runner (plans/os-f6d2c267.md;
// SEED-NEXT.md Part II §8 "clean per-run workspace" and §6 "the
// verifier executes specs in a sandbox with declared, minimal
// capability"; conformance III.G row 4). The workspace is a detached
// local clone with the origin remote removed, deliberately not a git
// worktree: a worktree checkout shares the parent repository's refs and
// object store through its .git link, handing any hostile spec command
// git update-ref reach back into the host repo (review finding on the
// plan). The runner is an interface-shaped seam: v0 ships the exec
// profile, and a namespaced or containerized profile slots in at the
// executor-adapter seam (build plan Phase 7 item 3) without touching
// verdict logic.

package verdict

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/shaunlmason/open-seed/next/internal/artifact"
)

// Workspace is one verdict run's isolated checkout: root holds the
// clone (repo/), a scratch home (home/), and a scratch tmp (tmp/), all
// removed together by Cleanup whatever the run's outcome.
type Workspace struct {
	root string
	// Repo is the checkout directory commands run in.
	Repo string
	home string
	tmp  string
}

// NewWorkspace clones repoDir at head into a fresh per-run root. The
// clone is detached at exactly the submission head and its origin
// remote is removed, so nothing in the workspace names a path back to
// the parent repository.
func NewWorkspace(repoDir, head string) (*Workspace, error) {
	root, err := os.MkdirTemp("", "seed-verdict-*")
	if err != nil {
		return nil, fmt.Errorf("verdict workspace: %w", err)
	}
	ws := &Workspace{root: root, Repo: filepath.Join(root, "repo"), home: filepath.Join(root, "home"), tmp: filepath.Join(root, "tmp")}
	for _, d := range []string{ws.home, ws.tmp} {
		if err := os.Mkdir(d, 0o755); err != nil {
			ws.Cleanup()
			return nil, fmt.Errorf("verdict workspace: %w", err)
		}
	}
	steps := [][]string{
		// --no-hardlinks: a same-filesystem local clone otherwise
		// hard-links loose object files, so a hostile spec command
		// overwriting one through the shared inode would corrupt the
		// parent repository's object store despite the removed origin
		// (review finding on the task PR). Copied objects make the
		// isolation real; the drill corrupts every clone-side object
		// and asserts the parent still verifies.
		{"clone", "--quiet", "--no-checkout", "--no-hardlinks", repoDir, ws.Repo},
		{"-C", ws.Repo, "checkout", "--quiet", "--detach", head},
		{"-C", ws.Repo, "remote", "remove", "origin"},
	}
	for _, args := range steps {
		cmd := exec.Command("git", args...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			ws.Cleanup()
			return nil, fmt.Errorf("verdict workspace: git %s: %v: %s", args[0], err, strings.TrimSpace(stderr.String()))
		}
	}
	return ws, nil
}

// Cleanup removes the whole per-run root. It is safe to call twice and
// runs on the failure path exactly as on success: cleanup fires pass
// or fail (III.G row 4).
func (w *Workspace) Cleanup() {
	if w.root != "" {
		os.RemoveAll(w.root)
	}
}

// git runs a read-side git command inside the workspace clone and
// returns its stdout. This is the verifier's own reading, never a spec
// command: it runs with the invoking environment, not the profile.
func (w *Workspace) git(args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", w.Repo}, args...)...)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %v: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return out.String(), nil
}

// RunnerProfile names a declared capability profile; the receipt
// records which profile its transcripts ran under.
const ExecProfile = "exec"

// Runner executes spec commands under the declared v0 exec profile:
// scrubbed environment (explicit minimal PATH; HOME and TMPDIR inside
// the per-run root; nothing inherited), per-command wall-clock timeout
// with process-group kill, network honestly unrestricted.
type Runner struct {
	// Timeout bounds each command; zero means DefaultTimeout.
	Timeout time.Duration
}

// DefaultTimeout is the per-command wall-clock bound.
const DefaultTimeout = 10 * time.Minute

// Transcript is one executed command's receipt entry: the command, its
// exit (negative when the profile killed it at the timeout), and the
// digest and byte count of its combined output — never inline bytes,
// so receipts stay bounded.
type Transcript struct {
	Cmd          string `json:"cmd"`
	Exit         int    `json:"exit"`
	OutputSHA256 string `json:"output_sha256"`
	OutputBytes  int    `json:"output_bytes"`
}

// Run executes one spec command in the workspace under the exec
// profile and returns its transcript. The command's exit never aborts
// the run: a red check is a fact the receipt records and the render
// rule consumes.
func (r Runner) Run(ws *Workspace, command string) Transcript {
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = ws.Repo
	cmd.Env = []string{
		"PATH=/usr/local/bin:/usr/bin:/bin",
		"HOME=" + ws.home,
		"TMPDIR=" + ws.tmp,
		"GIT_CONFIG_NOSYSTEM=1",
		"LANG=C",
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		// Kill the whole process group so a hanging pipeline dies with
		// its children, not just the shell.
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	out, err := cmd.CombinedOutput()
	t := Transcript{Cmd: command, OutputSHA256: artifact.Digest(out), OutputBytes: len(out)}
	switch {
	case err == nil:
		t.Exit = 0
	case ctx.Err() != nil:
		t.Exit = -1
	default:
		if ee, ok := err.(*exec.ExitError); ok {
			t.Exit = ee.ExitCode()
		} else {
			t.Exit = -1
		}
	}
	return t
}
