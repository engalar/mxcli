// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	genJSA "github.com/mendixlabs/mxcli/modelsdk/gen/javascriptactions"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
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
	UpdateViewEntitySourceDocument(moduleName, docName, oqlQuery, documentation string) error
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

// MicroflowWriter provides microflow/nanoflow deletions.
type MicroflowWriter interface {
	DeleteMicroflow(id model.ID) error
	DeleteNanoflow(id model.ID) error
}

// WorkflowWriter provides workflow mutations.
type WorkflowWriter interface {
	CreateWorkflowGen(parentUUID, containmentName string, wf *genWf.Workflow) error
	UpdateWorkflowGen(wf *genWf.Workflow) error
	DeleteWorkflow(id model.ID) error
}

// JavaActionReader provides read-only Java action queries.
type JavaActionReader interface {
	ListJavaActionsGen() ([]*genJA.JavaAction, error)
	ReadJavaActionByNameGen(qualifiedName string) (*genJA.JavaAction, error)
	ReadJavaSourceFile(moduleName, actionName string) (string, error)
}

// JavaActionWriter provides Java action mutations.
type JavaActionWriter interface {
	DeleteJavaAction(id model.ID) error
	DeleteJavaSourceFile(moduleName, actionName string) error
	RenameJavaSourceFile(moduleName, oldName, newName string) error
	CreateJavaActionGen(parentUUID, containmentName string, ja *genJA.JavaAction) error
	UpdateJavaActionGen(ja *genJA.JavaAction) error
	WriteJavaSourceFileGen(moduleName, actionName string, javaCode string, params []*genJA.JavaActionParameter, returnType element.Element, extraImports []string, extraCode string) error
}

// JavaScriptActionReader provides read-only JavaScript action queries.
type JavaScriptActionReader interface {
	ListJavaScriptActionsGen() ([]*genJSA.JavaScriptAction, error)
	ReadJavaScriptActionByNameGen(qualifiedName string) (*genJSA.JavaScriptAction, error)
}

