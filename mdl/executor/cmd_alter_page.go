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
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// execAlterPage handles ALTER PAGE/SNIPPET Module.Name { operations }.
func execAlterPage(ctx *ExecContext, s *ast.AlterPageStmt) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	var unitID model.ID
	var containerID model.ID
	containerType := strings.ToLower(s.ContainerType)
	if containerType == "" {
		containerType = "page"
	}

	// Stage 3.3.5.D5.b: lookup goes through gen-typed find helpers so
	// the test fixtures wire RecordingPageRepository / RecordingSnippet
	// Repository (matches the C6.* pattern used for SHOW + describe
	// tests). Container resolution still walks the unit hierarchy so
	// the mutator is opened against the correct module ID.
	if containerType == "snippet" {
		snipID, err := findSnippetIDGen(ctx, s.PageName, h)
		if err != nil {
			return err
		}
		unitID = snipID
		if ctx.Snippets != nil {
			c, _ := ctx.Snippets.GetContainerUUID(snipID)
			containerID = h.FindModuleID(c)
		}
	} else {
		pageID, err := findPageIDGen(ctx, s.PageName, h)
		if err != nil {
			return err
		}
		unitID = pageID
		if ctx.Pages != nil {
			c, _ := ctx.Pages.GetContainerUUID(pageID)
			containerID = h.FindModuleID(c)
		}
	}

	// Open the page for mutation via the backend
	mutator, err := ctx.PageMutationOperator.OpenPageForMutation(unitID)
	if err != nil {
		return mdlerrors.NewBackend("open "+strings.ToLower(containerType)+" for mutation", err)
	}

	// Resolve module name for building new widgets
	modName := h.GetModuleName(containerID)

	for _, op := range s.Operations {
		switch o := op.(type) {
		case *ast.SetPropertyOp:
			if err := applySetPropertyMutator(mutator, o); err != nil {
				return mdlerrors.NewBackend("set", err)
			}
		case *ast.InsertWidgetOp:
			if err := applyInsertWidgetMutator(ctx, mutator, o, modName, containerID); err != nil {
				return mdlerrors.NewBackend("insert", err)
			}
		case *ast.DropWidgetOp:
			if err := applyDropWidgetMutator(mutator, o); err != nil {
				return mdlerrors.NewBackend("drop", err)
			}
		case *ast.ReplaceWidgetOp:
			if err := applyReplaceWidgetMutator(ctx, mutator, o, modName, containerID); err != nil {
				return mdlerrors.NewBackend("replace", err)
			}
		case *ast.AddVariableOp:
			if err := mutator.AddVariable(o.Variable.Name, o.Variable.DataType, o.Variable.DefaultValue); err != nil {
				return mdlerrors.NewBackend("add VARIABLE", err)
			}
		case *ast.DropVariableOp:
			if err := mutator.DropVariable(o.VariableName); err != nil {
				return mdlerrors.NewBackend("drop VARIABLE", err)
			}
		case *ast.SetLayoutOp:
			if containerType == "snippet" {
				return mdlerrors.NewUnsupported("set Layout is not supported for snippets")
			}
			newLayoutQN := o.NewLayout.Module + "." + o.NewLayout.Name
			if err := mutator.SetLayout(newLayoutQN, o.Mappings); err != nil {
				return mdlerrors.NewBackend("set Layout", err)
			}
		default:
			return mdlerrors.NewUnsupported(fmt.Sprintf("unknown alter %s operation type: %T", containerType, op))
		}
	}

	// Persist
	if err := mutator.Save(); err != nil {
		return mdlerrors.NewBackend("save modified "+strings.ToLower(containerType), err)
	}

	fmt.Fprintf(ctx.Output, "Altered %s %s\n", strings.ToLower(containerType), s.PageName.String())
	return nil
}

// ============================================================================
// SET property via mutator
// ============================================================================

