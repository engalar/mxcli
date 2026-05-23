// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.2.a — Object Actions family formatters (gen-typed).
// Stage 3.2.2.b — Form Actions family formatters (gen-typed).
// Stage 3.2.2.c — List operations family formatters (gen-typed).
// Stage 3.2.2.d — Microflow/Java/JavaScript call action family formatters
//                 (gen-typed). See `cmd_microflows_format_calls_gen.go` for
//                 the per-type implementations and parameter-value helpers.
// Stage 3.2.2.e — Variable / Expression / Data family formatters
//                 (gen-typed). See `cmd_microflows_format_data_gen.go`
//                 for Cast / CreateVariable / ChangeVariable / Retrieve /
//                 LogMessage / DownloadFile / ValidationFeedback.
// Stage 3.2.2.f — External integration family formatters (gen-typed).
//                 See `cmd_microflows_format_external_gen.go` for
//                 CallExternal / RestCall / RestOperationCall /
//                 ExecuteDatabaseQuery / ImportXml / ExportXml /
//                 TransformJson / WebServiceCall.
// Stage 3.2.2.g — Workflow / Misc family formatters (gen-typed).
//                 See `cmd_microflows_format_workflow_gen.go` for
//                 GetWorkflowData / WorkflowCall / GetWorkflows /
//                 GetWorkflowActivityRecords / WorkflowOperation
//                 (incl. Abort/Continue/Pause/Restart/Retry/Unpause/
//                 Resume sub-ops) / SetTaskOutcome / OpenUserTask /
//                 NotifyWorkflow / OpenWorkflow / LockWorkflow /
//                 UnlockWorkflow / GenerateJumpToOptions /
//                 ApplyJumpToOption. Stage 3.2.2 complete.
//
// This file implements the gen-typed counterpart to legacy
// `cmd_microflows_format_action.go`. It is invoked from
// `renderGenMicroflowBody` for every `*genMf.ActionActivity` node and
// formats the action kinds listed below.
//
// Object Actions (Stage 3.2.2.a):
//   | gen Go type                  | BSON $Type           | MDL keyword          |
//   |------------------------------|----------------------|----------------------|
//   | *genMf.CreateObjectAction    | CreateChangeAction   | `$X = create E (…)`  |
//   | *genMf.ChangeObjectAction    | ChangeAction         | `change $X (…)`      |
//   | *genMf.DeleteAction          | DeleteAction         | `delete $X;`         |
//   | *genMf.CommitAction          | CommitAction         | `commit $X;`         |
//   | *genMf.RollbackAction        | RollbackAction       | `rollback $X;`       |
//   | *genMf.AggregateListAction   | AggregateAction      | `$X = sum($L);`      |
//
// Form Actions (Stage 3.2.2.b):
//   | gen Go type                  | BSON $Type           | MDL keyword          |
//   |------------------------------|----------------------|----------------------|
//   | *genMf.ShowPageAction        | ShowFormAction       | `show page X(…);`    |
//   | *genMf.CloseFormAction       | CloseFormAction      | `close page;`        |
//   | *genMf.ShowHomePageAction    | ShowHomePageAction   | `show home page;`    |
//   | *genMf.ShowMessageAction     | ShowMessageAction    | `show message …;`    |
//
// List Operations (Stage 3.2.2.c):
//   | gen Go type                  | BSON $Type            | MDL keyword                |
//   |------------------------------|-----------------------|----------------------------|
//   | *genMf.CreateListAction      | CreateListAction      | `$L = create list of E;`   |
//   | *genMf.ChangeListAction      | ChangeListAction      | `add/remove/clear/set …;`  |
//   | *genMf.ListOperationAction   | ListOperationsAction  | `$X = head/tail/find(…);`  |
//
// ListOperationAction.Operation() dispatches to one of the gen
// list-operation primitives below (all live in `formatListOperationGen`):
//
//   | gen Go type           | BSON $Type            | Maps to legacy           |
//   |-----------------------|-----------------------|--------------------------|
//   | *genMf.Head           | Microflows$Head           | HeadOperation            |
//   | *genMf.Tail           | Microflows$Tail           | TailOperation            |
//   | *genMf.FindByExpression | Microflows$FindByExpression | FindOperation       |
//   | *genMf.Find           | Microflows$Find           | FindByAttributeOperation |
//   | *genMf.FilterByExpression | Microflows$FilterByExpression | FilterOperation |
//   | *genMf.Filter         | Microflows$Filter         | FilterByAttributeOperation |
//   | *genMf.Sort           | Microflows$Sort           | SortOperation            |
//   | *genMf.Union          | Microflows$Union          | UnionOperation           |
//   | *genMf.Intersect      | Microflows$Intersect      | IntersectOperation       |
//   | *genMf.Subtract       | Microflows$Subtract       | SubtractOperation        |
//   | *genMf.Contains       | Microflows$Contains       | ContainsOperation        |
//   | *genMf.ListEquals     | Microflows$ListEquals     | EqualsOperation          |
//   | *genMf.ListRange      | Microflows$ListRange      | ListRangeOperation       |
//
// The output strings are 1:1 with the legacy formatters in
// `cmd_microflows_format_action.go` so that the textual diff between
// the legacy and gen-typed paths is empty for these action kinds.
//
// Activity kinds outside this family return "" — the caller in
// `cmd_microflows_show_gen.go` then emits the existing TODO placeholder
// so coverage rolls out one family at a time.

