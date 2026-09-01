package main

// The situation drills (plans/os-52d5da3f.md D4): one
// position-stamped envelope carrying the caller's obligations and
// windows; --since is a COMPLETE change report whose discharged list
// lets a prior snapshot be brought forward exactly; and the read
// mutates nothing and journals nothing.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/refusals"
)

type situationEnv struct {
	Actor       string           `json:"actor"`
	Obligations []map[string]any `json:"obligations"`
	Windows     []map[string]any `json:"windows"`
	Since       string           `json:"since"`
	Discharged  []map[string]any `json:"discharged"`
	Unchanged   string           `json:"unchanged"`
}

func situationOf(t *testing.T, args ...string) (ledgerEnv, situationEnv, int) {
	t.Helper()
	e, code := runEnv(t, append([]string{"situation"}, args...)...)
	var s situationEnv
	if e.Result != nil {
		b, err := json.Marshal(e.Result)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(b, &s); err != nil {
			t.Fatal(err)
		}
	}
	return e, s, code
}

func kindsOf(rows []map[string]any) map[string]bool {
	out := map[string]bool{}
	for _, r := range rows {
		out[fmt.Sprint(r["kind"])] = true
	}
	return out
}

func TestSituationRead(t *testing.T) {
	ld, _, _, specCommit, _, priv, _, keys, fps := offerLedger(t)
	offerFile(t, ld, priv, specCommit, "c-1")

	// Before any claim the holder owes nothing and holds no window.
	_, s, code := situationOf(t, "--ledger", ld, "--key", keys["workerA"])
	if code != 0 {
		t.Fatalf("situation: %d", code)
	}
	if len(s.Obligations) != 0 || len(s.Windows) != 0 {
		t.Fatalf("an unclaimed board owes the worker nothing: %+v %+v", s.Obligations, s.Windows)
	}
	if s.Actor != fps["workerA"] {
		t.Fatalf("the response names the reading actor: %q", s.Actor)
	}

	// The claim creates the obligation and the window, and the
	// window carries the fence a lane would otherwise read out of a
	// projection by hand.
	fencePos, err := admitAppend(t, ld, workerRawKey(22), "claim.taken", "c-1", `{}`)
	if err != nil {
		t.Fatal(err)
	}
	e, s, code := situationOf(t, "--ledger", ld, "--key", keys["workerA"])
	if code != 0 || e.Position == nil {
		t.Fatalf("situation after claim: %d %+v", code, e)
	}
	if !kindsOf(s.Obligations)["claim.held"] {
		t.Fatalf("the holder owes a deliberate exit: %+v", s.Obligations)
	}
	if len(s.Windows) != 1 || fmt.Sprint(s.Windows[0]["fence"]) != fmt.Sprintf("%d", fencePos) {
		t.Fatalf("the window carries the active fence: %+v", s.Windows)
	}
	for _, row := range s.Obligations {
		if v, ok := row["discharged_by"].([]any); !ok || len(v) == 0 {
			t.Fatalf("every obligation names what discharges it: %+v", row)
		}
	}

	// A different lane sees its own board, not the holder's.
	_, verifier, code := situationOf(t, "--ledger", ld, "--key", keys["verifier"])
	if code != 0 {
		t.Fatalf("verifier situation: %d", code)
	}
	if kindsOf(verifier.Obligations)["claim.held"] {
		t.Fatalf("the claim is not the verifier's debt: %+v", verifier.Obligations)
	}

	// The read mutates nothing and journals nothing: it is not an
	// admission-boundary attempt.
	before := journalLen(t, ld)
	if _, _, code := situationOf(t, "--ledger", ld, "--key", keys["workerA"]); code != 0 {
		t.Fatalf("repeat read: %d", code)
	}
	if after := journalLen(t, ld); after != before {
		t.Fatalf("a read journals nothing: %d -> %d", before, after)
	}
}

// conformance: --since is a complete change report — applying its
// obligations and discharged list to a prior snapshot must reproduce
// the standing set exactly, which a filtered list alone cannot do.
func TestSituationSinceIsApplicable(t *testing.T) {
	ld, _, _, specCommit, _, priv, _, keys, _ := offerLedger(t)
	offerFile(t, ld, priv, specCommit, "c-1")
	if _, err := admitAppend(t, ld, workerRawKey(22), "claim.taken", "c-1", `{}`); err != nil {
		t.Fatal(err)
	}
	// The snapshot a resuming lane last saw, and the position it
	// stopped at.
	e, before, code := situationOf(t, "--ledger", ld, "--key", keys["workerA"])
	if code != 0 || e.Position == nil {
		t.Fatalf("snapshot: %d %+v", code, e)
	}
	at := *e.Position
	snapshot := kindsOf(before.Obligations)
	if !snapshot["claim.held"] {
		t.Fatalf("the snapshot holds the claim: %+v", before.Obligations)
	}

	// The holder releases: the obligation is discharged, so the delta
	// must SAY it is gone, not merely omit it.
	if _, err := admitAppend(t, ld, workerRawKey(22), "claim.released", "c-1", fmt.Sprintf(
		`{"fence": "%s", "packet": {"acceptance": ["done"], "decisions": [], "base": "0000000000000000000000000000000000000000..0000000000000000000000000000000000000000", "refs": [], "findings": []}}`,
		fmt.Sprint(before.Windows[0]["fence"]))); err != nil {
		t.Fatal(err)
	}
	_, delta, code := situationOf(t, "--ledger", ld, "--key", keys["workerA"], "--since", at)
	if code != 0 {
		t.Fatalf("delta: %d", code)
	}
	if len(delta.Discharged) == 0 {
		t.Fatalf("the delta names the removal: %+v", delta)
	}
	// Apply the response to the snapshot: add what arose, drop what
	// was discharged, and the result must equal the standing set.
	for _, row := range delta.Obligations {
		snapshot[fmt.Sprint(row["kind"])] = true
	}
	for _, row := range delta.Discharged {
		delete(snapshot, fmt.Sprint(row["kind"]))
	}
	_, full, code := situationOf(t, "--ledger", ld, "--key", keys["workerA"])
	if code != 0 {
		t.Fatalf("standing: %d", code)
	}
	standing := kindsOf(full.Obligations)
	if len(snapshot) != len(standing) {
		t.Fatalf("applying the delta must reproduce the standing set: applied %v, standing %v", snapshot, standing)
	}
	for kind := range standing {
		if !snapshot[kind] {
			t.Fatalf("applying the delta must reproduce the standing set: applied %v, standing %v", snapshot, standing)
		}
	}
}

func journalLen(t *testing.T, ld string) int {
	t.Helper()
	j, err := refusals.Load(filepath.Join(ld, refusals.File))
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	return len(j.Entries)
}
