// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
	"github.com/mendixlabs/mxcli/sdk/javaactions"
	"github.com/mendixlabs/mxcli/sdk/microflows"
	"github.com/mendixlabs/mxcli/sdk/workflows"
)

// execShowStructure handles SHOW STRUCTURE [DEPTH n] [IN module] [ALL].
func (e *Executor) execShowStructure(s *ast.ShowStmt) error {
	if e.reader == nil {
		return fmt.Errorf("not connected to a project")
	}

	depth := min(max(s.Depth, 1), 3)

	// Ensure catalog is built (fast mode is sufficient)
	if err := e.ensureCatalog(false); err != nil {
		return fmt.Errorf("failed to build catalog: %w", err)
	}

	// Get modules from catalog
	modules, err := e.getStructureModules(s.InModule, s.All)
	if err != nil {
		return err
	}

	if len(modules) == 0 {
		fmt.Fprintln(e.output, "(no modules found)")
		return nil
	}

	switch depth {
	case 1:
		return e.structureDepth1(modules)
	case 2:
		return e.structureDepth2(modules)
	case 3:
		return e.structureDepth3(modules)
	default:
		return e.structureDepth2(modules)
	}
}

// structureModule holds module info for structure output.
type structureModule struct {
	Name string
	ID   model.ID
}

// getStructureModules returns filtered and sorted modules for structure output.
func (e *Executor) getStructureModules(filterModule string, includeAll bool) ([]structureModule, error) {
	result, err := e.catalog.Query("SELECT Id, Name, Source, AppStoreGuid FROM modules ORDER BY Name")
	if err != nil {
		return nil, fmt.Errorf("failed to query modules: %w", err)
	}

	var modules []structureModule
	for _, row := range result.Rows {
		id := asString(row[0])
		name := asString(row[1])
		source := asString(row[2])
		appStoreGuid := asString(row[3])

		// Filter by module name if specified
		if filterModule != "" && !strings.EqualFold(name, filterModule) {
			continue
		}

		// Skip system/marketplace modules unless --all
		if !includeAll && !isUserModule(name, source, appStoreGuid) {
			continue
		}

		modules = append(modules, structureModule{Name: name, ID: model.ID(id)})
	}

	sort.Slice(modules, func(i, j int) bool {
		return strings.ToLower(modules[i].Name) < strings.ToLower(modules[j].Name)
	})

	return modules, nil
}

// isUserModule returns true if the module is a user-created module (not system or marketplace).
func isUserModule(name, source, appStoreGuid string) bool {
	if source != "" {
		return false
	}
	if appStoreGuid != "" {
		return false
	}
	if strings.HasPrefix(name, "_") {
		return false
	}
	return true
}

// asString converts an interface{} value to string.
func asString(v any) string {
	if v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	case []byte:
		return string(s)
	case int64:
		return fmt.Sprintf("%d", s)
	default:
		return fmt.Sprintf("%v", s)
	}
}

// ============================================================================
// Depth 1 — Module Summary
// ============================================================================

func (e *Executor) structureDepth1(modules []structureModule) error {
	// Query counts per module from catalog
	entityCounts := e.queryCountByModule("entities")
	mfCounts := e.queryCountByModule("microflows WHERE MicroflowType = 'MICROFLOW'")
	nfCounts := e.queryCountByModule("microflows WHERE MicroflowType = 'NANOFLOW'")
	pageCounts := e.queryCountByModule("pages")
	enumCounts := e.queryCountByModule("enumerations")
	snippetCounts := e.queryCountByModule("snippets")
	jaCounts := e.queryCountByModule("java_actions")
	wfCounts := e.queryCountByModule("workflows")
	odataClientCounts := e.queryCountByModule("odata_clients")
	odataServiceCounts := e.queryCountByModule("odata_services")
	beServiceCounts := e.queryCountByModule("business_event_services")

	// Get constants and scheduled events from reader (no catalog tables)
	constantCounts := e.countByModuleFromReader("constants")
	scheduledEventCounts := e.countByModuleFromReader("scheduled_events")

	// Calculate name column width for alignment
	nameWidth := 0
	for _, m := range modules {
		if len(m.Name) > nameWidth {
			nameWidth = len(m.Name)
		}
	}

	for _, m := range modules {
		var parts []string

		if c := entityCounts[m.Name]; c > 0 {
			parts = append(parts, pluralize(c, "entity", "entities"))
		}
		if c := enumCounts[m.Name]; c > 0 {
			parts = append(parts, pluralize(c, "enum", "enums"))
		}
		if c := mfCounts[m.Name]; c > 0 {
			parts = append(parts, pluralize(c, "microflow", "microflows"))
		}
		if c := nfCounts[m.Name]; c > 0 {
			parts = append(parts, pluralize(c, "nanoflow", "nanoflows"))
		}
		if c := wfCounts[m.Name]; c > 0 {
			parts = append(parts, pluralize(c, "workflow", "workflows"))
		}
		if c := pageCounts[m.Name]; c > 0 {
			parts = append(parts, pluralize(c, "page", "pages"))
		}
		if c := snippetCounts[m.Name]; c > 0 {
			parts = append(parts, pluralize(c, "snippet", "snippets"))
		}
		if c := jaCounts[m.Name]; c > 0 {
			parts = append(parts, pluralize(c, "java action", "java actions"))
		}
		if c := constantCounts[m.Name]; c > 0 {
			parts = append(parts, pluralize(c, "constant", "constants"))
		}
		if c := scheduledEventCounts[m.Name]; c > 0 {
			parts = append(parts, pluralize(c, "scheduled event", "scheduled events"))
		}
		if c := odataClientCounts[m.Name]; c > 0 {
			parts = append(parts, pluralize(c, "odata client", "odata clients"))
		}
		if c := odataServiceCounts[m.Name]; c > 0 {
			parts = append(parts, pluralize(c, "odata service", "odata services"))
		}
		if c := beServiceCounts[m.Name]; c > 0 {
			parts = append(parts, pluralize(c, "business event service", "business event services"))
		}

		if len(parts) > 0 {
			fmt.Fprintf(e.output, "%-*s  %s\n", nameWidth, m.Name, strings.Join(parts, ", "))
		}
	}
	return nil
}

