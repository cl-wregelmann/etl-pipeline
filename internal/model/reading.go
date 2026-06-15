// Package model defines the core data types that flow through the pipeline.
package model

import "time"

// RawReading is a single record exactly as it was read from a source file,
// before any parsing or validation. All fields are strings because the source
// data is untrusted and may be malformed.
type RawReading struct {
	SensorID  string
	Timestamp string
	Metric    string
	Value     string
	Unit      string

	// Provenance, useful for diagnostics and error messages.
	Source string // the file this record came from
	Line   int    // 1-based line number within the source file
}

// Reading is a validated, normalized sensor reading that is ready to load.
//
// Normalization rules (applied in the transform stage):
//   - temperature values are converted to degrees Celsius ("C")
//   - humidity is a percentage ("%")
//   - pressure is hectopascals ("hPa")
type Reading struct {
	SensorID  string
	Timestamp time.Time
	Metric    string  // temperature | humidity | pressure
	Value     float64 // normalized value
	Unit      string  // normalized unit: C | % | hPa
}
