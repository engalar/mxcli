// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mendixlabs/mxcli/model"
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
	docID := model.ID(modelsdkmpr.GenerateID())
	doc := bson.D{
		{Key: "$ID", Value: modelsdkmpr.IDToBsonBinary(string(docID))},
		{Key: "$Type", Value: "DomainModels$ViewEntitySourceDocument"},
		{Key: "Documentation", Value: documentation},
		{Key: "Excluded", Value: false},
		{Key: "ExportLevel", Value: "Hidden"},
		{Key: "Name", Value: docName},
		{Key: "Oql", Value: oqlQuery},
	}
	contents, err := bson.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("serialize ViewEntitySourceDocument: %w", err)
	}
	if err := b.msdkWriter.InsertUnit(string(docID), string(moduleID), "Documents", "DomainModels$ViewEntitySourceDocument", contents); err != nil {
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