// JavaScriptActionWriter provides JavaScript action mutations.
type JavaScriptActionWriter interface {
	CreateJavaScriptActionGen(parentUUID, containmentName string, jsa *genJSA.JavaScriptAction) error
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

// ImageCollectionWriter provides image collection read and write operations.
type ImageCollectionWriter interface {
	ListImageCollections() ([]*types.ImageCollection, error)
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

// ConnectionManager provides connection lifecycle operations.
type ConnectionManager interface {
	Connect(path string) error
	Disconnect() error
	IsConnected() bool
	Path() string
	Version() types.MPRVersion
	ProjectVersion() *types.ProjectVersion
	GetMendixVersion() (string, error)
}

// FolderManager provides folder CRUD operations.
type FolderManager interface {
	ListFolders() ([]*types.FolderInfo, error)
	CreateFolder(folder *model.Folder) error
	DeleteFolder(id model.ID) error
	MoveFolder(id model.ID, newContainerID model.ID) error
}

// ModuleSettingsReader provides read-only module settings access.
type ModuleSettingsReader interface {
	ListModuleSettings() ([]*types.ModuleSettings, error)
	GetModuleSettings(moduleID model.ID) (*types.ModuleSettings, error)
}

// ModuleSettingsWriter provides module settings mutations.
type ModuleSettingsWriter interface {
	UpdateModuleSettings(ms *types.ModuleSettings) error
}

// ServiceWriter provides write operations for OData, REST, business event,
// database connection, and data transformer services.
type ServiceWriter interface {
	CreateConsumedODataService(svc *model.ConsumedODataService) error
	CreatePublishedODataService(svc *model.PublishedODataService) error
	UpdateConsumedODataService(svc *model.ConsumedODataService) error
	UpdatePublishedODataService(svc *model.PublishedODataService) error
	DeleteConsumedODataService(id model.ID) error
	DeletePublishedODataService(id model.ID) error
	CreateConsumedRestService(svc *model.ConsumedRestService) error
	CreatePublishedRestService(svc *model.PublishedRestService) error
	UpdateConsumedRestService(svc *model.ConsumedRestService) error
	UpdatePublishedRestService(svc *model.PublishedRestService) error
	DeleteConsumedRestService(id model.ID) error
	DeletePublishedRestService(id model.ID) error
	CreateBusinessEventService(svc *model.BusinessEventService) error
	UpdateBusinessEventService(svc *model.BusinessEventService) error
	DeleteBusinessEventService(id model.ID) error
	CreateDatabaseConnection(conn *model.DatabaseConnection) error
	UpdateDatabaseConnection(conn *model.DatabaseConnection) error
	MoveDatabaseConnection(conn *model.DatabaseConnection) error
	DeleteDatabaseConnection(id model.ID) error
	CreateDataTransformer(dt *model.DataTransformer) error
	UpdateDataTransformer(dt *model.DataTransformer) error
	DeleteDataTransformer(id model.ID) error
}

// RenameManager provides cross-cutting rename and reference-update operations.
type RenameManager interface {
	UpdateQualifiedNameInAllUnits(oldName, newName string) (int, error)
	RenameReferences(oldName, newName string, dryRun bool) ([]types.RenameHit, error)
	RenameDocumentByName(moduleName, oldName, newName string) error
}

// SecurityProjectManager manages project-level security.
type SecurityProjectManager interface {
	GetProjectSecurityGen() (*genSec.ProjectSecurity, error)
	SetProjectSecurityLevel(unitID model.ID, level string) error
	SetProjectDemoUsersEnabled(unitID model.ID, enabled bool) error
	AddUserRole(unitID model.ID, name string, moduleRoles []string, manageAllRoles bool) error
	AlterUserRoleModuleRoles(unitID model.ID, userRoleName string, add bool, moduleRoles []string) error
	RemoveUserRole(unitID model.ID, name string) error
	AddDemoUser(unitID model.ID, userName, password, entity string, userRoles []string) error
	RemoveDemoUser(unitID model.ID, userName string) error
	SetPasswordPolicy(unitID model.ID, minLength *int32, requireDigit, requireMixedCase, requireSymbol *bool) error
}

// SecurityModuleManager manages module-level security.
type SecurityModuleManager interface {
	GetModuleSecurityGen(moduleID model.ID) (*genSec.ModuleSecurity, error)
	ListModuleSecurityGen() ([]*genSec.ModuleSecurity, error)
	AddModuleRole(unitID model.ID, roleName, description string) error
	RemoveModuleRole(unitID model.ID, roleName string) error
	RemoveModuleRoleFromAllUserRoles(unitID model.ID, qualifiedRole string) (int, error)
}

// SecurityEntityAccessManager manages entity-level access rules.
type SecurityEntityAccessManager interface {
	UpdateAllowedRoles(unitID model.ID, roles []string) error
	UpdatePublishedRestServiceRoles(unitID model.ID, roles []string) error
	RemoveFromAllowedRoles(unitID model.ID, roleName string) (bool, error)
	AddEntityAccessRule(params EntityAccessRuleParams) error
	RemoveEntityAccessRule(unitID model.ID, entityName string, roleNames []string) (int, error)
	RevokeEntityMemberAccess(unitID model.ID, entityName string, roleNames []string, revocation types.EntityAccessRevocation) (int, error)
	RemoveRoleFromAllEntities(unitID model.ID, roleName string) (int, error)
	ReconcileMemberAccesses(unitID model.ID, moduleName string) ([]string, error)
}

// PageModelAccess provides page model read/write access.
type PageModelAccess interface {
	GetPageModel(id model.ID) (*types.PageModel, error)
	GetSnippetModel(id model.ID) (*types.PageModel, error)
	GetLayoutModel(id model.ID) (*types.PageModel, error)
	WritePageModel(id model.ID, m *types.PageModel) error
	WriteSnippetModel(id model.ID, m *types.PageModel) error
}

// PageMutationOperator provides page/layout/snippet mutation capabilities.
type PageMutationOperator interface {
	OpenPageForMutation(unitID model.ID) (PageMutator, error)
}

// WorkflowMutationOperator provides workflow mutation capabilities.
type WorkflowMutationOperator interface {
	OpenWorkflowForMutation(unitID model.ID) (WorkflowMutator, error)
}

// WidgetBuilder provides pluggable widget construction lifecycle.
type WidgetBuilder interface {
	BeginPageBuild()
	EndPageBuild()
	// BuildDataGridDatasource is defined in WidgetSerializationBackend or on MprBackend directly.
}

// ScriptTransactionManager provides atomic script execution.
type ScriptTransactionManager interface {
	BeginScriptTransaction() (ScriptTransaction, error)
}

// AgentEditorOperator provides agent editor CRUD operations.
type AgentEditorOperator interface {
	ListAgentEditorModels() ([]*types.Model, error)
	ListAgentEditorKnowledgeBases() ([]*types.KnowledgeBase, error)
	ListAgentEditorConsumedMCPServices() ([]*types.ConsumedMCPService, error)
	ListAgentEditorAgents() ([]*types.Agent, error)
	CreateAgentEditorModel(m *types.Model) error
	UpdateAgentEditorModel(m *types.Model) error
	DeleteAgentEditorModel(id string) error
	CreateAgentEditorKnowledgeBase(k *types.KnowledgeBase) error
	UpdateAgentEditorKnowledgeBase(k *types.KnowledgeBase) error
	DeleteAgentEditorKnowledgeBase(id string) error
	CreateAgentEditorConsumedMCPService(c *types.ConsumedMCPService) error
	UpdateAgentEditorConsumedMCPService(c *types.ConsumedMCPService) error
	DeleteAgentEditorConsumedMCPService(id string) error
	CreateAgentEditorAgent(a *types.Agent) error
	UpdateAgentEditorAgent(a *types.Agent) error
	DeleteAgentEditorAgent(id string) error
}
