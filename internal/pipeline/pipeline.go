// Package pipeline wires the extract, transform, and load stages together.
package pipeline

import (
	"fmt"
	"log"

	"github.com/cl-wregelmann/etl-pipeline/internal/extract"
	"github.com/cl-wregelmann/etl-pipeline/internal/load"
	"github.com/cl-wregelmann/etl-pipeline/internal/model"
	"github.com/cl-wregelmann/etl-pipeline/internal/transform"
)

// Options configures a pipeline run.
type Options struct {
	SourceDir string // directory of source files to read
	DBPath    string // path to the SQLite database to write
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

	readings := make([]model.Reading, 0, len(raw))
	for _, rr := range raw {
		reading, err := transform.One(rr)
		if err != nil {
			res.Skipped++
			log.Printf("skipping %s:%d: %v", rr.Source, rr.Line, err)
			continue
		}
		readings = append(readings, reading)
	}

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
