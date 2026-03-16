SHELL := /bin/sh

PROJECT_DIR := $(abspath ../..)
EXAMPLES_DIR := $(CURDIR)
RUN_DIR := $(EXAMPLES_DIR)/.run
DATA_DIR := $(EXAMPLES_DIR)/.data
LOG_DIR := $(EXAMPLES_DIR)/.logs
DOCKER_COMPOSE := docker compose

MISE ?= mise
GO := $(MISE) exec -- go

MQTT_USERNAME ?= test
MQTT_PASSWORD ?= test
MQTT_PORT ?= 1883
INFLUXDB_HOST ?= 127.0.0.1
INFLUXDB_PORT ?= 8086
HTTP_PORT ?= 8080
GRAFANA_PORT ?= 3000
INFLUXDB_URL ?= http://$(INFLUXDB_HOST):$(INFLUXDB_PORT)
INFLUXDB_TOKEN ?= apiv3_dev_token_local_only
INFLUXDB_DATABASE ?= iot
HTTP_ADDR ?= :$(HTTP_PORT)
GRAFANA_URL ?= http://127.0.0.1:$(GRAFANA_PORT)
LOG_LEVEL ?= info

BRIDGE_CONFIG := $(EXAMPLES_DIR)/bridge.json
INFLUXDB_TOKEN_FILE := $(EXAMPLES_DIR)/influxdb/admin-token.json
COMPOSE_FILE := $(EXAMPLES_DIR)/docker-compose.yml
INFLUXDB_CONTAINER := influxdb

.PHONY: build lint test install check-tools prepare start start-infra start-mqtt start-influxdb create-database start-bridge start-grafana stop stop-infra stop-mqtt stop-influxdb stop-bridge stop-grafana status clean logs

build:
	@$(MAKE) -C $(PROJECT_DIR) build

lint:
	@$(MAKE) -C $(PROJECT_DIR) lint

test:
	@$(MAKE) -C $(PROJECT_DIR) test-unit

install:
	@cd $(PROJECT_DIR) && $(MISE) install

check-tools:
	@command -v $(MISE) >/dev/null 2>&1 || { echo "mise is required. Run 'mise install' from the repository root."; exit 1; }
	@command -v docker >/dev/null 2>&1 || { echo "docker is required on PATH."; exit 1; }
	@docker info >/dev/null 2>&1 || { echo "docker daemon is not available. Start Docker Desktop first."; exit 1; }
	@$(DOCKER_COMPOSE) version >/dev/null 2>&1 || { echo "docker compose is required."; exit 1; }
	@command -v lsof >/dev/null 2>&1 || { echo "lsof is required on PATH."; exit 1; }

prepare: check-tools install
	mkdir -p $(RUN_DIR) $(LOG_DIR) $(DATA_DIR)/influxdb3 $(DATA_DIR)/mosquitto $(DATA_DIR)/grafana/data
	chmod 600 $(INFLUXDB_TOKEN_FILE)

start: start-infra create-database start-bridge
	@echo "local stack started"
	@echo "health: http://127.0.0.1$(HTTP_ADDR)"
	@echo "grafana: $(GRAFANA_URL)"
	@echo "logs: $(LOG_DIR)"
	@echo "compose: $(COMPOSE_FILE)"

start-infra: prepare
	@cd $(EXAMPLES_DIR) && $(DOCKER_COMPOSE) up -d mosquitto influxdb grafana

start-mqtt: start-infra

start-influxdb: start-infra

create-database: start-influxdb
	@attempt=0; \
	while :; do \
		output="$$({ cd $(EXAMPLES_DIR) && $(DOCKER_COMPOSE) exec -T $(INFLUXDB_CONTAINER) sh -lc 'export INFLUXDB3_AUTH_TOKEN=$(INFLUXDB_TOKEN) && influxdb3 create database $(INFLUXDB_DATABASE) --host http://127.0.0.1:8086'; } 2>&1)"; \
		status=$$?; \
		case "$$output" in \
			*"already exists"*) echo "database $(INFLUXDB_DATABASE) already exists"; exit 0 ;; \
		esac; \
		if [ $$status -eq 0 ]; then \
			echo "database $(INFLUXDB_DATABASE) ready"; \
			exit 0; \
		fi; \
		attempt=$$((attempt + 1)); \
		if [ $$attempt -ge 30 ]; then \
			echo "failed to create database $(INFLUXDB_DATABASE)"; \
			echo "$$output"; \
			exit 1; \
		fi; \
		sleep 1; \
	done

start-bridge: build create-database
	@if [ -f "$(RUN_DIR)/stutzthings-server.pid" ] && kill -0 "$$(cat "$(RUN_DIR)/stutzthings-server.pid")" 2>/dev/null; then \
		echo "stutzthings-server already running"; \
		exit 0; \
	fi
	@cd $(PROJECT_DIR) && (exec env BRIDGE_CONFIG_PATH=$(BRIDGE_CONFIG) HTTP_ADDR=$(HTTP_ADDR) LOG_LEVEL=$(LOG_LEVEL) $(MISE) exec -- go run . > $(LOG_DIR)/stutzthings-server.log 2>&1) & echo $$! > $(RUN_DIR)/stutzthings-server.pid
	@sleep 2
	@kill -0 "$$(cat "$(RUN_DIR)/stutzthings-server.pid")" 2>/dev/null

start-grafana: start-infra

stop: stop-bridge stop-infra

stop-infra:
	@cd $(EXAMPLES_DIR) && $(DOCKER_COMPOSE) down --remove-orphans

stop-grafana:
	@$(MAKE) stop-infra

stop-bridge:
	@if [ -f "$(RUN_DIR)/stutzthings-server.pid" ]; then \
		pid="$$(cat "$(RUN_DIR)/stutzthings-server.pid")"; \
		kill "$$pid" 2>/dev/null || true; \
		for _ in 1 2 3 4 5; do kill -0 "$$pid" 2>/dev/null || break; sleep 1; done; \
		rm -f "$(RUN_DIR)/stutzthings-server.pid"; \
	fi
	@for pid in $$(lsof -tiTCP:$(HTTP_PORT) -sTCP:LISTEN 2>/dev/null); do kill "$$pid" 2>/dev/null || true; done

stop-influxdb:
	@$(MAKE) stop-infra

stop-mqtt:
	@$(MAKE) stop-infra

status:
	@cd $(EXAMPLES_DIR) && $(DOCKER_COMPOSE) ps || true
	@for service in stutzthings-server; do \
		pid_file="$(RUN_DIR)/$$service.pid"; \
		if [ -f "$$pid_file" ] && kill -0 "$$(cat "$$pid_file")" 2>/dev/null; then \
			echo "$$service: running ($$(cat "$$pid_file"))"; \
		else \
			echo "$$service: stopped"; \
		fi; \
	done

logs:
	@cd $(EXAMPLES_DIR) && $(DOCKER_COMPOSE) logs --tail=50 mosquitto influxdb grafana || true
	@echo "stutzthings-server: $(LOG_DIR)/stutzthings-server.log"

clean: stop
	rm -rf $(RUN_DIR) $(LOG_DIR) $(DATA_DIR)