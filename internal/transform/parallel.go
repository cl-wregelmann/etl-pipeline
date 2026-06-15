package transform

import (
	"log"
	"runtime"
	"sync"

	"github.com/cl-wregelmann/etl-pipeline/internal/model"
)

type parallelResult struct {
	reading model.Reading
	err     error
}

// RunParallel converts raw readings concurrently and returns valid readings
// plus the number of skipped records.
func RunParallel(raws []model.RawReading, worker int) ([]model.Reading, int) {
	if worker <= 0 {
		worker = runtime.NumCPU()
	}
	if worker > len(raws) {
		worker = len(raws)
	}

	readings := make([]model.Reading, 0, len(raws))
	if worker == 0 {
		return readings, 0
	}

	jobs := make(chan model.RawReading, worker)
	results := make(chan parallelResult, worker)

	var wg sync.WaitGroup
	wg.Add(worker)
	for i := 0; i < worker; i++ {
		go func() {
			defer wg.Done()
			for rr := range jobs {
				reading, err := One(rr)
				if err != nil {
					log.Printf("skipping %s:%d: %v", rr.Source, rr.Line, err)
					results <- parallelResult{err: err}
					continue
				}
				results <- parallelResult{reading: reading}
			}
		}()
	}

	go func() {
		for _, rr := range raws {
			jobs <- rr
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	skipped := 0
	for result := range results {
		if result.err != nil {
			skipped++
			continue
		}
		readings = append(readings, result.reading)
	}

	return readings, skipped
}
