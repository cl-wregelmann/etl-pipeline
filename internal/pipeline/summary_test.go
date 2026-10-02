package pipeline

import (
	"slices"
	"testing"

	"github.com/cl-wregelmann/etl-pipeline/internal/transform"
)

func TestSortedReasons(t *testing.T) {
	cases := map[string]struct {
		reasons map[transform.Reason]int
		want    []ReasonCount
	}{
		"count descending": {
			map[transform.Reason]int{"bad_value": 1, "out_of_range": 3, "bad_timestamp": 2},
			[]ReasonCount{{"out_of_range", 3}, {"bad_timestamp", 2}, {"bad_value", 1}},
		},
		"ties alphabetical": {
			map[transform.Reason]int{"unknown_metric": 1, "bad_value": 1, "missing_sensor_id": 1},
			[]ReasonCount{{"bad_value", 1}, {"missing_sensor_id", 1}, {"unknown_metric", 1}},
		},
		"empty": {map[transform.Reason]int{}, []ReasonCount{}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := Result{SkipReasons: tc.reasons}.SortedReasons()
			if got == nil || !slices.Equal(got, tc.want) {
				t.Errorf("SortedReasons = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestOneLine(t *testing.T) {
	const db, rep = "data/etl.db", "data/run-report.json"
	cases := map[string]struct {
		res  Result
		want string
	}{
		"sample data": {
			Result{Read: 27, Skipped: 6, Loaded: 21, SkipReasons: map[transform.Reason]int{
				"out_of_range": 2, "bad_timestamp": 1, "bad_value": 1,
				"missing_sensor_id": 1, "unknown_metric": 1,
			}},
			"done: read 27, skipped 6, loaded 21 -> data/etl.db (skipped: out_of_range=2 bad_timestamp=1 bad_value=1 missing_sensor_id=1 unknown_metric=1) report: data/run-report.json",
		},
		"no skips": {
			Result{Read: 21, Loaded: 21, SkipReasons: map[transform.Reason]int{}},
			"done: read 21, skipped 0, loaded 21 -> data/etl.db report: data/run-report.json",
		},
		"all skipped with other": {
			Result{Read: 4, Skipped: 4, SkipReasons: map[transform.Reason]int{"unknown_unit": 3, ReasonOther: 1}},
			"done: read 4, skipped 4, loaded 0 -> data/etl.db (skipped: unknown_unit=3 other=1) report: data/run-report.json",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := tc.res.OneLine(db, rep); got != tc.want {
				t.Errorf("OneLine =\n%s\nwant\n%s", got, tc.want)
			}
		})
	}
}
