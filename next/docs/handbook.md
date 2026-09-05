# The Seed operator handbook

This is the book a team adopts Seed from. It says what to type and what
will happen. It does not restate the charter (`SEED-NEXT.md`); it gets a
deployment running and names the one read each lane orients from. Every
command below is a fenced block a drill exercises, so a renamed verb
fails this handbook rather than the reader.

Seed coordinates its own development through a checked-in ledger of
signed events. There is no server to sign up for and no account: the
engine is pinned, the ledger is a git ref, and the CLI is the whole
interface. The generated companion tables live under
[`generated/`](generated/) — the lifecycle, the capabilities, the exit
codes and the per-lane worker docs, each rendered from the tables the
machinery reads, never from prose.

## 1. Install

The engine is pinned by the repository; you run it through `scripts/seed`,
which bootstraps the exact version the ledger was built with. No account,
one command:

```sh
seed version
```

Seed itself is released by `.github/workflows/seed-release.yml`, run by
the operator at the distribution step (`docs/next-build-plan.md` §5)
from the default branch, with the version (semver, without the `v`) as
its one input. The workflow builds `seed` and `seed-admit` for linux,
darwin and windows on amd64 and arm64 with the version stamped, writes
the archives and `checksums.txt`, and only then mints the tag
`seed/v<version>` at HEAD in-runner, so a failed build leaves no tag; it
creates the release as a draft, attests `checksums.txt` with build
provenance, and publishes the draft last, so a failed attestation leaves
a draft rather than a public release. A re-run for the same version on
the same commit resumes an incomplete cut; a tag that points at another
commit, or a release already published, refuses. One precondition is the
operator's and lives outside the tree: the job runs in the
`seed-release` GitHub environment, which must exist with a deployment
branch policy that allows the default branch alone (and required
reviewers, if wanted) before the first dispatch, because the
environment's policy is what keeps a branch's edited copy of the
workflow from the release privileges; the job's own ref guard only
restates that rule. To verify an archive before running it, check its
sha256 against `checksums.txt` and its provenance with
`gh attestation verify <archive> --repo <owner>/<repo>`. Until the first
release is cut the binary is built from source (`go run ./next/cmd/seed`)
and `seed version` prints the pre-release default.

## 2. Initialise a ledger

The genesis event is signed by the operator key and always joins the
governance root. A non-empty ledger refuses.

```sh
seed init --ledger ./ledger --key ./operator_ed25519
```

Check the deployment's declaration before anything else. `doctor` reads
no ledger; it states the posture and, for cooperative, the consequence
verbatim.

```sh
seed doctor --config seed.json
```

A preseed declares the vocabulary, the teams (naming shipped manifests),
and the protected surface; `preseed check` verifies the declaration is
whole.

```sh
seed preseed check --config seed.json --lanes lanes
```

## 3. The three postures

- **enforced-self-hosted** — a server you run executes the `pre-receive`
  hook (`seed-admit`); a push the boundary would refuse is rejected at
  the server. The security invariant holds against a hostile credential.
- **enforced-forge-hosted** — a managed forge runs the admission service;
  the boundary is enforced without a server you host.
- **cooperative** — no server-side enforcement: every writer
  self-validates. `doctor` prints the consequence in full — the security
  invariant does not hold, and protocol rules are advisory against a
  hostile credential. Run this only where every writer is trusted.

`doctor` reports the platform and which postures are available on it; a
bare checkout with no server runs cooperative or forge-hosted, never
enforced-self-hosted.

## 4. Enrol actors

An actor is a key the operator enrols and grants. `kind` records whether
the key is an `agent` or a `human`; it is provenance, not permission —
permission is the grant. Enrol, then grant each capability the lane's
manifest declares:

```sh
seed ledger append --ledger ./ledger --key ./operator_ed25519 --verb actor.enrolled --subject <fingerprint> --payload '{"key": "<hex>", "kind": "agent", "name": "impl"}'
```

```sh
seed ledger append --ledger ./ledger --key ./operator_ed25519 --verb actor.granted --subject <fingerprint> --payload '{"capability": "claim"}'
```

## 5. The lanes

Each lane is a manifest: its grants, and the SINGLE position-stamped read
it wakes on (`orients_from`). List them and read one — the generated
worker doc is the same resolution:

```sh
seed lane list --lanes lanes
```

```sh
seed lane show implementer --lanes lanes
```

The one read a working lane orients from is `situation`:

