# Feature Specification: Device Operations API and MCP Access

**Feature Branch**: `002-device-ops-api`  
**Created**: 2026-03-17  
**Status**: Draft  
**Input**: User description: "Implement the selected stutzthings-server operations scope: device state reads, history reads, desired-state writes, scoped bearer-token authorization, stateless device registration, MQTT broker authorization, and exposure through both REST API and MCP server."

## Overview

Provide a single device-operations capability in stutzthings-server that lets clients read the latest device state, inspect recent history, send desired-state updates to devices, register new device instances with scoped access, and use the same JWT credentials across MQTT, REST, and MCP. Current-state, history, desired-state, and registration operations are exposed through REST and MCP, while MQTT remains the shared telemetry and desired-state transport plus the ACL authority that those operations reuse. The auth model must align with the Mosquitto `wiomoc/mosquitto-jwt-auth` plugin so that broker ACL enforcement and server-side authorization both derive from the same JWT claims.

This feature builds on the existing MQTT topic and time-series data rules already defined for device telemetry. It adds the query, command, registration, and unified access-control surface that makes that data usable by applications, automation clients, and devices themselves.

## Clarifications

### Session 2026-03-17

- Q: What JWT format should bearer tokens use for MQTT, REST, and MCP authorization? → A: Mosquitto-plugin-compatible claims using `sub`, `publ`, and `subs`
- Q: How must MQTT topics map into JWT authorization? → A: `publ` and `subs` store MQTT topic filters following the existing topic standard `account/device/device_instance/node/attribute` and `/set` suffix
- Q: What credentials should registration return for a new device identity? → A: One JWT with `sub=account/device/device_instance`, `publ=[account/device/device_instance/+/+]`, and `subs=[account/device/device_instance/+/+/set]`
- Q: How should whole-device reads behave when the caller is authorized for only some attributes? → A: Return only authorized attributes and mark the result as partial when anything is excluded
- Q: What time-window contract should history queries use? → A: Explicit `from` and `to` timestamps
- Q: What exact time semantics do history queries use? → A: `from` is inclusive, `to` is exclusive, both must be UTC RFC3339 timestamps, and legacy relative forms such as `since=2d` are not accepted by this feature
- Q: What qualifies as REST/MCP parity? → A: The same input and bearer token must yield the same validation class, authorization outcome, empty-result semantics, error code, and business fields; only the transport envelope differs
- Q: What distinguishes a partial device snapshot from a device with sparse telemetry? → A: A partial snapshot must include `is_partial=true` and `excluded_attribute_count>0`; an empty but authorized snapshot returns `attributes=[]`, `is_partial=false`, and `excluded_attribute_count=0`
- Q: How are JWT topic filters interpreted across reads and writes? → A: Current-state and history operations authorize against the 5-segment telemetry topic, desired-state writes authorize against the 6-segment `/set` topic, `+` matches exactly one segment, and `#` matches the remaining trailing segments only
- Q: What additional claims must a registration-issued JWT include? → A: The token must also include `iss`, `iat`, and `exp`; `exp` defaults to 24 hours after issuance unless overridden by server configuration
- Q: How is registration uniqueness defined? → A: `device_instance_id` is a lowercase UUIDv7 string generated server-side and retried on collision until unique within `(account_id, device_id)`

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Read Current Device State (Priority: P1)

An application, automation client, or device principal needs to retrieve the latest known state of a device or one specific attribute so it can display current conditions or make a decision.

**Why this priority**: Read access is the minimum useful capability for the stored telemetry. Without it, the platform has collected data but provides no operational value to consuming clients.

**Independent Test**: Seed the platform with recent device telemetry, request the latest state for one attribute and for the whole device with a JWT whose MQTT topic filters cover the addressed telemetry topics, and verify both responses return the newest stored values for the requested scope.

**Acceptance Scenarios**:

