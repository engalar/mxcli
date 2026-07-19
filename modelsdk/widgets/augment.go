// SPDX-License-Identifier: Apache-2.0

package widgets

import (
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/mendixlabs/mxcli/modelsdk/widgets/mpk"
)

// AugmentTemplate modifies a template's Type and Object in-place to match an .mpk definition.
// It adds PropertyTypes (in Type) and Properties (in Object) for keys present in .mpk but
// missing from the template, and removes those present in the template but missing from .mpk.
// Only regular properties are compared (not system properties like Label, Visibility, Editability).
func AugmentTemplate(tmpl *WidgetTemplate, def *mpk.WidgetDefinition) error {
	if tmpl == nil || def == nil {
		return nil
	}

	// Get PropertyTypes array from Type.ObjectType.PropertyTypes
	objType, ok := getMapField(tmpl.Type, "ObjectType")
	if !ok {
		return nil
	}
	propTypes, ok := getArrayField(objType, "PropertyTypes")
	if !ok {
		return nil
	}

	// Get Properties array from Object.Properties
	objProps, ok := getArrayField(tmpl.Object, "Properties")
	if !ok {
		return nil
	}

	// Build set of existing template property keys (non-system only)
	templateKeys := make(map[string]bool)
	// Also build a map of XML type -> exemplar index for cloning
	typeExemplars := make(map[string]int) // ValueType.Type -> index in propTypes
	systemKeys := def.SystemPropertyKeys()

	for i, pt := range propTypes {
		ptMap, ok := pt.(map[string]any)
		if !ok {
			continue
		}
		key, _ := ptMap["PropertyKey"].(string)
		if key == "" {
			continue
		}
		// Skip system properties
		if systemKeys[key] {
			continue
		}
		templateKeys[key] = true

		// Record exemplar for this value type
		vt, ok := getMapField(ptMap, "ValueType")
		if ok {
			vtType, _ := vt["Type"].(string)
			if vtType != "" {
				if _, exists := typeExemplars[vtType]; !exists {
					typeExemplars[vtType] = i
				}
			}
		}
	}

	// Determine mpk property keys (regular only)
	mpkKeys := def.PropertyKeys()

	// Find missing keys (in mpk but not in template)
	var missing []mpk.PropertyDef
	for _, p := range def.Properties {
		if !templateKeys[p.Key] {
			missing = append(missing, p)
		}
	}

	// Find stale keys (in template but not in mpk, excluding system props)
	var stale []string
	for key := range templateKeys {
		if !mpkKeys[key] && !systemKeys[key] {
			stale = append(stale, key)
		}
	}

	// Check if nested augmentation is needed (skip early return if so)
	hasNestedChildren := false
	for _, p := range def.Properties {
		if len(p.Children) > 0 {
			hasNestedChildren = true
			break
		}
	}

	// Nothing to add/remove at top level, and no nested children to process
	if len(missing) == 0 && len(stale) == 0 && !hasNestedChildren {
		return nil
	}

	// Remove stale properties
	if len(stale) > 0 {
		staleSet := make(map[string]bool, len(stale))
		for _, key := range stale {
			staleSet[key] = true
		}
		propTypes, objProps = removeProperties(propTypes, objProps, staleSet)
	}

	// Create a cloner for property pair deep-cloning
	cloner := defaultCloner()

	// Add missing properties
	for _, p := range missing {
		bsonType := xmlTypeToBSONType(p.Type)
		if bsonType == "" {
			continue // Unknown type, skip
		}

		// Find an exemplar of the same type to clone
		exemplarIdx, hasExemplar := typeExemplars[bsonType]
		var newPropType, newProp map[string]any
		if hasExemplar {
			var err error
			newPropType, newProp, err = cloner.ClonePair(propTypes, objProps, exemplarIdx, p)
			if err != nil {
				return fmt.Errorf("augment %s: %w", tmpl.WidgetID, err)
			}
		}
		// Fall back to createPropertyPair if cloning failed (no exemplar or no matching property)
		if newPropType == nil || newProp == nil {
			newPropType, newProp = createPropertyPair(p, bsonType)
		}

		if newPropType != nil {
			propTypes = append(propTypes, newPropType)
		}
		if newProp != nil {
			objProps = append(objProps, newProp)
		}
	}

	// Write back top-level
	setArrayField(objType, "PropertyTypes", propTypes)
	setArrayField(tmpl.Object, "Properties", objProps)

	// Augment nested ObjectType properties (e.g., DataGrid2 column properties).
	// Top-level augmentation syncs the property list, but nested ObjectTypes inside
	// IsList Object properties also need syncing when the .mpk version differs
	// from the template version.
	for _, mpkProp := range def.Properties {
		if len(mpkProp.Children) == 0 {
			continue
		}
		if err := augmentNestedObjectType(propTypes, objProps, mpkProp); err != nil {
			return fmt.Errorf("augment nested %s: %w", mpkProp.Key, err)
		}
	}

	return nil
}

