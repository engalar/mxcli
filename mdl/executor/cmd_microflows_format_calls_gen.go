// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.2.d — Microflow/Java/JavaScript call action family
// formatters (gen-typed).
//
// This file implements the gen-typed counterpart to legacy
// `cmd_microflows_format_action.go` (lines 516-628 + 844-893) for the
// four call-style actions:
//
//   | gen Go type                       | BSON $Type                          | MDL keyword                                    |
//   |-----------------------------------|-------------------------------------|------------------------------------------------|
//   | *genMf.MicroflowCallAction        | Microflows$MicroflowCallAction      | `[$X =] call microflow Mod.MyFlow(p = …);`     |
//   | *genMf.NanoflowCallAction         | Microflows$NanoflowCallAction       | `[$X =] call nanoflow Mod.MyFlow(p = …);`      |
//   | *genMf.JavaActionCallAction       | Microflows$JavaActionCallAction     | `[$X =] call java action Mod.MyAction(p = …);` |
//   | *genMf.JavaScriptActionCallAction | Microflows$JavaScriptActionCallAction | `[$X =] call javascript action Mod.MyAction(p = …);` |
//
// Java/JavaScript actions both wrap each argument in a polymorphic
// `ParameterValue` element. Five sub-types are supported, mirroring
// legacy `parseCodeActionParameterValue`:
//
//   | gen Go type                                | BSON $Type                                       | Rendering                          |
//   |--------------------------------------------|--------------------------------------------------|------------------------------------|
//   | *genMf.StringTemplateParameterValue        | Microflows$StringTemplateParameterValue          | `'<text>'` (mdlQuote'd)            |
//   | *genMf.ExpressionBasedCodeActionParameterValue | Microflows$ExpressionBasedCodeActionParameterValue | bare expression text          |
//   | *genMf.BasicCodeActionParameterValue       | Microflows$BasicCodeActionParameterValue         | bare argument or `empty`           |
//   | *genMf.MicroflowParameterValue             | Microflows$MicroflowParameterValue               | `'<microflow QN>'` or `empty`      |
//   | *genMf.EntityTypeCodeActionParameterValue  | Microflows$EntityTypeCodeActionParameterValue    | `'<entity QN>'`                    |
//
// One notable BSON discrepancy between gen and legacy preserved here:
//
//  1. `MicroflowCallAction` and `JavaActionCallAction` legacy parsers
//     read the result variable from BSON key `ResultVariableName`,
//     while gen reads from `VariableName` (the modern Studio Pro key).
//     Real-world MPRs carry both during the deprecation overlap; gen
//     wins for Stage 3.2.2.d. Tests use the typed setters so this is
//     transparent.
//
// All output strings are 1:1 with the legacy formatters so the
// migrated body diff against the SDK path stays empty.

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// formatMicroflowCallActionGen emits one of:
//
//	`$Var = call microflow Mod.MyFlow(p1 = $arg, …);`
//	`call microflow Mod.MyFlow(p1 = $arg, …);`
//
// Falls back to the literal "Microflow" placeholder when the nested
// MicroflowCall element is missing or has no qualified name — matching
// legacy verbatim. Parameter names are reduced to the bare last-dot
// segment of the parameter qualified name (legacy uses `LastIndex`).
func formatMicroflowCallActionGen(a *genMf.MicroflowCallAction) string {
	mfName := "Microflow"
	var params []string
	if call, ok := a.MicroflowCall().(*genMf.MicroflowCall); ok && call != nil {
		if qn := strings.TrimSpace(call.MicroflowQualifiedName()); qn != "" {
			mfName = qn
		}
		for _, m := range call.ParameterMappingsItems() {
			pm, ok := m.(*genMf.MicroflowCallParameterMapping)
			if !ok || pm == nil {
				continue
			}
			paramName := lastDotSegment(pm.ParameterQualifiedName())
			params = append(params, fmt.Sprintf("%s = %s", paramName, pm.Argument()))
		}
	}
	paramStr := strings.Join(params, ", ")

	errSuffix := ""
	if a.ErrorHandlingType() == "Continue" {
		errSuffix = " on error continue"
	}
	if a.UseReturnVariable() && a.OutputVariableName() != "" {
		return fmt.Sprintf("$%s = call microflow %s(%s)%s;", a.OutputVariableName(), mfName, paramStr, errSuffix)
	}
	return fmt.Sprintf("call microflow %s(%s)%s;", mfName, paramStr, errSuffix)
}

