# deployments

Infrastructure-as-code — one sub-directory per deployment group.

Each group bundles all services needed for that environment: agents server, stutzthings core, MQTT broker (Mosquitto), InfluxDB, and any cloud-side components.

Primary target: Raspberry Pi (Docker Compose). Directory name convention: `<environment-name>/` (e.g. `home/`, `office/`).

## Architecture overview

Each deployment group is a Docker Compose project. Groups are independent — each can be deployed to a separate Raspberry Pi or environment without affecting others.

## How to build

```bash
make build
```

## How to run

```bash
cd <environment-name>
make deploy
```

> Full deployment instructions live inside each environment sub-directory once groups are added.
