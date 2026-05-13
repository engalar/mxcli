// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"

	"go.mongodb.org/mongo-driver/bson"
)

// CreateImportMapping creates a new import mapping document.
func (w *Writer) CreateImportMapping(im *model.ImportMapping) error {
	if im.ID == "" {
		im.ID = model.ID(generateUUID())
	}
	im.TypeName = "ImportMappings$ImportMapping"

	contents, err := serializeImportMapping(im)
	if err != nil {
		return fmt.Errorf("failed to serialize import mapping: %w", err)
	}

	return w.insertUnit(string(im.ID), string(im.ContainerID), "Documents", "ImportMappings$ImportMapping", contents)
}

// UpdateImportMapping updates an existing import mapping document.
func (w *Writer) UpdateImportMapping(im *model.ImportMapping) error {
	contents, err := serializeImportMapping(im)
	if err != nil {
		return fmt.Errorf("failed to serialize import mapping: %w", err)
	}
	return w.updateUnit(string(im.ID), contents)
}

// DeleteImportMapping deletes an import mapping document.
func (w *Writer) DeleteImportMapping(id model.ID) error {
	return w.deleteUnit(string(id))
}

// MoveImportMapping moves an import mapping to a new container.
func (w *Writer) MoveImportMapping(im *model.ImportMapping) error {
	return w.moveUnitByID(string(im.ID), string(im.ContainerID))
}

func serializeImportMapping(im *model.ImportMapping) ([]byte, error) {
	elements := bson.A{int32(2)}
	for _, elem := range im.Elements {
		elements = append(elements, serializeImportMappingElement(elem, "(Object)"))
	}

	exportLevel := im.ExportLevel
	if exportLevel == "" {
		exportLevel = "Hidden"
	}

	// ParameterType is a required sub-document even when not used (DataTypes$UnknownType).
	// Without it Studio Pro fails to render the schema source and mapping elements correctly.
	parameterType := bson.M{
		"$ID":   idToBsonBinary(generateUUID()),
		"$Type": "DataTypes$UnknownType",
	}

	doc := bson.M{
		"$ID":               idToBsonBinary(string(im.ID)),
		"$Type":             "ImportMappings$ImportMapping",
		"Name":              im.Name,
		"Documentation":     im.Documentation,
		"Excluded":          im.Excluded,
		"ExportLevel":       exportLevel,
		"JsonStructure":     im.JsonStructure,
		"XmlSchema":         im.XmlSchema,
		"MessageDefinition": im.MessageDefinition,
		"Elements":          elements,
		// Required fields with defaults — verified against Studio Pro-created BSON
		"UseSubtransactionsForMicroflows": false,
		"PublicName":                      "", // Studio Pro writes "" not the mapping name
		"XsdRootElementName":              "",
		"MappingSourceReference":          nil,
		"ParameterType":                   parameterType,
		"OperationName":                   "",
		"ServiceName":                     "",
		"WsdlFile":                        "",
	}
	return bson.Marshal(doc)
}

func serializeImportMappingElement(elem *model.ImportMappingElement, parentPath string) bson.M {
	id := string(elem.ID)
	if id == "" {
		id = generateUUID()
	}

	if elem.Kind == "Object" || elem.Kind == "Array" {
		return serializeImportObjectElement(id, elem, parentPath)
	}
	return serializeImportValueElement(id, elem, parentPath)
}

