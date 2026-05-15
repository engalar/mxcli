// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

// defaultSlotContainer is the MDLContainer name that receives default (non-containerized) child widgets.
const defaultSlotContainer = "template"

// =============================================================================
// Pluggable Widget Engine — Core Types
// =============================================================================

// WidgetDefinition describes how to construct a pluggable widget from MDL syntax.
// Loaded from embedded JSON definition files (*.def.json).
type WidgetDefinition struct {
	WidgetID         string             `json:"widgetId"`
	MDLName          string             `json:"mdlName"`
	WidgetKind       string             `json:"widgetKind,omitempty"` // "pluggable" (React) or "custom" (legacy Dojo)
	TemplateFile     string             `json:"templateFile"`
	DefaultEditable  string             `json:"defaultEditable"`
	PropertyMappings []PropertyMapping  `json:"propertyMappings,omitempty"`
	ChildSlots       []ChildSlotMapping `json:"childSlots,omitempty"`
	Modes            []WidgetMode       `json:"modes,omitempty"`
}

// PropertyMapping maps an MDL source (attribute, association, literal, etc.)
// to a pluggable widget property key via a named operation.
type PropertyMapping struct {
	PropertyKey string `json:"propertyKey"`
	Source      string `json:"source,omitempty"`
	Value       string `json:"value,omitempty"`
	Operation   string `json:"operation"`
	Default     string `json:"default,omitempty"`
}

// WidgetMode defines a conditional configuration variant for a widget.
// For example, ComboBox has "enumeration" and "association" modes.
// Modes are evaluated in order; the first matching condition wins.
// A mode with no condition acts as the default fallback.
type WidgetMode struct {
	Name             string             `json:"name,omitempty"`
	Condition        string             `json:"condition,omitempty"`
	Description      string             `json:"description,omitempty"`
	PropertyMappings []PropertyMapping  `json:"propertyMappings"`
	ChildSlots       []ChildSlotMapping `json:"childSlots,omitempty"`
}

// ChildSlotMapping maps an MDL child container (e.g., TEMPLATE, FILTER) to a
// widget property that holds child widgets.
type ChildSlotMapping struct {
	PropertyKey  string `json:"propertyKey"`
	MDLContainer string `json:"mdlContainer"`
	Operation    string `json:"operation"`
}

// BuildContext carries resolved values from MDL parsing for use by operations.
type BuildContext struct {
	AttributePath  string
	AttributePaths []string // For operations that process multiple attributes
	AssocPath      string
	EntityName     string
	PrimitiveVal   string
	DataSource     backend.OpaqueDataSource
	Action         backend.OpaqueAction
	pageBuilder    *pageBuilder
}

// =============================================================================
// Pluggable Widget Engine
// =============================================================================

// PluggableWidgetEngine builds CustomWidget instances from WidgetDefinition + AST.
type PluggableWidgetEngine struct {
	backend     backend.WidgetBuilderBackend
	pageBuilder *pageBuilder
}

// NewPluggableWidgetEngine creates a new engine with the given backend and page builder.
func NewPluggableWidgetEngine(b backend.WidgetBuilderBackend, pb *pageBuilder) *PluggableWidgetEngine {
	return &PluggableWidgetEngine{
		backend:     b,
		pageBuilder: pb,
	}
}

