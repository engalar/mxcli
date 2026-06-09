// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.2.f — External integration family formatters (gen-typed).
//
// This file implements the gen-typed counterpart to legacy
// `cmd_microflows_format_action.go` for the eight external-integration
// action kinds below.
//
//   | gen Go type                                      | BSON $Type                                           | MDL keyword                                                |
//   |--------------------------------------------------|------------------------------------------------------|------------------------------------------------------------|
//   | *genMf.CallExternalAction                        | Microflows$CallExternalAction                        | `[$X =] call external action <Service>.<Action>(p = …);`   |
//   | *genMf.RestCallAction                            | Microflows$RestCallAction                            | `[$X =] rest call <method> '<url>'\n    …\n    returns …;` |
//   | *genMf.RestOperationCallAction                   | Microflows$RestOperationCallAction                   | `[$X =] send rest request <Op>\n    with (…)\n    body …;` |
//   | *genDC.ExecuteDatabaseQueryAction                | DatabaseConnector$ExecuteDatabaseQueryAction         | `[$X =] execute database query <Query> [dynamic …] (…) [\n    connection (…)];` |
//   | *genMf.ImportXmlAction                           | Microflows$ImportXmlAction                           | `[$X =] import from mapping <Mapping>($SourceVar);`        |
//   | *genMf.ExportXmlAction                           | Microflows$ExportXmlAction                           | `[$X =] export to mapping <Mapping>($SourceVar);`          |
//   | *genMf.TransformJsonAction                       | Microflows$TransformJsonAction                       | `[$X =] transform $Input with <Transformer>;`              |
//   | *genMf.WebServiceCallAction                      | Microflows$WebServiceCallAction                      | `[$X =] call web service <Service>\n    operation …\n    …;` |
//
// Notable gen / legacy differences preserved verbatim (each is a known
// BSON-key / typed-getter alignment gap surfaced during Stage 3.2.2.{d,e}
// and reused here):
//
//  1. `RestCallAction` has no `OutputVariable` field on the gen type.
//     Real MPRs store the output variable on the `ResultHandling`
//     element (gen `ResultHandling.OutputVariableName()` is bound to
//     BSON key `OutputVariableName`, but Studio Pro writes the value
//     under `ResultVariableName`). The formatter reads via the typed
//     getter first and falls back to the legacy raw key for fixture
//     loads.
//
//  2. `RestOperationCallAction` carries the output variable inside the
//     nested `OutputVariable` element (`*genMf.OutputVariable`). The
//     gen type exposes only `VariableName()` on that element — no raw
//     fallback is needed because the BSON key matches.
//
//  3. `RestCallAction`'s URL template is stored in the BSON sub-document
//     `CustomLocationTemplate` as a `Microflows$StringTemplate` (Text +
//     Arguments). Gen exposes this through `HttpConfiguration.CustomLocationTemplate()`.
//     The legacy formatter reads `Text` directly and renders the
//     `Arguments` as `{1} = …, {2} = …` slots — same as `LogMessageAction`.
//
//  4. `RestCallAction`'s headers live under BSON key `HttpHeaderEntries`
//     (gen `HeaderEntriesItems()`). Each entry is a `*genMf.HttpHeaderEntry`
//     with `Key()` and `Value()` getters. Legacy renders them as
//     `\n    header '<name>' = <expr>` lines.
//
//  5. `RestCallAction` ResultHandling is dispatched on the parent's
//     `ResultHandlingType()` enum. The gen `ResultHandling` type is a
//     single struct (no subclasses) that carries the variable name plus
//     an optional `ImportMappingCall` for the Mapping case. Legacy
//     splits this into four sub-types; the gen formatter dispatches on
//     the discriminator string instead.
//
//  6. `RestOperationCallAction` parameter mappings come from two
//     PartLists: `ParameterMappings` (path params) and
//     `QueryParameterMappings` (query params). Each entry is a
//     `*genMf.RestParameterMapping` (path) or `*genMf.RestOperationParameterMapping`
//     (query). Both expose `ParameterQualifiedName()` and `Value()` —
//     and the BSON keys for the qualified name `Parameter`/`QueryParameter`
//     differ. The typed getter is consulted first; raw-BSON fallback
//     reads `QueryParameter` for query mappings.
//
//  7. `WebServiceCallAction` has a richer surface than the legacy
//     formatter exposes: legacy emits an `RawBSON`-driven
//     `call web service raw '<base64>';` form when any unmodeled BSON
//     field is present. The gen type carries every modeled field but
//     does not preserve unmodeled ones — there's no equivalent fallback
//     path. The gen formatter therefore always emits the structured
//     form; the rare unmodeled-field case (legacy's `webServiceActionRequiresRawBSON`)
//     loses fidelity but stays parseable.
//
//  8. `ExportXmlAction` legacy reads the output variable from the nested
//     `OutputMethod -> OutputVariableName` — the gen `OutputMethod`
//     base type has no fields. We read from raw BSON via
//     `codec.ReadBSONFieldString` to preserve parity.
//
//  9. `ImportXmlAction` legacy reads `ResultHandling -> ResultVariableName`
//     and `ResultHandling -> ImportMappingCall -> Mapping` (or the
//     legacy alias `ReturnValueMapping`). Gen `ResultHandling` exposes
//     `OutputVariableName()` (mapped to `OutputVariableName`, not
//     `ResultVariableName`) and `ImportMappingCall()` (which has
//     `MappingQualifiedName()`). We read both gen first and fall back
//     to raw `ResultVariableName` and `ReturnValueMapping` for fixture
//     loads.
//
//  10. `TransformJsonAction.TransformationQualifiedName()` is gen-aliased
//      to BSON key `Transformation` (singular, matches legacy). No raw
//      fallback needed.
//
// All output strings are 1:1 with the legacy formatters in
// `cmd_microflows_format_action.go` so the migrated body diff against
// the SDK path stays empty for these action kinds.

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDC "github.com/mendixlabs/mxcli/modelsdk/gen/databaseconnector"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// ────────────────────────────────────────────────────────
// CallExternalAction
// ────────────────────────────────────────────────────────

