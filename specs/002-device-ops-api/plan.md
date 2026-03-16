# Implementation Plan: Device Operations API and MCP Access

**Branch**: `002-device-ops-api` | **Date**: 2026-03-17 | **Spec**: `/Users/flaviostutz/Documents/development/flaviostutz/appo/specs/002-device-ops-api/spec.md`
**Input**: Feature specification from `/specs/002-device-ops-api/spec.md`

## Summary

Extend `stutzthings/stutzthings-server` from a bridge-plus-health binary into a single-process Go service that exposes device operations through both REST and MCP while reusing the existing bridge's InfluxDB and MQTT integrations. The implementation will introduce a shared operations service layer for current-state reads, history queries, desired-state command publication, and stateless registration; a JWT auth layer that supports the clarified `i:` and `r:` scope families with wildcard matching; and thin REST/MCP adapters mounted into the existing `net/http` server.

## Technical Context

**Language/Version**: Go 1.25.0  
**Primary Dependencies**: Go `net/http`, `github.com/InfluxCommunity/influxdb3-go/v2`, `github.com/eclipse/paho.mqtt.golang`, `github.com/sirupsen/logrus`, `github.com/golang-jwt/jwt/v5`, `github.com/mark3labs/mcp-go`  
**Storage**: InfluxDB v3 measurement `device_attributes`; MQTT broker for desired-state publication; file/env configuration via `bridge.json` and environment variables  
**Testing**: `go test ./...`, package unit tests, existing Testcontainers-based integration tests, new HTTP/MCP contract tests  
**Target Platform**: Single Go server running on Linux/macOS with local Docker-backed development infrastructure  
**Project Type**: Monorepo Go web service with embedded bridge runtime  
**Performance Goals**: Meet spec goals of p95 current-state reads under 2s, p95 history reads under 3s, and exact REST/MCP business parity  
**Constraints**: Preserve tenant isolation, keep files under 400 lines, use explicit `from`/`to` history bounds, support partial whole-device reads, avoid duplicate business logic across REST and MCP  
**Scale/Scope**: One service module, 6 external operations, one MCP endpoint, and paginated history access over time-series data already stored by the bridge

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Principle I — XDR-driven decisions**: PASS. Existing `_local` BDRs/ADR/EDR were reviewed before planning; this plan creates the additional local XDRs required for auth scopes, registration policy, dual-surface architecture, and local verification workflow.
- **Principle II — Monorepo structure**: PASS. Work stays inside the existing `stutzthings/stutzthings-server` Go module and preserves the module Makefile workflow.
- **Principle III — Quality standards**: PASS. The plan retains `make build`, `make lint`, and `make test`; design includes unit, integration, and contract coverage for new auth and API behavior.
- **Principle IV — Coding best practices**: PASS. The design explicitly splits new logic into focused packages (`auth`, `operations`, `api`, `mcpapi`) so no file must absorb unrelated responsibilities.
- **Principle V — CI/CD & versioning**: PASS. No workflow shape changes are required for planning; implementation will continue to fit the existing module release model.
- **Principle VI — XDR-first feature documentation**: PASS after Phase 1. This plan generates/updates local XDRs and records them in the XDR Sync table below.

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
│   ├── scopes.go
│   └── scopes_test.go
├── operations/
│   ├── models.go
│   ├── service.go
│   ├── query_service.go
│   ├── command_service.go
│   ├── registration_service.go
│   └── influx_queries.go
├── api/
│   ├── handlers.go
│   ├── request_validation.go
│   └── response_types.go
├── mcpapi/
│   ├── server.go
│   └── tools.go
├── cli/
│   └── vulnaudit/
└── examples/local/
```

**Structure Decision**: Keep the existing single Go module and add thin adapter packages around a shared `operations` service. `auth` owns JWT parsing and scope matching, `operations` owns business rules and Influx/MQTT coordination, `api` owns HTTP serialization/validation, and `mcpapi` owns MCP tool registration and transport wiring.

## Complexity Tracking

No constitution violations are required for this feature.

## XDR Sync

> **Principle VI obligation — fill during Phase 1**

| Type | File | Decision Summary | Status |
|------|------|-----------------|--------|
| BDR | `.xdrs/_local/bdrs/product/001-mqtt-topic-structure.md` | Reconfirm `/set` topic contract used by desired-state publication and add feature 002 references | updated |
| BDR | `.xdrs/_local/bdrs/product/002-influxdb-device-attributes-data-model.md` | Reconfirm query-side use of `device_attributes` tags/typed fields and add feature 002 references | updated |
| BDR | `.xdrs/_local/bdrs/product/003-device-authorization-scope-grammar.md` | Define `i:` device-operation scopes, `r:` registration scopes, wildcard matching, and partial device snapshots | created |
| BDR | `.xdrs/_local/bdrs/product/004-stateless-device-registration-policy.md` | Define protected registration, unique identity issuance, and one `rws` credential per new device instance | created |
| ADR | `.xdrs/_local/adrs/architecture/002-device-operations-dual-surface-architecture.md` | Expose one shared operations service through REST and MCP in the existing process and HTTP server | created |
| EDR | `.xdrs/_local/edrs/patterns/002-local-device-operations-development-workflow.md` | Standardize one-command local verification workflow for bridge + REST + MCP + auth settings | created |
