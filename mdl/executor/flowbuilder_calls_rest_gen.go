// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.g5c — gen-typed REST CALL adder.
//
// gen `RestCallAction` is the largest single action in the schema:
// HTTP method + URL template + headers + auth + request body
// (CustomRequestHandling | MappingRequestHandling) + result handling
// (gen `ResultHandling` element + ResultHandlingType discriminator
// covering String / HttpResponse / Mapping / None) + timeout + error
// handling.
//
// gen-vs-legacy schema notes:
//
//   - gen `HttpConfiguration` field naming differs from legacy:
//     the URL template lives on `CustomLocationTemplate` (not
//     `LocationTemplate`); headers live on `HeaderEntries` as
//     `*HttpHeaderEntry` (not `CustomHeaders` of `*HttpHeader`).
//
//   - URL template is a `*genMf.StringTemplate` (Text + Arguments)
//     same as the LogMessage template format.
//
//   - gen folds the four legacy `ResultHandling` sub-types
//     (Mapping/String/HttpResponse/None) into a single
//     `*ResultHandling` element + a `ResultHandlingType` enum on the
//     parent action. The describer dispatches on the enum; we set
//     both on write.
//
//   - For Result=Mapping, the mapping reference goes inside an
//     `ImportMappingCall` sub-element (same shape as
//     ImportFromMapping in g5a). `ForceSingleOccurrence` controls
//     single-vs-list cardinality.
//
//   - Basic-auth username/password setters take an `element.Element`
//     (polymorphic) — for fresh-author offline writes we leave them
//     unset and only flip `UseAuthentication=true` so describer +
//     Studio Pro know auth is enabled. The string variants
//     (`HttpAuthenticationUserName` / `AuthenticationPassword`) are
//     legacy fields kept for older MPRs.
//
// Schema-gap tracking (deferred to commit j):
//
//   - URL template Arguments PartList for `URLParams` (legacy
//     `LocationParams`) — not surfaced in the gen schema in a way
//     that matches legacy 1:1; left as a TODO. Most fresh-author
//     URLs are static literals so this gap is rarely hit.
//
//   - HttpAuthenticationUserName / AuthenticationPassword string
//     fields for basic auth — only set when fresh-author MDL
//     supplies plain string username + password literals. Expression
//     auth is deferred (need element.Element wrapping).

package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// addRestCallActionGen emits a `[$Var = ]rest call <method> '<url>'
// [headers] [auth] [body] [timeout] returns <handler>;` activity.
func (fb *flowBuilderGen) addRestCallActionGen(s *ast.RestCallStmt) element.ID {
	httpConfig := fb.buildRestHttpConfigGen(s)

	action := genMf.NewRestCallAction()
	assignFreshID(action)
	action.SetErrorHandlingType(fb.ehTypeGen(s.ErrorHandling))
	action.SetHttpConfiguration(httpConfig)

	// Request handling: dispatch on body type.
	requestHandling, requestType := fb.buildRestRequestHandlingGen(s.Body)
	action.SetRequestHandling(requestHandling)
	action.SetRequestHandlingType(requestType)

	// Result handling: gen folds 4 legacy sub-types into ResultHandling
	// + a discriminator string on the parent action.
	resultHandling, resultType := fb.buildRestResultHandlingGen(s)
	action.SetResultHandling(resultHandling)
	action.SetResultHandlingType(resultType)

	// Timeout — default 300 seconds when no expression supplied.
	if s.Timeout != nil {
		action.SetTimeOutExpression(fb.exprToString(s.Timeout))
	} else {
		action.SetTimeOutExpression("300")
	}

	return fb.genActivityWrap(action, s.ErrorHandling, s.OutputVariable)
}

// buildRestHttpConfigGen constructs the HttpConfiguration element
// (method + URL template + headers + auth flag).
func (fb *flowBuilderGen) buildRestHttpConfigGen(s *ast.RestCallStmt) *genMf.HttpConfiguration {
	hc := genMf.NewHttpConfiguration()
	assignFreshID(hc)

	hc.SetHttpMethod(restHttpMethodGen(s.Method))

	// URL template: wrap in StringTemplate (Text + optional args).
	tmpl := genMf.NewStringTemplate()
	assignFreshID(tmpl)
	tmpl.SetText(restURLTextGen(s, fb))
	hc.SetCustomLocationTemplate(tmpl)

	// Header entries: each becomes *HttpHeaderEntry with rendered value.
	for _, header := range s.Headers {
		he := genMf.NewHttpHeaderEntry()
		assignFreshID(he)
		he.SetKey(header.Name)
		he.SetValue(fb.exprToString(header.Value))
		hc.AddHeaderEntries(he)
	}

	// Basic auth — flip flag; string fields are deferred (see file
	// docstring) because they take element.Element setters.
	if s.Auth != nil {
		hc.SetUseAuthentication(true)
	}

	return hc
}

