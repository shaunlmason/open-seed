#!/bin/sh
# Flavor mechanism tests (ADR 0002), in two halves.
#
#   --offline   filesystem-only: manifest structure, install refusals,
#               core-gate independence, install confinement. Fast and
#               deterministic, so scripts/validate.sh runs it in `make check`.
#               Runs NO `make check` of its own: recursion is impossible.
#
#   (default)   the full integration test: instantiate, install, npm install,
#               assert `make check` is green, then red on a broken fixture,
#               exercise reconcile, and run the BEHAVIOURAL half of core-gate
#               independence (two `make check` runs, compared) — which lives
#               here, not offline, because it costs a `make check`. Needs
#               node + a registry, so it hangs
#               off `make flavor-test` and is NEVER wired into validate.sh:
#               §2.6 binds `make check` to be fast and deterministic, and a
#               flavored `check` runs `validate`, which would re-enter this.
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
fail=0
say()  { printf 'flavor-test: %s\n' "$*"; }
bad()  { printf 'flavor-test: FAIL: %s\n' "$*"; fail=1; }
skip() { printf 'flavor-test: SKIP — %s\n' "$*"; exit 0; }

# Re-entry guard (belt-and-braces; the real protection is not being in
# validate.sh at all).
if [ "${SEED_FLAVOR_TEST:-}" = "1" ]; then
  say "already inside a flavor test; not re-entering"
  exit 0
fi
export SEED_FLAVOR_TEST=1

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

# Instantiate the template from the WORKING TREE, so an uncommitted flavor or
# installer is what gets tested. `git archive` of a write-tree would capture
# the *index*, silently testing stale code whenever something is unstaged —
# so enumerate tracked plus untracked-not-ignored files and tar those instead.
# (smoke-loop.sh archives HEAD on purpose: it tests committed state.)
# Returns non-zero (quietly) when this tree cannot be archived — not a git
# repo, or an index git will not write a tree from. Callers must skip rather
# than fail: `make check` runs this, and a consumer working from a source
# export or mid-conflict must still get a usable gate.
instantiate() {
  _dst=$1; mkdir -p "$_dst"
  _tar="$work/inst.tar"
  if (cd "$root" && git ls-files -z --cached --others --exclude-standard 2>/dev/null \
        | tar --null -T - -cf - 2>/dev/null) > "$_tar" 2>/dev/null && [ -s "$_tar" ]; then
    :
  elif (cd "$root" && git archive --format=tar HEAD) > "$_tar" 2>/dev/null && [ -s "$_tar" ]; then
    :
  else
    rm -f "$_tar"; return 1
  fi
  tar -xf "$_tar" -C "$_dst" || { rm -f "$_tar"; return 1; }
  rm -f "$_tar"
}

# Every path in the tree with its hash, for the confinement diff.
# Same Darwin fallback as scripts/seed:72 — this runs inside `make check`, so
# a GNU-only invocation would break the gate on a supported platform.
hash_files() {
  if command -v sha256sum >/dev/null 2>&1; then find . -type f -not -path './.git/*' -exec sha256sum {} +
  else find . -type f -not -path './.git/*' -exec shasum -a 256 {} +
  fi
}
snapshot() { (cd "$1" && hash_files | sort -k2); }

# ---------------------------------------------------------------- offline --

