// SPDX-License-Identifier: Apache-2.0

// Package mprbackend provides the MprBackend implementation of
// backend.FullBackend. The package name is "mprbackend" (not "mpr") to
// avoid collision with the sdk/mpr package in import blocks.
package mprbackend

import (
	"log"

	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/linter"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
	"github.com/mendixlabs/mxcli/sdk/agenteditor"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
	"github.com/mendixlabs/mxcli/sdk/javaactions"
	"github.com/mendixlabs/mxcli/sdk/microflows"
	"github.com/mendixlabs/mxcli/sdk/mpr"
	"github.com/mendixlabs/mxcli/sdk/pages"
	"github.com/mendixlabs/mxcli/sdk/security"
	"github.com/mendixlabs/mxcli/sdk/workflows"
)

var _ backend.FullBackend = (*MprBackend)(nil)
var _ linter.LintReader = (*MprBackend)(nil)

// MprBackend implements backend.FullBackend by delegating to mpr.Reader
// and mpr.Writer.
//
// Methods that access reader or writer assume Connect() has been called.
// Calling read/write methods before Connect() will panic with a nil
// pointer dereference. The executor enforces connection state via
// ConnectionBackend.IsConnected() before dispatching handlers.
type MprBackend struct {
	reader     *mpr.Reader
	writer     *mpr.Writer
	msdkWriter modelsdkmpr.UnitWriter
	path       string
}

// New creates a new unconnected MprBackend. Call Connect(path) to open a project.
func New() *MprBackend {
	return &MprBackend{}
}

// Wrap creates an MprBackend that wraps an existing Writer (and its Reader).
// This is used during migration when the Executor already owns the Writer
// and we want to expose it through the Backend interface without opening
// a second connection.
func Wrap(writer *mpr.Writer, path string) *MprBackend {
	db := writer.Reader().DB()
	contentsDir := writer.Reader().ContentsDir()
	mw, err := modelsdkmpr.NewWriterFromDB(db, path, contentsDir)
	if err != nil {
		log.Printf("mprbackend: Wrap: failed to create modelsdk writer for %s: %v", path, err)
	}
	return &MprBackend{
		reader:     writer.Reader(),
		writer:     writer,
		msdkWriter: mw,
		path:       path,
	}
}

// ---------------------------------------------------------------------------
// ConnectionBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) Connect(path string) error {
	w, err := mpr.NewWriter(path)
	if err != nil {
		return err
	}
	db := w.Reader().DB()
	contentsDir := w.Reader().ContentsDir()
	mw, err := modelsdkmpr.NewWriterFromDB(db, path, contentsDir)
	if err != nil {
		_ = w.Close()
		return err
	}
	b.writer = w
	b.reader = w.Reader()
	b.msdkWriter = mw
	b.path = path
	return nil
}

func (b *MprBackend) Disconnect() error {
	if b.writer == nil {
		return nil
	}
	err := b.writer.Close()
	b.writer = nil
	b.reader = nil
	b.msdkWriter = nil
	b.path = ""
	return err
}

func (b *MprBackend) IsConnected() bool { return b.writer != nil }
func (b *MprBackend) Path() string      { return b.path }

// MprReader returns the underlying *mpr.Reader for callers that still
// require direct SDK access (e.g. linter rules). Prefer Backend methods
// for new code.
func (b *MprBackend) MprReader() *mpr.Reader { return b.reader }

func (b *MprBackend) Version() types.MPRVersion { return convertMPRVersion(b.reader.Version()) }
func (b *MprBackend) ProjectVersion() *types.ProjectVersion {
	return b.reader.ProjectVersion()
}
func (b *MprBackend) GetMendixVersion() (string, error) { return b.reader.GetMendixVersion() }

// Commit is a no-op — the MPR writer auto-commits on each write operation.
func (b *MprBackend) Commit() error { return nil }

// ---------------------------------------------------------------------------
// ModuleBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) ListModules() ([]*model.Module, error)        { return b.reader.ListModules() }
func (b *MprBackend) GetModule(id model.ID) (*model.Module, error) { return b.reader.GetModule(id) }
func (b *MprBackend) GetModuleByName(name string) (*model.Module, error) {
	return b.reader.GetModuleByName(name)
}
func (b *MprBackend) CreateModule(module *model.Module) error {
	return b.createModuleViaModelsdk(module)
}
func (b *MprBackend) UpdateModule(module *model.Module) error { return b.updateModuleViaModelsdk(module) }
func (b *MprBackend) DeleteModule(id model.ID) error          { return b.deleteModuleViaModelsdk(id) }
func (b *MprBackend) DeleteModuleWithCleanup(id model.ID, moduleName string) error {
	return b.deleteModuleWithCleanupViaModelsdk(id, moduleName)
}

// ---------------------------------------------------------------------------
// ModuleSettingsBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) ListModuleSettings() ([]*types.ModuleSettings, error) {
	return b.reader.ListModuleSettings()
}
func (b *MprBackend) GetModuleSettings(moduleID model.ID) (*types.ModuleSettings, error) {
	return b.reader.GetModuleSettings(moduleID)
}
func (b *MprBackend) UpdateModuleSettings(ms *types.ModuleSettings) error {
	return b.updateModuleSettingsViaModelsdk(ms)
}

