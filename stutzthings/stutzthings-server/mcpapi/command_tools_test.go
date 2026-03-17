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

func TestSetDesiredStateTool(t *testing.T) {
	handlers := &toolHandlers{service: operations.NewService(fakeStore{}, fakePublisher{}, nil)}
	ctx := auth.WithClaims(context.Background(), &auth.Claims{Subs: []string{"acct/device/instance/+/+/set"}})
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"account_id": "acct", "device_id": "device", "device_instance_id": "instance", "node_name": "node", "attribute_name": "temperature", "value": 23.5}

	result, err := handlers.setDesiredState(ctx, req)
	require.NoError(t, err)
	text, ok := mcp.AsTextContent(result.Content[0])
	require.True(t, ok)
	var response operations.DesiredStateResponse
	require.NoError(t, json.Unmarshal([]byte(text.Text), &response))
	assert.True(t, response.Accepted)
}
