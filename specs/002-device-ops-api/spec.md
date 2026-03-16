# Feature Specification: Device Operations API and MCP Access

**Feature Branch**: `002-device-ops-api`  
**Created**: 2026-03-17  
**Status**: Draft  
**Input**: User description: "Implement the selected stutzthings-server operations scope: device state reads, history reads, desired-state writes, scoped bearer-token authorization, stateless device registration, and exposure through both REST API and MCP server."

## Overview

Provide a single device-operations capability in stutzthings-server that lets clients read the latest device state, inspect recent history, send desired-state updates to devices, and register new device instances with scoped access. The same business capability must be available through both the HTTP API and the MCP server so that direct integrations and MCP hosts see consistent behavior.

This feature builds on the existing MQTT topic and time-series data rules already defined for device telemetry. It adds the query, command, and access-control surface that makes that data usable by applications and automation clients.

## Clarifications

### Session 2026-03-17

- Q: What scope grammar should bearer tokens use for device authorization? → A: `i:account_id/device_id/device_instance_id/node_name/attribute_name:actions`, with `*` allowed in any path segment
- Q: What credentials should registration return for a new device identity? → A: One credential scoped to the new device identity with `rws` actions
- Q: How should whole-device reads behave when the caller is authorized for only some attributes? → A: Return only authorized attributes and mark the result as partial when anything is excluded
- Q: What time-window contract should history queries use? → A: Explicit `from` and `to` timestamps
- Q: What authorization is required for device registration? → A: `POST /registration` requires a bearer token with scope `r:account_id/device_id`, with `*` allowed in all path levels

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Read Current Device State (Priority: P1)

An application or automation client needs to retrieve the latest known state of a device or one specific attribute so it can display current conditions or make a decision.

**Why this priority**: Read access is the minimum useful capability for the stored telemetry. Without it, the platform has collected data but provides no operational value to consuming clients.

**Independent Test**: Seed the platform with recent device telemetry, request the latest state for one attribute and for the whole device, and verify both responses return the newest stored values for the requested scope.

**Acceptance Scenarios**:

1. **Given** a device has reported a value for a specific attribute, **When** a client requests the latest value for that attribute, **Then** the platform returns the most recent stored value together with the attribute identity and observation time.
2. **Given** a device has reported values for multiple nodes and attributes, **When** a client requests the latest state for the device instance, **Then** the platform returns the latest known value for each reported attribute in that device instance.
3. **Given** a client is authorized for only some attributes within a device instance, **When** it requests current state for that device instance, **Then** the platform returns only the authorized attributes and indicates that the result is partial.

---

### User Story 2 - Inspect Device History (Priority: P2)

An operator or automation client needs recent device history for a node or a specific attribute so it can analyze trends, compare behavior over time, or troubleshoot a change.

**Why this priority**: Historical visibility is the next most valuable slice after current-state reads because it turns raw telemetry into diagnosable and explainable device behavior.

**Independent Test**: Seed multiple time-stamped readings for the same node and attribute across a known interval, request history with explicit `from` and `to` timestamps, and verify that only matching records inside the requested interval are returned in time order.

**Acceptance Scenarios**:

1. **Given** a node has multiple stored observations within the requested time window, **When** a client requests node history, **Then** the platform returns all matching observations for that node within the window.
2. **Given** an attribute has stored observations both inside and outside the requested time window, **When** a client requests attribute history, **Then** the platform returns only the observations within the requested window.
3. **Given** a client lacks read permission for the requested device scope, **When** it requests history, **Then** the platform denies the request without returning any device data.

---

### User Story 3 - Send Desired State Updates (Priority: P3)

An application needs to request a state change on a device, such as turning something on or updating a setting, and it needs that request to be delivered through the device command channel used by the platform.

**Why this priority**: Commanding devices is high value, but it depends on the read model and access-control rules already being in place so command requests can be scoped and audited correctly.

**Independent Test**: Submit a desired-state write for an authorized attribute, observe the command on the device command topic, and verify the platform returns a success response without altering stored state until the device later reports its new actual value.

**Acceptance Scenarios**:

1. **Given** a client has write access for an attribute, **When** it sends a desired-state update for that attribute, **Then** the platform publishes the requested value to the corresponding device command topic.
2. **Given** a client does not have write access for an attribute, **When** it attempts to send a desired-state update, **Then** the platform rejects the request and publishes no command.
3. **Given** a desired-state update has been accepted, **When** the device has not yet reported its actual new state, **Then** subsequent state reads continue to reflect the latest reported device value rather than assuming the command has already taken effect.

---

### User Story 4 - Register a Device Instance (Priority: P4)

