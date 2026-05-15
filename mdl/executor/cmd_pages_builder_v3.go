// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// ============================================================================
// V3 Page Builder
// ============================================================================

// buildPageV3 creates a Page from a CreatePageStmtV3.
func (pb *pageBuilder) buildPageV3(s *ast.CreatePageStmtV3) (*backend.Page, error) {
	// Resolve folder if specified
	containerID := pb.moduleID
	if s.Folder != "" {
		folderID, err := pb.resolveFolder(s.Folder)
		if err != nil {
			return nil, mdlerrors.NewBackend("resolve folder "+s.Folder, err)
		}
		containerID = folderID
	}

	page := &backend.Page{
		BaseElement: model.BaseElement{
			ID:       model.ID(types.GenerateID()),
			TypeName: "Forms$Page",
		},
		ContainerID:   containerID,
		Name:          s.Name.Name,
		Documentation: s.Documentation,
		URL:           s.URL,
		MarkAsUsed:    false,
		Excluded:      s.Excluded,
	}

	// Set title
	if s.Title != "" {
		page.Title = &model.Text{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Texts$Text",
			},
			Translations: map[string]string{"en_US": s.Title},
		}
	}

	// Resolve layout
	if s.Layout != "" {
		layoutID, err := pb.resolveLayout(s.Layout)
		if err != nil {
			// Layout not found is not fatal - page will work but may not render correctly
			log.Printf("warning: layout %s not found", s.Layout)
		} else {
			page.LayoutID = layoutID

			// Create LayoutCall with arguments for placeholders
			page.LayoutCall = &backend.LayoutCall{
				BaseElement: model.BaseElement{
					ID:       model.ID(types.GenerateID()),
					TypeName: "Forms$LayoutCall",
				},
				LayoutID:   layoutID,
				LayoutName: s.Layout, // Qualified name for "Form" field in BSON
			}
		}
	}

	// Build parameters
	for _, param := range s.Parameters {
		pageParam := &backend.PageParameter{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$PageParameter",
			},
			ContainerID: page.ID,
			Name:        param.Name,
			IsRequired:  true, // Page parameters are required by default
		}

		// Check if this is a primitive type or entity type
		if bsonType := pageParamBSONType(param.Type); bsonType != "" {
			// Primitive type parameter
			pageParam.TypeName = bsonType
		} else if param.EntityType.Name != "" {
			// Entity type parameter
			entityID, err := pb.resolveEntity(param.EntityType)
			if err != nil {
				return nil, mdlerrors.NewBackend("resolve entity "+param.EntityType.String(), err)
			}
			entityName := param.EntityType.String()
			pageParam.EntityID = entityID
			pageParam.EntityName = entityName // Qualified entity name for BSON
			pb.paramScope[param.Name] = entityID
			pb.paramEntityNames[param.Name] = entityName
		}

		page.Parameters = append(page.Parameters, pageParam)
	}

	// Build variables
	for _, v := range s.Variables {
		localVar := &backend.LocalVariable{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$LocalVariable",
			},
			ContainerID:  page.ID,
			Name:         v.Name,
			DefaultValue: v.DefaultValue,
			VariableType: mdlTypeToBsonType(v.DataType),
		}
		page.Variables = append(page.Variables, localVar)
	}

	// Build FormCallArgument for the main placeholder
	if page.LayoutCall != nil {
		mainPlaceholderRef := pb.getMainPlaceholderRef(s.Layout)

		arg := &backend.LayoutCallArgument{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$FormCallArgument",
			},
			ParameterID: model.ID(mainPlaceholderRef),
		}

		// Build V3 widgets (expanding fragments)
		if len(s.Widgets) > 0 {
			containerWidget := &backend.Container{
				BaseWidget: backend.BaseWidget{
					BaseElement: model.BaseElement{
						ID:       model.ID(types.GenerateID()),
						TypeName: "Forms$DivContainer",
					},
					Name: "conditionalVisibilityWidget1",
				},
			}

			expanded, err := pb.expandFragments(s.Widgets)
			if err != nil {
				return nil, err
			}
			for _, astWidget := range expanded {
				w, err := pb.buildWidgetV3(astWidget)
				if err != nil {
					return nil, mdlerrors.NewBackend("build widget", err)
				}
				containerWidget.Widgets = append(containerWidget.Widgets, w)
			}

			arg.Widget = containerWidget
		}

		page.LayoutCall.Arguments = append(page.LayoutCall.Arguments, arg)
	}

	return page, nil
}

// buildSnippetV3 creates a Snippet from a CreateSnippetStmtV3.
func (pb *pageBuilder) buildSnippetV3(s *ast.CreateSnippetStmtV3) (*backend.Snippet, error) {
	// Resolve folder if specified
	containerID := pb.moduleID
	if s.Folder != "" {
		folderID, err := pb.resolveFolder(s.Folder)
		if err != nil {
			return nil, mdlerrors.NewBackend("resolve folder "+s.Folder, err)
		}
		containerID = folderID
	}

	snippet := &backend.Snippet{
		BaseElement: model.BaseElement{
			ID:       model.ID(types.GenerateID()),
			TypeName: "Forms$Snippet",
		},
		ContainerID:   containerID,
		Name:          s.Name.Name,
		Documentation: s.Documentation,
	}

	// Build parameters
	for _, param := range s.Parameters {
		snippetParam := &backend.SnippetParameter{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$SnippetParameter",
			},
			ContainerID: snippet.ID,
			Name:        param.Name,
		}

		// Resolve entity type
		if param.EntityType.Name != "" {
			entityID, err := pb.resolveEntity(param.EntityType)
			if err != nil {
				return nil, mdlerrors.NewBackend("resolve entity "+param.EntityType.String(), err)
			}
			entityName := param.EntityType.String()
			snippetParam.EntityID = entityID
			snippetParam.EntityName = entityName
			pb.paramScope[param.Name] = entityID
			pb.paramEntityNames[param.Name] = entityName
		}

		snippet.Parameters = append(snippet.Parameters, snippetParam)
	}

	// Build variables
	for _, v := range s.Variables {
		localVar := &backend.LocalVariable{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$LocalVariable",
			},
			ContainerID:  snippet.ID,
			Name:         v.Name,
			DefaultValue: v.DefaultValue,
			VariableType: mdlTypeToBsonType(v.DataType),
		}
		snippet.Variables = append(snippet.Variables, localVar)
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
		snippet.Widgets = append(snippet.Widgets, w)
	}

	return snippet, nil
}

