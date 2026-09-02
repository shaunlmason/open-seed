# verdicts.md — the verdict pipeline's first half

> Status: v0, normative for `next/**`. Authority: [`SEED-NEXT.md`](../../SEED-NEXT.md)
> Part II §8 "The verdict pipeline" and §6 (the verifier executes specs
> in a sandbox with declared, minimal capability); conformance III.G
> rows 3–5; [`docs/next-build-plan.md`](../../docs/next-build-plan.md)
> Phase 6 item 1; plan `plans/os-f6d2c267.md`. Implemented by
> `internal/verdict`, `internal/artifact`, the `verdict` admission
> rule, and `seed verdict receipt|render|check`. The reconciliation
> chain and divergence detection landed with 6.2
> ([`reconciliation.md`](reconciliation.md));
> sealed checks 6.3; red-verdict lockout and the operator override verb
> 6.4; rubrics and L2/L3 are Phase 10/11.

## The event

`verdict.rendered` is a **fact, not a transition**: it admits only on a
subject whose folded state is `review` (elsewhere it refuses exit 3,
the illegal-verb-in-state posture), changes no state, and `done` still
arrives only through `merge.observed`, behind the full chain rule
([`reconciliation.md`](reconciliation.md)). Its payload is strict:

```json
{"verdict": "pass" | "fail", "receipt": "<sha256 hex>", "submission": "<position>", "independence": "L1" | "L2" | "L3", "tuple"?: {…}}
```

Unknown keys refuse; `verdict` admits only the two literals;
`independence` is the level the record supports, exactly (the section
below): the literal `"L1"` alone before `seed/4`, the ordered
vocabulary from it. `tuple`, from `seed/4`, is the verifier's declared
runtime configuration ([`qualification.md`](qualification.md)'s five
fields, strict), present when the verifier declared one and refused
before `seed/4`. `submission` is the chain position of the
`submission.made` that put the subject in its current `review` state
(the fold records it): a verdict is bound to the submission it judges,
and citing any other position refuses. `receipt` is the SHA-256 digest
of the receipt's JCS bytes (below).

## Independence (L1) at admission

