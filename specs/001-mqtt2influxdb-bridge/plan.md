# Implementation Plan: MQTT to InfluxDB Bridge Daemon

**Branch**: `001-mqtt2influxdb-bridge` | **Date**: 2026-03-14 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-mqtt2influxdb-bridge/spec.md`

## Summary

Background goroutine embedded in `stutzthings-server` that subscribes to all MQTT topics via wildcard (`#`), parses 5-segment device attribute topics, decodes scalar or JSON payloads, and writes typed measurements to InfluxDB v3 in batches. Implements MQTT 3.1.1 with `github.com/eclipse/paho.mqtt.golang`, username/password auth, exponential-backoff reconnect, FIFO-evicting in-memory buffer, and startup readiness gate against InfluxDB.

## Technical Context

**Language/Version**: Go 1.22+  
**Language/Version**: Go 1.22+  
**Primary Dependencies**: `github.com/eclipse/paho.mqtt.golang` (v1.5.1+, MQTT 3.1.1 client with auto-reconnect), `github.com/InfluxCommunity/influxdb3-go/v2/influxdb3` (v2.13.0+, InfluxDB v3 client), `github.com/cenkalti/backoff/v4` (exponential backoff), `github.com/testcontainers/testcontainers-go` (integration tests)  
**Storage**: InfluxDB v3 (external, plain HTTP, API token auth)  
**Testing**: `go test`, `testify/assert`, integration test with real broker + InfluxDB containers  
**Target Platform**: Linux server (Raspberry Pi + cloud, deployed via Docker Compose)  
**Project Type**: background daemon component embedded in a server process  
**Performance Goals**: ≥500 msg/s sustained ingestion (SC-003); end-to-end latency < 2s (SC-001)  
**Constraints**: 100ms default flush interval; 10,000-message buffer ceiling (FIFO eviction); 3 retry attempts with exponential backoff; <400 lines per source file (Constitution IV)  
**Scale/Scope**: Single-process goroutine; multi-tenant (multiple account_ids); no horizontal scaling in this iteration

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Check | Status |
|-----------|-------|--------|
| I — XDR-Driven | All candidate XDRs from spec identified; will be created/updated in Phase 1 before code | PASS |
| II — Monorepo | Feature lives in `stutzthings/stutzthings-server/bridge/`; Makefile will expose `all`, `build`, `lint`, `test` targets | PASS |
| III — Quality | ≥80% coverage enforced; README Getting Started within first 20 lines; zero-warning `golangci-lint` | PASS |
| IV — Coding | Source files capped at 400 lines; goroutine-per-concern split enforces file-size constraint naturally; errors returned, not swallowed | PASS |
| V — CI/CD | Out of scope for this plan (workflow scaffolding deferred to infrastructure XDR) | N/A |
| VI — XDR-First | BDR-002 (InfluxDB data model), ADR-001 (bridge topology), EDR-001 (retry/reconnect policy) to be created in Phase 1 | PASS |

**Gate result**: PASS — proceed to Phase 0.

## Project Structure

### Documentation (this feature)

```text
specs/001-mqtt2influxdb-bridge/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0: library and architecture decisions
├── data-model.md        # Phase 1: entities, field layout, parsing rules
├── quickstart.md        # Phase 1: local dev and resilience testing guide
└── tasks.md             # Phase 2 output (/speckit.tasks — NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
stutzthings/stutzthings-server/
├── bridge/
│   ├── bridge.go       # Bridge type: Start(ctx), Stop() — public API surface
│   ├── config.go       # BridgeConfig, env var loading and validation
│   ├── mqtt.go         # paho.mqtt.golang client setup, reconnect, and message dispatch
│   ├── payload.go      # Topic validation, scalar/JSON payload parsing
│   ├── payload_test.go # Unit tests for payload parsing and topic validation
│   ├── buffer.go       # In-memory FIFO ring buffer with configurable ceiling
│   ├── buffer_test.go  # Unit tests for ring buffer eviction and flush triggers
│   └── influx.go       # InfluxDB v3 batch writer, retry loop, startup probe
│   └── bridge_integration_test.go # End-to-end: real MQTT broker + InfluxDB (testcontainers)
└── Makefile            # Targets: all, build, lint, test
```

**Structure Decision**: Single-project layout within `stutzthings/stutzthings-server/`. The `bridge/` package is self-contained with its own source files capped at 400 lines each (Constitution IV). Go test files are colocated beside the source files they cover, following agentkit-edr-002. No new top-level application folder is needed (Constitution II). Integration tests use testcontainers-go to spin up real broker and InfluxDB instances.

## Complexity Tracking

No complexity violations. All decisions align with constitution baselines:
- Single project, no new top-level folder
- No repository pattern (direct InfluxDB client write)
- No new unapproved technology (Go is the approved stutzthings-server language)

## Post-Phase-1 Constitution Check

| Principle | Verdict |
|-----------|---------|
| I — XDR-Driven | All 4 spec XDR candidates realized: BDR-001 updated, BDR-002 created, ADR-001 created, EDR-001 created |
| II — Monorepo | `stutzthings/stutzthings-server/bridge/` — correct location; Makefile targets defined |
| III — Quality | Coverage target ≥80%, README Getting Started within 20 lines, zero-warning lint |
| IV — Coding | 6 source files all fit within 400-line limit by design; errors returned not swallowed |
| V — CI/CD | Out of scope for this plan |
| VI — XDR-First | 3 new XDRs + 1 update committed to `.xdrs/_local/` |

**Gate result**: PASS — ready for `/speckit.tasks`.

## XDR Sync

| Type | File | Decision Summary | Status |
|------|------|-----------------|--------|
| BDR | `.xdrs/_local/bdrs/product/001-mqtt-topic-structure.md` | Updated `ts` field name (ms epoch) and payload timestamp format | updated |
| BDR | `.xdrs/_local/bdrs/product/002-influxdb-device-attributes-data-model.md` | Measurement `device_attributes`, 5-tag identity, 3 typed fields, ms timestamp | created |
| ADR | `.xdrs/_local/adrs/architecture/001-bridge-process-topology.md` | In-process goroutines in `bridge/` package; goroutine-per-concern with channels | created |
| EDR | `.xdrs/_local/edrs/patterns/001-bridge-reconnect-retry-policy.md` | paho.mqtt.golang reconnect with `OnConnect` re-subscription, exponential backoff probe, fixed-interval write retry, FIFO eviction | created |
