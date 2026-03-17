# MCP Tool Contract

The MCP surface must expose the same business capability as the REST API. Tool handlers call the same shared Go operations service used by the HTTP handlers and use the same JWT topic-filter authorization model already enforced by Mosquitto.

## Tools

| Tool | Purpose | Required Input | Authorization | Result |
|------|---------|----------------|---------------|--------|
| `get_attribute_state` | Read latest value for one attribute | `account_id`, `device_id`, `device_instance_id`, `node_name`, `attribute_name` | Telemetry topic matches at least one filter in `publ` or `subs` | One latest observation or `found=false` |
| `get_device_state` | Read latest state snapshot for one device instance | `account_id`, `device_id`, `device_instance_id` | Visible telemetry topics match at least one filter in `publ` or `subs` | Snapshot with `is_partial` and visible attributes only |
| `get_node_history` | Read history for all authorized attributes under one node | `account_id`, `device_id`, `device_instance_id`, `node_name`, `from`, `to` | Telemetry topics under the node match at least one filter in `publ` or `subs` | Ordered history result for the requested range |
| `get_attribute_history` | Read history for one attribute | `account_id`, `device_id`, `device_instance_id`, `node_name`, `attribute_name`, `from`, `to` | Telemetry topic matches at least one filter in `publ` or `subs` | Ordered history result for the requested range |
| `set_desired_state` | Publish desired value to the device `/set` topic | `account_id`, `device_id`, `device_instance_id`, `node_name`, `attribute_name`, `value` | `/set` topic matches at least one filter in `subs` | `accepted=true` and published topic |
| `register_device_instance` | Create a new device instance and return one scoped credential | `account_id`, `device_id` | Wildcard `publ` and `subs` filters cover the requested account/device namespace | `device_instance_id`, JWT token, `sub`, `publ`, `subs` |

## Shared Rules

- MCP errors must mirror REST business failures: invalid input, unauthorized, forbidden, not found/empty result, and backend failures.
- Tool inputs use the same field names and validation rules as the REST contract.
- `get_device_state` must return `is_partial=true` when authorization filtered out stored attributes.
- History tools must reject missing, unparseable, or inverted `from`/`to` timestamps.
- `set_desired_state` acknowledges publication only; it does not imply the device has reported a new actual state.
- `register_device_instance` returns one JWT with `sub=account_id/device_id/device_instance_id`, `publ=[account_id/device_id/device_instance_id/+/+]`, and `subs=[account_id/device_id/device_instance_id/+/+/set]`.

## Transport

- MCP is served over HTTP from the same `stutzthings-server` process.
- The planned endpoint is `/mcp` on the same port as the REST API and `/health`.
- Authentication is handled before tool execution and uses the same bearer-token parsing and topic-filter matcher as REST.