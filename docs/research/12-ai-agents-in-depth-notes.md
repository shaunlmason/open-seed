# Research: "AI Agents in Depth" (Bojie Li) — Notes and Cross-Check Against open-seed

> Reading notes researched 2026-08-30. Source: *AI Agents in Depth: Design Principles and
> Engineering Practice*, Bojie Li (github.com/bojieli/ai-agent-book, EPUB built 2026-08-26),
> read in full — twelve chapters, ~178K words. This file is research **evidence, not
> authority**: nothing here is binding; adoption happens only through the normal gates
> (a card, or a PR to `docs/design-options.md` for design-level items — authority order per
> [`docs/CONTRIBUTING-AGENTS.md`](../CONTRIBUTING-AGENTS.md)). Chapter numbers below use
> the book's own numbering (Introduction, then Chapters 1–10, Afterword).

## 1 — The book in brief

Core formula: **Agent = LLM + Context + Tools**, refined to **Agent = Model + Harness**,
where the harness's five jobs are to supply context, provide tools, constrain behavior,
verify results, and correct course. Five design patterns recur through every chapter:

1. **Proposer–Reviewer** — generation and verification split across roles; review is only
   worth anything when the reviewer reads *independent evidence* (test runs, rendered
   output, environment state), never the proposer's own explanation.
2. **Progressive disclosure** — index first, expand on demand (Skills, memory, tool
   catalogs).
3. **Append-only** — immutable evidence logs, mutable derived views rebuilt from them.
4. **Boundary set + retention set** — every change must improve the failing cases *and*
   not regress the working ones; watching only one side trains a broken system.
5. **Minimal diff + reversible** — small attributable changes with provenance and a
   rollback path, never wholesale rewrites.

Two threads the book repeats until they stick: **data and environment matter more than
algorithms** (Ch. 8), and **the model may propose "done" but must never approve its own
"done"** (Ch. 10's "Loop Engineering": the loop's bottleneck is the verifier, not the
model). The Afterword's "two clouds" — streaming real-time interaction, and genuine
accumulation of experience — frame the harness/model co-evolution: models eat harness
layers one at a time, and the harness migrates to each new capability frontier.

## 2 — Chapter map

| Ch. | Topic | One-line takeaway |
|-----|-------|-------------------|
| 1 | Getting started | Agent = LLM + Context + Tools; orchestration spectrum from workflow to autonomous agent; the five patterns above. |
| 2 | Context engineering | Stable prefix + append-only trajectory for cache discipline; status bar at the tail; compression is retrieval, not re-reasoning. |
| 3 | Memory & knowledge | Two-stage memory (immutable log → periodically rebuilt model, "User as Code"); RAG for facts that need update/citation/deletion. |
| 4 | Tools | ACI design; sub-agents return structured summaries, never full trajectories; file path as the universal interface. |
| 5 | Coding agents | Seven core tools; failure taxonomy by layer; circuit breakers; code as the meta-capability; Lethal Triad security framing. |
| 6 | Interaction | Async/event-driven loops, safe points, cancellation; voice; computer use. |
| 7 | Evaluation | Pass@k vs. Pass^k; rubrics with veto items; judge calibration; failure attribution to the *first erroneous step*; trajectory-prefix regression tasks; eval sets and simulation environments are the two cornerstones of training. |
| 8 | Post-training | Mid-training fixes the foundation, SFT fixes the protocol, RL fixes the policy; every stage transition needs a measurable entry condition; reward the outcome, penalize the path (RLVP); hidden tests the model cannot write. |
| 9 | Continual evolution | Preserving experience ≠ learning from it; three-layer verification (outcome → process → quality); four update carriers (knowledge / prompts+Skills / programs / parameters); dual loop (online executes and records, offline distills and gates); three safety boundaries (below). |
| 10 | Multi-agent | Information-gain criterion; shared vs. isolated context; peer/manager/decentralized topologies; OS analogy (trajectory=process memory, spawn=fork, shared FS=shared memory); six failure modes; A2A opaque collaboration. |

