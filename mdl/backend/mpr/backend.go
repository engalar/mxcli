// SPDX-License-Identifier: Apache-2.0

// Package mprbackend provides the MprBackend implementation of
// backend.FullBackend. The package name is "mprbackend" (not "mpr") to
// avoid collision with the sdk/mpr package in import blocks.
package mprbackend

import (
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/mdl/backend"
	mprrepos "github.com/mendixlabs/mxcli/mdl/backend/mpr/repos"
	"github.com/mendixlabs/mxcli/mdl/backend/unitstore"
	"github.com/mendixlabs/mxcli/mdl/linter"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genBE "github.com/mendixlabs/mxcli/modelsdk/gen/businessevents"
	genDBC "github.com/mendixlabs/mxcli/modelsdk/gen/databaseconnector"
	genDTrans "github.com/mendixlabs/mxcli/modelsdk/gen/datatransformers"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	genJSA "github.com/mendixlabs/mxcli/modelsdk/gen/javascriptactions"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	genProj "github.com/mendixlabs/mxcli/modelsdk/gen/projects"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
	"github.com/mendixlabs/mxcli/modelsdk/mprread"
	mdlversion "github.com/mendixlabs/mxcli/modelsdk/version"
)

var _ backend.FullBackend = (*MprBackend)(nil)
var _ backend.PageModelBackend = (*MprBackend)(nil)
var _ backend.ImportBufferBackend = (*MprBackend)(nil)
var _ linter.LintReader = (*MprBackend)(nil)

// MprBackend implements backend.FullBackend by delegating to a single
// modelsdk/mpr Reader/Writer pair. Read methods either route through
// mprread free functions (which take a *modelsdkmpr.Reader) or through
// the hand-decoded *_compat.go helpers that pull raw bytes from the same
// Reader. Write methods route through the modelsdkmpr.UnitWriter / the
// gen-typed repositories.
//
// All read/write methods assume Connect() has been called; calling them
// before Connect() panics with a nil pointer dereference. The executor
// enforces connection state via ConnectionBackend.IsConnected().
type MprBackend struct {
	reader     *modelsdkmpr.Reader
	msdkReader *modelsdkmpr.Reader // alias of reader; kept for *_compat.go ergonomics
	msdkWriter modelsdkmpr.UnitWriter
	path       string
	// scriptBuf is non-nil while an EXECUTE SCRIPT block is open. Write helpers
	// route mutations into the buffer instead of opening per-statement SQL
	// transactions; the whole script commits atomically via a single BatchWrite
	// at the end — see backend.ScriptTransaction.
	scriptBuf *ScriptBuffer
	// unitBuf is non-nil when an ImportSession is active.
	// writeUnitContents routes writes through the buffer instead of opening
	// individual SQLite transactions. Reads are satisfied from the overlay.
	unitBuf *unitstore.BufferedUnitStore
	// widgetTypeCache is non-nil during a page build (BeginPageBuild..EndPageBuild).
	// It maps widgetID → widgetTypeCacheEntry, enabling one CustomWidgets$CustomWidgetType
	// schema to be shared across all instances of the same widget type on a page.
	// This matches Studio Pro's canonical format and reduces page BSON size by up to 50%
	// on pages with multiple same-type filter widgets.
	widgetTypeCache map[string]*widgetTypeCacheEntry

	// Domain-specific sub-backends. Populated lazily after Connect().
	// These extract method groups into focused types as part of the
	// MprBackend facade decomposition (Phase 3).
	modules          *moduleBackend
	microflows       *microflowBackend
	workflows        *workflowBackend
	pages            *pageBackend
	java             *javaBackend
	domainmodels     *domainModelBackend
	security         *securityBackend
	folders          *folderBackend
	scheduledEvents  *scheduledEventBackend
	enumerations     *enumerationBackend
	constants        *constantBackend
	rawUnits         *rawUnitBackend
	metadata         *metadataBackend
	mappings         *mappingBackend
}

// widgetTypeCacheEntry holds the per-page cached type schema for one widget type.
type widgetTypeCacheEntry struct {
	rawType     bson.D                               // shared CustomWidgets$CustomWidgetType
	rawObject   bson.D                               // template object; deep-cloned per instance
	propTypeIDs map[string]types.PropertyTypeIDEntry // for property-key → TypePointer lookups
}

