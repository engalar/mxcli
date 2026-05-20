// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// styledWidget is implemented by all gen-typed page widget types that support
// Class, Style, and Appearance (DesignProperties) mutations.
type styledWidget interface {
	Name() string
	Class() string
	SetClass(string)
	Style() string
	SetStyle(string)
	Appearance() element.Element
	SetAppearance(element.Element)
}

// ============================================================================
// SHOW DESIGN PROPERTIES
// ============================================================================

func execShowDesignProperties(ctx *ExecContext, s *ast.ShowDesignPropertiesStmt) error {

	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}
	if ctx.MprPath == "" {
		return mdlerrors.NewValidationf("project path unavailable — connected via mock backend without MprPath")
	}

	projectDir := filepath.Dir(ctx.MprPath)
	registry, err := loadThemeRegistry(projectDir)
	if err != nil {
		return mdlerrors.NewBackend("load theme registry", err)
	}

	if len(registry.WidgetProperties) == 0 {
		fmt.Fprintln(ctx.Output, "No design properties found. Check that themesource/*/web/design-properties.json exists in the project directory.")
		return nil
	}

	if s.WidgetType != "" {
		// Show properties for a specific widget type
		dpKey := resolveDesignPropsKey(s.WidgetType)
		props := registry.GetPropertiesForWidget(dpKey)
		if len(props) == 0 {
			fmt.Fprintf(ctx.Output, "No design properties found for widget type %s (%s)\n", s.WidgetType, dpKey)
			return nil
		}
		fmt.Fprintf(ctx.Output, "Design Properties for %s:\n\n", s.WidgetType)
		printDesignProperties(ctx, registry, dpKey)
	} else {
		// Show all widget types and their properties
		keys := make([]string, 0, len(registry.WidgetProperties))
		for k := range registry.WidgetProperties {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, key := range keys {
			props := registry.WidgetProperties[key]
			if len(props) == 0 {
				continue
			}
			fmt.Fprintf(ctx.Output, "=== %s ===\n", key)
			for _, p := range props {
				printOneProperty(ctx, p)
			}
			fmt.Fprintln(ctx.Output)
		}
	}

	return nil
}

// printDesignProperties prints properties for a widget type, showing inherited "Widget" props separately.
func printDesignProperties(ctx *ExecContext, registry *ThemeRegistry, dpKey string) {
	// Print inherited Widget properties
	if widgetProps, ok := registry.WidgetProperties["Widget"]; ok && len(widgetProps) > 0 {
		fmt.Fprintf(ctx.Output, "From: Widget (inherited)\n")
		for _, p := range widgetProps {
			printOneProperty(ctx, p)
		}
	}

	// Print type-specific properties
	if dpKey != "Widget" {
		if typeProps, ok := registry.WidgetProperties[dpKey]; ok && len(typeProps) > 0 {
			fmt.Fprintf(ctx.Output, "From: %s\n", dpKey)
			for _, p := range typeProps {
				printOneProperty(ctx, p)
			}
		}
	}
}

// printOneProperty prints a single design property in a readable format.
func printOneProperty(ctx *ExecContext, p ThemeProperty) {
	switch p.Type {
	case "Toggle":
		fmt.Fprintf(ctx.Output, "  %-24s Toggle      class: %s\n", p.Name, p.Class)
	case "Dropdown", "ColorPicker", "ToggleButtonGroup":
		options := make([]string, 0, len(p.Options))
		for _, o := range p.Options {
			options = append(options, o.Name)
		}
		fmt.Fprintf(ctx.Output, "  %-24s %-11s [%s]\n", p.Name, p.Type, strings.Join(options, ", "))
	default:
		fmt.Fprintf(ctx.Output, "  %-24s %s\n", p.Name, p.Type)
	}
}

// ============================================================================
// DESCRIBE STYLING
// ============================================================================

