# _local-edr-001: MQTT Reconnect and InfluxDB Retry Policy

## Context and Problem Statement

The bridge must tolerate temporary unavailability of both the MQTT broker (FR-006) and InfluxDB (FR-007, FR-014) without operator intervention and without silent data loss. Concrete decisions are needed on:
- MQTT reconnect strategy and re-subscription semantics
- InfluxDB startup readiness probe behaviour
- InfluxDB write retry backoff parameters
- In-memory buffer eviction policy during extended outages

Question: What are the concrete reconnect, retry, and backoff parameters for the bridge's resilience mechanisms?

## Decision Outcome

**paho.mqtt.golang auto-reconnect with OnConnect re-subscription; exponential backoff for InfluxDB probe and write retries; FIFO ring-buffer eviction at ceiling**

### Implementation Details

#### MQTT reconnect (FR-006)

Handled by `paho.mqtt.golang` with application-level re-subscription:
- `SetAutoReconnect(true)` on the client options — the library reconnects automatically after disconnection.
- `SetOnConnectHandler(onConnect)` — the `onConnect` function calls `client.Subscribe("#", qos1, handler)` every time a connection is (re)established. This is the standard pattern for re-subscription with this library.
- No application-level reconnect loop is required.
- On each reconnect: log INFO with broker address and attempt number.

#### InfluxDB startup readiness probe (FR-014)

The bridge blocks MQTT subscription until InfluxDB is reachable:

```
Probe loop (runs before MQTT connect):
  attempt 1: wait immediately
  attempt 2: wait 1s
  attempt 3: wait 2s
  attempt N: wait min(2^(N-1) seconds, 30s)
  no max elapsed time — probes indefinitely until reachable
```

- Uses exponential backoff with `InitialInterval = 1s`, `MaxInterval = 30s`, `Multiplier = 2.0`, `MaxElapsedTime = 0` (unlimited).
- On each probe failure: log WARNING with attempt count and error.
- On probe success: log INFO and proceed to MQTT connection.

#### InfluxDB write retry (FR-007)

When a batch write fails:

```
Attempt 1: immediate write
Attempt 2: wait 500ms, retry
Attempt 3: wait 1s, retry
After MaxWriteRetries (default 3) exhausted: discard batch, log WARNING with dropped count
```

- Uses fixed-interval backoff (not exponential) for write retries to bound latency spike.
- `InitialInterval = 500ms`, `MaxRetries = 3` (configurable via `BRIDGE_MAX_WRITE_RETRIES`).
- On final failure: log WARNING including batch size, first/last timestamps, and error.

#### In-memory buffer eviction (FR-015)

When the ring buffer reaches `MaxBufferSize` (default 10,000 messages):
- Evict the oldest `BatchSize` messages (FIFO eviction) to make room for incoming.
- Log one WARNING per eviction event, including the count of dropped messages and the oldest dropped timestamp.
- Continue accepting new messages.

Rationale: preserving the most recent device state is more valuable than historical readings during an outage.

#### Buffer sizing guidance

At 500 msg/s (SC-003), the default 10,000-message buffer covers 20 seconds of incoming data during an InfluxDB outage before eviction begins. The default 100ms flush interval gives 5 flush opportunities per second, well within the 2-second SC-001 latency requirement.

## Considered Options

- (CHOSEN — MQTT) **paho.mqtt.golang with OnConnect re-subscription**
  - Reason: Battle-tested library with 3.1K stars; `SetAutoReconnect` + `OnConnect` handler is a documented, standard pattern; MQTT 3.1.1 is universally supported by all brokers; simpler API than MQTT 5 clients.
- (REJECTED — MQTT) **Application-level reconnect loop**
  - Reason: Duplicates what `paho.mqtt.golang` `SetAutoReconnect` already provides; error-prone to implement correctly with clean shutdown.
- (CHOSEN — InfluxDB retry) **Fixed-interval backoff**
  - Reason: Bounds worst-case write-failure latency at 1.5s total (500ms + 1s); exponential would risk exceeding SC-001 on retry paths.
- (CHOSEN — eviction) **FIFO eviction (drop oldest)**
  - Reason: Most recent device state is most valuable; blocking MQTT consume would back-pressure the broker and corrupt QoS 1 semantics.

## References

- Feature spec: [specs/001-mqtt2influxdb-bridge/spec.md](../../../../specs/001-mqtt2influxdb-bridge/spec.md)
- Feature plan: [specs/001-mqtt2influxdb-bridge/plan.md](../../../../specs/001-mqtt2influxdb-bridge/plan.md)
- Research: [specs/001-mqtt2influxdb-bridge/research.md](../../../../specs/001-mqtt2influxdb-bridge/research.md)
