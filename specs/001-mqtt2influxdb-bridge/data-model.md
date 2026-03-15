# Data Model: MQTT to InfluxDB Bridge Daemon

**Feature**: `001-mqtt2influxdb-bridge`  
**Date**: 2026-03-14

---

## Entities

### 1. BridgeConfig

Operational configuration loaded from environment variables at startup. Immutable after initialization.

| Field | Type | Env Var | Default | Description |
|-------|------|---------|---------|-------------|
| MQTTBrokerURL | string | `MQTT_BROKER_URL` | — (required) | MQTT broker address, e.g. `mqtt://broker:1883` |
| MQTTUsername | string | `MQTT_USERNAME` | — (required) | MQTT auth username |
| MQTTPassword | string | `MQTT_PASSWORD` | — (required) | MQTT auth password |
| MQTTTLSEnabled | bool | `MQTT_TLS_ENABLED` | `false` | Enable TLS for MQTT connection |
| InfluxDBURL | string | `INFLUXDB_URL` | — (required) | InfluxDB v3 base URL, e.g. `http://influxdb:8086` |
| InfluxDBToken | string | `INFLUXDB_TOKEN` | — (required) | InfluxDB API token |
| InfluxDBDatabase | string | `INFLUXDB_DATABASE` | — (required) | InfluxDB v3 database name |
| BatchSize | int | `BRIDGE_BATCH_SIZE` | `100` | Max messages per InfluxDB write batch |
| FlushIntervalMs | int | `BRIDGE_FLUSH_INTERVAL_MS` | `100` | Max milliseconds between batch flushes |
| MaxBufferSize | int | `BRIDGE_MAX_BUFFER_SIZE` | `10000` | Max in-memory messages before FIFO eviction |
| MaxWriteRetries | int | `BRIDGE_MAX_WRITE_RETRIES` | `3` | Max InfluxDB write retries before discarding |

**Validation rules**:
- All required fields must be non-empty; startup fails with a descriptive error if missing.
- `BatchSize` must be ≥ 1.
- `FlushIntervalMs` must be ≥ 10.
- `MaxBufferSize` must be ≥ `BatchSize`.
- `MaxWriteRetries` must be ≥ 0.
- At the default `MaxBufferSize = 10000`, deployments should keep payloads at or below 1 KiB each to stay within the 32 MiB bridge-owned buffer memory budget; larger payloads require lowering `MaxBufferSize`.

---

### 2. MQTTMessage

Internal representation of a raw message received from the broker. Ephemeral — created on ingest, discarded after parsing.

| Field | Type | Description |
|-------|------|-------------|
| Topic | string | Full MQTT topic string as received (e.g. `acct1/sensor/dev01/temperature/celsius`) |
| Payload | []byte | Raw message payload bytes |
| ReceivedAt | time.Time | Timestamp set by the bridge at message receipt (wall clock) |

---

### 3. DeviceAttribute

A parsed, typed measurement ready for InfluxDB persistence. Produced from a valid `MQTTMessage`.

| Field | Type | InfluxDB Role | Description |
|-------|------|--------------|-------------|
| AccountID | string | Tag | Extracted from topic segment 1 |
| DeviceID | string | Tag | Extracted from topic segment 2 |
| DeviceInstanceID | string | Tag | Extracted from topic segment 3 |
| NodeName | string | Tag | Extracted from topic segment 4 |
| AttributeName | string | Tag | Extracted from topic segment 5 |
| ValueFloat | *float64 | Field `value_float` | Populated for numeric values; nil otherwise |
| ValueString | *string | Field `value_string` | Populated for string values; nil otherwise |
| ValueBool | *bool | Field `value_bool` | Populated for boolean values; nil otherwise |
| Timestamp | time.Time | InfluxDB timestamp | From `ts` field in JSON payload (Unix epoch ms); falls back to `MQTTMessage.ReceivedAt` |

**Invariants**:
- Exactly one of `ValueFloat`, `ValueString`, `ValueBool` is non-nil per record.
- `Timestamp` is always set (never zero).

**InfluxDB mapping**:
- Measurement: `device_attributes`
- Tags: `account_id`, `device_id`, `device_instance_id`, `node_name`, `attribute_name`
- Fields: `value_float` (float64), `value_string` (string), `value_bool` (bool)
- Timestamp precision: milliseconds

