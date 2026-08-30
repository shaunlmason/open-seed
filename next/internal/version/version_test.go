package version

import (
	"regexp"
	"testing"
)

func TestIdentity(t *testing.T) {
	if Name != "seed" {
		t.Fatalf("Name = %q, want %q (the successor claims the name)", Name, "seed")
	}
	if Version == "" {
		t.Fatal("Version must not be empty")
	}
	// conformance: III.A groundwork; a genesis event names the governance
	// root and protocol version. The protocol identifier must stay in the
	// seed/N form the spec defines so version-mismatch refusal can compare.
	if !regexp.MustCompile(`^seed/[0-9]+$`).MatchString(Protocol) {
		t.Fatalf("Protocol = %q, want seed/<n> per next/spec/protocol.md", Protocol)
	}
}
