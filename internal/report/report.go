// Package report builds and writes the JSON run report that downstream systems
// ingest after each successful pipeline run.
//
// The report format is versioned by SchemaVersion. Renaming or removing a
// field, or renaming or removing a skip reason key, is a breaking change and
// requires bumping SchemaVersion. Adding a field or a new skip reason is not;
// consumers must tolerate unknown fields and reason keys.
package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cl-wregelmann/etl-pipeline/internal/pipeline"
)

// SchemaVersion is bumped on any breaking change to the report format.
const SchemaVersion = 1

// Report is the v1 JSON run report.
type Report struct {
	SchemaVersion int            `json:"schema_version"`
	StartedAt     time.Time      `json:"started_at"`
	FinishedAt    time.Time      `json:"finished_at"`
	DurationMS    int64          `json:"duration_ms"`
	SourceDir     string         `json:"source_dir"`
	DBPath        string         `json:"db_path"`
	Counts        Counts         `json:"counts"`
	SkipReasons   map[string]int `json:"skip_reasons"`
}

// Counts holds the record totals for a run.
type Counts struct {
	Read    int `json:"read"`
	Skipped int `json:"skipped"`
	Loaded  int `json:"loaded"`
}

// New builds a Report from a completed pipeline run.
func New(res pipeline.Result, opts pipeline.Options) Report {
	// Non-nil so an empty breakdown encodes as {} rather than null.
	reasons := make(map[string]int, len(res.SkipReasons))
	for reason, n := range res.SkipReasons {
		if n > 0 {
			reasons[string(reason)] = n
		}
	}
	return Report{
		SchemaVersion: SchemaVersion,
		StartedAt:     res.StartedAt,
		FinishedAt:    res.FinishedAt,
		DurationMS:    res.FinishedAt.Sub(res.StartedAt).Milliseconds(),
		SourceDir:     opts.SourceDir,
		DBPath:        opts.DBPath,
		Counts:        Counts{Read: res.Read, Skipped: res.Skipped, Loaded: res.Loaded},
		SkipReasons:   reasons,
	}
}

// WriteFile writes r as indented JSON to path, creating parent directories.
// The write is atomic: a temp file in the same directory is renamed into place,
// so a downstream reader never sees a partially written report.
func WriteFile(path string, r Report) (err error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("encode report: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil { // #nosec G301 -- report dir must be readable by downstream ingest
		return err
	}
	// The temp file must live in the target directory so the rename stays on
	// one filesystem and is atomic.
	tmp, err := os.CreateTemp(dir, ".run-report-*.json")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()

	if _, err = tmp.Write(data); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	// CreateTemp uses 0o600; the report is meant to be read by downstream tools.
	if err = os.Chmod(tmp.Name(), 0o644); err != nil { // #nosec G302 -- report is not sensitive and must be world-readable
		return err
	}
	return os.Rename(tmp.Name(), path)
}
