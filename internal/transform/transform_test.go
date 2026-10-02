package transform

import (
	"errors"
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
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("error %v is %T, want *ValidationError", err, err)
	}
	if ve.Reason != ReasonOutOfRange {
		t.Errorf("reason = %q, want %q", ve.Reason, ReasonOutOfRange)
	}
	// The message feeds the per-record skip log line, which must not change.
	if want := "temperature value 999.00C out of range [-90, 60]"; err.Error() != want {
		t.Errorf("message = %q, want %q", err.Error(), want)
	}
}

func TestOne_RejectsBadInput(t *testing.T) {
	cases := map[string]struct {
		raw  model.RawReading
		want Reason
	}{
		"unknown metric":        {model.RawReading{SensorID: "s", Timestamp: "2026-01-01T00:00:00Z", Metric: "dewpoint", Value: "12", Unit: "C"}, ReasonUnknownMetric},
		"bad timestamp":         {model.RawReading{SensorID: "s", Timestamp: "not-a-timestamp", Metric: "temperature", Value: "20", Unit: "C"}, ReasonBadTimestamp},
		"empty value":           {model.RawReading{SensorID: "s", Timestamp: "2026-01-01T00:00:00Z", Metric: "temperature", Value: "", Unit: "C"}, ReasonBadValue},
		"missing sensorID":      {model.RawReading{SensorID: "", Timestamp: "2026-01-01T00:00:00Z", Metric: "humidity", Value: "50", Unit: "%"}, ReasonMissingSensorID},
		"unknown temp unit":     {model.RawReading{SensorID: "s", Timestamp: "2026-01-01T00:00:00Z", Metric: "temperature", Value: "20", Unit: "X"}, ReasonUnknownUnit},
		"unknown pressure unit": {model.RawReading{SensorID: "s", Timestamp: "2026-01-01T00:00:00Z", Metric: "pressure", Value: "1013", Unit: "psi"}, ReasonUnknownUnit},
		"humidity out of range": {model.RawReading{SensorID: "s", Timestamp: "2026-01-01T00:00:00Z", Metric: "humidity", Value: "150", Unit: "%"}, ReasonOutOfRange},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := One(tc.raw)
			if err == nil {
				t.Fatalf("expected error for %s, got nil", name)
			}
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("error %v is %T, want *ValidationError", err, err)
			}
			if ve.Reason != tc.want {
				t.Errorf("reason = %q, want %q", ve.Reason, tc.want)
			}
		})
	}
}
