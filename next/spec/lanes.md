# lanes.md — role definitions whose claims are decidable

> Status: v0, normative for `next/**`. Authority: [`SEED-NEXT.md`](../../SEED-NEXT.md)
> Part II §11 "Lanes" and Part III.J; [`docs/next-build-plan.md`](../../docs/next-build-plan.md)
> Phase 9 item 1; plan `plans/os-cf1c9688.md`. Implemented by
> `internal/lane`, `internal/loopverb`, `next/lanes/**`, and the
> `seed lane` read surface.

## What a lane is

Six lanes, each a **role** — grants plus conventions — and never a
binary: any qualified tuple can staff any lane it holds grants for.
The charter's six (§II.11) are `dispatcher`, `planner`, `implementer`,
`verifier`, `curator` and `maintenance`.

A lane is **a manifest plus an ordered list of prose fragments**:

- `next/lanes/<lane>.json` — the declarations, read by a validator.
- `next/lanes/fragments/**` — the conventions, read by an agent.

`seed lane show <name>` resolves the fragments by concatenating them
**in the order the manifest declares**, and `seed lane list` names the
six.

## Why the split, and why the order is declared

Composability is the point. The one-inbox doctrine, the orienting read,
the deliberate-exit rule and the liveness rule are written **once** and
composed by every lane that needs them, so a change to a shared
convention cannot drift between six copies. A convention that appears
in six files is a convention with six versions.

The order is **declared, never inferred**. III.J says *ordered*
fragments, and a resolution whose order comes from a directory listing
is a resolution that changes when someone renames a file.

Manifests are data and fragments are prose, kept in separate files
rather than as frontmatter on one: frontmatter would make a fragment
both a document and a record, and the ordered-composition rule would
then have to say whose frontmatter wins. A fragment is only ever prose,
so the merge question never arises.

Manifests are **JSON**, like every other machine-read file under
`next/` (the port schema, the projections, the packets). The plan
proposed YAML for readability; `next/` carries no YAML parser and a
deliberately small dependency set, and the reason the plan gave for
separating declarations from prose is served identically by JSON.

## The four obligations, as fields

`docs/next-build-plan.md` Phase 9 item 1 binds four obligations onto
every lane's fragment. They are **declared fields**, not paragraphs,
because a validator can find a field and cannot find a paragraph. Prose
may restate them; validation reads the fields.

| field | obligation |
| --- | --- |
| `orients_from` | the single position-stamped read on wake, in the posture the lane works in |
| `acts_through` | the loop acts the lane performs, never the raw append seam |
| `liveness_from` | which of the lane's own work steps emit observations |
| `inbox` | push channels wake, position-stamped reads convince |

## Every check consults an authority elsewhere

This is the whole design. `internal/lane` holds **no policy**: a
hand-written list of capability or verb names inside it would be the
drift it exists to prevent.

| check | authority |
| --- | --- |
| `grants` name real capabilities | `keyring.Capabilities()` |
| the lane's grants **intersect** what each act's verb accepts | `keyring.AcceptedCapabilities` |
| `acts_through` names real loop acts | `internal/loopverb` |
| `orients_from` is the situation read, with flags it takes and exactly one posture arm | `lane.SituationFlags`, pinned to the CLI by drill |
| fragments exist, and none is orphaned | the filesystem, against the manifests |
| resolution is deterministic | resolving twice, byte-compared |

**The relation is intersection, not containment.**
`AcceptedCapabilities` returns an OR-set — the grant rule consumes it
through `HasAnyCapability`, so any one of the returned capabilities
admits the verb, and most worker verbs return `{claim, operator}`.
Requiring a lane to hold *every* accepted capability would hand
`operator` to every lane that claims or spends, dissolving the
capability separation the charter is built on in the name of checking
it.

## Liveness rides the work

`liveness_from` names the lane's own steps whose execution emits
observations, and every entry must appear in `acts_through`: an
observation is only ever emitted **by a step that does work**. A lane
wanting a bare heartbeat must declare a work verb it does not perform,
and the capability check then asks it for the grant.

The obligation is **conditional on running a loop, and not dodgeable by
declining to**. Four of the six lanes perform no loop act — a verifier
acts through `verdict.rendered`, a dispatcher through `intent.filed`,
neither of which is a loop act — so requiring loop acts of all six
would force four manifests to claim work they never do. Instead: a lane
that declares acts must declare where its liveness comes from, and a
lane cannot escape by declaring none, because **holding the `claim`
capability means it claims, and claiming is a loop act**. The grant it
already declares decides whether the obligation applies.

**And the declaration is now enforced, not merely compared.** 1a could
only check two labels against each other and said so here: it could not
show the named step actually emits, because nothing executed. Phase 9
item 1c's loop (`internal/loop`, [`loop-verbs.md`](loop-verbs.md))
closes that:

- The loop emits an observation as a **side-effect of a declared
  liveness act that succeeded** — never as a step of its own. The set it
  emits for IS `liveness_from`, read from the manifest at run time
  rather than carried as a second list, so an act named there emits and
  an act not named there does not.