// formatCallExternalActionGen emits one of:
//
//	`$Var = call external action <Service>.<Name>(p = …);`
//	`call external action <Service>.<Name>(p = …);`
//
// Mirrors legacy `CallExternalAction` handling: missing
// ConsumedODataService falls back to "ODataService"; missing Name falls
// back to "Action". Parameter mappings are rendered as
// `<bare-name> = <expr>` using `RestOperationParameterMapping` (the same
// element type used by `RestOperationCallAction`).
//
// The output variable is stored on the action itself (gen
// `VariableName()`) and gated by `UseReturnVariable`. Note: gen exposes
// no `UseReturnVariable` field for this action — legacy uses it but the
// gen type doesn't surface the flag. We treat a non-empty `VariableName`
// as the "use return variable" signal, matching the surface MDL output.
func formatCallExternalActionGen(a *genMf.CallExternalAction) string {
	serviceName := strings.TrimSpace(a.ConsumedODataServiceQualifiedName())
	if serviceName == "" {
		serviceName = "ODataService"
	}
	actionName := strings.TrimSpace(a.Name())
	if actionName == "" {
		actionName = "Action"
	}

	var params []string
	for _, m := range a.ParameterMappingsItems() {
		pm, ok := m.(*genMf.RestOperationParameterMapping)
		if !ok || pm == nil {
			continue
		}
		paramName := lastDotSegment(pm.ParameterQualifiedName())
		params = append(params, fmt.Sprintf("%s = %s", paramName, pm.Value()))
	}
	paramStr := strings.Join(params, ", ")

	if outVar := strings.TrimSpace(a.VariableName()); outVar != "" {
		return fmt.Sprintf("$%s = call external action %s.%s(%s);", outVar, serviceName, actionName, paramStr)
	}
	return fmt.Sprintf("call external action %s.%s(%s);", serviceName, actionName, paramStr)
}

// ────────────────────────────────────────────────────────
// RestCallAction
// ────────────────────────────────────────────────────────

