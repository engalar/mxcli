// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// ============================================================================
// Page Builder
// ============================================================================

// pageBuilder constructs pages from AST.
type pageBuilder struct {
	moduleLister      backend.ModuleLister
	domainModelReader backend.DomainModelReader
	pageReader        backend.PageReader
	metadataReader    backend.MetadataReader
	folderManager     backend.FolderManager
	connectionManager backend.ConnectionManager
	serializationBackend backend.WidgetSerializationBackend
	moduleID          model.ID
	moduleName        string
	widgetScope       map[string]model.ID                // widget name -> widget ID
	paramScope        map[string]model.ID                // param name -> entity ID
	paramEntityNames  map[string]string                  // param name -> qualified entity name
	execCache         *executorCache                     // Shared cache from executor
	isSnippet         bool                               // True if building a snippet (affects parameter datasource)
	fragments         map[string]*ast.DefineFragmentStmt // Fragment registry from executor
	themeRegistry     *ThemeRegistry                     // Theme design property definitions (may be nil)
	widgetBackend     backend.WidgetBuilderBackend       // Backend for pluggable widget construction

	// Pluggable widget engine (lazily initialized)
	widgetRegistry     *WidgetRegistry
	pluggableEngine    *PluggableWidgetEngine
	pluggableEngineErr error // stores init failure reason for better error messages

	// Per-operation caches (may change during execution)
	layoutsCache    []*genPg.Layout
	pagesCache      []*genPg.Page
	microflowsCache []*genMf.Microflow
	foldersCache    []*types.FolderInfo

	// Microflow / nanoflow / layout / snippet repositories (populated from ctx.*).
	// Optional — when nil the legacy backend.List* fallback is used.
	microflowsRepo repos.MicroflowRepository
	nanoflowsRepo  repos.NanoflowRepository
	layoutsRepo    repos.LayoutRepository
	snippetsRepo   repos.SnippetRepository

	// Entity context for resolving short attribute names inside DataViews
	entityContext string // Qualified entity name (e.g., "Module.Entity")

	// lastContainerID is set by buildPageV3/buildSnippetV3 to the resolved
	// container (folder or module) so cmd_pages_create_v3.go can use it
	// when calling CreatePageGen / CreateSnippetGen.
	lastContainerID model.ID

	mxGraph *mxgraph.Graph // Injected from ExecContext for widget registry fast path
}

// role helpers return the role-specific field (always set when constructed from ExecContext).
func (pb *pageBuilder) moduleListerOrBackend() backend.ModuleLister  { return pb.moduleLister }
func (pb *pageBuilder) dmReaderOrBackend() backend.DomainModelReader { return pb.domainModelReader }
func (pb *pageBuilder) pageReaderOrBackend() backend.PageReader      { return pb.pageReader }
func (pb *pageBuilder) folderMgrOrBackend() backend.FolderManager    { return pb.folderManager }
func (pb *pageBuilder) connMgrOrBackend() backend.ConnectionManager  { return pb.connectionManager }

// initPluggableEngine lazily initializes the pluggable widget engine.
func (pb *pageBuilder) initPluggableEngine() {
	if pb.pluggableEngine != nil || pb.pluggableEngineErr != nil {
		return
	}
	registry, err := NewWidgetRegistry()
	if err != nil {
		pb.pluggableEngineErr = mdlerrors.NewBackend("widget registry init", err)
		log.Printf("warning: %v", pb.pluggableEngineErr)
		return
	}
	if pb.mxGraph != nil {
		registry.SetMxGraph(pb.mxGraph)
	}
	if pb.connMgrOrBackend() != nil {
		if loadErr := registry.LoadUserDefinitions(pb.connMgrOrBackend().Path()); loadErr != nil {
			log.Printf("warning: loading user widget definitions: %v", loadErr)
		}
		projectDir := filepath.Dir(pb.connMgrOrBackend().Path())
		if scanErr := registry.SetProjectDir(projectDir); scanErr != nil {
			log.Printf("warning: widget pre-scan: %v", scanErr)
		}
	}
	pb.widgetRegistry = registry
	pb.pluggableEngine = NewPluggableWidgetEngine(pb.widgetBackend, pb)
}

