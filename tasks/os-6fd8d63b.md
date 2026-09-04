---
id: os-6fd8d63b
title: 'CI: a pull request can be opened and silently receive no check runs at all'
state: backlog
priority: P1
squad: core
created_at: "2026-09-04T15:46:41Z"
updated_at: "2026-09-04T21:23:43Z"
---

Observed on #314 (os-f262585a) on 2026-09-04. The PR was opened at 14:49 and had ZERO check runs until 15:16, when an unrelated push finally triggered them. For 25 minutes it presented as a normal open PR with nothing red, and I reported it as ready for review on that basis.

THE EVIDENCE. check-validate.yml triggers on pull_request with no branch or path filter, so opening a PR should dispatch it. Listing runs for the branch returned total_count 0. That was not an artifact of the query: the same call against a control branch (seed/os-d63c7441-plan, a PR opened minutes later by the same actor through the same path) returned 12. Scanning the repo-wide run list confirmed the gap directly, with nothing between the push to main at 14:48 and the first os-d63c7441-plan run at 14:56, while #314 was created at 14:49. The branch had been force-pushed shortly before the PR was created, which is the only unusual thing about its history and may or may not be related.

WHY THIS IS P1. A PR with no checks is indistinguishable at a glance from a PR whose checks have not started, and mergeable_state reads "blocked" either way, which is also what an unreviewed PR reads. The failure mode is silent in the direction that matters: nothing is red, so nothing draws the eye, and the D3/D4.5 receipt and reviewer-identity gates that make a task PR trustworthy simply never ran. An agent driving a PR to green, or a human glancing at the checks tab, can conclude a PR is fine when it has been verified by nothing.

WHAT THIS CARD OWES. First, whether the cause is the force-push-then-open ordering, a dropped webhook, or something else, which needs the workflow run list and the delivery log for that window. Then a guard: the cheapest is a check that a task or plan PR carries a completed check-validate run for its head, so an absent run is itself a refusal rather than an absence. Note the guard cannot live in check-validate, because the failure is that check-validate did not run.

A workaround exists and is not a fix: any real push re-triggers dispatch. Never an empty commit.

## Comment cm-a9ff5d44 (seed-next-implementer, 2026-09-04T21:23:43Z)

Not taken this session, and why, so the next reader does not re-derive it. This is a v1 CI card on the protected workflow surface, outside the Seed scope the session worked (next/**), and its root cause needs the webhook delivery log for the window, which no agent here can read. The guard it asks for is nonetheless designable without the root cause, and this is the shape to plan: a scheduled workflow (hourly, like seed-maintenance) with pull-requests: read, checks: write and actions: read that lists open seed/* PRs, looks up check-validate workflow runs for each head SHA, and for a head with none creates a check run named check-guard on that SHA with conclusion failure and a summary saying no check-validate run exists for this head and that any real push re-triggers dispatch, never an empty commit. That makes an absent run a visible refusal rather than an absence, needs no change to check-validate.yml, and clears itself on the next push. Re-dispatching check-validate from the guard would need a workflow_dispatch trigger and a branch input on the gate itself, which is the riskier change and is not needed for the guard to be honest. Tier: standard (a new scheduled workflow on the protected surface); plan-first; owner review regardless.
