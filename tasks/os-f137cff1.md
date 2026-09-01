---
id: os-f137cff1
title: 'next: Phase 10 — Wasteland stamps as a design reference for qualification tuples and verdict calibration'
state: backlog
priority: P3
squad: core
created_at: "2026-09-01T06:03:03Z"
---

Research note, filed as a reference for Phase 10 (deps: 9), not as a dependency. Steve Yegge's Wasteland (gastownhall/gastown, federated over Dolt/DoltHub) has shipped a working version of the thing Phase 10 has to build: a shared wanted board where rigs post work, claim it from other towns, submit completion evidence, and earn portable reputation as multi-dimensional STAMPS — quality, reliability, creativity, each scored independently WITH A CONFIDENCE LEVEL — under a "yearbook rule": you cannot stamp your own work. Verbs: gt wl browse / gt wl claim <id> / gt wl done <id> --evidence <url>, with validators reviewing submissions and evidence carried as DoltHub pull requests.

Three things worth reading it for when Phase 10 items 1 and 2 are planned. (1) Multi-dimensional-with-confidence is a different shape from a single qualification scalar, and III.E/III.G both want dimensions (tuples) and calibration (confidence); a shipped taxonomy is cheaper to critique than to invent. (2) The yearbook rule is Seed's L1 independence arrived at socially: Seed already enforces the stronger form at admission (exit 17 not_independent against ANY implementing key on the contract, past or present, verified rather than agreed), which is a positioning argument and a sanity check that the constraint is the right one. (3) Portable reputation across federated instances is a question Seed has not asked: qualification today is per-ledger standing in one keyring, and whether a tuple means anything outside the ledger that minted it is a real design question a later phase may want, or may want to refuse deliberately.

Scope guard for whoever plans against this: it is a REFERENCE, not an interop obligation. Nothing here proposes federating Seed, adopting Dolt (the ledger is a signed total order; Dolt's merge semantics are right for a many-writer reputation database and wrong for that), or importing a vocabulary. Adjacent context: this repo already depends on beads (bd) and ships a beads backend adapter, so the ecosystem is not foreign.

Sources: github.com/gastownhall/gastown; blog.kilo.ai/p/gas-town-ga. Gas City is the Kilo-hosted Gas Town, not a separate architecture.