// buildWidgetV3 converts a V3 AST widget to a backend.Widget.
func (pb *pageBuilder) buildWidgetV3(w *ast.WidgetV3) (backend.Widget, error) {
	var widget backend.Widget
	var err error

	switch strings.ToLower(w.Type) {
	case "dataview":
		widget, err = pb.buildDataViewV3(w)
	case "datagrid":
		widget, err = pb.buildDataGridV3(w)
	case "listview":
		widget, err = pb.buildListViewV3(w)
	case "layoutgrid":
		widget, err = pb.buildLayoutGridV3(w)
	case "row":
		// ROW creates a container with LayoutGrid that contains one row
		widget, err = pb.buildContainerWithRowV3(w)
	case "column":
		// COLUMN creates a container with LayoutGrid that contains one column
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
		// Tab pages are handled inside TabContainer
		return nil, mdlerrors.NewValidation("tabpage must be a direct child of tabcontainer")
	case "groupbox":
		widget, err = pb.buildGroupBoxV3(w)
	case "radiobuttons":
		widget, err = pb.buildRadioButtonsV3(w)
	case "navigationlist":
		widget, err = pb.buildNavigationListV3(w)
	case "item":
		// Items are handled inside NavigationList
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
				return pb.pluggableEngine.Build(def, w)
			}
		}
		// Fallback to static image if pluggable engine unavailable
		widget, err = pb.buildStaticImageV3(w)
	default:
		pb.initPluggableEngine()
		if pb.widgetRegistry != nil {
			// Try by MDL name first
			if def, ok := pb.widgetRegistry.Get(strings.ToUpper(w.Type)); ok {
				return pb.pluggableEngine.Build(def, w)
			}
			// PLUGGABLEWIDGET/CUSTOMWIDGET 'widget.id' name — lookup by widget ID
			if w.Type == "pluggablewidget" || w.Type == "customwidget" {
				if widgetType, ok := w.Properties["WidgetType"].(string); ok {
					if def, ok := pb.widgetRegistry.GetByWidgetID(widgetType); ok {
						return pb.pluggableEngine.Build(def, w)
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

	// Apply Class/Style appearance properties to the widget
	applyWidgetAppearance(widget, w, pb.themeRegistry)

	// Apply conditional visibility/editability
	applyConditionalSettings(widget, w)

	return widget, nil
}

// applyConditionalSettings sets ConditionalVisibility and ConditionalEditability
// on a widget if VISIBLE IF or EDITABLE IF properties are specified in the AST.
func applyConditionalSettings(widget backend.Widget, w *ast.WidgetV3) {
	type baseWidgetGetter interface {
		GetBaseWidget() *backend.BaseWidget
	}
	bwg, ok := widget.(baseWidgetGetter)
	if !ok {
		return
	}
	bw := bwg.GetBaseWidget()

	if visibleIf := w.GetStringProp("VisibleIf"); visibleIf != "" {
		bw.ConditionalVisibility = &backend.ConditionalVisibilitySettings{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$ConditionalVisibilitySettings",
			},
			Expression: visibleIf,
		}
	}

	if editableIf := w.GetStringProp("EditableIf"); editableIf != "" {
		bw.ConditionalEditability = &backend.ConditionalEditabilitySettings{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$ConditionalEditabilitySettings",
			},
			Expression: editableIf,
		}
	}
}

// applyWidgetAppearance sets Class, Style, and DesignProperties on a widget if specified in the AST.
// The theme registry (if non-nil) is used to determine the correct BSON type for each design property.
func applyWidgetAppearance(widget backend.Widget, w *ast.WidgetV3, theme *ThemeRegistry) {
	class, style := w.GetClass(), w.GetStyle()
	if class != "" || style != "" {
		type appearanceSetter interface {
			SetAppearance(class, style string)
		}
		if setter, ok := widget.(appearanceSetter); ok {
			setter.SetAppearance(class, style)
		}
	}

	// Apply design properties
	astProps := w.GetDesignProperties()
	if len(astProps) > 0 {
		var dpValues []backend.DesignPropertyValue
		for _, p := range astProps {
			switch strings.ToLower(p.Value) {
			case "on":
				dpValues = append(dpValues, backend.DesignPropertyValue{
					Key:       p.Key,
					ValueType: "toggle",
				})
			case "off":
				// OFF means toggle absence - skip
			default:
				dpValues = append(dpValues, backend.DesignPropertyValue{
					Key:       p.Key,
					ValueType: "option",
					Option:    p.Value,
				})
			}
		}
		if len(dpValues) > 0 {
			type designPropSetter interface {
				SetDesignProperties(props []backend.DesignPropertyValue)
			}
			if setter, ok := widget.(designPropSetter); ok {
				setter.SetDesignProperties(dpValues)
			}
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
	// No theme info available — default to "option" (Dropdown)
	return "option"
}

// =============================================================================
// V3 DataSource and Action Builders
// =============================================================================

// buildDataSourceV3 converts a V3 DataSource AST to a backend.DataSource.
// Returns the datasource, the entity name for context, and any error.
func (pb *pageBuilder) buildDataSourceV3(ds *ast.DataSourceV3) (backend.DataSource, string, error) {
	switch ds.Type {
	case "parameter":
		// Parameter reference: $ParamName
		// Page parameters store names WITHOUT $ prefix (e.g., "Customer")
		// Snippet parameters store names WITH $ prefix (e.g., "$Customer")
		// Try both variants for compatibility
		paramName := strings.TrimPrefix(ds.Reference, "$")
		entityID, ok := pb.paramScope[paramName]
		entityName := pb.paramEntityNames[paramName]
		if !ok {
			// Try with $ prefix (for snippets)
			entityID, ok = pb.paramScope["$"+paramName]
			entityName = pb.paramEntityNames["$"+paramName]
		}
		if !ok {
			return nil, "", mdlerrors.NewNotFound("parameter", ds.Reference)
		}

		// Fallback to lookup if entity name not stored
		if entityName == "" {
			var err error
			entityName, err = pb.getEntityNameByID(entityID)
			if err != nil {
				log.Printf("warning: could not resolve entity name for ID %s: %v", entityID, err)
			}
		}

		// Use DataViewSource with IsSnippetParameter flag
		return &backend.DataViewSource{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$DataViewSource",
			},
			EntityID:           entityID,
			EntityName:         entityName,
			ParameterName:      paramName,
			IsSnippetParameter: pb.isSnippet,
		}, entityName, nil

	case "database":
		// Database source: DATABASE Entity
		entityID, err := pb.resolveEntity(ast.QualifiedName{
			Module: pb.extractModule(ds.Reference),
			Name:   pb.extractName(ds.Reference),
		})
		if err != nil {
			return nil, "", mdlerrors.NewBackend("resolve entity", err)
		}

		dbSource := &backend.DatabaseSource{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$DatabaseSource", // Note: actual BSON $Type depends on widget context (grid/listview/dataview)
			},
			EntityID:   entityID,
			EntityName: ds.Reference,
		}

		// Handle WHERE clause
		if ds.Where != "" {
			dbSource.XPathConstraint = ds.Where
		}

		// Handle ORDER BY
		for _, ob := range ds.OrderBy {
			direction := backend.SortDirectionAscending
			if strings.ToLower(ob.Direction) == "desc" {
				direction = backend.SortDirectionDescending
			}
			sortItem := &backend.GridSort{
				BaseElement: model.BaseElement{
					ID:       model.ID(types.GenerateID()),
					TypeName: "Forms$GridSort",
				},
				AttributePath: pb.resolveAttributePathForEntity(ob.Attribute, ds.Reference),
				Direction:     direction,
			}
			dbSource.Sorting = append(dbSource.Sorting, sortItem)
		}

		return dbSource, ds.Reference, nil

	case "microflow":
		// Microflow source
		mfID, err := pb.resolveMicroflow(ds.Reference)
		if err != nil {
			return nil, "", mdlerrors.NewBackend("resolve microflow", err)
		}

		// Get entity name from microflow's return type for context resolution
		entityName := pb.getMicroflowReturnEntityName(ds.Reference)

		return &backend.MicroflowSource{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$MicroflowSource",
			},
			MicroflowID: mfID,
			Microflow:   ds.Reference,
		}, entityName, nil

	case "nanoflow":
		// Nanoflow source - resolve by listing all nanoflows
		nfID, err := pb.resolveNanoflowByName(ds.Reference)
		if err != nil {
			return nil, "", mdlerrors.NewBackend("resolve nanoflow", err)
		}

		// Get entity name from nanoflow's return type for context resolution
		entityName := pb.getNanoflowReturnEntityName(ds.Reference)

		return &backend.NanoflowSource{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$NanoflowSource",
			},
			NanoflowID: nfID,
			Nanoflow:   ds.Reference,
		}, entityName, nil

	case "association":
		// Association path source — emits Forms$AssociationSource BSON.
		// ds.Reference is either "Module.Assoc" (single-segment) or
		// "Module.Assoc/Module.DestEntity" (multi-segment, dest explicit).
		// For single-segment, resolve DestinationEntity from the association
		// definition (the side opposite to the parent context entity).
		ctxVar := ds.ContextVariable
		if ctxVar == "currentObject" {
			ctxVar = "" // implicit context — no SourceVariable in BSON
		}

		path := ds.Reference
		destEntity := ""
		if idx := strings.Index(path, "/"); idx >= 0 {
			destEntity = path[idx+1:]
			path = path[:idx]
		} else {
			destEntity = pb.resolveAssociationDestination(path, pb.entityContext)
		}

		// Return destEntity as the child context so column bindings inside the
		// widget can resolve short attribute names against it.
		return &backend.AssociationSource{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$AssociationSource",
			},
			EntityPath:      path + "/" + destEntity,
			ContextVariable: ctxVar,
		}, destEntity, nil

	case "selection":
		// Selection from another widget
		widgetName := ds.Reference
		widgetID, ok := pb.widgetScope[widgetName]
		if !ok {
			return nil, "", mdlerrors.NewNotFound("widget", widgetName)
		}

		// Get the entity context from the source widget if available
		entityName := pb.paramEntityNames[widgetName]

		return &backend.ListenToWidgetSource{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$ListenTargetSource",
			},
			WidgetID:   widgetID,
			WidgetName: widgetName, // Widget name for BSON serialization
		}, entityName, nil

	default:
		return nil, "", mdlerrors.NewUnsupported("unsupported datasource type: " + ds.Type)
	}
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
		// Module-scope the search: only look at the domain model whose module name
		// matches the first segment of the qualified association name. Association
		// names are not unique across the project (e.g., both AssocGrid and ODataSvc
		// can have an "OrderLine_Order" association) — without this check, we'd
		// pick the wrong one.
		if pb.moduleNameByID(pair.ContainerID) != modName {
			continue
		}
		for _, a := range pair.DM.AssociationsItems() {
			assoc, ok := a.(*genDm.Association)
			if !ok || assoc.Name() != assocName {
				continue
			}
			// Resolve entity qualified names for ParentID and ChildID.
			parentEntity := pb.entityQNByID(model.ID(assoc.ParentRefID()))
			childEntity := pb.entityQNByID(model.ID(assoc.ChildRefID()))
			// The "destination" is the end OPPOSITE to the context.
			if contextEntity != "" {
				if contextEntity == childEntity {
					return parentEntity
				}
				if contextEntity == parentEntity {
					return childEntity
				}
			}
			// No context or mismatch — default to the child (TO) side, which
			// matches the common FROM=context pattern.
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
				// Look up module name via the domain model's container
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
// Returns empty string if the microflow doesn't return an entity or list of entities.
func (pb *pageBuilder) getMicroflowReturnEntityName(qualifiedName string) string {
	// First, check if the microflow was created during this session (not yet in backend cache)
	if pb.execCache != nil && pb.execCache.createdMicroflows != nil {
		if info, ok := pb.execCache.createdMicroflows[qualifiedName]; ok {
			return info.ReturnEntityName
		}
	}

	// Parse qualified name
	parts := strings.Split(qualifiedName, ".")
	if len(parts) < 2 {
		return ""
	}
	moduleName := parts[0]
	mfName := strings.Join(parts[1:], ".")

	// Get microflows from backend
	mfs, err := pb.getMicroflows()
	if err != nil {
		return ""
	}

	// Use hierarchy to resolve module names (handles microflows in folders)
	h, err := pb.getHierarchy()
	if err != nil {
		return ""
	}

	// Find matching microflow. Stage 3.2.6.5: gen Microflow exposes
	// container via repo.GetContainerUUID + return type via the
	// ReturnType() string accessor (qualified name when entity / list,
	// "Void" / primitive otherwise).
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
// Empty / void / primitive return types yield "" since they carry no
// entity identity.
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
// Returns empty string if the nanoflow doesn't return an entity or list of entities.
func (pb *pageBuilder) getNanoflowReturnEntityName(qualifiedName string) string {
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
		// Nanoflows live in the same Unit table as microflows, so the
		// microflow repo's container helper resolves them too.
		containerID, _ := pb.microflowsRepo.GetContainerUUID(model.ID(nf.ID()))
		modName := h.GetModuleName(h.FindModuleID(containerID))
		if modName == moduleName && nf.Name() == name {
			return extractEntityFromGenReturnType(nf.ReturnType())
		}
	}

	return ""
}

// buildClientActionV3 converts a V3 Action AST to a backend.ClientAction.
func (pb *pageBuilder) buildClientActionV3(action *ast.ActionV3) (backend.ClientAction, error) {
	switch action.Type {
	case "save":
		return &backend.SaveChangesClientAction{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$SaveChangesClientAction",
			},
			ClosePage: action.ClosePage,
		}, nil

	case "cancel":
		return &backend.CancelChangesClientAction{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$CancelChangesClientAction",
			},
			ClosePage: action.ClosePage,
		}, nil

	case "close":
		return &backend.ClosePageClientAction{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$ClosePageClientAction",
			},
		}, nil

	case "delete":
		return &backend.DeleteClientAction{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$DeleteClientAction",
			},
		}, nil

	case "create":
		entityID, err := pb.resolveEntity(ast.QualifiedName{
			Module: pb.extractModule(action.Target),
			Name:   pb.extractName(action.Target),
		})
		if err != nil {
			return nil, mdlerrors.NewBackend("resolve entity for create", err)
		}

		createAct := &backend.CreateObjectClientAction{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$CreateObjectClientAction",
			},
			EntityID:   entityID,
			EntityName: action.Target,
		}

		// Handle THEN action (show page)
		if action.ThenAction != nil && action.ThenAction.Type == "showPage" {
			pageID, err := pb.resolvePageRef(action.ThenAction.Target)
			if err != nil {
				return nil, mdlerrors.NewBackend("resolve page", err)
			}
			createAct.PageID = pageID
			createAct.PageName = action.ThenAction.Target
		}

		return createAct, nil

	case "showPage":
		_, err := pb.resolvePageRef(action.Target)
		if err != nil {
			return nil, mdlerrors.NewBackend("resolve page", err)
		}

		pageAction := &backend.PageClientAction{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$PageClientAction",
			},
			PageName: action.Target,
		}

		// Build parameter mappings from Args
		for _, arg := range action.Args {
			mapping := &backend.PageClientParameterMapping{
				BaseElement: model.BaseElement{
					ID:       model.ID(types.GenerateID()),
					TypeName: "Forms$PageParameterMapping",
				},
				ParameterName: arg.Name,
			}

			// Determine if value is a variable reference or expression
			if strVal, ok := arg.Value.(string); ok {
				if strings.HasPrefix(strVal, "$") {
					// Variable reference (including $currentObject)
					mapping.Variable = strVal
				} else {
					mapping.Expression = strVal
				}
			}

			pageAction.ParameterMappings = append(pageAction.ParameterMappings, mapping)
		}

		return pageAction, nil

	case "microflow":
		mfID, err := pb.resolveMicroflow(action.Target)
		if err != nil {
			return nil, mdlerrors.NewBackend("resolve microflow", err)
		}

		mfAction := &backend.MicroflowClientAction{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$MicroflowAction",
			},
			MicroflowID:   mfID,
			MicroflowName: action.Target,
		}

		// Build parameter mappings from Args
		for _, arg := range action.Args {
			mapping := &backend.MicroflowParameterMapping{
				BaseElement: model.BaseElement{
					ID:       model.ID(types.GenerateID()),
					TypeName: "Forms$MicroflowParameterMapping",
				},
				ParameterName: arg.Name,
			}

			// Determine if value is a variable reference or expression
			if strVal, ok := arg.Value.(string); ok {
				if strings.HasPrefix(strVal, "$") {
					// Variable reference (including $currentObject)
					mapping.Variable = strVal
				} else {
					mapping.Expression = strVal
				}
			}

			mfAction.ParameterMappings = append(mfAction.ParameterMappings, mapping)
		}

		return mfAction, nil

	case "nanoflow":
		nfID, err := pb.resolveNanoflowByName(action.Target)
		if err != nil {
			return nil, mdlerrors.NewBackend("resolve nanoflow", err)
		}

		nfAction := &backend.NanoflowClientAction{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$NanoflowAction",
			},
			NanoflowID:   nfID,
			NanoflowName: action.Target,
		}

		// Build parameter mappings from Args
		for _, arg := range action.Args {
			mapping := &backend.NanoflowParameterMapping{
				BaseElement: model.BaseElement{
					ID:       model.ID(types.GenerateID()),
					TypeName: "Forms$NanoflowParameterMapping",
				},
				ParameterName: arg.Name,
			}

			// Determine if value is a variable reference or expression
			if strVal, ok := arg.Value.(string); ok {
				if strings.HasPrefix(strVal, "$") {
					// Variable reference (including $currentObject)
					mapping.Variable = strVal
				} else {
					mapping.Expression = strVal
				}
			}

			nfAction.ParameterMappings = append(nfAction.ParameterMappings, mapping)
		}

		return nfAction, nil

	case "openLink":
		return &backend.LinkClientAction{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$LinkClientAction",
			},
			LinkType: backend.LinkTypeWeb,
			Address:  action.LinkURL,
		}, nil

	case "signOut":
		return &backend.SignOutClientAction{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$SignOutClientAction",
			},
		}, nil

	case "completeTask":
		return &backend.SetTaskOutcomeClientAction{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$SetTaskOutcomeClientAction",
			},
			ClosePage:    true,
			Commit:       true,
			OutcomeValue: action.OutcomeValue,
		}, nil

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
		// Could be an entity type - use ObjectType
		return "DataTypes$ObjectType"
	}
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
	// Save and restore entity context
	oldContext := pb.entityContext
	pb.entityContext = entityName
	defer func() { pb.entityContext = oldContext }()

	return pb.resolveAttributePath(attrName)
}

