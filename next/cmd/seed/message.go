package main

// The deliberate body read (plans/os-8451d939.md D6; build plan Phase 9
// item 5(b)). `seed situation` says a message EXISTS; this returns one
// message's payload, at one cited position, to a caller it is addressed
// to.
//
// The split is the whole design. situation is read on every wake,
// unbidden, so a body there would let message.sent — the injection
// suite's named relaying residual, which needs no capability at all —
// steer a lane that never asked to be told anything. A read naming one
// position is the opposite case: the reader chose to look, after a
// notice told it who sent the thing and how big it is. That is the
// posture next/spec/lanes.md's residual analysis already accepts.
//
// Three things this verb is deliberately not:
//
//   - NOT a ledger verb. It appends nothing, so the build plan's "no
//     message.read verb is introduced" holds: what that forbids is a
//     FACT recording read-state, which would hand a lane a second
//     cursor to disagree with the one it already carries.
//   - NOT a new refusal code. A caller the message does not address
//     gets not_found, byte for byte what a position holding no message
//     gets, so the refusal discloses nothing about what is there.
//     not_recipient (exit 23) is emphatically not reused: it names the
//     sealed-envelope recipient set, whose answer is "re-seal to the
//     current set" (next/spec/envelope.md's allocation rule forbids
//     sharing a code across two different answers).
//   - NOT a mailbox. One message at one position; the listing is
//     situation's job.

import (
	"crypto/ed25519"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/shaunlmason/open-seed/next/internal/envelope"
	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/project"
)

func runMessage(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return render(envelope.Fail(envelope.ExitUsage, "usage", "message requires a subverb: read"), stdout, stderr)
	}
	if args[0] == "read" {
		return runMessageRead(args[1:], stdout, stderr)
	}
	return render(envelope.Fail(envelope.ExitUsage, "usage",
		fmt.Sprintf("unknown message subverb %q — read", args[0])), stdout, stderr)
}

// messageNotFound is the ONE refusal this verb gives for every reason a
// caller does not get a body: no message at that position, an event
// that is not a message, a message addressed to someone else, and one
// whose addressing did not parse. Constructing it in a single place is
// what makes the indistinguishability a property rather than four
// strings that happen to match today.
func messageNotFound(at int) *envelope.Envelope {
	return envelope.Fail(envelope.ExitNotFound, "not_found",
		fmt.Sprintf("no message addressed to you at position %d", at))
}

func runMessageRead(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("message read", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	posture := bindReadPosture(fs)
	keyPath := fs.String("key", "", "OpenSSH ed25519 private key: the actor the message must address")
	at := fs.String("at", "", "the ledger position of the message to read")
	if err := fs.Parse(args); err != nil || !posture.resolved() || *at == "" || fs.NArg() != 0 {
		return render(envelope.Fail(envelope.ExitUsage, "usage",
			"message read requires --ledger <dir> or --remote <repo> (not both), --key <path> and --at <position>"), stdout, stderr)
	}
	pos, err := strconv.Atoi(*at)
	if err != nil || pos < 0 {
		return render(envelope.Fail(envelope.ExitUsage, "usage",
			fmt.Sprintf("--at takes a ledger position, got %q", *at)), stdout, stderr)
	}
	// A key is REQUIRED here, unlike situation's keyless whole-board
	// read: a body has a recipient set, and "no identity" addresses
	// nobody. The keyless read reports that mail exists; it does not
	// open it.
	if *keyPath == "" {
		return render(envelope.Fail(envelope.ExitUsage, "usage",
			"message read needs --key: a body is read as somebody, and a keyless read addresses no one"), stdout, stderr)
	}
	keyBytes, err := os.ReadFile(*keyPath)
	if err != nil {
		return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("cannot read --key: %v", err)), stdout, stderr)
	}
	signer, err := event.ParsePrivateKey(keyBytes)
	if err != nil {
		return render(envelope.Fail(envelope.ExitUsage, "usage", fmt.Sprintf("cannot parse --key: %v", err)), stdout, stderr)
	}
	fp, err := event.Fingerprint(signer.Public().(ed25519.PublicKey))
	if err != nil {
		return render(envelope.Fail(envelope.ExitUnavailable, "unavailable", err.Error()), stdout, stderr)
	}
	st, _, closePosture, failEnv := posture.open()
	defer closePosture()
	if failEnv != nil {
		return render(failEnv, stdout, stderr)
	}
	if pos >= len(st.records) {
		return render(stampTip(messageNotFound(pos), st.count), stdout, stderr)
	}
	rec := st.records[pos]
	if rec.Event.Verb != project.MessageSentVerb {
		return render(stampTip(messageNotFound(pos), st.count), stdout, stderr)
	}
	to, undeliverable := project.AddressedTo(rec.Event.Payload)
	notice := project.MessageNotice{To: to, Undeliverable: undeliverable}
	if !notice.Addresses(fp) {
		return render(stampTip(messageNotFound(pos), st.count), stdout, stderr)
	}
	return render(stampTip(envelope.OK(map[string]any{
		"from":    rec.Event.Actor,
		"subject": rec.Event.Subject,
		"at":      fmt.Sprintf("%d", pos),
		"ts":      rec.Event.TS,
		"body":    string(rec.Event.Payload),
	}), st.count), stdout, stderr)
}