func execDescribeStyling(ctx *ExecContext, s *ast.DescribeStylingStmt) error {

	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	var rawWidgets []rawWidget

	if s.ContainerType == "page" {
		pageID, err := findPageIDGen(ctx, s.ContainerName, h)
		if err != nil {
			return err
		}
		rawWidgets = getPageWidgetsFromRaw(ctx, pageID)
	} else if s.ContainerType == "snippet" {
		snippetID, err := findSnippetIDGen(ctx, s.ContainerName, h)
		if err != nil {
			return err
		}
		rawWidgets = getSnippetWidgetsFromRaw(ctx, snippetID)
	}

	if len(rawWidgets) == 0 {
		fmt.Fprintf(ctx.Output, "No widgets found in %s %s\n", s.ContainerType, s.ContainerName.String())
		return nil
	}

	// Collect styled widgets
	styledWidgets := collectStyledWidgets(rawWidgets, s.WidgetName)

	if len(styledWidgets) == 0 {
		if s.WidgetName != "" {
			return mdlerrors.NewNotFoundMsg("widget", s.WidgetName, fmt.Sprintf("widget %q not found in %s %s", s.WidgetName, s.ContainerType, s.ContainerName.String()))
		}
		fmt.Fprintf(ctx.Output, "No styled widgets found in %s %s\n", s.ContainerType, s.ContainerName.String())
		return nil
	}

	// Output
	for i, w := range styledWidgets {
		if i > 0 {
			fmt.Fprintln(ctx.Output)
		}
		displayName := getWidgetDisplayName(w.Type)
		fmt.Fprintf(ctx.Output, "widget %s (%s)\n", w.Name, displayName)
		if w.Class != "" {
			fmt.Fprintf(ctx.Output, "  Class: '%s'\n", w.Class)
		}
		if w.Style != "" {
			fmt.Fprintf(ctx.Output, "  Style: '%s'\n", w.Style)
		}
		if len(w.DesignProperties) > 0 {
			fmt.Fprintf(ctx.Output, "  DesignProperties: [")
			for j, dp := range w.DesignProperties {
				if j > 0 {
					fmt.Fprint(ctx.Output, ", ")
				}
				if dp.ValueType == "toggle" {
					fmt.Fprintf(ctx.Output, "'%s': on", dp.Key)
				} else {
					fmt.Fprintf(ctx.Output, "'%s': '%s'", dp.Key, dp.Option)
				}
			}
			fmt.Fprintln(ctx.Output, "]")
		}
	}

	return nil
}

// collectStyledWidgets walks rawWidget tree and collects widgets that have styling.
// If widgetName is set, only returns the widget matching that name.
func collectStyledWidgets(widgets []rawWidget, widgetName string) []rawWidget {
	var result []rawWidget
	var walk func(ws []rawWidget)
	walk = func(ws []rawWidget) {
		for _, w := range ws {
			if widgetName != "" {
				// Looking for specific widget
				if w.Name == widgetName {
					result = append(result, w)
					return // Found it
				}
			} else {
				// Collect all widgets with any styling
				if w.Class != "" || w.Style != "" || len(w.DesignProperties) > 0 {
					result = append(result, w)
				}
			}
			// Walk children
			walk(w.Children)
			// Walk rows (for LayoutGrid)
			for _, row := range w.Rows {
				for _, col := range row.Columns {
					walk(col.Widgets)
				}
			}
			// Walk filter/controlbar widgets
			walk(w.FilterWidgets)
			walk(w.ControlBar)
		}
	}
	walk(widgets)
	return result
}

// ============================================================================
// ALTER STYLING
// ============================================================================

func execAlterStyling(ctx *ExecContext, s *ast.AlterStylingStmt) error {

	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	if s.ContainerType == "page" {
		return alterStylingOnPage(ctx, s, h)
	} else if s.ContainerType == "snippet" {
		return alterStylingOnSnippet(ctx, s, h)
	}

	return mdlerrors.NewUnsupported("unsupported container type: " + s.ContainerType)
}

func alterStylingOnPage(ctx *ExecContext, s *ast.AlterStylingStmt, h *ContainerHierarchy) error {

	// Find page via gen-typed listing
	pairs, err := listPagesWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list pages", err)
	}

	var page *genPg.Page
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(model.ID(p.ContainerID))
		modName := h.GetModuleName(modID)
		if p.Elem.Name() == s.ContainerName.Name && (s.ContainerName.Module == "" || modName == s.ContainerName.Module) {
			page = p.Elem
			break
		}
	}
	if page == nil {
		return mdlerrors.NewNotFound("page", s.ContainerName.String())
	}

	// Walk the page to find the widget by name
	found := false
	err = walkPageWidgetsGen(page, func(widget element.Element) error {
		name := getWidgetName(widget)
		if name != s.WidgetName {
			return nil
		}
		found = true
		return applyStylingAssignments(widget, s.Assignments, s.ClearDesignProps)
	})
	if err != nil {
		return err
	}

	if !found {
		return mdlerrors.NewNotFoundMsg("widget", s.WidgetName, fmt.Sprintf("widget %q not found in page %s", s.WidgetName, s.ContainerName.String()))
	}

	// Save the page via gen path
	if err := ctx.Backend.UpdatePageGen(page); err != nil {
		return mdlerrors.NewBackend("save page", err)
	}

	fmt.Fprintf(ctx.Output, "Updated styling on widget %q in page %s\n", s.WidgetName, s.ContainerName.String())
	return nil
}

