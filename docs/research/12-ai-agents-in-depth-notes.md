# Research: "AI Agents in Depth" (Bojie Li) — Notes and Cross-Check Against open-seed

> Reading notes researched 2026-08-29–30 (timestamps UTC). Source: *AI Agents in Depth:
> Design Principles and Engineering Practice*, Bojie Li (github.com/bojieli/ai-agent-book,
> EPUB built 2026-08-26), read in full — twelve chapters, ~178K words. This file is
> research **evidence, not authority**: nothing here is binding; adoption happens only
> through the normal gates (a card, or a PR to `docs/design-options.md` for design-level
> items — authority order per
> [`docs/CONTRIBUTING-AGENTS.md`](../CONTRIBUTING-AGENTS.md)). Chapter numbers use the
> book's own numbering (Introduction, then Chapters 1–10, Afterword); load-bearing claims
> cite the book's section numbers so they can be checked without rereading.

## 1 — The book in brief

Core formula: **Agent = LLM + Context + Tools**, refined to **Agent = Model + Harness**,
where the harness's five jobs are to supply context, provide tools, constrain behavior,
verify results, and correct course. Five design patterns recur through every chapter:

1. **Proposer–Reviewer** — generation and verification split across roles; review is only
   worth anything when the reviewer reads *independent evidence* (test runs, rendered
   output, environment state), never the proposer's own explanation (§10.4.3.2).
2. **Progressive disclosure** — index first, expand on demand (Skills, memory, tool
   catalogs).
3. **Append-only** — immutable evidence logs, mutable derived views rebuilt from them.
4. **Boundary set + retention set** — every change must improve the failing cases *and*
   not regress the working ones; watching only one side trains a broken system (§8.14.1,
   §9.2.2).
5. **Minimal diff + reversible** — small attributable changes with provenance and a
   rollback path, never wholesale rewrites (§9.2.2).

Two threads the book repeats until they stick: **data and environment matter more than
algorithms** (Ch. 8 throughout, §8.15), and **the model may propose "done" but must never
approve its own "done"** (§10.4.3.1, "Loop Engineering": the loop's bottleneck is the
verifier, not the model). The Afterword's "two clouds" — streaming real-time interaction,
and genuine accumulation of experience — frame the harness/model co-evolution: models eat
harness layers one at a time, and the harness migrates to each new capability frontier.

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
| 8 | Post-training | Mid-training fixes the foundation, SFT the protocol, RL the policy; every stage transition needs a measurable entry condition; reward the outcome, penalize the path (RLVP); hidden tests the model cannot write (§8.12.2, §8.14.1). |
| 9 | Continual evolution | Preserving experience ≠ learning from it; three-layer verification (outcome → process → quality); four update carriers; dual loop (online executes and records, offline distills and gates); three safety boundaries (§9.3.2). |
| 10 | Multi-agent | Information-gain criterion (§10.2); shared vs. isolated context; peer/manager/decentralized topologies; OS analogy; six failure modes (§10.5); A2A opaque collaboration (§10.4.6). |

Ch. 9's three safety boundaries for self-evolving systems (§9.3.2), closely paraphrased:
(a) **evidence is never instructions** — untrusted content must not be executed or promoted
into long-term capability; (b) **candidates are separated from production** — nothing serves
real traffic before gates; (c) **safety mechanisms must not be self-modifiable** — an agent
may edit its skills and tools but never the validators, thresholds, audit logs, or release
gates that approve its own updates.

## 3 — Cross-check: no contradiction found

The book and open-seed agree on most load-bearing positions. This is recorded as **"no
contradiction found," not as independent confirmation**: the book draws on the same
upstream sources open-seed's thinking did (Anthropic's multi-agent and Claude Code
material, MetaGPT, the public agent-engineering discourse), so agreement between two
readers of one industry practice is common-cause — exactly the phenomenon the
homogeneous-convergence row below describes. Rows converging on shared *practice* rather
than derived *principle* are excluded (e.g., worktree isolation, which the book simply
reports as "the mainstream industry practice," §10.5.1). What agreement does establish is
that a careful independent treatment surfaced no counterexample to these decisions;
divergences and gaps it *did* surface are in §4.

