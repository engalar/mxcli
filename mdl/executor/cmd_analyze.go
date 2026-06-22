// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
	"github.com/mendixlabs/mxcli/modelsdk"
)

// Ensure time import triggers no build error
var _ = fmt.Sprintf

func perfStart(ctx *ExecContext, name string) func() {
	if ctx.Perf == nil {
		return func() {}
	}
	return ctx.Perf.Begin(name)
}

func perfReport(ctx *ExecContext) {
	if ctx.Perf != nil {
		ctx.Perf.Report()
	}
}

// AnalyzeNavigation profiles and menu tree.
func AnalyzeNavigation(ctx *ExecContext) error {
	defer perfReport(ctx)
	if err := ensureGraph(ctx); err != nil {
		return err
	}

	nav := ctx.Graph.NavigationProfiles()
	if len(nav) == 0 {
		fmt.Fprintln(ctx.Output, "No navigation profiles found.")
		return nil
	}

	for _, p := range nav {
		fmt.Fprintf(ctx.Output, "\n=== Profile: %s (%s)", p.Name, p.Kind)
		if p.IsNative {
			fmt.Fprint(ctx.Output, " native")
		}
		fmt.Fprintln(ctx.Output)
		if p.LoginPage != "" {
			fmt.Fprintf(ctx.Output, "  Login: %s\n", p.LoginPage)
		}
		if p.NotFoundPage != "" {
			fmt.Fprintf(ctx.Output, "  Not Found: %s\n", p.NotFoundPage)
		}

		if hp := ctx.Graph.NavigationHomePage(p.Name); hp != nil {
			if hp.Page != "" {
				fmt.Fprintf(ctx.Output, "  Home: page %s\n", hp.Page)
			} else if hp.Microflow != "" {
				fmt.Fprintf(ctx.Output, "  Home: microflow %s\n", hp.Microflow)
			}
		}

		tree := ctx.Graph.NavigationMenuTree(p.Name)
		if tree != nil && len(tree.MenuItems) > 0 {
			fmt.Fprintln(ctx.Output, "  Menu:")
			printMenuTreeAnalyze(ctx, tree.MenuItems, 4)
		}
	}
	return nil
}

func printMenuTreeAnalyze(ctx *ExecContext, items []graphcatalog.NavigationMenuItemTree, indent int) {
	prefix := strings.Repeat(" ", indent)
	for _, item := range items {
		target := ""
		if item.Item.Page != "" {
			target = fmt.Sprintf(" -> page %s", item.Item.Page)
		} else if item.Item.Microflow != "" {
			target = fmt.Sprintf(" -> microflow %s", item.Item.Microflow)
		}
		fmt.Fprintf(ctx.Output, "%s- %s%s\n", prefix, item.Item.Caption, target)
		if len(item.Children) > 0 {
			printMenuTreeAnalyze(ctx, item.Children, indent+2)
		}
	}
}

// AnalyzePage data container hierarchy, context variables, widget appearance.
func AnalyzePage(ctx *ExecContext, pageQN string) error {
	defer perfReport(ctx)
	if err := ensureGraph(ctx); err != nil {
		return err
	}
	if pageQN == "" {
		return mdlerrors.NewValidation("page qualified name required")
	}

	dcs := ctx.Graph.PageDataContainerTree(pageQN)
	fmt.Fprintf(ctx.Output, "\nPage: %s\n", pageQN)

	roles := ctx.Graph.PageAllowedRoles(pageQN)
	if len(roles) > 0 {
		fmt.Fprintf(ctx.Output, "Allowed Roles: %s\n", strings.Join(roles, ", "))
	}

	navRefs := findPageNavRefsAnalyze(ctx, pageQN)
	if len(navRefs) > 0 {
		fmt.Fprintf(ctx.Output, "Navigation: %s\n", strings.Join(navRefs, ", "))
	} else {
		fmt.Fprintln(ctx.Output, "Navigation: (none — orphan or microflow-only)")
	}

	if len(dcs) == 0 {
		fmt.Fprintln(ctx.Output, "Data Containers: (none)")
		return nil
	}

	fmt.Fprintln(ctx.Output, "\nData Containers:")
	printDataContainersAnalyze(ctx, dcs, 0)
	return nil
}

