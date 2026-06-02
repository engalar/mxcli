// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

// ---------------------------------------------------------------------------
// GetPageModel / GetSnippetModel / GetLayoutModel
// ---------------------------------------------------------------------------

func (b *MprBackend) GetPageModel(id model.ID) (*types.PageModel, error) {
	return b.loadPageModel(id, "page")
}

func (b *MprBackend) GetSnippetModel(id model.ID) (*types.PageModel, error) {
	return b.loadPageModel(id, "snippet")
}

func (b *MprBackend) GetLayoutModel(id model.ID) (*types.PageModel, error) {
	return b.loadPageModel(id, "layout")
}

func (b *MprBackend) loadPageModel(id model.ID, kind string) (*types.PageModel, error) {
	_ = kind
	raw, err := b.msdkReader.GetRawUnitBytes(string(id))
	if err != nil {
		return nil, fmt.Errorf("load unit bytes: %w", err)
	}
	var doc bson.D
	if err := bson.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal BSON: %w", err)
	}
	return pageDocToModel(doc), nil
}

// ---------------------------------------------------------------------------
// pageDocToModel: top-level page/snippet/layout BSON → PageModel
// ---------------------------------------------------------------------------

func pageDocToModel(doc bson.D) *types.PageModel {
	pm := &types.PageModel{}

	callDoc := dGetDoc(doc, "LayoutCall")
	if callDoc == nil {
		callDoc = dGetDoc(doc, "FormCall")
	}

	if callDoc != nil {
		if layoutRef := dGetString(callDoc, "LayoutQualifiedName"); layoutRef != "" {
			pm.Layout = layoutRef
		}

		args := dGetArrayElements(dGet(callDoc, "Arguments"))
		for _, arg := range args {
			argDoc, ok := arg.(bson.D)
			if !ok {
				continue
			}
			if widgetDoc := dGetDoc(argDoc, "Widget"); widgetDoc != nil {
				widgets := widgetsFromDivContainer(widgetDoc)
				pm.Widgets = append(pm.Widgets, widgets...)
			} else {
				for _, w := range dGetArrayElements(dGet(argDoc, "Widgets")) {
					if wd, ok := w.(bson.D); ok {
						if node := widgetNodeFromBSON(wd); node != nil {
							pm.Widgets = append(pm.Widgets, node)
						}
					}
				}
			}
		}
	}

	return pm
}

// widgetsFromDivContainer unwraps a conditionalVisibilityWidget DivContainer.
// The gen-encoded write path wraps page content in a transparent DivContainer
// whose name starts with "conditionalVisibilityWidget". We unwrap it so the
// describe output omits the phantom container.
func widgetsFromDivContainer(doc bson.D) []*types.WidgetNode {
	name := dGetString(doc, "Name")
	typeName := dGetString(doc, "$Type")
	isWrapper := strings.HasPrefix(name, "conditionalVisibilityWidget") &&
		(typeName == "Pages$DivContainer" || typeName == "Forms$DivContainer")

	if isWrapper {
		var nodes []*types.WidgetNode
		for _, w := range dGetArrayElements(dGet(doc, "Widgets")) {
			if wd, ok := w.(bson.D); ok {
				if node := widgetNodeFromBSON(wd); node != nil {
					nodes = append(nodes, node)
				}
			}
		}
		return nodes
	}
	if node := widgetNodeFromBSON(doc); node != nil {
		return []*types.WidgetNode{node}
	}
	return nil
}

// ---------------------------------------------------------------------------
// BSON $type → WidgetKind mapping
// ---------------------------------------------------------------------------

