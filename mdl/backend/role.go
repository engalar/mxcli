// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	genJSA "github.com/mendixlabs/mxcli/modelsdk/gen/javascriptactions"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

// Role interfaces — segregated by responsibility for dependency injection.
//
// Each interface groups methods that are logically co-consumed. Handler
// functions should accept the narrowest role interface they need, not
// FullBackend.
//
// Production backends satisfy all role interfaces. Consumers declare
// only the roles they use.

// ModuleLister provides read-only module queries.
type ModuleLister interface {
	ListModules() ([]*model.Module, error)
	GetModule(id model.ID) (*model.Module, error)
	GetModuleByName(name string) (*model.Module, error)
}

// ModuleWriter provides module mutations.
type ModuleWriter interface {
	CreateModule(module *model.Module) error
	UpdateModule(module *model.Module) error
	DeleteModule(id model.ID) error
	DeleteModuleWithCleanup(id model.ID, moduleName string) error
}

// DomainModelReader provides read-only domain model / entity / association queries.
type DomainModelReader interface {
	ListDomainModelsGen() ([]*genDm.DomainModel, error)
	GetDomainModelGen(moduleID model.ID) (*genDm.DomainModel, error)
	GetDomainModelByIDGen(id model.ID) (*genDm.DomainModel, error)
	GetEntityIDByQualifiedName(qualifiedName string) (element.ID, error)
}

// DomainModelWriter provides domain model / entity / association mutations.
type DomainModelWriter interface {
	UpdateDomainModelGen(dm *genDm.DomainModel) error
	CreateEntityGen(domainModelID model.ID, entity *genDm.Entity) error
	UpdateEntityGen(domainModelID model.ID, entity *genDm.Entity) error
	MoveEntityGen(entity *genDm.Entity, sourceDMID, targetDMID model.ID, sourceModuleName, targetModuleName string) ([]string, error)
	CreateAssociationGen(domainModelID model.ID, assoc *genDm.Association) error
	DeleteEntity(domainModelID model.ID, entityID model.ID) error
	DeleteAttribute(domainModelID model.ID, entityID model.ID, attrID model.ID) error
	DeleteAssociation(domainModelID model.ID, assocID model.ID) error
	DeleteCrossAssociation(domainModelID model.ID, assocID model.ID) error
	CreateViewEntitySourceDocument(moduleID model.ID, moduleName, docName, oqlQuery, documentation string) (model.ID, error)
	DeleteViewEntitySourceDocument(id model.ID) error
	DeleteViewEntitySourceDocumentByName(moduleName, docName string) error
	FindViewEntitySourceDocumentID(moduleName, docName string) (model.ID, error)
	FindAllViewEntitySourceDocumentIDs(moduleName, docName string) ([]model.ID, error)
	MoveViewEntitySourceDocument(sourceModuleName string, targetModuleID model.ID, docName string) error
	UpdateOqlQueriesForMovedEntity(oldQualifiedName, newQualifiedName string) (int, error)
	UpdateEnumerationRefsInAllDomainModels(oldQualifiedName, newQualifiedName string) error
	RelayoutDomainModel(domainModelID model.ID) error
}

// MicroflowReader provides read-only microflow/nanoflow queries.
type MicroflowReader interface {
	ListMicroflowsGen() ([]*genMf.Microflow, error)
	ListNanoflowsGen() ([]*genMf.Nanoflow, error)
	GetMicroflowGen(id model.ID) (*genMf.Microflow, error)
	IsRule(qualifiedName string) (bool, error)
}

// MicroflowWriter provides microflow/nanoflow deletions.
type MicroflowWriter interface {
	DeleteMicroflow(id model.ID) error
	DeleteNanoflow(id model.ID) error
}

// WorkflowReader provides read-only workflow queries.
type WorkflowReader interface {
	ListWorkflowsGen() ([]*genWf.Workflow, error)
	GetWorkflowGen(id model.ID) (*genWf.Workflow, error)
}

// WorkflowWriter provides workflow mutations.
type WorkflowWriter interface {
	CreateWorkflowGen(parentUUID, containmentName string, wf *genWf.Workflow) error
	UpdateWorkflowGen(wf *genWf.Workflow) error
	DeleteWorkflow(id model.ID) error
}

// JavaActionReader provides read-only Java/JavaScript action queries.
type JavaActionReader interface {
	ListJavaActionsGen() ([]*genJA.JavaAction, error)
	ReadJavaActionByNameGen(qualifiedName string) (*genJA.JavaAction, error)
	ListJavaScriptActionsGen() ([]*genJSA.JavaScriptAction, error)
	ReadJavaScriptActionByNameGen(qualifiedName string) (*genJSA.JavaScriptAction, error)
	ReadJavaSourceFile(moduleName, actionName string) (string, error)
}

// JavaActionWriter provides Java/JavaScript action mutations.
type JavaActionWriter interface {
	DeleteJavaAction(id model.ID) error
	DeleteJavaSourceFile(moduleName, actionName string) error
	RenameJavaSourceFile(moduleName, oldName, newName string) error
	CreateJavaActionGen(parentUUID, containmentName string, ja *genJA.JavaAction) error
	UpdateJavaActionGen(ja *genJA.JavaAction) error
	WriteJavaSourceFileGen(moduleName, actionName string, javaCode string, params []*genJA.JavaActionParameter, returnType element.Element, extraImports []string, extraCode string) error
	UpdateJavaScriptActionGen(jsa *genJSA.JavaScriptAction) error
}

