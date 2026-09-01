# Plan: next — the loop verbs' three post-merge defects (os-9b3f3ef3)

Three findings from the Codex review of #173, which merged before the
review was worked, so all three are live on `main`. Each was verified
against source before this plan was written; one was reproduced by
running it.

## Why these three belong on one card

They are not a grab bag. Two of the three are the **same mistake at the
same seam**: `commit` computes something against `openLoopSession`'s
view, hands it to `pushDraft`, and the optimistic loop then refreshes
the tip underneath it. One case sends a stale *argument* to the
boundary; the other renders a stale *answer* back to the caller. The
third is an ordinary panic, and it rides along because it is in the
same function family and its drill belongs beside the packet drills
that missed it.

## The defects

### 1 (P1) — a derived argument is not re-derived after the tip refreshes

`gitref.AppendLoop` re-fetches, re-materializes, re-signs with a fresh
`Prev`, and re-runs `validate` **against the refreshed store** on every
attempt (`internal/gitref/loop.go`). What it never re-does is the
payload: `Draft.Payload` is fixed at the call.

So when a rival appends between `openLoopSession` and the push, `budget
settle` still cites the reservation that was the only open one when the
session opened. Admission accepts it, because the budget rule checks
only that the citation **exists, is valid and is unclosed** — it never
requires it to be the *sole* open one. That check lives in
`soleOpenReservation`, in the CLI, against the stale view.

**The command therefore makes, silently, exactly the choice
`soleOpenReservation` exists to refuse.** The fence is exposed the same
way.

This is the defect `plans/os-7e197768.md` D3 was written to prevent:
"a fence read from a stale local copy would be wrong under exactly the
contention that makes claiming online-only." The session was shared as
D3 required; the derivation still ran once, outside the retry.

### 2 (P2) — a remote refusal is stamped and answered from the stale view

`loopSession.refuse` ends with `journalAttempt(stampTip(env,
ls.ctx.Count), …)` and computes affordances from `ls.ctx`. On the
remote path the refusal was computed one or more positions later, and
`remoteFailureEnvelope` **already stamps the refreshed position
correctly** through the `refusalAt` wrapper.

So `refuse` does not merely fail to refresh: it **overwrites a correct
stamp with a stale one**, and advertises affordances from before the
race (it can list `claim.taken` as legal on a subject a rival just
claimed). Worse than the review states, and the fix is correspondingly
different: stop clobbering, not just recompute.

### 3 (P2) — a JSON `null` packet panics the CLI

`loopPacket` unmarshals `--packet` into `map[string]json.RawMessage`.
The valid JSON value `null` unmarshals **with no error** and leaves the
map nil; if `--base` or a usable `--repo` then supplies a range,
`parts["base"] = b` panics. Reproduced:

```
unmarshal null -> err=<nil>, parts==nil: true
PANIC: assignment to entry in nil map
```

Without `--base`/`--repo` it refuses cleanly, which is exactly why the
malformed-packet drill missed it: the drill's bad packets were all
objects.

## Design decisions (binding for this task)

- **D1 — re-derive against the refreshed tip and REFUSE on divergence;
  never silently substitute.** The obvious reading of the finding is
  "re-derive inside each attempt and use the new value". That is wrong
  for this surface, and the reason is the same one that motivates the
  derivations at all: a value derived from a view that has since moved
  is not a better argument, it is a **different decision**. If a second
  reservation appeared, the lane's act is ambiguous and
  `soleOpenReservation`'s refusal is the correct answer. If the claim
  window was reaped and re-taken, the new fence is a new authorization
  the lane has not made.

  So the derivation is re-run against the refreshed store inside the
  loop's own validation seam, and the act is refused, naming what
  changed, when it no longer yields what was signed. Both dangerous
  cases already end in a refusal under either reading; this one never
  substitutes a value the caller did not choose.
