package bridge

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDeviceAttributeRawScalarTypes(t *testing.T) {
	receivedAt := time.UnixMilli(1741910400000).UTC()

	tests := []struct {
		name    string
		payload string
		assert  func(t *testing.T, attribute DeviceAttribute)
	}{
		{
			name:    "float",
			payload: "23.5",
			assert: func(t *testing.T, attribute DeviceAttribute) {
				require.NotNil(t, attribute.ValueFloat)
				assert.Equal(t, 23.5, *attribute.ValueFloat)
			},
		},
		{
			name:    "boolean",
			payload: "true",
			assert: func(t *testing.T, attribute DeviceAttribute) {
				require.NotNil(t, attribute.ValueBool)
				assert.True(t, *attribute.ValueBool)
			},
		},
		{
			name:    "string",
			payload: "on",
			assert: func(t *testing.T, attribute DeviceAttribute) {
				require.NotNil(t, attribute.ValueString)
				assert.Equal(t, "on", *attribute.ValueString)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			attribute, reason, err := ParseDeviceAttribute(MQTTMessage{
				Topic:      "account1/deviceA/device01/node/attribute",
				Payload:    []byte(testCase.payload),
				ReceivedAt: receivedAt,
			})
			require.NoError(t, err)
			assert.Empty(t, reason)
			assert.Equal(t, receivedAt, attribute.Timestamp)
			assert.Equal(t, "account1", attribute.AccountID)
			testCase.assert(t, attribute)
		})
	}
}

func TestParseDeviceAttributeJSONPayload(t *testing.T) {
	attribute, reason, err := ParseDeviceAttribute(MQTTMessage{
		Topic:      "account1/deviceA/device01/node/attribute",
		Payload:    []byte(`{"value":65.2,"ts":1741910400000}`),
		ReceivedAt: time.UnixMilli(1741910500000).UTC(),
	})

	require.NoError(t, err)
	assert.Empty(t, reason)
	require.NotNil(t, attribute.ValueFloat)
	assert.Equal(t, 65.2, *attribute.ValueFloat)
	assert.Equal(t, time.UnixMilli(1741910400000).UTC(), attribute.Timestamp)
}

func TestParseDeviceAttributeInvalidInputs(t *testing.T) {
	tests := []struct {
		name    string
		topic   string
		payload string
		reason  discardReason
	}{
		{name: "invalid topic", topic: "a/b/c", payload: "1", reason: reasonInvalidTopic},
		{name: "set topic", topic: "a/b/c/d/e/set", payload: "1", reason: reasonSetTopicIgnored},
		{name: "empty payload", topic: "a/b/c/d/e", payload: " ", reason: reasonEmptyPayload},
		{name: "missing value", topic: "a/b/c/d/e", payload: `{}`, reason: reasonMissingValue},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			_, reason, err := ParseDeviceAttribute(MQTTMessage{
				Topic:      testCase.topic,
				Payload:    []byte(testCase.payload),
				ReceivedAt: time.Now().UTC(),
			})
			require.Error(t, err)
			assert.Equal(t, testCase.reason, reason)
		})
	}
}
