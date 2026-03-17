package mcpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/auth"
	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/operations"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Server struct {
	handler http.Handler
	tools   *toolHandlers
}

type toolHandlers struct {
	service *operations.Service
}

func New(service *operations.Service, baseURL string) *Server {
	core := server.NewMCPServer(
		"stutzthings-server",
		"0.1.0",
		server.WithToolCapabilities(true),
	)
	handlers := &toolHandlers{service: service}
	handlers.registerStateTools(core)
	handlers.registerHistoryTools(core)
	handlers.registerCommandTools(core)
	handlers.registerRegistrationTools(core)

	sse := server.NewSSEServer(
		core,
		server.WithBaseURL(baseURL),
		server.WithStaticBasePath("/mcp"),
		server.WithSSEEndpoint(""),
		server.WithMessageEndpoint("/message"),
		server.WithSSEContextFunc(func(ctx context.Context, r *http.Request) context.Context {
			return r.Context()
		}),
	)

	return &Server{handler: sse, tools: handlers}
}

func (s *Server) Handler() http.Handler {
	return s.handler
}

func claimsFromContext(ctx context.Context) (*auth.Claims, error) {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return nil, operations.Unauthorized("missing auth context", nil)
	}
	return claims, nil
}

func jsonToolResult(value any) (*mcp.CallToolResult, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(string(payload)), nil
}

func jsonToolError(err error) (*mcp.CallToolResult, error) {
	code := string(operations.CodeBackendFailure)
	message := err.Error()
	if opErr := operations.AsError(err); opErr != nil {
		code = string(opErr.Code)
		message = opErr.Message
	}
	payload, marshalErr := json.Marshal(map[string]string{"error": code, "message": message})
	if marshalErr != nil {
		return mcp.NewToolResultError(message), nil
	}
	return mcp.NewToolResultError(string(payload)), nil
}

func stringArg(req mcp.CallToolRequest, name string) (string, error) {
	value, ok := req.Params.Arguments[name]
	if !ok {
		return "", operations.InvalidInput(fmt.Sprintf("%s is required", name), nil)
	}
	stringValue, ok := value.(string)
	if !ok {
		return "", operations.InvalidInput(fmt.Sprintf("%s must be a string", name), nil)
	}
	return stringValue, nil
}
