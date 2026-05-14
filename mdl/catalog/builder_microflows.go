// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"database/sql"
	"strings"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

func (b *Builder) buildMicroflows() error {
	// Get all microflows (cached — avoids re-parsing in later phases)
	mfs, err := b.cachedMicroflows()
	if err != nil {
		return err
	}

	// Get all nanoflows (cached)
	nfs, err := b.cachedNanoflows()
	if err != nil {
		return err
	}

	mfStmt, err := b.tx.Prepare(`
		INSERT INTO microflows (Id, Name, QualifiedName, ModuleName, Folder, MicroflowType,
			Description, ReturnType, ParameterCount, ActivityCount, Complexity, Excluded,
			ProjectId, ProjectName, SnapshotId, SnapshotDate, SnapshotSource,
			SourceId, SourceBranch, SourceRevision)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer mfStmt.Close()

	// Prepare activity statement only in full mode
	var actStmt *sql.Stmt
	if b.fullMode {
		actStmt, err = b.tx.Prepare(`
			INSERT INTO activities (Id, Name, Caption, ActivityType, Sequence, MicroflowId, MicroflowQualifiedName,
				ModuleName, Folder, EntityRef, ActionType, ServiceRef, ActionRef, Description,
				ProjectId, ProjectName, SnapshotId, SnapshotDate, SnapshotSource,
				SourceId, SourceBranch, SourceRevision)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return err
		}
		defer actStmt.Close()
	}

	projectID, projectName, snapshotID, snapshotDate, snapshotSource, sourceID, sourceBranch, sourceRevision := b.snapshotMeta()

	mfCount := 0
	nfCount := 0
	actCount := 0

	// Process microflows
	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		// Resolve owning module via the unit-table hierarchy.
		// Codec-decoded gen flows don't carry Container linkage, so
		// we walk the unit hierarchy from the flow's own UUID; the
		// hierarchy maps mf.UUID -> parent (module/folder) -> ... -> module.
		moduleID := b.hierarchy.findModuleID(model.ID(mf.ID()))
		moduleName := b.hierarchy.getModuleName(moduleID)
		qualifiedName := moduleName + "." + mf.Name()

		oc := flowObjectCollection(mf.ObjectCollection())
		paramCount := countFlowParameters(oc)
		activityCount := countFlowActivities(oc)
		complexity := calculateFlowComplexity(oc)
		returnType := strings.TrimSpace(mf.ReturnType())

		_, err = mfStmt.Exec(
			string(mf.ID()),
			mf.Name(),
			qualifiedName,
			moduleName,
			moduleName, // Folder
			"MICROFLOW",
			mf.Documentation(),
			returnType,
			paramCount,
			activityCount,
			complexity,
			mf.Excluded(),
			projectID, projectName, snapshotID, snapshotDate, snapshotSource,
			sourceID, sourceBranch, sourceRevision,
		)
		if err != nil {
			return err
		}
		mfCount++

		// Insert activities only in full mode
		if b.fullMode && oc != nil {
			actCount += b.insertFlowActivities(actStmt, oc, string(mf.ID()), qualifiedName, moduleName,
				projectID, projectName, snapshotID, snapshotDate, snapshotSource,
				sourceID, sourceBranch, sourceRevision)
		}
	}

	// Process nanoflows
	for _, nf := range nfs {
		if nf == nil {
			continue
		}
		moduleID := b.hierarchy.findModuleID(model.ID(nf.ID()))
		moduleName := b.hierarchy.getModuleName(moduleID)
		qualifiedName := moduleName + "." + nf.Name()

		oc := flowObjectCollection(nf.ObjectCollection())
		paramCount := countFlowParameters(oc)
		activityCount := countFlowActivities(oc)
		complexity := calculateFlowComplexity(oc)
		returnType := strings.TrimSpace(nf.ReturnType())

		_, err = mfStmt.Exec(
			string(nf.ID()),
			nf.Name(),
			qualifiedName,
			moduleName,
			moduleName, // Folder
			"NANOFLOW",
			nf.Documentation(),
			returnType,
			paramCount,
			activityCount,
			complexity,
			nf.Excluded(),
			projectID, projectName, snapshotID, snapshotDate, snapshotSource,
			sourceID, sourceBranch, sourceRevision,
		)
		if err != nil {
			return err
		}
		nfCount++

		// Insert activities only in full mode
		if b.fullMode && oc != nil {
			actCount += b.insertFlowActivities(actStmt, oc, string(nf.ID()), qualifiedName, moduleName,
				projectID, projectName, snapshotID, snapshotDate, snapshotSource,
				sourceID, sourceBranch, sourceRevision)
		}
	}

	b.report("Microflows", mfCount)
	b.report("Nanoflows", nfCount)
	if b.fullMode {
		b.report("Activities", actCount)
	}
	return nil
}

