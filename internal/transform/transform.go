// Package transform validates and normalizes raw readings into clean Readings.
package transform

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cl-wregelmann/etl-pipeline/internal/model"
)

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
		return model.Reading{}, fmt.Errorf("unknown metric %q", raw.Metric)
	}

	sensorID := strings.TrimSpace(raw.SensorID)
	if sensorID == "" {
		return model.Reading{}, fmt.Errorf("missing sensor_id")
	}

	ts, err := parseTimestamp(raw.Timestamp)
	if err != nil {
		return model.Reading{}, err
	}

	value, err := strconv.ParseFloat(strings.TrimSpace(raw.Value), 64)
	if err != nil {
		return model.Reading{}, fmt.Errorf("invalid value %q", raw.Value)
	}

	value, unit, err := normalizeUnit(metric, value, raw.Unit)
	if err != nil {
		return model.Reading{}, err
	}

	min, max := plausibleRange(metric)
	if value < min || value > max {
		return model.Reading{}, fmt.Errorf("%s value %.2f%s out of range [%.0f, %.0f]", metric, value, unit, min, max)
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
	return time.Time{}, fmt.Errorf("unparseable timestamp %q", s)
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
		return 0, "", fmt.Errorf("unknown temperature unit %q", rawUnit)
	case "humidity":
		switch unit {
		case "%", "":
			return value, "%", nil
		}
		return 0, "", fmt.Errorf("unknown humidity unit %q", rawUnit)
	case "pressure":
		switch strings.ToLower(unit) {
		case "hpa", "":
			return value, "hPa", nil
		case "pa":
			return value / 100, "hPa", nil
		case "kpa":
			return value * 10, "hPa", nil
		}
		return 0, "", fmt.Errorf("unknown pressure unit %q", rawUnit)
	}
	return 0, "", fmt.Errorf("unknown metric %q", metric)
}
