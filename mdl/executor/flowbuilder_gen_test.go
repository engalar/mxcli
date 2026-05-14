// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.a — flowBuilderGen foundation tests.
//
// Covers the primitives that the rest of the gen builder family will
// build on: error collection, variable-state snapshot/restore, layout
// string formatting, and microflow-existence lookup via the gen
// MicroflowRepository (offline fall-through behaviour).

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

func TestFlowBuilderGenAddErrorAccumulates(t *testing.T) {
	fb := &flowBuilderGen{}
	fb.addError("first %d", 1)
	fb.addError("second")
	got := fb.GetErrors()
	if len(got) != 2 {
		t.Fatalf("want 2 errors, got %d: %v", len(got), got)
	}
	if got[0] != "first 1" || got[1] != "second" {
		t.Fatalf("unexpected errors: %v", got)
	}
}

func TestFlowBuilderGenAddErrorWithExampleEmbedsExample(t *testing.T) {
	fb := &flowBuilderGen{}
	fb.addErrorWithExample("variable '$X' is not declared", "    declare $X Boolean = true;")
	got := fb.GetErrors()
	if len(got) != 1 {
		t.Fatalf("want 1 error, got %d", len(got))
	}
	want := "variable '$X' is not declared\n\n  Example:\n    declare $X Boolean = true;"
	if got[0] != want {
		t.Fatalf("unexpected error text:\nwant: %q\n got: %q", want, got[0])
	}
}

func TestFlowBuilderGenIsVariableDeclared(t *testing.T) {
	fb := &flowBuilderGen{
		varTypes:     map[string]string{"Customer": "Sales.Customer"},
		declaredVars: map[string]string{"Counter": "Integer"},
	}
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"plain entity var", "Customer", true},
		{"dollar-prefixed entity", "$Customer", true},
		{"path-traversed entity", "$Customer/Name", true},
		{"plain primitive", "Counter", true},
		{"unknown", "Missing", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := fb.isVariableDeclared(tc.in); got != tc.want {
				t.Fatalf("%s: want %t got %t", tc.in, tc.want, got)
			}
		})
	}
}

func TestFlowBuilderGenSnapshotRestoreVariableState(t *testing.T) {
	fb := &flowBuilderGen{
		varTypes:     map[string]string{"A": "Mod.X"},
		declaredVars: map[string]string{"i": "Integer"},
	}
	snap := fb.snapshotVariableState()
	// Mutate post-snapshot.
	fb.varTypes["A"] = "Mod.Y"
	fb.varTypes["B"] = "Mod.Z"
	fb.declaredVars["i"] = "String"
	fb.declaredVars["j"] = "Boolean"

	fb.restoreVariableState(snap)

	if fb.varTypes["A"] != "Mod.X" {
		t.Fatalf("varTypes[A] = %q, want Mod.X", fb.varTypes["A"])
	}
	if _, ok := fb.varTypes["B"]; ok {
		t.Fatalf("varTypes[B] should be gone after restore")
	}
	if fb.declaredVars["i"] != "Integer" {
		t.Fatalf("declaredVars[i] = %q, want Integer", fb.declaredVars["i"])
	}
	if _, ok := fb.declaredVars["j"]; ok {
		t.Fatalf("declaredVars[j] should be gone after restore")
	}
}

func TestLayoutPosFormat(t *testing.T) {
	cases := []struct {
		x, y int
		want string
	}{
		{0, 0, "0 0"},
		{200, 200, "200 200"},
		{-10, 25, "-10 25"},
	}
	for _, tc := range cases {
		if got := layoutPos(tc.x, tc.y); got != tc.want {
			t.Fatalf("layoutPos(%d,%d) = %q want %q", tc.x, tc.y, got, tc.want)
		}
	}
}

func TestLayoutSizeFormat(t *testing.T) {
	if got := layoutSize(120, 60); got != "120 60" {
		t.Fatalf("layoutSize(120,60) = %q want %q", got, "120 60")
	}
}

func TestFlowBuilderGenHasDeclaredReturnValue(t *testing.T) {
	cases := []struct {
		name string
		rt   *ast.MicroflowReturnType
		want bool
	}{
		{"nil return type", nil, false},
		{"void", &ast.MicroflowReturnType{Type: ast.DataType{Kind: ast.TypeVoid}}, false},
		{"string", &ast.MicroflowReturnType{Type: ast.DataType{Kind: ast.TypeString}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fb := &flowBuilderGen{returnType: tc.rt}
			if got := fb.hasDeclaredReturnValue(); got != tc.want {
				t.Fatalf("want %t got %t", tc.want, got)
			}
		})
	}
}

func TestFlowBuilderGenMicroflowExistsOfflineFallsThroughToTrue(t *testing.T) {
	// No repo wired — every lookup must report true so syntax-check
	// mode stays unaffected.
	fb := &flowBuilderGen{}
	if !fb.microflowExistsGen("Mod.Foo") {
		t.Fatal("offline microflowExistsGen should return true")
	}
	if !fb.nanoflowExistsGen("Mod.Foo") {
		t.Fatal("offline nanoflowExistsGen should return true")
	}
}

func TestFlowBuilderGenContainerModuleNameOfflineReturnsEmpty(t *testing.T) {
	fb := &flowBuilderGen{}
	if got := fb.containerModuleName(""); got != "" {
		t.Fatalf("offline containerModuleName(empty) = %q, want empty", got)
	}
	if got := fb.containerModuleName("anything"); got != "" {
		t.Fatalf("offline containerModuleName(anything) = %q, want empty", got)
	}
}

func TestFlowBuilderGenExprToStringOfflinePathsResolveAssociation(t *testing.T) {
	// Without a backend, resolvePathSegments returns the path
	// unchanged (its first guard short-circuits on nil backend), so
	// the rendered expression is the raw `$Var/Path` form.
	fb := &flowBuilderGen{}
	expr := &ast.AttributePathExpr{
		Variable: "Order",
		Path:     []string{"Mod.Order_Customer", "Name"},
	}
	got := fb.exprToString(expr)
	want := "$Order/Mod.Order_Customer/Name"
	if got != want {
		t.Fatalf("exprToString = %q want %q", got, want)
	}
}

// setExtraStringField was removed in Stage 3.2.3.f0b; coverage for
// the ad-hoc Property injection path now lives in
// flowbuilder_raw_setter_gen_test.go (TestSetRawBSONField*).
