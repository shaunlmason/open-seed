# The obligations projection and the situation read

Seed represents **permission** already: `admit.Affordances`
([`envelope.md`](envelope.md)) answers "what may I do", computed from
the rule set admission enforces. This spec defines the other half —
**obligation**: what is owed on a subject, by whom, since when, and
which verbs discharge it. Design authority: `SEED-NEXT.md` §II.4
(projections) and §II.10, and `docs/next-build-plan.md` Phase 9
item 5, whose loop-completeness criterion this surface exists to
satisfy: a lane that cannot orient or choose cannot run unattended.

Obligation is a **projection over the fold**, never a new authority.
Every fact it reads is already folded — the active claim with its
fence, the bound submission, the standing verdict and merge request,
run starts against run settles, open reservations, and the state with
the position that set it.

## The row

```json
{"subject": "c-1", "kind": "claim.held", "owed_by": "<fp>",
 "since": 12, "discharged_by": ["claim.parked", "claim.released", "submission.made"]}
```

Identity is **`(subject, kind)`** — normative, not incidental: the
situation read's delta names removals by that pair, and a removal
list is only applicable if a client can match rows it already holds.
`owed_by` is a fingerprint, or a `lane:<capability>` name where the
debt belongs to a role rather than an actor (independence forbids
naming the claimant as its own verifier). `discharged_by` is never
empty: an obligation nobody can discharge is an anomaly, not an
obligation, so a kind with no reachable discharger is not emitted.

## Where dischargers come from

**State-shaped** obligations read their verbs from the transition
table — the table already says which verbs leave a state, so legality
is never restated here. **Fact-shaped** obligations close with a verb
that changes no lifecycle state and therefore appears in no table
row; that set is closed and each member cites the spec pairing it
with its fact:

| kind | discharged by | spec |
| --- | --- | --- |
| `run.unsettled` | `run.settled` | [`executors.md`](executors.md) |
| `budget.open` | `budget.settle`, `budget.release` | [`budgets.md`](budgets.md) |
| `submission.pending` | `verdict.rendered` | [`verdicts.md`](verdicts.md) |
| `verdict.unmerged` (unrequested) | `merge.requested` | [`reconciliation.md`](reconciliation.md) |

## The kinds

- **`claim.held`** — an active claim window; owed by its holder;
  discharged by the verbs leaving `in_progress`.
- **`submission.pending`** — a submission no verdict cites; owed by
  the verifier lane.
- **`verdict.unmerged`** — a pass verdict with no observed merge. The
  merge chain is **two events**, so this kind has two shapes: until a
  request cites the verdict the debt is the operator's and
  `merge.requested` pays it; after that the forge fact is the
  observer's to record with `merge.observed`.
- **`run.unsettled`** — an admitted `run.started` whose fence carries
  no `run.settled`. **Position-anchored**: flagged only once the
  subject has taken a subsequent claim window or reached a terminal
  state, because post-close settlement is a valid intermediate state
  and a closed-without-settle predicate would file spurious findings
  mid park or reap flow (the Phase 7 exit's routing).
- **`budget.open`** — an open valid reservation, **wherever it
  stands**: from the reserve until a settle or a release closes it,
  inside the window that opened it and after. Admission gates only
  the reserve on `in_progress` ([`budgets.md`](budgets.md)), so both
  closing verbs stay reachable once the window ends and the debt is
  a debt rather than a maintenance concern. This matters most on the
  failed-verdict retry, where the next claimant is a different worker
  and an unclosed hold would come out of their remaining.

  Owed by **whoever can still discharge it**, which is not always
  whoever opened it and is never merely whoever holds the window: a
  close admits for the reservation's own reserving signer or the
  operator lane and for nobody else, so the row names that signer
  while their standing lets them close, and `lane:operator` once
  suspension or revocation means every close from them refuses.
  Attributing a debt to a fingerprint nobody can sign for would hide
  it from the one actor able to pay it, on exactly the
  revocation-recovery path the charter cares about.
- **`contract.blocked`** — a blocked subject; owed by the operator
  lane; discharged by the verbs leaving `blocked`.

The list is **closed**. An open-ended taxonomy would make this a
policy surface rather than a derivation.

## The situation read

`seed situation --ledger <dir> [--key <path>] [--subject <id>]
[--since <position>]` renders one position-stamped envelope: the
caller's obligations, the windows they hold with the active fence
(the argument a lane would otherwise read out of a projection by
hand), and the standard advisory fields. A keyless read guesses no
identity and reports the board unfiltered; probes must be signed, so
affordance stamping needs the key itself.

`--since <position>` makes the response a **complete change report**,
not a filtered list: obligations that arose **or changed** after the
cited position, an explicit `discharged` list naming every obligation
that stood at it and no longer does, and a count of the unchanged.
Applying the response to a prior snapshot must reproduce the standing
set exactly — the property the drills assert. A delta of standing rows
alone would leave a resuming lane holding a discharged obligation
forever, and an unchanged *count* cannot say what disappeared. The
cited position is a tip **ordinal**, so the prefix a lane last saw is
`records[:position+1]`.

**Changed** is content, not position. A row whose `owed_by` moves
keeps the position it arose at, because the obligation changed hands
rather than restarting — so a delta keyed on `since` alone would call
a transfer unchanged, and the removals, derived from the prior set
filtered to the caller, would not carry it either: the party it moved
TO would hear nothing at all. That is the standing-aware `budget.open`
transfer above, and the party it moves to is by construction the only
one who can act. So the delta compares each standing row against what
stood at the cited position, on the **unfiltered** prior set, because
"it was not mine then and is now" is exactly the case a
caller-filtered comparison cannot see. A row that stood at the cited
position under no entry at all — `run.unsettled` begins standing at a
position later than its own `since` — is reported for the same reason.

The read is read-only and idempotent: it opens the ledger read-only,
mutates nothing, and journals no attempt, because a read is not an
admission-boundary attempt ([`refusals.md`](refusals.md)).

## The drift class

An obligation must be **dischargeable by the party it is owed by**,
at the position it is emitted at, judged by the same `admit.Check`
admission enforces. The sweep asserts *at least one* discharging verb
admits for the owed actor — not all of them — because `discharged_by`
names the acts that end the obligation while **who** may perform each
is the affordance layer's business: a claim is discharged by release,
park, reap or submission, of which the holder may take three and the
supervisor the fourth. Requiring every verb to admit for the owed
actor would force capability policy into a derivation that must stay
a projection.

One global exception: under a declared halt nothing admits but the
lift, so every obligation still *stands* while none is dischargeable.
That is the halt working, not an obligation defect, and the sweep
checks only well-formedness at halted positions.

The sweep walks the shared scenario's every prefix, and the scenario
ends by **suspending and then revoking** a lane that still holds an
open reservation: a walk of only active actors can never reach the
positions where an obligation's usual owner has lost the power to
discharge it, which are the positions standing-aware attribution
exists for.

## Deliberately absent (v0)

No priority or ranking, no due-date policy beyond what a caller
declares, and no cross-subject aggregation.

The acts that create and discharge these obligations are the loop
verbs ([`loop-verbs.md`](loop-verbs.md)), Phase 9 item 5's third
part. The relationship runs one way: a verb never re-derives what
discharges what — this projection already says it, and the loop
verbs' drills assert the pairing against it rather than restating
it.