// formatRestCallActionGen emits the multi-line `rest call` statement.
//
// Surface mirrors legacy verbatim:
//
//	[$Var = ]rest call <method> '<url>'[ with ({1} = …, …)]
//	    [header '<name>' = <expr>]…
//	    [auth basic <user> password <pass>]
//	    [body '<template>'[ with (…)] | body mapping <id>[ from $var]]
//	    [timeout <expr>]
//	    returns <handler>;
//
// HTTP method enum maps to the legacy lower-cased verb; missing /
// unrecognised methods fall back to "get" (legacy default).
//
// `ResultHandlingType` is the dispatch discriminator since gen folds
// the four legacy result subtypes into a single `ResultHandling`
// struct. We read the type off the parent action and the variable
// names / mapping / entity off the child element.
func formatRestCallActionGen(a *genMf.RestCallAction) string {
	var sb strings.Builder

	resultHandlingType := a.ResultHandlingType()
	rh, _ := a.ResultHandling().(*genMf.ResultHandling)

	outputVar := readRestCallOutputVarGen(rh)
	if outputVar != "" {
		sb.WriteString("$")
		sb.WriteString(outputVar)
		sb.WriteString(" = ")
	}

	sb.WriteString("rest call ")

	method := "get"
	httpConfig, _ := a.HttpConfiguration().(*genMf.HttpConfiguration)
	if httpConfig != nil {
		switch httpConfig.HttpMethod() {
		case genMf.HttpMethodGet:
			method = "get"
		case genMf.HttpMethodPost:
			method = "post"
		case genMf.HttpMethodPut:
			method = "put"
		case genMf.HttpMethodPatch:
			method = "patch"
		case genMf.HttpMethodDelete:
			method = "delete"
		}
	}
	sb.WriteString(method)
	sb.WriteString(" ")

	url := "''"
	var urlParams []string
	if httpConfig != nil {
		if tmpl, ok := httpConfig.CustomLocationTemplate().(*genMf.StringTemplate); ok && tmpl != nil {
			if text := tmpl.Text(); text != "" {
				url = mdlQuote(text)
			}
			urlParams = stringTemplateArgsFromGen(tmpl)
			if len(urlParams) == 0 {
				urlParams = stringTemplateArgsFromRaw(tmpl.Raw())
			}
		}
	}
	sb.WriteString(url)

	if len(urlParams) > 0 {
		sb.WriteString(" with (")
		for i, expr := range urlParams {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("{%d} = %s", i+1, expr))
		}
		sb.WriteString(")")
	}

	if httpConfig != nil {
		for _, h := range httpConfig.HeaderEntriesItems() {
			he, ok := h.(*genMf.HttpHeaderEntry)
			if !ok || he == nil {
				continue
			}
			sb.WriteString("\n    header ")
			sb.WriteString(mdlQuote(he.Key()))
			sb.WriteString(" = ")
			sb.WriteString(he.Value())
		}

		if httpConfig.UseAuthentication() {
			sb.WriteString("\n    auth basic ")
			sb.WriteString(httpConfig.HttpAuthenticationUserName())
			sb.WriteString(" password ")
			sb.WriteString(httpConfig.AuthenticationPassword())
		}
	}

	switch req := a.RequestHandling().(type) {
	case *genMf.CustomRequestHandling:
		if tmpl, ok := req.Template().(*genMf.StringTemplate); ok && tmpl != nil {
			if text := tmpl.Text(); text != "" {
				sb.WriteString("\n    body ")
				sb.WriteString(mdlQuote(text))
				params := stringTemplateArgsFromGen(tmpl)
				if len(params) == 0 {
					params = stringTemplateArgsFromRaw(tmpl.Raw())
				}
				if len(params) > 0 {
					sb.WriteString(" with (")
					for i, expr := range params {
						if i > 0 {
							sb.WriteString(", ")
						}
						sb.WriteString(fmt.Sprintf("{%d} = %s", i+1, expr))
					}
					sb.WriteString(")")
				}
			}
		}
	case *genMf.MappingRequestHandling:
		if mq := req.MappingQualifiedName(); mq != "" {
			sb.WriteString("\n    body mapping ")
			sb.WriteString(mq)
			if pv := req.MappingArgumentVariableName(); pv != "" {
				sb.WriteString(" from $")
				sb.WriteString(pv)
			}
		}
	}

	if to := a.TimeOutExpression(); to != "" {
		sb.WriteString("\n    timeout ")
		sb.WriteString(to)
	}

	sb.WriteString("\n    returns ")
	switch resultHandlingType {
	case genMf.ResultHandlingTypeString:
		sb.WriteString("String")
	case genMf.ResultHandlingTypeHttpResponse:
		sb.WriteString("response")
	case genMf.ResultHandlingTypeMapping:
		sb.WriteString("mapping ")
		mappingID, entityID, singleObject := readMappingResultHandlingGen(rh)
		sb.WriteString(mappingID)
		if entityID != "" {
			if singleObject {
				sb.WriteString(" as ")
			} else {
				sb.WriteString(" as list of ")
			}
			sb.WriteString(entityID)
		}
	case genMf.ResultHandlingTypeNone:
		sb.WriteString("Nothing")
	default:
		sb.WriteString("String")
	}

	sb.WriteString(";")
	return sb.String()
}