// augmentNestedObjectType syncs nested ObjectType PropertyTypes for an Object-type property.
// When a .mpk defines children for a property (e.g., DataGrid2 "columns" has showContentAs,
// attribute, content, header, etc.), this function ensures the template's nested ObjectType
// has the same PropertyTypes as the .mpk, adding missing ones and removing stale ones.
func augmentNestedObjectType(propTypes []any, objProps []any, mpkProp mpk.PropertyDef) error {
	matchedPT, matchedPTID := findPropertyType(propTypes, mpkProp.Key)
	if matchedPT == nil {
		return nil
	}

	ctx := resolveNestedObjectContext(matchedPT, matchedPTID, objProps)
	if ctx == nil {
		return nil
	}

	existingKeys, nestedExemplars := buildExemplarIndex(ctx.nestedPropTypes)
	missing, staleKeys := diffPropertyKeys(mpkProp.Children, existingKeys)

	if len(missing) == 0 && len(staleKeys) == 0 {
		return nil
	}

	nestedPropTypes := ctx.nestedPropTypes
	nestedObjProps := ctx.nestedObjProps

	if len(staleKeys) > 0 {
		nestedPropTypes, nestedObjProps = removeNestedProperties(nestedPropTypes, nestedObjProps, staleKeys)
	}

	nestedPropTypes, nestedObjProps, err := addMissingProperties(nestedPropTypes, nestedObjProps, nestedExemplars, missing)
	if err != nil {
		return err
	}

	// Write back
	setArrayField(ctx.nestedObjType, "PropertyTypes", nestedPropTypes)
	for i, container := range ctx.objPropContainers {
		if i < len(nestedObjProps) {
			setArrayField(container, "Properties", nestedObjProps[i])
		}
	}
	return nil
}

// addMissingProperties clones or creates PropertyTypes and Properties for missing keys.
func addMissingProperties(nestedPropTypes []any, nestedObjProps [][]any, exemplars map[string]int, missing []mpk.PropertyDef) ([]any, [][]any, error) {
	cloner := defaultCloner()
	for _, child := range missing {
		bsonType := xmlTypeToBSONType(child.Type)
		if bsonType == "" {
			continue
		}

		exemplarIdx, hasExemplar := exemplars[bsonType]
		var newPropType, newProp map[string]any
		if hasExemplar && len(nestedObjProps) > 0 {
			var err error
			newPropType, newProp, err = cloner.ClonePair(nestedPropTypes, nestedObjProps[0], exemplarIdx, child)
			if err != nil {
				return nil, nil, fmt.Errorf("clone nested property %q: %w", child.Key, err)
			}
		}
		if newPropType == nil || newProp == nil {
			newPropType, newProp = createPropertyPair(child, bsonType)
		}

		if newPropType != nil {
			nestedPropTypes = append(nestedPropTypes, newPropType)
		}
		if newProp != nil {
			for i := range nestedObjProps {
				nestedObjProps[i] = append(nestedObjProps[i], newProp)
			}
		}
	}
	return nestedPropTypes, nestedObjProps, nil
}

// removeProperties removes PropertyTypes and their corresponding Properties by PropertyKey.
func removeProperties(propTypes []any, objProps []any, staleKeys map[string]bool) ([]any, []any) {
	// Collect IDs of PropertyTypes to remove
	removeIDs := make(map[string]bool)
	var newPropTypes []any
	for _, pt := range propTypes {
		ptMap, ok := pt.(map[string]any)
		if !ok {
			newPropTypes = append(newPropTypes, pt) // Keep markers (e.g., float64(2))
			continue
		}
		key, _ := ptMap["PropertyKey"].(string)
		if staleKeys[key] {
			id, _ := ptMap["$ID"].(string)
			if id != "" {
				removeIDs[id] = true
			}
			continue // Skip this PropertyType
		}
		newPropTypes = append(newPropTypes, pt)
	}

	// Remove corresponding Properties whose TypePointer matches a removed PropertyType
	var newObjProps []any
	for _, prop := range objProps {
		propMap, ok := prop.(map[string]any)
		if !ok {
			newObjProps = append(newObjProps, prop) // Keep markers
			continue
		}
		tp, _ := propMap["TypePointer"].(string)
		if removeIDs[tp] {
			continue // Remove this property
		}
		newObjProps = append(newObjProps, prop)
	}

	return newPropTypes, newObjProps
}

