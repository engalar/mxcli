package rules

import (
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/modelsdk/widgets"
)


// RebuildWidgetObject rebuilds a widget's Type and Object sections from a
// canonical template augmented by the project's .mpk files, preserving
// user-configured values from the current widget instance.
//
// currentObj, currentType: widget Object and Type from page BSON (map[string]any
//   decoded via GetRawUnit — binary GUIDs in []byte, values as maps/nil)
// canonTemplate: augmented canonical *widgets.WidgetTemplate (JSON format with
//   hex string placeholder IDs, correct PropertyTypes and TextTemplate states)
//
// Returns merged Type and Object in JSON map format (hex string placeholder IDs)
// that can be fed into GetTemplateFullBSON's conversion pipeline.
func RebuildWidgetObject(
	currentObj, currentType map[string]any,
	canonTemplate *widgets.WidgetTemplate,
) (mergedType, mergedObj map[string]any) {
	// Build current PropertyKey → current WidgetValue map
	curKeyToVal := buildCurrentKeyToValueMap(currentObj, currentType)
	if curKeyToVal == nil {
		curKeyToVal = make(map[string]map[string]any)
	}

	// Deep-clone canonical template (don't mutate cached template)
	mergedType = deepCloneMapSimple(canonTemplate.Type)
	mergedObj = deepCloneMapSimple(canonTemplate.Object)

	// Build canonical PropertyKey → TypePointer $ID map
	canonKeyToTP := buildCanonicalKeyToTypePointer(mergedType)

	// For each canonical Object property, overlay user-configured values
	props := getBsonArray(mergedObj["Properties"])
	for _, prop := range props {
		propMap, ok := prop.(map[string]any)
		if !ok {
			continue
		}
		canonTP := extractStr(propMap["TypePointer"])

		propKey := lookupKeyByTP(canonKeyToTP, canonTP)
		if propKey == "" {
			continue
		}

		curVal, hasCurVal := curKeyToVal[propKey]
		if !hasCurVal {
			continue
		}

		canonVal, ok := propMap["Value"].(map[string]any)
		if !ok {
			continue
		}
		propMap["Value"] = mergeValue(canonVal, curVal)
	}

	return mergedType, mergedObj
}

// buildCurrentKeyToValueMap builds PropertyKey → current WidgetValue map
// by resolving each Object Property's TypePointer through the current Type's
// PropertyTypes to find its PropertyKey.
func buildCurrentKeyToValueMap(obj, typ map[string]any) map[string]map[string]any {
	typeObj, ok := typ["ObjectType"].(map[string]any)
	if !ok {
		return nil
	}

	// Step 1: Build binary TypePointer hex → PropertyKey from Type.PropertyTypes
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

	// Step 2: Build PropertyKey → WidgetValue from Object.Properties
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
		val, _ := propMap["Value"].(map[string]any)
		if val != nil {
			result[key] = val
		}
	}

	return result
}

// buildCanonicalKeyToTypePointer builds PropertyKey → TypePointer $ID from
// the canonical Type JSON (which has hex string placeholder IDs).
func buildCanonicalKeyToTypePointer(typ map[string]any) map[string]string {
	result := make(map[string]string)
	typeObj, ok := typ["ObjectType"].(map[string]any)
	if !ok {
		return result
	}
	for _, pt := range getBsonArray(typeObj["PropertyTypes"]) {
		ptMap, ok := pt.(map[string]any)
		if !ok {
			continue
		}
		key := extractStr(ptMap["PropertyKey"])
		id := extractStr(ptMap["$ID"])
		if key != "" && id != "" {
			result[key] = id
		}
	}
	return result
}

// lookupKeyByTP reverse-looks up a PropertyKey by its TypePointer $ID.
func lookupKeyByTP(keyToTP map[string]string, tp string) string {
	for key, id := range keyToTP {
		if id == tp {
			return key
		}
	}
	return ""
}

// mergeValue produces a canonical WidgetValue with user-configured values
// from current overlaid on top. TextTemplate and TypePointer always come
// from canonical (CE0463-critical). Other non-nil/non-empty fields
// come from current if present, else fall back to canonical.
func mergeValue(canon, current map[string]any) map[string]any {
	result := make(map[string]any, len(canon))

	// Copy everything from canonical as the base
	for k, v := range canon {
		result[k] = v
	}

	// Overlay user-configured fields from current, except:
	// - TextTemplate: always use canonical (correct null/ClientTemplate state)
	// - TypePointer: always use canonical (references canonical ValueType $ID)
	// - $ID: always use canonical (will be remapped by conversion pipeline)
	for k, v := range current {
		switch k {
		case "$ID", "TypePointer", "TextTemplate":
			// Skip — always use canonical
			continue
		case "AttributeRef", "DataSource", "Action", "EntityRef", "Icon",
			"SourceVariable", "TranslatableValue", "Widgets", "Objects":
			if isSet(v) {
				result[k] = v
			}
		case "PrimitiveValue", "Expression", "Selection", "XPathConstraint",
			"Microflow", "Nanoflow", "Form", "Image":
			if s, ok := v.(string); ok && s != "" {
				result[k] = s
			}
		default:
			if isSet(v) {
				result[k] = v
			}
		}
	}

	return result
}

// isSet returns true if v is non-nil and non-empty.
func isSet(v any) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case map[string]any:
		return len(val) > 0
	case []any:
		return len(val) > 0
	case string:
		return val != ""
	}
	return true
}

// extractBinary extracts []byte from an interface value.
// Handles both []byte (Go native) and bson.Binary (from BSON decode).
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

// deepCloneMapSimple deep-clones a map[string]any at one level.
// Nested maps and slices are shallow-copied only (sufficient for
// template cloning where we only modify top-level Properties/Value).
func deepCloneMapSimple(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}


