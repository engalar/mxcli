// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/microflows"
	"github.com/mendixlabs/mxcli/sdk/mpr"
	"github.com/mendixlabs/mxcli/sdk/pages"
)

// ============================================================================
// V3 Page Builder
// ============================================================================

// buildPageV3 creates a Page from a CreatePageStmtV3.
func (pb *pageBuilder) buildPageV3(s *ast.CreatePageStmtV3) (*pages.Page, error) {
	// Resolve folder if specified
	containerID := pb.moduleID
	if s.Folder != "" {
		folderID, err := pb.resolveFolder(s.Folder)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve folder %s: %w", s.Folder, err)
		}
		containerID = folderID
	}

	page := &pages.Page{
		BaseElement: model.BaseElement{
			ID:       model.ID(mpr.GenerateID()),
			TypeName: "Forms$Page",
		},
		ContainerID:   containerID,
		Name:          s.Name.Name,
		Documentation: s.Documentation,
		URL:           s.URL,
		MarkAsUsed:    false,
		Excluded:      false,
	}

	// Set title
	if s.Title != "" {
		page.Title = &model.Text{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
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
			fmt.Printf("Warning: layout %s not found\n", s.Layout)
		} else {
			page.LayoutID = layoutID

			// Create LayoutCall with arguments for placeholders
			page.LayoutCall = &pages.LayoutCall{
				BaseElement: model.BaseElement{
					ID:       model.ID(mpr.GenerateID()),
					TypeName: "Forms$LayoutCall",
				},
				LayoutID:   layoutID,
				LayoutName: s.Layout, // Qualified name for "Form" field in BSON
			}
		}
	}

	// Build parameters
	for _, param := range s.Parameters {
		pageParam := &pages.PageParameter{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "Forms$PageParameter",
			},
			ContainerID: page.ID,
			Name:        param.Name,
			IsRequired:  true, // Page parameters are required by default
		}

		// Resolve entity type
		if param.EntityType.Name != "" {
			entityID, err := pb.resolveEntity(param.EntityType)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve entity %s: %w", param.EntityType.String(), err)
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
		localVar := &pages.LocalVariable{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
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

		arg := &pages.LayoutCallArgument{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "Forms$FormCallArgument",
			},
			ParameterID: model.ID(mainPlaceholderRef),
		}

		// Build V3 widgets (expanding fragments)
		if len(s.Widgets) > 0 {
			containerWidget := &pages.Container{
				BaseWidget: pages.BaseWidget{
					BaseElement: model.BaseElement{
						ID:       model.ID(mpr.GenerateID()),
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
					return nil, fmt.Errorf("failed to build widget: %w", err)
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
func (pb *pageBuilder) buildSnippetV3(s *ast.CreateSnippetStmtV3) (*pages.Snippet, error) {
	// Resolve folder if specified
	containerID := pb.moduleID
	if s.Folder != "" {
		folderID, err := pb.resolveFolder(s.Folder)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve folder %s: %w", s.Folder, err)
		}
		containerID = folderID
	}

	snippet := &pages.Snippet{
		BaseElement: model.BaseElement{
			ID:       model.ID(mpr.GenerateID()),
			TypeName: "Forms$Snippet",
		},
		ContainerID:   containerID,
		Name:          s.Name.Name,
		Documentation: s.Documentation,
	}

	// Build parameters
	for _, param := range s.Parameters {
		snippetParam := &pages.SnippetParameter{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "Forms$SnippetParameter",
			},
			ContainerID: snippet.ID,
			Name:        param.Name,
		}

		// Resolve entity type
		if param.EntityType.Name != "" {
			entityID, err := pb.resolveEntity(param.EntityType)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve entity %s: %w", param.EntityType.String(), err)
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
		localVar := &pages.LocalVariable{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
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
			return nil, fmt.Errorf("failed to build widget: %w", err)
		}
		snippet.Widgets = append(snippet.Widgets, w)
	}

	return snippet, nil
}

// buildWidgetV3 converts a V3 AST widget to a pages.Widget.
func (pb *pageBuilder) buildWidgetV3(w *ast.WidgetV3) (pages.Widget, error) {
	var widget pages.Widget
	var err error

	switch strings.ToUpper(w.Type) {
	case "DATAVIEW":
		widget, err = pb.buildDataViewV3(w)
	case "DATAGRID":
		widget, err = pb.buildDataGridV3(w)
	case "LISTVIEW":
		widget, err = pb.buildListViewV3(w)
	case "LAYOUTGRID":
		widget, err = pb.buildLayoutGridV3(w)
	case "ROW":
		// ROW creates a container with LayoutGrid that contains one row
		widget, err = pb.buildContainerWithRowV3(w)
	case "COLUMN":
		// COLUMN creates a container with LayoutGrid that contains one column
		widget, err = pb.buildContainerWithColumnV3(w)
	case "CONTAINER", "CUSTOMCONTAINER":
		widget, err = pb.buildContainerV3(w)
	case "TEXTBOX":
		widget, err = pb.buildTextBoxV3(w)
	case "TEXTAREA":
		widget, err = pb.buildTextAreaV3(w)
	case "DATEPICKER":
		widget, err = pb.buildDatePickerV3(w)
	case "DROPDOWN":
		widget, err = pb.buildDropdownV3(w)
	case "CHECKBOX":
		widget, err = pb.buildCheckBoxV3(w)
	case "TEXT", "STATICTEXT":
		widget, err = pb.buildTextWidgetV3(w)
	case "DYNAMICTEXT":
		widget, err = pb.buildDynamicTextV3(w)
	case "TITLE":
		widget, err = pb.buildTitleV3(w)
	case "BUTTON", "ACTIONBUTTON":
		widget, err = pb.buildButtonV3(w)
	case "TABCONTAINER":
		widget, err = pb.buildTabContainerV3(w)
	case "TABPAGE":
		// Tab pages are handled inside TabContainer
		return nil, fmt.Errorf("TABPAGE must be a direct child of TABCONTAINER")
	case "GROUPBOX":
		widget, err = pb.buildGroupBoxV3(w)
	case "COMBOBOX":
		widget, err = pb.buildComboBoxV3(w)
	case "RADIOBUTTONS":
		widget, err = pb.buildRadioButtonsV3(w)
	case "GALLERY":
		widget, err = pb.buildGalleryV3(w)
	case "NAVIGATIONLIST":
		widget, err = pb.buildNavigationListV3(w)
	case "ITEM":
		// Items are handled inside NavigationList
		return nil, fmt.Errorf("ITEM must be a direct child of NAVIGATIONLIST")
	case "SNIPPETCALL":
		widget, err = pb.buildSnippetCallV3(w)
	case "FOOTER":
		widget, err = pb.buildFooterV3(w)
	case "HEADER":
		widget, err = pb.buildHeaderV3(w)
	case "CONTROLBAR":
		widget, err = pb.buildControlBarV3(w)
	case "TEMPLATE":
		widget, err = pb.buildTemplateV3(w)
	case "FILTER":
		widget, err = pb.buildFilterV3(w)
	case "TEXTFILTER":
		widget, err = pb.buildTextFilterV3(w)
	case "NUMBERFILTER":
		widget, err = pb.buildNumberFilterV3(w)
	case "DROPDOWNFILTER":
		widget, err = pb.buildDropdownFilterV3(w)
	case "DATEFILTER":
		widget, err = pb.buildDateFilterV3(w)
	case "IMAGE", "STATICIMAGE":
		widget, err = pb.buildStaticImageV3(w)
	case "DYNAMICIMAGE":
		widget, err = pb.buildDynamicImageV3(w)
	default:
		return nil, fmt.Errorf("unsupported V3 widget type: %s", w.Type)
	}

	if err != nil {
		return nil, err
	}

	// Apply Class/Style appearance properties to the widget
	applyWidgetAppearance(widget, w)

	return widget, nil
}

// applyWidgetAppearance sets Class, Style, and DesignProperties on a widget if specified in the AST.
func applyWidgetAppearance(widget pages.Widget, w *ast.WidgetV3) {
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
		var dpValues []pages.DesignPropertyValue
		for _, p := range astProps {
			switch strings.ToUpper(p.Value) {
			case "ON":
				dpValues = append(dpValues, pages.DesignPropertyValue{
					Key:       p.Key,
					ValueType: "toggle",
				})
			case "OFF":
				// OFF means toggle absence - skip
			default:
				dpValues = append(dpValues, pages.DesignPropertyValue{
					Key:       p.Key,
					ValueType: "option",
					Option:    p.Value,
				})
			}
		}
		if len(dpValues) > 0 {
			type designPropSetter interface {
				SetDesignProperties(props []pages.DesignPropertyValue)
			}
			if setter, ok := widget.(designPropSetter); ok {
				setter.SetDesignProperties(dpValues)
			}
		}
	}
}

// =============================================================================
// V3 DataSource and Action Builders
// =============================================================================

// buildDataSourceV3 converts a V3 DataSource AST to a pages.DataSource.
// Returns the datasource, the entity name for context, and any error.
func (pb *pageBuilder) buildDataSourceV3(ds *ast.DataSourceV3) (pages.DataSource, string, error) {
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
			return nil, "", fmt.Errorf("parameter not found: %s", ds.Reference)
		}

		// Fallback to lookup if entity name not stored
		if entityName == "" {
			entityName, _ = pb.getEntityNameByID(entityID)
		}

		// Use DataViewSource with IsSnippetParameter flag
		return &pages.DataViewSource{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
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
			return nil, "", fmt.Errorf("failed to resolve entity: %w", err)
		}

		dbSource := &pages.DatabaseSource{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
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
			direction := pages.SortDirectionAscending
			if strings.ToUpper(ob.Direction) == "DESC" {
				direction = pages.SortDirectionDescending
			}
			sortItem := &pages.GridSort{
				BaseElement: model.BaseElement{
					ID:       model.ID(mpr.GenerateID()),
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
			return nil, "", fmt.Errorf("failed to resolve microflow: %w", err)
		}

		// Get entity name from microflow's return type for context resolution
		entityName := pb.getMicroflowReturnEntityName(ds.Reference)

		return &pages.MicroflowSource{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "Forms$MicroflowSource",
			},
			MicroflowID: mfID,
			Microflow:   ds.Reference,
		}, entityName, nil

	case "nanoflow":
		// Nanoflow source - resolve by listing all nanoflows
		nfID, err := pb.resolveNanoflowByName(ds.Reference)
		if err != nil {
			return nil, "", fmt.Errorf("failed to resolve nanoflow: %w", err)
		}

		// Get entity name from nanoflow's return type for context resolution
		entityName := pb.getNanoflowReturnEntityName(ds.Reference)

		return &pages.NanoflowSource{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "Forms$NanoflowSource",
			},
			NanoflowID: nfID,
			Nanoflow:   ds.Reference,
		}, entityName, nil

	case "association":
		// Association path source
		return &pages.AssociationSource{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "Forms$AssociationSource",
			},
			EntityPath: ds.Reference,
		}, "", nil

	case "selection":
		// Selection from another widget
		widgetName := ds.Reference
		widgetID, ok := pb.widgetScope[widgetName]
		if !ok {
			return nil, "", fmt.Errorf("widget not found for selection: %s", widgetName)
		}

		// Get the entity context from the source widget if available
		entityName := pb.paramEntityNames[widgetName]

		return &pages.ListenToWidgetSource{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "Forms$ListenTargetSource",
			},
			WidgetID:   widgetID,
			WidgetName: widgetName, // Widget name for BSON serialization
		}, entityName, nil

	default:
		return nil, "", fmt.Errorf("unsupported datasource type: %s", ds.Type)
	}
}

