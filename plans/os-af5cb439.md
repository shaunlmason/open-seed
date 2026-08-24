# Plan: Q5: dedicated machine identity for seed-state pushes (ref-push scoping, R10) (os-af5cb439)

Design R10 (docs/design-options.md:287): state-ref integrity is
push-access-deep — anyone who can push `seed-state` can bypass the shim (clear
a rejection list, forge lease fields, rewrite the log, forge an operator verb).
The named mitigation is "scope which principals hold state-ref push access
(Q5)". Status: deliberately deferred as the optional Q5 hardening in
os-18135882 (evidence, 2026-08-23): the ref is still contributor-writable. No
card/branch/PR tracked it (verified 2026-08-24).

Verified live on this repo (2026-08-24):

- `seed-state` has the branch ruleset "seed-state: no force-push, no delete"
  (rules `deletion` + `non_fast_forward`) and no branch protection (GET
  `branches/seed-state/protection` → 404 "Branch not protected").
- **Branch-protection push restrictions are unavailable here**: this repo is
  user-owned (`owner.type == "User"`), and GET
  `branches/main/protection/restrictions` returns 404 "Push restrictions not
  enabled" — that mechanism is an organization-repository feature.
- The primitive that DOES express principal scoping on this user-owned repo is
  a **branch ruleset carrying an `update` rule**: the update rule refuses ref
  updates from every principal NOT listed in the ruleset's `bypass_actors`,
  which makes `bypass_actors` the effective push allowlist. Verified by probe:
  creating such a ruleset here succeeds; a `restrictions` object sent on a
  branch ruleset is silently dropped on read-back (`bypass_actors` is the
  field that persists); and the literal `target: "push"` ruleset is not usable
  (its `ref_name` conditions are rejected 422; a repository-conditioned create
  returned 500).