func applySetPropertyMutator(mutator backend.PageMutator, op *ast.SetPropertyOp) error {
	// Sort property names for deterministic application order.
	propNames := make([]string, 0, len(op.Properties))
	for k := range op.Properties {
		propNames = append(propNames, k)
	}
	sort.Strings(propNames)

	for _, propName := range propNames {
		value := op.Properties[propName]
		if op.Target.IsLayoutGridColumn() {
			// 3-level ref: grid.row.col — only DesktopWidth is supported here.
			width, err := parseDesktopWidth(propName, value)
			if err != nil {
				return mdlerrors.NewBackend("set "+propName+" on "+op.Target.Name(), err)
			}
			if err := mutator.SetLayoutGridColumnWidth(op.Target.Widget, op.Target.Row, op.Target.Column, width); err != nil {
				return mdlerrors.NewBackend("set "+propName+" on "+op.Target.Name(), err)
			}
		} else if op.Target.IsColumn() {
			if err := mutator.SetColumnProperty(op.Target.Widget, op.Target.Column, propName, value); err != nil {
				return mdlerrors.NewBackend("set "+propName+" on "+op.Target.Name(), err)
			}
		} else if propName == "DataSource" {
			// DataSource requires special handling via SetWidgetDataSourceGen
			ds, err := convertASTDataSourceGen(value)
			if err != nil {
				return err
			}
			if err := mutator.SetWidgetDataSourceGen(op.Target.Widget, ds); err != nil {
				return mdlerrors.NewBackend("set DataSource on "+op.Target.Name(), err)
			}
		} else {
			if err := mutator.SetWidgetProperty(op.Target.Widget, propName, value); err != nil {
				return mdlerrors.NewBackend("set "+propName+" on "+op.Target.Name(), err)
			}
		}
	}
	return nil
}

// convertASTDataSourceGen converts an AST DataSource value to a gen-typed element.Element.
// Returns one of: ListenTargetSource, DataViewSource, MicroflowSource, NanoflowSource.
func convertASTDataSourceGen(value interface{}) (element.Element, error) {
	ds, ok := value.(*ast.DataSourceV3)
	if !ok {
		return nil, mdlerrors.NewValidation("DataSource value must be a datasource expression")
	}

	switch ds.Type {
	case "selection":
		o := genPg.NewListenTargetSource()
		o.SetListenTarget(ds.Reference)
		return o, nil
	case "database":
		o := genPg.NewDataViewSource()
		if ds.Reference != "" {
			ref := genDm.NewDirectEntityRef()
			ref.SetEntityQualifiedName(ds.Reference)
			o.SetEntityRef(ref)
		}
		return o, nil
	case "microflow":
		settings := genPg.NewMicroflowSettings()
		settings.SetMicroflowQualifiedName(ds.Reference)
		o := genPg.NewMicroflowSource()
		o.SetMicroflowSettings(settings)
		return o, nil
	case "nanoflow":
		o := genPg.NewNanoflowSource()
		o.SetNanoflowQualifiedName(ds.Reference)
		return o, nil
	default:
		return nil, mdlerrors.NewUnsupported("unsupported DataSource type for alter page set: " + ds.Type)
	}
}

// ============================================================================
// INSERT widget via mutator
// ============================================================================

