// SPDX-License-Identifier: Apache-2.0

// Package executor - Layout commands (SHOW LAYOUTS)
package executor

import (
	"fmt"
	"sort"
	"strings"

	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
)

// listLayouts handles SHOW LAYOUTS command.
func listLayouts(ctx *ExecContext, moduleName string) error {
	// Get hierarchy for module/folder resolution
	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	// Get all layouts
	layoutPairs, err := listLayoutsWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list layouts", err)
	}

	// Collect rows
	type row struct {
		qualifiedName string
		module        string
		name          string
		folderPath    string
		layoutType    string
	}
	var rows []row

	for _, pair := range layoutPairs {
		l := pair.Elem
		containerID := model.ID(pair.ContainerID)
		modID := h.FindModuleID(containerID)
		modName := h.GetModuleName(modID)
		if moduleName == "" || modName == moduleName {
			qualifiedName := modName + "." + l.Name()
			folderPath := h.BuildFolderPath(containerID)
			layoutType := l.LayoutType()

			rows = append(rows, row{qualifiedName, modName, l.Name(), folderPath, layoutType})
		}
	}

	// Sort by qualified name
	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	result := &TableResult{
		Columns: []string{"Qualified Name", "Module", "Name", "Folder", "Type"},
		Summary: fmt.Sprintf("(%d layouts)", len(rows)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.qualifiedName, r.module, r.name, r.folderPath, r.layoutType})
	}
	return writeResult(ctx, result)
}
