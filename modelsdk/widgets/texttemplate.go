// SPDX-License-Identifier: Apache-2.0

package widgets

import (
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/mdl/types"
)

// extractBinary returns a []byte from an interface{} value, or nil.
// BSON binary fields deserialize as bson.Binary (not []byte) when
// unmarshalled into map[string]any, so we handle both forms.
func extractBinary(v any) []byte {
	if v == nil {
		return nil
	}
	switch x := v.(type) {
	case []byte:
		return x
	case bson.Binary:
		return x.Data
	}
	return nil
}

// This file is the SINGLE source of truth for a pluggable widget's top-level
// TextTemplate (Forms$ClientTemplate) states. Getting them wrong produces
// CE0463 "the definition of this widget has changed".
//
// The rule, established with a key-based BSON oracle that diffs mxcli output
// against `mx update-widgets` golden output (see .claude/skills/debug-bson.md):
//
//	Every WidgetValue carries an (empty) TextTemplate ClientTemplate by DEFAULT,
//	EXCEPT properties that are HIDDEN in the widget's default editor mode, which
//	carry TextTemplate = null. Property TYPE is irrelevant — visibility decides.
//
// mxcli's builders already produce a ClientTemplate for every property (via
// createDefaultWidgetValue) and, for DataGrid2, nullify its non-TextTemplate
// properties in datagrid_builder.go. What was missing — and what this file adds —
// is nullifying the small, widget-specific set of DEFAULT-HIDDEN properties.
//
// The previous implementation instead keyed off ValueType.Translations, which is
// uncorrelated with visibility (loadMoreButtonCaption ships with a translation
// yet is hidden; the accessibility labels ship without one yet are visible), so
// it nullified and kept the wrong properties.

// defaultHiddenTextTemplates maps widgetID -> set of property keys that are
// HIDDEN in the widget's DEFAULT editor mode and must therefore carry
// TextTemplate = null. Includes both TextTemplate-type and other-type properties
// (a hidden non-TextTemplate property is also nulled by `mx update-widgets`).
//
// Anti-pattern justification (per repo SOLID mandate): this is a hand-maintained
// table. The authoritative source is each widget's editorConfig.js hide predicate
// (minified JavaScript, e.g. `"loadMore" !== e.pagination &&
// hidePropertyIn(..., "loadMoreButtonCaption")`); evaluating it would require a
// JS engine at build time — disproportionate. mxcli only ever generates widgets
// in their DEFAULT property mode, so the hidden set is small and stable.
// TestBuildDataGrid2_TextTemplate and the key-based BSON oracle validate this
// table against real `mx update-widgets` output, so it cannot silently drift.
// Only widgets that mxcli generates need an entry; DataGrid2's non-TextTemplate
// properties are nullified in datagrid_builder.go, so only its default-hidden
// TextTemplate property is listed here.
var defaultHiddenTextTemplates = map[string]map[string]bool{
	"com.mendix.widget.web.datagrid.Datagrid": {
		"loadMoreButtonCaption": true, // hidden unless pagination == "loadMore"
	},
	"com.mendix.widget.web.gallery.Gallery": {
		"emptyPlaceholder":      true, // hidden unless showEmptyPlaceholder == "custom"
		"loadMoreButtonCaption": true, // hidden unless pagination == "loadMore"
	},
	"com.mendix.widget.web.datagriddropdownfilter.DatagridDropdownFilter": {
		// Association-source properties are hidden in the default (attribute) mode.
		"attr":                          true,
		"refEntity":                     true,
		"linkedDs":                      true,
		"refCaption":                    true,
		"refSearchAttr":                 true,
		"refOptions":                    true,
		"filterInputPlaceholderCaption": true,
	},
	"com.mendix.widget.web.image.Image": {
		// imageUrl is a TextTemplate-type property hidden in the default editor mode;
		// mx update-widgets nullifies it. Only visible when displayAs == "imageUrl".
		"imageUrl": true,
	},
	"com.mendix.widget.web.treenode.TreeNode": {
		// headerCaption is a TextTemplate-type property hidden in the default editor
		// mode (headerType == "none"); mx update-widgets nullifies it.
		"headerCaption": true,
	},
	"com.mendix.widget.web.accordion.Accordion": {
		// groups.headerText is the TextTemplate of each Accordion group. In the default
		// editor mode it is hidden, so mx update-widgets nullifies its TextTemplate.
		"groups.headerText": true,
	},
}