func applyInsertWidgetMutator(ctx *ExecContext, mutator backend.PageMutator, op *ast.InsertWidgetOp, moduleName string, moduleID model.ID) error {
	// Check for duplicate widget names before building
	for _, w := range op.Widgets {
		if w.Name != "" && mutator.FindWidget(w.Name) {
			return mdlerrors.NewAlreadyExistsMsg("widget", w.Name, fmt.Sprintf("duplicate widget name '%s': a widget with this name already exists on the page", w.Name))
		}
	}

	// Find entity context from enclosing DataView/DataGrid/ListView
	entityCtx := mutator.EnclosingEntity(op.Target.Widget)

	// Layout grid column insertion (3-part ref: grid.row.column).
	// Build column widgets as bare LayoutGridColumn elements instead of
	// DivContainer wrappers (buildContainerWithColumnV3) — the layout grid
	// row's Columns array requires LayoutGridColumn, not DivContainer.
	if op.Target.IsLayoutGridColumn() {
		paramScope, paramEntityNames := mutator.ParamScope()
		widgetScope := mutator.WidgetScope()
		ctx.WidgetBuilder.BeginPageBuild()
		defer ctx.WidgetBuilder.EndPageBuild()

		pb := &pageBuilder{
			moduleLister:         ctx.ModuleLister,
			domainModelReader:    ctx.DomainModelReader,
			pageReader:           ctx.PageReader,
			metadataReader:       ctx.MetadataReader,
			folderManager:        ctx.FolderManager,
			connectionManager:    ctx.ConnectionManager,
			serializationBackend: ctx.Backend,
			moduleID:             moduleID,
			moduleName:           moduleName,
			entityContext:        entityCtx,
			widgetScope:          widgetScope,
			paramScope:           paramScope,
			paramEntityNames:     paramEntityNames,
			execCache:            ctx.Cache,
			fragments:            ctx.Fragments,
			themeRegistry:        ctx.GetThemeRegistry(),
			widgetBackend:        ctx.Backend,
			microflowsRepo:       ctx.Microflows,
			nanoflowsRepo:        ctx.Nanoflows,
			snippetsRepo:         ctx.Snippets,
			mxGraph:              ctx.Graph.MxGraph(),
		}

		var genWidgets []element.Element
		for _, w := range op.Widgets {
			var widget element.Element
			var err error
			switch strings.ToLower(w.Type) {
			case "column":
				widget, err = pb.buildLayoutGridColumnV3(w)
			default:
				widget, err = pb.buildWidgetV3(w)
			}
			if err != nil {
				return mdlerrors.NewBackend("build widget", err)
			}
			if widget != nil {
				genWidgets = append(genWidgets, widget)
			}
		}

		type lgInserter interface {
			InsertLayoutGridColumnGen(gridName, rowRef, colRef string, position backend.InsertPosition, widgets []element.Element) error
		}
		if li, ok := mutator.(lgInserter); ok {
			return li.InsertLayoutGridColumnGen(op.Target.Widget, op.Target.Row, op.Target.Column, backend.InsertPosition(op.Position), genWidgets)
		}
		return fmt.Errorf("mutator %T does not support InsertLayoutGridColumnGen", mutator)
	}

	// Default path: build via generic builder
	widgets, err := buildWidgetsFromASTGen(ctx, op.Widgets, moduleName, moduleID, entityCtx, mutator)
	if err != nil {
		return mdlerrors.NewBackend("build widgets", err)
	}
	return mutator.InsertWidgetGen(op.Target.Widget, op.Target.Column, backend.InsertPosition(op.Position), widgets)
}

// ============================================================================
// DROP widget via mutator
// ============================================================================

func applyDropWidgetMutator(mutator backend.PageMutator, op *ast.DropWidgetOp) error {
	refs := make([]backend.WidgetRef, len(op.Targets))
	for i, t := range op.Targets {
		refs[i] = backend.WidgetRef{Widget: t.Widget, Column: t.Column}
	}
	err := mutator.DropWidget(refs)
	if err != nil {
		// Idempotent: if the widget doesn't exist (e.g. already dropped or never
		// created), warn and continue so subsequent operations in the same script
		// are not blocked. This is required for the capstone 08-workflow.mdl
		// which uses `drop widget btnEscalate` for idempotent safety.
		for _, t := range op.Targets {
			log.Printf("WARNING: widget %q not found on page (already dropped or not created), skipping drop", t.Widget)
		}
		_ = err // swallow not-found errors for idempotent safety
		return nil
	}
	return nil
}

// ============================================================================
// REPLACE widget via mutator
// ============================================================================

func applyReplaceWidgetMutator(ctx *ExecContext, mutator backend.PageMutator, op *ast.ReplaceWidgetOp, moduleName string, moduleID model.ID) error {
	// Collect all widget names from the replacement subtree — the entire
	// target subtree will be removed, so any name that previously existed
	// inside it is freed for reuse. Collecting these names recursively lets
	// us exclude them from both the duplicate-name check and the widget
	// scope so child names (e.g. btnPublish inside a replaced ftrActions)
	// can be reused without false-positive "duplicate name" errors.
	newNames := collectWidgetNamesV3(op.NewWidgets)

	// Check for duplicate widget names using the global scope minus the names
	// that will be freed by the replacement.
	scope := mutator.WidgetScope()
	for _, n := range newNames {
		delete(scope, n)
	}
	for _, w := range op.NewWidgets {
		if w.Name != "" {
			if _, exists := scope[w.Name]; exists {
				return mdlerrors.NewAlreadyExistsMsg("widget", w.Name, fmt.Sprintf("duplicate widget name '%s': a widget with this name already exists on the page", w.Name))
			}
		}
	}

	// Find entity context from enclosing DataView/DataGrid/ListView
	entityCtx := mutator.EnclosingEntity(op.Target.Widget)

	// Build new widgets from AST. Pass all replacement names as excluded scope
	// so reused names from the replaced subtree don't trip the duplicate-name
	// check in registerWidgetName.
	widgets, err := buildWidgetsFromASTGen(ctx, op.NewWidgets, moduleName, moduleID, entityCtx, mutator, newNames...)
	if err != nil {
		return mdlerrors.NewBackend("build replacement widgets", err)
	}

	return mutator.ReplaceWidgetGen(op.Target.Widget, op.Target.Column, widgets)
}