// readRestCallOutputVarGen extracts the output variable name from a
// gen `ResultHandling`. Legacy reads from BSON `ResultVariableName`,
// gen `OutputVariableName()` is aliased to BSON `OutputVariableName`.
// We try the typed getter first, then fall back to the legacy raw key
// so direct-construction tests and fixture loads both work.
func readRestCallOutputVarGen(rh *genMf.ResultHandling) string {
	if rh == nil {
		return ""
	}
	if v := strings.TrimSpace(rh.OutputVariableName()); v != "" {
		return v
	}
	return strings.TrimSpace(genMf.ResultHandlingResultVariableName(rh))
}

// readMappingResultHandlingGen pulls (mappingID, entityID, singleObject)
// from a `ResultHandling` element configured for the Mapping result.
// The mapping reference comes from the nested `ImportMappingCall.Mapping`
// (or legacy alias `ReturnValueMapping`); the entity name comes from the
// `VariableType` element under the result handling. SingleObject is set
// either via `Range.SingleObject` or — for the gen-decoded case — via
// the `ObjectType` discriminator on the `VariableType` element.
func readMappingResultHandlingGen(rh *genMf.ResultHandling) (string, string, bool) {
	if rh == nil {
		return "", "", false
	}
	mappingID := ""
	if call, ok := rh.ImportMappingCall().(*genMf.ImportMappingCall); ok && call != nil {
		mappingID = call.MappingQualifiedName()
		if mappingID == "" {
			mappingID = genMf.ImportMappingCallReturnValueMapping(call)
		}
	}
	// VariableType is a polymorphic element under ResultHandling. For
	// `DataTypes$ObjectType` the legacy parser sets SingleObject=true.
	entityID := ""
	singleObject := false
	if vt := rh.VariableType(); vt != nil {
		if base, ok := vt.(*element.Base); ok && base != nil {
			if base.TypeName() == "DataTypes$ObjectType" {
				singleObject = true
			}
			entityID = genMf.RawFieldStringFromBase(base, "Entity")
		}
	}
	if call, ok := rh.ImportMappingCall().(*genMf.ImportMappingCall); ok && call != nil {
		if r, ok := call.Range().(*genMf.ConstantRange); ok && r != nil {
			if r.SingleObject() {
				singleObject = true
			}
		}
	}
	return mappingID, entityID, singleObject
}

// ────────────────────────────────────────────────────────
// RestOperationCallAction
// ────────────────────────────────────────────────────────

