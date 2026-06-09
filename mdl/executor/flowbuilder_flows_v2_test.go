// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.c — sequence-flow constructor tests.
//
// Verifies anchor selection, case-value typing, and error-handler
// flagging across the gen flow factories.

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

func TestNewHorizontalFlowGenAnchors(t *testing.T) {
	f := newHorizontalFlowGen("o", "d")
	if f.OriginRefID() != element.ID("o") {
		t.Fatalf("origin = %s, want o", f.OriginRefID())
	}
	if f.DestinationRefID() != element.ID("d") {
		t.Fatalf("destination = %s, want d", f.DestinationRefID())
	}
	if f.OriginConnectionIndex() != int32(AnchorRight) {
		t.Fatalf("origin anchor = %d, want %d", f.OriginConnectionIndex(), AnchorRight)
	}
	if f.DestinationConnectionIndex() != int32(AnchorLeft) {
		t.Fatalf("dest anchor = %d, want %d", f.DestinationConnectionIndex(), AnchorLeft)
	}
	if f.IsErrorHandler() {
		t.Fatal("normal horizontal flow must not be flagged as error handler")
	}
	if f.ID() == "" {
		t.Fatal("flow must carry a generated ID")
	}
}

func TestNewHorizontalFlowWithCaseGenBoolean(t *testing.T) {
	f := newHorizontalFlowWithCaseGen("o", "d", "true")
	items := f.CaseValuesItems()
	if len(items) != 1 {
		t.Fatalf("CaseValues len = %d, want 1", len(items))
	}
	cv, ok := items[0].(*genMf.EnumerationCase)
	if !ok || cv == nil {
		t.Fatalf("CaseValues[0] should be *EnumerationCase, got %T", items[0])
	}
	if cv.Value() != "true" {
		t.Fatalf("case value = %q, want true", cv.Value())
	}
}

func TestNewHorizontalFlowWithCaseGenEmptyProducesNoCase(t *testing.T) {
	// Empty case string → flow is unconditional → must carry NoCase.
	f := newHorizontalFlowWithCaseGen("o", "d", "")
	items := f.CaseValuesItems()
	if len(items) != 0 {
		t.Fatalf("empty case value should produce 0 CaseValues items, got %d", len(items))
	}
}

func TestNewHorizontalFlowWithEnumCaseGenAttachesEnum(t *testing.T) {
	f := newHorizontalFlowWithEnumCaseGen("o", "d", "RED")
	items := f.CaseValuesItems()
	if len(items) != 1 {
		t.Fatalf("CaseValues len = %d, want 1", len(items))
	}
	cv, ok := items[0].(*genMf.EnumerationCase)
	if !ok {
		t.Fatalf("want *EnumerationCase, got %T", items[0])
	}
	if cv.Value() != "RED" {
		t.Fatalf("value = %q, want RED", cv.Value())
	}
}

func TestNewHorizontalFlowWithInheritanceCaseGenAttachesInheritance(t *testing.T) {
	f := newHorizontalFlowWithInheritanceCaseGen("o", "d", "Sales.PremiumCustomer")
	items := f.CaseValuesItems()
	if len(items) != 1 {
		t.Fatalf("CaseValues len = %d, want 1", len(items))
	}
	cv, ok := items[0].(*genMf.InheritanceCase)
	if !ok {
		t.Fatalf("want *InheritanceCase, got %T", items[0])
	}
	if cv.ValueQualifiedName() != "Sales.PremiumCustomer" {
		t.Fatalf("entity = %q, want Sales.PremiumCustomer", cv.ValueQualifiedName())
	}
}

func TestNewDownwardFlowWithCaseGenAnchorsAndCase(t *testing.T) {
	f := newDownwardFlowWithCaseGen("o", "d", "true")
	if f.OriginConnectionIndex() != int32(AnchorBottom) {
		t.Fatalf("origin anchor = %d, want bottom", f.OriginConnectionIndex())
	}
	if f.DestinationConnectionIndex() != int32(AnchorLeft) {
		t.Fatalf("dest anchor = %d, want left", f.DestinationConnectionIndex())
	}
	items := f.CaseValuesItems()
	if len(items) != 1 {
		t.Fatalf("CaseValues len = %d, want 1", len(items))
	}
	cv, ok := items[0].(*genMf.EnumerationCase)
	if !ok || cv.Value() != "true" {
		t.Fatalf("CaseValues[0] = %T / %q, want EnumerationCase / true", items[0], cv.Value())
	}
}

