package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/bridge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthHandlerRejectsNonGET(t *testing.T) {
	handler := healthHandler(func(_ context.Context) bridge.BridgeHealthStatus {
		return bridge.BridgeHealthStatus{Health: bridge.HealthOK, Message: "ok"}
	})

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	assert.Equal(t, http.StatusMethodNotAllowed, res.Code)
	assert.Equal(t, http.MethodGet, res.Header().Get("Allow"))
}

func TestHealthHandlerReturnsComputedLatency(t *testing.T) {
	handler := healthHandler(func(_ context.Context) bridge.BridgeHealthStatus {
		return bridge.BridgeHealthStatus{
			Health:          bridge.HealthWarning,
			LatencyMs:       12,
			Message:         "MQTT is reconnecting",
			MQTTConnected:   false,
			InfluxReachable: true,
			BufferUsage:     3,
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	assert.Equal(t, 210, res.Code)
	assert.Equal(t, "application/json", res.Header().Get("Content-Type"))

	var status bridge.BridgeHealthStatus
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &status))
	assert.Equal(t, bridge.HealthWarning, status.Health)
	assert.Equal(t, "MQTT is reconnecting", status.Message)
	assert.False(t, status.MQTTConnected)
	assert.True(t, status.InfluxReachable)
	assert.Equal(t, 3, status.BufferUsage)
	assert.Equal(t, int64(12), status.LatencyMs)
}
