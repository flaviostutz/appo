# Tasks: MQTT to InfluxDB Bridge Daemon

**Input**: Design documents from `/specs/001-mqtt2influxdb-bridge/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md

**Tests**: This feature explicitly requires unit and integration coverage from the specification and plan, so test tasks are included for every user story.

**Organization**: Tasks are grouped by user story so each story can be implemented and verified independently.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to (`[US1]`, `[US2]`, `[US3]`)
- Every task includes an exact file path

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create the Go module, local development scaffolding, and quality tooling required by the bridge package.

- [X] T001 Initialize the Go module and add bridge dependencies in `stutzthings/stutzthings-server/go.mod`
- [X] T002 Create build, test, lint, and local-run targets in `stutzthings/stutzthings-server/Makefile`
- [X] T003 [P] Add zero-warning GolangCI-Lint configuration in `stutzthings/stutzthings-server/.golangci.yml`
- [X] T004 [P] Create local MQTT and InfluxDB development stack in `stutzthings/stutzthings-server/docker-compose.yml`

**Checkpoint**: The server module can install dependencies, run lint/test commands, and boot local infrastructure.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Build the core bridge package skeleton and shared runtime primitives that every story depends on.

**⚠️ CRITICAL**: No user story work should start until this phase is complete.

- [X] T005 Implement environment-backed bridge configuration loading and validation in `stutzthings/stutzthings-server/bridge/config.go`
- [X] T006 [P] Define the bridge lifecycle, channels, and goroutine orchestration API in `stutzthings/stutzthings-server/bridge/bridge.go`
- [X] T007 [P] Create MQTT client construction with MQTT 3.1.1 auth and optional TLS settings in `stutzthings/stutzthings-server/bridge/mqtt.go`
- [X] T008 [P] Create InfluxDB client construction and writer scaffolding in `stutzthings/stutzthings-server/bridge/influx.go`

**Checkpoint**: Shared configuration, bridge lifecycle, MQTT client setup, and InfluxDB client setup are in place for story work.

---

## Phase 3: User Story 1 - Sensor Data Persisted to Time-Series Store (Priority: P1) 🎯 MVP

**Goal**: Accept valid MQTT attribute messages, parse scalar or JSON payloads, and persist typed records into InfluxDB batches.

**Independent Test**: Publish numeric, string, boolean, and JSON payloads to valid 5-segment topics and verify the records appear in InfluxDB within 2 seconds with correct tags and timestamps.

### Tests for User Story 1

- [X] T009 [P] [US1] Add topic validation and payload parsing unit tests in `stutzthings/stutzthings-server/bridge/payload_test.go`
- [X] T010 [P] [US1] Add end-to-end ingestion integration coverage for numeric, string, boolean, and JSON payloads in `stutzthings/stutzthings-server/bridge/bridge_integration_test.go`

### Implementation for User Story 1

- [X] T011 [US1] Implement topic parsing and scalar-or-JSON payload decoding in `stutzthings/stutzthings-server/bridge/payload.go`
- [X] T012 [US1] Implement batch accumulation and flush interval handling in `stutzthings/stutzthings-server/bridge/buffer.go`
- [X] T013 [US1] Implement InfluxDB point mapping and batch write execution for `device_attributes` in `stutzthings/stutzthings-server/bridge/influx.go`
- [X] T014 [US1] Connect MQTT message dispatch to the internal ingest channel in `stutzthings/stutzthings-server/bridge/mqtt.go`
- [X] T015 [US1] Wire the bridge start flow so valid MQTT messages reach the buffer and writer goroutines in `stutzthings/stutzthings-server/bridge/bridge.go`

**Checkpoint**: User Story 1 is independently functional and can persist valid device attributes to InfluxDB.

---

## Phase 4: User Story 2 - Connection Resilience (Priority: P2)

**Goal**: Recover automatically from MQTT or InfluxDB outages while preserving the documented retry, buffering, and eviction behavior.

**Independent Test**: Start the bridge, interrupt MQTT and InfluxDB availability, restore services, then verify reconnect, re-subscription, retry, and eventual persistence behavior without restarting the process.

### Tests for User Story 2

- [X] T016 [P] [US2] Add unit coverage for buffer overflow eviction and flush triggers in `stutzthings/stutzthings-server/bridge/buffer_test.go`
- [X] T017 [P] [US2] Extend resilience integration coverage for MQTT reconnect and InfluxDB retry behavior in `stutzthings/stutzthings-server/bridge/bridge_integration_test.go`

### Implementation for User Story 2

- [X] T018 [US2] Implement FIFO ring-buffer eviction with warning logs in `stutzthings/stutzthings-server/bridge/buffer.go`
- [X] T019 [US2] Implement startup readiness probing and bounded write retries in `stutzthings/stutzthings-server/bridge/influx.go`
- [X] T020 [US2] Implement MQTT auto-reconnect, `OnConnect` wildcard re-subscription, and reconnect logging in `stutzthings/stutzthings-server/bridge/mqtt.go`
- [X] T021 [US2] Implement coordinated shutdown and goroutine stop handling in `stutzthings/stutzthings-server/bridge/bridge.go`

**Checkpoint**: User Story 2 is independently functional and the bridge survives transient broker or InfluxDB outages without manual intervention.

---

## Phase 5: User Story 3 - Multi-Account and Multi-Device Isolation (Priority: P3)

**Goal**: Preserve the full account/device/node/attribute identity for concurrent messages so downstream access control remains meaningful.

**Independent Test**: Publish messages for multiple accounts, devices, and attributes concurrently, then verify every stored record is tagged with the correct identity hierarchy.

### Tests for User Story 3

- [X] T022 [P] [US3] Extend integration coverage for multi-account and multi-attribute tagging in `stutzthings/stutzthings-server/bridge/bridge_integration_test.go`

### Implementation for User Story 3

- [X] T023 [US3] Enforce full 5-segment identity extraction and invalid-topic rejection paths in `stutzthings/stutzthings-server/bridge/payload.go`
- [X] T024 [US3] Ensure all InfluxDB writes include the complete account, device, instance, node, and attribute tag set in `stutzthings/stutzthings-server/bridge/influx.go`
- [X] T025 [US3] Preserve per-message identity isolation through the ingest pipeline under concurrent load in `stutzthings/stutzthings-server/bridge/bridge.go`

**Checkpoint**: User Story 3 is independently functional and records remain isolated by full device identity across accounts and devices.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finish documentation, developer ergonomics, and feature verification that spans all stories.

- [X] T026 [P] Document local setup, environment variables, and bridge usage in `stutzthings/stutzthings-server/README.md`
- [X] T027 Update quickstart commands and verification steps to match the final implementation in `specs/001-mqtt2influxdb-bridge/quickstart.md`
- [X] T028 Run and stabilize the full lint and test workflow in `stutzthings/stutzthings-server/Makefile`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup completion and blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational completion and delivers the MVP.
- **User Story 2 (Phase 4)**: Depends on User Story 1 because resilience builds on the ingestion pipeline.
- **User Story 3 (Phase 5)**: Depends on User Story 1 and can proceed after the core ingest/write path is stable.
- **Polish (Phase 6)**: Depends on completion of the desired user stories.

### User Story Dependencies

- **US1**: No dependency on other user stories after Foundational.
- **US2**: Depends on US1 pipeline components in `buffer.go`, `mqtt.go`, `influx.go`, and `bridge.go`.
- **US3**: Depends on US1 payload parsing and InfluxDB write paths, but not on US2.

### Within Each User Story

- Tests should be written first and must fail before implementation begins.
- Parsing and model-shaping logic should land before pipeline wiring.
- Pipeline wiring should land before end-to-end verification.
- Each story should be validated independently before moving forward.

## Parallel Opportunities

- Phase 1 tasks `T003` and `T004` can run in parallel after `T001` and `T002` establish the module tooling.
- Phase 2 tasks `T006`, `T007`, and `T008` can run in parallel after `T005` defines shared configuration.
- In US1, `T009` and `T010` can run in parallel, and `T013` and `T014` can proceed in parallel once `T011` defines the parsed attribute shape.
- In US2, `T016` and `T017` can run in parallel, while `T018`, `T019`, and `T020` can be split across contributors before `T021` finishes shutdown coordination.
- In US3, `T022` can be prepared in parallel with `T023` and `T024`.

## Parallel Example: User Story 1

```bash
# Run the US1 tests in parallel:
Task: "Add topic validation and payload parsing unit tests in stutzthings/stutzthings-server/bridge/payload_test.go"
Task: "Add end-to-end ingestion integration coverage for numeric, string, boolean, and JSON payloads in stutzthings/stutzthings-server/bridge/bridge_integration_test.go"

