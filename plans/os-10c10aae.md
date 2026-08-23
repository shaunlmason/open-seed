# Plan: multi-squad routing activation (os-10c10aae, §6)

Activate the normative routing semantics §6 defines (v1 ships one
`core` squad that satisfies every rule trivially): non-overlapping
scopes, unique priorities, explicit-else-priority-else-core card
routing, `ready --squad` filtering, cross-squad merges under the scope
owner's gate, and goal-ancestry validation that only wakes when a
second squad or a mission appears: a solo clone pays nothing.

## Steps

1. **Team-file validation grows the §6 rules** (engine,
   `internal/validate`): across all `.seed/teams/*.yaml`: squad
   `scope` globs may not overlap except via an explicit shared-scope
   entry naming one owning squad; `priority` ints unique; every squad
   has a human `lead`; `review: codeowners|agent`; `tier` ≤ the
   guardrails ceiling (already checked for core, now for all).
   **Core's bare-`**` fallback scope is exempt from the pairwise
   overlap rule**: `**` is the "matches what nothing else claims"
   catch-all §6 itself requires, so it necessarily intersects every
   scope: only two *specific* scopes overlapping (or a second squad
   also claiming bare `**`) is a violation. Violations fail
   `seed validate` (and CI).
2. **Card routing** (engine, `internal/task` + card schema): cards
   gain an optional `squad` field (create `--squad` exists in the
   port spec); resolution order is **explicit `squad:` → the
   matching squad by scope/backlog filter with the lowest priority →
   `core`, no card can be invisible. `get`/`list` surface the
   resolved squad.
3. **`ready --squad`** (shim-side default): the shim filters the
   ready queue by resolved squad; backends declaring the
   `ready_squad` capability filter server-side (the port spec already
   reserves it). `scripts/loop.sh --squad <name>` passes it through,
   so one loop per squad is the natural scaling unit.
4. **Cross-squad gate** (validation + docs, no new mechanism): the
   scope owner's gate already grounds in CODEOWNERS + tier: the
   validator now checks that every squad with `review: codeowners`
   has its lead present in CODEOWNERS for its scope paths, **as a
   warning, and only once multi-squad is active** (>1 squad): the
   shipped single-squad template, whose core.yaml carries a
   placeholder lead the instantiator replaces, must stay green out of
   the box (CODEOWNERS stays hand-owned control surface: the engine
   never edits it). The handbook documents that the owning squad's
   tier governs merges into its scope.
5. **Goal-ancestry activation rule**: the activation literal is
   `len(squads) > 1 || any squad declares a mission`, when it holds,
   cards missing a resolvable `parent` chain to a mission get a
   validation *warning* (report, not refusal: §6's alignment
   mitigation); solo/core-only repos see nothing. The shipped
   core.yaml currently sets a placeholder `mission:` that would trip
   this on every fresh clone: the task PR comments it out (the
   inline comment already tells instantiators to set their own), so
   activation stays a deliberate act.
6. **Template + docs**: `.seed/teams/` gains a commented example
   second-squad file (`platform.yaml.example`: inert, not parsed);
   handbook §5's squads rung and §6 routing get the activated story;
   `make smoke` gains a two-squad scenario (create a second team
   file in the scratch instantiation, assert routing + `ready
   --squad` + the overlap refusal).
7. **Tests** (engine, offline): overlap refusal incl. the shared-
   scope exception, priority uniqueness, routing resolution order
   (explicit / priority / core fallback), ready filtering shim-side,
   ancestry warning activation on/off, single-squad repos unaffected.
   Engine release + `seed upgrade` pin bump in the task PR.

## File Scope

- Engine repo: `internal/validate/`, `internal/task/`,
  `internal/card/`, `cmd/seed/`
- Template: `.seed/teams/platform.yaml.example` (new),
  `.seed/teams/core.yaml` (comment out the placeholder mission),
  `scripts/loop.sh` (--squad), `scripts/smoke-loop.sh`,
  `docs/handbook.md`, `.seed/engine.lock` (release pin via
  `seed upgrade`)

## Acceptance Criteria

- Overlapping scopes without a shared-scope owner, duplicate
  priorities, or a missing human lead fail validation; the shared-
  scope exception passes; core's bare-`**` fallback coexists with
  any second squad without tripping the overlap rule.
- Routing resolves explicit → lowest-priority match → core; no card
  is invisible; `ready --squad` and `loop.sh --squad` filter
  correctly.
- Goal-ancestry warnings appear only once a second squad or mission
  exists; a core-only repo's behavior is byte-identical to today.
- The smoke scenario proves two-squad routing end to end.

## Validation Commands

- `test -f .seed/teams/platform.yaml.example`
- `grep -q "squad" scripts/loop.sh`
- `make smoke`
- `sh scripts/validate.sh`
