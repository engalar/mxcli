// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
)

// DomainModelBackend provides domain model, entity, attribute, and
// association operations.
type DomainModelBackend interface {
	// Domain models
	ListDomainModels() ([]*domainmodel.DomainModel, error)
	GetDomainModel(moduleID model.ID) (*domainmodel.DomainModel, error)
	GetDomainModelByID(id model.ID) (*domainmodel.DomainModel, error)
	UpdateDomainModel(dm *domainmodel.DomainModel) error

	// Stage 3.3.4 C1 — gen-typed read methods (additive).
	ListDomainModelsGen() ([]*genDm.DomainModel, error)
	GetDomainModelGen(moduleID model.ID) (*genDm.DomainModel, error)
	GetDomainModelByIDGen(id model.ID) (*genDm.DomainModel, error)
	UpdateDomainModelGen(dm *genDm.DomainModel) error
	// Stage 3.3.4 D8 — gen-typed entity write methods. The bridge
	// implementation in MprBackend converts to sdk types and delegates
	// to *ViaModelsdk write helpers (Stage 4 will retire the bridge).
	CreateEntityGen(domainModelID model.ID, entity *genDm.Entity) error
	UpdateEntityGen(domainModelID model.ID, entity *genDm.Entity) error
	CreateAssociationGen(domainModelID model.ID, assoc *genDm.Association) error

	// Entities
	CreateEntity(domainModelID model.ID, entity *domainmodel.Entity) error
	UpdateEntity(domainModelID model.ID, entity *domainmodel.Entity) error
	DeleteEntity(domainModelID model.ID, entityID model.ID) error
	MoveEntity(entity *domainmodel.Entity, sourceDMID, targetDMID model.ID, sourceModuleName, targetModuleName string) ([]string, error)

	// Attributes
	AddAttribute(domainModelID model.ID, entityID model.ID, attr *domainmodel.Attribute) error
	UpdateAttribute(domainModelID model.ID, entityID model.ID, attr *domainmodel.Attribute) error
	DeleteAttribute(domainModelID model.ID, entityID model.ID, attrID model.ID) error

	// Associations
	CreateAssociation(domainModelID model.ID, assoc *domainmodel.Association) error
	CreateCrossAssociation(domainModelID model.ID, ca *domainmodel.CrossModuleAssociation) error
	DeleteAssociation(domainModelID model.ID, assocID model.ID) error
	DeleteCrossAssociation(domainModelID model.ID, assocID model.ID) error

	// View entities
	CreateViewEntitySourceDocument(moduleID model.ID, moduleName, docName, oqlQuery, documentation string) (model.ID, error)
	DeleteViewEntitySourceDocument(id model.ID) error
	DeleteViewEntitySourceDocumentByName(moduleName, docName string) error
	FindViewEntitySourceDocumentID(moduleName, docName string) (model.ID, error)
	FindAllViewEntitySourceDocumentIDs(moduleName, docName string) ([]model.ID, error)
	MoveViewEntitySourceDocument(sourceModuleName string, targetModuleID model.ID, docName string) error
	UpdateOqlQueriesForMovedEntity(oldQualifiedName, newQualifiedName string) (int, error)
	UpdateEnumerationRefsInAllDomainModels(oldQualifiedName, newQualifiedName string) error
}