// resolveTemplateAttributePath resolves template parameter values like $widgetName.Attribute
// to fully qualified entity paths like Module.Entity.Attribute.
// It handles patterns like:
// - $widgetName.Attribute -> looks up widget's entity and returns Entity.Attribute
// - simple Attribute -> uses current entity context
// - Module.Entity.Attribute -> returns as-is
func (pb *pageBuilder) resolveTemplateAttributePath(attrRef string) string {
	if attrRef == "" {
		return ""
	}

	// Check for $widgetName.Attribute pattern
	if after, ok := strings.CutPrefix(attrRef, "$"); ok {
		// Parse $widgetName.Attribute
		withoutDollar := after
		parts := strings.SplitN(withoutDollar, ".", 2)
		if len(parts) == 2 {
			widgetName := parts[0]
			attrName := parts[1]

			// Look up the widget's entity context from paramEntityNames
			// The widget name should match a parameter or widget scope entry
			if entityName, ok := pb.paramEntityNames[widgetName]; ok {
				return entityName + "." + attrName
			}
			// Try with $ prefix (for snippet parameters)
			if entityName, ok := pb.paramEntityNames["$"+widgetName]; ok {
				return entityName + "." + attrName
			}
			// Use current entity context as fallback
			if pb.entityContext != "" {
				return pb.entityContext + "." + attrName
			}
			// Return as-is if we can't resolve
			return attrRef
		}
	}

	// For other patterns, use regular attribute path resolution
	return pb.resolveAttributePath(attrRef)
}