// ---------------------------------------------------------------------------
// FolderBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) ListFolders() ([]*types.FolderInfo, error) {
	return convertFolderInfoSlice(b.reader.ListFolders())
}
func (b *MprBackend) CreateFolder(folder *model.Folder) error {
	return b.createFolderViaModelsdk(folder)
}
func (b *MprBackend) DeleteFolder(id model.ID) error          { return b.deleteFolderViaModelsdk(id) }
func (b *MprBackend) MoveFolder(id model.ID, newContainerID model.ID) error {
	return b.moveFolderViaModelsdk(id, newContainerID)
}

// ---------------------------------------------------------------------------
// DomainModelBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) ListDomainModels() ([]*domainmodel.DomainModel, error) {
	return b.reader.ListDomainModels()
}
func (b *MprBackend) GetDomainModel(moduleID model.ID) (*domainmodel.DomainModel, error) {
	return b.reader.GetDomainModel(moduleID)
}
func (b *MprBackend) GetDomainModelByID(id model.ID) (*domainmodel.DomainModel, error) {
	return b.reader.GetDomainModelByID(id)
}
func (b *MprBackend) UpdateDomainModel(dm *domainmodel.DomainModel) error {
	return b.updateDomainModelViaModelsdk(dm)
}

func (b *MprBackend) CreateEntity(domainModelID model.ID, entity *domainmodel.Entity) error {
	return b.createEntityViaModelsdk(domainModelID, entity)
}
func (b *MprBackend) UpdateEntity(domainModelID model.ID, entity *domainmodel.Entity) error {
	return b.updateEntityViaModelsdk(domainModelID, entity)
}
func (b *MprBackend) DeleteEntity(domainModelID model.ID, entityID model.ID) error {
	return b.deleteEntityViaModelsdk(domainModelID, entityID)
}
func (b *MprBackend) MoveEntity(entity *domainmodel.Entity, sourceDMID, targetDMID model.ID, sourceModuleName, targetModuleName string) ([]string, error) {
	return b.moveEntityViaModelsdk(entity, sourceDMID, targetDMID, sourceModuleName, targetModuleName)
}

func (b *MprBackend) AddAttribute(domainModelID model.ID, entityID model.ID, attr *domainmodel.Attribute) error {
	return b.addAttributeViaModelsdk(domainModelID, entityID, attr)
}
func (b *MprBackend) UpdateAttribute(domainModelID model.ID, entityID model.ID, attr *domainmodel.Attribute) error {
	return b.updateAttributeViaModelsdk(domainModelID, entityID, attr)
}
func (b *MprBackend) DeleteAttribute(domainModelID model.ID, entityID model.ID, attrID model.ID) error {
	return b.deleteAttributeViaModelsdk(domainModelID, entityID, attrID)
}

func (b *MprBackend) CreateAssociation(domainModelID model.ID, assoc *domainmodel.Association) error {
	return b.createAssociationViaModelsdk(domainModelID, assoc)
}
func (b *MprBackend) CreateCrossAssociation(domainModelID model.ID, ca *domainmodel.CrossModuleAssociation) error {
	return b.createCrossAssociationViaModelsdk(domainModelID, ca)
}
func (b *MprBackend) DeleteAssociation(domainModelID model.ID, assocID model.ID) error {
	return b.deleteAssociationViaModelsdk(domainModelID, assocID)
}
func (b *MprBackend) DeleteCrossAssociation(domainModelID model.ID, assocID model.ID) error {
	return b.deleteCrossAssociationViaModelsdk(domainModelID, assocID)
}

func (b *MprBackend) CreateViewEntitySourceDocument(moduleID model.ID, moduleName, docName, oqlQuery, documentation string) (model.ID, error) {
	return b.createViewEntitySourceDocumentViaModelsdk(moduleID, moduleName, docName, oqlQuery, documentation)
}
func (b *MprBackend) DeleteViewEntitySourceDocument(id model.ID) error {
	return b.deleteViewEntitySourceDocumentViaModelsdk(id)
}
func (b *MprBackend) DeleteViewEntitySourceDocumentByName(moduleName, docName string) error {
	return b.deleteViewEntitySourceDocumentByNameViaModelsdk(moduleName, docName)
}
func (b *MprBackend) FindViewEntitySourceDocumentID(moduleName, docName string) (model.ID, error) {
	return b.reader.FindViewEntitySourceDocumentID(moduleName, docName)
}
func (b *MprBackend) FindAllViewEntitySourceDocumentIDs(moduleName, docName string) ([]model.ID, error) {
	return b.reader.FindAllViewEntitySourceDocumentIDs(moduleName, docName)
}
func (b *MprBackend) MoveViewEntitySourceDocument(sourceModuleName string, targetModuleID model.ID, docName string) error {
	return b.moveViewEntitySourceDocumentViaModelsdk(sourceModuleName, targetModuleID, docName)
}
func (b *MprBackend) UpdateOqlQueriesForMovedEntity(oldQualifiedName, newQualifiedName string) (int, error) {
	return b.updateOqlQueriesForMovedEntityViaModelsdk(oldQualifiedName, newQualifiedName)
}
func (b *MprBackend) UpdateEnumerationRefsInAllDomainModels(oldQualifiedName, newQualifiedName string) error {
	return b.updateEnumerationRefsInAllDomainModelsViaModelsdk(oldQualifiedName, newQualifiedName)
}

