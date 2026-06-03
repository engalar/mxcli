// SPDX-License-Identifier: Apache-2.0

// Unit tests for button and pluggable widget action describe output.
//
// Issue 008: DESCRIBE PAGE outputs call_nanoflow / call_microflow which are
// lexer tokens not used in actionExprV3, causing mxcli check failures.
// The grammar expects: nanoflow / microflow.
package executor

import "testing"

// TestExtractButtonAction_Nanoflow verifies that a nanoflow action button
// is described as "nanoflow Module.NF" (matches actionExprV3 grammar rule),
// not "call_nanoflow Module.NF" (CALL_NANOFLOW token, not in actionExprV3).
func TestExtractButtonAction_Nanoflow(t *testing.T) {
	ctx := &ExecContext{}
	widget := map[string]any{
		"Action": map[string]any{
			"$Type":    "Pages$CallNanoflowClientAction",
			"Nanoflow": "MyModule.MyNanoflow",
		},
	}
	got := extractButtonAction(ctx, widget)
	want := "nanoflow MyModule.MyNanoflow"
	if got != want {
		t.Errorf("extractButtonAction nanoflow: got %q, want %q", got, want)
	}
}

// TestExtractButtonAction_NanoflowForms verifies the Forms$ type prefix variant.
func TestExtractButtonAction_NanoflowForms(t *testing.T) {
	ctx := &ExecContext{}
	widget := map[string]any{
		"Action": map[string]any{
			"$Type":    "Forms$CallNanoflowClientAction",
			"Nanoflow": "Sales.ACT_Confirm",
		},
	}
	got := extractButtonAction(ctx, widget)
	want := "nanoflow Sales.ACT_Confirm"
	if got != want {
		t.Errorf("extractButtonAction nanoflow (Forms prefix): got %q, want %q", got, want)
	}
}

// buildOnClickWidget builds a mock pluggable widget map with an onClick
// action property, for use in extractCustomWidgetPropertyAction tests.
func buildOnClickWidget(actionMap map[string]any) map[string]any {
	const typeID = "onclick-prop-type-001"
	return map[string]any{
		"Type": map[string]any{
			"ObjectType": map[string]any{
				"PropertyTypes": []any{
					map[string]any{
						"PropertyKey": "onClick",
						"$ID":         typeID,
					},
				},
			},
		},
		"Object": map[string]any{
			"Properties": []any{
				map[string]any{
					"TypePointer": typeID,
					"Value": map[string]any{
						"Action": actionMap,
					},
				},
			},
		},
	}
}

// TestExtractCustomWidgetPropertyAction_Microflow verifies that a pluggable
// widget onClick microflow action is described as "microflow Module.MF"
// (matches actionExprV3 grammar), not "call_microflow Module.MF".
func TestExtractCustomWidgetPropertyAction_Microflow(t *testing.T) {
	ctx := &ExecContext{}
	w := buildOnClickWidget(map[string]any{
		"$Type": "Pages$MicroflowClientAction",
		"MicroflowSettings": map[string]any{
			"Microflow": "MyModule.ACT_Process",
		},
	})
	got := extractCustomWidgetPropertyAction(ctx, w, "onClick")
	want := "microflow MyModule.ACT_Process"
	if got != want {
		t.Errorf("extractCustomWidgetPropertyAction microflow: got %q, want %q", got, want)
	}
}

// TestExtractCustomWidgetPropertyAction_MicroflowForms verifies Forms$ prefix.
func TestExtractCustomWidgetPropertyAction_MicroflowForms(t *testing.T) {
	ctx := &ExecContext{}
	w := buildOnClickWidget(map[string]any{
		"$Type": "Forms$MicroflowAction",
		"MicroflowSettings": map[string]any{
			"Microflow": "Sales.ACT_Order",
		},
	})
	got := extractCustomWidgetPropertyAction(ctx, w, "onClick")
	want := "microflow Sales.ACT_Order"
	if got != want {
		t.Errorf("extractCustomWidgetPropertyAction microflow (Forms prefix): got %q, want %q", got, want)
	}
}

// TestExtractCustomWidgetPropertyAction_Nanoflow verifies that a pluggable
// widget onClick nanoflow action is described as "nanoflow Module.NF"
// (matches actionExprV3 grammar), not "call_nanoflow Module.NF".
func TestExtractCustomWidgetPropertyAction_Nanoflow(t *testing.T) {
	ctx := &ExecContext{}
	w := buildOnClickWidget(map[string]any{
		"$Type": "Pages$CallNanoflowClientAction",
		"NanoflowSettings": map[string]any{
			"Nanoflow": "MyModule.NF_ShowFull",
		},
	})
	got := extractCustomWidgetPropertyAction(ctx, w, "onClick")
	want := "nanoflow MyModule.NF_ShowFull"
	if got != want {
		t.Errorf("extractCustomWidgetPropertyAction nanoflow: got %q, want %q", got, want)
	}
}

// TestExtractCustomWidgetPropertyAction_NanoflowForms verifies Forms$ prefix.
func TestExtractCustomWidgetPropertyAction_NanoflowForms(t *testing.T) {
	ctx := &ExecContext{}
	w := buildOnClickWidget(map[string]any{
		"$Type": "Forms$CallNanoflowClientAction",
		"NanoflowSettings": map[string]any{
			"Nanoflow": "Sales.NF_Validate",
		},
	})
	got := extractCustomWidgetPropertyAction(ctx, w, "onClick")
	want := "nanoflow Sales.NF_Validate"
	if got != want {
		t.Errorf("extractCustomWidgetPropertyAction nanoflow (Forms prefix): got %q, want %q", got, want)
	}
}

// TestExtractNavigationListItemAction_ShowPage verifies show_page outputs a
// qualifiedName WITHOUT single quotes (grammar: SHOW_PAGE qualifiedName, not
// STRING_LITERAL). Issue 008: parse path was producing "show_page 'Module.Page'"
// which actionExprV3's SHOW_PAGE qualifiedName rule does not match, breaking roundtrip.
func TestExtractNavigationListItemAction_ShowPage(t *testing.T) {
	ctx := &ExecContext{}
	widget := map[string]any{
		"Action": map[string]any{
			"$Type": "Pages$FormAction",
			"FormSettings": map[string]any{
				"Form": "MyModule.OverviewPage",
			},
		},
	}
	got := extractNavigationListItemAction(ctx, widget)
	want := "show_page MyModule.OverviewPage"
	if got != want {
		t.Errorf("show_page action: got %q, want %q", got, want)
	}
}
