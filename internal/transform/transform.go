// Package transform validates and normalizes raw readings into clean Readings.
package transform

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cl-wregelmann/etl-pipeline/internal/model"
)

// Reason is a stable, machine-readable category for why a record was rejected.
// Reason values are part of the run report's public contract: renaming or
// removing one is a breaking change.
type Reason string

// Rejection reasons returned in ValidationError.Reason.
const (
	ReasonUnknownMetric   Reason = "unknown_metric"
	ReasonMissingSensorID Reason = "missing_sensor_id"
	ReasonBadTimestamp    Reason = "bad_timestamp"
	ReasonBadValue        Reason = "bad_value"
	ReasonUnknownUnit     Reason = "unknown_unit"
	ReasonOutOfRange      Reason = "out_of_range"
)

// ValidationError is returned by One when a raw record is rejected. Reason
// categorizes the rejection; Msg is the human-readable detail.
type ValidationError struct {
	Reason Reason
	Msg    string
}

// Error returns the human-readable message, without the reason.
func (e *ValidationError) Error() string { return e.Msg }

func invalid(r Reason, format string, args ...any) error {
	return &ValidationError{Reason: r, Msg: fmt.Sprintf(format, args...)}
}

// knownMetrics lists the metrics the pipeline understands. Anything else is
// treated as invalid.
var knownMetrics = map[string]bool{
	"temperature": true,
	"humidity":    true,
	"pressure":    true,
}

// timestampLayouts are tried in order. Source files are not consistent about
// formatting, so we accept a few common shapes.
var timestampLayouts = []string{
	time.RFC3339,
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
}

// plausibleRange returns the accepted [min, max] range (in normalized units)
// for a metric. Values outside the range are rejected as bad data.
func plausibleRange(metric string) (min, max float64) {
	switch metric {
	case "temperature": // degrees Celsius
		return -90, 60
	case "humidity": // percent
		return 0, 100
	case "pressure": // hectopascals
		return 850, 1100
	}
	return 0, 0
}

// One converts a single raw record into a normalized Reading. It returns an
// error describing why the record is invalid; callers decide what to do with
// rejected records.
func One(raw model.RawReading) (model.Reading, error) {
	metric := strings.ToLower(strings.TrimSpace(raw.Metric))
	if !knownMetrics[metric] {
		return model.Reading{}, invalid(ReasonUnknownMetric, "unknown metric %q", raw.Metric)
	}

	sensorID := strings.TrimSpace(raw.SensorID)
	if sensorID == "" {
		return model.Reading{}, invalid(ReasonMissingSensorID, "missing sensor_id")
	}

	ts, err := parseTimestamp(raw.Timestamp)
	if err != nil {
		return model.Reading{}, err
	}

	value, err := strconv.ParseFloat(strings.TrimSpace(raw.Value), 64)
	if err != nil {
		return model.Reading{}, invalid(ReasonBadValue, "invalid value %q", raw.Value)
	}

	value, unit, err := normalizeUnit(metric, value, raw.Unit)
	if err != nil {
		return model.Reading{}, err
	}

	min, max := plausibleRange(metric)
	if value < min || value > max {
		return model.Reading{}, invalid(ReasonOutOfRange, "%s value %.2f%s out of range [%.0f, %.0f]", metric, value, unit, min, max)
	}

	return model.Reading{
		SensorID:  sensorID,
		Timestamp: ts.UTC(),
		Metric:    metric,
		Value:     value,
		Unit:      unit,
	}, nil
}

func parseTimestamp(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, layout := range timestampLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, invalid(ReasonBadTimestamp, "unparseable timestamp %q", s)
}

// normalizeUnit converts a value to the canonical unit for its metric and
// returns the normalized value and unit.
func normalizeUnit(metric string, value float64, rawUnit string) (float64, string, error) {
	unit := strings.TrimSpace(rawUnit)
	switch metric {
	case "temperature":
		switch strings.ToUpper(unit) {
		case "C", "°C", "":
			return value, "C", nil
		case "F", "°F":
			return (value - 32) * 5 / 9, "C", nil
		case "K":
			return value - 273.15, "C", nil
		}
		return 0, "", invalid(ReasonUnknownUnit, "unknown temperature unit %q", rawUnit)
	case "humidity":
		switch unit {
		case "%", "":
			return value, "%", nil
		}
		return 0, "", invalid(ReasonUnknownUnit, "unknown humidity unit %q", rawUnit)
	case "pressure":
		switch strings.ToLower(unit) {
		case "hpa", "":
			return value, "hPa", nil
		case "pa":
			return value / 100, "hPa", nil
		case "kpa":
			return value * 10, "hPa", nil
		}
		return 0, "", invalid(ReasonUnknownUnit, "unknown pressure unit %q", rawUnit)
	}
	return 0, "", invalid(ReasonUnknownMetric, "unknown metric %q", metric)
}
