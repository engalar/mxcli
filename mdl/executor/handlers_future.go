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
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
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

// listEnumerationsFuture is the ExecContext-free version of listEnumerations.
func listEnumerationsFuture(
	ctx context.Context,
	output io.Writer,
	format OutputFormat,
	cm backend.ConnectionManager,
	er backend.EnumerationReader,
	ml backend.ModuleLister,
	mr backend.MetadataReader,
	fm backend.FolderManager,
	inModule string,
) error {
	if cm == nil || !cm.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	enums, err := er.ListEnumerations()
	if err != nil {
		return mdlerrors.NewBackend("list enumerations", err)
	}

	h, err := NewContainerHierarchyFromRoles(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	type row struct {
		qualifiedName string
		module        string
		name          string
		folderPath    string
		values        int
	}
	var rows []row

	for _, enum := range enums {
		modID := h.FindModuleID(enum.ContainerID)
		modName := h.GetModuleName(modID)
		if inModule == "" || modName == inModule {
			qualifiedName := modName + "." + enum.Name
			folderPath := h.BuildFolderPath(enum.ContainerID)
			rows = append(rows, row{qualifiedName, modName, enum.Name, folderPath, len(enum.Values)})
		}
	}

	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	result := &TableResult{
		Columns: []string{"Qualified Name", "Module", "Name", "Folder", "Values"},
		Summary: fmt.Sprintf("(%d enumerations)", len(rows)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.qualifiedName, r.module, r.name, r.folderPath, r.values})
	}
	return writeResultTo(output, format, result)
}

// listConstantsFuture is the ExecContext-free version of listConstants.
func listConstantsFuture(
	ctx context.Context,
	output io.Writer,
	format OutputFormat,
	cm backend.ConnectionManager,
	cr backend.ConstantReader,
	ml backend.ModuleLister,
	mr backend.MetadataReader,
	fm backend.FolderManager,
	inModule string,
) error {
	if cm == nil || !cm.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	constants, err := cr.ListConstants()
	if err != nil {
		return mdlerrors.NewBackend("list constants", err)
	}

	h, err := NewContainerHierarchyFromRoles(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	type row struct {
		qualifiedName string
		module        string
		name          string
		folderPath    string
		typeStr       string
		defaultStr    string
		exposed       string
	}
	var rows []row

	for _, c := range constants {
		modID := h.FindModuleID(c.ContainerID)
		modName := h.GetModuleName(modID)
		if inModule != "" && !strings.EqualFold(modName, inModule) {
			continue
		}
		qualifiedName := modName + "." + c.Name
		folderPath := h.BuildFolderPath(c.ContainerID)
		typeStr := formatConstantType(c.Type)
		defaultStr := c.DefaultValue
		if len(defaultStr) > 40 {
			defaultStr = defaultStr[:37] + "..."
		}
		exposed := "No"
		if c.ExposedToClient {
			exposed = "Yes"
		}
		rows = append(rows, row{qualifiedName, modName, c.Name, folderPath, typeStr, defaultStr, exposed})
	}

	if len(rows) == 0 && format != FormatJSON {
		fmt.Fprintln(output, "No constants found.")
		return nil
	}

	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	result := &TableResult{
		Columns: []string{"Qualified Name", "Module", "Name", "Folder", "Type", "Default", "Exposed"},
		Summary: fmt.Sprintf("(%d constants)", len(rows)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.qualifiedName, r.module, r.name, r.folderPath, r.typeStr, r.defaultStr, r.exposed})
	}
	return writeResultTo(output, format, result)
}

