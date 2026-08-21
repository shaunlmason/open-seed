# Research: SOTA Landscape — Harness-Native Primitives, Task Tracking, Guardrails, Methodology

> Researched 2026-08-21 for the open-seed design study. Everything below was verified against
> live documentation and current sources unless flagged as uncertain.

---

## 1. Claude Code Native Primitives

Claude Code (code.claude.com/docs) now ships the richest checked-in-tooling surface of any harness. The complete inventory, with repo-shareability:

**CLAUDE.md memory.** Read at session start from the project root (plus parent directories and nested per-directory files in monorepos), `~/.claude/CLAUDE.md` for user scope, and enterprise managed locations. Claude Code additionally builds "auto memory" — learnings it saves across sessions without user authoring. CLAUDE.md is fully repo-checked-in. It can `@`-import other files, which is the standard bridge to AGENTS.md (`@AGENTS.md` in a one-line CLAUDE.md).

**Skills — and the collapse of slash commands into them.** Skills live at `.claude/skills/<name>/SKILL.md` (project, checked in), `~/.claude/skills/` (personal), and inside plugins (namespaced). Custom slash commands have been *merged into skills*: `.claude/commands/deploy.md` and `.claude/skills/deploy/SKILL.md` both create `/deploy`; commands files keep working but skills are the recommended form. Key properties: YAML frontmatter (`description`, `allowed-tools`, `disable-model-invocation`, subagent-execution and dynamic-context-injection extensions), progressive disclosure (only name+description load at startup; the body loads on use), nested `.claude/skills/` in monorepo subdirectories, and live reload. Critically, Claude Code skills follow the **Agent Skills open standard at agentskills.io** — verified: the format was originated by Anthropic and released as an open standard, and the adopter showcase now lists Gemini CLI, Cursor, GitHub Copilot/VS Code, OpenCode, Codex/ChatGPT, Amp, Goose, Kiro, Roo Code, OpenHands, Factory, JetBrains Junie, Letta, Tabnine, and dozens more. This is the single most important convergence fact for a template repo.

**Subagents.** Markdown+frontmatter files at `.claude/agents/` (project, checked in — the docs explicitly recommend committing them), `~/.claude/agents/` (user), plugin `agents/`, or `--agents` flag. Frontmatter now covers `name`, `description`, `tools`/`disallowedTools`, `model`, `permissionMode`, `maxTurns`, `memory`, `skills` preloading, `mcpServers`, `isolation: worktree` (isolated git worktree per subagent), and `background: true`. Each subagent gets an isolated context window; only the summary returns. Subagent definitions double as **teammate templates for agent teams**.

**Hooks.** Configured in `.claude/settings.json` (checked in), `settings.local.json` (gitignored), user settings, managed settings, plugin `hooks/hooks.json`, and even skill/subagent frontmatter. The event surface has grown far beyond PreToolUse/PostToolUse: `SessionStart`, `Setup`, `UserPromptSubmit`, `Stop`, `PermissionRequest`, `PostToolUseFailure`, `SubagentStart/Stop`, `TaskCreated`, `TaskCompleted`, `TeammateIdle`, `PreCompact`/`PostCompact`, `WorktreeCreate`, `FileChanged`, and more. Five hook types now exist: **command** (shell, exit code 2 = block, or structured JSON `permissionDecision`), **HTTP**, **MCP tool**, **prompt** (single-turn model evaluation), and **agent** (spawns a verification subagent — experimental). Hooks are the deterministic policy-enforcement layer, and `TaskCompleted`/`TeammateIdle` hooks are explicitly documented as *quality gates for agent teams* (exit 2 keeps a teammate working or blocks task completion).

**Permissions/settings.** `.claude/settings.json` carries `permissions` (allow/ask/deny rules like `Bash(npm run test:*)`), `env`, hooks, and sandbox settings — checked in and merged with user/local/managed layers by documented precedence.

**Sandboxing.** The sandboxed Bash tool is now built in: macOS Seatbelt; Linux/WSL2 bubblewrap + socat network proxy, with an optional seccomp filter. OS-enforced filesystem writes limited to the working dir, network egress domain-allowlisted via proxy, auto-allow mode runs sandboxed commands without prompting. Configurable in settings (thus committable). Separate docs cover devcontainers/custom containers/VMs as heavier isolation.

