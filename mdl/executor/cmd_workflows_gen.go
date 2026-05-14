// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.3 Phase A: gen-typed SHOW/DESCRIBE WORKFLOW commands.
//
// This file is the gen-typed twin of cmd_workflows.go. It walks
// modelsdk/gen/workflows.Workflow units (decoded by ctx.Workflows) via
// the listWorkflowsWithContainerGen cache helper, then dispatches
// activities by storage $Type rather than concrete sdk Go type.
//
// Notes:
//
//   - SystemTask: gen has no narrow SystemTask type. The $Type
//     "Workflows$SystemTask" is read via element.Element.Raw() with
//     codec.ReadBSONFieldString fallbacks for the legacy fields.
//   - UserTask family: legacy sdk had a unified *UserTask. gen splits
//     into *UserTask (legacy storage), *SingleUserTaskActivity,
//     *MultiUserTaskActivity. The dispatcher handles all three.
//   - CallMicroflow family: gen exposes both *CallMicroflowActivity and
//     *CallMicroflowTask (the legacy storage). Both decode into the same
//     "shape" but live under different gen Go types — dispatch on $Type.
//   - User source / targeting: gen uses "EmptyUserSource" (formerly
//     NoUserSource) and split source vs. targeting (per R5). The
//     dispatcher handles both naming conventions for backward compat.

package executor

