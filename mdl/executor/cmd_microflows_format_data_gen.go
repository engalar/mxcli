// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.2.e — Variable / Expression / Data family formatters
// (gen-typed).
//
// This file implements the gen-typed counterpart to legacy
// `cmd_microflows_format_action.go` for the seven action kinds below.
//
//   | gen Go type                       | BSON $Type                          | MDL keyword                                          |
//   |-----------------------------------|-------------------------------------|------------------------------------------------------|
//   | *genMf.CastAction                 | Microflows$CastAction               | `[$Var = ]cast $Object;`                             |
//   | *genMf.CreateVariableAction       | Microflows$CreateVariableAction     | `declare $Var <Type> = <expr>;`                      |
//   | *genMf.ChangeVariableAction       | Microflows$ChangeVariableAction     | `set $Var = <expr>;` or `change $Obj (Attr = ...);`  |
//   | *genMf.RetrieveAction             | Microflows$RetrieveAction           | `retrieve $Var from <source> [where ...] [...];`     |
//   | *genMf.LogMessageAction           | Microflows$LogMessageAction         | `log <level> node <expr> '<msg>'[ with (...)];`      |
//   | *genMf.DownloadFileAction         | Microflows$DownloadFileAction       | `download file $Var [show in browser];`              |
//   | *genMf.ValidationFeedbackAction   | Microflows$ValidationFeedbackAction | `validation feedback $Obj/Attr message '<msg>';`     |
//
// RetrieveAction wraps a polymorphic `RetrieveSource` element, dispatched
// to one of two concrete sub-types (mirroring legacy):
//
//   | gen Go type                              | BSON $Type                              | Rendering                                                                  |
//   |------------------------------------------|-----------------------------------------|----------------------------------------------------------------------------|
//   | *genMf.DatabaseRetrieveSource            | Microflows$DatabaseRetrieveSource       | `retrieve $V from Module.Entity[\n    where X][\n    sort by …][\n    limit/offset]` |
//   | *genMf.AssociationRetrieveSource         | Microflows$AssociationRetrieveSource    | `retrieve $V from $Start/Module.Assoc;`                                    |
//
// `DatabaseRetrieveSource` is further specialised: a single XPath
// predicate of the form `[Module.Assoc = $Start]` plus no sorting and
// no range collapses to the AssociationRetrieveSource surface
// (`retrieve $V from $Start/Module.Assoc;`) — this is the legacy
// "reverse association" detection. See `parseReverseAssociationRetrieveGen`.
//
// All output strings are 1:1 with the legacy formatters in
// `cmd_microflows_format_action.go` so the migrated body diff against
// the SDK path stays empty for these action kinds.
//
// Notable gen / legacy differences preserved verbatim:
//
//  1. CastAction's gen type only exposes `OutputVariableName()`. The
//     legacy `ObjectVariableName` field has no getter on gen, so the
//     formatter reads it from raw BSON via `codec.ReadBSONFieldString`.
//     Likewise the legacy `VariableName` fallback for `OutputVariable`
//     is preserved by reading the raw field.
//  2. CreateVariableAction's `VariableType` is a polymorphic gen Element
//     pointing at `DataTypes$<XxxType>` (DataTypes domain) — primitives
//     have no extra fields, ObjectType/ListType/EnumerationType expose
//     a `…QualifiedName()` getter. `formatGenDataType` mirrors legacy's
//     `formatMicroflowDataType` over the gen `datatypes` package.
//  3. ConstantRange in gen exposes only `SingleObject()`; the legacy
//     `Microflows$ConstantRange` BSON also carries
//     `LimitExpression`/`OffsetExpression` strings. We read those from
//     raw BSON to preserve the legacy "ConstantRange-as-CustomRange"
//     fallback (Studio Pro stores both shapes interchangeably).
//  4. LogMessageAction's `MessageTemplate` is a `Microflows$StringTemplate`
//     in the BSON: a flat `Text` string + an `Arguments` PartList of
//     `Microflows$TemplateArgument`. The gen `StringTemplate` exposes
//     `Text() string` and `ArgumentsItems()`.
//  5. ValidationFeedbackAction's `FeedbackTemplate` is the standard
//     `Microflows$TextTemplate -> Texts$Text` nesting (same as
//     ShowMessageAction); reusing `pickTextTranslationGen` from the
//     existing `cmd_microflows_format_action_gen.go` keeps the
//     translation precedence (en_US first) consistent.
//  6. ChangeVariableAction supports an XPath-style `ChangeVariableName`
//     like `$Product/Price` — that surface is rewritten as
//     `change $Product (Price = <expr>);` to match legacy. Plain names
//     render as `set $X = <expr>;`.

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDT "github.com/mendixlabs/mxcli/modelsdk/gen/datatypes"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genTx "github.com/mendixlabs/mxcli/modelsdk/gen/texts"
)

