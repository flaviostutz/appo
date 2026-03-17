# _local BDRs Index

Project-local business decisions for this repository. These decisions override or extend decisions from all higher-positioned scopes in `.xdrs/index.md`.

Feature `002-device-ops-api` relies on `_local-bdr-003` for unified JWT ACL semantics and `_local-bdr-004` for registration-token issuance policy.

## product

| ID | Title | File |
|----|-------|------|
| _local-bdr-001 | MQTT Topic Structure for Device Attribute Reporting | [product/001-mqtt-topic-structure.md](product/001-mqtt-topic-structure.md) |
| _local-bdr-002 | InfluxDB Data Model for Device Attributes | [product/002-influxdb-device-attributes-data-model.md](product/002-influxdb-device-attributes-data-model.md) |
| _local-bdr-003 | JWT Topic ACL Claims for MQTT, REST, and MCP | [product/003-device-authorization-scope-grammar.md](product/003-device-authorization-scope-grammar.md) |
| _local-bdr-004 | Stateless Device Registration Policy | [product/004-stateless-device-registration-policy.md](product/004-stateless-device-registration-policy.md) |
