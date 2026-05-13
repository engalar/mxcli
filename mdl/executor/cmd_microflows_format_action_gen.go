// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.2.a — Object Actions family formatters (gen-typed).
//
// This file implements the gen-typed counterpart to legacy
// `cmd_microflows_format_action.go`. It is invoked from
// `renderGenMicroflowBody` for every `*genMf.ActionActivity` node and
// formats six "object actions":
//
//   | gen Go type                  | BSON $Type           | MDL keyword          |
//   |------------------------------|----------------------|----------------------|
//   | *genMf.CreateObjectAction    | CreateChangeAction   | `$X = create E (…)`  |
//   | *genMf.ChangeObjectAction    | ChangeAction         | `change $X (…)`      |
//   | *genMf.DeleteAction          | DeleteAction         | `delete $X;`         |
//   | *genMf.CommitAction          | CommitAction         | `commit $X;`         |
//   | *genMf.RollbackAction        | RollbackAction       | `rollback $X;`       |
//   | *genMf.AggregateListAction   | AggregateAction      | `$X = sum($L);`      |
//
// The output strings are 1:1 with the legacy formatters in
// `cmd_microflows_format_action.go` so that the textual diff between
// the legacy and gen-typed paths is empty for these six action kinds.
//
// Activity kinds outside this family return "" — the caller in
// `cmd_microflows_show_gen.go` then emits the existing TODO placeholder
// so coverage rolls out one family at a time.

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// formatActivityGen formats a single gen activity node as an MDL
// statement. It returns "" when the activity is an ActionActivity whose
// inner Action is not yet supported, so the caller can decide whether
// to emit a placeholder or skip the line entirely. Non-ActionActivity
// nodes (Annotations, control-flow events) are out of scope and also
// return "".
func formatActivityGen(_ *ExecContext, obj element.Element) string {
	aa, ok := obj.(*genMf.ActionActivity)
	if !ok || aa == nil {
		return ""
	}
	return formatActionGen(aa.Action())
}

// formatActionGen dispatches the inner action of an ActionActivity to a
// per-type formatter. Returns "" for action kinds not yet covered by
// Stage 3.2.2.a so the caller falls back to a placeholder.
func formatActionGen(action element.Element) string {
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
