// The seed ledger verb family (docs/next-build-plan.md Phase 1 item 7;
// plans/os-89412090.md): verify replays the whole chain from genesis in
// one command, append is the dev tool that signs at the current tip after
// the classification lint, and show reads without writing. Every response
// is a position-stamped seed-envelope/0; failure detail renders into the
// error message ("position N: reason: detail"), never into new fields.
package main

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/shaunlmason/open-seed/next/internal/classify"
	"github.com/shaunlmason/open-seed/next/internal/envelope"
	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/genesis"
	"github.com/shaunlmason/open-seed/next/internal/keyring"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
	"github.com/shaunlmason/open-seed/next/internal/refusal"
	"github.com/shaunlmason/open-seed/next/internal/transition"
	"github.com/shaunlmason/open-seed/next/internal/version"
)

func runLedger(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "ledger requires a subverb: verify, append, or show"), stdout, stderr)
	}
	switch args[0] {
	case "verify":
		return runLedgerVerify(args[1:], stdout, stderr)
	case "append":
		return runLedgerAppend(args[1:], stdout, stderr)
	case "show":
		return runLedgerShow(args[1:], stdout, stderr)
	default:
		return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("unknown ledger subverb %q", args[0])), stdout, stderr)
	}
}

// failureEnvelope maps a verification failure to its exit code: version
// discipline troubles refuse at 10, everything else at 8, with the
// deterministic "position N: reason: detail" message. The envelope's
// position field carries the failing position too: ledger-aware refusals
// are stamped like every other ledger-aware response, not only narrated.
func failureEnvelope(fail *ledger.Failure) *envelope.Envelope {
	return refusal.FailureEnvelope(fail)
}

// supportedList renders a supported-version set deterministically for
// refusal messages.
func supportedList(set map[string]bool) string {
	vs := make([]string, 0, len(set))
	for v := range set {
		vs = append(vs, v)
	}
	sort.Strings(vs)
	return strings.Join(vs, ", ")
}

func stampTip(env *envelope.Envelope, count int) *envelope.Envelope {
	return refusal.StampTip(env, count)
}

func openStore(dir string) (*ledger.Store, *envelope.Envelope) {
	store, err := ledger.Open(dir)
	if err != nil {
		return nil, envelope.Fail(envelope.ExitUnavailable, "unavailable", fmt.Sprintf("cannot open ledger dir: %v", err))
	}
	return store, nil
}

// openStoreReadOnly is the show path: inspecting a ledger must not change
// it, so no layout creation and no head healing (plans/os-89412090.md,
// "show never writes").
func openStoreReadOnly(dir string) (*ledger.Store, *envelope.Envelope) {
	store, err := ledger.OpenReadOnly(dir)
	if err != nil {
		return nil, envelope.Fail(envelope.ExitUnavailable, "unavailable", fmt.Sprintf("cannot open ledger dir: %v", err))
	}
	return store, nil
}

func runLedgerVerify(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("ledger verify", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("ledger", "", "ledger directory")
	supported := fs.String("supported", "", "comma-separated supported protocol versions (default: this build's)")
	if err := fs.Parse(args); err != nil || *dir == "" || fs.NArg() != 0 {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "ledger verify requires --ledger <dir>"), stdout, stderr)
	}
	store, failEnv := openStore(*dir)
	if failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	resolve, _, err := genesis.Bootstrap(store)
	if err != nil {
		return render(envelope.Fail(envelope.ExitChainInvalid, "chain_invalid", err.Error()), stdout, stderr)
	}
	var opts []ledger.VerifyOption
	if *supported != "" {
		opts = append(opts, ledger.WithSupportedVersions(strings.Split(*supported, ",")...))
	}
	rep, err := store.VerifyFromGenesis(resolve, opts...)
	if err != nil {
		var fail *ledger.Failure
		if errors.As(err, &fail) {
			return render(failureEnvelope(fail), stdout, stderr)
		}
		return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()), stdout, stderr)
	}
	env := envelope.OK(map[string]any{"count": rep.Count, "tip": rep.Tip})
	return render(stampTip(env, rep.Count), stdout, stderr)
}

