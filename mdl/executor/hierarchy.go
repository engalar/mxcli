// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"strings"

	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/meta"
)

// hierarchySource is the minimal interface needed to build a ContainerHierarchy.
// Implementations include *mpr.Reader, backend.FullBackend, and hierarchyRolesSource.
type hierarchySource interface {
	ListModules() ([]*model.Module, error)
	ListUnits() ([]*types.UnitInfo, error)
	ListFolders() ([]*types.FolderInfo, error)
}

// hierarchyRolesSource wraps role-specific backend interfaces into a hierarchySource.
type hierarchyRolesSource struct {
	ml  backend.ModuleLister
	mur backend.MetadataReader
	fm  backend.FolderManager
}

func (s hierarchyRolesSource) ListModules() ([]*model.Module, error) {
	return s.ml.ListModules()
}

func (s hierarchyRolesSource) ListUnits() ([]*types.UnitInfo, error) {
	return s.mur.ListUnits()
}

func (s hierarchyRolesSource) ListFolders() ([]*types.FolderInfo, error) {
	return s.fm.ListFolders()
}

// ContainerHierarchy provides efficient module and folder resolution for documents.
// It caches the container hierarchy to avoid repeated lookups.
type ContainerHierarchy struct {
	moduleIDs       map[model.ID]bool
	moduleNames     map[model.ID]string
	containerParent map[model.ID]model.ID
	folderNames     map[model.ID]string
}

// NewContainerHierarchy creates a new hierarchy from any source that provides
// modules, units, and folders.
func NewContainerHierarchy(src hierarchySource) (*ContainerHierarchy, error) {
	return newContainerHierarchyImpl(src)
}

// NewContainerHierarchyFromRoles creates a hierarchy from role-specific backend interfaces.
func NewContainerHierarchyFromRoles(ml backend.ModuleLister, mur backend.MetadataReader, fm backend.FolderManager) (*ContainerHierarchy, error) {
	return newContainerHierarchyImpl(hierarchyRolesSource{ml: ml, mur: mur, fm: fm})
}

// GetOrBuildHierarchy returns the cached hierarchy from deps.Cache if available,
// otherwise builds one from role-specific interfaces. Fn functions should call
// this instead of NewContainerHierarchyFromRoles to respect test cache setup.
func GetOrBuildHierarchy(deps *HandlerDeps) (*ContainerHierarchy, error) {
	if deps.Cache != nil && deps.Cache.hierarchy != nil {
		return deps.Cache.hierarchy, nil
	}
	return NewContainerHierarchyFromRoles(deps.ModuleLister, deps.MetadataReader, deps.FolderManager)
}

func newContainerHierarchyImpl(src hierarchySource) (*ContainerHierarchy, error) {
	h := &ContainerHierarchy{
		moduleIDs:       make(map[model.ID]bool),
		moduleNames:     make(map[model.ID]string),
		containerParent: make(map[model.ID]model.ID),
		folderNames:     make(map[model.ID]string),
	}

	modules, err := src.ListModules()
	if err != nil {
		return nil, err
	}
	for _, m := range modules {
		h.moduleIDs[m.ID] = true
		h.moduleNames[m.ID] = m.Name
	}

	units, _ := src.ListUnits()
	for _, u := range units {
		h.containerParent[u.ID] = u.ContainerID
	}

	folders, _ := src.ListFolders()
	for _, f := range folders {
		h.folderNames[f.ID] = f.Name
		h.containerParent[f.ID] = f.ContainerID
	}

	// The System domain model is a virtual document not present in ListUnits,
	// so its parent link would otherwise be missing. Pre-seed it here so that
	// FindModuleID resolves SystemDomainModelID → SystemModuleID correctly.
	h.containerParent[model.ID(meta.SystemDomainModelID)] = model.ID(meta.SystemModuleID)

	return h, nil
}

// FindModuleID finds the module ID for any container by traversing the hierarchy.
func (h *ContainerHierarchy) FindModuleID(containerID model.ID) model.ID {
	current := containerID
	for range 100 {
		if h.moduleIDs[current] {
			return current
		}
		parent, ok := h.containerParent[current]
		if !ok || parent == current {
			return containerID
		}
		current = parent
	}
	return containerID
}

// GetModuleName returns the module name for a module ID.
func (h *ContainerHierarchy) GetModuleName(moduleID model.ID) string {
	return h.moduleNames[moduleID]
}

// IsModule returns true if the ID is a module ID.
func (h *ContainerHierarchy) IsModule(id model.ID) bool {
	return h.moduleIDs[id]
}

// BuildFolderPath builds a folder path string from container to module.
func (h *ContainerHierarchy) BuildFolderPath(containerID model.ID) string {
	var parts []string
	current := containerID
	for range 100 {
		if h.moduleIDs[current] {
			break
		}
		if name := h.folderNames[current]; name != "" {
			parts = append([]string{name}, parts...)
		}
		parent, ok := h.containerParent[current]
		if !ok || parent == current {
			break
		}
		current = parent
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "/")
}

// ResolveFolderPath resolves a slash-separated folder path (e.g. "Ticket"
// or "Escalation/Sub") under the given container to its folder ID, using the
// in-memory hierarchy to avoid a SQLite ListFolders round-trip.
// Returns (id, true) on success, ("", false) if any segment is unknown.
func (h *ContainerHierarchy) ResolveFolderPath(containerID model.ID, folderPath string) (model.ID, bool) {
	parts := strings.Split(folderPath, "/")
	current := containerID
	for _, part := range parts {
		if part == "" {
			continue
		}
		found := false
		for id, name := range h.folderNames {
			if name == part && h.containerParent[id] == current {
				current = id
				found = true
				break
			}
		}
		if !found {
			return "", false
		}
	}
	return current, true
}

// GetQualifiedName returns the fully qualified name for a document.
func (h *ContainerHierarchy) GetQualifiedName(containerID model.ID, name string) string {
	modID := h.FindModuleID(containerID)
	modName := h.GetModuleName(modID)
	return modName + "." + name
}

// getHierarchy returns a cached ContainerHierarchy or creates a new one.
func getHierarchy(ctx *ExecContext) (*ContainerHierarchy, error) {
	if !ctx.Connected() {
		return nil, nil
	}
	ctx.initRoles()
	if ctx.Cache == nil {
		ctx.Cache = &executorCache{}
	}
	if ctx.Cache.hierarchy != nil {
		return ctx.Cache.hierarchy, nil
	}
	h, err := NewContainerHierarchy(hierarchyRolesSource{
		ml:  ctx.ModuleLister,
		mur: ctx.MetadataReader,
		fm:  ctx.FolderManager,
	})
	if err != nil {
		return nil, err
	}
	ctx.Cache.hierarchy = h
	return h, nil
}

// invalidateHierarchy clears the cached hierarchy so it will be rebuilt on next access.
// This should be called after any write operation that creates or deletes units.
func invalidateHierarchy(ctx *ExecContext) {
	if ctx.Cache != nil {
		ctx.Cache.hierarchy = nil
	}
}

// invalidateDomainModelsCache clears the Stage 3.3.4 gen-typed
// domain-model caches so the next read reloads from the writer. Call
// after any write that creates or modifies entities, associations,
// indexes, or generalizations.
func invalidateDomainModelsCache(ctx *ExecContext) {
	invalidateDomainModelsGenCache(ctx)
}
