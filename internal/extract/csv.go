// Package extract reads raw sensor records from source files.
package extract

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cl-wregelmann/etl-pipeline/internal/model"
)

// expectedHeader is the column order the CSV reader expects.
var expectedHeader = []string{"sensor_id", "timestamp", "metric", "value", "unit"}

// FromDir reads every *.csv file in dir (non-recursively) and returns the raw
// records in a stable order (sorted by filename, then by line). Files are read
// fully into memory; the sample datasets are small.
func FromDir(dir string) ([]model.RawReading, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read source dir %q: %w", dir, err)
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Ext(e.Name()), ".csv") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)

	if len(files) == 0 {
		return nil, fmt.Errorf("no .csv files found in %q", dir)
	}

	var out []model.RawReading
	for _, f := range files {
		records, err := fromFile(f)
		if err != nil {
			return nil, err
		}
		out = append(out, records...)
	}
	return out, nil
}

func fromFile(path string) ([]model.RawReading, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = len(expectedHeader)
	r.TrimLeadingSpace = true

	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read header of %q: %w", path, err)
	}
	if !headerMatches(header) {
		return nil, fmt.Errorf("unexpected header in %q: got %v, want %v", path, header, expectedHeader)
	}

	name := filepath.Base(path)
	var out []model.RawReading
	line := 1 // header was line 1
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		line++
		if err != nil {
			return nil, fmt.Errorf("read %q line %d: %w", path, line, err)
		}
		out = append(out, model.RawReading{
			SensorID:  rec[0],
			Timestamp: rec[1],
			Metric:    rec[2],
			Value:     rec[3],
			Unit:      rec[4],
			Source:    name,
			Line:      line,
		})
	}
	return out, nil
}

func headerMatches(got []string) bool {
	if len(got) != len(expectedHeader) {
		return false
	}
	for i := range got {
		if !strings.EqualFold(strings.TrimSpace(got[i]), expectedHeader[i]) {
			return false
		}
	}
	return true
}