// getMicroflowReturnEntityName looks up a microflow and returns its return type entity name.
// Returns empty string if the microflow doesn't return an entity or list of entities.
func (pb *pageBuilder) getMicroflowReturnEntityName(qualifiedName string) string {
	// First, check if the microflow was created during this session (not yet in reader cache)
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

	// Get microflows from reader cache
	mfs, err := pb.getMicroflows()
	if err != nil {
		return ""
	}

	// Use hierarchy to resolve module names (handles microflows in folders)
	h, err := pb.getHierarchy()
	if err != nil {
		return ""
	}

	// Find matching microflow
	for _, mf := range mfs {
		modID := h.FindModuleID(mf.ContainerID)
		modName := h.GetModuleName(modID)
		if modName == moduleName && mf.Name == mfName {
			// Extract entity name from return type
			return extractEntityFromReturnType(mf.ReturnType)
		}
	}

	return ""
}

// extractEntityFromReturnType extracts the entity qualified name from a DataType.
func extractEntityFromReturnType(dt microflows.DataType) string {
	if dt == nil {
		return ""
	}

	switch t := dt.(type) {
	case *microflows.ObjectType:
		return t.EntityQualifiedName
	case *microflows.ListType:
		return t.EntityQualifiedName
	default:
		return ""
	}
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

	nanoflows, err := pb.reader.ListNanoflows()
	if err != nil {
		return ""
	}

	h, err := pb.getHierarchy()
	if err != nil {
		return ""
	}

	for _, nf := range nanoflows {
		modID := h.FindModuleID(nf.ContainerID)
		modName := ""
		for _, m := range pb.getModules() {
			if m.ID == modID {
				modName = m.Name
				break
			}
		}
		if modName == moduleName && nf.Name == name {
			return extractEntityFromReturnType(nf.ReturnType)
		}
	}

	return ""
}