func printDataContainersAnalyze(ctx *ExecContext, dcs []graphcatalog.DataContainerNode, depth int) {
	for _, dc := range dcs {
		prefix := strings.Repeat("  ", depth)
		dsInfo := ""
		if dc.DataSourceType != "none" {
			dsInfo = fmt.Sprintf(" (%s → %s)", dc.DataSourceType, dc.TargetEntity)
		}
		fmt.Fprintf(ctx.Output, "%s├── %s \"%s\"%s  [depth=%d]\n", prefix, dc.WidgetType, dc.WidgetName, dsInfo, dc.Depth)

		if len(dc.ContextVariables) > 0 {
			var varStrs []string
			for _, v := range dc.ContextVariables {
				varStrs = append(varStrs, fmt.Sprintf("%s (%s)", v.Name, v.EntityType))
			}
			fmt.Fprintf(ctx.Output, "%s│     Context: %s\n", prefix, strings.Join(varStrs, ", "))
		}

		if dc.HasSelection && dc.SelectionName != "" {
			fmt.Fprintf(ctx.Output, "%s│     Selection: $%s (%s)\n", prefix, dc.SelectionName, dc.TargetEntity)
		}

		if len(dc.ChildWidgets) > 0 {
			fmt.Fprintf(ctx.Output, "%s│     Widgets:\n", prefix)
			for _, cw := range dc.ChildWidgets {
				appearance := ""
				if cw.Class != "" {
					appearance += fmt.Sprintf(" class=\"%s\"", cw.Class)
				}
				if cw.Style != "" {
					appearance += fmt.Sprintf(" style=\"%s\"", cw.Style)
				}
				condInfo := ""
				if cw.ConditionalVisibility != "" {
					condInfo = fmt.Sprintf(", visible when \"%s\"", cw.ConditionalVisibility)
				}
				if cw.ConditionalEditability != "" {
					condInfo += fmt.Sprintf(", editable when \"%s\"", cw.ConditionalEditability)
				}
				attrInfo := ""
				if cw.Attribute != "" {
					attrInfo = fmt.Sprintf(" attr=%s", cw.Attribute)
				}
				captionInfo := ""
				if cw.Caption != "" {
					captionInfo = fmt.Sprintf(" caption=\"%s\"", cw.Caption)
				}
				fmt.Fprintf(ctx.Output, "%s│       └─ %s [%s]%s%s%s%s\n",
					prefix, cw.Name, cw.WidgetType, attrInfo, captionInfo, appearance, condInfo)
			}
		}
	}
}

// AnalyzeEntity shows entity data flow: pages, creators, retrievers.
func AnalyzeEntity(ctx *ExecContext, entityQN string) error {
	defer perfReport(ctx)
	if err := ensureGraph(ctx); err != nil {
		return err
	}
	if entityQN == "" {
		return mdlerrors.NewValidation("entity qualified name required")
	}

	flow := ctx.Graph.EntityDataFlow(entityQN)
	if flow == nil {
		fmt.Fprintf(ctx.Output, "Entity %s not found.\n", entityQN)
		return nil
	}

	fmt.Fprintf(ctx.Output, "\nEntity: %s\n", entityQN)

	if len(flow.Pages) > 0 {
		fmt.Fprintln(ctx.Output, "Referenced by pages:")
		for _, p := range flow.Pages {
			fmt.Fprintf(ctx.Output, "  - page %s (via %s)\n", p.Page, p.DataSourceType)
			if len(p.AllowedRoles) > 0 {
				fmt.Fprintf(ctx.Output, "    Roles: %s\n", strings.Join(p.AllowedRoles, ", "))
			}
			if len(p.NavigationRefs) > 0 {
				fmt.Fprintf(ctx.Output, "    Navigation: %s\n", strings.Join(p.NavigationRefs, ", "))
			}
		}
	} else {
		fmt.Fprintln(ctx.Output, "Pages: (none)")
	}

	if len(flow.Creators) > 0 {
		sort.Strings(flow.Creators)
		fmt.Fprintf(ctx.Output, "Created by microflows: %s\n", strings.Join(flow.Creators, ", "))
	}
	if len(flow.Retrievers) > 0 {
		sort.Strings(flow.Retrievers)
		fmt.Fprintf(ctx.Output, "Retrieved by microflows: %s\n", strings.Join(flow.Retrievers, ", "))
	}
	return nil
}

