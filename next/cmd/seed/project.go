// The project verbs (docs/next-build-plan.md Phase 4 item 1;
// plans/os-4d5cacff.md): one-command projection rebuild over the
// engine in internal/project. The engine opens the ledger read-only
// and refuses ledger/output overlap before anything is created, so the
// verb cannot touch authoritative state.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/shaunlmason/open-seed/next/internal/envelope"
	"github.com/shaunlmason/open-seed/next/internal/genesis"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
	"github.com/shaunlmason/open-seed/next/internal/project"
)

func runProject(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "project requires a subverb: rebuild, current"), stdout, stderr)
	}
	switch args[0] {
	case "rebuild":
		return runProjectRebuild(args[1:], stdout, stderr)
	case "current":
		return runProjectCurrent(args[1:], stdout, stderr)
	default:
		return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("unknown project subverb %q", args[0])), stdout, stderr)
	}
}

// runProjectCurrent is the consumer verb (plans/os-fecfb3f7.md step 5;
// conformance III.D "consumers can demand a minimum position"). It is
// structurally a consumer: no --ledger flag exists, and it only reads
// the published layout. The envelope position carries the stamp's
// verified record count verbatim (the rebuild envelope stamps the
// tip's zero-based index; both conventions are stated in
// next/spec/projections.md).
func runProjectCurrent(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("project current", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	out := flags.String("out", "projections", "projection output root")
	name := flags.String("name", "", "projection name")
	minPos := flags.Int("min-position", -1, "minimum acceptable build position")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 || *name == "" {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "project current --name <projection> [--out <dir>] [--min-position <n>]"), stdout, stderr)
	}
	// Only registered projections resolve (review finding on #111):
	// a name outside the registry is unknown whatever directories
	// exist under --out, which also keeps traversal components out of
	// the path entirely.
	registered := false
	var names []string
	for _, p := range project.Default() {
		names = append(names, p.Name)
		if p.Name == *name {
			registered = true
		}
	}
	if !registered {
		return render(envelope.Fail(envelope.ExitNotFound, "not_found", fmt.Sprintf("unknown projection %q (registered: %s)", *name, strings.Join(names, ", "))), stdout, stderr)
	}
	build, err := project.Current(*out, *name)
	if err != nil {
		// Nothing published at all is not_found; a published layout
		// that exists but cannot be resolved (an unreadable or empty
		// CURRENT) is an operational failure, unavailable (review
		// finding on #111).
		if errors.Is(err, fs.ErrNotExist) {
			return render(envelope.Fail(envelope.ExitNotFound, "not_found", fmt.Sprintf("no published build for projection %q under %s: %v", *name, *out, err)), stdout, stderr)
		}
		return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", fmt.Sprintf("projection %q has a published layout that does not resolve: %v", *name, err)), stdout, stderr)
	}
	raw, err := os.ReadFile(filepath.Join(build, project.StampFile))
	if err != nil {
		return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", fmt.Sprintf("published build for %q has no readable stamp: %v", *name, err)), stdout, stderr)
	}
	var stamp project.Stamp
	if err := json.Unmarshal(raw, &stamp); err != nil {
		return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", fmt.Sprintf("published stamp for %q does not parse: %v", *name, err)), stdout, stderr)
	}
	if *minPos >= 0 && stamp.Position < *minPos {
		return render(envelope.Fail(envelope.ExitStale, "stale", fmt.Sprintf("projection %q is stamped at position %d, below the demanded minimum %d — rebuild before consuming", *name, stamp.Position, *minPos)), stdout, stderr)
	}
	abs, err := filepath.Abs(build)
	if err != nil {
		abs = build
	}
	env := envelope.OK(map[string]any{"name": stamp.Name, "position": fmt.Sprintf("%d", stamp.Position), "tip": stamp.Tip, "version": stamp.Version, "path": abs})
	pos := fmt.Sprintf("%d", stamp.Position)
	env.Position = &pos
	return render(env, stdout, stderr)
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
		list = append(list, map[string]any{"name": r.Name, "position": fmt.Sprintf("%d", r.Position), "tip": r.Tip, "version": r.Version})
	}
	env := envelope.OK(map[string]any{"out": outAbs, "projections": list})
	if len(results) > 0 {
		env = stampTip(env, results[0].Position)
	}
	return render(env, stdout, stderr)
}
