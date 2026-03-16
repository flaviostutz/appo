# Quickstart: Device Operations API and MCP Access

This quickstart describes the intended end-to-end verification flow for the feature after implementation lands.

## Prerequisites

- `mise`
- Docker Desktop or compatible `docker compose`
- A local JWT signing secret for development

## Start the local stack

```sh
cd stutzthings/stutzthings-server
make install
export JWT_SIGNING_SECRET=dev_jwt_secret_local_only_please_change
make run
```

Expected result:
- Mosquitto and InfluxDB start locally
- `stutzthings-server` starts on `:8080`
- `GET /health` returns `200` with bridge and API dependencies healthy

## Verify bridge ingestion prerequisites

If the local InfluxDB database is not already present, create it once:

```sh
cd stutzthings/stutzthings-server/examples/local
make create-database
```

## Exercise the REST API

Use a valid registration bearer token with scope `r:demo-account/demo-device`:

```sh
curl -X POST http://127.0.0.1:8080/registration \
  -H "Authorization: Bearer <registration-token>" \
  -H "Content-Type: application/json" \
  -d '{"accountId":"demo-account","deviceId":"demo-device"}'
```

Publish device telemetry to the broker, then read it back:

```sh
curl -H "Authorization: Bearer <device-read-token>" \
  "http://127.0.0.1:8080/demo-account/demo-device/<device-instance-id>/temperature/celsius"

curl -H "Authorization: Bearer <device-read-token>" \
  "http://127.0.0.1:8080/demo-account/demo-device/<device-instance-id>/temperature/celsius/history?from=2026-03-17T00:00:00Z&to=2026-03-17T01:00:00Z"
```

Publish a desired-state command:

```sh
curl -X PUT http://127.0.0.1:8080/demo-account/demo-device/<device-instance-id>/lamp/power \
  -H "Authorization: Bearer <device-set-token>" \
  -H "Content-Type: application/json" \
  -d '{"value":"on"}'
```

## Exercise the MCP endpoint

Connect an MCP client or inspector to `http://127.0.0.1:8080/mcp` and call:

- `get_attribute_state`
- `get_device_state`
- `get_node_history`
- `get_attribute_history`
- `set_desired_state`
- `register_device_instance`

REST and MCP results must match for the same authorization context.

## Verification commands

```sh
cd stutzthings/stutzthings-server
make test
make lint
```

For richer local observability, use the Grafana-based stack in `examples/local/`.