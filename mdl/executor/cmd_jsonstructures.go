// SPDX-License-Identifier: Apache-2.0

// Package executor - JSON structure commands (SHOW/DESCRIBE/CREATE/DROP JSON STRUCTURE)
package executor

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

func listJsonStructuresFn(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	structures, err := deps.MapperReader.ListJsonStructures()
	if err != nil {
		return mdlerrors.NewBackend("list json structures", err)
	}
	ectx := phase3d2bNewExecContext(ctx, deps)
	h, err := getHierarchy(ectx)
	if err != nil {
		return err
	}
	type row struct {
		qualifiedName string
		elemCount     int
		source        string
	}
	var rows []row
	for _, js := range structures {
		modID := h.FindModuleID(js.ContainerID)
		modName := h.GetModuleName(modID)
		if moduleName != "" && modName != moduleName {
			continue
		}
		qualifiedName := fmt.Sprintf("%s.%s", modName, js.Name)
		elemCount := 0
		if len(js.Elements) > 0 {
			elemCount = len(js.Elements[0].Children)
		}
		source := "Manual"
		if js.JsonSnippet != "" {
			source = "json Snippet"
		}
		rows = append(rows, row{qualifiedName: qualifiedName, elemCount: elemCount, source: source})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].qualifiedName < rows[j].qualifiedName })
	tr := &TableResult{
		Columns: []string{"json Structure", "Elements", "Source"},
		Summary: fmt.Sprintf("(%d json structure(s))", len(rows)),
	}
	for _, r := range rows {
		tr.Rows = append(tr.Rows, []any{r.qualifiedName, r.elemCount, r.source})
	}
	return writeResultTo(deps.Output, deps.Format, tr)
}

func describeJsonStructureFn(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	js := findJsonStructureFn(deps, name.Module, name.Name)
	if js == nil {
		return mdlerrors.NewNotFound("json structure", name.String())
	}
	ectx := phase3d2bNewExecContext(ctx, deps)
	h, err := getHierarchy(ectx)
	if err != nil {
		return err
	}
	modID := h.FindModuleID(js.ContainerID)
	modName := h.GetModuleName(modID)
	qualifiedName := fmt.Sprintf("%s.%s", modName, js.Name)
	if js.Documentation != "" {
		fmt.Fprintf(deps.Output, "/**\n * %s\n */\n", js.Documentation)
	}
	fmt.Fprintf(deps.Output, "create or modify json structure %s", qualifiedName)
	if folderPath := h.BuildFolderPath(js.ContainerID); folderPath != "" {
		fmt.Fprintf(deps.Output, "\n  folder '%s'", folderPath)
	}
	if js.Documentation != "" {
		fmt.Fprintf(deps.Output, "\n  comment '%s'", strings.ReplaceAll(js.Documentation, "'", "''"))
	}
	if js.JsonSnippet != "" {
		snippet := types.PrettyPrintJSON(js.JsonSnippet)
		if strings.Contains(snippet, "'") || strings.Contains(snippet, "\n") {
			fmt.Fprintf(deps.Output, "\n  snippet $$%s$$", snippet)
		} else {
			fmt.Fprintf(deps.Output, "\n  snippet '%s'", snippet)
		}
	}
	customMappings := collectCustomNameMappings(js.Elements)
	if len(customMappings) > 0 {
		keys := make([]string, 0, len(customMappings))
		for k := range customMappings {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fmt.Fprintf(deps.Output, "\n  CUSTOM NAME map (\n")
		for i, jsonKey := range keys {
			sep := ","
			if i == len(keys)-1 {
				sep = ""
			}
			fmt.Fprintf(deps.Output, "    '%s' as '%s'%s\n", jsonKey, customMappings[jsonKey], sep)
		}
		fmt.Fprintf(deps.Output, "  )")
	}
	fmt.Fprintln(deps.Output, ";")
	return nil
}

func execCreateJsonStructureFn(ctx context.Context, s *ast.CreateJsonStructureStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}
	ectx := phase3d2bNewExecContext(ctx, deps)
	module, err := findOrCreateModule(ectx, s.Name.Module)
	if err != nil {
		return err
	}
	containerID := module.ID
	if s.Folder != "" {
		folderID, err := resolveFolder(ectx, module.ID, s.Folder, nil)
		if err != nil {
			return mdlerrors.NewBackend("resolve folder "+s.Folder, err)
		}
		containerID = folderID
	}
	existing := findJsonStructureFn(deps, s.Name.Module, s.Name.Name)
	if existing != nil && !s.CreateOrModify {
		return mdlerrors.NewAlreadyExists("json structure", s.Name.Module+"."+s.Name.Name)
	}
	elements, err := types.BuildJsonElementsFromSnippet(s.JsonSnippet, s.CustomNameMap)
	if err != nil {
		return mdlerrors.NewBackend("build element tree", err)
	}
	if existing != nil && s.Folder == "" {
		containerID = existing.ContainerID
	}
	js := &types.JsonStructure{
		ContainerID:   containerID,
		Name:          s.Name.Name,
		Documentation: s.Documentation,
		JsonSnippet:   types.PrettyPrintJSON(s.JsonSnippet),
		Elements:      elements,
	}
	if existing != nil {
		js.ID = existing.ID
		if err := deps.MapperWriter.UpdateJsonStructure(js); err != nil {
			return mdlerrors.NewBackend("update json structure", err)
		}
		fmt.Fprintf(deps.Output, "Modified json structure: %s\n", s.Name)
	} else {
		if err := deps.MapperWriter.CreateJsonStructure(js); err != nil {
			return mdlerrors.NewBackend("create json structure", err)
		}
		fmt.Fprintf(deps.Output, "Created json structure: %s\n", s.Name)
	}
	invalidateHierarchy(ectx)
	return nil
}

