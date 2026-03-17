package mcpapi

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/auth"
	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/operations"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterDeviceInstanceTool(t *testing.T) {
	secret, err := auth.DecodeBase64Secret("ZGV2X3NlY3JldA==")
	require.NoError(t, err)
	signer, err := auth.NewSigner(secret, "issuer", time.Hour)
	require.NoError(t, err)
	handlers := &toolHandlers{service: operations.NewService(fakeStore{}, fakePublisher{}, signer)}
	ctx := auth.WithClaims(context.Background(), &auth.Claims{Publ: []string{"acct/device/+/+/+"}, Subs: []string{"acct/device/+/+/+/set"}})
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"account_id": "acct", "device_id": "device"}

	result, err := handlers.registerDeviceInstance(ctx, req)
	require.NoError(t, err)
	text, ok := mcp.AsTextContent(result.Content[0])
	require.True(t, ok)
	var response operations.RegistrationResponse
	require.NoError(t, json.Unmarshal([]byte(text.Text), &response))
	assert.Equal(t, "acct", response.AccountID)
	assert.NotEmpty(t, response.Token)
}