**MCP.** `.mcp.json` at the project root is the checked-in, team-shared MCP server config (with user- and local-scope alternatives, env-var expansion for secrets).

**Plugins & marketplaces.** A plugin bundles skills, agents, hooks, `.mcp.json`, LSP servers, background monitors, `bin/` executables, and default settings, under a `.claude-plugin/plugin.json` manifest. Marketplaces are git repos with `.claude-plugin/marketplace.json`; Anthropic runs official (curated) and community (reviewed, SHA-pinned) marketplaces. Crucially for templates: **a repo can declare plugins in `.claude/settings.json`**, and cloud sessions install repo-declared plugins at session start — so a template can ship a plugin dependency list, not just vendored files.

**Agent teams (the "swarm" feature).** Real and shipped, but still experimental as of v2.1.x: enabled with `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1` (shipped alongside Opus 4.6 in early February 2026). A lead session spawns teammates — full independent Claude Code sessions — coordinated through a **shared task list** (pending/in-progress/completed, with dependencies and file-locked claiming) and **mailbox messaging** (JSON inboxes under `~/.claude/teams/`). Teammates self-claim unblocked tasks, message each other directly, and can be required to submit plans for lead approval. Teammates can be spawned from checked-in subagent definitions. Important for template design: team/task state lives under `~/.claude/`, **not** in the repo — the repo-checked-in leverage points are subagent definitions, the TaskCreated/TaskCompleted/TeammateIdle hooks, and permissions. Known limitations: no session resumption of in-process teammates, one team per session, no nested teams, lead is fixed.

**Dynamic workflows.** A newer orchestration primitive (v2.1.154+): Claude writes a JavaScript orchestration script (`agent()` / `pipeline()` fan-out, up to 16 concurrent / 1,000 agents per run) executed by a runtime outside the conversation, keeping intermediate results out of context. Saved workflows are **checked into `.claude/workflows/`** ("shared with everyone who clones the repo") or distributed via plugin `workflows/` directories. This is the codified, rerunnable counterpart to agent teams' improvisational coordination, and it's directly shareable in a template.

**Headless/SDK/CI.** `claude -p` headless mode; the Claude Agent SDK (TypeScript/Python) for custom agents; `anthropics/claude-code-action@v1` (GA) for GitHub Actions; a separate GitHub Code Review product; GitLab CI/CD support; Routines (cloud-scheduled runs) that load repo-committed `.claude/skills/` and repo-declared plugins.

**Checked-in summary for Claude Code:** `CLAUDE.md`, `.claude/skills/`, `.claude/commands/` (legacy), `.claude/agents/`, `.claude/settings.json` (permissions + hooks + env + sandbox + plugin declarations), `.claude/workflows/`, `.mcp.json`, `.github/workflows/*` using claude-code-action, and optionally a vendored plugin/marketplace. Not checked in: agent-team state, auto memory, `settings.local.json`, `~/.claude/*`.

---

## 2. Other Harnesses

**OpenAI Codex CLI.** Instructions: `AGENTS.md` (the format OpenAI proposed in August 2025), with nested per-directory files ("closest wins") and `AGENTS.override.md` for replacement rather than accretion; `project_doc_fallback_filenames` in config can make Codex read CLAUDE.md too. Config: `~/.codex/config.toml` (model, `approval_policy`, `sandbox_mode`, MCP servers, profiles) — primarily user-level, with a managed `requirements.toml` layer for admins (including `allow_managed_hooks_only`, implying Codex now has lifecycle hooks; the project-level hook layout is lower-confidence). Codex has native sandboxing (macOS Seatbelt, Linux landlock/seccomp) with approval policies. **Codex adopted Agent Skills**: developers.openai.com/codex/skills exists; guides describe repo-level and `~/.codex`-level skill directories.

