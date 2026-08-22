# open-seed — v1 Build Plan

> Sequencing authority for implementing v1 as scoped in
> [`design-options.md`](./design-options.md) §7.3. The design doc is the authority on
> *what*; this file is the authority on *order* and *done-ness*. Phases are ordered by
> hard dependency — later phases may start early only where they don't depend on an
> unfinished deliverable. Status column is updated by PR as work lands.

## Decision status (read this first)

| Layer | Status |
|---|---|
| §2 settled ground (worktrees, AGENTS.md, Agent Skills, MCP, fresh-context, CI backpressure, status vocabulary) | **Decided** |
| D1–D8 recommendations (filecards default, squad-shaped topology, plan gate, guardrails stack, memory, lifecycle, CI layer, packaging) | **Decided** |
| §7.1 plugins + claim protocol · §7.2 state ref · §7.3 v1/v2 cut · §7.4 teams · §7.5 Go engine binary | **Decided** |
| §10 Q1–Q5 (harness posture, automation-on-clone, stack coupling, distribution, state-ref principals) | **Provisional defaults** — implement as written; changing one requires a reviewed edit to §10 |
| Anything not covered above | Implementer judgment, recorded in `decisions/` once that convention exists (Phase 6) |

## Phase 0 — Engine repository bootstrap

The two-repo split (§7.5) makes this step one, and it happens **outside this repo**.

- Create the engine repo (working name `open-seed-engine`; final name is an
  implementer decision recorded in `decisions/`). Go module, MIT license — matching
  the template's root `LICENSE`, which establishes MIT for both repos —
  `cmd/seed/` entrypoint.
- Release pipeline: goreleaser matrix (`linux/darwin/windows × amd64/arm64`),
  `checksums.txt`, GitHub artifact attestations (build provenance).
- Tag `v0.1.0` with a stub binary (`seed version` only) to prove the pipeline
  end-to-end before any protocol code exists.

**Done when:** a semver tag produces downloadable, checksummed, attested binaries for
all six targets, and `go install .../cmd/seed@v0.1.0` works.