An onboarding flow needs to create a new device identity and receive scoped credentials that the device can use immediately without requiring server-side registration state to be stored.

**Why this priority**: Registration is critical for scaling device onboarding, but the platform still delivers value without it if identities and credentials are provisioned manually.

**Independent Test**: Submit a registration request, verify a new device identity is returned with access credentials scoped to that identity, and confirm the credentials authorize only the granted actions.

**Acceptance Scenarios**:

1. **Given** a valid registration request, **When** the platform creates a new device instance identity, **Then** it returns the generated device identity and credentials scoped to that device instance.
2. **Given** a device uses registration-issued credentials, **When** it publishes telemetry and subscribes for commands, **Then** the credentials authorize only the actions granted for that device identity.
3. **Given** two consecutive registration requests for the same account and device type, **When** both are accepted, **Then** each request returns a distinct device instance identity.

### Edge Cases

- Requests for current state or history on a device instance that has no stored telemetry return an empty result rather than synthetic defaults.
- Requests with malformed or incomplete device identity paths are rejected before any data lookup or command publication occurs.
- A read scope that uses wildcards grants access only to the matching device identities and must not leak data outside the matched path.
- Whole-device reads with partial attribute authorization return only authorized attributes and explicitly indicate that additional attributes were excluded.
- A write request to an authorized attribute publishes a desired-state command but does not itself create or overwrite the device's reported state.
- Registration requests without a matching registration scope are rejected and do not create device identities or issue credentials.
- Registration failures must not consume or expose a partially created device identity.
- History requests with invalid, unparseable, or inverted `from`/`to` timestamps are rejected with a client-visible validation error.
- If the requested history window contains a very large number of records, the platform still returns results in a deterministic order and does not mix data from other device identities.
- REST API and MCP access must return equivalent outcomes for the same request, including authorization decisions and empty-result behavior.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The platform MUST provide an operation to read the latest known value for a specific attribute identified by account, device type, device instance, node, and attribute.
- **FR-002**: The platform MUST provide an operation to read the latest known state for an entire device instance, aggregated from the latest stored value of each reported attribute in that device instance.
- **FR-003**: The platform MUST provide an operation to read historical observations for a specific node within a device instance over a caller-supplied `from` and `to` time range.
- **FR-004**: The platform MUST provide an operation to read historical observations for a specific attribute within a device instance over a caller-supplied `from` and `to` time range.
- **FR-005**: Current-state and history operations MUST return only data belonging to the exact requested account, device type, device instance, node, and attribute scope.
- **FR-006**: Whole-device current-state reads MUST include only the attributes authorized by the caller's scopes.
- **FR-007**: If any device attributes are excluded from a whole-device current-state read due to authorization filtering, the response MUST indicate that the result is partial.
- **FR-008**: The platform MUST expose the same device-operations capability through both the REST API and the MCP server.
- **FR-009**: For the same request scope and authorization context, REST API and MCP responses MUST produce the same success or failure outcome and equivalent business data.
- **FR-010**: Protected device operations MUST require a bearer token whose scope claims authorize the requested device path and action before any data is returned or any command is published.
- **FR-011**: Device-operation scope claims MUST use the grammar `i:account_id/device_id/device_instance_id/node_name/attribute_name:actions`.
- **FR-012**: The `actions` suffix in a device-operation scope claim MUST support `r` for current-state and history reads, `w` for attribute-value writes where applicable, and `s` for desired-state command publication.
- **FR-013**: Device-operation scope claims MUST support `*` in any path segment, and wildcard evaluation MUST be limited to the matching account, device type, device instance, node, and attribute paths requested by the caller.
- **FR-014**: If a bearer token does not grant the requested device-operation action for the requested device path, the platform MUST reject the request and return no device data.
- **FR-015**: The platform MUST provide an operation that accepts a desired-state value for a specific attribute and publishes that value to the corresponding device command topic for the addressed device identity.
- **FR-016**: Desired-state publication MUST target the existing device command channel that corresponds to the requested attribute's identity path and desired-state suffix.
- **FR-017**: Accepting a desired-state write MUST NOT be treated as confirmation that the device changed state; subsequent reads MUST continue to reflect only reported device telemetry until a new reported value is ingested.
- **FR-018**: The platform MUST provide a stateless registration operation that creates a new device instance identity and returns one credential scoped for that device's allowed telemetry and command interactions.
- **FR-019**: `POST /registration` MUST require a bearer token with registration scope `r:account_id/device_id`.
- **FR-020**: Registration scope claims MUST support `*` in any path segment, including account and device type levels, for example `r:*/*`.
- **FR-021**: Each successful registration request MUST return a newly generated device instance identity that is unique within the account and device type scope.
- **FR-022**: Registration-issued credentials MUST be limited to the new device identity and MUST grant `rws` actions for that identity.
- **FR-023**: The platform MUST validate all required path segments for every operation and reject malformed requests before touching storage or publishing commands.
- **FR-024**: History operations MUST require caller-supplied `from` and `to` timestamps and MUST apply that range consistently to the returned observations.
- **FR-025**: When no data exists for an otherwise valid current-state or history request, the platform MUST return an empty result rather than an authorization error or fabricated default values.
- **FR-026**: The platform MUST preserve tenant isolation by ensuring that account identifiers remain part of every read, write, command, and registration decision.
- **FR-027**: The platform MUST make it possible to run the local development stack in a single command so the device-operations flow can be exercised end to end in a developer environment.

