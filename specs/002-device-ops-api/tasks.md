# Tasks: Device Operations API and MCP Access

**Input**: Design documents from `/specs/002-device-ops-api/`
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: Include service, MQTT interoperability, REST, and MCP parity tests because the specification requires independent verification per user story and exact REST/MCP business parity using the same JWT token model as Mosquitto.

**Organization**: Tasks are grouped by user story so each story can be implemented and validated independently.

## Phase 0: XDR Sync (Blocking Documentation)

**Purpose**: Bring the local XDR set promised by the plan into sync before implementation work begins.

- [ ] T044 Update the local product BDRs for MQTT topic/JWT ACL mapping, observation-query behavior, authorization scope grammar, and stateless registration policy in `.xdrs/_local/bdrs/product/001-mqtt-topic-structure.md`, `.xdrs/_local/bdrs/product/002-influxdb-device-attributes-data-model.md`, `.xdrs/_local/bdrs/product/003-device-authorization-scope-grammar.md`, and `.xdrs/_local/bdrs/product/004-stateless-device-registration-policy.md`
- [ ] T045 Update the local ADR and EDR for dual-surface operations and the one-command local workflow in `.xdrs/_local/adrs/architecture/002-device-operations-dual-surface-architecture.md` and `.xdrs/_local/edrs/patterns/002-local-device-operations-development-workflow.md`
- [ ] T046 Update `.xdrs/_local/bdrs/index.md`, `.xdrs/_local/adrs/index.md`, and `.xdrs/_local/edrs/index.md` with any new or revised feature-002 references created during the XDR sync

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add the dependencies and local workflow hooks required by the planned implementation.

