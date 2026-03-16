# local example stack

Local development stack for `stutzthings-server` with Mosquitto, InfluxDB 3 Core, and Grafana managed by Docker Compose and the Go bridge running on the host.

## prerequisites

- `mise`
- Docker Desktop or another local Docker engine with `docker compose`

## included files

- `bridge.json` points the bridge at localhost services
- `docker-compose.yml` starts Mosquitto, InfluxDB 3 Core, and Grafana with the required volumes and ports
- `influxdb/admin-token.json` contains the local InfluxDB admin token used by the example stack
- `grafana/` provisions Grafana with an InfluxDB SQL datasource and a dashboard panel
- `Makefile` starts and stops the compose-managed infrastructure and the local `stutzthings-server` process

## usage

```sh
cd stutzthings/stutzthings-server/examples/local
make start
curl -i http://127.0.0.1:8080/health
make stop
```

If you want to drive the infrastructure directly, use:

```sh
cd stutzthings/stutzthings-server/examples/local
docker compose up -d
make create-database
make start-bridge
```

Open Grafana at `http://127.0.0.1:3000/d/stutzthings-local-bridge/stutzthings-local-mqtt-bridge` after `make start`.

Useful targets:

- `make start-infra`
- `make start-mqtt`
- `make start-influxdb`
- `make create-database`
- `make start-bridge`
- `make start-grafana`
- `make status`
- `make logs`
- `make clean`

Runtime state is written to `.run/`, `.logs/`, and `.data/` under this folder. Docker Compose uses `.data/` for Mosquitto, InfluxDB, and Grafana persistence.

The provisioned dashboard panel shows numeric values from the `device_attributes` measurement in the `iot` database. The panel stays empty until devices publish numeric MQTT attributes through the bridge.