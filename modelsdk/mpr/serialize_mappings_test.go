// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/model"
)

// TestSerializeImportMapping_RootBindsToSource verifies the root object mapping
// element gets ExposedName "Root" and MinOccurs 1 so it binds to the JSON
// structure root. A blank ExposedName triggers CE0271 in mx check.
func TestSerializeImportMapping_RootBindsToSource(t *testing.T) {
	t.Parallel()
	im := &model.ImportMapping{
		Name:          "WorkOrderImport_Mapping",
		JsonStructure: "FT.WorkOrderPayload",
		Elements: []*model.ImportMappingElement{
			{
				Kind:           "Object",
				Entity:         "FT.WorkOrderImport",
				ObjectHandling: "Create",
				// ExposedName intentionally blank — serialize must fill "Root".
			},
		},
	}
	im.ID = model.ID("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	contents, err := SerializeImportMapping(im)
	if err != nil {
		t.Fatalf("SerializeImportMapping: %v", err)
	}

	root := rootMappingElement(t, contents)
	if got := bsonString(root, "ExposedName"); got != "Root" {
		t.Errorf("root ExposedName = %q, want Root", got)
	}
	// MinOccurs is intentionally NOT forced to 1: it must match the JSON
	// structure root element (default 0). Forcing it triggers CE5015.
	if got := bsonInt(root, "MinOccurs"); got != 0 {
		t.Errorf("root MinOccurs = %d, want 0 (must match schema root)", got)
	}
}

// TestSerializeExportMapping_RootExposedName verifies the export root element
// gets ExposedName "Root" when blank.
func TestSerializeExportMapping_RootExposedName(t *testing.T) {
	t.Parallel()
	em := &model.ExportMapping{
		Name:          "DispatchOrder_Export",
		JsonStructure: "FT.WorkOrderPayload",
		Elements: []*model.ExportMappingElement{
			{
				Kind:           "Object",
				Entity:         "FT.DispatchOrder",
				ObjectHandling: "Parameter",
			},
		},
	}
	em.ID = model.ID("cccccccc-cccc-cccc-cccc-cccccccccccc")

	contents, err := SerializeExportMapping(em)
	if err != nil {
		t.Fatalf("SerializeExportMapping: %v", err)
	}

	root := rootMappingElement(t, contents)
	if got := bsonString(root, "ExposedName"); got != "Root" {
		t.Errorf("root ExposedName = %q, want Root", got)
	}
}

func rootMappingElement(t *testing.T, contents []byte) bson.D {
	t.Helper()
	var doc bson.D
	if err := bson.Unmarshal(contents, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, e := range doc {
		if e.Key != "Elements" {
			continue
		}
		arr, ok := e.Value.(bson.A)
		if !ok || len(arr) < 2 {
			t.Fatalf("Elements = %v, want version marker + root", e.Value)
		}
		root, ok := arr[1].(bson.D)
		if !ok {
			t.Fatalf("Elements[1] = %T, want bson.D", arr[1])
		}
		return root
	}
	t.Fatal("no Elements field")
	return nil
}

func bsonString(doc bson.D, key string) string {
	for _, e := range doc {
		if e.Key == key {
			s, _ := e.Value.(string)
			return s
		}
	}
	return ""
}

func bsonInt(doc bson.D, key string) int {
	for _, e := range doc {
		if e.Key == key {
			switch v := e.Value.(type) {
			case int32:
				return int(v)
			case int64:
				return int(v)
			case int:
				return v
			}
		}
	}
	return -999
}
