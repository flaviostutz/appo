# _local-bdr-003: Device Authorization Scope Grammar and Wildcard Rules

## Context and Problem Statement

The device-operations feature introduces protected reads, history queries, desired-state writes, and protected registration. The platform needs one durable authorization grammar that preserves tenant isolation while allowing bounded wildcard delegation.

Question: What bearer-token scope grammars must the platform support for device operations and registration, and how are wildcard matches evaluated?

## Decision Outcome

**Two scope families: `i:` for device operations and `r:` for registration; `*` matches whole path segments only**

### Implementation Details

#### Device-operation scopes

- Grammar: `i:account_id/device_id/device_instance_id/node_name/attribute_name:actions`
- `actions` may contain only `r`, `w`, and `s`
  - `r`: read current state and history
  - `w`: attribute write capability when applicable
  - `s`: desired-state command publication
- `*` is allowed in any path segment and matches exactly one whole segment.

#### Registration scopes

- Grammar: `r:account_id/device_id`
- `*` is allowed in both path segments, for example `r:*/*`.
- Registration scopes never include an action suffix.

#### Matching rules

- Matching is segment-by-segment; substring or prefix matching is forbidden.
- Authorization checks MUST always include the `account_id` segment so tenant isolation is preserved.
- If a scope does not authorize the requested path and action, the operation is rejected with no data leakage.
- Whole-device reads may return only the authorized attributes; when any stored attributes are excluded, the result is marked partial.

## Considered Options

- (CHOSEN) **Attribute-level `i:` scopes plus separate `r:` registration scopes**
  - Reason: Fits the clarified feature behavior, supports partial whole-device reads, and keeps registration authorization simpler.
- (REJECTED) **Coarser device-instance scopes only**
  - Reason: Cannot represent attribute-level filtering or partial results cleanly.
- (REJECTED) **One generic scope grammar for all protected operations**
  - Reason: Registration authorization is structurally different from attribute-level device access.

## References

- Feature spec: [specs/002-device-ops-api/spec.md](../../../../specs/002-device-ops-api/spec.md)
- Feature plan: [specs/002-device-ops-api/plan.md](../../../../specs/002-device-ops-api/plan.md)
- Related: [product/001-mqtt-topic-structure.md](001-mqtt-topic-structure.md)
- Related: [product/002-influxdb-device-attributes-data-model.md](002-influxdb-device-attributes-data-model.md)
*** Add File: /Users/flaviostutz/Documents/development/flaviostutz/appo/.xdrs/_local/bdrs/product/004-stateless-device-registration-policy.md
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