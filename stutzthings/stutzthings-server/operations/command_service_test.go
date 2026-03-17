package operations

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetDesiredStatePublishesSetTopic(t *testing.T) {
	publisher := &fakePublisher{}
	service := NewService(fakeStore{query: func(string) ([]map[string]any, error) { return nil, nil }}, publisher, nil)

	response, err := service.SetDesiredState(context.Background(), testClaims(), DeviceIdentity{DeviceScope: DeviceScope{AccountID: "acct", DeviceID: "device", DeviceInstanceID: "instance"}, NodeName: "node", AttributeName: "temperature"}, DesiredStateRequest{Value: 23.5})
	require.NoError(t, err)
	assert.True(t, response.Accepted)
	assert.Equal(t, "acct/device/instance/node/temperature/set", publisher.topic)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(publisher.payload, &payload))
	assert.Equal(t, 23.5, payload["value"])
}