func runLedgerAppend(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("ledger append", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("ledger", "", "ledger directory")
	keyPath := fs.String("key", "", "OpenSSH ed25519 private key of the appending actor")
	verb := fs.String("verb", "", "event verb (namespace.verb)")
	subject := fs.String("subject", "", "event subject")
	payload := fs.String("payload", "", "event payload (JSON object)")
	supported := fs.String("supported", "", "comma-separated supported protocol versions (default: this build's)")
	remote := fs.String("remote", "", "remote ledger repository (cooperative posture: validate locally, then push)")
	refName := fs.String("ref", DefaultRemoteRef, "remote ledger ref")
	stateDir := fs.String("state", "", "client state dir for the persisted verified head (default: user cache)")
	config := fs.String("config", "", "deployment declaration (default: $SEED_CONFIG, else ./seed.json when present)")
	if err := fs.Parse(args); err != nil || (*dir == "") == (*remote == "") || *keyPath == "" || *verb == "" || *subject == "" || *payload == "" || fs.NArg() != 0 {
		return render(envelope.Fail(envelope.ExitUsage, "usage",
			"ledger append (dev tool) requires --ledger or --remote (not both), --key, --verb, --subject, --payload"), stdout, stderr)
	}
	if vs := classify.Lint([]byte(*payload)); len(vs) > 0 {
		parts := make([]string, 0, len(vs))
		for _, v := range vs {
			parts = append(parts, fmt.Sprintf("%s: %s", v.Pointer, v.Rule))
		}
		return render(envelope.Fail(envelope.ExitClassificationRef, "classification_refused",
			"payload fails data classification: "+strings.Join(parts, "; ")), stdout, stderr)
	}
	keyBytes, err := os.ReadFile(*keyPath)
	if err != nil {
		return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("cannot read --key: %v", err)), stdout, stderr)
	}
	signer, err := event.ParsePrivateKey(keyBytes)
	if err != nil {
		return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("--key: %v", err)), stdout, stderr)
	}
	if *remote != "" {
		return runLedgerAppendRemote(*remote, *refName, *stateDir, *verb, *subject, *payload, *supported, *config, signer, stdout, stderr)
	}
	// Claiming is online-only (plans/os-5dc16a7c.md): exclusivity is a
	// property granted at admission, and only the push round-trip can
	// order two rivals — two offline actors claiming the same contract
	// have not claimed anything. The boundary enforces this regardless;
	// refusing here keeps the dev tool from drafting doomed work.
	if table, terr := transition.Default(); terr == nil && table.Exclusive(*verb) {
		return render(exclusiveOnlineOnly(*verb), stdout, stderr)
	}
	store, failEnv := openStore(*dir)
	if failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	resolve, _, err := genesis.Bootstrap(store)
	if err != nil {
		return render(envelope.Fail(envelope.ExitChainInvalid, "chain_invalid", err.Error()), stdout, stderr)
	}
	// Appending signs at the version active at the tip, which only a
	// replay can name (an upgrade event anywhere in the chain moves it off
	// the build default). The same replay refuses to grow a chain that
	// does not verify.
	supportedSet := map[string]bool{}
	for _, v := range version.Supported() {
		supportedSet[v] = true
	}
	var opts []ledger.VerifyOption
	if *supported != "" {
		vs := strings.Split(*supported, ",")
		opts = append(opts, ledger.WithSupportedVersions(vs...))
		supportedSet = map[string]bool{}
		for _, v := range vs {
			supportedSet[v] = true
		}
	}
	var records []*event.Record
	opts = append(opts, ledger.WithObserver(func(pos int, r *event.Record) {
		records = append(records, r)
	}))
	rep, err := store.VerifyFromGenesis(resolve, opts...)
	if err != nil {
		var fail *ledger.Failure
		if errors.As(err, &fail) {
			return render(failureEnvelope(fail), stdout, stderr)
		}
		return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()), stdout, stderr)
	}
	if !supportedSet[rep.ActiveVersion] {
		env := envelope.Fail(envelope.ExitVersionMismatch, ledger.ReasonVersionUnsupported,
			fmt.Sprintf("the version active at the tip is %q; this build appends only %s", rep.ActiveVersion, supportedList(supportedSet)))
		return render(stampTip(env, rep.Count), stdout, stderr)
	}
	fp, err := event.Fingerprint(signer.Public().(ed25519.PublicKey))
	if err != nil {
		return render(envelope.Fail(envelope.ExitUsage, "usage", err.Error()), stdout, stderr)
	}
	rec, err := event.Sign(event.Event{
		V:       rep.ActiveVersion,
		TS:      time.Now().UTC().Format(time.RFC3339),
		Actor:   fp,
		Verb:    *verb,
		Subject: *subject,
		Payload: json.RawMessage(*payload),
		Prev:    rep.Tip,
	}, signer)
	if err != nil {
		return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("cannot sign event: %v", err)), stdout, stderr)
	}
	// From seed/1 the keyring projection resolves the signer (standing
	// decides who appends) and previews actor events before anything is
	// written: a local append must never create a record the next replay
	// refuses.
	appendResolve := resolve
	ring, _, ringErr := keyring.StateAt(records)
	if ringErr == nil && keyring.Applies(rep.ActiveVersion) && ring.Seeded() {
		appendResolve = ring.Resolver()
		if err := ring.Preview(rec); err != nil {
			env := envelope.Fail(envelope.ExitChainInvalid, "chain_invalid",
				fmt.Sprintf("actor event would fail verification: %v", err))
			return render(journalAttempt(stampTip(stampAffordances(env, *dir, signer, *subject), rep.Count), *dir, signer, *verb, *subject), stdout, stderr)
		}
	}
	pos, err := store.Append(rec, appendResolve)
	if err != nil {
		if errors.Is(err, ledger.ErrUnknownActor) {
			env := envelope.Fail(envelope.ExitChainInvalid, "chain_invalid",
				fmt.Sprintf("signer is not resolvable at the tip (genesis root, or keyring standing from %s): %v", version.Seed1, err))
			return render(stampAffordances(env, *dir, signer, *subject), stdout, stderr)
		}
		return render(stampAffordances(envelope.Fail(envelope.ExitChainInvalid, "chain_invalid", err.Error()), *dir, signer, *subject), stdout, stderr)
	}
	hash, err := rec.Event.Hash()
	if err != nil {
		return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()), stdout, stderr)
	}
	env := envelope.OK(map[string]any{"appended": hash, "verb": *verb})
	return render(journalAttempt(stampTip(stampAffordances(env, *dir, signer, *subject), pos+1), *dir, signer, *verb, *subject), stdout, stderr)
}

