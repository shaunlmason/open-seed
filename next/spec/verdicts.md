# verdicts.md — the verdict pipeline's first half

> Status: v0, normative for `next/**`. Authority: [`SEED-NEXT.md`](../../SEED-NEXT.md)
> Part II §8 "The verdict pipeline" and §6 (the verifier executes specs
> in a sandbox with declared, minimal capability); conformance III.G
> rows 3–5; [`docs/next-build-plan.md`](../../docs/next-build-plan.md)
> Phase 6 item 1; plan `plans/os-f6d2c267.md`. Implemented by
> `internal/verdict`, `internal/artifact`, the `verdict` admission
> rule, and `seed verdict receipt|render|check`. The reconciliation
> chain (`merge.requested`/`merge.observed` piping, divergence) is 6.2;
> sealed checks 6.3; red-verdict lockout and the operator override verb
> 6.4; rubrics and L2/L3 are Phase 10/11.

## The event

`verdict.rendered` is a **fact, not a transition**: it admits only on a
subject whose folded state is `review` (elsewhere it refuses exit 3,
the illegal-verb-in-state posture), changes no state, and `done` still
arrives only through `merge.observed`. Its payload is strict:

```json
{"verdict": "pass" | "fail", "receipt": "<sha256 hex>", "submission": "<position>", "independence": "L1"}
```

Unknown keys refuse; `verdict` admits only the two literals;
`independence` admits only `"L1"` in v0 — the level vocabulary widens
when Phase 10 declares levels per tier, and the verdict records the
level actually achieved. `submission` is the chain position of the
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

## The verifier workspace

The verifier executes in **clean per-run isolation**: a detached local
clone of the repository at the submission head under a fresh unique
temp dir, with the origin remote removed — deliberately not a
`git worktree` checkout, whose `.git` link shares the parent
repository's refs and object store and would hand a hostile spec
command `git update-ref` reach back into the host. Parallel runs never
collide (unique dirs); cleanup fires pass or fail.

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
  "environment": {"os": "<GOOS>", "arch": "<GOARCH>", "go": "<version>", "runner": "exec"}
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
check`, refusing exit **21 `receipt_mismatch`** naming both digests),
not only a test.

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
