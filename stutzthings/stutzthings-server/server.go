package main

import (
	"context"
	"net/http"
	"strings"

	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/api"
	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/auth"
	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/bridge"
	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/mcpapi"
	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/operations"
)

func newHTTPHandler(service *operations.Service, authenticator auth.BearerAuthenticator, getStatus func(context.Context) bridge.BridgeHealthStatus, baseURL string) http.Handler {
	apiHandlers := api.NewHandlers(service)
	mcpServer := mcpapi.New(service, baseURL)
	mcpHandler := authenticator.Require(mcpServer.Handler())

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler(getStatus))
	mux.Handle("POST /registration", authenticator.Require(http.HandlerFunc(apiHandlers.RegisterDeviceInstance)))
	mux.Handle("GET /{account_id}/{device_id}/{device_instance_id}", authenticator.Require(http.HandlerFunc(apiHandlers.GetDeviceState)))
	mux.Handle("GET /{account_id}/{device_id}/{device_instance_id}/{node_name}/{attribute_name}", authenticator.Require(http.HandlerFunc(apiHandlers.GetAttributeState)))
	mux.Handle("PUT /{account_id}/{device_id}/{device_instance_id}/{node_name}/{attribute_name}", authenticator.Require(http.HandlerFunc(apiHandlers.SetDesiredState)))
	mux.Handle("GET /{account_id}/{device_id}/{device_instance_id}/{node_name}/history", authenticator.Require(http.HandlerFunc(apiHandlers.GetNodeHistory)))
	mux.Handle("GET /{account_id}/{device_id}/{device_instance_id}/{node_name}/{attribute_name}/history", authenticator.Require(http.HandlerFunc(apiHandlers.GetAttributeHistory)))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/mcp" || strings.HasPrefix(r.URL.Path, "/mcp/") {
			mcpHandler.ServeHTTP(w, r)
			return
		}
		mux.ServeHTTP(w, r)
	})
}
