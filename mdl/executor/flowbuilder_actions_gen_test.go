// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.f1 — simple-action adder tests.
//
// Verifies field wiring on each of the 10 simple action adders + the
// shared genActivityWrap helper + convertASTToGenDataType. CreateObject /
// ChangeObject / Retrieve / split / list-operation are deferred to
// f2/f3 commits; this file exercises only the simple-construction
// adders that don't need MemberChange resolution or branching topology.

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDt "github.com/mendixlabs/mxcli/modelsdk/gen/datatypes"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

func newActionTestFb() *flowBuilderGen {
	return &flowBuilderGen{
		posX: 200, posY: 200, baseY: 200, spacing: HorizontalSpacing,
		varTypes:     map[string]string{},
		declaredVars: map[string]string{},
	}
}

func actionFromObjects(t *testing.T, fb *flowBuilderGen) element.Element {
	t.Helper()
	if len(fb.objects) != 1 {
		t.Fatalf("want exactly 1 object, got %d", len(fb.objects))
	}
	a, ok := fb.objects[0].(*genMf.ActionActivity)
	if !ok {
		t.Fatalf("object[0] = %T, want *ActionActivity", fb.objects[0])
	}
	return a.Action()
}

func TestAddCreateVariableActionGenSetsFieldsAndRegistersDeclared(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.DeclareStmt{
		Variable: "Counter",
		Type:     ast.DataType{Kind: ast.TypeInteger},
		InitialValue: &ast.LiteralExpr{
			Kind:  ast.LiteralInteger,
			Value: int64(0),
		},
	}
	id := fb.addCreateVariableActionGen(stmt)
	if id == "" {
		t.Fatal("expected non-empty ID")
	}
	if fb.declaredVars["Counter"] == "" {
		t.Fatal("Counter should be registered as declared")
	}
	act := actionFromObjects(t, fb).(*genMf.CreateVariableAction)
	if act.VariableName() != "Counter" {
		t.Fatalf("variable name = %q", act.VariableName())
	}
	if act.InitialValue() != "0\n" {
		t.Fatalf("initial value = %q, want 0\n", act.InitialValue())
	}
	if _, ok := act.VariableType().(*genDt.IntegerType); !ok {
		t.Fatalf("variable type = %T, want *IntegerType", act.VariableType())
	}
	if act.ErrorHandlingType() != "Rollback" {
		t.Fatalf("eh = %q, want Rollback", act.ErrorHandlingType())
	}
}

func TestAddChangeVariableActionGenReportsUndeclaredError(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.MfSetStmt{
		Target: "$Missing",
		Value:  &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true},
	}
	fb.addChangeVariableActionGen(stmt)
	errs := fb.GetErrors()
	if len(errs) == 0 {
		t.Fatal("expected validation error for undeclared variable")
	}
}

func TestAddChangeVariableActionGenSetsFields(t *testing.T) {
	fb := newActionTestFb()
	fb.declaredVars["X"] = "Boolean"
	stmt := &ast.MfSetStmt{
		Target: "X",
		Value:  &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true},
	}
	fb.addChangeVariableActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ChangeVariableAction)
	if act.ChangeVariableName() != "X" {
		t.Fatalf("change var = %q", act.ChangeVariableName())
	}
	if act.Value() != "true\n" {
		t.Fatalf("value = %q, want true\n", act.Value())
	}
}

func TestAddCommitActionGenAllFlags(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.MfCommitStmt{
		Variable:        "Order",
		WithEvents:      true,
		RefreshInClient: true,
	}
	fb.addCommitActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.CommitAction)
	if act.CommitVariableName() != "Order" {
		t.Fatalf("commit var = %q", act.CommitVariableName())
	}
	if !act.WithEvents() {
		t.Fatal("with events should be true")
	}
	if !act.RefreshInClient() {
		t.Fatal("refresh in client should be true")
	}
}

