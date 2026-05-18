// SPDX-License-Identifier: Apache-2.0

package typecheck

import (
	"strings"

	"github.com/mendixlabs/mxcli/internal/expr/scan"
	"github.com/mendixlabs/mxcli/mdl/exprcheck"
)

func slotKey(rec scan.ExprRecord) string {
	return rec.UnitType + "/" + rec.Field
}

// staticSlots maps UnitType/Field → expected TypeKind for slots with fixed expectations.
// Note: ExpressionSplitCondition is intentionally absent — it can hold a Boolean expression
// OR an enumeration attribute for enum-based splits, so we cannot safely require Boolean.
var staticSlots = map[string]exprcheck.TypeKind{
	"Microflows$WhileLoopCondition/WhileExpression":  exprcheck.KindBoolean,
	"DomainModels$AccessRule/XPathConstraint":        exprcheck.KindBoolean,
	"Microflows$CustomRange/LimitExpression":         exprcheck.KindInteger,
	"Microflows$CustomRange/OffsetExpression":        exprcheck.KindInteger,
	"Microflows$TemplateParameter/Expression":        exprcheck.KindString,
}

type slotResolverFn func(rec scan.ExprRecord, cat AttrCatalog, idx IndexReader) (exprcheck.TypeKind, bool)

var dynamicSlotKeys = map[string]slotResolverFn{
	"Microflows$ChangeActionItem/Value":                  resolveAttrTarget,
	"Microflows$ChangeVariableAction/Value":              resolveVarTarget,
	"Microflows$CreateVariableAction/InitialValue":       resolveVarTarget,
	"Microflows$MicroflowCallParameterMapping/Argument":  resolveCallArgTarget,
	"Mappings$MicroflowCallParameterMappingImpl/Argument": resolveCallArgTarget,
	"Workflows$MicroflowCallParameterMapping/Expression": resolveCallArgTarget,
	"Microflows$EndEvent/ReturnValue":                    resolveMicroflowReturn,
}

type defaultSlotResolver struct{ idx IndexReader }

// NewSlotResolver returns a SlotResolver backed by the given index.
func NewSlotResolver(idx IndexReader) SlotResolver {
	return &defaultSlotResolver{idx: idx}
}

func (r *defaultSlotResolver) Expect(rec scan.ExprRecord, cat AttrCatalog) (exprcheck.TypeKind, bool) {
	key := slotKey(rec)

	if k, ok := staticSlots[key]; ok {
		return k, true
	}

	if fn, ok := dynamicSlotKeys[key]; ok {
		return fn(rec, cat, r.idx)
	}

	return exprcheck.KindUnknown, false
}

func resolveAttrTarget(rec scan.ExprRecord, cat AttrCatalog, _ IndexReader) (exprcheck.TypeKind, bool) {
	qn := rec.TargetAttrQN
	if qn == "" {
		return exprcheck.KindUnknown, false
	}
	last := strings.LastIndex(qn, ".")
	if last <= 0 {
		return exprcheck.KindUnknown, false
	}
	entityQN := qn[:last]
	attrName := qn[last+1:]
	return cat.AttributeKind(entityQN, attrName)
}

func resolveVarTarget(rec scan.ExprRecord, _ AttrCatalog, idx IndexReader) (exprcheck.TypeKind, bool) {
	if rec.TargetVarName == "" {
		return exprcheck.KindUnknown, false
	}
	kind := idx.VarTypeKind(rec.UnitPath, rec.TargetVarName)
	if kind == exprcheck.KindUnknown {
		return exprcheck.KindUnknown, false
	}
	return kind, true
}

func resolveCallArgTarget(rec scan.ExprRecord, _ AttrCatalog, idx IndexReader) (exprcheck.TypeKind, bool) {
	if rec.CalleeQN == "" || rec.ParamName == "" {
		return exprcheck.KindUnknown, false
	}
	return idx.MicroflowParamKind(rec.CalleeQN, rec.ParamName)
}

func resolveMicroflowReturn(_ scan.ExprRecord, _ AttrCatalog, _ IndexReader) (exprcheck.TypeKind, bool) {
	// TODO(SEM-03.2): add unitPath → bare MF name lookup to Index.
	return exprcheck.KindUnknown, false
}
