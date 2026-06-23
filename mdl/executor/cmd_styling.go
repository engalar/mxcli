// SPDX-License-Identifier: Apache-2.0

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

// ────────────────────────────────────────────────────────────
// Fn (HandlerDeps) versions
// ────────────────────────────────────────────────────────────

func execShowDesignPropertiesFn(ctx context.Context, s *ast.ShowDesignPropertiesStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}
	if deps.MprPath == "" {
		return mdlerrors.NewValidationf("project path unavailable — connected via mock backend without MprPath")
	}

	projectDir := filepath.Dir(deps.MprPath)
	registry, err := loadThemeRegistry(projectDir)
	if err != nil {
		return mdlerrors.NewBackend("load theme registry", err)
	}

	if len(registry.WidgetProperties) == 0 {
		fmt.Fprintln(deps.Output, "No design properties found. Check that themesource/*/web/design-properties.json exists in the project directory.")
		return nil
	}

	if s.WidgetType != "" {
		dpKey := resolveDesignPropsKey(s.WidgetType)
		props := registry.GetPropertiesForWidget(dpKey)
		if len(props) == 0 {
			fmt.Fprintf(deps.Output, "No design properties found for widget type %s (%s)\n", s.WidgetType, dpKey)
			return nil
		}
		fmt.Fprintf(deps.Output, "Design Properties for %s:\n\n", s.WidgetType)
		printDesignPropertiesFn(deps.Output, registry, dpKey)
	} else {
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
			fmt.Fprintf(deps.Output, "=== %s ===\n", key)
			for _, p := range props {
				printOnePropertyFn(deps.Output, p)
			}
			fmt.Fprintln(deps.Output)
		}
	}

	return nil
}

func printDesignPropertiesFn(output io.Writer, registry *ThemeRegistry, dpKey string) {
	if widgetProps, ok := registry.WidgetProperties["Widget"]; ok && len(widgetProps) > 0 {
		fmt.Fprintf(output, "From: Widget (inherited)\n")
		for _, p := range widgetProps {
			printOnePropertyFn(output, p)
		}
	}

	if dpKey != "Widget" {
		if typeProps, ok := registry.WidgetProperties[dpKey]; ok && len(typeProps) > 0 {
			fmt.Fprintf(output, "From: %s\n", dpKey)
			for _, p := range typeProps {
				printOnePropertyFn(output, p)
			}
		}
	}
}

func printOnePropertyFn(output io.Writer, p ThemeProperty) {
	switch p.Type {
	case "Toggle":
		fmt.Fprintf(output, "  %-24s Toggle      class: %s\n", p.Name, p.Class)
	case "Dropdown", "ColorPicker", "ToggleButtonGroup":
		options := make([]string, 0, len(p.Options))
		for _, o := range p.Options {
			options = append(options, o.Name)
		}
		fmt.Fprintf(output, "  %-24s %-11s [%s]\n", p.Name, p.Type, strings.Join(options, ", "))
	default:
		fmt.Fprintf(output, "  %-24s %s\n", p.Name, p.Type)
	}
}

