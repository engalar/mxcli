// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	msdkconstants "github.com/mendixlabs/mxcli/modelsdk/gen/constants"
	msdkdatatypes "github.com/mendixlabs/mxcli/modelsdk/gen/datatypes"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

func (b *MprBackend) updateConstantViaModelsdk(constant *model.Constant) error {
	return b.msdkWrite(constant.ID, func(elem element.Element) error {
		c, ok := elem.(*msdkconstants.Constant)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *Constant)", elem)
		}
		c.SetName(constant.Name)
		c.SetDocumentation(constant.Documentation)
		c.SetDefaultValue(constant.DefaultValue)
		c.SetExposedToClient(constant.ExposedToClient)
		c.SetExcluded(constant.Excluded)
		if constant.ExportLevel != "" {
			c.SetExportLevel(constant.ExportLevel)
		}
		c.SetType(buildConstantTypeElement(constant.Type))
		return nil
	})
}

// buildConstantTypeElement maps a model.ConstantDataType to a DataTypes$* element.
// Mirrors sdk/mpr.serializeConstantDataType.
func buildConstantTypeElement(dt model.ConstantDataType) element.Element {
	id := element.ID(modelsdkmpr.GenerateID())
	switch dt.Kind {
	case "String":
		t := msdkdatatypes.NewStringType()
		t.SetID(id)
		return t
	case "Integer", "Long":
		// Mendix uses IntegerType for both Integer and Long in BSON storage.
		t := msdkdatatypes.NewIntegerType()
		t.SetID(id)
		return t
	case "Decimal":
		t := msdkdatatypes.NewDecimalType()
		t.SetID(id)
		return t
	case "Boolean":
		t := msdkdatatypes.NewBooleanType()
		t.SetID(id)
		return t
	case "DateTime", "Date":
		t := msdkdatatypes.NewDateTimeType()
		t.SetID(id)
		return t
	case "Binary":
		t := msdkdatatypes.NewBinaryType()
		t.SetID(id)
		return t
	case "Float":
		t := msdkdatatypes.NewFloatType()
		t.SetID(id)
		return t
	case "Enumeration":
		t := msdkdatatypes.NewEnumerationType()
		t.SetID(id)
		t.SetEnumerationQualifiedName(dt.EnumRef)
		return t
	case "Object":
		t := msdkdatatypes.NewObjectType()
		t.SetID(id)
		t.SetEntityQualifiedName(dt.EntityRef)
		return t
	case "List":
		t := msdkdatatypes.NewListType()
		t.SetID(id)
		t.SetEntityQualifiedName(dt.EntityRef)
		return t
	default:
		t := msdkdatatypes.NewStringType()
		t.SetID(id)
		return t
	}
}

func (b *MprBackend) moveConstantViaModelsdk(constant *model.Constant) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.UpdateUnitContainer(string(constant.ID), string(constant.ContainerID))
}

func (b *MprBackend) deleteConstantViaModelsdk(id model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.DeleteUnit(string(id))
}