// resolveTemplateAttributePathFull resolves a template parameter reference and sets
// both AttributeRef and SourceVariable on the parameter. This preserves the page
// parameter context so that DESCRIBE can output $Product.Name instead of Entity.Name.
//
// When attrRef is $paramName.Attribute (where paramName is a page/snippet parameter),
// it sets SourceVariable to paramName and AttributeRef to the resolved entity path.
//
// For non-String attributes (Integer, Decimal, DateTime, Boolean, etc.), the binding
// is automatically converted to a toString() expression since DYNAMICTEXT template
// parameters require String values.
func (pb *pageBuilder) resolveTemplateAttributePathFull(attrRef string, param *backend.ClientTemplateParameter) {
	if attrRef == "" {
		return
	}

	// Check for $paramName.Attribute pattern where paramName is a page parameter
	if after, ok := strings.CutPrefix(attrRef, "$"); ok {
		withoutDollar := after
		parts := strings.SplitN(withoutDollar, ".", 2)
		if len(parts) == 2 {
			paramName := parts[0]
			attrName := parts[1]

			// Check if this is a page/snippet parameter (not a widget reference)
			if entityName, ok := pb.paramEntityNames[paramName]; ok {
				fullPath := entityName + "." + attrName
				if pb.isNonStringAttribute(fullPath) {
					param.Expression = "toString($" + paramName + "/" + attrName + ")"
					return
				}
				param.SourceVariable = paramName
				param.AttributeRef = fullPath
				return
			}
			// Try with $ prefix (for snippet parameters)
			if entityName, ok := pb.paramEntityNames["$"+paramName]; ok {
				fullPath := entityName + "." + attrName
				if pb.isNonStringAttribute(fullPath) {
					param.Expression = "toString($" + paramName + "/" + attrName + ")"
					return
				}
				param.SourceVariable = paramName
				param.AttributeRef = fullPath
				return
			}
		}
	}

	// For other patterns, resolve and check type
	resolved := pb.resolveTemplateAttributePath(attrRef)
	if !strings.HasPrefix(attrRef, "$") && pb.isNonStringAttribute(resolved) {
		// Convert bare attribute names to toString() for non-String types.
		// Only for bare names (e.g., "TotalOrders") in DataView context,
		// not for $param.Attr references which are resolved via SourceVariable.
		param.Expression = "toString($currentObject/" + attrRef + ")"
		return
	}
	param.AttributeRef = resolved
}

// isNonStringAttribute checks if an attribute path refers to a non-String type.
// Returns false if the type can't be determined (fail-open to preserve existing behavior).
func (pb *pageBuilder) isNonStringAttribute(attrPath string) bool {
	attrType := pb.findAttributeType(attrPath)
	if attrType == nil {
		return false // can't determine type, assume String
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
		clone.Properties[k] = v // Property values are immutable (strings, ints, etc.)
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

func (pb *pageBuilder) buildLayoutGridV3(w *ast.WidgetV3) (*backend.LayoutGrid, error) {
	lg := &backend.LayoutGrid{
		BaseWidget: backend.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$LayoutGrid",
			},
			Name: w.Name,
		},
	}

	// Build rows from children
	for _, child := range w.Children {
		if strings.ToLower(child.Type) == "row" {
			row, err := pb.buildLayoutGridRowV3(child)
			if err != nil {
				return nil, err
			}
			lg.Rows = append(lg.Rows, row)
		}
	}

	return lg, nil
}

func (pb *pageBuilder) buildLayoutGridRowV3(w *ast.WidgetV3) (*backend.LayoutGridRow, error) {
	row := &backend.LayoutGridRow{
		BaseElement: model.BaseElement{
			ID:       model.ID(types.GenerateID()),
			TypeName: "Forms$LayoutGridRow",
		},
	}

	// Build columns from children
	for _, child := range w.Children {
		if strings.ToLower(child.Type) == "column" {
			col, err := pb.buildLayoutGridColumnV3(child)
			if err != nil {
				return nil, err
			}
			row.Columns = append(row.Columns, col)
		}
	}

	return row, nil
}

func (pb *pageBuilder) buildLayoutGridColumnV3(w *ast.WidgetV3) (*backend.LayoutGridColumn, error) {
	col := &backend.LayoutGridColumn{
		BaseElement: model.BaseElement{
			ID:       model.ID(types.GenerateID()),
			TypeName: "Forms$LayoutGridColumn",
		},
		Weight: 0, // 0 → columnWeight() maps to -1 (auto-fill) in the serializer
	}

	// Handle DesktopWidth
	if dw := w.GetDesktopWidth(); dw != nil {
		switch v := dw.(type) {
		case int:
			col.Weight = v
		case string:
			if strings.ToUpper(v) == "autofill" {
				col.Weight = -1 // Auto
			}
		}
	}

	// Handle TabletWidth
	if tw := w.Properties["TabletWidth"]; tw != nil {
		switch v := tw.(type) {
		case int:
			col.TabletWeight = v
		case string:
			if strings.ToUpper(v) == "autofill" {
				col.TabletWeight = -1
			}
		}
	}

	// Handle PhoneWidth
	if pw := w.Properties["PhoneWidth"]; pw != nil {
		switch v := pw.(type) {
		case int:
			col.PhoneWeight = v
		case string:
			if strings.ToUpper(v) == "autofill" {
				col.PhoneWeight = -1
			}
		}
	}

	// Build child widgets
	for _, child := range w.Children {
		widget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		col.Widgets = append(col.Widgets, widget)
	}

	return col, nil
}

