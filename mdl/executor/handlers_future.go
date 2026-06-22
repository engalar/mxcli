package executor

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/cmd/mxcli/syntax"
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

// execHelpFuture is the ExecContext-free version of execHelp.
func execHelpFuture(ctx context.Context, s *ast.HelpStmt, output io.Writer, format OutputFormat) error {
	if len(s.Topic) > 0 {
		path := syntax.ResolveAlias(resolveHelpPath(s.Topic))
		features := syntax.ByPrefix(path)
		if len(features) == 0 {
			fmt.Fprintf(output, "No syntax help found for: %s\n", path)
			fmt.Fprintln(output, "Use HELP; for a command overview.")
			return nil
		}
		if format == FormatJSON {
			return syntax.WriteJSON(output, features)
		}
		syntax.WriteText(output, features)
		return nil
	}

	fmt.Fprint(output, `MDL Commands:

Connection:
  connect local '<path>'      Connect to local .mpr file
  disconnect                  Disconnect from project
  status                      Show connection status

Domain Model - Enumerations:
  create enumeration Module.Name ...   Create enumeration
  alter  enumeration Module.Name ...   Alter enumeration
  drop   enumeration Module.Name       Drop enumeration

Domain Model - Entities:
  create entity Module.Name ( ... )   Create entity (with attributes and associations)
  alter  entity Module.Name ...       Alter entity
  drop   entity Module.Name            Drop entity

Microflows:
  create microflow Module.Name ( ... )  Create microflow
  drop   microflow Module.Name          Drop microflow

For detailed help: HELP <topic>, e.g. HELP create entity
`)
	return nil
}

// execExitFuture is the ExecContext-free version of execExit.
func execExitFuture(ctx context.Context) error {
	return ErrExit
}

// execShowCatalogTablesFuture is the ExecContext-free version of execShowCatalogTables.
func execShowCatalogTablesFuture(ctx context.Context, output io.Writer) error {
	fmt.Fprintf(output, "Catalog SQLite system has been replaced by MXGraph.\n")
	fmt.Fprintf(output, "Use SHOW MODULES / SHOW ENTITIES / SHOW PAGES / etc. directly.\n")
	return nil
}

// execShowCatalogStatusFuture is the ExecContext-free version of execShowCatalogStatus.
func execShowCatalogStatusFuture(ctx context.Context, output io.Writer) error {
	fmt.Fprintf(output, "Catalog SQLite system has been replaced by MXGraph.\n")
	return nil
}

// listVersionFuture is the ExecContext-free version of listVersion.
func listVersionFuture(ctx context.Context, output io.Writer, cm backend.ConnectionManager) error {
	if !cm.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	pv := cm.ProjectVersion()
	fmt.Fprintf(output, "Mendix Version: %s\n", pv.ProductVersion)
	fmt.Fprintf(output, "Build Version:  %s\n", pv.BuildVersion)
	fmt.Fprintf(output, "MPR Format:     v%d\n", pv.FormatVersion)
	if pv.SchemaHash != "" {
		fmt.Fprintf(output, "Schema Hash:    %s\n", pv.SchemaHash)
	}
	return nil
}

// listFragmentsFuture is the ExecContext-free version of listFragments.
func listFragmentsFuture(ctx context.Context, output io.Writer, fragments map[string]*ast.DefineFragmentStmt) error {
	if len(fragments) == 0 {
		fmt.Fprintln(output, "No fragments defined.")
		return nil
	}

	names := make([]string, 0, len(fragments))
	for name := range fragments {
		names = append(names, name)
	}
	sort.Strings(names)

	fmt.Fprintf(output, "%-30s %s\n", "Fragment", "Widgets")
	fmt.Fprintf(output, "%-30s %s\n", strings.Repeat("-", 30), strings.Repeat("-", 10))
	for _, name := range names {
		frag := fragments[name]
		fmt.Fprintf(output, "%-30s %d\n", name, len(frag.Widgets))
	}
	return nil
}