// ────────────────────────────────────────────────────────
// CastAction
// ────────────────────────────────────────────────────────

// formatCastActionGen emits one of (legacy parity):
//
//	`$Out = cast $Obj;`   (both vars present)
//	`cast $Obj;`          (no output var)
//	`cast $Out;`          (no object var)
//
// gen `CastAction` exposes only `OutputVariableName()`, and that getter
// is bound to BSON key `"VariableName"` (the storage alias picked by
// codegen). Legacy reads BSON `"OutputVariableName"` first and falls
// back to `"VariableName"`. To match legacy 1:1, we read both raw keys
// in the same order; the `OutputVariableName()` getter is consulted as
// the third fallback so direct-construction tests (which use the typed
// setter) still work.
//
// `ObjectVariableName` has no gen getter, so we read it from raw BSON.
func formatCastActionGen(a *genMf.CastAction) string {
	outputVar := strings.TrimSpace(genMf.CastActionOutputVariableName(a))
	if outputVar == "" {
		outputVar = strings.TrimSpace(genMf.CastActionVariableName(a))
	}
	if outputVar == "" {
		outputVar = strings.TrimSpace(a.OutputVariableName())
	}

	objectVar := strings.TrimSpace(genMf.CastActionObjectVariableName(a))

	outputVar = ensureDollar(outputVar)
	objectVar = ensureDollar(objectVar)

	if objectVar == "" {
		return fmt.Sprintf("cast %s;", outputVar)
	}
	if outputVar == "" {
		return fmt.Sprintf("cast %s;", objectVar)
	}
	return fmt.Sprintf("%s = cast %s;", outputVar, objectVar)
}

// ensureDollar prepends a "$" to a non-empty bare variable name. Used
// by the cast/validation formatters for legacy parity.
func ensureDollar(name string) string {
	if name == "" || strings.HasPrefix(name, "$") {
		return name
	}
	return "$" + name
}

// ────────────────────────────────────────────────────────
// CreateVariableAction
// ────────────────────────────────────────────────────────

// formatCreateVariableActionGen emits `declare $Var <Type> = <expr>;`
// with `<expr>` defaulting to `empty` when the action's InitialValue
// is missing. Legacy strips a single trailing newline from the
// initial value before the empty check — we mirror that.
//
// `<Type>` is rendered through `formatGenDataType` over the action's
// `VariableType()` polymorphic element. When the element is missing
// or carries an unknown $Type, the rendered text falls back to
// "Object" (legacy parity).
func formatCreateVariableActionGen(a *genMf.CreateVariableAction) string {
	varType := formatGenDataType(a.VariableType())
	if varType == "" {
		varType = "Object"
	}
	initialValue := strings.TrimSuffix(a.InitialValue(), "\n")
	if initialValue == "" {
		initialValue = "empty"
	}
	return fmt.Sprintf("declare $%s %s = %s;", a.VariableName(), varType, initialValue)
}