- The stream is keyed to the lane's own actor and to **the fence its
  orienting read reports**, which is exactly how `internal/obs` keys
  what it classifies. A stream written under any other actor or fence
  is invisible to the classifier and therefore useless as liveness, so
  the drills sample that exact key rather than merely counting lines.
- A **refused** act emits nothing. Otherwise a lane wedged at a
  boundary would look busiest exactly when it is most stuck, and the
  classification the maintenance reap depends on would read failure as
  progress.
- The loop reaches no liveness-only surface, asserted as a property of
  what it invoked: every call it makes is either one of the two reads it
  orients from or an act its manifest declares.

Each of those is mutation-checked — emitting per act rather than per
declared act, keying by a constant fence, and emitting on a refused act
each fail their drill.

Fragment prose is additionally swept for an instruction to run a bare
`seed obs emit`, since a fragment could tell an agent to heartbeat
without declaring it. That sweep **is** a spelling rule and is treated
as one: a second line of defence, never the argument.

## One posture, or the read is not the view

`orients_from` names **exactly one** of `--ledger` or `--remote`, and
validation enforces the exclusive-or rather than a single required flag.
The reason is not symmetry:

`claim take` is **remote-only**. `claim.taken` is the one exclusive act,
and only the push round-trip can order two rivals, so the local path
refuses it outright. Until Phase 9 item 1c, `offer list` and `situation`
bound `--ledger` alone — so in the only posture where a lane could
claim, it could neither poll nor orient. A loop there would read a local
copy that nothing refreshes and call its position authoritative, which
is precisely the staleness the exclusive act exists to prevent.

Both surfaces now take the pair, and the six shipped manifests declare
the remote posture: the one a lane that claims can actually orient in.
A read naming neither has no ledger to derive from; one naming both
cannot say which view its position stamps. Both refuse.

The loop could instead have read the remote-materialized store through
internal packages, since it is a library in the same module. It must
not: every manifest declares `seed situation …` as its orienting read,
and a loop that oriented by internal call would make that declaration a
fiction and reopen the drift these manifests exist to close.

## The dispatcher's posture is an allowlist

The dispatcher touches the most untrusted text in the system and runs
with least standing capability. Its manifest declares its permitted
grants and validation checks the set is **exactly** that allowlist —
`dispatch` alone in v0.

A blocklist would be the wrong shape: it must be extended every time a
capability is added, and one nobody thought to exclude is admitted by
default. That is not hypothetical — an earlier draft checked "holds no
authoring, verdict or sealing grant", which admits `operator`, the
strongest capability in the keyring.

III.J's second row is **half met** by this: the standing claim is
checkable here, and the hostile-corpus suite proving the dispatcher's
*input handling* is Phase 9 item 1b's.

## The loop-verb registry

`internal/loopverb` is the one authority for which loop acts exist and
what each appends, with two consumers: the CLI dispatch and this
validator. It existed nowhere before — the acts were `case` arms inside
`cmd/seed`, in package `main`, which nothing can import — so any second
consumer would have had to write the seven names down again.

It carries no policy. Whether an act admits at a position is the
admission boundary's answer; which capabilities it accepts is the
keyring's.

## Surfaces

- `seed lane list [--lanes <dir>]` — the six, with grants and fragment counts.
- `seed lane show <name> [--lanes <dir>]` — the declarations plus the resolved prose.
- `seed lane validate [--lanes <dir>]` — every check; findings name the lane, the field, and what refused.

The reads a lane orients through take the posture pair as well:
`seed situation` and `seed offer list` each accept `--ledger <dir>` xor
`--remote <repo> [--ref <ref>] [--state <dir>]`.

All three are read-only and idempotent: they open no ledger, mutate
nothing, and journal no attempt, because a read is not an
admission-boundary attempt ([`refusals.md`](refusals.md)). They carry
**no position stamp**: a resolved role derives from checked-in files
rather than from the ledger, so there is no position they could
honestly cite, and nothing is ever written back — a resolved role on
disk would be a second copy that can go stale, which is the failure the
ordered fragment list prevents.

### Exit code

| code | name | meaning |
|---|---|---|
| 26 | `lane_invalid` | a checked-in lane manifest makes a claim the tables refuse: a grant outside the vocabulary, an act whose accepted capabilities the lane does not hold, an empty or non-work liveness source, a missing or orphaned fragment, a dispatcher grant outside its allowlist. Distinct from `posture_invalid`, which judges a deployment's posture declaration rather than a role definition |

## Conformance mapping

- III.J "Role definitions exist for all six lanes as grants +
  conventions, composable from ordered fragments, resolved and checked
  by validation" — `next/lanes/**`, `lane.Resolve`, `lane.Validate`,
  and the drill asserting the shipped set validates clean.
- III.J "The dispatcher runs with least standing capability" — the
  allowlist above, drilled against `operator` explicitly. The paired
  injection-conformance half is Phase 9 item 1b.
