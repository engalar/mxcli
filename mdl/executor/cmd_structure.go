// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"sort"
	"strings"

	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
	"github.com/mendixlabs/mxcli/sdk/workflows"

	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
)

// Stage 3.2.6.3a: `execShowStructure` removed — the dispatch in
// executor_query.go (ShowStructure case) now calls
// `execShowStructureGen`. The legacy `structureDepth2` /
// `structureDepth3` (which iterated sdk/microflows-typed slices) and
// the helpers `formatMicroflowSignature` / `formatDataTypeDisplay` /
// `sortMicroflows` / `sortNanoflows` were also dropped — equivalents
// (`structureDepth2Gen`, `formatMicroflowSignatureGen`,
// `sortGenMicroflows`, etc.) live in cmd_structure_gen.go.
//
// `structureDepth1` / `structureDepth1JSON` and the non-microflow
// helpers (`getStructureModules`, `structureEntities`, `structurePages`,
// `structureSnippets`, `structureWorkflows`, `outputJavaActions`,
// `structureODataClients/Services/BusinessEventServices`,
// `formatConstantTypeBrief`, `pluralize`, `shortName`, ...) stay in
// this file — both the gen path and other callers depend on them.

// structureDepth1JSON emits structure as a JSON table with one row per module
// and columns for each element type count.
func structureDepth1JSON(ctx *ExecContext, modules []structureModule) error {
	entityCounts := queryCountByModule(ctx, "entities")
	mfCounts := queryCountByModule(ctx, "microflows where MicroflowType = 'microflow'")
	nfCounts := queryCountByModule(ctx, "microflows where MicroflowType = 'nanoflow'")
	pageCounts := queryCountByModule(ctx, "pages")
	enumCounts := queryCountByModule(ctx, "enumerations")
	snippetCounts := queryCountByModule(ctx, "snippets")
	jaCounts := queryCountByModule(ctx, "java_actions")
	wfCounts := queryCountByModule(ctx, "workflows")
	odataClientCounts := queryCountByModule(ctx, "odata_clients")
	odataServiceCounts := queryCountByModule(ctx, "odata_services")
	beServiceCounts := queryCountByModule(ctx, "business_event_services")
	constantCounts := countByModuleFromBackend(ctx, "constants")
	scheduledEventCounts := countByModuleFromBackend(ctx, "scheduled_events")

	tr := &TableResult{
		Columns: []string{
			"Module", "Entities", "Enumerations", "Microflows", "Nanoflows",
			"Workflows", "Pages", "Snippets", "JavaActions", "Constants",
			"ScheduledEvents", "ODataClients", "ODataServices", "BusinessEventServices",
		},
	}
	for _, m := range modules {
		tr.Rows = append(tr.Rows, []any{
			m.Name,
			entityCounts[m.Name],
			enumCounts[m.Name],
			mfCounts[m.Name],
			nfCounts[m.Name],
			wfCounts[m.Name],
			pageCounts[m.Name],
			snippetCounts[m.Name],
			jaCounts[m.Name],
			constantCounts[m.Name],
			scheduledEventCounts[m.Name],
			odataClientCounts[m.Name],
			odataServiceCounts[m.Name],
			beServiceCounts[m.Name],
		})
	}
	return writeResult(ctx, tr)
}

// structureModule holds module info for structure output.
type structureModule struct {
	Name string
	ID   model.ID
}

// getStructureModules returns filtered and sorted modules for structure output.
func getStructureModules(ctx *ExecContext, filterModule string, includeAll bool) ([]structureModule, error) {
	result, err := ctx.Catalog.Query("select Id, Name, Source, AppStoreGuid from modules ORDER by Name")
	if err != nil {
		return nil, mdlerrors.NewBackend("query modules", err)
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

func structureDepth1(ctx *ExecContext, modules []structureModule) error {
	// Query counts per module from catalog
	entityCounts := queryCountByModule(ctx, "entities")
	mfCounts := queryCountByModule(ctx, "microflows where MicroflowType = 'microflow'")
	nfCounts := queryCountByModule(ctx, "microflows where MicroflowType = 'nanoflow'")
	pageCounts := queryCountByModule(ctx, "pages")
	enumCounts := queryCountByModule(ctx, "enumerations")
	snippetCounts := queryCountByModule(ctx, "snippets")
	jaCounts := queryCountByModule(ctx, "java_actions")
	wfCounts := queryCountByModule(ctx, "workflows")
	odataClientCounts := queryCountByModule(ctx, "odata_clients")
	odataServiceCounts := queryCountByModule(ctx, "odata_services")
	beServiceCounts := queryCountByModule(ctx, "business_event_services")

	// Get constants and scheduled events from backend (no catalog tables)
	constantCounts := countByModuleFromBackend(ctx, "constants")
	scheduledEventCounts := countByModuleFromBackend(ctx, "scheduled_events")

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
			fmt.Fprintf(ctx.Output, "%-*s  %s\n", nameWidth, m.Name, strings.Join(parts, ", "))
		}
	}
	return nil
}

