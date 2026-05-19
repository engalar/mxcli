// SPDX-License-Identifier: Apache-2.0

package typecheck

import (
	"fmt"

	"github.com/mendixlabs/mxcli/internal/expr/parse"
	"github.com/mendixlabs/mxcli/internal/expr/validate"
	"github.com/mendixlabs/mxcli/mdl/exprcheck"
)

// TypeChecker orchestrates all typecheck components to produce SEM-03 results.
type TypeChecker struct {
	infer Inferrer
	slots SlotResolver
	funcs FuncReg
	idx   IndexReader
}

// NewChecker builds a TypeChecker wired to the provided index.
// Returns nil if idx is nil (nil TypeChecker is safe to call Check() on).
func NewChecker(idx IndexReader) *TypeChecker {
	if idx == nil {
		return nil
	}
	return &TypeChecker{
		infer: NewInferrer(),
		slots: NewSlotResolver(idx),
		funcs: NewFuncReg(),
		idx:   idx,
	}
}

// Check runs SEM-03 type checking on a single ParseResult.
// Returns nil if no type mismatch is found or if checking is skipped.
func (tc *TypeChecker) Check(pr parse.ParseResult) []validate.ValidationResult {
	if tc == nil || pr.AST == nil {
		return nil
	}

	rec := pr.Record

	// Build per-call adapters with the current unitPath.
	scope := &varScopeAdapter{idx: tc.idx, unitPath: rec.UnitPath}
	cat := &attrCatalogAdapter{idx: tc.idx, unitPath: rec.UnitPath}

	// Resolve expected TypeKind for this slot.
	expected, ok := tc.slots.Expect(rec, cat)
	if !ok || expected == exprcheck.KindUnknown {
		return nil
	}

	// Infer actual TypeKind from the AST.
	actual := tc.infer.Infer(pr.AST, scope, cat, tc.funcs)
	if actual == exprcheck.KindUnknown {
		return nil
	}

	if Compatible(actual, expected) {
		return nil
	}

	return []validate.ValidationResult{{
		UnitID:   rec.UnitID,
		Project:  rec.Project,
		UnitType: rec.UnitType,
		UnitPath: rec.UnitPath,
		Field:    rec.Field,
		Raw:      rec.Raw,
		RuleID:   "SEM-03",
		Severity: "ERROR",
		Message:  fmt.Sprintf("Expression returns %s but slot expects %s.", KindName(actual), KindName(expected)),
		Fix:      FixSuggestion(actual, expected, rec.Raw),
	}}
}

// CheckStructural runs SEM-L01 and SEM-L02 on a single ParseResult.
// SEM-L01: attribute accessed on a List variable ($ListVar/attr).
// SEM-L02: 'id' is a reserved Mendix system attribute, cannot be read.
func (tc *TypeChecker) CheckStructural(pr parse.ParseResult) []validate.ValidationResult {
	if tc == nil || pr.AST == nil {
		return nil
	}
	rec := pr.Record
	scope := &varScopeAdapter{idx: tc.idx, unitPath: rec.UnitPath}
	var out []validate.ValidationResult
	structuralWalk(pr.AST, rec.UnitID, rec.Project, rec.UnitType, rec.UnitPath, rec.Field, rec.Raw, scope, &out)
	return out
}

// structuralWalk recursively visits AST nodes to detect SEM-L01 / SEM-L02.
func structuralWalk(node exprcheck.RobustExpr, unitID, project, unitType, unitPath, field, raw string, scope *varScopeAdapter, out *[]validate.ValidationResult) {
	if node == nil {
		return
	}
	switch n := node.(type) {
	case *exprcheck.AttributePathExpr:
		if len(n.Path) > 0 {
			varKind := scope.TypeOf(n.Variable)
			if varKind == exprcheck.KindList {
				*out = append(*out, validate.ValidationResult{
					UnitID: unitID, Project: project, UnitType: unitType,
					UnitPath: unitPath, Field: field, Raw: raw,
					RuleID:   "SEM-L01",
					Severity: "ERROR",
					Message:  fmt.Sprintf("Attribute '%s' accessed on List variable '$%s'. A List cannot be accessed like a single object.", n.Path[0], n.Variable),
					Fix:      "Change the Retrieve action to return a single object (set range to First 1) instead of a list.",
				})
			}
			for _, seg := range n.Path {
				if seg == "id" {
					*out = append(*out, validate.ValidationResult{
						UnitID: unitID, Project: project, UnitType: unitType,
						UnitPath: unitPath, Field: field, Raw: raw,
						RuleID:   "SEM-L02",
						Severity: "ERROR",
						Message:  fmt.Sprintf("'id' is a reserved Mendix system attribute and cannot be read in expressions (in '$%s/id').", n.Variable),
						Fix:      "Run 'mxcli hint E012' for fix options (return the object, or add an AutoNumber attribute).",
					})
					break
				}
			}
		}
	case *exprcheck.BinExpr:
		structuralWalk(n.L, unitID, project, unitType, unitPath, field, raw, scope, out)
		structuralWalk(n.R, unitID, project, unitType, unitPath, field, raw, scope, out)
	case *exprcheck.UnaryExpr:
		structuralWalk(n.Operand, unitID, project, unitType, unitPath, field, raw, scope, out)
	case *exprcheck.IfThenElseExpr:
		structuralWalk(n.Cond, unitID, project, unitType, unitPath, field, raw, scope, out)
		structuralWalk(n.Then, unitID, project, unitType, unitPath, field, raw, scope, out)
		structuralWalk(n.Else, unitID, project, unitType, unitPath, field, raw, scope, out)
	case *exprcheck.CallExpr:
		for _, arg := range n.Args {
			structuralWalk(arg, unitID, project, unitType, unitPath, field, raw, scope, out)
		}
	case *exprcheck.ParenExpr:
		structuralWalk(n.Inner, unitID, project, unitType, unitPath, field, raw, scope, out)
	}
}

// varScopeAdapter adapts IndexReader to VarScope for a specific microflow unit.
type varScopeAdapter struct {
	idx      IndexReader
	unitPath string
}

func (a *varScopeAdapter) TypeOf(varName string) exprcheck.TypeKind {
	return a.idx.VarTypeKind(a.unitPath, varName)
}

// attrCatalogAdapter adapts IndexReader to AttrCatalog for a specific microflow unit.
type attrCatalogAdapter struct {
	idx      IndexReader
	unitPath string
}

func (a *attrCatalogAdapter) AttributeKind(entityQN, attrName string) (exprcheck.TypeKind, bool) {
	return a.idx.AttributeKind(entityQN, attrName)
}

func (a *attrCatalogAdapter) EntityQNOf(varName string) string {
	return a.idx.VarEntityQN(a.unitPath, varName)
}
