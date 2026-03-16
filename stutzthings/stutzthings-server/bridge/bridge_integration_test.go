//go:build integration

package bridge

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestBridgeIntegrationIngestsAndRecovers(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker is not available for integration tests in this environment")
	}

	ctx := context.Background()
	stack := startIntegrationStack(ctx, t)
	defer stack.terminate(ctx, t)

	logger := logrus.New()
	logger.SetOutput(io.Discard)
	config := BridgeConfig{
		MQTTBrokerURL:    stack.mqttURL,
		MQTTUsername:     "test",
		MQTTPassword:     "test",
		InfluxDBURL:      stack.influxURL,
		InfluxDBToken:    stack.token,
		InfluxDBDatabase: "iot",
		BatchSize:        2,
		FlushIntervalMs:  100,
		MaxBufferSize:    50,
		MaxWriteRetries:  3,
	}

	bridge, err := NewBridge(config, logger)
	require.NoError(t, err)
	defer func() { _ = bridge.Stop() }()
	require.NoError(t, bridge.Start(ctx))

	require.NoError(t, publishMQTT(config, "account1/sensor/device01/temperature/celsius", "23.5"))
	require.NoError(t, publishMQTT(config, "account2/sensor/device02/power/enabled", "true"))
	require.NoError(t, publishMQTT(config, "account1/sensor/device01/humidity/percent", `{"value":65.2,"ts":1741910400000}`))

	eventually(t, 20*time.Second, func() bool {
		rows, err := bridge.influx.QueryRows(ctx, `SELECT * FROM device_attributes WHERE time >= now() - interval '10 minutes'`)
		if err != nil {
			return false
		}
		return len(rows) >= 3
	})

	rows, err := bridge.influx.QueryRows(ctx, `SELECT * FROM device_attributes WHERE time >= now() - interval '10 minutes'`)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(rows), 3)

	require.NoError(t, stack.mqtt.Stop(ctx, nil))
	time.Sleep(2 * time.Second)
	require.NoError(t, stack.mqtt.Start(ctx))
	require.NoError(t, publishMQTT(config, "account1/sensor/device01/temperature/celsius", "25.0"))
	eventually(t, 20*time.Second, func() bool {
		rows, err := bridge.influx.QueryRows(ctx, `SELECT * FROM device_attributes WHERE attribute_name = 'celsius' AND time >= now() - interval '10 minutes'`)
		return err == nil && len(rows) >= 2
	})

	require.NoError(t, stack.influx.Stop(ctx, nil))
	require.NoError(t, publishMQTT(config, "account1/sensor/device01/temperature/celsius", "26.0"))
	require.NoError(t, publishMQTT(config, "account1/sensor/device01/temperature/celsius", "27.0"))
	time.Sleep(500 * time.Millisecond)
	require.NoError(t, stack.influx.Start(ctx))
	require.NoError(t, stack.createDatabase(ctx))

	eventually(t, 30*time.Second, func() bool {
		rows, err := bridge.influx.QueryRows(ctx, `SELECT * FROM device_attributes WHERE attribute_name = 'celsius' AND time >= now() - interval '10 minutes'`)
		return err == nil && len(rows) >= 4
	})
}

type integrationStack struct {
	mqtt      testcontainers.Container
	influx    testcontainers.Container
	mqttURL   string
	influxURL string
	token     string
}

func startIntegrationStack(ctx context.Context, t *testing.T) integrationStack {
	t.Helper()
	token := "apiv3_test_token_local_only"
	tokenFile := filepath.Join(t.TempDir(), "admin-token.json")
	require.NoError(t, os.WriteFile(tokenFile, []byte(fmt.Sprintf(`{"token":"%s","name":"admin"}`, token)), 0o600))

	mqttContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "eclipse-mosquitto:2",
			ExposedPorts: []string{"1883/tcp"},
			Env: map[string]string{
				"MQTT_USERNAME": "test",
				"MQTT_PASSWORD": "test",
			},
			Cmd:        []string{"sh", "-ec", "cat >/mosquitto/config/mosquitto.conf <<'EOF'\nlistener 1883\nallow_anonymous false\npassword_file /mosquitto/config/passwd\npersistence false\nEOF\nmosquitto_passwd -b -c /mosquitto/config/passwd \"$MQTT_USERNAME\" \"$MQTT_PASSWORD\"\nexec mosquitto -c /mosquitto/config/mosquitto.conf"},
			WaitingFor: wait.ForListeningPort(nat.Port("1883/tcp")).WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	requireContainerOrSkip(t, err, "eclipse-mosquitto:2")

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

	host, err := mqttContainer.Host(ctx)
	require.NoError(t, err)
	mqttPort, err := mqttContainer.MappedPort(ctx, nat.Port("1883/tcp"))
	require.NoError(t, err)

	influxHost, err := influxContainer.Host(ctx)
	require.NoError(t, err)
	influxPort, err := influxContainer.MappedPort(ctx, nat.Port("8086/tcp"))
	require.NoError(t, err)

	stack := integrationStack{
		mqtt:      mqttContainer,
		influx:    influxContainer,
		mqttURL:   fmt.Sprintf("mqtt://%s:%s", host, mqttPort.Port()),
		influxURL: fmt.Sprintf("http://%s:%s", influxHost, influxPort.Port()),
		token:     token,
	}
	require.NoError(t, stack.createDatabase(ctx))

	return stack
}

func (s integrationStack) createDatabase(ctx context.Context) error {
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

func (s integrationStack) terminate(ctx context.Context, t *testing.T) {
	t.Helper()
	if s.mqtt != nil {
		_ = s.mqtt.Terminate(ctx)
	}
	if s.influx != nil {
		_ = s.influx.Terminate(ctx)
	}
}

func publishMQTT(config BridgeConfig, topic string, payload string) error {
	brokerURL, err := normalizeMQTTBrokerURL(config.MQTTBrokerURL, false)
	if err != nil {
		return err
	}
	opts := mqtt.NewClientOptions().AddBroker(brokerURL)
	opts.SetClientID(defaultClientID() + "-publisher")
	opts.SetUsername(config.MQTTUsername)
	opts.SetPassword(config.MQTTPassword)
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