// buildContainerWithRowV3 creates a Container holding a LayoutGrid with one row.
func (pb *pageBuilder) buildContainerWithRowV3(w *ast.WidgetV3) (*backend.Container, error) {
	container := &backend.Container{
		BaseWidget: backend.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$DivContainer",
			},
			Name: w.Name,
		},
	}

	lg := &backend.LayoutGrid{
		BaseWidget: backend.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$LayoutGrid",
			},
			Name: w.Name + "_grid",
		},
	}

	row, err := pb.buildLayoutGridRowV3(w)
	if err != nil {
		return nil, err
	}
	lg.Rows = append(lg.Rows, row)
	container.Widgets = append(container.Widgets, lg)

	return container, nil
}

// buildContainerWithColumnV3 creates a Container holding a LayoutGrid with one column.
func (pb *pageBuilder) buildContainerWithColumnV3(w *ast.WidgetV3) (*backend.Container, error) {
	container := &backend.Container{
		BaseWidget: backend.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$DivContainer",
			},
			Name: w.Name,
		},
	}

	lg := &backend.LayoutGrid{
		BaseWidget: backend.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$LayoutGrid",
			},
			Name: w.Name + "_grid",
		},
	}

	row := &backend.LayoutGridRow{
		BaseElement: model.BaseElement{
			ID:       model.ID(types.GenerateID()),
			TypeName: "Forms$LayoutGridRow",
		},
	}

	col, err := pb.buildLayoutGridColumnV3(w)
	if err != nil {
		return nil, err
	}
	row.Columns = append(row.Columns, col)
	lg.Rows = append(lg.Rows, row)
	container.Widgets = append(container.Widgets, lg)

	return container, nil
}

func (pb *pageBuilder) buildContainerV3(w *ast.WidgetV3) (*backend.Container, error) {
	container := &backend.Container{
		BaseWidget: backend.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$DivContainer",
			},
			Name: w.Name,
		},
	}

	// Handle RenderMode
	if rm := w.GetRenderMode(); rm != "" {
		container.RenderMode = backend.ContainerRenderMode(rm)
	}

	// Build child widgets
	for _, child := range w.Children {
		widget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		container.Widgets = append(container.Widgets, widget)
	}

	return container, nil
}

func (pb *pageBuilder) buildTabContainerV3(w *ast.WidgetV3) (*backend.TabContainer, error) {
	tc := &backend.TabContainer{
		BaseWidget: backend.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$TabControl",
			},
			Name: w.Name,
		},
	}

	// Build tab pages from children
	for _, child := range w.Children {
		if strings.ToLower(child.Type) == "tabpage" {
			tp, err := pb.buildTabPageV3(child)
			if err != nil {
				return nil, err
			}
			tc.TabPages = append(tc.TabPages, tp)
		}
	}

	if err := pb.registerWidgetName(w.Name, tc.ID); err != nil {
		return nil, err
	}

	return tc, nil
}

func (pb *pageBuilder) buildTabPageV3(w *ast.WidgetV3) (*backend.TabPage, error) {
	tp := &backend.TabPage{
		BaseElement: model.BaseElement{
			ID:       model.ID(types.GenerateID()),
			TypeName: "Forms$TabPage",
		},
		Name: w.Name,
	}

	// Handle Caption
	if caption := w.GetCaption(); caption != "" {
		tp.Caption = &model.Text{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Texts$Text",
			},
			Translations: map[string]string{"en_US": caption},
		}
	}

	// Build child widgets
	for _, child := range w.Children {
		widget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		tp.Widgets = append(tp.Widgets, widget)
	}

	if err := pb.registerWidgetName(w.Name, tp.ID); err != nil {
		return nil, err
	}

	return tp, nil
}

func (pb *pageBuilder) buildGroupBoxV3(w *ast.WidgetV3) (*backend.GroupBox, error) {
	gb := &backend.GroupBox{
		BaseWidget: backend.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$GroupBox",
			},
			Name: w.Name,
		},
		Collapsible: "No",
		HeaderMode:  "Div",
	}

	// Handle Caption — uses ClientTemplate (same as DynamicText Content)
	if caption := w.GetCaption(); caption != "" {
		gb.Caption = &backend.ClientTemplate{
			Template: &model.Text{
				BaseElement: model.BaseElement{
					ID:       model.ID(types.GenerateID()),
					TypeName: "Texts$Text",
				},
				Translations: map[string]string{"en_US": caption},
			},
		}
	}

	// Handle Collapsible: Yes/YesExpanded/YesCollapsed/No
	if collapsible := w.GetStringProp("Collapsible"); collapsible != "" {
		switch strings.ToLower(collapsible) {
		case "yesexpanded", "yesinitiallyexpanded", "yes":
			gb.Collapsible = "YesInitiallyExpanded"
		case "yescollapsed", "yesinitiallycollapsed":
			gb.Collapsible = "YesInitiallyCollapsed"
		case "no":
			gb.Collapsible = "No"
		default:
			gb.Collapsible = collapsible
		}
	}

	// Handle HeaderMode: Div, H1-H6
	if headerMode := w.GetStringProp("HeaderMode"); headerMode != "" {
		gb.HeaderMode = headerMode
	}

	// Build child widgets
	for _, child := range w.Children {
		widget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		gb.Widgets = append(gb.Widgets, widget)
	}

	if err := pb.registerWidgetName(w.Name, gb.ID); err != nil {
		return nil, err
	}

	return gb, nil
}

// buildFooterV3 creates a Footer container widget from V3 syntax.
func (pb *pageBuilder) buildFooterV3(w *ast.WidgetV3) (*backend.Container, error) {
	footer := &backend.Container{
		BaseWidget: backend.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$DivContainer",
			},
			Name: w.Name,
		},
	}

	// Build children
	for _, child := range w.Children {
		childWidget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		footer.Widgets = append(footer.Widgets, childWidget)
	}

	if err := pb.registerWidgetName(w.Name, footer.ID); err != nil {
		return nil, err
	}

	return footer, nil
}

// buildHeaderV3 creates a Header container widget from V3 syntax.
func (pb *pageBuilder) buildHeaderV3(w *ast.WidgetV3) (*backend.Container, error) {
	header := &backend.Container{
		BaseWidget: backend.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$DivContainer",
			},
			Name: w.Name,
		},
	}

	// Build children
	for _, child := range w.Children {
		childWidget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		header.Widgets = append(header.Widgets, childWidget)
	}

	if err := pb.registerWidgetName(w.Name, header.ID); err != nil {
		return nil, err
	}

	return header, nil
}

