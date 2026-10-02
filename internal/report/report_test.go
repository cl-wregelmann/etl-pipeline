package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/cl-wregelmann/etl-pipeline/internal/pipeline"
	"github.com/cl-wregelmann/etl-pipeline/internal/transform"
)

var started = time.Date(2026, 10, 2, 14, 30, 0, 0, time.UTC)

var opts = pipeline.Options{SourceDir: "data/raw", DBPath: "data/etl.db"}

func sampleResult() pipeline.Result {
	return pipeline.Result{
		Read: 27, Skipped: 6, Loaded: 21,
		SkipReasons: map[transform.Reason]int{
			"out_of_range": 2, "bad_timestamp": 1, "bad_value": 1,
			"missing_sensor_id": 1, "unknown_metric": 1,
		},
		StartedAt:  started,
		FinishedAt: started.Add(184 * time.Millisecond),
	}
}

func cleanResult() pipeline.Result {
	return pipeline.Result{
		Read: 21, Loaded: 21,
		SkipReasons: map[transform.Reason]int{},
		StartedAt:   started,
		FinishedAt:  started.Add(184 * time.Millisecond),
	}
}

// Golden outputs from docs/design/run-summary/example-output.md §1 and §2.
const goldenSample = `{
  "schema_version": 1,
  "started_at": "2026-10-02T14:30:00Z",
  "finished_at": "2026-10-02T14:30:00.184Z",
  "duration_ms": 184,
  "source_dir": "data/raw",
  "db_path": "data/etl.db",
  "counts": {
    "read": 27,
    "skipped": 6,
    "loaded": 21
  },
  "skip_reasons": {
    "bad_timestamp": 1,
    "bad_value": 1,
    "missing_sensor_id": 1,
    "out_of_range": 2,
    "unknown_metric": 1
  }
}
`

const goldenClean = `{
  "schema_version": 1,
  "started_at": "2026-10-02T14:30:00Z",
  "finished_at": "2026-10-02T14:30:00.184Z",
  "duration_ms": 184,
  "source_dir": "data/raw",
  "db_path": "data/etl.db",
  "counts": {
    "read": 21,
    "skipped": 0,
    "loaded": 21
  },
  "skip_reasons": {}
}
`

func TestNew(t *testing.T) {
	got := New(sampleResult(), opts)
	want := Report{
		SchemaVersion: 1,
		StartedAt:     started,
		FinishedAt:    started.Add(184 * time.Millisecond),
		DurationMS:    184,
		SourceDir:     "data/raw",
		DBPath:        "data/etl.db",
		Counts:        Counts{Read: 27, Skipped: 6, Loaded: 21},
		SkipReasons: map[string]int{
			"out_of_range": 2, "bad_timestamp": 1, "bad_value": 1,
			"missing_sensor_id": 1, "unknown_metric": 1,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("New =\n%+v\nwant\n%+v", got, want)
	}

	for name, reasons := range map[string]map[transform.Reason]int{"empty": {}, "nil": nil} {
		t.Run(name, func(t *testing.T) {
			res := cleanResult()
			res.SkipReasons = reasons
			if r := New(res, opts); r.SkipReasons == nil {
				t.Error("SkipReasons is nil; it must encode as {}")
			}
		})
	}
}

func TestWriteFile_Golden(t *testing.T) {
	cases := map[string]struct {
		res  pipeline.Result
		want string
	}{
		"sample data": {sampleResult(), goldenSample},
		"clean run":   {cleanResult(), goldenClean},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "nested", "dir", "report.json")
			if err := WriteFile(path, New(tc.res, opts)); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Errorf("file =\n%s\nwant\n%s", got, tc.want)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if mode := info.Mode().Perm(); mode != 0o644 {
				t.Errorf("mode = %o, want 644", mode)
			}
		})
	}
}

func TestWriteFile_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.json")
	want := New(sampleResult(), opts)
	if err := WriteFile(path, want); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got Report
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round trip =\n%+v\nwant\n%+v", got, want)
	}
}

func TestWriteFile_Overwrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")
	if err := WriteFile(path, New(sampleResult(), opts)); err != nil {
		t.Fatalf("first WriteFile: %v", err)
	}
	if err := WriteFile(path, New(cleanResult(), opts)); err != nil {
		t.Fatalf("second WriteFile: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != goldenClean {
		t.Errorf("file =\n%s\nwant second write\n%s", got, goldenClean)
	}
	assertNoTempFiles(t, dir)
}

func TestWriteFile_Error(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(filepath.Join(blocker, "report.json"), New(sampleResult(), opts)); err == nil {
		t.Fatal("WriteFile succeeded, want error when the parent is a regular file")
	}
	assertNoTempFiles(t, dir)
}

func TestWriteFile_RenameError(t *testing.T) {
	dir := t.TempDir()
	// A directory at the target path makes the final rename fail after the
	// temp file has been written, exercising the cleanup path.
	path := filepath.Join(dir, "report.json")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "keep"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(path, New(sampleResult(), opts)); err == nil {
		t.Fatal("WriteFile succeeded, want rename error")
	}
	assertNoTempFiles(t, dir)
}

func assertNoTempFiles(t *testing.T, dir string) {
	t.Helper()
	leftovers, err := filepath.Glob(filepath.Join(dir, ".run-report-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(leftovers) > 0 {
		t.Errorf("temp files left behind: %v", leftovers)
	}
}