1. **Given** a device has reported a value for a specific attribute, **When** a client requests the latest value for that attribute with a JWT whose `publ` or `subs` claim matches the telemetry topic, **Then** the platform returns the most recent stored value together with the attribute identity and observation time.
2. **Given** a device has reported values for multiple nodes and attributes, **When** a client requests the latest state for the device instance, **Then** the platform returns the latest known value for each reported attribute the caller is authorized to read.
3. **Given** a client is authorized for only some attributes within a device instance, **When** it requests current state for that device instance, **Then** the platform returns only the authorized attributes and indicates that the result is partial.
4. **Given** a client requests one attribute and the token does not authorize that exact telemetry topic, **When** the request is evaluated, **Then** the platform rejects the request rather than returning a partial attribute result.
5. **Given** a client is authorized for a device instance that has no stored telemetry, **When** it requests whole-device current state, **Then** the platform returns an empty successful snapshot with `attributes=[]`, `is_partial=false`, and `excluded_attribute_count=0`.

---

### User Story 2 - Inspect Device History (Priority: P2)

An operator, automation client, or device principal needs recent device history for a node or a specific attribute so it can analyze trends, compare behavior over time, or troubleshoot a change.

**Why this priority**: Historical visibility is the next most valuable slice after current-state reads because it turns raw telemetry into diagnosable and explainable device behavior.

**Independent Test**: Seed multiple time-stamped readings for the same node and attribute across a known interval, request history with explicit `from` and `to` timestamps using a JWT whose MQTT topic filters cover the addressed telemetry topics, and verify that only matching records inside the requested interval are returned in time order.

**Acceptance Scenarios**:

1. **Given** a node has multiple stored observations within the requested time window, **When** a client requests node history, **Then** the platform returns all matching observations for that node within the window in deterministic order.
2. **Given** an attribute has stored observations both inside and outside the requested time window, **When** a client requests attribute history, **Then** the platform returns only the observations within the requested window.
3. **Given** a client lacks telemetry-topic authorization for the requested device scope, **When** it requests history, **Then** the platform denies the request without returning any device data.
4. **Given** a valid node-history or attribute-history request with no matching observations, **When** the request is processed, **Then** the platform returns `observations=[]` with the validated `from` and `to` values rather than treating the request as forbidden.
5. **Given** multiple observations for the same history result share the same timestamp, **When** they are returned, **Then** they remain ordered by `observed_at` ascending and then by `(node_name, attribute_name, canonical_value_string)` ascending so REST and MCP remain deterministic.

---

### User Story 3 - Send Desired State Updates (Priority: P3)

An application, automation client, or device principal needs to request a state change on a device, such as turning something on or updating a setting, and it needs that request to be delivered through the device command channel used by the platform.

**Why this priority**: Commanding devices is high value, but it depends on the read model and access-control rules already being in place so command requests can be scoped and audited correctly.

**Independent Test**: Submit a desired-state write for an authorized attribute using a JWT whose `subs` claim matches the corresponding `/set` topic, observe the command on the device command topic, and verify the platform returns a success response without altering stored state until the device later reports its new actual value.

**Acceptance Scenarios**:

1. **Given** a client has command-channel authority for an attribute, **When** it sends a desired-state update for that attribute, **Then** the platform publishes the requested value to the corresponding device command topic.
2. **Given** a client does not have command-channel authority for an attribute, **When** it attempts to send a desired-state update, **Then** the platform rejects the request and publishes no command.
3. **Given** a desired-state update has been accepted, **When** the device has not yet reported its actual new state, **Then** subsequent state reads continue to reflect the latest reported device value rather than assuming the command has already taken effect.

---

### User Story 4 - Register a Device Instance (Priority: P4)

An onboarding flow needs to create a new device identity and receive plugin-compatible JWT credentials that the device can use immediately with Mosquitto and with the REST/MCP surfaces without requiring server-side registration state to be stored.

**Why this priority**: Registration is critical for scaling device onboarding, but the platform still delivers value without it if identities and credentials are provisioned manually.