var bsonTypeToKind = map[string]types.WidgetKind{
	"Forms$DivContainer":         types.WidgetContainer,
	"Pages$DivContainer":         types.WidgetContainer,
	"Forms$ScrollContainer":      types.WidgetScrollView,
	"Pages$ScrollContainer":      types.WidgetScrollView,
	"Forms$GroupBox":             types.WidgetGroupBox,
	"Pages$GroupBox":             types.WidgetGroupBox,
	"Forms$LayoutGrid":           types.WidgetLayoutGrid,
	"Pages$LayoutGrid":           types.WidgetLayoutGrid,
	"Forms$LayoutGridRow":        types.WidgetLayoutRow,
	"Pages$LayoutGridRow":        types.WidgetLayoutRow,
	"Forms$LayoutGridColumn":     types.WidgetLayoutCol,
	"Pages$LayoutGridColumn":     types.WidgetLayoutCol,
	"Forms$TabControl":           types.WidgetTabContainer,
	"Pages$TabControl":           types.WidgetTabContainer,
	"Forms$TabPage":              types.WidgetTabPage,
	"Pages$TabPage":              types.WidgetTabPage,
	"Forms$DataView":             types.WidgetDataView,
	"Pages$DataView":             types.WidgetDataView,
	"Forms$ListView":             types.WidgetListView,
	"Pages$ListView":             types.WidgetListView,
	"Forms$Gallery":              types.WidgetGallery,
	"Pages$Gallery":              types.WidgetGallery,
	"Forms$ActionButton":         types.WidgetButton,
	"Pages$ActionButton":         types.WidgetButton,
	"Forms$TextBox":              types.WidgetTextBox,
	"Pages$TextBox":              types.WidgetTextBox,
	"Forms$TextArea":             types.WidgetTextArea,
	"Pages$TextArea":             types.WidgetTextArea,
	"Forms$DatePicker":           types.WidgetDatePicker,
	"Pages$DatePicker":           types.WidgetDatePicker,
	"Forms$RadioButtons":         types.WidgetRadioButtons,
	"Pages$RadioButtons":         types.WidgetRadioButtons,
	"Forms$CheckBox":             types.WidgetCheckBox,
	"Pages$CheckBox":             types.WidgetCheckBox,
	"Forms$Label":                types.WidgetLabel,
	"Pages$Label":                types.WidgetLabel,
	"Forms$Text":                 types.WidgetText,
	"Pages$Text":                 types.WidgetText,
	"Forms$DynamicText":          types.WidgetDynamicText,
	"Pages$DynamicText":          types.WidgetDynamicText,
	"Forms$Title":                types.WidgetTitle,
	"Pages$Title":                types.WidgetTitle,
	"Forms$NavigationList":       types.WidgetNavList,
	"Pages$NavigationList":       types.WidgetNavList,
	"Forms$SnippetCallWidget":    types.WidgetSnippet,
	"Pages$SnippetCallWidget":    types.WidgetSnippet,
	"CustomWidgets$CustomWidget": types.WidgetUnknown, // refined below
}

// widgetNodeFromBSON converts a single raw BSON widget document to WidgetNode.
func widgetNodeFromBSON(doc bson.D) *types.WidgetNode {
	typeName := dGetString(doc, "$Type")
	name := dGetString(doc, "Name")

	kind, ok := bsonTypeToKind[typeName]
	if !ok {
		return nil
	}

	if typeName == "CustomWidgets$CustomWidget" {
		kind = customWidgetKind(doc)
	}

	node := &types.WidgetNode{Kind: kind, Name: name}

	if app := dGetDoc(doc, "Appearance"); app != nil {
		node.Class = dGetString(app, "CSSClasses")
		node.Style = dGetString(app, "Style")
		node.DesignProps = extractDesignPropsFromBSON(app)
	}

	if cv := dGetDoc(doc, "ConditionalVisibilitySettings"); cv != nil {
		node.VisibleIf = dGetString(cv, "Expression")
	}

	switch kind {
	case types.WidgetContainer, types.WidgetScrollView:
		node.Children = extractChildWidgets(doc, "Widgets")
	case types.WidgetLayoutGrid:
		node.Children = extractLayoutGridRows(doc)
	case types.WidgetLayoutRow:
		node.Children = extractChildWidgets(doc, "Columns")
	case types.WidgetLayoutCol:
		node.ColWidth = extractColWidth(doc)
		node.Children = extractChildWidgets(doc, "Widgets")
	case types.WidgetGroupBox:
		// Builder uses "CaptionTemplate"; IR overlay uses "Caption". Try both.
		node.Caption = extractTextFromTemplate(doc, "CaptionTemplate")
		if node.Caption == "" {
			node.Caption = extractTextFromTemplate(doc, "Caption")
		}
		node.GroupBox = &types.GroupBoxProps{
			Collapsible: dGetString(doc, "Collapsible"),
			HeaderMode:  dGetString(doc, "HeaderMode"),
		}
		node.Children = extractChildWidgets(doc, "Widgets")
	case types.WidgetTabContainer:
		node.Children = extractChildWidgets(doc, "TabPages")
	case types.WidgetTabPage:
		node.Caption = extractTextFromTemplate(doc, "Caption")
		node.Children = extractChildWidgets(doc, "Widgets")
	case types.WidgetDataView:
		node.DataSource = extractBSONDataSource(doc)
		node.EntityCtx = extractDataViewEntityCtx(doc)
		node.Children = extractChildWidgets(doc, "Widgets")
	case types.WidgetListView:
		node.DataSource = extractBSONDataSource(doc)
		node.Children = extractChildWidgets(doc, "Templates")
	case types.WidgetButton:
		node.Caption = extractTextFromTemplate(doc, "Caption")
		node.ButtonStyle = dGetString(doc, "ButtonStyle")
		node.OnClick = extractButtonAction(doc)
	case types.WidgetLabel:
		node.Caption = extractTextFromTemplate(doc, "Caption")
	case types.WidgetText, types.WidgetTitle:
		node.Content = extractTextFromTemplate(doc, "Content")
	case types.WidgetTextBox, types.WidgetTextArea, types.WidgetDatePicker,
		types.WidgetRadioButtons, types.WidgetCheckBox:
		node.EntityAttr = extractAttributeRefStr(doc)
		node.Editable = extractEditableStr(doc)
		if ed := dGetDoc(doc, "ConditionalEditabilitySettings"); ed != nil {
			node.EditableIf = dGetString(ed, "Expression")
		}
	case types.WidgetSnippet:
		node.Snippet = &types.SnippetProps{SnippetName: extractSnippetName(doc)}
	case types.WidgetDataGrid:
		node.DataSource = extractPluggableDataSource(doc)
		node.DataGrid = extractDataGridProps(doc)
	case types.WidgetGallery:
		node.DataSource = extractPluggableDataSource(doc)
		node.Gallery = extractGalleryProps(doc)
	case types.WidgetImage:
		node.Image = extractImageProps(doc)
	case types.WidgetComboBox:
		node.DataSource = extractPluggableDataSource(doc)
		node.EntityAttr = extractCustomWidgetAttrRef(doc, "attribute")
	case types.WidgetUnknown:
		node.Unknown = &types.UnknownProps{
			WidgetID: extractCustomWidgetIDStr(doc),
		}
	}

	return node
}