- **D2 — this needs no `internal/gitref` change.** The `Validate`
  callback already receives `(store *ledger.Store, rec *event.Record)`,
  which is the refreshed store and the candidate record: everything the
  check needs. Threading a payload-producing function down into the
  loop would teach a transport layer about argument derivation, which
  is not its business, and would buy nothing D1 does not already give.
- **D3 — a refusal keeps the position it was computed at.**
  `loopSession.refuse` stops calling `stampTip` on an envelope that is
  already stamped, and takes the affordance context from the view the
  refusal came from. Concretely: stamp only when `env.Position == nil`,
  and pass the refreshed context where one exists. The local path is
  unchanged, because there the session context IS the view the refusal
  was computed at.
- **D4 — a packet must be an object before it is treated as one.**
  `loopPacket` verifies the root value is a JSON object and refuses
  with the documented usage envelope otherwise. The `null` case joins
  the malformed-packet drills, alongside a bare array and a bare
  string, because the drill missed this class by only ever trying
  malformed *objects*.
- **D5 — scope guard.** No surface change: same seven verbs, same
  flags, same envelope and journal contracts, same spec. Three
  correctness fixes and their drills.

## Steps

1. `cmd/seed/loop.go`: give the act a way to re-derive. `loopAct`
   carries the derivation it used (a closure over the subject taking a
   context and returning the payload), and the remote path's validate
   wrapper re-runs it against the refreshed store, refusing when the
   result differs from what was signed.
2. `cmd/seed/remote.go`: `pushDraft` composes that check into the
   `Validate` it already builds, keeping `admit.Check` as the other
   half. No signature change to `AppendLoop`.
3. `cmd/seed/loop.go`: fix `refuse` per D3 and `loopPacket` per D4.
4. **Drills.** The contention drill is the one that matters: with a
   claim held on a remote, a rival reservation lands **between** the
   session opening and the push, and `budget settle` refuses naming
   both candidates rather than closing one. Build it by making the
   validate seam observable, in the shape `remote_test.go` already uses
   to drive races. Plus: a remote refusal carrying the refreshed
   position rather than the session's (assert the stamp differs from
   the pre-race count); `null`, a bare array and a bare string as
   `--packet`, each refused with the usage envelope and the chain
   unchanged; and the existing drills unchanged, since no surface moves.
5. `next/docs/decisions.md`, `memory/*`; receipt; evidence; review.

## File Scope

- `next/cmd/seed/loop.go`, `next/cmd/seed/remote.go`
- `next/cmd/seed/loop_cli_test.go`, `next/cmd/seed/remote_test.go`
- `next/docs/decisions.md`, `memory/*`
- `receipts/os-9b3f3ef3.json`

## Acceptance Criteria

1. A rival reservation landing between session open and push makes
   `budget settle` **refuse naming both candidates**, never close one;
   the same shape holds for a fence whose window changed. Asserted end
   to end on a remote, not by unit-testing the helper.
2. Nothing is silently substituted: no path re-derives a value and
   proceeds with it in place of the one the caller's act was drafted
   against.
3. A remote refusal carries the position it was **computed at**, not
   the session's opening tip, and its affordances come from that same
   view. A drill asserts the stamped position is the refreshed one.
4. `--packet null`, `--packet '[]'` and `--packet '"x"'` are each
   refused with the documented usage envelope, with `--base` supplied
   (the combination that panicked) and without; the chain does not
   grow, and the CLI does not terminate.
5. `ledger append`, the seven verbs' flags, the envelope contract and
   the attempts journal are all unchanged; `next/spec/loop-verbs.md`
   needs no edit beyond any sentence these fixes falsify.
6. `make check` green, coverage gate ≥90% held.

## Validation Commands

```sh
cd next && gofmt -l . && go vet ./... && go build ./...
cd next && go test ./cmd/seed/ -count=1
make check
```

## Expected diff shape

Two CLI files and their drills. Roughly +180/-25 lines, all under
`next/**` plus the memory files.
