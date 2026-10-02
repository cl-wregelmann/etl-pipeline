package pipeline

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/cl-wregelmann/etl-pipeline/internal/transform"
)

// ReasonCount is the number of records skipped for one reason.
type ReasonCount struct {
	Reason transform.Reason
	Count  int
}

// SortedReasons returns the skip reasons ordered by count descending, then by
// reason name ascending. Reasons with a zero count are left out, matching the
// JSON report. It never returns nil.
func (r Result) SortedReasons() []ReasonCount {
	out := make([]ReasonCount, 0, len(r.SkipReasons))
	for reason, n := range r.SkipReasons {
		if n > 0 {
			out = append(out, ReasonCount{Reason: reason, Count: n})
		}
	}
	slices.SortFunc(out, func(a, b ReasonCount) int {
		return cmp.Or(cmp.Compare(b.Count, a.Count), cmp.Compare(a.Reason, b.Reason))
	})
	return out
}

// OneLine formats the single-line run summary printed to stdout. The
// "(skipped: …)" group is omitted when nothing was skipped.
func (r Result) OneLine(dbPath, reportPath string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "done: read %d, skipped %d, loaded %d -> %s", r.Read, r.Skipped, r.Loaded, dbPath)
	if r.Skipped > 0 {
		b.WriteString(" (skipped:")
		for _, rc := range r.SortedReasons() {
			fmt.Fprintf(&b, " %s=%d", rc.Reason, rc.Count)
		}
		b.WriteString(")")
	}
	fmt.Fprintf(&b, " report: %s", reportPath)
	return b.String()
}
