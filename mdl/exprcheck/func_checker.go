// SPDX-License-Identifier: Apache-2.0

package exprcheck

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/exprcheck/hints"
)

type funcSig struct {
	args []TypeKind
	ret  TypeKind
}

var funcTable = map[string]funcSig{
	"length":       {args: []TypeKind{KindString}, ret: KindInteger},
	"toString":     {args: []TypeKind{KindAny}, ret: KindString},
	"parseInteger": {args: []TypeKind{KindString}, ret: KindInteger},
	"parseDecimal": {args: []TypeKind{KindString}, ret: KindDecimal},
	"trim":         {args: []TypeKind{KindString}, ret: KindString},
	"toUpperCase":  {args: []TypeKind{KindString}, ret: KindString},
	"toLowerCase":  {args: []TypeKind{KindString}, ret: KindString},
	"substring":    {args: []TypeKind{KindString, KindInteger, KindInteger}, ret: KindString},
	"find":         {args: []TypeKind{KindString, KindString}, ret: KindInteger},
	"contains":     {args: []TypeKind{KindString, KindString}, ret: KindBoolean},
	"startsWith":   {args: []TypeKind{KindString, KindString}, ret: KindBoolean},
	"endsWith":     {args: []TypeKind{KindString, KindString}, ret: KindBoolean},
}

// checkCallExpr returns hints for arity / arg-type mismatches against
// the built-in function signature table. User-defined microflow calls
// (CallExpr.Name containing a dot) are not checked here — they require
// catalog lookup.
func checkCallExpr(c *CallExpr, ctx Context) []Hint {
	sig, ok := funcTable[c.Name]
	if !ok {
		return nil
	}
	var out []Hint
	if len(c.Args) != len(sig.args) {
		out = append(out, Hint{
			Code: "E006", Slug: "func-arg-arity",
			Severity: hints.SeverityError,
			Where: hints.Location{
				File: ctx.File, Line: c.Pos().Line, Column: c.Pos().Column,
				Microflow: ctx.Microflow,
				Context:   fmt.Sprintf("call to %s()", c.Name),
			},
			YouWrote: fmt.Sprintf("%s(...) with %d arguments", c.Name, len(c.Args)),
			Problem:  fmt.Sprintf("%s() expects %d argument(s), got %d.", c.Name, len(sig.args), len(c.Args)),
			Fix:      fmt.Sprintf("Provide %d argument(s) for %s().", len(sig.args), c.Name),
			Reference: &hints.Reference{
				FunctionName:    c.Name,
				FunctionReturns: typeKindName(sig.ret),
				FunctionArgs:    typeKindNames(sig.args),
			},
		})
	}
	return out
}

func typeKindName(k TypeKind) string {
	switch k {
	case KindBoolean:
		return "Boolean"
	case KindString:
		return "String"
	case KindInteger:
		return "Integer"
	case KindLong:
		return "Long"
	case KindDecimal:
		return "Decimal"
	case KindDateTime:
		return "DateTime"
	case KindBinary:
		return "Binary"
	case KindObject:
		return "Object"
	case KindList:
		return "List"
	case KindEnumeration:
		return "Enumeration"
	case KindAny:
		return "Any"
	case KindEmpty:
		return "Empty"
	}
	return "Unknown"
}

func typeKindNames(ks []TypeKind) []string {
	out := make([]string, len(ks))
	for i, k := range ks {
		out[i] = typeKindName(k)
	}
	return out
}

// KindName is the public accessor for the human-readable name of a TypeKind
// (e.g. KindBoolean → "Boolean"). Used by cmd/mxcli for help/explain output.
func KindName(k TypeKind) string { return typeKindName(k) }

// PublicFuncSig is a JSON/CLI-friendly view of a built-in function signature.
type PublicFuncSig struct {
	Args    []string
	Returns string
}

// PublicFuncTable returns a JSON-friendly view of the built-in function
// signatures used by the checker.
func PublicFuncTable() map[string]PublicFuncSig {
	out := make(map[string]PublicFuncSig, len(funcTable))
	for k, v := range funcTable {
		out[k] = PublicFuncSig{
			Args:    typeKindNames(v.args),
			Returns: typeKindName(v.ret),
		}
	}
	return out
}
