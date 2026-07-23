// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/mdl/backend"
	mprrepos "github.com/mendixlabs/mxcli/mdl/backend/mpr/repos"
	"github.com/mendixlabs/mxcli/mdl/backend/unitstore"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	genJSA "github.com/mendixlabs/mxcli/modelsdk/gen/javascriptactions"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
	"github.com/mendixlabs/mxcli/modelsdk/mprread"
	mdlversion "github.com/mendixlabs/mxcli/modelsdk/version"
)

// ---------------------------------------------------------------------------
// DomainModelBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) DeleteEntity(domainModelID model.ID, entityID model.ID) error {
	return b.deleteEntityViaModelsdk(domainModelID, entityID)
}
func (b *MprBackend) MoveEntityGen(entity *genDm.Entity, sourceDMID, targetDMID model.ID, sourceModuleName, targetModuleName string) ([]string, error) {
	return b.moveEntityGen(sourceDMID, targetDMID, sourceModuleName, targetModuleName, entity)
}

func (b *MprBackend) DeleteAttribute(domainModelID model.ID, entityID model.ID, attrID model.ID) error {
	return b.deleteAttributeViaModelsdk(domainModelID, entityID, attrID)
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
func (b *MprBackend) UpdateViewEntitySourceDocument(moduleName, docName, oqlQuery, documentation string) error {
	return b.updateViewEntitySourceDocumentViaModelsdk(moduleName, docName, oqlQuery, documentation)
}
func (b *MprBackend) FindViewEntitySourceDocumentID(moduleName, docName string) (model.ID, error) {
	return b.msdkReader.FindViewEntitySourceDocumentID(moduleName, docName)
}
func (b *MprBackend) FindAllViewEntitySourceDocumentIDs(moduleName, docName string) ([]model.ID, error) {
	return b.msdkReader.FindAllViewEntitySourceDocumentIDs(moduleName, docName)
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
//
// Followup E6 retired Get / Create / Update / Move / Parse on the
// FullBackend interface; Followup F3 retired the sdk-typed
// ListMicroflows / GetMicroflow / ListNanoflows. Production routes
// through ctx.Microflows / ctx.Nanoflows (modelsdk-native repos)
// directly. The remaining surface keeps the gen-typed reads and the
// three small fallbacks (Delete*, IsRule) consumed by mock-only test
// contexts that don't wire ctx.Microflows.

func (b *MprBackend) DeleteMicroflow(id model.ID) error { return b.deleteMicroflowViaModelsdk(id) }
func (b *MprBackend) IsRule(qualifiedName string) (bool, error) {
	b.initSubBackends()
	return b.microflows.IsRule(qualifiedName)
}

func (b *MprBackend) DeleteNanoflow(id model.ID) error { return b.deleteNanoflowViaModelsdk(id) }

// ListMicroflowsGen routes through the modelsdk-native microflow repo
// (b.Microflows()), returning gen-typed values. Returns an error if the
// modelsdk writer is unavailable (backend not connected).
func (b *MprBackend) ListMicroflowsGen() ([]*genMf.Microflow, error) {
	b.initSubBackends()
	return b.microflows.ListMicroflowsGen()
}

// ListNanoflowsGen routes through the modelsdk-native nanoflow repo
// (b.Nanoflows()). Empty moduleID means "all modules".
func (b *MprBackend) ListNanoflowsGen() ([]*genMf.Nanoflow, error) {
	b.initSubBackends()
	return b.microflows.ListNanoflowsGen()
}

// GetMicroflowGen fetches a single microflow body by ID as a
// modelsdk-native gen object via b.Microflows().Get. Linter rules and
// the catalog's per-flow walks consume this. Returns (nil, nil) when
// the modelsdk writer is unavailable so callers can fall through to a
// no-op rather than failing the entire build.
func (b *MprBackend) GetMicroflowGen(id model.ID) (*genMf.Microflow, error) {
	b.initSubBackends()
	return b.microflows.GetMicroflowGen(id)
}

// ---------------------------------------------------------------------------
// PageBackend
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Stage 3.3.5.C1 — gen-typed Page / Layout / Snippet surface
//
// Each method routes through the gen-native repos
// (mdl/backend/mpr/repos/{pages,layouts,snippets}.go) using
// `mprrepos.NewPageRepository(w)` etc. The legacy sdk-typed siblings
// were retired in Stage 3.3.5.E1.

func (b *MprBackend) ListPagesGen() ([]*genPg.Page, error) {
	b.initSubBackends()
	return b.pages.ListPagesGen()
}

func (b *MprBackend) GetPageGen(id model.ID) (*genPg.Page, error) {
	b.initSubBackends()
	return b.pages.GetPageGen(id)
}

func (b *MprBackend) CreatePageGen(parentUUID, containmentName string, page *genPg.Page) error {
	if page == nil {
		return fmt.Errorf("CreatePageGen: nil Page")
	}
	if b.writer == nil {
		return fmt.Errorf("CreatePageGen: no modelsdk writer")
	}
	return mprrepos.NewPageRepository(b.writer).Create(parentUUID, containmentName, page)
}

func (b *MprBackend) UpdatePageGen(page *genPg.Page) error {
	if page == nil {
		return fmt.Errorf("UpdatePageGen: nil Page")
	}
	if b.writer == nil {
		return fmt.Errorf("UpdatePageGen: no modelsdk writer")
	}
	return mprrepos.NewPageRepository(b.writer).Update(page)
}

func (b *MprBackend) ListLayoutsGen() ([]*genPg.Layout, error) {
	b.initSubBackends()
	return b.pages.ListLayoutsGen()
}

func (b *MprBackend) GetLayoutGen(id model.ID) (*genPg.Layout, error) {
	b.initSubBackends()
	return b.pages.GetLayoutGen(id)
}

func (b *MprBackend) CreateLayoutGen(parentUUID, containmentName string, layout *genPg.Layout) error {
	if layout == nil {
		return fmt.Errorf("CreateLayoutGen: nil Layout")
	}
	if b.writer == nil {
		return fmt.Errorf("CreateLayoutGen: no modelsdk writer")
	}
	return mprrepos.NewLayoutRepository(b.writer).Create(parentUUID, containmentName, layout)
}

func (b *MprBackend) UpdateLayoutGen(layout *genPg.Layout) error {
	if layout == nil {
		return fmt.Errorf("UpdateLayoutGen: nil Layout")
	}
	if b.writer == nil {
		return fmt.Errorf("UpdateLayoutGen: no modelsdk writer")
	}
	return mprrepos.NewLayoutRepository(b.writer).Update(layout)
}

func (b *MprBackend) ListSnippetsGen() ([]*genPg.Snippet, error) {
	b.initSubBackends()
	return b.pages.ListSnippetsGen()
}

func (b *MprBackend) GetSnippetGen(id model.ID) (*genPg.Snippet, error) {
	b.initSubBackends()
	return b.pages.GetSnippetGen(id)
}

func (b *MprBackend) CreateSnippetGen(parentUUID, containmentName string, snippet *genPg.Snippet) error {
	if snippet == nil {
		return fmt.Errorf("CreateSnippetGen: nil Snippet")
	}
	if b.writer == nil {
		return fmt.Errorf("CreateSnippetGen: no modelsdk writer")
	}
	return mprrepos.NewSnippetRepository(b.writer).Create(parentUUID, containmentName, snippet)
}

func (b *MprBackend) UpdateSnippetGen(snippet *genPg.Snippet) error {
	if snippet == nil {
		return fmt.Errorf("UpdateSnippetGen: nil Snippet")
	}
	if b.writer == nil {
		return fmt.Errorf("UpdateSnippetGen: no modelsdk writer")
	}
	return mprrepos.NewSnippetRepository(b.writer).Update(snippet)
}

// GetPageContainerUUID exposes the gen-native PageRepository's
// GetContainerUUID lookup on the FullBackend surface so lint rules
// (and other callers without direct repo access) can resolve a Page's
// parent container without re-implementing the SQL probe.
func (b *MprBackend) GetPageContainerUUID(id model.ID) (model.ID, error) {
	b.initSubBackends()
	return b.pages.GetPageContainerUUID(id)
}

// Stage 3.3.5.D5.c gen-typed delete + move surface. All four methods
// route through the modelsdk writer's DeleteUnit / UpdateUnitContainer
// directly — there is no per-element BSON shaping, so no separate
// "ViaModelsdk" helper is needed. Mirrors the workflow / agenteditor
// equivalents from Stage 3.3.3.E1 / Stage 3.3.4 C7.

func (b *MprBackend) DeletePageGen(id model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("DeletePageGen: no modelsdk writer")
	}
	return b.msdkWriter.DeleteUnit(string(id))
}

func (b *MprBackend) MoveDocumentGen(id, containerID model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("MoveDocumentGen: no modelsdk writer")
	}
	return b.msdkWriter.UpdateUnitContainer(string(id), string(containerID))
}

