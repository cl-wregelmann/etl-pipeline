// Package pipeline wires the extract, transform, and load stages together.
package pipeline

import (
	"fmt"

	"github.com/cl-wregelmann/etl-pipeline/internal/extract"
	"github.com/cl-wregelmann/etl-pipeline/internal/load"
	"github.com/cl-wregelmann/etl-pipeline/internal/transform"
)

// Options configures a pipeline run.
type Options struct {
	SourceDir string // directory of source files to read
	DBPath    string // path to the SQLite database to write
	Workers   int    // transport worker count;<=0 uses runtime.NumCPU()
}

// Result reports what happened during a run.
type Result struct {
	Read    int // raw records read from source files
	Skipped int // records that failed transformation/validation
	Loaded  int // readings written to the database
}

// Run executes the full extract -> transform -> load pipeline.
//
// The transform stage runs sequentially over every raw record. Records that
// fail validation are skipped: a warning is logged and the run continues.
func Run(opts Options) (Result, error) {
	var res Result

	raw, err := extract.FromDir(opts.SourceDir)
	if err != nil {
		return res, fmt.Errorf("extract: %w", err)
	}
	res.Read = len(raw)

	readings, skipped := transform.RunParallel(raw, opts.Workers)
	res.Skipped = skipped

	loader, err := load.Open(opts.DBPath)
	if err != nil {
		return res, fmt.Errorf("load: %w", err)
	}
	defer loader.Close()

	n, err := loader.Insert(readings)
	if err != nil {
		return res, fmt.Errorf("load: %w", err)
	}
	res.Loaded = n

	return res, nil
}
