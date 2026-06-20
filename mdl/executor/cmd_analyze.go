// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
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

	defer perfStart(ctx, "flow.chains")()
	chains := ctx.Graph.ReachablePagesFromEntry(entryKind, entryName)
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
