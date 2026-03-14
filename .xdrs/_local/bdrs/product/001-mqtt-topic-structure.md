# _local-bdr-001: MQTT Topic Structure for Device Attribute Reporting

## Context and Problem Statement

IoT devices must publish sensor readings and attribute values to the platform over MQTT. Without a defined topic convention, the platform cannot reliably route, store, or enforce access control on device data.

Question: What MQTT topic structure and payload schema must IoT devices use when publishing attribute readings to the platform, and how does the platform communicate desired state back to devices?

## Decision Outcome

**5-segment hierarchical topic path with scalar or JSON payload; `/set` suffix for desired-state write-back**

Device attribute readings must be published to topics structured as:
`[account_id]/[device_id]/[device_instance_id]/[node_name]/[attribute_name]`

The platform writes desired state back to devices via:
`[account_id]/[device_id]/[device_instance_id]/[node_name]/[attribute_name]/set`

### Implementation Details

#### Topic structure

- Topic MUST follow exactly 5 `/`-separated segments for telemetry. Messages with fewer or more segments must be silently discarded and a warning logged.
- **Segment definitions:**
  - `account_id`: Unique identifier for the account (tenant) that owns the device.
  - `device_id`: Identifies the device model or type.
  - `device_instance_id`: Identifies the specific physical device instance.
  - `node_name`: Logical grouping of attributes within a device (e.g., a sensor cluster or sub-component).
  - `attribute_name`: The specific attribute being reported (e.g., `temperature`, `humidity`, `battery_level`).
- The platform subscribes via a wildcard (`#`) and enforces structure validation internally.
- All 5 segments must be stored as metadata tags alongside the value, enabling account-level access control at the query layer.

#### Payload schema (telemetry, device → platform)

Two payload formats are accepted; both MUST be supported:

1. **Raw scalar** — the entire payload is the value, encoded as UTF-8 text:
   - Number: `23.5` or `42`
   - Boolean: `true` or `false`
   - String: `on` (any other UTF-8 text)

2. **JSON object** — a JSON object with at least a `value` key:
   ```json
   { "value": 23.5 }
   { "value": "on", "ts": 1741910400000 }
   ```
   - The `value` field MUST be present; the `ts` field is optional (Unix epoch, **milliseconds**). When `ts` is absent, the message receipt time is used.
   - Extra fields in the JSON object SHOULD be ignored.

Messages with an empty, null, or unparseable payload must be silently discarded and a warning logged including the topic and raw payload.

#### Set mechanism (desired state, platform → device)

- The platform publishes the desired value for an attribute to the 6-segment `/set` topic:
  `[account_id]/[device_id]/[device_instance_id]/[node_name]/[attribute_name]/set`
- The payload MUST follow the same scalar or JSON schema defined above.
- Devices MUST subscribe to their own `/set` topics and act on received values.

## Considered Options

* (CHOSEN) **5-segment hierarchy** — Account / Device / Instance / Node / Attribute
  * Reason: Mirrors the platform identity hierarchy; aligns with access control boundaries; enables fine-grained per-attribute filtering; follows widespread IoT platform conventions.
* (REJECTED) **3-segment shorthand** — Account / Device / Attribute
  * Reason: Loses device-instance disambiguation required for multi-device deployments and collapses node-level grouping.

## References

- Feature spec: [specs/001-mqtt2influxdb-bridge/spec.md](../../../../specs/001-mqtt2influxdb-bridge/spec.md)