// queryCountByModule queries a catalog table and returns a map of module name → count.
func (e *Executor) queryCountByModule(tableAndWhere string) map[string]int {
	counts := make(map[string]int)
	sql := fmt.Sprintf("SELECT ModuleName, COUNT(*) FROM %s GROUP BY ModuleName", tableAndWhere)
	result, err := e.catalog.Query(sql)
	if err != nil {
		return counts
	}
	for _, row := range result.Rows {
		name := asString(row[0])
		counts[name] = toInt(row[1])
	}
	return counts
}

// countByModuleFromReader counts elements per module using the reader (for types without catalog tables).
func (e *Executor) countByModuleFromReader(kind string) map[string]int {
	counts := make(map[string]int)
	h, err := e.getHierarchy()
	if err != nil {
		return counts
	}

	switch kind {
	case "constants":
		if constants, err := e.reader.ListConstants(); err == nil {
			for _, c := range constants {
				modID := h.FindModuleID(c.ContainerID)
				modName := h.GetModuleName(modID)
				counts[modName]++
			}
		}
	case "scheduled_events":
		if events, err := e.reader.ListScheduledEvents(); err == nil {
			for _, ev := range events {
				modID := h.FindModuleID(ev.ContainerID)
				modName := h.GetModuleName(modID)
				counts[modName]++
			}
		}
	}
	return counts
}

// pluralize returns "N thing" or "N things" depending on count.
func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, singular)
	}
	return fmt.Sprintf("%d %s", count, plural)
}

// ============================================================================
// Depth 2 — Elements with Signatures
// ============================================================================