// formatGenDataType mirrors legacy `formatMicroflowDataType` over the
// gen `datatypes` package. Returns the surface MDL type name; the
// caller decides what to do with empty / unknown.
func formatGenDataType(dt element.Element) string {
	if dt == nil {
		return "Unknown"
	}
	switch t := dt.(type) {
	case *genDT.BooleanType:
		return "Boolean"
	case *genDT.IntegerType:
		return "Integer"
	case *genDT.DecimalType:
		return "Decimal"
	case *genDT.FloatType:
		return "Float"
	case *genDT.StringType:
		return "String"
	case *genDT.DateTimeType:
		return "DateTime"
	case *genDT.BinaryType:
		return "Binary"
	case *genDT.VoidType:
		return "Void"
	case *genDT.ObjectType:
		if qn := t.EntityQualifiedName(); qn != "" {
			return qn
		}
		return "Object"
	case *genDT.ListType:
		if qn := t.EntityQualifiedName(); qn != "" {
			return "List of " + qn
		}
		return "List"
	case *genDT.EnumerationType:
		if qn := t.EnumerationQualifiedName(); qn != "" {
			return "enum " + qn
		}
		return "Enumeration"
	case *element.Base:
		return t.TypeName()
	default:
		return ""
	}
}

// ────────────────────────────────────────────────────────
// ChangeVariableAction
// ────────────────────────────────────────────────────────

// formatChangeVariableActionGen emits one of (legacy parity):
//
//	`change $Object (Attr = <expr>);` — when the variable name is an
//	   XPath like "$Product/Price" or "Product/Module.Assoc/Attr".
//	`set $Var = <expr>;`              — for plain variable names.
//
// The XPath-recognition path mirrors legacy: split on the first "/",
// take the last segment of the remainder as the attribute name, and
// emit a `change` statement. This handles the Studio Pro shape where
// "Change variable" is used to set an attribute via XPath.
func formatChangeVariableActionGen(a *genMf.ChangeVariableAction) string {
	varName := a.ChangeVariableName()
	if strings.Contains(varName, "/") {
		parts := strings.SplitN(varName, "/", 2)
		objectName := parts[0]
		attrPath := parts[1]
		attrParts := strings.Split(attrPath, "/")
		attrName := attrParts[len(attrParts)-1]
		if !strings.HasPrefix(objectName, "$") {
			objectName = "$" + objectName
		}
		return fmt.Sprintf("change %s (%s = %s);", objectName, attrName, a.Value())
	}
	if strings.HasPrefix(varName, "$") {
		return fmt.Sprintf("set %s = %s;", varName, a.Value())
	}
	return fmt.Sprintf("set $%s = %s;", varName, a.Value())
}

// ────────────────────────────────────────────────────────
// RetrieveAction
// ────────────────────────────────────────────────────────

// formatRetrieveActionGen emits the multi-form `retrieve` statement.
// See file header for the source-type matrix; this dispatches to
// `formatDatabaseRetrieveSourceGen` or `formatAssociationRetrieveSourceGen`
// depending on the concrete type of `RetrieveSource()`.
//
// `ctx` is required by the database path so it can:
//   - detect the legacy "reverse association" XPath shape and collapse
//     it to the AssociationRetrieveSource surface;
//   - enrich enum literals in XPath constraints with their qualified
//     names (e.g. `Status = 'Open'` → `Status = Module.Status.Open`).
//
// `OutputVariableName()` is gen-aliased to BSON key `"VariableName"`,
// but real Mendix MPRs store the value under `"ResultVariableName"`.
// We try the typed getter first, then fall back to the legacy raw key
// so direct-construction tests and fixture loads both work.
func formatRetrieveActionGen(ctx *ExecContext, a *genMf.RetrieveAction) string {
	outputVar := a.OutputVariableName()
	if outputVar == "" {
		outputVar = genMf.RetrieveActionResultVariableName(a)
	}
	if outputVar == "" {
		outputVar = "Result"
	}

	switch source := a.RetrieveSource().(type) {
	case *genMf.DatabaseRetrieveSource:
		return formatDatabaseRetrieveSourceGen(ctx, source, outputVar)
	case *genMf.AssociationRetrieveSource:
		return formatAssociationRetrieveSourceGen(source, outputVar)
	default:
		return fmt.Sprintf("retrieve $%s from ...;", outputVar)
	}
}