// ---------------------------------------------------------------------------
// MicroflowBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) ListMicroflows() ([]*microflows.Microflow, error) {
	return b.reader.ListMicroflows()
}
func (b *MprBackend) GetMicroflow(id model.ID) (*microflows.Microflow, error) {
	return b.reader.GetMicroflow(id)
}
func (b *MprBackend) CreateMicroflow(mf *microflows.Microflow) error {
	return b.createMicroflowViaModelsdk(mf)
}
func (b *MprBackend) UpdateMicroflow(mf *microflows.Microflow) error {
	return b.updateMicroflowViaModelsdk(mf)
}
func (b *MprBackend) DeleteMicroflow(id model.ID) error { return b.deleteMicroflowViaModelsdk(id) }
func (b *MprBackend) MoveMicroflow(mf *microflows.Microflow) error {
	return b.moveMicroflowViaModelsdk(mf)
}
func (b *MprBackend) IsRule(qualifiedName string) (bool, error) {
	return b.reader.IsRule(qualifiedName)
}

func (b *MprBackend) ListNanoflows() ([]*microflows.Nanoflow, error) {
	return b.reader.ListNanoflows()
}
func (b *MprBackend) ParseMicroflowFromRaw(raw map[string]any, unitID, containerID model.ID) *microflows.Microflow {
	return mpr.ParseMicroflowFromRaw(raw, unitID, containerID)
}
func (b *MprBackend) ParseMicroflowBSON(contents []byte, unitID, containerID model.ID) (*microflows.Microflow, error) {
	return mpr.ParseMicroflowBSON(contents, unitID, containerID)
}
func (b *MprBackend) GetNanoflow(id model.ID) (*microflows.Nanoflow, error) {
	return b.reader.GetNanoflow(id)
}
func (b *MprBackend) CreateNanoflow(nf *microflows.Nanoflow) error {
	return b.createNanoflowViaModelsdk(nf)
}
func (b *MprBackend) UpdateNanoflow(nf *microflows.Nanoflow) error {
	return b.updateNanoflowViaModelsdk(nf)
}
func (b *MprBackend) DeleteNanoflow(id model.ID) error { return b.deleteNanoflowViaModelsdk(id) }
func (b *MprBackend) MoveNanoflow(nf *microflows.Nanoflow) error {
	return b.moveNanoflowViaModelsdk(nf)
}

// ---------------------------------------------------------------------------
// PageBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) ListPages() ([]*pages.Page, error)        { return b.reader.ListPages() }
func (b *MprBackend) GetPage(id model.ID) (*pages.Page, error) { return b.reader.GetPage(id) }
func (b *MprBackend) CreatePage(page *pages.Page) error        { return b.createPageViaModelsdk(page) }
func (b *MprBackend) UpdatePage(page *pages.Page) error        { return b.updatePageViaModelsdk(page) }
func (b *MprBackend) DeletePage(id model.ID) error             { return b.deletePageViaModelsdk(id) }
func (b *MprBackend) MovePage(page *pages.Page) error          { return b.movePageViaModelsdk(page) }

func (b *MprBackend) ListLayouts() ([]*pages.Layout, error)        { return b.reader.ListLayouts() }
func (b *MprBackend) GetLayout(id model.ID) (*pages.Layout, error) { return b.reader.GetLayout(id) }
func (b *MprBackend) CreateLayout(layout *pages.Layout) error      { return b.createLayoutViaModelsdk(layout) }
func (b *MprBackend) UpdateLayout(layout *pages.Layout) error      { return b.updateLayoutViaModelsdk(layout) }
func (b *MprBackend) DeleteLayout(id model.ID) error               { return b.deleteLayoutViaModelsdk(id) }

func (b *MprBackend) ListSnippets() ([]*pages.Snippet, error) { return b.reader.ListSnippets() }
func (b *MprBackend) CreateSnippet(snippet *pages.Snippet) error {
	return b.createSnippetViaModelsdk(snippet)
}
func (b *MprBackend) UpdateSnippet(snippet *pages.Snippet) error {
	return b.updateSnippetViaModelsdk(snippet)
}
func (b *MprBackend) DeleteSnippet(id model.ID) error          { return b.deleteSnippetViaModelsdk(id) }
func (b *MprBackend) MoveSnippet(snippet *pages.Snippet) error { return b.moveSnippetViaModelsdk(snippet) }

func (b *MprBackend) ListBuildingBlocks() ([]*pages.BuildingBlock, error) {
	return b.reader.ListBuildingBlocks()
}
func (b *MprBackend) ListPageTemplates() ([]*pages.PageTemplate, error) {
	return b.reader.ListPageTemplates()
}

