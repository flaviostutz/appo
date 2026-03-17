package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/auth"
	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/operations"
)

type Handlers struct {
	service *operations.Service
}

func NewHandlers(service *operations.Service) Handlers {
	return Handlers{service: service}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeOperationError(w http.ResponseWriter, err error) {
	var opErr *operations.Error
	if errors.As(err, &opErr) {
		writeJSON(w, opErr.HTTPStatus(), ErrorResponse{Error: string(opErr.Code), Message: opErr.Message})
		return
	}
	writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: string(operations.CodeBackendFailure), Message: err.Error()})
}

func decodeJSONBody(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return operations.InvalidInput("invalid json body", err)
	}
	return nil
}

func claimsFromRequest(r *http.Request) (*auth.Claims, error) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return nil, operations.Unauthorized("missing auth context", nil)
	}
	return claims, nil
}