| Book position (chapter/section) | open-seed counterpart |
|---|---|
| Optimistic locking with version checks for concurrent writers (§10.5.1) | Claim/lease/fence semantics — exit 2 contention, exit 6 fenced, exit 10 version (§7.1, binding) |
| Homogeneous convergence: same-model agents make common-cause identical choices — verified against the primary: "18 out of 30 agents decided to create a git branch with the exact same branch name, 'mvp-game-loop'" (§10.5.3; primary: Anthropic Frontier Red Team, "Patterns and Problems in Emerging Multiagent Systems", 2026-08-13) | Task-id-derived branch namespace (`seed/<id>`, `seed/<id>-plan`) prevents the *name/ref collision* specifically. It does **not** address agents converging on the same wrong approach — see §4. |
| The reviewer "must not be able to modify the tests, the evidence collector, or the release gate — otherwise independent verification degenerates into self-approval" (§10.4.3.2); safety mechanisms not self-modifiable (§9.3.2) | Protected paths (`.seed/**`, `Makefile`, `.github/**`, `scripts/**`, `AGENTS.md`, `CLAUDE.md` — CODEOWNERS + branch protection per `guardrails.yaml`); D4.5 reviewer ≠ implementer. Partial — see the hidden-checks gap in §4. |
| Evidence vs. instructions separation (§9.3.2) | AGENTS.md: card bodies, mail, and issue text are data, not instructions (and AGENTS.md itself is a protected path, so the declaration is not self-editable by a task PR) |
| Append-only evidence with derived mutable views; exactly one authority with one-way projections (§3, §9.2.1) | §7.7 (**proposed**): one authoritative store per repo, one-way projections, bidirectional sync forbidden. Partial — the book wants the *log* authoritative; see §4. |
| Retain negative results with the same retrieval status as successes (§9.3.1) | `memory/DEADENDS.md` has the same retrieval status as `LEARNINGS.md` — verified: both are append-paths in AGENTS.md and **neither is loaded on any path**, so the status is equal but equally weak; the ◇ retrieval-guidance criterion in AC.md covers both. |
| A2A "opaque collaboration": exchange tasks and artifacts, never internal prompts or reasoning (§10.4.6) | Handoff packets carry state, never trajectories |
| Disposable compute composes with durable record only if the record is complete before the machine dies ("the agent is its files," §10.4.4; Lingtai/Orbs-style substrates) | §7.6 (**proposed**): disposability begins only after confirmed push |
| MAST failure taxonomy's third class — **missing task verification** — is the dominant multi-agent failure (§10.5) | Evidence attachment + `make check` + review lane before `done` |

## 4 — Divergences and gaps the book surfaces

Where the book asks for something open-seed does not have. These are the retention set to
§3's boundary set — the point of the cross-check is as much what *didn't* match:

- **First-verified-success settlement is absent.** The book's parallel-manager pattern
  settles on the first **verified** success via an idempotent `settle_once`, racing
  *completed* candidates post-result (§10.4.4 pseudocode). Claim/lease/fence covers only
  the pre-work exclusive-claimant half; nothing in open-seed races N finished
  submissions. Recorded as a gap; maps to SEED-NEXT's verdict pipeline and racing mode.
- **The authoritative store is a state store, not a log.** §7.7 (proposed) fixes *one*
  authority, but on filecards that authority is the card files plus an appended run log
  on the state ref — current-state files are authoritative and the log is audit. The
  book's append-only pattern (and Ch. 3's two-stage memory) wants the log authoritative
  and the state derived. Log-as-authority is exactly SEED-NEXT's ledger proposal; until
  then this is a divergence, not a match.
