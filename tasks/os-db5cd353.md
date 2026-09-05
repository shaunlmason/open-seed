---
id: os-db5cd353
title: 'next: III.A row 7 — erasure is surfaced but not attributable; the protocol defines no erasure verb'
state: review
priority: P2
squad: core
author: seed-next-implementer
created_at: "2026-09-04T15:02:29Z"
updated_at: "2026-09-05T00:23:13Z"
---

Found while verifying what the Phase 13 exit record (os-d63c7441) can claim.

III.A row 7 reads: "Erasure obligations are honorable: erasing a referenced artifact never breaks chain verification, and the erasure is itself an attributable event."

HALF OF IT IS DRILLED. TestSealEndToEndCLI (cmd/seed/seal_cli_test.go) removes a sealed ciphertext and asserts the maintenance pass reports seal_evidence_missing, twice over; cmd/seed/seal.go words it "erasure is a surfaced state, never silence". internal/artifact/artifact.go says the same of the sealed bucket: "deleting one is the charter erasure path, surfaced by the audit, never silence".

THE OTHER HALF IS ABSENT BY CONSTRUCTION. "The erasure is itself an attributable event" means a record in the chain, signed by whoever erased. There is no erasure verb: a grep for a Verb constant matching eras/delet/redact/forget/tombstone across internal/ returns nothing, and admit CatalogVerbs drafts none. What exists is a maintenance LINT FINDING, which is a projection of absence, not an attributable event: it says the ciphertext is gone, never who removed it or when. A lint finding also cannot be verified by a fresh reader from the chain alone, which is what "attributable" buys.

The first clause is arguably structural: the chain hash-links records and references artifacts by digest, so erasing artifact bytes cannot break verification. It is still undrilled. A cheap drill would erase an artifact and assert ledger verify stays green.

WHAT THIS CARD OWES A DECIDER. Either (a) add the erasure verb so the act is a signed record and the row is met, which touches the protocol and the transition table and is not small, or (b) revise the row the way the Phase 10 record revised its criterion, saying erasure is surfaced rather than attributed, and say so in the exit record. Not decidable from the tree: the charter wrote "attributable" deliberately.

Until then III.A row 7 cannot be flipped to met by the exit record, and the record should say UNMET and cite this card.

## Evidence ev-d9ee11cb (receipt, seed-next-implementer, 2026-09-05T00:23:04Z)

receipts/os-db5cd353.json

## Evidence ev-634098a4 (pr, seed-next-implementer, 2026-09-05T00:23:07Z)

325

## Comment cm-5422c274 (seed-next-implementer, 2026-09-05T00:23:10Z)

Plan #324 merged; main merged forward into seed/os-db5cd353, receipt receipts/os-db5cd353.json regenerated against the plan at the merge base with the validation commands run (all exit 0), PR #325 marked ready for review.
