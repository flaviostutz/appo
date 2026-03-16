# appo Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-03-17

## Active Technologies
- Go 1.25.0 + Go `net/http`, `github.com/InfluxCommunity/influxdb3-go/v2`, `github.com/eclipse/paho.mqtt.golang`, `github.com/sirupsen/logrus`, `github.com/golang-jwt/jwt/v5`, `github.com/mark3labs/mcp-go` (002-device-ops-api)
- InfluxDB v3 measurement `device_attributes`; MQTT broker for desired-state publication; file/env configuration via `bridge.json` and environment variables (002-device-ops-api)

- Go 1.22+ (001-mqtt2influxdb-bridge)

## Project Structure

```text
src/
tests/
```

## Commands

# Add commands for Go 1.22+

## Code Style

Go 1.22+: Follow standard conventions

## Recent Changes
- 002-device-ops-api: Added Go 1.25.0 + Go `net/http`, `github.com/InfluxCommunity/influxdb3-go/v2`, `github.com/eclipse/paho.mqtt.golang`, `github.com/sirupsen/logrus`, `github.com/golang-jwt/jwt/v5`, `github.com/mark3labs/mcp-go`

- 001-mqtt2influxdb-bridge: Added Go 1.22+

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
