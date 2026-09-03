package main

// The handbook's commands run (plans/os-16e55c11.md D2, AC2): every
// fenced `seed` command names a dispatchable group and subverb, so a
// renamed verb fails the handbook rather than the reader; and the
// flagship read commands run live against the fixture, so a renamed flag
// on them fails too.

import (
	"bufio"
	"os"
	"strings"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/envelope"
)

// knownGroups is the top-level verb set the dispatch switch routes.
var knownGroups = map[string]bool{
	"version": true, "init": true, "ledger": true, "project": true,
	"situation": true, "obs": true, "plan": true, "verdict": true,
	"seal": true, "offer": true, "budget": true, "claim": true,
	"escalation": true, "decision": true, "submission": true, "merge": true,
	"reconcile": true, "maintain": true, "lane": true, "message": true,
	"doctor": true, "protections": true, "perf": true, "import": true,
	"preseed": true, "run": true, "eval": true, "knowledge": true,
	"flywheel": true, "trajectory": true, "docs": true, "simulate": true,
}

// handbookCommands extracts the token lists of every `seed` command in a
// fenced code block.
func handbookCommands(t *testing.T) [][]string {
	t.Helper()
	f, err := os.Open("../../docs/handbook.md")
	if err != nil {
		t.Fatalf("read handbook: %v", err)
	}
	defer f.Close()
	var cmds [][]string
	inFence := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "```") {
			inFence = !inFence
			continue
		}
		if inFence && strings.HasPrefix(strings.TrimSpace(line), "seed ") {
			cmds = append(cmds, strings.Fields(strings.TrimSpace(line))[1:])
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if len(cmds) == 0 {
		t.Fatal("the handbook must carry fenced seed commands")
	}
	return cmds
}

func TestHandbookCommandsAreDispatchable(t *testing.T) {
	for _, toks := range handbookCommands(t) {
		if len(toks) == 0 {
			continue
		}
		group := toks[0]
		if !knownGroups[group] {
			t.Errorf("handbook uses unknown top-level verb %q", group)
			continue
		}
		// Probe the group and (if present) its subverb: a dispatchable
		// verb never answers "unknown ...". Other failures (a missing
		// ledger) are fine — we are pinning the command surface, not
		// running the deployment.
		probe := []string{group}
		if len(toks) > 1 && !strings.HasPrefix(toks[1], "-") {
			probe = append(probe, toks[1])
		}
		e, _ := runEnv(t, probe...)
		if e.Error != nil && strings.Contains(e.Error.Message, "unknown") {
			t.Errorf("handbook command %v is not dispatchable: %s", probe, e.Error.Message)
		}
	}
}

func TestHandbookFlagshipCommandsRunLive(t *testing.T) {
	// Each flagship command runs against the fixture; a renamed flag on
	// any of them fails here.
	cases := [][]string{
		{"version"},
		{"docs", "check", "--root", "../../.."},
		{"lane", "list", "--lanes", "../../lanes"},
		{"doctor", "--config", "../../fixtures/deployment/seed.json"},
		{"simulate", "--lanes", "../../lanes", "--intents", "1", "--posture", "cooperative", "--work", t.TempDir()},
	}
	for _, args := range cases {
		e, code := runEnv(t, args...)
		if code != envelope.ExitOK || !e.OK {
			t.Errorf("flagship command %v must run, got exit %d: %+v", args, code, e.Error)
		}
	}
}