// ---------------------------------------------------------------------------
// EnumerationBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) ListEnumerations() ([]*model.Enumeration, error) {
	return b.reader.ListEnumerations()
}
func (b *MprBackend) GetEnumeration(id model.ID) (*model.Enumeration, error) {
	return b.reader.GetEnumeration(id)
}
func (b *MprBackend) CreateEnumeration(enum *model.Enumeration) error {
	return b.createEnumerationViaModelsdk(enum)
}
func (b *MprBackend) UpdateEnumeration(enum *model.Enumeration) error {
	return b.updateEnumerationViaModelsdk(enum)
}
func (b *MprBackend) MoveEnumeration(enum *model.Enumeration) error {
	return b.moveEnumerationViaModelsdk(enum)
}
func (b *MprBackend) DeleteEnumeration(id model.ID) error {
	return b.deleteEnumerationViaModelsdk(id)
}

// ---------------------------------------------------------------------------
// ConstantBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) ListConstants() ([]*model.Constant, error) { return b.reader.ListConstants() }
func (b *MprBackend) GetConstant(id model.ID) (*model.Constant, error) {
	return b.reader.GetConstant(id)
}
func (b *MprBackend) CreateConstant(constant *model.Constant) error {
	return b.createConstantViaModelsdk(constant)
}
func (b *MprBackend) UpdateConstant(constant *model.Constant) error {
	return b.updateConstantViaModelsdk(constant)
}
func (b *MprBackend) MoveConstant(constant *model.Constant) error {
	return b.moveConstantViaModelsdk(constant)
}
func (b *MprBackend) DeleteConstant(id model.ID) error {
	return b.deleteConstantViaModelsdk(id)
}

// ---------------------------------------------------------------------------
// SecurityBackend (ProjectSecurityBackend + ModuleSecurityBackend + EntityAccessBackend)
// ---------------------------------------------------------------------------

func (b *MprBackend) GetProjectSecurity() (*security.ProjectSecurity, error) {
	return b.reader.GetProjectSecurity()
}
func (b *MprBackend) SetProjectSecurityLevel(unitID model.ID, level string) error {
	return b.setSecurityLevelViaModelsdk(unitID, level)
}
func (b *MprBackend) SetProjectDemoUsersEnabled(unitID model.ID, enabled bool) error {
	return b.setProjectDemoUsersEnabledViaModelsdk(unitID, enabled)
}
func (b *MprBackend) AddUserRole(unitID model.ID, name string, moduleRoles []string, manageAllRoles bool) error {
	return b.addUserRoleViaModelsdk(unitID, name, moduleRoles, manageAllRoles)
}
func (b *MprBackend) AlterUserRoleModuleRoles(unitID model.ID, userRoleName string, add bool, moduleRoles []string) error {
	return b.alterUserRoleModuleRolesViaModelsdk(unitID, userRoleName, add, moduleRoles)
}
func (b *MprBackend) RemoveUserRole(unitID model.ID, name string) error {
	return b.removeUserRoleViaModelsdk(unitID, name)
}
func (b *MprBackend) AddDemoUser(unitID model.ID, userName, password, entity string, userRoles []string) error {
	return b.addDemoUserViaModelsdk(unitID, userName, password, entity, userRoles)
}
func (b *MprBackend) RemoveDemoUser(unitID model.ID, userName string) error {
	return b.removeDemoUserViaModelsdk(unitID, userName)
}

func (b *MprBackend) ListModuleSecurity() ([]*security.ModuleSecurity, error) {
	return b.reader.ListModuleSecurity()
}
func (b *MprBackend) GetModuleSecurity(moduleID model.ID) (*security.ModuleSecurity, error) {
	return b.reader.GetModuleSecurity(moduleID)
}
func (b *MprBackend) AddModuleRole(unitID model.ID, roleName, description string) error {
	return b.addModuleRoleViaModelsdk(unitID, roleName, description)
}
func (b *MprBackend) RemoveModuleRole(unitID model.ID, roleName string) error {
	return b.removeModuleRoleViaModelsdk(unitID, roleName)
}
func (b *MprBackend) RemoveModuleRoleFromAllUserRoles(unitID model.ID, qualifiedRole string) (int, error) {
	return 0, b.removeModuleRoleFromAllUserRolesViaModelsdk(unitID, qualifiedRole)
}

func (b *MprBackend) UpdateAllowedRoles(unitID model.ID, roles []string) error {
	return b.updateAllowedRolesViaModelsdk(unitID, roles)
}
func (b *MprBackend) UpdatePublishedRestServiceRoles(unitID model.ID, roles []string) error {
	return b.updatePublishedRestServiceRolesViaModelsdk(unitID, roles)
}
func (b *MprBackend) RemoveFromAllowedRoles(unitID model.ID, roleName string) (bool, error) {
	return b.removeFromAllowedRolesViaModelsdk(unitID, roleName)
}
func (b *MprBackend) AddEntityAccessRule(params backend.EntityAccessRuleParams) error {
	return b.addEntityAccessRuleViaModelsdk(params.UnitID, params.EntityName, params.RoleNames, params.AllowCreate, params.AllowDelete, params.DefaultMemberAccess, params.XPathConstraint, unconvertEntityMemberAccessSlice(params.MemberAccesses))
}
func (b *MprBackend) RemoveEntityAccessRule(unitID model.ID, entityName string, roleNames []string) (int, error) {
	return b.removeEntityAccessRuleViaModelsdk(unitID, entityName, roleNames)
}
func (b *MprBackend) RevokeEntityMemberAccess(unitID model.ID, entityName string, roleNames []string, revocation types.EntityAccessRevocation) (int, error) {
	return b.revokeEntityMemberAccessViaModelsdk(unitID, entityName, roleNames, unconvertEntityAccessRevocation(revocation))
}
func (b *MprBackend) RemoveRoleFromAllEntities(unitID model.ID, roleName string) (int, error) {
	return b.removeRoleFromAllEntitiesViaModelsdk(unitID, roleName)
}
func (b *MprBackend) ReconcileMemberAccesses(unitID model.ID, moduleName string) (int, error) {
	return b.reconcileMemberAccessesViaModelsdk(unitID, moduleName)
}