package executor

import (
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDC "github.com/mendixlabs/mxcli/modelsdk/gen/databaseconnector"
	genDM "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	genTx "github.com/mendixlabs/mxcli/modelsdk/gen/texts"
)

// formatActivityGen formats a single gen activity node as an MDL
// statement. It returns "" when the activity is an ActionActivity whose
// inner Action is not yet supported, so the caller can decide whether
// to emit a placeholder or skip the line entirely. Non-ActionActivity
// nodes (Annotations, control-flow events) are out of scope and also
// return "".
func formatActivityGen(ctx *ExecContext, obj element.Element) string {
	aa, ok := obj.(*genMf.ActionActivity)
	if !ok || aa == nil {
		return ""
	}
	return formatActionGen(ctx, aa.Action())
}

// formatActionGen dispatches the inner action of an ActionActivity to a
// per-type formatter. Returns "" for action kinds not yet covered by
// Stage 3.2.2.{a,b,c,d,e,f,g} so the caller falls back to a placeholder.
//
// `ctx` is required by the data family's RetrieveAction path (XPath
// enum enrichment + reverse-association detection); other formatters
// ignore it.
func formatActionGen(ctx *ExecContext, action element.Element) string {
	if action == nil {
		return "-- Empty action"
	}
	switch a := action.(type) {
	case *genMf.CreateObjectAction:
		return formatCreateObjectActionGen(a)
	case *genMf.ChangeObjectAction:
		return formatChangeObjectActionGen(a)
	case *genMf.DeleteAction:
		return formatDeleteActionGen(a)
	case *genMf.CommitAction:
		return formatCommitActionGen(a)
	case *genMf.RollbackAction:
		return formatRollbackActionGen(a)
	case *genMf.AggregateListAction:
		return formatAggregateListActionGen(a)
	case *genMf.ShowPageAction:
		return formatShowPageActionGen(a)
	case *genMf.CloseFormAction:
		return formatCloseFormActionGen(a)
	case *genMf.ShowHomePageAction:
		return formatShowHomePageActionGen(a)
	case *genMf.ShowMessageAction:
		return formatShowMessageActionGen(a)
	case *genMf.CreateListAction:
		return formatCreateListActionGen(a)
	case *genMf.ChangeListAction:
		return formatChangeListActionGen(a)
	case *genMf.ListOperationAction:
		return formatListOperationActionGen(a)
	case *genMf.MicroflowCallAction:
		return formatMicroflowCallActionGen(a)
	case *genMf.NanoflowCallAction:
		return formatNanoflowCallActionGen(a)
	case *genMf.JavaActionCallAction:
		return formatJavaActionCallActionGen(a)
	case *genMf.JavaScriptActionCallAction:
		return formatJavaScriptActionCallActionGen(a)
	// Stage 3.2.2.e — Variable / Expression / Data family.
	case *genMf.CastAction:
		return formatCastActionGen(a)
	case *genMf.CreateVariableAction:
		return formatCreateVariableActionGen(a)
	case *genMf.ChangeVariableAction:
		return formatChangeVariableActionGen(a)
	case *genMf.RetrieveAction:
		return formatRetrieveActionGen(ctx, a)
	case *genMf.LogMessageAction:
		return formatLogMessageActionGen(a)
	case *genMf.DownloadFileAction:
		return formatDownloadFileActionGen(a)
	case *genMf.ValidationFeedbackAction:
		return formatValidationFeedbackActionGen(a)
	// Stage 3.2.2.f — External integration family.
	case *genMf.CallExternalAction:
		return formatCallExternalActionGen(a)
	case *genMf.RestCallAction:
		return formatRestCallActionGen(a)
	case *genMf.RestOperationCallAction:
		return formatRestOperationCallActionGen(a)
	case *genDC.ExecuteDatabaseQueryAction:
		return formatExecuteDatabaseQueryActionGen(a)
	case *genMf.ImportXmlAction:
		return formatImportXmlActionGen(a)
	case *genMf.ExportXmlAction:
		return formatExportXmlActionGen(a)
	case *genMf.TransformJsonAction:
		return formatTransformJsonActionGen(a)
	case *genMf.WebServiceCallAction:
		return formatWebServiceCallActionGen(a)
	// Stage 3.2.2.g — Workflow / Misc family.
	case *genMf.GetWorkflowDataAction:
		return formatGetWorkflowDataActionGen(a)
	case *genMf.WorkflowCallAction:
		return formatWorkflowCallActionGen(a)
	case *genMf.GetWorkflowsAction:
		return formatGetWorkflowsActionGen(a)
	case *genMf.GetWorkflowActivityRecordsAction:
		return formatGetWorkflowActivityRecordsActionGen(a)
	case *genMf.WorkflowOperationAction:
		return formatWorkflowOperationActionGen(a)
	case *genMf.SetTaskOutcomeAction:
		return formatSetTaskOutcomeActionGen(a)
	case *genMf.OpenUserTaskAction:
		return formatOpenUserTaskActionGen(a)
	case *genMf.NotifyWorkflowAction:
		return formatNotifyWorkflowActionGen(a)
	case *genMf.OpenWorkflowAction:
		return formatOpenWorkflowActionGen(a)
	case *genMf.LockWorkflowAction:
		return formatLockWorkflowActionGen(a)
	case *genMf.UnlockWorkflowAction:
		return formatUnlockWorkflowActionGen(a)
	case *genMf.GenerateJumpToOptionsAction:
		return formatGenerateJumpToOptionsActionGen(a)
	case *genMf.ApplyJumpToOptionAction:
		return formatApplyJumpToOptionActionGen(a)
	default:
		return ""
	}
}