// Build constructs a CustomWidget from a definition and AST widget node.
func (e *PluggableWidgetEngine) Build(def *WidgetDefinition, w *ast.WidgetV3) (*backend.CustomWidget, error) {
	// Save and restore entity context (DataSource mappings may change it)
	oldEntityContext := e.pageBuilder.entityContext
	defer func() { e.pageBuilder.entityContext = oldEntityContext }()

	// 1. Load template via backend
	builder, err := e.backend.LoadWidgetTemplate(def.WidgetID, e.pageBuilder.getProjectPath())
	if err != nil {
		return nil, mdlerrors.NewBackend("load "+def.MDLName+" template", err)
	}
	if builder == nil {
		return nil, mdlerrors.NewNotFound("template", def.MDLName)
	}

	propertyTypeIDs := builder.PropertyTypeIDs()

	// 2. Select mode and get mappings/slots
	mappings, slots, err := e.selectMappings(def, w)
	if err != nil {
		return nil, err
	}

	// 3. Apply property mappings
	for _, mapping := range mappings {
		ctx, err := e.resolveMapping(mapping, w)
		if err != nil {
			return nil, mdlerrors.NewBackend("resolve mapping for "+mapping.PropertyKey, err)
		}

		if err := e.applyOperation(builder, mapping.Operation, mapping.PropertyKey, ctx); err != nil {
			return nil, err
		}
	}

	// 3.5 Warn about DataSource-type properties that were not configured.
	// This gives AI assistants actionable feedback when they forget DataSource:
	for _, m := range mappings {
		if m.Source != "DataSource" {
			continue
		}
		if w.GetDataSource() == nil {
			// Collect linked attribute keys for the hint
			linked := linkedPropertyKeysFor(m.PropertyKey, propertyTypeIDs)
			hint := fmt.Sprintf("HINT [%s]: property '%s' is a DataSource type and was not configured.\n"+
				"  Add inside this widget block: DataSource: Module.Entity [where [XPath]]\n"+
				"  Linked attribute properties that draw from this datasource: %s",
				w.Name, m.PropertyKey, linked)
			log.Print(hint)
		}
	}

	// 4. Auto datasource: map AST DataSource to first DataSource-type property.
	// This must run before child slots so that entityContext is available
	// for child widgets that depend on the parent's data source.
	dsHandledByMapping := false
	for _, m := range mappings {
		if m.Source == "DataSource" {
			dsHandledByMapping = true
			break
		}
	}
	if !dsHandledByMapping {
		if ds := w.GetDataSource(); ds != nil {
			for propKey, entry := range propertyTypeIDs {
				if entry.ValueType == "DataSource" {
					dataSource, entityName, err := e.pageBuilder.buildDataSourceV3(ds)
					if err != nil {
						return nil, mdlerrors.NewBackend("auto datasource for "+propKey, err)
					}
					builder.SetDataSourceOpaque(propKey, e.backend.SerializeDataSourceToOpaque(dataSource))
					if entityName != "" {
						e.pageBuilder.entityContext = entityName
					}
					break
				}
			}
		}
	}

	// 4.1 Apply child slots (.def.json)
	if err := e.applyChildSlots(builder, slots, w, propertyTypeIDs); err != nil {
		return nil, err
	}

	// 4.3 Auto child slots: match AST children to Widgets-type template properties.
	handledSlotKeys := make(map[string]bool)
	for _, s := range slots {
		handledSlotKeys[s.PropertyKey] = true
	}
	var widgetsPropKeys []string
	for propKey, entry := range propertyTypeIDs {
		if entry.ValueType == "Widgets" && !handledSlotKeys[propKey] {
			widgetsPropKeys = append(widgetsPropKeys, propKey)
		}
	}
	sort.Strings(widgetsPropKeys)
	// Phase 1: Named matching — match children by name against property keys
	matchedChildren := make(map[int]bool)
	for _, propKey := range widgetsPropKeys {
		upperKey := strings.ToUpper(propKey)
		for i, child := range w.Children {
			if matchedChildren[i] {
				continue
			}
			if strings.ToUpper(child.Name) == upperKey {
				var childWidgets []backend.OpaqueWidget
				for _, slotChild := range child.Children {
					widget, err := e.pageBuilder.buildWidgetV3(slotChild)
					if err != nil {
						return nil, err
					}
					if widget != nil {
						if opaque := e.backend.SerializeWidgetToOpaque(widget); opaque != nil {
							childWidgets = append(childWidgets, opaque)
						}
					}
				}
				if len(childWidgets) > 0 {
					builder.SetChildWidgetsOpaque(propKey, childWidgets)
					handledSlotKeys[propKey] = true
				}
				matchedChildren[i] = true
				break
			}
		}
	}
	// Phase 2: Default slot — unmatched direct children go to first unmatched Widgets property.
	defSlotContainers := make(map[string]bool)
	for _, s := range slots {
		defSlotContainers[strings.ToUpper(s.MDLContainer)] = true
	}
	var defaultWidgets []backend.OpaqueWidget
	for i, child := range w.Children {
		if matchedChildren[i] {
			continue
		}
		if len(slots) > 0 {
			continue // applyChildSlots handles both container and direct children
		}
		if defSlotContainers[strings.ToUpper(child.Type)] {
			continue
		}
		widget, err := e.pageBuilder.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		if widget != nil {
			if opaque := e.backend.SerializeWidgetToOpaque(widget); opaque != nil {
				defaultWidgets = append(defaultWidgets, opaque)
			}
		}
	}
	if len(defaultWidgets) > 0 {
		for _, propKey := range widgetsPropKeys {
			if !handledSlotKeys[propKey] {
				builder.SetChildWidgetsOpaque(propKey, defaultWidgets)
				break
			}
		}
	}

	// 4.6 Apply explicit properties (not covered by .def.json mappings)
	mappedKeys := make(map[string]bool)
	for _, m := range mappings {
		if m.Source != "" {
			mappedKeys[m.Source] = true
		}
	}
	for _, s := range slots {
		mappedKeys[s.MDLContainer] = true
	}
	for propName, propVal := range w.Properties {
		if mappedKeys[propName] || isBuiltinPropName(propName) {
			continue
		}
		entry, ok := propertyTypeIDs[propName]
		if !ok {
			continue // not a known widget property key
		}

		// Handle action-type properties (e.g., onChange: nanoflow Module.NF)
		if actionAST, isAction := propVal.(*ast.ActionV3); isAction {
			if entry.ValueType == "Action" {
				act, err := e.pageBuilder.buildClientActionV3(actionAST)
				if err != nil {
					log.Printf("warning: widget %s property %s: %v", w.Name, propName, err)
				} else {
					builder.SetActionOpaque(propName, e.backend.SerializeClientActionToOpaque(act))
				}
			}
			continue
		}

		// Convert non-string values (bool, int, float) to string for property setting
		var strVal string
		switch v := propVal.(type) {
		case string:
			strVal = v
		case bool:
			strVal = fmt.Sprintf("%t", v)
		case int:
			strVal = fmt.Sprintf("%d", v)
		case float64:
			strVal = fmt.Sprintf("%g", v)
		default:
			continue
		}

		// Route by ValueType when available
		switch entry.ValueType {
		case "Expression":
			builder.SetExpression(propName, strVal)
		case "TextTemplate":
			entityCtx := e.pageBuilder.entityContext
			builder.SetTextTemplateWithParams(propName, strVal, entityCtx)
		case "Attribute":
			attrPath := ""
			if strings.Count(strVal, ".") >= 2 {
				attrPath = strVal
			} else if e.pageBuilder.entityContext != "" {
				attrPath = e.pageBuilder.resolveAttributePath(strVal)
			}
			if attrPath != "" {
				builder.SetAttribute(propName, attrPath)
			}
			// Warn when a linked attribute appears to use the context entity instead of
			// the datasource entity. Only fires when the attribute path has 2+ dots
			// (fully qualified) and the entity portion matches the context entity.
			if entry.DataSourceProperty != "" && strings.Count(strVal, ".") >= 2 {
				parts := strings.SplitN(strVal, ".", 3)
				attrEntity := parts[0] + "." + parts[1]
				// Warn when the attribute entity doesn't match the datasource entity.
				// After DataSource: processes gridData, entityContext is set to the
				// datasource entity — linked attrs should use that same entity.
				if e.pageBuilder.entityContext != "" && !strings.EqualFold(attrEntity, e.pageBuilder.entityContext) {
					log.Printf("HINT [%s]: property '%s' is linked to datasource '%s'.\n"+
						"  The attribute should reference the datasource entity '%s', not '%s'.\n"+
						"  Correct: %s.%s (or similar) — configure DataSource: first so the entity context is set.",
						w.Name, propName, entry.DataSourceProperty,
						e.pageBuilder.entityContext, attrEntity,
						e.pageBuilder.entityContext, parts[2])
				}
			}
		default:
			// Known non-attribute types: always use primitive
			if entry.ValueType != "" && entry.ValueType != "Attribute" {
				builder.SetPrimitive(propName, strVal)
				continue
			}
			// Legacy routing for properties without ValueType info
			if strings.Count(strVal, ".") >= 2 {
				builder.SetAttribute(propName, strVal)
			} else if e.pageBuilder.entityContext != "" && !strings.ContainsAny(strVal, " '\"") {
				builder.SetAttribute(propName, e.pageBuilder.resolveAttributePath(strVal))
			} else {
				builder.SetPrimitive(propName, strVal)
			}
		}
	}

	// 4.9 Auto-populate required empty object lists
	builder.EnsureRequiredObjectLists()

	// 5. Build CustomWidget via the legacy sdk path.
	// Stage 3.3.5.D1 NOTE: FinalizeGen() exists on the interface for future Cat-B
	// use, but cannot be wired here yet — the result (*GenCustomWidgetElem) goes
	// into an sdk-typed Container.Widgets list that mpr.SerializePage walks via
	// a concrete-type switch; GenCustomWidgetElem hits the default branch and
	// loses Object/Type BSON. Cat-B full page-gen rewrite will switch to FinalizeGen
	// and eliminate the SDKPageToGen bridge entirely.
	widgetID := model.ID(types.GenerateID())
	cw := builder.Finalize(widgetID, w.Name, w.GetLabel(), def.DefaultEditable)

	if err := e.pageBuilder.registerWidgetName(w.Name, cw.ID); err != nil {
		return nil, err
	}

	return cw, nil
}