// formatNanoflowCallActionGen emits one of:
//
//	`$Var = call nanoflow Mod.MyFlow(p1 = $arg, …);`
//	`call nanoflow Mod.MyFlow(p1 = $arg, …);`
//
// Same fallback semantics as Microflow: missing NanoflowCall renders
// as the literal "Nanoflow" placeholder (legacy parity).
func formatNanoflowCallActionGen(a *genMf.NanoflowCallAction) string {
	nfName := "Nanoflow"
	var params []string
	if call, ok := a.NanoflowCall().(*genMf.NanoflowCall); ok && call != nil {
		if qn := strings.TrimSpace(call.NanoflowQualifiedName()); qn != "" {
			nfName = qn
		}
		for _, m := range call.ParameterMappingsItems() {
			pm, ok := m.(*genMf.NanoflowCallParameterMapping)
			if !ok || pm == nil {
				continue
			}
			paramName := lastDotSegment(pm.ParameterQualifiedName())
			params = append(params, fmt.Sprintf("%s = %s", paramName, pm.Argument()))
		}
	}
	paramStr := strings.Join(params, ", ")

	if a.UseReturnVariable() && a.OutputVariableName() != "" {
		return fmt.Sprintf("$%s = call nanoflow %s(%s);", a.OutputVariableName(), nfName, paramStr)
	}
	return fmt.Sprintf("call nanoflow %s(%s);", nfName, paramStr)
}

// formatJavaActionCallActionGen emits one of:
//
//	`$Var = call java action Mod.MyAction(p1 = …, …);`
//	`call java action Mod.MyAction(p1 = …, …);`
//
// Falls back to the literal "JavaAction" placeholder when the action
// reference is missing — legacy parity. Each parameter mapping is
// rendered through `formatCodeActionParameterValueGen`; mappings whose
// rendered value is empty are dropped (matches legacy filter).
//
// Java mappings carry the value in either the `Value` or
// `ParameterValue` field; we prefer `Value` first because that's the
// long-standing legacy field, falling back to `ParameterValue` for
// newer MPRs.
func formatJavaActionCallActionGen(a *genMf.JavaActionCallAction) string {
	javaActionName := strings.TrimSpace(a.JavaActionQualifiedName())
	if javaActionName == "" {
		javaActionName = "JavaAction"
	}

	var params []string
	for _, m := range a.ParameterMappingsItems() {
		pm, ok := m.(*genMf.JavaActionParameterMapping)
		if !ok || pm == nil {
			continue
		}
		paramName := lastDotSegment(pm.ParameterQualifiedName())
		value := pm.Value()
		if value == nil {
			value = pm.ParameterValue()
		}
		valueStr := formatCodeActionParameterValueGen(value, true)
		if valueStr == "" {
			continue
		}
		params = append(params, fmt.Sprintf("%s = %s", paramName, valueStr))
	}
	paramStr := strings.Join(params, ", ")

	if a.UseReturnVariable() && a.OutputVariableName() != "" {
		return fmt.Sprintf("$%s = call java action %s(%s);", a.OutputVariableName(), javaActionName, paramStr)
	}
	return fmt.Sprintf("call java action %s(%s);", javaActionName, paramStr)
}