// AnalyzeOrphans lists pages with no navigation or microflow references.
func AnalyzeOrphans(ctx *ExecContext) error {
	defer perfReport(ctx)
	if err := ensureGraph(ctx); err != nil {
		return err
	}

	defer perfStart(ctx, "orphans.pages_refd")()
	pages := ctx.Graph.PagesReferencedByNavigation()
	if len(pages) > 0 {
		fmt.Fprintln(ctx.Output, "\nPages referenced by navigation:")
		for _, p := range pages {
			fmt.Fprintf(ctx.Output, "  ✅ %s\n", p)
		}
	}

	defer perfStart(ctx, "orphans.find")()
	orphans := ctx.Graph.OrphanPages()
	if len(orphans) > 0 {
		fmt.Fprintf(ctx.Output, "\nOrphan pages (no navigation/microflow ref):\n")
		for _, p := range orphans {
			fmt.Fprintf(ctx.Output, "  ❌ %s\n", p)
		}
	} else {
		fmt.Fprintln(ctx.Output, "\nNo orphan pages found.")
	}
	return nil
}

// findPageNavRefsAnalyze finds navigation profiles referencing a page.
func findPageNavRefsAnalyze(ctx *ExecContext, pageQN string) []string {
	defer perfStart(ctx, "page.nav_refs")()
	profiles := ctx.Graph.NavigationProfiles()
	var refs []string
	for _, p := range profiles {
		tree := ctx.Graph.NavigationMenuTree(p.Name)
		if tree == nil {
			continue
		}
		if hasPageRef(tree.MenuItems, pageQN) {
			refs = append(refs, p.Name)
		}
	}
	return refs
}

func hasPageRef(items []graphcatalog.NavigationMenuItemTree, pageQN string) bool {
	for _, item := range items {
		if item.Item.Page == pageQN || strings.HasSuffix(item.Item.Page, "."+pageQN) {
			return true
		}
		if len(item.Children) > 0 && hasPageRef(item.Children, pageQN) {
			return true
		}
	}
	return false
}

