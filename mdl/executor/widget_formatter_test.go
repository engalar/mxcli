// mdl/executor/widget_formatter_test.go
package executor

import (
	"bytes"
	"strings"
	"testing"
)

func TestFormatterDispatcher_DispatchesByBSONType(t *testing.T) {
	d := newDefaultDispatcher()
	var buf bytes.Buffer
	called := false
	d.registerBSONType("Test$Widget", FactoryEntry{
		Factory: func(raw map[string]any) WidgetFormatter {
			return &funcFormatter{fn: func(ctx *FormatContext) {
				called = true
				ctx.Write("test-widget")
			}}
		},
	})
	ctx := &FormatContext{Output: &buf, Indent: 0, Dispatcher: d}
	d.Format(ctx, map[string]any{"$Type": "Test$Widget", "Name": "w1"})
	if !called {
		t.Error("formatter was not called")
	}
	if !strings.Contains(buf.String(), "test-widget") {
		t.Errorf("unexpected output: %q", buf.String())
	}
}

func TestFormatterDispatcher_SubKeyDispatch(t *testing.T) {
	d := newDefaultDispatcher()
	var buf bytes.Buffer
	d.registerBSONType("Meta$Widget", FactoryEntry{
		Factory:         d.fallback, // default for unknown sub-key
		SubKeyExtractor: func(raw map[string]any) string { return safeStr(raw, "widgetID") },
	})
	d.registerBSONType("com.co.specific", FactoryEntry{
		Factory: func(raw map[string]any) WidgetFormatter {
			return &funcFormatter{fn: func(ctx *FormatContext) { ctx.Write("specific") }}
		},
	})
	ctx := &FormatContext{Output: &buf, Indent: 0, Dispatcher: d}
	d.Format(ctx, map[string]any{"$Type": "Meta$Widget", "widgetID": "com.co.specific"})
	if !strings.Contains(buf.String(), "specific") {
		t.Errorf("sub-key dispatch failed: %q", buf.String())
	}
}

func TestFormatterDispatcher_FallbackForUnknown(t *testing.T) {
	d := newDefaultDispatcher()
	var buf bytes.Buffer
	ctx := &FormatContext{Output: &buf, Indent: 0, Dispatcher: d}
	d.Format(ctx, map[string]any{"$Type": "Unknown$Widget", "Name": "w1"})
	if !strings.Contains(buf.String(), "-- widget") {
		t.Errorf("fallback should emit comment: %q", buf.String())
	}
}

func TestFormatContext_Child_IncrementsIndent(t *testing.T) {
	var buf bytes.Buffer
	d := newDefaultDispatcher()
	ctx := &FormatContext{Output: &buf, Indent: 2, Dispatcher: d}
	child := ctx.Child()
	if child.Indent != 3 {
		t.Errorf("Child().Indent = %d, want 3", child.Indent)
	}
}

// funcFormatter is a test helper implementing WidgetFormatter via a closure.
type funcFormatter struct{ fn func(*FormatContext) }

func (f *funcFormatter) FormatMDL(ctx *FormatContext) { f.fn(ctx) }
