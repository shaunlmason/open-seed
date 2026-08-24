# Plan: CI: verify server-side seed-state protections (Phase 2 API-side check, unlanded) (os-7b6afa4d)

Build plan Phase 2 shipped `seed init-github` as a printed checklist only
(engine `cmd/seed/main.go:698`, "The engine cannot call the GitHub API;
apply these in repo settings"), deferring the API-side verification "to
Phase 5's workflows" (docs/build-plan.md:100). Phase 5 closed
(docs/build-plan.md:184) and none of the five live workflows performs that
read-back: no `.github/workflows/*.yml` calls the GitHub API to confirm the
protections are still active. The protections applied by os-18135882 (main
branch protection; the `seed-state` ruleset; the `seed-anchor/*` and
release-tag rulesets) are enforced server-side but never re-verified: an
admin who later relaxes or deletes one is not flagged by anything. This
card lands that check.

Verified against the live repo (2026-08-24, gh api):

- `GET /repos/{o}/{r}/branches/main/protection` returns
  `required_status_checks.contexts == ["check","verify"]`,
  `allow_force_pushes.enabled == false`, `allow_deletions.enabled == false`,
  `enforce_admins.enabled == false`.
- `GET /repos/{o}/{r}/rulesets` lists the three rulesets (ids 21223590/91/92);
  `GET /repos/{o}/{r}/rulesets/{id}` returns `enforcement: "active"` plus
  `conditions.ref_name.include` (the ref patterns the ruleset matches:
  `refs/heads/seed-state`, `refs/tags/seed-anchor/*`, `refs/tags/v*` — there is
  no `target_id` field in the payload) and the per-target rule lists
  (`deletion`, `non_fast_forward`, `update` per the three rulesets).
- `required_conversation_resolution.enabled == true` and
  `required_pull_request_reviews.require_code_owner_reviews == true` on
  `main` today: both are load-bearing merge gates and both must be asserted
  (an existing `required_pull_request_reviews` object alone would still pass
  after an admin disables conversation resolution or code-owner review).
- **The default `GITHUB_TOKEN` gets 403 on the rulesets endpoints** on this
  repo (it holds `contents/pull-requests/checks/issues`, which grant
  `branches:read` but not ruleset read). The check therefore needs an
  injected fine-grained token with **Administration: read-only** on the
  repository: that is the fine-grained-PAT permission that exposes
  branch-protection and ruleset reads (`rules:read`/`admin:read` are
  permission names of the classic/OAuth family, not selectable fine-grained
  PAT permissions).

## Steps

1. **Author `scripts/check-protections.sh`** (POSIX sh, jq). Reads
   `GITHUB_REPOSITORY` (or `-repo o/r` override) and
   `GITHUB_API_BASE` (default `https://api.github.com`); token from
   `GH_TOKEN`. Assertions, each printed as `ok: <name>` / `FAIL: <name>:
   <detail>`:
    - **main**: `branches/main/protection` exists (404 = FAIL "no main
      branch protection"); `required_status_checks.contexts` contains
      `check` and `verify`; `allow_force_pushes.enabled == false`;
      `allow_deletions.enabled == false`;
      `required_conversation_resolution.enabled == true` (handbook
      `docs/handbook.md:24-27`);
      `required_pull_request_reviews.require_code_owner_reviews == true`
      (the checklist's "require a review" is a CODEOWNER review, and the
      field is what survives an admin weakening the other toggles);
      `enforce_admins.enabled == false` (checklist item 5: scheduled jobs
      push via their non-admin token).
    - **seed-state**: a ruleset exists with `target: "branch"`,
      `enforcement: "active"`, `conditions.ref_name.include` matching
      `refs/heads/seed-state` (the ruleset payload carries no `target_id`
      field; the ref patterns live in `conditions.ref_name.include`), and
      rules of type `deletion` and `non_fast_forward`.
    - **seed-anchor/\***: a tag-target ruleset whose
      `conditions.ref_name.include` matches the pattern
      `refs/tags/seed-anchor/*`, `enforcement: "active"`, with `deletion`,
      `non_fast_forward`, and `update` rules (create-only).
    - **v\*** (release tags, the §1 hardening): a tag-target ruleset whose
      `conditions.ref_name.include` matches `refs/tags/v*`, `active`, with
      the same three rule types.
   - A ruleset name alone is never sufficient: enforcement and rule types
     are asserted, so a renamed-but-weakened ruleset still fails.
   Exit codes: **0** all pass (or degraded, step 2); **3** any assertion
   failed (findings printed); **1** usage/API error other than the degraded
   cases in step 2.

2. **Define the degraded cases** (the check must not wall up a repo where it
   cannot see, mirroring the repo's engine-absent posture): the token gets
   **403** on `rulesets` (scope missing) or the **`seed-state` ref does not
   exist** (fresh instantiation, `seed init` not run): print a `WARNING:`
   naming the gap and exit 0. Every other read error is exit 1.

3. **Wire it into `seed-maintenance.yml`** as a step **before** "Reap
   expired leases", gated on `steps.gate.outputs.active == 'true'` (the
   check needs the state ref to exist to be meaningful). On exit 3 it
   writes the `HALT` marker to the state ref and aborts the job (stop
   before reaping: a drifted-protections job must not mutate state).
   `scripts/seed state lint --halt-on-fail --actor seed-maintenance`
   already knows how to write `HALT`, but the conformance step sits
   *after* reap and close, so the step itself performs the write — through
   an **isolated worktree rooted at `origin/seed-state`** (fetch,
   `git worktree add /tmp/seed-halt origin/seed-state`, write `HALT`, commit
   as actor `seed-maintenance`, `git push origin HEAD:refs/heads/seed-state`,
   remove the worktree): the commit's parent is the current state-ref head,
   so the push is a fast-forward to the protected ref, and the write goes
   through the same fetch→commit→push path the engine uses for every state
   mutation (one commit per verb, §7.2) rather than editing the job's
   default-branch checkout. This repo's pinned engine version has no
   `seed state halt` verb (the write lives in `state lint --halt-on-fail`
   only); if the engine later lands one, switch this step to it. On green
   or degraded the step prints the read-back summary (ruleset names +
   `conditions.ref_name.include` + enforcement, main-protection fields) to
   `$GITHUB_STEP_SUMMARY`.

    **Token**: add `SEED_GH_TOKEN: ${{ secrets.SEED_GH_TOKEN }}` to the job
    env, falling back to `github.token`. The check exports
    `GH_TOKEN="${SEED_GH_TOKEN:-$GITHUB_TOKEN}"` for its `gh api` calls.
    (Default token gets 403 on rulesets here, so the secret is the
    documented path; its absence degrades per step 2 rather than failing.)

4. **Handbook §1** (`docs/handbook.md` "Apply server-side protections"
   bullet, line 22): note that the protections are re-verified by
   `seed-maintenance` (step name), the token it reads
   (`SEED_GH_TOKEN`, fine-grained PAT with **Administration: read-only** on
   the repository), the degraded cases, and what a red check means: a `HALT`
   marker on `seed-state` blocks every mutating verb until an operator runs
   `seed state resume`; the fix is re-applying the lost rule per the
   checklist, not resuming over it.

## File Scope

- `.github/workflows/seed-maintenance.yml`: the one new step (plus the job
  env var). Protected path: the change needs owner review.
- `scripts/check-protections.sh`: new file. `scripts/**` is protected.
- `docs/handbook.md`: §1 bullet only.
- No `plans/**` in this task PR (D3 purity; the plan PR touches only
  `plans/os-7b6afa4d.md`). No engine change: the check is sh + jq against
  the GitHub REST API.

## Acceptance Criteria

- `seed-maintenance` runs the check before reaping; on the live template it
  prints per-protection `ok:` lines and the job summary carries the read
  back.
- A red case exists and is proven: with `GITHUB_API_BASE`/`-repo` pointed
  at a scratch private repo that has the `seed-state` ref but no rulesets
  (or one rule removed), the script exits 3 listing each missing
  protection; the workflow step then writes `HALT` and the job stops before
  reaping.
- The degraded cases are proven: a token without Administration: read-only
  (403 on the rulesets endpoint) or a repo with no `seed-state` ref
  produces `WARNING:` + exit 0, and `make check` stays green there.
- The main assertions fail (exit 3) when `required_conversation_resolution`
  or `require_code_owner_reviews` is disabled on a scratch repo, and pass
  on the live configuration.
- The HALT write is proven: the drift case leaves `HALT` on `seed-state`
  with a fast-forward parent, and a subsequent `seed task claim` (or any
  mutating verb) refuses until `seed state resume`.
- Handbook §1 names the check, the token and its permission, the degraded
  cases, and the HALT/resume loop.
- `make check` (incl. the workflow YAML parse) passes.

## Validation Commands

- `make check`
- `sh scripts/seed workflow validate --all`
- `sh -n scripts/check-protections.sh`
- `GITHUB_REPOSITORY=shaunlmason/open-seed sh scripts/check-protections.sh`