func (e *Executor) structureDepth2(modules []structureModule) error {
	// Pre-load data that needs the reader
	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	// Load domain models for associations
	domainModels, _ := e.reader.ListDomainModels()
	dmByModule := make(map[string]*domainmodel.DomainModel)
	for _, dm := range domainModels {
		modID := h.FindModuleID(dm.ContainerID)
		modName := h.GetModuleName(modID)
		dmByModule[modName] = dm
	}

	// Load enumerations for values
	allEnums, _ := e.reader.ListEnumerations()
	enumsByModule := make(map[string][]*model.Enumeration)
	for _, enum := range allEnums {
		modID := h.FindModuleID(enum.ContainerID)
		modName := h.GetModuleName(modID)
		enumsByModule[modName] = append(enumsByModule[modName], enum)
	}

	// Load microflows for parameter types
	allMicroflows, _ := e.reader.ListMicroflows()
	mfByModule := make(map[string][]*microflows.Microflow)
	for _, mf := range allMicroflows {
		modID := h.FindModuleID(mf.ContainerID)
		modName := h.GetModuleName(modID)
		mfByModule[modName] = append(mfByModule[modName], mf)
	}

	// Load nanoflows
	allNanoflows, _ := e.reader.ListNanoflows()
	nfByModule := make(map[string][]*microflows.Nanoflow)
	for _, nf := range allNanoflows {
		modID := h.FindModuleID(nf.ContainerID)
		modName := h.GetModuleName(modID)
		nfByModule[modName] = append(nfByModule[modName], nf)
	}

	// Load constants
	allConstants, _ := e.reader.ListConstants()
	constByModule := make(map[string][]*model.Constant)
	for _, c := range allConstants {
		modID := h.FindModuleID(c.ContainerID)
		modName := h.GetModuleName(modID)
		constByModule[modName] = append(constByModule[modName], c)
	}

	// Load scheduled events
	allEvents, _ := e.reader.ListScheduledEvents()
	eventsByModule := make(map[string][]*model.ScheduledEvent)
	for _, ev := range allEvents {
		modID := h.FindModuleID(ev.ContainerID)
		modName := h.GetModuleName(modID)
		eventsByModule[modName] = append(eventsByModule[modName], ev)
	}

	// Load java actions for parameter types
	allJavaActions, _ := e.reader.ListJavaActionsFull()
	jaByModule := make(map[string][]*javaactions.JavaAction)
	for _, ja := range allJavaActions {
		modID := h.FindModuleID(ja.ContainerID)
		modName := h.GetModuleName(modID)
		jaByModule[modName] = append(jaByModule[modName], ja)
	}

	// Load workflows
	allWorkflows, _ := e.reader.ListWorkflows()
	wfByModule := make(map[string][]*workflows.Workflow)
	for _, wf := range allWorkflows {
		modID := h.FindModuleID(wf.ContainerID)
		modName := h.GetModuleName(modID)
		wfByModule[modName] = append(wfByModule[modName], wf)
	}

	for i, m := range modules {
		if i > 0 {
			fmt.Fprintln(e.output)
		}
		fmt.Fprintln(e.output, m.Name)

		// Entities
		e.structureEntities(m.Name, dmByModule[m.Name], false)

		// Enumerations
		if enums, ok := enumsByModule[m.Name]; ok {
			sortEnumerations(enums)
			for _, enum := range enums {
				values := make([]string, len(enum.Values))
				for i, v := range enum.Values {
					values[i] = v.Name
				}
				fmt.Fprintf(e.output, "  Enumeration %s.%s [%s]\n", m.Name, enum.Name, strings.Join(values, ", "))
			}
		}

		// Microflows
		if mfs, ok := mfByModule[m.Name]; ok {
			sortMicroflows(mfs)
			for _, mf := range mfs {
				fmt.Fprintf(e.output, "  Microflow %s.%s%s\n", m.Name, mf.Name, formatMicroflowSignature(mf.Parameters, mf.ReturnType, false))
			}
		}

		// Nanoflows
		if nfs, ok := nfByModule[m.Name]; ok {
			sortNanoflows(nfs)
			for _, nf := range nfs {
				fmt.Fprintf(e.output, "  Nanoflow %s.%s%s\n", m.Name, nf.Name, formatMicroflowSignature(nf.Parameters, nf.ReturnType, false))
			}
		}

		// Workflows
		e.structureWorkflows(m.Name, wfByModule[m.Name], false)

		// Pages (from catalog)
		e.structurePages(m.Name)

		// Snippets (from catalog)
		e.structureSnippets(m.Name)

		// Java Actions
		e.outputJavaActions(m.Name, jaByModule[m.Name], false)

		// Constants
		if consts, ok := constByModule[m.Name]; ok {
			sortConstants(consts)
			for _, c := range consts {
				fmt.Fprintf(e.output, "  Constant %s.%s: %s\n", m.Name, c.Name, formatConstantTypeBrief(c.Type))
			}
		}

		// Scheduled Events
		if events, ok := eventsByModule[m.Name]; ok {
			sortScheduledEvents(events)
			for _, ev := range events {
				fmt.Fprintf(e.output, "  ScheduledEvent %s.%s\n", m.Name, ev.Name)
			}
		}

		// OData Clients
		e.structureODataClients(m.Name)

		// OData Services
		e.structureODataServices(m.Name)

		// Business Event Services
		e.structureBusinessEventServices(m.Name)
	}

	return nil
}

// ============================================================================
// Depth 3 — Include Types and Details
// ============================================================================

