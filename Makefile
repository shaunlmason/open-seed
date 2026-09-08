# This repository is retired: there is nothing here to build, lint or test.
# The successor is https://github.com/shaunlmason/open-seed-v2, whose own
# `make check` is the gate that matters. This target exists so that anything
# still invoking `make check` out of habit gets a clear answer rather than a
# missing-target error.

.PHONY: check

check:
	@echo "open-seed v1 is retired: nothing to check here."
	@echo "The successor is https://github.com/shaunlmason/open-seed-v2 (see README.md)."