// AnalyzeFlow performs transitive data flow analysis from an entry point.
func AnalyzeFlow(ctx *ExecContext, entryKind, entryName string, maxDepth int) error {
	defer perfReport(ctx)
	if err := ensureGraph(ctx); err != nil {
		return err
	}

	if entryKind == "" {
		defer perfStart(ctx, "flow.all_entry_points")()
		eps := ctx.Graph.AllEntryPoints()
		fmt.Fprintf(ctx.Output, "\nApplication Entry Points (%d):\n", len(eps))
		for _, ep := range eps {
			fmt.Fprintf(ctx.Output, "  %s: %s\n", ep.Kind, ep.Description)
		}

		fmt.Fprintln(ctx.Output, "\nAll Reachable Pages (from all entry points):")
		allPages := ctx.Graph.AllReachablePages()
		orphans := ctx.Graph.OrphanPages()
		orphanSet := map[string]bool{}
		for _, p := range orphans {
			orphanSet[p] = true
		}

		for page, sources := range allPages {
			status := "✅"
			if orphanSet[page] {
				status = "⚠️"
			}
			fmt.Fprintf(ctx.Output, "  %s %s (via %s)\n", status, page, strings.Join(sources, ", "))
		}

		defer perfStart(ctx, "flow.unreachable")()
		pages := ctx.Graph.Pages("")
		for _, p := range pages {
			if _, ok := allPages[p.QualifiedName]; !ok {
				fmt.Fprintf(ctx.Output, "  ❌ %s (unreachable)\n", p.QualifiedName)
			}
		}
		return nil
	}

	// entryKind specified but no entryName → list all entry points of that kind
	if entryName == "" {
		defer perfStart(ctx, "flow.list_kind")()
		eps := ctx.Graph.AllEntryPoints()
		var filtered []graphcatalog.EntryPoint
		for _, ep := range eps {
			if ep.Kind == entryKind {
				filtered = append(filtered, ep)
			}
		}
		if len(filtered) == 0 {
			fmt.Fprintf(ctx.Output, "No %s entry points found.\n", entryKind)
			return nil
		}
		fmt.Fprintf(ctx.Output, "\n%s Entry Points (%d):\n", entryKind, len(filtered))
		for _, ep := range filtered {
			fmt.Fprintf(ctx.Output, "  %s\n", ep.Description)
		}
		return nil
	}

	defer perfStart(ctx, "flow.chains")()

	// Try exact match first; fall back to partial (case-insensitive contains)
	chains := ctx.Graph.ReachablePagesFromEntry(entryKind, entryName)
	if len(chains) == 0 {
		// Partial match: find entry points where Name contains entryName
		eps := ctx.Graph.AllEntryPoints()
		entryNameLower := strings.ToLower(entryName)
		for _, ep := range eps {
			if ep.Kind == entryKind && strings.Contains(strings.ToLower(ep.Description), entryNameLower) {
				chains = ctx.Graph.ReachablePagesFromEntry(entryKind, ep.Name)
				if len(chains) > 0 {
					entryName = ep.Name
					break
				}
			}
		}
	}

	if len(chains) == 0 {
		fmt.Fprintf(ctx.Output, "No reachable pages from %s:%s\n", entryKind, entryName)
		return nil
	}

	fmt.Fprintf(ctx.Output, "\nData Flow: %s → %s\n", entryKind, entryName)
	seen := map[string]bool{}
	for _, chain := range chains {
		if seen[chain.TerminalPage] {
			continue
		}
		seen[chain.TerminalPage] = true

		path := entryName
		for _, step := range chain.Steps {
			displayName := step.NodeName
			if step.NodeType == "NavigationMenuItem" {
				caption := ""
				if n := ctx.Graph.MxGraph().GetNode(mxgraph.NodeID(step.NodeName)); n != nil {
					if c, ok := n.Props["Caption"].(string); ok && c != "" {
						caption = c
					}
				}
				if caption != "" {
					displayName = fmt.Sprintf("「%s」", caption)
				}
			}
			arrow := "→"
			switch step.EdgeType {
			case "TARGETS_PAGE", "SHOWS_PAGE":
				arrow = "→📄"
			case "TARGETS_MICROFLOW", "CALLS", "CALLS_MICROFLOW", "HAS_DATASOURCE_MICROFLOW":
				arrow = "→⚡"
			case "HAS_DATA_CONTAINER":
				arrow = "→🗂️"
			case "HAS_MENU_ITEM":
				arrow = "→📁"
			}
			path += fmt.Sprintf(" %s %s", arrow, displayName)
		}
		fmt.Fprintf(ctx.Output, "  📍 %s\n", path)
	}
	return nil
}

// ────────────────────────────────────────────────────────────
// Phase 3d-5f: Fn (HandlerDeps) versions of analyzer functions
// ────────────────────────────────────────────────────────────

func perfStartFn(perf *PerfTimer, name string) func() {
	if perf == nil {
		return func() {}
	}
	return perf.Begin(name)
}

func perfReportFn(perf *PerfTimer) {
	if perf != nil {
		perf.Report()
	}
}

func ensureGraphFn(output io.Writer, quiet bool, mprPath string, graph **graphcatalog.ProjectGraph) error {
	if *graph != nil {
		return nil
	}
	if mprPath == "" {
		return mdlerrors.NewNotConnected()
	}
	return buildGraphFromDeps(output, quiet, mprPath, graph)
}