- [ ] T001 Update dependency declarations for JWT and MCP support in `stutzthings/stutzthings-server/go.mod`
- [ ] T002 Update the one-command local verification workflow and shared JWT environment bootstrap in `stutzthings/stutzthings-server/Makefile` and `stutzthings/stutzthings-server/docker-compose.yml`
- [ ] T003 [P] Update the example local Mosquitto JWT plugin bootstrap and username=`sub` enforcement in `stutzthings/stutzthings-server/examples/local/docker-compose.yml` and `stutzthings/stutzthings-server/examples/local/mosquitto/mosquitto.conf`
- [ ] T004 [P] Document MQTT, REST, MCP, and shared JWT bootstrap requirements in `stutzthings/stutzthings-server/README.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Build the shared auth, transport, and service primitives that every user story depends on.

**⚠️ CRITICAL**: No user story work should begin until this phase is complete.

- [ ] T005 [P] Implement JWT claim validation and signing helpers in `stutzthings/stutzthings-server/auth/jwt.go`
- [ ] T006 [P] Implement MQTT topic-filter authorization matching for `publ` and `subs` claims in `stutzthings/stutzthings-server/auth/topic_acl.go`
- [ ] T007 [P] Add JWT claim and topic-filter matcher coverage in `stutzthings/stutzthings-server/auth/jwt_test.go` and `stutzthings/stutzthings-server/auth/topic_acl_test.go`
- [ ] T008 [P] Define shared device-operation domain models in `stutzthings/stutzthings-server/operations/models.go`
- [ ] T009 [P] Implement Influx row decoding and reusable query builders in `stutzthings/stutzthings-server/operations/influx_queries.go`
- [ ] T010 Implement the shared operations service facade for reads, history, desired-state, and registration in `stutzthings/stutzthings-server/operations/service.go`
- [ ] T011 [P] Implement bearer-token middleware and request auth context extraction in `stutzthings/stutzthings-server/auth/middleware.go`
- [ ] T012 [P] Implement shared device-identity, registration, and history-range validation used by both REST and MCP in `stutzthings/stutzthings-server/operations/validation.go`
- [ ] T013 [P] Define shared REST and MCP response DTOs, including registration claim payloads, in `stutzthings/stutzthings-server/api/response_types.go`
- [ ] T047 [P] Add REST/MCP validation-parity coverage for malformed identity paths and invalid history and registration inputs in `stutzthings/stutzthings-server/main_test.go`
- [ ] T048 [P] Add MQTT broker interoperability coverage proving connections are rejected when username does not equal JWT `sub` in `stutzthings/stutzthings-server/main_test.go`
- [ ] T014 [P] Create the MCP server bootstrap and tool registration scaffold in `stutzthings/stutzthings-server/mcpapi/server.go`
- [ ] T015 Update HTTP server bootstrap to mount auth, shared validation, split REST handlers, split MCP tools, and health handlers in `stutzthings/stutzthings-server/main.go`

**Checkpoint**: Foundation is ready and the user stories can now be implemented against stable auth, shared validation, service, and broker integration interfaces.

---

## Phase 3: User Story 1 - Read Current Device State (Priority: P1) 🎯 MVP

**Goal**: Let authorized clients read the latest known value for one attribute or the latest visible snapshot for one device instance.

**Independent Test**: Seed recent telemetry, call the attribute and whole-device read surfaces through REST and MCP with a JWT whose topic filters cover the addressed telemetry topics, and verify the latest values match storage while empty and partial reads behave as specified.

### Tests for User Story 1

- [ ] T016 [P] [US1] Add latest-state service coverage for attribute and whole-device reads, including empty and partial results, in `stutzthings/stutzthings-server/operations/query_service_test.go`
- [ ] T017 [P] [US1] Add REST handler coverage for latest-state reads in `stutzthings/stutzthings-server/api/state_handlers_test.go`
- [ ] T018 [P] [US1] Add MCP parity coverage for latest-state tools in `stutzthings/stutzthings-server/mcpapi/state_tools_test.go`

### Implementation for User Story 1

- [ ] T019 [US1] Implement latest-attribute and whole-device snapshot queries with topic-filter authorization and partial filtering in `stutzthings/stutzthings-server/operations/query_service.go`
- [ ] T020 [US1] Implement REST handlers for attribute and device snapshot reads in `stutzthings/stutzthings-server/api/state_handlers.go`
- [ ] T021 [US1] Implement MCP tools for attribute and device snapshot reads in `stutzthings/stutzthings-server/mcpapi/state_tools.go`

**Checkpoint**: User Story 1 is functional when authorized clients can read current state through both transports with identical business results.

---

## Phase 4: User Story 2 - Inspect Device History (Priority: P2)

**Goal**: Let authorized clients query time-bounded node and attribute history with deterministic ordering.

**Independent Test**: Seed multiple observations across a known window, call both history surfaces with explicit `from` and `to` bounds, and verify only matching records are returned in order while invalid ranges are rejected.

### Tests for User Story 2

- [ ] T022 [P] [US2] Add history query coverage for valid, empty, and invalid ranges in `stutzthings/stutzthings-server/operations/history_service_test.go`
- [ ] T023 [P] [US2] Add REST handler coverage for node and attribute history in `stutzthings/stutzthings-server/api/history_handlers_test.go`
- [ ] T024 [P] [US2] Add MCP parity coverage for history tools in `stutzthings/stutzthings-server/mcpapi/history_tools_test.go`

### Implementation for User Story 2

- [ ] T025 [US2] Implement ordered node and attribute history queries with explicit range enforcement in `stutzthings/stutzthings-server/operations/query_service.go`
- [ ] T026 [US2] Implement REST handlers for node and attribute history endpoints in `stutzthings/stutzthings-server/api/history_handlers.go`
- [ ] T027 [US2] Implement MCP tools for node and attribute history in `stutzthings/stutzthings-server/mcpapi/history_tools.go`

**Checkpoint**: User Story 2 is functional when both transports return the same ordered history results and validation failures for the same requests.

---

## Phase 5: User Story 3 - Send Desired State Updates (Priority: P3)

**Goal**: Let authorized clients publish desired-state commands to the existing device `/set` MQTT topic without mutating reported state.

**Independent Test**: Submit authorized and unauthorized desired-state requests through REST and MCP, verify only authorized calls publish the correct topic payload according to `subs` authorization, and confirm reads still reflect reported telemetry until new observations arrive.

### Tests for User Story 3

- [ ] T028 [P] [US3] Add desired-state publication coverage for `/set` authorization and topic selection in `stutzthings/stutzthings-server/operations/command_service_test.go`
- [ ] T029 [P] [US3] Add REST handler coverage for desired-state writes in `stutzthings/stutzthings-server/api/command_handlers_test.go`
- [ ] T030 [P] [US3] Add MCP parity coverage for desired-state tools in `stutzthings/stutzthings-server/mcpapi/command_tools_test.go`

### Implementation for User Story 3

- [ ] T031 [US3] Implement desired-state publication against the existing MQTT command channel in `stutzthings/stutzthings-server/operations/command_service.go`
- [ ] T032 [US3] Implement the REST desired-state endpoint in `stutzthings/stutzthings-server/api/command_handlers.go`
- [ ] T033 [US3] Implement the MCP desired-state tool in `stutzthings/stutzthings-server/mcpapi/command_tools.go`

**Checkpoint**: User Story 3 is functional when desired-state requests publish exactly once to the correct `/set` topic and transport parity holds.

---

## Phase 6: User Story 4 - Register a Device Instance (Priority: P4)

**Goal**: Let authorized onboarding clients mint a new device instance identifier and one plugin-compatible JWT without storing registration session state.

**Independent Test**: Submit registration requests with valid and invalid wildcard topic filters, verify successful responses return unique device instance identifiers plus narrowed `sub`, `publ`, and `subs` claims, and confirm the same token works for MQTT, REST, and MCP within the granted scope.

### Tests for User Story 4

- [ ] T034 [P] [US4] Add registration and token-issuance coverage in `stutzthings/stutzthings-server/operations/registration_service_test.go`
- [ ] T035 [P] [US4] Add REST handler coverage for registration in `stutzthings/stutzthings-server/api/registration_handlers_test.go`
- [ ] T036 [P] [US4] Add MCP parity coverage for registration in `stutzthings/stutzthings-server/mcpapi/registration_tools_test.go`

### Implementation for User Story 4

- [ ] T037 [US4] Implement stateless device registration and plugin-compatible JWT issuance in `stutzthings/stutzthings-server/operations/registration_service.go`
- [ ] T038 [US4] Implement the REST registration endpoint in `stutzthings/stutzthings-server/api/registration_handlers.go`
- [ ] T039 [US4] Implement the MCP registration tool in `stutzthings/stutzthings-server/mcpapi/registration_tools.go`

**Checkpoint**: User Story 4 is functional when registration returns unique device identities and constrained credentials through both transports.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Close the remaining parity, documentation, and verification gaps across all stories.

- [ ] T040 [P] Add end-to-end MQTT plus REST/MCP token interoperability regression coverage in `stutzthings/stutzthings-server/main_test.go`
- [ ] T041 [P] Add read-path performance verification for current-state and history in `stutzthings/stutzthings-server/operations/performance_test.go`
- [ ] T042 [P] Document the local MQTT, REST, and MCP verification flow in `stutzthings/stutzthings-server/examples/local/README.md`
- [ ] T043 Update the feature verification steps with the final implementation commands in `specs/002-device-ops-api/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 0: XDR Sync**: No dependencies and must complete before implementation work begins.
- **Phase 1: Setup**: Depends on Phase 0.
- **Phase 2: Foundational**: Depends on Phase 1 and blocks every user story.
- **Phase 3: User Story 1**: Depends on Phase 2 only and defines the MVP slice.
- **Phase 4: User Story 2**: Depends on Phase 2 and can begin after the foundation is stable, but should follow User Story 1 if one developer is implementing sequentially because both extend the same query path.
- **Phase 5: User Story 3**: Depends on Phase 2 and the shared command path assumptions created in the foundation.
- **Phase 6: User Story 4**: Depends on Phase 2 and the JWT signing plus broker-integration foundation.
- **Phase 7: Polish**: Depends on the completion of the stories you intend to ship.