func (b *MprBackend) MovePageGen(id, containerID model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("MovePageGen: no modelsdk writer")
	}
	return b.msdkWriter.UpdateUnitContainer(string(id), string(containerID))
}

func (b *MprBackend) DeleteLayoutGen(id model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("DeleteLayoutGen: no modelsdk writer")
	}
	return b.msdkWriter.DeleteUnit(string(id))
}

func (b *MprBackend) MoveLayoutGen(id, containerID model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("MoveLayoutGen: no modelsdk writer")
	}
	return b.msdkWriter.UpdateUnitContainer(string(id), string(containerID))
}

func (b *MprBackend) DeleteSnippetGen(id model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("DeleteSnippetGen: no modelsdk writer")
	}
	return b.msdkWriter.DeleteUnit(string(id))
}

func (b *MprBackend) MoveSnippetGen(id, containerID model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("MoveSnippetGen: no modelsdk writer")
	}
	return b.msdkWriter.UpdateUnitContainer(string(id), string(containerID))
}

// ---------------------------------------------------------------------------
// EnumerationBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) ListEnumerations() ([]*model.Enumeration, error) {
	b.initSubBackends()
	return b.enumerations.ListEnumerations()
}
func (b *MprBackend) GetEnumeration(id model.ID) (*model.Enumeration, error) {
	b.initSubBackends()
	return b.enumerations.GetEnumeration(id)
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

func (b *MprBackend) ListConstants() ([]*model.Constant, error) {
	b.initSubBackends()
	return b.constants.ListConstants()
}
func (b *MprBackend) GetConstant(id model.ID) (*model.Constant, error) {
	b.initSubBackends()
	return b.constants.GetConstant(id)
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
// JavaBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) DeleteJavaAction(id model.ID) error {
	return b.deleteJavaActionViaModelsdk(id)
}
func (b *MprBackend) DeleteJavaSourceFile(moduleName, actionName string) error {
	return b.deleteJavaSourceFileViaPath(moduleName, actionName)
}
func (b *MprBackend) RenameJavaSourceFile(moduleName, oldName, newName string) error {
	return b.renameJavaSourceFileViaPath(moduleName, oldName, newName)
}

// ── Stage 3.3.2.C3 gen-typed siblings ─────────────────────────────────
// List/Read route through the modelsdk-native repo (introduced in A0).
// Create/Update delegate to the repo's Phase D stubs (return descriptive
// errors until D2/D3 land). WriteJavaSourceFileGen routes through the
// existing path-based writer with gen-typed parameters.

func (b *MprBackend) ListJavaActionsGen() ([]*genJA.JavaAction, error) {
	b.initSubBackends()
	return b.java.ListJavaActionsGen()
}

func (b *MprBackend) ReadJavaActionByNameGen(qualifiedName string) (*genJA.JavaAction, error) {
	b.initSubBackends()
	return b.java.ReadJavaActionByNameGen(qualifiedName)
}

// CreateJavaActionGen writes a gen-typed JavaAction directly via the
// gen-native repo Create (landed in Stage 3.3.2.D, commit c5695850).
// The previous bridge through createJavaActionViaModelsdk +
// genJavaActionToSDK is retired — all collection fields
// (ActionParametersItems, return type, type parameters) now serialize
// through modelsdk/codec rather than sdk/mpr.SerializeJavaAction.
func (b *MprBackend) CreateJavaActionGen(parentUUID, containmentName string, ja *genJA.JavaAction) error {
	if ja == nil {
		return fmt.Errorf("CreateJavaActionGen: nil JavaAction")
	}
	if b.writer == nil {
		return fmt.Errorf("CreateJavaActionGen: no modelsdk writer")
	}
	return mprrepos.NewJavaActionRepository(b.writer).Create(parentUUID, containmentName, ja)
}

// UpdateJavaActionGen mirrors CreateJavaActionGen — gen-native repo Update.
func (b *MprBackend) UpdateJavaActionGen(ja *genJA.JavaAction) error {
	if ja == nil {
		return fmt.Errorf("UpdateJavaActionGen: nil JavaAction")
	}
	if b.writer == nil {
		return fmt.Errorf("UpdateJavaActionGen: nil JavaAction")
	}
	return mprrepos.NewJavaActionRepository(b.writer).Update(ja)
}

func (b *MprBackend) WriteJavaSourceFileGen(moduleName, actionName string, javaCode string, params []*genJA.JavaActionParameter, returnType element.Element, extraImports []string, extraCode string) error {
	return b.writeJavaSourceFileViaPathGen(moduleName, actionName, javaCode, params, returnType, extraImports, extraCode)
}

func (b *MprBackend) ListJavaScriptActionsGen() ([]*genJSA.JavaScriptAction, error) {
	b.initSubBackends()
	return b.java.ListJavaScriptActionsGen()
}

func (b *MprBackend) ReadJavaScriptActionByNameGen(qualifiedName string) (*genJSA.JavaScriptAction, error) {
	b.initSubBackends()
	return b.java.ReadJavaScriptActionByNameGen(qualifiedName)
}

func (b *MprBackend) CreateJavaScriptActionGen(parentUUID, containmentName string, jsa *genJSA.JavaScriptAction) error {
	if jsa == nil {
		return fmt.Errorf("CreateJavaScriptActionGen: nil JavaScriptAction")
	}
	if b.writer == nil {
		return fmt.Errorf("CreateJavaScriptActionGen: no modelsdk writer")
	}
	return mprrepos.NewJavaScriptActionRepository(b.writer).Create(parentUUID, containmentName, jsa)
}

func (b *MprBackend) UpdateJavaScriptActionGen(jsa *genJSA.JavaScriptAction) error {
	if jsa == nil {
		return fmt.Errorf("UpdateJavaScriptActionGen: nil JavaScriptAction")
	}
	if b.writer == nil {
		return fmt.Errorf("UpdateJavaScriptActionGen: no modelsdk writer")
	}
	return mprrepos.NewJavaScriptActionRepository(b.writer).Update(jsa)
}

func (b *MprBackend) ReadJavaSourceFile(moduleName, actionName string) (string, error) {
	return b.readJavaSourceFileViaPath(moduleName, actionName)
}

// ---------------------------------------------------------------------------
// WorkflowBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) DeleteWorkflow(id model.ID) error { return b.deleteWorkflowViaModelsdk(id) }

// Stage 3.3.3.C1 — gen-typed Workflow surface.
//
// All four methods route through the gen-native workflowRepo
// (mdl/backend/mpr/repos/workflows.go) using `mprrepos.NewWorkflowRepository(w)`.
// Stage 3.3.3.E1 retired the legacy sdk-typed siblings; only the
// pure-ID DeleteWorkflow remains alongside this gen-typed quartet.

func (b *MprBackend) ListWorkflowsGen() ([]*genWf.Workflow, error) {
	b.initSubBackends()
	return b.workflows.ListWorkflowsGen()
}

func (b *MprBackend) GetWorkflowGen(id model.ID) (*genWf.Workflow, error) {
	b.initSubBackends()
	return b.workflows.GetWorkflowGen(id)
}

func (b *MprBackend) CreateWorkflowGen(parentUUID, containmentName string, wf *genWf.Workflow) error {
	if wf == nil {
		return fmt.Errorf("CreateWorkflowGen: nil Workflow")
	}
	if b.writer == nil {
		return fmt.Errorf("CreateWorkflowGen: no modelsdk writer")
	}
	return mprrepos.NewWorkflowRepository(b.writer).Create(parentUUID, containmentName, wf)
}

func (b *MprBackend) UpdateWorkflowGen(wf *genWf.Workflow) error {
	if wf == nil {
		return fmt.Errorf("UpdateWorkflowGen: nil Workflow")
	}
	if b.writer == nil {
		return fmt.Errorf("UpdateWorkflowGen: no modelsdk writer")
	}
	return mprrepos.NewWorkflowRepository(b.writer).Update(wf)
}

// ---------------------------------------------------------------------------
// ScheduledEventBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) ListScheduledEvents() ([]*model.ScheduledEvent, error) {
	b.initSubBackends()
	return b.scheduledEvents.ListScheduledEvents()
}
func (b *MprBackend) GetScheduledEvent(id model.ID) (*model.ScheduledEvent, error) {
	b.initSubBackends()
	return b.scheduledEvents.GetScheduledEvent(id)
}

