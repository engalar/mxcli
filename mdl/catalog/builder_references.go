// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"database/sql"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDt "github.com/mendixlabs/mxcli/modelsdk/gen/datatypes"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genPages "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

// Reference kinds for the refs table
const (
	RefKindCall       = "call"       // Microflow calls microflow
	RefKindCreate     = "create"     // Microflow creates entity
	RefKindRetrieve   = "retrieve"   // Microflow retrieves entity
	RefKindShowPage   = "show_page"  // Microflow shows page
	RefKindGeneralize = "generalize" // Entity extends entity
	RefKindAssociate  = "associate"  // Association targets entity
	RefKindLayout     = "layout"     // Page uses layout
	RefKindDatasource = "datasource" // Page/widget uses entity via datasource
	RefKindParameter  = "parameter"  // Page parameter entity type
	RefKindAction     = "action"     // Widget calls microflow/nanoflow
	RefKindHomePage   = "home_page"  // Navigation home page reference
	RefKindLoginPage  = "login_page" // Navigation login page reference
	RefKindMenuItem   = "menu_item"  // Navigation menu item page reference
)

// collectActionActivities returns all ActionActivity objects from a
// gen ObjectCollection, recursing into LoopedActivity bodies to find
// nested actions. Activities with a nil inner Action are skipped.
func collectActionActivities(oc *genMf.MicroflowObjectCollection) []*genMf.ActionActivity {
	if oc == nil {
		return nil
	}
	var result []*genMf.ActionActivity
	for _, obj := range oc.ObjectsItems() {
		switch o := obj.(type) {
		case *genMf.ActionActivity:
			if o.Action() != nil {
				result = append(result, o)
			}
		case *genMf.LoopedActivity:
			result = append(result, collectActionActivities(flowObjectCollection(o.ObjectCollection()))...)
		}
	}
	return result
}

