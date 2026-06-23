// SPDX-License-Identifier: Apache-2.0

// Package executor - Widget commands (SHOW WIDGETS, UPDATE WIDGETS, DESCRIBE WIDGET, SHOW INSTALLED WIDGETS)
//
// 全部用 MXGraph 重构: 小部件定义查询走 MXGraph Widget 节点（MPK/.def.json），
// 小部件实例查询走 MXGraph WidgetInstance 节点 + 附底面使用 catalog。
package executor

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
	"github.com/mendixlabs/mxcli/model"
)

// ────────────────────────────────────────────────────────────
// Fn (HandlerDeps) versions
// ────────────────────────────────────────────────────────────

func ensureGraphForWidgetsFn(deps *HandlerDeps) *graphcatalog.ProjectGraph {
	if deps.Graph != nil {
		return deps.Graph
	}
	if deps.MprPath == "" {
		return nil
	}
	tryLoadGraphSnapshot(filepath.Dir(deps.MprPath), nil, &deps.Graph)
	return deps.Graph
}

func execShowWidgetsFn(ctx context.Context, s *ast.ShowWidgetsStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}
	pg := ensureGraphForWidgetsFn(deps)
	if pg == nil {
		return mdlerrors.NewUnsupported("SHOW WIDGETS requires MXGraph (not available)")
	}
	return execShowWidgetsFromGraphFn(deps.Output, pg, s)
}

func execShowWidgetsFromGraphFn(output io.Writer, pg *graphcatalog.ProjectGraph, s *ast.ShowWidgetsStmt) error {
	type row struct {
		name, widgetType, container, module string
	}

	var rows []row

	for _, page := range pg.Pages(s.InModule) {
		instances := pg.WidgetInstances(page.QualifiedName)
		for _, wi := range instances {
			if !matchWidgetFilter(wi.Name, wi.WidgetType, page.QualifiedName, page.Module, s) {
				continue
			}
			rows = append(rows, row{wi.Name, wi.WidgetType, page.QualifiedName, page.Module})
		}
		for _, w := range pg.Widgets(page.QualifiedName) {
			dup := false
			for _, wi := range rows {
				if wi.name == w.Name && wi.container == page.QualifiedName {
					dup = true
					break
				}
			}
			if dup {
				continue
			}
			if !matchWidgetFilter(w.Name, w.WidgetType, page.QualifiedName, page.Module, s) {
				continue
			}
			rows = append(rows, row{w.Name, w.WidgetType, page.QualifiedName, page.Module})
		}
	}

	if len(rows) == 0 {
		fmt.Fprintln(output, "No widgets found matching the criteria")
		return nil
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].module != rows[j].module {
			return rows[i].module < rows[j].module
		}
		if rows[i].container != rows[j].container {
			return rows[i].container < rows[j].container
		}
		return rows[i].name < rows[j].name
	})

	fmt.Fprintf(output, "\n%-30s %-40s %-40s %-20s\n",
		"NAME", "widget type", "container", "module")
	fmt.Fprintln(output, strings.Repeat("-", 130))
	for _, r := range rows {
		fmt.Fprintf(output, "%-30s %-40s %-40s %-20s\n",
			formatCell(r.name, 30), formatCell(r.widgetType, 40),
			formatCell(r.container, 40), formatCell(r.module, 20))
	}
	fmt.Fprintf(output, "\n%d widget(s) found\n", len(rows))
	return nil
}

