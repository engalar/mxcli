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
	w, ok := b.concreteWriter()
	if !ok {
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

	return mprrepos.NewEnumerationRepository(w).Create(
		string(enum.ContainerID), "Documents", genEnum,
	)
}

func (b *MprBackend) createConstantGen(c *model.Constant) error {
	if c.ID == "" {
		c.ID = model.ID(modelsdkmpr.GenerateID())
	}
	w, ok := b.concreteWriter()
	if !ok {
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

	return mprrepos.NewConstantRepository(w).Create(
		string(c.ContainerID), "Documents", genC,
	)
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
