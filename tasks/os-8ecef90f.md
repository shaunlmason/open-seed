---
id: os-8ecef90f
title: 'next: III.L row 4 — per-verb policy on the machine-protocol surface is not drilled by #273'
state: backlog
priority: P2
squad: core
created_at: "2026-09-04T15:02:49Z"
---

Found while verifying what the Phase 13 exit record (os-d63c7441) can claim. The exit record plan routing table says of III.L row 4: "judged against #273: the registry is the one verb table, every method runs the CLI own run function under the same admission, and approvals are ledger facts attributable to their signer; met if the drills say so, else re-routed with a card." This is that card: the drills do not say so.

III.L row 4 reads: "Per-verb policy governs the machine-protocol surface with attributable approvals."

WHAT #273 ACTUALLY DRILLS. cmd/seed/serve_test.go carries exactly three tests: TestRegistryMirrorsTheCLIVocabulary, TestProtocolMethodsAreTheCLIVerbs, TestServeReturnsTheCLIEnvelopeVerbatim. All three are parity assertions: the protocol surface exposes the same verbs as the CLI and returns the same envelope. Parity is III.I row 3 and it is met. None of the three asserts anything about POLICY PER VERB, and none asserts an approval attributable to a signer on that surface.

WHY PARITY IS NOT THE ROW. Parity says the protocol surface can do what the CLI can do. The row asks the opposite question: that what a caller may do THROUGH that surface is governed per verb, and that approvals granted there are attributable. Those are different claims, and the stronger one is the row. An implementation that mirrors the CLI perfectly and applies no per-verb policy satisfies every drill #273 added and fails this row.

The admission boundary does govern by verb, and approvals are ledger facts, so the row may well be TRUE. The point is that nothing DRILLS it on the protocol surface, and the exit record cites drills by name.

WHAT THIS CARD OWES. Either a drill on the serve surface showing a verb refused by policy there and an approval attributable to its signer, at which point the row flips; or, if the seam makes it structural, the record says so in the Phase 12 telemetry-grep style and the row is met by assertion with the reasoning written down.

Until then III.L row 4 stays routed rather than met, and the exit record cites this card.