// formatDatabaseRetrieveSourceGen renders a database retrieve. Legacy
// parity, including:
//   - reverse-association detection (collapses `[Mod.Assoc = $Start]` to
//     the association surface when no sort/range);
//   - XPath enum literal enrichment via `enrichXPathConstraintForDescribe`;
//   - multi-predicate splitting (`[a][b]` → `[a]\n    [b]`);
//   - sort by clauses;
//   - limit/offset / first(limit 1) clauses.
func formatDatabaseRetrieveSourceGen(ctx *ExecContext, source *genMf.DatabaseRetrieveSource, outputVar string) string {
	entityName := strings.TrimSpace(source.EntityQualifiedName())
	if entityName == "" {
		entityName = "Entity"
	}

	if startVar, assocName, ok := parseReverseAssociationRetrieveGen(ctx, source, entityName); ok {
		return fmt.Sprintf("retrieve $%s from $%s/%s;", outputVar, startVar, assocName)
	}

	stmt := fmt.Sprintf("retrieve $%s from %s", outputVar, entityName)

	xpath := source.XPathConstraint()
	if xpath == "" {
		// gen alias mismatch: real MPRs use the lowercase-`p` key
		// `XpathConstraint`. Fall back to the raw read.
		xpath = genMf.DatabaseRetrieveSourceXpathConstraint(source)
	}
	if raw := xpath; strings.TrimSpace(raw) != "" {
		constraint := strings.TrimSpace(raw)
		// Enum-literal enrichment requires a connected backend; in
		// nil-ctx unit tests we leave the constraint unchanged so the
		// formatter is exercisable without a fixture.
		if ctx != nil {
			constraint = enrichXPathConstraintForDescribe(ctx, entityName, constraint)
		}
		if strings.HasPrefix(constraint, "[") && strings.HasSuffix(constraint, "]") {
			inner := constraint[1 : len(constraint)-1]
			inner = strings.ReplaceAll(inner, "]\n[", "][")
			parts := strings.Split(inner, "][")
			if len(parts) > 1 {
				var wrapped []string
				for _, p := range parts {
					wrapped = append(wrapped, "["+strings.TrimSpace(p)+"]")
				}
				constraint = strings.Join(wrapped, "\n    ")
			} else {
				constraint = parts[0]
			}
		}
		stmt += fmt.Sprintf("\n    where %s", constraint)
	}

	if sortParts := collectDatabaseSortPartsGen(source); len(sortParts) > 0 {
		stmt += fmt.Sprintf("\n    sort by %s", strings.Join(sortParts, ", "))
	}

	if limit, offset, kind := extractDatabaseRangeGen(source); kind != databaseRangeAll {
		switch kind {
		case databaseRangeFirst:
			stmt += "\n    limit 1"
		case databaseRangeCustom:
			if limit != "" {
				stmt += fmt.Sprintf("\n    limit %s", limit)
			}
			if offset != "" {
				stmt += fmt.Sprintf("\n    offset %s", offset)
			}
		}
	}

	return stmt + ";"
}

// formatAssociationRetrieveSourceGen renders the bare
// `retrieve $V from $Start/Module.Assoc;` surface.
//
// gen `AssociationQualifiedName()` decodes the BSON key `"Association"`,
// but real MPRs store the qualified name under `"AssociationId"`. We
// fall back to the legacy raw key when the typed getter returns empty.
func formatAssociationRetrieveSourceGen(source *genMf.AssociationRetrieveSource, outputVar string) string {
	startVar := source.StartVariableName()
	if startVar == "" {
		startVar = "Object"
	}
	assocName := source.AssociationQualifiedName()
	if assocName == "" {
		assocName = genMf.AssociationRetrieveSourceAssociationId(source)
	}
	if assocName == "" {
		assocName = "..."
	}
	return fmt.Sprintf("retrieve $%s from $%s/%s;", outputVar, startVar, assocName)
}

