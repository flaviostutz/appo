# _local-bdr-002: InfluxDB Data Model for Device Attributes

## Context and Problem Statement

Device attribute readings ingested via MQTT must be persisted to InfluxDB v3 in a schema that supports:
- Per-account access control enforcement at query time
- Heterogeneous value types (numeric, string, boolean) without schema conflicts
- Efficient tag-based filtering across account, device, and attribute dimensions
- Accurate timestamping from either the message payload or message receipt time

Question: What measurement name, tag set, field layout, and timestamp policy should the bridge use when writing device attribute data to InfluxDB v3?

## Decision Outcome

**Single measurement `device_attributes` with a 5-tag identity hierarchy and 3 typed value fields; timestamp from `ts` payload field (ms) or receipt time fallback**

### Implementation Details

#### Measurement

- Measurement name: `device_attributes` (fixed, not configurable)

#### Tag set

All 5 identity segments extracted from the MQTT topic path are stored as tags:

| Tag | Source | Example |
|-----|--------|---------|
| `account_id` | Topic segment 1 | `account1` |
| `device_id` | Topic segment 2 | `sensor` |
| `device_instance_id` | Topic segment 3 | `device01` |
| `node_name` | Topic segment 4 | `temperature` |
| `attribute_name` | Topic segment 5 | `celsius` |

Tags enable account-level access control enforcement at the query layer (SC-005).

#### Field layout

Three typed fields; exactly one is populated per write (the others are omitted):

| Field | Type | Condition |
|-------|------|-----------|
| `value_float` | float64 | Payload value is numeric (integer or float) |
| `value_string` | string | Payload value is a string |
| `value_bool` | bool | Payload value is a boolean (`true`/`false`) |

Using per-type fields avoids InfluxDB v3 type-conflict errors that would occur if a single `value` field received mixed types across attribute records.

#### Timestamp policy

- If the JSON payload contains a `ts` field (integer, Unix epoch in **milliseconds**), that value is used as the record timestamp.
- Otherwise, the bridge's wall-clock time at message receipt is used.
- Timestamp precision stored in InfluxDB: milliseconds.

#### Batch writes

The bridge accumulates points into batches before writing. A batch is flushed when:
- The batch size reaches `BRIDGE_BATCH_SIZE` (default: 100 messages), OR
- `BRIDGE_FLUSH_INTERVAL_MS` milliseconds elapse since the last flush (default: 100ms)

whichever occurs first.

## Considered Options

- (CHOSEN) **Per-type fields** (`value_float`, `value_string`, `value_bool`)
  - Reason: Eliminates type-conflict writes; follows InfluxDB best-practice for heterogeneous telemetry; explicit and queryable per type.
- (REJECTED) **Single `value` field coerced to string**
  - Reason: Loses numeric precision; requires query-time casting; complicates aggregation queries.
- (REJECTED) **Per-measurement-per-type strategy** (e.g., `device_attributes_float`)
  - Reason: Splits attribute data across measurements; complicates cross-type queries and access control.

## References

- Feature spec: [specs/001-mqtt2influxdb-bridge/spec.md](../../../../specs/001-mqtt2influxdb-bridge/spec.md)
- Feature plan: [specs/001-mqtt2influxdb-bridge/plan.md](../../../../specs/001-mqtt2influxdb-bridge/plan.md)
- Related: [bdrs/product/001-mqtt-topic-structure.md](001-mqtt-topic-structure.md)
