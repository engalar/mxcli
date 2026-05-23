// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/model"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	genTexts "github.com/mendixlabs/mxcli/modelsdk/gen/texts"
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

// TestBuildDynamicTextV3_EmptyContentUsesSafeDefault verifies CE0720 fix:
// when a DYNAMICTEXT widget has an empty content string, buildDynamicTextV3
// must NOT produce a template with text "{1}" and zero parameters — that
// combination causes Studio Pro to raise CE0720. A single space " " is the
// safe fallback (documented in CLAUDE.md).
func TestBuildDynamicTextV3_EmptyContentUsesSafeDefault(t *testing.T) {
	pb := &pageBuilder{
		widgetScope:      map[string]model.ID{},
		paramEntityNames: map[string]string{},
	}
	w := &ast.WidgetV3{
		Type:       "dynamictext",
		Name:       "txt1",
		Properties: map[string]interface{}{
			// no Content property → GetContent() returns ""
		},
	}
	result, err := pb.buildDynamicTextV3(w)
	if err != nil {
		t.Fatalf("buildDynamicTextV3 failed: %v", err)
	}
	dt, ok := result.(*genPg.DynamicText)
	if !ok {
		t.Fatalf("result = %T, want *DynamicText", result)
	}
	ct, ok := dt.Content().(*genPg.ClientTemplate)
	if !ok {
		t.Fatalf("Content = %T, want *ClientTemplate", dt.Content())
	}
	// CE0720 fix: template text must NOT be "{1}" when there are no parameters.
	if tmpl, ok := ct.Template().(*genTexts.Text); ok {
		for _, item := range tmpl.TranslationsItems() {
			if tr, ok := item.(*genTexts.Translation); ok {
				if tr.Text() == "{1}" && len(ct.ParametersItems()) == 0 {
					t.Fatalf("CE0720: template text is {1} but there are 0 parameters — Studio Pro will raise CE0720")
				}
			}
		}
	}
	// Parameters list must be empty (no auto-generated params for empty content).
	if len(ct.ParametersItems()) != 0 {
		t.Fatalf("expected 0 parameters for empty content, got %d", len(ct.ParametersItems()))
	}
}

// newPageBuilderWithMicroflowStub returns a pageBuilder primed so that
// resolveMicroflow finds qualifiedName via execCache.createdMicroflows,
// avoiding any backend round-trip.
func newPageBuilderWithMicroflowStub(qualifiedName string) *pageBuilder {
	return &pageBuilder{
		execCache: &executorCache{
			createdMicroflows: map[string]*createdMicroflowInfo{
				qualifiedName: {ID: model.ID("00000000-0000-0000-0000-000000000001"), Name: "SomeMF", ModuleName: "MyMod"},
			},
		},
	}
}

// TestBuildClientActionBSON_MicroflowClosePage verifies that an action with
// Type="microflow" and ClosePage=true emits ClosePage:true in BSON.
// Regression guard for the `action: microflow M.F (...) close_page` syntax.
func TestBuildClientActionBSON_MicroflowClosePage(t *testing.T) {
	pb := newPageBuilderWithMicroflowStub("MyMod.SomeMF")
	action := &ast.ActionV3{
		Type:      "microflow",
		Target:    "MyMod.SomeMF",
		ClosePage: true,
	}
	got, err := pb.buildClientActionBSON(action)
	if err != nil {
		t.Fatalf("buildClientActionBSON: %v", err)
	}
	for _, kv := range got {
		if kv.Key == "ClosePage" {
			if v, ok := kv.Value.(bool); ok && v {
				return // pass
			}
			t.Errorf("ClosePage = %v, want true", kv.Value)
			return
		}
	}
	t.Error("ClosePage key not found in BSON for microflow action with ClosePage=true")
}

// TestBuildClientActionBSON_MicroflowNoClosePage verifies that a microflow
// action without close_page emits ClosePage:false (not missing).
func TestBuildClientActionBSON_MicroflowNoClosePage(t *testing.T) {
	pb := newPageBuilderWithMicroflowStub("MyMod.SomeMF")
	action := &ast.ActionV3{
		Type:      "microflow",
		Target:    "MyMod.SomeMF",
		ClosePage: false,
	}
	got, err := pb.buildClientActionBSON(action)
	if err != nil {
		t.Fatalf("buildClientActionBSON: %v", err)
	}
	for _, kv := range got {
		if kv.Key == "ClosePage" {
			if v, ok := kv.Value.(bool); ok && !v {
				return
			}
			t.Errorf("ClosePage = %v, want false", kv.Value)
			return
		}
	}
	t.Error("ClosePage key not found in BSON for microflow action without close_page")
}