// ---------------------------------------------------------------------------
// RenameBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) UpdateQualifiedNameInAllUnits(oldName, newName string) (int, error) {
	return b.updateQualifiedNameInAllUnitsViaModelsdk(oldName, newName)
}
func (b *MprBackend) RenameReferences(oldName, newName string, dryRun bool) ([]types.RenameHit, error) {
	return b.renameReferencesViaModelsdk(oldName, newName, dryRun)
}
func (b *MprBackend) RenameDocumentByName(moduleName, oldName, newName string) error {
	return b.renameDocumentByNameViaModelsdk(moduleName, oldName, newName)
}

// ---------------------------------------------------------------------------
// RawUnitBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) GetRawUnit(id model.ID) (map[string]any, error) {
	b.initSubBackends()
	return b.rawUnits.GetRawUnit(id)
}

// SetPageParameterRequired sets the IsRequired field of a page parameter.
// Uses the gen-typed page API to ensure BSON field order is preserved.
func (b *MprBackend) SetPageParameterRequired(pageID model.ID, paramName string, required bool) error {
	b.initSubBackends()
	repo := mprrepos.NewPageRepository(b.writer)
	page, err := repo.Get(pageID)
	if err != nil {
		return fmt.Errorf("SetPageParameterRequired: load page: %w", err)
	}
	if page == nil {
		return fmt.Errorf("SetPageParameterRequired: page not found: %s", pageID)
	}
	for _, elem := range page.ParametersItems() {
		pp, ok := elem.(*genPg.PageParameter)
		if !ok || pp == nil {
			continue
		}
		if pp.Name() != paramName {
			continue
		}
		pp.SetIsRequired(required)
		return repo.Update(page)
	}
	return fmt.Errorf("SetPageParameterRequired: parameter %q not found in page %s", paramName, pageID)
}
func (b *MprBackend) GetRawUnitBytes(id model.ID) ([]byte, error) {
	b.initSubBackends()
	return b.rawUnits.GetRawUnitBytes(id)
}
func (b *MprBackend) ListRawUnitsByType(typePrefix string) ([]*types.RawUnit, error) {
	b.initSubBackends()
	return b.rawUnits.ListRawUnitsByType(typePrefix)
}
func (b *MprBackend) ListRawUnits(objectType string) ([]*types.RawUnitInfo, error) {
	b.initSubBackends()
	return b.rawUnits.ListRawUnits(objectType)
}
func (b *MprBackend) GetRawUnitByName(objectType, qualifiedName string) (*types.RawUnitInfo, error) {
	b.initSubBackends()
	return b.rawUnits.GetRawUnitByName(objectType, qualifiedName)
}
func (b *MprBackend) GetRawMicroflowByName(qualifiedName string) ([]byte, error) {
	b.initSubBackends()
	return b.rawUnits.GetRawMicroflowByName(qualifiedName)
}
func (b *MprBackend) UpdateRawUnit(unitID string, contents []byte) error {
	b.initSubBackends()
	if b.rawUnits == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.rawUnits.UpdateRawUnit(unitID, contents)
}