// ---------------------------------------------------------------------------
// NavigationBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) ListNavigationDocuments() ([]*types.NavigationDocument, error) {
	return convertNavDocSlice(b.reader.ListNavigationDocuments())
}
func (b *MprBackend) GetNavigation() (*types.NavigationDocument, error) {
	return convertNavDocPtr(b.reader.GetNavigation())
}
func (b *MprBackend) UpdateNavigationProfile(navDocID model.ID, profileName string, spec types.NavigationProfileSpec) error {
	return b.updateNavigationProfileViaModelsdk(navDocID, profileName, unconvertNavProfileSpec(spec))
}

// ---------------------------------------------------------------------------
// ServiceBackend (OData + REST + BusinessEvent + DatabaseConnection + DataTransformer)
// ---------------------------------------------------------------------------

func (b *MprBackend) ListConsumedODataServices() ([]*model.ConsumedODataService, error) {
	return b.reader.ListConsumedODataServices()
}
func (b *MprBackend) ListPublishedODataServices() ([]*model.PublishedODataService, error) {
	return b.reader.ListPublishedODataServices()
}
func (b *MprBackend) CreateConsumedODataService(svc *model.ConsumedODataService) error {
	return b.createConsumedODataServiceViaModelsdk(svc)
}
func (b *MprBackend) UpdateConsumedODataService(svc *model.ConsumedODataService) error {
	return b.updateConsumedODataServiceViaModelsdk(svc)
}
func (b *MprBackend) DeleteConsumedODataService(id model.ID) error {
	return b.deleteConsumedODataServiceViaModelsdk(id)
}
func (b *MprBackend) CreatePublishedODataService(svc *model.PublishedODataService) error {
	return b.createPublishedODataServiceViaModelsdk(svc)
}
func (b *MprBackend) UpdatePublishedODataService(svc *model.PublishedODataService) error {
	return b.updatePublishedODataServiceViaModelsdk(svc)
}
func (b *MprBackend) DeletePublishedODataService(id model.ID) error {
	return b.deletePublishedODataServiceViaModelsdk(id)
}

func (b *MprBackend) ListConsumedRestServices() ([]*model.ConsumedRestService, error) {
	return b.reader.ListConsumedRestServices()
}
func (b *MprBackend) ListPublishedRestServices() ([]*model.PublishedRestService, error) {
	return b.reader.ListPublishedRestServices()
}
func (b *MprBackend) CreateConsumedRestService(svc *model.ConsumedRestService) error {
	return b.createConsumedRestServiceViaModelsdk(svc)
}
func (b *MprBackend) UpdateConsumedRestService(svc *model.ConsumedRestService) error {
	return b.updateConsumedRestServiceViaModelsdk(svc)
}
func (b *MprBackend) DeleteConsumedRestService(id model.ID) error {
	return b.deleteConsumedRestServiceViaModelsdk(id)
}
func (b *MprBackend) CreatePublishedRestService(svc *model.PublishedRestService) error {
	return b.createPublishedRestServiceViaModelsdk(svc)
}
func (b *MprBackend) UpdatePublishedRestService(svc *model.PublishedRestService) error {
	return b.updatePublishedRestServiceViaModelsdk(svc)
}
func (b *MprBackend) DeletePublishedRestService(id model.ID) error {
	return b.deletePublishedRestServiceViaModelsdk(id)
}

func (b *MprBackend) ListBusinessEventServices() ([]*model.BusinessEventService, error) {
	return b.reader.ListBusinessEventServices()
}
func (b *MprBackend) CreateBusinessEventService(svc *model.BusinessEventService) error {
	return b.createBusinessEventServiceViaModelsdk(svc)
}
func (b *MprBackend) UpdateBusinessEventService(svc *model.BusinessEventService) error {
	return b.updateBusinessEventServiceViaModelsdk(svc)
}
func (b *MprBackend) DeleteBusinessEventService(id model.ID) error {
	return b.deleteBusinessEventServiceViaModelsdk(id)
}

func (b *MprBackend) ListDatabaseConnections() ([]*model.DatabaseConnection, error) {
	return b.reader.ListDatabaseConnections()
}
func (b *MprBackend) CreateDatabaseConnection(conn *model.DatabaseConnection) error {
	return b.createDatabaseConnectionViaModelsdk(conn)
}
func (b *MprBackend) UpdateDatabaseConnection(conn *model.DatabaseConnection) error {
	return b.updateDatabaseConnectionViaModelsdk(conn)
}
func (b *MprBackend) MoveDatabaseConnection(conn *model.DatabaseConnection) error {
	return b.moveDatabaseConnectionViaModelsdk(conn)
}
func (b *MprBackend) DeleteDatabaseConnection(id model.ID) error {
	return b.deleteDatabaseConnectionViaModelsdk(id)
}

