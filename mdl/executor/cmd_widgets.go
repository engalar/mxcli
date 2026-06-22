// SPDX-License-Identifier: Apache-2.0

// Package executor - Widget commands (SHOW WIDGETS, UPDATE WIDGETS, DESCRIBE WIDGET, SHOW INSTALLED WIDGETS)
//
// 全部用 MXGraph 重构: 小部件定义查询走 MXGraph Widget 节点（MPK/.def.json），
// 小部件实例查询走 MXGraph WidgetInstance 节点 + 附底面使用 catalog。
package executor

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
	"github.com/mendixlabs/mxcli/model"
)

// ensureGraphForWidgets 确保 MXGraph 已加载（优先使用快照，不阻塞等待完整重建）。
// 返回 graph 是否可用。
func ensureGraphForWidgets(ctx *ExecContext) *graphcatalog.ProjectGraph {
	if ctx.Graph != nil {
		return ctx.Graph
	}
	// 尝试从 backend 或快照文件加载
	if ctx.MprPath == "" {
		return nil
	}
	tryLoadGraphSnapshot(filepath.Dir(ctx.MprPath), ctx.Cache, &ctx.Graph)
	return ctx.Graph
}

// execShowWidgets handles the SHOW WIDGETS statement.
// Uses MXGraph WidgetInstance/Widget nodes; graph must be available.
func execShowWidgets(ctx *ExecContext, s *ast.ShowWidgetsStmt) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}
	pg := ensureGraphForWidgets(ctx)
	if pg == nil {
		return mdlerrors.NewUnsupported("SHOW WIDGETS requires MXGraph (not available)")
	}
	return execShowWidgetsFromGraph(ctx, pg, s)
}

