package mcpapi

import (
	"context"
	"encoding/json"

	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/operations"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (h *toolHandlers) registerCommandTools(core *server.MCPServer) {
	rawSchema := json.RawMessage(`{"type":"object","properties":{"account_id":{"type":"string"},"device_id":{"type":"string"},"device_instance_id":{"type":"string"},"node_name":{"type":"string"},"attribute_name":{"type":"string"},"value":{}},"required":["account_id","device_id","device_instance_id","node_name","attribute_name","value"]}`)
	core.AddTool(mcp.NewToolWithRawSchema("set_desired_state", "Publish desired value to the device /set topic", rawSchema), h.setDesiredState)
}

func (h *toolHandlers) setDesiredState(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	claims, err := claimsFromContext(ctx)
	if err != nil {
		return jsonToolError(err)
	}
	accountID, err := stringArg(req, "account_id")
	if err != nil {
		return jsonToolError(err)
	}
	deviceID, err := stringArg(req, "device_id")
	if err != nil {
		return jsonToolError(err)
	}
	deviceInstanceID, err := stringArg(req, "device_instance_id")
	if err != nil {
		return jsonToolError(err)
	}
	nodeName, err := stringArg(req, "node_name")
	if err != nil {
		return jsonToolError(err)
	}
	attributeName, err := stringArg(req, "attribute_name")
	if err != nil {
		return jsonToolError(err)
	}
	response, err := h.service.SetDesiredState(ctx, claims, operations.DeviceIdentity{
		DeviceScope:   operations.DeviceScope{AccountID: accountID, DeviceID: deviceID, DeviceInstanceID: deviceInstanceID},
		NodeName:      nodeName,
		AttributeName: attributeName,
	}, operations.DesiredStateRequest{Value: req.Params.Arguments["value"]})
	if err != nil {
		return jsonToolError(err)
	}
	return jsonToolResult(response)
}