// defaultCloner returns a PropertyCloner using the package-level placeholderID generator.
func defaultCloner() *PropertyCloner {
	return NewPropertyCloner(placeholderID)
}

// createPropertyPair creates a new PropertyType/Property pair from scratch.
func createPropertyPair(p mpk.PropertyDef, bsonType string) (map[string]any, map[string]any) {
	ptID := placeholderID()
	vtID := placeholderID()

	// Create PropertyType
	pt := map[string]any{
		"$ID":         ptID,
		"$Type":       "CustomWidgets$WidgetPropertyType",
		"Caption":     p.Caption,
		"Category":    p.Category,
		"Description": p.Description,
		"IsDefault":   false,
		"PropertyKey": p.Key,
		"ValueType":   createDefaultValueType(vtID, bsonType, p),
	}

	// Create Property (WidgetProperty with WidgetValue)
	prop := map[string]any{
		"$ID":         placeholderID(),
		"$Type":       "CustomWidgets$WidgetProperty",
		"TypePointer": ptID,
		"Value":       createDefaultWidgetValue(vtID, bsonType, p),
	}

	return pt, prop
}

// createDefaultValueType creates a default ValueType structure for a given BSON type.
// Only type-relevant fields are included to match Studio Pro's expectations.
func createDefaultValueType(vtID string, bsonType string, p mpk.PropertyDef) map[string]any {
	allowedTypes := []any{float64(1)}
	for _, t := range p.AllowedTypes {
		allowedTypes = append(allowedTypes, t)
	}

	vt := map[string]any{
		"$ID":          vtID,
		"$Type":        "CustomWidgets$WidgetValueType",
		"DefaultValue": p.DefaultValue,
		"IsList":       p.IsList,
		"Required":     p.Required,
		"Type":         bsonType,
	}

	switch bsonType {
	case "Attribute":
		vt["AllowedTypes"] = allowedTypes
		vt["DataSourceProperty"] = p.DataSource
		vt["IsPath"] = "No"
		vt["PathType"] = "None"
	case "Association":
		vt["AssociationTypes"] = []any{float64(1)}
		vt["DataSourceProperty"] = p.DataSource
		vt["IsPath"] = "No"
		vt["PathType"] = "None"
	case "Action":
		vt["ActionVariables"] = []any{float64(2)}
	case "Boolean", "Integer", "Decimal", "String":
		// No additional type-specific fields needed
	case "Enumeration":
		enumValues := []any{float64(2)}
		for _, ev := range p.EnumerationValues {
			enumValues = append(enumValues, map[string]any{
				"$ID":     placeholderID(),
				"$Type":   "CustomWidgets$WidgetEnumerationValue",
				"Caption": ev.Caption,
				"_Key":    ev.Key,
			})
		}
		vt["EnumerationValues"] = enumValues
	case "Expression":
		vt["EntityProperty"] = ""
		vt["ReturnType"] = buildReturnType(bsonType, p)
	case "DataSource":
		vt["EntityProperty"] = ""
	case "Icon":
		// No additional fields needed
	case "Image":
		// No additional fields needed
	case "Object":
		if len(p.Children) > 0 {
			vt["ObjectType"] = buildNestedObjectType(p.Children)
		}
	case "Selection":
		selectionTypes := []any{float64(1)}
		for _, st := range p.SelectionTypes {
			selectionTypes = append(selectionTypes, st)
		}
		vt["DataSourceProperty"] = p.DataSource
		vt["SelectionTypes"] = selectionTypes
	case "TextTemplate":
		vt["Translations"] = []any{float64(2)}
	case "Widgets":
		// No additional fields needed
	case "File":
		// No additional fields needed
	}

	if p.DataSource != "" && bsonType != "Attribute" && bsonType != "Association" &&
		bsonType != "Selection" {
		vt["DataSourceProperty"] = p.DataSource
	}

	return vt
}



