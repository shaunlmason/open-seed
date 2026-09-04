package main

// The erasure verb (plans/os-db5cd353.md; SEED-NEXT.md III.A row 7;
// next/spec/protocol.md "Erasure"): `artifact erase` records that an
// operator erased an artifact the chain references by digest, then
// removes the bytes from the store. The order is deliberate: a record
// with the bytes still present is a promise the next run keeps, bytes
// gone with no record is the silence the row forbids. The record is a
// loop act in the shared transport shape, judged by the boundary's
// erasure rule; an erasure that already stands is not recorded twice,
// and a re-run only finishes the removal.

import (
	"flag"
	"fmt"
	"io"

	"github.com/shaunlmason/open-seed/next/internal/artifact"
	"github.com/shaunlmason/open-seed/next/internal/envelope"
	"github.com/shaunlmason/open-seed/next/internal/erasure"
)

func runArtifact(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "artifact requires a subverb: erase"), stdout, stderr)
	}
	switch args[0] {
	case "erase":
		return runArtifactErase(args[1:], stdout, stderr)
	}
	return render(envelope.Fail(envelope.ExitUsage, "usage",
		fmt.Sprintf("unknown artifact subverb %q — erase", args[0])), stdout, stderr)
}

// runArtifactErase appends artifact.erased on the contract that
// references the artifact (or on system), then empties the store's
// buckets under the digest and reports which.
func runArtifactErase(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("artifact erase", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	f := bindLoopFlags(fs)
	digest := fs.String("artifact", "", "the artifact's digest, a lowercase-hex sha256: the sealed commitment or the receipt the contract references")
	reason := fs.String("reason", "", fmt.Sprintf("one line naming the obligation honored, at most %d bytes", erasure.MaxReasonBytes))
	repo := fs.String("repo", "", "repository whose artifact store holds the bytes")
	artifacts := fs.String("artifacts", "", "artifact store root (default <repo>/next/var/artifacts)")
	parseErr := fs.Parse(args)
	missing := ""
	if *digest == "" || *reason == "" || (*repo == "" && *artifacts == "") {
		missing = "and --artifact <digest>, --reason <text>, --repo <dir> (or --artifacts <dir>); --subject is the contract that references the artifact, or system"
	}
	if env := f.usage("artifact erase", parseErr, fs.NArg(), missing); env != nil {
		return render(env, stdout, stderr)
	}
	payload, err := erasure.Render(erasure.Erased{Artifact: *digest, Reason: *reason})
	if err != nil {
		return render(envelope.Fail(envelope.ExitInvalidTransition, "erasure_refused", err.Error()), stdout, stderr)
	}
	signer, env := loopSigner(*f.keyPath, *f.as)
	if env != nil {
		return render(env, stdout, stderr)
	}
	ls, env := openLoopSession(f)
	if env != nil {
		return render(env, stdout, stderr)
	}
	defer ls.done()
	store := artifact.Open(artifactsDir(*artifacts, *repo))
	subject, art := *f.subject, *digest
	erase := func(out map[string]any) map[string]any {
		removed, err := store.Erase(art)
		if removed == nil {
			removed = []string{}
		}
		out["removed"] = removed
		if err != nil {
			// The record stands and the bytes do not: the next run
			// finishes the removal, never re-records.
			out["erase_error"] = err.Error()
		}
		return out
	}
	// An erasure that already stands is finished, not re-recorded: a
	// second record would attribute an act that did nothing.
	if ls.ctx.Lifecycle != nil {
		if prior, ok := ls.ctx.Lifecycle.Erasure(subject, art); ok {
			env := envelope.OK(erase(map[string]any{"subject": subject, "artifact": art, "erased_at": fmt.Sprintf("%d", prior.Pos), "by": prior.Signer, "recorded": false}))
			return render(stampTip(env, ls.ctx.Count), stdout, stderr)
		}
	}
	return ls.commit(f, loopAct{verb: erasure.Verb, payload: payload, resultAt: func(pos int) map[string]any {
		return erase(map[string]any{"subject": subject, "artifact": art, "erased_at": fmt.Sprintf("%d", pos), "recorded": true})
	}}, signer, stdout, stderr)
}