// insertFlowActivities iterates a gen ObjectCollection and inserts one
// row per activity into the activities table. Returns the number of
// rows successfully inserted.
func (b *Builder) insertFlowActivities(actStmt *sql.Stmt, oc *genMf.MicroflowObjectCollection,
	flowID, flowQN, moduleName,
	projectID, projectName, snapshotID, snapshotDate, snapshotSource, sourceID, sourceBranch, sourceRevision string) int {

	if oc == nil {
		return 0
	}
	count := 0
	for seq, obj := range oc.ObjectsItems() {
		activityType := getMicroflowObjectType(obj)
		activityName := activityType
		caption := "Activity"
		entityRef := ""
		actionType := ""
		serviceRef := ""
		actionRef := ""

		if act, ok := obj.(*genMf.ActionActivity); ok {
			inner := act.Action()
			if inner != nil {
				actionType = getMicroflowActionType(inner)
				activityName = actionType

				switch a := inner.(type) {
				case *genMf.CreateObjectAction:
					entityRef = a.EntityQualifiedName()
				case *genMf.CallExternalAction:
					serviceRef = a.ConsumedODataServiceQualifiedName()
					actionRef = a.Name()
				}
			}
		}

		_, err := actStmt.Exec(
			string(obj.ID()),
			activityName,
			caption,
			activityType,
			seq+1,
			flowID,
			flowQN,
			moduleName,
			moduleName,
			entityRef,
			actionType,
			serviceRef,
			actionRef,
			"",
			projectID, projectName, snapshotID, snapshotDate, snapshotSource,
			sourceID, sourceBranch, sourceRevision,
		)
		if err != nil {
			continue
		}
		count++
	}
	return count
}

// flowObjectCollection unwraps a Microflow/Nanoflow ObjectCollection
// element.Element to its concrete *genMf.MicroflowObjectCollection.
// Returns nil if the element is missing or of an unexpected type.
func flowObjectCollection(e element.Element) *genMf.MicroflowObjectCollection {
	if e == nil {
		return nil
	}
	oc, _ := e.(*genMf.MicroflowObjectCollection)
	return oc
}

// getMicroflowObjectType returns the type name for a microflow object.
// Type names mirror the legacy sdk-typed catalog so downstream
// consumers (`activities.ActivityType` filters etc) keep working
// across the migration.
func getMicroflowObjectType(obj element.Element) string {
	switch obj.(type) {
	case *genMf.ActionActivity:
		return "ActionActivity"
	case *genMf.StartEvent:
		return "StartEvent"
	case *genMf.EndEvent:
		return "EndEvent"
	case *genMf.ExclusiveSplit:
		return "ExclusiveSplit"
	case *genMf.InheritanceSplit:
		return "InheritanceSplit"
	case *genMf.ExclusiveMerge:
		return "ExclusiveMerge"
	case *genMf.LoopedActivity:
		return "LoopedActivity"
	case *genMf.Annotation:
		return "Annotation"
	case *genMf.BreakEvent:
		return "BreakEvent"
	case *genMf.ContinueEvent:
		return "ContinueEvent"
	case *genMf.ErrorEvent:
		return "ErrorEvent"
	default:
		return "MicroflowObject"
	}
}

