package api

import (
	"net/http"

	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/operations"
)

func (h Handlers) GetDeviceState(w http.ResponseWriter, r *http.Request) {
	claims, err := claimsFromRequest(r)
	if err != nil {
		writeOperationError(w, err)
		return
	}
	response, err := h.service.GetDeviceState(r.Context(), claims, operations.DeviceScope{
		AccountID:        r.PathValue("account_id"),
		DeviceID:         r.PathValue("device_id"),
		DeviceInstanceID: r.PathValue("device_instance_id"),
	})
	if err != nil {
		writeOperationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h Handlers) GetAttributeState(w http.ResponseWriter, r *http.Request) {
	claims, err := claimsFromRequest(r)
	if err != nil {
		writeOperationError(w, err)
		return
	}
	response, err := h.service.GetAttributeState(r.Context(), claims, operations.DeviceIdentity{
		DeviceScope: operations.DeviceScope{
			AccountID:        r.PathValue("account_id"),
			DeviceID:         r.PathValue("device_id"),
			DeviceInstanceID: r.PathValue("device_instance_id"),
		},
		NodeName:      r.PathValue("node_name"),
		AttributeName: r.PathValue("attribute_name"),
	})
	if err != nil {
		writeOperationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}
