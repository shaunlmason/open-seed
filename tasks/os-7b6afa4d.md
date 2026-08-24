---
id: os-7b6afa4d
title: 'CI: verify server-side seed-state protections (Phase 2 API-side check, unlanded)'
state: in_progress
priority: P2
squad: core
claim:
    actor: opencode
    token: c-54f45e1c3efbff00
    claimed_at: "2026-08-24T02:29:59Z"
    lease_expires: "2026-08-24T03:29:59Z"
created_at: "2026-08-24T02:19:59Z"
updated_at: "2026-08-24T02:29:59Z"
---

Build plan Phase 2 shipped `seed init-github` as a printed checklist only, deferring the API-side verification "to Phase 5's workflows" (docs/build-plan.md:100). Phase 5 closed (build-plan.md:184) and none of the five live workflows performs that read-back: no `.github/workflows/*.yml` calls the GitHub API to confirm the branch protection / rulesets on `seed-state` and the tag rules are still active (grep of .github/workflows shows no protection/ruleset read-back).

So the protections applied by os-18135882 (main protection; seed-state no-force-push/no-delete; seed-anchor create-only; release-tag immutable) are enforced server-side but never re-verified by CI. A later admin could relax or delete them and nothing would flag it.

Scope:
- Add a read-only verification step that reads back the protection state via `gh api` and goes red (or writes HALT) when a required rule is missing or weakened. Decide where it lives (check-validate vs a dedicated job) and what it does on drift (halt vs report) — record the choice where it lands.
- Decide the credential: the protection/ruleset read endpoints need `admin:read` or `rules:read` scope, which the default `contents: read` GITHUB_TOKEN lacks. Note the required token/scopes in the handbook.

Acceptance: with a required protection deliberately relaxed, the check goes red; with all protections in place, green; the handbook documents the check next to the os-18135882 checklist and the token it needs.