func execDropJsonStructureFn(ctx context.Context, s *ast.DropJsonStructureStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}
	js := findJsonStructureFn(deps, s.Name.Module, s.Name.Name)
	if js == nil {
		return mdlerrors.NewNotFound("json structure", s.Name.String())
	}
	if err := deps.MapperWriter.DeleteJsonStructure(string(js.ID)); err != nil {
		return mdlerrors.NewBackend("delete json structure", err)
	}
	fmt.Fprintf(deps.Output, "Dropped json structure: %s\n", s.Name)
	return nil
}

// findJsonStructureFn finds a JSON structure by module and name using deps.
func findJsonStructureFn(deps *HandlerDeps, moduleName, structName string) *types.JsonStructure {
	structures, err := deps.MapperReader.ListJsonStructures()
	if err != nil {
		return nil
	}
	for _, js := range structures {
		if js.Name == structName {
			modID := findModuleByName(deps.ModuleLister, moduleName)
			if modID != "" {
				if js.ContainerID == modID {
					return js
				}
			}
		}
	}
	return nil
}

func listJsonStructures(ctx *ExecContext, moduleName string) error {
	return listJsonStructuresFn(ctx, execContextToDeps(ctx), moduleName)
}

func describeJsonStructure(ctx *ExecContext, name ast.QualifiedName) error {
	return describeJsonStructureFn(ctx, execContextToDeps(ctx), name)
}

func execCreateJsonStructure(ctx *ExecContext, s *ast.CreateJsonStructureStmt) error {
	return execCreateJsonStructureFn(ctx, s, execContextToDeps(ctx))
}

func execDropJsonStructure(ctx *ExecContext, s *ast.DropJsonStructureStmt) error {
	return execDropJsonStructureFn(ctx, s, execContextToDeps(ctx))
}

func findJsonStructure(ctx *ExecContext, moduleName, structName string) *types.JsonStructure {
	return findJsonStructureFn(execContextToDeps(ctx), moduleName, structName)
}

// collectCustomNameMappings walks the element tree and returns JSON key → ExposedName
// mappings where the ExposedName differs from the auto-generated default (capitalizeFirst).
func collectCustomNameMappings(elements []*types.JsonElement) map[string]string {
	mappings := make(map[string]string)
	for _, elem := range elements {
		collectCustomNames(elem, mappings)
	}
	return mappings
}

func collectCustomNames(elem *types.JsonElement, mappings map[string]string) {
	if parts := strings.Split(elem.Path, "|"); len(parts) > 1 {
		jsonKey := parts[len(parts)-1]
		if jsonKey != "" && jsonKey[0] != '(' {
			expected := capitalizeFirstRune(jsonKey)
			if elem.ExposedName != expected && elem.ExposedName != "" {
				mappings[jsonKey] = elem.ExposedName
			}
		}
	}
	for _, child := range elem.Children {
		collectCustomNames(child, mappings)
	}
}

// capitalizeFirstRune capitalizes the first rune of s (for ExposedName comparison).
func capitalizeFirstRune(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// findModuleByName looks up a module ID by name.
func findModuleByName(ml backend.ModuleLister, moduleName string) model.ID {
	if ml == nil {
		return ""
	}
	modules, err := ml.ListModules()
	if err != nil {
		return ""
	}
	for _, m := range modules {
		if m.Name == moduleName {
			return m.ID
		}
	}
	return ""
}


