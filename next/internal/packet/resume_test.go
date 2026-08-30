package packet_test

// The packet-resume drill (plans/os-b07b0f59.md; the Phase 5 exit
// criterion; conformance III.F "a fresh executor completes a killed
// executor's contract from the packet alone, including not re-trying
// recorded dead ends"). Executor A is a scripted harness: it does the
// work in a real git repository, records one verified decision, one
// asserted decision, and one dead end, pushes its commit, and is
// force-reaped leaving only the packet. Executor B is a deterministic
// function whose ONLY input is the packet: a fresh clone at the
// packet's anchors, no transcript, no workspace reuse. Sufficiency is
// the assertion, not a vibe.

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/packet"
)

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=drill", "GIT_AUTHOR_EMAIL=drill@example.invalid",
		"GIT_COMMITTER_NAME=drill", "GIT_COMMITTER_EMAIL=drill@example.invalid")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// resumeAction is one line of B's action log: what B did, mechanically.
type resumeAction struct {
	Approach string
	Done     string
}

// executorB is a deterministic function of the packet alone: it clones
// fresh at the packet's anchors, skips every recorded dead end, works
// through the acceptance list, and returns its action log plus the
// artifact bytes it can reproduce from the anchors.
func executorB(t *testing.T, remote string, p *packet.Packet) ([]resumeAction, map[string][]byte) {
	t.Helper()
	clone := t.TempDir()
	git(t, ".", "clone", "-q", remote, clone)
	_, head, ok := strings.Cut(p.Base, "..")
	if !ok {
		t.Fatalf("packet base is not a range: %s", p.Base)
	}
	git(t, clone, "checkout", "-q", head)

	// B's candidate approaches: the recorded dead ends are excluded
	// before anything runs — that is what findings are FOR.
	dead := map[string]bool{}
	for _, f := range p.Findings {
		dead[f.Tried] = true
	}
	var log []resumeAction
	for _, approach := range []string{"approach-X", "approach-Y"} {
		if dead[approach] {
			continue
		}
		for _, item := range p.Acceptance {
			log = append(log, resumeAction{Approach: approach, Done: item})
		}
		break
	}

	// The refs reproduce A's artifacts from the anchors alone.
	artifacts := map[string][]byte{}
	for _, r := range p.Refs {
		path, anchor, ok := strings.Cut(r, " @ ")
		if !ok || strings.Contains(anchor, "..") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(clone, path))
		if err != nil {
			t.Fatalf("packet ref %q does not resolve in the fresh clone: %v", r, err)
		}
		artifacts[path] = b
	}
	return log, artifacts
}

func TestPacketResumeDrill(t *testing.T) {
	// A real code remote, distinct from the coordination ledger.
	remote := t.TempDir()
	git(t, ".", "init", "-q", "--bare", remote)
	workA := t.TempDir()
	git(t, ".", "clone", "-q", remote, workA)

	// Executor A: base commit (the merge-base), then the work commit.
	if err := os.WriteFile(filepath.Join(workA, "README.md"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, workA, "add", ".")
	git(t, workA, "commit", "-q", "-m", "base")
	mergeBase := git(t, workA, "rev-parse", "HEAD")
	artifact := []byte("executor A's result\n")
	if err := os.WriteFile(filepath.Join(workA, "artifact.txt"), artifact, 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, workA, "add", ".")
	git(t, workA, "commit", "-q", "-m", "work")
	head := git(t, workA, "rev-parse", "HEAD")
	git(t, workA, "push", "-q", "origin", "HEAD:refs/heads/main")

	// A is force-reaped; the packet is everything it leaves behind.
	reapPacket, err := json.Marshal(map[string]any{
		"acceptance": []string{"artifact.txt carries executor A's result"},
		"decisions": []map[string]string{
			{"decision": "the artifact lives at the repository root", "basis": "verified"},
			{"decision": "no consumer reads it yet", "basis": "asserted"},
		},
		"base":     mergeBase + ".." + head,
		"refs":     []string{"artifact.txt @ " + head},
		"findings": []map[string]string{{"tried": "approach-X", "outcome": "fails: the root layout rejects it"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	p, perr := packet.Parse("c-resume", reapPacket)
	if perr != nil {
		t.Fatalf("A's reap packet must be shape-valid: %v", perr)
	}

	// Executor A's workspace is gone: B gets the packet and nothing
	// else.
	if err := os.RemoveAll(workA); err != nil {
		t.Fatal(err)
	}
	log, artifacts := executorB(t, remote, p)

	// B never re-tries the recorded dead end.
	for _, a := range log {
		if a.Approach == "approach-X" {
			t.Fatalf("B re-tried the recorded dead end: %+v", log)
		}
	}
	// B completed the acceptance list.
	done := map[string]bool{}
	for _, a := range log {
		done[a.Done] = true
	}
	for _, item := range p.Acceptance {
		if !done[item] {
			t.Fatalf("B must complete the acceptance list from the packet alone; missing %q in %+v", item, log)
		}
	}
	// B's fresh clone plus the packet's refs reproduce A's artifacts.
	got, ok := artifacts["artifact.txt"]
	if !ok || string(got) != string(artifact) {
		t.Fatalf("B must reproduce A's artifact from the commit anchors, got %q", got)
	}
}
