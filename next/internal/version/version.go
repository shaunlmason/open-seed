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

// Seed2 is the protocol version that activates runtime-tuple semantics
// (next/spec/qualification.md; plans/os-8e53ffd9.md): actor.granted may
// cite a tuple, run.started must declare one, and drift refuses as
// out_of_grant. A seed/1 validator strictly decodes actor.granted as
// {capability} and so judges a tuple-bearing grant differently, which
// is next/spec/protocol.md's bump trigger; records at seed/1 positions
// keep seed/1's judgment.
const Seed2 = "seed/2"

// Supported lists the protocol versions this build verifies and appends:
// the genesis default plus every later version it implements. Verifiers
// and admission points seed their default supported sets from it, so a
// chain that upgrades to a version this build implements keeps verifying
// without per-caller configuration.
func Supported() []string { return []string{Protocol, Seed1, Seed2} }

// Activated reports whether the semantics seed/1 introduced (the actor
// keyring, the lifecycle fold, budgets, offers) are active at a record
// carrying version v: seed/1 and every later version this build
// registers. A named list, never an ordering, so a version this build
// has not registered activates nothing however it would sort
// (plans/os-8e53ffd9.md D8). tuple.Applies is the narrower gate for
// what seed/2 added on top.
func Activated(v string) bool { return v == Seed1 || v == Seed2 }