func alterStylingOnSnippet(ctx *ExecContext, s *ast.AlterStylingStmt, h *ContainerHierarchy) error {

	// Find snippet via gen-typed listing
	pairs, err := listSnippetsWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list snippets", err)
	}

	var snippet *genPg.Snippet
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(model.ID(p.ContainerID))
		modName := h.GetModuleName(modID)
		if p.Elem.Name() == s.ContainerName.Name && (s.ContainerName.Module == "" || modName == s.ContainerName.Module) {
			snippet = p.Elem
			break
		}
	}
	if snippet == nil {
		return mdlerrors.NewNotFound("snippet", s.ContainerName.String())
	}

	// Walk the snippet to find the widget by name
	found := false
	err = walkSnippetWidgetsGen(snippet, func(widget element.Element) error {
		name := getWidgetName(widget)
		if name != s.WidgetName {
			return nil
		}
		found = true
		return applyStylingAssignments(widget, s.Assignments, s.ClearDesignProps)
	})
	if err != nil {
		return err
	}

	if !found {
		return mdlerrors.NewNotFoundMsg("widget", s.WidgetName, fmt.Sprintf("widget %q not found in snippet %s", s.WidgetName, s.ContainerName.String()))
	}

	// Save the snippet via gen path
	if err := ctx.Backend.UpdateSnippetGen(snippet); err != nil {
		return mdlerrors.NewBackend("save snippet", err)
	}

	fmt.Fprintf(ctx.Output, "Updated styling on widget %q in snippet %s\n", s.WidgetName, s.ContainerName.String())
	return nil
}

// getWidgetName extracts the Name from a gen-typed widget element.
func getWidgetName(widget element.Element) string {
	if widget == nil {
		return ""
	}
	if w, ok := widget.(interface{ Name() string }); ok {
		return w.Name()
	}
	return ""
}

// applyStylingAssignments applies styling changes to a gen-typed widget element.
func applyStylingAssignments(widget element.Element, assignments []ast.StylingAssignment, clearDesignProps bool) error {
	sw, ok := widget.(styledWidget)
	if !ok {
		return mdlerrors.NewUnsupported("widget type does not support styling")
	}

	// Clear design properties if requested
	if clearDesignProps {
		if app, ok := sw.Appearance().(*genPg.Appearance); ok && app != nil {
			for i := len(app.DesignPropertiesItems()) - 1; i >= 0; i-- {
				app.RemoveDesignProperties(i)
			}
		}
	}

	for _, a := range assignments {
		switch a.Property {
		case "Class":
			sw.SetClass(a.Value)
		case "Style":
			sw.SetStyle(a.Value)
		default:
			if err := setDesignPropertyGen(sw, a); err != nil {
				return err
			}
		}
	}

	return nil
}

// setDesignPropertyGen sets or updates a design property on a gen-typed widget via its Appearance.
func setDesignPropertyGen(sw styledWidget, a ast.StylingAssignment) error {
	// Ensure Appearance exists
	var app *genPg.Appearance
	if existing := sw.Appearance(); existing != nil {
		if ap, ok := existing.(*genPg.Appearance); ok {
			app = ap
		}
	}
	if app == nil {
		app = newDefaultAppearance()
		sw.SetAppearance(app)
	}

	items := app.DesignPropertiesItems()

	if a.IsToggle && !a.ToggleOn {
		// OFF: remove the matching design property value
		for i, item := range items {
			dpv, ok := item.(*genPg.DesignPropertyValue)
			if ok && dpv.Key() == a.Property {
				app.RemoveDesignProperties(i)
				return nil
			}
		}
		return nil
	}

	// Update existing or append new
	for _, item := range items {
		dpv, ok := item.(*genPg.DesignPropertyValue)
		if !ok || dpv.Key() != a.Property {
			continue
		}
		// Replace the Value sub-element with updated type
		dpv.SetValue(buildDesignPropertySubValue(a))
		return nil
	}

	// Not found — append new DesignPropertyValue
	dpv := genPg.NewDesignPropertyValue()
	dpv.SetKey(a.Property)
	dpv.SetValue(buildDesignPropertySubValue(a))
	app.AddDesignProperties(dpv)
	return nil
}

// buildDesignPropertySubValue constructs the nested Value element for a design property assignment.
func buildDesignPropertySubValue(a ast.StylingAssignment) element.Element {
	if a.IsToggle {
		return genPg.NewToggleDesignPropertyValue()
	}
	opt := genPg.NewOptionDesignPropertyValue()
	opt.SetOption(a.Value)
	return opt
}