// registerWidgetName registers a widget name and returns an error if it's already used.
// Widget names must be unique within a page/snippet.

// getProjectPath returns the project directory path from the backend.
func (pb *pageBuilder) getProjectPath() string {
	if pb.connMgrOrBackend() != nil {
		return pb.connMgrOrBackend().Path()
	}
	return ""
}
func (pb *pageBuilder) registerWidgetName(name string, id model.ID) error {
	if name == "" {
		return nil // Anonymous widgets are allowed
	}
	if existingID, exists := pb.widgetScope[name]; exists {
		return mdlerrors.NewAlreadyExistsMsg("widget", name, fmt.Sprintf("duplicate widget name '%s': widget names must be unique within a page (existing ID: %s)", name, existingID))
	}
	pb.widgetScope[name] = id
	return nil
}

// getModules returns cached modules or loads them.
func (pb *pageBuilder) getModules() []*model.Module {
	if pb.execCache != nil && pb.execCache.modules != nil {
		return pb.execCache.modules
	}
	modules, _ := pb.moduleListerOrBackend().ListModules()
	if pb.execCache != nil {
		pb.execCache.modules = modules
	}
	return modules
}

// getHierarchy returns cached hierarchy or creates one.
func (pb *pageBuilder) getHierarchy() (*ContainerHierarchy, error) {
	if pb.execCache != nil && pb.execCache.hierarchy != nil {
		return pb.execCache.hierarchy, nil
	}
	h, err := NewContainerHierarchyFromRoles(
		pb.moduleListerOrBackend(),
		pb.metadataReader,
		pb.folderMgrOrBackend(),
	)
	if err != nil {
		return nil, err
	}
	if pb.execCache != nil {
		pb.execCache.hierarchy = h
	}
	return h, nil
}

// getLayouts returns cached gen layouts or loads them via the backend.
func (pb *pageBuilder) getLayouts() ([]*genPg.Layout, error) {
	if pb.layoutsCache == nil {
		var err error
		pb.layoutsCache, err = pb.pageReaderOrBackend().ListLayoutsGen()
		if err != nil {
			return nil, err
		}
	}
	return pb.layoutsCache, nil
}

// getDomainModelsWithContainer returns gen-typed domain models paired with
// their owning module ID. Caches on execCache.domainModelsWithContainerGen
// (shared with listDomainModelsWithContainerGen). Stage 3.3.4.C7 migration
// from legacy sdk/domainmodel.
func (pb *pageBuilder) getDomainModelsWithContainer() ([]DomainModelGenWithContainer, error) {
	if pb.execCache != nil && pb.execCache.domainModelsWithContainerGen != nil {
		return pb.execCache.domainModelsWithContainerGen, nil
	}
	var dms []*genDm.DomainModel
	if pb.execCache != nil && pb.execCache.domainModelsGen != nil {
		dms = pb.execCache.domainModelsGen
	} else {
		var err error
		dms, err = pb.dmReaderOrBackend().ListDomainModelsGen()
		if err != nil {
			return nil, err
		}
		if pb.execCache != nil {
			pb.execCache.domainModelsGen = dms
		}
	}

	h, err := pb.getHierarchy()
	if err != nil {
		return nil, err
	}

	out := make([]DomainModelGenWithContainer, 0, len(dms))
	for _, dm := range dms {
		if dm == nil {
			continue
		}
		containerID := h.FindModuleID(model.ID(dm.ID()))
		out = append(out, DomainModelGenWithContainer{
			DM:          dm,
			ContainerID: model.ID(containerID),
		})
	}
	if pb.execCache != nil {
		pb.execCache.domainModelsWithContainerGen = out
	}
	return out, nil
}

// getPages returns cached gen pages or loads them via the backend.
func (pb *pageBuilder) getPages() ([]*genPg.Page, error) {
	if pb.pagesCache == nil {
		var err error
		pb.pagesCache, err = pb.pageReaderOrBackend().ListPagesGen()
		if err != nil {
			return nil, err
		}
	}
	return pb.pagesCache, nil
}

