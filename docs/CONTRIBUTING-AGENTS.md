# AGENTS.md: instructions for agents working in this repository

## What this repository is

open-seed is a **template repository** teams will clone to give new projects
standardized, checked-in tooling for multi-agent orchestration, task tracking, and
guardrails. Two streams of work exist here:

- **v1 (the template)**: design is **complete**; the work is **implementing v1** per
  [`docs/build-plan.md`](build-plan.md). Your job in this stream is to build
  open-seed, not to redesign it and not to extend the research.
- **Seed (the successor, under `next/**`)**: chartered in [`SEED-NEXT.md`](../SEED-NEXT.md),
  sequenced by [`docs/next-build-plan.md`](next-build-plan.md), designed to be
  implemented by agents **without human intervention** — see the "Implementing the
  next version" section of the root AGENTS.md for the entry point. Seed (`next/**`) work never
  modifies v1 surfaces except the integration points the next-build-plan names.

## Authority order (binding)

1. **[`docs/design-options.md`](design-options.md)** is the design authority
   **for v1** (everything outside `next/**`). Within it: §2 settled ground, the §7
   decisions (7.1–7.5), the §6 team layer, the §9 glossary, and the §10 defaults are
   **binding**. The "Recommendation:" lines in D1–D8 are **decided**, not open: the
   options tables around them are retained history, not an invitation to re-litigate.
2. **[`docs/build-plan.md`](build-plan.md)** governs v1 sequencing and per-phase
   acceptance criteria.
3. **[`SEED-NEXT.md`](../SEED-NEXT.md)** (the Seed charter) is the design authority
   **for `next/**`** (Part II normative, Part III conformance), and
   **[`docs/next-build-plan.md`](next-build-plan.md)** governs its sequencing and
   per-phase acceptance, including binding defaults for open decisions. The two
   authorities never conflict because their scopes are disjoint; a change that would
   need both (a v1 integration point) follows the next-build-plan's ground rules and
   ordinary protected-path review. Amending the charter itself requires a PR reviewed
   by the owner — implementation agents propose, never self-ratify.
4. **`docs/research/**`** is *evidence*, never authority. Where a research file
   carries an **erratum header**, the erratum wins over the text below it. On
   open-seed's own schemas, the design doc supersedes the research: notably
   [`research/10-org-control-planes.md`](research/10-org-control-planes.md)
   Part 5 (the port spec), which is amended by erratum: the D1 transition table and
   §7.1 claim protocol in the design doc are the single authority for verbs, verb
   classes, and exit codes. Do not implement Part 5 as written without applying the
   erratum.

To change a decision: open a PR editing `docs/design-options.md` (v1) or
`SEED-NEXT.md` (Seed) with the rationale, and get it reviewed. Never silently
diverge in implementation.

## Rules of engagement

- **Use the glossary.** §9 terms (harness, backend, engine, port/shim/verb, role,
  runner, gate, control surface, …) have exactly one meaning. Do not coin synonyms;
  qualify "runner" and "gate" as the glossary requires.
- **Spec is data.** The transition table and verb classes are implemented
  table-driven from checked-in spec files (`.seed/port-schema/`), with an exhaustive
  conformance test, never as hand-written branching that could drift from the doc
  (§7.5).
- **Honesty over polish.** Known limits (R1–R12) are documented, not papered over.
  If an implementation cannot meet a stated guarantee, surface it: do not weaken
  the guarantee quietly.
- **Two repos.** The protocol engine is a separate Go repository publishing pinned
  release binaries (§7.5); this repository is the template and never contains the
  binary. Engine work happens there; template work here.
- **Namespace note.** The root `AGENTS.md` is the shipped template's user-facing
  agent contract (managed rules block, §4): a Phase 3 build artifact. This file is
  the *contributor* guidance, moved to `docs/` when scaffolding began; the root
  AGENTS.md points here. For work on open-seed itself, this file governs.
- **Commit and push** completed work to the designated branch; keep commits scoped
  and messages descriptive.
