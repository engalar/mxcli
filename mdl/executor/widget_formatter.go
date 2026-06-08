// mdl/executor/widget_formatter.go
package executor

import (
	"fmt"
	"io"
	"strings"
)

// ─── Narrow interfaces (I principle) ─────────────────────────────────────────

// NameResolver resolves entity and module names from container IDs.
// ExecContext implements this interface.
type NameResolver interface {
	resolveEntityContext(containerID string) string
	resolveModuleName(containerID string) string
}

// WidgetDispatcher dispatches widget formatting by raw BSON type.
type WidgetDispatcher interface {
	Format(ctx *FormatContext, raw map[string]any)
}

// ─── WidgetFormatter ─────────────────────────────────────────────────────────

// WidgetFormatter formats a single widget node as MDL.
type WidgetFormatter interface {
	FormatMDL(ctx *FormatContext)
}

// FormatterFactory creates a WidgetFormatter from raw widget BSON.
type FormatterFactory func(raw map[string]any) WidgetFormatter

// ─── FormatContext ────────────────────────────────────────────────────────────

// FormatContext carries shared state for one describe pass.
// It depends only on interfaces, never on concrete types (D principle).
type FormatContext struct {
	Output     io.Writer
	Indent     int
	Dispatcher WidgetDispatcher
}

// Child returns a FormatContext with Indent incremented by one.
func (ctx *FormatContext) Child() *FormatContext {
	return &FormatContext{
		Output:     ctx.Output,
		Indent:     ctx.Indent + 1,
		Dispatcher: ctx.Dispatcher,
	}
}

// withIndent returns a FormatContext with Indent set to n (absolute), sharing
// the same writer and dispatcher. Used by formatters that nest children at a
// fixed depth (e.g. LayoutGrid columns) rather than a single +1 step.
func (ctx *FormatContext) withIndent(n int) *FormatContext {
	return &FormatContext{Output: ctx.Output, Indent: n, Dispatcher: ctx.Dispatcher}
}

// Write writes a single line at the current indent level.
func (ctx *FormatContext) Write(s string) {
	fmt.Fprintf(ctx.Output, "%s%s\n", strings.Repeat("  ", ctx.Indent), s)
}

// WriteRaw writes s without any indent prefix (for multi-line props blocks).
func (ctx *FormatContext) WriteRaw(s string) {
	fmt.Fprint(ctx.Output, s)
}

// ─── FactoryEntry ────────────────────────────────────────────────────────────

// FactoryEntry is a registration record. SubKeyExtractor, when non-nil,
// extracts a secondary dispatch key (e.g. pluggable widget ID) from the raw map.
// This keeps Mendix widget-category knowledge out of the dispatcher (O principle).
type FactoryEntry struct {
	Factory         FormatterFactory
	SubKeyExtractor func(raw map[string]any) string
}

// ─── FormatterDispatcher ─────────────────────────────────────────────────────

// FormatterDispatcher maps BSON $Type strings (and sub-keys for pluggable widgets)
// to FormatterFactory functions. Dispatch is purely data-driven: no hard-coded
// special cases for any widget category (O principle).
type FormatterDispatcher struct {
	entries  map[string]FactoryEntry
	fallback FormatterFactory
}

func newDefaultDispatcher() *FormatterDispatcher {
	d := &FormatterDispatcher{
		entries: make(map[string]FactoryEntry),
	}
	d.fallback = func(raw map[string]any) WidgetFormatter {
		return &unknownWidgetFormatter{raw: raw}
	}
	return d
}

func (d *FormatterDispatcher) registerBSONType(bsonType string, entry FactoryEntry) {
	d.entries[bsonType] = entry
}

// Format dispatches to the appropriate formatter. If no entry exists, the
// fallback is used. For entries with a SubKeyExtractor, a second lookup is
// performed on the extracted key; if that also misses, fallback is used.
func (d *FormatterDispatcher) Format(ctx *FormatContext, raw map[string]any) {
	bsonType, _ := raw["$Type"].(string)
	entry, ok := d.entries[bsonType]
	if !ok {
		d.fallback(raw).FormatMDL(ctx)
		return
	}
	if entry.SubKeyExtractor != nil {
		subKey := entry.SubKeyExtractor(raw)
		if sub, ok := d.entries[subKey]; ok {
			sub.Factory(raw).FormatMDL(ctx)
			return
		}
		// Unknown sub-key → fallback (GenericPluggableFormatter once installed)
		d.fallback(raw).FormatMDL(ctx)
		return
	}
	entry.Factory(raw).FormatMDL(ctx)
}

// unknownWidgetFormatter is the default fallback for unregistered widget types.
type unknownWidgetFormatter struct{ raw map[string]any }

func (f *unknownWidgetFormatter) FormatMDL(ctx *FormatContext) {
	bsonType, _ := f.raw["$Type"].(string)
	name := safeStr(f.raw, "Name")
	ctx.Write(fmt.Sprintf("-- widget %s %s", bsonType, name))
}

// ─── Shared helpers ───────────────────────────────────────────────────────────

// safeStr safely extracts a string from a raw BSON map. Returns "" on miss.
func safeStr(raw map[string]any, key string) string {
	v, _ := raw[key].(string)
	return v
}

// indentStr returns n repetitions of two spaces.
func indentStr(n int) string { return strings.Repeat("  ", n) }
