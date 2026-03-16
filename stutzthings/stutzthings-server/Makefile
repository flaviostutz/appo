SHELL := /bin/sh

BINARY := stutzthings-server
DIST_DIR := dist
GOFLAGS ?=
MISE ?= mise
GO := $(MISE) exec -- go
GOLANGCI_LINT := $(MISE) exec -- golangci-lint
DOCKER ?= docker
DOCKER_COMPOSE := $(DOCKER) compose
RUN_IMAGES := eclipse-mosquitto:2 influxdb:3-core

.PHONY: all check-mise check-docker check-run-images install build lint lint-fix test test-unit test-integration coverage clean start run

all: build lint test

check-mise:
	@command -v $(MISE) >/dev/null 2>&1 || { echo "mise is required. Install Mise and run 'mise install' from the repository root."; exit 1; }

check-docker:
	@command -v $(DOCKER) >/dev/null 2>&1 || { echo "docker is required for 'make run'. Install Docker Desktop or use stutzthings/stutzthings-server/examples/local."; exit 1; }
	@$(DOCKER) info >/dev/null 2>&1 || { echo "docker daemon is not available. Start Docker Desktop or use stutzthings/stutzthings-server/examples/local."; exit 1; }

check-run-images: check-docker
	@for image in $(RUN_IMAGES); do \
		if ! $(DOCKER) image inspect "$$image" >/dev/null 2>&1; then \
			echo "docker image '$$image' is not available locally; attempting to pull it..."; \
			if ! $(DOCKER) pull "$$image" >/dev/null 2>&1; then \
				echo "failed to pull '$$image'. Network or registry access is blocked."; \
				echo "If Docker Hub access is restricted on this machine, use the host-based stack in stutzthings/stutzthings-server/examples/local."; \
				echo "See stutzthings/stutzthings-server/examples/local/README.md for prerequisites and commands."; \
				exit 1; \
			fi; \
		fi; \
	done

install: check-mise
	$(MISE) install
	$(GO) mod download

build: install
	mkdir -p $(DIST_DIR)
	$(GO) build $(GOFLAGS) -o $(DIST_DIR)/$(BINARY) .

lint: check-mise
	$(GOLANGCI_LINT) run ./...

lint-fix: check-mise
	$(GOLANGCI_LINT) run --fix ./...

test: test-unit test-integration

test-unit: check-mise
	$(GO) test ./...
	@packages="$$($(GO) list -f '{{if or (gt (len .TestGoFiles) 0) (gt (len .XTestGoFiles) 0)}}{{.ImportPath}}{{end}}' ./... | sed '/^$$/d')"; \
	if [ -n "$$packages" ]; then $(GO) test -cover $$packages; fi

test-integration: check-mise
	@packages="$$($(GO) list -tags=integration -f '{{if or (gt (len .TestGoFiles) 0) (gt (len .XTestGoFiles) 0)}}{{.ImportPath}}{{end}}' ./... | sed '/^$$/d')"; \
	if [ -n "$$packages" ]; then $(GO) test -cover -tags=integration $$packages; fi

coverage: check-mise
	$(GO) test ./...
	@packages="$$($(GO) list -f '{{if or (gt (len .TestGoFiles) 0) (gt (len .XTestGoFiles) 0)}}{{.ImportPath}}{{end}}' ./... | sed '/^$$/d')"; \
	if [ -n "$$packages" ]; then \
		$(GO) test -coverprofile=coverage.out $$packages; \
		$(GO) tool cover -func=coverage.out; \
	else \
		rm -f coverage.out; \
	fi

clean:
	rm -rf $(DIST_DIR) coverage.out

start: check-mise
	$(GO) run .

run: check-mise check-run-images
	$(DOCKER_COMPOSE) up -d
	$(GO) run .

