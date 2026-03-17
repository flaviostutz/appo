package operations

import (
	"fmt"
	"strings"
	"time"
)

func ValidateDeviceScope(scope DeviceScope) error {
	if err := validateSegment("accountId", scope.AccountID); err != nil {
		return err
	}
	if err := validateSegment("deviceId", scope.DeviceID); err != nil {
		return err
	}
	if err := validateSegment("deviceInstanceId", scope.DeviceInstanceID); err != nil {
		return err
	}
	return nil
}

func ValidateDeviceIdentity(identity DeviceIdentity) error {
	if err := ValidateDeviceScope(identity.DeviceScope); err != nil {
		return err
	}
	if err := validateSegment("nodeName", identity.NodeName); err != nil {
		return err
	}
	if err := validateSegment("attributeName", identity.AttributeName); err != nil {
		return err
	}
	return nil
}

func ValidateRegistrationRequest(req RegistrationRequest) error {
	if err := validateSegment("accountId", req.AccountID); err != nil {
		return err
	}
	if err := validateSegment("deviceId", req.DeviceID); err != nil {
		return err
	}
	return nil
}

func ParseHistoryRange(fromValue string, toValue string) (HistoryRange, error) {
	from, err := parseRFC3339UTC("from", fromValue)
	if err != nil {
		return HistoryRange{}, err
	}
	to, err := parseRFC3339UTC("to", toValue)
	if err != nil {
		return HistoryRange{}, err
	}
	if !from.Before(to) {
		return HistoryRange{}, invalidInput("from must be before to", nil)
	}
	return HistoryRange{From: from, To: to}, nil
}

func validateSegment(name string, value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return invalidInput(fmt.Sprintf("%s is required", name), nil)
	}
	if strings.Contains(trimmed, "/") || strings.Contains(trimmed, "+") || strings.Contains(trimmed, "#") {
		return invalidInput(fmt.Sprintf("%s must not contain '/', '+' or '#'", name), nil)
	}
	return nil
}

func parseRFC3339UTC(name string, value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, invalidInput(fmt.Sprintf("%s is required", name), nil)
	}
	if strings.Contains(trimmed, "since=") || !strings.HasSuffix(trimmed, "Z") {
		return time.Time{}, invalidInput(fmt.Sprintf("%s must be an RFC3339 UTC timestamp", name), nil)
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339Nano, trimmed)
	}
	if err != nil {
		return time.Time{}, invalidInput(fmt.Sprintf("%s must be an RFC3339 UTC timestamp", name), err)
	}
	return parsed.UTC(), nil
}
