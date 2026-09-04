# Plan: next — the packet and the frontier after the substitution (os-4fde2bdf)

`next/docs/promotion.md` on `main` (#316) records criterion 4 as
`partial` under the operator's substitution of the accelerated
simulation for the live seven-day shadow run, and every III.R row as
`not measured`. Around that honest section the packet still speaks in
the protocol's tense: the gate paragraph names only the criterion's
status, the Self-hosting question is asked "at the position the shadow
window closed", and its preconditions list "the shadow window closed
with every divergence reconciled", a window the substitution means
will not open. The Frontier in `next/docs/progress.md` is older still:
it names criterion 4 as `reserved` on the POSIX git server question and
carries five merged cards as `in review`. Neither document says what
now stands between the packet and the self-hosting question, which is
the one thing the operator reads them for. This card re-derives both
from `main` and nothing else. Tier: trivial in code (one drill),
plan-first because the packet is what the gate reads (the os-f79bc5a0
precedent). Deps: none.

## What the tree actually shows

- **Section 4 is honest and the rest of the packet has not caught up.**
  The status, the `Missing:` line, the deviation named, the III.R
  intro paragraph and the ledger all reflect the substitution. The
  gate paragraph ("The gate is not open: criterion 4 is partial")
  stops there; "The two cutovers are escalations" asks the
  Self-hosting question at a window's close and lists that close among
  its preconditions; "The shadow run, as a protocol" is preserved as
  the substituted protocol, correctly.
- **What the cutover needs that no agent can supply.** The packet's
  own "The deployment" paragraph names it: a ledger remote whose
  `pre-receive` is the `seed-admit` binary (or, under
  `enforced-forge-hosted`, the admission service and its git
  credential), the operator's root key at genesis, one enrolled key
  per lane. The autonomy contract reserves credentials and
  infrastructure to the operator, and no Seed deployment for this
  repository exists (no `seed.json` at the root, no `refs/seed/ledger`
  on the remote).
- **The proposed declaration is prose.** The JSON block under "The
  deployment" is what the operator would copy to `seed.json`; nothing
  holds it to `seed preseed check`, so a stale block would be found at
  the cutover rather than in CI. `next/fixtures/deployment/seed.json`
  is held that way under `make check-next`.
- **The Frontier misstates the ledger.** os-aaec6a3c (#313),
  os-88df7ab2 (#311), os-b86dab4c (#306), os-f262585a (#314) and
  os-5063e8ba (#300) read `in review` and are merged; the "Next
  action" paragraph names a `reserved` criterion 4 and a git-server
  question the operator has since answered with the substitution; the
  agent-side remainder (os-8ecef90f, os-b5051f2e, os-db5cd353) is
  unnamed.

## Design decisions (binding for this task)

- **D1 — the gate paragraph says what stands between the packet and
  the question.** After the criteria table: criterion 4 is `partial`
  by the operator's substitution, so the gate does not open on the
  criteria's own terms and the packet does not pretend otherwise; and
  what remains before the Self-hosting question can be put is not
  agent work: a deployment at the enforced self-hosted posture, which
  only the operator can stand up (the ledger remote with the hook, or
  the service and its credential; the root key; the lane keys), named
  by reference to "The deployment". Nothing agent-side remains open
  once the three cards in review merge, and the paragraph says so.
- **D2 — the Self-hosting question is asked at a position the operator
  records.** "Does this repository's own development move to Seed at
  the position the operator records as the start, on the terms in
  'The cutover and the rollback', with criterion 4 standing `partial`
  under the substitution section 4 records?" Its preconditions become:
  six criteria `met` and the fourth `partial` by the operator's own
  substitution, a deployment standing at the enforced self-hosted
  posture, the v1 state anchored and imported at the flip, and the
  compromised-actor drill green on the cutover commit. The
  Distribution question and its preconditions are unchanged. Both stay
  questions the parser models; neither is answered here.
- **D3 — the protocol section is framed as the substituted protocol.**
  Its first sentence gains one clause saying the operator substituted
  the simulation for it (section 4), and "The deployment" is
  re-headed as what the cutover needs whether or not a window runs,
  since the cutover's ledger is that deployment either way. The
  protocol's text otherwise stands as the record of what was proposed.
- **D4 — the proposed declaration is held to the tree.** A drill
  (`TestPacketDeclarationLints`, `cmd/seed`) extracts the JSON block
  under "The deployment" from the packet and runs `seed preseed check
  --config <it> --lanes lanes` on it, so the block the operator copies
  to `seed.json` is one `make check` has linted: tiers in the
  vocabulary, teams naming shipped manifests, the protected surface
  complete, the protocol in the register. The block's `protocol` moves
  to the register's newest version if it is behind.
- **D5 — the Frontier is re-derived, not edited.** The five merged
  lines read `done` with their PRs; the "Next action" paragraph
  states, from the cards: the agent-side remainder in review by card
  and plan, the two rows they flip, the doctor's count once they merge
  (28 outstanding: 21 Phase 13 rows the exit record flips, C.4 and Q.7
  routed, III.R's seven), and the one decision that is the operator's,
  the deployment and the Self-hosting answer, in that order. The
  destination paragraph and the closing rule stand.
- **D6 — bounds.** The III.R ledger stays `not measured`; no
  conformance row moves; neither cutover is performed; no root file is
  created; `internal/promotion`'s parser and drills are unchanged
  except that the packet must still parse and every citation still
  resolve.

## Steps

1. D1, D2, D3 in `next/docs/promotion.md`; the parser and the citation
   check green (`go test ./internal/promotion/`).
2. D4 in `next/cmd/seed/promotion_cli_test.go`, cited in the packet's
   section 5 evidence rows beside `TestPacketWritesTheCutoverDown`.
3. D5 in `next/docs/progress.md`; `next/docs/decisions.md`,
   `memory/LEARNINGS.md`; receipt; evidence; review.

## File Scope

- `next/docs/promotion.md`
- `next/cmd/seed/promotion_cli_test.go` (new)
- `next/docs/progress.md`, `next/docs/decisions.md`, `memory/*`
- `receipts/os-4fde2bdf.json`

Nothing else. NOT `next/spec/**`, NOT `next/internal/**`, NOT
`next/cmd/seed/*.go` outside `_test.go`, NOT `docs/next-build-plan.md`,
NOT `.seed/**`, NOT a root `seed.json`.

## Acceptance Criteria

**Boundary set (new, shown working):**

1. The packet's gate paragraph names the deployment as what stands
   between the packet and the Self-hosting question, and says that
   nothing agent-side remains once the three cards in review merge.
2. The Self-hosting question is asked at a position the operator
   records, its preconditions name the substitution, the deployment,
   the anchored import and the drill, and both cutovers still parse as
   questions (`TestPacketWritesTheCutoverDown` green).
3. The declaration block in the packet passes `seed preseed check`
   under the drill, and the drill fails on a planted unknown tier.
4. The Frontier reads `done` for the five merged cards, names the
   three cards in review with their rows, and states the operator's
   decision; `TestPacketCitesRealDrills` and `seed docs check` are
   green.
5. `make check` green; no model identifiers in any committed artifact.

**Retention set (existing, shown unharmed):**

- Every III.R row reads `not measured`; every conformance row keeps its
  status; the shadow-run protocol's text stands; `internal/promotion`
  is byte-identical to `main`.

## Validation Commands

- Boundary: `cd next && go test ./internal/promotion/ -count=1`
- Boundary: `cd next && go test ./cmd/seed/ -run 'Packet' -count=1`
- Retention: `make check` (exit checked separately from any pipe)

## Expected diff shape

Modified: `promotion.md` (the gate paragraph, the cutover questions and
preconditions, two sentences in the protocol section, one evidence row;
roughly +30/-15), `progress.md` (five ledger lines and the Frontier),
the two docs files, the receipt. Added: `promotion_cli_test.go`
(roughly 60 lines). No other file.
