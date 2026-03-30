// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/mpr"
)

// execAlterPage handles ALTER PAGE/SNIPPET Module.Name { operations }.
func (e *Executor) execAlterPage(s *ast.AlterPageStmt) error {
	if e.reader == nil {
		return fmt.Errorf("not connected to a project")
	}
	if e.writer == nil {
		return fmt.Errorf("project not opened for writing")
	}

	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	var unitID model.ID
	var containerID model.ID
	containerType := s.ContainerType
	if containerType == "" {
		containerType = "PAGE"
	}

	if containerType == "SNIPPET" {
		snippet, modID, err := e.findSnippetByName(s.PageName, h)
		if err != nil {
			return err
		}
		unitID = snippet.ID
		containerID = modID
	} else {
		page, err := e.findPageByName(s.PageName, h)
		if err != nil {
			return err
		}
		unitID = page.ID
		containerID = h.FindModuleID(page.ContainerID)
	}

	// Load raw BSON as ordered document (bson.D preserves field ordering,
	// which is required by Mendix Studio Pro).
	rawBytes, err := e.reader.GetRawUnitBytes(unitID)
	if err != nil {
		return fmt.Errorf("failed to load raw %s data: %w", strings.ToLower(containerType), err)
	}
	var rawData bson.D
	if err := bson.Unmarshal(rawBytes, &rawData); err != nil {
		return fmt.Errorf("failed to unmarshal %s BSON: %w", strings.ToLower(containerType), err)
	}

	// Resolve module name for building new widgets
	modName := h.GetModuleName(containerID)

	// Apply operations sequentially using the appropriate BSON finder
	findWidget := findBsonWidget // page default
	if containerType == "SNIPPET" {
		findWidget = findBsonWidgetInSnippet
	}

	for _, op := range s.Operations {
		switch o := op.(type) {
		case *ast.SetPropertyOp:
			if err := applySetPropertyWith(rawData, o, findWidget); err != nil {
				return fmt.Errorf("SET failed: %w", err)
			}
		case *ast.InsertWidgetOp:
			if err := e.applyInsertWidgetWith(rawData, o, modName, containerID, findWidget); err != nil {
				return fmt.Errorf("INSERT failed: %w", err)
			}
		case *ast.DropWidgetOp:
			if err := applyDropWidgetWith(rawData, o, findWidget); err != nil {
				return fmt.Errorf("DROP failed: %w", err)
			}
		case *ast.ReplaceWidgetOp:
			if err := e.applyReplaceWidgetWith(rawData, o, modName, containerID, findWidget); err != nil {
				return fmt.Errorf("REPLACE failed: %w", err)
			}
		case *ast.AddVariableOp:
			if err := applyAddVariable(&rawData, o); err != nil {
				return fmt.Errorf("ADD VARIABLE failed: %w", err)
			}
		case *ast.DropVariableOp:
			if err := applyDropVariable(rawData, o); err != nil {
				return fmt.Errorf("DROP VARIABLE failed: %w", err)
			}
		case *ast.SetLayoutOp:
			if containerType == "SNIPPET" {
				return fmt.Errorf("SET Layout is not supported for snippets")
			}
			if err := applySetLayout(rawData, o); err != nil {
				return fmt.Errorf("SET Layout failed: %w", err)
			}
		default:
			return fmt.Errorf("unknown ALTER %s operation type: %T", containerType, op)
		}
	}

	// Marshal back to BSON bytes (bson.D preserves field ordering)
	outBytes, err := bson.Marshal(rawData)
	if err != nil {
		return fmt.Errorf("failed to marshal modified %s: %w", strings.ToLower(containerType), err)
	}

	// Save
	if err := e.writer.UpdateRawUnit(string(unitID), outBytes); err != nil {
		return fmt.Errorf("failed to save modified %s: %w", strings.ToLower(containerType), err)
	}

	fmt.Fprintf(e.output, "Altered %s %s\n", strings.ToLower(containerType), s.PageName.String())
	return nil
}