// buildControlBarV3 creates a ControlBar container widget from V3 syntax.
func (pb *pageBuilder) buildControlBarV3(w *ast.WidgetV3) (*backend.Container, error) {
	controlBar := &backend.Container{
		BaseWidget: backend.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$DivContainer",
			},
			Name: w.Name,
		},
	}

	// Build children
	for _, child := range w.Children {
		childWidget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		controlBar.Widgets = append(controlBar.Widgets, childWidget)
	}

	if err := pb.registerWidgetName(w.Name, controlBar.ID); err != nil {
		return nil, err
	}

	return controlBar, nil
}
func (pb *pageBuilder) buildDataViewV3(w *ast.WidgetV3) (*backend.DataView, error) {
	dv := &backend.DataView{
		BaseWidget: backend.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$DataView",
			},
			Name: w.Name,
		},
	}

	// Handle DataSource
	if ds := w.GetDataSource(); ds != nil {
		dataSource, entityName, err := pb.buildDataSourceV3(ds)
		if err != nil {
			return nil, mdlerrors.NewBackend("build datasource", err)
		}
		dv.DataSource = dataSource

		// Save and restore entity context so nested DataViews work correctly
		oldContext := pb.entityContext
		pb.entityContext = entityName
		defer func() { pb.entityContext = oldContext }()

		// Register the widget name with its entity so template params like $dvOrder.Attr
		// can be resolved to Entity.Attr
		if w.Name != "" && entityName != "" {
			pb.paramEntityNames[w.Name] = entityName
		}
	}

	// Build child widgets, separating FOOTER widgets into FooterWidgets
	for _, child := range w.Children {
		// Check if this is a FOOTER widget - its children go to FooterWidgets
		if child.Type == "footer" {
			dv.ShowFooter = true
			for _, fw := range child.Children {
				widget, err := pb.buildWidgetV3(fw)
				if err != nil {
					return nil, err
				}
				dv.FooterWidgets = append(dv.FooterWidgets, widget)
			}
			continue
		}
		childWidget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		dv.Widgets = append(dv.Widgets, childWidget)
	}

	// Also build footer widgets from Properties (legacy support)
	if footerWidgets, ok := w.Properties["Footer"].([]*ast.WidgetV3); ok {
		dv.ShowFooter = true
		for _, fw := range footerWidgets {
			widget, err := pb.buildWidgetV3(fw)
			if err != nil {
				return nil, err
			}
			dv.FooterWidgets = append(dv.FooterWidgets, widget)
		}
	}

	if err := pb.registerWidgetName(w.Name, dv.ID); err != nil {
		return nil, err
	}

	return dv, nil
}

func (pb *pageBuilder) buildDataGridV3(w *ast.WidgetV3) (*backend.CustomWidget, error) {
	widgetID := model.ID(types.GenerateID())

	// Build datasource from V3 DataSource property
	var datasource backend.DataSource
	if ds := w.GetDataSource(); ds != nil {
		dataSource, entityName, err := pb.buildDataSourceV3(ds)
		if err != nil {
			return nil, mdlerrors.NewBackend("build datasource", err)
		}
		datasource = dataSource

		// Save and restore entity context so nested containers work correctly
		oldContext := pb.entityContext
		pb.entityContext = entityName
		defer func() { pb.entityContext = oldContext }()
	}

	// Extract column definitions and CONTROLBAR widgets from children
	var columns []backend.DataGridColumnSpec
	var headerWidgets []backend.Widget
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
			// Build child widgets; filter-type children go to the column filter slot
			for _, grandchild := range child.Children {
				if filterWidgetID := dataGridFilterWidgetID(grandchild.Type); filterWidgetID != "" {
					fw, err := pb.widgetBackend.BuildFilterWidget(backend.FilterWidgetSpec{
						WidgetID:   filterWidgetID,
						FilterName: grandchild.Name,
					}, pb.backend.Path())
					if err != nil {
						return nil, mdlerrors.NewBackend("build column filter widget", err)
					}
					col.FilterWidget = fw
				} else {
					childWidget, err := pb.buildWidgetV3(grandchild)
					if err != nil {
						return nil, mdlerrors.NewBackend("build column child widget", err)
					}
					if childWidget != nil {
						col.ChildWidgets = append(col.ChildWidgets, childWidget)
					}
				}
			}
			columns = append(columns, col)
		case "controlbar":
			for _, controlBarChild := range child.Children {
				childWidget, err := pb.buildWidgetV3(controlBarChild)
				if err != nil {
					return nil, mdlerrors.NewBackend("build controlbar widget", err)
				}
				if childWidget != nil {
					headerWidgets = append(headerWidgets, childWidget)
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
		DataSource:      datasource,
		Columns:         columns,
		HeaderWidgets:   headerWidgets,
		PagingOverrides: pagingOverrides,
		SelectionMode:   w.GetSelection(),
	}

	// Stage 3.3.5.D2 NOTE: BuildDataGrid2WidgetGen exists for future Cat-B use.
	// Using legacy path here for same reason as widget_engine.go Build — see that comment.
	grid, err := pb.widgetBackend.BuildDataGrid2Widget(widgetID, w.Name, spec, pb.backend.Path())
	if err != nil {
		return nil, err
	}

	if err := pb.registerWidgetName(w.Name, grid.ID); err != nil {
		return nil, err
	}

	return grid, nil
}

func (pb *pageBuilder) buildDataGridColumnV3(w *ast.WidgetV3) (*backend.DataGridColumn, error) {
	col := &backend.DataGridColumn{
		BaseElement: model.BaseElement{
			ID:       model.ID(types.GenerateID()),
			TypeName: "Forms$DataGridColumn",
		},
		Name:     w.Name,
		Editable: true,
	}

	// Get attribute from Attribute property
	if attr := w.GetAttribute(); attr != "" {
		col.AttributePath = pb.resolveAttributePath(attr)
	}

	// Get caption
	if caption := w.GetCaption(); caption != "" {
		col.Caption = &model.Text{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Texts$Text",
			},
			Translations: map[string]string{"en_US": caption},
		}
	}

	return col, nil
}

func (pb *pageBuilder) buildListViewV3(w *ast.WidgetV3) (*backend.ListView, error) {
	lv := &backend.ListView{
		BaseWidget: backend.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$ListView",
			},
			Name: w.Name,
		},
		PageSize: 20,
	}

	// Handle DataSource
	if ds := w.GetDataSource(); ds != nil {
		dataSource, entityName, err := pb.buildDataSourceV3(ds)
		if err != nil {
			return nil, mdlerrors.NewBackend("build datasource", err)
		}
		lv.DataSource = dataSource

		// Save and restore entity context so nested containers work correctly
		oldContext := pb.entityContext
		pb.entityContext = entityName
		defer func() { pb.entityContext = oldContext }()

		// Register widget name with entity for SELECTION datasource lookup
		if w.Name != "" && entityName != "" {
			pb.paramEntityNames[w.Name] = entityName
		}
	}

	// Register widget scope for SELECTION references
	if err := pb.registerWidgetName(w.Name, lv.ID); err != nil {
		return nil, err
	}

	// Build template widgets
	for _, child := range w.Children {
		widget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		lv.Widgets = append(lv.Widgets, widget)
	}

	return lv, nil
}

func (pb *pageBuilder) buildTextBoxV3(w *ast.WidgetV3) (*backend.TextBox, error) {
	tb := &backend.TextBox{
		BaseWidget: backend.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$TextBox",
			},
			Name: w.Name,
		},
	}

	// Handle Attribute (attribute path)
	if attr := w.GetAttribute(); attr != "" {
		tb.AttributePath = pb.resolveAttributePath(attr)
	}

	// Handle Label
	if label := w.GetLabel(); label != "" {
		tb.Label = label
	}

	if err := pb.registerWidgetName(w.Name, tb.ID); err != nil {
		return nil, err
	}

	return tb, nil
}

func (pb *pageBuilder) buildTextAreaV3(w *ast.WidgetV3) (*backend.TextArea, error) {
	ta := &backend.TextArea{
		BaseWidget: backend.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$TextArea",
			},
			Name: w.Name,
		},
	}

	// Handle Attribute
	if attr := w.GetAttribute(); attr != "" {
		ta.AttributePath = pb.resolveAttributePath(attr)
	}

	// Handle Label
	if label := w.GetLabel(); label != "" {
		ta.Label = label
	}

	if err := pb.registerWidgetName(w.Name, ta.ID); err != nil {
		return nil, err
	}

	return ta, nil
}

func (pb *pageBuilder) buildDatePickerV3(w *ast.WidgetV3) (*backend.DatePicker, error) {
	dp := &backend.DatePicker{
		BaseWidget: backend.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$DatePicker",
			},
			Name: w.Name,
		},
	}

	// Handle Attribute
	if attr := w.GetAttribute(); attr != "" {
		dp.AttributePath = pb.resolveAttributePath(attr)
	}

	// Handle Label
	if label := w.GetLabel(); label != "" {
		dp.Label = label
	}

	if err := pb.registerWidgetName(w.Name, dp.ID); err != nil {
		return nil, err
	}

	return dp, nil
}

