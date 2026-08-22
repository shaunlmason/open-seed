# ADR 0001 — Engine repository name and release channel

- **Date:** 2026-08-22
- **Status:** accepted

The protocol engine (design decision 7.5) lives at
`shaunlmason/open-seed-engine`, public, MIT. Releases are driven by that
repo's `VERSION` file (or a manual `workflow_dispatch`): the release workflow
mints the semver tag at HEAD in-runner — tag and released commit cannot
disagree, and no contributor needs tag-push rights — then goreleaser publishes
six-target archives with `checksums.txt` and GitHub build-provenance
attestations. This template pins the engine in `.seed/engine.lock`
(version + per-target SHA-256), consumed by `scripts/seed` / `scripts/seed.ps1`.

ADRs in this directory are append-only (`merge=union` via .gitattributes);
never rewrite one — supersede it with a new ADR.
