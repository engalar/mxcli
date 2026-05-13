// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.2.a — Object Actions family formatters (gen-typed).
// Stage 3.2.2.b — Form Actions family formatters (gen-typed).
//
// This file implements the gen-typed counterpart to legacy
// `cmd_microflows_format_action.go`. It is invoked from
// `renderGenMicroflowBody` for every `*genMf.ActionActivity` node and
// formats the action kinds listed below.
//
// Object Actions (Stage 3.2.2.a):
//   | gen Go type                  | BSON $Type           | MDL keyword          |
//   |------------------------------|----------------------|----------------------|
//   | *genMf.CreateObjectAction    | CreateChangeAction   | `$X = create E (…)`  |
//   | *genMf.ChangeObjectAction    | ChangeAction         | `change $X (…)`      |
//   | *genMf.DeleteAction          | DeleteAction         | `delete $X;`         |
//   | *genMf.CommitAction          | CommitAction         | `commit $X;`         |
//   | *genMf.RollbackAction        | RollbackAction       | `rollback $X;`       |
//   | *genMf.AggregateListAction   | AggregateAction      | `$X = sum($L);`      |
//
// Form Actions (Stage 3.2.2.b):
//   | gen Go type                  | BSON $Type           | MDL keyword          |
//   |------------------------------|----------------------|----------------------|
//   | *genMf.ShowPageAction        | ShowFormAction       | `show page X(…);`    |
//   | *genMf.CloseFormAction       | CloseFormAction      | `close page;`        |
//   | *genMf.ShowHomePageAction    | ShowHomePageAction   | `show home page;`    |
//   | *genMf.ShowMessageAction     | ShowMessageAction    | `show message …;`    |
//
// The output strings are 1:1 with the legacy formatters in
// `cmd_microflows_format_action.go` so that the textual diff between
// the legacy and gen-typed paths is empty for these action kinds.
//
// Activity kinds outside this family return "" — the caller in
// `cmd_microflows_show_gen.go` then emits the existing TODO placeholder
// so coverage rolls out one family at a time.

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	genTx "github.com/mendixlabs/mxcli/modelsdk/gen/texts"
)

// formatActivityGen formats a single gen activity node as an MDL
// statement. It returns "" when the activity is an ActionActivity whose
// inner Action is not yet supported, so the caller can decide whether
// to emit a placeholder or skip the line entirely. Non-ActionActivity
// nodes (Annotations, control-flow events) are out of scope and also
// return "".
func formatActivityGen(_ *ExecContext, obj element.Element) string {
	aa, ok := obj.(*genMf.ActionActivity)
	if !ok || aa == nil {
		return ""
	}
	return formatActionGen(aa.Action())
}

// formatActionGen dispatches the inner action of an ActionActivity to a
// per-type formatter. Returns "" for action kinds not yet covered by
// Stage 3.2.2.{a,b} so the caller falls back to a placeholder.
func formatActionGen(action element.Element) string {
	if action == nil {
		return "-- Empty action"
	}
	switch a := action.(type) {
	case *genMf.CreateObjectAction:
		return formatCreateObjectActionGen(a)
	case *genMf.ChangeObjectAction:
		return formatChangeObjectActionGen(a)
	case *genMf.DeleteAction:
		return formatDeleteActionGen(a)
	case *genMf.CommitAction:
		return formatCommitActionGen(a)
	case *genMf.RollbackAction:
		return formatRollbackActionGen(a)
	case *genMf.AggregateListAction:
		return formatAggregateListActionGen(a)
	case *genMf.ShowPageAction:
		return formatShowPageActionGen(a)
	case *genMf.CloseFormAction:
		return formatCloseFormActionGen(a)
	case *genMf.ShowHomePageAction:
		return formatShowHomePageActionGen(a)
	case *genMf.ShowMessageAction:
		return formatShowMessageActionGen(a)
	default:
		return ""
	}
}