func buildGraphFromDeps(output io.Writer, quiet bool, mprPath string, graph **graphcatalog.ProjectGraph) error {
	projectDir := filepath.Dir(mprPath)
	snapPath := graphcatalog.SnapshotPath(projectDir)
	deltaPath := graphcatalog.DeltaPath(projectDir)

	if g, err := mxgraph.RestoreFromSnapshot(snapPath, deltaPath); err != nil {
		return mdlerrors.NewBackend("restore graph cache", err)
	} else if g != nil {
		mgr := mxgraph.NewIndexManagerFromGraph(g)
		pg := graphcatalog.NewProjectGraph(mgr)
		*graph = pg
		if !quiet {
			fmt.Fprintf(output, "Graph restored: %d nodes, %d edges (from cache)\n",
				len(g.AllNodes()), len(g.AllEdges()))
		}
		return nil
	}

	m, err := modelsdk.Open(mprPath)
	if err != nil {
		return mdlerrors.NewBackend("open project for graph build", err)
	}
	defer m.Close()

	pg, err := buildGraphFromModel(m, projectDir, snapPath, deltaPath)
	if err != nil {
		return err
	}

	*graph = pg

	if !quiet {
		g := pg.MxGraph()
		fmt.Fprintf(output, "Graph built: %d nodes, %d edges\n",
			len(g.AllNodes()), len(g.AllEdges()))
	}
	return nil
}

// analyzeNavigationFn is the HandlerDeps version of AnalyzeNavigation.
func analyzeNavigationFn(ctx context.Context, output io.Writer, deps *HandlerDeps) error {
	if deps.Perf != nil {
		defer deps.Perf.Report()
	}
	if err := ensureGraphFn(output, deps.Quiet, deps.MprPath, &deps.Graph); err != nil {
		return err
	}

	nav := deps.Graph.NavigationProfiles()
	if len(nav) == 0 {
		fmt.Fprintln(output, "No navigation profiles found.")
		return nil
	}

	for _, p := range nav {
		fmt.Fprintf(output, "\n=== Profile: %s (%s)", p.Name, p.Kind)
		if p.IsNative {
			fmt.Fprint(output, " native")
		}
		fmt.Fprintln(output)
		if p.LoginPage != "" {
			fmt.Fprintf(output, "  Login: %s\n", p.LoginPage)
		}
		if p.NotFoundPage != "" {
			fmt.Fprintf(output, "  Not Found: %s\n", p.NotFoundPage)
		}

		if hp := deps.Graph.NavigationHomePage(p.Name); hp != nil {
			if hp.Page != "" {
				fmt.Fprintf(output, "  Home: page %s\n", hp.Page)
			} else if hp.Microflow != "" {
				fmt.Fprintf(output, "  Home: microflow %s\n", hp.Microflow)
			}
		}

		tree := deps.Graph.NavigationMenuTree(p.Name)
		if tree != nil && len(tree.MenuItems) > 0 {
			fmt.Fprintln(output, "  Menu:")
			printMenuTreeAnalyzeFn(output, tree.MenuItems, 4)
		}
	}
	return nil
}

func printMenuTreeAnalyzeFn(output io.Writer, items []graphcatalog.NavigationMenuItemTree, indent int) {
	prefix := strings.Repeat(" ", indent)
	for _, item := range items {
		target := ""
		if item.Item.Page != "" {
			target = fmt.Sprintf(" -> page %s", item.Item.Page)
		} else if item.Item.Microflow != "" {
			target = fmt.Sprintf(" -> microflow %s", item.Item.Microflow)
		}
		fmt.Fprintf(output, "%s- %s%s\n", prefix, item.Item.Caption, target)
		if len(item.Children) > 0 {
			printMenuTreeAnalyzeFn(output, item.Children, indent+2)
		}
	}
}

