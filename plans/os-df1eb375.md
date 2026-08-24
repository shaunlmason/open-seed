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

Scope is deliberately narrow: no adapter change, no test change. The
behaviour is already correct and already exercised (`live-test.sh` pauses
each corpus agent after provisioning, and the quiescence gate fails loudly
if the pause did not take). Only the README misplaces the instruction.

## Steps

1. **Add the pause as a numbered Connect step.** Extend
   `.seed/backends/paperclip/README.md`'s Connect list with a step that
   pauses every agent the deployment owns, placed after agent creation
   and before the `backend verify` step, so the checklist reads in the
   order an operator actually performs it. Include a copy-pasteable
   snippet (`PATCH /api/agents/<id> {"status":"paused"}`) and state the
   trap that cost the original investigation an experiment: `status` is
   **ignored on create**, so pausing is necessarily a second request.
2. **Say what goes wrong without it**, in one sentence, at the point of
   use: the first claim is moved to `blocked` within ~10s by the
   platform's recovery sweep and presents as a flaky backend rather than
   a configuration error. An operator who skips the step should be able
   to recognise the symptom from the checklist alone.
3. **Keep the mechanism where it belongs.** The Declared variances entry
   stays and keeps the causal explanation (checkout dispatches work;
   which sweep fires; how it was diagnosed from the platform activity
   log). Connect says *what to do*, variances say *why it is necessary*,
   and each cross-references the other rather than duplicating it, so
   the two cannot drift into disagreement.
4. **Point at the worked example rather than restating it.** Reference
   `live-test.sh` as the executable version of the same setup (provision
   agents, pause them, verify quiescence), so a reader who wants a
   working script has one that is kept honest by CI instead of a snippet
   in prose that nothing exercises.

## File Scope

- `.seed/backends/paperclip/README.md`

## Acceptance Criteria

- An operator following Connect end to end configures a deployment whose
  first claim survives, without having read Declared variances.
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

- `grep -q "paused" .seed/backends/paperclip/README.md`
- `sh .seed/backends/paperclip/test.sh`
- `sh scripts/validate.sh`
