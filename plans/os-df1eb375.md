# Plan: surface the paperclip paused-agent setup step (os-df1eb375)

`os-2c0c474c` established, against a live server, that Paperclip's
checkout **dispatches work** rather than merely locking a card: assigning
an issue wakes the agent, a runtime-less agent's run fails
(`adapter_failed`), and `recovery.reconcile_stranded_assigned_issue`
moves the issue `in_progress -> blocked` within about ten seconds. Agents
a seed deployment owns must therefore be paused in Paperclip.

That is a **setup step wearing a semantics costume**. It currently lives
only under "Declared variances", which is where a reader goes to
understand why the backend behaves as it does, not where an operator goes
to make it work. An operator following the Connect checklist end to end
will produce a deployment whose very first `claim` dies within ten
seconds, and it will not look like a missing setup step: it will look
like a flaky backend. The fix is placement, not new material.

Review of this plan surfaced that the gap is wider than misplacement.
Connect has **no agent-creation step and no agent-id discovery step** at
all (its three steps are: self-host, API key + env, config + verify). So
there is no "after agent creation" to insert a pause after, and an
operator has no `<id>` to pause. Worse, `claim` resolves `--actor` against
the company agent roster, so a Connect-only deployment has no agents for
any actor to resolve to and **every claim refuses with exit 5** before the
pause requirement is even reachable. `os-2c0c474c` introduced that
resolution contract and did not extend Connect to match. This plan closes
both gaps together, since neither instruction is usable alone.

Scope stays documentation-only: no adapter change, no test change. The
behaviour is already correct and already exercised (`live-test.sh`
provisions one agent per actor, pauses each, and its quiescence gate fails
loudly if the pause did not take).

## Steps

1. **Add the missing agent-provisioning step to Connect first.** The
   pause step has no anchor without it. Add a numbered step, before
   `backend verify`, that creates one Paperclip agent per seed actor
   (named so `--actor` resolves by `name`/`urlKey`) or points at
   `PAPERCLIP_AGENT_MAP` for deployments whose actor names differ, and
   that shows how to **list agents and read their ids**, since every
   later instruction needs them. Note the server-side `role` enum, whose
   rejection is otherwise a silent fixture failure.
2. **Then add the pause as its own numbered Connect step**, immediately
   after provisioning and before `backend verify`, so the checklist reads
   in the order an operator performs it. The snippet must be a real,
   pasteable **`curl`** carrying the configured API URL, bearer token and
   JSON content type, not HTTP shorthand: `PATCH /api/agents/<id> {...}`
   pasted into a shell is `PATCH: command not found`. State the trap that
   cost the original investigation an experiment: `status` is **ignored
   on create**, so pausing is necessarily a second request.
3. **Say what goes wrong without it**, in one sentence, at the point of
   use: the first claim is moved to `blocked` within ~10s by the
   platform's recovery sweep and presents as a flaky backend rather than
   a configuration error. An operator who skips the step should be able
   to recognise the symptom from the checklist alone.
4. **Keep the mechanism where it belongs.** The Declared variances entry
   stays and keeps the causal explanation (checkout dispatches work;
   which sweep fires; how it was diagnosed from the platform activity
   log). Connect says *what to do*, variances say *why it is necessary*,
   and each cross-references the other rather than duplicating it, so
   the two cannot drift into disagreement.
5. **Point at the worked example rather than restating it.** Reference
   `live-test.sh` as the executable version of the same setup (provision
   agents, pause them, verify quiescence), so a reader who wants a
   working script has one that is kept honest by CI instead of a snippet
   in prose that nothing exercises.

## File Scope

- `.seed/backends/paperclip/README.md`

## Acceptance Criteria

- An operator following Connect end to end configures a deployment whose
  first claim **resolves an agent and then survives**, without having read
  Declared variances.
- Connect creates agents and shows how to read their ids: the pause step
  and the actor-resolution contract are both performable from the
  checklist alone.
- The pause snippet is a complete `curl` an operator can paste, with URL,
  auth header and content type.
- The pause step names the create-ignores-`status` trap explicitly.
- The symptom of skipping the step (claim moves to `blocked` in ~10s,
  presenting as flakiness) is stated at the point of use.
- Connect and Declared variances cross-reference rather than duplicate:
  the mechanism is explained in exactly one place.
- The snippet is accurate against the `paperclip_version` pinned in
  `backend.toml`, and points at `live-test.sh` as the executed example.
- No adapter, corpus, fake, or harness change; `make check` stays green
  and `sh .seed/backends/paperclip/test.sh` still passes.

## Validation Commands

Each assertion is **scoped to the Connect section** and matches text that
does not exist there today, so every one of them fails on the current
README and can only pass once the step is actually written. An unscoped
`grep -q "paused"` would have passed before any implementation (the word
already appears in the budget bullet and in Declared variances), which is
a validation command that certifies nothing.

- `sed -n '/^## Connect/,/^## State/p' .seed/backends/paperclip/README.md | grep -q paused`
- `sed -n '/^## Connect/,/^## State/p' .seed/backends/paperclip/README.md | grep -q curl`
- `sed -n '/^## Connect/,/^## State/p' .seed/backends/paperclip/README.md | grep -q agents`
- `sh .seed/backends/paperclip/test.sh`
- `sh scripts/validate.sh`
