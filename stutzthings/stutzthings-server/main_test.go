package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/auth"
	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/bridge"
	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/operations"
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

func TestMainRoutesPreserveRESTMCPValidationParity(t *testing.T) {
	server, _, adminToken := newMainTestServer(t)

	t.Run("attribute identity validation", func(t *testing.T) {
		rest := doJSONRequest(t, http.MethodGet, server.URL+"/acct/device/instance/node/tem+perature", adminToken, "")
		assert.Equal(t, http.StatusBadRequest, rest.StatusCode)
		restError := decodeRESTError(t, rest)

		mcp := callMCPToolOverHTTP(t, server.URL, adminToken, "get_attribute_state", map[string]any{
			"account_id":         "acct",
			"device_id":          "device",
			"device_instance_id": "instance",
			"node_name":          "node",
			"attribute_name":     "tem+perature",
		})
		require.Nil(t, mcp.Error)
		require.True(t, mcp.Result.IsError)
		require.Len(t, mcp.Result.Content, 1)
		var mcpError mainRESTError
		require.NoError(t, json.Unmarshal([]byte(mcp.Result.Content[0].Text), &mcpError))
		assert.Equal(t, restError, mcpError)
	})

	t.Run("history range validation", func(t *testing.T) {
		rest := doJSONRequest(t, http.MethodGet, server.URL+"/acct/device/instance/node/temperature/history?from=since=2d&to=2026-03-17T01:00:00Z", adminToken, "")
		assert.Equal(t, http.StatusBadRequest, rest.StatusCode)
		restError := decodeRESTError(t, rest)

		mcp := callMCPToolOverHTTP(t, server.URL, adminToken, "get_attribute_history", map[string]any{
			"account_id":         "acct",
			"device_id":          "device",
			"device_instance_id": "instance",
			"node_name":          "node",
			"attribute_name":     "temperature",
			"from":               "since=2d",
			"to":                 "2026-03-17T01:00:00Z",
		})
		require.Nil(t, mcp.Error)
		require.True(t, mcp.Result.IsError)
		var mcpError mainRESTError
		require.NoError(t, json.Unmarshal([]byte(mcp.Result.Content[0].Text), &mcpError))
		assert.Equal(t, restError, mcpError)
	})

	t.Run("registration validation", func(t *testing.T) {
		rest := doJSONRequest(t, http.MethodPost, server.URL+"/registration", adminToken, `{"accountId":"acct/tenant","deviceId":"device"}`)
		assert.Equal(t, http.StatusBadRequest, rest.StatusCode)
		restError := decodeRESTError(t, rest)

		mcp := callMCPToolOverHTTP(t, server.URL, adminToken, "register_device_instance", map[string]any{
			"account_id": "acct/tenant",
			"device_id":  "device",
		})
		require.Nil(t, mcp.Error)
		require.True(t, mcp.Result.IsError)
		var mcpError mainRESTError
		require.NoError(t, json.Unmarshal([]byte(mcp.Result.Content[0].Text), &mcpError))
		assert.Equal(t, restError, mcpError)
	})
}

func TestMQTTBrokerRejectsUsernameThatDoesNotMatchSub(t *testing.T) {
	ctx := context.Background()
	stack := startJWTMosquittoStack(ctx, t)

	deviceToken := mustSignToken(t, stack.signer, "acct/device/instance", []string{"acct/device/instance/+/+"}, []string{"acct/device/instance/+/+/set"})
	require.NoError(t, publishMQTTWithToken(stack.mqttURL, "acct/device/instance", deviceToken, "acct/device/instance/temperature/celsius", `{"value":22.5}`))
	assert.Error(t, publishMQTTWithToken(stack.mqttURL, "wrong-user", deviceToken, "acct/device/instance/temperature/celsius", `{"value":22.5}`))
}

