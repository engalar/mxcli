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
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
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
	DeleteEntityFunc                           func(domainModelID model.ID, entityID model.ID) error
	DeleteAttributeFunc                        func(domainModelID model.ID, entityID model.ID, attrID model.ID) error
	DeleteAssociationFunc                      func(domainModelID model.ID, assocID model.ID) error
	DeleteCrossAssociationFunc                 func(domainModelID model.ID, assocID model.ID) error
	CreateViewEntitySourceDocumentFunc         func(moduleID model.ID, moduleName, docName, oqlQuery, documentation string) (model.ID, error)
	DeleteViewEntitySourceDocumentFunc         func(id model.ID) error
	DeleteViewEntitySourceDocumentByNameFunc   func(moduleName, docName string) error
	UpdateViewEntitySourceDocumentFunc         func(moduleName, docName, oqlQuery, documentation string) error
	FindViewEntitySourceDocumentIDFunc         func(moduleName, docName string) (model.ID, error)
	FindAllViewEntitySourceDocumentIDsFunc     func(moduleName, docName string) ([]model.ID, error)
	MoveViewEntitySourceDocumentFunc           func(sourceModuleName string, targetModuleID model.ID, docName string) error
	UpdateOqlQueriesForMovedEntityFunc         func(oldQualifiedName, newQualifiedName string) (int, error)
	UpdateEnumerationRefsInAllDomainModelsFunc func(oldQualifiedName, newQualifiedName string) error

	// Stage 3.3.4 C1 — gen-typed domain model Func fields.
	ListDomainModelsGenFunc   func() ([]*genDm.DomainModel, error)
	GetDomainModelGenFunc     func(moduleID model.ID) (*genDm.DomainModel, error)
	GetDomainModelByIDGenFunc func(id model.ID) (*genDm.DomainModel, error)
	UpdateDomainModelGenFunc  func(dm *genDm.DomainModel) error
	// Stage 3.3.4 D8 — gen-typed entity write Func fields (additive)
	CreateEntityGenFunc            func(domainModelID model.ID, entity *genDm.Entity) error
	UpdateEntityGenFunc            func(domainModelID model.ID, entity *genDm.Entity) error
	MoveEntityGenFunc              func(entity *genDm.Entity, sourceDMID, targetDMID model.ID, sourceModuleName, targetModuleName string) ([]string, error)
	CreateAssociationGenFunc       func(domainModelID model.ID, assoc *genDm.Association) error
	GetEntityIDByQualifiedNameFunc func(qualifiedName string) (element.ID, error)
	RelayoutDomainModelFunc        func(domainModelID model.ID) error

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

	// Stage 3.3.5.C1 gen-typed PageBackend surface — sole supported
	// page read/write API after the Stage 3.3.5.E1 cutover.
	ListPagesGenFunc         func() ([]*genPg.Page, error)
	GetPageGenFunc           func(id model.ID) (*genPg.Page, error)
	CreatePageGenFunc        func(parentUUID, containmentName string, page *genPg.Page) error
	UpdatePageGenFunc        func(page *genPg.Page) error
	ListLayoutsGenFunc       func() ([]*genPg.Layout, error)
	GetLayoutGenFunc         func(id model.ID) (*genPg.Layout, error)
	CreateLayoutGenFunc      func(parentUUID, containmentName string, layout *genPg.Layout) error
	UpdateLayoutGenFunc      func(layout *genPg.Layout) error
	ListSnippetsGenFunc      func() ([]*genPg.Snippet, error)
	GetSnippetGenFunc        func(id model.ID) (*genPg.Snippet, error)
	CreateSnippetGenFunc     func(parentUUID, containmentName string, snippet *genPg.Snippet) error
	UpdateSnippetGenFunc     func(snippet *genPg.Snippet) error
	GetPageContainerUUIDFunc func(id model.ID) (model.ID, error)

	// Stage 3.3.5.D5.c gen-typed delete + move Func fields.
	DeletePageGenFunc    func(id model.ID) error
	MovePageGenFunc      func(id, containerID model.ID) error
	MoveDocumentGenFunc  func(id, containerID model.ID) error
	DeleteLayoutGenFunc  func(id model.ID) error
	MoveLayoutGenFunc    func(id, containerID model.ID) error
	GetContainerIDFunc   func(moduleID model.ID, folder string) (model.ID, error)
	DeleteSnippetGenFunc func(id model.ID) error
	MoveSnippetGenFunc   func(id, containerID model.ID) error

	// Stage 3.3.5.E0.create_v3 transitional bridge Func fields.

	// PageModelBackend
	GetPageModelFunc      func(id model.ID) (*types.PageModel, error)
	GetSnippetModelFunc   func(id model.ID) (*types.PageModel, error)
	GetLayoutModelFunc    func(id model.ID) (*types.PageModel, error)
	WritePageModelFunc    func(id model.ID, m *types.PageModel) error
	WriteSnippetModelFunc func(id model.ID, m *types.PageModel) error

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
	SetPasswordPolicyFunc                func(unitID model.ID, minLength *int32, requireDigit, requireMixedCase, requireSymbol *bool) error
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
	ReconcileMemberAccessesFunc          func(unitID model.ID, moduleName string) ([]string, error)

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
	CreateJavaScriptActionGenFunc     func(parentUUID, containmentName string, jsa *genJSA.JavaScriptAction) error
	UpdateJavaScriptActionGenFunc     func(jsa *genJSA.JavaScriptAction) error

	// WorkflowBackend (gen-typed surface; pure-ID DeleteWorkflow lives
	// alongside the gen-typed quartet after Stage 3.3.3.E1).
	DeleteWorkflowFunc    func(id model.ID) error
	ListWorkflowsGenFunc  func() ([]*genWf.Workflow, error)
	GetWorkflowGenFunc    func(id model.ID) (*genWf.Workflow, error)
	CreateWorkflowGenFunc func(parentUUID, containmentName string, wf *genWf.Workflow) error
	UpdateWorkflowGenFunc func(wf *genWf.Workflow) error

	// SettingsBackend
	GetProjectSettingsFunc              func() (*model.ProjectSettings, error)
	UpdateProjectSettingsFunc           func(ps *model.ProjectSettings) error
	ListTranslationNodesFunc            func(docQN, docType string) ([]model.TranslationNode, error)
	SetEnumerationTranslationFunc       func(enumQN, valueName, langCode, text string) error
	SetMicroflowActionTranslationFunc   func(docQN, actionType string, index int, property, langCode, text string) error
	SetNavigationCaptionTranslationFunc func(profileName string, menuPath []string, langCode, text string) error

	// ImageBackend
	ListImageCollectionsFunc  func() ([]*types.ImageCollection, error)
	CreateImageCollectionFunc func(ic *types.ImageCollection) error
	UpdateImageCollectionFunc func(ic *types.ImageCollection) error
	DeleteImageCollectionFunc func(id string) error
	MoveImageCollectionFunc   func(ic *types.ImageCollection) error

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
	ListUnitHashesFunc   func() (map[string]string, error)
	GetUnitTypesFunc     func() (map[string]int, error)
	GetProjectRootIDFunc func() (string, error)
	ContentsDirFunc      func() string
	InvalidateCacheFunc  func()

	// WidgetBackend
	FindCustomWidgetTypeFunc     func(widgetID string) (*types.RawCustomWidgetType, error)
	FindAllCustomWidgetTypesFunc func(widgetID string) ([]*types.RawCustomWidgetType, error)

	// PageMutationBackend
	OpenPageForMutationFunc func(unitID model.ID) (backend.PageMutator, error)

	// WorkflowMutationBackend
	OpenWorkflowForMutationFunc func(unitID model.ID) (backend.WorkflowMutator, error)

	// WidgetSerializationBackend
	SerializeWorkflowActivityGenFunc func(a element.Element) (any, error)
	SerializePageGenElementFunc      func(elem element.Element) ([]byte, error)

	// WidgetBuilderBackend
	LoadWidgetTemplateFunc         func(widgetID string, projectPath string) (backend.WidgetObjectBuilder, error)
	BuildCreateAttributeObjectFunc func(attributePath string, objectTypeID, propertyTypeID, valueTypeID string) (any, error)
	BuildDataGrid2WidgetGenFunc    func(id model.ID, name string, spec backend.DataGridSpec, projectPath string) (*backend.GenCustomWidgetElem, error)
	BuildFilterWidgetGenFunc       func(spec backend.FilterWidgetSpec, projectPath string) (*backend.GenCustomWidgetElem, error)
	SerializeGenElemToOpaqueFunc   func(elem element.Element) backend.OpaqueWidget

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

	// ScriptTransactionBackend
	BeginScriptTransactionFunc func() (backend.ScriptTransaction, error)

	// ImportBufferBackend
	BeginImportBufferFunc   func() backend.ImportBuffer
	DisableImportBufferFunc func()
}
