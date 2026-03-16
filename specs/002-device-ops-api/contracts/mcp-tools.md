# MCP Tool Contract

The MCP surface must expose the same business capability as the REST API. Tool handlers call the same shared Go operations service used by the HTTP handlers.

## Tools

| Tool | Purpose | Required Input | Authorization | Result |
|------|---------|----------------|---------------|--------|
| `get_attribute_state` | Read latest value for one attribute | `account_id`, `device_id`, `device_instance_id`, `node_name`, `attribute_name` | Matching `i:...:r` scope | One latest observation or `found=false` |
| `get_device_state` | Read latest state snapshot for one device instance | `account_id`, `device_id`, `device_instance_id` | Matching `i:...:r` scopes for visible attributes | Snapshot with `is_partial` and visible attributes only |
| `get_node_history` | Read history for all authorized attributes under one node | `account_id`, `device_id`, `device_instance_id`, `node_name`, `from`, `to`, optional `cursor` | Matching `i:...:r` scopes | Ordered history page plus optional `next_cursor` |
| `get_attribute_history` | Read history for one attribute | `account_id`, `device_id`, `device_instance_id`, `node_name`, `attribute_name`, `from`, `to`, optional `cursor` | Matching `i:...:r` scope | Ordered history page plus optional `next_cursor` |
| `set_desired_state` | Publish desired value to the device `/set` topic | `account_id`, `device_id`, `device_instance_id`, `node_name`, `attribute_name`, `value` | Matching `i:...:s` scope | `accepted=true` and published topic |
| `register_device_instance` | Create a new device instance and return one scoped credential | `account_id`, `device_id` | Matching `r:account_id/device_id` scope | `device_instance_id`, JWT token, issued scope |

## Shared Rules

- MCP errors must mirror REST business failures: invalid input, unauthorized, forbidden, not found/empty result, and backend failures.
- Tool inputs use the same field names and validation rules as the REST contract.
- `get_device_state` must return `is_partial=true` when authorization filtered out stored attributes.
- History tools must reject missing, unparseable, or inverted `from`/`to` timestamps.
- `set_desired_state` acknowledges publication only; it does not imply the device has reported a new actual state.
- `register_device_instance` returns one JWT scoped to `i:account_id/device_id/device_instance_id/*/*:rws`.

## Transport

- MCP is served over HTTP from the same `stutzthings-server` process.
- The planned endpoint is `/mcp` on the same port as the REST API and `/health`.
- Authentication is handled before tool execution and uses the same bearer-token parsing and scope matcher as REST.