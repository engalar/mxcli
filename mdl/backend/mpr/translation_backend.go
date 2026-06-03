// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
)

// SetEnumerationTranslation loads the enumeration unit, sets the langCode
// translation on the caption of the value named valueName, and saves the unit.
func (b *MprBackend) SetEnumerationTranslation(enumQN, valueName, langCode, text string) error {
	info, err := b.GetRawUnitByName("enumeration", enumQN)
	if err != nil {
		return fmt.Errorf("locate enumeration %q: %w", enumQN, err)
	}
	if info == nil {
		return mdlerrors.NewValidationf("enumeration %q not found", enumQN)
	}

	rawBytes, err := b.msdkReader.GetRawUnitBytes(info.ID)
	if err != nil {
		return fmt.Errorf("read enumeration unit: %w", err)
	}
	var enumDoc bson.D
	if err := bson.Unmarshal(rawBytes, &enumDoc); err != nil {
		return fmt.Errorf("unmarshal enumeration BSON: %w", err)
	}

	if !setEnumValueTranslation(enumDoc, valueName, langCode, text) {
		return mdlerrors.NewValidationf(
			"enumeration value %q not found in %q", valueName, enumQN)
	}

	contents, err := bson.Marshal(enumDoc)
	if err != nil {
		return fmt.Errorf("marshal enumeration BSON: %w", err)
	}
	return b.writeUnitContents(model.ID(info.ID), contents)
}

// ListTranslationNodes resolves docQN to a unit and walks its BSON for
// Texts$Text nodes, returning one TranslationNode per translatable field with
// every language's text. docType is accepted for interface symmetry but the
// lookup is type-agnostic (a qualified name is unique within the project).
func (b *MprBackend) ListTranslationNodes(docQN, docType string) ([]model.TranslationNode, error) {
	info, err := b.GetRawUnitByName("", docQN)
	if err != nil {
		return nil, fmt.Errorf("locate document %q: %w", docQN, err)
	}
	if info == nil {
		return nil, mdlerrors.NewValidationf("document %q not found", docQN)
	}

	rawBytes, err := b.msdkReader.GetRawUnitBytes(info.ID)
	if err != nil {
		return nil, fmt.Errorf("read document unit: %w", err)
	}
	var root bson.D
	if err := bson.Unmarshal(rawBytes, &root); err != nil {
		return nil, fmt.Errorf("unmarshal document BSON: %w", err)
	}

	docKind := docTypeLabelFromBsonType(dGetString(root, "$Type"))
	var nodes []model.TranslationNode
	collectTranslationNodes(root, "", docKind, &nodes)
	return nodes, nil
}

// docTypeLabelFromBsonType maps a unit's BSON $Type to the DocType label used in
// TranslationNode (PAGE, SNIPPET, ENUMERATION, WORKFLOW, MICROFLOW).
func docTypeLabelFromBsonType(bsonType string) string {
	switch {
	case strings.Contains(bsonType, "Snippet"):
		return "SNIPPET"
	case strings.Contains(bsonType, "Enumeration"):
		return "ENUMERATION"
	case strings.Contains(bsonType, "Workflow"):
		return "WORKFLOW"
	case strings.Contains(bsonType, "Microflow"):
		return "MICROFLOW"
	case strings.Contains(bsonType, "Forms$") || strings.Contains(bsonType, "Page"):
		return "PAGE"
	default:
		return ""
	}
}

// collectTranslationNodes walks a BSON document tree, appending a
// TranslationNode for every Texts$Text node it finds. namePrefix is the Name of
// the nearest named ancestor, used to build a readable path.
func collectTranslationNodes(doc bson.D, namePrefix, docKind string, out *[]model.TranslationNode) {
	// A named element refreshes the path prefix for its descendants.
	if name := dGetString(doc, "Name"); name != "" {
		namePrefix = name
	}

	for _, elem := range doc {
		switch v := elem.Value.(type) {
		case bson.D:
			if dGetString(v, "$Type") == "Texts$Text" {
				*out = append(*out, model.TranslationNode{
					Path:     translationNodePath(namePrefix, elem.Key),
					Property: strings.ToLower(elem.Key),
					DocType:  docKind,
					Texts:    extractTranslationsMap(v),
				})
				continue
			}
			collectTranslationNodes(v, namePrefix, docKind, out)
		case bson.A:
			for _, item := range v {
				if child, ok := item.(bson.D); ok {
					collectTranslationNodes(child, namePrefix, docKind, out)
				}
			}
		}
	}
}

// translationNodePath builds the "Owner.property" path, or just "property" when
// the node has no named ancestor (e.g. a page-level title).
func translationNodePath(namePrefix, key string) string {
	prop := strings.ToLower(key)
	if namePrefix == "" {
		return prop
	}
	return namePrefix + "." + prop
}

// extractTranslationsMap reads the langCode->text map from a Texts$Text node.
func extractTranslationsMap(textsDoc bson.D) map[string]string {
	texts := map[string]string{}
	for _, item := range extractBsonArray(dGet(textsDoc, "Items")) {
		tr, ok := item.(bson.D)
		if !ok {
			continue
		}
		lang := dGetString(tr, "LanguageCode")
		if lang == "" {
			continue
		}
		texts[lang] = dGetString(tr, "Text")
	}
	return texts
}
