// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/bsonutil"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	"github.com/mendixlabs/mxcli/modelsdk/widgets"
)

// ---------------------------------------------------------------------------
// mprWidgetObjectBuilder — implements backend.WidgetObjectBuilder
// ---------------------------------------------------------------------------

type mprWidgetObjectBuilder struct {
	embeddedType    bson.D
	object          bson.D // the mutable widget object BSON
	propertyTypeIDs map[string]types.PropertyTypeIDEntry
	objectTypeID    string
}

var _ backend.WidgetObjectBuilder = (*mprWidgetObjectBuilder)(nil)

// ---------------------------------------------------------------------------
// WidgetBuilderBackend — MprBackend methods
// ---------------------------------------------------------------------------

// LoadWidgetTemplate loads a widget template by ID and returns a builder.
func (b *MprBackend) LoadWidgetTemplate(widgetID string, projectPath string) (backend.WidgetObjectBuilder, error) {
	embeddedType, embeddedObject, embeddedIDs, objectTypeID, err :=
		widgets.GetTemplateFullBSON(widgetID, types.GenerateID, projectPath)
	if err != nil {
		return nil, err
	}
	if embeddedType == nil || embeddedObject == nil {
		return nil, nil
	}

	propertyTypeIDs := embeddedIDs

	return &mprWidgetObjectBuilder{
		embeddedType:    embeddedType,
		object:          embeddedObject,
		propertyTypeIDs: propertyTypeIDs,
		objectTypeID:    objectTypeID,
	}, nil
}

// SerializeGenElemToOpaque encodes a gen element.Element to opaque BSON form
// for use as a child widget, datasource, or action in pluggable widget templates.
// This is the Cat-B complement to SerializeWidgetToOpaque for use when buildWidgetV3
// returns element.Element directly.
func (b *MprBackend) SerializeGenElemToOpaque(elem element.Element) backend.OpaqueWidget {
	if elem == nil {
		return nil
	}
	enc := codec.Encoder{}
	raw, err := enc.Encode(elem)
	if err != nil {
		log.Printf("warning: SerializeGenElemToOpaque encode failed (%v)", err)
		return nil
	}
	return bson.Raw(raw)
}

// BuildCreateAttributeObject creates an attribute object for filter widgets.
func (b *MprBackend) BuildCreateAttributeObject(attributePath string, objectTypeID, propertyTypeID, valueTypeID string) (any, error) {
	return createAttributeObject(attributePath, objectTypeID, propertyTypeID, valueTypeID)
}

// ---------------------------------------------------------------------------
// WidgetObjectBuilder — property operations
// ---------------------------------------------------------------------------

func (ob *mprWidgetObjectBuilder) SetAttribute(propertyKey string, attributePath string) {
	if attributePath == "" {
		return
	}
	ob.object = updateWidgetPropertyValue(ob.object, ob.propertyTypeIDs, propertyKey, func(val bson.D) bson.D {
		return setAttributeRef(val, attributePath)
	})
}

func (ob *mprWidgetObjectBuilder) SetAssociation(propertyKey string, assocPath string, entityName string) {
	if assocPath == "" {
		return
	}
	ob.object = updateWidgetPropertyValue(ob.object, ob.propertyTypeIDs, propertyKey, func(val bson.D) bson.D {
		return setAssociationRef(val, assocPath, entityName)
	})
}

func (ob *mprWidgetObjectBuilder) SetPrimitive(propertyKey string, value string) {
	if value == "" {
		return
	}
	ob.object = updateWidgetPropertyValue(ob.object, ob.propertyTypeIDs, propertyKey, func(val bson.D) bson.D {
		return setPrimitiveValue(val, value)
	})
}

func (ob *mprWidgetObjectBuilder) SetSelection(propertyKey string, value string) {
	if value == "" {
		return
	}
	ob.object = updateWidgetPropertyValue(ob.object, ob.propertyTypeIDs, propertyKey, func(val bson.D) bson.D {
		result := make(bson.D, 0, len(val))
		for _, elem := range val {
			if elem.Key == "Selection" {
				result = append(result, bson.E{Key: "Selection", Value: value})
			} else {
				result = append(result, elem)
			}
		}
		return result
	})
}

