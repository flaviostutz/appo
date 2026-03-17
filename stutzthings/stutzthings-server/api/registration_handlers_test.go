package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/auth"
	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/operations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterDeviceInstanceHandler(t *testing.T) {
	secret, err := auth.DecodeBase64Secret("ZGV2X3NlY3JldA==")
	require.NoError(t, err)
	signer, err := auth.NewSigner(secret, "issuer", time.Hour)
	require.NoError(t, err)
	handlers := NewHandlers(operations.NewService(fakeAPIStore{}, &operationsFakePublisher{}, signer))
	req := httptest.NewRequest(http.MethodPost, "/registration", bytes.NewBufferString(`{"accountId":"acct","deviceId":"device"}`))
	req = req.WithContext(auth.WithClaims(context.Background(), &auth.Claims{Publ: []string{"acct/device/+/+/+"}, Subs: []string{"acct/device/+/+/+/set"}}))
	res := httptest.NewRecorder()

	handlers.RegisterDeviceInstance(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
}