func (e *Executor) structureDepth3(modules []structureModule) error {
	// Same data loading as depth 2
	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	domainModels, _ := e.reader.ListDomainModels()
	dmByModule := make(map[string]*domainmodel.DomainModel)
	for _, dm := range domainModels {
		modID := h.FindModuleID(dm.ContainerID)
		modName := h.GetModuleName(modID)
		dmByModule[modName] = dm
	}

	allEnums, _ := e.reader.ListEnumerations()
	enumsByModule := make(map[string][]*model.Enumeration)
	for _, enum := range allEnums {
		modID := h.FindModuleID(enum.ContainerID)
		modName := h.GetModuleName(modID)
		enumsByModule[modName] = append(enumsByModule[modName], enum)
	}

	allMicroflows, _ := e.reader.ListMicroflows()
	mfByModule := make(map[string][]*microflows.Microflow)
	for _, mf := range allMicroflows {
		modID := h.FindModuleID(mf.ContainerID)
		modName := h.GetModuleName(modID)
		mfByModule[modName] = append(mfByModule[modName], mf)
	}

	allNanoflows, _ := e.reader.ListNanoflows()
	nfByModule := make(map[string][]*microflows.Nanoflow)
	for _, nf := range allNanoflows {
		modID := h.FindModuleID(nf.ContainerID)
		modName := h.GetModuleName(modID)
		nfByModule[modName] = append(nfByModule[modName], nf)
	}

	allConstants, _ := e.reader.ListConstants()
	constByModule := make(map[string][]*model.Constant)
	for _, c := range allConstants {
		modID := h.FindModuleID(c.ContainerID)
		modName := h.GetModuleName(modID)
		constByModule[modName] = append(constByModule[modName], c)
	}

	allEvents, _ := e.reader.ListScheduledEvents()
	eventsByModule := make(map[string][]*model.ScheduledEvent)
	for _, ev := range allEvents {
		modID := h.FindModuleID(ev.ContainerID)
		modName := h.GetModuleName(modID)
		eventsByModule[modName] = append(eventsByModule[modName], ev)
	}

	allJavaActions, _ := e.reader.ListJavaActionsFull()
	jaByModule := make(map[string][]*javaactions.JavaAction)
	for _, ja := range allJavaActions {
		modID := h.FindModuleID(ja.ContainerID)
		modName := h.GetModuleName(modID)
		jaByModule[modName] = append(jaByModule[modName], ja)
	}

	// Load workflows
	allWorkflows, _ := e.reader.ListWorkflows()
	wfByModule := make(map[string][]*workflows.Workflow)
	for _, wf := range allWorkflows {
		modID := h.FindModuleID(wf.ContainerID)
		modName := h.GetModuleName(modID)
		wfByModule[modName] = append(wfByModule[modName], wf)
	}

	for i, m := range modules {
		if i > 0 {
			fmt.Fprintln(e.output)
		}
		fmt.Fprintln(e.output, m.Name)

		// Entities (with types)
		e.structureEntities(m.Name, dmByModule[m.Name], true)

		// Enumerations
		if enums, ok := enumsByModule[m.Name]; ok {
			sortEnumerations(enums)
			for _, enum := range enums {
				values := make([]string, len(enum.Values))
				for i, v := range enum.Values {
					values[i] = v.Name
				}
				fmt.Fprintf(e.output, "  Enumeration %s.%s [%s]\n", m.Name, enum.Name, strings.Join(values, ", "))
			}
		}

		// Microflows (with param names)
		if mfs, ok := mfByModule[m.Name]; ok {
			sortMicroflows(mfs)
			for _, mf := range mfs {
				fmt.Fprintf(e.output, "  Microflow %s.%s%s\n", m.Name, mf.Name, formatMicroflowSignature(mf.Parameters, mf.ReturnType, true))
			}
		}

		// Nanoflows (with param names)
		if nfs, ok := nfByModule[m.Name]; ok {
			sortNanoflows(nfs)
			for _, nf := range nfs {
				fmt.Fprintf(e.output, "  Nanoflow %s.%s%s\n", m.Name, nf.Name, formatMicroflowSignature(nf.Parameters, nf.ReturnType, true))
			}
		}

		// Workflows (with details)
		e.structureWorkflows(m.Name, wfByModule[m.Name], true)

		// Pages
		e.structurePages(m.Name)

		// Snippets
		e.structureSnippets(m.Name)

		// Java Actions (with param names)
		e.outputJavaActions(m.Name, jaByModule[m.Name], true)

		// Constants (with default value)
		if consts, ok := constByModule[m.Name]; ok {
			sortConstants(consts)
			for _, c := range consts {
				s := fmt.Sprintf("  Constant %s.%s: %s", m.Name, c.Name, formatConstantTypeBrief(c.Type))
				if c.DefaultValue != "" {
					s += " = " + c.DefaultValue
				}
				fmt.Fprintln(e.output, s)
			}
		}

		// Scheduled Events
		if events, ok := eventsByModule[m.Name]; ok {
			sortScheduledEvents(events)
			for _, ev := range events {
				fmt.Fprintf(e.output, "  ScheduledEvent %s.%s\n", m.Name, ev.Name)
			}
		}

		// OData
		e.structureODataClients(m.Name)
		e.structureODataServices(m.Name)

		// Business Event Services
		e.structureBusinessEventServices(m.Name)
	}

	return nil
}

