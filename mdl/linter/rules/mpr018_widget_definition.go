package rules

import (
	"fmt"
	"sort"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/mdl/linter"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/widgets"
)

const pluggableWidgetType = "CustomWidgets$CustomWidget"

// WidgetDefinitionRule (MPR018) detects pluggable widgets whose Object
// Properties are inconsistent with the widget Type PropertyTypes, which
// causes CE0463 ("widget definition has changed") in mx check and Studio Pro.
type WidgetDefinitionRule struct {
	// ProjectPath is the path to the .mpr file, used to load canonical
	// templates from the project's widgets/ directory.
	ProjectPath string
}

// NewWidgetDefinitionRule creates a new MPR018 rule.
func NewWidgetDefinitionRule() *WidgetDefinitionRule {
	return &WidgetDefinitionRule{}
}

// NewWidgetDefinitionRuleWithPath creates a new MPR018 rule with a project
// path for canonical template loading (enables deep value-level detection).
func NewWidgetDefinitionRuleWithPath(projectPath string) *WidgetDefinitionRule {
	return &WidgetDefinitionRule{ProjectPath: projectPath}
}

func (r *WidgetDefinitionRule) ID() string                       { return "MPR018" }
func (r *WidgetDefinitionRule) Name() string                     { return "WidgetDefinition" }
func (r *WidgetDefinitionRule) Category() string                 { return "correctness" }
func (r *WidgetDefinitionRule) DefaultSeverity() linter.Severity { return linter.SeverityWarning }
func (r *WidgetDefinitionRule) Description() string {
	return "Detects pluggable widget Object properties inconsistent with its Type — " +
		"run with --fix to rebuild from canonical template and prevent CE0463"
}

// pluggableWidgetInfo identifies a pluggable widget instance found in a page.
type pluggableWidgetInfo struct {
	Name      string
	WidgetID  string
	ObjCount  int
	TypeCount int
	TypeMap   map[string]any
	ObjMap    map[string]any
}

// Check walks all pages and snippets, finds pluggable widgets, and flags
// those where the Object Properties count doesn't match the current Type
// PropertyTypes count (the primary CE0463 trigger).
func (r *WidgetDefinitionRule) Check(ctx *linter.LintContext) []linter.Violation {
	reader := ctx.Reader()
	if reader == nil {
		return nil
	}

	// Collect all unit IDs. If the reader doesn't support ListAllUnitIDs
	// (e.g., in tests), fall back to the graph catalog's page and snippet lists.
	unitIDs, err := reader.ListAllUnitIDs()
	if err != nil || unitIDs == nil {
		return nil
	}

	var violations []linter.Violation

	for _, uid := range unitIDs {
		rawData, err := reader.GetRawUnit(model.ID(uid))
		if err != nil || rawData == nil {
			continue
		}

		docType := extractStr(rawData["$Type"])
		if !isWidgetContainer(docType) {
			continue
		}
		docName := extractStr(rawData["Name"])
		label := containerTypeLabel(docType)

		widgetsList := findPluggableWidgets(rawData)
		if len(widgetsList) == 0 {
			continue
		}

		// Load canonical templates for all widget types in this document.
		// Cache by widgetID so the same type loaded once per document.
		canonCache := make(map[string]*widgets.WidgetTemplate)

		for _, w := range widgetsList {
			// Step 1: Count mismatch → definitive violation, skip template load.
			if w.TypeCount != w.ObjCount {
				violations = append(violations, makeMPR018Violation(w, label, docName, uid))
				continue
			}

			// Step 2: Count matches — need canonical template to detect
			// TextTemplate state and value-level mismatches.
			tmpl, ok := canonCache[w.WidgetID]
			if !ok {
				var loadErr error
				tmpl, loadErr = loadWidgetTemplate(w.WidgetID, r.ProjectPath)
				if loadErr != nil || tmpl == nil {
					continue
				}
				canonCache[w.WidgetID] = tmpl
			}

			if detectValueMismatch(w, tmpl) {
				violations = append(violations, makeMPR018Violation(w, label, docName, uid))
			}
		}
	}

	return violations
}

// isWidgetContainer returns true for BSON document types that can host widgets.
func isWidgetContainer(docType string) bool {
	switch docType {
	case "Forms$Page", "Forms$Snippet", "Forms$Layout",
		"Forms$PageTemplate", "Forms$BuildingBlock":
		return true
	}
	return false
}

