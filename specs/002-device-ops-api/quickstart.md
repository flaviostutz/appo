# Quickstart: Device Operations API and MCP Access

This quickstart describes the intended end-to-end verification flow for the feature after implementation lands.

## Prerequisites

- `mise`
- Docker Desktop or compatible `docker compose`
- A built `wiomoc/mosquitto-jwt-auth` plugin shared library
- `mosquitto_pub` and `mosquitto_sub` for MQTT verification
- A base64-encoded JWT signing secret shared by the Go service and Mosquitto

## Start the local stack

```sh
cd stutzthings/stutzthings-server
make install
export JWT_SIGNING_SECRET_BASE64="$(printf %s 'dev_jwt_secret_local_only_please_change' | base64)"
export MOSQUITTO_JWT_AUTH_SO=/absolute/path/to/libmosquitto_jwt_auth.so
make run
```

Expected result:
- Mosquitto starts locally with `wiomoc/mosquitto-jwt-auth`
- InfluxDB starts locally
- `stutzthings-server` starts on `:8080`
- `GET /health` returns `200` with bridge, API, and broker dependencies healthy

## Verify bridge ingestion prerequisites

If the local InfluxDB database is not already present, create it once:

```sh
cd stutzthings/stutzthings-server/examples/local
make create-database
```

## Register a device instance

Use a registration JWT whose wildcard topic filters cover `demo-account/demo-device`, for example:

- `sub=demo-admin`
- `publ=["demo-account/demo-device/+/+/+"]`
- `subs=["demo-account/demo-device/+/+/+/set"]`

Request a new device credential:

```sh
curl -X POST http://127.0.0.1:8080/registration \
  -H "Authorization: Bearer <registration-admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"accountId":"demo-account","deviceId":"demo-device"}'
```

Expected response fields:
- `deviceInstanceId`
- `sub`
- `publ`
- `subs`
- `token`

## Exercise MQTT with the issued token

Use the returned `sub` as the MQTT username and the returned `token` as the MQTT password.

Publish telemetry:

```sh
mosquitto_pub -h 127.0.0.1 -p 1883 \
  -u "<issued-sub>" \
  -P "<issued-token>" \
  -t "demo-account/demo-device/<device-instance-id>/temperature/celsius" \
  -m '{"value":22.5}'
```

Listen for desired-state commands:

```sh
mosquitto_sub -h 127.0.0.1 -p 1883 \
  -u "<issued-sub>" \
  -P "<issued-token>" \
  -t "demo-account/demo-device/<device-instance-id>/+/+/set"
```

Verify broker rejection when the MQTT username does not match `sub`:

```sh
mosquitto_pub -h 127.0.0.1 -p 1883 \
  -u "wrong-user" \
  -P "<issued-token>" \
  -t "demo-account/demo-device/<device-instance-id>/temperature/celsius" \
  -m '{"value":22.5}'
```

Expected result:
- Mosquitto rejects the connection or publish attempt because the MQTT username does not equal the JWT `sub` claim.

## Exercise the REST API with the same token

Read current state:

```sh
curl -H "Authorization: Bearer <issued-token>" \
  "http://127.0.0.1:8080/demo-account/demo-device/<device-instance-id>/temperature/celsius"

curl -H "Authorization: Bearer <issued-token>" \
  "http://127.0.0.1:8080/demo-account/demo-device/<device-instance-id>/temperature/celsius/history?from=2026-03-17T00:00:00Z&to=2026-03-17T01:00:00Z"
```

Request desired state using the same issued token:

```sh
curl -X PUT http://127.0.0.1:8080/demo-account/demo-device/<device-instance-id>/lamp/power \
  -H "Authorization: Bearer <issued-token>" \
  -H "Content-Type: application/json" \
  -d '{"value":"on"}'
```

The `mosquitto_sub` session should receive the `/set` message for the device.

## Exercise the MCP endpoint

Connect an MCP client or inspector to `http://127.0.0.1:8080/mcp` with `Authorization: Bearer <issued-token>` and call:

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