# progress.md — the Seed implementation frontier

The single resume point (docs/next-build-plan.md §4): one line per plan item,
`phase.item — card id — PR — state`. A fresh agent resumes from this file
alone: read it top to bottom, verify the PR states it claims against the
forge, then take the frontier line's next action. Keep it truthful in every
PR that touches `next/**`; never start new work while it misstates the
frontier.

## Roster and conventions

- Implementer actor: `seed-next-implementer` (work verbs). Operator queue
  verbs for `next:` cards run under the session principal per
  `decisions/0003-next-loop-delegation.md`.
- Cards carry the `next:` title prefix; branches `seed/<card>` (task) and
  `seed/<card>-plan` (plan); receipts generated with `--run --write` before
  review.
- Phase gate: a phase starts when its dependencies' exit criteria are
  **merged** (docs/next-build-plan.md §2).

## Ledger

- 0.1 module scaffold + CI wiring (`make check-next`) — os-116ca9ac — plan
  PR #71, task PR #72 — **done** (merged; card closed)
- 0.2 spec skeleton (`next/spec/protocol.md`, `next/spec/envelope.md`) —
  os-116ca9ac — task PR #72 — **done** (merged; encodings and upgrade-safe
  version semantics per #71/#75 review)
- 0.3 decision log (`next/docs/decisions.md`; plus this frontier file) —
  os-116ca9ac — task PR #72 — **done** (merged)
- 1.1 event model + JCS + Ed25519 — os-aa146827 — plan PR #73, task PR
  #76 — **done** (merged; strict wire parsing + lowercase-only hex per
  review; card closed)
- 1.2 chain/segments/HEAD — os-ead12024 — plan PR #74 (merged, amended),
  task PR #79 — **done** (merged; card closed)
- 1.3 genesis via `seed init` — os-d636299d — plan PR #75 (merged,
  amended), task PR #83 — **done** (merged; genesis-payload version
  bootstrap + position-stamped init refusal per review; card closed)
- 1.4 push-race append loop — os-62e2aa1d — plan PR #81 (merged,
  amended: monotonic head + rollback drill), task PR #86 — **done**
  (merged; vanished-ref regression refusal + typed remote-rejection per
  review; card closed)
- 1.5 halt semantics in the rule set — os-bce3fb98 — plan PR #78 (merged,
  amended: exit code 7, reason-carrying state), task PR #84 — **done**
  (merged; halted refusal ordering + empty lift payload per review; card
  closed)
- 1.6 payload classification lint + hostile corpus — os-d6f81ec6 — plan
  PR #77 (merged, amended: aggregate free-text budget, embedded rules),
  task PR #80 — **done** (merged; narrowed anchor exemption + RFC 6901
  pointers per review; card closed)
- 1.7 CLI `seed ledger verify/append/show` — os-89412090 — plan PR #82
  (merged, amended: envelope v0 preserved, exit 9), task PR #85 —
  **done** (merged; position-stamped verify refusals, read-only show,
  active-version append per review; card closed)

**Phase 1 exit (charter III.A): met.** The chain verifies from genesis in
one command (`seed ledger verify`, #85); corrupted fixtures (reordered,
rewritten, forged-sig, bad-prev, lying HEAD, lying genesis) are detected
with positioned reasons (#79/#83/#85); the hostile classification corpus
passes (#80); the race drill (two concurrent appenders, no lost updates,
a real retry observed) is green on main (#86). This exit record is card
os-beac85e1's task PR (an administrative card, not a Phase 2 item).

## Phase 2 — Admission (docs/next-build-plan.md Phase 2; deps: 1 ✓)

- 2.1 admission rule set library (`internal/admit`) — os-3898f232 —
  plan PR #88 (merged, amended: ledger `WithObserver` + shared upgrade
  schema in scope), task PR #90 — **done** (merged; card closed)
- 2.2 cooperative posture (client self-validation) — os-895bf828 —
  plan PR #91 (merged, amended: gitref verify-option pass-through +
  update-phase race marker), task PR #93 — **done** (merged; state-dir
  lock + position-stamped refusals per review; card closed)
- 2.3 enforced posture, `seed-admit` pre-receive hook — os-d3591e09 —
  plan PR #92 (merged), task PR #94 — **done** (merged; tree-shape gate
  + exclusion-based rule selection per review; card closed)
- 2.4 posture declaration + `seed doctor` — os-3c72f93f — plan PR #95
  (merged, amended: exit 13 `posture_invalid` allocation per review),
  task PR #98 — **done** (merged; exit 66 `unreadable` for read
  failures + valid-postures wording per review; card closed)
- 2.5 admission drills (raw-git adversary; kill-and-replace) —
  os-028dda91 — plan PR #96 (merged, amended: fresh-clone
  kill-and-replace per review), task PR #99 — **done** (merged;
  shared adversary table under both postures + original host deleted
  with an independently installed hook copy, per review; card closed)

## Phase 3 — Identity and grants (docs/next-build-plan.md Phase 3; deps: 2)

- 3.1 actor events + keyring projection — os-52a2d688 — plan PR #97
  (merged, amended: root-liveness guard + seed/1 activation boundary
  per review), task PR #100 — **done** (merged; standing/grant rule
  split per review; card closed)
