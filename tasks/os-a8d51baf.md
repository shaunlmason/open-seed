---
id: os-a8d51baf
title: 'next: tool-edge authority passes over a declared policy map (backlog, §II.5 capability scoping, §II.9 executor adapters, §II.10 least standing capability)'
state: backlog
priority: P2
squad: core
labels:
    - next
created_at: "2026-09-07T06:09:27Z"
---

docs/next-build-plan.md §3 "Borrowed from surveyed tools (2026-09-06, pigeon)", the one card. Replaces os-46f19906, whose title said "derived from admitted facts" — the exact claim the PR #362 review corrected. The port surface has no title verb (Title is set at create only), so the correction is a refile; the merged build-plan text names no card id, so nothing downstream moves. Authoritative text: docs/next-build-plan.md §3 at 359c0e6.

Source: github.com/pigeonlabsHQ/pigeon (Pigeon Pass, a v0.1 delegated-authority credential: Ed25519-signed chain, every child provably narrower than its parent, closed constraint set with rate/count metered against every ancestor, fail-closed verifier).

At wake the executor adapter mints a short-lived pass for the run and a tool proxy it runs outside the agent process (container and cloud adapters; the local adapter declares it cannot, as it does for the tuple) verifies before each side effect and refuses on denial; a harness that spawns subagents hands each an attenuated child.

THE FIRST ITEM IS THE MAPPING, because the inputs do not exist yet: tool_policy is an opaque tuple string, a contract routing is one squad name (internal/transition), branch names are forge state the ledger cannot see (next/spec/verdicts.md), and Seed holds no lease at all, a claim standing until deliberate exit or reap (next/spec/observations.md). So capabilities, resources and TTL come from a declared tool-policy map keyed by the tuple tool_policy value, beside the lane manifest grants, read the way the claim ceiling and routing rule read the declaration: admission policy, not chain validity (next/spec/postures.md), by the cooperative client from its working tree and by the hook at the default branch tip. The adapter re-mints once per metering/poll cycle (next/spec/executors.md) so the pass dies with the run silence rather than citing a deadline the ledger does not record.

Three rules keep the ledger the one authority (charter §I.3, §II.18): the pass is a projection of declaration plus chain, never presented at admission, so actor.granted stays the only capability data admission reads; the operation cap is a declared number, NOT the budget reservation, whose units are abstract until Phase 7.3 metering (next/spec/budgets.md), so the two stay separate until units mean something; denials are metered as tool.refused observations (§II.9 telemetry, never an escalation) surfacing in the packet and the report and NEVER in a receipt, whose inputs are exclusively verifier-executed or verifier-read and reproducible from the submission head (next/spec/verdicts.md).

Format: pigeon SPEC §4 to §9 in Go under next/internal/pass with its fixtures/ as the conformance corpus; the credential never enters the ledger, so the JCS default governs nothing it touches (one decision-log line). Mock-total like every adapter (§II.13). Not a dependency: the reference is Python, in-memory stores, own canonical form, and no proof the leaf subject holds its key, so the harness binds the pass. Not conformance-blocking; changes no Part III row.