// BeginPageBuild initialises the per-page widget-type cache.  Call before building
// any widget BSON for a page, and call EndPageBuild when the page is complete.
func (b *MprBackend) BeginPageBuild() {
	b.widgetTypeCache = make(map[string]*widgetTypeCacheEntry)
}

// EndPageBuild clears the per-page widget-type cache so type-schema $IDs do not
// accidentally leak across different pages.
func (b *MprBackend) EndPageBuild() {
	b.widgetTypeCache = nil
}

// getWidgetTypeCacheEntry returns the cached type entry for widgetID, or nil.
func (b *MprBackend) getWidgetTypeCacheEntry(widgetID string) *widgetTypeCacheEntry {
	if b.widgetTypeCache == nil {
		return nil
	}
	return b.widgetTypeCache[widgetID]
}

// setWidgetTypeCacheEntry stores an entry in the per-page cache (no-op when cache is nil).
func (b *MprBackend) setWidgetTypeCacheEntry(widgetID string, entry *widgetTypeCacheEntry) {
	if b.widgetTypeCache != nil {
		b.widgetTypeCache[widgetID] = entry
	}
}

// New creates a new unconnected MprBackend. Call Connect(path) to open a project.
func New() *MprBackend {
	return &MprBackend{}
}

// NewFromPath opens path for read-write and returns a fully-wired MprBackend.
// Equivalent to Connect on a zero-value backend; useful in tests.
func NewFromPath(path string) (*MprBackend, error) {
	b := &MprBackend{}
	if err := b.Connect(path); err != nil {
		return nil, err
	}
	return b, nil
}

// ---------------------------------------------------------------------------
// ConnectionBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) Connect(path string) error {
	r, err := modelsdkmpr.OpenWithOptions(path, modelsdkmpr.OpenOptions{ReadOnly: false})
	if err != nil {
		return err
	}
	// Reuse r as the writer's internal reader so that w.reader.InvalidateCache()
	// (called by insertUnit after every write) invalidates the same Reader that
	// b.msdkReader points to. With separate Reader objects (old behaviour), writes
	// only invalidated the Writer's private cache while listing methods on b.msdkReader
	// returned stale data from their own independent unitCacheValid flag.
	mw := modelsdkmpr.NewWriterWithReader(r)
	b.reader = r
	b.msdkReader = r
	b.msdkWriter = mw
	b.path = path
	b.modules = newModuleBackend(r)
	b.microflows = newMicroflowBackend(mw)
	b.workflows = newWorkflowBackend(mw)
	b.pages = newPageBackend(mw)
	b.java = newJavaBackend(mw)
	b.domainmodels = newDomainModelBackend(mw)
	b.security = newSecurityBackend(mw)
	b.folders = newFolderBackend(r)
	b.scheduledEvents = newScheduledEventBackend(r)
	b.enumerations = newEnumerationBackend(r)
	b.constants = newConstantBackend(r)
	b.rawUnits = newRawUnitBackend(r, mw)
	b.metadata = newMetadataBackend(r)
	b.mappings = newMappingBackend(r)
	return nil
}