- 3.2 admission checks grants per verb — os-3979d48b — plan PR #101
  (merged, amended: versioning stance held, checkpoint accepts
  maintenance|operator, per review), task PR #102 — **done** (merged;
  vocabulary test parses the normative spec table per review; card
  closed)
- 3.3 key rotation/revocation drill — os-d1f35a8c — plan PR #103
  (merged, amended: exit record scoped to the III.E subset with the
  full unmet-remainder enumeration, per review), task PR #104 —
  **done** (merged; card closed)

**Phase 3 exit (the III.E subset docs/next-build-plan.md scopes): met.**
Signature and grant checks live in admission (#100: standing-aware
resolution behind the seed/1 boundary, root liveness, grandfathering;
#102: the capability vocabulary checked on every verb, operator-only
refusals structural, delegation via actor.granted proven end-to-end,
kind a drilled assertion), and the revocation drill is green (#104:
rotation with history attributed and every post-revocation proposal
refused at the rule set, the seed-admit boundary, and the CLI; the
compromised-key cut per posture with the cooperative consequence
observable; terminality, grants-die-with-standing, and root liveness
held at the boundary). Still-unmet III.E criteria, by landing phase:
implementer-disjoint self-approval and sealed-check keyring rotation on
verifier-key revocation (Phase 6); claim reaping on revocation (Phase
5); qualification tuples, grants citing them, and scheduled spot-check
suspension (Phase 10); the roster distinction consumed by agent-only
guardrails and human/agent metrics (with those surfaces, Phases 8 and
11).

**Phase 2 exit (charter III.B subset): met.** The validator is the
guarded ref's sole writer under enforced posture (#94: invalid-stream,
rewrite, truncation, force-update, and deletion refusals, tree shape
included); statelessness is drilled by kill-and-replace (#99: a fresh
host rebuilt from a bare clone makes identical decisions and the chain
verifies from genesis); the posture declaration prints the cooperative
consequence verbatim (#98); the raw-git adversary drill runs per posture
with the cooperative consequence made observable (#99); both postures
are selectable in fixtures through the shared declaration (#99).
Capability rules slot in at Phase 3, fences at Phase 5, reservations at
Phase 7, as added rules on the shared set.

## Phase 4 — Projections (docs/next-build-plan.md Phase 4; deps: 1 ✓)

- 4.1 projection engine (deterministic build, stamps, one-command
  rebuild) — os-4d5cacff — plan PR #105 (merged, amended: immutable
  builds + `CURRENT` pointer publication, roster includes genesis
  roots, overlap refusal, full-tree immutability and stale-`HEAD`
  fixtures, interrupted-publication drill, per review; second round:
  version-bearing build ids so derivation changes republish, and the
  superseded build survives one swap for in-flight readers), task PR
  #109 — **done** (merged; card closed)
- 4.2 standard projections (contract detail, ready queue, actor view,
  report skeleton, `seed project current`, exit 15 `stale`) —
  os-fecfb3f7 — plan PR #106 (merged, amended: the ready queue ships
  registered with `derivation: "none"` v0, per review), task PR
  #111 (stack-collapsed; its diff never reached main), re-landed as
  task PR #117 (merged, amended per review: registry-validated
  consumer verb, absence-4/damage-5 split refined so a layout
  missing its CURRENT pointer is damage, incomplete-stamp refusal,
  stale envelope position, unprivileged-cleanup test fix) —
  **done** (merged; card closed)
- 4.3 SQLite cache projection + mid-operation deletion drill —
  os-acc1ac78 — plan PR #108 (merged) + amendment PR #110 (merged:
  the cache is a registered projection, byte-identical like every
  view; the stamp table carries exactly the tree stamp's fields) —
  **ready to implement**, stacked on the Phase 4 stack tip
- 4.4 write-boundary lint wired into check-next — os-8d5e9c45 — plan
  PR #107 (merged, amended: seam/write-separation lint + locked trees
  `0444`/`0555` with the engine unlock window, deletion via rebuild,
  per review), task PR #112 (merged into the stack base during the
  stack collapse — its diff never reached main) — **re-landing as
  task PR #118**, stacked on #117

## Frontier

Phases 0 through 3 are done and closed; 4.1 is merged, and every
Phase 4 and Phase 5 plan (#105–#108, #110, #113–#116) is merged.
The #111/#112 stack collapse folded their diffs into stack branches
without reaching main; the re-land landed 4.2 as #117 (merged, card
closed). 4.4 (#118, this PR) and 4.3 (#119, stacked on it) are in
review with all Codex rounds addressed; the last two Phase 5 plans
are open as #120 (5.5 falsifiable-plan lint) and #121 (5.6
observations v0). **Next action: as #118/#119 merge, close
os-8d5e9c45/os-acc1ac78 — Phase 4 complete. 5.1 implements once the
Phase 4 implementations are on main; 5.2–5.4 follow per their plan
dependencies; 5.5/5.6 implement when their plans merge.**
If an open task PR is red or carries review feedback, drive it green
first — nothing merges out of order.