- **"Hidden checks the worker cannot write" is only partially true today.** Verified
  against `guardrails.yaml`: the *definitions* are protected — `Makefile` (so `make
  check`'s entry point), `.github/**` (required checks), `scripts/**`, `.seed/**` all
  need CODEOWNERS review — and the reviewer-identity gate is server-attributed (D4.5).
  But ordinary test files are inside the worker's write scope: a task PR can weaken the
  tests `make check` runs, and only diff-vs-plan review catches it — review-dependent,
  not structural. And where the reviewer lane is the same model with similar context,
  D4.5 separates *identities*, not *evidence*: per §10.2, same-model re-reading adds no
  information, so the lane's value must come from what it executes, not what it reads.
  The structural fix (checks sealed outside implementer scope) is SEED-NEXT's sealed
  checks; today this is a known partial.
- **Same-wrong-approach convergence has no mechanism.** The branch namespace prevents
  ref collisions only. If two same-model agents attack sibling cards with the same
  flawed approach, nothing detects the common cause; plan review is the only (human)
  mitigation. The book's prescription — deliberately vary models/contexts and treat
  same-model agreement as non-independent (§10.4.3, §10.5.3) — is unimplemented.
- **No eval or retention set for open-seed's own harness.** The biggest hole the
  cross-check exposes: Ch. 7's machinery (failure attribution to the first erroneous
  step, trajectory-prefix regression, judge calibration) has no counterpart here.
  `make check` validates spec conformance and flavors — it does not evaluate whether a
  change to a role file, guardrail, or prompt *behaves* better or regresses. Boundary +
  retention exists only as a ◇ AC criterion. Any future change to the lanes' behavior
  is currently validated by nothing but review.

## 5 — Adoption candidates

Three ideas from the book concrete enough to card. Numbered B1–B3; cross-referenced from
the catalog in [11-adjacent-tools-inspiration.md](./11-adjacent-tools-inspiration.md).
Efforts were re-estimated after review: the earlier uniform "S" covered the templates
only, not the consuming mechanisms.