// ---------------------------------------------------------------------------
// Helper extractors
// ---------------------------------------------------------------------------

func extractChildWidgets(doc bson.D, field string) []*types.WidgetNode {
	var nodes []*types.WidgetNode
	for _, w := range dGetArrayElements(dGet(doc, field)) {
		if wd, ok := w.(bson.D); ok {
			if node := widgetNodeFromBSON(wd); node != nil {
				nodes = append(nodes, node)
			}
		}
	}
	return nodes
}

func extractLayoutGridRows(doc bson.D) []*types.WidgetNode {
	var nodes []*types.WidgetNode
	for i, r := range dGetArrayElements(dGet(doc, "Rows")) {
		if rd, ok := r.(bson.D); ok {
			if node := widgetNodeFromBSON(rd); node != nil {
				// LayoutGridRow has no Name field in BSON — synthesise one
				// from position so the rendered MDL is parseable.
				if node.Name == "" {
					node.Name = fmt.Sprintf("row%d", i+1)
				}
				// Same for column children of this row.
				for j, c := range node.Children {
					if c != nil && c.Name == "" && c.Kind == types.WidgetLayoutCol {
						c.Name = fmt.Sprintf("col%d", j+1)
					}
				}
				nodes = append(nodes, node)
			}
		}
	}
	return nodes
}

func extractColWidth(doc bson.D) types.ColWidthDef {
	toInt := func(v any) int {
		switch n := v.(type) {
		case int32:
			return int(n)
		case int64:
			return int(n)
		case int:
			return n
		}
		return 0
	}
	return types.ColWidthDef{
		Desktop: toInt(dGet(doc, "DesktopWeight")),
		Tablet:  toInt(dGet(doc, "TabletWeight")),
		Phone:   toInt(dGet(doc, "PhoneWeight")),
	}
}

func extractTextFromTemplate(doc bson.D, field string) string {
	tmpl := dGetDoc(doc, field)
	if tmpl == nil {
		return ""
	}
	// Caption/Content may be wrapped in a ClientTemplate{Template, Fallback}
	// when written by the gen builder (genSimpleLabel) — unwrap to the
	// inner Text doc first if present.
	if inner := dGetDoc(tmpl, "Template"); inner != nil {
		tmpl = inner
	}
	// Text translations are stored under "Items" (gen Texts$Text PartList
	// property name) or "Translations" (the older alias used by the
	// hand-written IR encoder). Try both.
	translations := dGet(tmpl, "Items")
	if translations == nil {
		translations = dGet(tmpl, "Translations")
	}
	for _, t := range dGetArrayElements(translations) {
		if td, ok := t.(bson.D); ok {
			if dGetString(td, "LanguageCode") == "en_US" {
				return dGetString(td, "Text")
			}
		}
	}
	for _, t := range dGetArrayElements(translations) {
		if td, ok := t.(bson.D); ok {
			if txt := dGetString(td, "Text"); txt != "" {
				return txt
			}
		}
	}
	return ""
}

func extractAttributeRefStr(doc bson.D) string {
	aref := dGetDoc(doc, "AttributeRef")
	if aref == nil {
		return ""
	}
	return dGetString(aref, "AttributeQualifiedName")
}