**Independent Test**: Submit a registration request using a JWT whose wildcard MQTT topic filters cover the requested account and device namespace, verify a new device identity is returned with `sub`, `publ`, and `subs` claims narrowed to that identity, and confirm the same JWT works against MQTT, REST, and MCP for the granted device scope.

**Acceptance Scenarios**:

1. **Given** a valid registration request, **When** the platform creates a new device instance identity, **Then** it returns the generated device identity and one JWT whose `sub`, `publ`, and `subs` claims are limited to that device instance.
2. **Given** a device uses registration-issued credentials, **When** it publishes telemetry, subscribes for `/set` commands, and calls protected REST or MCP operations for its own device scope, **Then** the credentials authorize only the granted topic filters for that device identity.
3. **Given** two consecutive registration requests for the same account and device type, **When** both are accepted, **Then** each request returns a distinct device instance identity.
4. **Given** a registration request whose bearer token does not wildcard-cover the requested `account_id/device_id` namespace, **When** the request is processed, **Then** the platform rejects it, issues no credentials, and does not reserve a device instance identity.

### Edge Cases

- Requests for current state or history on a device instance that has no stored telemetry return an empty result rather than synthetic defaults.
- Requests with malformed or incomplete device identity paths are rejected before any data lookup or command publication occurs.
- JWT `publ` and `subs` claims use standard MQTT topic-filter semantics; authorization must not leak data outside the matched account, device, instance, node, and attribute paths.
- JWT `+` matches exactly one MQTT segment and `#` matches only trailing remaining segments; wildcard filters must never skip the mandatory `account_id` segment or bridge telemetry-path reads to `/set` command authorization.
- Whole-device reads with partial attribute authorization return only authorized attributes and explicitly indicate that additional attributes were excluded.
- Attribute-level reads do not return partial results; they either return one authorized latest observation, an authorized empty result, or a denial.
- A desired-state request to an authorized attribute publishes a command but does not itself create or overwrite the device's reported state.
- Registration requests whose JWT topic filters do not cover the requested account and device namespace are rejected and do not create device identities or issue credentials.
- Registration failures must not consume or expose a partially created device identity.
- History requests with invalid, unparseable, or inverted `from`/`to` timestamps are rejected with a client-visible validation error.
- History requests must reject legacy relative-window inputs such as `since=2d`; only explicit `from` and `to` UTC RFC3339 timestamps are valid.
- If the requested history window contains a very large number of records, the platform still returns results in deterministic order and does not mix data from other device identities.
- If a history request would return more than 10,000 observations, the platform rejects it with a client-visible `window_too_large` validation error instead of truncating silently.
- REST API and MCP access must return equivalent outcomes for the same request, including authorization decisions and empty-result behavior.
- REST API and MCP parity includes success payload fields, empty-result payload shapes, validation errors, authorization denials, and backend-failure classes for the same request and bearer token.
- When a JWT is used with Mosquitto, the MQTT username must equal the `sub` claim or the broker rejects the connection.
- Registration-issued JWTs include `iss`, `iat`, and `exp`; the initial lifetime is 24 hours by default, and rotating the shared signing secret invalidates older tokens unless an overlap window is configured.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The platform MUST provide an operation to read the latest known value for a specific attribute identified by account, device type, device instance, node, and attribute.
- **FR-002**: The platform MUST provide an operation to read the latest known state for an entire device instance, aggregated from the latest stored value of each reported attribute in that device instance.
- **FR-003**: The platform MUST provide an operation to read historical observations for a specific node within a device instance over a caller-supplied `from` and `to` time range.
- **FR-004**: The platform MUST provide an operation to read historical observations for a specific attribute within a device instance over a caller-supplied `from` and `to` time range.
- **FR-005**: Current-state and history operations MUST return only data belonging to the exact requested account, device type, device instance, node, and attribute scope.
- **FR-006**: Whole-device current-state reads MUST include only the attributes authorized by the caller's JWT topic filters.
- **FR-007**: If any device attributes are excluded from a whole-device current-state read due to authorization filtering, the response MUST indicate that the result is partial.
- **FR-007a**: A whole-device current-state response that is partial MUST include an `excluded_attribute_count` greater than zero so clients can distinguish authorization filtering from simply missing telemetry.
- **FR-008**: The platform MUST expose the same current-state, history, desired-state, and registration capability through the REST API and the MCP server, while MQTT remains the shared telemetry and `/set` command transport whose ACL enforcement uses the same JWT claim model as REST and MCP authorization.
- **FR-009**: For the same request scope and authorization context, REST API and MCP responses MUST produce the same success or failure outcome and equivalent business data.
- **FR-009a**: REST/MCP parity MUST cover the same business fields, empty-result payload shapes, validation class, authorization class, and backend-failure class for the same input and bearer token; only the transport envelope and protocol-native status representation may differ.
- **FR-010**: Protected device operations MUST require a bearer token whose Mosquitto-plugin-compatible JWT claims authorize the MQTT topic path corresponding to the requested read, history, or desired-state operation before any data is returned or any command is published.
- **FR-011**: JWTs accepted by MQTT, REST, and MCP MUST use the claims `sub`, `publ`, and `subs`, and MUST remain compatible with `wiomoc/mosquitto-jwt-auth`.
- **FR-012**: When a JWT is used with MQTT, the MQTT username MUST equal the `sub` claim.
- **FR-013**: `publ` and `subs` claims MUST contain MQTT topic filters that follow standard MQTT wildcard semantics and the existing topic hierarchy `account_id/device_id/device_instance_id/node_name/attribute_name` with `/set` used for desired-state commands.
- **FR-014**: Current-state and history reads MUST be authorized only when the requested telemetry topic matches at least one filter in the caller's `publ` or `subs` claim; otherwise the platform MUST reject the request and return no device data.
- **FR-014a**: Current-state and history authorization MUST evaluate only against the 5-segment telemetry topic, while desired-state authorization MUST evaluate only against the 6-segment `/set` topic; authorization for one topic shape MUST NOT imply authorization for the other.
- **FR-015**: The platform MUST provide an operation that accepts a desired-state value for a specific attribute and publishes that value to the corresponding device command topic for the addressed device identity.
- **FR-016**: Desired-state publication through REST and MCP MUST be authorized only when the requested `/set` topic matches at least one filter in the caller's `subs` claim.
- **FR-017**: Accepting a desired-state write MUST NOT be treated as confirmation that the device changed state; subsequent reads MUST continue to reflect only reported device telemetry until a new reported value is ingested.
- **FR-018**: The platform MUST provide a stateless registration operation that creates a new device instance identity and returns one Mosquitto-plugin-compatible JWT scoped for MQTT, REST, and MCP operations on that device identity.
- **FR-019**: `POST /registration` MUST require a bearer token whose wildcard `publ` and `subs` filters authorize the requested account and device namespace before a new device instance can be created.
- **FR-020**: Registration authorization and issued credentials MUST use MQTT topic filters and MUST NOT rely on custom `i:` or `r:` scope grammars.
- **FR-021**: Each successful registration request MUST return a newly generated device instance identity that is unique within the account and device type scope.
- **FR-021a**: `device_instance_id` MUST be generated as a lowercase UUIDv7 string; if a collision is detected within `(account_id, device_id)`, generation MUST retry until a unique identifier is produced.
- **FR-022**: Registration-issued credentials MUST be limited to the new device identity and MUST include `sub=account_id/device_id/device_instance_id`, `publ=[account_id/device_id/device_instance_id/+/+]`, and `subs=[account_id/device_id/device_instance_id/+/+/set]`.
- **FR-022a**: Registration-issued JWTs MUST also include `iss`, `iat`, and `exp`; the default token lifetime is 24 hours unless overridden by deployment configuration.
- **FR-023**: The platform MUST validate all required path segments for every operation and all required registration request body fields, and reject malformed requests before touching storage or publishing commands.
- **FR-024**: History operations MUST require caller-supplied `from` and `to` timestamps and MUST apply that range consistently to the returned observations.
- **FR-024a**: `from` MUST be inclusive, `to` MUST be exclusive, both timestamps MUST be RFC3339 UTC instants, and legacy relative-window query forms such as `since=2d` MUST be rejected.
- **FR-024b**: History operations MUST return observations ordered by `observed_at` ascending and, when timestamps are equal within one result set, by `(node_name, attribute_name, canonical_value_string)` ascending.
- **FR-025**: When no data exists for an otherwise valid current-state or history request, the platform MUST return an empty result rather than an authorization error or fabricated default values.
- **FR-025a**: Empty successful responses MUST use explicit transport-stable shapes: attribute reads return `found=false` and `observation=null`; whole-device reads return `attributes=[]`, `is_partial=false`, and `excluded_attribute_count=0`; history reads return `observations=[]` together with the validated `from` and `to` values.
- **FR-026**: The platform MUST preserve tenant isolation by ensuring that account identifiers remain part of every MQTT topic, JWT topic filter, read decision, write decision, command publication, and registration decision.
- **FR-027**: The platform MUST make it possible to run the local development stack in a single command with Mosquitto configured to use `wiomoc/mosquitto-jwt-auth` so the device-operations flow can be exercised end to end in a developer environment.
- **FR-027a**: The single-command development workflow is a developer-experience requirement for local verification and MUST NOT be interpreted as a production runtime topology requirement.
- **FR-028**: If a history request would return more than 10,000 observations, the platform MUST reject it with a client-visible validation error rather than truncating or partially streaming the result silently.
- **FR-029**: The service and Mosquitto plugin MUST consume the same base64-encoded JWT signing secret source; coordinated secret rotation behavior and its impact on previously issued device-registration tokens MUST be documented for operators.
- **FR-030**: The platform MUST emit structured audit or observability events for authorization denials, successful registration issuance, failed registration attempts, and desired-state publication failures without logging raw bearer tokens or shared secrets.
- **FR-031**: MCP authentication MUST be supplied as a bearer token on each HTTP request to `/mcp`; the server MUST NOT rely on transport-session auth state that differs from REST.