// normalizeTextTemplateStates nullifies the TextTemplate of every top-level
// WidgetValue whose PropertyKey is in the widget's default-hidden set, matching
// `mx update-widgets`. Visible properties are left untouched (they already carry
// a ClientTemplate from createDefaultWidgetValue). Nested ObjectType properties
// (DataGrid2 columns, chart series) are mode-dependent per instance and are
// handled by the concrete builders.
//
// It must run AFTER augmentation (so the property set is settled) and BEFORE $ID
// remapping (createDefaultClientTemplate emits placeholder IDs that
// loader.collectIDs later rewrites).
func normalizeTextTemplateStates(tmpl *WidgetTemplate) {
	if tmpl == nil {
		return
	}
	objType, ok := getMapField(tmpl.Type, "ObjectType")
	if !ok {
		return
	}
	propTypes, ok := getArrayField(objType, "PropertyTypes")
	if !ok {
		return
	}
	objProps, ok := getArrayField(tmpl.Object, "Properties")
	if !ok {
		return
	}

	// Build $ID -> PropertyKey and $ID -> is TextTemplate-type
	idToKey := make(map[string]string, len(propTypes))
	typeIsTextTemplate := make(map[string]bool, len(propTypes))
	for _, pt := range propTypes {
		ptMap, ok := pt.(map[string]any)
		if !ok {
			continue
		}
		id, _ := ptMap["$ID"].(string)
		key, _ := ptMap["PropertyKey"].(string)
		if id != "" && key != "" {
			idToKey[id] = key
		}
		if vt, ok := getMapField(ptMap, "ValueType"); ok {
			if vtType, _ := vt["Type"].(string); vtType == "TextTemplate" {
				typeIsTextTemplate[id] = true
			}
		}
	}
	hidden := defaultHiddenTextTemplates[tmpl.WidgetID]

	for _, op := range objProps {
		opMap, ok := op.(map[string]any)
		if !ok {
			continue
		}
		tp, _ := opMap["TypePointer"].(string)
		key := idToKey[tp]
		wantNull := false
		if !typeIsTextTemplate[tp] {
			wantNull = true
		} else if hidden[key] {
			wantNull = true
		}

		if wantNull {
			if val, ok := getMapField(opMap, "Value"); ok {
				val["TextTemplate"] = nil
			}
		}
	}
}

