// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
)

// listDataTransformers handles LIST DATA TRANSFORMERS [IN module].
func listDataTransformers(ctx *ExecContext, moduleName string) error {
	deps := execContextToDeps(ctx)
	return listDataTransformersFn(ctx, moduleName, deps)
}

// listDataTransformersFn is the HandlerDeps version of listDataTransformers.
func listDataTransformersFn(ctx context.Context, moduleName string, deps *HandlerDeps) error {
	transformers, err := deps.ServiceLister.ListDataTransformers()
	if err != nil {
		return mdlerrors.NewBackend("list data transformers", err)
	}

	h, err := NewContainerHierarchyFromRoles(deps.ModuleLister, deps.MetadataReader, deps.FolderManager)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	var rows [][]any
	for _, dt := range transformers {
		modID := h.FindModuleID(dt.ContainerID)
		modName := h.GetModuleName(modID)
		if moduleName != "" && !strings.EqualFold(modName, moduleName) {
			continue
		}
		qn := modName + "." + dt.Name
		steps := ""
		for _, s := range dt.Steps {
			if steps != "" {
				steps += " → "
			}
			steps += s.Technology
		}
		rows = append(rows, []any{qn, modName, dt.Name, dt.SourceType, steps})
	}

	if len(rows) == 0 {
		fmt.Fprintln(deps.Output, "No data transformers found.")
		return nil
	}

	result := &TableResult{
		Columns: []string{"Qualified Name", "Module", "Name", "Source", "Steps"},
		Rows:    rows,
		Summary: fmt.Sprintf("(%d data transformers)", len(rows)),
	}
	return writeResultTo(deps.Output, deps.Format, result)
}

// describeDataTransformer handles DESCRIBE DATA TRANSFORMER Module.Name.
func describeDataTransformer(ctx *ExecContext, name ast.QualifiedName) error {
	deps := execContextToDeps(ctx)
	return describeDataTransformerFn(ctx, name, deps)
}

// describeDataTransformerFn is the HandlerDeps version of describeDataTransformer.
func describeDataTransformerFn(ctx context.Context, name ast.QualifiedName, deps *HandlerDeps) error {
	transformers, err := deps.ServiceLister.ListDataTransformers()
	if err != nil {
		return mdlerrors.NewBackend("list data transformers", err)
	}

	h, err := NewContainerHierarchyFromRoles(deps.ModuleLister, deps.MetadataReader, deps.FolderManager)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	for _, dt := range transformers {
		modID := h.FindModuleID(dt.ContainerID)
		modName := h.GetModuleName(modID)
		if !strings.EqualFold(modName, name.Module) || !strings.EqualFold(dt.Name, name.Name) {
			continue
		}

		w := deps.Output

		fmt.Fprintf(w, "create data transformer %s.%s\n", modName, dt.Name)

		sourceContent := strings.ReplaceAll(dt.SourceJSON, "\n", " ")
		sourceContent = strings.ReplaceAll(sourceContent, "'", "''")
		fmt.Fprintf(w, "source %s '%s'\n", dt.SourceType, sourceContent)
		fmt.Fprintln(w, "{")

		for _, step := range dt.Steps {
			if strings.Contains(step.Expression, "\n") {
				fmt.Fprintf(w, "  %s $$\n%s\n  $$;\n", step.Technology, step.Expression)
			} else {
				expr := strings.ReplaceAll(step.Expression, "'", "''")
				fmt.Fprintf(w, "  %s '%s';\n", step.Technology, expr)
			}
		}

		fmt.Fprintln(w, "};")
		return nil
	}

	return mdlerrors.NewNotFound("data transformer", name.Module+"."+name.Name)
}

// execCreateDataTransformer creates a new data transformer.
func execCreateDataTransformer(ctx *ExecContext, s *ast.CreateDataTransformerStmt) error {
	deps := execContextToDeps(ctx)
	return execCreateDataTransformerFn(ctx, s, deps)
}

