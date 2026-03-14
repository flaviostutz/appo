# Quickstart: MQTT to InfluxDB Bridge

Run the bridge locally for development and testing in under 5 minutes.

## Prerequisites

- Go 1.22+ installed (`go version`)
- Docker and Docker Compose installed
- `make` available

## 1. Start local infrastructure

```bash
cd stutzthings/stutzthings-server
docker compose up -d
```

This starts:
- **EMQX** MQTT broker on `localhost:1883` (username: `test`, password: `test`)
- **InfluxDB v3** on `localhost:8086` (token: `dev-token`, database: `iot`)

## 2. Set environment variables

```bash
export MQTT_BROKER_URL=mqtt://localhost:1883
export MQTT_USERNAME=test
export MQTT_PASSWORD=test
export INFLUXDB_URL=http://localhost:8086
export INFLUXDB_TOKEN=dev-token
export INFLUXDB_DATABASE=iot
```

Optional tuning (defaults shown):

```bash
export BRIDGE_BATCH_SIZE=100
export BRIDGE_FLUSH_INTERVAL_MS=100
export BRIDGE_MAX_BUFFER_SIZE=10000
export BRIDGE_MAX_WRITE_RETRIES=3
```

## 3. Build and run

```bash
make build
./stutzthings-server
```

The server starts and the bridge goroutine logs:

```
INFO  bridge: InfluxDB reachable, subscribing to MQTT wildcard (#)
INFO  bridge: connected to mqtt://localhost:1883
```

## 4. Publish a test message

Install `mosquitto-clients` or use any MQTT client. Example with `mosquitto_pub`:

```bash
# Numeric value (raw scalar)
mosquitto_pub -h localhost -p 1883 \
  -u test -P test \
  -t "account1/sensor/device01/temperature/celsius" \
  -m "23.5"

# JSON payload with embedded timestamp
mosquitto_pub -h localhost -p 1883 \
  -u test -P test \
  -t "account1/sensor/device01/humidity/percent" \
  -m '{"value": 65.2, "ts": 1741910400000}'

# Boolean value
mosquitto_pub -h localhost -p 1883 \
  -u test -P test \
  -t "account1/sensor/device01/power/on" \
  -m "true"
```

## 5. Verify data in InfluxDB

Query via the InfluxDB UI at `http://localhost:8086` or via curl:

```bash
curl -s "http://localhost:8086/query" \
  -H "Authorization: Token dev-token" \
  -H "Content-Type: application/json" \
  -d '{"query": "SELECT * FROM device_attributes WHERE time > now() - 1m", "database": "iot"}'
```

Expected result: records tagged with `account_id=account1`, `device_id=sensor`, `device_instance_id=device01`, `node_name=temperature`, `attribute_name=celsius`, field `value_float=23.5`.

## 6. Run tests

```bash
make test
```

Integration tests spin up their own broker and InfluxDB containers via testcontainers-go and tear them down automatically. Unit tests run without any infrastructure.

## Resilience testing

**MQTT broker outage**:
```bash
docker compose stop emqx
sleep 15
docker compose start emqx
# Publish a message — bridge reconnects automatically within 10s
mosquitto_pub -h localhost -p 1883 -u test -P test \
  -t "account1/sensor/device01/temperature/celsius" -m "25.0"
```

**InfluxDB outage** (messages buffer up to `BRIDGE_MAX_BUFFER_SIZE`, then oldest are evicted):
```bash
docker compose stop influxdb
# Publish messages — bridge buffers them
mosquitto_pub -h localhost -p 1883 -u test -P test \
  -t "account1/sensor/device01/temperature/celsius" -m "26.0"
docker compose start influxdb
# Bridge retries and flushes the buffer
```