func TestAddDeleteActionGenSetsFields(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.DeleteObjectStmt{Variable: "Order"}
	fb.addDeleteActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.DeleteAction)
	if act.DeleteVariableName() != "Order" {
		t.Fatalf("delete var = %q", act.DeleteVariableName())
	}
}

func TestAddRollbackActionGenSetsFields(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.RollbackStmt{Variable: "Order", RefreshInClient: true}
	fb.addRollbackActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.RollbackAction)
	if act.RollbackVariableName() != "Order" {
		t.Fatalf("rollback var = %q", act.RollbackVariableName())
	}
	if !act.RefreshInClient() {
		t.Fatal("refresh in client should be true")
	}
}

func TestAddCastActionGenInjectsObjectVariableNameViaExtraField(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.CastObjectStmt{
		ObjectVariable: "Source",
		OutputVariable: "Cast",
	}
	fb.addCastActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.CastAction)
	if act.OutputVariableName() != "Cast" {
		t.Fatalf("output var = %q", act.OutputVariableName())
	}
	// ObjectVariableName is injected as an ad-hoc Property — verify
	// it lives in the property list.
	found := false
	for _, p := range act.Properties() {
		if p.Name() == "ObjectVariableName" {
			found = true
			wp, ok := p.(element.WritableProperty)
			if !ok || !wp.Dirty() {
				t.Fatal("ObjectVariableName property must be dirty")
			}
			if got := wp.BSONValue(); got != "Source" {
				t.Fatalf("ObjectVariableName value = %v, want Source", got)
			}
			break
		}
	}
	if !found {
		t.Fatal("ObjectVariableName ad-hoc property must be attached")
	}
}

func TestAddCreateListActionGenRegistersListType(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.CreateListStmt{
		Variable:   "Items",
		EntityType: ast.QualifiedName{Module: "Sales", Name: "Order"},
	}
	fb.addCreateListActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.CreateListAction)
	if act.OutputVariableName() != "Items" {
		t.Fatalf("output var = %q", act.OutputVariableName())
	}
	if act.EntityQualifiedName() != "Sales.Order" {
		t.Fatalf("entity QN = %q", act.EntityQualifiedName())
	}
	if fb.varTypes["Items"] != "List of Sales.Order" {
		t.Fatalf("varTypes[Items] = %q", fb.varTypes["Items"])
	}
}

func TestAddCreateListActionGen_NonPersistentEntity_ReportsError(t *testing.T) {
	fb := newActionTestFb()
	// Directly preset nonPersistentEntities, bypassing backend call (backend=nil returns false)
	fb.nonPersistentEntities = map[string]bool{"M.NonPersistDto": true}

	stmt := &ast.CreateListStmt{
		Variable:   "Items",
		EntityType: ast.QualifiedName{Module: "M", Name: "NonPersistDto"},
	}
	fb.addCreateListActionGen(stmt)

	if len(fb.errors) == 0 {
		t.Error("expected CE0053 error for non-persistent entity in create list, got none")
	}
	if len(fb.objects) != 0 {
		t.Error("expected no action created for non-persistent entity (should return early)")
	}
}

func TestAddCreateListActionGen_PersistentEntity_NoError(t *testing.T) {
	fb := newActionTestFb()
	fb.nonPersistentEntities = map[string]bool{"M.NonPersistDto": true}

	stmt := &ast.CreateListStmt{
		Variable:   "Items",
		EntityType: ast.QualifiedName{Module: "M", Name: "PersistDto"},
	}
	fb.addCreateListActionGen(stmt)

	if len(fb.errors) != 0 {
		t.Errorf("expected no error for persistent entity, got: %v", fb.errors)
	}
	if len(fb.objects) == 0 {
		t.Error("expected action to be created for persistent entity")
	}
}

