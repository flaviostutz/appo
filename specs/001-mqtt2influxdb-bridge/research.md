# Research: MQTT to InfluxDB Bridge Daemon

**Feature**: `001-mqtt2influxdb-bridge`  
**Date**: 2026-03-14  
**Status**: Complete — all NEEDS CLARIFICATION resolved

---

## Decision 1: Go MQTT Client Library

**Decision**: Use `github.com/eclipse/paho.mqtt.golang` (v1.5.1)
**Version**: v1.5.1 (September 2025, actively maintained with recent commits)

**Rationale**: `paho.mqtt.golang` is the battle-tested, production-proven MQTT 3.1.1 client for Go with 3.1K stars and 75 contributors. With MQTT downgraded to 3.1.1 for reliability, this library is the clear choice: built-in `SetAutoReconnect(true)` with connection-loss hooks, QoS 1 subscriptions, wildcard support, username/password auth, and optional TLS. The newer `paho.golang` (v5 client) requires more complex session management and has a smaller adoption base.

**Re-subscription on reconnect**: `paho.mqtt.golang` auto-reconnects but does not automatically re-subscribe. The bridge MUST re-subscribe in the `OnConnect` handler (registered at client creation). This is a one-line pattern and is the standard usage for this library.

**Alternatives considered**:
- `eclipse/paho.golang` (`autopaho`) — MQTT v5 client; more reliable for v5 but the spec now requires v3.1.1. Larger API surface, immature session persistence. Rejected.
- `at-wat/mqtt-go` — MQTT 3.1.1 only, minimal maintenance (3 contributors, 41 stars). Rejected.
- `mochi-mqtt/server` — primarily a broker; client usage is for internal testing only. Rejected.

**Import path**: `github.com/eclipse/paho.mqtt.golang`

**Key features used**:
- `mqtt.NewClient(opts)` with `SetAutoReconnect(true)`, `SetOnConnectHandler` for re-subscription
- `client.Subscribe("#", qos1, handler)` — wildcard subscription in `OnConnect`
- `opts.SetUsername()` / `opts.SetPassword()` for auth
- `opts.SetTLSConfig()` for optional TLS

---

## Decision 2: InfluxDB v3 Go Client Library

**Decision**: Use `github.com/InfluxCommunity/influxdb3-go/v2/influxdb3`  
**Version**: v2.13.0 (actively maintained, last release ~3 weeks before 2026-03-14)

**Rationale**: The `influxdata/influxdb-client-go` v2 client explicitly does not support InfluxDB v3. The `influxdb3-go` client is the official v3 client from InfluxData's community repositories, supports plain HTTP (no TLS at app layer), token authentication, and native line-protocol write. Actively maintained with 21 contributors and regular releases.

**Alternatives considered**:
- `influxdata/influxdb-client-go` (v2 client) — incompatible with InfluxDB v3 API. Rejected.
- Raw HTTP line protocol — requires manual batching, retry, and error handling; unnecessary complexity. Rejected.

**Import path**: `github.com/InfluxCommunity/influxdb3-go/v2/influxdb3`

**Batch write strategy**: `influxdb3-go` exposes `WritePoints(ctx, []*Point)`. Our bridge will maintain its own in-memory ring buffer (FR-013, FR-015) and call `WritePoints` when the flush condition is met. This gives us full control over batch size (default 100) and flush interval (default 100ms), independent of any library-internal batching.

---

## Decision 3: In-Process Goroutine Architecture

**Decision**: Single-package `bridge` embedded in `stutzthings-server` with goroutine-per-concern decomposition

**Rationale**: FR-010 mandates in-process goroutines (not a separate binary). The bridge has three independent, blocking concerns that map naturally to goroutines:
1. **MQTT subscriber goroutine**: blocks on `paho.mqtt.golang` message callback delivery
2. **Buffer manager goroutine**: accumulates messages into the ring buffer, triggers flush on size or timer
3. **InfluxDB writer goroutine**: receives batches via channel, writes with retry

**Go channel design**:
```
MQTT callback → unbuffered chan MQTTMessage → ring buffer goroutine → chan []Point → writer goroutine → InfluxDB
```
Using channels for inter-goroutine communication avoids shared-state races. The ring buffer is the only shared structure and uses a mutex.

---

## Decision 4: Exponential Backoff Library

**Decision**: Use `github.com/cenkalti/backoff/v4`  
**Rationale**: Battle-tested, widely adopted in the Go ecosystem, supports context cancellation, provides `ExponentialBackOff` with configurable `InitialInterval`, `MaxInterval`, `MaxElapsedTime`, and `Multiplier`. Zero-dependency alternative would be a hand-rolled ticker, but cenkalti/backoff is standard enough to not require an XDR.

Used for:
- MQTT reconnect (handled internally by `paho.mqtt.golang` `SetAutoReconnect`; re-subscription handled in `OnConnect` handler)
- InfluxDB startup readiness probe (FR-014)
- InfluxDB write retry loop (FR-007, default 3 attempts)

---

## Note: BDR-001 Already Updated

BDR-001 (`001-mqtt-topic-structure.md`) has been updated to use `ts` (Unix epoch, milliseconds) as the JSON timestamp field name, replacing the previous `timestamp` (ISO 8601). This was applied during Phase 1 of the plan.
