// The doctor verb is the preflight tool the charter's posture section
// names (SEED-NEXT.md Part II "Postures"; plans/os-3c72f93f.md): it
// states the deployment's declared posture and, for cooperative, says
// the named consequence in plain words — verbatim from the one constant
// in internal/posture. It reads no ledger, so its envelope carries no
// position.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/shaunlmason/open-seed/next/executor"
	"github.com/shaunlmason/open-seed/next/internal/envelope"
	"github.com/shaunlmason/open-seed/next/internal/platform"
	"github.com/shaunlmason/open-seed/next/internal/posture"
	"github.com/shaunlmason/open-seed/next/internal/propose"
	"github.com/shaunlmason/open-seed/next/internal/protections"
)

func runDoctor(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	config := fs.String("config", posture.DeclarationPath, "deployment declaration file")
	probe := fs.Bool("probe", false, "probe the admission service's health (forge-hosted)")
	current := fs.String("current", "", "the forge's state as a snapshot file: report protections drift (forge-hosted)")
	repo := fs.String("repo", "", "working tree for the CODEOWNERS and workflow checks beside --current")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "doctor takes --config <path> [--probe] [--current <snapshot> [--repo <dir>]]"), stdout, stderr)
	}
	cfg, err := posture.Load(*config)
	if err != nil {
		if errors.Is(err, posture.ErrUndeclared) {
			return render(envelope.Fail(envelope.ExitNotFound, "posture_undeclared", err.Error()), stdout, stderr)
		}
		if errors.Is(err, posture.ErrUnreadable) {
			// An operational read failure, never a judgment on the
			// declaration's content (exit 13 is reserved for that).
			return render(envelope.Fail(envelope.ExitUnreadable, "posture_unreadable", err.Error()), stdout, stderr)
		}
		return render(envelope.Fail(envelope.ExitPostureInvalid, "posture_invalid", err.Error()), stdout, stderr)
	}
	result := map[string]any{
		"posture":  string(cfg.Posture),
		"enforced": cfg.Posture.Enforced(),
		// The protected surface the enforced hook write-denies to
		// agent credentials (plans/os-465e356e.md D1): the declared
		// entries plus the declaration itself, so the operator sees
		// exactly what the remote will refuse.
		"protected": cfg.ProtectedSurface(),
	}
	// The executor substrates this build provisions through, each with
	// its budget posture (plans/os-083112ac.md D2): local worktree
	// always; the container, cloud and remote adapters when declared.
	result["adapters"] = doctorAdapters(cfg)
	// The preseed's blocks, each declared or not (plans/os-0d4f2af3.md).
	// The platform and the postures it can run
	// (plans/os-b55e5647.md D4; next/spec/platform.md): the enforced
	// self-hosted hook needs a server that executes it, so a bare
	// Windows checkout names the postures it must run instead.
	result["platform"] = platform.Report()
	result["preseed"] = map[string]any{
		"protocol":   cfg.Protocol,
		"governance": cfg.Governance != nil,
		"guardrails": cfg.Guardrails != nil,
		"teams":      cfg.Teams != nil,
		"protected":  len(cfg.Protected),
	}
	// The checkpoint-trust choice is reported as declared or as
	// undeclared, never filled in (next/spec/checkpoints.md).
	if trust := cfg.CheckpointTrust(); trust != "" {
		result["checkpoints"] = map[string]any{"trust": trust}
	} else {
		// Reported, not narrated: stderr is for the consequences the
		// charter wants in front of an operator, and an unmade choice
		// is a fact the machine field states.
		result["checkpoints"] = map[string]any{"trust": nil, "undeclared": true}
	}
	switch cfg.Posture {
	case posture.Cooperative:
		result["consequence"] = posture.Consequence
		// The charter requires plain words in front of the operator, not
		// only a machine field.
		fmt.Fprintln(stderr, posture.Consequence)
	case posture.EnforcedForgeHosted:
		// The third posture reports the deployment it can see: where
		// proposals go, which branch the ledger rides and which forge
		// identity is its sole writer (plans/os-5c8a312c.md D7). The
		// gap sentence that stood here until the service landed is
		// gone with the gap.
		a := cfg.Admission
		result["admission"] = map[string]any{
			"endpoint":   a.Endpoint,
			"identity":   a.Identity,
			"ledger_ref": cfg.LedgerRef(),
			"checks":     append([]string{}, a.Checks...),
			"reviews":    a.Reviews,
			"owners":     append([]string{}, a.Owners...),
			"forge":      a.ForgeKind(),
			"api":        a.Api,
		}
		fmt.Fprintf(stderr, "enforced-forge-hosted: proposals go to %s; the ledger rides %s, written by %s alone; forge %s\n", a.Endpoint, cfg.LedgerRef(), a.Identity, a.ForgeKind())
		if *probe {
			h, err := propose.New(a.Endpoint).Probe()
			if err != nil {
				return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", fmt.Sprintf("the admission service at %s did not answer the probe: %v", a.Endpoint, err)), stdout, stderr)
			}
			service := map[string]any{"remote": h.Remote, "ref": h.Ref, "tip": h.Tip, "position": nil}
			if h.Position != nil {
				service["position"] = *h.Position
			}
			result["service"] = service
			if h.Ref != cfg.LedgerRef() {
				fmt.Fprintf(stderr, "warning: the service serves %s while the declaration names %s\n", h.Ref, cfg.LedgerRef())
			}
		}
		if *current != "" {
			rep, _, err := protections.Plan(cfg, protections.Snapshot{Path: *current}, *repo)
			if err != nil {
				return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()), stdout, stderr)
			}
			result["protections"] = reportResult(rep)
			if rep.DriftCount > 0 {
				fmt.Fprintf(stderr, "protections: %d drift(s) against the declaration; run `seed protections plan` for each\n", rep.DriftCount)
				return render(envelope.Fail(envelope.ExitDrift, "protections_drift", fmt.Sprintf("%d drift(s) against the declaration", rep.DriftCount)), stdout, stderr)
			}
		}
	}
	return render(envelope.OK(result), stdout, stderr)
}

// doctorAdapters lists the executor substrates this build provisions
// through with their budget postures, from the declaration's executors
// block.
func doctorAdapters(cfg *posture.Config) []map[string]any {
	list := []executor.Adapter{executor.LocalWorktree{}}
	if ex := cfg.Executors; ex != nil {
		if ex.Container != nil {
			list = append(list, executor.Container{})
		}
		if ex.Cloud != nil {
			list = append(list, executor.CloudSession{})
		}
		if ex.Remote != nil {
			list = append(list, executor.RemoteWorker{})
		}
	}
	out := make([]map[string]any, 0, len(list))
	for _, a := range list {
		d := executor.DescribeOf(a.Tuple().Harness, a)
		out = append(out, map[string]any{"name": d.Name, "harness": d.Harness, "budget": d.Budget, "reason": d.Reason})
	}
	return out
}