func execUpdateWidgetsFn(ctx context.Context, s *ast.UpdateWidgetsStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}
	if deps.PageMutationOperator == nil {
		return mdlerrors.NewNotConnectedWrite()
	}

	widgets, err := findMatchingWidgetsFn(s.Filters, s.InModule)
	if err != nil {
		return mdlerrors.NewBackend("find widgets", err)
	}

	if len(widgets) == 0 {
		fmt.Fprintln(deps.Output, "No widgets found matching the criteria")
		return nil
	}

	containers := groupWidgetsByContainer(widgets)

	fmt.Fprintf(deps.Output, "\nFound %d widget(s) in %d container(s) matching the criteria\n",
		len(widgets), len(containers))

	if s.DryRun {
		fmt.Fprintln(deps.Output, "\n[dry run] The following changes would be made:")
	}

	totalUpdated := 0
	for containerID, widgetRefs := range containers {
		updated, err := updateWidgetsInContainerFn(deps, containerID, widgetRefs, s.Assignments, s.DryRun)
		if err != nil {
			fmt.Fprintf(deps.Output, "Warning: Failed to update widgets in %s: %v\n", containerID, err)
			continue
		}
		totalUpdated += updated
	}

	if s.DryRun {
		fmt.Fprintf(deps.Output, "\n[dry run] Would update %d widget(s)\n", totalUpdated)
		fmt.Fprintln(deps.Output, "\nRun without dry run to apply changes.")
	} else {
		fmt.Fprintf(deps.Output, "\nUpdated %d widget(s)\n", totalUpdated)
		fmt.Fprintln(deps.Output, "\nNote: Run 'refresh catalog full force' to update the catalog with changes.")
	}

	return nil
}

func findMatchingWidgetsFn(filters []ast.WidgetFilter, module string) ([]widgetRef, error) {
	return nil, mdlerrors.NewUnsupported("UPDATE WIDGETS requires MXGraph (not available)")
}

func updateWidgetsInContainerFn(deps *HandlerDeps, containerID string, widgetRefs []widgetRef, assignments []ast.WidgetPropertyAssignment, dryRun bool) (int, error) {
	if len(widgetRefs) == 0 {
		return 0, nil
	}

	containerName := widgetRefs[0].ContainerName

	mutator, err := deps.PageMutationOperator.OpenPageForMutation(model.ID(containerID))
	if err != nil {
		return 0, mdlerrors.NewBackend(fmt.Sprintf("open %s for mutation", containerName), err)
	}
	if mutator == nil {
		return 0, mdlerrors.NewBackend(fmt.Sprintf("open %s for mutation", containerName),
			fmt.Errorf("backend returned nil mutator for %s", containerID))
	}

	updated := 0
	for _, ref := range widgetRefs {
		if !mutator.FindWidget(ref.Name) {
			fmt.Fprintf(deps.Output, "  Warning: Widget %q not found in %s %s\n",
				ref.Name, mutator.ContainerType(), containerName)
			continue
		}
		for _, assignment := range assignments {
			if dryRun {
				fmt.Fprintf(deps.Output, "  Would set '%s' = %v on %s (%s) in %s\n",
					assignment.PropertyPath, assignment.Value, ref.Name, ref.WidgetType, containerName)
			} else {
				if err := mutator.SetWidgetProperty(ref.Name, assignment.PropertyPath, assignment.Value); err != nil {
					fmt.Fprintf(deps.Output, "  Warning: Failed to set '%s' on %s: %v\n",
						assignment.PropertyPath, ref.Name, err)
				}
			}
		}
		updated++
	}

	if !dryRun && updated > 0 {
		if err := mutator.Save(); err != nil {
			return updated, mdlerrors.NewBackend(fmt.Sprintf("save %s", containerName), err)
		}
	}

	return updated, nil
}

func execShowInstalledWidgetsFn(ctx context.Context, s *ast.ShowInstalledWidgetsStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}
	if deps.MprPath == "" {
		return fmt.Errorf("SHOW INSTALLED WIDGETS requires a project connection (-p app.mpr)")
	}

	if pg := ensureGraphForWidgetsFn(deps); pg != nil {
		return showInstalledWidgetsFromGraphFn(deps.Output, pg)
	}

	return showInstalledWidgetsFromMPKFn(deps.Output, deps.MprPath)
}

func showInstalledWidgetsFromGraphFn(output io.Writer, pg *graphcatalog.ProjectGraph) error {
	defs := pg.DefinedWidgets()
	if len(defs) == 0 {
		fmt.Fprintln(output, "No widget definitions found in graph (run 'refresh catalog full' first)")
		return nil
	}

	sort.Slice(defs, func(i, j int) bool {
		return strings.ToLower(defs[i].MDLName) < strings.ToLower(defs[j].MDLName)
	})

	fmt.Fprintf(output, "\n%-30s %-60s %s\n", "Widget Name", "Widget ID", "Display Name")
	fmt.Fprintln(output, strings.Repeat("-", 120))
	for _, d := range defs {
		fmt.Fprintf(output, "%-30s %-60s %s\n",
			strings.ToLower(d.MDLName), d.ID, d.Name)
	}

	fmt.Fprintf(output, "\n%d widget definition(s) found (source: mxgraph)\n", len(defs))
	fmt.Fprintf(output, "\nMDL usage: PLUGGABLEWIDGET '<Widget ID>' instanceName (prop: val)\n")
	return nil
}

