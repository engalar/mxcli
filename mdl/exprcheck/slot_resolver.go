// SPDX-License-Identifier: Apache-2.0

package exprcheck

// staticExpectations maps MDL slot paths to their expected expression type.
// Add a new entry whenever a new MDL statement slot is added to the executor.
// Slot paths mirror the AST node + field name, e.g. "IfStmt.Condition".
var staticExpectations = map[string]SlotConstraint{
	"IfStmt.Condition":         {Kind: KindBoolean},
	"WhileStmt.Condition":      {Kind: KindBoolean},
	"RetrieveStmt.LimitExpr":   {Kind: KindInteger},
	"RetrieveStmt.OffsetExpr":  {Kind: KindInteger},
	"ChangeItem.Value":         {Kind: KindUnknown, ResolveBy: "AttributeOf:Parent"},
	"CreateItem.Value":         {Kind: KindUnknown, ResolveBy: "AttributeOf:Parent"},
	"ReturnStmt.Value":         {Kind: KindUnknown, ResolveBy: "MicroflowReturn"},
	"CallArgument.Value":       {Kind: KindUnknown, ResolveBy: "TargetParameter"},
	"LogStmt.Message":          {Kind: KindString},
	"MfSetStmt.Value":          {Kind: KindUnknown, ResolveBy: "TargetVariable"},
	"DeclareStmt.InitialValue": {Kind: KindUnknown, ResolveBy: "DeclareType"},
}

type defaultSlotResolver struct{}

func DefaultSlotResolver() SlotResolver { return &defaultSlotResolver{} }

func (r *defaultSlotResolver) Expect(path string) (SlotConstraint, bool) {
	sc, ok := staticExpectations[path]
	return sc, ok
}