So this plan implements the adopted Q5 default (docs/design-options.md §10 Q5:
"a push ruleset restricting `seed-state` to a dedicated machine identity
(fine-grained PAT or deploy key) plus squad leads") through the branch-ruleset
form the API actually supports on this repo: `update` + `non_fast_forward` +
`deletion` rules scoped to `refs/heads/seed-state`, with `bypass_actors` =
machine principal + squad lead. The scoping semantics are exactly the adopted
ones; only the ruleset target type differs, because the literal push-target
ruleset is not implemented by the API here. No reviewed design edit is needed.

Decision recorded: the machine identity is a **dedicated account** holding its
own fine-grained PAT (a GitHub App installation is the documented upgrade
path). A fine-grained PAT authenticates as the account that owns it, so the
token MUST be minted under the dedicated account: minting it under the
operator's own account and allow-listing "the PAT's user" would silently admit
the operator's ordinary credentials and achieve no scoping at all.

## Steps

1. **Provision the machine principal + secret** (operator action, handbook
   documents the commands):
   - Create/reuse a dedicated account (e.g. `open-seed-machine`); mint a
     fine-grained PAT under THAT account: repository access = this repo only,
     Contents: read & write, Metadata: read-only.
   - Store it as repo secret `SEED_STATE_TOKEN`.
   - Record the account's numeric id (`gh api users/<login> --jq .id`):
     `bypass_actors` entries need `actor_id`.

2. **Enforce push scoping via a branch ruleset** (never hardcoded ids;
   discover by stable name):
   - Resolve the legacy ruleset id by name: GET `/repos/{o}/{r}/rulesets`,
     match name "seed-state: no force-push, no delete".
   - Create the Q5 ruleset: POST `/repos/{o}/{r}/rulesets` with name
     "seed-state: machine + lead pushes only (Q5)", target "branch",
     enforcement "active", `conditions.ref_name.include`
     ["refs/heads/seed-state"], rules `[{"type":"update"},
     {"type":"non_fast_forward"},{"type":"deletion"}]`, bypass_actors
     `[{actor_id:<machine>, actor_type:"User", bypass_mode:"always"},
     {actor_id:<lead>, actor_type:"User", bypass_mode:"always"}]`.
   - Semantics, stated honestly in the handbook: `update` refuses every ref
     update from principals not in `bypass_actors` (that is the allowlist);
     but a bypass actor with `bypass_mode: always` is exempt from ALL rules of
     this ruleset — so force-push/deletion protection narrows to non-bypass
     principals once Q5 is applied. For the two privileged principals the
     integrity net remains the shim's non-fast-forward halt + anchor-tag
     ancestry checks (§7.2), the same trust position admins hold today
     (`enforce_admins.enabled == false` on main). If that residual is
     unacceptable, the follow-up is a second non-bypassable ruleset, not a
     weaker Q5.
   - Delete the legacy ruleset (the id resolved above); its two rules are
     carried by the Q5 ruleset. `main` protection and the tag rulesets are
     untouched.

3. **Authenticate the state-ref pushes as the machine identity**:
   - **CI**: `seed-maintenance.yml`'s checkout step sets
     `persist-credentials: false` — actions/checkout stores the workflow
     token as an http extraheader in `.git/config`, and leaving it in place
     would make the engine's git push authenticate as the workflow identity
     and be refused (correctly) once scoping is active. The state-mutating
     steps then export an askpass supplying
     `Authorization: Bearer ${{ secrets.SEED_STATE_TOKEN }}` for pushes to
     the state remote only; the secret is never echoed.
   - **Local sessions**: handbook documents exporting `SEED_STATE_TOKEN`;
     **engine affordance (small, open-seed-engine)**: the stateref push call
     site (`internal/gitx` `Repo.Push`, called from `internal/stateref`)
     honors `SEED_STATE_TOKEN` when set (same askpass mechanism as CI, no new
     flag), so local and CI share one code path. Pin bump recorded as a
     follow-up line in this card's close; do not force a release for it.

4. **Docs**:
   - `docs/handbook.md` §1 item 4 becomes the replayable Q5 step:
     provision principal → set secret → ruleset create payload (exact JSON,
     ids resolved via the user/ruleset name lookups) → delete legacy ruleset
     by resolved id → workflow secret → verify (step 5). Include both honesty
     notes: pre-Q5 the ref stays contributor-writable and R10 audit claims
     stay conditional; post-Q5 bypass principals remain exempt from the shape
     rules (see step 2).
   - Engine `seed init-github` text (`cmd/seed/main.go:698`): item 4 names
     the branch-ruleset + `bypass_actors` shape and points at handbook §1.
   - `memory/LEARNINGS.md`: the durable insight — principal scoping on
     user-owned repos lives in a ruleset's `update` rule + `bypass_actors`;
     branch-protection push restrictions are org-only; the literal
     push-target ruleset is unsupported by the API here.

5. **Verify on the live template repo** (evidence for the close):
   - Negative: a push to `seed-state` authenticated as a principal NOT in
     `bypass_actors` is refused; capture the rejection text.
   - Positive: the same push as the machine identity succeeds;
     `seed-maintenance` runs a green tick using `SEED_STATE_TOKEN` (reap +
     a shim verb round-trip + anchor).
   - Anonymous read unaffected: `git ls-remote origin seed-state` resolves.
   - Read-back: GET `/repos/{o}/{r}/rulesets` shows the Q5 ruleset (enforcement
     active; the three rules; `bypass_actors` listing both principals) and no
     legacy no-force-push ruleset.

## File Scope

- **this repo (template)**: `.github/workflows/seed-maintenance.yml`
  (`persist-credentials: false` + askpass env; protected path),
  `docs/handbook.md` (§1), `memory/LEARNINGS.md`. The ruleset changes are
  repo settings, not repo files: replayed from the handbook payloads and
  attached as evidence, as os-18135882 did. No `plans/**` in the task PR
  (D3 purity).
- **open-seed-engine** (cross-repo, closed like os-488323ec):
  `internal/gitx` / `internal/stateref` honor `SEED_STATE_TOKEN`;
  `cmd/seed/main.go` init-github text; README note.
- The plan file `plans/os-af5cb439.md` lands via this plan PR only.

## Acceptance Criteria

- A `seed-state` push authenticated as a non-bypass principal is refused
  (rejection captured); the same push as the machine identity succeeds.
- The `seed-maintenance` workflow runs a green tick authenticating with
  `SEED_STATE_TOKEN`: reap + a shim verb + anchor.
- Ruleset read-back shows the Q5 ruleset enforcement-active with
  `update`/`non_fast_forward`/`deletion` and `bypass_actors` =
  {machine principal, squad lead}; the legacy no-force-push ruleset is gone
  (matched by name, never by id).
- `docs/handbook.md` §1 is replayable end-to-end from a fresh instantiation
  (commands and exact payloads, including the actor-id and by-name ruleset
  lookups), and carries both residual-trust statements (pre-Q5 conditionality;
  bypass-principal exemption from the shape rules).
- Engine: `SEED_STATE_TOKEN` honored on the state-ref push; touched packages
  build and test green.
- `make check` green in the template.

## Validation Commands

The engine lives in a sibling checkout; its path comes from
`$SEED_ENGINE_SRC` (default: the engine checkout next to this repo) so the
commands run mechanically outside one author's machine:

- `make check`
- `sh scripts/seed workflow validate --all`
- `test -n "$SEED_ENGINE_SRC" && cd "$SEED_ENGINE_SRC" && go build ./... && go test ./internal/gitx/... ./internal/stateref/... ./cmd/seed/...`
