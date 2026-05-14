// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.3.C3 — gen-typed workflow catalog ingestion. Replaces the
// legacy sdk-typed walker; activity dispatch is by storage $Type rather
// than concrete sdk Go type because gen splits user-task and call-microflow
// into multiple subtypes.

package catalog

import (
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

func (b *Builder) buildWorkflows() error {
	wfs, err := b.cachedWorkflows()
	if err != nil {
		return err
	}

	if len(wfs) == 0 {
		return nil
	}

	stmt, err := b.tx.Prepare(`
		INSERT INTO workflows (Id, Name, QualifiedName, ModuleName, Folder, Description,
			ExportLevel, ParameterEntity, ActivityCount, UserTaskCount, MicroflowCallCount, DecisionCount,
			DueDate, ProjectId, ProjectName, SnapshotId, SnapshotDate, SnapshotSource,
			SourceId, SourceBranch, SourceRevision)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	projectID, projectName, snapshotID, snapshotDate, snapshotSource, sourceID, sourceBranch, sourceRevision := b.snapshotMeta()

	count := 0
	for _, wf := range wfs {
		if wf == nil {
			continue
		}
		// Codec-decoded gen Workflow loses Container() linkage; resolve
		// owning module by walking the unit hierarchy from the workflow's
		// own UUID (same pattern as javaactions / microflows).
		moduleID := b.hierarchy.findModuleID(model.ID(wf.ID()))
		moduleName := b.hierarchy.getModuleName(moduleID)
		qualifiedName := moduleName + "." + wf.Name()
		folderPath := b.hierarchy.buildFolderPath(model.ID(wf.ID()))

		paramEntity := workflowParamEntityGen(wf)

		actCount, utCount, mfCount, decCount := countWorkflowActivityTypesGen(wf)

		_, err = stmt.Exec(
			string(wf.ID()),
			wf.Name(),
			qualifiedName,
			moduleName,
			folderPath,
			wf.Documentation(),
			wf.ExportLevel(),
			paramEntity,
			actCount,
			utCount,
			mfCount,
			decCount,
			wf.DueDate(),
			projectID, projectName, snapshotID, snapshotDate, snapshotSource,
			sourceID, sourceBranch, sourceRevision,
		)
		if err != nil {
			return err
		}
		count++
	}

	b.report("Workflows", count)
	return nil
}

// workflowParamEntityGen returns Workflow.Parameter.EntityQualifiedName
// or "" when absent.
func workflowParamEntityGen(wf *genWf.Workflow) string {
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
	v, _ := codec.ReadBSONFieldString(param.Raw(), "Entity")
	return v
}

// countWorkflowActivityTypesGen counts activity types in a gen workflow.
func countWorkflowActivityTypesGen(wf *genWf.Workflow) (total, userTasks, microflowCalls, decisions int) {
	if wf == nil {
		return
	}
	flow, _ := wf.Flow().(*genWf.Flow)
	if flow == nil {
		return
	}
	countFlowActivityTypesGen(flow, &total, &userTasks, &microflowCalls, &decisions)
	return
}

// countFlowActivityTypesGen recursively counts activity types in a gen flow.
// Dispatch by storage $Type because gen splits the user-task and
// call-microflow hierarchies into multiple concrete types.
func countFlowActivityTypesGen(flow *genWf.Flow, total, userTasks, microflowCalls, decisions *int) {
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
			countActivityNestedFlowsGen(act, total, userTasks, microflowCalls, decisions)
		case "Workflows$CallMicroflowTask",
			"Workflows$CallMicroflowActivity",
			"Workflows$SystemTask":
			*microflowCalls++
			countActivityNestedFlowsGen(act, total, userTasks, microflowCalls, decisions)
		case "Workflows$ExclusiveSplitActivity":
			*decisions++
			countActivityNestedFlowsGen(act, total, userTasks, microflowCalls, decisions)
		case "Workflows$ParallelSplitActivity":
			countActivityNestedFlowsGen(act, total, userTasks, microflowCalls, decisions)
		}
	}
}

// countActivityNestedFlowsGen walks the outcomes of a composite activity
// and recurses into each nested Flow.
func countActivityNestedFlowsGen(act element.Element, total, userTasks, microflowCalls, decisions *int) {
	switch v := act.(type) {
	case *genWf.UserTask:
		for _, oc := range v.OutcomesItems() {
			countOutcomeFlowGenCatalog(oc, total, userTasks, microflowCalls, decisions)
		}
	case *genWf.SingleUserTaskActivity:
		for _, oc := range v.OutcomesItems() {
			countOutcomeFlowGenCatalog(oc, total, userTasks, microflowCalls, decisions)
		}
	case *genWf.MultiUserTaskActivity:
		for _, oc := range v.OutcomesItems() {
			countOutcomeFlowGenCatalog(oc, total, userTasks, microflowCalls, decisions)
		}
	case *genWf.CallMicroflowActivity:
		for _, oc := range v.OutcomesItems() {
			countOutcomeFlowGenCatalog(oc, total, userTasks, microflowCalls, decisions)
		}
	case *genWf.CallMicroflowTask:
		for _, oc := range v.OutcomesItems() {
			countOutcomeFlowGenCatalog(oc, total, userTasks, microflowCalls, decisions)
		}
	case *genWf.ExclusiveSplitActivity:
		for _, oc := range v.OutcomesItems() {
			countOutcomeFlowGenCatalog(oc, total, userTasks, microflowCalls, decisions)
		}
	case *genWf.ParallelSplitActivity:
		for _, oc := range v.OutcomesItems() {
			countOutcomeFlowGenCatalog(oc, total, userTasks, microflowCalls, decisions)
		}
	}
}

// countOutcomeFlowGenCatalog extracts the nested Flow from a gen
// outcome element and recurses.
func countOutcomeFlowGenCatalog(oc element.Element, total, userTasks, microflowCalls, decisions *int) {
	if oc == nil {
		return
	}
	switch v := oc.(type) {
	case *genWf.UserTaskOutcome:
		if f, ok := v.Flow().(*genWf.Flow); ok {
			countFlowActivityTypesGen(f, total, userTasks, microflowCalls, decisions)
		}
	case *genWf.BooleanConditionOutcome:
		if f, ok := v.Flow().(*genWf.Flow); ok {
			countFlowActivityTypesGen(f, total, userTasks, microflowCalls, decisions)
		}
	case *genWf.EnumerationValueConditionOutcome:
		if f, ok := v.Flow().(*genWf.Flow); ok {
			countFlowActivityTypesGen(f, total, userTasks, microflowCalls, decisions)
		}
	case *genWf.VoidConditionOutcome:
		if f, ok := v.Flow().(*genWf.Flow); ok {
			countFlowActivityTypesGen(f, total, userTasks, microflowCalls, decisions)
		}
	case *genWf.ParallelSplitOutcome:
		if f, ok := v.Flow().(*genWf.Flow); ok {
			countFlowActivityTypesGen(f, total, userTasks, microflowCalls, decisions)
		}
	case *genWf.ExclusiveSplitOutcome:
		if f, ok := v.Flow().(*genWf.Flow); ok {
			countFlowActivityTypesGen(f, total, userTasks, microflowCalls, decisions)
		}
	}
}
