// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

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
