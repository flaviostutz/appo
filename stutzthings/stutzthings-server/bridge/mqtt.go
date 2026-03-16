package bridge

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
)

type mqttRuntime struct {
	client mqtt.Client
	logger *logrus.Entry
}

func newMQTTRuntime(config BridgeConfig, logger *logrus.Entry, onMessage func(string, []byte), onConnected func(bool)) (*mqttRuntime, error) {
	brokerURL, err := normalizeMQTTBrokerURL(config.MQTTBrokerURL, config.MQTTTLSEnabled)
	if err != nil {
		return nil, err
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerURL)
	opts.SetClientID(defaultClientID())
	opts.SetUsername(config.MQTTUsername)
	opts.SetPassword(config.MQTTPassword)
	opts.SetProtocolVersion(4)
	opts.SetOrderMatters(false)
	opts.SetConnectTimeout(15 * time.Second)
	opts.SetMaxReconnectInterval(10 * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		onConnected(false)
		logger.WithFields(logrus.Fields{"event": "mqtt_connection_lost"}).WithError(err).Warn("MQTT connection lost")
	})
	opts.SetReconnectingHandler(func(_ mqtt.Client, _ *mqtt.ClientOptions) {
		logger.WithFields(logrus.Fields{"event": "mqtt_reconnecting"}).Info("reconnecting to MQTT broker")
	})
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		token := client.Subscribe("#", 1, func(_ mqtt.Client, message mqtt.Message) {
			onMessage(message.Topic(), message.Payload())
		})
		token.Wait()
		if err := token.Error(); err != nil {
			onConnected(false)
			logger.WithFields(logrus.Fields{"event": "mqtt_subscribe_failed"}).WithError(err).Error("subscribe to wildcard topic failed")
			return
		}
		onConnected(true)
		logger.WithFields(logrus.Fields{"event": "mqtt_connected", "broker": brokerURL}).Info("connected to MQTT broker")
	})

	if config.MQTTTLSEnabled {
		opts.SetTLSConfig(&tls.Config{MinVersion: tls.VersionTLS12})
	}

	return &mqttRuntime{client: mqtt.NewClient(opts), logger: logger}, nil
}

func (r *mqttRuntime) Connect(ctx context.Context) error {
	token := r.client.Connect()
	done := make(chan struct{})
	go func() {
		defer close(done)
		token.Wait()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return token.Error()
	}
}

func (r *mqttRuntime) Disconnect(quiesce uint) {
	if r.client != nil && r.client.IsConnected() {
		r.client.Disconnect(quiesce)
	}
}

func (r *mqttRuntime) Probe(ctx context.Context, config BridgeConfig) error {
	if r == nil || r.client == nil || !r.client.IsConnected() {
		return errors.New("mqtt client is not connected")
	}

	brokerURL, err := normalizeMQTTBrokerURL(config.MQTTBrokerURL, config.MQTTTLSEnabled)
	if err != nil {
		return err
	}

	parsed, err := url.Parse(brokerURL)
	if err != nil {
		return fmt.Errorf("parse mqtt broker url: %w", err)
	}

	address := parsed.Host
	if parsed.Port() == "" {
		switch parsed.Scheme {
		case "ssl", "wss":
			address = net.JoinHostPort(parsed.Hostname(), "8883")
		default:
			address = net.JoinHostPort(parsed.Hostname(), "1883")
		}
	}

	dialer := &net.Dialer{Timeout: 5 * time.Second}
	var connection net.Conn
	switch parsed.Scheme {
	case "ssl", "wss":
		tlsDialer := &tls.Dialer{NetDialer: dialer, Config: &tls.Config{MinVersion: tls.VersionTLS12}}
		connection, err = tlsDialer.DialContext(ctx, "tcp", address)
	default:
		connection, err = dialer.DialContext(ctx, "tcp", address)
	}
	if err != nil {
		return err
	}
	return connection.Close()
}

func normalizeMQTTBrokerURL(rawURL string, tlsEnabled bool) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse mqtt url: %w", err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	switch scheme {
	case "mqtt", "tcp":
		if tlsEnabled {
			parsed.Scheme = "ssl"
		} else {
			parsed.Scheme = "tcp"
		}
	case "mqtts", "ssl", "tls":
		parsed.Scheme = "ssl"
	case "ws", "wss":
		if tlsEnabled || scheme == "wss" {
			parsed.Scheme = "wss"
		}
	default:
		return "", fmt.Errorf("unsupported mqtt scheme %q", parsed.Scheme)
	}

	return parsed.String(), nil
}

func defaultClientID() string {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "unknown"
	}
	return fmt.Sprintf("stutzthings-bridge-%s", hostname)
}