```sh
seed situation --ledger ./ledger --key ./impl_ed25519
```

## 6. The loop and its deliberate exits

The loop is a library (`internal/loop`), not a verb: a lane polls, orients
from its one read, and acts through the CLI's own verbs. There is
deliberately no `seed loop run` — the work step is yours. A window opened
by `claim take` ends only through one of four deliberate exits:
`submission make`, `claim release`, `claim park`, or `claim reaped`
(the maintenance pass's). A window that ends any other way is a silent
abandonment the audit names.

The verifier renders a receipt through the real machinery; a distinct key
from the claimant's is required (independence):

```sh
seed verdict render --ledger ./ledger --subject c-1 --repo ./work --key ./verify_ed25519 --verdict pass
```

The observer records the merge that ends the contract at `done`:

```sh
seed merge observe --ledger ./ledger --subject c-1 --key ./observer_ed25519 --merged <sha> --pr pr/1
```

## 7. Escalation and decisions

Any lane can raise `blocked(needs-you)`; raising grants nothing. A raised
contract leaves blocked only through the operator's recorded decision.

```sh
seed decision record --ledger ./ledger --subject c-1 --key ./operator_ed25519 --decision proceed
```

## 8. Maintenance

The maintenance pass reaps orphaned claims (an interrupt, a wedge, or a
revoked holder), files defect contracts, and runs on a declared instant —
admission itself reads no clock:

```sh
seed maintain run --ledger ./ledger --key ./maintenance_ed25519 --as-of 2026-09-03T00:00:00Z
```

## 9. Migration from open-seed

Import the predecessor's export against its source clone and seed-anchor
tag into an empty ledger, in one operator-signed pass:

```sh
seed import --from-open-seed ./export.json --source ./open-seed --ledger ./ledger --key ./operator_ed25519
```

## 10. The drills

`make check` is the backpressure command: it runs `gofmt`, `vet`, `build`,
the suites behind the coverage gate, the performance gate, the preseed
check, and the governed-docs drift check. Keep it green.

```sh
seed docs check --root .
```

`docs check` runs two stages. The drift stage regenerates the governed
documents and diffs them, failing `docs_drift`. The citation stage then
holds every relative markdown link in the tree to the tree, failing
`broken_citation` and naming the file, the line and the target: a
document that cites a path which is not there renders broken on the
forge, and nothing else catches it. Both stages share exit 28. Code
spans and fenced blocks are masked before any link is read, so a
link-shaped regex in prose is not a citation; external URLs are out of
scope, because resolving one needs the network and this command must
stay offline and deterministic; and `plans/` is out of scope, because a
plan file changes only through its own single-file plan PR, so a
refusal there would demand a fix no branch carrying this gate may make.
Dangling citations under `plans/` are repaired by a plan PR.

**The conformance report.** Part III of the charter is a checked-in
table, `next/spec/conformance.json`, one row per charter criterion with
the status the phase exit records gave it; `seed docs generate` renders
it as `next/docs/generated/conformance.md` under the same drift check,
and `seed doctor --config <declaration> --repo .` reports it at the
declared posture: the counts by status, every row not yet met by
pillar and row with its status, the enforced-only rows a cooperative
deployment must document as not holding for it, and `complete` only
when every row is met at an enforced posture
([`conformance.md`](../spec/conformance.md)).

## 11. Simulation mode

The whole system runs end to end against synthetic intents with a mock
executor and zero credentials — no forge, no model, no network beyond a
local bare git remote. It drives every lane to done and audits the ledger
from the chain alone:

```sh
seed simulate --lanes lanes --intents 3 --posture cooperative
```

The accelerated clock runs a week-long backlog, the reporting instant
advancing while admission reads no clock:

```sh
seed simulate --lanes lanes --intents 5 --days 7 --posture enforced-self-hosted
```

## 12. The promotion gate

Promotion is two human cutovers (this repository's own development
moving to Seed, then Seed becoming what new users clone), gated by the
seven criteria in `docs/next-build-plan.md` §5. `next/docs/promotion.md`
is what the operator reads there: each criterion's status and the
drills on `main` that back it, the shadow run proposed as a protocol,
the cutover and its rollback written down, the two cutovers named as
the reserved decisions they are, and the ledger of the III.R
measurements the shadow run supplies. `make check` holds every drill
the packet cites to the tree, so the packet cannot claim what the tree
no longer holds.
