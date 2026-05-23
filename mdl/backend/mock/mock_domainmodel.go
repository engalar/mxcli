// SPDX-License-Identifier: Apache-2.0

package mock

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// Stage 3.3.4 C1/D8: gen-typed domain model reads return descriptive
// errors when no Func is configured. The remaining sdk-typed delete
// methods keep the silent-success default since many tests pre-flight a
// write without caring about the result.

func (m *MockBackend) DeleteEntity(domainModelID model.ID, entityID model.ID) error {
	if m.DeleteEntityFunc != nil {
		return m.DeleteEntityFunc(domainModelID, entityID)
	}
	return nil
}

func (m *MockBackend) DeleteAttribute(domainModelID model.ID, entityID model.ID, attrID model.ID) error {
	if m.DeleteAttributeFunc != nil {
		return m.DeleteAttributeFunc(domainModelID, entityID, attrID)
	}
	return nil
}

func (m *MockBackend) DeleteAssociation(domainModelID model.ID, assocID model.ID) error {
	if m.DeleteAssociationFunc != nil {
		return m.DeleteAssociationFunc(domainModelID, assocID)
	}
	return nil
}

func (m *MockBackend) DeleteCrossAssociation(domainModelID model.ID, assocID model.ID) error {
	if m.DeleteCrossAssociationFunc != nil {
		return m.DeleteCrossAssociationFunc(domainModelID, assocID)
	}
	return nil
}

func (m *MockBackend) CreateViewEntitySourceDocument(moduleID model.ID, moduleName, docName, oqlQuery, documentation string) (model.ID, error) {
	if m.CreateViewEntitySourceDocumentFunc != nil {
		return m.CreateViewEntitySourceDocumentFunc(moduleID, moduleName, docName, oqlQuery, documentation)
	}
	return "", fmt.Errorf("MockBackend.CreateViewEntitySourceDocument not configured")
}

func (m *MockBackend) DeleteViewEntitySourceDocument(id model.ID) error {
	if m.DeleteViewEntitySourceDocumentFunc != nil {
		return m.DeleteViewEntitySourceDocumentFunc(id)
	}
	return nil
}

func (m *MockBackend) DeleteViewEntitySourceDocumentByName(moduleName, docName string) error {
	if m.DeleteViewEntitySourceDocumentByNameFunc != nil {
		return m.DeleteViewEntitySourceDocumentByNameFunc(moduleName, docName)
	}
	return nil
}

func (m *MockBackend) FindViewEntitySourceDocumentID(moduleName, docName string) (model.ID, error) {
	if m.FindViewEntitySourceDocumentIDFunc != nil {
		return m.FindViewEntitySourceDocumentIDFunc(moduleName, docName)
	}
	return "", fmt.Errorf("MockBackend.FindViewEntitySourceDocumentID not configured")
}

func (m *MockBackend) FindAllViewEntitySourceDocumentIDs(moduleName, docName string) ([]model.ID, error) {
	if m.FindAllViewEntitySourceDocumentIDsFunc != nil {
		return m.FindAllViewEntitySourceDocumentIDsFunc(moduleName, docName)
	}
	return nil, fmt.Errorf("MockBackend.FindAllViewEntitySourceDocumentIDs not configured")
}

func (m *MockBackend) MoveViewEntitySourceDocument(sourceModuleName string, targetModuleID model.ID, docName string) error {
	if m.MoveViewEntitySourceDocumentFunc != nil {
		return m.MoveViewEntitySourceDocumentFunc(sourceModuleName, targetModuleID, docName)
	}
	return nil
}

func (m *MockBackend) UpdateOqlQueriesForMovedEntity(oldQualifiedName, newQualifiedName string) (int, error) {
	if m.UpdateOqlQueriesForMovedEntityFunc != nil {
		return m.UpdateOqlQueriesForMovedEntityFunc(oldQualifiedName, newQualifiedName)
	}
	return 0, nil
}

func (m *MockBackend) UpdateEnumerationRefsInAllDomainModels(oldQualifiedName, newQualifiedName string) error {
	if m.UpdateEnumerationRefsInAllDomainModelsFunc != nil {
		return m.UpdateEnumerationRefsInAllDomainModelsFunc(oldQualifiedName, newQualifiedName)
	}
	return nil
}

// Stage 3.3.4 C1 — gen-typed Mock implementations.

func (m *MockBackend) ListDomainModelsGen() ([]*genDm.DomainModel, error) {
	if m.ListDomainModelsGenFunc != nil {
		return m.ListDomainModelsGenFunc()
	}
	return nil, fmt.Errorf("MockBackend.ListDomainModelsGen not configured")
}

func (m *MockBackend) GetDomainModelGen(moduleID model.ID) (*genDm.DomainModel, error) {
	if m.GetDomainModelGenFunc != nil {
		return m.GetDomainModelGenFunc(moduleID)
	}
	return nil, fmt.Errorf("MockBackend.GetDomainModelGen not configured")
}

func (m *MockBackend) GetDomainModelByIDGen(id model.ID) (*genDm.DomainModel, error) {
	if m.GetDomainModelByIDGenFunc != nil {
		return m.GetDomainModelByIDGenFunc(id)
	}
	return nil, fmt.Errorf("MockBackend.GetDomainModelByIDGen not configured")
}

func (m *MockBackend) UpdateDomainModelGen(dm *genDm.DomainModel) error {
	if m.UpdateDomainModelGenFunc != nil {
		return m.UpdateDomainModelGenFunc(dm)
	}
	return nil
}

// Stage 3.3.4 D8 — gen-typed entity write Mock impls.

func (m *MockBackend) CreateEntityGen(domainModelID model.ID, entity *genDm.Entity) error {
	if m.CreateEntityGenFunc != nil {
		return m.CreateEntityGenFunc(domainModelID, entity)
	}
	return fmt.Errorf("MockBackend.CreateEntityGen not configured")
}

func (m *MockBackend) UpdateEntityGen(domainModelID model.ID, entity *genDm.Entity) error {
	if m.UpdateEntityGenFunc != nil {
		return m.UpdateEntityGenFunc(domainModelID, entity)
	}
	return fmt.Errorf("MockBackend.UpdateEntityGen not configured")
}

func (m *MockBackend) MoveEntityGen(entity *genDm.Entity, sourceDMID, targetDMID model.ID, sourceModuleName, targetModuleName string) ([]string, error) {
	if m.MoveEntityGenFunc != nil {
		return m.MoveEntityGenFunc(entity, sourceDMID, targetDMID, sourceModuleName, targetModuleName)
	}
	return nil, fmt.Errorf("MockBackend.MoveEntityGen not configured")
}

func (m *MockBackend) CreateAssociationGen(domainModelID model.ID, assoc *genDm.Association) error {
	if m.CreateAssociationGenFunc != nil {
		return m.CreateAssociationGenFunc(domainModelID, assoc)
	}
	return nil
}
