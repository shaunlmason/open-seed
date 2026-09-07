---
id: os-d82cd66c
title: 'the frontier''s status words drift: twenty stale claims in next/docs/progress.md, and nothing checks them'
state: backlog
priority: P2
squad: core
created_at: "2026-09-07T07:19:41Z"
---

Found by review on #364 and confirmed by a sweep: `next/docs/progress.md` carried twenty status claims and every one was stale. Sixteen item lines read **in review** for cards long since done and closed, one still called a merged PR a draft, three prose claims said the same of merged work, and one said a card still stood at the gate. The oldest had been wrong since Phase 10. #364 fixed all twenty by hand; this card is about why they went wrong and stayed wrong.

WHY IT MATTERS MORE THAN A DOC NIT. AGENTS.md makes this file the single resume point: "A fresh agent resumes from this file alone", and "never start new work while it misstates the frontier". Its own closing paragraph then says "If an open task PR is red or carries review feedback, drive it green first, nothing merges out of order." So sixteen false in-review lines do not merely read wrong: they route a fresh agent into finished work, ahead of the actual frontier, under a rule that tells it to prioritise exactly those lines. A drift of one entry is noise; a drift of twenty makes the resume point worse than no file.

WHERE THE STALENESS IS BORN, EXACTLY. `.github/workflows/seed-maintenance.yml` already closes merged task PRs on its hourly tick ("Close merged task PRs (accept + cascade after merge)"): it walks `review` cards, finds the merged `seed/<id>` PR, and calls `seed task close`. That step is the precise moment a `**in review**` line in the progress file becomes false, and the loop is already standing there holding both facts (the card just moved to done, and the PR number it merged as). Nothing looks at the doc.

WHAT THIS CARD OWES. A check that makes the truthfulness rule real instead of stated. Shape, as far as this card fixes it: read the status claims out of `next/docs/progress.md`, resolve each to its card on the seed-state ref, and report every line whose claim disagrees with the card's state and review evidence. The card record is the authority (state plus the server-attributed evidence URL its close recorded), never a git-log grep: this session's clone is shallow and absence in the log read as a missing merge, which is how a naive version of this check would produce confident nonsense.

THREE CONSTRAINTS THE PLAN MUST RESPECT, EACH ALREADY VISIBLE.

1. IT MUST NOT HALT. `seed state lint --halt-on-fail` writes the HALT marker, and that consequence is calibrated to state-ref integrity: a forged accept, a rewritten log, a card claiming done with no merged PR (design-options D7, R10). A stale sentence in a document is not a forged accept and must not stop the queue. This is report-grade or a soft gate on the PR that touches the file, and the plan should say which and why. Conflating the two would make the HALT marker mean less, which is a real cost paid for a doc lint.

2. IT IS REPOSITORY-SPECIFIC, NOT A TEMPLATE OR ENGINE FEATURE. No adopter has a `next/docs/progress.md`; that file is this repository's own Seed-implementation frontier. So the check belongs to this repo's loop, not to `scripts/seed` (which ships to adopters) and not to the shipped template. The plan picks the home and states the boundary it respects: the ground rules limit Seed's v1 integration points to the Makefile and the docs tree, and a check that reads a `next/**` doc while consulting the v1 state ref sits across that seam. Naming which side owns it is most of the design.

3. DETECTION AND CORRECTION ARE DIFFERENT ASKS. Flagging is cheap and safe. Auto-editing the prose is not: the file's entries carry a long parenthetical of substance after the status word, and two entries in it are legitimately not flat. os-29e2fef2 is a genuine mixed case (task PR merged, card still in review), and the Phase 13 record's aside reads "in review AT THIS RECORD'S WRITING", which is historical narrative that is true as written and that an eager rewriter would falsify. Any correcting variant must distinguish a live status claim from a record of a past moment. The plan may reasonably scope this card to detection and leave correction to a follow-up, and should say so rather than leave it implied.

WHAT WOULD MAKE IT WORTH LESS. If the check only runs on PRs that touch `next/docs/progress.md`, it catches nothing: the failure mode is a card merging and NOBODY touching the file. It has to run where the frontier moves, which points at the maintenance tick beside the close step, or at a gate every PR passes.

NOT THIS CARD. Rewriting the file's structure, or generating the frontier from the cards wholesale. The file's value is its prose, the parentheticals recording what each card actually did and why, which no generator produces; the status word is the only part that should have been derived and was not.
