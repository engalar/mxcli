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