// formatJavaScriptActionCallActionGen emits one of:
//
//	`$Var = call javascript action Mod.MyAction(p1 = …, …);`
//	`call javascript action Mod.MyAction(p1 = …, …);`
//
// When the action reference is missing, returns the legacy
// `-- JavaScriptAction: missing action reference[ (N param/s)]`
// placeholder so the body still parses as a comment. Mappings with
// empty values are dropped (legacy parity).
//
// MicroflowParameterValue is intentionally NOT supported here — legacy
// limits JavaScript actions to the four code-action sub-types
// (StringTemplate / ExpressionBased / BasicCode / EntityType). Passing
// MicroflowParameterValue through this path renders as empty and the
// mapping is dropped.
func formatJavaScriptActionCallActionGen(a *genMf.JavaScriptActionCallAction) string {
	jsActionName := strings.TrimSpace(a.JavaScriptActionQualifiedName())
	if jsActionName == "" {
		mappings := a.ParameterMappingsItems()
		if n := len(mappings); n > 0 {
			label := "params"
			if n == 1 {
				label = "param"
			}
			return fmt.Sprintf("-- JavaScriptAction: missing action reference (%d %s)", n, label)
		}
		return "-- JavaScriptAction: missing action reference"
	}

	var params []string
	for _, m := range a.ParameterMappingsItems() {
		pm, ok := m.(*genMf.JavaScriptActionParameterMapping)
		if !ok || pm == nil {
			continue
		}
		paramName := lastDotSegment(pm.ParameterQualifiedName())
		valueStr := formatCodeActionParameterValueGen(pm.ParameterValue(), false)
		if valueStr == "" {
			continue
		}
		params = append(params, fmt.Sprintf("%s = %s", paramName, valueStr))
	}
	paramStr := strings.Join(params, ", ")

	if a.UseReturnVariable() && a.OutputVariableName() != "" {
		return fmt.Sprintf("$%s = call javascript action %s(%s);", a.OutputVariableName(), jsActionName, paramStr)
	}
	return fmt.Sprintf("call javascript action %s(%s);", jsActionName, paramStr)
}

// formatCodeActionParameterValueGen renders a single
// {String,Expression,Basic,Microflow,EntityType}-CodeActionParameterValue
// to the surface string used inside a `(p = <value>, …)` clause.
//
// `allowMicroflowParam` toggles whether MicroflowParameterValue is
// recognised (true for Java actions, false for JavaScript actions —
// matching legacy's per-host restriction).
//
// Returns "" for unrecognised value types. The caller is expected to
// drop mappings whose rendered value is empty (mirrors legacy's
// `if valueStr != ""` guard before appending to the slice).
//
// ExpressionBasedCodeActionParameterValue is registered in the
// microflows gen package supplement, which also exposes an Expression()
// getter that pulls the field from raw BSON on demand.
func formatCodeActionParameterValueGen(v element.Element, allowMicroflowParam bool) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case *genMf.StringTemplateParameterValue:
		if tt, ok := t.TypedTemplate().(*genMf.TypedTemplate); ok && tt != nil {
			return mdlQuote(tt.Text())
		}
		return ""
	case *genMf.ExpressionBasedCodeActionParameterValue:
		return t.Expression()
	case *genMf.BasicCodeActionParameterValue:
		if t.Argument() == "" {
			if allowMicroflowParam {
				// Java legacy renders empty BasicCode as `empty` so the
				// rendered call still has the correct parameter slot.
				return "empty"
			}
			// JavaScript legacy renders empty BasicCode as the bare
			// empty string, which the caller drops.
			return ""
		}
		return t.Argument()
	case *genMf.MicroflowParameterValue:
		if !allowMicroflowParam {
			return ""
		}
		if qn := t.MicroflowQualifiedName(); qn != "" {
			return mdlQuote(qn)
		}
		return "empty"
	case *genMf.EntityTypeCodeActionParameterValue:
		if qn := t.EntityQualifiedName(); qn != "" {
			return mdlQuote(qn)
		}
		return ""
	default:
		return ""
	}
}
