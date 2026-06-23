// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
)

// ────────────────────────────────────────────────────────────
// Fn (HandlerDeps) versions
// ────────────────────────────────────────────────────────────

func listPublishedRestServicesFn(ctx context.Context, output io.Writer, format OutputFormat, deps *HandlerDeps, moduleName string) error {
	services, err := deps.ServiceLister.ListPublishedRestServices()
	if err != nil {
		return mdlerrors.NewBackend("list published rest services", err)
	}

	h, err := GetOrBuildHierarchy(deps)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	type row struct {
		module        string
		qualifiedName string
		path          string
		version       string
		resources     int
		operations    int
	}
	var rows []row

	for _, svc := range services {
		modID := h.FindModuleID(svc.ContainerID)
		modName := h.GetModuleName(modID)
		if moduleName != "" && !strings.EqualFold(modName, moduleName) {
			continue
		}

		qn := modName + "." + svc.Name
		opCount := 0
		for _, res := range svc.Resources {
			opCount += len(res.Operations)
		}

		path := svc.Path
		if len(path) > 50 {
			path = path[:47] + "..."
		}

		rows = append(rows, row{modName, qn, path, svc.Version, len(svc.Resources), opCount})
	}

	if len(rows) == 0 && format != FormatJSON {
		fmt.Fprintln(output, "No published rest services found.")
		return nil
	}

	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	result := &TableResult{
		Columns: []string{"Module", "QualifiedName", "Path", "Version", "Resources", "Operations"},
		Summary: fmt.Sprintf("(%d published rest services)", len(rows)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.module, r.qualifiedName, r.path, r.version, r.resources, r.operations})
	}
	return writeResultTo(output, format, result)
}

func describePublishedRestServiceFn(ctx context.Context, output io.Writer, deps *HandlerDeps, name ast.QualifiedName) error {
	services, err := deps.ServiceLister.ListPublishedRestServices()
	if err != nil {
		return mdlerrors.NewBackend("list published rest services", err)
	}

	h, err := GetOrBuildHierarchy(deps)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	for _, svc := range services {
		modID := h.FindModuleID(svc.ContainerID)
		modName := h.GetModuleName(modID)
		qualifiedName := modName + "." + svc.Name

		if !strings.EqualFold(modName, name.Module) || !strings.EqualFold(svc.Name, name.Name) {
			continue
		}

		fmt.Fprintf(output, "create or modify published rest service %s (\n", qualifiedName)
		fmt.Fprintf(output, "  Path: '%s'", svc.Path)
		if svc.Version != "" {
			fmt.Fprintf(output, ",\n  Version: '%s'", svc.Version)
		}
		if svc.ServiceName != "" {
			fmt.Fprintf(output, ",\n  ServiceName: '%s'", svc.ServiceName)
		}
		folderPath := h.BuildFolderPath(svc.ContainerID)
		if folderPath != "" {
			fmt.Fprintf(output, ",\n  Folder: '%s'", folderPath)
		}
		fmt.Fprintln(output, "\n)")

		if len(svc.Resources) > 0 {
			fmt.Fprintln(output, "{")
			for _, res := range svc.Resources {
				fmt.Fprintf(output, "  resource '%s' {\n", res.Name)
				for _, op := range res.Operations {
					deprecated := ""
					if op.Deprecated {
						deprecated = " deprecated"
					}
					mf := ""
					if op.Microflow != "" {
						mf = fmt.Sprintf(" microflow %s", op.Microflow)
					}
					summary := ""
					if op.Summary != "" {
						summary = fmt.Sprintf(" -- %s", op.Summary)
					}
					opPath := ""
					if op.Path != "" {
						opPath = fmt.Sprintf(" '%s'", op.Path)
					}
					fmt.Fprintf(output, "    %s%s%s%s;%s\n",
						strings.ToUpper(op.HTTPMethod), opPath, mf, deprecated, summary)
				}
				fmt.Fprintln(output, "  }")
			}
			fmt.Fprintln(output, "};")
		} else {
			fmt.Fprintln(output, ";")
		}
		fmt.Fprintln(output, "/")

		if len(svc.AllowedRoles) > 0 {
			fmt.Fprintf(output, "\ngrant access on published rest service %s.%s to %s;\n",
				modName, svc.Name, strings.Join(svc.AllowedRoles, ", "))
		}

		return nil
	}

	return mdlerrors.NewNotFound("published rest service", name.String())
}