---

### 4. BridgeHealthStatus

Operational health state exported by the bridge to the host service `/health` endpoint.

| Field | Type | Description |
|-------|------|-------------|
| Health | string | One of `OK`, `WARNING`, `ERROR` |
| LatencyMs | int64 | Total dependency-check latency in milliseconds |
| Message | string | Human-readable summary suitable for `/health` output |
| MQTTConnected | bool | Whether the MQTT client is currently connected |
| InfluxReachable | bool | Whether the latest read-only InfluxDB probe succeeded |
| BufferUsage | int | Current number of buffered messages |

**Health rules**:
- `OK`: MQTT connected, Influx reachable, buffer usage below eviction threshold.
- `WARNING`: bridge is still operating but dependency latency is elevated, MQTT is reconnecting, or buffer usage is elevated without active eviction.
- `ERROR`: Influx is unreachable for health checks, the bridge cannot accept work, or the host process has marked the bridge unhealthy.

---

## Payload Parsing Rules

### Accepted payload formats

**Format 1 — Raw scalar** (entire payload is the value, UTF-8 encoded):

| Detected as | Examples | Mapped to |
|-------------|----------|-----------|
| Boolean | `true`, `false` | `ValueBool` |
| Integer | `42`, `-7` | `ValueFloat` (stored as float64) |
| Float | `23.5`, `-0.1` | `ValueFloat` |
| String | `on`, `off`, any other text | `ValueString` |

**Detection order**: Boolean → Integer → Float → String (string is the catch-all). This means raw scalar `true` is always parsed as boolean, raw scalar `1` is parsed as numeric, and quoted JSON strings such as `{ "value": "1" }` remain strings.

**Format 2 — JSON object**:

```json
{ "value": <scalar> }
{ "value": <scalar>, "ts": <unix_epoch_ms> }
```

- `value` field: required; same type detection rules as Format 1.
- `ts` field: optional; integer, Unix epoch in **milliseconds**. Overrides `ReceivedAt` as timestamp.
- Extra fields in the JSON object are silently ignored.

### Discard conditions

| Condition | Action |
|-----------|--------|
| Topic has ≠ 5 segments | Discard + log WARNING with topic |
| Topic matches 6-segment `/set` desired-state pattern | Discard + log WARNING with topic and reason=`set_topic_ignored` |
| Payload is empty or null | Discard + log WARNING with topic and raw payload |
| JSON payload missing `value` field | Discard + log WARNING with topic and raw payload |
| `value` is JSON null | Discard + log WARNING |

### Logging fields

All bridge log records are structured and include:

| Field | Required | Notes |
|-------|----------|-------|
| `component` | yes | Always `bridge` |
| `event` | yes | Examples: `message_discarded`, `influx_retry`, `buffer_evicted`, `shutdown_drop` |
| `topic` | for message-related events | Full MQTT topic |
| `reason` | for warnings/discards | Machine-readable reason code |
| `payload_sample` | for payload parse failures | Bounded sample, not full payload dump |
| `attempt` | for retry logs | 1-based retry attempt |
| `max_retries` | for retry logs | Configured retry ceiling |
| `dropped_count` | for eviction/shutdown drops | Number of messages discarded |

---

## State Transitions

```
MQTT message received
        │
        ▼
  [Topic validation]
  5 segments? ──No──► Discard + WARN
        │
       Yes
        ▼
  [Payload parse]
  Parseable? ─────No──► Discard + WARN
        │
       Yes
        ▼
  [Ring buffer]
  Buffer full? ──Yes──► Evict oldest batch + WARN, then enqueue
        │
       No (space available)
        ▼
  [Batch accumulator]
  Size >= BatchSize OR timer >= FlushIntervalMs
        │
        ▼
  [InfluxDB writer]
  Write attempt ──Fail──► Retry w/ backoff (up to MaxWriteRetries)
        │                         │
      Success              Max retries exhausted
        │                         │
      Done              Discard batch + WARN + continue consuming

Shutdown requested
        │
        ▼
  Stop MQTT intake
        │
        ▼
  Flush buffer for up to 5s
        │
   Buffer empty? ──Yes──► Exit cleanly
        │
       No
        ▼
  Discard remaining buffered messages + WARN + Exit
```
