# TypeScript flavor

Pre-wires lint, test and typecheck into `make check`, the one contract the
language-agnostic core has with your stack (design §10 Q3).

    scripts/seed-flavor install typescript
    npm install          # this script never runs a package manager

## What it wires

| Verb | Tool | Why |
|---|---|---|
| `typecheck` | `tsc --noEmit` | The only one of the three with no built-in equivalent |
| `lint` | `biome lint src` | One dependency, one binary; ESLint's plugin graph is not worth the install time in a gate that runs on every merge |
| `test` | `node --test` | Node strips types natively (>= 22.18), so the test runner costs **zero** dependencies |

Two devDependencies total. `make check` is the term that multiplies at scale
(R12), so the flavor spends its dependency budget only where a built-in will
not do.

## Layout, and why it is split this way

`seed template upgrade` merges by path, so anything this flavor copies to a
root destination stops receiving updates on its own. Everything that churns
therefore stays in this directory:

| Root (thin, stable) | Here (fat, churns) |
|---|---|
| `tsconfig.json` — `extends` + `include` | `tsconfig.base.json` — every compiler option |
| `Makefile` — recipes delegate to npm scripts | — |
| `package.json` — genuinely thick: npm requires deps at the root | — |

Compiler-option changes reach you through ordinary template merging with no
reapply at all. `package.json` is the one file that needs reconciling, which is
what `seed-flavor upgrade` is for.

## Keeping it current

Two separate, consumer-initiated commands, in order:

    scripts/seed template upgrade     # brings new payload into flavors/typescript/
    scripts/seed-flavor upgrade       # applies it to the installed destinations

Run the first without the second and you keep running the payload you
installed. `scripts/seed-flavor status` shows which destinations have diverged.
See `decisions/0002-template-flavors.md` for why it is two commands.