// formatCreateObjectActionGen emits `$Var = create Module.Entity (…);`.
// Mirrors legacy CreateObjectAction handling: bare attribute names,
// fully-qualified association names with same-module prefix stripped,
// fall-back default outputVar of "NewObject" when the source has none.
func formatCreateObjectActionGen(a *genMf.CreateObjectAction) string {
	entityName := strings.TrimSpace(a.EntityQualifiedName())
	if entityName == "" {
		entityName = "Entity"
	}
	outputVar := strings.TrimSpace(a.OutputVariableName())
	if outputVar == "" {
		outputVar = "NewObject"
	}
	entityModule := ""
	if parts := strings.SplitN(entityName, ".", 2); len(parts) == 2 {
		entityModule = parts[0]
	}
	members := collectInitialMemberAssignmentsGen(a.ItemsItems(), entityModule)
	if len(members) > 0 {
		return fmt.Sprintf("$%s = create %s (%s);", outputVar, entityName, strings.Join(members, ", "))
	}
	return fmt.Sprintf("$%s = create %s;", outputVar, entityName)
}

// collectInitialMemberAssignmentsGen renders MemberChange items the
// same way legacy formats them inside a `create` statement: attribute
// name only (last dot segment); association uses its qualified name,
// stripping the module prefix when it matches the create target's
// module so the rendered text round-trips through the visitor.
func collectInitialMemberAssignmentsGen(items []element.Element, entityModule string) []string {
	var out []string
	for _, item := range items {
		mc, ok := item.(*genMf.MemberChange)
		if !ok || mc == nil {
			continue
		}
		var name string
		if assoc := mc.AssociationQualifiedName(); assoc != "" {
			name = assoc
			if parts := strings.SplitN(assoc, ".", 2); len(parts) == 2 && parts[0] == entityModule {
				name = parts[1]
			}
		} else {
			name = mc.AttributeQualifiedName()
			if name == "" {
				continue
			}
			// Module.Entity.Attr → Attr.
			if parts := strings.Split(name, "."); len(parts) > 0 {
				name = parts[len(parts)-1]
			}
		}
		out = append(out, fmt.Sprintf("%s = %s", name, escapeExpressionValue(mc.Value())))
	}
	return out
}

