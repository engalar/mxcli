// mdl/executor/widget_fmt_init.go
package executor

import "sync"

var (
	defaultDispatcherOnce sync.Once
	defaultDispatcherInst *FormatterDispatcher
)

// DefaultDispatcher returns the process-level formatter dispatcher.
// All widget formatters register themselves here via init() in their respective files.
func DefaultDispatcher() *FormatterDispatcher {
	defaultDispatcherOnce.Do(func() {
		d := newDefaultDispatcher()
		// Phase 1: only the unknown-widget fallback is installed here. Specific
		// registrations are added by init() calls in widget_fmt_*.go files.
		// The legacy bridge fallback (legacyWidgetFallback) is set by
		// describePage before first use and removed in Phase 3.
		defaultDispatcherInst = d
	})
	return defaultDispatcherInst
}
