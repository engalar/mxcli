// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"sort"
	"strings"

	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"

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
	mods, err := ctx.ModuleLister.ListModules()
	if err != nil {
		return nil, mdlerrors.NewBackend("list modules", err)
	}

	var modules []structureModule
	for _, m := range mods {
		if filterModule != "" && !strings.EqualFold(m.Name, filterModule) {
			continue
		}
		if !includeAll && !isUserModule(m.Name, "", "") {
			continue
		}
		modules = append(modules, structureModule{Name: m.Name, ID: m.ID})
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

// queryCountByModule is a no-op — catalog has been replaced by MXGraph.
// Use countByModuleViaBackend for backend-based counting.
func queryCountByModule(ctx *ExecContext, tableAndWhere string) map[string]int {
	return make(map[string]int)
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
		if constants, err := ctx.ConstantReader.ListConstants(); err == nil {
			for _, c := range constants {
				modID := h.FindModuleID(c.ContainerID)
				modName := h.GetModuleName(modID)
				counts[modName]++
			}
		}
	case "scheduled_events":
		if events, err := ctx.ScheduledEventReader.ListScheduledEvents(); err == nil {
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

// structurePages outputs pages for a module via backend.
func structurePages(ctx *ExecContext, moduleName string) {
	if ctx.Pages == nil {
		return
	}
	pages, err := ctx.Pages.ListAll()
	if err != nil {
		return
	}
	for _, p := range pages {
		if p == nil {
			continue
		}
		modName := findModuleNameByContainer(ctx, model.ID(p.ID()))
		if modName == moduleName {
			fmt.Fprintf(ctx.Output, "  Page %s.%s\n", moduleName, p.Name())
		}
	}
}

// structureSnippets outputs snippets for a module via backend.
func structureSnippets(ctx *ExecContext, moduleName string) {
	if ctx.Snippets == nil {
		return
	}
	snippets, err := ctx.Snippets.ListAll()
	if err != nil {
		return
	}
	for _, s := range snippets {
		if s == nil {
			continue
		}
		modName := findModuleNameByContainer(ctx, model.ID(s.ID()))
		if modName == moduleName {
			fmt.Fprintf(ctx.Output, "  Snippet %s.%s\n", moduleName, s.Name())
		}
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

// structureODataClients outputs OData clients for a module (stub — catalog removed).
func structureODataClients(ctx *ExecContext, moduleName string) {
	// OData indexing requires MXGraph adapter; not yet available.
}

// structureODataServices outputs OData services for a module (stub — catalog removed).
func structureODataServices(ctx *ExecContext, moduleName string) {
	// OData indexing requires MXGraph adapter; not yet available.
}

// structureBusinessEventServices outputs business event services for a module (stub — catalog removed).
func structureBusinessEventServices(ctx *ExecContext, moduleName string) {
	// Business event indexing requires MXGraph adapter; not yet available.
}

// structureWorkflows outputs workflows for a module (gen-typed).
func structureWorkflows(ctx *ExecContext, moduleName string, wfs []*genWf.Workflow, withDetails bool) {
	if len(wfs) == 0 {
		return
	}

	sorted := make([]*genWf.Workflow, len(wfs))
	copy(sorted, wfs)
	sort.Slice(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].Name()) < strings.ToLower(sorted[j].Name())
	})

	for _, wf := range sorted {
		qualName := moduleName + "." + wf.Name()
		var parts []string

		total, userTasks, _, decisions := countStructureWorkflowActivitiesGen(wf)
		if total > 0 {
			parts = append(parts, pluralize(total, "activity", "activities"))
		}
		if userTasks > 0 {
			parts = append(parts, pluralize(userTasks, "user task", "user tasks"))
		}
		if decisions > 0 {
			parts = append(parts, pluralize(decisions, "decision", "decisions"))
		}

		if withDetails {
			if entity := workflowParameterEntityGen(wf); entity != "" {
				parts = append(parts, "param: "+shortName(entity))
			}
		}

		if len(parts) > 0 {
			fmt.Fprintf(ctx.Output, "  Workflow %s (%s)\n", qualName, strings.Join(parts, ", "))
		} else {
			fmt.Fprintf(ctx.Output, "  Workflow %s\n", qualName)
		}
	}
}

// countStructureWorkflowActivitiesGen counts activity types in a gen
// workflow for structure output. Reuses the same activity-counting
// algorithm as cmd_workflows_gen.go but with the microflowCalls metric
// the structure command needs (decision and userTasks are already
// counted by countWorkflowActivitiesGen, but it doesn't expose
// microflowCalls). We dispatch here directly.
func countStructureWorkflowActivitiesGen(wf *genWf.Workflow) (total, userTasks, microflowCalls, decisions int) {
	if wf == nil {
		return
	}
	flow, ok := wf.Flow().(*genWf.Flow)
	if !ok || flow == nil {
		return
	}
	countStructureFlowActivitiesGen(flow, &total, &userTasks, &microflowCalls, &decisions)
	return
}

func countStructureFlowActivitiesGen(flow *genWf.Flow, total, userTasks, microflowCalls, decisions *int) {
	if flow == nil {
		return
	}
	for _, act := range flow.ActivitiesItems() {
		if act == nil {
			continue
		}
		*total++
		switch act.TypeName() {
		case "Workflows$UserTask",
			"Workflows$SingleUserTaskActivity",
			"Workflows$MultiUserTaskActivity":
			*userTasks++
		case "Workflows$CallMicroflowTask",
			"Workflows$CallMicroflowActivity",
			"Workflows$SystemTask":
			*microflowCalls++
		case "Workflows$ExclusiveSplitActivity":
			*decisions++
		}
		// Recurse into outcomes' nested flows.
		switch v := act.(type) {
		case *genWf.UserTask:
			for _, oc := range v.OutcomesItems() {
				if utc, ok := oc.(*genWf.UserTaskOutcome); ok {
					if f, ok := utc.Flow().(*genWf.Flow); ok {
						countStructureFlowActivitiesGen(f, total, userTasks, microflowCalls, decisions)
					}
				}
			}
		case *genWf.SingleUserTaskActivity:
			for _, oc := range v.OutcomesItems() {
				if utc, ok := oc.(*genWf.UserTaskOutcome); ok {
					if f, ok := utc.Flow().(*genWf.Flow); ok {
						countStructureFlowActivitiesGen(f, total, userTasks, microflowCalls, decisions)
					}
				}
			}
		case *genWf.MultiUserTaskActivity:
			for _, oc := range v.OutcomesItems() {
				if utc, ok := oc.(*genWf.UserTaskOutcome); ok {
					if f, ok := utc.Flow().(*genWf.Flow); ok {
						countStructureFlowActivitiesGen(f, total, userTasks, microflowCalls, decisions)
					}
				}
			}
		case *genWf.CallMicroflowActivity:
			structureRecurseConditionOutcomesGen(v.OutcomesItems(), total, userTasks, microflowCalls, decisions)
		case *genWf.CallMicroflowTask:
			structureRecurseConditionOutcomesGen(v.OutcomesItems(), total, userTasks, microflowCalls, decisions)
		case *genWf.ExclusiveSplitActivity:
			structureRecurseConditionOutcomesGen(v.OutcomesItems(), total, userTasks, microflowCalls, decisions)
		case *genWf.ParallelSplitActivity:
			for _, oc := range v.OutcomesItems() {
				if pso, ok := oc.(*genWf.ParallelSplitOutcome); ok {
					if f, ok := pso.Flow().(*genWf.Flow); ok {
						countStructureFlowActivitiesGen(f, total, userTasks, microflowCalls, decisions)
					}
				}
			}
		}
	}
}

func structureRecurseConditionOutcomesGen(outcomes []element.Element, total, userTasks, microflowCalls, decisions *int) {
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
			countStructureFlowActivitiesGen(f, total, userTasks, microflowCalls, decisions)
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

// getStructureModulesDeps is the HandlerDeps version of getStructureModules.
func getStructureModulesDeps(deps *HandlerDeps, filterModule string, includeAll bool) ([]structureModule, error) {
	mods, err := deps.ModuleLister.ListModules()
	if err != nil {
		return nil, mdlerrors.NewBackend("list modules", err)
	}

	var modules []structureModule
	for _, m := range mods {
		if filterModule != "" && !strings.EqualFold(m.Name, filterModule) {
			continue
		}
		if !includeAll && !isUserModule(m.Name, "", "") {
			continue
		}
		modules = append(modules, structureModule{Name: m.Name, ID: m.ID})
	}

	sort.Slice(modules, func(i, j int) bool {
		return strings.ToLower(modules[i].Name) < strings.ToLower(modules[j].Name)
	})

	return modules, nil
}

// structureDepth1JSONDeps is the HandlerDeps version of structureDepth1JSON.
func structureDepth1JSONDeps(deps *HandlerDeps, modules []structureModule) error {
	if deps.Format != FormatJSON {
		return nil
	}
	tr := &TableResult{
		Columns: []string{"Module"},
	}
	for _, m := range modules {
		tr.Rows = append(tr.Rows, []any{m.Name})
	}
	return writeResultDeps(deps, tr)
}

// structureDepth1Deps is the HandlerDeps version of structureDepth1.
func structureDepth1Deps(deps *HandlerDeps, modules []structureModule) error {
	constantCounts := countByModuleFromBackendDeps(deps, "constants")
	scheduledEventCounts := countByModuleFromBackendDeps(deps, "scheduled_events")

	nameWidth := 0
	for _, m := range modules {
		if len(m.Name) > nameWidth {
			nameWidth = len(m.Name)
		}
	}

	for _, m := range modules {
		var parts []string
		if c := constantCounts[m.Name]; c > 0 {
			parts = append(parts, pluralize(c, "constant", "constants"))
		}
		if c := scheduledEventCounts[m.Name]; c > 0 {
			parts = append(parts, pluralize(c, "scheduled event", "scheduled events"))
		}
		if len(parts) > 0 {
			fmt.Fprintf(deps.Output, "%-*s  %s\n", nameWidth, m.Name, strings.Join(parts, ", "))
		}
	}
	return nil
}

// countByModuleFromBackendDeps is the HandlerDeps version of countByModuleFromBackend.
func countByModuleFromBackendDeps(deps *HandlerDeps, kind string) map[string]int {
	counts := make(map[string]int)
	h, err := getHierarchyDeps(deps)
	if err != nil {
		return counts
	}

	switch kind {
	case "constants":
		if constants, err := deps.ConstantReader.ListConstants(); err == nil {
			for _, c := range constants {
				modID := h.FindModuleID(c.ContainerID)
				modName := h.GetModuleName(modID)
				counts[modName]++
			}
		}
	case "scheduled_events":
		if events, err := deps.ScheduledEventReader.ListScheduledEvents(); err == nil {
			for _, ev := range events {
				modID := h.FindModuleID(ev.ContainerID)
				modName := h.GetModuleName(modID)
				counts[modName]++
			}
		}
	}
	return counts
}

// structurePagesDeps is the HandlerDeps version of structurePages.
func structurePagesDeps(deps *HandlerDeps, moduleName string) {
	if deps.PageRepo == nil {
		return
	}
	pages, err := deps.PageRepo.ListAll()
	if err != nil {
		return
	}
	h, err := getHierarchyDeps(deps)
	if err != nil {
		return
	}
	for _, p := range pages {
		if p == nil {
			continue
		}
		containerID, err := deps.PageRepo.GetContainerUUID(model.ID(p.ID()))
		if err != nil || containerID == "" {
			continue
		}
		modID := h.FindModuleID(containerID)
		modName := h.GetModuleName(modID)
		if modName == moduleName {
			fmt.Fprintf(deps.Output, "  Page %s.%s\n", moduleName, p.Name())
		}
	}
}

// structureSnippetsDeps is the HandlerDeps version of structureSnippets.
func structureSnippetsDeps(deps *HandlerDeps, moduleName string) {
	if deps.SnippetRepo == nil {
		return
	}
	snippets, err := deps.SnippetRepo.ListAll()
	if err != nil {
		return
	}
	h, err := getHierarchyDeps(deps)
	if err != nil {
		return
	}
	for _, s := range snippets {
		if s == nil {
			continue
		}
		containerID, err := deps.SnippetRepo.GetContainerUUID(model.ID(s.ID()))
		if err != nil || containerID == "" {
			continue
		}
		modID := h.FindModuleID(containerID)
		modName := h.GetModuleName(modID)
		if modName == moduleName {
			fmt.Fprintf(deps.Output, "  Snippet %s.%s\n", moduleName, s.Name())
		}
	}
}

// outputJavaActionsGenDeps is the HandlerDeps version of outputJavaActionsGen.
func outputJavaActionsGenDeps(deps *HandlerDeps, moduleName string, actions []*genJA.JavaAction, withNames bool) {
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
		fmt.Fprintf(deps.Output, "  JavaAction %s.%s%s\n", moduleName, ja.Name(), sig)
	}
}

// structureWorkflowsDeps is the HandlerDeps version of structureWorkflows.
func structureWorkflowsDeps(deps *HandlerDeps, moduleName string, wfs []*genWf.Workflow, withDetails bool) {
	if len(wfs) == 0 {
		return
	}

	sorted := make([]*genWf.Workflow, len(wfs))
	copy(sorted, wfs)
	sort.Slice(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].Name()) < strings.ToLower(sorted[j].Name())
	})

	for _, wf := range sorted {
		qualName := moduleName + "." + wf.Name()
		var parts []string

		total, userTasks, _, decisions := countStructureWorkflowActivitiesGen(wf)
		if total > 0 {
			parts = append(parts, pluralize(total, "activity", "activities"))
		}
		if userTasks > 0 {
			parts = append(parts, pluralize(userTasks, "user task", "user tasks"))
		}
		if decisions > 0 {
			parts = append(parts, pluralize(decisions, "decision", "decisions"))
		}

		if withDetails {
			if entity := workflowParameterEntityGen(wf); entity != "" {
				parts = append(parts, "param: "+shortName(entity))
			}
		}

		if len(parts) > 0 {
			fmt.Fprintf(deps.Output, "  Workflow %s (%s)\n", qualName, strings.Join(parts, ", "))
		} else {
			fmt.Fprintf(deps.Output, "  Workflow %s\n", qualName)
		}
	}
}