// applyOperation dispatches a named operation to the corresponding builder method.
func (e *PluggableWidgetEngine) applyOperation(builder backend.WidgetObjectBuilder, opName string, propKey string, ctx *BuildContext) error {
	switch opName {
	case "attribute":
		builder.SetAttribute(propKey, ctx.AttributePath)
	case "association":
		builder.SetAssociation(propKey, ctx.AssocPath, ctx.EntityName)
	case "primitive":
		builder.SetPrimitive(propKey, ctx.PrimitiveVal)
	case "selection":
		builder.SetSelection(propKey, ctx.PrimitiveVal)
	case "expression":
		builder.SetExpression(propKey, ctx.PrimitiveVal)
	case "datasource":
		builder.SetDataSourceOpaque(propKey, ctx.DataSource)
	case "widgets":
		// ctx doesn't carry child widgets for this path — handled by applyChildSlots
	case "texttemplate":
		builder.SetTextTemplate(propKey, ctx.PrimitiveVal)
	case "action":
		builder.SetActionOpaque(propKey, ctx.Action)
	case "attributeObjects":
		builder.SetAttributeObjects(propKey, ctx.AttributePaths)
	default:
		return mdlerrors.NewValidationf("unknown operation %q for property %s", opName, propKey)
	}
	return nil
}

