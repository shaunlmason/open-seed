# make check is the one fast backpressure command (§2.6): agents merge
# whatever passes, so keep it fast and deterministic — it is the term that
# multiplies at scale (R12). Wire your project's real lint/test/typecheck
# here when instantiating the template.

.PHONY: check validate

check: validate
	@echo "check: add your project's lint/test/typecheck here (keep it fast)"

validate:
	@sh scripts/validate.sh