### Key Entities

- **Device Identity**: The full address of a device context, composed of account, device type, device instance, node, and attribute segments used for reads, history queries, command targeting, and MQTT ACL derivation.
- **JWT Access Claims**: The shared auth model composed of `sub`, `publ`, `subs`, `iat`, and `exp`, reused across Mosquitto, REST, and MCP.
- **Device State Snapshot**: The latest known reported values for all attributes within one device instance at the time of a read request, returned with `is_partial` and `excluded_attribute_count` metadata.
- **Partial Device State Snapshot**: A whole-device read result that contains only the attributes the caller is authorized to see, sets `is_partial=true`, and reports a positive `excluded_attribute_count` because additional stored attributes were excluded.
- **Device Observation**: One stored time-stamped report for a node or attribute, used to serve current-state and history queries.
- **Desired State Command**: A requested target value for a specific device attribute that is published to the device command channel and later confirmed only by subsequent reported telemetry.
- **Registration Grant**: The output of a successful registration request, containing the newly created device instance identity and one JWT with `sub`, `publ`, `subs`, `iss`, `iat`, and `exp` claims narrowed to that identity.
- **History Range**: The explicit UTC RFC3339 `from` and `to` timestamps that bound a history query for a node or attribute using inclusive lower and exclusive upper semantics.