// analyzePageFn is the HandlerDeps version of AnalyzePage.
func analyzePageFn(ctx context.Context, output io.Writer, deps *HandlerDeps, pageQN string) error {
	if deps.Perf != nil {
		defer deps.Perf.Report()
	}
	if err := ensureGraphFn(output, deps.Quiet, deps.MprPath, &deps.Graph); err != nil {
		return err
	}
	if pageQN == "" {
		return mdlerrors.NewValidation("page qualified name required")
	}

	dcs := deps.Graph.PageDataContainerTree(pageQN)
	fmt.Fprintf(output, "\nPage: %s\n", pageQN)

	roles := deps.Graph.PageAllowedRoles(pageQN)
	if len(roles) > 0 {
		fmt.Fprintf(output, "Allowed Roles: %s\n", strings.Join(roles, ", "))
	}

	navRefs := findPageNavRefsAnalyzeFn(deps, pageQN)
	if len(navRefs) > 0 {
		fmt.Fprintf(output, "Navigation: %s\n", strings.Join(navRefs, ", "))
	} else {
		fmt.Fprintln(output, "Navigation: (none — orphan or microflow-only)")
	}

	if len(dcs) == 0 {
		fmt.Fprintln(output, "Data Containers: (none)")
		return nil
	}

	fmt.Fprintln(output, "\nData Containers:")
	printDataContainersAnalyzeFn(output, dcs, 0)
	return nil
}

func printDataContainersAnalyzeFn(output io.Writer, dcs []graphcatalog.DataContainerNode, depth int) {
	for _, dc := range dcs {
		prefix := strings.Repeat("  ", depth)
		dsInfo := ""
		if dc.DataSourceType != "none" {
			dsInfo = fmt.Sprintf(" (%s → %s)", dc.DataSourceType, dc.TargetEntity)
		}
		fmt.Fprintf(output, "%s├── %s \"%s\"%s  [depth=%d]\n", prefix, dc.WidgetType, dc.WidgetName, dsInfo, dc.Depth)

		if len(dc.ContextVariables) > 0 {
			var varStrs []string
			for _, v := range dc.ContextVariables {
				varStrs = append(varStrs, fmt.Sprintf("%s (%s)", v.Name, v.EntityType))
			}
			fmt.Fprintf(output, "%s│     Context: %s\n", prefix, strings.Join(varStrs, ", "))
		}

		if dc.HasSelection && dc.SelectionName != "" {
			fmt.Fprintf(output, "%s│     Selection: $%s (%s)\n", prefix, dc.SelectionName, dc.TargetEntity)
		}

		if len(dc.ChildWidgets) > 0 {
			fmt.Fprintf(output, "%s│     Widgets:\n", prefix)
			for _, cw := range dc.ChildWidgets {
				appearance := ""
				if cw.Class != "" {
					appearance += fmt.Sprintf(" class=\"%s\"", cw.Class)
				}
				if cw.Style != "" {
					appearance += fmt.Sprintf(" style=\"%s\"", cw.Style)
				}
				condInfo := ""
				if cw.ConditionalVisibility != "" {
					condInfo = fmt.Sprintf(", visible when \"%s\"", cw.ConditionalVisibility)
				}
				if cw.ConditionalEditability != "" {
					condInfo += fmt.Sprintf(", editable when \"%s\"", cw.ConditionalEditability)
				}
				attrInfo := ""
				if cw.Attribute != "" {
					attrInfo = fmt.Sprintf(" attr=%s", cw.Attribute)
				}
				captionInfo := ""
				if cw.Caption != "" {
					captionInfo = fmt.Sprintf(" caption=\"%s\"", cw.Caption)
				}
				fmt.Fprintf(output, "%s│       └─ %s [%s]%s%s%s%s\n",
					prefix, cw.Name, cw.WidgetType, attrInfo, captionInfo, appearance, condInfo)
			}
		}
	}
}

