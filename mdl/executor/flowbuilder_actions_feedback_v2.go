// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.f3 — gen-typed Log / Download / ValidationFeedback adders.
//
// Third instalment of the actions family. These three are siblings —
// all carry an output template (StringTemplate for log, TextTemplate
// for validation feedback, none for download) and surface a small
// number of additional fields. They share the genActivityWrap helper
// from f1 and don't introduce any new state machinery.
//
//   - addLogMessageActionGen — `log <level> node <expr> '<msg>' [with (...)]`
//   - addDownloadFileActionGen — `download file $X [show in browser];`
//   - addValidationFeedbackActionGen — `validation feedback $Obj/Attr message '<msg>';`
//
// Template handling differs from legacy in two ways the gen schema
// dictates:
//
//   - LogMessage's MessageTemplate is `Microflows$StringTemplate` —
//     a flat (Text, Arguments) pair. The legacy `model.Text` with
//     translation map is *not* the gen shape. We build a fresh
//     StringTemplate per emission.
//   - ValidationFeedback's FeedbackTemplate is the standard
//     `Microflows$TextTemplate -> Texts$Text` nesting (two-level).
//     The Text element holds a list of Translation children, one per
//     language code.
//
// Template arguments come from positional `s.Template[]` entries
// (LogMessage) or from the `s.TemplateArgs[]` slice
// (ValidationFeedback). Both render as gen `TemplateArgument`
// elements with `Expression` set to the rendered Mendix expression.

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genTexts "github.com/mendixlabs/mxcli/modelsdk/gen/texts"
)

// addLogMessageActionGen emits a LOG activity. Mirrors
// flowBuilder.addLogMessageAction template-text logic:
//
//   - simple string literal → use raw value as template text
//   - complex expression → use "{1}" placeholder + add expression as
//     positional parameter
//   - explicit `with (...)` template params → use authored params
//
// Level defaults to "Info" (matches sdk LogLevelInfo string).
func (fb *flowBuilderGen) addLogMessageActionGen(s *ast.LogStmt) element.ID {
	level := "Info"
	switch s.Level {
	case ast.LogTrace:
		level = "Trace"
	case ast.LogDebug:
		level = "Debug"
	case ast.LogWarning:
		level = "Warning"
	case ast.LogError:
		level = "Error"
	case ast.LogCritical:
		level = "Critical"
	}

	templateText, templateParams := fb.buildLogTemplateText(s)
	logNode := defaultLogNodeExpression
	if s.Node != nil {
		logNode = fb.exprToString(s.Node)
	}

	tmpl := genMf.NewStringTemplate()
	assignFreshID(tmpl)
	tmpl.SetText(templateText)
	for _, p := range templateParams {
		arg := genMf.NewTemplateArgument()
		assignFreshID(arg)
		arg.SetExpression(p)
		tmpl.AddArguments(arg)
	}

	action := genMf.NewLogMessageAction()
	assignFreshID(action)
	action.SetErrorHandlingType(fb.ehTypeGen(nil))
	action.SetLevel(level)
	action.SetNode(logNode)
	action.SetMessageTemplate(tmpl)

	return fb.genActivityWrap(action, nil, "")
}

// buildLogTemplateText extracts the (text, params) pair from a
// LogStmt — split out so the template-text logic is independently
// testable and so the calling adder stays declarative.
func (fb *flowBuilderGen) buildLogTemplateText(s *ast.LogStmt) (string, []string) {
	if len(s.Template) > 0 {
		var text string
		if lit, ok := ast.Unwrap(s.Message).(*ast.LiteralExpr); ok && lit.Kind == ast.LiteralString {
			text = fmt.Sprintf("%v", lit.Value)
		} else {
			text = fb.exprToString(s.Message)
		}
		maxIndex := 0
		for _, p := range s.Template {
			if p.Index > maxIndex {
				maxIndex = p.Index
			}
		}
		params := make([]string, maxIndex)
		for _, p := range s.Template {
			if p.Index > 0 && p.Index <= maxIndex {
				params[p.Index-1] = fb.exprToString(p.Value)
			}
		}
		return text, params
	}
	if lit, ok := ast.Unwrap(s.Message).(*ast.LiteralExpr); ok && lit.Kind == ast.LiteralString {
		return fmt.Sprintf("%v", lit.Value), nil
	}
	return "{1}", []string{fb.exprToString(s.Message)}
}

