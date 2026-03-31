// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"log"
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
	ActionBSON    bson.D // Serialized client action BSON for opAction
}

// OperationFunc updates a template object's property identified by propertyKey.
// It receives the current object BSON, the property type ID map, the target key,
// and the build context containing resolved values.
type OperationFunc func(obj bson.D, propTypeIDs map[string]pages.PropertyTypeIDEntry, propertyKey string, ctx *BuildContext) bson.D

// OperationRegistry maps operation names to their implementations.
type OperationRegistry struct {
	operations map[string]OperationFunc
}

// NewOperationRegistry creates a registry pre-loaded with the built-in operations.
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
	reg.Register("texttemplate", opTextTemplate)
	reg.Register("action", opAction)
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
	return updateWidgetPropertyValue(obj, propTypeIDs, propertyKey, func(val bson.D) bson.D {
		return setChildWidgets(val, ctx.ChildWidgets)
	})
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

// opTextTemplate sets a text template value on a widget property.
// It replaces the Template.Items in the TextTemplate with a single text item.
func opTextTemplate(obj bson.D, propTypeIDs map[string]pages.PropertyTypeIDEntry, propertyKey string, ctx *BuildContext) bson.D {
	if ctx.PrimitiveVal == "" {
		return obj
	}
	return updateWidgetPropertyValue(obj, propTypeIDs, propertyKey, func(val bson.D) bson.D {
		return setTextTemplateValue(val, ctx.PrimitiveVal)
	})
}

// setTextTemplateValue sets the text content in a TextTemplate WidgetValue field.
func setTextTemplateValue(val bson.D, text string) bson.D {
	result := make(bson.D, 0, len(val))
	for _, elem := range val {
		if elem.Key == "TextTemplate" {
			if tmpl, ok := elem.Value.(bson.D); ok && tmpl != nil {
				result = append(result, bson.E{Key: "TextTemplate", Value: updateTemplateText(tmpl, text)})
			} else {
				// TextTemplate was null — create a new one
				result = append(result, bson.E{Key: "TextTemplate", Value: bson.D{
					{Key: "$ID", Value: mpr.IDToBsonBinary(mpr.GenerateID())},
					{Key: "$Type", Value: "Forms$ClientTemplate"},
					{Key: "Fallback", Value: bson.D{
						{Key: "$ID", Value: mpr.IDToBsonBinary(mpr.GenerateID())},
						{Key: "$Type", Value: "Texts$Text"},
						{Key: "Items", Value: bson.A{int32(3)}},
					}},
					{Key: "Parameters", Value: bson.A{int32(2)}},
					{Key: "Template", Value: bson.D{
						{Key: "$ID", Value: mpr.IDToBsonBinary(mpr.GenerateID())},
						{Key: "$Type", Value: "Texts$Text"},
						{Key: "Items", Value: bson.A{
							int32(3),
							bson.D{
								{Key: "$ID", Value: mpr.IDToBsonBinary(mpr.GenerateID())},
								{Key: "$Type", Value: "Texts$LiteralText"},
								{Key: "Text", Value: text},
							},
						}},
					}},
				}})
			}
		} else {
			result = append(result, elem)
		}
	}
	return result
}

// updateTemplateText updates the Template.Items in a Forms$ClientTemplate with a text value.
func updateTemplateText(tmpl bson.D, text string) bson.D {
	result := make(bson.D, 0, len(tmpl))
	for _, elem := range tmpl {
		if elem.Key == "Template" {
			if template, ok := elem.Value.(bson.D); ok {
				updated := make(bson.D, 0, len(template))
				for _, tElem := range template {
					if tElem.Key == "Items" {
						updated = append(updated, bson.E{Key: "Items", Value: bson.A{
							int32(3),
							bson.D{
								{Key: "$ID", Value: mpr.IDToBsonBinary(mpr.GenerateID())},
								{Key: "$Type", Value: "Texts$LiteralText"},
								{Key: "Text", Value: text},
							},
						}})
					} else {
						updated = append(updated, tElem)
					}
				}
				result = append(result, bson.E{Key: "Template", Value: updated})
			} else {
				result = append(result, elem)
			}
		} else {
			result = append(result, elem)
		}
	}
	return result
}

// opAction sets a client action on a widget property.
func opAction(obj bson.D, propTypeIDs map[string]pages.PropertyTypeIDEntry, propertyKey string, ctx *BuildContext) bson.D {
	if ctx.ActionBSON == nil {
		return obj
	}
	return updateWidgetPropertyValue(obj, propTypeIDs, propertyKey, func(val bson.D) bson.D {
		result := make(bson.D, 0, len(val))
		for _, elem := range val {
			if elem.Key == "Action" {
				result = append(result, bson.E{Key: "Action", Value: ctx.ActionBSON})
			} else {
				result = append(result, elem)
			}
		}
		return result
	})
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

	// 4. Apply child slots
	if err := e.applyChildSlots(slots, w, propertyTypeIDs, &updatedObject); err != nil {
		return nil, err
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

	case "OnClick":
		// Resolve AST action (stored as Properties["Action"]) into serialized BSON
		if action := w.GetAction(); action != nil {
			act, err := e.pageBuilder.buildClientActionV3(action)
			if err != nil {
				return nil, fmt.Errorf("failed to build action: %w", err)
			}
			ctx.ActionBSON = mpr.SerializeClientAction(act)
		}

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
