# _local-adr-001: Bridge Process Topology — In-Process Goroutines vs Standalone Binary

## Context and Problem Statement

The MQTT-to-InfluxDB bridge must run continuously alongside the REST API and MCP server components of `stutzthings-server`. A decision is needed on whether the bridge should be:
1. A separate binary process (own container/service), or
2. A set of goroutines embedded inside the existing `stutzthings-server` process.

Question: How should the bridge be deployed relative to the stutzthings-server process?

## Decision Outcome

**In-process goroutines embedded in `stutzthings-server`, structured as the `bridge` package with goroutine-per-concern decomposition**

### Implementation Details

#### Package structure

```
stutzthings/stutzthings-server/
└── bridge/
    ├── bridge.go     # Bridge type, Start(ctx), Stop() — public API
    ├── config.go     # BridgeConfig, env var loading and validation
    ├── mqtt.go       # paho.mqtt.golang client, OnConnect re-subscription, message dispatch
    ├── payload.go    # Topic validation, payload parsing (scalar + JSON)
    ├── buffer.go     # In-memory FIFO ring buffer with configurable ceiling
    └── influx.go     # InfluxDB v3 batch writer, retry loop, startup probe
```

Each source file MUST NOT exceed 400 lines (Constitution IV).

#### Goroutine model

Three goroutines, coordinated via Go channels and a shared context:

```
paho.mqtt.golang (OnConnect callback) → message handler
    → [chan MQTTMessage, unbuffered]
    → buffer goroutine (ring buffer accumulator + flush timer)
    → [chan []DeviceAttribute, capacity = 1]
    → writer goroutine (InfluxDB batch writer + retry)
```

- **mqtt.go** wraps `paho.mqtt.golang` client; `SetAutoReconnect(true)` + `OnConnect` handler re-subscribes to `#` and posts messages to the channel.
- **buffer.go** goroutine reads from the MQTT channel, accumulates points, and flushes on size or interval.
- **influx.go** goroutine reads batches, calls `influxdb3.WritePoints`, retries on failure.
- All goroutines are started by `Bridge.Start(ctx context.Context)` and stop when `ctx` is cancelled.

#### Startup sequence

1. `bridge.Start(ctx)` probes InfluxDB with exponential backoff until reachable (FR-014).
2. Once InfluxDB is reachable, `paho.mqtt.golang` client is connected.
3. Buffer and writer goroutines start in parallel.
4. On successful MQTT connect (`OnConnect` callback), client subscribes to `#` at QoS 1.

#### External libraries

- MQTT: `github.com/eclipse/paho.mqtt.golang` (v1.5.1+)
- InfluxDB: `github.com/InfluxCommunity/influxdb3-go/v2/influxdb3` (v2.13.0+)
- Backoff: `github.com/cenkalti/backoff/v4`

## Considered Options

- (CHOSEN) **In-process goroutines**
  - Reason: FR-010 explicitly requires this; simpler deployment (no inter-process comms); lower latency; single binary to build, test, and monitor.
- (REJECTED) **Separate binary/container**
  - Reason: Increases deployment complexity; requires a network boundary between bridge and server; adds operational overhead (separate health checks, networking, container lifecycle). No current requirement drives this.

## References

- Feature spec: [specs/001-mqtt2influxdb-bridge/spec.md](../../../../specs/001-mqtt2influxdb-bridge/spec.md)
- Feature plan: [specs/001-mqtt2influxdb-bridge/plan.md](../../../../specs/001-mqtt2influxdb-bridge/plan.md)
- Research: [specs/001-mqtt2influxdb-bridge/research.md](../../../../specs/001-mqtt2influxdb-bridge/research.md)
