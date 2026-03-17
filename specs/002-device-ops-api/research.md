# Research: Device Operations API and MCP Access

## Decision: Use Mosquitto-plugin-compatible JWT claims as the canonical auth contract

**Rationale**:
- The feature now requires one JWT model that works for MQTT, REST, and MCP.
- `wiomoc/mosquitto-jwt-auth` authorizes MQTT clients through `sub`, `publ`, and `subs`, so reusing those claims avoids maintaining parallel auth semantics.
- The existing MQTT topic hierarchy already encodes tenant, device, instance, node, and attribute identity, making topic-filter-based authorization a natural fit.

**Alternatives considered**:
- Custom `i:` and `r:` scope grammars: rejected because they duplicate broker auth semantics and create drift between MQTT and HTTP transports.
- Separate JWT formats for MQTT and REST/MCP: rejected because they would complicate registration and parity.

## Decision: Use HS256-signed JWTs with one shared base64-encoded secret for Go and Mosquitto

**Rationale**:
- The registration flow is stateless and server-issued, so one server-side signing secret is sufficient for the first implementation.
- `wiomoc/mosquitto-jwt-auth` accepts a base64-encoded secret from config or environment, so the Go service should consume the same base64 source to avoid secret drift.
- HS256 keeps the local developer workflow lightweight while remaining compatible with the plugin.

**Alternatives considered**:
- RS256/JWKS from day one: stronger federation story, but more moving parts than the current monolithic service and local workflow require.
- Separate secrets for broker and API: rejected because it breaks the one-token model.

## Decision: Registration-issued device JWTs include standard timing claims and a bounded lifetime

**Rationale**:
- Device-registration tokens are intended for immediate operational use, so the token payload needs explicit issuance and expiry semantics rather than only transport-specific topic claims.
- Requiring `iss`, `iat`, and `exp` keeps the credential format explicit across MQTT, REST, and MCP and gives operators a clear rotation boundary.
- A default 24-hour lifetime is long enough for the local workflow and short enough to avoid pretending the first implementation already has indefinite credential management solved.

**Alternatives considered**:
- Non-expiring tokens: rejected because they make shared-secret rotation and compromise response undefined.
- Leaving lifetime entirely unspecified: rejected because it makes registration output incomplete and untestable.

## Decision: Shared-secret rotation is coordinated across the Go service and Mosquitto

**Rationale**:
- The service and broker must accept the same token, so rotation has to be described as one coordinated action rather than two independent configuration changes.
- The simplest first-release rule is that rotating the shared base64 secret invalidates previously issued registration tokens unless an explicit overlap policy is introduced later.

**Alternatives considered**:
- Independent broker and API rotation schedules: rejected because they break the one-token contract.
- Silent overlap behavior without documentation: rejected because operators could not reason about token validity.

## Decision: MCP authentication is per HTTP request, not session state

**Rationale**:
- The parity requirement is easiest to preserve when MCP uses the same bearer-token parsing model as REST.
- Per-request authentication avoids hidden session state that could make REST and MCP diverge for the same token.

**Alternatives considered**:
- Session-bound MCP authentication: rejected because it would create transport-specific auth semantics.

## Decision: Derive REST and MCP authorization from MQTT topic filters

**Rationale**:
- Reads, history queries, and desired-state writes already map onto telemetry and `/set` topics.
- The server can authorize current-state and history reads when the addressed telemetry topic matches a filter in `publ` or `subs`.
- The server can authorize desired-state writes when the addressed `/set` topic matches a filter in `subs`, which aligns command authority with the command channel a client is allowed to receive.

**Alternatives considered**:
- Separate REST/MCP permission claims: rejected because they reintroduce transport-specific auth rules.
- Mapping desired-state writes to `publ` on `/set`: rejected because the registration-issued device JWT should publish telemetry and subscribe for commands without needing a second token.

## Decision: Reuse the current bridge's InfluxDB client and query the existing `device_attributes` measurement directly

**Rationale**:
- The bridge already writes device observations with the tags and typed fields defined in `_local-bdr-002`.
- `bridge/influx.go` already exposes `QueryRows`, so the new feature can add read-oriented query builders without introducing a second Influx integration path.
- Latest-attribute and history queries can be expressed directly against the stored measurement, while whole-device reads can be assembled by selecting the newest row per `(node_name, attribute_name)` pair and filtering authorized attributes in application code.

**Alternatives considered**:
- Separate read model or cache: rejected because the existing measurement already satisfies the read use cases.
- Querying through an external reporting service: rejected because it would add a new deployment boundary for no current gain.

## Decision: Implement one shared operations service and adapt it to REST and MCP

**Rationale**:
- The spec requires REST and MCP parity for both success and failure behavior.
- The current server already uses `net/http` and a single process, so the least risky architecture is to keep one business service inside `stutzthings-server` and expose it through two transport adapters.
- `mark3labs/mcp-go` fits the existing `http.ServeMux` model and can be mounted as another handler on the same server.

**Alternatives considered**:
- REST only in Go and MCP in another process or language: rejected because it duplicates business logic and violates parity.
- Hand-rolled MCP JSON-RPC transport: rejected because a maintained library is cheaper and safer.

## Decision: Canonical local verification flow remains `make run`, but it must start plugin-enabled Mosquitto

**Rationale**:
- The module already has a `make run` target and local Docker workflow; the feature requirement is to make that the one-command end-to-end entrypoint.
- Because MQTT is now part of the unified JWT model, local verification must include Mosquitto configured with `wiomoc/mosquitto-jwt-auth` and the same signing secret used by the Go service.
- `examples/local/` remains the richer environment for Grafana and host-run debugging, but it should not replace the canonical developer command.

**Alternatives considered**:
- Making `examples/local/` the only supported path: rejected because it violates the feature's explicit one-command requirement.
- Keeping username/password auth for local Mosquitto: rejected because it would fail to validate the actual feature behavior.
