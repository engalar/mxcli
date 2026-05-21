// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// buildTestWidgetV3WithDesignProp creates a WidgetV3 with a single design property.
func buildTestWidgetV3WithDesignProp(key, value string) *ast.WidgetV3 {
	w := &ast.WidgetV3{Properties: map[string]interface{}{}}
	w.Properties["DesignProperties"] = []ast.DesignPropertyEntryV3{{Key: key, Value: value}}
	return w
}

// TestApplyWidgetAppearanceGenDesignPropertiesWrappedInDesignPropertyValue
// verifies that design properties are serialised as:
//
//	Appearance.DesignProperties[] → DesignPropertyValue → Value: OptionDesignPropertyValue
//
// NOT as:
//
//	Appearance.DesignProperties[] → OptionDesignPropertyValue  (missing wrapper)
//
// The missing wrapper causes Studio Pro to crash with:
//
//	"OptionDesignPropertyValue does not contain a constructor with a
//	 parameter of type Appearance"
func TestApplyWidgetAppearanceGenDesignPropertiesWrappedInDesignPropertyValue(t *testing.T) {
	w := buildTestWidgetV3WithDesignProp("spacing", "large")
	widget := genPg.NewDivContainer()

	applyWidgetAppearanceGen(widget, w, nil)

	app, ok := widget.Appearance().(*genPg.Appearance)
	if !ok || app == nil {
		t.Fatal("Appearance must be set on widget after applyWidgetAppearanceGen")
	}

	items := app.DesignPropertiesItems()
	if len(items) != 1 {
		t.Fatalf("expected 1 DesignProperties item, got %d", len(items))
	}

	// The item MUST be a DesignPropertyValue wrapper, not OptionDesignPropertyValue directly.
	dpv, ok := items[0].(*genPg.DesignPropertyValue)
	if !ok {
		t.Fatalf("DesignProperties[0] = %T, want *DesignPropertyValue — "+
			"OptionDesignPropertyValue must be wrapped in DesignPropertyValue, "+
			"otherwise Studio Pro crashes with: 'OptionDesignPropertyValue does not "+
			"contain a constructor with a parameter of type Appearance'", items[0])
	}

	if dpv.Key() != "spacing" {
		t.Errorf("DesignPropertyValue.Key = %q, want %q", dpv.Key(), "spacing")
	}

	val, ok := dpv.Value().(*genPg.OptionDesignPropertyValue)
	if !ok {
		t.Fatalf("DesignPropertyValue.Value = %T, want *OptionDesignPropertyValue", dpv.Value())
	}
	if val.Option() != "large" {
		t.Errorf("OptionDesignPropertyValue.Option = %q, want %q", val.Option(), "large")
	}
}

// TestApplyWidgetAppearanceGenToggleOnWrappedInDesignPropertyValue verifies that
// toggle-on design properties (value="on") are also wrapped in DesignPropertyValue.
func TestApplyWidgetAppearanceGenToggleOnWrappedInDesignPropertyValue(t *testing.T) {
	w := buildTestWidgetV3WithDesignProp("bold", "on")
	widget := genPg.NewDivContainer()
	applyWidgetAppearanceGen(widget, w, nil)

	app, ok := widget.Appearance().(*genPg.Appearance)
	if !ok || app == nil {
		t.Fatal("Appearance must be set")
	}
	items := app.DesignPropertiesItems()
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	dpv, ok := items[0].(*genPg.DesignPropertyValue)
	if !ok {
		t.Fatalf("DesignProperties[0] = %T, want *DesignPropertyValue (toggle-on must also be wrapped)", items[0])
	}
	if dpv.Key() != "bold" {
		t.Errorf("Key = %q, want bold", dpv.Key())
	}
	if _, ok := dpv.Value().(*genPg.ToggleDesignPropertyValue); !ok {
		t.Fatalf("DesignPropertyValue.Value = %T, want *ToggleDesignPropertyValue", dpv.Value())
	}
}