// ListTranslationNodes returns the translatable text fields of a document with
// their per-language translations. Implemented in translation_backend.go.

// ---------------------------------------------------------------------------
// MetadataBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) ListAllUnitIDs() ([]string, error) {
	b.initSubBackends()
	return b.metadata.ListAllUnitIDs()
}
func (b *MprBackend) ListUnits() ([]*types.UnitInfo, error) {
	b.initSubBackends()
	return b.metadata.ListUnits()
}
func (b *MprBackend) ListUnitHashes() (map[string]string, error) {
	b.initSubBackends()
	return b.metadata.ListUnitHashes()
}
func (b *MprBackend) GetUnitTypes() (map[string]int, error) {
	b.initSubBackends()
	return b.metadata.GetUnitTypes()
}
func (b *MprBackend) GetProjectRootID() (string, error) {
	b.initSubBackends()
	return b.metadata.GetProjectRootID()
}
func (b *MprBackend) ContentsDir() string {
	b.initSubBackends()
	return b.metadata.ContentsDir()
}
func (b *MprBackend) InvalidateCache() {
	b.metadata.InvalidateCache()
	b.invalidateSubCaches()
}

func (b *MprBackend) invalidateSubCaches() {
	if b.microflows != nil {
		b.microflows.InvalidateCache()
	}
	if b.pages != nil {
		b.pages.InvalidateCache()
	}
	if b.domainmodels != nil {
		b.domainmodels.InvalidateCache()
	}
	// workflowBackend and securityBackend caches TBD
}

