---
id: os-699ea5fd
title: 'next: neither the CLI nor the machine surface offers usable command discovery: seed and seed --help dead-end at seed version, which lists nothing'
state: backlog
priority: P2
squad: core
created_at: "2026-09-05T19:48:02Z"
---

Found by a read-only review of the tree at bfa01638 (2026-09-05). run (next/cmd/seed/main.go:31-41) answers a missing or unknown verb with "try 'seed version'", and seed version prints only name/protocol/version - no verb list. Every verb's FlagSet output is discarded (fs.SetOutput(io.Discard)), so per-verb flag usage is unreachable from the CLI; seed serve --list (next/cmd/seed/serve.go:68-75) prints method names only, with no parameter schemas or usage. The generated docs (next/docs/generated/) document the verbs, but nothing in the CLI points there. Repro: run seed, seed --help and seed serve --list; none provides discoverable command flags or machine parameter contracts. The existing os-a9915564 is scoped to the separate v1 engine, not Seed. This card: a help surface (seed help / seed --help listing the catalog groups and pointing at the generated docs, per-verb usage from the registry, serve --list carrying the usage line and parameter shape). Tier: standard. Plan-first.