// getMicroflowActionType returns the type name for a microflow action.
// Returned names use the legacy sdk-typed action class names (e.g.
// "ClosePageAction", not gen's "CloseFormAction") so that catalog
// rows remain stable for SQL consumers and lint rules; the gen→legacy
// alias map below mirrors the BSON storage-name table from CLAUDE.md.
func getMicroflowActionType(action element.Element) string {
	switch action.(type) {
	case *genMf.CreateObjectAction:
		return "CreateObjectAction"
	case *genMf.ChangeObjectAction:
		return "ChangeObjectAction"
	case *genMf.RetrieveAction:
		return "RetrieveAction"
	case *genMf.MicroflowCallAction:
		return "MicroflowCallAction"
	case *genMf.JavaActionCallAction:
		return "JavaActionCallAction"
	case *genMf.ShowMessageAction:
		return "ShowMessageAction"
	case *genMf.LogMessageAction:
		return "LogMessageAction"
	case *genMf.ValidationFeedbackAction:
		return "ValidationFeedbackAction"
	case *genMf.ChangeVariableAction:
		return "ChangeVariableAction"
	case *genMf.CreateVariableAction:
		return "CreateVariableAction"
	case *genMf.AggregateListAction:
		return "AggregateListAction"
	case *genMf.ListOperationAction:
		return "ListOperationAction"
	case *genMf.CastAction:
		return "CastAction"
	case *genMf.DownloadFileAction:
		return "DownloadFileAction"
	// Gen renames — the legacy class names are the catalog contract.
	case *genMf.CloseFormAction:
		return "ClosePageAction"
	case *genMf.ShowPageAction:
		return "ShowPageAction"
	case *genMf.CallExternalAction:
		return "CallExternalAction"
	case *genMf.DeleteAction:
		return "DeleteObjectAction"
	case *genMf.CommitAction:
		return "CommitObjectsAction"
	case *genMf.RollbackAction:
		return "RollbackObjectAction"
	default:
		return "MicroflowAction"
	}
}

// countFlowParameters counts MicroflowParameter / MicroflowParameterObject
// elements at the top level of an ObjectCollection. Mirrors the legacy
// `len(mf.Parameters)` field — gen models the parameters as
// MicroflowParameter children of the ObjectCollection (not as a
// flat slice on the flow itself).
func countFlowParameters(oc *genMf.MicroflowObjectCollection) int {
	if oc == nil {
		return 0
	}
	count := 0
	for _, obj := range oc.ObjectsItems() {
		switch obj.(type) {
		case *genMf.MicroflowParameter, *genMf.MicroflowParameterObject:
			count++
		}
	}
	return count
}

// countFlowActivities counts business-meaningful activities, excluding
// structural elements (start/end, merges, parameters). Mirrors the
// legacy countMicroflowActivities/countNanoflowActivities.
func countFlowActivities(oc *genMf.MicroflowObjectCollection) int {
	if oc == nil {
		return 0
	}
	count := 0
	for _, obj := range oc.ObjectsItems() {
		switch obj.(type) {
		case *genMf.StartEvent, *genMf.EndEvent:
			// Don't count start/end events
		case *genMf.ExclusiveMerge:
			// Don't count merge nodes (they're structural)
		case *genMf.MicroflowParameter, *genMf.MicroflowParameterObject:
			// Don't count parameters (they're structural)
		default:
			count++
		}
	}
	return count
}

// calculateFlowComplexity computes McCabe cyclomatic complexity for a
// gen flow. Each conditional branch (ExclusiveSplit, InheritanceSplit,
// LoopedActivity, ErrorEvent) adds 1; baseline = 1 (the entry path).
// Loop bodies recurse so nested decisions also contribute.
func calculateFlowComplexity(oc *genMf.MicroflowObjectCollection) int {
	complexity := 1
	if oc == nil {
		return complexity
	}
	complexity += countDecisionPoints(oc.ObjectsItems())
	return complexity
}

// countDecisionPoints recursively counts decision points in a list of
// gen microflow objects, descending into LoopedActivity bodies.
func countDecisionPoints(objects []element.Element) int {
	count := 0
	for _, obj := range objects {
		switch a := obj.(type) {
		case *genMf.ExclusiveSplit:
			count++
		case *genMf.InheritanceSplit:
			count++
		case *genMf.LoopedActivity:
			count++
			if body := flowObjectCollection(a.ObjectCollection()); body != nil {
				count += countDecisionPoints(body.ObjectsItems())
			}
		case *genMf.ErrorEvent:
			count++
		}
	}
	return count
}

// Note: legacy `getDataTypeName(microflows.DataType)` is gone
// (Followup F1). The gen *Microflow.ReturnType() / *Nanoflow.ReturnType()
// methods return the formatted name string directly, removing the
// per-DataType-class switch the legacy code needed.