// queryCountByModule queries a catalog table and returns a map of module name → count.
func queryCountByModule(ctx *ExecContext, tableAndWhere string) map[string]int {
	counts := make(map[string]int)
	sql := fmt.Sprintf("select ModuleName, count(*) from %s GROUP by ModuleName", tableAndWhere)
	result, err := ctx.Catalog.Query(sql)
	if err != nil {
		return counts
	}
	for _, row := range result.Rows {
		name := asString(row[0])
		counts[name] = toInt(row[1])
	}
	return counts
}

// countByModuleFromBackend counts elements per module using the backend (for types without catalog tables).
func countByModuleFromBackend(ctx *ExecContext, kind string) map[string]int {
	counts := make(map[string]int)
	h, err := getHierarchy(ctx)
	if err != nil {
		return counts
	}

	switch kind {
	case "constants":
		if constants, err := ctx.Backend.ListConstants(); err == nil {
			for _, c := range constants {
				modID := h.FindModuleID(c.ContainerID)
				modName := h.GetModuleName(modID)
				counts[modName]++
			}
		}
	case "scheduled_events":
		if events, err := ctx.Backend.ListScheduledEvents(); err == nil {
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
// Shared Element Formatters
// ============================================================================

// structureEntities outputs entities for a module.
func structureEntities(ctx *ExecContext, moduleName string, dm *domainmodel.DomainModel, withTypes bool) {
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
			fmt.Fprintf(ctx.Output, "  Entity %s [%s]\n", qualName, strings.Join(attrParts, ", "))
		} else {
			fmt.Fprintf(ctx.Output, "  Entity %s\n", qualName)
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
						part += " cascade"
					} else if assoc.ChildDeleteBehavior != nil && assoc.ChildDeleteBehavior.Type == domainmodel.DeleteBehaviorTypeDeleteMeIfNoReferences {
						part += " RESTRICT"
					}
				}
				assocParts = append(assocParts, part)
			}
			if len(assocParts) > 0 {
				fmt.Fprintf(ctx.Output, "    %s\n", strings.Join(assocParts, ", "))
			}
		}
	}
}

// structurePages outputs pages for a module from the catalog.
func structurePages(ctx *ExecContext, moduleName string) {
	// Query pages from catalog
	result, err := ctx.Catalog.Query(fmt.Sprintf(
		"select Name from pages where ModuleName = '%s' ORDER by Name",
		escapeSQLString(moduleName)))
	if err != nil || len(result.Rows) == 0 {
		return
	}

	// Try to get top-level data widgets from widgets table
	widgetsByPage := make(map[string][]string)
	widgetResult, err := ctx.Catalog.Query(fmt.Sprintf(
		"select ContainerQualifiedName, WidgetType, EntityRef from widgets where ModuleName = '%s' and ParentWidget = '' ORDER by ContainerQualifiedName, WidgetType",
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
			fmt.Fprintf(ctx.Output, "  Page %s [%s]\n", qualName, strings.Join(widgets, ", "))
		} else {
			fmt.Fprintf(ctx.Output, "  Page %s\n", qualName)
		}
	}
}

// structureSnippets outputs snippets for a module from the catalog.
func structureSnippets(ctx *ExecContext, moduleName string) {
	result, err := ctx.Catalog.Query(fmt.Sprintf(
		"select Name from snippets where ModuleName = '%s' ORDER by Name",
		escapeSQLString(moduleName)))
	if err != nil || len(result.Rows) == 0 {
		return
	}

	for _, row := range result.Rows {
		name := asString(row[0])
		fmt.Fprintf(ctx.Output, "  Snippet %s.%s\n", moduleName, name)
	}
}

// outputJavaActionsGen outputs java actions for a module (gen-typed).
// Stage 3.3.2.E1: legacy outputJavaActions / formatJavaActionSignature
// / formatJATypeDisplay deleted — only the gen variants survive
// (cmd_structure_gen.go already routes here).
func outputJavaActionsGen(ctx *ExecContext, moduleName string, actions []*genJA.JavaAction, withNames bool) {
	if len(actions) == 0 {
		return
	}
	sorted := make([]*genJA.JavaAction, len(actions))
	copy(sorted, actions)
	sort.Slice(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].Name()) < strings.ToLower(sorted[j].Name())
	})
	for _, ja := range sorted {
		sig := formatJavaActionSignatureGen(ja, withNames)
		fmt.Fprintf(ctx.Output, "  JavaAction %s.%s%s\n", moduleName, ja.Name(), sig)
	}
}

