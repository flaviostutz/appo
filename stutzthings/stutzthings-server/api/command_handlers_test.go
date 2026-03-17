package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/auth"
	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/operations"
	"github.com/stretchr/testify/assert"
)

func TestSetDesiredStateHandler(t *testing.T) {
	handlers := NewHandlers(operations.NewService(operationsFakeStore(), &operationsFakePublisher{}, nil))
	req := httptest.NewRequest(http.MethodPut, "/acct/device/instance/node/temperature", bytes.NewBufferString(`{"value":23.5}`))
	req.SetPathValue("account_id", "acct")
	req.SetPathValue("device_id", "device")
	req.SetPathValue("device_instance_id", "instance")
	req.SetPathValue("node_name", "node")
	req.SetPathValue("attribute_name", "temperature")
	req = req.WithContext(auth.WithClaims(context.Background(), &auth.Claims{Subs: []string{"acct/device/instance/+/+/set"}}))
	res := httptest.NewRecorder()

	handlers.SetDesiredState(res, req)

	assert.Equal(t, http.StatusAccepted, res.Code)
}
