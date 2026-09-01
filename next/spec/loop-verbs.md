# The loop verbs

`seed situation` ([`obligations.md`](obligations.md)) answers *what is
true for me now*. This spec defines the other half of the lane's
surface: the **acts** it takes. Design authority: `SEED-NEXT.md` §II.9
and §II.10, and `docs/next-build-plan.md` Phase 9 item 5, whose
loop-completeness criterion this surface exists to satisfy — a lane
that cannot act without hand-assembling protocol arguments is not
running unattended, it is being driven.

The gap these verbs close is not verbosity. `seed ledger append` is
the raw seam and stays exactly as it is: it signs at the tip and
appends, **consulting the admission boundary not at all**. A lane
acting through it learns that its act was illegal from a chain-level
refusal, after signing, instead of from the boundary that would have
explained it. Two principles follow, and they are the whole design.

## Derive every argument the system already holds

An argument the system can compute is never asked for, because a
value the boundary would refuse is not a choice the caller is being
offered — it is an invitation to be wrong.

| Derived | From | Because |
| --- | --- | --- |
| the fence citation | the active claim window in the fold | a holder-signed event citing anything else is refused anyway |
| the reservation a close cites | the single open valid reservation in the shared budget view | the view is the one admission judges against |
| the plan anchor a submission cites | the approved `plan.approved` on the subject | an approval admits ONE exact revision, so no other value could be legal |
| the resume range | the repository at `--repo`: `HEAD`, and the merge-base against `origin/HEAD` | git already holds both facts |

What stays caller-supplied is what is a **judgment**, not a lookup:
`--amount`, `--actuals`, and the packet's own prose.

Where a derivation cannot be made, the shape of the failure decides
who explains it:

- **A missing fact** refuses here, naming what would establish it
  ("no open valid reservation stands on c-1 — `seed budget reserve
  --amount <n>` establishes it").
- **An ambiguity** refuses here too, naming the candidates and
  declining to pick: two open reservations are a spend decision the
  lane owns, and a silent choice would make it for them.
- **An absent window** is not a derivation failure at all: the key is
  simply omitted, and the boundary decides on its own account of the
  state, which is better than anything a derivation could say. It
  does not always refuse: a budget close outside a window is legal
  and cites no fence ([`budgets.md`](budgets.md)), which is precisely
  why omitting the key beats inventing one.

## Refuse before signing, and say what IS legal

Every verb drafts its record, runs the **same `admit.Check`**
admission enforces, and on refusal renders the boundary's own typed
error **beside the caller's current affordances on that subject**,
computed from the same view the refusal was computed at. Nothing is
appended and nothing enters the chain.

This is Phase 8's principle — one rule set, enforcement and
advertisement — carried from legality to construction. It is also why
these verbs are not sugar: the raw seam structurally cannot do it.

## The surface

`<noun> <verb>`, matching `offer publish`, `verdict render`, `seal
create`, `budget status`. Seven acts, one spelling each, no aliases,
chosen because together they close poll → claim → work → meter →
submit → deliberate exit:

| Act | Verb | Derives | Takes |
| --- | --- | --- | --- |
| `claim take` | `claim.taken` | — | — |
| `claim release` | `claim.released` | fence | `--packet` |
| `claim park` | `claim.parked` | fence | `--packet` |
| `submission make` | `submission.made` | fence, plan anchor | `--packet` |
| `budget reserve` | `budget.reserve` | fence | `--amount` |
| `budget settle` | `budget.settle` | fence, reservation | `--actuals` |
| `budget release` | `budget.release` | fence, reservation | — |

Each takes `--ledger` **xor** `--remote` (with `--ref` and `--state`),
exactly as `ledger append` does and through the same client
machinery, because a lane in any real posture works against a remote
ref. On the remote path the derivation and the pre-flight read the
**same materialized remote tip**: a fence read from a stale local
copy would be wrong under exactly the contention that makes claiming
online-only.

**`claim take` is remote-only.** `Table.Exclusive` marks
`claim.taken` alone, exclusivity is granted at the push round-trip,
and two offline actors claiming one contract have claimed nothing.
The verb refuses `--ledger` with the one account the raw seam already
gives, never a second explanation of one rule.

### The packet

The exits take `--packet <file>` and validate the four-part shape
([`packets.md`](packets.md)) **at the door**, before a session is
opened, so a malformed packet never becomes a signed record. The
`base` range may come from the file, from `--base`, or from `--repo`;
a file and a flag that disagree refuse rather than have a winner
picked by precedence.

### The envelope

Unchanged ([`envelope.md`](envelope.md)): position stamp,
affordances, budget block, and a journaled attempt at these
admission-boundary seams ([`refusals.md`](refusals.md)). Success
envelopes stay terse — teaching text lives in refusals — except where
the response names a position the caller would otherwise have to look
up: a take names the **fence** it established, a reserve names the
**reservation id** its close will cite.

Derivation refusals are journaled like admission refusals. A lane
that could not act is exactly the affordance gap the metric measures,
and the journal's `by_code` breakdown keeps `usage` and `not_found`
distinguishable from the boundary's own codes.

## Deliberately absent (v0)

- **No new authority.** Every act is a verb the tables already carry;
  nothing here can admit what admission refuses.
- **No orchestration and no retries.** These are acts, not a loop.
  The worker-lane loop that sequences them is Phase 9 item 1.
- **No state outside the ledger.** Nothing is cached between
  invocations; every derivation is recomputed from the authoritative
  view.
- **No filing or specification verbs.** Those stay operator acts on
  the raw seam until a later card widens the set.
- **On the remote path, no success affordances and no journal.**
  There is no local ledger to reopen at the landed tip and none to
  journal beside; refusals there still carry affordances, computed
  from the materialized tip the act was judged against. Giving the
  remote posture its own journal home is a client-state decision this
  card does not make.
