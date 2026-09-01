---
id: os-9b3f3ef3
title: 'next: loop verbs — three post-merge defects on the remote path (Codex review of #173)'
state: ready
priority: P1
squad: core
created_at: "2026-09-01T06:12:59Z"
updated_at: "2026-09-01T07:14:04Z"
---

Three findings from the Codex review of #173, all landed on main because the PR merged before the review was worked. All three verified against source; two are on the remote optimistic-retry path and one is a panic. Plan-first (L2).

(1) P1 — DERIVED ARGUMENTS ARE NOT RE-DERIVED AFTER THE TIP REFRESHES. loop.commit derives the payload once from openLoopSession's view, then hands the finished payload string to remoteSession.pushDraft, which re-signs and re-runs admission per attempt against the REFRESHED store but never re-derives. If a second valid reservation lands between session open and push, admission still accepts closing the originally cited reservation, because the budget rule checks only that the citation exists, is valid and is unclosed — it never requires it to be the SOLE open one. So the command silently makes exactly the choice soleOpenReservation exists to refuse. Same exposure for the fence. This is the defect plans/os-7e197768.md D3 was written to prevent ("a fence read from a stale local copy would be wrong under exactly the contention that makes claiming online-only") — the session was shared, but the derivation still ran once, outside the retry. Fix direction: derive inside each optimistic attempt against the same refreshed store admission uses, which means pushDraft taking a payload FUNCTION of the refreshed store rather than a string.

(2) P2 — A REMOTE REFUSAL IS STAMPED FROM THE STALE VIEW. loopSession.refuse recomputes affordances from ls.ctx and calls stampTip(env, ls.ctx.Count). On the remote path the refusal was computed one or more positions later, and remoteFailureEnvelope ALREADY stamps the refreshed position correctly through the refusalAt wrapper — so refuse actively OVERWRITES a correct stamp with a stale one, and advertises affordances from before the race (it can list claim.taken as legal on a subject a rival just claimed). Worse than the review states: the fix is not only to propagate the refreshed context but to stop clobbering a position the refusal already established.

(3) P2 — A JSON null PACKET PANICS THE CLI. loopPacket unmarshals --packet into map[string]json.RawMessage. The valid JSON value null unmarshals with no error and leaves the map NIL; if --base or a usable --repo then supplies a range, `parts["base"] = b` panics with "assignment to entry in nil map" (reproduced: assignment to entry in nil map). Without --base/--repo it refuses cleanly, which is why the drills missed it. A malformed packet must produce the documented usage envelope, never terminate the CLI. Fix: verify the root value is an object before mutating, and drill null alongside the existing malformed-packet cases.

Scope guard: no surface change. The verbs, flags, envelope and journal contracts stay exactly as they are; this is three correctness fixes plus their drills. The drills should include a remote contention case that proves re-derivation (a rival reservation landing between session open and push must refuse naming both candidates, not silently close one).