func showInstalledWidgetsFromMPKFn(output io.Writer, mprPath string) error {
	projectDir := filepath.Dir(mprPath)
	registry, err := NewWidgetRegistry()
	if err != nil {
		return fmt.Errorf("creating widget registry: %w", err)
	}
	if err := registry.SetProjectDir(projectDir); err != nil {
		return fmt.Errorf("scanning widgets/ directory: %w", err)
	}

	discovered := registry.MPKDiscovered()
	if len(discovered) == 0 {
		fmt.Fprintln(output, "No widget packages found in widgets/")
		fmt.Fprintf(output, "Copy a .mpk file to %s/widgets/ to install a widget.\n", projectDir)
		return nil
	}

	fmt.Fprintf(output, "\n%-30s %-60s %s\n", "Widget Name", "Widget ID", "Display Name")
	fmt.Fprintln(output, strings.Repeat("-", 120))

	names := make([]string, 0, len(discovered))
	for name := range discovered {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		w := discovered[name]
		fmt.Fprintf(output, "%-30s %-60s %s\n",
			strings.ToLower(name), w.WidgetID, w.Name)
	}

	fmt.Fprintf(output, "\n%d widget definition(s) found (source: mpk)\n", len(discovered))
	fmt.Fprintf(output, "\nMDL usage: PLUGGABLEWIDGET '<Widget ID>' instanceName (prop: val)\n")
	return nil
}

func describeWidgetFn(ctx context.Context, output io.Writer, deps *HandlerDeps, name ast.QualifiedName) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	describeQN := name.Name
	if name.Module != "" {
		describeQN = name.Module + "." + name.Name
	}

	if pg := ensureGraphForWidgetsFn(deps); pg != nil {
		if def := pg.FindDefinedWidget(describeQN); def != nil {
			fmt.Fprintf(output, "Widget Definition (mxgraph): %s\n", def.ID)
			fmt.Fprintf(output, "  Display Name: %s\n", def.Name)
			fmt.Fprintf(output, "  MDL Short Name: %s\n", strings.ToLower(def.MDLName))
			fmt.Fprintf(output, "  Widget Kind: %s\n", def.WidgetKind)
			fmt.Fprintf(output, "  Source: %s\n", def.Source)
			return nil
		}
	}

	{
		projectDir := filepath.Dir(deps.MprPath)
		reg, err := NewWidgetRegistry()
		if err == nil {
			if mpkErr := reg.SetProjectDir(projectDir); mpkErr == nil {
				upper := strings.ToUpper(describeQN)
				if def, ok := reg.Get(upper); ok {
					fmt.Fprintf(output, "Widget Definition (mpk): %s\n", def.WidgetID)
					fmt.Fprintf(output, "  Display Name: %s\n", def.MDLName)
					fmt.Fprintf(output, "  Widget Kind: %s\n", def.WidgetKind)
					if def.TemplateFile != "" {
						fmt.Fprintf(output, "  Template: %s\n", def.TemplateFile)
					}
					return nil
				}
				if def, ok := reg.GetByWidgetID(describeQN); ok {
					fmt.Fprintf(output, "Widget Definition (mpk): %s\n", def.WidgetID)
					fmt.Fprintf(output, "  Display Name: %s\n", def.MDLName)
					fmt.Fprintf(output, "  Widget Kind: %s\n", def.WidgetKind)
					if def.TemplateFile != "" {
						fmt.Fprintf(output, "  Template: %s\n", def.TemplateFile)
					}
					return nil
				}
				for mdlName, dw := range reg.MPKDiscovered() {
					lowerMdl := strings.ToLower(mdlName)
					lowerQN := strings.ToLower(describeQN)
					if lowerMdl == lowerQN || strings.Contains(strings.ToLower(dw.WidgetID), lowerQN) || strings.Contains(strings.ToLower(dw.Name), lowerQN) {
						fmt.Fprintf(output, "Widget (mpk discovered): %s\n", dw.WidgetID)
						fmt.Fprintf(output, "  Display Name: %s\n", dw.Name)
						fmt.Fprintf(output, "  MDL Short Name: %s\n", strings.ToLower(mdlName))
						return nil
					}
				}
			}
		}
	}

	return mdlerrors.NewNotFound("widget", name.String())
}

