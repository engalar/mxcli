// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.f3 — Log / Download / ValidationFeedback adder tests.

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genTexts "github.com/mendixlabs/mxcli/modelsdk/gen/texts"
)

func TestAddLogMessageActionGenLevelMapping(t *testing.T) {
	cases := []struct {
		in   ast.LogLevel
		want string
	}{
		{ast.LogTrace, "Trace"},
		{ast.LogDebug, "Debug"},
		{ast.LogInfo, "Info"},
		{ast.LogWarning, "Warning"},
		{ast.LogError, "Error"},
		{ast.LogCritical, "Critical"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			fb := newActionTestFb()
			stmt := &ast.LogStmt{
				Level:   tc.in,
				Message: &ast.LiteralExpr{Kind: ast.LiteralString, Value: "x"},
			}
			fb.addLogMessageActionGen(stmt)
			act := actionFromObjects(t, fb).(*genMf.LogMessageAction)
			if act.Level() != tc.want {
				t.Fatalf("level = %q, want %q", act.Level(), tc.want)
			}
		})
	}
}

func TestAddLogMessageActionGenSimpleStringTemplate(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.LogStmt{
		Level:   ast.LogInfo,
		Message: &ast.LiteralExpr{Kind: ast.LiteralString, Value: "Hello"},
	}
	fb.addLogMessageActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.LogMessageAction)
	tmpl := act.MessageTemplate().(*genMf.StringTemplate)
	if tmpl.Text() != "Hello" {
		t.Fatalf("text = %q, want Hello", tmpl.Text())
	}
	if len(tmpl.ArgumentsItems()) != 0 {
		t.Fatalf("args = %d, want 0", len(tmpl.ArgumentsItems()))
	}
}

func TestAddLogMessageActionGenComplexExpressionUsesPlaceholder(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.LogStmt{
		Level:   ast.LogInfo,
		Message: &ast.VariableExpr{Name: "Var"},
	}
	fb.addLogMessageActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.LogMessageAction)
	tmpl := act.MessageTemplate().(*genMf.StringTemplate)
	if tmpl.Text() != "{1}" {
		t.Fatalf("text = %q, want {1}", tmpl.Text())
	}
	args := tmpl.ArgumentsItems()
	if len(args) != 1 {
		t.Fatalf("args = %d, want 1", len(args))
	}
	a := args[0].(*genMf.TemplateArgument)
	if a.Expression() != "$Var" {
		t.Fatalf("arg expr = %q, want $Var", a.Expression())
	}
}

func TestAddLogMessageActionGenWithExplicitTemplateParams(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.LogStmt{
		Level:   ast.LogInfo,
		Message: &ast.LiteralExpr{Kind: ast.LiteralString, Value: "Hello {1} from {2}"},
		Template: []ast.TemplateParam{
			{Index: 1, Value: &ast.VariableExpr{Name: "User"}},
			{Index: 2, Value: &ast.VariableExpr{Name: "Module"}},
		},
	}
	fb.addLogMessageActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.LogMessageAction)
	tmpl := act.MessageTemplate().(*genMf.StringTemplate)
	if tmpl.Text() != "Hello {1} from {2}" {
		t.Fatalf("text = %q", tmpl.Text())
	}
	args := tmpl.ArgumentsItems()
	if len(args) != 2 {
		t.Fatalf("args = %d, want 2", len(args))
	}
	if args[0].(*genMf.TemplateArgument).Expression() != "$User" {
		t.Fatalf("arg 1 = %q", args[0].(*genMf.TemplateArgument).Expression())
	}
	if args[1].(*genMf.TemplateArgument).Expression() != "$Module" {
		t.Fatalf("arg 2 = %q", args[1].(*genMf.TemplateArgument).Expression())
	}
}

func TestAddLogMessageActionGenDefaultsLogNode(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.LogStmt{
		Level:   ast.LogInfo,
		Message: &ast.LiteralExpr{Kind: ast.LiteralString, Value: "x"},
		// Node is nil — should default
	}
	fb.addLogMessageActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.LogMessageAction)
	if act.Node() != defaultLogNodeExpression {
		t.Fatalf("node = %q, want %q", act.Node(), defaultLogNodeExpression)
	}
}