// createDefaultWidgetValue creates a default WidgetValue for a given BSON type.
// Only type-relevant fields are included to match Studio Pro's expectations.
// Nil-valued fields are omitted during BSON serialization to avoid CE0463.
func createDefaultWidgetValue(vtID string, bsonType string, p mpk.PropertyDef) map[string]any {
	val := map[string]any{
		"$ID":               placeholderID(),
		"$Type":             "CustomWidgets$WidgetValue",
		"AttributeRef":      nil,
		"DataSource":        nil,
		"EntityRef":         nil,
		"Icon":              nil,
		"SourceVariable":    nil,
		"TextTemplate":      nil,
		"TranslatableValue": nil,
		"TypePointer":       vtID,
	}

	switch bsonType {
	case "Boolean":
		if p.DefaultValue != "" {
			val["PrimitiveValue"] = p.DefaultValue
		} else {
			val["PrimitiveValue"] = "false"
		}
	case "Integer":
		if p.DefaultValue != "" {
			val["PrimitiveValue"] = p.DefaultValue
		} else {
			val["PrimitiveValue"] = "0"
		}
	case "Decimal", "String":
		if p.DefaultValue != "" {
			val["PrimitiveValue"] = p.DefaultValue
		}
	case "Enumeration":
		if p.DefaultValue != "" {
			val["PrimitiveValue"] = p.DefaultValue
		}
	case "Expression":
		if p.DefaultValue != "" {
			val["Expression"] = p.DefaultValue
		} else {
			val["Expression"] = ""
		}
	case "TextTemplate":
		val["TextTemplate"] = createDefaultClientTemplate()
	case "Action":
		val["Action"] = createDefaultNoAction()
	case "Widgets":
		val["Widgets"] = []any{float64(2)}
	case "Object":
		val["Objects"] = []any{float64(2)}
	case "Selection":
		if p.DefaultValue != "" {
			val["Selection"] = p.DefaultValue
		} else {
			val["Selection"] = "None"
		}
	case "Image":
		if p.DefaultValue != "" {
			val["Image"] = p.DefaultValue
		}
	case "Attribute":
		// AttributeRef is nil by default, filled by builder
	case "Association":
		// EntityRef is nil by default, filled by builder
	case "DataSource":
		// DataSource is nil by default, filled by builder
	}

	return val
}

// createDefaultNoAction creates a default Forms$NoAction structure.
func createDefaultNoAction() map[string]any {
	return map[string]any{
		"$ID":                     placeholderID(),
		"$Type":                   "Forms$NoAction",
		"DisabledDuringExecution": true,
	}
}

// createDefaultClientTemplate creates a default Forms$ClientTemplate structure.
func createDefaultClientTemplate() map[string]any {
	return map[string]any{
		"$ID":   placeholderID(),
		"$Type": "Forms$ClientTemplate",
		"Fallback": map[string]any{
			"$ID":   placeholderID(),
			"$Type": "Texts$Text",
			"Items": []any{float64(3)},
		},
		"Parameters": []any{float64(2)},
		"Template": map[string]any{
			"$ID":   placeholderID(),
			"$Type": "Texts$Text",
			"Items": []any{float64(3)},
		},
	}
}

// resetPropertyValue resets a WidgetValue to defaults for the given property type.
func resetPropertyValue(val map[string]any, p mpk.PropertyDef) {
	bsonType := xmlTypeToBSONType(p.Type)

	// Clear type-specific keys, keeping only common reference fields.
	for _, k := range []string{"Action", "Expression", "Form", "Image", "Microflow",
		"Nanoflow", "Objects", "PrimitiveValue", "Selection", "Widgets", "XPathConstraint"} {
		delete(val, k)
	}

	// Reset common reference fields.
	val["AttributeRef"] = nil
	val["DataSource"] = nil
	val["EntityRef"] = nil
	val["Icon"] = nil
	val["SourceVariable"] = nil
	val["TextTemplate"] = nil
	val["TranslatableValue"] = nil

	// Set type-specific defaults.
	switch bsonType {
	case "Boolean":
		if p.DefaultValue != "" {
			val["PrimitiveValue"] = p.DefaultValue
		} else {
			val["PrimitiveValue"] = "false"
		}
	case "Integer":
		if p.DefaultValue != "" {
			val["PrimitiveValue"] = p.DefaultValue
		} else {
			val["PrimitiveValue"] = "0"
		}
	case "Decimal", "String":
		if p.DefaultValue != "" {
			val["PrimitiveValue"] = p.DefaultValue
		}
	case "Enumeration":
		if p.DefaultValue != "" {
			val["PrimitiveValue"] = p.DefaultValue
		}
	case "Expression":
		if p.DefaultValue != "" {
			val["Expression"] = p.DefaultValue
		} else {
			val["Expression"] = ""
		}
	case "TextTemplate":
		val["TextTemplate"] = createDefaultClientTemplate()
	case "Action":
		val["Action"] = createDefaultNoAction()
	case "Widgets":
		val["Widgets"] = []any{float64(2)}
	case "Object":
		val["Objects"] = []any{float64(2)}
	case "Selection":
		if p.DefaultValue != "" {
			val["Selection"] = p.DefaultValue
		} else {
			val["Selection"] = "None"
		}
	case "Image":
		if p.DefaultValue != "" {
			val["Image"] = p.DefaultValue
		}
	}
}

