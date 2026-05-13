// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	msdkprojects "github.com/mendixlabs/mxcli/modelsdk/gen/projects"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
)

// ────────────────────────────────────────────────────────
// DeleteModule / DeleteModuleWithCleanup
// ────────────────────────────────────────────────────────

func (b *MprBackend) deleteModuleViaModelsdk(id model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	if err := b.msdkWriter.DeleteChildUnits(string(id)); err != nil {
		return fmt.Errorf("delete child units: %w", err)
	}
	return b.msdkWriter.DeleteUnit(string(id))
}

func (b *MprBackend) deleteModuleWithCleanupViaModelsdk(id model.ID, moduleName string) error {
	if err := b.deleteModuleViaModelsdk(id); err != nil {
		return err
	}
	projectDir := filepath.Dir(b.path)
	themesourceDir := filepath.Join(projectDir, "themesource", strings.ToLower(moduleName))
	if stat, err := os.Stat(themesourceDir); err == nil && stat.IsDir() {
		_ = os.RemoveAll(themesourceDir)
	}
	return nil
}

// ────────────────────────────────────────────────────────
// DeleteFolder / MoveFolder
// ────────────────────────────────────────────────────────

func (b *MprBackend) deleteFolderViaModelsdk(id model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	// 与旧 sdk/mpr.Writer.DeleteFolder 行为一致：非空文件夹拒绝删除
	db := b.msdkWriter.Reader().DB()
	blob := modelsdkmpr.IDToBsonBinary(string(id)).Data
	var count int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM Unit WHERE ContainerID = ? AND UnitID != ContainerID",
		blob,
	).Scan(&count); err != nil {
		return fmt.Errorf("check folder contents: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("folder is not empty: contains %d child unit(s)", count)
	}
	return b.msdkWriter.DeleteUnit(string(id))
}

func (b *MprBackend) moveFolderViaModelsdk(id, newContainerID model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.UpdateUnitContainer(string(id), string(newContainerID))
}

// ────────────────────────────────────────────────────────
// UpdateModule
// ────────────────────────────────────────────────────────

func (b *MprBackend) updateModuleViaModelsdk(module *model.Module) error {
	return b.msdkWrite(model.ID(module.ID), func(elem element.Element) error {
		mod, ok := elem.(*msdkprojects.Module)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *projects.Module)", elem)
		}
		mod.SetName(module.Name)
		mod.SetFromAppStore(module.FromAppStore)
		mod.SetIsReusableComponent(module.IsReusableComponent)
		mod.SetAppStoreGuid(module.AppStoreGuid)
		mod.SetAppStoreVersion(module.AppStoreVersion)
		return nil
	})
}

// ────────────────────────────────────────────────────────
// UpdateModuleSettings
// ────────────────────────────────────────────────────────

func (b *MprBackend) updateModuleSettingsViaModelsdk(ms *types.ModuleSettings) error {
	return b.msdkWrite(model.ID(ms.ID), func(elem element.Element) error {
		settings, ok := elem.(*msdkprojects.ModuleSettings)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *projects.ModuleSettings)", elem)
		}

		exportLevel := ms.ExportLevel
		if exportLevel == "" {
			exportLevel = "Source"
		}
		protectedType := ms.ProtectedModuleType
		if protectedType == "" {
			protectedType = "AddOn"
		}
		ver := ms.Version
		if ver == "" {
			ver = "1.0.0"
		}

		settings.SetExportLevel(exportLevel)
		settings.SetProtectedModuleType(protectedType)
		settings.SetVersion(ver)
		settings.SetBasedOnVersion(ms.BasedOnVersion)
		settings.SetExtensionName(ms.ExtensionName)
		settings.SetSolutionIdentifier(ms.SolutionIdentifier)

		// Replace JarDependencies wholesale.
		for i := len(settings.JarDependenciesItems()) - 1; i >= 0; i-- {
			settings.RemoveJarDependencies(i)
		}
		for _, d := range ms.JarDependencies {
			depID := string(d.ID)
			if depID == "" {
				depID = modelsdkmpr.GenerateID()
			}
			dep := msdkprojects.NewJarDependency()
			dep.SetID(element.ID(depID))
			dep.SetGroupId(d.GroupID)
			dep.SetArtifactId(d.ArtifactID)
			dep.SetVersion(d.Version)
			dep.SetIsIncluded(d.IsIncluded)
			for _, e := range d.Exclusions {
				excID := string(e.ID)
				if excID == "" {
					excID = modelsdkmpr.GenerateID()
				}
				exc := msdkprojects.NewJarDependencyExclusion()
				exc.SetID(element.ID(excID))
				exc.SetGroupId(e.GroupID)
				exc.SetArtifactId(e.ArtifactID)
				dep.AddExclusions(exc)
			}
			settings.AddJarDependencies(dep)
		}
		return nil
	})
}

