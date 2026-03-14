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

## Payload Parsing Rules

### Accepted payload formats

**Format 1 — Raw scalar** (entire payload is the value, UTF-8 encoded):

| Detected as | Examples | Mapped to |
|-------------|----------|-----------|
| Boolean | `true`, `false` | `ValueBool` |
| Integer | `42`, `-7` | `ValueFloat` (stored as float64) |
| Float | `23.5`, `-0.1` | `ValueFloat` |
| String | `on`, `off`, any other text | `ValueString` |

**Detection order**: Boolean → Integer → Float → String (string is the catch-all).

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
| Payload is empty or null | Discard + log WARNING with topic and raw payload |
| JSON payload missing `value` field | Discard + log WARNING with topic and raw payload |
| `value` is JSON null | Discard + log WARNING |

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
      Done              Discard batch + WARN
```