// formatChangeObjectActionGen emits `change $Var (…) [refresh];`.
// Mirrors legacy ChangeObjectAction handling.
func formatChangeObjectActionGen(a *genMf.ChangeObjectAction) string {
	varName := strings.TrimSpace(a.ChangeVariableName())
	if varName == "" {
		varName = "Object"
	}
	members := collectChangeMemberAssignmentsGen(a.ItemsItems())
	refreshSuffix := ""
	if a.RefreshInClient() {
		refreshSuffix = " refresh"
	}
	if len(members) > 0 {
		return fmt.Sprintf("change $%s (%s)%s;", varName, strings.Join(members, ", "), refreshSuffix)
	}
	return fmt.Sprintf("change $%s%s;", varName, refreshSuffix)
}

// collectChangeMemberAssignmentsGen renders MemberChange items the same
// way legacy formats them inside a `change` statement: associations
// keep the full qualified name (legacy makes no module-prefix strip in
// the change path); attributes use only the last dot segment.
func collectChangeMemberAssignmentsGen(items []element.Element) []string {
	var out []string
	for _, item := range items {
		mc, ok := item.(*genMf.MemberChange)
		if !ok || mc == nil {
			continue
		}
		var name string
		if assoc := mc.AssociationQualifiedName(); assoc != "" {
			name = assoc
		} else {
			name = mc.AttributeQualifiedName()
			if name == "" {
				continue
			}
			if parts := strings.Split(name, "."); len(parts) > 0 {
				name = parts[len(parts)-1]
			}
		}
		out = append(out, fmt.Sprintf("%s = %s", name, escapeExpressionValue(mc.Value())))
	}
	return out
}

// formatDeleteActionGen emits `delete $Var;` (matches legacy verbatim).
func formatDeleteActionGen(a *genMf.DeleteAction) string {
	return fmt.Sprintf("delete $%s;", a.DeleteVariableName())
}

// formatCommitActionGen emits `commit $Var [with events] [refresh];`.
// Mirrors legacy CommitObjectsAction.
func formatCommitActionGen(a *genMf.CommitAction) string {
	varName := strings.TrimSpace(a.CommitVariableName())
	if varName == "" {
		varName = "Object"
	}
	suffix := ""
	if a.WithEvents() {
		suffix += " with events"
	}
	if a.RefreshInClient() {
		suffix += " refresh"
	}
	return fmt.Sprintf("commit $%s%s;", varName, suffix)
}

// formatRollbackActionGen emits `rollback $Var [refresh];`.
// Mirrors legacy RollbackObjectAction.
func formatRollbackActionGen(a *genMf.RollbackAction) string {
	if a.RefreshInClient() {
		return fmt.Sprintf("rollback $%s refresh;", a.RollbackVariableName())
	}
	return fmt.Sprintf("rollback $%s;", a.RollbackVariableName())
}

// formatAggregateListActionGen emits one of three forms depending on
// the action's flavour, mirroring legacy AggregateListAction:
//
//   - Expression-based: `$Out = sum($List, <expr>);`
//   - Attribute-based:  `$Out = sum($List.Attr);`
//   - Bare/count:       `$Out = count($List);`
func formatAggregateListActionGen(a *genMf.AggregateListAction) string {
	outputVar := strings.TrimSpace(a.OutputVariableName())
	if outputVar == "" {
		outputVar = "Result"
	}
	fn := strings.TrimSpace(a.AggregateFunction())
	if fn == "" {
		fn = "Count"
	}
	fnLower := strings.ToLower(fn)

	if a.UseExpression() && strings.TrimSpace(a.Expression()) != "" {
		return fmt.Sprintf("$%s = %s($%s, %s);", outputVar, fnLower, a.InputListVariableName(), a.Expression())
	}

	attrName := a.AttributeQualifiedName()
	if attrName != "" {
		if parts := strings.Split(attrName, "."); len(parts) > 0 {
			attrName = parts[len(parts)-1]
		}
	}
	if attrName != "" && fn != genMf.AggregateFunctionEnumCount {
		return fmt.Sprintf("$%s = %s($%s.%s);", outputVar, fnLower, a.InputListVariableName(), attrName)
	}
	return fmt.Sprintf("$%s = %s($%s);", outputVar, fnLower, a.InputListVariableName())
}