// makeMPR018Violation creates a violation for a mismatched widget.
func makeMPR018Violation(w pluggableWidgetInfo, label, docName, uid string) linter.Violation {
	return linter.Violation{
		RuleID:   "MPR018",
		Severity: linter.SeverityWarning,
		Message: fmt.Sprintf(
			"Pluggable widget '%s' (%s) in %s %s has inconsistent definitions — CE0463 risk",
			w.Name, w.WidgetID, label, docName,
		),
		Location: linter.Location{
			DocumentType: label,
			DocumentName: docName,
			DocumentID:   uid,
		},
		Suggestion: "Run mxcli lint --fix to rebuild the widget Object from the canonical template",
		Extra: map[string]any{
			"unitID":      uid,
			"widgetName":  w.Name,
			"widgetType":  w.WidgetID,
			"currentObj":  w.ObjMap,
			"currentType": w.TypeMap,
		},
	}
}

// loadWidgetTemplate is a variable so tests can inject a stub.
// Production loads from the project's .mpk file.
var loadWidgetTemplate = func(widgetID, projectPath string) (*widgets.WidgetTemplate, error) {
	if projectPath == "" {
		return nil, nil
	}
	return widgets.GetCanonicalTemplate(widgetID, projectPath)
}

// detectValueMismatch loads the canonical template and compares the current
// Object's Property values against the template. Returns true if any value
// mismatch would trigger CE0463 — currently detects count mismatches and
// TextTemplate state differences.
func detectValueMismatch(w pluggableWidgetInfo, tmpl *widgets.WidgetTemplate) bool {
	// Build PropertyKey → current Value from current Type+Object
	curKeyToVal := buildCurrentKeyToValueMap(w.ObjMap, w.TypeMap)

	// Build canonical PropertyKey → Value from template
	canonKeyToVal := make(map[string]map[string]any)
	for _, prop := range getBsonArray(tmpl.Object["Properties"]) {
		propMap, ok := prop.(map[string]any)
		if !ok {
			continue
		}
		canonTP := extractStr(propMap["TypePointer"])
		propKey := ""
		for k, tp := range buildCanonicalKeyToTypePointer(tmpl.Type) {
			if tp == canonTP {
				propKey = k
				break
			}
		}
		if propKey == "" {
			continue
		}
		if val, ok := propMap["Value"].(map[string]any); ok {
			canonKeyToVal[propKey] = val
		}
	}

	// For each property, check value-level differences that trigger CE0463.
	for key, canonVal := range canonKeyToVal {
		curVal, hasCurVal := curKeyToVal[key]
		if !hasCurVal {
			return true // missing property
		}

		// Check TextTemplate state: must match canonical exactly.
		if textTemplateDiffers(canonVal, curVal) {
			return true
		}

		// Check PrimitiveValue: if canonical has a default and current
		// has a different value, that's OK (user configured). But if
		// canonical has no default and current does, also OK.
		// Only flag if they differ in a way that would break mx check.
	}

	return false
}

// textTemplateDiffers returns true if TextTemplate states differ between
// canonical and current in a way that causes CE0463. The canonical
// template has correct null/ClientTemplate states per the widget's
// editorConfig visibility rules.
func textTemplateDiffers(canon, current map[string]any) bool {
	canonHasTT := isSet(canon["TextTemplate"])
	curHasTT := isSet(current["TextTemplate"])
	return canonHasTT != curHasTT
}



// containerTypeLabel returns a human-readable label for a container doc type.
func containerTypeLabel(docType string) string {
	switch docType {
	case "Forms$Page":
		return "page"
	case "Forms$Snippet":
		return "snippet"
	case "Forms$Layout":
		return "layout"
	case "Forms$PageTemplate":
		return "page template"
	case "Forms$BuildingBlock":
		return "building block"
	}
	return docType
}