// formatCreateObjectActionGen emits `$Var = create Module.Entity (…);`.
// Mirrors legacy CreateObjectAction handling: bare attribute names,
// fully-qualified association names with same-module prefix stripped,
// fall-back default outputVar of "NewObject" when the source has none.
func formatCreateObjectActionGen(a *genMf.CreateObjectAction) string {
	entityName := strings.TrimSpace(a.EntityQualifiedName())
	if entityName == "" {
		entityName = "Entity"
	}
	outputVar := strings.TrimSpace(a.OutputVariableName())
	if outputVar == "" {
		outputVar = "NewObject"
	}
	entityModule := ""
	if parts := strings.SplitN(entityName, ".", 2); len(parts) == 2 {
		entityModule = parts[0]
	}
	members := collectInitialMemberAssignmentsGen(a.ItemsItems(), entityModule)
	if len(members) > 0 {
		return fmt.Sprintf("$%s = create %s (%s);", outputVar, entityName, strings.Join(members, ", "))
	}
	return fmt.Sprintf("$%s = create %s;", outputVar, entityName)
}

// collectInitialMemberAssignmentsGen renders MemberChange items the
// same way legacy formats them inside a `create` statement: attribute
// name only (last dot segment); association uses its qualified name,
// stripping the module prefix when it matches the create target's
// module so the rendered text round-trips through the visitor.
func collectInitialMemberAssignmentsGen(items []element.Element, entityModule string) []string {
	var out []string
	for _, item := range items {
		mc, ok := item.(*genMf.MemberChange)
		if !ok || mc == nil {
			continue
		}
		var name string
		if assoc := mc.AssociationQualifiedName(); assoc != "" {
			name = assoc
			if parts := strings.SplitN(assoc, ".", 2); len(parts) == 2 && parts[0] == entityModule {
				name = parts[1]
			}
		} else {
			name = mc.AttributeQualifiedName()
			if name == "" {
				continue
			}
			// Module.Entity.Attr → Attr.
			if parts := strings.Split(name, "."); len(parts) > 0 {
				name = parts[len(parts)-1]
			}
		}
		out = append(out, fmt.Sprintf("%s = %s", name, escapeExpressionValue(mc.Value())))
	}
	return out
}

// formatChangeObjectActionGen emits `change $Var (…) [refresh];`.
// Mirrors legacy ChangeObjectAction handling.
func formatChangeObjectActionGen(a *genMf.ChangeObjectAction) string {
	varName := strings.TrimSpace(a.ChangeVariableName())
	if varName == "" {
		varName = "Object"
	}
	members := collectChangeMemberAssignmentsGen(a.ItemsItems())
	refreshSuffix := ""
	if a.RefreshInClient() {
		refreshSuffix = " refresh"
	}
	if len(members) > 0 {
		return fmt.Sprintf("change $%s (%s)%s;", varName, strings.Join(members, ", "), refreshSuffix)
	}
	return fmt.Sprintf("change $%s%s;", varName, refreshSuffix)
}

// collectChangeMemberAssignmentsGen renders MemberChange items the same
// way legacy formats them inside a `change` statement: associations
// keep the full qualified name (legacy makes no module-prefix strip in
// the change path); attributes use only the last dot segment.
func collectChangeMemberAssignmentsGen(items []element.Element) []string {
	var out []string
	for _, item := range items {
		mc, ok := item.(*genMf.MemberChange)
		if !ok || mc == nil {
			continue
		}
		var name string
		if assoc := mc.AssociationQualifiedName(); assoc != "" {
			name = assoc
		} else {
			name = mc.AttributeQualifiedName()
			if name == "" {
				continue
			}
			if parts := strings.Split(name, "."); len(parts) > 0 {
				name = parts[len(parts)-1]
			}
		}
		out = append(out, fmt.Sprintf("%s = %s", name, escapeExpressionValue(mc.Value())))
	}
	return out
}

