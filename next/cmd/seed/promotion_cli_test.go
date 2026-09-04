package main

// The promotion packet's proposed declaration is held to the tree
// (plans/os-4fde2bdf.md D4): the JSON block under "The deployment" is
// what the operator copies to seed.json at the cutover, so it is
// linted here the way the fixture deployment's declaration is linted
// under make check-next, and a stale block is found in CI rather than
// at the flip.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// packetDeclaration extracts the proposed declaration from the packet.
func packetDeclaration(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "docs", "promotion.md"))
	if err != nil {
		t.Fatal(err)
	}
	doc := string(b)
	i := strings.Index(doc, "**The deployment.**")
	if i < 0 {
		t.Fatal("the packet has no deployment paragraph")
	}
	rest := doc[i:]
	start := strings.Index(rest, "```json\n")
	if start < 0 {
		t.Fatal("the deployment paragraph carries no json block")
	}
	rest = rest[start+len("```json\n"):]
	end := strings.Index(rest, "\n```")
	if end < 0 {
		t.Fatal("the json block is unterminated")
	}
	return rest[:end]
}

// conformance: build plan section 5 criterion 5 (the cutover written
// down) — the declaration the packet proposes passes seed preseed
// check against the shipped lanes: tiers in the vocabulary, teams
// naming shipped manifests, the protected surface complete, the
// protocol in the register; and the drill fails on a planted tier
// outside the vocabulary, so it is a check and not a parse.
func TestPacketDeclarationLints(t *testing.T) {
	block := packetDeclaration(t)
	write := func(content string) string {
		t.Helper()
		p := filepath.Join(t.TempDir(), "seed.json")
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	e, code := runEnv(t, "preseed", "check", "--config", write(block), "--lanes", filepath.Join("..", "..", "lanes"))
	if code != 0 || !e.OK {
		t.Fatalf("the packet's proposed declaration lints: %d %+v", code, e)
	}
	if e.Result["governance"] != true || e.Result["guardrails"] != true || e.Result["teams"] != true {
		t.Fatalf("the declaration declares governance, guardrails and teams: %+v", e.Result)
	}
	planted := strings.Replace(block, `"max_agent": "standard"`, `"max_agent": "sky"`, 1)
	if planted == block {
		t.Fatal("the mutation did not apply")
	}
	if e, code := runEnv(t, "preseed", "check", "--config", write(planted), "--lanes", filepath.Join("..", "..", "lanes")); code != 13 || e.Error == nil || e.Error.Code != "preseed_incomplete" {
		t.Fatalf("a tier outside the vocabulary is refused by name: %d %+v", code, e)
	}
}