### Assumptions

- The selected feature scope covers only the operations listed in the source spec excerpt: current-state reads, history reads, desired-state writes, JWT-backed authorization, device registration, MQTT broker authorization, and dual REST/MCP exposure.
- Existing telemetry ingestion and storage rules remain the source of truth for how device observations are stored and retrieved.
- JWT claims use the Mosquitto plugin format `sub`, `publ`, and `subs` rather than custom scope strings.
- `publ` and `subs` use standard MQTT topic-filter semantics and the existing telemetry topic hierarchy plus `/set` suffix.
- REST and MCP authorization are derived from those same MQTT topic filters rather than a parallel auth model.
- Legacy source examples that used relative history inputs such as `since=2d` are superseded by the explicit `from`/`to` contract in this feature and should not be implemented.
- MQTT remains the transport for telemetry publication and `/set` command delivery; it is not a separate query surface for current-state or history operations.
- Registration is stateless from the server perspective after credentials are issued; the server does not maintain session state for the registration transaction.
- Registration returns a single JWT for the new device identity rather than separate credentials per transport.
- A desired-state command is considered delivered when the platform successfully hands it to the device command channel, not when the physical device has completed the action.
- The local developer workflow must support starting the supporting services, Mosquitto JWT plugin, and the server together for manual end-to-end verification.
- The first implementation uses a default 24-hour registration-token lifetime and coordinated shared-secret rotation across the Go service and Mosquitto plugin.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In a healthy environment, 95% of current-state read requests return a complete result or a valid empty result in under 2 seconds.
- **SC-002**: In a healthy environment, 95% of attribute-history requests for a 2-day window containing 2,880 observations return matching ordered observations in under 3 seconds.
- **SC-003**: 100% of unauthorized read and desired-state write attempts are denied without exposing device data outside the granted MQTT topic filters.
- **SC-004**: 100% of successful desired-state write requests publish exactly one command to the addressed `/set` topic and return an accepted response that identifies that topic without changing the reported state until the device later reports a new value.
- **SC-005**: 100% of successful registration requests return a unique device instance identity and one JWT whose `sub`, `publ`, and `subs` claims are limited to that identity.
- **SC-006**: For the same request and authorization context, REST API and MCP executions produce the same business fields, empty-result payload shape, validation class, authorization class, and backend-failure class in 100% of conformance tests.

