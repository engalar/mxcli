// SPDX-License-Identifier: Apache-2.0

// widget_builder_filteroptions_test.go — regression guard for the CE0463 bug
// where a Gallery/DataGrid drop-down filter (DatagridDropdownFilter) got a
// spurious default `filterOptions` entry seeded into it.
//
// Root cause: ensureRequiredObjectLists() seeded a default WidgetObject into
// *optional* object-list properties. The DatagridDropdownFilter `filterOptions`
// property is OPTIONAL (Required=false) and, when the MDL specifies no manual
// options, must stay empty. Seeding a default entry (caption=" ", value="")
// produced a widget instance that Studio Pro rejected with CE0463
// ("The definition of this widget has changed") because the stored object no
// longer matched the platform widget definition. `mx update-widgets` fixed it by
// emptying filterOptions — confirming empty is the correct output.
//
// Evidence chain: full built fStatus widget → CE0463=1; replacing only its
// filterOptions with an empty list → CE0463=0.
package mprbackend

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/mdl/types"
)

// objectListEntryCount returns the number of WidgetObject entries in the named
// object-list property's Value.Objects array (excluding the leading version marker).
func objectListEntryCount(t *testing.T, obj bson.D, propTypeIDs map[string]types.PropertyTypeIDEntry, propKey string) int {
	t.Helper()
	entry := propTypeIDs[propKey]
	for _, elem := range obj {
		if elem.Key != "Properties" {
			continue
		}
		arr, ok := elem.Value.(bson.A)
		if !ok {
			return -1
		}
		for _, item := range arr {
			prop, ok := item.(bson.D)
			if !ok {
				continue
			}
			if !matchesTypePointer(prop, entry.PropertyTypeID) {
				continue
			}
			for _, pe := range prop {
				if pe.Key != "Value" {
					continue
				}
				val, ok := pe.Value.(bson.D)
				if !ok {
					return -1
				}
				for _, ve := range val {
					if ve.Key == "Objects" {
						if objs, ok := ve.Value.(bson.A); ok {
							n := len(objs) - 1 // subtract version marker
							if n < 0 {
								n = 0
							}
							return n
						}
					}
				}
			}
		}
	}
	return -1
}

// makeOptionalObjectListWidget builds a minimal widget object + propertyTypeIDs
// modelling a DatagridDropdownFilter `filterOptions` (optional object list with
// nested caption[TextTemplate] + value[Expression] props) whose list is empty.
func makeOptionalObjectListWidget() (bson.D, map[string]types.PropertyTypeIDEntry) {
	filterOptionsPTID := types.GenerateID()
	filterOptionsVTID := types.GenerateID()
	objectTypeID := types.GenerateID()
	captionPTID := types.GenerateID()
	captionVTID := types.GenerateID()
	valuePTID := types.GenerateID()
	valueVTID := types.GenerateID()

	propTypeIDs := map[string]types.PropertyTypeIDEntry{
		"filterOptions": {
			PropertyTypeID: filterOptionsPTID,
			ValueTypeID:    filterOptionsVTID,
			ValueType:      "Object",
			Required:       false, // OPTIONAL — must not be seeded
			ObjectTypeID:   objectTypeID,
			NestedKeyOrder: []string{"caption", "value"},
			NestedPropertyIDs: map[string]types.PropertyTypeIDEntry{
				"caption": {PropertyTypeID: captionPTID, ValueTypeID: captionVTID, ValueType: "TextTemplate", Required: true},
				"value":   {PropertyTypeID: valuePTID, ValueTypeID: valueVTID, ValueType: "Expression", Required: true},
			},
		},
	}

	obj := bson.D{
		{Key: "$Type", Value: "CustomWidgets$WidgetObject"},
		{Key: "Properties", Value: bson.A{
			int32(2),
			bson.D{
				{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
				{Key: "$Type", Value: "CustomWidgets$WidgetProperty"},
				{Key: "TypePointer", Value: types.UUIDToBlob(filterOptionsPTID)},
				{Key: "Value", Value: bson.D{
					{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
					{Key: "$Type", Value: "CustomWidgets$WidgetValue"},
					{Key: "TypePointer", Value: types.UUIDToBlob(filterOptionsVTID)},
					{Key: "Objects", Value: bson.A{int32(2)}}, // empty list
				}},
			},
		}},
	}
	return obj, propTypeIDs
}

// TestEnsureRequiredObjectLists_OptionalListNotSeeded is the CE0463 regression
// guard: an optional object-list property (like DatagridDropdownFilter
// filterOptions) must remain empty — never get a default entry seeded.
func TestEnsureRequiredObjectLists_OptionalListNotSeeded(t *testing.T) {
	obj, propTypeIDs := makeOptionalObjectListWidget()

	if got := objectListEntryCount(t, obj, propTypeIDs, "filterOptions"); got != 0 {
		t.Fatalf("precondition: expected empty filterOptions before seeding, got %d entries", got)
	}

	out := ensureRequiredObjectLists(obj, propTypeIDs)

	if got := objectListEntryCount(t, out, propTypeIDs, "filterOptions"); got != 0 {
		t.Errorf("optional filterOptions must stay empty, but %d default entry(ies) were seeded (CE0463 regression)", got)
	}
}

// TestEnsureRequiredObjectLists_RequiredListSeeded guards the other direction:
// a genuinely required object list still gets its default entry, so the fix does
// not over-correct.
func TestEnsureRequiredObjectLists_RequiredListSeeded(t *testing.T) {
	obj, propTypeIDs := makeOptionalObjectListWidget()
	// Flip filterOptions to required to model a mandatory object list.
	entry := propTypeIDs["filterOptions"]
	entry.Required = true
	propTypeIDs["filterOptions"] = entry

	out := ensureRequiredObjectLists(obj, propTypeIDs)

	if got := objectListEntryCount(t, out, propTypeIDs, "filterOptions"); got != 1 {
		t.Errorf("required object list should be seeded with exactly one default entry, got %d", got)
	}
}
