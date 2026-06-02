// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/bsonutil"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genCW "github.com/mendixlabs/mxcli/modelsdk/gen/customwidgets"
	genDt "github.com/mendixlabs/mxcli/modelsdk/gen/datatypes"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	genTexts "github.com/mendixlabs/mxcli/modelsdk/gen/texts"
)

// ============================================================================
// Gen-native helper functions (V3 builder support)
// ============================================================================

// buildNanoflowSourceGen constructs a NanoflowSource gen element from a nanoflow
// DataSourceV3, including parameter mappings for any explicit datasource arguments.
// DataGrid2 callers convert to bson.D via genElementToBSONDoc; Forms callers
// return the element directly.
func (pb *pageBuilder) buildNanoflowSourceGen(ds *ast.DataSourceV3) (*genPg.NanoflowSource, string, error) {
	nfID, err := pb.resolveNanoflowByName(ds.Reference)
	if err != nil {
		return nil, "", mdlerrors.NewBackend("resolve nanoflow", err)
	}
	_ = nfID
	entityName := pb.getNanoflowReturnEntityName(ds.Reference)
	ns := genPg.NewNanoflowSource()
	assignFreshID(ns)
	ns.SetNanoflowQualifiedName(ds.Reference)
	for _, arg := range ds.Args {
		pm := genPg.NewNanoflowParameterMapping()
		assignFreshID(pm)
		pm.SetParameterQualifiedName(ds.Reference + "." + arg.Name)
		if expr, ok := arg.Value.(string); ok {
			pm.SetExpression(expr)
		}
		ns.AddParameterMappings(pm)
	}
	return ns, entityName, nil
}

// genElementToBSONDoc encodes a gen element to bson.D via the codec.
// Used in BSON-builder functions that need raw bson.D rather than a gen element.
func genElementToBSONDoc(elem element.Element) (bson.D, error) {
	enc := codec.Encoder{}
	raw, err := enc.Encode(elem)
	if err != nil {
		return nil, err
	}
	var doc bson.D
	return doc, bson.Unmarshal(raw, &doc)
}

// genSimpleText creates a gen Text element with a single en_US translation.
func genSimpleText(text string) *genTexts.Text {
	t := genTexts.NewText()
	assignFreshID(t)
	tr := genTexts.NewTranslation()
	assignFreshID(tr)
	tr.SetLanguageCode("en_US")
	tr.SetText(text)
	t.AddTranslations(tr)
	return t
}

// genSimpleLabel creates a gen ClientTemplate (used as a label/caption) with a
// single en_US translation. This matches the Mendix pattern for caption/title
// properties that take a ClientTemplate element.
func genSimpleLabel(text string) element.Element {
	ct := genPg.NewClientTemplate()
	assignFreshID(ct)
	tmpl := genSimpleText(text)
	ct.SetTemplate(tmpl)
	fallback := genTexts.NewText()
	assignFreshID(fallback)
	ct.SetFallback(fallback)
	return ct
}

// genPageParamType creates the appropriate gen DataType element for a page
// parameter based on the MDL AST data type.
func genPageParamType(dt ast.DataType, entityQN string) element.Element {
	switch dt.Kind {
	case ast.TypeString:
		return genPg.NewTextBox() // placeholder — we use direct type string in buildPageV3
	}
	return nil
}

// ============================================================================
// V3 Page Builder
// ============================================================================

// buildPageV3 creates a *genPg.Page from a CreatePageStmtV3. The builder's
// lastContainerID field is set to the resolved container (folder or module).
func (pb *pageBuilder) buildPageV3(s *ast.CreatePageStmtV3) (*genPg.Page, error) {
	// Resolve folder if specified
	containerID := pb.moduleID
	if s.Folder != "" {
		folderID, err := pb.resolveFolder(s.Folder)
		if err != nil {
			return nil, mdlerrors.NewBackend("resolve folder "+s.Folder, err)
		}
		containerID = folderID
	}
	pb.lastContainerID = containerID

	page := genPg.NewPage()
	assignFreshID(page)
	page.SetName(s.Name.Name)
	page.SetDocumentation(s.Documentation)
	page.SetExcluded(s.Excluded)
	page.SetMarkAsUsed(false)
	if s.URL != "" {
		page.SetUrl(s.URL)
	}

	// Set title: Mendix stores page Title as Texts$Text (not Forms$ClientTemplate).
	// Using genSimpleLabel (which wraps in ClientTemplate) would cause a
	// StorageLoadException in Studio Pro 11 ("ClientTemplate cannot be converted to Text").
	if s.Title != "" {
		page.SetTitle(genSimpleText(s.Title))
	}

	// Build parameters FIRST so paramScope/paramEntityNames are populated
	// before widget building (widgets may reference parameters as datasources).
	for _, param := range s.Parameters {
		pp := genPg.NewPageParameter()
		assignFreshID(pp)
		pp.SetName(param.Name)
		pp.SetIsRequired(true)

		if param.EntityType.Name != "" {
			// Entity-typed parameter: build a proper DataTypes$ObjectType nested element.
			// Previously used setRawBSONField("ParameterType_type/entity") which wrote flat
			// keys instead of a nested ParameterType document, causing DataTypes$UnknownType
			// on read-back (CE0170 / CE0566).
			entityID, err := pb.resolveEntity(param.EntityType)
			if err != nil {
				return nil, mdlerrors.NewBackend("resolve entity "+param.EntityType.String(), err)
			}
			entityName := param.EntityType.String()
			pb.paramScope[param.Name] = entityID
			pb.paramEntityNames[param.Name] = entityName
			objType := genDt.NewObjectType()
			assignFreshID(objType)
			objType.SetEntityQualifiedName(entityName)
			pp.SetParameterType(objType)
		} else if bsonType := pageParamBSONType(param.Type); bsonType != "" {
			// Primitive type: build the corresponding gen DataType element.
			primType := newPrimPageParamType(bsonType)
			if primType != nil {
				if withID, ok := primType.(genElementWithID); ok {
					assignFreshID(withID)
				}
				pp.SetParameterType(primType)
			}
		}

		page.AddParameters(pp)
	}

	// Build variables
	for _, v := range s.Variables {
		lv := genPg.NewLocalVariable()
		assignFreshID(lv)
		lv.SetName(v.Name)
		lv.SetVariableType(mdlTypeToDataTypeElement(v.DataType))
		if v.DefaultValue != "" {
			setRawBSONField(lv, "DefaultValue", v.DefaultValue)
		}
		page.AddVariables(lv)
	}

	// Resolve layout and build LayoutCall (after parameters so widgets can use paramScope)
	if s.Layout != "" {
		layoutID, err := pb.resolveLayout(s.Layout)
		if err != nil {
			log.Printf("warning: layout %s not found", s.Layout)
		} else {
			_ = layoutID

			lc := genPg.NewLayoutCall()
			assignFreshID(lc)
			// Mendix stores the layout name under the "Form" BSON key (historic naming).
			// After the types.go fix, initLayoutCall() uses "Form" as the property name,
			// so SetLayoutQualifiedName now writes "Form" directly.
			lc.SetLayoutQualifiedName(s.Layout)

			mainPlaceholderRef := pb.getMainPlaceholderRef(s.Layout)

			arg := genPg.NewLayoutCallArgument()
			assignFreshID(arg)
			// The gen type is "Forms$LayoutCallArgument" but Mendix uses "Forms$FormCallArgument"
			// as the storage type. Override so Studio Pro can parse the argument correctly.
			arg.SetTypeName("Forms$FormCallArgument")
			arg.SetParameterQualifiedName(mainPlaceholderRef)

			if len(s.Widgets) > 0 {
				// SP11.6.6 reads FormCallArgument.Widgets (plural flat list).
				// The old pattern wrapped content in a DivContainer and stored it via
				// SetWidget (singular) — SP11.6.6 cannot find this content.
				// Use AddWidgets to write a flat Widgets list instead.
				expanded, err := pb.expandFragments(s.Widgets)
				if err != nil {
					return nil, err
				}
				for _, astWidget := range expanded {
					w, err := pb.buildWidgetV3(astWidget)
					if err != nil {
						return nil, mdlerrors.NewBackend("build widget", err)
					}
					arg.AddWidgets(w)
				}
			}

			lc.AddArguments(arg)
			page.SetLayoutCall(lc)
		}
	}

	return page, nil
}

// newPrimPageParamType returns a fresh gen DataType element for a primitive page
// parameter type (e.g. "DataTypes$StringType"). Returns nil for unknown type strings.
// Callers must call assignFreshID on the returned element before use.
func newPrimPageParamType(bsonTypeName string) element.Element {
	switch bsonTypeName {
	case "DataTypes$StringType":
		return genDt.NewStringType()
	case "DataTypes$IntegerType":
		return genDt.NewIntegerType()
	case "DataTypes$DecimalType":
		return genDt.NewDecimalType()
	case "DataTypes$BooleanType":
		return genDt.NewBooleanType()
	case "DataTypes$DateTimeType":
		return genDt.NewDateTimeType()
	default:
		// DataTypes$LongType and others: fall back to nil (caller skips SetParameterType).
		return nil
	}
}

// buildSnippetV3 creates a *genPg.Snippet from a CreateSnippetStmtV3.
// The builder's lastContainerID field is set to the resolved container.
func (pb *pageBuilder) buildSnippetV3(s *ast.CreateSnippetStmtV3) (*genPg.Snippet, error) {
	// Resolve folder if specified
	containerID := pb.moduleID
	if s.Folder != "" {
		folderID, err := pb.resolveFolder(s.Folder)
		if err != nil {
			return nil, mdlerrors.NewBackend("resolve folder "+s.Folder, err)
		}
		containerID = folderID
	}
	pb.lastContainerID = containerID

	snippet := genPg.NewSnippet()
	assignFreshID(snippet)
	snippet.SetName(s.Name.Name)
	snippet.SetDocumentation(s.Documentation)

	// Build parameters
	for _, param := range s.Parameters {
		sp := genPg.NewSnippetParameter()
		assignFreshID(sp)
		sp.SetName(param.Name)

		if param.EntityType.Name != "" {
			entityID, err := pb.resolveEntity(param.EntityType)
			if err != nil {
				return nil, mdlerrors.NewBackend("resolve entity "+param.EntityType.String(), err)
			}
			entityName := param.EntityType.String()
			pb.paramScope[param.Name] = entityID
			pb.paramEntityNames[param.Name] = entityName
			// Build a proper DataTypes$ObjectType nested element (same fix as page params).
			objType := genDt.NewObjectType()
			assignFreshID(objType)
			objType.SetEntityQualifiedName(entityName)
			sp.SetParameterType(objType)
		}

		snippet.AddParameters(sp)
	}

	// Build variables
	for _, v := range s.Variables {
		lv := genPg.NewLocalVariable()
		assignFreshID(lv)
		lv.SetName(v.Name)
		lv.SetVariableType(mdlTypeToDataTypeElement(v.DataType))
		if v.DefaultValue != "" {
			setRawBSONField(lv, "DefaultValue", v.DefaultValue)
		}
		snippet.AddVariables(lv)
	}

	// Build widgets (expanding fragments)
	pb.isSnippet = true
	defer func() { pb.isSnippet = false }()

	expanded, err := pb.expandFragments(s.Widgets)
	if err != nil {
		return nil, err
	}
	for _, astWidget := range expanded {
		w, err := pb.buildWidgetV3(astWidget)
		if err != nil {
			return nil, mdlerrors.NewBackend("build widget", err)
		}
		snippet.AddWidgets(w)
	}

	return snippet, nil
}