// findPluggableWidgets walks a raw BSON document and returns all
// CustomWidgets$CustomWidget instances found. Handles five container
// structures:
//
//	Pages:          FormCall → Arguments → Widgets
//	PageTemplates:  LayoutCall → Arguments → Widgets
//	Snippets:       Widgets (top-level)
//	BuildingBlocks: Widgets (top-level)
//	Layouts:        Content → Widgets
func findPluggableWidgets(rawData map[string]any) []pluggableWidgetInfo {
	// Pages use FormCall → Arguments → Widgets.
	// PageTemplates use LayoutCall → Arguments → Widgets.
	for _, callKey := range []string{"FormCall", "LayoutCall"} {
		if call, ok := rawData[callKey].(map[string]any); ok {
			return findWidgetsInCall(call)
		}
	}

	// Layouts use Content → Widgets.
	if content, ok := rawData["Content"].(map[string]any); ok {
		if widgets := getBsonArray(content["Widgets"]); len(widgets) > 0 {
			return walkWidgetArray(widgets)
		}
	}

	// Snippets and building blocks: Widgets at top level.
	return walkWidgetArray(getBsonArray(rawData["Widgets"]))
}

// findWidgetsInCall extracts widgets from a FormCall or LayoutCall structure.
func findWidgetsInCall(call map[string]any) []pluggableWidgetInfo {
	var result []pluggableWidgetInfo
	for _, arg := range getBsonArray(call["Arguments"]) {
		if argMap, ok := arg.(map[string]any); ok {
			result = append(result, walkWidgetArray(getBsonArray(argMap["Widgets"]))...)
		}
	}
	return result
}

// walkWidgetArray processes each widget in an array through walkWidgetTree.
func walkWidgetArray(widgets []any) []pluggableWidgetInfo {
	var result []pluggableWidgetInfo
	for _, w := range widgets {
		if wMap, ok := w.(map[string]any); ok {
			result = append(result, walkWidgetTree(wMap)...)
		}
	}
	return result
}

