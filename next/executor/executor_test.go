package executor

// The adapter surface's pure drills; the full bracket (Provision
// against an admitted run.started, metering, the SIGKILL
// disposability drill) runs in cmd/seed, where the ledger and
// repository fixtures live.

import (
	"testing"
)

func TestLocalWorktreeSurface(t *testing.T) {
	var lw LocalWorktree
	if got := lw.Tuple(); got.Runtime != "local-worktree/v0" {
		t.Fatalf("the v0 tuple stub: %+v", got)
	}
	if err := lw.Wake("anyone"); err != nil {
		t.Fatalf("wake is the advisory no-op — its total failure costs latency, and the honest v0 does nothing: %v", err)
	}
	if _, err := lw.Provision(ProvisionSpec{Ledger: t.TempDir()}); err == nil {
		t.Fatal("a provision without a verifiable ledger refuses — no run outside the reservation gate")
	}
}
