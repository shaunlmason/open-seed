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
	@# -p 1 serializes package test binaries: concurrent binaries under the
	@# subprocess-heavy drills can collide coverage counter files (same pid
	@# and second after heavy pid recycling), silently dropping one package
	@# from the merged profile and misreading coverage far below truth.
	@cd next && out="$$(go test -p 1 ./... -coverprofile=coverage.out -covermode=atomic -coverpkg=./internal/... 2>&1)" || { echo "check-next: go test failed:"; echo "$$out"; exit 1; }
	@cd next && go tool cover -func=coverage.out | awk '/^total:/ { cov=$$3; sub(/%/,"",cov); if (cov+0.0 < 90.0) { printf "check-next: coverage %s%% is below the 90%% gate (docs/next-build-plan.md §0)\n", cov; exit 1 } printf "check-next: gofmt/vet/build/test ok; coverage %s%% (gate 90%%)\n", cov }'

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