// buildWidgetV3 converts a V3 AST widget to an element.Element.
//
// Keyword dispatch (Phase 2 — issue #539): the keywordDispatchTable encodes
// our editorial policy for dual-stack keywords (e.g. DATAGRID → pluggable
// Datagrid 2.x). Today the existing switch cases handle this correctly via
// the hand-coded builders (buildDataGridV3 already produces pluggable BSON).
// The dispatch table is consumed by inspection commands and DESCRIBE-side
// keyword resolution rather than overriding write-side routing here.
func (pb *pageBuilder) buildWidgetV3(w *ast.WidgetV3) (element.Element, error) {
	var widget element.Element
	var err error

	switch strings.ToLower(w.Type) {
	case "dataview":
		widget, err = pb.buildDataViewV3(w)
	case "datagrid":
		widget, err = pb.buildDataGridV3(w)
	case "legacydatagrid":
		return nil, mdlerrors.NewUnsupported(
			"LEGACYDATAGRID (native Forms$DataGrid) is not yet implemented. " +
				"Use DATAGRID for the pluggable equivalent on Mendix 11+, " +
				"or open the project in Studio Pro to add native datagrids manually.")
	case "listview":
		widget, err = pb.buildListViewV3(w)
	case "layoutgrid":
		widget, err = pb.buildLayoutGridV3(w)
	case "row":
		widget, err = pb.buildContainerWithRowV3(w)
	case "column":
		widget, err = pb.buildContainerWithColumnV3(w)
	case "container", "customcontainer":
		widget, err = pb.buildContainerV3(w)
	case "textbox":
		widget, err = pb.buildTextBoxV3(w)
	case "textarea":
		widget, err = pb.buildTextAreaV3(w)
	case "datepicker":
		widget, err = pb.buildDatePickerV3(w)
	case "dropdown":
		widget, err = pb.buildDropdownV3(w)
	case "checkbox":
		widget, err = pb.buildCheckBoxV3(w)
	case "text", "statictext":
		widget, err = pb.buildTextWidgetV3(w)
	case "dynamictext":
		widget, err = pb.buildDynamicTextV3(w)
	case "title":
		widget, err = pb.buildTitleV3(w)
	case "button", "actionbutton":
		widget, err = pb.buildButtonV3(w)
	case "tabcontainer":
		widget, err = pb.buildTabContainerV3(w)
	case "tabpage":
		return nil, mdlerrors.NewValidation("tabpage must be a direct child of tabcontainer")
	case "groupbox":
		widget, err = pb.buildGroupBoxV3(w)
	case "radiobuttons":
		widget, err = pb.buildRadioButtonsV3(w)
	case "navigationlist":
		widget, err = pb.buildNavigationListV3(w)
	case "item":
		return nil, mdlerrors.NewValidation("item must be a direct child of navigationlist")
	case "snippetcall":
		widget, err = pb.buildSnippetCallV3(w)
	case "footer":
		widget, err = pb.buildFooterV3(w)
	case "header":
		widget, err = pb.buildHeaderV3(w)
	case "controlbar":
		widget, err = pb.buildControlBarV3(w)
	case "template":
		widget, err = pb.buildTemplateV3(w)
	case "filter":
		widget, err = pb.buildFilterV3(w)
	case "staticimage":
		widget, err = pb.buildStaticImageV3(w)
	case "dynamicimage":
		widget, err = pb.buildDynamicImageV3(w)
	case "image":
		// IMAGE routes to the pluggable React widget (com.mendix.widget.web.image.Image)
		pb.initPluggableEngine()
		if pb.widgetRegistry != nil {
			if def, ok := pb.widgetRegistry.Get("image"); ok {
				cw, buildErr := pb.pluggableEngine.Build(def, w)
				if buildErr != nil {
					return nil, buildErr
				}
				return pb.customWidgetToElement(cw)
			}
		}
		// Fallback to static image if pluggable engine unavailable
		widget, err = pb.buildStaticImageV3(w)
	default:
		pb.initPluggableEngine()
		if pb.widgetRegistry != nil {
			if def, ok := pb.widgetRegistry.Get(strings.ToUpper(w.Type)); ok {
				cw, buildErr := pb.pluggableEngine.Build(def, w)
				if buildErr != nil {
					return nil, buildErr
				}
				return pb.customWidgetToElement(cw)
			}
			if w.Type == "pluggablewidget" || w.Type == "customwidget" {
				if widgetType, ok := w.Properties["WidgetType"].(string); ok {
					if def, ok := pb.widgetRegistry.GetByWidgetID(widgetType); ok {
						cw, buildErr := pb.pluggableEngine.Build(def, w)
						if buildErr != nil {
							return nil, buildErr
						}
						return pb.customWidgetToElement(cw)
					}
					return nil, mdlerrors.NewNotFoundMsg("widget", widgetType, "no definition for widget "+widgetType+" (run 'mxcli widget init -p app.mpr')")
				}
			}
		}
		if pb.pluggableEngineErr != nil {
			return nil, mdlerrors.NewUnsupported(fmt.Sprintf("unsupported widget type: %s (%v)", w.Type, pb.pluggableEngineErr))
		}
		return nil, mdlerrors.NewUnsupported("unsupported widget type: " + w.Type)
	}

	if err != nil {
		return nil, err
	}

	// Apply Class/Style and design properties via gen Appearance
	applyWidgetAppearanceGen(widget, w, pb.themeRegistry)

	// Apply conditional visibility/editability
	applyConditionalSettingsGen(widget, w)

	return widget, nil
}

// customWidgetToElement returns the element.Element from a *backend.GenCustomWidgetElem.
// Build() now returns *GenCustomWidgetElem which implements element.Element directly.
func (pb *pageBuilder) customWidgetToElement(cw *backend.GenCustomWidgetElem) (element.Element, error) {
	if cw == nil {
		return nil, fmt.Errorf("customWidgetToElement: nil GenCustomWidgetElem")
	}
	return cw.AsElement(), nil
}

// applyConditionalSettingsGen sets ConditionalVisibilitySettings and
// ConditionalEditabilitySettings on a gen widget element.
func applyConditionalSettingsGen(widget element.Element, w *ast.WidgetV3) {
	if visibleIf := w.GetStringProp("VisibleIf"); visibleIf != "" {
		cvs := genPg.NewConditionalVisibilitySettings()
		assignFreshID(cvs)
		cvs.SetExpression(visibleIf)
		type cvsSetter interface {
			SetConditionalVisibilitySettings(element.Element)
		}
		if s, ok := widget.(cvsSetter); ok {
			s.SetConditionalVisibilitySettings(cvs)
		}
	}

	if editableIf := w.GetStringProp("EditableIf"); editableIf != "" {
		ces := genPg.NewConditionalEditabilitySettings()
		assignFreshID(ces)
		ces.SetExpression(editableIf)
		type cesSetter interface {
			SetConditionalEditabilitySettings(element.Element)
		}
		if s, ok := widget.(cesSetter); ok {
			s.SetConditionalEditabilitySettings(ces)
		}
	}
}

// applyWidgetAppearanceGen sets Class, Style, and DesignProperties on a gen widget.
// Uses the gen Appearance sub-element for design properties.
func applyWidgetAppearanceGen(widget element.Element, w *ast.WidgetV3, theme *ThemeRegistry) {
	class, style := w.GetClass(), w.GetStyle()
	astProps := w.GetDesignProperties()

	// If no appearance data, skip
	if class == "" && style == "" && len(astProps) == 0 {
		return
	}

	// Apply class/style directly on the widget (most gen widgets have these setters)
	if class != "" {
		type classSetter interface{ SetClass(string) }
		if s, ok := widget.(classSetter); ok {
			s.SetClass(class)
		}
	}
	if style != "" {
		type styleSetter interface{ SetStyle(string) }
		if s, ok := widget.(styleSetter); ok {
			s.SetStyle(style)
		}
	}

	// Apply design properties via Appearance element
	if len(astProps) > 0 {
		appearance := newDefaultAppearance()
		if class != "" {
			appearance.SetClass(class)
		}
		if style != "" {
			appearance.SetStyle(style)
		}

		for _, p := range astProps {
			switch strings.ToLower(p.Value) {
			case "on":
				appearance.AddDesignProperties(newDesignPropertyEntry(p.Key, genPg.NewToggleDesignPropertyValue()))
			case "off":
				// OFF means toggle absence — skip
			default:
				opt := genPg.NewOptionDesignPropertyValue()
				opt.SetOption(p.Value)
				appearance.AddDesignProperties(newDesignPropertyEntry(p.Key, opt))
			}
		}

		type appearanceSetter interface {
			SetAppearance(element.Element)
		}
		if s, ok := widget.(appearanceSetter); ok {
			s.SetAppearance(appearance)
		}
	}
}

// resolveDesignPropertyValueType determines the correct ValueType for a design property
// based on the theme definition. ToggleButtonGroup and ColorPicker use "custom" type;
// Dropdown uses "option" type. Falls back to "option" if theme info is unavailable.
func resolveDesignPropertyValueType(key string, themeProps []ThemeProperty) string {
	for _, tp := range themeProps {
		if tp.Name == key {
			switch tp.Type {
			case "ToggleButtonGroup", "ColorPicker":
				return "custom"
			default:
				return "option"
			}
		}
	}
	return "option"
}

// =============================================================================
// V3 DataSource Builders
// =============================================================================

// buildDataSourceV3 converts a V3 DataSource AST to an element.Element.
// Returns the datasource element, the entity name for context, and any error.
// For DataView context (database type), produces Forms$DataViewSource.
// For ListView context (database type), use buildListViewDataSourceV3 instead.
func (pb *pageBuilder) buildDataSourceV3(ds *ast.DataSourceV3) (element.Element, string, error) {
	switch ds.Type {
	case "parameter":
		paramName := strings.TrimPrefix(ds.Reference, "$")
		entityID, ok := pb.paramScope[paramName]
		entityName := pb.paramEntityNames[paramName]
		if !ok {
			entityID, ok = pb.paramScope["$"+paramName]
			entityName = pb.paramEntityNames["$"+paramName]
		}
		if !ok {
			return nil, "", mdlerrors.NewNotFound("parameter", ds.Reference)
		}

		if entityName == "" {
			var err error
			entityName, err = pb.getEntityNameByID(entityID)
			if err != nil {
				log.Printf("warning: could not resolve entity name for ID %s: %v", entityID, err)
			}
		}

		dvs := genPg.NewDataViewSource()
		assignFreshID(dvs)
		dvs.SetForceFullObjects(false)
		if pb.isSnippet {
			dvs.SetSnippetParameterQualifiedName(paramName)
		} else {
			// SP11.6.6: use SourceVariable (nested PageVariable) instead of flat PageParameter
			sv := genPg.NewPageVariable()
			assignFreshID(sv)
			sv.SetPageParameterQualifiedName(paramName)
			dvs.SetSourceVariable(sv)
		}
		if entityName != "" {
			// Set entity ref for type awareness
			ref := genDm.NewDirectEntityRef()
			assignFreshID(ref)
			ref.SetEntityQualifiedName(entityName)
			dvs.SetEntityRef(ref)
		}
		return dvs, entityName, nil

	case "database":
		entityID, err := pb.resolveEntity(ast.QualifiedName{
			Module: pb.extractModule(ds.Reference),
			Name:   pb.extractName(ds.Reference),
		})
		if err != nil {
			return nil, "", mdlerrors.NewBackend("resolve entity", err)
		}
		_ = entityID

		// DataView database source → Forms$DataViewSource with EntityRef
		dvs := genPg.NewDataViewSource()
		assignFreshID(dvs)
		dvs.SetEntityPath(ds.Reference)
		ref := genDm.NewDirectEntityRef()
		assignFreshID(ref)
		ref.SetEntityQualifiedName(ds.Reference)
		dvs.SetEntityRef(ref)
		return dvs, ds.Reference, nil

	case "microflow":
		mfID, err := pb.resolveMicroflow(ds.Reference)
		if err != nil {
			return nil, "", mdlerrors.NewBackend("resolve microflow", err)
		}
		_ = mfID

		entityName := pb.getMicroflowReturnEntityName(ds.Reference)

		ms := genPg.NewMicroflowSource()
		assignFreshID(ms)
		settings := genPg.NewMicroflowSettings()
		assignFreshID(settings)
		settings.SetMicroflowQualifiedName(ds.Reference)
		ms.SetMicroflowSettings(settings)
		return ms, entityName, nil

	case "nanoflow":
		return pb.buildNanoflowSourceGen(ds)

	case "association":
		ctxVar := ds.ContextVariable
		if ctxVar == "currentObject" {
			ctxVar = ""
		}

		path := ds.Reference
		destEntity := ""
		if idx := strings.Index(path, "/"); idx >= 0 {
			destEntity = path[idx+1:]
			path = path[:idx]
		} else {
			destEntity = pb.resolveAssociationDestination(path, pb.entityContext)
		}

		as := genPg.NewAssociationSource()
		assignFreshID(as)
		as.SetEntityPath(path + "/" + destEntity)
		if ctxVar != "" {
			// SourceVariable is a PageVariable element in gen
			pv := genPg.NewPageVariable()
			assignFreshID(pv)
			pv.SetPageParameterQualifiedName(ctxVar)
			as.SetSourceVariable(pv)
		}
		return as, destEntity, nil

	case "selection":
		widgetName := ds.Reference
		widgetID, ok := pb.widgetScope[widgetName]
		if !ok {
			return nil, "", mdlerrors.NewNotFound("widget", widgetName)
		}
		_ = widgetID

		entityName := pb.paramEntityNames[widgetName]

		lts := genPg.NewListenTargetSource()
		assignFreshID(lts)
		lts.SetListenTarget(widgetName)
		return lts, entityName, nil

	default:
		return nil, "", mdlerrors.NewUnsupported("unsupported datasource type: " + ds.Type)
	}
}

