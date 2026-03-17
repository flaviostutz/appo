# MCP Tool Contract

The MCP surface must expose the same business capability as the REST API. Tool handlers call the same shared Go operations service used by the HTTP handlers and use the same JWT topic-filter authorization model already enforced by Mosquitto.

## Tools

| Tool | Purpose | Required Input | Authorization | Result |
|------|---------|----------------|---------------|--------|
| `get_attribute_state` | Read latest value for one attribute | `account_id`, `device_id`, `device_instance_id`, `node_name`, `attribute_name` | Telemetry topic matches at least one filter in `publ` or `subs` | One latest observation or `found=false` |
| `get_device_state` | Read latest state snapshot for one device instance | `account_id`, `device_id`, `device_instance_id` | Visible telemetry topics match at least one filter in `publ` or `subs` | Snapshot with `is_partial`, `excluded_attribute_count`, and visible attributes only |
| `get_node_history` | Read history for all authorized attributes under one node | `account_id`, `device_id`, `device_instance_id`, `node_name`, `from`, `to` | Telemetry topics under the node match at least one filter in `publ` or `subs` | Ordered history result for the requested range or `observations=[]` |
| `get_attribute_history` | Read history for one attribute | `account_id`, `device_id`, `device_instance_id`, `node_name`, `attribute_name`, `from`, `to` | Telemetry topic matches at least one filter in `publ` or `subs` | Ordered history result for the requested range or `observations=[]` |
| `set_desired_state` | Publish desired value to the device `/set` topic | `account_id`, `device_id`, `device_instance_id`, `node_name`, `attribute_name`, `value` | `/set` topic matches at least one filter in `subs` | `accepted=true` and published topic |
| `register_device_instance` | Create a new device instance and return one scoped credential | `account_id`, `device_id` | Wildcard `publ` and `subs` filters cover the requested account/device namespace | `device_instance_id`, JWT token, `sub`, `publ`, `subs`, `iss`, `issued_at`, `expires_at` |

## Shared Rules

- MCP errors must mirror REST business failures using the same error codes: `invalid_input`, `window_too_large`, `unauthorized`, `forbidden`, and `backend_failure`.
- Tool inputs use the same field names and validation rules as the REST contract.
- `get_device_state` must return `is_partial=true` and `excluded_attribute_count>0` when authorization filtered out stored attributes.
- An authorized empty device snapshot returns `attributes=[]`, `is_partial=false`, and `excluded_attribute_count=0`.
- History tools must reject missing, unparseable, inverted, non-UTC, or legacy relative `from`/`to` inputs.
- History tools return `observations=[]` for authorized empty results and reject requests expected to exceed 10,000 observations with `window_too_large`.
- History tools order observations by `observed_at` ascending and then by `(node_name, attribute_name, canonical_value_string)` ascending when timestamps tie.
- `set_desired_state` acknowledges publication only; it does not imply the device has reported a new actual state.
- `register_device_instance` returns one JWT with `sub=account_id/device_id/device_instance_id`, `publ=[account_id/device_id/device_instance_id/+/+]`, `subs=[account_id/device_id/device_instance_id/+/+/set]`, and the standard claims `iss`, `iat`, and `exp`.
- REST and MCP parity means the same input and bearer token produce the same business fields, empty-result semantics, validation class, authorization class, and backend-failure class.

## Transport

- MCP is served over HTTP from the same `stutzthings-server` process.
- The planned endpoint is `/mcp` on the same port as the REST API and `/health`.
- Authentication is handled before tool execution and uses the same bearer-token parsing and topic-filter matcher as REST.
- The bearer token is conveyed on every HTTP request to `/mcp`; no session-scoped authentication state is assumed between requests.