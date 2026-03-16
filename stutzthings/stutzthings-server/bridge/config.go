package bridge

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultConfigPath      = "bridge.json"
	defaultBatchSize       = 100
	defaultFlushIntervalMs = 100
	defaultMaxBufferSize   = 10000
	defaultMaxWriteRetries = 3

	measurementName = "device_attributes"

	HealthOK      = "OK"
	HealthWarning = "WARNING"
	HealthError   = "ERROR"
)

type BridgeConfig struct {
	MQTTBrokerURL    string
	MQTTUsername     string
	MQTTPassword     string
	MQTTTLSEnabled   bool
	InfluxDBURL      string
	InfluxDBToken    string
	InfluxDBDatabase string
	BatchSize        int
	FlushIntervalMs  int
	MaxBufferSize    int
	MaxWriteRetries  int
}

type MQTTMessage struct {
	Topic      string
	Payload    []byte
	ReceivedAt time.Time
}

type DeviceAttribute struct {
	AccountID        string
	DeviceID         string
	DeviceInstanceID string
	NodeName         string
	AttributeName    string
	ValueFloat       *float64
	ValueString      *string
	ValueBool        *bool
	Timestamp        time.Time
}

type BridgeHealthStatus struct {
	Health          string `json:"health"`
	LatencyMs       int64  `json:"latencyMs"`
	Message         string `json:"message"`
	MQTTConnected   bool   `json:"mqttConnected"`
	InfluxReachable bool   `json:"influxReachable"`
	BufferUsage     int    `json:"bufferUsage"`
}

type bridgeConfigFile struct {
	MQTTBrokerURL    string `json:"mqttBrokerUrl"`
	MQTTUsername     string `json:"mqttUsername"`
	MQTTPassword     string `json:"mqttPassword"`
	MQTTTLSEnabled   *bool  `json:"mqttTlsEnabled"`
	InfluxDBURL      string `json:"influxDbUrl"`
	InfluxDBToken    string `json:"influxDbToken"`
	InfluxDBDatabase string `json:"influxDbDatabase"`
	BatchSize        *int   `json:"batchSize"`
	FlushIntervalMs  *int   `json:"flushIntervalMs"`
	MaxBufferSize    *int   `json:"maxBufferSize"`
	MaxWriteRetries  *int   `json:"maxWriteRetries"`
}

func LoadConfig() (BridgeConfig, error) {
	configuredPath := strings.TrimSpace(os.Getenv("BRIDGE_CONFIG_PATH"))
	configPath := configuredPath
	if configPath == "" {
		configPath = defaultConfigPath
	}

	config, err := LoadConfigFromFile(configPath)
	if err == nil {
		return config, nil
	}
	if errors.Is(err, os.ErrNotExist) && configuredPath == "" {
		return LoadConfigFromEnv()
	}

	return BridgeConfig{}, fmt.Errorf("load config file %q: %w", configPath, err)
}

func LoadConfigFromFile(path string) (BridgeConfig, error) {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return BridgeConfig{}, fmt.Errorf("config path must not be empty")
	}

	contents, err := os.ReadFile(trimmedPath)
	if err != nil {
		return BridgeConfig{}, err
	}

	var fileConfig bridgeConfigFile
	if err := json.Unmarshal(contents, &fileConfig); err != nil {
		return BridgeConfig{}, err
	}

	config := BridgeConfig{
		MQTTBrokerURL:    strings.TrimSpace(fileConfig.MQTTBrokerURL),
		MQTTUsername:     strings.TrimSpace(fileConfig.MQTTUsername),
		MQTTPassword:     fileConfig.MQTTPassword,
		MQTTTLSEnabled:   boolOrDefault(fileConfig.MQTTTLSEnabled, false),
		InfluxDBURL:      strings.TrimSpace(fileConfig.InfluxDBURL),
		InfluxDBToken:    strings.TrimSpace(fileConfig.InfluxDBToken),
		InfluxDBDatabase: strings.TrimSpace(fileConfig.InfluxDBDatabase),
		BatchSize:        intOrDefault(fileConfig.BatchSize, defaultBatchSize),
		FlushIntervalMs:  intOrDefault(fileConfig.FlushIntervalMs, defaultFlushIntervalMs),
		MaxBufferSize:    intOrDefault(fileConfig.MaxBufferSize, defaultMaxBufferSize),
		MaxWriteRetries:  intOrDefault(fileConfig.MaxWriteRetries, defaultMaxWriteRetries),
	}

	if err := config.Validate(); err != nil {
		return BridgeConfig{}, err
	}

	return config, nil
}