func (b *MprBackend) ListDataTransformers() ([]*model.DataTransformer, error) {
	return b.reader.ListDataTransformers()
}
func (b *MprBackend) CreateDataTransformer(dt *model.DataTransformer) error {
	return b.createDataTransformerViaModelsdk(dt)
}
func (b *MprBackend) UpdateDataTransformer(dt *model.DataTransformer) error {
	return b.updateDataTransformerViaModelsdk(dt)
}
func (b *MprBackend) DeleteDataTransformer(id model.ID) error {
	return b.deleteDataTransformerViaModelsdk(id)
}

// ---------------------------------------------------------------------------
// MappingBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) ListImportMappings() ([]*model.ImportMapping, error) {
	return b.reader.ListImportMappings()
}
func (b *MprBackend) GetImportMappingByQualifiedName(moduleName, name string) (*model.ImportMapping, error) {
	return b.reader.GetImportMappingByQualifiedName(moduleName, name)
}
func (b *MprBackend) CreateImportMapping(im *model.ImportMapping) error {
	return b.createImportMappingViaModelsdk(im)
}
func (b *MprBackend) UpdateImportMapping(im *model.ImportMapping) error {
	return b.updateImportMappingViaModelsdk(im)
}
func (b *MprBackend) DeleteImportMapping(id model.ID) error {
	return b.deleteImportMappingViaModelsdk(id)
}
func (b *MprBackend) MoveImportMapping(im *model.ImportMapping) error {
	return b.moveImportMappingViaModelsdk(im)
}

func (b *MprBackend) ListExportMappings() ([]*model.ExportMapping, error) {
	return b.reader.ListExportMappings()
}
func (b *MprBackend) GetExportMappingByQualifiedName(moduleName, name string) (*model.ExportMapping, error) {
	return b.reader.GetExportMappingByQualifiedName(moduleName, name)
}
func (b *MprBackend) CreateExportMapping(em *model.ExportMapping) error {
	return b.createExportMappingViaModelsdk(em)
}
func (b *MprBackend) UpdateExportMapping(em *model.ExportMapping) error {
	return b.updateExportMappingViaModelsdk(em)
}
func (b *MprBackend) DeleteExportMapping(id model.ID) error {
	return b.deleteExportMappingViaModelsdk(id)
}
func (b *MprBackend) MoveExportMapping(em *model.ExportMapping) error {
	return b.moveExportMappingViaModelsdk(em)
}

func (b *MprBackend) ListJsonStructures() ([]*types.JsonStructure, error) {
	return convertJsonStructureSlice(b.reader.ListJsonStructures())
}
func (b *MprBackend) GetJsonStructureByQualifiedName(moduleName, name string) (*types.JsonStructure, error) {
	return convertJsonStructurePtr(b.reader.GetJsonStructureByQualifiedName(moduleName, name))
}
func (b *MprBackend) CreateJsonStructure(js *types.JsonStructure) error {
	return b.createJsonStructureViaModelsdk(js)
}
func (b *MprBackend) UpdateJsonStructure(js *types.JsonStructure) error {
	return b.updateJsonStructureViaModelsdk(js)
}
func (b *MprBackend) DeleteJsonStructure(id string) error {
	return b.deleteJsonStructureViaModelsdk(id)
}

// ---------------------------------------------------------------------------
// JavaBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) ListJavaActions() ([]*types.JavaAction, error) {
	return convertJavaActionSlice(b.reader.ListJavaActions())
}
func (b *MprBackend) ListJavaActionsFull() ([]*javaactions.JavaAction, error) {
	return b.reader.ListJavaActionsFull()
}
func (b *MprBackend) ListJavaScriptActions() ([]*types.JavaScriptAction, error) {
	return convertJavaScriptActionSlice(b.reader.ListJavaScriptActions())
}
func (b *MprBackend) ReadJavaActionByName(qualifiedName string) (*javaactions.JavaAction, error) {
	return b.reader.ReadJavaActionByName(qualifiedName)
}
func (b *MprBackend) ReadJavaScriptActionByName(qualifiedName string) (*types.JavaScriptAction, error) {
	return convertJavaScriptActionPtr(b.reader.ReadJavaScriptActionByName(qualifiedName))
}
func (b *MprBackend) CreateJavaAction(ja *javaactions.JavaAction) error {
	return b.createJavaActionViaModelsdk(ja)
}
func (b *MprBackend) UpdateJavaAction(ja *javaactions.JavaAction) error {
	return b.updateJavaActionViaModelsdk(ja)
}
func (b *MprBackend) DeleteJavaAction(id model.ID) error {
	return b.deleteJavaActionViaModelsdk(id)
}
func (b *MprBackend) WriteJavaSourceFile(moduleName, actionName string, javaCode string, params []*javaactions.JavaActionParameter, returnType javaactions.CodeActionReturnType, extraImports []string, extraCode string) error {
	return b.writer.WriteJavaSourceFile(moduleName, actionName, javaCode, params, returnType, extraImports, extraCode)
}
func (b *MprBackend) DeleteJavaSourceFile(moduleName, actionName string) error {
	return b.writer.DeleteJavaSourceFile(moduleName, actionName)
}
func (b *MprBackend) RenameJavaSourceFile(moduleName, oldName, newName string) error {
	return b.writer.RenameJavaSourceFile(moduleName, oldName, newName)
}

