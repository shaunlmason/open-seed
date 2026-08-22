# make check is the one fast backpressure command (§2.6): agents merge
# whatever passes, so keep it fast and deterministic — it is the term that
# multiplies at scale (R12). Wire your project's real lint/test/typecheck
# here when instantiating the template.

.PHONY: check validate smoke

check: validate
	@echo "check: add your project's lint/test/typecheck here (keep it fast)"

validate:
	@sh scripts/validate.sh

# End-to-end loop smoke in a temp instantiation (no model, no secrets).
smoke:
	@bash scripts/smoke-loop.sh
