# Feature Specification: MQTT to InfluxDB Bridge Daemon

**Feature Branch**: `001-mqtt2influxdb-bridge`  
**Created**: 2026-03-14  
**Status**: Draft  
**Input**: User description: "let's implement the mqtt2influx part of the server"

## Overview

A background daemon running within the stutzthings-server process that subscribes to MQTT topics published by IoT devices and persists the received attribute data into InfluxDB for later querying and historical analysis. This is the core data ingestion pipeline for the platform.

The MQTT topic structure follows the device hierarchy:
`[account_id]/[device_id]/[device_instance_id]/[node_name]/[attribute_name]`

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Sensor Data Persisted to Time-Series Store (Priority: P1)

An IoT device publishes a sensor reading (e.g., temperature, humidity, battery level) to its MQTT attribute topic. The bridge receives the message and stores it in InfluxDB so it can later be queried via the REST API or MCP server.

**Why this priority**: This is the foundational capability of the entire platform — without it, no device data is queryable, the REST endpoints return nothing, and no value is delivered. Everything else depends on this working.

**Independent Test**: Publish a message to a topic matching the device hierarchy pattern, wait 1 second, then query InfluxDB and verify the value appears with correct tags and timestamp.

**Acceptance Scenarios**:

1. **Given** a device publishes a numeric value to `account1/sensor/device01/temperature/celsius`, **When** the bridge processes the message, **Then** the value is stored in InfluxDB tagged with account_id, device_id, device_instance_id, node_name, and attribute_name, with a timestamp matching the message receipt time.
2. **Given** a device publishes a string value (e.g., "on"/"off") to an attribute topic, **When** the bridge processes the message, **Then** the string value is stored correctly and is retrievable.
3. **Given** a device publishes a boolean value, **When** the bridge processes the message, **Then** the value is stored and is distinguishable from numeric zero/one.
4. **Given** a well-formed JSON payload on an attribute topic, **When** the bridge processes the message, **Then** the value field from the JSON is extracted and stored correctly.

---

### User Story 2 - Connection Resilience (Priority: P2)

The bridge must survive temporary MQTT broker or InfluxDB unavailability without data loss and without requiring manual intervention to recover.

**Why this priority**: IoT environments are inherently unreliable. Loss of connectivity to either MQTT or InfluxDB must not silently discard device data or crash the daemon in a way that requires operator action.

**Independent Test**: Start the bridge, stop the MQTT broker for 10 seconds, restart it, publish a message, and verify that the first successfully published post-recovery message is persisted within 10 seconds of the broker becoming available. Separately, stop InfluxDB for 10 seconds, publish messages during the outage, restart InfluxDB, and verify the buffered messages are eventually persisted without restarting the process.

**Acceptance Scenarios**:

1. **Given** the MQTT broker is temporarily unavailable, **When** the broker becomes reachable again, **Then** the bridge automatically reconnects without manual intervention.
2. **Given** the InfluxDB instance is temporarily unavailable, **When** a message arrives, **Then** the bridge retries the write and persists the data once InfluxDB is reachable again.
3. **Given** the bridge is running and loses its MQTT connection, **When** reconnecting, **Then** the bridge re-subscribes to all previously subscribed topics.

---

### User Story 3 - Multi-Account and Multi-Device Isolation (Priority: P3)

The bridge handles data from multiple accounts and device types simultaneously, ensuring data from one account is stored with correct isolation identifiers so that access control enforced by the REST layer remains meaningful.

**Why this priority**: Multi-tenancy correctness is critical for the platform's viability but can be validated once basic ingestion (Story 1) works.

**Independent Test**: Publish messages for two different account_id values, then query InfluxDB and verify each record is tagged with its respective account_id.

**Acceptance Scenarios**:

1. **Given** two devices owned by different accounts publish to their respective MQTT topics simultaneously, **When** the bridge ingests both messages, **Then** each record in InfluxDB is tagged with the correct account_id and device identifiers.
2. **Given** a device publishes to multiple node/attribute combinations within the same device instance, **When** the bridge processes the messages, **Then** each attribute is stored as a distinct, independently queryable record.

