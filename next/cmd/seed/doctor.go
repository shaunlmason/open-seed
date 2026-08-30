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

	"github.com/shaunlmason/open-seed/next/internal/envelope"
	"github.com/shaunlmason/open-seed/next/internal/posture"
)

func runDoctor(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	config := fs.String("config", "seed.json", "deployment declaration file")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "doctor takes only --config <path>"), stdout, stderr)
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
	}
	switch cfg.Posture {
	case posture.Cooperative:
		result["consequence"] = posture.Consequence
		// The charter requires plain words in front of the operator, not
		// only a machine field.
		fmt.Fprintln(stderr, posture.Consequence)
	case posture.EnforcedForgeHosted:
		result["gap"] = posture.ForgeHostedGap
		fmt.Fprintln(stderr, posture.ForgeHostedGap)
	}
	return render(envelope.OK(result), stdout, stderr)
}