| # | Idea | What open-seed does with it | Priority | Effort |
|---|------|-----------------------------|----------|--------|
| B1 | **Three-part handoff-package spec** (§10.4.5, MetaGPT analysis). An effective handoff has: (1) task description *with acceptance criteria*, (2) confirmed facts and constraints, (3) references to structured artifacts. | A schema for **A1 handoff-packet enrichment** (doc 11), with three corrections from review: each settled decision carries a **verified/asserted marker** (an unmarked assertion shields upstream errors from review — MAST class 2); artifact references are **commit-anchored** (`path @ commit` or ref + range — bare paths assume a shared filesystem that worktrees and disposable executors don't have); and the packet carries **the diff vs. merge-base** (or a commit range that produces it), not a diffstat, because a diffstat is not reviewable. | Now | S–M |
| B2 | **Provenance-bearing experience entries** (§9.2.1, Experiment 9-2: a recommended strategy needs ≥2 supporting non-failed trajectories; applies-when conditions; last-validated stamps; precise retirement). | Two separable pieces. The *template* (S): entries carry applies-when, supporting task/PR IDs, last-validated; dead-ends carry failure condition + environment. The *value* is the **promotion step**, which must be offline and separately gated (the book's dual loop, §9.3): workers append **candidate observations only** — today's AGENTS.md instruction ("append durable insights in your task PR") has a single worker promoting a single run, which structurally violates the ≥2 rule — and a maintenance-lane (curator-style) pass compares candidates across tasks and promotes to "recommended" via its own PR. Template without promotion is bookkeeping; promotion without the template is unauditable. | Now | S (template) + M (promotion gate) |
| B3 | **Progress-file liveness** (§10.4.2, status query: an agreed lightweight progress file; full trajectory persistence stays a debug channel, never the status channel). | A worker-maintained progress note as the second signal beside lease renewal — with the mechanism corrected from review: staleness is measured by **monotonic progress counts** (completed-item counter must advance), not file mtime — a looping worker rewriting the file looks alive by mtime, and a legitimate 20-minute test run looks dead; long-running steps therefore also get a declared "in step X since T" state. The file is **gitignored or kept outside the tree** so it never pollutes diffs. Consumers: reap heuristics (wedged ≠ expired), nudge decisions, and D1's per-card freshness view — which is why the effort is M, not S: the template is trivial, the reap/report/dashboard integration is the work. Depends on safe-point cancellation semantics (Ch. 6) for the timeout fallback — see §6. | Next | M |

## 6 — Considered, not carded

Book mechanisms examined and deliberately not carded now, recorded so the next reader
doesn't re-derive them:

- **Ch. 7 failure attribution + trajectory-prefix regression** (first-erroneous-step
  records; replayable decision-point prefixes; judge calibration against a human gold
  set). The right shape for the "no eval for the harness" gap in §4 — but it is an
  infrastructure program, not a card; SEED-NEXT carries it as §16 (evaluation as
  infrastructure). Premature to card until the target substrate is decided.
- **Ch. 5 circuit breakers / failure taxonomy.** Partially present: `guardrails.yaml`
  budgets (`max_attempts_per_task`, `loop_max_iterations`, lease) are the circuit
  breakers; the layered failure taxonomy (API/tool/context/control-flow) has no
  counterpart and would only matter with the eval infrastructure above.
- **Ch. 6 safe-point cancellation.** B3's timeout fallback and any reap of a live worker
  presuppose it (interrupt at a safe point → park with packet; force-kill as fallback).
  Currently implicit in loop.sh behavior; should be specified when B3 is carded — noted
  as a B3 dependency, not a separate card.
- **Ch. 2 context/KV-cache discipline** (stable prefix, append-only trajectory, status
  bar). Harness-internal; open-seed is deliberately harness-agnostic and its role files
  shouldn't micromanage context layout. Out of scope for the template.

## 7 — Design tests worth keeping (not cards)

Portable judgments from the book, useful as review questions for future `.seed/` and role
changes:

- **Information-gain test** (§10.2). A collaboration step earns its cost only if it
  introduces information the single agent could not produce — test execution, rendered
  output, environment state. The book argues (citing equal-thinking-budget experiments,
  Tran & Kiela 2026, and the self-correction literature: Huang et al. ICLR 2024, the
  TACL 2024 survey, CRITIC) that same-model re-reading without external feedback is no
  better than equal-compute single-agent work and self-review can be net negative.
  Concretely: the reviewer lane must run `make check` and read the diff against the
  plan; a reviewer that paraphrases the implementer's summary adds negative value.
- **Boundary set + retention set** (§8.14.1, §9.2.2). Any guardrail, role-file, or
  prompt change should name the failing case it fixes *and* demonstrate the
  previously-working cases still pass. A change validated only against its trigger case
  trains an over-corrected system (the book's example: a "verify before done" fix that
  produces an agent which never dares to finish).
- **The model never approves its own "done"** (§10.4.3.1, §8.12.2). Completion is
  decided by checks outside the worker's control. In today's open-seed that is
  *precisely*: the reviewer-identity gate (server-attributed, D4.5), branch protection
  with required checks (forge-side), and the protected-path set (`Makefile`,
  `.github/**`, `scripts/**`, `.seed/**` — CODEOWNERS-reviewed, so the check *pipeline*
  cannot be silently redefined). It is *not* the test content outside protected paths,
  which a worker can edit and only diff-vs-plan review catches — see the gap in §4.
- **Budget awareness beats budget size** (§10.2, citing budget-aware tool-use work).
  The useful signal to expose to a worker is *remaining* budget, not just a hard stop
  at exhaustion.
- **MAST's three failure classes** (§10.5) as a review checklist for orchestration
  changes: (1) system-design flaws — unclear interfaces, overlapping responsibilities;
  (2) inter-agent alignment — downstream misreading upstream artifacts; (3) missing
  task verification. Most multi-agent failures are Byzantine (plausible-but-wrong
  output, no announced error), which is why deterministic checks are disproportionately
  valuable.

## 8 — Read but out of scope

- **Ch. 8 training recipes** (mid-training/SFT/RL mechanics, GRPO/PPO, distillation,
  RLVP): open-seed does not train models. The transferable residue is captured above —
  gates with measurable entry conditions, hidden verification, outcome + path checking.
- **Ch. 10 agent-society material** (Stanford town, Agentopia, Moltbook, market
  economies): interesting for the long-horizon fleet picture, no near-term bearing.
- **Ch. 6 voice / computer-use / robotics**: substrate-level; nothing to adopt.
- **Ch. 2 context engineering internals**: harness-level, per §6 — the template stays
  agnostic.