func TestAddAddToListActionGenWithVariable(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.AddToListStmt{List: "Items", Item: "Order"}
	fb.addAddToListActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ChangeListAction)
	if act.ChangeVariableName() != "Items" {
		t.Fatalf("change var = %q", act.ChangeVariableName())
	}
	if act.Type() != "Add" {
		t.Fatalf("type = %q, want Add", act.Type())
	}
	if act.Value() != "$Order" {
		t.Fatalf("value = %q, want $Order", act.Value())
	}
}

func TestAddRemoveFromListActionGenSetsFields(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.RemoveFromListStmt{List: "Items", Item: "Order"}
	fb.addRemoveFromListActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ChangeListAction)
	if act.Type() != "Remove" {
		t.Fatalf("type = %q, want Remove", act.Type())
	}
	if act.Value() != "$Order" {
		t.Fatalf("value = %q, want $Order", act.Value())
	}
}

func TestAddAggregateListActionGenAttributePath(t *testing.T) {
	fb := newActionTestFb()
	fb.varTypes["Items"] = "List of Sales.Order"
	stmt := &ast.AggregateListStmt{
		Operation:      ast.AggregateSum,
		InputVariable:  "Items",
		OutputVariable: "Total",
		Attribute:      "Amount",
	}
	fb.addAggregateListActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.AggregateListAction)
	if act.AggregateFunction() != "Sum" {
		t.Fatalf("function = %q, want Sum", act.AggregateFunction())
	}
	if act.AttributeQualifiedName() != "Sales.Order.Amount" {
		t.Fatalf("attribute QN = %q, want Sales.Order.Amount", act.AttributeQualifiedName())
	}
	if act.UseExpression() {
		t.Fatal("UseExpression should be false for attribute path")
	}
}

func TestAddAggregateListActionGenExpressionMode(t *testing.T) {
	fb := newActionTestFb()
	fb.varTypes["Items"] = "List of Sales.Order"
	stmt := &ast.AggregateListStmt{
		Operation:      ast.AggregateReduce,
		InputVariable:  "Items",
		OutputVariable: "Total",
		IsExpression:   true,
		Expression:     &ast.LiteralExpr{Kind: ast.LiteralString, Value: "expr"},
	}
	fb.addAggregateListActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.AggregateListAction)
	if act.AggregateFunction() != "Reduce" {
		t.Fatalf("function = %q, want Reduce", act.AggregateFunction())
	}
	if !act.UseExpression() {
		t.Fatal("UseExpression should be true")
	}
	// The expression literal renders as `'expr'` (single-quoted string).
	if act.Expression() != "'expr'\n" {
		t.Fatalf("expression = %q, want 'expr'\n", act.Expression())
	}
}

func TestAddAggregateListActionGenUnknownOpReturnsEmpty(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.AggregateListStmt{Operation: ast.AggregateListOperationType(99)}
	if id := fb.addAggregateListActionGen(stmt); id != "" {
		t.Fatalf("unknown op should return empty ID, got %s", id)
	}
	if len(fb.objects) != 0 {
		t.Fatalf("unknown op should not emit any object, got %d", len(fb.objects))
	}
}

func TestGenActivityWrapAdvancesPosAndAttachesAction(t *testing.T) {
	fb := newActionTestFb()
	startX := fb.posX
	dummy := genMf.NewCommitAction()
	assignFreshID(dummy)
	id := fb.genActivityWrap(dummy, nil, "")
	if id == "" {
		t.Fatal("non-empty ID expected")
	}
	if fb.posX != startX+HorizontalSpacing {
		t.Fatalf("posX = %d, want %d", fb.posX, startX+HorizontalSpacing)
	}
	act := fb.objects[0].(*genMf.ActionActivity)
	if act.Action() == nil {
		t.Fatal("action must be attached")
	}
	if !act.AutoGenerateCaption() {
		t.Fatal("AutoGenerateCaption should default to true")
	}
}