// collectDatabaseSortPartsGen renders the SortItem list of a database
// retrieve source as the `Module.Entity.Attr asc, … desc` clause used
// inside `sort by …`. Mirrors legacy: each item shows the qualified
// attribute name (no last-segment trim — legacy preserves the full
// qualified name in the sort clause) and the direction.
//
// gen `SortItemList()` decodes the BSON key `"SortItemList"`, but
// real MPRs use the legacy `"NewSortings"` (a Microflows$SortingsList
// whose array key is `"Sortings"`). The gen-typed path is tried first;
// the legacy raw-BSON path is used as a fallback so fixture loads
// surface the sort clause.
func collectDatabaseSortPartsGen(source *genMf.DatabaseRetrieveSource) []string {
	if listEl, ok := source.SortItemList().(*genMf.SortItemList); ok && listEl != nil {
		if parts := sortPartsFromGenList(listEl); len(parts) > 0 {
			return parts
		}
	}
	return sortPartsFromRawBSON(source.Raw())
}

// sortPartsFromGenList renders the gen-typed SortItemList. Used by
// `collectDatabaseSortPartsGen` when gen decoded the modern key.
func sortPartsFromGenList(listEl *genMf.SortItemList) []string {
	var parts []string
	for _, it := range listEl.ItemsItems() {
		si, ok := it.(*genMf.SortItem)
		if !ok || si == nil {
			continue
		}
		attrName := ""
		if ar, ok := si.AttributeRef().(*element.Base); ok && ar != nil {
			attrName = genMf.RawFieldStringFromBase(ar, "Attribute")
		}
		if attrName == "" {
			attrName = si.AttributePath()
		}
		if attrName == "" {
			continue
		}
		order := "asc"
		if si.SortOrder() == genMf.SortOrderEnumDescending {
			order = "desc"
		}
		parts = append(parts, attrName+" "+order)
	}
	return parts
}

// sortPartsFromRawBSON pulls sort items out of the legacy
// `NewSortings -> Sortings` array on a DatabaseRetrieveSource raw BSON
// document via the genMf supplement (which owns BSON decoding).
func sortPartsFromRawBSON(raw []byte) []string {
	decoded := genMf.SortPartsFromRawBSON(raw)
	if len(decoded) == 0 {
		return nil
	}
	parts := make([]string, 0, len(decoded))
	for _, p := range decoded {
		order := "asc"
		if p.Descending {
			order = "desc"
		}
		parts = append(parts, p.AttributeName+" "+order)
	}
	return parts
}

type databaseRangeKind int

const (
	databaseRangeAll databaseRangeKind = iota
	databaseRangeFirst
	databaseRangeCustom
)

// extractDatabaseRangeGen returns (limit, offset, kind) for a database
// retrieve range. Mirrors `parseRange` in legacy: ConstantRange with
// SingleObject=true is "first"; ConstantRange with limit/offset
// expressions is "custom"; CustomRange is always custom.
//
// ConstantRange's `LimitExpression` and `OffsetExpression` are not
// exposed by the gen type, so we read them from raw BSON.
func extractDatabaseRangeGen(source *genMf.DatabaseRetrieveSource) (string, string, databaseRangeKind) {
	switch r := source.Range().(type) {
	case *genMf.ConstantRange:
		if r.SingleObject() {
			return "", "", databaseRangeFirst
		}
		limit := genMf.ConstantRangeLimitExpression(r)
		offset := genMf.ConstantRangeOffsetExpression(r)
		if limit != "" || offset != "" {
			return limit, offset, databaseRangeCustom
		}
		return "", "", databaseRangeAll
	case *genMf.CustomRange:
		return r.LimitExpression(), r.OffsetExpression(), databaseRangeCustom
	default:
		return "", "", databaseRangeAll
	}
}

