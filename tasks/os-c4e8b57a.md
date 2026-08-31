---
id: os-c4e8b57a
title: 'next: harden remote-contention test fixture against the auto-gc cleanup race'
state: backlog
priority: P3
squad: core
created_at: "2026-08-31T09:20:03Z"
---

TestRemoteAppendExhaustsAtContention (next/cmd/seed) flaked in CI on 2026-08-31 (flavor-test job, run 33376438803, PR #138 pre-amendment head): the test body passed, then t.TempDir's RemoveAll cleanup raced git's detached auto-gc in the bare remote fixture (unlinkat .../remote.git/objects: directory not empty). Green on the immediate re-run; a latent race, not a regression. Fix: configure the bare-remote fixture at creation so no detached git process can outlive the test (gc.auto=0, gc.autoDetach=false, receive.autogc off; sweep remote_test.go fixtures for the same pattern). Trivial tier.
