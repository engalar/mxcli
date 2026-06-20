// SPDX-License-Identifier: Apache-2.0

// Gen-native helpers for creating Enumeration and Constant units.
// Replaces mpr.SerializeEnumeration / mpr.SerializeConstant for create path.

package mprbackend

import (
	"fmt"
	"sort"

	mprrepos "github.com/mendixlabs/mxcli/mdl/backend/mpr/repos"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genCo "github.com/mendixlabs/mxcli/modelsdk/gen/constants"
	genDT "github.com/mendixlabs/mxcli/modelsdk/gen/datatypes"
	genEn "github.com/mendixlabs/mxcli/modelsdk/gen/enumerations"
	genTexts "github.com/mendixlabs/mxcli/modelsdk/gen/texts"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

func (b *MprBackend) createEnumerationGen(enum *model.Enumeration) error {
	if enum.ID == "" {
		enum.ID = model.ID(modelsdkmpr.GenerateID())
	}
	if b.writer == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	genEnum := genEn.NewEnumeration()
	genEnum.SetID(element.ID(string(enum.ID)))
	genEnum.SetName(enum.Name)
	genEnum.SetDocumentation(enum.Documentation)

	for _, v := range enum.Values {
		val := genEn.NewEnumerationValue()
		if v.ID != "" {
			val.SetID(element.ID(string(v.ID)))
		} else {
			val.SetID(element.ID(modelsdkmpr.GenerateID()))
		}
		val.SetName(v.Name)

		if v.Caption != nil && len(v.Caption.Translations) > 0 {
			txt := genTexts.NewText()
			txt.SetID(element.ID(modelsdkmpr.GenerateID()))
			langs := make([]string, 0, len(v.Caption.Translations))
			for lang := range v.Caption.Translations {
				langs = append(langs, lang)
			}
			sort.Strings(langs)
			for _, lang := range langs {
				tr := genTexts.NewTranslation()
				tr.SetID(element.ID(modelsdkmpr.GenerateID()))
				tr.SetLanguageCode(lang)
				tr.SetText(v.Caption.Translations[lang])
				txt.AddTranslations(tr)
			}
			val.SetCaption(txt)
		}
		genEnum.AddValues(val)
	}

	if err := mprrepos.NewEnumerationRepository(b.writer).Create(
		string(enum.ContainerID), "Documents", genEnum,
	); err != nil {
		return err
	}
	// The writer owns a private Reader distinct from b.msdkReader; invalidate
	// b.msdkReader's unit cache so that subsequent ListEnumerations calls within
	// the same exec session see the newly created unit.
	b.msdkReader.InvalidateCache()
	return nil
}

func (b *MprBackend) createConstantGen(c *model.Constant) error {
	if c.ID == "" {
		c.ID = model.ID(modelsdkmpr.GenerateID())
	}
	if b.writer == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	genC := genCo.NewConstant()
	genC.SetID(element.ID(string(c.ID)))
	genC.SetName(c.Name)
	genC.SetDocumentation(c.Documentation)
	genC.SetExcluded(c.Excluded)
	if c.ExportLevel != "" {
		genC.SetExportLevel(c.ExportLevel)
	}
	genC.SetDefaultValue(c.DefaultValue)
	genC.SetExposedToClient(c.ExposedToClient)

	typeElem, err := constantDataTypeToGen(c.Type)
	if err != nil {
		return err
	}
	genC.SetType(typeElem)

	if err := mprrepos.NewConstantRepository(b.writer).Create(
		string(c.ContainerID), "Documents", genC,
	); err != nil {
		return err
	}
	b.msdkReader.InvalidateCache()
	return nil
}

func constantDataTypeToGen(dt model.ConstantDataType) (element.Element, error) {
	switch dt.Kind {
	case "String":
		return genDT.NewStringType(), nil
	case "Integer":
		return genDT.NewIntegerType(), nil
	case "Long":
		// Mendix uses IntegerType BSON for both Integer and Long — LongType
		// does not exist in the metamodel type cache (see sdk/mpr comment).
		return genDT.NewIntegerType(), nil
	case "Decimal":
		return genDT.NewDecimalType(), nil
	case "Boolean":
		return genDT.NewBooleanType(), nil
	case "DateTime":
		return genDT.NewDateTimeType(), nil
	case "Enumeration":
		e := genDT.NewEnumerationType()
		e.SetEnumerationQualifiedName(dt.EnumRef)
		return e, nil
	default:
		return nil, fmt.Errorf("unsupported constant data type: %q", dt.Kind)
	}
}
