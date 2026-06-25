// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.5.D0 — gen-typed PageMutator sibling BSON-shape tests.
//
// Parity note: the gen path (codec.Encoder → bson.D) and the legacy path
// (mpr.SerializeWidget hand-crafted BSON) produce different wire shapes for
// the same semantic widget — the legacy serializer emits extra defaults for
// backward decode. We therefore do NOT assert byte-identity vs the legacy path.
// Instead we verify:
//  1. serializeElementToBsonD produces a bson.D with the correct $Type and scalar fields.
//  2. InsertWidgetGen correctly stitches the encoded bson.D into the widget tree.
//  3. ReplaceWidgetGen correctly replaces the target widget.
//  4. SetWidgetDataSourceGen correctly sets the DataSource field.
//  5. Mock returns a descriptive error when Func is nil.
//
// Full cross-path parity (legacy vs gen) requires a gen widget decoded from
// the exact BSON the legacy serializer produces — that is out of scope for D0.

package mprbackend

import (
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genPages "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// ---------------------------------------------------------------------------
// serializeElementToBsonD shape tests
// ---------------------------------------------------------------------------

func TestSerializeElementToBsonD_TextBoxShape(t *testing.T) {
	t.Parallel()
	tb := genPages.NewTextBox()
	tb.SetID(element.ID(mmpr.GenerateID()))
	tb.SetName("txtFoo")

	doc, err := serializeElementToBsonD(tb)
	if err != nil {
		t.Fatalf("serializeElementToBsonD: %v", err)
	}
	got := make(map[string]any, len(doc))
	for _, e := range doc {
		got[e.Key] = e.Value
	}

	if got["$Type"] != "Forms$TextBox" {
		t.Errorf("$Type = %v, want Forms$TextBox", got["$Type"])
	}
	if got["Name"] != "txtFoo" {
		t.Errorf("Name = %v, want txtFoo", got["Name"])
	}
}

func TestSerializeElementToBsonD_NilInput(t *testing.T) {
	t.Parallel()
	_, err := serializeElementToBsonD(nil)
	if err == nil {
		t.Error("expected error for nil element")
	}
}

// ---------------------------------------------------------------------------
// InsertWidgetGen tree-stitching test
// ---------------------------------------------------------------------------

func TestPageMutator_InsertWidgetGen_StitchesIntoTree(t *testing.T) {
	t.Parallel()
	// Build a raw page with one existing TextBox widget.
	existing := makeWidget("txtExisting", "Forms$TextBox")
	rawPage := makeRawPage(existing)

	m := &mprPageMutator{
		rawData:       rawPage,
		containerType: backend.ContainerPage,
		widgetFinder:  findBsonWidget,
	}

	// Create a gen TextBox to insert after "txtExisting".
	tb := genPages.NewTextBox()
	tb.SetID(element.ID(mmpr.GenerateID()))
	tb.SetName("txtInserted")

	err := m.InsertWidgetGen("txtExisting", "", backend.InsertAfter, []element.Element{tb})
	if err != nil {
		t.Fatalf("InsertWidgetGen: %v", err)
	}

	// Verify "txtInserted" now appears in the widget tree.
	result := findBsonWidget(m.rawData, "txtInserted")
	if result == nil {
		t.Fatal("expected txtInserted in tree after InsertWidgetGen")
	}
	if dGetString(result.widget, "Name") != "txtInserted" {
		t.Errorf("Name = %q, want txtInserted", dGetString(result.widget, "Name"))
	}
	if dGetString(result.widget, "$Type") != "Forms$TextBox" {
		t.Errorf("$Type = %q, want Forms$TextBox", dGetString(result.widget, "$Type"))
	}

	// Verify "txtInserted" comes AFTER "txtExisting" (index 1).
	existingResult := findBsonWidget(m.rawData, "txtExisting")
	if existingResult == nil {
		t.Fatal("expected txtExisting to remain in tree")
	}
	if result.index <= existingResult.index {
		t.Errorf("txtInserted index %d should be > txtExisting index %d", result.index, existingResult.index)
	}
}

// ---------------------------------------------------------------------------
// ReplaceWidgetGen tree-stitching test
// ---------------------------------------------------------------------------

func TestPageMutator_ReplaceWidgetGen_ReplacesTarget(t *testing.T) {
	t.Parallel()
	existing := makeWidget("txtOld", "Forms$TextBox")
	rawPage := makeRawPage(existing)

	m := &mprPageMutator{
		rawData:       rawPage,
		containerType: backend.ContainerPage,
		widgetFinder:  findBsonWidget,
	}

	replacement := genPages.NewTextBox()
	replacement.SetID(element.ID(mmpr.GenerateID()))
	replacement.SetName("txtNew")

	err := m.ReplaceWidgetGen("txtOld", "", []element.Element{replacement})
	if err != nil {
		t.Fatalf("ReplaceWidgetGen: %v", err)
	}

	// txtOld must be gone.
	if findBsonWidget(m.rawData, "txtOld") != nil {
		t.Error("txtOld should have been replaced")
	}
	// txtNew must be present.
	result := findBsonWidget(m.rawData, "txtNew")
	if result == nil {
		t.Fatal("expected txtNew in tree after ReplaceWidgetGen")
	}
}

// BUG-08: REPLACE must allow the replacement widget to reuse the target's
// name. The mpr layer itself imposes no duplicate-name check (that lives in
// the executor's pageBuilder.registerWidgetName via widgetScope), but pin
// the behaviour here so a future tightening doesn't regress the fix.
func TestPageMutator_ReplaceWidgetGen_SameName(t *testing.T) {
	t.Parallel()
	existing := makeWidget("txtLabel", "Forms$TextBox")
	rawPage := makeRawPage(existing)

	m := &mprPageMutator{
		rawData:       rawPage,
		containerType: backend.ContainerPage,
		widgetFinder:  findBsonWidget,
	}

	replacement := genPages.NewTextBox()
	replacement.SetID(element.ID(mmpr.GenerateID()))
	replacement.SetName("txtLabel")

	if err := m.ReplaceWidgetGen("txtLabel", "", []element.Element{replacement}); err != nil {
		t.Fatalf("ReplaceWidgetGen same-name: %v", err)
	}

	result := findBsonWidget(m.rawData, "txtLabel")
	if result == nil {
		t.Fatal("expected txtLabel to remain after same-name REPLACE")
	}
}

// ---------------------------------------------------------------------------
// SetWidgetDataSourceGen test
// ---------------------------------------------------------------------------

func TestPageMutator_SetWidgetDataSourceGen_SetsDataSource(t *testing.T) {
	t.Parallel()
	// Build a DataView widget that includes a pre-existing nil DataSource slot,
	// matching how real Mendix pages store DataViews. dSet only updates existing
	// keys, so the slot must be present for the assignment to take effect.
	dv := bson.D{
		{Key: "$Type", Value: "Forms$DataView"},
		{Key: "Name", Value: "dvMain"},
		{Key: "DataSource", Value: nil},
	}
	rawPage := makeRawPage(dv)

	m := &mprPageMutator{
		rawData:       rawPage,
		containerType: backend.ContainerPage,
		widgetFinder:  findBsonWidget,
	}

	ds := genPages.NewDataViewSource()
	ds.SetID(element.ID(mmpr.GenerateID()))

	err := m.SetWidgetDataSourceGen("dvMain", ds)
	if err != nil {
		t.Fatalf("SetWidgetDataSourceGen: %v", err)
	}

	// DataSource field should now exist on the widget.
	result := findBsonWidget(m.rawData, "dvMain")
	if result == nil {
		t.Fatal("dvMain not found after SetWidgetDataSourceGen")
	}
	dsVal := dGet(result.widget, "DataSource")
	if dsVal == nil {
		t.Fatal("DataSource not set on dvMain")
	}
	dsDoc, ok := dsVal.(bson.D)
	if !ok {
		t.Fatalf("DataSource value is %T, want bson.D", dsVal)
	}
	typeName := dGetString(dsDoc, "$Type")
	if !strings.Contains(typeName, "DataViewSource") {
		t.Errorf("$Type = %q, want to contain DataViewSource", typeName)
	}
}

// ---------------------------------------------------------------------------
// Error cases
// ---------------------------------------------------------------------------

func TestPageMutator_InsertWidgetGen_WidgetNotFound(t *testing.T) {
	t.Parallel()
	rawPage := makeRawPage(makeWidget("txtExisting", "Forms$TextBox"))
	m := &mprPageMutator{rawData: rawPage, widgetFinder: findBsonWidget}

	tb := genPages.NewTextBox()
	tb.SetName("txtNew")
	err := m.InsertWidgetGen("nonexistent", "", backend.InsertAfter, []element.Element{tb})
	if err == nil {
		t.Error("expected error for missing widget ref")
	}
}

// ---------------------------------------------------------------------------
// Mock descriptive error tests
// ---------------------------------------------------------------------------

func TestMockPageMutator_InsertWidgetGenNotConfigured(t *testing.T) {
	t.Parallel()
	m := &mock.MockPageMutator{}
	err := m.InsertWidgetGen("w", "", backend.InsertAfter, nil)
	if err == nil {
		t.Fatal("expected error when InsertWidgetGenFunc is nil")
	}
	if !strings.Contains(err.Error(), "MockPageMutator.InsertWidgetGen not configured") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestMockPageMutator_ReplaceWidgetGenNotConfigured(t *testing.T) {
	t.Parallel()
	m := &mock.MockPageMutator{}
	err := m.ReplaceWidgetGen("w", "", nil)
	if err == nil {
		t.Fatal("expected error when ReplaceWidgetGenFunc is nil")
	}
	if !strings.Contains(err.Error(), "MockPageMutator.ReplaceWidgetGen not configured") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestMockPageMutator_SetWidgetDataSourceGenNotConfigured(t *testing.T) {
	t.Parallel()
	m := &mock.MockPageMutator{}
	err := m.SetWidgetDataSourceGen("w", nil)
	if err == nil {
		t.Fatal("expected error when SetWidgetDataSourceGenFunc is nil")
	}
	if !strings.Contains(err.Error(), "MockPageMutator.SetWidgetDataSourceGen not configured") {
		t.Errorf("unexpected error message: %v", err)
	}
}
