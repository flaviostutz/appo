SHELL := /bin/sh

BINARY := stutzthings-server
DIST_DIR := dist
GOFLAGS ?=
MISE ?= mise
GO := $(MISE) exec -- go
GOLANGCI_LINT := $(MISE) exec -- golangci-lint
GOFMT := $(MISE) exec -- gofmt
VULN_FAIL_LEVEL ?= critical
GO_SOURCE_FILES = $(shell find . -type f -name '*.go' -not -path './dist/*' | sort)
GO_TEST_PACKAGES = $(shell $(GO) list -f '{{if or (gt (len .TestGoFiles) 0) (gt (len .XTestGoFiles) 0)}}{{.ImportPath}}{{end}}' ./... | sed '/^$$/d')
GO_INTEGRATION_TEST_PACKAGES = $(shell $(GO) list -tags=integration -f '{{if or (gt (len .TestGoFiles) 0) (gt (len .XTestGoFiles) 0)}}{{.ImportPath}}{{end}}' ./... | sed '/^$$/d')
UNIT_COVERAGE_THRESHOLD ?= 80
DOCKER ?= docker
DOCKER_COMPOSE := $(DOCKER) compose
RUN_IMAGES := eclipse-mosquitto:2 influxdb:3-core
HTTP_ADDR ?= :8080
HEALTH_URL ?=
JWT_SIGNING_SECRET_BASE64 ?= ZGV2X2p3dF9zZWNyZXRfbG9jYWxfb25seV9wbGVhc2VfY2hhbmdl
JWT_ISSUER ?= stutzthings-server-dev
JWT_TOKEN_TTL ?= 24h
MOSQUITTO_JWT_AUTH_SO ?=
RUN_SERVER := env HTTP_ADDR=$(HTTP_ADDR) JWT_SIGNING_SECRET_BASE64=$(JWT_SIGNING_SECRET_BASE64) JWT_ISSUER=$(JWT_ISSUER) JWT_TOKEN_TTL=$(JWT_TOKEN_TTL) $(GO) run .

.PHONY: all check-mise check-docker check-run-images check-mosquitto-plugin install build lint lint-fix lint-format lint-style lint-audit lint-vuln test test-unit test-integration coverage clean start run health

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

check-mosquitto-plugin:
	@if [ -z "$(MOSQUITTO_JWT_AUTH_SO)" ]; then \
		echo "MOSQUITTO_JWT_AUTH_SO must point to libmosquitto_jwt_auth.so for 'make run'."; \
		echo "Build or download the plugin, export MOSQUITTO_JWT_AUTH_SO, then retry."; \
		exit 1; \
	fi
	@if [ ! -f "$(MOSQUITTO_JWT_AUTH_SO)" ]; then \
		echo "MOSQUITTO_JWT_AUTH_SO does not exist: $(MOSQUITTO_JWT_AUTH_SO)"; \
		exit 1; \
	fi

install: check-mise
	$(MISE) install
	$(GO) mod download

build: install
	mkdir -p $(DIST_DIR)
	$(GO) build $(GOFLAGS) -o $(DIST_DIR)/$(BINARY) .

lint: check-mise lint-format
	$(GOLANGCI_LINT) run ./...
	$(GO) run ./cli/vulnaudit --fail-level $(VULN_FAIL_LEVEL) ./...

	@files="$(GO_SOURCE_FILES)"; \
	if [ -z "$$files" ]; then exit 0; fi; \
	output="$$($(GOFMT) -l $$files)"; \
	if [ -n "$$output" ]; then \
		echo "gofmt found unformatted files:"; \
		echo "$$output"; \
		exit 1; \
	fi

lint-fix: check-mise
	files="$(GO_SOURCE_FILES)"; \
	if [ -n "$$files" ]; then $(GOFMT) -w $$files; fi
	$(GOLANGCI_LINT) run --fix ./...

test:
	# Run unit tests
	@packages="$(GO_TEST_PACKAGES)"; \
	if [ -n "$$packages" ]; then \
		$(GO) test -coverprofile=coverage.out $$packages >/dev/null 2>&1; \
	else \
		echo "no unit test packages found"; \
		rm -f coverage.out; \
	fi

	# Enforce unit test coverage threshold
	@total_cov="$$($(GO) tool cover -func=coverage.out | awk '/^total:/ {gsub(/%/, "", $$3); print $$3}')"; \
	if [ -z "$$total_cov" ]; then \
		echo "failed to determine unit test coverage"; \
		exit 1; \
	fi; \
	if awk -v cov="$$total_cov" -v threshold="$(UNIT_COVERAGE_THRESHOLD)" 'BEGIN { exit !(cov + 0 < threshold + 0) }'; then \
		$(GO) tool cover -func=coverage.out; \
		awk -v cov="$$total_cov" -v threshold="$(UNIT_COVERAGE_THRESHOLD)" 'BEGIN { printf "unit test coverage %.1f%% is below threshold %.1f%%\n", cov + 0, threshold + 0; exit 1 }'; \
	fi

	# Run integration tests (if any)
	@packages="$(GO_INTEGRATION_TEST_PACKAGES)"; \
	if [ -n "$$packages" ]; then $(GO) test -tags=integration $$packages; fi

clean:
	rm -rf $(DIST_DIR) coverage.out


run: check-mise check-run-images check-mosquitto-plugin
	$(DOCKER_COMPOSE) up -d
	$(RUN_SERVER)

health:
	@command -v curl >/dev/null 2>&1 || { echo "curl is required for 'make health'."; exit 1; }
	@addr="$(HTTP_ADDR)"; \
	url="$(HEALTH_URL)"; \
	if [ -z "$$url" ]; then \
		case "$$addr" in \
			http://*|https://*) url="$$addr" ;; \
			:*) url="http://127.0.0.1$$addr" ;; \
			*) url="http://$$addr" ;; \
		esac; \
		url="$$url/health"; \
	fi; \
	echo "checking $$url"; \
	curl --silent --show-error --fail "$$url"