// selectMappings selects the active PropertyMappings and ChildSlotMappings based on mode.
func (e *PluggableWidgetEngine) selectMappings(def *WidgetDefinition, w *ast.WidgetV3) ([]PropertyMapping, []ChildSlotMapping, error) {
	if len(def.Modes) == 0 {
		return def.PropertyMappings, def.ChildSlots, nil
	}

	var fallback *WidgetMode
	var fallbackCount int
	for i := range def.Modes {
		mode := &def.Modes[i]
		if mode.Condition == "" {
			fallbackCount++
			if fallback == nil {
				fallback = mode
			}
			continue
		}
		if e.evaluateCondition(mode.Condition, w) {
			return mode.PropertyMappings, mode.ChildSlots, nil
		}
	}

	if fallback != nil {
		if fallbackCount > 1 {
			return nil, nil, mdlerrors.NewValidationf("widget %s has %d modes without conditions; only one default mode is allowed", def.MDLName, fallbackCount)
		}
		return fallback.PropertyMappings, fallback.ChildSlots, nil
	}

	return nil, nil, mdlerrors.NewValidationf("no matching mode for widget %s", def.MDLName)
}

// evaluateCondition checks a built-in condition string against the AST widget.
func (e *PluggableWidgetEngine) evaluateCondition(condition string, w *ast.WidgetV3) bool {
	switch {
	case condition == "hasDataSource":
		return w.GetDataSource() != nil
	case condition == "hasAttribute":
		return w.GetAttribute() != ""
	case strings.HasPrefix(condition, "hasProp:"):
		propName := strings.TrimPrefix(condition, "hasProp:")
		return w.GetStringProp(propName) != ""
	default:
		return false
	}
}