func TestRegistrationTokenWorksAcrossMQTTRESTAndMCP(t *testing.T) {
	ctx := context.Background()
	stack := startOperationsIntegrationStack(ctx, t)

	bridgeToken := mustSignToken(t, stack.signer, "bridge", []string{"#"}, []string{"#"})
	runtimeBridge, err := bridge.NewBridge(bridge.BridgeConfig{
		MQTTBrokerURL:    stack.mqttURL,
		MQTTUsername:     "bridge",
		MQTTPassword:     bridgeToken,
		InfluxDBURL:      stack.influxURL,
		InfluxDBToken:    stack.token,
		InfluxDBDatabase: "iot",
		BatchSize:        1,
		FlushIntervalMs:  50,
		MaxBufferSize:    50,
		MaxWriteRetries:  3,
	}, newDiscardLogger())
	require.NoError(t, err)
	require.NoError(t, runtimeBridge.Start(ctx))
	defer func() { _ = runtimeBridge.Stop() }()

	signer, err := auth.NewSigner(stack.secret, "issuer", time.Hour)
	require.NoError(t, err)
	service := operations.NewService(runtimeBridge, runtimeBridge, signer)
	server := httptest.NewServer(newHTTPHandler(service, auth.NewBearerAuthenticator(stack.secret), runtimeBridge.CheckHealth, ""))
	defer server.Close()

	adminToken := mustSignToken(t, signer, "demo-admin", []string{"demo-account/demo-device/+/+/+"}, []string{"demo-account/demo-device/+/+/+/set"})
	registrationRes := doJSONRequest(t, http.MethodPost, server.URL+"/registration", adminToken, `{"accountId":"demo-account","deviceId":"demo-device"}`)
	require.Equal(t, http.StatusOK, registrationRes.StatusCode)
	defer func() { _ = registrationRes.Body.Close() }()
	var registration operations.RegistrationResponse
	require.NoError(t, json.NewDecoder(registrationRes.Body).Decode(&registration))

	commandMessages, stopSub := subscribeMQTTWithToken(t, stack.mqttURL, registration.Sub, registration.Token, registration.AccountID+"/"+registration.DeviceID+"/"+registration.DeviceInstanceID+"/+/+/set")
	defer stopSub()

	telemetryTopic := registration.AccountID + "/" + registration.DeviceID + "/" + registration.DeviceInstanceID + "/temperature/celsius"
	require.NoError(t, publishMQTTWithToken(stack.mqttURL, registration.Sub, registration.Token, telemetryTopic, `{"value":22.5}`))

	eventually(t, 20*time.Second, func() bool {
		res := doJSONRequest(t, http.MethodGet, server.URL+"/"+registration.AccountID+"/"+registration.DeviceID+"/"+registration.DeviceInstanceID+"/temperature/celsius", registration.Token, "")
		defer func() { _ = res.Body.Close() }()
		if res.StatusCode != http.StatusOK {
			return false
		}
		var response operations.AttributeStateResponse
		if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
			return false
		}
		return response.Found && response.Observation != nil && response.Observation.Value == 22.5
	})

	mcp := callMCPToolOverHTTP(t, server.URL, registration.Token, "get_attribute_state", map[string]any{
		"account_id":         registration.AccountID,
		"device_id":          registration.DeviceID,
		"device_instance_id": registration.DeviceInstanceID,
		"node_name":          "temperature",
		"attribute_name":     "celsius",
	})
	require.Nil(t, mcp.Error)
	require.False(t, mcp.Result.IsError)
	var mcpState operations.AttributeStateResponse
	require.NoError(t, json.Unmarshal([]byte(mcp.Result.Content[0].Text), &mcpState))
	require.True(t, mcpState.Found)
	require.NotNil(t, mcpState.Observation)
	assert.Equal(t, 22.5, mcpState.Observation.Value)

	commandRes := doJSONRequest(t, http.MethodPut, server.URL+"/"+registration.AccountID+"/"+registration.DeviceID+"/"+registration.DeviceInstanceID+"/lamp/power", registration.Token, `{"value":"on"}`)
	require.Equal(t, http.StatusOK, commandRes.StatusCode)
	defer func() { _ = commandRes.Body.Close() }()
	select {
	case message := <-commandMessages:
		assert.Equal(t, registration.AccountID+"/"+registration.DeviceID+"/"+registration.DeviceInstanceID+"/lamp/power/set", message.Topic())
		assert.JSONEq(t, `{"value":"on"}`, string(message.Payload()))
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for desired-state MQTT message")
	}
}
