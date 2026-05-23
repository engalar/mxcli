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

// TestAddValidationFeedbackActionGenSourceExprMessage verifies that when the
// message is wrapped in *ast.SourceExpr (as buildSourceExpression always does),
// the template text is still extracted correctly from the literal (not stored as
// a {1} placeholder expression).
func TestAddValidationFeedbackActionGenSourceExprMessage(t *testing.T) {
	fb := newActionTestFb()
	fb.varTypes["Ticket"] = "HD.Ticket"
	stmt := &ast.ValidationFeedbackStmt{
		AttributePath: &ast.AttributePathExpr{
			Variable: "Ticket",
			Segments: []ast.PathSegment{{Name: "Subject"}},
		},
		Message: &ast.SourceExpr{
			Expression: &ast.LiteralExpr{Kind: ast.LiteralString, Value: "Subject is required"},
			Source:     "'Subject is required'",
		},
	}
	fb.addValidationFeedbackActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ValidationFeedbackAction)
	tmpl := act.FeedbackTemplate().(*genMf.TextTemplate)
	textElem := tmpl.Text().(*genTexts.Text)
	t1 := textElem.TranslationsItems()[0].(*genTexts.Translation)
	if t1.Text() != "Subject is required" {
		t.Fatalf("template text = %q, want literal text (not {1})", t1.Text())
	}
	if len(tmpl.ArgumentsItems()) != 0 {
		t.Fatalf("args = %d, want 0", len(tmpl.ArgumentsItems()))
	}
}

// TestClassifyValidationTargetEntityParamAsEnumRef verifies CE0639 fix:
// when the entity type for a microflow parameter is stored in EnumRef
// (not EntityRef) — as happens for bare qualified names like "HD.Ticket"
// parsed by buildDataType — the varTypes map must still be populated so
// classifyValidationTarget returns a fully-qualified attribute name.
//
// Without the fix, varTypes["Ticket"] is empty and the returned attribute
// QN is the bare segment name "Subject" rather than "HD.Ticket.Subject",
// causing Studio Pro CE0639 "No variable selected".
// TestClassifyValidationTargetEntityParamAsEnumRef verifies CE0639 fix:
// when varTypes["Ticket"] is correctly populated (as happens after the EnumRef
// fix in execCreateMicroflowGen), classifyValidationTarget returns the fully
// qualified attribute name "HD.Ticket.Subject" rather than bare "Subject".
func TestClassifyValidationTargetEntityParamAsEnumRef(t *testing.T) {
	fb := newActionTestFb()
	// Simulate the correct varTypes state after the EnumRef fix:
	// entity params parsed as TypeEnumeration+EnumRef must populate varTypes
	// exactly like EntityRef params do. Variable is stored WITHOUT $ prefix.
	fb.varTypes["Ticket"] = "HD.Ticket"

	stmt := &ast.ValidationFeedbackStmt{
		AttributePath: &ast.AttributePathExpr{
			Variable: "Ticket", // buildAttributePathFromContext strips the $ prefix
			Segments: []ast.PathSegment{{Name: "Subject"}},
		},
		Message: &ast.LiteralExpr{Kind: ast.LiteralString, Value: "Subject is required"},
	}
	fb.addValidationFeedbackActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ValidationFeedbackAction)
	// Must be fully qualified — CE0639 fires when this is bare "Subject".
	if act.AttributeQualifiedName() != "HD.Ticket.Subject" {
		t.Fatalf("attribute QN = %q, want HD.Ticket.Subject (CE0639: unqualified name causes Studio Pro error)", act.AttributeQualifiedName())
	}
	if act.AssociationQualifiedName() != "" {
		t.Fatalf("association should be empty, got %q", act.AssociationQualifiedName())
	}
}

// TestClassifyValidationTargetMissingVarTypes verifies that when varTypes is
// not populated for the variable (old broken behaviour from EnumRef params),
// the returned attribute QN is just the bare segment — demonstrating the
// CE0639 root cause. The fix in execCreateMicroflowGen/execCreateNanoflowGen
// ensures this case never occurs in production.
func TestClassifyValidationTargetMissingVarTypesGivesBareAttr(t *testing.T) {
	fb := newActionTestFb()
	// varTypes is empty — simulates the broken state before the EnumRef fix.
	stmt := &ast.ValidationFeedbackStmt{
		AttributePath: &ast.AttributePathExpr{
			Variable: "Ticket", // no $ prefix — matches production code
			Segments: []ast.PathSegment{{Name: "Subject"}},
		},
		Message: &ast.LiteralExpr{Kind: ast.LiteralString, Value: "Subject is required"},
	}
	fb.addValidationFeedbackActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ValidationFeedbackAction)
	// Without varTypes populated, the QN is just the bare segment.
	// This documents the broken behaviour (CE0639) that the fix prevents.
	if act.AttributeQualifiedName() != "Subject" {
		t.Fatalf("expected bare 'Subject' from broken path, got %q", act.AttributeQualifiedName())
	}
}
