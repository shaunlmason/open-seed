---
id: os-8ecef90f
title: 'next: III.L row 4 — per-verb policy on the machine-protocol surface is not drilled by #273'
state: in_progress
priority: P2
squad: core
claim:
    actor: seed-next-implementer
    token: c-b9752b8e8731f263
    claimed_at: "2026-09-05T00:26:40Z"
    lease_expires: "2026-09-05T01:26:40Z"
created_at: "2026-09-04T15:02:49Z"
updated_at: "2026-09-05T00:26:40Z"
---

Found while verifying what the Phase 13 exit record (os-d63c7441) can claim. The exit record plan routing table says of III.L row 4: "judged against #273: the registry is the one verb table, every method runs the CLI own run function under the same admission, and approvals are ledger facts attributable to their signer; met if the drills say so, else re-routed with a card." This is that card: the drills do not say so.

III.L row 4 reads: "Per-verb policy governs the machine-protocol surface with attributable approvals."

WHAT #273 ACTUALLY DRILLS. cmd/seed/serve_test.go carries exactly three tests: TestRegistryMirrorsTheCLIVocabulary, TestProtocolMethodsAreTheCLIVerbs, TestServeReturnsTheCLIEnvelopeVerbatim. All three are parity assertions: the protocol surface exposes the same verbs as the CLI and returns the same envelope. Parity is III.I row 3 and it is met. None of the three asserts anything about POLICY PER VERB, and none asserts an approval attributable to a signer on that surface.

WHY PARITY IS NOT THE ROW. Parity says the protocol surface can do what the CLI can do. The row asks the opposite question: that what a caller may do THROUGH that surface is governed per verb, and that approvals granted there are attributable. Those are different claims, and the stronger one is the row. An implementation that mirrors the CLI perfectly and applies no per-verb policy satisfies every drill #273 added and fails this row.

The admission boundary does govern by verb, and approvals are ledger facts, so the row may well be TRUE. The point is that nothing DRILLS it on the protocol surface, and the exit record cites drills by name.

WHAT THIS CARD OWES. Either a drill on the serve surface showing a verb refused by policy there and an approval attributable to its signer, at which point the row flips; or, if the seam makes it structural, the record says so in the Phase 12 telemetry-grep style and the row is met by assertion with the reasoning written down.

Until then III.L row 4 stays routed rather than met, and the exit record cites this card.

## Comment cm-9df0f462 (seed-next-implementer, 2026-09-04T15:18:04Z)

Determination, so this card is decided rather than open. The row is structurally TRUE and what it needs is a drill, not an implementation.

WHY IT IS TRUE. cmd/seed/serve.go dispatch resolves the method through the same registry the CLI dispatches (reg.Resolve) and then calls g.Run(append(prefix, argv...), &stdout, &stderr) — the CLI OWN run function, with the params as its argv — and returns the seed-envelope it rendered verbatim as the JSON-RPC result. The file header states the design: "A method invocation runs the CLI own run function with the params as its argv and returns the seed-envelope it rendered, verbatim ... a refusal is a result carrying the failing envelope, never a transport error, so a machine caller reads the same structured code the exit status carries."

There is therefore no second admission path to govern. Per-verb policy governs the protocol surface for the same reason it governs the CLI: they are one code path. Approvals granted through it are ledger facts signed by their actor, attributable like any other record. An implementation could not diverge without introducing a second dispatch, which the registry seam forecloses.

WHY THE ROW STILL IS NOT MET. Nothing DRILLS it. The three tests in cmd/seed/serve_test.go are parity assertions (vocabulary, verb set, envelope verbatim), which is III.I row 3. Parity says the surface can do what the CLI can do; this row says what a caller MAY do through it is governed per verb. A refactor that gave serve its own dispatch would pass all three parity drills and silently break this row, which is exactly the gap worth a test.

THE DRILL THIS CARD OWES. Over seed serve, invoke a verb that admission refuses on policy grounds (the ceiling rule is the cleanest: an agent-kind key claiming above its squad declared max_agent), and assert the JSON-RPC result is a seed-envelope carrying that refusal code — not a transport error, and not a success. That proves policy governs the surface rather than only the CLI. A second arm asserting an approval made through the surface lands as a ledger fact naming its signer closes the attributable clause.

So the exit record can cite this card and, once the drill lands, flip III.L row 4 to met rather than re-routing it again. No design decision is needed after all; I was wrong to file this as one. What it needs is one test file, which needs this card promoted.