func LoadConfigFromEnv() (BridgeConfig, error) {
	config := BridgeConfig{
		MQTTBrokerURL:    strings.TrimSpace(os.Getenv("MQTT_BROKER_URL")),
		MQTTUsername:     strings.TrimSpace(os.Getenv("MQTT_USERNAME")),
		MQTTPassword:     os.Getenv("MQTT_PASSWORD"),
		InfluxDBURL:      strings.TrimSpace(os.Getenv("INFLUXDB_URL")),
		InfluxDBToken:    strings.TrimSpace(os.Getenv("INFLUXDB_TOKEN")),
		InfluxDBDatabase: strings.TrimSpace(os.Getenv("INFLUXDB_DATABASE")),
		BatchSize:        envInt("BRIDGE_BATCH_SIZE", defaultBatchSize),
		FlushIntervalMs:  envInt("BRIDGE_FLUSH_INTERVAL_MS", defaultFlushIntervalMs),
		MaxBufferSize:    envInt("BRIDGE_MAX_BUFFER_SIZE", defaultMaxBufferSize),
		MaxWriteRetries:  envInt("BRIDGE_MAX_WRITE_RETRIES", defaultMaxWriteRetries),
		MQTTTLSEnabled:   envBool("MQTT_TLS_ENABLED", false),
	}

	if err := config.Validate(); err != nil {
		return BridgeConfig{}, err
	}

	return config, nil
}

func (c BridgeConfig) Validate() error {
	missing := make([]string, 0, 6)
	if c.MQTTBrokerURL == "" {
		missing = append(missing, "mqttBrokerUrl")
	}
	if c.MQTTUsername == "" {
		missing = append(missing, "mqttUsername")
	}
	if c.MQTTPassword == "" {
		missing = append(missing, "mqttPassword")
	}
	if c.InfluxDBURL == "" {
		missing = append(missing, "influxDbUrl")
	}
	if c.InfluxDBToken == "" {
		missing = append(missing, "influxDbToken")
	}
	if c.InfluxDBDatabase == "" {
		missing = append(missing, "influxDbDatabase")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required config values: %s", strings.Join(missing, ", "))
	}
	if c.BatchSize < 1 {
		return fmt.Errorf("batchSize must be >= 1")
	}
	if c.FlushIntervalMs < 10 {
		return fmt.Errorf("flushIntervalMs must be >= 10")
	}
	if c.MaxBufferSize < c.BatchSize {
		return fmt.Errorf("maxBufferSize must be >= batchSize")
	}
	if c.MaxWriteRetries < 0 {
		return fmt.Errorf("maxWriteRetries must be >= 0")
	}

	return nil
}

func (a DeviceAttribute) Fields() map[string]any {
	fields := make(map[string]any, 1)
	if a.ValueFloat != nil {
		fields["value_float"] = *a.ValueFloat
	}
	if a.ValueString != nil {
		fields["value_string"] = *a.ValueString
	}
	if a.ValueBool != nil {
		fields["value_bool"] = *a.ValueBool
	}
	return fields
}

func (a DeviceAttribute) Tags() map[string]string {
	return map[string]string{
		"account_id":         a.AccountID,
		"device_id":          a.DeviceID,
		"device_instance_id": a.DeviceInstanceID,
		"node_name":          a.NodeName,
		"attribute_name":     a.AttributeName,
	}
}

func envBool(name string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func boolOrDefault(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func intOrDefault(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}
