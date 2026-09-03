---
id: os-cf13fb51
title: 'next: Phase 12 item 5 — migration: seed import --from-open-seed, lossless export → anchors verified → transform → genesis import, drilled against a real v1 export'
state: backlog
priority: P1
squad: core
created_at: "2026-09-03T00:10:19Z"
---

Build plan Phase 12 item 5 (docs/next-build-plan.md): migration — `seed import --from-open-seed <export>`: v1 lossless export → verify anchors → transform (cards → contracts, run-log entries → events, receipts → verdict records, mail → messages) → genesis import refusing non-empty ledgers; drilled against a real v1 fixture. This carries promotion's migration gate (build plan §5 criterion 3: drilled against a real export of THIS repository's v1 state, not only a fixture).

Charter: §II.17 "Genesis first" and Appendix D.2 (two commands, documented, drilled in CI against a fixture; prior tamper-evidence verified before conversion; the import boundary is the genesis transform, not per-system compatibility code in the core — D.3); III.P row 4.

What the tree has, v1 side: `scripts/seed state export` dumps the whole store as one JSON document ({schema_version, backend, head, files: {path: content} — 241 files today}) and `state import` loads one into a fresh store; `state anchor` tags the state head as seed-anchor/<ts>, the tamper-evidence the charter says to verify. Seed side: genesis via `seed init` (#83), the transition table and lifecycle verbs (#122), packets (#124), messages (#211), verdict records and receipts (#135), the classification lint that refuses content bodies in payloads (#80) — an imported card body is a reference to an artifact, never a payload.

Expected shape, for the plan to settle: the export's tamper-evidence checked first (its head against the anchor tags and the state ref's history it came from; a mismatch refuses before any transform); a deterministic transform table as data, mapping v1 card states, run-log verbs, receipts and mail onto Seed events, signed by the importing operator's key with the original actors carried as ASSERTED provenance (attribution is not trust, and v1 identities were never keys); genesis import refusing a non-empty ledger by exit code; losslessness made checkable (every export record lands as an event, an artifact, or a named drop with its reason, and a coverage drill asserts it); the fixture a real export of this repository's v1 state, drilled in CI; the two-command path documented. Phase 12 opens when the Phase 11 exit record merges; plan now, implement as a draft until then (decisions/0003).
