# open-seed

A template repository for new projects that ships standardized, checked-in tooling for
multi-agent orchestration, task tracking, and guardrails. The contract is **files, not
an app**: a task port with pluggable backends, coordination state on a dedicated git
ref, a plan→implement→receipt evidence chain, enforced guardrails, and a loop runner:
all reviewable, all degrading gracefully from a fleet of agents down to one human with
no engine installed.

## Quickstart

```sh
# after instantiating the template:
scripts/seed init                  # create the coordination state ref
scripts/seed init-github           # print the server-side protection checklist
scripts/seed task create --title "First task" --actor you
scripts/loop.sh --once             # or work the lifecycle by hand
make smoke                         # deterministic end-to-end proof, no model needed
```

**Read next: [the handbook](docs/handbook.md)**: setup, the task lifecycle, the loop,
guardrails, the degradation ladder, and scaling. Agents working in an instantiated repo
follow [`AGENTS.md`](AGENTS.md). The `seed` engine is a pinned, hash-verified binary
from [open-seed-engine](https://github.com/shaunlmason/open-seed-engine); the shim
fetches it on first use.

## Status: v1 + v2 scope complete

All seven build-plan phases are done, and the §7.3 v2 scope has shipped
(workflow engine, skills lockfile, mailboxes + handoff packets, multi-squad
routing, MCP transport, the fastcards builtin, the beads/paperclip/linear/jira
adapters, the mirror exporter + label router). open-seed dogfoods its own
port: remaining work is tracked as cards on this repo's own seed-state ref
(`scripts/seed task ready --actor you`). What remains for an instantiating
owner is activation, not building: flipping the lane secrets and the autonomy
tier (handbook, "Activating the agent lanes").

## Design & research

Agents building open-seed itself: read
[`docs/CONTRIBUTING-AGENTS.md`](docs/CONTRIBUTING-AGENTS.md) first (authority order,
binding decisions).

- **[Handbook](docs/handbook.md)**: user-facing conventions
- **[Design](docs/design-options.md)**: the design authority: decisions, team layer, risks, glossary, resolved defaults
- **[Architecture map](docs/architecture.md)**: the cross-repo map: layering, the port, the evidence chain, where each gate grounds
- **[Build plan](docs/build-plan.md)**: v1 phase ordering and per-phase acceptance criteria
- **[Research reports](docs/research/)**: per-category evidence:
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
- **[Inspiration deep dives](docs/research/inspirations/)**: implementation-grade format/schema extraction from the key inspiration projects:
  1. [Git-native task substrates (beads, gnap, ORCH, squad, tick-md)](docs/research/inspirations/01-git-native-task-substrates.md)
  2. [Ralph loop implementations (ralphex, ralph-claude-code, dex, wreckit, martin-loop)](docs/research/inspirations/02-ralph-loop-implementations.md)
  3. [Governance & gates (loop-engineering, orc, antfarm, loki-mode, kodo)](docs/research/inspirations/03-governance-and-gates.md)
  4. [Workflow-as-config (tutti, agent-runbook, crewplane, Fusion, Archon)](docs/research/inspirations/04-workflow-as-config.md)
  5. [Skills packaging (skillfold, sub-agents-skills, Agent Skills spec, humanlayer trilogy)](docs/research/inspirations/05-skills-packaging.md)
  6. [CI-native automation (gh-aw, aeon, sortie, contrabass, claude-code-action)](docs/research/inspirations/06-ci-native-automation.md)
  7. [Lifecycle contracts (superset, octomux, vibe-tree, amux, dmux, agent-deck, tmux-ide)](docs/research/inspirations/07-lifecycle-contracts.md)
  8. [Coordination mechanics (swarm-protocol, wit, hcom, shogun, CCB, claudexor)](docs/research/inspirations/08-coordination-mechanics.md)
  9. [Harness CLIs: headless surfaces, adapter contract, permission-tier fidelity](docs/research/inspirations/09-harnesses.md)