offline_tests() {
  if [ ! -d "$root/flavors" ]; then
    say "no flavors/ tree in this instantiation; nothing to check"
    return 0
  fi
  say "offline: manifest structure"
  for d in "$root"/flavors/*/; do
    [ -d "$d" ] || continue
    n=$(basename "$d")
    [ -f "$d/README.md" ] || bad "flavor $n has no README.md"
    [ -f "$d/manifest" ]  || { bad "flavor $n has no manifest"; continue; }
    while IFS= read -r line || [ -n "$line" ]; do
      case "$line" in ''|\#*) continue ;; esac
      src=$(printf '%s\n' "$line" | awk '{print $1}')
      dst=$(printf '%s\n' "$line" | awk '{print $2}')
      [ -n "$dst" ] || { bad "flavor $n: manifest line has no destination: $line"; continue; }
      [ -f "$d/$src" ] || bad "flavor $n: manifest names a missing payload file: $src"
      case "$dst" in
        /*|*..*) bad "flavor $n: destination escapes the repo: $dst" ;;
      esac
      # The contract in flavors/README.md: a flavor never writes control surface.
      case "$dst" in
        .seed/*|.github/*|scripts/*|AGENTS.md|CLAUDE.md|CODEOWNERS|seed.yaml|seed.lock)
          bad "flavor $n: destination is control surface, which a flavor may not write: $dst" ;;
      esac
    done < "$d/manifest"
  done

  say "offline: the core-Makefile reference matches the real core Makefile"
  # Only meaningful in an unflavored tree: after install, ./Makefile is the
  # flavored one by design.
  if [ ! -f "$root/.seed/flavor.lock" ]; then
    if [ ! -f "$root/flavors/core-Makefile" ]; then
      bad "flavors/core-Makefile is missing; install cannot verify what it replaces"
    elif ! cmp -s "$root/Makefile" "$root/flavors/core-Makefile"; then
      bad "flavors/core-Makefile has drifted from ./Makefile; refresh it (cp Makefile flavors/core-Makefile)"
    fi
  fi

  say "offline: seed-flavor list names the shipped flavors"
  out=$("$root/scripts/seed-flavor" list)
  echo "$out" | grep -qx typescript || bad "seed-flavor list did not name typescript (got: $out)"

  if ! instantiate "$work/probe"; then
    say "cannot archive this tree (not a git repo, or an unwritable index);"
    say "skipped the install-refusal, confinement and core-gate-recipe checks"
    say "that need a throwaway instantiation. Structure checks above still ran."
    return 0
  fi
  rm -rf "$work/probe"

  say "offline: install refusals"
  inst="$work/refuse"; instantiate "$inst"

  # Refusal: unknown flavor.
  rc=0; "$inst/scripts/seed-flavor" install nosuchflavor >/dev/null 2>&1 || rc=$?
  [ "$rc" != 0 ] || bad "install accepted an unknown flavor"

  # Refusal: a Makefile that is not the core template's.
  cp "$inst/Makefile" "$work/Makefile.core"
  echo "check:" > "$inst/Makefile"; echo "	@echo mine" >> "$inst/Makefile"
  rc=0; msg=$("$inst/scripts/seed-flavor" install typescript 2>&1) || rc=$?
  [ "$rc" != 0 ] || bad "install ran over a customized Makefile"
  echo "$msg" | grep -qi "refusing" || bad "customized-Makefile refusal did not say why: $msg"
  cp "$work/Makefile.core" "$inst/Makefile"

  # Refusal: a Makefile with the consumer's OWN targets appended, placeholder
  # left intact. install replaces the whole file, so a partial check (e.g. a
  # retained marker string) would let those targets be silently discarded.
  printf '\nrelease:\n\t@echo my release target\n' >> "$inst/Makefile"
  rc=0; msg=$("$inst/scripts/seed-flavor" install typescript 2>&1) || rc=$?
  [ "$rc" != 0 ] || bad "install replaced a Makefile carrying the consumer's own targets"
  echo "$msg" | grep -qi "byte-identical" || bad "appended-target refusal did not name the comparison: $msg"
  cp "$work/Makefile.core" "$inst/Makefile"

  # Refusal: a destination that already exists.
  mkdir -p "$inst/src"; echo "mine" > "$inst/src/index.ts"
  rc=0; msg=$("$inst/scripts/seed-flavor" install typescript 2>&1) || rc=$?
  [ "$rc" != 0 ] || bad "install overwrote an existing destination"
  echo "$msg" | grep -q "src/index.ts" || bad "existing-destination refusal did not name the file: $msg"
  rm -f "$inst/src/index.ts"

  # Refusal: already flavored, and it must point at `upgrade`.
  "$inst/scripts/seed-flavor" install typescript >/dev/null
  rc=0; msg=$("$inst/scripts/seed-flavor" install typescript 2>&1) || rc=$?
  [ "$rc" != 0 ] || bad "install ran twice without refusing"
  echo "$msg" | grep -q "seed-flavor upgrade" || bad "already-flavored refusal did not name the upgrade path: $msg"

  say "offline: install confinement (writes exactly its manifest)"
  conf="$work/confine"; instantiate "$conf"
  snapshot "$conf" > "$work/before"
  "$conf/scripts/seed-flavor" install typescript >/dev/null
  snapshot "$conf" > "$work/after"
  changed=$(diff "$work/before" "$work/after" | grep -E '^[<>]' | awk '{print $3}' | sed 's|^\./||' | sort -u)
  declared=$(awk '!/^#/ && NF {print $2}' "$root/flavors/typescript/manifest" | sort -u)
  # install also writes its own bookkeeping, which is not a manifest entry.
  undeclared=$(echo "$changed" | grep -v '^\.seed/flavor\.lock$' | grep -v '^\.seed/flavor-base/' | sort -u)
  if [ "$undeclared" != "$declared" ]; then
    bad "install touched paths outside its manifest"
    printf 'declared:\n%s\nchanged (minus bookkeeping):\n%s\n' "$declared" "$undeclared"
  fi

  say "offline: the flavored Makefile extends the core gate, never replaces it"
  # Cheap and static: the flavor may add to `check`, but `validate` must stay
  # its first prerequisite and the core recipes must survive verbatim. This is
  # the in-gate half of core-gate independence; the behavioural half runs in
  # the integration block, where a `make check` is affordable.
  core_mk="$root/Makefile"
  for d in "$root"/flavors/*/; do
    [ -d "$d" ] || continue
    n=$(basename "$d"); fmk="$d/Makefile"
    [ -f "$fmk" ] || continue
    grep -qE '^check:[[:space:]]+validate( |$)' "$fmk" \
      || bad "flavor $n: validate is not the first prerequisite of check"
    for target in validate smoke; do
      want=$(awk -v t="^$target:" '$0 ~ t {f=1;print;next} f && /^\t/ {print;next} f {exit}' "$core_mk")
      have=$(awk -v t="^$target:" '$0 ~ t {f=1;print;next} f && /^\t/ {print;next} f {exit}' "$fmk")
      [ "$want" = "$have" ] || bad "flavor $n: the core '$target' recipe is not preserved verbatim"
    done
  done
}