### Performance Verification Protocol

- The latency goals in SC-001 and SC-002 are measured against a healthy local Docker-backed stack with InfluxDB, Mosquitto, and the Go service running without induced failures.
- Seed one account containing 100 device instances, each with 20 attributes, and store 2 days of 1-minute observations per attribute before timing begins.
- Warm the system with 20 untimed authorized requests per scenario before collecting measurements.
- Measure 200 timed authorized REST requests per scenario at concurrency 10.
- The current-state scenario measures whole-device snapshot reads for one device instance with all 20 attributes authorized.
- The history scenario measures one authorized attribute-history request over the full 2-day window for one attribute containing 2,880 observations.
- Latency is measured end to end from HTTP request start until the full response body is read, and p95 is computed across the 200 timed requests.

## XDR Candidates *(filled by speckit.specify, realised by speckit.plan)*

- [BDR] MQTT topic structure and JWT ACL mapping — telemetry, `/set`, and JWT topic filters all derive from the same canonical MQTT topic hierarchy. (exists: ../../.xdrs/_local/bdrs/product/001-mqtt-topic-structure.md)
- [BDR] JWT topic-filter claims for MQTT, REST, and MCP — authorization must use plugin-compatible `sub`, `publ`, and `subs` claims across all three surfaces.
- [BDR] Stateless device registration policy — registration must issue a single JWT with plugin-compatible `sub`, `publ`, and `subs` claims narrowed to the new device identity.
- [BDR] Device observation query model — current-state and history reads depend on the existing stored observation identity and time-series model. (exists: ../../.xdrs/_local/bdrs/product/002-influxdb-device-attributes-data-model.md)
- [ADR] Dual-surface device operations contract — the same business capability is intentionally exposed through REST and MCP and must share the JWT claim model already used by MQTT.
- [EDR] Local end-to-end developer stack startup — the feature requires a consistent one-command workflow to boot supporting services, Mosquitto JWT plugin, and the server for full-flow verification.
