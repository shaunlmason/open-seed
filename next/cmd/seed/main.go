// Command seed is the Seed CLI skeleton (docs/next-build-plan.md Phase 0):
// argument parsing, envelope emission, and exit codes, mirroring the
// engine's cmd/seed seam so later verbs land as cases, not rework. During
// incubation the binary is built to next/bin/ and invoked explicitly;
// scripts/seed (v1) remains the only coordination entry point until
// spin-out.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/shaunlmason/open-seed/next/internal/envelope"
	"github.com/shaunlmason/open-seed/next/internal/version"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run dispatches one verb and renders exactly one envelope on stdout,
// returning the process exit code. The envelope carries the same code, so
// machine callers can branch without inspecting wait status.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "missing verb — try 'seed version'"), stdout, stderr)
	}
	switch args[0] {
	case "version":
		return render(envelope.OK(map[string]any{
			"name":     version.Name,
			"version":  version.Version,
			"protocol": version.Protocol,
		}), stdout, stderr)
	default:
		return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("unknown verb %q — try 'seed version'", args[0])), stdout, stderr)
	}
}

// render writes the envelope; a render failure is the one condition that
// cannot itself be an envelope, so it reports on stderr and exits 1.
func render(e *envelope.Envelope, stdout, stderr io.Writer) int {
	if err := e.Render(stdout); err != nil {
		fmt.Fprintf(stderr, "seed: envelope render failed: %v\n", err)
		return 1
	}
	return e.Exit
}
