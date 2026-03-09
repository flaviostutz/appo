# Root Makefile — delegates targets to each module.
# Module directories — add entries here as new modules are created.
MODULES := \
	appo \
	stutzthings \
	shared \
	devices \
	deployments

.PHONY: build lint test lint-fix setup $(MODULES)

## Aggregate targets

build:
	@for dir in $(MODULES); do \
		$(MAKE) -C $$dir build || exit 1; \
	done

lint:
	@for dir in $(MODULES); do \
		$(MAKE) -C $$dir lint || exit 1; \
	done

test:
	@for dir in $(MODULES); do \
		$(MAKE) -C $$dir test || exit 1; \
	done

lint-fix:
	@for dir in $(MODULES); do \
		$(MAKE) -C $$dir lint-fix || exit 1; \
	done

setup:
	@echo "Install Mise: https://mise.jdx.dev/getting-started.html"
	@echo "Then run: mise install"
	@echo "See README.md for full setup instructions."