// parseReverseAssociationRetrieveGen mirrors legacy
// `parseReverseAssociationRetrieve` over the gen DatabaseRetrieveSource.
// Returns (startVar, assocName, true) when the source's XPath is a
// single `[Module.Assoc = $Start]` predicate, no sorting, no range, and
// the association exists with the source entity as its FROM side.
//
// `XPathConstraint()` is read with the same legacy fallback used by
// the database renderer (gen alias mismatch — real key is the
// lowercase-`p` `XpathConstraint`).
func parseReverseAssociationRetrieveGen(
	ctx *ExecContext,
	source *genMf.DatabaseRetrieveSource,
	entityName string,
) (string, string, bool) {
	if ctx == nil || ctx.Backend == nil || source == nil || entityName == "" {
		return "", "", false
	}
	if len(collectDatabaseSortPartsGen(source)) > 0 {
		return "", "", false
	}
	if _, _, kind := extractDatabaseRangeGen(source); kind != databaseRangeAll {
		return "", "", false
	}
	xpath := source.XPathConstraint()
	if xpath == "" {
		xpath = genMf.DatabaseRetrieveSourceXpathConstraint(source)
	}
	assocName, startVar, ok := parseReverseAssociationXPath(xpath)
	if !ok || !databaseRetrieveMatchesAssociationTarget(ctx, entityName, assocName) {
		return "", "", false
	}
	return startVar, assocName, true
}

// ────────────────────────────────────────────────────────
// LogMessageAction
// ────────────────────────────────────────────────────────

// formatLogMessageActionGen emits
// `log <level> node <node-expr> '<msg>'[ with ({1} = …, {2} = …)];`.
//
// Legacy parity:
//   - Level: lower-cased; defaults to "info" when missing (legacy uses
//     the bare string "info" — we mirror that, not the gen
//     `LogLevelInfo` ("Info") constant).
//   - Node:  Mendix expression; defaults to `defaultLogNodeExpression`
//     (the constant `'Application'`) when empty.
//   - Message: the `MessageTemplate` is a `Microflows$StringTemplate`
//     whose `Text` field is the template body. When missing, the
//     literal `'Message'` is emitted (already pre-quoted, mirroring
//     legacy).
//   - Parameters: positional `{N}` placeholders rendered from the
//     template's `Arguments` PartList of `Microflows$TemplateArgument`.
func formatLogMessageActionGen(a *genMf.LogMessageAction) string {
	level := strings.ToLower(strings.TrimSpace(a.Level()))
	if level == "" {
		level = "info"
	}
	node := a.Node()
	if node == "" {
		node = defaultLogNodeExpression
	}

	message := "'Message'"
	tmpl, _ := a.MessageTemplate().(*genMf.StringTemplate)
	if tmpl != nil {
		if text := tmpl.Text(); text != "" {
			message = mdlQuote(text)
		}
	}

	withClause := ""
	if tmpl != nil {
		params := collectStringTemplateArgsGen(tmpl)
		if len(params) > 0 {
			withClause = fmt.Sprintf(" with (%s)", strings.Join(params, ", "))
		}
	}

	return fmt.Sprintf("log %s node %s %s%s;", level, node, message, withClause)
}

// collectStringTemplateArgsGen renders the template's expression
// arguments as `{N} = expr` slots (positional, 1-indexed). Mirrors the
// legacy `LogMessageAction.TemplateParameters` slice. Empty arguments
// are skipped.
//
// gen `StringTemplate.ArgumentsItems()` decodes the BSON key
// `"Arguments"`, but real MPRs store the array under the legacy key
// `"Parameters"`. We try the typed getter first, then fall back to
// reading the raw `Parameters` array.
func collectStringTemplateArgsGen(tmpl *genMf.StringTemplate) []string {
	exprs := stringTemplateArgsFromGen(tmpl)
	if len(exprs) == 0 {
		exprs = stringTemplateArgsFromRaw(tmpl.Raw())
	}
	var out []string
	for i, expr := range exprs {
		out = append(out, fmt.Sprintf("{%d} = %s", i+1, expr))
	}
	return out
}

// stringTemplateArgsFromGen pulls the gen-typed argument expressions.
func stringTemplateArgsFromGen(tmpl *genMf.StringTemplate) []string {
	var exprs []string
	for _, arg := range tmpl.ArgumentsItems() {
		ta, ok := arg.(*genMf.TemplateArgument)
		if !ok || ta == nil {
			continue
		}
		if expr := ta.Expression(); expr != "" {
			exprs = append(exprs, expr)
		}
	}
	return exprs
}