---

### Edge Cases

- Messages received on topics that do not match the expected 5-segment hierarchy are silently discarded with a warning log (FR-008).
- Messages received on 6-segment `/set` topics are treated as invalid ingest input, discarded, and logged as warnings because desired-state write-back is out of scope for this bridge (FR-020).
- Empty or null message payloads are treated as unparseable and discarded with a warning log (FR-009).
- Attribute values that cannot be parsed as numeric, string, boolean, or JSON are discarded with a warning log including topic and raw payload (FR-009).
- Concurrent messages for the same account/device/device_instance/node/attribute may carry different scalar types; each message is stored as its own record with exactly one typed value field populated (FR-022).
- MQTT QoS 1 may deliver duplicates during reconnect or retry scenarios; duplicate deliveries are stored as distinct records and are not deduplicated by the bridge (FR-021).
- At startup, if InfluxDB is not yet reachable, the bridge blocks and retries with exponential backoff until InfluxDB becomes available before subscribing to MQTT (FR-014).
- On shutdown, the bridge stops accepting new MQTT messages, attempts to flush buffered data for a bounded period, and logs a warning if any buffered messages must be dropped (FR-016).
- InfluxDB schema conflicts (same attribute publishing different types) are avoided by using separate typed fields `value_float`, `value_string`, `value_bool` — only one is populated per write (FR-003).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The bridge MUST subscribe to all MQTT topics that match the device attribute hierarchy pattern `[account_id]/[device_id]/[device_instance_id]/[node_name]/[attribute_name]`.
- **FR-002**: The bridge MUST extract the account_id, device_id, device_instance_id, node_name, and attribute_name from the MQTT topic path and store them as metadata tags alongside the value.
- **FR-003**: The bridge MUST support attribute values of type number (integer and float), string, and boolean. Each type MUST be stored in a dedicated InfluxDB field: `value_float` (for numeric values), `value_string` (for string values), and `value_bool` (for boolean values). Only one field is populated per write; the others are omitted.
- **FR-004**: The bridge MUST persist each received message to InfluxDB with a timestamp reflecting when the data was received; if the JSON payload contains a `ts` field (Unix epoch, milliseconds), that value MUST be used as the timestamp instead.
- **FR-005**: The bridge MUST handle JSON payloads by extracting a `value` field (and optionally a `ts` field for timestamp override); raw scalar payloads (plain number, boolean, or string) MUST also be accepted.
- **FR-006**: The bridge MUST automatically reconnect to the MQTT broker after a connection loss without requiring a process restart.
- **FR-007**: The bridge MUST automatically retry failed InfluxDB writes with exponential backoff. The default maximum retry count is 3 (configurable via environment variable); after exhausting retries the message is discarded and a warning is logged.
- **FR-008**: The bridge MUST silently discard messages received on topics that do not match the expected hierarchy structure and log a warning.
- **FR-009**: The bridge MUST silently discard messages with unparseable payloads and log a warning including the topic and raw payload.
- **FR-010**: The bridge MUST start as a background goroutine within the stutzthings-server process (not a separate binary).
- **FR-011**: The bridge MUST be configurable via environment variables for MQTT broker address, InfluxDB v3 URL, API token, and target database/table name. The bridge MUST connect using MQTT protocol version 3.1.1. The InfluxDB connection uses plain HTTP (no TLS at the application layer); TLS MUST be enforced at the infrastructure/network layer for non-local deployments.
- **FR-012**: The bridge MUST authenticate to the MQTT broker using username and password credentials supplied via environment variables. TLS encryption for the MQTT connection MUST be optionally configurable. Anonymous (no-auth) connections MUST NOT be the default.
- **FR-013**: The bridge MUST buffer incoming messages and write to InfluxDB in batches. A batch is flushed when either the configurable maximum batch size (number of messages) or the configurable maximum flush interval is reached, whichever comes first. Default flush interval is 100ms. Both thresholds MUST be configurable via environment variables.
- **FR-014**: At startup, the bridge MUST verify that InfluxDB is reachable (with retries and exponential backoff) before subscribing to any MQTT topics. The bridge MUST NOT begin consuming messages until this readiness check passes.
- **FR-015**: The bridge MUST enforce a configurable maximum in-memory buffer size (number of messages). When the buffer is full, the oldest unwritten messages MUST be evicted (FIFO) and a warning MUST be logged per evicted batch, including the count of dropped messages.
- **FR-016**: On process shutdown, the bridge MUST stop consuming new MQTT messages, attempt to flush buffered messages for up to 5 seconds, and then exit. If buffered messages remain after the flush window, the bridge MUST discard only the remaining buffered messages and log a warning including the dropped count.
- **FR-017**: The bridge MUST expose its operational status to the host service health check at `GET /health`. The reported health MUST reflect MQTT connectivity, InfluxDB reachability, and buffer pressure using the states `OK`, `WARNING`, and `ERROR`.
- **FR-018**: The bridge MUST emit structured logs. Every bridge log record MUST include `component=bridge` and an `event` field; discard warnings MUST include `reason` and `topic`, payload-related discards MUST also include a bounded `payload_sample`, retry warnings MUST include `attempt` and `max_retries`, and eviction warnings MUST include `dropped_count`.
- **FR-019**: After exhausting write retries for a batch, the bridge MUST discard only that failed batch, log the warning required by FR-007 and FR-018, and continue accepting and processing subsequent messages without entering a circuit-breaker or stopped state.
- **FR-020**: MQTT topics matching the 6-segment desired-state pattern `[account_id]/[device_id]/[device_instance_id]/[node_name]/[attribute_name]/set` MUST be rejected by the bridge as invalid ingest input and handled exactly as malformed topics under FR-008.
- **FR-021**: MQTT QoS 1 duplicate deliveries are allowed; the bridge MUST NOT attempt deduplication and MUST persist each delivered message independently.
- **FR-022**: If multiple messages for the same identity path arrive with different scalar value types, the bridge MUST persist each message as a separate record with exactly one of `value_float`, `value_string`, or `value_bool` populated.
- **FR-023**: With the default `BRIDGE_MAX_BUFFER_SIZE=10000` and payloads up to 1 KiB each, bridge-owned in-memory buffering MUST remain below 32 MiB. Deployments expecting larger payloads MUST reduce `BRIDGE_MAX_BUFFER_SIZE` accordingly.