// ────────────────────────────────────────────────────────
// Stage 3.2.2.b — Form Actions family
// ────────────────────────────────────────────────────────

// formatShowPageActionGen emits `show page <PageQN>[(<param = arg, …>)];`.
// Mirrors legacy ShowPageAction handling but reads the page reference
// from the gen `PageSettings` element (BY_NAME_REFERENCE on `Form`),
// which collapses the legacy PageName / PageID branching into a single
// qualified-name source. The legacy PageID-based fallback (resolve
// container chain via `ctx.Backend.ListPages`) is not exercised here:
// gen `PageSettings.InitFromRaw` reads only `Form` and modern MPRs
// always carry the qualified name. When neither source resolves the
// page, we emit the same `UnknownPage` placeholder as legacy.
func formatShowPageActionGen(a *genMf.ShowPageAction) string {
	pageName := "UnknownPage"
	var mappings []element.Element
	if ps, ok := a.PageSettings().(*genPg.PageSettings); ok && ps != nil {
		if qn := strings.TrimSpace(ps.PageQualifiedName()); qn != "" {
			pageName = qn
		}
		mappings = ps.ParameterMappingsItems()
	}

	var params []string
	for _, m := range mappings {
		ppm, ok := m.(*genPg.PageParameterMapping)
		if !ok || ppm == nil {
			continue
		}
		paramName := lastDotSegment(ppm.ParameterQualifiedName())
		params = append(params, fmt.Sprintf("$%s = %s", paramName, ppm.Argument()))
	}

	paramStr := ""
	if len(params) > 0 {
		paramStr = "(" + strings.Join(params, ", ") + ")"
	}
	return fmt.Sprintf("show page %s%s;", pageName, paramStr)
}

// formatCloseFormActionGen emits `close page;` or `close page <N>;`.
// Mirrors legacy ClosePageAction handling: only the integer-valued
// `NumberOfPages` field is consulted (the newer `NumberOfPagesToClose`
// expression-form is not surfaced by legacy and is therefore ignored
// here for 1:1 parity).
func formatCloseFormActionGen(a *genMf.CloseFormAction) string {
	if n := a.NumberOfPages(); n > 1 {
		return fmt.Sprintf("close page %d;", n)
	}
	return "close page;"
}

// formatShowHomePageActionGen emits the constant `show home page;`.
// ShowHomePageAction has no surface-level parameters in legacy MDL,
// so the gen-typed formatter is a literal mirror.
func formatShowHomePageActionGen(_ *genMf.ShowHomePageAction) string {
	return "show home page;"
}

// formatShowMessageActionGen emits
// `show message <text> type <Type>[ objects [<expr>, …]];`.
//
// Mirrors legacy ShowMessageAction.Type/Template/TemplateParameters
// handling. Two subtle quirks of legacy preserved verbatim:
//
//   1. When the action has no Template (or an empty Translations map),
//      the rendered text is the literal three-character `'...'` —
//      already wrapped in single quotes. When a Template is present
//      we wrap the text via `mdlQuote` (escaping internal quotes).
//   2. The Template's text source prefers the `en_US` translation; any
//      other available translation is the fallback.
//
// In gen the translations live inside a Texts$Text element nested under
// the action's TextTemplate (not the legacy `model.Text.Translations`
// map), and TemplateParameters are PartList items on the TextTemplate
// (not a separate string slice on the action).
func formatShowMessageActionGen(a *genMf.ShowMessageAction) string {
	msgType := strings.TrimSpace(a.Type())
	if msgType == "" {
		msgType = genMf.ShowMessageTypeInformation
	}

	message := "'...'"
	tmpl, _ := a.Template().(*genMf.TextTemplate)
	if tmpl != nil {
		if text, ok := tmpl.Text().(*genTx.Text); ok && text != nil {
			if picked, found := pickTextTranslationGen(text); found {
				message = mdlQuote(picked)
			}
		}
	}

	result := fmt.Sprintf("show message %s type %s", message, msgType)

	if tmpl != nil {
		var params []string
		for _, arg := range tmpl.ArgumentsItems() {
			if ta, ok := arg.(*genMf.TemplateArgument); ok && ta != nil {
				if expr := strings.TrimSpace(ta.Expression()); expr != "" {
					params = append(params, expr)
				}
			}
		}
		if len(params) > 0 {
			result += " objects [" + strings.Join(params, ", ") + "]"
		}
	}

	return result + ";"
}

