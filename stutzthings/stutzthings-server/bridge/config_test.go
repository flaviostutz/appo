package bridge

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfigUsesDefaultBridgeJSON(t *testing.T) {
	clearBridgeEnv(t)
	configDir := t.TempDir()
	writeConfigFile(t, configDir, `{
		"mqttBrokerUrl": "mqtt://localhost:1883",
		"mqttUsername": "file-user",
		"mqttPassword": "file-pass",
		"influxDbUrl": "http://localhost:8086",
		"influxDbToken": "file-token",
		"influxDbDatabase": "iot"
	}`)

	restoreCWD := chdirForTest(t, configDir)
	defer restoreCWD()

	config, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, "mqtt://localhost:1883", config.MQTTBrokerURL)
	assert.Equal(t, "file-user", config.MQTTUsername)
	assert.Equal(t, "file-pass", config.MQTTPassword)
	assert.Equal(t, "http://localhost:8086", config.InfluxDBURL)
	assert.Equal(t, "file-token", config.InfluxDBToken)
	assert.Equal(t, "iot", config.InfluxDBDatabase)
	assert.False(t, config.MQTTTLSEnabled)
	assert.Equal(t, defaultBatchSize, config.BatchSize)
	assert.Equal(t, defaultFlushIntervalMs, config.FlushIntervalMs)
	assert.Equal(t, defaultMaxBufferSize, config.MaxBufferSize)
	assert.Equal(t, defaultMaxWriteRetries, config.MaxWriteRetries)
}

func TestLoadConfigFallsBackToEnvWhenDefaultFileMissing(t *testing.T) {
	clearBridgeEnv(t)
	t.Setenv("MQTT_BROKER_URL", "mqtt://localhost:1883")
	t.Setenv("MQTT_USERNAME", "env-user")
	t.Setenv("MQTT_PASSWORD", "env-pass")
	t.Setenv("MQTT_TLS_ENABLED", "true")
	t.Setenv("INFLUXDB_URL", "http://localhost:8086")
	t.Setenv("INFLUXDB_TOKEN", "env-token")
	t.Setenv("INFLUXDB_DATABASE", "iot")
	t.Setenv("BRIDGE_BATCH_SIZE", "50")
	t.Setenv("BRIDGE_FLUSH_INTERVAL_MS", "250")
	t.Setenv("BRIDGE_MAX_BUFFER_SIZE", "500")
	t.Setenv("BRIDGE_MAX_WRITE_RETRIES", "2")

	configDir := t.TempDir()
	restoreCWD := chdirForTest(t, configDir)
	defer restoreCWD()

	config, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, "env-user", config.MQTTUsername)
	assert.Equal(t, "env-pass", config.MQTTPassword)
	assert.True(t, config.MQTTTLSEnabled)
	assert.Equal(t, 50, config.BatchSize)
	assert.Equal(t, 250, config.FlushIntervalMs)
	assert.Equal(t, 500, config.MaxBufferSize)
	assert.Equal(t, 2, config.MaxWriteRetries)
}

func TestLoadConfigFromFileSupportsExplicitPath(t *testing.T) {
	clearBridgeEnv(t)
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "custom-bridge.json")
	require.NoError(t, os.WriteFile(configPath, []byte(`{
		"mqttBrokerUrl": "mqtt://example:1883",
		"mqttUsername": "custom-user",
		"mqttPassword": "custom-pass",
		"mqttTlsEnabled": true,
		"influxDbUrl": "http://example:8086",
		"influxDbToken": "custom-token",
		"influxDbDatabase": "custom-db",
		"batchSize": 7,
		"flushIntervalMs": 150,
		"maxBufferSize": 14,
		"maxWriteRetries": 0
	}`), 0o600))
	t.Setenv("BRIDGE_CONFIG_PATH", configPath)

	config, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, "mqtt://example:1883", config.MQTTBrokerURL)
	assert.Equal(t, "custom-user", config.MQTTUsername)
	assert.Equal(t, "custom-pass", config.MQTTPassword)
	assert.True(t, config.MQTTTLSEnabled)
	assert.Equal(t, "http://example:8086", config.InfluxDBURL)
	assert.Equal(t, "custom-token", config.InfluxDBToken)
	assert.Equal(t, "custom-db", config.InfluxDBDatabase)
	assert.Equal(t, 7, config.BatchSize)
	assert.Equal(t, 150, config.FlushIntervalMs)
	assert.Equal(t, 14, config.MaxBufferSize)
	assert.Equal(t, 0, config.MaxWriteRetries)
}

func clearBridgeEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"BRIDGE_CONFIG_PATH",
		"MQTT_BROKER_URL",
		"MQTT_USERNAME",
		"MQTT_PASSWORD",
		"MQTT_TLS_ENABLED",
		"INFLUXDB_URL",
		"INFLUXDB_TOKEN",
		"INFLUXDB_DATABASE",
		"BRIDGE_BATCH_SIZE",
		"BRIDGE_FLUSH_INTERVAL_MS",
		"BRIDGE_MAX_BUFFER_SIZE",
		"BRIDGE_MAX_WRITE_RETRIES",
	} {
		t.Setenv(name, "")
	}
}

func writeConfigFile(t *testing.T, dir string, contents string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, defaultConfigPath), []byte(contents), 0o600))
}

func chdirForTest(t *testing.T, dir string) func() {
	t.Helper()
	currentDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	return func() {
		require.NoError(t, os.Chdir(currentDir))
	}
}
