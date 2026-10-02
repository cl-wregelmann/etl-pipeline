// Command etl runs the sensor-reading ETL pipeline.
//
// Usage:
//
//	etl [flags]
//
// It reads every *.csv file in the source directory, validates and normalizes
// each reading, and loads the results into a SQLite database. After a
// successful run it writes a JSON run report (see package report) and prints a
// one-line summary to stdout.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/cl-wregelmann/etl-pipeline/internal/pipeline"
	"github.com/cl-wregelmann/etl-pipeline/internal/report"
)

func main() {
	log.SetFlags(0)

	source := flag.String("source", "data/raw", "directory of source CSV files to ingest")
	db := flag.String("db", "data/etl.db", "path to the SQLite database to write")
	reportPath := flag.String("report", "data/run-report.json", "path to write the JSON run report")
	flag.Parse()

	opts := pipeline.Options{
		SourceDir: *source,
		DBPath:    *db,
	}
	res, err := pipeline.Run(opts)
	if err != nil {
		log.Printf("pipeline failed: %v", err)
		os.Exit(1)
	}

	// A missing report breaks the downstream contract, so the run fails even
	// though the load has already committed.
	if err := report.WriteFile(*reportPath, report.New(res, opts)); err != nil {
		log.Printf("pipeline failed: write report (database already loaded): %v", err)
		os.Exit(1)
	}

	fmt.Println(res.OneLine(*db, *reportPath))
}