func (ob *mprWidgetObjectBuilder) SetExpression(propertyKey string, value string) {
	if value == "" {
		return
	}
	ob.object = updateWidgetPropertyValue(ob.object, ob.propertyTypeIDs, propertyKey, func(val bson.D) bson.D {
		result := make(bson.D, 0, len(val))
		for _, elem := range val {
			if elem.Key == "Expression" {
				result = append(result, bson.E{Key: "Expression", Value: value})
			} else {
				result = append(result, elem)
			}
		}
		return result
	})
}

func (ob *mprWidgetObjectBuilder) SetDataSourceOpaque(propertyKey string, ds backend.OpaqueDataSource) {
	if ds == nil {
		return
	}
	ob.object = updateWidgetPropertyValue(ob.object, ob.propertyTypeIDs, propertyKey, func(val bson.D) bson.D {
		return setDataSourceOpaque(val, ds)
	})
}

func (ob *mprWidgetObjectBuilder) SetChildWidgetsOpaque(propertyKey string, children []backend.OpaqueWidget) {
	if len(children) == 0 {
		return
	}
	ob.object = updateWidgetPropertyValue(ob.object, ob.propertyTypeIDs, propertyKey, func(val bson.D) bson.D {
		return setChildWidgetsOpaque(val, children)
	})
}

func (ob *mprWidgetObjectBuilder) SetTextTemplate(propertyKey string, text string) {
	if text == "" {
		return
	}
	ob.object = updateWidgetPropertyValue(ob.object, ob.propertyTypeIDs, propertyKey, func(val bson.D) bson.D {
		return setTextTemplateValue(val, text)
	})
}

func (ob *mprWidgetObjectBuilder) SetTextTemplateWithParams(propertyKey string, text string, entityContext string) {
	if text == "" {
		return
	}
	tmpl := createClientTemplateBSONWithParams(text, entityContext)
	ob.object = updateWidgetPropertyValue(ob.object, ob.propertyTypeIDs, propertyKey, func(val bson.D) bson.D {
		result := make(bson.D, 0, len(val))
		for _, elem := range val {
			if elem.Key == "TextTemplate" {
				result = append(result, bson.E{Key: "TextTemplate", Value: tmpl})
			} else {
				result = append(result, elem)
			}
		}
		return result
	})
}

func (ob *mprWidgetObjectBuilder) SetActionOpaque(propertyKey string, action backend.OpaqueAction) {
	if action == nil {
		return
	}
	ob.object = updateWidgetPropertyValue(ob.object, ob.propertyTypeIDs, propertyKey, func(val bson.D) bson.D {
		result := make(bson.D, 0, len(val))
		for _, elem := range val {
			if elem.Key == "Action" {
				result = append(result, bson.E{Key: "Action", Value: action})
			} else {
				result = append(result, elem)
			}
		}
		return result
	})
}

func (ob *mprWidgetObjectBuilder) SetAttributeObjects(propertyKey string, attributePaths []string) {
	if len(attributePaths) == 0 {
		return
	}

	entry, ok := ob.propertyTypeIDs[propertyKey]
	if !ok || entry.ObjectTypeID == "" {
		return
	}

	nestedEntry, ok := entry.NestedPropertyIDs["attribute"]
	if !ok {
		return
	}

	ob.object = updateWidgetPropertyValue(ob.object, ob.propertyTypeIDs, propertyKey, func(val bson.D) bson.D {
		objects := make([]any, 0, len(attributePaths)+1)
		objects = append(objects, int32(2)) // BSON array version marker

		for _, attrPath := range attributePaths {
			attrObj, err := createAttributeObject(attrPath, entry.ObjectTypeID, nestedEntry.PropertyTypeID, nestedEntry.ValueTypeID)
			if err != nil {
				// TODO(shared-types): propagate error instead of logging — requires interface change.
				log.Printf("warning: skipping attribute %s: %v", attrPath, err)
				continue
			}
			objects = append(objects, attrObj)
		}

		result := make(bson.D, 0, len(val))
		for _, elem := range val {
			if elem.Key == "Objects" {
				result = append(result, bson.E{Key: "Objects", Value: bson.A(objects)})
			} else {
				result = append(result, elem)
			}
		}
		return result
	})
}

