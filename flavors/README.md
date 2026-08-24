# `flavors/`: opinionated stack variants (v2)

The v1 core is language-agnostic on purpose: its only contract with your
project's stack is `make check` plus the hooks (design §10 Q3). A **flavor**
is how a template goes beyond that contract without breaking it: a set of
files, shipped in this directory, applied once into a fresh instantiation.

    scripts/seed-flavor list
    scripts/seed-flavor install typescript
    scripts/seed-flavor status

Install is a **fresh-instantiation scaffold**, not an idempotent converger: it
refuses rather than overwrite work you already have. Keeping a flavored repo
current is two separate, consumer-initiated commands, in order:

    scripts/seed template upgrade     # brings new payload into flavors/<name>/
    scripts/seed-flavor upgrade       # applies it to the installed destinations

See `decisions/0002-template-flavors.md` for why it is two commands and not
one. Nothing here promises automatic flavor updates.

## The rule that keeps §10 Q3 true

**A flavor never changes the core template.** It only writes into the
consumer's instantiation. Three invariants enforce this
(`scripts/flavor-test.sh`), split by cost:

In every `make check`, offline:

- **Install confinement** — the set of paths `install` changes is exactly the
  manifest's declared destination set. A flavor that writes anything it did not
  declare fails the build.
- **Core-gate independence, statically** — `validate` stays the first
  prerequisite of the flavored `check`, and the core `validate`/`smoke`
  recipes survive verbatim.

In `make flavor-test` only, because it costs two `make check` runs:

- **Core-gate independence, observed** — an unflavored instantiation's `make
  check` output is identical with and without `flavors/` and
  `scripts/seed-flavor` present.

## Layout of `flavors/<name>/`

| File | What |
|---|---|
| `README.md` | What the flavor wires, the dependency-install command the consumer runs afterwards, and the two-step upgrade story |
| `manifest` | The payload-file → destination-path pairs `install` writes |
| everything else | The payload |

The `manifest` is line-oriented; `#` comments and blank lines are ignored.
Each line is `<payload-path> <destination-path>`. The two are resolved
differently, so do not mix them up: the **payload path is relative to this
flavor's own directory** (`flavors/<name>/`), and the **destination path is
relative to the repo root**:

    Makefile        Makefile
    package.json    package.json
    tsconfig.json   tsconfig.json

## Thin root, fat flavor dir (binding on flavor authors)

`seed template upgrade` merges by **path**. A destination outside `flavors/` is
a path upstream may not own, so payload updates do not reach it on their own —
that is exactly what `seed-flavor upgrade` and `.seed/flavor.lock` exist to
reconcile, and reconciliation is manual work for the consumer.

So: **keep every destination outside `flavors/` thin and stable, and keep the
churn inside the flavor directory.** The TypeScript flavor is the worked
example. Its root `tsconfig.json` is two lines that `extends` a
`tsconfig.base.json` living here, so compiler-option churn arrives through
ordinary template merging and needs no reapply at all. Its flavored `check`
recipe delegates to npm scripts rather than inlining tool flags, for the same
reason.

Some residue resists this and that is fine as long as it is deliberate: npm
requires dependency versions in the root `package.json`, so that file is
genuinely thick and is what `seed-flavor upgrade` is for.

## What a flavor may and may not touch

May: the root `Makefile` (its `check` recipe, extended — never the `validate`
or `smoke` targets), stack config the tools require at the root, and source
scaffolding.

May not: `.seed/**` other than `.seed/flavor.lock` (which `install` writes),
`.github/**`, `scripts/**`, `AGENTS.md`, `CLAUDE.md`, `CODEOWNERS`, or anything
else in `protected_paths` (`.seed/guardrails.yaml`). Those are the control
surface; a flavor is not a mechanism for editing it.
