// SPDX-License-Identifier: Apache-2.0

package typecheck

import "github.com/mendixlabs/mxcli/mdl/exprcheck"

// Compatible reports whether an expression of type actual can fill a slot
// expecting expected. Returns true (skip) when either side is KindUnknown.
func Compatible(actual, expected exprcheck.TypeKind) bool {
	if actual == exprcheck.KindUnknown || expected == exprcheck.KindUnknown {
		return true
	}
	if actual == exprcheck.KindAny || expected == exprcheck.KindAny {
		return true
	}
	if actual == expected {
		return true
	}
	if actual == exprcheck.KindEmpty &&
		(expected == exprcheck.KindObject || expected == exprcheck.KindList) {
		return true
	}
	return numericWiden(actual, expected)
}

func numericWiden(actual, expected exprcheck.TypeKind) bool {
	rank := map[exprcheck.TypeKind]int{
		exprcheck.KindInteger: 1,
		exprcheck.KindLong:    2,
		exprcheck.KindDecimal: 3,
	}
	ra, oka := rank[actual]
	re, oke := rank[expected]
	return oka && oke && ra <= re
}

var kindDisplayName = map[exprcheck.TypeKind]string{
	exprcheck.KindString:      "String",
	exprcheck.KindInteger:     "Integer",
	exprcheck.KindLong:        "Long",
	exprcheck.KindDecimal:     "Decimal",
	exprcheck.KindBoolean:     "Boolean",
	exprcheck.KindDateTime:    "DateTime",
	exprcheck.KindObject:      "Object",
	exprcheck.KindList:        "List",
	exprcheck.KindBinary:      "Binary",
	exprcheck.KindEnumeration: "Enumeration",
	exprcheck.KindEmpty:       "Empty",
	exprcheck.KindAny:         "Any",
	exprcheck.KindUnknown:     "Unknown",
}

// KindName returns the human-readable name for a TypeKind.
func KindName(k exprcheck.TypeKind) string {
	if s, ok := kindDisplayName[k]; ok {
		return s
	}
	return "Unknown"
}

// FixSuggestion returns a human-readable fix hint for the actual→expected mismatch.
func FixSuggestion(actual, expected exprcheck.TypeKind, rawExpr string) string {
	disp := rawExpr
	if len(disp) > 40 {
		disp = disp[:40] + "..."
	}

	isNumeric := func(k exprcheck.TypeKind) bool {
		return k == exprcheck.KindInteger || k == exprcheck.KindLong || k == exprcheck.KindDecimal
	}

	switch {
	case isNumeric(actual) && expected == exprcheck.KindString:
		return "Wrap with toString(): toString(" + disp + ")"
	case actual == exprcheck.KindBoolean && expected == exprcheck.KindString:
		return "Wrap with toString(): toString(" + disp + ")"
	case actual == exprcheck.KindDateTime && expected == exprcheck.KindString:
		return "Wrap with formatDateTime(): formatDateTime(" + disp + ", 'format')"
	case actual == exprcheck.KindString && expected == exprcheck.KindInteger:
		return "Wrap with parseInteger(): parseInteger(" + disp + ")"
	case actual == exprcheck.KindString && expected == exprcheck.KindDecimal:
		return "Wrap with parseDecimal(): parseDecimal(" + disp + ")"
	case actual == exprcheck.KindString && expected == exprcheck.KindBoolean:
		return "Wrap with parseBoolean(): parseBoolean(" + disp + ")"
	case actual == exprcheck.KindString && expected == exprcheck.KindDateTime:
		return "Wrap with parseDateTime(): parseDateTime(" + disp + ", 'format')"
	case actual == exprcheck.KindDateTime && isNumeric(expected):
		return "Extract the needed component: year(" + disp + "), month(" + disp + "), etc."
	case actual == exprcheck.KindEmpty:
		return "Expression may be empty; add a null-check: if " + disp + " = empty then ... else " + disp
	case expected == exprcheck.KindBoolean:
		return "Slot requires a Boolean expression (e.g. a comparison or boolean function call)"
	default:
		return "Expression type (" + KindName(actual) + ") is not compatible with expected type (" + KindName(expected) + ")"
	}
}
