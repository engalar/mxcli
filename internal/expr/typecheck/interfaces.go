// SPDX-License-Identifier: Apache-2.0

// Package typecheck implements SEM-03: expression type-mismatch detection.
// It checks that every expression's inferred TypeKind matches the TypeKind
// expected by the slot it fills (variable assignment, function argument, etc.).
package typecheck

import (
	"github.com/mendixlabs/mxcli/internal/expr/scan"
	"github.com/mendixlabs/mxcli/internal/expr/validate"
	"github.com/mendixlabs/mxcli/mdl/exprcheck"
)

// Inferrer infers the TypeKind of a parsed AST node.
type Inferrer interface {
	Infer(node exprcheck.RobustExpr, scope VarScope, cat AttrCatalog, funcs FuncReg) exprcheck.TypeKind
}

// VarScope resolves a microflow variable name to its TypeKind.
type VarScope interface {
	TypeOf(varName string) exprcheck.TypeKind
}

// AttrCatalog resolves entity attribute types and variable entity QNs.
type AttrCatalog interface {
	AttributeKind(entityQN, attrName string) (exprcheck.TypeKind, bool)
	EntityQNOf(varName string) string
}

// FuncReg provides the return TypeKind of a named built-in function.
type FuncReg interface {
	ReturnType(funcName string) (exprcheck.TypeKind, bool)
}

// SlotResolver resolves the expected TypeKind for an expression slot.
type SlotResolver interface {
	Expect(rec scan.ExprRecord, cat AttrCatalog) (exprcheck.TypeKind, bool)
}

// IndexReader is the subset of meta.IndexReader required by typecheck.
type IndexReader interface {
	AttributeKind(entityQN, attrName string) (exprcheck.TypeKind, bool)
	VarTypeKind(unitPath, varName string) exprcheck.TypeKind
	VarEntityQN(unitPath, varName string) string
	MicroflowParamKind(calleeQN, paramName string) (exprcheck.TypeKind, bool)
	MicroflowReturnKind(mfName string) (exprcheck.TypeKind, bool)
}

// Result is a type alias so callers use the shared ValidationResult type.
type Result = validate.ValidationResult