// EnumerationReader provides read-only enumeration queries.
type EnumerationReader interface {
	ListEnumerations() ([]*model.Enumeration, error)
	GetEnumeration(id model.ID) (*model.Enumeration, error)
}

// EnumerationWriter provides enumeration mutations.
type EnumerationWriter interface {
	CreateEnumeration(enum *model.Enumeration) error
	UpdateEnumeration(enum *model.Enumeration) error
	MoveEnumeration(enum *model.Enumeration) error
	DeleteEnumeration(id model.ID) error
}

// ConstantReader provides read-only constant queries.
type ConstantReader interface {
	ListConstants() ([]*model.Constant, error)
	GetConstant(id model.ID) (*model.Constant, error)
}

// ConstantWriter provides constant mutations.
type ConstantWriter interface {
	CreateConstant(constant *model.Constant) error
	UpdateConstant(constant *model.Constant) error
	MoveConstant(constant *model.Constant) error
	DeleteConstant(id model.ID) error
}

// SettingsReader provides read-only settings queries.
type SettingsReader interface {
	GetProjectSettings() (*model.ProjectSettings, error)
}

// SettingsWriter provides settings mutations.
type SettingsWriter interface {
	UpdateProjectSettings(ps *model.ProjectSettings) error
	ListTranslationNodes(docQN, docType string) ([]model.TranslationNode, error)
	SetEnumerationTranslation(enumQN, valueName, langCode, text string) error
	SetMicroflowActionTranslation(docQN, actionType string, index int, property, langCode, text string) error
	SetNavigationCaptionTranslation(profileName string, menuPath []string, langCode, text string) error
}

// MappingReader provides read-only import/export mapping and JSON structure queries.
type MappingReader interface {
	ListImportMappings() ([]*model.ImportMapping, error)
	GetImportMappingByQualifiedName(moduleName, name string) (*model.ImportMapping, error)
	ListExportMappings() ([]*model.ExportMapping, error)
	GetExportMappingByQualifiedName(moduleName, name string) (*model.ExportMapping, error)
	ListJsonStructures() ([]*types.JsonStructure, error)
	GetJsonStructureByQualifiedName(moduleName, name string) (*types.JsonStructure, error)
}

// MappingWriter provides import/export mapping and JSON structure mutations.
type MappingWriter interface {
	CreateImportMapping(im *model.ImportMapping) error
	UpdateImportMapping(im *model.ImportMapping) error
	DeleteImportMapping(id model.ID) error
	MoveImportMapping(im *model.ImportMapping) error
	CreateExportMapping(em *model.ExportMapping) error
	UpdateExportMapping(em *model.ExportMapping) error
	DeleteExportMapping(id model.ID) error
	MoveExportMapping(em *model.ExportMapping) error
	CreateJsonStructure(js *types.JsonStructure) error
	UpdateJsonStructure(js *types.JsonStructure) error
	DeleteJsonStructure(id string) error
}

// UnitReader provides read-only low-level unit access.
type UnitReader interface {
	GetRawUnit(id model.ID) (map[string]any, error)
	GetRawUnitBytes(id model.ID) ([]byte, error)
	ListRawUnitsByType(typePrefix string) ([]*types.RawUnit, error)
	ListRawUnits(objectType string) ([]*types.RawUnitInfo, error)
	GetRawUnitByName(objectType, qualifiedName string) (*types.RawUnitInfo, error)
	GetRawMicroflowByName(qualifiedName string) ([]byte, error)
}

// UnitWriter provides low-level unit mutations.
type UnitWriter interface {
	UpdateRawUnit(unitID string, contents []byte) error
}

// NavigationReader provides read-only navigation queries.
type NavigationReader interface {
	ListNavigationDocuments() ([]*types.NavigationDocument, error)
	GetNavigation() (*types.NavigationDocument, error)
}

// NavigationWriter provides navigation mutations.
type NavigationWriter interface {
	UpdateNavigationProfile(navDocID model.ID, profileName string, spec types.NavigationProfileSpec) error
}

// ImageCollectionWriter provides image collection mutations.
type ImageCollectionWriter interface {
	CreateImageCollection(ic *types.ImageCollection) error
	UpdateImageCollection(ic *types.ImageCollection) error
	DeleteImageCollection(id string) error
	MoveImageCollection(ic *types.ImageCollection) error
}

// ServiceLister provides read-only service queries.
type ServiceLister interface {
	ODataBackend
	RESTBackend
	BusinessEventBackend
	DatabaseConnectionBackend
	DataTransformerBackend
}

// ScheduledEventReader provides read-only scheduled event queries.
type ScheduledEventReader interface {
	ListScheduledEvents() ([]*model.ScheduledEvent, error)
	GetScheduledEvent(id model.ID) (*model.ScheduledEvent, error)
}

// MetadataReader provides project metadata and introspection queries.
type MetadataReader interface {
	ListAllUnitIDs() ([]string, error)
	ListUnits() ([]*types.UnitInfo, error)
	ListUnitHashes() (map[string]string, error)
	GetUnitTypes() (map[string]int, error)
	GetProjectRootID() (string, error)
	ContentsDir() string
	InvalidateCache()
}

// WidgetInspector provides widget type introspection.
type WidgetInspector interface {
	FindCustomWidgetType(widgetID string) (*types.RawCustomWidgetType, error)
	FindAllCustomWidgetTypes(widgetID string) ([]*types.RawCustomWidgetType, error)
}
