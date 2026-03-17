package operations

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

func buildLatestAttributeQuery(identity DeviceIdentity) string {
	return fmt.Sprintf(
		"SELECT * FROM device_attributes WHERE account_id = '%s' AND device_id = '%s' AND device_instance_id = '%s' AND node_name = '%s' AND attribute_name = '%s' ORDER BY time DESC LIMIT 1",
		escapeLiteral(identity.AccountID), escapeLiteral(identity.DeviceID), escapeLiteral(identity.DeviceInstanceID), escapeLiteral(identity.NodeName), escapeLiteral(identity.AttributeName),
	)
}

func buildLatestDeviceQuery(scope DeviceScope) string {
	return fmt.Sprintf(
		"SELECT * FROM device_attributes WHERE account_id = '%s' AND device_id = '%s' AND device_instance_id = '%s' ORDER BY time DESC",
		escapeLiteral(scope.AccountID), escapeLiteral(scope.DeviceID), escapeLiteral(scope.DeviceInstanceID),
	)
}

func buildNodeHistoryQuery(request NodeHistoryRequest, limit int) string {
	return fmt.Sprintf(
		"SELECT * FROM device_attributes WHERE account_id = '%s' AND device_id = '%s' AND device_instance_id = '%s' AND node_name = '%s' AND time >= TIMESTAMP '%s' AND time < TIMESTAMP '%s' ORDER BY time ASC, node_name ASC, attribute_name ASC LIMIT %d",
		escapeLiteral(request.AccountID), escapeLiteral(request.DeviceID), escapeLiteral(request.DeviceInstanceID), escapeLiteral(request.NodeName), request.From.UTC().Format(time.RFC3339Nano), request.To.UTC().Format(time.RFC3339Nano), limit,
	)
}

func buildAttributeHistoryQuery(request AttributeHistoryRequest, limit int) string {
	return fmt.Sprintf(
		"SELECT * FROM device_attributes WHERE account_id = '%s' AND device_id = '%s' AND device_instance_id = '%s' AND node_name = '%s' AND attribute_name = '%s' AND time >= TIMESTAMP '%s' AND time < TIMESTAMP '%s' ORDER BY time ASC LIMIT %d",
		escapeLiteral(request.AccountID), escapeLiteral(request.DeviceID), escapeLiteral(request.DeviceInstanceID), escapeLiteral(request.NodeName), escapeLiteral(request.AttributeName), request.From.UTC().Format(time.RFC3339Nano), request.To.UTC().Format(time.RFC3339Nano), limit,
	)
}

func buildRegistrationCollisionQuery(accountID string, deviceID string, deviceInstanceID string) string {
	return fmt.Sprintf(
		"SELECT device_instance_id FROM device_attributes WHERE account_id = '%s' AND device_id = '%s' AND device_instance_id = '%s' LIMIT 1",
		escapeLiteral(accountID), escapeLiteral(deviceID), escapeLiteral(deviceInstanceID),
	)
}

func decodeObservationRow(row map[string]any) (Observation, error) {
	observedAt, err := decodeRowTime(row)
	if err != nil {
		return Observation{}, err
	}
	value, err := decodeRowValue(row)
	if err != nil {
		return Observation{}, err
	}
	return Observation{
		AccountID:        stringValue(row["account_id"]),
		DeviceID:         stringValue(row["device_id"]),
		DeviceInstanceID: stringValue(row["device_instance_id"]),
		NodeName:         stringValue(row["node_name"]),
		AttributeName:    stringValue(row["attribute_name"]),
		ObservedAt:       observedAt,
		Value:            value,
	}, nil
}

func sortObservations(observations []Observation) {
	sort.Slice(observations, func(i int, j int) bool {
		if observations[i].ObservedAt.Equal(observations[j].ObservedAt) {
			if observations[i].NodeName == observations[j].NodeName {
				if observations[i].AttributeName == observations[j].AttributeName {
					return observations[i].CanonicalValueString() < observations[j].CanonicalValueString()
				}
				return observations[i].AttributeName < observations[j].AttributeName
			}
			return observations[i].NodeName < observations[j].NodeName
		}
		return observations[i].ObservedAt.Before(observations[j].ObservedAt)
	})
}

func decodeRowTime(row map[string]any) (time.Time, error) {
	for _, key := range []string{"time", "timestamp"} {
		if value, ok := row[key]; ok {
			switch typed := value.(type) {
			case time.Time:
				return typed.UTC(), nil
			case string:
				parsed, err := time.Parse(time.RFC3339Nano, typed)
				if err == nil {
					return parsed.UTC(), nil
				}
			}
		}
	}
	return time.Time{}, fmt.Errorf("decode observation time")
}

func decodeRowValue(row map[string]any) (any, error) {
	if value, ok := row["value_float"]; ok && value != nil {
		switch typed := value.(type) {
		case float64:
			return typed, nil
		case int64:
			return float64(typed), nil
		case string:
			parsed, err := strconv.ParseFloat(typed, 64)
			if err == nil {
				return parsed, nil
			}
		}
	}
	if value, ok := row["value_bool"]; ok && value != nil {
		switch typed := value.(type) {
		case bool:
			return typed, nil
		case string:
			parsed, err := strconv.ParseBool(typed)
			if err == nil {
				return parsed, nil
			}
		}
	}
	if value, ok := row["value_string"]; ok && value != nil {
		return stringValue(value), nil
	}
	return nil, fmt.Errorf("decode observation value")
}

func escapeLiteral(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprintf("%v", typed)
	}
}
