package mcpapi

import (
	"context"

	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/operations"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (h *toolHandlers) registerStateTools(core *server.MCPServer) {
	core.AddTool(mcp.NewTool(
		"get_attribute_state",
		mcp.WithDescription("Read latest value for one attribute"),
		mcp.WithString("account_id", mcp.Required()),
		mcp.WithString("device_id", mcp.Required()),
		mcp.WithString("device_instance_id", mcp.Required()),
		mcp.WithString("node_name", mcp.Required()),
		mcp.WithString("attribute_name", mcp.Required()),
	), h.getAttributeState)

	core.AddTool(mcp.NewTool(
		"get_device_state",
		mcp.WithDescription("Read latest state snapshot for one device instance"),
		mcp.WithString("account_id", mcp.Required()),
		mcp.WithString("device_id", mcp.Required()),
		mcp.WithString("device_instance_id", mcp.Required()),
	), h.getDeviceState)
}

func (h *toolHandlers) getAttributeState(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	response, err := h.service.GetAttributeState(ctx, claims, operations.DeviceIdentity{
		DeviceScope:   operations.DeviceScope{AccountID: accountID, DeviceID: deviceID, DeviceInstanceID: deviceInstanceID},
		NodeName:      nodeName,
		AttributeName: attributeName,
	})
	if err != nil {
		return jsonToolError(err)
	}
	return jsonToolResult(response)
}

func (h *toolHandlers) getDeviceState(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	response, err := h.service.GetDeviceState(ctx, claims, operations.DeviceScope{AccountID: accountID, DeviceID: deviceID, DeviceInstanceID: deviceInstanceID})
	if err != nil {
		return jsonToolError(err)
	}
	return jsonToolResult(response)
}
