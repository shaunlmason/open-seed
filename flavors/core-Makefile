# make check is the one fast backpressure command (§2.6): agents merge
# whatever passes, so keep it fast and deterministic: it is the term that
# multiplies at scale (R12). Wire your project's real lint/test/typecheck
# here when instantiating the template.

.PHONY: check validate smoke flavor-test check-next

check: validate check-next
	@echo "check: add your project's lint/test/typecheck here (keep it fast)"

validate:
	@sh scripts/validate.sh

# Seed (the successor, next/**): build + vet + gofmt + tests + coverage gate,
# the one v1 integration point docs/next-build-plan.md §0 names. Self-skips
# when next/ is absent (template instantiations, flavor tests) so `check`
# stays green off-tree; requires Go where next/ exists. Tool output is
# captured and shown only on failure: `check` output must be byte-stable
# run to run (the flavor-test core-gate-independence check diffs it), and
# go's per-test timings and toolchain-download notices are not.
check-next:
	@if [ ! -d next ]; then echo "check-next: next/ absent — skipping (template instantiation)"; exit 0; fi
	@command -v go >/dev/null 2>&1 || { echo "check-next: next/ exists but Go is not installed — install Go (next/go.mod pins the toolchain)"; exit 1; }
	@cd next && badfmt="$$(gofmt -l .)" && { test -z "$$badfmt" || { echo "check-next: gofmt failures:"; echo "$$badfmt"; exit 1; }; }
	@cd next && out="$$(go vet ./... 2>&1)" || { echo "check-next: go vet failed:"; echo "$$out"; exit 1; }
	@cd next && out="$$(go build ./... 2>&1)" || { echo "check-next: go build failed:"; echo "$$out"; exit 1; }
	@# The suite and the gate run through next/cmd/covergate, because the
	@# collection is lossy at a low rate and the rule that saves you -
	@# re-collect COLD once, then treat a second failure as real - is one
	@# an unattended agent must otherwise apply against its own instinct.
	@# cmd/go's mergeCoverProfile drops a package's profile fragment
	@# SILENTLY when the fragment file is missing or zero-length, with no
	@# error and `ok` still printed, so the merged total reads far below
	@# truth on a tree that is fine (card os-cafba959).
	@#
	@# The re-collection engages ONLY below the threshold, so a healthy
	@# tree never pays for it and it cannot false-alarm; and the second
	@# reading is cold, because go test caches a package's coverage
	@# contribution and a warm re-run replays the loss at the same number.
	@cd next && go run ./cmd/covergate -gate 90 -dir .
	@# The performance gate (plans/os-7508ab9e.md D6): the four metrics
	@# against the representative history, each held to the ceiling in
	@# next/perf/budgets.json, a miss re-measured cold once before it
	@# fails. Ceilings carry their provenance; raising one is a reviewed
	@# edit of that file, never a silent change.
	@cd next && go run ./cmd/perfgate -budgets perf/budgets.json -dir .

# End-to-end loop smoke in a temp instantiation (no model, no secrets).
smoke:
	@bash scripts/smoke-loop.sh

# Flavor integration test (ADR 0002): instantiates the template, installs the
# TypeScript flavor, and asserts `make check` is green then red on a broken
# fixture. Needs node + a registry, so it is deliberately NOT part of `check`:
# §2.6 binds the gate to be fast and deterministic, and a flavored `check`
# runs `validate`, which would otherwise re-enter this test. Self-skips
# (exit 0, explicit message) when the toolchain or registry is unavailable.
flavor-test:
	@sh scripts/flavor-test.sh