// ---------------------------------------------------------------------------
// WidgetBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) FindCustomWidgetType(widgetID string) (*types.RawCustomWidgetType, error) {
	return convertRawCustomWidgetTypePtr(b.msdkReader.FindCustomWidgetType(widgetID))
}
func (b *MprBackend) FindAllCustomWidgetTypes(widgetID string) ([]*types.RawCustomWidgetType, error) {
	return convertRawCustomWidgetTypeSlice(b.msdkReader.FindAllCustomWidgetTypes(widgetID))
}

// ---------------------------------------------------------------------------
// AgentEditorBackend
// ---------------------------------------------------------------------------

// sdk/agenteditor types are now aliases to types.*, so reader List* methods
// return []*types.* directly — no conversion shim needed.

func (b *MprBackend) ListAgentEditorModels() ([]*types.Model, error) {
	return mprread.ListAgentEditorModels(b.msdkReader)
}
func (b *MprBackend) ListAgentEditorKnowledgeBases() ([]*types.KnowledgeBase, error) {
	return mprread.ListAgentEditorKnowledgeBases(b.msdkReader)
}
func (b *MprBackend) ListAgentEditorConsumedMCPServices() ([]*types.ConsumedMCPService, error) {
	return mprread.ListAgentEditorConsumedMCPServices(b.msdkReader)
}
func (b *MprBackend) ListAgentEditorAgents() ([]*types.Agent, error) {
	return mprread.ListAgentEditorAgents(b.msdkReader)
}
func (b *MprBackend) CreateAgentEditorModel(m *types.Model) error {
	return b.createAgentEditorModelViaModelsdk(m)
}
func (b *MprBackend) UpdateAgentEditorModel(m *types.Model) error {
	return b.updateAgentEditorModelViaModelsdk(m)
}
func (b *MprBackend) DeleteAgentEditorModel(id string) error {
	return b.deleteAgentEditorModelViaModelsdk(id)
}
func (b *MprBackend) CreateAgentEditorKnowledgeBase(k *types.KnowledgeBase) error {
	return b.createAgentEditorKnowledgeBaseViaModelsdk(k)
}
func (b *MprBackend) UpdateAgentEditorKnowledgeBase(k *types.KnowledgeBase) error {
	return b.updateAgentEditorKnowledgeBaseViaModelsdk(k)
}
func (b *MprBackend) DeleteAgentEditorKnowledgeBase(id string) error {
	return b.deleteAgentEditorKnowledgeBaseViaModelsdk(id)
}
func (b *MprBackend) CreateAgentEditorConsumedMCPService(c *types.ConsumedMCPService) error {
	return b.createAgentEditorConsumedMCPServiceViaModelsdk(c)
}
func (b *MprBackend) UpdateAgentEditorConsumedMCPService(c *types.ConsumedMCPService) error {
	return b.updateAgentEditorConsumedMCPServiceViaModelsdk(c)
}
func (b *MprBackend) DeleteAgentEditorConsumedMCPService(id string) error {
	return b.deleteAgentEditorConsumedMCPServiceViaModelsdk(id)
}
func (b *MprBackend) CreateAgentEditorAgent(a *types.Agent) error {
	return b.createAgentEditorAgentViaModelsdk(a)
}
func (b *MprBackend) UpdateAgentEditorAgent(a *types.Agent) error {
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

// SerializeWorkflowActivityGen routes through codec.Encode + bson.Unmarshal
// (Stage 3.3.3.D7). Returns a bson.D that the mutator's raw-bson
// manipulation (serializeAndDedupGen / buildSubFlowBsonGen) can append
// to existing Activity / Outcome / Path / Branch / BoundaryEvent arrays.
//
// BSON byte-identity argument: the same Encoder is used by
// CreateWorkflowGen / UpdateWorkflowGen — so any divergence from
// mpr.SerializeWorkflowActivity here would also break the gen-typed
// CREATE WORKFLOW round-trip caught by D2's unit tests.
func (b *MprBackend) SerializeWorkflowActivityGen(a element.Element) (any, error) {
	if a == nil {
		return nil, fmt.Errorf("SerializeWorkflowActivityGen: nil element")
	}
	enc := b.newEncoder()
	bytes, err := enc.Encode(a)
	if err != nil {
		return nil, fmt.Errorf("encode %s: %w", a.TypeName(), err)
	}
	var doc bson.D
	if err := bson.Unmarshal(bytes, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", a.TypeName(), err)
	}
	return doc, nil
}

// SerializePageGenElement implements backend.WidgetSerializationBackend.
func (b *MprBackend) SerializePageGenElement(elem element.Element) ([]byte, error) {
	if elem == nil {
		return nil, fmt.Errorf("SerializePageGenElement: nil element")
	}
	enc := b.newEncoder()
	return enc.Encode(elem)
}

// newEncoder returns a codec.Encoder configured with the project's Mendix version
// for property-level gating. Properties introduced after the project version
// are skipped when serializing new elements.
// Falls back to a zero-version (no gating) encoder when the reader is not
// yet connected (e.g., in unit tests that exercise serialization in isolation).
func (b *MprBackend) newEncoder() *codec.Encoder {
	if b.msdkReader == nil {
		return &codec.Encoder{}
	}
	pv := b.msdkReader.ProjectVersion()
	if pv == nil {
		return &codec.Encoder{}
	}
	return &codec.Encoder{
		Version: mdlversion.Parse(pv.ProductVersion),
	}
}

// serializeWorkflowActivityGenStandalone encodes a workflow activity element
// to a bson.D using a zero-version (no gating) encoder. Called only from
// mprWorkflowMutator.serializeAndDedupGen when the mutator's backend is nil
// (isolated unit-test contexts). Production paths use SerializeWorkflowActivityGen.
func serializeWorkflowActivityGenStandalone(a element.Element) (any, error) {
	if a == nil {
		return nil, fmt.Errorf("serializeWorkflowActivityGenStandalone: nil element")
	}
	enc := &codec.Encoder{}
	bytes, err := enc.Encode(a)
	if err != nil {
		return nil, fmt.Errorf("encode %s: %w", a.TypeName(), err)
	}
	var doc bson.D
	if err := bson.Unmarshal(bytes, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", a.TypeName(), err)
	}
	return doc, nil
}

// Stage 3.3.4 C1 — gen-typed domain model read/write methods.
// Routes through mprrepos.NewDomainModelRepository which exposes the
// modelsdk-native gen DomainModel; bypasses the legacy sdk parser.

func (b *MprBackend) ListDomainModelsGen() ([]*genDm.DomainModel, error) {
	b.initSubBackends()
	return b.domainmodels.ListDomainModelsGen()
}

func (b *MprBackend) GetDomainModelGen(moduleID model.ID) (*genDm.DomainModel, error) {
	b.initSubBackends()
	return b.domainmodels.GetDomainModelGen(moduleID)
}

func (b *MprBackend) GetDomainModelByIDGen(id model.ID) (*genDm.DomainModel, error) {
	b.initSubBackends()
	return b.domainmodels.GetDomainModelByIDGen(id)
}

func (b *MprBackend) UpdateDomainModelGen(dm *genDm.DomainModel) error {
	if dm == nil {
		return fmt.Errorf("UpdateDomainModelGen: nil DomainModel")
	}
	// Encode via the same codec the repos layer uses, then route through
	// writeUnitContents so ScriptBuffer intercepts the write during EXECUTE SCRIPT.
	contents, err := b.newEncoder().Encode(dm)
	if err != nil {
		return fmt.Errorf("UpdateDomainModelGen: encode: %w", err)
	}
	return b.writeUnitContents(model.ID(dm.ID()), contents)
}

// ---------------------------------------------------------------------------
// Import performance: BufferedUnitStore
// ---------------------------------------------------------------------------

// MprUnitPersistence implements unitstore.UnitPersistence for MprBackend.
type MprUnitPersistence struct {
	b *MprBackend
}

// NewUnitPersistence returns a UnitPersistence backed by this MprBackend.
func (b *MprBackend) NewUnitPersistence() *MprUnitPersistence {
	return &MprUnitPersistence{b: b}
}

// Load reads raw BSON bytes for a single unit. Satisfies unitstore.UnitPersistence.
func (p *MprUnitPersistence) Load(id model.ID) ([]byte, error) {
	data, err := p.b.msdkReader.GetRawUnitBytes(string(id))
	if err != nil {
		return nil, fmt.Errorf("load unit %s: %w", id, err)
	}
	return data, nil
}

// BatchStore writes all units in a single SQLite transaction. Satisfies unitstore.UnitPersistence.
func (p *MprUnitPersistence) BatchStore(units map[model.ID][]byte) error {
	if p.b.msdkWriter == nil {
		return fmt.Errorf("BatchStore: modelsdk writer not initialized")
	}
	wtx, err := p.b.msdkWriter.BeginWriteTransaction()
	if err != nil {
		return fmt.Errorf("BatchStore: begin tx: %w", err)
	}
	for id, data := range units {
		if err := wtx.WriteUnit(string(id), data); err != nil {
			_ = wtx.Rollback()
			return fmt.Errorf("BatchStore: write unit %s: %w", id, err)
		}
	}
	if err := wtx.Commit(); err != nil {
		return fmt.Errorf("BatchStore: commit: %w", err)
	}
	p.b.msdkReader.InvalidateCache()
	return nil
}

// BatchHash computes SHA-256 hex for each unit. Satisfies unitstore.UnitPersistence.
func (p *MprUnitPersistence) BatchHash(units map[model.ID][]byte) (map[model.ID]string, error) {
	out := make(map[model.ID]string, len(units))
	for id, data := range units {
		h := mprUnitSHA256Hex(data)
		out[id] = h
	}
	return out, nil
}

// BeginImportBuffer implements backend.ImportBufferBackend.
func (b *MprBackend) BeginImportBuffer() backend.ImportBuffer {
	return b.EnableImportBuffer()
}

// EnableImportBuffer activates the BufferedUnitStore for an import session.
// All writeUnitContents calls will be buffered in memory.
func (b *MprBackend) EnableImportBuffer() *unitstore.BufferedUnitStore {
	buf := unitstore.New(b.NewUnitPersistence())
	b.unitBuf = buf
	// Wire the gen-type write path (UpdateDomainModelGen → repo.Update →
	// b.writer.WriteUnit → updateUnit) to route through the buffer.
	// The low-level path (writeUnitContents) checks b.unitBuf separately.
	if b.writer != nil {
		b.writer.SetSessionBuf(func(unitID string, data []byte) error {
			if err := buf.Write(model.ID(unitID), data); err != nil {
				return err
			}
			// Set overlay on both readers: b.msdkReader (Reader A, used by
			// GetRawUnitBytes / low-level paths) and b.writer.ConcreteReader() (Reader B,
			// the writer's internal reader used by mprrepos.DomainModelRepository
			// and other gen-type repos). Connect() opens them separately from the
			// same DB, so overlays are not shared between the two instances.
			b.msdkReader.SetOverlay(unitID, data)
			b.writer.ConcreteReader().SetOverlay(unitID, data)
			return nil
		})
	}
	return buf
}

// DisableImportBuffer deactivates the buffer and discards any pending writes.
func (b *MprBackend) DisableImportBuffer() {
	// ClearSessionBuf first: prevents any in-flight write from reaching
	// a buf that is about to be discarded.
	if b.writer != nil {
		b.writer.ClearSessionBuf()
		b.writer.ConcreteReader().ClearAllOverlays()
	}
	if b.unitBuf != nil {
		b.unitBuf.Discard()
		b.unitBuf = nil
		b.msdkReader.ClearAllOverlays()
	}
}

var _ unitstore.UnitPersistence = (*MprUnitPersistence)(nil)

// insertUnit routes to ScriptBuffer when a script is active, otherwise delegates to msdkWriter.InsertUnit.
func (b *MprBackend) insertUnit(unitID, containerID, containmentName, unitType string, contents []byte) error {
	if b.scriptBuf != nil {
		return b.scriptBuf.AddInsert(unitID, containerID, containmentName, unitType, contents)
	}
	return b.msdkWriter.InsertUnit(unitID, containerID, containmentName, unitType, contents)
}

// commitScriptBuffer flushes all buffered writes atomically via BatchWrite.
func (b *MprBackend) commitScriptBuffer() error {
	if b.scriptBuf == nil {
		return fmt.Errorf("commitScriptBuffer: no active script buffer")
	}
	// Clear Writer interceptors before flushing so BatchWrite goes direct.
	if b.writer != nil {
		b.writer.ClearScriptBuf()
	}
	ops := b.scriptBuf.toBatchOps()
	b.scriptBuf = nil
	b.msdkReader.ClearScriptMode()
	if len(ops) == 0 {
		return nil
	}
	return b.msdkWriter.BatchWrite(ops)
}
