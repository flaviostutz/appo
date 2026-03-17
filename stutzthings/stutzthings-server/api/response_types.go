package api

import "github.com/flaviostutz/appo/stutzthings/stutzthings-server/operations"

type AttributeStateResponse = operations.AttributeStateResponse
type DeviceStateResponse = operations.DeviceStateResponse
type HistoryResponse = operations.HistoryResponse
type DesiredStateResponse = operations.DesiredStateResponse
type RegistrationResponse = operations.RegistrationResponse

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
