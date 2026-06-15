// Command etl runs the sensor-reading ETL pipeline.
//
// Usage:
//
//	etl [flags]
//
// It reads every *.csv file in the source directory, validates and normalizes
// each reading, and loads the results into a SQLite database.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/cl-wregelmann/etl-pipeline/internal/pipeline"
)

func main() {
	log.SetFlags(0)

	source := flag.String("source", "data/raw", "directory of source CSV files to ingest")
	db := flag.String("db", "data/etl.db", "path to the SQLite database to write")
	flag.Parse()

	res, err := pipeline.Run(pipeline.Options{
		SourceDir: *source,
		DBPath:    *db,
	})
	if err != nil {
		log.Printf("pipeline failed: %v", err)
		os.Exit(1)
	}

	fmt.Printf("done: read %d, skipped %d, loaded %d -> %s\n", res.Read, res.Skipped, res.Loaded, *db)
}