### User Story Dependencies

- **US1 (P1)**: No dependency on other user stories after the foundation.
- **US2 (P2)**: No business dependency on US1, but reuses the same query service.
- **US3 (P3)**: No business dependency on US1 or US2, but reuses the same transport and auth scaffolding.
- **US4 (P4)**: No business dependency on other stories beyond the shared JWT and broker foundation.

### Within Each User Story

- Tests should be written before implementation and should fail before the matching code is added.
- Service logic should land before REST and MCP adapter work.
- REST and MCP implementations must preserve parity for success, empty, validation, and authorization outcomes.

### Parallel Opportunities

- `T044`, `T045`, and `T046` should run before code changes; `T044` and `T045` can proceed in parallel, then `T046` can finalize the index updates.
- `T001` and `T002` are sequential because dependency and workflow changes should settle before documentation updates; `T003` and `T004` can run in parallel once the workflow shape is known.
- In Phase 2, `T005`, `T006`, `T007`, `T008`, `T009`, `T011`, `T012`, `T013`, `T014`, `T047`, and `T048` can run in parallel because they target different files.
- In each user story, the three test tasks can run in parallel.
- After a story’s service implementation lands, the REST and MCP adapter tasks for that story can run in parallel.
- Cross-story work can be split across multiple developers after Phase 2, but shared files such as `operations/query_service.go`, `operations/service.go`, and `main.go` should be assigned carefully to avoid merge conflicts.