// initSubBackends lazily initialises domain-specific sub-backends.
// Safe to call multiple times.
func (b *MprBackend) initSubBackends() {
	if b.reader != nil {
		if b.modules == nil {
			b.modules = newModuleBackend(b.reader)
		}
		if b.folders == nil {
			b.folders = newFolderBackend(b.reader)
		}
		if b.scheduledEvents == nil {
			b.scheduledEvents = newScheduledEventBackend(b.reader)
		}
		if b.enumerations == nil {
			b.enumerations = newEnumerationBackend(b.reader)
		}
		if b.constants == nil {
			b.constants = newConstantBackend(b.reader)
		}
		if b.rawUnits == nil && b.msdkWriter != nil {
			b.rawUnits = newRawUnitBackend(b.reader, b.msdkWriter)
		}
		if b.metadata == nil {
			b.metadata = newMetadataBackend(b.reader)
		}
		if b.mappings == nil {
			b.mappings = newMappingBackend(b.reader)
		}
		if b.microflows == nil && b.msdkWriter != nil {
			if w, ok := b.msdkWriter.(*modelsdkmpr.Writer); ok {
				b.microflows = newMicroflowBackend(w)
			}
		}
		if b.workflows == nil && b.msdkWriter != nil {
			if w, ok := b.msdkWriter.(*modelsdkmpr.Writer); ok {
				b.workflows = newWorkflowBackend(w)
			}
		}
		if b.pages == nil && b.msdkWriter != nil {
			if w, ok := b.msdkWriter.(*modelsdkmpr.Writer); ok {
				b.pages = newPageBackend(w)
			}
		}
		if b.java == nil && b.msdkWriter != nil {
			if w, ok := b.msdkWriter.(*modelsdkmpr.Writer); ok {
				b.java = newJavaBackend(w)
			}
		}
		if b.domainmodels == nil && b.msdkWriter != nil {
			if w, ok := b.msdkWriter.(*modelsdkmpr.Writer); ok {
				b.domainmodels = newDomainModelBackend(w)
			}
		}
		if b.security == nil && b.msdkWriter != nil {
			if w, ok := b.msdkWriter.(*modelsdkmpr.Writer); ok {
				b.security = newSecurityBackend(w)
			}
		}
	}
}

func (b *MprBackend) Disconnect() error {
	if b.reader == nil {
		return nil
	}
	err := b.reader.Close()
	b.reader = nil
	b.msdkReader = nil
	b.msdkWriter = nil
	b.path = ""
	return err
}

func (b *MprBackend) IsConnected() bool { return b.reader != nil }
func (b *MprBackend) Path() string      { return b.path }

// EnableContentCache activates in-memory caching of mxunit file contents.
// Call once after Connect when the backend will be held persistently across
// multiple requests (per-MPR daemon mode). The cache is cleared on any write.
func (b *MprBackend) EnableContentCache() {
	if b.reader != nil {
		b.reader.EnableContentCache()
	}
}

func (b *MprBackend) Version() types.MPRVersion { return types.MPRVersion(b.msdkReader.Version()) }
func (b *MprBackend) ProjectVersion() *types.ProjectVersion {
	return convertProjectVersionFromMsdk(b.msdkReader.ProjectVersion())
}
func (b *MprBackend) GetMendixVersion() (string, error) { return b.msdkReader.GetMendixVersion() }

// Commit is a no-op — the MPR writer auto-commits on each write operation.
func (b *MprBackend) Commit() error { return nil }

// ---------------------------------------------------------------------------
// ModuleBackend — reads delegate to moduleBackend, writes stay on MprBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) ListModules() ([]*model.Module, error) {
	b.initSubBackends()
	return b.modules.ListModules()
}
func (b *MprBackend) GetModule(id model.ID) (*model.Module, error) {
	b.initSubBackends()
	return b.modules.GetModule(id)
}
func (b *MprBackend) GetModuleByName(name string) (*model.Module, error) {
	b.initSubBackends()
	return b.modules.GetModuleByName(name)
}
func (b *MprBackend) CreateModule(module *model.Module) error {
	return b.createModuleViaModelsdk(module)
}
func (b *MprBackend) UpdateModule(module *model.Module) error {
	return b.updateModuleViaModelsdk(module)
}
func (b *MprBackend) DeleteModule(id model.ID) error { return b.deleteModuleViaModelsdk(id) }
func (b *MprBackend) DeleteModuleWithCleanup(id model.ID, moduleName string) error {
	return b.deleteModuleWithCleanupViaModelsdk(id, moduleName)
}

// ---------------------------------------------------------------------------
// ModuleSettingsBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) ListModuleSettings() ([]*types.ModuleSettings, error) {
	units, err := mprread.ListUnitsWithContainer[*genProj.ModuleSettings](b.msdkReader)
	if err != nil {
		return nil, err
	}
	return moduleSettingsUnitsToTypes(units), nil
}
func (b *MprBackend) GetModuleSettings(moduleID model.ID) (*types.ModuleSettings, error) {
	all, err := b.ListModuleSettings()
	if err != nil {
		return nil, err
	}
	for _, ms := range all {
		if ms.ContainerID == moduleID {
			return ms, nil
		}
	}
	return nil, nil
}
func (b *MprBackend) UpdateModuleSettings(ms *types.ModuleSettings) error {
	return b.updateModuleSettingsViaModelsdk(ms)
}