func extractEditableStr(doc bson.D) string {
	return dGetString(doc, "Editable")
}

func extractDataViewEntityCtx(doc bson.D) string {
	ds := extractBSONDataSource(doc)
	if ds == nil {
		return ""
	}
	return ds.Entity
}

func extractBSONDataSource(doc bson.D) *types.DataSourceDef {
	dsd := dGetDoc(doc, "DataSource")
	if dsd == nil {
		return nil
	}
	typeName := dGetString(dsd, "$Type")
	ds := &types.DataSourceDef{}
	switch {
	case strings.Contains(typeName, "MicroflowSource"):
		ds.Kind = types.DataSourceMicroflow
		ds.Reference = dGetString(dsd, "MicroflowQualifiedName")
	case strings.Contains(typeName, "NanoflowSource"):
		ds.Kind = types.DataSourceNanoflow
		ds.Reference = dGetString(dsd, "NanoflowQualifiedName")
	case strings.Contains(typeName, "ContextSource") || strings.Contains(typeName, "ParameterSource"):
		ds.Kind = types.DataSourceParameter
		ds.Reference = dGetString(dsd, "ParameterName")
	case strings.Contains(typeName, "SelectionSource"):
		ds.Kind = types.DataSourceSelection
		ds.Reference = dGetString(dsd, "WidgetName")
	default:
		ds.Kind = types.DataSourceDatabase
		ds.Entity = dGetString(dsd, "EntityQualifiedName")
		ds.XPathConstraint = dGetString(dsd, "XPathConstraint")
	}
	return ds
}

func extractButtonAction(doc bson.D) string {
	act := dGetDoc(doc, "Action")
	if act == nil {
		return ""
	}
	typeName := dGetString(act, "$Type")
	// Mendix BSON storage names — "Forms$MicroflowAction" / "Forms$NanoflowAction"
	// / "Forms$OpenLinkAction" are the actual aliases (gen TypeName, not the
	// SDK ClientAction class name).
	switch {
	case strings.Contains(typeName, "MicroflowAction") || strings.Contains(typeName, "MicroflowClientAction"):
		// MicroflowSettings.MicroflowQualifiedName is the canonical path;
		// fallback for older formats checks the bare field directly.
		if settings := dGetDoc(act, "MicroflowSettings"); settings != nil {
			if name := dGetString(settings, "MicroflowQualifiedName"); name != "" {
				return name
			}
		}
		return dGetString(act, "MicroflowQualifiedName")
	case strings.Contains(typeName, "NanoflowAction") || strings.Contains(typeName, "NanoflowClientAction"):
		if settings := dGetDoc(act, "NanoflowSettings"); settings != nil {
			if name := dGetString(settings, "NanoflowQualifiedName"); name != "" {
				return name
			}
		}
		return dGetString(act, "NanoflowQualifiedName")
	case strings.Contains(typeName, "OpenLinkAction") || strings.Contains(typeName, "OpenLinkClientAction"):
		if settings := dGetDoc(act, "PageSettings"); settings != nil {
			if name := dGetString(settings, "PageQualifiedName"); name != "" {
				return name
			}
		}
		return dGetString(act, "PageQualifiedName")
	}
	return ""
}

func extractSnippetName(doc bson.D) string {
	ref := dGetDoc(doc, "Snippet")
	if ref == nil {
		return ""
	}
	return dGetString(ref, "SnippetQualifiedName")
}

func extractDesignPropsFromBSON(app bson.D) []types.DesignProp {
	var props []types.DesignProp
	for _, dp := range dGetArrayElements(dGet(app, "DesignProperties")) {
		dpd, ok := dp.(bson.D)
		if !ok {
			continue
		}
		p := types.DesignProp{
			Key:       dGetString(dpd, "Key"),
			ValueType: dGetString(dpd, "Type"),
		}
		switch p.ValueType {
		case "option":
			p.Option = dGetString(dpd, "Value")
		}
		if p.Key != "" {
			props = append(props, p)
		}
	}
	return props
}

// customWidgetKind returns the WidgetKind for a CustomWidgets$CustomWidget
// by inspecting the widget type string stored in its object properties.
func customWidgetKind(doc bson.D) types.WidgetKind {
	widgetType := extractCustomWidgetIDStr(doc)
	switch {
	case strings.Contains(widgetType, "datagrid2"):
		return types.WidgetDataGrid
	case strings.Contains(widgetType, "gallery"):
		return types.WidgetGallery
	case strings.Contains(widgetType, "combobox"):
		return types.WidgetComboBox
	case strings.Contains(widgetType, "image"):
		return types.WidgetImage
	}
	return types.WidgetUnknown
}