// addDownloadFileActionGen emits a `download file $X [show in browser];`
// activity.
func (fb *flowBuilderGen) addDownloadFileActionGen(s *ast.DownloadFileStmt) element.ID {
	action := genMf.NewDownloadFileAction()
	assignFreshID(action)
	action.SetErrorHandlingType(fb.ehTypeGen(s.ErrorHandling))
	action.SetFileDocumentVariableName(s.FileDocument)
	action.SetShowFileInBrowser(s.ShowInBrowser)
	return fb.genActivityWrap(action, s.ErrorHandling, "")
}

// addValidationFeedbackActionGen emits a `validation feedback
// $Obj/Attr message '<msg>';` activity. Mirrors legacy attribute /
// association classification:
//
//   - 0 dots in segment: bare attribute on entity
//   - 1 dot:             association qualified by module
//   - 2+ dots:           fully-qualified attribute
//
// Multi-hop paths fall back to the first segment treated as attribute.
func (fb *flowBuilderGen) addValidationFeedbackActionGen(s *ast.ValidationFeedbackStmt) element.ID {
	templateText, templateParams := fb.buildValidationTemplateText(s)
	for _, arg := range s.TemplateArgs {
		templateParams = append(templateParams, fb.exprToString(arg))
	}

	tmpl := buildTextTemplateGen(templateText, templateParams)

	attribute, association := fb.classifyValidationTarget(s)

	varName := strings.TrimPrefix(s.AttributePath.Variable, "$")

	action := genMf.NewValidationFeedbackAction()
	assignFreshID(action)
	action.SetErrorHandlingType(fb.ehTypeGen(nil))
	action.SetObjectVariableName(varName)
	if attribute != "" {
		action.SetAttributeQualifiedName(attribute)
	}
	if association != "" {
		action.SetAssociationQualifiedName(association)
	}
	action.SetFeedbackTemplate(tmpl)

	return fb.genActivityWrap(action, nil, "")
}

// buildValidationTemplateText splits the template-text logic out for
// independent testing. Same shape as buildLogTemplateText but without
// the explicit-template path (validation feedback authored MDL has
// no `with (...)` clause for positional params; TemplateArgs is the
// only secondary-param source and the caller appends those).
//
// ast.Unwrap is required because buildSourceExpression wraps every
// parsed node in *ast.SourceExpr, so a direct *ast.LiteralExpr
// assertion fails without first unwrapping the source wrapper.
func (fb *flowBuilderGen) buildValidationTemplateText(s *ast.ValidationFeedbackStmt) (string, []string) {
	if lit, ok := ast.Unwrap(s.Message).(*ast.LiteralExpr); ok && lit.Kind == ast.LiteralString {
		return fmt.Sprintf("%v", lit.Value), nil
	}
	return "{1}", []string{fb.exprToString(s.Message)}
}

// classifyValidationTarget resolves the attribute / association
// pair for a validation-feedback target. Returns ("", "") for
// missing-path inputs so the caller skips both setters and the
// codec emits neither field.
func (fb *flowBuilderGen) classifyValidationTarget(s *ast.ValidationFeedbackStmt) (string, string) {
	entityQName := ""
	if fb.varTypes != nil {
		entityQName = fb.varTypes[s.AttributePath.Variable]
	}
	if len(s.AttributePath.Segments) == 1 {
		seg := s.AttributePath.Segments[0].Name
		switch strings.Count(seg, ".") {
		case 0:
			if entityQName != "" {
				return entityQName + "." + seg, ""
			}
			return seg, ""
		case 1:
			return "", seg
		default:
			return seg, ""
		}
	}
	if len(s.AttributePath.Segments) > 1 {
		seg := s.AttributePath.Segments[0].Name
		if entityQName != "" {
			return entityQName + "." + seg, ""
		}
		return seg, ""
	}
	if entityQName != "" && len(s.AttributePath.Path) > 0 {
		return entityQName + "." + s.AttributePath.Path[0], ""
	}
	return "", ""
}

// buildTextTemplateGen wraps a template text + parameter list into
// the gen TextTemplate → Text → Translation nesting. The single
// translation uses the en_US locale (matches what the legacy
// `model.Text` carried; gen reads it back via the
// `pickTextTranslationGen` helper that prefers en_US).
func buildTextTemplateGen(text string, params []string) *genMf.TextTemplate {
	tmpl := genMf.NewTextTemplate()
	assignFreshID(tmpl)

	textElem := genTexts.NewText()
	assignFreshID(textElem)
	tr := genTexts.NewTranslation()
	assignFreshID(tr)
	tr.SetLanguageCode("en_US")
	tr.SetText(text)
	textElem.AddTranslations(tr)

	tmpl.SetText(textElem)

	for _, p := range params {
		arg := genMf.NewTemplateArgument()
		assignFreshID(arg)
		arg.SetExpression(p)
		tmpl.AddArguments(arg)
	}
	return tmpl
}
