// SPDX-License-Identifier: Apache-2.0

package exprcheck

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/exprcheck/hints"
)

type funcSig struct {
	args    []TypeKind // full parameter list (max arity)
	minArgs int        // minimum required args; 0 means all args required (= len(args))
	ret     TypeKind
}

// min returns the effective minimum argument count.
func (s funcSig) min() int {
	if s.minArgs > 0 {
		return s.minArgs
	}
	return len(s.args)
}

var funcTable = map[string]funcSig{
	// Boolean
	"not": {args: []TypeKind{KindBoolean}, ret: KindBoolean},

	// String — predicates
	"contains":   {args: []TypeKind{KindString, KindString}, ret: KindBoolean},
	"startsWith": {args: []TypeKind{KindString, KindString}, ret: KindBoolean},
	"endsWith":   {args: []TypeKind{KindString, KindString}, ret: KindBoolean},
	"isMatch":    {args: []TypeKind{KindString, KindString}, ret: KindBoolean},

	// String — transforms
	"length":     {args: []TypeKind{KindString}, ret: KindInteger},
	"find":       {args: []TypeKind{KindString, KindString}, ret: KindInteger},
	// substring(string, startPos)         → from startPos to end
	// substring(string, startPos, length) → length chars from startPos (optional 3rd arg)
	"substring": {args: []TypeKind{KindString, KindInteger, KindInteger}, minArgs: 2, ret: KindString},
	"trim":       {args: []TypeKind{KindString}, ret: KindString},
	"toUpperCase": {args: []TypeKind{KindString}, ret: KindString},
	"toLowerCase": {args: []TypeKind{KindString}, ret: KindString},
	"replaceAll": {args: []TypeKind{KindString, KindString, KindString}, ret: KindString},
	"replace":    {args: []TypeKind{KindString, KindString, KindString}, ret: KindString},
	"urlEncode":  {args: []TypeKind{KindString}, ret: KindString},
	"urlDecode":  {args: []TypeKind{KindString}, ret: KindString},

	// Type conversion
	"toString":     {args: []TypeKind{KindAny}, ret: KindString},
	"parseInteger": {args: []TypeKind{KindString}, ret: KindInteger},
	"parseDecimal": {args: []TypeKind{KindString}, ret: KindDecimal},
	"parseBoolean": {args: []TypeKind{KindString}, ret: KindBoolean},

	// Math
	"abs":   {args: []TypeKind{KindDecimal}, ret: KindDecimal},
	// round(number)              → rounds to 0 decimal places
	// round(number, decimals)    → rounds to specified decimal places (optional 2nd arg)
	"round": {args: []TypeKind{KindDecimal, KindInteger}, minArgs: 1, ret: KindDecimal},
	"floor": {args: []TypeKind{KindDecimal}, ret: KindDecimal},
	"ceil":  {args: []TypeKind{KindDecimal}, ret: KindDecimal},
	"pow":   {args: []TypeKind{KindDecimal, KindDecimal}, ret: KindDecimal},
	"sqrt":  {args: []TypeKind{KindDecimal}, ret: KindDecimal},
	"max":   {args: []TypeKind{KindDecimal, KindDecimal}, ret: KindDecimal},
	"min":   {args: []TypeKind{KindDecimal, KindDecimal}, ret: KindDecimal},
	"random": {args: []TypeKind{}, ret: KindDecimal},

	// DateTime — construction / current
	"currentDateTime":   {args: []TypeKind{}, ret: KindDateTime},
	"dateTime":          {args: []TypeKind{KindInteger, KindInteger, KindInteger, KindInteger, KindInteger, KindInteger}, ret: KindDateTime},

	// DateTime — arithmetic
	"addSeconds": {args: []TypeKind{KindDateTime, KindInteger}, ret: KindDateTime},
	"addMinutes": {args: []TypeKind{KindDateTime, KindInteger}, ret: KindDateTime},
	"addHours":   {args: []TypeKind{KindDateTime, KindInteger}, ret: KindDateTime},
	"addDays":    {args: []TypeKind{KindDateTime, KindInteger}, ret: KindDateTime},
	"addWeeks":   {args: []TypeKind{KindDateTime, KindInteger}, ret: KindDateTime},
	"addMonths":  {args: []TypeKind{KindDateTime, KindInteger}, ret: KindDateTime},
	"addYears":   {args: []TypeKind{KindDateTime, KindInteger}, ret: KindDateTime},

	// DateTime — difference
	"secondsBetween": {args: []TypeKind{KindDateTime, KindDateTime}, ret: KindInteger},
	"minutesBetween": {args: []TypeKind{KindDateTime, KindDateTime}, ret: KindInteger},
	"hoursBetween":   {args: []TypeKind{KindDateTime, KindDateTime}, ret: KindInteger},
	"daysBetween":    {args: []TypeKind{KindDateTime, KindDateTime}, ret: KindInteger},
	"weeksBetween":   {args: []TypeKind{KindDateTime, KindDateTime}, ret: KindInteger},
	"monthsBetween":  {args: []TypeKind{KindDateTime, KindDateTime}, ret: KindInteger},
	"yearsBetween":   {args: []TypeKind{KindDateTime, KindDateTime}, ret: KindInteger},

	// DateTime — formatting / parsing
	"formatDateTime": {args: []TypeKind{KindDateTime, KindString}, ret: KindString},
	"parseDateTime":  {args: []TypeKind{KindString, KindString}, ret: KindDateTime},

	// DateTime — extraction
	"dateTimeToDate": {args: []TypeKind{KindDateTime}, ret: KindDateTime},
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
	got := len(c.Args)
	minA, maxA := sig.min(), len(sig.args)
	if got < minA || got > maxA {
		var expectStr, fixStr string
		if minA == maxA {
			expectStr = fmt.Sprintf("%d", maxA)
			fixStr = fmt.Sprintf("Provide %d argument(s) for %s().", maxA, c.Name)
		} else {
			expectStr = fmt.Sprintf("%d to %d", minA, maxA)
			fixStr = fmt.Sprintf("Provide %d to %d argument(s) for %s().", minA, maxA, c.Name)
		}
		out = append(out, Hint{
			Code: "E006", Slug: "func-arg-arity",
			Severity: hints.SeverityError,
			Where: hints.Location{
				File: ctx.File, Line: c.Pos().Line, Column: c.Pos().Column,
				Microflow: ctx.Microflow,
				Context:   fmt.Sprintf("call to %s()", c.Name),
			},
			YouWrote: fmt.Sprintf("%s(...) with %d argument(s)", c.Name, got),
			Problem:  fmt.Sprintf("%s() expects %s argument(s), got %d.", c.Name, expectStr, got),
			Fix:      fixStr,
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
	MinArgs int // 0 means all Args are required
	Returns string
}

// PublicFuncTable returns a JSON-friendly view of the built-in function
// signatures used by the checker.
func PublicFuncTable() map[string]PublicFuncSig {
	out := make(map[string]PublicFuncSig, len(funcTable))
	for k, v := range funcTable {
		out[k] = PublicFuncSig{
			Args:    typeKindNames(v.args),
			MinArgs: v.minArgs,
			Returns: typeKindName(v.ret),
		}
	}
	return out
}
