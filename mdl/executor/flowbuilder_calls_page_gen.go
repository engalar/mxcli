// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.g3 — gen-typed page-action adders.
//
// Covers:
//
//   - addShowPageActionGen — `show page Mod.Page(...) [for $obj] [title '...'];`
//   - addShowHomePageActionGen — `show home page;`
//   - addShowMessageActionGen — `show message '...' type X [objects [...]];`
//   - addClosePageActionGen — `close page [N];`
//
// Two important gen-vs-legacy schema differences:
//
//   1. Page mappings live on a nested `*genPg.PageSettings` element
//      rather than directly on the action. Legacy flat
//      `ShowPageAction.PageParameterMappings` becomes
//      `action.PageSettings.ParameterMappings`. The page reference
//      itself also lives on PageSettings (`PageQualifiedName`,
//      surfaced via the legacy BSON key `Form`).
//
//   2. ShowMessageAction.Template is a `*genMf.TextTemplate` with the
//      same `Texts$Text → Translation` nesting as
//      ValidationFeedback (cf. flowbuilder_actions_feedback_gen.go).
//      We reuse `buildTextTemplateGen` to assemble it.
//
// CloseFormAction is the gen storage name for what the legacy and
// MDL surface call "close page" — Mendix renamed Page→Form and the
// BSON kept the historical `CloseFormAction` $Type. See
// `cmd_microflows_format_action_gen.go` for the read-side parity.
//
// Schema-gap tracking: none new — every field accessed here has a
// typed setter on the relevant gen element.

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	genTexts "github.com/mendixlabs/mxcli/modelsdk/gen/texts"
)

// addShowPageActionGen emits a `show page Mod.Page(...) [for $obj]
// [title '...'];` activity. Builds a nested PageSettings element
// carrying the page reference, location, parameter mappings, and
// optional title override.
func (fb *flowBuilderGen) addShowPageActionGen(s *ast.ShowPageStmt) element.ID {
	pageQN := s.PageName.Module + "." + s.PageName.Name

	settings := genPg.NewPageSettings()
	assignFreshID(settings)
	settings.SetPageQualifiedName(pageQN)
	settings.SetLocation(showPageLocationGen(s.Location))

	for _, arg := range s.Arguments {
		mapping := genPg.NewPageParameterMapping()
		assignFreshID(mapping)
		// Mendix uses "Forms$PageParameterMapping" as the BSON $Type for show-page
		// parameter mappings inside ShowFormAction.FormSettings.ParameterMappings.
		// The default initPageParameterMapping() uses "Forms$FormCallArgument" which
		// Studio Pro interprets as LayoutCallArgument (for layout slot arguments), causing
		// a deserialization error. Override to the correct storage type here.
		mapping.SetTypeName("Forms$PageParameterMapping")
		mapping.SetParameterQualifiedName(pageQN + "." + arg.ParamName)
		mapping.SetArgument(fb.exprToString(arg.Value))
		settings.AddParameterMappings(mapping)
	}

	if s.Title != "" {
		settings.SetTitleOverride(buildTextElementGen(s.Title))
	}

	action := genMf.NewShowPageAction()
	assignFreshID(action)
	action.SetErrorHandlingType(fb.ehTypeGen(nil))
	action.SetPageSettings(settings)
	if s.ForObject != "" {
		action.SetPassedObjectVariableName(s.ForObject)
	}

	return fb.genActivityWrap(action, nil, "")
}

// showPageLocationGen normalises the AST location string to one of
// the gen Location values. Anything other than "Popup" / "Modal" /
// "Content" defaults to "Content" (matches legacy fall-through).
func showPageLocationGen(in string) string {
	switch in {
	case "Popup":
		return "Popup"
	case "Modal":
		return "Modal"
	case "Content":
		return "Content"
	}
	return "Content"
}

// buildTextElementGen builds a *genTexts.Text with a single en_US
// translation. Used for ShowPage title overrides — distinct from
// buildTextTemplateGen (which wraps Text inside a TextTemplate).
func buildTextElementGen(text string) *genTexts.Text {
	textElem := genTexts.NewText()
	assignFreshID(textElem)
	tr := genTexts.NewTranslation()
	assignFreshID(tr)
	tr.SetLanguageCode("en_US")
	tr.SetText(text)
	textElem.AddTranslations(tr)
	return textElem
}

// addShowHomePageActionGen emits `show home page;`. Mendix has no
// surface-level parameters for this action — the constructor only
// sets ErrorHandlingType.
func (fb *flowBuilderGen) addShowHomePageActionGen(_ *ast.ShowHomePageStmt) element.ID {
	action := genMf.NewShowHomePageAction()
	assignFreshID(action)
	action.SetErrorHandlingType(fb.ehTypeGen(nil))
	return fb.genActivityWrap(action, nil, "")
}

// addShowMessageActionGen emits `show message '<text>' type <Type>
// [objects [<expr>, …]];`. Same template-text logic as
// ValidationFeedback: a string literal is the template text;
// anything else uses the `{1}` placeholder + the rendered expression
// as a positional parameter.
func (fb *flowBuilderGen) addShowMessageActionGen(s *ast.ShowMessageStmt) element.ID {
	templateText, templateParams := fb.buildShowMessageTemplateText(s)
	for _, arg := range s.TemplateArgs {
		templateParams = append(templateParams, fb.exprToString(arg))
	}

	tmpl := buildTextTemplateGen(templateText, templateParams)

	msgType := strings.TrimSpace(s.Type)
	if msgType == "" {
		msgType = "Information"
	}

	action := genMf.NewShowMessageAction()
	assignFreshID(action)
	action.SetErrorHandlingType(fb.ehTypeGen(nil))
	action.SetTemplate(tmpl)
	action.SetType(msgType)
	return fb.genActivityWrap(action, nil, "")
}

// buildShowMessageTemplateText extracts (text, params) from a
// ShowMessageStmt. Mirrors the legacy template-text logic: a string
// literal is the template; anything else uses the "{1}" placeholder.
func (fb *flowBuilderGen) buildShowMessageTemplateText(s *ast.ShowMessageStmt) (string, []string) {
	if lit, ok := s.Message.(*ast.LiteralExpr); ok && lit.Kind == ast.LiteralString {
		return fmt.Sprintf("%v", lit.Value), nil
	}
	return "{1}", []string{fb.exprToString(s.Message)}
}

// addClosePageActionGen emits `close page [N];`. NumberOfPages
// defaults to 1 when the AST stmt's value is non-positive (matches
// legacy clamping).
func (fb *flowBuilderGen) addClosePageActionGen(s *ast.ClosePageStmt) element.ID {
	n := s.NumberOfPages
	if n <= 0 {
		n = 1
	}
	action := genMf.NewCloseFormAction()
	assignFreshID(action)
	action.SetErrorHandlingType(fb.ehTypeGen(nil))
	action.SetNumberOfPages(int32(n))
	return fb.genActivityWrap(action, nil, "")
}
