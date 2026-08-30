// Package version carries the Seed binary's identity. Seed is the successor
// system chartered in SEED-NEXT.md, incubating under next/ per
// docs/next-build-plan.md section 0. The protocol version is the wire seam
// (charter Part II section 1): genesis names it, every event carries it, and
// a mismatch refuses with a distinct exit code.
package version

// Name is the binary name: the successor claims the name (SEED-NEXT.md).
const Name = "seed"

// Version is the module version during incubation: pre-release until
// spin-out cuts the first tagged release.
const Version = "0.0.0-dev"

// Protocol is the protocol version minted at genesis (build-plan fixed
// default). The bump discipline is recorded in next/spec/protocol.md.
const Protocol = "seed/0"

// Seed1 is the protocol version that activates the actor keyring
// semantics (next/spec/actors.md; plans/os-52a2d688.md). Chains reach it
// through system.protocol.upgraded; Protocol stays the genesis default
// until a recorded decision moves it.
const Seed1 = "seed/1"

// Supported lists the protocol versions this build verifies and appends:
// the genesis default plus every later version it implements. Verifiers
// and admission points seed their default supported sets from it, so a
// chain that upgrades to a version this build implements keeps verifying
// without per-caller configuration.
func Supported() []string { return []string{Protocol, Seed1} }