### Key Entities

- **DeviceAttribute**: A single time-stamped measurement produced by a device. Identified by account_id + device_id + device_instance_id + node_name + attribute_name. Stored in InfluxDB v3 under the measurement name `device_attributes`, with account_id, device_id, device_instance_id, node_name, and attribute_name as tags. The value is stored in exactly one of three typed fields: `value_float` (numeric), `value_string` (string), or `value_bool` (boolean).
- **MQTTMessage**: An incoming raw message from the broker with a topic string and byte payload.
- **BridgeConfig**: Operational parameters — MQTT broker URL, MQTT username/password, optional MQTT TLS flag, InfluxDB v3 URL, InfluxDB API token, target database/table name, write batch size (max messages per batch, default: 100), batch flush interval (default: 100ms), max in-memory buffer size (default: 10,000 messages), retry count (default: 3).

### Assumptions

- MQTT topics for device attribute reporting follow the exact 5-segment hierarchy: `account_id/device_id/device_instance_id/node_name/attribute_name`. Topics with more or fewer segments are ignored.
- Wildcard MQTT subscription (`#`) is used to receive all topics; filtering to the expected structure is done by the bridge itself.
- InfluxDB v3 is already provisioned with a database; the bridge does not manage database or table creation. This is a deployment prerequisite.
- The InfluxDB connection uses plain HTTP. **Security note**: for production environments, TLS MUST be enforced at the network layer (e.g., service mesh, reverse proxy, or VPN) to protect the API token and data in transit.
- The bridge connects to the MQTT broker using protocol version 3.1.1.
- Message QoS level is assumed to be at least QoS 1 (at-least-once) to prevent silent data loss during transient disconnects. Duplicate deliveries are therefore possible and are intentionally not deduplicated by the bridge.
- The bridge does not need to handle the `/set` topic direction (desired state write-back) — that is handled by the REST API PUT endpoint.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A sensor value published to an MQTT attribute topic appears in InfluxDB within 2 seconds under normal operating conditions, measured from the instant the publisher receives a successful MQTT publish acknowledgment to the instant the record becomes queryable from InfluxDB.
- **SC-002**: After an MQTT broker outage of up to 60 seconds, the bridge reconnects automatically and resumes ingestion within 10 seconds of the broker becoming available — with zero operator intervention.
- **SC-003**: The bridge sustains ingestion of at least 500 attribute messages per second from multiple concurrent devices for a continuous 60-second healthy-dependency load test with 0 dropped messages and 0 discard/eviction warning logs.
- **SC-004**: Zero data is silently dropped — every discarded message (malformed topic, unparseable payload, persistent write failure) appears as a log warning that is observable.
- **SC-005**: All stored records in InfluxDB are correctly tagged with the full device identity hierarchy, enabling account-level access control enforcement at the query layer with 100% accuracy.