// ────────────────────────────────────────────────────────────
// Old ExecContext wrappers (delegate to Fn versions)
// ────────────────────────────────────────────────────────────

func ensureGraphForWidgets(ctx *ExecContext) *graphcatalog.ProjectGraph {
	if ctx.Graph != nil {
		return ctx.Graph
	}
	if ctx.MprPath == "" {
		return nil
	}
	tryLoadGraphSnapshot(filepath.Dir(ctx.MprPath), ctx.Cache, &ctx.Graph)
	return ctx.Graph
}


func execShowWidgetsFromGraph(ctx *ExecContext, pg *graphcatalog.ProjectGraph, s *ast.ShowWidgetsStmt) error {
	return execShowWidgetsFromGraphFn(ctx.Output, pg, s)
}


func findMatchingWidgets(ctx *ExecContext, filters []ast.WidgetFilter, module string) ([]widgetRef, error) {
	return findMatchingWidgetsFn(filters, module)
}

func updateWidgetsInContainer(ctx *ExecContext, containerID string, widgetRefs []widgetRef, assignments []ast.WidgetPropertyAssignment, dryRun bool) (int, error) {
	return updateWidgetsInContainerFn(execContextToDeps(ctx), containerID, widgetRefs, assignments, dryRun)
}


func showInstalledWidgetsFromGraph(ctx *ExecContext, pg *graphcatalog.ProjectGraph) error {
	return showInstalledWidgetsFromGraphFn(ctx.Output, pg)
}

func showInstalledWidgetsFromMPK(ctx *ExecContext) error {
	return showInstalledWidgetsFromMPKFn(ctx.Output, ctx.MprPath)
}

func describeWidget(ctx *ExecContext, name ast.QualifiedName) error {
	deps := execContextToDeps(ctx)
	return describeWidgetFn(ctx, deps.Output, deps, name)
}

// ────────────────────────────────────────────────────────────
// Stateless helpers (no ctx/deps needed)
// ────────────────────────────────────────────────────────────

// matchWidgetFilter 判断一行是否匹配 SHOW WIDGETS 过滤条件
func matchWidgetFilter(name, widgetType, container, module string, s *ast.ShowWidgetsStmt) bool {
	if s.InModule != "" && !strings.EqualFold(module, s.InModule) {
		return false
	}
	for _, f := range s.Filters {
		val := ""
		switch strings.ToLower(f.Field) {
		case "name":
			val = name
		case "widgettype":
			val = widgetType
		case "container":
			val = container
		case "module":
			val = module
		default:
			continue
		}
		if strings.EqualFold(f.Operator, "like") {
			likeVal := strings.ReplaceAll(strings.ToLower(f.Value), "%", "")
			if !strings.Contains(strings.ToLower(val), likeVal) {
				return false
			}
		} else if !strings.EqualFold(val, f.Value) {
			return false
		}
	}
	return true
}

// widgetRef holds information about a widget to be updated.
type widgetRef struct {
	ID            string
	Name          string
	WidgetType    string
	ContainerID   string
	ContainerName string
	ContainerType string
}

// groupWidgetsByContainer groups widgets by their container ID.
func groupWidgetsByContainer(widgets []widgetRef) map[string][]widgetRef {
	containers := make(map[string][]widgetRef)
	for _, w := range widgets {
		containers[w.ContainerID] = append(containers[w.ContainerID], w)
	}
	return containers
}

// mapWidgetFilterField maps user-facing field names to catalog column names.
func mapWidgetFilterField(field string) string {
	switch strings.ToLower(field) {
	case "widgettype":
		return "WidgetType"
	case "name":
		return "Name"
	case "container":
		return "ContainerQualifiedName"
	case "module":
		return "ModuleName"
	default:
		return field
	}
}

// formatCell formats a cell value for display, truncating if needed.
func formatCell(val any, maxLen int) string {
	s := fmt.Sprintf("%v", val)
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}
