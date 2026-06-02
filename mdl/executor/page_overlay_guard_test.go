// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/types"
)

// TestPageOverlayGateNotAlwaysTrue verifies that pageModelHasLossyWidget no
// longer returns an unconditional true (the old stub). The write-path overlay
// gate must actually inspect the widget tree: a tree of only safe widgets must
// be reported as non-lossy so the IR overlay can fire, while a tree containing
// a lossy pluggable widget must still be gated to the gen builder.
func TestPageOverlayGateNotAlwaysTrue(t *testing.T) {
	t.Run("nil model is not lossy", func(t *testing.T) {
		if pageModelHasLossyWidget(nil) {
			t.Fatal("nil PageModel must not be reported as lossy")
		}
	})

	t.Run("safe widget tree is not lossy", func(t *testing.T) {
		pm := &types.PageModel{
			Widgets: []*types.WidgetNode{
				{
					Kind: types.WidgetContainer,
					Children: []*types.WidgetNode{
						{Kind: types.WidgetText},
						{Kind: types.WidgetLabel},
					},
				},
			},
		}
		if pageModelHasLossyWidget(pm) {
			t.Fatal("a tree of safe widgets must not be reported as lossy (gate was always-true stub?)")
		}
	})

	t.Run("lossy widget tree is lossy", func(t *testing.T) {
		pm := &types.PageModel{
			Widgets: []*types.WidgetNode{
				{
					Kind: types.WidgetContainer,
					Children: []*types.WidgetNode{
						{Kind: types.WidgetDataGrid},
					},
				},
			},
		}
		if !pageModelHasLossyWidget(pm) {
			t.Fatal("a tree containing a DataGrid must be reported as lossy")
		}
	})
}

// TestPageLossyWidgets_DocumentedGap documents which widget kinds remain lossy
// (routed through the gen builder + legacy describe instead of the IR overlay)
// after the page-migration plan Tasks 1-6. Each entry is a known gap with a
// rationale. When a kind is genuinely fixed in the IR, remove it from this list
// AND from widgetTreeHasLossyKind together.
//
// Rationale per kind:
//   - DataGrid/Gallery/ComboBox/Image: pluggable CustomWidgets — need template
//     extraction + Object property trees; deferred to a dedicated widget spec.
//   - Unknown: unrecognised pluggable widgets are lossy by definition.
//   - DataView: overlay writes incomplete BSON — loses column width params,
//     footer, and action buttons. Legacy describe + gen builder are authoritative.
//     Fixing requires full Forms$DataViewSource + Forms$LayoutColumn + footer
//     serialization in widgetToBSON, which is a separate future spec.
//   - ListView: data source is a complex nested gen structure the IR cannot
//     reproduce yet.
//   - Button: action BSON is rich (microflow/nanoflow/page/create + parameter
//     mappings + then-chains) and the legacy describe renders it faithfully
//     (e.g. `microflow M.F(Product: $Product)`); the IR's bare OnClick string
//     would drop parameter mappings and non-microflow actions. Keeping it lossy
//     preserves TestRoundtripPage_MicroflowButton* output.
//   - CheckBox: renderWidget has no dedicated case; legacy path handles it
//     via outputWidgetMDLV3.
//   - RadioButtons: storage type is Forms$RadioButtonGroup (a reference
//     selector), not the slim AttributeRef input form; writing the IR shape
//     triggers TypeCacheUnknownTypeException in mx check.
//   - ScrollView: no MDL grammar path creates a scrollcontainer and the gen
//     builder has no ScrollContainer support, so the IR write path is never
//     exercised; the legacy describe path reads existing scroll containers.
func TestPageLossyWidgets_DocumentedGap(t *testing.T) {
	stillLossy := []types.WidgetKind{
		types.WidgetDataGrid,
		types.WidgetGallery,
		types.WidgetComboBox,
		types.WidgetImage,
		types.WidgetUnknown,
		types.WidgetDataView,
		types.WidgetListView,
		types.WidgetButton,
		types.WidgetCheckBox,
		types.WidgetRadioButtons,
		types.WidgetScrollView,
	}
	for _, kind := range stillLossy {
		n := &types.WidgetNode{Kind: kind}
		if !widgetTreeHasLossyKind(n) {
			t.Errorf("WidgetKind %q should still be lossy but widgetTreeHasLossyKind returned false — either the kind was fixed (remove it from this list AND widgetTreeHasLossyKind) or the gate was accidentally broken", kind)
		}
	}
}

// TestPageOverlayEnabledWidgets asserts the widget kinds that are safe for IR
// overlay (not lossy). These widget kinds pass through widgetTreeHasLossyKind
// as non-lossy; they can be written by widgetToBSON and read back correctly.
// When a lossy kind is fixed, move it here and remove from
// TestPageLossyWidgets_DocumentedGap.
func TestPageOverlayEnabledWidgets(t *testing.T) {
	enabled := []types.WidgetKind{
		types.WidgetContainer,
		types.WidgetLayoutGrid,
		types.WidgetLayoutRow,
		types.WidgetLayoutCol,
		types.WidgetGroupBox,
		types.WidgetTabContainer,
		types.WidgetTabPage,
		types.WidgetTextBox,
		types.WidgetTextArea,
		types.WidgetDatePicker,
		types.WidgetLabel,
		types.WidgetText,
		types.WidgetTitle,
		types.WidgetSnippet,
	}
	for _, kind := range enabled {
		n := &types.WidgetNode{Kind: kind}
		if widgetTreeHasLossyKind(n) {
			t.Errorf("WidgetKind %q should be overlay-enabled (not lossy) but widgetTreeHasLossyKind returned true", kind)
		}
	}
}