## Clarifications

### Session 2026-03-14

- Q: How should the bridge authenticate to the MQTT broker? → A: Username + Password (plaintext credentials, optionally over TLS)
- Q: What should the InfluxDB measurement name be for stored device attribute records? → A: `device_attributes` (single measurement, all attributes differentiated by tags)
- Q: What JSON field name carries the embedded timestamp in the payload? → A: `ts`
- Q: What is the default maximum InfluxDB write retry count before discarding a message? → A: 3
- Q: Which MQTT protocol version should the bridge use? → A: MQTT 3.1.1 (downgraded from v5 for better client library reliability)

### Session 2026-03-14 (continued)

- Q: Which InfluxDB API version should the bridge target? → A: InfluxDB v3
- Q: How should typed field storage be handled to avoid InfluxDB v3 schema conflicts? → A: Separate typed fields — `value_float`, `value_string`, `value_bool`; only one populated per write
- Q: Should the bridge connect to InfluxDB v3 over TLS? → A: Plain HTTP only (no TLS); assumed local/dev environment — production deployments must add TLS at the network layer

- Q: Should the bridge write each message individually or batch writes to InfluxDB? → A: Batch writes — buffer up to N messages or flush every T milliseconds, whichever comes first
- Q: How does the bridge behave during startup if InfluxDB is not yet reachable? → A: Block startup — wait with retries until InfluxDB is reachable before subscribing to MQTT topics

### Session 2026-03-14 (continued 2)

- Q: What should the default batch flush interval be? → A: 100ms (aggressive, near real-time)
- Q: What happens when the in-memory buffer exceeds its ceiling during an InfluxDB outage? → A: Drop oldest messages (FIFO eviction) and log a warning per dropped batch
- Q: Should the bridge use any MQTT 5-specific features? → A: N/A — decision superseded; bridge uses MQTT 3.1.1

## XDR Candidates *(filled by speckit.specify, realised by speckit.plan)*

- [BDR] MQTT topic structure and payload schema — 5-segment hierarchy and scalar/JSON payload formats; `/set` suffix for desired-state write-back. (exists: [.xdrs/_local/bdrs/product/001-mqtt-topic-structure.md](../../.xdrs/_local/bdrs/product/001-mqtt-topic-structure.md))
- [BDR] InfluxDB data model for device attributes — measurement name, tag set (account, device, instance, node, attribute), field layout, and timestamp policy.
- [ADR] Bridge process topology — goroutine-per-concern design embedded in stutzthings-server vs. standalone binary; rationale for in-process choice.
- [EDR] MQTT reconnect and InfluxDB retry policy — backoff strategy, maximum retry limits, re-subscription semantics on reconnect.
