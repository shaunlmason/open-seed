package main

// The messages section of the situation read, and the deliberate body
// read beside it (plans/os-8451d939.md; build plan Phase 9 item 5(b)).
//
// The containment half — that no payload text reaches situation — is
// swept by marker in injection_cli_test.go rather than here, because a
// drill that read this package's structs would not notice a field
// added later. What lives here is the behavior: who sees what, when a
// message counts as unread, and what a body read gives back.

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// mailLedger stands up a ledger with two enrolled workers and a
// supervisor, and returns the pieces the drills address each other by.
func mailLedger(t *testing.T) (ld string, keys, fps map[string]string, send func(payload string) int) {
	t.Helper()
	ld, _, _, specCommit, _, priv, _, keys, fps := offerLedger(t)
	offerFile(t, ld, priv, specCommit, "c-1")
	send = func(payload string) int {
		t.Helper()
		if e, code := runEnv(t, "ledger", "append", "--ledger", ld, "--key", keys["supervisor"],
			"--verb", "message.sent", "--subject", "c-1", "--payload", payload); code != 0 {
			t.Fatalf("message.sent %s: %d %+v", payload, code, e)
		}
		// The position the message landed at is the tip's ordinal,
		// which the envelope of the NEXT read reports; reading it back
		// here keeps the drills from counting appends by hand.
		e, code := runEnv(t, "situation", "--ledger", ld, "--key", keys["workerA"])
		if code != 0 {
			t.Fatalf("situation after send: %d %+v", code, e)
		}
		if e.Position == nil {
			t.Fatal("the read must stamp a position")
		}
		pos, err := strconv.Atoi(*e.Position)
		if err != nil {
			t.Fatalf("the stamped position must be a number: %q", *e.Position)
		}
		return pos
	}
	return ld, keys, fps, send
}

func messagesIn(t *testing.T, ld, key string, extra ...string) []map[string]any {
	t.Helper()
	args := append([]string{"situation", "--ledger", ld, "--key", key}, extra...)
	e, code := runEnv(t, args...)
	if code != 0 {
		t.Fatalf("situation: %d %+v", code, e)
	}
	raw, ok := e.Result["messages"]
	if !ok {
		t.Fatal("the messages section is present whether or not it is empty: an absent section and an " +
			"empty one are the same question a lane should not have to ask twice (D4)")
	}
	rows, _ := raw.([]any)
	out := []map[string]any{}
	for _, r := range rows {
		m, _ := r.(map[string]any)
		out = append(out, m)
	}
	return out
}

// conformance: AC1 — the read reports the messages addressed to the
// caller, and ONLY those. A broadcast reaches everyone; a message
// addressed to another actor reaches nobody else.
func TestSituationCarriesTheCallersMessages(t *testing.T) {
	ld, keys, fps, send := mailLedger(t)
	send(fmt.Sprintf(`{"to": %q, "n": 1}`, fps["workerA"]))
	send(fmt.Sprintf(`{"to": %q, "n": 2}`, fps["workerB"]))
	send(`{"n": 3}`)

	a := messagesIn(t, ld, keys["workerA"])
	if len(a) != 2 {
		t.Fatalf("workerA sees its own message and the broadcast, saw %d: %+v", len(a), a)
	}
	b := messagesIn(t, ld, keys["workerB"])
	if len(b) != 2 {
		t.Fatalf("workerB sees its own message and the broadcast, saw %d: %+v", len(b), b)
	}
	// The one that matters: neither sees the other's.
	for _, row := range a {
		if row["bytes"] == fmt.Sprintf("%d", len(fmt.Sprintf(`{"to": %q, "n": 2}`, fps["workerB"]))) &&
			row["at"] == b[0]["at"] {
			t.Errorf("workerA must not see a message addressed to workerB: %+v", row)
		}
	}
	// The sender and the contract are reported, because a notice a
	// lane cannot act on is not worth carrying.
	for _, row := range a {
		if row["from"] != fps["supervisor"] || row["subject"] != "c-1" {
			t.Errorf("a notice names who sent it and what it concerns: %+v", row)
		}
		if _, has := row["body"]; has {
			t.Errorf("NO BODY in the orienting read (D1): %+v", row)
		}
	}
}

