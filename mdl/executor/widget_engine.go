// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"encoding/hex"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/mpr"
	"github.com/mendixlabs/mxcli/sdk/pages"
	"github.com/mendixlabs/mxcli/sdk/widgets"
	"go.mongodb.org/mongo-driver/bson"
)

// defaultSlotContainer is the MDLContainer name that receives default (non-containerized) child widgets.
const defaultSlotContainer = "TEMPLATE"

// =============================================================================
// Pluggable Widget Engine — Core Types and Operation Registry
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

// PropertyMapping maps an MDL source (attribute, association, literal, etc.)
// to a pluggable widget property key via a named operation.
type PropertyMapping struct {
	PropertyKey string `json:"propertyKey"`
	Source      string `json:"source,omitempty"`
	Value       string `json:"value,omitempty"`
	Operation   string `json:"operation"`
	Default     string `json:"default,omitempty"`
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
	AttributePath string
	AssocPath     string
	EntityName    string
	PrimitiveVal  string
	DataSource    pages.DataSource
	ChildWidgets  []bson.D
}

// OperationFunc updates a template object's property identified by propertyKey.
// It receives the current object BSON, the property type ID map, the target key,
// and the build context containing resolved values.
type OperationFunc func(obj bson.D, propTypeIDs map[string]pages.PropertyTypeIDEntry, propertyKey string, ctx *BuildContext) bson.D

// OperationRegistry maps operation names to their implementations.
type OperationRegistry struct {
	operations map[string]OperationFunc
}

// NewOperationRegistry creates a registry pre-loaded with the 6 built-in operations.
func NewOperationRegistry() *OperationRegistry {
	reg := &OperationRegistry{
		operations: make(map[string]OperationFunc),
	}
	reg.Register("attribute", opAttribute)
	reg.Register("association", opAssociation)
	reg.Register("primitive", opPrimitive)
	reg.Register("selection", opSelection)
	reg.Register("datasource", opDatasource)
	reg.Register("widgets", opWidgets)
	return reg
}

// Register adds or replaces an operation by name.
func (r *OperationRegistry) Register(name string, fn OperationFunc) {
	r.operations[name] = fn
}

// Lookup returns the operation function for the given name, or nil if not found.
func (r *OperationRegistry) Lookup(name string) OperationFunc {
	return r.operations[name]
}

// Has returns true if the named operation is registered.
func (r *OperationRegistry) Has(name string) bool {
	_, ok := r.operations[name]
	return ok
}

// =============================================================================
// Built-in Operations
// =============================================================================

// opAttribute sets an attribute reference on a widget property.
func opAttribute(obj bson.D, propTypeIDs map[string]pages.PropertyTypeIDEntry, propertyKey string, ctx *BuildContext) bson.D {
	if ctx.AttributePath == "" {
		return obj
	}
	return updateWidgetPropertyValue(obj, propTypeIDs, propertyKey, func(val bson.D) bson.D {
		return setAttributeRef(val, ctx.AttributePath)
	})
}

// opAssociation sets an association reference (AttributeRef + EntityRef) on a widget property.
func opAssociation(obj bson.D, propTypeIDs map[string]pages.PropertyTypeIDEntry, propertyKey string, ctx *BuildContext) bson.D {
	if ctx.AssocPath == "" {
		return obj
	}
	return updateWidgetPropertyValue(obj, propTypeIDs, propertyKey, func(val bson.D) bson.D {
		return setAssociationRef(val, ctx.AssocPath, ctx.EntityName)
	})
}

// opPrimitive sets a primitive string value on a widget property.
func opPrimitive(obj bson.D, propTypeIDs map[string]pages.PropertyTypeIDEntry, propertyKey string, ctx *BuildContext) bson.D {
	if ctx.PrimitiveVal == "" {
		return obj
	}
	return updateWidgetPropertyValue(obj, propTypeIDs, propertyKey, func(val bson.D) bson.D {
		return setPrimitiveValue(val, ctx.PrimitiveVal)
	})
}

