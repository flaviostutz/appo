package operations

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeStore struct {
	query func(string) ([]map[string]any, error)
}

func (f fakeStore) QueryRows(_ context.Context, query string) ([]map[string]any, error) {
	return f.query(query)
}

type fakePublisher struct {
	topic   string
	payload []byte
	err     error
}

func (f *fakePublisher) Publish(_ context.Context, topic string, _ byte, _ bool, payload []byte) error {
	f.topic = topic
	f.payload = append([]byte(nil), payload...)
	return f.err
}

func testClaims() *auth.Claims {
	return &auth.Claims{Publ: []string{"acct/device/instance/node/temperature", "acct/device/instance/node/humidity", "acct/device/instance/+/+"}, Subs: []string{"acct/device/instance/+/+/set"}}
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
	switch typed := value.(type) {
	case float64:
		row["value_float"] = typed
	case bool:
		row["value_bool"] = typed
	case string:
		row["value_string"] = typed
	}
	return row
}

func TestGetDeviceStatePartialAndEmpty(t *testing.T) {
	service := NewService(fakeStore{query: func(query string) ([]map[string]any, error) {
		if strings.Contains(query, "ORDER BY time DESC") {
			return []map[string]any{
				makeObservationRow("node", "temperature", 21.5, "2026-03-17T00:00:00Z"),
				makeObservationRow("node", "pressure", 101.0, "2026-03-17T00:00:00Z"),
			}, nil
		}
		return nil, nil
	}}, &fakePublisher{}, nil)

	response, err := service.GetDeviceState(context.Background(), &auth.Claims{Publ: []string{"acct/device/instance/node/temperature"}}, DeviceScope{AccountID: "acct", DeviceID: "device", DeviceInstanceID: "instance"})
	require.NoError(t, err)
	assert.Len(t, response.Attributes, 1)
	assert.True(t, response.IsPartial)
	assert.Equal(t, 1, response.ExcludedAttributeCount)

	emptyService := NewService(fakeStore{query: func(string) ([]map[string]any, error) { return nil, nil }}, &fakePublisher{}, nil)
	empty, err := emptyService.GetDeviceState(context.Background(), testClaims(), DeviceScope{AccountID: "acct", DeviceID: "device", DeviceInstanceID: "instance"})
	require.NoError(t, err)
	assert.Empty(t, empty.Attributes)
	assert.False(t, empty.IsPartial)
	assert.Equal(t, 0, empty.ExcludedAttributeCount)
}

func TestGetAttributeState(t *testing.T) {
	service := NewService(fakeStore{query: func(string) ([]map[string]any, error) {
		return []map[string]any{makeObservationRow("node", "temperature", 21.5, "2026-03-17T00:00:00Z")}, nil
	}}, &fakePublisher{}, nil)

	response, err := service.GetAttributeState(context.Background(), testClaims(), DeviceIdentity{DeviceScope: DeviceScope{AccountID: "acct", DeviceID: "device", DeviceInstanceID: "instance"}, NodeName: "node", AttributeName: "temperature"})
	require.NoError(t, err)
	assert.True(t, response.Found)
	assert.Equal(t, 21.5, response.Observation.Value)
	assert.Equal(t, time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC), response.Observation.ObservedAt)
}