// ============================================================================
// Shared Element Formatters
// ============================================================================

// structureEntities outputs entities for a module.
func (e *Executor) structureEntities(moduleName string, dm *domainmodel.DomainModel, withTypes bool) {
	if dm == nil {
		return
	}

	// Build entity ID → name map for association resolution
	entityByID := make(map[model.ID]string)
	for _, ent := range dm.Entities {
		entityByID[ent.ID] = ent.Name
	}

	// Sort entities alphabetically
	entities := make([]*domainmodel.Entity, len(dm.Entities))
	copy(entities, dm.Entities)
	sort.Slice(entities, func(i, j int) bool {
		return strings.ToLower(entities[i].Name) < strings.ToLower(entities[j].Name)
	})

	// Build association lookup: parent entity ID → associations
	assocByParent := make(map[model.ID][]*domainmodel.Association)
	for _, assoc := range dm.Associations {
		assocByParent[assoc.ParentID] = append(assocByParent[assoc.ParentID], assoc)
	}

	for _, ent := range entities {
		// Format attributes
		var attrParts []string
		for _, attr := range ent.Attributes {
			if withTypes {
				attrParts = append(attrParts, formatAttributeWithType(attr))
			} else {
				attrParts = append(attrParts, attr.Name)
			}
		}
		qualName := moduleName + "." + ent.Name
		if len(attrParts) > 0 {
			fmt.Fprintf(e.output, "  Entity %s [%s]\n", qualName, strings.Join(attrParts, ", "))
		} else {
			fmt.Fprintf(e.output, "  Entity %s\n", qualName)
		}

		// Format associations (owned by this entity)
		if assocs, ok := assocByParent[ent.ID]; ok {
			var assocParts []string
			for _, assoc := range assocs {
				childName := entityByID[assoc.ChildID]
				if childName == "" {
					childName = "?"
				}
				cardinality := "(1)"
				if assoc.Type == domainmodel.AssociationTypeReferenceSet {
					cardinality = "(*)"
				}
				part := fmt.Sprintf("→ %s %s", childName, cardinality)
				if withTypes {
					// Add delete behavior if non-default (DeleteMeButKeepReferences is default)
					if assoc.ChildDeleteBehavior != nil && assoc.ChildDeleteBehavior.Type == domainmodel.DeleteBehaviorTypeDeleteMeAndReferences {
						part += " CASCADE"
					} else if assoc.ChildDeleteBehavior != nil && assoc.ChildDeleteBehavior.Type == domainmodel.DeleteBehaviorTypeDeleteMeIfNoReferences {
						part += " RESTRICT"
					}
				}
				assocParts = append(assocParts, part)
			}
			if len(assocParts) > 0 {
				fmt.Fprintf(e.output, "    %s\n", strings.Join(assocParts, ", "))
			}
		}
	}
}

// structurePages outputs pages for a module from the catalog.
func (e *Executor) structurePages(moduleName string) {
	// Query pages from catalog
	result, err := e.catalog.Query(fmt.Sprintf(
		"SELECT Name FROM pages WHERE ModuleName = '%s' ORDER BY Name",
		escapeSQLString(moduleName)))
	if err != nil || len(result.Rows) == 0 {
		return
	}

	// Try to get top-level data widgets from widgets table
	widgetsByPage := make(map[string][]string)
	widgetResult, err := e.catalog.Query(fmt.Sprintf(
		"SELECT ContainerQualifiedName, WidgetType, EntityRef FROM widgets WHERE ModuleName = '%s' AND ParentWidget = '' ORDER BY ContainerQualifiedName, WidgetType",
		escapeSQLString(moduleName)))
	if err == nil {
		for _, row := range widgetResult.Rows {
			pageName := asString(row[0])
			widgetType := asString(row[1])
			entityRef := asString(row[2])

			// Only include data-bound widgets
			if !isDataWidget(widgetType) {
				continue
			}

			// Extract short widget type name
			shortType := shortWidgetType(widgetType)
			if entityRef != "" {
				// Extract entity name from qualified name
				shortEntity := shortName(entityRef)
				widgetsByPage[pageName] = append(widgetsByPage[pageName], fmt.Sprintf("%s<%s>", shortType, shortEntity))
			} else {
				widgetsByPage[pageName] = append(widgetsByPage[pageName], shortType)
			}
		}
	}

	for _, row := range result.Rows {
		name := asString(row[0])
		qualName := moduleName + "." + name
		if widgets, ok := widgetsByPage[qualName]; ok && len(widgets) > 0 {
			fmt.Fprintf(e.output, "  Page %s [%s]\n", qualName, strings.Join(widgets, ", "))
		} else {
			fmt.Fprintf(e.output, "  Page %s\n", qualName)
		}
	}
}

