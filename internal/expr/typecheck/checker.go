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
		Field:    rec.Field,
		Raw:      rec.Raw,
		RuleID:   "SEM-03",
		Severity: "ERROR",
		Message:  fmt.Sprintf("Expression returns %s but slot expects %s.", KindName(actual), KindName(expected)),
		Fix:      FixSuggestion(actual, expected, rec.Raw),
	}}
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
