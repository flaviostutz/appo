# _local-edr-001: Monorepo organization and module conventions

## Context and Problem Statement

**appo** is a home AI agent system that spans multiple domains: agentic AI, IoT middleware, hardware/firmware, and deployments. These components are heterogeneous in language, toolchain, and release cycle but share a common purpose and need to interoperate.

How should the monorepo be structured so that modules are independently buildable, clearly scoped, and easy to navigate for both human developers and AI coding agents?

## Decision Outcome

**Use a flat top-level directory layout with one directory per domain, a shared library area, and per-group deployments. Each module owns a `Makefile` with standardized targets. A root `Makefile` delegates to all modules.**

This layout keeps each module self-contained while making cross-cutting concerns (shared code, deployments) explicit and easy to discover.

---

### Directory layout

```
appo/
├── appo/               # Agentic AI (Python, LangGraph) — the agent itself
├── stutzthings/        # IoT framework (Python libs + Docker services)
├── devices/            # Hardware and firmware — one sub-dir per device type
│   └── <device-name>/ #   schematics, PCB layout, firmware source, or lib if virtual
├── deployments/        # Infrastructure-as-code — one sub-dir per deployment group
│   └── <group-name>/  #   agents server + stutzthings + MQTT + DBs + cloud components
├── shared/             # Libraries shared across modules
├── .xdrs/              # Decision records (ADR, BDR, EDR)
├── Makefile            # Root Makefile — delegates to module Makefiles
├── AGENTS.md           # AI agent behavior overrides
└── README.md
```

### Module types and conventions

#### `appo/`

- Language: Python
- Framework: LangGraph
- Role: The Appo agent itself. Receives events from external sources (MQTT, webhooks, timers), maintains persistent memory, reads instruction files, and reacts through tool calls.
- Sub-structure: one sub-directory per agent or agent group.

#### `stutzthings/`

- Language: Python
- Deliverables: importable Python library **and** one or more Docker service images
- Role: IoT middleware layer. Responsibilities:
  - Expose an MCP (Model Context Protocol) server so agents can discover devices, query current sensor state, send actuator commands, and read historical data (InfluxDB)
  - Define a standard MQTT topic structure used by all devices and agents
  - Provide a Python SDK for device-side applications to publish state and subscribe to commands

#### `devices/`

- One sub-directory per device type (physical **or** virtual).
- **Physical devices** (e.g. `esp32-cam-mic-matrix/`): schematics (KiCad), PCB layout files, and firmware source (PlatformIO/C++).
- **Virtual devices** (e.g. `alexa/`): a Python integration library that implements the stutzthings MQTT protocol for the virtual device.
- Device directory name convention: `<platform>-<feature-list>/` for physical boards, `<service-name>/` for virtual integrations.

#### `deployments/`

- One sub-directory per deployment group (a group = one running environment).
- Each group bundles all services needed for that environment: agents server, stutzthings core, MQTT broker (Mosquitto), InfluxDB, and any cloud-side components.
- Primary target: Raspberry Pi (Docker Compose). Cloud components are co-located in the same group directory.
- Directory name convention: `<environment-name>/` (e.g. `home/`, `office/`).

#### `shared/`

- Language-agnostic; may contain Python packages and other artifacts.
- Code that is imported by two or more of `appo/`, `stutzthings/`, or `devices/<virtual>/`.
- Each shared library lives in its own sub-directory with its own `Makefile`.

---

### Build system

- Every module directory (including `shared/<lib>/` and `deployments/<group>/`) must contain a `Makefile` with at minimum the following targets:

  | Target | Purpose |
  |---|---|
  | `build-module` | Compile / package the module |
  | `lint-module` | Run linters |
  | `test-module` | Run unit and integration tests |
  | `lint-fix` | Auto-fix lint issues |

- The root `Makefile` provides corresponding aggregate targets (`build`, `lint`, `test`) that invoke `make <target>-module` in each module directory.
- AI coding agents must run quality checks using the canonical sequence defined in [`_general-edr-001`](../../_general/edrs/principles/001-coding-agent-behavior.md): `STAGE=dev make build-module` → `make lint-module` → `make test-module`.
- No ad-hoc shell commands outside Makefiles.

---

### Shared code strategy

- Prefer extracting shared code to `shared/` over duplicating it across modules.
- Modules import shared libraries as local packages (Python path / editable install in dev, copied artifact in CI).
- `shared/` libraries are versioned independently and have their own `Makefile`.

---

### Interoperability

- Agents talk to stutzthings **only** through the MCP interface (no direct MQTT or DB access from agent code).
- Stutzthings talks to devices **only** through MQTT using the standard topic structure defined in the stutzthings module.
- No other cross-module runtime coupling is permitted without a new ADR.

---

## Considered Options

* **(REJECTED) Language-grouped layout** (`python/`, `firmware/`, `infra/`) — groups by technology rather than domain. Harder to reason about system boundaries; agent and stutzthings code would sit in the same `python/` tree despite being independently deployable.
* **(REJECTED) Separate repositories per module** — maximum isolation but loses the benefits of atomic cross-module changes, shared tooling, and unified decision records.
* **(CHOSEN) Flat domain-per-directory monorepo** — domain boundaries match team mental model, modules are independently buildable, and deployments explicitly declare their composition.
