package bridge

import (
	"context"
	"fmt"
	"time"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
	"github.com/cenkalti/backoff/v4"
	lineprotocol "github.com/influxdata/line-protocol/v2/lineprotocol"
	"github.com/sirupsen/logrus"
)

type influxRuntime struct {
	client *influxdb3.Client
	config BridgeConfig
	logger *logrus.Entry
}

func newInfluxRuntime(config BridgeConfig, logger *logrus.Entry) (*influxRuntime, error) {
	client, err := influxdb3.New(influxdb3.ClientConfig{
		Host:         config.InfluxDBURL,
		Token:        config.InfluxDBToken,
		AuthScheme:   "Bearer",
		Database:     config.InfluxDBDatabase,
		WriteTimeout: 10 * time.Second,
		QueryTimeout: 10 * time.Second,
		WriteOptions: &influxdb3.WriteOptions{Precision: lineprotocol.Millisecond},
	})
	if err != nil {
		return nil, fmt.Errorf("create influxdb client: %w", err)
	}

	return &influxRuntime{client: client, config: config, logger: logger}, nil
}

func (r *influxRuntime) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

func (r *influxRuntime) Probe(ctx context.Context) error {
	iterator, err := r.client.Query(ctx, "SHOW TABLES", influxdb3.WithDatabase(r.config.InfluxDBDatabase))
	if err != nil {
		return err
	}
	for iterator.Next() {
	}
	return iterator.Err()
}

func (r *influxRuntime) WriteBatch(ctx context.Context, batch []DeviceAttribute) error {
	points := make([]*influxdb3.Point, 0, len(batch))
	for _, attribute := range batch {
		points = append(points, influxdb3.NewPoint(measurementName, attribute.Tags(), attribute.Fields(), attribute.Timestamp))
	}
	return r.client.WritePoints(ctx, points, influxdb3.WithPrecision(lineprotocol.Millisecond))
}

func (r *influxRuntime) QueryRows(ctx context.Context, query string) ([]map[string]any, error) {
	iterator, err := r.client.Query(ctx, query, influxdb3.WithDatabase(r.config.InfluxDBDatabase), influxdb3.WithPrecision(lineprotocol.Millisecond))
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]any, 0)
	for iterator.Next() {
		row := iterator.Value()
		clone := make(map[string]any, len(row))
		for key, value := range row {
			clone[key] = value
		}
		rows = append(rows, clone)
	}
	return rows, iterator.Err()
}

func (b *Bridge) probeInfluxUntilReady(ctx context.Context) error {
	policy := backoff.NewExponentialBackOff()
	policy.InitialInterval = time.Second
	policy.MaxInterval = 30 * time.Second
	policy.Multiplier = 2
	policy.MaxElapsedTime = 0

	attempt := 0
	for {
		attempt++
		started := time.Now()
		probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := b.influx.Probe(probeCtx)
		cancel()
		if err == nil {
			b.logger.WithField("event", "influx_ready").Info("InfluxDB reachable")
			b.setInfluxReachable(true, time.Since(started), "All dependencies operational")
			return nil
		}

		b.logger.WithFields(logrus.Fields{
			"event":       "influx_probe_retry",
			"attempt":     attempt,
			"max_retries": 0,
		}).WithError(err).Warn("InfluxDB probe failed")
		b.setInfluxReachable(false, time.Since(started), "InfluxDB is unreachable")

		waitFor := policy.NextBackOff()
		select {
		case <-time.After(waitFor):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (b *Bridge) runWriter() {
	defer b.wg.Done()
	for batch := range b.batchCh {
		b.writeBatchWithRetry(batch)
	}
}

func (b *Bridge) writeBatchWithRetry(batch []DeviceAttribute) {
	maxAttempts := b.config.MaxWriteRetries
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		ctx, cancel := b.shutdownContext()
		err := b.influx.WriteBatch(ctx, batch)
		cancel()
		if err == nil {
			b.setInfluxReachable(true, 0, "All dependencies operational")
			return
		}

		if attempt == maxAttempts {
			b.logger.WithFields(logrus.Fields{
				"event":         "influx_batch_discarded",
				"attempt":       attempt,
				"max_retries":   maxAttempts,
				"dropped_count": len(batch),
			}).WithError(err).Warn("dropping batch after write retries exhausted")
			b.setInfluxReachable(false, 0, "InfluxDB write retries exhausted")
			return
		}

		b.logger.WithFields(logrus.Fields{
			"event":       "influx_retry",
			"attempt":     attempt,
			"max_retries": maxAttempts,
		}).WithError(err).Warn("retrying InfluxDB batch write")

		delay := time.Duration(attempt) * 500 * time.Millisecond
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
		case <-time.After(time.Until(b.HealthStatusDeadline())):
			timer.Stop()
			return
		}
	}
}

func (b *Bridge) HealthStatusDeadline() time.Time {
	b.stateMu.RLock()
	defer b.stateMu.RUnlock()
	return b.shutdownDeadline
}
