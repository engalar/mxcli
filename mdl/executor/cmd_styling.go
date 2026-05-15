// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/pages"
)

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

	// Find page
	allPages, err := ctx.Backend.ListPages()
	if err != nil {
		return mdlerrors.NewBackend("list pages", err)
	}

	var page *pages.Page
	for _, p := range allPages {
		modID := h.FindModuleID(p.ContainerID)
		modName := h.GetModuleName(modID)
		if p.Name == s.ContainerName.Name && (s.ContainerName.Module == "" || modName == s.ContainerName.Module) {
			page = p
			break
		}
	}
	if page == nil {
		return mdlerrors.NewNotFound("page", s.ContainerName.String())
	}

	// Walk the page to find the widget by name
	found := false
	err = walkPageWidgets(page, func(widget any) error {
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

	// Save the page
	if err := ctx.Backend.UpdatePage(page); err != nil {
		return mdlerrors.NewBackend("save page", err)
	}

	fmt.Fprintf(ctx.Output, "Updated styling on widget %q in page %s\n", s.WidgetName, s.ContainerName.String())
	return nil
}

func alterStylingOnSnippet(ctx *ExecContext, s *ast.AlterStylingStmt, h *ContainerHierarchy) error {

	// Find snippet
	allSnippets, err := ctx.Backend.ListSnippets()
	if err != nil {
		return mdlerrors.NewBackend("list snippets", err)
	}

	var snippet *pages.Snippet
	for _, sn := range allSnippets {
		modID := h.FindModuleID(sn.ContainerID)
		modName := h.GetModuleName(modID)
		if sn.Name == s.ContainerName.Name && (s.ContainerName.Module == "" || modName == s.ContainerName.Module) {
			snippet = sn
			break
		}
	}
	if snippet == nil {
		return mdlerrors.NewNotFound("snippet", s.ContainerName.String())
	}

	// Walk the snippet to find the widget by name
	found := false
	err = walkSnippetWidgets(snippet, func(widget any) error {
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

	// Save the snippet
	if err := ctx.Backend.UpdateSnippet(snippet); err != nil {
		return mdlerrors.NewBackend("save snippet", err)
	}

	fmt.Fprintf(ctx.Output, "Updated styling on widget %q in snippet %s\n", s.WidgetName, s.ContainerName.String())
	return nil
}

// getWidgetName extracts the Name from a widget using reflection.
func getWidgetName(widget any) string {
	if widget == nil {
		return ""
	}

	// Try Widget interface first
	if w, ok := widget.(pages.Widget); ok {
		return w.GetName()
	}

	// Fall back to reflection
	v := reflect.ValueOf(widget)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return ""
	}

	// Try BaseWidget.Name
	if baseWidget := v.FieldByName("BaseWidget"); baseWidget.IsValid() {
		if nameField := baseWidget.FieldByName("Name"); nameField.IsValid() {
			return nameField.String()
		}
	}

	// Direct Name field
	if nameField := v.FieldByName("Name"); nameField.IsValid() {
		return nameField.String()
	}

	return ""
}

// applyStylingAssignments applies styling changes to a widget.
func applyStylingAssignments(widget any, assignments []ast.StylingAssignment, clearDesignProps bool) error {
	v := reflect.ValueOf(widget)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return mdlerrors.NewValidation("widget is not a struct")
	}

	// Get BaseWidget
	baseWidget := v.FieldByName("BaseWidget")
	if !baseWidget.IsValid() {
		return mdlerrors.NewValidation("widget has no BaseWidget field")
	}

	// Clear design properties if requested
	if clearDesignProps {
		dpField := baseWidget.FieldByName("DesignProperties")
		if dpField.IsValid() && dpField.CanSet() {
			dpField.Set(reflect.Zero(dpField.Type()))
		}
	}

	for _, a := range assignments {
		switch a.Property {
		case "Class":
			classField := baseWidget.FieldByName("Class")
			if classField.IsValid() && classField.CanSet() {
				classField.SetString(a.Value)
			}
		case "Style":
			styleField := baseWidget.FieldByName("Style")
			if styleField.IsValid() && styleField.CanSet() {
				styleField.SetString(a.Value)
			}
		default:
			// Design property assignment
			if err := setDesignProperty(baseWidget, a); err != nil {
				return err
			}
		}
	}

	return nil
}

// setDesignProperty sets or updates a design property on the widget's BaseWidget.
func setDesignProperty(baseWidget reflect.Value, a ast.StylingAssignment) error {
	dpField := baseWidget.FieldByName("DesignProperties")
	if !dpField.IsValid() || !dpField.CanSet() {
		return mdlerrors.NewUnsupported("widget does not support design properties")
	}

	// Get existing design properties
	var existing []pages.DesignPropertyValue
	if !dpField.IsNil() {
		existing = dpField.Interface().([]pages.DesignPropertyValue)
	}

	if a.IsToggle && !a.ToggleOn {
		// OFF: remove the design property
		var updated []pages.DesignPropertyValue
		for _, dp := range existing {
			if dp.Key != a.Property {
				updated = append(updated, dp)
			}
		}
		dpField.Set(reflect.ValueOf(updated))
		return nil
	}

	// Update existing or append new
	found := false
	for i, dp := range existing {
		if dp.Key == a.Property {
			if a.IsToggle {
				existing[i].ValueType = "toggle"
				existing[i].Option = ""
			} else {
				existing[i].ValueType = "option"
				existing[i].Option = a.Value
			}
			found = true
			break
		}
	}

	if !found {
		newProp := pages.DesignPropertyValue{
			Key: a.Property,
		}
		if a.IsToggle {
			newProp.ValueType = "toggle"
		} else {
			newProp.ValueType = "option"
			newProp.Option = a.Value
		}
		existing = append(existing, newProp)
	}

	dpField.Set(reflect.ValueOf(existing))
	return nil
}

// findPageByName looks up a page by qualified name.
func findPageByName(ctx *ExecContext, name ast.QualifiedName, h *ContainerHierarchy) (*pages.Page, error) {

	allPages, err := ctx.Backend.ListPages()
	if err != nil {
		return nil, mdlerrors.NewBackend("list pages", err)
	}
	for _, p := range allPages {
		modID := h.FindModuleID(p.ContainerID)
		modName := h.GetModuleName(modID)
		if p.Name == name.Name && (name.Module == "" || modName == name.Module) {
			return p, nil
		}
	}
	return nil, mdlerrors.NewNotFound("page", name.String())
}

// findSnippetByName looks up a snippet by qualified name.
func findSnippetByName(ctx *ExecContext, name ast.QualifiedName, h *ContainerHierarchy) (*pages.Snippet, model.ID, error) {

	allSnippets, err := ctx.Backend.ListSnippets()
	if err != nil {
		return nil, "", mdlerrors.NewBackend("list snippets", err)
	}
	for _, s := range allSnippets {
		modID := h.FindModuleID(s.ContainerID)
		modName := h.GetModuleName(modID)
		if s.Name == name.Name && (name.Module == "" || modName == name.Module) {
			return s, modID, nil
		}
	}
	return nil, "", mdlerrors.NewNotFound("snippet", name.String())
}

// walkPageWidgets walks all sdk-typed widgets in a page for styling operations.
// Uses sdk/pages types because applyStylingAssignments relies on BaseWidget reflection.
func walkPageWidgets(page *pages.Page, visitor func(widget any) error) error {
	if page == nil || page.LayoutCall == nil {
		return nil
	}
	for _, arg := range page.LayoutCall.Arguments {
		if arg.Widget != nil {
			if err := walkWidget(arg.Widget, visitor); err != nil {
				return err
			}
		}
	}
	return nil
}

// walkSnippetWidgets walks all sdk-typed widgets in a snippet for styling operations.
func walkSnippetWidgets(snippet *pages.Snippet, visitor func(widget any) error) error {
	if snippet == nil {
		return nil
	}
	for _, widget := range snippet.Widgets {
		if err := walkWidget(widget, visitor); err != nil {
			return err
		}
	}
	return nil
}

// walkWidget recursively walks an sdk-typed widget and its children.
func walkWidget(widget pages.Widget, visitor func(widget any) error) error {
	if widget == nil {
		return nil
	}
	if err := visitor(widget); err != nil {
		return err
	}
	switch w := widget.(type) {
	case *pages.LayoutGrid:
		for _, row := range w.Rows {
			for _, col := range row.Columns {
				for _, child := range col.Widgets {
					if err := walkWidget(child, visitor); err != nil {
						return err
					}
				}
			}
		}
	case *pages.DataView:
		for _, child := range w.Widgets {
			if err := walkWidget(child, visitor); err != nil {
				return err
			}
		}
		for _, child := range w.FooterWidgets {
			if err := walkWidget(child, visitor); err != nil {
				return err
			}
		}
	case *pages.ListView:
		for _, child := range w.Widgets {
			if err := walkWidget(child, visitor); err != nil {
				return err
			}
		}
	case *pages.Container:
		for _, child := range w.Widgets {
			if err := walkWidget(child, visitor); err != nil {
				return err
			}
		}
	case *pages.GroupBox:
		for _, child := range w.Widgets {
			if err := walkWidget(child, visitor); err != nil {
				return err
			}
		}
	case *pages.TabContainer:
		for _, pg := range w.TabPages {
			for _, child := range pg.Widgets {
				if err := walkWidget(child, visitor); err != nil {
					return err
				}
			}
		}
	case *pages.ScrollContainer:
		for _, child := range w.Widgets {
			if err := walkWidget(child, visitor); err != nil {
				return err
			}
		}
	case *pages.CustomWidget:
		if w.WidgetObject != nil {
			for _, prop := range w.WidgetObject.Properties {
				if prop.Value != nil {
					for _, child := range prop.Value.Widgets {
						if err := walkWidget(child, visitor); err != nil {
							return err
						}
					}
				}
			}
		}
	}
	return nil
}
