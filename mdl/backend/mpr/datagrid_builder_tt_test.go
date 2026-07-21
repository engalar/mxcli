// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/widgets"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func init() {
	codec.DefaultRegistry.RegisterAlias("Forms$FormCallArgument", "Forms$LayoutCallArgument")
}

// TestBuildDataGrid2_TextTemplate verifies the DataGrid2 builder produces
// TextTemplate states matching what `mx update-widgets` produces (CE0463
// compliance). Assertions are keyed by PropertyKey (joined via
// TypePointer -> PropertyType.$ID -> PropertyKey), NOT by fragile numeric
// index, so they survive property-order changes across widget versions.
//
// The expected states below are the ground truth captured with the key-based
// BSON oracle (built vs `mx update-widgets` golden) — see
// docs/superpowers/plans and .claude/skills/debug-bson.md. The rule they encode:
//   - non-TextTemplate-type property        -> TextTemplate = null      (Rule 1)
//   - TextTemplate-type, visible in default -> TextTemplate = document  (Rule 2)
//   - TextTemplate-type, hidden in default  -> TextTemplate = null      (Rule 2)
//
// For DataGrid2 the only default-hidden TextTemplate property is
// loadMoreButtonCaption (editorConfig hides it unless pagination == "loadMore").
func TestBuildDataGrid2_TextTemplate(t *testing.T) {
	projectPath := "/mnt/data_sdb/mxcli/app-golden/minimal.mpr"
	b := &MprBackend{}

	widgets.ResetGeneratedCache()
	idGen := types.GenerateID

	spec := backend.DataGridSpec{
		DataSourceBSON: bson.D{
			{Key: "$Type", Value: "CustomWidgets$CustomWidgetDatabaseSource"},
			{Key: "EntityPath", Value: "DebugDG.Item"},
		},
		Columns: []backend.DataGridColumnSpec{
			{Attribute: "Name", Caption: "Name"},
		},
	}

	doc, err := b.buildDataGrid2WidgetDoc(model.ID(idGen()), "dgTest", spec, projectPath)
	if err != nil {
		t.Fatal("buildDataGrid2WidgetDoc:", err)
	}
	rawBytes, err := bson.Marshal(doc)
	if err != nil {
		t.Fatal("marshal:", err)
	}
	states := topLevelTextTemplateByKey(t, bson.Raw(rawBytes))

	// Golden truth from `mx update-widgets` (key-based oracle).
	want := map[string]string{
		// TextTemplate-type, visible -> document
		"filterSectionTitle":            "document",
		"exportDialogLabel":             "document",
		"cancelExportLabel":             "document",
		"selectRowLabel":                "document",
		"selectAllRowsLabel":            "document",
		"selectedCountTemplateSingular": "document",
		"selectedCountTemplatePlural":   "document",
		// TextTemplate-type, hidden in default mode -> null
		"loadMoreButtonCaption": "null",
	}
	for key, exp := range want {
		got, ok := states[key]
		if !ok {
			t.Errorf("property %q not found in built widget", key)
			continue
		}
		if got != exp {
			t.Errorf("TextTemplate[%s] = %s, want %s", key, got, exp)
		}
	}
}

// --- key-based helpers (mirror /tmp/tt_oracle.py join logic in Go) ---

// topLevelTextTemplateByKey returns PropertyKey -> TextTemplate state
// ("null"/"document"/"absent") for the CustomWidget's top-level Object
// properties, joining Object.Properties[].TypePointer to
// Type.ObjectType.PropertyTypes[].$ID -> PropertyKey.
func topLevelTextTemplateByKey(t *testing.T, raw bson.Raw) map[string]string {
	t.Helper()

	// Build $ID(hex) -> PropertyKey from Type.ObjectType.PropertyTypes.
	id2key := map[string]string{}
	typeRaw, err := raw.LookupErr("Type")
	if err != nil {
		t.Fatal("no Type:", err)
	}
	otRaw, err := bson.Raw(typeRaw.Document()).LookupErr("ObjectType")
	if err != nil {
		t.Fatal("no ObjectType:", err)
	}
	ptsRaw, err := bson.Raw(otRaw.Document()).LookupErr("PropertyTypes")
	if err != nil {
		t.Fatal("no PropertyTypes:", err)
	}
	ptVals, _ := ptsRaw.Array().Values()
	for _, pv := range ptVals {
		if pv.Type != bson.TypeEmbeddedDocument {
			continue
		}
		pd := bson.Raw(pv.Document())
		idHex := binaryHex(pd, "$ID")
		key, _ := pd.Lookup("PropertyKey").StringValueOK()
		if idHex != "" && key != "" {
			id2key[idHex] = key
		}
	}

	// Walk Object.Properties, map TypePointer -> key -> TextTemplate state.
	out := map[string]string{}
	objRaw, err := raw.LookupErr("Object")
	if err != nil {
		t.Fatal("no Object:", err)
	}
	propsRaw, err := bson.Raw(objRaw.Document()).LookupErr("Properties")
	if err != nil {
		t.Fatal("no Object.Properties:", err)
	}
	propVals, _ := propsRaw.Array().Values()
	for _, pv := range propVals {
		if pv.Type != bson.TypeEmbeddedDocument {
			continue
		}
		pd := bson.Raw(pv.Document())
		tp := binaryHex(pd, "TypePointer")
		key, ok := id2key[tp]
		if !ok {
			continue
		}
		out[key] = textTemplateStateOf(pd)
	}
	return out
}

// binaryHex returns the hex string of a BSON binary field, or "".
func binaryHex(d bson.Raw, field string) string {
	v, err := d.LookupErr(field)
	if err != nil || v.Type != bson.TypeBinary {
		return ""
	}
	_, data := v.Binary()
	const hexdigits = "0123456789abcdef"
	buf := make([]byte, len(data)*2)
	for i, bb := range data {
		buf[i*2] = hexdigits[bb>>4]
		buf[i*2+1] = hexdigits[bb&0x0f]
	}
	return string(buf)
}

// textTemplateStateOf returns the TextTemplate state of a property document's Value.
func textTemplateStateOf(propDoc bson.Raw) string {
	valRaw, err := propDoc.LookupErr("Value")
	if err != nil || valRaw.Type != bson.TypeEmbeddedDocument {
		return "absent"
	}
	tt, err := bson.Raw(valRaw.Document()).LookupErr("TextTemplate")
	if err != nil {
		return "absent"
	}
	switch tt.Type {
	case bson.TypeNull:
		return "null"
	case bson.TypeEmbeddedDocument:
		return "document"
	default:
		return tt.Type.String()
	}
}