### Key Entities

- **Device Identity**: The full address of a device context, composed of account, device type, device instance, node, and attribute segments used for reads, history queries, and command targeting.
- **Device State Snapshot**: The latest known reported values for all attributes within one device instance at the time of a read request.
- **Partial Device State Snapshot**: A whole-device read result that contains only the attributes the caller is authorized to see and explicitly indicates that additional attributes were excluded.
- **Device Observation**: One stored time-stamped report for a node or attribute, used to serve current-state and history queries.
- **Desired State Command**: A requested target value for a specific device attribute that is published to the device command channel and later confirmed only by subsequent reported telemetry.
- **Access Scope**: A bearer-token claim using either the device-operation grammar `i:account_id/device_id/device_instance_id/node_name/attribute_name:actions` or the registration grammar `r:account_id/device_id`, where `*` may appear in any path segment and the prefix determines which protected capability is being authorized.
- **Registration Grant**: The output of a successful registration request, containing the newly created device instance identity and one scoped credential with `rws` actions for that identity.
- **History Range**: The explicit `from` and `to` timestamps that bound a history query for a node or attribute.

### Assumptions

- The selected feature scope covers only the operations listed in the source spec excerpt: current-state reads, history reads, desired-state writes, scoped authorization, device registration, and dual REST/MCP exposure.
- Existing telemetry ingestion and storage rules remain the source of truth for how device observations are stored and retrieved.
- Scope claims use the full 5-segment hierarchical device path plus action suffix and support `*` in any path segment for bounded delegation.
- Registration authorization uses a separate scope family `r:account_id/device_id`, with `*` accepted at both path levels.
- History queries use explicit `from` and `to` timestamps rather than relative durations.
- Registration is stateless from the server perspective after credentials are issued; the server does not maintain session state for the registration transaction.
- Registration returns a single credential for the new device identity rather than separate credentials per operation type.
- A desired-state command is considered delivered when the platform successfully hands it to the device command channel, not when the physical device has completed the action.
- The local developer workflow must support starting the supporting services and the server together for manual end-to-end verification.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In a healthy environment, 95% of current-state read requests return a complete result or a valid empty result in under 2 seconds.
- **SC-002**: In a healthy environment, 95% of history requests for a 2-day window return matching observations in under 3 seconds.
- **SC-003**: 100% of unauthorized read and desired-state write attempts are denied without exposing device data outside the granted scope.
- **SC-004**: 100% of successful desired-state write requests publish to the addressed device command path without changing the reported state until the device later reports a new value.
- **SC-005**: 100% of successful registration requests return a unique device instance identity and one scoped credential that grants `rws` actions only for that identity.
- **SC-006**: For the same request and authorization context, REST API and MCP executions produce equivalent business results in 100% of conformance tests.

## XDR Candidates *(filled by speckit.specify, realised by speckit.plan)*

- [BDR] Device authorization scope grammar and wildcard rules — the bearer-token claim uses the full 5-segment device path plus action suffix, with `*` allowed in any path segment for bounded path-level access.
- [BDR] Stateless device registration policy — registration requires `r:account_id/device_id` authorization, then issues unique device identities and a single `rws`-scoped credential without server-held registration session state.
- [BDR] MQTT topic structure and desired-state suffix — command publication and device addressing depend on the existing path structure and `/set` convention. (exists: ../../.xdrs/_local/bdrs/product/001-mqtt-topic-structure.md)
- [BDR] Device observation query model — current-state and history reads depend on the existing stored observation identity and time-series model. (exists: ../../.xdrs/_local/bdrs/product/002-influxdb-device-attributes-data-model.md)
- [ADR] Dual-surface device operations contract — the same business capability is intentionally exposed through REST and MCP and requires parity rules across both interaction surfaces.
- [EDR] Local end-to-end developer stack startup — the feature requires a consistent one-command workflow to boot supporting services and the server for full-flow verification.
