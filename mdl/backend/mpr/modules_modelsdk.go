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
