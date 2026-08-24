# make check is the one fast backpressure command (§2.6): agents merge
# whatever passes, so keep it fast and deterministic: it is the term that
# multiplies at scale (R12). Wire your project's real lint/test/typecheck
# here when instantiating the template.

.PHONY: check validate smoke flavor-test

check: validate
	@echo "check: add your project's lint/test/typecheck here (keep it fast)"

validate:
	@sh scripts/validate.sh

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