// ────────────────────────────────────────────────────────
// Stage 3.2.2.c — List operations family
// ────────────────────────────────────────────────────────

// formatCreateListActionGen emits `$Var = create list of Module.Entity;`.
// Mirrors legacy CreateListAction handling. Unlike the object variant,
// CreateListAction never carries member initialisers — it just declares
// an empty typed list. Falls back to bare "Entity" when no qualified
// name is set, matching legacy.
func formatCreateListActionGen(a *genMf.CreateListAction) string {
	entityName := strings.TrimSpace(a.EntityQualifiedName())
	if entityName == "" {
		entityName = "Entity"
	}
	return fmt.Sprintf("$%s = create list of %s;", a.OutputVariableName(), entityName)
}

// formatChangeListActionGen emits one of four forms depending on
// `Type`, mirroring legacy ChangeListAction:
//
//   - Add:    `add <value> to $Var;`
//   - Remove: `remove <value> from $Var;`
//   - Clear:  `clear $Var;`
//   - Set:    `set $Var = <value>;`
//
// Anything else (defensive default) renders as
// `change list $Var (<Type>);` to mirror legacy's fall-through.
func formatChangeListActionGen(a *genMf.ChangeListAction) string {
	varName := a.ChangeVariableName()
	switch a.Type() {
	case genMf.ChangeListActionTypeAdd:
		return fmt.Sprintf("add %s to $%s;", a.Value(), varName)
	case genMf.ChangeListActionTypeRemove:
		return fmt.Sprintf("remove %s from $%s;", a.Value(), varName)
	case genMf.ChangeListActionTypeClear:
		return fmt.Sprintf("clear $%s;", varName)
	case genMf.ChangeListActionTypeSet:
		return fmt.Sprintf("set $%s = %s;", varName, a.Value())
	default:
		return fmt.Sprintf("change list $%s (%s);", varName, a.Type())
	}
}

// formatListOperationActionGen wraps the inner Operation element with
// the action's output variable. Mirrors legacy ListOperationAction:
// missing OutputVariable defaults to "Result"; nil Operation gives the
// stable `$Var = list operation ...;` placeholder.
func formatListOperationActionGen(a *genMf.ListOperationAction) string {
	outputVar := a.OutputVariableName()
	if outputVar == "" {
		outputVar = "Result"
	}
	return formatListOperationGen(a.Operation(), outputVar)
}

