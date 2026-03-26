// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"

	"go.mongodb.org/mongo-driver/bson"
)

// CreateEnumeration creates a new enumeration.
func (w *Writer) CreateEnumeration(enum *model.Enumeration) error {
	if enum.ID == "" {
		enum.ID = model.ID(generateUUID())
	}
	enum.TypeName = "Enumerations$Enumeration"

	contents, err := w.serializeEnumeration(enum)
	if err != nil {
		return fmt.Errorf("failed to serialize enumeration: %w", err)
	}

	return w.insertUnit(string(enum.ID), string(enum.ContainerID), "Documents", "Enumerations$Enumeration", contents)
}

// UpdateEnumeration updates an existing enumeration.
func (w *Writer) UpdateEnumeration(enum *model.Enumeration) error {
	contents, err := w.serializeEnumeration(enum)
	if err != nil {
		return fmt.Errorf("failed to serialize enumeration: %w", err)
	}

	return w.updateUnit(string(enum.ID), contents)
}

// MoveEnumeration moves an enumeration to a new container (module or folder).
// Only updates the ContainerID in the database, preserving all BSON content as-is.
func (w *Writer) MoveEnumeration(enum *model.Enumeration) error {
	return w.moveUnitByID(string(enum.ID), string(enum.ContainerID))
}

// DeleteEnumeration deletes an enumeration.
func (w *Writer) DeleteEnumeration(id model.ID) error {
	return w.deleteUnit(string(id))
}

// MoveConstant moves a constant to a new container (module or folder).
func (w *Writer) MoveConstant(constant *model.Constant) error {
	return w.moveUnitByID(string(constant.ID), string(constant.ContainerID))
}

// CreateConstant creates a new constant.
func (w *Writer) CreateConstant(constant *model.Constant) error {
	if constant.ID == "" {
		constant.ID = model.ID(generateUUID())
	}
	constant.TypeName = "Constants$Constant"

	contents, err := w.serializeConstant(constant)
	if err != nil {
		return fmt.Errorf("failed to serialize constant: %w", err)
	}

	return w.insertUnit(string(constant.ID), string(constant.ContainerID), "Documents", "Constants$Constant", contents)
}

// UpdateConstant updates an existing constant.
func (w *Writer) UpdateConstant(constant *model.Constant) error {
	contents, err := w.serializeConstant(constant)
	if err != nil {
		return fmt.Errorf("failed to serialize constant: %w", err)
	}

	return w.updateUnit(string(constant.ID), contents)
}

// DeleteConstant deletes a constant.
func (w *Writer) DeleteConstant(id model.ID) error {
	return w.deleteUnit(string(id))
}
func (w *Writer) serializeEnumeration(enum *model.Enumeration) ([]byte, error) {
	values := bson.A{int32(3)} // Version prefix
	for _, v := range enum.Values {
		valueID := string(v.ID)
		if valueID == "" {
			valueID = generateUUID()
		}
		captionID := generateUUID()

		// Build translation items
		translationItems := bson.A{int32(3)}
		if v.Caption != nil {
			for langCode, text := range v.Caption.Translations {
				translationItems = append(translationItems, bson.M{
					"$ID":          idToBsonBinary(generateUUID()),
					"$Type":        "Texts$Translation",
					"LanguageCode": langCode,
					"Text":         text,
				})
			}
		}

		valueDoc := bson.M{
			"$ID":   idToBsonBinary(valueID),
			"$Type": "Enumerations$EnumerationValue",
			"Name":  v.Name,
			"Caption": bson.M{
				"$ID":   idToBsonBinary(captionID),
				"$Type": "Texts$Text",
				"Items": translationItems,
			},
			"Image":       "",
			"RemoteValue": nil,
		}
		values = append(values, valueDoc)
	}

	doc := bson.M{
		"$ID":           idToBsonBinary(string(enum.ID)),
		"$Type":         "Enumerations$Enumeration",
		"Name":          enum.Name,
		"Documentation": enum.Documentation,
		"Excluded":      false,
		"ExportLevel":   "Hidden",
		"RemoteSource":  nil,
		"Values":        values,
	}
	return bson.Marshal(doc)
}

func (w *Writer) serializeConstant(constant *model.Constant) ([]byte, error) {
	doc := bson.M{
		"$ID":             idToBsonBinary(string(constant.ID)),
		"$Type":           "Constants$Constant",
		"Name":            constant.Name,
		"Documentation":   constant.Documentation,
		"Type":            serializeConstantDataType(constant.Type),
		"DefaultValue":    constant.DefaultValue,
		"ExposedToClient": constant.ExposedToClient,
		"Excluded":        constant.Excluded,
		"ExportLevel":     constant.ExportLevel,
	}
	return bson.Marshal(doc)
}

// serializeConstantDataType converts a ConstantDataType to BSON.
func serializeConstantDataType(dt model.ConstantDataType) bson.D {
	typeID := idToBsonBinary(GenerateID())

	switch dt.Kind {
	case "String":
		return bson.D{
			{Key: "$ID", Value: typeID},
			{Key: "$Type", Value: "DataTypes$StringType"},
		}
	case "Integer":
		return bson.D{
			{Key: "$ID", Value: typeID},
			{Key: "$Type", Value: "DataTypes$IntegerType"},
		}
	case "Long":
		return bson.D{
			{Key: "$ID", Value: typeID},
			{Key: "$Type", Value: "DataTypes$LongType"},
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
	case "Float":
		return bson.D{
			{Key: "$ID", Value: typeID},
			{Key: "$Type", Value: "DataTypes$FloatType"},
		}
	case "Enumeration":
		return bson.D{
			{Key: "$ID", Value: typeID},
			{Key: "$Type", Value: "DataTypes$EnumerationType"},
			{Key: "Enumeration", Value: dt.EnumRef},
		}
	case "Object":
		return bson.D{
			{Key: "$ID", Value: typeID},
			{Key: "$Type", Value: "DataTypes$ObjectType"},
			{Key: "Entity", Value: dt.EntityRef},
		}
	case "List":
		return bson.D{
			{Key: "$ID", Value: typeID},
			{Key: "$Type", Value: "DataTypes$ListType"},
			{Key: "Entity", Value: dt.EntityRef},
		}
	default:
		// Default to string type
		return bson.D{
			{Key: "$ID", Value: typeID},
			{Key: "$Type", Value: "DataTypes$StringType"},
		}
	}
}
