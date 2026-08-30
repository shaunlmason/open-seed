package envelope

import (
	"bytes"
	"strings"
	"testing"
)

// conformance: III.I groundwork; every verb response is a versioned,
// schema-stable envelope with structured errors and meaningful exit codes.
// This test pins the serialized field set: adding, removing, or renaming a
// field is a schema change and must bump V alongside next/spec/envelope.md.
func TestEnvelopeSchemaStable(t *testing.T) {
	var buf bytes.Buffer
	if err := OK(map[string]any{"k": "v"}).Render(&buf); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	want := `{"v":"seed-envelope/0","ok":true,"result":{"k":"v"},"error":null,"position":null,"affordances":[],"budget":null,"exit":0}` + "\n"
	if got != want {
		t.Fatalf("success envelope drifted from the spec shape:\n got %s\nwant %s", got, want)
	}

	buf.Reset()
	if err := Fail(ExitUsage, "usage", "missing verb").Render(&buf); err != nil {
		t.Fatal(err)
	}
	got = buf.String()
	want = `{"v":"seed-envelope/0","ok":false,"result":null,"error":{"code":"usage","message":"missing verb"},"position":null,"affordances":[],"budget":null,"exit":64}` + "\n"
	if got != want {
		t.Fatalf("refusal envelope drifted from the spec shape:\n got %s\nwant %s", got, want)
	}
}

// conformance: III.I groundwork; exit codes are meaningful and documented.
// The build-plan fixed default inherits v1 semantics where they match
// (2 contention, 6 fence, 10 version); the spec table is authoritative and
// these constants must mirror it exactly.
func TestExitCodesMatchSpecTable(t *testing.T) {
	table := map[string]struct{ got, want int }{
		"ok":                 {ExitOK, 0},
		"contention":         {ExitContention, 2},
		"invalid_transition": {ExitInvalidTransition, 3},
		"not_found":          {ExitNotFound, 4},
		"unavailable":        {ExitUnavailable, 5},
		"fenced_out":         {ExitFenced, 6},
		"halted":             {ExitHalted, 7},
		"chain_invalid":      {ExitChainInvalid, 8},
		"classification":     {ExitClassificationRef, 9},
		"version_mismatch":   {ExitVersionMismatch, 10},
		"remote_rejected":    {ExitRemoteRejected, 11},
		"head_regression":    {ExitHeadRegression, 12},
		"posture_invalid":    {ExitPostureInvalid, 13},
		"out_of_grant":       {ExitOutOfGrant, 14},
		"stale":              {ExitStale, 15},
		"usage":              {ExitUsage, 64},
		"unreadable":         {ExitUnreadable, 66},
	}
	for name, c := range table {
		if c.got != c.want {
			t.Errorf("exit code %s = %d, want %d (next/spec/envelope.md)", name, c.got, c.want)
		}
	}
}

func TestRenderSingleLine(t *testing.T) {
	var buf bytes.Buffer
	if err := Fail(ExitNotFound, "not_found", "no such subject").Render(&buf); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.HasSuffix(s, "\n") || strings.Count(s, "\n") != 1 {
		t.Fatalf("envelope must render as exactly one JSON line, got %q", s)
	}
}

func TestRenderErrorPath(t *testing.T) {
	e := OK(map[string]any{"bad": make(chan int)})
	if err := e.Render(&bytes.Buffer{}); err == nil {
		t.Fatal("expected a marshal error for an unserializable result")
	}
}
