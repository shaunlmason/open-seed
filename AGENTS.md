# AGENTS.md — instructions for agents working in this repository

## What this repository is

open-seed is a **template repository** teams will clone to give new projects
standardized, checked-in tooling for multi-agent orchestration, task tracking, and
guardrails. The design phase is **complete**; the current work is **implementing v1**
per [`docs/build-plan.md`](docs/build-plan.md).

Your job here is to build open-seed, not to redesign it and not to extend the research.

## Authority order (binding)

1. **[`docs/design-options.md`](docs/design-options.md)** is the design authority.
   Within it: §2 settled ground, the §7 decisions (7.1–7.5), the §6 team layer, the
   §9 glossary, and the §10 defaults are **binding**. The "Recommendation:" lines in
   D1–D8 are **decided**, not open — the options tables around them are retained
   history, not an invitation to re-litigate.
2. **[`docs/build-plan.md`](docs/build-plan.md)** governs sequencing and per-phase
   acceptance criteria.
3. **`docs/research/**`** is *evidence*, never authority. Where a research file
   carries an **erratum header**, the erratum wins over the text below it. On
   open-seed's own schemas, the design doc supersedes the research — notably
   [`research/10-org-control-planes.md`](docs/research/10-org-control-planes.md)
   Part 5 (the port spec), which is amended by erratum: the D1 transition table and
   §7.1 claim protocol in the design doc are the single authority for verbs, verb
   classes, and exit codes. Do not implement Part 5 as written without applying the
   erratum.

To change a decision: open a PR editing `docs/design-options.md` with the rationale,
and get it reviewed. Never silently diverge in implementation.

## Rules of engagement

- **Use the glossary.** §9 terms (harness, backend, engine, port/shim/verb, role,
  runner, gate, control surface, …) have exactly one meaning. Do not coin synonyms;
  qualify "runner" and "gate" as the glossary requires.
- **Spec is data.** The transition table and verb classes are implemented
  table-driven from checked-in spec files (`.seed/port-schema/`), with an exhaustive
  conformance test — never as hand-written branching that could drift from the doc
  (§7.5).
- **Honesty over polish.** Known limits (R1–R12) are documented, not papered over.
  If an implementation cannot meet a stated guarantee, surface it — do not weaken
  the guarantee quietly.
- **Two repos.** The protocol engine is a separate Go repository publishing pinned
  release binaries (§7.5); this repository is the template and never contains the
  binary. Engine work happens there; template work here.
- **Namespace note.** The shipped template will itself contain an `AGENTS.md` (agent
  instructions for *users of* open-seed, with a managed rules block — §4). That file
  is a Phase 3 build artifact and is not this file. When scaffolding begins, this
  contributor guidance moves to `docs/` so the template's own `AGENTS.md` can take
  the root path.
- **Commit and push** completed work to the designated branch; keep commits scoped
  and messages descriptive.
