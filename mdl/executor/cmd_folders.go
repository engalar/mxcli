// SPDX-License-Identifier: Apache-2.0

// Package executor - DROP/MOVE FOLDER commands
package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

// findFolderByPath walks a folder path under a module and returns the folder ID.
func findFolderByPath(ctx *ExecContext, moduleID model.ID, folderPath string, folders []*types.FolderInfo) (model.ID, error) {
	parts := strings.Split(folderPath, "/")
	currentContainerID := moduleID

	var targetFolderID model.ID
	for i, part := range parts {
		if part == "" {
			continue
		}

		var found bool
		for _, f := range folders {
			if f.ContainerID == currentContainerID && f.Name == part {
				currentContainerID = f.ID
				if i == len(parts)-1 {
					targetFolderID = f.ID
				}
				found = true
				break
			}
		}

		if !found {
			return "", mdlerrors.NewNotFound("folder", folderPath)
		}
	}

	if targetFolderID == "" {
		return "", mdlerrors.NewNotFound("folder", folderPath)
	}

	return targetFolderID, nil
}

// findFolderByPathFn is the HandlerDeps version of findFolderByPath.
func findFolderByPathFn(_ *HandlerDeps, moduleID model.ID, folderPath string, folders []*types.FolderInfo) (model.ID, error) {
	parts := strings.Split(folderPath, "/")
	currentContainerID := moduleID

	var targetFolderID model.ID
	for i, part := range parts {
		if part == "" {
			continue
		}
		var found bool
		for _, f := range folders {
			if f.ContainerID == currentContainerID && f.Name == part {
				currentContainerID = f.ID
				if i == len(parts)-1 {
					targetFolderID = f.ID
				}
				found = true
				break
			}
		}
		if !found {
			return "", mdlerrors.NewNotFound("folder", folderPath)
		}
	}
	if targetFolderID == "" {
		return "", mdlerrors.NewNotFound("folder", folderPath)
	}
	return targetFolderID, nil
}

// execDropFolder handles DROP FOLDER 'path' IN Module statements.
// The folder must be empty (no child documents or sub-folders).

// execDropFolderFn is the HandlerDeps version of execDropFolder.
func execDropFolderFn(ctx context.Context, s *ast.DropFolderStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	module, err := findModuleFn(deps.ModuleLister, s.Module)
	if err != nil {
		return mdlerrors.NewNotFound("module", s.Module)
	}

	folders, err := deps.FolderManager.ListFolders()
	if err != nil {
		return mdlerrors.NewBackend("list folders", err)
	}

	folderID, err := findFolderByPathFn(deps, module.ID, s.FolderPath, folders)
	if err != nil {
		return fmt.Errorf("%w in %s", err, s.Module)
	}

	if err := deps.FolderManager.DeleteFolder(folderID); err != nil {
		return mdlerrors.NewBackend(fmt.Sprintf("delete folder '%s'", s.FolderPath), err)
	}

	invalidateHierarchyFn(deps)
	fmt.Fprintf(deps.Output, "Dropped folder: '%s' in %s\n", s.FolderPath, s.Module)
	return nil
}

// execMoveFolder handles MOVE FOLDER Module.FolderName TO ... statements.

// execMoveFolderFn is the HandlerDeps version of execMoveFolder.
func execMoveFolderFn(ctx context.Context, s *ast.MoveFolderStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	sourceModule, err := findModuleFn(deps.ModuleLister, s.Name.Module)
	if err != nil {
		return mdlerrors.NewNotFound("source module", s.Name.Module)
	}

	folders, err := deps.FolderManager.ListFolders()
	if err != nil {
		return mdlerrors.NewBackend("list folders", err)
	}

	folderID, err := findFolderByPathFn(deps, sourceModule.ID, s.Name.Name, folders)
	if err != nil {
		return fmt.Errorf("%w in %s", err, s.Name.Module)
	}

	var targetModule *model.Module
	if s.TargetModule != "" {
		targetModule, err = findModuleFn(deps.ModuleLister, s.TargetModule)
		if err != nil {
			return mdlerrors.NewNotFound("target module", s.TargetModule)
		}
	} else {
		targetModule = sourceModule
	}

	var targetContainerID model.ID
	if s.TargetFolder != "" {
		targetContainerID, err = resolveFolderDeps(deps, targetModule.ID, s.TargetFolder, nil)
		if err != nil {
			return mdlerrors.NewBackend("resolve target folder", err)
		}
	} else {
		targetContainerID = targetModule.ID
	}

	if err := deps.FolderManager.MoveFolder(folderID, targetContainerID); err != nil {
		return mdlerrors.NewBackend("move folder", err)
	}

	invalidateHierarchyFn(deps)

	target := targetModule.Name
	if s.TargetFolder != "" {
		target += "/" + s.TargetFolder
	}
	fmt.Fprintf(deps.Output, "Moved folder %s to %s\n", s.Name.String(), target)
	return nil
}