// ReadJavaSourceFile delegates to writer because the mpr SDK places this
// read operation on Writer (it needs write-transaction access to the
// contents directory).
func (b *MprBackend) ReadJavaSourceFile(moduleName, actionName string) (string, error) {
	return b.writer.ReadJavaSourceFile(moduleName, actionName)
}

// ---------------------------------------------------------------------------
// WorkflowBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) ListWorkflows() ([]*workflows.Workflow, error) {
	return b.reader.ListWorkflows()
}
func (b *MprBackend) GetWorkflow(id model.ID) (*workflows.Workflow, error) {
	return b.reader.GetWorkflow(id)
}
func (b *MprBackend) CreateWorkflow(wf *workflows.Workflow) error {
	return b.createWorkflowViaModelsdk(wf)
}
func (b *MprBackend) UpdateWorkflow(wf *workflows.Workflow) error {
	return b.updateWorkflowViaModelsdk(wf)
}
func (b *MprBackend) DeleteWorkflow(id model.ID) error { return b.deleteWorkflowViaModelsdk(id) }

// ---------------------------------------------------------------------------
// SettingsBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) GetProjectSettings() (*model.ProjectSettings, error) {
	return b.reader.GetProjectSettings()
}
func (b *MprBackend) UpdateProjectSettings(ps *model.ProjectSettings) error {
	return b.updateProjectSettingsViaModelsdk(ps)
}

// ---------------------------------------------------------------------------
// ImageBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) ListImageCollections() ([]*types.ImageCollection, error) {
	return convertImageCollectionSlice(b.reader.ListImageCollections())
}
func (b *MprBackend) CreateImageCollection(ic *types.ImageCollection) error {
	return b.createImageCollectionViaModelsdk(ic)
}
func (b *MprBackend) UpdateImageCollection(ic *types.ImageCollection) error {
	return b.updateImageCollectionViaModelsdk(ic)
}
func (b *MprBackend) DeleteImageCollection(id string) error {
	return b.deleteImageCollectionViaModelsdk(id)
}

// ---------------------------------------------------------------------------
// ScheduledEventBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) ListScheduledEvents() ([]*model.ScheduledEvent, error) {
	return b.reader.ListScheduledEvents()
}
func (b *MprBackend) GetScheduledEvent(id model.ID) (*model.ScheduledEvent, error) {
	return b.reader.GetScheduledEvent(id)
}

// ---------------------------------------------------------------------------
// RenameBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) UpdateQualifiedNameInAllUnits(oldName, newName string) (int, error) {
	return b.updateQualifiedNameInAllUnitsViaModelsdk(oldName, newName)
}
func (b *MprBackend) RenameReferences(oldName, newName string, dryRun bool) ([]types.RenameHit, error) {
	return convertRenameHitSlice(b.renameReferencesViaModelsdk(oldName, newName, dryRun))
}
func (b *MprBackend) RenameDocumentByName(moduleName, oldName, newName string) error {
	return b.renameDocumentByNameViaModelsdk(moduleName, oldName, newName)
}

// ---------------------------------------------------------------------------
// RawUnitBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) GetRawUnit(id model.ID) (map[string]any, error) {
	return b.reader.GetRawUnit(id)
}
func (b *MprBackend) GetRawUnitBytes(id model.ID) ([]byte, error) {
	return b.reader.GetRawUnitBytes(id)
}
func (b *MprBackend) ListRawUnitsByType(typePrefix string) ([]*types.RawUnit, error) {
	return convertRawUnitSlice(b.reader.ListRawUnitsByType(typePrefix))
}
func (b *MprBackend) ListRawUnits(objectType string) ([]*types.RawUnitInfo, error) {
	return convertRawUnitInfoSlice(b.reader.ListRawUnits(objectType))
}
func (b *MprBackend) GetRawUnitByName(objectType, qualifiedName string) (*types.RawUnitInfo, error) {
	return convertRawUnitInfoPtr(b.reader.GetRawUnitByName(objectType, qualifiedName))
}
func (b *MprBackend) GetRawMicroflowByName(qualifiedName string) ([]byte, error) {
	return b.reader.GetRawMicroflowByName(qualifiedName)
}
func (b *MprBackend) UpdateRawUnit(unitID string, contents []byte) error {
	if b.msdkWriter == nil {
		return b.writer.UpdateRawUnit(unitID, contents)
	}
	return b.msdkWriter.UpdateRawUnit(unitID, contents)
}