// ---------------------------------------------------------------------------
// Template metadata
// ---------------------------------------------------------------------------

func (ob *mprWidgetObjectBuilder) PropertyTypeIDs() map[string]types.PropertyTypeIDEntry {
	return ob.propertyTypeIDs
}

// ---------------------------------------------------------------------------
// Object list defaults
// ---------------------------------------------------------------------------

func (ob *mprWidgetObjectBuilder) EnsureRequiredObjectLists() {
	ob.object = ensureRequiredObjectLists(ob.object, ob.propertyTypeIDs)
}

// ---------------------------------------------------------------------------
// Gallery-specific
// ---------------------------------------------------------------------------

func (ob *mprWidgetObjectBuilder) CloneGallerySelectionProperty(propertyKey string, selectionMode string) {
	propEntry, ok := ob.propertyTypeIDs[propertyKey]
	if !ok {
		return
	}

	// Work at the Properties array level: find the property, clone it with new
	// IDs and updated Selection, then append.
	result := make(bson.D, 0, len(ob.object))
	for _, elem := range ob.object {
		if elem.Key == "Properties" {
			if arr, ok := elem.Value.(bson.A); ok {
				newArr := make(bson.A, len(arr))
				copy(newArr, arr)
				// Find the matching property and clone it
				for _, item := range arr {
					if prop, ok := item.(bson.D); ok {
						if matchesTypePointer(prop, propEntry.PropertyTypeID) {
							cloned := buildGallerySelectionProperty(prop, selectionMode)
							newArr = append(newArr, cloned)
							break
						}
					}
				}
				result = append(result, bson.E{Key: "Properties", Value: newArr})
				continue
			}
		}
		result = append(result, elem)
	}
	ob.object = result
}

// ---------------------------------------------------------------------------
// Finalize
// ---------------------------------------------------------------------------