// conformance: AC4 — addressing resolves by D2's three cases, and a
// `to` that does not parse addresses NOBODY rather than everybody.
//
// Fail-closed is the point. Broadcasting a malformed address widens
// delivery from one intended recipient to every actor on an encoding
// slip, which contradicts "and only those" above (review finding on
// #209).
func TestMalformedAddressingReachesNobody(t *testing.T) {
	ld, keys, fps, send := mailLedger(t)
	for _, tc := range []struct {
		name, payload string
		reaches       bool
	}{
		{"no to key at all", `{"n": 1}`, true},
		{"a string recipient", fmt.Sprintf(`{"to": %q}`, fps["workerA"]), true},
		{"an all-string array", fmt.Sprintf(`{"to": [%q, %q]}`, fps["workerA"], fps["workerB"]), true},
		// The review finding's own example: the intended recipient is
		// visible in the array, and delivering to it anyway would be
		// inventing which half the sender meant.
		{"an array with a number in it", fmt.Sprintf(`{"to": [%q, 7]}`, fps["workerA"]), false},
		{"a number", `{"to": 7}`, false},
		{"an object", `{"to": {"fp": "x"}}`, false},
		{"an empty array", `{"to": []}`, false},
		{"an empty string", `{"to": ""}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := len(messagesIn(t, ld, keys["workerA"]))
			send(tc.payload)
			after := messagesIn(t, ld, keys["workerA"])
			got := len(after) > before
			if got != tc.reaches {
				t.Fatalf("reaches workerA = %v, want %v: %+v", got, tc.reaches, after)
			}
		})
	}
	// And the undeliverable ones are not ERASED: the keyless read
	// applies no caller filter, so an operator looking at the board
	// still finds them. A typo costs delivery, not the message (D2).
	e, code := runEnv(t, "situation", "--ledger", ld)
	if code != 0 {
		t.Fatalf("keyless situation: %d %+v", code, e)
	}
	rows, _ := e.Result["messages"].([]any)
	if len(rows) != 8 {
		t.Fatalf("the keyless whole-board read filters nothing and must show all eight, showed %d", len(rows))
	}
	undeliverable := 0
	for _, r := range rows {
		if m, _ := r.(map[string]any); m["undeliverable"] == true {
			undeliverable++
		}
	}
	if undeliverable != 5 {
		t.Errorf("the five unreadable addresses are marked undeliverable, not hidden: %d", undeliverable)
	}
}

// conformance: AC3 — unread is the cursor and nothing else. The
// boundary is drilled AT the cited position, which is where an
// off-by-one lives: --since cites a tip ordinal, so the message AT
// that position was already seen.
func TestUnreadIsTheCitedCursor(t *testing.T) {
	ld, keys, _, send := mailLedger(t)
	first := send(`{"n": 1}`)
	second := send(`{"n": 2}`)

	all := messagesIn(t, ld, keys["workerA"])
	if len(all) != 2 {
		t.Fatalf("with no cursor cited, everything the caller can see is unread: %+v", all)
	}
	// Position order, oldest first: the cursor a lane carries forward
	// is a position, so a section ordered any other way would make
	// "everything after my cursor" mean reading the list backwards.
	if all[0]["at"] != strconv.Itoa(first) || all[1]["at"] != strconv.Itoa(second) {
		t.Fatalf("notices are in position order, oldest first: %+v", all)
	}
	for _, row := range messagesIn(t, ld, keys["workerA"]) {
		if row["unread"] != true {
			t.Errorf("a caller that names no cursor has said nothing about what it has seen: %+v", row)
		}
	}
	// AT the first message's position: it was seen, the second was not.
	at := messagesIn(t, ld, keys["workerA"], "--since", strconv.Itoa(first))
	if len(at) != 1 || at[0]["at"] != strconv.Itoa(second) {
		t.Fatalf("--since %d must exclude the message AT %d and keep the one after: %+v", first, first, at)
	}
	// One BEFORE it: both are new.
	before := messagesIn(t, ld, keys["workerA"], "--since", strconv.Itoa(first-1))
	if len(before) != 2 {
		t.Fatalf("--since %d must keep both: %+v", first-1, before)
	}
	// At the tip: nothing is new, and the section is still present.
	if got := messagesIn(t, ld, keys["workerA"], "--since", strconv.Itoa(second)); len(got) != 0 {
		t.Fatalf("--since at the tip leaves nothing unread: %+v", got)
	}
}

// conformance: AC5 — the section is present and empty for a caller
// with no mail, so a lane never has to tell an absent section from an
// empty one.
func TestTheMessagesSectionIsAlwaysPresent(t *testing.T) {
	ld, keys, _, _ := mailLedger(t)
	if got := messagesIn(t, ld, keys["workerA"]); len(got) != 0 {
		t.Fatalf("no mail sent, so the section is present and empty: %+v", got)
	}
}

// conformance: AC6 — the deliberate body read. A recipient gets the
// body; everyone else gets not_found, byte for byte what a position
// holding no message gives, so the refusal discloses nothing about
// what is there.
func TestMessageReadGivesTheBodyToRecipientsOnly(t *testing.T) {
	ld, keys, fps, send := mailLedger(t)
	mine := send(fmt.Sprintf(`{"to": %q, "secret": "for A"}`, fps["workerA"]))
	theirs := send(fmt.Sprintf(`{"to": %q, "secret": "for B"}`, fps["workerB"]))
	cast := send(`{"secret": "for everyone"}`)
	bad := send(fmt.Sprintf(`{"to": [%q, 7], "secret": "for nobody"}`, fps["workerA"]))

	read := func(key string, at int) (map[string]any, int, string) {
		t.Helper()
		e, code := runEnv(t, "message", "read", "--ledger", ld, "--key", key, "--at", strconv.Itoa(at))
		msg := ""
		if e.Error != nil {
			msg = e.Error.Code + ": " + e.Error.Message
		}
		return e.Result, code, msg
	}

	res, code, _ := read(keys["workerA"], mine)
	if code != 0 {
		t.Fatalf("a recipient reads its own message: %d", code)
	}
	if body, _ := res["body"].(string); !strings.Contains(body, "for A") {
		t.Errorf("the body is what the read is for: %+v", res)
	}
	if res, code, _ := read(keys["workerA"], cast); code != 0 {
		t.Errorf("a broadcast is readable by anyone: %d %+v", code, res)
	}

	// The disclosure property: four different reasons, ONE refusal.
	// If these ever diverge, the refusal starts telling a caller
	// whether something is there.
	notMine, codeA, msgA := read(keys["workerA"], theirs)
	_ = notMine
	nothingThere, codeB, msgB := read(keys["workerA"], 0)
	_ = nothingThere
	pastTheTip, codeC, msgC := read(keys["workerA"], 9999)
	_ = pastTheTip
	nobodys, codeD, msgD := read(keys["workerA"], bad)
	_ = nobodys
	for _, c := range []int{codeA, codeB, codeC, codeD} {
		if c != envelopeNotFoundExit {
			t.Fatalf("every reason a caller gets no body is the same not_found: got %d", c)
		}
	}
	// Byte for byte, except for the position each names — which the
	// caller supplied, so it discloses nothing it did not already know.
	want := []string{
		fmt.Sprintf("not_found: no message addressed to you at position %d", theirs),
		"not_found: no message addressed to you at position 0",
		"not_found: no message addressed to you at position 9999",
		fmt.Sprintf("not_found: no message addressed to you at position %d", bad),
	}
	for i, got := range []string{msgA, msgB, msgC, msgD} {
		if got != want[i] {
			t.Errorf("refusal %d differs from the one shape: %q != %q", i, got, want[i])
		}
	}
	// A key is required: a body is read as somebody.
	if _, code := runEnv(t, "message", "read", "--ledger", ld, "--at", strconv.Itoa(cast)); code != envelopeUsageExit {
		t.Errorf("a keyless body read addresses no one and must refuse as usage: %d", code)
	}
}

// envelopeNotFoundExit is next/spec/envelope.md row 4.
const envelopeNotFoundExit = 4
