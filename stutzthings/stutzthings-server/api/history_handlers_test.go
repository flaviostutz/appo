package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/auth"
	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/operations"
	"github.com/stretchr/testify/assert"
)

func TestGetNodeHistoryRejectsInvalidRange(t *testing.T) {
	handlers := NewHandlers(operations.NewService(operationsFakeStore(), &operationsFakePublisher{}, nil))
	req := httptest.NewRequest(http.MethodGet, "/acct/device/instance/node/history?from=bad&to=2026-03-17T01:00:00Z", nil)
	req.SetPathValue("account_id", "acct")
	req.SetPathValue("device_id", "device")
	req.SetPathValue("device_instance_id", "instance")
	req.SetPathValue("node_name", "node")
	req = req.WithContext(auth.WithClaims(context.Background(), &auth.Claims{Publ: []string{"acct/device/instance/+/+"}}))
	res := httptest.NewRecorder()

	handlers.GetNodeHistory(res, req)

	assert.Equal(t, http.StatusBadRequest, res.Code)
}