> **Status: complete (2026-08-22).** Repo:
> [`shaunlmason/open-seed-engine`](https://github.com/shaunlmason/open-seed-engine)
> (public, MIT). One refinement over the plan as written: releases are driven by a
> `VERSION` file (or manual dispatch) — the workflow mints the semver tag at HEAD
> in-runner, so the tag and the released commit can never disagree and no
> contributor needs tag-push rights. Verified against
> [v0.1.1](https://github.com/shaunlmason/open-seed-engine/releases/tag/v0.1.1):
> six targets released with `checksums.txt`, provenance attestation green,
> download + sha256 + exec proven (the bootstrap shim's exact path), and
> `go install` proven via the module proxy.

## Phase 1 — Port spec as data + protocol core

- Author `.seed/port-schema/` in **this** repo: JSON Schemas for the nine required
  verbs' inputs/outputs, the transition table (D1) and verb classes (§7.1) as data
  files, exit-code registry (0/2/3/6/10), card frontmatter schema, `backend.toml`
  schema. Source of truth: design doc D1 + §7.1 (research/10 Part 5 only via its
  erratum).
- Engine implements the port shim against those files: table-driven transitions,
  claim-token fencing, `--json` envelopes, `.seed/version` check (exit 10). All
  remote git operations shell out to system git.
- Exhaustive conformance test: every cell of the transition table × verb class ×
  credential class, generated from the spec files.

**Done when:** the conformance suite passes and a schema/table edit changes engine
behavior with no engine code change.

> **Status: complete (2026-08-22), engine v0.2.0.** `.seed/port-schema/` authored
> (16-edge D1 table with classes/effects/overrides as data; nine verbs; exit-code
> registry; envelope/card/backend schemas). Engine `internal/spec` +
> `internal/port` implement the port table-driven with zero per-edge branching;
> the conformance suite sweeps state × verb × credential × token × lease against
> an independent interpreter of the same tables, asserts the design-doc
> invariants directly, and proves spec edits flip behavior with no code change.
> `seed spec lint` validates a repo's spec (exit 10 on mismatch).

## Phase 2 — Filecards backend + `seed-state` ref

- `filecards` backend plugin (ships in-template): card files, `backend.toml`
  capability manifest, `backends.lock.json` pinning.
- State-ref lifecycle (§7.2): `seed init` orphan-ref creation with race handling;
  fetch→commit→push with jittered backoff, one commit per verb (card mutation +
  run-log line atomically); shallow/filtered fetches.
- Claim protocol (§7.1): synchronous claim, mandatory leases with `lease-renew`,
  reap semantics (maintenance-side logic lands Phase 5), handoff stubs, claim-ending
  exits from `in_progress`, multi-valued `blocked_on`.
- Integrity: anchor-tag ancestry verification on fetch, non-fast-forward
  halt+escalate, `HALT` marker honored by mutating verbs, `seed state resume`.
- `seed init-github`: verifies branch protection on `seed-state` (no force-push, no
  delete), the anchor-tag protection rule, and prints the Q5 hardening option.

**Done when:** two concurrent claimants on one card deterministically produce one
winner (exit 0) and one loser (exit 2); a fenced-out stale claimant gets exit 6; a
simulated ref rewrite halts the shim.

> **Status: complete (2026-08-22), engine v0.3.0.** All three done-when scenarios
> pass as integration tests against real local git remotes, plus: anchor-ancestry
> failure halting a *fresh* clone, HALT marker blocking mutations until operator
> `seed state resume`, reject lockout via author-of-record, dep-cascade
> auto-unblock in the closing verb's own commit, one-commit-per-verb with atomic
> run-log lines, and the push-race retry loop observing the winner's write.
> Notes: filecards is implemented inside the engine (`entry = "builtin"` in its
> manifest — the external-plugin exec seam remains for installed backends);
> claim on an already-claimed card maps the table's exit 3 to exit 2
> (contention) per §7.1; `seed init-github` prints the protection checklist
> (the engine has no GitHub API access) — the API-side verification moves to
> Phase 5's workflows.

## Phase 3 — Template scaffold

- Lay down the §4 tree: `.seed/` (config.toml, guardrails.yaml, version, agents/,
  teams/, backends/, hooks/), root work-product dirs (plans/, receipts/, memory/,
  decisions/, skills/, rules/), `.worktreeinclude`, `.gitattributes` (merge=union on
  `decisions/**` only), CODEOWNERS, Makefile with `make check`, `.mcp.json`.
- Bootstrap shim: `scripts/seed` (POSIX sh) + `scripts/seed.ps1` — read pin +
  SHA-256 from the lockfile, download to a cache outside the repo, verify, exec;
  vendored-binary config key for air-gapped use; clear failure message when the
  fetch fails.
- The template's own `AGENTS.md` (user-facing, managed rules block) + one-line
  `CLAUDE.md`; move this repo's contributor `AGENTS.md` content into `docs/`.
- One `core` squad file; the default agent role trio + `dispatcher.md` in
  `.seed/agents/`.

**Done when:** a fresh template instantiation + `seed init` on a new GitHub repo
reaches a working claim/transition cycle with only the bootstrap shim checked in.
**Degradation check (differentiator #4):** the same instantiation with *no engine
installed* must still be a workable repo — readable cards, CODEOWNERS + server-side
gates intact.

> **Status: complete (2026-08-22).** Full §4 tree laid down (guardrails, CODEOWNERS,
> Makefile/validate.sh, hooks contract with the blocking pre-merge gate, role trio +
> dispatcher, `core` squad, work-product dirs, fan-out markers, `.worktreeinclude`,
> `merge=union` on decisions only). Bootstrap shim pair pins engine v0.3.0 via
> `.seed/engine.lock` (cold download+verify+exec 0.7s; SHA-256 tamper refused;
> `vendor`/`SEED_ENGINE` escape hatches). AGENTS.md namespace swap done — user-facing
> contract at root with the `seed:rules` managed block, contributor guidance at
> `docs/CONTRIBUTING-AGENTS.md`. Done-when verified on a scratch instantiation
> (create→promote→claim→review→close through the shim only); degradation verified
> (engine unavailable → validate warns and passes, cards + run log readable with
> plain git). Note: `.claude` fan-out and rules→AGENTS sync are markers until
> `seed sync` lands (Phase 6); `seed hooks run` fallback lands with a later engine
> release.

## Phase 4 — Plan/receipt chain

- Plan grammar validator (D3): `plans/<task-id>.md`, `## Validation Commands`
  parsing, merge-base blob rule, PR purity rule (plan PRs vs task PRs, classified
  server-side by head branch).
- `seed receipt` generate + `seed receipt verify`: merge-base diff excluding
  `receipts/**`, plan-hash recording, stale-plan failure ("plan changed since branch
  base"), `merge_group` adaptation as specified in D3.
- Guardrails validator: auto-merge allowlist may intersect neither the control
  surface nor `plans/**`; tier ≤ ceiling; scope-overlap and priority-uniqueness
  checks for team files; body-hash identity across role variants.

**Done when:** the D3/D4.5 gate scenarios each have a passing and a failing fixture:
plan tamper via task PR (rejected), stale plan replay (rejected), forged local
receipt (overwritten by CI author-of-record), amendment flow (rebase required, work
salvaged).

> **Status: complete (2026-08-22), engine v0.4.0.** All four done-when scenarios
> have passing and failing fixtures (plus: red validation commands, missing
> merge-base plan, merge-group-ref refusal). `seed receipt verify` reads the plan
> from the merge-base blob only — the tamper fixture proves the commands executed
> come from the merge-base even when the head copy is malicious. Validators:
> auto-merge intersection (conservative glob-prefix overlap), tier ≤ ceiling,
> human lead, unique priorities, non-overlapping scopes, role-variant body-hash,
> repo-wide plan lint — `seed validate`, wired into `scripts/validate.sh` with
> engine-absent degradation. Deferred to Phase 5 as planned: the `merge_group`
> workflow trigger and PR-number derivation (the engine side — refusing to
> classify a queue ref and requiring the real head branch — is in).

## Phase 5 — CI workflows

- **check+validate** (live): `make check` + all validators + fan-out drift check +
  state-ref fetch and lint (card lint, run-log commit-over-commit inclusion,
  transition-table conformance).
- **seed-maintenance** (live): lease reaping with handoff stubs, plan-unblock
  (state-shaped condition), anchor tagging, done-consistency lint, merged-PR close
  step under the operator credential, contention reporting, stalled-review
  reporting. Conformance failure writes `HALT`.
- **seed-dispatch** and **pr-review**: in-tree, inert until secrets exist (Q2
  default); activation documented via claude-code-action, gh-aw as the upgrade.
- `workflow_dispatch` `--no-pr` close path with actor-vs-roster resolution (D7).

**Done when:** both live workflows run green on the template itself with no model
secrets configured, and the reap/close/HALT paths each have an integration test
against a scratch repo.

> **Status: complete (2026-08-22), engine v0.5.0.** Five workflows shipped:
> check-validate (make check + validators + read-only state lint; on PRs and
> merge_group, the verify gate — with the D3 merge-queue adaptation deriving the
> PR from the queue ref and classifying by the real head branch), seed-maintenance
> (reap → state-shaped plan-unblock via `gh` → merged-PR close gated on the green
> verify check → conformance lint that writes HALT and stops the job before
> anchoring → anchor tag → report to the job summary), seed-close-no-pr
> (`workflow_dispatch`, server-attributed actor + run URL as evidence, engine
> enforces the operator roster), and the inert pair seed-dispatch/pr-review
> (secret-gated no-ops until activation, claude-code-action based). The
> reap/plan-unblock/no-PR-close/forged-transition-HALT paths all have integration
> tests against scratch git remotes; every live-workflow engine step was run green
> against the template itself (including the state lint replaying the repo's real
> seed-state history). Both live workflows execute on the default branch after
> this branch merges — verified locally step-for-step until then.

## Phase 6 — Loop, roles, memory

- `loop.sh`: dual-gate exit, circuit breaker, budgets, lease renewal at half-lease
  cadence.
- Fan-out sync: `seed sync` + `seed sync --check` (hash-based, offline) for
  `.claude/agents|skills` and `.agents/skills/` from the source trees; "do not edit
  here" markers.
- `memory/` (LEARNINGS.md, DEADENDS.md append conventions), `decisions/` ADRs
  (merge=union), `rules/` → AGENTS.md managed-block sync.
- Harness adapters per inspirations/09: `seed-harness` contract, claude + codex
  adapters (v1 cut), declared permission-tier variance surfaced by the validator.

**Done when:** a single agent under `loop.sh` takes a card from `ready` to `review`
unattended at L2 in a scratch project, with receipt, memory append, and green
check+validate.

> **Status: complete (2026-08-22), engine v0.6.0.** `seed sync` generates the
> fan-outs (byte-identical copies; AGENTS.md managed rules block from `rules/`
> fragments, regenerating byte-stable) with an offline `--check` wired into
> validate.sh. `loop.sh` runs the full cycle: claim (skipping planless cards),
> fresh worktree + post-create hooks, half-lease renewal in a detached
> background job, harness invocation via the `seed-harness` adapter contract
> (claude + codex adapters with declared tier mappings), the dual mechanical
> gate (blocking pre-merge hooks + receipt generate executing the merge-base
> plan's validation commands), evidence + hand-off to review; circuit breaker
> and iteration budget from guardrails.yaml. The done-when is proven by
> `make smoke` (scripts/smoke-loop.sh): a temp instantiation with a
> deterministic fake harness goes ready→review unattended with the receipt on
> the task branch, memory appended, evidence attached, gates green — no model,
> no secrets, CI-safe.

## Phase 7 — Docs, dogfood, release

- Conventions handbook in `docs/` (user-facing: lifecycle walkthrough, guardrails
  vocabulary, degradation ladder, backend upgrade path to beads, merge-queue note).
- **Dogfood:** open-seed tracks its own remaining work through its own port
  (`seed init` on this repo, remaining tasks as cards).
- Template release: tag, `.seed/version` set, engine pin recorded; `seed upgrade`
  guidance (R8).

**Done when:** a team can instantiate the template and reach Phase 6's unattended
loop scenario using only shipped docs — no knowledge from this repo's history
required.

> **Status: complete (2026-08-22) — v1 done.** `docs/handbook.md` is the user-facing
> conventions handbook (setup + protections, lifecycle walkthrough with real
> commands, the loop, guardrails honesty, the degradation ladder, scaling/upgrade
> guidance incl. the beads path and merge-queue note); README rewritten as the
> template front page with a quickstart. Dogfood is live: the remaining work — three
> P1 tasks (server-side protections, first live CI runs, template v0.1.0 tag) and
> the eight v2 items — are cards on this repo's own seed-state ref, created through
> the port. The done-when is `make smoke`: the shipped scripts + docs take a fresh
> instantiation to the unattended ready→review cycle with no repo-history knowledge.
> Note: the template release tag itself is a card (session credentials cannot push
> tags); tag `v0.1.0` on main after the v1 branch merges.

## Standing constraints (all phases)

- Every shipped convention ships with its validator (R9) — they are one deliverable.
- No model secrets in live v1 CI; deterministic workflows only (§7.3).
- The glossary (§9) governs all naming in code, config, and docs.
- v2 items (beads/github-issues backends, mirror, workflow engine, skills lockfile,
  mailboxes, multi-squad activation, MCP transport, Paperclip adapter) are out of
  scope; leave seams, not implementations.