// ────────────────────────────────────────────────────────
// CreateModule
// ────────────────────────────────────────────────────────

func (b *MprBackend) createModuleViaModelsdk(module *model.Module) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	if module.ID == "" {
		module.ID = model.ID(modelsdkmpr.GenerateID())
	}
	module.TypeName = "Projects$ModuleImpl"

	projectRootID, err := b.reader.GetProjectRootID()
	if err != nil {
		return fmt.Errorf("failed to get project root: %w", err)
	}

	moduleContents, err := b.writer.SerializeModule(module)
	if err != nil {
		return fmt.Errorf("failed to serialize module: %w", err)
	}
	if err := b.msdkWriter.InsertUnit(string(module.ID), projectRootID, "Modules", "Projects$ModuleImpl", moduleContents); err != nil {
		return fmt.Errorf("failed to insert module unit: %w", err)
	}

	dmID := modelsdkmpr.GenerateID()
	dm := &domainmodel.DomainModel{
		ContainerID: module.ID,
	}
	dm.ID = model.ID(dmID)
	dm.TypeName = "DomainModels$DomainModel"
	dmContents, err := b.writer.SerializeDomainModel(dm)
	if err != nil {
		return fmt.Errorf("failed to serialize domain model: %w", err)
	}
	if err := b.msdkWriter.InsertUnit(dmID, string(module.ID), "DomainModel", "DomainModels$DomainModel", dmContents); err != nil {
		return fmt.Errorf("failed to insert domain model unit: %w", err)
	}

	msID := modelsdkmpr.GenerateID()
	msContents, err := b.writer.SerializeModuleSecurity(msID)
	if err != nil {
		return fmt.Errorf("failed to serialize module security: %w", err)
	}
	if err := b.msdkWriter.InsertUnit(msID, string(module.ID), "ModuleSecurity", "Security$ModuleSecurity", msContents); err != nil {
		return fmt.Errorf("failed to insert module security unit: %w", err)
	}

	settingsID := modelsdkmpr.GenerateID()
	settingsContents, err := b.writer.SerializeModuleSettings(settingsID)
	if err != nil {
		return fmt.Errorf("failed to serialize module settings: %w", err)
	}
	if err := b.msdkWriter.InsertUnit(settingsID, string(module.ID), "ModuleSettings", "Projects$ModuleSettings", settingsContents); err != nil {
		return fmt.Errorf("failed to insert module settings unit: %w", err)
	}

	b.reader.InvalidateCache()
	return nil
}

// ────────────────────────────────────────────────────────
// CreateFolder
// ────────────────────────────────────────────────────────

func (b *MprBackend) createFolderViaModelsdk(folder *model.Folder) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	if folder.ID == "" {
		folder.ID = model.ID(modelsdkmpr.GenerateID())
	}
	folder.TypeName = "Projects$Folder"

	contents, err := b.writer.SerializeFolder(folder)
	if err != nil {
		return fmt.Errorf("failed to serialize folder: %w", err)
	}
	if err := b.msdkWriter.InsertUnit(string(folder.ID), string(folder.ContainerID), "Folders", "Projects$Folder", contents); err != nil {
		return fmt.Errorf("failed to insert folder unit: %w", err)
	}

	b.reader.InvalidateCache()
	return nil
}