// buildReferences extracts cross-references from all documents.
// This is only run in full mode as it requires parsing all documents.
func (b *Builder) buildReferences() error {
	if !b.fullMode {
		return nil
	}

	stmt, err := b.tx.Prepare(`
		INSERT INTO refs (SourceType, SourceId, SourceName, TargetType, TargetId, TargetName, RefKind, ModuleName, ProjectId, SnapshotId)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	projectID := b.catalog.projectID
	snapshotID := b.snapshot.ID
	refCount := 0

	// Extract microflow references (using cached list — no re-parsing)
	mfs, err := b.cachedMicroflows()
	if err != nil {
		return err
	}

	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		moduleID := b.hierarchy.findModuleID(model.ID(mf.ID()))
		moduleName := b.hierarchy.getModuleName(moduleID)
		sourceQN := moduleName + "." + mf.Name()
		sourceType := "MICROFLOW"
		mfID := string(mf.ID())

		oc := flowObjectCollection(mf.ObjectCollection())
		if oc == nil {
			continue
		}

		for _, act := range collectActionActivities(oc) {
			switch a := act.Action().(type) {
			case *genMf.MicroflowCallAction:
				if call, ok := a.MicroflowCall().(*genMf.MicroflowCall); ok && call != nil {
					if qn := call.MicroflowQualifiedName(); qn != "" {
						_, err = stmt.Exec(sourceType, mfID, sourceQN,
							"MICROFLOW", "", qn,
							RefKindCall, moduleName, projectID, snapshotID)
						if err == nil {
							refCount++
						}
					}
				}

			case *genMf.CreateObjectAction:
				if qn := a.EntityQualifiedName(); qn != "" {
					_, err = stmt.Exec(sourceType, mfID, sourceQN,
						"ENTITY", "", qn,
						RefKindCreate, moduleName, projectID, snapshotID)
					if err == nil {
						refCount++
					}
				}

			case *genMf.RetrieveAction:
				if src := a.RetrieveSource(); src != nil {
					if dbSrc, ok := src.(*genMf.DatabaseRetrieveSource); ok {
						if qn := dbSrc.EntityQualifiedName(); qn != "" {
							_, err = stmt.Exec(sourceType, mfID, sourceQN,
								"ENTITY", "", qn,
								RefKindRetrieve, moduleName, projectID, snapshotID)
							if err == nil {
								refCount++
							}
						}
					}
				}

			case *genMf.ShowPageAction:
				if settings, ok := a.PageSettings().(*genPages.PageSettings); ok && settings != nil {
					if qn := settings.PageQualifiedName(); qn != "" {
						_, err = stmt.Exec(sourceType, mfID, sourceQN,
							"PAGE", "", qn,
							RefKindShowPage, moduleName, projectID, snapshotID)
						if err == nil {
							refCount++
						}
					}
				}

			case *genMf.JavaActionCallAction:
				if qn := a.JavaActionQualifiedName(); qn != "" {
					_, err = stmt.Exec(sourceType, mfID, sourceQN,
						"JAVA_ACTION", "", qn,
						RefKindCall, moduleName, projectID, snapshotID)
					if err == nil {
						refCount++
					}
				}
			}
		}
	}

	// Extract entity references (generalization) — using cached list
	dms, err := b.cachedDomainModels()
	if err == nil {
		for _, dm := range dms {
			moduleID := b.hierarchy.findModuleID(dm.ContainerID)
			moduleName := b.hierarchy.getModuleName(moduleID)

			for _, ent := range dm.Entities {
				sourceQN := moduleName + "." + ent.Name
				// Check generalization
				if ent.GeneralizationRef != "" {
					_, err = stmt.Exec("ENTITY", string(ent.ID), sourceQN,
						"ENTITY", "", ent.GeneralizationRef,
						RefKindGeneralize, moduleName, projectID, snapshotID)
					if err == nil {
						refCount++
					}
				}
			}

			// Note: Association references require resolving ChildID to entity name
			// which requires a lookup table. Skipping for now - can be added later.
		}
	}

	// Extract page references (layout, datasources, parameters) — using cached gen list
	pageGenList, err := b.cachedPagesGen()
	if err == nil {
		for _, pg := range pageGenList {
			if pg == nil {
				continue
			}
			pgID := model.ID(pg.ID())
			moduleID := b.hierarchy.findModuleID(pgID)
			moduleName := b.hierarchy.getModuleName(moduleID)
			sourceQN := moduleName + "." + pg.Name()

			// Layout reference
			if lcElem := pg.LayoutCall(); lcElem != nil {
				if lc, ok := lcElem.(*genPages.LayoutCall); ok && lc.LayoutQualifiedName() != "" {
					_, err = stmt.Exec("PAGE", string(pgID), sourceQN,
						"LAYOUT", "", lc.LayoutQualifiedName(),
						RefKindLayout, moduleName, projectID, snapshotID)
					if err == nil {
						refCount++
					}

					// Extract refs from widgets in layout arguments
					for _, argElem := range lc.ArgumentsItems() {
						arg, ok := argElem.(*genPages.LayoutCallArgument)
						if !ok {
							continue
						}
						if w := arg.Widget(); w != nil {
							refCount += b.extractWidgetRefs(stmt, w, "PAGE", string(pgID), sourceQN, moduleName, projectID, snapshotID)
						}
						for _, w := range arg.WidgetsItems() {
							refCount += b.extractWidgetRefs(stmt, w, "PAGE", string(pgID), sourceQN, moduleName, projectID, snapshotID)
						}
					}
				}
			}

			// Page parameter entity types
			for _, paramElem := range pg.ParametersItems() {
				param, ok := paramElem.(*genPages.PageParameter)
				if !ok {
					continue
				}
				entityName := pageParamEntityName(param)
				if entityName != "" {
					_, err = stmt.Exec("PAGE", string(pgID), sourceQN,
						"ENTITY", "", entityName,
						RefKindParameter, moduleName, projectID, snapshotID)
					if err == nil {
						refCount++
					}
				}
			}
		}
	}

	// Extract navigation references (home pages, menu items, login pages)
	nav, err := b.reader.GetNavigation()
	if err == nil {
		for _, profile := range nav.Profiles {
			sourceName := "Navigation." + profile.Name

			// Default home page
			if profile.HomePage != nil {
				if profile.HomePage.Page != "" {
					_, err = stmt.Exec("NAVIGATION", "", sourceName,
						"PAGE", "", profile.HomePage.Page,
						RefKindHomePage, "", projectID, snapshotID)
					if err == nil {
						refCount++
					}
				}
				if profile.HomePage.Microflow != "" {
					_, err = stmt.Exec("NAVIGATION", "", sourceName,
						"MICROFLOW", "", profile.HomePage.Microflow,
						RefKindHomePage, "", projectID, snapshotID)
					if err == nil {
						refCount++
					}
				}
			}

			// Role-based home pages
			for _, rh := range profile.RoleBasedHomePages {
				if rh.Page != "" {
					_, err = stmt.Exec("NAVIGATION", "", sourceName,
						"PAGE", "", rh.Page,
						RefKindHomePage, "", projectID, snapshotID)
					if err == nil {
						refCount++
					}
				}
				if rh.Microflow != "" {
					_, err = stmt.Exec("NAVIGATION", "", sourceName,
						"MICROFLOW", "", rh.Microflow,
						RefKindHomePage, "", projectID, snapshotID)
					if err == nil {
						refCount++
					}
				}
			}

			// Login page
			if profile.LoginPage != "" {
				_, err = stmt.Exec("NAVIGATION", "", sourceName,
					"PAGE", "", profile.LoginPage,
					RefKindLoginPage, "", projectID, snapshotID)
				if err == nil {
					refCount++
				}
			}

			// Menu items (recursive)
			refCount += b.extractMenuItemRefs(stmt, profile.MenuItems, sourceName, projectID, snapshotID)
		}
	}

	// Extract workflow references — using cached gen list
	wfs, wfErr := b.cachedWorkflows()
	if wfErr == nil {
		for _, wf := range wfs {
			if wf == nil {
				continue
			}
			moduleID := b.hierarchy.findModuleID(model.ID(wf.ID()))
			moduleName := b.hierarchy.getModuleName(moduleID)
			sourceQN := moduleName + "." + wf.Name()

			// Parameter entity reference
			if entity := workflowParamEntityGen(wf); entity != "" {
				_, err = stmt.Exec("WORKFLOW", string(wf.ID()), sourceQN,
					"ENTITY", "", entity,
					RefKindParameter, moduleName, projectID, snapshotID)
				if err == nil {
					refCount++
				}
			}

			// Overview page reference
			if op := wf.OverviewPageQualifiedName(); op != "" {
				_, err = stmt.Exec("WORKFLOW", string(wf.ID()), sourceQN,
					"PAGE", "", op,
					RefKindShowPage, moduleName, projectID, snapshotID)
				if err == nil {
					refCount++
				}
			}

			// Extract references from workflow activities
			if flow, ok := wf.Flow().(*genWf.Flow); ok && flow != nil {
				refCount += b.extractWorkflowFlowRefsGen(stmt, flow, string(wf.ID()), sourceQN, moduleName, projectID, snapshotID)
			}
		}
	}

	b.report("References", refCount)
	return nil
}

// extractMenuItemRefs extracts page and microflow references from menu items recursively.
func (b *Builder) extractMenuItemRefs(stmt *sql.Stmt, items []*types.NavMenuItem, sourceName, projectID, snapshotID string) int {
	refCount := 0
	for _, item := range items {
		if item.Page != "" {
			_, err := stmt.Exec("NAVIGATION", "", sourceName,
				"PAGE", "", item.Page,
				RefKindMenuItem, "", projectID, snapshotID)
			if err == nil {
				refCount++
			}
		}
		if item.Microflow != "" {
			_, err := stmt.Exec("NAVIGATION", "", sourceName,
				"MICROFLOW", "", item.Microflow,
				RefKindMenuItem, "", projectID, snapshotID)
			if err == nil {
				refCount++
			}
		}
		if len(item.Items) > 0 {
			refCount += b.extractMenuItemRefs(stmt, item.Items, sourceName, projectID, snapshotID)
		}
	}
	return refCount
}

// extractWidgetRefs extracts entity and microflow references from a gen widget element and its children.
func (b *Builder) extractWidgetRefs(stmt *sql.Stmt, w element.Element, sourceType, sourceID, sourceQN, moduleName, projectID, snapshotID string) int {
	if w == nil {
		return 0
	}

	refCount := 0

	switch widget := w.(type) {
	case *genPages.DataView:
		refCount += b.extractDataSourceRefs(stmt, widget.DataSource(), sourceType, sourceID, sourceQN, moduleName, projectID, snapshotID)
		for _, child := range widget.WidgetsItems() {
			refCount += b.extractWidgetRefs(stmt, child, sourceType, sourceID, sourceQN, moduleName, projectID, snapshotID)
		}
		for _, child := range widget.FooterWidgetsItems() {
			refCount += b.extractWidgetRefs(stmt, child, sourceType, sourceID, sourceQN, moduleName, projectID, snapshotID)
		}

	case *genPages.ListView:
		refCount += b.extractDataSourceRefs(stmt, widget.DataSource(), sourceType, sourceID, sourceQN, moduleName, projectID, snapshotID)
		for _, child := range widget.WidgetsItems() {
			refCount += b.extractWidgetRefs(stmt, child, sourceType, sourceID, sourceQN, moduleName, projectID, snapshotID)
		}

	case *genPages.DataGrid:
		refCount += b.extractDataSourceRefs(stmt, widget.DataSource(), sourceType, sourceID, sourceQN, moduleName, projectID, snapshotID)

	case *genPages.TemplateGrid:
		refCount += b.extractDataSourceRefs(stmt, widget.DataSource(), sourceType, sourceID, sourceQN, moduleName, projectID, snapshotID)
		// TemplateGrid.Contents() holds a TemplateGridContents which has WidgetsItems
		if contents, ok := widget.Contents().(*genPages.TemplateGridContents); ok && contents != nil {
			for _, child := range contents.WidgetsItems() {
				refCount += b.extractWidgetRefs(stmt, child, sourceType, sourceID, sourceQN, moduleName, projectID, snapshotID)
			}
		}

	case *genPages.DivContainer:
		for _, child := range widget.WidgetsItems() {
			refCount += b.extractWidgetRefs(stmt, child, sourceType, sourceID, sourceQN, moduleName, projectID, snapshotID)
		}

	case *genPages.GroupBox:
		for _, child := range widget.WidgetsItems() {
			refCount += b.extractWidgetRefs(stmt, child, sourceType, sourceID, sourceQN, moduleName, projectID, snapshotID)
		}

	case *genPages.LayoutGrid:
		for _, rowElem := range widget.RowsItems() {
			row, ok := rowElem.(*genPages.LayoutGridRow)
			if !ok {
				continue
			}
			for _, colElem := range row.ColumnsItems() {
				col, ok := colElem.(*genPages.LayoutGridColumn)
				if !ok {
					continue
				}
				for _, child := range col.WidgetsItems() {
					refCount += b.extractWidgetRefs(stmt, child, sourceType, sourceID, sourceQN, moduleName, projectID, snapshotID)
				}
			}
		}

	case *genPages.TabContainer:
		for _, tabElem := range widget.TabPagesItems() {
			tab, ok := tabElem.(*genPages.TabPage)
			if !ok {
				continue
			}
			for _, child := range tab.WidgetsItems() {
				refCount += b.extractWidgetRefs(stmt, child, sourceType, sourceID, sourceQN, moduleName, projectID, snapshotID)
			}
		}

	case *genPages.ScrollContainer:
		for _, regionElem := range []element.Element{widget.Center(), widget.Left(), widget.Right(), widget.Top(), widget.Bottom()} {
			if regionElem == nil {
				continue
			}
			region, ok := regionElem.(*genPages.ScrollContainerRegion)
			if !ok {
				continue
			}
			for _, child := range region.WidgetsItems() {
				refCount += b.extractWidgetRefs(stmt, child, sourceType, sourceID, sourceQN, moduleName, projectID, snapshotID)
			}
		}

	default:
		// gen/pages has no CustomWidget type (codegen gap) and no Gallery type.
		// For pluggable widgets, extract entity/microflow refs from raw BSON via
		// well-known property keys. This covers CustomWidgets$CustomWidget and
		// Forms$Gallery which lack narrow gen types.
		refCount += b.extractWidgetRefsFromRaw(stmt, w, sourceType, sourceID, sourceQN, moduleName, projectID, snapshotID)
	}

	return refCount
}

// extractWidgetRefsFromRaw extracts entity/microflow refs from a widget element that
// has no narrow gen type (CustomWidget, Gallery). It uses raw-BSON string extraction
// for well-known property keys: EntityRef, Microflow, Nanoflow, Form (page).
func (b *Builder) extractWidgetRefsFromRaw(stmt *sql.Stmt, w element.Element, sourceType, sourceID, sourceQN, moduleName, projectID, snapshotID string) int {
	if w == nil {
		return 0
	}
	refCount := 0
	raw := w.Raw()
	if raw == nil {
		return 0
	}
	if v, _ := codec.ReadBSONFieldString(raw, "EntityRef"); v != "" {
		stmt.Exec(sourceType, sourceID, sourceQN, "ENTITY", "", v, RefKindDatasource, moduleName, projectID, snapshotID)
		refCount++
	}
	if v, _ := codec.ReadBSONFieldString(raw, "Microflow"); v != "" {
		stmt.Exec(sourceType, sourceID, sourceQN, "MICROFLOW", "", v, RefKindAction, moduleName, projectID, snapshotID)
		refCount++
	}
	if v, _ := codec.ReadBSONFieldString(raw, "Nanoflow"); v != "" {
		stmt.Exec(sourceType, sourceID, sourceQN, "NANOFLOW", "", v, RefKindAction, moduleName, projectID, snapshotID)
		refCount++
	}
	if v, _ := codec.ReadBSONFieldString(raw, "Form"); v != "" {
		stmt.Exec(sourceType, sourceID, sourceQN, "PAGE", "", v, RefKindShowPage, moduleName, projectID, snapshotID)
		refCount++
	}
	return refCount
}

// extractDataSourceRefs extracts entity and microflow references from a gen datasource element.
func (b *Builder) extractDataSourceRefs(stmt *sql.Stmt, ds element.Element, sourceType, sourceID, sourceQN, moduleName, projectID, snapshotID string) int {
	if ds == nil {
		return 0
	}

	refCount := 0

	switch src := ds.(type) {
	case *genPages.DatabaseSourceBase:
		if ep := src.EntityPath(); ep != "" {
			parts := strings.Split(ep, "/")
			if len(parts) > 0 && parts[0] != "" {
				stmt.Exec(sourceType, sourceID, sourceQN,
					"ENTITY", "", ep,
					RefKindDatasource, moduleName, projectID, snapshotID)
				refCount++
			}
		}

	case *genPages.DataViewSource:
		if ep := src.EntityPath(); ep != "" {
			stmt.Exec(sourceType, sourceID, sourceQN,
				"ENTITY", "", ep,
				RefKindDatasource, moduleName, projectID, snapshotID)
			refCount++
		}

	case *genPages.EntityPathSource:
		if ep := src.EntityPath(); ep != "" {
			parts := strings.Split(ep, "/")
			if len(parts) > 0 && parts[0] != "" {
				stmt.Exec(sourceType, sourceID, sourceQN,
					"ENTITY", "", ep,
					RefKindDatasource, moduleName, projectID, snapshotID)
				refCount++
			}
		}

	case *genPages.AssociationSource:
		if ep := src.EntityPath(); ep != "" {
			stmt.Exec(sourceType, sourceID, sourceQN,
				"ENTITY", "", ep,
				RefKindDatasource, moduleName, projectID, snapshotID)
			refCount++
		}

	case *genPages.MicroflowSource:
		if ms, ok := src.MicroflowSettings().(*genPages.MicroflowSettings); ok && ms != nil {
			if qn := ms.MicroflowQualifiedName(); qn != "" {
				stmt.Exec(sourceType, sourceID, sourceQN,
					"MICROFLOW", "", qn,
					RefKindDatasource, moduleName, projectID, snapshotID)
				refCount++
			}
		}

	case *genPages.NanoflowSource:
		if qn := src.NanoflowQualifiedName(); qn != "" {
			stmt.Exec(sourceType, sourceID, sourceQN,
				"NANOFLOW", "", qn,
				RefKindDatasource, moduleName, projectID, snapshotID)
			refCount++
		}
	}

	return refCount
}

// pageParamEntityName extracts the entity qualified name from a gen PageParameter's
// ParameterType. Returns "" when the type is not an entity type.
func pageParamEntityName(param *genPages.PageParameter) string {
	if param == nil {
		return ""
	}
	pt := param.ParameterType()
	if pt == nil {
		return ""
	}
	switch t := pt.(type) {
	case *genDt.ObjectType:
		return t.EntityQualifiedName()
	case *genDt.EntityType:
		return t.EntityQualifiedName()
	}
	return ""
}

// resolveEntityID looks up the qualified name for an entity ID.
func (b *Builder) resolveEntityID(entityID model.ID) string {
	if entityID == "" {
		return ""
	}
	var qualifiedName string
	err := b.tx.QueryRow("SELECT QualifiedName FROM entities WHERE Id = ?", string(entityID)).Scan(&qualifiedName)
	if err != nil {
		return ""
	}
	return qualifiedName
}

// resolveMicroflowID looks up the qualified name for a microflow/nanoflow ID.
func (b *Builder) resolveMicroflowID(mfID model.ID) string {
	if mfID == "" {
		return ""
	}
	var qualifiedName string
	err := b.tx.QueryRow("SELECT QualifiedName FROM microflows WHERE Id = ?", string(mfID)).Scan(&qualifiedName)
	if err != nil {
		return ""
	}
	return qualifiedName
}

// extractWorkflowFlowRefsGen extracts references from a gen workflow
// flow and its nested sub-flows. Mirrors the legacy extractor; dispatch
// is by storage $Type because gen splits user-task and call-microflow
// into multiple subtypes.
func (b *Builder) extractWorkflowFlowRefsGen(stmt *sql.Stmt, flow *genWf.Flow, sourceID, sourceQN, moduleName, projectID, snapshotID string) int {
	if flow == nil {
		return 0
	}
	refCount := 0
	for _, act := range flow.ActivitiesItems() {
		if act == nil {
			continue
		}
		switch v := act.(type) {
		case *genWf.UserTask:
			if v.PageQualifiedName() != "" {
				if _, err := stmt.Exec("WORKFLOW", sourceID, sourceQN,
					"PAGE", "", v.PageQualifiedName(),
					RefKindShowPage, moduleName, projectID, snapshotID); err == nil {
					refCount++
				}
			}
			if ent := v.UserTaskEntityQualifiedName(); ent != "" {
				if _, err := stmt.Exec("WORKFLOW", sourceID, sourceQN,
					"ENTITY", "", ent,
					RefKindDatasource, moduleName, projectID, snapshotID); err == nil {
					refCount++
				}
			}
			refCount += extractUserSourceRefGen(stmt, v.UserSource(), sourceID, sourceQN, moduleName, projectID, snapshotID)
			for _, oc := range v.OutcomesItems() {
				if utc, ok := oc.(*genWf.UserTaskOutcome); ok {
					if f, ok := utc.Flow().(*genWf.Flow); ok {
						refCount += b.extractWorkflowFlowRefsGen(stmt, f, sourceID, sourceQN, moduleName, projectID, snapshotID)
					}
				}
			}
		case *genWf.SingleUserTaskActivity:
			refCount += extractUserSourceRefGen(stmt, v.UserSource(), sourceID, sourceQN, moduleName, projectID, snapshotID)
			for _, oc := range v.OutcomesItems() {
				if utc, ok := oc.(*genWf.UserTaskOutcome); ok {
					if f, ok := utc.Flow().(*genWf.Flow); ok {
						refCount += b.extractWorkflowFlowRefsGen(stmt, f, sourceID, sourceQN, moduleName, projectID, snapshotID)
					}
				}
			}
		case *genWf.MultiUserTaskActivity:
			refCount += extractUserSourceRefGen(stmt, v.UserSource(), sourceID, sourceQN, moduleName, projectID, snapshotID)
			for _, oc := range v.OutcomesItems() {
				if utc, ok := oc.(*genWf.UserTaskOutcome); ok {
					if f, ok := utc.Flow().(*genWf.Flow); ok {
						refCount += b.extractWorkflowFlowRefsGen(stmt, f, sourceID, sourceQN, moduleName, projectID, snapshotID)
					}
				}
			}
		case *genWf.CallMicroflowActivity:
			if v.MicroflowQualifiedName() != "" {
				if _, err := stmt.Exec("WORKFLOW", sourceID, sourceQN,
					"MICROFLOW", "", v.MicroflowQualifiedName(),
					RefKindCall, moduleName, projectID, snapshotID); err == nil {
					refCount++
				}
			}
			refCount += b.extractConditionOutcomeRefsGen(stmt, v.OutcomesItems(), sourceID, sourceQN, moduleName, projectID, snapshotID)
		case *genWf.CallMicroflowTask:
			if v.MicroflowQualifiedName() != "" {
				if _, err := stmt.Exec("WORKFLOW", sourceID, sourceQN,
					"MICROFLOW", "", v.MicroflowQualifiedName(),
					RefKindCall, moduleName, projectID, snapshotID); err == nil {
					refCount++
				}
			}
			refCount += b.extractConditionOutcomeRefsGen(stmt, v.OutcomesItems(), sourceID, sourceQN, moduleName, projectID, snapshotID)
		case *genWf.CallWorkflowActivity:
			if v.WorkflowQualifiedName() != "" {
				if _, err := stmt.Exec("WORKFLOW", sourceID, sourceQN,
					"WORKFLOW", "", v.WorkflowQualifiedName(),
					RefKindCall, moduleName, projectID, snapshotID); err == nil {
					refCount++
				}
			}
		case *genWf.ExclusiveSplitActivity:
			refCount += b.extractConditionOutcomeRefsGen(stmt, v.OutcomesItems(), sourceID, sourceQN, moduleName, projectID, snapshotID)
		case *genWf.ParallelSplitActivity:
			for _, oc := range v.OutcomesItems() {
				if pso, ok := oc.(*genWf.ParallelSplitOutcome); ok {
					if f, ok := pso.Flow().(*genWf.Flow); ok {
						refCount += b.extractWorkflowFlowRefsGen(stmt, f, sourceID, sourceQN, moduleName, projectID, snapshotID)
					}
				}
			}
		default:
			// Workflows$SystemTask: gen has no narrow type, read raw fields.
			if act.TypeName() == "Workflows$SystemTask" {
				if mf, _ := /* Microflow */ stringFromRaw(act, "Microflow"); mf != "" {
					if _, err := stmt.Exec("WORKFLOW", sourceID, sourceQN,
						"MICROFLOW", "", mf,
						RefKindCall, moduleName, projectID, snapshotID); err == nil {
						refCount++
					}
				}
				// SystemTask outcomes are not exposed via raw extraction
				// (legacy walked them; gen schema gap — minor: only loses
				// transitive references through nested condition outcomes).
			}
		}
	}
	return refCount
}

// extractUserSourceRefGen emits the microflow reference from a
// MicroflowBasedUserSource if present.
func extractUserSourceRefGen(stmt *sql.Stmt, src element.Element, sourceID, sourceQN, moduleName, projectID, snapshotID string) int {
	if src == nil {
		return 0
	}
	if mb, ok := src.(*genWf.MicroflowBasedUserSource); ok && mb.MicroflowQualifiedName() != "" {
		if _, err := stmt.Exec("WORKFLOW", sourceID, sourceQN,
			"MICROFLOW", "", mb.MicroflowQualifiedName(),
			RefKindCall, moduleName, projectID, snapshotID); err == nil {
			return 1
		}
	}
	return 0
}

// extractConditionOutcomeRefsGen recurses into ConditionOutcome flows.
func (b *Builder) extractConditionOutcomeRefsGen(stmt *sql.Stmt, outcomes []element.Element, sourceID, sourceQN, moduleName, projectID, snapshotID string) int {
	refCount := 0
	for _, oc := range outcomes {
		var f *genWf.Flow
		switch v := oc.(type) {
		case *genWf.BooleanConditionOutcome:
			f, _ = v.Flow().(*genWf.Flow)
		case *genWf.EnumerationValueConditionOutcome:
			f, _ = v.Flow().(*genWf.Flow)
		case *genWf.VoidConditionOutcome:
			f, _ = v.Flow().(*genWf.Flow)
		case *genWf.ExclusiveSplitOutcome:
			f, _ = v.Flow().(*genWf.Flow)
		}
		if f != nil {
			refCount += b.extractWorkflowFlowRefsGen(stmt, f, sourceID, sourceQN, moduleName, projectID, snapshotID)
		}
	}
	return refCount
}

// stringFromRaw extracts a top-level string field from an element's
// raw BSON. Used for SystemTask which has no narrow gen type.
func stringFromRaw(elem element.Element, key string) (string, error) {
	if elem == nil {
		return "", nil
	}
	return codec.ReadBSONFieldString(elem.Raw(), key)
}