# Split the implementation once payload parsing exists:
Task: "Implement InfluxDB point mapping and batch write execution for device_attributes in stutzthings/stutzthings-server/bridge/influx.go"
Task: "Connect MQTT message dispatch to the internal ingest channel in stutzthings/stutzthings-server/bridge/mqtt.go"
```

## Parallel Example: User Story 2

```bash
# Run resilience-focused tests in parallel:
Task: "Add unit coverage for buffer overflow eviction and flush triggers in stutzthings/stutzthings-server/bridge/buffer_test.go"
Task: "Extend resilience integration coverage for MQTT reconnect and InfluxDB retry behavior in stutzthings/stutzthings-server/bridge/bridge_integration_test.go"

# Split recovery implementation across files:
Task: "Implement FIFO ring-buffer eviction with warning logs in stutzthings/stutzthings-server/bridge/buffer.go"
Task: "Implement startup readiness probing and bounded write retries in stutzthings/stutzthings-server/bridge/influx.go"
Task: "Implement MQTT auto-reconnect, OnConnect wildcard re-subscription, and reconnect logging in stutzthings/stutzthings-server/bridge/mqtt.go"
```

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1.
4. Validate the US1 independent test before moving on.

### Incremental Delivery

1. Finish Setup and Foundational work to establish the bridge runtime.
2. Deliver US1 for end-to-end ingestion.
3. Add US2 for reconnect, retry, and buffering resilience.
4. Add US3 for multi-account and multi-device isolation verification.
5. Finish with Phase 6 documentation and full workflow validation.

### Suggested MVP Scope

- **MVP**: Phase 1, Phase 2, and Phase 3 (through `T015`)

## Notes

- All tasks follow the required checklist format with task IDs, optional `[P]` markers, and `[US#]` labels for story phases.
- No contracts directory exists for this feature, so no contract-specific tasks are included.
- The plan checklist still contains open author-review questions, but they do not block generation of an executable implementation task list for the current spec and plan.