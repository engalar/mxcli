// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.f1 — gen-typed simple-action adders.
//
// First instalment of the actions family: covers every per-statement
// adder in `cmd_microflows_builder_actions.go` whose construction is
// trivial (no MemberChange member-resolution logic, no XPath /
// retrieve branching, no enum-split topology). Specifically:
//
//   - addCreateVariableActionGen  — `declare $X T = expr;`
//   - addChangeVariableActionGen  — `set $X = expr;`
//   - addCommitActionGen          — `commit $X;`
//   - addDeleteActionGen          — `delete $X;`
//   - addRollbackActionGen        — `rollback $X;`
//   - addCastActionGen            — `[$Y = ]cast $X;` (uses
//                                    setRawBSONField for the
//                                    ObjectVariableName gap)
//   - addCreateListActionGen      — `$L = create list of E;`
//   - addAddToListActionGen       — `add $X to $L;`
//   - addRemoveFromListActionGen  — `remove $X from $L;`
//   - addAggregateListActionGen   — `$X = sum/count/avg/min/max/reduce(...);`
//
// What's NOT here (next sub-commits):
//
//   - addCreateObjectActionGen / addChangeObjectActionGen — need the
//     full MemberChange resolveMemberChange port (~200 LoC) ⇒ f2
//   - addRetrieveActionGen — DatabaseRetrieveSource +
//     AssociationRetrieveSource branching, sort+range+xpath ⇒ f2
//   - addEnumSplitGen / addInheritanceSplitGen / addStructured-
//     InheritanceSplitGen — split/merge constructors with branch
//     wiring, depend on addStatement dispatcher ⇒ f3 (or h with
//     control-flow)
//   - addListOperationActionGen — 13 sub-operations (head/tail/find/
//     filter/sort/union/intersect/subtract/contains/equals/findByAttr/
//     filterByAttr/listRange) ⇒ f3
//
// The shared `genActivityWrap` helper wraps a freshly-built gen
// action into an *ActionActivity at the current cursor and registers
// any empty `on error custom` handler — same shape as wrapActionGen
// in flowbuilder_workflow_gen.go but exposed as a free function so
// every adder in this file uses the same wrapping path.

package executor