func TestGenActivityWrapRegistersEmptyCustomHandlerWithSkipVar(t *testing.T) {
	fb := newActionTestFb()
	dummy := genMf.NewDeleteAction()
	assignFreshID(dummy)
	eh := &ast.ErrorHandlingClause{Type: ast.ErrorHandlingCustom}
	fb.genActivityWrap(dummy, eh, "OutVar")
	if fb.errorHandlerSource == "" {
		t.Fatal("with skip var, errorHandlerSource should be set")
	}
	if fb.errorHandlerSkipVar != "OutVar" {
		t.Fatalf("skipVar = %q, want OutVar", fb.errorHandlerSkipVar)
	}
}

func TestConvertASTToGenDataTypePrimitives(t *testing.T) {
	cases := []struct {
		kind ast.DataTypeKind
		want string // expected concrete type name for clarity
	}{
		{ast.TypeBoolean, "*datatypes.BooleanType"},
		{ast.TypeInteger, "*datatypes.IntegerType"},
		{ast.TypeLong, "*datatypes.IntegerType"}, // gen has no LongType
		{ast.TypeDecimal, "*datatypes.DecimalType"},
		{ast.TypeString, "*datatypes.StringType"},
		{ast.TypeDateTime, "*datatypes.DateTimeType"},
		{ast.TypeBinary, "*datatypes.BinaryType"},
		{ast.TypeVoid, "*datatypes.VoidType"},
	}
	for _, tc := range cases {
		t.Run(tc.kind.String(), func(t *testing.T) {
			elem := convertASTToGenDataType(ast.DataType{Kind: tc.kind})
			if elem == nil {
				t.Fatalf("kind %s returned nil", tc.kind)
			}
		})
	}
}

func TestConvertASTToGenDataTypeEntityListEnum(t *testing.T) {
	qn := &ast.QualifiedName{Module: "Sales", Name: "Order"}
	t.Run("entity", func(t *testing.T) {
		dt := convertASTToGenDataType(ast.DataType{Kind: ast.TypeEntity, EntityRef: qn})
		obj, ok := dt.(*genDt.ObjectType)
		if !ok {
			t.Fatalf("got %T, want *ObjectType", dt)
		}
		if obj.EntityQualifiedName() != "Sales.Order" {
			t.Fatalf("entity QN = %q", obj.EntityQualifiedName())
		}
	})
	t.Run("list", func(t *testing.T) {
		dt := convertASTToGenDataType(ast.DataType{Kind: ast.TypeListOf, EntityRef: qn})
		lt, ok := dt.(*genDt.ListType)
		if !ok {
			t.Fatalf("got %T, want *ListType", dt)
		}
		if lt.EntityQualifiedName() != "Sales.Order" {
			t.Fatalf("list entity QN = %q", lt.EntityQualifiedName())
		}
	})
	t.Run("enum", func(t *testing.T) {
		enumQN := &ast.QualifiedName{Module: "Sales", Name: "OrderStatus"}
		dt := convertASTToGenDataType(ast.DataType{Kind: ast.TypeEnumeration, EnumRef: enumQN})
		et, ok := dt.(*genDt.EnumerationType)
		if !ok {
			t.Fatalf("got %T, want *EnumerationType", dt)
		}
		if et.EnumerationQualifiedName() != "Sales.OrderStatus" {
			t.Fatalf("enum QN = %q", et.EnumerationQualifiedName())
		}
	})
}

func TestConvertASTToGenDataTypeMissingRefReturnsNil(t *testing.T) {
	if got := convertASTToGenDataType(ast.DataType{Kind: ast.TypeEntity}); got != nil {
		t.Fatalf("nil EntityRef should produce nil, got %T", got)
	}
	if got := convertASTToGenDataType(ast.DataType{Kind: ast.TypeListOf}); got != nil {
		t.Fatalf("nil EntityRef on list should produce nil, got %T", got)
	}
	if got := convertASTToGenDataType(ast.DataType{Kind: ast.TypeEnumeration}); got != nil {
		t.Fatalf("nil EnumRef should produce nil, got %T", got)
	}
}
