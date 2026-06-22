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
