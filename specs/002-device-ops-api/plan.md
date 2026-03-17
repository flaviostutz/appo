# Implementation Plan: Device Operations API and MCP Access

**Branch**: `002-device-ops-api` | **Date**: 2026-03-17 | **Spec**: `/Users/flaviostutz/Documents/development/flaviostutz/appo/specs/002-device-ops-api/spec.md`
**Input**: Feature specification from `/specs/002-device-ops-api/spec.md`

## Summary

Extend `stutzthings/stutzthings-server` from a bridge-plus-health binary into a single-process Go service that exposes device operations through REST and MCP while continuing to use a Mosquitto broker protected by `wiomoc/mosquitto-jwt-auth` as the shared telemetry and `/set` command transport. The implementation will introduce a shared operations service layer for current-state reads, history queries, desired-state command publication, registration, and canonical input validation; a JWT auth layer that validates plugin-compatible `sub`, `publ`, and `subs` claims; and thin REST/MCP adapters mounted into the existing `net/http` server while Mosquitto enforces the same JWT on MQTT connections.

## Technical Context

**Language/Version**: Go 1.25.0  
**Primary Dependencies**: Go `net/http`, `github.com/InfluxCommunity/influxdb3-go/v2`, `github.com/eclipse/paho.mqtt.golang`, `github.com/sirupsen/logrus`, `github.com/golang-jwt/jwt/v5`, `github.com/mark3labs/mcp-go`, Mosquitto with `wiomoc/mosquitto-jwt-auth`  
**Storage**: InfluxDB v3 measurement `device_attributes`; Mosquitto broker for telemetry and desired-state topics; file/env configuration via `bridge.json`, environment variables, and Mosquitto config files  
**Testing**: `go test ./...`, package unit tests, existing Testcontainers-based integration tests, new HTTP/MCP/MQTT interoperability tests, explicit MQTT username=`sub` rejection coverage, and REST/MCP validation-parity tests  
**Target Platform**: Single Go server and plugin-enabled Mosquitto running on Linux/macOS with local Docker-backed development infrastructure  
**Project Type**: Monorepo Go web service with embedded bridge runtime and external broker integration  
**Performance Goals**: Meet spec goals of p95 current-state reads under 2s, p95 history reads under 3s, and exact REST/MCP business parity  
**Constraints**: Preserve tenant isolation, keep files under 400 lines, use explicit `from`/`to` history bounds, use one JWT claim model across MQTT/REST/MCP, enforce MQTT username=`sub`, share a base64-encoded signing secret between the Go service and Mosquitto plugin, centralize device-operation validation so REST and MCP cannot drift, and avoid duplicate business logic across REST and MCP  
**Scale/Scope**: One service module, one plugin-enabled broker config, 6 external operations, one MCP endpoint, and time-bounded history access over time-series data already stored by the bridge

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Principle I — XDR-driven decisions**: PASS. Existing `_local` BDRs/ADR/EDR were reviewed before planning; this plan updates the local XDRs required for MQTT topic ACL claims, registration policy, dual-surface architecture, and local verification workflow.
- **Principle II — Monorepo structure**: PASS. Work stays inside the existing `stutzthings/stutzthings-server` Go module and preserves the module Makefile workflow.
- **Principle III — Quality standards**: PASS. The plan retains `make build`, `make lint`, and `make test`; design includes unit, integration, interoperability, and parity coverage for new auth and API behavior.
- **Principle IV — Coding best practices**: PASS. The design splits new logic into focused packages and transport files so no file must absorb unrelated responsibilities.
- **Principle V — CI/CD & versioning**: PASS. No workflow shape changes are required for planning; implementation will continue to fit the existing module release model.
- **Principle VI — XDR-first feature documentation**: PASS after Phase 1. This plan updates the local XDRs and records them in the XDR Sync table below.

## Project Structure

### Documentation (this feature)

```text
specs/002-device-ops-api/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── mcp-tools.md
│   └── rest-openapi.yaml
└── tasks.md
```

### Source Code (repository root)

```text
stutzthings/stutzthings-server/
├── main.go
├── main_test.go
├── bridge/
│   ├── bridge.go
│   ├── config.go
│   ├── influx.go
│   ├── mqtt.go
│   └── ...
├── auth/
│   ├── jwt.go
│   ├── middleware.go
│   ├── topic_acl.go
│   ├── topic_acl_test.go
│   └── jwt_test.go
├── operations/
│   ├── models.go
│   ├── validation.go
│   ├── service.go
│   ├── query_service.go
│   ├── command_service.go
│   ├── registration_service.go
│   └── influx_queries.go
├── api/
│   ├── state_handlers.go
│   ├── history_handlers.go
│   ├── command_handlers.go
│   ├── registration_handlers.go
│   ├── request_validation.go
│   └── response_types.go
├── mcpapi/
│   ├── server.go
│   ├── state_tools.go
│   ├── history_tools.go
│   ├── command_tools.go
│   └── registration_tools.go
├── docker-compose.yml
├── examples/local/
│   ├── docker-compose.yml
│   └── mosquitto/mosquitto.conf
└── cli/
	└── vulnaudit/
```

**Structure Decision**: Keep the existing single Go module and add thin adapter packages around a shared `operations` service. `auth` owns JWT validation and MQTT topic-filter matching, `operations` owns business rules, canonical device-operation validation, and Influx/MQTT coordination, `api` owns HTTP serialization and request binding, and `mcpapi` owns MCP tool registration and transport wiring. Mosquitto remains a separate broker process, but it enforces the same JWT contract through `wiomoc/mosquitto-jwt-auth`.

## Complexity Tracking

No constitution violations are required for this feature.

## XDR Sync

> **Principle VI obligation — fill during Phase 1**

| Type | File | Decision Summary | Status |
|------|------|-----------------|--------|
| BDR | `.xdrs/_local/bdrs/product/001-mqtt-topic-structure.md` | Reconfirm telemetry and `/set` topics, and define JWT ACL filters against the same topic structure | updated |
| BDR | `.xdrs/_local/bdrs/product/002-influxdb-device-attributes-data-model.md` | Reconfirm query-side use of `device_attributes` tags/typed fields and add feature 002 references | updated |
| BDR | `.xdrs/_local/bdrs/product/003-device-authorization-scope-grammar.md` | Define shared JWT claims `sub`, `publ`, and `subs` plus topic-filter authorization rules for MQTT, REST, and MCP | updated |
| BDR | `.xdrs/_local/bdrs/product/004-stateless-device-registration-policy.md` | Define protected registration and issuance of plugin-compatible JWTs narrowed to a new device instance | updated |
| ADR | `.xdrs/_local/adrs/architecture/002-device-operations-dual-surface-architecture.md` | Expose one shared operations service through REST and MCP while reusing the MQTT JWT claim model enforced by Mosquitto | updated |
| EDR | `.xdrs/_local/edrs/patterns/002-local-device-operations-development-workflow.md` | Standardize one-command local verification workflow for bridge + REST + MCP + plugin-enabled Mosquitto + shared JWT settings | updated |
