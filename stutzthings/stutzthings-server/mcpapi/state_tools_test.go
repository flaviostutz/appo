package mcpapi

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/auth"
	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/operations"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeStore struct{}

func (fakeStore) QueryRows(_ context.Context, query string) ([]map[string]any, error) {
	if strings.Contains(query, "SELECT device_instance_id") {
		return nil, nil
	}
	return []map[string]any{{
		"account_id":         "acct",
		"device_id":          "device",
		"device_instance_id": "instance",
		"node_name":          "node",
		"attribute_name":     "temperature",
		"value_float":        21.5,
		"time":               "2026-03-17T00:00:00Z",
	}}, nil
}

type fakePublisher struct{}

func (fakePublisher) Publish(context.Context, string, byte, bool, []byte) error { return nil }

func TestGetAttributeStateTool(t *testing.T) {
	handlers := &toolHandlers{service: operations.NewService(fakeStore{}, fakePublisher{}, nil)}
	ctx := auth.WithClaims(context.Background(), &auth.Claims{Publ: []string{"acct/device/instance/node/temperature"}})
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"account_id": "acct", "device_id": "device", "device_instance_id": "instance", "node_name": "node", "attribute_name": "temperature"}

	result, err := handlers.getAttributeState(ctx, req)
	require.NoError(t, err)
	text, ok := mcp.AsTextContent(result.Content[0])
	require.True(t, ok)
	var response operations.AttributeStateResponse
	require.NoError(t, json.Unmarshal([]byte(text.Text), &response))
	assert.True(t, response.Found)
}