func extractCustomWidgetIDStr(doc bson.D) string {
	obj := dGetDoc(doc, "Object")
	if obj == nil {
		return ""
	}
	return dGetString(obj, "Type")
}

func extractCustomWidgetAttrRef(doc bson.D, propKey string) string {
	obj := dGetDoc(doc, "Object")
	if obj == nil {
		return ""
	}
	for _, prop := range dGetArrayElements(dGet(obj, "Properties")) {
		pd, ok := prop.(bson.D)
		if !ok {
			continue
		}
		if dGetString(pd, "key") == propKey {
			val := dGetDoc(pd, "Value")
			if val == nil {
				return ""
			}
			return dGetString(val, "AttributeQualifiedName")
		}
	}
	return ""
}

func extractPluggableDataSource(doc bson.D) *types.DataSourceDef {
	obj := dGetDoc(doc, "Object")
	if obj == nil {
		return nil
	}
	for _, prop := range dGetArrayElements(dGet(obj, "Properties")) {
		pd, ok := prop.(bson.D)
		if !ok {
			continue
		}
		if dGetString(pd, "key") == "datasource" {
			val := dGetDoc(pd, "Value")
			if val == nil {
				return nil
			}
			ds := &types.DataSourceDef{}
			typeName := dGetString(val, "$Type")
			switch {
			case strings.Contains(typeName, "NanoflowSource"):
				ds.Kind = types.DataSourceNanoflow
				ds.Reference = dGetString(val, "NanoflowQualifiedName")
			case strings.Contains(typeName, "MicroflowSource"):
				ds.Kind = types.DataSourceMicroflow
				ds.Reference = dGetString(val, "MicroflowQualifiedName")
			case strings.Contains(typeName, "ContextSource"):
				ds.Kind = types.DataSourceParameter
			default:
				ds.Kind = types.DataSourceDatabase
				ds.Entity = dGetString(val, "EntityQualifiedName")
				ds.XPathConstraint = dGetString(val, "XPathConstraint")
			}
			return ds
		}
	}
	return nil
}

func extractDataGridProps(doc bson.D) *types.DataGridProps {
	dgp := &types.DataGridProps{}
	obj := dGetDoc(doc, "Object")
	if obj == nil {
		return dgp
	}
	for _, prop := range dGetArrayElements(dGet(obj, "Properties")) {
		pd, ok := prop.(bson.D)
		if !ok {
			continue
		}
		key := dGetString(pd, "key")
		val := dGetDoc(pd, "Value")
		if val == nil {
			continue
		}
		switch key {
		case "pageSize":
			if s := dGetString(val, "Value"); s != "" {
				if n, err := strconv.Atoi(s); err == nil {
					dgp.PageSize = n
				}
			}
		case "pagination":
			dgp.Pagination = dGetString(val, "Value")
		case "pagingPosition":
			dgp.PagingPos = dGetString(val, "Value")
		case "columns":
			dgp.Columns = extractDataGridColumns(val)
		}
	}
	return dgp
}

func extractDataGridColumns(columnsVal bson.D) []types.ColumnDef {
	var cols []types.ColumnDef
	for _, item := range dGetArrayElements(dGet(columnsVal, "ObjectItems")) {
		itemDoc, ok := item.(bson.D)
		if !ok {
			continue
		}
		col := types.ColumnDef{}
		col.Name = dGetString(itemDoc, "Name")
		for _, prop := range dGetArrayElements(dGet(itemDoc, "Properties")) {
			pd, ok := prop.(bson.D)
			if !ok {
				continue
			}
			key := dGetString(pd, "key")
			val := dGetDoc(pd, "Value")
			if val == nil {
				continue
			}
			switch key {
			case "attribute":
				col.Attribute = dGetString(val, "AttributeQualifiedName")
			case "header":
				col.Caption = dGetString(val, "Value")
			case "showContentAs":
				col.ShowContentAs = dGetString(val, "Value")
			case "alignment":
				col.Alignment = dGetString(val, "Value")
			case "wrapText":
				col.WrapText = dGetString(val, "Value") == "true"
			case "sortable":
				col.Sortable = dGetString(val, "Value") == "true"
			case "resizable":
				col.Resizable = dGetString(val, "Value") == "true"
			case "draggable":
				col.Draggable = dGetString(val, "Value") == "true"
			case "hidable":
				col.Hidable = dGetString(val, "Value")
			case "width":
				col.ColumnWidth = dGetString(val, "Value")
			case "size":
				col.Size = dGetString(val, "Value")
			}
		}
		cols = append(cols, col)
	}
	return cols
}