func findPublishedRestServiceFn(deps *HandlerDeps, moduleName, name string) (*model.PublishedRestService, error) {
	services, err := deps.ServiceLister.ListPublishedRestServices()
	if err != nil {
		return nil, err
	}
	h, err := GetOrBuildHierarchy(deps)
	if err != nil {
		return nil, err
	}
	for _, svc := range services {
		modID := h.FindModuleID(svc.ContainerID)
		modName := h.GetModuleName(modID)
		if modName == moduleName && svc.Name == name {
			return svc, nil
		}
	}
	return nil, mdlerrors.NewNotFound("published rest service", moduleName+"."+name)
}

func execCreatePublishedRestServiceFn(ctx context.Context, s *ast.CreatePublishedRestServiceStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}

	tmpCtx := NewExecContext(ctx, deps)

	if err := checkFeature(tmpCtx, "integration", "published_rest_service",
		"create published rest service",
		"upgrade your project to 10.0+"); err != nil {
		return err
	}

	// Check for existing service
	existing, findErr := findPublishedRestServiceFn(deps, s.Name.Module, s.Name.Name)
	var nfe *mdlerrors.NotFoundError
	if findErr != nil && !errors.As(findErr, &nfe) {
		return mdlerrors.NewBackend("find existing service", findErr)
	}
	if existing != nil && !s.CreateOrModify {
		return mdlerrors.NewAlreadyExistsMsg("published rest service", s.Name.Module+"."+s.Name.Name,
			fmt.Sprintf("published rest service already exists: %s.%s (use create or modify to update)", s.Name.Module, s.Name.Name))
	}

	module, err := findModule(tmpCtx, s.Name.Module)
	if err != nil {
		return mdlerrors.NewNotFound("module", s.Name.Module)
	}

	containerID := module.ID
	if s.Folder != "" {
		folderID, err := resolveFolder(tmpCtx, module.ID, s.Folder, nil)
		if err != nil {
			return mdlerrors.NewBackend(fmt.Sprintf("resolve folder '%s'", s.Folder), err)
		}
		containerID = folderID
	}

	svc := &model.PublishedRestService{
		ContainerID: containerID,
		Name:        s.Name.Name,
		Path:        s.Path,
		Version:     s.Version,
		ServiceName: s.ServiceName,
	}
	if existing != nil {
		svc.ID = existing.ID
		svc.AllowedRoles = existing.AllowedRoles
	}

	for _, resDef := range s.Resources {
		resource := &model.PublishedRestResource{
			Name: resDef.Name,
		}
		for _, opDef := range resDef.Operations {
			op := &model.PublishedRestOperation{
				HTTPMethod: opDef.HTTPMethod,
				Path:       opDef.Path,
				Microflow:  opDef.Microflow.String(),
				Summary:    "",
				Deprecated: opDef.Deprecated,
			}
			resource.Operations = append(resource.Operations, op)
		}
		svc.Resources = append(svc.Resources, resource)
	}

	if existing != nil {
		if err := deps.ServiceWriter.UpdatePublishedRestService(svc); err != nil {
			return mdlerrors.NewBackend("update published rest service", err)
		}
		if !deps.Quiet {
			fmt.Fprintf(deps.Output, "Modified published rest service %s.%s\n", s.Name.Module, s.Name.Name)
		}
	} else {
		if err := deps.ServiceWriter.CreatePublishedRestService(svc); err != nil {
			return mdlerrors.NewBackend("create published rest service", err)
		}
		if !deps.Quiet {
			fmt.Fprintf(deps.Output, "Created published rest service %s.%s\n", s.Name.Module, s.Name.Name)
		}
	}
	return nil
}

func execDropPublishedRestServiceFn(ctx context.Context, s *ast.DropPublishedRestServiceStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}

	services, err := deps.ServiceLister.ListPublishedRestServices()
	if err != nil {
		return mdlerrors.NewBackend("list published rest services", err)
	}

	h, err := GetOrBuildHierarchy(deps)
	if err != nil {
		return err
	}

	for _, svc := range services {
		modID := h.FindModuleID(svc.ContainerID)
		modName := h.GetModuleName(modID)
		if modName == s.Name.Module && svc.Name == s.Name.Name {
			if err := deps.ServiceWriter.DeletePublishedRestService(svc.ID); err != nil {
				return mdlerrors.NewBackend("drop published rest service", err)
			}
			if !deps.Quiet {
				fmt.Fprintf(deps.Output, "Dropped published rest service %s.%s\n", s.Name.Module, s.Name.Name)
			}
			return nil
		}
	}

	return mdlerrors.NewNotFound("published rest service", s.Name.Module+"."+s.Name.Name)
}