func execDescribeStylingFn(ctx context.Context, s *ast.DescribeStylingStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	tmpCtx := NewExecContext(ctx, deps)

	h, err := getHierarchy(tmpCtx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	var rawWidgets []rawWidget

	if s.ContainerType == "page" {
		pageID, err := findPageIDGen(tmpCtx, s.ContainerName, h)
		if err != nil {
			return err
		}
		rawWidgets = getPageWidgetsFromRaw(tmpCtx, pageID)
	} else if s.ContainerType == "snippet" {
		snippetID, err := findSnippetIDGen(tmpCtx, s.ContainerName, h)
		if err != nil {
			return err
		}
		rawWidgets = getSnippetWidgetsFromRaw(tmpCtx, snippetID)
	}

	if len(rawWidgets) == 0 {
		fmt.Fprintf(deps.Output, "No widgets found in %s %s\n", s.ContainerType, s.ContainerName.String())
		return nil
	}

	styledWidgets := collectStyledWidgets(rawWidgets, s.WidgetName)

	if len(styledWidgets) == 0 {
		if s.WidgetName != "" {
			return mdlerrors.NewNotFoundMsg("widget", s.WidgetName, fmt.Sprintf("widget %q not found in %s %s", s.WidgetName, s.ContainerType, s.ContainerName.String()))
		}
		fmt.Fprintf(deps.Output, "No styled widgets found in %s %s\n", s.ContainerType, s.ContainerName.String())
		return nil
	}

	for i, w := range styledWidgets {
		if i > 0 {
			fmt.Fprintln(deps.Output)
		}
		displayName := getWidgetDisplayName(w.Type)
		fmt.Fprintf(deps.Output, "widget %s (%s)\n", w.Name, displayName)
		if w.Class != "" {
			fmt.Fprintf(deps.Output, "  Class: '%s'\n", w.Class)
		}
		if w.Style != "" {
			fmt.Fprintf(deps.Output, "  Style: '%s'\n", w.Style)
		}
		if len(w.DesignProperties) > 0 {
			fmt.Fprintf(deps.Output, "  DesignProperties: [")
			for j, dp := range w.DesignProperties {
				if j > 0 {
					fmt.Fprint(deps.Output, ", ")
				}
				if dp.ValueType == "toggle" {
					fmt.Fprintf(deps.Output, "'%s': on", dp.Key)
				} else {
					fmt.Fprintf(deps.Output, "'%s': '%s'", dp.Key, dp.Option)
				}
			}
			fmt.Fprintln(deps.Output, "]")
		}
	}

	return nil
}

func execAlterStylingFn(ctx context.Context, s *ast.AlterStylingStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}
	if deps.PageWriter == nil {
		return mdlerrors.NewNotConnectedWrite()
	}

	tmpCtx := NewExecContext(ctx, deps)

	h, err := getHierarchy(tmpCtx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	if s.ContainerType == "page" {
		return alterStylingOnPageFn(ctx, s, h, deps)
	} else if s.ContainerType == "snippet" {
		return alterStylingOnSnippetFn(ctx, s, h, deps)
	}

	return mdlerrors.NewUnsupported("unsupported container type: " + s.ContainerType)
}

func alterStylingOnPageFn(ctx context.Context, s *ast.AlterStylingStmt, h *ContainerHierarchy, deps *HandlerDeps) error {
	pairs, err := listPagesWithContainerGenCtx(deps)
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

	if err := deps.PageWriter.UpdatePageGen(page); err != nil {
		return mdlerrors.NewBackend("save page", err)
	}

	fmt.Fprintf(deps.Output, "Updated styling on widget %q in page %s\n", s.WidgetName, s.ContainerName.String())
	return nil
}

func alterStylingOnSnippetFn(ctx context.Context, s *ast.AlterStylingStmt, h *ContainerHierarchy, deps *HandlerDeps) error {
	pairs, err := listSnippetsWithContainerGenCtx(deps)
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

	if err := deps.PageWriter.UpdateSnippetGen(snippet); err != nil {
		return mdlerrors.NewBackend("save snippet", err)
	}

	fmt.Fprintf(deps.Output, "Updated styling on widget %q in snippet %s\n", s.WidgetName, s.ContainerName.String())
	return nil
}

// listPagesWithContainerGenCtx builds a page list from HandlerDeps
// (no cache), matching the ContainerWithGen shape the other helpers expect.
func listPagesWithContainerGenCtx(deps *HandlerDeps) ([]ContainerWithGen[*genPg.Page], error) {
	if deps.PageRepo == nil {
		return nil, nil
	}
	all, err := deps.PageRepo.ListAll()
	if err != nil {
		return nil, err
	}
	var result []ContainerWithGen[*genPg.Page]
	for _, p := range all {
		if p == nil {
			continue
		}
		containerID, _ := deps.PageRepo.GetContainerUUID(model.ID(p.ID()))
		result = append(result, ContainerWithGen[*genPg.Page]{Elem: p, ContainerID: element.ID(containerID)})
	}
	return result, nil
}

// listSnippetsWithContainerGenCtx builds a snippet list from HandlerDeps.
func listSnippetsWithContainerGenCtx(deps *HandlerDeps) ([]ContainerWithGen[*genPg.Snippet], error) {
	if deps.SnippetRepo == nil {
		return nil, nil
	}
	all, err := deps.SnippetRepo.ListAll()
	if err != nil {
		return nil, err
	}
	var result []ContainerWithGen[*genPg.Snippet]
	for _, p := range all {
		if p == nil {
			continue
		}
		containerID, _ := deps.SnippetRepo.GetContainerUUID(model.ID(p.ID()))
		result = append(result, ContainerWithGen[*genPg.Snippet]{Elem: p, ContainerID: element.ID(containerID)})
	}
	return result, nil
}

// ────────────────────────────────────────────────────────────
// Old ExecContext wrappers (delegate to Fn versions)
// ────────────────────────────────────────────────────────────


func printDesignProperties(ctx *ExecContext, registry *ThemeRegistry, dpKey string) {
	printDesignPropertiesFn(ctx.Output, registry, dpKey)
}

func printOneProperty(ctx *ExecContext, p ThemeProperty) {
	printOnePropertyFn(ctx.Output, p)
}



func alterStylingOnPage(ctx *ExecContext, s *ast.AlterStylingStmt, h *ContainerHierarchy) error {
	return alterStylingOnPageFn(ctx, s, h, execContextToDeps(ctx))
}

func alterStylingOnSnippet(ctx *ExecContext, s *ast.AlterStylingStmt, h *ContainerHierarchy) error {
	return alterStylingOnSnippetFn(ctx, s, h, execContextToDeps(ctx))
}

// ────────────────────────────────────────────────────────────
// Stateless helpers (no ctx/deps needed)
// ────────────────────────────────────────────────────────────

// collectStyledWidgets walks rawWidget tree and collects widgets that have styling.
// If widgetName is set, only returns the widget matching that name.
func collectStyledWidgets(widgets []rawWidget, widgetName string) []rawWidget {
	var result []rawWidget
	var walk func(ws []rawWidget)
	walk = func(ws []rawWidget) {
		for _, w := range ws {
			if widgetName != "" {
				if w.Name == widgetName {
					result = append(result, w)
					return
				}
			} else {
				if w.Class != "" || w.Style != "" || len(w.DesignProperties) > 0 {
					result = append(result, w)
				}
			}
			walk(w.Children)
			for _, row := range w.Rows {
				for _, col := range row.Columns {
					walk(col.Widgets)
				}
			}
			walk(w.FilterWidgets)
			walk(w.ControlBar)
		}
	}
	walk(widgets)
	return result
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
		for i, item := range items {
			dpv, ok := item.(*genPg.DesignPropertyValue)
			if ok && dpv.Key() == a.Property {
				app.RemoveDesignProperties(i)
				return nil
			}
		}
		return nil
	}

	for _, item := range items {
		dpv, ok := item.(*genPg.DesignPropertyValue)
		if !ok || dpv.Key() != a.Property {
			continue
		}
		dpv.SetValue(buildDesignPropertySubValue(a))
		return nil
	}

	app.AddDesignProperties(newDesignPropertyEntry(a.Property, buildDesignPropertySubValue(a)))
	return nil
}