// buildReturnType creates the ReturnType BSON for a ValueType.
// Only Expression-type properties need a non-nil ReturnType;
// Studio Pro's WidgetValue.GetExpectedExpressionType() dereferences this field
// for expression properties and crashes with NPE if it is nil.
func buildReturnType(bsonType string, p mpk.PropertyDef) any {
	if bsonType != "Expression" {
		return nil
	}
	retType := "String" // safe default
	var assignableTo, entityProperty string
	if p.ReturnType != nil {
		if p.ReturnType.Type != "" {
			retType = p.ReturnType.Type
		}
		assignableTo = p.ReturnType.AssignableTo
		entityProperty = p.ReturnType.EntityProperty
	}
	return map[string]any{
		"$ID":            placeholderID(),
		"$Type":          "CustomWidgets$WidgetReturnType",
		"AssignableTo":   assignableTo,
		"EntityProperty": entityProperty,
		"IsList":         false,
		"Type":           retType,
	}
}

// xmlTypeToBSONType maps XML property type to BSON ValueType.Type.
func xmlTypeToBSONType(xmlType string) string {
	switch mpk.NormalizeType(xmlType) {
	case "attribute":
		return "Attribute"
	case "expression":
		return "Expression"
	case "textTemplate":
		return "TextTemplate"
	case "widgets":
		return "Widgets"
	case "enumeration":
		return "Enumeration"
	case "boolean":
		return "Boolean"
	case "integer":
		return "Integer"
	case "datasource":
		return "DataSource"
	case "action":
		return "Action"
	case "selection":
		return "Selection"
	case "association":
		return "Association"
	case "object":
		return "Object"
	case "string":
		return "String"
	case "decimal":
		return "Decimal"
	case "icon":
		return "Icon"
	case "image":
		return "Image"
	case "file":
		return "File"
	default:
		return ""
	}
}

// buildNestedObjectType creates a WidgetObjectType with PropertyTypes for nested children
// of an object-type property. This is needed for properties like filterList and sortList
// that contain sub-properties (e.g., filter, attribute, caption).
func buildNestedObjectType(children []mpk.PropertyDef) map[string]any {
	propTypes := []any{float64(2)} // version marker

	for _, child := range children {
		childBsonType := xmlTypeToBSONType(child.Type)
		if childBsonType == "" {
			continue
		}

		childVTID := placeholderID()
		childPT := map[string]any{
			"$ID":         placeholderID(),
			"$Type":       "CustomWidgets$WidgetPropertyType",
			"Caption":     child.Caption,
			"Category":    "General",
			"Description": child.Description,
			"IsDefault":   false,
			"PropertyKey": child.Key,
			"ValueType":   createDefaultValueType(childVTID, childBsonType, child),
		}

		propTypes = append(propTypes, childPT)
	}

	return map[string]any{
		"$ID":           placeholderID(),
		"$Type":         "CustomWidgets$WidgetObjectType",
		"PropertyTypes": propTypes,
	}
}

// --- Helpers ---

// placeholderCounter generates sequential placeholder IDs (atomic for concurrent safety).
var placeholderCounter atomic.Uint32

// placeholderID generates a placeholder hex ID. These will be remapped by collectIDs
// in GetTemplateFullBSON, so exact values don't matter — they just need to be unique
// 32-char hex strings.
func placeholderID() string {
	n := placeholderCounter.Add(1)
	return fmt.Sprintf("aa000000000000000000000000%06x", n)
}

// ResetPlaceholderCounter resets the counter (for testing).
func ResetPlaceholderCounter() {
	placeholderCounter.Store(0)
}

// getMapField gets a nested map field from a JSON map.
func getMapField(m map[string]any, key string) (map[string]any, bool) {
	val, ok := m[key]
	if !ok {
		return nil, false
	}
	nested, ok := val.(map[string]any)
	return nested, ok
}

