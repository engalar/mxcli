// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/mpr"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
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
	docID, err := b.writer.FindViewEntitySourceDocumentID(moduleName, docName)
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
	docID, err := b.writer.FindViewEntitySourceDocumentID(sourceModuleName, docName)
	if err != nil {
		return fmt.Errorf("find view entity source document: %w", err)
	}
	if docID == "" {
		return nil
	}
	return b.msdkWriter.UpdateUnitContainer(string(docID), string(targetModuleID))
}

// writeDomainModel reads the domain model by ID, applies mutateFn, then writes
// back via modelsdk WriteTransaction (avoids sdk/mpr updateTransactionID).
func (b *MprBackend) writeDomainModel(domainModelID model.ID, mutateFn func(dm *domainmodel.DomainModel) error) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	dm, err := b.reader.GetDomainModelByID(domainModelID)
	if err != nil {
		return fmt.Errorf("read domain model: %w", err)
	}
	if err := mutateFn(dm); err != nil {
		return err
	}
	return b.updateDomainModelViaModelsdk(dm)
}

// ── CreateEntity ─────────────────────────────────────

func (b *MprBackend) createEntityViaModelsdk(domainModelID model.ID, entity *domainmodel.Entity) error {
	return b.writeDomainModel(domainModelID, func(dm *domainmodel.DomainModel) error {
		if entity.ID == "" {
			entity.ID = model.ID(mpr.GenerateID())
		}
		entity.TypeName = "DomainModels$Entity"
		entity.ContainerID = domainModelID
		for _, attr := range entity.Attributes {
			if attr.ID == "" {
				attr.ID = model.ID(mpr.GenerateID())
			}
			attr.TypeName = "DomainModels$Attribute"
			attr.ContainerID = entity.ID
		}
		dm.Entities = append(dm.Entities, entity)
		return nil
	})
}

// ── UpdateEntity ─────────────────────────────────────

func (b *MprBackend) updateEntityViaModelsdk(domainModelID model.ID, entity *domainmodel.Entity) error {
	return b.writeDomainModel(domainModelID, func(dm *domainmodel.DomainModel) error {
		for i, e := range dm.Entities {
			if e.ID == entity.ID {
				dm.Entities[i] = entity
				return nil
			}
		}
		return fmt.Errorf("entity not found: %s", entity.ID)
	})
}

// ── AddAttribute ──────────────────────────────────────

func (b *MprBackend) addAttributeViaModelsdk(domainModelID, entityID model.ID, attr *domainmodel.Attribute) error {
	return b.writeDomainModel(domainModelID, func(dm *domainmodel.DomainModel) error {
		for _, e := range dm.Entities {
			if e.ID == entityID {
				if attr.ID == "" {
					attr.ID = model.ID(mpr.GenerateID())
				}
				attr.TypeName = "DomainModels$Attribute"
				attr.ContainerID = entityID
				e.Attributes = append(e.Attributes, attr)
				return nil
			}
		}
		return fmt.Errorf("entity not found: %s", entityID)
	})
}

// ── UpdateAttribute ───────────────────────────────────

func (b *MprBackend) updateAttributeViaModelsdk(domainModelID, entityID model.ID, attr *domainmodel.Attribute) error {
	return b.writeDomainModel(domainModelID, func(dm *domainmodel.DomainModel) error {
		for _, e := range dm.Entities {
			if e.ID == entityID {
				for i, a := range e.Attributes {
					if a.ID == attr.ID {
						e.Attributes[i] = attr
						return nil
					}
				}
				return fmt.Errorf("attribute not found: %s", attr.ID)
			}
		}
		return fmt.Errorf("entity not found: %s", entityID)
	})
}

// ── CreateAssociation ─────────────────────────────────

func (b *MprBackend) createAssociationViaModelsdk(domainModelID model.ID, assoc *domainmodel.Association) error {
	return b.writeDomainModel(domainModelID, func(dm *domainmodel.DomainModel) error {
		if assoc.ID == "" {
			assoc.ID = model.ID(mpr.GenerateID())
		}
		assoc.TypeName = "DomainModels$Association"
		assoc.ContainerID = domainModelID
		dm.Associations = append(dm.Associations, assoc)
		return nil
	})
}

// ── CreateCrossAssociation ────────────────────────────

func (b *MprBackend) createCrossAssociationViaModelsdk(domainModelID model.ID, ca *domainmodel.CrossModuleAssociation) error {
	return b.writeDomainModel(domainModelID, func(dm *domainmodel.DomainModel) error {
		if ca.ID == "" {
			ca.ID = model.ID(mpr.GenerateID())
		}
		ca.TypeName = "DomainModels$CrossAssociation"
		ca.ContainerID = domainModelID
		dm.CrossAssociations = append(dm.CrossAssociations, ca)
		return nil
	})
}

// ── CreateViewEntitySourceDocument ────────────────────