// restHttpMethodGen normalises the AST HttpMethod to the gen string
// enum value. Defaults to "Get" for unknown methods (legacy parity).
func restHttpMethodGen(m ast.HttpMethod) string {
	switch m {
	case ast.HttpMethodGet:
		return "Get"
	case ast.HttpMethodPost:
		return "Post"
	case ast.HttpMethodPut:
		return "Put"
	case ast.HttpMethodPatch:
		return "Patch"
	case ast.HttpMethodDelete:
		return "Delete"
	}
	return "Get"
}

// restURLTextGen extracts the URL template text from the AST
// expression. String literals use the raw value; complex expressions
// render through exprToString.
func restURLTextGen(s *ast.RestCallStmt, fb *flowBuilderGen) string {
	if lit, ok := s.URL.(*ast.LiteralExpr); ok && lit.Kind == ast.LiteralString {
		if v, ok := lit.Value.(string); ok {
			return v
		}
	}
	return fb.exprToString(s.URL)
}

// buildRestRequestHandlingGen dispatches on s.Body.Type:
//
//   - nil or RestBodyNone → CustomRequestHandling with empty template
//     (matches legacy "no body" default), discriminator = "Custom"
//   - RestBodyCustom → CustomRequestHandling with body template
//   - RestBodyMapping → MappingRequestHandling with mapping ref
//
// Returns (handling element, discriminator string).
func (fb *flowBuilderGen) buildRestRequestHandlingGen(body *ast.RestBody) (element.Element, string) {
	if body == nil || body.Type == ast.RestBodyNone {
		ch := genMf.NewCustomRequestHandling()
		assignFreshID(ch)
		// Empty template → no body emitted by describer.
		tmpl := genMf.NewStringTemplate()
		assignFreshID(tmpl)
		ch.SetTemplate(tmpl)
		return ch, "Custom"
	}

	switch body.Type {
	case ast.RestBodyCustom:
		ch := genMf.NewCustomRequestHandling()
		assignFreshID(ch)
		text := ""
		if lit, ok := body.Template.(*ast.LiteralExpr); ok && lit.Kind == ast.LiteralString {
			if v, ok := lit.Value.(string); ok {
				text = v
			}
		} else if body.Template != nil {
			text = fb.exprToString(body.Template)
		}
		tmpl := genMf.NewStringTemplate()
		assignFreshID(tmpl)
		tmpl.SetText(text)
		ch.SetTemplate(tmpl)
		return ch, "Custom"

	case ast.RestBodyMapping:
		mh := genMf.NewMappingRequestHandling()
		assignFreshID(mh)
		mh.SetMappingQualifiedName(body.MappingName.Module + "." + body.MappingName.Name)
		mh.SetMappingArgumentVariableName(body.SourceVariable)
		return mh, "Mapping"
	}

	// Defensive default — empty custom request handling.
	ch := genMf.NewCustomRequestHandling()
	assignFreshID(ch)
	return ch, "Custom"
}

// buildRestResultHandlingGen dispatches on s.Result.Type and returns
// the (ResultHandling element, discriminator string) pair.
//
// gen folds the four legacy sub-types into a single ResultHandling
// element with the discriminator on the parent action:
//
//   - String       → empty ResultHandling, discriminator "String"
//   - HttpResponse → ResultHandling{OutputVariableName}, discriminator "HttpResponse"
//   - Mapping      → ResultHandling{OutputVariableName + ImportMappingCall},
//     discriminator "Mapping"
//   - None         → empty ResultHandling, discriminator "None"
func (fb *flowBuilderGen) buildRestResultHandlingGen(s *ast.RestCallStmt) (element.Element, string) {
	rh := genMf.NewResultHandling()
	assignFreshID(rh)

	switch s.Result.Type {
	case ast.RestResultString:
		return rh, "String"

	case ast.RestResultResponse:
		rh.SetOutputVariableName(s.OutputVariable)
		return rh, "HttpResponse"

	case ast.RestResultMapping:
		rh.SetOutputVariableName(s.OutputVariable)
		call := genMf.NewImportMappingCall()
		assignFreshID(call)
		call.SetMappingQualifiedName(s.Result.MappingName.Module + "." + s.Result.MappingName.Name)
		// IsList drives ForceSingleOccurrence: list → false, single → true.
		call.SetForceSingleOccurrence(!s.Result.IsList)
		rh.SetImportMappingCall(call)
		return rh, "Mapping"

	case ast.RestResultNone:
		return rh, "None"
	}

	// Defensive default — String result.
	return rh, "String"
}
