# ADR 0002: Template flavors: mechanism, and how a flavored repo tracks releases

- **Date:** 2026-08-24
- **Status:** accepted

Design §10 Q3 keeps the v1 core language-agnostic: the template's only contract
with a project's stack is `make check` plus the hooks, and opinionated stack
variants are **v2 template flavors, never the v1 core**. This ADR fixes the
mechanism for those flavors, decided before the first flavor was built
(`os-501a29c2`).

## Decision

A flavor is a directory in the core template, `flavors/<name>/`, applied once
into a fresh instantiation by `scripts/seed-flavor install <name>`, with the
install recorded in `.seed/flavor.lock` and reconciled later by
`scripts/seed-flavor upgrade`.

This is the **template channel**, which is the right channel under the §10 Q4
split that `plans/os-221f5929.md` states as "plugin carrying capabilities,
template carrying structure". A flavor is structure: a `Makefile` recipe,
compiler options, a dependency set. The same reading keeps seed workflow DAGs
on the template channel.

## Rejected: template variant repos

One repo per flavor. Each variant becomes a fork that must independently track
upstream. `.seed/template.lock` carries exactly one `repo` line, so a flavored
instantiation would track either the variant (and never see core releases) or
the core (and never see flavor updates). That breaks the §10 Q4 update surface,
and it multiplies the maintainer release checklist by the number of flavors.

## Rejected: an engine-side `seed init --flavor`

The engine is a separate Go repository shipping pinned release binaries (§7.5,
ADR 0001), pinned by `.seed/engine.lock`, which advances on its own release
train independent of `.seed/template.lock`. Flavor *content* there would
version against the engine rather than against the template it was installed
into, so `seed template upgrade` could never reason about it.

## How a flavored instantiation tracks template releases

`seed template upgrade` merges by **path**. That single fact drives the rest of
this ADR, and getting it wrong is how the first draft of the plan went wrong: it
claimed ordinary template merging kept installed flavors current. It does not.
If install copies `flavors/typescript/package.json` to `/package.json`, a later
release editing `flavors/typescript/package.json` updates only that path in the
consumer's repo. Upstream has never owned `/package.json`, so the file the
consumer's `make check` actually runs stays at the version it was installed at,
forever.

Two rules follow, and both are binding on flavor authors:

**Thin root, fat flavor dir.** The install step writes as little as possible
into paths upstream does not own, and what it does write is thin and stable.
Everything that churns — compiler options, lint rules, the check recipe's body
— lives under `flavors/<name>/`, where `template upgrade` merges it normally.
The TypeScript flavor's root `tsconfig.json` is a two-line `extends` of
`flavors/typescript/tsconfig.base.json`; its flavored `check` recipe delegates
to npm scripts rather than inlining tool flags. The root `Makefile` is
additionally a path upstream *does* own, so upstream changes to it three-way
merge, with conflicts against the flavored recipe surfacing as ordinary
markers.

**Recorded ownership metadata plus an explicit reapply verb.** Some residue
cannot be made thin: npm requires dependency versions in the root
`package.json`, and those churn. So `install` records what it wrote — flavor
name, the `.seed/template.lock` `version` at install time, and a SHA-256 per
written file — in `.seed/flavor.lock`, and `seed-flavor upgrade` reconciles
against that recorded base: clean overwrite where the destination is unchanged
since install, `git merge-file` with standard conflict markers where it has
diverged. Without the recorded base there is no honest merge to perform, only a
clobber.

## The limit, stated rather than papered over

Flavor reconciliation is a **separate, consumer-initiated step** from `seed
template upgrade`, not an automatic consequence of it. Keeping a flavored repo
current is two commands, in order:

    scripts/seed template upgrade     # brings new payload into flavors/<name>/
    scripts/seed-flavor upgrade       # applies it to the installed destinations

A consumer who runs the first and not the second keeps running the payload they
installed. The handbook and `flavors/README.md` say so plainly; nothing in this
mechanism should be read as promising automatic flavor updates.

## Keeping the flavor out of the core

§10 Q3 is only true if the core gate does not depend on the flavor machinery.
Two invariants enforce it, both offline and both meaningful on any branch
(`scripts/flavor-test.sh`, run from `scripts/validate.sh`):

- **Core-gate independence:** an unflavored instantiation's `make check` output
  is identical with and without `flavors/` and `scripts/seed-flavor` present.
- **Install confinement:** the set of paths `install` changes is exactly the
  manifest's declared destination set.

The flavor's own integration test needs a dependency install, so it hangs off
`make flavor-test` and is deliberately **not** wired into `check` or
`validate.sh`: §2.6 binds `make check` to be fast and deterministic, and a
flavored `check` runs `validate`, so wiring the instantiation test into
`validate.sh` would also re-enter itself without bound. `make smoke` has the
same separation for the same reasons.

ADRs in this directory are append-only (`merge=union` via `.gitattributes`);
never rewrite one: supersede it with a new ADR.
