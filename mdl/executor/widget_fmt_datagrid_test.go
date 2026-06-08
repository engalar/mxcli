// mdl/executor/widget_fmt_datagrid_test.go
package executor

import "testing"

// TestBuiltinDataGridFormatter_DelegatesToLegacy verifies the built-in
// Forms$DataGrid formatter reproduces the legacy outputWidgetMDLV3 result during
// the Phase 1/2 bridge. The legacy pipeline has no Forms$DataGrid case, so the
// output is the default "-- Forms$DataGrid (name)" comment line.
func TestBuiltinDataGridFormatter_DelegatesToLegacy(t *testing.T) {
	ctx, buf := newTestFormatCtx(t)
	raw := map[string]any{"$Type": "Forms$DataGrid", "Name": "dgItems"}
	builtinDataGridFactory(raw).FormatMDL(ctx)
	assertOutput(t, buf, "-- Forms$DataGrid (dgItems)")
}

// TestBuiltinDataGridFormatter_Registered verifies the dispatcher routes
// Forms$DataGrid to the built-in DataGrid formatter rather than the fallback.
func TestBuiltinDataGridFormatter_Registered(t *testing.T) {
	d := newDefaultDispatcher()
	for _, ty := range []string{"Forms$DataGrid", "Pages$DataGrid"} {
		d.registerBSONType(ty, FactoryEntry{Factory: builtinDataGridFactory})
		if _, ok := d.entries[ty]; !ok {
			t.Errorf("%s not registered in dispatcher", ty)
		}
	}
}
