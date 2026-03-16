# _local-bdr-004: Stateless Device Registration Policy

## Context and Problem Statement

The platform needs a protected registration flow that creates a new device instance identity and returns credentials the device can immediately use for telemetry and command participation, without storing registration session state on the server.

Question: How must device registration be authorized, what must it return, and what state does the server retain afterward?

## Decision Outcome

**Protected `POST /registration` that issues one new device instance identity and one `rws` credential scoped to that instance; no server-side registration session is retained**

### Implementation Details

- `POST /registration` requires a bearer token with registration scope `r:account_id/device_id`.
- Each successful request returns:
  - `account_id`
  - `device_id`
  - a newly generated `device_instance_id`
  - one JWT with scope `i:account_id/device_id/device_instance_id/*/*:rws`
- The issued JWT is limited to the new device instance only.
- Registration does not create or retain a server-side registration session after the response is sent.
- Failed or unauthorized registration requests must not consume or expose a partially created device identity.

## Considered Options

- (CHOSEN) **Protected stateless registration with one scoped JWT**
  - Reason: Matches the feature spec, minimizes operational state, and keeps the device bootstrap experience simple.
- (REJECTED) **Public registration endpoint**
  - Reason: Creates an unnecessary abuse surface for identity creation.
- (REJECTED) **Multiple returned credentials**
  - Reason: Adds complexity without a requirement for separate publish/command identities.

## References

- Feature spec: [specs/002-device-ops-api/spec.md](../../../../specs/002-device-ops-api/spec.md)
- Feature plan: [specs/002-device-ops-api/plan.md](../../../../specs/002-device-ops-api/plan.md)
- Related: [product/003-device-authorization-scope-grammar.md](003-device-authorization-scope-grammar.md)