// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// ── ViewEntitySourceDocument ──────────────────────────

func (b *MprBackend) deleteViewEntitySourceDocumentViaModelsdk(id model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.DeleteUnit(string(id))
}

func (b *MprBackend) deleteViewEntitySourceDocumentByNameViaModelsdk(moduleName, docName string) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	docID, err := b.msdkReader.FindViewEntitySourceDocumentID(moduleName, docName)
	if err != nil {
		return fmt.Errorf("find view entity source document: %w", err)
	}
	if docID == "" {
		return nil
	}
	return b.msdkWriter.DeleteUnit(string(docID))
}

func (b *MprBackend) moveViewEntitySourceDocumentViaModelsdk(sourceModuleName string, targetModuleID model.ID, docName string) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	docID, err := b.msdkReader.FindViewEntitySourceDocumentID(sourceModuleName, docName)
	if err != nil {
		return fmt.Errorf("find view entity source document: %w", err)
	}
	if docID == "" {
		return nil
	}
	return b.msdkWriter.UpdateUnitContainer(string(docID), string(targetModuleID))
}

// ── CreateViewEntitySourceDocument ────────────────────

func (b *MprBackend) createViewEntitySourceDocumentViaModelsdk(moduleID model.ID, moduleName, docName, oqlQuery, documentation string) (model.ID, error) {
	if b.msdkWriter == nil {
		return "", fmt.Errorf("modelsdk writer not initialized")
	}
	doc := genDm.NewViewEntitySourceDocument()
	if doc.ID() == "" {
		doc.SetID(element.ID(modelsdkmpr.GenerateID()))
	}
	doc.SetName(docName)
	doc.SetDocumentation(documentation)
	doc.SetOql(oqlQuery)
	doc.SetExcluded(false)
	doc.SetExportLevel("Hidden")

	contents, err := b.newEncoder().Encode(doc)
	if err != nil {
		return "", fmt.Errorf("serialize ViewEntitySourceDocument: %w", err)
	}
	docID := model.ID(doc.ID())
	if err := b.insertUnit(string(docID), string(moduleID), "Documents", doc.TypeName(), contents); err != nil {
		return "", fmt.Errorf("insert ViewEntitySourceDocument: %w", err)
	}
	_ = moduleName
	return docID, nil
}

// ── UpdateOqlQueriesForMovedEntity ────────────────────

func (b *MprBackend) updateOqlQueriesForMovedEntityViaModelsdk(oldQualifiedName, newQualifiedName string) (int, error) {
	if b.msdkWriter == nil {
		return 0, fmt.Errorf("modelsdk writer not initialized")
	}
	patches, count, err := b.scanOqlQueryUpdatesCompat(oldQualifiedName, newQualifiedName)
	if err != nil {
		return 0, err
	}
	for _, p := range patches {
		if err := b.writeUnitContents(model.ID(p.ID), p.Contents); err != nil {
			return count, fmt.Errorf("write ViewEntitySourceDocument %s: %w", p.ID, err)
		}
	}
	return count, nil
}

// ── UpdateEnumerationRefsInAllDomainModels ────────────

func (b *MprBackend) updateEnumerationRefsInAllDomainModelsViaModelsdk(oldQualifiedName, newQualifiedName string) error {
	dms, err := b.ListDomainModelsGen()
	if err != nil {
		return fmt.Errorf("list domain models: %w", err)
	}
	for _, dm := range dms {
		changed := false
		for _, entityElem := range dm.EntitiesItems() {
			entity, ok := entityElem.(*genDm.Entity)
			if !ok || entity == nil {
				continue
			}
			for _, attrElem := range entity.AttributesItems() {
				attr, ok := attrElem.(*genDm.Attribute)
				if !ok || attr == nil {
					continue
				}
				if enumType, ok := attr.Type().(*genDm.EnumerationAttributeType); ok && enumType != nil {
					if enumType.EnumerationQualifiedName() == oldQualifiedName {
						enumType.SetEnumerationQualifiedName(newQualifiedName)
						changed = true
					}
				}
			}
		}
		if changed {
			if err := b.UpdateDomainModelGen(dm); err != nil {
				return fmt.Errorf("update domain model %s: %w", dm.ID(), err)
			}
		}
	}
	return nil
}
