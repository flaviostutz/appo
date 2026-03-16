package bridge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type discardReason string

const (
	reasonInvalidTopic    discardReason = "invalid_topic"
	reasonSetTopicIgnored discardReason = "set_topic_ignored"
	reasonEmptyPayload    discardReason = "empty_payload"
	reasonInvalidJSON     discardReason = "invalid_json"
	reasonMissingValue    discardReason = "missing_value"
	reasonNullValue       discardReason = "null_value"
)

func ParseDeviceAttribute(message MQTTMessage) (DeviceAttribute, discardReason, error) {
	accountID, deviceID, instanceID, nodeName, attributeName, reason, err := parseTopic(message.Topic)
	if err != nil {
		return DeviceAttribute{}, reason, err
	}

	timestamp := message.ReceivedAt
	floatValue, stringValue, boolValue, reason, err := parsePayload(message.Payload, &timestamp)
	if err != nil {
		return DeviceAttribute{}, reason, err
	}

	return DeviceAttribute{
		AccountID:        accountID,
		DeviceID:         deviceID,
		DeviceInstanceID: instanceID,
		NodeName:         nodeName,
		AttributeName:    attributeName,
		ValueFloat:       floatValue,
		ValueString:      stringValue,
		ValueBool:        boolValue,
		Timestamp:        timestamp,
	}, "", nil
}

func samplePayload(payload []byte) string {
	trimmed := strings.TrimSpace(string(payload))
	if len(trimmed) <= 120 {
		return trimmed
	}
	return trimmed[:120] + "..."
}

func parseTopic(topic string) (string, string, string, string, string, discardReason, error) {
	parts := strings.Split(topic, "/")
	if len(parts) == 6 && parts[5] == "set" {
		return "", "", "", "", "", reasonSetTopicIgnored, fmt.Errorf("set topics are not ingested")
	}
	if len(parts) != 5 {
		return "", "", "", "", "", reasonInvalidTopic, fmt.Errorf("expected 5 topic segments, got %d", len(parts))
	}
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return "", "", "", "", "", reasonInvalidTopic, fmt.Errorf("topic contains empty segment")
		}
	}

	return parts[0], parts[1], parts[2], parts[3], parts[4], "", nil
}

func parsePayload(payload []byte, timestamp *time.Time) (*float64, *string, *bool, discardReason, error) {
	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil, nil, nil, reasonEmptyPayload, fmt.Errorf("payload is empty")
	}
	if trimmed[0] == '{' {
		return parseJSONPayload(trimmed, timestamp)
	}
	return parseScalarString(string(trimmed))
}

func parseJSONPayload(payload []byte, timestamp *time.Time) (*float64, *string, *bool, discardReason, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()

	var document map[string]json.RawMessage
	if err := decoder.Decode(&document); err != nil {
		return nil, nil, nil, reasonInvalidJSON, fmt.Errorf("decode json payload: %w", err)
	}

	valueRaw, ok := document["value"]
	if !ok {
		return nil, nil, nil, reasonMissingValue, fmt.Errorf("json payload missing value field")
	}
	if bytes.Equal(bytes.TrimSpace(valueRaw), []byte("null")) {
		return nil, nil, nil, reasonNullValue, fmt.Errorf("json payload value is null")
	}

	if tsRaw, ok := document["ts"]; ok && !bytes.Equal(bytes.TrimSpace(tsRaw), []byte("null")) {
		var ts json.Number
		if err := json.Unmarshal(tsRaw, &ts); err != nil {
			return nil, nil, nil, reasonInvalidJSON, fmt.Errorf("decode ts field: %w", err)
		}
		tsInt, err := ts.Int64()
		if err != nil {
			return nil, nil, nil, reasonInvalidJSON, fmt.Errorf("ts must be an integer milliseconds epoch: %w", err)
		}
		*timestamp = time.UnixMilli(tsInt).UTC()
	}

	var decoded any
	if err := json.Unmarshal(valueRaw, &decoded); err != nil {
		return nil, nil, nil, reasonInvalidJSON, fmt.Errorf("decode value field: %w", err)
	}

	switch value := decoded.(type) {
	case bool:
		return nil, nil, boolPtr(value), "", nil
	case float64:
		return floatPtr(value), nil, nil, "", nil
	case string:
		return nil, stringPtr(value), nil, "", nil
	case nil:
		return nil, nil, nil, reasonNullValue, fmt.Errorf("json payload value is null")
	default:
		return nil, nil, nil, reasonInvalidJSON, fmt.Errorf("unsupported json value type %T", value)
	}
}

func parseScalarString(value string) (*float64, *string, *bool, discardReason, error) {
	trimmed := strings.TrimSpace(value)
	if parsedBool, err := strconv.ParseBool(trimmed); err == nil {
		return nil, nil, boolPtr(parsedBool), "", nil
	}
	if parsedInt, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return floatPtr(float64(parsedInt)), nil, nil, "", nil
	}
	if parsedFloat, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return floatPtr(parsedFloat), nil, nil, "", nil
	}
	return nil, stringPtr(trimmed), nil, "", nil
}

func floatPtr(value float64) *float64 {
	return &value
}

func stringPtr(value string) *string {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}