// opSelection sets a selection mode on a widget property, updating the Selection field
// inside the WidgetValue (which requires a deeper update than opPrimitive's PrimitiveValue).
func opSelection(obj bson.D, propTypeIDs map[string]pages.PropertyTypeIDEntry, propertyKey string, ctx *BuildContext) bson.D {
	if ctx.PrimitiveVal == "" {
		return obj
	}
	return updateWidgetPropertyValue(obj, propTypeIDs, propertyKey, func(val bson.D) bson.D {
		result := make(bson.D, 0, len(val))
		for _, elem := range val {
			if elem.Key == "Selection" {
				result = append(result, bson.E{Key: "Selection", Value: ctx.PrimitiveVal})
			} else {
				result = append(result, elem)
			}
		}
		return result
	})
}

// opExpression sets an expression string on a widget property.
func opExpression(obj bson.D, propTypeIDs map[string]pages.PropertyTypeIDEntry, propertyKey string, ctx *BuildContext) bson.D {
	if ctx.PrimitiveVal == "" {
		return obj
	}
	return updateWidgetPropertyValue(obj, propTypeIDs, propertyKey, func(val bson.D) bson.D {
		result := make(bson.D, 0, len(val))
		for _, elem := range val {
			if elem.Key == "Expression" {
				result = append(result, bson.E{Key: "Expression", Value: ctx.PrimitiveVal})
			} else {
				result = append(result, elem)
			}
		}
		return result
	})
}

// opDatasource sets a data source on a widget property.
func opDatasource(obj bson.D, propTypeIDs map[string]pages.PropertyTypeIDEntry, propertyKey string, ctx *BuildContext) bson.D {
	if ctx.DataSource == nil {
		return obj
	}
	return updateWidgetPropertyValue(obj, propTypeIDs, propertyKey, func(val bson.D) bson.D {
		return setDataSource(val, ctx.DataSource)
	})
}

// opWidgets replaces the Widgets array in a widget property value with child widgets.
func opWidgets(obj bson.D, propTypeIDs map[string]pages.PropertyTypeIDEntry, propertyKey string, ctx *BuildContext) bson.D {
	if len(ctx.ChildWidgets) == 0 {
		return obj
	}
	result := updateWidgetPropertyValue(obj, propTypeIDs, propertyKey, func(val bson.D) bson.D {
		updated := setChildWidgets(val, ctx.ChildWidgets)
		// Verify Widgets was actually set
		for _, e := range updated {
			if e.Key == "Widgets" {
				if arr, ok := e.Value.(bson.A); ok {
					log.Printf("opWidgets %s: Widgets array has %d items", propertyKey, len(arr))
				}
			}
		}
		return updated
	})
	return result
}

// setChildWidgets replaces the Widgets field in a WidgetValue with the given child widgets.
func setChildWidgets(val bson.D, childWidgets []bson.D) bson.D {
	widgetsArr := bson.A{int32(2)} // version marker
	for _, w := range childWidgets {
		widgetsArr = append(widgetsArr, w)
	}

	result := make(bson.D, 0, len(val))
	for _, elem := range val {
		if elem.Key == "Widgets" {
			result = append(result, bson.E{Key: "Widgets", Value: widgetsArr})
		} else {
			result = append(result, elem)
		}
	}
	return result
}

// =============================================================================
// Pluggable Widget Engine
// =============================================================================

// PluggableWidgetEngine builds CustomWidget instances from WidgetDefinition + AST.
type PluggableWidgetEngine struct {
	operations  *OperationRegistry
	pageBuilder *pageBuilder
}

// NewPluggableWidgetEngine creates a new engine with the given registry and page builder.
func NewPluggableWidgetEngine(ops *OperationRegistry, pb *pageBuilder) *PluggableWidgetEngine {
	return &PluggableWidgetEngine{
		operations:  ops,
		pageBuilder: pb,
	}
}

