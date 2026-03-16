package bridge

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

type Bridge struct {
	config BridgeConfig
	logger *logrus.Entry

	influx *influxRuntime
	mqtt   *mqttRuntime

	ingestCh   chan MQTTMessage
	batchCh    chan []DeviceAttribute
	shutdownCh chan time.Time
	doneCh     chan struct{}

	started   atomic.Bool
	accepting atomic.Bool
	healthMu  sync.Mutex

	stateMu          sync.RWMutex
	mqttConnected    bool
	influxReachable  bool
	bufferUsage      int
	healthCheckedAt  time.Time
	shutdownDeadline time.Time
	healthStatus     BridgeHealthStatus

	stopOnce sync.Once
	wg       sync.WaitGroup
}

func NewBridge(config BridgeConfig, logger *logrus.Logger) (*Bridge, error) {
	if logger == nil {
		logger = logrus.New()
	}
	entry := logger.WithField("component", "bridge")

	influx, err := newInfluxRuntime(config, entry)
	if err != nil {
		return nil, err
	}

	bridge := &Bridge{
		config:     config,
		logger:     entry,
		influx:     influx,
		ingestCh:   make(chan MQTTMessage, config.MaxBufferSize),
		batchCh:    make(chan []DeviceAttribute, 4),
		shutdownCh: make(chan time.Time, 1),
		doneCh:     make(chan struct{}),
	}
	bridge.updateHealth("bridge not started")

	return bridge, nil
}

func (b *Bridge) Start(ctx context.Context) error {
	if b.started.Load() {
		return fmt.Errorf("bridge already started")
	}

	if err := b.probeInfluxUntilReady(ctx); err != nil {
		return err
	}

	b.wg.Add(2)
	go b.runWriter()
	go b.runBuffer()
	go func() {
		b.wg.Wait()
		close(b.doneCh)
	}()

	mqttRuntime, err := newMQTTRuntime(b.config, b.logger, b.handleMQTTMessage, b.setMQTTConnected)
	if err != nil {
		return err
	}
	b.mqtt = mqttRuntime
	b.accepting.Store(true)

	if err := b.mqtt.Connect(ctx); err != nil {
		b.accepting.Store(false)
		return err
	}

	b.started.Store(true)
	go func() {
		<-ctx.Done()
		_ = b.Stop()
	}()

	return nil
}

func (b *Bridge) Stop() error {
	var stopErr error
	b.stopOnce.Do(func() {
		b.accepting.Store(false)

		deadline := time.Now().Add(5 * time.Second)
		b.stateMu.Lock()
		b.shutdownDeadline = deadline
		b.stateMu.Unlock()

		if b.mqtt != nil {
			b.mqtt.Disconnect(250)
		}

		select {
		case b.shutdownCh <- deadline:
		default:
		}

		select {
		case <-b.doneCh:
		case <-time.After(6 * time.Second):
			stopErr = errors.New("bridge shutdown timed out")
		}

		if b.influx != nil {
			if err := b.influx.Close(); err != nil && stopErr == nil {
				stopErr = err
			}
		}
	})

	return stopErr
}

func (b *Bridge) HealthStatus() BridgeHealthStatus {
	b.stateMu.RLock()
	defer b.stateMu.RUnlock()
	return b.healthStatus
}

func (b *Bridge) CheckHealth(ctx context.Context) BridgeHealthStatus {
	if status, ok := b.cachedHealthStatus(); ok {
		return status
	}

	b.healthMu.Lock()
	defer b.healthMu.Unlock()

	if status, ok := b.cachedHealthStatus(); ok {
		return status
	}

	return b.refreshHealthStatus(ctx)
}

func (b *Bridge) handleMQTTMessage(topic string, payload []byte) {
	if !b.accepting.Load() {
		return
	}

	message := MQTTMessage{Topic: topic, Payload: append([]byte(nil), payload...), ReceivedAt: time.Now().UTC()}
	select {
	case b.ingestCh <- message:
	case <-time.After(2 * time.Second):
		b.logger.WithFields(logrus.Fields{
			"event":          "message_discarded",
			"reason":         "ingest_backpressure",
			"topic":          topic,
			"payload_sample": samplePayload(payload),
		}).Warn("dropping MQTT message after ingest channel timeout")
	}
}

func (b *Bridge) setMQTTConnected(connected bool) {
	b.stateMu.Lock()
	defer b.stateMu.Unlock()
	b.mqttConnected = connected
	b.healthCheckedAt = time.Time{}
	b.recomputeHealthLocked()
}