// execCreateDataTransformerFn is the HandlerDeps version of execCreateDataTransformer.
func execCreateDataTransformerFn(ctx context.Context, s *ast.CreateDataTransformerStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}

	if err := checkFeatureFn(ctx, deps, "integration", "data_transformer",
		"create data transformer",
		"upgrade your project to 11.9+"); err != nil {
		return err
	}

	existing, existingID := findDataTransformerFn(deps, s.Name.Module, s.Name.Name)
	if existing != nil && !s.CreateOrModify {
		return mdlerrors.NewAlreadyExists("data transformer", s.Name.String())
	}

	module, err := findModuleFn(deps.ModuleLister, s.Name.Module)
	if err != nil {
		return mdlerrors.NewNotFound("module", s.Name.Module)
	}

	dt := &model.DataTransformer{
		ContainerID: module.ID,
		Name:        s.Name.Name,
		SourceType:  s.SourceType,
		SourceJSON:  s.SourceJSON,
	}

	for _, step := range s.Steps {
		dt.Steps = append(dt.Steps, &model.DataTransformerStep{
			Technology: step.Technology,
			Expression: step.Expression,
		})
	}

	if existingID != "" {
		dt.ID = existingID
		if err := deps.ServiceWriter.UpdateDataTransformer(dt); err != nil {
			return mdlerrors.NewBackend("update data transformer", err)
		}
		if !deps.Quiet {
			fmt.Fprintf(deps.Output, "Modified data transformer: %s.%s (%d steps)\n",
				s.Name.Module, s.Name.Name, len(dt.Steps))
		}
		return nil
	}

	if err := deps.ServiceWriter.CreateDataTransformer(dt); err != nil {
		return mdlerrors.NewBackend("create data transformer", err)
	}

	if !deps.Quiet {
		fmt.Fprintf(deps.Output, "Created data transformer: %s.%s (%d steps)\n",
			s.Name.Module, s.Name.Name, len(dt.Steps))
	}
	return nil
}

// findDataTransformer looks up a data transformer by module and name, returning the struct and its ID.
func findDataTransformer(ctx *ExecContext, moduleName, name string) (*model.DataTransformer, model.ID) {
	return findDataTransformerFn(execContextToDeps(ctx), moduleName, name)
}

// findDataTransformerFn is the HandlerDeps version of findDataTransformer.
func findDataTransformerFn(deps *HandlerDeps, moduleName, name string) (*model.DataTransformer, model.ID) {
	transformers, err := deps.ServiceLister.ListDataTransformers()
	if err != nil {
		return nil, ""
	}
	h, err := NewContainerHierarchyFromRoles(deps.ModuleLister, deps.MetadataReader, deps.FolderManager)
	if err != nil {
		return nil, ""
	}
	for _, dt := range transformers {
		modID := h.FindModuleID(dt.ContainerID)
		modName := h.GetModuleName(modID)
		if strings.EqualFold(modName, moduleName) && strings.EqualFold(dt.Name, name) {
			return dt, dt.ID
		}
	}
	return nil, ""
}

// execDropDataTransformer deletes a data transformer.
func execDropDataTransformer(ctx *ExecContext, s *ast.DropDataTransformerStmt) error {
	deps := execContextToDeps(ctx)
	return execDropDataTransformerFn(ctx, s, deps)
}

// execDropDataTransformerFn is the HandlerDeps version of execDropDataTransformer.
func execDropDataTransformerFn(ctx context.Context, s *ast.DropDataTransformerStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}

	transformers, err := deps.ServiceLister.ListDataTransformers()
	if err != nil {
		return mdlerrors.NewBackend("list data transformers", err)
	}

	h, err := NewContainerHierarchyFromRoles(deps.ModuleLister, deps.MetadataReader, deps.FolderManager)
	if err != nil {
		return err
	}

	for _, dt := range transformers {
		modID := h.FindModuleID(dt.ContainerID)
		modName := h.GetModuleName(modID)
		if modName == s.Name.Module && dt.Name == s.Name.Name {
			if err := deps.ServiceWriter.DeleteDataTransformer(dt.ID); err != nil {
				return mdlerrors.NewBackend("drop data transformer", err)
			}
			if !deps.Quiet {
				fmt.Fprintf(deps.Output, "Dropped data transformer: %s.%s\n", s.Name.Module, s.Name.Name)
			}
			return nil
		}
	}

	return mdlerrors.NewNotFound("data transformer", s.Name.Module+"."+s.Name.Name)
}