// buildListViewDataSourceV3 builds a datasource suitable for ListView context.
// For database type, produces Forms$ListViewXPathSource instead of DataViewSource.
func (pb *pageBuilder) buildListViewDataSourceV3(ds *ast.DataSourceV3) (element.Element, string, error) {
	if ds.Type != "database" {
		return pb.buildDataSourceV3(ds)
	}

	entityID, err := pb.resolveEntity(ast.QualifiedName{
		Module: pb.extractModule(ds.Reference),
		Name:   pb.extractName(ds.Reference),
	})
	if err != nil {
		return nil, "", mdlerrors.NewBackend("resolve entity", err)
	}
	_ = entityID

	lvs := genPg.NewListViewXPathSource()
	assignFreshID(lvs)
	lvs.SetEntityPath(ds.Reference)
	ref := genDm.NewDirectEntityRef()
	assignFreshID(ref)
	ref.SetEntityQualifiedName(ds.Reference)
	lvs.SetEntityRef(ref)
	if ds.Where != "" {
		lvs.SetXPathConstraint(ds.Where)
	}
	return lvs, ds.Reference, nil
}

// resolveAssociationDestination looks up an association by qualified name and returns
// the qualified name of the entity OPPOSITE to contextEntity. Returns "" if the
// association can't be resolved or the context isn't on either end.
//
// Convention (per CLAUDE.md): ParentID = FROM entity, ChildID = TO entity.
// For `Module.OrderLine_Order` (`FROM OrderLine TO Order`), context=Order → dest=OrderLine (parent side).
func (pb *pageBuilder) resolveAssociationDestination(assocQN, contextEntity string) string {
	if assocQN == "" {
		return ""
	}
	parts := strings.SplitN(assocQN, ".", 2)
	if len(parts) != 2 {
		return ""
	}
	modName, assocName := parts[0], parts[1]

	pairs, err := pb.getDomainModelsWithContainer()
	if err != nil {
		return ""
	}
	for _, pair := range pairs {
		if pair.DM == nil {
			continue
		}
		if pb.moduleNameByID(pair.ContainerID) != modName {
			continue
		}
		for _, a := range pair.DM.AssociationsItems() {
			assoc, ok := a.(*genDm.Association)
			if !ok || assoc.Name() != assocName {
				continue
			}
			parentEntity := pb.entityQNByID(model.ID(assoc.ParentRefID()))
			childEntity := pb.entityQNByID(model.ID(assoc.ChildRefID()))
			if contextEntity != "" {
				if contextEntity == childEntity {
					return parentEntity
				}
				if contextEntity == parentEntity {
					return childEntity
				}
			}
			return childEntity
		}
	}
	return ""
}

// entityQNByID returns the qualified name (Module.Entity) for a given entity ID
// by scanning all domain models. Returns "" if not found.
func (pb *pageBuilder) entityQNByID(entityID model.ID) string {
	if entityID == "" {
		return ""
	}
	pairs, err := pb.getDomainModelsWithContainer()
	if err != nil {
		return ""
	}
	for _, pair := range pairs {
		if pair.DM == nil {
			continue
		}
		for _, elem := range pair.DM.EntitiesItems() {
			e, ok := elem.(*genDm.Entity)
			if !ok {
				continue
			}
			if model.ID(e.ID()) == entityID {
				modName := pb.moduleNameByID(pair.ContainerID)
				if modName == "" {
					return e.Name()
				}
				return modName + "." + e.Name()
			}
		}
	}
	return ""
}

// moduleNameByID returns the module name for a given module ID. Cached via hierarchy.
func (pb *pageBuilder) moduleNameByID(moduleID model.ID) string {
	if moduleID == "" {
		return ""
	}
	modules := pb.getModules()
	for _, m := range modules {
		if m.ID == moduleID {
			return m.Name
		}
	}
	return ""
}

// getMicroflowReturnEntityName looks up a microflow and returns its return type entity name.
func (pb *pageBuilder) getMicroflowReturnEntityName(qualifiedName string) string {
	if pb.execCache != nil && pb.execCache.createdMicroflows != nil {
		if info, ok := pb.execCache.createdMicroflows[qualifiedName]; ok {
			return info.ReturnEntityName
		}
	}

	parts := strings.Split(qualifiedName, ".")
	if len(parts) < 2 {
		return ""
	}
	moduleName := parts[0]
	mfName := strings.Join(parts[1:], ".")

	mfs, err := pb.getMicroflows()
	if err != nil {
		return ""
	}

	h, err := pb.getHierarchy()
	if err != nil {
		return ""
	}

	for _, mf := range mfs {
		if mf == nil || pb.microflowsRepo == nil {
			continue
		}
		containerID, _ := pb.microflowsRepo.GetContainerUUID(model.ID(mf.ID()))
		modName := h.GetModuleName(h.FindModuleID(containerID))
		if modName == moduleName && mf.Name() == mfName {
			return extractEntityFromGenReturnType(mf.ReturnType())
		}
	}

	return ""
}

// extractEntityFromGenReturnType peels the "List of " prefix off a
// gen-rendered return-type string and returns the bare entity QN.
func extractEntityFromGenReturnType(rt string) string {
	rt = strings.TrimSpace(rt)
	if rt == "" || rt == "Void" {
		return ""
	}
	if after, ok := strings.CutPrefix(rt, "List of "); ok {
		return after
	}
	if isPrimitiveReturnType(rt) {
		return ""
	}
	if strings.Contains(rt, ".") {
		return rt
	}
	return ""
}

func isPrimitiveReturnType(rt string) bool {
	switch rt {
	case "Boolean", "Integer", "Long", "Decimal", "String", "DateTime", "Date", "Binary":
		return true
	}
	return false
}

// getNanoflowReturnEntityName looks up a nanoflow and returns its return type entity name.
func (pb *pageBuilder) getNanoflowReturnEntityName(qualifiedName string) string {
	// Check session-local cache first (mirrors getMicroflowReturnEntityName).
	if pb.execCache != nil && pb.execCache.createdNanoflows != nil {
		if info, ok := pb.execCache.createdNanoflows[qualifiedName]; ok {
			return info.ReturnEntityName
		}
	}

	parts := strings.Split(qualifiedName, ".")
	var moduleName, name string
	if len(parts) >= 2 {
		moduleName = parts[0]
		name = parts[1]
	} else {
		moduleName = pb.moduleName
		name = qualifiedName
	}

	if pb.nanoflowsRepo == nil {
		return ""
	}
	nanoflows, err := pb.nanoflowsRepo.List("")
	if err != nil {
		return ""
	}

	h, err := pb.getHierarchy()
	if err != nil {
		return ""
	}

	for _, nf := range nanoflows {
		if nf == nil {
			continue
		}
		containerID, _ := pb.microflowsRepo.GetContainerUUID(model.ID(nf.ID()))
		modName := h.GetModuleName(h.FindModuleID(containerID))
		if modName == moduleName && nf.Name() == name {
			return extractEntityFromGenReturnType(nf.ReturnType())
		}
	}

	return ""
}

// =============================================================================
// V3 Client Action Builder
// =============================================================================

// buildClientActionV3 converts a V3 Action AST to a gen element.Element.
func (pb *pageBuilder) buildClientActionV3(action *ast.ActionV3) (element.Element, error) {
	switch action.Type {
	case "save":
		act := genPg.NewSaveChangesClientAction()
		assignFreshID(act)
		act.SetClosePage(action.ClosePage)
		return act, nil

	case "cancel":
		act := genPg.NewCancelChangesClientAction()
		assignFreshID(act)
		act.SetClosePage(action.ClosePage)
		return act, nil

	case "close":
		act := genPg.NewClosePageClientAction()
		assignFreshID(act)
		return act, nil

	case "delete":
		act := genPg.NewDeleteClientAction()
		assignFreshID(act)
		return act, nil

	case "create":
		entityID, err := pb.resolveEntity(ast.QualifiedName{
			Module: pb.extractModule(action.Target),
			Name:   pb.extractName(action.Target),
		})
		if err != nil {
			return nil, mdlerrors.NewBackend("resolve entity for create", err)
		}
		_ = entityID

		act := genPg.NewCreateObjectClientAction()
		assignFreshID(act)
		ref := genDm.NewDirectEntityRef()
		assignFreshID(ref)
		ref.SetEntityQualifiedName(action.Target)
		act.SetEntityRef(ref)

		// Handle THEN action (show page)
		if action.ThenAction != nil && action.ThenAction.Type == "showPage" {
			// Resolution is for validation only — warn on forward reference.
			if _, err := pb.resolvePageRef(action.ThenAction.Target); err != nil {
				log.Printf("warning: then show_page %s not found (will still create action by name)", action.ThenAction.Target)
			}
			ps := genPg.NewPageSettings()
			assignFreshID(ps)
			ps.SetPageQualifiedName(action.ThenAction.Target)
			act.SetPageSettings(ps)
		}
		return act, nil

	case "showPage":
		// Resolution is for validation only — warn on missing page (forward reference)
		// but still emit the action so the page name is stored correctly.
		if _, err := pb.resolvePageRef(action.Target); err != nil {
			log.Printf("warning: action show_page %s not found (will still create action by name)", action.Target)
		}

		act := genPg.NewPageClientAction()
		assignFreshID(act)
		ps := genPg.NewPageSettings()
		assignFreshID(ps)
		ps.SetPageQualifiedName(action.Target)
		// ParameterMappings intentionally left empty: Mendix propagates the page
		// parameter from the calling context automatically. Storing PageParameterMapping
		// (Forms$FormCallArgument) objects inside FormSettings (PageSettings) causes
		// LayoutCallArgument constructor failures in Studio Pro when the container type
		// is PageSettings rather than LayoutCall.
		act.SetPageSettings(ps)
		return act, nil

	case "microflow":
		// Resolution is for validation only — the BSON action stores the qualified
		// name string, not a UUID. Log a warning for missing microflows but continue
		// so that pages can reference microflows defined later in the same script.
		if _, err := pb.resolveMicroflow(action.Target); err != nil {
			log.Printf("warning: action microflow %s not found (will still create action by name)", action.Target)
		}

		act := genPg.NewMicroflowClientAction()
		assignFreshID(act)
		settings := genPg.NewMicroflowSettings()
		assignFreshID(settings)
		settings.SetMicroflowQualifiedName(action.Target)

		for _, arg := range action.Args {
			mm := genPg.NewMicroflowParameterMapping()
			assignFreshID(mm)
			// SP11.6.6: fully-qualified "Module.MicroflowName.ParamName"
			mm.SetParameterQualifiedName(action.Target + "." + arg.Name)

			if strVal, ok := arg.Value.(string); ok {
				// SP11.6.6: use Expression for all values (not Variable sub-object)
				mm.SetExpression(strVal)
			}
			settings.AddParameterMappings(mm)
		}

		act.SetMicroflowSettings(settings)
		if action.ClosePage {
			setRawBSONField(act, "ClosePage", true)
		}
		return act, nil

	case "nanoflow":
		nfID, err := pb.resolveNanoflowByName(action.Target)
		if err != nil {
			return nil, mdlerrors.NewBackend("resolve nanoflow", err)
		}
		_ = nfID

		act := genPg.NewCallNanoflowClientAction()
		assignFreshID(act)
		act.SetNanoflowQualifiedName(action.Target)

		for _, arg := range action.Args {
			nm := genPg.NewNanoflowParameterMapping()
			assignFreshID(nm)
			nm.SetParameterQualifiedName(arg.Name)

			if strVal, ok := arg.Value.(string); ok {
				if strings.HasPrefix(strVal, "$") {
					pv := genPg.NewPageVariable()
					assignFreshID(pv)
					pv.SetPageParameterQualifiedName(strVal)
					nm.SetVariable(pv)
				} else {
					nm.SetExpression(strVal)
				}
			}
			act.AddParameterMappings(nm)
		}
		return act, nil

	case "openLink":
		act := genPg.NewOpenLinkClientAction()
		assignFreshID(act)
		act.SetLinkType("Web")
		addr := genPg.NewStaticOrDynamicString()
		assignFreshID(addr)
		addr.SetValue(action.LinkURL)
		act.SetAddress(addr)
		return act, nil

	case "signOut":
		act := genPg.NewSignOutClientAction()
		assignFreshID(act)
		return act, nil

	case "completeTask":
		act := genPg.NewSetTaskOutcomeClientAction()
		assignFreshID(act)
		act.SetClosePage(true)
		act.SetCommit(true)
		act.SetOutcomeValue(action.OutcomeValue)
		return act, nil

	default:
		return nil, mdlerrors.NewUnsupported("unsupported action type: " + action.Type)
	}
}

