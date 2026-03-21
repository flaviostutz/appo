package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/auth"
	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/bridge"
	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/operations"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type mainTestStore struct{}

func (mainTestStore) QueryRows(_ context.Context, query string) ([]map[string]any, error) {
	if strings.Contains(query, "SELECT device_instance_id") {
		return nil, nil
	}
	return []map[string]any{mainTestObservationRow("node", "temperature", 21.5, "2026-03-17T00:00:00Z")}, nil
}

type mainTestPublisher struct{}

func (mainTestPublisher) Publish(context.Context, string, byte, bool, []byte) error {
	return nil
}

type mainRESTError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type mainMCPResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError,omitempty"`
	} `json:"result"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type jwtMosquittoStack struct {
	container testcontainers.Container
	mqttURL   string
	secret    []byte
	secretB64 string
	signer    *auth.Signer
}

type operationsIntegrationStack struct {
	mqtt      testcontainers.Container
	influx    testcontainers.Container
	mqttURL   string
	influxURL string
	token     string
	secret    []byte
	secretB64 string
	signer    *auth.Signer
}

func newMainTestServer(t *testing.T) (*httptest.Server, *auth.Signer, string) {
	t.Helper()
	secret, err := auth.DecodeBase64Secret("ZGV2X3NlY3JldA==")
	require.NoError(t, err)
	signer, err := auth.NewSigner(secret, "issuer", time.Hour)
	require.NoError(t, err)
	service := operations.NewService(mainTestStore{}, mainTestPublisher{}, signer)
	handler := newHTTPHandler(service, auth.NewBearerAuthenticator(secret), func(_ context.Context) bridge.BridgeHealthStatus {
		return bridge.BridgeHealthStatus{Health: bridge.HealthOK, Message: "ok"}
	}, "")
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	adminToken := mustSignToken(t, signer, "demo-admin", []string{"acct/device/+/+/+"}, []string{"acct/device/+/+/+/set"})
	return server, signer, adminToken
}

func mustSignToken(t *testing.T, signer *auth.Signer, subject string, publ []string, subs []string) string {
	t.Helper()
	token, _, err := signer.Sign(subject, publ, subs, time.Now().UTC())
	require.NoError(t, err)
	return token
}

func doJSONRequest(t *testing.T, method string, requestURL string, token string, body string) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, requestURL, reader)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = res.Body.Close() })
	return res
}

func decodeRESTError(t *testing.T, res *http.Response) mainRESTError {
	t.Helper()
	defer func() { _ = res.Body.Close() }()
	var response mainRESTError
	require.NoError(t, json.NewDecoder(res.Body).Decode(&response))
	return response
}

func callMCPToolOverHTTP(t *testing.T, serverURL string, token string, toolName string, arguments map[string]any) mainMCPResponse {
	t.Helper()
	client := &http.Client{Timeout: 10 * time.Second}

	sseReq, err := http.NewRequest(http.MethodGet, serverURL+"/mcp", nil)
	require.NoError(t, err)
	sseReq.Header.Set("Authorization", "Bearer "+token)
	sseResp, err := client.Do(sseReq)
	require.NoError(t, err)
	defer func() { _ = sseResp.Body.Close() }()
	require.Equal(t, http.StatusOK, sseResp.StatusCode)

	reader := bufio.NewReader(sseResp.Body)
	endpointEvent := readSSEEvent(t, reader)
	messageURL := resolveMessageURL(t, serverURL, endpointEvent)

	postMCPMessage(t, client, messageURL, token, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"clientInfo": map[string]any{
				"name":    "main-test",
				"version": "1.0.0",
			},
		},
	})
	_ = readMCPResponse(t, reader)

	postMCPMessage(t, client, messageURL, token, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      toolName,
			"arguments": arguments,
		},
	})

	return readMCPResponse(t, reader)
}

func postMCPMessage(t *testing.T, client *http.Client, messageURL string, token string, payload map[string]any) {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, messageURL, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = res.Body.Close() }()
	require.Equal(t, http.StatusAccepted, res.StatusCode)
}

func readMCPResponse(t *testing.T, reader *bufio.Reader) mainMCPResponse {
	t.Helper()
	event := readSSEEvent(t, reader)
	data := sseEventData(t, event)
	var response mainMCPResponse
	require.NoError(t, json.Unmarshal([]byte(data), &response))
	return response
}

func readSSEEvent(t *testing.T, reader *bufio.Reader) string {
	t.Helper()
	var builder strings.Builder
	for {
		line, err := reader.ReadString('\n')
		require.NoError(t, err)
		builder.WriteString(line)
		if strings.TrimSpace(line) == "" {
			return builder.String()
		}
	}
}

func sseEventData(t *testing.T, event string) string {
	t.Helper()
	for _, line := range strings.Split(event, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "data:") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
		}
	}
	t.Fatalf("event does not contain data line: %s", event)
	return ""
}

func resolveMessageURL(t *testing.T, serverURL string, endpointEvent string) string {
	t.Helper()
	messageURL := sseEventData(t, endpointEvent)
	parsed, err := url.Parse(messageURL)
	require.NoError(t, err)
	if parsed.IsAbs() {
		return parsed.String()
	}
	base, err := url.Parse(serverURL)
	require.NoError(t, err)
	return base.ResolveReference(parsed).String()
}

func mainTestObservationRow(node string, attribute string, value any, timestamp string) map[string]any {
	row := map[string]any{
		"account_id":         "acct",
		"device_id":          "device",
		"device_instance_id": "instance",
		"node_name":          node,
		"attribute_name":     attribute,
		"time":               timestamp,
	}
	row["value_float"] = value
	return row
}

func mqttPluginPathForDocker(t *testing.T) string {
	t.Helper()
	path := strings.TrimSpace(os.Getenv("MOSQUITTO_JWT_AUTH_SO"))
	if path == "" {
		t.Skip("MOSQUITTO_JWT_AUTH_SO is not set")
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		t.Skipf("MOSQUITTO_JWT_AUTH_SO is not a readable file: %v", err)
	}
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	if len(content) < 4 || !bytes.Equal(content[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		t.Skip("MOSQUITTO_JWT_AUTH_SO is not a Linux ELF shared library usable in the Mosquitto container")
	}
	return path
}

func startJWTMosquittoStack(ctx context.Context, t *testing.T) jwtMosquittoStack {
	t.Helper()
	if !dockerAvailable() {
		t.Skip("docker is not available for MQTT interoperability tests in this environment")
	}
	pluginPath := mqttPluginPathForDocker(t)
	secretB64 := bridge.DefaultJWTSecretBase64()
	secret, err := auth.DecodeBase64Secret(secretB64)
	require.NoError(t, err)
	signer, err := auth.NewSigner(secret, "issuer", time.Hour)
	require.NoError(t, err)
	configFile := filepath.Join(t.TempDir(), "mosquitto.conf")
	config := strings.Join([]string{
		"listener 1883",
		"allow_anonymous false",
		"auth_plugin /mosquitto/auth/libmosquitto_jwt_auth.so",
		"auth_opt_jwt_alg HS256",
		"auth_opt_jwt_sec_env JWT_SIGNING_SECRET_BASE64",
		"auth_opt_jwt_validate_exp true",
		"auth_opt_jwt_validate_sub_match_username true",
		"persistence false",
		"log_dest stdout",
		"log_type error",
		"log_type warning",
		"log_type notice",
		"log_type information",
	}, "\n") + "\n"
	require.NoError(t, os.WriteFile(configFile, []byte(config), 0o600))

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "eclipse-mosquitto:2",
			ExposedPorts: []string{"1883/tcp"},
			Env: map[string]string{
				"JWT_SIGNING_SECRET_BASE64": secretB64,
			},
			Files: []testcontainers.ContainerFile{{
				HostFilePath:      configFile,
				ContainerFilePath: "/mosquitto/config/mosquitto.conf",
				FileMode:          0o644,
			}, {
				HostFilePath:      pluginPath,
				ContainerFilePath: "/mosquitto/auth/libmosquitto_jwt_auth.so",
				FileMode:          0o755,
			}},
			Cmd:        []string{"mosquitto", "-c", "/mosquitto/config/mosquitto.conf"},
			WaitingFor: wait.ForListeningPort(nat.Port("1883/tcp")).WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	requireContainerOrSkip(t, err, "eclipse-mosquitto:2")
	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, nat.Port("1883/tcp"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })
	return jwtMosquittoStack{
		container: container,
		mqttURL:   fmt.Sprintf("mqtt://%s:%s", host, port.Port()),
		secret:    secret,
		secretB64: secretB64,
		signer:    signer,
	}
}

func startOperationsIntegrationStack(ctx context.Context, t *testing.T) operationsIntegrationStack {
	t.Helper()
	mqttStack := startJWTMosquittoStack(ctx, t)
	tokenFile := filepath.Join(t.TempDir(), "admin-token.json")
	influxToken := "apiv3_test_token_local_only"
	require.NoError(t, os.WriteFile(tokenFile, []byte(fmt.Sprintf(`{"token":"%s","name":"admin"}`, influxToken)), 0o600))

	influxContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "influxdb:3-core",
			ExposedPorts: []string{"8086/tcp"},
			Files: []testcontainers.ContainerFile{{
				HostFilePath:      tokenFile,
				ContainerFilePath: "/run/secrets/admin-token",
				FileMode:          0o600,
			}},
			Cmd:        []string{"influxdb3", "serve", "--node-id=node0", "--object-store=file", "--data-dir=/var/lib/influxdb3/data", "--http-bind=0.0.0.0:8086", "--admin-token-file=/run/secrets/admin-token"},
			WaitingFor: wait.ForListeningPort(nat.Port("8086/tcp")).WithStartupTimeout(90 * time.Second),
		},
		Started: true,
	})
	requireContainerOrSkip(t, err, "influxdb:3-core")
	host, err := influxContainer.Host(ctx)
	require.NoError(t, err)
	port, err := influxContainer.MappedPort(ctx, nat.Port("8086/tcp"))
	require.NoError(t, err)

	stack := operationsIntegrationStack{
		mqtt:      mqttStack.container,
		influx:    influxContainer,
		mqttURL:   mqttStack.mqttURL,
		influxURL: fmt.Sprintf("http://%s:%s", host, port.Port()),
		token:     influxToken,
		secret:    mqttStack.secret,
		secretB64: mqttStack.secretB64,
		signer:    mqttStack.signer,
	}
	require.NoError(t, stack.createDatabase(ctx))
	t.Cleanup(func() { _ = influxContainer.Terminate(ctx) })
	return stack
}

func (s operationsIntegrationStack) createDatabase(ctx context.Context) error {
	exitCode, output, err := s.influx.Exec(ctx, []string{"sh", "-lc", fmt.Sprintf("export INFLUXDB3_AUTH_TOKEN=%s && influxdb3 create database iot --host http://127.0.0.1:8086", s.token)})
	if err != nil {
		return err
	}
	if exitCode != 0 {
		body, _ := io.ReadAll(output)
		if string(body) != "" && !stringsContainsIgnoreCase(string(body), "already exists") {
			return fmt.Errorf("create database failed: %s", string(body))
		}
	}
	return nil
}

func publishMQTTWithToken(brokerURL string, username string, password string, topic string, payload string) error {
	clientID := fmt.Sprintf("main-test-%d", time.Now().UnixNano())
	opts := mqtt.NewClientOptions().AddBroker(brokerURL)
	opts.SetClientID(clientID)
	opts.SetUsername(username)
	opts.SetPassword(password)
	opts.SetProtocolVersion(4)
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	defer client.Disconnect(100)
	token := client.Publish(topic, 1, false, payload)
	token.Wait()
	return token.Error()
}

func subscribeMQTTWithToken(t *testing.T, brokerURL string, username string, password string, topic string) (<-chan mqtt.Message, func()) {
	t.Helper()
	messages := make(chan mqtt.Message, 4)
	clientID := fmt.Sprintf("main-sub-%d", time.Now().UnixNano())
	opts := mqtt.NewClientOptions().AddBroker(brokerURL)
	opts.SetClientID(clientID)
	opts.SetUsername(username)
	opts.SetPassword(password)
	opts.SetProtocolVersion(4)
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		require.NoError(t, token.Error())
	}
	if token := client.Subscribe(topic, 1, func(_ mqtt.Client, message mqtt.Message) {
		messages <- message
	}); token.Wait() && token.Error() != nil {
		client.Disconnect(100)
		require.NoError(t, token.Error())
	}
	return messages, func() { client.Disconnect(100) }
}

func eventually(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatal("condition was not met before timeout")
}

func stringsContainsIgnoreCase(value string, needle string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(needle))
}

func dockerAvailable() bool {
	cmd := exec.Command("docker", "info")
	cmd.Env = os.Environ()
	return cmd.Run() == nil
}

func requireContainerOrSkip(t *testing.T, err error, image string) {
	t.Helper()
	if err == nil {
		return
	}
	if dockerRegistryBlocked(err) {
		t.Skipf("docker image %q cannot be pulled in this environment: %v", image, err)
	}
	require.NoError(t, err)
}

func dockerRegistryBlocked(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "zscaler") ||
		strings.Contains(message, "docker and mirrors") ||
		strings.Contains(message, "registry-1.docker.io")
}

func newDiscardLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return logger
}