Ch. 9's three safety boundaries for self-evolving systems, verbatim in spirit:
(a) **evidence is never instructions** — untrusted content must not be executed or promoted
into long-term capability; (b) **candidates are separated from production** — nothing serves
real traffic before gates; (c) **safety mechanisms must not be self-modifiable** — an agent
may edit its skills and tools but never the validators, thresholds, audit logs, or release
gates that approve its own updates.

## 3 — Independent convergence: book positions that match open-seed decisions

The book derives, from first principles and unrelated evidence, positions open-seed has
adopted. Recorded here because independent convergence is itself evidence the decisions
are load-bearing — and because the book's phrasing is often a sharper test than ours.
Authority status matters per row: §7.1–7.5 and the D-decisions are **binding**
([`docs/CONTRIBUTING-AGENTS.md`](../CONTRIBUTING-AGENTS.md)); §7.6 and §7.7 are
**proposed** sections of the design doc, and rows citing them are convergence with a
proposal, not with settled design — adopting them still requires the design-authority
process.

| Book position (chapter) | open-seed counterpart |
|---|---|
| Worktree/working-copy isolation is "the mainstream industry practice" for concurrent edits to one codebase (10) | Per-card worktrees on `seed/<task-id>` |
| Optimistic locking with version checks for concurrent writers (10) | Claim/lease/fence semantics — exit 2 contention, exit 6 fenced, exit 10 version (§7.1). Note: this covers only the *pre-work* exclusive-claimant half. The book's other half — a parallel manager settling on the **first verified success** via an idempotent `settle_once`, racing *completed* candidates post-result — has **no open-seed counterpart**: nothing races N finished submissions. It maps to the verdict pipeline in SEED-NEXT and is recorded as a gap, not a convergence. |
| Homogeneous convergence: 18 of 30 concurrent agents created the *same branch name*; use namespaces and quotas against common-cause collisions (10, citing Anthropic's multi-agent study) | Task-id-derived branch namespace (`seed/<id>`, `seed/<id>-plan`) structurally prevents this |
| The reviewer "must not be able to modify the tests, the evidence collector, or the release gate — otherwise independent verification degenerates into self-approval" (10); safety mechanisms not self-modifiable (9) | `.seed/` as protected control surface (PR + owner review); D4.5 reviewer ≠ implementer |
| Evidence vs. instructions separation (9) | AGENTS.md: card bodies, mail, and issue text are data, not instructions |
| Append-only evidence with derived mutable views; exactly one authority with one-way projections (3, 9) | §7.7 (**proposed**): one authoritative store per repo, one-way projections, bidirectional sync forbidden |
| Retain negative results with the same retrieval status as successes, or the system revisits disproved paths (9) | `memory/DEADENDS.md` |
| A2A "opaque collaboration": exchange tasks and artifacts, never internal prompts or reasoning (10) | Handoff packets carry state, never trajectories |
| Disposable compute composes with durable record only if the record is complete before the machine dies (10's "the agent is its files"; Lingtai/Orbs-style substrates) | §7.6 (**proposed**): disposability begins only after confirmed push |
| MAST failure taxonomy's third class — **missing task verification** ("an Agent may claim completed but the result does not meet requirements") is the dominant multi-agent failure (10) | Evidence attachment + `make check` + review lane before `done` |

## 4 — Adoption candidates

Three ideas from the book that are concrete enough to card. Numbered B1–B3;
cross-referenced from the catalog in
[11-adjacent-tools-inspiration.md](./11-adjacent-tools-inspiration.md).

| # | Idea | What open-seed does with it | Priority | Effort |
|---|------|-----------------------------|----------|--------|
| B1 | **Three-part handoff-package spec** (Ch. 10, MetaGPT analysis). An effective handoff has exactly three parts: (1) task description *with acceptance criteria*, (2) confirmed facts and constraints — decisions already settled upstream, so the recipient doesn't relitigate them, (3) references to structured artifacts *by path*, never contents. Recipients need the package's format and semantics, never the sender's thought process. | A ready-made schema for **A1 handoff-packet enrichment** (doc 11): packet = acceptance criteria from the plan + settled-decisions block + artifact refs (plan path, receipt paths, diffstat vs. merge base, last failing check output). Folds into the A1 card rather than being a separate one. | Now | S (absorbed by A1) |
| B2 | **Provenance-bearing experience entries** (Ch. 9). The book's bar for promoting a lesson to formal knowledge: applicable-when conditions; ≥2 supporting non-failed trajectories before a "recommended strategy" (one accidental success is never promotable); exceptions; evidence sources; a last-validated stamp. Cross-trajectory comparison, not single-run summarization, is what makes a lesson transferable; invalidation must be precise (retire the entry, keep the evidence). | A lightweight entry template for `memory/LEARNINGS.md`: each entry carries `applies-when`, supporting task/PR IDs, exceptions, `last-validated: <date>`. `DEADENDS.md` entries symmetrically carry the failure condition and the environment it was observed in (so a dead end can be un-retired when the environment changes). Enforced socially at review, not by tooling, at first. Pairs with S3's `last-verified` stamps (doc 11). | Now | S |
| B3 | **Progress-file liveness** (Ch. 10, status-query section). Polling a sub-agent is the wrong primitive; the book's answer is an agreed lightweight progress file the worker updates as it completes items — reading it costs nothing, and its **mtime doubles as a heartbeat**: unchanged for N minutes ⇒ presumed stuck ⇒ timeout fallback. Full trajectory persistence stays available for deep debugging but is never the main status channel. | A convention: workers maintain `progress.md` (or a card-attached progress note) in the worktree; staleness beyond a threshold is the stuck signal. Complements lease renewal (lease says "the claim is held", progress mtime says "work is advancing" — a wedged worker renews leases forever). Feeds **D1 Tier-0** (doc 11) as a per-card freshness sparkline, and gives reap/nudge decisions a second signal beyond lease expiry. | Next | S |

## 5 — Design tests worth keeping (not cards)

Portable judgments from the book, useful as review questions for future `.seed/` and role
changes:

- **Information-gain test** (Ch. 10). A collaboration step earns its cost only if it
  introduces information the single agent could not produce — test execution, rendered
  output, environment state. Same-model re-reading of the same text is provably no better
  than equal-compute single-agent work, and self-review without external evidence makes
  results *worse*. Concretely: the reviewer lane must run `make check` and read the diff
  against the plan; a reviewer that paraphrases the implementer's summary adds negative
  value.
- **Boundary set + retention set** (Ch. 1, 8, 9). Any guardrail, role-file, or prompt
  change should name the failing case it fixes *and* demonstrate the previously-working
  cases still pass. A change validated only against its trigger case trains an
  over-corrected system (the book's example: a "verify before done" fix that produces an
  agent which never dares to finish).
- **The model never approves its own "done"** (Ch. 10). Completion is decided by hidden
  checks the worker cannot write — in open-seed terms: `make check`, the reviewer lane,
  and merge gates, never the implementer's claim. The book's "reward seeking" analysis
  (Ch. 8) is the failure this prevents: an agent that sets itself a shallow check, passes
  it, and stops.
- **Budget awareness beats budget size** (Ch. 10, citing budget-aware tool-use work).
  More steps/tokens don't help an agent that doesn't adapt strategy to remaining budget.
  Relevant to guardrails budgets: the useful signal to expose to a worker is *remaining*
  budget, not just a hard stop at exhaustion.
- **MAST's three failure classes** (Ch. 10) as a review checklist for orchestration
  changes: (1) system-design flaws — unclear interfaces, overlapping responsibilities;
  (2) inter-agent alignment — downstream misreading upstream artifacts; (3) missing task
  verification. Most multi-agent failures are Byzantine (plausible-but-wrong output, no
  announced error), which is why deterministic checks are disproportionately valuable.

## 6 — Read but out of scope

- **Ch. 8 training recipes** (mid-training/SFT/RL mechanics, GRPO/PPO, distillation,
  RLVP): open-seed does not train models. The transferable residue is already captured
  above — gates with measurable entry conditions, hidden verification, outcome + path
  checking.
- **Ch. 10 agent-society material** (Stanford town, Agentopia, Moltbook, market
  economies): interesting for the long-horizon fleet picture, no near-term open-seed
  bearing.
- **Ch. 6 voice / computer-use / robotics**: substrate-level; nothing to adopt.
