# make check is the one fast backpressure command (§2.6): agents merge
# whatever passes, so keep it fast and deterministic: it is the term that
# multiplies at scale (R12). Wire your project's real lint/test/typecheck
# here when instantiating the template.
#
# TypeScript flavor (flavors/typescript): check runs validate, then typecheck,
# lint and test, failing on the first failure. The recipes delegate to npm
# scripts on purpose: tool flags live in package.json and biome.json, which
# `seed template upgrade` merges, not in this file.

.PHONY: check validate smoke flavor-test typecheck lint test

check: validate typecheck lint test

validate:
	@sh scripts/validate.sh

typecheck:
	@npm run --silent typecheck

lint:
	@npm run --silent lint

test:
	@npm run --silent test

# End-to-end loop smoke in a temp instantiation (no model, no secrets).
smoke:
	@bash scripts/smoke-loop.sh

# Flavor integration test (ADR 0002): kept out of `check` for the same reason
# it is kept out upstream — it needs node and a registry, and `check` must
# stay fast and deterministic (§2.6).
flavor-test:
	@sh scripts/flavor-test.sh
