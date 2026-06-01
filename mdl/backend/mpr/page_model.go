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

// WritePageModel will be implemented in Task 7 (PageModel → BSON). For now
// the stub satisfies the PageModelBackend interface so the read path can
// land independently.
func (b *MprBackend) WritePageModel(id model.ID, m *types.PageModel) error {
	return fmt.Errorf("MprBackend.WritePageModel: not yet implemented (Task 7)")
}

// WriteSnippetModel mirrors WritePageModel; real implementation arrives in Task 7.
func (b *MprBackend) WriteSnippetModel(id model.ID, m *types.PageModel) error {
	return fmt.Errorf("MprBackend.WriteSnippetModel: not yet implemented (Task 7)")
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
		node.Caption = extractTextFromTemplate(doc, "Caption")
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
	for _, r := range dGetArrayElements(dGet(doc, "Rows")) {
		if rd, ok := r.(bson.D); ok {
			if node := widgetNodeFromBSON(rd); node != nil {
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
	for _, t := range dGetArrayElements(dGet(tmpl, "Translations")) {
		if td, ok := t.(bson.D); ok {
			if dGetString(td, "LanguageCode") == "en_US" {
				return dGetString(td, "Text")
			}
		}
	}
	for _, t := range dGetArrayElements(dGet(tmpl, "Translations")) {
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
	switch {
	case strings.Contains(typeName, "MicroflowClientAction"):
		return dGetString(act, "MicroflowQualifiedName")
	case strings.Contains(typeName, "NanoflowClientAction"):
		return dGetString(act, "NanoflowQualifiedName")
	case strings.Contains(typeName, "OpenLinkClientAction"):
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
