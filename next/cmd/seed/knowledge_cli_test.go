package main

// The knowledge verbs end to end (plans/os-f30ee0d3.md AC7): the four
// subverbs against a real ledger with the fence and the hypothesis id
// derived, refusing at usage what the boundary would refuse, and the
// projection and the report carrying the stages.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/curation"
	"github.com/shaunlmason/open-seed/next/internal/version"
)

func TestKnowledgeVerbsDriveTheStages(t *testing.T) {
	ld, _, _, _, _, priv, rootKey, keys, _ := offerLedger(t)
	curatorKey, curatorPub, curatorFP := writeWorkerKey(t, 26)
	observerKey, observerPub, observerFP := writeWorkerKey(t, 27)
	for _, step := range [][]string{
		{"actor.enrolled", curatorFP, fmt.Sprintf(`{"key": %q, "kind": "agent", "name": "curator"}`, curatorPub)},
		{"actor.granted", curatorFP, `{"capability": "curate"}`},
		{"actor.enrolled", observerFP, fmt.Sprintf(`{"key": %q, "kind": "agent", "name": "observer"}`, observerPub)},
		{"actor.granted", observerFP, `{"capability": "observer"}`},
	} {
		if e, code := runEnv(t, "ledger", "append", "--ledger", ld, "--key", priv,
			"--verb", step[0], "--subject", step[1], "--payload", step[2]); code != 0 {
			t.Fatalf("%s: %d %+v", step[0], code, e)
		}
	}
	open := func(subject string) {
		rawAppendAt(t, ld, rootKey, version.Seed1, "intent.filed", subject, `{"intent": "drill", "tier": "trivial", "budget": "small", "routing": "core"}`)
		rawAppendAt(t, ld, rootKey, version.Seed1, "contract.specified", subject, `{"acceptance": {"ref": "accept.md @ 0123456", "executable": false}}`)
	}
	open("c-1")
	open("c-2")
	rawAppendAt(t, ld, workerRawKey(22), version.Seed1, "claim.taken", "c-1", `{}`)
	deadend := func(key, subject string) (ledgerEnv, int) {
		return runEnv(t, "knowledge", "deadend", "--ledger", ld, "--key", key, "--subject", subject,
			"--tried", "retrying the fetch", "--outcome", "the mirror timed out", "--condition", "the mirror was cold", "--environment", "ci-runner/v0")
	}
	if e, code := deadend(keys["workerA"], "c-1"); code != 0 || e.Result["subject"] != "c-1" {
		t.Fatalf("the holder records a dead end inside its window, the fence derived: %d %+v", code, e)
	}
	if e, code := deadend(keys["workerA"], "c-2"); code != 3 || e.Error == nil || e.Error.Code != "invalid_transition" {
		t.Fatalf("no window on c-2 refuses before anything is signed: %d %+v", code, e.Error)
	}
	if e, code := deadend(keys["workerB"], "c-1"); code == 0 || e.Error == nil || !strings.Contains(e.Error.Message, "holder") {
		t.Fatalf("a non-holder's dead end refuses naming the holder: %d %+v", code, e.Error)
	}
	if e, code := runEnv(t, "knowledge", "deadend", "--ledger", ld, "--key", keys["workerA"], "--subject", "c-1",
		"--tried", "x", "--outcome", "y", "--condition", "z"); code != 64 || e.Error == nil || !strings.Contains(e.Error.Message, "--environment") {
		t.Fatalf("a missing field refuses at usage naming it: %d %+v", code, e.Error)
	}
	rawAppendAt(t, ld, workerRawKey(22), version.Seed1, "claim.taken", "c-2", `{}`)
	if e, code := deadend(keys["workerA"], "c-2"); code != 0 {
		t.Fatalf("the second dead end: %d %+v", code, e)
	}

	// The fold names the positions the proposal cites.
	show := func() map[string]any {
		t.Helper()
		e, code := runEnv(t, "knowledge", "show", "--ledger", ld)
		if code != 0 {
			t.Fatalf("knowledge show: %d %+v", code, e)
		}
		return e.Result
	}
	positionOf := func(view map[string]any, contract string) int {
		t.Helper()
		ends, _ := view["dead_ends"].(map[string]any)
		list, _ := ends[contract].([]any)
		if len(list) == 0 {
			t.Fatalf("no dead end on %s: %+v", contract, view)
		}
		first, _ := list[0].(map[string]any)
		pos, _ := first["position"].(float64)
		return int(pos)
	}
	view := show()
	p1, p2 := positionOf(view, "c-1"), positionOf(view, "c-2")
	if stages, _ := view["stages"].(map[string]any); stages["observations"] != 2.0 {
		t.Fatalf("two observations stand: %+v", view["stages"])
	}

	claim := "retry the fetch once when the mirror is cold"
	propose := func(key string, support ...string) (ledgerEnv, int) {
		args := []string{"knowledge", "propose", "--ledger", ld, "--key", key, "--claim", claim, "--applies-when", "the mirror is cold",
			"--provenance", "plans/os-f30ee0d3.md @ 0123456", "--exception", "a warm mirror"}
		for _, s := range support {
			args = append(args, "--support", s)
		}
		return runEnv(t, args...)
	}
	if e, code := propose(keys["workerA"], fmt.Sprintf("c-1@%d", p1)); code != 64 || e.Error == nil || !strings.Contains(e.Error.Message, "2 --support") {
		t.Fatalf("one citation refuses at usage naming the floor: %d %+v", code, e.Error)
	}
	if e, code := propose(keys["workerA"], fmt.Sprintf("c-1@%d", p1), fmt.Sprintf("c-2@%d", p2)); code != 14 || e.Error == nil || e.Error.Code != "out_of_grant" {
		t.Fatalf("a claim key cannot propose: %d %+v", code, e.Error)
	}
	id := curation.HypothesisID(claim)
	e, code := propose(curatorKey, fmt.Sprintf("c-1@%d", p1), fmt.Sprintf("c-2@%d", p2))
	if code != 0 || e.Result["hypothesis"] != id {
		t.Fatalf("the curator proposes on the derived subject: %d %+v", code, e)
	}
	if e, code := propose(curatorKey, fmt.Sprintf("c-1@%d", p1), fmt.Sprintf("c-2@%d", p2)); code == 0 || e.Error == nil || !strings.Contains(e.Error.Message, "proposed at position") {
		t.Fatalf("a re-proposal refuses as a duplicate: %d %+v", code, e.Error)
	}
	view = show()
	hyps, _ := view["hypotheses"].([]any)
	if len(hyps) != 1 {
		t.Fatalf("one hypothesis stands: %+v", view)
	}
	hyp, _ := hyps[0].(map[string]any)
	hpos, _ := hyp["position"].(float64)
	if hyp["id"] != id || hyp["stage"] != "proposed" {
		t.Fatalf("the hypothesis is proposed: %+v", hyp)
	}

	zeros := strings.Repeat("0", 40)
	promote := func(key, lesson, hypothesis string) (ledgerEnv, int) {
		return runEnv(t, "knowledge", "promote", "--ledger", ld, "--key", key, "--lesson", lesson, "--hypothesis", hypothesis, "--pr", "pr/7 @ "+zeros)
	}
	lesson := curation.LessonsDir + "/retry-when-cold.md @ " + zeros
	if e, code := promote(observerKey, curation.LessonsDir+"/retry-when-cold.md", fmt.Sprintf("%s@%d", id, int(hpos))); code != 64 || e.Error == nil {
		t.Fatalf("a bare lesson path refuses at usage: %d %+v", code, e.Error)
	}
	if e, code := promote(observerKey, lesson, fmt.Sprintf("%s@%d", id, int(hpos)+1)); code == 0 || e.Error == nil || !strings.Contains(e.Error.Message, "admitted hypothesis") {
		t.Fatalf("citing a position that is no hypothesis refuses: %d %+v", code, e.Error)
	}
	if e, code := promote(curatorKey, lesson, fmt.Sprintf("%s@%d", id, int(hpos))); code != 14 {
		t.Fatalf("the curator cannot promote: %d %+v", code, e.Error)
	}
	if e, code := promote(observerKey, lesson, fmt.Sprintf("%s@%d", id, int(hpos))); code != 0 || e.Result["subject"] != id {
		t.Fatalf("the observer promotes the admitted hypothesis: %d %+v", code, e)
	}
	view = show()
	stages, _ := view["stages"].(map[string]any)
	if stages["observations"] != 2.0 || stages["hypotheses"] != 1.0 || stages["promoted"] != 1.0 || stages["lessons"] != 1.0 || stages["unbound"] != 0.0 {
		t.Fatalf("the stages count: %+v", stages)
	}

	// The projection publishes the same view and the report counts.
	out := filepath.Join(t.TempDir(), "out")
	unlockForCleanup(t, out)
	if e, code := runEnv(t, "project", "rebuild", "--ledger", ld, "--out", out); code != 0 {
		t.Fatalf("project rebuild: %d %+v", code, e)
	}
	for _, name := range []string{"knowledge", "report"} {
		cur, code := runEnv(t, "project", "current", "--out", out, "--name", name)
		if code != 0 {
			t.Fatalf("project current %s: %d %+v", name, code, cur)
		}
		b, err := os.ReadFile(filepath.Join(cur.Result["path"].(string), name+".json"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), `"promoted": 1`) || !strings.Contains(string(b), `"lessons": 1`) {
			t.Fatalf("%s carries the stage counts: %s", name, b)
		}
	}
}