// formatRestOperationCallActionGen emits:
//
//	[$Var = ]send rest request <Op>
//	    with ($p1 = expr, $p2 = expr, …)
//	    body $bodyVar;
//
// Path and query parameters are merged in declaration order: path
// params first, then query. Each parameter name uses the bare name
// (last dot segment of the parameter qualified name).
func formatRestOperationCallActionGen(a *genMf.RestOperationCallAction) string {
	var sb strings.Builder

	if ov, ok := a.OutputVariable().(*genMf.OutputVariable); ok && ov != nil {
		if name := ov.VariableName(); name != "" {
			sb.WriteString("$")
			sb.WriteString(name)
			sb.WriteString(" = ")
		}
	}

	sb.WriteString("send rest request ")
	sb.WriteString(a.OperationQualifiedName())

	type pair struct {
		name  string
		value string
	}
	var allParams []pair
	for _, m := range a.ParameterMappingsItems() {
		pm, ok := m.(*genMf.RestParameterMapping)
		if !ok || pm == nil {
			continue
		}
		allParams = append(allParams, pair{
			name:  lastDotSegment(pm.ParameterQualifiedName()),
			value: pm.Value(),
		})
	}
	for _, m := range a.QueryParameterMappingsItems() {
		// Query mappings are decoded as `*genMf.RestOperationParameterMapping`;
		// the BSON key for the qualified name is `QueryParameter` (gen
		// uses `Parameter`). Fall back to the raw read when the typed
		// getter is empty so fixture loads work.
		pm, ok := m.(*genMf.RestOperationParameterMapping)
		if !ok || pm == nil {
			continue
		}
		name := pm.ParameterQualifiedName()
		if name == "" {
			name = genMf.RestOperationParameterMappingQueryParameter(pm)
		}
		allParams = append(allParams, pair{
			name:  lastDotSegment(name),
			value: pm.Value(),
		})
	}
	if len(allParams) > 0 {
		sb.WriteString("\n    with (")
		for i, p := range allParams {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString("$")
			sb.WriteString(p.name)
			sb.WriteString(" = ")
			sb.WriteString(p.value)
		}
		sb.WriteString(")")
	}

	if bv, ok := a.BodyVariable().(*genMf.BodyVariable); ok && bv != nil {
		if name := bv.VariableName(); name != "" {
			sb.WriteString("\n    body $")
			sb.WriteString(name)
		}
	}

	sb.WriteString(";")
	return sb.String()
}

// ────────────────────────────────────────────────────────
// ExecuteDatabaseQueryAction
// ────────────────────────────────────────────────────────

// formatExecuteDatabaseQueryActionGen emits:
//
//	[$Var = ]execute database query <Query>[ dynamic <expr>][ (p1 = v1, …)]
//	    [connection (p1 = v1, …)];
//
// Mirrors legacy verbatim. `Query` is a BY_NAME_REFERENCE, surfaced
// here as `QueryQualifiedName()`.
func formatExecuteDatabaseQueryActionGen(a *genDC.ExecuteDatabaseQueryAction) string {
	var sb strings.Builder

	if outVar := a.OutputVariableName(); outVar != "" {
		sb.WriteString(fmt.Sprintf("$%s = ", outVar))
	}

	sb.WriteString("execute database query ")
	sb.WriteString(a.QueryQualifiedName())

	if dq := a.DynamicQuery(); dq != "" {
		sb.WriteString(fmt.Sprintf(" dynamic %s", dq))
	}

	if items := a.ParameterMappingsItems(); len(items) > 0 {
		sb.WriteString(" (")
		first := true
		for _, m := range items {
			pm, ok := m.(*genDC.QueryParameterMapping)
			if !ok || pm == nil {
				continue
			}
			if !first {
				sb.WriteString(", ")
			}
			first = false
			sb.WriteString(fmt.Sprintf("%s = %s", pm.ParameterName(), pm.Value()))
		}
		sb.WriteString(")")
	}

	if items := a.ConnectionParameterMappingsItems(); len(items) > 0 {
		sb.WriteString("\n    connection (")
		first := true
		for _, m := range items {
			cm, ok := m.(*genDC.ConnectionParameterMapping)
			if !ok || cm == nil {
				continue
			}
			if !first {
				sb.WriteString(", ")
			}
			first = false
			sb.WriteString(fmt.Sprintf("%s = %s", cm.ParameterName(), cm.Value()))
		}
		sb.WriteString(")")
	}

	sb.WriteString(";")
	return sb.String()
}

// ────────────────────────────────────────────────────────
// ImportXmlAction
// ────────────────────────────────────────────────────────