func TestNewDownwardFlowWithInheritanceCaseGenAnchorsAndCase(t *testing.T) {
	f := newDownwardFlowWithInheritanceCaseGen("o", "d", "Mod.Sub")
	if f.OriginConnectionIndex() != int32(AnchorBottom) {
		t.Fatalf("origin anchor = %d, want bottom", f.OriginConnectionIndex())
	}
	items := f.CaseValuesItems()
	if len(items) != 1 {
		t.Fatalf("CaseValues len = %d, want 1", len(items))
	}
	cv, ok := items[0].(*genMf.InheritanceCase)
	if !ok || cv.ValueQualifiedName() != "Mod.Sub" {
		t.Fatalf("CaseValues[0] = %T / %q, want InheritanceCase / Mod.Sub", items[0], cv.ValueQualifiedName())
	}
}

func TestNewUpwardFlowGenAnchors(t *testing.T) {
	f := newUpwardFlowGen("o", "d")
	if f.OriginConnectionIndex() != int32(AnchorRight) {
		t.Fatalf("origin anchor = %d, want right", f.OriginConnectionIndex())
	}
	if f.DestinationConnectionIndex() != int32(AnchorBottom) {
		t.Fatalf("dest anchor = %d, want bottom", f.DestinationConnectionIndex())
	}
}

func TestNewErrorHandlerFlowGenAnchorsAndFlag(t *testing.T) {
	f := newErrorHandlerFlowGen("src", "handler")
	if !f.IsErrorHandler() {
		t.Fatal("error handler flag must be set")
	}
	if f.OriginConnectionIndex() != int32(AnchorBottom) {
		t.Fatalf("origin anchor = %d, want bottom", f.OriginConnectionIndex())
	}
	if f.DestinationConnectionIndex() != int32(AnchorTop) {
		t.Fatalf("dest anchor = %d, want top", f.DestinationConnectionIndex())
	}
}

func TestCaseValueForFlowGenEmptyReturnsNil(t *testing.T) {
	if got := caseValueForFlowGen(""); got != nil {
		t.Fatalf("caseValueForFlowGen(empty) should be nil, got %T", got)
	}
}

func TestCaseValueForFlowGenValueProducesEnumerationCase(t *testing.T) {
	cv := caseValueForFlowGen("CUSTOM_VALUE")
	ec, ok := cv.(*genMf.EnumerationCase)
	if !ok {
		t.Fatalf("want *EnumerationCase, got %T", cv)
	}
	if ec.Value() != "CUSTOM_VALUE" {
		t.Fatalf("value = %q, want CUSTOM_VALUE", ec.Value())
	}
}

func TestConvertErrorHandlingTypeGenDefaultsToRollback(t *testing.T) {
	if got := convertErrorHandlingTypeGen(nil); got != "Rollback" {
		t.Fatalf("nil clause = %q, want Rollback", got)
	}
}

