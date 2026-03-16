# stutzthings-server

Embedded Go server for the MQTT-to-InfluxDB bridge and its health endpoint.

## Getting Started

```sh
make install
docker compose up -d
make build
./dist/stutzthings-server
```

If Docker cannot pull `eclipse-mosquitto:2` or `influxdb:3-core` on your machine, `make run` now stops with a short diagnostic and points to the host-based fallback in `examples/local/`.

```sh
curl -i http://localhost:8080/health
```

The bridge loads its runtime settings from environment variables and exposes dependency-aware health at `GET /health`.

By default the bridge loads runtime settings from `bridge.json` in the current working directory. If that file is absent, it falls back to environment variables. Set `BRIDGE_CONFIG_PATH` to load a different JSON file.

Example `bridge.json`:

```json
{
	"mqttBrokerUrl": "mqtt://localhost:1883",
	"mqttUsername": "test",
	"mqttPassword": "test",
	"mqttTlsEnabled": false,
	"influxDbUrl": "http://localhost:8086",
	"influxDbToken": "apiv3_dev_token_local_only",
	"influxDbDatabase": "iot",
	"batchSize": 100,
	"flushIntervalMs": 100,
	"maxBufferSize": 10000,
	"maxWriteRetries": 3
}
```

## Configuration

`bridge.json` required fields:

- `mqttBrokerUrl`
- `mqttUsername`
- `mqttPassword`
- `influxDbUrl`
- `influxDbToken`
- `influxDbDatabase`

Optional fields:

- `mqttTlsEnabled` default `false`
- `batchSize` default `100`
- `flushIntervalMs` default `100`
- `maxBufferSize` default `10000`
- `maxWriteRetries` default `3`

## Environment Fallback

Required variables:

- `MQTT_BROKER_URL`
- `MQTT_USERNAME`
- `MQTT_PASSWORD`
- `INFLUXDB_URL`
- `INFLUXDB_TOKEN`
- `INFLUXDB_DATABASE`

Optional variables:

- `MQTT_TLS_ENABLED` default `false`
- `BRIDGE_BATCH_SIZE` default `100`
- `BRIDGE_FLUSH_INTERVAL_MS` default `100`
- `BRIDGE_MAX_BUFFER_SIZE` default `10000`
- `BRIDGE_MAX_WRITE_RETRIES` default `3`
- `HTTP_ADDR` default `:8080`
- `LOG_LEVEL` default `info`

## Commands

- `make build` builds the binary into `dist/`
- `make lint` runs zero-warning GolangCI-Lint checks
- `make test` runs unit and integration tests
- `make run` verifies Docker access, starts the local Docker stack, and runs the server

## Local Example Stack

For a local development flow with containerized infrastructure, use `examples/local/README.md`. It includes a dedicated `docker-compose.yml`, `Makefile`, `bridge.json`, an InfluxDB admin token file, and Grafana provisioning for running Mosquitto, InfluxDB, and Grafana in Docker while keeping the Go bridge on the host machine.

## Layout

- `main.go` wires the bridge into an HTTP `/health` endpoint
- `bridge/` contains configuration, MQTT, buffering, payload parsing, and InfluxDB writing
- `docker-compose.yml` starts a local Mosquitto broker and InfluxDB 3 Core
- `examples/local/` contains a compose-managed local stack for Mosquitto, InfluxDB, and Grafana plus a host-run bridge

Before running the bridge against the local InfluxDB instance, create the target database manually, for example:

```sh
docker exec stutzthings-influxdb3 sh -lc 'export INFLUXDB3_AUTH_TOKEN=apiv3_dev_token_local_only && influxdb3 create database iot'
```

