// The project verbs (docs/next-build-plan.md Phase 4 item 1;
// plans/os-4d5cacff.md): one-command projection rebuild over the
// engine in internal/project. The engine opens the ledger read-only
// and refuses ledger/output overlap before anything is created, so the
// verb cannot touch authoritative state.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/shaunlmason/open-seed/next/internal/envelope"
	"github.com/shaunlmason/open-seed/next/internal/genesis"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
	"github.com/shaunlmason/open-seed/next/internal/project"
)

func runProject(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "project requires a subverb: rebuild"), stdout, stderr)
	}
	switch args[0] {
	case "rebuild":
		return runProjectRebuild(args[1:], stdout, stderr)
	default:
		return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("unknown project subverb %q", args[0])), stdout, stderr)
	}
}

func runProjectRebuild(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("project rebuild", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("ledger", "", "ledger directory")
	out := fs.String("out", "projections", "projection output root")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 || *dir == "" {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "project rebuild --ledger <dir> [--out <dir>]"), stdout, stderr)
	}
	if err := project.CheckOverlap(*dir, *out); err != nil {
		return render(envelope.Fail(envelope.ExitUsage, "usage", err.Error()), stdout, stderr)
	}
	store, err := ledger.OpenReadOnly(*dir)
	if err != nil {
		return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()), stdout, stderr)
	}
	resolve, _, err := genesis.Bootstrap(store)
	if err != nil {
		return render(envelope.Fail(envelope.ExitChainInvalid, "chain_invalid", err.Error()), stdout, stderr)
	}
	results, err := project.Rebuild(*dir, *out, project.Default(), resolve)
	if err != nil {
		var fail *ledger.Failure
		if errors.As(err, &fail) {
			return render(failureEnvelope(fail), stdout, stderr)
		}
		if errors.Is(err, project.ErrOverlap) {
			return render(envelope.Fail(envelope.ExitUsage, "usage", err.Error()), stdout, stderr)
		}
		return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()), stdout, stderr)
	}
	outAbs, err := filepath.Abs(*out)
	if err != nil {
		outAbs = *out
	}
	list := make([]map[string]any, 0, len(results))
	for _, r := range results {
		list = append(list, map[string]any{"name": r.Name, "position": fmt.Sprintf("%d", r.Position), "tip": r.Tip})
	}
	env := envelope.OK(map[string]any{"out": outAbs, "projections": list})
	if len(results) > 0 {
		env = stampTip(env, results[0].Position)
	}
	return render(env, stdout, stderr)
}
