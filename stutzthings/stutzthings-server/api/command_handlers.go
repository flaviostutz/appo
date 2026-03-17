package api

import (
	"net/http"

	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/operations"
)

func (h Handlers) SetDesiredState(w http.ResponseWriter, r *http.Request) {
	claims, err := claimsFromRequest(r)
	if err != nil {
		writeOperationError(w, err)
		return
	}
	var request operations.DesiredStateRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeOperationError(w, err)
		return
	}
	response, err := h.service.SetDesiredState(r.Context(), claims, operations.DeviceIdentity{
		DeviceScope: operations.DeviceScope{
			AccountID:        r.PathValue("account_id"),
			DeviceID:         r.PathValue("device_id"),
			DeviceInstanceID: r.PathValue("device_instance_id"),
		},
		NodeName:      r.PathValue("node_name"),
		AttributeName: r.PathValue("attribute_name"),
	}, request)
	if err != nil {
		writeOperationError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, response)
}