**Gemini CLI.** `GEMINI.md` context files (hierarchical); custom slash commands as TOML; **extensions** as its plugin unit — bundling MCP servers, commands, context files, and now skills, distributed via a public extensions gallery. **Gemini CLI adopted Agent Skills** with a documented discovery hierarchy: built-in → extension skills → user (`~/.gemini/skills/` **or the `~/.agents/skills/` alias**) → workspace (`.gemini/skills/` **or `.agents/skills/`**). The `.agents/skills/` alias is notable: a harness-neutral checked-in skills directory is emerging.

**OpenCode.** `opencode.json` (project root, checked in) configures providers, models, permissions (allow/ask/deny), agents, plugins, MCP. Agents as markdown in `.opencode/agent/*.md` with frontmatter. Reads AGENTS.md natively; supports Agent Skills.

**Cursor.** `.cursor/rules/*.mdc` scoped rules (globs, alwaysApply), reads AGENTS.md, and now supports Agent Skills. Cursor CLI reuses the same rule system.

**GitHub Copilot coding agent.** `.github/copilot-instructions.md` plus AGENTS.md support; **custom agents** as YAML-frontmatter markdown profiles in repo; **Copilot skills** per the Agent Skills standard; MCP config for the cloud agent; runs in its own ephemeral Actions-based environment with firewall.

**AGENTS.md standard status.** Governance transferred to the **Agentic AI Foundation under the Linux Foundation** (late 2025); emerged from OpenAI Codex, Amp, Google Jules, Cursor, and Factory. Adopted by 20+ tools natively and 60,000+ OSS repos (secondary-source figure). Spec is deliberately minimal: plain markdown, no required fields, nested files with closest-wins; a v1.1 proposal adds optional frontmatter — proposal status, not landed. **Claude Code remains the notable holdout**, reading CLAUDE.md; the universal workaround is a CLAUDE.md that imports AGENTS.md. Practitioner consensus ("2026 default"): write AGENTS.md as source of truth, add tool-specific files only as shims.

---

## 3. Task Tracking for Agents

**Beads (steveyegge/beads)** is the clear leader in agent-native issue tracking (~26.5k stars, very active). Design: a distributed **graph** issue tracker where issues carry typed edges — blocking dependencies plus `relates-to`, `duplicates`, `supersedes`, `replies-to` — and **hash-based IDs** (`bd-a1b2`, hierarchical `bd-a3f8.1.1`) chosen specifically so parallel agents creating issues never collide on sequential numbering. The killer query is `bd ready`: only unblocked work, which turns the tracker into a work-dispatch queue for agents. `bd init` writes agent instructions into **AGENTS.md** by default; agents run `bd prime` for workflow context, then `bd ready` / `bd show` / `bd update --claim` / `bd close`. **Important design evolution**: the original (Oct 2025) storage was SQLite-cached JSONL synced through git; current beads is **powered by Dolt** (version-controlled SQL database) — embedded Dolt in `.beads/embeddeddolt/` by default or an external `dolt sql-server` for concurrent writers — with `.beads/issues.jsonl` retained *only as an export/interchange format, explicitly not the source of truth*. Dolt brings cell-level merge, native branching, and `bd dolt push/pull` sync. A template that vendors beads should treat `.beads/` per beads' own gitignore guidance and pin a version, since schema migrations are ongoing.

**Gas Town** (yegge.ai/gastown; open-sourced January 1, 2026, MIT) is the orchestrator built on beads: worker agents ("polecats") with persistent identities and ephemeral sessions, a "Mayor" coordinator, "Witness" monitors, a "Refinery" merge-queue that serializes changes, all persisting every task and agent note as beads so crashed/handed-off sessions resume from the ledger. It drives Claude Code, Codex, Copilot, and Gemini at 20–30 parallel agents. The ecosystem grew in 2026: **Gas City** (declarative, enterprise-scale successor) and **Wasteland** (March 2026 — cross-Gas-Town shared work board). Yegge's thesis (from "Revenge of the Junior Developer," March 2025, onward): the unit of leverage is shifting from the agent to the *fleet*, and the bottleneck is durable coordination state, not model capability.

