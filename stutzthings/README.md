# stutzthings

IoT middleware layer (Python library + Docker services).

Responsibilities:
- Expose an MCP (Model Context Protocol) server so agents can discover devices, query current sensor state, send actuator commands, and read historical data (InfluxDB)
- Define a standard MQTT topic structure used by all devices and agents
- Provide a Python SDK for device-side applications to publish state and subscribe to commands

## Architecture overview

One sub-directory per component (e.g. MCP server, Python SDK, MQTT bridge). Components share a common MQTT topic schema and data model defined in `shared/`.

## How to build

```bash
make build
```

## How to run

```bash
docker compose up
```

> Full instructions live inside each component sub-directory once modules are added.
