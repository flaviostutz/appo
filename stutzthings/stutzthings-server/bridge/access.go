package bridge

import "context"

func (b *Bridge) QueryRows(ctx context.Context, query string) ([]map[string]any, error) {
	return b.influx.QueryRows(ctx, query)
}

func (b *Bridge) Publish(ctx context.Context, topic string, qos byte, retained bool, payload []byte) error {
	if b.mqtt == nil {
		return errMQTTNotConnected
	}
	return b.mqtt.Publish(ctx, topic, qos, retained, payload)
}
