// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// DomainModelBackend provides domain model, entity, attribute, and
// association operations.
type DomainModelBackend interface {
	// Domain models
	ListDomainModelsGen() ([]*genDm.DomainModel, error)
	GetDomainModelGen(moduleID model.ID) (*genDm.DomainModel, error)
	GetDomainModelByIDGen(id model.ID) (*genDm.DomainModel, error)
	UpdateDomainModelGen(dm *genDm.DomainModel) error
	// Stage 3.3.4 D8 — gen-typed entity write methods. The bridge
	// implementation in MprBackend converts to sdk types and delegates
	// to *ViaModelsdk write helpers (Stage 4 will retire the bridge).
	CreateEntityGen(domainModelID model.ID, entity *genDm.Entity) error
	UpdateEntityGen(domainModelID model.ID, entity *genDm.Entity) error
	MoveEntityGen(entity *genDm.Entity, sourceDMID, targetDMID model.ID, sourceModuleName, targetModuleName string) ([]string, error)
	CreateAssociationGen(domainModelID model.ID, assoc *genDm.Association) error

	// Entities
	DeleteEntity(domainModelID model.ID, entityID model.ID) error

	// Attributes
	DeleteAttribute(domainModelID model.ID, entityID model.ID, attrID model.ID) error

	// Associations
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
