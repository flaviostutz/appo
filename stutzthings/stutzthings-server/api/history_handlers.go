package api

import (
	"net/http"

	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/operations"
)

func (h Handlers) GetNodeHistory(w http.ResponseWriter, r *http.Request) {
	claims, err := claimsFromRequest(r)
	if err != nil {
		writeOperationError(w, err)
		return
	}
	historyRange, err := operations.ParseHistoryRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		writeOperationError(w, err)
		return
	}
	response, err := h.service.GetNodeHistory(r.Context(), claims, operations.NodeHistoryRequest{
		DeviceScope: operations.DeviceScope{
			AccountID:        r.PathValue("account_id"),
			DeviceID:         r.PathValue("device_id"),
			DeviceInstanceID: r.PathValue("device_instance_id"),
		},
		NodeName: r.PathValue("node_name"),
		From:     historyRange.From,
		To:       historyRange.To,
	})
	if err != nil {
		writeOperationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h Handlers) GetAttributeHistory(w http.ResponseWriter, r *http.Request) {
	claims, err := claimsFromRequest(r)
	if err != nil {
		writeOperationError(w, err)
		return
	}
	historyRange, err := operations.ParseHistoryRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		writeOperationError(w, err)
		return
	}
	response, err := h.service.GetAttributeHistory(r.Context(), claims, operations.AttributeHistoryRequest{
		DeviceIdentity: operations.DeviceIdentity{
			DeviceScope: operations.DeviceScope{
				AccountID:        r.PathValue("account_id"),
				DeviceID:         r.PathValue("device_id"),
				DeviceInstanceID: r.PathValue("device_instance_id"),
			},
			NodeName:      r.PathValue("node_name"),
			AttributeName: r.PathValue("attribute_name"),
		},
		From: historyRange.From,
		To:   historyRange.To,
	})
	if err != nil {
		writeOperationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}