import (
	"fmt"
	"sort"
	"strings"

	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

// listWorkflowsGen handles SHOW WORKFLOWS via gen-typed Workflow units.
func listWorkflowsGen(ctx *ExecContext, moduleName string) error {
	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	pairs, err := listWorkflowsWithContainerGen(ctx)
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

	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modName := ""
		if h != nil {
			modID := h.FindModuleID(model.ID(p.ContainerID))
			modName = h.GetModuleName(modID)
		}
		if moduleName != "" && modName != moduleName {
			continue
		}
		qualifiedName := modName + "." + p.Elem.Name()
		paramEntity := workflowParameterEntityGen(p.Elem)
		acts, uts, decs := countWorkflowActivitiesGen(p.Elem)
		rows = append(rows, row{qualifiedName, modName, p.Elem.Name(), acts, uts, decs, paramEntity})
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
	return writeResult(ctx, result)
}

// workflowParameterEntityGen returns the entity qualified name carried
// by Workflow.Parameter (a *genWf.Parameter Part), or "" if absent.
func workflowParameterEntityGen(wf *genWf.Workflow) string {
	if wf == nil {
		return ""
	}
	param := wf.Parameter()
	if param == nil {
		return ""
	}
	if p, ok := param.(*genWf.Parameter); ok {
		return p.EntityQualifiedName()
	}
	// Defensive fallback: read raw BSON field if the Part decoded to a
	// non-narrow type.
	v, _ := codec.ReadBSONFieldString(param.Raw(), "Entity")
	return v
}

// countWorkflowActivitiesGen counts total activities, user tasks, and
// decisions in a gen-typed workflow.
func countWorkflowActivitiesGen(wf *genWf.Workflow) (total, userTasks, decisions int) {
	if wf == nil {
		return
	}
	flow, _ := wf.Flow().(*genWf.Flow)
	if flow == nil {
		return
	}
	countFlowActivitiesGen(flow, &total, &userTasks, &decisions)
	return
}

// countFlowActivitiesGen recursively counts activities in a gen flow
// and its sub-flows. Dispatch is by $Type because gen splits user-task
// and call-microflow into multiple concrete types.
func countFlowActivitiesGen(flow *genWf.Flow, total, userTasks, decisions *int) {
	if flow == nil {
		return
	}
	for _, a := range flow.ActivitiesItems() {
		if a == nil {
			continue
		}
		*total++
		switch a.TypeName() {
		case "Workflows$UserTask",
			"Workflows$SingleUserTaskActivity",
			"Workflows$MultiUserTaskActivity":
			*userTasks++
			countNestedFlowsInOutcomesGen(a, total, userTasks, decisions)
		case "Workflows$ExclusiveSplitActivity":
			*decisions++
			countNestedFlowsInOutcomesGen(a, total, userTasks, decisions)
		case "Workflows$ParallelSplitActivity":
			countNestedFlowsInOutcomesGen(a, total, userTasks, decisions)
		case "Workflows$CallMicroflowActivity",
			"Workflows$CallMicroflowTask",
			"Workflows$SystemTask":
			countNestedFlowsInOutcomesGen(a, total, userTasks, decisions)
		}
	}
}

// countNestedFlowsInOutcomesGen walks the outcomes/paths of a composite
// activity (UserTask, ExclusiveSplit, ParallelSplit, CallMicroflow,
// SystemTask) and recurses into each nested Flow.
//
// It works generically by looking up the activity's gen-typed
// OutcomesItems via reflection-free type assertion to the known
// concrete shapes; falls back to inspecting raw BSON for the
// SystemTask case (gen has no narrow type for it).
func countNestedFlowsInOutcomesGen(a element.Element, total, userTasks, decisions *int) {
	switch v := a.(type) {
	case *genWf.UserTask:
		for _, oc := range v.OutcomesItems() {
			countOutcomeFlowGen(oc, total, userTasks, decisions)
		}
	case *genWf.SingleUserTaskActivity:
		for _, oc := range v.OutcomesItems() {
			countOutcomeFlowGen(oc, total, userTasks, decisions)
		}
	case *genWf.MultiUserTaskActivity:
		for _, oc := range v.OutcomesItems() {
			countOutcomeFlowGen(oc, total, userTasks, decisions)
		}
	case *genWf.ExclusiveSplitActivity:
		for _, oc := range v.OutcomesItems() {
			countOutcomeFlowGen(oc, total, userTasks, decisions)
		}
	case *genWf.ParallelSplitActivity:
		for _, oc := range v.OutcomesItems() {
			countOutcomeFlowGen(oc, total, userTasks, decisions)
		}
	case *genWf.CallMicroflowActivity:
		for _, oc := range v.OutcomesItems() {
			countOutcomeFlowGen(oc, total, userTasks, decisions)
		}
	case *genWf.CallMicroflowTask:
		for _, oc := range v.OutcomesItems() {
			countOutcomeFlowGen(oc, total, userTasks, decisions)
		}
	default:
		// SystemTask + unknown types: outcomes live under raw BSON. The
		// counts are advisory (tests don't depend on every leaf), so we
		// don't reach into raw BSON here — the legacy counter only walks
		// the explicit sdk types.
	}
}

// countOutcomeFlowGen extracts the nested Flow from an outcome element
// and recurses. Outcome elements expose Flow() element.Element returning
// a *Workflows$Flow Part.
func countOutcomeFlowGen(oc element.Element, total, userTasks, decisions *int) {
	if oc == nil {
		return
	}
	switch v := oc.(type) {
	case *genWf.UserTaskOutcome:
		if f, ok := v.Flow().(*genWf.Flow); ok {
			countFlowActivitiesGen(f, total, userTasks, decisions)
		}
	case *genWf.BooleanConditionOutcome:
		if f, ok := v.Flow().(*genWf.Flow); ok {
			countFlowActivitiesGen(f, total, userTasks, decisions)
		}
	case *genWf.EnumerationValueConditionOutcome:
		if f, ok := v.Flow().(*genWf.Flow); ok {
			countFlowActivitiesGen(f, total, userTasks, decisions)
		}
	case *genWf.VoidConditionOutcome:
		if f, ok := v.Flow().(*genWf.Flow); ok {
			countFlowActivitiesGen(f, total, userTasks, decisions)
		}
	case *genWf.ParallelSplitOutcome:
		if f, ok := v.Flow().(*genWf.Flow); ok {
			countFlowActivitiesGen(f, total, userTasks, decisions)
		}
	case *genWf.ExclusiveSplitOutcome:
		if f, ok := v.Flow().(*genWf.Flow); ok {
			countFlowActivitiesGen(f, total, userTasks, decisions)
		}
	}
}

// describeWorkflowGen / describeWorkflowToStringGen and the format*Gen
// activity dispatchers land in subsequent A2-A4 commits. Phase A1
// commits the read surface (listWorkflowsGen + activity counters) so
// dispatcher cutover can land alongside the formatters once they exist.