// exclusiveOnlineOnly is the one account of why an exclusive verb
// needs the remote, shared by the raw seam and by the loop verbs so
// a lane never meets two explanations of one rule.
func exclusiveOnlineOnly(verb string) *envelope.Envelope {
	return envelope.Fail(envelope.ExitContention, "contention",
		fmt.Sprintf("%s is an exclusive verb and claiming is online-only — exclusivity is granted at admission, so it needs --remote: two offline actors claiming the same contract have not claimed anything", verb))
}

func runLedgerShow(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("ledger show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("ledger", "", "ledger directory")
	position := fs.Int("position", -1, "derived ordinal of the record to show")
	if err := fs.Parse(args); err != nil || *dir == "" || fs.NArg() != 0 {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "ledger show requires --ledger <dir> [--position <n>]"), stdout, stderr)
	}
	store, failEnv := openStoreReadOnly(*dir)
	if failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	if *position < 0 {
		tip, count, err := store.Tip()
		if err != nil {
			return render(envelope.Fail(envelope.ExitChainInvalid, "chain_invalid", err.Error()), stdout, stderr)
		}
		head, ok, err := store.ReadHead()
		if err != nil {
			return render(envelope.Fail(envelope.ExitChainInvalid, "chain_invalid", fmt.Sprintf("HEAD unreadable: %v", err)), stdout, stderr)
		}
		result := map[string]any{"count": count, "tip": tip}
		if ok {
			result["head"] = head
		}
		return render(stampTip(envelope.OK(result), count), stdout, stderr)
	}
	var found *event.Record
	// The scan already visits every record when the position is
	// absent, so the count comes out of the iteration already running
	// (plans/os-fa69345e.md D1): a not_found that read the whole
	// chain KNOWS the tip, and discarding it would leave the caller
	// unable to tell "the position does not exist yet" from "the
	// position never will", which is the one refusal where a race is
	// most likely.
	count := 0
	stop := errors.New("stop")
	err := store.Records(func(pos int, rec *event.Record) error {
		count = pos + 1
		if pos == *position {
			found = rec
			return stop
		}
		return nil
	})
	if err != nil && !errors.Is(err, stop) {
		// The stamp is the position the response was computed at
		// (next/spec/envelope.md), and a scan that read records 0..p-1
		// and failed at p was computed at p: store.Records returns the
		// parse error before invoking the callback for p, so count is
		// exactly the failing position, the stamp verify gives the same
		// chain (plans/os-37fcf7c6.md D1). Null stays for a refusal
		// raised before any position was read.
		env := envelope.Fail(envelope.ExitChainInvalid, "chain_invalid", err.Error())
		pos := fmt.Sprintf("%d", count)
		env.Position = &pos
		return render(env, stdout, stderr)
	}
	if found == nil {
		// stampTip declines at zero, so an empty ledger correctly
		// establishes no position without a branch saying so (D2).
		return render(stampTip(envelope.Fail(envelope.ExitNotFound, "not_found", fmt.Sprintf("no record at position %d", *position)), count), stdout, stderr)
	}
	env := envelope.OK(map[string]any{"event": found.Event, "sig": found.Sig})
	pos := fmt.Sprintf("%d", *position)
	env.Position = &pos
	return render(env, stdout, stderr)
}