// =============================================================================
// Helper functions
// =============================================================================

func (pb *pageBuilder) extractModule(qualifiedName string) string {
	qualifiedName = unquoteQualifiedName(qualifiedName)
	parts := strings.Split(qualifiedName, ".")
	if len(parts) >= 2 {
		return parts[0]
	}
	return pb.moduleName
}

func (pb *pageBuilder) extractName(qualifiedName string) string {
	qualifiedName = unquoteQualifiedName(qualifiedName)
	parts := strings.Split(qualifiedName, ".")
	if len(parts) >= 2 {
		return parts[1]
	}
	return qualifiedName
}

func (pb *pageBuilder) getEntityNameByID(entityID model.ID) (string, error) {
	pairs, err := pb.getDomainModelsWithContainer()
	if err != nil {
		return "", err
	}

	modules := pb.getModules()
	moduleNames := make(map[model.ID]string)
	for _, m := range modules {
		moduleNames[m.ID] = m.Name
	}

	for _, pair := range pairs {
		for _, elem := range pair.DM.EntitiesItems() {
			e, ok := elem.(*genDm.Entity)
			if !ok {
				continue
			}
			if model.ID(e.ID()) == entityID {
				moduleName := moduleNames[pair.ContainerID]
				return moduleName + "." + e.Name(), nil
			}
		}
	}
	return "", mdlerrors.NewNotFound("entity", string(entityID))
}

// pageParamBSONType maps a DataType to the BSON $Type string for primitive page parameters.
// Returns empty string for entity/enum types (which use DataTypes$ObjectType instead).
func pageParamBSONType(dt ast.DataType) string {
	switch dt.Kind {
	case ast.TypeString:
		return "DataTypes$StringType"
	case ast.TypeInteger:
		return "DataTypes$IntegerType"
	case ast.TypeLong:
		return "DataTypes$LongType"
	case ast.TypeDecimal:
		return "DataTypes$DecimalType"
	case ast.TypeBoolean:
		return "DataTypes$BooleanType"
	case ast.TypeDateTime:
		return "DataTypes$DateTimeType"
	default:
		return ""
	}
}

// resolveNanoflowByName resolves a nanoflow qualified name to its ID.
func (pb *pageBuilder) resolveNanoflowByName(nfName string) (model.ID, error) {
	if pb.execCache != nil && pb.execCache.createdNanoflows != nil {
		if info, ok := pb.execCache.createdNanoflows[nfName]; ok {
			return info.ID, nil
		}
	}

	parts := strings.Split(nfName, ".")
	var moduleName, name string
	if len(parts) >= 2 {
		moduleName = parts[0]
		name = parts[1]
	} else {
		moduleName = pb.moduleName
		name = nfName
	}

	if pb.nanoflowsRepo == nil {
		return "", mdlerrors.NewNotFound("nanoflow", nfName)
	}
	nanoflows, err := pb.nanoflowsRepo.List("")
	if err != nil {
		return "", mdlerrors.NewBackend("list nanoflows", err)
	}

	h, err := pb.getHierarchy()
	if err != nil {
		return "", err
	}

	for _, nf := range nanoflows {
		if nf == nil {
			continue
		}
		containerID, _ := pb.microflowsRepo.GetContainerUUID(model.ID(nf.ID()))
		modName := h.GetModuleName(h.FindModuleID(containerID))
		if modName == moduleName && nf.Name() == name {
			return model.ID(nf.ID()), nil
		}
	}

	return "", mdlerrors.NewNotFound("nanoflow", nfName)
}

// mdlTypeToBsonType converts an MDL type name to a BSON DataTypes$* type string.
func mdlTypeToBsonType(mdlType string) string {
	switch strings.ToLower(mdlType) {
	case "boolean":
		return "DataTypes$BooleanType"
	case "string":
		return "DataTypes$StringType"
	case "integer":
		return "DataTypes$IntegerType"
	case "long":
		return "DataTypes$LongType"
	case "decimal":
		return "DataTypes$DecimalType"
	case "datetime", "date":
		return "DataTypes$DateTimeType"
	default:
		return "DataTypes$ObjectType"
	}
}

// mdlTypeToDataTypeElement creates a gen DataTypes element for use as LocalVariable.VariableType.
// Studio Pro requires VariableType to be a nested object ($Type + $ID), not a plain string.
func mdlTypeToDataTypeElement(mdlType string) element.Element {
	bsonType := mdlTypeToBsonType(mdlType)
	dt := newPrimPageParamType(bsonType)
	if dt == nil {
		// Fallback for types not in newPrimPageParamType (e.g. Long): use ObjectType.
		dt = genDt.NewObjectType()
	}
	if withID, ok := dt.(genElementWithID); ok {
		assignFreshID(withID)
	}
	return dt
}

// bsonTypeToMDLType converts a BSON DataTypes$* type to an MDL type name.
func bsonTypeToMDLType(bsonType string) string {
	switch bsonType {
	case "DataTypes$BooleanType":
		return "Boolean"
	case "DataTypes$StringType":
		return "String"
	case "DataTypes$IntegerType":
		return "Integer"
	case "DataTypes$LongType":
		return "Long"
	case "DataTypes$DecimalType":
		return "Decimal"
	case "DataTypes$DateTimeType":
		return "DateTime"
	case "DataTypes$ObjectType":
		return "Object"
	default:
		return "Unknown"
	}
}

func (pb *pageBuilder) resolveAttributePathForEntity(attrName string, entityName string) string {
	oldContext := pb.entityContext
	pb.entityContext = entityName
	defer func() { pb.entityContext = oldContext }()
	return pb.resolveAttributePath(attrName)
}

// resolveTemplateAttributePath resolves template parameter values like $widgetName.Attribute
// to fully qualified entity paths like Module.Entity.Attribute.
func (pb *pageBuilder) resolveTemplateAttributePath(attrRef string) string {
	if attrRef == "" {
		return ""
	}

	if after, ok := strings.CutPrefix(attrRef, "$"); ok {
		withoutDollar := after
		parts := strings.SplitN(withoutDollar, ".", 2)
		if len(parts) == 2 {
			widgetName := parts[0]
			attrName := parts[1]

			if entityName, ok := pb.paramEntityNames[widgetName]; ok {
				return entityName + "." + attrName
			}
			if entityName, ok := pb.paramEntityNames["$"+widgetName]; ok {
				return entityName + "." + attrName
			}
			if pb.entityContext != "" {
				return pb.entityContext + "." + attrName
			}
			return attrRef
		}
	}

	return pb.resolveAttributePath(attrRef)
}

// resolveTemplateAttributePathFull resolves a template parameter reference and populates
// the gen ClientTemplateParameter with AttributePath, Expression, or SourceVariable.
func (pb *pageBuilder) resolveTemplateAttributePathFull(attrRef string, param *genPg.ClientTemplateParameter) {
	if attrRef == "" {
		return
	}

	if after, ok := strings.CutPrefix(attrRef, "$"); ok {
		withoutDollar := after
		parts := strings.SplitN(withoutDollar, ".", 2)
		if len(parts) == 2 {
			paramName := parts[0]
			attrName := parts[1]

			if entityName, ok := pb.paramEntityNames[paramName]; ok {
				fullPath := entityName + "." + attrName
				if pb.isNonStringAttribute(fullPath) {
					param.SetExpression("toString($" + paramName + "/" + attrName + ")")
					return
				}
				sv := genPg.NewPageVariable()
				assignFreshID(sv)
				sv.SetPageParameterQualifiedName(paramName)
				param.SetSourceVariable(sv)
				param.SetAttributePath(fullPath)
				return
			}
			if entityName, ok := pb.paramEntityNames["$"+paramName]; ok {
				fullPath := entityName + "." + attrName
				if pb.isNonStringAttribute(fullPath) {
					param.SetExpression("toString($" + paramName + "/" + attrName + ")")
					return
				}
				sv := genPg.NewPageVariable()
				assignFreshID(sv)
				sv.SetPageParameterQualifiedName(paramName)
				param.SetSourceVariable(sv)
				param.SetAttributePath(fullPath)
				return
			}
		}
	}

	resolved := pb.resolveTemplateAttributePath(attrRef)
	if !strings.HasPrefix(attrRef, "$") {
		// If the attrRef is already a full expression (contains function call or embedded $
		// variable), use it directly rather than prepending $currentObject/.
		if strings.Contains(attrRef, "(") || strings.Contains(attrRef, "$") {
			param.SetExpression(attrRef)
			return
		}
		if pb.isNonStringAttribute(resolved) {
			// Non-string attributes (enum, datetime, etc.) need explicit toString().
			param.SetExpression("toString($currentObject/" + attrRef + ")")
		} else {
			// String attributes: use expression format so the describe code and
			// Mendix validator can read the binding. AttributePath alone is not
			// checked by the describe and causes CE0402 "No value specified".
			param.SetExpression("$currentObject/" + attrRef)
		}
		return
	}
	// $var/attr is already in Mendix expression form; use SetExpression so the
	// describe code can read it back (describe reads Expression, not AttributePath).
	if strings.Contains(attrRef, "/") {
		param.SetExpression(attrRef)
		return
	}
	param.SetAttributePath(resolved)
}

// isNonStringAttribute checks if an attribute path refers to a non-String type.
func (pb *pageBuilder) isNonStringAttribute(attrPath string) bool {
	attrType := pb.findAttributeType(attrPath)
	if attrType == nil {
		return false
	}
	_, isString := attrType.(*genDm.StringAttributeType)
	return !isString
}

// ============================================================================
// Fragment Expansion
// ============================================================================

// expandFragments processes a widget list, expanding any USE_FRAGMENT sentinels
// into their referenced fragment widgets. Non-fragment widgets pass through unchanged.
func (pb *pageBuilder) expandFragments(widgets []*ast.WidgetV3) ([]*ast.WidgetV3, error) {
	var result []*ast.WidgetV3
	for _, w := range widgets {
		expanded, err := pb.expandIfFragment(w)
		if err != nil {
			return nil, err
		}
		result = append(result, expanded...)
	}
	return result, nil
}

// expandIfFragment returns the widget as-is if it's not a USE_FRAGMENT sentinel,
// or expands it into cloned fragment widgets with optional prefix.
func (pb *pageBuilder) expandIfFragment(w *ast.WidgetV3) ([]*ast.WidgetV3, error) {
	if w.Type != "USE_FRAGMENT" {
		return []*ast.WidgetV3{w}, nil
	}

	if pb.fragments == nil {
		return nil, mdlerrors.NewNotFound("fragment", w.Name)
	}
	frag, ok := pb.fragments[w.Name]
	if !ok {
		return nil, mdlerrors.NewNotFound("fragment", w.Name)
	}

	widgets := cloneWidgets(frag.Widgets)
	if prefix, ok := w.Properties["Prefix"].(string); ok && prefix != "" {
		prefixWidgetNames(widgets, prefix)
	}
	return widgets, nil
}

// cloneWidgets deep-copies a widget tree to avoid mutating the fragment definition.
func cloneWidgets(widgets []*ast.WidgetV3) []*ast.WidgetV3 {
	if widgets == nil {
		return nil
	}
	result := make([]*ast.WidgetV3, len(widgets))
	for i, w := range widgets {
		result[i] = cloneWidget(w)
	}
	return result
}

func cloneWidget(w *ast.WidgetV3) *ast.WidgetV3 {
	clone := &ast.WidgetV3{
		Type:       w.Type,
		Name:       w.Name,
		Properties: make(map[string]interface{}, len(w.Properties)),
		Children:   cloneWidgets(w.Children),
	}
	for k, v := range w.Properties {
		clone.Properties[k] = v
	}
	return clone
}

// prefixWidgetNames recursively prepends a prefix to all widget names.
func prefixWidgetNames(widgets []*ast.WidgetV3, prefix string) {
	for _, w := range widgets {
		if w.Name != "" {
			w.Name = prefix + w.Name
		}
		prefixWidgetNames(w.Children, prefix)
	}
}