// FinalizeGen is the Stage 3.3.5 gen-native Finalize path.
// It constructs the CustomWidget BSON directly without going through sdk/pages
// types, then decodes via the gen codec to produce a *backend.GenCustomWidgetElem
// that satisfies both backend.Widget and element.Element.
func (ob *mprWidgetObjectBuilder) FinalizeGen(id model.ID, name string, label string, editable string) (*backend.GenCustomWidgetElem, error) {
	if editable == "" {
		editable = "Always"
	}

	// Build LabelTemplate BSON only when a label is provided.
	var labelTemplate any
	if label != "" {
		labelTemplate = bson.D{
			{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
			{Key: "$Type", Value: "Forms$ClientTemplate"},
			{Key: "Fallback", Value: bson.D{
				{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
				{Key: "$Type", Value: "Texts$Text"},
				{Key: "Items", Value: bson.A{int32(3)}},
			}},
			{Key: "Parameters", Value: bson.A{int32(3)}},
			{Key: "Template", Value: bson.D{
				{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
				{Key: "$Type", Value: "Texts$Text"},
				{Key: "Items", Value: bson.A{int32(3), bson.D{
					{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
					{Key: "$Type", Value: "Texts$Translation"},
					{Key: "LanguageCode", Value: "en_US"},
					{Key: "Text", Value: label},
				}}},
			}},
		}
	}

	doc := bson.D{
		{Key: "$ID", Value: bsonutil.IDToBsonBinary(string(id))},
		{Key: "$Type", Value: "CustomWidgets$CustomWidget"},
		{Key: "Appearance", Value: bson.D{
			{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
			{Key: "$Type", Value: "Forms$Appearance"},
			{Key: "Class", Value: ""},
			{Key: "DesignProperties", Value: bson.A{int32(3)}},
			{Key: "DynamicClasses", Value: ""},
			{Key: "Style", Value: ""},
		}},
		{Key: "ConditionalEditabilitySettings", Value: nil},
		{Key: "ConditionalVisibilitySettings", Value: nil},
		{Key: "Editable", Value: editable},
		{Key: "LabelTemplate", Value: labelTemplate},
		{Key: "Name", Value: name},
		{Key: "Object", Value: ob.object},
		{Key: "TabIndex", Value: int32(0)},
		{Key: "Type", Value: ob.embeddedType},
	}

	raw, err := bson.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("FinalizeGen: marshal: %w", err)
	}

	// Decode via gen codec — produces *genCW.CustomWidget (element.Element).
	dec := codec.NewDecoder(codec.DefaultRegistry)
	elem, err := dec.Decode(bson.Raw(raw))
	if err != nil {
		return nil, fmt.Errorf("FinalizeGen: decode: %w", err)
	}

	return backend.NewGenCustomWidgetElem(elem), nil
}

// ===========================================================================
// Package-level helpers (moved from executor)
// ===========================================================================

// ---------------------------------------------------------------------------
// Property update core
// ---------------------------------------------------------------------------

// updateWidgetPropertyValue finds and updates a specific property value in a WidgetObject.
func updateWidgetPropertyValue(obj bson.D, propTypeIDs map[string]types.PropertyTypeIDEntry, propertyKey string, updateFn func(bson.D) bson.D) bson.D {
	propEntry, ok := propTypeIDs[propertyKey]
	if !ok {
		return obj
	}

	result := make(bson.D, 0, len(obj))
	for _, elem := range obj {
		if elem.Key == "Properties" {
			if arr, ok := elem.Value.(bson.A); ok {
				result = append(result, bson.E{Key: "Properties", Value: updatePropertyInArray(arr, propEntry.PropertyTypeID, updateFn)})
				continue
			}
		}
		result = append(result, elem)
	}
	return result
}

// updatePropertyInArray finds a property by TypePointer and updates its value.
func updatePropertyInArray(arr bson.A, propertyTypeID string, updateFn func(bson.D) bson.D) bson.A {
	result := make(bson.A, len(arr))
	matched := false
	for i, item := range arr {
		if prop, ok := item.(bson.D); ok {
			if matchesTypePointer(prop, propertyTypeID) {
				result[i] = updatePropertyValue(prop, updateFn)
				matched = true
			} else {
				result[i] = item
			}
		} else {
			result[i] = item
		}
	}
	if !matched {
		// TODO(shared-types): propagate warning instead of logging — requires interface change.
		log.Printf("warning: updatePropertyInArray: no match for TypePointer %s in %d properties", propertyTypeID, len(arr)-1)
	}
	return result
}

// matchesTypePointer checks if a WidgetProperty has the given TypePointer.
func matchesTypePointer(prop bson.D, propertyTypeID string) bool {
	normalizedTarget := strings.ReplaceAll(propertyTypeID, "-", "")
	for _, elem := range prop {
		if elem.Key == "TypePointer" {
			switch v := elem.Value.(type) {
			case primitive.Binary:
				propID := strings.ReplaceAll(types.BlobToUUID(v.Data), "-", "")
				return propID == normalizedTarget
			case []byte:
				propID := strings.ReplaceAll(types.BlobToUUID(v), "-", "")
				if propID == normalizedTarget {
					return true
				}
				rawHex := fmt.Sprintf("%x", v)
				return rawHex == normalizedTarget
			}
		}
	}
	return false
}

// updatePropertyValue updates the Value field in a WidgetProperty.
func updatePropertyValue(prop bson.D, updateFn func(bson.D) bson.D) bson.D {
	result := make(bson.D, 0, len(prop))
	for _, elem := range prop {
		if elem.Key == "Value" {
			if val, ok := elem.Value.(bson.D); ok {
				result = append(result, bson.E{Key: "Value", Value: updateFn(val)})
			} else {
				result = append(result, elem)
			}
		} else {
			result = append(result, elem)
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// Value setters
// ---------------------------------------------------------------------------

func setPrimitiveValue(val bson.D, value string) bson.D {
	result := make(bson.D, 0, len(val))
	for _, elem := range val {
		if elem.Key == "PrimitiveValue" {
			result = append(result, bson.E{Key: "PrimitiveValue", Value: value})
		} else {
			result = append(result, elem)
		}
	}
	return result
}

func setDataSourceOpaque(val bson.D, ds backend.OpaqueDataSource) bson.D {
	result := make(bson.D, 0, len(val))
	for _, elem := range val {
		if elem.Key == "DataSource" {
			result = append(result, bson.E{Key: "DataSource", Value: ds})
		} else {
			result = append(result, elem)
		}
	}
	return result
}

func setAssociationRef(val bson.D, assocPath string, entityName string) bson.D {
	result := make(bson.D, 0, len(val))
	for _, elem := range val {
		if elem.Key == "EntityRef" && entityName != "" {
			result = append(result, bson.E{Key: "EntityRef", Value: bson.D{
				{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
				{Key: "$Type", Value: "DomainModels$IndirectEntityRef"},
				{Key: "Steps", Value: bson.A{
					int32(2),
					bson.D{
						{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
						{Key: "$Type", Value: "DomainModels$EntityRefStep"},
						{Key: "Association", Value: assocPath},
						{Key: "DestinationEntity", Value: entityName},
					},
				}},
			}})
		} else {
			result = append(result, elem)
		}
	}
	return result
}

func setAttributeRef(val bson.D, attrPath string) bson.D {
	result := make(bson.D, 0, len(val))
	for _, elem := range val {
		if elem.Key == "AttributeRef" {
			if strings.Count(attrPath, ".") >= 2 {
				result = append(result, bson.E{Key: "AttributeRef", Value: bson.D{
					{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
					{Key: "$Type", Value: "DomainModels$AttributeRef"},
					{Key: "Attribute", Value: attrPath},
					{Key: "EntityRef", Value: nil},
				}})
			} else {
				result = append(result, bson.E{Key: "AttributeRef", Value: nil})
			}
		} else {
			result = append(result, elem)
		}
	}
	return result
}

func setChildWidgetsOpaque(val bson.D, children []backend.OpaqueWidget) bson.D {
	widgetsArr := bson.A{int32(2)}
	for _, w := range children {
		widgetsArr = append(widgetsArr, w)
	}

	result := make(bson.D, 0, len(val))
	for _, elem := range val {
		if elem.Key == "Widgets" {
			result = append(result, bson.E{Key: "Widgets", Value: widgetsArr})
		} else {
			result = append(result, elem)
		}
	}
	return result
}

func setTextTemplateValue(val bson.D, text string) bson.D {
	result := make(bson.D, 0, len(val))
	for _, elem := range val {
		if elem.Key == "TextTemplate" {
			if tmpl, ok := elem.Value.(bson.D); ok && tmpl != nil {
				result = append(result, bson.E{Key: "TextTemplate", Value: updateTemplateText(tmpl, text)})
			} else {
				result = append(result, elem)
			}
		} else {
			result = append(result, elem)
		}
	}
	return result
}

func updateTemplateText(tmpl bson.D, text string) bson.D {
	result := make(bson.D, 0, len(tmpl))
	for _, elem := range tmpl {
		if elem.Key == "Template" {
			if template, ok := elem.Value.(bson.D); ok {
				updated := make(bson.D, 0, len(template))
				for _, tElem := range template {
					if tElem.Key == "Items" {
						updated = append(updated, bson.E{Key: "Items", Value: bson.A{
							int32(3),
							bson.D{
								{Key: "$ID", Value: bsonutil.IDToBsonBinary(types.GenerateID())},
								{Key: "$Type", Value: "Texts$Translation"},
								{Key: "LanguageCode", Value: "en_US"},
								{Key: "Text", Value: text},
							},
						}})
					} else {
						updated = append(updated, tElem)
					}
				}
				result = append(result, bson.E{Key: "Template", Value: updated})
			} else {
				result = append(result, elem)
			}
		} else {
			result = append(result, elem)
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// Template helpers
// ---------------------------------------------------------------------------

func createClientTemplateBSONWithParams(text string, entityContext string) bson.D {
	re := regexp.MustCompile(`\{([A-Za-z][A-Za-z0-9_]*)\}`)
	matches := re.FindAllStringSubmatchIndex(text, -1)

	if len(matches) == 0 {
		return createDefaultClientTemplateBSON(text)
	}

	// Collect attribute names (skip numeric placeholders)
	var attrNames []string
	for i := 0; i < len(matches); i++ {
		match := matches[i]
		attrName := text[match[2]:match[3]]
		if _, err := fmt.Sscanf(attrName, "%d", new(int)); err == nil {
			continue
		}
		attrNames = append(attrNames, attrName)
	}

	paramText := re.ReplaceAllStringFunc(text, func(s string) string {
		name := s[1 : len(s)-1]
		if _, err := fmt.Sscanf(name, "%d", new(int)); err == nil {
			return s
		}
		for i, an := range attrNames {
			if an == name {
				return fmt.Sprintf("{%d}", i+1)
			}
		}
		return s
	})

	// Build parameters BSON
	params := bson.A{int32(2)}
	for _, attrName := range attrNames {
		attrPath := attrName
		if entityContext != "" && !strings.Contains(attrName, ".") {
			attrPath = entityContext + "." + attrName
		}
		params = append(params, bson.D{
			{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
			{Key: "$Type", Value: "Forms$ClientTemplateParameter"},
			{Key: "AttributeRef", Value: bson.D{
				{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
				{Key: "$Type", Value: "DomainModels$AttributeRef"},
				{Key: "Attribute", Value: attrPath},
				{Key: "EntityRef", Value: nil},
			}},
			{Key: "Expression", Value: ""},
			{Key: "FormattingInfo", Value: bson.D{
				{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
				{Key: "$Type", Value: "Forms$FormattingInfo"},
				{Key: "CustomDateFormat", Value: ""},
				{Key: "DateFormat", Value: "Date"},
				{Key: "DecimalPrecision", Value: int64(2)},
				{Key: "EnumFormat", Value: "Text"},
				{Key: "GroupDigits", Value: false},
				{Key: "TimeFormat", Value: "HoursMinutes"},
			}},
			{Key: "SourceVariable", Value: nil},
		})
	}

	makeText := func(t string) bson.D {
		return bson.D{
			{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
			{Key: "$Type", Value: "Texts$Text"},
			{Key: "Items", Value: bson.A{int32(3), bson.D{
				{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
				{Key: "$Type", Value: "Texts$Translation"},
				{Key: "LanguageCode", Value: "en_US"},
				{Key: "Text", Value: t},
			}}},
		}
	}

	return bson.D{
		{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
		{Key: "$Type", Value: "Forms$ClientTemplate"},
		{Key: "Fallback", Value: makeText(paramText)},
		{Key: "Parameters", Value: params},
		{Key: "Template", Value: makeText(paramText)},
	}
}

func createDefaultClientTemplateBSON(text string) bson.D {
	makeText := func(t string) bson.D {
		return bson.D{
			{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
			{Key: "$Type", Value: "Texts$Text"},
			{Key: "Items", Value: bson.A{int32(3), bson.D{
				{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
				{Key: "$Type", Value: "Texts$Translation"},
				{Key: "LanguageCode", Value: "en_US"},
				{Key: "Text", Value: t},
			}}},
		}
	}
	return bson.D{
		{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
		{Key: "$Type", Value: "Forms$ClientTemplate"},
		{Key: "Fallback", Value: makeText(text)},
		{Key: "Parameters", Value: bson.A{int32(2)}},
		{Key: "Template", Value: makeText(text)},
	}
}

// ---------------------------------------------------------------------------
// Default object lists
// ---------------------------------------------------------------------------

func ensureRequiredObjectLists(obj bson.D, propertyTypeIDs map[string]types.PropertyTypeIDEntry) bson.D {
	// Sort keys for deterministic BSON output.
	keys := make([]string, 0, len(propertyTypeIDs))
	for k := range propertyTypeIDs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, propKey := range keys {
		entry := propertyTypeIDs[propKey]
		if entry.ObjectTypeID == "" || len(entry.NestedPropertyIDs) == 0 {
			continue
		}
		if !entry.Required {
			hasNestedDS := false
			for _, nested := range entry.NestedPropertyIDs {
				if nested.ValueType == "DataSource" {
					hasNestedDS = true
					break
				}
			}
			if hasNestedDS {
				continue
			}
		}
		hasRequiredAttr := false
		for _, nested := range entry.NestedPropertyIDs {
			if nested.Required && nested.ValueType == "Attribute" {
				hasRequiredAttr = true
				break
			}
		}
		if hasRequiredAttr {
			continue
		}
		obj = updateWidgetPropertyValue(obj, propertyTypeIDs, propKey, func(val bson.D) bson.D {
			for _, elem := range val {
				if elem.Key == "Objects" {
					if arr, ok := elem.Value.(bson.A); ok && len(arr) <= 1 {
						defaultObj := createDefaultWidgetObject(entry.ObjectTypeID, entry.NestedPropertyIDs)
						newArr := bson.A{int32(2), defaultObj}
						result := make(bson.D, 0, len(val))
						for _, e := range val {
							if e.Key == "Objects" {
								result = append(result, bson.E{Key: "Objects", Value: newArr})
							} else {
								result = append(result, e)
							}
						}
						return result
					}
				}
			}
			return val
		})
	}
	return obj
}

func createDefaultWidgetObject(objectTypeID string, nestedProps map[string]types.PropertyTypeIDEntry) bson.D {
	propsArr := bson.A{int32(2)}
	// Sort keys for deterministic BSON output.
	nestedKeys := make([]string, 0, len(nestedProps))
	for k := range nestedProps {
		nestedKeys = append(nestedKeys, k)
	}
	sort.Strings(nestedKeys)
	for _, k := range nestedKeys {
		entry := nestedProps[k]
		prop := createDefaultWidgetProperty(entry)
		propsArr = append(propsArr, prop)
	}
	return bson.D{
		{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
		{Key: "$Type", Value: "CustomWidgets$WidgetObject"},
		{Key: "TypePointer", Value: types.UUIDToBlob(objectTypeID)},
		{Key: "Properties", Value: propsArr},
	}
}

func createDefaultWidgetProperty(entry types.PropertyTypeIDEntry) bson.D {
	return bson.D{
		{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
		{Key: "$Type", Value: "CustomWidgets$WidgetProperty"},
		{Key: "TypePointer", Value: types.UUIDToBlob(entry.PropertyTypeID)},
		{Key: "Value", Value: createDefaultWidgetValue(entry)},
	}
}

func createDefaultWidgetValue(entry types.PropertyTypeIDEntry) bson.D {
	primitiveVal := entry.DefaultValue
	expressionVal := ""
	var textTemplate any

	switch entry.ValueType {
	case "Expression":
		expressionVal = primitiveVal
		primitiveVal = ""
	case "TextTemplate":
		text := primitiveVal
		if text == "" {
			text = " "
		}
		textTemplate = createDefaultClientTemplateBSON(text)
	case "String":
		if primitiveVal == "" {
			primitiveVal = " "
		}
	}

	return bson.D{
		{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
		{Key: "$Type", Value: "CustomWidgets$WidgetValue"},
		{Key: "Action", Value: bson.D{
			{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
			{Key: "$Type", Value: "Forms$NoAction"},
			{Key: "DisabledDuringExecution", Value: true},
		}},
		{Key: "AttributeRef", Value: nil},
		{Key: "DataSource", Value: nil},
		{Key: "EntityRef", Value: nil},
		{Key: "Expression", Value: expressionVal},
		{Key: "Form", Value: ""},
		{Key: "Icon", Value: nil},
		{Key: "Image", Value: ""},
		{Key: "Microflow", Value: ""},
		{Key: "Nanoflow", Value: ""},
		{Key: "Objects", Value: bson.A{int32(2)}},
		{Key: "PrimitiveValue", Value: primitiveVal},
		{Key: "Selection", Value: "None"},
		{Key: "SourceVariable", Value: nil},
		{Key: "TextTemplate", Value: textTemplate},
		{Key: "TranslatableValue", Value: nil},
		{Key: "TypePointer", Value: types.UUIDToBlob(entry.ValueTypeID)},
		{Key: "Widgets", Value: bson.A{int32(2)}},
		{Key: "XPathConstraint", Value: ""},
	}
}

// ---------------------------------------------------------------------------
// Gallery cloning
// ---------------------------------------------------------------------------

func buildGallerySelectionProperty(propMap bson.D, selectionMode string) bson.D {
	result := make(bson.D, 0, len(propMap))

	for _, elem := range propMap {
		if elem.Key == "$ID" {
			result = append(result, bson.E{Key: "$ID", Value: bsonutil.NewIDBsonBinary()})
		} else if elem.Key == "Value" {
			if valueMap, ok := elem.Value.(bson.D); ok {
				result = append(result, bson.E{Key: "Value", Value: cloneGallerySelectionValue(valueMap, selectionMode)})
			} else {
				result = append(result, elem)
			}
		} else {
			result = append(result, elem)
		}
	}

	return result
}

func cloneGallerySelectionValue(valueMap bson.D, selectionMode string) bson.D {
	result := make(bson.D, 0, len(valueMap))

	for _, elem := range valueMap {
		if elem.Key == "$ID" {
			result = append(result, bson.E{Key: "$ID", Value: bsonutil.NewIDBsonBinary()})
		} else if elem.Key == "Selection" {
			result = append(result, bson.E{Key: "Selection", Value: selectionMode})
		} else if elem.Key == "Action" {
			if actionMap, ok := elem.Value.(bson.D); ok {
				result = append(result, bson.E{Key: "Action", Value: cloneActionWithNewID(actionMap)})
			} else {
				result = append(result, elem)
			}
		} else {
			result = append(result, elem)
		}
	}

	return result
}

func cloneActionWithNewID(actionMap bson.D) bson.D {
	result := make(bson.D, 0, len(actionMap))

	for _, elem := range actionMap {
		if elem.Key == "$ID" {
			result = append(result, bson.E{Key: "$ID", Value: bsonutil.NewIDBsonBinary()})
		} else {
			result = append(result, elem)
		}
	}

	return result
}

// ---------------------------------------------------------------------------
// Attribute object creation
// ---------------------------------------------------------------------------

func createAttributeObject(attributePath string, objectTypeID, propertyTypeID, valueTypeID string) (bson.D, error) {
	if strings.Count(attributePath, ".") < 2 {
		return nil, mdlerrors.NewValidationf("invalid attribute path %q: expected Module.Entity.Attribute format", attributePath)
	}
	return bson.D{
		{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
		{Key: "$Type", Value: "CustomWidgets$WidgetObject"},
		{Key: "Properties", Value: bson.A{
			int32(2),
			bson.D{
				{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
				{Key: "$Type", Value: "CustomWidgets$WidgetProperty"},
				{Key: "TypePointer", Value: types.UUIDToBlob(propertyTypeID)},
				{Key: "Value", Value: bson.D{
					{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
					{Key: "$Type", Value: "CustomWidgets$WidgetValue"},
					{Key: "Action", Value: bson.D{
						{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
						{Key: "$Type", Value: "Forms$NoAction"},
						{Key: "DisabledDuringExecution", Value: true},
					}},
					{Key: "AttributeRef", Value: bson.D{
						{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
						{Key: "$Type", Value: "DomainModels$AttributeRef"},
						{Key: "Attribute", Value: attributePath},
						{Key: "EntityRef", Value: nil},
					}},
					{Key: "DataSource", Value: nil},
					{Key: "EntityRef", Value: nil},
					{Key: "Expression", Value: ""},
					{Key: "Form", Value: ""},
					{Key: "Icon", Value: nil},
					{Key: "Image", Value: ""},
					{Key: "Microflow", Value: ""},
					{Key: "Nanoflow", Value: ""},
					{Key: "Objects", Value: bson.A{int32(2)}},
					{Key: "PrimitiveValue", Value: ""},
					{Key: "Selection", Value: "None"},
					{Key: "SourceVariable", Value: nil},
					{Key: "TextTemplate", Value: nil},
					{Key: "TranslatableValue", Value: nil},
					{Key: "TypePointer", Value: types.UUIDToBlob(valueTypeID)},
					{Key: "Widgets", Value: bson.A{int32(2)}},
					{Key: "XPathConstraint", Value: ""},
				}},
			},
		}},
		{Key: "TypePointer", Value: types.UUIDToBlob(objectTypeID)},
	}, nil
}
