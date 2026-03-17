package operations

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/auth"
)

type performanceStore struct {
	deviceRows  []map[string]any
	historyRows []map[string]any
}

func (p performanceStore) QueryRows(_ context.Context, query string) ([]map[string]any, error) {
	if strings.Contains(query, "LIMIT 10001") {
		return p.historyRows, nil
	}
	return p.deviceRows, nil
}

func BenchmarkGetDeviceState(b *testing.B) {
	service := NewService(performanceStore{deviceRows: buildDeviceRows(20)}, &fakePublisher{}, nil)
	claims := &auth.Claims{Publ: []string{"acct/device/instance/+/+"}, Subs: []string{"acct/device/instance/+/+/set"}}
	scope := DeviceScope{AccountID: "acct", DeviceID: "device", DeviceInstanceID: "instance"}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.GetDeviceState(context.Background(), claims, scope)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetAttributeHistory(b *testing.B) {
	service := NewService(performanceStore{historyRows: buildHistoryRows(2880)}, &fakePublisher{}, nil)
	claims := &auth.Claims{Publ: []string{"acct/device/instance/node/temperature"}, Subs: []string{"acct/device/instance/+/+/set"}}
	request := AttributeHistoryRequest{
		DeviceIdentity: DeviceIdentity{
			DeviceScope:   DeviceScope{AccountID: "acct", DeviceID: "device", DeviceInstanceID: "instance"},
			NodeName:      "node",
			AttributeName: "temperature",
		},
		From: time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC),
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.GetAttributeHistory(context.Background(), claims, request)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func buildDeviceRows(attributeCount int) []map[string]any {
	rows := make([]map[string]any, 0, attributeCount)
	for i := 0; i < attributeCount; i++ {
		rows = append(rows, makeObservationRow("node", fmt.Sprintf("attribute-%02d", i), float64(i), "2026-03-17T00:00:00Z"))
	}
	return rows
}

func buildHistoryRows(observationCount int) []map[string]any {
	rows := make([]map[string]any, 0, observationCount)
	start := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	for i := 0; i < observationCount; i++ {
		rows = append(rows, makeObservationRow("node", "temperature", float64(i%100), start.Add(time.Duration(i)*time.Minute).Format(time.RFC3339)))
	}
	return rows
}
