package transform

import (
	"math"
	"testing"

	"github.com/cl-wregelmann/etl-pipeline/internal/model"
)

func TestOne_NormalizesFahrenheit(t *testing.T) {
	got, err := One(model.RawReading{
		SensorID:  "sensor-02",
		Timestamp: "2026-01-01T00:05:00Z",
		Metric:    "temperature",
		Value:     "71.2",
		Unit:      "F",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Unit != "C" {
		t.Errorf("unit = %q, want C", got.Unit)
	}
	want := 21.777
	if math.Abs(got.Value-want) > 0.01 {
		t.Errorf("value = %.3f, want ~%.3f", got.Value, want)
	}
}

func TestOne_RejectsOutOfRange(t *testing.T) {
	_, err := One(model.RawReading{
		SensorID:  "sensor-02",
		Timestamp: "2026-01-01T00:20:00Z",
		Metric:    "temperature",
		Value:     "999",
		Unit:      "C",
	})
	if err == nil {
		t.Fatal("expected out-of-range error, got nil")
	}
}

func TestOne_RejectsBadInput(t *testing.T) {
	cases := map[string]model.RawReading{
		"unknown metric":   {SensorID: "s", Timestamp: "2026-01-01T00:00:00Z", Metric: "dewpoint", Value: "12", Unit: "C"},
		"bad timestamp":    {SensorID: "s", Timestamp: "not-a-timestamp", Metric: "temperature", Value: "20", Unit: "C"},
		"empty value":      {SensorID: "s", Timestamp: "2026-01-01T00:00:00Z", Metric: "temperature", Value: "", Unit: "C"},
		"missing sensorID": {SensorID: "", Timestamp: "2026-01-01T00:00:00Z", Metric: "humidity", Value: "50", Unit: "%"},
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := One(raw); err == nil {
				t.Errorf("expected error for %s, got nil", name)
			}
		})
	}
}