// ---------------------------------------------------------------------------
// FolderBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) ListFolders() ([]*types.FolderInfo, error) {
	b.initSubBackends()
	return b.folders.ListFolders()
}
func (b *MprBackend) CreateFolder(folder *model.Folder) error {
	return b.createFolderViaModelsdk(folder)
}
func (b *MprBackend) DeleteFolder(id model.ID) error { return b.deleteFolderViaModelsdk(id) }
func (b *MprBackend) MoveFolder(id model.ID, newContainerID model.ID) error {
	return b.moveFolderViaModelsdk(id, newContainerID)
}

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
	w, ok := b.concreteWriter()
	if !ok {
		return fmt.Errorf("CreatePageGen: no modelsdk writer")
	}
	return mprrepos.NewPageRepository(w).Create(parentUUID, containmentName, page)
}

func (b *MprBackend) UpdatePageGen(page *genPg.Page) error {
	if page == nil {
		return fmt.Errorf("UpdatePageGen: nil Page")
	}
	w, ok := b.concreteWriter()
	if !ok {
		return fmt.Errorf("UpdatePageGen: no modelsdk writer")
	}
	return mprrepos.NewPageRepository(w).Update(page)
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
	w, ok := b.concreteWriter()
	if !ok {
		return fmt.Errorf("CreateLayoutGen: no modelsdk writer")
	}
	return mprrepos.NewLayoutRepository(w).Create(parentUUID, containmentName, layout)
}

func (b *MprBackend) UpdateLayoutGen(layout *genPg.Layout) error {
	if layout == nil {
		return fmt.Errorf("UpdateLayoutGen: nil Layout")
	}
	w, ok := b.concreteWriter()
	if !ok {
		return fmt.Errorf("UpdateLayoutGen: no modelsdk writer")
	}
	return mprrepos.NewLayoutRepository(w).Update(layout)
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
	w, ok := b.concreteWriter()
	if !ok {
		return fmt.Errorf("CreateSnippetGen: no modelsdk writer")
	}
	return mprrepos.NewSnippetRepository(w).Create(parentUUID, containmentName, snippet)
}