// structureSnippets outputs snippets for a module from the catalog.
func (e *Executor) structureSnippets(moduleName string) {
	result, err := e.catalog.Query(fmt.Sprintf(
		"SELECT Name FROM snippets WHERE ModuleName = '%s' ORDER BY Name",
		escapeSQLString(moduleName)))
	if err != nil || len(result.Rows) == 0 {
		return
	}

	for _, row := range result.Rows {
		name := asString(row[0])
		fmt.Fprintf(e.output, "  Snippet %s.%s\n", moduleName, name)
	}
}

// outputJavaActions outputs java actions for a module.
func (e *Executor) outputJavaActions(moduleName string, actions []*javaactions.JavaAction, withNames bool) {
	if len(actions) == 0 {
		return
	}

	// Sort alphabetically
	sorted := make([]*javaactions.JavaAction, len(actions))
	copy(sorted, actions)
	sort.Slice(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})

	for _, ja := range sorted {
		sig := formatJavaActionSignature(ja, withNames)
		fmt.Fprintf(e.output, "  JavaAction %s.%s%s\n", moduleName, ja.Name, sig)
	}
}

// formatJavaActionSignature formats the parameter list and return type of a java action.
func formatJavaActionSignature(ja *javaactions.JavaAction, withNames bool) string {
	var paramParts []string
	for _, p := range ja.Parameters {
		typeName := ""
		if p.ParameterType != nil {
			typeName = formatJATypeDisplay(p.ParameterType.TypeString())
		}
		if withNames && p.Name != "" {
			paramParts = append(paramParts, fmt.Sprintf("%s: %s", p.Name, typeName))
		} else {
			paramParts = append(paramParts, typeName)
		}
	}

	sig := "(" + strings.Join(paramParts, ", ") + ")"

	// Add return type
	if ja.ReturnType != nil {
		retStr := ja.ReturnType.TypeString()
		if retStr != "" && retStr != "Void" && retStr != "Nothing" {
			sig += " → " + formatJATypeDisplay(retStr)
		}
	}

	return sig
}

// formatJATypeDisplay formats a java action type string for display.
func formatJATypeDisplay(typeStr string) string {
	// TypeString() returns things like "Module.Entity", "List of Module.Entity", "Boolean", etc.
	if after, ok := strings.CutPrefix(typeStr, "List of "); ok {
		entity := after
		return "List<" + shortName(entity) + ">"
	}
	// Check if it's a qualified name (contains a dot)
	if strings.Contains(typeStr, ".") {
		return shortName(typeStr)
	}
	return typeStr
}

// structureODataClients outputs OData clients for a module.
func (e *Executor) structureODataClients(moduleName string) {
	result, err := e.catalog.Query(fmt.Sprintf(
		"SELECT Name, ODataVersion FROM odata_clients WHERE ModuleName = '%s' ORDER BY Name",
		escapeSQLString(moduleName)))
	if err != nil || len(result.Rows) == 0 {
		return
	}

	for _, row := range result.Rows {
		name := asString(row[0])
		version := asString(row[1])
		qualName := moduleName + "." + name
		if version != "" {
			fmt.Fprintf(e.output, "  ODataClient %s (%s)\n", qualName, version)
		} else {
			fmt.Fprintf(e.output, "  ODataClient %s\n", qualName)
		}
	}
}

// structureODataServices outputs OData services for a module.
func (e *Executor) structureODataServices(moduleName string) {
	result, err := e.catalog.Query(fmt.Sprintf(
		"SELECT Name, Path, EntitySetCount FROM odata_services WHERE ModuleName = '%s' ORDER BY Name",
		escapeSQLString(moduleName)))
	if err != nil || len(result.Rows) == 0 {
		return
	}

	for _, row := range result.Rows {
		name := asString(row[0])
		path := asString(row[1])
		entitySetCount := toInt(row[2])
		qualName := moduleName + "." + name
		if path != "" {
			fmt.Fprintf(e.output, "  ODataService %s %s (%s)\n", qualName, path, pluralize(entitySetCount, "entity", "entities"))
		} else {
			fmt.Fprintf(e.output, "  ODataService %s\n", qualName)
		}
	}
}

