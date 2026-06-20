package executor

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"
)

// PerfTimer collects timing statistics for named operations.
// Thread-safe for concurrent use.
type PerfTimer struct {
	mu     sync.Mutex
	start  time.Time
	ops    map[string]*perfOp
	output io.Writer
}

type perfOp struct {
	Count   int
	TotalNs int64
	MinNs   int64
	MaxNs   int64
}

// NewPerfTimer creates a new perf timer.
func NewPerfTimer() *PerfTimer {
	return &PerfTimer{
		start: time.Now(),
		ops:   make(map[string]*perfOp),
	}
}

// SetOutput sets where the report is written.
func (pt *PerfTimer) SetOutput(w io.Writer) {
	pt.mu.Lock()
	pt.output = w
	pt.mu.Unlock()
}

// Begin starts timing an operation, returns a completion function.
// Usage: defer perf.Begin("query.pages")()
func (pt *PerfTimer) Begin(name string) func() {
	start := time.Now()
	return func() {
		elapsed := time.Since(start).Nanoseconds()
		pt.mu.Lock()
		op, ok := pt.ops[name]
		if !ok {
			op = &perfOp{MinNs: elapsed, MaxNs: elapsed}
			pt.ops[name] = op
		}
		op.Count++
		op.TotalNs += elapsed
		if elapsed < op.MinNs {
			op.MinNs = elapsed
		}
		if elapsed > op.MaxNs {
			op.MaxNs = elapsed
		}
		pt.mu.Unlock()
	}
}

// Report writes the timing report.
func (pt *PerfTimer) Report() {
	pt.mu.Lock()
	total := time.Since(pt.start)
	output := pt.output
	names := make([]string, 0, len(pt.ops))
	for name := range pt.ops {
		names = append(names, name)
	}
	pt.mu.Unlock()
	sort.Strings(names)

	fmt.Fprintf(output, "\n── Performance ──────────────────────────────\n")
	fmt.Fprintf(output, "  Total wall time: %s\n", perfFormatDuration(total))

	var grandTotalNs int64
	for _, name := range names {
		pt.mu.Lock()
		op := pt.ops[name]
		pt.mu.Unlock()
		grandTotalNs += op.TotalNs
		avg := op.TotalNs / int64(op.Count)
		pct := float64(op.TotalNs*100) / float64(total.Nanoseconds())
		fmt.Fprintf(output, "  %s:\n", name)
		fmt.Fprintf(output, "    calls=%d  total=%s  avg=%s  min=%s  max=%s  (%.1f%%)\n",
			op.Count,
			perfFormatDuration(time.Duration(op.TotalNs)),
			perfFormatDuration(time.Duration(avg)),
			perfFormatDuration(time.Duration(op.MinNs)),
			perfFormatDuration(time.Duration(op.MaxNs)),
			pct,
		)
	}
	fmt.Fprintf(output, "────────────────────────────────────────────\n")
}

func perfFormatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.1fµs", float64(d.Nanoseconds())/1000)
	}
	if d < time.Second {
		return fmt.Sprintf("%.1fms", float64(d.Nanoseconds())/1e6)
	}
	return fmt.Sprintf("%.3fs", d.Seconds())
}

// graphLabel returns a short label for graph query type analysis.
func graphLabel(category, detail string) string {
	return strings.TrimSpace(category + "." + detail)
}