// NormalizeNestedTextTemplates nullifies TextTemplate for nested Object-type
// properties (e.g., Accordion groups[].headerText) after merge. These nested
// properties don't exist in the canonical template (which has empty Object
// arrays for Object-type values), so they must be normalized after the current
// Object's nested data is merged in.
//
// Hidden entries in defaultHiddenTextTemplates use a dot separator: "groups.headerText"
// means the top-level property "groups" has nested property "headerText" that should
// have TextTemplate = null.
func NormalizeNestedTextTemplates(mergedObj, typ map[string]any, widgetID string) {
	if widgetID == "" {
		return
	}
	hidden := defaultHiddenTextTemplates[widgetID]

	nestedHidden := make(map[string]map[string]bool)
	for key, val := range hidden {
		if parts := strings.SplitN(key, ".", 2); len(parts) == 2 {
			parent, child := parts[0], parts[1]
			if nestedHidden[parent] == nil {
				nestedHidden[parent] = make(map[string]bool)
			}
			nestedHidden[parent][child] = val
		}
	}
	if len(nestedHidden) == 0 {
		return
	}

	objType, ok := getMapField(typ, "ObjectType")
	if !ok {
		return
	}
	propTypes, ok := getArrayField(objType, "PropertyTypes")
	if !ok {
		return
	}

	// Build top-level PropertyType ID -> PropertyKey and
	// parentKey -> nestedPropertyKey -> isTextTemplate for ObjectType properties.
	// IDs in typ come from BSON, so $ID is []byte (binary blob).
	// Use BlobToUUID for consistent matching.
	idToKey := make(map[string]string, len(propTypes))
	nestedTypeInfo := make(map[string]map[string]bool)  // parentKey -> childKey -> isTextTemplate
	nestedIDToKey := make(map[string]map[string]string) // parentKey -> childID (UUID-with-dashes) -> childKey

	for _, pt := range propTypes {
		ptMap, ok := pt.(map[string]any)
		if !ok {
			continue
		}
		id := types.BlobToUUID(extractBinary(ptMap["$ID"]))
		key, _ := ptMap["PropertyKey"].(string)
		if id != "" && key != "" {
			idToKey[id] = key
		}

		vt, ok := getMapField(ptMap, "ValueType")
		if !ok {
			continue
		}
		nestedOT, ok := getMapField(vt, "ObjectType")
		if !ok || nestedOT == nil {
			continue
		}
		nestedPTs, ok := getArrayField(nestedOT, "PropertyTypes")
		if !ok {
			continue
		}

		info := make(map[string]bool, len(nestedPTs))
		nidMap := make(map[string]string, len(nestedPTs))
		for _, npt := range nestedPTs {
			nptMap, ok := npt.(map[string]any)
			if !ok {
				continue
			}
			nkey, _ := nptMap["PropertyKey"].(string)
			nid := types.BlobToUUID(extractBinary(nptMap["$ID"]))
			if nkey == "" || nid == "" {
				continue
			}
			nidMap[nid] = nkey
			nvt, ok := getMapField(nptMap, "ValueType")
			if ok {
				vtType, _ := nvt["Type"].(string)
				info[nkey] = (vtType == "TextTemplate")
			}
		}
		nestedTypeInfo[key] = info
		nestedIDToKey[key] = nidMap
	}

	objProps, ok := getArrayField(mergedObj, "Properties")
	if !ok {
		return
	}

	for _, op := range objProps {
		opMap, ok := op.(map[string]any)
		if !ok {
			continue
		}

		tp := types.BlobToUUID(extractBinary(opMap["TypePointer"]))
		key := idToKey[tp]
		if key == "" {
			continue
		}

		childHidden := nestedHidden[key]
		info := nestedTypeInfo[key]
		nidMap := nestedIDToKey[key]
		if len(childHidden) == 0 && info == nil {
			continue
		}

		val, ok := getMapField(opMap, "Value")
		if !ok {
			continue
		}
		objects, ok := getArrayField(val, "Objects")
		if !ok {
			continue
		}

		for _, obj := range objects {
			objMap, ok := obj.(map[string]any)
			if !ok {
				continue
			}
			nestedProps, ok := getArrayField(objMap, "Properties")
			if !ok {
				continue
			}
			for _, np := range nestedProps {
				npMap, ok := np.(map[string]any)
				if !ok {
					continue
				}

				ntp := types.BlobToUUID(extractBinary(npMap["TypePointer"]))
				if ntp == "" {
					continue
				}

				nKey := nidMap[ntp]
				if nKey == "" {
					continue
				}

				isTextTemplate := info[nKey]
				isHidden := childHidden[nKey]

				wantNull := false
				if !isTextTemplate {
					wantNull = true
				} else if isHidden {
					wantNull = true
				}

				if wantNull {
					if nval, ok := getMapField(npMap, "Value"); ok {
						nval["TextTemplate"] = nil
					}
				}
			}
		}
	}
}
