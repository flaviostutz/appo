# _local-bdr-004: Stateless Device Registration Policy

## Context and Problem Statement

The platform needs a protected registration flow that creates a new device instance identity and returns one JWT the device can immediately use with Mosquitto, REST, and MCP, without storing registration session state on the server.

Question: How must device registration be authorized, what JWT must it return, and what state does the server retain afterward?

## Decision Outcome

**Protected `POST /registration` that issues one new device instance identity and one plugin-compatible JWT narrowed to that instance; no server-side registration session is retained**

### Implementation Details

- `POST /registration` requires a bearer JWT whose wildcard `publ` and `subs` filters cover the requested `account_id/device_id` namespace.
- Each successful request returns:
  - `account_id`
  - `device_id`
  - a newly generated lowercase UUIDv7 `device_instance_id`
  - `sub=account_id/device_id/device_instance_id`
  - `publ=[account_id/device_id/device_instance_id/+/+]`
  - `subs=[account_id/device_id/device_instance_id/+/+/set]`
  - `iss`, `iat`, and `exp`
  - one signed JWT containing those claims
- The issued JWT is limited to the new device instance only and is valid for Mosquitto, REST, and MCP.
- The default registration-token lifetime is 24 hours unless deployment configuration overrides it.
- When the JWT is used with MQTT, the client must connect with `username=sub` and `password=token`.
- If a generated UUID collides within `(account_id, device_id)`, generation is retried until unique.
- Registration does not create or retain a server-side registration session after the response is sent.
- Failed or unauthorized registration requests must not consume or expose a partially created device identity.
- The same base64-encoded signing secret source is shared between the Go service and Mosquitto; rotating that secret invalidates previously issued tokens unless an explicit overlap policy is introduced.

## Considered Options

- (CHOSEN) **Protected stateless registration with one plugin-compatible JWT**
  - Reason: Matches the feature spec, minimizes operational state, and keeps the device bootstrap experience simple.
- (REJECTED) **Public registration endpoint**
  - Reason: Creates an unnecessary abuse surface for identity creation.
- (REJECTED) **Multiple returned credentials**
  - Reason: Adds complexity without a requirement for separate transport-specific identities.

## References

- Feature spec: [specs/002-device-ops-api/spec.md](../../../../specs/002-device-ops-api/spec.md)
- Feature plan: [specs/002-device-ops-api/plan.md](../../../../specs/002-device-ops-api/plan.md)
- Related: [product/003-device-authorization-scope-grammar.md](003-device-authorization-scope-grammar.md)