// ---------------------------------------------------------------------------
// MetadataBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) ListAllUnitIDs() ([]string, error) { return b.reader.ListAllUnitIDs() }
func (b *MprBackend) ListUnits() ([]*types.UnitInfo, error) {
	return convertUnitInfoSlice(b.reader.ListUnits())
}
func (b *MprBackend) GetUnitTypes() (map[string]int, error) { return b.reader.GetUnitTypes() }
func (b *MprBackend) GetProjectRootID() (string, error)     { return b.reader.GetProjectRootID() }
func (b *MprBackend) ContentsDir() string                   { return b.reader.ContentsDir() }
func (b *MprBackend) ExportJSON() ([]byte, error)           { return b.reader.ExportJSON() }
func (b *MprBackend) InvalidateCache()                      { b.reader.InvalidateCache() }

// ---------------------------------------------------------------------------
// WidgetBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) FindCustomWidgetType(widgetID string) (*types.RawCustomWidgetType, error) {
	return convertRawCustomWidgetTypePtr(b.reader.FindCustomWidgetType(widgetID))
}
func (b *MprBackend) FindAllCustomWidgetTypes(widgetID string) ([]*types.RawCustomWidgetType, error) {
	return convertRawCustomWidgetTypeSlice(b.reader.FindAllCustomWidgetTypes(widgetID))
}

// ---------------------------------------------------------------------------
// AgentEditorBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) ListAgentEditorModels() ([]*agenteditor.Model, error) {
	return b.reader.ListAgentEditorModels()
}
func (b *MprBackend) ListAgentEditorKnowledgeBases() ([]*agenteditor.KnowledgeBase, error) {
	return b.reader.ListAgentEditorKnowledgeBases()
}
func (b *MprBackend) ListAgentEditorConsumedMCPServices() ([]*agenteditor.ConsumedMCPService, error) {
	return b.reader.ListAgentEditorConsumedMCPServices()
}
func (b *MprBackend) ListAgentEditorAgents() ([]*agenteditor.Agent, error) {
	return b.reader.ListAgentEditorAgents()
}
func (b *MprBackend) CreateAgentEditorModel(m *agenteditor.Model) error {
	return b.createAgentEditorModelViaModelsdk(m)
}
func (b *MprBackend) UpdateAgentEditorModel(m *agenteditor.Model) error {
	return b.updateAgentEditorModelViaModelsdk(m)
}
func (b *MprBackend) DeleteAgentEditorModel(id string) error {
	return b.deleteAgentEditorModelViaModelsdk(id)
}
func (b *MprBackend) CreateAgentEditorKnowledgeBase(k *agenteditor.KnowledgeBase) error {
	return b.createAgentEditorKnowledgeBaseViaModelsdk(k)
}
func (b *MprBackend) UpdateAgentEditorKnowledgeBase(k *agenteditor.KnowledgeBase) error {
	return b.updateAgentEditorKnowledgeBaseViaModelsdk(k)
}
func (b *MprBackend) DeleteAgentEditorKnowledgeBase(id string) error {
	return b.deleteAgentEditorKnowledgeBaseViaModelsdk(id)
}
func (b *MprBackend) CreateAgentEditorConsumedMCPService(c *agenteditor.ConsumedMCPService) error {
	return b.createAgentEditorConsumedMCPServiceViaModelsdk(c)
}
func (b *MprBackend) UpdateAgentEditorConsumedMCPService(c *agenteditor.ConsumedMCPService) error {
	return b.updateAgentEditorConsumedMCPServiceViaModelsdk(c)
}
func (b *MprBackend) DeleteAgentEditorConsumedMCPService(id string) error {
	return b.deleteAgentEditorConsumedMCPServiceViaModelsdk(id)
}
func (b *MprBackend) CreateAgentEditorAgent(a *agenteditor.Agent) error {
	return b.createAgentEditorAgentViaModelsdk(a)
}
func (b *MprBackend) UpdateAgentEditorAgent(a *agenteditor.Agent) error {
	return b.updateAgentEditorAgentViaModelsdk(a)
}
func (b *MprBackend) DeleteAgentEditorAgent(id string) error {
	return b.deleteAgentEditorAgentViaModelsdk(id)
}

// ---------------------------------------------------------------------------
// PageMutationBackend — implemented in page_mutator.go
// ---------------------------------------------------------------------------

// OpenPageForMutation is implemented in page_mutator.go.

// ---------------------------------------------------------------------------
// WorkflowMutationBackend

// OpenWorkflowForMutation is implemented in workflow_mutator.go.
func (b *MprBackend) OpenWorkflowForMutation(unitID model.ID) (backend.WorkflowMutator, error) {
	return b.openWorkflowForMutation(unitID)
}

// ---------------------------------------------------------------------------
// WidgetSerializationBackend

func (b *MprBackend) SerializeWidget(w pages.Widget) (any, error) {
	return mpr.SerializeWidget(w), nil
}

func (b *MprBackend) SerializeClientAction(a pages.ClientAction) (any, error) {
	return mpr.SerializeClientAction(a), nil
}

func (b *MprBackend) SerializeDataSource(ds pages.DataSource) (any, error) {
	return mpr.SerializeCustomWidgetDataSource(ds), nil
}

func (b *MprBackend) SerializeWorkflowActivity(a workflows.WorkflowActivity) (any, error) {
	return mpr.SerializeWorkflowActivity(a), nil
}