# ------------------------------------------------------------ integration --

integration_tests() {
  command -v node >/dev/null 2>&1 || skip "node is not on PATH (the TypeScript flavor's gate needs it)"
  command -v npm  >/dev/null 2>&1 || skip "npm is not on PATH (the TypeScript flavor's gate needs it)"
  npm ping >/dev/null 2>&1 || skip "the npm registry is unreachable; the flavor's dependency install cannot run offline"

  inst="$work/full"
  instantiate "$inst" || skip "this tree cannot be archived (not a git repo, or an unwritable index)"
  say "integration: install + npm install"
  (cd "$inst" && scripts/seed-flavor install typescript >/dev/null)
  (cd "$inst" && npm install --silent --no-audit --no-fund >/dev/null 2>&1) \
    || skip "npm install failed (no usable registry or cache)"

  say "integration: make check is green on a fresh flavored repo"
  if ! (cd "$inst" && make check >"$work/green.log" 2>&1); then
    bad "make check failed on a freshly flavored repo"; tail -20 "$work/green.log"
  fi

  say "integration: make check fails fast on a deliberately broken fixture"
  printf 'export const broken: number = "not a number";\n' > "$inst/src/broken.ts"
  if (cd "$inst" && make check >"$work/red.log" 2>&1); then
    bad "make check passed with a type error present"
  else
    grep -q "TS2322" "$work/red.log" || bad "make check failed but not on the typecheck error"
  fi
  rm -f "$inst/src/broken.ts"

  say "integration: core-gate independence (unflavored gate ignores the flavor tree)"
  a="$work/core-a"; instantiate "$a" || bad "could not instantiate for the core-gate comparison"
  b="$work/core-b"; instantiate "$b" || bad "could not instantiate for the core-gate comparison"
  rm -rf "$b/flavors" "$b/scripts/seed-flavor"
  (cd "$a" && make check >"$work/out-a" 2>&1) || true
  (cd "$b" && make check >"$work/out-b" 2>&1) || true
  # Normalise the per-run temp path, and drop the flavor block's own lines from
  # both sides: that block is by definition not the core gate, and it is the
  # only thing that legitimately differs when the flavor tree is gone.
  for f in "$work/out-a" "$work/out-b"; do
    sed -i "s|$a|ROOT|g; s|$b|ROOT|g" "$f"
    grep -v 'flavor-test:' "$f" > "$f.core" || true
  done
  if ! diff -q "$work/out-a.core" "$work/out-b.core" >/dev/null; then
    bad "the core gate's output changes when the flavor machinery is removed"
    diff "$work/out-a.core" "$work/out-b.core" | head -20
  fi

  say "integration: reconcile applies payload updates to an undiverged file"
  printf 'export const fromUpstream = 1;\n' >> "$inst/flavors/typescript/src/index.ts"
  (cd "$inst" && scripts/seed-flavor upgrade >/dev/null)
  grep -q "fromUpstream" "$inst/src/index.ts" \
    || bad "upgrade did not apply a payload change to an undiverged destination"

  say "integration: a merged local edit stays diverged and survives the NEXT upgrade"
  # Regression: recording the merged result as the baseline would make status
  # call this clean, and the next payload change would overwrite the edit.
  mrg="$work/merge"; instantiate "$mrg" || bad "could not instantiate for the merge-baseline test"
  (cd "$mrg" && scripts/seed-flavor install typescript >/dev/null)
  printf '\n// consumer edit, top of file\n' | cat - "$mrg/src/index.ts" > "$mrg/src/index.ts.new"
  mv "$mrg/src/index.ts.new" "$mrg/src/index.ts"
  printf 'export const wave1 = 1;\n' >> "$mrg/flavors/typescript/src/index.ts"
  (cd "$mrg" && scripts/seed-flavor upgrade >/dev/null 2>&1) || true
  grep -q "consumer edit" "$mrg/src/index.ts" || bad "first reconcile lost the consumer edit"
  (cd "$mrg" && scripts/seed-flavor status) | grep -q "diverged src/index.ts" \
    || bad "a file carrying a merged local edit is reported clean, not diverged"
  printf 'export const wave2 = 2;\n' >> "$mrg/flavors/typescript/src/index.ts"
  (cd "$mrg" && scripts/seed-flavor upgrade >/dev/null 2>&1) || true
  grep -q "consumer edit" "$mrg/src/index.ts" \
    || bad "the SECOND reconcile silently deleted the consumer edit"

  say "integration: a new payload destination that already exists is not overwritten"
  col="$work/collide"; instantiate "$col" || bad "could not instantiate for the collision test"
  (cd "$col" && scripts/seed-flavor install typescript >/dev/null)
  echo "mine, not the flavor's" > "$col/src/extra.ts"
  echo "src/extra.ts    src/extra.ts" >> "$col/flavors/typescript/manifest"
  echo "export const theirs = 1;" > "$col/flavors/typescript/src/extra.ts"
  rc=0; out=$( (cd "$col" && scripts/seed-flavor upgrade 2>&1) ) || rc=$?
  grep -q "mine, not the flavor" "$col/src/extra.ts" \
    || bad "upgrade overwrote a pre-existing file the flavor never installed"
  echo "$out" | grep -q "COLLISION" || bad "upgrade did not report the collision: $out"
  [ "$rc" != 0 ] || bad "upgrade exited 0 despite a collision"

  say "integration: reconcile conflicts (not clobbers) on a diverged file"
  printf 'export const mine = 2;\n' >> "$inst/src/index.ts"
  printf 'export const theirs = 3;\n' >> "$inst/flavors/typescript/src/index.ts"
  rc=0; (cd "$inst" && scripts/seed-flavor upgrade >/dev/null 2>&1) || rc=$?
  [ "$rc" = 4 ] || bad "upgrade on a diverged destination exited $rc, expected 4"
  grep -q '<<<<<<<' "$inst/src/index.ts" || bad "upgrade clobbered a diverged destination instead of conflicting"
  grep -q 'mine = 2' "$inst/src/index.ts" || bad "upgrade lost the consumer's local edit"
}

case "${1:-}" in
  --offline) offline_tests ;;
  *)         offline_tests; integration_tests ;;
esac

[ "$fail" -eq 0 ] && say "ok" || exit 1
