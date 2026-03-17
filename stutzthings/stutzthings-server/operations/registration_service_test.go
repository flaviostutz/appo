package operations

import (
	"context"
	"testing"
	"time"

	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterDeviceInstance(t *testing.T) {
	secret, err := auth.DecodeBase64Secret("ZGV2X3NlY3JldA==")
	require.NoError(t, err)
	signer, err := auth.NewSigner(secret, "issuer", time.Hour)
	require.NoError(t, err)
	service := NewService(fakeStore{query: func(string) ([]map[string]any, error) { return nil, nil }}, &fakePublisher{}, signer)
	claims := &auth.Claims{Publ: []string{"acct/device/+/+/+"}, Subs: []string{"acct/device/+/+/+/set"}}

	response, err := service.RegisterDeviceInstance(context.Background(), claims, RegistrationRequest{AccountID: "acct", DeviceID: "device"})
	require.NoError(t, err)
	assert.Equal(t, "acct", response.AccountID)
	assert.Equal(t, "device", response.DeviceID)
	assert.NotEmpty(t, response.DeviceInstanceID)
	assert.Equal(t, "issuer", response.Iss)
	assert.NotEmpty(t, response.Token)
	assert.Equal(t, []string{"acct/device/" + response.DeviceInstanceID + "/+/+"}, response.Publ)
	assert.Equal(t, []string{"acct/device/" + response.DeviceInstanceID + "/+/+/set"}, response.Subs)
}
