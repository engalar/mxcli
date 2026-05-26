// SPDX-License-Identifier: Apache-2.0

// Tests for Issue #21: ComboBox association Attribute written as pointer; DESCRIBE shows CaptionAttribute.
//
// Root cause: extractCustomWidgetAttribute blindly scans all properties for the first AttributeRef.
// In association mode, attributeAssociation uses EntityRef (not AttributeRef), so the scan
// skips it and finds optionsSourceAssociationCaptionAttribute's AttributeRef ("Name") instead.
//
// Fix: extractCustomWidgetPropertyAssociation — generic property-key-aware association extractor,
// symmetric to the existing extractCustomWidgetPropertyAttributeRef.
package executor

import (
	"testing"
)

// buildComboBoxAssocWidget builds a mock CustomWidget map matching the BSON structure
// written by the pluggable widget engine for COMBOBOX in association mode.
// TypePointer IDs are plain strings (extractBinaryID accepts strings directly).
func buildComboBoxAssocWidget(assocPath, captionAttrPath string) map[string]any {
	const (
		idOptionsSourceType = "type-id-001"
		idAttrAssociation   = "type-id-002"
		idAssocDataSource   = "type-id-003"
		idCaptionAttribute  = "type-id-004"
	)

	widgetType := map[string]any{
		"WidgetId": "com.mendix.widget.web.combobox.Combobox",
		"ObjectType": map[string]any{
			"PropertyTypes": []any{
				map[string]any{"$ID": idOptionsSourceType, "PropertyKey": "optionsSourceType"},
				map[string]any{"$ID": idAttrAssociation, "PropertyKey": "attributeAssociation"},
				map[string]any{"$ID": idAssocDataSource, "PropertyKey": "optionsSourceAssociationDataSource"},
				map[string]any{"$ID": idCaptionAttribute, "PropertyKey": "optionsSourceAssociationCaptionAttribute"},
			},
		},
	}

	// Properties mirror what setAssociationRef and setAttributeRef produce in the engine.
	properties := []any{
		// optionsSourceType = "association"
		map[string]any{
			"TypePointer": idOptionsSourceType,
			"Value":       map[string]any{"PrimitiveValue": "association"},
		},
		// attributeAssociation — uses EntityRef (written by opAssociation / setAssociationRef)
		map[string]any{
			"TypePointer": idAttrAssociation,
			"Value": map[string]any{
				"EntityRef": map[string]any{
					"$Type": "DomainModels$IndirectEntityRef",
					"Steps": []any{
						int32(2), // version marker
						map[string]any{
							"$Type":             "DomainModels$EntityRefStep",
							"Association":       assocPath,
							"DestinationEntity": "MyFirstModule.Category",
						},
					},
				},
			},
		},
		// optionsSourceAssociationDataSource
		map[string]any{
			"TypePointer": idAssocDataSource,
			"Value": map[string]any{
				"DataSource": map[string]any{
					"$Type":     "CustomWidgets$CustomWidgetXPathSource",
					"EntityRef": map[string]any{"Entity": "MyFirstModule.Category"},
				},
			},
		},
		// optionsSourceAssociationCaptionAttribute — uses AttributeRef (written by opAttribute)
		map[string]any{
			"TypePointer": idCaptionAttribute,
			"Value": map[string]any{
				"AttributeRef": map[string]any{
					"$Type":     "DomainModels$AttributeRef",
					"Attribute": captionAttrPath,
				},
			},
		},
	}

	return map[string]any{
		"Type":   widgetType,
		"Object": map[string]any{"Properties": properties},
	}
}