// Build constructs a CustomWidget from a definition and AST widget node.
func (e *PluggableWidgetEngine) Build(def *WidgetDefinition, w *ast.WidgetV3) (*pages.CustomWidget, error) {
	// Save and restore entity context (DataSource mappings may change it)
	oldEntityContext := e.pageBuilder.entityContext
	defer func() { e.pageBuilder.entityContext = oldEntityContext }()

	// 1. Load template
	embeddedType, embeddedObject, embeddedIDs, embeddedObjectTypeID, err :=
		widgets.GetTemplateFullBSON(def.WidgetID, mpr.GenerateID, e.pageBuilder.reader.Path())
	if err != nil {
		return nil, fmt.Errorf("failed to load %s template: %w", def.MDLName, err)
	}
	if embeddedType == nil || embeddedObject == nil {
		return nil, fmt.Errorf("%s template not found", def.MDLName)
	}

	propertyTypeIDs := convertPropertyTypeIDs(embeddedIDs)
	updatedObject := embeddedObject

	// 2. Select mode and get mappings/slots
	mappings, slots, err := e.selectMappings(def, w)
	if err != nil {
		return nil, err
	}

	// 3. Apply property mappings
	for _, mapping := range mappings {
		ctx, err := e.resolveMapping(mapping, w)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve mapping for %s: %w", mapping.PropertyKey, err)
		}

		op := e.operations.Lookup(mapping.Operation)
		if op == nil {
			return nil, fmt.Errorf("unknown operation %q for property %s", mapping.Operation, mapping.PropertyKey)
		}

		updatedObject = op(updatedObject, propertyTypeIDs, mapping.PropertyKey, ctx)
	}

	// 4. Apply child slots (.def.json)
	if err := e.applyChildSlots(slots, w, propertyTypeIDs, &updatedObject); err != nil {
		return nil, err
	}

	// 4.1 Auto datasource: map AST DataSource to first DataSource-type property.
	// Must run BEFORE child slots and explicit properties so entityContext is set.
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
						return nil, fmt.Errorf("auto datasource for %s: %w", propKey, err)
					}
					ctx := &BuildContext{DataSource: dataSource, EntityName: entityName}
					updatedObject = opDatasource(updatedObject, propertyTypeIDs, propKey, ctx)
					if entityName != "" {
						e.pageBuilder.entityContext = entityName
					}
					break
				}
			}
		}
	}

	// 4.3 Auto child slots: match AST children to Widgets-type template properties.
	// Two matching strategies:
	//   1. Named match: CONTAINER trigger { ... } → property "trigger" (by child name)
	//   2. Default slot: direct children not matching any named slot → first Widgets property
	// This allows pluggable widget child containers without requiring .def.json ChildSlot entries.
	handledSlotKeys := make(map[string]bool)
	for _, s := range slots {
		handledSlotKeys[s.PropertyKey] = true
	}
	// Collect Widgets-type property keys
	var widgetsPropKeys []string
	for propKey, entry := range propertyTypeIDs {
		if entry.ValueType == "Widgets" && !handledSlotKeys[propKey] {
			widgetsPropKeys = append(widgetsPropKeys, propKey)
		}
	}
	// Debug: log Widgets-type properties and children for matching
	if len(w.Children) > 0 && len(widgetsPropKeys) > 0 {
		var childNames []string
		for _, c := range w.Children {
			childNames = append(childNames, c.Name+"("+c.Type+")")
		}
		log.Printf("auto-slot %s: widgetProps=%v children=%v", w.Name, widgetsPropKeys, childNames)
	}
	// Phase 1: Named matching — match children by name against property keys
	matchedChildren := make(map[int]bool) // indices of matched children
	for _, propKey := range widgetsPropKeys {
		upperKey := strings.ToUpper(propKey)
		for i, child := range w.Children {
			if matchedChildren[i] {
				continue
			}
			if strings.ToUpper(child.Name) == upperKey {
				var childBSONs []bson.D
				log.Printf("auto-slot match: %s.%s has %d AST children", w.Name, propKey, len(child.Children))
				for _, slotChild := range child.Children {
					widgetBSON, err := e.pageBuilder.buildWidgetV3ToBSON(slotChild)
					if err != nil {
						return nil, err
					}
					if widgetBSON != nil {
						childBSONs = append(childBSONs, widgetBSON)
					}
				}
				log.Printf("auto-slot match: %s.%s built %d BSON widgets", w.Name, propKey, len(childBSONs))
				if len(childBSONs) > 0 {
					updatedObject = opWidgets(updatedObject, propertyTypeIDs, propKey, &BuildContext{ChildWidgets: childBSONs})
					handledSlotKeys[propKey] = true
				}
				matchedChildren[i] = true
				break
			}
		}
	}
	// Phase 2: Default slot — unmatched direct children go to first unmatched Widgets property.
	// Skip children already consumed by .def.json child slots (matched by type).
	defSlotContainers := make(map[string]bool)
	for _, s := range slots {
		defSlotContainers[strings.ToUpper(s.MDLContainer)] = true
	}
	var defaultWidgetBSONs []bson.D
	for i, child := range w.Children {
		if matchedChildren[i] {
			continue
		}
		if defSlotContainers[strings.ToUpper(child.Type)] {
			continue // already consumed by applyChildSlots
		}
		widgetBSON, err := e.pageBuilder.buildWidgetV3ToBSON(child)
		if err != nil {
			return nil, err
		}
		if widgetBSON != nil {
			defaultWidgetBSONs = append(defaultWidgetBSONs, widgetBSON)
		}
	}
	if len(defaultWidgetBSONs) > 0 {
		for _, propKey := range widgetsPropKeys {
			if !handledSlotKeys[propKey] {
				updatedObject = opWidgets(updatedObject, propertyTypeIDs, propKey, &BuildContext{ChildWidgets: defaultWidgetBSONs})
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
		ctx := &BuildContext{}

		// Route by ValueType when available
		switch entry.ValueType {
		case "Expression":
			// Expression properties: set Expression field (not PrimitiveValue)
			ctx.PrimitiveVal = strVal
			updatedObject = opExpression(updatedObject, propertyTypeIDs, propName, ctx)
		case "TextTemplate":
			// TextTemplate properties: create ClientTemplate with attribute parameter binding.
			// Syntax: '{AttributeName} - {OtherAttr}' → text '{1} - {2}' with TemplateParameters.
			entityCtx := e.pageBuilder.entityContext
			tmplBSON := createClientTemplateBSONWithParams(strVal, entityCtx)
			updatedObject = updateWidgetPropertyValue(updatedObject, propertyTypeIDs, propName, func(val bson.D) bson.D {
				result := make(bson.D, 0, len(val))
				for _, elem := range val {
					if elem.Key == "TextTemplate" {
						result = append(result, bson.E{Key: "TextTemplate", Value: tmplBSON})
					} else {
						result = append(result, elem)
					}
				}
				return result
			})
		case "Attribute":
			// Attribute properties: resolve path
			if strings.Count(strVal, ".") >= 2 {
				ctx.AttributePath = strVal
			} else if e.pageBuilder.entityContext != "" {
				ctx.AttributePath = e.pageBuilder.resolveAttributePath(strVal)
			}
			if ctx.AttributePath != "" {
				updatedObject = opAttribute(updatedObject, propertyTypeIDs, propName, ctx)
			}
		default:
			// Known non-attribute types: always use primitive
			if entry.ValueType != "" && entry.ValueType != "Attribute" {
				ctx.PrimitiveVal = strVal
				updatedObject = opPrimitive(updatedObject, propertyTypeIDs, propName, ctx)
				continue
			}
			// Legacy routing for properties without ValueType info
			if strings.Count(strVal, ".") >= 2 {
				ctx.AttributePath = strVal
				updatedObject = opAttribute(updatedObject, propertyTypeIDs, propName, ctx)
			} else if e.pageBuilder.entityContext != "" && !strings.ContainsAny(strVal, " '\"") {
				ctx.AttributePath = e.pageBuilder.resolveAttributePath(strVal)
				updatedObject = opAttribute(updatedObject, propertyTypeIDs, propName, ctx)
			} else {
				ctx.PrimitiveVal = strVal
				updatedObject = opPrimitive(updatedObject, propertyTypeIDs, propName, ctx)
			}
		}
	}

	// 4.9 Auto-populate required empty object lists (e.g., Accordion groups, AreaChart series)
	updatedObject = ensureRequiredObjectLists(updatedObject, propertyTypeIDs)

	// Debug: verify updatedObject has Widgets content before building
	if w.Name == "timelineCustom" {
		for _, elem := range updatedObject {
			if elem.Key == "Properties" {
				if arr, ok := elem.Value.(bson.A); ok {
					for _, item := range arr {
						if prop, ok := item.(bson.D); ok {
							for _, pe := range prop {
								if pe.Key == "Value" {
									if val, ok := pe.Value.(bson.D); ok {
										for _, ve := range val {
											if ve.Key == "Widgets" {
												if wa, ok := ve.Value.(bson.A); ok && len(wa) > 1 {
													log.Printf("BUILD CHECK: timelineCustom has non-empty Widgets: %d items", len(wa))
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// 5. Build CustomWidget
	widgetID := model.ID(mpr.GenerateID())
	cw := &pages.CustomWidget{
		BaseWidget: pages.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       widgetID,
				TypeName: "CustomWidgets$CustomWidget",
			},
			Name: w.Name,
		},
		Label:             w.GetLabel(),
		Editable:          def.DefaultEditable,
		RawType:           embeddedType,
		RawObject:         updatedObject,
		PropertyTypeIDMap: propertyTypeIDs,
		ObjectTypeID:      embeddedObjectTypeID,
	}

	if err := e.pageBuilder.registerWidgetName(w.Name, cw.ID); err != nil {
		return nil, err
	}

	return cw, nil
}

// selectMappings selects the active PropertyMappings and ChildSlotMappings based on mode.
// Modes are evaluated in definition order; the first matching condition wins.
// A mode with no condition acts as the default fallback.
func (e *PluggableWidgetEngine) selectMappings(def *WidgetDefinition, w *ast.WidgetV3) ([]PropertyMapping, []ChildSlotMapping, error) {
	// No modes defined — use top-level mappings directly
	if len(def.Modes) == 0 {
		return def.PropertyMappings, def.ChildSlots, nil
	}

	// Evaluate modes in order; first match wins
	var fallback *WidgetMode
	for i := range def.Modes {
		mode := &def.Modes[i]
		if mode.Condition == "" {
			// No condition = default fallback (use first one if multiple)
			if fallback == nil {
				fallback = mode
			}
			continue
		}
		if e.evaluateCondition(mode.Condition, w) {
			return mode.PropertyMappings, mode.ChildSlots, nil
		}
	}

	// Use fallback mode
	if fallback != nil {
		return fallback.PropertyMappings, fallback.ChildSlots, nil
	}

	return nil, nil, fmt.Errorf("no matching mode for widget %s", def.MDLName)
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
		log.Printf("warning: unknown widget condition %q — returning false", condition)
		return false
	}
}

// resolveMapping resolves a PropertyMapping's source into a BuildContext.
func (e *PluggableWidgetEngine) resolveMapping(mapping PropertyMapping, w *ast.WidgetV3) (*BuildContext, error) {
	ctx := &BuildContext{}

	// Static value takes priority
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

	case "DataSource":
		if ds := w.GetDataSource(); ds != nil {
			dataSource, entityName, err := e.pageBuilder.buildDataSourceV3(ds)
			if err != nil {
				return nil, fmt.Errorf("failed to build datasource: %w", err)
			}
			ctx.DataSource = dataSource
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
			// Resolve relative to entity context
			if !strings.Contains(captionAttr, ".") && e.pageBuilder.entityContext != "" {
				captionAttr = e.pageBuilder.entityContext + "." + captionAttr
			}
			ctx.AttributePath = captionAttr
		}

	case "Association":
		// For association operation: resolve both assoc path AND entity name from DataSource
		if attr := w.GetAttribute(); attr != "" {
			ctx.AssocPath = e.pageBuilder.resolveAssociationPath(attr)
		}
		// Entity name comes from DataSource context (must be resolved first by a DataSource mapping)
		ctx.EntityName = e.pageBuilder.entityContext

	default:
		// Generic fallback: treat source as a property name on the AST widget
		val := w.GetStringProp(source)
		if val == "" && mapping.Default != "" {
			val = mapping.Default
		}
		ctx.PrimitiveVal = val
	}

	return ctx, nil
}

// applyChildSlots processes child slot mappings, building child widgets and embedding them.
func (e *PluggableWidgetEngine) applyChildSlots(slots []ChildSlotMapping, w *ast.WidgetV3, propertyTypeIDs map[string]pages.PropertyTypeIDEntry, updatedObject *bson.D) error {
	if len(slots) == 0 {
		return nil
	}

	// Build a set of slot container names for matching
	slotContainers := make(map[string]*ChildSlotMapping, len(slots))
	for i := range slots {
		slotContainers[slots[i].MDLContainer] = &slots[i]
	}

	// Group children by slot
	slotWidgets := make(map[string][]bson.D)
	var defaultWidgets []bson.D

	for _, child := range w.Children {
		upperType := strings.ToUpper(child.Type)
		if slot, ok := slotContainers[upperType]; ok {
			// Container matches a slot — build its children
			for _, slotChild := range child.Children {
				widgetBSON, err := e.pageBuilder.buildWidgetV3ToBSON(slotChild)
				if err != nil {
					return err
				}
				if widgetBSON != nil {
					slotWidgets[slot.PropertyKey] = append(slotWidgets[slot.PropertyKey], widgetBSON)
				}
			}
		} else {
			// Direct child — default content
			widgetBSON, err := e.pageBuilder.buildWidgetV3ToBSON(child)
			if err != nil {
				return err
			}
			if widgetBSON != nil {
				defaultWidgets = append(defaultWidgets, widgetBSON)
			}
		}
	}

	// Apply each slot's widgets via its operation
	for _, slot := range slots {
		childBSONs := slotWidgets[slot.PropertyKey]
		// If no explicit container children, use default widgets for the first slot
		if len(childBSONs) == 0 && len(defaultWidgets) > 0 && slot.MDLContainer == defaultSlotContainer {
			childBSONs = defaultWidgets
			defaultWidgets = nil // consume once
		}
		if len(childBSONs) == 0 {
			continue
		}

		op := e.operations.Lookup(slot.Operation)
		if op == nil {
			return fmt.Errorf("unknown operation %q for child slot %s", slot.Operation, slot.PropertyKey)
		}

		ctx := &BuildContext{ChildWidgets: childBSONs}
		*updatedObject = op(*updatedObject, propertyTypeIDs, slot.PropertyKey, ctx)
	}

	return nil
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

// =============================================================================
// Default Object List Population
// =============================================================================

// ensureRequiredObjectLists populates empty Object list properties with one default
// entry. This prevents CE0642 "Property 'X' is required" errors for widget properties
// like Accordion groups, AreaChart series, etc.
func ensureRequiredObjectLists(obj bson.D, propertyTypeIDs map[string]pages.PropertyTypeIDEntry) bson.D {
	for propKey, entry := range propertyTypeIDs {
		if entry.ObjectTypeID == "" || len(entry.NestedPropertyIDs) == 0 {
			continue
		}
		// Skip non-required object lists that have nested DataSource properties —
		// auto-populating these creates entries that trigger widget-level validation errors.
		// Required object lists (like AreaChart series) are populated even with nested DataSource
		// because the DataSource is conditional (e.g., depends on dataSet enum).
		if !entry.Required {
			hasNestedDS := false
			for _, nested := range entry.NestedPropertyIDs {
				if nested.ValueType == "DataSource" {
					hasNestedDS = true
					break
				}
			}
			if hasNestedDS {
				continue
			}
		}
		// Skip if any Required nested property is Attribute (needs entity context)
		hasRequiredAttr := false
		for _, nested := range entry.NestedPropertyIDs {
			if nested.Required && nested.ValueType == "Attribute" {
				hasRequiredAttr = true
				break
			}
		}
		if hasRequiredAttr {
			continue
		}
		obj = updateWidgetPropertyValue(obj, propertyTypeIDs, propKey, func(val bson.D) bson.D {
			for _, elem := range val {
				if elem.Key == "Objects" {
					if arr, ok := elem.Value.(bson.A); ok && len(arr) <= 1 {
						// Empty Objects array — create one default entry
						defaultObj := createDefaultWidgetObject(entry.ObjectTypeID, entry.NestedPropertyIDs)
						newArr := bson.A{int32(2), defaultObj}
						result := make(bson.D, 0, len(val))
						for _, e := range val {
							if e.Key == "Objects" {
								result = append(result, bson.E{Key: "Objects", Value: newArr})
							} else {
								result = append(result, e)
							}
						}
						return result
					}
				}
			}
			return val
		})
	}
	return obj
}

// createDefaultWidgetObject creates a minimal WidgetObject BSON entry for an object list.
func createDefaultWidgetObject(objectTypeID string, nestedProps map[string]pages.PropertyTypeIDEntry) bson.D {
	propsArr := bson.A{int32(2)} // version marker
	for _, entry := range nestedProps {
		prop := createDefaultWidgetProperty(entry)
		propsArr = append(propsArr, prop)
	}
	return bson.D{
		{Key: "$ID", Value: generateBinaryID()},
		{Key: "$Type", Value: "CustomWidgets$WidgetObject"},
		{Key: "TypePointer", Value: hexIDToBlob(objectTypeID)},
		{Key: "Properties", Value: propsArr},
	}
}

// createDefaultWidgetProperty creates a WidgetProperty with default WidgetValue.
func createDefaultWidgetProperty(entry pages.PropertyTypeIDEntry) bson.D {
	return bson.D{
		{Key: "$ID", Value: generateBinaryID()},
		{Key: "$Type", Value: "CustomWidgets$WidgetProperty"},
		{Key: "TypePointer", Value: hexIDToBlob(entry.PropertyTypeID)},
		{Key: "Value", Value: createDefaultWidgetValue(entry)},
	}
}

// createDefaultWidgetValue creates a WidgetValue with standard default fields.
// Sets type-specific defaults: Expression→Expression field, TextTemplate→template, etc.
func createDefaultWidgetValue(entry pages.PropertyTypeIDEntry) bson.D {
	primitiveVal := entry.DefaultValue
	expressionVal := ""
	var textTemplate interface{} // nil by default

	// Route default value to the correct field based on ValueType
	switch entry.ValueType {
	case "Expression":
		expressionVal = primitiveVal
		primitiveVal = ""
	case "TextTemplate":
		// Create a ClientTemplate with a placeholder translation to satisfy CE4899
		text := primitiveVal
		if text == "" {
			text = " " // non-empty to satisfy "required" translation check
		}
		textTemplate = createDefaultClientTemplateBSON(text)
	case "String":
		if primitiveVal == "" {
			primitiveVal = " " // non-empty to satisfy required String properties
		}
	}

	return bson.D{
		{Key: "$ID", Value: generateBinaryID()},
		{Key: "$Type", Value: "CustomWidgets$WidgetValue"},
		{Key: "Action", Value: bson.D{
			{Key: "$ID", Value: generateBinaryID()},
			{Key: "$Type", Value: "Forms$NoAction"},
			{Key: "DisabledDuringExecution", Value: true},
		}},
		{Key: "AttributeRef", Value: nil},
		{Key: "DataSource", Value: nil},
		{Key: "EntityRef", Value: nil},
		{Key: "Expression", Value: expressionVal},
		{Key: "Form", Value: ""},
		{Key: "Icon", Value: nil},
		{Key: "Image", Value: ""},
		{Key: "Microflow", Value: ""},
		{Key: "Nanoflow", Value: ""},
		{Key: "Objects", Value: bson.A{int32(2)}},
		{Key: "PrimitiveValue", Value: primitiveVal},
		{Key: "Selection", Value: "None"},
		{Key: "SourceVariable", Value: nil},
		{Key: "TextTemplate", Value: textTemplate},
		{Key: "TranslatableValue", Value: nil},
		{Key: "TypePointer", Value: hexIDToBlob(entry.ValueTypeID)},
		{Key: "Widgets", Value: bson.A{int32(2)}},
		{Key: "XPathConstraint", Value: ""},
	}
}

// createClientTemplateBSONWithParams creates a Forms$ClientTemplate that supports
// attribute parameter binding. Syntax: '{AttrName} - {OtherAttr}' extracts attribute
// names from curly braces, replaces them with {1}, {2}, etc., and generates
// TemplateParameter entries with AttributeRef bindings.
// If no {AttrName} patterns are found, creates a static text template.
func createClientTemplateBSONWithParams(text string, entityContext string) bson.D {
	// Extract {AttributeName} patterns and build parameter list
	re := regexp.MustCompile(`\{([A-Za-z][A-Za-z0-9_]*)\}`)
	matches := re.FindAllStringSubmatchIndex(text, -1)

	if len(matches) == 0 {
		// No attribute references — static text
		return createDefaultClientTemplateBSON(text)
	}

	// Replace {AttrName} with {1}, {2}, etc. and collect attribute names
	var attrNames []string
	paramText := text
	// Process in reverse to preserve indices
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		attrName := text[match[2]:match[3]]
		// Check if it's a pure number (like {1}) — keep as-is
		if _, err := fmt.Sscanf(attrName, "%d", new(int)); err == nil {
			continue
		}
		attrNames = append([]string{attrName}, attrNames...) // prepend
		paramText = paramText[:match[0]] + fmt.Sprintf("{%d}", len(attrNames)) + paramText[match[1]:]
	}

	// Rebuild paramText with sequential numbering
	paramText = text
	attrNames = nil
	for i := 0; i < len(matches); i++ {
		match := matches[i]
		attrName := text[match[2]:match[3]]
		if _, err := fmt.Sscanf(attrName, "%d", new(int)); err == nil {
			continue
		}
		attrNames = append(attrNames, attrName)
	}
	paramText = re.ReplaceAllStringFunc(text, func(s string) string {
		name := s[1 : len(s)-1]
		if _, err := fmt.Sscanf(name, "%d", new(int)); err == nil {
			return s // keep numeric {1} as-is
		}
		for i, an := range attrNames {
			if an == name {
				return fmt.Sprintf("{%d}", i+1)
			}
		}
		return s
	})

	// Build parameters BSON
	params := bson.A{int32(2)} // version marker for non-empty array
	for _, attrName := range attrNames {
		attrPath := attrName
		if entityContext != "" && !strings.Contains(attrName, ".") {
			attrPath = entityContext + "." + attrName
		}
		params = append(params, bson.D{
			{Key: "$ID", Value: generateBinaryID()},
			{Key: "$Type", Value: "Forms$ClientTemplateParameter"},
			{Key: "AttributeRef", Value: bson.D{
				{Key: "$ID", Value: generateBinaryID()},
				{Key: "$Type", Value: "DomainModels$AttributeRef"},
				{Key: "Attribute", Value: attrPath},
				{Key: "EntityRef", Value: nil},
			}},
			{Key: "Expression", Value: ""},
			{Key: "FormattingInfo", Value: bson.D{
				{Key: "$ID", Value: generateBinaryID()},
				{Key: "$Type", Value: "Forms$FormattingInfo"},
				{Key: "CustomDateFormat", Value: ""},
				{Key: "DateFormat", Value: "Date"},
				{Key: "DecimalPrecision", Value: int64(2)},
				{Key: "EnumFormat", Value: "Text"},
				{Key: "GroupDigits", Value: false},
				{Key: "TimeFormat", Value: "HoursMinutes"},
			}},
			{Key: "SourceVariable", Value: nil},
		})
	}

	makeText := func(t string) bson.D {
		return bson.D{
			{Key: "$ID", Value: generateBinaryID()},
			{Key: "$Type", Value: "Texts$Text"},
			{Key: "Items", Value: bson.A{int32(3), bson.D{
				{Key: "$ID", Value: generateBinaryID()},
				{Key: "$Type", Value: "Texts$Translation"},
				{Key: "LanguageCode", Value: "en_US"},
				{Key: "Text", Value: t},
			}}},
		}
	}

	return bson.D{
		{Key: "$ID", Value: generateBinaryID()},
		{Key: "$Type", Value: "Forms$ClientTemplate"},
		{Key: "Fallback", Value: makeText(paramText)},
		{Key: "Parameters", Value: params},
		{Key: "Template", Value: makeText(paramText)},
	}
}

// createDefaultClientTemplateBSON creates a Forms$ClientTemplate with an en_US translation.
func createDefaultClientTemplateBSON(text string) bson.D {
	makeText := func(t string) bson.D {
		return bson.D{
			{Key: "$ID", Value: generateBinaryID()},
			{Key: "$Type", Value: "Texts$Text"},
			{Key: "Items", Value: bson.A{int32(3), bson.D{
				{Key: "$ID", Value: generateBinaryID()},
				{Key: "$Type", Value: "Texts$Translation"},
				{Key: "LanguageCode", Value: "en_US"},
				{Key: "Text", Value: t},
			}}},
		}
	}
	return bson.D{
		{Key: "$ID", Value: generateBinaryID()},
		{Key: "$Type", Value: "Forms$ClientTemplate"},
		{Key: "Fallback", Value: makeText(text)},
		{Key: "Parameters", Value: bson.A{int32(2)}},
		{Key: "Template", Value: makeText(text)},
	}
}

// generateBinaryID creates a new random 16-byte UUID in Microsoft GUID binary format.
func generateBinaryID() []byte {
	return hexIDToBlob(mpr.GenerateID())
}

// hexIDToBlob converts a hex UUID string to a 16-byte binary blob in Microsoft GUID format.
func hexIDToBlob(hexStr string) []byte {
	hexStr = strings.ReplaceAll(hexStr, "-", "")
	data, err := hex.DecodeString(hexStr)
	if err != nil || len(data) != 16 {
		return data
	}
	// Swap bytes to match Microsoft GUID format (little-endian for first 3 segments)
	data[0], data[1], data[2], data[3] = data[3], data[2], data[1], data[0]
	data[4], data[5] = data[5], data[4]
	data[6], data[7] = data[7], data[6]
	return data
}