// formatImportXmlActionGen emits `[$Var =] import from mapping <Mapping>($SourceVar);`.
//
// Reads the mapping reference and result variable from the nested
// `ResultHandling` element. Both the gen-typed getters and the legacy
// raw keys are tried (`ResultVariableName` for the variable,
// `ReturnValueMapping` for older mapping refs).
func formatImportXmlActionGen(a *genMf.ImportXmlAction) string {
	var sb strings.Builder

	mappingName := ""
	resultVar := ""
	if rh, ok := a.ResultHandling().(*genMf.ResultHandling); ok && rh != nil {
		resultVar = readRestCallOutputVarGen(rh)
		if call, ok := rh.ImportMappingCall().(*genMf.ImportMappingCall); ok && call != nil {
			mappingName = call.MappingQualifiedName()
			if mappingName == "" {
				mappingName = genMf.ImportMappingCallReturnValueMapping(call)
			}
		}
	}

	if resultVar != "" {
		sb.WriteString("$")
		sb.WriteString(resultVar)
		sb.WriteString(" = ")
	}

	sb.WriteString("import from mapping ")
	sb.WriteString(mappingName)
	sb.WriteString("($")
	sb.WriteString(a.XmlDocumentVariableName())
	sb.WriteString(");")

	return sb.String()
}

// ────────────────────────────────────────────────────────
// ExportXmlAction
// ────────────────────────────────────────────────────────

// formatExportXmlActionGen emits `[$Var =] export to mapping <Mapping>[($SourceVar)];`.
//
// Output variable lives on the `OutputMethod` element under raw BSON
// key `OutputVariableName` (gen `OutputMethod` exposes no fields).
// The mapping reference and parameter variable live on the
// `ResultHandling` element as a `Microflows$MappingRequestHandling`
// (legacy uses raw BSON keys `MappingId` + `MappingVariableName`).
//
// Note the unusual mapping in legacy: the `ResultHandling` field is
// reused as a *RequestHandling* on this action — that's a Mendix
// quirk preserved here for byte-identical output.
func formatExportXmlActionGen(a *genMf.ExportXmlAction) string {
	var sb strings.Builder

	if om := a.OutputMethod(); om != nil {
		if base, ok := om.(*element.Base); ok && base != nil {
			if v := genMf.RawFieldStringFromBase(base, "OutputVariableName"); v != "" {
				sb.WriteString("$")
				sb.WriteString(v)
				sb.WriteString(" = ")
			}
		}
	}

	sb.WriteString("export to mapping ")

	mappingName := ""
	paramVar := ""
	if rh := a.ResultHandling(); rh != nil {
		if base, ok := rh.(*element.Base); ok && base != nil {
			mappingName = genMf.RawFieldStringFromBase(base, "MappingId")
			paramVar = genMf.RawFieldStringFromBase(base, "MappingVariableName")
		} else if mr, ok := rh.(*genMf.MappingRequestHandling); ok && mr != nil {
			mappingName = mr.MappingQualifiedName()
			paramVar = mr.MappingArgumentVariableName()
		}
	}

	sb.WriteString(mappingName)
	if paramVar != "" {
		sb.WriteString("($")
		sb.WriteString(paramVar)
		sb.WriteString(")")
	}
	sb.WriteString(";")

	return sb.String()
}

// ────────────────────────────────────────────────────────
// TransformJsonAction
// ────────────────────────────────────────────────────────

// formatTransformJsonActionGen emits `[$Var =] transform $Input with <Transformer>;`.
// `Transformation` is a BY_NAME_REFERENCE, surfaced as
// `TransformationQualifiedName()`. Mirrors legacy verbatim.
func formatTransformJsonActionGen(a *genMf.TransformJsonAction) string {
	var sb strings.Builder
	if outVar := a.OutputVariableName(); outVar != "" {
		sb.WriteString("$")
		sb.WriteString(outVar)
		sb.WriteString(" = ")
	}
	sb.WriteString("transform $")
	sb.WriteString(a.InputVariableName())
	sb.WriteString(" with ")
	sb.WriteString(a.TransformationQualifiedName())
	sb.WriteString(";")
	return sb.String()
}

