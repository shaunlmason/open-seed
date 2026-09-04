---
id: os-2e46aa2f
title: 'next: the Seed release workflow — checksummed, provenance-attested binaries, the distribution step''s precondition (III.P row 1''s residual)'
state: blocked
priority: P1
squad: core
blocked_on:
    - plan:328
created_at: "2026-09-04T21:29:06Z"
updated_at: "2026-09-04T23:58:04Z"
---

Build plan section 5 names the distribution step's preconditions: self-hosting held for a stated period, a released Seed binary with checksums and provenance (charter III.P row 1's one residual today, per the conformance table: the binary is built from source and is not yet a released artifact), and a README a team can adopt from. Cutting the release is the operator's act at the distribution step, but the workflow that cuts it is agent work and does not exist. This card adds .github/workflows/seed-release.yml: workflow_dispatch only (the CI-identity lint holds scheduled workflows to contents: read, and a release is a human act), taking the version as input, minting the tag seed/v<version> at HEAD in-runner so the tag and the released commit cannot disagree (the engine repository's discipline), building seed and seed-admit from next/ for linux, darwin and windows on amd64 and arm64 with the version stamped into internal/version, packaging archives, writing sha256 checksums, publishing a GitHub Release, and attesting build provenance with actions/attest-build-provenance. internal/version.Version becomes a var so the stamp can land. A drill holds the workflow to dispatch-only, to the tag namespace, and to the attestation step, beside TestTreeWorkflowsHaveNoScheduledWriters. The handbook's Install section names the release. No release is cut here; III.P row 1's residual closes when the operator cuts the first one at the distribution step. Tier: standard (a workflow on the protected surface, owner review regardless). Plan-first.
