.PHONY: all check-mise check-docker check-run-images check-mosquitto-plugin install build lint lint-fix lint-format lint-style lint-audit lint-vuln test test-unit test-integration coverage clean start run health

all: build lint test

install:
	mise install
	mise exec -- go mod download

build: install
	mkdir -p dist
	mise exec -- go build $${GOFLAGS:-} -o dist/stutzthings-server .

lint: lint-format
	# Style and correctness checks
	mise exec -- golangci-lint run ./...

	# Format checks
	@files="$$(find . -type f -name '*.go' -not -path './dist/*' | sort)"; \
	if [ -z "$$files" ]; then exit 0; fi; \
	output="$$(mise exec -- gofmt -l $$files)"; \
	if [ -n "$$output" ]; then \
		echo "gofmt found unformatted files:"; \
		echo "$$output"; \
		exit 1; \
	fi

	# Vulnerability checks
	mise exec -- go run ./cli/vulnaudit --fail-level "$${VULN_FAIL_LEVEL:-critical}" ./...

lint-fix:
	files="$$(find . -type f -name '*.go' -not -path './dist/*' | sort)"; \
	if [ -n "$$files" ]; then mise exec -- gofmt -w $$files; fi
	mise exec -- golangci-lint run --fix ./...

test:
	# Run unit tests
	@packages="$$(mise exec -- go list -f '{{if or (gt (len .TestGoFiles) 0) (gt (len .XTestGoFiles) 0)}}{{.ImportPath}}{{end}}' ./... | sed '/^$$/d')"; \
	if [ -n "$$packages" ]; then \
		mise exec -- go test -coverprofile=coverage.out $$packages >/dev/null 2>&1; \
	else \
		echo "no unit test packages found"; \
		rm -f coverage.out; \
	fi

	# Enforce unit test coverage threshold
	@threshold="$${UNIT_COVERAGE_THRESHOLD:-80}"; \
	total_cov="$$(mise exec -- go tool cover -func=coverage.out | awk '/^total:/ {gsub(/%/, "", $$3); print $$3}')"; \
	if [ -z "$$total_cov" ]; then \
		echo "failed to determine unit test coverage"; \
		exit 1; \
	fi; \
	if awk -v cov="$$total_cov" -v threshold="$$threshold" 'BEGIN { exit !(cov + 0 < threshold + 0) }'; then \
		mise exec -- go tool cover -func=coverage.out; \
		awk -v cov="$$total_cov" -v threshold="$$threshold" 'BEGIN { printf "unit test coverage %.1f%% is below threshold %.1f%%\n", cov + 0, threshold + 0; exit 1 }'; \
	fi

	# Run integration tests (if any)
	@packages="$$(mise exec -- go list -tags=integration -f '{{if or (gt (len .TestGoFiles) 0) (gt (len .XTestGoFiles) 0)}}{{.ImportPath}}{{end}}' ./... | sed '/^$$/d')"; \
	if [ -n "$$packages" ]; then mise exec -- go test -tags=integration $$packages; fi

clean:
	rm -rf dist coverage.out

run:
	mise exec -- go run .

dev:
	mkdir -p ./.docker
	@plugin=./.docker/libmosquitto_jwt_auth.so; \
	if [ ! -s "$$plugin" ] || ! file "$$plugin" | grep -q 'ELF 64-bit LSB shared object, x86-64'; then \
		rm -f "$$plugin"; \
		curl -k -L -o "$$plugin" https://github.com/wiomoc/mosquitto-jwt-auth/releases/latest/download/libmosquitto_jwt_auth.so; \
	fi; \
	if ! file "$$plugin" | grep -q 'ELF 64-bit LSB shared object, x86-64'; then \
		echo "expected ./.docker/libmosquitto_jwt_auth.so to be an x86-64 Linux ELF shared library"; \
		exit 1; \
	fi
# 	docker compose up -d
# 	make run

health:
	url="$${HEALTH_URL:-https://localhost:$${HTTP_PORT}/health}"; \
	curl --silent --show-error --fail "$$url" 
