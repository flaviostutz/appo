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

> Full setup instructions live inside each module directory. See the relevant `README.md` under `appo/`, `stutzthings/`, or `deployments/<group>/`.

### Prerequisites

- Python 3.11+ (agents, stutzthings)
- Docker & Docker Compose (service deployments)
- PlatformIO (ESP32 firmware)
- Make (all modules expose a standard `Makefile`)

### Build a module

```bash
cd <module-dir>
STAGE=dev make build-module
make lint-module
make test-module
```

### Deploy an environment

```bash
cd deployments/<group>
make deploy
```

---

## Project decisions

Architecture, engineering, and business decisions are captured as XDRs (Decision Records) in [`.xdrs/`](.xdrs/). Consult the [index](.xdrs/index.md) before making implementation decisions.