// stringTemplateArgsFromRaw decodes the legacy `Parameters` array on a
// raw StringTemplate document via the genMf supplement (which owns BSON
// decoding). Each entry is a Microflows$TemplateArgument with an
// `Expression` string field.
func stringTemplateArgsFromRaw(raw []byte) []string {
	return genMf.StringTemplateArgsFromRaw(raw)
}

// ────────────────────────────────────────────────────────
// DownloadFileAction
// ────────────────────────────────────────────────────────

// formatDownloadFileActionGen emits
// `download file $FileVar[ show in browser];`. Legacy parity: the
// variable name is prefixed with `$` if it isn't already; the `show
// in browser` suffix is appended when `ShowFileInBrowser()` is true.
func formatDownloadFileActionGen(a *genMf.DownloadFileAction) string {
	fileDocument := a.FileDocumentVariableName()
	if fileDocument != "" && !strings.HasPrefix(fileDocument, "$") {
		fileDocument = "$" + fileDocument
	}
	result := fmt.Sprintf("download file %s", fileDocument)
	if a.ShowFileInBrowser() {
		result += " show in browser"
	}
	return result + ";"
}

// ────────────────────────────────────────────────────────
// ValidationFeedbackAction
// ────────────────────────────────────────────────────────

// formatValidationFeedbackActionGen emits
// `validation feedback $Obj[/Attr|/Module.Assoc] message '<msg>';`.
//
// Legacy parity:
//   - Object variable: `$` is prepended when the bare name is set.
//   - Attribute path: only the last dot segment of the attribute QN is
//     appended to the variable (e.g. `Mod.Entity.Subject` → `/Subject`).
//     If the attribute QN has fewer than three segments, only the
//     bare variable is emitted (no slash). This matches legacy's
//     `len(parts) >= 3` guard.
//   - Association path: rendered verbatim (legacy makes no module-prefix
//     trim for associations in this surface).
//   - Message text: pulled from `FeedbackTemplate -> Text` translations
//     using `pickTextTranslationGen` (en_US first, otherwise the first
//     non-empty translation). Defaults to literal `'...'` when no
//     translation is available — the placeholder is already pre-quoted
//     and is NOT passed through `mdlQuote` a second time.
func formatValidationFeedbackActionGen(a *genMf.ValidationFeedbackAction) string {
	msgText := "'...'"
	if tmpl, ok := a.FeedbackTemplate().(*genMf.TextTemplate); ok && tmpl != nil {
		if text, ok := tmpl.Text().(*genTx.Text); ok && text != nil {
			if picked, found := pickTextTranslationGen(text); found {
				msgText = mdlQuote(picked)
			}
		}
	}

	// gen `ObjectVariableName()` is bound to the BSON key
	// `"ObjectVariableName"`, but real Mendix MPRs store the value
	// under `"ValidationVariableName"` (the legacy key). Fall back to
	// reading the raw BSON when the typed getter returns empty.
	objVar := a.ObjectVariableName()
	if objVar == "" {
		objVar = genMf.ValidationFeedbackActionValidationVariableName(a)
	}
	varName := ensureDollar(objVar)
	attrPath := varName
	if attrName := a.AttributeQualifiedName(); attrName != "" {
		parts := strings.Split(attrName, ".")
		if len(parts) >= 3 {
			attrPath = varName + "/" + parts[len(parts)-1]
		}
	} else if assocName := a.AssociationQualifiedName(); assocName != "" {
		attrPath = varName + "/" + assocName
	}
	var objExprs []string
	if tmpl, ok := a.FeedbackTemplate().(*genMf.TextTemplate); ok && tmpl != nil {
		for _, elem := range tmpl.ArgumentsItems() {
			if arg, ok := elem.(*genMf.TemplateArgument); ok {
				if expr := arg.Expression(); expr != "" {
					objExprs = append(objExprs, expr)
				}
			}
		}
	}
	if len(objExprs) > 0 {
		return fmt.Sprintf("validation feedback %s message %s objects [%s];",
			attrPath, msgText, strings.Join(objExprs, ", "))
	}
	return fmt.Sprintf("validation feedback %s message %s;", attrPath, msgText)
}