// ============================================================================
// Individual Widget Builders (V3 Gen-native)
// ============================================================================

func (pb *pageBuilder) buildLayoutGridV3(w *ast.WidgetV3) (element.Element, error) {
	lg := genPg.NewLayoutGrid()
	assignFreshID(lg)
	lg.SetName(w.Name)

	for _, child := range w.Children {
		if strings.ToLower(child.Type) == "row" {
			row, err := pb.buildLayoutGridRowV3(child)
			if err != nil {
				return nil, err
			}
			lg.AddRows(row)
		}
	}

	return lg, nil
}

func (pb *pageBuilder) buildLayoutGridRowV3(w *ast.WidgetV3) (element.Element, error) {
	row := genPg.NewLayoutGridRow()
	assignFreshID(row)

	for _, child := range w.Children {
		if strings.ToLower(child.Type) == "column" {
			col, err := pb.buildLayoutGridColumnV3(child)
			if err != nil {
				return nil, err
			}
			row.AddColumns(col)
		}
	}

	return row, nil
}

func (pb *pageBuilder) buildLayoutGridColumnV3(w *ast.WidgetV3) (element.Element, error) {
	col := genPg.NewLayoutGridColumn()
	assignFreshID(col)

	// Studio Pro defaults all size fields to -1 (AutoFill) when unset.
	// Apply the same defaults so our BSON matches SP output.
	col.SetWeight(-1)
	col.SetTabletWeight(-1)
	col.SetPhoneWeight(-1)
	col.SetPreviewWidth(-1)

	// Handle DesktopWidth
	if dw := w.GetDesktopWidth(); dw != nil {
		switch v := dw.(type) {
		case int:
			col.SetWeight(int32(v))
		case string:
			if strings.EqualFold(v, "AutoFill") {
				col.SetWeight(-1)
			}
		}
	}

	// Handle TabletWidth
	if tw := w.Properties["TabletWidth"]; tw != nil {
		switch v := tw.(type) {
		case int:
			col.SetTabletWeight(int32(v))
		case string:
			if strings.EqualFold(v, "AutoFill") {
				col.SetTabletWeight(-1)
			}
		}
	}

	// Handle PhoneWidth
	if pw := w.Properties["PhoneWidth"]; pw != nil {
		switch v := pw.(type) {
		case int:
			col.SetPhoneWeight(int32(v))
		case string:
			if strings.EqualFold(v, "AutoFill") {
				col.SetPhoneWeight(-1)
			}
		}
	}

	// Build child widgets
	for _, child := range w.Children {
		widget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		col.AddWidgets(widget)
	}

	return col, nil
}

// buildContainerWithRowV3 creates a DivContainer holding a LayoutGrid with one row.
func (pb *pageBuilder) buildContainerWithRowV3(w *ast.WidgetV3) (element.Element, error) {
	container := genPg.NewDivContainer()
	assignFreshID(container)
	container.SetName(w.Name)

	lg := genPg.NewLayoutGrid()
	assignFreshID(lg)
	lg.SetName(w.Name + "_grid")

	row, err := pb.buildLayoutGridRowV3(w)
	if err != nil {
		return nil, err
	}
	lg.AddRows(row)
	container.AddWidgets(lg)

	return container, nil
}

// buildContainerWithColumnV3 creates a DivContainer holding a LayoutGrid with one column.
func (pb *pageBuilder) buildContainerWithColumnV3(w *ast.WidgetV3) (element.Element, error) {
	container := genPg.NewDivContainer()
	assignFreshID(container)
	container.SetName(w.Name)

	lg := genPg.NewLayoutGrid()
	assignFreshID(lg)
	lg.SetName(w.Name + "_grid")

	row := genPg.NewLayoutGridRow()
	assignFreshID(row)

	col, err := pb.buildLayoutGridColumnV3(w)
	if err != nil {
		return nil, err
	}
	row.AddColumns(col)
	lg.AddRows(row)
	container.AddWidgets(lg)

	return container, nil
}

func (pb *pageBuilder) buildContainerV3(w *ast.WidgetV3) (element.Element, error) {
	container := genPg.NewDivContainer()
	assignFreshID(container)
	container.SetName(w.Name)

	// Handle RenderMode
	if rm := w.GetRenderMode(); rm != "" {
		container.SetRenderMode(rm)
	}

	// Build child widgets
	for _, child := range w.Children {
		widget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		container.AddWidgets(widget)
	}

	return container, nil
}

func (pb *pageBuilder) buildTabContainerV3(w *ast.WidgetV3) (element.Element, error) {
	tc := genPg.NewTabContainer()
	assignFreshID(tc)
	tc.SetName(w.Name)

	for _, child := range w.Children {
		if strings.ToLower(child.Type) == "tabpage" {
			tp, err := pb.buildTabPageV3(child)
			if err != nil {
				return nil, err
			}
			tc.AddTabPages(tp)
		}
	}

	if err := pb.registerWidgetName(w.Name, model.ID(tc.ID())); err != nil {
		return nil, err
	}

	return tc, nil
}

func (pb *pageBuilder) buildTabPageV3(w *ast.WidgetV3) (element.Element, error) {
	tp := genPg.NewTabPage()
	assignFreshID(tp)
	tp.SetName(w.Name)

	if caption := w.GetCaption(); caption != "" {
		tp.SetCaption(genSimpleLabel(caption))
	}

	for _, child := range w.Children {
		widget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		tp.AddWidgets(widget)
	}

	if err := pb.registerWidgetName(w.Name, model.ID(tp.ID())); err != nil {
		return nil, err
	}

	return tp, nil
}

func (pb *pageBuilder) buildGroupBoxV3(w *ast.WidgetV3) (element.Element, error) {
	gb := genPg.NewGroupBox()
	assignFreshID(gb)
	gb.SetName(w.Name)
	gb.SetCollapsible("No")
	gb.SetHeaderMode("Div")

	if caption := w.GetCaption(); caption != "" {
		gb.SetCaption(genSimpleLabel(caption))
	}

	if collapsible := w.GetStringProp("Collapsible"); collapsible != "" {
		switch strings.ToLower(collapsible) {
		case "yesexpanded", "yesinitiallyexpanded", "yes":
			gb.SetCollapsible("YesInitiallyExpanded")
		case "yescollapsed", "yesinitiallycollapsed":
			gb.SetCollapsible("YesInitiallyCollapsed")
		case "no":
			gb.SetCollapsible("No")
		default:
			gb.SetCollapsible(collapsible)
		}
	}

	if headerMode := w.GetStringProp("HeaderMode"); headerMode != "" {
		gb.SetHeaderMode(headerMode)
	}

	for _, child := range w.Children {
		widget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		gb.AddWidgets(widget)
	}

	if err := pb.registerWidgetName(w.Name, model.ID(gb.ID())); err != nil {
		return nil, err
	}

	return gb, nil
}

// buildFooterV3 creates a DivContainer (footer) from V3 syntax.
func (pb *pageBuilder) buildFooterV3(w *ast.WidgetV3) (element.Element, error) {
	footer := genPg.NewDivContainer()
	assignFreshID(footer)
	footer.SetName(w.Name)

	for _, child := range w.Children {
		childWidget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		footer.AddWidgets(childWidget)
	}

	if err := pb.registerWidgetName(w.Name, model.ID(footer.ID())); err != nil {
		return nil, err
	}

	return footer, nil
}

// buildHeaderV3 creates a DivContainer (header) from V3 syntax.
func (pb *pageBuilder) buildHeaderV3(w *ast.WidgetV3) (element.Element, error) {
	header := genPg.NewDivContainer()
	assignFreshID(header)
	header.SetName(w.Name)

	for _, child := range w.Children {
		childWidget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		header.AddWidgets(childWidget)
	}

	if err := pb.registerWidgetName(w.Name, model.ID(header.ID())); err != nil {
		return nil, err
	}

	return header, nil
}

// buildControlBarV3 creates a DivContainer (control bar) from V3 syntax.
func (pb *pageBuilder) buildControlBarV3(w *ast.WidgetV3) (element.Element, error) {
	controlBar := genPg.NewDivContainer()
	assignFreshID(controlBar)
	controlBar.SetName(w.Name)

	for _, child := range w.Children {
		childWidget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		controlBar.AddWidgets(childWidget)
	}

	if err := pb.registerWidgetName(w.Name, model.ID(controlBar.ID())); err != nil {
		return nil, err
	}

	return controlBar, nil
}

func (pb *pageBuilder) buildDataViewV3(w *ast.WidgetV3) (element.Element, error) {
	dv := genPg.NewDataView()
	assignFreshID(dv)
	dv.SetName(w.Name)

	if ds := w.GetDataSource(); ds != nil {
		dataSource, entityName, err := pb.buildDataSourceV3(ds)
		if err != nil {
			return nil, mdlerrors.NewBackend("build datasource", err)
		}
		dv.SetDataSource(dataSource)

		oldContext := pb.entityContext
		pb.entityContext = entityName
		defer func() { pb.entityContext = oldContext }()

		if w.Name != "" && entityName != "" {
			pb.paramEntityNames[w.Name] = entityName
		}
	}

	// Build child widgets, separating FOOTER widgets
	for _, child := range w.Children {
		if child.Type == "footer" {
			dv.SetShowFooter(true)
			for _, fw := range child.Children {
				widget, err := pb.buildWidgetV3(fw)
				if err != nil {
					return nil, err
				}
				dv.AddFooterWidgets(widget)
			}
			continue
		}
		childWidget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		dv.AddWidgets(childWidget)
	}

	// Also build footer widgets from Properties (legacy support)
	if footerWidgets, ok := w.Properties["Footer"].([]*ast.WidgetV3); ok {
		dv.SetShowFooter(true)
		for _, fw := range footerWidgets {
			widget, err := pb.buildWidgetV3(fw)
			if err != nil {
				return nil, err
			}
			dv.AddFooterWidgets(widget)
		}
	}

	if err := pb.registerWidgetName(w.Name, model.ID(dv.ID())); err != nil {
		return nil, err
	}

	return dv, nil
}

func (pb *pageBuilder) buildDataGridV3(w *ast.WidgetV3) (element.Element, error) {
	widgetID := model.ID(types.GenerateID())

	// Build datasource BSON (pre-serialized for DataGridSpec)
	var datasourceBSON bson.D
	if ds := w.GetDataSource(); ds != nil {
		dsDoc, entityName, err := pb.buildDataGridDataSourceBSON(ds)
		if err != nil {
			return nil, mdlerrors.NewBackend("build datasource", err)
		}
		datasourceBSON = dsDoc

		oldContext := pb.entityContext
		pb.entityContext = entityName
		defer func() { pb.entityContext = oldContext }()
	}

	// Extract column definitions and CONTROLBAR widgets from children
	var columns []backend.DataGridColumnSpec
	var headerWidgetsBSON []bson.D
	for _, child := range w.Children {
		switch strings.ToLower(child.Type) {
		case "column":
			attr := child.GetAttribute()
			if attr == "" && child.Name != "" && len(child.Children) == 0 {
				attr = child.Name
			}
			col := backend.DataGridColumnSpec{
				Attribute:  pb.resolveAttributePath(attr),
				Caption:    child.GetCaption(),
				Properties: child.Properties,
			}
			for _, grandchild := range child.Children {
				if filterWidgetID := dataGridFilterWidgetID(grandchild.Type); filterWidgetID != "" {
					// Attributes left empty: column-level filters are auto-bound by DataGrid2 to the column attribute.
					fw, err := pb.widgetBackend.BuildFilterWidgetGen(backend.FilterWidgetSpec{
						WidgetID:   filterWidgetID,
						FilterName: grandchild.Name,
						FilterType: grandchild.GetFilterType(),
					}, pb.backend.Path())
					if err != nil {
						return nil, mdlerrors.NewBackend("build column filter widget", err)
					}
					// Serialize GenCustomWidgetElem to bson.D
					fwDoc, err := pb.serializeGenWidgetToBsonD(fw)
					if err != nil {
						return nil, mdlerrors.NewBackend("serialize filter widget", err)
					}
					col.FilterWidgetBSON = fwDoc
				} else {
					// Build child widget as pre-serialized BSON
					childDoc, err := pb.buildWidgetBSON(grandchild)
					if err != nil {
						return nil, mdlerrors.NewBackend("build column child widget", err)
					}
					if childDoc != nil {
						col.ChildWidgetsBSON = append(col.ChildWidgetsBSON, childDoc)
					}
				}
			}
			columns = append(columns, col)
		case "controlbar":
			for _, controlBarChild := range child.Children {
				childDoc, err := pb.buildWidgetBSON(controlBarChild)
				if err != nil {
					return nil, mdlerrors.NewBackend("build controlbar widget", err)
				}
				if childDoc != nil {
					headerWidgetsBSON = append(headerWidgetsBSON, childDoc)
				}
			}
		}
	}

	// Collect paging overrides from AST properties
	pagingOverrides := make(map[string]string)
	for mdlKey, widgetKey := range dataGridPagingPropMap {
		if v := w.GetStringProp(mdlKey); v != "" {
			pagingOverrides[widgetKey] = v
		} else if iv := w.GetIntProp(mdlKey); iv > 0 {
			pagingOverrides[widgetKey] = fmt.Sprintf("%d", iv)
		} else if bv, ok := w.Properties[mdlKey]; ok {
			if boolVal, isBool := bv.(bool); isBool {
				if boolVal {
					pagingOverrides[widgetKey] = "yes"
				} else {
					pagingOverrides[widgetKey] = "no"
				}
			}
		}
	}

	spec := backend.DataGridSpec{
		DataSourceBSON:    datasourceBSON,
		Columns:           columns,
		HeaderWidgetsBSON: headerWidgetsBSON,
		PagingOverrides:   pagingOverrides,
		SelectionMode:     w.GetSelection(),
	}

	// Use gen-native DataGrid2 builder
	grid, err := pb.widgetBackend.BuildDataGrid2WidgetGen(widgetID, w.Name, spec, pb.backend.Path())
	if err != nil {
		return nil, err
	}

	if err := pb.registerWidgetName(w.Name, model.ID(grid.AsElement().ID())); err != nil {
		return nil, err
	}

	return grid.AsElement(), nil
}