func TestConvertErrorHandlingTypeGenAllVariants(t *testing.T) {
	cases := []struct {
		mode ast.ErrorHandlingType
		want string
	}{
		{ast.ErrorHandlingContinue, "Continue"},
		{ast.ErrorHandlingRollback, "Rollback"},
		{ast.ErrorHandlingCustom, "Custom"},
		{ast.ErrorHandlingCustomWithoutRollback, "CustomWithoutRollBack"},
	}
	for _, tc := range cases {
		t.Run(string(tc.mode), func(t *testing.T) {
			eh := &ast.ErrorHandlingClause{Type: tc.mode}
			if got := convertErrorHandlingTypeGen(eh); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestEhTypeGenNanoflowDefaultsToAbort(t *testing.T) {
	fb := &flowBuilderGen{isNanoflow: true}
	if got := fb.ehTypeGen(nil); got != "Abort" {
		t.Fatalf("nanoflow default = %q, want Abort", got)
	}
}

func TestEhTypeGenMicroflowDefaultsToRollback(t *testing.T) {
	fb := &flowBuilderGen{isNanoflow: false}
	if got := fb.ehTypeGen(nil); got != "Rollback" {
		t.Fatalf("microflow default = %q, want Rollback", got)
	}
}

func TestEhTypeGenExplicitOverridesDefault(t *testing.T) {
	fb := &flowBuilderGen{isNanoflow: true}
	eh := &ast.ErrorHandlingClause{Type: ast.ErrorHandlingContinue}
	if got := fb.ehTypeGen(eh); got != "Continue" {
		t.Fatalf("explicit clause = %q, want Continue", got)
	}
}

func TestIsEmptyCustomErrorHandlerGen(t *testing.T) {
	cases := []struct {
		name string
		eh   *ast.ErrorHandlingClause
		want bool
	}{
		{"nil", nil, false},
		{"custom no body", &ast.ErrorHandlingClause{Type: ast.ErrorHandlingCustom}, true},
		{"custom-without-rollback no body", &ast.ErrorHandlingClause{Type: ast.ErrorHandlingCustomWithoutRollback}, true},
		{"continue no body", &ast.ErrorHandlingClause{Type: ast.ErrorHandlingContinue}, false},
		{"custom with body", &ast.ErrorHandlingClause{
			Type: ast.ErrorHandlingCustom,
			Body: []ast.MicroflowStatement{&ast.ReturnStmt{}},
		}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isEmptyCustomErrorHandlerGen(tc.eh); got != tc.want {
				t.Fatalf("got %t, want %t", got, tc.want)
			}
		})
	}
}

func TestApplyUserAnchorsGenOverrides(t *testing.T) {
	f := newHorizontalFlowGen("o", "d")
	originAnch := &ast.FlowAnchors{From: ast.AnchorSide(AnchorBottom)}
	destAnch := &ast.FlowAnchors{To: ast.AnchorSide(AnchorRight)}

	applyUserAnchorsGen(f, originAnch, destAnch)

	if f.OriginConnectionIndex() != int32(AnchorBottom) {
		t.Fatalf("origin = %d, want %d", f.OriginConnectionIndex(), AnchorBottom)
	}
	if f.DestinationConnectionIndex() != int32(AnchorRight) {
		t.Fatalf("dest = %d, want %d", f.DestinationConnectionIndex(), AnchorRight)
	}
}

func TestApplyUserAnchorsGenAnchorSideUnsetIsNoOp(t *testing.T) {
	f := newHorizontalFlowGen("o", "d")
	// Default is right→left; an unset anchor must not change either side.
	originAnch := &ast.FlowAnchors{From: ast.AnchorSideUnset}
	destAnch := &ast.FlowAnchors{To: ast.AnchorSideUnset}

	applyUserAnchorsGen(f, originAnch, destAnch)

	if f.OriginConnectionIndex() != int32(AnchorRight) {
		t.Fatalf("origin shifted from default: %d", f.OriginConnectionIndex())
	}
	if f.DestinationConnectionIndex() != int32(AnchorLeft) {
		t.Fatalf("dest shifted from default: %d", f.DestinationConnectionIndex())
	}
}

func TestApplyUserAnchorsGenNilFlowIsSafe(t *testing.T) {
	// Must not panic on nil flow input.
	applyUserAnchorsGen(nil, &ast.FlowAnchors{From: ast.AnchorSide(AnchorTop)}, nil)
}

// TestNewHorizontalFlowGenHasCaseValuesNoCase guards against the CE0108
// "Variable not in scope" bug: Studio Pro uses CaseValues (plural
// PartList) with a NoCase element for unconditional flows. Without it,
// Mendix cannot traverse the SequenceFlow graph and reports CE0108.
func TestNewHorizontalFlowGenHasCaseValuesNoCase(t *testing.T) {
	f := newHorizontalFlowGen("origin", "dest")
	items := f.CaseValuesItems()
	if len(items) != 1 {
		t.Fatalf("CaseValues len = %d, want 1 (NoCase element)", len(items))
	}
	if _, ok := items[0].(*genMf.NoCase); !ok {
		t.Fatalf("CaseValues[0] = %T, want *genMf.NoCase", items[0])
	}
}

// TestNewHorizontalFlowWithCaseGenUsesCaseValues guards that branching
// flows store the case value in CaseValues (plural PartList) not the
// defunct CaseValue (singular Part). Studio Pro 11 only reads
// CaseValues; using CaseValue causes silent loss of branch wiring.
func TestNewHorizontalFlowWithCaseGenUsesCaseValues(t *testing.T) {
	f := newHorizontalFlowWithCaseGen("o", "d", "true")
	items := f.CaseValuesItems()
	if len(items) != 1 {
		t.Fatalf("CaseValues len = %d, want 1", len(items))
	}
	ec, ok := items[0].(*genMf.EnumerationCase)
	if !ok {
		t.Fatalf("CaseValues[0] = %T, want *EnumerationCase", items[0])
	}
	if ec.Value() != "true" {
		t.Fatalf("EnumerationCase.Value = %q, want true", ec.Value())
	}
}

// TestNewDownwardFlowWithCaseGenUsesCaseValues mirrors the check above
// for the vertically-oriented branching flow variant.
func TestNewDownwardFlowWithCaseGenUsesCaseValues(t *testing.T) {
	f := newDownwardFlowWithCaseGen("o", "d", "false")
	items := f.CaseValuesItems()
	if len(items) != 1 {
		t.Fatalf("CaseValues len = %d, want 1", len(items))
	}
	ec, ok := items[0].(*genMf.EnumerationCase)
	if !ok || ec.Value() != "false" {
		t.Fatalf("CaseValues[0] = %T / %q, want EnumerationCase / false", items[0], ec.Value())
	}
}