func (b *MprBackend) UpdateSnippetGen(snippet *genPg.Snippet) error {
	if snippet == nil {
		return fmt.Errorf("UpdateSnippetGen: nil Snippet")
	}
	w, ok := b.concreteWriter()
	if !ok {
		return fmt.Errorf("UpdateSnippetGen: no modelsdk writer")
	}
	return mprrepos.NewSnippetRepository(w).Update(snippet)
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

// GetContainerID resolves moduleID + optional folder path to a container UUID.
// If folder is empty, returns moduleID directly (root of module). Path segments
// are separated by "/"; folders that do not yet exist are created.
func (b *MprBackend) GetContainerID(moduleID model.ID, folder string) (model.ID, error) {
	if folder == "" {
		return moduleID, nil
	}

	folders, err := b.ListFolders()
	if err != nil {
		return "", err
	}

	current := moduleID
	for _, part := range strings.Split(folder, "/") {
		if part == "" {
			continue
		}

		var found *types.FolderInfo
		for _, f := range folders {
			if f.ContainerID == current && f.Name == part {
				found = f
				break
			}
		}

		if found != nil {
			current = found.ID
			continue
		}

		newFolder := &model.Folder{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Projects$Folder",
			},
			ContainerID: current,
			Name:        part,
		}
		if err := b.CreateFolder(newFolder); err != nil {
			return "", fmt.Errorf("create folder %q: %w", part, err)
		}
		folders = append(folders, &types.FolderInfo{
			ID:          newFolder.ID,
			ContainerID: current,
			Name:        part,
		})
		current = newFolder.ID
	}

	return current, nil
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
// SecurityBackend (ProjectSecurityBackend + ModuleSecurityBackend + EntityAccessBackend)
// ---------------------------------------------------------------------------

func (b *MprBackend) GetProjectSecurityGen() (*genSec.ProjectSecurity, error) {
	b.initSubBackends()
	return b.security.GetProjectSecurityGen()
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
func (b *MprBackend) SetPasswordPolicy(unitID model.ID, minLength *int32, requireDigit, requireMixedCase, requireSymbol *bool) error {
	return b.setPasswordPolicyViaModelsdk(unitID, minLength, requireDigit, requireMixedCase, requireSymbol)
}

func (b *MprBackend) GetModuleSecurityGen(moduleID model.ID) (*genSec.ModuleSecurity, error) {
	b.initSubBackends()
	return b.security.GetModuleSecurityGen(moduleID)
}
func (b *MprBackend) ListModuleSecurityGen() ([]*genSec.ModuleSecurity, error) {
	b.initSubBackends()
	return b.security.ListModuleSecurityGen()
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
	return b.addEntityAccessRuleViaModelsdk(params.UnitID, params.EntityName, params.RoleNames, params.AllowCreate, params.AllowDelete, params.DefaultMemberAccess, params.XPathConstraint, params.MemberAccesses)
}
func (b *MprBackend) RemoveEntityAccessRule(unitID model.ID, entityName string, roleNames []string) (int, error) {
	return b.removeEntityAccessRuleViaModelsdk(unitID, entityName, roleNames)
}
func (b *MprBackend) RevokeEntityMemberAccess(unitID model.ID, entityName string, roleNames []string, revocation types.EntityAccessRevocation) (int, error) {
	return b.revokeEntityMemberAccessViaModelsdk(unitID, entityName, roleNames, revocation)
}
func (b *MprBackend) RemoveRoleFromAllEntities(unitID model.ID, roleName string) (int, error) {
	return b.removeRoleFromAllEntitiesViaModelsdk(unitID, roleName)
}
func (b *MprBackend) ReconcileMemberAccesses(unitID model.ID, moduleName string) ([]string, error) {
	changes, err := b.reconcileMemberAccessesViaModelsdk(unitID, moduleName)
	if err != nil {
		return nil, err
	}
	msgs := make([]string, len(changes))
	for i, ch := range changes {
		msgs[i] = ch.String()
	}
	return msgs, nil
}

// ---------------------------------------------------------------------------
// NavigationBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) ListNavigationDocuments() ([]*types.NavigationDocument, error) {
	return b.listNavigationDocumentsFromRaw()
}
func (b *MprBackend) GetNavigation() (*types.NavigationDocument, error) {
	return b.getNavigationFromRaw()
}
func (b *MprBackend) UpdateNavigationProfile(navDocID model.ID, profileName string, spec types.NavigationProfileSpec) error {
	return b.updateNavigationProfileViaModelsdk(navDocID, profileName, spec)
}

// ---------------------------------------------------------------------------
// ServiceBackend (OData + REST + BusinessEvent + DatabaseConnection + DataTransformer)
// ---------------------------------------------------------------------------

func (b *MprBackend) ListConsumedODataServices() ([]*model.ConsumedODataService, error) {
	// TODO(phase4): migrate to mprread. ConsumedODataService has 28 scalar fields plus
	// HttpConfiguration + HttpHeaderEntry nested parts; full converter ~80 lines.
	// gen package: modelsdk/gen/rest (Rest$ConsumedODataService).
	return b.listConsumedODataServicesFromRaw()
}
func (b *MprBackend) ListPublishedODataServices() ([]*model.PublishedODataService, error) {
	// TODO(phase4): migrate to mprread. PublishedODataService has nested EntityTypes
	// (with Members) + EntitySets + AuthenticationTypes + AllowedModuleRoles.
	// gen package: modelsdk/gen/odatapublish (ODataPublish$PublishedODataService2).
	return b.listPublishedODataServicesFromRaw()
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
	// TODO(phase4): migrate to mprread. Same shape class as ConsumedODataService —
	// scalar transport config + Operations nested list.
	// gen package: modelsdk/gen/rest (Rest$ConsumedRestService).
	return b.listConsumedRestServicesFromRaw()
}
func (b *MprBackend) ListPublishedRestServices() ([]*model.PublishedRestService, error) {
	// TODO(phase4): migrate to mprread. Has Resources → Operations → PathParams tree.
	// gen package: modelsdk/gen/rest (Rest$PublishedRestService).
	return b.listPublishedRestServicesFromRaw()
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
	units, err := mprread.ListUnitsWithContainer[*genBE.BusinessEventService](b.msdkReader)
	if err != nil {
		return nil, err
	}
	return businessEventServiceUnitsToModel(units), nil
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
	units, err := mprread.ListUnitsWithContainer[*genDBC.DatabaseConnection](b.msdkReader)
	if err != nil {
		return nil, err
	}
	return databaseConnectionUnitsToModel(units), nil
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
	units, err := mprread.ListUnitsWithContainer[*genDTrans.DataTransformer](b.msdkReader)
	if err != nil {
		return nil, err
	}
	return dataTransformerUnitsToModel(units), nil
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
	b.initSubBackends()
	return b.mappings.ListImportMappings()
}
func (b *MprBackend) GetImportMappingByQualifiedName(moduleName, name string) (*model.ImportMapping, error) {
	b.initSubBackends()
	return b.mappings.GetImportMappingByQualifiedName(moduleName, name)
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
	b.initSubBackends()
	return b.mappings.ListExportMappings()
}
func (b *MprBackend) GetExportMappingByQualifiedName(moduleName, name string) (*model.ExportMapping, error) {
	b.initSubBackends()
	return b.mappings.GetExportMappingByQualifiedName(moduleName, name)
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
	b.initSubBackends()
	return b.mappings.ListJsonStructures()
}
func (b *MprBackend) GetJsonStructureByQualifiedName(moduleName, name string) (*types.JsonStructure, error) {
	all, err := b.ListJsonStructures()
	if err != nil {
		return nil, err
	}
	// js.ContainerID may be a folder (MPR v2) rather than the module itself,
	// so walk the container hierarchy via the sdk/mpr helper to resolve the
	// enclosing module name. TODO(phase4): port buildContainerModuleNameMap
	// to use mprread once Folder lister exists there.
	containerToModule, err := b.buildContainerModuleNameMapViaSdk()
	if err != nil {
		return nil, err
	}
	for _, js := range all {
		if js.Name == name && containerToModule[js.ContainerID] == moduleName {
			return js, nil
		}
	}
	return nil, fmt.Errorf("json structure %s.%s not found", moduleName, name)
}

// buildContainerModuleNameMapViaSdk delegates to the legacy sdk/mpr reader's
// hierarchy walker. Used by GetXxxByQualifiedName methods until the helper
// is ported to mprread.
func (b *MprBackend) buildContainerModuleNameMapViaSdk() (map[model.ID]string, error) {
	modules, err := b.ListModules()
	if err != nil {
		return nil, err
	}
	moduleNames := make(map[model.ID]string, len(modules))
	for _, m := range modules {
		moduleNames[m.ID] = m.Name
	}
	units, err := b.ListUnits()
	if err != nil {
		return nil, err
	}
	parentOf := make(map[model.ID]model.ID, len(units))
	for _, u := range units {
		parentOf[u.ID] = u.ContainerID
	}
	result := make(map[model.ID]string)
	var find func(id model.ID) string
	find = func(id model.ID) string {
		if cached, ok := result[id]; ok {
			return cached
		}
		if name, ok := moduleNames[id]; ok {
			result[id] = name
			return name
		}
		parent, ok := parentOf[id]
		if !ok || parent == id || parent == "" {
			result[id] = ""
			return ""
		}
		name := find(parent)
		result[id] = name
		return name
	}
	for id := range parentOf {
		find(id)
	}
	return result, nil
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
	w, ok := b.concreteWriter()
	if !ok {
		return fmt.Errorf("CreateJavaActionGen: no modelsdk writer")
	}
	return mprrepos.NewJavaActionRepository(w).Create(parentUUID, containmentName, ja)
}

// UpdateJavaActionGen mirrors CreateJavaActionGen — gen-native repo Update.
func (b *MprBackend) UpdateJavaActionGen(ja *genJA.JavaAction) error {
	if ja == nil {
		return fmt.Errorf("UpdateJavaActionGen: nil JavaAction")
	}
	w, ok := b.concreteWriter()
	if !ok {
		return fmt.Errorf("UpdateJavaActionGen: no modelsdk writer")
	}
	return mprrepos.NewJavaActionRepository(w).Update(ja)
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

func (b *MprBackend) UpdateJavaScriptActionGen(jsa *genJSA.JavaScriptAction) error {
	if jsa == nil {
		return fmt.Errorf("UpdateJavaScriptActionGen: nil JavaScriptAction")
	}
	w, ok := b.concreteWriter()
	if !ok {
		return fmt.Errorf("UpdateJavaScriptActionGen: no modelsdk writer")
	}
	return mprrepos.NewJavaScriptActionRepository(w).Update(jsa)
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
	w, ok := b.concreteWriter()
	if !ok {
		return fmt.Errorf("CreateWorkflowGen: no modelsdk writer")
	}
	return mprrepos.NewWorkflowRepository(w).Create(parentUUID, containmentName, wf)
}

func (b *MprBackend) UpdateWorkflowGen(wf *genWf.Workflow) error {
	if wf == nil {
		return fmt.Errorf("UpdateWorkflowGen: nil Workflow")
	}
	w, ok := b.concreteWriter()
	if !ok {
		return fmt.Errorf("UpdateWorkflowGen: no modelsdk writer")
	}
	return mprrepos.NewWorkflowRepository(w).Update(wf)
}

// ---------------------------------------------------------------------------
// SettingsBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) GetProjectSettings() (*model.ProjectSettings, error) {
	// TODO(phase4): migrate to mprread. Settings$ProjectSettings holds a polymorphic
	// Settings array (10+ Part subtypes: WebUI, Integration, Configuration, Model,
	// Convention, Language, Certificate, Workflows, JarDeployment, Distribution).
	// model.ProjectSettings.RawParts []map[string]any is critical for round-trip
	// fidelity of unrecognized part types — gen-typed read drops this.
	// gen package: modelsdk/gen/settings (Settings$ProjectSettings).
	return b.getProjectSettingsFromRaw()
}
func (b *MprBackend) UpdateProjectSettings(ps *model.ProjectSettings) error {
	return b.updateProjectSettingsViaModelsdk(ps)
}

// ---------------------------------------------------------------------------
// ImageBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) ListImageCollections() ([]*types.ImageCollection, error) {
	return b.listImageCollectionsFromRaw()
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
func (b *MprBackend) MoveImageCollection(ic *types.ImageCollection) error {
	return b.moveImageCollectionViaModelsdk(ic)
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
	b.initSubBackends()
	b.metadata.InvalidateCache()
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
	// concreteWriter().WriteUnit → updateUnit) to route through the buffer.
	// The low-level path (writeUnitContents) checks b.unitBuf separately.
	if w, ok := b.concreteWriter(); ok {
		w.SetSessionBuf(func(unitID string, data []byte) error {
			if err := buf.Write(model.ID(unitID), data); err != nil {
				return err
			}
			// Set overlay on both readers: b.msdkReader (Reader A, used by
			// GetRawUnitBytes / low-level paths) and w.ConcreteReader() (Reader B,
			// the writer's internal reader used by mprrepos.DomainModelRepository
			// and other gen-type repos). Connect() opens them separately from the
			// same DB, so overlays are not shared between the two instances.
			b.msdkReader.SetOverlay(unitID, data)
			w.ConcreteReader().SetOverlay(unitID, data)
			return nil
		})
	}
	return buf
}

// DisableImportBuffer deactivates the buffer and discards any pending writes.
func (b *MprBackend) DisableImportBuffer() {
	// ClearSessionBuf first: prevents any in-flight write from reaching
	// a buf that is about to be discarded.
	if w, ok := b.concreteWriter(); ok {
		w.ClearSessionBuf()
		w.ConcreteReader().ClearAllOverlays()
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
	ops := b.scriptBuf.toBatchOps()
	b.scriptBuf = nil
	b.msdkReader.ClearScriptMode()
	if len(ops) == 0 {
		return nil
	}
	return b.msdkWriter.BatchWrite(ops)
}