// listConstantValuesFuture is the ExecContext-free version of listConstantValues.
func listConstantValuesFuture(
	ctx context.Context,
	output io.Writer,
	format OutputFormat,
	cm backend.ConnectionManager,
	cr backend.ConstantReader,
	sr backend.SettingsReader,
	ml backend.ModuleLister,
	mr backend.MetadataReader,
	fm backend.FolderManager,
	inModule string,
) error {
	if cm == nil || !cm.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	constants, err := cr.ListConstants()
	if err != nil {
		return mdlerrors.NewBackend("list constants", err)
	}

	ps, err := sr.GetProjectSettings()
	if err != nil {
		return mdlerrors.NewBackend("read project settings", err)
	}

	h, err := NewContainerHierarchyFromRoles(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	type constInfo struct {
		qualifiedName string
		defaultValue  string
		typeStr       string
	}
	var consts []constInfo
	for _, c := range constants {
		modID := h.FindModuleID(c.ContainerID)
		modName := h.GetModuleName(modID)
		if inModule != "" && !strings.EqualFold(modName, inModule) {
			continue
		}
		consts = append(consts, constInfo{
			qualifiedName: modName + "." + c.Name,
			defaultValue:  c.DefaultValue,
			typeStr:       formatConstantType(c.Type),
		})
	}

	if len(consts) == 0 && format != FormatJSON {
		fmt.Fprintln(output, "No constants found.")
		return nil
	}

	sort.Slice(consts, func(i, j int) bool {
		return strings.ToLower(consts[i].qualifiedName) < strings.ToLower(consts[j].qualifiedName)
	})

	configValues := make(map[string]map[string]string)
	var configNames []string
	if ps.Configuration != nil {
		for _, cfg := range ps.Configuration.Configurations {
			configNames = append(configNames, cfg.Name)
			m := make(map[string]string)
			for _, cv := range cfg.ConstantValues {
				m[cv.ConstantId] = cv.Value
			}
			configValues[cfg.Name] = m
		}
	}

	type row struct {
		constant      string
		configuration string
		value         string
	}
	var rows []row

	for _, c := range consts {
		rows = append(rows, row{c.qualifiedName, "(default)", c.defaultValue})
		for _, cfgName := range configNames {
			if val, ok := configValues[cfgName][c.qualifiedName]; ok {
				rows = append(rows, row{c.qualifiedName, cfgName, val})
			}
		}
	}

	result := &TableResult{
		Columns: []string{"Constant", "Configuration", "Value"},
		Summary: fmt.Sprintf("(%d rows)", len(rows)),
	}
	for _, r := range rows {
		val := r.value
		if len(val) > 60 {
			val = val[:57] + "..."
		}
		result.Rows = append(result.Rows, []any{r.constant, r.configuration, val})
	}
	return writeResultTo(output, format, result)
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

// listEntitiesGenFuture is the ExecContext-free version of listEntitiesGen.
func listEntitiesGenFuture(
	ctx context.Context,
	output io.Writer,
	format OutputFormat,
	ml backend.ModuleLister,
	dmr repos.DomainModelRepository,
	inModule string,
) error {
	pairs, err := dmr.ListAllWithContainerID()
	if err != nil {
		return mdlerrors.NewBackend("list domain models", err)
	}

	mods, err := ml.ListModules()
	if err != nil {
		return mdlerrors.NewBackend("list modules", err)
	}
	moduleNames := make(map[model.ID]string, len(mods))
	moduleIDs := make(map[model.ID]bool, len(mods))
	for _, m := range mods {
		moduleNames[m.ID] = m.Name
		moduleIDs[m.ID] = true
	}

	assocCounts := make(map[model.ID]int)
	var validPairs []repos.DomainModelWithContainer
	for _, p := range pairs {
		if p.DM == nil || !moduleIDs[p.ContainerID] {
			continue
		}
		validPairs = append(validPairs, p)
		for _, a := range p.DM.AssociationsItems() {
			assoc, ok := a.(*genDm.Association)
			if !ok {
				continue
			}
			parent := model.ID(assoc.ParentRefID())
			child := model.ID(assoc.ChildRefID())
			if parent != "" {
				assocCounts[parent]++
			}
			if child != "" {
				assocCounts[child]++
			}
		}
	}

	systemEntitiesSet := make(map[string]bool)
	for _, p := range validPairs {
		for _, e := range p.DM.EntitiesItems() {
			entity, ok := e.(*genDm.Entity)
			if !ok {
				continue
			}
			gen := entityGeneralizationQNGen(entity)
			if gen != "" && strings.HasPrefix(gen, "System.") {
				systemEntitiesSet[gen] = true
			}
		}
	}

	type row struct {
		qualifiedName  string
		entityType     string
		generalization string
		attrs          int
		assocs         int
		validations    int
		indexes        int
		events         int
		accessRules    int
	}
	var rows []row

	if inModule == "" || inModule == "System" {
		systemNames := make([]string, 0, len(systemEntitiesSet))
		for n := range systemEntitiesSet {
			systemNames = append(systemNames, n)
		}
		sort.Strings(systemNames)
		for _, n := range systemNames {
			rows = append(rows, row{
				qualifiedName: n,
				entityType:    "System",
				attrs:         -1,
				assocs:        -1,
				validations:   -1,
				indexes:       -1,
				events:        -1,
				accessRules:   -1,
			})
		}
	}

	for _, p := range validPairs {
		modName := moduleNames[p.ContainerID]
		if inModule != "" && modName != inModule {
			continue
		}
		for _, e := range p.DM.EntitiesItems() {
			entity, ok := e.(*genDm.Entity)
			if !ok {
				continue
			}
			rows = append(rows, row{
				qualifiedName:  modName + "." + entity.Name(),
				entityType:     entityKindForGen(entity),
				generalization: entityGeneralizationQNGen(entity),
				attrs:          len(entity.AttributesItems()),
				assocs:         assocCounts[model.ID(entity.ID())],
				validations:    len(entity.ValidationRulesItems()),
				indexes:        len(entity.IndexesItems()),
				events:         len(entity.EventHandlersItems()),
				accessRules:    len(entity.AccessRulesItems()),
			})
		}
	}

	hasGeneralizations := false
	for _, r := range rows {
		if r.generalization != "" {
			hasGeneralizations = true
			break
		}
	}

	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	columns := []string{"Entity", "Type"}
	if hasGeneralizations {
		columns = append(columns, "Extends")
	}
	columns = append(columns, "Attrs", "Assocs", "Validations", "Indexes", "Events", "AccessRules")

	result := &TableResult{
		Columns: columns,
		Summary: fmt.Sprintf("(%d entities)", len(rows)),
	}
	for _, r := range rows {
		var rowData []any
		rowData = append(rowData, r.qualifiedName, r.entityType)
		if hasGeneralizations {
			rowData = append(rowData, r.generalization)
		}
		if r.entityType == "System" {
			rowData = append(rowData, "-", "-", "-", "-", "-", "-")
		} else {
			rowData = append(rowData, r.attrs, r.assocs, r.validations, r.indexes, r.events, r.accessRules)
		}
		result.Rows = append(result.Rows, rowData)
	}
	return writeResultTo(output, format, result)
}

// listEntityFuture is the ExecContext-free version of listEntity.
func listEntityFuture(
	ctx context.Context,
	output io.Writer,
	ml backend.ModuleLister,
	dmr repos.DomainModelRepository,
	name *ast.QualifiedName,
) error {
	if name == nil {
		return mdlerrors.NewValidation("entity name required")
	}

	entity, modName, err := findEntityFromRepos(ml, dmr, *name)
	if err != nil {
		return mdlerrors.NewBackend("get entity", err)
	}
	if entity == nil {
		return mdlerrors.NewNotFound("entity", name.String())
	}

	fmt.Fprintf(output, "**Entity: %s.%s**\n\n", modName, entity.Name())
	fmt.Fprintf(output, "- Type: %s\n", entityKindForGen(entity))
	if extends := entityGeneralizationQNGen(entity); extends != "" {
		fmt.Fprintf(output, "- Extends: %s\n", extends)
	}
	if loc := entity.Location(); loc != "" {
		fmt.Fprintf(output, "- Location: %s\n", loc)
	}
	fmt.Fprintln(output)

	if len(entity.AttributesItems()) > 0 {
		nameWidth, typeWidth := len("Attribute"), len("Type")
		type attrRow struct {
			name, typeName string
		}
		var rows []attrRow
		for _, a := range entity.AttributesItems() {
			attr, ok := a.(*genDm.Attribute)
			if !ok {
				continue
			}
			typeName := formatAttributeTypeGen(attr.Type())
			rows = append(rows, attrRow{attr.Name(), typeName})
			if len(attr.Name()) > nameWidth {
				nameWidth = len(attr.Name())
			}
			if len(typeName) > typeWidth {
				typeWidth = len(typeName)
			}
		}

		fmt.Fprintf(output, "| %-*s | %-*s |\n", nameWidth, "Attribute", typeWidth, "Type")
		fmt.Fprintf(output, "|-%s-|-%s-|\n", strings.Repeat("-", nameWidth), strings.Repeat("-", typeWidth))
		for _, r := range rows {
			fmt.Fprintf(output, "| %-*s | %-*s |\n", nameWidth, r.name, typeWidth, r.typeName)
		}
		fmt.Fprintf(output, "\n(%d attributes)\n", len(rows))
	}

	return nil
}

// findEntityFromRepos searches for an entity by qualified name across all domain models via repos.
func findEntityFromRepos(ml backend.ModuleLister, dmr repos.DomainModelRepository, qn ast.QualifiedName) (*genDm.Entity, string, error) {
	pairs, err := dmr.ListAllWithContainerID()
	if err != nil {
		return nil, "", err
	}
	mods, err := ml.ListModules()
	if err != nil {
		return nil, "", err
	}
	moduleNames := make(map[model.ID]string, len(mods))
	for _, m := range mods {
		moduleNames[m.ID] = m.Name
	}
	for _, p := range pairs {
		if p.DM == nil {
			continue
		}
		modName := moduleNames[p.ContainerID]
		if modName != qn.Module {
			continue
		}
		for _, e := range p.DM.EntitiesItems() {
			entity, ok := e.(*genDm.Entity)
			if !ok {
				continue
			}
			if entity.Name() == qn.Name {
				return entity, modName, nil
			}
		}
	}
	return nil, "", nil
}

// listAssociationsFuture is the ExecContext-free version of listAssociations.
func listAssociationsFuture(
	ctx context.Context,
	output io.Writer,
	format OutputFormat,
	ml backend.ModuleLister,
	dmr repos.DomainModelRepository,
	inModule string,
) error {
	pairs, err := dmr.ListAllWithContainerID()
	if err != nil {
		return mdlerrors.NewBackend("list domain models", err)
	}

	mods, err := ml.ListModules()
	if err != nil {
		return mdlerrors.NewBackend("list modules", err)
	}
	moduleNames := make(map[model.ID]string, len(mods))
	moduleIDs := make(map[model.ID]bool, len(mods))
	for _, m := range mods {
		moduleNames[m.ID] = m.Name
		moduleIDs[m.ID] = true
	}

	entityNames := make(map[model.ID]string)
	var validPairs []repos.DomainModelWithContainer
	for _, p := range pairs {
		if p.DM == nil || !moduleIDs[p.ContainerID] {
			continue
		}
		validPairs = append(validPairs, p)
		modName := moduleNames[p.ContainerID]
		for _, e := range p.DM.EntitiesItems() {
			entity, ok := e.(*genDm.Entity)
			if !ok {
				continue
			}
			entityNames[model.ID(entity.ID())] = modName + "." + entity.Name()
		}
	}

	type row struct {
		qualifiedName string
		module        string
		name          string
		parent        string
		child         string
		multiplicity  string
		assocType     string
		owner         string
		storage       string
	}
	var rows []row

	for _, p := range validPairs {
		modName := moduleNames[p.ContainerID]
		if inModule != "" && modName != inModule {
			continue
		}
		for _, a := range p.DM.AssociationsItems() {
			assoc, ok := a.(*genDm.Association)
			if !ok {
				continue
			}
			parentID := model.ID(assoc.ParentRefID())
			childID := model.ID(assoc.ChildRefID())
			parent := entityNames[parentID]
			if parent == "" {
				parent = string(parentID)
			}
			child := entityNames[childID]
			if child == "" {
				child = string(childID)
			}
			rows = append(rows, row{
				qualifiedName: modName + "." + assoc.Name(),
				module:        modName,
				name:          assoc.Name(),
				parent:        parent,
				child:         child,
				multiplicity:  associationMultiplicity(assoc.Type(), assoc.Owner()),
				assocType:     assoc.Type(),
				owner:         assoc.Owner(),
				storage:       assoc.StorageFormat(),
			})
		}
		for _, c := range p.DM.CrossAssociationsItems() {
			ca, ok := c.(*genDm.CrossAssociation)
			if !ok {
				continue
			}
			parentID := model.ID(ca.ParentRefID())
			parent := entityNames[parentID]
			if parent == "" {
				parent = string(parentID)
			}
			rows = append(rows, row{
				qualifiedName: modName + "." + ca.Name(),
				module:        modName,
				name:          ca.Name(),
				parent:        parent,
				child:         ca.ChildQualifiedName(),
				multiplicity:  associationMultiplicity(ca.Type(), ca.Owner()),
				assocType:     ca.Type(),
				owner:         ca.Owner(),
				storage:       ca.StorageFormat(),
			})
		}
	}

	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	result := &TableResult{
		Columns: []string{"Qualified Name", "Module", "Name", "FROM (owner)", "TO (referenced)", "Multiplicity", "Type", "Owner", "Storage"},
		Summary: fmt.Sprintf("(%d associations)", len(rows)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.qualifiedName, r.module, r.name, r.parent, r.child, r.multiplicity, r.assocType, r.owner, r.storage})
	}
	return writeResultTo(output, format, result)
}

// listAssociationFuture is the ExecContext-free version of listAssociation.
func listAssociationFuture(
	ctx context.Context,
	output io.Writer,
	ml backend.ModuleLister,
	dmr backend.DomainModelReader,
	name *ast.QualifiedName,
) error {
	if name == nil {
		return mdlerrors.NewValidation("association name required")
	}

	module, err := ml.GetModuleByName(name.Module)
	if err != nil {
		return mdlerrors.NewBackend("find module", err)
	}
	if module == nil {
		return mdlerrors.NewNotFound("module", name.Module)
	}

	dm, err := dmr.GetDomainModelGen(module.ID)
	if err != nil {
		return mdlerrors.NewBackend("get domain model", err)
	}
	if dm == nil {
		return mdlerrors.NewNotFound("association", name.String())
	}

	entityNames := make(map[model.ID]string)
	for _, e := range dm.EntitiesItems() {
		entity, ok := e.(*genDm.Entity)
		if !ok {
			continue
		}
		entityNames[model.ID(entity.ID())] = module.Name + "." + entity.Name()
	}

	for _, assocElem := range dm.AssociationsItems() {
		assoc, ok := assocElem.(*genDm.Association)
		if !ok || assoc.Name() != name.Name {
			continue
		}
		parent := entityNames[model.ID(assoc.ParentRefID())]
		child := entityNames[model.ID(assoc.ChildRefID())]
		fmt.Fprintf(output, "Association: %s.%s\n", module.Name, assoc.Name())
		fmt.Fprintf(output, "  Multiplicity: %s\n", associationMultiplicity(assoc.Type(), assoc.Owner()))
		fmt.Fprintf(output, "  FROM (owner): %s\n", parent)
		fmt.Fprintf(output, "  TO (referenced): %s\n", child)
		fmt.Fprintf(output, "  Type: %s\n", assoc.Type())
		fmt.Fprintf(output, "  Owner: %s\n", assoc.Owner())
		fmt.Fprintf(output, "  Storage: %s\n", assoc.StorageFormat())
		return nil
	}
	for _, crossElem := range dm.CrossAssociationsItems() {
		ca, ok := crossElem.(*genDm.CrossAssociation)
		if !ok || ca.Name() != name.Name {
			continue
		}
		parent := entityNames[model.ID(ca.ParentRefID())]
		fmt.Fprintf(output, "Association: %s.%s (cross-module)\n", module.Name, ca.Name())
		fmt.Fprintf(output, "  Multiplicity: %s\n", associationMultiplicity(ca.Type(), ca.Owner()))
		fmt.Fprintf(output, "  FROM (owner): %s\n", parent)
		fmt.Fprintf(output, "  TO (referenced): %s\n", ca.ChildQualifiedName())
		fmt.Fprintf(output, "  Type: %s\n", ca.Type())
		fmt.Fprintf(output, "  Owner: %s\n", ca.Owner())
		fmt.Fprintf(output, "  Storage: %s\n", ca.StorageFormat())
		return nil
	}

	return mdlerrors.NewNotFound("association", name.String())
}

// listMicroflowsFuture is the ExecContext-free version of listMicroflows.
func listMicroflowsFuture(
	ctx context.Context,
	output io.Writer,
	format OutputFormat,
	ml backend.ModuleLister,
	mr backend.MetadataReader,
	fm backend.FolderManager,
	mfRepo repos.MicroflowRepository,
	inModule string,
) error {
	h, err := NewContainerHierarchyFromRoles(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	mfs, err := mfRepo.ListAll()
	if err != nil {
		return mdlerrors.NewBackend("list microflows", err)
	}

	type row struct {
		qualifiedName string
		module        string
		name          string
		excluded      bool
		folderPath    string
		params        int
		activities    int
		complexity    int
		returnType    string
	}
	var rows []row

	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		containerID, _ := mfRepo.GetContainerUUID(model.ID(mf.ID()))
		modName := h.GetModuleName(h.FindModuleID(containerID))
		if inModule != "" && modName != inModule {
			continue
		}
		qualifiedName := modName + "." + mf.Name()
		folderPath := h.BuildFolderPath(containerID)
		returnType := strings.TrimSpace(mf.ReturnType())
		oc, ok := mf.ObjectCollection().(*genMf.MicroflowObjectCollection)
		if !ok {
			oc = nil
		}
		p := genFlowParameterElems(mf.ObjectCollection())
		activities := countGenFlowActivities(oc)
		complexity := calculateGenFlowComplexity(oc)
		rows = append(rows, row{
			qualifiedName: qualifiedName,
			module:        modName,
			name:          mf.Name(),
			excluded:      mf.Excluded(),
			folderPath:    folderPath,
			params:        len(p),
			activities:    activities,
			complexity:    complexity,
			returnType:    returnType,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	result := &TableResult{
		Columns: []string{"Qualified Name", "Module", "Name", "Excluded", "Folder", "Params", "Actions", "McCabe", "Returns"},
		Summary: fmt.Sprintf("(%d microflows)", len(rows)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.qualifiedName, r.module, r.name, r.excluded, r.folderPath, r.params, r.activities, r.complexity, r.returnType})
	}
	return writeResultTo(output, format, result)
}

// listNanoflowsFuture is the ExecContext-free version of listNanoflows.
func listNanoflowsFuture(
	ctx context.Context,
	output io.Writer,
	format OutputFormat,
	ml backend.ModuleLister,
	mr backend.MetadataReader,
	fm backend.FolderManager,
	nfRepo repos.NanoflowRepository,
	inModule string,
) error {
	h, err := NewContainerHierarchyFromRoles(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	nfs, err := nfRepo.ListAll()
	if err != nil {
		return mdlerrors.NewBackend("list nanoflows", err)
	}

	type row struct {
		qualifiedName string
		module        string
		name          string
		excluded      bool
		folderPath    string
		params        int
		activities    int
		complexity    int
		returnType    string
	}
	var rows []row

	for _, nf := range nfs {
		if nf == nil {
			continue
		}
		containerID, _ := nfRepo.GetContainerUUID(model.ID(nf.ID()))
		modName := h.GetModuleName(h.FindModuleID(containerID))
		if inModule != "" && modName != inModule {
			continue
		}
		qualifiedName := modName + "." + nf.Name()
		folderPath := h.BuildFolderPath(containerID)
		returnType := strings.TrimSpace(nf.ReturnType())
		oc, ok := nf.ObjectCollection().(*genMf.MicroflowObjectCollection)
		if !ok {
			oc = nil
		}
		p := genFlowParameterElems(nf.ObjectCollection())
		activities := countGenFlowActivities(oc)
		complexity := calculateGenFlowComplexity(oc)
		rows = append(rows, row{
			qualifiedName: qualifiedName,
			module:        modName,
			name:          nf.Name(),
			excluded:      nf.Excluded(),
			folderPath:    folderPath,
			params:        len(p),
			activities:    activities,
			complexity:    complexity,
			returnType:    returnType,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	result := &TableResult{
		Columns: []string{"Qualified Name", "Module", "Name", "Excluded", "Folder", "Params", "Actions", "McCabe", "Returns"},
		Summary: fmt.Sprintf("(%d nanoflows)", len(rows)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.qualifiedName, r.module, r.name, r.excluded, r.folderPath, r.params, r.activities, r.complexity, r.returnType})
	}
	return writeResultTo(output, format, result)
}

// listPagesFuture is the ExecContext-free version of listPagesGen.
func listPagesFuture(
	ctx context.Context,
	output io.Writer,
	format OutputFormat,
	ml backend.ModuleLister,
	mr backend.MetadataReader,
	fm backend.FolderManager,
	pageRepo repos.PageRepository,
	inModule string,
) error {
	h, err := NewContainerHierarchyFromRoles(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	pages, err := pageRepo.ListAll()
	if err != nil {
		return mdlerrors.NewBackend("list pages", err)
	}

	type row struct {
		qualifiedName string
		module        string
		name          string
		excluded      bool
		folderPath    string
		title         string
		url           string
		params        int
	}
	var rows []row

	for _, pg := range pages {
		if pg == nil {
			continue
		}
		containerID, _ := pageRepo.GetContainerUUID(model.ID(pg.ID()))
		modName := h.GetModuleName(h.FindModuleID(containerID))
		if inModule != "" && modName != inModule {
			continue
		}
		qualifiedName := modName + "." + pg.Name()
		folderPath := h.BuildFolderPath(containerID)
		title := pickPageTitleGen(pg)
		url := pg.Url()
		params := len(pg.ParametersItems())
		rows = append(rows, row{
			qualifiedName: qualifiedName,
			module:        modName,
			name:          pg.Name(),
			excluded:      pg.Excluded(),
			folderPath:    folderPath,
			title:         title,
			url:           url,
			params:        params,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	result := &TableResult{
		Columns: []string{"Qualified Name", "Module", "Name", "Excluded", "Folder", "Title", "url", "Params"},
		Summary: fmt.Sprintf("(%d pages)", len(rows)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.qualifiedName, r.module, r.name, r.excluded, r.folderPath, r.title, r.url, r.params})
	}
	return writeResultTo(output, format, result)
}

// listSnippetsFuture is the ExecContext-free version of listSnippetsGen.
func listSnippetsFuture(
	ctx context.Context,
	output io.Writer,
	format OutputFormat,
	ml backend.ModuleLister,
	mr backend.MetadataReader,
	fm backend.FolderManager,
	snpRepo repos.SnippetRepository,
	inModule string,
) error {
	h, err := NewContainerHierarchyFromRoles(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	snps, err := snpRepo.ListAll()
	if err != nil {
		return mdlerrors.NewBackend("list snippets", err)
	}

	type row struct {
		qualifiedName string
		module        string
		name          string
		folderPath    string
		params        int
	}
	var rows []row

	for _, s := range snps {
		if s == nil {
			continue
		}
		containerID, _ := snpRepo.GetContainerUUID(model.ID(s.ID()))
		modName := h.GetModuleName(h.FindModuleID(containerID))
		if inModule != "" && modName != inModule {
			continue
		}
		qualifiedName := modName + "." + s.Name()
		folderPath := h.BuildFolderPath(containerID)
		rows = append(rows, row{
			qualifiedName: qualifiedName,
			module:        modName,
			name:          s.Name(),
			folderPath:    folderPath,
			params:        len(s.ParametersItems()),
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	result := &TableResult{
		Columns: []string{"Qualified Name", "Module", "Name", "Folder", "Params"},
		Summary: fmt.Sprintf("(%d snippets)", len(rows)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.qualifiedName, r.module, r.name, r.folderPath, r.params})
	}
	return writeResultTo(output, format, result)
}

// listLayoutsFuture is the ExecContext-free version of listLayoutsGen.
func listLayoutsFuture(
	ctx context.Context,
	output io.Writer,
	format OutputFormat,
	ml backend.ModuleLister,
	mr backend.MetadataReader,
	fm backend.FolderManager,
	layRepo repos.LayoutRepository,
	inModule string,
) error {
	h, err := NewContainerHierarchyFromRoles(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	lays, err := layRepo.ListAll()
	if err != nil {
		return mdlerrors.NewBackend("list layouts", err)
	}

	type row struct {
		qualifiedName string
		module        string
		name          string
		folderPath    string
		layoutType    string
	}
	var rows []row

	for _, l := range lays {
		if l == nil {
			continue
		}
		containerID, _ := layRepo.GetContainerUUID(model.ID(l.ID()))
		modName := h.GetModuleName(h.FindModuleID(containerID))
		if inModule != "" && modName != inModule {
			continue
		}
		qualifiedName := modName + "." + l.Name()
		folderPath := h.BuildFolderPath(containerID)
		rows = append(rows, row{
			qualifiedName: qualifiedName,
			module:        modName,
			name:          l.Name(),
			folderPath:    folderPath,
			layoutType:    l.LayoutType(),
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	result := &TableResult{
		Columns: []string{"Qualified Name", "Module", "Name", "Folder", "Type"},
		Summary: fmt.Sprintf("(%d layouts)", len(rows)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.qualifiedName, r.module, r.name, r.folderPath, r.layoutType})
	}
	return writeResultTo(output, format, result)
}

// listJavaActionsFuture is the ExecContext-free version of listJavaActionsGen.
func listJavaActionsFuture(
	ctx context.Context,
	output io.Writer,
	format OutputFormat,
	ml backend.ModuleLister,
	mr backend.MetadataReader,
	fm backend.FolderManager,
	jaRepo repos.JavaActionRepository,
	inModule string,
) error {
	h, err := NewContainerHierarchyFromRoles(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	jas, err := jaRepo.ListAll()
	if err != nil {
		return mdlerrors.NewBackend("list java actions", err)
	}

	type row struct {
		qualifiedName string
		module        string
		name          string
		folderPath    string
	}
	var rows []row

	for _, ja := range jas {
		if ja == nil {
			continue
		}
		containerID, _ := jaRepo.GetContainerUUID(model.ID(ja.ID()))
		modName := h.GetModuleName(h.FindModuleID(containerID))
		if inModule != "" && modName != inModule {
			continue
		}
		qn := modName + "." + ja.Name()
		folder := h.BuildFolderPath(containerID)
		rows = append(rows, row{qn, modName, ja.Name(), folder})
	}

	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	result := &TableResult{
		Columns: []string{"Qualified Name", "Module", "Name", "Folder"},
		Summary: fmt.Sprintf("(%d java actions)", len(rows)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.qualifiedName, r.module, r.name, r.folderPath})
	}
	return writeResultTo(output, format, result)
}

// listJavaScriptActionsFuture is the ExecContext-free version of listJavaScriptActionsGen.
func listJavaScriptActionsFuture(
	ctx context.Context,
	output io.Writer,
	format OutputFormat,
	ml backend.ModuleLister,
	mr backend.MetadataReader,
	fm backend.FolderManager,
	jsaRepo repos.JavaScriptActionRepository,
	inModule string,
) error {
	h, err := NewContainerHierarchyFromRoles(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	jsas, err := jsaRepo.ListAll()
	if err != nil {
		return mdlerrors.NewBackend("list javascript actions", err)
	}

	type row struct {
		qualifiedName string
		module        string
		name          string
		platform      string
		folderPath    string
	}
	var rows []row

	for _, jsa := range jsas {
		if jsa == nil {
			continue
		}
		containerID, _ := jsaRepo.GetContainerUUID(model.ID(jsa.ID()))
		modName := h.GetModuleName(h.FindModuleID(containerID))
		if inModule != "" && modName != inModule {
			continue
		}
		qn := modName + "." + jsa.Name()
		folder := h.BuildFolderPath(containerID)
		platform := jsa.Platform()
		if platform == "" {
			platform = "All"
		}
		rows = append(rows, row{qn, modName, jsa.Name(), platform, folder})
	}

	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	result := &TableResult{
		Columns: []string{"Qualified Name", "Module", "Name", "Platform", "Folder"},
		Summary: fmt.Sprintf("(%d javascript actions)", len(rows)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.qualifiedName, r.module, r.name, r.platform, r.folderPath})
	}
	return writeResultTo(output, format, result)
}

// listWorkflowsFuture is the ExecContext-free version of listWorkflowsGen.
func listWorkflowsFuture(
	ctx context.Context,
	output io.Writer,
	format OutputFormat,
	cm backend.ConnectionManager,
	ml backend.ModuleLister,
	mr backend.MetadataReader,
	fm backend.FolderManager,
	wfRepo repos.WorkflowRepository,
	inModule string,
) error {
	if cm == nil || !cm.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	h, err := NewContainerHierarchyFromRoles(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	all, err := wfRepo.ListAll()
	if err != nil {
		return mdlerrors.NewBackend("list workflows", err)
	}

	type row struct {
		qualifiedName string
		module        string
		name          string
		activities    int
		userTasks     int
		decisions     int
		paramEntity   string
	}
	var rows []row

	for _, wf := range all {
		if wf == nil {
			continue
		}
		containerID, _ := wfRepo.GetContainerUUID(model.ID(wf.ID()))
		modName := h.GetModuleName(h.FindModuleID(containerID))
		if inModule != "" && modName != inModule {
			continue
		}
		qualifiedName := modName + "." + wf.Name()
		paramEntity := workflowParameterEntityGen(wf)
		acts, uts, decs := countWorkflowActivitiesGen(wf)
		rows = append(rows, row{qualifiedName, modName, wf.Name(), acts, uts, decs, paramEntity})
	}

	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	result := &TableResult{
		Columns: []string{"Qualified Name", "Activities", "User Tasks", "Decisions", "Parameter Entity"},
		Summary: fmt.Sprintf("(%d workflows)", len(rows)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.qualifiedName, r.activities, r.userTasks, r.decisions, r.paramEntity})
	}
	return writeResultTo(output, format, result)
}

// listBusinessEventServicesFuture is the ExecContext-free version of listBusinessEventServices.
func listBusinessEventServicesFuture(
	ctx context.Context,
	output io.Writer,
	format OutputFormat,
	cm backend.ConnectionManager,
	ml backend.ModuleLister,
	mr backend.MetadataReader,
	fm backend.FolderManager,
	beBackend backend.BusinessEventBackend,
	inModule string,
) error {
	if cm == nil || !cm.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	services, err := beBackend.ListBusinessEventServices()
	if err != nil {
		return mdlerrors.NewBackend("list business event services", err)
	}

	h, err := NewContainerHierarchyFromRoles(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	var filtered []*model.BusinessEventService
	for _, svc := range services {
		modID := h.FindModuleID(svc.ContainerID)
		moduleName := h.GetModuleName(modID)
		if inModule != "" && !strings.EqualFold(moduleName, inModule) {
			continue
		}
		filtered = append(filtered, svc)
	}

	if len(filtered) == 0 && format != FormatJSON {
		if inModule != "" {
			fmt.Fprintf(output, "No business event services found in module %s\n", inModule)
		} else {
			fmt.Fprintln(output, "No business event services found")
		}
		return nil
	}

	type row struct {
		module, qualifiedName, name            string
		msgCount, publishCount, subscribeCount int
	}
	var rows []row

	for _, svc := range filtered {
		modID := h.FindModuleID(svc.ContainerID)
		moduleName := h.GetModuleName(modID)
		qn := moduleName + "." + svc.Name
		r := row{module: moduleName, qualifiedName: qn, name: svc.Name}

		if svc.Definition != nil {
			for _, ch := range svc.Definition.Channels {
				r.msgCount += len(ch.Messages)
			}
		}
		for _, op := range svc.OperationImplementations {
			switch op.Operation {
			case "publish":
				r.publishCount++
			case "subscribe":
				r.subscribeCount++
			}
		}

		rows = append(rows, r)
	}

	result := &TableResult{
		Columns: []string{"Module", "QualifiedName", "Service", "Messages", "Publish", "Subscribe"},
		Summary: fmt.Sprintf("(%d business event services)", len(filtered)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.module, r.qualifiedName, r.name, r.msgCount, r.publishCount, r.subscribeCount})
	}
	return writeResultTo(output, format, result)
}

// listBusinessEventClientsFuture is the ExecContext-free stub version of listBusinessEventClients.
func listBusinessEventClientsFuture(ctx context.Context, output io.Writer) error {
	fmt.Fprintln(output, "Business event clients are not yet implemented.")
	return nil
}

// listBusinessEventsFuture is the ExecContext-free version of listBusinessEvents.
func listBusinessEventsFuture(
	ctx context.Context,
	output io.Writer,
	format OutputFormat,
	cm backend.ConnectionManager,
	ml backend.ModuleLister,
	mr backend.MetadataReader,
	fm backend.FolderManager,
	beBackend backend.BusinessEventBackend,
	inModule string,
) error {
	if cm == nil || !cm.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	services, err := beBackend.ListBusinessEventServices()
	if err != nil {
		return mdlerrors.NewBackend("list business event services", err)
	}

	h, err := NewContainerHierarchyFromRoles(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	type row struct {
		service, message, operation, entity string
		attrs                               int
	}
	var rows []row

	for _, svc := range services {
		modID := h.FindModuleID(svc.ContainerID)
		moduleName := h.GetModuleName(modID)
		if inModule != "" && !strings.EqualFold(moduleName, inModule) {
			continue
		}

		svcQN := moduleName + "." + svc.Name

		opMap := make(map[string]*model.ServiceOperation)
		for _, op := range svc.OperationImplementations {
			opMap[op.MessageName] = op
		}

		if svc.Definition != nil {
			for _, ch := range svc.Definition.Channels {
				for _, msg := range ch.Messages {
					opStr := ""
					entityStr := ""
					if op, ok := opMap[msg.MessageName]; ok {
						opStr = strings.ToUpper(op.Operation)
						entityStr = op.Entity
					}
					rows = append(rows, row{
						service:   svcQN,
						message:   msg.MessageName,
						operation: opStr,
						entity:    entityStr,
						attrs:     len(msg.Attributes),
					})
				}
			}
		}
	}

	if len(rows) == 0 && format != FormatJSON {
		if inModule != "" {
			fmt.Fprintf(output, "No business events found in module %s\n", inModule)
		} else {
			fmt.Fprintln(output, "No business events found")
		}
		return nil
	}

	result := &TableResult{
		Columns: []string{"Service", "Message", "Operation", "Entity", "Attributes"},
		Summary: fmt.Sprintf("(%d business events)", len(rows)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.service, r.message, r.operation, r.entity, r.attrs})
	}
	return writeResultTo(output, format, result)
}

// ────────────────────────────────────────────────────────────
// Phase 3d-1f: security show handlers migrated from *ExecContext
// ────────────────────────────────────────────────────────────

// listProjectSecurityFuture is the ExecContext-free version of listProjectSecurityGen.
func listProjectSecurityFuture(
	ctx context.Context,
	output io.Writer,
	format OutputFormat,
	sec repos.SecurityRepository,
) error {
	ps, err := sec.Get()
	if err != nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if ps == nil {
		return mdlerrors.NewBackend("read project security", fmt.Errorf("ProjectSecurity unit not found"))
	}

	var pp *genSec.PasswordPolicySettings
	if raw := ps.PasswordPolicySettings(); raw != nil {
		pp, _ = raw.(*genSec.PasswordPolicySettings)
	}

	if format == FormatJSON {
		result := &TableResult{Columns: []string{"Property", "Value"}}
		result.Rows = append(result.Rows,
			[]any{"SecurityLevel", securityLevelDisplay(ps.SecurityLevel())},
			[]any{"CheckSecurity", fmt.Sprintf("%v", ps.CheckSecurity())},
			[]any{"StrictMode", fmt.Sprintf("%v", ps.StrictMode())},
			[]any{"DemoUsersEnabled", fmt.Sprintf("%v", ps.EnableDemoUsers())},
			[]any{"GuestAccess", fmt.Sprintf("%v", ps.EnableGuestAccess())},
			[]any{"UserRoles", fmt.Sprintf("%d", len(ps.UserRolesItems()))},
			[]any{"DemoUsers", fmt.Sprintf("%d", len(ps.DemoUsersItems()))},
		)
		if ps.AdminUserName() != "" {
			result.Rows = append(result.Rows, []any{"AdminUser", ps.AdminUserName()})
		}
		if ps.GuestUserRoleName() != "" {
			result.Rows = append(result.Rows, []any{"GuestUserRole", ps.GuestUserRoleName()})
		}
		if pp != nil {
			result.Rows = append(result.Rows,
				[]any{"PasswordPolicy.MinimumLength", fmt.Sprintf("%d", pp.MinimumLength())},
				[]any{"PasswordPolicy.RequireDigit", fmt.Sprintf("%v", pp.RequireDigit())},
				[]any{"PasswordPolicy.RequireMixedCase", fmt.Sprintf("%v", pp.RequireMixedCase())},
				[]any{"PasswordPolicy.RequireSymbol", fmt.Sprintf("%v", pp.RequireSymbol())},
			)
		}
		return writeResultTo(output, format, result)
	}

	fmt.Fprintf(output, "Security Level: %s\n", securityLevelDisplay(ps.SecurityLevel()))
	fmt.Fprintf(output, "Check Security: %v\n", ps.CheckSecurity())
	fmt.Fprintf(output, "Strict Mode: %v\n", ps.StrictMode())
	fmt.Fprintf(output, "Demo Users Enabled: %v\n", ps.EnableDemoUsers())
	fmt.Fprintf(output, "Guest Access: %v\n", ps.EnableGuestAccess())
	if ps.AdminUserName() != "" {
		fmt.Fprintf(output, "Admin User: %s\n", ps.AdminUserName())
	}
	if ps.GuestUserRoleName() != "" {
		fmt.Fprintf(output, "Guest User Role: %s\n", ps.GuestUserRoleName())
	}
	fmt.Fprintf(output, "User Roles: %d\n", len(ps.UserRolesItems()))
	fmt.Fprintf(output, "Demo Users: %d\n", len(ps.DemoUsersItems()))

	if pp != nil {
		fmt.Fprintf(output, "\nPassword Policy:\n")
		fmt.Fprintf(output, "  Minimum Length: %d\n", pp.MinimumLength())
		fmt.Fprintf(output, "  Require Digit: %v\n", pp.RequireDigit())
		fmt.Fprintf(output, "  Require Mixed Case: %v\n", pp.RequireMixedCase())
		fmt.Fprintf(output, "  Require Symbol: %v\n", pp.RequireSymbol())
	}
	return nil
}

// listModuleRolesFuture is the ExecContext-free version of listModuleRolesGen.
func listModuleRolesFuture(
	ctx context.Context,
	output io.Writer,
	format OutputFormat,
	ml backend.ModuleLister,
	mr backend.MetadataReader,
	fm backend.FolderManager,
	sec repos.SecurityRepository,
	inModule string,
) error {
	h, err := NewContainerHierarchyFromRoles(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	modules, err := ml.ListModules()
	if err != nil {
		return mdlerrors.NewBackend("list modules", err)
	}

	result := &TableResult{Columns: []string{"Qualified Name", "Module", "Role", "Description"}}
	for _, mod := range modules {
		if inModule != "" && mod.Name != inModule {
			continue
		}
		ms, err := sec.GetModuleSecurity(mod.ID)
		if err != nil || ms == nil {
			continue
		}
		modName := h.GetModuleName(mod.ID)
		if modName == "" {
			continue
		}
		for _, mrItem := range ms.ModuleRolesItems() {
			typed, ok := mrItem.(*genSec.ModuleRole)
			if !ok {
				continue
			}
			qn := modName + "." + typed.Name()
			result.Rows = append(result.Rows, []any{qn, modName, typed.Name(), typed.Description()})
		}
	}
	result.Summary = fmt.Sprintf("(%d module roles)", len(result.Rows))
	return writeResultTo(output, format, result)
}

// listUserRolesFuture is the ExecContext-free version of listUserRolesGen.
func listUserRolesFuture(
	ctx context.Context,
	output io.Writer,
	format OutputFormat,
	sec repos.SecurityRepository,
) error {
	ps, err := sec.Get()
	if err != nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if ps == nil {
		return mdlerrors.NewBackend("read project security", fmt.Errorf("ProjectSecurity not found"))
	}

	result := &TableResult{Columns: []string{"Name", "Module Roles", "Manage All", "Check Security"}}
	for _, ur := range ps.UserRolesItems() {
		typed, ok := ur.(*genSec.UserRole)
		if !ok {
			continue
		}
		ma := "No"
		if typed.ManageAllRoles() {
			ma = "Yes"
		}
		cs := "No"
		if typed.CheckSecurity() {
			cs = "Yes"
		}
		result.Rows = append(result.Rows, []any{typed.Name(), len(typed.ModuleRolesQualifiedNames()), ma, cs})
	}
	result.Summary = fmt.Sprintf("(%d user roles)", len(result.Rows))
	return writeResultTo(output, format, result)
}

// listDemoUsersFuture is the ExecContext-free version of listDemoUsersGen.
func listDemoUsersFuture(
	ctx context.Context,
	output io.Writer,
	format OutputFormat,
	sec repos.SecurityRepository,
) error {
	ps, err := sec.Get()
	if err != nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if ps == nil {
		return mdlerrors.NewBackend("read project security", fmt.Errorf("ProjectSecurity not found"))
	}

	if !ps.EnableDemoUsers() {
		if format != FormatJSON {
			fmt.Fprintln(output, "Demo users are disabled.")
			fmt.Fprintln(output, "Enable with: alter project security demo users on;")
			return nil
		}
		return writeResultTo(output, format, &TableResult{Columns: []string{"User Name", "User Roles"}})
	}

	result := &TableResult{Columns: []string{"User Name", "User Roles"}}
	for _, du := range ps.DemoUsersItems() {
		typed, ok := du.(*genSec.DemoUser)
		if !ok {
			continue
		}
		rolesStr := strings.Join(typed.UserRolesQualifiedNames(), ", ")
		result.Rows = append(result.Rows, []any{typed.UserName(), rolesStr})
	}
	result.Summary = fmt.Sprintf("(%d demo users)", len(result.Rows))
	return writeResultTo(output, format, result)
}

// listAccessOnEntityFuture is the ExecContext-free version of listAccessOnEntityGen.
func listAccessOnEntityFuture(
	ctx context.Context,
	output io.Writer,
	format OutputFormat,
	ml backend.ModuleLister,
	dmr repos.DomainModelRepository,
	name *ast.QualifiedName,
) error {
	if name == nil {
		return mdlerrors.NewValidation("entity name required")
	}

	pairs, err := dmr.ListAllWithContainerID()
	if err != nil {
		return mdlerrors.NewBackend("list domain models", err)
	}

	mods, err := ml.ListModules()
	if err != nil {
		return mdlerrors.NewBackend("list modules", err)
	}
	moduleNames := make(map[model.ID]string, len(mods))
	for _, m := range mods {
		moduleNames[m.ID] = m.Name
	}

	var entity *genDm.Entity
	for _, p := range pairs {
		if p.DM == nil {
			continue
		}
		modName := moduleNames[p.ContainerID]
		if modName != name.Module {
			continue
		}
		for _, e := range p.DM.EntitiesItems() {
			ent, ok := e.(*genDm.Entity)
			if !ok {
				continue
			}
			if ent.Name() == name.Name {
				entity = ent
				break
			}
		}
	}
	if entity == nil {
		return mdlerrors.NewNotFound("entity", name.String())
	}

	attrNames := make(map[string]string)
	for _, a := range entity.AttributesItems() {
		attr, ok := a.(*genDm.Attribute)
		if !ok {
			continue
		}
		attrNames[string(attr.ID())] = attr.Name()
		attrNames[attr.Name()] = attr.Name()
	}

	if format == FormatJSON {
		result := &TableResult{
			Columns: []string{"Rule", "Roles", "Rights", "DefaultMemberAccess", "MemberAccess", "XPath"},
		}
		ruleNum := 0
		for _, r := range entity.AccessRulesItems() {
			rule, ok := r.(*genDm.AccessRule)
			if !ok {
				continue
			}
			ruleNum++
			var memberParts []string
			for _, m := range rule.MemberAccessesItems() {
				ma, ok := m.(*genDm.MemberAccess)
				if !ok {
					continue
				}
				memberParts = append(memberParts, memberAccessLocalName(ma, attrNames)+":"+ma.AccessRights())
			}
			result.Rows = append(result.Rows, []any{
				ruleNum,
				strings.Join(entityRuleRoleStringsGen(rule), ", "),
				strings.ToLower(strings.Join(entityRuleRightStringsGen(rule), ", ")),
				rule.DefaultMemberAccessRights(),
				strings.Join(memberParts, ", "),
				rule.XPathConstraint(),
			})
		}
		return writeResultTo(output, format, result)
	}

	if len(entity.AccessRulesItems()) == 0 {
		fmt.Fprintf(output, "No access rules on %s\n", name)
		return nil
	}

	fmt.Fprintf(output, "Access rules for %s.%s:\n\n", name.Module, name.Name)

	ruleNum := 0
	for _, r := range entity.AccessRulesItems() {
		rule, ok := r.(*genDm.AccessRule)
		if !ok {
			continue
		}
		ruleNum++
		var rights []string
		for _, right := range entityRuleRightStringsGen(rule) {
			rights = append(rights, strings.ToLower(right))
		}
		fmt.Fprintf(output, "Rule %d: %s\n", ruleNum, strings.Join(entityRuleRoleStringsGen(rule), ", "))
		fmt.Fprintf(output, "  Rights: %s\n", strings.Join(rights, ", "))

		if rule.DefaultMemberAccessRights() != "" {
			fmt.Fprintf(output, "  Default member access: %s\n", rule.DefaultMemberAccessRights())
		}

		for _, m := range rule.MemberAccessesItems() {
			ma, ok := m.(*genDm.MemberAccess)
			if !ok {
				continue
			}
			fmt.Fprintf(output, "  %s: %s\n", memberAccessLocalName(ma, attrNames), ma.AccessRights())
		}

		if rule.XPathConstraint() != "" {
			fmt.Fprintf(output, "  where '%s'\n", rule.XPathConstraint())
		}
		fmt.Fprintln(output)
	}
	return nil
}

// listAccessOnMicroflowFuture is the ExecContext-free version of listAccessOnMicroflowGen.
func listAccessOnMicroflowFuture(
	ctx context.Context,
	output io.Writer,
	format OutputFormat,
	ml backend.ModuleLister,
	mr backend.MetadataReader,
	fm backend.FolderManager,
	mfRepo repos.MicroflowRepository,
	name *ast.QualifiedName,
) error {
	if name == nil {
		return mdlerrors.NewValidation("microflow name required")
	}

	h, err := NewContainerHierarchyFromRoles(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	mfs, err := mfRepo.ListAll()
	if err != nil {
		return mdlerrors.NewBackend("list microflows", err)
	}

	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		containerID, _ := mfRepo.GetContainerUUID(model.ID(mf.ID()))
		modName := h.GetModuleName(h.FindModuleID(containerID))
		if modName != name.Module || mf.Name() != name.Name {
			continue
		}
		roles := mf.AllowedModuleRolesQualifiedNames()
		if format == FormatJSON {
			result := &TableResult{Columns: []string{"Module", "Role"}}
			for _, role := range roles {
				mod, r := splitRoleQualifiedName(role)
				result.Rows = append(result.Rows, []any{mod, r})
			}
			return writeResultTo(output, format, result)
		}
		if len(roles) == 0 {
			fmt.Fprintf(output, "No module roles granted execute access on %s.%s\n", modName, mf.Name())
			return nil
		}
		fmt.Fprintf(output, "Allowed module roles for %s.%s:\n", modName, mf.Name())
		for _, role := range roles {
			fmt.Fprintf(output, "  %s\n", role)
		}
		return nil
	}

	return mdlerrors.NewNotFound("microflow", name.String())
}

// listAccessOnPageFuture is the ExecContext-free version of listAccessOnPageGen.
func listAccessOnPageFuture(
	ctx context.Context,
	output io.Writer,
	format OutputFormat,
	ml backend.ModuleLister,
	mr backend.MetadataReader,
	fm backend.FolderManager,
	pgRepo repos.PageRepository,
	name *ast.QualifiedName,
) error {
	if name == nil {
		return mdlerrors.NewValidation("page name required")
	}

	h, err := NewContainerHierarchyFromRoles(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	pages, err := pgRepo.ListAll()
	if err != nil {
		return mdlerrors.NewBackend("list pages", err)
	}

	for _, pg := range pages {
		if pg == nil {
			continue
		}
		containerID, _ := pgRepo.GetContainerUUID(model.ID(pg.ID()))
		modName := h.GetModuleName(h.FindModuleID(containerID))
		if modName == name.Module && pg.Name() == name.Name {
			allowed := pg.AllowedRolesQualifiedNames()
			if format == FormatJSON {
				result := &TableResult{Columns: []string{"Module", "Role"}}
				for _, role := range allowed {
					parts := strings.SplitN(role, ".", 2)
					mod, r := "", role
					if len(parts) == 2 {
						mod, r = parts[0], parts[1]
					}
					result.Rows = append(result.Rows, []any{mod, r})
				}
				return writeResultTo(output, format, result)
			}
			if len(allowed) == 0 {
				fmt.Fprintf(output, "No module roles granted view access on %s.%s\n", modName, pg.Name())
				return nil
			}
			fmt.Fprintf(output, "Allowed module roles for %s.%s:\n", modName, pg.Name())
			for _, role := range allowed {
				fmt.Fprintf(output, "  %s\n", role)
			}
			return nil
		}
	}

	return mdlerrors.NewNotFound("page", name.String())
}

// listAccessOnWorkflowFuture is the ExecContext-free version of listAccessOnWorkflow.
func listAccessOnWorkflowFuture(ctx context.Context, name *ast.QualifiedName) error {
	return mdlerrors.NewUnsupported("show access on workflow is not supported: Mendix workflows do not have document-level AllowedModuleRoles (unlike microflows and pages). Workflow access is controlled through the microflow that triggers the workflow and UserTask targeting")
}

// listAccessOnNanoflowFuture is the ExecContext-free version of listAccessOnNanoflowGen.
func listAccessOnNanoflowFuture(
	ctx context.Context,
	output io.Writer,
	format OutputFormat,
	ml backend.ModuleLister,
	mr backend.MetadataReader,
	fm backend.FolderManager,
	nfRepo repos.NanoflowRepository,
	name *ast.QualifiedName,
) error {
	if name == nil {
		return mdlerrors.NewValidation("nanoflow name required")
	}

	h, err := NewContainerHierarchyFromRoles(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	nfs, err := nfRepo.ListAll()
	if err != nil {
		return mdlerrors.NewBackend("list nanoflows", err)
	}

	for _, nf := range nfs {
		if nf == nil {
			continue
		}
		containerID, _ := nfRepo.GetContainerUUID(model.ID(nf.ID()))
		modName := h.GetModuleName(h.FindModuleID(containerID))
		if modName != name.Module || nf.Name() != name.Name {
			continue
		}
		roles := nf.AllowedModuleRolesQualifiedNames()
		if format == FormatJSON {
			result := &TableResult{Columns: []string{"Module", "Role"}}
			for _, role := range roles {
				mod, r := splitRoleQualifiedName(role)
				result.Rows = append(result.Rows, []any{mod, r})
			}
			return writeResultTo(output, format, result)
		}
		if len(roles) == 0 {
			fmt.Fprintf(output, "No module roles granted execute access on %s.%s\n", modName, nf.Name())
			return nil
		}
		fmt.Fprintf(output, "Allowed module roles for %s.%s:\n", modName, nf.Name())
		for _, role := range roles {
			fmt.Fprintf(output, "  %s\n", role)
		}
		return nil
	}

	return mdlerrors.NewNotFound("nanoflow", name.String())
}

// listSecurityMatrixFuture is the ExecContext-free version of listSecurityMatrixGen.
func listSecurityMatrixFuture(
	ctx context.Context,
	output io.Writer,
	format OutputFormat,
	ml backend.ModuleLister,
	mr backend.MetadataReader,
	fm backend.FolderManager,
	sec repos.SecurityRepository,
	dmr repos.DomainModelRepository,
	mfRepo repos.MicroflowRepository,
	pgRepo repos.PageRepository,
	inModule string,
) error {
	if format == FormatJSON {
		return listSecurityMatrixJSONFuture(ctx, output, format, ml, mr, fm, sec, dmr, mfRepo, pgRepo, inModule)
	}

	h, err := NewContainerHierarchyFromRoles(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	modules, err := ml.ListModules()
	if err != nil {
		return mdlerrors.NewBackend("list modules", err)
	}

	type moduleRoleInfo struct {
		moduleName string
		roleName   string
	}
	var roles []moduleRoleInfo
	for _, mod := range modules {
		if inModule != "" && mod.Name != inModule {
			continue
		}
		ms, err := sec.GetModuleSecurity(mod.ID)
		if err != nil || ms == nil {
			continue
		}
		for _, mrItem := range ms.ModuleRolesItems() {
			mr2, ok := mrItem.(*genSec.ModuleRole)
			if !ok {
				continue
			}
			roles = append(roles, moduleRoleInfo{mod.Name, mr2.Name()})
		}
	}

	if len(roles) == 0 {
		if inModule != "" {
			fmt.Fprintf(output, "No module roles found in %s\n", inModule)
		} else {
			fmt.Fprintln(output, "No module roles found")
		}
		return nil
	}

	pairs, err := dmr.ListAllWithContainerID()
	if err != nil {
		return mdlerrors.NewBackend("list domain models", err)
	}
	moduleIDs := make(map[model.ID]bool, len(modules))
	for _, m := range modules {
		moduleIDs[m.ID] = true
	}

	fmt.Fprintf(output, "Security Matrix")
	if inModule != "" {
		fmt.Fprintf(output, " for %s", inModule)
	}
	fmt.Fprintln(output, ":")
	fmt.Fprintln(output)

	// Entities section
	fmt.Fprintln(output, "## Entity Access")
	fmt.Fprintln(output)

	entityFound := false
	for _, p := range pairs {
		if p.DM == nil || !moduleIDs[p.ContainerID] {
			continue
		}
		modID := h.FindModuleID(p.ContainerID)
		modName := h.GetModuleName(modID)
		if inModule != "" && modName != inModule {
			continue
		}
		for _, e := range p.DM.EntitiesItems() {
			entity, ok := e.(*genDm.Entity)
			if !ok {
				continue
			}
			rules := entity.AccessRulesItems()
			if len(rules) == 0 {
				continue
			}
			entityFound = true
			fmt.Fprintf(output, "### %s.%s\n", modName, entity.Name())
			for _, r := range rules {
				rule, ok := r.(*genDm.AccessRule)
				if !ok {
					continue
				}
				roleStrs := entityRuleRoleStringsGen(rule)
				rights := entityRuleRightStringsGen(rule)
				fmt.Fprintf(output, "  %s: %s\n", strings.Join(roleStrs, ", "), strings.Join(rights, ""))
			}
			fmt.Fprintln(output)
		}
	}
	if !entityFound {
		fmt.Fprintln(output, "(no entity access rules configured)")
		fmt.Fprintln(output)
	}

	// Microflow section
	fmt.Fprintln(output, "## Microflow Access")
	fmt.Fprintln(output)

	mfs, err := mfRepo.ListAll()
	if err != nil {
		return mdlerrors.NewBackend("list microflows", err)
	}

	mfFound := false
	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		roleStrs := mf.AllowedModuleRolesQualifiedNames()
		if len(roleStrs) == 0 {
			continue
		}
		containerID, _ := mfRepo.GetContainerUUID(model.ID(mf.ID()))
		modName := h.GetModuleName(h.FindModuleID(containerID))
		if inModule != "" && modName != inModule {
			continue
		}
		mfFound = true
		fmt.Fprintf(output, "  %s.%s: %s\n", modName, mf.Name(), strings.Join(roleStrs, ", "))
	}
	if !mfFound {
		fmt.Fprintln(output, "(no microflow access rules configured)")
	}
	fmt.Fprintln(output)

	// Page section
	fmt.Fprintln(output, "## Page Access")
	fmt.Fprintln(output)

	pages, err := pgRepo.ListAll()
	if err != nil {
		return mdlerrors.NewBackend("list pages", err)
	}

	pgFound := false
	for _, pg := range pages {
		if pg == nil {
			continue
		}
		roles2 := pg.AllowedRolesQualifiedNames()
		if len(roles2) == 0 {
			continue
		}
		containerID2, _ := pgRepo.GetContainerUUID(model.ID(pg.ID()))
		modName2 := h.GetModuleName(h.FindModuleID(containerID2))
		if inModule != "" && modName2 != inModule {
			continue
		}
		pgFound = true
		fmt.Fprintf(output, "  %s.%s: %s\n", modName2, pg.Name(), strings.Join(roles2, ", "))
	}
	if !pgFound {
		fmt.Fprintln(output, "(no page access rules configured)")
	}
	fmt.Fprintln(output)

	// Workflow section
	fmt.Fprintln(output, "## Workflow Access")
	fmt.Fprintln(output)
	fmt.Fprintln(output, "(workflow access is controlled through triggering microflows and UserTask targeting, not document-level roles)")
	fmt.Fprintln(output)

	return nil
}

// listSecurityMatrixJSONFuture is the JSON helper for listSecurityMatrixFuture.
func listSecurityMatrixJSONFuture(
	ctx context.Context,
	output io.Writer,
	format OutputFormat,
	ml backend.ModuleLister,
	mr backend.MetadataReader,
	fm backend.FolderManager,
	sec repos.SecurityRepository,
	dmr repos.DomainModelRepository,
	mfRepo repos.MicroflowRepository,
	pgRepo repos.PageRepository,
	inModule string,
) error {
	h, err := NewContainerHierarchyFromRoles(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	tr := &TableResult{
		Columns: []string{"ObjectType", "QualifiedName", "Roles", "Rights"},
	}

	modules, err := ml.ListModules()
	if err != nil {
		return mdlerrors.NewBackend("list modules", err)
	}
	moduleIDs := make(map[model.ID]bool, len(modules))
	for _, m := range modules {
		moduleIDs[m.ID] = true
	}

	// Entities
	pairs, _ := dmr.ListAllWithContainerID()
	for _, p := range pairs {
		if p.DM == nil || !moduleIDs[p.ContainerID] {
			continue
		}
		modID := h.FindModuleID(p.ContainerID)
		modName := h.GetModuleName(modID)
		if inModule != "" && modName != inModule {
			continue
		}
		for _, e := range p.DM.EntitiesItems() {
			entity, ok := e.(*genDm.Entity)
			if !ok {
				continue
			}
			for _, r := range entity.AccessRulesItems() {
				rule, ok := r.(*genDm.AccessRule)
				if !ok {
					continue
				}
				roleStrs := entityRuleRoleStringsGen(rule)
				rights := entityRuleRightStringsGen(rule)
				tr.Rows = append(tr.Rows, []any{
					"Entity",
					modName + "." + entity.Name(),
					strings.Join(roleStrs, ", "),
					strings.Join(rights, ""),
				})
			}
		}
	}

	// Microflows
	mfs, _ := mfRepo.ListAll()
	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		roleStrs := mf.AllowedModuleRolesQualifiedNames()
		if len(roleStrs) == 0 {
			continue
		}
		containerID, _ := mfRepo.GetContainerUUID(model.ID(mf.ID()))
		modName := h.GetModuleName(h.FindModuleID(containerID))
		if inModule != "" && modName != inModule {
			continue
		}
		tr.Rows = append(tr.Rows, []any{
			"Microflow",
			modName + "." + mf.Name(),
			strings.Join(roleStrs, ", "),
			"X",
		})
	}

	// Pages
	pages2, _ := pgRepo.ListAll()
	for _, pg := range pages2 {
		if pg == nil {
			continue
		}
		roles3 := pg.AllowedRolesQualifiedNames()
		if len(roles3) == 0 {
			continue
		}
		containerID2, _ := pgRepo.GetContainerUUID(model.ID(pg.ID()))
		modName2 := h.GetModuleName(h.FindModuleID(containerID2))
		if inModule != "" && modName2 != inModule {
			continue
		}
		tr.Rows = append(tr.Rows, []any{
			"Page",
			modName2 + "." + pg.Name(),
			strings.Join(roles3, ", "),
			"X",
		})
	}

	return writeResultTo(output, format, tr)
}

// ────────────────────────────────────────────────────────────
// Phase 3d-1g: navigation/settings/structure show handlers
// ────────────────────────────────────────────────────────────

// listNavigationFuture is the ExecContext-free version of listNavigation.
func listNavigationFuture(ctx context.Context, output io.Writer, nr backend.NavigationReader) error {
	nav, err := nr.GetNavigation()
	if err != nil {
		return mdlerrors.NewBackend("get navigation", err)
	}

	if len(nav.Profiles) == 0 {
		fmt.Fprintln(output, "No navigation profiles found.")
		return nil
	}

	type row struct {
		name      string
		kind      string
		homePage  string
		loginPage string
		menuItems int
		roleHomes int
	}
	var rows []row

	for _, p := range nav.Profiles {
		homePage := ""
		if p.HomePage != nil {
			if p.HomePage.Page != "" {
				homePage = p.HomePage.Page
			} else if p.HomePage.Microflow != "" {
				homePage = "MF:" + p.HomePage.Microflow
			}
		}

		loginPage := p.LoginPage
		if loginPage == "" {
			loginPage = "-"
		}

		menuCount := countMenuItems(p.MenuItems)

		kind := p.Kind
		if p.IsNative {
			kind += " (native)"
		}

		rows = append(rows, row{p.Name, kind, homePage, loginPage, menuCount, len(p.RoleBasedHomePages)})
	}

	result := &TableResult{
		Columns: []string{"Profile", "Kind", "HomePage", "LoginPage", "MenuItems", "RoleHomes"},
		Summary: fmt.Sprintf("(%d navigation profiles)", len(rows)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.name, r.kind, r.homePage, r.loginPage, r.menuItems, r.roleHomes})
	}
	return writeResultTo(output, FormatTable, result)
}

// listNavigationMenuFuture is the ExecContext-free version of listNavigationMenu.
func listNavigationMenuFuture(ctx context.Context, output io.Writer, nr backend.NavigationReader, profileName *ast.QualifiedName) error {
	nav, err := nr.GetNavigation()
	if err != nil {
		return mdlerrors.NewBackend("get navigation", err)
	}

	for _, p := range nav.Profiles {
		if profileName != nil && !strings.EqualFold(p.Name, profileName.Name) {
			continue
		}

		fmt.Fprintf(output, "-- Navigation Menu: %s (%s)\n", p.Name, p.Kind)
		if len(p.MenuItems) == 0 {
			fmt.Fprintln(output, "  (no menu items)")
		} else {
			printMenuTree(output, p.MenuItems, 0)
		}
		fmt.Fprintln(output)
	}

	return nil
}

// listNavigationHomesFuture is the ExecContext-free version of listNavigationHomes.
func listNavigationHomesFuture(ctx context.Context, output io.Writer, nr backend.NavigationReader) error {
	nav, err := nr.GetNavigation()
	if err != nil {
		return mdlerrors.NewBackend("get navigation", err)
	}

	for _, p := range nav.Profiles {
		fmt.Fprintf(output, "-- Profile: %s (%s)\n", p.Name, p.Kind)

		if p.HomePage != nil {
			if p.HomePage.Page != "" {
				fmt.Fprintf(output, "  Default Home: page %s\n", p.HomePage.Page)
			} else if p.HomePage.Microflow != "" {
				fmt.Fprintf(output, "  Default Home: microflow %s\n", p.HomePage.Microflow)
			}
		} else {
			fmt.Fprintln(output, "  Default Home: (none)")
		}

		if len(p.RoleBasedHomePages) > 0 {
			fmt.Fprintln(output, "  Role-Based Homes:")
			for _, rh := range p.RoleBasedHomePages {
				target := ""
				if rh.Page != "" {
					target = "page " + rh.Page
				} else if rh.Microflow != "" {
					target = "microflow " + rh.Microflow
				}
				fmt.Fprintf(output, "    %s -> %s\n", rh.UserRole, target)
			}
		}

		fmt.Fprintln(output)
	}

	return nil
}

// listSettingsFuture is the ExecContext-free version of listSettings.
func listSettingsFuture(ctx context.Context, output io.Writer, format OutputFormat, sr backend.SettingsReader) error {
	ps, err := sr.GetProjectSettings()
	if err != nil {
		return mdlerrors.NewBackend("read project settings", err)
	}

	tr := &TableResult{
		Columns: []string{"Section", "Key Values"},
	}

	if ps.Model != nil {
		ms := ps.Model
		values := []string{}
		if ms.AfterStartupMicroflow != "" {
			values = append(values, "AfterStartup: "+ms.AfterStartupMicroflow)
		}
		values = append(values, "Hash: "+ms.HashAlgorithm)
		values = append(values, "Java: "+ms.JavaVersion)
		tr.Rows = append(tr.Rows, []any{"Model Settings", strings.Join(values, ", ")})
	}

	if ps.Configuration != nil {
		for _, cfg := range ps.Configuration.Configurations {
			values := []string{}
			values = append(values, cfg.DatabaseType)
			values = append(values, cfg.DatabaseUrl)
			values = append(values, "db="+cfg.DatabaseName)
			values = append(values, fmt.Sprintf("http=%d", cfg.HttpPortNumber))
			if len(cfg.ConstantValues) > 0 {
				values = append(values, fmt.Sprintf("%d constants", len(cfg.ConstantValues)))
			}
			tr.Rows = append(tr.Rows, []any{
				fmt.Sprintf("Configuration '%s'", cfg.Name),
				strings.Join(values, ", "),
			})
		}
	}

	if ps.Language != nil {
		tr.Rows = append(tr.Rows, []any{"Language Settings", "Default: " + ps.Language.DefaultLanguageCode})
	}

	if ps.Workflows != nil {
		ws := ps.Workflows
		values := []string{}
		if ws.UserEntity != "" {
			values = append(values, "UserEntity: "+ws.UserEntity)
		}
		if ws.DefaultTaskParallelism > 0 {
			values = append(values, fmt.Sprintf("TaskParallelism: %d", ws.DefaultTaskParallelism))
		}
		tr.Rows = append(tr.Rows, []any{"Workflow Settings", strings.Join(values, ", ")})
	}

	if ps.Convention != nil {
		tr.Rows = append(tr.Rows, []any{"Convention Settings", "AssocStorage: " + ps.Convention.DefaultAssociationStorage})
	}

	if ps.WebUI != nil {
		tr.Rows = append(tr.Rows, []any{"Web UI Settings", "OptimizedClient: " + ps.WebUI.UseOptimizedClient})
	}

	return writeResultTo(output, format, tr)
}

// listLanguagesFuture is the ExecContext-free version of listLanguages.
func listLanguagesFuture(ctx context.Context, output io.Writer, format OutputFormat, sr backend.SettingsReader) error {
	ps, err := sr.GetProjectSettings()
	if err != nil {
		return mdlerrors.NewBackend("read project settings", err)
	}
	if ps == nil || ps.Language == nil || len(ps.Language.Languages) == 0 {
		fmt.Fprintln(output, "No project languages configured.")
		return nil
	}

	defaultCode := ps.Language.DefaultLanguageCode
	tr := &TableResult{
		Columns: []string{"Code", "Language", "Default", "CheckCompleteness"},
		Summary: fmt.Sprintf("(%d languages)", len(ps.Language.Languages)),
	}
	for _, l := range ps.Language.Languages {
		name := supportedLanguages[l.Code]
		if name == "" {
			name = "(unknown)"
		}
		def := ""
		if l.Code == defaultCode {
			def = "yes"
		}
		cc := ""
		if l.CheckCompleteness {
			cc = "yes"
		}
		tr.Rows = append(tr.Rows, []any{l.Code, name, def, cc})
	}
	return writeResultTo(output, format, tr)
}

// listSupportedLanguagesFuture is the ExecContext-free version of listSupportedLanguages.
func listSupportedLanguagesFuture(ctx context.Context, output io.Writer, format OutputFormat) error {
	tr := &TableResult{
		Columns: []string{"Code", "Language"},
		Summary: fmt.Sprintf("(%d supported languages)", len(supportedLanguages)),
	}
	codes := make([]string, 0, len(supportedLanguages))
	for code := range supportedLanguages {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	for _, code := range codes {
		tr.Rows = append(tr.Rows, []any{code, supportedLanguages[code]})
	}
	return writeResultTo(output, format, tr)
}

// execShowStructureGenFuture is the ExecContext-free version of execShowStructureGen.
func execShowStructureGenFuture(ctx context.Context, output io.Writer, format OutputFormat, s *ast.ShowStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}
	deps.Output = output
	deps.Format = format
	return execShowStructureGenFn(ctx, s, deps)
}

// ────────────────────────────────────────────────────────────
// Phase 3d-2a: connection/session handlers migrated from *ExecContext
// ────────────────────────────────────────────────────────────

// execConnectFuture is the ExecContext-free version of execConnect.
// Bridge pattern: constructs a temporary ExecContext from executor state,
// delegates to execConnect, then syncs mutated fields back.
func execConnectFuture(ctx context.Context, s *ast.ConnectStmt, ex *Executor) error {
	tmpCtx := &ExecContext{
		Context: ctx,
		ExecIO: ExecIO{
			Output:       ex.output,
			StatusOutput: ex.statusOutput,
			Format:       ex.format,
			Quiet:        ex.quiet,
		},
		ExecConnection: ExecConnection{
			BackendFactory: ex.backendFactory,
			MprPath:        ex.mprPath,
			Graph:          ex.graphCatalog,
		},
		Logger: ex.logger,
	}
	if ex.backend != nil {
		tmpCtx.Backend = ex.backend
		tmpCtx.ConnectionManager = ex.backend
	}
	if ex.cache != nil {
		tmpCtx.Cache = ex.cache
	}
	err := execConnect(tmpCtx, s)
	// Inline syncBack for Connect: propagate Executor state
	if tmpCtx.Backend != nil {
		ex.backend = tmpCtx.Backend
	}
	ex.mprPath = tmpCtx.MprPath
	ex.cache = tmpCtx.Cache
	if tmpCtx.Graph != nil {
		ex.graphCatalog = tmpCtx.Graph
	}
	return err
}

// execDisconnectFuture is the ExecContext-free version of execDisconnect.
func execDisconnectFuture(ctx context.Context, ex *Executor) error {
	if ex.backend == nil || !ex.backend.IsConnected() {
		fmt.Fprintln(ex.output, "Not connected")
		return nil
	}
	tmpCtx := &ExecContext{
		Context:           ctx,
		Backend:           ex.backend,
		ConnectionManager: ex.backend,
		ExecIO:            ExecIO{Output: ex.output},
		ExecConnection:    ExecConnection{MprPath: ex.mprPath},
		ExecCallbacks: ExecCallbacks{
			FinalizeFn: ex.finalizeProgramExecution,
		},
		Logger: ex.logger,
	}
	if ex.cache != nil {
		tmpCtx.Cache = ex.cache
	}
	err := execDisconnect(tmpCtx)
	// Inline syncBack for Disconnect
	ex.mprPath = ""
	ex.cache = nil
	ex.backend = nil
	return err
}

// execStatusFuture is the ExecContext-free version of execStatus.
func execStatusFuture(ctx context.Context, output io.Writer, cm backend.ConnectionManager, ml backend.ModuleLister, mprPath string) error {
	if cm == nil || !cm.IsConnected() {
		fmt.Fprintln(output, "Status: Not connected")
		return nil
	}
	pv := cm.ProjectVersion()
	fmt.Fprintf(output, "Status: Connected\n")
	fmt.Fprintf(output, "Project: %s\n", mprPath)
	fmt.Fprintf(output, "Mendix Version: %s\n", pv.ProductVersion)
	fmt.Fprintf(output, "MPR Format: v%d\n", pv.FormatVersion)
	modules, err := ml.ListModules()
	if err == nil {
		fmt.Fprintf(output, "Modules: %d\n", len(modules))
	}
	return nil
}

// execSetFuture is the ExecContext-free version of execSet.
// Format is updated via pointer so the caller's format changes persist.
func execSetFuture(ctx context.Context, s *ast.SetStmt, output io.Writer, format *OutputFormat) error {
	deps := &HandlerDeps{Output: output, Format: *format}
	err := execSetFn(ctx, s, deps)
	*format = deps.Format
	return err
}

// execUpdateFuture is the ExecContext-free version of execUpdate.
func execUpdateFuture(ctx context.Context, deps *HandlerDeps, ex *Executor) error {
	return execUpdateFn(ctx, deps, ex)
}

// execRefreshFuture is the ExecContext-free version of execRefresh.
func execRefreshFuture(ctx context.Context, deps *HandlerDeps, ex *Executor) error {
	return execRefreshFn(ctx, deps, ex)
}

// execExecuteScriptFuture is the ExecContext-free version of execExecuteScript.
func execExecuteScriptFuture(ctx context.Context, s *ast.ExecuteScriptStmt, deps *HandlerDeps, ex *Executor) error {
	tmpCtx := &ExecContext{
		Context:           ctx,
		Backend:           deps.Backend,
		ConnectionManager: deps.ConnectionManager,
		ExecIO:            ExecIO{Output: deps.Output},
		ExecConnection: ExecConnection{
			BackendFactory: ex.backendFactory,
		},
		ExecCallbacks: ExecCallbacks{
			ExecuteFn:        ex.Execute,
			ExecuteProgramFn: ex.ExecuteProgram,
		},
		ScriptTransactionManager: deps.Backend,
		Logger:                   deps.Logger,
	}
	err := execExecuteScript(tmpCtx, s)
	// Inline syncBack for ExecuteScript
	if tmpCtx.Backend != nil {
		ex.backend = tmpCtx.Backend
	}
	ex.mprPath = tmpCtx.MprPath
	ex.cache = tmpCtx.Cache
	ex.format = tmpCtx.Format
	if tmpCtx.Graph != nil {
		ex.graphCatalog = tmpCtx.Graph
	}
	return err
}

// ────────────────────────────────────────────────────────────
// Phase 3d-2b: module/entity/association CRUD handler bridge
// ────────────────────────────────────────────────────────────

// phase3d2bNewExecContext builds a temporary *ExecContext from HandlerDeps
// for bridge functions that still call old *ExecContext handlers.
// Populates role-specific backend fields from deps.Backend.
func phase3d2bNewExecContext(ctx context.Context, deps *HandlerDeps) *ExecContext {
	ectx := &ExecContext{
		Context: ctx,
		Backend: deps.Backend,
		ExecIO:  ExecIO{Output: deps.Output},
		ExecSession: ExecSession{Cache: deps.Cache},
		ExecRepos: ExecRepos{
			DomainModels:      deps.DomainModels,
			Microflows:        deps.MicroflowRepo,
			Nanoflows:         deps.NanoflowRepo,
			Security:          deps.Security,
			JavaActions:       deps.JavaActionRepo,
			JavaScriptActions: deps.JavaScriptActionRepo,
			Workflows:         deps.WorkflowRepo,
			Pages:             deps.PageRepo,
			Layouts:           deps.LayoutRepo,
			Snippets:          deps.SnippetRepo,
		},
	}
	// Populate role-specific backend fields from deps.Backend
	// for old handler functions that still access ctx.XxxReader/Writer/Manager.
	if deps.Backend != nil {
		ectx.ModuleLister = deps.Backend
		ectx.ModuleWriter = deps.Backend
		ectx.DomainModelReader = deps.Backend
		ectx.DomainModelWriter = deps.Backend
		ectx.MicroflowReader = deps.Backend
		ectx.MicroflowWriter = deps.Backend
		ectx.WorkflowReader = deps.Backend
		ectx.WorkflowWriter = deps.Backend
		ectx.PageReader = deps.Backend
		ectx.PageWriter = deps.Backend
		ectx.JavaActionReader = deps.Backend
		ectx.JavaActionWriter = deps.Backend
		ectx.JavaScriptActionWriter = deps.Backend
		ectx.EnumerationReader = deps.Backend
		ectx.EnumerationWriter = deps.Backend
		ectx.ConstantReader = deps.Backend
		ectx.ConstantWriter = deps.Backend
		ectx.SettingsReader = deps.Backend
		ectx.SettingsWriter = deps.Backend
		ectx.MappingReader = deps.Backend
		ectx.MappingWriter = deps.Backend
		ectx.UnitReader = deps.Backend
		ectx.UnitWriter = deps.Backend
		ectx.NavigationReader = deps.Backend
		ectx.NavigationWriter = deps.Backend
		ectx.ImageCollectionWriter = deps.Backend
		ectx.ScheduledEventReader = deps.Backend
		ectx.ServiceLister = deps.Backend
		ectx.ServiceWriter = deps.Backend
		ectx.MetadataReader = deps.Backend
		ectx.ConnectionManager = deps.Backend
		ectx.FolderManager = deps.Backend
		ectx.ModuleSettingsReader = deps.Backend
		ectx.ModuleSettingsWriter = deps.Backend
		ectx.RenameManager = deps.Backend
		ectx.SecurityProjectManager = deps.Backend
		ectx.SecurityModuleManager = deps.Backend
		ectx.SecurityEntityAccessManager = deps.Backend
		ectx.PageModelAccess = deps.Backend
		ectx.PageMutationOperator = deps.Backend
		ectx.WorkflowMutationOperator = deps.Backend
		ectx.WidgetBuilder = deps.Backend
		ectx.ScriptTransactionManager = deps.Backend
		ectx.AgentEditorOperator = deps.Backend
	}
	return ectx
}

// ────────────────────────────────────────────────────────────
// Phase 3d-2e: remaining handler bridges (Enumeration, Constant, Module, etc.)
// ────────────────────────────────────────────────────────────

func execCreateEnumerationFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execCreateEnumeration(ectx, stmt.(*ast.CreateEnumerationStmt))
}

func execAlterEnumerationFuture(ctx context.Context, deps *HandlerDeps) error {
	return mdlerrors.NewUnsupported("alter enumeration not yet implemented")
}

func execDropEnumerationFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execDropEnumeration(ectx, stmt.(*ast.DropEnumerationStmt))
}

func execCreateConstantFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return createConstant(ectx, stmt.(*ast.CreateConstantStmt))
}

func execDropConstantFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return dropConstant(ectx, stmt.(*ast.DropConstantStmt))
}

func execAlterModuleJarDepFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execAlterModuleJarDep(ectx, stmt.(*ast.AlterModuleJarDepStmt))
}

func execCreateDatabaseConnectionFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return createDatabaseConnection(ectx, stmt.(*ast.CreateDatabaseConnectionStmt))
}

func execCreateJavaActionFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	return execCreateJavaActionGenFn(ctx, stmt.(*ast.CreateJavaActionStmt), deps)
}

func execDropJavaActionFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	return execDropJavaActionGenFn(ctx, stmt.(*ast.DropJavaActionStmt), deps)
}

func execCreateJavaScriptActionFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	return execCreateJavaScriptActionFn(ctx, stmt.(*ast.CreateJavaScriptActionStmt), deps)
}

func execDropFolderFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	return execDropFolderFn(ctx, stmt.(*ast.DropFolderStmt), deps)
}

func execMoveFolderFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	return execMoveFolderFn(ctx, stmt.(*ast.MoveFolderStmt), deps)
}

func execMoveFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	return execMoveFn(ctx, stmt.(*ast.MoveStmt), deps)
}

func execRenameFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	return execRenameFn(ctx, stmt.(*ast.RenameStmt), deps)
}

func execAlterNavigationFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	return execAlterNavigationFn(ctx, stmt.(*ast.AlterNavigationStmt), deps)
}

func execCreateImageCollectionFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	return execCreateImageCollectionFn(ctx, stmt.(*ast.CreateImageCollectionStmt), deps)
}

func execDropImageCollectionFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	return execDropImageCollectionFn(ctx, stmt.(*ast.DropImageCollectionStmt), deps)
}

func execAlterImageCollectionFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	return execAlterImageCollectionFn(ctx, stmt.(*ast.AlterImageCollectionStmt), deps)
}

func execAlterSettingsFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return alterSettings(ectx, stmt.(*ast.AlterSettingsStmt))
}

func execTranslateFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return translateDocument(ectx, stmt.(*ast.TranslateStmt))
}

func execTranslateMicroflowFuture(ctx context.Context, deps *HandlerDeps) error {
	return mdlerrors.NewUnsupported("TRANSLATE MICROFLOW not yet implemented")
}

func execCreateConfigurationFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return createConfiguration(ectx, stmt.(*ast.CreateConfigurationStmt))
}

func execDropConfigurationFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return dropConfiguration(ectx, stmt.(*ast.DropConfigurationStmt))
}

func execCreateBusinessEventServiceFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return createBusinessEventService(ectx, stmt.(*ast.CreateBusinessEventServiceStmt))
}

func execDropBusinessEventServiceFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return dropBusinessEventService(ectx, stmt.(*ast.DropBusinessEventServiceStmt))
}

func execCreateODataClientFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return createODataClient(ectx, stmt.(*ast.CreateODataClientStmt))
}

func execAlterODataClientFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return alterODataClient(ectx, stmt.(*ast.AlterODataClientStmt))
}

func execDropODataClientFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return dropODataClient(ectx, stmt.(*ast.DropODataClientStmt))
}

func execCreateODataServiceFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return createODataService(ectx, stmt.(*ast.CreateODataServiceStmt))
}

func execAlterODataServiceFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return alterODataService(ectx, stmt.(*ast.AlterODataServiceStmt))
}

func execDropODataServiceFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return dropODataService(ectx, stmt.(*ast.DropODataServiceStmt))
}

func execCreateJsonStructureFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execCreateJsonStructure(ectx, stmt.(*ast.CreateJsonStructureStmt))
}

func execDropJsonStructureFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execDropJsonStructure(ectx, stmt.(*ast.DropJsonStructureStmt))
}

func execCreateImportMappingFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	return execCreateImportMappingFn(ctx, stmt.(*ast.CreateImportMappingStmt), deps)
}

func execDropImportMappingFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	return execDropImportMappingFn(ctx, stmt.(*ast.DropImportMappingStmt), deps)
}

func execCreateExportMappingFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	return execCreateExportMappingFn(ctx, stmt.(*ast.CreateExportMappingStmt), deps)
}

func execDropExportMappingFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	return execDropExportMappingFn(ctx, stmt.(*ast.DropExportMappingStmt), deps)
}

func execCreateRestClientFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return createRestClient(ectx, stmt.(*ast.CreateRestClientStmt))
}

func execDropRestClientFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return dropRestClient(ectx, stmt.(*ast.DropRestClientStmt))
}

func execDescribeContractFromOpenAPIFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return describeContractFromOpenAPI(ectx, stmt.(*ast.DescribeContractFromOpenAPIStmt))
}

func execCreatePublishedRestServiceFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	return execCreatePublishedRestServiceFn(ctx, stmt.(*ast.CreatePublishedRestServiceStmt), deps)
}

func execDropPublishedRestServiceFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	return execDropPublishedRestServiceFn(ctx, stmt.(*ast.DropPublishedRestServiceStmt), deps)
}

func execAlterPublishedRestServiceFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	return execAlterPublishedRestServiceFn(ctx, stmt.(*ast.AlterPublishedRestServiceStmt), deps)
}

func execCreateExternalEntityFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execCreateExternalEntity(ectx, stmt.(*ast.CreateExternalEntityStmt))
}

func execCreateExternalEntitiesFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return createExternalEntities(ectx, stmt.(*ast.CreateExternalEntitiesStmt))
}

func execCreateDataTransformerFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	return execCreateDataTransformerFn(ctx, stmt.(*ast.CreateDataTransformerStmt), deps)
}

func execDropDataTransformerFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	return execDropDataTransformerFn(ctx, stmt.(*ast.DropDataTransformerStmt), deps)
}

func execShowWidgetsFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	return execShowWidgetsFn(ctx, stmt.(*ast.ShowWidgetsStmt), deps)
}

func execShowInstalledWidgetsFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	return execShowInstalledWidgetsFn(ctx, stmt.(*ast.ShowInstalledWidgetsStmt), deps)
}

func execUpdateWidgetsFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	return execUpdateWidgetsFn(ctx, stmt.(*ast.UpdateWidgetsStmt), deps)
}

func execSelectFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execCatalogQuery(ectx, stmt.(*ast.SelectStmt).Query)
}

func execDescribeTranslationsFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return describeTranslations(ectx, stmt.(*ast.DescribeTranslationsStmt))
}

func execDescribeCatalogTableFuture(ctx context.Context, deps *HandlerDeps) error {
	return mdlerrors.NewUnsupported("Catalog SQLite system has been removed")
}

func execShowFeaturesFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execShowFeatures(ectx, stmt.(*ast.ShowFeaturesStmt))
}

func execShowDesignPropertiesFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	return execShowDesignPropertiesFn(ctx, stmt.(*ast.ShowDesignPropertiesStmt), deps)
}

func execDescribeStylingFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	return execDescribeStylingFn(ctx, stmt.(*ast.DescribeStylingStmt), deps)
}

func execAlterStylingFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	return execAlterStylingFn(ctx, stmt.(*ast.AlterStylingStmt), deps)
}

func execShowThemeVariablesFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	return execShowThemeVariablesFn(ctx, stmt.(*ast.ShowThemeVariablesStmt), deps)
}

func execSearchFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execSearch(ectx, stmt.(*ast.SearchStmt))
}

func execRefreshCatalogFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	if err := execRefreshCatalogStmt(ectx, stmt.(*ast.RefreshCatalogStmt)); err != nil {
		return err
	}
	return buildGraph(ectx)
}

func execLintFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execLint(ectx, stmt.(*ast.LintStmt))
}

func execDefineFragmentFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execDefineFragment(ectx, stmt.(*ast.DefineFragmentStmt))
}

func execDescribeFragmentFromFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return describeFragmentFrom(ectx, stmt.(*ast.DescribeFragmentFromStmt))
}

func execSQLConnectFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execSQLConnect(ectx, stmt.(*ast.SQLConnectStmt))
}

func execSQLDisconnectFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execSQLDisconnect(ectx, stmt.(*ast.SQLDisconnectStmt))
}

func execSQLConnectionsFuture(ctx context.Context, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execSQLConnections(ectx)
}

func execSQLQueryFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execSQLQuery(ectx, stmt.(*ast.SQLQueryStmt))
}

func execSQLShowTablesFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execSQLShowTables(ectx, stmt.(*ast.SQLShowTablesStmt))
}

func execSQLShowViewsFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execSQLShowViews(ectx, stmt.(*ast.SQLShowViewsStmt))
}

func execSQLShowFunctionsFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execSQLShowFunctions(ectx, stmt.(*ast.SQLShowFunctionsStmt))
}

func execSQLDescribeTableFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execSQLDescribeTable(ectx, stmt.(*ast.SQLDescribeTableStmt))
}

func execSQLGenerateConnectorFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execSQLGenerateConnector(ectx, stmt.(*ast.SQLGenerateConnectorStmt))
}

func execImportFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	return execImportFn(ctx, stmt.(*ast.ImportStmt), deps)
}

func execCreateModelFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execCreateAgentEditorModel(ectx, stmt.(*ast.CreateModelStmt))
}

func execDropModelFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execDropAgentEditorModel(ectx, stmt.(*ast.DropModelStmt))
}

func execCreateConsumedMCPServiceFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execCreateConsumedMCPService(ectx, stmt.(*ast.CreateConsumedMCPServiceStmt))
}

func execDropConsumedMCPServiceFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execDropConsumedMCPService(ectx, stmt.(*ast.DropConsumedMCPServiceStmt))
}

func execCreateKnowledgeBaseFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execCreateKnowledgeBase(ectx, stmt.(*ast.CreateKnowledgeBaseStmt))
}

func execDropKnowledgeBaseFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execDropKnowledgeBase(ectx, stmt.(*ast.DropKnowledgeBaseStmt))
}

func execCreateAgentFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execCreateAgent(ectx, stmt.(*ast.CreateAgentStmt))
}

func execDropAgentFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execDropAgent(ectx, stmt.(*ast.DropAgentStmt))
}

// Missing Fn wrappers (bridges to old *ExecContext functions for handlers that
// reference Fn patterns in handler_deps.go but don't have native Fn versions yet).
func execCreateModuleFn(ctx context.Context, s *ast.CreateModuleStmt, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execCreateModule(ectx, s)
}

func execDropModuleFn(ctx context.Context, s *ast.DropModuleStmt, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execDropModule(ectx, s)
}

func execCreateAssociationFn(ctx context.Context, s *ast.CreateAssociationStmt, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execCreateAssociation(ectx, s)
}

func execAlterAssociationFn(ctx context.Context, s *ast.AlterAssociationStmt, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execAlterAssociationGen(ectx, s)
}

func execDropAssociationFn(ctx context.Context, s *ast.DropAssociationStmt, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execDropAssociationGen(ectx, s)
}

func execCreatePageV3Fn(ctx context.Context, s *ast.CreatePageStmtV3, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execCreatePageV3(ectx, s)
}

func execCreateSnippetV3Fn(ctx context.Context, s *ast.CreateSnippetStmtV3, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execCreateSnippetV3(ectx, s)
}

func execAlterPageFn(ctx context.Context, s *ast.AlterPageStmt, deps *HandlerDeps) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execAlterPage(ectx, s)
}

func listODataClientsFn(ctx context.Context, deps *HandlerDeps, format OutputFormat, moduleName string) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	ectx.Format = format
	return listODataClients(ectx, moduleName)
}

func listODataServicesFn(ctx context.Context, deps *HandlerDeps, format OutputFormat, moduleName string) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	ectx.Format = format
	return listODataServices(ectx, moduleName)
}

func listExternalEntitiesFn(ctx context.Context, deps *HandlerDeps, format OutputFormat, moduleName string) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	ectx.Format = format
	return listExternalEntities(ectx, moduleName)
}

func listExternalActionsFn(ctx context.Context, deps *HandlerDeps, format OutputFormat, moduleName string) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	ectx.Format = format
	return listExternalActions(ectx, moduleName)
}

func describeODataClientFn(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return describeODataClient(ectx, name)
}

func describeODataServiceFn(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return describeODataService(ectx, name)
}

func describeExternalEntityFn(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := phase3d2bNewExecContext(ctx, deps)
	return describeExternalEntity(ectx, name)
}