// newDesignPropertyEntry creates the correct BSON structure for a design
// property entry:
//
//	Appearance.DesignProperties[]
//	  └── DesignPropertyValue          ← this wrapper is required by Studio Pro
//	        ├── Key: "spacing"
//	        └── Value: OptionDesignPropertyValue | ToggleDesignPropertyValue
//
// Always use this constructor — never add OptionDesignPropertyValue or
// ToggleDesignPropertyValue directly to AddDesignProperties(). Studio Pro
// requires DesignPropertyValue as the parent; omitting it causes a load crash:
//
//	"OptionDesignPropertyValue does not contain a constructor with a
//	 parameter of type Appearance"
func newDesignPropertyEntry(key string, value element.Element) *genPg.DesignPropertyValue {
	dpv := genPg.NewDesignPropertyValue()
	assignFreshID(dpv)
	dpv.SetKey(key)
	dpv.SetValue(value)
	return dpv
}

// buildDesignPropertySubValue constructs the inner Value element for a design
// property. The returned element must be wrapped in newDesignPropertyEntry
// before being added to Appearance.DesignProperties.
func buildDesignPropertySubValue(a ast.StylingAssignment) element.Element {
	if a.IsToggle {
		return genPg.NewToggleDesignPropertyValue()
	}
	opt := genPg.NewOptionDesignPropertyValue()
	opt.SetOption(a.Value)
	return opt
}
