# Platform parity

Status: normative for `next/**`. Charter authority: SEED-NEXT.md
III.I row 3 (a verb exists on the machine protocol iff it exists on
the CLI) and row 4 (platform parity documented and tested). Plan:
`plans/os-b55e5647.md`.

Parity is a set of tested statements, never an assertion: what each
platform can run is what its drills pass, and what it cannot is
named with the reason.

## The machine surface (normative)

`seed serve` is JSON-RPC 2.0 over stdio, one request per line on
stdin and one response per line on stdout, framed as
`machine-envelope/0`. Every method is a CLI verb and every CLI verb is
a method: both surfaces are drawn from one registry
(`cmd/seed/registry`), the CLI dispatching a group's run function and
the protocol invoking the same function with the request's params as
its argv. A method is `group.subverb` (`ledger.append`, `claim.take`)
or the bare group for a verb that takes flags alone (`situation`,
`doctor`); `seed serve --list` prints the set. The one named carve-out
is `serve` itself: a CLI verb that is the protocol, not a method of it.

Params are the verb's argv: an array of strings verbatim, or an object
whose keys become `--key value` flags (`true` a bare `--key`, an array
the flag repeated, a number its text) with the reserved key `args`
appended verbatim. The result is the `seed-envelope/0` the verb
rendered, byte for byte — the exit code, the machine code, the
position stamp and the affordances included — so a refusal is a
result carrying the failing envelope, never a transport error. JSON-RPC
errors are the transport's own: `-32700` for a line that is not JSON,
`-32600` for a request without `"jsonrpc": "2.0"` and a method,
`-32601` for a method the CLI lacks, `-32602` for params of the wrong
shape. A notification (no `id`) runs and is answered by silence. The
protocol reaches the ledger by no path the CLI does not and holds no
credential the CLI would not: a key, a ledger or a remote is a param,
as it is a flag. `plan`, the one verb that reads stdin, receives an
empty stream under the protocol and takes its body from a file.

`seed-admit serve` (the forge-hosted admission service) is unchanged
and separate: it speaks signed proposals to one ref over HTTP;
`seed serve` is the general verb surface over stdio. Neither adds
semantics; both consume the CLI's.

## Paths (normative)

Every filesystem path is built with `path/filepath`, never with a
literal `/`; slash-joined strings are for refs, URLs and
repository-relative names (a packet ref, a CODEOWNERS line, a registry
path), which git and the spec define with slashes on every platform.
The lint reads the call site: an `os` file call whose argument is
joined with a literal slash or with `path.Join` fails the gate.

## Line endings (normative)

The ledger is LF-only. Every canonical form — a record's JCS bytes, a
card's canonical bytes — is LF, and the segment reader keeps a
carriage return in the line so the record parser refuses it, naming
the carriage return: a segment mangled by `core.autocrlf`, an editor
or a transfer is refused at verification, never silently normalized,
because the bytes a signature covers are the bytes on disk. A
checkout on any platform must keep `next/**` ledgers and fixtures LF.

## Capabilities per platform (normative)

| platform | cooperative | enforced-forge-hosted | enforced-self-hosted |
|---|---|---|---|
| Linux | yes | yes | yes: a git server executes the pre-receive hook; the hook drills run here |
| macOS | yes | yes | yes: the same hook, the same drills |
| Windows | yes | yes | no: no server executes the pre-receive hook on a bare Windows checkout; host the ledger on a POSIX git server or run the other two postures |

`seed doctor` reports `platform`: the OS and architecture, each
posture with its availability and reason, the available list, and
the two rules above. A platform the drills have not run on lists the
enforced self-hosted posture as unavailable with that reason — never
by assertion.

## Tested, not asserted (normative)

CI runs the Go suites on Linux, macOS and Windows (`platform` in
`.github/workflows/check-validate.yml`); the path lint and the CRLF
drill run on each; `make check` and the coverage gate run on Linux.
Every platform-gated skip names its reason (a lint over every test
source refuses a bare `t.Skip()`), so a platform never passes
vacuously. A posture is marked available on a platform when its
drills pass there.

## Conformance

- `cmd/seed/registry`: the table; `cmd/seed/catalog.go`: every verb;
  `cmd/seed/serve.go`: the transport.
- Drills: `TestRegistryMirrorsTheCLIVocabulary` (each group's subverbs
  are the dispatcher's own usage vocabulary, both directions; run
  dispatches through the registry alone), `TestProtocolMethodsAreTheCLIVerbs`
  (`serve --list` equals the registry's derivation; the transport is
  no method), `TestServeReturnsTheCLIEnvelopeVerbatim` (a corpus of
  read, write, refusing and flags-only verbs byte-identical through
  both surfaces; the transport errors; the notification),
  `TestFilesystemPathsUseFilepath`, `TestEverySkipNamesItsReason`,
  `TestCRLFSegmentIsRefusedNotNormalized`, `TestDoctorReportsThePlatform`,
  `internal/platform`'s posture table drill.
