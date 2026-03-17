# _local-edr-002: Local Device Operations Development Workflow

## Context and Problem Statement

The feature requires a single-command local verification workflow that exercises the bridge, plugin-enabled Mosquitto, REST routes, MCP routes, and JWT-backed authorization. The repository already has `make run` in `stutzthings-server` and a richer `examples/local/` stack.

Question: What is the canonical local developer workflow for end-to-end verification of device operations?

## Decision Outcome

**`make run` in `stutzthings-server` is the canonical entrypoint; it must bootstrap plugin-enabled Mosquitto with the same JWT secret as the Go service; `examples/local/` remains the advanced observability stack**

### Implementation Details

- `make run` from `stutzthings/stutzthings-server` is the default local end-to-end command.
- The command must bring up local dependencies and run the Go server with bridge, REST, MCP, and JWT settings suitable for development.
- The same base64-encoded signing secret must be supplied to both the Go service and `wiomoc/mosquitto-jwt-auth`.
- Mosquitto must run with the plugin enabled so MQTT, REST, and MCP all exercise the same JWT semantics.
- `examples/local/` remains the richer optional workflow for Grafana and direct infrastructure inspection.
- Documentation and quickstart examples must reference `make run` first and `examples/local/` second.
- Regression verification for this feature must include `make test`, `make lint`, and at least one manual MQTT + REST/MCP smoke flow against the local stack.

## Considered Options

- (CHOSEN) **Canonical `make run` plus optional `examples/local/`**
  - Reason: Meets the feature requirement while preserving the existing richer stack for troubleshooting.
- (REJECTED) **`examples/local/` as the only supported workflow**
  - Reason: Fails the one-command expectation for normal development.
- (REJECTED) **Manual multi-step startup only**
  - Reason: Increases friction and makes regressions harder to reproduce.

## References

- Feature spec: [specs/002-device-ops-api/spec.md](../../../../specs/002-device-ops-api/spec.md)
- Feature plan: [specs/002-device-ops-api/plan.md](../../../../specs/002-device-ops-api/plan.md)
- Related: [patterns/001-bridge-reconnect-retry-policy.md](001-bridge-reconnect-retry-policy.md)