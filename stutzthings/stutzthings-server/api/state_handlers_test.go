package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/auth"
	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/operations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAttributeStateHandler(t *testing.T) {
	handlers := NewHandlers(operations.NewService(operationsFakeStore(), &operationsFakePublisher{}, nil))
	req := httptest.NewRequest(http.MethodGet, "/acct/device/instance/node/temperature", nil)
	req.SetPathValue("account_id", "acct")
	req.SetPathValue("device_id", "device")
	req.SetPathValue("device_instance_id", "instance")
	req.SetPathValue("node_name", "node")
	req.SetPathValue("attribute_name", "temperature")
	req = req.WithContext(auth.WithClaims(context.Background(), &auth.Claims{Publ: []string{"acct/device/instance/node/temperature"}}))
	res := httptest.NewRecorder()

	handlers.GetAttributeState(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	var response operations.AttributeStateResponse
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &response))
	assert.True(t, response.Found)
}

type operationsFakePublisher struct{}

func (o *operationsFakePublisher) Publish(context.Context, string, byte, bool, []byte) error {
	return nil
}

func operationsFakeStore() operations.QueryStore {
	return fakeAPIStore{}
}

type fakeAPIStore struct{}

func (fakeAPIStore) QueryRows(_ context.Context, query string) ([]map[string]any, error) {
	if strings.Contains(query, "SELECT device_instance_id") {
		return nil, nil
	}
	if query == "" {
		return nil, nil
	}
	return []map[string]any{makeObservationRow("node", "temperature", 21.5, "2026-03-17T00:00:00Z")}, nil
}

func makeObservationRow(node string, attribute string, value any, timestamp string) map[string]any {
	row := map[string]any{
		"account_id":         "acct",
		"device_id":          "device",
		"device_instance_id": "instance",
		"node_name":          node,
		"attribute_name":     attribute,
		"time":               timestamp,
	}
	row["value_float"] = value
	return row
}