func (pb *pageBuilder) buildDropdownV3(w *ast.WidgetV3) (*backend.DropDown, error) {
	dd := &backend.DropDown{
		BaseWidget: backend.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$DropDown",
			},
			Name: w.Name,
		},
	}

	// Handle Attribute
	if attr := w.GetAttribute(); attr != "" {
		dd.AttributePath = pb.resolveAttributePath(attr)
	}

	// Handle Label
	if label := w.GetLabel(); label != "" {
		dd.Label = label
	}

	if err := pb.registerWidgetName(w.Name, dd.ID); err != nil {
		return nil, err
	}

	return dd, nil
}

func (pb *pageBuilder) buildCheckBoxV3(w *ast.WidgetV3) (*backend.CheckBox, error) {
	cb := &backend.CheckBox{
		BaseWidget: backend.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$CheckBox",
			},
			Name: w.Name,
		},
	}

	// Handle Attribute
	if attr := w.GetAttribute(); attr != "" {
		cb.AttributePath = pb.resolveAttributePath(attr)
	}

	// Handle Label
	if label := w.GetLabel(); label != "" {
		cb.Label = label
	}

	if err := pb.registerWidgetName(w.Name, cb.ID); err != nil {
		return nil, err
	}

	return cb, nil
}

// buildRadioButtonsV3 creates RadioButtons from V3 syntax.
func (pb *pageBuilder) buildRadioButtonsV3(w *ast.WidgetV3) (*backend.RadioButtons, error) {
	rb := &backend.RadioButtons{
		BaseWidget: backend.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$RadioButtonGroup",
			},
			Name: w.Name,
		},
		Label: w.GetLabel(),
	}

	// Get attribute path from Attribute property
	if attr := w.GetAttribute(); attr != "" {
		rb.AttributePath = pb.resolveAttributePath(attr)
	}

	if err := pb.registerWidgetName(w.Name, rb.ID); err != nil {
		return nil, err
	}

	return rb, nil
}

func (pb *pageBuilder) buildTextWidgetV3(w *ast.WidgetV3) (*backend.Text, error) {
	st := &backend.Text{
		BaseWidget: backend.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$Text",
			},
			Name: w.Name,
		},
		RenderMode: backend.TextRenderModeText,
	}

	// Handle Content
	if content := w.GetContent(); content != "" {
		st.Caption = &model.Text{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Texts$Text",
			},
			Translations: map[string]string{"en_US": content},
		}
	}

	// Handle RenderMode
	if rm := w.GetRenderMode(); rm != "" {
		st.RenderMode = backend.TextRenderMode(rm)
	}

	if err := pb.registerWidgetName(w.Name, st.ID); err != nil {
		return nil, err
	}

	return st, nil
}