**Spec-driven development (SDD) frameworks:**
- **GitHub spec-kit** (~80k stars): `specify` CLI bootstraps `/constitution`, `/specify`, `/plan`, `/tasks`, `/implement` command sets into 30+ agents' native command formats; artifacts are markdown specs/plans/task lists in-repo. Greenfield-leaning, opinionated phases.
- **BMAD-METHOD** (~37k stars): full phase-gated agile simulation with persona agents (PM, architect, dev, QA); heaviest process and heaviest token cost.
- **OpenSpec**: lightweight vendor-neutral spec/change-proposal format; best documented for brownfield.
- **Task Master (eyaltoledano/claude-task-master**, ~25k stars): PRD → `tasks.json` with dependencies/complexity scores, consumed via MCP or CLI. Active but architecturally an AI-parsed *derived* artifact rather than a durable coordination substrate.

**Comparison for agent workflows.** Markdown checklists are universally readable and diff-reviewable but degrade under parallelism: no locking, no dependency queries, merge conflicts, and agents "lose" or hallucinate state. Graph trackers (beads) give collision-free IDs, `ready`-work dispatch, dependency-aware ordering, and cross-session memory — the properties multi-agent fleets actually need — at the cost of a binary/daemon dependency and a young, still-moving schema. External trackers (GitHub Issues/Linear via MCP) give human visibility and zero new infra, but are slow for high-frequency agent churn, rate-limited, sequential-ID-collision-prone under parallel creation, and put agent working memory outside the repo clone. Claude Code's own native task list (agent teams) is session-scoped and machine-local — a coordination bus, not a durable tracker; beads and it are complementary, not competing.

---

## 4. Guardrails & Quality Gates

The 2026 guardrail stack has settled into layers:

**Isolation (outermost).** In ascending strength: Claude Code's built-in OS sandbox (Seatbelt/bubblewrap + network proxy — per-command, zero setup); Codex's native sandbox with approval policies; devcontainers (standard `.devcontainer/` — checked in, reproducible; Anthropic maintains a reference devcontainer with a firewall init script; community caveats: default images ship passwordless sudo, and Docker-socket exposure undermines isolation); Docker Sandboxes/microVMs (Firecracker-class — strongest boundary, marketed for "YOLO-mode" agent runs); and cloud execution environments (Claude Code on the web, Copilot coding agent) which are isolated by construction. Practitioner rule: full-auto permission bypass only inside a container/VM with mounted-secrets minimization.

**Permission allowlists.** Checked-in allow/ask/deny rules (Claude Code `.claude/settings.json`; OpenCode `opencode.json`; Codex approval_policy + requirements.toml managed layer). These are the *declarative* layer.

**Hooks as programmable policy.** The *imperative* layer: PreToolUse deny rules (block `rm -rf`, writes outside repo, force-push), PostToolUse formatters/linters, Stop-event test runs, secret-redaction on Bash, and — new this cycle — TaskCompleted/TeammateIdle gates that hold multi-agent work to done-criteria.

