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
  `GET /repos/{o}/{r}/rulesets/{id}` returns `enforcement: "active"` and
  per-target rule lists: `seed-state: no force-push, no delete` (branch:
  `deletion`, `non_fast_forward`), `seed-anchor tags: create-only` (tag:
  `deletion`, `non_fast_forward`, `update`), `release tags: immutable` (tag:
  same three).
- **The default `GITHUB_TOKEN` gets 403 on the rulesets endpoints** on this
  repo (it holds `contents/pull-requests/checks/issues`, which grant
  `branches:read` but not `rules:read`). The check therefore needs an
  injected token with `rules:read` (plus `admin:read` for
  `conversation_resolution`).

## Steps

1. **Author `scripts/check-protections.sh`** (POSIX sh, jq). Reads
   `GITHUB_REPOSITORY` (or `-repo o/r` override) and
   `GITHUB_API_BASE` (default `https://api.github.com`); token from
   `GH_TOKEN`. Assertions, each printed as `ok: <name>` / `FAIL: <name>:
   <detail>`:
   - **main**: `branches/main/protection` exists (404 = FAIL "no main
     branch protection"); `required_status_checks.contexts` contains
     `check` and `verify`; `allow_force_pushes.enabled == false`;
     `allow_deletions.enabled == false`; `required_pull_request_reviews`
     exists (a review is required); `enforce_admins.enabled == false`
     (checklist item 5: scheduled jobs push via their non-admin token).
   - **seed-state**: a ruleset exists with `target: "branch"`,
     `target_id: "seed-state"`, `enforcement: "active"`, and rules of type
     `deletion` and `non_fast_forward`. (Branch protection is the accepted
     fallback shape; the rule-types mapping is recorded in the script
     comments: `deletion` = block deletion, `non_fast_forward` = block
     force-push.)
   - **seed-anchor/\***: a tag-target ruleset whose `target_id` matches the
     pattern `seed-anchor/*` (case-insensitive), `enforcement: "active"`,
     with `deletion`, `non_fast_forward`, and `update` rules (create-only).
   - **v\*** (release tags, the §1 hardening): a tag-target ruleset on `v*`,
     `active`, with the same three rule types.
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
   writes the `HALT` marker to the state ref and aborts the job:
   `scripts/seed state lint --halt-on-fail --actor seed-maintenance` writes
   `HALT` when findings exist, but the conformance step sits *after* reap
   and close, so a drifted-protections job would already have mutated
   state; instead the step itself writes `HALT`
   (`git fetch origin seed-state; git checkout origin/seed-state -- .` into
   the worktree's `HALT` file, commit, push) and exits 3, stopping the job
   before reaping. On green or degraded it prints the read-back summary
   (rule ids + enforcement, main-protection fields) to
   `$GITHUB_STEP_SUMMARY`.

   **Token**: add `SEED_GH_TOKEN: ${{ secrets.SEED_GH_TOKEN }}` to the job
   env, falling back to `github.token`. The check exports
   `GH_TOKEN="${SEED_GH_TOKEN:-$GITHUB_TOKEN}"` for its `gh api` calls.
   (Default token gets 403 on rulesets here, so the secret is the
   documented path; its absence degrades per step 2 rather than failing.)

4. **Handbook §1** (`docs/handbook.md` "Apply server-side protections"
   bullet, line 22): note that the protections are re-verified by
   `seed-maintenance` (step name), the token it reads
   (`SEED_GH_TOKEN`, fine-grained, `rules:read` + `admin:read`), the
   degraded cases, and what a red check means: a `HALT` marker on
   `seed-state` blocks every mutating verb until an operator runs
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
- The degraded cases are proven: a token without `rules:read` (403) or a
  repo with no `seed-state` ref produces `WARNING:` + exit 0, and
  `make check` stays green there.
- Handbook §1 names the check, the token and its scopes, the degraded
  cases, and the HALT/resume loop.
- `make check` (incl. the workflow YAML parse) passes.

## Validation Commands

- `make check`
- `sh scripts/seed workflow validate --all`
- `bash -n scripts/check-protections.sh`
- `GITHUB_REPOSITORY=shaunlmason/open-seed sh scripts/check-protections.sh`
