# _local-bdr-003: JWT Topic ACL Claims for MQTT, REST, and MCP

## Context and Problem Statement

The device-operations feature introduces protected reads, history queries, desired-state writes, MQTT broker authorization, and protected registration. The platform needs one durable JWT authorization model that works unchanged in Mosquitto, REST, and MCP.

Question: What JWT claims must the platform support across MQTT, REST, and MCP, and how are topic-filter matches evaluated?

## Decision Outcome

**Use Mosquitto-plugin-compatible JWT claims `sub`, `publ`, and `subs`; derive REST and MCP authorization from MQTT topic filters**

### Implementation Details

#### JWT claims

- `sub` identifies the principal. When the JWT is used with MQTT, the MQTT username MUST equal `sub`.
- `publ` contains MQTT topic filters the principal may publish to.
- `subs` contains MQTT topic filters the principal may subscribe to.
- JWTs may also include standard claims such as `iat` and `exp`.

#### Topic-filter rules

- `publ` and `subs` use standard MQTT topic-filter semantics, including `+` and `#`.
- Filters are evaluated against the canonical topic hierarchy:
  - telemetry: `account_id/device_id/device_instance_id/node_name/attribute_name`
  - command: `account_id/device_id/device_instance_id/node_name/attribute_name/set`
- Matching is filter-based; substring or ad-hoc prefix matching outside MQTT semantics is forbidden.

#### Cross-surface authorization rules

- MQTT publication is authorized by `publ` filters.
- MQTT subscription is authorized by `subs` filters.
- REST and MCP current-state/history reads are authorized when the addressed telemetry topic matches at least one filter in `publ` or `subs`.
- REST and MCP desired-state writes are authorized when the addressed `/set` topic matches at least one filter in `subs`.
- Registration is authorized only when wildcard `publ` and `subs` filters cover the requested account/device namespace.
- Authorization checks MUST always include the `account_id` segment so tenant isolation is preserved.
- If the JWT does not authorize the requested path, the operation is rejected with no data leakage.
- Whole-device reads may return only the authorized attributes; when any stored attributes are excluded, the result is marked partial.

## Considered Options

- (CHOSEN) **Mosquitto-plugin-compatible JWT claims reused across MQTT, REST, and MCP**
  - Reason: Eliminates transport-specific auth drift and makes registration-issued credentials immediately useful everywhere.
- (REJECTED) **Custom `i:` and `r:` scope families**
  - Reason: Duplicates broker authorization semantics and creates two sources of truth.
- (REJECTED) **Separate JWT models for MQTT and HTTP transports**
  - Reason: Makes parity and registration much harder.

## References

- Feature spec: [specs/002-device-ops-api/spec.md](../../../../specs/002-device-ops-api/spec.md)
- Feature plan: [specs/002-device-ops-api/plan.md](../../../../specs/002-device-ops-api/plan.md)
- Related: [product/001-mqtt-topic-structure.md](001-mqtt-topic-structure.md)
- Related: [product/002-influxdb-device-attributes-data-model.md](002-influxdb-device-attributes-data-model.md)
