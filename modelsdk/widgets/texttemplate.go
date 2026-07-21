// SPDX-License-Identifier: Apache-2.0

package widgets

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
			// Rule 1: non-TextTemplate-type property -- TextTemplate must be null.
			wantNull = true
		} else if hidden[key] {
			// Rule 2: TextTemplate-type, hidden in default -- TextTemplate must be null.
			wantNull = true
		}
		// Rule 3 (else): TextTemplate-type, visible -- keep the ClientTemplate as-is.

		if wantNull {
			if val, ok := getMapField(opMap, "Value"); ok {
				val["TextTemplate"] = nil
			}
		}
	}
}