The **implementing-key set** of a contract is every fingerprint that
ever signed a `claim.taken` on it (the current holder and every prior
claimant, the fold's claim facts) plus the signer of the bound
`submission.made`. A `verdict.rendered` whose signer is in that set
refuses exit **17 `not_independent`**, naming the fingerprint and its
implementing role. This is deliberately distinct from exit 14
`out_of_grant`: the signer may hold a perfectly good `verdict` grant
and still be disqualified on this one contract — capability is global,
independence is per-contract.

The capability row is `verdict.rendered` → `verdict` **only**, with no
operator fallback (the one such row in `actors.md`): III.G names
operator override its own attributable verb, never a disguised verdict,
and that verb lands with 6.4. A governance root that wants to judge
holds an explicit `verdict` grant, and the independence check applies
to it like any signer.

Because `review` is outside `in_progress`, no fence is active: a
verdict citing one refuses under the established fence rule (a fence
dies with its claim window).

## Independence levels (L1, L2, L3) from `seed/4`

The charter's levels (§7, "Independence is failure-domain separation")
are **ordered**, achieved is **computed from the record, never
asserted**, and the tier table declares the minimum per tier
([`tiers.md`](tiers.md), the `independence` column: `trivial` L1,
`standard` L1, `critical` L2, the strictest row L3). Any achieved level
at or above the requirement satisfies it, which is III.G's "L2 or L3"
for high-consequence tiers. From `seed/4` (plans/os-99829835.md):

- **L1** holds for every admitted verdict: the key and workspace
  disjointness above.
- **L2** holds when the verifier's declared `tuple` differs from the
  claim window's admitted `run.started` declaration in model
  **provider or family**, or in **harness name**: the charter's
  "different model family or provider, different harness image".
  Principal, tool policy, environment and versions do not count; a
  three-part model (`<provider>/<family>/<version>`) compares provider
  and family, a two-part one family alone, so two providers serving
  one family through one harness are L2. A window with no admitted
  declaration supports no L2.
- **L3** holds when the record proves deterministic-first verification
  on a distinct path: the acceptance is executable and gated, and the
  cited receipt **reproduces**, `verdict check`'s recomputation from
  the verifier's own checkout yielding the cited digest. That is the
  record-backed determinism predicate: a nondeterministic command, a
  model call that answers differently, or an evidence path the
  implementer could shape yields a different transcript digest and
  fails to reproduce. Distinct acquisition is the workspace rule
  above. The boundary checks the executable-and-gated half (it runs
  nothing); recomputation and reconcile's evidence grade check the
  reproduction. What the record cannot prove is stated: the `exec`
  profile declares the network rather than denying it, and a
  deterministic command that still reaches a model is caught only when
  the model disagrees with itself; the denying profile is Phase 13's
  adapter.

**The verifier declares its tuple at render, as the supervisor declares
the worker's at start.** `seed verdict render --principal <p> --model
<m> --tool-policy <t>` fills `harness` and `environment` from the
workspace adapter it verifies in and never invents the three it cannot
know. The declaration is a declaration, not a report: what the
verifier ran as is what its operator says, exactly as the worker's is;
the qualification machinery that would prove it is
[`evals.md`](evals.md) applied to verifier keys, Phase 10 item 4's.

**Enforced at admission, at render, and along the merge chain.** The
verdict rule requires `independence` to be a member of the vocabulary
and to **equal** the level the record supports (a claim below the
computed level would underreport the audit data the level exists to
produce; one above it asserts what the record does not show), and the
level to satisfy the tier's requirement, read through `TierGates`. A
level short of the tier refuses exit **17 `not_independent`** with the
refining code **`level_short`** ([`envelope.md`](envelope.md)): the
family's answer, "this verifier cannot judge this contract", stands,
and the word says it is the configuration rather than the key. `seed
verdict render` computes the achieved level from the same facts before
drafting, so the client never drafts a doomed verdict, and on a tier
the record cannot satisfy without a declaration it refuses at usage
naming the three flags. The fold records the level and the tuple, and
**the merge chain reapplies both**: the boundary `merge.requested`,
`merge.observed`, `contract.returned` and the red-verdict lockout
consult refuses a folded verdict whose recorded level the record does
not support or which is short of the subject's tier, so a raw-pushed
`critical` verdict at `L1` cannot be laundered into `done`;
[`reconciliation.md`](reconciliation.md)'s `independence_unverified`
surfaces the same two conditions and the evidence-grade reproduction,
a sealed subject's under an identity able to unseal (`seed reconcile
--key`, the maintenance actor's key), never skipped silently.

Before `seed/4` the level is the literal `L1` and a declaration has
nowhere to go; a `seed/3` chain keeps `seed/3`'s judgment
([`protocol.md`](protocol.md)).

## The verifier workspace

The verifier executes in **clean per-run isolation**: a detached local
clone of the repository at the submission head under a fresh unique
temp dir, with the origin remote removed and objects **copied, never
hard-linked** (`--no-hardlinks`: a same-filesystem local clone
otherwise hard-links loose object files, so a hostile spec command
overwriting one through the shared inode would corrupt the parent's
object store) — and deliberately not a `git worktree` checkout, whose
`.git` link shares the parent repository's refs and object store and
would hand a hostile spec command `git update-ref` reach back into the
host. Both holes are drilled. Parallel runs never collide (unique
dirs); cleanup fires pass or fail. The clone carries auto-gc disabled
in its own config from the moment it exists (`gc.auto`,
`gc.autoDetach`, `receive.autoGC`, written between the clone and the
checkout), so nothing the engine made mutates after the engine exits:
a collector git detaches after a checkout would otherwise race the
cleanup that follows it.

The charter's "sandbox with declared, minimal capability" lands as a
**runner capability profile, declared in the receipt**. v0 ships the
`exec` profile: every command runs via `sh -c` inside the workspace
with a scrubbed environment (an explicit minimal `PATH`; `HOME`,
`TMPDIR`, and `GIT_*` pointed inside the workspace; nothing inherited
from the invoking process), no path back to the parent repository, and
a per-command wall-clock timeout with process-group kill. The profile
declares `network: unrestricted` honestly — portable no-root execution
cannot deny the network, and a pretended boundary would be worse than a
declared one. The runner is an interface: a namespaced or containerized
profile slots in at the executor-adapter seam (build plan Phase 7
item 3, Phase 12 hardening) without touching verdict logic, and every
receipt names the profile its transcripts ran under.

**The verifier's inputs are enumerable and exclusively self-executed or
self-read** (III.G row 4): the submission packet's anchors are used
only to *name* the range; every hash, diff, inventory, and transcript
is recomputed from the verifier's own checkout and the ledger. Nothing
in the receipt repeats an implementer claim.

## The receipt

The receipt is JCS-canonicalized JSON; its digest is the SHA-256 of the
canonical bytes; the body is stored content-addressed in the artifact
store (`internal/artifact`, filesystem-rooted under
`next/var/artifacts/sha256/<digest>`; the build plan's git-addressed
`refs/seed/artifacts` push is deferred, recorded in the decision log).

```json
{
  "contract": "<subject>",
  "merge_base": "<full sha>",
  "head": "<full sha>",
  "plan": {"path": "<path>", "sha256": "<hex>"} | null,
  "diff_sha256": "<hex>",
  "files": ["<changed path>", ...],
  "transcripts": [{"cmd": "<command>", "exit": 0, "output_sha256": "<hex>", "output_bytes": 0}, ...],
  "environment": {"os": "<GOOS>", "arch": "<GOARCH>", "go": "<version>", "runner": "exec"},
  "commitment": "<hex, sealed subjects only>",
  "sealed_transcripts": [{"cmd": "...", "exit": 0, "output_sha256": "<hex>", "output_bytes": 0}, ...]
}
```

`merge_base` and `head` resolve from the submission packet's mandatory
`base` range (`packets.md`) to **full commit SHAs, stored immutably**;
a head that does not descend from its merge-base refuses before
checkout. A verdict attests exactly the `{merge_base, head,
diff_sha256}` triple and nothing else — never a ref or branch name.
Branches are forge state the ledger cannot see: "the merge landed code
other than the attested head" is precisely the verdict/merge
divergence 6.2's reconciliation detects by comparing `merge.observed`'s
forge fact against the attested head. `plan` is the approved plan blob
hashed **at the merge-base** (null for a planless trivial-tier
contract); `diff_sha256` and `files` recompute from `git diff` in the
workspace; transcripts carry each command, its exit, and the digest and
byte count of its combined output — never inline bytes, so receipts
stay bounded. **Verification recomputes everything from the submission
head and fails on mismatch** — a first-class verb (`seed verdict
check`, refusing exit **21 `receipt_mismatch`**), not only a test.
Check verifies two things and names what failed: the cited artifact is
retrievable intact from the store (the evidence a verdict points at
must survive verbatim), and the fresh recomputation reproduces the
cited digest. It runs in **every post-submission state**, not only
`review`: reconciliation needs it exactly after `merge.observed` has
moved the contract on, and the fold retains the bound submission —
only `receipt` and `render` stay review-gated.

On a **sealed** subject ([`sealed-checks.md`](sealed-checks.md)) the
receipt gains the charter's "visible and sealed check transcripts":
`commitment` is the ledger's salted hash the run unsealed against, and
`sealed_transcripts` the sealed commands' outcomes, run under the same
profile in the same workspace; both are omitted on unsealed subjects,
so every pre-6.3 receipt's canonical bytes and digest are unchanged.
The recompute-and-mismatch guarantee covers them: `verdict check` on a
sealed subject requires `--key` with an identity able to unseal,
decrypts, verifies the commitment, and reruns the sealed commands into
the recomputation — invented sealed transcripts in a raw-pushed
receipt recompute differently and fail at exit 21, an identity outside
the recipient set refuses exit **23 `not_recipient`**, and a broken
seal (missing or tampered ciphertext, commitment mismatch, or an
empty-checks envelope) refuses exit **22 `seal_broken`**. There is no
silent partial verification of a sealed subject. `verdict receipt`
stays the visible-half preview; the render is the authoritative
sealed run.

## Gate-before-run

Before any command runs, the verifier reads the contract's folded
acceptance (`acceptance.md`): `executable: true` without gate evidence
refuses exit **18 `ungated`** with nothing executed — the
gate-before-run half III.F row 1 assigns to verdicts, consuming the
projection's `gated` flag. `executable: true` whose gated body yields
no parseable commands refuses exit **19 `spec_unrunnable`**: the
declaration promised runnable content, the body carries none, and
silence must never decide — a vacuous pass is not a pass.
`executable: false` runs nothing and the receipt carries an empty
transcript list.

The command grammar is the plan grammar: the acceptance body's
"validation commands" marked section, extracted by the same walk
`internal/plan` lints (`plan.Commands`), one transcript entry per
command line.

## Rendering derives from the transcripts

`seed verdict render` computes the receipt fresh and **derives the
permissible verdict from the transcripts it just executed**: any
nonzero transcript exit forbids `--verdict pass`, refusing exit **20
`checks_red`** with the failing command named; `fail` is always
renderable; a prose-only spec
(no transcripts) leaves pass or fail the verifier's explicit judgment.
The enforcement seam is the render verb because the verifier is the
only party holding the transcripts it just ran — admission can neither
re-run commands nor read a verifier-local artifact store (the
cooperative-posture precedent: the client refuses to draft doomed
work). A raw-pushed pass-over-red is not silent: `seed verdict check`
recomputes from the submission head and goes red, and that mismatch is
6.2 reconciliation input.

**The red-verdict lockout** (6.4, plans/os-d2497eb7.md): once an
**authenticated** fail verdict judges the bound submission, rendering
`pass` refuses — at admission and at render, exit **25
`red_locked`** — until a **new submission** arrives through
`contract.returned`, a fresh claim, and a resubmission
([`lifecycle.md`](lifecycle.md)). Only boundary-validated fails lock
(verdict grant plus implementing-key disjointness): the tolerant fold
records any well-shaped raw verdict, so the lockout scans the whole
submission window and an unauthenticated fail locks nothing,
authorizes nothing, and surfaces as `verdict_unverified`. Fail
restatements stay renderable, and no implementer-held lane can clear
the lockout. The operator's escape hatch is `merge.overridden`
([`reconciliation.md`](reconciliation.md)): it clears the merge path,
never `verdict.rendered(pass)` — an override substitutes for a
verdict, it does not manufacture one. Sealed checks ride the same rule: a red
sealed transcript forbids pass exactly like a visible one, and render
on an **above-trivial subject with no commitment refuses exit 24
`unsealed`** — the "contracts carry sealed checks" gate, enforced at
the verifier boundary where checks run; the trivial tier is exempt
(`sealed-checks.md`).

## Visibility

The contracts view is unchanged by 6.1: surfacing verdicts, submissions,
and divergence in projections rides 6.2's reconciliation work. The fold
records `submission {position, signer}` per contract for the binding
and disjointness checks above.

## Conformance mapping

- III.G row 3 (verdict-granted keys provably disjoint from every
  implementing key; override its own verb) — the `verdict` capability
  row, the L1 independence rule with exit 17, the operator-fallback
  omission; the override verb itself is 6.4.
- III.G row 1 (independence levels L1–L3 defined, declared per tier,
  enforced at verdict time, recorded in the verdict; high-consequence
  tiers require L2 or L3) — the levels section above, the tier table's
  `independence` column, the verdict rule's equality and tier checks
  with `level_short`, the fold's recorded level and tuple, the merge
  chain's reapplication, and `independence_unverified`.
- III.G row 4 (clean per-run isolation; parallel verdicts never
  collide; cleanup fires pass or fail; enumerable, self-executed
  inputs) — the workspace, the runner profile, and their drills.
- III.G row 5 (receipts bind contract id, plan hash at merge-base,
  diff hash, inventory, transcripts, environment fingerprint;
  verification recomputes and fails on mismatch) — the receipt schema
  and `seed verdict check`; sealed-check transcripts join in 6.3.
- III.G rows 1–2 (the reconciliation chain and divergence) — 6.2, with
  the verdict's immutable head attestation as its comparison anchor.
- Part II §6 (sandbox with declared, minimal capability) — the runner
  profile, declared per receipt, with the stronger boundary at the
  Phase 7 adapter seam.

## Rendering reaches both postures

`seed verdict render` takes `--ledger` or `--remote`. It was local-only
until Phase 9 item 4, which meant a fleet's verifier lane could not act
against the shared ledger its workers claim on: the terminal half of
the contract lifecycle had no reachable surface in the deployment the
charter's fleet mode describes ([`modes.md`](modes.md)).

The payload's `submission` is a chain POSITION, so on the remote path
it is re-derived against each refreshed tip and REFUSED on a change
rather than re-pointed — a verdict bound to whatever submission happens
to be current is the laundering shape, and binding is the whole point
of the field. The receipt is not re-derived: it is content-derived from
the repository rather than from the ledger view, so a moving tip cannot
change it.