// analyzeEntityFn is the HandlerDeps version of AnalyzeEntity.
func analyzeEntityFn(ctx context.Context, output io.Writer, deps *HandlerDeps, entityQN string) error {
	if deps.Perf != nil {
		defer deps.Perf.Report()
	}
	if err := ensureGraphFn(output, deps.Quiet, deps.MprPath, &deps.Graph); err != nil {
		return err
	}
	if entityQN == "" {
		return mdlerrors.NewValidation("entity qualified name required")
	}

	flow := deps.Graph.EntityDataFlow(entityQN)
	if flow == nil {
		fmt.Fprintf(output, "Entity %s not found.\n", entityQN)
		return nil
	}

	fmt.Fprintf(output, "\nEntity: %s\n", entityQN)

	if len(flow.Pages) > 0 {
		fmt.Fprintln(output, "Referenced by pages:")
		for _, p := range flow.Pages {
			fmt.Fprintf(output, "  - page %s (via %s)\n", p.Page, p.DataSourceType)
			if len(p.AllowedRoles) > 0 {
				fmt.Fprintf(output, "    Roles: %s\n", strings.Join(p.AllowedRoles, ", "))
			}
			if len(p.NavigationRefs) > 0 {
				fmt.Fprintf(output, "    Navigation: %s\n", strings.Join(p.NavigationRefs, ", "))
			}
		}
	} else {
		fmt.Fprintln(output, "Pages: (none)")
	}

	if len(flow.Creators) > 0 {
		sort.Strings(flow.Creators)
		fmt.Fprintf(output, "Created by microflows: %s\n", strings.Join(flow.Creators, ", "))
	}
	if len(flow.Retrievers) > 0 {
		sort.Strings(flow.Retrievers)
		fmt.Fprintf(output, "Retrieved by microflows: %s\n", strings.Join(flow.Retrievers, ", "))
	}
	return nil
}

// analyzeOrphansFn is the HandlerDeps version of AnalyzeOrphans.
func analyzeOrphansFn(ctx context.Context, output io.Writer, deps *HandlerDeps) error {
	if deps.Perf != nil {
		defer deps.Perf.Report()
	}
	if err := ensureGraphFn(output, deps.Quiet, deps.MprPath, &deps.Graph); err != nil {
		return err
	}

	endPerf := perfStartFn(deps.Perf, "orphans.pages_refd")
	pages := deps.Graph.PagesReferencedByNavigation()
	if len(pages) > 0 {
		fmt.Fprintln(output, "\nPages referenced by navigation:")
		for _, p := range pages {
			fmt.Fprintf(output, "  ✅ %s\n", p)
		}
	}
	endPerf()

	endPerf2 := perfStartFn(deps.Perf, "orphans.find")
	orphans := deps.Graph.OrphanPages()
	if len(orphans) > 0 {
		fmt.Fprintf(output, "\nOrphan pages (no navigation/microflow ref):\n")
		for _, p := range orphans {
			fmt.Fprintf(output, "  ❌ %s\n", p)
		}
	} else {
		fmt.Fprintln(output, "\nNo orphan pages found.")
	}
	endPerf2()
	return nil
}

// findPageNavRefsAnalyzeFn finds navigation profiles referencing a page (HandlerDeps version).
func findPageNavRefsAnalyzeFn(deps *HandlerDeps, pageQN string) []string {
	endPerf := perfStartFn(deps.Perf, "page.nav_refs")
	defer endPerf()
	profiles := deps.Graph.NavigationProfiles()
	var refs []string
	for _, p := range profiles {
		tree := deps.Graph.NavigationMenuTree(p.Name)
		if tree == nil {
			continue
		}
		if hasPageRef(tree.MenuItems, pageQN) {
			refs = append(refs, p.Name)
		}
	}
	return refs
}