func (b *Bridge) setInfluxReachable(reachable bool, latency time.Duration, message string) {
	b.stateMu.Lock()
	defer b.stateMu.Unlock()
	b.influxReachable = reachable
	b.healthCheckedAt = time.Time{}
	if message != "" {
		b.healthStatus.Message = message
	}
	b.recomputeHealthLocked()
}

func (b *Bridge) setBufferUsage(usage int) {
	b.stateMu.Lock()
	defer b.stateMu.Unlock()
	b.bufferUsage = usage
	b.healthCheckedAt = time.Time{}
	b.recomputeHealthLocked()
}

func (b *Bridge) updateHealth(message string) {
	b.stateMu.Lock()
	defer b.stateMu.Unlock()
	b.healthStatus.Message = message
	b.healthCheckedAt = time.Time{}
	b.recomputeHealthLocked()
}

func (b *Bridge) cachedHealthStatus() (BridgeHealthStatus, bool) {
	b.stateMu.RLock()
	defer b.stateMu.RUnlock()
	if b.healthCheckedAt.IsZero() || time.Since(b.healthCheckedAt) > 5*time.Second {
		return BridgeHealthStatus{}, false
	}
	return b.healthStatus, true
}

func (b *Bridge) refreshHealthStatus(ctx context.Context) BridgeHealthStatus {
	started := time.Now()
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	type healthCheckResult struct {
		name string
		err  error
	}

	results := make(chan healthCheckResult, 2)
	go func() {
		results <- healthCheckResult{name: "mqtt", err: b.mqtt.Probe(checkCtx, b.config)}
	}()
	go func() {
		results <- healthCheckResult{name: "influx", err: b.influx.Probe(checkCtx)}
	}()

	var mqttErr error
	var influxErr error
	for range 2 {
		result := <-results
		switch result.name {
		case "mqtt":
			mqttErr = result.err
		case "influx":
			influxErr = result.err
		}
	}

	b.stateMu.Lock()
	defer b.stateMu.Unlock()

	b.mqttConnected = mqttErr == nil
	b.influxReachable = influxErr == nil
	b.healthStatus.MQTTConnected = b.mqttConnected
	b.healthStatus.InfluxReachable = b.influxReachable
	b.healthStatus.BufferUsage = b.bufferUsage
	b.healthStatus.LatencyMs = time.Since(started).Milliseconds()

	switch {
	case influxErr != nil:
		b.healthStatus.Health = HealthError
		b.healthStatus.Message = "InfluxDB is unreachable"
	case mqttErr != nil:
		b.healthStatus.Health = HealthWarning
		b.healthStatus.Message = "MQTT is reconnecting"
	case b.bufferUsage >= int(float64(b.config.MaxBufferSize)*0.8):
		b.healthStatus.Health = HealthWarning
		b.healthStatus.Message = "buffer usage is elevated"
	default:
		b.healthStatus.Health = HealthOK
		b.healthStatus.Message = "All dependencies operational"
	}

	b.healthCheckedAt = time.Now()
	return b.healthStatus
}

func (b *Bridge) recomputeHealthLocked() {
	b.healthStatus.MQTTConnected = b.mqttConnected
	b.healthStatus.InfluxReachable = b.influxReachable
	b.healthStatus.BufferUsage = b.bufferUsage

	switch {
	case !b.influxReachable:
		b.healthStatus.Health = HealthError
		if b.healthStatus.Message == "" {
			b.healthStatus.Message = "InfluxDB is unreachable"
		}
	case !b.mqttConnected:
		b.healthStatus.Health = HealthWarning
		if b.healthStatus.Message == "" {
			b.healthStatus.Message = "MQTT is reconnecting"
		}
	case b.bufferUsage >= int(float64(b.config.MaxBufferSize)*0.8):
		b.healthStatus.Health = HealthWarning
		if b.healthStatus.Message == "" {
			b.healthStatus.Message = "buffer usage is elevated"
		}
	default:
		b.healthStatus.Health = HealthOK
		if b.healthStatus.Message == "" {
			b.healthStatus.Message = "All dependencies operational"
		}
	}
}

func (b *Bridge) logDiscard(topic string, payload []byte, reason discardReason, err error) {
	b.logger.WithFields(logrus.Fields{
		"event":          "message_discarded",
		"reason":         string(reason),
		"topic":          topic,
		"payload_sample": samplePayload(payload),
	}).WithError(err).Warn("discarded MQTT message")
}

func (b *Bridge) shutdownContext() (context.Context, context.CancelFunc) {
	b.stateMu.RLock()
	deadline := b.shutdownDeadline
	b.stateMu.RUnlock()

	if deadline.IsZero() {
		return context.WithTimeout(context.Background(), 10*time.Second)
	}
	return context.WithDeadline(context.Background(), deadline)
}
