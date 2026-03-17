package operations

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ErrorCode string

const (
	CodeInvalidInput   ErrorCode = "invalid_input"
	CodeWindowTooLarge ErrorCode = "window_too_large"
	CodeUnauthorized   ErrorCode = "unauthorized"
	CodeForbidden      ErrorCode = "forbidden"
	CodeBackendFailure ErrorCode = "backend_failure"

	telemetryTopicSegments = 5
	commandTopicSegments   = 6
	historyResultLimit     = 10000
)

type Error struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e *Error) HTTPStatus() int {
	switch e.Code {
	case CodeInvalidInput, CodeWindowTooLarge:
		return http.StatusBadRequest
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

type DeviceScope struct {
	AccountID        string `json:"accountId"`
	DeviceID         string `json:"deviceId"`
	DeviceInstanceID string `json:"deviceInstanceId"`
}

type DeviceIdentity struct {
	DeviceScope
	NodeName      string `json:"nodeName"`
	AttributeName string `json:"attributeName"`
}

type NodeHistoryRequest struct {
	DeviceScope
	NodeName string    `json:"nodeName"`
	From     time.Time `json:"from"`
	To       time.Time `json:"to"`
}

type AttributeHistoryRequest struct {
	DeviceIdentity
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

type HistoryRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

type Observation struct {
	AccountID        string    `json:"accountId"`
	DeviceID         string    `json:"deviceId"`
	DeviceInstanceID string    `json:"deviceInstanceId"`
	NodeName         string    `json:"nodeName"`
	AttributeName    string    `json:"attributeName"`
	ObservedAt       time.Time `json:"observedAt"`
	Value            any       `json:"value"`
}

func (o Observation) CanonicalValueString() string {
	switch value := o.Value.(type) {
	case nil:
		return ""
	case string:
		return value
	case bool:
		return strconv.FormatBool(value)
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(value), 'f', -1, 64)
	case int:
		return strconv.Itoa(value)
	case int64:
		return strconv.FormatInt(value, 10)
	default:
		return fmt.Sprintf("%v", value)
	}
}

type AttributeStateResponse struct {
	Found       bool         `json:"found"`
	Observation *Observation `json:"observation"`
}

type DeviceStateResponse struct {
	AccountID              string        `json:"accountId"`
	DeviceID               string        `json:"deviceId"`
	DeviceInstanceID       string        `json:"deviceInstanceId"`
	IsPartial              bool          `json:"isPartial"`
	ExcludedAttributeCount int           `json:"excludedAttributeCount"`
	Attributes             []Observation `json:"attributes"`
}

type HistoryResponse struct {
	From         time.Time     `json:"from"`
	To           time.Time     `json:"to"`
	Observations []Observation `json:"observations"`
}

type DesiredStateRequest struct {
	Value any `json:"value"`
}

type DesiredStateResponse struct {
	Accepted bool   `json:"accepted"`
	Topic    string `json:"topic"`
}

type RegistrationRequest struct {
	AccountID string `json:"accountId"`
	DeviceID  string `json:"deviceId"`
}

type RegistrationResponse struct {
	AccountID        string    `json:"accountId"`
	DeviceID         string    `json:"deviceId"`
	DeviceInstanceID string    `json:"deviceInstanceId"`
	Sub              string    `json:"sub"`
	Publ             []string  `json:"publ"`
	Subs             []string  `json:"subs"`
	Iss              string    `json:"iss"`
	IssuedAt         time.Time `json:"issuedAt"`
	ExpiresAt        time.Time `json:"expiresAt"`
	Token            string    `json:"token"`
}

func (d DeviceScope) TelemetryPrefix() []string {
	return []string{d.AccountID, d.DeviceID, d.DeviceInstanceID}
}

func (d DeviceScope) TelemetryTopic() string {
	return strings.Join(d.TelemetryPrefix(), "/")
}

func (d DeviceIdentity) TelemetryTopic() string {
	return strings.Join([]string{d.AccountID, d.DeviceID, d.DeviceInstanceID, d.NodeName, d.AttributeName}, "/")
}

func (d DeviceIdentity) CommandTopic() string {
	return d.TelemetryTopic() + "/set"
}

func invalidInput(message string, err error) *Error {
	return &Error{Code: CodeInvalidInput, Message: message, Err: err}
}

func InvalidInput(message string, err error) *Error {
	return invalidInput(message, err)
}

func windowTooLarge(message string) *Error {
	return &Error{Code: CodeWindowTooLarge, Message: message}
}

func unauthorized(message string, err error) *Error {
	return &Error{Code: CodeUnauthorized, Message: message, Err: err}
}

func Unauthorized(message string, err error) *Error {
	return unauthorized(message, err)
}

func forbidden(message string) *Error {
	return &Error{Code: CodeForbidden, Message: message}
}

func backendFailure(message string, err error) *Error {
	return &Error{Code: CodeBackendFailure, Message: message, Err: err}
}

func AsError(err error) *Error {
	if err == nil {
		return nil
	}
	var opErr *Error
	if errors.As(err, &opErr) {
		return opErr
	}
	return nil
}
