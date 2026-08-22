# open-seed

A template repository for new projects that ships standardized, checked-in tooling for
multi-agent orchestration, task tracking, and guardrails.

## Status: design complete — implementing v1

A survey of the multi-agent orchestration ecosystem (all 180 projects in
[awesome-agent-orchestrators](https://github.com/andyrewlee/awesome-agent-orchestrators)
plus the current SOTA in harness-native primitives, task tracking, and guardrails)
grounded the design; the design is settled and v1 implementation follows the build plan.
Agents building open-seed itself: read [`docs/CONTRIBUTING-AGENTS.md`](docs/CONTRIBUTING-AGENTS.md) first (authority order, binding decisions). The root [`AGENTS.md`](AGENTS.md) is the template's user-facing agent contract (a Phase 3 artifact).

- **[Design](docs/design-options.md)** — the design authority: decisions, team layer, risks, glossary, resolved defaults
- **[Build plan](docs/build-plan.md)** — v1 phase ordering and per-phase acceptance criteria
- **[Research reports](docs/research/)** — per-category evidence:
  1. [Terminal TUI orchestrators](docs/research/01-terminal-tui-orchestrators.md)
  2. [Desktop & web orchestrators (A)](docs/research/02-desktop-web-orchestrators-a.md)
  3. [Desktop & web orchestrators (B)](docs/research/03-desktop-web-orchestrators-b.md)
  4. [Autonomous loop runners (Ralph family)](docs/research/04-autonomous-loop-runners.md)
  5. [Multi-agent swarms](docs/research/05-multi-agent-swarms.md)
  6. [Autonomous task runners (issue/CI-driven)](docs/research/06-autonomous-task-runners.md)
  7. [Agent infrastructure & primitives](docs/research/07-infrastructure-primitives.md)
  8. [Personal assistants & inactive projects](docs/research/08-assistants-and-inactive.md)
  9. [SOTA landscape (harnesses, beads, guardrails, methodology)](docs/research/09-sota-landscape.md)
  10. [Org control planes: Paperclip vs Gas Town vs build-your-own](docs/research/10-org-control-planes.md)
- **[Inspiration deep dives](docs/research/inspirations/)** — implementation-grade format/schema extraction from the key inspiration projects:
  1. [Git-native task substrates (beads, gnap, ORCH, squad, tick-md)](docs/research/inspirations/01-git-native-task-substrates.md)
  2. [Ralph loop implementations (ralphex, ralph-claude-code, dex, wreckit, martin-loop)](docs/research/inspirations/02-ralph-loop-implementations.md)
  3. [Governance & gates (loop-engineering, orc, antfarm, loki-mode, kodo)](docs/research/inspirations/03-governance-and-gates.md)
  4. [Workflow-as-config (tutti, agent-runbook, crewplane, Fusion, Archon)](docs/research/inspirations/04-workflow-as-config.md)
  5. [Skills packaging (skillfold, sub-agents-skills, Agent Skills spec, humanlayer trilogy)](docs/research/inspirations/05-skills-packaging.md)
  6. [CI-native automation (gh-aw, aeon, sortie, contrabass, claude-code-action)](docs/research/inspirations/06-ci-native-automation.md)
  7. [Lifecycle contracts (superset, octomux, vibe-tree, amux, dmux, agent-deck, tmux-ide)](docs/research/inspirations/07-lifecycle-contracts.md)
  8. [Coordination mechanics (swarm-protocol, wit, hcom, shogun, CCB, claudexor)](docs/research/inspirations/08-coordination-mechanics.md)
  9. [Harness CLIs: headless surfaces, adapter contract, permission-tier fidelity](docs/research/inspirations/09-harnesses.md)