import (
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDt "github.com/mendixlabs/mxcli/modelsdk/gen/datatypes"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// addCreateVariableActionGen emits a `declare $X T = expr;` activity.
// Mirrors the legacy flowBuilder.addCreateVariableAction body shape:
//
//   - resolves TypeEnumeration vs TypeEntity ambiguity (a bare
//     `Module.Name` parsed as enum may actually be an entity);
//   - registers the variable as declared so subsequent SET / change
//     statements pass validation;
//   - sets ErrorHandlingType per ehTypeGen (gen builder default).
func (fb *flowBuilderGen) addCreateVariableActionGen(s *ast.DeclareStmt) element.ID {
	declType := s.Type
	if declType.Kind == ast.TypeEnumeration && declType.EnumRef != nil && fb.backend != nil {
		legacy := &flowBuilder{backend: fb.backend}
		if legacy.isEntity(declType.EnumRef.Module, declType.EnumRef.Name) {
			declType = ast.DataType{Kind: ast.TypeEntity, EntityRef: declType.EnumRef}
		}
	}

	if fb.declaredVars != nil {
		fb.declaredVars[s.Variable] = declType.Kind.String()
	}

	action := genMf.NewCreateVariableAction()
	assignFreshID(action)
	action.SetErrorHandlingType(fb.ehTypeGen(nil))
	action.SetVariableName(s.Variable)
	if dt := convertASTToGenDataType(declType); dt != nil {
		action.SetVariableType(dt)
	}
	action.SetInitialValue(fb.exprToString(s.InitialValue))

	return fb.genActivityWrap(action, nil, "")
}

// addChangeVariableActionGen emits a `set $X = expr;` activity.
// Mirrors flowBuilder.addChangeVariableAction.
func (fb *flowBuilderGen) addChangeVariableActionGen(s *ast.MfSetStmt) element.ID {
	if !fb.isVariableDeclared(s.Target) {
		fb.addErrorWithExample(
			"variable '"+s.Target+"' is not declared",
			errorExampleDeclareVariable(s.Target))
	}

	action := genMf.NewChangeVariableAction()
	assignFreshID(action)
	action.SetErrorHandlingType(fb.ehTypeGen(nil))
	action.SetChangeVariableName(s.Target)
	action.SetValue(fb.exprToString(s.Value))

	return fb.genActivityWrap(action, nil, "")
}

// addCommitActionGen emits a `commit $X [with events] [refresh in client];`
// activity. Mirrors flowBuilder.addCommitAction.
func (fb *flowBuilderGen) addCommitActionGen(s *ast.MfCommitStmt) element.ID {
	action := genMf.NewCommitAction()
	assignFreshID(action)
	action.SetErrorHandlingType(fb.ehTypeGen(s.ErrorHandling))
	action.SetCommitVariableName(s.Variable)
	action.SetWithEvents(s.WithEvents)
	action.SetRefreshInClient(s.RefreshInClient)
	return fb.genActivityWrap(action, s.ErrorHandling, "")
}

// addDeleteActionGen emits a `delete $X;` activity.
// Mirrors flowBuilder.addDeleteAction.
func (fb *flowBuilderGen) addDeleteActionGen(s *ast.DeleteObjectStmt) element.ID {
	action := genMf.NewDeleteAction()
	assignFreshID(action)
	action.SetErrorHandlingType(fb.ehTypeGen(s.ErrorHandling))
	action.SetDeleteVariableName(s.Variable)
	return fb.genActivityWrap(action, s.ErrorHandling, "")
}

// addRollbackActionGen emits a `rollback $X [refresh in client];`
// activity. Mirrors flowBuilder.addRollbackAction.
func (fb *flowBuilderGen) addRollbackActionGen(s *ast.RollbackStmt) element.ID {
	action := genMf.NewRollbackAction()
	assignFreshID(action)
	action.SetRollbackVariableName(s.Variable)
	action.SetRefreshInClient(s.RefreshInClient)
	return fb.genActivityWrap(action, nil, "")
}

// addCastActionGen emits a `[$Y = ]cast $X;` activity. The gen
// CastAction lacks `SetObjectVariableName` — Stage 3.2.3.f0
// (setRawBSONField) bridges that gap by injecting an ad-hoc Property
// into the element, which the codec overlays on encode.
//
// Schema-gap tracking: CastAction.ObjectVariableName (string).
func (fb *flowBuilderGen) addCastActionGen(s *ast.CastObjectStmt) element.ID {
	action := genMf.NewCastAction()
	assignFreshID(action)
	action.SetOutputVariableName(s.OutputVariable)
	setRawBSONField(action, "ObjectVariableName", s.ObjectVariable)
	return fb.genActivityWrap(action, nil, "")
}

// addCreateListActionGen emits a `$L = create list of E;` activity.
// Registers the variable as `List of E` so subsequent ADD / REMOVE
// / loop / aggregate statements pass validation.
func (fb *flowBuilderGen) addCreateListActionGen(s *ast.CreateListStmt) element.ID {
	entityQN := ""
	if s.EntityType.Module != "" && s.EntityType.Name != "" {
		entityQN = s.EntityType.Module + "." + s.EntityType.Name
	}
	if fb.varTypes != nil && entityQN != "" {
		fb.varTypes[s.Variable] = "List of " + entityQN
	}

	action := genMf.NewCreateListAction()
	assignFreshID(action)
	action.SetOutputVariableName(s.Variable)
	action.SetEntityQualifiedName(entityQN)
	return fb.genActivityWrap(action, nil, "")
}

// addAddToListActionGen emits an `add $X to $L;` activity. Wraps a
// ChangeListAction with type "Add". When s.Value is nil but s.Item
// is set, the item name is rendered as `$Item` to match legacy.
func (fb *flowBuilderGen) addAddToListActionGen(s *ast.AddToListStmt) element.ID {
	value := fb.exprToString(s.Value)
	if value == "" && s.Item != "" {
		value = "$" + s.Item
	}
	action := genMf.NewChangeListAction()
	assignFreshID(action)
	action.SetType("Add")
	action.SetChangeVariableName(s.List)
	action.SetValue(value)
	return fb.genActivityWrap(action, nil, "")
}

// addRemoveFromListActionGen emits a `remove $X from $L;` activity.
// Wraps a ChangeListAction with type "Remove".
func (fb *flowBuilderGen) addRemoveFromListActionGen(s *ast.RemoveFromListStmt) element.ID {
	action := genMf.NewChangeListAction()
	assignFreshID(action)
	action.SetType("Remove")
	action.SetChangeVariableName(s.List)
	action.SetValue("$" + s.Item)
	return fb.genActivityWrap(action, nil, "")
}

// addAggregateListActionGen emits a `$X = <fn>(...)` activity. The
// aggregate function maps directly to the gen `AggregateFunction`
// enum (Count / Sum / Average / Minimum / Maximum / Reduce). When an
// expression is supplied it overrides the attribute path; otherwise
// the attribute path is built from the input variable's list element
// type so the action carries `Module.Entity.Attribute` rather than
// the bare attribute name.
func (fb *flowBuilderGen) addAggregateListActionGen(s *ast.AggregateListStmt) element.ID {
	function := ""
	switch s.Operation {
	case ast.AggregateCount:
		function = "Count"
	case ast.AggregateSum:
		function = "Sum"
	case ast.AggregateAverage:
		function = "Average"
	case ast.AggregateMinimum:
		function = "Minimum"
	case ast.AggregateMaximum:
		function = "Maximum"
	case ast.AggregateReduce:
		function = "Reduce"
	default:
		return ""
	}

	action := genMf.NewAggregateListAction()
	assignFreshID(action)
	action.SetInputListVariableName(s.InputVariable)
	action.SetOutputVariableName(s.OutputVariable)
	action.SetAggregateFunction(function)

	if s.IsExpression && s.Expression != nil {
		action.SetUseExpression(true)
		action.SetExpression(expressionToString(s.Expression))
	} else if s.Attribute != "" && fb.varTypes != nil {
		listType := fb.varTypes[s.InputVariable]
		if after, ok := strings.CutPrefix(listType, "List of "); ok {
			action.SetAttributeQualifiedName(after + "." + s.Attribute)
		}
	}

	return fb.genActivityWrap(action, nil, "")
}

// genActivityWrap is the shared activity-wrapping helper for the
// per-statement adders in this file. Functionally identical to
// flowBuilderGen.wrapActionGen (workflow file), exposed as a free
// receiver method so the actions adders don't reach across files.
//
// The third argument is the `outputVar` carried into the empty-error-
// handler register helper — many adders pass "" because their stmt
// has no output variable; CreateObject/ChangeObject (f2) will use the
// real variable name to enable skip-propagation through the queue.
func (fb *flowBuilderGen) genActivityWrap(action element.Element, errorHandling *ast.ErrorHandlingClause, outputVar string) element.ID {
	activity := genMf.NewActionActivity()
	id := assignFreshID(activity)
	activity.SetRelativeMiddlePoint(layoutPos(fb.posX, fb.posY))
	activity.SetSize(layoutSize(ActivityWidth, ActivityHeight))
	activity.SetAutoGenerateCaption(true)
	activity.SetAction(action)

	fb.objects = append(fb.objects, activity)
	fb.posX += fb.spacing

	if isEmptyCustomErrorHandlerGen(errorHandling) {
		fb.registerEmptyCustomErrorHandlerWithSkipGen(id, errorHandling, outputVar)
	}
	// TODO Stage 3.2.3.h: emit non-empty error-handler bodies via
	// finishCustomErrorHandlerGen → addErrorHandlerFlowGen.

	return id
}

// convertASTToGenDataType maps an AST DataType to the gen DataTypes
// element the gen Microflow / nanoflow / variable types expect.
// Mirrors `convertASTToMicroflowDataType` (cmd_microflows_helpers.go)
// over the gen `datatypes` package.
//
// Limitations vs legacy:
//
//   - gen has no LongType — TypeLong falls back to IntegerType
//     (matches what Mendix surfaces in the BSON for legacy LongType
//     integers in 10.x roundtrips; integration test will catch any
//     regressions).
//   - EntityType / ListType reference entities by qualified name; the
//     legacy entityResolver lookup (which translated qualified names
//     to internal IDs) is not needed because gen ObjectType /
//     ListType use BY_NAME_REFERENCE storage exclusively.
//
// Returns nil for unknown / void kinds — caller should treat that as
// "leave the type unset" so the BSON omits the field rather than
// emit a stray empty document.
func convertASTToGenDataType(dt ast.DataType) element.Element {
	switch dt.Kind {
	case ast.TypeBoolean:
		t := genDt.NewBooleanType()
		assignFreshID(t)
		return t
	case ast.TypeInteger, ast.TypeLong:
		t := genDt.NewIntegerType()
		assignFreshID(t)
		return t
	case ast.TypeDecimal:
		t := genDt.NewDecimalType()
		assignFreshID(t)
		return t
	case ast.TypeString:
		t := genDt.NewStringType()
		assignFreshID(t)
		return t
	case ast.TypeDateTime:
		t := genDt.NewDateTimeType()
		assignFreshID(t)
		return t
	case ast.TypeBinary:
		t := genDt.NewBinaryType()
		assignFreshID(t)
		return t
	case ast.TypeEntity:
		if dt.EntityRef == nil {
			return nil
		}
		t := genDt.NewObjectType()
		assignFreshID(t)
		t.SetEntityQualifiedName(dt.EntityRef.Module + "." + dt.EntityRef.Name)
		return t
	case ast.TypeListOf:
		if dt.EntityRef == nil {
			return nil
		}
		t := genDt.NewListType()
		assignFreshID(t)
		t.SetEntityQualifiedName(dt.EntityRef.Module + "." + dt.EntityRef.Name)
		return t
	case ast.TypeEnumeration:
		if dt.EnumRef == nil {
			return nil
		}
		t := genDt.NewEnumerationType()
		assignFreshID(t)
		t.SetEnumerationQualifiedName(dt.EnumRef.Module + "." + dt.EnumRef.Name)
		return t
	case ast.TypeVoid:
		t := genDt.NewVoidType()
		assignFreshID(t)
		return t
	}
	return nil
}