func serializeImportObjectElement(id string, elem *model.ImportMappingElement, parentPath string) bson.M {
	// Use pre-computed JsonPath from the executor when available.
	// The executor aligns JsonPath with the JSON structure element paths.
	jsonPath := elem.JsonPath
	if jsonPath == "" {
		if elem.ExposedName == "" {
			jsonPath = parentPath
		} else {
			jsonPath = parentPath + "|" + elem.ExposedName
		}
	}

	children := bson.A{int32(2)}
	for _, child := range elem.Children {
		children = append(children, serializeImportMappingElement(child, jsonPath))
	}

	objectHandling := elem.ObjectHandling
	if objectHandling == "" {
		objectHandling = "Create"
	}
	objectHandlingBackup := objectHandling
	if objectHandling == "FindOrCreate" {
		objectHandling = "Find"
		objectHandlingBackup = "Create"
	}

	// IMPORTANT: The correct $Type is "ImportMappings$ObjectMappingElement" (no "Import" prefix in the element name).
	// The generated metamodel (ImportMappingsImportObjectMappingElement) is misleading — Studio Pro will throw
	// TypeCacheUnknownTypeException if you use "ImportMappings$ImportObjectMappingElement".
	// Rule: MappingElement $Type names do NOT repeat the namespace prefix (same for ExportMappings).
	return bson.M{
		"$ID":                               idToBsonBinary(id),
		"$Type":                             "ImportMappings$ObjectMappingElement",
		"Entity":                            elem.Entity,
		"ExposedName":                       elem.ExposedName,
		"JsonPath":                          jsonPath,
		"XmlPath":                           "",
		"ObjectHandling":                    objectHandling,
		"ObjectHandlingBackup":              objectHandlingBackup,
		"ObjectHandlingBackupAllowOverride": false,
		"Association":                       elem.Association,
		"Children":                          children,
		"MinOccurs":                         int32(elem.MinOccurs),
		"MaxOccurs":                         int32(elem.MaxOccurs),
		"Nillable":                          elem.Nillable,
		"IsDefaultType":                     false,
		"ElementType":                       elementTypeForKind(elem.Kind),
		"Documentation":                     "",
		"CustomHandlerCall":                 nil,
	}
}

func serializeImportValueElement(id string, elem *model.ImportMappingElement, parentPath string) bson.M {
	dataType := serializeImportValueDataType(elem.DataType)
	jsonPath := elem.JsonPath
	if jsonPath == "" {
		jsonPath = parentPath + "|" + elem.ExposedName
	}

	return bson.M{
		"$ID":              idToBsonBinary(id),
		"$Type":            "ImportMappings$ValueMappingElement",
		"Attribute":        elem.Attribute,
		"ExposedName":      elem.ExposedName,
		"JsonPath":         jsonPath,
		"XmlPath":          "",
		"IsKey":            elem.IsKey,
		"Type":             dataType,
		"MinOccurs":        int32(elem.MinOccurs),
		"MaxOccurs":        int32(elem.MaxOccurs),
		"Nillable":         elem.Nillable,
		"IsDefaultType":    false,
		"ElementType":      "Value",
		"Documentation":    "",
		"Converter":        "",
		"FractionDigits":   int32(elem.FractionDigits),
		"TotalDigits":      int32(elem.TotalDigits),
		"MaxLength":        int32(elem.MaxLength),
		"IsContent":        false,
		"IsXmlAttribute":   false,
		"OriginalValue":    elem.OriginalValue,
		"XmlPrimitiveType": xmlPrimitiveTypeName(elem.DataType),
	}
}

func xmlPrimitiveTypeName(dataType string) string {
	switch dataType {
	case "Integer", "Long":
		return "Integer"
	case "Decimal":
		return "Decimal"
	case "Boolean":
		return "Boolean"
	case "DateTime":
		return "DateTime"
	default:
		return "String"
	}
}

// elementTypeForKind maps model Kind to BSON ElementType.
func elementTypeForKind(kind string) string {
	if kind == "Array" {
		return "Array"
	}
	if kind == "Value" {
		return "Value"
	}
	return "Object"
}

func serializeImportValueDataType(typeName string) bson.D {
	typeID := idToBsonBinary(GenerateID())
	switch typeName {
	case "Integer", "Long":
		return bson.D{
			{Key: "$ID", Value: typeID},
			{Key: "$Type", Value: "DataTypes$IntegerType"},
		}
	case "Decimal":
		return bson.D{
			{Key: "$ID", Value: typeID},
			{Key: "$Type", Value: "DataTypes$DecimalType"},
		}
	case "Boolean":
		return bson.D{
			{Key: "$ID", Value: typeID},
			{Key: "$Type", Value: "DataTypes$BooleanType"},
		}
	case "DateTime":
		return bson.D{
			{Key: "$ID", Value: typeID},
			{Key: "$Type", Value: "DataTypes$DateTimeType"},
		}
	case "Binary":
		return bson.D{
			{Key: "$ID", Value: typeID},
			{Key: "$Type", Value: "DataTypes$BinaryType"},
		}
	default: // String
		return bson.D{
			{Key: "$ID", Value: typeID},
			{Key: "$Type", Value: "DataTypes$StringType"},
		}
	}
}
