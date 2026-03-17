package mcpapi

import (
	"context"

	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/operations"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (h *toolHandlers) registerRegistrationTools(core *server.MCPServer) {
	core.AddTool(mcp.NewTool(
		"register_device_instance",
		mcp.WithDescription("Create a new device instance and return one scoped credential"),
		mcp.WithString("account_id", mcp.Required()),
		mcp.WithString("device_id", mcp.Required()),
	), h.registerDeviceInstance)
}

func (h *toolHandlers) registerDeviceInstance(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	response, err := h.service.RegisterDeviceInstance(ctx, claims, operations.RegistrationRequest{AccountID: accountID, DeviceID: deviceID})
	if err != nil {
		return jsonToolError(err)
	}
	return jsonToolResult(response)
}
