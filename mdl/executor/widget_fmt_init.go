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
		// Production fallback (Phase 3): any widget type not explicitly registered
		// is handled by schema introspection. This is only reached for
		// CustomWidgets$CustomWidget documents whose widget ID has no dedicated
		// formatter — exactly the unknown-pluggable-widget case the generic
		// formatter is designed for. All built-in widget $Types are registered by
		// the init() functions in widget_fmt_*.go.
		d.fallback = GenericPluggableFactory
		defaultDispatcherInst = d
	})
	return defaultDispatcherInst
}