func (b *MprBackend) createViewEntitySourceDocumentViaModelsdk(moduleID model.ID, moduleName, docName, oqlQuery, documentation string) (model.ID, error) {
	if b.msdkWriter == nil {
		return "", fmt.Errorf("modelsdk writer not initialized")
	}
	docID := model.ID(mpr.GenerateID())
	doc := bson.D{
		{Key: "$ID", Value: mpr.IDToBsonBinary(string(docID))},
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

// ── MoveEntity ────────────────────────────────────────

func (b *MprBackend) moveEntityViaModelsdk(entity *domainmodel.Entity, sourceDMID, targetDMID model.ID, sourceModuleName, targetModuleName string) ([]string, error) {
	if b.msdkWriter == nil {
		return nil, fmt.Errorf("modelsdk writer not initialized")
	}

	sourceDM, err := b.reader.GetDomainModelByID(sourceDMID)
	if err != nil {
		return nil, fmt.Errorf("load source domain model: %w", err)
	}

	found := false
	for i, e := range sourceDM.Entities {
		if e.ID == entity.ID {
			sourceDM.Entities = append(sourceDM.Entities[:i], sourceDM.Entities[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("entity not found in source domain model: %s", entity.ID)
	}

	targetDM, err := b.reader.GetDomainModelByID(targetDMID)
	if err != nil {
		return nil, fmt.Errorf("load target domain model: %w", err)
	}

	var convertedAssocs []string
	var keptAssocs []*domainmodel.Association
	for _, a := range sourceDM.Associations {
		if a.ChildID == entity.ID {
			ca := &domainmodel.CrossModuleAssociation{}
			ca.ID = a.ID
			ca.TypeName = "DomainModels$CrossAssociation"
			ca.ContainerID = sourceDMID
			ca.Name = a.Name
			ca.Documentation = a.Documentation
			ca.ParentID = a.ParentID
			ca.ChildRef = targetModuleName + "." + entity.Name
			ca.Type = a.Type
			ca.Owner = a.Owner
			ca.StorageFormat = a.StorageFormat
			ca.ParentDeleteBehavior = a.ParentDeleteBehavior
			ca.ChildDeleteBehavior = a.ChildDeleteBehavior
			sourceDM.CrossAssociations = append(sourceDM.CrossAssociations, ca)
			convertedAssocs = append(convertedAssocs, a.Name)
		} else if a.ParentID == entity.ID {
			var childEntityName string
			for _, e := range sourceDM.Entities {
				if e.ID == a.ChildID {
					childEntityName = e.Name
					break
				}
			}
			ca := &domainmodel.CrossModuleAssociation{}
			ca.ID = a.ID
			ca.TypeName = "DomainModels$CrossAssociation"
			ca.ContainerID = targetDMID
			ca.Name = a.Name
			ca.Documentation = a.Documentation
			ca.ParentID = a.ParentID
			ca.ChildRef = sourceModuleName + "." + childEntityName
			ca.Type = a.Type
			ca.Owner = a.Owner
			ca.StorageFormat = a.StorageFormat
			ca.ParentDeleteBehavior = a.ParentDeleteBehavior
			ca.ChildDeleteBehavior = a.ChildDeleteBehavior
			targetDM.CrossAssociations = append(targetDM.CrossAssociations, ca)
			convertedAssocs = append(convertedAssocs, a.Name)
		} else {
			keptAssocs = append(keptAssocs, a)
		}
	}
	sourceDM.Associations = keptAssocs

	oldPrefix := sourceModuleName + "."
	newPrefix := targetModuleName + "."
	for _, vr := range entity.ValidationRules {
		attrIDStr := string(vr.AttributeID)
		if strings.HasPrefix(attrIDStr, oldPrefix) {
			vr.AttributeID = model.ID(newPrefix + attrIDStr[len(oldPrefix):])
		}
	}
	if entity.Source == "DomainModels$OqlViewEntitySource" && entity.SourceDocumentRef != "" {
		if strings.HasPrefix(entity.SourceDocumentRef, oldPrefix) {
			entity.SourceDocumentRef = newPrefix + entity.SourceDocumentRef[len(oldPrefix):]
		}
	}

	if err := b.updateDomainModelViaModelsdk(sourceDM); err != nil {
		return nil, fmt.Errorf("update source domain model: %w", err)
	}

	entity.ContainerID = targetDMID
	targetDM.Entities = append(targetDM.Entities, entity)
	return convertedAssocs, b.updateDomainModelViaModelsdk(targetDM)
}

// ── UpdateOqlQueriesForMovedEntity ────────────────────

func (b *MprBackend) updateOqlQueriesForMovedEntityViaModelsdk(oldQualifiedName, newQualifiedName string) (int, error) {
	patches, count, err := b.writer.ScanOqlQueryUpdates(oldQualifiedName, newQualifiedName)
	if err != nil {
		return 0, err
	}
	for _, p := range patches {
		if err := b.msdkWriteRaw(model.ID(p.ID), p.Contents); err != nil {
			return count, fmt.Errorf("write ViewEntitySourceDocument %s: %w", p.ID, err)
		}
	}
	return count, nil
}

// ── UpdateEnumerationRefsInAllDomainModels ────────────

func (b *MprBackend) updateEnumerationRefsInAllDomainModelsViaModelsdk(oldQualifiedName, newQualifiedName string) error {
	dms, err := b.reader.ListDomainModels()
	if err != nil {
		return fmt.Errorf("list domain models: %w", err)
	}
	for _, dm := range dms {
		changed := false
		for _, entity := range dm.Entities {
			for _, attr := range entity.Attributes {
				if enumType, ok := attr.Type.(*domainmodel.EnumerationAttributeType); ok {
					if enumType.EnumerationRef == oldQualifiedName {
						enumType.EnumerationRef = newQualifiedName
						enumType.EnumerationID = model.ID(newQualifiedName)
						changed = true
					}
				}
			}
		}
		if changed {
			if err := b.updateDomainModelViaModelsdk(dm); err != nil {
				return fmt.Errorf("update domain model %s: %w", dm.ID, err)
			}
		}
	}
	return nil
}