// resolveMapping resolves a PropertyMapping's source into a BuildContext.
func (e *PluggableWidgetEngine) resolveMapping(mapping PropertyMapping, w *ast.WidgetV3) (*BuildContext, error) {
	ctx := &BuildContext{pageBuilder: e.pageBuilder}

	if mapping.Value != "" {
		ctx.PrimitiveVal = mapping.Value
		return ctx, nil
	}

	source := mapping.Source
	if source == "" {
		return ctx, nil
	}

	switch source {
	case "Attribute":
		if attr := w.GetAttribute(); attr != "" {
			ctx.AttributePath = e.pageBuilder.resolveAttributePath(attr)
		}

	case "Attributes":
		if attrs := w.GetAttributes(); len(attrs) > 0 {
			ctx.AttributePaths = make([]string, 0, len(attrs))
			for _, attr := range attrs {
				ctx.AttributePaths = append(ctx.AttributePaths, e.pageBuilder.resolveAttributePath(attr))
			}
		}

	case "DataSource":
		if ds := w.GetDataSource(); ds != nil {
			dataSource, entityName, err := e.pageBuilder.buildDataSourceV3(ds)
			if err != nil {
				return nil, mdlerrors.NewBackend("build datasource", err)
			}
			ctx.DataSource = e.backend.SerializeDataSourceToOpaque(dataSource)
			ctx.EntityName = entityName
			if entityName != "" {
				e.pageBuilder.entityContext = entityName
				if w.Name != "" {
					e.pageBuilder.paramEntityNames[w.Name] = entityName
				}
			}
		}

	case "Selection":
		val := w.GetSelection()
		if val == "" && mapping.Default != "" {
			val = mapping.Default
		}
		ctx.PrimitiveVal = val

	case "CaptionAttribute":
		if captionAttr := w.GetStringProp("CaptionAttribute"); captionAttr != "" {
			if !strings.Contains(captionAttr, ".") && e.pageBuilder.entityContext != "" {
				captionAttr = e.pageBuilder.entityContext + "." + captionAttr
			}
			ctx.AttributePath = captionAttr
		}

	case "Association":
		if attr := w.GetAttribute(); attr != "" {
			ctx.AssocPath = e.pageBuilder.resolveAssociationPath(attr)
		}
		ctx.EntityName = e.pageBuilder.entityContext
		if ctx.AssocPath != "" && ctx.EntityName == "" {
			return nil, mdlerrors.NewValidationf("association %q requires an entity context (add a DataSource mapping before Association)", ctx.AssocPath)
		}

	case "OnClick":
		if action := w.GetAction(); action != nil {
			act, err := e.pageBuilder.buildClientActionV3(action)
			if err != nil {
				return nil, mdlerrors.NewBackend("build action", err)
			}
			ctx.Action = e.backend.SerializeClientActionToOpaque(act)
		}

	default:
		val := w.GetStringProp(source)
		if val == "" && mapping.Default != "" {
			val = mapping.Default
		}
		ctx.PrimitiveVal = val
	}

	return ctx, nil
}