func extractGalleryProps(doc bson.D) *types.GalleryProps {
	gp := &types.GalleryProps{}
	obj := dGetDoc(doc, "Object")
	if obj == nil {
		return gp
	}
	toInt := func(s string) int {
		n, _ := strconv.Atoi(s)
		return n
	}
	for _, prop := range dGetArrayElements(dGet(obj, "Properties")) {
		pd, ok := prop.(bson.D)
		if !ok {
			continue
		}
		key := dGetString(pd, "key")
		val := dGetDoc(pd, "Value")
		if val == nil {
			continue
		}
		switch key {
		case "desktopItems":
			gp.DesktopColumns = toInt(dGetString(val, "Value"))
		case "tabletItems":
			gp.TabletColumns = toInt(dGetString(val, "Value"))
		case "phoneItems":
			gp.PhoneColumns = toInt(dGetString(val, "Value"))
		case "itemSelection":
			gp.Selection = dGetString(val, "Value")
		}
	}
	return gp
}

func extractImageProps(doc bson.D) *types.ImageProps {
	ip := &types.ImageProps{}
	obj := dGetDoc(doc, "Object")
	if obj == nil {
		return ip
	}
	for _, prop := range dGetArrayElements(dGet(obj, "Properties")) {
		pd, ok := prop.(bson.D)
		if !ok {
			continue
		}
		key := dGetString(pd, "key")
		val := dGetDoc(pd, "Value")
		if val == nil {
			continue
		}
		switch key {
		case "imageUrl":
			ip.URL = dGetString(val, "Value")
		case "alternativeText":
			ip.AltText = dGetString(val, "Value")
		case "width":
			ip.Width = dGetString(val, "Value")
		case "height":
			ip.Height = dGetString(val, "Value")
		case "widthUnit":
			ip.WidthUnit = dGetString(val, "Value")
		case "heightUnit":
			ip.HeightUnit = dGetString(val, "Value")
		case "displayAs":
			ip.DisplayAs = dGetString(val, "Value")
		case "responsive":
			ip.Responsive = dGetString(val, "Value") == "true"
		case "type":
			ip.ImageType = dGetString(val, "Value")
		case "onClick":
			ip.OnClickType = dGetString(val, "Value")
		}
	}
	return ip
}

// ---------------------------------------------------------------------------
// WritePageModel / WriteSnippetModel
// ---------------------------------------------------------------------------

func (b *MprBackend) WritePageModel(id model.ID, pm *types.PageModel) error {
	return b.writeUnitWidgets(id, pm)
}

func (b *MprBackend) WriteSnippetModel(id model.ID, pm *types.PageModel) error {
	return b.writeUnitWidgets(id, pm)
}

// writeUnitWidgets loads the existing unit BSON, replaces the widget tree
// inside the LayoutCall/FormCall Arguments with the serialised PageModel
// widgets, and writes back.
func (b *MprBackend) writeUnitWidgets(id model.ID, pm *types.PageModel) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("WritePageModel: backend not open for writing")
	}

	raw, err := b.msdkReader.GetRawUnitBytes(string(id))
	if err != nil {
		return fmt.Errorf("load unit: %w", err)
	}
	var doc bson.D
	if err := bson.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	widgetsBSON := widgetsToBSON(pm.Widgets)

	callKey := "LayoutCall"
	if dGetDoc(doc, "LayoutCall") == nil {
		callKey = "FormCall"
	}
	callDoc := dGetDoc(doc, callKey)
	if callDoc == nil {
		return fmt.Errorf("writeUnitWidgets: no %s in unit %s", callKey, id)
	}

	args := dGetArrayElements(dGet(callDoc, "Arguments"))
	if len(args) == 0 {
		return fmt.Errorf("writeUnitWidgets: no Arguments in %s", callKey)
	}

	arg0, ok := args[0].(bson.D)
	if !ok {
		return fmt.Errorf("writeUnitWidgets: first argument is not a bson.D")
	}

	if wrapper := dGetDoc(arg0, "Widget"); wrapper != nil {
		dSet(wrapper, "Widgets", bsonVersionedArray(widgetsBSON))
	} else {
		dSet(arg0, "Widgets", bsonVersionedArray(widgetsBSON))
	}

	out, err := bson.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return b.msdkWriter.UpdateRawUnit(string(id), out)
}

// bsonVersionedArray wraps a []bson.D in Mendix's versioned-array format
// [int32(1), elem0, elem1, ...].
func bsonVersionedArray(docs []bson.D) bson.A {
	arr := bson.A{int32(1)}
	for _, d := range docs {
		arr = append(arr, d)
	}
	return arr
}

// ---------------------------------------------------------------------------
// PageModel → BSON (widgetsToBSON + widgetToBSON)
// ---------------------------------------------------------------------------