// formatDeleteActionGen emits `delete $Var;` (matches legacy verbatim).
func formatDeleteActionGen(a *genMf.DeleteAction) string {
	return fmt.Sprintf("delete $%s;", a.DeleteVariableName())
}

// formatCommitActionGen emits `commit $Var [with events] [refresh];`.
// Mirrors legacy CommitObjectsAction.
func formatCommitActionGen(a *genMf.CommitAction) string {
	varName := strings.TrimSpace(a.CommitVariableName())
	if varName == "" {
		varName = "Object"
	}
	suffix := ""
	if a.WithEvents() {
		suffix += " with events"
	}
	if a.RefreshInClient() {
		suffix += " refresh"
	}
	return fmt.Sprintf("commit $%s%s;", varName, suffix)
}

// formatRollbackActionGen emits `rollback $Var [refresh];`.
// Mirrors legacy RollbackObjectAction.
func formatRollbackActionGen(a *genMf.RollbackAction) string {
	if a.RefreshInClient() {
		return fmt.Sprintf("rollback $%s refresh;", a.RollbackVariableName())
	}
	return fmt.Sprintf("rollback $%s;", a.RollbackVariableName())
}

// formatAggregateListActionGen emits one of three forms depending on
// the action's flavour, mirroring legacy AggregateListAction:
//
//   - Expression-based: `$Out = sum($List, <expr>);`
//   - Attribute-based:  `$Out = sum($List.Attr);`
//   - Bare/count:       `$Out = count($List);`
func formatAggregateListActionGen(a *genMf.AggregateListAction) string {
	outputVar := strings.TrimSpace(a.OutputVariableName())
	if outputVar == "" {
		outputVar = "Result"
	}
	fn := strings.TrimSpace(a.AggregateFunction())
	if fn == "" {
		fn = "Count"
	}
	fnLower := strings.ToLower(fn)

	if a.UseExpression() && strings.TrimSpace(a.Expression()) != "" {
		return fmt.Sprintf("$%s = %s($%s, %s);", outputVar, fnLower, a.InputListVariableName(), a.Expression())
	}

	attrName := a.AttributeQualifiedName()
	if attrName != "" {
		if parts := strings.Split(attrName, "."); len(parts) > 0 {
			attrName = parts[len(parts)-1]
		}
	}
	if attrName != "" && fn != genMf.AggregateFunctionEnumCount {
		return fmt.Sprintf("$%s = %s($%s.%s);", outputVar, fnLower, a.InputListVariableName(), attrName)
	}
	return fmt.Sprintf("$%s = %s($%s);", outputVar, fnLower, a.InputListVariableName())
}

// ────────────────────────────────────────────────────────
// Stage 3.2.2.b — Form Actions family
// ────────────────────────────────────────────────────────

// formatShowPageActionGen emits `show page <PageQN>[(<param = arg, …>)];`.
// Mirrors legacy ShowPageAction handling but reads the page reference
// from the gen `PageSettings` element (BY_NAME_REFERENCE on `Form`),
// which collapses the legacy PageName / PageID branching into a single
// qualified-name source. The legacy PageID-based fallback (resolve
// container chain via `ctx.Backend.ListPages`) is not exercised here:
// gen `PageSettings.InitFromRaw` reads only `Form` and modern MPRs
// always carry the qualified name. When neither source resolves the
// page, we emit the same `UnknownPage` placeholder as legacy.
func formatShowPageActionGen(a *genMf.ShowPageAction) string {
	pageName := "UnknownPage"
	var mappings []element.Element
	if ps, ok := a.PageSettings().(*genPg.PageSettings); ok && ps != nil {
		if qn := strings.TrimSpace(ps.PageQualifiedName()); qn != "" {
			pageName = qn
		}
		mappings = ps.ParameterMappingsItems()
	}

	var params []string
	for _, m := range mappings {
		ppm, ok := m.(*genPg.PageParameterMapping)
		if !ok || ppm == nil {
			continue
		}
		paramName := lastDotSegment(ppm.ParameterQualifiedName())
		params = append(params, fmt.Sprintf("$%s = %s", paramName, ppm.Argument()))
	}

	paramStr := ""
	if len(params) > 0 {
		paramStr = "(" + strings.Join(params, ", ") + ")"
	}
	return fmt.Sprintf("show page %s%s;", pageName, paramStr)
}