// getMicroflows returns cached gen microflows or loads them via the
// modelsdk repo. Stage 3.2.6.5: rewired from
// backend.ListMicroflows (sdk-typed) to ctx.Microflows.ListAll.
// Returns an empty slice when the repo is nil (e.g., in tests that
// don't seed a repo) so callers don't crash.
func (pb *pageBuilder) getMicroflows() ([]*genMf.Microflow, error) {
	if pb.microflowsCache != nil {
		return pb.microflowsCache, nil
	}
	if pb.microflowsRepo == nil {
		return nil, nil
	}
	mfs, err := pb.microflowsRepo.ListAll()
	if err != nil {
		return nil, err
	}
	pb.microflowsCache = mfs
	return pb.microflowsCache, nil
}

// resolveLayout finds a layout by qualified name.
// Gen path (preferred): uses layoutsRepo.FindByQualifiedName when available.
// Legacy fallback: iterates ListLayoutsGen() + hierarchy module resolution.
func (pb *pageBuilder) resolveLayout(layoutName string) (model.ID, error) {
	if pb.layoutsRepo != nil {
		l, err := pb.layoutsRepo.FindByQualifiedName(layoutName)
		if err != nil {
			return "", mdlerrors.NewBackend("find layout "+layoutName, err)
		}
		if l != nil {
			return model.ID(l.ID()), nil
		}
		return "", mdlerrors.NewNotFound("layout", layoutName)
	}

	// Legacy fallback: list + hierarchy module resolution.
	layouts, err := pb.getLayouts()
	if err != nil {
		return "", mdlerrors.NewBackend("list layouts", err)
	}

	h, err := pb.getHierarchy()
	if err != nil {
		return "", mdlerrors.NewBackend("build hierarchy", err)
	}

	parts := strings.Split(layoutName, ".")
	var moduleName, name string
	if len(parts) >= 2 {
		moduleName = parts[0]
		name = parts[len(parts)-1]
	} else {
		name = layoutName
	}

	for _, l := range layouts {
		containerID, _ := pb.pageReaderOrBackend().GetPageContainerUUID(model.ID(l.ID()))
		modID := h.FindModuleID(containerID)
		modName := h.GetModuleName(modID)
		if l.Name() == name && (moduleName == "" || modName == moduleName) {
			return model.ID(l.ID()), nil
		}
	}

	return "", mdlerrors.NewNotFound("layout", layoutName)
}

// resolveEntity finds an entity by qualified name.
func (pb *pageBuilder) resolveEntity(entityRef ast.QualifiedName) (model.ID, error) {
	pairs, err := pb.getDomainModelsWithContainer()
	if err != nil {
		return "", mdlerrors.NewBackend("list domain models", err)
	}

	h, err := pb.getHierarchy()
	if err != nil {
		return "", mdlerrors.NewBackend("build hierarchy", err)
	}

	for _, pair := range pairs {
		modName := h.GetModuleName(pair.ContainerID)
		for _, elem := range pair.DM.EntitiesItems() {
			e, ok := elem.(*genDm.Entity)
			if !ok {
				continue
			}
			if e.Name() == entityRef.Name && (entityRef.Module == "" || modName == entityRef.Module) {
				return model.ID(e.ID()), nil
			}
		}
	}

	return "", mdlerrors.NewNotFound("entity", entityRef.String())
}

// getModuleID returns the module ID for any container by using the hierarchy.
// Deprecated: prefer using getHierarchy().FindModuleID() directly.
func getModuleID(ctx *ExecContext, containerID model.ID) model.ID {
	h, err := getHierarchy(ctx)
	if err != nil {
		return containerID
	}
	return h.FindModuleID(containerID)
}

// getModuleName returns the module name for a module ID.
// Deprecated: prefer using getHierarchy().GetModuleName() directly.
func getModuleName(ctx *ExecContext, moduleID model.ID) string {
	h, err := getHierarchy(ctx)
	if err != nil {
		return ""
	}
	return h.GetModuleName(moduleID)
}

// getMainPlaceholderRef returns the qualified name reference for the main placeholder.
// The format is "Module.Layout.Main" (e.g., "Atlas_Core.Atlas_TopBar.Main").
func (pb *pageBuilder) getMainPlaceholderRef(layoutName string) string {
	// Standard convention: the main placeholder is named "Main"
	// So the reference is "LayoutQualifiedName.Main"
	if layoutName == "" {
		return ""
	}
	return layoutName + ".Main"
}