func execAlterPublishedRestServiceFn(ctx context.Context, s *ast.AlterPublishedRestServiceStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}

	tmpCtx := NewExecContext(ctx, deps)

	if err := checkFeature(tmpCtx, "integration", "published_rest_alter",
		"alter published rest service",
		"upgrade your project to 10.0+"); err != nil {
		return err
	}

	svc, err := findPublishedRestServiceFn(deps, s.Name.Module, s.Name.Name)
	if err != nil {
		return err
	}

	for _, action := range s.Actions {
		switch a := action.(type) {
		case *ast.PublishedRestSetAction:
			for key, val := range a.Changes {
				switch strings.ToLower(key) {
				case "path":
					svc.Path = val
				case "version":
					svc.Version = val
				case "servicename":
					svc.ServiceName = val
				default:
					return mdlerrors.NewUnsupported(fmt.Sprintf("unknown published rest service property: %s (allowed: Path, Version, ServiceName)", key))
				}
			}

		case *ast.PublishedRestAddResourceAction:
			for _, existing := range svc.Resources {
				if existing.Name == a.Resource.Name {
					return mdlerrors.NewAlreadyExistsMsg("resource", a.Resource.Name, fmt.Sprintf("resource '%s' already exists on %s.%s", a.Resource.Name, s.Name.Module, s.Name.Name))
				}
			}
			svc.Resources = append(svc.Resources, astResourceDefToModel(a.Resource))

		case *ast.PublishedRestDropResourceAction:
			idx := -1
			for i, existing := range svc.Resources {
				if existing.Name == a.Name {
					idx = i
					break
				}
			}
			if idx == -1 {
				return mdlerrors.NewNotFoundMsg("resource", a.Name, fmt.Sprintf("resource '%s' not found on %s.%s", a.Name, s.Name.Module, s.Name.Name))
			}
			svc.Resources = append(svc.Resources[:idx], svc.Resources[idx+1:]...)

		default:
			return mdlerrors.NewUnsupported(fmt.Sprintf("unsupported alter action: %T", action))
		}
	}

	if err := deps.ServiceWriter.UpdatePublishedRestService(svc); err != nil {
		return mdlerrors.NewBackend("alter published rest service", err)
	}

	if !deps.Quiet {
		fmt.Fprintf(deps.Output, "Altered published rest service %s.%s\n", s.Name.Module, s.Name.Name)
	}
	return nil
}

// ────────────────────────────────────────────────────────────
// Old ExecContext wrappers (delegate to Fn versions)
// ────────────────────────────────────────────────────────────

func listPublishedRestServices(ctx *ExecContext, moduleName string) error {
	deps := execContextToDeps(ctx)
	return listPublishedRestServicesFn(ctx, deps.Output, deps.Format, deps, moduleName)
}

func describePublishedRestService(ctx *ExecContext, name ast.QualifiedName) error {
	deps := execContextToDeps(ctx)
	return describePublishedRestServiceFn(ctx, deps.Output, deps, name)
}

func findPublishedRestService(ctx *ExecContext, moduleName, name string) (*model.PublishedRestService, error) {
	return findPublishedRestServiceFn(execContextToDeps(ctx), moduleName, name)
}




// ────────────────────────────────────────────────────────────
// Stateless helpers (no ctx/deps needed)
// ────────────────────────────────────────────────────────────

// astResourceDefToModel converts an AST PublishedRestResourceDef to the
// runtime model type used by the writer.
func astResourceDefToModel(def *ast.PublishedRestResourceDef) *model.PublishedRestResource {
	resource := &model.PublishedRestResource{Name: def.Name}
	for _, opDef := range def.Operations {
		resource.Operations = append(resource.Operations, &model.PublishedRestOperation{
			HTTPMethod: opDef.HTTPMethod,
			Path:       opDef.Path,
			Microflow:  opDef.Microflow.String(),
			Deprecated: opDef.Deprecated,
		})
	}
	return resource
}