// ────────────────────────────────────────────────────────
// WebServiceCallAction
// ────────────────────────────────────────────────────────

// formatWebServiceCallActionGen emits the multi-line
// `call web service <Service>` statement, with one optional clause per
// modeled field:
//
//	[$Var = ]call web service <Service>
//	[\n    operation '<OperationName>']
//	[\n    send mapping <SendMapping>]
//	[\n    receive mapping <ReceiveMapping>]
//	[\n    timeout <expr>];
//
// Differences from legacy:
//
//   - The legacy `RawBSON` fallback (used when an unmodeled BSON field
//     was present on the original `Microflows$WebServiceCallAction`)
//     has no gen counterpart — gen does not preserve unmodeled fields
//     on this action. The structured form is always emitted.
//   - The output variable is read from the result handling's variable
//     name (legacy gen-incompleteness fallback covers both possible BSON
//     keys, same as `RestCallAction`).
//   - Mapping references come from typed getters where available,
//     falling back to raw BSON for the keys legacy used.
//
// Mapping reference rendering uses the legacy `formatWebServiceReference`
// helper so qualified-name vs literal-string formatting stays identical.
func formatWebServiceCallActionGen(a *genMf.WebServiceCallAction) string {
	prefix := ""

	rh, _ := a.ResultHandling().(*genMf.ResultHandling)
	outputVar := readRestCallOutputVarGen(rh)
	if outputVar != "" {
		prefix = fmt.Sprintf("$%s = ", outputVar)
	}

	serviceRef := strings.TrimSpace(a.ImportedWebServiceQualifiedName())
	if serviceRef == "" {
		// Legacy reads BSON key `ImportedService`; the gen alias is
		// `ImportedWebService`. Fall back to the legacy key so fixture
		// loads (which carry the raw key) still resolve.
		serviceRef = genMf.WebServiceCallActionImportedService(a)
	}

	parts := []string{prefix + "call web service " + formatWebServiceReference(serviceRef)}

	if op := a.OperationName(); op != "" {
		parts = append(parts, "operation "+formatWebServiceReference(op))
	}

	sendMapping, receiveMapping := readWebServiceMappingsGen(a, rh)
	if sendMapping != "" {
		parts = append(parts, "send mapping "+formatWebServiceReference(sendMapping))
	}
	if receiveMapping != "" {
		parts = append(parts, "receive mapping "+formatWebServiceReference(receiveMapping))
	}

	if to := a.TimeOutExpression(); to != "" {
		parts = append(parts, "timeout "+strings.TrimRight(to, " \t\n\r"))
	}
	return strings.Join(parts, "\n") + ";"
}

// readWebServiceMappingsGen extracts (sendMapping, receiveMapping) from
// the action's request and result handling elements. Legacy reads:
//
//   - send: `RequestHandling -> ExportMappingCall -> Mapping`
//   - receive: `NewResultHandling -> ImportMappingCall -> ReturnValueMapping`
//
// Gen exposes `RequestBodyHandling()` (not `RequestHandling()` — the
// legacy field name has no gen counterpart). Raw-BSON walks for the
// historical mapping shapes live in the genMf supplement so this file
// stays free of bson/codec imports.
func readWebServiceMappingsGen(a *genMf.WebServiceCallAction, rh *genMf.ResultHandling) (string, string) {
	send := ""
	if req := a.RequestBodyHandling(); req != nil {
		if base, ok := req.(*element.Base); ok && base != nil {
			send = genMf.WebServiceRequestBodyExportMapping(base)
		}
	}

	receive := ""
	if rh != nil {
		if call, ok := rh.ImportMappingCall().(*genMf.ImportMappingCall); ok && call != nil {
			receive = call.MappingQualifiedName()
			if receive == "" {
				receive = genMf.ImportMappingCallReturnValueMapping(call)
			}
		}
	}

	legacySend, legacyReceive := genMf.WebServiceLegacyMappings(a)
	if send == "" {
		send = legacySend
	}
	if receive == "" {
		receive = legacyReceive
	}
	return send, receive
}