// execShowWidgetsFromGraph 用 MXGraph 查询小部件实例
func execShowWidgetsFromGraph(ctx *ExecContext, pg *graphcatalog.ProjectGraph, s *ast.ShowWidgetsStmt) error {
	type row struct {
		name, widgetType, container, module string
	}

	var rows []row

	// 从每个页面获取 Widget 子节点（通过 HAS_WIDGET_INSTANCE 边）
	for _, page := range pg.Pages(s.InModule) {
		instances := pg.WidgetInstances(page.QualifiedName)
		for _, wi := range instances {
			if !matchWidgetFilter(wi.Name, wi.WidgetType, page.QualifiedName, page.Module, s) {
				continue
			}
			rows = append(rows, row{wi.Name, wi.WidgetType, page.QualifiedName, page.Module})
		}
		// 也遍历 HAS_WIDGET 的子节点（包括无样式实例）
		for _, w := range pg.Widgets(page.QualifiedName) {
			// 去重: 跳过已在 WidgetInstances 中出现过的
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
		fmt.Fprintln(ctx.Output, "No widgets found matching the criteria")
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

	fmt.Fprintf(ctx.Output, "\n%-30s %-40s %-40s %-20s\n",
		"NAME", "widget type", "container", "module")
	fmt.Fprintln(ctx.Output, strings.Repeat("-", 130))
	for _, r := range rows {
		fmt.Fprintf(ctx.Output, "%-30s %-40s %-40s %-20s\n",
			formatCell(r.name, 30), formatCell(r.widgetType, 40),
			formatCell(r.container, 40), formatCell(r.module, 20))
	}
	fmt.Fprintf(ctx.Output, "\n%d widget(s) found\n", len(rows))
	return nil
}

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

// execUpdateWidgets handles the UPDATE WIDGETS statement.
func execUpdateWidgets(ctx *ExecContext, s *ast.UpdateWidgetsStmt) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	if err := ensureCatalog(ctx, true); err != nil {
		return mdlerrors.NewBackend("build catalog", err)
	}

	widgets, err := findMatchingWidgets(ctx, s.Filters, s.InModule)
	if err != nil {
		return mdlerrors.NewBackend("find widgets", err)
	}

	if len(widgets) == 0 {
		fmt.Fprintln(ctx.Output, "No widgets found matching the criteria")
		return nil
	}

	containers := groupWidgetsByContainer(widgets)

	fmt.Fprintf(ctx.Output, "\nFound %d widget(s) in %d container(s) matching the criteria\n",
		len(widgets), len(containers))

	if s.DryRun {
		fmt.Fprintln(ctx.Output, "\n[dry run] The following changes would be made:")
	}

	totalUpdated := 0
	for containerID, widgetRefs := range containers {
		updated, err := updateWidgetsInContainer(ctx, containerID, widgetRefs, s.Assignments, s.DryRun)
		if err != nil {
			fmt.Fprintf(ctx.Output, "Warning: Failed to update widgets in %s: %v\n", containerID, err)
			continue
		}
		totalUpdated += updated
	}

	if s.DryRun {
		fmt.Fprintf(ctx.Output, "\n[dry run] Would update %d widget(s)\n", totalUpdated)
		fmt.Fprintln(ctx.Output, "\nRun without dry run to apply changes.")
	} else {
		fmt.Fprintf(ctx.Output, "\nUpdated %d widget(s)\n", totalUpdated)
		fmt.Fprintln(ctx.Output, "\nNote: Run 'refresh catalog full force' to update the catalog with changes.")
	}

	return nil
}

// widgetRef holds information about a widget to be updated.
type widgetRef struct {
	ID            string
	Name          string
	WidgetType    string
	ContainerID   string
	ContainerName string
	ContainerType string // "page" or "snippet"
}

// findMatchingWidgets is not available — catalog has been replaced by MXGraph.
func findMatchingWidgets(ctx *ExecContext, filters []ast.WidgetFilter, module string) ([]widgetRef, error) {
	return nil, mdlerrors.NewUnsupported("UPDATE WIDGETS requires MXGraph (not available)")
}

// groupWidgetsByContainer groups widgets by their container ID.
func groupWidgetsByContainer(widgets []widgetRef) map[string][]widgetRef {
	containers := make(map[string][]widgetRef)
	for _, w := range widgets {
		containers[w.ContainerID] = append(containers[w.ContainerID], w)
	}
	return containers
}

// updateWidgetsInContainer updates widgets within a single page or snippet
// using the PageMutator backend (no direct BSON manipulation).
func updateWidgetsInContainer(ctx *ExecContext, containerID string, widgetRefs []widgetRef, assignments []ast.WidgetPropertyAssignment, dryRun bool) (int, error) {
	if len(widgetRefs) == 0 {
		return 0, nil
	}

	containerName := widgetRefs[0].ContainerName

	mutator, err := ctx.PageMutationOperator.OpenPageForMutation(model.ID(containerID))
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
			fmt.Fprintf(ctx.Output, "  Warning: Widget %q not found in %s %s\n",
				ref.Name, mutator.ContainerType(), containerName)
			continue
		}
		for _, assignment := range assignments {
			if dryRun {
				fmt.Fprintf(ctx.Output, "  Would set '%s' = %v on %s (%s) in %s\n",
					assignment.PropertyPath, assignment.Value, ref.Name, ref.WidgetType, containerName)
			} else {
				if err := mutator.SetWidgetProperty(ref.Name, assignment.PropertyPath, assignment.Value); err != nil {
					fmt.Fprintf(ctx.Output, "  Warning: Failed to set '%s' on %s: %v\n",
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

// execShowInstalledWidgets uses MXGraph to list all installed widget definitions.
// 优先从 MXGraph Widget 节点获取；若 graph 不可用则附底面到 MPK 扫描。
func execShowInstalledWidgets(ctx *ExecContext, _ *ast.ShowInstalledWidgetsStmt) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}
	if ctx.MprPath == "" {
		return fmt.Errorf("SHOW INSTALLED WIDGETS requires a project connection (-p app.mpr)")
	}

	// MXGraph 路径
	if pg := ensureGraphForWidgets(ctx); pg != nil {
		return showInstalledWidgetsFromGraph(ctx, pg)
	}

	// 附底面：MPK 扫描
	return showInstalledWidgetsFromMPK(ctx)
}

// showInstalledWidgetsFromGraph 用 MXGraph Widget 节点列出所有已安装小部件定义
func showInstalledWidgetsFromGraph(ctx *ExecContext, pg *graphcatalog.ProjectGraph) error {
	defs := pg.DefinedWidgets()
	if len(defs) == 0 {
		fmt.Fprintln(ctx.Output, "No widget definitions found in graph (run 'refresh catalog full' first)")
		return nil
	}

	sort.Slice(defs, func(i, j int) bool {
		return strings.ToLower(defs[i].MDLName) < strings.ToLower(defs[j].MDLName)
	})

	fmt.Fprintf(ctx.Output, "\n%-30s %-60s %s\n", "Widget Name", "Widget ID", "Display Name")
	fmt.Fprintln(ctx.Output, strings.Repeat("-", 120))
	for _, d := range defs {
		fmt.Fprintf(ctx.Output, "%-30s %-60s %s\n",
			strings.ToLower(d.MDLName), d.ID, d.Name)
	}

	fmt.Fprintf(ctx.Output, "\n%d widget definition(s) found (source: mxgraph)\n", len(defs))
	fmt.Fprintf(ctx.Output, "\nMDL usage: PLUGGABLEWIDGET '<Widget ID>' instanceName (prop: val)\n")
	return nil
}

// showInstalledWidgetsFromMPK 附底面：扫描 widgets/*.mpk
func showInstalledWidgetsFromMPK(ctx *ExecContext) error {
	projectDir := filepath.Dir(ctx.MprPath)
	registry, err := NewWidgetRegistry()
	if err != nil {
		return fmt.Errorf("creating widget registry: %w", err)
	}
	if err := registry.SetProjectDir(projectDir); err != nil {
		return fmt.Errorf("scanning widgets/ directory: %w", err)
	}

	discovered := registry.MPKDiscovered()
	if len(discovered) == 0 {
		fmt.Fprintln(ctx.Output, "No widget packages found in widgets/")
		fmt.Fprintf(ctx.Output, "Copy a .mpk file to %s/widgets/ to install a widget.\n", projectDir)
		return nil
	}

	fmt.Fprintf(ctx.Output, "\n%-30s %-60s %s\n", "Widget Name", "Widget ID", "Display Name")
	fmt.Fprintln(ctx.Output, strings.Repeat("-", 120))

	names := make([]string, 0, len(discovered))
	for name := range discovered {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		w := discovered[name]
		fmt.Fprintf(ctx.Output, "%-30s %-60s %s\n",
			strings.ToLower(name), w.WidgetID, w.Name)
	}

	fmt.Fprintf(ctx.Output, "\n%d widget definition(s) found (source: mpk)\n", len(discovered))
	fmt.Fprintf(ctx.Output, "\nMDL usage: PLUGGABLEWIDGET '<Widget ID>' instanceName (prop: val)\n")
	return nil
}

// describeWidget handles DESCRIBE WIDGET Module.WidgetName
// 查找顺序: MXGraph Widget 节点 → widget registry (MPK 扫描) → catalog 实例。
func describeWidget(ctx *ExecContext, name ast.QualifiedName) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}

	describeQN := name.Name
	if name.Module != "" {
		describeQN = name.Module + "." + name.Name
	}

	// 1) MXGraph Widget 定义节点
	if pg := ensureGraphForWidgets(ctx); pg != nil {
		if def := pg.FindDefinedWidget(describeQN); def != nil {
			fmt.Fprintf(ctx.Output, "Widget Definition (mxgraph): %s\n", def.ID)
			fmt.Fprintf(ctx.Output, "  Display Name: %s\n", def.Name)
			fmt.Fprintf(ctx.Output, "  MDL Short Name: %s\n", strings.ToLower(def.MDLName))
			fmt.Fprintf(ctx.Output, "  Widget Kind: %s\n", def.WidgetKind)
			fmt.Fprintf(ctx.Output, "  Source: %s\n", def.Source)
			return nil
		}
	}

	// 2) widget registry (MPK 扫描)——无需 graph 快照
	{
		projectDir := filepath.Dir(ctx.MprPath)
		reg, err := NewWidgetRegistry()
		if err == nil {
			if mpkErr := reg.SetProjectDir(projectDir); mpkErr == nil {
				// 尝试按 MDLName 查找
				upper := strings.ToUpper(describeQN)
				if def, ok := reg.Get(upper); ok {
					fmt.Fprintf(ctx.Output, "Widget Definition (mpk): %s\n", def.WidgetID)
					fmt.Fprintf(ctx.Output, "  Display Name: %s\n", def.MDLName)
					fmt.Fprintf(ctx.Output, "  Widget Kind: %s\n", def.WidgetKind)
					if def.TemplateFile != "" {
						fmt.Fprintf(ctx.Output, "  Template: %s\n", def.TemplateFile)
					}
					return nil
				}
				// 尝试按完整 WidgetID 查找
				if def, ok := reg.GetByWidgetID(describeQN); ok {
					fmt.Fprintf(ctx.Output, "Widget Definition (mpk): %s\n", def.WidgetID)
					fmt.Fprintf(ctx.Output, "  Display Name: %s\n", def.MDLName)
					fmt.Fprintf(ctx.Output, "  Widget Kind: %s\n", def.WidgetKind)
					if def.TemplateFile != "" {
						fmt.Fprintf(ctx.Output, "  Template: %s\n", def.TemplateFile)
					}
					return nil
				}
				// 尝试反向查找：从 mpkDiscovered 中找 display name 或 id 匹配
				for mdlName, dw := range reg.MPKDiscovered() {
					lowerMdl := strings.ToLower(mdlName)
					lowerQN := strings.ToLower(describeQN)
					if lowerMdl == lowerQN || strings.Contains(strings.ToLower(dw.WidgetID), lowerQN) || strings.Contains(strings.ToLower(dw.Name), lowerQN) {
						fmt.Fprintf(ctx.Output, "Widget (mpk discovered): %s\n", dw.WidgetID)
						fmt.Fprintf(ctx.Output, "  Display Name: %s\n", dw.Name)
						fmt.Fprintf(ctx.Output, "  MDL Short Name: %s\n", strings.ToLower(mdlName))
						return nil
					}
				}
			}
		}
	}

	return mdlerrors.NewNotFound("widget", name.String())
}

// formatCell formats a cell value for display, truncating if needed.
func formatCell(val any, maxLen int) string {
	s := fmt.Sprintf("%v", val)
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}