// formatCloseFormActionGen emits `close page;` or `close page <N>;`.
// Mirrors legacy ClosePageAction handling: only the integer-valued
// `NumberOfPages` field is consulted (the newer `NumberOfPagesToClose`
// expression-form is not surfaced by legacy and is therefore ignored
// here for 1:1 parity).
func formatCloseFormActionGen(a *genMf.CloseFormAction) string {
	if n := a.NumberOfPages(); n > 1 {
		return fmt.Sprintf("close page %d;", n)
	}
	return "close page;"
}

// formatShowHomePageActionGen emits the constant `show home page;`.
// ShowHomePageAction has no surface-level parameters in legacy MDL,
// so the gen-typed formatter is a literal mirror.
func formatShowHomePageActionGen(_ *genMf.ShowHomePageAction) string {
	return "show home page;"
}

// formatShowMessageActionGen emits
// `show message <text> type <Type>[ objects [<expr>, …]];`.
//
// Mirrors legacy ShowMessageAction.Type/Template/TemplateParameters
// handling. Two subtle quirks of legacy preserved verbatim:
//
//   1. When the action has no Template (or an empty Translations map),
//      the rendered text is the literal three-character `'...'` —
//      already wrapped in single quotes. When a Template is present
//      we wrap the text via `mdlQuote` (escaping internal quotes).
//   2. The Template's text source prefers the `en_US` translation; any
//      other available translation is the fallback.
//
// In gen the translations live inside a Texts$Text element nested under
// the action's TextTemplate (not the legacy `model.Text.Translations`
// map), and TemplateParameters are PartList items on the TextTemplate
// (not a separate string slice on the action).
func formatShowMessageActionGen(a *genMf.ShowMessageAction) string {
	msgType := strings.TrimSpace(a.Type())
	if msgType == "" {
		msgType = genMf.ShowMessageTypeInformation
	}

	message := "'...'"
	tmpl, _ := a.Template().(*genMf.TextTemplate)
	if tmpl != nil {
		if text, ok := tmpl.Text().(*genTx.Text); ok && text != nil {
			if picked, found := pickTextTranslationGen(text); found {
				message = mdlQuote(picked)
			}
		}
	}

	result := fmt.Sprintf("show message %s type %s", message, msgType)

	if tmpl != nil {
		var params []string
		for _, arg := range tmpl.ArgumentsItems() {
			if ta, ok := arg.(*genMf.TemplateArgument); ok && ta != nil {
				if expr := strings.TrimSpace(ta.Expression()); expr != "" {
					params = append(params, expr)
				}
			}
		}
		if len(params) > 0 {
			result += " objects [" + strings.Join(params, ", ") + "]"
		}
	}

	return result + ";"
}

// pickTextTranslationGen returns the text body of a Texts$Text, with
// the same precedence as legacy: prefer en_US, otherwise the first
// non-empty translation encountered. The bool is false if no
// translations are present at all (so the caller can keep the legacy
// `'...'` placeholder, which is intentionally pre-quoted and must not
// flow through mdlQuote a second time).
func pickTextTranslationGen(t *genTx.Text) (string, bool) {
	items := t.TranslationsItems()
	if len(items) == 0 {
		return "", false
	}
	var firstAny string
	var foundAny bool
	for _, it := range items {
		tr, ok := it.(*genTx.Translation)
		if !ok || tr == nil {
			continue
		}
		if tr.LanguageCode() == "en_US" {
			return tr.Text(), true
		}
		if !foundAny {
			firstAny = tr.Text()
			foundAny = true
		}
	}
	if !foundAny {
		return "", false
	}
	return firstAny, true
}
