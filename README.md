# appo

Home AI agent system that connects intelligent agents to physical devices, sensors, and actuators. The system bridges large language model reasoning with the real world through IoT integration, persistent memory, and event-driven automation.

---

## What it does

**appo** lets autonomous AI agents perceive and act on the physical world:

- Agents observe events (sensor readings, calendar entries, messages, etc.) and react using tools
- Devices (ESP32 boards, smart speakers, virtual integrations) publish state and receive commands over MQTT
- Agents discover devices, query historical sensor data, and issue actuator commands through a Model Context Protocol (MCP) interface
- Everything runs on self-hosted Raspberry Pi infrastructure alongside cloud integrations when needed

---

## Modules

| Directory | Type | Description |
|---|---|---|
| [`appo/`](appo/) | Python library | LangGraph-based agentic AI (Appo — the agent itself). Receives events, manages memories, reads instructions, and reacts through tool calls. |
| [`stutzthings/`](stutzthings/) | Python library + Docker services | IoT framework. Exposes an MCP endpoint for agents to discover devices, get sensor state, send actuator commands, and query historical data (InfluxDB). Defines a standard MQTT topic structure and a Python SDK for device apps. |
| [`devices/`](devices/) | Hardware + firmware | One sub-directory per device type. Includes schematics, PCB layout, firmware source, and—for virtual devices (e.g. Alexa)—a Python integration library. |
| [`deployments/`](deployments/) | Infrastructure-as-code | One sub-directory per deployment group. Each group bundles the agents server, stutzthings core, MQTT broker, databases, and any cloud-side components needed for that environment. |
| [`shared/`](shared/) | Shared libraries | Code shared across multiple modules (common data models, utilities, etc.). |

### Current devices

| Device | Location | Description |
|---|---|---|
| ESP32-CAM-MIC-MATRIX | [`devices/esp32-cam-mic-matrix/`](devices/esp32-cam-mic-matrix/) | Custom ESP32 board with camera, microphone, and LED matrix. |

---

## Architecture overview

```
                  ┌─────────────────────────────────┐
                  │            appo/               │
                  │   LangGraph event-driven agent  │
                  │  (memory, instructions, tools)  │
                  └────────────┬────────────────────┘
                               │ MCP
                  ┌────────────▼────────────────────┐
                  │          stutzthings/            │
                  │  MCP server · MQTT bridge        │
                  │  InfluxDB historical data        │
                  │  Python SDK for device apps      │
                  └──────┬─────────────┬────────────┘
                         │ MQTT        │ HTTP/InfluxDB
          ┌──────────────▼──┐     ┌───▼──────────┐
          │   devices/      │     │  InfluxDB    │
          │  esp32-cam-…    │     │  (time-series│
          │  alexa / etc.   │     │   history)   │
          └─────────────────┘     └──────────────┘

  All of the above deployed via deployments/<group>/
  on Raspberry Pi + optional cloud components
```

---

## Getting started

### Machine setup

1. **Install [Mise](https://mise.jdx.dev/getting-started.html)** — manages tool versions across the monorepo:

   ```bash
   curl https://mise.run | sh
   ```

2. **Activate tools** — from the repository root:

   ```bash
   mise install
   ```

   This installs the exact tool versions declared in `.mise.toml` (Python, etc.).

3. **Additional prerequisites** (not managed by Mise):
   - Docker & Docker Compose — for service deployments
   - PlatformIO — for ESP32 firmware (install via `pip install platformio`)
   - Make — typically pre-installed on macOS/Linux

   Or run:

   ```bash
   make setup
   ```

   for a printed checklist.

### Quickstart

Build, lint, and test all modules from the repository root:

```bash
make build
make lint
make test
```

Deploy a specific environment:

```bash
cd deployments/<group>
make deploy
```

> Full per-module instructions live inside each directory's `README.md`.

---

## Project decisions

Architecture, engineering, and business decisions are captured as XDRs (Decision Records) in [`.xdrs/`](.xdrs/). Consult the [index](.xdrs/index.md) before making implementation decisions.