// buildDataGridDataSourceBSON builds a pre-serialized bson.D datasource for use in DataGridSpec.
// Returns the datasource BSON document, the resolved entity name, and any error.
func (pb *pageBuilder) buildDataGridDataSourceBSON(ds *ast.DataSourceV3) (bson.D, string, error) {
	switch ds.Type {
	case "parameter":
		paramName := strings.TrimPrefix(ds.Reference, "$")
		entityID, ok := pb.paramScope[paramName]
		entityName := pb.paramEntityNames[paramName]
		if !ok {
			entityID, ok = pb.paramScope["$"+paramName]
			entityName = pb.paramEntityNames["$"+paramName]
		}
		if !ok {
			return nil, "", mdlerrors.NewNotFound("parameter", ds.Reference)
		}
		if entityName == "" {
			var err error
			entityName, err = pb.getEntityNameByID(entityID)
			if err != nil {
				log.Printf("warning: could not resolve entity name for ID %s: %v", entityID, err)
			}
		}
		var entityRef any
		if entityName != "" {
			entityRef = bson.D{
				{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
				{Key: "$Type", Value: "DomainModels$DirectEntityRef"},
				{Key: "Entity", Value: entityName},
			}
		}
		doc := bson.D{
			{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
			{Key: "$Type", Value: "Forms$DataViewSource"},
			{Key: "EntityRef", Value: entityRef},
			{Key: "ForceFullObjects", Value: false},
			{Key: "SourceVariable", Value: nil},
		}
		return doc, entityName, nil

	case "database":
		_, err := pb.resolveEntity(ast.QualifiedName{
			Module: pb.extractModule(ds.Reference),
			Name:   pb.extractName(ds.Reference),
		})
		if err != nil {
			return nil, "", mdlerrors.NewBackend("resolve entity", err)
		}
		var entityRef any
		if ds.Reference != "" {
			entityRef = bson.D{
				{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
				{Key: "$Type", Value: "DomainModels$DirectEntityRef"},
				{Key: "Entity", Value: ds.Reference},
			}
		}
		sortItems := bson.A{int32(2)}
		for _, ob := range ds.OrderBy {
			direction := "Ascending"
			if strings.ToLower(ob.Direction) == "desc" {
				direction = "Descending"
			}
			sortItem := bson.D{
				{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
				{Key: "$Type", Value: "Forms$GridSortItem"},
				{Key: "AttributeRef", Value: bson.D{
					{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
					{Key: "$Type", Value: "DomainModels$AttributeRef"},
					{Key: "Attribute", Value: pb.resolveAttributePathForEntity(ob.Attribute, ds.Reference)},
					{Key: "EntityRef", Value: nil},
				}},
				{Key: "SortOrder", Value: direction},
			}
			sortItems = append(sortItems, sortItem)
		}
		doc := bson.D{
			{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
			{Key: "$Type", Value: "CustomWidgets$CustomWidgetXPathSource"},
			{Key: "EntityRef", Value: entityRef},
			{Key: "ForceFullObjects", Value: false},
			{Key: "SortBar", Value: bson.D{
				{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
				{Key: "$Type", Value: "Forms$GridSortBar"},
				{Key: "SortItems", Value: sortItems},
			}},
			{Key: "SourceVariable", Value: nil},
			{Key: "XPathConstraint", Value: ds.Where},
		}
		return doc, ds.Reference, nil

	case "microflow":
		mfID, err := pb.resolveMicroflow(ds.Reference)
		if err != nil {
			return nil, "", mdlerrors.NewBackend("resolve microflow", err)
		}
		_ = mfID
		entityName := pb.getMicroflowReturnEntityName(ds.Reference)
		doc := bson.D{
			{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
			{Key: "$Type", Value: "Forms$MicroflowSource"},
			{Key: "MicroflowSettings", Value: bson.D{
				{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
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
		return doc, entityName, nil

	case "nanoflow":
		ns, entityName, err := pb.buildNanoflowSourceGen(ds)
		if err != nil {
			return nil, "", err
		}
		doc, err := genElementToBSONDoc(ns)
		if err != nil {
			return nil, "", mdlerrors.NewBackend("encode nanoflow source", err)
		}
		return doc, entityName, nil

	case "association":
		ctxVar := ds.ContextVariable
		if ctxVar == "currentObject" {
			ctxVar = ""
		}
		path := ds.Reference
		destEntity := ""
		if idx := strings.Index(path, "/"); idx >= 0 {
			destEntity = path[idx+1:]
			path = path[:idx]
		} else {
			destEntity = pb.resolveAssociationDestination(path, pb.entityContext)
		}
		step := bson.D{
			{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
			{Key: "$Type", Value: "DomainModels$EntityRefStep"},
			{Key: "Association", Value: path},
			{Key: "DestinationEntity", Value: destEntity},
		}
		entityRef := bson.D{
			{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
			{Key: "$Type", Value: "DomainModels$IndirectEntityRef"},
			{Key: "Steps", Value: bson.A{int32(2), step}},
		}
		var sourceVar any
		if ctxVar != "" {
			sourceVar = bson.D{
				{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
				{Key: "$Type", Value: "Forms$PageVariable"},
				{Key: "LocalVariable", Value: ""},
				{Key: "PageParameter", Value: ctxVar},
				{Key: "SnippetParameter", Value: ""},
				{Key: "SubKey", Value: ""},
				{Key: "UseAllPages", Value: false},
				{Key: "Widget", Value: ""},
			}
		}
		doc := bson.D{
			{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
			{Key: "$Type", Value: "Forms$AssociationSource"},
			{Key: "EntityRef", Value: entityRef},
			{Key: "ForceFullObjects", Value: false},
			{Key: "SourceVariable", Value: sourceVar},
		}
		return doc, destEntity, nil

	case "selection":
		widgetName := ds.Reference
		widgetID, ok := pb.widgetScope[widgetName]
		if !ok {
			return nil, "", mdlerrors.NewNotFound("widget", widgetName)
		}
		_ = widgetID
		entityName := pb.paramEntityNames[widgetName]
		doc := bson.D{
			{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
			{Key: "$Type", Value: "Forms$ListenTargetSource"},
			{Key: "ListenTarget", Value: widgetName},
		}
		return doc, entityName, nil

	default:
		return nil, "", mdlerrors.NewUnsupported("unsupported datasource type: " + ds.Type)
	}
}

// buildWidgetBSON builds a pre-serialized bson.D widget for use in DataGridSpec.
// Used internally by buildDataGridV3 for column child widgets and controlbar widgets.
func (pb *pageBuilder) buildWidgetBSON(w *ast.WidgetV3) (bson.D, error) {
	switch strings.ToLower(w.Type) {
	case "button", "actionbutton":
		buttonStyle := "Default"
		if style := w.GetButtonStyle(); style != "" {
			buttonStyle = style
		}
		// Build action BSON
		actionBSON := bson.D{
			{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
			{Key: "$Type", Value: "Forms$NoAction"},
			{Key: "DisabledDuringExecution", Value: true},
		}
		if action := w.GetAction(); action != nil {
			ab, err := pb.genClientActionToBsonD(action)
			if err != nil {
				return nil, mdlerrors.NewBackend("build action", err)
			}
			if ab != nil {
				actionBSON = ab
			}
		}
		// Build caption template
		captionTemplate := buildMinimalClientTemplate(w.GetCaption())
		doc := bson.D{
			{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
			{Key: "$Type", Value: "Forms$ActionButton"},
			{Key: "Action", Value: actionBSON},
			{Key: "Appearance", Value: buildMinimalAppearance()},
			{Key: "AriaRole", Value: "Button"},
			{Key: "ButtonStyle", Value: buttonStyle},
			{Key: "CaptionTemplate", Value: captionTemplate},
			{Key: "ConditionalVisibilitySettings", Value: nil},
			{Key: "Icon", Value: nil},
			{Key: "Name", Value: w.Name},
			{Key: "NativeAccessibilitySettings", Value: nil},
			{Key: "RenderType", Value: "Button"},
			{Key: "TabIndex", Value: int64(0)},
			{Key: "Tooltip", Value: bson.D{
				{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
				{Key: "$Type", Value: "Texts$Text"},
				{Key: "Items", Value: bson.A{int32(3)}},
			}},
		}
		return doc, nil
	case "textfilter", "numberfilter", "datefilter", "dropdownfilter":
		filterWidgetID := dataGridFilterWidgetID(w.Type)
		rawAttrs := w.GetAttributes()
		resolvedAttrs := make([]string, 0, len(rawAttrs))
		for _, a := range rawAttrs {
			resolvedAttrs = append(resolvedAttrs, pb.resolveAttributePath(a))
		}
		fw, err := pb.widgetBackend.BuildFilterWidgetGen(backend.FilterWidgetSpec{
			WidgetID:   filterWidgetID,
			FilterName: w.Name,
			FilterType: w.GetFilterType(),
			Attributes: resolvedAttrs,
		}, pb.backend.Path())
		if err != nil {
			return nil, mdlerrors.NewBackend("build controlbar filter widget", err)
		}
		return pb.serializeGenWidgetToBsonD(fw)
	default:
		// For other widget types in DataGrid context, create a minimal DivContainer
		doc := bson.D{
			{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
			{Key: "$Type", Value: "Forms$DivContainer"},
			{Key: "Appearance", Value: buildMinimalAppearance()},
			{Key: "ConditionalVisibilitySettings", Value: nil},
			{Key: "Name", Value: w.Name},
			{Key: "NativeAccessibilitySettings", Value: nil},
			{Key: "OnClickAction", Value: bson.D{
				{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
				{Key: "$Type", Value: "Forms$NoAction"},
				{Key: "DisabledDuringExecution", Value: true},
			}},
			{Key: "RenderMode", Value: "Div"},
			{Key: "ScreenReaderHidden", Value: false},
			{Key: "TabIndex", Value: int64(0)},
			{Key: "Widgets", Value: bson.A{int32(2)}},
		}
		return doc, nil
	}
}

// genClientActionToBsonD builds a pre-serialized bson.D client action for DataGrid button widgets.
// It delegates to buildClientActionV3 (the gen path) and serializes the result via the codec,
// eliminating hand-rolled BSON that was prone to wrong type names and missing fields.
func (pb *pageBuilder) genClientActionToBsonD(action *ast.ActionV3) (bson.D, error) {
	elem, err := pb.buildClientActionV3(action)
	if err != nil {
		return nil, err
	}
	if elem == nil {
		return nil, nil
	}
	opaque := pb.widgetBackend.SerializeGenElemToOpaque(elem)
	if opaque == nil {
		return nil, fmt.Errorf("genClientActionToBsonD: serialize returned nil for action type %q", action.Type)
	}
	var doc bson.D
	switch v := opaque.(type) {
	case bson.Raw:
		if err := bson.Unmarshal([]byte(v), &doc); err != nil {
			return nil, fmt.Errorf("genClientActionToBsonD: unmarshal: %w", err)
		}
	case bson.D:
		doc = v
	default:
		return nil, fmt.Errorf("genClientActionToBsonD: unexpected opaque type %T", opaque)
	}
	return doc, nil
}

// buildPluggableDataSourceOpaque builds the datasource for pluggable widgets (Gallery, etc.)
// using modelsdk gen types (CustomWidgetXPathSource) serialised via the codec path so no
// direct bson usage is needed here or in widget_engine.go.
func (pb *pageBuilder) buildPluggableDataSourceOpaque(ds *ast.DataSourceV3) (backend.OpaqueWidget, string, error) {
	src := genCW.NewCustomWidgetXPathSource()
	assignFreshID(src)
	src.SetForceFullObjects(false)

	entityName := ""
	switch ds.Type {
	case "database":
		_, err := pb.resolveEntity(ast.QualifiedName{
			Module: pb.extractModule(ds.Reference),
			Name:   pb.extractName(ds.Reference),
		})
		if err != nil {
			log.Printf("warning: buildPluggableDataSourceOpaque: entity %s not found: %v", ds.Reference, err)
		}
		entityName = ds.Reference
		ref := genDm.NewDirectEntityRef()
		assignFreshID(ref)
		ref.SetEntityQualifiedName(ds.Reference)
		src.SetEntityRef(ref)

		// Build sort bar if sort order is specified.
		sortBar := genPg.NewGridSortBar()
		assignFreshID(sortBar)
		for _, ob := range ds.OrderBy {
			item := genPg.NewGridSortItem()
			assignFreshID(item)
			attrRef := genDm.NewAttributeRef()
			assignFreshID(attrRef)
			attrRef.SetAttributeQualifiedName(pb.resolveAttributePathForEntity(ob.Attribute, ds.Reference))
			item.SetAttributeRef(attrRef)
			direction := "Ascending"
			if ob.Direction == "DESC" {
				direction = "Descending"
			}
			item.SetSortDirection(direction)
			sortBar.AddSortItems(item)
		}
		src.SetSortBar(sortBar)
		if ds.Where != "" {
			src.SetXPathConstraint(ds.Where)
		}
	default:
		// For non-database types (parameter, microflow, selection) fall back to
		// the gen DataViewSource path via the standard builder.
		elem, eName, err := pb.buildDataSourceV3(ds)
		if err != nil {
			return nil, "", err
		}
		opaque := pb.widgetBackend.SerializeGenElemToOpaque(elem)
		return opaque, eName, nil
	}

	opaque := pb.widgetBackend.SerializeGenElemToOpaque(src)
	if opaque == nil {
		return nil, "", fmt.Errorf("buildPluggableDataSourceOpaque: serialize returned nil")
	}
	return opaque, entityName, nil
}

// serializeGenWidgetToBsonD encodes a GenCustomWidgetElem to bson.D.
// Used for pre-serializing filter widgets into DataGridColumnSpec.FilterWidgetBSON.
func (pb *pageBuilder) serializeGenWidgetToBsonD(fw *backend.GenCustomWidgetElem) (bson.D, error) {
	if fw == nil {
		return nil, nil
	}
	opaque := pb.widgetBackend.SerializeGenElemToOpaque(fw.AsElement())
	if opaque == nil {
		return nil, fmt.Errorf("serializeGenWidgetToBsonD: encode returned nil")
	}
	// SerializeGenElemToOpaque returns bson.Raw — unmarshal to bson.D for embedding.
	var doc bson.D
	switch v := opaque.(type) {
	case bson.Raw:
		if err := bson.Unmarshal([]byte(v), &doc); err != nil {
			return nil, fmt.Errorf("serializeGenWidgetToBsonD: unmarshal: %w", err)
		}
	case bson.D:
		doc = v
	default:
		return nil, fmt.Errorf("serializeGenWidgetToBsonD: unexpected opaque type %T", opaque)
	}
	return doc, nil
}

// buildMinimalClientTemplate builds a minimal Forms$ClientTemplate BSON with optional text.
func buildMinimalClientTemplate(text string) bson.D {
	var templateItems bson.A
	if text != "" {
		templateItems = bson.A{int32(3), bson.D{
			{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
			{Key: "$Type", Value: "Texts$Translation"},
			{Key: "LanguageCode", Value: "en_US"},
			{Key: "Text", Value: text},
		}}
	} else {
		templateItems = bson.A{int32(3)}
	}
	return bson.D{
		{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
		{Key: "$Type", Value: "Forms$ClientTemplate"},
		{Key: "Fallback", Value: bson.D{
			{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
			{Key: "$Type", Value: "Texts$Text"},
			{Key: "Items", Value: bson.A{int32(3)}},
		}},
		{Key: "Parameters", Value: bson.A{int32(2)}},
		{Key: "Template", Value: bson.D{
			{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
			{Key: "$Type", Value: "Texts$Text"},
			{Key: "Items", Value: templateItems},
		}},
	}
}

// buildMinimalAppearance builds a minimal Forms$Appearance BSON.
func buildMinimalAppearance() bson.D {
	return bson.D{
		{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
		{Key: "$Type", Value: "Forms$Appearance"},
		{Key: "Class", Value: ""},
		{Key: "DesignProperties", Value: bson.A{int32(3)}},
		{Key: "DynamicClasses", Value: ""},
		{Key: "Style", Value: ""},
	}
}

func (pb *pageBuilder) buildListViewV3(w *ast.WidgetV3) (element.Element, error) {
	lv := genPg.NewListView()
	assignFreshID(lv)
	lv.SetName(w.Name)
	lv.SetPageSize(20)

	if ds := w.GetDataSource(); ds != nil {
		dataSource, entityName, err := pb.buildListViewDataSourceV3(ds)
		if err != nil {
			return nil, mdlerrors.NewBackend("build datasource", err)
		}
		lv.SetDataSource(dataSource)

		oldContext := pb.entityContext
		pb.entityContext = entityName
		defer func() { pb.entityContext = oldContext }()

		if w.Name != "" && entityName != "" {
			pb.paramEntityNames[w.Name] = entityName
		}
	}

	if err := pb.registerWidgetName(w.Name, model.ID(lv.ID())); err != nil {
		return nil, err
	}

	for _, child := range w.Children {
		widget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		lv.AddWidgets(widget)
	}

	return lv, nil
}

func (pb *pageBuilder) buildTextBoxV3(w *ast.WidgetV3) (element.Element, error) {
	tb := genPg.NewTextBox()
	assignFreshID(tb)
	tb.SetName(w.Name)
	applyFormWidgetDefaults(tb)

	if attr := w.GetAttribute(); attr != "" {
		tb.SetAttributeRef(newAttributeRef(pb.resolveAttributePath(attr)))
	}

	if label := w.GetLabel(); label != "" {
		tb.SetLabel(genSimpleLabel(label))
	}

	if err := pb.registerWidgetName(w.Name, model.ID(tb.ID())); err != nil {
		return nil, err
	}

	return tb, nil
}

func (pb *pageBuilder) buildTextAreaV3(w *ast.WidgetV3) (element.Element, error) {
	ta := genPg.NewTextArea()
	assignFreshID(ta)
	ta.SetName(w.Name)
	applyFormWidgetDefaults(ta)

	if attr := w.GetAttribute(); attr != "" {
		ta.SetAttributeRef(newAttributeRef(pb.resolveAttributePath(attr)))
	}

	if label := w.GetLabel(); label != "" {
		ta.SetLabel(genSimpleLabel(label))
	}

	if err := pb.registerWidgetName(w.Name, model.ID(ta.ID())); err != nil {
		return nil, err
	}

	return ta, nil
}

func (pb *pageBuilder) buildDatePickerV3(w *ast.WidgetV3) (element.Element, error) {
	dp := genPg.NewDatePicker()
	assignFreshID(dp)
	dp.SetName(w.Name)
	applyFormWidgetDefaults(dp)

	if attr := w.GetAttribute(); attr != "" {
		dp.SetAttributeRef(newAttributeRef(pb.resolveAttributePath(attr)))
	}

	if label := w.GetLabel(); label != "" {
		dp.SetLabel(genSimpleLabel(label))
	}

	// ConditionalVisibility/Editability must be explicit null — DatePicker does not
	// implement textLike so applyFormWidgetDefaults does not set them, but Mendix
	// requires these fields to validate attribute type (CE2421 without them).
	dp.SetConditionalVisibilitySettings(nil)
	dp.SetConditionalEditabilitySettings(nil)
	// FormattingInfo is required for DatePicker — without it Mendix cannot validate
	// the attribute type and raises CE2421. All fields must be set to match Studio Pro.
	fi := genPg.NewFormattingInfo()
	assignFreshID(fi)
	fi.SetDateFormat("DateTime")
	fi.SetCustomDateFormat("")
	fi.SetDecimalPrecision(2)
	fi.SetEnumFormat("Text")
	fi.SetGroupDigits(false)
	dp.SetFormattingInfo(fi)

	if err := pb.registerWidgetName(w.Name, model.ID(dp.ID())); err != nil {
		return nil, err
	}

	return dp, nil
}

func (pb *pageBuilder) buildDropdownV3(w *ast.WidgetV3) (element.Element, error) {
	dd := genPg.NewDropDown()
	assignFreshID(dd)
	dd.SetName(w.Name)
	applyFormWidgetDefaults(dd)

	if attr := w.GetAttribute(); attr != "" {
		dd.SetAttributeRef(newAttributeRef(pb.resolveAttributePath(attr)))
	}

	if label := w.GetLabel(); label != "" {
		dd.SetLabel(genSimpleLabel(label))
	}

	if err := pb.registerWidgetName(w.Name, model.ID(dd.ID())); err != nil {
		return nil, err
	}

	return dd, nil
}

func (pb *pageBuilder) buildCheckBoxV3(w *ast.WidgetV3) (element.Element, error) {
	cb := genPg.NewCheckBox()
	assignFreshID(cb)
	cb.SetName(w.Name)
	applyFormWidgetDefaults(cb)

	if attr := w.GetAttribute(); attr != "" {
		cb.SetAttributeRef(newAttributeRef(pb.resolveAttributePath(attr)))
	}

	if label := w.GetLabel(); label != "" {
		cb.SetLabel(genSimpleLabel(label))
	}

	if err := pb.registerWidgetName(w.Name, model.ID(cb.ID())); err != nil {
		return nil, err
	}

	return cb, nil
}

// buildRadioButtonsV3 creates a RadioButtonGroup from V3 syntax.
func (pb *pageBuilder) buildRadioButtonsV3(w *ast.WidgetV3) (element.Element, error) {
	rb := genPg.NewRadioButtonGroup()
	assignFreshID(rb)
	rb.SetName(w.Name)
	applyFormWidgetDefaults(rb)

	if attr := w.GetAttribute(); attr != "" {
		rb.SetAttributeRef(newAttributeRef(pb.resolveAttributePath(attr)))
	}

	if label := w.GetLabel(); label != "" {
		rb.SetLabel(genSimpleLabel(label))
	}

	if err := pb.registerWidgetName(w.Name, model.ID(rb.ID())); err != nil {
		return nil, err
	}

	return rb, nil
}

func (pb *pageBuilder) buildTextWidgetV3(w *ast.WidgetV3) (element.Element, error) {
	lbl := genPg.NewLabel()
	assignFreshID(lbl)
	lbl.SetName(w.Name)

	if content := w.GetContent(); content != "" {
		lbl.SetCaption(genSimpleLabel(content))
	}

	if err := pb.registerWidgetName(w.Name, model.ID(lbl.ID())); err != nil {
		return nil, err
	}

	return lbl, nil
}

func (pb *pageBuilder) buildDynamicTextV3(w *ast.WidgetV3) (element.Element, error) {
	dt := genPg.NewDynamicText()
	assignFreshID(dt)
	dt.SetName(w.Name)

	if rm := w.GetRenderMode(); rm != "" {
		dt.SetRenderMode(rm)
	}

	content := w.GetContent()
	explicitParams := w.GetContentParams()

	var autoGeneratedParams []string
	if content != "" && explicitParams == nil {
		isEntityPath := false
		if strings.Contains(content, ".") && !strings.HasPrefix(content, "$") {
			isEntityPath = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*\.[A-Za-z_][A-Za-z0-9_]*$`).MatchString(content)
		}
		if strings.HasPrefix(content, "$") || isEntityPath {
			autoGeneratedParams = append(autoGeneratedParams, content)
			content = "{1}"
		}
	}

	if content == "" {
		// Empty content would produce a {1} template with 0 parameters,
		// causing Studio Pro CE0720 "Place holder index 1 is greater than 0".
		// Use a single space as a safe placeholder (CLAUDE.md note).
		content = " "
	}

	ct := genPg.NewClientTemplate()
	assignFreshID(ct)
	ct.SetTemplate(genSimpleText(content))
	fallbackText := genTexts.NewText()
	assignFreshID(fallbackText)
	ct.SetFallback(fallbackText)

	// Add auto-generated parameters first
	for _, attrRef := range autoGeneratedParams {
		param := genPg.NewClientTemplateParameter()
		assignFreshID(param)
		pb.resolveTemplateAttributePathFull(attrRef, param)
		ct.AddParameters(param)
	}

	// Handle explicit ContentParams
	if explicitParams != nil {
		for _, p := range explicitParams {
			param := genPg.NewClientTemplateParameter()
			assignFreshID(param)
			if strVal, ok := p.Value.(string); ok {
				if strings.HasPrefix(strVal, "'") || strings.HasPrefix(strVal, "\"") {
					param.SetExpression(strVal)
				} else if strings.HasPrefix(strVal, "$") || strings.Contains(strVal, ".") {
					pb.resolveTemplateAttributePathFull(strVal, param)
				} else {
					pb.resolveTemplateAttributePathFull(strVal, param)
				}
			}
			ct.AddParameters(param)
		}
	}

	dt.SetContent(ct)

	if err := pb.registerWidgetName(w.Name, model.ID(dt.ID())); err != nil {
		return nil, err
	}

	return dt, nil
}

func (pb *pageBuilder) buildTitleV3(w *ast.WidgetV3) (element.Element, error) {
	title := genPg.NewTitle()
	assignFreshID(title)
	title.SetName(w.Name)

	if content := w.GetContent(); content != "" {
		setRawBSONField(title, "Caption", content)
	}

	if err := pb.registerWidgetName(w.Name, model.ID(title.ID())); err != nil {
		return nil, err
	}

	return title, nil
}

func (pb *pageBuilder) buildButtonV3(w *ast.WidgetV3) (element.Element, error) {
	btn := genPg.NewActionButton()
	assignFreshID(btn)
	btn.SetName(w.Name)
	btn.SetButtonStyle("Default")

	if caption := w.GetCaption(); caption != "" {
		// Build caption ClientTemplate
		ct := genPg.NewClientTemplate()
		assignFreshID(ct)
		ct.SetTemplate(genSimpleText(caption))
		fallback := genTexts.NewText()
		assignFreshID(fallback)
		ct.SetFallback(fallback)

		// Handle CaptionParams
		if params := w.GetCaptionParams(); params != nil {
			for _, p := range params {
				param := genPg.NewClientTemplateParameter()
				assignFreshID(param)
				if strVal, ok := p.Value.(string); ok {
					if strings.HasPrefix(strVal, "'") || strings.HasPrefix(strVal, "\"") {
						param.SetExpression(strVal)
					} else if strings.HasPrefix(strVal, "$") || strings.Contains(strVal, ".") {
						param.SetAttributePath(pb.resolveTemplateAttributePath(strVal))
					} else {
						param.SetExpression("'" + strVal + "'")
					}
				}
				ct.AddParameters(param)
			}
		}
		btn.SetCaption(ct)
	}

	if style := w.GetButtonStyle(); style != "" {
		btn.SetButtonStyle(style)
	}

	if action := w.GetAction(); action != nil {
		act, err := pb.buildClientActionV3(action)
		if err != nil {
			return nil, mdlerrors.NewBackend("build action", err)
		}
		btn.SetAction(act)
	}

	if err := pb.registerWidgetName(w.Name, model.ID(btn.ID())); err != nil {
		return nil, err
	}

	return btn, nil
}

// buildNavigationListV3 creates a NavigationList widget from V3 syntax.
func (pb *pageBuilder) buildNavigationListV3(w *ast.WidgetV3) (element.Element, error) {
	nl := genPg.NewNavigationList()
	assignFreshID(nl)
	nl.SetName(w.Name)

	for _, child := range w.Children {
		if strings.ToLower(child.Type) == "item" {
			item, err := pb.buildNavigationListItemV3(child)
			if err != nil {
				return nil, err
			}
			nl.AddItems(item)
		}
	}

	if err := pb.registerWidgetName(w.Name, model.ID(nl.ID())); err != nil {
		return nil, err
	}

	return nl, nil
}

// buildNavigationListItemV3 creates a NavigationListItem from V3 syntax.
func (pb *pageBuilder) buildNavigationListItemV3(w *ast.WidgetV3) (element.Element, error) {
	if w.Name == "" {
		return nil, mdlerrors.NewValidation("item inside navigationlist requires a name")
	}

	item := genPg.NewNavigationListItem()
	assignFreshID(item)
	// NavigationListItem has no SetName in gen — inject via raw BSON
	setRawBSONField(item, "Name", w.Name)

	if err := pb.registerWidgetName(w.Name, model.ID(item.ID())); err != nil {
		return nil, err
	}

	if action := w.GetAction(); action != nil {
		clientAction, err := pb.buildClientActionV3(action)
		if err != nil {
			return nil, err
		}
		item.SetAction(clientAction)
	}

	// Build explicit child widgets or create a DynamicText from Caption
	if len(w.Children) > 0 {
		for _, child := range w.Children {
			childWidget, err := pb.buildWidgetV3(child)
			if err != nil {
				return nil, err
			}
			item.AddWidgets(childWidget)
		}
	} else if caption := w.GetCaption(); caption != "" {
		// Create DynamicText for caption (matches old SDK serializer behavior)
		dt := genPg.NewDynamicText()
		assignFreshID(dt)
		dt.SetName("text_" + w.Name)
		ct := genPg.NewClientTemplate()
		assignFreshID(ct)
		ct.SetTemplate(genSimpleText(caption))
		fallback := genTexts.NewText()
		assignFreshID(fallback)
		ct.SetFallback(fallback)
		dt.SetContent(ct)
		item.AddWidgets(dt)
	}

	return item, nil
}

// buildSnippetCallV3 creates a SnippetCallWidget from V3 syntax.
func (pb *pageBuilder) buildSnippetCallV3(w *ast.WidgetV3) (element.Element, error) {
	scw := genPg.NewSnippetCallWidget()
	assignFreshID(scw)
	scw.SetName(w.Name)

	snippetName := w.GetSnippet()
	if snippetName != "" {
		snippetID, err := pb.resolveSnippetRef(snippetName)
		if err != nil {
			return nil, mdlerrors.NewBackend(fmt.Sprintf("resolve snippet %s", snippetName), err)
		}
		_ = snippetID

		// Build SnippetCall sub-element
		sc := genPg.NewSnippetCall()
		assignFreshID(sc)
		sc.SetSnippetQualifiedName(snippetName)

		// Validate and wire up parameter mappings
		if err := pb.buildSnippetCallParamsGen(sc, snippetName, w.GetSnippetParams()); err != nil {
			return nil, err
		}

		scw.SetSnippetCall(sc)
	}

	if err := pb.registerWidgetName(w.Name, model.ID(scw.ID())); err != nil {
		return nil, err
	}

	return scw, nil
}

// buildSnippetCallParamsGen validates the supplied param mappings against the
// snippet's declared parameters and populates sc.ParameterMappings.
func (pb *pageBuilder) buildSnippetCallParamsGen(sc *genPg.SnippetCall, snippetQName string, supplied []ast.SnippetCallParam) error {
	snippets, err := pb.backend.ListSnippetsGen()
	if err != nil {
		return err
	}

	var targetSnippet *genPg.Snippet
	for _, s := range snippets {
		if s == nil {
			continue
		}
		name := s.Name()
		if name != "" && (name == snippetQName || strings.HasSuffix(snippetQName, "."+name)) {
			targetSnippet = s
			break
		}
	}
	if targetSnippet == nil || len(targetSnippet.ParametersItems()) == 0 {
		return nil
	}

	suppliedByName := make(map[string]string, len(supplied))
	for _, p := range supplied {
		name := strings.TrimPrefix(p.ParamName, "$")
		suppliedByName[name] = p.Variable
	}

	for _, rawParam := range targetSnippet.ParametersItems() {
		sp, ok := rawParam.(*genPg.SnippetParameter)
		if !ok || sp == nil {
			continue
		}
		paramName := sp.Name()
		argument, found := suppliedByName[paramName]
		if !found {
			return mdlerrors.NewValidationf(
				"snippet %s requires parameter $%s — add Params: {%s: $<variable>} to the SNIPPETCALL",
				snippetQName, paramName, paramName,
			)
		}

		pm := genPg.NewSnippetParameterMapping()
		assignFreshID(pm)
		// Full qualified name: Module.SnippetName.ParamName
		pm.SetParameterQualifiedName(snippetQName + "." + paramName)

		// Build PageVariable using gen modelsdk to hold the argument reference.
		varName := strings.TrimPrefix(argument, "$")
		pv := genPg.NewPageVariable()
		assignFreshID(pv)
		if _, isParam := pb.paramEntityNames[varName]; isParam {
			pv.SetPageParameterQualifiedName(varName)
		} else {
			pv.SetLocalVariableQualifiedName(varName)
		}
		pm.SetVariable(pv)
		sc.AddParameterMappings(pm)
	}

	return nil
}

// buildTemplateV3 creates a DivContainer to hold template content.
func (pb *pageBuilder) buildTemplateV3(w *ast.WidgetV3) (element.Element, error) {
	container := genPg.NewDivContainer()
	assignFreshID(container)
	container.SetName(w.Name)

	for _, child := range w.Children {
		childWidget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		container.AddWidgets(childWidget)
	}

	return container, nil
}

// buildFilterV3 creates a DivContainer to hold filter widgets.
func (pb *pageBuilder) buildFilterV3(w *ast.WidgetV3) (element.Element, error) {
	container := genPg.NewDivContainer()
	assignFreshID(container)
	container.SetName(w.Name)

	for _, child := range w.Children {
		childWidget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		container.AddWidgets(childWidget)
	}

	return container, nil
}

func (pb *pageBuilder) buildStaticImageV3(w *ast.WidgetV3) (element.Element, error) {
	img := genPg.NewStaticImageViewer()
	assignFreshID(img)
	img.SetName(w.Name)
	img.SetResponsive(true)

	if width := w.GetIntProp("Width"); width > 0 {
		img.SetWidth(int32(width))
	}
	if height := w.GetIntProp("Height"); height > 0 {
		img.SetHeight(int32(height))
	}

	if err := pb.registerWidgetName(w.Name, model.ID(img.ID())); err != nil {
		return nil, err
	}

	return img, nil
}

func (pb *pageBuilder) buildDynamicImageV3(w *ast.WidgetV3) (element.Element, error) {
	img := genPg.NewDynamicImageViewer()
	assignFreshID(img)
	img.SetName(w.Name)
	img.SetResponsive(true)

	if width := w.GetIntProp("Width"); width > 0 {
		img.SetWidth(int32(width))
	}
	if height := w.GetIntProp("Height"); height > 0 {
		img.SetHeight(int32(height))
	}

	if err := pb.registerWidgetName(w.Name, model.ID(img.ID())); err != nil {
		return nil, err
	}

	return img, nil
}

// dataGridFilterWidgetID maps a MDL filter type keyword to its pluggable widget ID.
// Returns "" for non-filter widget types.
func dataGridFilterWidgetID(widgetType string) string {
	switch strings.ToLower(widgetType) {
	case "textfilter":
		return widgetIDDataGridTextFilter
	case "numberfilter":
		return widgetIDDataGridNumberFilter
	case "datefilter":
		return widgetIDDataGridDateFilter
	case "dropdownfilter":
		return widgetIDDataGridDropdownFilter
	}
	return ""
}

// dataGridPagingPropMap maps lowercase MDL property names to camelCase widget property keys.
// The visitor stores generic widget properties with the literal text from the grammar (lowercase),
// so these keys must match the lowercased MDL keyword form.
var dataGridPagingPropMap = map[string]string{
	"pagesize":          "pageSize",
	"pagination":        "pagination",
	"pagingposition":    "pagingPosition",
	"showpagingbuttons": "showPagingButtons",
}
