// Package executor is the public executor-adapter interface (charter
// III.H "the adapter interface is public"; SEED-NEXT.md §II.9;
// plans/os-1dad487d.md; next/spec/executors.md): provision a
// workspace with its handoff packet, wake advisorily, meter usage
// onto the observation stream, and report the provisioned runtime
// tuple. The local worktree adapter is the first implementation;
// container, cloud-session, and enrolled-remote adapters are later
// phases' rows.
//
// Provisioning is fenced to the reservation gate: a run provisions
// only against an admitted run.started, which admission granted only
// against an open, valid budget reservation — reserve, start,
// provision, meter, settle. Disposal is the caller's contract:
// dispose only after the last confirmed, admitted synchronization
// (the disposability drill proves that discipline loses nothing
// admitted; the loss window is the observation lines after the last
// sync).
package executor

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/genesis"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
	"github.com/shaunlmason/open-seed/next/internal/obs"
	"github.com/shaunlmason/open-seed/next/internal/transition"
)

// Tuple is the runtime tuple the adapter actually provisioned
// (qualification depends on it, charter §II.5). The v0 value is an
// honest stub: Phase 10 gives tuples meaning, and nothing here
// pretends qualification exists yet.
type Tuple struct {
	Runtime string `json:"runtime"`
}

// ProvisionSpec names everything a provision needs: the ledger the
// admitted run.started lives in, the repository and base revision
// the workspace detaches from, the contract subject, the executing
// actor and claim fence (the observation stream key), the handoff
// packet bytes, and the observation directory.
type ProvisionSpec struct {
	Ledger  string
	Repo    string
	Base    string
	Subject string
	Actor   string
	Fence   int
	Started int
	Packet  []byte
	ObsDir  string
}

// Run is one provisioned execution run.
type Run interface {
	// Workspace is the provisioned working directory.
	Workspace() string
	// Meter appends one metered observation line to the run's
	// stream: usage rides the ephemeral channel and settles to the
	// ledger at run end via run.settled.
	Meter(units int, step string) error
	// Dispose removes the workspace. The caller disposes only after
	// the last confirmed, admitted synchronization.
	Dispose() error
}

// Adapter is the executor adapter: provision, wake, and the
// provisioned tuple. Wake is advisory transport only — its total
// failure costs latency, never correctness (the wakeless poll-only
// drill).
type Adapter interface {
	Provision(spec ProvisionSpec) (Run, error)
	Wake(actor string) error
	Tuple() Tuple
}

// ErrNoAdmittedStart refuses a provision whose spec cites no
// admitted run.started for its fence: no execution run provisions
// outside the reservation gate.
var ErrNoAdmittedStart = errors.New("no admitted run.started for this fence — a run provisions only inside the reservation gate (reserve, then run.started, then Provision)")

// LocalWorktree is the local worktree adapter: a detached git
// worktree as the workspace, the packet at .seed-run/packet.json,
// and metering onto the per-fence observation stream.
type LocalWorktree struct{}

// Tuple reports the v0 stub.
func (LocalWorktree) Tuple() Tuple { return Tuple{Runtime: "local-worktree/v0"} }

// Wake is the documented no-op: the advisory channel that does
// nothing is the honest v0, and polling loses only latency.
func (LocalWorktree) Wake(string) error { return nil }

// Provision verifies the admitted run.started, creates the detached
// worktree, writes the packet, and ensures the observation stream
// directory. Git runs with fixed argument vectors; nothing is
// interpolated into a shell.
func (LocalWorktree) Provision(spec ProvisionSpec) (Run, error) {
	if err := verifyStarted(spec); err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "seed-run-")
	if err != nil {
		return nil, err
	}
	workspace := filepath.Join(dir, "ws")
	add := exec.Command("git", "-C", spec.Repo, "worktree", "add", "--detach", workspace, spec.Base)
	if out, err := add.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("worktree add: %v: %s", err, out)
	}
	runDir := filepath.Join(workspace, ".seed-run")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(runDir, "packet.json"), spec.Packet, 0o644); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(spec.ObsDir, spec.Actor), 0o755); err != nil {
		return nil, err
	}
	return &localRun{spec: spec, workspace: workspace}, nil
}

// verifyStarted replays the ledger and requires the admitted
// run.started the spec cites, at its position, for this fence.
func verifyStarted(spec ProvisionSpec) error {
	store, err := ledger.Open(spec.Ledger)
	if err != nil {
		return err
	}
	resolve, _, err := genesis.Bootstrap(store)
	if err != nil {
		return err
	}
	var records []*event.Record
	if _, err := store.VerifyFromGenesis(resolve, ledger.WithObserver(func(pos int, r *event.Record) {
		records = append(records, r)
	})); err != nil {
		return err
	}
	table, err := transition.Default()
	if err != nil {
		return err
	}
	s, ok := table.FoldRecords(records).State(spec.Subject)
	if !ok {
		return ErrNoAdmittedStart
	}
	for _, st := range s.RunStarts {
		if st.Pos == spec.Started && st.Fence == spec.Fence {
			return nil
		}
	}
	return ErrNoAdmittedStart
}

type localRun struct {
	spec      ProvisionSpec
	workspace string
}

func (r *localRun) Workspace() string { return r.workspace }

func (r *localRun) Meter(units int, step string) error {
	return obs.Append(r.spec.ObsDir, r.spec.Actor, obs.FormatFence(r.spec.Fence), obs.Line{
		TS:      time.Now().UTC().Format(time.RFC3339),
		Subject: r.spec.Subject,
		Step:    step,
		Units:   units,
	})
}

func (r *localRun) Dispose() error {
	remove := exec.Command("git", "-C", r.spec.Repo, "worktree", "remove", "--force", r.workspace)
	if out, err := remove.CombinedOutput(); err != nil {
		return fmt.Errorf("worktree remove: %v: %s", err, out)
	}
	return nil
}