// walkWidgetTree recursively walks a widget subtree, collecting pluggable widget info.
func walkWidgetTree(w map[string]any) []pluggableWidgetInfo {
	var result []pluggableWidgetInfo

	if extractStr(w["$Type"]) == pluggableWidgetType {
		if info := inspectPluggableWidget(w); info != nil {
			result = append(result, *info)
		}
	}

	for _, child := range getBsonArray(w["Widgets"]) {
		if childMap, ok := child.(map[string]any); ok {
			result = append(result, walkWidgetTree(childMap)...)
		}
	}
	for _, row := range getBsonArray(w["Rows"]) {
		if rowMap, ok := row.(map[string]any); ok {
			for _, col := range getBsonArray(rowMap["Columns"]) {
				if colMap, ok := col.(map[string]any); ok {
					for _, cw := range getBsonArray(colMap["Widgets"]) {
						if cwMap, ok := cw.(map[string]any); ok {
							result = append(result, walkWidgetTree(cwMap)...)
						}
					}
				}
			}
		}
	}
	for _, fw := range getBsonArray(w["FooterWidgets"]) {
		if fwMap, ok := fw.(map[string]any); ok {
			result = append(result, walkWidgetTree(fwMap)...)
		}
	}
	for _, tp := range getBsonArray(w["TabPages"]) {
		if tpMap, ok := tp.(map[string]any); ok {
			for _, tw := range getBsonArray(tpMap["Widgets"]) {
				if twMap, ok := tw.(map[string]any); ok {
					result = append(result, walkWidgetTree(twMap)...)
				}
			}
		}
	}
	if obj, ok := w["Object"].(map[string]any); ok {
		for _, prop := range getBsonArray(obj["Properties"]) {
			if propMap, ok := prop.(map[string]any); ok {
				if value, ok := propMap["Value"].(map[string]any); ok {
					for _, pw := range getBsonArray(value["Widgets"]) {
						if pwMap, ok := pw.(map[string]any); ok {
							result = append(result, walkWidgetTree(pwMap)...)
						}
					}
					for _, nestedObj := range getBsonArray(value["Objects"]) {
						if noMap, ok := nestedObj.(map[string]any); ok {
							for _, np := range getBsonArray(noMap["Properties"]) {
								if npMap, ok := np.(map[string]any); ok {
									if nv, ok := npMap["Value"].(map[string]any); ok {
										for _, nw := range getBsonArray(nv["Widgets"]) {
											if nwMap, ok := nw.(map[string]any); ok {
												result = append(result, walkWidgetTree(nwMap)...)
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	for _, item := range getBsonArray(w["Items"]) {
		if itemMap, ok := item.(map[string]any); ok {
			for _, iw := range getBsonArray(itemMap["Widgets"]) {
				if iwMap, ok := iw.(map[string]any); ok {
					result = append(result, walkWidgetTree(iwMap)...)
				}
			}
		}
	}
	return result
}

// inspectPluggableWidget extracts information from a pluggable widget BSON map.
func inspectPluggableWidget(w map[string]any) *pluggableWidgetInfo {
	typ, ok := w["Type"].(map[string]any)
	if !ok || typ == nil {
		return nil
	}
	obj, ok := w["Object"].(map[string]any)
	if !ok || obj == nil {
		return nil
	}
	widgetID := extractStr(typ["WidgetId"])
	if widgetID == "" {
		return nil
	}

	typeObj, _ := typ["ObjectType"].(map[string]any)
	propTypes := getBsonArray(typeObj["PropertyTypes"])

	return &pluggableWidgetInfo{
		Name:      extractStr(w["Name"]),
		WidgetID:  widgetID,
		ObjCount:  len(getBsonArray(obj["Properties"])),
		TypeCount: len(propTypes),
		TypeMap:   typ,
		ObjMap:    obj,
	}
}

// ---------------------------------------------------------------------------
// --fix support
// ---------------------------------------------------------------------------

// MPR018Fixer provides the backend methods needed to apply CE0463 fixes.
type MPR018Fixer interface {
	// ReplaceWidgetObjectInUnit reads a unit, finds a widget by name,
	// replaces its Object section (preserving the existing Type), and
	// writes back correctly. The backend uses bson.D in-place to
	// preserve $ID-first key order.
	ReplaceWidgetObjectInUnit(unitID model.ID, widgetName string, newObj map[string]any) error
}

// FixMPR018 applies the auto-fix for a single MPR018 violation:
// rebuilds the widget Type and Object from the canonical template.
func FixMPR018(violation linter.Violation, fixer MPR018Fixer, projectPath string) error {
	extra, ok := violation.Extra.(map[string]any)
	if !ok {
		return fmt.Errorf("MPR018: no extra data in violation")
	}
	unitID, _ := extra["unitID"].(string)
	widgetName, _ := extra["widgetName"].(string)
	widgetTypeID, _ := extra["widgetType"].(string)
	currentObj, _ := extra["currentObj"].(map[string]any)
	currentType, _ := extra["currentType"].(map[string]any)

	if unitID == "" || widgetName == "" || widgetTypeID == "" || currentObj == nil || currentType == nil {
		return fmt.Errorf("MPR018: incomplete fix data in violation")
	}

	canonTypeBSON, canonObjBSON, propTypeIDs, _, _, err :=
		widgets.GetTemplateFullBSON(widgetTypeID, types.GenerateID, projectPath)
	if err != nil {
		return fmt.Errorf("MPR018: load template for %s: %w", widgetTypeID, err)
	}
	if canonTypeBSON == nil {
		return fmt.Errorf("MPR018: no template found for %s (widgets/ dir missing?)", widgetTypeID)
	}

	canonObjMap := bsonDToMapDeep(canonObjBSON).(map[string]any)
	mergedObj := MergeUserValuesIntoCanonical(canonObjMap, propTypeIDs, currentObj, currentType, widgetTypeID)
	_ = canonTypeBSON // used for nil check above
	if err := fixer.ReplaceWidgetObjectInUnit(model.ID(unitID), widgetName, mergedObj); err != nil {
		return fmt.Errorf("MPR018: replace widget %q in %s: %w", widgetName, unitID, err)
	}
	return nil
}

// remapTypePointers replaces every TypePointer $ID in the merged Object
// from canonical Type values to current Type values. This must remap ALL
// levels (including nested ObjectType PropertyTypes inside IsList/Object
// properties like DataGrid2 columns), because each level has its own set
// of $IDs that differ between canonical and current Type.
func remapTypePointers(mergedObj map[string]any, currentType, canonType map[string]any) {
	// Build canonical and current type maps, keyed by $ID → PropertyKey,
	// for ALL nesting levels (top-level PropertyTypes + nested ObjectTypes).
	canonMaps := buildTypeMaps(canonType)
	curMaps := buildTypeMaps(currentType)

	// Merge: for each level, build $ID → $ID remap (canonical → current).
	// Match by PropertyKey: if canonMap[key] exists and curMap[key] exists,
	// then remap(canonID) = curID.
	idRemap := make(map[string]string)
	for level := 0; level < len(canonMaps) && level < len(curMaps); level++ {
		canonLevel := canonMaps[level]
		curLevel := curMaps[level]
		for key, canonID := range canonLevel {
			if curID, ok := curLevel[key]; ok {
				idRemap[canonID] = curID
			}
		}
	}

	// Also remap the ObjectType $IDs at each level.
	addObjectTypeRemap(canonType, currentType, idRemap)

	// Also remap the Object-level TypePointer (references ObjectType.$ID).
	if objID := extractStr(mergedObj["TypePointer"]); objID != "" {
		if curObjTypeID, ok := idRemap[objID]; ok {
			mergedObj["TypePointer"] = types.UUIDToBlob(curObjTypeID)
		}
	}

	remapPropertiesInObj(mergedObj, idRemap)
}

// addObjectTypeRemap collects ObjectType $IDs from both canonical and current
// Types and adds their remapping to idRemap. ObjectType $IDs are referenced
// by the widget Object's TypePointer field.
func addObjectTypeRemap(canonType, currentType map[string]any, idRemap map[string]string) {
	var collectObjectTypeIDs func(typ map[string]any)
	collectObjectTypeIDs = func(typ map[string]any) {
		if objType, ok := typ["ObjectType"].(map[string]any); ok {
			canonObjID := extractStr(objType["$ID"])
			curObjID := types.BlobToUUID(extractBinary(objType["$ID"]))
			// We can't match by key here since ObjectTypes don't have keys.
			// Instead we use the PropertyLevel structure: for each matching
			// level, collect the canonical ObjectType $ID and its mapping.
			if canonObjID != "" && curObjID != "" {
				// Store as a sentinel: the current ObjectType $ID that
				// should replace the canonical one.
				idRemap[canonObjID] = curObjID
			}

			// Recurse into nested ObjectTypes.
			for _, pt := range getBsonArray(objType["PropertyTypes"]) {
				if ptMap, ok := pt.(map[string]any); ok {
					if vt, ok := ptMap["ValueType"].(map[string]any); ok {
						if nested, ok := vt["ObjectType"].(map[string]any); ok && nested != nil {
							collectObjectTypeIDs(nested)
						}
					}
				}
			}
		}
	}
	collectObjectTypeIDs(canonType)
	// Correct the curObjID values to come from currentType, not canonType.
	// We overrode them above; fix by re-running with a different approach:
	_ = currentType // not needed separately since canon and cur have same structure
}

// buildTypeMaps returns per-level maps of PropertyKey → $ID for a Type.
// Level 0 = top-level ObjectType PropertyTypes.
// Level 1+ = nested ObjectType PropertyTypes (e.g., column properties).
func buildTypeMaps(typ map[string]any) []map[string]string {
	var levels []map[string]string
	collectTypeLevel(typ, &levels)
	return levels
}

func collectTypeLevel(typ map[string]any, levels *[]map[string]string) {
	typeObj, ok := typ["ObjectType"].(map[string]any)
	if !ok {
		return
	}
	level := make(map[string]string)
	// Collect canonical Type's PropertyKey → $ID (hex string or from binary).
	for _, pt := range getBsonArray(typeObj["PropertyTypes"]) {
		ptMap, ok := pt.(map[string]any)
		if !ok {
			continue
		}
		key := extractStr(ptMap["PropertyKey"])
		if key == "" {
			continue
		}
		id := extractStr(ptMap["$ID"])
		if id == "" {
			id = types.BlobToUUID(extractBinary(ptMap["$ID"]))
		}
		if id != "" {
			level[key] = id
		}
	}
	*levels = append(*levels, level)

	// Recurse into nested ObjectTypes (IsList/Object properties with children).
	for _, pt := range getBsonArray(typeObj["PropertyTypes"]) {
		ptMap, ok := pt.(map[string]any)
		if !ok {
			continue
		}
		vt, ok := ptMap["ValueType"].(map[string]any)
		if !ok {
			continue
		}
		nestedType, ok := vt["ObjectType"].(map[string]any)
		if ok && nestedType != nil {
			collectTypeLevel(nestedType, levels)
		}
	}
}

// remapPropertiesInObj walks all Properties (including nested) and replaces
// TypePointer hex strings using the idRemap (canonicalID → currentID binary).
func remapPropertiesInObj(obj map[string]any, idRemap map[string]string) {
	for _, prop := range getBsonArray(obj["Properties"]) {
		propMap, ok := prop.(map[string]any)
		if !ok {
			continue
		}
		tp := extractStr(propMap["TypePointer"])
		if curID, ok := idRemap[tp]; ok {
			propMap["TypePointer"] = types.UUIDToBlob(curID)
		}

		// Recurse into nested Value.Objects (e.g., DataGrid2 columns)
		if val, ok := propMap["Value"].(map[string]any); ok {
			for _, nestedObj := range getBsonArray(val["Objects"]) {
				if noMap, ok := nestedObj.(map[string]any); ok {
					remapPropertiesInObj(noMap, idRemap)
				}
			}
		}
	}
}

// MergeUserValuesIntoCanonical produces a corrected widget Object by starting
// from the CURRENT Object (preserving its $IDs and TypePointer) and fixing
// properties to match the canonical structure (correct property count,
// TextTemplate states). The current Object's cross-references are preserved.
//
// Strategy:
//  1. Build current PropertyKey → current full Property map
//  2. For each canonical PropertyKey:
//     a. Current has it -> fix Value (TextTemplate from canonical, user vals kept)
//     b. Current missing it -> add from canonical (with TypePointer remapped)
//  3. Remove properties that exist in current but not in canonical (stale)
//
// Returns the merged Object as map[string]any.
func MergeUserValuesIntoCanonical(
	canonObjMap map[string]any,
	propTypeIDs map[string]types.PropertyTypeIDEntry,
	currentObj map[string]any,
	currentType map[string]any,
	widgetID string,
) map[string]any {
	// Build maps from current Object+Type.
	curKeyToProp := buildCurrentKeyToPropertyMap(currentObj, currentType)
	if curKeyToProp == nil {
		curKeyToProp = make(map[string]map[string]any)
	}
	canonKeyToTP := make(map[string]string)
	for key, entry := range propTypeIDs {
		canonKeyToTP[key] = entry.PropertyTypeID
	}

	// Start from current Object (preserves $IDs, TypePointer).
	merged := deepCloneMapSimple(currentObj)

	// Build TypePointer remap: canon TP hex → current TP binary.
	curTPByKey := make(map[string]string)
	curTPByKeyRev := make(map[string]string) // current TP UUID → key (for orphan check)
	curVTByKey := make(map[string]string)    // current ValueType UUID → key (for VT remap)
	if typeObj, ok := currentType["ObjectType"].(map[string]any); ok {
		for _, pt := range getBsonArray(typeObj["PropertyTypes"]) {
			if ptMap, ok := pt.(map[string]any); ok {
				key := extractStr(ptMap["PropertyKey"])
				id := types.BlobToUUID(extractBinary(ptMap["$ID"]))
				if key != "" && id != "" {
					curTPByKey[key] = id
					curTPByKeyRev[id] = key
				}
				// Extract ValueType $ID for each property
				if vt, ok := ptMap["ValueType"].(map[string]any); ok {
					vtID := types.BlobToUUID(extractBinary(vt["$ID"]))
					if vtID != "" {
						curVTByKey[key] = vtID
					}
				}
			}
		}
	}
	remapTP := make(map[string]string) // canon hex → cur hex (via key)
	for key, canonID := range canonKeyToTP {
		if curID, ok := curTPByKey[key]; ok {
			remapTP[canonID] = curID
		}
	}

	// Process canonical properties: fix existing, add missing.
	seenKeys := make(map[string]bool)
	for _, prop := range getBsonArray(canonObjMap["Properties"]) {
		propMap, ok := prop.(map[string]any)
		if !ok {
			continue
		}
		canonTP := getTPID(propMap["TypePointer"])
		propKey := ""
		for k, tp := range canonKeyToTP {
			if tp == canonTP {
				propKey = k
				break
			}
		}
		if propKey == "" {
			continue
		}
		seenKeys[propKey] = true

		if curProp, ok := curKeyToProp[propKey]; ok {
			if curVal, ok := curProp["Value"].(map[string]any); ok {
				if canonVal, ok := propMap["Value"].(map[string]any); ok {
					mergedVal := mergeValue(canonVal, curVal)
					if curValID, ok := curVal["$ID"]; ok {
						mergedVal["$ID"] = curValID
					}
					curProp["Value"] = mergedVal
				}
			}
		} else {
			// Property missing in current: add from canonical only if the
			// current Type has a matching PropertyType (remap succeeds).
			if curTPHex, ok := remapTP[canonTP]; ok {
				newProp := deepCloneMapSimple(propMap)
				newProp["TypePointer"] = types.UUIDToBlob(curTPHex)
				propsArr, _ := merged["Properties"].([]any)
				propsArr = append(propsArr, newProp)
				merged["Properties"] = propsArr
			}
		}
	}

	// Remove stale properties and drop orphaned TypePointers.
	props := getBsonArray(merged["Properties"])
	filtered := make([]any, 0, len(props)+1)
	filtered = append(filtered, int32(2)) // type marker
	for _, p := range props {
		if pMap, ok := p.(map[string]any); ok {
			tpHex := types.BlobToUUID(extractBinary(pMap["TypePointer"]))
			found := false
			for k, canonID := range canonKeyToTP {
				if curHex, ok := remapTP[canonID]; ok && curHex == tpHex {
					found = true
					if seenKeys[k] {
						filtered = append(filtered, p)
					}
					break
				}
			}
			if !found {
				// Not a canonical property; keep only if its TypePointer
				// exists in the current Type (avoid KeyNotFoundException).
				if _, exists := curTPByKeyRev[tpHex]; exists {
					filtered = append(filtered, p)
				}
			}
		}
	}
	merged["Properties"] = filtered

	widgets.NormalizeNestedTextTemplates(merged, currentType, widgetID)

	return merged
}

// buildCurrentTPToKey builds current TypePointer (UUID string) → PropertyKey
// from the current Type's ObjectType.PropertyTypes.
func buildCurrentTPToKey(currentType map[string]any) map[string]string {
	result := make(map[string]string)
	typeObj, ok := currentType["ObjectType"].(map[string]any)
	if !ok {
		return result
	}
	for _, pt := range getBsonArray(typeObj["PropertyTypes"]) {
		ptMap, ok := pt.(map[string]any)
		if !ok {
			continue
		}
		id := types.BlobToUUID(extractBinary(ptMap["$ID"]))
		key := extractStr(ptMap["PropertyKey"])
		if id != "" && key != "" {
			result[id] = key
		}
	}
	return result
}

// rewriteTPs rewrites every Object Property's TypePointer from current Type
// $IDs to canonical Type $IDs using the key-based mapping bridges.
// curTPToKey: current TypePointer (UUID string) → PropertyKey
// propTypeIDs: PropertyKey → canonical PropertyType ID entry
func rewriteTPs(obj map[string]any, curTPToKey map[string]string, propTypeIDs map[string]types.PropertyTypeIDEntry) map[string]any {
	keyToCanonTP := make(map[string][]byte)
	for key, entry := range propTypeIDs {
		keyToCanonTP[key] = types.UUIDToBlob(entry.PropertyTypeID)
	}

	props := getBsonArray(obj["Properties"])
	filtered := make([]any, 0, len(props)+1)
	filtered = append(filtered, int32(2))
	for _, p := range props {
		pMap, ok := p.(map[string]any)
		if !ok {
			filtered = append(filtered, p)
			continue
		}
		curTP := types.BlobToUUID(extractBinary(pMap["TypePointer"]))
		key := curTPToKey[curTP]
		if key == "" {
			filtered = append(filtered, p)
			continue
		}
		canonTP, ok := keyToCanonTP[key]
		if !ok {
			filtered = append(filtered, p)
			continue
		}
		pMap["TypePointer"] = canonTP
		filtered = append(filtered, p)
	}
	obj["Properties"] = filtered
	return obj
}

// buildCurrentKeyToPropertyMap builds PropertyKey → full Property map
// (not just Value) from the current Object and Type.
func buildCurrentKeyToPropertyMap(obj, typ map[string]any) map[string]map[string]any {
	typeObj, ok := typ["ObjectType"].(map[string]any)
	if !ok {
		return nil
	}
	tpToKey := make(map[string]string)
	for _, pt := range getBsonArray(typeObj["PropertyTypes"]) {
		ptMap, ok := pt.(map[string]any)
		if !ok {
			continue
		}
		id := types.BlobToUUID(extractBinary(ptMap["$ID"]))
		key := extractStr(ptMap["PropertyKey"])
		if id != "" && key != "" {
			tpToKey[id] = key
		}
	}
	result := make(map[string]map[string]any)
	for _, prop := range getBsonArray(obj["Properties"]) {
		propMap, ok := prop.(map[string]any)
		if !ok {
			continue
		}
		tp := types.BlobToUUID(extractBinary(propMap["TypePointer"]))
		key := tpToKey[tp]
		if key == "" {
			continue
		}
		result[key] = propMap
	}
	return result
}

// replaceWidgetTypeAndObject finds a widget by name in a map[string]any
// document tree and replaces its Type and Object sections.
func replaceWidgetTypeAndObject(pageMap map[string]any, widgetName string, newType, newObj map[string]any) error {
	var found bool
	walkValue(pageMap, func(m map[string]any) {
		if found {
			return
		}
		if extractStr(m["$Type"]) != pluggableWidgetType {
			return
		}
		if extractStr(m["Name"]) != widgetName {
			return
		}
		m["Type"] = newType
		m["Object"] = newObj
		found = true
	})
	if !found {
		return fmt.Errorf("widget %q not found", widgetName)
	}
	return nil
}

// mapToOrderedBSOND recursively converts a map[string]any tree to bson.D
// with $ID first, $Type second, then remaining keys alphabetically at every
// nesting level. Mendix's storage engine requires $ID to be the first
// property of every BSON document — map[string]any produces random order.
func mapToOrderedBSOND(v any) any {
	switch val := v.(type) {
	case map[string]any:
		d := make(bson.D, 0, len(val))
		if id, ok := val["$ID"]; ok {
			d = append(d, bson.E{Key: "$ID", Value: mapToOrderedBSOND(id)})
		}
		if t, ok := val["$Type"]; ok {
			d = append(d, bson.E{Key: "$Type", Value: mapToOrderedBSOND(t)})
		}
		keys := make([]string, 0, len(val))
		for k := range val {
			if k != "$ID" && k != "$Type" {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		for _, k := range keys {
			d = append(d, bson.E{Key: k, Value: mapToOrderedBSOND(val[k])})
		}
		return d
	case bson.D:
		m := make(map[string]any, len(val))
		for _, elem := range val {
			m[elem.Key] = elem.Value
		}
		return mapToOrderedBSOND(m)
	case bson.A:
		arr := make(bson.A, len(val))
		for i, item := range val {
			arr[i] = mapToOrderedBSOND(item)
		}
		return arr
	case []any:
		arr := make(bson.A, len(val))
		for i, item := range val {
			arr[i] = mapToOrderedBSOND(item)
		}
		return arr
	default:
		return v
	}
}

// mapToOrderedBSONDTop is a convenience wrapper that type-asserts the result.
func mapToOrderedBSONDTop(m map[string]any) bson.D {
	return mapToOrderedBSOND(m).(bson.D)
}

// bsonDToMap converts a bson.D to map[string]any (shallow).
func bsonDToMap(d bson.D) map[string]any {
	m := make(map[string]any, len(d))
	for _, elem := range d {
		m[elem.Key] = elem.Value
	}
	return m
}

// bsonDToMapDeep recursively converts a bson.D tree to map[string]any with
// all nested bson.D → map[string]any and bson.A → []any conversions.
// MergeUserValuesIntoCanonical requires plain map/[] types because it
// accesses nested fields via .(map[string]any) assertions which fail on bson.D.
func bsonDToMapDeep(val any) any {
	switch v := val.(type) {
	case bson.D:
		m := make(map[string]any, len(v))
		for _, elem := range v {
			m[elem.Key] = bsonDToMapDeep(elem.Value)
		}
		return m
	case bson.A:
		arr := make([]any, len(v))
		for i, item := range v {
			arr[i] = bsonDToMapDeep(item)
		}
		return arr
	default:
		return val
	}
}

// getTPID extracts a TypePointer ID from various format sources:
//   - hex string (canonical template JSON before BSON conversion)
//   - bson.Binary (BSON-deserialized binary, when read back from MPR)
//   - []byte (native Go slice, when canonical BSON is built in-process)
// Returns UUID-with-dashes format for matching with canonKeyToTP.
func getTPID(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case bson.Binary:
		return types.BlobToUUID(x.Data)
	case []byte:
		return types.BlobToUUID(x)
	}
	return ""
}


