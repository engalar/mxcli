// SPDX-License-Identifier: Apache-2.0

package pageast

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

func TestPropInt_AutoFill(t *testing.T) {
	w := &ast.WidgetV3{
		Properties: map[string]any{
			"TabletWidth": "AutoFill",
			"PhoneWidth":  "autofill", // lowercase variant
		},
	}
	if got := propInt(w, "tabletwidth"); got != -1 {
		t.Errorf("propInt AutoFill: want -1, got %d", got)
	}
	if got := propInt(w, "phonewidth"); got != -1 {
		t.Errorf("propInt autofill (lowercase): want -1, got %d", got)
	}
}

func TestPropInt_NormalInt(t *testing.T) {
	w := &ast.WidgetV3{
		Properties: map[string]any{"DesktopWidth": 12},
	}
	if got := propInt(w, "desktopwidth"); got != 12 {
		t.Errorf("propInt int: want 12, got %d", got)
	}
}
