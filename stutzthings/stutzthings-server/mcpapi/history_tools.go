package mcpapi

import (
	"context"

	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/operations"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (h *toolHandlers) registerHistoryTools(core *server.MCPServer) {
	core.AddTool(mcp.NewTool(
		"get_node_history",
		mcp.WithDescription("Read history for all authorized attributes under one node"),
		mcp.WithString("account_id", mcp.Required()),
		mcp.WithString("device_id", mcp.Required()),
		mcp.WithString("device_instance_id", mcp.Required()),
		mcp.WithString("node_name", mcp.Required()),
		mcp.WithString("from", mcp.Required()),
		mcp.WithString("to", mcp.Required()),
	), h.getNodeHistory)

	core.AddTool(mcp.NewTool(
		"get_attribute_history",
		mcp.WithDescription("Read history for one attribute"),
		mcp.WithString("account_id", mcp.Required()),
		mcp.WithString("device_id", mcp.Required()),
		mcp.WithString("device_instance_id", mcp.Required()),
		mcp.WithString("node_name", mcp.Required()),
		mcp.WithString("attribute_name", mcp.Required()),
		mcp.WithString("from", mcp.Required()),
		mcp.WithString("to", mcp.Required()),
	), h.getAttributeHistory)
}

func (h *toolHandlers) getNodeHistory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	fromValue, err := stringArg(req, "from")
	if err != nil {
		return jsonToolError(err)
	}
	toValue, err := stringArg(req, "to")
	if err != nil {
		return jsonToolError(err)
	}
	historyRange, err := operations.ParseHistoryRange(fromValue, toValue)
	if err != nil {
		return jsonToolError(err)
	}
	response, err := h.service.GetNodeHistory(ctx, claims, operations.NodeHistoryRequest{
		DeviceScope: operations.DeviceScope{AccountID: accountID, DeviceID: deviceID, DeviceInstanceID: deviceInstanceID},
		NodeName:    nodeName,
		From:        historyRange.From,
		To:          historyRange.To,
	})
	if err != nil {
		return jsonToolError(err)
	}
	return jsonToolResult(response)
}

func (h *toolHandlers) getAttributeHistory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	fromValue, err := stringArg(req, "from")
	if err != nil {
		return jsonToolError(err)
	}
	toValue, err := stringArg(req, "to")
	if err != nil {
		return jsonToolError(err)
	}
	historyRange, err := operations.ParseHistoryRange(fromValue, toValue)
	if err != nil {
		return jsonToolError(err)
	}
	response, err := h.service.GetAttributeHistory(ctx, claims, operations.AttributeHistoryRequest{
		DeviceIdentity: operations.DeviceIdentity{
			DeviceScope:   operations.DeviceScope{AccountID: accountID, DeviceID: deviceID, DeviceInstanceID: deviceInstanceID},
			NodeName:      nodeName,
			AttributeName: attributeName,
		},
		From: historyRange.From,
		To:   historyRange.To,
	})
	if err != nil {
		return jsonToolError(err)
	}
	return jsonToolResult(response)
}