// formatJavaActionSignatureGen formats parameter list and return type for a gen-typed JavaAction.
// Uses the dual-accessor helpers from cmd_javaactions_gen.go (ParametersItems preference over ActionParametersItems).
func formatJavaActionSignatureGen(ja *genJA.JavaAction, withNames bool) string {
	typeParams := javaActionTypeParametersOf(ja)
	params := javaActionParametersOf(ja)

	var paramParts []string
	for _, p := range params {
		pp, ok := p.(*genJA.JavaActionParameter)
		if !ok {
			continue
		}
		typeName := formatJavaActionTypeGen(javaActionParameterParameterType(pp), typeParams)
		if withNames && pp.Name() != "" {
			paramParts = append(paramParts, fmt.Sprintf("%s: %s", pp.Name(), typeName))
		} else {
			paramParts = append(paramParts, typeName)
		}
	}

	sig := "(" + strings.Join(paramParts, ", ") + ")"

	rt := javaActionReturnTypeElement(ja)
	if rt != nil {
		retStr := formatJavaActionReturnTypeGen(rt, typeParams)
		if retStr != "" && retStr != "Void" && retStr != "Nothing" {
			sig += " → " + retStr
		}
	}

	return sig
}

// structureODataClients outputs OData clients for a module.
func structureODataClients(ctx *ExecContext, moduleName string) {
	result, err := ctx.Catalog.Query(fmt.Sprintf(
		"select Name, ODataVersion from odata_clients where ModuleName = '%s' ORDER by Name",
		escapeSQLString(moduleName)))
	if err != nil || len(result.Rows) == 0 {
		return
	}

	for _, row := range result.Rows {
		name := asString(row[0])
		version := asString(row[1])
		qualName := moduleName + "." + name
		if version != "" {
			fmt.Fprintf(ctx.Output, "  ODataClient %s (%s)\n", qualName, version)
		} else {
			fmt.Fprintf(ctx.Output, "  ODataClient %s\n", qualName)
		}
	}
}

// structureODataServices outputs OData services for a module.
func structureODataServices(ctx *ExecContext, moduleName string) {
	result, err := ctx.Catalog.Query(fmt.Sprintf(
		"select Name, Path, EntitySetCount from odata_services where ModuleName = '%s' ORDER by Name",
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
			fmt.Fprintf(ctx.Output, "  ODataService %s %s (%s)\n", qualName, path, pluralize(entitySetCount, "entity", "entities"))
		} else {
			fmt.Fprintf(ctx.Output, "  ODataService %s\n", qualName)
		}
	}
}

// structureBusinessEventServices outputs business event services for a module.
func structureBusinessEventServices(ctx *ExecContext, moduleName string) {
	result, err := ctx.Catalog.Query(fmt.Sprintf(
		"select Name, MessageCount, PublishCount, SubscribeCount from business_event_services where ModuleName = '%s' ORDER by Name",
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
			fmt.Fprintf(ctx.Output, "  BusinessEventService %s (%s)\n", qualName, strings.Join(parts, ", "))
		} else {
			fmt.Fprintf(ctx.Output, "  BusinessEventService %s\n", qualName)
		}
	}
}

// structureWorkflows outputs workflows for a module.
func structureWorkflows(ctx *ExecContext, moduleName string, wfs []*workflows.Workflow, withDetails bool) {
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
			fmt.Fprintf(ctx.Output, "  Workflow %s (%s)\n", qualName, strings.Join(parts, ", "))
		} else {
			fmt.Fprintf(ctx.Output, "  Workflow %s\n", qualName)
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

// Stage 3.2.6.3a: `formatMicroflowSignature` and
// `formatDataTypeDisplay` removed. Equivalents using gen types live in
// cmd_structure_gen.go (`formatMicroflowSignatureGen` /
// `formatGenParameterTypeDisplay`). Same applies to `sortMicroflows` /
// `sortNanoflows` further below — replaced by `sortGenMicroflows` /
// `sortGenNanoflows`.

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

// sortMicroflows / sortNanoflows: see cmd_structure_gen.go's
// sortGenMicroflows / sortGenNanoflows for the gen replacements.

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
