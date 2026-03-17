package operations

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAttributeHistoryOrdered(t *testing.T) {
	service := NewService(fakeStore{query: func(string) ([]map[string]any, error) {
		return []map[string]any{
			makeObservationRow("node", "temperature", 22.0, "2026-03-17T00:01:00Z"),
			makeObservationRow("node", "temperature", 21.0, "2026-03-17T00:00:00Z"),
		}, nil
	}}, &fakePublisher{}, nil)

	response, err := service.GetAttributeHistory(context.Background(), testClaims(), AttributeHistoryRequest{
		DeviceIdentity: DeviceIdentity{DeviceScope: DeviceScope{AccountID: "acct", DeviceID: "device", DeviceInstanceID: "instance"}, NodeName: "node", AttributeName: "temperature"},
		From:           time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC),
		To:             time.Date(2026, 3, 17, 1, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	assert.Len(t, response.Observations, 2)
	assert.Equal(t, 21.0, response.Observations[0].Value)
	assert.Equal(t, 22.0, response.Observations[1].Value)
}