// formatListOperationGen dispatches a list-operation primitive (Head,
// Tail, Find, Filter, Sort, Union, Intersect, Subtract, Contains,
// ListEquals, ListRange, FindByExpression, FilterByExpression) to its
// MDL form. The output is 1:1 with legacy `formatListOperation` so the
// migrated body can be diffed against the SDK path during the cutover.
//
// Note on naming: legacy's `FindOperation` / `FilterOperation` are the
// expression-only variants; in gen those live under
// `FindByExpression` / `FilterByExpression`. Conversely, legacy's
// `FindByAttributeOperation` / `FilterByAttributeOperation` (which
// also accept an attribute or association reference alongside the
// expression) are gen's bare `Find` / `Filter`. This file handles all
// four forms with the same MDL surface.
func formatListOperationGen(op element.Element, outputVar string) string {
	if op == nil {
		return fmt.Sprintf("$%s = list operation ...;", outputVar)
	}

	switch o := op.(type) {
	case *genMf.Head:
		return fmt.Sprintf("$%s = head($%s);", outputVar, o.ListVariableName())
	case *genMf.Tail:
		return fmt.Sprintf("$%s = tail($%s);", outputVar, o.ListVariableName())
	case *genMf.FindByExpression:
		return fmt.Sprintf("$%s = find($%s, %s);", outputVar, o.ListVariableName(), o.Expression())
	case *genMf.FilterByExpression:
		return fmt.Sprintf("$%s = filter($%s, %s);", outputVar, o.ListVariableName(), o.Expression())
	case *genMf.Find:
		fieldName := extractFieldNameGen(o.AttributeQualifiedName(), o.AssociationQualifiedName())
		expr := o.Expression()
		if fieldName != "" && expr != "" {
			return fmt.Sprintf("$%s = find($%s, %s = %s);", outputVar, o.ListVariableName(), fieldName, expr)
		} else if expr != "" {
			return fmt.Sprintf("$%s = find($%s, %s);", outputVar, o.ListVariableName(), expr)
		}
		return fmt.Sprintf("-- $%s = find($%s) — missing attribute/expression", outputVar, o.ListVariableName())
	case *genMf.Filter:
		fieldName := extractFieldNameGen(o.AttributeQualifiedName(), o.AssociationQualifiedName())
		expr := o.Expression()
		if fieldName != "" && expr != "" {
			return fmt.Sprintf("$%s = filter($%s, %s = %s);", outputVar, o.ListVariableName(), fieldName, expr)
		} else if expr != "" {
			return fmt.Sprintf("$%s = filter($%s, %s);", outputVar, o.ListVariableName(), expr)
		}
		return fmt.Sprintf("-- $%s = filter($%s) — missing attribute/expression", outputVar, o.ListVariableName())
	case *genMf.Sort:
		sortCols := collectSortColumnsGen(o)
		if len(sortCols) > 0 {
			return fmt.Sprintf("$%s = sort($%s, %s);", outputVar, o.ListVariableName(), strings.Join(sortCols, ", "))
		}
		return fmt.Sprintf("$%s = sort($%s);", outputVar, o.ListVariableName())
	case *genMf.Union:
		return fmt.Sprintf("$%s = union($%s, $%s);", outputVar, o.ListVariableName(), o.SecondListOrObjectVariableName())
	case *genMf.Intersect:
		return fmt.Sprintf("$%s = intersect($%s, $%s);", outputVar, o.ListVariableName(), o.SecondListOrObjectVariableName())
	case *genMf.Subtract:
		return fmt.Sprintf("$%s = subtract($%s, $%s);", outputVar, o.ListVariableName(), o.SecondListOrObjectVariableName())
	case *genMf.Contains:
		return fmt.Sprintf("$%s = contains($%s, $%s);", outputVar, o.ListVariableName(), o.SecondListOrObjectVariableName())
	case *genMf.ListEquals:
		return fmt.Sprintf("$%s = equals($%s, $%s);", outputVar, o.ListVariableName(), o.SecondListOrObjectVariableName())
	case *genMf.ListRange:
		offset, limit := extractListRangeBoundsGen(o)
		if offset != "" && limit != "" {
			return fmt.Sprintf("$%s = range($%s, %s, %s);", outputVar, o.ListVariableName(), offset, limit)
		} else if offset != "" {
			return fmt.Sprintf("$%s = range($%s, %s);", outputVar, o.ListVariableName(), offset)
		} else if limit != "" {
			return fmt.Sprintf("$%s = range($%s, 0, %s);", outputVar, o.ListVariableName(), limit)
		}
		return fmt.Sprintf("$%s = range($%s);", outputVar, o.ListVariableName())
	default:
		return fmt.Sprintf("$%s = list operation %T;", outputVar, op)
	}
}

// extractFieldNameGen returns the short field name from a qualified
// attribute or association reference (e.g. "MyModule.Entity.Status" →
// "Status", "MyModule.Order_Customer" → "Order_Customer"). Mirrors
// legacy `extractFieldName` (preferring attribute over association,
// taking the substring after the last '.').
func extractFieldNameGen(attribute, association string) string {
	ref := attribute
	if ref == "" {
		ref = association
	}
	if ref == "" {
		return ""
	}
	parts := strings.Split(ref, ".")
	return parts[len(parts)-1]
}

