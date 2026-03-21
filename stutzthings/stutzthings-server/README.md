# stutzthings-server

Embedded Go server for the MQTT-to-InfluxDB bridge plus the device-operations REST and MCP surfaces.

## Getting Started

```sh
make install
export JWT_SIGNING_SECRET_BASE64="$(printf %s 'dev_jwt_secret_local_only_please_change' | base64)"
export MOSQUITTO_JWT_AUTH_SO=/absolute/path/to/libmosquitto_jwt_auth.so
make build
make run
```

If Docker cannot pull `eclipse-mosquitto:2` or `influxdb:3-core` on your machine, `make run` stops with a short diagnostic and points to the host-based fallback in `examples/local/`.

`make run` now expects the Mosquitto JWT auth plugin shared library path in `MOSQUITTO_JWT_AUTH_SO` and uses the same `JWT_SIGNING_SECRET_BASE64` value for both the Go service and Mosquitto.

`make dev` downloads the published Mosquitto JWT auth plugin into `./.docker/` when needed, replaces corrupt zero-byte artifacts automatically, and runs the Mosquitto container as `linux/amd64` so the released `x86-64` plugin works on Apple Silicon hosts.

```sh
curl -i http://localhost:8080/health
```

The bridge loads its runtime settings from environment variables and exposes dependency-aware health at `GET /health`. The same process is also the planned home for protected REST routes and the `/mcp` endpoint.

By default the bridge loads runtime settings from `.stutzthingsrc` in the current working directory. If that file is absent, it falls back to environment variables. Set `BRIDGE_CONFIG_PATH` to load a different JSON file.

Example `.stutzthingsrc`:

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
	"maxWriteRetries": 3,
	"httpAddr": ":8080",
	"jwtSigningSecretBase64": "ZGV2X2p3dF9zZWNyZXRfbG9jYWxfb25seV9wbGVhc2VfY2hhbmdl",
	"jwtIssuer": "stutzthings-server-dev",
	"jwtTokenTtl": "24h"
}
```

## Configuration

`.stutzthingsrc` required fields:

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
- `httpAddr` default `:8080`
- `jwtSigningSecretBase64` default base64-encoded local development secret
- `jwtIssuer` default `stutzthings-server-dev`
- `jwtTokenTtl` default `24h`

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
- `JWT_SIGNING_SECRET_BASE64` shared base64-encoded HS256 secret for the Go service and Mosquitto JWT plugin
- `JWT_ISSUER` token issuer string for registration-issued credentials
- `JWT_TOKEN_TTL` default registration-token lifetime, default `24h`
- `MOSQUITTO_JWT_AUTH_SO` absolute path to `libmosquitto_jwt_auth.so` used by `make run`

## Commands

- `make build` builds the binary into `dist/`
- `make lint` validates `gofmt`, runs zero-warning GolangCI-Lint checks, and executes the vulnerability audit using `VULN_FAIL_LEVEL` (default: `critical`)
- `make lint-style` runs GolangCI-Lint only, and `make lint-audit` runs the vulnerability audit only
- `make test` runs unit and integration tests, then enforces the unit coverage threshold and prints the coverage report only when the threshold fails
- `make coverage` generates the unit coverage report without enforcing the threshold
- `make health` checks `GET /health` on the locally running server
- `make run` verifies Docker access, requires the Mosquitto JWT plugin path, starts the local Docker stack, and runs the server
- `make dev` refreshes the local Mosquitto JWT plugin if needed, starts the Docker stack with Mosquitto pinned to `linux/amd64`, and then runs the server

## Local Example Stack

For a local development flow with containerized infrastructure, use `examples/local/README.md`. It includes a dedicated `docker-compose.yml`, `Makefile`, `.stutzthingsrc`, an InfluxDB admin token file, Grafana provisioning, and Mosquitto JWT plugin wiring for running Mosquitto, InfluxDB, and Grafana in Docker while keeping the Go bridge on the host machine.

## Layout

- `main.go` wires the bridge into the HTTP server that hosts `/health`, REST routes, and MCP
- `bridge/` contains configuration, MQTT, buffering, payload parsing, and InfluxDB writing
- `docker-compose.yml` starts a local Mosquitto broker with `wiomoc/mosquitto-jwt-auth` and InfluxDB 3 Core
- `examples/local/` contains a compose-managed local stack for Mosquitto, InfluxDB, and Grafana plus a host-run bridge

Before running the bridge against the local InfluxDB instance, create the target database manually, for example:

```sh
docker exec stutzthings-influxdb3 sh -lc 'export INFLUXDB3_AUTH_TOKEN=apiv3_dev_token_local_only && influxdb3 create database iot'
```