// getArrayField gets an array field from a JSON map.
func getArrayField(m map[string]any, key string) ([]any, bool) {
	val, ok := m[key]
	if !ok {
		return nil, false
	}
	arr, ok := val.([]any)
	return arr, ok
}

// setArrayField sets an array field in a JSON map.
func setArrayField(m map[string]any, key string, arr []any) {
	m[key] = arr
}

// deepCloneMap deep-clones a map[string]interface{} via JSON round-trip.
func deepCloneMap(m map[string]any) (map[string]any, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("deep clone marshal: %w", err)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("deep clone unmarshal: %w", err)
	}
	return result, nil
}

// deepCloneTemplate deep-clones a WidgetTemplate so augmentation doesn't mutate the cache.
func deepCloneTemplate(tmpl *WidgetTemplate) (*WidgetTemplate, error) {
	clone := &WidgetTemplate{
		WidgetID:      tmpl.WidgetID,
		Name:          tmpl.Name,
		Version:       tmpl.Version,
		ExtractedFrom: tmpl.ExtractedFrom,
		Generated:     tmpl.Generated,
		StableIds:     tmpl.StableIds,
	}

	if tmpl.Type != nil {
		var err error
		clone.Type, err = deepCloneMap(tmpl.Type)
		if err != nil {
			return nil, fmt.Errorf("clone template type %s: %w", tmpl.WidgetID, err)
		}
	}
	if tmpl.Object != nil {
		var err error
		clone.Object, err = deepCloneMap(tmpl.Object)
		if err != nil {
			return nil, fmt.Errorf("clone template object %s: %w", tmpl.WidgetID, err)
		}
	}

	return clone, nil
}

// collectNestedPropertyTypeIDs extracts PropertyKey→$ID mappings from a ValueType's ObjectType.
func collectNestedPropertyTypeIDs(vt map[string]any) map[string]string {
	result := make(map[string]string)
	objType, ok := getMapField(vt, "ObjectType")
	if !ok {
		return result
	}
	propTypes, ok := getArrayField(objType, "PropertyTypes")
	if !ok {
		return result
	}
	for _, pt := range propTypes {
		ptMap, ok := pt.(map[string]any)
		if !ok {
			continue
		}
		key, _ := ptMap["PropertyKey"].(string)
		id, _ := ptMap["$ID"].(string)
		if key != "" && id != "" {
			result[key] = id
		}
	}
	return result
}

// collectNestedPropertyTypeIDsByKey extracts PropertyKey→$ID from a rebuilt ObjectType map.
func collectNestedPropertyTypeIDsByKey(objType map[string]any) map[string]string {
	result := make(map[string]string)
	propTypes, ok := getArrayField(objType, "PropertyTypes")
	if !ok {
		return result
	}
	for _, pt := range propTypes {
		ptMap, ok := pt.(map[string]any)
		if !ok {
			continue
		}
		key, _ := ptMap["PropertyKey"].(string)
		id, _ := ptMap["$ID"].(string)
		if key != "" && id != "" {
			result[key] = id
		}
	}
	return result
}

// remapObjectTypePointers walks the Object Properties array and updates TypePointers
// that reference old PropertyType IDs from a rebuilt ObjectType.
func remapObjectTypePointers(objProps []any, idRemap map[string]string) {
	if len(idRemap) == 0 {
		return
	}
	for _, prop := range objProps {
		propMap, ok := prop.(map[string]any)
		if !ok {
			continue
		}
		// Check Value.Objects for nested WidgetObjects with TypePointers
		val, ok := getMapField(propMap, "Value")
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
			// Remap the object's nested properties' TypePointers
			nestedProps, ok := getArrayField(objMap, "Properties")
			if !ok {
				continue
			}
			for _, nestedProp := range nestedProps {
				npMap, ok := nestedProp.(map[string]any)
				if !ok {
					continue
				}
				if tp, ok := npMap["TypePointer"].(string); ok {
					if newTP, exists := idRemap[tp]; exists {
						npMap["TypePointer"] = newTP
					}
				}
				// Also remap Value.TypePointer (references ValueType $ID)
				if nestedVal, ok := getMapField(npMap, "Value"); ok {
					if tp, ok := nestedVal["TypePointer"].(string); ok {
						if newTP, exists := idRemap[tp]; exists {
							nestedVal["TypePointer"] = newTP
						}
					}
				}
			}
		}
	}
}