// applyChildSlots processes child slot mappings, building child widgets and embedding them.
func (e *PluggableWidgetEngine) applyChildSlots(builder backend.WidgetObjectBuilder, slots []ChildSlotMapping, w *ast.WidgetV3, propertyTypeIDs map[string]types.PropertyTypeIDEntry) error {
	if len(slots) == 0 {
		return nil
	}

	slotContainers := make(map[string]*ChildSlotMapping, len(slots))
	for i := range slots {
		slotContainers[slots[i].MDLContainer] = &slots[i]
	}

	slotWidgets := make(map[string][]backend.OpaqueWidget)
	var defaultWidgets []backend.OpaqueWidget

	for _, child := range w.Children {
		lowerType := strings.ToLower(child.Type)
		if slot, ok := slotContainers[lowerType]; ok {
			for _, slotChild := range child.Children {
				widget, err := e.pageBuilder.buildWidgetV3(slotChild)
				if err != nil {
					return err
				}
				if widget != nil {
					if opaque := e.backend.SerializeWidgetToOpaque(widget); opaque != nil {
						slotWidgets[slot.PropertyKey] = append(slotWidgets[slot.PropertyKey], opaque)
					}
				}
			}
		} else {
			widget, err := e.pageBuilder.buildWidgetV3(child)
			if err != nil {
				return err
			}
			if widget != nil {
				if opaque := e.backend.SerializeWidgetToOpaque(widget); opaque != nil {
					defaultWidgets = append(defaultWidgets, opaque)
				}
			}
		}
	}

	for _, slot := range slots {
		children := slotWidgets[slot.PropertyKey]
		if len(children) == 0 && len(defaultWidgets) > 0 && slot.MDLContainer == defaultSlotContainer {
			children = defaultWidgets
			defaultWidgets = nil
		}
		if len(children) == 0 {
			continue
		}

		ctx := &BuildContext{}
		if slot.Operation != "widgets" {
			return mdlerrors.NewValidationf("childSlots operation must be %q, got %q for property %s", "widgets", slot.Operation, slot.PropertyKey)
		}
		if err := e.applyOperation(builder, slot.Operation, slot.PropertyKey, ctx); err != nil {
			return err
		}
		// SetChildWidgets directly — applyOperation skips "widgets" since ctx doesn't carry children
		builder.SetChildWidgetsOpaque(slot.PropertyKey, children)
	}

	return nil
}

// linkedPropertyKeysFor returns a comma-separated list of property keys whose
// DataSourceProperty field matches dsKey, drawn from the propertyTypeIDs map.
func linkedPropertyKeysFor(dsKey string, propertyTypeIDs map[string]types.PropertyTypeIDEntry) string {
	var keys []string
	for k, e := range propertyTypeIDs {
		if e.DataSourceProperty == dsKey {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return "(none)"
	}
	return strings.Join(keys, ", ")
}

// isBuiltinPropName returns true for property names that are handled by
// dedicated MDL keywords (DataSource, Attribute, etc.) rather than by
// the explicit property pass.
func isBuiltinPropName(name string) bool {
	switch name {
	case "DataSource", "Attribute", "Label", "Caption", "Action",
		"Selection", "Class", "Style", "Editable", "Visible",
		"WidgetType", "DesignProperties", "Association", "CaptionAttribute",
		"Content", "RenderMode", "ContentParams", "CaptionParams",
		"ButtonStyle", "DesktopWidth", "DesktopColumns", "TabletColumns",
		"PhoneColumns", "PageSize", "Pagination", "PagingPosition",
		"ShowPagingButtons", "Attributes", "FilterType", "Width", "Height",
		"Tooltip", "Name":
		return true
	}
	return false
}
