package bridge

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageBufferFlushesByBatchSize(t *testing.T) {
	buffer := newMessageBuffer(BridgeConfig{BatchSize: 2, MaxBufferSize: 4})
	buffer.enqueue(makeAttribute(time.Unix(1, 0)))
	buffer.enqueue(makeAttribute(time.Unix(2, 0)))

	batch := buffer.nextBatch(false)
	require.Len(t, batch, 2)
	buffer.consume(len(batch))
	assert.Equal(t, 0, buffer.len())
}

func TestMessageBufferEvictsOldestBatch(t *testing.T) {
	buffer := newMessageBuffer(BridgeConfig{BatchSize: 2, MaxBufferSize: 3})
	buffer.enqueue(makeAttribute(time.Unix(1, 0)))
	buffer.enqueue(makeAttribute(time.Unix(2, 0)))
	buffer.enqueue(makeAttribute(time.Unix(3, 0)))

	evicted := buffer.enqueue(makeAttribute(time.Unix(4, 0)))

	assert.Equal(t, 2, evicted.droppedCount)
	assert.Equal(t, time.Unix(1, 0), evicted.oldestTime)
	assert.Equal(t, 2, buffer.len())
	assert.Equal(t, time.Unix(3, 0), buffer.pending[0].Timestamp)
	assert.Equal(t, time.Unix(4, 0), buffer.pending[1].Timestamp)
}

func TestMessageBufferForceFlushesPartialBatch(t *testing.T) {
	buffer := newMessageBuffer(BridgeConfig{BatchSize: 5, MaxBufferSize: 10})
	buffer.enqueue(makeAttribute(time.Unix(1, 0)))

	batch := buffer.nextBatch(true)
	require.Len(t, batch, 1)
	assert.Equal(t, time.Unix(1, 0), batch[0].Timestamp)
}

func makeAttribute(timestamp time.Time) DeviceAttribute {
	value := 1.0
	return DeviceAttribute{
		AccountID:        "account1",
		DeviceID:         "deviceA",
		DeviceInstanceID: "device01",
		NodeName:         "node",
		AttributeName:    "attribute",
		ValueFloat:       &value,
		Timestamp:        timestamp,
	}
}