// getFolders returns cached folders or loads them.
func (pb *pageBuilder) getFolders() ([]*types.FolderInfo, error) {
	if pb.foldersCache == nil {
		var err error
		pb.foldersCache, err = pb.folderMgrOrBackend().ListFolders()
		if err != nil {
			return nil, err
		}
	}
	return pb.foldersCache, nil
}

// resolveFolder resolves a folder path (e.g., "Resources/Images") to a folder ID.
// The path is relative to the current module. If the folder doesn't exist, it creates it.
func (pb *pageBuilder) resolveFolder(folderPath string) (model.ID, error) {
	if folderPath == "" {
		return pb.moduleID, nil
	}

	folders, err := pb.getFolders()
	if err != nil {
		return "", mdlerrors.NewBackend("list folders", err)
	}

	// Split path into parts
	parts := strings.Split(folderPath, "/")
	currentContainerID := pb.moduleID

	for _, part := range parts {
		if part == "" {
			continue
		}

		// Find folder with this name under current container
		var foundFolder *types.FolderInfo
		for _, f := range folders {
			if f.ContainerID == currentContainerID && f.Name == part {
				foundFolder = f
				break
			}
		}

		if foundFolder != nil {
			currentContainerID = foundFolder.ID
		} else {
			// Create the folder
			newFolderID, err := pb.createFolder(part, currentContainerID)
			if err != nil {
				return "", mdlerrors.NewBackend(fmt.Sprintf("create folder %s", part), err)
			}
			parentContainerID := currentContainerID
			currentContainerID = newFolderID

			// Add to cache
			pb.foldersCache = append(pb.foldersCache, &types.FolderInfo{
				ID:          newFolderID,
				ContainerID: parentContainerID,
				Name:        part,
			})
			currentContainerID = newFolderID
		}
	}

	return currentContainerID, nil
}

// createFolder creates a new folder in the project.
func (pb *pageBuilder) createFolder(name string, containerID model.ID) (model.ID, error) {
	folder := &model.Folder{
		BaseElement: model.BaseElement{
			ID:       model.ID(types.GenerateID()),
			TypeName: "Projects$Folder",
		},
		ContainerID: containerID,
		Name:        name,
	}

	if err := pb.folderMgrOrBackend().CreateFolder(folder); err != nil {
		return "", err
	}

	return folder.ID, nil
}

// execDropPage handles DROP PAGE statement.
func execDropPage(ctx *ExecContext, s *ast.DropPageStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	pairs, err := listPagesWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list pages", err)
	}

	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(model.ID(p.ContainerID))
		modName := h.GetModuleName(modID)
		if modName == s.Name.Module && p.Elem.Name() == s.Name.Name {
			pgID := model.ID(p.Elem.ID())
			if err := ctx.PageWriter.DeletePageGen(pgID); err != nil {
				return mdlerrors.NewBackend("delete page", err)
			}
			invalidatePagesGenCache(ctx)
			fmt.Fprintf(ctx.Output, "Dropped page %s\n", s.Name.String())
			return nil
		}
	}

	return mdlerrors.NewNotFound("page", s.Name.String())
}

// execDropSnippet handles DROP SNIPPET statement.
func execDropSnippet(ctx *ExecContext, s *ast.DropSnippetStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	pairs, err := listSnippetsWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list snippets", err)
	}

	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(model.ID(p.ContainerID))
		modName := h.GetModuleName(modID)
		if modName == s.Name.Module && p.Elem.Name() == s.Name.Name {
			snpID := model.ID(p.Elem.ID())
			if err := ctx.PageWriter.DeleteSnippetGen(snpID); err != nil {
				return mdlerrors.NewBackend("delete snippet", err)
			}
			invalidatePagesGenCache(ctx)
			fmt.Fprintf(ctx.Output, "Dropped snippet %s\n", s.Name.String())
			return nil
		}
	}

	return mdlerrors.NewNotFound("snippet", s.Name.String())
}

func (e *Executor) getModuleName(moduleID model.ID) string {
	return getModuleName(e.newExecContext(context.Background()), moduleID)
}