// listModulesFuture is the ExecContext-free version of listModules.
func listModulesFuture(
	ctx context.Context,
	output io.Writer,
	format OutputFormat,
	cm backend.ConnectionManager,
	ml backend.ModuleLister,
	mr backend.MetadataReader,
	fm backend.FolderManager,
	dmr repos.DomainModelRepository,
) error {
	if cm == nil || !cm.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	modules, err := ml.ListModules()
	if err != nil {
		return mdlerrors.NewBackend("list modules", err)
	}

	h, err := NewContainerHierarchyFromRoles(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	units, err := mr.ListUnits()
	if err != nil {
		return mdlerrors.NewBackend("list units", err)
	}

	entityCounts := make(map[model.ID]int)
	enumCounts := make(map[model.ID]int)
	pageCounts := make(map[model.ID]int)
	snippetCounts := make(map[model.ID]int)
	microflowCounts := make(map[model.ID]int)
	nanoflowCounts := make(map[model.ID]int)
	constantCounts := make(map[model.ID]int)
	javaActionCounts := make(map[model.ID]int)
	workflowCounts := make(map[model.ID]int)
	pubRestCounts := make(map[model.ID]int)
	pubODataCounts := make(map[model.ID]int)
	conODataCounts := make(map[model.ID]int)
	bizEventCounts := make(map[model.ID]int)
	extDbCounts := make(map[model.ID]int)

	for _, u := range units {
		modID := h.FindModuleID(u.ContainerID)
		switch u.Type {
		case types.UnitTypePage:
			pageCounts[modID]++
		case types.UnitTypeSnippet:
			snippetCounts[modID]++
		case types.UnitTypeMicroflow:
			microflowCounts[modID]++
		case types.UnitTypeNanoflow:
			nanoflowCounts[modID]++
		case types.UnitTypeEnumeration:
			enumCounts[modID]++
		case types.UnitTypeConstant:
			constantCounts[modID]++
		case types.UnitTypeJavaAction:
			javaActionCounts[modID]++
		case types.UnitTypePublishedRestService:
			pubRestCounts[modID]++
		case types.UnitTypePublishedODataService:
			pubODataCounts[modID]++
		case types.UnitTypeConsumedODataService:
			conODataCounts[modID]++
		case types.UnitTypeWorkflow:
			workflowCounts[modID]++
		case types.UnitTypeDatabaseConnection:
			extDbCounts[modID]++
		default:
			if strings.HasPrefix(u.Type, types.UnitTypeBusinessEventPrefix) {
				bizEventCounts[modID]++
			}
		}
	}

	if dmr != nil {
		if pairs, err := dmr.ListAllWithContainerID(); err == nil {
			for _, p := range pairs {
				if p.DM == nil {
					continue
				}
				modID := h.FindModuleID(p.ContainerID)
				entityCounts[modID] += len(p.DM.EntitiesItems())
			}
		}
	}

	sort.Slice(modules, func(i, j int) bool {
		return strings.ToLower(modules[i].Name) < strings.ToLower(modules[j].Name)
	})

	type row struct {
		name        string
		source      string
		entities    int
		enums       int
		pages       int
		snippets    int
		microflows  int
		nanoflows   int
		workflows   int
		constants   int
		javaActions int
		pubRest     int
		pubOData    int
		conOData    int
		bizEvents   int
		extDb       int
	}
	var rows []row

	for _, m := range modules {
		source := ""
		if m.FromAppStore {
			if m.AppStoreVersion != "" {
				source = "Marketplace v" + m.AppStoreVersion
			} else {
				source = "Marketplace"
			}
		}

		r := row{
			name:        m.Name,
			source:      source,
			entities:    entityCounts[m.ID],
			enums:       enumCounts[m.ID],
			pages:       pageCounts[m.ID],
			snippets:    snippetCounts[m.ID],
			microflows:  microflowCounts[m.ID],
			nanoflows:   nanoflowCounts[m.ID],
			workflows:   workflowCounts[m.ID],
			constants:   constantCounts[m.ID],
			javaActions: javaActionCounts[m.ID],
			pubRest:     pubRestCounts[m.ID],
			pubOData:    pubODataCounts[m.ID],
			conOData:    conODataCounts[m.ID],
			bizEvents:   bizEventCounts[m.ID],
			extDb:       extDbCounts[m.ID],
		}
		rows = append(rows, r)
	}

	result := &TableResult{
		Columns: []string{"Module", "Source", "Entities", "Enums", "Pages", "Snippets", "Microflows", "Nanoflows", "Workflows", "Constants", "JavaActions", "PubREST", "PubOData", "ConOData", "BizEvents", "ExtDB"},
		Summary: fmt.Sprintf("(%d modules)", len(modules)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.name, r.source, r.entities, r.enums, r.pages, r.snippets, r.microflows, r.nanoflows, r.workflows, r.constants, r.javaActions, r.pubRest, r.pubOData, r.conOData, r.bizEvents, r.extDb})
	}
	return writeResultTo(output, format, result)
}
