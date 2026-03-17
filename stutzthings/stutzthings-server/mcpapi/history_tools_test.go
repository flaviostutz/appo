package mcpapi

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/auth"
	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/operations"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetNodeHistoryTool(t *testing.T) {
	handlers := &toolHandlers{service: operations.NewService(fakeStore{}, fakePublisher{}, nil)}
	ctx := auth.WithClaims(context.Background(), &auth.Claims{Publ: []string{"acct/device/instance/+/+"}})
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"account_id": "acct", "device_id": "device", "device_instance_id": "instance", "node_name": "node", "from": "2026-03-17T00:00:00Z", "to": "2026-03-17T01:00:00Z"}

	result, err := handlers.getNodeHistory(ctx, req)
	require.NoError(t, err)
	text, ok := mcp.AsTextContent(result.Content[0])
	require.True(t, ok)
	var response operations.HistoryResponse
	require.NoError(t, json.Unmarshal([]byte(text.Text), &response))
	assert.Len(t, response.Observations, 1)
}
