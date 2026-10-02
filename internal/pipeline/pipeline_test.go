package pipeline

import (
	"errors"
	"fmt"
	"io"
	"log"
	"maps"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cl-wregelmann/etl-pipeline/internal/transform"
)

func TestReasonOf(t *testing.T) {
	ve := &transform.ValidationError{Reason: transform.ReasonBadValue, Msg: "invalid value \"\""}
	cases := map[string]struct {
		err  error
		want transform.Reason
	}{
		"validation error": {ve, transform.ReasonBadValue},
		"wrapped":          {fmt.Errorf("ctx: %w", ve), transform.ReasonBadValue},
		"plain error":      {errors.New("boom"), ReasonOther},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := reasonOf(tc.err); got != tc.want {
				t.Errorf("reasonOf = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRun_CountsSkipReasons(t *testing.T) {
	src := t.TempDir()
	csv := "sensor_id,timestamp,metric,value,unit\n" +
		"s1,2026-01-01T00:00:00Z,temperature,21.5,C\n" +
		"s1,2026-01-01T00:00:00Z,humidity,45,%\n" +
		"s2,2026-01-01T00:00:00Z,temperature,999,C\n" +
		"s2,2026-01-01T00:00:00Z,humidity,150,%\n" +
		"s3,not-a-timestamp,temperature,20,C\n" +
		"s4,2026-01-01T00:00:00Z,dewpoint,12,C\n"
	if err := os.WriteFile(filepath.Join(src, "readings.csv"), []byte(csv), 0o644); err != nil {
		t.Fatal(err)
	}
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	res, err := Run(Options{SourceDir: src, DBPath: filepath.Join(t.TempDir(), "test.db")})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if res.Read != 6 || res.Skipped != 4 || res.Loaded != 2 {
		t.Errorf("read/skipped/loaded = %d/%d/%d, want 6/4/2", res.Read, res.Skipped, res.Loaded)
	}
	wantReasons := map[transform.Reason]int{
		transform.ReasonOutOfRange:    2,
		transform.ReasonBadTimestamp:  1,
		transform.ReasonUnknownMetric: 1,
	}
	if !maps.Equal(res.SkipReasons, wantReasons) {
		t.Errorf("SkipReasons = %v, want %v", res.SkipReasons, wantReasons)
	}
	if res.Read != res.Skipped+res.Loaded {
		t.Errorf("Read %d != Skipped %d + Loaded %d", res.Read, res.Skipped, res.Loaded)
	}
	sum := 0
	for _, n := range res.SkipReasons {
		sum += n
	}
	if sum != res.Skipped {
		t.Errorf("sum(SkipReasons) = %d, want Skipped %d", sum, res.Skipped)
	}
	if res.StartedAt.IsZero() || res.StartedAt.Location() != time.UTC {
		t.Errorf("StartedAt = %v, want non-zero UTC", res.StartedAt)
	}
	if res.FinishedAt.Before(res.StartedAt) {
		t.Errorf("FinishedAt %v before StartedAt %v", res.FinishedAt, res.StartedAt)
	}
}
