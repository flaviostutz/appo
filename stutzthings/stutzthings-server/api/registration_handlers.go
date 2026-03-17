package api

import (
	"net/http"

	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/operations"
)

func (h Handlers) RegisterDeviceInstance(w http.ResponseWriter, r *http.Request) {
	claims, err := claimsFromRequest(r)
	if err != nil {
		writeOperationError(w, err)
		return
	}
	var request operations.RegistrationRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeOperationError(w, err)
		return
	}
	response, err := h.service.RegisterDeviceInstance(r.Context(), claims, request)
	if err != nil {
		writeOperationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}