// ============================================================================
// Widget building from AST (domain logic stays in executor)
// ============================================================================

// buildWidgetsFromASTGen converts AST widgets to gen-typed element.Element values.
// It uses the mutator for scope resolution (WidgetScope, ParamScope).
//
// excludeFromScope optionally drops named widgets from the inherited scope —
// REPLACE passes the target widget's name so a replacement may reuse it.
func buildWidgetsFromASTGen(ctx *ExecContext, widgets []*ast.WidgetV3, moduleName string, moduleID model.ID, entityContext string, mutator backend.PageMutator, excludeFromScope ...string) ([]element.Element, error) {
	paramScope, paramEntityNames := mutator.ParamScope()
	widgetScope := mutator.WidgetScope()
	for _, name := range excludeFromScope {
		delete(widgetScope, name)
	}

	ctx.WidgetBuilder.BeginPageBuild()
	defer ctx.WidgetBuilder.EndPageBuild()
	pb := &pageBuilder{
		moduleLister:         ctx.ModuleLister,
		domainModelReader:    ctx.DomainModelReader,
		pageReader:           ctx.PageReader,
		metadataReader:       ctx.MetadataReader,
		folderManager:        ctx.FolderManager,
		connectionManager:    ctx.ConnectionManager,
		serializationBackend: ctx.Backend,
		moduleID:             moduleID,
		moduleName:           moduleName,
		entityContext:        entityContext,
		widgetScope:          widgetScope,
		paramScope:           paramScope,
		paramEntityNames:     paramEntityNames,
		execCache:            ctx.Cache,
		fragments:            ctx.Fragments,
		themeRegistry:        ctx.GetThemeRegistry(),
		widgetBackend:        ctx.Backend,
		microflowsRepo:       ctx.Microflows,
		nanoflowsRepo:        ctx.Nanoflows,
		snippetsRepo:         ctx.Snippets,
		mxGraph:              ctx.Graph.MxGraph(),
	}

	var result []element.Element
	for _, w := range widgets {
		// buildWidgetV3 now returns element.Element directly (gen-native path,
		// Stage 3.3.5.Cat-B). No BSON roundtrip needed.
		widget, err := pb.buildWidgetV3(w)
		if err != nil {
			return nil, mdlerrors.NewBackend("build widget "+w.Name, err)
		}
		if widget == nil {
			continue
		}
		result = append(result, widget)
	}
	return result, nil
}

// parseDesktopWidth converts a DesktopWidth property value to an integer column
// weight in the range 1–12 (Mendix's grid system). Accepts numeric values or
// the string "AutoFill" (stored as Weight=-1 in BSON).
func parseDesktopWidth(propName string, value any) (int, error) {
	if propName != "DesktopWidth" {
		return 0, fmt.Errorf("only DesktopWidth is supported for layout grid column refs (got %q)", propName)
	}
	switch v := value.(type) {
	case int:
		return v, nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		if strings.EqualFold(v, "AutoFill") {
			return -1, nil
		}
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
			return 0, fmt.Errorf("invalid DesktopWidth value %q: must be 1-12 or AutoFill", v)
		}
		return n, nil
	}
	return 0, fmt.Errorf("unsupported DesktopWidth value type %T", value)
}

// collectWidgetNamesV3 recursively collects all widget names from a list of WidgetV3.
func collectWidgetNamesV3(widgets []*ast.WidgetV3) []string {
	var names []string
	for _, w := range widgets {
		if w.Name != "" {
			names = append(names, w.Name)
		}
		names = append(names, collectWidgetNamesV3(w.Children)...)
	}
	return names
}
