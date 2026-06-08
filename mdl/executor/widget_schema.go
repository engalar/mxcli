// mdl/executor/widget_schema.go
package executor

import (
	"fmt"

	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// ─── Types ───────────────────────────────────────────────────────────────────

// SchemaEntry describes one property type from a widget's Type BSON.
type SchemaEntry struct {
	Key          string // PropertyKey (e.g. "source", "attribute")
	DefaultValue string // from ValueType.DefaultValue
	ValueType    string // "Boolean" | "Integer" | "Enumeration" | "String" | "Expression" | "Object" | ...
}

// SchemaMap maps TypePointer ID strings to their schema entry.
type SchemaMap map[string]SchemaEntry

// PropertyValue is one entry from Object.Properties[].
type PropertyValue struct {
	TypePointerID  string
	Key            string // resolved from schema
	PrimitiveValue string
	ValueType      string // mirrors SchemaEntry.ValueType
}

// ─── normalizeArray ──────────────────────────────────────────────────────────

// normalizeArray returns BSON array elements as []any. It handles both the real
// BSON array shape (a leading int32/int version prefix, via getBsonArrayElements)
// and the plain []map[string]any / []any shapes used in unit tests.
func normalizeArray(v any) []any {
	switch arr := v.(type) {
	case []map[string]any:
		out := make([]any, len(arr))
		for i := range arr {
			out[i] = arr[i]
		}
		return out
	case []any:
		// Could be a versioned BSON array; delegate so the version prefix is
		// stripped. getBsonArrayElements is a no-op when there is no prefix.
		return getBsonArrayElements(arr)
	default:
		return getBsonArrayElements(v)
	}
}

// asMap normalizes a raw BSON document value into a map[string]any.
func asMap(v any) (map[string]any, bool) {
	if m, ok := v.(map[string]any); ok {
		return m, true
	}
	return genPg.BSONDocumentMap(v)
}

// ─── buildSchemaMap ───────────────────────────────────────────────────────────

// buildSchemaMap reads Type.ObjectType.PropertyTypes from a CustomWidget raw map
// and returns a map from TypePointer $ID to SchemaEntry.
func buildSchemaMap(raw map[string]any) SchemaMap {
	// nolint:describe-raw-bson — pluggable widgets are unknown types with no gen
	// struct; reading the embedded Type schema by raw key is the intended design
	// (see describe-system-solid-refactor-design.md "Schema Self-Introspection").
	result := make(SchemaMap)
	typeDoc, ok := asMap(raw["Type"]) // nolint:describe-raw-bson
	if !ok {
		return result
	}
	objType, ok := asMap(typeDoc["ObjectType"]) // nolint:describe-raw-bson
	if !ok {
		return result
	}
	propTypes := normalizeArray(objType["PropertyTypes"]) // nolint:describe-raw-bson
	for _, pt := range propTypes {
		ptMap, ok := asMap(pt)
		if !ok {
			continue
		}
		id := extractBinaryID(ptMap["$ID"])
		key, _ := ptMap["PropertyKey"].(string) // nolint:describe-raw-bson
		var defaultVal, valueType string
		if vt, ok := asMap(ptMap["ValueType"]); ok { // nolint:describe-raw-bson
			defaultVal, _ = vt["DefaultValue"].(string) // nolint:describe-raw-bson
			valueType, _ = vt["Type"].(string)          // nolint:describe-raw-bson
		}
		if id != "" && key != "" {
			result[id] = SchemaEntry{Key: key, DefaultValue: defaultVal, ValueType: valueType}
		}
	}
	return result
}

// ─── readProperties ───────────────────────────────────────────────────────────

// readProperties reads Object.Properties[] from a CustomWidget raw map and
// returns all entries as PropertyValues. Schema is used later to resolve Key names.
func readProperties(raw map[string]any) []PropertyValue {
	// nolint:describe-raw-bson — see buildSchemaMap; pluggable widget properties
	// have no gen accessor, so raw Object.Properties access is by design.
	objDoc, ok := asMap(raw["Object"]) // nolint:describe-raw-bson
	if !ok {
		return nil
	}
	propsRaw := normalizeArray(objDoc["Properties"]) // nolint:describe-raw-bson
	result := make([]PropertyValue, 0, len(propsRaw))
	for _, pr := range propsRaw {
		prMap, ok := asMap(pr)
		if !ok {
			continue
		}
		ptrID := extractBinaryID(prMap["TypePointer"]) // nolint:describe-raw-bson
		if ptrID == "" {
			continue
		}
		primVal := ""
		if val, ok := asMap(prMap["Value"]); ok { // nolint:describe-raw-bson
			primVal, _ = val["PrimitiveValue"].(string) // nolint:describe-raw-bson
		}
		result = append(result, PropertyValue{
			TypePointerID:  ptrID,
			PrimitiveValue: primVal,
		})
	}
	return result
}

// ─── filterDefaults ───────────────────────────────────────────────────────────

// filterDefaults returns only those PropertyValues whose PrimitiveValue differs
// from the default declared in the schema. It also populates Key and ValueType
// on each returned entry.
func filterDefaults(props []PropertyValue, schema SchemaMap) []PropertyValue {
	var result []PropertyValue
	for _, p := range props {
		entry, ok := schema[p.TypePointerID]
		if !ok {
			continue // skip properties not in schema
		}
		p.Key = entry.Key
		p.ValueType = entry.ValueType

		// Determine effective default
		def := entry.DefaultValue
		if def == "" {
			switch entry.ValueType {
			case "Boolean":
				def = "false"
			case "Integer":
				def = "0"
			}
		}
		if p.PrimitiveValue == def {
			continue // matches default, skip
		}
		result = append(result, p)
	}
	return result
}

// ─── formatPropertyValue ─────────────────────────────────────────────────────

// formatPropertyValue formats a PropertyValue's primitive as an MDL value string.
func formatPropertyValue(p PropertyValue) string {
	switch p.ValueType {
	case "Boolean", "Integer":
		return p.PrimitiveValue
	default:
		return fmt.Sprintf("'%s'", p.PrimitiveValue)
	}
}

// ─── extractWidgetTypeID ─────────────────────────────────────────────────────

// extractWidgetTypeID reads the widget ID from Type.WidgetId in a
// CustomWidgets$CustomWidget raw map. Returns "" if not present.
func extractWidgetTypeID(raw map[string]any) string {
	typeDoc, ok := asMap(raw["Type"]) // nolint:describe-raw-bson — pluggable widget ID has no gen accessor
	if !ok {
		return ""
	}
	id, _ := typeDoc["WidgetId"].(string) // nolint:describe-raw-bson
	return id
}
