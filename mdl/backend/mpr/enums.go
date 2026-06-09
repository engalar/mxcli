// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"
	"sort"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	msdkenums "github.com/mendixlabs/mxcli/modelsdk/gen/enumerations"
	msdktexts "github.com/mendixlabs/mxcli/modelsdk/gen/texts"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

func (b *MprBackend) updateEnumerationViaModelsdk(enum *model.Enumeration) error {
	return b.msdkWrite(enum.ID, func(elem element.Element) error {
		e, ok := elem.(*msdkenums.Enumeration)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *Enumeration)", elem)
		}
		e.SetName(enum.Name)
		e.SetDocumentation(enum.Documentation)

		existing := e.ValuesItems()
		for i := len(existing) - 1; i >= 0; i-- {
			e.RemoveValues(i)
		}
		for _, v := range enum.Values {
			ev := msdkenums.NewEnumerationValue()
			valueID := string(v.ID)
			if valueID == "" {
				valueID = modelsdkmpr.GenerateID()
			}
			ev.SetID(element.ID(valueID))
			ev.SetName(v.Name)
			ev.SetCaption(buildCaptionText(v.Caption))
			ev.SetImageQualifiedName("")
			e.AddValues(ev)
		}
		return nil
	})
}

// buildCaptionText constructs a Texts$Text element from a model.Text.
// Always returns a Text element (with empty translations if caption is nil)
// to match sdk/mpr serializeEnumeration's always-emit-Caption behavior.
func buildCaptionText(caption *model.Text) element.Element {
	t := msdktexts.NewText()
	t.SetID(element.ID(modelsdkmpr.GenerateID()))
	if caption == nil {
		return t
	}
	langs := make([]string, 0, len(caption.Translations))
	for lang := range caption.Translations {
		langs = append(langs, lang)
	}
	sort.Strings(langs)
	for _, lang := range langs {
		tr := msdktexts.NewTranslation()
		tr.SetID(element.ID(modelsdkmpr.GenerateID()))
		tr.SetLanguageCode(lang)
		tr.SetText(caption.Translations[lang])
		t.AddTranslations(tr)
	}
	return t
}

func (b *MprBackend) moveEnumerationViaModelsdk(enum *model.Enumeration) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.UpdateUnitContainer(string(enum.ID), string(enum.ContainerID))
}

func (b *MprBackend) deleteEnumerationViaModelsdk(id model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.DeleteUnit(string(id))
}
