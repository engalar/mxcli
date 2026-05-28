// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"github.com/mendixlabs/mxcli/model"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// SerializeImportMapping returns BSON bytes for an import mapping unit.
func SerializeImportMapping(im *model.ImportMapping) ([]byte, error) {
	elements := bson.A{int32(2)}
	for _, elem := range im.Elements {
		elements = append(elements, serImportMappingElement(elem, "(Object)"))
	}

	exportLevel := im.ExportLevel
	if exportLevel == "" {
		exportLevel = "Hidden"
	}

	// ParameterType is a required sub-document even when not used (DataTypes$UnknownType).
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
		"PublicName":                      "",
		"XsdRootElementName":              "",
		"MappingSourceReference":          nil,
		"ParameterType":                   parameterType,
		"OperationName":                   "",
		"ServiceName":                     "",
		"WsdlFile":                        "",
	}
	return bson.Marshal(doc)
}

func serImportMappingElement(elem *model.ImportMappingElement, parentPath string) bson.M {
	id := string(elem.ID)
	if id == "" {
		id = generateUUID()
	}

	if elem.Kind == "Object" || elem.Kind == "Array" {
		return serImportObjectElement(id, elem, parentPath)
	}
	return serImportValueElement(id, elem, parentPath)
}

func serImportObjectElement(id string, elem *model.ImportMappingElement, parentPath string) bson.M {
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
		children = append(children, serImportMappingElement(child, jsonPath))
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
		"ElementType":                       serElementTypeForKind(elem.Kind),
		"Documentation":                     "",
		"CustomHandlerCall":                 nil,
	}
}

func serImportValueElement(id string, elem *model.ImportMappingElement, parentPath string) bson.M {
	dataType := serMappingValueDataType(elem.DataType)
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
		"XmlPrimitiveType": serXmlPrimitiveTypeName(elem.DataType),
	}
}

// SerializeExportMapping returns BSON bytes for an export mapping unit.
func SerializeExportMapping(em *model.ExportMapping) ([]byte, error) {
	elements := bson.A{int32(2)}
	for _, elem := range em.Elements {
		elements = append(elements, serExportMappingElement(elem, "(Object)"))
	}

	exportLevel := em.ExportLevel
	if exportLevel == "" {
		exportLevel = "Hidden"
	}

	nullValueOption := em.NullValueOption
	if nullValueOption == "" {
		nullValueOption = "LeaveOutElement"
	}

	doc := bson.M{
		"$ID":               idToBsonBinary(string(em.ID)),
		"$Type":             "ExportMappings$ExportMapping",
		"Name":              em.Name,
		"Documentation":     em.Documentation,
		"Excluded":          em.Excluded,
		"ExportLevel":       exportLevel,
		"JsonStructure":     em.JsonStructure,
		"XmlSchema":         em.XmlSchema,
		"MessageDefinition": em.MessageDefinition,
		"NullValueOption":   nullValueOption,
		"Elements":          elements,
		// Required fields with defaults — verified against Studio Pro-created BSON
		"PublicName":             "",
		"XsdRootElementName":     "",
		"IsHeaderParameter":      false,
		"ParameterName":          "",
		"OperationName":          "",
		"ServiceName":            "",
		"WsdlFile":               "",
		"MappingSourceReference": nil,
	}
	return bson.Marshal(doc)
}

func serExportMappingElement(elem *model.ExportMappingElement, parentPath string) bson.M {
	id := string(elem.ID)
	if id == "" {
		id = generateUUID()
	}

	if elem.Kind == "Object" || elem.Kind == "Array" {
		return serExportObjectElement(id, elem, parentPath)
	}
	return serExportValueElement(id, elem, parentPath)
}

func serExportObjectElement(id string, elem *model.ExportMappingElement, parentPath string) bson.M {
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
		children = append(children, serExportMappingElement(child, jsonPath))
	}

	objectHandling := elem.ObjectHandling
	if objectHandling == "" {
		objectHandling = "Parameter"
	}

	maxOccurs := int32(elem.MaxOccurs)

	return bson.M{
		"$ID":                               idToBsonBinary(id),
		"$Type":                             "ExportMappings$ObjectMappingElement",
		"Entity":                            elem.Entity,
		"ExposedName":                       elem.ExposedName,
		"JsonPath":                          jsonPath,
		"XmlPath":                           "",
		"ObjectHandling":                    objectHandling,
		"ObjectHandlingBackup":              objectHandling,
		"ObjectHandlingBackupAllowOverride": false,
		"Association":                       elem.Association,
		"Children":                          children,
		"MinOccurs":                         int32(0),
		"MaxOccurs":                         maxOccurs,
		"Nillable":                          true,
		"IsDefaultType":                     false,
		"ElementType":                       serElementTypeForKind(elem.Kind),
		"Documentation":                     "",
		"CustomHandlerCall":                 nil,
	}
}

func serExportValueElement(id string, elem *model.ExportMappingElement, parentPath string) bson.M {
	dataType := serMappingValueDataType(elem.DataType)
	jsonPath := elem.JsonPath
	if jsonPath == "" {
		jsonPath = parentPath + "|" + elem.ExposedName
	}

	return bson.M{
		"$ID":              idToBsonBinary(id),
		"$Type":            "ExportMappings$ValueMappingElement",
		"Attribute":        elem.Attribute,
		"ExposedName":      elem.ExposedName,
		"JsonPath":         jsonPath,
		"XmlPath":          "",
		"Type":             dataType,
		"MinOccurs":        int32(0),
		"MaxOccurs":        int32(0),
		"Nillable":         true,
		"IsDefaultType":    false,
		"ElementType":      "Value",
		"Documentation":    "",
		"Converter":        "",
		"FractionDigits":   int32(-1),
		"TotalDigits":      int32(-1),
		"MaxLength":        int32(0),
		"IsContent":        false,
		"IsXmlAttribute":   false,
		"OriginalValue":    "",
		"XmlPrimitiveType": serXmlPrimitiveTypeName(elem.DataType),
	}
}

// serXmlPrimitiveTypeName maps a Mendix data type name to an XML primitive type name.
func serXmlPrimitiveTypeName(dataType string) string {
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

// serElementTypeForKind maps model Kind to BSON ElementType.
func serElementTypeForKind(kind string) string {
	if kind == "Array" {
		return "Array"
	}
	if kind == "Value" {
		return "Value"
	}
	return "Object"
}

// serMappingValueDataType converts a simple type name to a DataTypes$ BSON object.
func serMappingValueDataType(typeName string) bson.D {
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