// collectSortColumnsGen renders the SortItem list of a gen Sort op as
// the `Attr asc, Other desc, …` clause used inside `sort(…)`. Mirrors
// legacy SortOperation handling: each item shows the bare attribute
// name (last '.' segment) and the direction ("asc"/"desc"). When the
// attribute can't be determined, the placeholder "..." is used so the
// rendered text remains parseable.
//
// Two attribute sources are tried in order, matching the legacy parser
// (parseSortItems): the modern AttributeRef.AttributeQualifiedName,
// then the legacy AttributePath fallback.
func collectSortColumnsGen(s *genMf.Sort) []string {
	listEl, ok := s.SortItemList().(*genMf.SortItemList)
	if !ok || listEl == nil {
		return nil
	}
	var cols []string
	for _, it := range listEl.ItemsItems() {
		si, ok := it.(*genMf.SortItem)
		if !ok || si == nil {
			continue
		}
		dir := "asc"
		if si.SortOrder() == genMf.SortOrderEnumDescending {
			dir = "desc"
		}
		attrName := ""
		if ar, ok := si.AttributeRef().(*genDM.AttributeRef); ok && ar != nil {
			attrName = ar.AttributeQualifiedName()
		}
		if attrName == "" {
			attrName = si.AttributePath()
		}
		if attrName != "" {
			if parts := strings.Split(attrName, "."); len(parts) > 0 {
				attrName = parts[len(parts)-1]
			}
		}
		if attrName == "" {
			attrName = "..."
		}
		cols = append(cols, fmt.Sprintf("%s %s", attrName, dir))
	}
	return cols
}

// extractListRangeBoundsGen returns (offsetExpr, limitExpr) from a
// ListRange's nested CustomRange, or ("","") when the range has no
// CustomRange (e.g. a ConstantRange single-object form, which we treat
// the same as "no bounds" so the rendered output collapses to
// `$X = range($L);` — matching legacy's behaviour for that BSON shape).
func extractListRangeBoundsGen(r *genMf.ListRange) (string, string) {
	cr, ok := r.CustomRange().(*genMf.CustomRange)
	if !ok || cr == nil {
		return "", ""
	}
	return cr.OffsetExpression(), cr.LimitExpression()
}

// pickTextTranslationGen returns the text body of a Texts$Text, with
// the same precedence as legacy: prefer en_US, otherwise the first
// non-empty translation encountered. The bool is false if no
// translations are present at all (so the caller can keep the legacy
// `'...'` placeholder, which is intentionally pre-quoted and must not
// flow through mdlQuote a second time).
//
// gen's Texts$Text decodes its translations from the BSON key
// "Translations", but real Mendix MPRs store the array under "Items"
// (the legacy parser fixed this discrepancy in its hand-written
// `parseText` helper). When `TranslationsItems()` is empty we fall
// back to reading the "Items" array from the element's raw BSON.
func pickTextTranslationGen(t *genTx.Text) (string, bool) {
	items := t.TranslationsItems()
	var translations []translationPair
	if len(items) > 0 {
		for _, it := range items {
			tr, ok := it.(*genTx.Translation)
			if !ok || tr == nil {
				continue
			}
			translations = append(translations, translationPair{
				lang: tr.LanguageCode(),
				text: tr.Text(),
			})
		}
	}
	if len(translations) == 0 {
		// gen-incompleteness fallback: real MPR shape uses "Items".
		translations = readTextItemsFromRaw(t.Raw())
	}
	if len(translations) == 0 {
		return "", false
	}
	var firstAny string
	var foundAny bool
	for _, tr := range translations {
		if tr.lang == "en_US" {
			return tr.text, true
		}
		if !foundAny {
			firstAny = tr.text
			foundAny = true
		}
	}
	return firstAny, foundAny
}

// translationPair is the minimal lang/text shape used by
// pickTextTranslationGen so the gen-decoded and raw-BSON code paths
// can share the precedence loop.
type translationPair struct {
	lang string
	text string
}

// readTextItemsFromRaw decodes the "Items" array of a Texts$Text raw
// BSON document. The array is a versioned BSON array
// `[<int32 version>, <doc>, <doc>, …]`; each `<doc>` is a
// Texts$Translation with LanguageCode + Text string fields.
func readTextItemsFromRaw(raw []byte) []translationPair {
	if len(raw) == 0 {
		return nil
	}
	var doc bson.M
	if err := bson.Unmarshal(raw, &doc); err != nil {
		return nil
	}
	itemsRaw, ok := doc["Items"]
	if !ok {
		return nil
	}
	arr, ok := itemsRaw.(bson.A)
	if !ok {
		return nil
	}
	var out []translationPair
	for _, item := range arr {
		m, ok := item.(bson.M)
		if !ok {
			continue
		}
		lang, _ := m["LanguageCode"].(string)
		text, _ := m["Text"].(string)
		out = append(out, translationPair{lang: lang, text: text})
	}
	return out
}
