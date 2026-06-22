// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDt "github.com/mendixlabs/mxcli/modelsdk/gen/datatypes"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	genTexts "github.com/mendixlabs/mxcli/modelsdk/gen/texts"
)

// ============================================================================
// Gen-native helper functions (V3 builder support)
// ============================================================================

// genElementToBSONDoc encodes a gen element to bson.D via the backend serializer.
// Routes through WidgetSerializationBackend so executor does not import modelsdk/codec directly.
func (pb *pageBuilder) genElementToBSONDoc(elem element.Element) (bson.D, error) {
	raw, err := pb.serializationBackend.SerializePageGenElement(elem)
	if err != nil {
		return nil, err
	}
	var doc bson.D
	return doc, bson.Unmarshal(raw, &doc)
}

// genSimpleText creates a gen Text element with a single en_US translation.
func genSimpleText(text string) *genTexts.Text {
	t := genTexts.NewText()
	assignFreshID(t)
	tr := genTexts.NewTranslation()
	assignFreshID(tr)
	tr.SetLanguageCode("en_US")
	tr.SetText(text)
	t.AddTranslations(tr)
	return t
}

// genSimpleLabel creates a gen ClientTemplate (used as a label/caption) with a
// single en_US translation. This matches the Mendix pattern for caption/title
// properties that take a ClientTemplate element.
func genSimpleLabel(text string) element.Element {
	ct := genPg.NewClientTemplate()
	assignFreshID(ct)
	tmpl := genSimpleText(text)
	ct.SetTemplate(tmpl)
	fallback := genTexts.NewText()
	assignFreshID(fallback)
	ct.SetFallback(fallback)
	return ct
}

// genPageParamType creates the appropriate gen DataType element for a page
// parameter based on the MDL AST data type.
func genPageParamType(dt ast.DataType, entityQN string) element.Element {
	switch dt.Kind {
	case ast.TypeString:
		return genPg.NewTextBox() // placeholder — we use direct type string in buildPageV3
	}
	return nil
}

// ============================================================================
// Type conversion helpers (V3 builder)
// ============================================================================

// newPrimPageParamType returns a fresh gen DataType element for a primitive page
// parameter type (e.g. "DataTypes$StringType"). Returns nil for unknown type strings.
// Callers must call assignFreshID on the returned element before use.
func newPrimPageParamType(bsonTypeName string) element.Element {
	switch bsonTypeName {
	case "DataTypes$StringType":
		return genDt.NewStringType()
	case "DataTypes$IntegerType":
		return genDt.NewIntegerType()
	case "DataTypes$DecimalType":
		return genDt.NewDecimalType()
	case "DataTypes$BooleanType":
		return genDt.NewBooleanType()
	case "DataTypes$DateTimeType":
		return genDt.NewDateTimeType()
	default:
		// DataTypes$LongType and others: fall back to nil (caller skips SetParameterType).
		return nil
	}
}

// extractEntityFromGenReturnType peels the "List of " prefix off a
// gen-rendered return-type string and returns the bare entity QN.
func extractEntityFromGenReturnType(rt string) string {
	rt = strings.TrimSpace(rt)
	if rt == "" || rt == "Void" {
		return ""
	}
	if after, ok := strings.CutPrefix(rt, "List of "); ok {
		return after
	}
	if isPrimitiveReturnType(rt) {
		return ""
	}
	if strings.Contains(rt, ".") {
		return rt
	}
	return ""
}

func isPrimitiveReturnType(rt string) bool {
	switch rt {
	case "Boolean", "Integer", "Long", "Decimal", "String", "DateTime", "Date", "Binary":
		return true
	}
	return false
}

// pageParamBSONType maps a DataType to the BSON $Type string for primitive page parameters.
// Returns empty string for entity/enum types (which use DataTypes$ObjectType instead).
func pageParamBSONType(dt ast.DataType) string {
	switch dt.Kind {
	case ast.TypeString:
		return "DataTypes$StringType"
	case ast.TypeInteger:
		return "DataTypes$IntegerType"
	case ast.TypeLong:
		return "DataTypes$LongType"
	case ast.TypeDecimal:
		return "DataTypes$DecimalType"
	case ast.TypeBoolean:
		return "DataTypes$BooleanType"
	case ast.TypeDateTime:
		return "DataTypes$DateTimeType"
	default:
		return ""
	}
}

// mdlTypeToBsonType converts an MDL type name to a BSON DataTypes$* type string.
func mdlTypeToBsonType(mdlType string) string {
	switch strings.ToLower(mdlType) {
	case "boolean":
		return "DataTypes$BooleanType"
	case "string":
		return "DataTypes$StringType"
	case "integer":
		return "DataTypes$IntegerType"
	case "long":
		return "DataTypes$LongType"
	case "decimal":
		return "DataTypes$DecimalType"
	case "datetime", "date":
		return "DataTypes$DateTimeType"
	default:
		return "DataTypes$ObjectType"
	}
}

// mdlTypeToDataTypeElement creates a gen DataTypes element for use as LocalVariable.VariableType.
// Studio Pro requires VariableType to be a nested object ($Type + $ID), not a plain string.
func mdlTypeToDataTypeElement(mdlType string) element.Element {
	bsonType := mdlTypeToBsonType(mdlType)
	dt := newPrimPageParamType(bsonType)
	if dt == nil {
		// Fallback for types not in newPrimPageParamType (e.g. Long): use ObjectType.
		dt = genDt.NewObjectType()
	}
	if withID, ok := dt.(genElementWithID); ok {
		assignFreshID(withID)
	}
	return dt
}

// bsonTypeToMDLType converts a BSON DataTypes$* type to an MDL type name.
func bsonTypeToMDLType(bsonType string) string {
	switch bsonType {
	case "DataTypes$BooleanType":
		return "Boolean"
	case "DataTypes$StringType":
		return "String"
	case "DataTypes$IntegerType":
		return "Integer"
	case "DataTypes$LongType":
		return "Long"
	case "DataTypes$DecimalType":
		return "Decimal"
	case "DataTypes$DateTimeType":
		return "DateTime"
	case "DataTypes$ObjectType":
		return "Object"
	default:
		return "Unknown"
	}
}

// resolveDesignPropertyValueType determines the correct ValueType for a design property
// based on the theme definition. ToggleButtonGroup and ColorPicker use "custom" type;
// Dropdown uses "option" type. Falls back to "option" if theme info is unavailable.
func resolveDesignPropertyValueType(key string, themeProps []ThemeProperty) string {
	for _, tp := range themeProps {
		if tp.Name == key {
			switch tp.Type {
			case "ToggleButtonGroup", "ColorPicker":
				return "custom"
			default:
				return "option"
			}
		}
	}
	return "option"
}
