# stutzthings module Makefile — IoT middleware (Python library + Docker services)

MODULES := \
	stutzthings-server

.PHONY: build lint test lint-fix $(MODULES)

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