**CI as backpressure.** The Ralph/Huntley framing has gone mainstream: agents will merge whatever passes, so the test suite, typechecker, and lint config *are* the guardrail; teams invest in making `make check` fast, deterministic, and comprehensive. Branch-protection patterns for agent-heavy repos: agents never push to default branches; required status checks + required review; merge queues (or Gas Town's Refinery) to serialize landings from parallel agents; CODEOWNERS to force human review on sensitive paths; scoped bot identities so agent commits are attributable.

**Agent code review.** Now table stakes: CodeRabbit (broadest platform support + a Claude Code plugin creating build→review→fix loops), Anthropic's GitHub Code Review product (GA 2026), Copilot code review, Greptile, Qodo, Cursor Bugbot. Claude Code also ships bundled `/code-review` and `/security-review` skills. The emerging pattern is *reviewer-role separation*: a checked-in reviewer subagent (read-only tools, plan mode) distinct from the implementer.

**Secret hygiene.** Layered: GitHub push protection + secret scanning; pre-commit/PreToolUse gitleaks/trufflehog hooks; keeping secrets in env/`settings.local.json`/`.mcp.json` env-expansion rather than any checked-in file; sandbox network allowlists limiting exfiltration paths; treating fetched web/comment content as untrusted.

---

## 5. Methodology SOTA

**The Ralph Wiggum loop** (Geoffrey Huntley, went viral late 2025; "the year of the Ralph loop" discourse through 2026): run an agent in a dumb `while true` shell loop against the same prompt file; the filesystem and git history are the memory; each iteration gets fresh context. Its insight — "deterministically bad in a non-deterministic world" — is that predictable failure modes can be fenced with *signs* (guardrail notes in the prompt/repo), *specs*, and *backpressure* (tests/CI), which beats the unpredictable failure surface of elaborate multi-agent frameworks.

**Context engineering.** Anthropic's "Effective context engineering for AI agents" (Sept 2025) codified the vocabulary: context is a finite resource degraded by "context rot"; the remedies are compaction, structured note-taking (external memory files), and sub-agent architectures whose isolated windows return summaries. The June 2025 "How we built our multi-agent research system" remains the canonical orchestrator-worker reference: parallel subagents beat single-agent on breadth-first research by ~90% on their eval, at ~15× token cost — the cost/benefit asymmetry that still governs when to fan out. Consensus: multi-agent for parallelizable read-heavy work; single agent for tightly coupled write-heavy work.

**Parallel worktrees.** Git worktrees are now a first-class native concept: Claude Code has worktree docs, `isolation: worktree` subagent frontmatter, and WorktreeCreate hooks; desktop apps productize agent-per-worktree. One-branch-per-agent with a serializing merge step (merge queue or Refinery) is the consensus answer to parallel write conflicts.

**Spec-driven development debate.** Contested. Pro-SDD (GitHub, Kiro, BMAD, Tessl): specs as durable source-of-truth reduce rework (claims of 60–80% are vendor/community figures, not rigorous). Skeptics: heavyweight phase-gating burns tokens and calendar on tasks agents could just do, and specs go stale exactly like documentation always has; the Ralph school treats a *short* spec plus strong backpressure as sufficient. Observed migration patterns suggest teams tune process weight to project phase rather than adopting one framework forever. A template should therefore ship a *thin* spec convention with optional framework adapters, not a hard SDD dependency.

**Team-scale practice** per Yegge and Anthropic's engineering posts: checked-in, versioned agent configuration reviewed like code; durable work ledgers outside any single context window; humans shifting to specification, review, and exception-handling.

---

## 6. Synthesis

### (a) Native-primitive inventory a template can rely on, per harness

| Primitive | Claude Code | Codex CLI | Gemini CLI | OpenCode | Copilot agent | Cursor |
|---|---|---|---|---|---|---|
| Repo instructions | CLAUDE.md (nested) | AGENTS.md (nested, override) | GEMINI.md + AGENTS.md | AGENTS.md | copilot-instructions.md + AGENTS.md | .cursor/rules + AGENTS.md |
| Skills (Agent Skills std) | `.claude/skills/` | repo + `~/.codex` skills | `.gemini/skills/` or **`.agents/skills/`** | `.opencode/skill` + std | Copilot skills | Cursor skills |
| Custom agents | `.claude/agents/` | (profiles limited) | via extensions | `.opencode/agent/` | agent profiles (.md) | limited |
| Hooks | settings.json, 30+ events, 5 types | config/requirements hooks (newer) | limited | plugins | — | — |
| Permissions | settings.json allow/ask/deny | approval_policy/sandbox_mode | settings | opencode.json | env firewall | — |
| MCP config in repo | `.mcp.json` | config.toml (user-level) | extensions/settings | opencode.json | repo MCP config | .cursor/mcp.json |
| Plugin/bundle unit | plugins + marketplaces | (emerging) | extensions + gallery | plugins | — | — |
| Sandbox | native OS sandbox | native OS sandbox | — | — | cloud-isolated | — |
| Multi-agent | subagents, agent teams (exp.), dynamic workflows, Agent SDK | threads | subagents (early) | subagents | parallel cloud tasks | background agents |

### (b) Cross-harness common denominator
Three things are genuinely portable in August 2026: **(1) AGENTS.md** (Linux Foundation-stewarded, everything but Claude Code natively, and Claude Code via one-line import); **(2) Agent Skills** (`SKILL.md` folders per agentskills.io — now adopted by essentially every major harness; Gemini's `.agents/skills/` alias hints at a converged directory, but per-harness directories still differ); **(3) MCP** for external tools. Everything else — hooks, permissions, subagent definitions, plugins — remains harness-specific. A template's portable core is AGENTS.md + a skills tree + `.mcp.json`, with thin per-harness shims.

### (c) Task-tracking substrate options
1. **Markdown specs/checklists in-repo** — zero deps, human-reviewable, fine for single-agent/Ralph loops; breaks under parallel agents. Ship as the default floor.
2. **Beads** — purpose-built for the multi-agent case (hash IDs, dependency graph, `bd ready`, AGENTS.md integration, memory across sessions); costs a binary dependency and schema churn risk; the natural opt-in tier, version-pinned.
3. **External trackers via MCP** (GitHub Issues/Linear) — best human visibility; poor as high-frequency agent working memory; best used as the *human-facing mirror* synced from the in-repo substrate.
4. **Harness-native task lists** (Claude Code agent teams) — free intra-session coordination, but ephemeral and machine-local; never a system of record.

The strongest current architecture is layered: durable graph tracker in-repo (beads or markdown), harness task list for intra-session dispatch, external tracker mirrored for humans.

### (d) Guardrail stack options
Defense in depth, all largely checked-in-able: (1) isolation — built-in OS sandbox settings for interactive use; a hardened `.devcontainer/` for full-auto; microVM sandboxes for the paranoid tier; (2) declarative permission allowlists in each harness's settings file; (3) hooks for deterministic policy; (4) CI backpressure — fast comprehensive checks + branch protection + required review + merge queue, with agents confined to feature branches; (5) agent code review plus security scanning; (6) secret hygiene — push protection, gitleaks hooks, secrets only via env/local files, network egress allowlists.

### (e) Converging vs. contested
**Converged:** AGENTS.md for instructions; Agent Skills for procedural knowledge; MCP for tools; markdown-with-frontmatter as the universal agent-config file format; subagent context isolation; fresh-context-per-task with state on disk; sandbox-by-default trajectories; agent code review on every PR; git worktrees for parallelism. **Contested:** orchestration topology (native teams/workflows vs. external orchestrators like Gas Town vs. dumb Ralph loops — with native features rapidly absorbing the middle); task substrate (markdown vs. beads-style graph vs. external — no standard); SDD process weight; plugin distribution; hooks (no cross-harness standard at all); and where durable agent memory should live. A template repo should build on the converged layer, make the contested layers pluggable, and expect the harness vendors to keep absorbing orchestration features natively — February-to-August 2026 alone brought agent teams, dynamic workflows, five hook types, and universal skills adoption.

---

## Sources

- Claude Code docs: https://code.claude.com/docs/en/overview · /skills · /sub-agents · /hooks · /agent-teams · /workflows · /sandboxing · /plugins · /github-actions
- Agent Skills standard: https://agentskills.io · https://github.com/agentskills/agentskills
- AGENTS.md: https://agents.md
- Codex: https://github.com/openai/codex/blob/main/docs/config.md · https://developers.openai.com/codex/skills
- Gemini CLI: https://geminicli.com/docs/cli/skills/ · https://geminicli.com/docs/extensions/writing-extensions/
- OpenCode: https://opencode.ai/docs/config/ · https://opencode.ai/docs/agents/
- Copilot: https://docs.github.com/en/copilot/concepts/agents/cloud-agent/about-custom-agents · https://docs.github.com/en/copilot/concepts/agents/about-agent-skills
- Beads: https://github.com/steveyegge/beads · https://steveyegge.github.io/beads/
- Gas Town: https://yegge.ai/gastown · https://steve-yegge.medium.com/welcome-to-gas-town-4f25ee16dd04
- Spec frameworks: https://github.com/github/spec-kit · https://github.com/eyaltoledano/claude-task-master
- Ralph: https://ghuntley.com/ralph · https://github.com/snwfdhmp/awesome-ralph · https://www.humanlayer.dev/blog/brief-history-of-ralph
- Anthropic engineering: https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents · https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents · https://claude.com/blog/building-multi-agent-systems-when-and-how-to-use-them
- Review/guardrails: https://github.com/anthropics/claude-code-action · https://docs.coderabbit.ai/cli/claude-code-integration