// applySetLayout rewrites the FormCall to reference a new layout.
// It updates the Form field and remaps Parameter strings in each FormCallArgument.
func applySetLayout(rawData bson.D, op *ast.SetLayoutOp) error {
	newLayoutQN := op.NewLayout.Module + "." + op.NewLayout.Name

	// Find FormCall in the page BSON
	var formCall bson.D
	for _, elem := range rawData {
		if elem.Key == "FormCall" {
			if doc, ok := elem.Value.(bson.D); ok {
				formCall = doc
			}
			break
		}
	}
	if formCall == nil {
		return fmt.Errorf("page has no FormCall (layout reference)")
	}

	// Detect the old layout name from existing Parameter values
	oldLayoutQN := ""
	for _, elem := range formCall {
		if elem.Key == "Form" {
			if s, ok := elem.Value.(string); ok && s != "" {
				oldLayoutQN = s
			}
		}
		if elem.Key == "Arguments" {
			if arr, ok := elem.Value.(bson.A); ok {
				for _, item := range arr {
					if doc, ok := item.(bson.D); ok {
						for _, field := range doc {
							if field.Key == "Parameter" {
								if s, ok := field.Value.(string); ok && oldLayoutQN == "" {
									// Extract layout QN from "Atlas_Core.Atlas_TopBar.Main"
									if lastDot := strings.LastIndex(s, "."); lastDot > 0 {
										oldLayoutQN = s[:lastDot]
									}
								}
							}
						}
					}
				}
			}
		}
	}

	if oldLayoutQN == "" {
		return fmt.Errorf("cannot determine current layout from FormCall")
	}

	if oldLayoutQN == newLayoutQN {
		return nil // Already using the target layout
	}

	// Update Form field
	for i, elem := range formCall {
		if elem.Key == "Form" {
			formCall[i].Value = newLayoutQN
		}
	}

	// If Form field doesn't exist, add it
	hasForm := false
	for _, elem := range formCall {
		if elem.Key == "Form" {
			hasForm = true
			break
		}
	}
	if !hasForm {
		// Insert before Arguments
		for i, elem := range formCall {
			if elem.Key == "Arguments" {
				formCall = append(formCall[:i+1], formCall[i:]...)
				formCall[i] = bson.E{Key: "Form", Value: newLayoutQN}
				break
			}
		}
	}

	// Remap Parameter strings in each FormCallArgument
	for _, elem := range formCall {
		if elem.Key != "Arguments" {
			continue
		}
		arr, ok := elem.Value.(bson.A)
		if !ok {
			continue
		}
		for _, item := range arr {
			doc, ok := item.(bson.D)
			if !ok {
				continue
			}
			for j, field := range doc {
				if field.Key != "Parameter" {
					continue
				}
				paramStr, ok := field.Value.(string)
				if !ok {
					continue
				}
				// Extract placeholder name: "Atlas_Core.Atlas_Default.Main" -> "Main"
				placeholder := paramStr
				if strings.HasPrefix(paramStr, oldLayoutQN+".") {
					placeholder = paramStr[len(oldLayoutQN)+1:]
				}

				// Apply explicit mapping if provided
				if op.Mappings != nil {
					if mapped, ok := op.Mappings[placeholder]; ok {
						placeholder = mapped
					}
				}

				// Write new parameter value
				doc[j].Value = newLayoutQN + "." + placeholder
			}
		}
	}

	// Write FormCall back into rawData
	for i, elem := range rawData {
		if elem.Key == "FormCall" {
			rawData[i].Value = formCall
			break
		}
	}

	return nil
}

// ============================================================================
// bson.D helper functions for ordered document access
// ============================================================================

// dGet returns the value for a key in a bson.D, or nil if not found.
func dGet(doc bson.D, key string) any {
	for _, elem := range doc {
		if elem.Key == key {
			return elem.Value
		}
	}
	return nil
}

// dGetDoc returns a nested bson.D field value, or nil.
func dGetDoc(doc bson.D, key string) bson.D {
	v := dGet(doc, key)
	if d, ok := v.(bson.D); ok {
		return d
	}
	return nil
}

