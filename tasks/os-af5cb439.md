---
id: os-af5cb439
title: 'Q5: dedicated machine identity for seed-state pushes (ref-push scoping, R10)'
state: ready
priority: P3
squad: core
created_at: "2026-08-24T02:20:42Z"
updated_at: "2026-08-24T04:31:50Z"
---

Design R10 (docs/design-options.md:287): state-ref integrity is push-access-deep — anyone who can push `seed-state` can bypass the shim (clear a rejection list, forge lease fields, rewrite the log, forge an operator verb). The named mitigation is to "scope which principals hold state-ref push access (Q5)".

Status: deliberately deferred as the optional Q5 hardening in os-18135882 (evidence, 2026-08-23): the `seed-state` ref is still contributor-writable — any contributor can push to it directly, which is exactly the forge surface R10 describes. No card, branch, or PR tracks it anywhere (verified across worktrees, open PRs #54-56, and the engine repo, 2026-08-24).

Scope:
- Introduce a dedicated machine identity (a GitHub App or fine-grained PAT with `contents:write` scoped to the repo) that is the only principal allowed to push `seed-state`; contributors read via fetch.
- Enforce via a ruleset (branch, refs/heads/seed-state, `pull_requests`-style push restriction / custom rule) rather than admin-only branch protection, so the maintenance workflow keeps working and humans stay read-only.
- Update the os-18135882 checklist (docs/handbook.md §1) and `seed init-github` output: item 4 moves from "optional Q5 hardening" to a documented, reproducible step with the ruleset shape.
- Keep the degradation ladder honest: a repo that has not applied Q5 still works; document the trust assumption (R10 audit claims are conditional on push scoping).

Acceptance: a contributor token without the machine identity is refused on push to `seed-state` (ruleset 403) while the maintenance workflow (running as the machine identity) pushes normally; the handbook documents the identity, the ruleset, and the residual trust assumption.