func widgetsToBSON(nodes []*types.WidgetNode) []bson.D {
	var docs []bson.D
	for _, n := range nodes {
		if d := widgetToBSON(n); d != nil {
			docs = append(docs, d)
		}
	}
	return docs
}

// widgetToBSON converts a single WidgetNode to a BSON document.
// This is the inverse of widgetNodeFromBSON.
func widgetToBSON(node *types.WidgetNode) bson.D {
	if node == nil {
		return nil
	}

	typeName := kindToBSONType(node.Kind)
	if typeName == "" {
		return nil
	}

	doc := bson.D{
		// Every widget needs a fresh $ID (16-byte GUID blob); Mendix's
		// ContentsUtil.GetGuidFromBson throws NullReferenceException if
		// $ID is missing.
		{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
		{Key: "$Type", Value: typeName},
		{Key: "Name", Value: node.Name},
	}

	app := bson.D{
		{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
		{Key: "$Type", Value: "Forms$Appearance"},
	}
	if node.Class != "" {
		app = append(app, bson.E{Key: "CSSClasses", Value: node.Class})
	}
	if node.Style != "" {
		app = append(app, bson.E{Key: "Style", Value: node.Style})
	}
	doc = append(doc, bson.E{Key: "Appearance", Value: app})

	switch node.Kind {
	case types.WidgetContainer, types.WidgetScrollView:
		doc = append(doc, bson.E{Key: "Widgets", Value: bsonVersionedArray(widgetsToBSON(node.Children))})

	case types.WidgetLayoutGrid:
		rows := widgetsToBSON(node.Children)
		doc = append(doc, bson.E{Key: "Rows", Value: bsonVersionedArray(rows)})

	case types.WidgetLayoutRow:
		cols := widgetsToBSON(node.Children)
		doc = append(doc, bson.E{Key: "Columns", Value: bsonVersionedArray(cols)})

	case types.WidgetLayoutCol:
		doc = append(doc,
			bson.E{Key: "DesktopWeight", Value: int32(node.ColWidth.Desktop)},
			bson.E{Key: "TabletWeight", Value: int32(node.ColWidth.Tablet)},
			bson.E{Key: "PhoneWeight", Value: int32(node.ColWidth.Phone)},
			bson.E{Key: "Widgets", Value: bsonVersionedArray(widgetsToBSON(node.Children))},
		)

	case types.WidgetGroupBox:
		doc = append(doc,
			bson.E{Key: "Caption", Value: simpleTextBSON(node.Caption)},
			bson.E{Key: "Widgets", Value: bsonVersionedArray(widgetsToBSON(node.Children))},
		)
		if node.GroupBox != nil {
			if node.GroupBox.Collapsible != "" {
				doc = append(doc, bson.E{Key: "Collapsible", Value: node.GroupBox.Collapsible})
			}
			if node.GroupBox.HeaderMode != "" {
				doc = append(doc, bson.E{Key: "HeaderMode", Value: node.GroupBox.HeaderMode})
			}
		}

	case types.WidgetTabContainer:
		doc = append(doc, bson.E{Key: "TabPages", Value: bsonVersionedArray(widgetsToBSON(node.Children))})

	case types.WidgetTabPage:
		doc = append(doc,
			bson.E{Key: "Caption", Value: simpleTextBSON(node.Caption)},
			bson.E{Key: "Widgets", Value: bsonVersionedArray(widgetsToBSON(node.Children))},
		)

	case types.WidgetDataView:
		if node.DataSource != nil {
			doc = append(doc, bson.E{Key: "DataSource", Value: dataSourceToBSON(node.DataSource)})
		}
		doc = append(doc, bson.E{Key: "Widgets", Value: bsonVersionedArray(widgetsToBSON(node.Children))})

	case types.WidgetButton:
		doc = append(doc,
			bson.E{Key: "Caption", Value: simpleTextBSON(node.Caption)},
			bson.E{Key: "ButtonStyle", Value: node.ButtonStyle},
		)

	case types.WidgetLabel:
		doc = append(doc, bson.E{Key: "Caption", Value: simpleTextBSON(node.Caption)})

	case types.WidgetText, types.WidgetTitle:
		doc = append(doc, bson.E{Key: "Content", Value: simpleTextBSON(node.Content)})

	case types.WidgetTextBox, types.WidgetTextArea, types.WidgetDatePicker,
		types.WidgetRadioButtons, types.WidgetCheckBox:
		if node.EntityAttr != "" {
			doc = append(doc, bson.E{Key: "AttributeRef", Value: bson.D{
				{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
				{Key: "$Type", Value: "Forms$AttributeRef"},
				{Key: "AttributeQualifiedName", Value: node.EntityAttr},
			}})
		}
		if node.Editable != "" {
			doc = append(doc, bson.E{Key: "Editable", Value: node.Editable})
		}

	case types.WidgetSnippet:
		if node.Snippet != nil {
			doc = append(doc, bson.E{Key: "Snippet", Value: bson.D{
				{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
				{Key: "$Type", Value: "Forms$SnippetRef"},
				{Key: "SnippetQualifiedName", Value: node.Snippet.SnippetName},
			}})
		}
	}

	return doc
}

// kindToBSONType maps WidgetKind → canonical BSON $Type using the Mendix
// STORAGE namespace ("Forms$"), not the SDK display namespace ("Pages$").
// Studio Pro's storage layer rejects pages whose widget $Type isn't in the
// type cache, and the type cache only registers "Forms$X" aliases — writing
// "Pages$LayoutGrid" triggers TypeCacheUnknownTypeException.
func kindToBSONType(kind types.WidgetKind) string {
	switch kind {
	case types.WidgetContainer:
		return "Forms$DivContainer"
	case types.WidgetScrollView:
		return "Forms$ScrollContainer"
	case types.WidgetGroupBox:
		return "Forms$GroupBox"
	case types.WidgetLayoutGrid:
		return "Forms$LayoutGrid"
	case types.WidgetLayoutRow:
		return "Forms$LayoutGridRow"
	case types.WidgetLayoutCol:
		return "Forms$LayoutGridColumn"
	case types.WidgetTabContainer:
		return "Forms$TabControl"
	case types.WidgetTabPage:
		return "Forms$TabPage"
	case types.WidgetDataView:
		return "Forms$DataView"
	case types.WidgetListView:
		return "Forms$ListView"
	case types.WidgetButton:
		return "Forms$ActionButton"
	case types.WidgetLabel:
		return "Forms$Label"
	case types.WidgetText:
		return "Forms$Text"
	case types.WidgetDynamicText:
		return "Forms$DynamicText"
	case types.WidgetTitle:
		return "Forms$Title"
	case types.WidgetTextBox:
		return "Forms$TextBox"
	case types.WidgetTextArea:
		return "Forms$TextArea"
	case types.WidgetDatePicker:
		return "Forms$DatePicker"
	case types.WidgetRadioButtons:
		return "Forms$RadioButtons"
	case types.WidgetCheckBox:
		return "Forms$CheckBox"
	case types.WidgetNavList:
		return "Forms$NavigationList"
	case types.WidgetSnippet:
		return "Forms$SnippetCallWidget"
	case types.WidgetDataGrid, types.WidgetGallery, types.WidgetComboBox,
		types.WidgetImage, types.WidgetUnknown:
		return "CustomWidgets$CustomWidget"
	}
	return ""
}

// simpleTextBSON creates a minimal Text BSON doc with a single en_US translation.
func simpleTextBSON(text string) bson.D {
	tr := bson.D{
		{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
		{Key: "$Type", Value: "Texts$Translation"},
		{Key: "LanguageCode", Value: "en_US"},
		{Key: "Text", Value: text},
	}
	return bson.D{
		{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
		{Key: "$Type", Value: "Texts$Text"},
		{Key: "Translations", Value: bson.A{int32(1), tr}},
	}
}

// dataSourceToBSON converts a DataSourceDef to its BSON representation.
// Note: WidgetDataView / WidgetListView are gated as lossy in
// pageModelHasLossyWidget so the gen builder's rich Forms$DataViewSource
// (which includes SourceVariable+PageVariable for parameter sources) wins.
// dataSourceToBSON here only sees pluggable widget sources today.
func dataSourceToBSON(ds *types.DataSourceDef) bson.D {
	switch ds.Kind {
	case types.DataSourceDatabase:
		doc := bson.D{
			{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
			{Key: "$Type", Value: "Forms$GridXPathSource"},
			{Key: "EntityQualifiedName", Value: ds.Entity},
		}
		if ds.XPathConstraint != "" {
			doc = append(doc, bson.E{Key: "XPathConstraint", Value: ds.XPathConstraint})
		}
		return doc
	case types.DataSourceMicroflow:
		return bson.D{
			{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
			{Key: "$Type", Value: "Forms$MicroflowSource"},
			{Key: "MicroflowQualifiedName", Value: ds.Reference},
		}
	case types.DataSourceNanoflow:
		return bson.D{
			{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
			{Key: "$Type", Value: "Forms$NanoflowSource"},
			{Key: "NanoflowQualifiedName", Value: ds.Reference},
		}
	case types.DataSourceParameter:
		return bson.D{
			{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
			{Key: "$Type", Value: "Forms$AssociationSource"},
			{Key: "ParameterName", Value: ds.Reference},
		}
	}
	return nil
}
