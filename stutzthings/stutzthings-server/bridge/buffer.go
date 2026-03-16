package bridge

import (
	"time"

	"github.com/sirupsen/logrus"
)

type messageBuffer struct {
	pending       []DeviceAttribute
	batchSize     int
	maxBufferSize int
}

type evictionInfo struct {
	droppedCount int
	oldestTime   time.Time
}

func newMessageBuffer(config BridgeConfig) *messageBuffer {
	return &messageBuffer{batchSize: config.BatchSize, maxBufferSize: config.MaxBufferSize}
}

func (b *messageBuffer) enqueue(attribute DeviceAttribute) evictionInfo {
	info := evictionInfo{}
	if len(b.pending) >= b.maxBufferSize {
		dropCount := b.batchSize
		if dropCount > len(b.pending) {
			dropCount = len(b.pending)
		}
		if dropCount > 0 {
			info.droppedCount = dropCount
			info.oldestTime = b.pending[0].Timestamp
			b.pending = append([]DeviceAttribute(nil), b.pending[dropCount:]...)
		}
	}
	b.pending = append(b.pending, attribute)
	return info
}

func (b *messageBuffer) nextBatch(force bool) []DeviceAttribute {
	if len(b.pending) == 0 {
		return nil
	}
	if !force && len(b.pending) < b.batchSize {
		return nil
	}
	limit := b.batchSize
	if force && len(b.pending) < limit {
		limit = len(b.pending)
	}
	batch := make([]DeviceAttribute, limit)
	copy(batch, b.pending[:limit])
	return batch
}

func (b *messageBuffer) consume(count int) {
	if count <= 0 || count > len(b.pending) {
		return
	}
	b.pending = append([]DeviceAttribute(nil), b.pending[count:]...)
}

func (b *messageBuffer) drainAll() int {
	dropped := len(b.pending)
	b.pending = nil
	return dropped
}

func (b *messageBuffer) len() int {
	return len(b.pending)
}

func (b *Bridge) runBuffer() {
	defer b.wg.Done()
	defer close(b.batchCh)

	buffer := newMessageBuffer(b.config)
	ticker := time.NewTicker(time.Duration(b.config.FlushIntervalMs) * time.Millisecond)
	defer ticker.Stop()

	var shutdownDeadline time.Time
	shuttingDown := false

	for {
		if shuttingDown {
			if buffer.len() == 0 {
				b.setBufferUsage(0)
				return
			}
			if time.Now().After(shutdownDeadline) {
				dropped := buffer.drainAll()
				if dropped > 0 {
					b.logger.WithFields(logrus.Fields{
						"event":         "shutdown_drop",
						"dropped_count": dropped,
					}).Warn("dropping buffered messages after shutdown deadline")
				}
				b.setBufferUsage(0)
				return
			}
			if b.tryDispatchBatch(buffer, true, shutdownDeadline) {
				continue
			}
			time.Sleep(25 * time.Millisecond)
			continue
		}

		select {
		case message := <-b.ingestCh:
			attribute, reason, err := ParseDeviceAttribute(message)
			if err != nil {
				b.logDiscard(message.Topic, message.Payload, reason, err)
				continue
			}
			evicted := buffer.enqueue(attribute)
			if evicted.droppedCount > 0 {
				b.logger.WithFields(logrus.Fields{
					"event":               "buffer_evicted",
					"dropped_count":       evicted.droppedCount,
					"oldest_dropped_time": evicted.oldestTime.Format(time.RFC3339Nano),
				}).Warn("evicted oldest buffered messages")
			}
			b.setBufferUsage(buffer.len())
			for b.tryDispatchBatch(buffer, false, time.Time{}) {
			}
		case <-ticker.C:
			b.tryDispatchBatch(buffer, true, time.Time{})
		case shutdownDeadline = <-b.shutdownCh:
			shuttingDown = true
		}
	}
}

func (b *Bridge) tryDispatchBatch(buffer *messageBuffer, force bool, deadline time.Time) bool {
	batch := buffer.nextBatch(force)
	if len(batch) == 0 {
		return false
	}

	if deadline.IsZero() {
		select {
		case b.batchCh <- batch:
			buffer.consume(len(batch))
			b.setBufferUsage(buffer.len())
			return true
		default:
			return false
		}
	}

	wait := time.Until(deadline)
	if wait <= 0 {
		return false
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case b.batchCh <- batch:
		buffer.consume(len(batch))
		b.setBufferUsage(buffer.len())
		return true
	case <-timer.C:
		return false
	}
}