// analyzeFlowFn is the HandlerDeps version of AnalyzeFlow.
func analyzeFlowFn(ctx context.Context, output io.Writer, deps *HandlerDeps, entryKind, entryName string, maxDepth int) error {
	if deps.Perf != nil {
		defer deps.Perf.Report()
	}
	if err := ensureGraphFn(output, deps.Quiet, deps.MprPath, &deps.Graph); err != nil {
		return err
	}

	if entryKind == "" {
		endPerf := perfStartFn(deps.Perf, "flow.all_entry_points")
		eps := deps.Graph.AllEntryPoints()
		fmt.Fprintf(output, "\nApplication Entry Points (%d):\n", len(eps))
		for _, ep := range eps {
			fmt.Fprintf(output, "  %s: %s\n", ep.Kind, ep.Description)
		}
		endPerf()

		fmt.Fprintln(output, "\nAll Reachable Pages (from all entry points):")
		allPages := deps.Graph.AllReachablePages()
		orphans := deps.Graph.OrphanPages()
		orphanSet := map[string]bool{}
		for _, p := range orphans {
			orphanSet[p] = true
		}

		for page, sources := range allPages {
			status := "✅"
			if orphanSet[page] {
				status = "⚠️"
			}
			fmt.Fprintf(output, "  %s %s (via %s)\n", status, page, strings.Join(sources, ", "))
		}

		endPerf2 := perfStartFn(deps.Perf, "flow.unreachable")
		pages := deps.Graph.Pages("")
		for _, p := range pages {
			if _, ok := allPages[p.QualifiedName]; !ok {
				fmt.Fprintf(output, "  ❌ %s (unreachable)\n", p.QualifiedName)
			}
		}
		endPerf2()
		return nil
	}

	if entryName == "" {
		endPerf := perfStartFn(deps.Perf, "flow.list_kind")
		eps := deps.Graph.AllEntryPoints()
		var filtered []graphcatalog.EntryPoint
		for _, ep := range eps {
			if ep.Kind == entryKind {
				filtered = append(filtered, ep)
			}
		}
		if len(filtered) == 0 {
			fmt.Fprintf(output, "No %s entry points found.\n", entryKind)
			return nil
		}
		fmt.Fprintf(output, "\n%s Entry Points (%d):\n", entryKind, len(filtered))
		for _, ep := range filtered {
			fmt.Fprintf(output, "  %s\n", ep.Description)
		}
		endPerf()
		return nil
	}

	endPerf := perfStartFn(deps.Perf, "flow.chains")

	chains := deps.Graph.ReachablePagesFromEntry(entryKind, entryName)
	if len(chains) == 0 {
		eps := deps.Graph.AllEntryPoints()
		entryNameLower := strings.ToLower(entryName)
		for _, ep := range eps {
			if ep.Kind == entryKind && strings.Contains(strings.ToLower(ep.Description), entryNameLower) {
				chains = deps.Graph.ReachablePagesFromEntry(entryKind, ep.Name)
				if len(chains) > 0 {
					entryName = ep.Name
					break
				}
			}
		}
	}
	endPerf()

	if len(chains) == 0 {
		fmt.Fprintf(output, "No reachable pages from %s:%s\n", entryKind, entryName)
		return nil
	}

	fmt.Fprintf(output, "\nData Flow: %s → %s\n", entryKind, entryName)
	seen := map[string]bool{}
	for _, chain := range chains {
		if seen[chain.TerminalPage] {
			continue
		}
		seen[chain.TerminalPage] = true

		path := entryName
		for _, step := range chain.Steps {
			displayName := step.NodeName
			if step.NodeType == "NavigationMenuItem" {
				caption := ""
				if n := deps.Graph.MxGraph().GetNode(mxgraph.NodeID(step.NodeName)); n != nil {
					if c, ok := n.Props["Caption"].(string); ok && c != "" {
						caption = c
					}
				}
				if caption != "" {
					displayName = fmt.Sprintf("「%s」", caption)
				}
			}
			arrow := "→"
			switch step.EdgeType {
			case "TARGETS_PAGE", "SHOWS_PAGE":
				arrow = "→📄"
			case "TARGETS_MICROFLOW", "CALLS", "CALLS_MICROFLOW", "HAS_DATASOURCE_MICROFLOW":
				arrow = "→⚡"
			case "HAS_DATA_CONTAINER":
				arrow = "→🗂️"
			case "HAS_MENU_ITEM":
				arrow = "→📁"
			}
			path += fmt.Sprintf(" %s %s", arrow, displayName)
		}
		fmt.Fprintf(output, "  📍 %s\n", path)
	}
	return nil
}