---

## Parallel Example: User Story 1

```bash
# Launch the User Story 1 test tasks together:
T016 operations/query_service_test.go
T017 api/state_handlers_test.go
T018 mcpapi/state_tools_test.go

# After T019 lands, implement both transport adapters in parallel:
T020 api/state_handlers.go
T021 mcpapi/state_tools.go
```

## Parallel Example: User Story 3

```bash
# Launch the User Story 3 test tasks together:
T028 operations/command_service_test.go
T029 api/command_handlers_test.go
T030 mcpapi/command_tools_test.go

# After T031 lands, implement both transport adapters in parallel:
T032 api/command_handlers.go
T033 mcpapi/command_tools.go
```

## Parallel Example: User Story 4

```bash
# Launch the User Story 4 test tasks together:
T034 operations/registration_service_test.go
T035 api/registration_handlers_test.go
T036 mcpapi/registration_tools_test.go

# After T037 lands, implement both transport adapters in parallel:
T038 api/registration_handlers.go
T039 mcpapi/registration_tools.go
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 0.
2. Complete Phase 1.
3. Complete Phase 2.
4. Complete Phase 3.
5. Validate MQTT, REST, and MCP token compatibility for current-state reads before widening scope.

### Incremental Delivery

1. Deliver XDR sync plus Setup + Foundational work as one shared base.
2. Deliver User Story 1 as the first usable slice.
3. Add User Story 2 for history once current-state parity is stable.
4. Add User Story 3 for desired-state publication.
5. Add User Story 4 for onboarding and plugin-compatible credential issuance.
6. Finish with interoperability, performance, and local verification documentation.

### Suggested MVP Scope

Deliver through **Phase 3 / User Story 1** first. That gives the project an authorized, transport-parity read surface over existing telemetry without waiting for command or registration flows.

---

## Notes

- All checklist items use the required `- [ ] T### [P] [US#] Description` format.
- User-story tasks always include a `[US#]` label; setup, foundational, and polish tasks do not.
- The task order is immediately executable against the current Go module layout under `stutzthings/stutzthings-server`.