// buildClientActionV3 converts a V3 Action AST to a pages.ClientAction.
func (pb *pageBuilder) buildClientActionV3(action *ast.ActionV3) (pages.ClientAction, error) {
	switch action.Type {
	case "save":
		return &pages.SaveChangesClientAction{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "Forms$SaveChangesClientAction",
			},
			ClosePage: action.ClosePage,
		}, nil

	case "cancel":
		return &pages.CancelChangesClientAction{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "Forms$CancelChangesClientAction",
			},
			ClosePage: action.ClosePage,
		}, nil

	case "close":
		return &pages.ClosePageClientAction{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "Forms$ClosePageClientAction",
			},
		}, nil

	case "delete":
		return &pages.DeleteClientAction{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "Forms$DeleteClientAction",
			},
		}, nil

	case "create":
		entityID, err := pb.resolveEntity(ast.QualifiedName{
			Module: pb.extractModule(action.Target),
			Name:   pb.extractName(action.Target),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to resolve entity for create: %w", err)
		}

		createAct := &pages.CreateObjectClientAction{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "Forms$CreateObjectClientAction",
			},
			EntityID:   entityID,
			EntityName: action.Target,
		}

		// Handle THEN action (show page)
		if action.ThenAction != nil && action.ThenAction.Type == "showPage" {
			pageID, err := pb.resolvePageRef(action.ThenAction.Target)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve page: %w", err)
			}
			createAct.PageID = pageID
			createAct.PageName = action.ThenAction.Target
		}

		return createAct, nil

	case "showPage":
		_, err := pb.resolvePageRef(action.Target)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve page: %w", err)
		}

		pageAction := &pages.PageClientAction{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "Forms$PageClientAction",
			},
			PageName: action.Target,
		}

		// Build parameter mappings from Args
		for _, arg := range action.Args {
			mapping := &pages.PageClientParameterMapping{
				BaseElement: model.BaseElement{
					ID:       model.ID(mpr.GenerateID()),
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
			return nil, fmt.Errorf("failed to resolve microflow: %w", err)
		}

		mfAction := &pages.MicroflowClientAction{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "Forms$MicroflowAction",
			},
			MicroflowID:   mfID,
			MicroflowName: action.Target,
		}

		// Build parameter mappings from Args
		for _, arg := range action.Args {
			mapping := &pages.MicroflowParameterMapping{
				BaseElement: model.BaseElement{
					ID:       model.ID(mpr.GenerateID()),
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
			return nil, fmt.Errorf("failed to resolve nanoflow: %w", err)
		}

		nfAction := &pages.NanoflowClientAction{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "Forms$NanoflowAction",
			},
			NanoflowID:   nfID,
			NanoflowName: action.Target,
		}

		// Build parameter mappings from Args
		for _, arg := range action.Args {
			mapping := &pages.NanoflowParameterMapping{
				BaseElement: model.BaseElement{
					ID:       model.ID(mpr.GenerateID()),
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
		return &pages.LinkClientAction{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "Forms$LinkClientAction",
			},
			LinkType: pages.LinkTypeWeb,
			Address:  action.LinkURL,
		}, nil

	case "signOut":
		return &pages.SignOutClientAction{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "Forms$SignOutClientAction",
			},
		}, nil

	default:
		return nil, fmt.Errorf("unsupported action type: %s", action.Type)
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
	domainModels, err := pb.getDomainModels()
	if err != nil {
		return "", err
	}

	modules := pb.getModules()
	moduleNames := make(map[model.ID]string)
	for _, m := range modules {
		moduleNames[m.ID] = m.Name
	}

	for _, dm := range domainModels {
		for _, e := range dm.Entities {
			if e.ID == entityID {
				moduleName := moduleNames[dm.ContainerID]
				return moduleName + "." + e.Name, nil
			}
		}
	}
	return "", fmt.Errorf("entity not found by ID: %s", entityID)
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

	nanoflows, err := pb.reader.ListNanoflows()
	if err != nil {
		return "", fmt.Errorf("failed to list nanoflows: %w", err)
	}

	h, err := pb.getHierarchy()
	if err != nil {
		return "", err
	}

	for _, nf := range nanoflows {
		modID := h.FindModuleID(nf.ContainerID)
		modName := ""
		for _, m := range pb.getModules() {
			if m.ID == modID {
				modName = m.Name
				break
			}
		}
		if modName == moduleName && nf.Name == name {
			return nf.ID, nil
		}
	}

	return "", fmt.Errorf("nanoflow not found: %s", nfName)
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
	case "datetime":
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
func (pb *pageBuilder) resolveTemplateAttributePathFull(attrRef string, param *pages.ClientTemplateParameter) {
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
				// This is a page parameter reference
				param.SourceVariable = paramName
				param.AttributeRef = entityName + "." + attrName
				return
			}
			// Try with $ prefix (for snippet parameters)
			if entityName, ok := pb.paramEntityNames["$"+paramName]; ok {
				param.SourceVariable = paramName
				param.AttributeRef = entityName + "." + attrName
				return
			}
		}
	}

	// For other patterns, just set AttributeRef
	param.AttributeRef = pb.resolveTemplateAttributePath(attrRef)
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
		return nil, fmt.Errorf("fragment %q not found", w.Name)
	}
	frag, ok := pb.fragments[w.Name]
	if !ok {
		return nil, fmt.Errorf("fragment %q not found", w.Name)
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