func TestAddDownloadFileActionGenSetsFields(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.DownloadFileStmt{
		FileDocument:  "Doc",
		ShowInBrowser: true,
	}
	fb.addDownloadFileActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.DownloadFileAction)
	if act.FileDocumentVariableName() != "Doc" {
		t.Fatalf("file doc = %q", act.FileDocumentVariableName())
	}
	if !act.ShowFileInBrowser() {
		t.Fatal("show in browser should be true")
	}
}

func TestAddValidationFeedbackActionGenAttributeFromBareName(t *testing.T) {
	fb := newActionTestFb()
	fb.varTypes["Order"] = "Sales.Order"
	stmt := &ast.ValidationFeedbackStmt{
		AttributePath: &ast.AttributePathExpr{
			Variable: "Order",
			Segments: []ast.PathSegment{{Name: "Status"}},
		},
		Message: &ast.LiteralExpr{Kind: ast.LiteralString, Value: "must be set"},
	}
	fb.addValidationFeedbackActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ValidationFeedbackAction)
	if act.ObjectVariableName() != "Order" {
		t.Fatalf("obj var = %q", act.ObjectVariableName())
	}
	if act.AttributeQualifiedName() != "Sales.Order.Status" {
		t.Fatalf("attribute QN = %q", act.AttributeQualifiedName())
	}
	if act.AssociationQualifiedName() != "" {
		t.Fatalf("association should be empty, got %q", act.AssociationQualifiedName())
	}
}

func TestAddValidationFeedbackActionGenAssociationFromQualifiedName(t *testing.T) {
	fb := newActionTestFb()
	fb.varTypes["Order"] = "Sales.Order"
	stmt := &ast.ValidationFeedbackStmt{
		AttributePath: &ast.AttributePathExpr{
			Variable: "Order",
			Segments: []ast.PathSegment{{Name: "Sales.Order_Customer"}},
		},
		Message: &ast.LiteralExpr{Kind: ast.LiteralString, Value: "ref required"},
	}
	fb.addValidationFeedbackActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ValidationFeedbackAction)
	if act.AssociationQualifiedName() != "Sales.Order_Customer" {
		t.Fatalf("association = %q", act.AssociationQualifiedName())
	}
	if act.AttributeQualifiedName() != "" {
		t.Fatalf("attribute should be empty, got %q", act.AttributeQualifiedName())
	}
}

func TestAddValidationFeedbackActionGenMessageBecomesTextTemplate(t *testing.T) {
	fb := newActionTestFb()
	fb.varTypes["Order"] = "Sales.Order"
	stmt := &ast.ValidationFeedbackStmt{
		AttributePath: &ast.AttributePathExpr{
			Variable: "Order",
			Segments: []ast.PathSegment{{Name: "Status"}},
		},
		Message: &ast.LiteralExpr{Kind: ast.LiteralString, Value: "Bad"},
	}
	fb.addValidationFeedbackActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ValidationFeedbackAction)
	tmpl, ok := act.FeedbackTemplate().(*genMf.TextTemplate)
	if !ok {
		t.Fatalf("template = %T, want *TextTemplate", act.FeedbackTemplate())
	}
	textElem, ok := tmpl.Text().(*genTexts.Text)
	if !ok {
		t.Fatalf("text = %T, want *texts.Text", tmpl.Text())
	}
	translations := textElem.TranslationsItems()
	if len(translations) != 1 {
		t.Fatalf("translations = %d, want 1", len(translations))
	}
	tr := translations[0].(*genTexts.Translation)
	if tr.LanguageCode() != "en_US" {
		t.Fatalf("language = %q, want en_US", tr.LanguageCode())
	}
	if tr.Text() != "Bad" {
		t.Fatalf("translation text = %q, want Bad", tr.Text())
	}
}

func TestBuildTextTemplateGenWithArgs(t *testing.T) {
	tmpl := buildTextTemplateGen("Got {1}", []string{"$Var"})
	args := tmpl.ArgumentsItems()
	if len(args) != 1 {
		t.Fatalf("args = %d, want 1", len(args))
	}
	if args[0].(*genMf.TemplateArgument).Expression() != "$Var" {
		t.Fatalf("arg = %q", args[0].(*genMf.TemplateArgument).Expression())
	}
}
