// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	genProj "github.com/mendixlabs/mxcli/modelsdk/gen/projects"
	"github.com/mendixlabs/mxcli/modelsdk/mprread"
)

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
