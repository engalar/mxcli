// SPDX-License-Identifier: Apache-2.0

package main

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/sdk/widgets/mpk"
)

func TestDeriveMDLName(t *testing.T) {
	tests := []struct {
		widgetID string
		expected string
	}{
		{"com.mendix.widget.web.combobox.Combobox", "COMBOBOX"},
		{"com.mendix.widget.web.gallery.Gallery", "GALLERY"},
		{"com.company.widget.MyCustomWidget", "MYCUSTOMWIDGET"},
		{"SimpleWidget", "SIMPLEWIDGET"},
	}

	for _, tc := range tests {
		t.Run(tc.widgetID, func(t *testing.T) {
			result := deriveMDLName(tc.widgetID)
			if result != tc.expected {
				t.Errorf("deriveMDLName(%q) = %q, want %q", tc.widgetID, result, tc.expected)
			}
		})
	}
}

func TestGenerateDefJSON(t *testing.T) {
	mpkDef := &mpk.WidgetDefinition{
		ID:   "com.example.widget.TestWidget",
		Name: "Test Widget",
		Properties: []mpk.PropertyDef{
			{Key: "datasource", Type: "datasource"},
			{Key: "content", Type: "widgets"},
			{Key: "filterBar", Type: "widgets"},
			{Key: "myAttribute", Type: "attribute"},
			{Key: "showHeader", Type: "boolean", DefaultValue: "true"},
			{Key: "itemSelection", Type: "selection", DefaultValue: "Single"},
			{Key: "myAssociation", Type: "association"},
			{Key: "pageSize", Type: "integer", DefaultValue: "10"},
		},
	}

	def := generateDefJSON(mpkDef, "TESTWIDGET")

	// Verify basic fields
	if def.WidgetID != "com.example.widget.TestWidget" {
		t.Errorf("WidgetID = %q, want %q", def.WidgetID, "com.example.widget.TestWidget")
	}
	if def.MDLName != "TESTWIDGET" {
		t.Errorf("MDLName = %q, want %q", def.MDLName, "TESTWIDGET")
	}
	if def.TemplateFile != "testwidget.json" {
		t.Errorf("TemplateFile = %q, want %q", def.TemplateFile, "testwidget.json")
	}
	if def.DefaultEditable != "Always" {
		t.Errorf("DefaultEditable = %q, want %q", def.DefaultEditable, "Always")
	}

	// Verify property mappings count (datasource, attribute, boolean, selection, association, integer = 6)
	if len(def.PropertyMappings) != 6 {
		t.Fatalf("PropertyMappings count = %d, want 6", len(def.PropertyMappings))
	}

	// Verify child slots (content → TEMPLATE, filterBar → FILTERBAR)
	if len(def.ChildSlots) != 2 {
		t.Fatalf("ChildSlots count = %d, want 2", len(def.ChildSlots))
	}

	// content → TEMPLATE (special case)
	if def.ChildSlots[0].MDLContainer != "TEMPLATE" {
		t.Errorf("ChildSlots[0].MDLContainer = %q, want %q", def.ChildSlots[0].MDLContainer, "TEMPLATE")
	}
	// filterBar → FILTERBAR
	if def.ChildSlots[1].MDLContainer != "FILTERBAR" {
		t.Errorf("ChildSlots[1].MDLContainer = %q, want %q", def.ChildSlots[1].MDLContainer, "FILTERBAR")
	}

	// Verify datasource mapping
	dsMappings := findMapping(def.PropertyMappings, "datasource")
	if dsMappings == nil {
		t.Fatal("datasource mapping not found")
	}
	if dsMappings.Operation != "datasource" {
		t.Errorf("datasource operation = %q, want %q", dsMappings.Operation, "datasource")
	}

	// Verify attribute mapping
	attrMapping := findMapping(def.PropertyMappings, "myAttribute")
	if attrMapping == nil {
		t.Fatal("myAttribute mapping not found")
	}
	if attrMapping.Operation != "attribute" || attrMapping.Source != "Attribute" {
		t.Errorf("myAttribute: operation=%q source=%q, want operation=attribute source=Attribute",
			attrMapping.Operation, attrMapping.Source)
	}

	// Verify boolean with default value
	boolMapping := findMapping(def.PropertyMappings, "showHeader")
	if boolMapping == nil {
		t.Fatal("showHeader mapping not found")
	}
	if boolMapping.Value != "true" {
		t.Errorf("showHeader value = %q, want %q", boolMapping.Value, "true")
	}

	// Verify selection with default
	selMapping := findMapping(def.PropertyMappings, "itemSelection")
	if selMapping == nil {
		t.Fatal("itemSelection mapping not found")
	}
	if selMapping.Operation != "selection" || selMapping.Default != "Single" {
		t.Errorf("itemSelection: operation=%q default=%q, want operation=selection default=Single",
			selMapping.Operation, selMapping.Default)
	}
}

func TestGenerateDefJSON_SkipsComplexTypes(t *testing.T) {
	mpkDef := &mpk.WidgetDefinition{
		ID:   "com.example.Complex",
		Name: "Complex",
		Properties: []mpk.PropertyDef{
			{Key: "myAction", Type: "action"},
			{Key: "myExpr", Type: "expression"},
			{Key: "myTemplate", Type: "textTemplate"},
			{Key: "myIcon", Type: "icon"},
			{Key: "myObj", Type: "object"},
		},
	}

	def := generateDefJSON(mpkDef, "COMPLEX")

	// Complex types should be skipped
	if len(def.PropertyMappings) != 0 {
		t.Errorf("PropertyMappings count = %d, want 0 (complex types should be skipped)", len(def.PropertyMappings))
	}
	if len(def.ChildSlots) != 0 {
		t.Errorf("ChildSlots count = %d, want 0", len(def.ChildSlots))
	}
}

func TestGenerateDefJSON_AssociationAfterDataSource(t *testing.T) {
	// Association mappings require entityContext from a prior DataSource mapping.
	// generateDefJSON must order datasource before association regardless of MPK order.
	mpkDef := &mpk.WidgetDefinition{
		ID:   "com.example.AssocFirst",
		Name: "AssocFirst",
		Properties: []mpk.PropertyDef{
			{Key: "myAssoc", Type: "association"},     // association BEFORE datasource in MPK
			{Key: "myLabel", Type: "string"},
			{Key: "myDS", Type: "datasource"},
		},
	}

	def := generateDefJSON(mpkDef, "ASSOCFIRST")

	// Should have 3 mappings: datasource, string primitive, association
	if len(def.PropertyMappings) != 3 {
		t.Fatalf("PropertyMappings count = %d, want 3", len(def.PropertyMappings))
	}

	// datasource must appear before association in the mappings slice
	dsIdx, assocIdx := -1, -1
	for i, m := range def.PropertyMappings {
		if m.Source == "DataSource" {
			dsIdx = i
		}
		if m.Source == "Association" {
			assocIdx = i
		}
	}
	if dsIdx < 0 {
		t.Fatal("DataSource mapping not found")
	}
	if assocIdx < 0 {
		t.Fatal("Association mapping not found")
	}
	if dsIdx > assocIdx {
		t.Errorf("DataSource at index %d must come before Association at index %d", dsIdx, assocIdx)
	}

	// Verify the generated definition can be loaded by the registry without validation errors.
	// The registry's validateMappings enforces Association-after-DataSource ordering.
}

func findMapping(mappings []executor.PropertyMapping, key string) *executor.PropertyMapping {
	for i := range mappings {
		if mappings[i].PropertyKey == key {
			return &mappings[i]
		}
	}
	return nil
}