// structureBusinessEventServices outputs business event services for a module.
func (e *Executor) structureBusinessEventServices(moduleName string) {
	result, err := e.catalog.Query(fmt.Sprintf(
		"SELECT Name, MessageCount, PublishCount, SubscribeCount FROM business_event_services WHERE ModuleName = '%s' ORDER BY Name",
		escapeSQLString(moduleName)))
	if err != nil || len(result.Rows) == 0 {
		return
	}

	for _, row := range result.Rows {
		name := asString(row[0])
		msgCount := toInt(row[1])
		publishCount := toInt(row[2])
		subscribeCount := toInt(row[3])
		qualName := moduleName + "." + name

		var parts []string
		if msgCount > 0 {
			parts = append(parts, pluralize(msgCount, "message", "messages"))
		}
		if publishCount > 0 {
			parts = append(parts, pluralize(publishCount, "publish", "publish"))
		}
		if subscribeCount > 0 {
			parts = append(parts, pluralize(subscribeCount, "subscribe", "subscribe"))
		}

		if len(parts) > 0 {
			fmt.Fprintf(e.output, "  BusinessEventService %s (%s)\n", qualName, strings.Join(parts, ", "))
		} else {
			fmt.Fprintf(e.output, "  BusinessEventService %s\n", qualName)
		}
	}
}

// structureWorkflows outputs workflows for a module.
func (e *Executor) structureWorkflows(moduleName string, wfs []*workflows.Workflow, withDetails bool) {
	if len(wfs) == 0 {
		return
	}

	// Sort alphabetically
	sorted := make([]*workflows.Workflow, len(wfs))
	copy(sorted, wfs)
	sort.Slice(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})

	for _, wf := range sorted {
		qualName := moduleName + "." + wf.Name
		var parts []string

		// Count activities
		total, userTasks, _, decisions := countStructureWorkflowActivities(wf)
		if total > 0 {
			parts = append(parts, pluralize(total, "activity", "activities"))
		}
		if userTasks > 0 {
			parts = append(parts, pluralize(userTasks, "user task", "user tasks"))
		}
		if decisions > 0 {
			parts = append(parts, pluralize(decisions, "decision", "decisions"))
		}

		if withDetails && wf.Parameter != nil && wf.Parameter.EntityRef != "" {
			entityPart := "param: " + shortName(wf.Parameter.EntityRef)
			parts = append(parts, entityPart)
		}

		if len(parts) > 0 {
			fmt.Fprintf(e.output, "  Workflow %s (%s)\n", qualName, strings.Join(parts, ", "))
		} else {
			fmt.Fprintf(e.output, "  Workflow %s\n", qualName)
		}
	}
}

// countStructureWorkflowActivities counts activity types in a workflow for structure output.
func countStructureWorkflowActivities(wf *workflows.Workflow) (total, userTasks, microflowCalls, decisions int) {
	if wf.Flow == nil {
		return
	}
	countStructureFlowActivities(wf.Flow, &total, &userTasks, &microflowCalls, &decisions)
	return
}

// countStructureFlowActivities recursively counts activity types in a flow.
func countStructureFlowActivities(flow *workflows.Flow, total, userTasks, microflowCalls, decisions *int) {
	if flow == nil {
		return
	}
	for _, act := range flow.Activities {
		*total++
		switch a := act.(type) {
		case *workflows.UserTask:
			*userTasks++
			for _, outcome := range a.Outcomes {
				countStructureFlowActivities(outcome.Flow, total, userTasks, microflowCalls, decisions)
			}
		case *workflows.CallMicroflowTask:
			*microflowCalls++
			for _, outcome := range a.Outcomes {
				if outcome != nil {
					countStructureFlowActivities(outcome.GetFlow(), total, userTasks, microflowCalls, decisions)
				}
			}
		case *workflows.SystemTask:
			*microflowCalls++
			for _, outcome := range a.Outcomes {
				if outcome != nil {
					countStructureFlowActivities(outcome.GetFlow(), total, userTasks, microflowCalls, decisions)
				}
			}
		case *workflows.ExclusiveSplitActivity:
			*decisions++
			for _, outcome := range a.Outcomes {
				if outcome != nil {
					countStructureFlowActivities(outcome.GetFlow(), total, userTasks, microflowCalls, decisions)
				}
			}
		case *workflows.ParallelSplitActivity:
			for _, outcome := range a.Outcomes {
				countStructureFlowActivities(outcome.Flow, total, userTasks, microflowCalls, decisions)
			}
		}
	}
}

// ============================================================================
// Formatting Helpers
// ============================================================================

// formatMicroflowSignature formats the parameter list and return type of a microflow.
func formatMicroflowSignature(params []*microflows.MicroflowParameter, returnType microflows.DataType, withNames bool) string {
	var paramParts []string
	for _, p := range params {
		typeName := formatDataTypeDisplay(p.Type)
		if withNames && p.Name != "" {
			paramParts = append(paramParts, fmt.Sprintf("%s: %s", p.Name, typeName))
		} else {
			paramParts = append(paramParts, typeName)
		}
	}

	sig := "(" + strings.Join(paramParts, ", ") + ")"

	// Add return type
	if returnType != nil {
		retName := formatDataTypeDisplay(returnType)
		if retName != "" && retName != "Void" && retName != "Nothing" {
			sig += " → " + retName
		}
	}

	return sig
}