// dGetString returns a string field value, or "".
func dGetString(doc bson.D, key string) string {
	v := dGet(doc, key)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// dSet sets a field value in a bson.D in place. If the key exists, it's updated.
func dSet(doc bson.D, key string, value any) {
	for i := range doc {
		if doc[i].Key == key {
			doc[i].Value = value
			return
		}
	}
}

// dGetArrayElements extracts Mendix array elements from a bson.D field value.
// Handles the int32 type marker at index 0. Works with bson.A and []any.
func dGetArrayElements(val any) []any {
	arr := toBsonA(val)
	if len(arr) == 0 {
		return nil
	}
	// Skip type marker (int32) at index 0
	if _, ok := arr[0].(int32); ok {
		return arr[1:]
	}
	if _, ok := arr[0].(int); ok {
		return arr[1:]
	}
	return arr
}

// toBsonA converts various BSON array types to []any.
func toBsonA(v any) []any {
	switch arr := v.(type) {
	case bson.A:
		return []any(arr)
	case []any:
		return arr
	default:
		return nil
	}
}

// dSetArray sets a Mendix-style BSON array field, preserving the int32 marker.
func dSetArray(doc bson.D, key string, elements []any) {
	existing := toBsonA(dGet(doc, key))
	var marker any
	if len(existing) > 0 {
		if _, ok := existing[0].(int32); ok {
			marker = existing[0]
		} else if _, ok := existing[0].(int); ok {
			marker = existing[0]
		}
	}
	var result bson.A
	if marker != nil {
		result = make(bson.A, 0, len(elements)+1)
		result = append(result, marker)
		result = append(result, elements...)
	} else {
		result = make(bson.A, len(elements))
		copy(result, elements)
	}
	dSet(doc, key, result)
}

// extractBinaryIDFromDoc extracts a binary ID string from a bson.D field.
func extractBinaryIDFromDoc(val any) string {
	if bin, ok := val.(primitive.Binary); ok {
		return mpr.BlobToUUID(bin.Data)
	}
	return ""
}

// ============================================================================
// BSON widget tree walking
// ============================================================================

// bsonWidgetResult holds a found widget and its parent context.
type bsonWidgetResult struct {
	widget    bson.D // the widget document itself
	parentArr []any  // the parent array elements (without marker)
	parentKey string // key in the parent doc that holds this array
	parentDoc bson.D // the doc containing parentKey
	index     int    // index in parentArr
}

// widgetFinder is a function type for locating widgets in a raw BSON tree.
type widgetFinder func(rawData bson.D, widgetName string) *bsonWidgetResult

// findBsonWidget searches the raw BSON page tree for a widget by name.
// Page format: FormCall.Arguments[].Widgets[]
func findBsonWidget(rawData bson.D, widgetName string) *bsonWidgetResult {
	formCall := dGetDoc(rawData, "FormCall")
	if formCall == nil {
		return nil
	}

	args := dGetArrayElements(dGet(formCall, "Arguments"))
	for _, arg := range args {
		argDoc, ok := arg.(bson.D)
		if !ok {
			continue
		}
		if result := findInWidgetArray(argDoc, "Widgets", widgetName); result != nil {
			return result
		}
	}
	return nil
}

// findBsonWidgetInSnippet searches the raw BSON snippet tree for a widget by name.
// Snippet format: Widgets[] (Studio Pro) or Widget.Widgets[] (mxcli).
func findBsonWidgetInSnippet(rawData bson.D, widgetName string) *bsonWidgetResult {
	// Studio Pro format: top-level "Widgets" array
	if result := findInWidgetArray(rawData, "Widgets", widgetName); result != nil {
		return result
	}
	// mxcli format: "Widget" (singular) container with "Widgets" inside
	if widgetContainer := dGetDoc(rawData, "Widget"); widgetContainer != nil {
		if result := findInWidgetArray(widgetContainer, "Widgets", widgetName); result != nil {
			return result
		}
	}
	return nil
}

// findInWidgetArray searches a widget array (by key in parentDoc) for a named widget.
func findInWidgetArray(parentDoc bson.D, key string, widgetName string) *bsonWidgetResult {
	elements := dGetArrayElements(dGet(parentDoc, key))
	for i, elem := range elements {
		wDoc, ok := elem.(bson.D)
		if !ok {
			continue
		}
		if dGetString(wDoc, "Name") == widgetName {
			return &bsonWidgetResult{
				widget:    wDoc,
				parentArr: elements,
				parentKey: key,
				parentDoc: parentDoc,
				index:     i,
			}
		}
		// Recurse into children
		if result := findInWidgetChildren(wDoc, widgetName); result != nil {
			return result
		}
	}
	return nil
}

// findInWidgetChildren recursively searches widget children for a named widget.
func findInWidgetChildren(wDoc bson.D, widgetName string) *bsonWidgetResult {
	typeName := dGetString(wDoc, "$Type")

	// Direct Widgets[] children (Container, DataView body, TabPage, GroupBox, etc.)
	if result := findInWidgetArray(wDoc, "Widgets", widgetName); result != nil {
		return result
	}

	// FooterWidgets[] (DataView footer)
	if result := findInWidgetArray(wDoc, "FooterWidgets", widgetName); result != nil {
		return result
	}

	// LayoutGrid: Rows[].Columns[].Widgets[]
	if strings.Contains(typeName, "LayoutGrid") {
		rows := dGetArrayElements(dGet(wDoc, "Rows"))
		for _, row := range rows {
			rowDoc, ok := row.(bson.D)
			if !ok {
				continue
			}
			cols := dGetArrayElements(dGet(rowDoc, "Columns"))
			for _, col := range cols {
				colDoc, ok := col.(bson.D)
				if !ok {
					continue
				}
				if result := findInWidgetArray(colDoc, "Widgets", widgetName); result != nil {
					return result
				}
			}
		}
	}

	// TabContainer: TabPages[].Widgets[]
	if result := findInTabPages(wDoc, widgetName); result != nil {
		return result
	}

	// ControlBar widgets
	if result := findInControlBar(wDoc, widgetName); result != nil {
		return result
	}

	// CustomWidget (pluggable): Object.Properties[].Value.Widgets[]
	if strings.Contains(typeName, "CustomWidget") {
		if obj := dGetDoc(wDoc, "Object"); obj != nil {
			props := dGetArrayElements(dGet(obj, "Properties"))
			for _, prop := range props {
				propDoc, ok := prop.(bson.D)
				if !ok {
					continue
				}
				if valDoc := dGetDoc(propDoc, "Value"); valDoc != nil {
					if result := findInWidgetArray(valDoc, "Widgets", widgetName); result != nil {
						return result
					}
				}
			}
		}
	}

	return nil
}

// findInTabPages searches TabPages[].Widgets[] for a named widget.
func findInTabPages(wDoc bson.D, widgetName string) *bsonWidgetResult {
	tabPages := dGetArrayElements(dGet(wDoc, "TabPages"))
	for _, tp := range tabPages {
		tpDoc, ok := tp.(bson.D)
		if !ok {
			continue
		}
		if result := findInWidgetArray(tpDoc, "Widgets", widgetName); result != nil {
			return result
		}
	}
	return nil
}

// findInControlBar searches ControlBarItems within a ControlBar for a named widget.
func findInControlBar(wDoc bson.D, widgetName string) *bsonWidgetResult {
	controlBar := dGetDoc(wDoc, "ControlBar")
	if controlBar == nil {
		return nil
	}
	return findInWidgetArray(controlBar, "Items", widgetName)
}

// ============================================================================
// SET property
// ============================================================================

// applySetProperty modifies widget properties in the raw BSON tree (page format).
func applySetProperty(rawData bson.D, op *ast.SetPropertyOp) error {
	return applySetPropertyWith(rawData, op, findBsonWidget)
}

// applySetPropertyWith modifies widget properties using the given widget finder.
func applySetPropertyWith(rawData bson.D, op *ast.SetPropertyOp, find widgetFinder) error {
	if op.WidgetName == "" {
		// Page/snippet-level SET
		return applyPageLevelSet(rawData, op.Properties)
	}

	// Find the widget
	result := find(rawData, op.WidgetName)
	if result == nil {
		return fmt.Errorf("widget %q not found", op.WidgetName)
	}

	// Apply each property
	for propName, value := range op.Properties {
		if err := setRawWidgetProperty(result.widget, propName, value); err != nil {
			return fmt.Errorf("failed to set %s on %s: %w", propName, op.WidgetName, err)
		}
	}
	return nil
}

// applyPageLevelSet handles page-level SET (e.g., SET Title = 'New Title').
func applyPageLevelSet(rawData bson.D, properties map[string]interface{}) error {
	for propName, value := range properties {
		switch propName {
		case "Title":
			// Title is stored as FormCall.Title or at the top level
			if formCall := dGetDoc(rawData, "FormCall"); formCall != nil {
				setTranslatableText(formCall, "Title", value)
			} else {
				setTranslatableText(rawData, "Title", value)
			}
		default:
			return fmt.Errorf("unsupported page-level property: %s", propName)
		}
	}
	return nil
}

// setRawWidgetProperty sets a property on a raw BSON widget document.
func setRawWidgetProperty(widget bson.D, propName string, value interface{}) error {
	// Handle known standard BSON properties
	switch propName {
	case "Caption":
		return setWidgetCaption(widget, value)
	case "Content":
		return setWidgetContent(widget, value)
	case "Label":
		return setWidgetLabel(widget, value)
	case "ButtonStyle":
		if s, ok := value.(string); ok {
			dSet(widget, "ButtonStyle", s)
		}
		return nil
	case "Class":
		if appearance := dGetDoc(widget, "Appearance"); appearance != nil {
			if s, ok := value.(string); ok {
				dSet(appearance, "Class", s)
			}
		}
		return nil
	case "Style":
		if appearance := dGetDoc(widget, "Appearance"); appearance != nil {
			if s, ok := value.(string); ok {
				dSet(appearance, "Style", s)
			}
		}
		return nil
	case "Editable":
		if s, ok := value.(string); ok {
			dSet(widget, "Editable", s)
		}
		return nil
	case "Visible":
		if s, ok := value.(string); ok {
			dSet(widget, "Visible", s)
		} else if b, ok := value.(bool); ok {
			if b {
				dSet(widget, "Visible", "True")
			} else {
				dSet(widget, "Visible", "False")
			}
		}
		return nil
	case "Name":
		if s, ok := value.(string); ok {
			dSet(widget, "Name", s)
		}
		return nil
	case "Attribute":
		return setWidgetAttributeRef(widget, value)
	case "DataSource":
		return setWidgetDataSource(widget, value)
	default:
		// Try as pluggable widget property (quoted string property name)
		return setPluggableWidgetProperty(widget, propName, value)
	}
}

// setWidgetCaption sets the Caption property on a button or text widget.
func setWidgetCaption(widget bson.D, value interface{}) error {
	caption := dGetDoc(widget, "Caption")
	if caption == nil {
		// Try direct caption text
		setTranslatableText(widget, "Caption", value)
		return nil
	}
	setTranslatableText(caption, "", value)
	return nil
}

// setWidgetAttributeRef sets or updates the AttributeRef on an input widget.
// The value must be a fully qualified path (Module.Entity.Attribute, 2+ dots).
// If not fully qualified, AttributeRef is set to nil to avoid Studio Pro crash.
func setWidgetAttributeRef(widget bson.D, value interface{}) error {
	attrPath, ok := value.(string)
	if !ok {
		return fmt.Errorf("Attribute value must be a string")
	}

	// Build the new AttributeRef value
	var attrRefValue interface{}
	if strings.Count(attrPath, ".") >= 2 {
		attrRefValue = bson.D{
			{Key: "$ID", Value: mpr.IDToBsonBinary(mpr.GenerateID())},
			{Key: "$Type", Value: "DomainModels$AttributeRef"},
			{Key: "Attribute", Value: attrPath},
			{Key: "EntityRef", Value: nil},
		}
	} else {
		// Not fully qualified — clear the ref to avoid Mendix crash
		attrRefValue = nil
	}

	// Try to update existing AttributeRef field
	for i, elem := range widget {
		if elem.Key == "AttributeRef" {
			widget[i].Value = attrRefValue
			return nil
		}
	}

	// No existing AttributeRef field — this widget may not support it
	return fmt.Errorf("widget does not have an AttributeRef property; Attribute can only be SET on input widgets (TextBox, TextArea, DatePicker, etc.)")
}

// setWidgetDataSource sets the DataSource on a DataView or list widget.
func setWidgetDataSource(widget bson.D, value interface{}) error {
	ds, ok := value.(*ast.DataSourceV3)
	if !ok {
		return fmt.Errorf("DataSource value must be a datasource expression")
	}

	var serialized interface{}

	switch ds.Type {
	case "selection":
		// SELECTION widgetName → Forms$ListenTargetSource
		serialized = bson.D{
			{Key: "$ID", Value: mpr.IDToBsonBinary(mpr.GenerateID())},
			{Key: "$Type", Value: "Forms$ListenTargetSource"},
			{Key: "ListenTarget", Value: ds.Reference},
		}
	case "database":
		// DATABASE Entity → Forms$DataViewSource with entity ref
		var entityRef interface{}
		if ds.Reference != "" {
			entityRef = bson.D{
				{Key: "$ID", Value: mpr.IDToBsonBinary(mpr.GenerateID())},
				{Key: "$Type", Value: "DomainModels$DirectEntityRef"},
				{Key: "Entity", Value: ds.Reference},
			}
		}
		serialized = bson.D{
			{Key: "$ID", Value: mpr.IDToBsonBinary(mpr.GenerateID())},
			{Key: "$Type", Value: "Forms$DataViewSource"},
			{Key: "EntityRef", Value: entityRef},
			{Key: "ForceFullObjects", Value: false},
			{Key: "SourceVariable", Value: nil},
		}
	case "microflow":
		serialized = bson.D{
			{Key: "$ID", Value: mpr.IDToBsonBinary(mpr.GenerateID())},
			{Key: "$Type", Value: "Forms$MicroflowSource"},
			{Key: "MicroflowSettings", Value: bson.D{
				{Key: "$ID", Value: mpr.IDToBsonBinary(mpr.GenerateID())},
				{Key: "$Type", Value: "Forms$MicroflowSettings"},
				{Key: "Asynchronous", Value: false},
				{Key: "ConfirmationInfo", Value: nil},
				{Key: "FormValidations", Value: "All"},
				{Key: "Microflow", Value: ds.Reference},
				{Key: "ParameterMappings", Value: bson.A{int32(3)}},
				{Key: "ProgressBar", Value: "None"},
				{Key: "ProgressMessage", Value: nil},
			}},
		}
	case "nanoflow":
		serialized = bson.D{
			{Key: "$ID", Value: mpr.IDToBsonBinary(mpr.GenerateID())},
			{Key: "$Type", Value: "Forms$NanoflowSource"},
			{Key: "NanoflowSettings", Value: bson.D{
				{Key: "$ID", Value: mpr.IDToBsonBinary(mpr.GenerateID())},
				{Key: "$Type", Value: "Forms$NanoflowSettings"},
				{Key: "Nanoflow", Value: ds.Reference},
				{Key: "ParameterMappings", Value: bson.A{int32(3)}},
			}},
		}
	default:
		return fmt.Errorf("unsupported DataSource type for ALTER PAGE SET: %s", ds.Type)
	}

	dSet(widget, "DataSource", serialized)
	return nil
}

// setWidgetLabel sets the Label.Caption text on input widgets.
func setWidgetLabel(widget bson.D, value interface{}) error {
	label := dGetDoc(widget, "Label")
	if label == nil {
		return nil
	}
	setTranslatableText(label, "Caption", value)
	return nil
}

// setWidgetContent sets the Content property on a DYNAMICTEXT widget.
// Content is stored as Forms$ClientTemplate → Template (Forms$Text) → Items[] → Translation{Text}.
// This mirrors extractTextContent which reads Content.Template.Items[].Text.
func setWidgetContent(widget bson.D, value interface{}) error {
	strVal, ok := value.(string)
	if !ok {
		return fmt.Errorf("Content value must be a string")
	}
	content := dGetDoc(widget, "Content")
	if content == nil {
		return fmt.Errorf("widget has no Content property")
	}
	template := dGetDoc(content, "Template")
	if template == nil {
		return fmt.Errorf("Content has no Template")
	}
	items := dGetArrayElements(dGet(template, "Items"))
	if len(items) > 0 {
		if itemDoc, ok := items[0].(bson.D); ok {
			dSet(itemDoc, "Text", strVal)
			return nil
		}
	}
	return fmt.Errorf("Content.Template has no Items with Text")
}

// setTranslatableText sets a translatable text value in BSON.
// If key is empty, modifies the doc directly; otherwise navigates to doc[key].
func setTranslatableText(parent bson.D, key string, value interface{}) {
	strVal, ok := value.(string)
	if !ok {
		return
	}

	target := parent
	if key != "" {
		if nested := dGetDoc(parent, key); nested != nil {
			target = nested
		} else {
			// Try to set directly
			dSet(parent, key, strVal)
			return
		}
	}

	// Navigate to Translations[].Text
	translations := dGetArrayElements(dGet(target, "Translations"))
	if len(translations) > 0 {
		if tDoc, ok := translations[0].(bson.D); ok {
			dSet(tDoc, "Text", strVal)
			return
		}
	}

	// Direct text value
	dSet(target, "Text", strVal)
}

// setPluggableWidgetProperty sets a property on a pluggable widget's Object.Properties[].
// Properties are identified by TypePointer referencing a PropertyType entry in the widget's
// Type.ObjectType.PropertyTypes array, NOT by a "Key" field on the property itself.
func setPluggableWidgetProperty(widget bson.D, propName string, value interface{}) error {
	obj := dGetDoc(widget, "Object")
	if obj == nil {
		return fmt.Errorf("property %q not found (widget has no pluggable Object)", propName)
	}

	// Build TypePointer ID -> PropertyKey map from Type.ObjectType.PropertyTypes
	propTypeKeyMap := make(map[string]string)
	if widgetType := dGetDoc(widget, "Type"); widgetType != nil {
		if objType := dGetDoc(widgetType, "ObjectType"); objType != nil {
			propTypes := dGetArrayElements(dGet(objType, "PropertyTypes"))
			for _, pt := range propTypes {
				ptDoc, ok := pt.(bson.D)
				if !ok {
					continue
				}
				key := dGetString(ptDoc, "PropertyKey")
				if key == "" {
					continue
				}
				id := extractBinaryIDFromDoc(dGet(ptDoc, "$ID"))
				if id != "" {
					propTypeKeyMap[id] = key
				}
			}
		}
	}

	props := dGetArrayElements(dGet(obj, "Properties"))
	for _, prop := range props {
		propDoc, ok := prop.(bson.D)
		if !ok {
			continue
		}
		// Resolve property key via TypePointer
		typePointerID := extractBinaryIDFromDoc(dGet(propDoc, "TypePointer"))
		propKey := propTypeKeyMap[typePointerID]
		if propKey != propName {
			continue
		}
		// Set the value
		if valDoc := dGetDoc(propDoc, "Value"); valDoc != nil {
			switch v := value.(type) {
			case string:
				dSet(valDoc, "PrimitiveValue", v)
			case bool:
				if v {
					dSet(valDoc, "PrimitiveValue", "yes")
				} else {
					dSet(valDoc, "PrimitiveValue", "no")
				}
			case int:
				dSet(valDoc, "PrimitiveValue", fmt.Sprintf("%d", v))
			case float64:
				dSet(valDoc, "PrimitiveValue", fmt.Sprintf("%g", v))
			default:
				dSet(valDoc, "PrimitiveValue", fmt.Sprintf("%v", v))
			}
			return nil
		}
		return fmt.Errorf("property %q has no Value map", propName)
	}
	return fmt.Errorf("pluggable property %q not found in widget Object", propName)
}

// ============================================================================
// INSERT widget
// ============================================================================

// applyInsertWidget inserts new widgets before or after a target widget (page format).
func (e *Executor) applyInsertWidget(rawData bson.D, op *ast.InsertWidgetOp, moduleName string, moduleID model.ID) error {
	return e.applyInsertWidgetWith(rawData, op, moduleName, moduleID, findBsonWidget)
}

// applyInsertWidgetWith inserts new widgets using the given widget finder.
func (e *Executor) applyInsertWidgetWith(rawData bson.D, op *ast.InsertWidgetOp, moduleName string, moduleID model.ID, find widgetFinder) error {
	result := find(rawData, op.TargetName)
	if result == nil {
		return fmt.Errorf("widget %q not found", op.TargetName)
	}

	// Check for duplicate widget names before building
	for _, w := range op.Widgets {
		if w.Name != "" && find(rawData, w.Name) != nil {
			return fmt.Errorf("duplicate widget name '%s': a widget with this name already exists on the page", w.Name)
		}
	}

	// Find entity context from enclosing DataView/DataGrid/ListView
	entityCtx := findEnclosingEntityContext(rawData, op.TargetName)

	// Build new widget BSON from AST
	newBsonWidgets, err := e.buildWidgetsBson(op.Widgets, moduleName, moduleID, entityCtx)
	if err != nil {
		return fmt.Errorf("failed to build widgets: %w", err)
	}

	// Calculate insertion index
	insertIdx := result.index
	if op.Position == "AFTER" {
		insertIdx = result.index + 1
	}

	// Insert into the parent array
	newArr := make([]any, 0, len(result.parentArr)+len(newBsonWidgets))
	newArr = append(newArr, result.parentArr[:insertIdx]...)
	newArr = append(newArr, newBsonWidgets...)
	newArr = append(newArr, result.parentArr[insertIdx:]...)

	// Update parent
	dSetArray(result.parentDoc, result.parentKey, newArr)

	return nil
}

// ============================================================================
// DROP widget
// ============================================================================

// applyDropWidget removes widgets from the raw BSON tree (page format).
func applyDropWidget(rawData bson.D, op *ast.DropWidgetOp) error {
	return applyDropWidgetWith(rawData, op, findBsonWidget)
}

// applyDropWidgetWith removes widgets using the given widget finder.
func applyDropWidgetWith(rawData bson.D, op *ast.DropWidgetOp, find widgetFinder) error {
	for _, name := range op.WidgetNames {
		result := find(rawData, name)
		if result == nil {
			return fmt.Errorf("widget %q not found", name)
		}

		// Remove from parent array
		newArr := make([]any, 0, len(result.parentArr)-1)
		newArr = append(newArr, result.parentArr[:result.index]...)
		newArr = append(newArr, result.parentArr[result.index+1:]...)

		// Update parent
		dSetArray(result.parentDoc, result.parentKey, newArr)
	}
	return nil
}

// ============================================================================
// REPLACE widget
// ============================================================================

// applyReplaceWidget replaces a widget with new widgets (page format).
func (e *Executor) applyReplaceWidget(rawData bson.D, op *ast.ReplaceWidgetOp, moduleName string, moduleID model.ID) error {
	return e.applyReplaceWidgetWith(rawData, op, moduleName, moduleID, findBsonWidget)
}

// applyReplaceWidgetWith replaces a widget using the given widget finder.
func (e *Executor) applyReplaceWidgetWith(rawData bson.D, op *ast.ReplaceWidgetOp, moduleName string, moduleID model.ID, find widgetFinder) error {
	result := find(rawData, op.WidgetName)
	if result == nil {
		return fmt.Errorf("widget %q not found", op.WidgetName)
	}

	// Check for duplicate widget names (skip the widget being replaced)
	for _, w := range op.NewWidgets {
		if w.Name != "" && w.Name != op.WidgetName && find(rawData, w.Name) != nil {
			return fmt.Errorf("duplicate widget name '%s': a widget with this name already exists on the page", w.Name)
		}
	}

	// Find entity context from enclosing DataView/DataGrid/ListView
	entityCtx := findEnclosingEntityContext(rawData, op.WidgetName)

	// Build new widget BSON from AST
	newBsonWidgets, err := e.buildWidgetsBson(op.NewWidgets, moduleName, moduleID, entityCtx)
	if err != nil {
		return fmt.Errorf("failed to build replacement widgets: %w", err)
	}

	// Replace: remove old widget, insert new ones at same position
	newArr := make([]any, 0, len(result.parentArr)-1+len(newBsonWidgets))
	newArr = append(newArr, result.parentArr[:result.index]...)
	newArr = append(newArr, newBsonWidgets...)
	newArr = append(newArr, result.parentArr[result.index+1:]...)

	// Update parent
	dSetArray(result.parentDoc, result.parentKey, newArr)

	return nil
}

// ============================================================================
// Entity context extraction from BSON tree
// ============================================================================

// findEnclosingEntityContext walks the raw BSON tree to find the DataView, DataGrid,
// ListView, or Gallery ancestor of a target widget and extracts the entity name.
// This is needed for INSERT/REPLACE operations so that input widget Binds can be
// resolved to fully qualified attribute paths.
func findEnclosingEntityContext(rawData bson.D, widgetName string) string {
	// Start from FormCall.Arguments[].Widgets[] (page format)
	if formCall := dGetDoc(rawData, "FormCall"); formCall != nil {
		args := dGetArrayElements(dGet(formCall, "Arguments"))
		for _, arg := range args {
			argDoc, ok := arg.(bson.D)
			if !ok {
				continue
			}
			if ctx := findEntityContextInWidgets(argDoc, "Widgets", widgetName, ""); ctx != "" {
				return ctx
			}
		}
	}
	// Snippet format: Widgets[] or Widget.Widgets[]
	if ctx := findEntityContextInWidgets(rawData, "Widgets", widgetName, ""); ctx != "" {
		return ctx
	}
	if widgetContainer := dGetDoc(rawData, "Widget"); widgetContainer != nil {
		if ctx := findEntityContextInWidgets(widgetContainer, "Widgets", widgetName, ""); ctx != "" {
			return ctx
		}
	}
	return ""
}

// findEntityContextInWidgets searches a widget array for the target widget,
// tracking entity context from DataView/DataGrid/ListView/Gallery ancestors.
func findEntityContextInWidgets(parentDoc bson.D, key string, widgetName string, currentEntity string) string {
	elements := dGetArrayElements(dGet(parentDoc, key))
	for _, elem := range elements {
		wDoc, ok := elem.(bson.D)
		if !ok {
			continue
		}
		if dGetString(wDoc, "Name") == widgetName {
			return currentEntity
		}
		// Update entity context if this is a data container
		entityCtx := currentEntity
		if ent := extractEntityFromDataSource(wDoc); ent != "" {
			entityCtx = ent
		}
		// Recurse into children
		if ctx := findEntityContextInChildren(wDoc, widgetName, entityCtx); ctx != "" {
			return ctx
		}
	}
	return ""
}

// findEntityContextInChildren recursively searches widget children for the target,
// tracking entity context. Mirrors the traversal logic of findInWidgetChildren.
func findEntityContextInChildren(wDoc bson.D, widgetName string, currentEntity string) string {
	typeName := dGetString(wDoc, "$Type")

	// Direct Widgets[] children
	if ctx := findEntityContextInWidgets(wDoc, "Widgets", widgetName, currentEntity); ctx != "" {
		return ctx
	}
	// FooterWidgets[]
	if ctx := findEntityContextInWidgets(wDoc, "FooterWidgets", widgetName, currentEntity); ctx != "" {
		return ctx
	}
	// LayoutGrid: Rows[].Columns[].Widgets[]
	if strings.Contains(typeName, "LayoutGrid") {
		rows := dGetArrayElements(dGet(wDoc, "Rows"))
		for _, row := range rows {
			rowDoc, ok := row.(bson.D)
			if !ok {
				continue
			}
			cols := dGetArrayElements(dGet(rowDoc, "Columns"))
			for _, col := range cols {
				colDoc, ok := col.(bson.D)
				if !ok {
					continue
				}
				if ctx := findEntityContextInWidgets(colDoc, "Widgets", widgetName, currentEntity); ctx != "" {
					return ctx
				}
			}
		}
	}
	// TabContainer: TabPages[].Widgets[]
	tabPages := dGetArrayElements(dGet(wDoc, "TabPages"))
	for _, tp := range tabPages {
		tpDoc, ok := tp.(bson.D)
		if !ok {
			continue
		}
		if ctx := findEntityContextInWidgets(tpDoc, "Widgets", widgetName, currentEntity); ctx != "" {
			return ctx
		}
	}
	// ControlBar
	if controlBar := dGetDoc(wDoc, "ControlBar"); controlBar != nil {
		if ctx := findEntityContextInWidgets(controlBar, "Items", widgetName, currentEntity); ctx != "" {
			return ctx
		}
	}
	// CustomWidget (pluggable): Object.Properties[].Value.Widgets[]
	if strings.Contains(typeName, "CustomWidget") {
		if obj := dGetDoc(wDoc, "Object"); obj != nil {
			props := dGetArrayElements(dGet(obj, "Properties"))
			for _, prop := range props {
				propDoc, ok := prop.(bson.D)
				if !ok {
					continue
				}
				if valDoc := dGetDoc(propDoc, "Value"); valDoc != nil {
					if ctx := findEntityContextInWidgets(valDoc, "Widgets", widgetName, currentEntity); ctx != "" {
						return ctx
					}
				}
			}
		}
	}
	return ""
}

// extractEntityFromDataSource extracts the entity qualified name from a widget's
// DataSource BSON. Handles DataView, DataGrid, ListView, and Gallery data sources.
func extractEntityFromDataSource(wDoc bson.D) string {
	ds := dGetDoc(wDoc, "DataSource")
	if ds == nil {
		return ""
	}
	// EntityRef.Entity contains the qualified name (e.g., "Module.Entity")
	if entityRef := dGetDoc(ds, "EntityRef"); entityRef != nil {
		if entity := dGetString(entityRef, "Entity"); entity != "" {
			return entity
		}
	}
	return ""
}

// ============================================================================
// ADD / DROP variable
// ============================================================================

// applyAddVariable adds a new LocalVariable to the raw BSON page/snippet.
func applyAddVariable(rawData *bson.D, op *ast.AddVariableOp) error {
	// Check for duplicate variable name
	existingVars := dGetArrayElements(dGet(*rawData, "Variables"))
	for _, ev := range existingVars {
		if evDoc, ok := ev.(bson.D); ok {
			if dGetString(evDoc, "Name") == op.Variable.Name {
				return fmt.Errorf("variable $%s already exists", op.Variable.Name)
			}
		}
	}

	// Build VariableType BSON
	varTypeID := mpr.GenerateID()
	bsonTypeName := mdlTypeToBsonType(op.Variable.DataType)
	varType := bson.D{
		{Key: "$ID", Value: mpr.IDToBsonBinary(varTypeID)},
		{Key: "$Type", Value: bsonTypeName},
	}
	if bsonTypeName == "DataTypes$ObjectType" {
		varType = append(varType, bson.E{Key: "Entity", Value: op.Variable.DataType})
	}

	// Build LocalVariable BSON document
	varID := mpr.GenerateID()
	varDoc := bson.D{
		{Key: "$ID", Value: mpr.IDToBsonBinary(varID)},
		{Key: "$Type", Value: "Forms$LocalVariable"},
		{Key: "DefaultValue", Value: op.Variable.DefaultValue},
		{Key: "Name", Value: op.Variable.Name},
		{Key: "VariableType", Value: varType},
	}

	// Append to existing Variables array, or create new field
	existing := toBsonA(dGet(*rawData, "Variables"))
	if existing != nil {
		elements := dGetArrayElements(dGet(*rawData, "Variables"))
		elements = append(elements, varDoc)
		dSetArray(*rawData, "Variables", elements)
	} else {
		// Field doesn't exist — append to the document
		*rawData = append(*rawData, bson.E{Key: "Variables", Value: bson.A{int32(3), varDoc}})
	}

	return nil
}

// applyDropVariable removes a LocalVariable from the raw BSON page/snippet.
func applyDropVariable(rawData bson.D, op *ast.DropVariableOp) error {
	elements := dGetArrayElements(dGet(rawData, "Variables"))
	if elements == nil {
		return fmt.Errorf("variable $%s not found", op.VariableName)
	}

	// Find and remove the variable
	found := false
	var kept []any
	for _, elem := range elements {
		if doc, ok := elem.(bson.D); ok {
			if dGetString(doc, "Name") == op.VariableName {
				found = true
				continue
			}
		}
		kept = append(kept, elem)
	}

	if !found {
		return fmt.Errorf("variable $%s not found", op.VariableName)
	}

	dSetArray(rawData, "Variables", kept)
	return nil
}

// ============================================================================
// Widget BSON building
// ============================================================================

// buildWidgetsBson converts AST widgets to ordered BSON documents.
// Returns bson.D elements (not map[string]any) to preserve field ordering.
func (e *Executor) buildWidgetsBson(widgets []*ast.WidgetV3, moduleName string, moduleID model.ID, entityContext string) ([]any, error) {
	pb := &pageBuilder{
		writer:           e.writer,
		reader:           e.reader,
		moduleID:         moduleID,
		moduleName:       moduleName,
		entityContext:    entityContext,
		widgetScope:      make(map[string]model.ID),
		paramScope:       make(map[string]model.ID),
		paramEntityNames: make(map[string]string),
		execCache:        e.cache,
		fragments:        e.fragments,
		themeRegistry:    e.getThemeRegistry(),
	}

	var result []any
	for _, w := range widgets {
		bsonD, err := pb.buildWidgetV3ToBSON(w)
		if err != nil {
			return nil, fmt.Errorf("failed to build widget %s: %w", w.Name, err)
		}
		if bsonD == nil {
			continue
		}

		// Keep as bson.D (ordered document) - no conversion to map[string]any needed.
		// This preserves field ordering when marshaled back to BSON bytes.
		result = append(result, bsonD)
	}
	return result, nil
}

// ============================================================================
// Helper: SerializeWidget is already available via mpr package
// ============================================================================

var _ = mpr.SerializeWidget // ensure import is used