// TestExtractCustomWidgetPropertyAssociation_ReturnsAssociationName verifies that
// the generic association extractor returns the short association name from the
// named property's EntityRef.Steps[1].Association.
// Regression test for Issue #21.
func TestExtractCustomWidgetPropertyAssociation_ReturnsAssociationName(t *testing.T) {
	e := &Executor{}
	w := buildComboBoxAssocWidget("MyFirstModule.Task_Category", "MyFirstModule.Category.Name")

	got := e.extractCustomWidgetPropertyAssociation(w, "attributeAssociation")

	if got != "Task_Category" {
		t.Errorf("extractCustomWidgetPropertyAssociation(w, \"attributeAssociation\") = %q, want \"Task_Category\"", got)
	}
}

// TestExtractCustomWidgetPropertyAssociation_WrongKey returns empty for a non-matching key.
func TestExtractCustomWidgetPropertyAssociation_WrongKey(t *testing.T) {
	e := &Executor{}
	w := buildComboBoxAssocWidget("MyFirstModule.Task_Category", "MyFirstModule.Category.Name")

	got := e.extractCustomWidgetPropertyAssociation(w, "nonExistentProperty")

	if got != "" {
		t.Errorf("extractCustomWidgetPropertyAssociation with wrong key = %q, want empty", got)
	}
}

// TestExtractCustomWidgetPropertyAssociation_NilEntityRef returns empty when EntityRef is nil.
func TestExtractCustomWidgetPropertyAssociation_NilEntityRef(t *testing.T) {
	e := &Executor{}
	w := map[string]any{
		"Type": map[string]any{
			"ObjectType": map[string]any{
				"PropertyTypes": []any{
					map[string]any{"$ID": "id-1", "PropertyKey": "attributeAssociation"},
				},
			},
		},
		"Object": map[string]any{
			"Properties": []any{
				map[string]any{
					"TypePointer": "id-1",
					"Value":       map[string]any{"EntityRef": nil},
				},
			},
		},
	}

	got := e.extractCustomWidgetPropertyAssociation(w, "attributeAssociation")

	if got != "" {
		t.Errorf("extractCustomWidgetPropertyAssociation with nil EntityRef = %q, want empty", got)
	}
}

// TestExtractCustomWidgetPropertyAssociation_DoesNotReturnCaptionAttribute confirms that
// the fixer does not accidentally return the CaptionAttribute value.
// This documents the exact bug: the old generic scan returned "Name" (CaptionAttribute)
// because it found the first AttributeRef, which belonged to CaptionAttribute, not Attribute.
func TestExtractCustomWidgetPropertyAssociation_DoesNotReturnCaptionAttribute(t *testing.T) {
	e := &Executor{}
	w := buildComboBoxAssocWidget("MyFirstModule.Task_Category", "MyFirstModule.Category.Name")

	got := e.extractCustomWidgetPropertyAssociation(w, "attributeAssociation")

	if got == "Name" {
		t.Errorf("extractCustomWidgetPropertyAssociation returned CaptionAttribute value %q; this is the original bug (Issue #21)", got)
	}
}

// TestDecodeMicroflowQNFromSource_MicroflowSource verifies that the gen-typed helper
// correctly reads the microflow qualified name via MicroflowSource → MicroflowSettings.
func TestDecodeMicroflowQNFromSource(t *testing.T) {
	const mf = "MyModule.DS_Items"
	ds := map[string]any{
		"$Type": "Forms$MicroflowSource",
		"MicroflowSettings": map[string]any{
			"$Type":     "Forms$MicroflowSettings",
			"Microflow": mf,
		},
	}

	got := decodeMicroflowQNFromSource(ds)

	if got != mf {
		t.Errorf("decodeMicroflowQNFromSource = %q, want %q", got, mf)
	}
}

// TestDecodeNanoflowQNFromSource verifies the gen-typed helper for NanoflowSource.
func TestDecodeNanoflowQNFromSource(t *testing.T) {
	const nf = "MyModule.NF_Items"
	ds := map[string]any{
		"$Type":    "Forms$NanoflowSource",
		"Nanoflow": nf,
	}

	got := decodeNanoflowQNFromSource(ds)

	if got != nf {
		t.Errorf("decodeNanoflowQNFromSource = %q, want %q", got, nf)
	}
}