// formatDataTypeDisplay formats a microflow DataType for display.
func formatDataTypeDisplay(dt microflows.DataType) string {
	if dt == nil {
		return ""
	}
	switch t := dt.(type) {
	case *microflows.BooleanType:
		return "Boolean"
	case *microflows.IntegerType:
		return "Integer"
	case *microflows.LongType:
		return "Long"
	case *microflows.DecimalType:
		return "Decimal"
	case *microflows.StringType:
		return "String"
	case *microflows.DateTimeType:
		return "DateTime"
	case *microflows.DateType:
		return "Date"
	case *microflows.ObjectType:
		return shortName(t.EntityQualifiedName)
	case *microflows.ListType:
		return "List<" + shortName(t.EntityQualifiedName) + ">"
	case *microflows.EnumerationType:
		return shortName(t.EnumerationQualifiedName)
	case *microflows.VoidType:
		return "Void"
	case *microflows.BinaryType:
		return "Binary"
	default:
		return dt.GetTypeName()
	}
}

// formatAttributeWithType formats an attribute with its type for depth 3.
func formatAttributeWithType(attr *domainmodel.Attribute) string {
	if attr.Type == nil {
		return attr.Name
	}
	switch t := attr.Type.(type) {
	case *domainmodel.StringAttributeType:
		if t.Length > 0 {
			return fmt.Sprintf("%s: String(%d)", attr.Name, t.Length)
		}
		return attr.Name + ": String(unlimited)"
	case *domainmodel.EnumerationAttributeType:
		return attr.Name + ": " + shortName(t.EnumerationRef)
	default:
		return attr.Name + ": " + attr.Type.GetTypeName()
	}
}

// formatConstantTypeBrief formats a constant type for display.
func formatConstantTypeBrief(dt model.ConstantDataType) string {
	switch dt.Kind {
	case "Enumeration":
		if dt.EnumRef != "" {
			return shortName(dt.EnumRef)
		}
		return "Enumeration"
	default:
		return dt.Kind
	}
}

// shortName extracts the name part from a qualified name (Module.Name → Name).
func shortName(qualifiedName string) string {
	if idx := strings.LastIndex(qualifiedName, "."); idx >= 0 {
		return qualifiedName[idx+1:]
	}
	return qualifiedName
}

// shortWidgetType extracts a readable widget type from the full type string.
func shortWidgetType(widgetType string) string {
	// Widget types may look like "DataGrid", "DataView", "ListView", etc.
	// Or pluggable widgets like "com.mendix.widget.web.datagrid2.DataGrid2"
	if idx := strings.LastIndex(widgetType, "."); idx >= 0 {
		return widgetType[idx+1:]
	}
	return widgetType
}

// isDataWidget returns true if the widget type is a data-bound widget worth showing in structure.
func isDataWidget(widgetType string) bool {
	lower := strings.ToLower(widgetType)
	return strings.Contains(lower, "dataview") ||
		strings.Contains(lower, "datagrid") ||
		strings.Contains(lower, "listview") ||
		strings.Contains(lower, "templategrid") ||
		strings.Contains(lower, "gallery")
}

// escapeSQLString escapes single quotes in a string for SQL.
func escapeSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// ============================================================================
// Sort Helpers
// ============================================================================

func sortEnumerations(enums []*model.Enumeration) {
	sort.Slice(enums, func(i, j int) bool {
		return strings.ToLower(enums[i].Name) < strings.ToLower(enums[j].Name)
	})
}

func sortMicroflows(mfs []*microflows.Microflow) {
	sort.Slice(mfs, func(i, j int) bool {
		return strings.ToLower(mfs[i].Name) < strings.ToLower(mfs[j].Name)
	})
}

func sortNanoflows(nfs []*microflows.Nanoflow) {
	sort.Slice(nfs, func(i, j int) bool {
		return strings.ToLower(nfs[i].Name) < strings.ToLower(nfs[j].Name)
	})
}

func sortConstants(consts []*model.Constant) {
	sort.Slice(consts, func(i, j int) bool {
		return strings.ToLower(consts[i].Name) < strings.ToLower(consts[j].Name)
	})
}

func sortScheduledEvents(events []*model.ScheduledEvent) {
	sort.Slice(events, func(i, j int) bool {
		return strings.ToLower(events[i].Name) < strings.ToLower(events[j].Name)
	})
}
