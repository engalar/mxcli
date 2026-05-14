// SPDX-License-Identifier: Apache-2.0

// Package mock provides a configurable MockBackend for testing that
// implements backend.FullBackend. Each method delegates to an optional
// function field; when the field is nil the method returns zero values.
package mock

import (
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	genJSA "github.com/mendixlabs/mxcli/modelsdk/gen/javascriptactions"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
	"github.com/mendixlabs/mxcli/sdk/pages"
	"github.com/mendixlabs/mxcli/sdk/workflows"
)

var _ backend.FullBackend = (*MockBackend)(nil)

// MockBackend implements backend.FullBackend. Every interface method is
// backed by a public function field. If the field is nil the method
// returns zero values / nil error (never panics).
type MockBackend struct {
	// ConnectionBackend
	ConnectFunc          func(path string) error
	DisconnectFunc       func() error
	CommitFunc           func() error
	IsConnectedFunc      func() bool
	PathFunc             func() string
	VersionFunc          func() types.MPRVersion
	ProjectVersionFunc   func() *types.ProjectVersion
	GetMendixVersionFunc func() (string, error)

	// ModuleBackend
	ListModulesFunc             func() ([]*model.Module, error)
	GetModuleFunc               func(id model.ID) (*model.Module, error)
	GetModuleByNameFunc         func(name string) (*model.Module, error)
	CreateModuleFunc            func(module *model.Module) error
	UpdateModuleFunc            func(module *model.Module) error
	DeleteModuleFunc            func(id model.ID) error
	DeleteModuleWithCleanupFunc func(id model.ID, moduleName string) error

	// ModuleSettingsBackend
	ListModuleSettingsFunc   func() ([]*types.ModuleSettings, error)
	GetModuleSettingsFunc    func(moduleID model.ID) (*types.ModuleSettings, error)
	UpdateModuleSettingsFunc func(ms *types.ModuleSettings) error

	// FolderBackend
	ListFoldersFunc  func() ([]*types.FolderInfo, error)
	CreateFolderFunc func(folder *model.Folder) error
	DeleteFolderFunc func(id model.ID) error
	MoveFolderFunc   func(id model.ID, newContainerID model.ID) error

	// DomainModelBackend
	ListDomainModelsFunc                       func() ([]*domainmodel.DomainModel, error)
	GetDomainModelFunc                         func(moduleID model.ID) (*domainmodel.DomainModel, error)
	GetDomainModelByIDFunc                     func(id model.ID) (*domainmodel.DomainModel, error)
	UpdateDomainModelFunc                      func(dm *domainmodel.DomainModel) error
	CreateEntityFunc                           func(domainModelID model.ID, entity *domainmodel.Entity) error
	UpdateEntityFunc                           func(domainModelID model.ID, entity *domainmodel.Entity) error
	DeleteEntityFunc                           func(domainModelID model.ID, entityID model.ID) error
	MoveEntityFunc                             func(entity *domainmodel.Entity, sourceDMID, targetDMID model.ID, sourceModuleName, targetModuleName string) ([]string, error)
	AddAttributeFunc                           func(domainModelID model.ID, entityID model.ID, attr *domainmodel.Attribute) error
	UpdateAttributeFunc                        func(domainModelID model.ID, entityID model.ID, attr *domainmodel.Attribute) error
	DeleteAttributeFunc                        func(domainModelID model.ID, entityID model.ID, attrID model.ID) error
	CreateAssociationFunc                      func(domainModelID model.ID, assoc *domainmodel.Association) error
	CreateCrossAssociationFunc                 func(domainModelID model.ID, ca *domainmodel.CrossModuleAssociation) error
	DeleteAssociationFunc                      func(domainModelID model.ID, assocID model.ID) error
	DeleteCrossAssociationFunc                 func(domainModelID model.ID, assocID model.ID) error
	CreateViewEntitySourceDocumentFunc         func(moduleID model.ID, moduleName, docName, oqlQuery, documentation string) (model.ID, error)
	DeleteViewEntitySourceDocumentFunc         func(id model.ID) error
	DeleteViewEntitySourceDocumentByNameFunc   func(moduleName, docName string) error
	FindViewEntitySourceDocumentIDFunc         func(moduleName, docName string) (model.ID, error)
	FindAllViewEntitySourceDocumentIDsFunc     func(moduleName, docName string) ([]model.ID, error)
	MoveViewEntitySourceDocumentFunc           func(sourceModuleName string, targetModuleID model.ID, docName string) error
	UpdateOqlQueriesForMovedEntityFunc         func(oldQualifiedName, newQualifiedName string) (int, error)
	UpdateEnumerationRefsInAllDomainModelsFunc func(oldQualifiedName, newQualifiedName string) error

	// Stage 3.3.4 C1 — gen-typed sibling Func fields (additive)
	ListDomainModelsGenFunc   func() ([]*genDm.DomainModel, error)
	GetDomainModelGenFunc     func(moduleID model.ID) (*genDm.DomainModel, error)
	GetDomainModelByIDGenFunc func(id model.ID) (*genDm.DomainModel, error)
	UpdateDomainModelGenFunc  func(dm *genDm.DomainModel) error
	// Stage 3.3.4 D8 — gen-typed entity write Func fields (additive)
	CreateEntityGenFunc func(domainModelID model.ID, entity *genDm.Entity) error
	UpdateEntityGenFunc func(domainModelID model.ID, entity *genDm.Entity) error

	// MicroflowBackend — Followup E6 retired Get / Create / Update /
	// Move / Parse; Followup F3 retired the sdk-typed ListMicroflows /
	// GetMicroflow / ListNanoflows. The remaining surface keeps the
	// gen-typed reads and the small fallbacks (Delete*, IsRule).
	// Tests that need to seed flow data should use the gen-typed
	// repostesting.RecordingMicroflowRepository (via withMicroflowsRepo)
	// or configure ListMicroflowsGenFunc / GetMicroflowGenFunc.
	DeleteMicroflowFunc   func(id model.ID) error
	DeleteNanoflowFunc    func(id model.ID) error
	IsRuleFunc            func(qualifiedName string) (bool, error)
	ListMicroflowsGenFunc func() ([]*genMf.Microflow, error)
	ListNanoflowsGenFunc  func() ([]*genMf.Nanoflow, error)
	GetMicroflowGenFunc   func(id model.ID) (*genMf.Microflow, error)

	// PageBackend
	ListPagesFunc          func() ([]*pages.Page, error)
	GetPageFunc            func(id model.ID) (*pages.Page, error)
	CreatePageFunc         func(page *pages.Page) error
	UpdatePageFunc         func(page *pages.Page) error
	DeletePageFunc         func(id model.ID) error
	MovePageFunc           func(page *pages.Page) error
	ListLayoutsFunc        func() ([]*pages.Layout, error)
	GetLayoutFunc          func(id model.ID) (*pages.Layout, error)
	CreateLayoutFunc       func(layout *pages.Layout) error
	UpdateLayoutFunc       func(layout *pages.Layout) error
	DeleteLayoutFunc       func(id model.ID) error
	ListSnippetsFunc       func() ([]*pages.Snippet, error)
	CreateSnippetFunc      func(snippet *pages.Snippet) error
	UpdateSnippetFunc      func(snippet *pages.Snippet) error
	DeleteSnippetFunc      func(id model.ID) error
	MoveSnippetFunc        func(snippet *pages.Snippet) error
	ListBuildingBlocksFunc func() ([]*pages.BuildingBlock, error)
	ListPageTemplatesFunc  func() ([]*pages.PageTemplate, error)

	// EnumerationBackend
	ListEnumerationsFunc  func() ([]*model.Enumeration, error)
	GetEnumerationFunc    func(id model.ID) (*model.Enumeration, error)
	CreateEnumerationFunc func(enum *model.Enumeration) error
	UpdateEnumerationFunc func(enum *model.Enumeration) error
	MoveEnumerationFunc   func(enum *model.Enumeration) error
	DeleteEnumerationFunc func(id model.ID) error

	// ConstantBackend
	ListConstantsFunc  func() ([]*model.Constant, error)
	GetConstantFunc    func(id model.ID) (*model.Constant, error)
	CreateConstantFunc func(constant *model.Constant) error
	UpdateConstantFunc func(constant *model.Constant) error
	MoveConstantFunc   func(constant *model.Constant) error
	DeleteConstantFunc func(id model.ID) error

	// SecurityBackend
	GetProjectSecurityGenFunc            func() (*genSec.ProjectSecurity, error)
	SetProjectSecurityLevelFunc          func(unitID model.ID, level string) error
	SetProjectDemoUsersEnabledFunc       func(unitID model.ID, enabled bool) error
	AddUserRoleFunc                      func(unitID model.ID, name string, moduleRoles []string, manageAllRoles bool) error
	AlterUserRoleModuleRolesFunc         func(unitID model.ID, userRoleName string, add bool, moduleRoles []string) error
	RemoveUserRoleFunc                   func(unitID model.ID, name string) error
	AddDemoUserFunc                      func(unitID model.ID, userName, password, entity string, userRoles []string) error
	RemoveDemoUserFunc                   func(unitID model.ID, userName string) error
	GetModuleSecurityGenFunc             func(moduleID model.ID) (*genSec.ModuleSecurity, error)
	ListModuleSecurityGenFunc            func() ([]*genSec.ModuleSecurity, error)
	AddModuleRoleFunc                    func(unitID model.ID, roleName, description string) error
	RemoveModuleRoleFunc                 func(unitID model.ID, roleName string) error
	RemoveModuleRoleFromAllUserRolesFunc func(unitID model.ID, qualifiedRole string) (int, error)
	UpdateAllowedRolesFunc               func(unitID model.ID, roles []string) error
	UpdatePublishedRestServiceRolesFunc  func(unitID model.ID, roles []string) error
	RemoveFromAllowedRolesFunc           func(unitID model.ID, roleName string) (bool, error)
	AddEntityAccessRuleFunc              func(params backend.EntityAccessRuleParams) error
	RemoveEntityAccessRuleFunc           func(unitID model.ID, entityName string, roleNames []string) (int, error)
	RevokeEntityMemberAccessFunc         func(unitID model.ID, entityName string, roleNames []string, revocation types.EntityAccessRevocation) (int, error)
	RemoveRoleFromAllEntitiesFunc        func(unitID model.ID, roleName string) (int, error)
	ReconcileMemberAccessesFunc          func(unitID model.ID, moduleName string) (int, error)

	// NavigationBackend
	ListNavigationDocumentsFunc func() ([]*types.NavigationDocument, error)
	GetNavigationFunc           func() (*types.NavigationDocument, error)
	UpdateNavigationProfileFunc func(navDocID model.ID, profileName string, spec types.NavigationProfileSpec) error

	// ServiceBackend
	ListConsumedODataServicesFunc   func() ([]*model.ConsumedODataService, error)
	ListPublishedODataServicesFunc  func() ([]*model.PublishedODataService, error)
	CreateConsumedODataServiceFunc  func(svc *model.ConsumedODataService) error
	UpdateConsumedODataServiceFunc  func(svc *model.ConsumedODataService) error
	DeleteConsumedODataServiceFunc  func(id model.ID) error
	CreatePublishedODataServiceFunc func(svc *model.PublishedODataService) error
	UpdatePublishedODataServiceFunc func(svc *model.PublishedODataService) error
	DeletePublishedODataServiceFunc func(id model.ID) error
	ListConsumedRestServicesFunc    func() ([]*model.ConsumedRestService, error)
	ListPublishedRestServicesFunc   func() ([]*model.PublishedRestService, error)
	CreateConsumedRestServiceFunc   func(svc *model.ConsumedRestService) error
	UpdateConsumedRestServiceFunc   func(svc *model.ConsumedRestService) error
	DeleteConsumedRestServiceFunc   func(id model.ID) error
	CreatePublishedRestServiceFunc  func(svc *model.PublishedRestService) error
	UpdatePublishedRestServiceFunc  func(svc *model.PublishedRestService) error
	DeletePublishedRestServiceFunc  func(id model.ID) error
	ListBusinessEventServicesFunc   func() ([]*model.BusinessEventService, error)
	CreateBusinessEventServiceFunc  func(svc *model.BusinessEventService) error
	UpdateBusinessEventServiceFunc  func(svc *model.BusinessEventService) error
	DeleteBusinessEventServiceFunc  func(id model.ID) error
	ListDatabaseConnectionsFunc     func() ([]*model.DatabaseConnection, error)
	CreateDatabaseConnectionFunc    func(conn *model.DatabaseConnection) error
	UpdateDatabaseConnectionFunc    func(conn *model.DatabaseConnection) error
	MoveDatabaseConnectionFunc      func(conn *model.DatabaseConnection) error
	DeleteDatabaseConnectionFunc    func(id model.ID) error
	ListDataTransformersFunc        func() ([]*model.DataTransformer, error)
	CreateDataTransformerFunc       func(dt *model.DataTransformer) error
	UpdateDataTransformerFunc       func(dt *model.DataTransformer) error
	DeleteDataTransformerFunc       func(id model.ID) error

	// MappingBackend
	ListImportMappingsFunc              func() ([]*model.ImportMapping, error)
	GetImportMappingByQualifiedNameFunc func(moduleName, name string) (*model.ImportMapping, error)
	CreateImportMappingFunc             func(im *model.ImportMapping) error
	UpdateImportMappingFunc             func(im *model.ImportMapping) error
	DeleteImportMappingFunc             func(id model.ID) error
	MoveImportMappingFunc               func(im *model.ImportMapping) error
	ListExportMappingsFunc              func() ([]*model.ExportMapping, error)
	GetExportMappingByQualifiedNameFunc func(moduleName, name string) (*model.ExportMapping, error)
	CreateExportMappingFunc             func(em *model.ExportMapping) error
	UpdateExportMappingFunc             func(em *model.ExportMapping) error
	DeleteExportMappingFunc             func(id model.ID) error
	MoveExportMappingFunc               func(em *model.ExportMapping) error
	ListJsonStructuresFunc              func() ([]*types.JsonStructure, error)
	GetJsonStructureByQualifiedNameFunc func(moduleName, name string) (*types.JsonStructure, error)
	CreateJsonStructureFunc             func(js *types.JsonStructure) error
	UpdateJsonStructureFunc             func(js *types.JsonStructure) error
	DeleteJsonStructureFunc             func(id string) error

	// JavaBackend (sdk-typed methods retired in Stage 3.3.2.E1; only
	// ID-/string-keyed file operations remain)
	DeleteJavaActionFunc     func(id model.ID) error
	DeleteJavaSourceFileFunc func(moduleName, actionName string) error
	RenameJavaSourceFileFunc func(moduleName, oldName, newName string) error
	ReadJavaSourceFileFunc   func(moduleName, actionName string) (string, error)

	// Gen-typed JavaBackend surface
	ListJavaActionsGenFunc            func() ([]*genJA.JavaAction, error)
	ReadJavaActionByNameGenFunc       func(qualifiedName string) (*genJA.JavaAction, error)
	CreateJavaActionGenFunc           func(parentUUID, containmentName string, ja *genJA.JavaAction) error
	UpdateJavaActionGenFunc           func(ja *genJA.JavaAction) error
	WriteJavaSourceFileGenFunc        func(moduleName, actionName string, javaCode string, params []*genJA.JavaActionParameter, returnType element.Element, extraImports []string, extraCode string) error
	ListJavaScriptActionsGenFunc      func() ([]*genJSA.JavaScriptAction, error)
	ReadJavaScriptActionByNameGenFunc func(qualifiedName string) (*genJSA.JavaScriptAction, error)

	// WorkflowBackend
	ListWorkflowsFunc  func() ([]*workflows.Workflow, error)
	GetWorkflowFunc    func(id model.ID) (*workflows.Workflow, error)
	CreateWorkflowFunc func(wf *workflows.Workflow) error
	UpdateWorkflowFunc func(wf *workflows.Workflow) error
	DeleteWorkflowFunc func(id model.ID) error

	// SettingsBackend
	GetProjectSettingsFunc    func() (*model.ProjectSettings, error)
	UpdateProjectSettingsFunc func(ps *model.ProjectSettings) error

	// ImageBackend
	ListImageCollectionsFunc  func() ([]*types.ImageCollection, error)
	CreateImageCollectionFunc func(ic *types.ImageCollection) error
	UpdateImageCollectionFunc func(ic *types.ImageCollection) error
	DeleteImageCollectionFunc func(id string) error

	// ScheduledEventBackend
	ListScheduledEventsFunc func() ([]*model.ScheduledEvent, error)
	GetScheduledEventFunc   func(id model.ID) (*model.ScheduledEvent, error)

	// RenameBackend
	UpdateQualifiedNameInAllUnitsFunc func(oldName, newName string) (int, error)
	RenameReferencesFunc              func(oldName, newName string, dryRun bool) ([]types.RenameHit, error)
	RenameDocumentByNameFunc          func(moduleName, oldName, newName string) error

	// RawUnitBackend
	GetRawUnitFunc            func(id model.ID) (map[string]any, error)
	GetRawUnitBytesFunc       func(id model.ID) ([]byte, error)
	ListRawUnitsByTypeFunc    func(typePrefix string) ([]*types.RawUnit, error)
	ListRawUnitsFunc          func(objectType string) ([]*types.RawUnitInfo, error)
	GetRawUnitByNameFunc      func(objectType, qualifiedName string) (*types.RawUnitInfo, error)
	GetRawMicroflowByNameFunc func(qualifiedName string) ([]byte, error)
	UpdateRawUnitFunc         func(unitID string, contents []byte) error

	// MetadataBackend
	ListAllUnitIDsFunc   func() ([]string, error)
	ListUnitsFunc        func() ([]*types.UnitInfo, error)
	GetUnitTypesFunc     func() (map[string]int, error)
	GetProjectRootIDFunc func() (string, error)
	ContentsDirFunc      func() string
	ExportJSONFunc       func() ([]byte, error)
	InvalidateCacheFunc  func()

	// WidgetBackend
	FindCustomWidgetTypeFunc     func(widgetID string) (*types.RawCustomWidgetType, error)
	FindAllCustomWidgetTypesFunc func(widgetID string) ([]*types.RawCustomWidgetType, error)

	// PageMutationBackend
	OpenPageForMutationFunc func(unitID model.ID) (backend.PageMutator, error)

	// WorkflowMutationBackend
	OpenWorkflowForMutationFunc func(unitID model.ID) (backend.WorkflowMutator, error)

	// WidgetSerializationBackend
	SerializeWidgetFunc           func(w pages.Widget) (any, error)
	SerializeClientActionFunc     func(a pages.ClientAction) (any, error)
	SerializeDataSourceFunc       func(ds pages.DataSource) (any, error)
	SerializeWorkflowActivityFunc func(a workflows.WorkflowActivity) (any, error)

	// WidgetBuilderBackend
	LoadWidgetTemplateFunc          func(widgetID string, projectPath string) (backend.WidgetObjectBuilder, error)
	SerializeWidgetToOpaqueFunc     func(w pages.Widget) any
	SerializeDataSourceToOpaqueFunc func(ds pages.DataSource) any
	BuildCreateAttributeObjectFunc  func(attributePath string, objectTypeID, propertyTypeID, valueTypeID string) (any, error)
	BuildDataGrid2WidgetFunc        func(id model.ID, name string, spec backend.DataGridSpec, projectPath string) (*pages.CustomWidget, error)
	BuildFilterWidgetFunc           func(spec backend.FilterWidgetSpec, projectPath string) (pages.Widget, error)

	// AgentEditorBackend
	ListAgentEditorModelsFunc               func() ([]*types.Model, error)
	ListAgentEditorKnowledgeBasesFunc       func() ([]*types.KnowledgeBase, error)
	ListAgentEditorConsumedMCPServicesFunc  func() ([]*types.ConsumedMCPService, error)
	ListAgentEditorAgentsFunc               func() ([]*types.Agent, error)
	CreateAgentEditorModelFunc              func(m *types.Model) error
	UpdateAgentEditorModelFunc              func(m *types.Model) error
	DeleteAgentEditorModelFunc              func(id string) error
	CreateAgentEditorKnowledgeBaseFunc      func(kb *types.KnowledgeBase) error
	UpdateAgentEditorKnowledgeBaseFunc      func(kb *types.KnowledgeBase) error
	DeleteAgentEditorKnowledgeBaseFunc      func(id string) error
	CreateAgentEditorConsumedMCPServiceFunc func(svc *types.ConsumedMCPService) error
	UpdateAgentEditorConsumedMCPServiceFunc func(svc *types.ConsumedMCPService) error
	DeleteAgentEditorConsumedMCPServiceFunc func(id string) error
	CreateAgentEditorAgentFunc              func(a *types.Agent) error
	UpdateAgentEditorAgentFunc              func(a *types.Agent) error
	DeleteAgentEditorAgentFunc              func(id string) error
}