// buildDataGrid2MicroflowWidget builds a mock DataGrid2 CustomWidget map with a
// Forms$MicroflowSource datasource. This mirrors the BSON written by buildDataGridDataSourceBSON
// for the "microflow" case: datasource is stored as Forms$MicroflowSource with the
// microflow reference nested under MicroflowSettings.Microflow, not at the top level.
func buildDataGrid2MicroflowWidget(microflowQN string) map[string]any {
	const datasourceTypeID = "ds-type-id-001"

	widgetType := map[string]any{
		"WidgetId": "com.mendix.widget.web.datagrid.Datagrid",
		"ObjectType": map[string]any{
			"PropertyTypes": []any{
				map[string]any{"$ID": datasourceTypeID, "PropertyKey": "datasource"},
			},
		},
	}

	properties := []any{
		map[string]any{
			"TypePointer": datasourceTypeID,
			"Value": map[string]any{
				"DataSource": map[string]any{
					"$Type": "Forms$MicroflowSource",
					"MicroflowSettings": map[string]any{
						"$Type":     "Forms$MicroflowSettings",
						"Microflow": microflowQN,
					},
				},
			},
		},
	}

	return map[string]any{
		"Type":   widgetType,
		"Object": map[string]any{"Properties": properties},
	}
}

// TestExtractDataGrid2DataSource_MicroflowSource verifies that extractDataGrid2DataSource
// correctly reads the microflow reference from Forms$MicroflowSource.MicroflowSettings.Microflow.
// Regression test for: extractDataGrid2DataSource read ds["Microflow"] (top-level, always nil)
// instead of ds["MicroflowSettings"]["Microflow"] — causing DESCRIBE to show no datasource
// and column attributes to be unresolvable.
func TestExtractDataGrid2DataSource_MicroflowSource(t *testing.T) {
	const mf = "MyModule.DS_Items"
	w := buildDataGrid2MicroflowWidget(mf)

	got := extractDataGrid2DataSource(nil, w)

	if got == nil {
		t.Fatal("extractDataGrid2DataSource returned nil; want *rawDataSource with microflow reference")
	}
	if got.Type != "microflow" {
		t.Errorf("got.Type = %q, want %q", got.Type, "microflow")
	}
	if got.Reference != mf {
		t.Errorf("got.Reference = %q, want %q", got.Reference, mf)
	}
}

// buildFormsGalleryMicroflowWidget builds a mock Forms$Gallery widget map with a
// Forms$MicroflowSource datasource at the top level (not inside Object.Properties).
// This is the legacy built-in gallery format. Microflow is stored under MicroflowSettings.
func buildFormsGalleryMicroflowWidget(microflowQN string) map[string]any {
	return map[string]any{
		"$Type": "Forms$Gallery",
		"DataSource": map[string]any{
			"$Type": "Forms$MicroflowSource",
			"MicroflowSettings": map[string]any{
				"$Type":     "Forms$MicroflowSettings",
				"Microflow": microflowQN,
			},
		},
	}
}

// TestExtractGalleryDataSource_MicroflowSource verifies that extractGalleryDataSource
// reads the microflow reference from Forms$MicroflowSource.MicroflowSettings.Microflow.
// Regression guard: the same bug as DataGrid2 existed here — direct ds["Microflow"] access.
func TestExtractGalleryDataSource_MicroflowSource(t *testing.T) {
	const mf = "MyModule.DS_GalleryItems"
	w := buildFormsGalleryMicroflowWidget(mf)

	got := extractGalleryDataSource(nil, w)

	if got == nil {
		t.Fatal("extractGalleryDataSource returned nil; want *rawDataSource with microflow reference")
	}
	if got.Type != "microflow" {
		t.Errorf("got.Type = %q, want %q", got.Type, "microflow")
	}
	if got.Reference != mf {
		t.Errorf("got.Reference = %q, want %q", got.Reference, mf)
	}
}
