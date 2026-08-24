# Plan: Q5: dedicated machine identity for seed-state pushes (ref-push scoping, R10) (os-af5cb439)

Design R10 (docs/design-options.md:287): state-ref integrity is
push-access-deep — anyone who can push `seed-state` can bypass the shim (clear
a rejection list, forge lease fields, rewrite the log, forge an operator verb).
The named mitigation is "scope which principals hold state-ref push access
(Q5)". Status: deliberately deferred as the optional Q5 hardening in
os-18135882 (evidence, 2026-08-23): the ref is still contributor-writable. No
card/branch/PR tracked it (verified 2026-08-24).

Verified live (2026-08-24): `seed-state` has **ruleset 21223590**
(`no force-push, no delete`: rules `deletion` + `non_fast_forward`, branch
target) but **no branch protection** (GET `branches/seed-state/protection`
→ 404 "Branch not protected"). The ruleset API cannot express *who* may push
(principal scoping); the GitHub primitive for that is branch-protection **push
restrictions** (`allowed users/teams/apps`). Q5 therefore means adding that
protection on top of (and ultimately superseding) the ruleset, not a new
ruleset.

Decision recorded here (the one this card authorizes): a **fine-grained
personal access token** (repo-scoped, Contents: read & write + Metadata:
read-only on this repo) is the v1 machine identity. A GitHub App install is
the documented upgrade path when the operator wants app-scoped, per-repo
rotatable credentials (R10's "dedicated machine identity" reads cleaner then);
the enforcement shape is identical, so switching is a token swap, not a
re-protection.

## Steps

1. **Create the machine identity + secret** (operator action, handbook
   documents the commands):
   - Fine-grained PAT on the repo, `Contents: read & write`, `Metadata:
     read-only`; call it `seed-state-machine` (name in the token's notes, not
     the token itself).
   - Store it as repo secret `SEED_STATE_TOKEN`. (CI consumes it as a
     `GITHUB_TOKEN`-style secret; local sessions consume it from
     `SEED_STATE_TOKEN` in the environment.)

2. **Enforce push scoping via branch protection** on `seed-state`:
   `PUT /repos/{o}/{r}/branches/seed-state/protection` with
   `{"required_status_checks": {"strict": true}, "enforce_admins": true,
   "required_pull_request_reviews": null, "restrictions": {"allow_pushes":
   [<machine principal>], "include_allowances": true}}` — `enforce_admins`
   true is load-bearing (admins otherwise bypass push restrictions), and the
   principal is the PAT's user for v1 (the App installation for the upgrade
   path). `allow_force_pushes`/`allow_deletions` are implicitly false (not in
   the payload) so the rule semantics of 21223590 are preserved. Then
   **delete ruleset 21223590** (its two rules are subsumed; a redundant
   no-force-push ruleset plus the protection would double-report and drift).
   Record the exact `gh api` payloads in the handbook step so a fresh
   instantiation can replay them.
   - `main` protection and the tag rulesets are untouched.

3. **Authenticate the state-ref pushes as the machine identity**:
   - **CI**: `seed-maintenance.yml` — the engine shells out to system git for
     the state-ref push (`internal/gitx`), so the step exports an askpass for
     the push only: `GIT_ASKPASS` (or `GIT_CONFIG_COUNT`-style config)
     supplying `Authorization: Bearer ${{ secrets.SEED_STATE_TOKEN }}` for
     pushes to the state remote; the secret is never logged. Reap/close/anchor
     then push as the machine identity, and the protection's allowed-principal
     list admits them.
   - **Local sessions**: the operator adds
     `git -c http.extraHeader="Authorization: Bearer $SEED_STATE_TOKEN"` to
     their state-ref push (handbook documents the one-liner and a per-remote
     credential-store variant). **Engine affordance (small, engine repo)**:
     the state-ref push path honors an existing `SEED_STATE_TOKEN` env var
     when set (same askpass mechanism as CI, no new flag), so local and CI
     share one code path. That is the only engine change; it lands in
     open-seed-engine and the template's pin bump is a follow-up line in this
     same plan's close (record it, do not force a release for it — same
     discipline as os-a353ef17 step 6).

4. **Docs**:
   - `docs/handbook.md` §1 "Apply server-side protections": item 4 moves from
     "Hardening option (Q5)" to a numbered, replayable step: create identity →
     set secret → `PUT` protection (exact payload) → `DELETE`
     `/repos/{o}/{r}/rulesets/21223590` → set the secret in the workflow →
     verify (step 5). Add the degradation note: a repo that has not applied
     Q5 still works (the ref is contributor-writable); R10's audit claims
     (run-log integrity) are conditional on push scoping — say so explicitly.
   - Engine `seed init-github` text (engine repo, `cmd/seed/main.go:698`):
     item 4 ("Hardening option (Q5)") becomes "restrict `seed-state` pushes
     to the machine identity (fine-grained PAT or GitHub App) — see the
     template handbook §1; branch-protection push restrictions,
     `enforce_admins: true`, then delete the no-force-push ruleset". One
     paragraph; the handbook carries the commands.

5. **Verify on the live template repo** (evidence for the close):
   - With a plain contributor token (the session user, not the PAT user), a
     push to `seed-state` is **refused** (403 on push; protection
     `restrictions` denies). Capture the 403.
   - With the machine token, the same push succeeds; the
     `seed-maintenance` workflow (using `SEED_STATE_TOKEN`) runs a green tick
     (reap + anchor + a `seed task comment` round-trip through the shim).
   - `git ls-remote origin seed-state` still resolves for anonymous read.
   - Read back: `gh api .../branches/seed-state/protection` shows the
     restriction; `gh api .../rulesets` no longer lists 21223590.

## File Scope

- **this repo (template)**: `.github/workflows/seed-maintenance.yml`
  (askpass env for the state push; protected path), `docs/handbook.md` (§1).
  The `gh api` replay of the protection is **repo settings, not a repo file**
  (recorded as evidence + handbook commands, as in os-18135882).
- **open-seed-engine** (cross-repo, D7 no-PR close like os-488323ec):
  `internal/stateref` (or the push call site in `internal/gitx`) honors
  `SEED_STATE_TOKEN`; `cmd/seed/main.go` init-github text; README note.
- No `plans/**` in the task PR (D3). The plan file lands via this plan PR
  only.

## Acceptance Criteria

- `seed-state` push with a contributor credential not in the allowed
  principals is refused (403, captured); push as the machine identity
  succeeds.
- The `seed-maintenance` workflow (authenticating with `SEED_STATE_TOKEN`)
  runs a green tick after the change: reap + a shim verb + anchor.
- `gh api repos/{o}/{r}/rulesets` no longer lists the no-force-push ruleset;
  `branches/seed-state/protection` read-back shows `enforce_admins: true` and
  the machine principal in `restrictions.allow_pushes`.
- `docs/handbook.md` §1 is replayable from a fresh instantiation (commands
  present, not just described), and states the residual trust assumption:
  pre-Q5, run-log/claim integrity is conditional on push scoping (R10).
- Engine: `SEED_STATE_TOKEN` honored on the state-ref push; engine tests
  cover the askpass path with a fake remote; `go build ./...` + the touched
  packages' tests green.
- `make check` green in the template.

## Validation Commands

- `make check`
- `sh scripts/seed workflow validate --all`
- `cd /home/shaun/code/@shaunlmason/open-seed-engine && go build ./... && go test ./internal/stateref/... ./internal/gitx/... ./cmd/seed/...`