func (pb *pageBuilder) buildDynamicTextV3(w *ast.WidgetV3) (*backend.DynamicText, error) {
	dt := &backend.DynamicText{
		BaseWidget: backend.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$DynamicText",
			},
			Name: w.Name,
		},
		RenderMode: backend.TextRenderModeText,
	}

	// Handle RenderMode
	if rm := w.GetRenderMode(); rm != "" {
		dt.RenderMode = backend.TextRenderMode(rm)
	}

	// Handle Content
	content := w.GetContent()
	explicitParams := w.GetContentParams()

	// Check if Content is an attribute reference AND no explicit params provided
	// If so, auto-generate template {1} and add the attribute as a parameter
	// Examples:
	//   Content: $widget.Name            -> auto-generate {1} with $widget.Name as param
	//   Content: Entity.Attribute        -> auto-generate {1} with Entity.Attribute as param
	//   Content: SomeStaticText          -> literal string, no params (no dot, no $)
	//   Content: 'Name: {1}', ContentParams: [Name] -> use explicit template and params
	var autoGeneratedParams []string
	if content != "" && explicitParams == nil {
		// Only auto-generate for:
		// - Variable references: $var or $widget.Attr (starts with $)
		// - Entity paths: Entity.Attribute (identifier.identifier pattern, not version numbers like "1.0")
		// Simple identifiers without dots are treated as static text
		isEntityPath := false
		if strings.Contains(content, ".") && !strings.HasPrefix(content, "$") {
			// Check if it looks like Entity.Attribute (letter followed by word chars, dot, letter followed by word chars)
			// This avoids matching strings like "Version 1.0" or "Dashboard - V2.1"
			isEntityPath = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*\.[A-Za-z_][A-Za-z0-9_]*$`).MatchString(content)
		}
		if strings.HasPrefix(content, "$") || isEntityPath {
			autoGeneratedParams = append(autoGeneratedParams, content)
			content = "{1}"
		}
	}

	if content == "" {
		content = "{1}"
	}

	dt.Content = &backend.ClientTemplate{
		BaseElement: model.BaseElement{
			ID:       model.ID(types.GenerateID()),
			TypeName: "Forms$ClientTemplate",
		},
		Template: &model.Text{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Texts$Text",
			},
			Translations: map[string]string{"en_US": content},
		},
	}

	// Add auto-generated parameters first
	for _, attrRef := range autoGeneratedParams {
		param := &backend.ClientTemplateParameter{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$ClientTemplateParameter",
			},
		}
		pb.resolveTemplateAttributePathFull(attrRef, param)
		dt.Content.Parameters = append(dt.Content.Parameters, param)
	}

	// Handle explicit ContentParams
	if explicitParams != nil {
		for _, p := range explicitParams {
			param := &backend.ClientTemplateParameter{
				BaseElement: model.BaseElement{
					ID:       model.ID(types.GenerateID()),
					TypeName: "Forms$ClientTemplateParameter",
				},
			}
			// Check if it's an attribute reference or literal
			if strVal, ok := p.Value.(string); ok {
				if strings.HasPrefix(strVal, "'") || strings.HasPrefix(strVal, "\"") {
					// Already a quoted string literal - use as-is
					param.Expression = strVal
				} else if strings.HasPrefix(strVal, "$") || strings.Contains(strVal, ".") {
					// Attribute reference - resolve widget references to entity paths
					pb.resolveTemplateAttributePathFull(strVal, param)
				} else {
					// Unquoted literal value - assume attribute in current context
					pb.resolveTemplateAttributePathFull(strVal, param)
				}
			}
			dt.Content.Parameters = append(dt.Content.Parameters, param)
		}
	}

	if err := pb.registerWidgetName(w.Name, dt.ID); err != nil {
		return nil, err
	}

	return dt, nil
}

func (pb *pageBuilder) buildTitleV3(w *ast.WidgetV3) (*backend.Title, error) {
	title := &backend.Title{
		BaseWidget: backend.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$Title",
			},
			Name: w.Name,
		},
	}

	// Set caption from Content property
	content := w.GetContent()
	if content != "" {
		title.Caption = &model.Text{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Texts$Text",
			},
			Translations: map[string]string{"en_US": content},
		}
	}

	if err := pb.registerWidgetName(w.Name, title.ID); err != nil {
		return nil, err
	}

	return title, nil
}

func (pb *pageBuilder) buildButtonV3(w *ast.WidgetV3) (*backend.ActionButton, error) {
	btn := &backend.ActionButton{
		BaseWidget: backend.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$ActionButton",
			},
			Name: w.Name,
		},
		ButtonStyle: backend.ButtonStyleDefault,
	}

	// Handle Caption
	if caption := w.GetCaption(); caption != "" {
		btn.CaptionTemplate = &backend.ClientTemplate{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$ClientTemplate",
			},
			Template: &model.Text{
				BaseElement: model.BaseElement{
					ID:       model.ID(types.GenerateID()),
					TypeName: "Texts$Text",
				},
				Translations: map[string]string{"en_US": caption},
			},
		}

		// Handle CaptionParams (template parameters like {1}, {2})
		if params := w.GetCaptionParams(); params != nil {
			for _, p := range params {
				param := &backend.ClientTemplateParameter{
					BaseElement: model.BaseElement{
						ID:       model.ID(types.GenerateID()),
						TypeName: "Forms$ClientTemplateParameter",
					},
				}
				// Check if it's an attribute reference or literal
				if strVal, ok := p.Value.(string); ok {
					if strings.HasPrefix(strVal, "'") || strings.HasPrefix(strVal, "\"") {
						// Already a quoted string literal - use as-is
						param.Expression = strVal
					} else if strings.HasPrefix(strVal, "$") || strings.Contains(strVal, ".") {
						// Attribute reference - resolve widget references to entity paths
						param.AttributeRef = pb.resolveTemplateAttributePath(strVal)
					} else {
						// Unquoted literal value - wrap in quotes for expression
						param.Expression = "'" + strVal + "'"
					}
				}
				btn.CaptionTemplate.Parameters = append(btn.CaptionTemplate.Parameters, param)
			}
		}
	}

	// Handle ButtonStyle
	if style := w.GetButtonStyle(); style != "" {
		btn.ButtonStyle = backend.ButtonStyle(style)
	}

	// Handle Action
	if action := w.GetAction(); action != nil {
		act, err := pb.buildClientActionV3(action)
		if err != nil {
			return nil, mdlerrors.NewBackend("build action", err)
		}
		btn.Action = act
	}

	if err := pb.registerWidgetName(w.Name, btn.ID); err != nil {
		return nil, err
	}

	return btn, nil
}

// buildNavigationListV3 creates a NavigationList widget from V3 syntax.
func (pb *pageBuilder) buildNavigationListV3(w *ast.WidgetV3) (*backend.NavigationList, error) {
	navList := &backend.NavigationList{
		BaseWidget: backend.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$NavigationList",
			},
			Name: w.Name,
		},
	}

	// Build items from children (ITEM widgets)
	for _, child := range w.Children {
		if strings.ToLower(child.Type) == "item" {
			item, err := pb.buildNavigationListItemV3(child)
			if err != nil {
				return nil, err
			}
			navList.Items = append(navList.Items, item)
		}
	}

	if err := pb.registerWidgetName(w.Name, navList.ID); err != nil {
		return nil, err
	}

	return navList, nil
}

// buildNavigationListItemV3 creates a NavigationListItem from V3 syntax.
func (pb *pageBuilder) buildNavigationListItemV3(w *ast.WidgetV3) (*backend.NavigationListItem, error) {
	if w.Name == "" {
		return nil, mdlerrors.NewValidation("item inside navigationlist requires a name")
	}

	item := &backend.NavigationListItem{
		BaseElement: model.BaseElement{
			ID:       model.ID(types.GenerateID()),
			TypeName: "Forms$NavigationListItem",
		},
		Name: w.Name,
	}

	if err := pb.registerWidgetName(w.Name, item.ID); err != nil {
		return nil, err
	}

	// Set caption from Caption property
	if caption := w.GetCaption(); caption != "" {
		item.Caption = &model.Text{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Texts$Text",
			},
			Translations: map[string]string{"en_US": caption},
		}
	}

	// Handle Action property
	if action := w.GetAction(); action != nil {
		clientAction, err := pb.buildClientActionV3(action)
		if err != nil {
			return nil, err
		}
		item.Action = clientAction
	}

	// Build child widgets
	for _, child := range w.Children {
		childWidget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		item.Widgets = append(item.Widgets, childWidget)
	}

	return item, nil
}

// buildSnippetCallV3 creates a SnippetCallWidget from V3 syntax.
func (pb *pageBuilder) buildSnippetCallV3(w *ast.WidgetV3) (*backend.SnippetCallWidget, error) {
	sc := &backend.SnippetCallWidget{
		BaseWidget: backend.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$SnippetCallWidget",
			},
			Name: w.Name,
		},
	}

	// Handle Snippet property - resolve snippet and store both ID and name
	snippetName := w.GetSnippet()
	if snippetName != "" {
		snippetID, err := pb.resolveSnippetRef(snippetName)
		if err != nil {
			return nil, mdlerrors.NewBackend(fmt.Sprintf("resolve snippet %s", snippetName), err)
		}
		sc.SnippetID = snippetID
		sc.SnippetName = snippetName // Store qualified name for BY_NAME_REFERENCE serialization

		// Validate and wire up parameter mappings.
		if err := pb.buildSnippetCallParams(sc, snippetName, w.GetSnippetParams()); err != nil {
			return nil, err
		}
	}

	if err := pb.registerWidgetName(w.Name, sc.ID); err != nil {
		return nil, err
	}

	return sc, nil
}

// buildSnippetCallParams validates the supplied param mappings against the
// snippet's declared parameters and populates sc.ParameterMappings.
func (pb *pageBuilder) buildSnippetCallParams(sc *backend.SnippetCallWidget, snippetQName string, supplied []ast.SnippetCallParam) error {
	snippets, err := pb.backend.ListSnippetsGen()
	if err != nil {
		return err
	}

	// Find the target snippet to read its declared parameters.
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
		// Snippet has no declared parameters — nothing to validate or map.
		return nil
	}

	// Build a lookup of supplied mappings by parameter name (strip leading $).
	suppliedByName := make(map[string]string, len(supplied))
	for _, p := range supplied {
		name := strings.TrimPrefix(p.ParamName, "$")
		suppliedByName[name] = p.Variable
	}

	// Validate that every declared parameter has a mapping, then build the list.
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
		sc.ParameterMappings = append(sc.ParameterMappings, backend.SnippetParamMapping{
			ParamName: paramName,
			Argument:  argument,
		})
	}

	return nil
}

// buildTemplateV3 creates a Container to hold template content.
func (pb *pageBuilder) buildTemplateV3(w *ast.WidgetV3) (*backend.Container, error) {
	container := &backend.Container{
		BaseWidget: backend.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$DivContainer",
			},
			Name: w.Name,
		},
	}

	// Build children
	for _, child := range w.Children {
		childWidget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		container.Widgets = append(container.Widgets, childWidget)
	}

	return container, nil
}

// buildFilterV3 creates a Container to hold filter widgets.
func (pb *pageBuilder) buildFilterV3(w *ast.WidgetV3) (*backend.Container, error) {
	container := &backend.Container{
		BaseWidget: backend.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$DivContainer",
			},
			Name: w.Name,
		},
	}

	// Build children (filter widgets)
	for _, child := range w.Children {
		childWidget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		container.Widgets = append(container.Widgets, childWidget)
	}

	return container, nil
}

func (pb *pageBuilder) buildStaticImageV3(w *ast.WidgetV3) (*backend.StaticImage, error) {
	img := &backend.StaticImage{
		BaseWidget: backend.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$StaticImageViewer",
			},
			Name: w.Name,
		},
		Responsive: true,
	}

	if width := w.GetIntProp("Width"); width > 0 {
		img.Width = width
	}
	if height := w.GetIntProp("Height"); height > 0 {
		img.Height = height
	}

	if err := pb.registerWidgetName(w.Name, img.ID); err != nil {
		return nil, err
	}

	return img, nil
}

func (pb *pageBuilder) buildDynamicImageV3(w *ast.WidgetV3) (*backend.DynamicImage, error) {
	img := &backend.DynamicImage{
		BaseWidget: backend.BaseWidget{
			BaseElement: model.BaseElement{
				ID:       model.ID(types.GenerateID()),
				TypeName: "Forms$ImageViewer",
			},
			Name: w.Name,
		},
		Responsive: true,
	}

	if width := w.GetIntProp("Width"); width > 0 {
		img.Width = width
	}
	if height := w.GetIntProp("Height"); height > 0 {
		img.Height = height
	}

	if err := pb.registerWidgetName(w.Name, img.ID); err != nil {
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

// dataGridPagingPropMap maps PascalCase MDL property names to camelCase widget property keys.
var dataGridPagingPropMap = map[string]string{
	"PageSize":          "pageSize",
	"Pagination":        "pagination",
	"PagingPosition":    "pagingPosition",
	"ShowPagingButtons": "showPagingButtons",
	// "ShowNumberOfRows" is defined in DataGrid2 type but not yet fully supported;
	// setting it to a non-default value causes CE0463 "widget definition changed".
}
